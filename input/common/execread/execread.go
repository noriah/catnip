// Package execread provides a shared struct that wraps around cmd.
package execread

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"math"
	"os"
	"os/exec"
	"sync"

	"github.com/noriah/catnip/input"
	"github.com/pkg/errors"
)

// Session is a session that reads floating-point audio values from a Cmd.
type Session struct {
	cmd        *exec.Cmd
	f32mode    bool
	frameSize  int
	sampleSize int

	copymut sync.Mutex
	isFull  bool // mutex guarded

	readbuf   [][]input.Sample // copied from middlebuf on demand
	middlebuf [][]input.Sample // copied from writebuf after each fill
	writebuf  [][]input.Sample // free write without mutex
}

// NewSession creates a new execread session. It never returns an error.
func NewSession(cmd *exec.Cmd, f32mode bool, cfg input.SessionConfig) (*Session, error) {
	var sampleSize = cfg.SampleSize * cfg.FrameSize

	return &Session{
		cmd:        cmd,
		frameSize:  cfg.FrameSize,
		sampleSize: sampleSize,
		f32mode:    f32mode,
		readbuf:    input.MakeBuffers(cfg),
		middlebuf:  input.MakeBuffers(cfg),
		writebuf:   input.MakeBuffers(cfg),
	}, nil
}

func (s *Session) Start() error {
	o, err := s.cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdout pipe")
	}

	if err := s.cmd.Start(); err != nil {
		o.Close()
		return errors.Wrap(err, "failed to start ffmpeg")
	}

	// Kind of optimal guess to reduce syscalls.
	const bufferMultiplier = 10

	// Calculate the optimum size of the buffer.
	var bufsz = s.sampleSize * 4 * bufferMultiplier
	if !s.f32mode {
		bufsz *= 2
	}

	// Make a read buffer the size of sampleSize float64s in bytes.
	var outbuf = bufio.NewReaderSize(o, bufsz)
	var flread = NewFloatReader(outbuf, binary.LittleEndian, s.f32mode)

	go func() {
		defer o.Close()

		var cursor = 0 // cursor

		for {
			f, err := flread.ReadFloat64()
			if err != nil {
				s.Stop()
				return
			}

			// Attempt to acquire the mutex-channel. If fail to acquire, skip
			// writing and keep reading.
			s.writebuf[cursor%s.frameSize][cursor/s.frameSize] = f

			// Override the write buffer if the read loop is too slow to catch
			// up.
			if cursor++; cursor == s.sampleSize {
				cursor = 0

				s.copymut.Lock()
				s.isFull = true
				input.CopyBuffers(s.middlebuf, s.writebuf)
				s.copymut.Unlock()
			}
		}
	}()

	return nil
}

func (s *Session) Stop() error {
	s.cmd.Process.Signal(os.Interrupt)
	return s.cmd.Wait()
}

func (s *Session) SampleBuffers() [][]input.Sample {
	return s.readbuf
}

// ReadyRead blocks until there is enough data in the sample buffer.
func (s *Session) ReadyRead() int {
	s.copymut.Lock()
	if !s.isFull {
		s.copymut.Unlock()
		return 0
	}

	return s.sampleSize
}

func (s *Session) Read(ctx context.Context) error {
	// Deep copy.
	input.CopyBuffers(s.readbuf, s.middlebuf)
	s.copymut.Unlock()
	return nil
}

// FloatReader is an io.Reader abstraction that allows using a shared bytes
// buffer.
type FloatReader struct {
	order   binary.ByteOrder
	reader  io.Reader
	buffer  []byte
	f64mode bool
}

// NewFloatReader creates a new FloatReader that optionally reads float32 or
// float64.
func NewFloatReader(r io.Reader, order binary.ByteOrder, f32mode bool) *FloatReader {
	var buf []byte
	if f32mode {
		buf = make([]byte, 4)
	} else {
		buf = make([]byte, 8)
	}

	return &FloatReader{
		order:   order,
		reader:  r,
		buffer:  buf,
		f64mode: !f32mode,
	}
}

// ReadFloat64 reads maximum 4 or 8 bytes and returns a float64.
func (f *FloatReader) ReadFloat64() (float64, error) {
	n, err := f.reader.Read(f.buffer)
	if err != nil {
		return 0, err
	}
	if n != len(f.buffer) {
		return 0, io.ErrUnexpectedEOF
	}

	if f.f64mode {
		return math.Float64frombits(f.order.Uint64(f.buffer)), nil
	}

	return float64(math.Float32frombits(f.order.Uint32(f.buffer))), nil
}
