package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/zetetos/gt-telemetry/pkg/vehicles"
)

// mergeInventories merges PD inventory data into the GT inventory file.
func mergeInventories(gtInventoryFile, pdInventoryFile string, noColor, dryRun bool) error {
	gtVehicleMap, err := loadGTInventory(gtInventoryFile)
	if err != nil {
		return err
	}

	pdVehicleMap, err := loadPDInventory(pdInventoryFile)
	if err != nil {
		return err
	}

	colors := newColorPrinter(noColor)
	allChanges, mergedCount, addedCount := performMerge(gtVehicleMap, pdVehicleMap, colors)

	printChanges(allChanges, colors)

	if dryRun {
		printDryRunSummary(gtInventoryFile, addedCount, mergedCount)
	} else {
		err := writeMergedInventory(gtInventoryFile, gtVehicleMap, addedCount, mergedCount)
		if err != nil {
			return err
		}
	}

	return nil
}

// loadGTInventory loads the GT inventory from file.
func loadGTInventory(gtInventoryFile string) (map[string]vehicles.Vehicle, error) {
	gtData, err := os.ReadFile(gtInventoryFile)
	if err != nil {
		return nil, fmt.Errorf("reading GT inventory file: %w", err)
	}

	gtVehicleMap := map[string]vehicles.Vehicle{}

	err = json.Unmarshal(gtData, &gtVehicleMap)
	if err != nil {
		return nil, fmt.Errorf("parsing GT inventory JSON: %w", err)
	}

	return gtVehicleMap, nil
}

// loadPDInventory loads the PD inventory from file.
func loadPDInventory(pdInventoryFile string) (map[string]PDVehicle, error) {
	pdData, err := os.ReadFile(pdInventoryFile)
	if err != nil {
		return nil, fmt.Errorf("reading PD inventory file: %w", err)
	}

	pdVehicleMap := map[string]PDVehicle{}

	err = json.Unmarshal(pdData, &pdVehicleMap)
	if err != nil {
		return nil, fmt.Errorf("parsing PD inventory JSON: %w", err)
	}

	return pdVehicleMap, nil
}

