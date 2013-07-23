// +build linux

// Package jerasure wraps the Jerasure C library which calculates
// recovery symbols for use in implementing erasure coding.
package goerasure

// #include "jerasure.h"
import "C"
import (
	"fmt"
	"unsafe"
	"io"
	"path/filepath"
	"os"
	"errors"
)

var (
	sizeInt = int(unsafe.Sizeof(int(0)))
)

var noDataErr = errors.New("Source data is empty")
var blocksUnequalErr = errors.New("Input block sizes do not match.")

//IntSliceToC converts a Go slice into a C int pointer array
func IntSliceToC(slice []int) *C.int {
	return (*C.int)(unsafe.Pointer(&slice[0]))
}

//PrintMatrix prints the contents of a coding matrix.
func PrintMatrix(matrix []int, r, c, w int) {
	C.jerasure_print_matrix(IntSliceToC(matrix), C.int(r), C.int(c), C.int(w))
}

type LenReader interface {
	io.Reader
	Len() int64
}

func NewFileLenReader(f *os.File) LenReader {
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

func NewFileBlockWriter(filename string) BlockWriter {
	base := filepath.Base(filename)
	ext := filepath.Ext(filename)
	return &fileBlockWriter{base, ext, make(map[int] []byte), make(map[int] []byte)}
}

type fileBlockWriter struct {
	base string
	ext string
	data map[int] []byte
	coding map[int] []byte
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

func (this *fileBlockWriter) WriteData() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	dirpath := filepath.Join(wd, "MyCoding")
	err = os.Mkdir(dirpath, 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}
	for i, buf := range this.data {
		path := filepath.Join(dirpath, fmt.Sprintf("%s_k%d%s", this.base, i, this.ext))
		//fmt.Printf("Path is: %s\n", path)
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

func (this *fileBlockWriter) WriteCoding() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	dirpath := filepath.Join(wd, "MyCoding")
	err = os.Mkdir(dirpath, 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}
	for i, buf := range this.coding {
		path := filepath.Join(dirpath, fmt.Sprintf("%s_m%d%s", this.base, i, this.ext))
		//fmt.Printf("Path is: %s\n", path)
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

func (this *fileBlockWriter) WriteErased(erasures []int) error {
	var path string
	
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	
	dirpath := filepath.Join(wd, "MyCoding")
	err = os.Mkdir(dirpath, 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}
	
	
	for _, erasure := range erasures {
	
		if erasure == -1 {
			break
		}
		
		if erasure < len(this.data) {
			path = filepath.Join(dirpath, fmt.Sprintf("%s_k%d%s", this.base, erasure, this.ext))
		} else {
			path = filepath.Join(dirpath, fmt.Sprintf("%s_m%d%s", this.base, erasure, this.ext))
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
			_, err = file.Write(this.coding[erasure - len(this.data)])
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
	
	for i := 0 ; i < len(src) ; i++ {
	
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

func allocateBuffers(n int, buffersize int64) ([][]byte) {
	bufs := make([][]byte, n)
	
	for i := 0 ; i < n ; i++ {
		bufs[i] = make([]byte, buffersize)
	}
	return bufs
}