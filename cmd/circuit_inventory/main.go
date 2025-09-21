package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gtcircuits "github.com/zetetos/gt-telemetry/pkg/circuits"
	gtmodels "github.com/zetetos/gt-telemetry/pkg/models"
)

type CircuitCoordinates struct {
	Circuit      []gtmodels.CoordinateNorm `json:"circuit"`
	StartingLine gtmodels.CoordinateNorm   `json:"starting_line"`
}

type CircuitData struct {
	Name         string             `json:"name"`
	LengthMeters int                `json:"length_meters"`
	Coordinates  CircuitCoordinates `json:"coordinates"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <circuits_directory> <output_file>\n", os.Args[0])
		os.Exit(1)
	}

	circuitsDir := os.Args[1]
	outputFile := os.Args[2]

	coordinateMap := make(map[string][]string)
	circuitsMap := make(map[string]map[string]interface{})
	circuitStartLines := make(map[string]gtmodels.CoordinateNorm) // Store starting lines for analysis

	err := filepath.Walk(circuitsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}
		// Ignore the output file if it exists in the same dir
		if strings.HasSuffix(info.Name(), "circuit_inventory.json") || strings.HasSuffix(info.Name(), filepath.Base(outputFile)) {
			return nil
		}

		// region is the parent directory name under circuits
		rel, _ := filepath.Rel(circuitsDir, path)
		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) < 2 {
			return nil
		}
		region := parts[0]
		circuitID := strings.TrimSuffix(parts[1], ".json")

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read %s: %v\n", path, err)
			return nil
		}
		var circuitData CircuitData
		if err := json.Unmarshal(data, &circuitData); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse %s: %v\n", path, err)
			return nil
		}

		// 1. Coordinate map
		for _, coordinate := range circuitData.Coordinates.Circuit {
			key := gtcircuits.CoordinateNormToKey(coordinate)
			coordinateMap[key] = append(coordinateMap[key], circuitID)
		}

		// Store starting line for analysis
		circuitStartLines[circuitID] = circuitData.Coordinates.StartingLine

		// 3. Circuit info map
		circuitsMap[circuitID] = map[string]interface{}{
			"id":        circuitID,
			"name":      circuitData.Name,
			"region":    region,
			"length":    uint16(circuitData.LengthMeters),
			"startline": circuitData.Coordinates.StartingLine,
		}

		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking circuits dir: %v\n", err)
		os.Exit(1)
	}

	// Analysis: Find circuits without unique coordinates
	fmt.Println("\n=== ANALYSIS: Circuit Coordinate Uniqueness ===")

	type CircuitStats struct {
		ID              string
		Name            string
		Region          string
		TotalCoords     int
		UniqueCoords    int
		UniquePercent   float64
		StartLineUnique bool
	}

	var circuitStats []CircuitStats
	circuitsWithoutUniqueCoords := make([]string, 0)

	// Build a map of circuit -> coordinates for easier counting
	circuitCoordinates := make(map[string][]string)
	for coord, circuitList := range coordinateMap {
		for _, circuitID := range circuitList {
			circuitCoordinates[circuitID] = append(circuitCoordinates[circuitID], coord)
		}
	}

	// For each circuit, calculate uniqueness stats
	for circuitID := range circuitsMap {
		totalCoords := len(circuitCoordinates[circuitID])
		uniqueCoords := 0

		// Count unique coordinates for this circuit
		for _, coord := range circuitCoordinates[circuitID] {
			if len(coordinateMap[coord]) == 1 {
				uniqueCoords++
			}
		}

		var uniquePercent float64
		if totalCoords > 0 {
			uniquePercent = float64(uniqueCoords) / float64(totalCoords) * 100
		}

		circuitName := circuitsMap[circuitID]["name"].(string)
		region := circuitsMap[circuitID]["region"].(string)

		// Check if starting line coordinate is unique
		startLineCoord := circuitStartLines[circuitID]
		startLineUnique := true
		for otherCircuitID, otherStartLine := range circuitStartLines {
			if otherCircuitID != circuitID && otherStartLine == startLineCoord {
				startLineUnique = false
				break
			}
		}

		circuitStats = append(circuitStats, CircuitStats{
			ID:              circuitID,
			Name:            circuitName,
			Region:          region,
			TotalCoords:     totalCoords,
			UniqueCoords:    uniqueCoords,
			UniquePercent:   uniquePercent,
			StartLineUnique: startLineUnique,
		})

		if uniqueCoords == 0 {
			circuitsWithoutUniqueCoords = append(circuitsWithoutUniqueCoords, circuitID)
		}
	}

	// Sort by circuit name (alphabetically)
	for i := 0; i < len(circuitStats)-1; i++ {
		for j := i + 1; j < len(circuitStats); j++ {
			if circuitStats[i].Name > circuitStats[j].Name {
				circuitStats[i], circuitStats[j] = circuitStats[j], circuitStats[i]
			}
		}
	}

	// Calculate the maximum circuit name length for proper column alignment
	maxCircuitNameLen := 0
	for _, stats := range circuitStats {
		if len(stats.Name) > maxCircuitNameLen {
			maxCircuitNameLen = len(stats.Name)
		}
	}
	// Add some padding for the column
	circuitNameColWidth := maxCircuitNameLen + 2

	// Display results
	headerFormat := fmt.Sprintf("%%-%ds %%-%ds %%8s %%8s %%8s %%10s\n", circuitNameColWidth, 12)
	separatorFormat := fmt.Sprintf("%%-%ds %%-%ds %%8s %%8s %%8s %%10s\n", circuitNameColWidth, 12)
	dataFormat := fmt.Sprintf("%%s%%-%ds %%-%ds %%8d %%8d %%7.1f%%%% %%8s\n", circuitNameColWidth-2, 12) // -2 for marker space

	fmt.Printf(headerFormat, "Circuit Name", "Region", "Total", "Unique", "% Unique", "Start Uniq")
	fmt.Printf(separatorFormat, strings.Repeat("-", circuitNameColWidth), strings.Repeat("-", 12), "--------", "--------", "--------", "----------")

	for _, stats := range circuitStats {
		marker := "  "
		if stats.UniqueCoords == 0 {
			marker = "⚠️ "
		}

		startLineMarker := "✅"
		if !stats.StartLineUnique {
			startLineMarker = "❌"
		}

		fmt.Printf(dataFormat,
			marker,
			stats.Name,
			stats.Region,
			stats.TotalCoords,
			stats.UniqueCoords,
			stats.UniquePercent,
			startLineMarker)
	}

	if len(circuitsWithoutUniqueCoords) > 0 {
		fmt.Printf("\n⚠️  %d circuits have ZERO unique coordinates and are completely composed of shared coordinates.\n", len(circuitsWithoutUniqueCoords))
	} else {
		fmt.Println("\n✅ All circuits have at least one unique coordinate.")
	}

	// Count circuits with non-unique starting lines
	nonUniqueStartLines := 0
	for _, stats := range circuitStats {
		if !stats.StartLineUnique {
			nonUniqueStartLines++
		}
	}

	if nonUniqueStartLines > 0 {
		fmt.Printf("❌ %d circuits have non-unique starting line positions.\n", nonUniqueStartLines)
	} else {
		fmt.Println("✅ All circuits have unique starting line positions.")
	}

	out := map[string]interface{}{
		"coordinates": coordinateMap,
		"circuits":    circuitsMap,
	}

	outData, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal output: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputFile, outData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote inventory to %s\n", outputFile)
}
