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
	ErrEOF                      = errors.New("EOF")
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

// packetSplitFunc is the bufio.SplitFunc for packet splitting.
func packetSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	headerLen := len(PacketHeader())
	if len(data) >= headerLen && bytes.Equal(data[:headerLen], PacketHeader()) {
		return headerLen, []byte{}, nil
	}

	if idx := bytes.Index(data, PacketHeader()); idx != -1 {
		packet := append([]byte{}, PacketHeader()...)
		packet = append(packet, data[:idx]...)

		return len(packet), packet, nil
	}

	if atEOF {
		if len(data) == 0 {
			return 0, nil, ErrEOF
		}

		packet := append([]byte{}, PacketHeader()...)
		packet = append(packet, data...)

		return len(packet), packet, nil
	}

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

// Close closes the underlying file reader.
func (r *FileReader) Close() error {
	return nil
}
