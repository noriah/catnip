package dsp

// N2S3State is the stateholder for N2S3
type N2S3State struct {
	bins  []float64
	level float64
}

// NewN2S3State returns a new N2S3 state.
func NewN2S3State(max int) *N2S3State {

	var state = &N2S3State{
		bins:  make([]float64, max),
		level: 1,
	}

	return state
}

// N2S3 does nora's not so special smoothing
func N2S3(buf []float64, count int, state *N2S3State, factor float64) {

	var dip = 1.0

	for xBin := 0; xBin < count; xBin++ {

		buf[xBin] += state.bins[xBin] * factor

		if buf[xBin] > dip {
			dip = buf[xBin]
		}

		var diff = state.level - buf[xBin]
		if diff < 0.0 {
			diff = 0.0
		}

		state.bins[xBin] = buf[xBin] * (1 - ((1 / (diff + 1)) / 20))
	}

	state.level = dip
}
