package jpegstrip

import (
	"io"
)

func Strip(in io.Reader, out io.Writer) error {
	return nil
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
