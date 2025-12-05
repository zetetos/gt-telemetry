package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/zetetos/gt-telemetry/pkg/vehicles"
)

var (
	ErrUnsupportedFormat          = errors.New("unsupported format")
	ErrInvalidCSVHeader           = errors.New("invalid CSV header: column count mismatch")
	ErrInvalidCSVRecord           = errors.New("invalid CSV record: column count mismatch")
	ErrCarIDRequired              = errors.New("CarID is required")
	ErrCarIDAlreadyExists         = errors.New("a vehicle with this CarID already exists")
	ErrVehicleNotFound            = errors.New("vehicle not found in inventory")
	ErrMainJSBundleNotFound       = errors.New("could not find main JS bundle in HTML")
	ErrCarsJSNotFound             = errors.New("could not find cars JS file in main bundle")
	ErrTunersJSNotFound           = errors.New("could not find tuners JS file in main bundle")
	ErrVariableNameNotFound       = errors.New("could not find variable name in JavaScript")
	ErrTunersVariableNameNotFound = errors.New("could not find variable name in tuners JavaScript")
	ErrCarsObjectNotFound         = errors.New("cars object not found in JavaScript")
	ErrTunersObjectNotFound       = errors.New("tuners object not found in JavaScript")
)

//nolint:dupword
const usage = `inventory - Import and export vehicle inventory data between JSON and CSV formats

Usage:
  inventory <action> [arguments]

Actions:
  convert <file.csv|file.json>   Convert between JSON and CSV formats
  add     <file.json>            Add a new vehicle entry interactively
  edit    <file.json> <car-id>   Edit an existing vehicle entry
  delete  <file.json> <car-id>   Delete a vehicle entry
  fetch   <file.json> [locale]   Fetch and merge car data from Gran Turismo website

Arguments:
  file                     Path to input file.
  car-id                   CarID of the vehicle to edit or delete
  locale                   Locale code for fetch (default: gb). Examples: gb, us, jp, au

Flags:
  -help                    Show this help message
  -no-color                Disable colored output
  -dry-run                 Show changes without modifying files

Output format is determined by input file extension:
  .json files are converted to CSV format
  .csv files are converted to JSON format

Examples:
  # Convert JSON to CSV
  inventory convert inventory.json > inventory.csv

  # Convert CSV to JSON
  inventory convert inventory.csv > inventory.json

  # Add a new vehicle entry to inventory
  inventory add internal/vehicles/inventory.json

  # Edit an existing vehicle
  inventory edit internal/vehicles/inventory.json 1234

  # Delete a vehicle
  inventory delete internal/vehicles/inventory.json 1234

  # Fetch and merge data from Gran Turismo website (default GB locale)
  inventory fetch pkg/vehicles/vehicles.json

  # Fetch and merge data for a specific locale
  inventory fetch pkg/vehicles/vehicles.json us
`

const pdNullValue = "---"

func main() {
	var (
		help    = flag.Bool("help", false, "Show help message")
		noColor = flag.Bool("no-color", false, "Disable colored output")
		dryRun  = flag.Bool("dry-run", false, "Show changes without modifying files")
	)

	flag.Parse()

	if *help {
		fmt.Print(usage)
		os.Exit(0)
	}

	// Get positional arguments
	args := flag.Args()

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Action is required\n\n")
		fmt.Print(usage)
		os.Exit(1)
	}

	action := args[0]

	switch action {
	case "convert":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: Input file argument is required for convert action\n\n")
			fmt.Print(usage)
			os.Exit(1)
		}

		inputFile := args[1]

		// Determine format from file extension or special cases
		var outputFormat string

		if inputFile == "/dev/stdin" || inputFile == "-" {
			fmt.Fprintf(os.Stderr, "Error: Cannot determine format from stdin. Please use a file with .json or .csv extension\n")
			os.Exit(1)
		}

		ext := strings.ToLower(filepath.Ext(inputFile))
		switch ext {
		case ".json":
			outputFormat = "csv"
		case ".csv":
			outputFormat = "json"
		default:
			fmt.Fprintf(os.Stderr, "Error: Unsupported file extension '%s'. Supported extensions: .json, .csv\n", ext)
			os.Exit(1)
		}

		err := convertFile(inputFile, outputFormat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "add":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: Inventory file argument is required for add action\n\n")
			fmt.Print(usage)
			os.Exit(1)
		}

		inventoryFile := args[1]

		err := addVehicleInteractive(inventoryFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding vehicle: %v\n", err)
			os.Exit(1)
		}

	case "edit":
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: Both inventory file and car ID arguments are required for edit action\n\n")
			fmt.Print(usage)
			os.Exit(1)
		}

		inventoryFile := args[1]
		carIDStr := args[2]

		carID, err := strconv.Atoi(carIDStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid car ID '%s': %v\n", carIDStr, err)
			os.Exit(1)
		}

		err = editVehicleInteractively(inventoryFile, carID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error editing vehicle: %v\n", err)
			os.Exit(1)
		}

	case "delete":
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: Both inventory file and car ID arguments are required for delete action\n\n")
			fmt.Print(usage)
			os.Exit(1)
		}

		inventoryFile := args[1]
		carIDStr := args[2]

		carID, err := strconv.Atoi(carIDStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid car ID '%s': %v\n", carIDStr, err)
			os.Exit(1)
		}

		err = deleteVehicle(inventoryFile, carID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting vehicle: %v\n", err)
			os.Exit(1)
		}

	case "fetch":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: Inventory file argument is required for fetch action\n\n")
			fmt.Print(usage)
			os.Exit(1)
		}

		inventoryFile := args[1]

		locale := "gb" // default locale
		if len(args) > 2 {
			locale = args[2]
		}

		err := fetchAndMergeGTData(inventoryFile, locale, *noColor, *dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching GT data: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s'. Supported actions: convert, add, edit, delete, fetch\n\n", action)
		fmt.Print(usage)
		os.Exit(1)
	}
}

