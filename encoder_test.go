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
	k := 6
	m := 2
	w := 7
	packetsize := 128
	buffersize := int64(258048)

	// Select a Liberation code from the codes.go library
	code := NewLiberationCode(k, m, w, packetsize, buffersize)

	err := Encode("testfiles/encoderTest", code)
	if err != nil {
		panic(err)
	}
}

func BenchmarkEncoder(b *testing.B) {
	// Set coding parameters to test for
	k := 6
	m := 2
	w := 7
	packetsize := 128
	buffersize := int64(258048)
	
	// Select a Liberation code from the codes.go library
	code := NewLiberationCode(k, m, w, packetsize, buffersize)
	
	stripeName := "testfiles/encoderTest"
	
	for i := 0; i < b.N; i++ {
		err := Encode(stripeName, code)
		if err != nil {
			panic(err)
		}
		b.SetBytes(67108864)
	}
}
