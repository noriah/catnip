package main

import (
	"math"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/dsp/window"
	"github.com/noriah/catnip/fft"
	"github.com/noriah/catnip/graphic"
	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/util"
)

type visualizer struct {
	cfg *Config

	slowWindow util.MovingWindow
	fastWindow util.MovingWindow

	fftBuf    []complex128
	inputBufs [][]input.Sample
	barBufs   [][]float64

	plans    []*fft.Plan
	spectrum dsp.Spectrum

	bars    int
	display graphic.Display
}

// Process runs one draw refresh with the visualizer on the termbox screen.
func (vis *visualizer) Process() {
	if n := vis.display.Bars(vis.cfg.ChannelCount); n != vis.bars {
		vis.bars = vis.spectrum.Recalculate(n)
	}

	var peak float64

	for idx := range vis.barBufs {
		window.Lanczos(vis.inputBufs[idx])
		vis.plans[idx].Execute()

		buf := vis.barBufs[idx]

		for bIdx := range buf[:vis.bars] {
			v := vis.spectrum.ProcessBin(idx, bIdx, vis.fftBuf)

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

	vis.display.Draw(vis.barBufs, vis.cfg.ChannelCount, vis.bars, scale)
}
