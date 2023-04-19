package wzexplorer

import (
	"errors"
	"image"
	"io"
	"path"
	"strconv"
	"strings"
)

type EachObjectFunc = func(string, Object) bool

type GetObject interface {
	GetByPaths([]string) Object
	Get(string) Object
	Each(EachObjectFunc) error
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
	ObjectVariant
	ObjectVariantNil
	ObjectVariantInt16
	ObjectVariantInt32
	ObjectVariantInt64
	ObjectVariantFloat32
	ObjectVariantFloat64
	ObjectVariantString
)

type object struct {
	checksum           int32
	baseOffset, offset int64
	size               int32
	t                  ObjectType
	f                  *file
	o                  interface{}
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
		o.t = ObjectVariantNil
	case 0x02, 0x0b: // int16
		o.t = ObjectVariantInt16
		if o.o, err = b.ReadInt16(); err != nil {
			return
		}
	case 0x03, 0x13: // int32
		o.t = ObjectVariantInt32
		if o.o, err = b.ReadCompressInt32(); err != nil {
			return
		}
	case 0x14: // int64
		o.t = ObjectVariantInt64
		if o.o, err = b.ReadCompressInt64(); err != nil {
			return
		}
	case 0x04: // float32
		o.t = ObjectVariantFloat32
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
		o.t = ObjectVariantFloat64
		o.o, err = b.ReadFloat64()
	case 0x08: // string
		o.t = ObjectVariantString
		o.o, err = b.ReadUOLString(o.baseOffset)
	case 0x09: // object
		var size int32
		if size, err = b.ReadInt32(); err != nil {
			return
		}
		newObj := newObject(o.f, o.baseOffset)
		newObj.size = size
		if err = newObj.parseImage(); err != nil {
			return err
		}
		o.t = ObjectVariant
		o.o = newObj
	default:
		err = errors.New("invalid variant type")
	}
	return
}

func (o *object) parseObjectProperty() error {
	b := o.f.b
	m := make(map[string]*object)
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
		if err = obj.parseVariant(); err != nil {
			return err
		}

		m[name] = obj
	}
	o.o = m
	o.t = ObjectTypeProperties
	return nil
}

func (o *object) parseObjectCanvas() error {
	c := &canvas{}
	if err := c.parse(o.f, o.baseOffset); err != nil {
		return err
	}
	o.o = c
	o.t = ObjectTypeCanvas
	return nil
}

func (o *object) parseObjectConvex() error {
	b := o.f.b
	props, err := b.ReadCompressInt32()
	if err != nil {
		return err
	}
	m := make(map[string]*object)
	for i := 0; i < int(props); i++ {
		no := newObject(o.f, o.baseOffset)
		if err = no.parseImage(); err != nil {
			return err
		}
		m[strconv.FormatInt(int64(i), 10)] = no
	}
	o.o = m
	o.t = ObjectTypeConvex
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
	o.t = ObjectTypeVector
	return
}

func (o *object) parseObjectUOL() (err error) {
	b := o.f.b
	var path string
	if _, err = b.Seek(1, io.SeekCurrent); err != nil {
		return
	}
	if path, err = b.ReadUOLString(o.baseOffset); err != nil {
		return
	}
	o.o = path
	o.t = ObjectTypeUOL
	return
}

func (o *object) parseSound() (err error) {
	s := &sound{}
	if err = s.parse(o.f); err != nil {
		return
	}
	o.o = s
	o.t = ObjectTypeSound
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

	switch tag {
	case "Property":
		if err = o.parseObjectProperty(); err != nil {
			return err
		}
	case "Canvas":
		if err = o.parseObjectCanvas(); err != nil {
			return err
		}
	case "Shape2D#Convex2D":
		if err = o.parseObjectConvex(); err != nil {
			return err
		}
	case "Shape2D#Vector2D":
		if err = o.parseObjectVector(); err != nil {
			return err
		}
	case "UOL":
		if err = o.parseObjectUOL(); err != nil {
			return err
		}
	case "Sound_DX8":
		if err = o.parseSound(); err != nil {
			return err
		}
	default:
		return errors.New("invalid tag")
	}

	return nil
}

func (o *object) parseDirectory() error {
	b := o.f.b
	if _, err := b.Seek(o.offset, io.SeekStart); err != nil {
		return err
	}

	m := make(map[string]*object)

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
				if err = e.parseDirectory(); err != nil {
					return err
				}
			} else {
				if err = e.parseImage(); err != nil {
					return err
				}
			}

			return nil
		}); err != nil {
			return err
		}

		m[name] = e
	}

	o.o = m
	o.t = ObjectTypeDirectory

	return nil
}

func (o *object) getMap() map[string]*object {
	if o == nil {
		return nil
	}
	switch o.t {
	case ObjectTypeDirectory, ObjectTypeConvex, ObjectTypeProperties:
		return o.o.(map[string]*object)
	case ObjectTypeCanvas:
		return o.o.(*canvas).getMap()
	case ObjectVariant:
		return o.o.(*object).getMap()
	}
	return nil
}

func (o *object) get(name string) *object {
	if m := o.getMap(); m != nil {
		return m[name]
	}
	return nil
}

func (o *object) Type() ObjectType {
	return o.t
}

func (o *object) Value() interface{} {
	return o.o
}

func (o *object) Object() Object {
	return o.Value().(*object)
}

func (o *object) Canvas() Canvas {
	return o.Object().Value().(*canvas)
}

func (o *object) Vector() image.Point {
	return o.Value().(image.Point)
}

func (o *object) Int16() int16 {
	return o.Value().(int16)
}

func (o *object) Int32() int32 {
	return o.Value().(int32)
}

func (o *object) Int64() int64 {
	return o.Value().(int64)
}

func (o *object) Float32() float32 {
	return o.Value().(float32)
}

func (o *object) Float64() float64 {
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
	case ObjectVariant:
		return o.Object().String()
	case ObjectVariantNil:
		return "<nil>"
	case ObjectVariantInt16:
		return strconv.FormatInt(int64(o.Int16()), 10)
	case ObjectVariantInt32:
		return strconv.FormatInt(int64(o.Int32()), 10)
	case ObjectVariantInt64:
		return strconv.FormatInt(o.Int64(), 10)
	case ObjectVariantFloat32:
		return strconv.FormatFloat(float64(o.Float32()), 'f', -1, 32)
	case ObjectVariantFloat64:
		return strconv.FormatFloat(o.Float64(), 'f', -1, 64)
	case ObjectVariantString, ObjectTypeUOL:
		return o.Value().(string)
	}

	return ""
}

func (o *object) Sound() Sound {
	return o.Object().Value().(*sound)
}

func (o *object) GetByPaths(paths []string) Object {
	if len(paths) == 0 || o == nil {
		return nil
	}

	var cur *object
	cur = o
	for i := 0; i < len(paths); i++ {
		p := paths[i]
		if p == "" {
			continue
		}
		if cur = cur.get(p); cur == nil {
			return nil
		}
	}

	if cur != nil && cur.t == ObjectTypeUOL {
		// get uol object
		p := path.Join(append(paths,
			strings.Split(cur.o.(string), "/")...)...)
		return o.Get(p)
	}

	return cur
}

func (o *object) Get(p string) Object {
	return o.GetByPaths(
		strings.Split(
			strings.TrimPrefix(p, "/"), "/"))
}

func (o *object) Each(cb func(name string, obj Object) bool) error {
	m := o.getMap()
	if m == nil {
		return errors.New("invalid map instance")
	}
	for k, v := range m {
		if next := cb(k, v); !next {
			return nil
		}
	}
	return nil
}
