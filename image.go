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

	protobuf "github.com/ubccr/terf/protobuf"

	_ "image/gif"
	_ "image/png"
)

const (
	ColorSpace = "RGB"
	Channels   = 3
	Format     = "JPEG"
)

// Image is an Example image for training/validating in Tensorflow
type Image struct {
	image.Image

	ID           int
	Width        int
	Height       int
	LabelID      int
	LabelText    string
	Organization string
	Filename     string
}

// RGBImage is a JPEG encoded image in the RGB colorspace
type RGBImage struct {
	img image.Image
}

// NewImage returns a new Image. r is the io.Reader for the raw image data, id
// is the unique identifier for the image, labelID is the integer identifier of
// the label, labelText is the label, filename is the name of the file, and org
// is the organization that produced the image
func NewImage(r io.Reader, id, labelID int, labelText, filename, org string) (*Image, error) {
	im, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	rimg := &Image{
		Image:        im,
		ID:           id,
		LabelID:      labelID,
		LabelText:    labelText,
		Organization: org,
		Filename:     filename,
	}

	b := im.Bounds()
	rimg.Width = b.Max.X
	rimg.Height = b.Max.Y

	return rimg, nil
}

// NewImageFromExample returns a new Image from a Tensorflow example
func NewImageFromExample(example *protobuf.Example) (*Image, error) {
	// TODO handle errors if feature key does not exist or is wrong type
	raw := example.Features.Feature["image/encoded"].Kind.(*protobuf.Feature_BytesList).BytesList.Value[0]

	im, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	// TODO handle errors if feature key does not exist or is wrong type
	// TODO make organization optional?
	rimg := &Image{
		Image:        im,
		ID:           int(example.Features.Feature["image/id"].Kind.(*protobuf.Feature_Int64List).Int64List.Value[0]),
		LabelID:      int(example.Features.Feature["image/class/label"].Kind.(*protobuf.Feature_Int64List).Int64List.Value[0]),
		LabelText:    string(example.Features.Feature["image/class/text"].Kind.(*protobuf.Feature_BytesList).BytesList.Value[0]),
		Filename:     string(example.Features.Feature["image/filename"].Kind.(*protobuf.Feature_BytesList).BytesList.Value[0]),
		Organization: string(example.Features.Feature["image/organization"].Kind.(*protobuf.Feature_BytesList).BytesList.Value[0]),
		Height:       int(example.Features.Feature["image/height"].Kind.(*protobuf.Feature_Int64List).Int64List.Value[0]),
		Width:        int(example.Features.Feature["image/width"].Kind.(*protobuf.Feature_Int64List).Int64List.Value[0]),
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

func (i *Image) int64Feature(val int64) *protobuf.Feature {
	return &protobuf.Feature{
		Kind: &protobuf.Feature_Int64List{
			Int64List: &protobuf.Int64List{
				Value: []int64{val},
			},
		},
	}
}

func (i *Image) floatFeature(val float32) *protobuf.Feature {
	return &protobuf.Feature{
		Kind: &protobuf.Feature_FloatList{
			FloatList: &protobuf.FloatList{
				Value: []float32{val},
			},
		},
	}
}

func (i *Image) bytesFeature(val []byte) *protobuf.Feature {
	return &protobuf.Feature{
		Kind: &protobuf.Feature_BytesList{
			BytesList: &protobuf.BytesList{
				Value: [][]byte{val},
			},
		},
	}
}

// ToExample converts the Image to a Tensorflow Example converting the
// raw image to JPEG format in RGB colorspace
func (i *Image) ToExample() (*protobuf.Example, error) {

	// Convert image to RGB JPEG
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, &RGBImage{i}, nil)
	if err != nil {
		return nil, err
	}

	return &protobuf.Example{
		Features: &protobuf.Features{
			Feature: map[string]*protobuf.Feature{
				"image/height":       i.int64Feature(int64(i.Height)),
				"image/width":        i.int64Feature(int64(i.Width)),
				"image/colorspace":   i.bytesFeature([]byte(ColorSpace)),
				"image/channels":     i.int64Feature(Channels),
				"image/class/label":  i.int64Feature(int64(i.LabelID)),
				"image/class/text":   i.bytesFeature([]byte(i.LabelText)),
				"image/format":       i.bytesFeature([]byte(Format)),
				"image/filename":     i.bytesFeature([]byte(i.Filename)),
				"image/id":           i.int64Feature(int64(i.ID)),
				"image/organization": i.bytesFeature([]byte(i.Organization)),
				"image/encoded":      i.bytesFeature(buf.Bytes()),
			},
		},
	}, nil
}

// Write writes the Image in JPEG format to w
func (i *Image) Write(w io.Writer) error {
	return jpeg.Encode(w, i, nil)
}

// Save writes the Image in JPEG format to a file
func (i *Image) Save(file string) error {
	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, i, nil)
}
