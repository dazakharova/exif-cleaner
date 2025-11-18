package e2e

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/daria/exif-cleaner/e2e-tests/internal/runner"
)

func TestE2E(t *testing.T) {
	webuiBaseURL := os.Getenv("WEBUI_URL")

	if webuiBaseURL == "" {
		webuiBaseURL = "http://localhost:3000"
	}

	waitForService(t, webuiBaseURL+"/health")

	runner.Run(t, webuiBaseURL)
}

func waitForService(t *testing.T, url string) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			return
		}
		time.Sleep(3 * time.Second)
	}

	t.Fatalf("Service not ready: %s", url)
}
