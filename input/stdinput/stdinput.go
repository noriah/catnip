package stdinput

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"os"
	"sync"
	"time"
	_ "unsafe"

	"github.com/noriah/catnip/input"
	"github.com/pkg/errors"
)

func init() {
	input.RegisterBackend("stdin", StdinBackend{})
}

type StdinBackend struct{}

func (b StdinBackend) Init() error {
	return nil
}

func (b StdinBackend) Close() error {
	return nil
}

func (b StdinBackend) Devices() ([]input.Device, error) {
	return []input.Device{StdInputDevice{}}, nil
}

func (b StdinBackend) DefaultDevice() (input.Device, error) {
	return StdInputDevice{}, nil
}

func (b StdinBackend) Start(config input.SessionConfig) (input.Session, error) {
	return NewStdinSession(config), nil
}

type StdInputDevice struct{}

func (d StdInputDevice) String() string {
	return "stdin"
}

type Session struct {
	cfg     input.SessionConfig
	samples int
	// maligned.
	f32mode bool
}

func NewStdinSession(cfg input.SessionConfig) *Session {
	return &Session{
		cfg:     cfg,
		f32mode: true,
		samples: cfg.SampleSize * cfg.FrameSize,
	}
}

func (s *Session) Start(ctx context.Context, dst [][]input.Sample, kickChan chan bool, mu *sync.Mutex) error {
	if !input.EnsureBufferLen(s.cfg, dst) {
		return errors.New("invalid dst length given")
	}

	// if initial_flags, err := Fcntl(int(uintptr(syscall.Stdin)), syscall.F_GETFL, 0); err == nil {
	// 	initial_flags |= syscall.O_NONBLOCK
	// 	if _, err := Fcntl(int(uintptr(syscall.Stdin)), syscall.F_SETFL, initial_flags); err != nil {
	// 		return errors.Wrap(err, "failed to set non block flag.")
	// 	}
	// } else {
	// 	return errors.Wrap(err, "failed to set non block flag.")
	// }

	o := os.Stdin
	defer o.Close()

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

	// timer := time.NewTimer(time.Millisecond * 100)

	for {
		// Set us a read deadline. If the deadline is reached, we'll write zeros
		// to the buffer.
		timeout := sampleDuration
		if !readExpired {
			timeout *= 2
		}
		// if err := o.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		// 	return errors.Wrap(err, "failed to set read deadline")
		// }

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
