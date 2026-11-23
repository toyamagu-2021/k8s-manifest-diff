# Filter resources by kind, label, and annotation

Filtering flags on `k8s-manifest-diff diff` and `k8s-manifest-diff parse` let a user exclude specific Kubernetes resource kinds (`--exclude-kinds`) or include only resources matching exact label (`--label`) and annotation (`--annotation`) key-value pairs.

## Sub-features

- `filter-exclude-kinds` omits specified resource kinds (comma-separated or repeated flags) from comparison.
- `filter-label` includes only resources matching all specified `--label key=value` selectors (AND logic).
- `filter-annotation` includes only resources matching all specified `--annotation key=value` selectors (AND logic).
- `filter-combined` combines `--exclude-kinds`, `--label`, and `--annotation` in a single invocation, returning exit code `0` (`No differences found`) when all changed resources are filtered out.

## How to get to it (user POV)

- Pass `--exclude-kinds Kind1,Kind2` (or repeated `--exclude-kinds Kind`) to `k8s-manifest-diff diff` or `k8s-manifest-diff parse`.
- Pass one or more `--label key=value` flags to `k8s-manifest-diff diff` or `k8s-manifest-diff parse`.
- Pass one or more `--annotation key=value` flags to `k8s-manifest-diff diff` or `k8s-manifest-diff parse`.

## Driving it with verify-cli

Preconditions:

- `verify-cli doctor` prints `DOCTOR OK` for the current `RUN_ID`.
- Baseline fixtures `fixtures/kinds/mixed-base.yaml`, `fixtures/kinds/mixed-head.yaml`, `fixtures/basic/test-base.yaml`, `fixtures/basic/test-head.yaml`, `fixtures/selectors/annotation-test-base.yaml`, and `fixtures/selectors/annotation-test-head.yaml` exist under `/tmp/k8s-manifest-diff-verify-$RUN_ID/`.

- **Exclude kinds.** Diff `mixed-base.yaml` and `mixed-head.yaml` while excluding `Deployment` and `Service`. Run `.agents/skills/verify/scripts/verify-cli run --artifact filter-resources/exclude-kinds --expect-exit 1 -- diff --exclude-kinds Deployment,Service fixtures/kinds/mixed-base.yaml fixtures/kinds/mixed-head.yaml`. Exit code is `1`; stdout contains `argoproj.io/Workflow /test-workflow` and does not contain `apps/Deployment` or `/Service`.
- **Filter by multiple labels (AND logic).** Diff `test-base.yaml` and `test-head.yaml` with `--label app=nginx` and `--label tier=frontend`. Run `.agents/skills/verify/scripts/verify-cli run --artifact filter-resources/label-and --expect-exit 1 -- diff --label app=nginx --label tier=frontend fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml`. Exit code is `1`; stdout contains `frontend-app` and `app-config`, and does not contain `backend-app`.
- **Filter by multiple annotations (AND logic).** Diff `annotation-test-base.yaml` and `annotation-test-head.yaml` with `--annotation app.kubernetes.io/managed-by=helm` and `--annotation deployment.category=web`. Run `.agents/skills/verify/scripts/verify-cli run --artifact filter-resources/annotation-and --expect-exit 1 -- diff --annotation app.kubernetes.io/managed-by=helm --annotation deployment.category=web fixtures/selectors/annotation-test-base.yaml fixtures/selectors/annotation-test-head.yaml`. Exit code is `1`; stdout contains `frontend-app` and does not contain `backend-app`, `app-config`, or `db-secret`.
- **Combine label, annotation, and kind exclusion.** Diff `annotation-test-base.yaml` and `annotation-test-head.yaml` with `--label tier=frontend`, `--annotation app.kubernetes.io/managed-by=helm`, and `--exclude-kinds Deployment`. Run `.agents/skills/verify/scripts/verify-cli run --artifact filter-resources/combined --expect-exit 1 -- diff --label tier=frontend --annotation app.kubernetes.io/managed-by=helm --exclude-kinds Deployment fixtures/selectors/annotation-test-base.yaml fixtures/selectors/annotation-test-head.yaml`. Exit code is `1`; stdout contains `app-config` and does not contain `frontend-app`, `backend-app`, or `db-secret`.
- **Filter out all resources.** Diff `test-base.yaml` and `test-head.yaml` with a non-matching label `--label nonexistent=value`. Run `.agents/skills/verify/scripts/verify-cli run --artifact filter-resources/no-match --expect-exit 0 -- diff --label nonexistent=value fixtures/basic/test-base.yaml fixtures/basic/test-head.yaml`. Exit code is `0` and stdout is `No differences found`.

## Gotchas

- `--exclude-kinds` matches `Kind` case-sensitively (`Deployment`, not `deployment` or `deployments`).
- A `--label` or `--annotation` value that does not contain `=` (for example `--label invalidlabel`) is silently ignored rather than raising a CLI error, so a typo that omits `=` runs an unfiltered diff.
- `--annotation` uses `StringSliceVar`, so an annotation value containing a comma will be split into multiple entries unless passed appropriately.
