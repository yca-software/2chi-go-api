.PHONY: test test-integration

# Unit tests (CI default — no Docker).
test:
	go test -race -count=1 -tags=!integration ./...

# Repository integration tests (Docker required). -p 1 keeps package output readable;
# cross-process locking in testutil serializes shared-container access when -p > 1.
test-integration:
	go test -tags=integration -race -count=1 -p 1 ./internals/repositories/...
