package reader

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

const packetInterval = (1000 / 60) * time.Millisecond

var (
	ErrFilenameTooShort         = errors.New("filename too short")
	ErrUnsupportedFileExtension = errors.New("unsupported file extension")
)

// FileReader reads GT7 replay files packet by packet.
type FileReader struct {
	fileContent *bufio.Scanner
	lastRead    time.Time
	log         zerolog.Logger
	closer      func() error
}

// NewFileReader creates a new FileReader for the specified GT7 replay file.
func NewFileReader(file string, log zerolog.Logger) (*FileReader, error) {
	err := validateFile(file)
	if err != nil {
		return nil, err
	}

	fileHandle, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	reader, err := getFileReader(file, fileHandle)
	if err != nil {
		fileHandle.Close()

		return nil, err
	}

	scanner := bufio.NewScanner(reader)
	scanner.Split(packetSplitFunc)

	return &FileReader{
		fileContent: scanner,
		lastRead:    time.Unix(0, 0),
		log:         log,
		closer:      fileHandle.Close,
	}, nil
}

// validateFile checks file existence and length.
func validateFile(file string) error {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %w", err)
	} else if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	if len(file) < 3 {
		return ErrFilenameTooShort
	}

	return nil
}

// getFileReader returns the appropriate reader based on file extension.
func getFileReader(file string, fileHandle *os.File) (io.Reader, error) {
	fileExt := file[len(file)-3:]

	switch fileExt {
	case "gtz":
		reader, err := gzip.NewReader(fileHandle)
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}

		return reader, nil
	case "gtr":
		return fileHandle, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFileExtension, fileExt)
	}
}

// packetSplitFunc is the bufio.SplitFunc for splitting packets on magic header sequence boundaries.
// Packets are delimited by a 4-byte header (0x30 0x53 0x37 0x47).
// This function extracts complete packets by finding the boundary between consecutive headers.
func packetSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	header := packetHeaderBytes()
	headerLen := len(header)

	// Data available is less than header length
	if len(data) < headerLen {
		if atEOF {
			return len(data), nil, nil // Discard incomplete data at EOF
		}

		return 0, nil, nil // Request more data
	}

	// Check if data starts with a valid header
	startsWithHeader := bytes.Equal(data[:headerLen], header)

	if !startsWithHeader {
		// Scan forward to find a header
		if idx := bytes.Index(data, header); idx != -1 {
			return idx, nil, nil // Skip junk bytes
		}

		// No header found in current data
		if atEOF {
			return len(data), nil, nil // Discard all junk up to EOF
		}

		return 0, nil, nil // Request more data
	}

	// Data starts with a header - find where this packet ends (next header position)
	nextHeaderIdx := bytes.Index(data[headerLen:], header)

	// Next header located - return packet data between current and next header
	if nextHeaderIdx != -1 {
		packetLen := headerLen + nextHeaderIdx

		return packetLen, data[:packetLen], nil
	}

	// Return remaining data at end of file
	if atEOF {
		return len(data), data, nil
	}

	// Request more data to find the next header
	return 0, nil, nil
}

// Read reads the next packet from the file, simulating real-time intervals.
func (r *FileReader) Read() (int, []byte, error) {
	if r.lastRead.IsZero() {
		r.log.Debug().Msg("reset last read time")
		r.lastRead = time.Now()
	}

	ok := r.fileContent.Scan()
	if !ok {
		err := r.fileContent.Err()
		if err != nil {
			return 0, nil, err
		}
		// Scanner finished with no error - EOF reached
		return 0, nil, io.EOF
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

// Close closes the underlying file reader.
func (r *FileReader) Close() error {
	return nil
}
