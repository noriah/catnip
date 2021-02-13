package input

import "context"

type NamedBackend struct {
	Name string
	Backend
}

var Backends []NamedBackend

// RegisterBackend registers a backend globally. This function is not
// thread-safe, and most packages should call it on init().
func RegisterBackend(name string, b Backend) {
	Backends = append(Backends, NamedBackend{
		Name:    name,
		Backend: b,
	})
}

// FindBackend is a helper function that finds a backend. It returns nil if the
// backend is not found.
func FindBackend(name string) Backend {
	for _, backend := range Backends {
		if backend.Name == name {
			return backend
		}
	}
	return nil
}

type Device interface {
	// String returns the device name.
	String() string
}

type Backend interface {
	// Init should do nothing if called more than once.
	Init() error
	Close() error

	Devices() ([]Device, error)
	DefaultDevice() (Device, error)
	Start(SessionConfig) (Session, error)
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
	Start(context.Context, [][]Sample, Processor) error
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
func MakeBuffers(cfg SessionConfig) [][]Sample {
	var buf = make([][]Sample, cfg.FrameSize)
	for i := range buf {
		buf[i] = make([]Sample, cfg.SampleSize)
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
	for i := range src {
		copy(dst[i], src[i])
	}
}
