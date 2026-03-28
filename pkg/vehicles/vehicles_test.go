package vehicles_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/zetetos/gt-telemetry/v2/pkg/vehicles"
)

type VehiclesTestSuite struct {
	suite.Suite
}

func TestVehiclesTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VehiclesTestSuite))
}

func (suite *VehiclesTestSuite) TestEmptyJSONParameterFallsBackToBaseInventory() {
	// Arrange
	inventoryJSON := []byte{}

	wantCount, err := vehicles.EmbeddedInventoryCount()
	suite.Require().NoError(err)

	// Act
	db, err := VehicleDBWithFetcherDisabled(inventoryJSON)

	// Assert
	suite.Require().NoError(err)
	suite.Equal(wantCount, db.Len(), "Should have loaded all embedded inventory entries as fallback")
}

func (suite *VehiclesTestSuite) TestValidJSONParameterCanConstructInventory() {
	// Arrange
	inventoryJSON := []byte(`{
		"0": {
			"Model": "",
			"Manufacturer": "",
			"Category": "",
			"Drivetrain": "",
			"Aspiration": "",
			"EngineLayout": "",
			"EngineAngle": "",
			"Year": 0,
			"CarID": 0,
			"OpenCockpit": false,
			"CarType": "",
			"Length": 0,
			"Width": 0,
			"Height": 0,
			"Wheelbase": 0,
			"TrackFront": 0,
			"TrackRear": 0
		}
	}`)

	// Act
	db, err := VehicleDBWithFetcherDisabled(inventoryJSON)

	// Assert
	suite.Require().NoError(err)
	suite.Equal(1, db.Len())
}

func (suite *VehiclesTestSuite) TestInvalidJSONParameterReturnsError() {
	// Arrange
	inventoryJSON := []byte(`{
		not_an_int: {}
	}`)

	// Act
	_, err := VehicleDBWithFetcherDisabled(inventoryJSON)

	// Assert
	suite.ErrorContains(err, "unmarshall vehicle inventory JSON")
}

func (suite *VehiclesTestSuite) TestGetVehicleIDWithInvalidIDReturnsError() {
	// Arrange
	inventoryJSON := []byte(`{}`)

	// Act
	inventory, err := VehicleDBWithFetcherDisabled(inventoryJSON)
	suite.Require().NoError(err)

	_, err = inventory.GetVehicleByID(1)

	// Assert
	suite.ErrorContains(err, "no vehicle found with id")
}

func (suite *VehiclesTestSuite) TestGetVehicleWIthValidIDReturnsVehicle() {
	// Arrange
	wantValue := vehicles.Vehicle{
		CarID:                 12345,
		Model:                 "Dummy Model",
		Manufacturer:          "Dummy Manufacturer",
		Category:              "",
		Drivetrain:            "-",
		Aspiration:            "-",
		EngineLayout:          "-",
		EngineBankAngle:       0,
		EngineCrankPlaneAngle: 0,
		Year:                  0,
		OpenCockpit:           false,
		CarType:               "street",
		Length:                4500,
		Width:                 1800,
		Height:                1300,
		Wheelbase:             2700,
		TrackFront:            1550,
		TrackRear:             1600,
	}

	inventoryJSON := []byte(`{
		"12345": {
			"Model": "Dummy Model",
			"Manufacturer": "Dummy Manufacturer",
			"Category": "",
			"Drivetrain": "-",
			"Aspiration": "-",
			"EngineLayout": "-",
			"EngineAngle": "-",
			"Year": 0,
			"CarID": 12345,
			"OpenCockpit": false,
			"CarType": "street",
			"Length": 4500,
			"Width": 1800,
			"Height": 1300,
			"Wheelbase": 2700,
			"TrackFront": 1550,
			"TrackRear": 1600
		}
	}`)

	// Act
	inventory, err := VehicleDBWithFetcherDisabled(inventoryJSON)
	suite.Require().NoError(err)

	gotValue, err := inventory.GetVehicleByID(wantValue.CarID)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *VehiclesTestSuite) TestShortAspirationExpandsToLongName() {
	tests := map[string]string{
		"NA":      "Naturally Aspirated",
		"TC":      "Turbocharged",
		"SC":      "Supercharged",
		"TC+SC":   "Compound Charged",
		"EV":      "Electric Vehicle",
		"-":       "-",
		"INVALID": "INVALID",
	}

	for shortName, wantValue := range tests {
		suite.Run(shortName, func() {
			// Arrange
			vehicle := vehicles.Vehicle{
				Aspiration: shortName,
			}

			// Act
			gotValue := vehicle.ExpandedAspiration()

			// Assert
			suite.Equal(wantValue, gotValue)
		})
	}
}

