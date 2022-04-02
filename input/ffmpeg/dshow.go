//go:build windows

package ffmpeg

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/input/common/execread"
)

func init() {
	input.RegisterBackend("ffmpeg-dshow", DShow{})
}

func NewWindowsSession(b FFmpegBackend, cfg input.SessionConfig) (*execread.Session, error) {
	args := []string{"ffmpeg", "-hide_banner", "-loglevel", "panic"}
	args = append(args,
		"-f", "dshow", "-audio_buffer_size", "20",
		"-sample_rate", fmt.Sprintf("%.0f", cfg.SampleRate),
		"-channels", fmt.Sprintf("%d", cfg.FrameSize),
		// "-sample_size", fmt.Sprintf("%d", 16),
	)
	args = append(args, b.InputArgs()...)
	args = append(args, "-f", "f64le", "-")

	return execread.NewSession(args, false, cfg)
}

// DShow is the DirectShow input for FFmpeg on Windows.
type DShow struct{}

func (p DShow) Init() error {
	return nil
}

func (p DShow) Close() error {
	return nil
}

// Devices returns a list of dshow devices.
func (p DShow) Devices() ([]input.Device, error) {
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-loglevel", "info",
		"-f", "dshow", "-list_devices", "true",
		"-i", "",
	)

	o, _ := cmd.CombinedOutput()

	audio := true
	var devices []input.Device

	var scanner = bufio.NewScanner(bytes.NewReader(o))

	for scanner.Scan() {
		text := scanner.Text()

		// Trim away the prefix.
		if strings.HasPrefix(text, "[dshow") {
			parts := strings.SplitN(text, "] ", 2)
			if len(parts) == 2 {
				text = parts[1][1:]
			}
		}

		// If we're not scanning a device (which starts with a square bracket)
		// anymore, then we stop.
		if strings.HasPrefix(text, ":") {
			audio = false
			continue
		}

		// If we're not under the audio section yet, then skip.
		if !audio {
			continue
		}

		// Parse.
		parts := strings.SplitN(text, "\" (", 2)
		if len(parts) != 2 {
			continue
		}

		if !strings.HasPrefix(parts[1], "audio") {
			continue
		}

		devices = append(devices, DShowDevice{
			Name: parts[0],
		})
	}

	if len(devices) == 0 {
		// This is completely for visual.
		var lines = strings.Split(string(o), "\n")
		for i, line := range lines {
			lines[i] = "\t" + line
		}
		var output = strings.Join(lines, "\n")

		return nil, fmt.Errorf("no devices found; ffmpeg output:\n%s", output)
	}

	return devices, nil
}

func (p DShow) DefaultDevice() (input.Device, error) {
	return DShowDevice{"no default device"}, nil
}

func (p DShow) Start(cfg input.SessionConfig) (input.Session, error) {
	dv, ok := cfg.Device.(DShowDevice)
	if !ok {
		return nil, fmt.Errorf("invalid device type %T", cfg.Device)
	}

	return NewWindowsSession(dv, cfg)
}

type DShowDevice struct {
	Name string
}

func (d DShowDevice) InputArgs() []string {
	input := fmt.Sprintf("audio=%s", d.Name)
	return []string{"-i", input}
}

func (d DShowDevice) String() string {
	return fmt.Sprintf("%s", d.Name)
}
