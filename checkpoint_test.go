package taskfence_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/smallprimitives/taskfence"
)

func TestWaitOnPendingCheckpointWithEndedContext(t *testing.T) {
	var group taskfence.Group
	finish := group.Register()
	checkpoint := group.Checkpoint()

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()

	if err := checkpoint.Wait(canceled); err != context.Canceled {
		t.Fatalf("Wait returned %v, want context.Canceled", err)
	}
	if err := checkpoint.Wait(expired); err != context.DeadlineExceeded {
		t.Fatalf("Wait returned %v, want context.DeadlineExceeded", err)
	}
	requirePending(t, checkpoint)

	finish()
	if err := checkpoint.Wait(canceled); err != nil {
		t.Fatalf("ready checkpoint with a canceled context returned %v", err)
	}
	if err := (taskfence.Checkpoint{}).Wait(expired); err != nil {
		t.Fatalf("zero checkpoint with an expired context returned %v", err)
	}
}

func TestCancelingOneWaiterDoesNotAffectAnother(t *testing.T) {
	var group taskfence.Group
	finish := group.Register()
	defer finish()
	checkpoint := group.Checkpoint()

	canceled, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelingWaiter := startWaiter(t, checkpoint, canceled)
	remainingWaiter := startWaiter(t, checkpoint, context.Background())

	cancel()
	requireWaitResult(t, cancelingWaiter, context.Canceled)
	requirePending(t, checkpoint)
	select {
	case err := <-remainingWaiter:
		t.Fatalf("remaining waiter finished prematurely: %v", err)
	default:
	}

	finish()
	requireWaitResult(t, remainingWaiter, nil)
}

func TestWaitPrefersReadinessWhenBothChannelsBecomeReady(t *testing.T) {
	var group taskfence.Group
	for range 64 {
		finish := group.Register()
		checkpoint := group.Checkpoint()
		ctx, cancel := context.WithCancel(context.Background())
		observed := observedContext{Context: ctx, onDone: func() {
			cancel()
			finish()
		}}

		if err := checkpoint.Wait(observed); err != nil {
			t.Fatalf("Wait returned %v with both channels ready, want nil", err)
		}
	}
}

type observedContext struct {
	context.Context
	onDone func()
}

func (c observedContext) Done() <-chan struct{} {
	c.onDone()
	return c.Context.Done()
}

func startWaiter(t *testing.T, checkpoint taskfence.Checkpoint, ctx context.Context) <-chan error {
	t.Helper()
	waiting := make(chan struct{})
	result := make(chan error, 1)
	observed := observedContext{Context: ctx, onDone: sync.OnceFunc(func() { close(waiting) })}
	go func() { result <- checkpoint.Wait(observed) }()
	select {
	case <-waiting:
	case <-time.After(5 * time.Second):
		t.Fatal("waiter did not reach Wait")
	}
	return result
}

func requireWaitResult(t *testing.T, waiter <-chan error, want error) {
	t.Helper()
	select {
	case got := <-waiter:
		if got != want {
			t.Fatalf("Wait returned %v, want %v", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waiter did not finish")
	}
}
