# Player-Buff Periodic Tick Effects — Design

Task: task-214-buff-tick-effects
PRD: [`prd.md`](prd.md)
Status: Approved for planning
Created: 2026-08-12

---

## 1. Summary

atlas-buffs gains a declarative **periodic-effect table** and one generic tick
task that drives it. POISON stops being a hard-coded string scan and becomes a
row; DRAGON_BLOOD and RECOVERY become rows and start doing what their skill
descriptions say. The single-purpose `buffs-poison` Redis registry is replaced
by a `(characterId, statType)`-keyed last-tick store that is actually cleared on
cancel/expire.

All three open WZ questions from the PRD (§9 Q1, Q2, Q4) are resolved below
against the local v83 WZ dump — with the values quoted — so the plan phase has
no unverified inputs.

---

## 2. WZ verification (PRD §9 Q1, Q2, Q4)

Source: local extracted dump
`tmp/<tenant-uuid>/GMS/83.1/Skill.wz/*.img.xml` and
`tmp/<tenant-uuid>/GMS/83.1/String.wz/Skill.img.xml`
(see the WZ-inspection procedure in project memory; the same values appear in
all three tenant dumps present locally).

### Q1 — Dragon Blood `1311008`: `x` is the HP cost, `pad` is the attack bonus

`Skill.wz/131.img.xml`, node `1311008/level/*` carries exactly four keys:
`mpCon`, `x`, `time`, `pad`.

| level | mpCon | x | time | pad |
|---|---|---|---|---|
| 1 | 12 | 48 | 8 | 1 |
| 5 | 12 | 40 | 40 | 4 |
| 10 | 12 | 30 | 80 | 7 |
| 20 | 24 | 20 | 160 | 12 |

`x` **decreases** with level and `pad` increases. `String.wz/Skill.img.xml`
resolves it outright:

> `1311008` `h1` = `"Use 12 MP, 48 HP in every 4 seconds,  Attack + 1 in 8 Seconds"`

So at level 1: `mpCon`=12 MP, `x`=48 HP **per 4 seconds**, `pad`=+1 attack,
`time`=8 s duration. **`e.X()` is the per-tick HP cost.**

**Verdict:** `services/atlas-data/atlas.com/data/skill/reader.go:342` already
stores the correct field. No atlas-data change is required for FR-3.3, and the
`DRAGON_BLOOD` statup amount serves both the client icon and the drain
magnitude. The 4 s cadence in FR-3.2 is confirmed by the description string.

Also recorded: the skill's `desc` string reads *"Increases the attacking
capacity, but decreases HP steadily until 4 seconds before the remaining HP
exhausts."* That phrasing describes a **cancel-when-HP-would-exhaust** behavior
rather than the floor-at-1 behavior the PRD specifies (FR-3.4/FR-3.5). The PRD's
choice is deliberate and is what this design implements: floor at 1, buff stays
applied. The divergence is noted here so nobody re-derives it from the string
later and files it as a bug.

### Q2 — Recovery: `x` is HP per tick, cadence 5 s

`Skill.wz/1000.img.xml` node `10001001` (Noblesse Recovery; `2000.img.xml`
`20001001` is byte-identical for the Evan line):

| level | mpCon | time | x | cooltime |
|---|---|---|---|---|
| 1 | 5 | 30 | 4 | 120 |
| 2 | 10 | 30 | 8 | 120 |
| 3 | 15 | 30 | 12 | 120 |

`String.wz` `10001001`:

> `h1` = `"MP -10; Recover HP 24 in 30 sec."`
> `h2` = `"MP -20; Recover HP 48 in 30 sec."`
> `h3` = `"MP -30; Recover HP 72 in 30 sec."`

24 HP over a 30 s duration at `x`=4 per tick ⇒ 6 ticks ⇒ **one tick per 5
seconds**. Level 2 (8×6=48) and level 3 (12×6=72) confirm it. **`e.X()` is the
per-tick heal and the cadence is 5 s** — FR-4.2 and FR-4.3 both confirmed;
`reader.go:318` stores the right field.

Note the natural bound this gives us: `time`=30 s with `cooltime`=120 s means
Recovery produces **at most 6 ticks per cast, at most once every 2 minutes per
character**. It is not a permanent heartbeat. This is the deciding fact for D6
below.

### Q4 — INFINITY is not periodic in v83

