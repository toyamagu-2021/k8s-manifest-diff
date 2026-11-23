# Format diff and summary output

Output formatting options on `k8s-manifest-diff diff` let a user switch between full unified diffs and resource-only summaries (`--summary`) in either plain text (`--output-format default`) or GitHub-flavored Markdown (`--output-format markdown`).

## Sub-features

- `format-default-summary` prints a plain-text summary grouped by change type (`Unchanged`, `Changed`, `Create`, `Delete`) without unified diff hunks when `--summary` is set.
- `format-markdown-diff` prints a Markdown report with `# Kubernetes Manifest Diff`, a `## Summary` counts block, per-kind resource lists, and fenced ```` ```diff ```` blocks under `## Resource Changes`.
- `format-markdown-summary` prints only the Markdown summary and resource lists without `## Resource Changes` when both `--summary` and `--output-format markdown` are set.
- `format-invalid` rejects unsupported `--output-format` values with exit code `2`.

## How to get to it (user POV)

- Run `k8s-manifest-diff diff --summary <base-file> <head-file>` for a plain-text change summary.
- Run `k8s-manifest-diff diff --output-format markdown <base-file> <head-file>` for a full Markdown diff report.
- Run `k8s-manifest-diff diff --summary --output-format markdown <base-file> <head-file>` for a Markdown summary without diff hunks.

## Driving it with verify-cli

Preconditions:

- `verify-cli doctor` prints `DOCTOR OK` for the current `RUN_ID`.
- Baseline fixtures `fixtures/basic/test-base.yaml`, `fixtures/basic/test-head.yaml`, and `fixtures/basic/identical.yaml` exist under `/tmp/k8s-manifest-diff-verify-$RUN_ID/`.

- **Plain-text summary.** Run `diff --summary` on `test-base.yaml` and `test-head.yaml`. Run `.agents/skills/verify/scripts/verify-cli run --artifact diff-output-formats/plain-summary --expect-exit 1 -- diff --summary fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml`. Exit code is `1`; stdout contains `# Summary: 3 total, 3 changed, 0 created, 0 deleted, 0 unchanged` and `Changed (3):` listing `Deployment/default/frontend-app`, `Deployment/default/backend-app`, and `ConfigMap/default/app-config`, and does not contain `--- frontend-app-live.yaml`.
- **Full Markdown diff.** Run `diff --output-format markdown` on `test-base.yaml` and `test-head.yaml`. Run `.agents/skills/verify/scripts/verify-cli run --artifact diff-output-formats/markdown-diff --expect-exit 1 -- diff --output-format markdown fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml`. Exit code is `1`; stdout contains `# Kubernetes Manifest Diff`, `**Total Resources**: 3`, `**Changed**: 3 | **Created**: 0 | **Deleted**: 0 | **Unchanged**: 0`, `## Changed Resources (3)`, `## Resource Changes`, `### apps/Deployment default/frontend-app`, and fenced ```` ```diff ```` blocks.
- **Markdown summary only.** Run `diff --summary --output-format markdown` on `test-base.yaml` and `test-head.yaml`. Run `.agents/skills/verify/scripts/verify-cli run --artifact diff-output-formats/markdown-summary --expect-exit 1 -- diff --summary --output-format markdown fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml`. Exit code is `1`; stdout contains `# Kubernetes Manifest Diff` and `## Changed Resources (3)`, and does not contain `## Resource Changes` or ```` ```diff ````.
- **Reject invalid output format.** Pass `--output-format json`. Run `.agents/skills/verify/scripts/verify-cli run --artifact diff-output-formats/invalid-format --expect-exit 2 -- diff --output-format json fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml`. Exit code is `2` and stderr contains `invalid output format: json (supported formats: default, markdown)`.

## Gotchas

- When two files have no differences, `diff --summary` and `diff --output-format markdown` still print `No differences found` (plain text) and exit `0` rather than a Markdown header.
- Core group resources (such as `ConfigMap` and `Secret`) have an empty API group, so their Markdown diff section heading has a leading slash: `### /ConfigMap default/app-config` and `### /Secret default/db-secret`.
