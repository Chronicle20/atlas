# Zombify Healing Consequences — Implementation Plan

Input: `docs/tasks/task-256-zombify-healing-consequences/design.md` (v1)
PRD: `docs/tasks/task-256-zombify-healing-consequences/prd.md`

Four tasks, executed in order. Tasks 1→2 are `atlas-consumables`; Tasks 3→4 are
`atlas-channel`. Task 2 depends on Task 1 (it calls the new
`buff.Processor.GetByCharacterId`). Task 4 depends on Task 3 (it calls the new
`buff.IsZombified`). Tasks 1/2 and 3/4 are independent of each other.

No FR-17 code is written — design OQ-1 established `HPConsume == 0` for skill
2301002 on all ten provisioned tenants, so the reference's `hpCon = 0` forcing is
already a structural no-op at `skill/handler/common.go:137`.

---

## Task 1: `atlas-consumables` read-side buff client + `IsZombified`

`atlas-consumables/character/buff` is write-only today (Kafka producers). Add the
REST read half, modeled on `atlas-channel`'s client — including its
`ErrNotFound` → empty-slice normalization, which `atlas-monsters`' simpler client
omits and which this path needs because most characters carry no buffs at all
(design D2).

### Files

- `services/atlas-consumables/atlas.com/consumables/character/buff/requests.go` — **new file**; `RootUrlFor(ctx, "BUFFS")` + `characters/%d/buffs`
- `services/atlas-consumables/atlas.com/consumables/character/buff/rest.go` — **new file**; `RestModel` + `Extract`
- `services/atlas-consumables/atlas.com/consumables/character/buff/stat/rest.go` — **new file**; `stat.RestModel` + `stat.Extract`
- `services/atlas-consumables/atlas.com/consumables/character/buff/model.go` — **new file**; read-side `Model`, `NewBuff`, `Expired`, `Changes`, `IsZombified`
- `services/atlas-consumables/atlas.com/consumables/character/buff/model_test.go` — **new file**
- `services/atlas-consumables/atlas.com/consumables/character/buff/processor_notfound_test.go` — **new file**
- `services/atlas-consumables/atlas.com/consumables/character/buff/processor.go` — add `GetByCharacterId` to `Processor` and `ProcessorImpl`
- `services/atlas-consumables/atlas.com/consumables/character/buff/mock/processor.go` — add `GetByCharacterIdFunc` + method
- `services/atlas-consumables/atlas.com/consumables/character/buff/stat/model.go` — read-only; the existing `stat.Model{Type character.TemporaryStatType; Amount int32}` is the extract target, do not change it

Patterns to copy, in order of authority. `atlas-channel`'s buff package is the
template file-for-file: requests.go lines 1-28 (the bare-URL shape and its
comment); rest.go lines 1-49 (`RestModel` field set, `Extract` via
`model.SliceMap`); stat/rest.go lines 1-13 (`stat.RestModel`); model.go lines
22-73 (`Model`, `Expired()` honouring `noExpiry`, `NewBuff`); processor.go lines
46-76 (`ByCharacterIdProvider`'s `errors.Is(err, requests.ErrNotFound)`
normalization and its comment — fold it directly into `GetByCharacterId`, since
consumables needs no separate provider method); model.go lines 10-20 (`IsMount`,
the single-buff predicate `IsZombified` generalizes to a slice). For the
slice-level predicate shape — skip `Expired()`, return on first match — see
`HasActiveGmHide` at lines 38-49 of
services/atlas-monsters/atlas.com/monsters/character/buff/model.go.

Module root for `go build ./... && go test ./...`:
`services/atlas-consumables/atlas.com/consumables`

Type note the implementer must not paper over: this service's `stat.Model.Type`
is an already-typed **`character.TemporaryStatType` struct field**, so the
comparison is `c.Type == charconst.TemporaryStatTypeUndead` with **no** string
cast — unlike `atlas-channel`, where `stat.Model.Type()` returns a `string` and
the cast is required. Import the constants package as `charconst` to avoid
reading as the sibling `atlas-consumables/character` package.

### Steps

- [ ] **Step 1: Write the failing tests**

`model_test.go` — package `buff`. Two functions.

`TestIsZombified` — table-driven. Build each `Model` with the new
`NewBuff(sourceId, level, duration, changes, createdAt, expiresAt, noExpiry)`
constructor; build `stat.Model` as a plain struct literal
(`stat.Model{Type: charconst.TemporaryStatTypeUndead, Amount: 1}`).

