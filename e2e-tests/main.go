package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daria/exif-cleaner/e2e-tests/internal/testutil"
)

const (
	MaxWaitAttempts = 10
	WaitDelay       = 3 * time.Second
)

func performSingleCheck(url string) error {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", url, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service at %s returned non-200 status code: %d", url, resp.StatusCode)
	}

	return nil
}

func waitForService(name, url string, maxAttempts int, delay time.Duration) {
	for i := 0; i < maxAttempts; i++ {
		if err := performSingleCheck(url); err == nil {
			log.Printf("%s service is ready!", name)
			return
		}
		time.Sleep(delay)
	}
	log.Fatalf("FATAL: %s service failed to become ready after %d attempts.", name, maxAttempts)
}

func formMultipartFile(w *multipart.Writer, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	fileWriter, err := w.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return err
	}

	if _, err := io.Copy(fileWriter, f); err != nil {
		return err
	}

	return nil
}

func runEndToEndTests(webuiURL string) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	err := formMultipartFile(w, "./testdata/test_valid.jpg")
	if err != nil {
		log.Fatal(err)
	}

	err = w.WriteField("metadataType", "exif")
	if err != nil {
		log.Fatal(err)
	}

	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", webuiURL, &body)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if err := verifyResponseHeaders(resp); err != nil {
		log.Fatalf("Header validation failed: %v", err)
	}

	if !hasValidJPEGStructure(respBody) {
		log.Fatalf("Invalid JPEG structure: missing SOI/EOI")
	}

	if err := verifyStripped(respBody, "exif"); err != nil {
		log.Fatalf("Strip verification failed: %v", err)
	}

	log.Printf("Status: %s", resp.Status)
	log.Printf("Content-Type: %s", resp.Header.Get("Content-Type"))
	log.Printf("Content-Disposition: %s", resp.Header.Get("Content-Disposition"))
	log.Printf("Cache-Control: %s", resp.Header.Get("Cache-Control"))

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status %d from WebUI", resp.StatusCode)
	}
}

func main() {
	webuiBaseURL := os.Getenv("WEBUI_URL")
	stripperBaseURL := os.Getenv("STRIPPER_URL")
	if webuiBaseURL == "" {
		webuiBaseURL = "http://localhost:3000"
	}
	if stripperBaseURL == "" {
		stripperBaseURL = "http://localhost:8080"
	}

	waitForService("WebUI", webuiBaseURL+"/health", MaxWaitAttempts, WaitDelay)
	waitForService("Stripper", stripperBaseURL+"/health", MaxWaitAttempts, WaitDelay)

	runEndToEndTests(webuiBaseURL + "/upload")

	fmt.Printf("Stripper Health Check Complete\n")
}

func verifyResponseHeaders(resp *http.Response) error {
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

func hasValidJPEGStructure(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 0xFF && data[1] == 0xD8 && // SOI
		data[len(data)-2] == 0xFF && data[len(data)-1] == 0xD9 // EOI
}

func verifyStripped(respBody []byte, metadataType string) error {
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
		if bytes.Contains(respBody, []byte("ICC_PROFILE\x00")) || testutil.ContainsMarker(respBody, 0xE2) {
			return fmt.Errorf("ICC profile still present in output")
		}
	case "comment", "com":
		// COM marker
		if testutil.ContainsMarker(respBody, 0xFE) {
			return fmt.Errorf("COM marker still present in output")
		}
	default:
		return fmt.Errorf("unsupported metadataType %q", metadataType)
	}

	return nil
}
