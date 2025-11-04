package httpc

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func formMultipartFile(w *multipart.Writer, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	fileWriter, err := w.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return err
	}

	if _, err := io.Copy(fileWriter, f); err != nil {
		return err
	}

	return nil
}

func NewUploadRequest(baseURL, metaType, filename string) (*http.Request, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if err := formMultipartFile(w, filename); err != nil {
		return nil, err
	}

	if err := w.WriteField("metadataType", metaType); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/upload", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, nil
}

func DoRequest(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}
	return resp, b, nil
}
