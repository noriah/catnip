package dsp

import (
	"math"
)

// Scaling Constants
const (

	// ScalingFastWindow in seconds
	ScalingSlowWindow = 10

	// ScalingFastWindow in seconds
	ScalingFastWindow = ScalingSlowWindow * 0.2

	// ScalingDumpPercent is how much we erase on rescale
	ScalingDumpPercent = 0.75

	ScalingResetDeviation = 1
)

// Scale scales the data
func Scale(height int, ds *DataSet) {

	var peak = 0.125

	var vSilent = true

	for xBin := 0; xBin < ds.numBins; xBin++ {

		if ds.binBuf[xBin] > 0 {

			vSilent = false

			if peak < ds.binBuf[xBin] {
				peak = ds.binBuf[xBin]
			}
		}
	}

	if !vSilent {
		ds.fastWindow.Update(peak)
		ds.slowWindow.Update(peak)
	}

	var vMean, vSD = ds.slowWindow.Stats()

	if length := ds.slowWindow.Len(); length >= ds.fastWindow.Cap() {

		if math.Abs(ds.fastWindow.Mean()-vMean) > (ScalingResetDeviation * vSD) {

			ds.slowWindow.Drop(int(float64(length) * ScalingDumpPercent))
			vMean, vSD = ds.slowWindow.Stats()
		}
	}

	var vMag = math.Max(vMean+(2*vSD), 1)

	for xBin, cHeight := 0, float64(height-1); xBin < ds.numBins; xBin++ {
		ds.binBuf[xBin] = math.Min(cHeight, (ds.binBuf[xBin]/vMag)*cHeight)
	}
}
