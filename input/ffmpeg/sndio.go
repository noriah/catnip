package ffmpeg

import (
	"fmt"
	"path/filepath"

	"github.com/noriah/catnip/input"
	"github.com/pkg/errors"
)

func init() {
	input.RegisterBackend("ffmpeg-sndio", Sndio{})
}

// Sndio is the sndio input for FFmpeg.
type Sndio struct{}

func (p Sndio) Init() error {
	return nil
}

func (p Sndio) Close() error {
	return nil
}

// Devices returns a list of sndio devices from /dev/audio*. This is
// kernel-specific and is only known to work on OpenBSD.
func (p Sndio) Devices() ([]input.Device, error) {
	n, err := filepath.Glob("/dev/audio*")
	if err != nil {
		return nil, errors.Wrap(err, "failed to glob /dev/audio")
	}

	var devices = make([]input.Device, len(n))
	for i, path := range n {
		devices[i] = SndioDevice(path)
	}

	return devices, nil
}

func (p Sndio) DefaultDevice() (input.Device, error) {
	return SndioDevice("/dev/audio0"), nil
}

func (p Sndio) Start(cfg input.SessionConfig) (input.Session, error) {
	dv, ok := cfg.Device.(SndioDevice)
	if !ok {
		return nil, fmt.Errorf("invalid device type %T", cfg.Device)
	}

	return NewSession(dv, cfg)
}

// SndioDevice is a string that is the path to /dev/audioN.
type SndioDevice string

func (d SndioDevice) InputArgs() []string {
	return []string{"-f", "sndio", "-i", string(d)}
}

func (d SndioDevice) String() string {
	return string(d)
}
