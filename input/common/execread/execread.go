// Package execread provides a shared struct that wraps around cmd.
package execread

import (
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
	// OnStart is called when the session starts. Nil by default.
	OnStart func(ctx context.Context, cmd *exec.Cmd) error

	argv []string
	cfg  input.SessionConfig

	samples int // multiplied

	// maligned.
	f32mode bool
}

// NewSession creates a new execread session. It never returns an error.
func NewSession(argv []string, f32mode bool, cfg input.SessionConfig) *Session {
	if len(argv) < 1 {
		panic("argv has no arg0")
	}

	return &Session{
		argv:    argv,
		cfg:     cfg,
		f32mode: f32mode,
		samples: cfg.SampleSize * cfg.FrameSize,
	}
}

func (s *Session) Start(ctx context.Context, dst [][]input.Sample, kickChan chan bool, mu *sync.Mutex) error {
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
	defer o.Close()

	bufsz := s.samples * 4
	if !s.f32mode {
		bufsz *= 2
	}

	framesz := s.cfg.FrameSize
	reader := floatReader{
		order: binary.LittleEndian,
		f64:   !s.f32mode,
	}

	// Allocate 4 times the buffer. We should ensure that we can read some of
	// the overflow.
	raw := make([]byte, bufsz)

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start ffmpeg")
	}
	defer cmd.Process.Signal(os.Interrupt)

	if s.OnStart != nil {
		if err := s.OnStart(ctx, cmd); err != nil {
			return err
		}
	}

	for {
		reader.reset(raw)

		mu.Lock()
		for n := 0; n < s.samples; n++ {
			dst[n%framesz][n/framesz] = reader.next()
		}
		mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
			// default:
		case kickChan <- true:
		}

		_, err := io.ReadFull(o, raw)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

type floatReader struct {
	order binary.ByteOrder
	buf   []byte
	n     int64
	f64   bool
}

func (f *floatReader) reset(b []byte) {
	f.n = 0
	f.buf = b
}

func (f *floatReader) next() float64 {
	n := f.n

	if f.f64 {
		f.n += 8
		return math.Float64frombits(f.order.Uint64(f.buf[n:]))
	}

	f.n += 4
	return float64(math.Float32frombits(f.order.Uint32(f.buf[n:])))
}