`Skill.wz/212.img.xml` node `2121004/level/*` carries only `time`, `x`, and
`cooltime`. `x` is **1 at every one of the 30 levels**; there is no `y`, no
`hpR`/`mpR`, no per-tick resource field of any kind. `String.wz` `2121004`
`desc` = *"Enables one to temporarily draw magic powers from sources surrounding
the mage and use it in place of one's own MP"*, and every level string is
*"For N secs, Mana is left intacted."*

**Verdict: excluded.** v83 Infinity is MP-cost nullification plus a constant
flag, not a periodic HP/MP restore. (The periodic-restore-plus-escalating-magic-
attack version of Infinity is a later-version redesign; it is not in this WZ.)
Consequence: **no MP-resource row is in scope**, so no `CHANGE_MP` mirror
provider is added to atlas-buffs (PRD §5).

### Q3 — Dragon Blood tick packet

No packet work. The tick emits `CHANGE_HP` on `COMMAND_TOPIC_CHARACTER`;
atlas-character's `ChangeHP` unconditionally emits `STAT_CHANGED`
(`services/atlas-character/atlas.com/character/character/processor.go:1324`),
which is what repaints the HP bar. There is no separate self-damage-number
packet in the buff path — the existing POISON tick has used exactly this shape
in production, and the client renders poison ticks from it. Nothing extra is
needed for Dragon Blood, which is the same command with a different magnitude.

### Q5 — floor is per-row

Kept per-row as the PRD's current answer proposes. POISON keeps `floor: false`
(may reach 0 and emit `DIED`, exactly as today); DRAGON_BLOOD is `floor: true`.

---

## 3. Architecture

### 3.1 Component map

```
periodic/                (new package — pure, no I/O, no service deps)
  table.go               Effect rows + Lookup(statType) (Effect, bool)
  model.go               Effect, Resource, Direction types

character/
  registry.go            GetPeriodicEntries(ctx) []PeriodicEntry      (replaces GetPoisonCharacters)
                         Get/Update/ClearPeriodicTick(ctx, key)       (replaces *PoisonTick, keyed by (char, stat))
                         ClearPeriodicTicksFor(ctx, characterId, changes)  (FR-6.1 helper)
  processor.go           ProcessPeriodicTicks()                        (replaces ProcessPoisonTicks)
                         Cancel/CancelAll/CancelByStatTypes/expireInto call the FR-6.1 helper

tasks/
  periodic.go            PeriodicTick task                             (replaces tasks/poison.go)
  periodic_test.go       ported from tasks/poison_test.go

main.go                  tasks.Register(...)(tasks.NewPeriodicTick(l, 1000))
```

Dependency direction is one-way: `character` imports `periodic`; `periodic`
imports only `libs/atlas-constants/character` and `time`. No cycle, and the
table is unit-testable without Redis.

### 3.2 The table

```go
package periodic

type Resource string
type Direction int8

const ResourceHP Resource = "HP"

const (
    Drain   Direction = -1
    Restore Direction = +1
)

type Effect struct {
    statType character.TemporaryStatType
    interval time.Duration
    resource Resource
    direction Direction
    floor    bool   // true: never reduce the resource below 1
}
```

Rows (compile-time constants, FR-1.3):

| statType | interval | resource | direction | floor | source |
|---|---|---|---|---|---|
| `POISON` | 1 s | HP | Drain | false | preserves today's behavior (`processor.go:260`) |
| `DRAGON_BLOOD` | 4 s | HP | Drain | true | WZ `1311008` §2 Q1 |
| `RECOVERY` | 5 s | HP | Restore | false | WZ `10001001` §2 Q2 |

`Lookup(statType string) (Effect, bool)` is the **only** place a stat type
string appears in the tick path (FR-1.2). Rows are keyed by the
`libs/atlas-constants/character` constants
(`TemporaryStatTypePoison`, `…DragonBlood`, `…Recovery`), not by hand-written
literals — DOM-21.

`ResourceHP` is the only resource constant defined. The emitter's `switch`
carries a `default` arm that returns an error naming the unmapped resource, so
adding an MP row later fails loudly at the first tick instead of silently
emitting nothing. That default arm is a guard, not a stub: it is unreachable
with today's rows and its whole job is to be unreachable.

### 3.3 Scan (FR-2.1)

`Registry.GetPeriodicEntries(ctx)` does **one** traversal of
`characters.GetAllValues` — the same single pass `GetPoisonCharacters` does
today — and yields:

```go
type PeriodicEntry struct {
    Tenant      tenant.Model
    WorldId     world.Id
    ChannelId   channel.Id
    CharacterId uint32
    StatType    string
    Amount      int32
}
```

