package telemetrysrc

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

func NewFileReader(file string, log zerolog.Logger) *FileReader {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		log.Fatal().Str("file", file).Msg("file does not exist")
	} else if err != nil {
		log.Fatal().Err(err).Msg("failed to check file")
	}

	fh, err := os.Open(file)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open file")
	}

	if len(file) < 3 {
		log.Fatal().Str("file", file).Msg("filename too short")
	}

	var reader io.Reader
	fileExt := file[len(file)-3:]
	switch fileExt {
	case "gtz":
		reader, err = gzip.NewReader(fh)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create gzip reader")
		}
	case "gtr":
		reader = fh
	default:
		log.Fatal().Str("extension", fileExt).Msg("unsupported file extension")
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
	}
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
