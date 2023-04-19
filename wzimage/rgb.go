package wzimage

import (
	"errors"
	"image"
	"image/color"
	"wzexplorer/wzcolor"
)

type RGB565 struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func NewRGB565(size image.Point, data []byte) (*RGB565, error) {
	dataSize := size.X * size.Y * 2
	img := &RGB565{
		Pix:    data,
		Stride: 2 * size.X,
		Rect:   image.Rect(0, 0, size.X, size.Y),
	}
	if data == nil {
		img.Pix = make([]byte, dataSize)
	}
	if len(img.Pix) < dataSize {
		return nil, errors.New("invalid data")
	}
	return img, nil
}

func (b *RGB565) ColorModel() color.Model {
	return wzcolor.RGB565Model
}

func (b *RGB565) Bounds() image.Rectangle {
	return b.Rect
}

func (b *RGB565) PixOffset(x, y int) int {
	return (y-b.Rect.Min.Y)*b.Stride + (x-b.Rect.Min.X)*2
}

func (b *RGB565) At(x, y int) color.Color {
	if !image.Pt(x, y).In(b.Rect) {
		return wzcolor.RGB565(0)
	}
	i := b.PixOffset(x, y)
	cr, cg, cb, ca := wzcolor.RGB565(uint16(b.Pix[i]) | (uint16(b.Pix[i+1]) << 8)).RGBA()
	return color.NRGBA{R: uint8(cr), G: uint8(cg), B: uint8(cb), A: uint8(ca)}
}

type RGB565Thumb struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func NewRGB565Thumb(size image.Point, data []byte) (*RGB565Thumb, error) {
	tx, ty := size.X/16, size.Y/16
	dataSize := tx * ty * 2
	img := &RGB565Thumb{
		Pix:    data,
		Stride: 2 * tx,
		Rect:   image.Rect(0, 0, size.X, size.Y),
	}
	if len(img.Pix) < dataSize {
		return nil, errors.New("invalid data")
	}
	return img, nil
}

func (b *RGB565Thumb) ColorModel() color.Model {
	return wzcolor.RGB565Model
}

func (b *RGB565Thumb) Bounds() image.Rectangle {
	return b.Rect
}

func (b *RGB565Thumb) PixOffset(x, y int) int {
	ry := (y - b.Rect.Min.Y) / 16
	rx := (x - b.Rect.Min.X) / 16
	return ry*b.Stride + rx*2
}

func (b *RGB565Thumb) At(x, y int) color.Color {
	if !image.Pt(x, y).In(b.Rect) {
		return wzcolor.RGB565(0)
	}
	i := b.PixOffset(x, y)
	cr, cg, cb, ca := wzcolor.RGB565(uint16(b.Pix[i]) | (uint16(b.Pix[i+1]) << 8)).RGBA()
	return color.NRGBA{R: uint8(cr), G: uint8(cg), B: uint8(cb), A: uint8(ca)}
}
