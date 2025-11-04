package testutil

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

func HasValidJPEGStructure(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 0xFF && data[1] == 0xD8 && // SOI
		data[len(data)-2] == 0xFF && data[len(data)-1] == 0xD9 // EOI
}

func containsMarker(image []byte, marker byte) bool {
	for i := 0; i+1 < len(image); i++ {
		if image[i] == 0xFF && image[i+1] == marker {
			return true
		}
	}
	return false
}

func VerifyStripped(respBody []byte, metadataType string) error {
	mt := strings.ToLower(strings.TrimSpace(metadataType))

	switch mt {
	case "exif":
		// EXIF = APP1 with "Exif\x00\x00" prefix
		if bytes.Contains(respBody, []byte("Exif\x00\x00")) {
			return fmt.Errorf("EXIF prefix still present in output")
		}
	case "xmp":
		// XMP = APP1 with XMP namespace prefix
		if bytes.Contains(respBody, []byte("http://ns.adobe.com/xap/1.0/")) {
			return fmt.Errorf("XMP prefix still present in output")
		}
	case "icc":
		// ICC = APP2 (0xE2) with "ICC_PROFILE\x00" prefix
		if bytes.Contains(respBody, []byte("ICC_PROFILE\x00")) || containsMarker(respBody, 0xE2) {
			return fmt.Errorf("ICC profile still present in output")
		}
	case "comment", "com":
		// COM marker
		if containsMarker(respBody, 0xFE) {
			return fmt.Errorf("COM marker still present in output")
		}
	default:
		return fmt.Errorf("unsupported metadataType %q", metadataType)
	}

	return nil
}

func VerifyResponseHeaders(resp *http.Response) error {
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		return fmt.Errorf("unexpected Content-Type: %s", ct)
	}

	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		return fmt.Errorf("missing Content-Disposition header")
	}
	if !strings.Contains(cd, `filename="cleaned.jpg"`) {
		return fmt.Errorf("unexpected Content-Disposition filename: %q", cd)
	}

	if cc := resp.Header.Get("Cache-Control"); cc == "" {
		return fmt.Errorf("missing Cache-Control header")
	}

	return nil
}
