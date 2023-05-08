package wzexplorer

import (
	"encoding/binary"
	"errors"
	"golang.org/x/exp/mmap"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

const PKG1 = 0x31474B50

var ErrInvalidWZFile = errors.New("invalid wz file")

type File interface {
	io.Closer
	GetObject
}

type file struct {
	*object
	filename string
	b        *Blob
	startPos int64
}

func (f *file) readOffset() (value uint32, err error) {
	offset := ((uint32(f.b.off-f.startPos) ^ math.MaxUint32) * uint32(f.b.provider.hash)) - 0x581c3f6d
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

	return f.b.provider.Verify(encryptedVersion)
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
	if f.flag&flagBase == flagBase || f.flag&flagDirectory == flagDirectory {
		if f.o != nil {
			m := f.o.(map[string]Object)
			for _, v := range m {
				if v.Type() == ObjectTypeDirectory {
					if val, ok := v.Value().(io.Closer); ok {
						if err := val.Close(); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return f.b.Close()
}

func NewFile(cp *CryptProvider, filename string) (File, error) {
	f := &file{}
	f.filename = filename

	mmapFd, err := mmap.Open(filename)
	if err != nil {
		return nil, err
	}

	f.b = newBlob(mmapFd, binary.LittleEndian, cp, int64(mmapFd.Len()))

	if err = f.initWithFileName(); err != nil {
		return nil, err
	}
	if err = f.initVersion(); err != nil {
		return nil, err
	}

	f.object = newObject(f, f.b.off)
	f.t = ObjectTypeDirectory

	return f, nil
}

type files struct {
	*object
}

func (f *files) Close() error {
	fs := f.o.([]File)
	for i := 0; i < len(fs); i++ {
		if err := fs[i].Close(); err != nil {
			return err
		}
	}
	return nil
}

func getIndexFile(i int, name, ext string) string {
	sb := strings.Builder{}
	sb.WriteString(name)
	if i >= 0 {
		sb.WriteByte('_')
		index := strconv.FormatInt(int64(i), 10)
		indexLen := 3
		x := indexLen - len(index)
		for j := 0; j < x; j++ {
			sb.WriteByte('0')
		}
		sb.WriteString(index)
	}
	sb.WriteByte('.')
	sb.WriteString(ext)
	return sb.String()
}

func NewFiles(cp *CryptProvider, folder string) (File, error) {
	basename := filepath.Base(folder)
	config, err := parseWzConfig(filepath.Join(folder, basename))
	if err != nil {
		return nil, err
	}
	strLastWzIndex, ok := config["LastWzIndex"]
	if !ok {
		return nil, errors.New("invalid wz config file")
	}
	lastWzIndex, err := strconv.ParseInt(strLastWzIndex, 10, 32)
	if err != nil {
		return nil, err
	}
	fileGroup := &files{}
	fileGroup.object = &object{}
	fileGroup.t = ObjectTypeDirectory
	fileGroup.flag = flagDirectory

	var groups []File
	for i := -1; i < int(lastWzIndex)+1; i++ {
		var f File
		f, err = NewFile(cp, filepath.Join(folder,
			getIndexFile(i, basename, "wz")))
		if err != nil {
			return nil, err
		}
		if cf, ok := f.(*file); ok {
			cf.flag = fileGroup.flag
		}
		groups = append(groups, f)
	}

	fileGroup.flag |= flagLoaded
	fileGroup.o = groups

	return fileGroup, nil
}
