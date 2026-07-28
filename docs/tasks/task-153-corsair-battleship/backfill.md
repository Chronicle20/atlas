# Battleship live-tenant config backfill

Seed templates apply only at tenant creation — existing tenants do NOT pick
up the new writer options (known gotcha: new opcodes/options never reach
live tenant configs automatically), and atlas-channel does not hot-reload
socket configuration.

## Scope

Nine of the ten live tenants need the backfill: GMS v61, v72, v79, v83, v84,
v87, v92, v95 and JMS v185.

GMS v48 is **excluded**: skill 5221006 does not exist in that client (its WZ
data returns 404 and the binary contains no reference to the skill or to the
5221999 gauge sentinel). GMS v12 has no live tenant. Neither is a gap.

## Per-tenant steps

1. Fetch the tenant's channel socket configuration from atlas-tenants
   (`GET /tenants/{tenantId}/configurations/{resourceName}` for the channel
   socket resource, via the admin UI or REST).
2. In `socket.writers`, add to the existing entries (do not change opCodes):
   - `CharacterBuffGive` → `"options": {"vehicles": {"CORSAIR_BATTLESHIP": 1932000}}`
   - `CharacterSkillCooldown` → `"options": {"skills": {"BATTLESHIP_HP_GAUGE": 5221999}}`
3. GMS v87, v92, v95, JMS v185 ONLY: these live configs are also missing
   `CharacterUseSkillHandle` and `CharacterDamageHandle` entirely (their seed
   templates were). Without both, the cast never reaches the server and damage
   is never processed — the feature is inert. Add to `socket.handlers`, at the
   sorted opCode position, with `"validator": "LoggedInValidator"` and
   `"services": ["channel"]`:
   - v87: `CharacterUseSkillHandle 0x5E`, `CharacterDamageHandle 0x32`
   - v95: `CharacterUseSkillHandle 0x67`, `CharacterDamageHandle 0x34`
   - JMS v185: `CharacterUseSkillHandle 0x56`, `CharacterDamageHandle 0x27`
   - v92: `CharacterUseSkillHandle 0x66`, `CharacterDamageHandle 0x35`
   (v87/v95/jms185 values come from docs/packets/registry/<version>.yaml and
   were cross-validated against the already-verified entries in the same
   files; v92 has no registry column; its two values were derived from its IDB and
   triple-cross-checked — see plan.md Task 11 Step 3.)
4. GMS v92 ONLY: the live config may also be missing the five buff/cooldown
   writers entirely (its seed template was). If absent, add:
   `CharacterBuffGive 0x21` (with the vehicles options),
   `CharacterBuffCancel 0x22`, `CharacterBuffGiveForeign 0xE3`,
   `CharacterBuffCancelForeign 0xE4`,
   `CharacterSkillCooldown 0x112` (with the skills options).
   These opcodes are CSV-derived (docs/packets/MapleStory Ops -
   ClientBound.csv), cross-validated against the other versions' verified
   template entries.
5. PATCH the configuration back, then restart atlas-channel (handlers and
   writers are read at listener build; the config projection does not
   hot-reload them).
6. Verify per tenant on a live client: mount/dismount visuals (self +
   foreign), gauge movement under damage, break → dismount + cooldown +
   greyed icon, remount-while-cooling rejected, Cannon/Torpedo on foot
   rejected (debug log `battleship_attack_rejected_not_riding`).

## Known blocker: GMS v95 has no ingested skill data

The v95 tenant returns `maxLevel: 0, effects: []` for skill 5221006 — and for
every other skill probed (5221004, 5221007, 5221008, 5121000, 1001003). This
is a tenant-wide WZ ingestion gap that predates this task, not a battleship
defect. Until v95 skill data is ingested, `GetEffect` yields nothing on that
tenant: no statup set, no MP cost, no cooldown value, so the mount cannot
apply. Do the config backfill anyway, then record v95 live verification as
BLOCKED — do not report it as verified.

## RESOLVED: GMS v92 SPECIAL_MOVE packet body shape

This was carried as an open blocker while the branch was in review. It has
since been checked against the client and is **not** a defect — v92's cast
path decodes correctly. Recorded here so the question is not re-opened.

The original concern was that v92's `SPECIAL_MOVE` body looked like
`Encode4(time), Encode4(skillId), Encode1(SLV), Encode2(?), Encode2(?)` —
a 5-byte trailer — where v95's looked like `... Encode1(FindParty),
Encode2(0)`, a 4-byte trailer, implying `SkillUsageInfo` might silently
decode garbage on v92.

**That comparison was between two different senders, not two versions.**
`SPECIAL_MOVE` (v92 opcode `0x66`) has no single per-version body: the
trailer is chosen **per skill category**. Read out of `GMS_v92_1_DEVM.exe`,
v92 alone carries at least three distinct shapes:

| v92 sender | body after the opcode | bytes |
|---|---|---|
| `sub_91B8C0` (ctor `0x91B9B6`) | `Encode4`(time) `Encode4`(skillId) `Encode1`(SLV) | 9 |
| `sub_91B630` (ctor `0x91B7C2`) | + `Encode2`(castX) `Encode2`(castY) | 13 |
| `sub_91B130` | + `Encode2`(x) `Encode2`(y) `Encode1` `Encode1` | 15+ |

Every v92 sender shares the same 9-byte prefix — `Encode4(update_time),
Encode4(skillId), Encode1(SLV)` — and so does v95's real
`CUserLocal::DoActiveSkill_Heal` (`0x93A830`). The decoder is gated on skill
id, not on version, and Battleship (5221006) is in none of
`isAntiRepeatBuffSkill` / `isPartyBuff` / `isMobAffectingBuff`, so it
consumes exactly that 9-byte prefix and stops. The skill-use handler reads
nothing after `SkillUsageInfo`, so any longer trailer is inert.

Pinned by `TestDecodeCorsairBattleshipV92Prefix` in
`libs/atlas-packet/model/skill_usage_info_test.go`, which asserts both the
bare 9-byte body and the 13-byte castX/castY body decode to the same
skill id and level.

**Trap for anyone re-treading this:** v92's `sub_91B630` was labelled
`DoActiveSkill_Heal` by propagation from v95. It is not — it contains no
`FindParty` call and writes castX/castY where v95's real Heal writes
`Encode1(FindParty)` + `Encode2(0)`. Comparing those two functions is what
produced the incorrect belief that v92's body diverged. Do not trust a v92
symbol name propagated from v95 without checking the body.

## Ship HP is version-dependent

When eyeballing the gauge, expect different full-pool values either side of
v87. The server mirrors each client's own `get_max_durability_of_vehicle`:

- GMS v61–v84: `200 × (charLevel + 2×SLV − 120)`
- GMS v87+ and JMS: `300 × charLevel + 500 × (SLV − 72)`

A level-200 character with SLV 10 starts at 20 000 on v83 and 29 000 on v95.
Battleship is maxLevel 10 on every version.

Full sweep required — do not spot-check one tenant and declare all nine done.
