package visualizer

import (
	"math"
	"sync"

	"github.com/noriah/catnip/config"
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

type spectrum interface {
	BinCount() int
	ProcessBin(int, int, []complex128) float64
	Recalculate(int) int
}

type graphicDisplay interface {
	Bars(...int) int
	Draw([][]float64, int, int, float64) error
}

type Visualizer struct {
	channelCount int
	frameRate    int

	bars int

	inputMut sync.Mutex

	slowWindow *util.MovingWindow
	fastWindow *util.MovingWindow

	fftBuf  []complex128
	barBufs [][]float64

	// Double-buffer the audio samples so we can read on it again while the code
	// is processing it.
	inputBufs   [][]input.Sample
	scratchBufs [][]input.Sample

	plans []*fft.Plan

	Spectrum spectrum
	Display  graphicDisplay
}

func New(cfg *config.Config, inputBuffers [][]input.Sample) *Visualizer {
	slowSize := ((int(ScalingSlowWindow * cfg.SampleRate)) / cfg.SampleSize) * 2

	fastSize := ((int(ScalingFastWindow * cfg.SampleRate)) / cfg.SampleSize) * 2

	vis := &Visualizer{
		channelCount: cfg.ChannelCount,
		frameRate:    cfg.FrameRate,
		slowWindow:   util.NewMovingWindow(slowSize),
		fastWindow:   util.NewMovingWindow(fastSize),
		fftBuf:       make([]complex128, cfg.SampleSize/2+1),
		barBufs:      make([][]float64, cfg.ChannelCount),
		inputBufs:    inputBuffers,
		scratchBufs:  input.MakeBuffers(cfg.ChannelCount, cfg.SampleSize),
		plans:        make([]*fft.Plan, cfg.ChannelCount),
	}

	for idx := range vis.barBufs {
		vis.barBufs[idx] = make([]float64, cfg.SampleSize)

		fft.InitPlan(&vis.plans[idx], vis.inputBufs[idx], vis.fftBuf)
	}

	return vis
}

// Process runs one draw refresh with the visualizer on the termbox screen.
func (vis *Visualizer) Process() {
	vis.inputMut.Lock()
	defer vis.inputMut.Unlock()

	input.CopyBuffers(vis.scratchBufs, vis.inputBufs)
	if vis.frameRate <= 0 {
		vis.Draw(false)
	}
}

func (vis *Visualizer) Draw(lock bool) {
	if lock {
		vis.inputMut.Lock()
		defer vis.inputMut.Unlock()
	}

	if n := vis.Display.Bars(vis.channelCount); n != vis.bars {
		vis.bars = vis.Spectrum.Recalculate(n)
	}

	var peak float64

	for idx := range vis.barBufs {
		window.Lanczos(vis.inputBufs[idx])
		vis.plans[idx].Execute()

		buf := vis.barBufs[idx]

		for bIdx := range buf[:vis.bars] {
			v := vis.Spectrum.ProcessBin(idx, bIdx, vis.fftBuf)

			if peak < v {
				peak = v
			}

			buf[bIdx] = v
		}
	}

	var scale = 1.0

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

	vis.Display.Draw(vis.barBufs, vis.channelCount, vis.bars, scale)
}
