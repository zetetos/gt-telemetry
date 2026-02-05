package reader

type Reader interface {
	Read() (int, []byte, error)
	Close() error
}

// packetHeaderBytes returns the GT packet header bytes.
func packetHeaderBytes() []byte {
	return []byte{0x30, 0x53, 0x37, 0x47}
}
