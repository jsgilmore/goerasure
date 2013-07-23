// +build linux

// Package jerasure wraps the Jerasure C library which calculates
// recovery symbols for use in implementing erasure coding.
package goerasure

// #include "jerasure.h"
import "C"

import (
	"fmt"
	"os"
)

func LoadBlocks(stripeName string, k, m int) (blocks []LenReader, erasures []int) {
	// Create an array to store handles to all data and parity blocks
	blocks = make([]LenReader, k + m)
	
	// The extra int is required for the -1 stopper
	erasures = make([]int, k + m + 1)
	
	x := 0
	var blockName string
	
	for i:= 0 ; i < k ; i++ {
		blockName = fmt.Sprintf("MyCoding/%s_k%d", stripeName, i)
		
		block, err := os.Open(blockName)
		if err != nil {
			erasures[x] = i
			blocks[i] = nil
			x++
			fmt.Printf("Failed to open file: %s\n", blockName)
		} else {
		
			// I'm assuming that if no file a found, the file pointer
			// returned will just dereference to a zero size file
			blocks[i] = NewFileLenReader(block)
			fmt.Printf("Successfully opened file: %s\n", blockName)
		}
	}
	
	for i:= 0 ; i < m ; i++ {
		blockName = fmt.Sprintf("MyCoding/%s_m%d", stripeName, i)
		
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
			blocks[k + i] = NewFileLenReader(block)
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
	
	blocks, erasures := LoadBlocks(stripeName, k, m)
	
	numErasures := NumErasures(erasures)
	if numErasures > m {
		msg := fmt.Sprintf("Found more erasures than parities. Found %d erasures with only %d parities.", numErasures, m)
		panic(msg)	//This panic might be downgraded to a return error agter testing.
	} else if numErasures == 0 {
		fmt.Println("No erasures found. Returning.")
		return nil
	}
	
	bw := NewFileBlockWriter(stripeName)
	
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
			
			if WasErased(erasures, j) {
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
			
			if WasErased(erasures, k + j) {
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
		
		// Debugging code, which I'll remove when I have more faith in
		// the jerasure library
		for u := int64(0) ; u < buffersize ; u++ {
			if data[0][u] != '0' {
				panic("Data contents after decoding do not match 0")
			}
		}

		bw.Data(data)
		bw.Coding(coding)
	}
	bw.WriteErased(erasures)
	return
}

func WasErased (erasures []int, id int) bool {
	for _, value := range(erasures) {
		if value == id {
			return true
		} else if value == -1 {
			break
		}
	}
	return false
}

func NumErasures (erasures []int) int {
	for num, value := range(erasures) {
		if (value == -1) {
			return num
		}
	}
	panic("No -1 found in erasure slice")
	return -1
}
