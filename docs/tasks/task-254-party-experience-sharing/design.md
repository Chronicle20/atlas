# Party EXP Sharing — Design

Task: `task-254-party-experience-sharing`
PRD: [prd.md](prd.md) (approved, v1)
Status: Draft for review
Created: 2026-08-21

---

## 0. Reading guide

Section 1 records what the code actually does today, because three PRD premises do not survive
contact with the source and the plan must be built on the corrected picture. Section 2 is the target
architecture. Section 3 is the decision log (D1–D16), including the resolutions for OQ-1..OQ-5.
Section 4 is the file-by-file change map. Section 5 is the test strategy. Section 6 lists the two
items that need the user's call before `/plan-task`.

---

## 1. Ground truth — corrections to PRD premises

These are code readings, not opinions. Each is quoted from the source in the worktree.

### 1.1 FR-1.1's stated defect does not exist on the live damage path

The PRD asserts that `ModelBuilder.AddDamageEntry`
(`services/atlas-monsters/atlas.com/monsters/monster/builder.go:139-146`) appends per hit and
therefore `Model.damageEntries` holds one element per damaging hit. That is true of the *builder*,
but the builder's append path is reached only from `Model.Damage()`
(`services/atlas-monsters/atlas.com/monsters/monster/model.go:215-222`), and **`Model.Damage()` has
no production caller**. A repo-wide sweep for `.Damage(` inside `atlas-monsters` returns exactly one
non-test hit — `kafka/consumer/monster/consumer.go:108`, which calls `Processor.Damage`, not
`Model.Damage`.

The authoritative write path is `Registry.ApplyDamage`
(`services/atlas-monsters/atlas.com/monsters/monster/registry.go:436-495`), which **already**
aggregates:

```go
for i := range cur.DamageEntries {
    if cur.DamageEntries[i].CharacterId == characterId {
        cur.DamageEntries[i].Damage += actual
        cur.DamageEntries[i].LastHitMs = nowMs
        found = true
        break
    }
}
if !found {
    cur.DamageEntries = append(cur.DamageEntries, storedDamageEntry{...})
}
```

`fromStored` (`registry.go:186-207`) aggregates a second time on read, defensively, for legacy rows.
`registry_test.go:1030-1053` already pins one-entry-per-character with summed damage and
`LastHitMs = max`. Every live damage source — character attack (`processor.go:564-597`) and
damage-over-time (`status_task.go:93`) — goes through `ApplyDamage`.

