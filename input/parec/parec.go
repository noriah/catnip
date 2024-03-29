package parec

import (
	"fmt"

	"github.com/noisetorch/pulseaudio"
	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/input/common/execread"
	"github.com/pkg/errors"
)

func init() {
	input.RegisterBackend("parec", Backend{})
}

type Backend struct{}

func (p Backend) Init() error {
	return nil
}

func (p Backend) Close() error {
	return nil
}

func (p Backend) Devices() ([]input.Device, error) {
	c, err := pulseaudio.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}
	defer c.Close()

	s, err := c.Sources()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sources")
	}

	devices := make([]input.Device, len(s))
	for i, source := range s {
		devices[i] = PulseDevice(source.Name)
	}

	return devices, nil
}

func (p Backend) DefaultDevice() (input.Device, error) {
	return PulseDevice(""), nil
}

func (p Backend) Start(cfg input.SessionConfig) (input.Session, error) {
	return NewSession(cfg)
}

type PulseDevice string

// InputArgs is used for FFmpeg only.
func (d PulseDevice) InputArgs() []string {
	return []string{"-f", "pulse", "-i", string(d)}
}

func (d PulseDevice) String() string {
	return string(d)
}

func NewSession(cfg input.SessionConfig) (*execread.Session, error) {
	dv, ok := cfg.Device.(PulseDevice)
	if !ok {
		return nil, fmt.Errorf("invalid device type %T", cfg.Device)
	}

	if cfg.FrameSize > 2 {
		return nil, errors.New("channel count not supported, mono/stereo only")
	}

	args := []string{
		"parec",
		"--format=float32le",
		fmt.Sprintf("--rate=%.0f", cfg.SampleRate),
		fmt.Sprintf("--channels=%d", cfg.FrameSize),
	}

	if dv != "" {
		args = append(args, "-d", dv.String())
	}

	return execread.NewSession(args, true, cfg), nil
}
