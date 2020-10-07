package display

import (
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/noriah/tavis/dsp"

	"github.com/pkg/errors"
)

const (

	// MaxWidth will be removed at some point
	MaxWidth = 5000

	// DrawCenterSpaces is tmp
	DrawCenterSpaces = false

	// DrawPaddingSpaces do we draw the outside padded spacing?
	DrawPaddingSpaces = true

	// DisplayBar is the block we use for bars
	DisplayBar rune = '\u2588'

	// DisplaySpace is the block we use for space (if we were to print one)
	DisplaySpace rune = '\u0020'
)

var (
	barRunes = [...][2]rune{
		{DisplaySpace, DisplayBar},
		{'\u2581', '\u2587'},
		{'\u2582', '\u2586'},
		{'\u2583', '\u2585'},
		{'\u2584', '\u2584'},
		{'\u2585', '\u2583'},
		{'\u2586', '\u2582'},
		{'\u2587', '\u2581'},
		{DisplayBar, DisplaySpace},
	}

	numRunes = len(barRunes)

	styleDefault = tcell.StyleDefault.Bold(true)
	styleCenter  = styleDefault.Foreground(tcell.ColorOrangeRed)
	styleReverse = tcell.StyleDefault.Reverse(true).Bold(true)
)

// Display handles drawing our visualizer
type Display struct {
	barWidth int
	binWidth int

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

	return &Display{
		barWidth: 2,
		binWidth: 3,
		screen:   screen,
	}
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
			d.setBarBin(d.barWidth+1, d.binWidth)
		case tcell.KeyRight:
			d.setBarBin(d.barWidth, d.binWidth+1)
		case tcell.KeyDown:
			d.setBarBin(d.barWidth-1, d.binWidth)
		case tcell.KeyLeft:
			d.setBarBin(d.barWidth, d.binWidth-1)
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

func (d *Display) setBarBin(bar, bin int) {
	if bar < 1 {
		bar = 1
	}

	if bin < bar {
		bin = bar
	}

	d.barWidth = bar
	d.binWidth = bin
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
	d.binWidth = bar + space

	return d.Bars()
}

// Bars returns the number of bars we will draw
func (d *Display) Bars() int {
	var width, _ = d.screen.Size()
	return (width + (d.binWidth - d.barWidth)) / d.binWidth
}

// Size returns the width and height of the screen in bars and rows
func (d *Display) Size() (int, int) {
	var _, height = d.screen.Size()
	return d.Bars(), height
}

func drawVars(bin float64) (int, int) {
	var whole, frac = math.Modf(bin)
	frac = float64(numRunes) * math.Abs(frac)
	return int(whole), int(frac)
}

// Draw takes data and draws
func (d *Display) Draw(height, delta int, sets ...*dsp.DataSet) error {
	var cSetCount = len(sets)

	if cSetCount < 1 {
		return errors.New("not enough sets to draw")
	}

	if cSetCount > 2 {
		return errors.New("too many sets to draw")
	}

	// We dont keep track of the offset/width because we have to assume that
	// the user changed the window always. It is easier to do this now, and
	// implement SIGWINCH handling later on (or not?)
	var cWidth = d.Bars() * d.binWidth
	var cPaddedWidth, _ = d.screen.Size()
	var cOffset = (cPaddedWidth - cWidth) / 2

	// xCol will be our column index.
	var xCol = cOffset

	// we want to break out when we have reached the max number of bars
	// we are able to display, including spacing
	var xBin, lBin = 0, sets[0].Len()

	// TODO(nora): benchmark
	for xBin < lBin {

		// We don't want to be calling lookups for the same value over and over
		// we also dont know how wide the bars are going to be
		var lRow, vLast = drawVars(sets[0].Bins()[xBin])
		var lRowN, vLastN = drawVars(sets[1%cSetCount].Bins()[xBin])
		var lCol = xCol + d.barWidth

		for xCol < lCol {

			// Draw Center Line
			d.screen.SetContent(
				xCol, height,
				DisplayBar, nil, styleCenter)

			// Invert delta becauase we draw up first
			// Set the style of the top block to be default
			var vDelta, vStyle = -delta, styleDefault
			var xSet = 0

			for xSet < cSetCount {
				// start at row 1 as row 0 is the center line. we handle that separate
				var xRow = 1

				for xRow <= lRow && xRow < height {
					d.screen.SetContent(
						xCol, height+(vDelta*xRow),
						DisplayBar, nil, styleDefault)

					xRow++
				}

				if vLast >= 0 {

					// Draw the top bar for this data set
					d.screen.SetContent(
						xCol, height+(vDelta*xRow),
						barRunes[vLast][xSet], nil, vStyle)
				}

				// we will only ever allow 2 loops
				// we are done with the first, set up the second

				// Invert the direction of drawing (up/down)
				vDelta = -vDelta

				// set the top bar rune cell style to be inverse of the default
				// We do this because unicode does not yet have standardized
				// codepoints for upper eighth/quarter blocks.
				// To get around this, we print the inverse character, with the
				// style reversed to appear as we do have the needed blocks
				vStyle = styleReverse

				// Swap our row limits with the next/previous set.
				// Each time this loop exits, it is called again on a new column
				// and we want to do everything again.
				// Same for our last block index
				// These will be updated to new values (both sets), when we change
				// bin index (xBin)
				lRow, lRowN = lRowN, lRow
				vLast, vLastN = vLastN, vLast

				xSet++
			}

			xCol++
		}

		xBin++

		// do we want to draw a center line throughout the entire
		if DrawCenterSpaces {

			lCol := (xBin * d.binWidth) + cOffset

			for xCol < lCol {
				d.screen.SetContent(
					xCol, height,
					DisplayBar, nil, styleCenter)

				xCol++
			}
		}

		xCol = (xBin * d.binWidth) + cOffset
	}

	d.screen.Show()

	d.screen.Clear()

	return nil
}
