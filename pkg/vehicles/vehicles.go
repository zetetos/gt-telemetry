package vehicles

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/singleflight"
)

const (
	initialBackoff    = 1 * time.Second
	maxBackoff        = 5 * time.Minute
	backoffMultiplier = 2
)

var ErrVehicleNotFound = errors.New("no vehicle found with id")

// Vehicle represents information about a specific vehicle.
type Vehicle struct {
	CarID                 int       `csv:"CarId"                 json:"carId"`
	Manufacturer          string    `csv:"Manufacturer"          json:"manufacturer"`
	Model                 string    `csv:"Model"                 json:"model"`
	Year                  int       `csv:"Year"                  json:"year"`
	OpenCockpit           bool      `csv:"OpenCockpit"           json:"openCockpit"`
	CarType               string    `csv:"CarType"               json:"carType"`
	Category              string    `csv:"Category"              json:"category"`
	Drivetrain            string    `csv:"Drivetrain"            json:"drivetrain"`
	Aspiration            string    `csv:"Aspiration"            json:"aspiration"`
	Length                int       `csv:"Length"                json:"length"`
	Width                 int       `csv:"Width"                 json:"width"`
	Height                int       `csv:"Height"                json:"height"`
	Wheelbase             int       `csv:"Wheelbase"             json:"wheelbase"`
	TrackFront            int       `csv:"TrackFront"            json:"trackFront"`
	TrackRear             int       `csv:"TrackRear"             json:"trackRear"`
	EngineLayout          string    `csv:"EngineLayout"          json:"engineLayout"`
	EngineBankAngle       float32   `csv:"EngineBankAngle"       json:"engineBankAngle"`
	EngineCrankPlaneAngle float32   `csv:"EngineCrankPlaneAngle" json:"engineCrankPlaneAngle"`
	LastModified          time.Time `csv:"-"                     json:"lastModified,omitzero"`
}

// VehicleInventory represents the complete JSON structure from the embedded vehicle inventory data.
type VehicleInventory map[string]Vehicle

type backoffState struct {
	nextAttempt time.Time
	failures    int
}

// DBOptions configures optional behaviour for VehicleDB.
type DBOptions struct {
	CacheDir      string
	UpdateBaseURL string
	Fetcher       Fetcher
	Logger        *zerolog.Logger
}

// VehicleDB provides an object and methods to access vehicle information from the embedded inventory.
type VehicleDB struct {
	mu             sync.RWMutex
	inventory      VehicleInventory
	latestModified time.Time
	fetcher        Fetcher
	group          singleflight.Group
	backoff        map[int]*backoffState
	backoffMu      sync.Mutex
	cacheDir       string
	updateBaseURL  string
	cancel         context.CancelFunc
	log            *zerolog.Logger
}

//go:embed inventory
var baseInventoryFS embed.FS

