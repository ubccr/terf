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

package terf

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path"

	protobuf "github.com/ubccr/terf/protobuf"

	_ "image/gif"
	_ "image/png"
)

const (
	ColorSpace = "RGB"
	Channels   = 3
	Format     = "JPEG"
)

// ImageRecord
type ImageRecord struct {
	image.Image

	Width     int
	Height    int
	LabelID   int
	LabelText string
	Filename  string
}

// RGBImage is a JPEG encoded image in the RGB colorspace
type RGBImage struct {
	img image.Image
}

// NewImage returns a new ImageRecord. r is the io.Reader for the raw image data,
// filename is the name of the file, labelText is the label, and labelID is the
// integer identifier of the label
func NewImage(r io.Reader, filename, labelText string, labelID int) (*ImageRecord, error) {
	im, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	rimg := &ImageRecord{
		Image:     im,
		LabelText: labelText,
		LabelID:   labelID,
		Filename:  filename,
	}

	b := im.Bounds()
	rimg.Width = b.Max.X
	rimg.Height = b.Max.Y

	return rimg, nil
}

// NewImageFromFile returns a new ImageRecord from a file
func NewImageFromFile(filename, labelText string, labelID int) (*ImageRecord, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return NewImage(file, path.Base(filename), labelText, labelID)
}

// NewImageFromExample returns a new ImageRecord from a Tensorflow example
func NewImageFromExample(example *protobuf.Example) (*ImageRecord, error) {
	// TODO handle errors if feature key does not exist or is wrong type
	raw := example.Features.Feature["image/encoded"].Kind.(*protobuf.Feature_BytesList).BytesList.Value[0]

	im, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	// TODO handle errors if feature key does not exist or is wrong type
	rimg := &ImageRecord{
		Image:     im,
		LabelText: string(example.Features.Feature["image/class/text"].Kind.(*protobuf.Feature_BytesList).BytesList.Value[0]),
		LabelID:   int(example.Features.Feature["image/class/label"].Kind.(*protobuf.Feature_Int64List).Int64List.Value[0]),
		Filename:  string(example.Features.Feature["image/filename"].Kind.(*protobuf.Feature_BytesList).BytesList.Value[0]),
		Height:    int(example.Features.Feature["image/height"].Kind.(*protobuf.Feature_Int64List).Int64List.Value[0]),
		Width:     int(example.Features.Feature["image/width"].Kind.(*protobuf.Feature_Int64List).Int64List.Value[0]),
	}

	b := im.Bounds()

	if rimg.Width != b.Max.X {
		return nil, errors.New("Invalid width")
	}

	if rimg.Height != b.Max.Y {
		return nil, errors.New("Invalid height")
	}

	return rimg, nil
}

func (i *RGBImage) ColorModel() color.Model {
	return color.NRGBAModel
}

func (i *RGBImage) Bounds() image.Rectangle {
	return i.img.Bounds()
}

func (i *RGBImage) At(x, y int) color.Color {
	return color.NRGBAModel.Convert(i.img.At(x, y))
}

func (i *ImageRecord) int64Feature(val int64) *protobuf.Feature {
	return &protobuf.Feature{
		Kind: &protobuf.Feature_Int64List{
			Int64List: &protobuf.Int64List{
				Value: []int64{val},
			},
		},
	}
}

func (i *ImageRecord) floatFeature(val float32) *protobuf.Feature {
	return &protobuf.Feature{
		Kind: &protobuf.Feature_FloatList{
			FloatList: &protobuf.FloatList{
				Value: []float32{val},
			},
		},
	}
}

func (i *ImageRecord) bytesFeature(val []byte) *protobuf.Feature {
	return &protobuf.Feature{
		Kind: &protobuf.Feature_BytesList{
			BytesList: &protobuf.BytesList{
				Value: [][]byte{val},
			},
		},
	}
}

// ToExample converts the ImageRecord to a Tensorflow Example converting the
// raw image to JPEG format in RGB colorspace
func (i *ImageRecord) ToExample() (*protobuf.Example, error) {

	// Convert image to RGB JPEG
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, &RGBImage{i}, nil)
	if err != nil {
		return nil, err
	}

	return &protobuf.Example{
		Features: &protobuf.Features{
			Feature: map[string]*protobuf.Feature{
				"image/height":      i.int64Feature(int64(i.Height)),
				"image/width":       i.int64Feature(int64(i.Width)),
				"image/colorspace":  i.bytesFeature([]byte(ColorSpace)),
				"image/channels":    i.int64Feature(Channels),
				"image/class/label": i.int64Feature(int64(i.LabelID)),
				"image/class/text":  i.bytesFeature([]byte(i.LabelText)),
				"image/format":      i.bytesFeature([]byte(Format)),
				"image/filename":    i.bytesFeature([]byte(i.Filename)),
				"image/encoded":     i.bytesFeature(buf.Bytes()),
			},
		},
	}, nil
}

// Write writes the ImageRecord in JPEG format to w
func (i *ImageRecord) Write(w io.Writer) error {
	return jpeg.Encode(w, i, nil)
}

// Save writes the ImageRecord in JPEG format to a file
func (i *ImageRecord) Save(file string) error {
	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, i, nil)
}
