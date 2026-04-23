package circuits

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
)

// Exported aliases for internal functions, used only by tests.
var (
	LoadFromFS        = loadFromFS        //nolint:gochecknoglobals // test export
	LoadCacheDir      = loadCacheDir      //nolint:gochecknoglobals // test export
	WriteCircuitCache = writeCircuitCache //nolint:gochecknoglobals // test export
)

// SetFetcher injects a Fetcher into a CircuitDB for testing.
func (db *CircuitDB) SetFetcher(f Fetcher) {
	db.fetcher = f
}

// FetchUpdates exposes fetchUpdates for testing.
func (db *CircuitDB) FetchUpdates(ctx context.Context) error {
	return db.fetchUpdates(ctx)
}

// DownloadUpdatedCircuits exposes downloadUpdatedCircuits for testing.
func (db *CircuitDB) DownloadUpdatedCircuits(ctx context.Context) (didUpdate bool, err error) {
	return db.downloadUpdatedCircuits(ctx)
}

// RebuildInventory exposes rebuildInventory for testing.
func (db *CircuitDB) RebuildInventory() error {
	return db.rebuildInventory()
}

// TestInventory holds the result of buildLookupMaps with exported fields for test assertions.
type TestInventory struct {
	Coordinates map[string]string
	StartLines  map[string][]string
	Circuits    map[string]CircuitInfo
}

// EmbeddedInventoryCount returns the number of circuit files in the embedded inventory.
func EmbeddedInventoryCount() (int, error) {
	entries, err := fs.ReadDir(embeddedInventoryFS, "inventory")
	if err != nil {
		return 0, fmt.Errorf("reading embedded inventory: %w", err)
	}

	count := 0

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && entry.Name() != "manifest.json" {
			count++
		}
	}

	return count, nil
}

// NewDBFromCircuits creates a CircuitDB pre-populated with the provided circuits,
// bypassing the embedded inventory and cache.
func NewDBFromCircuits(circuits map[string]CircuitInfo) *CircuitDB {
	return &CircuitDB{inventory: buildLookupMaps(circuits)}
}

// BuildLookupMapsForTest wraps buildLookupMaps and returns an exported struct.
func BuildLookupMapsForTest(circuits map[string]CircuitInfo) *TestInventory {
	inv := buildLookupMaps(circuits)

	return &TestInventory{
		Coordinates: inv.coordinates,
		StartLines:  inv.startLines,
		Circuits:    inv.circuits,
	}
}
