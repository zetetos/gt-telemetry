package gttelemetry

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kaitai-io/kaitai_struct_go_runtime/kaitai"
	"github.com/rs/zerolog"
	"github.com/zetetos/gt-telemetry/v2/internal/reader"
	"github.com/zetetos/gt-telemetry/v2/internal/telemetry"
	"github.com/zetetos/gt-telemetry/v2/pkg/circuits"
	"github.com/zetetos/gt-telemetry/v2/pkg/models"
	"github.com/zetetos/gt-telemetry/v2/pkg/vehicles"
)

const (
	autoDiscoveryURL = "udp://255.255.255.255:33739"
	defaultCachePath = "data/cache"
)

// recordingState represents the game state at the time recording was started.
type recordingState int

const (
	recordingStateNone      recordingState = iota
	recordingStateOnCircuit                // vehicle is live on circuit, not in any menu
	recordingStateRaceMenu                 // vehicle is loaded but in the race menu
)

var (
	ErrInvalidURLScheme           = reader.ErrInvalidURLScheme
	ErrNotAFileSource             = errors.New("Scan() requires a file:// source")
	ErrRecordingAlreadyInProgress = errors.New("recording already in progress")
	ErrUnsupportedFileExtension   = errors.New("unsupported file extension, use either .gtr or .gtz")
	ErrNoRecordingInProgress      = errors.New("no recording in progress")
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
	Source        string
	Format        models.Name
	LogLevel      string
	Logger        *zerolog.Logger
	StatsEnabled  bool
	CachePath     string
	UpdateBaseURL string
	VehicleDB     string // TODO: remove in future release, overrides can be added to cache
}

type Client struct {
	log              zerolog.Logger
	source           string
	format           models.Name
	DecipheredPacket []byte
	Finished         bool
	Statistics       *statistics
	Telemetry        *Transformer
	CircuitDB        *circuits.CircuitDB

	// Recording state
	recordingMutex     sync.RWMutex
	recordingFile      io.WriteCloser
	recordingBuffer    io.Writer
	isRecording        bool
	recordingInitState recordingState
}

