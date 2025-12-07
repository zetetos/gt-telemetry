package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema/v5"
	gtcircuits "github.com/zetetos/gt-telemetry/pkg/circuits"
	gtmodels "github.com/zetetos/gt-telemetry/pkg/models"
)

type CircuitCoordinates struct {
	Circuit      []gtmodels.Coordinate `json:"Circuit"`
	StartingLine gtmodels.Coordinate   `json:"StartingLine"`
}

type CircuitData struct {
	Schema        string             `json:"$schema"`
	Name          string             `json:"Name"`
	VariationName string             `json:"VariationName"`
	Default       bool               `json:"Default"`
	Country       string             `json:"Country"`
	LengthMeters  int                `json:"LengthMeters"`
	Coordinates   CircuitCoordinates `json:"Coordinates"`
}

// CircuitProcessingResult holds the results of processing circuit files during analysis.
type CircuitProcessingResult struct {
	CoordinateMap       map[string][]string
	CircuitsMap         map[string]map[string]any
	CircuitStartLines   map[string]gtmodels.CoordinateNorm
	RawCoordinateCounts map[string]int // Track raw coordinate counts per circuit
}

// CircuitStats holds statistical information about a circuit.
type CircuitStats struct {
	ID                    string
	VariationName         string
	Country               string
	RawCoordinates        int
	NormalizedCoordinates int
	UniquePoints          int
	UniquePercent         float64
	StartLineUnique       bool
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <circuits_directory> <output_file>\n", os.Args[0])
		os.Exit(1)
	}

	circuitsDir := os.Args[1]
	outputFile := os.Args[2]

	// Load and compile the JSON schema
	schema, err := loadCircuitSchema(circuitsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading schema: %v\n", err)
		os.Exit(1)
	}

	// Process all circuit files
	processed, err := processCircuitFiles(circuitsDir, outputFile, schema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing circuits: %v\n", err)
		os.Exit(1)
	}

	// Analyze circuit coordinates
	stats := analyzeCircuitCoordinates(processed)

	// Display analysis results
	displayAnalysisResults(stats)

	// Write output file
	err = writeInventoryFile(processed, outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote inventory to %s\n", outputFile)
}

// loadCircuitSchema loads and compiles the JSON schema for circuit validation.
func loadCircuitSchema(circuitsDir string) (*jsonschema.Schema, error) {
	schemaPath := filepath.Join(circuitsDir, "schema", "circuit-schema.json")

	// Check if schema file exists
	_, err := os.Stat(schemaPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("schema file not found at %s: %w", schemaPath, os.ErrNotExist)
	}

	// Read schema file
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Compile schema
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	err = compiler.AddResource("schema.json", strings.NewReader(string(schemaData)))
	if err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return schema, nil
}

// processCircuitFiles reads and processes all circuit JSON files in the given directory.
func processCircuitFiles(circuitsDir, outputFile string, schema *jsonschema.Schema) (*CircuitProcessingResult, error) {
	processed := &CircuitProcessingResult{
		CoordinateMap:       make(map[string][]string),
		CircuitsMap:         make(map[string]map[string]any),
		CircuitStartLines:   make(map[string]gtmodels.CoordinateNorm),
		RawCoordinateCounts: make(map[string]int),
	}

	err := filepath.Walk(circuitsDir, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".json") {
			return nil
		}

		// Ignore the schema file and output file if present
		if strings.HasSuffix(fileInfo.Name(), "circuit-schema.json") || strings.HasSuffix(fileInfo.Name(), filepath.Base(outputFile)) {
			return nil
		}

		return processSingleCircuitFile(path, processed, schema)
	})

	return processed, err
}

