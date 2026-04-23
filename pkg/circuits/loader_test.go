package circuits_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/zetetos/gt-telemetry/v2/pkg/circuits"
	"github.com/zetetos/gt-telemetry/v2/pkg/models"
)

var (
	errManifestUnavailable = errors.New("manifest unavailable")
	errShouldNotBeCalled   = errors.New("should not be called")
	errFetchFailed         = errors.New("fetch failed")
	errSimulatedReadError  = errors.New("simulated read error")
)

type LoaderTestSuite struct {
	suite.Suite
}

func TestLoaderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LoaderTestSuite))
}

func newTestCircuit(id, name, country string, startLine models.CoordinateNorm, coords []models.CoordinateNorm) circuits.CircuitInfo {
	return circuits.CircuitInfo{
		ID:           id,
		Name:         name,
		Variation:    name,
		Country:      country,
		Default:      true,
		Length:       1000,
		StartLine:    startLine,
		LastModified: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Coordinates:  coords,
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	return data
}

// --- loadFromFS tests ---

func (suite *LoaderTestSuite) TestLoadFromFSLoadsCircuitFiles() {
	// Arrange
	circuit := newTestCircuit("TestTrack", "Test Track", "jp",
		models.CoordinateNorm{X: 100, Y: 0, Z: 200},
		[]models.CoordinateNorm{{X: 100, Y: 0, Z: 200}, {X: 116, Y: 0, Z: 216}},
	)

	fsys := fstest.MapFS{
		"circuits/TestTrack.json": &fstest.MapFile{Data: mustMarshal(suite.T(), circuit)},
	}

	// Act
	got, err := circuits.LoadFromFS(fsys, "circuits")

	// Assert
	suite.Require().NoError(err)
	suite.Len(got, 1)
	suite.Equal("TestTrack", got["TestTrack"].ID)
	suite.Equal("Test Track", got["TestTrack"].Name)
}

func (suite *LoaderTestSuite) TestLoadFromFSSkipsManifestFile() {
	// Arrange
	circuit := newTestCircuit("TestTrack", "Test Track", "jp",
		models.CoordinateNorm{X: 100, Y: 0, Z: 200},
		nil,
	)

	fsys := fstest.MapFS{
		"circuits/TestTrack.json": &fstest.MapFile{Data: mustMarshal(suite.T(), circuit)},
		"circuits/manifest.json":  &fstest.MapFile{Data: []byte(`{"circuits":{}}`)},
	}

	// Act
	got, err := circuits.LoadFromFS(fsys, "circuits")

	// Assert
	suite.Require().NoError(err)
	suite.Len(got, 1)
}

func (suite *LoaderTestSuite) TestLoadFromFSSkipsDirectories() {
	// Arrange
	circuit := newTestCircuit("TestTrack", "Test Track", "jp",
		models.CoordinateNorm{X: 100, Y: 0, Z: 200},
		nil,
	)

	fsys := fstest.MapFS{
		"circuits/TestTrack.json": &fstest.MapFile{Data: mustMarshal(suite.T(), circuit)},
		"circuits/subdir/a.json":  &fstest.MapFile{Data: []byte(`{}`)},
	}

	// Act
	got, err := circuits.LoadFromFS(fsys, "circuits")

	// Assert
	suite.Require().NoError(err)
	suite.Len(got, 1)
}

func (suite *LoaderTestSuite) TestLoadFromFSSkipsNonJSONFiles() {
	// Arrange
	circuit := newTestCircuit("TestTrack", "Test Track", "jp",
		models.CoordinateNorm{X: 100, Y: 0, Z: 200},
		nil,
	)

	fsys := fstest.MapFS{
		"circuits/TestTrack.json": &fstest.MapFile{Data: mustMarshal(suite.T(), circuit)},
		"circuits/.gitkeep":       &fstest.MapFile{Data: []byte{}},
		"circuits/readme.txt":     &fstest.MapFile{Data: []byte("hello")},
	}

	// Act
	got, err := circuits.LoadFromFS(fsys, "circuits")

	// Assert
	suite.Require().NoError(err)
	suite.Len(got, 1)
}

func (suite *LoaderTestSuite) TestLoadFromFSReturnsErrorForMissingDirectory() {
	// Arrange
	fsys := fstest.MapFS{}

	// Act
	got, err := circuits.LoadFromFS(fsys, "nonexistent")

	// Assert
	suite.Require().Error(err)
	suite.Nil(got)
}

func (suite *LoaderTestSuite) TestLoadFromFSReturnsErrorForInvalidJSON() {
	// Arrange
	fsys := fstest.MapFS{
		"circuits/bad.json": &fstest.MapFile{Data: []byte(`{invalid}`)},
	}

	// Act
	got, err := circuits.LoadFromFS(fsys, "circuits")

	// Assert
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "parsing circuit file")
	suite.Nil(got)
}

