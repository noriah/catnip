package dsp

import "math"

// Scaling Constants
const (

	// ScalingDumpPercent is how much we erase on rescale
	ScalingDumpPercent = 0.75

	ScalingResetDeviation = 1
)

// Scale scales the data
func Scale(height float64, dSet *DataSet) {
	// 9 substeps
	height *= 9

	dSet.peakHeight = 1

	var vSilent = true

	for xBin := 0; xBin <= dSet.numBins; xBin++ {
		if dSet.binBuf[xBin] > 0 {
			vSilent = false
			if dSet.peakHeight < dSet.binBuf[xBin] {
				dSet.peakHeight = dSet.binBuf[xBin]
			}
		}
	}

	if vSilent {
		return
	}

	dSet.fastWindow.Update(dSet.peakHeight)

	var vMean, vSD = dSet.slowWindow.Update(dSet.peakHeight)

	if length := dSet.slowWindow.Len(); length > dSet.fastWindow.Cap() {
		var vMag = math.Abs(dSet.fastWindow.Mean() - vMean)
		if vMag > (ScalingResetDeviation * vSD) {
			dSet.slowWindow.Drop(int(float64(length) * ScalingDumpPercent))
			vMean, vSD = dSet.slowWindow.Stats()
		}
	}

	var vMag = math.Max(vMean+(2*vSD), 1)

	for xBin, cHeight := 0, math.Floor(height-1); xBin <= dSet.numBins; xBin++ {
		dSet.binBuf[xBin] = math.Min(cHeight, (dSet.binBuf[xBin]/vMag)*cHeight)
	}
}
