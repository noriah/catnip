package tavis

import (
	"github.com/gdamore/tcell/v2"
)

// DisplayBar is the block we use for bars
const (
	DisplayBar   rune = '\u2588'
	DisplaySpace rune = '\u0020'

	MaxWidth = 5000
)

// Display handles drawing our visualizer
type Display struct {
	screen     tcell.Screen
	DataSets   []*DataSet
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
func (d *Display) Start(endCh chan<- bool) error {
	go func() {
		var ev tcell.Event
		for ev = d.screen.PollEvent(); ev != nil; ev = d.screen.PollEvent() {
			if d.HandleEvent(ev) {
				break
			}
		}
		endCh <- true
	}()

	return nil
}

// HandleEvent will take events and do things with them
func (d *Display) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyCtrlC:
			return true
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
func (d *Display) Draw() error {
	var (
		cHeight int
		cWidth  int

		barSpaceWidth int

		xCol int
		xRow int
		xBin int

		vSet    *DataSet
		vTarget int
	)

	barSpaceWidth = d.barWidth + d.spaceWidth

	cWidth, cHeight = d.screen.Size()

	cHeight = cHeight / 2
	// if cHeight%2 == 0 {
	// 	cHeight++
	// }

	var offset int = 0

	for xCol = offset; xCol < cWidth; xCol++ {
		if (xCol%barSpaceWidth)/d.barWidth > 0 {
			continue
		}

		d.screen.SetContent(xCol, cHeight, DisplayBar, nil, tcell.StyleDefault)

		xBin = (xCol / barSpaceWidth)

		for _, vSet = range d.DataSets {
			vTarget = cHeight

			if vSet.id == 1 {
				vTarget = cHeight + 1 + int(vSet.Data[xBin])
			}

			for xRow = vTarget - int(vSet.Data[xBin]); xRow < vTarget; xRow++ {
				d.screen.SetContent(xCol, xRow, DisplayBar, nil, tcell.StyleDefault)
			}
		}
	}

	d.screen.Show()
	d.screen.Clear()

	return nil
}
