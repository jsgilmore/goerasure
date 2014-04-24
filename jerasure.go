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

// Package goerasure wraps the Jerasure C library in a Go object
// oriented interface that allows the user to perform erasure coding
// operations in Go using various erasure codes.
package goerasure

// #include "jerasure.h"
import "C"
import "unsafe"

// Create and print a matrix in GF(2^w)
func CreateAndPrint(r, c, w int) {
	matrix := make([]int, r*c)
	n:=1
	for i:=0; i<r*c; i++ {
		matrix[i] = n
		n = int(C.galois_single_multiply(C.int(n), 2, C.int(w)))
	}

	C.jerasure_print_matrix((*C.int)(unsafe.Pointer(&matrix[0])), C.int(r), C.int(c), C.int(w))
}