// processSingleCircuitFile processes a single circuit JSON file and updates the processed result.
func processSingleCircuitFile(path string, processed *CircuitProcessingResult, schema *jsonschema.Schema) error {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read %s: %v\n", path, err)

		return nil
	}

	var circuitData CircuitData

	err = json.Unmarshal(data, &circuitData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse %s: %v\n", path, err)

		return nil
	}

	// Validate against JSON schema
	var jsonData interface{}

	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse JSON for validation %s: %v\n", path, err)

		return nil
	}

	err = schema.Validate(jsonData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Schema validation failed for %s: %v\n", path, err)

		return nil
	}

	circuitID := nameToID(circuitData.VariationName)

	// Track raw coordinate count
	processed.RawCoordinateCounts[circuitID] = len(circuitData.Coordinates.Circuit)

	// Build coordinate map
	for _, coordinate := range circuitData.Coordinates.Circuit {
		coordinateNorm := gtcircuits.NormaliseCircuitCoordinate(coordinate)
		key := coordinateNorm.String()

		if !slices.Contains(processed.CoordinateMap[key], circuitID) {
			processed.CoordinateMap[key] = append(processed.CoordinateMap[key], circuitID)
		}
	}

	// Store starting line for analysis
	startingLineNorm := gtcircuits.NormaliseStartLineCoordinate(circuitData.Coordinates.StartingLine)
	processed.CircuitStartLines[circuitID] = startingLineNorm

	// Store circuit info
	processed.CircuitsMap[circuitID] = map[string]any{
		"id":        circuitID,
		"name":      circuitData.Name,
		"variation": circuitData.VariationName,
		"default":   circuitData.Default,
		"country":   circuitData.Country,
		"length":    uint16(circuitData.LengthMeters), //nolint:gosec // Length will always be positive and less than max uint16
		"startline": startingLineNorm,
	}

	return nil
}

// analyzeCircuitCoordinates performs statistical analysis on circuit coordinates.
func analyzeCircuitCoordinates(processed *CircuitProcessingResult) (stats []CircuitStats) {
	// Build a map of circuit -> coordinates for easier counting
	circuitCoordinates := make(map[string][]string)

	for coord, circuitList := range processed.CoordinateMap {
		for _, circuitID := range circuitList {
			circuitCoordinates[circuitID] = append(circuitCoordinates[circuitID], coord)
		}
	}

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

		variationName, ok := processed.CircuitsMap[circuitID]["variation"].(string) //nolint:varnamelen // idiomatic name
		if !ok {
			variationName = ""
		}

		country, ok := processed.CircuitsMap[circuitID]["country"].(string)
		if !ok {
			country = ""
		}

		// Check if starting line coordinate is unique
		startLineCoord := processed.CircuitStartLines[circuitID]
		startLineUnique := isStartLineUnique(circuitID, startLineCoord, processed.CircuitStartLines)

		rawCoords := processed.RawCoordinateCounts[circuitID]

		stats = append(stats, CircuitStats{
			ID:                    circuitID,
			VariationName:         variationName,
			Country:               country,
			RawCoordinates:        rawCoords,
			NormalizedCoordinates: totalCoords,
			UniquePoints:          uniqueCoords,
			UniquePercent:         uniquePercent,
			StartLineUnique:       startLineUnique,
		})
	}

	sortStatsByVariationName(stats)

	return stats
}

// isStartLineUnique checks if a circuit's starting line coordinate is unique.
func isStartLineUnique(circuitID string, startLineCoord gtmodels.CoordinateNorm, allStartLines map[string]gtmodels.CoordinateNorm) bool {
	for otherCircuitID, otherStartLine := range allStartLines {
		if otherCircuitID != circuitID && otherStartLine == startLineCoord {
			return false
		}
	}

	return true
}

