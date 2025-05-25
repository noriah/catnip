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
	"time"

	"github.com/noriah/catnip/input"
	"github.com/pkg/errors"
)

// Session is a session that reads floating-point audio values from a Cmd.
type Session struct {
	// OnStart is called when the session starts. Nil by default.
	OnStart func(ctx context.Context, cmd *exec.Cmd) error

	// prevents cmd.Stderr from poiting to os.Stderr. false by default.
	// this is a hack for github noriah/catnip#25
	DisconnectedStderr bool

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

	cmd := exec.CommandContext(ctx, s.argv[0], s.argv[1:]...)

	if !s.DisconnectedStderr {
		cmd.Stderr = os.Stderr
	}

	o, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdout pipe")
	}
	defer o.Close()

	// We need o as an *os.File for SetWriteDeadline.
	of, ok := o.(*os.File)
	if !ok {
		return errors.New("stdout pipe is not an *os.File (bug)")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start "+s.argv[0])
	}

	if s.OnStart != nil {
		if err := s.OnStart(ctx, cmd); err != nil {
			return err
		}
	}

	framesz := s.cfg.FrameSize
	reader := floatReader{
		order: binary.LittleEndian,
		f64:   !s.f32mode,
	}

	bufsz := s.samples
	if !s.f32mode {
		bufsz *= 2
	}

	raw := make([]byte, bufsz*4)

	// We double this as a workaround because sampleDuration is less than the
	// actual time that ReadFull blocks for some reason, probably because the
	// process decides to discard audio when it overflows.
	sampleDuration := time.Duration(
		float64(s.cfg.SampleSize) / s.cfg.SampleRate * float64(time.Second))
	// We also keep track of whether the deadline was hit once so we can half
	// the sample duration. This smooths out the jitter.
	var readExpired bool

	for {
		// Set us a read deadline. If the deadline is reached, we'll write zeros
		// to the buffer.
		timeout := sampleDuration
		if !readExpired {
			timeout *= 6
		}
		if err := of.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return errors.Wrap(err, "failed to set read deadline")
		}

		_, err := io.ReadFull(o, raw)
		if err != nil {
			switch {
			case errors.Is(err, io.EOF):
				return nil
			case errors.Is(err, os.ErrDeadlineExceeded):
				readExpired = true
			default:
				return err
			}
		} else {
			readExpired = false
		}

		if readExpired {
			mu.Lock()
			// We can write directly to dst just so we can avoid parsing zero
			// bytes to floats.
			for _, buf := range dst {
				// Go should optimize this to a memclr.
				for i := range buf {
					buf[i] = 0
				}
			}
			mu.Unlock()
		} else {
			reader.reset(raw)
			mu.Lock()
			for n := 0; n < s.samples; n++ {
				dst[n%framesz][n/framesz] = reader.next()
			}
			mu.Unlock()
		}

		// Signal that we've written to dst.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case kickChan <- true:
		}
	}
}

type floatReader struct {
	order binary.ByteOrder
	buf   []byte
	f64   bool
}

func (f *floatReader) reset(b []byte) {
	f.buf = b
}

func (f *floatReader) next() float64 {
	if f.f64 {
		b := f.buf[:8]
		f.buf = f.buf[8:]
		return math.Float64frombits(f.order.Uint64(b))
	}

	b := f.buf[:4]
	f.buf = f.buf[4:]
	return float64(math.Float32frombits(f.order.Uint32(b)))
}
