package wzexplorer

import (
	"bytes"
	"compress/zlib"
	"errors"
	"github.com/anonymous5l/wzexplorer/wzimage"
	"image"
	"io"
)

type CanvasFormat int

const (
	CanvasFormatBGRA4444    CanvasFormat = 1
	CanvasFormatBGRA8888    CanvasFormat = 2
	CanvasFormatGray        CanvasFormat = 3    // 0x2 + 1
	CanvasFormatARGB1555    CanvasFormat = 257  // 0x100 + 1
	CanvasFormatRGB565      CanvasFormat = 513  // 0x200
	CanvasFormatRGB565Thumb CanvasFormat = 517  // 0x200 + 5
	CanvasFormatDXT3        CanvasFormat = 1026 // 0x400 + 2
	CanvasFormatDXT5        CanvasFormat = 2050 // 0x800 + 2
)

func (f CanvasFormat) String() string {
	switch f {
	case CanvasFormatBGRA4444:
		return "BGRA4444"
	case CanvasFormatBGRA8888:
		return "BGRA8888"
	case CanvasFormatGray:
		return "Gray"
	case CanvasFormatARGB1555:
		return "ARGB1555"
	case CanvasFormatRGB565:
		return "RGB565"
	case CanvasFormatRGB565Thumb:
		return "RGB565Thumb"
	case CanvasFormatDXT3:
		return "DXT3"
	case CanvasFormatDXT5:
		return "DXT5"
	}
	return "Unknown"
}

type Canvas interface {
	GetObject
	Size() image.Point
	Image() (image.Image, error)
	Format() CanvasFormat
}

type canvas struct {
	*object
	img           image.Image
	format        CanvasFormat
	magLevel      byte
	width, height int32
}

func (c *canvas) Size() image.Point {
	return image.Pt(int(c.width), int(c.height))
}

func (c *canvas) Format() CanvasFormat {
	return c.format
}

func (c *canvas) build(deflated []byte) (err error) {
	switch c.format {
	case CanvasFormatBGRA4444:
		c.img, err = wzimage.NewBGRA4444(c.Size(), deflated)
	case CanvasFormatBGRA8888:
		c.img, err = wzimage.NewBGRA8888(c.Size(), deflated)
	case CanvasFormatARGB1555:
		c.img, err = wzimage.NewARGB1555(c.Size(), deflated)
	case CanvasFormatRGB565:
		c.img, err = wzimage.NewRGB565(c.Size(), deflated)
	case CanvasFormatRGB565Thumb:
		c.img, err = wzimage.NewRGB565Thumb(c.Size(), deflated)
	case CanvasFormatDXT3, CanvasFormatGray:
		c.img, err = wzimage.NewDXT3(c.Size(), deflated)
	case CanvasFormatDXT5:
		c.img, err = wzimage.NewDXT5(c.Size(), deflated)
	}
	return
}

func (c *canvas) Image() (bitmap image.Image, err error) {
	if c.img != nil {
		return c.img, nil
	}

	b := c.f.b
	offset := c.offset + 1
	size := c.size - 1
	if _, err = b.Seek(offset, io.SeekStart); err != nil {
		return
	}

	var n int

	data := make([]byte, size, size)
	if n, err = b.Read(data); err != nil {
		return
	} else if n != int(size) {
		err = io.EOF
		return
	}

	var deflated []byte

	header := b.o.Uint16(data)

	if header != 0x9c78 && header != 0xda78 && header != 0x0178 && header != 0x5e78 {
		var shrinkData []byte
		for len(data) > 0 {
			blockSize := int(b.o.Uint32(data))
			transform := data[4 : 4+blockSize]
			b.provider.crypt.Transform(transform)
			shrinkData = append(shrinkData, transform...)
			data = data[4+blockSize:]
		}
		deflated, err = c.deflate(shrinkData)
		if err != nil {
			return
		}
	} else {
		deflated, err = c.deflate(data)
		if err != nil {
			return
		}
	}

	if err = c.build(deflated); err != nil {
		return
	}

	bitmap = c.img
	return
}

func (c *canvas) deflate(data []byte) (deflated []byte, err error) {
	var stream io.ReadCloser
	if stream, err = zlib.NewReader(bytes.NewReader(data)); err != nil {
		return
	}
	defer stream.Close()

	buffer := bytes.NewBuffer([]byte{})
	swap := make([]byte, 1024)
	var n int
	for {
		if n, err = stream.Read(swap); err != nil {
			if err != io.ErrUnexpectedEOF {
				return
			}
			err = nil
			break
		}
		buffer.Write(swap[:n])
	}
	deflated = buffer.Bytes()
	return
}

func (c *canvas) parse(f *file, offset int64) error {
	b := f.b
	if _, err := b.Seek(1, io.SeekCurrent); err != nil {
		return err
	}

	hasProperty, err := b.ReadByte()
	if err != nil {
		return err
	}

	c.object = newObject(f, offset)
	c.object.t = ObjectTypeVariantNil
	if hasProperty > 0 {
		c.object.t = ObjectTypeProperties
		if err = c.object.parse(); err != nil {
			return err
		}
	}

	if c.width, err = b.ReadCompressInt32(); err != nil {
		return err
	}
	if c.height, err = b.ReadCompressInt32(); err != nil {
		return err
	}
	var (
		format  int32
		format2 byte
	)

	if format, err = b.ReadCompressInt32(); err != nil {
		return err
	}
	if format2, err = b.ReadByte(); err != nil {
		return err
	}
	format += int32(format2)
	c.format = CanvasFormat(format)
	var test int32
	if test, err = b.ReadInt32(); err != nil {
		return err
	}
	if test != 0 {
		return errors.New("invalid image struct")
	}
	if c.size, err = b.ReadInt32(); err != nil {
		return err
	}
	if c.offset, err = b.Seek(0, io.SeekCurrent); err != nil {
		return err
	}
	if _, err = b.Seek(int64(c.size), io.SeekCurrent); err != nil {
		return err
	}
	return nil
}