func loadBaseInventory() (VehicleInventory, error) {
	inventory := VehicleInventory{}

	entries, err := baseInventoryFS.ReadDir("inventory")
	if err != nil {
		return inventory, fmt.Errorf("read embedded inventory directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := baseInventoryFS.ReadFile("inventory/" + entry.Name())
		if err != nil {
			return inventory, fmt.Errorf("read embedded inventory file %s: %w", entry.Name(), err)
		}

		var vehicle Vehicle

		err = json.Unmarshal(data, &vehicle)
		if err != nil {
			return inventory, fmt.Errorf("parse embedded inventory file %s: %w", entry.Name(), err)
		}

		inventory[strconv.Itoa(vehicle.CarID)] = vehicle
	}

	return inventory, nil
}

// NewDB creates a new VehicleDB instance by loading the vehicle inventory from embedded JSON data.
// Optional DBOption values configure remote fetching, caching, and logging.
func NewDB(inventoryJSON []byte, opts DBOptions) (*VehicleDB, error) {
	inventory := VehicleInventory{}

	if len(inventoryJSON) != 0 {
		err := json.Unmarshal(inventoryJSON, &inventory)
		if err != nil {
			return &VehicleDB{}, fmt.Errorf("unmarshall vehicle inventory JSON: %w", err)
		}
	} else {
		var err error

		inventory, err = loadBaseInventory()
		if err != nil {
			return &VehicleDB{}, err
		}
	}

	cacheDir := opts.CacheDir
	fetcher := opts.Fetcher
	updateBaseURL := opts.UpdateBaseURL

	if updateBaseURL != "" {
		var err error

		updateBaseURL, err = url.JoinPath(opts.UpdateBaseURL, "vehicles")
		if err != nil {
			return nil, fmt.Errorf("build vehicle update URL: %w", err)
		}

		if fetcher == nil {
			fetcher = NewHTTPFetcher(updateBaseURL)
		}
	}

	var logger zerolog.Logger
	if opts.Logger == nil {
		logger = zerolog.Nop().With().Str("component", "vehicle db").Logger()
	} else {
		logger = opts.Logger.With().Str("component", "vehicle db").Logger()
	}

	vehicleDB := &VehicleDB{
		inventory:     inventory,
		backoff:       make(map[int]*backoffState),
		cacheDir:      cacheDir,
		updateBaseURL: updateBaseURL,
		fetcher:       fetcher,
		log:           &logger,
	}

	vehicleDB.loadCacheDir()
	vehicleDB.updateLatestModified()

	return vehicleDB, nil
}

// CheckForUpdates launches a background goroutine that fetches the remote manifest,
// downloads any vehicles newer than the local versions, writes them to cacheDir,
// and merges them into the in-memory inventory. The context controls the goroutine
// lifetime. Does nothing if UpdateBaseURL was not set in DBOptions.
func (db *VehicleDB) CheckForUpdates(ctx context.Context) {
	if db.updateBaseURL == "" {
		return
	}

	ctx, db.cancel = context.WithCancel(ctx)
	db.fetcher = NewHTTPFetcher(db.updateBaseURL)

	go db.fetchUpdates(ctx)
}

// Close cancels the background updater goroutine if one is running.
func (db *VehicleDB) Close() {
	if db.cancel != nil {
		db.cancel()
	}
}

// LatestModified returns the newest LastModified timestamp across all vehicles
// in the inventory.
func (db *VehicleDB) LatestModified() time.Time {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.latestModified
}

// GetVehicleByID retrieves a Vehicle from the inventory by its CarID.
// If the vehicle is not found locally, a stub Vehicle is returned immediately
// and a background fetch is triggered to retrieve it from the remote service.
func (db *VehicleDB) GetVehicleByID(vehicleID int) (Vehicle, error) {
	db.mu.RLock()
	vehicle, ok := db.inventory[strconv.Itoa(vehicleID)]
	db.mu.RUnlock()

	if ok {
		return vehicle, nil
	}

	if db.fetcher == nil {
		return Vehicle{}, fmt.Errorf("%w: %d", ErrVehicleNotFound, vehicleID)
	}

	if db.backoffActive(vehicleID) {
		return Vehicle{CarID: vehicleID}, nil
	}

	go db.fetchAndStore(vehicleID)

	return Vehicle{CarID: vehicleID}, nil
}

// updateLatestModified recalculates the latest modified time from all inventory entries.
func (db *VehicleDB) updateLatestModified() {
	var latest time.Time

	for _, v := range db.inventory {
		if v.LastModified.After(latest) {
			latest = v.LastModified
		}
	}

	db.latestModified = latest
}

// fetchUpdates fetches the remote manifest and downloads any vehicles newer than
// local versions, merging them into the in-memory inventory.
func (db *VehicleDB) fetchUpdates(ctx context.Context) {
	count, err := db.downloadUpdatedVehicles(ctx)
	if err != nil {
		db.log.Warn().Err(err).Msg("failed to check for vehicle updates")

		return
	}

	if count > 0 {
		db.log.Info().Int("count", count).Msg("downloaded updated vehicles")
	}
}

// downloadUpdatedVehicles fetches the remote manifest and downloads any vehicles
// that are newer than the local versions, writing them to the cache directory.
func (db *VehicleDB) downloadUpdatedVehicles(ctx context.Context) (int, error) {
	manifest, err := db.fetcher.FetchManifest(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetching vehicle manifest: %w", err)
	}

	count := 0

	for carIDStr, entry := range manifest.Vehicles {
		db.mu.RLock()
		existing, exists := db.inventory[carIDStr]
		db.mu.RUnlock()

		if exists && !existing.LastModified.IsZero() && !entry.LastModified.After(existing.LastModified) {
			continue
		}

		carID, err := strconv.Atoi(carIDStr)
		if err != nil {
			db.log.Warn().Str("car_id", carIDStr).Msg("invalid car ID in manifest")

			continue
		}

		vehicle, err := db.fetcher.FetchVehicle(ctx, carID)
		if err != nil {
			db.log.Warn().Err(err).Int("car_id", carID).Msg("failed to fetch updated vehicle")

			continue
		}

		if vehicle.LastModified.IsZero() {
			vehicle.LastModified = entry.LastModified
		}

		db.mu.Lock()

		db.inventory[carIDStr] = vehicle
		if vehicle.LastModified.After(db.latestModified) {
			db.latestModified = vehicle.LastModified
		}

		db.mu.Unlock()

		db.writeCache(carID, vehicle)

		db.log.Info().Int("car_id", carID).Str("manufacturer", vehicle.Manufacturer).Str("model", vehicle.Model).Msg("updated vehicle from remote")

		count++
	}

	return count, nil
}

// loadCacheDir scans the cache directory and merges any cached vehicle JSON files into the inventory.
func (db *VehicleDB) loadCacheDir() {
	if db.cacheDir == "" {
		return
	}

	entries, err := os.ReadDir(db.cacheDir)
	if err != nil {
		// Directory may not exist yet; that's fine.
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		db.loadCacheFile(entry.Name())
	}
}

// loadCacheFile reads a single cached vehicle JSON file and merges it into the inventory.
func (db *VehicleDB) loadCacheFile(name string) {
	data, err := os.ReadFile(filepath.Join(db.cacheDir, name))
	if err != nil {
		db.log.Warn().Err(err).Str("file", name).Msg("failed to read cached vehicle file")

		return
	}

	var vehicle Vehicle

	err = json.Unmarshal(data, &vehicle)
	if err != nil {
		db.log.Warn().Err(err).Str("file", name).Msg("failed to parse cached vehicle file")

		return
	}

	if vehicle.CarID == 0 && vehicle.Manufacturer == "" {
		return
	}

	key := strconv.Itoa(vehicle.CarID)

	existing, exists := db.inventory[key]
	if !exists || vehicle.LastModified.After(existing.LastModified) {
		db.inventory[key] = vehicle
	}
}

// fetchAndStore fetches a vehicle from the remote service via singleflight, stores it in the
// inventory, and writes it to the disk cache.
func (db *VehicleDB) fetchAndStore(vehicleID int) {
	key := strconv.Itoa(vehicleID)

	_, fetchErr, _ := db.group.Do(key, func() (any, error) {
		vehicle, err := db.fetcher.FetchVehicle(context.Background(), vehicleID)
		if err != nil {
			db.recordFailure(vehicleID)
			db.log.Warn().Err(err).Int("car_id", vehicleID).Msg("failed to fetch vehicle from remote")

			return nil, err //nolint:wrapcheck
		}

		if vehicle.LastModified.IsZero() {
			vehicle.LastModified = time.Now().UTC()
		}

		db.mu.Lock()

		db.inventory[key] = vehicle
		if vehicle.LastModified.After(db.latestModified) {
			db.latestModified = vehicle.LastModified
		}

		db.mu.Unlock()

		db.resetBackoff(vehicleID)
		db.writeCache(vehicleID, vehicle)

		db.log.Info().Int("car_id", vehicleID).Str("manufacturer", vehicle.Manufacturer).Str("model", vehicle.Model).Msg("fetched vehicle from remote")

		return vehicle, nil
	})
	if fetchErr != nil {
		return
	}
}

// writeCache writes a single vehicle JSON file to the cache directory.
func (db *VehicleDB) writeCache(vehicleID int, vehicle Vehicle) {
	if db.cacheDir == "" {
		return
	}

	err := os.MkdirAll(db.cacheDir, 0o750)
	if err != nil {
		db.log.Warn().Err(err).Str("path", db.cacheDir).Msg("failed to create vehicle cache directory")

		return
	}

	data, err := json.MarshalIndent(vehicle, "", "  ")
	if err != nil {
		db.log.Warn().Err(err).Int("car_id", vehicleID).Msg("failed to marshal vehicle for cache")

		return
	}

	path := filepath.Join(db.cacheDir, strconv.Itoa(vehicleID)+".json")

	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		db.log.Warn().Err(err).Str("path", path).Msg("failed to write vehicle cache file")
	}
}

