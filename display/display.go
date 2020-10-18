package display

import (
	"math"

	"github.com/noriah/tavis/util"

	"github.com/gdamore/tcell/v2"
	"github.com/pkg/errors"
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
	ScalingResetDeviation = 0.9
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
func New(hz float64, samples int) Display {

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

	var d = Display{
		slowWindow: util.NewMovingWindow(slowMax),
		fastWindow: util.NewMovingWindow(fastMax),
		screen:     screen,
	}

	d.SetWidths(2, 1)

	return d
}

// Start display is bad
func (d *Display) Start(endCh chan<- bool) error {
	go func() {
		var ev tcell.Event
		for ev = d.screen.PollEvent(); ev != nil; ev = d.screen.PollEvent() {
			if d.HandleEvent(ev) != nil {
				break
			}
		}
		endCh <- true
	}()

	return nil
}

// HandleEvent will take events and do things with them
// TODO(noraih): MAKE THIS MORE ROBUST LIKE PREGO TOMATO SAUCE LEVELS OF ROBUST
func (d *Display) HandleEvent(ev tcell.Event) error {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				return errors.New("rename this better please")
			default:
			}
		case tcell.KeyCtrlC:
			return errors.New("rename this better please")
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

	return nil
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
func (d *Display) SetWidths(bar, space int) int {
	if bar < 1 {
		bar = 1
	}

	if space < 0 {
		space = 0
	}

	d.barWidth = bar
	d.spaceWidth = space
	d.binWidth = bar + space
	return d.Bars()
}

// Bars returns the number of bars we will draw
func (d *Display) Bars() int {
	var width, _ = d.screen.Size()
	if width%d.binWidth >= d.barWidth {
		return (width / d.binWidth) + 1
	}
	return width / d.binWidth
}

// Size returns the width and height of the screen in bars and rows
func (d *Display) Size() (int, int) {
	var _, height = d.screen.Size()
	return d.Bars(), height
}

// Draw takes data and draws
func (d *Display) Draw(bHeight, delta, count int, bins ...[]float64) error {
	var cSetCount = len(bins)

	if cSetCount < 1 {
		return errors.New("not enough sets to draw")
	}

	if cSetCount > 2 {
		return errors.New("too many sets to draw")
	}

	var haveRight = cSetCount == 2

	// We dont keep track of the offset/width because we have to assume that
	// the user changed the window, always. It is easier to do this now, and
	// implement SIGWINCH handling later on (or not?)
	var cPaddedWidth, cHeight = d.screen.Size()
	var cWidth = (d.binWidth * count) - d.spaceWidth

	if cWidth > cPaddedWidth || cWidth < 0 {
		cWidth = cPaddedWidth
	}

	var cOffset = (cPaddedWidth - cWidth) / 2

	var height = cHeight
	if haveRight {
		height /= 2
	}

	var centerStart = height - (bHeight / 2)

	var centerStop = centerStart + bHeight

	var peak = 0.0

	for xSet := 0; xSet < cSetCount; xSet++ {
		for xBin := 0; xBin < count; xBin++ {
			if peak < bins[xSet][xBin] {
				peak = bins[xSet][xBin]
			}
		}
	}

	var scale = 1.0
	var fHeight = float64(centerStart - 1)

	// do some scaling if we are above 0
	if peak > 0 {
		d.fastWindow.Update(peak)
		var vMean, vSD = d.slowWindow.Update(peak)

		if length := d.slowWindow.Len(); length >= d.fastWindow.Cap() {

			if math.Abs(d.fastWindow.Mean()-vMean) > (ScalingResetDeviation * vSD) {
				vMean, vSD = d.slowWindow.Drop(int(float64(length) * ScalingDumpPercent))
			}
		}

		// value to scale by to make conditions easier to base on
		scale = fHeight / math.Max(vMean+(1.5*vSD), math.SmallestNonzeroFloat64)
	}

	// if DrawPaddingSpaces {
	// 	for xCol := 0; xCol < cOffset; xCol++ {
	// 		d.screen.SetContent(xCol, height, DisplayBar, nil, styleCenter)
	// 	}
	// }

	// we want to break out when we have reached the max number of bars
	// we are able to display, including spacing
	var xBin = 0
	var xCol = cOffset

	cWidth += cOffset

	// TODO(nora): benchmark
	for xCol < cWidth {

		// Left Channel
		var leftPart = int(math.Min(fHeight, bins[0][xBin]*scale) * NumRunes)

		var startRow = centerStart - (((leftPart / NumRunes) + 1) * delta)

		if startRow < 0 {
			startRow = 0
		}

		leftPart %= NumRunes

		var lRow = centerStop

		// Right Channel
		var rightPart = 0

		if haveRight {
			rightPart = int(math.Min(fHeight, bins[1][xBin]*scale) * NumRunes)
			lRow += (rightPart / NumRunes) * delta
			rightPart %= NumRunes
		}

		if lRow > cHeight {
			lRow = cHeight
		}

		var lCol = xCol + d.barWidth

		for xCol < lCol {

			var xRow = startRow

			if leftPart > 0 {
				d.screen.SetContent(
					xCol, xRow, barRunes[0][leftPart], nil, styleDefault)
			}

			xRow += delta

			for xRow < centerStart {
				d.screen.SetContent(xCol, xRow, DisplayBar, nil, styleDefault)
				xRow += delta
			}

			// center line
			for xRow < centerStop {
				d.screen.SetContent(xCol, xRow, DisplayBar, nil, styleCenter)
				xRow += delta
			}

			if haveRight {

				// right bars go down
				for xRow < lRow {
					d.screen.SetContent(xCol, xRow, DisplayBar, nil, styleDefault)
					xRow += delta
				}

				// last part of right bars.
				if rightPart > 0 {
					d.screen.SetContent(
						xCol, xRow, barRunes[1][rightPart], nil, styleReverse)
				}
			}

			xCol++
		}

		xBin++

		if xBin >= count {
			break
		}

		// do we want to draw a center line throughout the entire stage
		// if DrawCenterSpaces {
		// 	lCol = xCol + d.spaceWidth
		// 	for xCol < lCol {
		// 		d.screen.SetContent(xCol, height, DisplayBar, nil, styleCenter)
		// 		xCol++
		// 	}
		// } else {
		xCol += d.spaceWidth
		// }
	}

	// if DrawPaddingSpaces {
	// 	for xCol < cPaddedWidth {
	// 		d.screen.SetContent(xCol, height, DisplayBar, nil, styleCenter)
	// 		xCol++
	// 	}
	// }

	d.screen.Show()

	d.screen.Clear()

	return nil
}
