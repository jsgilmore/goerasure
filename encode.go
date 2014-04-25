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
	"fmt"
	"os"
)

func loadDataBlocks(stripeName string, k int) (blocks []LenReader) {
	// Create an array to store handles to all data blocks
	blocks = make([]LenReader, k)
	
	for i:= 0 ; i < k ; i++ {
		blockName := fmt.Sprintf("%s_k%d", stripeName, i)
		
		//fmt.Printf("Encoder opening file: %s\n", blockName)
		block, err := os.Open(blockName)
		if err != nil {
			panic(err)
		}
		
		// This is where I was. The issue is whether this function
		// returns something new, which has to be allocs, or whether
		// it just works.
		blocks[i] = newFileLenReader(block)
	}
	return
}

// The Encode function takes in a stripe name, where the stripe name
// is the base name of a stripe of data blocks. It then encodes the
// data blocks with the specified coder and writes the coding blocks to disc.
func Encode(stripeName string, code Coder) (err error) {

	k := code.K()
	m := code.M()
	bufferSize := code.Buffersize()
	
	blocks := loadDataBlocks(stripeName, k)
	
	bw := newFileBlockWriter(stripeName)
		
	// Read the block sizes and ensure that all blocks are the same size
	size, err := compareAndGetSizes(blocks)
	if err != nil {
		return err
	}

	// Ensure that block size is a multiple of buffer size and that
	// both block size and buffer size are multiples of the parameter product
	code.CheckFileSize(size)

	// After our parameter check, size is a multiple of buffer size
	// Compute the number of buffers that we'll have to read, before
	// we've read a complete file.
	readins := int(size / bufferSize)

	// Create data and coding buffers, where wach buffer stores all
	// the data or coding blocks respectivly
	data := allocateBuffers(k, bufferSize)
	coding := allocateBuffers(m, bufferSize)
	total := int64(0)

	for i := 0; i < readins; i++ {
		n := 0

		for j := 0; j < k; j++ {
			// Read the file contents of file j directly into the data
			n, err = blocks[j].Read(data[j])
			if err != nil {
				return err
			}
			if n < int(bufferSize) {
				panic("Less data than the buffer size was read.")
			}

			total += int64(n)
		}

		code.Encode(data, coding)

		bw.Data(data)
		bw.Coding(coding)
	}
	// Write out coding blocks
	bw.WriteCoding()
	return
}
