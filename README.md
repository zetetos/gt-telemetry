# GT Telemetry #

[![Build Status](https://github.com/zetetos/gt-telemetry/actions/workflows/main.yml/badge.svg?branch=main)](https://github.com/zetetos/gt-telemetry/actions?query=branch%3Amain)
[![codecov](https://codecov.io/gh/vwhitteron/gt-telemetry/branch/main/graph/badge.svg)](https://codecov.io/gh/vwhitteron/gt-telemetry)
[![Go Report Card](https://goreportcard.com/badge/github.com/zetetos/gt-telemetry)](https://goreportcard.com/report/github.com/zetetos/gt-telemetry)

GT Telemetry is a module for reading Gran Turismo race telemetry streams in Go.

## Features

* Support for all known fields contained within the telemetry data packet.
* Support for all current telemetry formats (A, B and ~)
* Access data in both metric and imperial units.
* Recording capability to capture telemetry data to files.
* An additional field for the differential gear ratio is computed based on the rolling wheel diameter of the driven wheels.
* A vehicle inventory database with methods for providing the following information on a given vehicle ID:
  * Manufacturer
  * Model
  * Year
  * Drivetrain
  * Aspiration
  * Type (racing or street)
  * Racing category
  * Open cockpit exposure
* A circuit inventory database with methods for matching a circuit based on a given coordinate and providing the following infromation:
  * Name
  * Length
  * Region

[![asciicast](https://asciinema.org/a/fSBcGOR1EPjhTCMFLY0gHP0Py.svg)](https://asciinema.org/a/fSBcGOR1EPjhTCMFLY0gHP0Py)

## Installation ##

To start using gt-telemetry, install Go 1.24.6 or above. From your project, run the following command to retrieve the module:

```bash
go get github.com/zetetos/gt-telemetry
```

## Usage ##

Construct a new GT client and start reading the telemetry stream. All configuration fields in the example are optional and use the default values that would be used when not provided.

```go
import "github.com/zetetos/gt-telemetry"

main() {
    options := gttelemetry.Options{
        Source: "udp://255.255.255.255:33739"
        Format: telemetryformat.Addendum2,
        LogLevel: "warn",
        StatsEnabled: false,
        VehicleDB: "./pkg/vehicles/vehicles.json",
        CircuitDB: "./pkg/circuits/cicuits.json",
    }
    gtclient, _ := gttelemetry.New(options)
    go func() {
        err, recoverable = gtclient.Run()
        if err != nil {
            if recoverable {
                log.Printf("Recoverable error: %s", err.Error())
            } else {
                log.Fatalf("Fatal client error: %s", err.Error())
            }
        }
    }()
}
```

_If the PlayStation is on the same network segment then you will probably find that the default broadcast address `255.255.255.255` will be sufficient to start reading data. If it does not work then enter the IP address of the PlayStation device instead._

Read some data from the stream:

```go
    fmt.Printf("Sequence ID:  %6d    %3.0f kph  %5.0f rpm\n",
        gt.Telemetry.SequenceID(),
        gt.Telemetry.GroundSpeedKPH(),
        gt.Telemetry.EngineRPM(),
    )
```

### Replay files ###

Offline saves of replay files can also be used to read in telemetry data. Files can be in either plain (`*.gtr`) or compressed (`*.gtz`) format.

Read telemetry from a replay file by setting the `Source` value in the `gttelemetry.Options` to a file URL, like so:

```go
config := gttelemetry.Options{
    Source: "file://data/replays/demo.gtz"
}
```

#### Saving a replay to a file ####

Replays can be captured and saved to a file using `cmd/capture_replay/main.go`. Captures will be saved in plain or compressed formats according to the file extension as mentioned in the section above.

A replay can be saved to a default file by running:

```bash
make run/capture-replay
```

Alternatively, the replay can be captured to a compressed file with a different name and location by running:

```bash
go run cmd/capture_replay/main.go -o /path/to/replay-file.gtz
```

#### Recording telemetry data programmatically ####

The GT Telemetry client provides built-in methods for recording telemetry data to files during runtime. This allows you to start and stop recording at any point in your application.

**Basic usage:**

```go
// Create a telemetry client.
client, err := gttelemetry.New(gttelemetry.Options{
    Source: "udp://255.255.255.255:33739", // Live telemetry
})
if err != nil {
    log.Fatal(err)
}

// Start recording to a compressed file.
err = client.StartRecording("my_recording.gtz")
if err != nil {
    log.Fatal(err)
}

// Your application logic here...
// Telemetry data will be automatically recorded at the same time.

// Stop recording
err = client.StopRecording()
if err != nil {
    log.Fatal(err)
}

// Check if currently recording.
if client.IsRecording() {
    fmt.Println("Recording is active")
}
```

**Supported file formats:**
- `.gtr` - Plain binary telemetry data
- `.gtz` - Compressed telemetry data (recommended for storage efficiency)

### Vehicle Inventory Management ###

The `inventory` CLI tool allows you to import and export vehicle inventory data between JSON and CSV formats, and manage vehicle entries with interactive add, edit, and delete operations. The tool uses action-based commands and outputs to stdout, making it compatible with Unix pipes and redirections.

The output format is automatically determined by the input file extension:
- `.json` files are converted to CSV format
- `.csv` files are converted to JSON format


#### Adding new vehicles interactively ####

```bash
go run cmd/inventory/main.go add internal/vehicles/inventory.json
```

The tool will prompt for each field and display a summary before saving. It also checks for duplicate vehicle IDs and provides confirmation prompts.

#### Editing existing vehicles ####

```bash
go run cmd/inventory/main.go edit internal/vehicles/inventory.json 3267
```

The edit action loads the existing vehicle data and allows you to modify any field. Current values are shown in brackets, and pressing Enter without input keeps the existing value.

#### Deleting vehicles ####

```bash
go run cmd/inventory/main.go delete internal/vehicles/inventory.json 3267
```

The delete action shows the vehicle details and asks for confirmation before removal.

#### Converting JSON to CSV ####

```bash
go run cmd/inventory/main.go convert internal/vehicles/inventory.json
```

#### Converting CSV to JSON ####

```bash
go run cmd/inventory/main.go convert data/inventory.csv
```

#### CSV Format ####

The CSV format includes the following columns:
- CarID: Unique vehicle identifier
- Manufacturer: Vehicle manufacturer
- Model: Vehicle model name
- Year: Model year (0 for unknown)
- OpenCockpit: Boolean indicating if the vehicle has an open cockpit
- CarType: Vehicle type (street, race)
- Category: Racing category (e.g., Gr.1, Gr.3, Gr.4, Gr.B)
- Drivetrain: Drivetrain type (FR, FF, MR, RR, 4WD)
- Aspiration: Engine aspiration (NA, TC, SC, EV, etc.)
- EngineLayout: Engine layout configuration
- EngineBankAngle: Engine cylinder bank angle in degrees
- EngineCrankPlaneAngle: Engine crank plane angle in degrees
```

## Examples ##

The [examples](./examples) directory contains example code for accessing most data made available by the library. The example app shown at the top of this page can be run against a replay file with the following command:

```bash
make run
```

The example code can also read live telemetry data from a PlayStation by removing the `Source` field in the `GTClientOpts`.

## Acknowledgements ##
Special thanks to [Nenkai](https://github.com/Nenkai) for the excellent work documenting the Gran Turismo telemetry protocol.