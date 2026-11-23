# Compare manifests with diff

Comparing manifests with `k8s-manifest-diff diff` computes differences between two Kubernetes YAML or JSON manifest files, reports summary comments and unified diffs per changed, created, or deleted resource, adjusts context lines via `--context`, and returns exit code `0` for no differences, `1` when differences exist, or `2` on error.

## Sub-features

- `diff-identical` returns exit code `0` and prints `No differences found` when manifests have no differences.
- `diff-changed` returns exit code `1` with summary comment headers and unified diff blocks for modified resources.
- `diff-created-deleted` detects resources present only in head (`Create`) or only in base (`Delete`).
- `diff-context` controls the number of surrounding context lines in unified diff blocks via `--context`.
- `diff-error` returns exit code `2` with an error message on stderr when a file is missing or arguments are invalid.

## How to get to it (user POV)

- Run `k8s-manifest-diff diff <base-file> <head-file>` in a terminal.
- Pass `--context <n>` to `k8s-manifest-diff diff` to change the unified diff context window from its default of `3`.

## Driving it with verify-cli

Preconditions:

- `verify-cli doctor` prints `DOCTOR OK` for the current `RUN_ID`.
- Baseline fixtures `fixtures/basic/identical.yaml`, `fixtures/basic/test-base.yaml`, and `fixtures/basic/test-head.yaml` exist under `/tmp/k8s-manifest-diff-verify-$RUN_ID/`.

- **Compare identical manifests.** Diff `identical.yaml` against itself. Run `.agents/skills/verify/scripts/verify-cli run --artifact diff-manifests/identical --expect-exit 0 -- diff fixtures/basic/identical.yaml fixtures/basic/identical.yaml`. Exit code is `0` and stdout is `No differences found`.
- **Compare changed manifests.** Diff `test-base.yaml` against `test-head.yaml`. Run `.agents/skills/verify/scripts/verify-cli run --artifact diff-manifests/changed --expect-exit 1 -- diff fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml`. Exit code is `1` and stdout contains `# Summary: 3 total, 3 changed, 0 created, 0 deleted, 0 unchanged`, resource headers `===== apps/Deployment default/frontend-app ======`, `===== apps/Deployment default/backend-app ======`, and `===== /ConfigMap default/app-config ======`, and unified diff headers `--- frontend-app-live.yaml` and `+++ frontend-app.yaml`.
- **Detect created and deleted resources.** Create an empty scratch manifest `touch "/tmp/k8s-manifest-diff-verify-${RUN_ID}/scratch/empty.yaml"`, then diff `scratch/empty.yaml` against `fixtures/basic/identical.yaml` and reverse the order. Run `.agents/skills/verify/scripts/verify-cli run --artifact diff-manifests/created --expect-exit 1 -- diff scratch/empty.yaml fixtures/basic/identical.yaml` and `.agents/skills/verify/scripts/verify-cli run --artifact diff-manifests/deleted --expect-exit 1 -- diff fixtures/basic/identical.yaml scratch/empty.yaml`. The first command exits `1` with `# Create: 2 resources` in stdout; the second exits `1` with `# Delete: 2 resources` in stdout.
- **Control context lines.** Diff `test-base.yaml` and `test-head.yaml` with zero context lines and with ten context lines. Run `.agents/skills/verify/scripts/verify-cli run --artifact diff-manifests/context-0 --expect-exit 1 -- diff --context 0 fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml` and `.agents/skills/verify/scripts/verify-cli run --artifact diff-manifests/context-10 --expect-exit 1 -- diff --context 10 fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml`. Both exit `1`, and `wc -l` on `context-0.stdout.txt` is strictly less than `wc -l` on `context-10.stdout.txt`.
- **Missing file error.** Diff a nonexistent base file against `identical.yaml`. Run `.agents/skills/verify/scripts/verify-cli run --artifact diff-manifests/missing-file --expect-exit 2 -- diff fixtures/basic/nonexistent.yaml fixtures/basic/identical.yaml`. Exit code is `2` and stderr contains `failed to open base file`.

## Gotchas

- In unified diff hunks, `--- <name>-live.yaml` renders the `head` manifest and `+++ <name>.yaml` renders the `base` manifest, so lines starting with `-` come from `head` and lines starting with `+` come from `base`.
- Multi-resource diffs iterate over a Go map; never assert a fixed relative ordering among `frontend-app`, `backend-app`, and `app-config` sections.
- `k8s-manifest-diff diff` exits with `1` whenever differences exist; in shell scripts with `set -e`, always drive through `verify-cli run --expect-exit 1` so an expected diff does not abort the script.
