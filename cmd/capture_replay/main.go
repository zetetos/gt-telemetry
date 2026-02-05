package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	gttelemetry "github.com/zetetos/gt-telemetry/v2"
)

func main() {
	var outFile string
	flag.StringVar(&outFile, "o", "gt7-replay.gtz", "Output file name. Default: gt7-replay.gtz")
	flag.Parse()

	validateFileExtension(outFile)

	client := createTelemetryClient()

	startTelemetryClient(client)

	sigChan := setupSignalHandling()

	fmt.Println("Waiting for replay to start...")

	captureReplayLoop(client, outFile, sigChan)
}

func setupSignalHandling() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	return sigChan
}

func validateFileExtension(outFile string) {
	fileExt := outFile[len(outFile)-3:]

	if fileExt != "gtz" && fileExt != "gtr" {
		log.Fatalf("Unsupported file extension %q, use either .gtr or .gtz", fileExt)
	}
}

func createTelemetryClient() *gttelemetry.Client {
	client, err := gttelemetry.New(
		gttelemetry.Options{
			LogLevel: "info",
		},
	)
	if err != nil {
		log.Fatalf("Error creating GT client: %v", err)
	}

	return client
}

func startTelemetryClient(client *gttelemetry.Client) {
	go func() {
		for {
			recoverable, err := client.Run(context.Background())
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
}

func captureReplayLoop(client *gttelemetry.Client, outFile string, sigChan chan os.Signal) {
	framesCaptured := 0
	lastTimeOfDay := time.Duration(0)
	sequenceID := ^uint32(0)
	startTime := time.Duration(0)
	recordingStarted := false

	for {
		select {
		case <-sigChan:
			handleInterrupt(client)

			return
		default:
			if shouldSkipFrame(sequenceID, client) {
				time.Sleep(4 * time.Millisecond)

				continue
			}

			sequenceID = client.Telemetry.SequenceID()

			if isFirstFrame(lastTimeOfDay) {
				lastTimeOfDay = client.Telemetry.TimeOfDay()

				continue
			}

			if shouldStopRecording(recordingStarted, client, startTime, framesCaptured) {
				handleReplayRestart(client, framesCaptured)

				return
			}

			if shouldStartRecording(recordingStarted, client, lastTimeOfDay) {
				startRecording(client, outFile)

				startTime = client.Telemetry.TimeOfDay()
				recordingStarted = true

				printSessionInfo(client, outFile)
			}

			if recordingStarted {
				framesCaptured++
				lastTimeOfDay = client.Telemetry.TimeOfDay()

				printFrameCount(framesCaptured)
			}

			time.Sleep(4 * time.Millisecond)
		}
	}
}

func shouldSkipFrame(sequenceID uint32, client *gttelemetry.Client) bool {
	return sequenceID == client.Telemetry.SequenceID()
}

func isFirstFrame(lastTimeOfDay time.Duration) bool {
	return lastTimeOfDay == time.Duration(0)
}

func shouldStopRecording(recordingStarted bool, client *gttelemetry.Client, startTime time.Duration, framesCaptured int) bool {
	return recordingStarted && client.Telemetry.TimeOfDay() <= startTime && framesCaptured >= 60
}

func handleReplayRestart(client *gttelemetry.Client, framesCaptured int) {
	fmt.Println("Replay restart detected, stopping recording...")
	stopRecordingIfNeeded(client)
	fmt.Printf("Capture complete, total frames: %d\n", framesCaptured)
}

func shouldStartRecording(recordingStarted bool, client *gttelemetry.Client, lastTimeOfDay time.Duration) bool {
	return !recordingStarted && client.Telemetry.TimeOfDay() != lastTimeOfDay
}

func printFrameCount(framesCaptured int) {
	if framesCaptured%300 == 0 {
		fmt.Printf("%d frames captured\n", framesCaptured)
	}
}

func handleInterrupt(client *gttelemetry.Client) {
	fmt.Println("\nInterrupt received, stopping recording...")
	stopRecordingIfNeeded(client)
}

func stopRecordingIfNeeded(client *gttelemetry.Client) {
	if client.IsRecording() {
		err := client.StopRecording()
		if err != nil {
			log.Printf("Error stopping recording: %v", err)
		}
	}
}

func startRecording(client *gttelemetry.Client, outFile string) {
	err := client.StartRecording(outFile)
	if err != nil {
		log.Fatalf("Failed to start recording: %v", err)
	}
}

func printSessionInfo(client *gttelemetry.Client, outFile string) {
	fmt.Printf("Starting capture to %s\n", outFile)
	fmt.Printf("Frame size: %d bytes\n", len(client.DecipheredPacket))
	fmt.Printf("Time of day: %+v\n", client.Telemetry.TimeOfDay())
	fmt.Printf("Vehicle: %s %s\n",
		client.Telemetry.VehicleManufacturer(),
		client.Telemetry.VehicleModel())
}
