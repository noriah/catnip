package main

import (
	"fmt"
	"log"
	"os"

	"github.com/noriah/tavis"
	"github.com/noriah/tavis/portaudio"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func init() {
	log.SetFlags(0)
}

var device = tavis.NewZeroDevice()

func main() {
	app := cli.App{
		Name:   "tavis",
		Usage:  "terminal audio visualizer",
		Action: run,
		Commands: []*cli.Command{
			{
				Name:        "list-devices",
				Action:      listDevices,
				Description: "List all visible input devices",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "device-name",
				Aliases:     []string{"d"},
				Value:       device.Name,
				Destination: &device.Name,
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
			&cli.Float64Flag{
				Name:        "monstercat-factor",
				Aliases:     []string{"mf"},
				Value:       device.MonstercatFactor,
				Destination: &device.MonstercatFactor,
			},
			&cli.Float64Flag{
				Name:        "falloff-weight",
				Aliases:     []string{"fw"},
				Value:       device.FalloffWeight,
				Destination: &device.FalloffWeight,
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
		log.Fatalln(err)
	}
}

func run(c *cli.Context) error {
	return tavis.Run(device)
}

func listDevices(c *cli.Context) error {
	if err := portaudio.Initialize(); err != nil {
		return errors.Wrap(err, "failed to initialize Portaudio")
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return errors.Wrap(err, "failed to get devices")
	}

	type host struct {
		name    string
		devices []*portaudio.DeviceInfo
	}

	var hosts = []host{}

DeviceLoop:
	for _, device := range devices {
		for i, host := range hosts {
			if host.name == device.HostApi.Name {
				host.devices = append(host.devices, device)
				hosts[i] = host

				continue DeviceLoop
			}
		}

		hosts = append(hosts, host{
			name:    device.HostApi.Name,
			devices: []*portaudio.DeviceInfo{device},
		})
	}

	for _, host := range hosts {
		fmt.Println("Host:", host.name)

		for _, device := range host.devices {
			fmt.Printf("  - %s\n", device.Name)
		}
	}

	return nil
}
