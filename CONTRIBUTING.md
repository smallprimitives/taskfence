# Contributing

Open an issue or pull request for a proposed change. Report vulnerabilities
privately using the instructions in [SECURITY.md](SECURITY.md).

Use Go 1.26 or later. Run these checks from the repository root.

```sh
go test ./...
go vet ./...
go test -race ./...
go test -run='^$' -fuzz=FuzzCheckpointReadiness -fuzztime=10s .
go test -run='^$' -bench=. -benchmem .
```

Use channels to synchronize concurrency tests. Use timeouts to bound test
failures, and include a reproducible test case or fuzz seed with regression fixes.
