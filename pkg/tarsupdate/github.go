package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GitHubClient queries release information from GitHub.
type GitHubClient struct {
	Owner      string
	Repo       string
	HTTPClient *http.Client
}

// NewGitHubClient creates a client for the given owner/repo.
func NewGitHubClient(owner, repo string) *GitHubClient {
	return &GitHubClient{
		Owner:      owner,
		Repo:       repo,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ReleaseInfo holds parsed release metadata.
type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
	HTMLURL     string    `json:"html_url"`
}

// Asset is a single release asset.
type Asset struct {
	Name        string `json:"name"`
	Size        int    `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}

// LatestRelease fetches the latest release.
func (c *GitHubClient) LatestRelease() (ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", c.Owner, c.Repo)
	return c.fetchRelease(url)
}

// ListReleases fetches the N most recent releases.
func (c *GitHubClient) ListReleases(perPage int) ([]ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", c.Owner, c.Repo, perPage)
	resp, err := c.get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var releases []ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode releases: %w", err)
	}
	return releases, nil
}

// FindAssetURL returns the download URL for an asset with the given name.
func (r *ReleaseInfo) FindAssetURL(name string) (string, error) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a.DownloadURL, nil
		}
	}
	return "", fmt.Errorf("asset %s not found in release %s", name, r.TagName)
}

// FindAsset returns the asset with the given name.
func (r *ReleaseInfo) FindAsset(name string) (Asset, error) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, nil
		}
	}
	return Asset{}, fmt.Errorf("asset %s not found in release %s", name, r.TagName)
}

func (c *GitHubClient) fetchRelease(url string) (ReleaseInfo, error) {
	resp, err := c.get(url)
	if err != nil {
		return ReleaseInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ReleaseInfo{}, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ReleaseInfo{}, fmt.Errorf("decode release: %w", err)
	}
	return rel, nil
}

func (c *GitHubClient) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "tars-update/1.0")

	// Use GitHub token if available via environment
	if token := getEnvToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.HTTPClient.Do(req)
}

// DownloadAsset downloads a release asset to a local file path.
func (c *GitHubClient) DownloadAsset(url, dstPath string) error {
	resp, err := c.get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	// Write atomically via temp file
	tmpPath := dstPath + ".tmp"
	if err := writeFile(tmpPath, data); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	return renameFile(tmpPath, dstPath)
}

// DownloadSHA256SUMS downloads the SHA256SUMS.txt file and parses it.
func (c *GitHubClient) DownloadSHA256SUMS(rel ReleaseInfo) (map[string]string, error) {
	url, err := rel.FindAssetURL("SHA256SUMS.txt")
	if err != nil {
		return nil, err
	}

	resp, err := c.get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download SHA256SUMS returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseSHA256SUMS(string(data)), nil
}

func parseSHA256SUMS(content string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			hash := parts[0]
			name := parts[1]
			// Handle "*filename" format from sha256sum -b
			name = strings.TrimPrefix(name, "*")
			m[name] = hash
		}
	}
	return m
}

func getEnvToken() string {
	// Check common env vars for GitHub token
	for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN", "TARS_GITHUB_TOKEN"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// os and renameFile / writeFile defined in update.go to avoid duplication
