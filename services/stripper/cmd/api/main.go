package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/daria/exif-cleaner/services/stripper/internal/jpegstrip"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func StripHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	defer r.Body.Close()

	header := make([]byte, 512)
	n, err := io.ReadFull(r.Body, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		http.Error(w, "failed to read file header", http.StatusBadRequest)
		return
	}

	contentType := http.DetectContentType(header[:n])
	if contentType != "image/jpeg" {
		http.Error(w, fmt.Sprintf("expected JPEG, got %s", contentType), http.StatusUnsupportedMediaType)
		return
	}

	q := r.URL.Query()
	metadataTypes := q["metadataType"]

	metadataRules := make(map[byte][]byte, len(metadataTypes))

	for _, t := range metadataTypes {
		marker, prefix := jpegstrip.MarkerFor(t)
		if marker != 0 {
			metadataRules[marker] = prefix
		}
	}

	fullReader := io.MultiReader(bytes.NewReader(header[:n]), r.Body)

	var buf bytes.Buffer
	if err := jpegstrip.Strip(fullReader, &buf, metadataRules); err != nil {
		http.Error(w, "failed to process JPEG", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", fmt.Sprint(buf.Len()))

	if _, err := io.Copy(w, &buf); err != nil {
		return
	}
}

func runHealthcheck(port string) {
	url := "http://localhost:" + port + "/health"

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("healthcheck request failed: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("unhealthy status: %s", resp.Status)
		os.Exit(1)
	}

	os.Exit(0)
}

func main() {
	var healthcheck = flag.Bool("healthcheck", false, "run healthcheck and exit")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	flag.Parse()

	if *healthcheck {
		runHealthcheck(port)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthHandler)
	mux.HandleFunc("POST /strip", StripHandler)

	log.Printf("Server started on port: %s", port)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
