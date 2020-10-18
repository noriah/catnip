# terminal audio visualizer - tavis

[![love][withlove]][noriah-dev]
[![made-with-go][withgo]][go-dev]
[![betamax-was-a-thing][betamax]][betawhat]

tavis is a terminal audio visualizer for linux/unix/macOS/windows*.
powered by go, it can pull from PortAudio, PulseAudio, FFmpeg.

## tavis is in early development. expect things to change and break

run `tavis -h` for usage

#### depends on

- tcell
- gonum
- Optional:
	- `CGO_ENABLED=1`
		- portaudio-2.0
		- fftw3
	- ffmpeg (non-cgo fallback)
		- avfoundation
		- alsa
		- pulse
		- sndio
	- parec (pulse only)


<!-- Links -->
[noriah-dev]: https://noriah.dev
[go-dev]: https://go.dev
[betawhat]: https://google.com/search?q=betamax

<!-- Images -->
[withlove]: https://forthebadge.com/images/badges/built-with-love.svg
[withgo]: https://forthebadge.com/images/badges/made-with-go.svg
[betamax]: https://forthebadge.com/images/badges/compatibility-betamax.svg
