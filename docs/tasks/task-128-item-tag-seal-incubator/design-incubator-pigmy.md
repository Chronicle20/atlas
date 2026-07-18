# task-128 correction — Incubator is the "Pigmy Egg" (pre-gachapon) mechanic

**Ships on:** the `task-128-item-tag-seal-incubator` branch (extends PR #909). This
corrects the incubator sub-feature; item-tag and sealing-lock are unaffected.

## 1. What the incubator actually is

The "Incubator" cash item (`5060002`, name "Incubator") is the **precursor to
gachapon**. You use it on a **Pigmy Egg** — an ETC item in `4170000–4170009`
(all named "Pigmy Egg"; the *region* is encoded in the id, e.g. `4170005 =
Ludibrium`). The egg is the **region key**: each egg has its own reward pool and
its own success NPC ("Pigmy & Etran", per region). On use the server consumes the
egg + the incubator, awards a weighted-random item from *that egg's* pool, and the
client shows a modal NPC "ok" dialog naming the awarded item.

### Client eligibility gate (why arbitrary items are rejected)
`CUIIncubator::PutItem` (v95 `@0x7ca3f0`) only accepts a dragged target whose id is
in the gachapon-aggregation range (`4170000 ≤ id ≤ 41700099` in v95), then calls
`CItemInfo::GetGachaponItemIDByAggID`. The real Pigmy Eggs occupy `4170000–4170009`.
Anything else → "this item cannot be incubated" (client-side; the server has no
eligibility check). The gate is client-only, so the server must be authored to the
same rule (only pigmy eggs are valid targets).

## 2. Authoritative data source — `Etc.wz/incubatorInfo.img`

Confirmed by decompiling `CItemInfo::RegisterGachaponItemInfo` (v95 `@0x5bf040`): the
client loads `Etc/incubatorInfo.img` into `m_mGachaponItemInfo` (`ZMap<long,
GACHAPONITEMINFO>`, keyed by egg item id). Per-egg `GACHAPONITEMINFO`:

| WZ node | struct field | meaning |
|---|---|---|
| `su` | `nSucessNpcID` | success NPC id (the Pigmy & Etran NPC for this region) |
| `msg` (`no`/`wr`/`su`/`fi…`) | `aMsg[]` | dialog text templates shown on award |
| `usingAggScope/{i}/{min,max}` | `aAbleUsingAggScope[]` (`nMinType/nMaxType`) | region/aggregation scope |
| `isBonus` | `bBonus` | whether a bonus item is granted |
| `isNoGradeResult` | `bNoGradeResult` | grade-result flag |
| `finalConfirm/{eq,co,etc,in}` | `aFinalconfirmInfo[4]` | final-confirm dialog config |

`GetGachaponSucessNpc(eggId)` (v95 `@0x5c06b0`) simply returns `su`. **The reward
pool is NOT a flat list in this node** — it derives from the aggregation scope, so
there is no simple "pigmy reward table" to ingest. Hence: **reward pools stay
tenant-configured** (decision), but the **NPC / message / eligible-egg set come from
`incubatorInfo.img`** and must be ingested.

## 3. Client result packet — `INCUBATOR_RESULT` is VERSION-GATED (current bug)

The client renders the award as a **modal NPC dialog** (`CUtilDlgEx` +
`GetGachaponSucessNpc(gachaponItemID)` + `GetGachaponMsg`), not just a chat line.
Decoded body **differs by version**:

| version | body | NPC source |
|---|---|---|
| v83 / v84 | `int itemId, short count` (**flat**) | fixed NPC (client-side) |
| v95 | `int itemId, short count, int gachaponItemID, int bonusItemID, int bonusCount` (**extended**) | `GetGachaponSucessNpc(gachaponItemID)` — needs the egg id |

Evidence: v95 `CWvsContext::OnIncubatorResult @0xa00380` decodes 5 fields and looks up
the NPC from `gachaponItemID`; v84 `@0xa73a5b` decodes only `itemId+count`.

**Correction (verified in code):** the `IncubatorResult` codec
(`libs/atlas-packet/incubator/clientbound/result.go`) is *already* version-gated —
flat `itemId+count` for v83/84/87/jms (IDA-re-verified in the file comment) and the
extended tail for v95. The actual bug is narrower: it hard-codes the v95 tail to
`WriteInt(0)/WriteInt(0)/WriteInt(0)`, so `gachaponItemID = 0`. The v95 client then
calls `GetGachaponSucessNpc(0)` → no match → a default/blank NPC instead of the
region's Pigmy & Etran. **Fix = send the sacrificed egg id as `gachaponItemID`** (the
bonus pair stays 0 — Atlas rolls one reward). The client owns `incubatorInfo.img`, so
it resolves the correct NPC locally from the egg id; **the server does not need to
ingest `incubatorInfo.img` for the dialog to work** — it only needs to emit the egg
id. Re-verify the v95 `INCUBATOR_RESULT` byte-fixture cell after populating it.

