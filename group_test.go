package taskfence_test

import (
	"sync"
	"testing"
	"time"

	"github.com/smallprimitives/taskfence"
)

func requireReady(t *testing.T, checkpoint taskfence.Checkpoint) {
	t.Helper()
	select {
	case <-checkpoint.Done():
	default:
		t.Fatal("checkpoint should be ready")
	}
}

func requirePending(t *testing.T, checkpoint taskfence.Checkpoint) {
	t.Helper()
	select {
	case <-checkpoint.Done():
		t.Fatal("checkpoint became ready before its tasks completed")
	default:
	}
}

func TestEmptyAndZeroCheckpointsAreReady(t *testing.T) {
	var group taskfence.Group
	requireReady(t, taskfence.Checkpoint{})
	requireReady(t, group.Checkpoint())
	group.Register()()
	requireReady(t, group.Checkpoint())
}

func TestCheckpointWaitsForOutOfOrderCompletions(t *testing.T) {
	var group taskfence.Group
	finishA := group.Register()
	finishB := group.Register()
	finishC := group.Register()
	checkpoint := group.Checkpoint()

	requirePending(t, checkpoint)
	finishC()
	requirePending(t, checkpoint)
	finishA()
	requirePending(t, checkpoint)
	finishB()
	requireReady(t, checkpoint)
}

func TestLaterRegistrationsDoNotDelayEarlierCheckpoint(t *testing.T) {
	var group taskfence.Group
	finishEarly := group.Register()
	early := group.Checkpoint()
	finishLate := group.Register()
	late := group.Checkpoint()

	finishEarly()
	requireReady(t, early)
	requirePending(t, late)
	finishLate()
	requireReady(t, early)
	requireReady(t, late)
}

func TestLaterCompletionCannotSkipAnEarlierGeneration(t *testing.T) {
	var group taskfence.Group
	finishFirst := group.Register()
	first := group.Checkpoint()
	finishSecond := group.Register()
	second := group.Checkpoint()
	finishThird := group.Register()
	third := group.Checkpoint()

	finishThird()
	finishSecond()
	requirePending(t, first)
	requirePending(t, second)
	requirePending(t, third)
	finishFirst()
	requireReady(t, first)
	requireReady(t, second)
	requireReady(t, third)
}

func TestIdenticalPendingCheckpointsShareReadiness(t *testing.T) {
	var group taskfence.Group
	finish := group.Register()
	first := group.Checkpoint()
	second := group.Checkpoint()
	if first.Done() != second.Done() {
		t.Fatal("identical pending checkpoints should share a channel")
	}

	finish()
	requireReady(t, first)
	requireReady(t, second)
}

func TestCompletionCanBeCalledConcurrentlyMoreThanOnce(t *testing.T) {
	var group taskfence.Group
	finish := group.Register()
	finishOther := group.Register()
	checkpoint := group.Checkpoint()

	var callers sync.WaitGroup
	for range 32 {
		callers.Go(finish)
	}
	callers.Wait()

	requirePending(t, checkpoint)
	finishOther()
	requireReady(t, checkpoint)
}

func TestCompletionPublishesTaskWrites(t *testing.T) {
	var group taskfence.Group
	finishFirst := group.Register()
	first := group.Checkpoint()
	finishSecond := group.Register()
	second := group.Checkpoint()

	var values [2]int
	go func() {
		values[0] = 10
		finishFirst()
	}()
	go func() {
		values[1] = 20
		finishSecond()
	}()
	select {
	case <-second.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("checkpoint did not finish")
	}
	requireReady(t, first)
	if values != [2]int{10, 20} {
		t.Fatalf("after checkpoint, got %v, want [10 20]", values)
	}
}
