# Parse and mask standalone manifests

`k8s-manifest-diff parse` parses one or more Kubernetes YAML/JSON manifest files, applies optional kind, label, and annotation filters, masks `Secret` `data` and `stringData` values by default, and prints normalized YAML documents with a `# Resources (<N>)` header comment.

## Sub-features

- `parse-single-masked` parses a manifest file, masks Secret values by default, and outputs a `# Resources (<N>)` header followed by YAML documents.
- `parse-multi-file` processes multiple manifest files in order, adding `# File: <path>` headers and `---` separators between files.
- `parse-filtered` filters parsed resources using `--exclude-kinds`, `--label`, and `--annotation`.
- `parse-unmasked` preserves raw Secret `data` and `stringData` values when `--disable-masking-secret` is set.
- `parse-error` returns exit code `2` when no files are provided, a file is missing, or Secret validation fails.

## How to get to it (user POV)

- Run `k8s-manifest-diff parse <file1> [file2...]` in a terminal.
- Pass `--exclude-kinds`, `--label`, `--annotation`, or `--disable-masking-secret` to `k8s-manifest-diff parse`.

## Driving it with verify-cli

Preconditions:

- `verify-cli doctor` prints `DOCTOR OK` for the current `RUN_ID`.
- Baseline fixtures `fixtures/basic/secret-mixed-base.yaml`, `fixtures/selectors/annotation-test-base.yaml`, and `fixtures/basic/secret-invalid-list.yaml` exist under `/tmp/k8s-manifest-diff-verify-$RUN_ID/`.

- **Parse and mask a single Secret manifest.** Run `parse` on `secret-mixed-base.yaml`. Run `.agents/skills/verify/scripts/verify-cli run --artifact parse-manifests/single-masked --expect-exit 0 -- parse fixtures/basic/secret-mixed-base.yaml`. Exit code is `0`; stdout contains `# Resources (1)`, `# /Secret default/mixed-secret`, and `++++++++++++++++`, and does not contain `bXlwYXNzd29yZA==` or `plaintext-api-key`.
- **Parse with secret masking disabled.** Run `parse --disable-masking-secret` on `secret-mixed-base.yaml`. Run `.agents/skills/verify/scripts/verify-cli run --artifact parse-manifests/single-unmasked --expect-exit 0 -- parse --disable-masking-secret fixtures/basic/secret-mixed-base.yaml`. Exit code is `0`; stdout contains `bXlwYXNzd29yZA==` and `plaintext-api-key`, and does not contain `++++++++++++++++`.
- **Parse multiple files with filtering.** Run `parse` on `annotation-test-base.yaml` and `secret-mixed-base.yaml` with `--exclude-kinds Deployment`. Run `.agents/skills/verify/scripts/verify-cli run --artifact parse-manifests/multi-filtered --expect-exit 0 -- parse --exclude-kinds Deployment fixtures/selectors/annotation-test-base.yaml fixtures/basic/secret-mixed-base.yaml`. Exit code is `0`; stdout contains `# File: fixtures/selectors/annotation-test-base.yaml`, `# Resources (2)`, `# File: fixtures/basic/secret-mixed-base.yaml`, and `# Resources (1)`, and does not contain `frontend-app` or `backend-app`.
- **Reject invalid Secret in parse.** Run `parse` on `secret-invalid-list.yaml`. Run `.agents/skills/verify/scripts/verify-cli run --artifact parse-manifests/invalid-secret --expect-exit 2 -- parse fixtures/basic/secret-invalid-list.yaml`. Exit code is `2` and stderr contains `failed to process file`.

## Gotchas

- `k8s-manifest-diff parse` deduplicates resources within a single file by `ResourceKey` (`Group/Kind/Namespace/Name`) because parsed objects are stored in a `map[ResourceKey]*unstructured.Unstructured`, and multi-resource output order within a file is non-deterministic.
- When all resources in a file are filtered out, `parse` prints an empty string (exit code `0`) without a `# Resources (0)` header.
