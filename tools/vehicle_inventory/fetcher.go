package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// fetchAndMergeGTData fetches car data from Gran Turismo website and merges it with local inventory.
func fetchAndMergeGTData(inventoryFile, locale string, noColor, dryRun bool) error {
	fmt.Fprintf(os.Stderr, "Fetching Gran Turismo car data for locale: %s\n", locale)

	gtCarsMap, gtTunersMap, err := fetchGTWebsiteData(locale)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Found %d cars in GT data\n", len(gtCarsMap))
	fmt.Fprintf(os.Stderr, "Found %d manufacturers in GT data\n", len(gtTunersMap))

	tempFileName, err := writePDVehicleTempFile(gtCarsMap, gtTunersMap)
	if err != nil {
		return err
	}

	defer func() {
		err := os.Remove(tempFileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error removing temp file: %v\n", err)
		}
	}()

	fmt.Fprintf(os.Stderr, "Merging with local inventory...\n")

	return mergeInventories(inventoryFile, tempFileName, noColor, dryRun)
}

// fetchGTWebsiteData fetches and parses GT data from the website.
func fetchGTWebsiteData(locale string) (map[string]GTCar, map[string]GTTuner, error) {
	baseURL := fmt.Sprintf("https://www.gran-turismo.com/%s/gt7/carlist/", locale)
	fmt.Fprintf(os.Stderr, "Fetching carlist page: %s\n", baseURL)

	htmlBody, err := fetchURL(baseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching carlist page: %w", err)
	}

	indexJsPath, err := extractMainJSBundle(htmlBody)
	if err != nil {
		return nil, nil, err
	}

	fmt.Fprintf(os.Stderr, "Found main JS bundle: %s\n", indexJsPath)

	bundleBody, err := fetchURL(indexJsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching main JS bundle: %w", err)
	}

	carsJsURL, tunersJsURL, err := extractDataFileURLs(bundleBody, indexJsPath, locale)
	if err != nil {
		return nil, nil, err
	}

	fmt.Fprintf(os.Stderr, "Found cars data file: %s\n", carsJsURL)
	fmt.Fprintf(os.Stderr, "Found tuners data file: %s\n", tunersJsURL)

	carsBody, err := fetchURL(carsJsURL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching cars data file: %w", err)
	}

	tunersBody, err := fetchURL(tunersJsURL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching tuners data file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Parsing car data...\n")

	gtCarsMap, err := parseGTCarData(carsBody)
	if err != nil {
		return nil, nil, err
	}

	fmt.Fprintf(os.Stderr, "Parsing tuner data...\n")

	gtTunersMap, err := parseGTTunerData(tunersBody)
	if err != nil {
		return nil, nil, err
	}

	return gtCarsMap, gtTunersMap, nil
}

// writePDVehicleTempFile creates a temp file and writes the PD vehicle map to it.
func writePDVehicleTempFile(gtCarsMap map[string]GTCar, gtTunersMap map[string]GTTuner) (string, error) {
	tempFile, err := os.CreateTemp("", "gt-cars-*.json")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error closing temp file: %v\n", err)
		}
	}()

	pdVehicleMap := convertGTToPD(gtCarsMap, gtTunersMap)

	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")

	err = encoder.Encode(pdVehicleMap)
	if err != nil {
		return "", fmt.Errorf("writing temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// fetchURL fetches the content of a URL and returns the body as []byte.
func fetchURL(url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error closing response body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", url, err)
	}

	return body, nil
}

// extractMainJSBundle extracts the main JS bundle path from HTML.
func extractMainJSBundle(htmlBody []byte) (string, error) {
	indexJsPattern := regexp.MustCompile(`src="([^"]*index-[^"]*\.js)"`)

	matches := indexJsPattern.FindSubmatch(htmlBody)
	if len(matches) < 2 {
		return "", ErrMainJSBundleNotFound
	}

	indexJsPath := string(matches[1])
	if !strings.HasPrefix(indexJsPath, "http") {
		if strings.HasPrefix(indexJsPath, "/") {
			indexJsPath = "https://www.gran-turismo.com" + indexJsPath
		} else {
			indexJsPath = "https://www.gran-turismo.com/" + indexJsPath
		}
	}

	return indexJsPath, nil
}

// extractDataFileURLs extracts cars and tuners JS file URLs from the bundle.
func extractDataFileURLs(bundleBody []byte, indexJsPath, locale string) (string, string, error) {
	carsJsURL, err := extractCarsJsURL(bundleBody, indexJsPath, locale)
	if err != nil {
		return "", "", err
	}

	tunersJsURL, err := extractTunersJsURL(bundleBody, indexJsPath, locale)
	if err != nil {
		return "", "", err
	}

	return carsJsURL, tunersJsURL, nil
}

// extractCarsJsURL extracts the cars JS file URL.
func extractCarsJsURL(bundleBody []byte, indexJsPath, locale string) (string, error) {
	carsJsPattern := regexp.MustCompile(fmt.Sprintf(`cars\.%s-([A-Za-z0-9_-]+)\.js`, locale))

	carsMatches := carsJsPattern.FindSubmatch(bundleBody)
	if len(carsMatches) < 1 {
		return "", fmt.Errorf("%w: locale %s", ErrCarsJSNotFound, locale)
	}

	carsJsFilename := string(carsMatches[0])
	carsJsURL := strings.Replace(indexJsPath, filepath.Base(indexJsPath), carsJsFilename, 1)

	return carsJsURL, nil
}

// extractTunersJsURL extracts the tuners JS file URL.
func extractTunersJsURL(bundleBody []byte, indexJsPath, locale string) (string, error) {
	tunersJsPattern := regexp.MustCompile(fmt.Sprintf(`tuners\.%s-([A-Za-z0-9_-]+)\.js`, locale))

	tunersMatches := tunersJsPattern.FindSubmatch(bundleBody)
	if len(tunersMatches) < 1 {
		return "", fmt.Errorf("%w: locale %s", ErrTunersJSNotFound, locale)
	}

	tunersJsFilename := string(tunersMatches[0])
	tunersJsURL := strings.Replace(indexJsPath, filepath.Base(indexJsPath), tunersJsFilename, 1)

	return tunersJsURL, nil
}

// parseGTJSData is a generic function that parses JavaScript data from Gran Turismo website.
// It strips export statements, executes the JS in a VM, extracts the variable, and unmarshals to the target type.
func parseGTJSData[T any](body []byte, varNotFoundErr, objNotFoundErr error, dataType string) (map[string]T, error) {
	jsCode := prepareJSCode(body)

	jsvm := goja.New()

	_, err := jsvm.RunString(jsCode)
	if err != nil {
		return nil, fmt.Errorf("executing %s JavaScript: %w", dataType, err)
	}

	varName, err := extractVariableName(jsCode, varNotFoundErr)
	if err != nil {
		return nil, err
	}

	value := jsvm.Get(varName)
	if value == nil {
		return nil, fmt.Errorf("%w: '%s'", objNotFoundErr, varName)
	}

	return exportToMap[T](value, dataType)
}

// prepareJSCode removes export statements from JavaScript code.
func prepareJSCode(body []byte) string {
	jsCode := string(body)

	return regexp.MustCompile(`;\s*export\s*{[^}]*}\s*;?\s*$`).ReplaceAllString(jsCode, "")
}

// extractVariableName extracts the variable name from JavaScript code.
func extractVariableName(jsCode string, notFoundErr error) (string, error) {
	varNamePattern := regexp.MustCompile(`^const\s+(\w+)\s*=`)

	varNameMatches := varNamePattern.FindStringSubmatch(jsCode)
	if len(varNameMatches) < 2 {
		return "", notFoundErr
	}

	return varNameMatches[1], nil
}

// exportToMap exports a goja value to a map of the specified type.
func exportToMap[T any](value goja.Value, dataType string) (map[string]T, error) {
	dataJSON, err := json.Marshal(value.Export())
	if err != nil {
		return nil, fmt.Errorf("converting %s to JSON: %w", dataType, err)
	}

	var resultMap map[string]T

	err = json.Unmarshal(dataJSON, &resultMap)
	if err != nil {
		return nil, fmt.Errorf("parsing %s data JSON: %w", dataType, err)
	}

	return resultMap, nil
}

// parseGTCarData parses car JS data and returns a map of GTCar.
func parseGTCarData(carsBody []byte) (map[string]GTCar, error) {
	return parseGTJSData[GTCar](carsBody, ErrVariableNameNotFound, ErrCarsObjectNotFound, "car")
}

// parseGTTunerData parses tuner JS data and returns a map of GTTuner.
func parseGTTunerData(tunersBody []byte) (map[string]GTTuner, error) {
	return parseGTJSData[GTTuner](tunersBody, ErrTunersVariableNameNotFound, ErrTunersObjectNotFound, "tuner")
}

// convertGTToPD converts GTCar and GTTuner maps to PDVehicle map.
func convertGTToPD(gtCarsMap map[string]GTCar, gtTunersMap map[string]GTTuner) map[string]PDVehicle {
	pdVehicleMap := make(map[string]PDVehicle)

	for carKey, gtCar := range gtCarsMap {
		carID, err := extractCarID(carKey)
		if err != nil {
			continue
		}

		manufacturerName := getManufacturerName(gtCar.ManufacturerID, gtTunersMap)
		year := extractYearFromName(gtCar.NameShort)

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

	return pdVehicleMap
}

// extractCarID extracts the numeric car ID from a car key like "car1234".
func extractCarID(carKey string) (string, error) {
	carIDPattern := regexp.MustCompile(`car(\d+)`)

	carIDMatches := carIDPattern.FindStringSubmatch(carKey)
	if len(carIDMatches) < 2 {
		return "", fmt.Errorf("invalid car key format: %s", carKey) //nolint:err113
	}

	return carIDMatches[1], nil
}

// getManufacturerName gets the manufacturer name from the tuners map.
func getManufacturerName(manufacturerID string, gtTunersMap map[string]GTTuner) string {
	if manufacturerID == "" {
		return ""
	}

	if tuner, exists := gtTunersMap[manufacturerID]; exists {
		return strings.TrimSpace(tuner.Name)
	}

	return ""
}

// extractYearFromName extracts the year from a car name (e.g., "'22" -> 2022).
func extractYearFromName(nameShort string) int {
	yearPattern := regexp.MustCompile(`'(\d{2})$`)

	matches := yearPattern.FindStringSubmatch(nameShort)
	if len(matches) <= 1 {
		return 0
	}

	shortYear, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}

	currentYear := time.Now().Year()
	cutoff := (currentYear + 1) % 100

	if shortYear <= cutoff {
		return 2000 + shortYear
	}

	return 1900 + shortYear
}
