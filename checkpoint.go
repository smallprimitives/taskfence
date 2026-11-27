package taskfence

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
