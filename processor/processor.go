package processor

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/noriah/catnip/dsp/window"
	"github.com/noriah/catnip/fft"
	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/util"
)

const (
	// ScalingSlowWindow in seconds
	ScalingSlowWindow = 5
	// ScalingFastWindow in seconds
	ScalingFastWindow = ScalingSlowWindow * 0.2
	// ScalingDumpPercent is how much we erase on rescale
	ScalingDumpPercent = 0.60
	// ScalingResetDeviation standard deviations from the mean before reset
	ScalingResetDeviation = 1.0
	// PeakThreshold is the threshold to not draw if the peak is less.
	PeakThreshold = 0.01
)

type Analyzer interface {
	BinCount() int
	ProcessBin(int, int, []complex128) float64
	Recalculate(int) int
}

type Smoother interface {
	SmoothBin(int, int, float64) float64
}

type Display interface {
	Bars(...int) int
	Draw([][]float64, int, int, float64, bool) error
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
	InvertDraw   bool             // invert the direction of bin drawing
	Buffers      [][]input.Sample // sample buffers
	Analyzer     Analyzer         // audio analyzer
	Smoother     Smoother         // time smoother
	Display      Display          // display
}

type processor struct {
	channelCount int
	frameRate    int

	bars int

	invertDraw bool

	slowWindow *util.MovingWindow
	fastWindow *util.MovingWindow

	fftBufs [][]complex128
	barBufs [][]float64

	// Double-buffer the audio samples so we can read on it again while the code
	// is processing it.
	inputBufs [][]input.Sample

	plans []*fft.Plan

	anlz Analyzer
	smth Smoother
	disp Display
}

func New(cfg Config) *processor {
	slowSize := ((int(ScalingSlowWindow * cfg.SampleRate)) / cfg.SampleSize) * 2

	fastSize := ((int(ScalingFastWindow * cfg.SampleRate)) / cfg.SampleSize) * 2

	vis := &processor{
		channelCount: cfg.ChannelCount,
		frameRate:    cfg.FrameRate,
		invertDraw:   cfg.InvertDraw,
		slowWindow:   util.NewMovingWindow(slowSize),
		fastWindow:   util.NewMovingWindow(fastSize),
		fftBufs:      make([][]complex128, cfg.ChannelCount),
		barBufs:      make([][]float64, cfg.ChannelCount),
		inputBufs:    cfg.Buffers,
		plans:        make([]*fft.Plan, cfg.ChannelCount),
		anlz:         cfg.Analyzer,
		smth:         cfg.Smoother,
		disp:         cfg.Display,
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

// Process runs one draw refresh with the processor on the termbox screen.
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

		if n := vis.disp.Bars(vis.channelCount); n != vis.bars {
			vis.bars = vis.anlz.Recalculate(n)
		}

		var peak float64
		for idx := range vis.fftBufs {

			buf := vis.barBufs[idx]

			for bIdx := range buf[:vis.bars] {
				v := vis.anlz.ProcessBin(idx, bIdx, vis.fftBufs[idx])
				v = vis.smth.SmoothBin(idx, bIdx, v)

				if peak < v {
					peak = v
				}

				buf[bIdx] = v
			}
		}

		scale := 1.0

		// do some scaling if we are above the PeakThreshold
		if peak >= PeakThreshold {
			vis.fastWindow.Update(peak)
			vMean, vSD := vis.slowWindow.Update(peak)

			// if our slow window finally has more values than our fast window
			if length := vis.slowWindow.Len(); length >= vis.fastWindow.Cap() {
				// no idea what this is doing
				if math.Abs(vis.fastWindow.Mean()-vMean) > (ScalingResetDeviation * vSD) {
					// drop some values and continue
					vMean, vSD = vis.slowWindow.Drop(int(float64(length) * ScalingDumpPercent))
				}
			}

			if t := vMean + (1.5 * vSD); t > 1.0 {
				scale = t
			}
		}

		vis.disp.Draw(vis.barBufs, vis.channelCount, vis.bars, scale, vis.invertDraw)

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