Two behavioral refinements over the poison scan:

- **All matching stat types per buff**, not `break`-after-first. A buff carrying
  both POISON and some future periodic stat yields two entries.
- **Deterministic dedupe by `(characterId, statType)`**: when two live buffs
  carry the same periodic stat type, the entry with the **largest `Amount`**
  wins. Today's code emits one entry per buff and lets the character-keyed
  throttle pick a random winner (Go map iteration order); max-wins removes the
  nondeterminism. For the single-buff case — every real case today — the result
  is byte-identical (FR-2.4).

### 3.4 Last-tick store (FR-2.2, D3)

One `atlas.TenantRegistry[TickKey, time.Time]` under namespace `buffs-tick`:

```go
type TickKey struct {
    CharacterId uint32
    StatType    string
}
// keyFn: strconv.FormatUint(uint64(k.CharacterId), 10) + ":" + k.StatType
```

Access stays entirely on `libs/atlas-redis` (`Get` / `PutWithTTL` / `Remove`),
so `tools/redis-key-guard.sh` is satisfied by construction.

Entries are written with **`PutWithTTL`, TTL 5 minutes**. Every live row
refreshes its key at most 5 s apart, so an active entry never lapses; an entry
whose owning buff vanished by a path we failed to wire evaporates on its own.
This is belt-and-braces for FR-6.2 — the explicit clears in FR-6.1 are still
wired, the TTL just means a missed clear degrades to "stale for ≤5 min" instead
of "leaked forever".

### 3.5 Tick pass (FR-2.3)

One task, `tasks.NewPeriodicTick(l, 1000)`, registered where `NewPoisonTick`
is today. Sleep stays 1000 ms — the shortest row's cadence (FR-2.3). Per pass,
per tenant (`GetTenants` → `routine.Go` → `tenant.WithContext`, unchanged from
`ProcessPoisonTicks`, FR-2.5):

```
entries := GetPeriodicEntries(ctx)
now := time.Now()
message.Emit(...)(func(buf) error {
    for each entry:
        eff, ok := periodic.Lookup(entry.StatType); if !ok { continue }
        last, has := GetPeriodicTick(ctx, TickKey{entry.CharacterId, entry.StatType})
        if has && now.Sub(last) < eff.interval { continue }
        magnitude := entry.Amount
        if magnitude == 0 { continue }                       // FR-1.5
        amount := int16(eff.direction) * int16(magnitude)
        if eff.floor && amount < 0 {
            hp := hpFor(entry.CharacterId)                   // memoized per pass, §3.6
            if hp <= 1 { debug-log suppressed; continue }    // FR-3.4
            if int32(hp) + int32(amount) < 1 {
                amount = -(int16(hp) - 1)
            }
        }
        buf.Put(EnvCommandTopicCharacter, changeHPCommandProvider(...))
        UpdatePeriodicTick(ctx, key, now)
})
```

Cadence aliasing: a 1 s pass driving a 4 s row gives 4 s ± <1 s of drift. That
is the same property today's poison tick has, and HP-per-4-s is not a
tight-tolerance contract. Not worth a per-row scheduler (YAGNI).

Ordering with redelivery (NFR): the throttle read and the store update straddle
the `buf.Put`, exactly as today. A crash between `Put` and `UpdatePeriodicTick`
can re-tick on the next pass; a crash before `Put` skips a tick. Both are
one-interval errors on a non-idempotent HP mutation, which is the same window
POISON has carried in production. Making this exactly-once would require an
idempotency key on the `CHANGE_HP` command — an atlas-character contract change
and out of this task's scope; recorded, not silently inherited.

### 3.6 Current-HP read (FR-3.6, NFR load)

`extchar.RequestById(characterId)(l, ctx)` → `RestModel.Hp` (`uint16`) — the
same client the berserk path already uses
(`berserk/processor.go:54`). Two bounds:

- **Lazy**: the call happens only inside the `eff.floor` branch, i.e. only for a
  character with a floor-sensitive row that is actually **due** this pass.
  Characters with only POISON/RECOVERY rows make no call.
- **Memoized per pass**: a `map[uint32]uint16` local to the pass, so a character
  with two floor rows fetches once (FR-3.6).

A fetch error is logged and the tick is **skipped** for that character this
pass, never emitted unclamped — failing closed is the only safe direction for a
drain that must not kill.

