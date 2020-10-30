package main

import (
	"fmt"
	"log"

	"github.com/noriah/catnip/input"
	"github.com/pkg/errors"

	_ "github.com/noriah/catnip/input/ffmpeg"
	_ "github.com/noriah/catnip/input/parec"

	"github.com/integrii/flaggy"
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

	var parser = flaggy.NewParser(AppName)
	parser.Description = AppDesc
	parser.AdditionalHelpPrepend = AppSite
	parser.Version = version

	var listBackendsCmd = &flaggy.Subcommand{
		Name:                 "list-backends",
		ShortName:            "lb",
		Description:          "list all supported backends",
		AdditionalHelpAppend: "\nuse the full name after the '-'",
	}

	parser.AttachSubcommand(listBackendsCmd, 1)

	var listDevicesCmd = &flaggy.Subcommand{
		Name:                 "list-devices",
		ShortName:            "ld",
		Description:          "list all devices for a backend",
		AdditionalHelpAppend: "\nuse the full name after the '-'",
	}

	parser.AttachSubcommand(listDevicesCmd, 1)

	var cfg = NewZeroConfig()

	parser.String(&cfg.Backend, "b", "backend", "backend name")
	parser.String(&cfg.Device, "d", "device", "device name")
	parser.Float64(&cfg.SampleRate, "r", "rate", "sample rate")
	parser.Int(&cfg.SampleSize, "n", "samples", "sample size")
	parser.Int(&cfg.ChannelCount, "ch", "channels", "channel count (1 or 2)")
	parser.Float64(&cfg.SmoothFactor, "sf", "smoothing", "smooth factor (0-100)")
	parser.Float64(&cfg.WinVar, "wv", "win", "a0 applied to the window function")
	parser.Int(&cfg.BaseThick, "bt", "base", "base thickness [0, +Inf)")
	parser.Int(&cfg.BarWidth, "bw", "bar", "bar width [1, +Inf)")
	parser.Int(&cfg.SpaceWidth, "sw", "space", "space width [0, +Inf)")
	parser.Int(&cfg.DrawType, "dt", "draw", "draw type (1, 2, 3)")
	parser.Int(&cfg.SpectrumType, "st", "distribute",
		"spectrum distribution type (here be dragons)")

	chk(parser.Parse())

	chk(sanitizeConfig(cfg))

	if listBackendsCmd.Used {
		for _, backend := range input.Backends {
			fmt.Printf("- %s\n", backend.Name)
		}
		return
	}

	if listDevicesCmd.Used {
		var backend, err = initBackend(cfg)
		chk(err)

		devices, err := backend.Devices()
		if err != nil {
			log.Fatalln("error", errors.Wrap(err, "failed to get devices"))
		}

		var defaultDevice, _ = backend.DefaultDevice()

		fmt.Printf("all devices for %q backend. '*' marks default\n", cfg.Backend)

		for idx := range devices {
			var star = 0x20
			if defaultDevice != nil && devices[idx].String() == defaultDevice.String() {
				star = 0x2a
			}

			fmt.Printf("- %v %c\n", devices[idx], rune(star))
		}

		return
	}

	chk(Catnip(cfg))
}

func chk(err error) {
	if err != nil {
		log.Fatalln("error", err)
	}
}
