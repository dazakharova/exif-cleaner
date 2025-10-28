package jpegstrip_test

import (
	"bytes"
	"testing"

	"github.com/daria/exif-cleaner/services/stripper/internal/jpegstrip"
)

func TestStripValidImage(t *testing.T) {
	app1 := makeSegment(0xE1, []byte("Exif\x00\x00SOME-EXIF-DATA"))
	com := makeSegment(0xFE, []byte("some comment"))
	dqt := makeSegment(0xDB, []byte{0x00})
	sos := makeSOS([]byte{0x11, 0x22, 0x33, 0x00, 0xFF, 0x00})

	img := makeJPEG(app1, com, dqt, sos)

	r := bytes.NewReader(img)
	var out bytes.Buffer

	err := jpegstrip.Strip(r, &out)
	if err != nil {
		t.Fatalf("Strip() unexpected error: %v", err)
	}
	got := out.Bytes()

	t.Run("Removes APP1 and COM", func(t *testing.T) {
		if containsMarker(got, 0xE1) {
			t.Fatal("APP1 (EXIF) not removed")
		}
		if containsMarker(got, 0xFE) {
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
