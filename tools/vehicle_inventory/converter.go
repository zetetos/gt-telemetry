package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/zetetos/gt-telemetry/pkg/vehicles"
)

// convertFile converts between JSON and CSV formats based on the format parameter.
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

// jsonToCSV converts a JSON vehicle inventory file to CSV format and writes to stdout.
func jsonToCSV(inputFile string) error {
	jsonData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("reading JSON file: %w", err)
	}

	vehicleMap := map[string]vehicles.Vehicle{}

	err = json.Unmarshal(jsonData, &vehicleMap)
	if err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	vehicleSlice := sortVehicleMapToSlice(vehicleMap)

	err = gocsv.Marshal(&vehicleSlice, os.Stdout)
	if err != nil {
		return fmt.Errorf("writing CSV: %w", err)
	}

	return nil
}

// csvToJSON converts a CSV vehicle inventory file to JSON format and writes to stdout.
func csvToJSON(inputFile string) error {
	inputF, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("opening CSV file: %w", err)
	}

	defer func() {
		err := inputF.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error closing input file: %v\n", err)
		}
	}()

	vehicleSlice := []vehicles.Vehicle{}

	err = gocsv.Unmarshal(inputF, &vehicleSlice)
	if err != nil {
		return fmt.Errorf("parsing CSV: %w", err)
	}

	vehicleMap := sliceToVehicleMap(vehicleSlice)

	err = writeOrderedJSON(os.Stdout, vehicleMap)
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	return nil
}

// sortVehicleMapToSlice converts a vehicle map to a sorted slice by CarID.
func sortVehicleMapToSlice(vehicleMap map[string]vehicles.Vehicle) []vehicles.Vehicle {
	carIDs := make([]int, 0, len(vehicleMap))
	for carIDStr := range vehicleMap {
		carID, err := strconv.Atoi(carIDStr)
		if err != nil {
			continue
		}

		carIDs = append(carIDs, carID)
	}

	sort.Ints(carIDs)

	vehicleSlice := make([]vehicles.Vehicle, 0, len(carIDs))
	for _, carID := range carIDs {
		vehicleSlice = append(vehicleSlice, vehicleMap[strconv.Itoa(carID)])
	}

	return vehicleSlice
}

// sliceToVehicleMap converts a vehicle slice to a map with CarID as key.
func sliceToVehicleMap(vehicleSlice []vehicles.Vehicle) map[string]vehicles.Vehicle {
	vehicleMap := make(map[string]vehicles.Vehicle, len(vehicleSlice))
	for _, vehicle := range vehicleSlice {
		vehicleMap[strconv.Itoa(vehicle.CarID)] = vehicle
	}

	return vehicleMap
}

// writeOrderedJSON writes a vehicle map to JSON with numerically ordered keys.
func writeOrderedJSON(writer io.Writer, vehicleMap map[string]vehicles.Vehicle) error {
	carIDs := extractAndSortCarIDs(vehicleMap)

	var buf bytes.Buffer

	buf.WriteString("{\n")

	for i, carID := range carIDs {
		if i > 0 {
			buf.WriteString(",\n")
		}

		carIDStr := strconv.Itoa(carID)
		vehicle := vehicleMap[carIDStr]

		buf.WriteString(fmt.Sprintf("  \"%s\": {\n", carIDStr))
		writeVehicleFieldsOrdered(&buf, vehicle)
		buf.WriteString("\n  }")
	}

	buf.WriteString("\n}\n")

	_, err := writer.Write(buf.Bytes())

	return err
}

// extractAndSortCarIDs extracts car IDs from a vehicle map and returns them sorted.
func extractAndSortCarIDs(vehicleMap map[string]vehicles.Vehicle) []int {
	carIDs := make([]int, 0, len(vehicleMap))
	for carIDStr := range vehicleMap {
		carID, err := strconv.Atoi(carIDStr)
		if err != nil {
			continue
		}

		carIDs = append(carIDs, carID)
	}

	sort.Ints(carIDs)

	return carIDs
}

// writeVehicleFieldsOrdered writes vehicle fields in consistent order to a buffer.
func writeVehicleFieldsOrdered(buf *bytes.Buffer, vehicle vehicles.Vehicle) {
	fields := []struct {
		name  string
		value any
	}{
		{"carId", vehicle.CarID},
		{"manufacturer", vehicle.Manufacturer},
		{"model", vehicle.Model},
		{"year", vehicle.Year},
		{"openCockpit", vehicle.OpenCockpit},
		{"carType", vehicle.CarType},
		{"category", vehicle.Category},
		{"drivetrain", vehicle.Drivetrain},
		{"aspiration", vehicle.Aspiration},
		{"length", vehicle.Length},
		{"width", vehicle.Width},
		{"height", vehicle.Height},
		{"wheelbase", vehicle.Wheelbase},
		{"trackFront", vehicle.TrackFront},
		{"trackRear", vehicle.TrackRear},
		{"engineLayout", vehicle.EngineLayout},
		{"engineBankAngle", vehicle.EngineBankAngle},
		{"engineCrankPlaneAngle", vehicle.EngineCrankPlaneAngle},
	}

	for i, field := range fields {
		if i > 0 {
			buf.WriteString(",\n")
		}

		buf.WriteString("    ")

		writeFieldValue(buf, field.name, field.value)
	}
}

// writeFieldValue writes a single field value to the buffer in JSON format.
func writeFieldValue(buf *bytes.Buffer, name string, value any) {
	switch valueType := value.(type) {
	case int:
		fmt.Fprintf(buf, "\"%s\": %d", name, valueType)
	case string:
		fmt.Fprintf(buf, "\"%s\": \"%s\"", name, escapeQuotes(valueType))
	case bool:
		fmt.Fprintf(buf, "\"%s\": %t", name, valueType)
	case float32:
		fmt.Fprintf(buf, "\"%s\": %g", name, valueType)
	default:
		fmt.Fprintf(buf, "\"%s\": \"%s\"", name, escapeQuotes(fmt.Sprintf("%v", valueType)))
	}
}

// escapeQuotes escapes only double quotes in a string, leaving all other characters as-is.
func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
