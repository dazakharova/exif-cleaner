package runner

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/daria/exif-cleaner/e2e-tests/internal/httpc"
	"github.com/daria/exif-cleaner/e2e-tests/internal/testutil"
)

type scenario struct {
	name           string
	metaType       string
	filename       string
	wantStatus     int
	shouldValidate bool
}

func runTestScenario(t *testing.T, baseURL string, s scenario) {
	t.Helper()

	req, err := httpc.NewUploadRequest(baseURL, s.metaType, s.filename)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, body, err := httpc.DoRequest(ctx, req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != s.wantStatus {
		t.Fatalf("[%s] wrong status: got %d, want %d", s.name, resp.StatusCode, s.wantStatus)
	}

	if s.shouldValidate {
		validateHappyPath(t, s, resp, body)
	}
}

func validateHappyPath(t *testing.T, s scenario, resp *http.Response, body []byte) {
	t.Helper()
	if err := testutil.VerifyResponseHeaders(resp); err != nil {
		t.Fatalf("[%s] header validation failed: %v", s.name, err)
	}

	if !testutil.HasValidJPEGStructure(body) {
		t.Fatalf("[%s] invalid JPEG structure: missing SOI/EOI", s.name)
	}

	if err := testutil.VerifyStripped(body, s.metaType); err != nil {
		t.Fatalf("[%s] strip verification failed: %v", s.name, err)
	}
}

func Run(t *testing.T, baseURL string) {
	t.Helper()

	testScenarios := []scenario{
		{name: "Strip EXIF metadata", metaType: "exif", filename: "./testdata/test_valid.jpg", wantStatus: http.StatusOK, shouldValidate: true},
		{name: "Strip ICC metadata", metaType: "icc", filename: "./testdata/test_valid.jpg", wantStatus: http.StatusOK, shouldValidate: true},
		{name: "Strip XMP metadata", metaType: "xmp", filename: "./testdata/test_valid.jpg", wantStatus: http.StatusOK, shouldValidate: true},
		{name: "Strip COM metadata", metaType: "com", filename: "./testdata/test_valid.jpg", wantStatus: http.StatusOK, shouldValidate: true},

		// Error paths
		{name: "Reject PNG via WebUI", metaType: "exif", filename: "./testdata/not_jpeg.png", wantStatus: http.StatusBadGateway, shouldValidate: false},
		{name: "Reject truncated JPEG", metaType: "exif", filename: "./testdata/truncated_jpeg.jpg", wantStatus: http.StatusBadGateway, shouldValidate: false},
	}

	for _, s := range testScenarios {
		t.Run(s.name, func(t *testing.T) {
			runTestScenario(t, baseURL, s)
		})
	}
}
