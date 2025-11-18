package jpegstrip

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/daria/exif-cleaner/services/stripper/internal/testutil"
)

// helper: build rules map from meta type names ("exif", "xmp", "icc", "com", ...)
func rulesFor(types ...string) map[byte][]byte {
	m := make(map[byte][]byte, len(types))
	for _, t := range types {
		marker, prefix := MarkerFor(t)
		if marker != 0 {
			m[marker] = prefix
		}
	}
	return m
}

func TestStripValidImage(t *testing.T) {
	t.Run("Removes APP1 (EXIF)", func(t *testing.T) {
		r, out, _ := makeFullTestJPEG()
		rules := rulesFor("exif")

		err := Strip(r, out, rules)
		if err != nil {
			t.Fatalf("Strip() unexpected error: %v", err)
		}
		got := out.Bytes()

		if bytes.Contains(got, []byte("Exif\x00\x00")) {
			t.Fatalf("APP1 (EXIF) not removed")
		}
		if !bytes.Contains(got, []byte("http://ns.adobe.com/xap/1.0/")) {
			t.Fatalf("expected XMP APP1 to be preserved")
		}
	})

	t.Run("Removes ICC from valid JPEG", func(t *testing.T) {
		r, out, _ := makeFullTestJPEG()
		rules := rulesFor("icc")

		err := Strip(r, out, rules)
		if err != nil {
			t.Fatalf("Strip() unexpected error: %v", err)
		}
		got := out.Bytes()

		if testutil.ContainsMarker(got, 0xE2) { // APP2
			t.Fatalf("ICC (APP2) not removed")
		}
	})

	t.Run("Removes XMP from valid JPEG", func(t *testing.T) {
		r, out, _ := makeFullTestJPEG()
		rules := rulesFor("xmp")

		err := Strip(r, out, rules)
		if err != nil {
			t.Fatalf("Strip() unexpected error: %v", err)
		}
		got := out.Bytes()

		if bytes.Contains(got, []byte("http://ns.adobe.com/xap/1.0/")) { // APP1 XMP
			t.Fatalf("APP1 (XMP) not removed")
		}
	})

	t.Run("Removes COM from valid JPEG", func(t *testing.T) {
		r, out, _ := makeFullTestJPEG()
		rules := rulesFor("com")

		err := Strip(r, out, rules)
		if err != nil {
			t.Fatalf("Strip() unexpected error: %v", err)
		}
		got := out.Bytes()

		if testutil.ContainsMarker(got, 0xFE) { // COM
			t.Fatalf("COM not removed")
		}
	})

	t.Run("Removes EXIF and ICC when both selected", func(t *testing.T) {
		r, out, _ := makeFullTestJPEG()
		rules := rulesFor("exif", "icc")

		err := Strip(r, out, rules)
		if err != nil {
			t.Fatalf("Strip() unexpected error: %v", err)
		}
		got := out.Bytes()

		if bytes.Contains(got, []byte("Exif\x00\x00")) {
			t.Fatalf("APP1 (EXIF) not removed when EXIF+ICC selected")
		}
		if testutil.ContainsMarker(got, 0xE2) { // APP2
			t.Fatalf("ICC (APP2) not removed when EXIF+ICC selected")
		}
		if !bytes.Contains(got, []byte("http://ns.adobe.com/xap/1.0/")) {
			t.Fatalf("expected XMP APP1 to be preserved when EXIF+ICC selected")
		}
		if !testutil.ContainsMarker(got, 0xFE) {
			t.Fatalf("expected COM to be preserved when EXIF+ICC selected")
		}
	})

	t.Run("Removes EXIF, ICC and COM when selected", func(t *testing.T) {
		r, out, _ := makeFullTestJPEG()
		rules := rulesFor("exif", "icc", "com")

		err := Strip(r, out, rules)
		if err != nil {
			t.Fatalf("Strip() unexpected error: %v", err)
		}
		got := out.Bytes()

		// EXIF gone
		if bytes.Contains(got, []byte("Exif\x00\x00")) {
			t.Fatalf("APP1 (EXIF) not removed when EXIF+ICC+COM selected")
		}

		// ICC gone
		if testutil.ContainsMarker(got, 0xE2) { // APP2
			t.Fatalf("ICC (APP2) not removed when EXIF+ICC+COM selected")
		}

		// COM gone
		if testutil.ContainsMarker(got, 0xFE) { // COM
			t.Fatalf("COM not removed when EXIF+ICC+COM selected")
		}

		// XMP still present
		if !bytes.Contains(got, []byte("http://ns.adobe.com/xap/1.0/")) {
			t.Fatalf("expected XMP APP1 to be preserved when EXIF+ICC+COM selected")
		}
	})

	t.Run("Preserves JPEG structure", func(t *testing.T) {
		r, out, _ := makeFullTestJPEG()
		rules := rulesFor("exif")

		err := Strip(r, out, rules)
		if err != nil {
			t.Fatalf("Strip() unexpected error: %v", err)
		}
		got := out.Bytes()

		if !(len(got) >= 4 &&
			got[0] == 0xFF && got[1] == 0xD8 &&
			got[len(got)-2] == 0xFF && got[len(got)-1] == 0xD9) {
			t.Fatalf("output not bounded by SOI/EOI; head=% X tail=% X", got[:2], got[len(got)-2:])
		}

		if !testutil.ContainsMarker(got, 0xDB) {
			t.Fatal("expected DQT (0xDB) to be preserved, but it’s missing")
		}
	})

	t.Run("Output is smaller", func(t *testing.T) {
		r, out, img := makeFullTestJPEG()
		rules := rulesFor("exif")

		err := Strip(r, out, rules)
		if err != nil {
			t.Fatalf("Strip() unexpected error: %v", err)
		}
		got := out.Bytes()

		if len(got) >= len(img) {
			t.Errorf("expected stripped image to be smaller; got input=%d, output=%d", len(img), len(got))
		}
	})
}

