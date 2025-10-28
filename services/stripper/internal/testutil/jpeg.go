package testutil

import (
	"bytes"
	"encoding/binary"
)

func MakeSegment(marker byte, payload []byte) []byte {
	var b bytes.Buffer
	b.Write([]byte{0xFF, marker})
	var length [2]byte

	// Write converted to unit16 len of payload + 2 bytes (of length itself) into length variable
	binary.BigEndian.PutUint16(length[:], uint16(2+len(payload)))

	b.Write(length[:])
	b.Write(payload)
	return b.Bytes()
}

func MakeJPEG(segments ...[]byte) []byte {
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

func MakeSOS(scan []byte) []byte {
	// Minimal SOS header payload
	sosHeader := []byte{0x00, 0x03, 0x01, 0x00, 0x02}
	seg := MakeSegment(0xDA, sosHeader) // SOS (Start of Scan)

	return append(seg, scan...)
}

func ContainsMarker(image []byte, marker byte) bool {
	for i := 0; i < len(image); i++ {
		if image[i] == 0xFF && image[i+1] == marker {
			return true
		}
	}

	return false
}
