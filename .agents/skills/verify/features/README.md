# k8s-manifest-diff verification map

This directory is the maintained source for verifying the user-facing behavior of `k8s-manifest-diff`. Read this index before driving the CLI, then follow the matching feature file as the recipe.

## Baseline preconditions

- Pick a unique `RUN_ID` (`export RUN_ID="verify-$(date +%s)-$$"`) so concurrent verification runs do not share state.
- Run `.agents/skills/verify/scripts/verify-cli launch` to compile `./cmd/k8s-manifest-diff` into `/tmp/k8s-manifest-diff-verify-$RUN_ID/bin/k8s-manifest-diff` and copy `testing/e2e/fixtures/` into `/tmp/k8s-manifest-diff-verify-$RUN_ID/fixtures/`.
- Run `.agents/skills/verify/scripts/verify-cli doctor` and require `DOCTOR OK` with the expected git `HEAD` commit and binary checksum.
- Write any scratch YAML files needed for a journey under `/tmp/k8s-manifest-diff-verify-$RUN_ID/scratch/`, never inside the git worktree.
- Never reuse a `RUN_ID` started by another agent or session.

## Driving conventions

- Start every recipe from the baseline state unless its preconditions say otherwise.
- Treat every command as literal. Keep quoted flags, selector strings, and file paths unchanged.
- Run all CLI actions through `.agents/skills/verify/scripts/verify-cli run --artifact <feature>/<step> [--expect-exit <code>] -- <args...>`.
- Relative paths passed to `verify-cli run` resolve from `/tmp/k8s-manifest-diff-verify-$RUN_ID`.
- Run `.agents/skills/verify/scripts/verify-cli cleanup` after finishing verification or after any failed run. Do not remove proof artifacts under `/tmp/k8s-manifest-diff-proof-$RUN_ID`.

## Proof and skip reporting

- Capture every CLI action and its resulting output (`<step>.command.txt`, `<step>.stdout.txt`, `<step>.stderr.txt`, `<step>.exit_code.txt`) under `/tmp/k8s-manifest-diff-proof-$RUN_ID/<feature>/`.
- Assert both exit code (`0` for no diff or parse success, `1` for diff found, `2` for error) and concrete stdout/stderr content.
- Multi-resource output iterates over a Go `map`, so assert the presence of each resource header or summary entry rather than assuming a fixed ordering across resources.
- Record the feature ID and sub-feature IDs verified with their proof artifact paths.
- Report an unreachable path with the attempted command and the unmet precondition.
- Do not report a skipped entry point as verified through a different path.

## Feature entry contract

Each feature file starts with an H1 title and one paragraph describing the user-visible behavior. It then uses exactly four H2 sections in this order.

1. `Sub-features` lists short IDs with one line for each behavior.
2. `How to get to it (user POV)` lists every user entry point.
3. `Driving it with verify-cli` starts with `Preconditions:` and uses labeled bullets that pair each user action with an exact command and observable result.
4. `Gotchas` lists traps that can waste or invalidate a verification run.

Keep implementation details out of the map. Name only user paths, CLI flags, required fixtures, commands, and observable proof.

## Features

- [Compare manifests with diff](./diff-manifests.md) covers comparing identical and changed manifests, created and deleted resources, context lines, and error exit codes.
- [Format diff and summary output](./diff-output-formats.md) covers `--summary` and `--output-format` (`default` and `markdown`) combinations.
- [Filter resources by kind, label, and annotation](./filter-resources.md) covers `--exclude-kinds`, `--label`, and `--annotation` across single and combined filters.
- [Mask Secret data in diffs](./secret-masking.md) covers default masking of `data` and `stringData` fields, consistent replacement tokens, and `--disable-masking-secret`.
- [Parse and mask standalone manifests](./parse-manifests.md) covers `k8s-manifest-diff parse` across single and multiple files with filtering and secret masking.
