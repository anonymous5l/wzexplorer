package wzexplorer

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"unicode/utf16"
)

type BlobReader interface {
	io.ReaderAt
	io.Closer
}

type Blob struct {
	off, len int64
	o        binary.ByteOrder
	fd       BlobReader
	crypt    *Crypt
	swap     []byte // attention: concurrent problem
}

func newBlob(reader BlobReader, o binary.ByteOrder, crypt *Crypt, size int64) *Blob {
	b := &Blob{}
	b.off = 0
	b.len = size
	b.fd = reader
	b.o = o
	b.crypt = crypt
	b.swap = make([]byte, 8, 8)
	return b
}

func (b *Blob) Read(data []byte) (n int, err error) {
	if b.len == -1 {
		return -1, io.ErrClosedPipe
	}
	n, err = b.fd.ReadAt(data, b.off)
	if n > 0 {
		b.off += int64(n)
	}
	return
}

func (b *Blob) readSwap(size int) (buf []byte, err error) {
	if size > 8 || size <= 0 {
		err = errors.New("invalid size out of range")
		return
	}

	var n int
	n, err = b.Read(b.swap[:size])
	if err != nil {
		return
	}
	if n > 0 {
		buf = b.swap[:size]
	} else {
		err = io.EOF
	}
	return
}

func (b *Blob) ReadByte() (value byte, err error) {
	var buf []byte
	buf, err = b.readSwap(1)
	if err != nil {
		return
	} else {
		value = buf[0]
	}
	return
}

func (b *Blob) ReadUInt8() (value uint8, err error) {
	value, err = b.ReadByte()
	return
}

func (b *Blob) ReadInt8() (value int8, err error) {
	var value1 byte
	value1, err = b.ReadByte()
	if err != nil {
		return
	} else {
		value = int8(value1)
	}
	return
}

func (b *Blob) ReadUInt16() (value uint16, err error) {
	var buf []byte
	buf, err = b.readSwap(2)
	if err != nil {
		return
	} else {
		value = b.o.Uint16(buf)
	}
	return
}

func (b *Blob) ReadInt16() (value int16, err error) {
	var u uint16
	u, err = b.ReadUInt16()
	if err != nil {
		return
	} else {
		value = int16(u)
	}
	return
}

func (b *Blob) ReadUInt32() (value uint32, err error) {
	var buf []byte
	buf, err = b.readSwap(4)
	if err != nil {
		return
	} else {
		value = b.o.Uint32(buf)
	}
	return
}

func (b *Blob) ReadInt32() (value int32, err error) {
	var u uint32
	u, err = b.ReadUInt32()
	if err != nil {
		return
	} else {
		value = int32(u)
	}
	return
}

func (b *Blob) ReadUInt64() (value uint64, err error) {
	var buf []byte
	buf, err = b.readSwap(8)
	if err != nil {
		return
	} else {
		value = b.o.Uint64(buf)
	}
	return
}

func (b *Blob) ReadInt64() (value int64, err error) {
	var u uint64
	u, err = b.ReadUInt64()
	if err != nil {
		return
	} else {
		value = int64(u)
	}
	return
}

func (b *Blob) ReadFloat32() (value float32, err error) {
	var u32 uint32
	u32, err = b.ReadUInt32()
	if err != nil {
		return
	}
	value = math.Float32frombits(u32)
	return
}

func (b *Blob) ReadFloat64() (value float64, err error) {
	var u64 uint64
	u64, err = b.ReadUInt64()
	if err != nil {
		return
	}
	value = math.Float64frombits(u64)
	return
}

func (b *Blob) ReadCompressInt32() (value int32, err error) {
	var flag int8
	flag, err = b.ReadInt8()
	if err != nil {
		return
	}
	if flag == -128 {
		value, err = b.ReadInt32()
	} else {
		value = int32(flag)
	}
	return
}

func (b *Blob) ReadCompressInt64() (value int64, err error) {
	var flag int8
	flag, err = b.ReadInt8()
	if err != nil {
		return
	}
	if flag == -128 {
		value, err = b.ReadInt64()
	} else {
		value = int64(flag)
	}
	return
}

func (b *Blob) ReadUTF8String(size int, mask bool) (value string, err error) {
	buf := make([]byte, size, size)

	var n int
	n, err = b.Read(buf)
	if err != nil {
		return
	}

	if n != size {
		err = io.EOF
		return
	}

	if mask {
		start := byte(0xaa)
		for i := 0; i < size; i++ {
			buf[i] ^= start
			start++
		}
	}

	b.crypt.Transform(buf)

	value = string(buf)

	return
}

func (b *Blob) ReadUTF16String(size int, mask bool) (value string, err error) {
	size = size << 1
	buf := make([]byte, size, size)

	var n int
	n, err = b.Read(buf)
	if err != nil {
		return
	}

	if n != size {
		err = io.EOF
		return
	}

	var unicode []uint16

	if mask {
		b.crypt.ExpandXorTable(size)
		xor := b.crypt.Xor()
		start := uint16(0xaaaa)
		for i := 0; i < size; i += 2 {
			unicode = append(unicode, b.o.Uint16(buf[i:])^start^b.o.Uint16(xor[i:]))
			start++
		}
	} else {
		b.crypt.Transform(buf)
		for i := 0; i < size; i += 2 {
			unicode = append(unicode, b.o.Uint16(buf[i:]))
		}
	}

	value = string(utf16.Decode(unicode))

	return
}

func (b *Blob) ReadEncryptString() (value string, err error) {
	var size int8
	size, err = b.ReadInt8()
	if err != nil {
		return
	}

	strLength := int32(size)

	unicode := size > 0

	if strLength == -128 || strLength == 127 {
		strLength, err = b.ReadInt32()
		if err != nil {
			return
		}
	}

	if strLength == 0 {
		return "", nil
	}

	if !unicode {
		strLength = int32(math.Abs(float64(strLength)))
	}

	if unicode {
		value, err = b.ReadUTF16String(int(strLength), true)
	} else {
		value, err = b.ReadUTF8String(int(strLength), true)
	}
	return
}

func (b *Blob) ReadUOLString(offset int64) (tag string, err error) {
	var k byte
	if k, err = b.ReadByte(); err != nil {
		return
	}

	// 00, 01 for UOL string
	switch k {
	case 0x00, 0x73:
		tag, err = b.ReadEncryptString()
	case 0x01, 0x1B:
		// UOL
		var off int32
		if off, err = b.ReadInt32(); err != nil {
			return
		}
		if err = b.Peek(func() error {
			if _, err = b.Seek(int64(off)+offset, io.SeekStart); err != nil {
				return err
			}
			tag, err = b.ReadEncryptString()
			return err
		}); err != nil {
			return
		}
	default:
		err = errors.New("invalid uol key")
	}
	return
}

func (b *Blob) Peek(cb func() error) error {
	cur, err := b.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	defer b.Seek(cur, io.SeekStart)
	if err = cb(); err != nil {
		return err
	}
	return nil
}

func (b *Blob) Seek(offset int64, whence int) (ret int64, err error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = b.off + offset
	case io.SeekEnd:
		abs = b.len + offset
	default:
		return 0, errors.New("invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("negative position")
	}
	b.off = abs
	return abs, nil
}

func (b *Blob) Len() int64 {
	return b.len
}

func (b *Blob) Close() error {
	b.len = -1
	return b.fd.Close()
}
