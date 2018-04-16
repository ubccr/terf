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
	Total  int
	Orgs   map[string]int
	Labels map[string]int
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

	allStats := &Stats{
		Orgs:   make(map[string]int),
		Labels: make(map[string]int),
	}

	for s := range stats {
		allStats.Add(s)
	}

	if err := g.Wait(); err != nil {
		return err
	}

	allStats.Print()

	return nil
}

func (s *Stats) Add(from *Stats) {
	s.Total += from.Total
	for key, val := range from.Labels {
		s.Labels[key] += val
	}
	for key, val := range from.Orgs {
		s.Orgs[key] += val
	}
}

func (s *Stats) Print() {
	fmt.Printf("Total: %d\n", s.Total)
	fmt.Printf("Labels: \n")
	for key, val := range s.Labels {
		fmt.Printf("    - %s: %d\n", key, val)
	}

	if len(s.Orgs) > 0 {
		fmt.Printf("Organizations: \n")
		for key, val := range s.Orgs {
			fmt.Printf("    - %s: %d\n", key, val)
		}
	}
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

	stats := &Stats{
		Orgs:   make(map[string]int),
		Labels: make(map[string]int),
	}

	for {
		ex, err := r.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		img, err := terf.NewImageFromExample(ex)
		if err != nil {
			return nil, err
		}

		stats.Total++
		if len(img.Organization) > 0 {
			stats.Orgs[img.Organization]++
		}
		stats.Labels[img.LabelText]++
	}

	return stats, nil
}
