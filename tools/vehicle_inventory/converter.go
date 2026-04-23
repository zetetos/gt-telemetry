package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/zetetos/gt-telemetry/v2/pkg/vehicles"
)

// convertFile converts between a per-vehicle inventory directory and CSV format.
// If inputArg is a directory it outputs CSV to stdout.
// If inputArg is a .csv file it writes individual JSON files to outputArg directory.
func convertFile(inputArg, outputArg string) error {
	info, err := os.Stat(inputArg)
	if err != nil {
		return fmt.Errorf("accessing input: %w", err)
	}

	if info.IsDir() {
		return dirToCSV(inputArg)
	}

	if strings.ToLower(filepath.Ext(inputArg)) != ".csv" {
		return errors.New("input must be a directory or a .csv file") //nolint:err113
	}

	if outputArg == "" {
		return errors.New("output directory is required when converting from CSV") //nolint:err113
	}

	return csvToDir(inputArg, outputArg)
}

// dirToCSV reads per-vehicle JSON files from inputDir and writes CSV to stdout.
func dirToCSV(inputDir string) error {
	vehicleMap, err := loadInventoryDir(inputDir)
	if err != nil {
		return fmt.Errorf("loading inventory: %w", err)
	}

	vehicleSlice := sortVehicleMapToSlice(vehicleMap)

	err = gocsv.Marshal(&vehicleSlice, os.Stdout)
	if err != nil {
		return fmt.Errorf("writing CSV: %w", err)
	}

	return nil
}

// csvToDir reads a CSV vehicle file and writes individual JSON files to outputDir.
func csvToDir(inputFile, outputDir string) error {
	inputF, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("opening CSV file: %w", err)
	}

	defer func() {
		closeErr := inputF.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: error closing input file: %v\n", closeErr)
		}
	}()

	var vehicleSlice []vehicles.Vehicle

	err = gocsv.Unmarshal(inputF, &vehicleSlice)
	if err != nil {
		return fmt.Errorf("parsing CSV: %w", err)
	}

	vehicleMap := make(map[string]vehicles.Vehicle, len(vehicleSlice))
	for _, v := range vehicleSlice {
		vehicleMap[strconv.Itoa(v.CarID)] = v
	}

	written, err := writeInventoryDir(vehicleMap, outputDir)
	if err != nil {
		return fmt.Errorf("writing inventory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Wrote %d vehicle files to %s/\n", written, outputDir)

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
