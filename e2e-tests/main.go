package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	MaxWaitAttempts = 10
	WaitDelay       = 3 * time.Second
)

func performSingleCheck(url string) error {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", url, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service at %s returned non-200 status code: %d", url, resp.StatusCode)
	}

	return nil
}

func waitForService(name, url string, maxAttempts int, delay time.Duration) {
	for i := 0; i < maxAttempts; i++ {
		if err := performSingleCheck(url); err == nil {
			log.Printf("%s service is ready!", name)
			return
		}
		time.Sleep(delay)
	}
	log.Fatalf("FATAL: %s service failed to become ready after %d attempts.", name, maxAttempts)
}

func main() {
	webuiBaseURL := os.Getenv("WEBUI_URL")
	stripperBaseURL := os.Getenv("STRIPPER_URL")
	if webuiBaseURL == "" {
		webuiBaseURL = "http://localhost:3000"
	}
	if stripperBaseURL == "" {
		stripperBaseURL = "http://localhost:8080"
	}

	waitForService("WebUI", webuiBaseURL+"/health", MaxWaitAttempts, WaitDelay)
	waitForService("Stripper", stripperBaseURL+"/health", MaxWaitAttempts, WaitDelay)

	fmt.Printf("Stripper Health Check Complete\n")
}
