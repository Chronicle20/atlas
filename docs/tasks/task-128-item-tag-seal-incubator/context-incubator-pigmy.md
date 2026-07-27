# Context — Incubator Pigmy-Egg correction

Companion to `plan-incubator-pigmy.md` (spec: `design-incubator-pigmy.md`). Scope is the
incubator sub-feature only; item-tag and sealing-lock are untouched.

## Key files (current state)

| File | Role | Change |
|---|---|---|
| `libs/atlas-packet/incubator/clientbound/result.go` | `INCUBATOR_RESULT` codec — already version-gated (flat v83/84/87/jms, extended v95) but hard-codes the v95 `gachaponItemID` to `0` | Task 1: carry the egg id |
| `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` | `IncubatorRewardRestModel{Id,ItemId,Quantity,Weight}` + `ExtractIncubatorReward` | Task 2: add `EggId` |
| `services/atlas-tenants/configurations/incubator-rewards/*.json` | seed pool (6 rows) | Task 2: add `eggId` |
| `services/atlas-channel/atlas.com/channel/incubator/{rest,roll,processor,requests}.go` | reads `/tenants/{id}/configurations/incubator-rewards`; `Reward{itemId,quantity,weight}`; `PickWeighted` | Task 3: add `EggId` + `GetRewardsForEgg`/`FilterByEgg` |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (incubator arm ~252–300) | rolls one flat pool; ignores the sacrificed target | Task 4: egg guard + per-egg pool + emit egg id |
| `services/atlas-ui/src/{services/api/incubator-rewards.service.ts, lib/schemas/incubator-rewards.schema.ts, pages/tenants-incubator-rewards-form.tsx}` | flat rewards admin page | Task 5: per-egg (region) |
| `docs/packets/audits/` v95 `IncubatorResult` cell | byte fixture | Task 6: re-verify with the egg id populated |

## Decisions (from the user)

1. **Reward pools are tenant-config, keyed per egg (region).** No WZ pigmy reward table exists (`incubatorInfo.img` holds NPC/msg/aggScope, not a flat reward list), so pools stay tenant-defined.
2. **`incubatorInfo.img` ingest is NOT required** for correct behavior — the client resolves the region NPC from its own copy once the server sends the egg id. Ingest is Phase 7 (optional) for authoritative region labels + eligible-egg set + `4170008` resolution.
3. **Region map is interim** (icon-derived; `4170005 = Ludibrium`, `4170009 = Nautilus`); authoritative labels come from `incubatorInfo.img/{eggId}/su → NPC → town` if Phase 7 is done.

## Key facts (IDA / data verified)

- Incubator item `5060002` ("Incubator", cash type 27). Eligible targets = Pigmy Eggs `4170000–4170009` (`4170008` has no item-string in current data — resolve in Phase 7).
- v95 client: `OnIncubatorResult @0xa00380` reads `int itemId, short count, int gachaponItemID, int bonusItemID, int bonusCount`; `GetGachaponSucessNpc(gachaponItemID)` picks the region NPC; `CUtilDlgEx.DoModal` shows the Pigmy & Etran dialog. v83/84/87/jms read the flat `itemId+count` only (IDA-re-verified in the codec comment).
- Client eligibility (`CUIIncubator::PutItem @0x7ca3f0`, v95): target id must be in the gachapon-agg range; the real eggs are `4170000–4170009`. Server had no check → Task 4 adds one.

## Dependencies / sequencing

Task 1 (packet) → Task 4 (handler uses the 3-arg constructor). Task 2 (config `eggId`) → Task 3 (reader) → Task 4 (handler) and Task 5 (UI). Task 6 (verify) after Task 1. No `go.mod` changes → no docker bake. Lands on the `task-128-item-tag-seal-incubator` branch / PR #909.
