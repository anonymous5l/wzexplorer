package wzexplorer

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"strconv"
)

var (
	IvGMS   = []byte{0x4D, 0x23, 0xC7, 0x2B}
	IvEMS   = []byte{0xB9, 0x7D, 0x63, 0xE9}
	IvEmpty = []byte{0x00, 0x00, 0x00, 0x00}
)

var (
	key = []byte{
		0x13, 0x00, 0x00, 0x00,
		0x08, 0x00, 0x00, 0x00,
		0x06, 0x00, 0x00, 0x00,
		0xB4, 0x00, 0x00, 0x00,
		0x1B, 0x00, 0x00, 0x00,
		0x0F, 0x00, 0x00, 0x00,
		0x33, 0x00, 0x00, 0x00,
		0x52, 0x00, 0x00, 0x00,
	}
)

type Crypt struct {
	xor   []byte
	iv    []byte
	block cipher.Block
}

func newCrypt(iv []byte) (*Crypt, error) {
	if len(iv) != 4 {
		return nil, errors.New("invalid iv size")
	}
	c := &Crypt{}
	sum := 0
	for i := 0; i < len(iv); i++ {
		sum += int(iv[i])
	}
	if sum > 0 {
		c.iv = bytes.Repeat(iv, 4)
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		c.block = block
	}
	return c, nil
}

func (c *Crypt) ExpandXorTable(size int) {
	if size < len(c.xor) {
		return
	}

	calcSize := size - len(c.xor)

	chunk := calcSize >> 4
	if calcSize%16 > 0 {
		chunk = chunk + 1
	}

	expandSize := chunk << 4
	expand := make([]byte, expandSize, expandSize)
	if c.block != nil {
		for i := 0; i < expandSize; i += 16 {
			c.block.Encrypt(expand[i:], c.iv)
			copy(c.iv, expand[i:])
		}
	}
	c.xor = append(c.xor, expand...)
}

func (c *Crypt) Xor() []byte {
	return c.xor
}

func (c *Crypt) Transform(data []byte) {
	if c.block != nil {
		c.ExpandXorTable(len(data))
		for i := 0; i < len(data); i++ {
			data[i] ^= c.xor[i]
		}
	}
}

type CryptProvider struct {
	version int
	hash    int
	crypt   *Crypt
}

func NewCryptProvider(version int, iv []byte) (*CryptProvider, error) {
	cp := &CryptProvider{}

	crypt, err := newCrypt(iv)
	if err != nil {
		return nil, err
	}

	cp.crypt = crypt

	asciiVersion := strconv.FormatInt(int64(version), 10)
	for i := 0; i < len(asciiVersion); i++ {
		cp.hash = (cp.hash << 5) + int(asciiVersion[i]) + 1
	}

	return cp, nil
}

func (cp *CryptProvider) Verify(target uint16) error {
	v := uint16(0xff)
	for i := 0; i < 4; i++ {
		v ^= uint16((cp.hash >> (i << 3)) & 0xff)
	}

	if v != target {
		return errors.New("invalid version")
	}

	return nil
}
