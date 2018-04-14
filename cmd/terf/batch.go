// Copyright 2018 terf Authors. All rights reserved.
//
// This file is part of terf.
//
// terf is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// terf is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with terf.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"compress/zlib"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"runtime"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/ubccr/terf"
	"golang.org/x/sync/errgroup"
)

type ImageRecord struct {
	Path         string
	LabelID      int
	LabelText    string
	LabelRaw     string
	Organization string
}

type Shard struct {
	BaseDir  string
	Name     string
	ID       int
	Total    int
	Compress bool
	Images   []*ImageRecord
}

func (s *Shard) Next() *Shard {
	return &Shard{
		BaseDir:  s.BaseDir,
		Name:     s.Name,
		Total:    s.Total,
		ID:       s.ID + 1,
		Compress: s.Compress,
		Images:   make([]*ImageRecord, 0),
	}
}

func (i *ImageRecord) FromRow(row []string) error {
	// Format: image_path,label_id,label_text,label_raw,organization
	id, err := strconv.Atoi(row[1])
	if err != nil {
		return err
	}
	i.Path = row[0]
	i.LabelID = id
	i.LabelText = row[2]
	i.LabelRaw = row[3]
	i.Organization = row[4]

	return nil
}

func Batch(infile, outdir, name string, numPerBatch, total, threads int, compress bool) error {
	if total == 0 {
		return errors.New("Please provide the number of images to process")
	}
	if len(outdir) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		outdir = cwd
	}

	if threads == 0 {
		threads = runtime.NumCPU()
	}

	if len(name) == 0 {
		name = "train"
	}

	if numPerBatch == 0 {
		numPerBatch = 1024
	}

	if numPerBatch > total {
		total = 1
	} else {
		total = int(math.Ceil(float64(total) / float64(numPerBatch)))
	}

	in, err := os.Open(infile)
	if err != nil {
		return err
	}
	defer in.Close()

	r := csv.NewReader(in)

	// Parse header info
	header, err := r.Read()
	if err != nil {
		return err
	}

	if header[0] != "image_path" {
		return errors.New("Invalid header")
	}

	shard := &Shard{
		ID:       1,
		Total:    total,
		Name:     name,
		BaseDir:  outdir,
		Compress: compress,
		Images:   make([]*ImageRecord, 0),
	}

	g, ctx := errgroup.WithContext(context.TODO())
	shards := make(chan *Shard)

	g.Go(func() error {
		defer close(shards)

		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			ir := &ImageRecord{}
			err = ir.FromRow(row)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("Failed to parse image record from csv")
				continue
			}

			shard.Images = append(shard.Images, ir)

			if len(shard.Images)%numPerBatch == 0 {
				select {
				case shards <- shard:
				case <-ctx.Done():
					return ctx.Err()
				}
				shard = shard.Next()
			}
		}

		if len(shard.Images) > 0 {
			select {
			case shards <- shard:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return nil
	})

	for i := 0; i < threads; i++ {
		g.Go(func() error { return process(ctx, shards) })
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func process(ctx context.Context, shards <-chan *Shard) error {
	for shard := range shards {
		outfile := fmt.Sprintf("%s-%.5d-of-%.5d", shard.Name, shard.ID, shard.Total)
		out, err := os.Create(path.Join(shard.BaseDir, outfile))
		if err != nil {
			return err
		}
		defer out.Close()

		var w *terf.Writer

		if shard.Compress {
			zout := zlib.NewWriter(out)
			defer zout.Close()

			w = terf.NewWriter(zout)
		} else {
			w = terf.NewWriter(out)
		}

		for _, ir := range shard.Images {
			fh, err := os.Open(ir.Path)
			if err != nil {
				log.WithFields(log.Fields{
					"imagePath": ir.Path,
					"error":     err,
				}).Error("Failed to open image file")
				continue
			}

			img, err := terf.NewImage(fh, ir.LabelID, path.Base(ir.Path), ir.LabelText, ir.LabelRaw, ir.Organization)
			if err != nil {
				log.WithFields(log.Fields{
					"imagePath": ir.Path,
					"error":     err,
				}).Error("Failed to load image file")
				continue
			}

			ex, err := img.ToExample()
			if err != nil {
				log.WithFields(log.Fields{
					"imagePath": ir.Path,
					"error":     err,
				}).Error("Failed to convert image to example proto")
				continue
			}

			err = w.Write(ex)
			if err != nil {
				log.WithFields(log.Fields{
					"imagePath": ir.Path,
					"error":     err,
				}).Error("Failed to write image")
			}
		}

		select {
		default:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
