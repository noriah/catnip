package ffmpeg

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/noriah/tavis/input"
	"github.com/pkg/errors"
)

func init() {
	input.RegisterBackend("ffmpeg-alsa", ALSA{})
}

type ALSA struct{}

func (p ALSA) Init() error {
	return nil
}

func (p ALSA) Close() error {
	return nil
}

func (p ALSA) Devices() ([]input.Device, error) {
	f, err := os.Open("/proc/asound/pcm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to open pcm")
	}
	defer f.Close()

	var devices []input.Device

	var scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		prefix := strings.Split(scanner.Text(), ":")[0]

		d, err := ParseALSADevice(prefix)
		if err != nil {
			return nil, fmt.Errorf("failed to parse device %q: %w", prefix, err)
		}

		devices = append(devices, d)
	}

	return devices, nil
}

func (p ALSA) DefaultDevice() (input.Device, error) {
	return ALSADevice("default"), nil
}

func (p ALSA) Start(cfg input.SessionConfig) (input.Session, error) {
	dv, ok := cfg.Device.(ALSADevice)
	if !ok {
		return nil, fmt.Errorf("invalid device type %T", cfg.Device)
	}

	return NewSession(dv, cfg)
}

// ALSADevice is a string that is the path to /dev/audioN.
type ALSADevice string

// ParseALSADevice parses %d:%d
func ParseALSADevice(hwString string) (ALSADevice, error) {
	nparts := strings.Split(hwString, "-")
	alsadv := "hw"

	if len(nparts) == 0 || len(nparts) > 2 {
		return "", fmt.Errorf("mismatch alsa format")
	}

	for i, part := range nparts {
		// Trim prefixed zeros.
		part = strings.TrimPrefix(part, "0")

		switch i {
		case 0:
			alsadv += ":" + part
		case 1:
			alsadv += "," + part
		}
	}

	return ALSADevice(alsadv), nil
}

func (d ALSADevice) InputArgs() []string {
	return []string{"-f", "alsa", "-i", string(d)}
}

func (d ALSADevice) String() string {
	return string(d)
}
