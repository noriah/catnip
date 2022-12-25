package ffmpeg

import (
	"fmt"

	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/input/common/execread"
)

type FFmpegBackend interface {
	InputArgs() []string
}

func NewSession(b FFmpegBackend, cfg input.SessionConfig) (*execread.Session, error) {
	args := []string{"ffmpeg", "-hide_banner", "-loglevel", "panic"}
	args = append(args, b.InputArgs()...)
	args = append(args,
		"-ar", fmt.Sprintf("%.0f", cfg.SampleRate),
		"-ac", fmt.Sprintf("%d", cfg.FrameSize),
		"-f", "f64le",
		"-",
	)

	return execread.NewSession(args, false, cfg), nil
}
