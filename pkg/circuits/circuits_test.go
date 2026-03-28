package circuits_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/zetetos/gt-telemetry/v2/pkg/circuits"
	"github.com/zetetos/gt-telemetry/v2/pkg/models"
)

type CircuitsTestSuite struct {
	suite.Suite
}

func TestCircuitsTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CircuitsTestSuite))
}

func (suite *CircuitsTestSuite) TestNewDBLoadsEmbeddedInventory() {
	// Arrange
	want, err := circuits.EmbeddedInventoryCount()
	suite.Require().NoError(err)

	testCases := []struct {
		name     string
		cacheDir string
	}{
		{name: "no cache directory", cacheDir: ""},
		{name: "invalid cache directory", cacheDir: "/nonexistent/path"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Act
			testDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: tc.cacheDir})
			suite.Require().NoError(err)

			// Assert
			got := testDB.GetAllCircuitIDs()
			suite.Len(got, want, "Should load all circuits from embedded inventory")
		})
	}
}

func (suite *CircuitsTestSuite) TestGetCircuitByIDWithInvalidIDReturnsNotFound() {
	// Arrange
	testDB := circuits.NewDBFromCircuits(map[string]circuits.CircuitInfo{
		"SomeCircuit": {Name: "Some Circuit"},
	})

	// Act
	_, found := testDB.GetCircuitByID("nonexistent_circuit")

	// Assert
	suite.False(found, "Should not find non-existent circuit")
}

func (suite *CircuitsTestSuite) TestGetCircuitByIDWithValidIDReturnsCircuit() {
	// Arrange
	testDB := circuits.NewDBFromCircuits(map[string]circuits.CircuitInfo{
		"TestCircuit": {
			Name:      "Test Circuit",
			Country:   "au",
			Length:    1234,
			StartLine: models.CoordinateNorm{X: 100, Y: 0, Z: 200},
		},
	})

	// Act
	got, found := testDB.GetCircuitByID("TestCircuit")

	// Assert
	suite.True(found, "Should find the circuit")
	suite.Equal("TestCircuit", got.ID)
	suite.Equal("Test Circuit", got.Name)
	suite.Equal("au", got.Country)
	suite.Equal(1234, got.Length)
	suite.Equal(models.CoordinateNorm{X: 100, Y: 0, Z: 200}, got.StartLine)
}

func (suite *CircuitsTestSuite) TestGetCircuitAtCoordinateWithInvalidCoordinateReturnsNotFound() {
	// Arrange
	testDB := circuits.NewDBFromCircuits(map[string]circuits.CircuitInfo{
		"TestCircuit": {
			Coordinates: []models.CoordinateNorm{{X: 128, Y: 8, Z: 192}},
		},
	})

	// Act
	got, found := testDB.GetCircuitAtCoordinate(models.Coordinate{X: 0, Y: 0, Z: 0}, models.CoordinateTypeCircuit)

	// Assert
	suite.False(found, "Should not find circuit at non-existent coordinate")
	suite.Empty(got)
}

func (suite *CircuitsTestSuite) TestGetCircuitAtStartLineWithValidCoordinateReturnsCircuit() {
	// Arrange
	// Coordinate {X: 250, Y: -2.4, Z: 619} normalises to {X: 240, Y: -2, Z: 608}
	testDB := circuits.NewDBFromCircuits(map[string]circuits.CircuitInfo{
		"TestCircuit": {
			StartLine: models.CoordinateNorm{X: 240, Y: -2, Z: 608},
		},
	})
	coordinate := models.Coordinate{X: 250, Y: -2.4, Z: 619}

	// Act
	got, found := testDB.GetCircuitAtCoordinate(coordinate, models.CoordinateTypeStartLine)

	// Assert
	suite.True(found, "Should find circuit at start line")
	suite.Equal("TestCircuit", got)
}

func (suite *CircuitsTestSuite) TestGetAllCircuitIDsReturnsAllIDs() {
	// Arrange
	testDB := circuits.NewDBFromCircuits(map[string]circuits.CircuitInfo{
		"CircuitA": {Name: "Circuit A"},
		"CircuitB": {Name: "Circuit B"},
		"CircuitC": {Name: "Circuit C"},
	})

	// Act
	got := testDB.GetAllCircuitIDs()

	// Assert
	suite.Len(got, 3, "Should return all injected circuit IDs")
}

