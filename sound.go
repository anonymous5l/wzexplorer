package wzexplorer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"time"
)

type FormatTag uint16

const (
	FormatTagPCM FormatTag = 1
	FormatTagMP3 FormatTag = 85
)

type WaveFormat struct {
	FormatTag      FormatTag
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
	ExtraSize      uint16
	Extra          []byte
}

type MediaType struct {
	SoundType  byte
	MajorType  []byte
	SubType    []byte
	Reserved1  byte
	Reserved2  byte
	FormatType []byte
	Format     WaveFormat
}

type Sound interface {
	Duration() time.Duration
	Media() MediaType
	Stream(raw bool) ([]byte, error)
}

// sound Sound_DX8
type sound struct {
	f        *file
	stream   []byte
	size     int32
	duration int32
	offset   int64
	media    MediaType
}

func (s *sound) parse(f *file) (err error) {
	b := f.b
	s.f = f

	// skip reserved idk field
	if _, err = b.ReadByte(); err != nil {
		return
	}

	if s.size, err = b.ReadCompressInt32(); err != nil {
		return
	}

	if s.duration, err = b.ReadCompressInt32(); err != nil {
		return
	}

	// header format

	// uint8 - 0x02
	// GUID - majortype AM_MEDIA_TYPE
	// GUID - subtype
	// BOOL - reserved1
	// BOOL - reserved2
	// GUID - formattype

	// uint8 - 0x1E size
	// WAVEFORMATEX
	//   uint16 - wFormatTag
	//   uint16 - nChannels
	//   uint32 - nSamplesPerSec
	//   uint32 - nAvgBytesPerSec
	//   uint16 - nBlockAlign
	//   uint16 - wBitsPerSample
	//   uint16 - cbSize

	if s.media.SoundType, err = b.ReadByte(); err != nil {
		return
	}
	s.media.MajorType = make([]byte, 16, 16)
	if _, err = b.Read(s.media.MajorType); err != nil {
		return
	}
	s.media.SubType = make([]byte, 16, 16)
	if _, err = b.Read(s.media.SubType); err != nil {
		return
	}
	if s.media.Reserved1, err = b.ReadByte(); err != nil {
		return
	}
	if s.media.Reserved2, err = b.ReadByte(); err != nil {
		return
	}
	s.media.FormatType = make([]byte, 16, 16)
	if _, err = b.Read(s.media.FormatType); err != nil {
		return
	}

	if s.media.Reserved1 == 0 {
		var waveFormatSize byte
		if waveFormatSize, err = b.ReadByte(); err != nil {
			return
		}

		if waveFormatSize < 18 {
			err = errors.New("unknown sound type")
			return
		}

		var ft uint16
		if ft, err = b.ReadUInt16(); err != nil {
			return
		}
		s.media.Format.FormatTag = FormatTag(ft)
		if s.media.Format.Channels, err = b.ReadUInt16(); err != nil {
			return
		}
		if s.media.Format.SamplesPerSec, err = b.ReadUInt32(); err != nil {
			return
		}
		if s.media.Format.AvgBytesPerSec, err = b.ReadUInt32(); err != nil {
			return
		}
		if s.media.Format.BlockAlign, err = b.ReadUInt16(); err != nil {
			return
		}
		if s.media.Format.BitsPerSample, err = b.ReadUInt16(); err != nil {
			return
		}
		if s.media.Format.ExtraSize, err = b.ReadUInt16(); err != nil {
			return
		}
		if s.media.Format.ExtraSize > 0 {
			s.media.Format.Extra = make([]byte, s.media.Format.ExtraSize, s.media.Format.ExtraSize)
			if _, err = b.Read(s.media.Format.Extra); err != nil {
				return
			}
		}
	}

	s.offset = b.off

	if _, err = b.Seek(int64(s.size), io.SeekCurrent); err != nil {
		return
	}

	return
}

func (s *sound) Stream(raw bool) (stream []byte, err error) {
	if s.stream == nil {
		if _, err = s.f.b.Seek(s.offset, io.SeekStart); err != nil {
			return
		}
		s.stream = make([]byte, s.size, s.size)

		var n int
		if n, err = s.f.b.Read(s.stream); err != nil {
			return
		} else if n != int(s.size) {
			err = io.EOF
			return
		}

		if s.media.Format.FormatTag == FormatTagPCM && !raw {
			// fix wav header
			buf := bytes.NewBuffer([]byte{})
			if _, err = buf.WriteString("RIFF"); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, uint32(s.size+36)); err != nil {
				return
			}
			if _, err = buf.WriteString("WAVE"); err != nil {
				return
			}
			if _, err = buf.WriteString("fmt "); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, uint32(16)); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.media.Format.FormatTag); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.media.Format.Channels); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.media.Format.SamplesPerSec); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian,
				s.media.Format.SamplesPerSec*uint32(s.media.Format.Channels*s.media.Format.BitsPerSample)/8); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.media.Format.BlockAlign); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.media.Format.BitsPerSample); err != nil {
				return
			}
			if _, err = buf.WriteString("data"); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.size); err != nil {
				return
			}
			s.stream = append(buf.Bytes(), s.stream...)
		}
	}
	stream = s.stream
	return
}

func (s *sound) Media() MediaType {
	return s.media
}

func (s *sound) Duration() time.Duration {
	return time.Millisecond * time.Duration(s.duration)
}
