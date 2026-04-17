# Solutions

One patch per exercise. Each patch is the canonical fix — the smallest diff
that turns the intentionally-buggy source into source that passes the
exercise test.

## When to use these

**Don't** open a patch before you've attempted the exercise. The learning
value of the Crucible is in the struggle, and the patch file explicitly
shows the fix. If you're stuck, work through `exercises/NN-*/HINTS.md`
first — those are progressive and designed to unblock you without giving
away the answer.

Open a patch when:

- You've fixed an exercise your own way and want to compare against the
  canonical form.
- You want to study several patterns side by side (e.g., read 08, 12, 13
  to compare race-condition fixes).
- You're a maintainer regenerating or verifying the patch.

## Applying, reverting, verifying

Each patch supports all three.

### Apply (introduce the fix)

```bash
git apply solutions/NN-<name>.patch
```

After applying, the exercise's test should pass:

```bash
make test-exercise N=NN
```

### Revert (reintroduce the bug)

```bash
git apply -R solutions/NN-<name>.patch
```

Useful when:

- The exercise is pre-solved on `main` (see the list below) and you want
  the buggy form to practise on.
- You want to reset after applying the solution for study.

### Verify (sandboxed apply + test)

```bash
make verify-solution N=NN
```

This copies the repo into a temporary directory, applies the patch there,
runs the matching exercise test, and reports pass/fail. It does not touch
your working tree.

## Exercises pre-solved on `main`

As of the current commit, these exercises are applied on the default
branch — their tests pass out of the box. Apply the inverse to practise:

| # | Title | Reintroduce |
|---|-------|-------------|
| 10 | The Hanging Health Check | `git apply -R solutions/10-hanging-health-check.patch` |
| 18 | The Ticking Leak | `git apply -R solutions/18-ticking-leak.patch` |
| 19 | The Graceless Shutdown | `git apply -R solutions/19-graceless-shutdown.patch` |

The authoritative list lives in `.crucible/exercises.yaml` (look for the
`solved_in_main:` field). Maintainers who solve additional exercises on
main should update that list and add a row here.

## Patch format

Patches are generated with `diff -u` (or `git diff`) against the full file
path. They are context-line patches, not binary patches. Because the
diff includes 3 lines of context around each hunk, small unrelated
refactors around an exercise can cause a patch to stop applying cleanly.
When that happens, regenerate the patch rather than editing it by hand —
see `.crucible/README.md` for the source-of-truth rules.

## Maintainer workflow: regenerating a patch

When an unrelated refactor breaks a patch:

1. Apply the current (broken) patch: if it partially applies, back out
   with `git checkout -- <files>`.
2. Manually write the fix on top of the buggy source. Confirm the
   exercise test passes.
3. Capture the diff:
   ```bash
   git diff <changed-files> > solutions/NN-<name>.patch
   ```
   Add the header block (title, app, difficulty, file, function) from the
   previous version of the patch.
4. Revert your working tree to the buggy state:
   ```bash
   git apply -R solutions/NN-<name>.patch
   ```
5. Confirm the exercise test fails again.
6. Run `make verify-solution N=NN` as a final sanity check.

A CI workflow that runs `make verify-solution` for every exercise on each
PR would catch patch rot automatically — it's on the roadmap.
