package wzcolor

import (
	"image/color"
)

type RGB565 uint16

func (c RGB565) RGBA() (r, g, b, a uint32) {
	r = uint32(c>>11) & 0x1f
	g = uint32(c>>5) & 0x3f
	b = uint32(c & 0x1f)
	r <<= 3
	g <<= 2
	b <<= 3
	a = 0xff
	return
}

func rgb565model(c color.Color) color.Color {
	if _, ok := c.(RGB565); ok {
		return c
	}
	r, g, b, _ := c.RGBA()
	r >>= 3
	g >>= 2
	b >>= 3
	return RGB565(r<<11 | g<<5 | b)
}

var RGB565Model = color.ModelFunc(rgb565model)
