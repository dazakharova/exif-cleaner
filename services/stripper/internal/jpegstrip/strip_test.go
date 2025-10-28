package jpegstrip

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/daria/exif-cleaner/services/stripper/internal/testutil"
)

func TestStripValidImage(t *testing.T) {
	app1 := testutil.MakeSegment(0xE1, []byte("Exif\x00\x00SOME-EXIF-DATA"))
	com := testutil.MakeSegment(0xFE, []byte("some comment"))
	dqt := testutil.MakeSegment(0xDB, []byte{0x00})
	sos := testutil.MakeSOS([]byte{0x11, 0x22, 0x33, 0x00, 0xFF, 0x00})

	img := makeJPEG(app1, com, dqt, sos)

	r := bytes.NewReader(img)
	var out bytes.Buffer

	err := Strip(r, &out)
	if err != nil {
		t.Fatalf("Strip() unexpected error: %v", err)
	}
	got := out.Bytes()

	t.Run("Removes APP1 and COM", func(t *testing.T) {
		if testutil.ContainsMarker(got, 0xE1) {
			t.Fatal("APP1 (EXIF) not removed")
		}
		if testutil.ContainsMarker(got, 0xFE) {
			t.Fatal("COM marker not removed")
		}
	})

	t.Run("Preserves JPEG structure", func(t *testing.T) {
		if !(len(got) >= 4 &&
			got[0] == 0xFF && got[1] == 0xD8 &&
			got[len(got)-2] == 0xFF && got[len(got)-1] == 0xD9) {
			t.Fatalf("output not bounded by SOI/EOI; head=% X tail=% X", got[:2], got[len(got)-2:])
		}

		if !containsMarker(got, 0xDB) {
			t.Fatal("expected DQT (0xDB) to be preserved, but it’s missing")
		}
	})

	t.Run("Output is smaller", func(t *testing.T) {
		if len(got) >= len(img) {
			t.Errorf("expected stripped image to be smaller; got input=%d, output=%d", len(img), len(got))
		}
	})
}

func TestStripInvalidImage(t *testing.T) {
	// Not a JPEG (missing SOI)
	t.Run("NotJPEG missing SOI", func(t *testing.T) {
		data := []byte("not-a-jpeg")
		var out bytes.Buffer

		err := Strip(bytes.NewReader(data), &out)
		if !errors.Is(err, ErrNotJPEG) {
			t.Fatalf("want ErrNotJPEG, got %v", err)
		}
	})

	// Truncated header (< 2 bytes)
	t.Run("Truncated header", func(t *testing.T) {
		data := []byte{0xFF} // only first byte of SOI
		var out bytes.Buffer

		err := Strip(bytes.NewReader(data), &out)
		if !errors.Is(err, ErrTruncated) {
			t.Fatalf("want ErrTruncated, got %v", err)
		}
	})

	t.Run("Invalid segment length less than two", func(t *testing.T) {
		var b bytes.Buffer
		b.Write([]byte{0xFF, 0xD8}) // SOI
		b.Write([]byte{0xFF, 0xE2}) // APP2
		b.Write([]byte{0x00, 0x01}) // length = 1 (invalid: must be >= 2)
		b.Write([]byte{0xFF, 0xD9}) // EOI (won't be reached)
		var out bytes.Buffer

		err := Strip(bytes.NewReader(b.Bytes()), &out)
		if !errors.Is(err, ErrTruncated) {
			t.Fatalf("want ErrTruncated for invalid length, got %v", err)
		}
	})

	t.Run("Truncated inside segment payload", func(t *testing.T) {
		var seg bytes.Buffer
		seg.Write([]byte{0xFF, 0xE3}) // APP3
		var L [2]byte
		// length = 2 (length bytes) + 5 (payload) = 7
		binary.BigEndian.PutUint16(L[:], 7)
		seg.Write(L[:])
		seg.Write([]byte{1, 2, 3}) // only 3 bytes of the promised 5 -> truncated

		file := append([]byte{0xFF, 0xD8}, seg.Bytes()...)
		// no EOI -> truncated

		var out bytes.Buffer
		err := Strip(bytes.NewReader(file), &out)
		if !errors.Is(err, ErrTruncated) {
			t.Fatalf("want ErrTruncated, got %v", err)
		}
	})

	t.Run("Truncated after SOS missing final EOI", func(t *testing.T) {
		// SOS header
		sosHeader := []byte{0x00, 0x03, 0x01, 0x00, 0x02}
		sos := makeSegment(0xDA, sosHeader) // FF DA <len> <hdr>

		// Build SOI, DQT, SOS, scan bytes, etc, BUT no final FFD9
		var b bytes.Buffer
		b.Write([]byte{0xFF, 0xD8}) // SOI
		b.Write(sos)
		b.Write([]byte{0x11, 0x22, 0x33, 0x00, 0xFF, 0x00}) // fake scan data (no ending FFD9)

		var out bytes.Buffer
		err := Strip(bytes.NewReader(b.Bytes()), &out)
		if !errors.Is(err, ErrTruncated) {
			t.Fatalf("want ErrTruncated when scan doesn't end with EOI, got %v", err)
		}
	})

	t.Run("Truncated immediately after SOI", func(t *testing.T) {
		data := []byte{0xFF, 0xD8} // SOI only
		var out bytes.Buffer

		err := Strip(bytes.NewReader(data), &out)
		if !errors.Is(err, ErrTruncated) {
			t.Fatalf("want ErrTruncated, got %v", err)
		}
	})
}
