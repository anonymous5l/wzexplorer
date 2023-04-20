# MapleStory WZ File Unpack

tools of maplestory wz files include `Canvas` `SoundDX8`

* support multiple directory struct
* support read directory struct from Base.wz or Base directory
* lazy loading save memory

## Usage

```bash
go get -u github.com/anonymous5l/wzexplorer
```

## Examples

example for cms v079 read all struct

```go
package main

import (
	"github.com/anonymous5l/wzexplorer"
	"image/png"
	"os"
)

func main() {
	cp, err := wzexplorer.NewCryptProvider(79, wzexplorer.IvEMS)
	if err != nil {
		panic(err)
	}

	/**
	wz_files_directory
        ├── Base.wz
        └── <...>.wz
	wz_files_directory
        ├── Base
        │   ├── Base.ini
        │   └── <...>.wz
        └── <...>
	*/

	archive, err := wzexplorer.NewBase(cp, "wz_files_directory")
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	obj, err := archive.GetPath("/Map/Back/poisonForest.img/back/12")
	if err != nil {
		panic(err)
	}

	o, err := os.Create("test.png")
	if err != nil {
		panic(err)
	}
	defer o.Close()

	img, err := obj.Canvas().Image()
	if err != nil {
		panic(err)
	}

	if err = png.Encode(o, img); err != nil {
		panic(err)
	}
}
```

* example for single file

```go
    cp, err := wzexplorer.NewCryptProvider(79, wzexplorer.IvEMS)
    if err != nil {
        panic(err)
    }

    f, err := wzexplorer.NewFile(cp, "filename.wz")
    if err != nil {
        panic(err)
    }
    defer f.Close()
```
