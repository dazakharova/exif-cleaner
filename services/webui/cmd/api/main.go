package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "failed to parse form or file too large", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	metadataTypes := r.Form["metadataType"]

	if len(metadataTypes) == 0 {
		metadataTypes = []string{"EXIF"} // default
	}

	stripperURL := os.Getenv("STRIPPER_URL")
	if stripperURL == "" {
		stripperURL = "http://localhost:8080/strip"
	}

	u, err := url.Parse(stripperURL)
	if err != nil {
		http.Error(w, "invalid stripper URL", http.StatusInternalServerError)
		return
	}

	q := u.Query()
	for _, mt := range metadataTypes {
		q.Add("metadataType", mt)
	}
	u.RawQuery = q.Encode()

	stripperURL = u.String()

	req, err := http.NewRequest(r.Method, stripperURL, file)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "image/jpeg")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to send file to remote server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusBadGateway)
		io.CopyN(w, resp.Body, 1024)
		return
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, "cleaned.jpg"))
	w.Header().Set("Cache-Control", "no-store")

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("stream to client failed: %v", err)
	}

}

func runHealthcheck(port string) {
	url := "http://localhost:" + port + "/health"

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("healthcheck request failed: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("unhealthy status: %s", resp.Status)
		os.Exit(1)
	}

	os.Exit(0)
}

func main() {
	var healthcheck = flag.Bool("healthcheck", false, "run healthcheck and exit")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	flag.Parse()

	if *healthcheck {
		runHealthcheck(port)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthHandler)
	mux.Handle("/", http.FileServer(http.Dir("./public")))
	mux.HandleFunc("POST /upload", UploadHandler)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server started on port: %s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
