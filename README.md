tavis
===

[![love][withlove]][noriah-dev]
[![made-with-go][withgo]][go-dev]
[![betamax-was-a-thing][betamax]][betawhat]

> terminal audio visualizer for linux/unix/macOS/windows*

<p align="center">
	<a href="https://www.youtube.com/watch?v=NGtCoEsgJww" target="_blank">
		<img src="../media/preview0.gif?raw=true"/>
	</a>
</p>

## early development - expect things to change and break

we are working on this project all the time. its a sort of time filler for us at this point. expect lots of additions and changes at random times.

*windows needs work

## supported audio backends
- PortAudio (linux/macOS/*windblows**)
- PulseAudio (parec/FFmpeg)
- AVFoundation (FFmpeg)
- ALSA (FFmpeg)

## dependencies

- go modules
	- github.com/nsf/termbox-go
	- github.com/urfave/cli/v2
	- github.com/pkg/errors
	- github.com/lawl/pulseaudio
	- gonum.org/v1/gonum

- c libraries (optional, disable with `CGO_ENABLED=0`)
	- fftw (fftw3)
	- portaudio (portaudio-2.0)

- binaries (optional)
	- ffmpeg
	- parec

## installation

### with `go get`

```sh
# with cgo (fftw, portaudio)
go get github.com/noriah/tavis
# without cgo
CGO_ENABLED=0 go get github.com/noriah/tavis
```

### with `git`

```sh
# get source
git clone https://github.com/noriah/tavis.git

# cd to source
cd tavis

# build and install tavis
go install
# without cgo
CGO_ENABLED=0 go install
```

## usage

```sh
NAME:
   tavis - terminal audio visualizer

USAGE:
   tavis [global options] command [command options] [arguments...]

COMMANDS:
   list-backends
   list-devices
   help, h        Shows a list of commands or help for one command

```

- use `tavis list-backends` to show available backends
- use `tavis -b {backend} list-devices` to show available devices
- use `tavis -b {backend} -d {device}` to run - use the full device name
- use `tavis -h` for information on several more customizations


<!-- Links -->
[noriah-dev]: https://noriah.dev
[go-dev]: https://go.dev
[betawhat]: https://google.com/search?q=betamax


<!-- Images -->
[withlove]: https://forthebadge.com/images/badges/built-with-love.svg
[withgo]: https://forthebadge.com/images/badges/made-with-go.svg
[betamax]: https://forthebadge.com/images/badges/compatibility-betamax.svg

