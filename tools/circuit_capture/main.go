package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"
	"unicode"

	gttelemetry "github.com/zetetos/gt-telemetry/v2"
	gtmodels "github.com/zetetos/gt-telemetry/v2/pkg/models"
)

const (
	defaultOutputDir = "./data/circuits/"
	sleepDuration    = 8 * time.Millisecond
	initCoordinateZ  = 1000000
)

var (
	ErrCircuitNameRequired        = errors.New("circuit name is required (use -n flag)")
	ErrCircuitCountryCodeRequired = errors.New("circuit country code is required (use -c flag)")
	ErrCaptureComplete            = errors.New("capture complete")
	ErrSessionExitedEarly         = errors.New("session exited before lap complete, capture aborted")
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
	CountryCode   string             `json:"country"`
	LengthMetres  int                `json:"lengthMetres"`
	Coordinates   CircuitCoordinates `json:"coordinates"`
}

// Config holds the application configuration.
type Config struct {
	OutputDir     string
	Name          string
	VariationName string
	Default       bool
	CountryCode   string
}

// CircuitCapture handles the capture process state.
type CircuitCapture struct {
	config            *Config
	gt                *gttelemetry.Client
	circuitData       CircuitData
	lastSeq           uint32
	lastLap           int16
	initCoordinate    gtmodels.Coordinate
	lastCoordinate    gtmodels.Coordinate
	startDropped      int
	distanceTravelled float64
	minX, maxX        float32
	minY, maxY        float32
	minZ, maxZ        float32
	captureActive     bool
	extentsInit       bool
	ready             bool
}

func main() {
	config := parseFlags()

	err := config.validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()

		os.Exit(1)
	}

	capture, err := NewCircuitCapture(config)
	if err != nil {
		log.Fatalf("Failed to initialize circuit capture: %v", err)
	}

	capture.startTelemetry()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Main capture loop
	for {
		select {
		case <-sigChan:
			fmt.Println("\nRecording interrupted, file save aborted.")

			return
		default:
			if !capture.captureActive && !capture.ready {
				fmt.Println("Ready...")

				capture.ready = true
			}

			err := capture.processCapture()
			if err != nil {
				if err.Error() == "capture complete" {
					return
				}

				log.Fatalf("Error during capture: %v", err)
			}
		}
	}
}

// parseFlags parses command-line flags and returns a Config.
func parseFlags() *Config {
	config := &Config{}
	flag.StringVar(&config.OutputDir, "d", defaultOutputDir, fmt.Sprintf("Output directory name (defaults to %s)", defaultOutputDir))
	flag.StringVar(&config.Name, "n", "", "Circuit name (required)")
	flag.StringVar(&config.VariationName, "v", "", "Circuit variation name (defaults to circuit name)")
	flag.BoolVar(&config.Default, "default", false, "Set as default variation for the circuit")
	flag.StringVar(&config.CountryCode, "c", "", "Circuit country code iso 3166-1 (required)")
	flag.Parse()

	return config
}