**Residual race, stated plainly:** the HP read is a snapshot. If the character
takes other damage between our read and atlas-character applying our reduced
negative, the sum can still land on 0 and emit `DIED`. Closing that window needs
a "cannot be lethal" flag on `ChangeHPCommandBody` — an atlas-character contract
change the PRD puts out of scope (§7: atlas-character None). The window is one
Kafka hop wide and only reachable for a character already being hit hard enough
to die that instant. Accepted and documented; not silently ignored.

### 3.7 Lifecycle clearing (FR-6.1)

`ClearPeriodicTicksFor(ctx, characterId, changes []stat.Model)` removes the tick
key for every change whose type is in the table. It is called from all four
removal paths in `character/processor.go`, each of which already holds the
removed buffs' `Changes()`:

| path | line today |
|---|---|
| `Cancel` | `processor.go:78` |
| `CancelAll` | `processor.go:108` |
| `CancelByStatTypes` | `processor.go:131` |
| `expireInto` (used by both `ExpireBuffs` and `ExpireForCharacter`) | `processor.go:212` |

`ClearPoisonTick` — zero callers today (`registry.go:344`) — is deleted along
with the rest of the poison-specific surface. The acceptance criterion is a test
that cancelling a periodic buff leaves no tick key, so the zero-caller state
cannot recur.

Channel/world change resets nothing (FR-6.3): keys are character-scoped and the
emitted command carries `m.worldId` / `m.channelId` read fresh from the
character model on each scan.

### 3.8 Migration

The `buffs-poison:*` key set is abandoned in place. Entries are ephemeral
throttle state; an orphaned set is self-evidently dead and costs a few bytes per
recently-poisoned character. No migration step, per PRD §6. The plan will carry
one line noting the abandoned prefix.

---

## 4. Decisions and alternatives

**D1 — Table in its own `periodic` package, driven from `character`.**
Alternative (a): put the table in `character/` next to the scan — fewer files,
but the table stops being unit-testable without Redis and `character/` grows
another responsibility. Alternative (b): a full `periodic/` package with its own
processor + registry, mirroring `berserk/` — but the buff store lives in
`character.Registry`, so a separate processor would either duplicate the scan or
force `character` to export its buff map. **Chosen: hybrid** — pure table in
`periodic`, scan and emit stay in `character` where the store and the Kafka
providers already are. The diff stays close to a rename of the poison path.

**D2 — Composite-key single registry over one registry per stat type.**
One registry per stat type means a new registry (and a new `InitRegistry` line)
per table row — exactly the per-effect ceremony this task exists to delete. The
composite key is one `keyFn` and scales to rows for free.

**D3 — 1 s driving task with per-row interval gates**, not a per-row scheduler
or one task per row. Preserves POISON's cadence exactly (FR-2.4), keeps one
scan pass (NFR load), and the aliasing cost is bounded by the pass interval.

**D4 — Floor by reducing the amount, not by cancelling the buff.**
Per PRD FR-3.4/FR-3.5, against the WZ description string's implied
cancel-on-exhaust (§2 Q1). Reducing the amount also keeps atlas-buffs from
inventing a cancel reason and firing an `EXPIRED` the client did not ask for.

**D5 — Fail closed on the HP read.** A drain that cannot resolve current HP is
skipped, not emitted raw. One missed 4 s tick is invisible; one unintended
`DIED` is not.

**D6 — Recovery emits unconditionally; no overheal suppression.**
Suppressing requires effective MaxHP in atlas-buffs, which is an
`exteffstats.RequestByCharacter` call per healing character per 5 s — trading
one outbound call for one avoided command. The deciding fact is §2 Q2: Recovery
is `time`=30 s on a `cooltime`=120 s, so it is **at most 6 ticks per cast per
character**, not a standing heartbeat. Paying a REST call to dodge at most 6
commands is a net loss. PRD FR-4.5's explicit fallback ("emit unconditionally
and record the decision") is taken, and this is the record. atlas-character
clamps at effective MaxHP in `enforceBounds`
(`processor.go:1305`), so no HP is created.

**D7 — Max-amount dedupe per `(character, statType)`** over first-wins, because
first-wins over a Go map is nondeterministic (§3.3).

**D8 — TTL on tick keys in addition to explicit clears**, not instead of them.
FR-6.1 requires the wiring; the TTL bounds the blast radius of a future removal
path that forgets to call it.

---

## 5. FR-5 audit sweep — every statup, with a verdict

