package reader

type Reader interface {
	Read() (int, []byte, error)
	Close() error
}
