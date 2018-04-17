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
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strconv"

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

	// Unique ID for the image
	ID int

	// Width in pixels of the image
	Width int

	// Height in pixels of the image
	Height int

	// Integer ID for the normalized label (class)
	LabelID int

	// Integer ID for the raw label
	LabelRaw int

	// The human-readable version of the normalized label
	LabelText string

	// Integer ID for the source of the image. This is typically the
	// organization or owner that created the image
	SourceID int

	// Base filename of the original image
	Filename string
}

// RGBImage is a JPEG encoded image in the RGB colorspace. This wraps
// image.Image and ensures the image will be decoded using NRGBAModel
type RGBImage struct {
	img image.Image
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

// Int64Feature is a helper function for encoding Tensorflow Example proto
// Int64 features
func Int64Feature(val int64) *protobuf.Feature {
	return &protobuf.Feature{
		Kind: &protobuf.Feature_Int64List{
			Int64List: &protobuf.Int64List{
				Value: []int64{val},
			},
		},
	}
}

// FloatFeature is a helper function for encoding Tensorflow Example proto
// Float features
func FloatFeature(val float32) *protobuf.Feature {
	return &protobuf.Feature{
		Kind: &protobuf.Feature_FloatList{
			FloatList: &protobuf.FloatList{
				Value: []float32{val},
			},
		},
	}
}

// BytesFeature is a helper function for encoding Tensorflow Example proto
// Bytes features
func BytesFeature(val []byte) *protobuf.Feature {
	return &protobuf.Feature{
		Kind: &protobuf.Feature_BytesList{
			BytesList: &protobuf.BytesList{
				Value: [][]byte{val},
			},
		},
	}
}

// ExampleFeatureInt64 is a helper function for decoding proto Int64 feature
// from a Tensorflow Example. If key is not found it returns default value
func ExampleFeatureInt64(example *protobuf.Example, key string) int {
	// TODO: return error if key is not found?
	f, ok := example.Features.Feature[key]
	if !ok {
		return 0
	}

	val, ok := f.Kind.(*protobuf.Feature_Int64List)
	if !ok {
		return 0
	}

	return int(val.Int64List.Value[0])
}

// ExampleFeatureFloat is a helper function for decoding proto Float feature
// from a Tensorflow Example. If key is not found it returns default value
func ExampleFeatureFloat(example *protobuf.Example, key string) float64 {
	// TODO: return error if key is not found?
	f, ok := example.Features.Feature[key]
	if !ok {
		return 0
	}

	val, ok := f.Kind.(*protobuf.Feature_FloatList)
	if !ok {
		return 0
	}

	return float64(val.FloatList.Value[0])
}

// ExampleFeatureBytes is a helper function for decoding proto Bytes feature
// from a Tensorflow Example. If key is not found it returns default value
func ExampleFeatureBytes(example *protobuf.Example, key string) []byte {
	// TODO: return error if key is not found?
	f, ok := example.Features.Feature[key]
	if !ok {
		return nil
	}

	val, ok := f.Kind.(*protobuf.Feature_BytesList)
	if !ok {
		return nil
	}

	return val.BytesList.Value[0]
}

// NewImage returns a new Image. r is the io.Reader for the raw image data, id
// is the unique identifier for the image, labelID is the integer identifier of
// the normalized label, labelRaw is the integer identifier for the raw label,
// labelText is the normalized label, filename is the base name of the file,
// and sourceID is the source that produced the image
func NewImage(r io.Reader, id, labelID, labelRaw int, labelText, filename string, sourceID int) (*Image, error) {
	im, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	rimg := &Image{
		Image:     im,
		ID:        id,
		LabelID:   labelID,
		LabelRaw:  labelRaw,
		LabelText: labelText,
		SourceID:  sourceID,
		Filename:  filename,
	}

	b := im.Bounds()
	rimg.Width = b.Max.X
	rimg.Height = b.Max.Y

	return rimg, nil
}

// UnmarshalCSV decodes data from a single CSV record row into Image i. The CSV
// record row is expected to be in the following format:
//
//  image_path,image_id,label_id,label_text,label_raw,source
//
// The image located at image_path will be decoded into a JPEG image
func (i *Image) UnmarshalCSV(row []string) error {
	if len(row) != 6 {
		return errors.New("Invalid CSV row format")
	}

	iid, err := strconv.Atoi(row[1])
	if err != nil {
		return err
	}
	lid, err := strconv.Atoi(row[2])
	if err != nil {
		return err

	}
	rid, err := strconv.Atoi(row[4])
	if err != nil {
		return err

	}
	sid, err := strconv.Atoi(row[5])
	if err != nil {
		return err

	}

	i.ID = iid
	i.LabelID = lid
	i.LabelText = row[3]
	i.LabelRaw = rid
	i.SourceID = sid

	fh, err := os.Open(row[0])
	if err != nil {
		return err
	}
	defer fh.Close()

	im, _, err := image.Decode(fh)
	if err != nil {
		return err
	}

	i.Image = im
	i.Filename = filepath.Base(row[0])

	b := im.Bounds()
	i.Width = b.Max.X
	i.Height = b.Max.Y

	return nil
}

// Name returns the generated base filename for the image: [id].jpg
func (i *Image) Name() string {
	if len(i.LabelText) > 0 {
		return filepath.Join(i.LabelText, fmt.Sprintf("%d.jpg", i.ID))
	}

	return fmt.Sprintf("%d.jpg", i.ID)
}

// MarshalCSV encodes Image i into a CSV record. This is the inverse of
// UnmarshalCSV. The image_path will be generated based on the id of the image
// and the provided baseDir.
func (i *Image) MarshalCSV(baseDir string) []string {
	return []string{
		filepath.Join(baseDir, i.Name()),
		strconv.Itoa(i.ID),
		strconv.Itoa(i.LabelID),
		i.LabelText,
		strconv.Itoa(i.LabelRaw),
		strconv.Itoa(i.SourceID),
	}
}

// UnmarshalExample decodes data from a Tensorflow example proto into Image i.
// This is the inverse of MarshalExample.
func (i *Image) UnmarshalExample(example *protobuf.Example) error {
	raw := ExampleFeatureBytes(example, "image/encoded")

	im, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return err
	}

	// TODO make features optional? or configurable?
	i.Image = im
	i.ID = ExampleFeatureInt64(example, "image/id")
	i.Height = ExampleFeatureInt64(example, "image/height")
	i.Width = ExampleFeatureInt64(example, "image/width")
	i.LabelID = ExampleFeatureInt64(example, "image/class/label")
	i.LabelRaw = ExampleFeatureInt64(example, "image/class/raw")
	i.LabelText = string(ExampleFeatureBytes(example, "image/class/text"))
	i.SourceID = ExampleFeatureInt64(example, "image/class/source")
	i.Filename = string(ExampleFeatureBytes(example, "image/filename"))

	b := im.Bounds()

	if i.Width != b.Max.X {
		return errors.New("Invalid width")
	}

	if i.Height != b.Max.Y {
		return errors.New("Invalid height")
	}

	return nil
}

