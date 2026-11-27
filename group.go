// Package taskfence waits for the tasks registered before a checkpoint.
package taskfence

import (
	"math"
	"sync"
)

// Group tracks tasks and their checkpoints. Its zero value is ready to use.
// A Group must not be copied after first use.
type Group struct {
	mu      sync.Mutex
	current *generation
	head    *generation
	tail    *generation
}

// Generations are sealed by checkpoints. A sealed generation stays queued
// until it and every earlier generation have completed.
type generation struct {
	pending uint64
	done    chan struct{}
	next    *generation
}

// Register records a task and returns an idempotent completion function.
// Register before starting or submitting the task. The completion function
// may be called from any goroutine, including concurrently. Register panics
// if a single generation already contains math.MaxUint64 outstanding tasks.
func (g *Group) Register() func() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.current == nil {
		g.current = new(generation)
	}
	gen := g.current
	if gen.pending == math.MaxUint64 {
		panic("taskfence: too many outstanding tasks in one generation")
	}
	gen.pending++
	return sync.OnceFunc(func() { g.complete(gen) })
}

// Checkpoint captures the tasks registered before it. Later registrations
// cannot delay it.
func (g *Group) Checkpoint() Checkpoint {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.current != nil && g.current.pending != 0 {
		gen := g.current
		gen.done = make(chan struct{})
		g.current = nil
		if g.tail == nil {
			g.head = gen
		} else {
			g.tail.next = gen
		}
		g.tail = gen
	}

	if g.tail == nil {
		return Checkpoint{}
	}
	return Checkpoint{done: g.tail.done}
}

func (g *Group) complete(gen *generation) {
	g.mu.Lock()
	defer g.mu.Unlock()

	gen.pending--
	for g.head != nil && g.head.pending == 0 {
		head := g.head
		g.head = head.next
		head.next = nil
		close(head.done)
	}
	if g.head == nil {
		g.tail = nil
	}
}
