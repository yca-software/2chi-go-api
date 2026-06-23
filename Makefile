.PHONY: test test-integration hookdeck-paddle

API_PORT ?= 1300
PADDLE_WEBHOOK_PATH ?= /api/v1/webhooks/paddle

# Unit tests (CI default — no Docker).
test:
	go test -race -count=1 -tags=!integration ./...

# Repository integration tests (Docker required). -p 1 keeps package output readable;
# cross-process locking in testutil serializes shared-container access when -p > 1.
test-integration:
	go test -tags=integration -race -count=1 -p 1 ./internals/repositories/...

# Forward Paddle sandbox webhooks to the local API (requires Hookdeck CLI).
# Configure Paddle notification URL to the Hookdeck source URL shown on startup.
hookdeck-paddle:
	hookdeck listen $(API_PORT) paddle --path $(PADDLE_WEBHOOK_PATH) --output compact
