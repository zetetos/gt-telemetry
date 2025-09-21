package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	gttelemetry "github.com/zetetos/gt-telemetry"
)

func main() {
	var outFile string

	flag.StringVar(&outFile, "o", "gt7-replay.gtz", "Output file name. Default: gt7-replay.gtz")
	flag.Parse()

	fh, err := os.Create(outFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()

	var buffer io.Writer
	fileExt := outFile[len(outFile)-3:]
	switch fileExt {
	case "gtz":
		buffer, err = gzip.NewWriterLevel(fh, gzip.BestCompression)
		if err != nil {
			log.Fatal(err.Error())
		}
		buffer.(*gzip.Writer).Comment = "Gran Turismo 7 Telemetry Replay"
	case "gtr":
		buffer = fh
	default:
		os.Remove(outFile)
		log.Fatalf("Unsupported file extension %q, use either .gtr or .gtz", fileExt)
	}

	gt, err := gttelemetry.New(gttelemetry.Options{})
	if err != nil {
		fmt.Println("Error creating GT client: ", err)
		os.Exit(1)
	}

	go func() {
		_, _ = gt.Run()
	}()

	fmt.Println("Waiting for replay to start")

	framesCaptured := -1
	lastTimeOfDay := time.Duration(0)
	sequenceID := ^uint32(0)
	startTime := time.Duration(0)
	diff := uint32(0)
	for {
		// ignore packets that have aldready been processed
		if sequenceID == gt.Telemetry.SequenceID() {
			timer := time.NewTimer(4 * time.Millisecond)
			<-timer.C
			continue
		}

		diff = gt.Telemetry.SequenceID() - sequenceID
		sequenceID = gt.Telemetry.SequenceID()

		// Set the last time seen when the first frame is received
		if lastTimeOfDay == time.Duration(0) {
			lastTimeOfDay = gt.Telemetry.TimeOfDay()
			continue
		}

		// Finish recording when the replay restarts
		if gt.Telemetry.TimeOfDay() <= startTime {
			// The time of day sometimes flaps in the first few frames
			if framesCaptured < 60 {
				continue
			}

			fmt.Println("Replay restart detected")
			if b, ok := buffer.(*gzip.Writer); ok {
				if err := b.Flush(); err != nil {
					log.Fatal(err)
				}
			}
			break
		}

		// Start recording when the replay starts
		if framesCaptured == -1 && gt.Telemetry.TimeOfDay() != lastTimeOfDay {
			fmt.Printf("Starting capture, frame size: %d bytes\n", len(gt.DecipheredPacket))

			startTime = gt.Telemetry.TimeOfDay()
			framesCaptured = 0

			extraData := fmt.Sprintf("Time of day: %+v, Manufacturer: %s, Model: %s",
				startTime,
				gt.Telemetry.VehicleManufacturer(),
				gt.Telemetry.VehicleModel(),
			)

			// add extra data to the gzip header
			if b, ok := buffer.(*gzip.Writer); ok {
				b.Extra = []byte(extraData)
			}

			fmt.Println(extraData)
		} else {
			time.Sleep(4 * time.Millisecond)
		}

		// write the frame to the file buffer
		if framesCaptured >= 0 {
			if diff > 1 {
				fmt.Printf("Dropped %d frames\n", diff-1)
			}

			_, err := buffer.Write(gt.DecipheredPacket)
			if err != nil {
				log.Fatal(err)
			}

			framesCaptured++
			lastTimeOfDay = gt.Telemetry.TimeOfDay()
		}

		timer := time.NewTimer(4 * time.Millisecond)
		<-timer.C

		if framesCaptured%300 == 0 {
			fmt.Printf("%d frames captured\n", framesCaptured)
		}
	}

	// flush and close the gzip file fuffer
	if b, ok := buffer.(*gzip.Writer); ok {
		b.Flush()
		b.Close()
	}

	fmt.Printf("Capture complete, total frames: %d\n", framesCaptured)
}
