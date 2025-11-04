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
	req, err := newUploadRequest(webuiURL, "exif", "./testdata/test_valid.jpg")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, respBody, err := doRequest(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	if err := verifyResponseHeaders(resp); err != nil {
		log.Fatalf("Header validation failed: %v", err)
	}

	if !testutil.HasValidJPEGStructure(respBody) {
		log.Fatalf("Invalid JPEG structure: missing SOI/EOI")
	}

	if err := testutil.VerifyStripped(respBody, "exif"); err != nil {
		log.Fatalf("Strip verification failed: %v", err)
	}

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

	runEndToEndTests(webuiBaseURL)

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

func newUploadRequest(baseURL, metaType, filename string) (*http.Request, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if err := formMultipartFile(w, filename); err != nil {
		return nil, err
	}

	if err := w.WriteField("metadataType", metaType); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/upload", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, nil
}

func doRequest(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}
	return resp, b, nil
}
