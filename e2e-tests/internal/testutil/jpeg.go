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

func ContainsMetadata(respBody []byte, metadataType string) (bool, error) {
	mt := strings.ToLower(strings.TrimSpace(metadataType))

	switch mt {
	case "exif":
		// EXIF = APP1 with "Exif\x00\x00" prefix
		if bytes.Contains(respBody, []byte("Exif\x00\x00")) {
			return true, nil
		}
	case "xmp":
		// XMP = APP1 with XMP namespace prefix
		if bytes.Contains(respBody, []byte("http://ns.adobe.com/xap/1.0/")) {
			return true, nil
		}
	case "icc":
		// ICC = APP2 (0xE2) with "ICC_PROFILE\x00" prefix
		if bytes.Contains(respBody, []byte("ICC_PROFILE\x00")) || containsMarker(respBody, 0xE2) {
			return true, nil
		}
	case "comment", "com":
		// COM marker
		if containsMarker(respBody, 0xFE) {
			return true, nil
		}
	default:
		return false, fmt.Errorf("unsupported metadataType %q", metadataType)
	}

	return false, nil
}

func VerifyStripped(respBody []byte, metadataTypes []string) error {
	for _, mt := range metadataTypes {
		present, err := ContainsMetadata(respBody, mt)
		if err != nil {
			return err
		}
		if present {
			return fmt.Errorf("%s metadata still present in output", mt)
		}
	}

	return nil
}

func VerifyPreserved(respBody []byte, stripped []string) error {
	strippedSet := map[string]struct{}{}
	for _, mt := range stripped {
		mtNorm := strings.ToLower(strings.TrimSpace(mt))
		strippedSet[mtNorm] = struct{}{}
	}

	allTypes := []string{"exif", "xmp", "icc", "com"}

	for _, mt := range allTypes {
		_, isStripped := strippedSet[mt]

		if isStripped {
			continue
		}

		present, err := ContainsMetadata(respBody, mt)
		if err != nil {
			return err
		}

		if !present {
			return fmt.Errorf("%s metadata unexpectedly missing from output", mt)
		}
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
