package graphic

import "github.com/nsf/termbox-go"

func drawUpDown(bins [][]float64, count int, cfg Config, scale float64) error {
	var cSetCount = len(bins)

	// We dont keep track of the offset/width because we have to assume that
	// the user changed the window, always. It is easier to do this now, and
	// implement SIGWINCH handling later on (or not?)
	var cWidth, cHeight = termbox.Size()

	var centerStart = (cHeight - cfg.BaseThick) / 2
	if centerStart < 0 {
		centerStart = 0
	}

	var centerStop = centerStart + cfg.BaseThick

	scale = float64(centerStart) / scale

	var xBin = 0
	var xCol = (cWidth - ((cfg.BinWidth * count) - cfg.SpaceWidth)) / 2

	if xCol < 0 {
		xCol = 0
	}

	// TODO(nora): benchmark

	var lStop, lTop = stopAndTop(bins[0][xBin]*scale, centerStart, true)
	var rStop, rTop = stopAndTop(bins[1%cSetCount][xBin]*scale, centerStart, false)
	if rStop += centerStop; rStop >= cHeight {
		rStop = cHeight
		rTop = BarRune
	}

	var lCol = xCol + cfg.BarWidth

	for {

		if xCol >= lCol {

			if xCol >= cWidth {
				break
			}

			if xBin++; xBin >= count {
				break
			}

			lStop, lTop = stopAndTop(bins[0][xBin]*scale, centerStart, true)
			rStop, rTop = stopAndTop(bins[1%cSetCount][xBin]*scale, centerStart, false)
			if rStop += centerStop; rStop >= cHeight {
				rStop = cHeight
				rTop = BarRune
			}

			xCol += cfg.SpaceWidth
			lCol = xCol + cfg.BarWidth
		}

		var xRow = lStop

		if lTop > BarRuneR {
			termbox.SetCell(xCol, xRow-1, lTop, StyleDefault, StyleDefaultBack)

		}

		for xRow < centerStart {
			termbox.SetCell(xCol, xRow, BarRune, StyleDefault, StyleDefaultBack)
			xRow++
		}

		// center line
		for xRow < centerStop {
			termbox.SetCell(xCol, xRow, BarRune, StyleCenter, StyleDefaultBack)
			xRow++
		}

		// right bars go down
		for xRow < rStop {
			termbox.SetCell(xCol, xRow, BarRune, StyleDefault, StyleDefaultBack)
			xRow++
		}

		// last part of right bars.
		if rTop < BarRune {
			termbox.SetCell(xCol, xRow, rTop, StyleReverse, StyleDefault)
		}

		xCol++
	}

	return nil
}
