// Package all imports all backends implemented by the input package.
package all

import (
	_ "github.com/noriah/catnip/input/ffmpeg"
	_ "github.com/noriah/catnip/input/parec"
	_ "github.com/noriah/catnip/input/pipewire"
)