// performMerge performs the merge operation and returns changes and counts.
func performMerge(gtVehicleMap map[string]vehicles.Vehicle, pdVehicleMap map[string]PDVehicle, colors *colorPrinter) ([]changeRecord, int, int) {
	allChanges := []changeRecord{}
	mergedCount := 0
	addedCount := 0

	// Update existing vehicles
	for carIDStr, gtVehicle := range gtVehicleMap {
		if pdVehicle, exists := pdVehicleMap[carIDStr]; exists {
			updated, changes := getVehicleUpdateChanges(gtVehicle, pdVehicle, colors)
			if updated {
				gtVehicleMap[carIDStr] = applyVehicleUpdates(gtVehicle, pdVehicle)
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

	// Add new vehicles
	for carIDStr, pdVehicle := range pdVehicleMap {
		if _, exists := gtVehicleMap[carIDStr]; !exists {
			carID, err := strconv.Atoi(carIDStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: skipping invalid CarID '%s': %v\n", carIDStr, err)

				continue
			}

			newVehicle := createNewVehicle(carID, pdVehicle)
			gtVehicleMap[carIDStr] = newVehicle

			addedCount++
			changes := getNewVehicleChanges(pdVehicle, colors)
			allChanges = append(allChanges, changeRecord{
				carID:   carID,
				changes: changes,
				isNew:   true,
			})
		}
	}

	return allChanges, mergedCount, addedCount
}

// createNewVehicle creates a new vehicle from PD data.
func createNewVehicle(carID int, pdVehicle PDVehicle) vehicles.Vehicle {
	return vehicles.Vehicle{
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
}

// printChanges prints all change records in sorted order.
func printChanges(allChanges []changeRecord, colors *colorPrinter) {
	sort.Slice(allChanges, func(i, j int) bool {
		return allChanges[i].carID < allChanges[j].carID
	})

	for _, record := range allChanges {
		if record.isNew {
			fmt.Fprintf(os.Stderr, "\n%s %s:\n", colors.Green("[NEW]"), colors.Cyan(fmt.Sprintf("CarID %d", record.carID)))
		} else {
			fmt.Fprintf(os.Stderr, "\n%s %s:\n", colors.Yellow("[UPDATE]"), colors.Cyan(fmt.Sprintf("CarID %d", record.carID)))
		}

		for _, change := range record.changes {
			fmt.Fprintln(os.Stderr, change)
		}
	}
}

// printDryRunSummary prints the summary for dry-run mode.
func printDryRunSummary(gtInventoryFile string, addedCount, mergedCount int) {
	fmt.Fprintf(os.Stderr, "\n[DRY RUN] Would write changes to %s\n", gtInventoryFile)

	if addedCount > 0 {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would add %d new vehicles and update %d existing vehicles\n", addedCount, mergedCount)
	} else {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would update %d vehicles\n", mergedCount)
	}
}

// writeMergedInventory writes the merged inventory to file and prints a summary.
func writeMergedInventory(gtInventoryFile string, gtVehicleMap map[string]vehicles.Vehicle, addedCount, mergedCount int) error {
	outputF, err := os.Create(gtInventoryFile)
	if err != nil {
		return fmt.Errorf("creating inventory file: %w", err)
	}

	defer func() {
		err := outputF.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error closing inventory file: %v\n", err)
		}
	}()

	err = writeOrderedJSON(outputF, gtVehicleMap)
	if err != nil {
		return fmt.Errorf("encoding merged JSON: %w", err)
	}

	if addedCount > 0 {
		fmt.Fprintf(os.Stderr, "Successfully added %d new vehicles and updated %d existing vehicles to %s\n", addedCount, mergedCount, gtInventoryFile)
	} else {
		fmt.Fprintf(os.Stderr, "Successfully updated %d vehicles in %s\n", mergedCount, gtInventoryFile)
	}

	return nil
}

// getVehicleUpdateChanges returns the changes needed to update a GT vehicle from a PD vehicle.
func getVehicleUpdateChanges(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle, colors *colorPrinter) (bool, []string) {
	updated := false
	changes := []string{}

	updated = checkManufacturerUpdate(gtVehicle, pdVehicle, colors, &changes) || updated
	updated = checkModelUpdate(gtVehicle, pdVehicle, colors, &changes) || updated
	updated = checkYearUpdate(gtVehicle, pdVehicle, colors, &changes) || updated
	updated = checkDrivetrainUpdate(gtVehicle, pdVehicle, colors, &changes) || updated
	updated = checkAspirationUpdate(gtVehicle, pdVehicle, colors, &changes) || updated
	updated = checkCategoryUpdate(gtVehicle, pdVehicle, colors, &changes) || updated
	updated = checkDimensionUpdates(gtVehicle, pdVehicle, colors, &changes) || updated

	return updated, changes
}

// checkManufacturerUpdate checks and records manufacturer field changes.
func checkManufacturerUpdate(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle, colors *colorPrinter, changes *[]string) bool {
	if pdVehicle.Manufacturer == "" || pdVehicle.Manufacturer == pdNullValue || gtVehicle.Manufacturer == pdVehicle.Manufacturer {
		return false
	}

	if gtVehicle.Manufacturer != "" {
		*changes = append(*changes, fmt.Sprintf("  %s Manufacturer: %s", colors.Red("-"), colors.Red("'"+gtVehicle.Manufacturer+"'")))
		*changes = append(*changes, fmt.Sprintf("  %s Manufacturer: %s", colors.Green("+"), colors.Green("'"+pdVehicle.Manufacturer+"'")))
	} else {
		*changes = append(*changes, fmt.Sprintf("  %s Manufacturer: %s", colors.Green("+"), colors.Green("'"+pdVehicle.Manufacturer+"'")))
	}

	return true
}

// checkModelUpdate checks and records model field changes.
func checkModelUpdate(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle, colors *colorPrinter, changes *[]string) bool {
	if pdVehicle.NameShort == "" || pdVehicle.NameShort == pdNullValue || gtVehicle.Model == pdVehicle.NameShort {
		return false
	}

	if gtVehicle.Model != "" {
		*changes = append(*changes, fmt.Sprintf("  %s Model: %s", colors.Yellow("|"), colors.Cyan("'"+gtVehicle.Model+"'")))
		*changes = append(*changes, fmt.Sprintf("  %s Model: %s", colors.Green("+"), colors.Green("'"+pdVehicle.NameShort+"'")))
	} else {
		*changes = append(*changes, fmt.Sprintf("  %s Model: %s", colors.Green("+"), colors.Green("'"+pdVehicle.NameShort+"'")))
	}

	return true
}

// checkYearUpdate checks and records year field changes.
func checkYearUpdate(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle, colors *colorPrinter, changes *[]string) bool {
	if pdVehicle.Year <= 0 || gtVehicle.Year == pdVehicle.Year {
		return false
	}

	if gtVehicle.Year > 0 {
		*changes = append(*changes, fmt.Sprintf("  %s Year: %s", colors.Red("-"), colors.Red(strconv.Itoa(gtVehicle.Year))))
		*changes = append(*changes, fmt.Sprintf("  %s Year: %s", colors.Green("+"), colors.Green(strconv.Itoa(pdVehicle.Year))))
	} else {
		*changes = append(*changes, fmt.Sprintf("  %s Year: %s", colors.Green("+"), colors.Green(strconv.Itoa(pdVehicle.Year))))
	}

	return true
}

// checkDrivetrainUpdate checks and records drivetrain field changes.
func checkDrivetrainUpdate(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle, colors *colorPrinter, changes *[]string) bool {
	if pdVehicle.DriveTrain == "" || pdVehicle.DriveTrain == pdNullValue || gtVehicle.Drivetrain == pdVehicle.DriveTrain {
		return false
	}

	if gtVehicle.Drivetrain != "" && gtVehicle.Drivetrain != "-" {
		*changes = append(*changes, fmt.Sprintf("  %s Drivetrain: %s", colors.Red("-"), colors.Red("'"+gtVehicle.Drivetrain+"'")))
		*changes = append(*changes, fmt.Sprintf("  %s Drivetrain: %s", colors.Green("+"), colors.Green("'"+pdVehicle.DriveTrain+"'")))
	} else {
		*changes = append(*changes, fmt.Sprintf("  %s Drivetrain: %s", colors.Green("+"), colors.Green("'"+pdVehicle.DriveTrain+"'")))
	}

	return true
}

// checkAspirationUpdate checks and records aspiration field changes.
func checkAspirationUpdate(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle, colors *colorPrinter, changes *[]string) bool {
	if pdVehicle.AspirationShort == "" || pdVehicle.AspirationShort == pdNullValue || gtVehicle.Aspiration == pdVehicle.AspirationShort {
		return false
	}

	if gtVehicle.Aspiration != "" && gtVehicle.Aspiration != "-" {
		*changes = append(*changes, fmt.Sprintf("  %s Aspiration: %s", colors.Red("-"), colors.Red("'"+gtVehicle.Aspiration+"'")))
		*changes = append(*changes, fmt.Sprintf("  %s Aspiration: %s", colors.Green("+"), colors.Green("'"+pdVehicle.AspirationShort+"'")))
	} else {
		*changes = append(*changes, fmt.Sprintf("  %s Aspiration: %s", colors.Green("+"), colors.Green("'"+pdVehicle.AspirationShort+"'")))
	}

	return true
}

// checkCategoryUpdate checks and records category field changes.
func checkCategoryUpdate(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle, colors *colorPrinter, changes *[]string) bool {
	if pdVehicle.CarClass == "" || pdVehicle.CarClass == pdNullValue || gtVehicle.Category == pdVehicle.CarClass {
		return false
	}

	if gtVehicle.Category != "" {
		*changes = append(*changes, fmt.Sprintf("  %s Category: %s", colors.Red("-"), colors.Red("'"+gtVehicle.Category+"'")))
		*changes = append(*changes, fmt.Sprintf("  %s Category: %s", colors.Green("+"), colors.Green("'"+pdVehicle.CarClass+"'")))
	} else {
		*changes = append(*changes, fmt.Sprintf("  %s Category: %s", colors.Green("+"), colors.Green("'"+pdVehicle.CarClass+"'")))
	}

	return true
}

// checkDimensionUpdates checks and records dimension field changes.
func checkDimensionUpdates(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle, colors *colorPrinter, changes *[]string) bool {
	updated := false

	if gtVehicle.Length == 0 && pdVehicle.LengthV > 0 {
		*changes = append(*changes, fmt.Sprintf("  %s Length: %s", colors.Green("+"), colors.Green(strconv.Itoa(pdVehicle.LengthV))))
		updated = true
	}

	if gtVehicle.Width == 0 && pdVehicle.WidthV > 0 {
		*changes = append(*changes, fmt.Sprintf("  %s Width: %s", colors.Green("+"), colors.Green(strconv.Itoa(pdVehicle.WidthV))))
		updated = true
	}

	if gtVehicle.Height == 0 && pdVehicle.HeightV > 0 {
		*changes = append(*changes, fmt.Sprintf("  %s Height: %s", colors.Green("+"), colors.Green(strconv.Itoa(pdVehicle.HeightV))))
		updated = true
	}

	return updated
}

// applyVehicleUpdates applies updates from PDVehicle to GT Vehicle.
func applyVehicleUpdates(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle) vehicles.Vehicle {
	gtVehicle = updateManufacturer(gtVehicle, pdVehicle)
	gtVehicle = updateModel(gtVehicle, pdVehicle)
	gtVehicle = updateYear(gtVehicle, pdVehicle)
	gtVehicle = updateDrivetrain(gtVehicle, pdVehicle)
	gtVehicle = updateAspiration(gtVehicle, pdVehicle)
	gtVehicle = updateCategory(gtVehicle, pdVehicle)
	gtVehicle = updateDimensions(gtVehicle, pdVehicle)

	return gtVehicle
}

// updateManufacturer updates the manufacturer field if needed.
func updateManufacturer(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle) vehicles.Vehicle {
	if pdVehicle.Manufacturer != "" && pdVehicle.Manufacturer != pdNullValue {
		gtVehicle.Manufacturer = pdVehicle.Manufacturer
	}

	return gtVehicle
}

// updateModel updates the model field if needed.
func updateModel(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle) vehicles.Vehicle {
	if pdVehicle.NameShort != "" && pdVehicle.NameShort != pdNullValue {
		gtVehicle.Model = pdVehicle.NameShort
	}

	return gtVehicle
}

// updateYear updates the year field if needed.
func updateYear(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle) vehicles.Vehicle {
	if pdVehicle.Year > 0 {
		gtVehicle.Year = pdVehicle.Year
	}

	return gtVehicle
}

// updateDrivetrain updates the drivetrain field if needed.
func updateDrivetrain(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle) vehicles.Vehicle {
	if pdVehicle.DriveTrain != "" && pdVehicle.DriveTrain != pdNullValue {
		gtVehicle.Drivetrain = pdVehicle.DriveTrain
	}

	return gtVehicle
}

// updateAspiration updates the aspiration field if needed.
func updateAspiration(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle) vehicles.Vehicle {
	if pdVehicle.AspirationShort != "" && pdVehicle.AspirationShort != pdNullValue {
		gtVehicle.Aspiration = pdVehicle.AspirationShort
	}

	return gtVehicle
}

// updateCategory updates the category field if needed.
func updateCategory(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle) vehicles.Vehicle {
	if pdVehicle.CarClass != "" && pdVehicle.CarClass != pdNullValue {
		gtVehicle.Category = pdVehicle.CarClass
	}

	return gtVehicle
}

// updateDimensions updates dimension fields if needed.
func updateDimensions(gtVehicle vehicles.Vehicle, pdVehicle PDVehicle) vehicles.Vehicle {
	if gtVehicle.Length == 0 && pdVehicle.LengthV > 0 {
		gtVehicle.Length = pdVehicle.LengthV
	}

	if gtVehicle.Width == 0 && pdVehicle.WidthV > 0 {
		gtVehicle.Width = pdVehicle.WidthV
	}

	if gtVehicle.Height == 0 && pdVehicle.HeightV > 0 {
		gtVehicle.Height = pdVehicle.HeightV
	}

	return gtVehicle
}

// getNewVehicleChanges returns the changes for a newly added vehicle.
func getNewVehicleChanges(pdVehicle PDVehicle, colors *colorPrinter) []string {
	var changes []string

	changes = appendManufacturerChange(changes, pdVehicle, colors)
	changes = appendModelChange(changes, pdVehicle, colors)
	changes = appendYearChange(changes, pdVehicle, colors)
	changes = appendCategoryChange(changes, pdVehicle, colors)
	changes = appendDrivetrainChange(changes, pdVehicle, colors)
	changes = appendAspirationChange(changes, pdVehicle, colors)
	changes = appendDimensionChanges(changes, pdVehicle, colors)

	return changes
}

// appendManufacturerChange appends manufacturer change if present.
func appendManufacturerChange(changes []string, pdVehicle PDVehicle, colors *colorPrinter) []string {
	if pdVehicle.Manufacturer != "" && pdVehicle.Manufacturer != pdNullValue {
		changes = append(changes, fmt.Sprintf("  %s Manufacturer: %s", colors.Green("+"), colors.Green("'"+pdVehicle.Manufacturer+"'")))
	}

	return changes
}

// appendModelChange appends model change if present.
func appendModelChange(changes []string, pdVehicle PDVehicle, colors *colorPrinter) []string {
	if pdVehicle.NameShort != "" && pdVehicle.NameShort != pdNullValue {
		changes = append(changes, fmt.Sprintf("  %s Model: %s", colors.Green("+"), colors.Green("'"+pdVehicle.NameShort+"'")))
	}

	return changes
}

// appendYearChange appends year change if present.
func appendYearChange(changes []string, pdVehicle PDVehicle, colors *colorPrinter) []string {
	if pdVehicle.Year > 0 {
		changes = append(changes, fmt.Sprintf("  %s Year: %s", colors.Green("+"), colors.Green(strconv.Itoa(pdVehicle.Year))))
	}

	return changes
}

// appendCategoryChange appends category change if present.
func appendCategoryChange(changes []string, pdVehicle PDVehicle, colors *colorPrinter) []string {
	if pdVehicle.CarClass != "" && pdVehicle.CarClass != pdNullValue {
		changes = append(changes, fmt.Sprintf("  %s Category: %s", colors.Green("+"), colors.Green("'"+pdVehicle.CarClass+"'")))
	}

	return changes
}

// appendDrivetrainChange appends drivetrain change if present.
func appendDrivetrainChange(changes []string, pdVehicle PDVehicle, colors *colorPrinter) []string {
	if pdVehicle.DriveTrain != "" && pdVehicle.DriveTrain != pdNullValue {
		changes = append(changes, fmt.Sprintf("  %s Drivetrain: %s", colors.Green("+"), colors.Green("'"+pdVehicle.DriveTrain+"'")))
	}

	return changes
}

// appendAspirationChange appends aspiration change if present.
func appendAspirationChange(changes []string, pdVehicle PDVehicle, colors *colorPrinter) []string {
	if pdVehicle.AspirationShort != "" && pdVehicle.AspirationShort != pdNullValue {
		changes = append(changes, fmt.Sprintf("  %s Aspiration: %s", colors.Green("+"), colors.Green("'"+pdVehicle.AspirationShort+"'")))
	}

	return changes
}

// appendDimensionChanges appends dimension changes if present.
func appendDimensionChanges(changes []string, pdVehicle PDVehicle, colors *colorPrinter) []string {
	if pdVehicle.LengthV > 0 {
		changes = append(changes, fmt.Sprintf("  %s Length: %s", colors.Green("+"), colors.Green(strconv.Itoa(pdVehicle.LengthV))))
	}

	if pdVehicle.WidthV > 0 {
		changes = append(changes, fmt.Sprintf("  %s Width: %s", colors.Green("+"), colors.Green(strconv.Itoa(pdVehicle.WidthV))))
	}

	if pdVehicle.HeightV > 0 {
		changes = append(changes, fmt.Sprintf("  %s Height: %s", colors.Green("+"), colors.Green(strconv.Itoa(pdVehicle.HeightV))))
	}

	return changes
}