func (suite *LoaderTestSuite) TestLoadFromFSFallsBackToFilenameForEmptyID() {
	// Arrange
	circuit := newTestCircuit("", "Test Track", "jp",
		models.CoordinateNorm{X: 100, Y: 0, Z: 200},
		nil,
	)

	fsys := fstest.MapFS{
		"circuits/MyCircuit.json": &fstest.MapFile{Data: mustMarshal(suite.T(), circuit)},
	}

	// Act
	got, err := circuits.LoadFromFS(fsys, "circuits")

	// Assert
	suite.Require().NoError(err)
	suite.Len(got, 1)
	suite.Contains(got, "MyCircuit")
	suite.Equal("MyCircuit", got["MyCircuit"].ID)
}

// --- buildLookupMaps tests ---

func (suite *LoaderTestSuite) TestBuildLookupMapsCreatesUniqueCoordinateEntries() {
	// Arrange
	circuitMap := map[string]circuits.CircuitInfo{
		"TrackA": newTestCircuit("TrackA", "Track A", "jp",
			models.CoordinateNorm{X: 100, Y: 0, Z: 200},
			[]models.CoordinateNorm{{X: 10, Y: 0, Z: 20}, {X: 30, Y: 0, Z: 40}},
		),
		"TrackB": newTestCircuit("TrackB", "Track B", "uk",
			models.CoordinateNorm{X: 500, Y: 0, Z: 600},
			[]models.CoordinateNorm{{X: 50, Y: 0, Z: 60}, {X: 70, Y: 0, Z: 80}},
		),
	}

	// Act
	got := circuits.BuildLookupMapsForTest(circuitMap)

	// Assert
	suite.Equal("TrackA", got.Coordinates["x:10,y:0,z:20"])
	suite.Equal("TrackA", got.Coordinates["x:30,y:0,z:40"])
	suite.Equal("TrackB", got.Coordinates["x:50,y:0,z:60"])
	suite.Equal("TrackB", got.Coordinates["x:70,y:0,z:80"])
}

func (suite *LoaderTestSuite) TestBuildLookupMapsExcludesSharedCoordinates() {
	// Arrange
	sharedCoord := models.CoordinateNorm{X: 10, Y: 0, Z: 20}

	circuitMap := map[string]circuits.CircuitInfo{
		"TrackA": newTestCircuit("TrackA", "Track A", "jp",
			models.CoordinateNorm{X: 100, Y: 0, Z: 200},
			[]models.CoordinateNorm{sharedCoord, {X: 30, Y: 0, Z: 40}},
		),
		"TrackB": newTestCircuit("TrackB", "Track B", "uk",
			models.CoordinateNorm{X: 500, Y: 0, Z: 600},
			[]models.CoordinateNorm{sharedCoord, {X: 70, Y: 0, Z: 80}},
		),
	}

	// Act
	got := circuits.BuildLookupMapsForTest(circuitMap)

	// Assert
	_, found := got.Coordinates[sharedCoord.String()]
	suite.False(found, "Shared coordinate should be excluded from lookup")
	suite.Equal("TrackA", got.Coordinates["x:30,y:0,z:40"])
	suite.Equal("TrackB", got.Coordinates["x:70,y:0,z:80"])
}

func (suite *LoaderTestSuite) TestBuildLookupMapsBuildsStartLineLookup() {
	// Arrange
	circuitMap := map[string]circuits.CircuitInfo{
		"TrackA": newTestCircuit("TrackA", "Track A", "jp",
			models.CoordinateNorm{X: 100, Y: 0, Z: 200},
			nil,
		),
		"TrackB": newTestCircuit("TrackB", "Track B", "uk",
			models.CoordinateNorm{X: 500, Y: 0, Z: 600},
			nil,
		),
	}

	// Act
	got := circuits.BuildLookupMapsForTest(circuitMap)

	// Assert
	suite.Equal([]string{"TrackA"}, got.StartLines["x:100,y:0,z:200"])
	suite.Equal([]string{"TrackB"}, got.StartLines["x:500,y:0,z:600"])
}

