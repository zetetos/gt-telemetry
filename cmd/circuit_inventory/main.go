package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	gtcircuits "github.com/zetetos/gt-telemetry/pkg/circuits"
	gtmodels "github.com/zetetos/gt-telemetry/pkg/models"
)

type CircuitCoordinates struct {
	Circuit      []gtmodels.CoordinateNorm `json:"circuit"`
	StartingLine gtmodels.CoordinateNorm   `json:"starting_line"`
}

type CircuitData struct {
	Name         string             `json:"name"`
	Region       string             `json:"region"`
	LengthMeters int                `json:"length_meters"`
	Coordinates  CircuitCoordinates `json:"coordinates"`
}

// ProcessedCircuits holds the results of processing circuit files
type ProcessedCircuits struct {
	CoordinateMap     map[string][]string
	CircuitsMap       map[string]map[string]interface{}
	CircuitStartLines map[string]gtmodels.CoordinateNorm
}

// CircuitStats holds statistical information about a circuit
type CircuitStats struct {
	ID              string
	Name            string
	Region          string
	TotalPoints     int
	UniquePoints    int
	UniquePercent   float64
	StartLineUnique bool
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <circuits_directory> <output_file>\n", os.Args[0])
		os.Exit(1)
	}

	circuitsDir := os.Args[1]
	outputFile := os.Args[2]

	// Process all circuit files
	processed, err := processCircuitFiles(circuitsDir, outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing circuits: %v\n", err)
		os.Exit(1)
	}

	// Analyze circuit coordinates
	stats := analyzeCircuitCoordinates(processed)

	// Display analysis results
	displayAnalysisResults(stats)

	// Write output file
	if err := writeInventoryFile(processed, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote inventory to %s\n", outputFile)
}

// processCircuitFiles reads and processes all circuit JSON files in the given directory
func processCircuitFiles(circuitsDir, outputFile string) (*ProcessedCircuits, error) {
	processed := &ProcessedCircuits{
		CoordinateMap:     make(map[string][]string),
		CircuitsMap:       make(map[string]map[string]interface{}),
		CircuitStartLines: make(map[string]gtmodels.CoordinateNorm),
	}

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

		circuitID := nameToID(circuitData.Name)

		// Build coordinate map
		for _, coordinate := range circuitData.Coordinates.Circuit {
			key := gtcircuits.CoordinateNormToKey(coordinate)
			processed.CoordinateMap[key] = append(processed.CoordinateMap[key], circuitID)
		}

		// Store starting line for analysis
		processed.CircuitStartLines[circuitID] = circuitData.Coordinates.StartingLine

		// Store circuit info
		processed.CircuitsMap[circuitID] = map[string]interface{}{
			"id":        circuitID,
			"name":      circuitData.Name,
			"region":    circuitData.Region,
			"length":    uint16(circuitData.LengthMeters),
			"startline": circuitData.Coordinates.StartingLine,
		}

		return nil
	})

	return processed, err
}

// analyzeCircuitCoordinates performs statistical analysis on circuit coordinates
func analyzeCircuitCoordinates(processed *ProcessedCircuits) []CircuitStats {
	// Build a map of circuit -> coordinates for easier counting
	circuitCoordinates := make(map[string][]string)
	for coord, circuitList := range processed.CoordinateMap {
		for _, circuitID := range circuitList {
			circuitCoordinates[circuitID] = append(circuitCoordinates[circuitID], coord)
		}
	}

	var stats []CircuitStats

	// For each circuit, calculate uniqueness stats
	for circuitID := range processed.CircuitsMap {
		totalCoords := len(circuitCoordinates[circuitID])
		uniqueCoords := 0

		// Count unique coordinates for this circuit
		for _, coord := range circuitCoordinates[circuitID] {
			if len(processed.CoordinateMap[coord]) == 1 {
				uniqueCoords++
			}
		}

		var uniquePercent float64
		if totalCoords > 0 {
			uniquePercent = float64(uniqueCoords) / float64(totalCoords) * 100
		}

		circuitName := processed.CircuitsMap[circuitID]["name"].(string)
		region := processed.CircuitsMap[circuitID]["region"].(string)

		// Check if starting line coordinate is unique
		startLineCoord := processed.CircuitStartLines[circuitID]
		startLineUnique := isStartLineUnique(circuitID, startLineCoord, processed.CircuitStartLines)

		stats = append(stats, CircuitStats{
			ID:              circuitID,
			Name:            circuitName,
			Region:          region,
			TotalPoints:     totalCoords,
			UniquePoints:    uniqueCoords,
			UniquePercent:   uniquePercent,
			StartLineUnique: startLineUnique,
		})
	}

	sortStatsByName(stats)

	return stats
}