| subtest name | buffs | expect |
|---|---|---|
| `unexpired undead` | one buff, changes `[{UNDEAD, 1}]`, `expiresAt = time.Now().Add(time.Minute)`, `noExpiry false` | `true` |
| `expired undead` | one buff, changes `[{UNDEAD, 1}]`, `expiresAt = time.Now().Add(-time.Second)`, `noExpiry false` | `false` |
| `no-expiry undead` | one buff, changes `[{UNDEAD, 1}]`, `expiresAt = time.Time{}`, `noExpiry true` | `true` |
| `unexpired non-undead` | one buff, changes `[{SPEED, 20}]`, unexpired | `false` |
| `undead not first change` | one buff, changes `[{SPEED, 20}, {UNDEAD, 1}]`, unexpired | `true` |
| `empty slice` | `nil` | `false` |
| `expired undead alongside unexpired speed` | two buffs: expired UNDEAD, unexpired SPEED | `false` |

`TestExpiredHonoursNoExpiry` — assert `NewBuff(..., expiresAt: time.Time{}, noExpiry: true).Expired() == false` and
`NewBuff(..., expiresAt: time.Now().Add(-time.Second), noExpiry: false).Expired() == true`.
Copy shape from `services/atlas-channel/atlas.com/channel/character/buff/model_test.go:9-24`.

`processor_notfound_test.go` — package `buff_test`. One function,
`TestGetByCharacterIdTreatsNotFoundAsNoBuffs`, copied nearly verbatim from
`services/atlas-channel/atlas.com/channel/character/buff/processor_notfound_test.go:34-62`:
`httptest` server replying `404` for every request, `t.Setenv("BUFFS_SERVICE_URL", srv.URL+"/")`,
tenant via `tenant.Create(uuid.New(), "GMS", 83, 1)` + `tenant.WithContext`,
`test.NewNullLogger()`. Assert: `err == nil`, `len(ms) == 0`, and the server was
actually hit (`calls > 0`).

- [ ] **Step 2: Implement**

`stat/rest.go`:

```go
package stat

import "github.com/Chronicle20/atlas/libs/atlas-constants/character"

type RestModel struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func Extract(rm RestModel) (Model, error) {
	return Model{Type: character.TemporaryStatType(rm.Type), Amount: rm.Amount}, nil
}
```

`rest.go` — `RestModel` with `Id string \`json:"-"\``, `SourceId int32`,
`Level byte`, `Duration int32`, `Changes []stat.RestModel`, `CreatedAt time.Time`,
`ExpiresAt time.Time`, `NoExpiry bool`; `GetName() string` returning `"buffs"`,
`GetID`, `SetID`; `Extract` mapping through `model.SliceMap(stat.Extract)`.
`SetToOneReferenceID`/`SetToManyReferenceIDs` are **not** required —
`atlas-channel`'s `RestModel` omits them and drains fine.

`model.go` — `Model` with unexported `sourceId/level/duration/changes/createdAt/expiresAt/noExpiry`,
getters `SourceId/Level/Changes/CreatedAt/ExpiresAt/NoExpiry`, `Expired()`
(returns `false` when `noExpiry`), `NewBuff(...)`, and:

```go
// IsZombified reports whether bs contains an unexpired buff carrying an
// UNDEAD stat change -- the ZOMBIFY disease. Slice-level because every
// caller already holds the drained list and the question is "does any of
// these". See task-256 FR-1.
func IsZombified(bs []Model) bool
```

`processor.go` — add `GetByCharacterId(characterId uint32) ([]Model, error)` to
the `Processor` interface and implement it on `ProcessorImpl`:
resolve the URL, `requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())()`,
and map `errors.Is(err, requests.ErrNotFound)` to `([]Model{}, nil)`. Carry the
rationale comment from the channel original — the normalization is the whole
reason this client copies `atlas-channel`'s and not `atlas-monsters`'.

`mock/processor.go` — add `GetByCharacterIdFunc func(characterId uint32) ([]Model, error)`
and the method, defaulting to `return nil, nil` when the func is unset.

- [ ] **Step 3: Verify**

From `services/atlas-consumables/atlas.com/consumables`: `go build ./... && go test ./...`.

---

## Task 2: `atlas-consumables` — halve HP restoration under zombify

### Files