Scope of the enumeration: every `character.TemporaryStatType*` constant
referenced anywhere under `services/atlas-data/atlas.com/data/` (58 distinct
types — this is the authoritative set atlas-data can attach to a character
buff), plus the disease stat types reaching atlas-buffs from the mob-skill and
consumable paths.

Verdicts: **wired** (table row in this branch) · **deferred** (periodic but
owned by another task) · **excluded** (not a timer-driven change to the
character's own HP/MP).

### 5.1 Wired

| Stat type | Cadence | Evidence |
|---|---|---|
| `POISON` | 1 s, HP drain, no floor | existing behavior, `character/processor.go:253-279`; magnitude is target-derived at apply time, tick just re-emits `Amount()` |
| `DRAGON_BLOOD` | 4 s, HP drain, floor 1 | WZ `1311008` §2 Q1; `reader.go:342` |
| `RECOVERY` | 5 s, HP restore | WZ `10001001` §2 Q2; `reader.go:318` |

### 5.2 Deferred

| Stat type | Owner | Reason |
|---|---|---|
| `COMBO_DRAIN` | task-166 (`.worktrees/task-166-combo-drain`) | WZ `21100005` levels carry `x`=1 and `time` only; `String.wz` `21100005` = *"will recover 1% of HP Damage dished out as HP"* — proportional to damage dealt, driven by attack events, not a timer. Confirms the PRD's expected verdict. |

### 5.3 Excluded — not periodic

Grouped by why. Every row's citation is the atlas-data line that produces the
statup; the WZ-verified ones carry their node too.

**Checked individually against WZ (the PRD's "needs a verdict" list):**

| Stat type | Verdict | Evidence |
|---|---|---|
| `INFINITY` | excluded | WZ `2121004`: `time`/`x`=1/`cooltime` only, no resource field at any of 30 levels; `desc` = MP-cost nullification. §2 Q4. `reader.go:354` |
| `BODY_PRESSURE` | excluded | WZ `21101003`: `prop` (success rate), `x`=neutralize seconds, `damage`=body-attack %; `String.wz` confirms *"success rate: neutralize for 5 sec, Body Attack Damage 5%"*. Outbound mob effect, nothing periodic on the caster's resources. `reader.go:478` |
| `MESO_GUARD` | excluded | on-damage meso substitution; fires when hit. `reader.go:389` |
| `MAGIC_GUARD` | excluded | on-damage HP→MP redirection; fires when hit. `reader.go:348` |
| `PICK_POCKET` | excluded | on-attack meso drop. `reader.go:391` |
| `HOMING_BEACON` | excluded | attack-targeting marker. `reader.go:407` |
| `PUPPET` | excluded | spawns a field object; lifetime owned by the summon path, not a resource tick. `reader.go:369` |
| `SUMMON` | excluded | same — summon lifecycle. `reader.go:421` |

**Flat combat-stat modifiers** — a constant folded by atlas-effective-stats for
the buff's lifetime; nothing changes between apply and expire:
`WEAPON_ATTACK` (`reader.go:237`), `WEAPON_DEFENSE` (`:238`), `MAGIC_ATTACK`
(`:239`), `MAGIC_DEFENSE` (`:240`), `ACCURACY` (`:241`), `AVOIDABILITY` (`:242`),
`SPEED` (`:243`), `JUMP` (`:244`), `BOOSTER` (`:415`), `MAPLE_WARRIOR` (`:419`),
`SHARP_EYES` (`:379`), `CONCENTRATE` (`:371`), `HYPER_BODY_HP` (`:332`),
`HYPER_BODY_MP` (`:333`), `WHITE_KNIGHT_CHARGE` (`:340`), `ELEMENTAL_RESET`
(`:360`), `MAGIC_SHIELD` (`:362`), `MAGIC_RESIST` (`:364`), `SPEED_INFUSION`
(`:405`), `DASH_SPEED` (`:402`), `DASH_JUMP` (`:403`), `WIND_WALK` (`:381`),
`ECHO_OF_HERO` (`:320`), `HOLY_SYMBOL` (`:352`), `MESO_UP` (`:385`),
`SHADOW_PARTNER` (`:387`), `WIND_BREAKER_FINAL` (`:346`), `SPARK` (`:409`),
`ARAN_COMBO` (`:470`), `COMBO_BARRIER` (`:472`), `BARRIER` (`:229`),
`STANCE` (`:344`).

**Event-driven or state flags** — evaluated on an incoming/outgoing action or
read as a boolean, never on a timer: `POWER_GUARD` (`:330`, on-damage
reflection), `MANA_REFLECTION` (`:356`, on-damage), `HOLY_SHIELD` (`:358`,
immunity flag consumed by `hasImmunityBuff`, `character/immunity.go:25`),
`INVINCIBLE` (`:350`), `DIVINE_BODY` (`:328`), `SMART_KNOCK_BACK` (`:476`),
`COMBO` (`:335`, crusader combo counter, incremented on attack via
`UpdateStatValue`), `DARK_SIGHT` (`:383`), `SOUL_ARROW` (`:367`),
`SHADOW_CLAW` (`:399`, placeholder overwritten by atlas-channel at cast),
`HAMSTRING` (`:373`) and `BLIND` (`:376`) (outbound monster statuses),
`THAW` (`:233`, map-protection flag), `MORPH` (`:488`), `MONSTER_RIDING`
(`:576`, vehicle id).

**Disease stat types** reaching atlas-buffs from mob skills / consumables
(`character/immunity.go:7-11`; consumable cure mapping
`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:134-139`):
`STUN`, `SEAL`, `DARKNESS`, `WEAKEN`, `CURSE`, `SEDUCE`, `CONFUSE`, `UNDEAD`,
`SLOW`, `STOP_PORTION` — all excluded: stat/ability suppression for the
duration, no resource tick. `POISON` is the sole periodic member of that set and
is wired above.

**FR-5.5 result: no statup was found to be periodic-and-unowned.** Nothing is
filed as a follow-up.

---

## 6. Testing

Package `periodic` (no I/O): table lookup returns the row for each wired stat
type and `false` for an arbitrary non-row type; every row's interval and
direction asserted explicitly (a row edited by accident fails a test, not
production).

Package `character` (miniredis + injected clock and HP fetcher, mirroring
`berserk/processor.go`'s `now func() time.Time` / injected-client shape):

- POISON parity: one character, `Amount` 25 ⇒ emitted `ChangeHPCommandBody`
  `Amount` = -25 on the first pass, suppressed on a pass 500 ms later, emitted
  again at 1 s. Ported from `tasks/poison_test.go` assertions (FR-2.4).
- Zero magnitude ⇒ no command (FR-1.5).
- Two effects, independent throttles: POISON (1 s) + DRAGON_BLOOD (4 s) on one
  character ⇒ pass at t=0 emits both; t=1 s emits only POISON; t=4 s emits both.
- Dragon Blood floor: HP 100 / drain 48 ⇒ -48. HP 30 / drain 48 ⇒ -29. HP 1 ⇒
  no command emitted. HP fetch error ⇒ no command emitted.
- Recovery: positive `Amount` emitted unclamped by atlas-buffs (FR-4.4), and no
  HP fetch performed for a Recovery-only character (NFR load bound).
- Dedupe: two live buffs both carrying POISON with amounts 10 and 25 ⇒ exactly
  one command, `Amount` -25.
- Lifecycle: after `Cancel` / `CancelAll` / `CancelByStatTypes` / expiry of a
  periodic buff, `GetPeriodicTick` reports no entry for that
  `(character, statType)` and no key remains under the `buffs-tick` namespace
  (FR-6.1, FR-6.2).
- Multi-tenancy: two tenants with same-id characters throttle independently.

Package `tasks`: `periodic_test.go` ports the three existing poison-task tests
(`SleepTime` honors the configured interval; `SleepTime` millisecond math;
`Run` does not panic against an empty miniredis tenant set).

---

## 7. Verification

Per CLAUDE.md, from the worktree root: `go test -race ./...` and `go vet ./...`
clean in `services/atlas-buffs`, plus `tools/redis-key-guard.sh`,
`tools/goroutine-guard.sh`, and `tools/lint.sh --check` clean from the repo
root. `go.mod` is not expected to change — no new dependency is introduced — so
`docker buildx bake atlas-buffs` is not triggered; if the plan phase finds
otherwise, the bake becomes mandatory.

Note for the reviewer: `tools/buff-duration-guard.sh` is unaffected — this task
adds no `duration` field to any `COMMAND_TOPIC_CHARACTER_BUFF` body.

---

## 8. Out of scope (restated, so it isn't re-litigated)

Berserk (HP-threshold, not periodic — keeps its own task and cache);
Combo Drain (task-166); any change to buff apply/store/expire semantics or to
`APPLIED`/`EXPIRED` event shapes; packet or client work; tenant- or
version-configurable intervals; an idempotency key on `CHANGE_HP` (§3.5);
a non-lethal flag on `ChangeHPCommandBody` (§3.6).
