package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/noriah/tavis/display"
	"github.com/noriah/tavis/dsp"
	"github.com/noriah/tavis/input"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	// Input backends.
	_ "github.com/noriah/tavis/input/ffmpeg"
	_ "github.com/noriah/tavis/input/parec"
)

// Device is a temporary struct to define parameters
type Device struct {
	// InputBackend is the backend that the input belongs to
	InputBackend input.Backend
	// InputDevice is the device we want to listen to
	InputDevice input.Device
	// SampleRate is the rate at which samples are read
	SampleRate float64
	//LoCutFrqq is the low end of our audio spectrum
	LoCutFreq float64
	// HiCutFreq is the high end of our audio spectrum
	HiCutFreq float64
	// SmoothFactor factor of smooth
	SmoothFactor float64
	// SmoothResponse response value
	SmoothResponse float64
	// BaseThick number of cells wide/high the base is
	BaseThick int
	// BarWidth is the width of bars, in columns
	BarWidth int
	// SpaceWidth is the width of spaces, in columns
	SpaceWidth int
	// TargetFPS is how fast we want to redraw. Play with it
	TargetFPS int
	// ChannelCount is the number of channels we want to look at. DO NOT TOUCH
	ChannelCount int
	// MaxBins maximum number of bins
	MaxBins int
}

var (
	device = Device{
		SampleRate:     44100,
		LoCutFreq:      20,
		HiCutFreq:      22050,
		SmoothFactor:   52.5,
		SmoothResponse: 43.5,
		BaseThick:      1,
		BarWidth:       2,
		SpaceWidth:     1,
		TargetFPS:      60,
		ChannelCount:   2,
		MaxBins:        256,
	}
)

func init() {
	log.SetFlags(0)
}

