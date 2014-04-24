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

import (
	"testing"
)

func TestEncoder(t *testing.T) {
	// Set coding parameters to test for
	k := 4
	m := 2
	w := 8
	packetsize := 1024
	// buffer_size := int64(500000)
	buffersize := int64(131072)
	// buffer_size := int64(0)

	// Select a Liber8tion code from the codes.go library
	code := NewLiber8tionCode(k, m, w, packetsize, buffersize)

	err := Encode("testfile", code)
	if err != nil {
		panic(err)
	}
}

func BenchmarkEncoder(b *testing.B) {
	// Set coding parameters to test for
	k := 8
	m := 2
	w := 8
	packetsize := 1024
	// buffer_size := int64(500000)
	buffersize := int64(524288)
	// buffer_size := int64(0)
	
	// Select a Liber8tion code from the codes.go library
	code := NewLiber8tionCode(k, m, w, packetsize, buffersize)
	
	stripeName := "testfile"
	
	for i := 0; i < b.N; i++ {
		err := Encode(stripeName, code)
		if err != nil {
			panic(err)
		}
		b.SetBytes(67108864)
	}
}
