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
	"image/draw"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	protobuf "github.com/ubccr/terf/protobuf"

	_ "image/gif"
	_ "image/png"
)

// Image is an Example image for training/validating in Tensorflow
type Image struct {
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

	// Image format (JPEG, PNG)
	Format string

	// Image colorpace (RGB, Gray)
	Colorspace string

	// Raw image data
	Raw []byte
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
	img := &Image{
		ID:        id,
		LabelID:   labelID,
		LabelRaw:  labelRaw,
		LabelText: labelText,
		SourceID:  sourceID,
		Filename:  filename,
	}

	err := img.Read(r)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// UnmarshalCSV decodes data from a single CSV record row into Image i. The CSV
// record row is expected to be in the following format:
//
//  image_path,image_id,label_id,label_text,label_raw,source
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

	err = i.Read(fh)
	if err != nil {
		return err
	}

	i.Filename = filepath.Base(row[0])

	return nil
}

// Name returns the generated base filename for the image: [id].[format]
func (i *Image) Name() string {
	var name string

	// Use ID if exists to ensure unique names otherwise use Filename
	if i.ID > 0 {
		name = fmt.Sprintf("%d.%s", i.ID, strings.ToLower(i.Format))
	} else if len(i.Filename) > 0 {
		name = i.Filename
	} else {
		// TODO generate a name?
		name = fmt.Sprintf("image.%s", strings.ToLower(i.Format))
	}

	return name
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

	// TODO make features optional? or configurable?
	i.ID = ExampleFeatureInt64(example, "image/id")
	i.Height = ExampleFeatureInt64(example, "image/height")
	i.Width = ExampleFeatureInt64(example, "image/width")
	i.LabelID = ExampleFeatureInt64(example, "image/class/label")
	i.LabelRaw = ExampleFeatureInt64(example, "image/class/raw")
	i.LabelText = string(ExampleFeatureBytes(example, "image/class/text"))
	i.SourceID = ExampleFeatureInt64(example, "image/class/source")
	i.Filename = string(ExampleFeatureBytes(example, "image/filename"))
	i.Raw = ExampleFeatureBytes(example, "image/encoded")
	i.Format = string(ExampleFeatureBytes(example, "image/format"))

	return nil
}

// MarshalExample converts the Image to a Tensorflow Example proto.
// The Example proto schema is as follows:
//
//  image/height: integer, image height in pixels
//  image/width: integer, image width in pixels
//  image/colorspace: string, specifying the colorspace
//  image/channels: integer, specifying the number of channels, always 3
//  image/class/label: integer, specifying the index in a normalized classification layer
//  image/class/raw: integer, specifying the index in the raw (original) classification layer
//  image/class/source: integer, specifying the index of the source (creator of the image)
//  image/class/text: string, specifying the human-readable version of the normalized label
//  image/format: string, specifying the format
//  image/filename: string containing the basename of the image file
//  image/id: integer, specifying the unique id for the image
//  image/encoded: string, containing the raw encoded image
func (i *Image) MarshalExample() (*protobuf.Example, error) {
	return &protobuf.Example{
		Features: &protobuf.Features{
			Feature: map[string]*protobuf.Feature{
				"image/height":       Int64Feature(int64(i.Height)),
				"image/width":        Int64Feature(int64(i.Width)),
				"image/colorspace":   BytesFeature([]byte(i.Colorspace)),
				"image/channels":     Int64Feature(3),
				"image/class/label":  Int64Feature(int64(i.LabelID)),
				"image/class/raw":    Int64Feature(int64(i.LabelRaw)),
				"image/class/source": Int64Feature(int64(i.SourceID)),
				"image/class/text":   BytesFeature([]byte(i.LabelText)),
				"image/format":       BytesFeature([]byte(strings.ToUpper(i.Format))),
				"image/filename":     BytesFeature([]byte(i.Filename)),
				"image/id":           Int64Feature(int64(i.ID)),
				"image/encoded":      BytesFeature(i.Raw),
			},
		},
	}, nil
}

// Write writes the raw Image data to w
func (i *Image) Write(w io.Writer) error {
	buf := bytes.NewReader(i.Raw)

	_, err := buf.WriteTo(w)
	return err
}

// Reads raw image data from r, parses image config and sets Format,
// Colorspace, Width and Height
func (i *Image) Read(r io.Reader) error {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r)
	if err != nil {
		return err
	}
	i.Raw = buf.Bytes()

	cfg, format, err := image.DecodeConfig(bytes.NewReader(i.Raw))
	if err != nil {
		return err
	}

	i.Width = cfg.Width
	i.Height = cfg.Height
	i.Format = format

	// TODO add better colorspace detection
	switch cfg.ColorModel {
	case color.YCbCrModel, color.RGBAModel, color.NRGBAModel:
		i.Colorspace = "RGB"
	case color.CMYKModel:
		i.Colorspace = "CMYK"
	case color.GrayModel, color.Gray16Model:
		i.Colorspace = "Gray"
	default:
		i.Colorspace = "Unknown"
	}

	return nil
}

// ToJPEG converts Image to JPEG format in RGB colorspace
func (i *Image) ToJPEG() error {
	orig, _, err := image.Decode(bytes.NewReader(i.Raw))
	if err != nil {
		return err
	}

	b := orig.Bounds()
	im := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(im, im.Bounds(), orig, b.Min, draw.Src)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, im, nil)
	if err != nil {
		return err
	}

	i.Raw = buf.Bytes()
	i.Format = "jpeg"
	i.Colorspace = "RGB"

	return nil
}

// Save writes the Image to a file
func (i *Image) Save(file string) error {
	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()

	return i.Write(out)
}
