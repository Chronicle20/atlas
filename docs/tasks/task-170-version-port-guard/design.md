# task-170: Version-Port Coverage Guard + wire legacy LB ports

Status: Approved (design)
Created: 2026-07-13
Branch/worktree: `task-170-version-port-guard`

## 1. Problem

PR #971 ("GMS Legacy Versions 48.1/61.1/72.1/79.1") shipped the protocol
deliverable for four pre-v83 GMS versions — codecs, seed socket-config
templates, and coverage-matrix columns — but **no LoadBalancer socket ports**.
The runtime bring-up (task-113 Stages G/H/I: WZ ingest, k8s ports + tenant
provisioning, playthrough) was a deliberate, owner-approved deferral, recorded
in `docs/tasks/task-113-gms-legacy-versions/audit-wholebranch.md`.

The existing **LB Port Drift Guard** (`tools/gen-lb-ports.sh --check`, CI job
`gen-lb-ports` in `.github/workflows/pr-validation.yml`) did not flag this. It
is structurally unable to: its only input is `deploy/k8s/base/versions.json`,
and it checks exactly one invariant —

```
versions.json  ──gen-lb-ports.sh──▶  atlas-{login,channel}.yaml port blocks
     ▲ (source of truth, unvalidated)          ▲ (guard diffs THIS vs regen)
```

A version that never lands in `versions.json` produces no drift, so the guard
stays green. The deferral was therefore **silent**: nothing in CI relates "this
version has protocol support / a socket template" to "this version has ports."

This task (1) closes that blind spot with a new coverage guard, and (2) wires
the LB ports for the four legacy versions so `versions.json` reflects reality.

## 2. Goals

- A CI guard that fails loudly when the set of versions that have a socket
  config template diverges from the set of versions declared in
  `versions.json` — in either direction.
- LB container/service ports wired for gms 48.1 / 61.1 / 72.1 / 79.1 via the
  canonical `versions.json` → `gen-lb-ports.sh` path.
- Hermetic regression tests for the new guard.

## 3. Non-goals (deferred Stage G/H/I — unchanged from task-113)

- WZ game-data ingestion for the four legacy versions.
- Tenant provisioning / the login-channel **services config** port entries
  (these are formula-derived at bootstrap time — see §4.1 — not a static
  artifact, so there is nothing to edit here).
- Real-client end-to-end playthrough.

Exposing the containerPorts now is forward-compatible: when a legacy tenant is
later provisioned, `bootstrap.sh` derives its services-config port from the
same formula, and the k8s port is already open.

## 4. Background findings

### 4.1 The services-config port half of FR-9.1 is not a static file

task-113 FR-9.1 requires ports to agree in "both places": the static k8s yaml
**and** the login/channel `services` config. Investigation shows the second
place is **not** a checked-in artifact. `services/atlas-pr-bootstrap/scripts/bootstrap.sh`
builds each tenant's login/channel service entry fresh at provisioning time via
`build_login_entry` / `build_channel_entry`, deriving the port from the
tenant's `majorVersion` through the same shared formula in
`services/atlas-pr-bootstrap/scripts/version-ports.sh` (`derive_login_port` =
`major×100`, `derive_channel_port` = login+1). The canonical
`services/login-service.json` `tenants[]` is rebuilt, not string-substituted.

Consequence: the two port sites are two consumers of one formula and cannot
diverge by construction. The only provisioning-free static wiring is
`versions.json` → LB yaml. The services-config side belongs to the (deferred)
provisioning step.

### 4.2 Version-set asymmetry — "audit column" is the wrong signal

Four sets, keyed by region+major:

| Version | versions.json (LB ports) | audit column | seed game-data | socket template |
|---|:-:|:-:|:-:|:-:|
| gms 12 | ✅ | ❌ | ✅ | ✅ |
| gms 48/61/72/79 | ❌ | ✅ | ❌ | ✅ |
| gms 83/84/87/95 | ✅ | ✅ | ✅ | ✅ |
| gms 92 | ✅ | ❌ | ✅ | ✅ |
| jms 185 | ✅ | ✅ | ✅ | ✅ |

The **socket config template** set is the superset — every version with any
support has exactly one `template_<region>_<major>_<minor>.json`. "Has an audit
column" is *not* a valid deployability signal (v12 and v92 are deployed with no
audit dir). The template set is therefore the authoritative "this version is a
real, provisionable version" signal for the guard.

