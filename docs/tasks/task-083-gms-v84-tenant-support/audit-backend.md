# Backend Audit — task-083-gms-v84-tenant-support

- **Scope:** GO changes in diff range `6a8b383d9..fb6d44f0f`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-09
- **Build:** PASS (all four affected modules)
- **Tests:** PASS (all four affected modules; no failures)
- **Vet:** PASS (tenant, packet, character, account)
- **Overall:** PASS

This is a focused, diff-scoped audit. The change is a library-level addition
(version-predicate helpers) plus ~45 mechanical boundary corrections in
`libs/atlas-packet`, two single-line service-processor predicate corrections,
and accompanying tests. No new domain packages, REST handlers, providers,
administrators, Kafka producers, external HTTP clients, or scaffolding were
introduced — so the bulk of the DOM-*/SUB-*/EXT-*/SCAFFOLD-*/SEC-* checklist is
**N/A**. Findings below cover the checks that actually apply plus the five
task-specific focus areas.

## Build & Test Results

| Module | build ./... | test ./... | vet ./... |
|--------|-------------|------------|-----------|
| `libs/atlas-tenant` | PASS | `ok` | PASS |
| `libs/atlas-packet` | PASS | `ok` (no failures) | PASS |
| `services/atlas-character/atlas.com/character` | PASS | `ok atlas-character/character 1.799s` | PASS |
| `services/atlas-account/atlas.com/account` | PASS | `ok atlas-account/account 0.857s` | PASS |
| `services/atlas-configurations/atlas.com/configurations` | PASS | `ok atlas-configurations/seeder 0.010s` | — |

`tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_84_1.json` → `OK: all template symbols resolve` (exit 0).

## Focus-Area Results

### 1. DOM-21 — Shared types / no duplication — PASS

- The new helpers `IsRegion`, `MajorAtLeast`, `MajorAtMost`, `MajorInRange` live
  on `tenant.Model` in `libs/atlas-tenant/tenant.go:88-105`. They operate on
  `tenant.Model`'s private `region`/`majorVersion` fields, which only this
  package can reach — correct home.
- `libs/atlas-constants/` contains no region/version predicate or `tenant.Model`
  equivalent (grep found zero matches; README has no version/tenant index entry).
  No duplication.
- All hand-rolled boundaries in the changed production scope were migrated to the
  helpers: grep for `MajorVersion() > 83 | == 83 | <= 83 | >= 84` across the
  changed `libs/atlas-packet`, `services/atlas-character`, `services/atlas-account`
  production (non-test) Go returns **zero** matches.

### 2. Immutability / receiver consistency — PASS (with one cosmetic note)

- The four helpers use **pointer receivers** (`func (m *Model) ...`) while the
  existing getters also use pointer receivers (`tenant.go:17-31`) — consistent.
- All call sites compile against pointer receivers because the receiver is an
  addressable local/field: `usesChooseGender(p.t)` /
  `appliesAutoAP(p.t)` take `t tenant.Model` by value (addressable parameter),
  and the test helper `mv()` / `tenant.Create()` returns an addressable `Model`.
  Verified by clean `go build`/`go vet` on all modules.
- COSMETIC (Minor): `tenant.Model` mixes pointer-receiver methods with a
  value-typed model passed around by value everywhere (`tenant.Create` returns
  `Model`, helpers take `tenant.Model` by value). This is the pre-existing
  convention, not introduced here, and does not break — noted only for awareness.

### 3. Behavior preservation (v83 / v87+ unchanged; flip only v84..86) — PASS

Algebraic equivalence of every migrated packet boundary:

- `>83` (i.e. `>=84`) → `MajorAtLeast(87)` (i.e. `>=87`): identical for
  v∈{83→false/false, 87→true/true, 95→true/true}; differs ONLY for v84/85/86
  (old true → new false). Correct off-by-one fix.
- `<=83` → `!MajorAtLeast(87)` (i.e. `<87`): identical for v∈{83→true, 87→false,
  95→false}; flips ONLY v84/85/86 (old false → new true). Complementary to the
  `>83` branch at the same boundary (87), so paired encode/decode partition with
  no overlap or gap — e.g. `character/serverbound/create.go` `>83`+`<=83` pair.
- Movement `>87` → `MajorAtLeast(88)`: identical for all of {83,84,85,86,87,95};
  unchanged.

The accompanying `version_bounds_test.go` files assert **real packet bytes**, not
tautologies:
- `model/version_bounds_test.go:28-37` asserts v84..87 encode `bytes.Equal` v83
  and v95 differs, then round-trips a v84 buffer and checks the post-XOffset
  fields (`BMoveAction`/`TElapse`) survive — i.e. it would catch a stale `>83`
  over-read.
