package jpegstrip

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

var ErrTruncated = errors.New("truncated or malformed JPEG")
var ErrNotJPEG = errors.New("not a JPEG (missing SOI)")

func MarkerFor(metaType string) (marker byte, prefix []byte) {
	switch strings.ToLower(strings.TrimSpace(metaType)) {
	case "exif":
		return 0xE1, []byte("Exif\x00\x00") // APP1 EXIF
	case "xmp":
		return 0xE1, []byte("http://ns.adobe.com/xap/1.0/") // APP1 XMP
	case "icc":
		return 0xE2, nil // APP2 ICC
	case "comment", "com":
		return 0xFE, nil // COM
	default:
		return 0, nil
	}
}

func Strip(in io.Reader, out io.Writer, metadataRules map[byte][]byte) error {
	var hdr [2]byte
	_, err := io.ReadFull(in, hdr[:])
	if err != nil {
		return ErrTruncated
	}

	if hdr[0] != 0xFF || hdr[1] != 0xD8 {
		return ErrNotJPEG
	}

	_, err = out.Write(hdr[:])
	if err != nil {
		return err
	}

	for {
		marker, err := readMarkerByte(in)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return ErrTruncated
			}
			return err
		}

		switch marker {
		case 0xD9: // EOI (End of Image)
			_, err = out.Write([]byte{0xFF, 0xD9})
			if err != nil {
				return err
			}

			return nil

		case 0xDA: // SOS (Start of Scan)
			// Copy segment + length + header
			err = copySegmentWithLength(in, out, marker)
			if err != nil {
				return err
			}

			// Copy the rest of the file and keep the last 2 bytes values
			var lastBytes [2]byte
			buf := make([]byte, 32*1024)
			for {
				n, err := in.Read(buf)

				if n > 0 {
					if n == 1 {
						lastBytes[0], lastBytes[1] = lastBytes[1], buf[0]
					} else {
						lastBytes[0] = buf[n-2]
						lastBytes[1] = buf[n-1]
					}
					_, writeErr := out.Write(buf[:n])
					if writeErr != nil {
						return writeErr
					}
				}

				if err == io.EOF {
					break
				}

				if err != nil {
					return err
				}
			}

			if !(lastBytes[0] == 0xFF && lastBytes[1] == 0xD9) {
				return ErrTruncated
			}

			return nil

		default:
			prefix, ok := metadataRules[marker]
			if ok {
				if len(prefix) == 0 {
					if err := dropSegmentWithLength(in); err != nil {
						return err
					}
					continue
				}

				if err := dropSegmentByPrefix(in, out, marker, prefix); err != nil {
					return err
				}

				continue
			}

			if isNoLengthMarker(marker) {
				_, err := out.Write([]byte{0xFF, marker})
				if err != nil {
					return err
				}

				continue
			}

			err = copySegmentWithLength(in, out, marker)
			if err != nil {
				return err
			}
		}
	}
}

func readMarkerByte(r io.Reader) (byte, error) {
	var b [1]byte
	for {
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}

		if b[0] == 0xFF {
			break
		}
	}

	for {
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}

		if b[0] != 0xFF {
			return b[0], nil
		}
	}
}

func copySegmentWithLength(in io.Reader, out io.Writer, marker byte) error {
	_, err := out.Write([]byte{0xFF, marker})
	if err != nil {
		return err
	}

	var lengthBuf [2]byte
	_, err = io.ReadFull(in, lengthBuf[:])
	if err != nil {
		return ErrTruncated
	}

	_, err = out.Write(lengthBuf[:])
	if err != nil {
		return err
	}

	length := binary.BigEndian.Uint16(lengthBuf[:])
	if length < 2 {
		return ErrTruncated
	}

	dataLen := length - 2

	_, err = io.CopyN(out, in, int64(dataLen))
	if err != nil {
		return ErrTruncated
	}

	return nil
}

func dropSegmentWithLength(in io.Reader) error {
	var lengthBuf [2]byte
	_, err := io.ReadFull(in, lengthBuf[:])
	if err != nil {
		return ErrTruncated
	}

	length := binary.BigEndian.Uint16(lengthBuf[:])
	if length < 2 {
		return ErrTruncated
	}

	dataLen := length - 2
	_, err = io.CopyN(io.Discard, in, int64(dataLen))
	if err != nil {
		return ErrTruncated
	}

	return nil
}

func isNoLengthMarker(marker byte) bool {
	// SOI D8, EOI D9, RST0-7 D0–D7, TEM 01
	if marker == 0xD8 || marker == 0xD9 || marker == 0x01 {
		return true
	}
	if marker >= 0xD0 && marker <= 0xD7 {
		return true
	}
	return false
}

// Reads one segment and drops it only if the payload begins with the given prefix
func dropSegmentByPrefix(in io.Reader, out io.Writer, marker byte, dropPrefix []byte) error {
	var lengthBuf [2]byte
	if _, err := io.ReadFull(in, lengthBuf[:]); err != nil {
		return ErrTruncated
	}

	length := binary.BigEndian.Uint16(lengthBuf[:])
	if length < 2 {
		return ErrTruncated
	}

	payloadLen := int64(length - 2)

	// Read up to len(dropPrefix) bytes for prefix comparison
	toPeek := int64(len(dropPrefix))
	if toPeek > payloadLen {
		toPeek = payloadLen
	}

	peek := make([]byte, toPeek)
	if _, err := io.ReadFull(in, peek); err != nil {
		return ErrTruncated
	}

	// Drop only if full prefix matches
	if toPeek == int64(len(dropPrefix)) && bytes.Equal(peek, dropPrefix) {
		// Discard the rest of the payload
		if _, err := io.CopyN(io.Discard, in, payloadLen-toPeek); err != nil {
			return ErrTruncated
		}
		return nil
	}

	// Not a match -> copy entire segment: marker, length, peek, remainder
	if _, err := out.Write([]byte{0xFF, marker}); err != nil {
		return err
	}
	if _, err := out.Write(lengthBuf[:]); err != nil {
		return err
	}
	if _, err := out.Write(peek); err != nil {
		return err
	}
	if _, err := io.CopyN(out, in, payloadLen-toPeek); err != nil {
		return ErrTruncated
	}
	return nil
}
