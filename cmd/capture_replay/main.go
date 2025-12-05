package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	gttelemetry "github.com/zetetos/gt-telemetry"
)

func main() {
	var outFile string

	flag.StringVar(&outFile, "o", "gt7-replay.gtz", "Output file name. Default: gt7-replay.gtz")
	flag.Parse()

	// Validate file extension
	fileExt := outFile[len(outFile)-3:]
	if fileExt != "gtz" && fileExt != "gtr" {
		log.Fatalf("Unsupported file extension %q, use either .gtr or .gtz", fileExt)
	}

	// Create telemetry client
	client, err := gttelemetry.New(gttelemetry.Options{
		LogLevel: "info",
	})
	if err != nil {
		log.Fatalf("Error creating GT client: %v", err)
	}

	// Start telemetry client in background
	go func() {
		for {
			err, recoverable := client.Run()
			if err != nil {
				if recoverable {
					log.Printf("Recoverable error: %s", err.Error())
					time.Sleep(1 * time.Second)
				} else {
					log.Printf("Telemetry client finished: %s", err.Error())

					return
				}
			}
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Waiting for replay to start...")

	framesCaptured := 0
	lastTimeOfDay := time.Duration(0)
	sequenceID := ^uint32(0)
	startTime := time.Duration(0)
	recordingStarted := false

	// Main capture loop
	for {
		select {
		case <-sigChan:
			fmt.Println("\nInterrupt received, stopping recording...")

			if client.IsRecording() {
				err := client.StopRecording()
				if err != nil {
					log.Printf("Error stopping recording: %v", err)
				}
			}

			return
		default:
			// Check if we have new telemetry data
			if sequenceID == client.Telemetry.SequenceID() {
				time.Sleep(4 * time.Millisecond)

				continue
			}

			sequenceID = client.Telemetry.SequenceID()

			// Set the initial time when first frame is received
			if lastTimeOfDay == time.Duration(0) {
				lastTimeOfDay = client.Telemetry.TimeOfDay()

				continue
			}

			// Detect replay restart (time goes backwards significantly)
			if recordingStarted && client.Telemetry.TimeOfDay() <= startTime {
				// Allow for small time fluctuations in the first few frames
				if framesCaptured < 60 {
					continue
				}

				fmt.Println("Replay restart detected, stopping recording...")

				err := client.StopRecording()
				if err != nil {
					log.Printf("Error stopping recording: %v", err)
				} else {
					fmt.Printf("Capture complete, total frames: %d\n", framesCaptured)
				}

				return
			}

			// Start recording when replay movement is detected
			if !recordingStarted && client.Telemetry.TimeOfDay() != lastTimeOfDay {
				fmt.Printf("Starting capture to %s\n", outFile)
				fmt.Printf("Frame size: %d bytes\n", len(client.DecipheredPacket))

				// Display session info
				fmt.Printf("Time of day: %+v\n", client.Telemetry.TimeOfDay())
				fmt.Printf("Vehicle: %s %s\n",
					client.Telemetry.VehicleManufacturer(),
					client.Telemetry.VehicleModel())

				// Start recording using the client's built-in functionality
				err := client.StartRecording(outFile)
				if err != nil {
					log.Fatalf("Failed to start recording: %v", err)
				}

				startTime = client.Telemetry.TimeOfDay()
				recordingStarted = true
			}

			// Update counters if recording
			if recordingStarted {
				framesCaptured++
				lastTimeOfDay = client.Telemetry.TimeOfDay()

				// Progress indicator
				if framesCaptured%300 == 0 {
					fmt.Printf("%d frames captured\n", framesCaptured)
				}
			}

			time.Sleep(4 * time.Millisecond)
		}
	}
}
