package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

  # Fetch and merge data from Gran Turismo website (default GB locale)
  inventory update pkg/vehicles/vehicles.json

  # Fetch and merge data for a specific locale
  inventory update pkg/vehicles/vehicles.json us
`

// cliFlags holds all command-line flags.
type cliFlags struct {
	help    bool
	noColor bool
	dryRun  bool
}

// parseCLI parses command-line arguments and returns flags and positional arguments.
func parseCLI() (cliFlags, []string) {
	flags := cliFlags{}

	flag.BoolVar(&flags.help, "help", false, "Show help message")
	flag.BoolVar(&flags.noColor, "no-color", false, "Disable colored output")
	flag.BoolVar(&flags.dryRun, "dry-run", false, "Show changes without modifying files")

	flag.Parse()

	return flags, flag.Args()
}

// runCLI executes the CLI based on parsed flags and arguments.
func main() {
	flags, args := parseCLI()

	if flags.help {
		fmt.Print(usage)

		os.Exit(0)
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Action is required\n\n")
		fmt.Print(usage)

		os.Exit(1)
	}

	var retCode int

	action := args[0]

	switch action {
	case "convert":
		retCode = handleConvertAction(args)
	case "update":
		retCode = handleUpdateAction(args, flags)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s'. Supported actions: convert, add, edit, delete, fetch\n\n", action)
		fmt.Print(usage)

		retCode = 1
	}

	os.Exit(retCode)
}

// handleConvertAction processes the convert action.
func handleConvertAction(args []string) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Input file argument is required for convert action\n\n")
		fmt.Print(usage)

		return 1
	}

	inputFile := args[1]

	outputFormat, err := determineOutputFormat(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		return 1
	}

	err = convertFile(inputFile, outputFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		return 1
	}

	return 0
}

// handleUpdateAction processes the update/fetch action.
func handleUpdateAction(args []string, flags cliFlags) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Inventory file argument is required for update action\n\n")
		fmt.Print(usage)

		return 1
	}

	inventoryFile := args[1]

	locale := "gb" // default locale
	if len(args) > 2 {
		locale = args[2]
	}

	err := fetchAndMergeGTData(inventoryFile, locale, flags.noColor, flags.dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching GT data: %v\n", err)

		return 1
	}

	return 0
}

// determineOutputFormat determines the output format based on the input file extension.
func determineOutputFormat(inputFile string) (string, error) {
	if inputFile == "/dev/stdin" || inputFile == "-" {
		return "", errors.New("cannot determine format from stdin. Please use a file with .json or .csv extension") //nolint:err113
	}

	ext := strings.ToLower(filepath.Ext(inputFile))
	switch ext {
	case ".json":
		return "csv", nil
	case ".csv":
		return "json", nil
	default:
		return "", fmt.Errorf("unsupported file extension '%s'. Supported extensions: .json, .csv", ext) //nolint:err113
	}
}
