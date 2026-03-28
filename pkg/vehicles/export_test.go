package vehicles

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
)

// EmbeddedInventoryCount returns the number of vehicle files in the embedded inventory.
func EmbeddedInventoryCount() (int, error) {
	entries, err := fs.ReadDir(baseInventoryFS, "inventory")
	if err != nil {
		return 0, fmt.Errorf("reading embedded inventory: %w", err)
	}

	count := 0

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" { //nolint:goconst // readability
			count++
		}
	}

	return count, nil
}

// Len returns the number of vehicles currently in the DB's inventory.
func (db *VehicleDB) Len() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return len(db.inventory)
}

// DownloadUpdatedVehicles exposes downloadUpdatedVehicles for testing.
func (db *VehicleDB) DownloadUpdatedVehicles(ctx context.Context) (int, error) {
	return db.downloadUpdatedVehicles(ctx)
}
