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
	"time"
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

	n, _ := io.Copy(io.Discard, resp.Body)

	log.Printf("Status: %s", resp.Status)
	log.Printf("Content-Type: %s", resp.Header.Get("Content-Type"))
	log.Printf("Content-Disposition: %s", resp.Header.Get("Content-Disposition"))
	log.Printf("Cache-Control: %s", resp.Header.Get("Cache-Control"))
	log.Printf("Response size: %d bytes", n)

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
