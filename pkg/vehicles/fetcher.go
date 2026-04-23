package vehicles

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

// ManifestEntry holds per-vehicle metadata in the remote manifest.
type ManifestEntry struct {
	LastModified time.Time `json:"lastModified"`
}

// Manifest is the structure served by the remote update server.
type Manifest struct {
	Vehicles map[string]ManifestEntry `json:"vehicles"`
}

// Fetcher defines the interface for retrieving vehicle data from a remote source.
type Fetcher interface {
	FetchManifest(ctx context.Context) (*Manifest, error)
	FetchVehicle(ctx context.Context, id int) (Vehicle, error)
}

// HTTPFetcher retrieves vehicle data from a remote HTTP service.
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

	var manifest Manifest

	err = json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, fmt.Errorf("parse manifest JSON: %w", err)
	}

	return &manifest, nil
}

// FetchVehicle retrieves a single vehicle by CarID from the remote service.
func (f *HTTPFetcher) FetchVehicle(ctx context.Context, vehicleID int) (Vehicle, error) {
	vehicleURL, err := url.JoinPath(f.baseURL, fmt.Sprintf("%d.json", vehicleID))
	if err != nil {
		return Vehicle{}, fmt.Errorf("build URL for vehicle %d: %w", vehicleID, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, vehicleURL, nil)
	if err != nil {
		return Vehicle{}, fmt.Errorf("create request for vehicle %d: %w", vehicleID, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return Vehicle{}, fmt.Errorf("fetch vehicle %d: %w", vehicleID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Vehicle{}, fmt.Errorf("fetch vehicle %d: %w: %d", vehicleID, ErrUnexpectedStatusCode, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB limit
	if err != nil {
		return Vehicle{}, fmt.Errorf("read response for vehicle %d: %w", vehicleID, err)
	}

	var vehicle Vehicle

	err = json.Unmarshal(body, &vehicle)
	if err != nil {
		return Vehicle{}, fmt.Errorf("parse vehicle %d JSON: %w", vehicleID, err)
	}

	return vehicle, nil
}
