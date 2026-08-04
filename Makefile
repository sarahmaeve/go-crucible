.PHONY: test test-race test-exercise vet status status-race verify-solution \
	verify verify-quick verify-vet verify-sanity verify-failures verify-patches \
	bridge bridge-check bridge-serve

HUGO ?= hugo
BRIDGE_BIND ?= 127.0.0.1
BRIDGE_PORT ?= 1313
BRIDGE_BASE_URL ?= http://localhost:$(BRIDGE_PORT)/

# All exercise numbers, in order. Add new exercises here (used by the
# status and verify targets; tools/verify checks this list against the
# registry).
EXERCISES := 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 18 19 20 21 22

# Exercises whose fix is applied on main — see the solved_in_main field in
# .crucible/exercises.yaml. Keep the two in sync (tools/verify checks).
PRESOLVED := 10 18 19

# Exercises whose tests need -race to produce their expected result on
# the buggy tree (08 skips without it; 12 may pass by luck without it).
RACE_EXERCISES := 08 12

test:
	go test ./...

test-race:
	go test -race -count=5 ./...

test-exercise:
	@if [ -z "$(N)" ]; then echo "Usage: make test-exercise N=01"; exit 1; fi
	@raceflag=""; \
	case " $(RACE_EXERCISES) " in *" $(N) "*) raceflag="-race";; esac; \
	go test $$raceflag ./... -run "^TestExercise$(N)" -count=1 -v

vet:
	go vet ./...

bridge:
	$(HUGO) --source cs-prod-bridge --cleanDestinationDir

bridge-check:
	@tmpdir=$$(mktemp -d); \
	trap 'rm -rf "$$tmpdir"' EXIT; \
	$(HUGO) --source cs-prod-bridge --destination "$$tmpdir" --quiet \
	&& diff -r cs-prod-bridge/public "$$tmpdir"

# Hugo's development server renders to memory by default, so this does not
# replace the committed production build under cs-prod-bridge/public. The
# local base URL is supplied only here; Cloudflare supplies its deployment URL.
bridge-serve:
	$(HUGO) server \
		--source cs-prod-bridge \
		--bind $(BRIDGE_BIND) \
		--port $(BRIDGE_PORT) \
		--baseURL $(BRIDGE_BASE_URL) \
		--renderToMemory \
		--disableFastRender

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
	@tmpdir=$$(mktemp -d); \
	trap 'rm -rf "$$tmpdir"' EXIT; \
	cp -R . "$$tmpdir/go-crucible" || exit 1; \
	cd "$$tmpdir/go-crucible" || exit 1; \
	patch=$$(ls solutions/$(N)-*.patch); \
	raceflag=""; \
	case " $(RACE_EXERCISES) " in *" $(N) "*) raceflag="-race";; esac; \
	case " $(PRESOLVED) " in \
	*" $(N) "*) \
		git apply -R "$$patch" \
		&& ! go test $$raceflag ./... -run "^TestExercise$(N)" -count=1 \
		&& git apply "$$patch" \
		&& go test $$raceflag ./... -run "^TestExercise$(N)" -count=1;; \
	*) \
		git apply "$$patch" \
		&& go test $$raceflag ./... -run "^TestExercise$(N)" -count=1;; \
	esac || exit 1; \
	echo "Solution $(N) verified successfully"

# ---------------------------------------------------------------------------
# Local verification harness. Remote CI is not available to this repo, so
# these targets are the rot protection — run verify-quick before any commit
# that touches exercises, and verify before anything that touches source
# files or patches.
#
#   verify-quick     seconds — structural checks (tools/verify)
#   verify           minutes — everything: quick + vet + sanity + expected
#                    failures + patch round-trips in a sandboxed copy
# ---------------------------------------------------------------------------

verify-quick:
	go run ./tools/verify

