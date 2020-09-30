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
	screen   tcell.Screen
	DataSets []*DataSet
	barWidth int
	binWidth int
}

// Init sets up the display
func (d *Display) Init() error {
	var err error

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
	d.binWidth = bar + space

	return d.Bars()
}

// Bars returns the number of bars we will draw
func (d *Display) Bars() int {
	var width, _ int = d.screen.Size()
	return width / d.binWidth
}

// Size returns the width and height of the screen in bars and rows
func (d *Display) Size() (int, int) {
	var width, height int = d.screen.Size()
	return (width / d.binWidth), height
}

func (d *Display) offset() int {
	var width, _ int = d.screen.Size()
	width = width - (d.binWidth * (width / d.binWidth))
	if width > 1 {
		return width / 2
	}
	return 0
}

// Draw takes data, and draws
func (d *Display) Draw() error {
	var (
		cHeight int
		cWidth  int

		cOffset int

		xCol int
		xRow int
		xBin int

		vSet    *DataSet
		vTarget int
		vDelta  int
	)

	// grab our dimensions
	cWidth, cHeight = d.screen.Size()

	// we want to break out when we have reached the max number of bars
	// we are able to display, including spacing
	cWidth = (d.binWidth * (cWidth / d.binWidth))

	// we want to draw at half height
	cHeight = cHeight / 2

	// get our offset
	cOffset = d.offset()

	// this seems a bit too much
	// can we do less work on draws, please?
	// TODO(winter): clean up draw loop
	for _, vSet = range d.DataSets {

		// our change per row will
		vDelta = 1

		// If we are looking at the first set (left channel)
		// we want to draw up
		if vSet.id == 0 {
			vDelta *= -1
		}

		// set up our loop. set the column by bin count on each loop
		for xCol, xBin = 0, 0; xCol < cWidth; xCol = xBin * d.binWidth {

			// work in our offset to center on the screen
			xCol += cOffset

			// we always want to target our bar height
			vTarget = int(vSet.Data[xBin])

			// Draw the bars for this data set
			for xRow = 0; xRow < vTarget*d.barWidth; xRow++ {
				d.screen.SetContent(

					// this is so dirty looking
					// it is done so we do not have two loops here, one for bar width,
					// and one for row. it has not been benchmarked which is better

					// TODO(nora): benchmark math vs. double loop

					xCol+(xRow/vTarget),

					cHeight+(vDelta*(1+(xRow%vTarget))),

					// Just use our const character for now
					DisplayBar, nil,

					// Working on color bars
					tcell.StyleDefault,
				)
			}

			// Draw our center line
			for xRow = xCol + d.barWidth; xCol < xRow; xCol++ {
				d.screen.SetContent(
					xCol, cHeight,
					DisplayBar, nil,
					tcell.StyleDefault,
				)
			}

			// increment the bin we are looking at.
			xBin++
		}
	}

	d.screen.Show()

	d.screen.Clear()

	return nil
}