**Consequence.** FR-1.1 is a defence-in-depth change to a code path that production does not
execute, not a bug fix. It is still worth doing (it makes `Model.Damage()` correct if it is ever
revived, and it makes `DamageSummary()`'s doc comment true through *both* paths), but it is cheap
and low-risk, and it must not be described in the plan as "the fix that makes party EXP work".
The PRD's FR-1.4 defence in `atlas-monster-death` remains required and is the one that actually
matters at the service boundary.

**OQ-4 is answered here.** No `atlas-monsters` test asserts per-hit append semantics.
`clear_aggro_test.go:51` expects 3 entries from 3 *distinct* characters, `aggro_task_test.go:168`
expects 1 entry from 1 character, and `model_test.go:23-51` constructs pre-aggregated entries
directly. All remain green under aggregation.

### 1.2 FR-3.2's justification is wrong; FR-3.1 is still the right choice

FR-3.2 argues that `Σ damage entries == HP actually removed` because `Model.Damage` clamps each hit
to remaining HP, and therefore healing is naturally absorbed. The clamp claim is true
(`ApplyDamage` clamps identically), but the conclusion does not hold, because **damage entries
decay**.

`Registry.DecayDamageEntries` (`registry.go:713-745`), driven by `MonsterAggroDecayTask` every
`AggroSweepInterval = 1500ms` (`aggro.go:26`), multiplies any entry idle for more than
`AggroIdleThresholdMs = 10_000` by `AggroDecayMultiplier = 0.85` per tick and prunes it entirely
once it falls below `AggroDecayFloor = 1`. So at kill time `Σ entries ≤ HP removed`, and for a
character who stopped attacking twenty seconds before the kill it is materially less.

FR-3.1 is nonetheless correct, for a *different* reason than the PRD gives: `totalDamage` is only
ever used as the denominator of `personalRatio` and of `experiencePerDamage`. Setting it to
`Σ entries` makes the per-participant shares sum to exactly `1.0`, so the monster's EXP is neither
over- nor under-distributed. Setting it to `mi.Hp()` (template max HP) makes the shares sum to
`> 1` whenever the monster was healed and `< 1` whenever entries decayed — the current behaviour,
and the actual bug. The design adopts FR-3.1 and restates its rationale as **normalisation**, not
heal-absorption.

The decay interaction has one visible consequence worth stating in the plan: a party member whose
contribution decayed to zero and was pruned is no longer a *contributor*, so they neither appear in
`partyDamage` nor widen the leech interval (§2.5). That is consistent with how aggro decay already
governs drop ownership and control, and is accepted rather than worked around.

### 1.3 `totalPartyLevel byte` overflows

`distributeCharacterExperience` takes `totalPartyLevel byte`
(`services/atlas-monster-death/atlas.com/monster/monster/processor.go:243`). Today it is only ever
passed the character's own level, so it never overflows. The moment it becomes a party sum it does:
six level-200 members sum to 1200, which wraps to 176 in a `byte`, inflating every member's share
by ~6.8×. The signature must widen to `uint32` (D6). This is not mentioned anywhere in the PRD and
is the single highest-severity latent defect in the change.

### 1.4 The `COMMAND_TOPIC_SYSTEM_MESSAGE` deployment change is already done

PRD §5.2 and §7 call for adding the env var to `atlas-monster-death`'s deployment.
`deploy/k8s/base/atlas-monster-death.yaml:20-22` uses `envFrom: configMapRef: atlas-env`, and
`deploy/k8s/base/env-configmap.yaml:92` already defines `COMMAND_TOPIC_SYSTEM_MESSAGE`. **No
deployment change is required** for the topic. (New level-gate env keys do need a configmap entry —
see D9.)

### 1.5 Observation, out of scope

`atlas-monster-death`'s local `awardExperienceCommandProvider`
(`monster/character/producer.go:11-40`) emits no `transactionId` and no `showEffect`, while
`atlas-character`'s `AwardExperienceCommandBody`
(`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:112-116`) reads both.
Today's messages therefore carry `transactionId = uuid.Nil` and `showEffect = false`. It also
appends a `PARTY` distribution with `Amount: 0` on every solo award. PRD §2 explicitly forbids
changing the `AWARD_EXPERIENCE` contract, so this design **does not** touch it. Recorded so the
reviewer does not re-discover it as a regression introduced here.

---

## 2. Target architecture

### 2.1 Shape: resolve → plan (pure) → award

The current `DistributeExperience` interleaves I/O and arithmetic in one loop, which is why the
existing tests only cover the two free functions that happen to be pure
(`calculateExperienceStandardDeviationThreshold`, `isWhiteExperienceGain`). The PRD's acceptance
criteria demand assertions on MVP determinism, `totalEntries`, interval-union semantics, exclusion
sets, and lookup counts — none of which are reachable without a seam.

The processor is restructured into three phases:

```
DistributeExperience(f, monsterId, damageEntries)
  ├─ 1. RESOLVE  (all I/O, concurrent where independent)
  │      monster information  → exp, hp, level, name
  │      characters in field  → set[characterId]   (authoritative co-location)
  │      party per in-field damager → party.Model with members
  │
  ├─ 2. PLAN     (pure, no I/O, deterministic)
  │      planDistribution(input ExperienceInput, cfg ExperienceConfig) ExperiencePlan
  │        → []Recipient{characterId, level, pooledExp, totalPartyLevel,
  │                      partyBonusMod, isMvp, white}
  │        → []Exclusion{characterId}
  │
  └─ 3. AWARD    (I/O, per recipient, failure-isolated)
         for each recipient (ascending characterId):
           rate := rates.GetForCharacter(...)
           personal, bonus := computeAward(recipient, rate, cfg)   // pure
           character.AwardExperience(ch, id, white, personal, bonus)
         for each exclusion (ascending characterId):
           systemmessage.ShowHint(...)   // throttled, D10
```

`planDistribution` and `computeAward` are pure functions over value types. Every numeric acceptance
criterion in the PRD becomes a table-driven unit test with no mocks, no goroutines, and no clock.
Mocks are then needed only for the two criteria that are genuinely about I/O — lookup counts and
tenant-header propagation.

**Rejected alternative — keep `produceDistribution` returning `DamageDistributionModel` and add a
party branch to the award loop.** This is the minimum diff and was the first candidate. It was
rejected because `DamageDistributionModel` cannot carry the per-party derived values
(`totalPartyLevel`, `partyBonusMod`, MVP, exclusions) without becoming the plan type anyway, and
because it leaves every party rule untestable except through mocks of five collaborators. The
migration cost is one file's worth of restructuring in a service with 382 lines of existing test.

**Rejected alternative — a `distribution` sub-package.** `monster/experience.go` +
`monster/interval.go` inside the existing `monster` package keeps `calculateExperienceStandard
DeviationThreshold` and `isWhiteExperienceGain` reachable without export churn, and keeps the two
existing test files valid. A sub-package would force exporting them.

### 2.2 Dependency injection

`monster.ProcessorImpl` currently constructs every collaborator inline
(`character.NewProcessor(p.l, p.ctx)`, `party.NewProcessor(...)`, `rates.GetForCharacter(...)`,
`information.GetById(...)`, `_map.CharacterIdsInFieldModelProvider(...)`). The repo's established
pattern for making these injectable is the `With(...)` option form used by `atlas-pets`
(`services/atlas-pets/atlas.com/pets/pet/processor.go:94-118`,
`processor_revive_test.go:44-48`):

```go
p := monster.NewProcessor(l, ctx).With(
    monster.WithPartyProcessor(pm),
    monster.WithCharacterProcessor(cm),
    monster.WithRatesProcessor(rm),
    monster.WithInformationProcessor(im),
    monster.WithFieldProcessor(fm),
    monster.WithSystemMessageProcessor(sm),
    monster.WithExperienceConfig(cfg),
)
```

`With` shallow-copies the struct and applies the options, returning a new `Processor`. Fields are
nil in production and fall through to `NewProcessor`-time defaults bound in the constructor —
`atlas-pets`' comment at `processor.go:106-114` documents the one hazard (never bind a *method
value* at construction time, or a `With` clone dispatches to the original receiver); we bind
*processor interfaces*, not method values, so the hazard does not apply.