// sortStatsByVariationName sorts circuit stats alphabetically by name.
func sortStatsByVariationName(stats []CircuitStats) {
	for i := range len(stats) - 1 {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].VariationName > stats[j].VariationName {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
}

// displayAnalysisResults prints the analysis results in a formatted table.
func displayAnalysisResults(stats []CircuitStats) {
	fmt.Println("\n=== ANALYSIS: Circuit Coordinate Uniqueness ===")

	// Align columns based on longest circuit name
	maxCircuitNameLen := 0
	for _, stat := range stats {
		if len(stat.VariationName) > maxCircuitNameLen {
			maxCircuitNameLen = len(stat.VariationName)
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

// printTableHeader prints the table header.
func printTableHeader(circuitNameColWidth int) {
	headerFormat := fmt.Sprintf("%%-%ds %%-%ds %%8s %%8s %%8s %%8s %%10s\n", circuitNameColWidth, 12)
	fmt.Printf(headerFormat, "Circuit Name", "Country", "Raw", "Norm", "Unique", "% Unique", "Start Uniq")
}

// printTableSeparator prints the table separator line.
func printTableSeparator(circuitNameColWidth int) {
	separatorFormat := fmt.Sprintf("%%-%ds %%-%ds %%8s %%8s %%8s %%8s %%10s\n", circuitNameColWidth, 12)
	fmt.Printf(separatorFormat,
		strings.Repeat("-", circuitNameColWidth),
		strings.Repeat("-", 12),
		"--------",
		"--------",
		"--------",
		"--------",
		"----------")
}

// printStatRow prints a single statistics row.
func printStatRow(stat CircuitStats, circuitNameColWidth int) {
	dataFormat := fmt.Sprintf("%%s%%-%ds %%-%ds %%8d %%8d %%8d %%7.1f%%%% %%8s\n", circuitNameColWidth-2, 12) // -2 for marker space

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
		stat.VariationName,
		stat.Country,
		stat.RawCoordinates,
		stat.NormalizedCoordinates,
		stat.UniquePoints,
		stat.UniquePercent,
		startLineMarker)
}

// printSummary prints the analysis summary.
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

// writeInventoryFile writes the processed circuit data to the output file.
func writeInventoryFile(processed *CircuitProcessingResult, outputFile string) error {
	// Filter coordinates to only include those with a single entry (unique to one circuit)
	// Store as single string values instead of arrays
	filteredCoordinateMap := make(map[string]string)

	for coord, circuits := range processed.CoordinateMap {
		if len(circuits) == 1 {
			filteredCoordinateMap[coord] = circuits[0]
		}
	}

	// Convert processed circuit maps to CircuitInfo structs
	circuits := make(map[string]gtcircuits.CircuitInfo)
	for circuitID, circuitData := range processed.CircuitsMap {
		circuits[circuitID] = gtcircuits.CircuitInfo{ //nolint:forcetypeassert // Safe due to controlled data source
			ID:                    circuitData["id"].(string),
			Name:                  circuitData["name"].(string),
			Variation:             circuitData["variation"].(string),
			Default:               circuitData["default"].(bool),
			Country:               circuitData["country"].(string),
			Length:                int(circuitData["length"].(uint16)),
			StartLine:             circuitData["startline"].(gtmodels.CoordinateNorm),
			UniqueCoordinateCount: getUniqCoordCount(processed, circuitID),
		}
	}

	inventory := gtcircuits.CircuitInventory{
		Coordinates: filteredCoordinateMap,
		Circuits:    circuits,
	}

	outData, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	err = os.WriteFile(outputFile, outData, 0644) //nolint:gosec // File permission is acceptable for this use case
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func getUniqCoordCount(processed *CircuitProcessingResult, id string) int {
	uniqueCoordCount := 0

	for _, circuitList := range processed.CoordinateMap {
		if len(circuitList) == 1 && circuitList[0] == id {
			uniqueCoordCount++
		}
	}

	return uniqueCoordCount
}

// nameToID converts a circuit name to an ID using Go Pascal case.
func nameToID(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var result strings.Builder

	for _, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			result.WriteRune(unicode.ToUpper(runes[0]))

			for i := 1; i < len(runes); i++ {
				result.WriteRune(unicode.ToLower(runes[i]))
			}
		}
	}

	return result.String()
}
