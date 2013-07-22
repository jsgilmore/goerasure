// +build linux

// Package jerasure wraps the Jerasure C library which calculates
// recovery symbols for use in implementing erasure coding.
package jerasure

// #include "jerasure.h"
import "C"

import (
	"fmt"
	"os"
)

func LoadDataBlocks(stripeName string, k int) (blocks []LenReader) {
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
		blocks[i] = NewFileLenReader(block)
	}
	return
}

func Encode(stripeName string, code Coder) (err error) {

	k := code.K()
	m := code.M()
	bufferSize := code.Buffersize()
	
	blocks := LoadDataBlocks(stripeName, k)
	
	bw := NewFileBlockWriter(stripeName)
		
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

	//code.PrintInfo()

	for i := 0; i < readins; i++ {
		//fmt.Printf("readins=%d,i=%d,size=%d,total=%d,buffersize=%d\n", readins, i, size, total, buffersize)
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

		// Debugging code
		/*for u := 0; u < int(bufferSize) ; u++ {
			if data[0][u] != '0' {
				msg := fmt.Sprintf("Data contents before encoding are not '0', but %d", data[0][u])
				panic(msg)
			}
		}*/
		
		code.Encode(data, coding)

		// Debugging code
		/*for u := 0; u < int(bufferSize) ; u++ {
			if data[0][u] != '0' {
				msg := fmt.Sprintf("Data contents after encoding are not '0', but %d", data[0][u])
				panic(msg)
			}
		}*/

		bw.Data(data)
		bw.Coding(coding)
	}
	// Only write out the coding blocks, since the original data still exists.
	//bw.WriteData()
	bw.WriteCoding()
	return
}
