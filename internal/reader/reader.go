package reader

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/zetetos/gt-telemetry/v2/pkg/models"
)

const (
	SchemeUDP  = "udp"
	SchemeFile = "file"
)

var ErrInvalidURLScheme = errors.New("invalid URL scheme")

// Reader is the interface for reading telemetry packets.
type Reader interface {
	Read() (int, []byte, error)
	Close() error
}

// Config holds a constructed Reader along with source metadata.
type Config struct {
	Reader      Reader
	Recoverable bool
	Throttle    time.Duration
}

// New constructs a Reader and associated source metadata from a parsed source URL.
func New(sourceURL *url.URL, format models.Name, log zerolog.Logger) (Config, error) {
	switch sourceURL.Scheme {
	case SchemeUDP:
		host, portStr, _ := net.SplitHostPort(sourceURL.Host)

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return Config{Recoverable: true}, fmt.Errorf("parse URL port: %w", err)
		}

		r, err := NewUDPReader(host, port, format, log)
		if err != nil {
			return Config{Recoverable: true}, fmt.Errorf("setup UDP reader: %w", err)
		}

		return Config{Reader: r, Recoverable: true, Throttle: 0}, nil
	case SchemeFile:
		r, err := NewFileReader(sourceURL.Host+sourceURL.Path, log)
		if err != nil {
			return Config{}, fmt.Errorf("setup file reader: %w", err)
		}

		return Config{Reader: r, Recoverable: false, Throttle: PacketInterval}, nil
	default:
		return Config{}, fmt.Errorf("%w: %q", ErrInvalidURLScheme, sourceURL.Scheme)
	}
}

// packetHeaderBytes returns the GT packet header bytes.
func packetHeaderBytes() []byte {
	return []byte{0x30, 0x53, 0x37, 0x47}
}
