package wzexplorer

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
)

type FormatTag uint16

const (
	FormatTagPCM FormatTag = 1
	FormatTagMP3 FormatTag = 85
)

type WaveFormat struct {
	FormatTag         FormatTag
	Channels          uint16
	SampleRate        uint32
	AvgBytesPerSecond uint32
	BlockAlign        uint16
	BitsPerSample     uint16
	ExtraSize         uint16
}

type Sound interface {
	Duration() time.Duration
	Format() WaveFormat
	Stream() ([]byte, error)
}

type sound struct {
	f        *file
	stream   []byte
	duration int32
	size     int32
	offset   int64
	format   WaveFormat
}

func (s *sound) parse(f *file) (err error) {
	b := f.b
	s.f = f

	if _, err = b.Seek(1, io.SeekCurrent); err != nil {
		return
	}

	if s.size, err = b.ReadCompressInt32(); err != nil {
		return
	}
	if s.duration, err = b.ReadCompressInt32(); err != nil {
		return
	}

	// skip SoundDX8 guids
	if _, err = b.Seek(51, io.SeekCurrent); err != nil {
		return
	}

	var wavFormatLen byte
	if wavFormatLen, err = b.ReadByte(); err != nil {
		return
	}
	header := make([]byte, wavFormatLen, wavFormatLen)
	var n int
	n, err = b.Read(header)
	if err != nil {
		return
	}
	if n != int(wavFormatLen) {
		err = io.EOF
		return
	}
	// encrypted header
	if binary.LittleEndian.Uint16(header[16:]) > 0 {
		b.crypt.Transform(header)
	}
	if err = binary.Read(bytes.NewReader(header), binary.LittleEndian, &s.format); err != nil {
		return
	}

	s.offset = b.off
	if _, err = b.Seek(int64(s.size), io.SeekCurrent); err != nil {
		return
	}
	return
}

func (s *sound) Stream() (stream []byte, err error) {
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

		if s.format.FormatTag == FormatTagPCM {
			// fix wav header
			buf := bytes.NewBuffer([]byte{})
			buf.WriteString("RIFF")
			if err = binary.Write(buf, binary.LittleEndian, uint32(s.size+36)); err != nil {
				return
			}
			buf.WriteString("WAVE")
			buf.WriteString("fmt ")
			if err = binary.Write(buf, binary.LittleEndian, uint32(0x10)); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.format.FormatTag); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.format.Channels); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.format.SampleRate); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian,
				s.format.SampleRate*uint32(s.format.Channels*s.format.BitsPerSample)/8); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.format.Channels*s.format.BitsPerSample/8); err != nil {
				return
			}
			if err = binary.Write(buf, binary.LittleEndian, s.format.BitsPerSample); err != nil {
				return
			}
			buf.WriteString("data")
			if err = binary.Write(buf, binary.LittleEndian, s.size); err != nil {
				return
			}
			s.stream = append(buf.Bytes(), s.stream...)
		}
	}
	stream = s.stream
	return
}

func (s *sound) Format() WaveFormat {
	return s.format
}

func (s *sound) Duration() time.Duration {
	return time.Millisecond * time.Duration(s.duration)
}
