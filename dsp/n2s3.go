package dsp

// N2S3State is the stateholder for N2S3
type N2S3State struct {
	bins []float64
}

// NewN2S3State returns a new N2S3 state.
func NewN2S3State(max int) *N2S3State {

	var state = &N2S3State{
		bins: make([]float64, max),
	}

	return state
}

// N2S3 does nora's not so special smoothing
func N2S3(buf []float64, count int, state *N2S3State, smth, res float64) {

	for xB := 0; xB < count; xB++ {
		buf[xB] += state.bins[xB] * smth

		state.bins[xB] = buf[xB] * (1 - (1 / (1 + (buf[xB] * res))))
	}
}
