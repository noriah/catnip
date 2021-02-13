// Package timer provides a way to concurrently read from a blocking audio
// source while guaranteeing a consistent tick for smooth drawing.
package timer

import (
	"time"

	"github.com/noriah/catnip/input"
)

// Process runs a new timer routine. The timer routine is ticked everytime
// callback returns or when a tick is supposed to occur based on the given
// sample rate and sample size.
//
// The given callback will be called in a busy loop; it should block until an
// event or error occurs.
//
// Processor is called on each tick.
func Process(cfg input.SessionConfig, proc input.Processor, callback func() error) error {
	// Calculate the theoretical tick duration to satisfy the requested sampling
	// rate without falling behind.
	rate := time.Second / time.Duration(cfg.SampleRate/float64(cfg.SampleSize))

	// Use a new ticker as a new synchronization routine in parallel with the
	// current one. We have to do this because the read will sometimes miss,
	// causing a gap. We would want to call Process on a consistent tick.
	ticker := time.NewTicker(rate)
	defer ticker.Stop()

	var errorCh = make(chan error)
	go func() {
		for {
			err := callback()
			errorCh <- err

			// Bail on error.
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		// Sustain the requested frame rate.
		case <-ticker.C:

		case err := <-errorCh:
			if err != nil {
				return err
			}

			// Re-synchronize the ticker to occur on the next supposed cursor to
			// minimize latency.
			ticker.Reset(rate)
		}

		proc.Process()
	}
}
