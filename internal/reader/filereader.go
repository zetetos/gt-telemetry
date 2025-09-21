package reader

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var packetHeader = []byte{0x30, 0x53, 0x37, 0x47}

const packetInterval = (1000 / 60) * time.Millisecond

type FileReader struct {
	fileContent *bufio.Scanner
	lastRead    time.Time
	log         zerolog.Logger
	closer      func() error
}

func NewFileReader(file string, log zerolog.Logger) (*FileReader, error) {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %w", err)
	} else if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	fh, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	if len(file) < 3 {
		return nil, fmt.Errorf("filename too short")
	}

	var reader io.Reader
	fileExt := file[len(file)-3:]
	switch fileExt {
	case "gtz":
		reader, err = gzip.NewReader(fh)
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}
	case "gtr":
		reader = fh
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", fileExt)
	}

	scanner := bufio.NewScanner(reader)

	// Usually splits on a delimiter at the end of a token and drops the delimiter. However since
	// the delimiter is a magic header at the beginning of a token it is re-added to the beginning
	// of the token and the length updated accordingly.
	splitFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// Aligned packet starting with a packet header, usually the first bytes of the file on
		// the first read.
		// Advances the scanner bhy the magic header length and returns empty bytes.
		headerLen := len(packetHeader)
		if bytes.Equal(data[:headerLen], packetHeader) {
			return headerLen, []byte{}, nil
		}

		// Non-aligned packet with magic header prefix removed.
		// Returns all data up to the next magic header and also prefixes the data with the
		// magic header to create a valid telemetry packet.
		if bytes.Contains(data, packetHeader) {
			packetLen := bytes.Index(data, packetHeader)
			packet := append(packetHeader, data[:packetLen]...)

			return len(packet), packet, nil
		}

		// When emd of file reached, assume that the packet is complete
		// and return the data with the magic header prefixed.
		if atEOF {
			if len(data) == 0 {
				return 0, nil, fmt.Errorf("EOF")
			}

			packet := append(packetHeader, data...)

			return len(packet), packet, nil
		}

		return 0, nil, nil
	}

	scanner.Split(splitFunc)

	return &FileReader{
		fileContent: scanner,
		lastRead:    time.Unix(0, 0),
		log:         log,
		closer:      fh.Close,
	}, nil
}

func (r *FileReader) Read() (int, []byte, error) {
	if r.lastRead.IsZero() {
		r.log.Debug().Msg("reset last read time")
		r.lastRead = time.Now()
	}

	ok := r.fileContent.Scan()
	if !ok {
		return 0, nil, r.fileContent.Err()
	}

	packet := r.fileContent.Bytes()
	if len(packet) == 4 {
		return 0, nil, nil
	}

	elapsed := time.Since(r.lastRead)
	waitTime := packetInterval - elapsed

	if waitTime > 0 {
		timer := time.NewTimer(waitTime)
		<-timer.C
	}

	r.lastRead = time.Now()

	return len(packet), packet, nil
}

func (r *FileReader) Close() error {
	return nil
}
