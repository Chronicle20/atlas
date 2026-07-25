# Whole-Branch Audit — task-113-gms-legacy-versions

**Plan:** `docs/tasks/task-113-gms-legacy-versions/plan.md` (Phase 5 gate)
**Audit Date:** 2026-07-04
**Branch:** task-113-gms-legacy-versions (297 commits ahead of main)
**Base:** main
**Scope:** Phase-5 reconciliation gate. Runtime Stages G/H/I DEFERRED by owner (out of scope).

## Executive Summary

Overall **PASS** (one minor, non-blocking documentation finding). All four protocol passes (v79/v72/v61/v48) delivered Stages A–F. Build/vet/test/redis-guard/matrix all green. The critical FR-7.2 invariant holds: **no existing version (v83/84/87/95/jms) packet evaluation changed** — verified counts are frozen exactly and every legacy gate fires strictly below the existing-version range. The three intentional existing-version-range corrections and the Phase-5 NPC_TALK fix are each IDA-backed, fixture-consistent, and provably leave v83+ untouched. Sole finding: two of the three corrections are documented in the v48 audit reports but were not back-filled as rows in `code-gate-audit.md`.

## Bars

| Check | Result | Evidence |
|---|---|---|
| `go build ./...` | PASS | exit 0 |
| `go vet` (per changed module) | PASS | libs/atlas-packet, atlas-channel, tools/packet-audit clean (root `./...` errors only on workspace layout, not a real failure) |
| `go test ./libs/atlas-packet/... ./tools/packet-audit/...` | PASS | all `ok`, exit 0 |
| `redis-key-guard.sh` (GOWORK=off) | PASS | exit 0, no violation lines |
| `matrix --check` | PASS | exit 0; 0 problem lines in STATUS.md and check output; Conflicts: None |
| Guard tests (VersionKeys/shortLabels/fnamedocOrder/templateFiles) | PASS | matrix + cmd test packages `ok` |

Changed Go modules: `libs/atlas-packet` (75 non-test files), `services/atlas-channel` (3), `tools/packet-audit` (4). **No go.mod touched → no docker bake required** (consistent with ledger).

## 5.1 Code-gate reconciliation — MOSTLY COMPLETE

`code-gate-audit.md`: 610 lines, 214 file:line data rows, all four version columns (v79/v72/v61/v48) present, no empty version cells, no blank Correct?/Action. Spot-checks (source matches table):
- `buddy/clientbound/invite.go:51` → `Region()!="GMS" || MajorVersion()>=87` ✓
- `cash/clientbound/query_result.go:42` → `Region()=="GMS" && MajorVersion()>12` ✓
- `cash/clientbound/shop_open.go:45` → `MajorVersion()<=12` ✓
- `messenger/clientbound/add.go:51` → `IsRegion("GMS") && MajorVersion()<=28` ✓ (row 491, CORRECTED, IDA sub_6D144E)
- `model/buddy.go:36` → `Region()!="GMS" || MajorVersion()>=72` ✓ (row 490)

**Finding (minor):** the buddy `operation_add.go` group>61 correction and the reactor `hit.go` isSkill>=72/skillId>=79 correction are **not rows** in `code-gate-audit.md`. They are documented in `audit-v48-backend.md` and `v48-stageE-batch1.md` and carry full IDA comments + fixtures in code, but FR-7 traceability in the reconciliation table is incomplete for these two. Correctness sound; table completeness is the gap.

## FR-7.2 — existing-version corrections (each IDA-backed + fixture-consistent + v83+ unchanged)

1. **messenger/clientbound/add.go `<=28`** — YES. On main the channelId+pad were unconditional; now gated `!legacyAdd` (legacy = GMS<=28). v83+ (>28) still writes → unchanged. IDA sub_6D144E (v61 reads 6 fields). Fixtures: v48/v61/v72/v79 test files. Only test-only v28 takes the omit path.
2. **buddy/serverbound/operation_add.go group now `>61`** — YES. On main `w.WriteAsciiString(m.group)` was unconditional; now `if MajorVersion()>61`. v83+ still writes group → unchanged. IDA @0x4e9c03 (v61 name-only), @0x515575 (v72), @0x558844 (v87). Fixture `operation_add_test.go` (`hasGroup := major>61`) + `v48_test.go` name-only. Consistent.
3. **reactor/serverbound/hit.go isSkill `MajorAtLeast(72)` / skillId `MajorAtLeast(79)`** — YES. On main both were unconditional; now GMS-gated (JMS always). v83 (>=79) writes both → unchanged. IDA @0x5a5d1a(v48)/@0x633ac7(v61)/@0x6928bc(v72)/@0x6b8077(v79)/@0x7356c7(v83). Fixture `hit_test.go` (`hasIsSkill`/`hasSkillId` mirror gates) + `TestHitBytesV48` explicit 3-field bytes. Consistent.

## Phase-5 NPC_TALK fix (commit a0b8614c63) — OK

`startConversationHasXY` gate `MajorAtLeast(79)` → `MajorAtLeast(72) || ==61 || ==48`. v72 changes false→true (the false-pass fix); v48/v61 already true; **v79/v83/84/87/95/jms unchanged (all still carry x/y)**. v72 fixture updated to oid+x+y = `34 08 00 00 FB FF C8 00`; marker re-pinned 0x70dd49→0x63fd91. IDA CUserLocal::TalkToNpc @0x63FD91 (op57) at 3 send-sites. v48 pass independently confirmed sub_568A2A oid+x+y.

## Existing-version wire change — NONE

Totals frozen exactly: v83 367 / v84 345 / v87 379 / v95 399 / JMS 362; new legacy v79 228 / v72 216 / v61 208 / v48 165. All ✅ counts + 🟥=0. Every legacy gate is region-scoped GMS and fires below the existing-version boundary.

## TODO / stub / absolute paths — NONE

No `// TODO`/FIXME/HACK/StatusNotImplemented/501/"not implemented" in added Go (apparent hits were IDA hex addresses like `0x501973` and the substring in "anti-hack CRC"). No `/home/` or `/Users/` literals in committed non-scratchpad files.

## Deferral honesty — HONEST

Stages G (WZ ingest), H (k8s ports/provision), I (playthrough) deferred by explicit owner decision, repeatedly recorded in `progress.md`. Corroborated: no `*-playthrough.md` files exist; no legacy socket ports (7900/7200/6100/4800…) in `deploy/k8s/base/atlas-{login,channel}.yaml`; no committed doc claims runtime/live completion. The goal statement's "runnable tenants" is architectural intent, not a completion claim.

## Overall Assessment

- **Plan Adherence (Phase 5 / Stages A–F):** MOSTLY_COMPLETE (one table-completeness gap)
- **Recommendation:** READY_TO_MERGE (protocol deliverable; G/H/I explicitly out of scope) — optionally back-fill the two correction rows first.

## Action Items

1. (Minor) Add rows to `code-gate-audit.md` for `buddy/serverbound/operation_add.go` (group `>61`) and `reactor/serverbound/hit.go` (isSkill `>=72`, skillId `>=79`) so all existing-version-range corrections have FR-7 traceability in the reconciliation table (both already IDA-backed + fixtured in code).
2. (Owner) Confirm the v28 test-only boundary decision (fold into legacy `<61`, unverified-by-inference) as flagged in the ledger.
3. (Tracked, out of scope) Guild-BBS jms-fold matrix tooling; runtime Stages G/H/I as a follow-up.
