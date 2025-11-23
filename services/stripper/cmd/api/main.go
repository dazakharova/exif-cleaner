package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/daria/exif-cleaner/services/stripper/internal/jpegstrip"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func StripHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/strip", StripHandler)

	log.Printf("Server started on port: %s", port)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
