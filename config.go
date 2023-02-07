package catnip

import (
	"errors"
	"fmt"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/dsp/window"
	"github.com/noriah/catnip/processor"
)

type Config struct {
	// The name of the backend from the input package
	Backend string
	// The name of the device to pull data from
	Device string
	// The rate that samples are read
	SampleRate float64
	// The number of samples per batch
	SampleSize int
	// The number of channels to read data from
	ChannelCount int
	// The number of times per second to process data
	ProcessRate int
	// Merge multiple channels into a single stream
	Combine bool

	// testing. leave false
	// Use threaded processor
	UseThreaded bool

	// Function to call when setting up the pipeline
	SetupFunc SetupFunc
	// Function to call when starting the pipeline
	StartFunc StartFunc
	// Function to call when cleaning up the pipeline
	CleanupFunc CleanupFunc
	// Where to send the data from the audio analysis
	Output processor.Output
	// Method to run on data before running fft
	Windower window.Function
	// Analyzer to run analysis on data
	Analyzer dsp.Analyzer
	// Smoother to run smoothing on output from Analyzer
	Smoother dsp.Smoother
}

func NewZeroConfig() Config {
	return Config{
		SampleRate:   44100,
		SampleSize:   1024,
		ChannelCount: 1,
	}
}

func (cfg *Config) Validate() error {
	if cfg.SampleRate < float64(cfg.SampleSize) {
		return errors.New("sample rate lower than sample size")
	}

	if cfg.SampleSize < 4 {
		return errors.New("sample size too small (4+ required)")
	}

	switch {
	case cfg.ChannelCount > MaxChannelCount:
		return fmt.Errorf("too many channels (%d max)", MaxChannelCount)

	case cfg.ChannelCount < 1:
		return errors.New("too few channels (1 min)")

	case cfg.SampleSize > MaxSampleSize:
		return fmt.Errorf("sample size too large (%d max)", MaxSampleSize)
	}

	return nil
}
