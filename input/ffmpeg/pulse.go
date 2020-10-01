package ffmpeg

import (
	"fmt"

	"github.com/noriah/tavis/input"
	"github.com/noriah/tavis/input/parec"
)

func init() {
	input.RegisterBackend("ffmpeg-pulse", Pulse{})
}

// Pulse is the pulse input for FFmpeg.
type Pulse struct {
	parec.Backend
}

func (p Pulse) Start(cfg input.SessionConfig) (input.Session, error) {
	dv, ok := cfg.Device.(parec.PulseDevice)
	if !ok {
		return nil, fmt.Errorf("invalid device type %T", cfg.Device)
	}

	return NewSession(dv, cfg)
}
