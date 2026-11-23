# Mask Secret data in diffs

Secret masking in `k8s-manifest-diff diff` replaces Kubernetes `Secret` values in both `data` and `stringData` with `+`-character masks by default so diffs show which keys changed without leaking credentials, while allowing raw value diffs via `--disable-masking-secret` and rejecting malformed Secrets.

## Sub-features

- `mask-diff-default` masks `data` (base64) and `stringData` (plaintext) values with `++++++++++++++++` tokens by default during `diff`.
- `mask-diff-disable` reveals raw `data` and `stringData` values in the unified diff when `--disable-masking-secret` is passed.
- `mask-diff-invalid-secret` rejects malformed `Secret` manifests (such as lists or nested maps inside `data`) with exit code `2` to prevent unmasked secret leakage.

## How to get to it (user POV)

- Run `k8s-manifest-diff diff <base-secret.yaml> <head-secret.yaml>` to compare Secrets with default masking enabled.
- Run `k8s-manifest-diff diff --disable-masking-secret <base-secret.yaml> <head-secret.yaml>` to compare Secrets with raw values shown.

## Driving it with verify-cli

Preconditions:

- `verify-cli doctor` prints `DOCTOR OK` for the current `RUN_ID`.
- Baseline fixtures `fixtures/basic/secret-mixed-base.yaml`, `fixtures/basic/secret-mixed-head.yaml`, and `fixtures/basic/secret-invalid-nested-map.yaml` exist under `/tmp/k8s-manifest-diff-verify-$RUN_ID/`.

- **Default masking of data and stringData.** Diff `secret-mixed-base.yaml` and `secret-mixed-head.yaml` without extra flags. Run `.agents/skills/verify/scripts/verify-cli run --artifact secret-masking/masked-default --expect-exit 1 -- diff fixtures/basic/secret-mixed-base.yaml fixtures/basic/secret-mixed-head.yaml`. Exit code is `1`; stdout contains `===== /Secret default/mixed-secret ======`, `data:`, `stringData:`, and `++++++++++++++++`, and does not contain raw values `bXlwYXNzd29yZA==`, `bmV3cGFzc3dvcmQ=`, `plaintext-api-key`, or `new-plaintext-key`.
- **Disable secret masking.** Diff `secret-mixed-base.yaml` and `secret-mixed-head.yaml` with `--disable-masking-secret`. Run `.agents/skills/verify/scripts/verify-cli run --artifact secret-masking/unmasked --expect-exit 1 -- diff --disable-masking-secret fixtures/basic/secret-mixed-base.yaml fixtures/basic/secret-mixed-head.yaml`. Exit code is `1`; stdout contains `bXlwYXNzd29yZA==`, `bmV3cGFzc3dvcmQ=`, `plaintext-api-key`, and `new-plaintext-key`, and does not contain `++++++++++++++++`.
- **Reject malformed Secret structure.** Diff `secret-invalid-nested-map.yaml` against `secret-mixed-head.yaml`. Run `.agents/skills/verify/scripts/verify-cli run --artifact secret-masking/invalid-secret --expect-exit 2 -- diff fixtures/basic/secret-invalid-nested-map.yaml fixtures/basic/secret-mixed-head.yaml`. Exit code is `2` and stderr contains `secret validation failed`.

## Gotchas

- Identical secret values map to the same `+`-string replacement (`++++++++++++++++`) within a process run, while each new distinct value receives a mask one `+` longer (`+++++++++++++++++`, `++++++++++++++++++`, etc.), so two keys with identical values remain equal after masking.
- Non-base64 strings in `Secret.data` (such as SOPS `ENC[...]` strings) are allowed and masked as strings, whereas non-string YAML values (maps, arrays, numbers) in `data` or `stringData` fail validation with exit code `2` unless `--disable-masking-secret` is set.
