# Packet completeness critique — task-205-player-trade

Diff base: `git merge-base main HEAD` = `1e0a321b808a1cf70e3638a6433408209ac744a9`

**Verdict: 4 findings (0 CHANGED-BUT-UNCLAIMED, 4 CLAIMED-BUT-UNVERIFIED — all pre-existing, non-regressing, and already documented by the manifest's own controller ruling).**

## Step 1 — manifest resolution

`docs/tasks/task-205-player-trade/coverage-manifest.yaml` declares 7 `ops` entries
× 10 `versions` (gms_v48/61/72/79/83/84/87/92/95, jms_v185). Resolution against
`docs/packets/audits/status.json` (HEAD):

| manifest entry | resolves to status.json row |
|---|---|
| `interaction/clientbound/InteractionInteractionTradePutItem` | own row (new, tier1 sub-struct) |
| `interaction/clientbound/InteractionInteractionTradeAddMeso` | own row (new, tier1 sub-struct) |
| `interaction/clientbound/InteractionInteractionTradeConfirm` | own row (new, tier1 sub-struct) |
| `interaction/clientbound/InteractionInteractionTradeMesoLimit` | own row (new, tier1 sub-struct) |
| `interaction/clientbound/InteractionInteractionEnterResultSuccess` | **no independent row** — 0 matches by packet path; consumed into the shared `PLAYER_INTERACTION` clientbound row (`op=PLAYER_INTERACTION`, `packet=interaction/clientbound/InteractionInteractionEnter`), per the manifest's own "COVERAGE TOPOLOGY" note (lines 16-23) |
| `interaction/clientbound/InteractionInteractionLeave` | same shared `PLAYER_INTERACTION` row (0 independent matches) |
| `interaction/serverbound/InteractionOperationTransaction` | own row (`interaction/serverbound/InteractionOperationTransaction`) |

## Step 2 — CHANGED-BUT-UNCLAIMED

**Touched codec files** (`git diff $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v _test`):

```
libs/atlas-packet/interaction/clientbound/interaction_body.go
libs/atlas-packet/interaction/clientbound/interaction_trade.go   (new)
libs/atlas-packet/interaction/room.go
libs/atlas-packet/interaction/serverbound/operation_transaction.go
libs/atlas-packet/interaction/serverbound/version_gate.go
```

All five files live under `interaction/clientbound` or `interaction/serverbound`,
both of which are claimed dirs (multiple manifest entries share each dir). No
touched file falls outside a claimed dir. Detail:

- `interaction_trade.go` — new file, adds exactly the 4 claimed structs
  (`InteractionTradePutItem`, `InteractionTradeAddMeso`, `InteractionTradeConfirm`,
  `InteractionTradeMesoLimit`). Matches manifest 1:1.
- `interaction_body.go` — adds the 4 trade mode-key consts + body wrappers
  (claimed), plus `CharacterInteractionInviteResultMode` /
  `CharacterInteractionInviteResultKeyBody` / two invite-refusal keys and 6
  trade `LeaveReason` keys. These are additive semantic-key wrappers over the
  **existing** `NewInteractionInviteResult` / `CharacterInteractionLeaveReasonBody`
  codecs (no struct/wire-shape change), consistent with the manifest's note
  (lines 36-38) that Leave reuses the existing codec and produces no new row.
  Dir-claimed; no separate finding.
- `room.go` — adds `NewTradeRoom` (new constructor, no change to `Room`'s wire
  encoding/switch). Matches manifest note (lines 29-34) that EnterResultSuccess
  reuses the existing `Room`/`InteractionEnterResultSuccess` codec.
- `operation_transaction.go` — comment-only fname correction
  (`CCashTradingRoomDlg::Trade` → `CTradingRoomDlg::OnTrade`), matches manifest
  line 39.
- `version_gate.go` — comment-only rewrite of the `tradeCrcPresent` gate's
  justification; the gate expression itself (`t.MajorVersion() >= 83`) is
  byte-identical before/after. No off-by-one risk introduced.

**Touched version gates**: `git diff $BASE...HEAD -- 'libs/atlas-packet' | grep -E '(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'` only matches
`pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)` inside new
`_test.go` round-trip loops (test harness boilerplate, not a gate). No
`MajorVersion`/`MajorAtLeast`/`IsRegion` comparison was added or moved in a
non-test file — confirmed above, `tradeCrcPresent`'s comparison is unchanged.

**Matrix delta** (`git diff $BASE...HEAD -- docs/packets/audits/status.json`):
4 new rows (the 4 claimed `InteractionTrade*` structs, all 10 cells `verified`
except `InteractionInteractionTradeMesoLimit`/`jms_v185` = `n-a`), and one
existing row's cells changed:
`interaction/serverbound/InteractionOperationTransaction`:
`gms_v48/61/72/79`: `verified → n-a`, `gms_v92`: `incomplete → verified`. All
five rows are explicitly claimed by the manifest. No matrix row outside the
claimed set changed state (verified `PLAYER_INTERACTION` and
`InteractionInteractionUpdateMerchant` rows are byte-identical before/after).

No CHANGED-BUT-UNCLAIMED findings.

## Step 3 — CLAIMED-BUT-UNVERIFIED

| op | version | actual state | recommendation |
|---|---|---|---|
| `interaction/clientbound/InteractionInteractionEnterResultSuccess` | gms_v48 | `incomplete` (shared `PLAYER_INTERACTION` row) | Pre-existing gap, not introduced by this branch (row's `gms_v48`/`gms_v92` cells are unchanged before/after this diff). The manifest's own "COVERAGE TOPOLOGY" note explains why: promoting the arm-specific claim risks nothing here because the row already grades worst-of-all and was already `incomplete` at v48/v92 before this task touched it. Recommendation: either explicitly list gms_v48/gms_v92 under an `out_of_scope`/notes carve-out for these two entries (since the task cannot and does not claim to fix the pre-existing `enterError`-table gap — see `version-matrix.md` §"gms_v48 18 / gms_v92 0" notes), or drop those two version cells from scope for these ops. Do not silently rely on the row already being incomplete pre-task. |
| `interaction/clientbound/InteractionInteractionEnterResultSuccess` | gms_v92 | `incomplete` (shared `PLAYER_INTERACTION` row) | Same as above — v92 writer has no `enterError` table (version-matrix.md line 374), pre-existing and unrelated to trade wiring. |
| `interaction/clientbound/InteractionInteractionLeave` | gms_v48 | `incomplete` (shared `PLAYER_INTERACTION` row) | Same shared row/cell as above; same recommendation. |
| `interaction/clientbound/InteractionInteractionLeave` | gms_v92 | `incomplete` (shared `PLAYER_INTERACTION` row) | Same shared row/cell as above; same recommendation. |

All four findings trace to the same two matrix cells
(`PLAYER_INTERACTION`/clientbound, `gms_v48` and `gms_v92`), which the
manifest's authors were aware of (design controller ruling, coverage-manifest.yaml
lines 16-23) and chose not to promote to `n-a` or explicitly carve out in
`out_of_scope`. Per the completeness-critic contract this is reported
mechanically rather than passed silently, since `incomplete` never satisfies a
claim regardless of whether the state predates the branch. No fixture or
codec change is required to clear these — only a manifest edit (either narrow
the version claim for these two ops, or add an explicit note that gms_v48/v92
are acknowledged pre-existing gaps outside this task's remit).

All other `claimedOps` pairs (28 of 32 op×version combinations from the two
shared-row ops, plus all cells of the 4 new tier-1 rows and the
`OperationTransaction` row) are `verified` or legitimately `n-a`
(`InteractionInteractionTradeMesoLimit`/jms_v185, documented in
version-matrix.md line 34; `InteractionOperationTransaction`/gms_v48-79,
documented in version-gate.go's updated comment and version-matrix.md line
89-90).

No manifest-missing condition — `coverage-manifest.yaml` is present and
well-formed.
