package runner

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/daria/exif-cleaner/e2e-tests/internal/httpc"
	"github.com/daria/exif-cleaner/e2e-tests/internal/testutil"
)

func RunE2ETest(baseURL, metadataType, filename string) error {
	req, err := httpc.NewUploadRequest(baseURL, metadataType, filename)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, body, err := httpc.DoRequest(ctx, req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		if err := testutil.VerifyResponseHeaders(resp); err != nil {
			return fmt.Errorf("Header validation failed: %v", err)
		}

		if !testutil.HasValidJPEGStructure(body) {
			return fmt.Errorf("Invalid JPEG structure: missing SOI/EOI")
		}

		if err := testutil.VerifyStripped(body, metadataType); err != nil {
			return fmt.Errorf("Strip verification failed: %v", err)
		}
	}

	return nil
}
