# Plan Audit — task-113 GMS v48 pass (Stages A–F)

**Plan/contract:** `.superpowers/sdd/progress.md` (PASS v48, tasks 4.A–4.F) + per-stage artifacts
**Audit Date:** 2026-07-04
**Branch:** task-113-gms-legacy-versions
**Anchor:** gms_v61 (IDB port 13337)

## Executive Summary

The v48 pass (final of 4) was faithfully executed across all six stages A–F. Every stage artifact exists and is non-trivial. No regression: all 8 pre-existing versions match their frozen verified counts exactly; v48 reached 165 verified tier-1 cells. `matrix --check` exits 0 with 0 conflicts/drift. Builds, vet, and both test suites are clean. **Zero real tier-1 gaps** — every remaining v48 tier-1 ❌ is either dispositioned in `_unimplemented.json` or a matrix double-count artifact that also affects the already-signed-off v84. **Verdict: PASS / READY.**

## Stage Artifact Verification

| Stage | Artifact | Status |
|---|---|---|
| A | `v48-packet-delta.md` (496 lines) | EXISTS, substantive |
| B | `docs/packets/registry/gms_v48.yaml` — 169 op entries (93 cb + 76 sb) + `discover_gms_v48.md` | EXISTS; self-validates; 0 dup opcodes |
| C | `template_gms_48_1.json` (75 handlers / 54 writers) + `v48-stageC-template.md` | Valid JSON; **0 missing validators, 0 dup opcodes** |
| D | `docs/packets/ida-exports/gms_v48.json` (335 KB) + `docs/packets/audits/gms_v48/` (1072 entries) + matrix wire-up + `v48-stageD-export.md` | EXISTS; matrix column live |
| E | 11 batch + 9 close reports + `v48-stageE-close.md` | All present |
| F | `code-gate-audit.md` (610 lines) + `v48-stageF-codegate.md` | EXISTS |

**Contract-vs-actual note (not a defect):** contract cited "168 registry / 74 handlers"; actual is 169 sb-entry / 75 handlers. The +1 is the `NOTE_ACTION`/`NoteOperationHandle 0x65` entry legitimately added in Stage D (progress 4.D: "NOTE_ACTION missing from registry ... added + routed"). Reconciles cleanly.

## Build & Test Results

| Check | Result |
|---|---|
| `go build ./...` libs/atlas-packet | PASS (exit 0) |
| `go vet ./...` libs/atlas-packet | PASS (exit 0) |
| `go build ./...` tools/packet-audit | PASS (exit 0) |
| `go test ./...` libs/atlas-packet | PASS (0 FAIL) |
| `go test ./...` tools/packet-audit | PASS (0 FAIL) |
| `packet-audit matrix --check` | **exit 0, 0 conflicts, 0 drift** |
| regenerated status.json vs committed | 0-line diff (committed matrix is current) |

## Regression (verified counts frozen)

All match exactly: v83 367, v84 345, v87 379, v95 399, jms 362, v72 216, v79 228, v61 208. v48 = 165. **No regression.**

## Stage E Completeness (the v79 trap — sub-struct + login cells)

Independent scan of `status.json` (all rows, op + sub-struct, not kind==op only): 19 rows where v48 ∈ {incomplete,partial} while v83 AND v61 are both verified.

- **6 tier-1 candidates, all resolved:**
  - `CashShopOperationEnableEquipSlot`, `NpcSayImage`, `NpcAskQuiz`, `NpcAskSpeedQuiz`, `NpcAskBoxText` → **dispositioned in `_unimplemented.json`** (version-absent, IDA-enumeration evidence).
  - `npc/serverbound/NpcContinueConversation` → **NOT a gap:** the op-cell is verified for v48 (opcode 47, byte-fixture `conversation_v48_test.go` passes). The residual `incomplete` is a matrix double-count artifact on the opcode=-1 sub-struct row — the **identical artifact appears on the completed v84** (`tier-1 without fixture; verdict ⚠️`). Matches progress note "sub ❌ = matrix gap-fill artifact."
- **13 non-tier-1** (12 login-flow ops: LOGIN_STATUS, SERVERSTATUS, CHARLIST_REQUEST, GENDER_DONE, CHECK_PINCODE, AFTER_LOGIN, REGISTER_PIN, SERVERLIST_REQUEST, WORLD_INFORMATION, PICK_ALL_CHAR, CHAR_SELECT, SERVERSTATUS_REQUEST + `login/serverbound/ServerListRequest` sub-struct). `tier1=False` → outside the tier-1 gap criterion. **Observation:** these are byte-fixture-verified on all other GMS versions but not v48; they were registered (Stage B2f) but not fixtured. Runtime G/H/I was owner-deferred and login fixtures are non-tier-1, so this is consistent scope, not a silent skip.

## Spot-checks (reconciliation-doc claims vs reality)

- Guild ops fix `c6184bb85e` — **confirmed**: `template_gms_48_1.json` `SET_SKILL_RESPONSE` 78→77 and phantom `BOARD_AUTH_KEY_UPDATE: 77` removed (1 file, −2/+1).
- `ITEM_SORT` / `ITEM_SORT2` → v48 cells **n-a** (match `_unimplemented.json`).
- `PARTY_OPERATION` (ChangeLeader/`SendChangePartyBossMsg` arm) → op **verified**, arm stripped n-a per disposition.
- Interaction merchant blacklist add/remove arms → v48 **incomplete = v61 anchor incomplete** (criterion-b shared gap, so not flagged).

## No stubs

Strict scan of all v48-pass Go added lines: **0** `// TODO`, `panic("not implemented")`, or `StatusNotImplemented`.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (v48 pass; runtime G/H/I deferred by owner, out of scope)
- **Real gaps found:** none

## Action Items

None required for the v48 pass. Optional follow-up (non-blocking, tracked separately): the 12 login-flow clientbound/serverbound ops are byte-fixtured on all other GMS versions but remain `incomplete` on v48 — if login-flow parity is desired at fixture level, add v48 login fixtures. They are `tier1=False` and outside the current contract.