- `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` — `halveIfZombified`, `resolveZombified`, `computeEffectPlan` signature + HP branches, `ApplyItemEffects`
- `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon.go` — `computeMorphCouponPlan` signature, `consumeMorphCoupon` read + warning gate
- `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` — 7 existing `computeEffectPlan` call sites gain a `false` argument; new zombify cases
- `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go` — 1 existing `computeMorphCouponPlan` call site gains `zombified`; new zombify cases
- `services/atlas-consumables/atlas.com/consumables/character/buff/mock/processor.go` — read-only here; Task 1 already added `GetByCharacterIdFunc`

Patterns to copy: `morphCouponHarness` at lines 208-255 of
services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go
— the five-mock deps wiring and its `hpChanges`/`applies` recorders — is the
harness for every `consumeMorphCoupon` case below.

Module root: `services/atlas-consumables/atlas.com/consumables`

Existing `computeEffectPlan` call sites to update (add `false` as the fourth
argument; **no expected value moves** — NFR "Backward compatibility"):
`processor.go:236`, `processor_test.go:494`, `:510`, `:528`, `:543`, `:561`,
`:578`, `:589`.
Existing `computeMorphCouponPlan` call sites: `morph_coupon.go:109`,
`morph_coupon_test.go:121`.

### Steps

- [ ] **Step 1: Write the failing tests**

All in package `consumable`. Build consumables through the existing
`extractConsumable(t, consumable3.RestModel{...})` helper
(`processor_test.go:474-483`) and cash items through
`extractCash(t, map[cash.SpecType]int32{...})` (`morph_coupon_test.go:34-41`);
characters through `character.NewModelBuilder().SetMaxHp(...).SetMaxMp(...).Build()`.

**`TestComputeEffectPlan_Zombify`** (new, in `processor_test.go`) — table-driven,
one `t.Run` per row, each row calling
`computeEffectPlan(discardLogger(), c, ci, tc.zombified)`.

| subtest name | spec | maxHp / maxMp | zombified | want hpChanges | want mpChanges |
|---|---|---|---|---|---|
| `flat hp not zombified` | `HP:300` | 500 / 500 | `false` | `[]int16{300}` | empty |
| `flat hp zombified` | `HP:300` | 500 / 500 | `true` | `[]int16{150}` | empty |
| `flat hp odd truncates down` | `HP:301` | 500 / 500 | `true` | `[]int16{150}` | empty |
| `ratio hp not zombified` | `HPRecovery:25` | 1000 / 500 | `false` | `[]int16{250}` | empty |
| `ratio hp zombified` | `HPRecovery:25` | 1000 / 500 | `true` | `[]int16{125}` | empty |
| `flat and ratio zombified keep order` | `HP:300, HPRecovery:25` | 1000 / 500 | `true` | `[]int16{150, 125}` | empty |
| `halved to zero is dropped` | `HP:1` | 500 / 500 | `true` | empty | empty |
| `mp untouched by zombify` | `HP:300, MP:200, MPRecovery:50` | 1000 / 1000 | `true` | `[]int16{150}` | `[]int16{200, 500}` |

**`TestComputeEffectPlan_ZombifyLeavesNonHpFieldsIdentical`** (new) — one
consumable carrying `Poison:1, HP:300, WeaponAttack:12, Time:300000` against
`maxHp 1000`; compute the plan twice, `zombified false` and `true`; assert
`cureTypes`, `mpChanges`, `statups`, and `duration` are equal between the two
plans, and that only `hpChanges` differs (`[300]` vs `[150]`).

**`TestResolveZombified`** (new) — table-driven against a
`buffmock.ProcessorMock`. Use `test.NewNullLogger()` from
`github.com/sirupsen/logrus/hooks/test` so the `Warn` is assertable.

| subtest name | `GetByCharacterIdFunc` returns | want | want Warn entries |
|---|---|---|---|
| `undead buff` | one unexpired buff with an `UNDEAD` change, `nil` | `true` | 0 |
| `no buffs` | `[]buff.Model{}, nil` | `false` | 0 |
| `non-undead buff` | one unexpired `SPEED` buff, `nil` | `false` | 0 |
| `read failure fails open` | `nil, errors.New("boom")` | `false` | exactly 1 entry at `logrus.WarnLevel` |

**`TestComputeMorphCouponPlan`** (existing, `morph_coupon_test.go:46`) — add a
`zombified bool` column; every existing row keeps its current expectations with
`zombified: false`. Add three rows:

