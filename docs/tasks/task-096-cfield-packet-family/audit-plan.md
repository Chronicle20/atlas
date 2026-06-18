# Plan Audit — task-096-cfield-packet-family

**Plan Path:** docs/tasks/task-096-cfield-packet-family/plan.md
**Audit Date:** 2026-06-15
**Branch:** task-096-cfield-packet-family
**Base Branch:** main

## Executive Summary

The plan was faithfully executed. All 75 work-list CField ops are driven to ✅ verified or
⬜ n/a across the applicable versions (gms_v83/v84/v87/v95 + jms_v185); **zero work-list ops
remain ❌**. `go run ./tools/packet-audit matrix --check` exits **0** with 0 conflicts. The
only two ❌ rows in STATUS.md are exactly the documented OUT-OF-SCOPE serverbound
name-collision rows (distinct fnames, not work-list ops). All Stage-0/Stage-1 artifacts exist,
deploy-notes.md honestly records all five caveats, and build/test/vet are clean on
libs/atlas-packet, atlas-channel, and atlas-configurations. **Recommendation: READY_TO_MERGE.**

## Stage / Goal Completion

| Stage / Item | Status | Evidence |
|---|---|---|
| Goal: 75 work-list ops ✅/⬜ across applicable versions | DONE | Burndown below; all X=0 except the 2 out-of-scope rows |
| Stage 0.1 baseline | DONE | structures/baseline.md (9.7K) |
| Stage 0.2 codec inventory | DONE | structures/codec-inventory.md (9.6K) |
| Stage 0.3 C-row resolution | DONE | structures/triage.md; commit 6669c3efa removed 4 spurious IDA_0x/CONTI_MOVE-Init placeholders |
| Stage 0.4 triage table (D4) | DONE | structures/triage.md (31K, one row/op) |
| Stage 1.A–1.E per-version harvest | DONE | structures/gms_v83.md, gms_v84.md, gms_v87.md, gms_v95.md; jms layouts folded into serverbound-r-sb.md / cfield-family-batch.md |
| Stage 1.F applicability | FOLDED | No standalone applicability.md; applicability is encoded in triage.md + per-version structures docs + evidence VERSION-ABSENT pins. Plan permitted folding ("applicability not required if folded"). |
| Stage 2 Cluster 1 (chat relocation + new chat) | DONE | git mv chat→field for multi.go/whisper.go (CB), general.go (SB); SpouseChat new codec; whisper.go also fixed x/y int16→int32 wire bug (commit 4de310349) |
| Stage 2 Cluster 1 admin/slash family (6 SB ops) | DONE | field/serverbound/admin_chat.go etc.; commits a6b579ed5, 19ca82225, 1ea1540fe |
| Stage 2 Clusters 2–3 (transfer/obstacle/quest/clock/boss/MTS/door/guild) | DONE | 77 field codec files; STATUS rows all ✅ |
| Stage 2 Cluster 4 (foothold/stalk C-cluster) | DONE | FOOTHOLD_INFO/StalkResult verified; IDA_0x rows resolved to ✅/⬜ or removed |
| Stage 2 Cluster 5 (minigames ~30 ops) | DONE | snowball/coconut/guildboss/tournament/wedding/ariant/pyramid/sheep-ranch codecs + routes |
| Stage 3 deploy-notes.md | DONE | docs/tasks/task-096-cfield-packet-family/deploy-notes.md (491 lines, per-version opcode tables + rollout + 5 caveats) |
| Stage 4 gates | DONE | build/vet/test clean (see Build & Test); matrix --check exit 0 |

## Burndown (work-list ops with remaining ❌)

Built the op list from `structures/cfield-ops.md` (73 unique op names; 75 rows minus the WHISPER
duplicate listing) and graded each op's STATUS.md row across the 5 version cells.

**Result: 0 work-list ops with remaining ❌.** Every CField clientbound/serverbound work-list op
is ✅ for applicable versions and ⬜ (VERSION-ABSENT) elsewhere.

The only ❌ rows in STATUS.md among CField fnames are the two documented OUT-OF-SCOPE
name-collision rows — distinct fnames from the work-list ops:
- STATUS.md:616 `WHISPER` / `CField::OnWhisper; CField::SendChatMsgWhisper; CField::SendLocationWhisper`
  (serverbound) — the verified clientbound work-list op `CField::OnWhisper` is at STATUS.md:180
  (✅ all 5 versions, `field/clientbound/FieldWhisperError`).
- STATUS.md:618 `SPOUSE_CHAT` / `CUIStatusBar::SendCoupleMessage` (serverbound) — the verified
  clientbound work-list op `CField::OnCoupleMessage` is at STATUS.md:182 (✅ v83/v84/v87/v95,
  ⬜ jms, `field/clientbound/FieldSpouseChat`).