func (suite *CircuitsTestSuite) TestCloseWithoutUpdaterDoesNotPanic() {
	// Arrange
	testDB := circuits.NewDBFromCircuits(map[string]circuits.CircuitInfo{})

	// Act & Assert - should not panic
	suite.NotPanics(func() {
		testDB.Close()
	})
}

func (suite *CircuitsTestSuite) TestNormaliseStartLineCoordinate() { //nolint:dupl // Intentional similarity
	// Arrange
	tests := []struct {
		name  string
		input models.Coordinate
		want  models.CoordinateNorm
	}{
		{
			name:  "coordinates divisible by normalisation factors",
			input: models.Coordinate{X: 64, Y: 8, Z: 96},
			want:  models.CoordinateNorm{X: 64, Y: 8, Z: 96},
		},
		{
			name:  "coordinates not divisible by normalisation factors",
			input: models.Coordinate{X: 50, Y: 7, Z: 70},
			want:  models.CoordinateNorm{X: 48, Y: 6, Z: 64},
		},
		{
			name:  "negative coordinates",
			input: models.Coordinate{X: -50, Y: -7, Z: -70},
			want:  models.CoordinateNorm{X: -48, Y: -6, Z: -64},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {
			// Act
			got := circuits.NormaliseStartLineCoordinate(test.input)

			// Assert
			suite.Equal(test.want, got)
		})
	}
}

func (suite *CircuitsTestSuite) TestNormaliseCircuitCoordinate() { //nolint:dupl // Intentional similarity
	// Arrange
	tests := []struct {
		name  string
		input models.Coordinate
		want  models.CoordinateNorm
	}{
		{
			name:  "coordinates divisible by normalisation factors",
			input: models.Coordinate{X: 128, Y: 12, Z: 192},
			want:  models.CoordinateNorm{X: 128, Y: 12, Z: 192},
		},
		{
			name:  "coordinates not divisible by normalisation factors",
			input: models.Coordinate{X: 100, Y: 13, Z: 150},
			want:  models.CoordinateNorm{X: 96, Y: 12, Z: 144},
		},
		{
			name:  "negative coordinates",
			input: models.Coordinate{X: -100, Y: -13, Z: -150},
			want:  models.CoordinateNorm{X: -96, Y: -12, Z: -144},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {
			// Act
			got := circuits.NormaliseCircuitCoordinate(test.input)

			// Assert
			suite.Equal(test.want, got)
		})
	}
}

func (suite *CircuitsTestSuite) TestCoordinateNormToString() {
	// Arrange
	want := "x:100,y:200,z:300"
	coordinate := models.CoordinateNorm{X: 100, Y: 200, Z: 300}

	// Act
	got := coordinate.String()

	// Assert
	suite.Equal(want, got)
}

func (suite *CircuitsTestSuite) TestNewDBWithUpdateBaseURL() {
	// Arrange & Act
	testDB, err := circuits.NewDB(circuits.CircuitDBOptions{
		UpdateBaseURL: "https://example.com/data",
	})

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(testDB)
}

func (suite *CircuitsTestSuite) TestCheckForUpdatesDoesNothingWithoutUpdateBaseURL() {
	// Arrange
	testDB := circuits.NewDBFromCircuits(map[string]circuits.CircuitInfo{})

	// Act & Assert — should not panic
	suite.NotPanics(func() {
		testDB.CheckForUpdates(context.Background())
	})
}

