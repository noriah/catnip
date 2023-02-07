package catnip

import (
	"context"
	"sync"

	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/processor"

	"github.com/pkg/errors"
)

const MaxChannelCount = 2
const MaxSampleSize = 2048

type SetupFunc func() error
type StartFunc func(ctx context.Context) (context.Context, error)
type CleanupFunc func() error

func Run(cfg *Config, ctx context.Context) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	inputBuffers := input.MakeBuffers(cfg.ChannelCount, cfg.SampleSize)

	procConfig := processor.Config{
		SampleRate:   cfg.SampleRate,
		SampleSize:   cfg.SampleSize,
		ChannelCount: cfg.ChannelCount,
		ProcessRate:  cfg.ProcessRate,
		Buffers:      inputBuffers,
		Analyzer:     cfg.Analyzer,
		Output:       cfg.Output,
		Smoother:     cfg.Smoother,
		Windower:     cfg.Windower,
	}

	var vis processor.Processor

	if cfg.UseThreaded {
		vis = processor.NewThreaded(procConfig)
	} else {
		vis = processor.New(procConfig)
	}

	backend, err := input.InitBackend(cfg.Backend)
	if err != nil {
		return err
	}

	sessConfig := input.SessionConfig{
		FrameSize:  cfg.ChannelCount,
		SampleSize: cfg.SampleSize,
		SampleRate: cfg.SampleRate,
	}

	if sessConfig.Device, err = input.GetDevice(backend, cfg.Device); err != nil {
		return err
	}

	audio, err := backend.Start(sessConfig)
	defer backend.Close()

	if err != nil {
		return errors.Wrap(err, "failed to start the input backend")
	}

	if cfg.SetupFunc != nil {
		if err := cfg.SetupFunc(); err != nil {
			return err
		}
	}

	if cfg.CleanupFunc != nil {
		defer cfg.CleanupFunc()
	}

	if cfg.StartFunc != nil {
		if ctx, err = cfg.StartFunc(ctx); err != nil {
			return err
		}
	}

	kickChan := make(chan bool, 1)
	mu := &sync.Mutex{}

	ctx = vis.Start(ctx, kickChan, mu)
	defer vis.Stop()

	if err := audio.Start(ctx, inputBuffers, kickChan, mu); err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			return errors.Wrap(err, "failed to start input session")
		}
	}

	return nil
}
