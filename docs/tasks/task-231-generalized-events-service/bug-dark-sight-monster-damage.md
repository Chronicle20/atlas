# bug: monsters still damage a character in Rogue Dark Sight

**Reproduced:** tenant `15d000e1-260e-414b-bdf0-4c2b68b8995c`, GMS 83.1,
namespace `atlas-pr-1375`. Character 1 (Rogue-line, Dark Sight 4001003 at
level 20) casts Dark Sight and stands in a monster's path.

**Observed:** the character takes full touch damage while the Dark Sight buff is
active. From `atlas-channel` (times UTC, 2026-08-18):

```
13:31:52.661  Character [1] using skill [4001003] at level [20].
13:32:05.538  [CharacterDamageHandle] read [characterId [1], … nAttackIdx [-1], … damage [0] …]
13:32:24.679  [CharacterDamageHandle] read [characterId [1], … nAttackIdx [-1], … damage [1882] …]
13:32:24.687  Character [1] damage [1882] mitigated to hp [1882] mp [0] meso [0] reflect [0]
```

Level 20 Dark Sight lasts 200 s (confirmed by `atlas-buffs`: applied 13:21:05 →
`Expired buff for character [1] from [4001003]` 13:24:30, i.e. 205 s), so the
13:32:24 hit lands ~32 s into a 200 s window. An earlier hit at 13:28:04
(`damage [1910]`) likewise falls inside the 13:24:56 cast's window.

`nAttackIdx == -1` on every one of these — mob touch/collision damage, not a mob
skill.

**Expected:** monsters do not target or touch-damage a character in Dark Sight;
this is the skill's defining behavior.

**Root cause: NOT established.** Do not guess one. What is ruled out, with
evidence:

- **The WZ effect data is correct.** `GET /api/data/skills/4001003` on this
  tenant returns, for every level, `"statups": [{"type":"SPEED", …},
  {"type":"DARK_SIGHT","amount":1}]`. The amount is 1, not 0 — which matters,
  because the v83 client's `CUser::IsDarkSight` tests the stat `!= 0`
  (`services/atlas-channel/atlas.com/channel/skill/handler/hide/hide.go:148-150`).
- **The buff is applied and expires on schedule.** `atlas-channel` logs
  `Character [1] applying effect from source [4001003]` on each cast, and
  `atlas-buffs` logs the matching expiry at the correct 200 s offset.
- **`DARK_SIGHT` is registered in the temporary-stat encoder.**
  `libs/atlas-packet/model/character_temporary_stat.go:91` registers it in the
  mask (non-diseased, `NoOpForeignValueWriter`), and it appears in
  `foreignReadOrder` at line 1020 as flag-only. So the stat has a mask bit on
  v83 and a flag-only foreign shape.
- **It is not GM-hide interference.** `atlas-monsters`' buff consumer reacts
  only to `SuperGmHide` and passes Dark Sight through untouched
  (`services/atlas-monsters/atlas.com/monsters/kafka/consumer/buff/consumer.go:65-80`);
  `buff.IsGmHidden` keys on `SourceId`, not on the `DARK_SIGHT` stat
  (`services/atlas-channel/atlas.com/channel/character/buff/hidden.go:18-19`).
- **There is no server-side suppression to have regressed.** The damage-taken
  pipeline (`services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go`,
  `processDamageTaken`) has short-circuits for protective mist and for the
  `damage == -1` block sentinel, and a mitigation chain for Magic Guard /
  Power Guard / Achilles / etc. It has **no** `DARK_SIGHT` branch at all — this
  has never been implemented server-side.
- **Only one character was online** (character 1 throughout the session), so the
  attacking mob was controlled by the victim's own client. A remote controller
  not knowing about the buff cannot be the explanation here.

So the character's own client produced touch damage while its own Dark Sight was
active. That points at the local temporary-stat set the client received, but the
packet itself has not been inspected.

## Fix

**No fix should be written before the open question below is answered** — the
answer decides whether this is a packet-encoding defect or a missing
server-side rule, and those have disjoint fixes.

If it turns out to be a **packet-encoding defect** (the local set does not carry
`DARK_SIGHT`, or carries it with value 0):

- `libs/atlas-packet/model/character_temporary_stat.go` — the local (self)
  encode path for `DARK_SIGHT` on GMS 83.1.
- `libs/atlas-packet/model/character_temporary_stat_test.go` — a byte-fixture
  asserting the v83 local set carries the `DARK_SIGHT` bit with a non-zero
  value; must fail before and pass after.

If it turns out the client is correct and the server must **suppress mob touch
damage** while the buff is active:

- `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go`
  — a short-circuit in `processDamageTaken` alongside the existing protective-mist
  one, gated on an active `DARK_SIGHT` statup in `deps.getBuffs` and on
  `p.AttackIdx() == -1`.
- `extractBuffAmounts` in the same file — add the `DARK_SIGHT` case.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go`
  — a case driving `processDamageTaken` with a Dark Sight buff and
  `attackIdx -1`, asserting `changeHP` is never called; must fail before and
  pass after.

## Not yet answered

**The blocking question:** does the v83 client gate mob touch damage on
`CUser::IsDarkSight`, and does the local temporary-stat packet this server sends
actually set the `DARK_SIGHT` bit with a non-zero value?

Two independent checks, both required:

1. **Client side (IDA).** Per [`docs/reverse-engineering.md`](../../reverse-engineering.md),
   read the v83 `CUserLocal::SetDamaged` / mob touch path and determine whether
   it consults `IsDarkSight` before producing a damage report. If it does not,
   the client is behaving correctly and the server must suppress the damage
   (second fix shape above).
2. **Wire side.** Capture or byte-assert the local `TemporaryStatSet` the
   channel sends on the Dark Sight cast and confirm the `DARK_SIGHT` mask bit
   and its value. `libs/atlas-packet/model/character_temporary_stat_test.go:717-728`
   already covers a `DARK_SIGHT` fixture for the GM-hide source (9101004); the
   Rogue 4001003 path on v83 is what needs asserting.

Also unresolved: whether the two `damage [0]` reports at 13:32:05 and elsewhere
inside a Dark Sight window are a partial suppression (mob bumped but dealt
nothing) or unrelated. Do not treat "some hits are 0" as evidence the buff
partly works without establishing what produces a 0-damage report.
