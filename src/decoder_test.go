package jerasure

/*import (
	"testing"
)

func TestDecoder(t *testing.T) {
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

	err := Decode("testfile", code)
	if err != nil {
		panic(err)
	}
}

func BenchmarkDecoder(b *testing.B) {
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
	
	stripeName := "testfile"
	
	for i := 0; i < b.N; i++ {
		Decode(stripeName, code)
	}
}*/