Both are explicitly NOT in the 75-op work-list (per task spec known-acceptable note) and are
recorded as out-of-scope follow-ups in deploy-notes.md caveat #2.

## Matrix Check

`go run ./tools/packet-audit matrix --check` → **exit 0** (0 conflicts).
`go run ./tools/packet-audit matrix` produces **no** orphan/stale/dangling/drift/conflict line
mentioning any CField/field. packet.

## Chain Spot-Checks (codec + test markers + evidence + route + audit report)

| Op | Codec | Markers | Evidence | Template route | Audit report |
|---|---|---|---|---|---|
| SPOUSE_CHAT (CB) | field/clientbound/spouse_chat.go | 4 (jms VERSION-ABSENT) | gms_v83/84/87/95 yaml | template_gms_83_1.json 0x88→SpouseChat | gms_v83/FieldSpouseChat.md |
| AdminChat (SB) | field/serverbound/admin_chat.go | 5 | gms_v83 yaml + others | handler/admin_chat.go + main.go `handlerMap[fieldsb.AdminChatHandle]` | present |
| GENERAL_CHAT (SB) | field/serverbound/general.go (git mv from chat/) | preserved | matrix ✅ all 5 | — | — |
| WHISPER (CB) | field/clientbound/whisper.go (git mv + int16→int32 fix) | yes | yes | ✅ all 5 | present |

Chat relocation (D3) confirmed as move-not-rewrite: `git mv` recorded for multi.go, whisper.go
(clientbound) and general.go (serverbound); shared `chat/clientbound/general.go` (world message)
and `chat/serverbound/multi.go|whisper.go` (distinct ops) correctly left in place.

## Documented Caveats (deploy-notes.md §Known caveats) — all honestly recorded

1. **WEDDING_ACTION/WEDDING_TALK v84 handler routes absent** — v84 opcodes 0x8F/0x90 collide with
   stale ALLIANCE_OPERATION/DENY_ALLIANCE_REQUEST registry rows; matrix grades ✅ via
   opcode-occupancy but the v84 channel will NOT dispatch inbound wedding packets. Flagged as
   out-of-scope v84-registry maintenance. This is a **genuine functional gap, honestly documented**
   (not a silent skip). v83/v87/v95 ship the wedding handler rows.
2. **2 out-of-scope serverbound ops remain ❌** (the WHISPER/SPOUSE_CHAT collisions above).
3. **v84 UNNAMED_R364/R366/R369 phantom registry rows** — stale, cleanup deferred.
4. **CSV transposition** — HORNTAIL_CAVE ↔ WITCH_TOWER_SCORE_UPDATE swapped in the reference CSV
   (GMS v83+v87 columns); registries corrected IDB-verified (commit 44488a7db), CSV left as-is.
5. **matrix grader change** (`routedElsewhere` op-identity-aware, commit baa937176) shipped with
   **no regression test** — self-flagged.

## Build & Test Results

| Module | Build | Vet | Tests | Notes |
|---|---|---|---|---|
| libs/atlas-packet | PASS | PASS | PASS | field/clientbound + field/serverbound ok |
| atlas-channel | PASS | PASS | PASS | socket/handler, socket/model, socket/writer ok |
| atlas-configurations | PASS | — | PASS | tenants, characters, preset ok |

No go.mod changed (`git diff --name-only -- '**/go.mod'` empty) → no docker bake required, per
plan Stage 4 Step 3.

## Notes / Minor Observations (non-blocking)

- **Plan checkboxes are all unchecked (0/88).** This is expected and not a gap: the checkboxes are
  the reusable recipe scaffold (R-CB.1–R-CB.10) and cluster sub-group lines, not per-op tracking.
  Completion is evidenced by the matrix burndown and committed artifacts, not box state.
- **No standalone structures/applicability.md** (Stage 1.F). Applicability is folded into triage.md
  + per-version structures docs + per-evidence VERSION-ABSENT pins; the plan explicitly allowed this
  ("applicability not required if folded"). Not a gap.
- **Caveat #1 (WEDDING v84)** is the one substantive functional limitation. It is correctly scoped
  out (stale v84 ALLIANCE registry rows are a pre-existing v84-table defect, not task-096 work) and
  honestly disclosed. Recommend tracking it as a follow-up task.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None blocking. Optional follow-ups (all already disclosed in deploy-notes.md):
1. File a follow-up task for the v84 WEDDING handler routing (stale ALLIANCE opcode re-derivation).
2. File a follow-up for the 2 out-of-scope serverbound WHISPER/SPOUSE_CHAT collision ops.
3. Consider a regression test for the `routedElsewhere` op-identity-aware grader change (caveat #5).
