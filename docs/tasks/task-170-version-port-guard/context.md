# task-170 — Implementation Context

Orienting facts for an engineer with zero prior context. Read this before `plan.md`.

## What & why

Close a CI blind spot: a client version can have full protocol support (a socket
config template + coverage-matrix columns) while having **no LoadBalancer ports**,
and nothing fails. PR #971 shipped gms v48/61/72/79 that way. This task adds a
guard that fails when the template set and `versions.json` diverge, and wires the
four missing versions' ports. Full rationale: `design.md` in this folder.

## Repo layout (all paths relative to repo root)

- `deploy/k8s/base/versions.json` — declared version set; single source of truth
  for LB ports. Array of `{ "region", "majorVersion", "minorVersion" }`.
- `deploy/k8s/base/atlas-login.yaml`, `deploy/k8s/base/atlas-channel.yaml` —
  contain marker-delimited generated port blocks
  (`# BEGIN generated:container-ports … # END …`, likewise `service-ports`).
- `tools/gen-lb-ports.sh` — regenerates those blocks from `versions.json`.
  `--check` diffs regen vs checked-in and exits 1 on drift (the existing CI guard).
- `tools/gen-lb-ports_test.sh` — hermetic shell test; **mirror this harness** for
  the new guard's test (throwaway git repo in `$(mktemp -d)`, `git init`, copy the
  script under test, build fixtures, assert exit code + message).
- `services/atlas-pr-bootstrap/scripts/version-ports.sh` — the port formula:
  `derive_login_port(major) = major*100`, `derive_channel_port(major) = login+1`.
- `services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`
  — one socket config template per supported version. This filename set is the
  guard's authoritative "version exists" signal. Current files:
  `template_{gms_12_1, gms_48_1, gms_61_1, gms_72_1, gms_79_1, gms_83_1, gms_84_1,
  gms_87_1, gms_92_1, gms_95_1, jms_185_1}.json`.
- `.github/workflows/pr-validation.yml` — the `gen-lb-ports` job (job name
  displayed as "LB Port Drift Guard") runs `./tools/gen-lb-ports.sh --check`. The
  new guard is added as a **second step in this same job** (pure shell+jq, no
  second runner). The summary row `LBPORTS_RESULT` already reports the job result;
  no summary/`needs` changes needed.

## Key decisions (from design.md, already approved)

- **Comparison key = `(region, majorVersion)`**, NOT the full triple. Ports are
  major-only and `gen-lb-ports.sh` already bans duplicate majors. Two minors of
  one major (`template_gms_83_1` + `template_gms_83_2`) collapse to one `gms 83`
  and must NOT produce a false mismatch.
- **Strict, bidirectional set-equality.** Fail if a template has no
  `versions.json` entry (the #971 case) AND fail if a `versions.json` entry has no
  template (phantom ports).
- **No `deferred`/allowlist escape hatch** — the simplest model; ports must land
  with templates.

## Ports being added (part B)

| Version | login port | channel port |
|---|:-:|:-:|
| gms 48 | 4800 | 4801 |
| gms 61 | 6100 | 6101 |
| gms 72 | 7200 | 7201 |
| gms 79 | 7900 | 7901 |

No collision with existing majors (12/83/84/87/92/95/185).

## Verification commands (run from repo root)

```bash
tools/gen-lb-ports.sh --check          # LB yaml matches versions.json
tools/check-version-coverage.sh        # template set == versions.json set
tools/gen-lb-ports_test.sh             # existing guard tests still pass
tools/check-version-coverage_test.sh   # new guard tests
shellcheck tools/check-version-coverage.sh tools/check-version-coverage_test.sh
```

## Conventions

- Both new scripts: `#!/usr/bin/env bash`, `set -euo pipefail`, resolve repo root
  via `git rev-parse --show-toplevel`, `chmod +x`, shellcheck-clean.
- This is a task worktree (`.worktrees/task-170-version-port-guard`, branch
  `task-170-version-port-guard`). Never edit the main checkout. Verify branch
  after each commit.
