package taskfence

import "context"

var alreadyDone = func() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}()

// Checkpoint is a copyable handle for a registration boundary.
// A zero-value Checkpoint is already ready.
type Checkpoint struct {
	done <-chan struct{}
}

// Done returns a channel closed once all tasks covered by the checkpoint
// have completed. The channel stays closed thereafter.
func (c Checkpoint) Done() <-chan struct{} {
	if c.done == nil {
		return alreadyDone
	}
	return c.done
}

// Wait waits for the checkpoint to become ready or the context to end.
// If readiness is observable when cancellation is checked, Wait returns nil.
// Otherwise it returns ctx.Err(). Cancellation affects only this call.
func (c Checkpoint) Wait(ctx context.Context) error {
	done := c.Done()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// If both channels are ready, select may choose cancellation.
		select {
		case <-done:
			return nil
		default:
			return ctx.Err()
		}
	}
}
