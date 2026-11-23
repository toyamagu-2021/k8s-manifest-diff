---
name: verify
description: Drive and verify the k8s-manifest-diff CLI (diff, parse, version subcommands) in an isolated workspace with captured stdout, stderr, and exit-code proof artifacts. Use whenever verifying CLI behavior, output formatting, filtering flags, or Secret masking in k8s-manifest-diff.
---
# Verify k8s-manifest-diff

`k8s-manifest-diff` is a short-lived Go CLI tool (`./cmd/k8s-manifest-diff`) and Go library (`pkg/diff`, `pkg/parser`, `pkg/filter`, `pkg/masking`) that parses Kubernetes YAML/JSON manifests, masks `Secret` data, filters resources by kind/label/annotation, and outputs unified diffs or summaries. The primary user surface is the `k8s-manifest-diff` binary (`diff`, `parse`, and `version` subcommands).

Read [`.agents/skills/verify/features/README.md`](features/README.md) before driving a feature. Each feature file under [`.agents/skills/verify/features/`](features/) provides exact commands, preconditions, expected outputs, and repo-specific gotchas.

## Launch

`k8s-manifest-diff` has no daemon, port, or database. Launching builds the binary with current git revision `ldflags` (`main.version`, `main.commit`, `main.date`) into a disposable per-run workspace (`/tmp/k8s-manifest-diff-verify-$RUN_ID`) and copies `testing/e2e/fixtures/` into that workspace so runs never mutate tracked files or collide across concurrent sessions.

Pick a unique `RUN_ID` for your verification run and launch:

```bash
export RUN_ID="verify-$(date +%s)-$$"
.agents/skills/verify/scripts/verify-cli launch
```

Ready indicator: stdout prints `READY` followed by `RUN_ID`, `WORK_DIR`, `PROOF_DIR`, `BIN_PATH`, and `COMMIT`.

## Doctor

Run `doctor` before driving commands or whenever output looks unexpected. It checks that `instance.env` exists, the compiled binary at `/tmp/k8s-manifest-diff-verify-$RUN_ID/bin/k8s-manifest-diff` matches its recorded SHA-256 and current git `HEAD` commit, the copied baseline fixtures exist, and the proof directory is writable.

```bash
.agents/skills/verify/scripts/verify-cli doctor
```

Healthy output begins with `DOCTOR OK` and exits `0`. If `doctor` exits non-zero, re-run `launch` instead of driving a stale binary.

## Drive

Drive every CLI invocation through `.agents/skills/verify/scripts/verify-cli run`. The helper runs `/tmp/k8s-manifest-diff-verify-$RUN_ID/bin/k8s-manifest-diff` with working directory `/tmp/k8s-manifest-diff-verify-$RUN_ID`, where `fixtures/` contains an isolated copy of `testing/e2e/fixtures/` and `scratch/` holds any temporary manifests created for the journey.

Exit code contract of `k8s-manifest-diff`:
- `0`: No differences found (`diff`), or command succeeded (`parse`, `version`, `--help`).
- `1`: Differences found (`diff`).
- `2`: Runtime or CLI argument error (missing file, malformed YAML, invalid `--output-format`, non-string Secret value, or bad flag).

Pass `--expect-exit <code>` to assert the CLI exit code while capturing proof:

```bash
# Compare manifests with differences (exit code 1)
.agents/skills/verify/scripts/verify-cli run \
  --artifact diff-manifests/basic-changed \
  --expect-exit 1 \
  -- diff fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml

# Compare identical manifests (exit code 0)
.agents/skills/verify/scripts/verify-cli run \
  --artifact diff-manifests/basic-identical \
  --expect-exit 0 \
  -- diff fixtures/basic/identical.yaml fixtures/basic/identical.yaml

# Parse and mask secrets in a single manifest (exit code 0)
.agents/skills/verify/scripts/verify-cli run \
  --artifact parse-manifests/single-secret \
  --expect-exit 0 \
  -- parse fixtures/basic/secret-with-data-base.yaml
```

Two instances can run side by side safely whenever each uses a distinct `RUN_ID` (or distinct `VERIFY_WORK_DIR` and `VERIFY_PROOF_DIR`). Never share one `RUN_ID` across concurrent agents.

## Evidence

Every `verify-cli run --artifact <feature>/<step>` writes four files under `/tmp/k8s-manifest-diff-proof-$RUN_ID/<feature>/` (or `$VERIFY_PROOF_DIR/<feature>/`):

- `<step>.command.txt`: Exact shell-quoted `k8s-manifest-diff` invocation.
- `<step>.stdout.txt`: Complete raw stdout.
- `<step>.stderr.txt`: Complete raw stderr.
- `<step>.exit_code.txt`: Numeric exit code returned by the binary.

Proof standards:
- Drive the compiled `k8s-manifest-diff` binary through its real CLI flags and manifest files, never internal Go functions or unit-test helpers.
- Capture both positive paths (diff detected, filter matched, secret masked) and contrasting control paths (no diff, filtered out, error exit code `2`) as specified in the feature file.
- Inspect the written `.stdout.txt`, `.stderr.txt`, and `.exit_code.txt` files directly. Do not rely on truncated output or exit code alone.

## Cleanup

Tear down the disposable workspace after verification completes or after any failed iteration:

```bash
.agents/skills/verify/scripts/verify-cli cleanup
```

`cleanup` deletes only `/tmp/k8s-manifest-diff-verify-$RUN_ID` (`WORK_DIR`) and preserves `/tmp/k8s-manifest-diff-proof-$RUN_ID` (`PROOF_DIR`) so all proof files survive teardown. After running `cleanup`, confirm the proof files still exist under `/tmp/k8s-manifest-diff-proof-$RUN_ID`.

## Helpers

[`.agents/skills/verify/scripts/verify-cli`](scripts/verify-cli) is the executable harness script for this repository:

```bash
# 1. Build isolated binary and seed fixtures
.agents/skills/verify/scripts/verify-cli launch

# 2. Verify binary health, checksum, commit revision, and fixture state
.agents/skills/verify/scripts/verify-cli doctor

# 3. Execute a CLI journey step and record command, stdout, stderr, and exit code
.agents/skills/verify/scripts/verify-cli run --artifact <feature>/<step> [--expect-exit <code>] -- <k8s-manifest-diff-args...>

# 4. Remove the disposable work directory while retaining proof artifacts
.agents/skills/verify/scripts/verify-cli cleanup
```
