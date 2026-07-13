# task-113 (v48) — Stage D: export + audit + matrix wire-up

Final pass. Picked up from the prior agent's uncommitted artifacts (export + 3
tooling `.go` edits) and carried them through audit generation, matrix
regeneration, and commit.

## Export

- `docs/packets/ida-exports/gms_v48.json` — **not re-harvested** (export is
  non-idempotent, §10). Ground-truth original recovered from the dropped stash
  had **1000** function entries; current has **999** = original minus exactly one
  orphan key (`sub_4E5FAE`). Set-diff confirmed zero spurious additions.
- Two surgical splices only (see the CHANGE_KEYMAP conflict fix below):
  - Renamed the send-site `CFuncKeyMappedMan::SaveFuncKeyMap` entry from an
    `unresolved` stub to a resolved stub carrying the real address `0x4e5fae`.
  - Deleted the now-orphan `sub_4E5FAE` key (nothing references it after the
    IDB rename).
- Format matches siblings (pretty-printed, 2-space, ~323 KB).

## Unresolved fnames (flagged)

**None.** All 169 registry fname entries in `docs/packets/registry/gms_v48.yaml`
resolve in the export (0 named-missing, 0 `sub_*`-missing). Nothing to fabricate,
nothing to flag.

## COutPacket-delegate artifact (§10 recurring cleanup)

**Not present for v48.** The audit pipeline ran to completion (exit 1 = worst
verdict Blocker, the normal state for a pre-campaign version) with empty stderr
and 1105 report files written. No `delegate to COutPacket: not in export`
failure fired, so there was no artifact to strip.

## Audit SUMMARY (`docs/packets/audits/gms_v48/SUMMARY.md`)

552 rows: **✅ 24 / ❌ 246 / ⚠️ 48 / 🔍 101 / 🚫 133**.

## validate (live v48 IDB, port 13337)

`verified 259 / divergent 15 / missing-mode 0 / extra-mode 0 / unverifiable 726 /
allowlisted 0`. **missing-mode = 0.**

## Guard tests

`go test ./tools/packet-audit/...` clean. The three named guards pass explicitly:
`TestFnamedocOrderCoversVersionKeys`, `TestEveryVersionKeyHasShortLabel`,
`TestEveryVersionKeyHasTemplateFile`. `go vet ./tools/packet-audit/...` clean.

## matrix --check

**Exit 0.** STATUS.md problem grep (`orphan|dangling|stale|drift|unresolv|malformed`)
= **0**. Zero conflicts across all versions.

### Two conflicts surfaced and resolved (both were producible, not deferrable)

1. **CHANGE_KEYMAP × gms_v87 and × jms_v185** — a regression my v48 column would
   have introduced. v48's Stage-B registry used the unnamed `sub_4E5FAE` as the
   CHANGE_KEYMAP fname; every sibling uses `CFuncKeyMappedMan::SaveFuncKeyMap`
   (task-109 uniformity). The unnamed fname bypassed the matrix op-identity guard
   (fell back to opcode-occupancy), making v48 the only version counted as
   routing CHANGE_KEYMAP, which flipped the previously-verified v87/jms cells to a
   false template-wiring conflict. Fix: named the send-site in the v48 IDB
   (`0x4E5FAE` → `CFuncKeyMappedMan::SaveFuncKeyMap`, body-verified, saved),
   updated the registry fname, aligned the export. v87 (0x08F) and jms (0x08A)
   returned to ✅.
2. **NOTE_ACTION × gms_v48** — `registry says absent but an Atlas audit report
   exists`. The v48 Stage-B registry omitted NOTE_ACTION even though the send-site
   `CMemoListDlg::SetRet` exists resolved in the export (`0x534dc4`) and the client
   demonstrably implements it. Body-verified in the v48 IDB: on YesNo-confirm it
   emits `COutPacket(101)` + note-list body (= v61 NOTE_ACTION `CMemoListDlg::SetRet`,
   op 119; v48 Delta-18). Fix: added the registry entry (op 101 / 0x65) and routed
   the serverbound handler in the v48 seed template
   (`0x65 → NoteOperationHandle`, `LoggedInValidator`, mirroring v61). Cell is now
   `0x065 ❌` (in-scope worklist), no longer a conflict.

## Regression check — existing versions FROZEN

Verified counts unchanged from the pass baselines:

| v83 | v84 | v87 | v95 | jms | v72 | v79 | v61 |
|-----|-----|-----|-----|-----|-----|-----|-----|
| 367 | 345 | 379 | 399 | 362 | 216 | 228 | 208 |

Zero conflicts anywhere. The CHANGE_KEYMAP fix specifically preserved v87 (379)
and jms (362).

## v48 in-scope ❌ (Stage E campaign worklist)

**324 total = 152 op + 172 sub-struct.** Of which tier-1: 222 (74 op + 148 sub).
v48 verified = 0 (expected pre-campaign).

## Commit

One commit, explicit `git add` per path (never `-A`): export + audit reports +
3 tooling `.go` edits + regenerated STATUS.md/status.json + registry + seed
template + this report. Branch remains `task-113-gms-legacy-versions`.