func convertFile(inputFile, format string) error {
	switch format {
	case "csv":
		return jsonToCSV(inputFile)
	case "json":
		return csvToJSON(inputFile)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

func jsonToCSV(inputFile string) error {
	// Read JSON file
	jsonData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("reading JSON file: %w", err)
	}

	// Parse JSON into vehicle map
	var vehicleMap map[string]vehicles.Vehicle

	err = json.Unmarshal(jsonData, &vehicleMap)
	if err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	// Write to stdout
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write CSV header
	err = writer.Write(orderedVehicleFields())
	if err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	// Write vehicle data
	for _, vehicle := range vehicleMap {
		record := make([]string, len(orderedVehicleFields()))
		for i, fieldName := range orderedVehicleFields() {
			record[i] = getVehicleFieldValueAsString(vehicle, fieldName)
		}

		err = writer.Write(record)
		if err != nil {
			return fmt.Errorf("writing CSV record: %w", err)
		}
	}

	return nil
}

func csvToJSON(inputFile string) error {
	// Open CSV file
	inputF, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("opening CSV file: %w", err)
	}
	defer inputF.Close()

	csvReader := csv.NewReader(inputF)

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV header: %w", err)
	}

	// Validate header format
	if len(header) != len(orderedVehicleFields()) {
		return fmt.Errorf("%w: expected %d columns, got %d", ErrInvalidCSVHeader, len(orderedVehicleFields()), len(header))
	}

	// Create map to store vehicles
	vehicleMap := make(map[string]vehicles.Vehicle)

	// Read CSV records
	for {
		record, err := csvReader.Read()

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("reading CSV record: %w", err)
		}

		if len(record) != len(header) {
			return fmt.Errorf("%w: expected %d columns, got %d", ErrInvalidCSVRecord, len(header), len(record))
		}

		// Parse CarID
		carID, err := strconv.Atoi(record[0])
		if err != nil {
			return fmt.Errorf("parsing CarID '%s': %w", record[6], err)
		}

		// Parse year
		year, err := strconv.Atoi(record[3])
		if err != nil {
			// Handle empty or invalid year
			if record[3] == "" || record[3] == "-" {
				year = 0
			} else {
				return fmt.Errorf("parsing year '%s': %w", record[5], err)
			}
		}

		// Parse OpenCockpit
		openCockpit, err := strconv.ParseBool(record[4])
		if err != nil {
			return fmt.Errorf("parsing OpenCockpit '%s': %w", record[7], err)
		}

		// Parse Length
		var length int
		if record[9] != "" && record[9] != "-" {
			length, err = strconv.Atoi(record[9])
			if err != nil {
				return fmt.Errorf("parsing Length '%s': %w", record[9], err)
			}
		}

		// Parse Width
		var width int
		if record[10] != "" && record[10] != "-" {
			width, err = strconv.Atoi(record[10])
			if err != nil {
				return fmt.Errorf("parsing Width '%s': %w", record[10], err)
			}
		}

		// Parse Height
		var height int
		if record[11] != "" && record[11] != "-" {
			height, err = strconv.Atoi(record[11])
			if err != nil {
				return fmt.Errorf("parsing Height '%s': %w", record[11], err)
			}
		}

		// Parse Wheelbase
		var wheelbase int
		if record[12] != "" && record[12] != "-" {
			wheelbase, err = strconv.Atoi(record[12])
			if err != nil {
				return fmt.Errorf("parsing Wheelbase '%s': %w", record[12], err)
			}
		}

		// Parse TrackFront
		var trackFront int
		if record[13] != "" && record[13] != "-" {
			trackFront, err = strconv.Atoi(record[13])
			if err != nil {
				return fmt.Errorf("parsing TrackFront '%s': %w", record[13], err)
			}
		}

		// Parse TrackRear
		var trackRear int
		if record[14] != "" && record[14] != "-" {
			trackRear, err = strconv.Atoi(record[14])
			if err != nil {
				return fmt.Errorf("parsing TrackRear '%s': %w", record[14], err)
			}
		}

		// Parse EngineBankAngle
		var EngineBankAngle float32

		if record[16] != "" && record[16] != "-" {
			angle, err := strconv.ParseFloat(record[16], 32)
			if err != nil {
				return fmt.Errorf("parsing EngineBankAngle '%s': %w", record[16], err)
			}

			EngineBankAngle = float32(angle)
		}

		// Parse EngineCrankPlaneAngle
		var engineCrankPlaneAngle float32

		if record[17] != "" && record[17] != "-" {
			angle, err := strconv.ParseFloat(record[17], 32)
			if err != nil {
				return fmt.Errorf("parsing EngineCrankPlaneAngle '%s': %w", record[17], err)
			}

			engineCrankPlaneAngle = float32(angle)
		}

		vehicle := vehicles.Vehicle{
			CarID:                 carID,
			Manufacturer:          record[1],
			Model:                 record[2],
			Year:                  year,
			OpenCockpit:           openCockpit,
			CarType:               record[5],
			Category:              record[6],
			Drivetrain:            record[7],
			Aspiration:            record[8],
			Length:                length,
			Width:                 width,
			Height:                height,
			Wheelbase:             wheelbase,
			TrackFront:            trackFront,
			TrackRear:             trackRear,
			EngineLayout:          record[15],
			EngineBankAngle:       EngineBankAngle,
			EngineCrankPlaneAngle: engineCrankPlaneAngle,
		}

		// Use CarID as the key (converted to string)
		vehicleMap[strconv.Itoa(carID)] = vehicle
	}

	// Write JSON to stdout
	err = writeOrderedJSON(os.Stdout, vehicleMap)
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	return nil
}

// writeOrderedJSON writes a vehicle map to JSON with numerically ordered keys.
func writeOrderedJSON(writer io.Writer, vehicleMap map[string]vehicles.Vehicle) error {
	carIDs := make([]int, 0, len(vehicleMap))

	for carIDStr := range vehicleMap {
		carID, err := strconv.Atoi(carIDStr)
		if err != nil {
			continue
		}

		carIDs = append(carIDs, carID)
	}

	sort.Ints(carIDs)

	var buf bytes.Buffer
	buf.WriteString("{\n")

	for i, carID := range carIDs {
		carIDStr := strconv.Itoa(carID)
		vehicle := vehicleMap[carIDStr]

		if i > 0 {
			buf.WriteString(",\n")
		}

		buf.WriteString(fmt.Sprintf("  \"%s\": {\n", carIDStr))

		writeVehicleFieldsOrdered(&buf, vehicle)

		buf.WriteString("\n  }")
	}

	buf.WriteString("\n}\n")

	_, err := writer.Write(buf.Bytes())

	return err
}

// writeVehicleFieldsOrdered writes vehicle fields in consistent order.
func writeVehicleFieldsOrdered(buf *bytes.Buffer, vehicle vehicles.Vehicle) {
	for i, fieldName := range orderedVehicleFields() {
		if i > 0 {
			buf.WriteString(",\n")
		}

		buf.WriteString("    ")

		value := getVehicleFieldValue(vehicle, fieldName)

		switch valueType := value.(type) {
		case int:
			fmt.Fprintf(buf, "\"%s\": %d", fieldName, valueType)
		case string:
			escapedValue := escapeQuotes(valueType)
			fmt.Fprintf(buf, "\"%s\": \"%s\"", fieldName, escapedValue)
		case bool:
			fmt.Fprintf(buf, "\"%s\": %t", fieldName, valueType)
		case float32:
			fmt.Fprintf(buf, "\"%s\": %g", fieldName, valueType)
		default:
			escapedValue := escapeQuotes(fmt.Sprintf("%v", valueType))
			fmt.Fprintf(buf, "\"%s\": \"%s\"", fieldName, escapedValue)
		}
	}
}

// escapeQuotes escapes only double quotes in a string, leaving all other characters as-is.
func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

// getVehicleFieldValueAsString returns the string representation of a vehicle field for CSV output.
func getVehicleFieldValueAsString(vehicle vehicles.Vehicle, fieldName string) string {
	value := getVehicleFieldValue(vehicle, fieldName)

	switch valueType := value.(type) {
	case int:
		return strconv.Itoa(valueType)
	case string:
		return valueType
	case bool:
		return strconv.FormatBool(valueType)
	case float32:
		return strconv.FormatFloat(float64(valueType), 'f', -1, 32)
	default:
		return fmt.Sprintf("%v", valueType)
	}
}