After Part B wires the four legacy versions, `versions.json` will equal the
template set exactly:
`{gms 12, 48, 61, 72, 79, 83, 84, 87, 92, 95, jms 185}`.

## 5. Design

### 5.1 Part A — `tools/check-version-coverage.sh` (the guard)

Pure shell + jq, a sibling of `gen-lb-ports.sh`, resolving the repo root via
`git rev-parse --show-toplevel`. It enforces **strict set-equality** between two
sets, each keyed on `(region, majorVersion)`:

- **Template set** — parsed from the filenames of
  `services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`.
- **Version set** — `.versions[]` of `deploy/k8s/base/versions.json`
  (`(region, majorVersion)`).

Rationale for keying on `(region, major)` and not the full triple: the port
formula is major-only, and `gen-lb-ports.sh` already rejects duplicate majors
in `versions.json` (same major → same port → LB collision). Minor is
informational; keying on the triple would let a hypothetical second minor of an
already-ported major (`template_gms_83_2.json`) register as a false mismatch.

Failure modes (each exits non-zero and names the offenders + the fix):

- **Template without version** → e.g. `gms 48 has a socket config template but
  no deploy/k8s/base/versions.json entry — add it and run
  tools/gen-lb-ports.sh, or remove the template.` This is the case that would
  have caught PR #971.
- **Version without template** → e.g. `gms 92 has a versions.json entry (LB
  ports) but no socket config template.`

No generation, no `--check` flag — the script is always a check. Exit 0 when the
sets are equal.

### 5.2 Part B — wire the four legacy versions

Append to `deploy/k8s/base/versions.json` `.versions[]`:

```json
{ "region": "gms", "majorVersion": 48, "minorVersion": 1 },
{ "region": "gms", "majorVersion": 61, "minorVersion": 1 },
{ "region": "gms", "majorVersion": 72, "minorVersion": 1 },
{ "region": "gms", "majorVersion": 79, "minorVersion": 1 }
```

Then run `tools/gen-lb-ports.sh` (no `--check`) to rewrite the marker blocks in
`deploy/k8s/base/atlas-login.yaml` and `atlas-channel.yaml`. Derived ports:

| Version | login (containerPort/Service) | channel |
|---|:-:|:-:|
| gms 48 | 4800 | 4801 |
| gms 61 | 6100 | 6101 |
| gms 72 | 7200 | 7201 |
| gms 79 | 7900 | 7901 |

No collision with existing majors (12/83/84/87/92/95/185) or their ports. The
existing dup-major check in `gen-lb-ports.sh` continues to hold.

### 5.3 CI wiring

Add a second step to the existing `gen-lb-ports` job in
`.github/workflows/pr-validation.yml` (both are pure shell+jq "version wiring is
consistent" gates; no second runner):

```yaml
      - name: Version coverage matches templates
        run: ./tools/check-version-coverage.sh
```

If either step fails the job fails; the existing `LB Port Drift Guard` summary
row (`LBPORTS_RESULT`) already reports the job result, and the aggregation/
`needs` wiring is unchanged.

## 6. Testing

`tools/check-version-coverage_test.sh`, following the hermetic pattern of
`tools/gen-lb-ports_test.sh`: build a throwaway git repo with fixture template
files + a fixture `versions.json`, invoke the script, assert exit code and
message. Cases:

1. In-sync sets → exit 0.
2. Template without a `versions.json` entry → exit non-zero, message names the
   missing version.
3. `versions.json` entry without a template → exit non-zero, message names it.
4. Two minors of one major with a single `versions.json` entry for that major →
   exit 0 (no false mismatch from the `(region, major)` key).

## 7. Verification

- `tools/check-version-coverage_test.sh` passes.
- `tools/gen-lb-ports_test.sh` still passes.
- `tools/gen-lb-ports.sh --check` clean after regenerating with the four new
  rows.
- `tools/check-version-coverage.sh` clean on the branch (sets now equal).
- shellcheck clean on both new files (matching repo convention for the existing
  tools scripts).

## 8. Files touched

- **new** `tools/check-version-coverage.sh`
- **new** `tools/check-version-coverage_test.sh`
- `deploy/k8s/base/versions.json` (+4 rows)
- `deploy/k8s/base/atlas-login.yaml` (regenerated marker blocks)
- `deploy/k8s/base/atlas-channel.yaml` (regenerated marker blocks)
- `.github/workflows/pr-validation.yml` (+1 step in the `gen-lb-ports` job)