// isStartLineUnique checks if a circuit's starting line coordinate is unique
func isStartLineUnique(circuitID string, startLineCoord gtmodels.CoordinateNorm, allStartLines map[string]gtmodels.CoordinateNorm) bool {
	for otherCircuitID, otherStartLine := range allStartLines {
		if otherCircuitID != circuitID && otherStartLine == startLineCoord {
			return false
		}
	}
	return true
}

// sortStatsByName sorts circuit stats alphabetically by name
func sortStatsByName(stats []CircuitStats) {
	for i := 0; i < len(stats)-1; i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].Name > stats[j].Name {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
}

// displayAnalysisResults prints the analysis results in a formatted table
func displayAnalysisResults(stats []CircuitStats) {
	fmt.Println("\n=== ANALYSIS: Circuit Coordinate Uniqueness ===")

	// Align columns based on longest circuit name
	maxCircuitNameLen := 0
	for _, stat := range stats {
		if len(stat.Name) > maxCircuitNameLen {
			maxCircuitNameLen = len(stat.Name)
		}
	}

	// Column padding
	circuitNameColWidth := maxCircuitNameLen + 2

	// Display table
	printTableHeader(circuitNameColWidth)
	printTableSeparator(circuitNameColWidth)

	circuitsWithoutUniqueCoords := 0
	nonUniqueStartLines := 0

	for _, stat := range stats {
		printStatRow(stat, circuitNameColWidth)

		if stat.UniquePoints == 0 {
			circuitsWithoutUniqueCoords++
		}
		if !stat.StartLineUnique {
			nonUniqueStartLines++
		}
	}

	printSummary(circuitsWithoutUniqueCoords, nonUniqueStartLines)
}

// printTableHeader prints the table header
func printTableHeader(circuitNameColWidth int) {
	headerFormat := fmt.Sprintf("%%-%ds %%-%ds %%8s %%8s %%8s %%10s\n", circuitNameColWidth, 12)
	fmt.Printf(headerFormat, "Circuit Name", "Region", "Total", "Unique", "% Unique", "Start Uniq")
}

// printTableSeparator prints the table separator line
func printTableSeparator(circuitNameColWidth int) {
	separatorFormat := fmt.Sprintf("%%-%ds %%-%ds %%8s %%8s %%8s %%10s\n", circuitNameColWidth, 12)
	fmt.Printf(separatorFormat,
		strings.Repeat("-", circuitNameColWidth),
		strings.Repeat("-", 12),
		"--------",
		"--------",
		"--------",
		"----------")
}

// printStatRow prints a single statistics row
func printStatRow(stat CircuitStats, circuitNameColWidth int) {
	dataFormat := fmt.Sprintf("%%s%%-%ds %%-%ds %%8d %%8d %%7.1f%%%% %%8s\n", circuitNameColWidth-2, 12) // -2 for marker space

	marker := "  "
	if stat.UniquePoints == 0 {
		marker = "⚠️ "
	}

	startLineMarker := "✅"
	if !stat.StartLineUnique {
		startLineMarker = "❌"
	}

	fmt.Printf(dataFormat,
		marker,
		stat.Name,
		stat.Region,
		stat.TotalPoints,
		stat.UniquePoints,
		stat.UniquePercent,
		startLineMarker)
}

// printSummary prints the analysis summary
func printSummary(circuitsWithoutUniqueCoords, nonUniqueStartLines int) {
	if circuitsWithoutUniqueCoords > 0 {
		fmt.Printf("\n⚠️  %d circuits have ZERO unique coordinates and are completely composed of shared coordinates.\n", circuitsWithoutUniqueCoords)
	} else {
		fmt.Println("\n✅ All circuits have at least one unique coordinate.")
	}

	if nonUniqueStartLines > 0 {
		fmt.Printf("❌ %d circuits have non-unique starting line positions.\n", nonUniqueStartLines)
	} else {
		fmt.Println("✅ All circuits have unique starting line positions.")
	}
}

// writeInventoryFile writes the processed circuit data to the output file
func writeInventoryFile(processed *ProcessedCircuits, outputFile string) error {
	out := map[string]interface{}{
		"coordinates": processed.CoordinateMap,
		"circuits":    processed.CircuitsMap,
	}

	outData, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %v", err)
	}

	if err := os.WriteFile(outputFile, outData, 0644); err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}

	return nil
}

// nameToID converts a circuit name to an ID
func nameToID(name string) string {
	circuitID := strings.ToLower(name)
	circuitID = strings.ReplaceAll(circuitID, " ", "_")
	circuitID = strings.ReplaceAll(circuitID, "-", "_")
	circuitID = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return -1
	}, circuitID)

	// Remove multiple consecutive underscores
	for strings.Contains(circuitID, "__") {
		circuitID = strings.ReplaceAll(circuitID, "__", "_")
	}

	// Remove leading and trailing underscores
	circuitID = strings.Trim(circuitID, "_")

	return circuitID
}