func TestStripInvalidImage(t *testing.T) {
	rules := rulesFor("exif")

	// Not a JPEG (missing SOI)
	t.Run("NotJPEG missing SOI", func(t *testing.T) {
		data := []byte("not-a-jpeg")
		var out bytes.Buffer

		err := Strip(bytes.NewReader(data), &out, rules)
		if !errors.Is(err, ErrNotJPEG) {
			t.Fatalf("want ErrNotJPEG, got %v", err)
		}
	})

	// Truncated header (< 2 bytes)
	t.Run("Truncated header", func(t *testing.T) {
		data := []byte{0xFF} // only first byte of SOI
		var out bytes.Buffer

		err := Strip(bytes.NewReader(data), &out, rules)
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

		err := Strip(bytes.NewReader(b.Bytes()), &out, rules)
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
		err := Strip(bytes.NewReader(file), &out, rules)
		if !errors.Is(err, ErrTruncated) {
			t.Fatalf("want ErrTruncated, got %v", err)
		}
	})

	t.Run("Truncated after SOS missing final EOI", func(t *testing.T) {
		// SOS header
		sosHeader := []byte{0x00, 0x03, 0x01, 0x00, 0x02}
		sos := testutil.MakeSegment(0xDA, sosHeader) // FF DA <len> <hdr>
		var b bytes.Buffer
		b.Write([]byte{0xFF, 0xD8}) // SOI
		b.Write(sos)
		b.Write([]byte{0x11, 0x22, 0x33, 0x00, 0xFF, 0x00}) // fake scan data (no ending FFD9)

		var out bytes.Buffer
		err := Strip(bytes.NewReader(b.Bytes()), &out, rules)
		if !errors.Is(err, ErrTruncated) {
			t.Fatalf("want ErrTruncated when scan doesn't end with EOI, got %v", err)
		}
	})

	t.Run("Truncated immediately after SOI", func(t *testing.T) {
		data := []byte{0xFF, 0xD8} // SOI only
		var out bytes.Buffer

		err := Strip(bytes.NewReader(data), &out, rules)
		if !errors.Is(err, ErrTruncated) {
			t.Fatalf("want ErrTruncated, got %v", err)
		}
	})
}

func makeFullTestJPEG() (*bytes.Reader, *bytes.Buffer, []byte) {
	app1 := testutil.MakeSegment(0xE1, []byte("Exif\x00\x00SOME-EXIF-DATA"))
	xmp := testutil.MakeSegment(0xE1, []byte("http://ns.adobe.com/xap/1.0/ XMP-PAYLOAD"))
	icc := testutil.MakeSegment(0xE2, []byte("ICC_PROFILE\x00ICC-PAYLOAD"))
	com := testutil.MakeSegment(0xFE, []byte("some comment"))
	dqt := testutil.MakeSegment(0xDB, []byte{0x00})
	sos := testutil.MakeSOS([]byte{0x11, 0x22, 0x33, 0x00, 0xFF, 0x00})

	img := testutil.MakeJPEG(app1, xmp, icc, com, dqt, sos)

	r := bytes.NewReader(img)
	var out bytes.Buffer

	return r, &out, img
}
