package jpegstrip

import (
	"encoding/binary"
	"errors"
	"io"
)

var ErrTruncated = errors.New("truncated or malformed JPEG")
var ErrNotJPEG = errors.New("not a JPEG (missing SOI)")

func Strip(in io.Reader, out io.Writer) error {
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

		case 0xE1, 0xED:
			err := dropSegmentWithLength(in)
			if err != nil {
				return err
			}

			continue

		default:
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
