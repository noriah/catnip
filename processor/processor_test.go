package processor

import (
	"sync"
	"testing"

	"github.com/noriah/catnip/input"
)

const (
	BinSize = 1024
	ChCount = 2
)

type testOutput struct{}

func (tO *testOutput) Bins(int) int {
	return BinSize
}

func (tO *testOutput) Write([][]float64, int) error {
	return nil
}

type testAnalyzer struct{}

func (tA *testAnalyzer) BinCount() int {
	return BinSize
}

func (tA *testAnalyzer) ProcessBin(int, []complex128) float64 {
	return 0.0
}

func (ta *testAnalyzer) Recalculate(int) int {
	return BinSize
}

type Analyzer interface {
	BinCount() int
	ProcessBin(int, []complex128) float64
	Recalculate(int) int
}

func BenchmarkSlices(b *testing.B) {

	inputBuffers := input.MakeBuffers(ChCount, BinSize)

	cfg := Config{
		SampleRate:   122880.0,
		SampleSize:   BinSize,
		ChannelCount: ChCount,
		Buffers:      inputBuffers,
		Output:       &testOutput{},
		Analyzer:     &testAnalyzer{},
	}

	proc := New(cfg)
	proc.mu = &sync.Mutex{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		proc.Process()
	}
}
