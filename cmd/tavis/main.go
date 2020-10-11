package main

import (
	"fmt"
	"log"
	"os"

	"github.com/noriah/tavis"
	"github.com/noriah/tavis/input"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	// Input backends.
	_ "github.com/noriah/tavis/input/ffmpeg"
	_ "github.com/noriah/tavis/input/parec"
)

var (
	device = tavis.NewZeroDevice()
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
				Aliases:     []string{"s"},
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
				Aliases:     []string{"fps"},
				Value:       device.TargetFPS,
				Destination: &device.TargetFPS,
			},
			&cli.IntFlag{
				Name:        "channel-count",
				Aliases:     []string{"ch"},
				Hidden:      true,
				Value:       device.ChannelCount,
				Destination: &device.ChannelCount,
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

	return tavis.Run(device)
}