// getVehicleFieldValue returns the value of a vehicle field by name.
func getVehicleFieldValue(vehicle vehicles.Vehicle, fieldName string) any {
	switch fieldName {
	case "CarID":
		return vehicle.CarID
	case "Manufacturer":
		return vehicle.Manufacturer
	case "Model":
		return vehicle.Model
	case "Year":
		return vehicle.Year
	case "OpenCockpit":
		return vehicle.OpenCockpit
	case "CarType":
		return vehicle.CarType
	case "Category":
		return vehicle.Category
	case "Drivetrain":
		return vehicle.Drivetrain
	case "Aspiration":
		return vehicle.Aspiration
	case "Length":
		return vehicle.Length
	case "Width":
		return vehicle.Width
	case "Height":
		return vehicle.Height
	case "Wheelbase":
		return vehicle.Wheelbase
	case "TrackFront":
		return vehicle.TrackFront
	case "TrackRear":
		return vehicle.TrackRear
	case "EngineLayout":
		return vehicle.EngineLayout
	case "EngineBankAngle":
		return vehicle.EngineBankAngle
	case "EngineCrankPlaneAngle":
		return vehicle.EngineCrankPlaneAngle
	default:
		return ""
	}
}

// promptVehicleData prompts the user for vehicle information interactively.
// If existingVehicle is nil, it prompts for a new vehicle.
// If existingVehicle is provided, it shows current values and allows editing.
// For edit mode, it also takes the vehicleMap to check for CarID conflicts.
func promptVehicleData(scanner *bufio.Scanner, existingVehicle *vehicles.Vehicle, vehicleMap map[string]vehicles.Vehicle, originalCarID int) (vehicles.Vehicle, error) {
	var vehicle vehicles.Vehicle

	// If editing, start with existing values
	if existingVehicle != nil {
		vehicle = *existingVehicle
	}

	// CarID handling
	if existingVehicle == nil {
		// Add mode - CarID is required
		fmt.Print("CarID (unique integer): ")
		scanner.Scan()

		carIDStr := strings.TrimSpace(scanner.Text())
		if carIDStr == "" {
			return vehicle, ErrCarIDRequired
		}

		carID, err := strconv.Atoi(carIDStr)
		if err != nil {
			return vehicle, fmt.Errorf("invalid CarID '%s': %w", carIDStr, err)
		}

		vehicle.CarID = carID
	} else {
		// Edit mode - show current CarID with option to change
		fmt.Printf("CarID [%d]: ", existingVehicle.CarID)
		scanner.Scan()

		if input := strings.TrimSpace(scanner.Text()); input != "" {
			newCarID, err := strconv.Atoi(input)
			if err != nil {
				return vehicle, fmt.Errorf("invalid CarID '%s': %w", input, err)
			}

			// Check if new CarID conflicts with existing vehicle (and it's not the same vehicle)
			if newCarID != originalCarID {
				newCarIDKey := strconv.Itoa(newCarID)
				if _, exists := vehicleMap[newCarIDKey]; exists {
					return vehicle, fmt.Errorf("%w: %d", ErrCarIDAlreadyExists, newCarID)
				}
			}

			vehicle.CarID = newCarID
		}
	}

	prompt := func(prompt string, currentValue string, dataType string) string {
		for {
			if currentValue == "" {
				fmt.Printf("%s: ", prompt)
			} else {
				fmt.Printf("%s [%s]: ", prompt, currentValue)
			}

			scanner.Scan()

			input := strings.TrimSpace(scanner.Text())

			switch dataType {
			case "string":
				if input != "" {
					return input
				}

				return currentValue
			case "bool":
				value := "false"

				if currentValue != "" {
					value = currentValue
				}

				if input != "" {
					value = input
				}

				if value == "true" || value == "false" {
					return value
				}

				fmt.Printf("Invalid value %q. Please enter true or false.\n", input)
			case "int":
				value := "0"

				if currentValue != "" {
					value = currentValue
				}

				if input != "" {
					value = input
				}

				_, err := strconv.Atoi(value)
				if err == nil {
					return value
				}

				fmt.Printf("Invalid value %q. Please enter a valid integer.\n", input)
			case "uint":
				value := "0"

				if currentValue != "" {
					value = currentValue
				}

				if input != "" {
					value = input
				}

				_, err := strconv.ParseUint(value, 10, 32)
				if err == nil {
					return value
				}

				fmt.Printf("Invalid value %q %q %q. Please enter a valid unsigned integer.\n", input, currentValue, value)
			case "float32":
				value := "0.0"

				if currentValue != "" {
					value = currentValue
				}

				if input != "" {
					value = input
				}

				_, err := strconv.ParseFloat(value, 32)
				if err == nil {
					return value
				}

				fmt.Printf("Invalid value %q. Please enter a valid float.\n", input)
			default:
				return input
			}
		}
	}

	// Get all vehicle fields
	var err error

	vehicle.Manufacturer = prompt("Manufacturer:", vehicle.Manufacturer, "string")
	vehicle.Model = prompt("Model:", vehicle.Model, "string")

	vehicle.Year, err = strconv.Atoi(prompt("Year:", strconv.Itoa(vehicle.Year), "uint"))
	if err != nil {
		return vehicle, fmt.Errorf("invalid year, must be a positive integer: %w", err)
	}

	vehicle.OpenCockpit, err = strconv.ParseBool(prompt("Open cockpit (true/false):", strconv.FormatBool(vehicle.OpenCockpit), "bool"))
	if err != nil {
		return vehicle, fmt.Errorf("invalid boolean, must be true or false: %w", err)
	}

	vehicle.CarType = prompt("Car type (street/race):", vehicle.CarType, "string")
	vehicle.Category = prompt("Category (e.g., Gr.1, Gr.3, Gr.4, Gr.B, or empty):", vehicle.Category, "string")
	vehicle.Drivetrain = prompt("Drivetrain (FR/FF/MR/RR/4WD):", vehicle.Drivetrain, "string")
	vehicle.Aspiration = prompt("Aspiration (NA/TC/SC/EV/TD/TC+SC):", vehicle.Aspiration, "string")

	vehicle.Length, err = strconv.Atoi(prompt("Length (mm):", strconv.Itoa(vehicle.Length), "int"))
	if err != nil {
		return vehicle, fmt.Errorf("invalid length: %w", err)
	}

	vehicle.Width, err = strconv.Atoi(prompt("Width (mm):", strconv.Itoa(vehicle.Width), "int"))
	if err != nil {
		return vehicle, fmt.Errorf("invalid width: %w", err)
	}

	vehicle.Height, err = strconv.Atoi(prompt("Height (mm):", strconv.Itoa(vehicle.Height), "int"))
	if err != nil {
		return vehicle, fmt.Errorf("invalid height: %w", err)
	}

	vehicle.Wheelbase, err = strconv.Atoi(prompt("Wheelbase (mm):", strconv.Itoa(vehicle.Wheelbase), "int"))
	if err != nil {
		return vehicle, fmt.Errorf("invalid wheelbase: %w", err)
	}

	vehicle.TrackFront, err = strconv.Atoi(prompt("Track Front (mm):", strconv.Itoa(vehicle.TrackFront), "int"))
	if err != nil {
		return vehicle, fmt.Errorf("invalid track front: %w", err)
	}

	vehicle.TrackRear, err = strconv.Atoi(prompt("Track Rear (mm):", strconv.Itoa(vehicle.TrackRear), "int"))
	if err != nil {
		return vehicle, fmt.Errorf("invalid track rear: %w", err)
	}

	vehicle.EngineLayout = prompt("Engine layout (e.g., V8, V6, I4, H4, or empty):", vehicle.EngineLayout, "string")

	engineBankAngleString := prompt("Engine cylinder bank angle (decimal degrees):", strconv.FormatFloat(float64(vehicle.EngineBankAngle), 'f', -1, 32), "float32")

	engineBankAngle, err := strconv.ParseFloat(engineBankAngleString, 32)
	if err != nil {
		return vehicle, fmt.Errorf("invalid engine banke angle: %w", err)
	}

	vehicle.EngineBankAngle = float32(engineBankAngle)

	engineCrankPlaneAngleString := prompt("Engine crank plane angle (decimal degrees):", strconv.FormatFloat(float64(vehicle.EngineCrankPlaneAngle), 'f', -1, 32), "float32")

	engineCrankPlaneAngle, err := strconv.ParseFloat(engineCrankPlaneAngleString, 32)
	if err != nil {
		return vehicle, fmt.Errorf("invalid crank angle angle: %w", err)
	}

	vehicle.EngineCrankPlaneAngle = float32(engineCrankPlaneAngle)

	return vehicle, nil
}