`character` and `party` already expose `Processor` + `mock` in this service and are used as-is. Three
collaborators are currently free functions and gain the same shape:

| Package | New interface | Rationale |
|---|---|---|
| `rates` | `Processor{ GetForCharacter(ch channel.Model, characterId uint32) Model }` | needed for the "one rate lookup per recipient" assertion |
| `monster/information` | `Processor{ GetById(monsterId uint32) (Model, error) }` | needed to inject mob level/name in tests |
| `map` | `Processor{ CharacterIdsInField(f field.Model) ([]uint32, error) }` | needed to control the co-location set |

Each gets a sibling `mock/processor.go` matching the existing `character/mock` and `party/mock`
shape (nil-func fallthrough to a zero value). The existing free functions stay as thin wrappers so
`CreateDrops` and the drop path are untouched.

### 2.3 Bucketing and co-location (resolves OQ-5)

The PRD (FR-5.4) proposes deciding co-location from `MemberRestModel.WorldId/ChannelId/MapId/
Instance`, and OQ-5 asks how stale that is. `atlas-parties` maintains member location from
`CHANGE_MAP`/`MAP_CHANGED` Kafka events
(`services/atlas-parties/atlas.com/parties/kafka/consumer/character/consumer.go:155`,
`character/processor.go:293`) — eventually consistent, typically sub-second, but not synchronous
with the kill.

**Decision (D11): do not use party member location for co-location at all.** The distributor
already fetches the authoritative co-location set — `_map.CharacterIdsInFieldModelProvider`
(`map/provider.go:20-30`) drains `atlas-maps`' paginated character list for the exact
`world/channel/map/instance` of the kill, and is the same set the current code uses to gate damage
entries. A party member is co-located **iff their `characterId` is in that set**.

This removes the staleness question entirely, adds zero REST calls, and needs no `field.Model`
comparison. `MemberRestModel` is still needed — for `Level` (FR-5.6, FR-6.1) and for enumerating
members who dealt no damage — but its location fields are not consulted. `Online` is likewise
redundant: an offline character is not in the field set. We deserialise `Online` anyway (it is free,
the reference implementation carries it) but do not branch on it; FR-5.5 is satisfied *a fortiori*.

Bucketing then reads:

