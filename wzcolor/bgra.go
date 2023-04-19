package wzcolor

import "image/color"

type BGRA4444 struct {
	BG, RA byte
}

func (c BGRA4444) RGBA() (r, g, b, a uint32) {
	cb := c.BG & 0xf
	cg := (c.BG >> 4) & 0xf
	cr := c.RA & 0xf
	ca := (c.RA >> 4) & 0xf

	cb |= cb << 4
	cg |= cg << 4
	cr |= cr << 4
	ca |= ca << 4

	return color.NRGBA{
		R: cr, G: cg,
		B: cb, A: ca,
	}.RGBA()
}

func bgra4444model(c color.Color) color.Color {
	if _, ok := c.(BGRA4444); ok {
		return c
	}
	r, g, b, a := c.RGBA()

	r = (r & 0xf0) >> 4
	g = (g & 0xf0) >> 4
	b = (b & 0xf0) >> 4
	a = (a & 0xf0) >> 4

	return BGRA4444{
		BG: uint8(b) | (uint8(g) << 4),
		RA: uint8(r) | (uint8(a) << 4),
	}
}

var BGRA4444Model = color.ModelFunc(bgra4444model)
