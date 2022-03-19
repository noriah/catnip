//go:build cgo && !noportaudio
// +build cgo,!noportaudio

package main

import _ "github.com/noriah/catnip/input/portaudio"
