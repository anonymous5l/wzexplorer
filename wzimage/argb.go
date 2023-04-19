package wzimage

import (
	"errors"
	"image"
	"image/color"
	"wzexplorer/wzcolor"
)

type ARGB1555 struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func NewARGB1555(size image.Point, data []byte) (*ARGB1555, error) {
	dataSize := size.X * size.Y * 2
	img := &ARGB1555{
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

func (b *ARGB1555) ColorModel() color.Model {
	return wzcolor.ARGB1555Model
}

func (b *ARGB1555) Bounds() image.Rectangle {
	return b.Rect
}

func (b *ARGB1555) PixOffset(x, y int) int {
	return (y-b.Rect.Min.Y)*b.Stride + (x-b.Rect.Min.X)*2
}

func (b *ARGB1555) At(x, y int) color.Color {
	if !image.Pt(x, y).In(b.Rect) {
		return wzcolor.ARGB1555(0)
	}
	i := b.PixOffset(x, y)
	return wzcolor.ARGB1555(uint16(b.Pix[i]) | (uint16(b.Pix[i+1]) << 8))
}