| subtest name | spec | zombified | wantHp | wantMorph | wantDuration |
|---|---|---|---|---|---|
| `full spec zombified` | `Morph:1, Hp:50, Time:600000` | `true` | `25` | `1` | `600000` |
| `odd hp zombified truncates down` | `Hp:51, Time:600000` | `true` | `25` | `0` | `600000` |
| `hp 1 zombified truncates to zero` | `Hp:1, Time:600000` | `true` | `0` | `0` | `600000` |

**`TestConsumeMorphCouponZombifiedHalvesHeal`** (new) — `newMorphCouponHarness(t, extractCash(t, fullMorphSpec()), nil)`,
with the harness's `buff` mock additionally given
`GetByCharacterIdFunc` returning one unexpired `UNDEAD` buff. Assert:
`len(h.consumeItems) == 1`, `h.hpChanges == []changeHPCall{{555, 25}}`,
`len(h.applies) == 1` with `duration == 600000` and the MORPH statup unchanged,
`len(h.errors) == 0`.

**`TestConsumeMorphCouponBuffReadFailureHealsFullValue`** (new) — same harness,
`GetByCharacterIdFunc` returning `nil, errors.New("boom")`. Assert
`h.hpChanges == []changeHPCall{{555, 50}}` and `len(h.errors) == 0` (a buff-read
failure must never bounce a paid coupon).

**`TestConsumeMorphCouponZombifiedZeroHealDoesNotWarnAboutCashData`** (new) —
harness built from `extractCash(t, map[cash.SpecType]int32{cash.SpecTypeHp: 1, cash.SpecTypeTime: 600000})`,
`GetByCharacterIdFunc` returning one unexpired `UNDEAD` buff, logger from
`test.NewNullLogger()`. Assert `len(h.hpChanges) == 0`, `len(h.applies) == 0`,
and that **no** logged entry contains `"neither a morph nor an hp"` — that
message blames the tenant's cash WZ ingest and would be a false diagnosis for a
heal the zombify halving truncated away.

- [ ] **Step 2: Implement**

In `processor.go`, next to `computeEffectPlan`:

```go
// halveIfZombified halves an HP restoration amount when the recipient is
// zombified, using Go integer division (truncation toward zero) to match the
// reference's `hpchange /= 2` on an int. Order matters for hpR: the reference
// computes (int)(maxHp * hpR) / 2 -- cast first, then integer-divide -- which
// is what passing the already-floored int16 here reproduces. (task-256 FR-5/FR-6)
func halveIfZombified(amount int16, zombified bool) int16 {
	if !zombified {
		return amount
	}
	return amount / 2
}

// resolveZombified reads the character's live buffs and resolves the zombify
// predicate. A failed read resolves to false and logs at Warn (task-256 FR-3):
// an atlas-buffs outage must degrade to "the debuff has no effect", never to a
// mis-halved heal. Takes the Processor rather than constructing one so the
// fail-open branch is directly unit-testable.
func resolveZombified(l logrus.FieldLogger, bp buff.Processor, characterId uint32) bool {
	bs, err := bp.GetByCharacterId(characterId)
	if err != nil {
		l.WithError(err).Warnf("Unable to read buffs for character [%d]; treating as not zombified.", characterId)
		return false
	}
	return buff.IsZombified(bs)
}
```

`computeEffectPlan` gains `zombified bool` as its fourth parameter. Only the two
HP branches change — MP, cure, statups, and duration are untouched (FR-8):

```go
if val, ok := ci.GetSpec(consumable3.SpecTypeHP); ok && val > 0 {
	if amt := halveIfZombified(int16(val), zombified); amt > 0 {
		plan.hpChanges = append(plan.hpChanges, amt)
	}
}
if val, ok := ci.GetSpec(consumable3.SpecTypeHPRecovery); ok && val > 0 {
	pct := float64(val) / float64(100)
	if amt := halveIfZombified(int16(math.Floor(float64(c.MaxHp())*pct)), zombified); amt > 0 {
		plan.hpChanges = append(plan.hpChanges, amt)
	}
}
```

Add a `Debugf` inside the halving branches, guarded on `zombified`, reporting the
character id and the pre/post amounts (design §2 Observability). The function
stays pure with respect to its return value.

`ApplyItemEffects` (`processor.go:232`) resolves the predicate **once**, using
the `bp` already constructed at line 233, before `computeEffectPlan` and
therefore before the cure dispatch — the task-051 D3 cure-then-HP ordering at
`processor.go:238-254` must not move (FR-9):

```go
zombified := resolveZombified(l, bp, characterId)
plan := computeEffectPlan(l, c, ci, zombified)
```

