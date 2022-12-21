package processor

import (
	"context"
	"sync"
	"time"

	"github.com/noriah/catnip/dsp/window"
	"github.com/noriah/catnip/fft"
	"github.com/noriah/catnip/input"
)

type Analyzer interface {
	BinCount() int
	ProcessBin(int, []complex128) float64
	Recalculate(int) int
}

type Smoother interface {
	SmoothBin(int, int, float64) float64
}

type Output interface {
	Bins(...int) int
	Write([][]float64, int, float64) error
}

type Processor interface {
	Start(ctx context.Context) context.Context
	Stop()
	Process(context.Context, chan bool, *sync.Mutex)
}

type Config struct {
	SampleRate   float64          // rate at which samples are read
	SampleSize   int              // number of samples per buffer
	ChannelCount int              // number of channels
	FrameRate    int              // target framerate
	Buffers      [][]input.Sample // sample buffers
	Analyzer     Analyzer         // audio analyzer
	Smoother     Smoother         // time smoother
	Output       Output           // data output
}

type processor struct {
	channelCount int
	frameRate    int

	bars int

	fftBufs [][]complex128
	barBufs [][]float64

	// Double-buffer the audio samples so we can read on it again while the code
	// is processing it.
	inputBufs [][]input.Sample

	plans []*fft.Plan

	anlz Analyzer
	smth Smoother
	out  Output
}

func New(cfg Config) *processor {

	vis := &processor{
		channelCount: cfg.ChannelCount,
		frameRate:    cfg.FrameRate,
		fftBufs:      make([][]complex128, cfg.ChannelCount),
		barBufs:      make([][]float64, cfg.ChannelCount),
		inputBufs:    cfg.Buffers,
		plans:        make([]*fft.Plan, cfg.ChannelCount),
		anlz:         cfg.Analyzer,
		smth:         cfg.Smoother,
		out:          cfg.Output,
	}

	for idx := range vis.barBufs {
		vis.barBufs[idx] = make([]float64, cfg.SampleSize)
		vis.fftBufs[idx] = make([]complex128, cfg.SampleSize/2+1)

		fft.InitPlan(&vis.plans[idx], vis.inputBufs[idx], vis.fftBufs[idx])
	}

	return vis
}

func (vis *processor) Start(ctx context.Context) context.Context {

	return ctx
}

func (vis *processor) Stop() {}

// Process runs processing on sample sets and calls Write on the output once per sample set.
func (vis *processor) Process(ctx context.Context, kickChan chan bool, mu *sync.Mutex) {

	if vis.frameRate <= 0 {
		// if we do not have a framerate set, allow at most 1 second per sampling
		vis.frameRate = 1
	}

	dur := time.Second / time.Duration(vis.frameRate)
	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	for {
		mu.Lock()
		for idx := range vis.barBufs {
			window.Lanczos(vis.inputBufs[idx])
			vis.plans[idx].Execute()
		}
		mu.Unlock()

		if n := vis.out.Bins(vis.channelCount); n != vis.bars {
			vis.bars = vis.anlz.Recalculate(n)
		}

		peak := 0.0
		for idx := range vis.fftBufs {

			buf := vis.barBufs[idx]

			for bIdx := range buf[:vis.bars] {
				v := vis.anlz.ProcessBin(bIdx, vis.fftBufs[idx])
				if vis.smth != nil {
					v = vis.smth.SmoothBin(idx, bIdx, v)
				}

				if peak < v {
					peak = v
				}

				buf[bIdx] = v
			}
		}

		vis.out.Write(vis.barBufs, vis.channelCount, peak)

		select {
		case <-ctx.Done():
			return
		case <-kickChan:
		case <-ticker.C:
			// default:
		}
		ticker.Reset(dur)
	}
}
