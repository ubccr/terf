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

package terf_test

import (
	"compress/zlib"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ubccr/terf"
)

func ExampleWriter() {
	// Open output file
	out, err := os.Create("train-001")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// Create new terf Writer
	w := terf.NewWriter(out)

	// Read in image data from file
	reader, err := os.Open("image.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	// Create new terf Image with labels and source
	img, err := terf.NewImage(reader, 1, 12, 104, "Crystal", "image.jpg", 10)
	if err != nil {
		log.Fatal(err)
	}

	// Marshal image to Example proto
	example, err := img.MarshalExample()
	if err != nil {
		log.Fatal(err)
	}

	// Write Example proto
	err = w.Write(example)
	if err != nil {
		log.Fatal(err)
	}

	// Write any buffered data to the underlying writer
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}

func ExampleWriter_compressed() {
	// Open output file
	out, err := os.Create("train-001")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// Create zlib writer
	zout := zlib.NewWriter(out)
	defer zout.Close()

	// Create new terf Writer
	w := terf.NewWriter(zout)

	// Read in image data from file
	reader, err := os.Open("image.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	// Create new terf Image with labels and source
	img, err := terf.NewImage(reader, 1, 12, 104, "Crystal", "image.jpg", 10)
	if err != nil {
		log.Fatal(err)
	}

	// Marshal image to Example proto
	example, err := img.MarshalExample()
	if err != nil {
		log.Fatal(err)
	}

	// Write Example proto
	err = w.Write(example)
	if err != nil {
		log.Fatal(err)
	}

	// Write any buffered data to the underlying writer
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}

func ExampleReader() {
	// Open TFRecord file
	in, err := os.Open("train-000")
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	r := terf.NewReader(in)

	count := 0
	for {
		// example will be a TensorFlow Example proto
		example, err := r.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		// Do something with example

		id := terf.ExampleFeatureInt64(example, "image/id")
		labelID := terf.ExampleFeatureInt64(example, "image/class/label")
		labelText := string(terf.ExampleFeatureBytes(example, "image/class/text"))

		fmt.Printf("Image: %d Label: %s (%d)\n", id, labelText, labelID)
		count++
	}

	fmt.Printf("Total records: %d\n", count)
}

func ExampleReader_compressed() {
	// Open TFRecord file
	in, err := os.Open("train-000")
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	// Create new zlib Reader
	zin, err := zlib.NewReader(in)
	if err != nil {
		log.Fatal(err)
	}
	defer zin.Close()

	r := terf.NewReader(zin)

	count := 0
	for {
		// example will be a TensorFlow Example proto
		example, err := r.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		// Do something with example

		id := terf.ExampleFeatureInt64(example, "image/id")
		labelID := terf.ExampleFeatureInt64(example, "image/class/label")
		labelText := string(terf.ExampleFeatureBytes(example, "image/class/text"))

		fmt.Printf("Image: %d Label: %s (%d)\n", id, labelText, labelID)
		count++
	}

	fmt.Printf("Total records: %d\n", count)
}