func main() {
	app := cli.App{
		Name:   "tavis",
		Usage:  "terminal audio visualizer",
		Action: run,
		Commands: []*cli.Command{
			{
				Name:        "list-backends",
				Action:      listBackends,
				Description: "List all available backends",
			},
			{
				Name:        "list-devices",
				Action:      listDevices,
				Description: "List all visible input devices",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "backend",
				Value:   "portaudio",
				Aliases: []string{"b"},
			},
			&cli.StringFlag{
				Name:    "device",
				Aliases: []string{"d"},
			},
			&cli.Float64Flag{
				Name:        "sample-rate",
				Aliases:     []string{"r"},
				Value:       device.SampleRate,
				Destination: &device.SampleRate,
			},
			&cli.Float64Flag{
				Name:        "low-cut-freq",
				Aliases:     []string{"lf"},
				Value:       device.LoCutFreq,
				Destination: &device.LoCutFreq,
			},
			&cli.Float64Flag{
				Name:        "high-cut-freq",
				Aliases:     []string{"hf"},
				Value:       device.HiCutFreq,
				Destination: &device.HiCutFreq,
			},
			&cli.Float64Flag{
				Name:        "smoothness-factor",
				Aliases:     []string{"sf"},
				Value:       device.SmoothFactor,
				Destination: &device.SmoothFactor,
			},
			&cli.Float64Flag{
				Name:        "smoothness-response",
				Aliases:     []string{"sr"},
				Value:       device.SmoothResponse,
				Destination: &device.SmoothResponse,
			},
			&cli.IntFlag{
				Name:        "base-thickness",
				Aliases:     []string{"bt"},
				Value:       device.BaseThick,
				Destination: &device.BaseThick,
			},
			&cli.IntFlag{
				Name:        "bar-width",
				Aliases:     []string{"bw"},
				Value:       device.BarWidth,
				Destination: &device.BarWidth,
			},
			&cli.IntFlag{
				Name:        "space-width",
				Aliases:     []string{"sw"},
				Value:       device.SpaceWidth,
				Destination: &device.SpaceWidth,
			},
			&cli.IntFlag{
				Name:        "target-fps",
				Aliases:     []string{"f"},
				Value:       device.TargetFPS,
				Destination: &device.TargetFPS,
			},
			&cli.IntFlag{
				Name:        "channel-count",
				Aliases:     []string{"c"},
				Hidden:      true,
				Value:       device.ChannelCount,
				Destination: &device.ChannelCount,
			},
			&cli.IntFlag{
				Name:        "max-bins",
				Aliases:     []string{"mb"},
				Hidden:      true,
				Value:       device.MaxBins,
				Destination: &device.MaxBins,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln("Error:", err)
	}
}

func listBackends(c *cli.Context) error {
	for _, backend := range input.Backends {
		fmt.Printf("- %s\n", backend.Name)
	}
	return nil
}

func initBackend(c *cli.Context) error {
	backendName := c.String("backend")

	device.InputBackend = input.FindBackend(backendName)
	if device.InputBackend == nil {
		return fmt.Errorf("backend not found: %q", backendName)
	}

	if err := device.InputBackend.Init(); err != nil {
		return errors.Wrap(err, "failed to initialize input backend")
	}

	return nil
}

func listDevices(c *cli.Context) error {
	if err := initBackend(c); err != nil {
		return err
	}

	devices, err := device.InputBackend.Devices()
	if err != nil {
		return errors.Wrap(err, "failed to get devices")
	}

	// optional default device
	defaultDevice, _ := device.InputBackend.DefaultDevice()

	for _, device := range devices {
		fmt.Printf("- %v", device)

		if defaultDevice != nil && device.String() == defaultDevice.String() {
			fmt.Print(" (default)")
		}

		fmt.Println()
	}

	return nil
}

func initInputDevice(c *cli.Context) error {
	deviceName := c.String("device")
	if deviceName == "" {
		def, err := device.InputBackend.DefaultDevice()
		if err != nil {
			return errors.Wrap(err, "failed to get default device")
		}

		device.InputDevice = def
		return nil
	}

	devices, err := device.InputBackend.Devices()
	if err != nil {
		return errors.Wrap(err, "failed to get devices")
	}

	for _, d := range devices {
		if d.String() == deviceName {
			device.InputDevice = d
			return nil
		}
	}

	return fmt.Errorf("device %q not found; check list-devices", deviceName)
}

func run(c *cli.Context) error {
	if err := initBackend(c); err != nil {
		return err
	}

	if err := initInputDevice(c); err != nil {
		return err
	}

	return visualize(device)
}

type channel struct {
	bins *dsp.BinSet
	n2s3 *dsp.N2S3State
}

// Run starts to draw the visualizer on the tcell Screen.
func visualize(d Device) error {
	var (
		// SampleSize is the number of frames per channel we want per read
		sampleSize = int(d.SampleRate / float64(d.TargetFPS))

		// DrawDelay is the time we wait between ticks to draw.
		drawDelay = time.Second / time.Duration(d.TargetFPS)
	)

	var source, err = d.InputBackend.Start(input.SessionConfig{
		Device:     d.InputDevice,
		FrameSize:  d.ChannelCount,
		SampleSize: sampleSize,
		SampleRate: d.SampleRate,
	})

	switch {
	case d.SmoothFactor > 100.0:
		d.SmoothFactor = 1.0
	case d.SmoothFactor < 0.0:
		d.SmoothFactor = 0.0
	default:
		d.SmoothFactor /= 100.0
	}

	switch {
	case d.SmoothResponse < 0.1:
		d.SmoothResponse = 0.1
	default:
	}

	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	var display = display.New(d.SampleRate, sampleSize)
	defer display.Close()

	var barCount = display.SetWidths(d.BarWidth, d.SpaceWidth)

	// Make a spectrum
	var spectrum = dsp.NewSpectrum(d.SampleRate, sampleSize, d.MaxBins)

	// Set it up with our values
	barCount = spectrum.Recalculate(barCount, d.LoCutFreq, d.HiCutFreq)

	if err := source.Start(); err != nil {
		return errors.Wrap(err, "failed to start input session")
	}
	defer source.Stop()

	var bufs = source.SampleBuffers()

	var channels = make([]*channel, d.ChannelCount)

	var chanBins = make([][]float64, d.ChannelCount)

	for ch := 0; ch < d.ChannelCount; ch++ {
		channels[ch] = &channel{
			bins: spectrum.BinSet(bufs[ch]),
			n2s3: dsp.NewN2S3State(d.MaxBins),
		}

		chanBins[ch] = channels[ch].bins.Bins()
	}
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var dispCtx = display.Start(ctx)
	defer display.Stop()

	var endSig = make(chan os.Signal, 3)
	signal.Notify(endSig, os.Interrupt)

	var tick = time.Now()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-dispCtx.Done():
			return nil
		case <-endSig:
			return nil
		default:
		}

		if since := time.Since(tick); since < drawDelay {
			time.Sleep(drawDelay - since)
		}

		tick = time.Now()

		var winWidth = display.Bars()

		if barCount != winWidth {
			barCount = winWidth
			barCount = spectrum.Recalculate(barCount, d.LoCutFreq, d.HiCutFreq)
		}

		if source.ReadyRead() < sampleSize {
			continue
		}

		if err := source.Read(ctx); err != nil {
			return errors.Wrap(err, "failed to read audio input")
		}

		for ch := range channels {
			spectrum.Generate(channels[ch].bins)

			// nora's not so special smoother (n2s3)
			dsp.N2S3(chanBins[ch], barCount,
				channels[ch].n2s3, d.SmoothFactor, d.SmoothResponse)
		}

		display.Draw(chanBins, barCount, d.BaseThick)
	}
}
