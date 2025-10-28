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
	t.Run("Valid JPEG", func(t *testing.T) {
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
}