// backoffActive checks if the given vehicle ID is currently in backoff due to recent fetch failures.
func (db *VehicleDB) backoffActive(id int) bool {
	db.backoffMu.Lock()
	defer db.backoffMu.Unlock()

	state, exists := db.backoff[id]
	if !exists {
		return false
	}

	return time.Now().Before(state.nextAttempt)
}

// recordFailure updates the backoff state for a vehicle ID after a failed fetch attempt.
func (db *VehicleDB) recordFailure(vehicleID int) {
	db.backoffMu.Lock()
	defer db.backoffMu.Unlock()

	state, exists := db.backoff[vehicleID]
	if !exists {
		state = &backoffState{}
		db.backoff[vehicleID] = state
	}

	state.failures++

	delay := initialBackoff
	for range state.failures - 1 {
		delay *= backoffMultiplier
		if delay > maxBackoff {
			delay = maxBackoff

			break
		}
	}

	state.nextAttempt = time.Now().Add(delay)
}

// resetBackoff clears the backoff state for a vehicle ID after a successful fetch.
func (db *VehicleDB) resetBackoff(id int) {
	db.backoffMu.Lock()
	defer db.backoffMu.Unlock()

	delete(db.backoff, id)
}

// ExpandedAspiration provides a human-readable description of the vehicle's aspiration type.
func (v *Vehicle) ExpandedAspiration() string {
	switch v.Aspiration {
	case "EV":
		return "Electric Vehicle"
	case "NA":
		return "Naturally Aspirated"
	case "TC":
		return "Turbocharged"
	case "SC":
		return "Supercharged"
	case "TC+SC":
		return "Compound Charged"
	default:
		return v.Aspiration
	}
}