## 4. Gap vs. the current task-128 implementation

| aspect | real mechanic | task-128 as built |
|---|---|---|
| target eligibility | must be a Pigmy Egg `4170000–4170009` | any non-empty slot (no check) |
| reward pool | **per-egg / per-region** | one **flat** tenant `incubator-rewards` list |
| region | egg id *is* the region | not modeled |
| result NPC | per-egg (`su` from `incubatorInfo.img`) | not sent (flat packet) |
| result packet | version-gated (flat v83/84, extended v95) | flat for all → wrong for v95 |

## 5. Corrected design (all on the task-128 branch)

1. **atlas-data — ingest `Etc.wz/incubatorInfo.img`.** New reader producing per-egg
   `{ eggId, successNpc, msg[], aggScope[], isBonus, isNoGradeResult, finalConfirm }`.
   Serve via a `GET /data/incubator/{eggId}` (or `/data/gachapon-info`) resource. This
   yields the authoritative eligible-egg set + NPC + message templates.
2. **Config — per-egg reward pools.** Extend the `incubator-rewards` tenant resource
   with an `eggId` (region) dimension: pools are keyed by egg id `4170000–4170009`,
   each `[{itemId, quantity, weight}]`. Seed defaults per region.
3. **Channel handler.** On incubator use: read the sacrificed target's templateId →
   validate it is a known pigmy egg (from atlas-data) → roll *that egg's* tenant pool →
   award. Emit **version-gated `INCUBATOR_RESULT`**: flat for v83/84, extended (carry
   `gachaponItemID = eggId`, plus optional bonus) for v95 (+ any others that read it),
   so the client shows the correct Pigmy & Etran NPC.
4. **libs/atlas-packet — fix the `INCUBATOR_RESULT` codec** to the extended body on the
   versions that require it, re-verify the matrix cells (single-cell verify procedure).
5. **atlas-ui.** The Incubator Rewards admin page groups pools by egg/region; show the
   region label + NPC (from atlas-data) per pool.

## 6. Region map (interim — finalize from `incubatorInfo.img/{eggId}/su`)

Authoritative regions come from each egg's `su` NPC → NPC location (via atlas-data
npc-strings) during the `incubatorInfo.img` ingest. Until then, from the icon set
(`/tmp/pigmy.png`) with `4170005 = Ludibrium` anchored:

| id | icon | region (interim guess) |
|---|---|---|
| 4170000 | orange | Henesys |
| 4170001 | green, leafy | Ellinia |
| 4170002 | pale stone | Perion |
| 4170003 | tan speckled | Kerning City / Ariant |
| 4170004 | red/lava | El Nath / Sleepywood |
| 4170005 | gold scaled | **Ludibrium** (confirmed) |
| 4170006 | light blue | Orbis |
| 4170007 | glossy blue | Aqua Road / Aquarium |
| 4170009 | grey owl-ish | **Nautilus** (user) |

## 7. `4170008` — missing icon / item (must be resolved)

`4170008` returned no name from atlas-data item-strings while `4170000–4170007`,
`4170009` did, and the icon strip has no glyph for it. To resolve during ingest:
- If `incubatorInfo.img` contains an `4170008` entry → it is a real egg and the
  missing item-string/icon is an **atlas-data ingest gap** to fix (so the egg is
  usable).
- If `incubatorInfo.img` has no `4170008` entry → it is not a live egg; the eligible
  set is the 9 eggs `{4170000–4170007, 4170009}` and `4170008` is simply skipped.
The feature must key strictly off the eggs present in `incubatorInfo.img`, not a hard
`4170000–4170009` range, so whichever is true is handled correctly.

## 8. Verification

- Per-egg incubate awards from the correct region pool; sacrifice + incubator consumed.
- v95 client shows the correct Pigmy & Etran NPC dialog (proves the extended packet +
  `gachaponItemID`); v83/84 still shows its fixed-NPC dialog (flat packet).
- `INCUBATOR_RESULT` byte-fixture cells re-verified per version.
- Empty pool / ineligible target / full inventory → consumes nothing.

## Appendix — v95 IDA anchors (reference for cross-version work)

| function | v95 addr | role |
|---|---|---|
| `CUIIncubator::PutItem` | `0x7ca3f0` | target eligibility gate |
| `CUIIncubator::CUIIncubator` (ctor) | `0x7ca8b0` | dialog open (from `SendConsumeCashItemUseRequest` / `CDraggableItem::OnDoubleClicked`) |
| `CItemInfo::RegisterGachaponItemInfo` | `0x5bf040` | loads `Etc/incubatorInfo.img` |
| `CItemInfo::GetGachaponSucessNpc` | `0x5c06b0` | egg → success NPC (`su`) |
| `CItemInfo::GetGachaponItemIDByAggID` | `0x59dc50` | agg-scope → reward item ids |
| `CWvsContext::OnIncubatorResult` | `0xa00380` | result → modal NPC dialog (extended body) |
