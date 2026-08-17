# Completeness Critic — task-230-scripted-items

**Verdict: CLEAN** on the two mechanical failure modes (CHANGED-BUT-UNCLAIMED
codec/gate, CLAIMED-BUT-UNVERIFIED). 1 informational finding below on
generated-doc scope, not a blocking scope hole.

Base: `git merge-base origin/main HEAD` = `312d74cfe47c5cc3b165ad2d67dcaef8efdb29a5`
(confirmed identical to `origin/main` tip at audit time). Branch:
`task-230-scripted-items` (confirmed via `git branch --show-current`).

## Manifest (`docs/tasks/task-230-scripted-items/coverage-manifest.yaml`)

```yaml
ops: [SCRIPTED_ITEM, NPC_ITEM_USE_REQUEST]
versions: [gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185]
fields:
  - inventory/serverbound/InventoryScriptedItem: no version gates
  - inventory/serverbound/InventoryNpcItemUse: no version gates
```

Resolved via `docs/packets/audits/status.json`:
- `SCRIPTED_ITEM` → `inventory/serverbound/InventoryScriptedItem`
- `NPC_ITEM_USE_REQUEST` → `inventory/serverbound/InventoryNpcItemUse`

`claimedPackets` = `{inventory/serverbound}`. No `out_of_scope` entries declared.

## CHANGED-BUT-UNCLAIMED

None found.

**Touched codecs.** `git diff --name-only $BASE...HEAD -- 'libs/atlas-packet'`
returns exactly:
```
libs/atlas-packet/inventory/serverbound/npc_item_use.go
libs/atlas-packet/inventory/serverbound/npc_item_use_test.go
libs/atlas-packet/inventory/serverbound/scripted_item.go
libs/atlas-packet/inventory/serverbound/scripted_item_test.go
```
Both non-test files live in `inventory/serverbound`, which is in
`claimedPackets`. No unclaimed codec file.

**Touched version gates.** `git diff $BASE...HEAD -- 'libs/atlas-packet' | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'`
matched only two lines, both in `*_test.go` files (`npc_item_use_test.go:166`,
`scripted_item_test.go:534`): `ctx := test.CreateContext(v.Region,
v.MajorVersion, v.MinorVersion)`. This is per-version test-table scaffolding
(iterating a version list to build a test context), not a `MajorAtLeast` /
`IsRegion` gate in codec logic — a false positive of the grep pattern
(`v.MajorVersion` substring-matches `MajorVersion`). Neither
`scripted_item.go` nor `npc_item_use.go` contains any gate diff, consistent
with the manifest's own claim ("no version gates — body identical on all
[eight|nine] versions"). Confirmed by file-scoped `diff --git` headers: both
`ctx := test.CreateContext(...)` hits fall inside the `_test.go` diff hunks,
none inside the codec files.

**Matrix delta.** `git diff $BASE...HEAD -- docs/packets/audits/status.json`
touches exactly 4 row blocks (2 removed, 2 added) — the `SCRIPTED_ITEM` and
`NPC_ITEM_USE_REQUEST` `op` rows, each replacing a stale duplicate
(`tier1: false`, all cells `incomplete`/`n-a`, no `"packet"` field) with the
canonical `tier1: true` row (all cells `verified`/`n-a`, `"packet"` field
present). No other `op` row changed state. Both are in `claimedOps`.

**ida-exports / registry additions.** Per the task's explicit ask: the nine
`docs/packets/ida-exports/*.json` files and the three
`docs/packets/registry/gms_v{61,72,79}.yaml` files were checked line-by-line.
Every added block is one of the two claimed fnames only:
- `CWvsContext::SendScriptRunItemRequest` (→ SCRIPTED_ITEM)
- `CWvsContext::SendSelectNpcItemUseRequest` (→ NPC_ITEM_USE_REQUEST)

registry diff sample (`gms_v61.yaml`): a single new `op: NPC_ITEM_USE_REQUEST`
entry with `provenance: ida-discovered` and a `task-230` note citing the IDA
address. No third fname, no unrelated op, appears anywhere in these 12 files.
These are additive-only and within the manifest's declared scope — not a
scope hole.

## Informational (not a blocking finding)

