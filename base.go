package wzexplorer

import (
	"os"
	"path/filepath"
)

type base struct {
	File
	cp *CryptProvider
}

func (a *base) readStructure(folder string, directory bool) (err error) {
	var b File
	if directory {
		b, err = NewFiles(a.cp, filepath.Join(folder, "Base"))
		if err != nil {
			return
		}
		if fs, ok := b.(*files); ok {
			if group, ok := fs.o.([]File); ok {
				for i := 0; i < len(group); i++ {
					gf := group[i].(*file)
					gf.flag |= flagBase
				}
			}
		}
	} else {
		b, err = NewFile(a.cp, filepath.Join(folder, "Base.wz"))
		if err != nil {
			return
		}
		if f, ok := b.(*file); ok {
			f.flag = flagFile | flagBase
		}
	}
	a.File = b
	return
}

func (a *base) Close() error {
	return a.File.Close()
}

func NewBase(cp *CryptProvider, folder string) (File, error) {
	a := &base{cp: cp}

	directory := false

	if s, err := os.Stat(filepath.Join(folder, "Base")); err == nil {
		if s.IsDir() {
			directory = true
		}
	}

	if err := a.readStructure(folder, directory); err != nil {
		return nil, err
	}
	return a, nil
}
