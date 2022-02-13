// Package execread provides a shared struct that wraps around cmd.
package execread

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"os"
	"os/exec"

	"github.com/noriah/catnip/input"
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
	defer o.Close()

	bufsz := s.samples * 4
	if !s.f32mode {
		bufsz *= 2
	}

	framesz := s.cfg.FrameSize
	fread := fread{
		order: binary.LittleEndian,
		f64:   !s.f32mode,
	}

	// Allocate 4 times the buffer. We should ensure that we can read some of
	// the overflow.
	raw := make([]byte, bufsz*4)

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start ffmpeg")
	}

	for {
		n, err := io.ReadAtLeast(o, raw, bufsz)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		fread.reset(raw[n-bufsz:])
		for n := 0; n < s.samples; n++ {
			dst[n%framesz][n/framesz] = fread.next()
		}

		proc.Process()
	}
}

type fread struct {
	order binary.ByteOrder
	buf   []byte
	n     int64
	f64   bool
}

func (f *fread) reset(b []byte) {
	f.n = 0
	f.buf = b
}

func (f *fread) next() float64 {
	n := f.n

	if f.f64 {
		f.n += 8
		return math.Float64frombits(f.order.Uint64(f.buf[n:]))
	}

	f.n += 4
	return float64(math.Float32frombits(f.order.Uint32(f.buf[n:])))
}
