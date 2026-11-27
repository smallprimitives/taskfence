package taskfence

import (
	"math"
	"testing"
)

func TestCompletedCheckpointEntriesAreReclaimed(t *testing.T) {
	var group Group
	finishFirst := group.Register()
	first := group.Checkpoint()
	const laterCount = 256
	for range laterCount {
		finish := group.Register()
		group.Checkpoint()
		finish()
	}

	queued := 0
	for entry := group.head; entry != nil; entry = entry.next {
		queued++
	}
	if queued != laterCount+1 {
		t.Fatalf("queued %d checkpoint entries, want %d", queued, laterCount+1)
	}

	finishFirst()
	if group.head != nil || group.tail != nil {
		t.Fatal("group retained resolved checkpoint entries")
	}
	select {
	case <-first.Done():
	default:
		t.Fatal("first checkpoint is still pending")
	}
}

func TestRegisterRejectsGenerationOverflow(t *testing.T) {
	group := Group{current: &generation{pending: math.MaxUint64}}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("Register did not panic for a full generation")
			}
		}()
		group.Register()
	}()
	if group.current.pending != math.MaxUint64 {
		t.Fatal("Register changed the pending count before panicking")
	}
	group.current.pending = 0
	group.Register()()
}
