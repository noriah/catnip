//go:build (cgo && !windows && !noportaudio) || (cgo && windows && enablePortaudio)

package main

import _ "github.com/noriah/catnip/input/portaudio"