```
inField := set from atlas-maps
partyOf := map[characterId]partyId       // memoised
parties := map[partyId]party.Model       // memoised

for each distinct characterId in damageEntries (ascending):
    if characterId not in inField:  → out-of-field damager (D12)
    if characterId already resolved via a previously fetched party's member list: reuse
    else: pt, err := party.GetByMemberId(characterId)
          err != nil  → warn, treat as solo (FR-2.3)
          pt.Id() == 0 → solo
          else memoise parties[pt.Id()] = pt and every member's partyId
```

Because `GetByMemberId` returns the full party including its member list, memoising by `partyId`
collapses an N-member party to **one** lookup, not N. Total lookups per kill =
`#distinct in-field solo damagers + #distinct participating parties`, which is at or below FR-2.4's
ceiling and satisfies NFR-1's stronger dedup-by-partyId phrasing.

### 2.4 Party arithmetic

For each `partyId` in ascending order:

| Quantity | Definition |
|---|---|
| `contributors` | in-field party members with a damage entry, ascending `characterId` |
| `partyDamage` | `Σ contributor damage` — **not** gated by level (D14) |
| `participationExp` | `partyDamage × experiencePerDamage` — the pooled figure |
| `expMembers` | party members ∩ `inField`, minus level-gate exclusions (§2.5) |
| `mvp` | member of `expMembers` with greatest damage (0 if none); ties → lowest `characterId` (D13) |
| `totalPartyLevel` | `Σ level of expMembers`, as `uint32` (D6); skip party if 0 |
| `hasPartySharers` | `len(expMembers) > 1` |
| `partyBonusMod` | `0.05 × len(expMembers)` when `hasPartySharers`, else `0.0` |

Each member of `expMembers` becomes a `Recipient` with `pooledExp = participationExp`. A party with
empty `expMembers` is skipped silently (FR-5.10). A one-member party is numerically identical to
solo — `0.8 × L / L + 0.2 = 1.0`, `partyBonusMod = 0` — and is asserted as such (FR-5.11).

`totalEntries` (the stddev divisor) = `#in-field solo damagers + #distinct participating parties +
#out-of-field damagers`, per FR-2.5. `entryExperienceRatio` receives one element per solo damager
and one per party (the sum of its contributors' personal ratios); out-of-field damagers contribute
to `totalDamage` and to `totalEntries` but produce no ratio element, matching the current code and
Cosmic's "independent party" handling.

### 2.5 Level gate — interval union

A new `monster/interval.go` ports `<cosmic>/src/main/java/tools/IntervalBuilder.java` as a small
value type:

```go
type intervalSet struct{ ivs []interval }   // interval{lo, hi int}

func (s *intervalSet) add(lo, hi int)   // clamps lo at 0
func (s intervalSet) build() intervalSet // sorts by lo, merges overlapping/adjacent
func (s intervalSet) contains(v int) bool // linear scan; ≤7 intervals per party
```

Merging on `build()` rather than on `add()` keeps `add` O(1) and the merge a single sort pass. Sizes
here are trivially small (one mob interval + at most six contributor intervals), so a linear
`contains` is correct and clearer than a binary search.

The gate, applied **only** to the party path (FR-6.3):

```
s := intervalSet{}
s.add(mobLevel - LevelInterval, mobLevel + LevelInterval)
for each contributor c:  s.add(c.level - LeachInterval, c.level + LeachInterval)
s = s.build()
expMembers = { m ∈ coLocatedMembers : s.contains(int(m.level)) }
excluded   = coLocatedMembers \ expMembers
```

Arithmetic is done in `int` with a `lo < 0 → 0` clamp, because mob level is `uint32`, member level
is `byte`, and `5 - 3` under either unsigned type wraps. The PRD's worked example (contributors at
30 and 120, mob at 125, `LEACH_INTERVAL = LEVEL_INTERVAL = 5`) yields the merged set
`{[25,35], [115,130]}` — admits 32, rejects 70 — and becomes a direct table-test row.

Gate ordering matters and is fixed here: **the gate selects recipients only.** `partyDamage`,
`participationExp`, `contributors`, and the interval set itself are all computed *before* the gate,
from the ungated contributor list. A gated-out member who dealt damage still widens the interval for
everyone else and still contributes to the pool they do not share in — which is Cosmic's behaviour
(`Monster.java:549-600`) and is what makes the interval union meaningful.

### 2.6 Award computation