// --- Mock Fetcher ---

type mockFetcher struct {
	mu              sync.Mutex
	calls           int
	vehicle         vehicles.Vehicle
	err             error
	fetchDelay      time.Duration
	onFetchManifest func(ctx context.Context) (*vehicles.Manifest, error)
}

func (m *mockFetcher) FetchManifest(ctx context.Context) (*vehicles.Manifest, error) {
	if m.onFetchManifest != nil {
		return m.onFetchManifest(ctx)
	}

	return &vehicles.Manifest{}, nil
}

func (m *mockFetcher) FetchVehicle(_ context.Context, _ int) (vehicles.Vehicle, error) {
	if m.fetchDelay > 0 {
		time.Sleep(m.fetchDelay)
	}

	m.mu.Lock()
	m.calls++
	m.mu.Unlock()

	return m.vehicle, m.err
}

func (m *mockFetcher) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.calls
}

// --- Remote Fetch Tests ---

func (suite *VehiclesTestSuite) TestGetVehicleByIDTriggersRemoteFetch() {
	// Arrange
	vehicleID := 12345
	wantVehicle := vehicles.Vehicle{
		CarID:        vehicleID,
		Manufacturer: "TestCo",
		Model:        "TestCar",
		Year:         2025,
		Drivetrain:   "4WD",
		Aspiration:   "TC",
		EngineLayout: "V6",
		CarType:      "race",
	}

	fetcher := &mockFetcher{vehicle: wantVehicle}

	inventoryJSON := []byte(`{}`)
	DBOptions := vehicles.DBOptions{
		Fetcher: fetcher,
	}

	vehicleDB, err := vehicles.NewDB(inventoryJSON, DBOptions)
	suite.Require().NoError(err)

	// Act — first call returns a stub
	got, err := vehicleDB.GetVehicleByID(vehicleID)
	suite.Require().NoError(err)
	suite.Equal(vehicleID, got.CarID)
	suite.Empty(got.Manufacturer) // stub

	// Wait for background fetch to complete
	suite.Eventually(func() bool {
		v, err := vehicleDB.GetVehicleByID(vehicleID)

		return err == nil && v.Manufacturer == "TestCo"
	}, 2*time.Second, 10*time.Millisecond)

	// Assert
	got, err = vehicleDB.GetVehicleByID(vehicleID)
	suite.Require().NoError(err)
	suite.Equal(wantVehicle.CarID, got.CarID)
	suite.Equal(wantVehicle.Manufacturer, got.Manufacturer)
	suite.Equal(wantVehicle.Model, got.Model)
	suite.False(got.LastModified.IsZero(), "LastModified should be set after remote fetch")
	suite.GreaterOrEqual(fetcher.callCount(), 1)
}

func (suite *VehiclesTestSuite) TestGetVehicleByIDDeduplicatesConcurrentFetches() {
	// Arrange
	vehicleID := 12345
	wantVehicle := vehicles.Vehicle{
		CarID:        vehicleID,
		Manufacturer: "DedupCo",
		Model:        "DedupCar",
	}

	fetcher := &mockFetcher{
		vehicle:    wantVehicle,
		fetchDelay: 100 * time.Millisecond,
	}

	inventoryJSON := []byte(`{}`)
	DBOptions := vehicles.DBOptions{
		Fetcher: fetcher,
	}

	vehicleDB, err := vehicles.NewDB(inventoryJSON, DBOptions)

	suite.Require().NoError(err)

	// Act — fire many concurrent requests for the same ID
	var waitGroup sync.WaitGroup

	for range 20 {
		waitGroup.Go(func() {
			_, _ = vehicleDB.GetVehicleByID(vehicleID)
		})
	}

	waitGroup.Wait()

	// Wait for fetch to complete
	suite.Eventually(func() bool {
		v, _ := vehicleDB.GetVehicleByID(vehicleID)

		return v.Manufacturer == "DedupCo"
	}, 2*time.Second, 10*time.Millisecond)

	// Assert — singleflight should have collapsed all calls into one fetch
	suite.Equal(1, fetcher.callCount())
}

