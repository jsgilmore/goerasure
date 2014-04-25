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

/*
#include <stdio.h>
#include "jerasure.h"
#include "liberation.h"
#include "reed_sol.h"
#include "cauchy.h"

char *jerasure_bytes_at(char **data, int i) {
  return data[i];
}

int jerasure_int_at(int **schedule, int i, int j) {
	return schedule[i][j];
}

*/
import "C"

import (
	"fmt"
	"math"
	"unsafe"
)

// Coder impliments the interface that is used to endode and decode data.
// It also implements validity checking across codes and getter methods
// that allow for the retrieval of required variables.
type Coder interface {
	// Encode data and produce parity for the given code.
	Encode(data, coding [][]byte)
	// Decode data and parity with the specified erasures taken into
	// account for the given code.
	Decode(data, coding [][]byte, erasures []int)
	// ValidateCode ensures that the coding parameters follow the
	// specific code's requirements.
	ValidateCode()
	// PrintInfo prints the coding parameters.
	PrintInfo()
	// Ensure that the file size is compatible with the coding parameters.
	CheckFileSize(size int64)
	// K retrieves the number of data blocks.
	K() int
	// M retrueves the number of parity blocks.
	M() int
	// Buffersize retrieves the buffer size
	Buffersize() int64
}

// code is a generic type that specifies the basic variables that a
// code should possess.
type code struct {
	// k is the number of data blocks in the stripe.
	k int
	// m is the number of parity blocks in the stripe.
	m int
	// w is the number of data words in a block.
	w int
	// packetSize is the size over which encoding and decoding occurs.
	packetSize int
	// bufferSize specifies over how many bytes of the file size
	// encoding and decoding should occur.
	bufferSize int64
}

// PrintInfo prints the contents of the code type
func (this *code) PrintInfo() {
	fmt.Printf("bufferSize=%d,packetSize=%d,k=%d,m=%d,w=%d\n", this.bufferSize, this.packetSize, this.k, this.m, this.w)
}

// CheckFileSize ensures that a specific code can be matched to a
// specific file size.
// 
// Both file size and buffer size have to be multiples of the code
// parameters and the file size has to be a multiple of the buffer size.
func (this *code) CheckFileSize(size int64) {
	var multiple int64
	var newsize float64

	// Calculate the multiple of which both buffer size and file size
	// must be a multiple
	if this.packetSize != 0 {
		multiple = int64(sizeInt) * int64(this.k) * int64(this.w) * int64(this.packetSize)
	} else {
		multiple = int64(sizeInt) * int64(this.k) * int64(this.w)
	}

	// Check whether the buffer size is a valid multiple of the
	// required coding parameters
	if this.bufferSize != 0 {
		newBuffersize := int64(math.Ceil(float64(this.bufferSize)/float64(multiple)) * float64(multiple))

		if newBuffersize != this.bufferSize {
			if newBuffersize <= size {
				msg := fmt.Sprintf("Buffer size (%d) is not alligned to the coding parameters. Suggested buffer size: %d", this.bufferSize, newBuffersize)
				panic(msg)
			} else {
				panic("Coding parameters are no valid for this small a file. Perhaps decrease the packet size.")
			}
		}
		// If bufferSize was not set, set it equal to the file size.
	} else {
		this.bufferSize = size
	}

	// Calculate the new file size that is a multiple of buffer size.
	newsize = math.Ceil(float64(size)/float64(this.bufferSize)) * float64(this.bufferSize)
	// Because we're using fixed size blocks, our old file size must be
	// the same as our new file size, otherwise we have to select new
	// coding parameters.
	if newsize/float64(size) != 1 {
		msg := fmt.Sprintf("File size is not a multiple of buffer size (%f). Suggested file size: %f", newsize, newsize)
		panic(msg)
	}
}

// K returns the number of data blocks in the stripe.
func (this *code) K() int {
	return this.k
}

// M returns the number of parity blocks in the stripe.
func (this *code) M() int {
	return this.m
}

// Buffersize returns the buffer size.
func (this *code) Buffersize() int64 {
	return this.bufferSize
}

// matrixCode defines a type of code that uses a coding matrix.
type matrixCode struct {
	code
	matrix *C.int
}

