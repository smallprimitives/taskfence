# Task Fence

Wait for the work registered before a checkpoint. A task finishes when its
completion function is called, and tasks can finish in any order. Register
each task before starting it. Tasks registered after a checkpoint have their
own completion schedule and cannot delay that checkpoint.

```go
var group taskfence.Group
done := group.Register()
go func() {
    defer done()
    process()
}()

checkpoint := group.Checkpoint()
<-checkpoint.Done()
```

`Done` closes when all covered tasks have called their completion functions.
Writes made by those tasks before completion are visible after observing
that closure. A task that waits for a checkpoint covering itself must be
able to complete independently.

The module requires Go 1.26 and uses only the standard library.
