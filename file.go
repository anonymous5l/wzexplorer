package wzexplorer

import (
	"encoding/binary"
	"errors"
	"golang.org/x/exp/mmap"
	"io"
	"math"
	"strconv"
)

const PKG1 = 0x31474B50

var ErrInvalidWZFile = errors.New("invalid wz file")

type File interface {
	io.Closer
	GetObject
}

type file struct {
	*object
	b                *Blob
	version          int
	encryptedVersion uint32
	startPos         int64
}

func (f *file) readOffset() (value uint32, err error) {
	offset := ((uint32(f.b.off-f.startPos) ^ math.MaxUint32) * f.encryptedVersion) - 0x581c3f6d
	factor := byte(offset & 0x1f)
	value, err = f.b.ReadUInt32()
	if err != nil {
		return
	}
	value = (((offset << factor) | (offset >> (0x20 - factor))) ^ value) + uint32(f.startPos<<1)
	return
}

func (f *file) initVersion() error {
	encryptedVersion, err := f.b.ReadUInt16()
	if err != nil {
		return err
	}

	var sum int

	asciiVersion := strconv.FormatInt(int64(f.version), 10)
	for i := 0; i < len(asciiVersion); i++ {
		sum = (sum << 5) + int(asciiVersion[i]) + 1
	}

	v := uint16(0xff)
	for i := 0; i < 4; i++ {
		v ^= uint16((sum >> (i << 3)) & 0xff)
	}

	if v != encryptedVersion {
		return errors.New("invalid version")
	}

	f.encryptedVersion = uint32(sum)

	return nil
}

func (f *file) initWithFileName() error {
	flag, err := f.b.ReadUInt32()
	if err != nil {
		return err
	}
	if flag != PKG1 {
		return ErrInvalidWZFile
	}
	// +8 file size
	if _, err = f.b.Seek(8, io.SeekCurrent); err != nil {
		return err
	}
	startPos, err := f.b.ReadUInt32()
	if err != nil {
		return err
	}
	f.startPos = int64(startPos)
	if _, err = f.b.Seek(f.startPos, io.SeekStart); err != nil {
		return err
	}
	return nil
}

func (f *file) Close() error {
	return f.b.Close()
}

func NewFile(version int, iv []byte, filename string) (File, error) {
	f := &file{}
	f.version = version

	crypt, err := newCrypt(iv)
	if err != nil {
		return nil, err
	}

	mmapFd, err := mmap.Open(filename)
	if err != nil {
		return nil, err
	}

	f.b = newBlob(mmapFd, binary.LittleEndian, crypt, int64(mmapFd.Len()))

	if err = f.initWithFileName(); err != nil {
		return nil, err
	}
	if err = f.initVersion(); err != nil {
		return nil, err
	}

	f.object = newObject(f, f.b.off)
	if err = f.object.parseDirectory(); err != nil {
		return nil, err
	}

	return f, nil
}
