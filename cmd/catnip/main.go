package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/noriah/catnip"
	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/dsp/window"
	"github.com/noriah/catnip/graphic"
	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/processor"

	_ "github.com/noriah/catnip/input/all"

	"github.com/integrii/flaggy"
)

// AppName is the app name
const AppName = "catnip"

// AppDesc is the app description
const AppDesc = "Continuous Automatic Terminal Number Interpretation Printer"

// AppSite is the app website
const AppSite = "https://github.com/noriah/catnip"

var version = "git"

func main() {
	log.SetFlags(0)

	cfg := newZeroConfig()

	if doFlags(&cfg) {
		return
	}

	chk(cfg.validate(), "invalid config")

	smoother := dsp.NewSmoother(dsp.SmootherConfig{
		SampleSize:      cfg.sampleSize,
		SampleRate:      cfg.sampleRate,
		ChannelCount:    cfg.channelCount,
		SmoothingFactor: cfg.smoothFactor,
		SmoothingMethod: dsp.SmoothingMethod(cfg.smoothingMethod),
		AverageSize:     cfg.smoothingAverageWindowSize,
	})

	display := graphic.NewDisplay()
	display.Smoother = smoother

	var output processor.Output
	output = display

	if cfg.useRawOutput {
		rawOutput := NewRawOutput()
		rawOutput.Init(cfg.sampleRate, cfg.sampleSize)
		rawOutput.SetBinCount(cfg.rawOutputBins)
		rawOutput.SetInvertDraw(cfg.invertDraw)
		rawOutput.SetMirrorOutput(cfg.rawOutputMirror)
		output = rawOutput
	}

	catnipCfg := catnip.Config{
		Backend:      cfg.backend,
		Device:       cfg.device,
		SampleRate:   cfg.sampleRate,
		SampleSize:   cfg.sampleSize,
		ChannelCount: cfg.channelCount,
		ProcessRate:  cfg.frameRate,
		Combine:      cfg.combine,
		UseThreaded:  cfg.useThreaded,
		SetupFunc:    setupFunc(!cfg.useRawOutput, &cfg, display),
		StartFunc:    startFunc(!cfg.useRawOutput, display),
		CleanupFunc:  cleanupFunc(!cfg.useRawOutput, display),
		Output:       output,
		Windower:     window.Lanczos(),
		Analyzer: dsp.NewAnalyzer(dsp.AnalyzerConfig{
			SampleRate:    cfg.sampleRate,
			SampleSize:    cfg.sampleSize,
			SquashLow:     true,
			SquashLowOld:  true,
			DontNormalize: cfg.dontNormalize,
			BinMethod:     dsp.MaxSampleValue(),
		}),
		Smoother: smoother,
	}

	// Root Context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	chk(catnip.Run(&catnipCfg, ctx), "failed to run catnip")
}

func setupFunc(isDisplay bool, cfg *config, display *graphic.Display) func() error {
	if !isDisplay {
		return func() error { return nil }
	}

	return func() error {
		if err := display.Init(cfg.sampleRate, cfg.sampleSize); err != nil {
			return err
		}

		display.SetSizes(cfg.barSize, cfg.spaceSize)
		display.SetBase(cfg.baseSize)
		display.SetDrawType(graphic.DrawType(cfg.drawType))
		display.SetStyles(cfg.styles)
		display.SetInvertDraw(cfg.invertDraw)

		return nil
	}
}

func startFunc(isDisplay bool, display *graphic.Display) func(context.Context) (context.Context, error) {
	if !isDisplay {
		return func(ctx context.Context) (context.Context, error) {
			return ctx, nil
		}
	}

	return func(ctx context.Context) (context.Context, error) {
		ctx = display.Start(ctx)
		return ctx, nil
	}
}

func cleanupFunc(isDisplay bool, display *graphic.Display) func() error {
	if !isDisplay {
		return func() error { return nil }
	}

	return func() error {
		display.Stop()
		display.Close()
		return nil
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
	parser.Int(&cfg.smoothingMethod, "sm", "smooth-method", "smoothing method (0, 1, 2, 3, 4, 5)")
	parser.Int(&cfg.smoothingAverageWindowSize, "sas", "smooth-average-size", "smoothing window size")
	parser.Int(&cfg.baseSize, "bt", "base", "base thickness [0, +Inf)")
	parser.Int(&cfg.barSize, "bw", "bar", "bar width [1, +Inf)")
	parser.Int(&cfg.spaceSize, "bs", "space", "space width [0, +Inf)")
	parser.Int(&cfg.drawType, "dt", "draw", "draw type (1, 2, 3, 4, 5, 6, 7, 8, 9)")
	parser.Bool(&cfg.dontNormalize, "dn", "dont-normalize", "dont normalize analyzer output")
	parser.Bool(&cfg.useThreaded, "t", "threaded", "use the threaded processor")
	parser.Bool(&cfg.invertDraw, "i", "invert", "invert the direction of bin drawing")

	parser.Bool(&cfg.useRawOutput, "raw", "output-raw", "print raw frequency bins")
	parser.Int(&cfg.rawOutputBins, "rawb", "output-raw-bins", "number of bins per channel for the raw output")
	parser.Bool(&cfg.rawOutputMirror, "rawm", "output-raw-mirror", "mirror the raw output similar to \"graphical\" output")

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
		defaultBackend := input.DefaultBackend()

		fmt.Println("all backends. '*' marks default")

		for _, backend := range input.Backends {
			star := ' '
			if defaultBackend == backend.Name {
				star = '*'
			}

			fmt.Printf("- %s %c\n", backend.Name, star)
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