// validate checks if the config is valid.
func (c *Config) validate() error {
	if c.Name == "" {
		return ErrCircuitNameRequired
	}

	if c.VariationName == "" {
		c.VariationName = c.Name
	}

	if c.CountryCode == "" {
		return ErrCircuitCountryCodeRequired
	}

	// Ensure output directory exists
	err := os.MkdirAll(c.OutputDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", c.OutputDir, err)
	}

	return nil
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

// NewCircuitCapture creates a new circuit capture instance.
func NewCircuitCapture(config *Config) (*CircuitCapture, error) {
	gtClient, err := gttelemetry.New(gttelemetry.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create GT telemetry client: %w", err)
	}

	newCircuit := &CircuitCapture{
		config:         config,
		gt:             gtClient,
		lastLap:        gtClient.Telemetry.CurrentLap(),
		initCoordinate: gtmodels.Coordinate{X: 0, Y: 0, Z: initCoordinateZ},
	}

	newCircuit.lastCoordinate = newCircuit.initCoordinate
	newCircuit.circuitData.Schema = "schema/circuit-schema.json"
	newCircuit.circuitData.Name = config.Name
	newCircuit.circuitData.VariationName = config.VariationName
	newCircuit.circuitData.Default = config.Default
	newCircuit.circuitData.CountryCode = config.CountryCode

	return newCircuit, nil
}

// startTelemetry starts the telemetry client in a goroutine.
func (c *CircuitCapture) startTelemetry() {
	go func() {
		for {
			recoverable, err := c.gt.Run(context.Background())
			if err != nil {
				if recoverable {
					log.Printf("GT client error (recoverable): %v", err)
					time.Sleep(time.Second)
				} else {
					log.Printf("GT client error (non-recoverable): %v", err)

					return
				}
			}
		}
	}()
}

// updateCoordinate adds a coordinate to the circuit and updates statistics.
func (c *CircuitCapture) updateCoordinate(coordinate gtmodels.Coordinate) {
	if c.circuitData.Coordinates.StartingLine == (gtmodels.Coordinate{}) {
		c.circuitData.Coordinates.StartingLine = coordinate
	}

	c.circuitData.Coordinates.Circuit = append(c.circuitData.Coordinates.Circuit, coordinate)
	c.updateExtents(coordinate)
	c.updateDistance(coordinate)
}

// updateExtents updates the min/max coordinate extents.
func (c *CircuitCapture) updateExtents(coordinate gtmodels.Coordinate) {
	if !c.extentsInit {
		c.minX, c.maxX = coordinate.X, coordinate.X
		c.minY, c.maxY = coordinate.Y, coordinate.Y
		c.minZ, c.maxZ = coordinate.Z, coordinate.Z
		c.extentsInit = true
	} else {
		c.minX = min(c.minX, coordinate.X)
		c.maxX = max(c.maxX, coordinate.X)
		c.minY = min(c.minY, coordinate.Y)
		c.maxY = max(c.maxY, coordinate.Y)
		c.minZ = min(c.minZ, coordinate.Z)
		c.maxZ = max(c.maxZ, coordinate.Z)
	}
}

// updateDistance calculates and updates the total distance travelled.
func (c *CircuitCapture) updateDistance(coordinate gtmodels.Coordinate) {
	if c.lastCoordinate != c.initCoordinate {
		dx := float64(coordinate.X - c.lastCoordinate.X)
		dy := float64(coordinate.Y - c.lastCoordinate.Y)
		dz := float64(coordinate.Z - c.lastCoordinate.Z)
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		c.distanceTravelled += dist
	}

	c.lastCoordinate = coordinate
}

// processCapture handles the main capture logic.
func (c *CircuitCapture) processCapture() error {
	if c.gt.Telemetry.Flags().GamePaused {
		time.Sleep(sleepDuration)

		return nil
	}

	seq := c.gt.Telemetry.SequenceID()
	if seq == c.lastSeq {
		time.Sleep(sleepDuration)

		return nil
	}

	c.lastSeq = seq

	currentLap := c.gt.Telemetry.CurrentLap()

	// Lap start detection
	if !c.gt.Telemetry.IsInMainMenu() && currentLap != c.lastLap {
		if c.captureActive {
			fmt.Println("Lap complete. Saving circuit data...")

			c.circuitData.LengthMetres = int(math.Round(c.distanceTravelled))
			dropped := c.gt.Statistics.PacketsDropped - c.startDropped

			err := c.saveCircuitData(dropped)
			if err != nil {
				return fmt.Errorf("failed to save circuit data: %w", err)
			}

			return ErrCaptureComplete
		}

		fmt.Println("Lap start detected.")

		c.captureActive = true
		c.lastLap = currentLap
		c.startDropped = c.gt.Statistics.PacketsDropped
	} else if c.gt.Telemetry.IsInMainMenu() && c.captureActive {
		return ErrSessionExitedEarly
	}

	coordinate := c.gt.Telemetry.PositionalMapCoordinates()
	if c.captureActive {
		c.updateCoordinate(coordinate)
	} else {
		c.lastCoordinate = coordinate
	}

	return nil
}

// saveCircuitData saves the captured circuit data to a JSON file.
func (c *CircuitCapture) saveCircuitData(dropped int) error {
	// Ensure output directory exists
	err := os.MkdirAll(c.config.OutputDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", c.config.OutputDir, err)
	}

	filename := path.Join(c.config.OutputDir, nameToID(c.circuitData.VariationName)+".json")

	fileHandle, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer fileHandle.Close()

	enc := json.NewEncoder(fileHandle)
	enc.SetIndent("", "  ")

	err = enc.Encode(c.circuitData)
	if err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	fmt.Printf("\nCircuit data saved to %s\n\n", filename)
	c.printSummary(dropped)

	return nil
}

// printSummary prints the capture summary.
func (c *CircuitCapture) printSummary(dropped int) {
	fmt.Println("#### Capture Summary ####")
	fmt.Printf("Frames dropped: %d\n", dropped)
	fmt.Printf("Points captured: %d\n", len(c.circuitData.Coordinates.Circuit))
	fmt.Printf("Circuit length: %d metres\n", c.circuitData.LengthMetres)

	fmt.Printf("Starting line: X = %.4f, Y = %.4f, Z = %.4f\n",
		c.circuitData.Coordinates.StartingLine.X,
		c.circuitData.Coordinates.StartingLine.Y,
		c.circuitData.Coordinates.StartingLine.Z,
	)

	if c.extentsInit {
		fmt.Printf("Circuit extents: X = [%.0f, %.0f], Y = [%.0f, %.0f], Z = [%.0f, %.0f]\n",
			c.minX, c.maxX, c.minY, c.maxY, c.minZ, c.maxZ)
		fmt.Printf("Circuit size: X = %.0f, Y = %.0f, Z = %.0f\n",
			c.maxX-c.minX, c.maxY-c.minY, c.maxZ-c.minZ)
	}
}
