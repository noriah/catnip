package display

import (
	"math"

	"github.com/noriah/tavis/dsp"
)

const (
	// DisplayBar is the block we use for bars
	DisplayBar rune = '\u2588'

	// DisplaySpace is the block we use for space (if we were to print one)
	// DisplaySpace rune = '\u0020'
)

var (
	barHeightRunes = [...]rune{
		'\u0020',
		'\u2581',
		'\u2582',
		'\u2583',
		'\u2584',
		'\u2585',
		'\u2586',
		'\u2587',
		DisplayBar,
	}

	numRunes = len(barHeightRunes)

	// temp for now
	drawDir = [...]int{-1, 1}
)

func drawVars(value float64) (int, int) {
	var whole, frac = math.Modf(value)
	frac = float64(numRunes) * frac
	return int(whole), int(frac)
}

func drawBars(d *Display, ds *dsp.DataSet, height, offset, delta int) {
	delta *= drawDir[ds.ID()%2]
	height += delta

	for xBin, bin := range ds.Bins() {
		lCol := (xBin * d.binWidth) + offset + d.barWidth

		lRow, vLast := drawVars(bin)

		// TODO(nora): benchmark math (single loop) vs. double loop
		for xCol, xRow := lCol-d.barWidth, 0; xCol < lCol; xCol++ {

			// we always want to target our bar height
			for xRow = 0; xRow < lRow; xRow++ {

				// Draw the bars for this data set
				d.screen.SetContent(

					xCol,

					height+(delta*xRow),

					DisplayBar, nil,

					styleDefault,
				)
			}

			if vLast > 0 {

				// Draw the bars for this data set
				if delta < 0 {

					d.screen.SetContent(
						xCol,

						height+(delta*xRow),

						barHeightRunes[vLast], nil,

						styleDefault,
					)
				} else {
					d.screen.SetContent(
						xCol,

						height+(delta*xRow),

						barHeightRunes[numRunes-vLast], nil,

						styleReverse,
					)
				}
			}
		}
	}

	d.drawWg.Done()
}