**Support-doc churn beyond the two declared ops.**
`docs/packets/audits/support/gms_v61.md`, `gms_v72.md`, `gms_v79.md` (and
`STATUS.md`'s tool-hash line) show large diffs listing dozens of unrelated
packets (e.g. `cash/serverbound/CashItemUseAvatarMegaphone`,
`field/clientbound/FieldMtsResultBuyItemDone`,
`guild/clientbound/GuildBBSThread`, several `IDA_0X*` op rows) appearing or
disappearing from the "gaps" tables, and the top-line verified/applicable
counts shifting (e.g. gms_v61: 232→269 verified, 493→559 applicable).

This is **not** a matrix-delta scope hole by the mechanical definition: the
actual coverage-of-record diff (`status.json`) is clean — only the two
declared `op` rows changed state, nothing else was promoted, demoted, or
newly marked `n-a`. The support docs are a *derived render*, not source of
truth. `STATUS.md`'s `Tool:` hash changed
(`830d32c7…` → `43783835…`), which traces to the branch's `run.go` edit
(commit `feat(packets): register SCRIPTED_ITEM/NPC_ITEM_USE_REQUEST on legacy
versions`, which added two `candidatesFromFName` cases for the two claimed
fnames — no other logic changed). A tool-hash change is the packet-audit
tool's own trigger to fully regenerate every rendered doc against current
registry/status data, which appears to be what surfaced pre-existing drift
between the previously-stale rendered docs and the already-true underlying
data (registry + status.json) for unrelated packets that this task did not
touch.

Root cause note: `git log --oneline` on this branch shows a
`Merge remote-tracking branch 'origin/main' into task-230-scripted-items`
commit (`0be022ffd`) partway through the branch's history. Since
`merge-base(origin/main, HEAD)` currently equals `origin/main`'s tip, none of
main's own commits should appear in the three-dot diff — but doc regeneration
performed *as part of resolving that merge* (or triggered by the later
tool-hash bump) is itself unique history on this branch, which is why it
shows up. Recommendation: confirm with the author that this support-doc
regeneration is an intentional side effect of the tool-hash bump / stale-doc
catch-up and not an accidental bundling of unrelated matrix work into this
PR — if intentional, no manifest change is needed (these are render
artifacts, not new coverage claims); if any of those unrelated
"gaps"-table entries actually reflect a coverage *promotion* for an
unclaimed op, that would need to show up in `status.json` too, which it does
not, so no action is required beyond the "please confirm" callout for
reviewer awareness.

## CLAIMED-BUT-UNVERIFIED

None found. Final (`HEAD`) `status.json` cell states for every claimed
`op × version` pair:

| op | version | state |
|---|---|---|
| SCRIPTED_ITEM | gms_v61 | n-a (manifest declares 9 versions incl. v61; row shows v61 absent/n-a — see note) |
| SCRIPTED_ITEM | gms_v72 | verified |
| SCRIPTED_ITEM | gms_v79 | verified |
| SCRIPTED_ITEM | gms_v83 | verified |
| SCRIPTED_ITEM | gms_v84 | verified |
| SCRIPTED_ITEM | gms_v87 | verified |
| SCRIPTED_ITEM | gms_v92 | verified |
| SCRIPTED_ITEM | gms_v95 | verified |
| SCRIPTED_ITEM | jms_v185 | verified |
| NPC_ITEM_USE_REQUEST | gms_v61 | verified |
| NPC_ITEM_USE_REQUEST | gms_v72 | verified |
| NPC_ITEM_USE_REQUEST | gms_v79 | verified |
| NPC_ITEM_USE_REQUEST | gms_v83 | verified |
| NPC_ITEM_USE_REQUEST | gms_v84 | verified |
| NPC_ITEM_USE_REQUEST | gms_v87 | verified |
| NPC_ITEM_USE_REQUEST | gms_v92 | verified |
| NPC_ITEM_USE_REQUEST | gms_v95 | verified |
| NPC_ITEM_USE_REQUEST | jms_v185 | verified |

Note on the one `n-a`: `SCRIPTED_ITEM`'s `status.json` row has no `gms_v48`
key at all (op is n-a there, consistent — v48 isn't in the manifest's
`versions` list) and `gms_v61: {"state": "n-a", "opcode": -1}` (the sender
`CWvsContext::SendScriptRunItemRequest` doesn't exist pre-v72 per the
`run.go` comment: "Opcode exists v72 through jms_v185; v12/v48/v61 lack the
sender entirely"). The manifest lists `gms_v61` in its `versions` array
(shared with `NPC_ITEM_USE_REQUEST`, which *is* verified on v61), so this is
a version genuinely inapplicable for this one op, not an unverified claim —
matches the task's own stated brief ("SCRIPTED_ITEM x8 — absent on v48/v61").
This is exactly the "legitimately inapplicable cell, `n-a` in the matrix"
case the playbook calls out — correctly `n-a`, not silently passed as
coverage. No manifest change needed (the manifest's `versions` list is
op-agnostic across both ops, which is why v61 legitimately shows n-a for one
op and verified for the other).

## Summary

- CHANGED-BUT-UNCLAIMED (codec/gate/matrix): **0**
- CLAIMED-BUT-UNVERIFIED: **0**
- Informational: **1** (support-doc regeneration churn beyond declared scope — recommend author confirms intentional, no manifest action required since `status.json` is clean)