func (suite *LoaderTestSuite) TestBuildLookupMapsGroupsSharedStartLines() {
	// Arrange
	sharedStart := models.CoordinateNorm{X: 100, Y: 0, Z: 200}

	circuitMap := map[string]circuits.CircuitInfo{
		"TrackA": newTestCircuit("TrackA", "Track A", "jp", sharedStart, nil),
		"TrackB": newTestCircuit("TrackB", "Track B", "uk", sharedStart, nil),
	}

	// Act
	got := circuits.BuildLookupMapsForTest(circuitMap)

	// Assert
	ids := got.StartLines[sharedStart.String()]
	suite.Len(ids, 2)
	suite.Contains(ids, "TrackA")
	suite.Contains(ids, "TrackB")
}

func (suite *LoaderTestSuite) TestBuildLookupMapsNilsOutCoordinatesAfterBuilding() {
	// Arrange
	circuitMap := map[string]circuits.CircuitInfo{
		"TrackA": newTestCircuit("TrackA", "Track A", "jp",
			models.CoordinateNorm{X: 100, Y: 0, Z: 200},
			[]models.CoordinateNorm{{X: 10, Y: 0, Z: 20}},
		),
	}

	// Act
	got := circuits.BuildLookupMapsForTest(circuitMap)

	// Assert
	suite.Nil(got.Circuits["TrackA"].Coordinates, "Coordinates should be nil after building lookup maps")
}

// --- writeCircuitCache / loadCacheDir round-trip tests ---

func (suite *LoaderTestSuite) TestWriteCircuitCacheAndLoadCacheDir() {
	// Arrange
	cacheDir := suite.T().TempDir()
	circuit := newTestCircuit("TestTrack", "Test Track", "jp",
		models.CoordinateNorm{X: 100, Y: 0, Z: 200},
		[]models.CoordinateNorm{{X: 10, Y: 0, Z: 20}},
	)

	// Act
	err := circuits.WriteCircuitCache(cacheDir, "TestTrack", &circuit)
	suite.Require().NoError(err)

	got, err := circuits.LoadCacheDir(cacheDir)

	// Assert
	suite.Require().NoError(err)
	suite.Len(got, 1)
	suite.Equal("TestTrack", got["TestTrack"].ID)
	suite.Equal("Test Track", got["TestTrack"].Name)
	suite.Equal("jp", got["TestTrack"].Country)
}

func (suite *LoaderTestSuite) TestWriteCircuitCacheCreatesDirectory() {
	// Arrange
	cacheDir := filepath.Join(suite.T().TempDir(), "nested", "dir")
	circuit := newTestCircuit("TestTrack", "Test Track", "jp",
		models.CoordinateNorm{X: 100, Y: 0, Z: 200},
		nil,
	)

	// Act
	err := circuits.WriteCircuitCache(cacheDir, "TestTrack", &circuit)

	// Assert
	suite.Require().NoError(err)

	_, statErr := os.Stat(filepath.Join(cacheDir, "TestTrack.json"))
	suite.NoError(statErr, "Cache file should exist")
}

func (suite *LoaderTestSuite) TestLoadCacheDirReturnsErrorForMissingDirectory() {
	// Arrange & Act
	result, err := circuits.LoadCacheDir("/nonexistent/path/that/does/not/exist")

	// Assert
	suite.Require().Error(err)
	suite.Nil(result)
}

// --- loadFromFS ReadFile error test ---

func (suite *LoaderTestSuite) TestLoadFromFSReturnsErrorWhenFileCannotBeRead() {
	// Arrange — wrap a MapFS with a ReadFile that always errors
	base := fstest.MapFS{
		"circuits/TestTrack.json": &fstest.MapFile{Data: []byte(`{"id":"TestTrack"}`)},
	}
	fsys := errReadFileFS{base}

	// Act
	got, err := circuits.LoadFromFS(fsys, "circuits")

	// Assert
	suite.Require().Error(err)
	suite.Contains(err.Error(), "reading circuit file")
	suite.Nil(got)
}

// --- writeCircuitCache error tests ---