// MarshalExample converts the Image to a Tensorflow Example proto converting
// the raw image to JPEG format in RGB colorspace. The Example proto schema is
// as follows:
//
//  image/height: integer, image height in pixels
//  image/width: integer, image width in pixels
//  image/colorspace: string, specifying the colorspace, always 'RGB'
//  image/channels: integer, specifying the number of channels, always 3
//  image/class/label: integer, specifying the index in a normalized classification layer
//  image/class/raw: integer, specifying the index in the raw (original) classification layer
//  image/class/source: integer, specifying the index of the source (creator of the image)
//  image/class/text: string, specifying the human-readable version of the normalized label
//  image/format: string, specifying the format, always 'JPEG'
//  image/filename: string containing the basename of the image file
//  image/id: integer, specifying the unique id for the image
//  image/encoded: string, containing JPEG encoded image in RGB colorspace
func (i *Image) MarshalExample() (*protobuf.Example, error) {

	// Convert image to RGB JPEG
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, &RGBImage{i}, nil)
	if err != nil {
		return nil, err
	}

	return &protobuf.Example{
		Features: &protobuf.Features{
			Feature: map[string]*protobuf.Feature{
				"image/height":       Int64Feature(int64(i.Height)),
				"image/width":        Int64Feature(int64(i.Width)),
				"image/colorspace":   BytesFeature([]byte(ColorSpace)),
				"image/channels":     Int64Feature(Channels),
				"image/class/label":  Int64Feature(int64(i.LabelID)),
				"image/class/raw":    Int64Feature(int64(i.LabelRaw)),
				"image/class/source": Int64Feature(int64(i.SourceID)),
				"image/class/text":   BytesFeature([]byte(i.LabelText)),
				"image/format":       BytesFeature([]byte(Format)),
				"image/filename":     BytesFeature([]byte(i.Filename)),
				"image/id":           Int64Feature(int64(i.ID)),
				"image/encoded":      BytesFeature(buf.Bytes()),
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
