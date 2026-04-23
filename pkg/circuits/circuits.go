package circuits

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/zetetos/gt-telemetry/v2/pkg/models"
)

const (
	circuitCoorindateResolutionX = 16
	circuitCoorindateResolutionY = 2
	circuitCoorindateResolutionZ = 16

	startLineCoorindateResolutionX = 16
	startLineCoorindateResolutionY = 2
	startLineCoorindateResolutionZ = 16
)

// CircuitInfo represents information about a specific race circuit.
type CircuitInfo struct {
	ID                    string                  `json:"id"`
	Name                  string                  `json:"name"`
	Variation             string                  `json:"variation"`
	Country               string                  `json:"country"`
	Default               bool                    `json:"default"`
	Length                int                     `json:"length"`
	StartLine             models.CoordinateNorm   `json:"startLine"`
	LastModified          time.Time               `json:"lastModified"`
	Coordinates           []models.CoordinateNorm `json:"coordinates"`
	UniqueCoordinateCount int                     `json:"-"`
}

// circuitInventory holds the lookup maps and circuit metadata built at load time.
type circuitInventory struct {
	coordinates map[string]string      // normalised coord string → circuitID (unique coords only)
	startLines  map[string][]string    // start coord string → []circuitID
	circuits    map[string]CircuitInfo // circuitID → metadata (coordinates nil after map building)
}

// CircuitDB provides thread-safe access to circuit information loaded from embedded and cached data.
type CircuitDB struct {
	mu             sync.RWMutex
	inventory      *circuitInventory
	latestModified time.Time
	fetcher        Fetcher
	cacheDir       string
	updateBaseURL  string
	cancel         context.CancelFunc
	log            *zerolog.Logger
}

// CircuitDBOptions configures optional behaviour for CircuitDB.
type CircuitDBOptions struct {
	CacheDir      string
	UpdateBaseURL string
	Logger        *zerolog.Logger
}

// NewDB creates a new CircuitDB by loading circuits from the embedded inventory files,
// overlaid by any cached circuit files found in opts.CacheDir. If opts.CacheDir is empty,
// only embedded circuits are loaded. The returned CircuitDB is immediately usable.
func NewDB(opts CircuitDBOptions) (*CircuitDB, error) {
	circuits, err := loadFromFS(embeddedInventoryFS, "inventory")
	if err != nil {
		return nil, err
	}

	inventory := buildLookupMaps(circuits)

	cacheDir := opts.CacheDir

	updateBaseURL := ""

	if opts.UpdateBaseURL != "" {
		var err error

		updateBaseURL, err = url.JoinPath(opts.UpdateBaseURL, "circuits")
		if err != nil {
			return nil, fmt.Errorf("build circuit update URL: %w", err)
		}
	}

	var logger zerolog.Logger
	if opts.Logger == nil {
		logger = zerolog.Nop().With().Str("component", "circuit db").Logger()
	} else {
		logger = opts.Logger.With().Str("component", "circuit db").Logger()
	}

	circuitDB := &CircuitDB{
		inventory:     inventory,
		cacheDir:      cacheDir,
		updateBaseURL: updateBaseURL,
		log:           &logger,
	}

	circuitDB.loadCacheDir()
	circuitDB.updateLatestModified()

	return circuitDB, nil
}

// CheckForUpdates launches a background goroutine that fetches the remote manifest,
// downloads any circuits newer than the local versions, writes them to cacheDir,
// and hot-swaps the in-memory inventory. The context controls the goroutine lifetime.
// Does nothing if UpdateBaseURL was not set in CircuitDBOptions.
func (db *CircuitDB) CheckForUpdates(ctx context.Context) {
	if db.updateBaseURL == "" {
		return
	}

	ctx, db.cancel = context.WithCancel(ctx)
	db.fetcher = NewHTTPFetcher(db.updateBaseURL)

	go func() {
		err := db.fetchUpdates(ctx)
		if err != nil {
			db.log.Warn().Err(err).Msg("failed to check for circuit updates")
		}
	}()
}

