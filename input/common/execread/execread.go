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

	"github.com/noriah/tavis/input"
	"github.com/pkg/errors"
)

// Session is a session that reads floating-point audio values from a Cmd.
type Session struct {
	cmd        *exec.Cmd
	f32mode    bool
	frameSize  int
	sampleSize int

	full    chan struct{} // indicate buffer is full
	copymut chan struct{} // copy mutex; uses TryLock

	writebuf []input.Sample   // alternating channels
	readbuf  [][]input.Sample // separated channels
}

// NewSession creates a new execread session. It never returns an error.
func NewSession(cmd *exec.Cmd, f32mode bool, cfg input.SessionConfig) (*Session, error) {
	var sampleSize = cfg.SampleSize * cfg.FrameSize

	return &Session{
		cmd:        cmd,
		f32mode:    f32mode,
		frameSize:  cfg.FrameSize,
		sampleSize: sampleSize,
		full:       make(chan struct{}, 1), // buffered to accept overflows
		copymut:    make(chan struct{}, 1),
		readbuf:    input.MakeBuffers(cfg),
		writebuf:   make([]input.Sample, sampleSize),
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

	// Calculate the optimum size of the buffer.
	var bufsz = s.sampleSize
	if s.f32mode {
		bufsz *= 4
	} else {
		bufsz *= 8
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
			select {
			case s.copymut <- struct{}{}:
				s.writebuf[cursor] = f
				<-s.copymut
			default:
			}

			// Override the write buffer if the read loop is too slow to catch
			// up.
			if cursor++; cursor >= s.sampleSize {
				cursor = 0

				// Attempt to indicate the buffer is ready to read. Do nothing
				// if the channel is already full.
				select {
				case s.full <- struct{}{}:
				default:
				}
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
	<-s.full
	return s.sampleSize
}

func (s *Session) Read(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.copymut <- struct{}{}:
		// mutex locked
	}

	for i := 0; i < s.sampleSize; i += s.frameSize {
		f := i / s.frameSize
		for j := 0; j < s.frameSize; j++ {
			s.readbuf[j][f] = s.writebuf[i+j]
		}
	}

	<-s.copymut
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
	_, err := io.ReadFull(f.reader, f.buffer)
	if err != nil {
		return 0, err
	}

	if f.f64mode {
		return math.Float64frombits(f.order.Uint64(f.buffer)), nil
	}

	return float64(math.Float32frombits(f.order.Uint32(f.buffer))), nil
}
