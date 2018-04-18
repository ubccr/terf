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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/ubccr/terf"
	"golang.org/x/sync/errgroup"
)

type Stats struct {
	Total      int
	Source     map[int]int
	LabelID    map[int]int
	LabelRaw   map[int]int
	LabelText  map[string]int
	Format     map[string]int
	Colorspace map[string]int
}

func NewStats() *Stats {
	return &Stats{
		Source:     make(map[int]int),
		LabelID:    make(map[int]int),
		LabelRaw:   make(map[int]int),
		LabelText:  make(map[string]int),
		Format:     make(map[string]int),
		Colorspace: make(map[string]int),
	}
}

func (s *Stats) Add(from *Stats) {
	s.Total += from.Total
	for key, val := range from.Source {
		s.Source[key] += val
	}
	for key, val := range from.LabelID {
		s.LabelID[key] += val
	}
	for key, val := range from.LabelRaw {
		s.LabelRaw[key] += val
	}
	for key, val := range from.LabelText {
		s.LabelText[key] += val
	}
	for key, val := range from.Format {
		s.Format[key] += val
	}
	for key, val := range from.Colorspace {
		s.Colorspace[key] += val
	}
}

func (s *Stats) Print() {
	fmt.Printf("Total: %d\n", s.Total)
	fmt.Printf("Label: \n")
	for key, val := range s.LabelText {
		fmt.Printf("    - %s: %d\n", key, val)
	}

	if len(s.Source) > 0 {
		fmt.Printf("Source: \n")
		for key, val := range s.Source {
			fmt.Printf("    - %d: %d\n", key, val)
		}
	}

	if len(s.LabelID) > 0 {
		fmt.Printf("Label ID: \n")
		for key, val := range s.LabelID {
			fmt.Printf("    - %d: %d\n", key, val)
		}
	}

	if len(s.LabelRaw) > 0 {
		fmt.Printf("Label Raw: \n")
		for key, val := range s.LabelRaw {
			fmt.Printf("    - %d: %d\n", key, val)
		}
	}
	if len(s.Format) > 0 {
		fmt.Printf("Format: \n")
		for key, val := range s.Format {
			fmt.Printf("    - %s: %d\n", key, val)
		}
	}
	if len(s.Colorspace) > 0 {
		fmt.Printf("Colorspace: \n")
		for key, val := range s.Colorspace {
			fmt.Printf("    - %s: %d\n", key, val)
		}
	}
}

func Summary(inputPath string, threads int, compress bool) error {
	if threads == 0 {
		threads = runtime.NumCPU()
	}

	stat, err := os.Stat(inputPath)
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		stats, err := fileSummary(inputPath, compress)
		if err != nil {
			return err
		}

		stats.Print()
		return nil
	}

	g, ctx := errgroup.WithContext(context.TODO())
	paths := make(chan string)

	g.Go(func() error {
		defer close(paths)

		files, err := ioutil.ReadDir(inputPath)
		if err != nil {
			return err
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}

			select {
			case paths <- filepath.Join(inputPath, f.Name()):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	stats := make(chan *Stats)

	for i := 0; i < threads; i++ {
		g.Go(func() error {
			for path := range paths {
				sum, err := fileSummary(path, compress)
				if err != nil {
					return err
				}

				select {
				case stats <- sum:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		})
	}

	go func() {
		g.Wait()
		close(stats)
	}()

	allStats := NewStats()

	for s := range stats {
		allStats.Add(s)
	}

	if err := g.Wait(); err != nil {
		return err
	}

	allStats.Print()

	return nil
}

func fileSummary(inputPath string, compress bool) (*Stats, error) {
	log.WithFields(log.Fields{
		"path": inputPath,
		"zlib": compress,
	}).Info("Processing file")

	in, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	var r *terf.Reader
	if compress {
		zin, err := zlib.NewReader(in)
		if err != nil {
			return nil, err
		}
		defer zin.Close()

		r = terf.NewReader(zin)
	} else {
		r = terf.NewReader(in)
	}

	stats := NewStats()

	for {
		ex, err := r.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		labelID := terf.ExampleFeatureInt64(ex, "image/class/label")
		labelRaw := terf.ExampleFeatureInt64(ex, "image/class/raw")
		labelText := string(terf.ExampleFeatureBytes(ex, "image/class/text"))
		format := string(terf.ExampleFeatureBytes(ex, "image/format"))
		colorspace := string(terf.ExampleFeatureBytes(ex, "image/colorspace"))
		sourceID := terf.ExampleFeatureInt64(ex, "image/class/source")

		stats.Total++
		stats.LabelText[labelText]++
		stats.LabelID[labelID]++
		stats.LabelRaw[labelRaw]++
		stats.Source[sourceID]++
		stats.Format[format]++
		stats.Colorspace[colorspace]++
	}

	return stats, nil
}
