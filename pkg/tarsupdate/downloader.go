package update

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Downloader handles downloading and verifying release assets.
type Downloader struct {
	HTTPClient *http.Client
}

// NewDownloader creates a new downloader.
func NewDownloader() *Downloader {
	return &Downloader{
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// DownloadResult holds the path to the downloaded file.
type DownloadResult struct {
	Path   string
	SHA256 string
	Size   int64
}

// Download fetches an asset to a temporary location.
func (d *Downloader) Download(asset Asset, dir string) (*DownloadResult, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	outPath := fmt.Sprintf("%s/%s", dir, asset.Name)
	out, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	req, err := http.NewRequest(http.MethodGet, asset.DownloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "tars-update/1.0")

	resp, err := d.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}

	hash := sha256.New()
	mw := io.MultiWriter(out, hash)
	size, err := io.Copy(mw, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download copy: %w", err)
	}

	return &DownloadResult{
		Path:   outPath,
		SHA256: fmt.Sprintf("%x", hash.Sum(nil)),
		Size:   size,
	}, nil
}

// VerifySHA256 checks the file hash against an expected value from SHA256SUMS.txt.
func VerifySHA256(filePath, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := fmt.Sprintf("%x", h.Sum(nil))
	if !strings.EqualFold(actual, strings.TrimSpace(expectedHash)) {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", expectedHash, actual)
	}
	return nil
}

// ParseSHA256SUMS parses a SHA256SUMS.txt content into a map filename->hash.
func ParseSHA256SUMS(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// Handle both "HASH  filename" and "HASH *filename" formats
			hash := parts[0]
			fname := strings.TrimPrefix(parts[1], "*")
			result[fname] = hash
		}
	}
	return result
}
