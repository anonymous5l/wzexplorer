package wzexplorer

import (
	"errors"
	"image"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type EachObjectFunc = func(string, Object) (bool, error)

type GetObject interface {
	Get(string) (Object, error)
	GetPath(string) (Object, error)
	GetPaths([]string) (Object, error)
	Each(EachObjectFunc) (bool, error)
}

type Object interface {
	GetObject
	Type() ObjectType
	Value() interface{}
	Object() Object
	Canvas() Canvas
	Sound() Sound
	Vector() image.Point
	Int16() int16
	Int32() int32
	Int64() int64
	Float32() float32
	Float64() float64
	String() string
}

type ObjectType byte

const (
	ObjectTypeDirectory ObjectType = iota + 1
	ObjectTypeProperties
	ObjectTypeCanvas
	ObjectTypeConvex
	ObjectTypeVector
	ObjectTypeUOL
	ObjectTypeSound
	ObjectTypeVariant
	ObjectTypeVariantNil
	ObjectTypeVariantInt16
	ObjectTypeVariantInt32
	ObjectTypeVariantInt64
	ObjectTypeVariantFloat32
	ObjectTypeVariantFloat64
	ObjectTypeVariantString
)

const (
	flagDirectory = 1 << iota
	flagLoaded    = 1 << iota
	flagFile      = 1 << iota
	flagBase      = 1 << iota
)

type object struct {
	checksum           int32
	baseOffset, offset int64
	size               int32
	t                  ObjectType
	f                  *file
	o                  interface{}
	flag               byte
}

func newObject(f *file, baseOffset int64) *object {
	o := &object{f: f, offset: f.b.off, baseOffset: baseOffset}
	return o
}

func (o *object) parseVariant() (err error) {
	b := o.f.b

	var t byte
	t, err = b.ReadByte()

	if err != nil {
		return err
	}
	switch t {
	case 0x00: // nil
		o.t = ObjectTypeVariantNil
	case 0x02, 0x0b: // int16
		o.t = ObjectTypeVariantInt16
		if o.o, err = b.ReadInt16(); err != nil {
			return
		}
	case 0x03, 0x13: // int32
		o.t = ObjectTypeVariantInt32
		if o.o, err = b.ReadCompressInt32(); err != nil {
			return
		}
	case 0x14: // int64
		o.t = ObjectTypeVariantInt64
		if o.o, err = b.ReadCompressInt64(); err != nil {
			return
		}
	case 0x04: // float32
		o.t = ObjectTypeVariantFloat32
		var f byte
		if f, err = b.ReadByte(); err != nil {
			return
		}
		if f == 0x80 {
			if o.o, err = b.ReadFloat32(); err != nil {
				return
			}
		} else {
			o.o = float32(0)
		}
	case 0x05: // float64
		o.t = ObjectTypeVariantFloat64
		o.o, err = b.ReadFloat64()
	case 0x08: // string
		o.t = ObjectTypeVariantString
		o.o, err = b.ReadUOLString(o.baseOffset)
	case 0x09: // object
		var size int32
		if size, err = b.ReadInt32(); err != nil {
			return
		}
		endPosition := int64(size) + b.off
		newObj := newObject(o.f, o.baseOffset)
		newObj.size = size
		if err = newObj.parseImage(); err != nil {
			return err
		}
		// parse image BUG
		if b.off != endPosition {
			return errors.New("image read out of range")
		}
		o.t = ObjectTypeVariant
		o.o = Object(newObj)
	default:
		err = errors.New("invalid variant type")
	}
	return
}

func (o *object) parseObjectProperty() error {
	b := o.f.b
	m := make(map[string]Object)
	if _, err := b.Seek(2, io.SeekCurrent); err != nil {
		return err
	}

	props, err := b.ReadCompressInt32()
	if err != nil {
		return err
	}

	var name string
	for i := 0; i < int(props); i++ {
		name, err = b.ReadUOLString(o.baseOffset)
		if err != nil {
			return err
		}

		obj := newObject(o.f, o.baseOffset)
		obj.t = ObjectTypeVariant
		if err = obj.parse(); err != nil {
			return err
		}

		m[name] = obj
	}
	o.o = m
	return nil
}

func (o *object) parseObjectCanvas() error {
	c := &canvas{}
	if err := c.parse(o.f, o.baseOffset); err != nil {
		return err
	}
	o.o = Canvas(c)
	return nil
}

func (o *object) parseObjectConvex() error {
	b := o.f.b
	props, err := b.ReadCompressInt32()
	if err != nil {
		return err
	}
	m := make(map[string]Object)
	for i := 0; i < int(props); i++ {
		no := newObject(o.f, o.baseOffset)
		if err = no.parseImage(); err != nil {
			return err
		}
		m[strconv.FormatInt(int64(i), 10)] = no
	}
	o.o = m
	return nil
}

func (o *object) parseObjectVector() (err error) {
	b := o.f.b
	var x, y int32
	if x, err = b.ReadCompressInt32(); err != nil {
		return
	}
	if y, err = b.ReadCompressInt32(); err != nil {
		return
	}
	o.o = image.Pt(int(x), int(y))
	return
}

func (o *object) parseObjectUOL() (err error) {
	b := o.f.b
	var p string
	if _, err = b.Seek(1, io.SeekCurrent); err != nil {
		return
	}
	if p, err = b.ReadUOLString(o.baseOffset); err != nil {
		return
	}
	o.o = p
	return
}

func (o *object) parseSound() (err error) {
	s := &sound{}
	if err = s.parse(o.f); err != nil {
		return
	}
	o.o = Sound(s)
	return
}

func (o *object) parseImage() error {
	b := o.f.b
	if _, err := b.Seek(o.offset, io.SeekStart); err != nil {
		return err
	}

	tag, err := b.ReadUOLString(o.baseOffset)
	if err != nil {
		return err
	}

	o.offset = b.off
	switch tag {
	case "Property":
		o.t = ObjectTypeProperties
	case "Canvas":
		o.t = ObjectTypeCanvas
	case "Shape2D#Convex2D":
		o.t = ObjectTypeConvex
	case "Shape2D#Vector2D":
		o.t = ObjectTypeVector
	case "UOL":
		o.t = ObjectTypeUOL
	case "Sound_DX8":
		o.t = ObjectTypeSound
	default:
		return errors.New("invalid tag")
	}

	return o.parse()
}

func (o *object) parseDirectory() error {
	b := o.f.b

	m := make(map[string]Object)
	elements, err := b.ReadCompressInt32()
	if err != nil {
		return err
	}

	var elemType byte
	for i := 0; i < int(elements); i++ {
		elemType, err = b.ReadByte()
		if err != nil {
			return err
		}

		var (
			name string
			e    *object
		)

		switch elemType {
		case 1:
			// ignore 10 bytes unknown content
			if _, err = b.Seek(10, io.SeekCurrent); err != nil {
				return err
			}
			continue
		case 2:
			// UOL
			var off uint32
			off, err = b.ReadUInt32()
			if err != nil {
				return err
			}
			offset := int64(off) + o.f.startPos
			if err = b.Peek(func() error {
				_, err = b.Seek(offset, io.SeekStart)
				if err != nil {
					return err
				}
				elemType, err = b.ReadByte()
				if err != nil {
					return err
				}
				name, err = b.ReadEncryptString()
				if err != nil {
					return err
				}
				return nil
			}); err != nil {
				return err
			}
		case 3:
			// Sub Directory
			name, err = b.ReadEncryptString()
			if err != nil {
				return err
			}
		case 4:
			// Image
			name, err = b.ReadEncryptString()
			if err != nil {
				return err
			}
		default:
			return errors.New("invalid element type")
		}

		var (
			size       int32
			checksum   int32
			dataOffset uint32
		)

		size, err = b.ReadCompressInt32()
		if err != nil {
			return err
		}

		checksum, err = b.ReadCompressInt32()
		if err != nil {
			return err
		}

		dataOffset, err = o.f.readOffset()
		if err != nil {
			return err
		}

		if err = b.Peek(func() error {
			e = newObject(o.f, int64(dataOffset))
			e.offset = e.baseOffset
			e.size = size
			e.checksum = checksum

			if elemType == 3 {
				e.t = ObjectTypeDirectory
				if o.flag&flagDirectory == flagDirectory {
					folder := filepath.Dir(o.f.filename)
					var filename string
					if o.flag&flagBase == flagBase {
						filename = filepath.Join(folder, "..", name)
					} else {
						filename = filepath.Join(folder, name)
					}

					if e.o, err = NewFiles(b.provider, filename); err != nil {
						return err
					}
					e.flag = flagLoaded | flagDirectory
				} else if o.flag&flagFile == flagFile {
					folder := filepath.Dir(o.f.filename)
					filename := filepath.Join(folder, name+".wz")
					if e.o, err = NewFile(b.provider, filename); err != nil {
						return err
					}
					e.flag = flagLoaded | flagFile
				}
			} else {
				if err = e.parseImage(); err != nil {
					return err
				}
				name = strings.TrimSuffix(name, ".img")
			}

			return nil
		}); err != nil {
			return err
		}

		m[name] = e
	}
	o.o = m

	return nil
}

func (o *object) parse() (err error) {
	if o.flag&flagLoaded == flagLoaded {
		return
	}

	if o.flag&flagBase == 0 && o.flag&flagFile == flagFile {
		err = o.o.(*file).parse()
		return
	}

	b := o.f.b
	if _, err = b.Seek(o.offset, io.SeekStart); err != nil {
		return
	}
	switch o.t {
	case ObjectTypeDirectory:
		err = o.parseDirectory()
	case ObjectTypeProperties:
		err = o.parseObjectProperty()
	case ObjectTypeCanvas:
		err = o.parseObjectCanvas()
	case ObjectTypeConvex:
		err = o.parseObjectConvex()
	case ObjectTypeVector:
		err = o.parseObjectVector()
	case ObjectTypeUOL:
		err = o.parseObjectUOL()
	case ObjectTypeSound:
		err = o.parseSound()
	case ObjectTypeVariant:
		err = o.parseVariant()
	default:
		err = errors.New("invalid object type in lazy parse")
	}

	if err == nil {
		o.flag |= flagLoaded
	}
	return
}

func (o *object) Type() ObjectType {
	return o.t
}

func (o *object) Value() interface{} {
	return o.o
}

func (o *object) Object() Object {
	if obj, ok := o.Value().(Object); ok {
		return obj
	}
	return nil
}

func (o *object) Canvas() Canvas {
	if o.Value() != nil {
		if o.Type() != ObjectTypeCanvas {
			return o.Object().Canvas()
		}
		return o.Value().(Canvas)
	}
	return nil
}

func (o *object) Vector() image.Point {
	if o.Value() != nil {
		if o.Type() != ObjectTypeVector {
			return o.Object().Vector()
		}
		return o.Value().(image.Point)
	}
	return image.Pt(0, 0)
}

func (o *object) Int16() int16 {
	if o.Type() != ObjectTypeVariantInt16 {
		return -1
	}
	return o.Value().(int16)
}

func (o *object) Int32() int32 {
	if o.Type() != ObjectTypeVariantInt32 {
		return -1
	}
	return o.Value().(int32)
}

func (o *object) Int64() int64 {
	if o.Type() != ObjectTypeVariantInt64 {
		return -1
	}
	return o.Value().(int64)
}

func (o *object) Float32() float32 {
	if o.Type() != ObjectTypeVariantFloat32 {
		return -1
	}
	return o.Value().(float32)
}

func (o *object) Float64() float64 {
	if o.Type() != ObjectTypeVariantFloat64 {
		return -1
	}
	return o.Value().(float64)
}

func (o *object) String() string {
	switch o.t {
	case ObjectTypeDirectory:
		return "<Directory>"
	case ObjectTypeConvex:
		return "<Convex>"
	case ObjectTypeProperties:
		return "<Properties>"
	case ObjectTypeCanvas:
		return "<Canvas>"
	case ObjectTypeSound:
		return "<Sound>"
	case ObjectTypeVector:
		p := o.Vector()
		return "X: " +
			strconv.FormatInt(int64(p.X), 10) +
			" Y: " +
			strconv.FormatInt(int64(p.Y), 10)
	case ObjectTypeVariant:
		return o.Object().String()
	case ObjectTypeVariantNil:
		return "<nil>"
	case ObjectTypeVariantInt16:
		return strconv.FormatInt(int64(o.Int16()), 10)
	case ObjectTypeVariantInt32:
		return strconv.FormatInt(int64(o.Int32()), 10)
	case ObjectTypeVariantInt64:
		return strconv.FormatInt(o.Int64(), 10)
	case ObjectTypeVariantFloat32:
		return strconv.FormatFloat(float64(o.Float32()), 'f', -1, 32)
	case ObjectTypeVariantFloat64:
		return strconv.FormatFloat(o.Float64(), 'f', -1, 64)
	case ObjectTypeVariantString, ObjectTypeUOL:
		return o.Value().(string)
	}

	return ""
}

func (o *object) Sound() Sound {
	if o.Value() != nil {
		if o.Type() != ObjectTypeSound {
			return o.Object().Sound()
		}
		return o.Value().(Sound)
	}
	return nil
}

func (o *object) GetPath(p string) (Object, error) {
	return o.GetPaths(strings.Split(filepath.Clean(p), string(os.PathSeparator)))
}

func (o *object) GetPaths(paths []string) (Object, error) {
	if len(paths) == 0 || o == nil {
		return nil, nil
	}

	var (
		cur Object
		err error
	)
	cur = o
	for i := 0; i < len(paths); i++ {
		p := paths[i]
		if p == "" {
			continue
		}
		if cur, err = cur.Get(p); err != nil {
			return nil, err
		} else if cur == nil {
			return nil, nil
		}
	}

	if cur != nil && cur.Type() == ObjectTypeUOL {
		// try get uol object
		// FIXME if path out of current object can't get right result
		return o.Get(filepath.Join(append(paths, strings.Split(cur.String(), "/")...)...))
	}

	return cur, nil
}

func (o *object) Get(name string) (Object, error) {
	if err := o.parse(); err != nil {
		return nil, err
	}

	switch o.t {
	case ObjectTypeDirectory:
		switch m := o.o.(type) {
		case map[string]Object:
			if obj, ok := m[name]; ok {
				return obj, nil
			}
		case GetObject:
			return m.Get(name)
		default:
			return nil, errors.New("invalid object")
		}
	case ObjectTypeConvex, ObjectTypeProperties:
		m := o.o.(map[string]Object)
		if obj, ok := m[name]; ok {
			return obj, nil
		}
	case ObjectTypeCanvas:
		return o.o.(*canvas).Get(name)
	case ObjectTypeVariant:
		return o.o.(*object).Get(name)
	}
	return nil, nil
}

func (o *object) Each(cb EachObjectFunc) (bool, error) {
	if err := o.parse(); err != nil {
		return false, err
	}

	switch m := o.o.(type) {
	case map[string]Object:
		for k, v := range m {
			if next, err := cb(k, v); err != nil {
				return false, err
			} else if !next {
				return false, nil
			}
		}
	case GetObject:
		return m.Each(cb)
	default:
		return false, errors.New("invalid object")
	}
	return true, nil
}
