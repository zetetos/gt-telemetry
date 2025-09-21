package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	gttelemetry "github.com/zetetos/gt-telemetry"
	gtcircuits "github.com/zetetos/gt-telemetry/pkg/circuits"
	gtmodels "github.com/zetetos/gt-telemetry/pkg/models"
)

type CircuitCoordinates struct {
	Circuit      []gtmodels.CoordinateNorm `json:"circuit"`
	StartingLine gtmodels.CoordinateNorm   `json:"starting_line"`
}

type CircuitData struct {
	Name         string             `json:"name"`
	Region       string             `json:"region"`
	LengthMeters int                `json:"length_meters"`
	Coordinates  CircuitCoordinates `json:"coordinates"`
}

func main() {
	var outFile, circuitName, circuitRegion string
	flag.StringVar(&outFile, "o", "circuit_map.json", "Output file name. Default: circuit_map.json")
	flag.StringVar(&circuitName, "name", "", "Circuit name (optional)")
	flag.StringVar(&circuitRegion, "region", "", "Circuit region (optional)")
	flag.Parse()

	gt, err := gttelemetry.New(gttelemetry.Options{})
	if err != nil {
		log.Fatalf("Error creating GT telemetry client: %v", err)
	}

	go func() {
		for {
			err, recoverable := gt.Run()
			if err != nil {
				if recoverable {
					log.Printf("GT client error (recoverable): %v", err)
				} else {
					log.Fatalf("GT client error (non-recoverable): %v", err)
				}
			}
		}
	}()

	var (
		circuitData       CircuitData
		lastLap           = gt.Telemetry.CurrentLap()
		lapStarted        bool
		seenCoords        = make(map[gtmodels.CoordinateNorm]struct{})
		lastCoordinate    *gtmodels.Coordinate
		distanceTravelled float64
		minX, maxX        float32
		minY, maxY        float32
		minZ, maxZ        float32
		extentsInit       bool
	)
	circuitData.Name = circuitName
	circuitData.Region = circuitRegion

	// Handle interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nRecording interrupted, file save aborted.")

		os.Exit(0)
	}()

	ready := false
	for {
		if !lapStarted && !ready {
			fmt.Println("Ready...")
			ready = true
		}

		if gt.Telemetry.Flags().GamePaused {
			time.Sleep(8 * time.Millisecond)

			continue
		}

		currentLap := gt.Telemetry.CurrentLap()

		// Lap start detection
		if gt.Telemetry.IsOnCircuit() && currentLap != lastLap {
			if lapStarted {
				fmt.Println("Lap complete. Saving circuit data...")

				circuitData.LengthMeters = int(math.Round(distanceTravelled))
				saveCircuitData(outFile, &circuitData, minX, maxX, minY, maxY, minZ, maxZ, extentsInit)

				os.Exit(0)
			} else {
				fmt.Println("Lap start detected.")
				lapStarted = true
				lastLap = currentLap
			}
		}

		if lapStarted {
			coordinate := gt.Telemetry.PositionalMapCoordinates()

			if circuitData.Coordinates.StartingLine == (gtmodels.CoordinateNorm{}) {
				circuitData.Coordinates.StartingLine = gtcircuits.NormaliseStartLineCoordinate(coordinate)
			}

			coordinateNorm := gtcircuits.NormaliseCircuitCoordinate(coordinate)

			if _, exists := seenCoords[coordinateNorm]; !exists {
				circuitData.Coordinates.Circuit = append(circuitData.Coordinates.Circuit, coordinateNorm)
				seenCoords[coordinateNorm] = struct{}{}

				// Update min/max extents as new points are added
				if !extentsInit {
					minX, maxX = coordinate.X, coordinate.X
					minY, maxY = coordinate.Y, coordinate.Y
					minZ, maxZ = coordinate.Z, coordinate.Z
					extentsInit = true
				} else {
					minX = min(minX, coordinate.X)
					maxX = max(maxX, coordinate.X)
					minY = min(minY, coordinate.Y)
					maxY = max(maxY, coordinate.Y)
					minZ = min(minZ, coordinate.Z)
					maxZ = max(maxZ, coordinate.Z)
				}

			}

			// Distance calculation (in meters)
			if lastCoordinate != nil {
				dx := float64(coordinate.X - lastCoordinate.X)
				dy := float64(coordinate.Y - lastCoordinate.Y)
				dz := float64(coordinate.Z - lastCoordinate.Z)
				dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
				if dist > 0 && dist < 500 {
					distanceTravelled += dist
				}
			}

			lastCoordinate = &coordinate
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func saveCircuitData(filename string, circuitData *CircuitData, minX, maxX, minY, maxY, minZ, maxZ float32, extentsInit bool) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(circuitData); err != nil {
		log.Fatalf("Failed to write JSON: %v", err)
	}
	fmt.Printf("Circuit data saved to %s\n\n", filename)
	fmt.Printf("Points: %d\n", len(circuitData.Coordinates.Circuit))
	fmt.Printf("Starting line: X = %d, Y = %d, Z = %d\n",
		circuitData.Coordinates.StartingLine.X,
		circuitData.Coordinates.StartingLine.Y,
		circuitData.Coordinates.StartingLine.Z,
	)
	fmt.Printf("Circuit length: %d meters\n", circuitData.LengthMeters)

	// Print circuit extents and size in xyz if available
	if extentsInit {
		fmt.Printf("Circuit extents: X = [%.0f, %.0f], Y = [%.0f, %.0f], Z = [%.0f, %.0f]\n", minX, maxX, minY, maxY, minZ, maxZ)
		fmt.Printf("Circuit size: X = %.0f, Y = %.0f, Z = %.0f\n", maxX-minX, maxY-minY, maxZ-minZ)
	}
}
