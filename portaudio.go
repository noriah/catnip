//go:build (cgo && !windows && !noportaudio) || (cgo && windows && portonwin)

package main

import _ "github.com/noriah/catnip/input/portaudio"