func (suite *VehiclesTestSuite) TestExponentialBackoffOnFetchFailure() {
	// Arrange
	var fetchCount atomic.Int32

	vehicleID := 12345

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetchCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	url, err := url.JoinPath(server.URL, "vehicles")
	suite.Require().NoError(err)

	fetcher := vehicles.NewHTTPFetcher(url)

	inventoryJSON := []byte(`{}`)
	DBOptions := vehicles.DBOptions{
		Fetcher: fetcher,
	}

	vehicleDB, err := vehicles.NewDB(inventoryJSON, DBOptions)
	suite.Require().NoError(err)

	// Act — first call triggers a fetch
	_, _ = vehicleDB.GetVehicleByID(vehicleID)

	// Wait for the background fetch to complete (and fail)
	suite.Eventually(func() bool {
		return fetchCount.Load() >= 1
	}, 2*time.Second, 10*time.Millisecond)

	// Second call should be backed off (no new fetch)
	countBefore := fetchCount.Load()
	_, _ = vehicleDB.GetVehicleByID(vehicleID)

	time.Sleep(50 * time.Millisecond) // give goroutine time to fire if it would

	countAfter := fetchCount.Load()

	// Assert
	suite.Equal(countBefore, countAfter, "backoff should prevent additional fetch attempts")
}

func (suite *VehiclesTestSuite) TestFetcherErrorDoesNotBlockCaller() {
	// Arrange
	wantVehicleID := 12345
	fetcher := &mockFetcher{
		err:        http.ErrServerClosed,
		fetchDelay: 200 * time.Millisecond,
	}

	inventoryJSON := []byte(`{}`)
	DBOptions := vehicles.DBOptions{
		Fetcher: fetcher,
	}

	vehicleDB, err := vehicles.NewDB(inventoryJSON, DBOptions)
	suite.Require().NoError(err)

	// Act — measure how long GetVehicleByID takes
	start := time.Now()
	got, err := vehicleDB.GetVehicleByID(wantVehicleID)
	elapsed := time.Since(start)

	// Assert — should return almost instantly with a stub, not block
	suite.Require().NoError(err)
	suite.Equal(wantVehicleID, got.CarID)
	suite.Empty(got.Manufacturer)
	suite.Less(elapsed, 50*time.Millisecond, "GetVehicleByID should not block on slow fetcher")
}

func (suite *VehiclesTestSuite) TestKnownVehicleDoesNotTriggerFetch() {
	// Arrange
	fetcher := &mockFetcher{
		vehicle: vehicles.Vehicle{CarID: 24, Manufacturer: "ShouldNotSee"},
	}

	inventoryJSON := []byte(`{
		"24": {
			"carId": 24,
			"manufacturer": "Nissan",
			"model": "180SX Type X '96"
		}
	}`)

	DBOptions := vehicles.DBOptions{
		Fetcher: fetcher,
	}

	vehicleDB, err := vehicles.NewDB(inventoryJSON, DBOptions)
	suite.Require().NoError(err)

	// Act
	got, err := vehicleDB.GetVehicleByID(24)

	// Assert — should return embedded data, no fetch
	suite.Require().NoError(err)
	suite.Equal("Nissan", got.Manufacturer)
	suite.Equal(0, fetcher.callCount())
}

