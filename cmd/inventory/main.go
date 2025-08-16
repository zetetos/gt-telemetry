package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zetetos/gt-telemetry/internal/vehicles"
)

const usage = `inventory - Import and export vehicle inventory data between JSON and CSV formats

Usage:
  inventory <action> [arguments]

Actions:
  convert <file.csv|file.json>   Convert between JSON and CSV formats
  add     <file.json>            Add a new vehicle entry interactively
  edit    <file.json> <car-id>   Edit an existing vehicle entry
  delete  <file.json> <car-id>   Delete a vehicle entry

Arguments:
  file                     Path to input file.
  car-id                   CarID of the vehicle to edit or delete

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
		if err := addVehicleInteractively(inventoryFile); err != nil {
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

	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s'. Supported actions: convert, add, edit, delete\n\n", action)
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
	header := []string{
		"CarID",
		"Manufacturer",
		"Model",
		"Year",
		"OpenCockpit",
		"CarType",
		"Category",
		"Drivetrain",
		"Aspiration",
		"EngineLayout",
		"EngineCylinderAngle",
		"EngineCrankPlaneAngle",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	// Write vehicle data
	for _, vehicle := range vehicleMap {
		record := []string{
			strconv.Itoa(vehicle.CarID),
			vehicle.Manufacturer,
			vehicle.Model,
			strconv.Itoa(vehicle.Year),
			strconv.FormatBool(vehicle.OpenCockpit),
			vehicle.CarType,
			vehicle.Category,
			vehicle.Drivetrain,
			vehicle.Aspiration,
			vehicle.EngineLayout,
			strconv.FormatFloat(float64(vehicle.EngineCylinderAngle), 'f', -1, 32),
			strconv.FormatFloat(float64(vehicle.EngineCrankPlaneAngle), 'f', -1, 32),
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

	reader := csv.NewReader(inputF)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV header: %w", err)
	}

	// Validate header format
	expectedHeader := []string{
		"CarID",
		"Manufacturer",
		"Model",
		"Year",
		"OpenCockpit",
		"CarType",
		"Category",
		"Drivetrain",
		"Aspiration",
		"EngineLayout",
		"EngineCylinderAngle",
		"EngineCrankPlaneAngle",
	}

	if len(header) != len(expectedHeader) {
		return fmt.Errorf("invalid CSV header: expected %d columns, got %d", len(expectedHeader), len(header))
	}

	// Create map to store vehicles
	vehicleMap := make(map[string]vehicles.Vehicle)

	// Read CSV records
	for {
		record, err := reader.Read()
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

		// Parse EngineCylinderAngle
		var engineCylinderAngle float32
		if record[10] != "" && record[10] != "-" {
			angle, err := strconv.ParseFloat(record[10], 32)
			if err != nil {
				return fmt.Errorf("parsing EngineCylinderAngle '%s': %w", record[10], err)
			}
			engineCylinderAngle = float32(angle)
		}

		// Parse EngineCrankPlaneAngle
		var engineCrankPlaneAngle float32
		if record[11] != "" && record[11] != "-" {
			angle, err := strconv.ParseFloat(record[11], 32)
			if err != nil {
				return fmt.Errorf("parsing EngineCrankPlaneAngle '%s': %w", record[11], err)
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
			EngineLayout:          record[9],
			EngineCylinderAngle:   engineCylinderAngle,
			EngineCrankPlaneAngle: engineCrankPlaneAngle,
		}

		// Use CarID as the key (converted to string)
		vehicleMap[strconv.Itoa(carID)] = vehicle
	}

	// Write JSON to stdout
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(vehicleMap); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	return nil
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

	vehicle.CarType = prompt("Car Type (street/race):", vehicle.CarType, "string")
	vehicle.Category = prompt("Category (e.g., Gr.1, Gr.3, Gr.4, Gr.B, or empty):", vehicle.Category, "string")
	vehicle.Drivetrain = prompt("Drivetrain (FR/FF/MR/RR/4WD):", vehicle.Drivetrain, "string")
	vehicle.Aspiration = prompt("Aspiration (NA/TC/SC/EV/TD/TC+SC):", vehicle.Aspiration, "string")
	vehicle.EngineLayout = prompt("Engine Layout (e.g., V8, V6, I4, H4, or empty):", vehicle.EngineLayout, "string")

	engineCylinderAngle, err := strconv.ParseFloat(prompt("Engine Cylinder Angle (decimal degrees):", strconv.FormatFloat(float64(vehicle.EngineCylinderAngle), 'f', -1, 32), "float32"), 32)
	if err != nil {
		return vehicle, fmt.Errorf("invalid angle: %w", err)
	}
	vehicle.EngineCylinderAngle = float32(engineCylinderAngle)

	engineCrankPlaneAngle, err := strconv.ParseFloat(prompt("Engine Crank Plane Angle (decimal degrees):", strconv.FormatFloat(float64(vehicle.EngineCrankPlaneAngle), 'f', -1, 32), "float32"), 32)
	if err != nil {
		return vehicle, fmt.Errorf("invalid angle: %w", err)
	}
	vehicle.EngineCrankPlaneAngle = float32(engineCrankPlaneAngle)

	return vehicle, nil
}

func addVehicleInteractively(inventoryFile string) error {
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
	fmt.Printf("EngineCylinderAngle: %.1f\n", vehicle.EngineCylinderAngle)
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

	encoder := json.NewEncoder(outputF)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(vehicleMap); err != nil {
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
	fmt.Printf("EngineLayout: %s\n", vehicle.EngineLayout)
	fmt.Printf("EngineCylinderAngle: %.1f\n", vehicle.EngineCylinderAngle)
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

	encoder := json.NewEncoder(outputF)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(vehicleMap); err != nil {
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

	encoder := json.NewEncoder(outputF)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(vehicleMap); err != nil {
		return fmt.Errorf("encoding inventory JSON: %w", err)
	}

	fmt.Printf("Vehicle with CarID %d successfully deleted from %s\n", carID, inventoryFile)
	return nil
}
