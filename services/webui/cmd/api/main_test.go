package main

import (
	"bytes"
	"mime/multipart"
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

	t.Run("GET /style.css has text/css", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
			t.Fatalf("Content-Type=%q", ct)
		}
	})

	t.Run("POST /upload without file field -> 400", func(t *testing.T) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		// no file part added
		_ = w.WriteField("foo", "bar")
		_ = w.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())

		rec := httptest.NewRecorder()
		UploadHandler(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}
