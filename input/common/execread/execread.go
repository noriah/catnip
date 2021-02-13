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
	"sync/atomic"

	"github.com/noriah/catnip/input"
	"github.com/pkg/errors"
)

// Session is a session that reads floating-point audio values from a Cmd.
type Session struct {
	cmd        *exec.Cmd
	cfg        input.SessionConfig
	sampleSize int // multiplied

	readErr atomic.Value
	swapMut sync.Mutex
	isFull  bool // atomic; only 1 bit used!

	readbuf   [][]input.Sample // copied from copybuf on demand
	middlebuf [][]input.Sample // atomic; swapped with writebuf after each fill

	// maligned.
	f32mode bool
}

// NewSession creates a new execread session. It never returns an error.
func NewSession(cmd *exec.Cmd, f32mode bool, cfg input.SessionConfig) (*Session, error) {
	var sampleSize = cfg.SampleSize * cfg.FrameSize

	return &Session{
		cmd:        cmd,
		cfg:        cfg,
		f32mode:    f32mode,
		sampleSize: sampleSize,
		readbuf:    input.MakeBuffers(cfg),
		middlebuf:  input.MakeBuffers(cfg),
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

	go func() {
		defer o.Close()

		// Calculate the frame rate and use that as the buffer multiplier. This
		// ensures that the read buffer is big enough for a 1-second lag before
		// Reads are spammed.
		var bufferMultiplier = int(s.cfg.SampleRate) / s.cfg.SampleSize

		// Calculate the optimum size of the buffer.
		var bufsz = s.sampleSize * 4 * bufferMultiplier
		if !s.f32mode {
			bufsz *= 2
		}

		// Make a read buffer the size of sampleSize float64s in bytes.
		var outbuf = bufio.NewReaderSize(o, bufsz)
		var flread = NewFloatReader(outbuf, binary.LittleEndian, s.f32mode)

		var cursor = 0 // cursor
		var frSize = s.cfg.FrameSize
		var smSize = s.sampleSize

		var writebuf = input.MakeBuffers(s.cfg)

		for {
			f, err := flread.ReadFloat64()
			if err != nil {
				// Store the error atomically.
				s.readErr.Store(err)

				// Mark the buffer as full to trick Read() into being called.
				s.swapMut.Lock()
				s.isFull = true
				s.swapMut.Unlock()

				return
			}

			// Write to an intermediary buffer.
			writebuf[cursor%frSize][cursor/frSize] = f

			if cursor++; cursor == smSize {
				cursor = 0

				s.swapMut.Lock()
				// Swap the local buffer's array with the shared's.
				s.middlebuf, writebuf = writebuf, s.middlebuf
				// Indicate that the buffer is full.
				s.isFull = true
				s.swapMut.Unlock()

				// Discard the buffer if it's too old.
				if buffered := outbuf.Buffered(); buffered >= outbuf.Size() {
					outbuf.Discard(buffered)
				}
			}
		}
	}()

	return nil
}

// Stop terminates the underlying process and wait for it to exit.
func (s *Session) Stop() error {
	s.cmd.Process.Signal(os.Interrupt)
	return s.cmd.Wait()
}

// SampleBuffers returns the read buffer.
func (s *Session) SampleBuffers() [][]input.Sample {
	return s.readbuf
}

// ReadyRead returns 0 if the buffer is not refilled. It returns sampleSize if
// the buffer is refilled. A ReadyRead call will change the state to assume that
// a buffer has been consumed. As such, calling ReadyRead immediately afterwards
// will very likely return 0.
func (s *Session) ReadyRead() int {
	s.swapMut.Lock()
	if !s.isFull {
		s.swapMut.Unlock()
		return 0
	}

	return s.sampleSize
}

// Read copies from the middle buffer to the read buffer.
func (s *Session) Read(context.Context) error {
	// Do a periodic error check.
	if readErr, ok := s.readErr.Load().(error); ok {
		return errors.Wrap(readErr, "read loop failed")
	}

	// Deep copy.
	input.CopyBuffers(s.readbuf, s.middlebuf)
	s.swapMut.Unlock()
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
