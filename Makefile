.PHONY: test test-race test-exercise vet status verify-solution

test:
	go test ./...

test-race:
	go test -race -count=5 ./...

test-exercise:
	@if [ -z "$(N)" ]; then echo "Usage: make test-exercise N=01"; exit 1; fi
	go test ./... -run "TestExercise$(N)" -v

vet:
	go vet ./...

status:
	@echo "=== Go Crucible Exercise Status ==="
	@for n in 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 18 19; do \
		result=$$(go test ./... -run "TestExercise$${n}" -count=1 2>&1); \
		if echo "$$result" | grep -q "PASS"; then \
			echo "  Exercise $$n: PASS"; \
		else \
			echo "  Exercise $$n: FAIL"; \
		fi; \
	done

verify-solution:
	@if [ -z "$(N)" ]; then echo "Usage: make verify-solution N=01"; exit 1; fi
	@tmpdir=$$(mktemp -d) && \
	cp -r . "$$tmpdir/go-crucible" && \
	cd "$$tmpdir/go-crucible" && \
	git apply solutions/$(N)-*.patch && \
	go test ./... -run "TestExercise$(N)" -v && \
	echo "Solution $(N) verified successfully" && \
	rm -rf "$$tmpdir"