func addVehicleInteractive(inventoryFile string) error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Adding a new vehicle to the inventory")
	fmt.Println()

	// Get vehicle information from user
	vehicle, err := promptVehicleData(scanner, nil, nil, 0)
	if err != nil {
		return err
	}

	// Display summary
	fmt.Println("\n--- Vehicle Summary ---")
	fmt.Printf("CarID: %d\n", vehicle.CarID)
	fmt.Printf("Manufacturer: %s\n", vehicle.Manufacturer)
	fmt.Printf("Model: %s\n", vehicle.Model)
	fmt.Printf("Year: %d\n", vehicle.Year)
	fmt.Printf("OpenCockpit: %t\n", vehicle.OpenCockpit)
	fmt.Printf("CarType: %s\n", vehicle.CarType)
	fmt.Printf("Category: %s\n", vehicle.Category)
	fmt.Printf("Drivetrain: %s\n", vehicle.Drivetrain)
	fmt.Printf("Aspiration: %s\n", vehicle.Aspiration)
	fmt.Printf("EngineLayout: %s\n", vehicle.EngineLayout)
	fmt.Printf("EngineBankAngle: %.1f\n", vehicle.EngineBankAngle)
	fmt.Printf("EngineCrankPlaneAngle: %.1f\n", vehicle.EngineCrankPlaneAngle)

	fmt.Print("\nSave this vehicle to inventory? (y/N): ")
	scanner.Scan()

	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirmNegative(confirm) {
		fmt.Println("Vehicle not saved.")

		return nil
	}

	// Load existing inventory
	var vehicleMap map[string]vehicles.Vehicle

	_, err = os.Stat(inventoryFile)
	if err == nil {
		// Load an existing file when it exists
		jsonData, err := os.ReadFile(inventoryFile)
		if err != nil {
			return fmt.Errorf("reading inventory file: %w", err)
		}

		err = json.Unmarshal(jsonData, &vehicleMap)
		if err != nil {
			return fmt.Errorf("parsing inventory JSON: %w", err)
		}
	} else {
		// Create a new map when the file doesn't exist
		vehicleMap = make(map[string]vehicles.Vehicle)
	}

	// Check if the ID already exists
	carIDKey := strconv.Itoa(vehicle.CarID)
	if _, exists := vehicleMap[carIDKey]; exists {
		fmt.Print("Warning: A vehicle with this CarID already exists. Overwrite? (y/N): ")
		scanner.Scan()

		confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if confirmNegative(confirm) {
			fmt.Println("Vehicle not saved.")

			return nil
		}
	}

	// Add vehicle to map
	vehicleMap[carIDKey] = vehicle

	// Create directory if it doesn't exist
	dir := strings.TrimSuffix(inventoryFile, "/inventory.json")
	if dir != inventoryFile {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}

	// Write updated inventory back to file
	outputF, err := os.Create(inventoryFile)
	if err != nil {
		return fmt.Errorf("creating inventory file: %w", err)
	}
	defer outputF.Close()

	err = writeOrderedJSON(outputF, vehicleMap)
	if err != nil {
		return fmt.Errorf("encoding inventory JSON: %w", err)
	}

	fmt.Printf("Vehicle successfully added to %s\n", inventoryFile)

	return nil
}

