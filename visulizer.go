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

	for idx, buf := range vis.barBufs {
		window.CosSum(vis.inputBufs[idx], vis.cfg.WinVar)
		vis.plans[idx].Execute()
		vis.spectrum.Process(buf, vis.fftBuf)

		for _, v := range buf[:vis.bars] {
			if peak < v {
				peak = v
			}
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
