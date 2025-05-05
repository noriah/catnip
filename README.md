# catnip

[![love][withlove]][noriah-dev]
[![made-with-go][withgo]][go-dev]
[![betamax-was-a-thing][betamax]][betawhat]

> terminal audio visualizer for linux/unix/macOS/windows*

<p align="center">
	<a href="https://www.youtube.com/watch?v=S0LJCGOsq-4">
		<img src="../media/preview0.gif" />
	</a>
</p>
^ click for a fun video

[![A visualization of catnip over time](https://img.youtube.com/vi/sU3fkOB5HZ8/0.jpg)](https://youtu.be/sU3fkOB5HZ8)

## it supports audio backends
- ALSA (linux FFmpeg)
- AVFoundation (macOS FFmpeg)
- DirectShow (windblows FFmpeg)
- Pipewire (pw-cat)
- PortAudio (linux/macOS/windblows (maybe))
- PulseAudio (parec/FFmpeg)

## it depends on

- go modules
	- github.com/nsf/termbox-go
	- github.com/integrii/flaggy
	- github.com/pkg/errors
	- github.com/noisetorch/pulseaudio
	- gonum.org/v1/gonum

- binaries
	- ffmpeg (required for FFmpeg backends)
	- parec (required for PulseAudio backend with parec)
  - pw-cat, pw-link (required for Pipewire backend)

- c libraries (optional, requires CGO - `CGO_ENABLED=1`)
	- fftw (fftw3) (enable with `-tags fftw`)
	- portaudio (portaudio-2.0) (enable with `-tags portaudio`)

## get it

```sh
# get source
git clone https://github.com/noriah/catnip.git

# cd to dir
cd catnip

# build and install catnip
go install ./cmd/catnip

# with portaudio
go install ./cmd/catnip -tags portaudio

# with fftw3
go install ./cmd/catnip -tags fftw

# with both portaudio and fftw3
go install ./cmd/catnip -tags portaudio,fftw
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
[betawhat]: https://en.wikipedia.org/wiki/Betamax
[speakers]: https://github.com/noriah/catnip/commit/b1dc3840fa0ed583eba40dbaaa2c0c34c425e26e
[speakers-2]: https://github.com/noriah/catnip/commit/d3c13fb16742184d7c506a567b938045f3be1c1a

<!-- Images -->
[withlove]: https://forthebadge.com/images/badges/built-with-love.svg
[withgo]: https://forthebadge.com/images/badges/made-with-go.svg
[betamax]: https://forthebadge.com/images/badges/compatibility-betamax.svg
[preview-0]: https://i.imgur.com/TfMrNpe.gifv