// Encode encodes a matrix code, given a data block and writes the
// output into the coding block
func (this *matrixCode) Encode(data, coding [][]byte) {
	dataC := blockToC(data)
	codingC := blockToC(coding)
	C.jerasure_matrix_encode(C.int(this.k), C.int(this.m), C.int(this.w), this.matrix, dataC, codingC, C.int(this.bufferSize))
	cToBlock(dataC, data)
	cToBlock(codingC, coding)
	C.free(unsafe.Pointer(dataC))
	C.free(unsafe.Pointer(codingC))
}

// Decode decodes a matrix code, given (partially filled) data and
// coding blocks and fills in the missing slices in the blocks.
func (this *matrixCode) Decode(data, coding [][]byte, erasures []int) {

	dataC := blockToC(data)
	codingC := blockToC(coding)

	erasuresC := intSliceToC(erasures)

	// TODO: Buffersize is int64, but the encoding and decoding methods
	// use int. This should probably be made int in the go code as well,
	// to enforce correctness.
	ret := C.jerasure_matrix_decode(C.int(this.k), C.int(this.m), C.int(this.w), this.matrix, 1, erasuresC, dataC, codingC, C.int(this.bufferSize))
	if ret == -1 {
		panic("Erasure decoding failed")
	}

	cToBlock(dataC, data)
	cToBlock(codingC, coding)
	C.free(unsafe.Pointer(dataC))
	C.free(unsafe.Pointer(codingC))
}

// bitMatrixCode defines a type of code that uses a coding bit matrix
// and schedule
//
// The schedule is calculated from the bitmatrix and used for efficient
// encoding. The bitmatrix is also used for decoding.
type bitmatrixCode struct {
	code
	bitmatrix *C.int
	schedule  **C.int
}

// Encode encodes a bit matrix code, given a data block and writes the
// output into the coding block
func (this *bitmatrixCode) Encode(data, coding [][]byte) {
	dataC := blockToC(data)
	codingC := blockToC(coding)

	C.jerasure_schedule_encode(C.int(this.k), C.int(this.m), C.int(this.w), this.schedule, dataC, codingC, C.int(this.bufferSize), C.int(this.packetSize))

	cToBlock(dataC, data)
	cToBlock(codingC, coding)
	C.free(unsafe.Pointer(dataC))
	C.free(unsafe.Pointer(codingC))
}

// Decode decodes a bit matrix code, given (partially filled) data and
// coding blocks and fills in the missing slices in the blocks.
func (this *bitmatrixCode) Decode(data, coding [][]byte, erasures []int) {

	dataC := blockToC(data)
	codingC := blockToC(coding)

	erasuresC := intSliceToC(erasures)

	// TODO: Buffersize is int64, but the encoding and decoding methods
	// use int.
	// This should probably be made int in the go code as well, to enforce
	// correctness, although that will also limit file sizes to int
	ret := C.jerasure_schedule_decode_lazy(C.int(this.k), C.int(this.m), C.int(this.w), this.bitmatrix, erasuresC, dataC, codingC, C.int(this.bufferSize), C.int(this.packetSize), 1)
	if ret == -1 {
		panic("Erasure decoding failed")
	}

	cToBlock(dataC, data)
	cToBlock(codingC, coding)
	C.free(unsafe.Pointer(dataC))
	C.free(unsafe.Pointer(codingC))
}

// CToBlock received a C char matrix as input and outputs a Go byte matrix. 
func cToBlock(dataC **C.char, data [][]byte) {
	for i := range data {
		data[i] = C.GoBytes(unsafe.Pointer(C.jerasure_bytes_at(dataC, C.int(i))), C.int(len(data[i])))
	}
}

// blockToC receives a Go byte matrix as input and outputs a C char matrix.
func blockToC(data [][]byte) **C.char {
	if len(data) < 1 {
		panic("no data given")
	}
	
	var b *C.char
	ptrSize := unsafe.Sizeof(b)
	
	//Allocate the char** list
	ptr := C.malloc(C.size_t(len(data)) * C.size_t(ptrSize))
	
	//Assign each byte slice to its appropriate offset
	for i := 0 ; i < len(data) ; i++ {
		element := (**C.char)(unsafe.Pointer(uintptr(ptr) + uintptr(i)*ptrSize))
		*element = (*C.char)(unsafe.Pointer(&data[i][0]))
	}

	return ((**C.char)(ptr))
}

