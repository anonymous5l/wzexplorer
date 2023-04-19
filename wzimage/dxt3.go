package wzimage

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"wzexplorer/wzcolor"
)

func genColorTable(colorTable []color.Color, c0, c1 uint16) {
	colorTable[0] = wzcolor.RGB565(c0)
	colorTable[1] = wzcolor.RGB565(c1)

	ar, ag, ab, _ := colorTable[0].RGBA()
	br, bg, bb, _ := colorTable[1].RGBA()

	if c0 > c1 {
		colorTable[2] = color.NRGBA{
			R: byte((ar*2 + br + 1) / 3),
			G: byte((ag*2 + bg + 1) / 3),
			B: byte((ab*2 + bb + 1) / 3),
			A: 0xFF,
		}
		colorTable[3] = color.NRGBA{
			R: byte((ar + br*2 + 1) / 3),
			G: byte((ag + bg*2 + 1) / 3),
			B: byte((ab + bb*2 + 1) / 3),
			A: 0xFF,
		}
	} else {
		colorTable[2] = color.NRGBA{
			R: byte((ar + br) / 2),
			G: byte((ag + bg) / 2),
			B: byte((ab + bb) / 2),
			A: 0xFF,
		}
		colorTable[3] = color.NRGBA{A: 0xFF}
	}
}

func genColorIndexTable(colorIndexTable []int, data []byte) {
	for i := 0; i < 16; i += 4 {
		dataIndex := i / 4
		colorIndexTable[i] = int(data[dataIndex] & 0x03)
		colorIndexTable[i+1] = int(data[dataIndex]&0x0c) >> 2
		colorIndexTable[i+2] = int(data[dataIndex]&0x30) >> 4
		colorIndexTable[i+3] = int(data[dataIndex]&0xc0) >> 6
	}
}

type DXT3 struct {
	*image.NRGBA
}

func NewDXT3(size image.Point, data []byte) (*DXT3, error) {
	blockSize := 16
	dataSize := ((size.X + 3) >> 2) * ((size.Y + 3) >> 2) * blockSize

	if len(data) != dataSize {
		return nil, errors.New("invalid data")
	}

	d := &DXT3{
		NRGBA: image.NewNRGBA(image.Rect(0, 0, size.X, size.Y)),
	}

	var (
		alphaTable      [16]byte
		colorTable      [4]color.Color
		colorIndexTable [16]int
	)

	genAlphaTable := func(data []byte) {
		for i := 0; i < 16; i += 2 {
			a := data[i/2]
			a0 := a & 0x0f
			a0 |= a0 << 4
			a1 := (a & 0xf0) >> 4
			a1 |= a1 << 4
			alphaTable[i] = a0
			alphaTable[i+1] = a1
		}
	}

	for y := 0; y < size.Y; y += 4 {
		for x := 0; x < size.X; x += 4 {
			offset := x*4 + y*size.X
			genAlphaTable(data[offset : offset+8])
			c0 := binary.LittleEndian.Uint16(data[offset+8 : offset+10])
			c1 := binary.LittleEndian.Uint16(data[offset+10 : offset+12])
			genColorTable(colorTable[:], c0, c1)
			genColorIndexTable(colorIndexTable[:], data[offset+12:offset+28])
			for py := 0; py < 4; py++ {
				for px := 0; px < 4; px++ {
					r, g, b, _ := colorTable[colorIndexTable[py*4+px]].RGBA()
					a := alphaTable[py*4+px]
					d.Set(x+px, y+py, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: a})
				}
			}
		}
	}

	return d, nil
}
