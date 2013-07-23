// +build linux

// Package jerasure wraps the Jerasure C library which calculates
// recovery symbols for use in implementing erasure coding.
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