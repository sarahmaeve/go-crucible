.PHONY: test test-race test-exercise vet status status-race verify-solution

# All exercise numbers, in order. Add new exercises here (used by both
# status targets below).
EXERCISES := 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 18 19 20 21 22

# Exercises whose fix is applied on main — see the solved_in_main field in
# .crucible/exercises.yaml. Keep the two in sync.
PRESOLVED := 10 18 19

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
	@echo "(run 'make status-race' for exercises 08 and 12 under the race detector)"
	@for n in $(EXERCISES); do \
		out=$$(go test ./... -run "^TestExercise$${n}" -count=1 -v 2>&1); \
		note=""; \
		case " $(PRESOLVED) " in *" $$n "*) note=" (pre-solved on main)";; esac; \
		if echo "$$out" | grep -qE "^--- PASS: TestExercise$${n}(_|$$)"; then \
			echo "  Exercise $$n: PASS$$note"; \
		elif echo "$$out" | grep -qE "^--- SKIP: TestExercise$${n}(_|$$)"; then \
			echo "  Exercise $$n: SKIP (see test output)"; \
		else \
			echo "  Exercise $$n: FAIL$$note"; \
		fi; \
	done

status-race:
	@echo "=== Go Crucible Exercise Status (race detector) ==="
	@for n in $(EXERCISES); do \
		out=$$(go test -race ./... -run "^TestExercise$${n}" -count=1 -v 2>&1); \
		note=""; \
		case " $(PRESOLVED) " in *" $$n "*) note=" (pre-solved on main)";; esac; \
		if echo "$$out" | grep -qE "^--- PASS: TestExercise$${n}(_|$$)"; then \
			echo "  Exercise $$n: PASS$$note"; \
		else \
			echo "  Exercise $$n: FAIL$$note"; \
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