```go
func computeAward(r Recipient, expRate float64, cfg ExperienceConfig) (personal uint32, bonus uint32)
```

1. `exp := r.pooledExp * expRate` — rate applied **before** the split, preserving today's ordering
   so that FR-8.2 (bonus is also rate-multiplied) falls out for free.
2. `share := cfg.SplitCommonMod * float64(r.level) / float64(r.totalPartyLevel)`
   (`SplitCommonMod = 0.8`).
3. `if r.isMvp { share += cfg.MvpMod }` (`MvpMod = 0.2`).
4. `personalF := share * exp`; `bonusF := r.partyBonusMod * personalF`.
5. Guard (FR-8.6): if either of `personalF`/`bonusF` is `NaN` or `±Inf`, log at warn with
   `monsterId` + `characterId` and award `0` for that value rather than casting. `uint32(NaN)` is
   implementation-defined in Go and must never reach the wire.
6. Cast to `uint32`.

The three magic numbers (`0.8`, `0.2`, `0.05`) move into `ExperienceConfig` alongside the gate
settings, satisfying the PRD's OQ-1 aside about naming the existing literals.

### 2.7 SHOW_HINT producer

A new `system_message` package in `atlas-monster-death`, mirroring `character/` exactly
(`kafka.go` constants + `Command[E]` + `ShowHintBody`, `producer.go` provider, `processor.go`
`Processor`/`ProcessorImpl`/`NewProcessor`, `mock/processor.go`). The struct definitions are copied
from `services/atlas-channel/atlas.com/channel/kafka/message/system_message/kafka.go:26-68` — copied
rather than imported, because per CLAUDE.md we do not reach across a service boundary for another
service's internals, and every other cross-service contract in this repo is duplicated the same way.

`ShowHint` publishes via `producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(...)` with
`producer.CreateKey(int(characterId))`, which is what carries tenant and span headers (§6.3) — the
same call shape as `character.AwardExperience`. `TransactionId` is a fresh `uuid.New()` per hint.

Hint text (FR-6.8), with `Width = 0`, `Height = 0`:

```
You have gained #rno experience#k from defeating #e#b<name>#k#n (lv. #b<level>#k)! Take note you must have around the same level as the mob to start earning EXP from it.
```

Publish failures are logged at warn and never abort the loop (FR-6.10) — hints are emitted after all
awards, so a hint failure cannot affect EXP at all.

### 2.8 Concurrency and failure isolation

Resolve-phase calls that are independent run concurrently: monster information and the field
character list have no ordering relationship and are issued together. Party lookups are inherently
sequential-with-memoisation (each result may satisfy later characters), and at ≤6 distinct parties
per kill the serial cost is not worth the dedup complexity of a parallel form; they stay in a loop.
Award and hint loops are serial, deterministic, and each iteration's error is captured and logged
without breaking the loop (FR-9.2, NFR-2).

The whole thing still runs inside the existing `routine.Go` goroutine in
`kafka/consumer/monster/consumer.go:63-89`, alongside drop creation. No change there.

### 2.9 Determinism

Every map is drained into a sorted slice before iteration: parties ascending by `partyId`, members
and solo characters ascending by `characterId`, `entryExperienceRatio` sorted ascending before the
stddev computation (float summation is associativity-sensitive, so this is what makes the threshold
byte-reproducible, not merely statistically stable). MVP tie-break is lowest `characterId`. The
`ExperiencePlan` returned by `planDistribution` is therefore a deterministic function of its input,
which is what NFR-5's "assert determinism, not merely observe it once" needs: the test asserts
`planDistribution(in, cfg)` is deeply equal across repeated calls **and** across input slices
presented in shuffled order.

---

## 3. Decision log

**D1 — Pure planner + thin I/O shell.** §2.1. Chosen over a minimal-diff party branch because every
PRD acceptance criterion about numbers becomes a mock-free table test.

**D2 — `With(...)` option-based DI on `monster.ProcessorImpl`.** §2.2. Follows `atlas-pets`. Rejected
alternative: passing collaborators as `NewProcessor` parameters — that breaks every existing
`monster.NewProcessor(l, ctx)` call site including `CreateDrops`'s consumer.

