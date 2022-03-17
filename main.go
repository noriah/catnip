package main

import (
	"fmt"
	"log"

	"github.com/noriah/catnip/graphic"
	"github.com/noriah/catnip/input"

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

	var cfg = NewZeroConfig()

	if doFlags(&cfg) {
		return
	}

	chk(cfg.Sanitize(), "invalid config")

	chk(Catnip(&cfg), "failed to run catnip")
}

func doFlags(cfg *Config) bool {

	var parser = flaggy.NewParser(AppName)
	parser.Description = AppDesc
	parser.AdditionalHelpPrepend = AppSite
	parser.Version = version

	var listBackendsCmd = flaggy.Subcommand{
		Name:                 "list-backends",
		ShortName:            "lb",
		Description:          "list all supported backends",
		AdditionalHelpAppend: "\nuse the full name after the '-'",
	}

	parser.AttachSubcommand(&listBackendsCmd, 1)

	var listDevicesCmd = flaggy.Subcommand{
		Name:                 "list-devices",
		ShortName:            "ld",
		Description:          "list all devices for a backend",
		AdditionalHelpAppend: "\nuse the full name after the '-'",
	}

	parser.AttachSubcommand(&listDevicesCmd, 1)

	parser.String(&cfg.Backend, "b", "backend", "backend name")
	parser.String(&cfg.Device, "d", "device", "device name")
	parser.Float64(&cfg.SampleRate, "r", "rate", "sample rate")
	parser.Int(&cfg.SampleSize, "n", "samples", "sample size")
	parser.Int(&cfg.FrameRate, "f", "fps", "frame rate (0 to draw on every sample)")
	parser.Int(&cfg.ChannelCount, "ch", "channels", "channel count (1 or 2)")
	parser.Float64(&cfg.SmoothFactor, "sf", "smoothing", "smooth factor (0-100)")
	parser.Float64(&cfg.WinVar, "wv", "win", "a0 applied to the window function")
	parser.Int(&cfg.BaseSize, "bt", "base", "base thickness [0, +Inf)")
	parser.Int(&cfg.BarSize, "bw", "bar", "bar width [1, +Inf)")
	parser.Int(&cfg.SpaceSize, "sw", "space", "space width [0, +Inf)")
	parser.Int(&cfg.DrawType, "dt", "draw", "draw type (1, 2, 3)")

	fg, bg, center := graphic.DefaultStyles().AsUInt16s()
	parser.UInt16(&fg, "fg", "foreground",
		"foreground color within the 256-color range [0, 255] with attributes")
	parser.UInt16(&bg, "bg", "background",
		"background color within the 256-color range [0, 255] with attributes")
	parser.UInt16(&center, "ct", "center",
		"center line color within the 256-color range [0, 255] with attributes")

	chk(parser.Parse(), "failed to parse arguments")

	// Manually set the styles.
	cfg.Styles = graphic.StylesFromUInt16(fg, bg, center)

	switch {
	case listBackendsCmd.Used:
		for _, backend := range input.Backends {
			fmt.Printf("- %s\n", backend.Name)
		}

		return true

	case listDevicesCmd.Used:
		backend, err := initBackend(cfg)
		chk(err, "failed to init backend")

		devices, err := backend.Devices()
		chk(err, "failed to get devices")

		// We don't really need the default device to be indicated.
		defaultDevice, _ := backend.DefaultDevice()

		fmt.Printf("all devices for %q backend. '*' marks default\n", cfg.Backend)

		for idx := range devices {
			var star = ' '
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
