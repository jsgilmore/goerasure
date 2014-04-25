// +build linux

//   Copyright 2013 Vastech SA (PTY) LTD
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package goerasure

// #include "jerasure.h"
import "C"

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unsafe"
)

var (
	sizeInt = int(unsafe.Sizeof(int(0)))
)

var noDataErr = errors.New("Source data is empty")
var blocksUnequalErr = errors.New("Input block sizes do not match.")

//intSliceToC converts a Go slice into a C int pointer array
func intSliceToC(slice []int) *C.int {
	return (*C.int)(unsafe.Pointer(&slice[0]))
}

//PrintMatrix prints the contents of a coding matrix.
func PrintMatrix(matrix []int, r, c, w int) {
	C.jerasure_print_matrix(intSliceToC(matrix), C.int(r), C.int(c), C.int(w))
}

type LenReader interface {
	io.Reader
	Len() int64
}

func newFileLenReader(f *os.File) LenReader {
	return &fileLenReader{f}
}

type fileLenReader struct {
	f *os.File
}

func (this *fileLenReader) Len() int64 {

	fi, err := this.f.Stat()
	if err != nil {
		return 0
	}
	return fi.Size()
}

func (this *fileLenReader) Read(buf []byte) (n int, err error) {
	return this.f.Read(buf)
}

type BlockWriter interface {
	Data(buf [][]byte)
	Coding(buf [][]byte)
	WriteData() error
	WriteCoding() error
	WriteErased(erasures []int) error
}

func newFileBlockWriter(filename string) BlockWriter {
	return &fileBlockWriter{
		base:   filepath.Base(filename),
		ext:    filepath.Ext(filename),
		dir:    filepath.Dir(filename),
		data:   make(map[int][]byte),
		coding: make(map[int][]byte),
	}
}

type fileBlockWriter struct {
	base   string
	ext    string
	dir    string
	data   map[int][]byte
	coding map[int][]byte
}

func (this *fileBlockWriter) Data(data [][]byte) {
	for i := range data {
		this.data[i] = append(this.data[i], data[i]...)
	}
}

func (this *fileBlockWriter) Coding(coding [][]byte) {
	for i := range coding {
		this.coding[i] = append(this.coding[i], coding[i]...)
	}
}

func (this *fileBlockWriter) write(blocks map[int][]byte, dataOrCode string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	for i, buf := range blocks {
		path := filepath.Join(wd, this.dir, fmt.Sprintf("%s_%s%d%s", this.base, dataOrCode, i, this.ext))
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = file.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *fileBlockWriter) WriteData() error {
	return this.write(this.data, "k")
}

func (this *fileBlockWriter) WriteCoding() error {
	return this.write(this.coding, "m")
}

func (this *fileBlockWriter) WriteErased(erasures []int) error {
	var path string

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	for _, erasure := range erasures {

		if erasure == -1 {
			break
		}

		if erasure < len(this.data) {
			path = filepath.Join(wd, this.dir, fmt.Sprintf("%s_k%d%s", this.base, erasure, this.ext))
		} else {
			path = filepath.Join(wd, this.dir, fmt.Sprintf("%s_m%d%s", this.base, erasure, this.ext))
		}
		fmt.Printf("Path is: %s\n", path)
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()

		if erasure < len(this.data) {
			_, err = file.Write(this.data[erasure])
		} else {
			_, err = file.Write(this.coding[erasure-len(this.data)])
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func compareAndGetSizes(src []LenReader) (size int64, err error) {
	size = 0
	err = nil

	for i := 0; i < len(src); i++ {

		test_size := int64(0)

		if src[i] != nil {
			test_size = src[i].Len()
		}

		//Make sure all blocks are the same length
		if test_size != 0 {
			if size != 0 && size != test_size {
				err = blocksUnequalErr
			}
			size = test_size
		}
		//If file size is zero, assume we're decoding
	}
	return
}

func allocateBuffers(n int, buffersize int64) [][]byte {
	bufs := make([][]byte, n)

	for i := 0; i < n; i++ {
		bufs[i] = make([]byte, buffersize)
	}
	return bufs
}