**D3 — Add `Processor` + `mock` to `rates`, `monster/information`, and `map`.** §2.2. Rejected
alternative: declaring narrow consumer-side interfaces in the `monster` package. That is idiomatic Go
but is not what this repo does; `character` and `party` in this very service already own their
`Processor` + `mock`, and matching them keeps `backend-guidelines-reviewer` (DOM/SUB rules) happy.

**D4 — Keep `calculateExperienceStandardDeviationThreshold` and `isWhiteExperienceGain` unchanged**
(FR-4.4). They are already pure and already tested; only their *inputs* change.

**D5 — `totalDamage = Σ aggregated damage entries`,** with the rationale restated as share
normalisation rather than heal absorption (§1.2). Zero total damage → warn + return (FR-3.3).

**D6 — Widen `totalPartyLevel` to `uint32`** in `distributeCharacterExperience` / `Recipient`
(§1.3). Non-negotiable correctness fix; the `byte` signature silently inflates every party award.

**D7 — `atlas-monsters` `AddDamageEntry` aggregates** (FR-1.1), summing `Damage` and taking
`max(LastHitMs)`, preserving the first-contact index — identical semantics to `fromStored`
(`registry.go:186-207`), so the two aggregation sites agree. Framed as defence-in-depth per §1.1,
not as the enabling fix.

**D8 — OQ-3 resolved: the consumer sweep.** Consumers of `damageEntries` off `DAMAGED`/`KILLED`:

| Consumer | Use | Effect of aggregation |
|---|---|---|
| `atlas-quest` `kafka/consumer/monster/consumer.go:52` | iterates entries, credits one kill per entry | **Improved.** Under per-hit entries a multi-hit character would be credited once per hit; under the registry's existing aggregation it is already correct, and D7 keeps it correct on the builder path too. |
| `atlas-monster-death` | this task | rewritten |
| `atlas-channel` `kafka/message/monster/kafka.go:205-211` | carries `DamageEntries` on `DAMAGED`; its own doc comment already warns not to read the last element as "damage this event" | no change |
| `atlas-maps`, `atlas-consumables` | REST/DTO passthrough of the monster's entry list | no change |

No consumer requires modification. The doc comment at `atlas-channel/.../kafka.go:205-209` is the
scar tissue OQ-3 suspected, and it was already fixed by the `Damage` field it references.

**D9 — OQ-1 resolved: `ExperienceConfig`, defaults in Go constants, overridable by env.** A struct
built once at bootstrap and injected via `WithExperienceConfig`:

| Setting | Env var | Default | Source |
|---|---|---|---|
| `EnforceMobLevelRange` | `USE_ENFORCE_MOB_LEVEL_RANGE` | `true` | `<cosmic>/config.yaml:243` |
| `LevelInterval` | `LEVEL_INTERVAL` | `5` | `<cosmic>/config.yaml:293` |
| `LeachInterval` | `LEACH_INTERVAL` | `5` | `<cosmic>/config.yaml:294` |
| `SplitCommonMod` | — | `0.8` | `processor.go:232` |
| `MvpMod` | — | `0.2` | `processor.go:235` |
| `PartyBonusPerMember` | — | `0.05` | `Monster.java` |

Env keys are read with `os.Getenv` + parse-or-default, so an absent key is not an error, and are
added to `deploy/k8s/base/env-configmap.yaml` for discoverability. The last three have no env key —
they are game-balance constants, not deployment knobs, and giving them one invites drift between
replicas of the same world. Chosen over pure Go constants (FR-6.5 asks for a toggle and a
compile-time toggle is a weak one) and over tenant configuration (no precedent for EXP tuning in
this service; would add a `configurations` dependency to a service that has none).

Injecting the config also means the gate-disabled acceptance test is a one-line `With` override
rather than an env mutation.

