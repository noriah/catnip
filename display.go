package tavis

import (
	"os"

	"github.com/gdamore/tcell/v2"
)

// DisplayBar is the block we use for bars
const (
	DisplayBar   rune = '\u2588'
	DisplaySpace rune = '\u0020'

	MaxWidth = 5000
)

var directions = [2]int{1, -1}

// Display handles drawing our visualizer
type Display struct {
	screen     tcell.Screen
	barWidth   int
	spaceWidth int
}

// Init sets up the display
func (d *Display) Init() error {
	var err error

	// cellBuf = &tcell.CellBuffer{}

	if d.screen, err = tcell.NewScreen(); err != nil {
		return err
	}

	if err = d.screen.Init(); err != nil {
		return err
	}

	d.screen.DisableMouse()
	d.screen.HideCursor()

	d.SetWidths(1, 1)

	return nil
}

// Start display is bad
func (d *Display) Start() error {
	go func() {
		var ev tcell.Event
		for ev = d.screen.PollEvent(); ev != nil; ev = d.screen.PollEvent() {
			d.HandleEvent(ev)
		}
	}()

	return nil
}

func (d *Display) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyCtrlC:
			os.Interrupt.Signal()
		default:

		}

	default:
	}

	return false
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
	d.barWidth = bar
	d.spaceWidth = space

	return d.Bars()
}

// Bars returns the number of bars we will draw
func (d *Display) Bars() int {
	var width, _ int = d.screen.Size()
	var perBar int = d.barWidth + d.spaceWidth

	width = width + d.spaceWidth
	width = width / perBar

	return width
}

// Size returns the width and height of the screen in bars and rows
func (d *Display) Size() (int, int) {
	var _, height int = d.screen.Size()
	var width = d.Bars()
	return width, height
}

// Draw takes data, and draws
func (d *Display) Draw(buf []float64, ch int) error {
	var (
		totalHeight int
		totalBars   int

		centerHeight int

		barSpaceWidth int

		bax     int // bar index
		rox     int
		col     int
		chx     int
		bHeight int
		dir     int
	)

	totalBars, totalHeight = d.Size()

	barSpaceWidth = d.barWidth + d.spaceWidth

	centerHeight = totalHeight / ch

	// Please do not do this.
	// Seriously.
	// Do not.
	for chx = 0; chx < ch; chx++ {
		dir = directions[chx%2]
		for bax = 0; bax < totalBars; bax++ {
			for col = (bax * barSpaceWidth); col < (bax*barSpaceWidth + d.barWidth); col++ {
				bHeight = int(buf[bax*ch+chx])
				if bHeight < 1 {
					bHeight = 1
				}
				for rox = 0; rox < bHeight; rox++ {

					d.screen.SetContent(
						bax*barSpaceWidth, centerHeight-(dir*rox),
						DisplayBar, nil, tcell.StyleDefault)
				}

				if rox < centerHeight {

					for ; rox <= centerHeight; rox++ {
						d.screen.SetContent(
							bax*barSpaceWidth, centerHeight-(dir*rox),
							DisplaySpace, nil, tcell.StyleDefault)
					}
				}
			}
		}
	}

	// d.screen.Clear()

	d.screen.Sync()

	return nil
}
