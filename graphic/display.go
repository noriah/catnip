package graphic

import (
	"context"
	"errors"
	"math"

	"github.com/noriah/tavis/util"

	"github.com/gdamore/tcell/v2"
)

const (

	// MaxWidth will be removed at some point
	MaxWidth = 5000

	// DrawCenterSpaces is tmp
	DrawCenterSpaces = false

	// DrawPaddingSpaces do we draw the outside padded spacing?
	DrawPaddingSpaces = false

	// DisplayBar is the block we use for bars
	DisplayBar rune = '\u2588'

	// DisplaySpace is the block we use for space (if we were to print one)
	DisplaySpace rune = '\u0020'

	// NumRunes number of runes for sub step bars
	NumRunes = 8

	// Scaling Constants

	// ScalingSlowWindow in seconds
	ScalingSlowWindow = 5

	// ScalingFastWindow in seconds
	ScalingFastWindow = ScalingSlowWindow * 0.2

	// ScalingDumpPercent is how much we erase on rescale
	ScalingDumpPercent = 0.75

	// ScalingResetDeviation standard deviations from the mean before reset
	ScalingResetDeviation = 1
)

// DrawType is the type
type DrawType int

// Draw Types
const (
	DrawDefault DrawType = iota
	DrawUp
	DrawUpDown
	DrawDown
)

var (
	barRunes = [2][NumRunes]rune{
		{
			DisplaySpace,
			'\u2581',
			'\u2582',
			'\u2583',
			'\u2584',
			'\u2585',
			'\u2586',
			'\u2587',
		},
		{
			DisplayBar,
			'\u2587',
			'\u2586',
			'\u2585',
			'\u2584',
			'\u2583',
			'\u2582',
			'\u2581',
		},
	}

	styleDefault = tcell.StyleDefault.Foreground(tcell.ColorWhite)
	styleCenter  = styleDefault.Foreground(tcell.ColorLightPink)
	// styleCenter  = styleDefault
	styleReverse = styleDefault.Reverse(true)
)

// Display handles drawing our visualizer
type Display struct {
	barWidth   int
	spaceWidth int
	binWidth   int

	slowWindow *util.MovingWindow
	fastWindow *util.MovingWindow

	screen tcell.Screen
}

// New sets up the display
// should we panic or return an error as well?
// something to think about
func New(hz float64, samples int) *Display {

	screen, err := tcell.NewScreen()

	if err != nil {
		panic(err)
	}

	if err = screen.Init(); err != nil {
		panic(err)
	}

	screen.DisableMouse()
	screen.HideCursor()

	slowMax := int((ScalingSlowWindow*hz)/float64(samples)) * 2
	fastMax := int((ScalingFastWindow*hz)/float64(samples)) * 2

	var d = &Display{
		slowWindow: util.NewMovingWindow(slowMax),
		fastWindow: util.NewMovingWindow(fastMax),
		screen:     screen,
	}

	d.SetWidths(2, 1)

	return d
}

// Start display is bad
func (d *Display) Start(ctx context.Context) context.Context {
	var dispCtx, dispCancel = context.WithCancel(ctx)
	go eventPoller(dispCtx, dispCancel, d)
	return dispCtx
}

// eventPoller will take events and do things with them
// TODO(noraih): MAKE THIS MORE ROBUST LIKE PREGO TOMATO SAUCE LEVELS OF ROBUST
func eventPoller(ctx context.Context, fn context.CancelFunc, d *Display) {
	defer fn()

	for {
		// first check if we need to exit
		select {
		case <-ctx.Done():
			return
		default:
		}

		var ev = d.screen.PollEvent()
		if ev == nil {
			return
		}

		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'q', 'Q':
					return

				default:

				}

			case tcell.KeyCtrlC:
				return

			case tcell.KeyUp:
				d.SetWidths(d.barWidth+1, d.spaceWidth)

			case tcell.KeyRight:
				d.SetWidths(d.barWidth, d.spaceWidth+1)

			case tcell.KeyDown:
				d.SetWidths(d.barWidth-1, d.spaceWidth)

			case tcell.KeyLeft:
				d.SetWidths(d.barWidth, d.spaceWidth-1)

			default:

			}

		default:

		}
	}
}

// Stop display not work
func (d *Display) Stop() error {
	return nil
}

// Close will stop display and clean up the terminal
func (d *Display) Close() error {
	d.screen.Fini()
	return nil
}

// SetWidths takes a bar width and spacing width
// Returns number of bars able to show
func (d *Display) SetWidths(bar, space int) {
	if bar < 1 {
		bar = 1
	}

	if space < 0 {
		space = 0
	}

	d.barWidth = bar
	d.spaceWidth = space
	d.binWidth = bar + space
}

