package gttelemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// RemoteVersion holds the lastModified timestamps for each data category
// as served by the remote version.json endpoint.
type RemoteVersion struct {
	Circuits struct {
		LastModified time.Time `json:"lastModified"`
	} `json:"circuits"`
	Vehicles struct {
		LastModified time.Time `json:"lastModified"`
	} `json:"vehicles"`
}

var errUnexpectedVersionStatus = errors.New("unexpected HTTP status code")

func fetchVersion(ctx context.Context, baseURL string) (*RemoteVersion, error) {
	versionURL, err := url.JoinPath(baseURL, "version.json")
	if err != nil {
		return nil, fmt.Errorf("build version URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for version: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch version: %w: %d", errUnexpectedVersionStatus, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read version response: %w", err)
	}

	var version RemoteVersion

	err = json.Unmarshal(body, &version)
	if err != nil {
		return nil, fmt.Errorf("parse version JSON: %w", err)
	}

	return &version, nil
}
