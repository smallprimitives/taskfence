package taskfence_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/smallprimitives/taskfence"
)

func BenchmarkRegisterComplete(b *testing.B) {
	b.Run("taskfence", func(b *testing.B) {
		var group taskfence.Group
		for b.Loop() {
			group.Register()()
		}
	})
	b.Run("waitgroup", func(b *testing.B) {
		var group sync.WaitGroup
		for b.Loop() {
			group.Add(1)
			group.Done()
		}
	})
	b.Run("parallel", func(b *testing.B) {
		var group taskfence.Group
		b.RunParallel(func(iterations *testing.PB) {
			for iterations.Next() {
				group.Register()()
			}
		})
	})
}

func BenchmarkCheckpoint(b *testing.B) {
	for _, batch := range []int{1, 32} {
		b.Run(fmt.Sprint(batch), func(b *testing.B) {
			var group taskfence.Group
			finish := make([]func(), batch)
			for b.Loop() {
				for i := range finish {
					finish[i] = group.Register()
				}
				checkpoint := group.Checkpoint()
				for i := len(finish) - 1; i >= 0; i-- {
					finish[i]()
				}
				<-checkpoint.Done()
			}
		})
	}
}

func BenchmarkOldestTaskPending(b *testing.B) {
	const laterCount = 256
	b.Run("taskfence", func(b *testing.B) {
		var group taskfence.Group
		for b.Loop() {
			finishFirst := group.Register()
			var last taskfence.Checkpoint
			for range laterCount {
				finish := group.Register()
				last = group.Checkpoint()
				finish()
			}
			finishFirst()
			<-last.Done()
		}
	})
	b.Run("mutex-map-reference", func(b *testing.B) {
		var group mapFence
		for b.Loop() {
			finishFirst := group.register()
			var last <-chan struct{}
			for range laterCount {
				finish := group.register()
				last = group.checkpoint()
				finish()
			}
			finishFirst()
			<-last
		}
	})
}

// mapFence is a benchmark reference that checks the active task IDs against
// the oldest pending checkpoint after each completion.
type mapFence struct {
	mu      sync.Mutex
	next    uint64
	active  map[uint64]bool
	pending []mapCheckpoint
}

type mapCheckpoint struct {
	target uint64
	done   chan struct{}
}

func (f *mapFence) register() func() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.active == nil {
		f.active = make(map[uint64]bool)
	}
	f.next++
	id := f.next
	f.active[id] = true
	return sync.OnceFunc(func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		delete(f.active, id)
		for len(f.pending) != 0 {
			for active := range f.active {
				if active <= f.pending[0].target {
					return
				}
			}
			close(f.pending[0].done)
			f.pending[0] = mapCheckpoint{}
			f.pending = f.pending[1:]
		}
		f.pending = nil
	})
}

func (f *mapFence) checkpoint() <-chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	done := make(chan struct{})
	f.pending = append(f.pending, mapCheckpoint{f.next, done})
	return done
}
