package reader

type Reader interface {
	Read() (int, []byte, error)
	Close() error
}

// PacketHeader returns the GT packet header bytes.
func PacketHeader() []byte {
	return []byte{0x30, 0x53, 0x37, 0x47}
}
