package main

import (
	"context"
	"fmt"

	"github.com/noriah/catnip/dsp"
	"github.com/noriah/catnip/util"
)

// Constants
const (
	// ScalingWindow in seconds
	ScalingWindow = 1.5
	// PeakThreshold is the threshold to not draw if the peak is less.
	PeakThreshold = 0.001
)

// Writer handles drawing our visualizer.
type Writer struct {
	Smoother   dsp.Smoother
	trackZero  int
	binCount   int
	invertDraw bool
	window     *util.MovingWindow
}

func NewWriter() *Writer {
	return &Writer{
		binCount: 50,
	}
}

// Init initializes the display.
// Should be called before any other display method.
func (d *Writer) Init(sampleRate float64, sampleSize int) error {
	// make a large buffer as this could be as big as the screen width/height.

	windowSize := ((int(ScalingWindow * sampleRate)) / sampleSize) * 2
	d.window = util.NewMovingWindow(windowSize)

	return nil
}

// Close will stop display and clean up the terminal.
func (d *Writer) Close() error {
	return nil
}

func (d *Writer) SetInvertDraw(invert bool) {
	d.invertDraw = invert
}

// Start display is bad.
func (d *Writer) Start(ctx context.Context) context.Context {
	return ctx
}

// Stop display not work.
func (d *Writer) Stop() error {
	return nil
}

// Draw takes data and draws.
func (d *Writer) Write(buffers [][]float64, channels int) error {

	peak := 0.0
	bins := d.Bins(channels)

	for i := 0; i < channels; i++ {
		for _, val := range buffers[i][:bins] {
			if val > peak {
				peak = val
			}
		}
	}

	scale := 1.0

	if peak >= PeakThreshold {
		d.trackZero = 0

		// do some scaling if we are above the PeakThreshold
		d.window.Update(peak)

	} else {
		if d.trackZero++; d.trackZero == 5 {
			d.window.Recalculate()
		}
	}

	vMean, vSD := d.window.Stats()

	if t := vMean + (2.0 * vSD); t > 1.0 {
		scale = t
	}

	scale = 100.0 / scale

	for xSet, chBins := range buffers {

		for xBar := 0; xBar < d.binCount; xBar++ {

			xBin := (xBar * (1 - xSet)) + (((d.binCount - 1) - xBar) * xSet)

			if d.invertDraw {
				xBin = d.binCount - 1 - xBin
			}

			fmt.Printf("%4.2f ", chBins[xBin]*scale)
		}
	}

	fmt.Println()

	return nil
}

// Bins returns the number of bars we will draw.
func (d *Writer) Bins(chCount int) int {
	return d.binCount
}
