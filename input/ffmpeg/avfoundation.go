//go:build darwin

package ffmpeg

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/noriah/catnip/input"
	"github.com/pkg/errors"
)

func init() {
	input.RegisterBackend("ffmpeg-avfoundation", AVFoundation{})
}

// AVFoundation is the avfoundation input for FFmpeg.
type AVFoundation struct{}

func (p AVFoundation) Init() error {
	return nil
}

func (p AVFoundation) Close() error {
	return nil
}

// Devices returns a list of avfoundation devices from /dev/audio*. This is
// kernel-specific and is only known to work on OpenBSD.
func (p AVFoundation) Devices() ([]input.Device, error) {
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-loglevel", "info",
		"-f", "avfoundation", "-list_devices", "true",
		"-i", "",
	)

	o, _ := cmd.CombinedOutput()

	var audio bool
	var devices []input.Device

	scanner := bufio.NewScanner(bytes.NewReader(o))
	for scanner.Scan() {
		text := scanner.Text()

		// Trim away the prefix.
		if strings.HasPrefix(text, "[AVFoundation") {
			parts := strings.SplitN(text, "] ", 2)
			if len(parts) == 2 {
				text = parts[1]
			}
		}

		// If we're starting the audio devices section, then mark the boolean to
		// scan.
		if text == "AVFoundation audio devices:" {
			audio = true
			continue
		}

		// If we're not scanning a device (which starts with a square bracket)
		// anymore, then we stop.
		if !strings.HasPrefix(text, "[") {
			audio = false
			continue
		}

		// If we're not under the audio section yet, then skip.
		if !audio {
			continue
		}

		// Parse.
		parts := strings.SplitN(text, " ", 2)
		if len(parts) != 2 {
			continue
		}

		n, err := strconv.Atoi(strings.Trim(parts[0], "[]"))
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse device index")
		}

		devices = append(devices, AVFoundationDevice{
			Index: n,
			Name:  parts[1],
		})
	}

	if len(devices) == 0 {
		// This is completely for visual.
		lines := strings.Split(string(o), "\n")
		for i, line := range lines {
			lines[i] = "\t" + line
		}
		output := strings.Join(lines, "\n")

		return nil, fmt.Errorf("no devices found; ffmpeg output:\n%s", output)
	}

	return devices, nil
}

func (p AVFoundation) DefaultDevice() (input.Device, error) {
	return AVFoundationDevice{-1, "default"}, nil
}

func (p AVFoundation) Start(cfg input.SessionConfig) (input.Session, error) {
	dv, ok := cfg.Device.(AVFoundationDevice)
	if !ok {
		return nil, fmt.Errorf("invalid device type %T", cfg.Device)
	}

	return NewSession(dv, cfg)
}

type AVFoundationDevice struct {
	Index int
	Name  string
}

func (d AVFoundationDevice) InputArgs() []string {
	input := "none:default"
	if d.Index > -1 {
		input = fmt.Sprintf("none:%d", d.Index)
	}
	return []string{"-f", "avfoundation", "-i", input}
}

func (d AVFoundationDevice) String() string {
	return fmt.Sprintf("%d:%s", d.Index, d.Name)
}
