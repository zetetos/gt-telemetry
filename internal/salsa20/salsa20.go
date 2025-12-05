package salsa20

import (
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/crypto/salsa20"
)

const cipherKey string = "Simulator Interface Packet GT7 ver 0.0"

var (
	ErrDataTooShort      = errors.New("salsa20 data is too short")
	ErrInvalidMagicValue = errors.New("invalid magic value")
)

func Decode(ivSeed uint32, dat []byte) ([]byte, error) {
	datLen := len(dat)
	if datLen < 32 {
		return nil, fmt.Errorf("%w: %d < 32", ErrDataTooShort, datLen)
	}

	key := [32]byte{}
	copy(key[:], cipherKey)

	nonce := make([]byte, 8)
	iv := binary.LittleEndian.Uint32(dat[0x40:0x44])
	binary.LittleEndian.PutUint32(nonce, iv^ivSeed)
	binary.LittleEndian.PutUint32(nonce[4:], iv)

	ddata := make([]byte, len(dat))
	salsa20.XORKeyStream(ddata, dat, nonce, &key)

	magic := binary.LittleEndian.Uint32(ddata[:4])
	if magic != 0x47375330 {
		return nil, fmt.Errorf("%w: %x", ErrInvalidMagicValue, magic)
	}

	return ddata, nil
}
