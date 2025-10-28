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

	t.Run("Remove APP1 and COM", func(t *testing.T) {
		r := bytes.NewReader(img)
		var out bytes.Buffer

		err := jpegstrip.Strip(r, &out)
		if err != nil {
			t.Fatalf("Strip() unexpected error: %v", err)
		}
		got := out.Bytes()

		if containsMarker(got, 0xE1) { // APP1
			t.Fatal("expected APP1 (EXIF) to be removed, but found APP1 marker in output")
		}
		if containsMarker(got, 0xFE) { // COM
			t.Fatal("expected COM to be removed, but found COM marker in output")
		}
	})
}
