package taskfence_test

import (
	"context"
	"testing"

	"github.com/smallprimitives/taskfence"
)

func FuzzCheckpointReadiness(f *testing.F) {
	const (
		register = iota
		complete
		checkpoint
		checkCanceledWait
	)
	// A completion byte uses its upper six bits as the task index.
	f.Add([]byte{checkpoint, checkCanceledWait})
	f.Add([]byte{register, register, checkpoint, complete + 4, checkpoint, checkCanceledWait, complete})
	f.Add([]byte{register, checkpoint, register, checkpoint, complete, complete, complete + 4})

	f.Fuzz(func(t *testing.T, operations []byte) {
		// The reference model keeps every task and checks every checkpoint.
		if len(operations) > 64 {
			operations = operations[:64]
		}

		type task struct {
			finish func()
			done   bool
		}
		type snapshot struct {
			checkpoint taskfence.Checkpoint
			boundary   int
		}
		var group taskfence.Group
		var tasks []task
		var snapshots []snapshot
		canceled, cancel := context.WithCancel(context.Background())
		cancel()

		for step, operation := range operations {
			switch operation & 3 {
			case register:
				tasks = append(tasks, task{finish: group.Register()})
			case complete:
				if index := int(operation >> 2); index < len(tasks) {
					tasks[index].finish()
					tasks[index].done = true
				}
			case checkpoint:
				snapshots = append(snapshots, snapshot{group.Checkpoint(), len(tasks)})
			}

			for i, saved := range snapshots {
				wantReady := true
				for _, task := range tasks[:saved.boundary] {
					if !task.done {
						wantReady = false
						break
					}
				}
				var gotReady bool
				select {
				case <-saved.checkpoint.Done():
					gotReady = true
				default:
				}
				if gotReady != wantReady {
					t.Fatalf("step %d, checkpoint %d: ready = %t, want %t", step, i, gotReady, wantReady)
				}

				if operation&3 == checkCanceledWait {
					var wantErr error
					if !wantReady {
						wantErr = context.Canceled
					}
					if err := saved.checkpoint.Wait(canceled); err != wantErr {
						t.Fatalf("step %d, checkpoint %d: Wait = %v, want %v", step, i, err, wantErr)
					}
				}
			}
		}

		for _, task := range tasks {
			task.finish()
		}
		for _, saved := range snapshots {
			requireReady(t, saved.checkpoint)
		}
	})
}
