package circuits_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/zetetos/gt-telemetry/v2/pkg/circuits"
	"github.com/zetetos/gt-telemetry/v2/pkg/models"
)

type FetcherTestSuite struct {
	suite.Suite
}

func TestFetcherTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FetcherTestSuite))
}

// --- NewHTTPFetcher tests ---

func (suite *FetcherTestSuite) TestNewHTTPFetcherTrimsTrailingSlash() {
	// Arrange
	var requestedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(circuits.Manifest{}) //nolint:errcheck // test helper
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL + "/")

	// Act
	_, err := fetcher.FetchManifest(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.Equal("/manifest.json", requestedPath)
}

// --- FetchManifest tests ---

func (suite *FetcherTestSuite) TestFetchManifestReturnsManifestData() {
	// Arrange
	wantManifest := circuits.Manifest{
		Circuits: map[string]circuits.ManifestEntry{
			"TrackA": {LastModified: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
			"TrackB": {LastModified: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("/manifest.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantManifest) //nolint:errcheck // test helper
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	// Act
	got, err := fetcher.FetchManifest(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.Len(got.Circuits, 2)
	suite.Contains(got.Circuits, "TrackA")
	suite.Contains(got.Circuits, "TrackB")
}

func (suite *FetcherTestSuite) TestFetchManifestConstructsCorrectURL() {
	// Arrange
	var requestedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(circuits.Manifest{}) //nolint:errcheck // test helper
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	// Act
	_, err := fetcher.FetchManifest(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.Equal("/manifest.json", requestedPath)
}

func (suite *FetcherTestSuite) TestFetchManifestReturnsErrorOnNon200() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	// Act
	got, err := fetcher.FetchManifest(suite.T().Context())

	// Assert
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, circuits.ErrUnexpectedStatusCode)
	suite.Contains(err.Error(), "500")
	suite.Nil(got)
}

func (suite *FetcherTestSuite) TestFetchManifestReturnsErrorOnMalformedJSON() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	// Act
	got, err := fetcher.FetchManifest(suite.T().Context())

	// Assert
	suite.Require().Error(err)
	suite.Contains(err.Error(), "parse manifest JSON")
	suite.Nil(got)
}

func (suite *FetcherTestSuite) TestFetchManifestReturnsErrorOnCancelledContext() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	got, err := fetcher.FetchManifest(ctx)

	// Assert
	suite.Require().ErrorIs(err, context.Canceled)
	suite.Nil(got)
}

// --- FetchCircuit tests ---

func (suite *FetcherTestSuite) TestFetchCircuitReturnsCircuitInfo() {
	// Arrange
	wantCircuit := circuits.CircuitInfo{
		ID:        "TestCircuit",
		Name:      "Test Circuit",
		Country:   "jp",
		StartLine: models.CoordinateNorm{X: 100, Y: 0, Z: 200},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("/TestCircuit.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantCircuit) //nolint:errcheck // test helper
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	// Act
	got, err := fetcher.FetchCircuit(suite.T().Context(), "TestCircuit")

	// Assert
	suite.Require().NoError(err)
	suite.Equal(wantCircuit.ID, got.ID)
	suite.Equal(wantCircuit.Name, got.Name)
	suite.Equal(wantCircuit.Country, got.Country)
}

func (suite *FetcherTestSuite) TestFetchCircuitConstructsCorrectURL() {
	// Arrange
	var requestedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(circuits.CircuitInfo{ID: "TestCircuit"}) //nolint:errcheck // test helper
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	// Act
	_, err := fetcher.FetchCircuit(suite.T().Context(), "TestCircuit")

	// Assert
	suite.Require().NoError(err)
	suite.Equal("/TestCircuit.json", requestedPath)
}

func (suite *FetcherTestSuite) TestFetchCircuitReturnsErrorOnNon200() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	// Act
	got, err := fetcher.FetchCircuit(suite.T().Context(), "Missing")

	// Assert
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, circuits.ErrUnexpectedStatusCode)
	suite.Contains(err.Error(), "404")
	suite.Nil(got)
}

func (suite *FetcherTestSuite) TestFetchCircuitReturnsErrorOnMalformedJSON() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	// Act
	got, err := fetcher.FetchCircuit(suite.T().Context(), "InvalidCircuit")

	// Assert
	suite.Require().Error(err)
	suite.Contains(err.Error(), "parse circuit InvalidCircuit JSON")
	suite.Nil(got)
}

func (suite *FetcherTestSuite) TestFetchCircuitFallsBackToCircuitIDWhenIDIsEmpty() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(circuits.CircuitInfo{ID: "", Name: "Test Circuit"}) //nolint:errcheck // test helper
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	// Act
	got, err := fetcher.FetchCircuit(suite.T().Context(), "FallbackID")

	// Assert
	suite.Require().NoError(err)
	suite.Equal("FallbackID", got.ID)
}

func (suite *FetcherTestSuite) TestFetchCircuitReturnsErrorOnCancelledContext() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := circuits.NewHTTPFetcher(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	got, err := fetcher.FetchCircuit(ctx, "TestCircuit")

	// Assert
	suite.Require().ErrorIs(err, context.Canceled)
	suite.Nil(got)
}
