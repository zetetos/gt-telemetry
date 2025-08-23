package telemetry

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/kaitai-io/kaitai_struct_go_runtime/kaitai"
	"github.com/rs/zerolog"

	"github.com/zetetos/gt-telemetry/internal/gttelemetry"
	"github.com/zetetos/gt-telemetry/internal/telemetrysrc"
	"github.com/zetetos/gt-telemetry/internal/vehicles"
)

type statistics struct {
	enabled           bool
	decodeTimeLast    time.Duration
	packetRateLast    time.Time
	packetIDLast      uint32
	DecodeTimeAvg     time.Duration
	DecodeTimeMax     time.Duration
	PacketRateAvg     int
	PacketRateCurrent int
	PacketRateMax     int
	PacketsDropped    int
	PacketsInvalid    int
	PacketsTotal      int
	PacketSize        int
}

type GTClientOpts struct {
	Source       string
	Format       telemetrysrc.TelemetryFormat
	LogLevel     string
	Logger       *zerolog.Logger
	StatsEnabled bool
	VehicleDB    string
}

type GTClient struct {
	log              zerolog.Logger
	source           string
	format           telemetrysrc.TelemetryFormat
	DecipheredPacket []byte
	Finished         bool
	Statistics       *statistics
	Telemetry        *transformer
}

func NewGTClient(opts GTClientOpts) (*GTClient, error) {
	var log zerolog.Logger
	if opts.Logger != nil {
		log = *opts.Logger
	} else {
		log = zerolog.New(os.Stdout).With().Timestamp().Logger()

		switch opts.LogLevel {
		case "trace":
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "fatal":
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		case "panic":
			zerolog.SetGlobalLevel(zerolog.PanicLevel)
		case "off":
			zerolog.SetGlobalLevel(zerolog.Disabled)
		case "":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		default:
			opts.LogLevel = "warn"
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
			log.Warn().Str("log_level", opts.LogLevel).Msg("unknown log level, setting level to warn")
		}
	}

	if opts.Source == "" {
		opts.Source = "udp://255.255.255.255:33739"
	}

	if opts.Format == "" {
		opts.Format = telemetrysrc.TelemetryFormatTilde
	}

	var inventoryJSON []byte
	if opts.VehicleDB != "" {
		var err error
		inventoryJSON, err = os.ReadFile(opts.VehicleDB)
		if err != nil {
			return nil, fmt.Errorf("reading vehicle DB from file: %w", err)
		}
	}

	inventory, err := vehicles.NewInventory(inventoryJSON)
	if err != nil {
		return nil, fmt.Errorf("setting up new inventory: %w", err)
	}

	return &GTClient{
		log:              log,
		source:           opts.Source,
		format:           opts.Format,
		DecipheredPacket: []byte{},
		Finished:         false,
		Statistics: &statistics{
			enabled:           opts.StatsEnabled,
			decodeTimeLast:    time.Duration(0),
			packetRateLast:    time.Now(),
			DecodeTimeAvg:     time.Duration(0),
			DecodeTimeMax:     time.Duration(0),
			PacketRateCurrent: 0,
			PacketRateMax:     0,
			PacketRateAvg:     0,
			PacketsTotal:      0,
			PacketsDropped:    0,
			PacketsInvalid:    0,
			packetIDLast:      0,
		},
		Telemetry: NewTransformer(inventory),
	}, nil
}

func (c *GTClient) Run() (err error, recoverable bool) {
	sourceURL, err := url.Parse(c.source)
	if err != nil {
		return fmt.Errorf("parse source URL: %w", err), false
	}

	var telemetrySource telemetrysrc.Reader

	switch sourceURL.Scheme {
	case "udp":
		host, portStr, _ := net.SplitHostPort(sourceURL.Host)
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("parse URL port: %w", err), false
		}
		telemetrySource, err = telemetrysrc.NewNetworkUDPReader(host, port, c.format, c.log)
		if err != nil {
			return fmt.Errorf("setup UDP reader: %w", err), true
		}
	case "file":
		telemetrySource, err = telemetrysrc.NewFileReader(sourceURL.Host+sourceURL.Path, c.log)
		if err != nil {
			return fmt.Errorf("setup file reader: %w", err), false
		}
	default:
		return fmt.Errorf("invalid URL scheme %q", sourceURL.Scheme), false
	}

	rawTelemetry := gttelemetry.NewGranTurismoTelemetry()

readTelemetry:
	for {
		bufLen, buffer, err := telemetrySource.Read()
		if err != nil {
			if err.Error() == "bufio.Scanner: SplitFunc returns advance count beyond input" {
				c.Finished = true

				continue readTelemetry
			}

			c.log.Debug().Err(err).Msg("failed to receive telemetry")

			continue readTelemetry
		}

		if len(buffer[:bufLen]) == 0 {
			c.log.Debug().Msg("no data received")

			continue readTelemetry
		}

		decodeStart := time.Now()

		c.DecipheredPacket = buffer[:bufLen]

		reader := bytes.NewReader(c.DecipheredPacket)
		stream := kaitai.NewStream(reader)

	parseTelemetry:
		for {
			err = rawTelemetry.Read(stream, nil, nil)
			if err != nil {
				if err.Error() == "EOF" {
					break parseTelemetry
				}
				c.log.Error().Err(err).Msg("failed to parse telemetry")
				c.Statistics.PacketsInvalid++
			}

			c.Telemetry.RawTelemetry = *rawTelemetry

			c.Statistics.decodeTimeLast = time.Since(decodeStart)
			c.collectStats()

			timer := time.NewTimer(4 * time.Millisecond)
			<-timer.C
		}
	}
}

func (c *GTClient) collectStats() {
	if !c.Statistics.enabled {
		return
	}

	c.Statistics.PacketsTotal++

	if c.Statistics.packetIDLast != c.Telemetry.SequenceID() {
		c.Statistics.PacketSize, _ = c.Telemetry.RawTelemetry.PacketSize()

		if c.Statistics.packetIDLast == 0 {
			c.Statistics.packetIDLast = c.Telemetry.SequenceID()
			return
		}

		c.Statistics.DecodeTimeAvg = (c.Statistics.DecodeTimeAvg + c.Statistics.decodeTimeLast) / 2
		if c.Statistics.decodeTimeLast > c.Statistics.DecodeTimeMax {
			c.Statistics.DecodeTimeMax = c.Statistics.decodeTimeLast
		}

		delta := int(c.Telemetry.SequenceID() - c.Statistics.packetIDLast)
		if delta > 1 {
			c.log.Warn().Int("count", delta-1).Msg("packets dropped")
			c.Statistics.PacketsDropped += int(delta - 1)
		} else if delta < 0 {
			c.log.Warn().Int("count", 1).Msg("packets delayed")
		}

		c.Statistics.packetIDLast = c.Telemetry.SequenceID()

		if c.Telemetry.SequenceID()%10 == 0 {
			rate := time.Since(c.Statistics.packetRateLast)
			c.Statistics.PacketRateCurrent = int(10 / rate.Seconds())
			c.Statistics.packetRateLast = time.Now()
			c.Statistics.PacketRateAvg = (c.Statistics.PacketRateAvg + c.Statistics.PacketRateCurrent) / 2
			if c.Statistics.PacketRateCurrent > c.Statistics.PacketRateMax {
				c.Statistics.PacketRateMax = c.Statistics.PacketRateCurrent
			}
		}
	}

}