func (suite *CircuitsTestSuite) TestCheckForUpdatesDownloadsUpdatesFromRemote() {
	// Arrange
	cachedCircuit := circuits.CircuitInfo{
		ID:           "TestTrack",
		Name:         "Test Track Remote",
		Country:      "de",
		LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	var requestedPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestedPaths = append(requestedPaths, request.URL.Path)

		response.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/circuits/manifest.json":
			err := json.NewEncoder(response).Encode(circuits.Manifest{
				Circuits: map[string]circuits.ManifestEntry{
					"TestTrack": {LastModified: cachedCircuit.LastModified},
				},
			})
			suite.NoError(err)
		case "/circuits/TestTrack.json":
			err := json.NewEncoder(response).Encode(cachedCircuit)
			suite.NoError(err)
		default:
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	testDB, err := circuits.NewDB(circuits.CircuitDBOptions{
		CacheDir:      suite.T().TempDir(),
		UpdateBaseURL: server.URL,
	})
	suite.Require().NoError(err)

	// Act
	testDB.CheckForUpdates(context.Background())
	defer testDB.Close()

	// Assert — eventually the inventory is updated
	suite.Eventually(func() bool {
		_, found := testDB.GetCircuitByID("TestTrack")

		return found
	}, 2*time.Second, 10*time.Millisecond, "circuit should be downloaded and available")

	got, found := testDB.GetCircuitByID("TestTrack")
	suite.True(found)
	suite.Equal("Test Track Remote", got.Name)
}

func (suite *CircuitsTestSuite) TestCloseStopsBackgroundUpdater() {
	// Arrange — start a slow server so the goroutine is definitely running
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testDB, err := circuits.NewDB(circuits.CircuitDBOptions{
		UpdateBaseURL: server.URL,
	})
	suite.Require().NoError(err)

	testDB.CheckForUpdates(context.Background())

	// Act & Assert — Close should not block or panic
	suite.NotPanics(func() {
		testDB.Close()
	})
}

func (suite *CircuitsTestSuite) TestGetCircuitAtCoordinateWithNilInventoryReturnsNotFound() {
	// Arrange — use a zero-value DB (nil inventory)
	testDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: "/nonexistent"})
	suite.Require().NoError(err)

	// Act — look up a coordinate that obviously doesn't exist
	_, found := testDB.GetCircuitAtCoordinate(models.Coordinate{X: 0, Y: 0, Z: 0}, models.CoordinateTypeCircuit)

	// Assert
	suite.False(found)
}

func (suite *CircuitsTestSuite) TestGetCircuitAtStartLineAmbiguousReturnsNotFound() {
	// Arrange — two circuits sharing the same start line
	testDB := circuits.NewDBFromCircuits(map[string]circuits.CircuitInfo{
		"CircuitA": {StartLine: models.CoordinateNorm{X: 240, Y: 0, Z: 608}},
		"CircuitB": {StartLine: models.CoordinateNorm{X: 240, Y: 0, Z: 608}},
	})

	// Act
	_, found := testDB.GetCircuitAtCoordinate(
		models.Coordinate{X: 240, Y: 0, Z: 608},
		models.CoordinateTypeStartLine,
	)

	// Assert — ambiguous start line returns not found
	suite.False(found)
}

func (suite *CircuitsTestSuite) TestLoadCacheFileSkipsMalformedJSON() {
	// Arrange
	tmpDir := suite.T().TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte(`{not json`), 0o600)
	suite.Require().NoError(err)

	// Act & Assert — should not panic, bad file silently skipped
	suite.NotPanics(func() {
		_, err = circuits.NewDB(circuits.CircuitDBOptions{CacheDir: tmpDir})
		suite.Require().NoError(err)
	})
}

func (suite *CircuitsTestSuite) TestLoadCacheFileSkipsEntryWithEmptyID() {
	// Arrange — circuit with empty ID
	tmpDir := suite.T().TempDir()

	data, err := json.Marshal(circuits.CircuitInfo{Name: "No ID Circuit"})
	suite.Require().NoError(err)

	err = os.WriteFile(filepath.Join(tmpDir, "noid.json"), data, 0o600)
	suite.Require().NoError(err)

	// Act
	testDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: tmpDir})
	suite.Require().NoError(err)

	// Assert — embedded circuits loaded, empty-ID entry not added
	want, err := circuits.EmbeddedInventoryCount()
	suite.Require().NoError(err)
	suite.Len(testDB.GetAllCircuitIDs(), want)
}

func (suite *CircuitsTestSuite) TestLoadCacheFileDoesNotDowngradeNewerEntry() {
	// Arrange — cache has an older version of an existing circuit
	tmpDir := suite.T().TempDir()
	olderCircuit := circuits.CircuitInfo{
		ID:           "HighSpeedRing",
		Name:         "Old Name",
		LastModified: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	err := circuits.WriteCircuitCache(tmpDir, "HighSpeedRing", &olderCircuit)
	suite.Require().NoError(err)

	// Act — embedded inventory has HighSpeedRing; cache has an older version
	testDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: tmpDir})
	suite.Require().NoError(err)

	got, found := testDB.GetCircuitByID("HighSpeedRing")

	// Assert — embedded (newer) version wins
	suite.True(found)
	suite.NotEqual("Old Name", got.Name)
}
