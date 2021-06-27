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
	"github.com/noriah/catnip/input/common/timer"
	"github.com/pkg/errors"
)

// Session is a session that reads floating-point audio values from a Cmd.
type Session struct {
	argv []string
	cfg  input.SessionConfig

	samples int // multiplied

	// maligned.
	f32mode bool
}

// NewSession creates a new execread session. It never returns an error.
func NewSession(argv []string, f32mode bool, cfg input.SessionConfig) (*Session, error) {
	if len(argv) < 1 {
		return nil, errors.New("argv has no arg0")
	}

	return &Session{
		argv:    argv,
		cfg:     cfg,
		f32mode: f32mode,
		samples: cfg.SampleSize * cfg.FrameSize,
	}, nil
}

func (s *Session) Start(ctx context.Context, dst [][]input.Sample, proc input.Processor) error {
	if !input.EnsureBufferLen(s.cfg, dst) {
		return errors.New("invalid dst length given")
	}

	// Take argv and free it soon after, since we won't be needing it again.
	cmd := exec.CommandContext(ctx, s.argv[0], s.argv[1:]...)
	cmd.Stderr = os.Stderr
	s.argv = nil

	o, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdout pipe")
	}

	if err := cmd.Start(); err != nil {
		o.Close()
		return errors.Wrap(err, "failed to start ffmpeg")
	}

	// Close the stdout pipe first, then send SIGINT, then wait for a graceful
	// death.
	defer cmd.Wait()
	defer cmd.Process.Signal(os.Interrupt)
	defer o.Close()

	bufsz := s.samples
	if !s.f32mode {
		bufsz *= 2
	}

	// Make a read buffer that's quadruple the size.
	outbuf := bufio.NewReaderSize(o, bufsz*4)
	flread := NewFrameReader(outbuf, binary.LittleEndian, s.f32mode)
	cursor := 0

	framesz := s.cfg.FrameSize
	flushsz := s.samples * framesz

	// Allocate a buffer specifically for the process routine to reduce lock
	// contention. The lengths of these buffers are guaranteed above.
	buf := input.MakeBuffers(s.cfg)

	return timer.Process(s.cfg, proc, func(mu *sync.Mutex) error {
		// Discard all but the last buffer so we get the latest data.
		if discard := outbuf.Buffered() - flushsz; discard > 0 {
			outbuf.Discard(discard)
		}

		for cursor = 0; cursor < s.samples; cursor++ {
			f, err := flread.ReadFloat64()
			if err != nil {
				return err
			}

			// Write to an intermediary buffer.
			buf[cursor%framesz][cursor/framesz] = f
		}

		mu.Lock()
		defer mu.Unlock()

		input.CopyBuffers(dst, buf)

		return nil
	})
}

// FrameReader is an io.Reader abstraction that allows using a shared bytes
// buffer.
type FrameReader struct {
	order   binary.ByteOrder
	reader  io.Reader
	buffer  []byte
	f64mode bool
}

// NewFrameReader creates a new FrameReader that concurrently reads a frame.
func NewFrameReader(r io.Reader, order binary.ByteOrder, f32mode bool) *FrameReader {
	var buf []byte
	if f32mode {
		buf = make([]byte, 4)
	} else {
		buf = make([]byte, 8)
	}

	return &FrameReader{
		order:   order,
		reader:  r,
		buffer:  buf,
		f64mode: !f32mode,
	}
}

// ReadFloat64 reads maximum 4 or 8 bytes and returns a float64.
func (f *FrameReader) ReadFloat64() (float64, error) {
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
