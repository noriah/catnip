package display

import (
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
)

var (
	barRunes = [2][8]rune{
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

	numRunes = 8

	styleDefault = tcell.StyleDefault.Bold(false)
	styleCenter  = styleDefault.Foreground(tcell.ColorOrangeRed)
	// styleCenter  = styleDefault.Foreground(tcell.ColorDefault)
	styleReverse = tcell.StyleDefault.Reverse(true).Bold(true)
)

// Display handles drawing our visualizer
type Display struct {
	barWidth   int
	spaceWidth int
	binWidth   int

	screen tcell.Screen
}

// New sets up the display
// should we panic or return an error as well?
// something to think about
func New() *Display {

	screen, err := tcell.NewScreen()

	if err != nil {
		panic(err)
	}

	if err = screen.Init(); err != nil {
		panic(err)
	}

	screen.DisableMouse()
	screen.HideCursor()

	var d = &Display{
		screen: screen,
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
	return (width + d.binWidth - d.barWidth) / d.binWidth
}

// Size returns the width and height of the screen in bars and rows
func (d *Display) Size() (int, int) {
	var _, height = d.screen.Size()
	return d.Bars(), height
}

// Draw takes data and draws
func (d *Display) Draw(height, delta, count int, left, right []float64) {
	var cSetCount = 1
	if right != nil {
		cSetCount++
	}

	// We dont keep track of the offset/width because we have to assume that
	// the user changed the window always. It is easier to do this now, and
	// implement SIGWINCH handling later on (or not?)
	var cPaddedWidth, cHeight = d.screen.Size()
	var cWidth = (d.binWidth * (count - 1)) + d.barWidth

	if cWidth > cPaddedWidth || cWidth < 0 {
		cWidth = cPaddedWidth
	}

	var cOffset = (cPaddedWidth - cWidth) / 2

	if DrawPaddingSpaces {
		for xCol := 0; xCol < cOffset; xCol++ {
			d.screen.SetContent(xCol, height, DisplayBar, nil, styleCenter)
		}
	}

	// we want to break out when we have reached the max number of bars
	// we are able to display, including spacing
	var xBin = 0
	var xCol = cOffset

	// TODO(nora): benchmark
	for xCol < cWidth {

		// We don't want to be calling lookups for the same value over and over
		// we also dont know how wide the bars are going to be
		var leftPart = int(left[xBin] * float64(numRunes))

		var rightPart = 0
		if right != nil {
			rightPart = int(right[xBin] * float64(numRunes))
		}

		var startRow = height - (((leftPart / numRunes) + 1) * delta)

		var lRow = startRow + (((leftPart / numRunes) + (rightPart / numRunes) + 2) * delta)

		if lRow > cHeight {
			lRow = cHeight
		}

		leftPart %= numRunes
		rightPart %= numRunes

		var lCol = xCol + d.barWidth

		for xCol < lCol {

			var xRow = startRow

			if leftPart > 0 {
				d.screen.SetContent(
					xCol, xRow, barRunes[0][leftPart], nil, styleDefault)
			}

			xRow += delta

			for xRow < height {
				d.screen.SetContent(xCol, xRow, DisplayBar, nil, styleDefault)
				xRow += delta
			}

			d.screen.SetContent(xCol, xRow, DisplayBar, nil, styleCenter)
			xRow += delta

			for xRow < lRow {
				d.screen.SetContent(xCol, xRow, DisplayBar, nil, styleDefault)
				xRow += delta
			}

			if rightPart > 0 {
				d.screen.SetContent(
					xCol, xRow, barRunes[1][rightPart], nil, styleReverse)
			}

			xCol++
		}

		xBin++

		if xBin >= count {
			break
		}

		// do we want to draw a center line throughout the entire
		if DrawCenterSpaces {
			var lCol = xCol + d.spaceWidth
			for xCol < lCol {
				d.screen.SetContent(xCol, height, DisplayBar, nil, styleCenter)
				xCol++
			}
		} else {
			xCol += d.spaceWidth
		}
	}

	if DrawPaddingSpaces {
		for xCol < cPaddedWidth {
			d.screen.SetContent(xCol, height, DisplayBar, nil, styleCenter)
			xCol++
		}
	}

	d.screen.Show()

	d.screen.Clear()

}
