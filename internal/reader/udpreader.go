package reader

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/zetetos/gt-telemetry/internal/salsa20"
	"github.com/zetetos/gt-telemetry/pkg/models"
)

const (
	HeartbeatInterval = 10 * time.Second
)

var (
	ErrFailedToReceiveTelemetry  = errors.New("failed to receive telemetry")
	ErrNoDataReceived            = errors.New("no data received")
	ErrFailedToDecipherTelemetry = errors.New("failed to decipher telemetry")
)

type UDPReader struct {
	conn       *net.UDPConn
	address    string
	sendPort   int
	format     models.Name
	ivSeed     uint32
	closeFunc  func() error
	stopTicker chan struct{}
	closeOnce  sync.Once
	log        zerolog.Logger
}

func NewUDPReader(host string, sendPort int, format models.Name, log zerolog.Logger) (*UDPReader, error) {
	log.Debug().Msg("creating UDP reader")

	receivePort := sendPort + 1

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", receivePort))
	if err != nil {
		return nil, fmt.Errorf("resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("setup UDP listener %d: %w", receivePort, err)
	}

	reader := UDPReader{
		conn:       conn,
		address:    host,
		sendPort:   sendPort,
		format:     format,
		ivSeed:     getIVSeedForFormat(format),
		closeFunc:  conn.Close,
		stopTicker: make(chan struct{}),
		log:        log,
	}

	ticker := time.NewTicker(HeartbeatInterval)

	go func() {
		defer ticker.Stop()

		// Initial heartbeat
		err := reader.sendHeartbeat()
		if err != nil {
			reader.log.Error().Err(err).Msg("send initial heartbeat")
		}

		// Keep sending heartbeats periodically until stopped
		for {
			select {
			case <-reader.stopTicker:
				reader.log.Debug().Msg("heartbeat goroutine stopping")

				return
			case <-ticker.C:
				err := reader.sendHeartbeat()
				if err != nil {
					reader.log.Error().Err(err).Msg("send heartbeat")
				}
			}
		}
	}()

	return &reader, nil
}

func (r *UDPReader) Read() (int, []byte, error) {
	buffer := make([]byte, 4096)

	bufLen, _, err := r.conn.ReadFromUDP(buffer)
	if err != nil {
		return 0, buffer, fmt.Errorf("%w: %s", ErrFailedToReceiveTelemetry, err.Error())
	}

	if len(buffer[:bufLen]) == 0 {
		return 0, buffer, ErrNoDataReceived
	}

	decipheredPacket, err := salsa20.Decode(r.ivSeed, buffer[:bufLen])
	if err != nil {
		return 0, buffer, fmt.Errorf("%w: %s", ErrFailedToDecipherTelemetry, err.Error())
	}

	return bufLen, decipheredPacket, nil
}

func (r *UDPReader) Close() error {
	var closeErr error

	r.closeOnce.Do(func() {
		r.log.Debug().Msg("closing UDP reader")

		// Stop the heartbeat goroutine
		close(r.stopTicker)

		// Set a short deadline to unblock any pending Read calls
		_ = r.conn.SetReadDeadline(time.Now())

		closeErr = r.closeFunc()
	})

	return closeErr
}

func (r *UDPReader) sendHeartbeat() error {
	r.log.Debug().Msgf("sending format %q heartbeat to %s:%d", r.format, r.address, r.sendPort)

	_, err := r.conn.WriteToUDP([]byte(r.format), &net.UDPAddr{
		IP:   net.ParseIP(r.address),
		Port: r.sendPort,
	})
	if err != nil {
		return fmt.Errorf("send UDP heartbeat: %w", err)
	}

	err = r.conn.SetReadDeadline(time.Now().Add(HeartbeatInterval))
	if err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}

	return nil
}

func getIVSeedForFormat(format models.Name) uint32 {
	switch format {
	case models.Standard:
		return 0xDEADBEAF
	case models.Addendum1:
		return 0xDEADBEEF
	case models.Addendum2:
		return 0x55FABB4F
	}

	return 0x00
}
