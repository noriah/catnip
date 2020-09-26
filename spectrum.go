package tavis

type BarType = int32

type BarBuffer []BarType

type FreqType = float64

type FreqBins []FreqType

type Spectrum struct {
	LowFreqCut, HighFreqCut FreqType

	constBin FreqBins

	bars BarBuffer
}

func (s *Spectrum) Init() error {
	return nil
}

func (s *Spectrum) Recalculate(num int) {

}
