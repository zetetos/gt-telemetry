package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zetetos/gt-telemetry/pkg/vehicles"
)

// vehicleFields defines the canonical field order for vehicle data
var vehicleFields = []string{
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

const usage = `inventory - Import and export vehicle inventory data between JSON and CSV formats

Usage:
  inventory <action> [arguments]

Actions:
  convert <file.csv|file.json>   Convert between JSON and CSV formats
  add     <file.json>            Add a new vehicle entry interactively
  edit    <file.json> <car-id>   Edit an existing vehicle entry
  delete  <file.json> <car-id>   Delete a vehicle entry
  merge   <gt.json> <pd.json>    Merge PD inventory dimensions into GT inventory

Arguments:
  file                     Path to input file.
  car-id                   CarID of the vehicle to edit or delete
  gt.json                  Path to GT inventory JSON file
  pd.json                  Path to PD inventory JSON file

Flags:
  -help                    Show this help message

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

  # Merge PD inventory dimensions into GT inventory
  inventory merge pkg/vehicles/vehicles.json pd_inventory.json > merged.json
`

func main() {
	var (
		help = flag.Bool("help", false, "Show help message")
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

		if err := convertFile(inputFile, outputFormat); err != nil {
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
		if err := addVehicleInteractive(inventoryFile); err != nil {
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

		if err := editVehicleInteractively(inventoryFile, carID); err != nil {
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

		if err := deleteVehicle(inventoryFile, carID); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting vehicle: %v\n", err)
			os.Exit(1)
		}

	case "merge":
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: Both GT inventory and PD inventory file arguments are required for merge action\n\n")
			fmt.Print(usage)
			os.Exit(1)
		}

		gtInventoryFile := args[1]
		pdInventoryFile := args[2]

		if err := mergeInventories(gtInventoryFile, pdInventoryFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error merging inventories: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s'. Supported actions: convert, add, edit, delete, merge\n\n", action)
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
		return fmt.Errorf("unsupported format: %s", format)
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
	if err := json.Unmarshal(jsonData, &vehicleMap); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	// Write to stdout
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write CSV header
	if err := writer.Write(vehicleFields); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	// Write vehicle data
	for _, vehicle := range vehicleMap {
		record := make([]string, len(vehicleFields))
		for i, fieldName := range vehicleFields {
			record[i] = getVehicleFieldValueAsString(vehicle, fieldName)
		}
		if err := writer.Write(record); err != nil {
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
	if len(header) != len(vehicleFields) {
		return fmt.Errorf("invalid CSV header: expected %d columns, got %d", len(vehicleFields), len(header))
	}

	// Create map to store vehicles
	vehicleMap := make(map[string]vehicles.Vehicle)

	// Read CSV records
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading CSV record: %w", err)
		}

		if len(record) != len(header) {
			return fmt.Errorf("invalid CSV record: expected %d columns, got %d", len(header), len(record))
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
	if err := writeOrderedJSON(os.Stdout, vehicleMap); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	return nil
}

// writeOrderedJSON writes a vehicle map to JSON with numerically ordered keys
func writeOrderedJSON(w io.Writer, vehicleMap map[string]vehicles.Vehicle) error {
	var carIDs []int
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

	_, err := w.Write(buf.Bytes())
	return err
}

// writeVehicleFieldsOrdered writes vehicle fields in consistent order
func writeVehicleFieldsOrdered(buf *bytes.Buffer, vehicle vehicles.Vehicle) {
	for i, fieldName := range vehicleFields {
		if i > 0 {
			buf.WriteString(",\n")
		}

		buf.WriteString("    ")
		value := getVehicleFieldValue(vehicle, fieldName)

		switch v := value.(type) {
		case int:
			fmt.Fprintf(buf, "\"%s\": %d", fieldName, v)
		case string:
			escapedValue := escapeQuotes(v)
			fmt.Fprintf(buf, "\"%s\": \"%s\"", fieldName, escapedValue)
		case bool:
			fmt.Fprintf(buf, "\"%s\": %t", fieldName, v)
		case float32:
			fmt.Fprintf(buf, "\"%s\": %g", fieldName, v)
		default:
			escapedValue := escapeQuotes(fmt.Sprintf("%v", v))
			fmt.Fprintf(buf, "\"%s\": \"%s\"", fieldName, escapedValue)
		}
	}
}

// escapeQuotes escapes only double quotes in a string, leaving all other characters as-is
func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

// getVehicleFieldValueAsString returns the string representation of a vehicle field for CSV output
func getVehicleFieldValueAsString(vehicle vehicles.Vehicle, fieldName string) string {
	value := getVehicleFieldValue(vehicle, fieldName)

	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// getVehicleFieldValue returns the value of a vehicle field by name
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
			return vehicle, fmt.Errorf("CarID is required")
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
					return vehicle, fmt.Errorf("a vehicle with CarID %d already exists", newCarID)
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

				if _, err := strconv.Atoi(value); err == nil {
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

				if _, err := strconv.ParseUint(value, 10, 32); err == nil {
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

				if _, err := strconv.ParseFloat(value, 32); err == nil {
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

	EngineBankAngle, err := strconv.ParseFloat(prompt("Engine cylinder bank angle (decimal degrees):", strconv.FormatFloat(float64(vehicle.EngineBankAngle), 'f', -1, 32), "float32"), 32)
	if err != nil {
		return vehicle, fmt.Errorf("invalid angle: %w", err)
	}
	vehicle.EngineBankAngle = float32(EngineBankAngle)

	engineCrankPlaneAngle, err := strconv.ParseFloat(prompt("Engine crank plane angle (decimal degrees):", strconv.FormatFloat(float64(vehicle.EngineCrankPlaneAngle), 'f', -1, 32), "float32"), 32)
	if err != nil {
		return vehicle, fmt.Errorf("invalid angle: %w", err)
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
	if confirm != "y" && confirm != "yes" {
		fmt.Println("Vehicle not saved.")
		return nil
	}

	// Load existing inventory
	var vehicleMap map[string]vehicles.Vehicle

	if _, err := os.Stat(inventoryFile); err == nil {
		// Load an existing file when it exists
		jsonData, err := os.ReadFile(inventoryFile)
		if err != nil {
			return fmt.Errorf("reading inventory file: %w", err)
		}

		if err := json.Unmarshal(jsonData, &vehicleMap); err != nil {
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
		if confirm != "y" && confirm != "yes" {
			fmt.Println("Vehicle not saved.")
			return nil
		}
	}

	// Add vehicle to map
	vehicleMap[carIDKey] = vehicle

	// Create directory if it doesn't exist
	dir := strings.TrimSuffix(inventoryFile, "/inventory.json")
	if dir != inventoryFile {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}

	// Write updated inventory back to file
	outputF, err := os.Create(inventoryFile)
	if err != nil {
		return fmt.Errorf("creating inventory file: %w", err)
	}
	defer outputF.Close()

	if err := writeOrderedJSON(outputF, vehicleMap); err != nil {
		return fmt.Errorf("encoding inventory JSON: %w", err)
	}

	fmt.Printf("Vehicle successfully added to %s\n", inventoryFile)
	return nil
}

func editVehicleInteractively(inventoryFile string, carID int) error {
	// Load existing inventory
	if _, err := os.Stat(inventoryFile); err != nil {
		return fmt.Errorf("inventory file does not exist: %w", err)
	}

	jsonData, err := os.ReadFile(inventoryFile)
	if err != nil {
		return fmt.Errorf("reading inventory file: %w", err)
	}

	var vehicleMap map[string]vehicles.Vehicle
	if err := json.Unmarshal(jsonData, &vehicleMap); err != nil {
		return fmt.Errorf("parsing inventory JSON: %w", err)
	}

	// Check if vehicle exists
	carIDKey := strconv.Itoa(carID)
	existingVehicle, exists := vehicleMap[carIDKey]
	if !exists {
		return fmt.Errorf("vehicle with CarID %d not found in inventory", carID)
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
	if confirm != "y" && confirm != "yes" {
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

	if err := writeOrderedJSON(outputF, vehicleMap); err != nil {
		return fmt.Errorf("encoding inventory JSON: %w", err)
	}

	fmt.Printf("Vehicle successfully updated in %s\n", inventoryFile)
	return nil
}

func deleteVehicle(inventoryFile string, carID int) error {
	// Load existing inventory
	if _, err := os.Stat(inventoryFile); err != nil {
		return fmt.Errorf("inventory file does not exist: %w", err)
	}

	jsonData, err := os.ReadFile(inventoryFile)
	if err != nil {
		return fmt.Errorf("reading inventory file: %w", err)
	}

	var vehicleMap map[string]vehicles.Vehicle
	if err := json.Unmarshal(jsonData, &vehicleMap); err != nil {
		return fmt.Errorf("parsing inventory JSON: %w", err)
	}

	// Check if vehicle exists
	carIDKey := strconv.Itoa(carID)
	vehicle, exists := vehicleMap[carIDKey]
	if !exists {
		return fmt.Errorf("vehicle with CarID %d not found in inventory", carID)
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
	if confirm != "y" && confirm != "yes" {
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

	if err := writeOrderedJSON(outputF, vehicleMap); err != nil {
		return fmt.Errorf("encoding inventory JSON: %w", err)
	}

	fmt.Printf("Vehicle with CarID %d successfully deleted from %s\n", carID, inventoryFile)
	return nil
}

// PDVehicle represents the structure of a vehicle entry in the PD inventory JSON
type PDVehicle struct {
	ID              string `json:"id"`
	NameShort       string `json:"nameShort"`
	DriveTrain      string `json:"driveTrain"`
	AspirationShort string `json:"aspirationShort"`
	CarClass        string `json:"carClass"`
	LengthV         int    `json:"length_v"`
	WidthV          int    `json:"width_v"`
	HeightV         int    `json:"height_v"`
}

func mergeInventories(gtInventoryFile, pdInventoryFile string) error {
	// Load GT inventory
	gtData, err := os.ReadFile(gtInventoryFile)
	if err != nil {
		return fmt.Errorf("reading GT inventory file: %w", err)
	}

	var gtVehicleMap map[string]vehicles.Vehicle
	if err := json.Unmarshal(gtData, &gtVehicleMap); err != nil {
		return fmt.Errorf("parsing GT inventory JSON: %w", err)
	}

	// Load PD inventory
	pdData, err := os.ReadFile(pdInventoryFile)
	if err != nil {
		return fmt.Errorf("reading PD inventory file: %w", err)
	}

	var pdVehicleMap map[string]PDVehicle
	if err := json.Unmarshal(pdData, &pdVehicleMap); err != nil {
		return fmt.Errorf("parsing PD inventory JSON: %w", err)
	}

	// Merge data from PD inventory into GT inventory
	mergedCount := 0
	for carIDStr, gtVehicle := range gtVehicleMap {
		if pdVehicle, exists := pdVehicleMap[carIDStr]; exists {
			updated := false
			var changes []string

			// Overwrite Model from nameShort
			if pdVehicle.NameShort != "" && pdVehicle.NameShort != "---" && gtVehicle.Model != pdVehicle.NameShort {
				if gtVehicle.Model != "" {
					changes = append(changes, fmt.Sprintf("Model: '%s' -> '%s'", gtVehicle.Model, pdVehicle.NameShort))
				}
				gtVehicle.Model = pdVehicle.NameShort
				updated = true
			}

			// Overwrite Drivetrain from driveTrain
			if pdVehicle.DriveTrain != "" && pdVehicle.DriveTrain != "---" && gtVehicle.Drivetrain != pdVehicle.DriveTrain {
				if gtVehicle.Drivetrain != "" && gtVehicle.Drivetrain != "-" {
					changes = append(changes, fmt.Sprintf("Drivetrain: '%s' -> '%s'", gtVehicle.Drivetrain, pdVehicle.DriveTrain))
				}
				gtVehicle.Drivetrain = pdVehicle.DriveTrain
				updated = true
			}

			// Overwrite Aspiration from aspirationShort
			if pdVehicle.AspirationShort != "" && pdVehicle.AspirationShort != "---" && gtVehicle.Aspiration != pdVehicle.AspirationShort {
				if gtVehicle.Aspiration != "" && gtVehicle.Aspiration != "-" {
					changes = append(changes, fmt.Sprintf("Aspiration: '%s' -> '%s'", gtVehicle.Aspiration, pdVehicle.AspirationShort))
				}
				gtVehicle.Aspiration = pdVehicle.AspirationShort
				updated = true
			}

			// Overwrite Category from carClass
			if pdVehicle.CarClass != "" && pdVehicle.CarClass != "---" && gtVehicle.Category != pdVehicle.CarClass {
				if gtVehicle.Category != "" {
					changes = append(changes, fmt.Sprintf("Category: '%s' -> '%s'", gtVehicle.Category, pdVehicle.CarClass))
				}
				gtVehicle.Category = pdVehicle.CarClass
				updated = true
			}

			// Only update dimensions if GT inventory has 0 values
			if gtVehicle.Length == 0 && pdVehicle.LengthV > 0 {
				gtVehicle.Length = pdVehicle.LengthV
				updated = true
			}

			if gtVehicle.Width == 0 && pdVehicle.WidthV > 0 {
				gtVehicle.Width = pdVehicle.WidthV
				updated = true
			}

			if gtVehicle.Height == 0 && pdVehicle.HeightV > 0 {
				gtVehicle.Height = pdVehicle.HeightV
				updated = true
			}

			if updated {
				gtVehicleMap[carIDStr] = gtVehicle
				mergedCount++
				if len(changes) > 0 {
					fmt.Fprintf(os.Stderr, "CarID %s: %s\n", carIDStr, strings.Join(changes, ", "))
				}
			}
		}
	}

	// Write merged inventory to stdout
	if err := writeOrderedJSON(os.Stdout, gtVehicleMap); err != nil {
		return fmt.Errorf("encoding merged JSON: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully merged dimension data for %d vehicles\n", mergedCount)
	return nil
}