func (suite *VehiclesTestSuite) TestCacheWriteAndLoad() {
	// Arrange
	tmpDir := suite.T().TempDir()

	vehicleID := 12345
	wantVehicle := vehicles.Vehicle{
		CarID:        vehicleID,
		Manufacturer: "CacheCo",
		Model:        "CacheCar",
		Year:         2024,
		Drivetrain:   "FR",
		Aspiration:   "NA",
		EngineLayout: "I4",
		CarType:      "street",
		LastModified: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	inventoryJSON := []byte(`{}`)
	fetcher := &mockFetcher{vehicle: wantVehicle}

	DBOptions := vehicles.DBOptions{
		Fetcher:  fetcher,
		CacheDir: tmpDir,
	}

	vehicleDB, err := vehicles.NewDB(inventoryJSON, DBOptions)
	suite.Require().NoError(err)

	// Act — trigger fetch and cache write
	_, _ = vehicleDB.GetVehicleByID(vehicleID)

	suite.Eventually(func() bool {
		v, _ := vehicleDB.GetVehicleByID(vehicleID)

		return v.Manufacturer == "CacheCo"
	}, 2*time.Second, 10*time.Millisecond)

	// Verify cache file was written
	cachePath := filepath.Join(tmpDir, strconv.Itoa(vehicleID)+".json")

	suite.Eventually(func() bool {
		_, err := os.Stat(cachePath)

		return err == nil
	}, 2*time.Second, 10*time.Millisecond, "cache file should exist")

	// Create a new DB that loads from the cache (no fetcher needed)
	DB2Options := vehicles.DBOptions{
		Fetcher:  nil,
		CacheDir: tmpDir,
	}
	db2, err := vehicles.NewDB(inventoryJSON, DB2Options)
	suite.Require().NoError(err)

	// Assert — should load from cache
	got, err := db2.GetVehicleByID(vehicleID)
	suite.Require().NoError(err)
	suite.Equal("CacheCo", got.Manufacturer)
	suite.Equal("CacheCar", got.Model)
}

func (suite *VehiclesTestSuite) TestCacheOverridesEmbeddedData() {
	// Arrange
	vehicleID := 12345
	tmpDir := suite.T().TempDir()

	cachedVehicle := vehicles.Vehicle{
		CarID:        vehicleID,
		Manufacturer: "UpdatedCo",
		Model:        "UpdatedCar",
		LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(cachedVehicle)
	suite.Require().NoError(err)

	err = os.WriteFile(filepath.Join(tmpDir, strconv.Itoa(vehicleID)+".json"), data, 0o600)
	suite.Require().NoError(err)

	inventoryJSON := []byte(`{
		"` + strconv.Itoa(vehicleID) + `": {
			"carId": ` + strconv.Itoa(vehicleID) + `,
			"manufacturer": "OriginalCo",
			"model": "OriginalCar"
		}
	}`)
	DBOptions := vehicles.DBOptions{
		Fetcher:  nil,
		CacheDir: tmpDir,
	}

	// Act
	vehicleDB, err := vehicles.NewDB(inventoryJSON, DBOptions)
	suite.Require().NoError(err)

	got, err := vehicleDB.GetVehicleByID(vehicleID)

	// Assert — cache should override embedded
	suite.Require().NoError(err)
	suite.Equal("UpdatedCo", got.Manufacturer)
	suite.Equal("UpdatedCar", got.Model)
}

func (suite *VehiclesTestSuite) TestDefaultRemoteFetchEnabledWithNoOptions() {
	// Arrange
	vehicleID := 12345
	wantVehicle := vehicles.Vehicle{
		CarID:        vehicleID,
		Manufacturer: "DefaultCo",
		Model:        "DefaultCar",
	}

	fetcher := &mockFetcher{vehicle: wantVehicle}

	inventoryJSON := []byte(`{}`)
	DBOptions := vehicles.DBOptions{
		Fetcher: fetcher,
	}

	vehicleDB, err := vehicles.NewDB(inventoryJSON, DBOptions)
	suite.Require().NoError(err)

	// Act
	got, _ := vehicleDB.GetVehicleByID(vehicleID)
	suite.Equal(vehicleID, got.CarID)
	suite.Empty(got.Manufacturer) // stub returned immediately

	// Assert — eventually populated
	suite.Eventually(func() bool {
		v, _ := vehicleDB.GetVehicleByID(vehicleID)

		return v.Manufacturer == "DefaultCo"
	}, 2*time.Second, 10*time.Millisecond)
}

func VehicleDBWithFetcherDisabled(inventoryJSON []byte) (*vehicles.VehicleDB, error) {
	options := vehicles.DBOptions{
		Fetcher: nil,
	}

	return vehicles.NewDB(inventoryJSON, options)
}

func (suite *VehiclesTestSuite) TestDownloadUpdatedVehiclesFetchesWhenExistingHasNoLastModified() {
	// Arrange — inventory has a vehicle with zero LastModified; manifest has a newer date
	vehicleID := 12345
	inventoryJSON := []byte(`{
		"12345": {
			"carId": 12345,
			"manufacturer": "OldCo",
			"model": "OldCar"
		}
	}`)

	updatedVehicle := vehicles.Vehicle{
		CarID:        vehicleID,
		Manufacturer: "NewCo",
		Model:        "NewCar",
		LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	fetcher := &mockFetcher{
		vehicle: updatedVehicle,
		onFetchManifest: func(_ context.Context) (*vehicles.Manifest, error) {
			return &vehicles.Manifest{
				Vehicles: map[string]vehicles.ManifestEntry{
					"12345": {LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
	}

	DBOptions := vehicles.DBOptions{
		Fetcher:       fetcher,
		UpdateBaseURL: "http://example.com",
	}

	vehicleDB, err := vehicles.NewDB(inventoryJSON, DBOptions)
	suite.Require().NoError(err)

	// Act
	count, err := vehicleDB.DownloadUpdatedVehicles(suite.T().Context())

	// Assert
	suite.Require().NoError(err)
	suite.Equal(1, count, "should download vehicle when existing has zero LastModified")

	got, err := vehicleDB.GetVehicleByID(vehicleID)
	suite.Require().NoError(err)
	suite.Equal("NewCo", got.Manufacturer)
	suite.Equal("NewCar", got.Model)
}

func (suite *VehiclesTestSuite) TestCheckForUpdatesDoesNothingWithoutUpdateBaseURL() {
	// Arrange
	inventoryJSON := []byte(`{}`)
	vehicleDB, err := vehicles.NewDB(inventoryJSON, vehicles.DBOptions{})
	suite.Require().NoError(err)

	// Act & Assert — should not panic, no goroutine launched
	suite.NotPanics(func() {
		vehicleDB.CheckForUpdates(suite.T().Context())
	})
}

func (suite *VehiclesTestSuite) TestCheckForUpdatesDownloadsUpdatesFromRemote() {
	// Arrange
	vehicleID := 42
	updatedVehicle := vehicles.Vehicle{
		CarID:        vehicleID,
		Manufacturer: "RemoteCo",
		Model:        "RemoteCar",
		LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")

		switch request.URL.Path {
		case "/vehicles/manifest.json":
			err := json.NewEncoder(response).Encode(vehicles.Manifest{
				Vehicles: map[string]vehicles.ManifestEntry{
					"42": {LastModified: updatedVehicle.LastModified},
				},
			})
			suite.NoError(err)
		case "/vehicles/42.json":
			err := json.NewEncoder(response).Encode(updatedVehicle)
			suite.NoError(err)
		default:
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	inventoryJSON := []byte(`{}`)
	vehicleDB, err := vehicles.NewDB(inventoryJSON, vehicles.DBOptions{
		UpdateBaseURL: server.URL,
	})
	suite.Require().NoError(err)

	// Act
	vehicleDB.CheckForUpdates(suite.T().Context())

	// Assert — eventually the inventory is updated
	suite.Eventually(func() bool {
		got, err := vehicleDB.GetVehicleByID(vehicleID)

		return err == nil && got.Manufacturer == "RemoteCo"
	}, 2*time.Second, 10*time.Millisecond)
}

func (suite *VehiclesTestSuite) TestCloseWithoutUpdaterDoesNotPanic() {
	// Arrange
	inventoryJSON := []byte(`{}`)
	vehicleDB, err := vehicles.NewDB(inventoryJSON, vehicles.DBOptions{})
	suite.Require().NoError(err)

	// Act & Assert
	suite.NotPanics(func() {
		vehicleDB.Close()
	})
}

func (suite *VehiclesTestSuite) TestCloseStopsBackgroundUpdater() {
	// Arrange — start a slow server so the goroutine is definitely running
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	inventoryJSON := []byte(`{}`)
	vehicleDB, err := vehicles.NewDB(inventoryJSON, vehicles.DBOptions{
		UpdateBaseURL: server.URL,
	})
	suite.Require().NoError(err)

	vehicleDB.CheckForUpdates(suite.T().Context())

	// Act & Assert — Close should not block or panic
	suite.NotPanics(func() {
		vehicleDB.Close()
	})
}

func (suite *VehiclesTestSuite) TestLoadCacheFileSkipsEmptyVehicle() {
	// Arrange — write a JSON file with zero carId and no manufacturer
	tmpDir := suite.T().TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "empty.json"), []byte(`{"carId":0,"manufacturer":""}`), 0o600)
	suite.Require().NoError(err)

	inventoryJSON := []byte(`{}`)
	vehicleDB, err := vehicles.NewDB(inventoryJSON, vehicles.DBOptions{CacheDir: tmpDir})
	suite.Require().NoError(err)

	// Assert — nothing loaded
	suite.Equal(0, vehicleDB.Len())
}

func (suite *VehiclesTestSuite) TestLoadCacheFileSkipsMalformedJSON() {
	// Arrange — write an invalid JSON file
	tmpDir := suite.T().TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte(`{not json`), 0o600)
	suite.Require().NoError(err)

	inventoryJSON := []byte(`{}`)

	// Act & Assert — should not panic, bad file is silently skipped
	suite.NotPanics(func() {
		_, err = vehicles.NewDB(inventoryJSON, vehicles.DBOptions{CacheDir: tmpDir})
		suite.Require().NoError(err)
	})
}

func (suite *VehiclesTestSuite) TestLoadCacheDirSkipsNonJSONFiles() {
	// Arrange
	tmpDir := suite.T().TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("not json"), 0o600)
	suite.Require().NoError(err)

	err = os.WriteFile(filepath.Join(tmpDir, "subdir"), []byte{}, 0o600)
	suite.Require().NoError(err)

	inventoryJSON := []byte(`{}`)
	vehicleDB, err := vehicles.NewDB(inventoryJSON, vehicles.DBOptions{CacheDir: tmpDir})
	suite.Require().NoError(err)

	// Assert — nothing loaded from non-JSON files
	suite.Equal(0, vehicleDB.Len())
}

func (suite *VehiclesTestSuite) TestWriteCacheDoesNothingWithNoCacheDir() {
	// Arrange — no cache dir set
	vehicleID := 99
	vehicle := vehicles.Vehicle{CarID: vehicleID, Manufacturer: "NoCacheCo"}

	fetcher := &mockFetcher{vehicle: vehicle}
	inventoryJSON := []byte(`{}`)
	vehicleDB, err := vehicles.NewDB(inventoryJSON, vehicles.DBOptions{Fetcher: fetcher})
	suite.Require().NoError(err)

	// Act — trigger fetch which calls writeCache internally
	_, _ = vehicleDB.GetVehicleByID(vehicleID)

	suite.Eventually(func() bool {
		got, _ := vehicleDB.GetVehicleByID(vehicleID)

		return got.Manufacturer == "NoCacheCo"
	}, 2*time.Second, 10*time.Millisecond)

	// Assert — no panic, vehicle is in memory but no file on disk
	suite.Equal("NoCacheCo", func() string {
		got, _ := vehicleDB.GetVehicleByID(vehicleID)

		return got.Manufacturer
	}())
}

func (suite *VehiclesTestSuite) TestWriteCacheCreatesDirectoryIfMissing() {
	// Arrange
	tmpDir := suite.T().TempDir()
	cacheDir := filepath.Join(tmpDir, "nested", "vehicles")
	vehicleID := 77
	vehicle := vehicles.Vehicle{
		CarID:        vehicleID,
		Manufacturer: "DirCo",
		Model:        "DirCar",
		LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	inventoryJSON := []byte(`{}`)
	vehicleDB, err := vehicles.NewDB(inventoryJSON, vehicles.DBOptions{
		Fetcher:  &mockFetcher{vehicle: vehicle},
		CacheDir: cacheDir,
	})
	suite.Require().NoError(err)

	// Act — trigger fetch
	_, _ = vehicleDB.GetVehicleByID(vehicleID)

	// Assert — cache file is eventually written
	cachePath := filepath.Join(cacheDir, strconv.Itoa(vehicleID)+".json")

	suite.Eventually(func() bool {
		_, err := os.Stat(cachePath)

		return err == nil
	}, 2*time.Second, 10*time.Millisecond, "cache file should be created in nested directory")
}

func (suite *VehiclesTestSuite) TestRecordFailureIncreasesBackoffDelay() {
	// Arrange — fetcher always errors so multiple failures accumulate
	vehicleID := 55
	fetchCount := atomic.Int32{}

	inventoryJSON := []byte(`{}`)
	vehicleDB, err := vehicles.NewDB(inventoryJSON, vehicles.DBOptions{
		Fetcher: &mockFetcher{
			err: errors.New("remote down"), //nolint:err113 // test-only sentinel
			onFetchManifest: func(_ context.Context) (*vehicles.Manifest, error) {
				fetchCount.Add(1)

				return nil, errors.New("remote down") //nolint:err113 // test-only sentinel
			},
		},
	})
	suite.Require().NoError(err)

	// Act — trigger two consecutive failures
	_, _ = vehicleDB.GetVehicleByID(vehicleID)

	suite.Eventually(func() bool {
		got, _ := vehicleDB.GetVehicleByID(vehicleID)

		return got.CarID == vehicleID
	}, 2*time.Second, 10*time.Millisecond)

	// The second call should be in backoff — returns stub without fetching
	got, err := vehicleDB.GetVehicleByID(vehicleID)
	suite.Require().NoError(err)
	suite.Equal(vehicleID, got.CarID)
}
