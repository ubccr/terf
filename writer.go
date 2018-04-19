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

// Package terf implements a reader/writer for TensorFlow TFRecords files
package terf

import (
	"bufio"
	"encoding/binary"
	"hash/crc32"
	"io"

	"github.com/golang/protobuf/proto"
	protobuf "github.com/ubccr/terf/protobuf"
)

const (
	kMaskDelta = 0xa282ead8
)

var (
	crc32c = crc32.MakeTable(crc32.Castagnoli)
)

// Writer implements a writer for TFRecords with Example protos
type Writer struct {
	writer *bufio.Writer
}

// NewWriter returns a new Writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer: bufio.NewWriter(w),
	}
}

// Returns the masked CRC32C of data
func (w *Writer) checksum(data []byte) uint32 {
	crc := crc32.Checksum(data, crc32c)
	return ((crc >> 15) | (crc << 17)) + kMaskDelta
}

// Error reports any error that has occurred during a previous Write or Flush.
func (w *Writer) Error() error {
	_, err := w.writer.Write(nil)
	return err
}

// Flush writes any buffered data to the underlying io.Writer. To check if an
// error occurred during the Flush, call Error.
func (w *Writer) Flush() {
	w.writer.Flush()
}

// Write writes the Example in TFRecords format
func (w *Writer) Write(ex *protobuf.Example) error {
	// Format of a single record:
	//  uint64    length
	//  uint32    masked crc of length
	//  byte      data[length]
	//  uint32    masked crc of data

	payload, err := proto.Marshal(ex)
	if err != nil {
		return err
	}

	length := len(payload)
	header := make([]byte, 12)
	footer := make([]byte, 4)

	binary.LittleEndian.PutUint64(header[0:8], uint64(length))
	binary.LittleEndian.PutUint32(header[8:12], w.checksum(header[0:8]))
	binary.LittleEndian.PutUint32(footer[0:4], w.checksum(payload))

	_, err = w.writer.Write(header)
	if err != nil {
		return err
	}
	_, err = w.writer.Write(payload)
	if err != nil {
		return err
	}
	_, err = w.writer.Write(footer)
	if err != nil {
		return err
	}

	return nil
}