func (suite *LoaderTestSuite) TestWriteCircuitCacheReturnsErrorWhenDirectoryCannotBeCreated() {
	// Arrange — place a file at a path, then try to use it as a directory
	parent := suite.T().TempDir()
	blockingFile := filepath.Join(parent, "not-a-dir")

	err := os.WriteFile(blockingFile, []byte("block"), 0o600)
	suite.Require().NoError(err)

	cacheDir := filepath.Join(blockingFile, "subdir")
	circuit := newTestCircuit("TestTrack", "Test Track", "jp", models.CoordinateNorm{}, nil)

	// Act
	err = circuits.WriteCircuitCache(cacheDir, "TestTrack", &circuit)

	// Assert
	suite.Require().Error(err)
	suite.Contains(err.Error(), "creating cache directory")
}

func (suite *LoaderTestSuite) TestWriteCircuitCacheReturnsErrorWhenFileCannotBeWritten() {
	if os.Getuid() == 0 {
		suite.T().Skip("skipping permission test when running as root")
	}

	// Arrange — make the cache directory read-only
	cacheDir := suite.T().TempDir()
	suite.Require().NoError(os.Chmod(cacheDir, 0o555))

	defer os.Chmod(cacheDir, 0o755) //nolint:errcheck // cleanup only

	circuit := newTestCircuit("TestTrack", "Test Track", "jp", models.CoordinateNorm{}, nil)

	// Act
	err := circuits.WriteCircuitCache(cacheDir, "TestTrack", &circuit)

	// Assert
	suite.Require().Error(err)
	suite.Contains(err.Error(), "writing cache file")
}

// --- downloadUpdatedCircuits tests ---

func (suite *LoaderTestSuite) TestDownloadUpdatedCircuitsReturnsErrorWhenManifestFetchFails() {
	// Arrange
	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{})
	suite.Require().NoError(err)

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return nil, errManifestUnavailable
		},
	})

	// Act
	didUpdate, err := circuitDB.DownloadUpdatedCircuits(suite.T().Context())

	// Assert
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, errManifestUnavailable)
	suite.False(didUpdate)
}

func (suite *LoaderTestSuite) TestDownloadUpdatedCircuitsReturnsFalseForEmptyManifest() {
	// Arrange
	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{})
	suite.Require().NoError(err)

	var circuitFetched bool

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return &circuits.Manifest{Circuits: map[string]circuits.ManifestEntry{}}, nil
		},
		onFetchCircuit: func(_ context.Context, _ string) (*circuits.CircuitInfo, error) {
			circuitFetched = true

			return nil, errShouldNotBeCalled
		},
	})

	// Act
	didUpdate, err := circuitDB.DownloadUpdatedCircuits(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.False(didUpdate)
	suite.False(circuitFetched, "FetchCircuit should not be called for empty manifest")
}

func (suite *LoaderTestSuite) TestDownloadUpdatedCircuitsReturnsFalseWhenCircuitIsUpToDate() {
	// Arrange — write a circuit with a known date to the cache dir
	cacheDir := suite.T().TempDir()
	circuit := newTestCircuit("TestTrack", "Test Track", "jp", models.CoordinateNorm{}, nil)

	err := circuits.WriteCircuitCache(cacheDir, "TestTrack", &circuit)
	suite.Require().NoError(err)

	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: cacheDir})
	suite.Require().NoError(err)

	var circuitFetched bool

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return &circuits.Manifest{
				Circuits: map[string]circuits.ManifestEntry{
					// Same date as the circuit in the cache — not newer
					"TestTrack": {LastModified: circuit.LastModified},
				},
			}, nil
		},
		onFetchCircuit: func(_ context.Context, _ string) (*circuits.CircuitInfo, error) {
			circuitFetched = true

			return nil, errShouldNotBeCalled
		},
	})

	// Act
	didUpdate, err := circuitDB.DownloadUpdatedCircuits(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.False(didUpdate)
	suite.False(circuitFetched, "FetchCircuit should not be called when circuit is up to date")
}

func (suite *LoaderTestSuite) TestDownloadUpdatedCircuitsReturnsTrueWhenCircuitIsNewer() {
	// Arrange — write an old circuit to the cache dir
	cacheDir := suite.T().TempDir()
	oldCircuit := newTestCircuit("TestTrack", "Test Track Old", "jp", models.CoordinateNorm{}, nil)

	err := circuits.WriteCircuitCache(cacheDir, "TestTrack", &oldCircuit)
	suite.Require().NoError(err)

	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: cacheDir})
	suite.Require().NoError(err)

	newerDate := oldCircuit.LastModified.Add(24 * time.Hour)
	newCircuit := circuits.CircuitInfo{ID: "TestTrack", Name: "Test Track New"}

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return &circuits.Manifest{
				Circuits: map[string]circuits.ManifestEntry{
					"TestTrack": {LastModified: newerDate},
				},
			}, nil
		},
		onFetchCircuit: func(_ context.Context, _ string) (*circuits.CircuitInfo, error) {
			return &newCircuit, nil
		},
	})

	// Act
	didUpdate, err := circuitDB.DownloadUpdatedCircuits(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.True(didUpdate)
}

