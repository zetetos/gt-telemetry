package gttelemetry

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kaitai-io/kaitai_struct_go_runtime/kaitai"
	"github.com/rs/zerolog"

	"github.com/zetetos/gt-telemetry/internal/reader"
	"github.com/zetetos/gt-telemetry/internal/telemetry"
	"github.com/zetetos/gt-telemetry/pkg/circuits"
	"github.com/zetetos/gt-telemetry/pkg/models"
	"github.com/zetetos/gt-telemetry/pkg/vehicles"
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

type Options struct {
	Source       string
	Format       models.Name
	LogLevel     string
	Logger       *zerolog.Logger
	StatsEnabled bool
	CircuitDB    string
	VehicleDB    string
}

type Client struct {
	log              zerolog.Logger
	source           string
	format           models.Name
	DecipheredPacket []byte
	Finished         bool
	Statistics       *statistics
	Telemetry        *transformer
	CircuitDB        *circuits.CircuitDB

	// Recording state
	recordingMutex  sync.RWMutex
	recordingFile   io.WriteCloser
	recordingBuffer io.Writer
	isRecording     bool
}

func New(opts Options) (*Client, error) {
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
		opts.Format = models.Addendum2
	}

	var circuitsJSON []byte
	if opts.CircuitDB != "" {
		var err error
		circuitsJSON, err = os.ReadFile(opts.CircuitDB)
		if err != nil {
			return nil, fmt.Errorf("reading circuit DB from file: %w", err)
		}
	}

	circuitDB, err := circuits.NewDB(circuitsJSON)
	if err != nil {
		return nil, fmt.Errorf("setting up new circuit database: %w", err)
	}

	var vehiclesJSON []byte
	if opts.VehicleDB != "" {
		var err error
		vehiclesJSON, err = os.ReadFile(opts.VehicleDB)
		if err != nil {
			return nil, fmt.Errorf("reading vehicle DB from file: %w", err)
		}
	}

	vehicleDB, err := vehicles.NewDB(vehiclesJSON)
	if err != nil {
		return nil, fmt.Errorf("setting up new vehicle database: %w", err)
	}

	return &Client{
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
		Telemetry: NewTransformer(vehicleDB),
		CircuitDB: circuitDB,
	}, nil
}

func (c *Client) Run() (err error, recoverable bool) {
	sourceURL, err := url.Parse(c.source)
	if err != nil {
		return fmt.Errorf("parse source URL: %w", err), false
	}

	var telemetryReader reader.Reader

	switch sourceURL.Scheme {
	case "udp":
		host, portStr, _ := net.SplitHostPort(sourceURL.Host)
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("parse URL port: %w", err), false
		}
		telemetryReader, err = reader.NewUDPReader(host, port, c.format, c.log)
		if err != nil {
			return fmt.Errorf("setup UDP reader: %w", err), true
		}
	case "file":
		telemetryReader, err = reader.NewFileReader(sourceURL.Host+sourceURL.Path, c.log)
		if err != nil {
			return fmt.Errorf("setup file reader: %w", err), false
		}
	default:
		return fmt.Errorf("invalid URL scheme %q", sourceURL.Scheme), false
	}

	rawTelemetry := telemetry.NewGranTurismoTelemetry()

readTelemetry:
	for {
		bufLen, buffer, err := telemetryReader.Read()
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

			c.recordPacket()

			timer := time.NewTimer(4 * time.Millisecond)
			<-timer.C
		}
	}
}

func (c *Client) collectStats() {
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

// StartRecording starts recording telemetry data to the specified file path.
// Supports both plain (.gtr) and compressed (.gtz) formats based on file extension.
func (c *Client) StartRecording(filePath string) error {
	c.recordingMutex.Lock()
	defer c.recordingMutex.Unlock()

	if c.isRecording {
		return fmt.Errorf("recording already in progress")
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create recording file: %w", err)
	}

	// Determine format based on file extension
	var buffer io.Writer
	fileExt := filePath[len(filePath)-3:]
	switch fileExt {
	case "gtz":
		gzipWriter, err := gzip.NewWriterLevel(file, gzip.BestCompression)
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to create gzip writer: %w", err)
		}
		gzipWriter.Comment = "Gran Turismo Telemetry Recording"
		buffer = gzipWriter
		c.recordingFile = &gzipFileWrapper{file: file, gzipWriter: gzipWriter}
	case "gtr":
		buffer = file
		c.recordingFile = file
	default:
		file.Close()
		return fmt.Errorf("unsupported file extension %q, use either .gtr or .gtz", fileExt)
	}

	c.recordingBuffer = buffer
	c.isRecording = true

	c.log.Info().Str("file", filePath).Msg("started recording telemetry data")
	return nil
}

// StopRecording stops the current recording and closes the file.
func (c *Client) StopRecording() error {
	c.recordingMutex.Lock()
	defer c.recordingMutex.Unlock()

	if !c.isRecording {
		return fmt.Errorf("no recording in progress")
	}

	// Flush and close the file
	if err := c.recordingFile.Close(); err != nil {
		c.log.Error().Err(err).Msg("error closing recording file")
		return fmt.Errorf("failed to close recording file: %w", err)
	}

	c.recordingFile = nil
	c.recordingBuffer = nil
	c.isRecording = false

	c.log.Info().Msg("stopped recording telemetry data")
	return nil
}

// IsRecording returns true if telemetry data is currently being recorded.
func (c *Client) IsRecording() bool {
	c.recordingMutex.RLock()
	defer c.recordingMutex.RUnlock()
	return c.isRecording
}

// recordPacket writes the current packet to the recording file if recording is active.
func (c *Client) recordPacket() {
	c.recordingMutex.RLock()
	defer c.recordingMutex.RUnlock()

	if !c.isRecording || c.recordingBuffer == nil {
		return
	}

	if len(c.DecipheredPacket) == 0 {
		return
	}

	_, err := c.recordingBuffer.Write(c.DecipheredPacket)
	if err != nil {
		c.log.Error().Err(err).Msg("failed to write packet to recording file")
	}
}

// gzipFileWrapper wraps a gzip writer and file to handle proper closing
type gzipFileWrapper struct {
	file       *os.File
	gzipWriter *gzip.Writer
}

func (g *gzipFileWrapper) Write(p []byte) (n int, err error) {
	return g.gzipWriter.Write(p)
}

func (g *gzipFileWrapper) Close() error {
	// First close the gzip writer to flush any remaining data
	if err := g.gzipWriter.Close(); err != nil {
		g.file.Close() // Still try to close the file
		return err
	}
	// Then close the underlying file
	return g.file.Close()
}
