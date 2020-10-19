package main

import (
	"fmt"
	"log"
	"os"

	"github.com/noriah/tavis/input"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	// Input backends.
	_ "github.com/noriah/tavis/input/ffmpeg"
	_ "github.com/noriah/tavis/input/parec"
)

func init() {
	log.SetFlags(0)
}

var globalCfg = NewZeroConfig()

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
				Value:       globalCfg.SampleRate,
				Destination: &globalCfg.SampleRate,
			},
			&cli.Float64Flag{
				Name:        "low-cut-freq",
				Aliases:     []string{"lf"},
				Value:       globalCfg.LoCutFreq,
				Destination: &globalCfg.LoCutFreq,
			},
			&cli.Float64Flag{
				Name:        "high-cut-freq",
				Aliases:     []string{"hf"},
				Value:       globalCfg.HiCutFreq,
				Destination: &globalCfg.HiCutFreq,
			},
			&cli.Float64Flag{
				Name:        "smoothness-factor",
				Aliases:     []string{"sf"},
				Value:       globalCfg.SmoothFactor,
				Destination: &globalCfg.SmoothFactor,
			},
			&cli.Float64Flag{
				Name:        "smoothness-response",
				Aliases:     []string{"sr"},
				Value:       globalCfg.SmoothResponse,
				Destination: &globalCfg.SmoothResponse,
			},
			&cli.IntFlag{
				Name:        "base-thickness",
				Aliases:     []string{"bt"},
				Value:       globalCfg.BaseThick,
				Destination: &globalCfg.BaseThick,
			},
			&cli.IntFlag{
				Name:        "bar-width",
				Aliases:     []string{"bw"},
				Value:       globalCfg.BarWidth,
				Destination: &globalCfg.BarWidth,
			},
			&cli.IntFlag{
				Name:        "space-width",
				Aliases:     []string{"sw"},
				Value:       globalCfg.SpaceWidth,
				Destination: &globalCfg.SpaceWidth,
			},
			&cli.IntFlag{
				Name:        "target-fps",
				Aliases:     []string{"f"},
				Value:       globalCfg.TargetFPS,
				Destination: &globalCfg.TargetFPS,
			},
			&cli.IntFlag{
				Name:        "channel-count",
				Aliases:     []string{"c"},
				Hidden:      true,
				Value:       globalCfg.ChannelCount,
				Destination: &globalCfg.ChannelCount,
			},
			&cli.IntFlag{
				Name:        "max-bins",
				Aliases:     []string{"mb"},
				Hidden:      true,
				Value:       globalCfg.MaxBins,
				Destination: &globalCfg.MaxBins,
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

	globalCfg.InputBackend = input.FindBackend(backendName)
	if globalCfg.InputBackend == nil {
		return fmt.Errorf("backend not found: %q", backendName)
	}

	if err := globalCfg.InputBackend.Init(); err != nil {
		return errors.Wrap(err, "failed to initialize input backend")
	}

	return nil
}

func listDevices(c *cli.Context) error {
	if err := initBackend(c); err != nil {
		return err
	}

	devices, err := globalCfg.InputBackend.Devices()
	if err != nil {
		return errors.Wrap(err, "failed to get devices")
	}

	// optional default device
	defaultDevice, _ := globalCfg.InputBackend.DefaultDevice()

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
		def, err := globalCfg.InputBackend.DefaultDevice()
		if err != nil {
			return errors.Wrap(err, "failed to get default device")
		}

		globalCfg.InputDevice = def
		return nil
	}

	devices, err := globalCfg.InputBackend.Devices()
	if err != nil {
		return errors.Wrap(err, "failed to get devices")
	}

	for _, d := range devices {
		if d.String() == deviceName {
			globalCfg.InputDevice = d
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

	return Run(globalCfg)
}