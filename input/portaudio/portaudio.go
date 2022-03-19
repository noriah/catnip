package portaudio

import (
	"context"
	"fmt"
	"sync"

	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/input/portaudio/portaudio"
	"github.com/pkg/errors"
)

var GlobalBackend = &Backend{}

func init() {
	input.RegisterBackend("portaudio", GlobalBackend)
}

// errors
var (
	ErrBadDevice    error = errors.New("device not found")
	ErrReadTimedOut error = errors.New("read timed out")
)

// Backend represents the Portaudio backend. A zero-value instance is a
// valid instance.
type Backend struct {
	devices []*portaudio.DeviceInfo
}

func (b *Backend) Init() error {
	return portaudio.Initialize()
}

func (b *Backend) Close() error {
	return portaudio.Terminate()
}

func (b *Backend) Devices() ([]input.Device, error) {
	if b.devices == nil {
		devices, err := portaudio.Devices()
		if err != nil {
			return nil, err
		}
		b.devices = devices
	}

	var gDevices = make([]input.Device, len(b.devices))
	for i, device := range b.devices {
		gDevices[i] = Device{device}
	}

	return gDevices, nil
}

func (b *Backend) DefaultDevice() (input.Device, error) {
	defaultHost, err := portaudio.DefaultHostApi()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get default host API")
	}

	if defaultHost.DefaultInputDevice == nil {
		return nil, errors.New("no default input device found")
	}

	return Device{defaultHost.DefaultInputDevice}, nil
}

func (b *Backend) Start(cfg input.SessionConfig) (input.Session, error) {
	return NewSession(cfg)
}

// Device represents a Portaudio device.
type Device struct {
	*portaudio.DeviceInfo
}

func (d *Device) discard() { d.DeviceInfo = nil }

// String returns the device name.
func (d Device) String() string {
	return d.Name
}

// SampleType is broken out because portaudio supports different types
type SampleType = float32

// Session is an input source that pulls from Portaudio.
type Session struct {
	device Device
	config input.SessionConfig
}

// NewSession creates and initializes a new Portaudio session.
func NewSession(config input.SessionConfig) (*Session, error) {
	dv, ok := config.Device.(Device)
	if !ok {
		return nil, fmt.Errorf("device is on unknown type %T", config.Device)
	}

	// Free up the device inside the config.
	config.Device = nil

	return &Session{
		dv,
		config,
	}, nil
}

func (s *Session) Start(ctx context.Context, dst [][]input.Sample, kickChan chan bool, mu *sync.Mutex) error {
	if !input.EnsureBufferLen(s.config, dst) {
		return errors.New("invalid dst length given")
	}

	param := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   s.device.DeviceInfo,
			Latency:  s.device.DefaultLowInputLatency,
			Channels: s.config.FrameSize,
		},
		SampleRate:      s.config.SampleRate,
		FramesPerBuffer: s.config.SampleSize,
		Flags:           portaudio.ClipOff | portaudio.DitherOff,
	}

	frameSize := s.config.FrameSize
	samples := s.config.SampleSize * frameSize

	// Source buffer in a different format than what we want (dst).
	src := make([]SampleType, samples)

	stream, err := portaudio.OpenStream(param, src)
	if err != nil {
		return errors.Wrap(err, "failed to open stream")
	}
	s.device.discard()
	defer stream.Close()

	if err := stream.Start(); err != nil {
		return errors.Wrap(err, "failed to start stream")
	}
	defer stream.Stop()

	for {

		// Ignore overflow in case the processing is too slow.
		if err := stream.Read(); err != nil && err != portaudio.InputOverflowed {
			return errors.Wrap(err, "failed to read stream")
		}

		mu.Lock()
		for x := 0; x < samples; x++ {
			dst[x%frameSize][x/frameSize] = input.Sample(src[x])
		}
		mu.Unlock()

	loop:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case kickChan <- true:
				break loop
			default:
				fmt.Println("waiting")
			}

			if ready, _ := stream.AvailableToRead(); ready >= samples {
				if ready > samples {
					fmt.Println("OVER", ready)
				}
				break
			}
		}

		// select {
		// case <-ctx.Done():
		// 	return ctx.Err()

		// default:
		// 	fmt.Println("waiting")
		// }

		// select {
		// case <-ctx.Done():
		// 	return ctx.Err()
		// case <-kickChan:
		// }
	}
}