In `morph_coupon.go`: `computeMorphCouponPlan(ci cash.Model, zombified bool)`,
with `plan.hp = halveIfZombified(int16(val), zombified)` in the `SpecTypeHp`
branch. The existing `if plan.hp > 0` guard at line 115 already implements the
zero-drop.

In `consumeMorphCoupon`, **move** the `computeMorphCouponPlan` call from its
current position (line 109, before the commit) to **after** the
`d.compartment.ConsumeItem` block, and resolve the predicate immediately before
it:

```go
if err := d.compartment.ConsumeItem(characterId, inventory2.TypeValueCash, transactionId, slot); err != nil {
	return d.onError(err)
}

// The zombify read is deliberately NOT in the pre-commit group. That group's
// contract is "every FALLIBLE read before the commit, so a data failure
// returns the paid cash item". Per FR-3 this read's failure mode is a logged
// false, never an abort -- putting it in the group would either add a new way
// to bounce a paid coupon or sit there ignoring its own error. (task-256 D4)
zombified := resolveZombified(l, d.buff, characterId)
plan := computeMorphCouponPlan(ci, zombified)
```

Gate the "neither a morph nor an hp" warning at line 129 on `!zombified`, with a
comment saying why: under zombify a truncated-to-zero heal is not a WZ ingest
defect, and the message would misdiagnose it.

- [ ] **Step 3: Verify**

From `services/atlas-consumables/atlas.com/consumables`: `go build ./... && go test ./...`.

---

## Task 3: `atlas-channel` — `buff.IsZombified`

### Files

- `services/atlas-channel/atlas.com/channel/character/buff/model.go` — add `IsZombified`
- `services/atlas-channel/atlas.com/channel/character/buff/model_test.go` — add `TestIsZombified`

Patterns to copy: `services/atlas-channel/atlas.com/channel/character/buff/model.go:10-20`
(`IsMount` — the same `charconst` cast, generalized to a slice and skipping
expired buffs).

Module root: `services/atlas-channel/atlas.com/channel`

Type note: here `stat.Model.Type()` returns a **`string`**
(`character/buff/stat/model.go:8-10`), so the comparison **requires** the cast:
`c.Type() == string(charconst.TemporaryStatTypeUndead)`. Build stats with
`stat.NewStat("UNDEAD", 1)`; build buffs with the existing
`NewBuff(sourceId, level, duration, changes, createdAt, expiresAt, noExpiry)`.

### Steps

- [ ] **Step 1: Write the failing test**

`TestIsZombified` in `model_test.go` (package `buff`) — table-driven, same seven
rows as Task 1's table, adapted to this package's constructors:

| subtest name | buffs | expect |
|---|---|---|
| `unexpired undead` | one buff, `[]stat.Model{stat.NewStat("UNDEAD", 1)}`, `expiresAt = time.Now().Add(time.Minute)`, `noExpiry false` | `true` |
| `expired undead` | same changes, `expiresAt = time.Now().Add(-time.Second)`, `noExpiry false` | `false` |
| `no-expiry undead` | same changes, `expiresAt = time.Time{}`, `noExpiry true` | `true` |
| `unexpired non-undead` | `[]stat.Model{stat.NewStat("SPEED", 20)}`, unexpired | `false` |
| `undead not first change` | `[]stat.Model{stat.NewStat("SPEED", 20), stat.NewStat("UNDEAD", 1)}`, unexpired | `true` |
| `empty slice` | `nil` | `false` |
| `expired undead alongside unexpired speed` | two buffs: expired UNDEAD, unexpired SPEED | `false` |

- [ ] **Step 2: Implement**

```go
// IsZombified reports whether bs contains an unexpired buff carrying an
// UNDEAD stat change -- the ZOMBIFY disease. Slice-level because every caller
// already holds the drained list and the question is "does any of these".
// See task-256 FR-1.
func IsZombified(bs []Model) bool
```

Skip `b.Expired()` buffs; compare each change's `Type()` against
`string(charconst.TemporaryStatTypeUndead)`; return on first match.

- [ ] **Step 3: Verify**

From `services/atlas-channel/atlas.com/channel`: `go build ./... && go test ./...`.

---

## Task 4: `atlas-channel` — Cleric Heal negation under caster zombify

### Files

