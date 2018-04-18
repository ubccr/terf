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
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"bufio"

	"github.com/golang/protobuf/proto"
	protobuf "github.com/ubccr/terf/protobuf"
)

// Reader implements a reader for TFRecords with Example protos
type Reader struct {
	reader *bufio.Reader
}

// NewReader returns a new Reader
func NewReader(r io.Reader) *Reader {
	return &Reader{
        reader: bufio.NewReader(r),
    }
}

// Verify checksum
func (w *Reader) verifyChecksum(data []byte, crcMasked uint32) bool {
	rot := crcMasked - kMaskDelta
	unmaskedCrc := ((rot >> 17) | (rot << 15))

	crc := crc32.Checksum(data, crc32c)

	return crc == unmaskedCrc
}

// Next reads the next Example from the TFRecords input
func (r *Reader) Next() (*protobuf.Example, error) {
	header := make([]byte, 12)
	_, err := io.ReadFull(r.reader, header)
	if err != nil {
		return nil, err
	}

	crc := binary.LittleEndian.Uint32(header[8:12])
	if !r.verifyChecksum(header[0:8], crc) {
		return nil, errors.New("Invalid crc for length")
	}

	length := binary.LittleEndian.Uint64(header[0:8])

	payload := make([]byte, length)
	_, err = io.ReadFull(r.reader, payload)
	if err != nil {
		return nil, err
	}

	footer := make([]byte, 4)
	_, err = io.ReadFull(r.reader, footer)
	if err != nil {
		return nil, err
	}

	crc = binary.LittleEndian.Uint32(footer[0:4])
	if !r.verifyChecksum(payload, crc) {
		return nil, errors.New("Invalid crc for payload")
	}

	ex := &protobuf.Example{}
	err = proto.Unmarshal(payload, ex)
	if err != nil {
		return nil, err
	}

	return ex, nil
}
