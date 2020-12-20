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

	"github.com/noriah/catnip/input"
	"github.com/pkg/errors"
)

// Session is a session that reads floating-point audio values from a Cmd.
type Session struct {
	argv []string
	cfg  input.SessionConfig

	sampleSize int // multiplied

	// maligned.
	f32mode bool
}

// NewSession creates a new execread session. It never returns an error.
func NewSession(argv []string, f32mode bool, cfg input.SessionConfig) (*Session, error) {
	if len(argv) < 1 {
		return nil, errors.New("argv has no arg0")
	}

	return &Session{
		argv:       argv,
		cfg:        cfg,
		f32mode:    f32mode,
		sampleSize: cfg.SampleSize * cfg.FrameSize,
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

	for {
		f, err := flread.ReadFloat64()
		if err != nil {
			// EOF is graceful.
			if errors.Is(err, io.EOF) {
				return nil
			}
			return errors.Wrap(err, "failed to read float64")
		}

		// Write to an intermediary buffer.
		dst[cursor%frSize][cursor/frSize] = f

		if cursor++; cursor == smSize {
			cursor = 0

			proc.Process()

			// Discard the buffer and read a new one.
			outbuf.Discard(outbuf.Buffered())
		}
	}
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
