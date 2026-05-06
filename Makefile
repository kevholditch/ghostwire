.PHONY: test test-unit test-e2e

test: test-unit test-e2e

test-unit:
	@echo "==> go test -v -count=1 ./..."
	go test -v -count=1 ./...

test-e2e:
	@echo "==> go test -v -count=1 -tags=e2e ./e2e"
	go test -v -count=1 -tags=e2e ./e2e