// Bars returns the number of bars we will draw
func (d *Display) Bars(dt DrawType, sets ...int) int {
	var x = 1
	if len(sets) > 0 {
		x = sets[0]
	}

	var width, _ = d.screen.Size()

	switch dt {
	case DrawDefault, DrawUpDown:
		return width / d.binWidth
	case DrawUp, DrawDown:
		return (width / d.binWidth) / x
	default:
		return -1
	}
}

// Size returns the width and height of the screen in bars and rows
func (d *Display) Size(dt DrawType) (int, int) {
	var _, height = d.screen.Size()
	return d.Bars(dt), height
}

func (d *Display) updateWindow(peak float64, scale float64) float64 {
	// do some scaling if we are above 0
	if peak > 0.0 {
		d.fastWindow.Update(peak)
		var vMean, vSD = d.slowWindow.Update(peak)

		if length := d.slowWindow.Len(); length >= d.fastWindow.Cap() {

			if math.Abs(d.fastWindow.Mean()-vMean) > (ScalingResetDeviation * vSD) {
				vMean, vSD = d.slowWindow.Drop(
					int(float64(length) * ScalingDumpPercent))
			}
		}

		// value to scale by to make conditions easier to base on
		if peak/math.Max(vMean+(1.5*vSD), 1) > 1.4 {
			vMean, vSD = d.slowWindow.Drop(
				int(float64(d.slowWindow.Len()) * ScalingDumpPercent))
		}

		scale /= math.Max(vMean+(1.5*vSD), 1)
	}

	return scale
}

// Draw takes data and draws
func (d *Display) Draw(bins [][]float64, count, thick int, dt DrawType) error {

	var cSetCount = len(bins)

	if cSetCount < 1 {
		return errors.New("not enough sets to draw")
	}

	if cSetCount > 2 {
		return errors.New("too many sets to draw")
	}

	switch dt {
	case DrawUp:
		return d.drawUp(bins, count, thick)
	case DrawDefault, DrawUpDown:
		return d.drawUpDown(bins, count, thick)
	case DrawDown:
		return d.drawDown(bins, count, thick)
	default:
		return nil
	}
}

// DrawUp takes data and draws
func (d *Display) drawUp(bins [][]float64, count, thick int) error {
	var cSetCount = len(bins)

	var cWidth, cHeight = d.screen.Size()
	cHeight -= thick

	var cChanWidth = (d.binWidth * count) - d.spaceWidth
	var cPaddedWidth = (d.binWidth * count * cSetCount) - d.spaceWidth
	var cOffset = (cWidth - cPaddedWidth) / 2

	var peak = getPeak(bins, count)

	var fHeight = float64(cHeight)
	var scale = d.updateWindow(peak, fHeight)

	var calcStopAndTop = func(value float64) (stop int, top int) {
		top = int(math.Min(fHeight, value*scale) * NumRunes)
		stop = cHeight - (top / NumRunes)
		top %= NumRunes

		if stop < 0 {
			stop = 0
			top = 0
		}

		return
	}

	var xBin int
	var xCol = cOffset
	var delta = 1

	for xCh := range bins {
		var stop, top = calcStopAndTop(bins[xCh][xBin])

		var lCol = xCol + d.barWidth
		var lColMax = xCol + cChanWidth

		for xCol < lColMax {

			if xCol >= lCol {
				if xBin += delta; xBin >= count || xBin < 0 {
					break
				}

				stop, top = calcStopAndTop(bins[xCh][xBin])

				xCol += d.spaceWidth
				lCol = xCol + d.barWidth
			}

			var xRow = cHeight + thick

			for xRow >= cHeight {
				d.screen.SetContent(xCol, xRow,
					DisplayBar, nil, styleCenter)

				xRow--
			}

			for xRow >= stop {
				d.screen.SetContent(xCol, xRow,
					DisplayBar, nil, styleDefault)

				xRow--
			}

			if top > 0 {
				d.screen.SetContent(xCol, xRow,
					barRunes[0][top], nil, styleDefault)
			}

			xCol++
		}

		xCol += d.spaceWidth
		delta = -delta
	}

	d.screen.Show()

	d.screen.Clear()

	return nil
}

