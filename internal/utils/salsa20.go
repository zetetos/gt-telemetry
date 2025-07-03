package utils

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/salsa20"
)

const cipherKey string = "Simulator Interface Packet GT7 ver 0.0"

func Salsa20Decode(ivSeed uint32, dat []byte) ([]byte, error) {
	datLen := len(dat)
	if datLen < 32 {
		return nil, fmt.Errorf("salsa20 data is too short: %d < 32", datLen)
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
		return nil, fmt.Errorf("invalid magic value: %x", magic)
	}

	return ddata, nil
}
