package transfer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

const maxArchiveBytes = 20 << 20

// Download and Upload use presigned URLs without forwarding the CLI credential.
func Download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxArchiveBytes {
		return nil, fmt.Errorf("download exceeds 20 MiB")
	}
	return data, nil
}

// Upload PUTs data with the content type the URL was presigned for.
func Upload(url, contentType string, data []byte) error {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload: HTTP %d", resp.StatusCode)
	}
	return nil
}