func (d *Display) drawUpDown(bins [][]float64, count, thick int) error {
	var cSetCount = len(bins)
	var cHaveRight = cSetCount == 2

	// We dont keep track of the offset/width because we have to assume that
	// the user changed the window, always. It is easier to do this now, and
	// implement SIGWINCH handling later on (or not?)
	var cWidth, cHeight = d.screen.Size()

	var cPaddedWidth = (d.binWidth * count) - d.spaceWidth

	if cPaddedWidth > cWidth || cPaddedWidth < 0 {
		cPaddedWidth = cWidth
	}

	var cOffset = (cWidth - cPaddedWidth) / 2

	var centerStart = (cHeight - thick) / cSetCount
	var centerStop = centerStart + thick

	var peak = getPeak(bins, count)
	var fHeight = float64(centerStart)
	var scale = d.updateWindow(peak, fHeight)

	var calcStopAndTop = func(value float64) (stop int, top int) {
		top = int(math.Min(fHeight, value*scale) * NumRunes)
		stop = top / NumRunes
		top %= NumRunes

		return
	}

	var xBin = 0
	var xCol = cOffset

	// TODO(nora): benchmark
	for xCol < cWidth {

		var leftStop, leftTop = calcStopAndTop(bins[0][xBin])
		var startRow = centerStart - leftStop

		if leftTop > 0 {
			startRow--
		}

		if startRow < 0 {
			startRow = 0
		}

		var rightStop, rightTop int
		var stopRow = centerStop

		if cHaveRight {
			rightStop, rightTop = calcStopAndTop(bins[1][xBin])
			stopRow += rightStop
		}

		if stopRow >= cHeight {
			stopRow = cHeight - 1
		}

		var lCol = xCol + d.barWidth

		for xCol < lCol {

			var xRow = startRow

			if leftTop > 0 {
				d.screen.SetContent(xCol, xRow,
					barRunes[0][leftTop], nil, styleDefault)
				xRow++
			}

			for xRow < centerStart {
				d.screen.SetContent(xCol, xRow,
					DisplayBar, nil, styleDefault)
				xRow++
			}

			// center line
			for xRow < centerStop {
				d.screen.SetContent(xCol, xRow,
					DisplayBar, nil, styleCenter)
				xRow++
			}

			if cHaveRight {
				// right bars go down
				for xRow < stopRow {
					d.screen.SetContent(xCol, xRow,
						DisplayBar, nil, styleDefault)
					xRow++
				}

				// last part of right bars.
				if rightTop > 0 {
					d.screen.SetContent(xCol, xRow,
						barRunes[1][rightTop], nil, styleReverse)
				}
			}

			xCol++
		}

		xBin++

		if xBin >= count {
			break
		}

		xCol += d.spaceWidth
	}

	d.screen.Show()

	d.screen.Clear()

	return nil
}

// DrawUp takes data and draws
func (d *Display) drawDown(bins [][]float64, count, thick int) error {
	var cSetCount = len(bins)

	var cWidth, cHeight = d.screen.Size()

	var cChanWidth = (d.binWidth * count) - d.spaceWidth
	var cPaddedWidth = (d.binWidth * count * cSetCount) - d.spaceWidth
	var cOffset = (cWidth - cPaddedWidth) / 2

	var peak = getPeak(bins, count)

	var fHeight = float64(cHeight - thick)
	var scale = d.updateWindow(peak, fHeight)

	var calcStopAndTop = func(value float64) (stop int, top int) {
		top = int(math.Min(fHeight, value*scale) * NumRunes)
		stop = (top / NumRunes) + thick
		top %= NumRunes

		if stop > cHeight {
			stop = cHeight
			top = 0
		}

		return
	}

	var xBin int
	var xCol = cOffset
	var delta = 1

	for xCh := range bins {
		var stop, top = calcStopAndTop(bins[xCh][xBin])

		var lCol = xCol + d.barWidth
		var lColMax = xCol + cChanWidth

		for xCol < lColMax {

			if xCol >= lCol {
				if xBin += delta; xBin >= count || xBin < 0 {
					break
				}

				stop, top = calcStopAndTop(bins[xCh][xBin])

				xCol += d.spaceWidth
				lCol = xCol + d.barWidth
			}

			var xRow = 0

			for xRow < thick {
				d.screen.SetContent(xCol, xRow,
					DisplayBar, nil, styleCenter)

				xRow++
			}

			for xRow < stop {
				d.screen.SetContent(xCol, xRow,
					DisplayBar, nil, styleDefault)

				xRow++
			}

			if top > 0 {
				d.screen.SetContent(xCol, xRow,
					barRunes[1][top], nil, styleReverse)
			}

			xCol++
		}

		xCol += d.spaceWidth
		delta = -delta
	}

	d.screen.Show()

	d.screen.Clear()

	return nil
}

func getPeak(bins [][]float64, count int) float64 {
	var peak = 0.0

	for xSet := 0; xSet < len(bins); xSet++ {
		for xBin := 0; xBin < count; xBin++ {
			if peak < bins[xSet][xBin] {
				peak = bins[xSet][xBin]
			}
		}
	}

	return peak
}
