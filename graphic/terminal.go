package graphic

import (
	"os"
	"strings"
)

// normalizeTerminal looks for incompatibilities in the terminal configuration
// with the underlying rendering libraries (Termbox) and makes some adjustments
// to avoid problems.
//
// Returns a function that allows you to restore the terminal configuration to its original state.
func normalizeTerminal() (func(), error) {
	prevTERMINFO := os.Getenv("TERMINFO")

	if strings.HasPrefix(os.Getenv("TERM"), "tmux") {
		// Some combinations of TERMINFO with TERM in some Tmux value
		// will cause Termbox to fail.
		if err := os.Unsetenv("TERMINFO"); err != nil {
			return nil, err
		}
	}

	restore := func() {
		if err := os.Setenv("TERMINFO", prevTERMINFO); err != nil {
			panic(err)
		}
	}

	return restore, nil
}
