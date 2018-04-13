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
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/ubccr/terf"
)

func Build(infile, outfile string, compress bool) error {
	if len(outfile) == 0 {
		return errors.New("Please specifiy an output file")
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

	out, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer out.Close()

	var w *terf.RecordWriter

	if compress {
		zout := zlib.NewWriter(out)
		defer zout.Close()

		w = terf.NewRecordWriter(zout)
	} else {
		w = terf.NewRecordWriter(out)
	}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		id, err := strconv.Atoi(record[1])
		if err != nil {
			log.WithFields(log.Fields{
				"imagePath": record[0],
				"id":        record[1],
				"error":     err,
			}).Error("Invalid label id")
		}

		img, err := terf.NewImageFromFile(record[0], record[2], id)
		if err != nil {
			log.WithFields(log.Fields{
				"imagePath": record[0],
				"error":     err,
			}).Error("Failed to load image file")
		}

		ex, err := img.ToExample()
		if err != nil {
			log.WithFields(log.Fields{
				"imagePath": record[0],
				"error":     err,
			}).Error("Failed to convert image to example proto")
		}

		err = w.Write(ex)
		if err != nil {
			log.WithFields(log.Fields{
				"imagePath": record[0],
				"error":     err,
			}).Error("Failed to write image")
		}
	}

	return nil
}
