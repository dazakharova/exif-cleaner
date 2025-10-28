package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daria/exif-cleaner/services/stripper/internal/testutil"
)

func TestRootHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	RootHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "Hello World" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	HealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	wantBody := `{"status":"ok"}`
	gotBody := rec.Body.String()

	if gotBody != wantBody {
		t.Fatalf("expected %q, got %q", wantBody, gotBody)
	}
}

func TestStripHandler(t *testing.T) {
	t.Run("POST valid JPEG returns status 200 and image/jpeg content", func(t *testing.T) {
		app1 := testutil.MakeSegment(0xE1, []byte("Exif\x00\x00something"))
		com := testutil.MakeSegment(0xFE, []byte("comment"))
		sos := testutil.MakeSOS([]byte{0x11, 0x22, 0x33})
		jpeg := testutil.MakeJPEG(app1, com, sos)

		req := httptest.NewRequest(http.MethodPost, "/strip", bytes.NewReader(jpeg))
		rec := httptest.NewRecorder()

		StripHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
			t.Fatalf("Content-Type = %q", got)
		}
	})

	t.Run("GET /strip returns 405", func(t *testing.T) {
		app1 := testutil.MakeSegment(0xE1, []byte("Exif\x00\x00something"))
		com := testutil.MakeSegment(0xFE, []byte("comment"))
		sos := testutil.MakeSOS([]byte{0x11, 0x22, 0x33})
		jpeg := testutil.MakeJPEG(app1, com, sos)

		req := httptest.NewRequest(http.MethodGet, "/strip", bytes.NewReader(jpeg))
		rec := httptest.NewRecorder()

		StripHandler(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("POST non-JPEG returns 415", func(t *testing.T) {
		var b bytes.Buffer
		app1 := testutil.MakeSegment(0xE1, []byte("Exif\x00\x00something"))
		com := testutil.MakeSegment(0xFE, []byte("comment"))
		sos := testutil.MakeSOS([]byte{0x11, 0x22, 0x33})

		b.Write(app1)
		b.Write(com)
		b.Write(sos)
		b.Write([]byte{0xFF, 0xD9})

		req := httptest.NewRequest(http.MethodPost, "/strip", bytes.NewReader(b.Bytes()))
		rec := httptest.NewRecorder()

		StripHandler(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d (body=%q)", rec.Code, rec.Body.String())
		}
	})
}
