package circuits

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

const httpTimeout = 10 * time.Second

// ErrUnexpectedStatusCode indicates an HTTP response with a non-200 status code.
var ErrUnexpectedStatusCode = errors.New("unexpected HTTP status code")

// Fetcher defines the interface for retrieving circuit data from a remote source.
type Fetcher interface {
	FetchManifest(ctx context.Context) (*Manifest, error)
	FetchCircuit(ctx context.Context, circuitID string) (*CircuitInfo, error)
}

// HTTPFetcher retrieves circuit data from a remote HTTP service.
type HTTPFetcher struct {
	baseURL string
	client  *http.Client
}

// NewHTTPFetcher creates a new HTTPFetcher pointed at the given base URL.
func NewHTTPFetcher(baseURL string) *HTTPFetcher {
	return &HTTPFetcher{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// FetchManifest downloads and parses the manifest from the remote server.
func (f *HTTPFetcher) FetchManifest(ctx context.Context) (*Manifest, error) {
	manifestURL, err := url.JoinPath(f.baseURL, "manifest.json")
	if err != nil {
		return nil, fmt.Errorf("build URL for manifest: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for manifest: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch manifest: %w: %d", ErrUnexpectedStatusCode, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response for manifest: %w", err)
	}

	var remoteManifest Manifest

	err = json.Unmarshal(body, &remoteManifest)
	if err != nil {
		return nil, fmt.Errorf("parse manifest JSON: %w", err)
	}

	return &remoteManifest, nil
}

// FetchCircuit downloads and parses a single circuit file from the remote server.
func (f *HTTPFetcher) FetchCircuit(ctx context.Context, circuitID string) (*CircuitInfo, error) {
	circuitURL, err := url.JoinPath(f.baseURL, circuitID+".json")
	if err != nil {
		return nil, fmt.Errorf("build URL for circuit %s: %w", circuitID, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, circuitURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for circuit %s: %w", circuitID, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch circuit %s: %w", circuitID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch circuit %s: %w: %d", circuitID, ErrUnexpectedStatusCode, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response for circuit %s: %w", circuitID, err)
	}

	var info CircuitInfo

	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, fmt.Errorf("parse circuit %s JSON: %w", circuitID, err)
	}

	if info.ID == "" {
		info.ID = circuitID
	}

	return &info, nil
}