// NewReedSolVanCode returns a Reed-Solomon code, and initialising the
// coding matrix to a Vandermonde matrix
func NewReedSolVanCode(k, m, w, packetSize int, bufferSize int64) Coder {
	code := &reedSolVanCode{matrixCode{code{k, m, w, packetSize, bufferSize}, nil}}
	code.matrix = C.reed_sol_vandermonde_coding_matrix(C.int(k), C.int(m), C.int(w))
	code.ValidateCode()
	return code
}

// reedSolVanCode is a type of matrixCode
type reedSolVanCode struct {
	matrixCode
}

// ValidateCode validates the code to ensure that the chosen coding
// parameters match what the code allows.
func (this *reedSolVanCode) ValidateCode() {
	checkArgs(this.k, this.m, this.w, this.packetSize, this.bufferSize)

	if this.w != 8 && this.w != 16 && this.w != 32 {
		panic("word size must be 8, 16 or 32")
	}
}

// NewCaucheOrigCode returns a type of bitmatrix code with both the
// bitmatrix and schedule initialised.
func NewCauchyOrigCode(k, m, w, packetSize int, bufferSize int64) Coder {
	code := &cauchyOrigCode{bitmatrixCode{code{k, m, w, packetSize, bufferSize}, nil, nil}}
	matrix := C.cauchy_original_coding_matrix(C.int(k), C.int(m), C.int(w))
	code.bitmatrix = C.jerasure_matrix_to_bitmatrix(C.int(k), C.int(m), C.int(w), matrix)
	code.schedule = C.jerasure_smart_bitmatrix_to_schedule(C.int(k), C.int(m), C.int(w), code.bitmatrix)
	code.ValidateCode()
	return code
}

// cauchyOrigCode is a type of bitmatrixCode
type cauchyOrigCode struct {
	bitmatrixCode
}

// ValidateCode validates the code to ensure that the chosen coding
// parameters match what the code allows.
func (this *cauchyOrigCode) ValidateCode() {
	checkArgs(this.k, this.m, this.w, this.packetSize, this.bufferSize)

	if this.packetSize == 0 {
		panic("packetSize > 0 required")
	}
}

// NewCaucheGoodCode returns a type of bitmatrix code with both the
// bitmatrix and schedule initialised.
func NewCauchyGoodCode(k, m, w, packetSize int, bufferSize int64) Coder {
	code := &cauchyGoodCode{bitmatrixCode{code{k, m, w, packetSize, bufferSize}, nil, nil}}
	matrix := C.cauchy_good_general_coding_matrix(C.int(k), C.int(m), C.int(w))
	code.bitmatrix = C.jerasure_matrix_to_bitmatrix(C.int(k), C.int(m), C.int(w), matrix)
	code.schedule = C.jerasure_smart_bitmatrix_to_schedule(C.int(k), C.int(m), C.int(w), code.bitmatrix)
	code.ValidateCode()
	return code
}

// caucheGoodCode is a type of bitmatrixCode
type cauchyGoodCode struct {
	bitmatrixCode
}

// ValidateCode validates the code to ensure that the chosen coding
// parameters match what the code allows.
func (this *cauchyGoodCode) ValidateCode() {
	checkArgs(this.k, this.m, this.w, this.packetSize, this.bufferSize)

	if this.packetSize == 0 {
		panic("packetSize > 0 required")
	}
}

// NewLiberationCode returns a type of bitmatrix code with both the
// bitmatrix and schedule initialised.
func NewLiberationCode(k, m, w, packetSize int, bufferSize int64) Coder {
	code := &liberationCode{bitmatrixCode{code{k, m, w, packetSize, bufferSize}, nil, nil}}
	code.bitmatrix = C.liberation_coding_bitmatrix(C.int(k), C.int(w))
	code.schedule = C.jerasure_smart_bitmatrix_to_schedule(C.int(k), C.int(m), C.int(w), code.bitmatrix)
	code.ValidateCode()
	return code
}

// liberationCode is a type of bitmatrixCode
type liberationCode struct {
	bitmatrixCode
}

