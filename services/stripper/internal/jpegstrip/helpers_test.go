package jpegstrip

import (
	"bytes"
	"encoding/binary"
)

func makeSegment(marker byte, payload []byte) []byte {
	var b bytes.Buffer
	b.Write([]byte{0xFF, marker})
	var length [2]byte

	// Write converted to unit16 len of payload + 2 bytes (of length itself) into length variable
	binary.BigEndian.PutUint16(length[:], uint16(2+len(payload)))

	b.Write(length[:])
	b.Write(payload)
	return b.Bytes()
}

func makeJPEG(segments ...[]byte) []byte {
	var b bytes.Buffer
	// Write SOI (Start of Image)
	b.Write([]byte{0xFF, 0xD8})
	for _, segment := range segments {
		b.Write(segment)
	}

	// Write EOI (End of Image)
	b.Write([]byte{0xFF, 0xD9})
	return b.Bytes()
}
