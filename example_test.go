package taskfence_test

import (
	"context"
	"fmt"

	"github.com/smallprimitives/taskfence"
)

func ExampleGroup_Checkpoint() {
	var group taskfence.Group
	var records [2]string
	for i, record := range []string{"first", "second"} {
		finish := group.Register()
		go func() {
			defer finish()
			records[i] = record
		}()
	}

	flush := group.Checkpoint()
	later := group.Register()
	defer later()

	if err := flush.Wait(context.Background()); err != nil {
		return
	}
	fmt.Println("flushed", records)
	// Output: flushed [first second]
}