**D10 — OQ-2 resolved: in-process, per-replica hint throttle.** A `hintThrottle` value inside the
`system_message` processor (or `monster` package — plan's call), keyed by
`(tenantId, characterId)`, storing last-emit `time.Time`, with a 1-minute window and an injectable
`now func() time.Time`. On insert, if the map exceeds a cap (4096 keys), entries older than the
window are swept. Guarded by a `sync.Mutex`; contention is negligible at kill rates.

`atlas-monster-death` runs `replicas: 2` (`deploy/k8s/base/atlas-monster-death.yaml:6`), so worst
case is one hint per replica per minute per character — bounded, and far better than one per kill.
Chosen over: (a) no throttle, rejected because a level-30 parked in a level-120 grind party gets a
hint per kill, which is the spam the PRD itself calls likely unacceptable; (b) shared state in Redis,
rejected as a new infrastructure dependency for a cosmetic notice; (c) relocating the notice to
`atlas-channel`, rejected because it would require a new event carrying the exclusion reason —
strictly more contract surface for the same result.

**D11 — Co-location comes from `atlas-maps`, not from party member location.** §2.3. Resolves OQ-5
by making it moot. FR-5.4 and FR-5.5 are satisfied by a stronger and cheaper mechanism.

**D12 — Out-of-field damagers count as entries but do not join a party pool.** They are never
party-resolved (no lookup issued), they contribute to `totalDamage` and add `1` each to
`totalEntries`, and they receive nothing. This follows FR-2.1 ("for each damage entry whose
character is present in the field") and FR-5.3, and matches Cosmic's independent-party accounting.
**It contradicts one PRD acceptance bullet** — see §6, item 1.

**D13 — MVP is chosen from `expMembers`, not from contributors.** `mvp = argmax(damage)` over
`expMembers`, treating a non-damager as `0`, ties broken by lowest `characterId`. This guarantees
exactly one MVP whenever a party has any recipient, so the `+0.2` stays inside the party instead of
evaporating when the top damager left the field or was level-gated out. A literal reading of FR-5.1
would pick the MVP from the damager set; that reading can leave a party with no MVP at all.

**D14 — The level gate does not shrink `partyDamage`.** §2.5. Gated-out members' damage still feeds
the pool and their level still widens the interval for others.

**D15 — Rate applied before the split, once per recipient.** §2.6. Preserves the existing ordering
that FR-8.2 depends on, and yields exactly one `rates` call per recipient (FR-8.4). No
`PARTY_BONUS_EXP_RATE` knob (FR-8.3).

**D16 — No `atlas-parties`, `atlas-channel`, `atlas-character`, `atlas-rates`, or deployment
changes.** §1.4 removes the only deployment item; `atlas-parties` already serves every field
needed; `atlas-channel` already consumes `SHOW_HINT`; `atlas-character` already handles `PARTY`.

---

## 4. Change map

### `atlas-monsters` (secondary — 2 files + tests)

| File | Change |
|---|---|
| `monster/builder.go` | `AddDamageEntry` aggregates by `characterId`: sum `Damage`, `max` `LastHitMs`, preserve first-contact index (D7) |
| `monster/model.go` | `DamageSummary()` doc comment: state that aggregation now holds on both the registry and builder paths |
| `monster/builder_test.go` | new cases: two calls same character → one summed entry; two characters → two entries in first-contact order; `LastHitMs` takes the max |
| `monster/model_test.go` | `DamageLeader()` over a builder-constructed 3×100 / 1×250 pair returns the 300 character (PRD acceptance) |

### `atlas-monster-death` (primary)

| File | Change |
|---|---|
| `monster/party/model.go` | add `MemberModel{id, name, level, jobId, field, online}` + accessors; `Model.members []MemberModel` + `Members()` |
| `monster/party/rest.go` | full `members` relationship: `GetReferencedIDs`, `GetReferencedStructs`, `SetToManyReferenceIDs`, `SetReferencedStructs`, `MemberRestModel`, `ExtractMember`. Port verbatim from `services/atlas-channel/atlas.com/channel/party/rest.go` (adding `Instance`, which the channel copy omits from its `SetToManyReferenceIDs` seed) |
| `monster/information/{model,builder,rest}.go` | add `level uint32`, `name string` + accessors, builder setters, `Extract` wiring |
| `monster/information/processor.go` (new) | `Processor{GetById}` wrapping the existing free function |
| `monster/information/mock/processor.go` (new) | mock |
| `rates/processor.go` (new) | `Processor{GetForCharacter}` wrapping `GetForCharacter` |
| `rates/mock/processor.go` (new) | mock |
| `map/processor.go` (new) | `Processor{CharacterIdsInField}` wrapping the drain provider |
| `map/mock/processor.go` (new) | mock |
| `system_message/{kafka,producer,processor}.go` (new) | `SHOW_HINT` command + `Processor{ShowHint}` |
| `system_message/mock/processor.go` (new) | mock |
| `system_message/throttle.go` (new) | `hintThrottle` (D10) |
| `monster/monster/interval.go` (new) | `intervalSet` port of Cosmic `IntervalBuilder` |
| `monster/monster/experience.go` (new) | `ExperienceConfig`, `ExperienceInput`, `Recipient`, `Exclusion`, `ExperiencePlan`, `planDistribution`, `computeAward` |
| `monster/monster/config.go` (new) | env-backed `ExperienceConfig` loader with the D9 defaults |
| `monster/monster/processor.go` | `With(...)` options + injected collaborator fields; `DistributeExperience` rewritten as resolve → plan → award; `distributeCharacterExperience` folded into `computeAward` + the award loop; `totalPartyLevel` widened to `uint32`; both `// TODO` comments removed |
| `monster/monster/model.go` | `DamageDistributionModel` retired in favour of the plan types, or reduced to what `produceDistribution`'s successor still needs — plan's call; whichever survives gains `Party()` / `TotalDamage()` per PRD §6.1 |
| `main.go` | build `ExperienceConfig` at bootstrap and pass it to the consumer's processor construction |
| `kafka/consumer/monster/consumer.go` | thread the config into `monster.NewProcessor(...).With(...)` |
| `deploy/k8s/base/env-configmap.yaml` | add `USE_ENFORCE_MOB_LEVEL_RANGE`, `LEVEL_INTERVAL`, `LEACH_INTERVAL` |

---

## 5. Test strategy

**Layer 1 — pure, table-driven, no mocks** (`monster/experience_test.go`, `monster/interval_test.go`).
Covers: pooled-vs-personal EXP, the `0.8·L/ΣL (+0.2)` split, `partyBonusMod` at 1/2/4 members, the
one-member-party ≡ solo identity (asserted numerically, FR-5.11), MVP tie-break, `totalEntries`
composition, white/yellow selection, zero-`totalDamage`, zero-`totalPartyLevel`, NaN/Inf guard, the
interval-union worked example (30/120 contributors, 125 mob, admit 32, reject 70), gate-disabled,
and determinism under shuffled input.

**Layer 2 — processor with mocks** (`monster/processor_experience_test.go`). Covers only what is
about I/O: one party lookup per distinct party, one rate lookup per recipient, party-lookup error →
solo path + warn, `SHOW_HINT` publish failure → remaining awards still emitted, exactly one hint per
excluded member per kill, `party > 0` on party recipients and `party == 0` on solo, and hint
throttling across two kills inside the window.

**Layer 3 — producer shape** (`system_message/producer_test.go`). Asserts the marshalled command
matches the PRD §5.2 JSON exactly: `type == "SHOW_HINT"`, `width == 0`, `height == 0`, key derived
from `characterId`, text interpolated with monster name and level. Tenant/span header propagation is
covered by using the same `producer.ProviderImpl` path as `character` — asserted structurally (same
call shape), since headers are attached by the shared library, not by this code.

**Layer 4 — `atlas-monsters` builder** as listed in §4.

All setup uses the existing `NewBuilder()` forms; no `*_testhelpers.go` file is introduced.

Gate: flagless `tools/verify.sh` must exit 0, and `backend-guidelines-reviewer` must pass on both
`services/atlas-monster-death` and `services/atlas-monsters`.

---

## 6. Needs the user's call before `/plan-task`

1. **Out-of-field damagers and the party pool (D12).** PRD FR-2.1 and FR-5.3 say party resolution is
   for in-field damagers and that an out-of-field damager receives nothing; but one PRD acceptance
   bullet says such a member "still contributes their damage to `partyDamage` and to
   `totalEntries`". Those cannot both hold — crediting their damage to `partyDamage` requires
   resolving their party, which FR-2.1 forbids. This design implements the FR-2.1/FR-5.3 reading
   (Cosmic-faithful: they count toward `totalDamage` and `totalEntries`, and their EXP share is
   simply not distributed). **Recommendation: adopt D12 and amend the acceptance bullet.** The
   alternative — resolve parties for all damagers and pool their damage — is a ~10-line difference
   and a leech vector (tag a mob, walk away, still boost the party pool).

2. **MVP selection source (D13).** A literal FR-5.1 picks the MVP from the party's damager set, which
   can leave a party with no MVP when the top damager left the field or was level-gated out, quietly
   dropping 20% of the pool. This design picks the MVP from `expMembers`.
   **Recommendation: adopt D13.**

Everything else in the PRD is implemented as specified, with the premise corrections in §1 folded in.