- `services/atlas-channel/atlas.com/channel/skill/handler/heal/heal.go` — the eight replaceable seams and the reworked cast body
- `services/atlas-channel/atlas.com/channel/skill/handler/heal/formula.go` — add `healDelta`; do **not** modify `appliedPerRecipient`, `HealAmount`, or `HealXp`
- `services/atlas-channel/atlas.com/channel/skill/handler/heal/formula_test.go` — add `TestHealDelta`
- `services/atlas-channel/atlas.com/channel/skill/handler/heal/heal_apply_test.go` — **new file**; the seam-driven cast tests
- `services/atlas-channel/atlas.com/channel/character/buff/model.go` — read-only; `IsZombified` from Task 3

Patterns to copy, all under services/atlas-channel/atlas.com/channel/skill/handler:
dispel/dispel.go lines 43-65 (the package-level replaceable-`var` seam idiom and
its comments); dispel/dispel_test.go lines 54-100 (save-original /
`t.Cleanup`-restore / recorder-closure test shape); common.go lines 37-42 (the
`loadCasterFunc(cp character.Processor, characterId uint32)` seam signature);
monstermagnet/monstermagnet_test.go lines 80-84
(`character.NewModelBuilder()...MustBuild()` inside a load seam).

Module root: `services/atlas-channel/atlas.com/channel`

**Why eight seams and not one.** design D5 specified a single
`casterZombifiedFunc` and left the harness question to this phase. `heal.Apply`
cannot be driven offline today — it loads the caster over REST and returns early
on failure — so a single seam would leave PRD acceptance criteria 2–7 and 9
unassertable at the `ChangeHP` boundary the PRD names. The resolution is the
`dispel`/`monstermagnet` idiom the design already endorses: one-line package
vars, no `healDeps` struct, no restructuring of the handler's flow.

### Steps

- [ ] **Step 1: Write the failing pure test**

`TestHealDelta` in `formula_test.go` — table-driven; `recipient` is a plain
struct literal (`recipient{Id: 1, Hp: 900, MaxHp: 1000}`), as at
`formula_test.go:91-124`.

| subtest name | perTarget | recipient Hp / MaxHp | zombified | want |
|---|---|---|---|---|
| `not zombified full headroom` | 80 | 900 / 1000 | `false` | `80` |
| `not zombified headroom clamp` | 80 | 950 / 1000 | `false` | `50` |
| `not zombified at max hp` | 80 | 1000 / 1000 | `false` | `0` |
| `zombified full magnitude` | 80 | 900 / 1000 | `true` | `-80` |
| `zombified clamped to current hp` | 80 | 50 / 1000 | `true` | `-50` |
| `zombified exact kill` | 80 | 80 / 1000 | `true` | `-80` |
| `zombified recipient already dead` | 80 | 0 / 1000 | `true` | `0` |
| `zombified zero magnitude` | 0 | 500 / 1000 | `true` | `0` |

Assert additionally, in the two `not zombified` clamp rows, that
`healDelta(perTarget, r, false) == appliedPerRecipient(perTarget, r)` — that
equality is PRD acceptance criterion 1 (the non-zombified path is byte-for-byte
unchanged).

The `math.MinInt16` saturation branch is **unreachable** from today's inputs
(`perTarget` is an `int16`, so `-perTarget >= -32767 > math.MinInt16`). Write no
test for it; keep it as a defensive guard against a future widening and say so in
the comment.

- [ ] **Step 2: Implement `healDelta`**

In `formula.go`, next to `appliedPerRecipient`:

```go
// healDelta returns the ChangeHP delta for one recipient of a Heal cast.
//
// Non-zombified: the existing headroom clamp -- never push Hp past MaxHp.
// Zombified: the reference negates the heal (StatEffect.calcHPChange), so the
// delta is damage. It is clamped to the recipient's CURRENT Hp so a cast never
// removes more HP than the recipient has; landing exactly on 0 kills them,
// which is intended (atlas-character emits DIED at adjusted == 0).
// appliedPerRecipient is deliberately never handed a negative value: its
// headroom clamp would mangle one. (task-256 FR-12/FR-13/FR-14)
func healDelta(perTarget int16, r recipient, zombified bool) int16 {
	if !zombified {
		return appliedPerRecipient(perTarget, r)
	}
	if r.Hp == 0 {
		return 0
	}
	magnitude := int32(perTarget)
	if magnitude < 0 {
		magnitude = 0
	}
	if magnitude > int32(r.Hp) {
		magnitude = int32(r.Hp)
	}
	if -magnitude < math.MinInt16 {
		return math.MinInt16
	}
	return int16(-magnitude)
}
```

- [ ] **Step 3: Write the failing cast tests**

`heal_apply_test.go`, package `heal`. Every test saves the originals of the seams
it replaces and restores them in `t.Cleanup`, per `dispel_test.go:54-59`.

