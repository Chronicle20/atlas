# v0.48 GM (job 500) / SuperGM (job 510) skill-range coverage + misfire assessment

Task-187 grounding-audit normalization pass (Task B/C). Enumerates every
`5001xxx`/`5101xxx` skill wire id present in the v48 WZ snapshot
(`libs/atlas-constants/gen/wzsnapshot/gms_48_1.json`; `gms_12_1.json`
confirmed byte-identical for this id range), drains its v48 WZ name, and
matches it by name to the canonical GM/SuperGM skill identity in
`libs/atlas-constants/skill/identities_gen.go`.

## Coverage result: all 9 wire ids map cleanly — zero unmapped

| wireId | v48 name (drained, tenant `e1f06ae2-...` major=48) | Mapped identity | Canonical token |
|---|---|---|---|
| 5001000 | Haste (Normal) | `GmHaste` | 9001000 |
| 5001001 | Super Dragon Roar | `GmSuperDragonRoar` | 9001001 |
| 5001002 | Teleport | `GmTeleport` | 9001002 |
| 5101000 | Heal + Dispel | `SuperGmHealDispel` | 9101000 |
| 5101001 | Haste (Super) | `SuperGmHaste` | 9101001 |
| 5101002 | Holy Symbol | `SuperGmHolySymbol` | 9101002 |
| 5101003 | Bless | `SuperGmBless` | 9101003 |
| 5101004 | Hide | `SuperGmHide` | 9101004 |
| 5101005 | Resurrection | `SuperGmResurrection` | 9101005 |

Every one of these 9 wire ids now has an explicit `divergences.csv` override
row (5101004→`SuperGmHide` was already present from Task 1; the other 8 were
added in this pass, each duplicated for `gms_12` per the confirmed-identical
snapshot id-set). **No wire id in this range required Task C's
un-mappable-wire fallback analysis** — there is nothing left over to assess.

## Why this matters: the would-have-misfired set (documented for context, not escalation)

Task C's brief asks, for any wire that *couldn't* be mapped, whether
auto-bind's fallback (binding the wire id to whatever v83-canonical
identity happens to share that exact token) would misfire onto a real
attack/handler skill. Since every wire above *was* mapped, this fallback
never actually executes — but checking what auto-bind *would have* produced
absent the override confirms the overrides are load-bearing, not
belt-and-suspenders:

| wireId | v48 meaning | Canonical identity at same token (if any) | Would auto-bind misfire? |
|---|---|---|---|
| 5001000 | Gm Haste | *(none)* | No — graceful miss (unbound) |
| 5001001 | Gm Super Dragon Roar | `PirateFlashFist` | **Yes** — real skill identity |
| 5001002 | Gm Teleport | `PirateSommersaultKick` | **Yes** — real skill identity |
| 5101000 | SuperGm Heal+Dispel | *(none)* | No — graceful miss (unbound) |
| 5101001 | SuperGm Haste | *(none)* | No — graceful miss (unbound) |
| 5101002 | SuperGm Holy Symbol | `BrawlerBackspinBlow` | **Yes** — gates a stun monster-status effect (`services/atlas-data/atlas.com/data/skill/reader.go:353`) |
| 5101003 | SuperGm Bless | `BrawlerDoubleUppercut` | **Yes** — same stun-status gate as above |
| 5101004 | SuperGm Hide | `BrawlerCorkscrewBlow` | **Yes** — IDA-verified keydown attack skill, `skill.IsKeyDownSkill` member (`libs/atlas-constants/skill/model.go:74`) — the original PRD-motivating bug |
| 5101005 | SuperGm Resurrection | `BrawlerMPRecovery` | **Yes** — has a live registered channel handler, `channelhandler.Register(skill2.BrawlerMPRecoveryId, Apply)` (`services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery.go:19`) |

6 of the 9 wire ids sit on a canonical Pirate/Brawler skill token and would
have silently misfired without an explicit override (one of them —
5101004/`BrawlerCorkscrewBlow` — is the exact bug the task-187 PRD was
written to fix). All 6 (plus 5101004, already covered) now have grounded
override rows in `divergences.csv`. The remaining 3 (5001000, 5101000,
5101001) have no canonical-token collision and would have failed gracefully
(left unbound) rather than misfiring — they are still overridden for
correctness/completeness, not because they were at risk.

## Verdict

**Clean.** No un-mappable v48 5001xxx/5101xxx wire id exists in this range
— every one resolved to a real canonical GM/SuperGM skill identity by exact
WZ-name match. No generator "absent-marker" design decision is needed for
this range. No escalation.