verify-vet:
	@out=$$(go vet ./... 2>&1) || true; \
	n=$$(printf '%s\n' "$$out" | grep -c "WaitGroup.Add called from inside new goroutine") || true; \
	other=$$(printf '%s\n' "$$out" | grep -v "WaitGroup.Add called from inside new goroutine" | grep -v '^#' | grep -c .) || true; \
	if [ "$$n" -eq 1 ] && [ "$$other" -eq 0 ]; then \
		echo "vet: OK (exactly the one expected WaitGroup.Add warning)"; \
	else \
		echo "vet: FAIL (want exactly one WaitGroup.Add warning and nothing else; got $$n + $$other other lines)"; \
		printf '%s\n' "$$out"; \
		exit 1; \
	fi

verify-sanity:
	@echo "sanity: all non-exercise tests must pass on the buggy tree"
	go test ./... -skip '^TestExercise'

verify-failures:
	@echo "failures: every exercise test must FAIL on the buggy tree (pre-solved must PASS)"
	@fail=0; \
	for n in $(EXERCISES); do \
		raceflag=""; \
		case " $(RACE_EXERCISES) " in *" $$n "*) raceflag="-race";; esac; \
		out=$$(go test $$raceflag ./... -run "^TestExercise$${n}" -count=1 -v 2>&1); \
		if echo "$$out" | grep -qE "^--- PASS: TestExercise$${n}(_|$$)"; then state=PASS; \
		elif echo "$$out" | grep -qE "^--- SKIP: TestExercise$${n}(_|$$)"; then state=SKIP; \
		else state=FAIL; fi; \
		case " $(PRESOLVED) " in \
			*" $$n "*) want=PASS;; \
			*) want=FAIL;; \
		esac; \
		if [ "$$state" = "$$want" ]; then \
			echo "  exercise $$n: $$state (as expected)"; \
		else \
			echo "  exercise $$n: $$state — EXPECTED $$want"; fail=1; \
		fi; \
	done; \
	exit $$fail

verify-patches:
	@echo "patches: every solution must round-trip (apply <-> revert with matching test results)"
	@tmpdir=$$(mktemp -d); \
	cp -R . "$$tmpdir/go-crucible"; \
	cd "$$tmpdir/go-crucible" || exit 1; \
	fail=0; \
	for n in $(EXERCISES); do \
		patch=$$(ls solutions/$$n-*.patch); \
		raceflag=""; \
		case " $(RACE_EXERCISES) " in *" $$n "*) raceflag="-race";; esac; \
		case " $(PRESOLVED) " in \
		*" $$n "*) \
			if git apply -R "$$patch" \
			&& ! go test $$raceflag ./... -run "^TestExercise$${n}" -count=1 >/dev/null 2>&1 \
			&& git apply "$$patch" \
			&& go test $$raceflag ./... -run "^TestExercise$${n}" -count=1 >/dev/null 2>&1; then \
				echo "  exercise $$n: OK (pre-solved round-trip)"; \
			else \
				echo "  exercise $$n: FAIL (pre-solved round-trip broken)"; fail=1; \
				git apply "$$patch" 2>/dev/null || true; \
			fi;; \
		*) \
			if git apply "$$patch" \
			&& go test $$raceflag ./... -run "^TestExercise$${n}" -count=1 >/dev/null 2>&1 \
			&& git apply -R "$$patch" \
			&& ! go test $$raceflag ./... -run "^TestExercise$${n}" -count=1 >/dev/null 2>&1; then \
				echo "  exercise $$n: OK (solution round-trip)"; \
			else \
				echo "  exercise $$n: FAIL (solution round-trip broken)"; fail=1; \
				git apply -R "$$patch" 2>/dev/null || true; \
			fi;; \
		esac; \
	done; \
	cd /; rm -rf "$$tmpdir"; \
	exit $$fail

verify: verify-quick verify-vet verify-sanity verify-failures verify-patches
	@echo ""
	@echo "verify: all checks passed"