Shared fixtures for this file:

- `castEffect(t)` — `effect.Extract(effect.RestModel{Hp: 300})`; `effect.Model`'s
  fields are unexported and there is no builder, so `Extract` is the only way to
  build one carrying an HP value. `t.Fatalf` on error.
- `castInfo(mobIds []uint32)` — `packetmodel.NewSkillUsageInfoBuilder().SetSkillId(2301002).SetSkillLevel(1).SetAffectedMobIds(mobIds).Build()`.
- `castField()` — `field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()`, matching `dispel_test.go:73`.
- a `changeHpCall struct{ characterId uint32; amount int16 }` recorder.

Seam replacements shared by both cast tests:

- `loadCasterFunc` → `character.NewModelBuilder().SetId(1).SetLevel(30).SetIntelligence(100).SetHp(500).SetMaxHp(1000).SetX(0).SetY(0).MustBuild(), nil`
- `effectiveStatsFunc` → `effective_stats.RestModel{Intelligence: 100, MagicAttack: 0, MaxHp: 1000}, nil` for every id (keeps the test hermetic — no REST attempt for the caster or any recipient)
- `selectPartyMembersFunc` → three `channelhandler.NewPartyRecipientBuilder()` recipients: id 2 `Hp 50 MaxHp 1000`, id 3 `Hp 60 MaxHp 1000`, id 4 `Hp 0 MaxHp 1000`
- `varianceFunc` → `1.0`
- `changeHpFunc`, `awardExperienceFunc`, `announceCastFunc` → recorders returning `nil`

With those fixtures the arithmetic is fully pinned: four recipients, so
`perTarget = HealAmount(300, 0, 100, 4, 1.0) = floor(300*(0*1.5+100*0.8)/100*1.0/4) = floor(240/4) = 60`.

**`TestApply_NotZombified_HealsEveryRecipient`** — `casterZombifiedFunc` → `false`.

| assertion | expected |
|---|---|
| `changeHpFunc` calls, in order | `{1, +60}, {2, +60}, {3, +60}, {4, +60}` |
| `awardExperienceFunc` calls | exactly 1; `distributions[0].ExperienceType == character2.ExperienceDistributionTypeWhite`, `Amount == 24` (`HealXp(60, 4 recipients, level 1)` = `60*4/10*1`) |
| `announceCastFunc` calls | exactly 1 |
| returned error | `nil` |

**`TestApply_ZombifiedCaster_DamagesEveryRecipient`** — `casterZombifiedFunc` → `true`.

| assertion | expected |
|---|---|
| `changeHpFunc` calls, in order | `{1, -60}, {2, -50}, {3, -60}` — recipient 4 (Hp 0) is skipped entirely (FR-13, acceptance criterion 4) |
| recipient 2 | `-50`, not `-60`: clamped to its current Hp (acceptance criterion 3) |
| recipient 3 | `-60` with `Hp == 60`, i.e. exactly `-Hp` — the delta that lands on 0 and triggers atlas-character's existing DIED emission (acceptance criterion 5, asserted at the call boundary) |
| `awardExperienceFunc` calls | exactly 0 (FR-15) |
| `announceCastFunc` calls | exactly 1 — the broadcast is unchanged on a negated cast (FR-16) |
| returned error | `nil` |

**`TestApply_ZombifyReadIsCasterOnlyAndIssuedOnce`** — same fixtures,
`casterZombifiedFunc` replaced with a closure recording every argument it is
called with. Assert the recorded slice is exactly `[]uint32{1}` with four
recipients resolved: one call, for the caster id, never for a recipient id
(FR-11, acceptance criteria 9 and 11).

- [ ] **Step 4: Implement the seams and the cast body**

Add to `heal.go`, above `init()`, each with a one-line comment in the
`dispel.go:43-65` register:

