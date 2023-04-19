//go:build cgo && (withportaudio || portaudio)

package all

import _ "github.com/noriah/catnip/input/portaudio"
