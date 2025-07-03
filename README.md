# GT Telemetry #

[![Build Status](https://github.com/zetetos/gt-telemetry/actions/workflows/main.yml/badge.svg?branch=main)](https://github.com/zetetos/gt-telemetry/actions?query=branch%3Amain)
[![codecov](https://codecov.io/gh/vwhitteron/gt-telemetry/branch/main/graph/badge.svg)](https://codecov.io/gh/vwhitteron/gt-telemetry)
[![Go Report Card](https://goreportcard.com/badge/github.com/zetetos/gt-telemetry)](https://goreportcard.com/report/github.com/zetetos/gt-telemetry)

GT Telemetry is a module for reading Gran Turismo race telemetry streams in Go.

## Features

* Support for all fields contained within the telemetry data packet.
* Access data in both metric and imperial units.
* An additional field for the differential gear ratio is computed based on the rolling wheel diameter of the driven wheels.
* A vehicle inventory database for providing the follwing information on a given vehicle ID:
  * Manufacturer
  * Model
  * Year
  * Drivetrain
  * Aspiration
  * Type (racing or street)
  * Racing category
  * Open cockpit exposure

## Installation ##

To start using gt-telemetry, install Go 1.21 or above. From your project, run the following command to retrieve the module:

```bash
go get github.com/zetetos/gt-telemetry
```

## Usage ##

```go
import telemetry_client "github.com/zetetos/gt-telemetry"
```

Construct a new GT client and start reading the telemetry stream. All configuration fields are optional with the default values show in the example.

```go
config := telemetry_client.GTClientOpts{
    Source: "udp://255.255.255.255:33739"
    Format: telemetrysrc.TelemetryFormatA,
    LogLevel: "warn",
    StatsEnabled: false,
    VehicleDB: "./internal/vehicles/inventory.json",
}
gt, _ := telemetry_client.NewGTClient(config)
go gt.Run()
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

Read telemetry from a replay file by setting the `Source` value in the `GTClientOpts` to a file URL, like so:

```go
config := telemetry_client.GTClientOpts{
    Source: "file://examples/simple/replay.gtz"
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

## Examples ##

The [examples](./examples) directory contains example code for accessing most data made available by the library. The telemetry data from a sample saved replay can be viewed by running:

```bash
make run/live
```

The example code can also read live telemetry data from a PlayStation by removing the `Source` field in the `GTClientOpts`.

[![asciicast](https://asciinema.org/a/fSBcGOR1EPjhTCMFLY0gHP0Py.svg)](https://asciinema.org/a/fSBcGOR1EPjhTCMFLY0gHP0Py)

## Acknowledgements ##
Special thanks to [Nenkai](https://github.com/Nenkai) for the excellent work documenting the Gran Turismo telemetry protocol.