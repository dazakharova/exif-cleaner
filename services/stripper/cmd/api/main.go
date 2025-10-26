package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello World"))
}

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

	var buf bytes.Buffer
	buf.Write(header[:n])
	_, err = io.Copy(&buf, r.Body)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	log.Printf("%s %s ct=%s size=%d", r.Method, r.URL.Path, r.Header.Get("Content-Type"), buf.Len())

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","size":%d}`, buf.Len())
}

func main() {
	port := "8080"

	mux := http.NewServeMux()
	mux.HandleFunc("/", RootHandler)
	mux.HandleFunc("/healthz", HealthHandler)
	mux.HandleFunc("/strip", StripHandler)

	log.Printf("Server started on port: %s", port)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
