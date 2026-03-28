package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema/v5"
	gtcircuits "github.com/zetetos/gt-telemetry/v2/pkg/circuits"
	gtmodels "github.com/zetetos/gt-telemetry/v2/pkg/models"
)

type CircuitCoordinates struct {
	Circuit      []gtmodels.Coordinate `json:"circuit"`
	StartingLine gtmodels.Coordinate   `json:"startingLine"`
}

type CircuitData struct {
	Schema        string             `json:"$schema"`
	Name          string             `json:"name"`
	VariationName string             `json:"variationName"`
	Default       bool               `json:"default"`
	Country       string             `json:"country"`
	LengthMetres  int                `json:"lengthMetres"`
	LastModified  string             `json:"lastModified"`
	Coordinates   CircuitCoordinates `json:"coordinates"`
}

// CircuitProcessingResult holds the results of processing circuit files during analysis.
type CircuitProcessingResult struct {
	CoordinateMap          map[string][]string
	CircuitsMap            map[string]map[string]any
	CircuitStartLines      map[string]gtmodels.CoordinateNorm
	CircuitCoordinatesNorm map[string][]gtmodels.CoordinateNorm
	RawCoordinateCounts    map[string]int // Track raw coordinate counts per circuit
	LastModified           time.Time      // Most recent modification time of any source circuit file
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

const usage = `Usage:
  %[1]s update   <circuits_directory> <output_directory>   Process circuit files and write inventory
  %[1]s manifest <inventory_directory>                     Generate manifest JSON from inventory (stdout)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		os.Exit(1)
	}

	switch os.Args[1] {
	case "update":
		if len(os.Args) != 4 {
			fmt.Fprintf(os.Stderr, usage, os.Args[0])
			os.Exit(1)
		}

		runUpdate(os.Args[2], os.Args[3])
	case "manifest":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, usage, os.Args[0])
			os.Exit(1)
		}

		runManifest(os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s'. Supported actions: update, manifest\n\n", os.Args[1])
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		os.Exit(1)
	}
}

// runUpdate processes circuit source files and writes per-circuit inventory files.
func runUpdate(circuitsDir, outputDir string) {
	// Process all circuit files
	processed, err := processCircuitFiles(circuitsDir, outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing circuits: %v\n", err)
		os.Exit(1)
	}

	// Analyze circuit coordinates
	stats := analyzeCircuitCoordinates(processed)

	// Display analysis results
	displayAnalysisResults(stats)

	// Write per-circuit inventory files
	inventoryDir := filepath.Join(filepath.Dir(outputDir), "inventory")

	count, err := writeCircuitInventoryFiles(processed, inventoryDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write circuit inventory files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %d circuit files to %s\n", count, inventoryDir)
}

// runManifest loads an inventory directory and writes manifest JSON to stdout.
func runManifest(inventoryDir string) {
	entries, err := os.ReadDir(inventoryDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read inventory directory: %v\n", err)
		os.Exit(1)
	}

	processed := &CircuitProcessingResult{
		CircuitsMap: make(map[string]map[string]any),
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || entry.Name() == "manifest.json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(inventoryDir, entry.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read %s: %v\n", entry.Name(), err)
			os.Exit(1)
		}

		var circuit gtcircuits.CircuitInfo

		err = json.Unmarshal(data, &circuit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse %s: %v\n", entry.Name(), err)
			os.Exit(1)
		}

		circuitID := strings.TrimSuffix(entry.Name(), ".json")
		processed.CircuitsMap[circuitID] = map[string]any{
			"lastModified": circuit.LastModified,
		}

		if circuit.LastModified.After(processed.LastModified) {
			processed.LastModified = circuit.LastModified
		}
	}

	out, err := buildCircuitManifestJSON(processed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(string(out))
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

// processCircuitFiles reads all circuit JSON files in the given directory and compiles them into a single structured object.
func processCircuitFiles(circuitsDir, outputFile string) (*CircuitProcessingResult, error) {
	// Load and compile the JSON schema
	schema, err := loadCircuitSchema(circuitsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading schema: %v\n", err)
		os.Exit(1)
	}

	processed := &CircuitProcessingResult{
		CoordinateMap:          make(map[string][]string),
		CircuitsMap:            make(map[string]map[string]any),
		CircuitStartLines:      make(map[string]gtmodels.CoordinateNorm),
		CircuitCoordinatesNorm: make(map[string][]gtmodels.CoordinateNorm),
		RawCoordinateCounts:    make(map[string]int),
	}

	err = filepath.Walk(circuitsDir, func(path string, fileInfo os.FileInfo, err error) error {
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
	jsonContent, err := readCircuitFile(path, schema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read %s: %v\n", path, err)

		return nil
	}

	var circuitData CircuitData

	err = json.Unmarshal(jsonContent, &circuitData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse %s: %v\n", path, err)

		return nil
	}

	circuitID := nameToID(circuitData.VariationName)

	circuitLastModified, err := time.Parse(time.RFC3339, circuitData.LastModified)
	if err != nil {
		return fmt.Errorf("failed to parse lastModified for %s: %w", path, err)
	}

	// Track most recent lastModified value across all circuit files
	if circuitLastModified.After(processed.LastModified) {
		processed.LastModified = circuitLastModified
	}

	processed.RawCoordinateCounts[circuitID] = len(circuitData.Coordinates.Circuit)

	processed.CircuitCoordinatesNorm[circuitID] = []gtmodels.CoordinateNorm{}
	lastCoordinateNorm := gtmodels.CoordinateNorm{}

	// Build coordinate map
	for _, coordinate := range circuitData.Coordinates.Circuit {
		coordinateNorm := gtcircuits.NormaliseCircuitCoordinate(coordinate)

		if coordinateNorm != lastCoordinateNorm {
			processed.CircuitCoordinatesNorm[circuitID] = append(processed.CircuitCoordinatesNorm[circuitID], coordinateNorm)
			lastCoordinateNorm = coordinateNorm
		}

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
		"id":           circuitID,
		"name":         circuitData.Name,
		"variation":    circuitData.VariationName,
		"default":      circuitData.Default,
		"country":      circuitData.Country,
		"length":       uint16(circuitData.LengthMetres), //nolint:gosec // Length will always be positive and less than max uint16
		"startline":    startingLineNorm,
		"lastModified": circuitLastModified,
		"coordinates":  processed.CircuitCoordinatesNorm[circuitID],
	}

	return nil
}

// readCircuitFile reads and validates a circuit JSON file, returning the raw data when valid.
func readCircuitFile(path string, schema *jsonschema.Schema) ([]byte, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var jsonData any

	err = json.Unmarshal(fileContent, &jsonData)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to parse JSON for validation %s: %w", path, err)
	}

	err = schema.Validate(jsonData)
	if err != nil {
		return []byte{}, fmt.Errorf("schema validation failed for %s: %w", path, err)
	}

	return fileContent, nil
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

// writeCircuitInventoryFiles writes one JSON file per circuit into inventoryDir.
func writeCircuitInventoryFiles(processed *CircuitProcessingResult, inventoryDir string) (int, error) {
	err := os.MkdirAll(inventoryDir, 0o755)
	if err != nil {
		return 0, fmt.Errorf("failed to create inventory directory: %w", err)
	}

	count := 0

	for circuitID, circuitData := range processed.CircuitsMap {
		file := gtcircuits.CircuitInfo{ //nolint:forcetypeassert // Safe due to controlled data source
			ID:           circuitData["id"].(string),
			Name:         circuitData["name"].(string),
			Variation:    circuitData["variation"].(string),
			Country:      circuitData["country"].(string),
			Default:      circuitData["default"].(bool),
			Length:       int(circuitData["length"].(uint16)),
			StartLine:    circuitData["startline"].(gtmodels.CoordinateNorm),
			LastModified: circuitData["lastModified"].(time.Time),
			Coordinates:  circuitData["coordinates"].([]gtmodels.CoordinateNorm),
		}

		outData, err := marshalCircuitJSON(file)
		if err != nil {
			return count, fmt.Errorf("failed to marshal circuit %s: %w", circuitID, err)
		}

		outPath := filepath.Join(inventoryDir, circuitID+".json")

		err = os.WriteFile(outPath, outData, 0o644) //nolint:gosec // File permission is acceptable for this use case
		if err != nil {
			return count, fmt.Errorf("failed to write circuit file %s: %w", outPath, err)
		}

		count++
	}

	return count, nil
}

// coordObjectPattern matches a multi-line JSON object containing only x, y, z fields.
var coordObjectPattern = regexp.MustCompile(`\{\s*\n\s*"x":\s*(-?\d+),\s*\n\s*"y":\s*(-?\d+),\s*\n\s*"z":\s*(-?\d+)\s*\n\s*\}`)

// marshalCircuitJSON marshals a CircuitInfo to indented JSON with coordinate objects inlined.
func marshalCircuitJSON(file gtcircuits.CircuitInfo) ([]byte, error) {
	data, err := json.MarshalIndent(file, "", "    ")
	if err != nil {
		return nil, err
	}

	result := coordObjectPattern.ReplaceAllString(string(data), `{ "x": $1, "y": $2, "z": $3 }`)

	return []byte(result), nil
}

// circuitManifestEntry holds per-circuit metadata in the manifest.
type circuitManifestEntry struct {
	LastModified time.Time `json:"lastModified"`
}

// circuitManifest is the structure written to manifest.json.
type circuitManifest struct {
	Circuits map[string]circuitManifestEntry `json:"circuits"`
}

// buildCircuitManifestJSON generates manifest JSON from a CircuitProcessingResult and returns the encoded bytes.
func buildCircuitManifestJSON(processed *CircuitProcessingResult) ([]byte, error) {
	circuits := make(map[string]circuitManifestEntry, len(processed.CircuitsMap))

	for circuitID, circuitData := range processed.CircuitsMap {
		circuits[circuitID] = circuitManifestEntry{LastModified: circuitData["lastModified"].(time.Time)} //nolint:forcetypeassert // Safe due to controlled data source
	}

	manifest := circuitManifest{
		Circuits: circuits,
	}

	outData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	outData = append(outData, '\n')

	return outData, nil
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
