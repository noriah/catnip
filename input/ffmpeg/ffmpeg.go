package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/input/common/execread"
)

type FFmpegBackend interface {
	InputArgs() []string
}

func NewSession(b FFmpegBackend, cfg input.SessionConfig) (*execread.Session, error) {
	args := []string{"-hide_banner", "-loglevel", "panic"}
	args = append(args, b.InputArgs()...)
	args = append(args,
		"-ar", fmt.Sprintf("%.0f", cfg.SampleRate),
		"-ac", fmt.Sprintf("%d", cfg.FrameSize),
		"-f", "f64le",
		"-",
	)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr

	return execread.NewSession(cmd, false, cfg)
}
