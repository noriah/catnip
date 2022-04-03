//go:build cgo && ((!windows && !noportaudio) || (windows && portonwin))

package main

import _ "github.com/noriah/catnip/input/portaudio"