- `field/clientbound/version_bounds_test.go:44-56` asserts v84/85/86 byte-equal
  v83 and v87/v95 diverge, exercising the full SetField/WarpToMap body.
- `character/serverbound/version_bounds_test.go`, `chat/serverbound/...`, and the
  updated `spawn.go` length test (`90d703ff5` — length, not time-fragile
  byte-equality, due to embedded timestamp) follow the same real-assertion shape.

NOTE on the two **service-level** predicate changes (these are intentional
semantic widenings, NOT behavior-preserving renames, per plan.md B3):
- `appliesAutoAP` (`atlas-character/.../processor.go`): old `MajorVersion()==83`
  → new `IsRegion("GMS") && MajorAtMost(94)`. This **does** change v87 (and all
  GMS 28..94) from false→true. That is the documented intent (plan.md B3:
  "auto-AP is a pre-Big-Bang behavior → `MajorAtMost(94)`"; context.md flags the
  exact `==83` as "the one unambiguous bug"). Test
  `processor_bounds_test.go:19-24` pins {83→true, 84→true, 94→true, 95→false,
  JMS→false}. ACCEPTED as designed.
- `usesChooseGender` (`atlas-account/.../processor.go`): old `>83` → new
  `IsRegion("GMS") && MajorAtLeast(87)`. v83 false (unchanged), v87+ true
  (unchanged), v84..86 flip true→false (the fix). Behavior-preserving for
  v83/v87+. Test `processor_bounds_test.go:16-21` pins it.

### 4. Encode/decode pairing for `model/movement.go` — PASS (fixes a pre-existing bug)

- Before: `Decode` gated `>83` (movement.go), `Encode` gated `>87` — **mismatched**
  (a latent packet-corruption bug for GMS v84..v87).
- After: both `Decode` and `Encode` gate `!t.IsRegion("GMS") || t.MajorAtLeast(88)`
  — textually identical (`movement.go` Decode and Encode blocks). The
  round-trip test at `model/version_bounds_test.go:41-52` proves a v84 buffer
  round-trips with zero leftover bytes and intact trailing fields.
- Same pairing verified for `character/clientbound/info.go:113-189` (chair int,
  encode+decode both `MajorAtLeast(87)`) and the monster movement files.

### 5. Test quality — `tenant.Create` used, no `*_testhelpers.go` — PASS

- `libs/atlas-tenant/tenant_test.go:37-40` `mv()` uses `tenant.Create(...)`.
- `atlas-character/.../processor_bounds_test.go:26` and
  `atlas-account/.../processor_bounds_test.go:24` use `tenant.Create(...)`.
- `libs/atlas-packet/test/context.go:27` `CreateContext` uses `tenant.Create(...)`.
- No `*_testhelpers.go` files added (grep clean across the diff). Tests are
  table-driven (`processor_bounds_test.go`, `tenant_test.go`, `seeder_test.go`)
  per testing-guide.md.

## Applicable Standard-Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-20 | Table-driven tests | PASS | `tenant_test.go`, `processor_bounds_test.go` (both services), `seeder_test.go` use `[]struct{...}` + loop / `t.Run` |
| DOM-21 | No duplication of atlas-constants types | PASS | Helpers live in `atlas-tenant/tenant.go:88-105`; no version predicate in `libs/atlas-constants`; all call sites migrated to helpers |
| DOM-24 | Kafka producer stubbed in emitting tests | N/A | No test in scope exercises an emit path. `usesChooseGender`/`appliesAutoAP` tests call pure helpers; packet tests call `Encode`/`Decode` only. `ProcessorImpl.Create` (account) is not invoked by any in-scope test. |
| SEC-04 | No hardcoded secrets | PASS | No secrets in changed files (predicate helpers + boundary literals only) |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix / note)
- (Minor, cosmetic) `appliesAutoAP` doc comment says "(..94 era)" while plan.md
  B3 wrote "(28..94 era)". No functional impact.
- (Minor, informational) The auto-AP widening changes GMS v87 behavior
  (false→true) — this is the documented design intent (plan B3), but it is a
  broader semantic change than the v84-only off-by-one corrections; flagged so a
  reviewer consciously confirms the pre-Big-Bang `MajorAtMost(94)` range is the
  intended one. Test coverage for the 28..82 / 85..93 interior of that range is
  absent (only 83/84/94/95 are pinned), but the boundary cases that matter
  (lower-edge behavior preservation at 83, upper edge at 94/95) are covered.

### Verdict per focus area
1. DOM-21 shared types: PASS
2. Immutability / receiver consistency: PASS
3. Behavior preservation: PASS
4. Encode/decode pairing (movement.go): PASS (fixes a pre-existing mismatch)
5. Test quality (`tenant.Create`, no testhelpers): PASS