func editVehicleInteractively(inventoryFile string, carID int) error {
	// Load existing inventory
	_, err := os.Stat(inventoryFile)
	if err != nil {
		return fmt.Errorf("inventory file does not exist: %w", err)
	}

	jsonData, err := os.ReadFile(inventoryFile)
	if err != nil {
		return fmt.Errorf("reading inventory file: %w", err)
	}

	var vehicleMap map[string]vehicles.Vehicle

	err = json.Unmarshal(jsonData, &vehicleMap)
	if err != nil {
		return fmt.Errorf("parsing inventory JSON: %w", err)
	}

	// Check if vehicle exists
	carIDKey := strconv.Itoa(carID)

	existingVehicle, exists := vehicleMap[carIDKey]
	if !exists {
		return fmt.Errorf("%w: CarID %d", ErrVehicleNotFound, carID)
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Editing vehicle with CarID %d\n", carID)
	fmt.Println("Current values are shown in [brackets]. Press Enter to keep current value or enter new value:")
	fmt.Println()

	// Get vehicle information from user, using existing values as defaults
	vehicle, err := promptVehicleData(scanner, &existingVehicle, vehicleMap, carID)
	if err != nil {
		return err
	}

	// Display summary
	fmt.Println("\n--- Updated Vehicle Summary ---")
	fmt.Printf("CarID: %d\n", vehicle.CarID)
	fmt.Printf("Manufacturer: %s\n", vehicle.Manufacturer)
	fmt.Printf("Model: %s\n", vehicle.Model)
	fmt.Printf("Year: %d\n", vehicle.Year)
	fmt.Printf("OpenCockpit: %t\n", vehicle.OpenCockpit)
	fmt.Printf("CarType: %s\n", vehicle.CarType)
	fmt.Printf("Category: %s\n", vehicle.Category)
	fmt.Printf("Drivetrain: %s\n", vehicle.Drivetrain)
	fmt.Printf("Aspiration: %s\n", vehicle.Aspiration)
	fmt.Printf("Length: %d mm\n", vehicle.Length)
	fmt.Printf("Width: %d mm\n", vehicle.Width)
	fmt.Printf("Height: %d mm\n", vehicle.Height)
	fmt.Printf("Wheelbase: %d mm\n", vehicle.Wheelbase)
	fmt.Printf("TrackFront: %d mm\n", vehicle.TrackFront)
	fmt.Printf("TrackRear: %d mm\n", vehicle.TrackRear)
	fmt.Printf("EngineLayout: %s\n", vehicle.EngineLayout)
	fmt.Printf("EngineBankAngle: %.1f\n", vehicle.EngineBankAngle)
	fmt.Printf("EngineCrankPlaneAngle: %.1f\n", vehicle.EngineCrankPlaneAngle)

	fmt.Print("\nSave changes to inventory? (y/N): ")
	scanner.Scan()

	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirmNegative(confirm) {
		fmt.Println("Changes not saved.")

		return nil
	}

	// Remove old entry if CarID changed
	if vehicle.CarID != carID {
		delete(vehicleMap, carIDKey)
	}

	// Update vehicle in map
	newCarIDKey := strconv.Itoa(vehicle.CarID)
	vehicleMap[newCarIDKey] = vehicle

	// Write updated inventory back to file
	outputF, err := os.Create(inventoryFile)
	if err != nil {
		return fmt.Errorf("creating inventory file: %w", err)
	}
	defer outputF.Close()

	err = writeOrderedJSON(outputF, vehicleMap)
	if err != nil {
		return fmt.Errorf("encoding inventory JSON: %w", err)
	}

	fmt.Printf("Vehicle successfully updated in %s\n", inventoryFile)

	return nil
}

func deleteVehicle(inventoryFile string, carID int) error {
	// Load existing inventory
	_, err := os.Stat(inventoryFile)
	if err != nil {
		return fmt.Errorf("inventory file does not exist: %w", err)
	}

	jsonData, err := os.ReadFile(inventoryFile)
	if err != nil {
		return fmt.Errorf("reading inventory file: %w", err)
	}

	var vehicleMap map[string]vehicles.Vehicle

	err = json.Unmarshal(jsonData, &vehicleMap)
	if err != nil {
		return fmt.Errorf("parsing inventory JSON: %w", err)
	}

	// Check if vehicle exists
	carIDKey := strconv.Itoa(carID)

	vehicle, exists := vehicleMap[carIDKey]
	if !exists {
		return fmt.Errorf("%w: CarID %d", ErrVehicleNotFound, carID)
	}

	// Display vehicle to be deleted
	fmt.Printf("Vehicle to be deleted:\n")
	fmt.Printf("  CarID: %d\n", vehicle.CarID)
	fmt.Printf("  Manufacturer: %s\n", vehicle.Manufacturer)
	fmt.Printf("  Model: %s\n", vehicle.Model)
	fmt.Printf("  Year: %d\n", vehicle.Year)

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("\nAre you sure you want to delete this vehicle? (y/N): ")
	scanner.Scan()

	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirmNegative(confirm) {
		fmt.Println("Vehicle not deleted.")

		return nil
	}

	// Remove vehicle from map
	delete(vehicleMap, carIDKey)

	// Write updated inventory back to file
	outputF, err := os.Create(inventoryFile)
	if err != nil {
		return fmt.Errorf("creating inventory file: %w", err)
	}
	defer outputF.Close()

	err = writeOrderedJSON(outputF, vehicleMap)
	if err != nil {
		return fmt.Errorf("encoding inventory JSON: %w", err)
	}

	fmt.Printf("Vehicle with CarID %d successfully deleted from %s\n", carID, inventoryFile)

	return nil
}

// PDVehicle represents the structure of a vehicle entry in the PD inventory JSON.
type PDVehicle struct {
	ID              string `json:"id"`              //nolint:tagliatelle // third party JSON schema
	NameShort       string `json:"nameShort"`       //nolint:tagliatelle // third party JSON schema
	Manufacturer    string `json:"manufacturer"`    //nolint:tagliatelle // third party JSON schema
	Year            int    `json:"year"`            //nolint:tagliatelle // third party JSON schema
	DriveTrain      string `json:"driveTrain"`      //nolint:tagliatelle // third party JSON schema
	AspirationShort string `json:"aspirationShort"` //nolint:tagliatelle // third party JSON schema
	CarClass        string `json:"carClass"`        //nolint:tagliatelle // third party JSON schema
	LengthV         int    `json:"length_v"`        //nolint:tagliatelle // third party JSON schema
	WidthV          int    `json:"width_v"`         //nolint:tagliatelle // third party JSON schema
	HeightV         int    `json:"height_v"`        //nolint:tagliatelle // third party JSON schema
}

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

func mergeInventories(gtInventoryFile, pdInventoryFile string, noColor, dryRun bool) error {
	// Load GT inventory
	gtData, err := os.ReadFile(gtInventoryFile)
	if err != nil {
		return fmt.Errorf("reading GT inventory file: %w", err)
	}

	var gtVehicleMap map[string]vehicles.Vehicle

	err = json.Unmarshal(gtData, &gtVehicleMap)
	if err != nil {
		return fmt.Errorf("parsing GT inventory JSON: %w", err)
	}

	// Load PD inventory
	pdData, err := os.ReadFile(pdInventoryFile)
	if err != nil {
		return fmt.Errorf("reading PD inventory file: %w", err)
	}

	var pdVehicleMap map[string]PDVehicle

	err = json.Unmarshal(pdData, &pdVehicleMap)
	if err != nil {
		return fmt.Errorf("parsing PD inventory JSON: %w", err)
	}

	// Color helpers
	red := func(s string) string {
		if noColor {
			return s
		}

		return colorRed + s + colorReset
	}
	green := func(s string) string {
		if noColor {
			return s
		}

		return colorGreen + s + colorReset
	}
	yellow := func(s string) string {
		if noColor {
			return s
		}

		return colorYellow + s + colorReset
	}
	cyan := func(s string) string {
		if noColor {
			return s
		}

		return colorCyan + s + colorReset
	}

	// Merge data from PD inventory into GT inventory
	mergedCount := 0
	addedCount := 0

	// Store changes for sorted output
	type changeRecord struct {
		carID   int
		changes []string
		isNew   bool
	}

	var allChanges []changeRecord

	// First, update existing vehicles
	for carIDStr, gtVehicle := range gtVehicleMap {
		if pdVehicle, exists := pdVehicleMap[carIDStr]; exists {
			updated := false

			var changes []string

			// Overwrite Manufacturer from manufacturer
			if pdVehicle.Manufacturer != "" && pdVehicle.Manufacturer != pdNullValue && gtVehicle.Manufacturer != pdVehicle.Manufacturer {
				if gtVehicle.Manufacturer != "" {
					changes = append(changes, fmt.Sprintf("  %s Manufacturer: %s", red("-"), red("'"+gtVehicle.Manufacturer+"'")))
					changes = append(changes, fmt.Sprintf("  %s Manufacturer: %s", green("+"), green("'"+pdVehicle.Manufacturer+"'")))
				} else {
					changes = append(changes, fmt.Sprintf("  %s Manufacturer: %s", green("+"), green("'"+pdVehicle.Manufacturer+"'")))
				}

				gtVehicle.Manufacturer = pdVehicle.Manufacturer
				updated = true
			}

			// Overwrite Model from nameShort
			if pdVehicle.NameShort != "" && pdVehicle.NameShort != pdNullValue && gtVehicle.Model != pdVehicle.NameShort {
				if gtVehicle.Model != "" {
					changes = append(changes, fmt.Sprintf("  %s Model: %s", yellow("|"), cyan("'"+gtVehicle.Model+"'")))
					changes = append(changes, fmt.Sprintf("  %s Model: %s", green("+"), green("'"+pdVehicle.NameShort+"'")))
				} else {
					changes = append(changes, fmt.Sprintf("  %s Model: %s", green("+"), green("'"+pdVehicle.NameShort+"'")))
				}

				gtVehicle.Model = pdVehicle.NameShort
				updated = true
			}

			// Overwrite Year if it's 0 or different
			if pdVehicle.Year > 0 && gtVehicle.Year != pdVehicle.Year {
				if gtVehicle.Year > 0 {
					changes = append(changes, fmt.Sprintf("  %s Year: %s", red("-"), red(strconv.Itoa(gtVehicle.Year))))
					changes = append(changes, fmt.Sprintf("  %s Year: %s", green("+"), green(strconv.Itoa(pdVehicle.Year))))
				} else {
					changes = append(changes, fmt.Sprintf("  %s Year: %s", green("+"), green(strconv.Itoa(pdVehicle.Year))))
				}

				gtVehicle.Year = pdVehicle.Year
				updated = true
			}

			// Overwrite Drivetrain from driveTrain
			if pdVehicle.DriveTrain != "" && pdVehicle.DriveTrain != pdNullValue && gtVehicle.Drivetrain != pdVehicle.DriveTrain {
				if gtVehicle.Drivetrain != "" && gtVehicle.Drivetrain != "-" {
					changes = append(changes, fmt.Sprintf("  %s Drivetrain: %s", red("-"), red("'"+gtVehicle.Drivetrain+"'")))
					changes = append(changes, fmt.Sprintf("  %s Drivetrain: %s", green("+"), green("'"+pdVehicle.DriveTrain+"'")))
				} else {
					changes = append(changes, fmt.Sprintf("  %s Drivetrain: %s", green("+"), green("'"+pdVehicle.DriveTrain+"'")))
				}

				gtVehicle.Drivetrain = pdVehicle.DriveTrain
				updated = true
			}

			// Overwrite Aspiration from aspirationShort
			if pdVehicle.AspirationShort != "" && pdVehicle.AspirationShort != pdNullValue && gtVehicle.Aspiration != pdVehicle.AspirationShort {
				if gtVehicle.Aspiration != "" && gtVehicle.Aspiration != "-" {
					changes = append(changes, fmt.Sprintf("  %s Aspiration: %s", red("-"), red("'"+gtVehicle.Aspiration+"'")))
					changes = append(changes, fmt.Sprintf("  %s Aspiration: %s", green("+"), green("'"+pdVehicle.AspirationShort+"'")))
				} else {
					changes = append(changes, fmt.Sprintf("  %s Aspiration: %s", green("+"), green("'"+pdVehicle.AspirationShort+"'")))
				}

				gtVehicle.Aspiration = pdVehicle.AspirationShort
				updated = true
			}

			// Overwrite Category from carClass
			if pdVehicle.CarClass != "" && pdVehicle.CarClass != pdNullValue && gtVehicle.Category != pdVehicle.CarClass {
				if gtVehicle.Category != "" {
					changes = append(changes, fmt.Sprintf("  %s Category: %s", red("-"), red("'"+gtVehicle.Category+"'")))
					changes = append(changes, fmt.Sprintf("  %s Category: %s", green("+"), green("'"+pdVehicle.CarClass+"'")))
				} else {
					changes = append(changes, fmt.Sprintf("  %s Category: %s", green("+"), green("'"+pdVehicle.CarClass+"'")))
				}

				gtVehicle.Category = pdVehicle.CarClass
				updated = true
			}

			// Only update dimensions if GT inventory has 0 values
			if gtVehicle.Length == 0 && pdVehicle.LengthV > 0 {
				changes = append(changes, fmt.Sprintf("  %s Length: %s", green("+"), green(strconv.Itoa(pdVehicle.LengthV))))
				gtVehicle.Length = pdVehicle.LengthV
				updated = true
			}

			if gtVehicle.Width == 0 && pdVehicle.WidthV > 0 {
				changes = append(changes, fmt.Sprintf("  %s Width: %s", green("+"), green(strconv.Itoa(pdVehicle.WidthV))))
				gtVehicle.Width = pdVehicle.WidthV
				updated = true
			}

			if gtVehicle.Height == 0 && pdVehicle.HeightV > 0 {
				changes = append(changes, fmt.Sprintf("  %s Height: %s", green("+"), green(strconv.Itoa(pdVehicle.HeightV))))
				gtVehicle.Height = pdVehicle.HeightV
				updated = true
			}

			if updated {
				gtVehicleMap[carIDStr] = gtVehicle
				mergedCount++

				if len(changes) > 0 {
					carID, _ := strconv.Atoi(carIDStr)
					allChanges = append(allChanges, changeRecord{
						carID:   carID,
						changes: changes,
						isNew:   false,
					})
				}
			}
		}
	}

	// Second, add new vehicles that don't exist in GT inventory
	for carIDStr, pdVehicle := range pdVehicleMap {
		if _, exists := gtVehicleMap[carIDStr]; !exists {
			// Parse CarID
			carID, err := strconv.Atoi(carIDStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: skipping invalid CarID '%s': %v\n", carIDStr, err)

				continue
			}

			// Create new vehicle from PD data
			newVehicle := vehicles.Vehicle{
				CarID:        carID,
				Manufacturer: pdVehicle.Manufacturer,
				Model:        pdVehicle.NameShort,
				Year:         pdVehicle.Year,
				Category:     pdVehicle.CarClass,
				Drivetrain:   pdVehicle.DriveTrain,
				Aspiration:   pdVehicle.AspirationShort,
				Length:       pdVehicle.LengthV,
				Width:        pdVehicle.WidthV,
				Height:       pdVehicle.HeightV,
			}

			gtVehicleMap[carIDStr] = newVehicle
			addedCount++

			var changes []string
			if pdVehicle.Manufacturer != "" && pdVehicle.Manufacturer != pdNullValue {
				changes = append(changes, fmt.Sprintf("  %s Manufacturer: %s", green("+"), green("'"+pdVehicle.Manufacturer+"'")))
			}

			if pdVehicle.NameShort != "" && pdVehicle.NameShort != pdNullValue {
				changes = append(changes, fmt.Sprintf("  %s Model: %s", green("+"), green("'"+pdVehicle.NameShort+"'")))
			}

			if pdVehicle.Year > 0 {
				changes = append(changes, fmt.Sprintf("  %s Year: %s", green("+"), green(strconv.Itoa(pdVehicle.Year))))
			}

			if pdVehicle.CarClass != "" && pdVehicle.CarClass != pdNullValue {
				changes = append(changes, fmt.Sprintf("  %s Category: %s", green("+"), green("'"+pdVehicle.CarClass+"'")))
			}

			if pdVehicle.DriveTrain != "" && pdVehicle.DriveTrain != pdNullValue {
				changes = append(changes, fmt.Sprintf("  %s Drivetrain: %s", green("+"), green("'"+pdVehicle.DriveTrain+"'")))
			}

			if pdVehicle.AspirationShort != "" && pdVehicle.AspirationShort != pdNullValue {
				changes = append(changes, fmt.Sprintf("  %s Aspiration: %s", green("+"), green("'"+pdVehicle.AspirationShort+"'")))
			}

			if pdVehicle.LengthV > 0 {
				changes = append(changes, fmt.Sprintf("  %s Length: %s", green("+"), green(strconv.Itoa(pdVehicle.LengthV))))
			}

			if pdVehicle.WidthV > 0 {
				changes = append(changes, fmt.Sprintf("  %s Width: %s", green("+"), green(strconv.Itoa(pdVehicle.WidthV))))
			}

			if pdVehicle.HeightV > 0 {
				changes = append(changes, fmt.Sprintf("  %s Height: %s", green("+"), green(strconv.Itoa(pdVehicle.HeightV))))
			}

			allChanges = append(allChanges, changeRecord{
				carID:   carID,
				changes: changes,
				isNew:   true,
			})
		}
	}

	// Sort changes by CarID
	sort.Slice(allChanges, func(i, j int) bool {
		return allChanges[i].carID < allChanges[j].carID
	})

	// Output changes in sorted order
	for _, record := range allChanges {
		if record.isNew {
			fmt.Fprintf(os.Stderr, "%s\n", green(fmt.Sprintf("+ New CarID %d:", record.carID)))
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", cyan(fmt.Sprintf("CarID %d:", record.carID)))
		}

		for _, change := range record.changes {
			fmt.Fprintf(os.Stderr, "%s\n", change)
		}
	}

	// Write merged inventory to file (unless dry-run)
	if dryRun {
		fmt.Fprintf(os.Stderr, "\n[DRY RUN] Would write changes to %s\n", gtInventoryFile)

		if addedCount > 0 {
			fmt.Fprintf(os.Stderr, "[DRY RUN] Would add %d new vehicles and update %d existing vehicles\n", addedCount, mergedCount)
		} else {
			fmt.Fprintf(os.Stderr, "[DRY RUN] Would update %d vehicles\n", mergedCount)
		}
	} else {
		outputF, err := os.Create(gtInventoryFile)
		if err != nil {
			return fmt.Errorf("creating inventory file: %w", err)
		}
		defer outputF.Close()

		err = writeOrderedJSON(outputF, gtVehicleMap)
		if err != nil {
			return fmt.Errorf("encoding merged JSON: %w", err)
		}

		if addedCount > 0 {
			fmt.Fprintf(os.Stderr, "Successfully added %d new vehicles and updated %d existing vehicles to %s\n", addedCount, mergedCount, gtInventoryFile)
		} else {
			fmt.Fprintf(os.Stderr, "Successfully updated %d vehicles in %s\n", mergedCount, gtInventoryFile)
		}
	}

	return nil
}

// GTCar represents a car entry from the Gran Turismo website cars.js file.
type GTCar struct {
	ID              string `json:"id"`              //nolint:tagliatelle // third party JSON schema
	NameShort       string `json:"nameShort"`       //nolint:tagliatelle // third party JSON schema
	NameLong        string `json:"nameLong"`        //nolint:tagliatelle // third party JSON schema
	ManufacturerID  string `json:"manufacturerId"`  //nolint:tagliatelle // third party JSON schema
	CarClass        string `json:"carClass"`        //nolint:tagliatelle // third party JSON schema
	DriveTrain      string `json:"driveTrain"`      //nolint:tagliatelle // third party JSON schema
	AspirationShort string `json:"aspirationShort"` //nolint:tagliatelle // third party JSON schema
	LengthV         int    `json:"length_v"`        //nolint:tagliatelle // third party JSON schema
	WidthV          int    `json:"width_v"`         //nolint:tagliatelle // third party JSON schema
	HeightV         int    `json:"height_v"`        //nolint:tagliatelle // third party JSON schema
}

// GTTuner represents a manufacturer/tuner entry from the Gran Turismo website tuners.js file.
type GTTuner struct {
	ID        string `json:"id"`        //nolint:tagliatelle // third party JSON schema
	Name      string `json:"name"`      //nolint:tagliatelle // third party JSON schema
	NameShort string `json:"nameShort"` //nolint:tagliatelle // third party JSON schema
}

// fetchAndMergeGTData fetches car data from Gran Turismo website and merges it with local inventory.
func fetchAndMergeGTData(inventoryFile, locale string, noColor, dryRun bool) error {
	fmt.Fprintf(os.Stderr, "Fetching Gran Turismo car data for locale: %s\n", locale)

	// Step 1: Fetch the main carlist page to get the JS bundle name
	baseURL := fmt.Sprintf("https://www.gran-turismo.com/%s/gt7/carlist/", locale)
	fmt.Fprintf(os.Stderr, "Fetching carlist page: %s\n", baseURL)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL, nil)
	if err != nil {
		return fmt.Errorf("creating request for carlist page: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching carlist page: %w", err)
	}
	defer resp.Body.Close()

	htmlBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading carlist HTML: %w", err)
	}

	// Step 2: Extract the main JS bundle filename
	indexJsPattern := regexp.MustCompile(`src="([^"]*index-[^"]*\.js)"`)

	matches := indexJsPattern.FindSubmatch(htmlBody)
	if len(matches) < 2 {
		return ErrMainJSBundleNotFound
	}

	indexJsPath := string(matches[1])
	// Handle relative paths
	if !strings.HasPrefix(indexJsPath, "http") {
		if strings.HasPrefix(indexJsPath, "/") {
			indexJsPath = "https://www.gran-turismo.com" + indexJsPath
		} else {
			indexJsPath = "https://www.gran-turismo.com/" + indexJsPath
		}
	}

	fmt.Fprintf(os.Stderr, "Found main JS bundle: %s\n", indexJsPath)

	// Step 3: Fetch the main JS bundle
	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, indexJsPath, nil)
	if err != nil {
		return fmt.Errorf("creating request for main JS bundle: %w", err)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching main JS bundle: %w", err)
	}
	defer resp.Body.Close()

	bundleBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading main JS bundle: %w", err)
	}

	// Step 4: Extract the cars data filename
	carsJsPattern := regexp.MustCompile(fmt.Sprintf(`cars\.%s-([A-Za-z0-9_-]+)\.js`, locale))

	matches = carsJsPattern.FindSubmatch(bundleBody)
	if len(matches) < 1 {
		return fmt.Errorf("%w: locale %s", ErrCarsJSNotFound, locale)
	}

	carsJsFilename := string(matches[0])
	carsJsURL := strings.Replace(indexJsPath, filepath.Base(indexJsPath), carsJsFilename, 1)

	fmt.Fprintf(os.Stderr, "Found cars data file: %s\n", carsJsURL)

	// Step 4b: Extract the tuners data filename
	tunersJsPattern := regexp.MustCompile(fmt.Sprintf(`tuners\.%s-([A-Za-z0-9_-]+)\.js`, locale))

	matches = tunersJsPattern.FindSubmatch(bundleBody)
	if len(matches) < 1 {
		return fmt.Errorf("%w: locale %s", ErrTunersJSNotFound, locale)
	}

	tunersJsFilename := string(matches[0])
	tunersJsURL := strings.Replace(indexJsPath, filepath.Base(indexJsPath), tunersJsFilename, 1)

	fmt.Fprintf(os.Stderr, "Found tuners data file: %s\n", tunersJsURL)

	// Step 5: Fetch the cars data file
	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, carsJsURL, nil)
	if err != nil {
		return fmt.Errorf("creating request for cars data file: %w", err)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching cars data file: %w", err)
	}

	defer resp.Body.Close()

	carsBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading cars data file: %w", err)
	}

	// Step 5b: Fetch the tuners data file
	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, tunersJsURL, nil)
	if err != nil {
		return fmt.Errorf("creating request for tuners data file: %w", err)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching tuners data file: %w", err)
	}
	defer resp.Body.Close()

	tunersBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading tuners data file: %w", err)
	}

	// Step 6: Parse JavaScript and convert to JSON using goja JavaScript engine
	fmt.Fprintf(os.Stderr, "Parsing car data...\n")

	// The file uses ES6 export syntax: const r={...};export{r as Cars};
	// Strip the export statement since goja doesn't support ES6 modules
	jsCode := string(carsBody)
	jsCode = regexp.MustCompile(`;\s*export\s*{[^}]*}\s*;?\s*$`).ReplaceAllString(jsCode, "")

	// Use goja to execute the JavaScript
	vm := goja.New() //nolint:varnamelen // descriptive enough

	// Execute the JavaScript code (just the const declaration)
	_, err = vm.RunString(jsCode)
	if err != nil {
		return fmt.Errorf("executing JavaScript: %w", err)
	}

	// Get the variable (it's typically 'r' or similar single-letter var)
	// Extract the variable name from the const declaration
	varNamePattern := regexp.MustCompile(`^const\s+(\w+)\s*=`)

	varNameMatches := varNamePattern.FindStringSubmatch(jsCode)
	if len(varNameMatches) < 2 {
		return ErrVariableNameNotFound
	}

	varName := varNameMatches[1]

	// Get the Cars object
	carsValue := vm.Get(varName)
	if carsValue == nil {
		return fmt.Errorf("%w: '%s'", ErrCarsObjectNotFound, varName)
	}

	// Convert to JSON
	carDataJSON, err := json.Marshal(carsValue.Export())
	if err != nil {
		return fmt.Errorf("converting Cars to JSON: %w", err)
	}

	// Parse the car data
	var gtCarsMap map[string]GTCar

	err = json.Unmarshal(carDataJSON, &gtCarsMap)
	if err != nil {
		return fmt.Errorf("parsing car data JSON: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d cars in GT data\n", len(gtCarsMap))

	// Step 6b: Parse tuners JavaScript data
	fmt.Fprintf(os.Stderr, "Parsing tuner data...\n")

	tunersJsCode := string(tunersBody)
	tunersJsCode = regexp.MustCompile(`;\s*export\s*{[^}]*}\s*;?\s*$`).ReplaceAllString(tunersJsCode, "")

	vm = goja.New()

	_, err = vm.RunString(tunersJsCode)
	if err != nil {
		return fmt.Errorf("executing tuners JavaScript: %w", err)
	}

	varNameMatches = varNamePattern.FindStringSubmatch(tunersJsCode)
	if len(varNameMatches) < 2 {
		return ErrTunersVariableNameNotFound
	}

	varName = varNameMatches[1]

	tunersValue := vm.Get(varName)
	if tunersValue == nil {
		return fmt.Errorf("%w: '%s'", ErrTunersObjectNotFound, varName)
	}

	tunerDataJSON, err := json.Marshal(tunersValue.Export())
	if err != nil {
		return fmt.Errorf("converting Tuners to JSON: %w", err)
	}

	var gtTunersMap map[string]GTTuner

	err = json.Unmarshal(tunerDataJSON, &gtTunersMap)
	if err != nil {
		return fmt.Errorf("parsing tuner data JSON: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d manufacturers in GT data\n", len(gtTunersMap))

	// Step 7: Convert to temporary file in PD format for merging
	tempFile, err := os.CreateTemp("", "gt-cars-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Convert GTCar to PDVehicle format
	pdVehicleMap := make(map[string]PDVehicle)

	for carKey, gtCar := range gtCarsMap {
		// Extract numeric car ID from the car key (e.g., "car123" -> "123")
		carIDPattern := regexp.MustCompile(`car(\d+)`)

		carIDMatches := carIDPattern.FindStringSubmatch(carKey)
		if len(carIDMatches) < 2 {
			continue
		}

		carID := carIDMatches[1]

		// Resolve manufacturer name from manufacturer ID
		manufacturerName := ""

		if gtCar.ManufacturerID != "" {
			if tuner, exists := gtTunersMap[gtCar.ManufacturerID]; exists {
				manufacturerName = strings.TrimSpace(tuner.Name)
			}
		}

		// Extract year from model name if it ends with '## (e.g., "Camaro Z28 '69")
		year := 0

		yearPattern := regexp.MustCompile(`'(\d{2})$`)
		if matches := yearPattern.FindStringSubmatch(gtCar.NameShort); len(matches) > 1 {
			shortYear, err := strconv.Atoi(matches[1])
			if err == nil {
				// Convert 2-digit year to 4-digit year
				// Use current year + 1 as cutoff (last 2 digits)
				currentYear := time.Now().Year()

				cutoff := (currentYear + 1) % 100
				if shortYear <= cutoff {
					year = 2000 + shortYear
				} else {
					year = 1900 + shortYear
				}
			}
		}

		pdVehicleMap[carID] = PDVehicle{
			ID:              gtCar.ID,
			NameShort:       gtCar.NameShort,
			Manufacturer:    manufacturerName,
			Year:            year,
			DriveTrain:      gtCar.DriveTrain,
			AspirationShort: gtCar.AspirationShort,
			CarClass:        gtCar.CarClass,
			LengthV:         gtCar.LengthV,
			WidthV:          gtCar.WidthV,
			HeightV:         gtCar.HeightV,
		}
	}

	// Write to temp file
	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")

	err = encoder.Encode(pdVehicleMap)
	if err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	tempFile.Close()

	fmt.Fprintf(os.Stderr, "Merging with local inventory...\n")

	// Step 8: Use existing merge function
	return mergeInventories(inventoryFile, tempFile.Name(), noColor, dryRun)
}

// orderedVehicleFields defines the canonical field order for vehicle data.
func orderedVehicleFields() []string {
	return []string{
		"CarID",
		"Manufacturer",
		"Model",
		"Year",
		"OpenCockpit",
		"CarType",
		"Category",
		"Drivetrain",
		"Aspiration",
		"Length",
		"Width",
		"Height",
		"Wheelbase",
		"TrackFront",
		"TrackRear",
		"EngineLayout",
		"EngineBankAngle",
		"EngineCrankPlaneAngle",
	}
}

func confirmNegative(confirm string) bool {
	if confirm != "y" && confirm != "yes" {
		return false
	}

	return true
}
