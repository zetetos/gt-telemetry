package telemetrysrc

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/zetetos/gt-telemetry/internal/utils"
)

type UDPReader struct {
	conn      *net.UDPConn
	address   string
	sendPort  int
	closeFunc func() error
	log       zerolog.Logger
}

func NewNetworkUDPReader(host string, sendPort int, log zerolog.Logger) *UDPReader {
	log.Debug().Msg("creating UDP reader")
	receivePort := sendPort + 1
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", receivePort))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to resolve UDP address")
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to setup UDP listener")
	}

	r := UDPReader{
		conn:      conn,
		address:   host,
		sendPort:  sendPort,
		closeFunc: conn.Close,
		log:       log,
	}

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		r.sendHeartbeat()

		for range ticker.C {
			r.sendHeartbeat()
		}
	}()

	return &r
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

	decipheredPacket, err := utils.Salsa20Decode(buffer[:bufLen])
	if err != nil {
		return 0, buffer, fmt.Errorf("failed to decipher telemetry: %s", err.Error())
	}

	return bufLen, decipheredPacket, nil
}

func (r *UDPReader) Close() error {
	return r.closeFunc()
}

func (r *UDPReader) sendHeartbeat() {
	r.log.Debug().Msgf("sending heartbeat to %s:%d", r.address, r.sendPort)

	_, err := r.conn.WriteToUDP([]byte("A"), &net.UDPAddr{
		IP:   net.ParseIP(r.address),
		Port: r.sendPort,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = r.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		log.Fatal(err)
	}
}
