package wzimage

import (
	"errors"
	"github.com/anonymous5l/wzexplorer/wzcolor"
	"image"
	"image/color"
)

type BGRA8888 struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func NewBGRA8888(size image.Point, data []byte) (*BGRA8888, error) {
	dataSize := size.X * size.Y * 4
	img := &BGRA8888{
		Pix:    data,
		Stride: 4 * size.X,
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

func (b *BGRA8888) ColorModel() color.Model {
	return color.NRGBAModel
}

func (b *BGRA8888) Bounds() image.Rectangle {
	return b.Rect
}

func (b *BGRA8888) PixOffset(x, y int) int {
	return (y-b.Rect.Min.Y)*b.Stride + (x-b.Rect.Min.X)*4
}

func (b *BGRA8888) At(x, y int) color.Color {
	if !image.Pt(x, y).In(b.Rect) {
		return color.NRGBA{}
	}
	i := b.PixOffset(x, y)
	return color.NRGBA{R: b.Pix[i+2], G: b.Pix[i+1], B: b.Pix[i], A: b.Pix[i+3]}
}

type BGRA4444 struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func NewBGRA4444(size image.Point, data []byte) (*BGRA4444, error) {
	dataSize := size.X * size.Y * 2
	img := &BGRA4444{
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

func (b *BGRA4444) ColorModel() color.Model {
	return wzcolor.BGRA4444Model
}

func (b *BGRA4444) Bounds() image.Rectangle {
	return b.Rect
}

func (b *BGRA4444) PixOffset(x, y int) int {
	return (y-b.Rect.Min.Y)*b.Stride + (x-b.Rect.Min.X)*2
}

func (b *BGRA4444) At(x, y int) color.Color {
	if !image.Pt(x, y).In(b.Rect) {
		return wzcolor.BGRA4444{}
	}
	i := b.PixOffset(x, y)
	bg, ra := b.Pix[i], b.Pix[i+1]
	return wzcolor.BGRA4444{BG: bg, RA: ra}
}
