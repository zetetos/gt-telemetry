package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/zetetos/gt-telemetry/v2/pkg/vehicles"
)

// loadInventoryDir reads individual vehicle JSON files from dir and returns a vehicle map keyed by CarID string.
func loadInventoryDir(dir string) (map[string]vehicles.Vehicle, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading inventory directory: %w", err)
	}

	vehicleMap := make(map[string]vehicles.Vehicle)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || entry.Name() == "manifest.json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		var vehicle vehicles.Vehicle

		err = json.Unmarshal(data, &vehicle)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		vehicleMap[strconv.Itoa(vehicle.CarID)] = vehicle
	}

	return vehicleMap, nil
}

// writeInventoryDir writes each vehicle in vehicleMap as an individual JSON file to outputDir
// and updates manifest.json. Returns the number of vehicle files written.
func writeInventoryDir(vehicleMap map[string]vehicles.Vehicle, outputDir string) (int, error) {
	err := os.MkdirAll(outputDir, 0o750)
	if err != nil {
		return 0, fmt.Errorf("creating output directory: %w", err)
	}

	written := 0

	for _, vehicle := range vehicleMap {
		if vehicle.CarID <= 0 {
			continue
		}

		zero := time.Time{}
		if vehicle.LastModified.Equal(zero) {
			vehicle.LastModified = time.Now().UTC()
		}

		data, err := json.MarshalIndent(vehicle, "", "  ")
		if err != nil {
			return written, fmt.Errorf("marshalling vehicle %d: %w", vehicle.CarID, err)
		}

		data = append(data, '\n')

		filename := filepath.Join(outputDir, strconv.Itoa(vehicle.CarID)+".json")

		err = os.WriteFile(filename, data, 0o644) //nolint:gosec // strong permissions not needed for data files
		if err != nil {
			return written, fmt.Errorf("writing %s: %w", filename, err)
		}

		written++
	}

	return written, nil
}

// vehicleManifestEntry holds per-vehicle metadata in the manifest.
type vehicleManifestEntry struct {
	LastModified time.Time `json:"lastModified"`
}

// vehicleManifest is the structure written to manifest.json.
type vehicleManifest struct {
	Vehicles map[string]vehicleManifestEntry `json:"vehicles"`
}

// buildManifestJSON generates manifest JSON from a vehicle map and returns the encoded bytes.
func buildManifestJSON(vehicleMap map[string]vehicles.Vehicle) ([]byte, error) {
	entries := make(map[string]vehicleManifestEntry, len(vehicleMap))

	for _, vehicle := range vehicleMap {
		if vehicle.CarID <= 0 {
			continue
		}

		key := strconv.Itoa(vehicle.CarID)
		entries[key] = vehicleManifestEntry{LastModified: vehicle.LastModified.UTC()}
	}

	// Sort keys for stable output
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}

	slices.SortFunc(keys, func(a, b string) int {
		ia, _ := strconv.Atoi(a)
		ib, _ := strconv.Atoi(b)

		return ia - ib
	})

	sortedEntries := make(map[string]vehicleManifestEntry, len(entries))
	for _, k := range keys {
		sortedEntries[k] = entries[k]
	}

	manifest := vehicleManifest{
		Vehicles: sortedEntries,
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling manifest: %w", err)
	}

	data = append(data, '\n')

	return data, nil
}