// ValidateCode validates the code to ensure that the chosen coding
// parameters match what the code allows.
func (this *liberationCode) ValidateCode() {
	checkArgs(this.k, this.m, this.w, this.packetSize, this.bufferSize)

	if this.k > this.w {
		panic("k must be less than or equal to w")
	}
	if this.w <= 2 || !isPrime(this.w) {
		panic("w must be greater than two and w must be prime")
	}
	if this.packetSize == 0 {
		panic("packetSize > 0 required")
	}
	if this.packetSize%sizeInt != 0 {
		panic("packetSize must be a multiple of sizeof(int64) == 8")
	}
}

// NewBlaumRothCode returns a type of bitmatrix code with both the
// bitmatrix and schedule initialised.
func NewBlaumRothCode(k, m, w, packetSize int, bufferSize int64) Coder {
	code := &blaumRothCode{bitmatrixCode{code{k, m, w, packetSize, bufferSize}, nil, nil}}
	code.bitmatrix = C.blaum_roth_coding_bitmatrix(C.int(k), C.int(w))
	code.schedule = C.jerasure_smart_bitmatrix_to_schedule(C.int(k), C.int(m), C.int(w), code.bitmatrix)
	code.ValidateCode()
	return code
}

// blaumRothCode is a type of bitmatrixCode
type blaumRothCode struct {
	bitmatrixCode
}

// ValidateCode validates the code to ensure that the chosen coding
// parameters match what the code allows.
func (this *blaumRothCode) ValidateCode() {
	checkArgs(this.k, this.m, this.w, this.packetSize, this.bufferSize)

	if this.k > this.w {
		panic("k must be less than or equal to w")
	}
	if this.w <= 2 || !isPrime(this.w+1) {
		panic("w must be greater than two and w must be prime")
	}
	if this.packetSize == 0 {
		panic("packetSize > 0 required")
	}
	if this.packetSize%sizeInt != 0 {
		panic("packetSize must be a multiple of sizeof(int64) == 8")
	}
}

// NewLiber8tionCode returns a type of bitmatrix code with both the
// bitmatrix and schedule initialised.
func NewLiber8tionCode(k, m, w, packetSize int, bufferSize int64) Coder {
	code := &liber8tionCode{bitmatrixCode{code{k, m, w, packetSize, bufferSize}, nil, nil}}
	code.bitmatrix = C.liber8tion_coding_bitmatrix(C.int(k))
	code.schedule = C.jerasure_smart_bitmatrix_to_schedule(C.int(k), C.int(m), C.int(w), code.bitmatrix)
	code.ValidateCode()
	return code
}

// liber8tionCode is a type of bitmatrixCode
type liber8tionCode struct {
	bitmatrixCode
}

// ValidateCode validates the code to ensure that the chosen coding
// parameters match what the code allows.
func (this *liber8tionCode) ValidateCode() {
	checkArgs(this.k, this.m, this.w, this.packetSize, this.bufferSize)

	if this.k > this.w {
		panic("k must be less than or equal to w")
	}
	if this.w != 8 {
		panic("w must equal 8")
	}
	if this.m != 2 {
		panic("m must equal 2")
	}
	if this.packetSize == 0 {
		panic("packetSize > 0 required")
	}
}

// checkArgs performs sanity checking on the coding parameters provided,
// to ensure that all are positive.
func checkArgs(k, m, w, packetSize int, bufferSize int64) {
	if k <= 0 {
		panic("k <= 0")
	}
	if m < 0 {
		panic("m < 0")
	}
	if w <= 0 {
		panic("w <= 0")
	}
	if packetSize < 0 {
		panic("packetSize < 0")
	}
	if bufferSize < 0 {
		panic("bufferSize < 0")
	}
}

// isPrime is an efficient implementation to check whether any number
// below 512 is prime.
func isPrime(n int) bool {
	fmt.Printf("n is %d\n", n)
	primes := []int{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71,
		73, 79, 83, 89, 97, 101, 103, 107, 109, 113, 127, 131, 137, 139, 149, 151, 157, 163, 167, 173, 179,
		181, 191, 193, 197, 199, 211, 223, 227, 229, 233, 239, 241, 251, 257}
	for _, p := range primes {
		if n%p == 0 {
			if n == p {
				return true
			}
			return false
		}
	}
	panic("n > 512")
}
