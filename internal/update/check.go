package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gobchat-log-browser/internal/version"
)

// DefaultEndpoint is the GitHub "latest release" API for this project.
const DefaultEndpoint = "https://api.github.com/repos/Shuro/Gobchat-Log-Browser/releases/latest"

const requestTimeout = 10 * time.Second

// Release is the subset of the GitHub release payload the checker needs.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// Client fetches the latest release. Endpoint and HTTPClient are overridable
// for tests.
type Client struct {
	Endpoint   string
	HTTPClient *http.Client
}

// NewClient returns a client against the real GitHub API with a sane timeout.
func NewClient() *Client {
	return &Client{
		Endpoint:   DefaultEndpoint,
		HTTPClient: &http.Client{Timeout: requestTimeout},
	}
}

// LatestRelease fetches the newest published release. Any non-200 response
// (including 403/429 rate limiting) is an error carrying the status code.
func (c *Client) LatestRelease(ctx context.Context) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint, nil)
	if err != nil {
		return Release{}, err
	}
	// GitHub rejects requests without a User-Agent.
	req.Header.Set("User-Agent", "Gobchat-Log-Browser/"+version.Version)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, c.Endpoint)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return Release{}, err
	}
	return rel, nil
}