func (suite *LoaderTestSuite) TestDownloadUpdatedCircuitsReturnsTrueForUnknownCircuit() {
	// Arrange — DB with no "NewCircuit" circuit
	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{})
	suite.Require().NoError(err)

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return &circuits.Manifest{
				Circuits: map[string]circuits.ManifestEntry{
					"NewCircuit": {LastModified: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
		onFetchCircuit: func(_ context.Context, circuitID string) (*circuits.CircuitInfo, error) {
			return &circuits.CircuitInfo{ID: circuitID, Name: "New Circuit"}, nil
		},
	})

	// Act
	didUpdate, err := circuitDB.DownloadUpdatedCircuits(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.True(didUpdate)
}

func (suite *LoaderTestSuite) TestDownloadUpdatedCircuitsSkipsCircuitOnFetchError() {
	// Arrange
	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{})
	suite.Require().NoError(err)

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return &circuits.Manifest{
				Circuits: map[string]circuits.ManifestEntry{
					"BadCircuit": {LastModified: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
		onFetchCircuit: func(_ context.Context, _ string) (*circuits.CircuitInfo, error) {
			return nil, errFetchFailed
		},
	})

	// Act
	didUpdate, err := circuitDB.DownloadUpdatedCircuits(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.False(didUpdate, "update should be skipped when FetchCircuit fails")
}

func (suite *LoaderTestSuite) TestDownloadUpdatedCircuitsSkipsUpdateWhenCacheWriteFails() {
	if os.Getuid() == 0 {
		suite.T().Skip("skipping permission test when running as root")
	}

	// Arrange — read-only cache dir so WriteCircuitCache will fail
	cacheDir := suite.T().TempDir()
	suite.Require().NoError(os.Chmod(cacheDir, 0o555))

	defer os.Chmod(cacheDir, 0o755) //nolint:errcheck // cleanup only

	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: cacheDir})
	suite.Require().NoError(err)

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return &circuits.Manifest{
				Circuits: map[string]circuits.ManifestEntry{
					"NewCircuit": {LastModified: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
		onFetchCircuit: func(_ context.Context, circuitID string) (*circuits.CircuitInfo, error) {
			return &circuits.CircuitInfo{ID: circuitID}, nil
		},
	})

	// Act
	didUpdate, err := circuitDB.DownloadUpdatedCircuits(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.False(didUpdate, "update should be skipped when cache write fails")
}

func (suite *LoaderTestSuite) TestDownloadUpdatedCircuitsDoesNotWriteCacheWhenNoCacheDir() {
	// Arrange — no cacheDir means didUpdate=true is set without a write
	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{})
	suite.Require().NoError(err)

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return &circuits.Manifest{
				Circuits: map[string]circuits.ManifestEntry{
					"NewCircuit": {LastModified: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
		onFetchCircuit: func(_ context.Context, circuitID string) (*circuits.CircuitInfo, error) {
			return &circuits.CircuitInfo{ID: circuitID}, nil
		},
	})

	// Act
	didUpdate, err := circuitDB.DownloadUpdatedCircuits(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.True(didUpdate)
}

// --- fetchUpdates tests ---

func (suite *LoaderTestSuite) TestFetchUpdatesReturnsErrorWhenManifestFetchFails() {
	// Arrange
	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{})
	suite.Require().NoError(err)

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return nil, errManifestUnavailable
		},
	})

	// Act
	err = circuitDB.FetchUpdates(suite.T().Context())

	// Assert
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, errManifestUnavailable)
}

func (suite *LoaderTestSuite) TestFetchUpdatesReturnsNilWhenNoUpdates() {
	// Arrange — empty manifest means no updates, no rebuild
	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{})
	suite.Require().NoError(err)

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return &circuits.Manifest{Circuits: map[string]circuits.ManifestEntry{}}, nil
		},
	})

	// Act
	err = circuitDB.FetchUpdates(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
}

