package dsp

import "math"

// Generate makes numBins and dumps them in the buffer
func Generate(sp *Spectrum, ds *DataSet) {
	ds.numBins = sp.numBins

	for xBin := 0; xBin <= ds.numBins; xBin++ {

		var vM = 0.0

		for xF := sp.loCuts[xBin]; xF <= sp.hiCuts[xBin] &&
			xF >= 0 &&
			xF < ds.fftSize; xF++ {

			vM += pyt(ds.fftBuf[xF])
		}

		vM /= float64(sp.hiCuts[xBin] - sp.loCuts[xBin] + 1)

		vM *= (math.Log2(float64(2+xBin)) * (100.0 / float64(ds.numBins)))

		ds.binBuf[xBin] = math.Pow(vM, 0.5)
	}
}

func pyt(value complex128) float64 {
	return math.Sqrt((real(value) * real(value)) + (imag(value) * imag(value)))
}
