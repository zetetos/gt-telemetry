package telemetrysrc

import (
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/zetetos/gt-telemetry/internal/utils"
)

type TelemetryFormat string

const (
	TelemetryFormatA     TelemetryFormat = "A"
	TelemetryFormatB     TelemetryFormat = "B"
	TelemetryFormatTilde TelemetryFormat = "~"

	HeartbeatInterval = 10 * time.Second
)

type UDPReader struct {
	conn      *net.UDPConn
	address   string
	sendPort  int
	format    TelemetryFormat
	ivSeed    uint32
	closeFunc func() error
	log       zerolog.Logger
}

func NewNetworkUDPReader(host string, sendPort int, format TelemetryFormat, log zerolog.Logger) (*UDPReader, error) {
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

	r := UDPReader{
		conn:      conn,
		address:   host,
		sendPort:  sendPort,
		format:    format,
		ivSeed:    getIVSeedForFormat(format),
		closeFunc: conn.Close,
		log:       log,
	}

	ticker := time.NewTicker(HeartbeatInterval)
	go func() {
		// Initial heartbeat
		err := r.sendHeartbeat()
		if err != nil {
			r.log.Error().Err(err).Msg("send initial heartbeat")
		}

		// Keep sending heartbeats periodically
		for range ticker.C {
			err := r.sendHeartbeat()
			if err != nil {
				r.log.Error().Err(err).Msg("send heartbeat")
			}
		}
	}()

	return &r, nil
}

func (r *UDPReader) Read() (int, []byte, error) {
	buffer := make([]byte, 4096)
	bufLen, _, err := r.conn.ReadFromUDP(buffer)
	if err != nil {
		return 0, buffer, fmt.Errorf("failed to receive telemetry: %s", err.Error())
	}

	if len(buffer[:bufLen]) == 0 {
		return 0, buffer, fmt.Errorf("no data received")
	}

	decipheredPacket, err := utils.Salsa20Decode(r.ivSeed, buffer[:bufLen])
	if err != nil {
		return 0, buffer, fmt.Errorf("failed to decipher telemetry: %s", err.Error())
	}

	return bufLen, decipheredPacket, nil
}

func (r *UDPReader) Close() error {
	return r.closeFunc()
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

func getIVSeedForFormat(format TelemetryFormat) uint32 {
	switch format {
	case TelemetryFormatA:
		return 0xDEADBEAF
	case TelemetryFormatB:
		return 0xDEADBEEF
	case TelemetryFormatTilde:
		return 0x55FABB4F
	}

	return 0x00
}
