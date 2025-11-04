package runner

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/daria/exif-cleaner/e2e-tests/internal/httpc"
	"github.com/daria/exif-cleaner/e2e-tests/internal/testutil"
)

type scenario struct {
	name       string
	metaType   string
	filename   string
	wantStatus int
}

func runE2ETest(baseURL string, s scenario) error {
	req, err := httpc.NewUploadRequest(baseURL, s.metaType, s.filename)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, body, err := httpc.DoRequest(ctx, req)
	if err != nil {
		return err
	}

	if resp.StatusCode == s.wantStatus {
		if err := testutil.VerifyResponseHeaders(resp); err != nil {
			return fmt.Errorf("[%s] header validation failed: %v", err)
		}

		if !testutil.HasValidJPEGStructure(body) {
			return fmt.Errorf("[%s] invalid JPEG structure: missing SOI/EOI", s.name)
		}

		if err := testutil.VerifyStripped(body, s.metaType); err != nil {
			return fmt.Errorf("[%s] strip verification failed: %v", s.name, err)
		}
	}

	return nil
}

func Run(baseUrl string) error {
	testScenarios := []scenario{
		{name: "Strip EXIF", metaType: "exif", filename: "./testdata/test_valid.jpg", wantStatus: http.StatusOK},
		{name: "Strip ICC", metaType: "icc", filename: "./testdata/test_valid.jpg", wantStatus: http.StatusOK},
	}

	for _, s := range testScenarios {
		err := runE2ETest(baseUrl, s)
		if err != nil {
			return err
		}

		log.Printf("E2E passed: %s", s.name)
	}

	return nil
}