```go
var loadCasterFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
	return cp.GetById()(characterId)
}

var effectiveStatsFunc = func(esp effective_stats.Processor, worldId world.Id, channelId channel.Id, characterId uint32) (effective_stats.RestModel, error) {
	return esp.GetByCharacterId(worldId, channelId, characterId)
}

var selectPartyMembersFunc = channelhandler.SelectInRangePartyMembers

// varianceFunc is the [0.9, 1.1] heal roll, seamed so a cast's arithmetic is
// pinnable end-to-end.
var varianceFunc = func() float64 {
	return 0.9 + rand.Float64()*0.2
}

// casterZombifiedFunc is the zombify-state seam tests can replace. Production
// drains the caster's buffs from atlas-buffs and applies buff.IsZombified.
// Per task-256 FR-3 a failed read resolves to false (not zombified): a
// buff-service fault must never turn a Cleric's Heal into party-wide damage.
var casterZombifiedFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) bool {
	bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
	if err != nil {
		l.WithError(err).Warnf("Heal: buff read failed for caster [%d]; treating as not zombified.", characterId)
		return false
	}
	return buff.IsZombified(bs)
}

var changeHpFunc = func(cp character.Processor, f field.Model, characterId uint32, amount int16) error {
	return cp.ChangeHP(f, characterId, amount)
}

var awardExperienceFunc = func(cp character.Processor, f field.Model, characterId uint32, distributions []character2.ExperienceDistributions, showEffect bool) error {
	return cp.AwardExperience(f, characterId, distributions, showEffect)
}

// announceCastFunc broadcasts the cast to the caster and to every other
// session in the map. Seamed as one unit because both announcements are
// unconditional on every cast, negated or not (task-256 FR-16).
var announceCastFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, characterId uint32, casterLevel byte, skillId uint32, skillLevel byte) {
	sp := session.NewProcessor(l, ctx)
	_ = sp.IfPresentByCharacterId(f.Channel())(
		characterId,
		socketHandler.AnnounceSkillUse(l)(ctx)(wp)(skillId, casterLevel, skillLevel),
	)
	_ = channelmap.NewProcessor(l, ctx).ForOtherSessionsInMap(
		f, characterId,
		socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(characterId, skillId, casterLevel, skillLevel),
	)
}
```

Route every existing call in `Apply` through its seam: `cp.GetById()(characterId)`
at `heal.go:83` → `loadCasterFunc(cp, characterId)`; `esp.GetByCharacterId(...)`
at `:90` and `:118` → `effectiveStatsFunc(esp, ...)`;
`channelhandler.SelectInRangePartyMembers(...)` at `:100` →
`selectPartyMembersFunc(...)`; the inline `0.9 + rand.Float64()*0.2` at `:126` →
`varianceFunc()`; `cp.ChangeHP(...)` at `:140` → `changeHpFunc(cp, ...)`;
`cp.AwardExperience(...)` at `:149` → `awardExperienceFunc(cp, ...)`; the
session/map block at `:158-166` → `announceCastFunc(l, ctx, wp, f, characterId, c.Level(), info.SkillId(), info.SkillLevel())`.
`Apply`'s flow, log lines, and error handling are otherwise unchanged.

Resolve the predicate **once**, immediately before the recipient loop and after
the MaxHp hydration — never inside the loop (FR-11, NFR "the read must not be
issued per-recipient"):

```go
zombified := casterZombifiedFunc(l, ctx, characterId)

variance := varianceFunc()
perTarget := HealAmount(e.HP(), int(stats.MagicAttack), int(stats.Intelligence), len(recipients), variance)

for _, r := range recipients {
	delta := healDelta(perTarget, r, zombified)
	if delta == 0 {
		continue
	}
	if hpErr := changeHpFunc(cp, f, r.Id, delta); hpErr != nil {
		l.WithError(hpErr).Errorf("Heal: ChangeHP failed for recipient [%d] from caster [%d].", r.Id, characterId)
	}
}
```

`perTarget` keeps its positive magnitude throughout — the negation happens
per-recipient inside `healDelta`. That is what keeps `HealXp` and the closing
`Debugf` from ever seeing a negative.

Gate the XP block (`heal.go:146`) on `!zombified` as well:

```go
// XP gate: skip when sole recipient AND no undead targets in this cast.
// Also skipped entirely on a negated cast -- HealXp derives from the applied
// heal, and a zombified cast heals nobody (task-256 FR-15).
if !zombified && !(len(recipients) == 1 && len(info.AffectedMobIds()) == 0) {
```

Extend the closing `Debugf` at `heal.go:168` with the `zombified` flag (design §2
Observability).

Undead **mobs** are out of scope: Atlas's Heal applies no mob damage today,
zombified or not, and this task neither widens nor closes that pre-existing gap
(design D6).

- [ ] **Step 5: Verify**

From `services/atlas-channel/atlas.com/channel`: `go build ./... && go test ./...`.

---

## Definition of done

- All four tasks' module-local `go build ./... && go test ./...` pass.
- Flagless `tools/verify.sh` exits 0.
- Code review completed before the PR is opened.
