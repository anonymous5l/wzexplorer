package wzimage

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
)

type DXT5 struct {
	*image.NRGBA
}

func NewDXT5(size image.Point, data []byte) (*DXT5, error) {
	blockSize := 16
	dataSize := ((size.X + 3) / 4) * ((size.Y + 3) / 4) * blockSize

	if len(data) != dataSize {
		return nil, errors.New("invalid data")
	}

	d := &DXT5{
		NRGBA: image.NewNRGBA(image.Rect(0, 0, size.X, size.Y)),
	}

	var (
		alphaTable      [8]byte
		alphaIndexTable [16]byte
		colorTable      [4]color.Color
		colorIndexTable [16]int
	)

	genAlphaTable := func(a0, a1 byte) {
		alphaTable[0] = a0
		alphaTable[1] = a1
		if a0 > a1 {
			for i := 2; i < 8; i++ {
				alphaTable[i] = byte(((8-i)*int(a0) + (i-1)*int(a1) + 3) / 7)
			}
		} else {
			for i := 2; i < 6; i++ {
				alphaTable[i] = byte(((6-i)*int(a0) + (i-1)*int(a1) + 2) / 5)
			}
			alphaTable[6] = 0
			alphaTable[7] = 0xFF
		}
	}

	genAlphaIndexTable := func(data []byte) {
		for i := 0; i < 16; i += 8 {
			dataIndex := (i / 8) * 3
			flags := int(data[dataIndex]) | int(data[dataIndex+1])<<8 | int(data[dataIndex+2])<<16
			for j := 0; j < 8; j++ {
				mask := 0x07 << (3 * j)
				alphaIndexTable[i+j] = byte((flags & mask) >> (3 * j))
			}
		}
	}

	for y := 0; y < size.Y; y += 4 {
		for x := 0; x < size.X; x += 4 {
			offset := x*4 + y*size.X
			genAlphaTable(data[offset], data[offset+1])
			genAlphaIndexTable(data[offset+2 : offset+8])
			c0 := binary.LittleEndian.Uint16(data[offset+8 : offset+10])
			c1 := binary.LittleEndian.Uint16(data[offset+10 : offset+12])
			genColorTable(colorTable[:], c0, c1)
			genColorIndexTable(colorIndexTable[:], data[offset+12:offset+28])
			for py := 0; py < 4; py++ {
				for px := 0; px < 4; px++ {
					r, g, b, _ := colorTable[colorIndexTable[py*4+px]].RGBA()
					a := alphaTable[alphaIndexTable[py*4+px]]
					d.Set(x+px, y+py, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: a})
				}
			}
		}
	}

	return d, nil
}
