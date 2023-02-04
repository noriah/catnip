package input

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/pkg/errors"
)

type Backend interface {
	// Init should do nothing if called more than once.
	Init() error
	Close() error

	Devices() ([]Device, error)
	DefaultDevice() (Device, error)
	Start(SessionConfig) (Session, error)
}

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

// Get all installed backend names.
func GetAllBackendNames() []string {
	out := make([]string, len(Backends))
	for i, backend := range Backends {
		out[i] = backend.Name
	}
	return out
}

// Get the default backend depending on
func DefaultBackend() string {
	switch runtime.GOOS {
	case "windows":
		if HasBackend("ffmpeg-dshow") {
			return "ffmpeg-dshow"
		}

	case "darwin":
		if backend := FindBackend("portaudio"); backend != nil {
			if HasBackend("portaudio") {
				return "portaudio"
			}
		}

		if HasBackend("ffmpeg-avfoundation") {
			return "ffmpeg-avfoundation"
		}

	case "linux":
		if path, _ := exec.LookPath("pw-cat"); path != "" {
			if HasBackend("pipewire") {
				return "pipewire"
			}
		}

		if path, _ := exec.LookPath("parec"); path != "" {
			if HasBackend("parec") {
				return "parec"
			}
		}

		if HasBackend("ffmpeg-alsa") {
			return "ffmpeg-alsa"
		}
	}

	return ""
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

func HasBackend(name string) bool {
	for _, backend := range Backends {
		if backend.Name == name {
			return true
		}
	}
	return false
}

func InitBackend(bknd string) (Backend, error) {
	backend := FindBackend(bknd)
	if backend == nil {
		return nil, fmt.Errorf("backend not found: %q; check list-backends", bknd)
	}

	if err := backend.Init(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize input backend")
	}

	return backend, nil
}

func GetDevice(backend Backend, device string) (Device, error) {
	if device == "" {
		def, err := backend.DefaultDevice()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get default device")
		}
		return def, nil
	}

	devices, err := backend.Devices()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get devices")
	}

	for idx := range devices {
		if devices[idx].String() == device {
			return devices[idx], nil
		}
	}

	return nil, errors.Errorf("device %q not found; check list-devices", device)
}
