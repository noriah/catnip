package input

import (
	"context"
	"sync"
)

type Device interface {
	// String returns the device name.
	String() string
}

type SessionConfig struct {
	Device     Device
	FrameSize  int     // number of channels per frame
	SampleSize int     // number of frames per buffer write
	SampleRate float64 // sample rate
}

// Session is the interface for an input session. Its task is to call the
// processor everytime the buffer is full using the parameters given in
// SessionConfig.
type Session interface {
	// Start blocks until either the context is canceled or an error is
	// encountered.
	Start(context.Context, [][]Sample, chan bool, *sync.Mutex) error
}

// Processor is called by Session everytime the buffer is full. Session may call
// this on another goroutine; the implementation must handle synchronization. It
// must also handle buffer swapping or copying if it wants to synchronize it
// away.
type Processor interface {
	Process()
}

type Sample = float64

// MakeBuffer allocates a slice of sample buffers.
func MakeBuffers(channels, samples int) [][]Sample {
	var buf = make([][]Sample, channels)
	for i := range buf {
		buf[i] = make([]Sample, samples)
	}
	return buf
}

// EnsureBufferLen ensures that the given buffer has matching sizes with the
// needed parameters from SessionConfig. It is effectively a bound check.
func EnsureBufferLen(cfg SessionConfig, buf [][]Sample) bool {
	if len(buf) != cfg.FrameSize {
		return false
	}
	for _, samples := range buf {
		if len(samples) != cfg.SampleSize {
			return false
		}
	}
	return true
}

// CopyBuffers deep copies src to dst. It does NOT do length check.
func CopyBuffers(dst, src [][]Sample) {
	frames := len(dst)
	size := len(dst[frames-1]) * frames
	for i := 0; i < size; i++ {
		dst[i%frames][i/frames] = src[i%frames][i/frames]
	}
}
