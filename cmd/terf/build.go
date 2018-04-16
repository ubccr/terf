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
	"bytes"
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
	ID           int
	LabelID      int
	LabelText    string
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
	// Format: image_path,id,label_id,label_text,organization
	if len(row) != 5 {
		return errors.New("Invalid row format")
	}

	id, err := strconv.Atoi(row[1])
	if err != nil {
		return err
	}
	lid, err := strconv.Atoi(row[2])
	if err != nil {
		return err

	}
	i.Path = row[0]
	i.ID = id
	i.LabelID = lid
	i.LabelText = row[3]
	i.Organization = row[4]

	return nil
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			// Skip required header row
			count--
			if count <= 0 {
				return count, errors.New("No lines found")
			}

			return count, nil
		case err != nil:
			return count, err
		}
	}

}

func Build(infile, outdir, name string, numPerBatch, threads int, compress bool) error {
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

	in, err := os.Open(infile)
	if err != nil {
		return err
	}
	defer in.Close()

	total, err := lineCounter(in)
	if err != nil {
		return err
	}

	if numPerBatch > total {
		total = 1
	} else {
		total = int(math.Ceil(float64(total) / float64(numPerBatch)))
	}

	in.Seek(0, 0)
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
		g.Go(func() error {
			for shard := range shards {

				err := process(shard)
				if err != nil {
					return err
				}

				select {
				default:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func process(shard *Shard) error {
	outfile := fmt.Sprintf("%s-%.5d-of-%.5d", shard.Name, shard.ID, shard.Total)

	log.WithFields(log.Fields{
		"file":   outfile,
		"images": len(shard.Images),
		"zlib":   shard.Compress,
	}).Info("Processing shard")

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
			return err
		}

		img, err := terf.NewImage(fh, ir.ID, ir.LabelID, ir.LabelText, path.Base(ir.Path), ir.Organization)
		if err != nil {
			return err
		}

		ex, err := img.ToExample()
		if err != nil {
			return err
		}

		err = w.Write(ex)
		if err != nil {
			return err
		}
	}

	return nil
}