func New(opts Options) (*Client, error) {
	logger := setupLogger(opts)

	if opts.Source == "" {
		opts.Source = autoDiscoveryURL
	}

	if opts.Format == "" {
		opts.Format = models.Addendum3
	}

	circuitDB, err := loadCircuitDB(opts.CachePath, opts.UpdateBaseURL, &logger)
	if err != nil {
		return nil, err
	}

	vehicleDB, err := loadVehicleDB(opts.VehicleDB, opts.CachePath, opts.UpdateBaseURL, &logger)
	if err != nil {
		return nil, err
	}

	if opts.UpdateBaseURL != "" {
		go checkForUpdates(context.Background(), opts.UpdateBaseURL, vehicleDB, circuitDB, &logger)
	}

	return &Client{
		log:              logger,
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

// setupLogger initializes the zerolog.Logger based on options.
func setupLogger(opts Options) zerolog.Logger {
	if opts.Logger != nil {
		return *opts.Logger
	}

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	logLevel, err := zerolog.ParseLevel(opts.LogLevel)
	if err != nil {
		logLevel = zerolog.WarnLevel

		log.Warn().Str("log_level", opts.LogLevel).Msg("unknown log level, setting level to warn")
	}

	zerolog.SetGlobalLevel(logLevel)

	return log
}

// loadCircuitDB loads the circuit database from embedded inventory files,
// overlaid by any cached circuit files found in cachePath.
func loadCircuitDB(cachePath, updateBaseURL string, logger *zerolog.Logger) (*circuits.CircuitDB, error) {
	if cachePath == "" {
		cachePath = filepath.Join(defaultCachePath, "circuits")
	}

	circuitDB, err := circuits.NewDB(circuits.CircuitDBOptions{
		CacheDir:      cachePath,
		UpdateBaseURL: updateBaseURL,
		Logger:        logger,
	})
	if err != nil {
		return nil, fmt.Errorf("setting up new circuit database: %w", err)
	}

	return circuitDB, nil
}

// loadVehicleDB loads the vehicle database from file if provided.
func loadVehicleDB(dbPath string, cachePath string, updateURL string, logger *zerolog.Logger) (*vehicles.VehicleDB, error) {
	var vehiclesJSON []byte

	var err error

	if dbPath != "" {
		vehiclesJSON, err = os.ReadFile(dbPath)
		if err != nil {
			return nil, fmt.Errorf("reading vehicle DB from file: %w", err)
		}
	}

	if cachePath == "" {
		cachePath = filepath.Join(defaultCachePath, "vehicles")
	}

	DBOptions := vehicles.DBOptions{
		CacheDir:      cachePath,
		UpdateBaseURL: updateURL,
		Logger:        logger,
	}

	vehicleDB, err := vehicles.NewDB(vehiclesJSON, DBOptions)
	if err != nil {
		return nil, fmt.Errorf("setting up new vehicle database: %w", err)
	}

	return vehicleDB, nil
}

// checkForUpdates fetches the remote version.json and triggers vehicle/circuit
// updates only when the remote data is newer than the local inventory.
func checkForUpdates(ctx context.Context, updateBaseURL string, vehicleDB *vehicles.VehicleDB, circuitDB *circuits.CircuitDB, logger *zerolog.Logger) {
	version, err := fetchVersion(ctx, updateBaseURL)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to fetch remote version")

		return
	}

	if version.Vehicles.LastModified.After(vehicleDB.LatestModified()) {
		vehicleDB.CheckForUpdates(ctx)
	}

	if version.Circuits.LastModified.After(circuitDB.LatestModified()) {
		circuitDB.CheckForUpdates(ctx)
	}
}

// Stream starts the telemetry client to read and process a live data stream.
// The context parameter allows for graceful cancellation.
func (c *Client) Stream(ctx context.Context) (recoverable bool, err error) {
	// Ensure recording is stopped when Stream exits
	defer func() {
		if c.IsRecording() {
			stopErr := c.StopRecording()
			if stopErr != nil {
				c.log.Error().Err(stopErr).Msg("failed to stop recording on exit")
			}
		}
	}()

	sourceURL, err := url.Parse(c.source)
	if err != nil {
		return false, fmt.Errorf("parse source URL: %w", err)
	}

	readerCfg, err := reader.New(sourceURL, c.format, c.log)
	if err != nil {
		return readerCfg.Recoverable, err
	}

	recoverable = readerCfg.Recoverable
	telemetryReader := readerCfg.Reader
	throttle := readerCfg.Throttle

	// Ensure the reader is closed when Stream exits
	defer func() {
		closeErr := telemetryReader.Close()
		if closeErr != nil {
			c.log.Error().Err(closeErr).Msg("failed to close telemetry reader")
		}
	}()

	// Watch for context cancellation and close the reader immediately to unblock ReadFromUDP
	go func() {
		<-ctx.Done()
		c.log.Debug().Msg("context cancelled, closing telemetry reader to unblock read")

		_ = telemetryReader.Close()
	}()

	rawTelemetry := telemetry.NewGranTurismoTelemetry()

	for {
		select {
		case <-ctx.Done():
			c.log.Debug().Msg("context cancelled, stopping telemetry client")

			return false, ctx.Err()
		default:
			if done, recovErr := c.readAndProcessPacket(telemetryReader, rawTelemetry); done {
				return recoverable, recovErr
			}

			time.Sleep(throttle)
		}
	}
}

// Run starts the telemetry client to read and process a live data stream.
//
// Deprecated: Use Stream instead.
func (c *Client) Run(ctx context.Context) (recoverable bool, err error) {
	return c.Stream(ctx)
}

// Scan returns an iterator for batch processing of a file source. Each iteration
// reads one packet, parses it, and yields the updated Transformer. The caller
// drives the loop so no packets are dropped. Only valid for file:// sources.
// The returned Transformer pointer is reused across iterations; callers must
// copy any needed data before advancing.
func (c *Client) Scan(ctx context.Context) iter.Seq2[*Transformer, error] {
	return func(yield func(*Transformer, error) bool) {
		telemetryReader, err := c.openFileReader()
		if err != nil {
			yield(nil, err)

			return
		}

		defer func() {
			closeErr := telemetryReader.Close()
			if closeErr != nil {
				c.log.Error().Err(closeErr).Msg("failed to close telemetry reader")
			}
		}()

		rawTelemetry := telemetry.NewGranTurismoTelemetry()

		for ctx.Err() == nil {
			done, readErr := c.scanNextPacket(telemetryReader, rawTelemetry)
			if done {
				if readErr != nil {
					yield(nil, readErr)
				}

				return
			}

			if len(c.DecipheredPacket) > 0 {
				if !yield(c.Telemetry, nil) {
					return
				}
			}
		}
	}
}

// IsReplaySource checks if the telemetry source is a replay file.
func (c *Client) IsReplaySource() (bool, error) {
	sourceURL, err := url.Parse(c.source)
	if err != nil {
		return false, fmt.Errorf("parse source URL: %w", err)
	}

	switch sourceURL.Scheme {
	case reader.SchemeFile:
		return true, nil
	case reader.SchemeUDP:
		return false, nil
	default:
		return false, fmt.Errorf("%w: %q", ErrInvalidURLScheme, sourceURL.Scheme)
	}
}

// StartRecording starts recording telemetry data to the specified file path.
// Supports both plain (.gtr) and compressed (.gtz) formats based on file extension.
func (c *Client) StartRecording(filePath string) error {
	c.recordingMutex.Lock()
	defer c.recordingMutex.Unlock()

	if c.isRecording {
		return ErrRecordingAlreadyInProgress
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

		return fmt.Errorf("%w: %q", ErrUnsupportedFileExtension, fileExt)
	}

	c.recordingBuffer = buffer
	c.isRecording = true

	switch {
	case c.Telemetry.IsInRaceMenu():
		c.recordingInitState = recordingStateRaceMenu
	case !c.Telemetry.IsInMainMenu():
		c.recordingInitState = recordingStateOnCircuit
	default:
		c.recordingInitState = recordingStateNone
	}

	c.log.Info().Str("file", filePath).Msg("started recording telemetry data")

	return nil
}

// StopRecording stops the current recording and closes the file.
func (c *Client) StopRecording() error {
	c.recordingMutex.Lock()
	defer c.recordingMutex.Unlock()

	if !c.isRecording {
		return ErrNoRecordingInProgress
	}

	// Flush and close the file
	err := c.recordingFile.Close()
	if err != nil {
		c.log.Error().Err(err).Msg("error closing recording file")

		return fmt.Errorf("failed to close recording file: %w", err)
	}

	c.recordingFile = nil
	c.recordingBuffer = nil
	c.isRecording = false
	c.recordingInitState = recordingStateNone

	c.log.Info().Msg("stopped recording telemetry data")

	return nil
}

// IsRecording returns true if telemetry data is currently being recorded.
func (c *Client) IsRecording() bool {
	c.recordingMutex.RLock()
	defer c.recordingMutex.RUnlock()

	return c.isRecording
}

// openFileReader parses the client source URL and opens a FileReader.
// Returns ErrNotAFileSource if the source is not a file:// URL.
func (c *Client) openFileReader() (*reader.FileReader, error) {
	sourceURL, err := url.Parse(c.source)
	if err != nil {
		return nil, fmt.Errorf("parse source URL: %w", err)
	}

	if sourceURL.Scheme != reader.SchemeFile {
		return nil, ErrNotAFileSource
	}

	r, err := reader.NewFileReader(sourceURL.Host+sourceURL.Path, c.log)
	if err != nil {
		return nil, fmt.Errorf("setup file reader: %w", err)
	}

	return r, nil
}

// scanNextPacket reads and processes one packet for the Scan iterator.
// Returns done=true when scanning is complete. A non-nil error should be yielded to the caller.
func (c *Client) scanNextPacket(r reader.Reader, raw *telemetry.GranTurismoTelemetry) (done bool, err error) {
	bufLen, buffer, readErr := r.Read()
	if readErr != nil {
		if errors.Is(readErr, io.EOF) {
			c.Finished = true

			return true, nil
		}

		return true, readErr
	}

	if len(buffer[:bufLen]) == 0 {
		return false, nil
	}

	c.DecipheredPacket = buffer[:bufLen]

	stream := kaitai.NewStream(bytes.NewReader(c.DecipheredPacket))

	c.processTelemetry(raw, stream, time.Now())

	return false, nil
}

// readAndProcessPacket reads a single packet and processes it.
// Returns true if the Run loop should return.
func (c *Client) readAndProcessPacket(telemetryReader reader.Reader, rawTelemetry *telemetry.GranTurismoTelemetry) (done bool, err error) {
	bufLen, buffer, err := telemetryReader.Read()
	if shouldContinue, finished := c.handleReadError(err); !shouldContinue {
		if finished {
			return false, nil
		}

		return true, err
	}

	if !c.handleEmptyBuffer(buffer, bufLen) {
		return false, nil
	}

	c.DecipheredPacket = buffer[:bufLen]

	decodeStart := time.Now()

	reader := bytes.NewReader(c.DecipheredPacket)
	stream := kaitai.NewStream(reader)

	c.processTelemetry(rawTelemetry, stream, decodeStart)

	return false, nil
}

// handleReadError processes errors from telemetryReader.Read.
func (c *Client) handleReadError(err error) (shouldContinue bool, finished bool) {
	if err == nil {
		return true, false
	}

	if errors.Is(err, io.EOF) {
		if !c.Finished {
			c.Finished = true
			c.log.Info().Msg("reached end of telemetry data")
		}

		return false, true
	}

	if err.Error() == "bufio.Scanner: SplitFunc returns advance count beyond input" {
		if !c.Finished {
			c.Finished = true
		}

		return false, true
	}

	c.log.Debug().Err(err).Msg("failed to receive telemetry")

	return false, true
}

// handleEmptyBuffer checks if the buffer is empty and logs if so.
func (c *Client) handleEmptyBuffer(buffer []byte, bufLen int) bool {
	if len(buffer[:bufLen]) == 0 {
		c.log.Debug().Msg("no data received")

		return false
	}

	return true
}

// processTelemetry parses and processes telemetry packets.
func (c *Client) processTelemetry(rawTelemetry *telemetry.GranTurismoTelemetry, stream *kaitai.Stream, decodeStart time.Time) {
	err := rawTelemetry.Read(stream, nil, nil)
	if err != nil {
		c.Statistics.PacketsInvalid++
		c.log.Error().Err(err).Msg("failed to parse telemetry")

		return
	}

	c.Telemetry.RawTelemetry = *rawTelemetry
	c.Statistics.decodeTimeLast = time.Since(decodeStart)
	c.collectStats()
	c.recordPacket()
}

// currentGameState returns the recording state that corresponds to the current game state.
func (c *Client) currentGameState() recordingState {
	switch {
	case c.Telemetry.IsInRaceMenu():
		return recordingStateRaceMenu
	case !c.Telemetry.IsInMainMenu():
		return recordingStateOnCircuit
	default:
		return recordingStateNone
	}
}

// recordPacket writes the current packet to the recording file if recording is active.
func (c *Client) recordPacket() {
	c.recordingMutex.RLock()
	active := c.isRecording && c.recordingBuffer != nil && len(c.DecipheredPacket) > 0
	initState := c.recordingInitState
	c.recordingMutex.RUnlock()

	if !active {
		return
	}

	if c.Telemetry.Flags().GamePaused {
		return
	}

	// Recording started in the main menu (no vehicle present) — never write packets.
	if initState == recordingStateNone {
		return
	}

	// Stop recording when the game state has changed from when recording began.
	if currentState := c.currentGameState(); currentState != initState {
		c.log.Info().
			Int("initialState", int(initState)).
			Int("currentState", int(currentState)).
			Msg("game state changed, stopping recording")

		err := c.StopRecording()
		if err != nil {
			c.log.Error().Err(err).Msg("failed to stop recording on state change")
		}

		return
	}

	c.recordingMutex.RLock()
	_, err := c.recordingBuffer.Write(c.DecipheredPacket)
	c.recordingMutex.RUnlock()

	if err != nil {
		c.log.Error().Err(err).Msg("failed to write packet to recording file")
	}
}

// gzipFileWrapper wraps a gzip writer and file to handle proper closing.
type gzipFileWrapper struct {
	file       *os.File
	gzipWriter *gzip.Writer
}

// Write writes data to the gzip writer.
func (g *gzipFileWrapper) Write(p []byte) (n int, err error) {
	return g.gzipWriter.Write(p)
}

// Close closes the gzip writer and the underlying file.
func (g *gzipFileWrapper) Close() error {
	err := g.gzipWriter.Close()
	if err != nil {
		g.file.Close()

		return err
	}

	return g.file.Close()
}

// collectStats updates the telemetry statistics based on the latest packet.
func (c *Client) collectStats() {
	if !c.Statistics.enabled {
		return
	}

	c.Statistics.PacketsTotal++

	if c.Statistics.packetIDLast == c.Telemetry.SequenceID() {
		return
	}

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
		c.Statistics.PacketsDropped += delta - 1
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
