package main

import (
	"flag"
	"fmt"
	"os"
)

const usage = `inventory - Import and export vehicle inventory data between JSON and CSV formats

Usage:
  inventory <action> [arguments]

Actions:
  convert  <dir>             Export per-vehicle JSON inventory to CSV (stdout)
  convert  <file.csv> <dir>  Import CSV and write per-vehicle JSON files to dir
  manifest <dir>             Generate manifest JSON from inventory directory (stdout)
  update   <dir> [locale]    Fetch and merge car data from Gran Turismo website

Arguments:
  dir                      Path to a directory containing per-vehicle JSON files.
  file.csv                 Path to a CSV inventory file.
  locale                   Locale code for fetch (default: gb). Examples: gb, us, jp, au

Flags:
  -help                    Show this help message
  -no-color                Disable colored output
  -dry-run                 Show changes without modifying files

Examples:
  # Export inventory directory to CSV
  inventory convert pkg/vehicles/inventory > inventory.csv

  # Import CSV into inventory directory
  inventory convert inventory.csv pkg/vehicles/inventory

  # Generate manifest from inventory directory
  inventory manifest pkg/vehicles/inventory > pkg/vehicles/inventory/manifest.json

  # Fetch and merge data from Gran Turismo website (default GB locale)
  inventory update pkg/vehicles/inventory

  # Fetch and merge data for a specific locale
  inventory update pkg/vehicles/inventory us
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
	case "manifest":
		retCode = handleManifestAction(args)
	case "update":
		retCode = handleUpdateAction(args, flags)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s'. Supported actions: convert, manifest, update\n\n", action)
		fmt.Print(usage)

		retCode = 1
	}

	os.Exit(retCode)
}

// handleConvertAction processes the convert action.
func handleConvertAction(args []string) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Input argument is required for convert action\n\n")
		fmt.Print(usage)

		return 1
	}

	inputArg := args[1]
	outputArg := ""

	if len(args) > 2 {
		outputArg = args[2]
	}

	err := convertFile(inputArg, outputArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		return 1
	}

	return 0
}

// handleManifestAction processes the manifest action.
func handleManifestAction(args []string) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Directory argument is required for manifest action\n\n")
		fmt.Print(usage)

		return 1
	}

	inventoryDir := args[1]

	vehicleMap, err := loadInventoryDir(inventoryDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		return 1
	}

	data, err := buildManifestJSON(vehicleMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		return 1
	}

	fmt.Print(string(data))

	return 0
}

// handleUpdateAction processes the update/fetch action.
func handleUpdateAction(args []string, flags cliFlags) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Inventory directory argument is required for update action\n\n")
		fmt.Print(usage)

		return 1
	}

	inventoryDir := args[1]

	locale := "gb" // default locale
	if len(args) > 2 {
		locale = args[2]
	}

	err := fetchAndMergeGTData(inventoryDir, locale, flags.noColor, flags.dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching GT data: %v\n", err)

		return 1
	}

	return 0
}
