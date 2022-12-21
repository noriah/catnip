package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/graphic"
	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/processor"

	_ "github.com/noriah/catnip/input/ffmpeg"
	_ "github.com/noriah/catnip/input/parec"

	"github.com/integrii/flaggy"
	"github.com/pkg/errors"
)

// AppName is the app name
const AppName = "catnip"

// AppDesc is the app description
const AppDesc = "Continuous Automatic Terminal Number Interpretation Printer"

// AppSite is the app website
const AppSite = "https://github.com/noriah/catnip"

var version = "unknown"

func main() {
	log.SetFlags(0)

	cfg := newZeroConfig()

	if doFlags(&cfg) {
		return
	}

	chk(cfg.Sanitize(), "invalid config")

	chk(catnip(&cfg), "failed to run catnip")
}

// Catnip starts to draw the processor on the termbox screen.
func catnip(cfg *config) error {

	display := &graphic.Display{}

	// PROCESSOR SETUP

	inputBuffers := input.MakeBuffers(cfg.channelCount, cfg.sampleSize)
	// visBuffers := input.MakeBuffers(cfg.channelCount, cfg.sampleSize)

	procConfig := processor.Config{
		SampleRate:   cfg.sampleRate,
		SampleSize:   cfg.sampleSize,
		ChannelCount: cfg.channelCount,
		FrameRate:    cfg.frameRate,
		InvertDraw:   cfg.invertDraw,
		Buffers:      inputBuffers,
		Analyzer: dsp.NewAnalyzer(dsp.AnalyzerConfig{
			SampleRate: cfg.sampleRate,
			SampleSize: cfg.sampleSize,
		}),
		Smoother: dsp.NewSmoother(dsp.SmootherConfig{
			SampleSize:      cfg.sampleSize,
			ChannelCount:    cfg.channelCount,
			SmoothingFactor: cfg.smoothFactor}),
		Output: display,
	}

	var vis processor.Processor

	if cfg.useThreaded {
		vis = processor.NewThreaded(procConfig)
	} else {
		vis = processor.New(procConfig)
	}

	// INPUT SETUP

	backend, err := input.InitBackend(cfg.backend)
	if err != nil {
		return err
	}

	sessConfig := input.SessionConfig{
		FrameSize:  cfg.channelCount,
		SampleSize: cfg.sampleSize,
		SampleRate: cfg.sampleRate,
	}

	if sessConfig.Device, err = input.GetDevice(backend, cfg.device); err != nil {
		return err
	}

	audio, err := backend.Start(sessConfig)
	defer backend.Close()

	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	// DISPLAY SETUP

	if err = display.Init(cfg.sampleRate, cfg.sampleSize); err != nil {
		return err
	}
	defer display.Close()

	display.SetSizes(cfg.barSize, cfg.spaceSize)
	display.SetBase(cfg.baseSize)
	display.SetDrawType(graphic.DrawType(cfg.drawType))
	display.SetStyles(cfg.styles)
	display.SetInvertDraw(cfg.invertDraw)

	// Root Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = display.Start(ctx)
	defer display.Stop()

	ctx = vis.Start(ctx)
	defer vis.Stop()

	kickChan := make(chan bool, 1)

	mu := &sync.Mutex{}

	// Start the processor
	go vis.Process(ctx, kickChan, mu)

	if err := audio.Start(ctx, inputBuffers, kickChan, mu); err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			return errors.Wrap(err, "failed to start input session")
		}
	}

	return nil
}

// NewZeroConfig returns a zero config
// it is the "default"
//
// nori's defaults:
//   - sampleRate: 122880
//   - sampleSize: 2048
//   - smoothFactor: 80.15
//   - super smooth detail view
func newZeroConfig() config {
	return config{
		sampleRate:   44100,
		sampleSize:   1024,
		smoothFactor: 80.15,
		frameRate:    0,
		baseSize:     1,
		barSize:      2,
		spaceSize:    1,
		channelCount: 2,
		drawType:     int(graphic.DrawDefault),
		combine:      false,
		useThreaded:  false,
	}
}

func doFlags(cfg *config) bool {

	parser := flaggy.NewParser(AppName)
	parser.Description = AppDesc
	parser.AdditionalHelpPrepend = AppSite
	parser.Version = version

	listBackendsCmd := flaggy.Subcommand{
		Name:                 "list-backends",
		ShortName:            "lb",
		Description:          "list all supported backends",
		AdditionalHelpAppend: "\nuse the full name after the '-'",
	}

	parser.AttachSubcommand(&listBackendsCmd, 1)

	listDevicesCmd := flaggy.Subcommand{
		Name:                 "list-devices",
		ShortName:            "ld",
		Description:          "list all devices for a backend",
		AdditionalHelpAppend: "\nuse the full name after the '-'",
	}

	parser.AttachSubcommand(&listDevicesCmd, 1)

	parser.String(&cfg.backend, "b", "backend", "backend name")
	parser.String(&cfg.device, "d", "device", "device name")
	parser.Float64(&cfg.sampleRate, "r", "rate", "sample rate")
	parser.Int(&cfg.sampleSize, "n", "samples", "sample size")
	parser.Int(&cfg.frameRate, "f", "fps", "frame rate (0 to draw on every sample)")
	parser.Int(&cfg.channelCount, "ch", "channels", "channel count (1 or 2)")
	parser.Float64(&cfg.smoothFactor, "sf", "smoothing", "smooth factor (0-100)")
	parser.Int(&cfg.baseSize, "bt", "base", "base thickness [0, +Inf)")
	parser.Int(&cfg.barSize, "bw", "bar", "bar width [1, +Inf)")
	parser.Int(&cfg.spaceSize, "sw", "space", "space width [0, +Inf)")
	parser.Int(&cfg.drawType, "dt", "draw", "draw type (1, 2, 3, 4, 5, 6)")
	parser.Bool(&cfg.useThreaded, "t", "threaded", "use the threaded processor")
	parser.Bool(&cfg.invertDraw, "i", "invert", "invert the direction of bin drawing")

	fg, bg, center := graphic.DefaultStyles().AsUInt16s()
	parser.UInt16(&fg, "fg", "foreground",
		"foreground color within the 256-color range [0, 255] with attributes")
	parser.UInt16(&bg, "bg", "background",
		"background color within the 256-color range [0, 255] with attributes")
	parser.UInt16(&center, "ct", "center",
		"center line color within the 256-color range [0, 255] with attributes")

	chk(parser.Parse(), "failed to parse arguments")

	// Manually set the styles.
	cfg.styles = graphic.StylesFromUInt16(fg, bg, center)

	switch {
	case listBackendsCmd.Used:
		for _, backend := range input.Backends {
			fmt.Printf("- %s\n", backend.Name)
		}

		return true

	case listDevicesCmd.Used:
		backend, err := input.InitBackend(cfg.backend)
		chk(err, "failed to init backend")

		devices, err := backend.Devices()
		chk(err, "failed to get devices")

		// We don't really need the default device to be indicated.
		defaultDevice, _ := backend.DefaultDevice()

		fmt.Printf("all devices for %q backend. '*' marks default\n", cfg.backend)

		for idx := range devices {
			star := ' '
			if defaultDevice != nil && devices[idx].String() == defaultDevice.String() {
				star = '*'
			}

			fmt.Printf("- %v %c\n", devices[idx], star)
		}

		return true
	}

	return false
}

func chk(err error, wrap string) {
	if err != nil {
		log.Fatalln(wrap+": ", err)
	}
}
