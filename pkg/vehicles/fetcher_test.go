package vehicles_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/zetetos/gt-telemetry/v2/pkg/vehicles"
)

type FetcherTestSuite struct {
	suite.Suite
}

func TestFetcherTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FetcherTestSuite))
}

func (suite *FetcherTestSuite) TestFetchVehicleWithValidID() {
	// Arrange
	wantVehicle := vehicles.Vehicle{
		CarID:        42,
		Manufacturer: "Nissan",
		Model:        "GT-R '17",
		Year:         2017,
		Drivetrain:   "4WD",
		Aspiration:   "TC",
		EngineLayout: "V6",
		CarType:      "street",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("/vehicles/42.json", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(wantVehicle)
		suite.NoError(err)
	}))
	defer server.Close()

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

	// Act
	gotVehicle, err := fetcher.FetchVehicle(suite.T().Context(), 42)

	// Assert
	suite.Require().NoError(err)
	suite.Equal(wantVehicle, gotVehicle)
}

func (suite *FetcherTestSuite) TestFetchVehicleWithMissingID() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

	// Act
	_, err = fetcher.FetchVehicle(suite.T().Context(), 99999)

	// Assert
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, vehicles.ErrUnexpectedStatusCode)
	suite.Contains(err.Error(), "404")
}

func (suite *FetcherTestSuite) TestFetchVehicleWithServerError() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

	// Act
	_, err = fetcher.FetchVehicle(suite.T().Context(), 1)

	// Assert
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, vehicles.ErrUnexpectedStatusCode)
	suite.Contains(err.Error(), "500")
}

func (suite *FetcherTestSuite) TestFetchVehicleWithMalformedJSON() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

	// Act
	_, err = fetcher.FetchVehicle(suite.T().Context(), 1)

	// Assert
	suite.Require().Error(err)
	suite.Contains(err.Error(), "parse vehicle 1 JSON")
}

func (suite *FetcherTestSuite) TestFetchVehicleConstructsCorrectURL() {
	// Arrange
	var requestedPath string

	wantURLPath := "/vehicles/" + strconv.Itoa(2119) + ".json"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(vehicles.Vehicle{CarID: 2119})
		suite.NoError(err)
	}))
	defer server.Close()

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

	// Act
	_, err = fetcher.FetchVehicle(suite.T().Context(), 2119)

	// Assert
	suite.Require().NoError(err)
	suite.Equal(wantURLPath, requestedPath)
}

// --- FetchManifest tests ---

func (suite *FetcherTestSuite) TestFetchManifestReturnsManifestData() {
	// Arrange
	wantManifest := vehicles.Manifest{
		Vehicles: map[string]vehicles.ManifestEntry{
			"1": {LastModified: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
			"2": {LastModified: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("/vehicles/manifest.json", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(wantManifest)
		suite.NoError(err)
	}))
	defer server.Close()

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

	// Act
	got, err := fetcher.FetchManifest(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.Len(got.Vehicles, 2)
	suite.Contains(got.Vehicles, "1")
	suite.Contains(got.Vehicles, "2")
}

func (suite *FetcherTestSuite) TestFetchManifestReturnsErrorOnNon200() {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

	// Act
	got, err := fetcher.FetchManifest(suite.T().Context())

	// Assert
	suite.Require().ErrorIs(err, vehicles.ErrUnexpectedStatusCode)
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

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

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

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

	ctx, cancel := context.WithCancel(suite.T().Context())
	cancel()

	// Act
	got, err := fetcher.FetchManifest(ctx)

	// Assert
	suite.Require().ErrorIs(err, context.Canceled)
	suite.Nil(got)
}
