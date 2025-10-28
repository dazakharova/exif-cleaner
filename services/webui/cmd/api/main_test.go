package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("../../public")))
	mux.HandleFunc("/upload", UploadHandler)

	t.Run("GET / serves index.html", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("got %d, want %d", rec.Code, http.StatusOK)
		}

		if !strings.Contains(rec.Body.String(), "<title>EXIF Cleaner</title>") {
			t.Fatalf("did not serve index.html")
		}
	})
}
