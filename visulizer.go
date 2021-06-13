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

	channels int
	bars     int
	display  graphic.Display
}

// Process runs one draw refresh with the visualizer on the termbox screen.
func (vis *visualizer) Process() {
	if n := vis.display.Bars(vis.cfg.ChannelCount); n != vis.bars {
		vis.bars = vis.spectrum.Recalculate(n)
	}

	var peak float64

	var mScale = 1.0
	vMean, vSD := vis.slowWindow.Stats()
	if t := vMean + (1.5 * vSD); t > 0.0 {
		mScale = t
	}

	for idx := range vis.barBufs {
		window.CosSum(vis.inputBufs[idx], vis.cfg.WinVar)
		vis.plans[idx].Execute()

		buf := vis.barBufs[idx]

		for bIdx := range buf[:vis.bars] {
			v := vis.spectrum.ProcessBin(bIdx, mScale, buf[bIdx], vis.fftBuf)
			if peak < v {
				peak = v
			}
			buf[bIdx] = v
		}
	}

	// Don't draw if the peak is too small to even draw.
	if peak < PeakThreshold {
		return
	}

	var scale = 1.0

	// do some scaling if we are above 0
	if peak > 0.0 {
		vis.fastWindow.Update(peak)
		vMean, vSD := vis.slowWindow.Update(peak)

		if length := vis.slowWindow.Len(); length >= vis.fastWindow.Cap() {
			if math.Abs(vis.fastWindow.Mean()-vMean) > (ScalingResetDeviation * vSD) {
				vMean, vSD = vis.slowWindow.Drop(int(float64(length) * ScalingDumpPercent))
			}
		}

		if t := vMean + (1.5 * vSD); t > 1.0 {
			scale = t
		}
	}

	vis.display.Draw(vis.barBufs, vis.cfg.ChannelCount, vis.bars, scale)
}
