package portaudio

import (
	"context"
	"fmt"
	"log"

	"github.com/noriah/tavis/input"
	"github.com/noriah/tavis/input/portaudio/portaudio"
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

// String returns the device name.
func (d Device) String() string {
	return d.Name
}

// SampleType is broken out because portaudio supports different types
type SampleType = float32

// Session is an input source that pulls from Portaudio.
type Session struct {
	stream    *portaudio.Stream // our input stream
	config    input.SessionConfig
	sampleBuf []SampleType // internal scratch buffer
	retBufs   [][]input.Sample
}

// NewSession creates and initializes a new Portaudio session.
func NewSession(config input.SessionConfig) (*Session, error) {
	dv, ok := config.Device.(Device)
	if !ok {
		return nil, fmt.Errorf("device is on unknown type %T", config.Device)
	}

	param := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   dv.DeviceInfo,
			Latency:  dv.DefaultLowInputLatency,
			Channels: config.FrameSize,
		},
		SampleRate:      config.SampleRate,
		FramesPerBuffer: config.SampleSize,
		// Flags:           portaudio.ClipOff | portaudio.DitherOff,
	}

	buffer := make([]SampleType, config.SampleSize*config.FrameSize)
	retbuf := input.MakeBuffers(config)

	stream, err := portaudio.OpenStream(param, buffer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open stream")
	}

	// Free up the device.
	config.Device = nil

	return &Session{
		stream,
		config,
		buffer,
		retbuf,
	}, nil
}

// Buffers returns a slice to our buffers
func (s *Session) SampleBuffers() [][]input.Sample {
	return s.retBufs
}

// ReadyRead returns the number of frames ready to read
func (s *Session) ReadyRead() int {
	var ready, _ = s.stream.AvailableToRead()
	return ready
}

// Read signals portaudio to dump some data into the buffer we gave it.
// Will block if there is not enough data yet.
func (s *Session) Read(ctx context.Context) error {
	for s.ReadyRead() < s.config.SampleSize {
		select {
		case <-ctx.Done():
			log.Println("read timed out")
			return ErrReadTimedOut
		default:
		}
	}

	err := s.stream.Read()

	for xBuf := range s.retBufs {
		for xSmpl := range s.retBufs[xBuf] {
			s.retBufs[xBuf][xSmpl] = input.Sample(s.sampleBuf[(xSmpl*s.config.FrameSize)+xBuf])
		}
	}

	if err != portaudio.InputOverflowed {
		return err
	}

	return nil
}

// Start opens the stream and starts audio processing.
func (s *Session) Start() error {
	return s.stream.Start()
}

// Stop stops the session.
func (s *Session) Stop() error {
	err := s.stream.Stop()
	s.stream.Close()
	return err
}
