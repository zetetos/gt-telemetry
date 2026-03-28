package circuits

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed all:inventory
var embeddedInventoryFS embed.FS

// ManifestEntry holds per-circuit metadata in the remote manifest.
type ManifestEntry struct {
	LastModified time.Time `json:"lastModified"`
}

// Manifest is the structure served by the remote update server.
type Manifest struct {
	Circuits map[string]ManifestEntry `json:"circuits"`
}

// loadFromFS reads all circuit JSON files from the given fs.FS and directory,
// returning a map of circuit ID to CircuitInfo.
func loadFromFS(fsys fs.FS, dir string) (map[string]CircuitInfo, error) {
	circuits := make(map[string]CircuitInfo)

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("reading circuit directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Skip manifest.json if present
		if entry.Name() == "manifest.json" {
			continue
		}

		data, readErr := fs.ReadFile(fsys, filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("reading circuit file %s: %w", entry.Name(), readErr)
		}

		var info CircuitInfo

		readErr = json.Unmarshal(data, &info)
		if readErr != nil {
			return nil, fmt.Errorf("parsing circuit file %s: %w", entry.Name(), readErr)
		}

		if info.ID == "" {
			info.ID = strings.TrimSuffix(entry.Name(), ".json")
		}

		circuits[info.ID] = info
	}

	return circuits, nil
}

// loadCacheDir reads circuit files from an OS directory using the fs.FS interface.
func loadCacheDir(cacheDir string) (map[string]CircuitInfo, error) {
	return loadFromFS(os.DirFS(cacheDir), ".")
}

// buildLookupMaps constructs the coordinate and start line lookup maps from a
// set of loaded circuits. Coordinates on each CircuitInfo are set to nil after
// building the maps to free memory.
func buildLookupMaps(circuits map[string]CircuitInfo) *circuitInventory {
	// Count how many circuits claim each coordinate
	coordCounts := make(map[string][]string)

	for circuitID, info := range circuits {
		for _, coord := range info.Coordinates {
			key := coord.String()
			coordCounts[key] = append(coordCounts[key], circuitID)
		}
	}

	// Only keep coordinates unique to a single circuit
	coordinates := make(map[string]string)
	uniquePerCircuit := make(map[string]int)

	for key, ids := range coordCounts {
		if len(ids) == 1 {
			coordinates[key] = ids[0]
			uniquePerCircuit[ids[0]]++
		}
	}

	// Build start line lookup
	startLines := make(map[string][]string)

	for circuitID, info := range circuits {
		key := info.StartLine.String()
		startLines[key] = append(startLines[key], circuitID)
	}

	// Nil out coordinate slices to free memory and set unique coordinate counts
	for id, info := range circuits {
		info.Coordinates = nil
		info.UniqueCoordinateCount = uniquePerCircuit[id]
		circuits[id] = info
	}

	return &circuitInventory{
		coordinates: coordinates,
		startLines:  startLines,
		circuits:    circuits,
	}
}

// fetchUpdates fetches the remote manifest, downloads any updated circuits, writes
// them to the cache directory, and hot-swaps the in-memory inventory.
func (db *CircuitDB) fetchUpdates(ctx context.Context) error {
	updated, err := db.downloadUpdatedCircuits(ctx)
	if err != nil {
		return err
	}

	if !updated {
		return nil
	}

	return db.rebuildInventory()
}

// downloadUpdatedCircuits fetches the remote manifest and downloads any circuits
// that are newer than the local versions, writing them to the cache directory.
func (db *CircuitDB) downloadUpdatedCircuits(ctx context.Context) (didUpdate bool, err error) {
	remoteManifest, err := db.fetcher.FetchManifest(ctx)
	if err != nil {
		return false, fmt.Errorf("fetching remote manifest: %w", err)
	}

	db.mu.RLock()
	currentCircuits := db.inventory.circuits
	db.mu.RUnlock()

	didUpdate = false

	for circuitID, entry := range remoteManifest.Circuits {
		local, exists := currentCircuits[circuitID]
		if exists && !entry.LastModified.After(local.LastModified) {
			continue
		}

		info, dlErr := db.fetcher.FetchCircuit(ctx, circuitID)
		if dlErr != nil {
			continue
		}

		if db.cacheDir != "" {
			writeErr := writeCircuitCache(db.cacheDir, circuitID, info)
			if writeErr != nil {
				continue
			}
		}

		didUpdate = true
	}

	return didUpdate, nil
}

// rebuildInventory reloads the inventory from embedded and cached circuit files
// and atomically swaps the in-memory inventory.
func (db *CircuitDB) rebuildInventory() error {
	circuits, err := loadFromFS(embeddedInventoryFS, "inventory")
	if err != nil {
		return fmt.Errorf("reloading embedded circuits: %w", err)
	}

	if db.cacheDir != "" {
		cached, cacheErr := loadCacheDir(db.cacheDir)
		if cacheErr == nil {
			maps.Copy(circuits, cached)
		}
	}

	inv := buildLookupMaps(circuits)

	db.mu.Lock()
	db.inventory = inv
	db.mu.Unlock()

	db.updateLatestModified()

	return nil
}

// writeCircuitCache writes a circuit file to the cache directory.
func writeCircuitCache(cacheDir string, circuitID string, info *CircuitInfo) error {
	err := os.MkdirAll(cacheDir, 0o755)
	if err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling circuit %s: %w", circuitID, err)
	}

	outPath := filepath.Join(cacheDir, circuitID+".json")

	err = os.WriteFile(outPath, data, 0o644) //nolint:gosec // Cache file permissions are acceptable
	if err != nil {
		return fmt.Errorf("writing cache file %s: %w", outPath, err)
	}

	return nil
}
