package wzcolor

import "image/color"

type ARGB1555 uint16

func (c ARGB1555) RGBA() (r, g, b, a uint32) {
	cr := (c >> 10) & 0x1f
	cg := (c >> 5) & 0x1f
	cb := c & 0x1f
	ca := ((c >> 15) & 0x1) * 0xff

	cr = (cr << 3) | (cr >> 2)
	cg = (cg << 3) | (cg >> 2)
	cb = (cb << 3) | (cb >> 2)
	return color.NRGBA{
		R: uint8(cr), G: uint8(cg),
		B: uint8(cb), A: uint8(ca),
	}.RGBA()
}

func argb1555model(c color.Color) color.Color {
	if _, ok := c.(ARGB1555); ok {
		return c
	}
	cr, cg, cb, ca := c.RGBA()
	r := (uint16(cr) & 0xf8) >> 3
	g := (uint16(cg) & 0xf8) >> 3
	b := (uint16(cb) & 0xf8) >> 3
	a := uint16(ca >> 7)

	return ARGB1555((a << 15) | (r << 10) | (g << 5) | b)
}

var ARGB1555Model = color.ModelFunc(argb1555model)
