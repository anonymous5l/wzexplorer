# MapleStory WZ File Unpack

tools of maplestory wz files include `Canvas` `SoundDX8`

## Usage

```bash
go get -u github.com/anonymous5l/wzexplorer
```

## Examples

for cms v079 example

```go
package main

import (
	"image/png"
	"os"
	"github.com/anonymous5l/wzexplorer"
)

func main() {
	f, err := wzexplorer.NewFile(79, wzexplorer.IvEMS, "/Users/anonymous/Documents/Projects/Golang/maplestory/resources/079/cms/wz/Map.wz")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	canvas := f.Get("/Back/poisonForest.img/back/0").Canvas()

	fmt.Println("Image properties:")
	if err = canvas.Each(func(s string, object wzexplorer.Object) bool {
		fmt.Println(s, object.String())
		return true
	}); err != nil {
		panic(err)
	}

	fmt.Println("size", canvas.Size())
	fmt.Println("format", canvas.Format().String())

	var img image.Image
	if img, err = canvas.Image(); err != nil {
		panic(err)
	}

	o, err := os.Create("test.png")
	if err != nil {
		panic(err)
	}
	defer o.Close()

	if err := png.Encode(o, img); err != nil {
		panic(err)
	}
}
```