# catnip

[![love][withlove]][noriah-dev]
[![made-with-go][withgo]][go-dev]
[![betamax-was-a-thing][betamax]][betawhat]

> terminal audio visualizer for linux/unix/macOS/windows*

<p align="center">
	<a href="https://www.youtube.com/watch?v=NGtCoEsgJww">
		<img src="../media/preview0.gif" />
	</a>
</p>

*windows needs work

## it supports audio backends
- PortAudio (linux/macOS/*windblows**)
- PulseAudio (parec/FFmpeg)
- AVFoundation (FFmpeg)
- ALSA (FFmpeg)

## it depends on

- go modules
	- github.com/nsf/termbox-go
	- github.com/integrii/flaggy
	- github.com/pkg/errors
	- github.com/lawl/pulseaudio
	- gonum.org/v1/gonum

- c libraries (optional, disable with `CGO_ENABLED=0`)
	- fftw (fftw3)
	- portaudio (portaudio-2.0) (disable with `-tags noportaudio`)

- binaries
	- ffmpeg (required for FFmpeg backends)
	- parec (required for PulseAudio backend with parec)

## get it

```sh
# get source
git clone https://github.com/noriah/catnip.git

# cd to source
cd catnip

# build and install catnip
go install

# without cgo
CGO_ENABLED=0 go install

# without portaudio
go install -tags noportaudio
```

## run it

- use `catnip list-backends` to show available backends
- use `catnip -b {backend} list-devices` to show available devices
- use `catnip -b {backend} -d {device}` to run - use the full device name
- use `catnip -h` for information on several more customizations

## question it
### catnip?
[long story, short explanation][speakers]

[update][speakers-2]

<!-- Links -->
[noriah-dev]: https://noriah.dev
[go-dev]: https://go.dev
[betawhat]: https://google.com/search?q=betamax
[speakers]: https://github.com/noriah/catnip/commit/98f989fd45bef8706cbc5c90422209600943ebc1
[speakers-2]: https://github.com/noriah/catnip/commit/d2b559a09936f1018d53a05c95cdfa12dd383105

<!-- Images -->
[withlove]: https://forthebadge.com/images/badges/built-with-love.svg
[withgo]: https://forthebadge.com/images/badges/made-with-go.svg
[betamax]: https://forthebadge.com/images/badges/compatibility-betamax.svg
[preview-0]: https://i.imgur.com/TfMrNpe.gifv
