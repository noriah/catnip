package input

import (
	"context"
)

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

type Session interface {
	Start() error
	Stop() error

	SampleBuffers() [][]Sample
	ReadyRead() int
	Read(context.Context) error
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
