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

func loadBlocks(stripeName string, k, m int) (blocks []LenReader, erasures []int) {
	// Create an array to store handles to all data and parity blocks
	blocks = make([]LenReader, k + m)
	
	// The extra int is required for the -1 stopper
	erasures = make([]int, k + m + 1)
	
	x := 0
	var blockName string
	
	for i:= 0 ; i < k ; i++ {
		blockName = fmt.Sprintf("%s_k%d", stripeName, i)
		
		block, err := os.Open(blockName)
		if err != nil {
			erasures[x] = i
			blocks[i] = nil
			x++
			fmt.Printf("Failed to open file: %s\n", blockName)
		} else {
		
			// I'm assuming that if no file a found, the file pointer
			// returned will just dereference to a zero size file
			blocks[i] = newFileLenReader(block)
			fmt.Printf("Successfully opened file: %s\n", blockName)
		}
	}
	
	for i:= 0 ; i < m ; i++ {
		blockName = fmt.Sprintf("%s_m%d", stripeName, i)
		
		block, err := os.Open(blockName)
		if err != nil {
			// The coding ids are above the data ids
			erasures[x] = k + i
			blocks[k+i] = nil
			x++
			fmt.Printf("Failed to open file: %s\n", blockName)
		} else {
		
			// I'm assuming that if no file a found, the file pointer
			// returned will just dereference to a zero size file
			blocks[k + i] = newFileLenReader(block)
			fmt.Printf("Successfully opened file: %s\n", blockName)
		}
	}
	// A stopper used by the jerasure library
	erasures[x] = -1
	
	return blocks, erasures
}

func Decode(stripeName string, code Coder) (err error) {
	
	k := code.K()
	m := code.M()
	buffersize := code.Buffersize()
	
	blocks, erasures := loadBlocks(stripeName, k, m)
	
	numErasures := numErasures(erasures)
	if numErasures > m {
		msg := fmt.Sprintf("Found more erasures than parities. Found %d erasures with only %d parities.", numErasures, m)
		panic(msg)	//This panic might be downgraded to a return error agter testing.
	} else if numErasures == 0 {
		fmt.Println("No erasures found. Returning.")
		return nil
	}
	
	bw := newFileBlockWriter(stripeName)
	
	// Read the block sizes and ensure that all blocks are the same size
	size, err := compareAndGetSizes(blocks);
	if err != nil {
		return err
	}

	fmt.Println("Erasure: ", erasures)
	
	// Ensure that block size is a multiple of buffer size and that
	// both block size and buffer size are multiples of the parameter product
	code.CheckFileSize(size)
	

	// After our parameter check, size is a multiple of buffer size
	// Compute the number of buffers that we'll have to read, before
	// we've read a complete file.
	readins := int(size / buffersize)

	// Create data and coding buffers, where each buffer stores all
	// the data or coding blocks respectivly
	data := allocateBuffers(k, buffersize)
	coding := allocateBuffers(m, buffersize)
	total := int64(0)
	
 	code.PrintInfo()

	for i:= 0; i < readins; i++ {
		fmt.Printf("readins=%d,i=%d,size=%d,total=%d,buffersize=%d\n", readins, i, size, total, buffersize)
		n := 0
		
		// Read the data blocks into memory
		for j := 0 ; j < k; j++ {
			
			if wasErased(erasures, j) {
				continue
			}
			
			n, err = blocks[j].Read(data[j])
			if err != nil {
				return err
			}
			if n < len(data[j]) {
				panic("Less data than the buffer size was read.")
			}
			
			total += int64(n)
		}
		
		// Read the coding blocks into memory
		for j := 0 ; j < m; j++ {
			
			if wasErased(erasures, k + j) {
				continue
			}
			
			n, err = blocks[k + j].Read(coding[j])
			if err != nil {
				return err
			}
			if n < len(coding[j]) {
				panic("Less data than the buffer size was read.")
			}
			
			total += int64(n)
		}
			
		code.Decode(data, coding, erasures)

		bw.Data(data)
		bw.Coding(coding)
	}
	bw.WriteErased(erasures)
	return
}

func wasErased (erasures []int, id int) bool {
	for _, value := range(erasures) {
		if value == id {
			return true
		} else if value == -1 {
			break
		}
	}
	return false
}

func numErasures (erasures []int) int {
	for num, value := range(erasures) {
		if (value == -1) {
			return num
		}
	}
	panic("No -1 found in erasure slice")
	return -1
}
