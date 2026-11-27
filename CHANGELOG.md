# Changelog

## v0.1.0

- Add zero-value task groups with idempotent completion functions.
- Add checkpoints that cover the tasks registered before capture, including tasks that complete out of order.
- Add cancellable waits that leave registered tasks and other waiters running.
- Support Go 1.26 and later with only the standard library.
