# Completeness critic — task-227-cash-name-change-world-transfer

**Verdict: CLEAN — 0 findings.**

Diff base: `git merge-base main HEAD` = `312d74cfe47c5cc3b165ad2d67dcaef8efdb29a5`.
Manifest: `docs/tasks/task-227-cash-name-change-world-transfer/coverage-manifest.yaml`
(present, declares 67 cells across 8 ops).

## Step 1 — resolved scope

`claimedPackets` (resolved from `ops:` against `docs/packets/audits/status.json`):

| op | packet path | dir |
|---|---|---|
| NAME_TRANSFER (serverbound) | `cash/serverbound/CashCheckNameChangePossible` | `cash/serverbound` |
| WORLD_TRANSFER (serverbound) | `cash/serverbound/CashCheckTransferWorldPossible` | `cash/serverbound` |
| CASHSHOP_CHECK_NAME_CHANGE | `cash/clientbound/CashCheckNameChange` | `cash/clientbound` |
| CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT | `cash/clientbound/CashCheckTransferWorldPossibleResult` | `cash/clientbound` |
| CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT | `cash/clientbound/CashCheckNameChangePossibleResult` | `cash/clientbound` |
| CANCEL_NAME_CHANGE_RESULT | (no `packet` field in status.json row; file `cash/clientbound/cancel_name_change_result.go`) | `cash/clientbound` |
| CANCEL_TRANSFER_WORLD_RESULT | (no `packet` field; file `cash/clientbound/cancel_transfer_world_result.go`) | `cash/clientbound` |
| CANCEL_NAME_CHANGE_BY_OTHER | (no `packet` field; file `character/clientbound/cancel_name_change_by_other.go`) | `character/clientbound` |

## Step 2 — CHANGED-BUT-UNCLAIMED

**Touched codecs** (`git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v _test`):

```
libs/atlas-packet/cash/clientbound/cancel_name_change_result.go
libs/atlas-packet/cash/clientbound/cancel_transfer_world_result.go
libs/atlas-packet/cash/clientbound/check_name_change.go
libs/atlas-packet/cash/clientbound/check_name_change_possible_result.go
libs/atlas-packet/cash/clientbound/check_transfer_world_possible_result.go
libs/atlas-packet/cash/serverbound/check_name_change_possible.go
libs/atlas-packet/cash/serverbound/check_transfer_world_possible.go
libs/atlas-packet/character/clientbound/cancel_name_change_by_other.go
```

All 8 files map to `cash/clientbound`, `cash/serverbound`, or `character/clientbound`
dirs, all of which are `claimedPackets`. No unclaimed codec file.

**Touched version gates** (`grep -E '(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'`
over the same diff, attributed per file):

| file | gate line | claimed? |
|---|---|---|
| `cash/clientbound/check_transfer_world_possible_result.go` | `return t.Region() != "JMS"` | yes (`cash/clientbound`) |
| `cash/serverbound/check_name_change_possible.go` | `return t.MajorAtLeast(95)` | yes (`cash/serverbound`) |
| `cash/serverbound/check_transfer_world_possible.go` | `return t.MajorAtLeast(95)`, `if t.Region() == "JMS" {`, `return t.Region() != "JMS"` | yes (`cash/serverbound`) |
| `character/clientbound/cancel_name_change_by_other_test.go` | added GMS v72–v95 version-table rows | yes (`character/clientbound`) |

No gate change lands outside a claimed dir.

**Matrix delta** (`git diff $BASE...HEAD -- docs/packets/audits/status.json`, op rows only):

Rows with cell-state transitions: `NAME_TRANSFER`, `WORLD_TRANSFER`,
`CASHSHOP_CHECK_NAME_CHANGE`, `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT`,
`CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT` — all 5 in `claimedPackets`/`ops`.
`WORLD_TRANSFER`'s row was moved (previously mis-filed jms_v185 `0x009` under
`NAME_TRANSFER`; the manifest's own header documents this exact correction in
its `derivation.md §1.5` note). No matrix row outside the declared 8 ops changed
state.

## Step 3 — CLAIMED-BUT-UNVERIFIED

Final (HEAD) `status.json` cell states for every declared op × version, cross-checked
against the manifest's own per-row cell-count table:

| op | verified cells (HEAD) | manifest declared count | match |
|---|---|---|---|
| CANCEL_NAME_CHANGE_RESULT | v61,v72,v79,v83,v84,v87,v92,v95 (8) | 8 | yes |
| CANCEL_TRANSFER_WORLD_RESULT | v61,v72,v79,v83,v84,v87,v92,v95 (8) | 8 | yes |
| CANCEL_NAME_CHANGE_BY_OTHER | v72,v79,v83,v84,v87,v92,v95 (7) | 7 | yes |
| CASHSHOP_CHECK_NAME_CHANGE | v48,v61,v72,v79,v83,v84,v87,v92,v95 (9) | 9 | yes |
| CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT | v48,v61,v72,v79,v83,v84,v87,v92,v95,jms_v185 (10) | 10 | yes |
| CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT | v79,v83,v84,v87,v92,v95 (6) | 6 | yes |
| NAME_TRANSFER (serverbound) | v48,v61,v72,v79,v83,v84,v87,v92,v95 (9) | 9 | yes |
| WORLD_TRANSFER (serverbound) | v48,v61,v72,v79,v83,v84,v87,v92,v95,jms_v185 (10) | 10 | yes |
| **Total** | **67** | **67** | **yes** |

Every remaining cell not counted above is `n-a` in `status.json` (e.g. `gms_v48`/
`jms_v185` on the three `CANCEL_*` rows, `gms_v48/v61/v72`/`jms_v185` on
`CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT`, `jms_v185` on `NAME_TRANSFER`,
`gms_v95` on `WORLD_TRANSFER`... etc.) and none of those `n-a` cells is treated
as in-scope coverage in the manifest's `fields`/version lists — the manifest's
own per-row version lists (line 25-32 of `coverage-manifest.yaml`) already
exclude every `n-a` cell. No mismatch.

No `partial` or `incomplete` state remains on any claimed op × version pair.

## Summary

- CHANGED-BUT-UNCLAIMED: **0**
- CLAIMED-BUT-UNVERIFIED: **0**

The manifest's own header (lines 6–34) already documents and reconciles the
59→67 cell-count discrepancy referenced in the dispatch context; the final
`status.json` state agrees with the manifest's declared 67 cells exactly, cell
for cell. `out_of_scope` entries (`character_pending_changes`,
`imprint-configs`, the saga, `NAME_CHANGED` consumers, the name-validity
endpoint, the atlas-ui panel, and the dead-code deletion) were not touched in
`libs/atlas-packet` or `docs/packets/audits/status.json`, so they required no
cross-check beyond confirming absence from the two diffs above, which held.
