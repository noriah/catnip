package processor

import (
	"context"
	"sync"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/dsp/window"
	"github.com/noriah/catnip/fft"
	"github.com/noriah/catnip/input"
)

type threadedProcessor struct {
	channelCount int

	bars int

	fftBufs [][]complex128
	barBufs [][]float64

	peaks []float64
	kicks []chan bool

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	// Double-buffer the audio samples so we can read on it again while the code
	// is processing it.
	inputBufs [][]input.Sample

	plans []*fft.Plan

	anlz dsp.Analyzer
	smth dsp.Smoother
	out  Output
}

func NewThreaded(cfg Config) *threadedProcessor {
	vis := &threadedProcessor{
		channelCount: cfg.ChannelCount,
		fftBufs:      make([][]complex128, cfg.ChannelCount),
		barBufs:      make([][]float64, cfg.ChannelCount),
		peaks:        make([]float64, cfg.ChannelCount),
		kicks:        make([]chan bool, cfg.ChannelCount),
		inputBufs:    cfg.Buffers,
		plans:        make([]*fft.Plan, cfg.ChannelCount),
		anlz:         cfg.Analyzer,
		smth:         cfg.Smoother,
		out:          cfg.Output,
	}

	for idx := range vis.barBufs {
		vis.barBufs[idx] = make([]float64, cfg.SampleSize)
		vis.fftBufs[idx] = make([]complex128, cfg.SampleSize/2+1)
		vis.kicks[idx] = make(chan bool, 1)

		fft.InitPlan(&vis.plans[idx], vis.inputBufs[idx], vis.fftBufs[idx])
	}

	return vis
}

func (vis *threadedProcessor) channelProcessor(ch int, kick <-chan bool) {
	buffer := vis.inputBufs[ch]
	plan := vis.plans[ch]
	barBuf := vis.barBufs[ch]
	fftBuf := vis.fftBufs[ch]

	for {
		select {
		case <-vis.ctx.Done():
			return
		case <-kick:
		}

		window.Lanczos(buffer)
		plan.Execute()

		peak := 0.0

		for i := range barBuf[:vis.bars] {
			v := vis.anlz.ProcessBin(i, fftBuf)
			v = vis.smth.SmoothBin(ch, i, v)

			if peak < v {
				peak = v
			}

			barBuf[i] = v
		}

		vis.peaks[ch] = peak

		vis.wg.Done()
	}
}

func (vis *threadedProcessor) Start(ctx context.Context) context.Context {
	vis.ctx, vis.cancel = context.WithCancel(ctx)

	for i, kick := range vis.kicks {
		go vis.channelProcessor(i, kick)
	}

	return vis.ctx
}

func (vis *threadedProcessor) Stop() {
	if vis.cancel != nil {
		vis.cancel()
	}
}

// Process runs one draw refresh with the visualizer on the termbox screen.
func (vis *threadedProcessor) Process(ctx context.Context, kickChan chan bool, mu *sync.Mutex) {
	if n := vis.out.Bins(vis.channelCount); n != vis.bars {
		vis.bars = vis.anlz.Recalculate(n)
	}

	vis.wg.Add(vis.channelCount)

	for _, kick := range vis.kicks {
		kick <- true
	}

	vis.wg.Wait()

	peak := 0.0

	for _, p := range vis.peaks {
		if peak < p {
			peak = p
		}
	}

	vis.out.Write(vis.barBufs, vis.channelCount, peak)
}