// Close cancels the background updater goroutine if one is running.
func (db *CircuitDB) Close() {
	if db.cancel != nil {
		db.cancel()
	}
}

// LatestModified returns the newest LastModified timestamp across all circuits
// in the inventory.
func (db *CircuitDB) LatestModified() time.Time {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.latestModified
}

// GetCircuitAtCoordinate returns the circuit at a given coordinate.
func (db *CircuitDB) GetCircuitAtCoordinate(coordinate models.Coordinate, coordType models.CoordinateType) (circuitID string, found bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.inventory == nil {
		return "", false
	}

	if coordType == models.CoordinateTypeStartLine {
		normalisedPos := NormaliseStartLineCoordinate(coordinate)
		key := normalisedPos.String()

		circuitIDs, ok := db.inventory.startLines[key]
		if ok && len(circuitIDs) == 1 {
			return circuitIDs[0], true
		}

		return "", false
	}

	normalisedPos := NormaliseCircuitCoordinate(coordinate)
	key := normalisedPos.String()

	circuitID, found = db.inventory.coordinates[key]

	return circuitID, found
}

// GetCircuitByID retrieves a CircuitInfo by its ID.
func (db *CircuitDB) GetCircuitByID(circuitID string) (circuit CircuitInfo, found bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.inventory == nil {
		return CircuitInfo{}, false
	}

	circuit, found = db.inventory.circuits[circuitID]
	circuit.ID = circuitID

	return circuit, found
}

// GetAllCircuitIDs returns all available circuit IDs.
func (db *CircuitDB) GetAllCircuitIDs() (circuitIDs []string) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.inventory == nil {
		return nil
	}

	circuitIDs = make([]string, 0, len(db.inventory.circuits))
	for circuitID := range db.inventory.circuits {
		circuitIDs = append(circuitIDs, circuitID)
	}

	return circuitIDs
}

// updateLatestModified recalculates the latest modified time from all inventory entries.
func (db *CircuitDB) updateLatestModified() {
	var latest time.Time

	for _, c := range db.inventory.circuits {
		if c.LastModified.After(latest) {
			latest = c.LastModified
		}
	}

	db.latestModified = latest
}

// NormaliseStartLineCoordinate normalises a start line coordinate to reduce precision for location matching.
func NormaliseStartLineCoordinate(coordinate models.Coordinate) (normalised models.CoordinateNorm) {
	// The FIA rules state that the starting grid has a min width of 15 metres.
	// 32m resolution should provide sufficient accuracy for most tracks.
	return coordinate.Normalise(
		startLineCoorindateResolutionX,
		startLineCoorindateResolutionY,
		startLineCoorindateResolutionZ,
	)
}

// NormaliseCircuitCoordinate normalises a circuit coordinate to reduce precision for location matching.
func NormaliseCircuitCoordinate(coordinate models.Coordinate) (normalised models.CoordinateNorm) {
	// Track map resultion is lower to reduce file size.
	// 64m resolution should be sufficient for most tracks.
	// Y (vertical) resolution is higher since elevation changes are much smaller than X/Z.
	return coordinate.Normalise(
		circuitCoorindateResolutionX,
		circuitCoorindateResolutionY,
		circuitCoorindateResolutionZ,
	)
}

// loadCacheDir scans the cache directory and merges any cached circuit JSON files into the inventory.
func (db *CircuitDB) loadCacheDir() {
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

// loadCacheFile reads a single cached circuit JSON file and merges it into the inventory.
func (db *CircuitDB) loadCacheFile(name string) {
	data, err := os.ReadFile(filepath.Join(db.cacheDir, name))
	if err != nil {
		return
	}

	var circuit CircuitInfo

	err = json.Unmarshal(data, &circuit)
	if err != nil {
		return
	}

	if circuit.ID == "" {
		return
	}

	existing, exists := db.inventory.circuits[circuit.ID]
	if !exists || circuit.LastModified.After(existing.LastModified) {
		db.inventory.circuits[circuit.ID] = circuit
	}
}