func (suite *LoaderTestSuite) TestFetchUpdatesRebuildsInventoryWhenUpdated() {
	// Arrange — write old circuit to cache dir
	cacheDir := suite.T().TempDir()
	oldCircuit := newTestCircuit("TestTrack", "Test Track Old", "jp", models.CoordinateNorm{}, nil)

	err := circuits.WriteCircuitCache(cacheDir, "TestTrack", &oldCircuit)
	suite.Require().NoError(err)

	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: cacheDir})
	suite.Require().NoError(err)

	newerDate := oldCircuit.LastModified.Add(24 * time.Hour)
	updatedCircuit := circuits.CircuitInfo{ID: "TestTrack", Name: "Test Track New", LastModified: newerDate}

	circuitDB.SetFetcher(&mockFetcher{
		onFetchManifest: func(_ context.Context) (*circuits.Manifest, error) {
			return &circuits.Manifest{
				Circuits: map[string]circuits.ManifestEntry{
					"TestTrack": {LastModified: newerDate},
				},
			}, nil
		},
		onFetchCircuit: func(_ context.Context, _ string) (*circuits.CircuitInfo, error) {
			return &updatedCircuit, nil
		},
	})

	// Act
	err = circuitDB.FetchUpdates(suite.T().Context())

	// Assert
	suite.Require().NoError(err)

	got, found := circuitDB.GetCircuitByID("TestTrack")
	suite.Require().True(found)
	suite.Equal("Test Track New", got.Name)
}

// --- rebuildInventory tests ---

func (suite *LoaderTestSuite) TestRebuildInventoryLoadsEmbeddedCircuits() {
	// Arrange
	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{})
	suite.Require().NoError(err)

	// Act
	err = circuitDB.RebuildInventory()

	// Assert — embedded inventory is not empty
	suite.Require().NoError(err)

	_, found := circuitDB.GetCircuitByID("HighSpeedRing")
	suite.True(found, "embedded circuit should be present after rebuild")
}

func (suite *LoaderTestSuite) TestRebuildInventoryMergesCachedCircuitsOverEmbedded() {
	// Arrange — write a custom circuit to cacheDir
	cacheDir := suite.T().TempDir()
	cached := newTestCircuit("MyCustomTrack", "My Custom Track", "us", models.CoordinateNorm{}, nil)

	err := circuits.WriteCircuitCache(cacheDir, "MyCustomTrack", &cached)
	suite.Require().NoError(err)

	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: cacheDir})
	suite.Require().NoError(err)

	// Act
	err = circuitDB.RebuildInventory()

	// Assert
	suite.Require().NoError(err)

	got, found := circuitDB.GetCircuitByID("MyCustomTrack")
	suite.Require().True(found)
	suite.Equal("My Custom Track", got.Name)
}

func (suite *LoaderTestSuite) TestRebuildInventoryIgnoresMissingCacheDir() {
	// Arrange — cacheDir does not exist; rebuild should still succeed
	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{CacheDir: "/nonexistent/cache/path"})
	suite.Require().NoError(err)

	// Act
	err = circuitDB.RebuildInventory()

	// Assert — embedded circuits still loaded, no error
	suite.Require().NoError(err)

	_, found := circuitDB.GetCircuitByID("HighSpeedRing")
	suite.True(found, "embedded circuit should still be present when cache dir is missing")
}

// --- Test helpers ---

// errReadFileFS wraps a MapFS and makes ReadFile always fail, while preserving directory listing.
type errReadFileFS struct {
	fstest.MapFS
}

func (f errReadFileFS) ReadFile(_ string) ([]byte, error) {
	return nil, errSimulatedReadError
}

// Ensure errReadFileFS implements fs.ReadFileFS.
var _ fs.ReadFileFS = errReadFileFS{}

// mockFetcher is a test double for circuits.Fetcher.
type mockFetcher struct {
	onFetchManifest func(ctx context.Context) (*circuits.Manifest, error)
	onFetchCircuit  func(ctx context.Context, circuitID string) (*circuits.CircuitInfo, error)
}

func (m *mockFetcher) FetchManifest(ctx context.Context) (*circuits.Manifest, error) {
	if m.onFetchManifest != nil {
		return m.onFetchManifest(ctx)
	}

	return &circuits.Manifest{}, nil
}

func (m *mockFetcher) FetchCircuit(ctx context.Context, circuitID string) (*circuits.CircuitInfo, error) {
	if m.onFetchCircuit != nil {
		return m.onFetchCircuit(ctx, circuitID)
	}

	return &circuits.CircuitInfo{ID: circuitID}, nil
}
