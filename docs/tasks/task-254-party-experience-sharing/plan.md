# Party EXP Sharing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give `atlas-monster-death` a real party EXP distribution path — pooled party damage, level-weighted splits, an MVP bonus, a level-range (leech) gate with a client-visible notice, and a corrected `totalDamage`/`totalEntries` basis — replacing the solo-only distributor whose party branch is an unimplemented `TODO parties` marker.

**Architecture:** The distributor is restructured into **resolve → plan → award**. `resolve` does all I/O (monster information, the authoritative in-field character set from `atlas-maps`, and one party lookup per distinct party). `plan` is a pure function `planDistribution(ExperienceInput, ExperienceConfig) ExperiencePlan` over value types — every numeric requirement becomes a mock-free table test. `award` walks the plan's recipients (ascending `characterId`), applies the character's EXP rate, and emits one `AWARD_EXPERIENCE` command each, then emits one throttled `SHOW_HINT` per level-gate exclusion. Collaborators are injected through an `atlas-pets`-style `With(...)` option set on `monster.ProcessorImpl`.

**Tech Stack:** Go, `libs/atlas-model` providers, `libs/atlas-rest/requests` (JSON:API via `jtumidanski/api2go/jsonapi`), `libs/atlas-kafka/producer`, `libs/atlas-constants` (`field`, `channel`, `world`, `job`, `map`), `libs/atlas-tenant`, `logrus`, stdlib `testing`.

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md))

## Global Constraints

Copy these verbatim into every task's mental checklist.

- **Module roots.** `atlas-monster-death` is `services/atlas-monster-death/atlas.com/monster` (module name `atlas-monster-death`; note the directory is `monster`, **not** `monster-death`). `atlas-monsters` is `services/atlas-monsters/atlas.com/monsters`. Run `go build ./... && go test ./...` from the module root of whatever you touched, and nothing more — repo-wide verification is a separate gate.
- **Immutable models.** Unexported fields, value receivers, construction through a `Builder` or `Extract`. No exported struct fields on domain models.
- **Builders for test setup.** Use the project Builder pattern. Do **not** create `*_testhelpers.go` files or test-only constructors.
- **Constants reuse.** `world.Id`, `channel.Id`, `_map.Id`, `job.Id`, `field.Model` come from `libs/atlas-constants/`. Never redeclare them.
- **No stubs.** No `// TODO`, no unimplemented handler, no placeholder return. Both existing `// TODO parties` and `// TODO account for healing` comments in `monster/monster/processor.go` are deleted by this plan (Task 12).
- **No service-boundary reach-in.** The `SHOW_HINT` command structs are **copied** into `atlas-monster-death`, not imported from `atlas-channel` or `atlas-saga-orchestrator`.
- **`ExperienceConfig` values (D9):** `EnforceMobLevelRange = true`, `LevelInterval = 5`, `LeachInterval = 5`, `SplitCommonMod = 0.8`, `MvpMod = 0.2`, `PartyBonusPerMember = 0.05`. Only the first three have env keys (`USE_ENFORCE_MOB_LEVEL_RANGE`, `LEVEL_INTERVAL`, `LEACH_INTERVAL`).
- **Determinism (FR-9.1).** Parties ascending by `partyId`; members and solo characters ascending by `characterId`; MVP ties broken by lowest `characterId`; `entryExperienceRatio` sorted ascending before the stddev call. No Go map iteration order may reach an observable output.
- **Resolved design decisions.** D12 (out-of-field damagers are never party-resolved; they count toward `totalDamage` and `totalEntries` and receive nothing) and D13 (MVP is chosen from `expMembers`, non-damagers counting as 0 damage) are both **adopted**. The PRD acceptance bullet that says an out-of-field member "still contributes their damage to `partyDamage`" is superseded by D12.
- **Preserve existing line endings** when editing; do not normalize.

---

### Task 1: `atlas-monsters` — `AddDamageEntry` aggregates by character

Defence in depth (D7). The live damage path is `Registry.ApplyDamage`, which already aggregates; `ModelBuilder.AddDamageEntry` is reached only from `Model.Damage()`, which has no production caller. Making the builder agree with the registry keeps `DamageSummary()`'s doc comment true through both paths. **Do not describe this as the change that makes party EXP work.**

`entry.LastHitMs` is not a parameter of `AddDamageEntry`, so a new entry gets `LastHitMs: 0` exactly as today, and an aggregating call leaves the existing entry's `LastHitMs` untouched (`max(existing, 0) == existing`).

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/builder.go` — `AddDamageEntry` at line 139-146: aggregate instead of append
- `services/atlas-monsters/atlas.com/monsters/monster/builder_test.go` — new test cases appended
- `services/atlas-monsters/atlas.com/monsters/monster/model.go` — read-only reference: `entry` at line 76-80, `DamageSummary()` at line 171-176, `DamageLeader()` at line 228-238

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `services/atlas-monsters/atlas.com/monsters/monster/registry.go:436-495` (`ApplyDamage`'s aggregation loop — match its semantics exactly). Test setup: `services/atlas-monsters/atlas.com/monsters/monster/builder_test.go:7-9` (`emptyBuilder()` helper already exists in the file).

- [x] **Step 1: Write the failing test**

Append to `builder_test.go`. `emptyBuilder()` and `testField()` already exist in the package — do not redefine them.

`TestAddDamageEntry_AggregatesByCharacter` — table-driven over `calls`, asserting the resulting `b.damageEntries` slice exactly.

| case | calls (`characterId`, `damage`) in order | expected `damageEntries` in order |
|---|---|---|
| `same character sums` | (1, 100), (1, 100), (1, 100) | `[{CharacterId:1, Damage:300, LastHitMs:0}]` |
| `two characters keep first-contact order` | (2, 50), (1, 10), (2, 25) | `[{CharacterId:2, Damage:75, LastHitMs:0}, {CharacterId:1, Damage:10, LastHitMs:0}]` |
| `single entry unchanged` | (7, 42) | `[{CharacterId:7, Damage:42, LastHitMs:0}]` |
| `zero damage still creates an entry` | (3, 0) | `[{CharacterId:3, Damage:0, LastHitMs:0}]` |

Assert `len`, then each element's `CharacterId`, `Damage`, and `LastHitMs` field-by-field with `t.Errorf` naming the index.

`TestAddDamageEntry_PreservesExistingLastHitMs` — start from a builder whose `damageEntries` already holds `{CharacterId: 5, Damage: 100, LastHitMs: 900}` (assign `b.damageEntries` directly; the test is in-package), call `b.AddDamageEntry(5, 50)`, and assert one entry `{CharacterId: 5, Damage: 150, LastHitMs: 900}`.

`TestDamageLeader_OverBuilderAggregatedEntries` (PRD acceptance) — on `emptyBuilder()`, call `AddDamageEntry(1, 100)` three times and `AddDamageEntry(2, 250)` once, `Build()`, then assert `len(m.DamageSummary()) == 2` and `m.DamageLeader() == 1` (300 > 250 — a per-hit append would give 250 > 100 and return 2).

- [x] **Step 2: Run test to verify it fails**

Run from `services/atlas-monsters/atlas.com/monsters`:

```
go test ./monster/ -run 'TestAddDamageEntry|TestDamageLeader_OverBuilder' -v
```

Expected: FAIL — `TestAddDamageEntry_AggregatesByCharacter/same character sums` reports 3 entries, and `TestDamageLeader_OverBuilderAggregatedEntries` reports leader 2.

- [x] **Step 3: Write minimal implementation**

Replace `AddDamageEntry` in `builder.go`:

```go
// AddDamageEntry credits damage to a character's damage entry, aggregating by
// characterId. A repeat call for the same character sums into the existing
// entry and leaves its LastHitMs alone (this signature carries no timestamp);
// a first call appends, so slice order records first contact. This mirrors
// Registry.ApplyDamage (registry.go:436-495) so both write paths agree, which
// is what makes Model.DamageSummary()'s "pre-aggregated" contract true.
func (b *ModelBuilder) AddDamageEntry(characterId uint32, damage uint32) *ModelBuilder {
	for i := range b.damageEntries {
		if b.damageEntries[i].CharacterId == characterId {
			b.damageEntries[i].Damage += damage
			return b
		}
	}
	b.damageEntries = append(b.damageEntries, entry{
		CharacterId: characterId,
		Damage:      damage,
	})
	return b
}
```

- [x] **Step 4: Run the full module test suite**

```
go build ./... && go test ./...
```

Expected: PASS. `clear_aggro_test.go:51` (3 distinct characters → 3 entries), `aggro_task_test.go:168` (1 character → 1 entry), and `model_test.go:23-51` (entries constructed directly) are all unaffected — confirm they stay green rather than editing them.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/builder.go services/atlas-monsters/atlas.com/monsters/monster/builder_test.go
git commit -m "fix(atlas-monsters): aggregate builder damage entries by character (FR-1.1)"
```

---

### Task 2: `atlas-monster-death` — party members model, REST relationship, builders

`monster-death`'s `party` package is an id-only stub. It gains the full `members` relationship so the distributor can read member levels and enumerate members who dealt no damage.

### Interfaces

- Produces: `party.MemberModel` with `Id() uint32`, `Name() string`, `Level() byte`, `JobId() job.Id`, `Field() field.Model`, `Online() bool`; `party.Model.LeaderId() uint32` and `party.Model.Members() []MemberModel`; `party.NewBuilder(id uint32) *Builder` and `party.NewMemberBuilder(id uint32) *MemberBuilder`.

### Files

- `services/atlas-monster-death/atlas.com/monster/party/model.go` — add `leaderId`, `members`, `MemberModel` + accessors
- `services/atlas-monster-death/atlas.com/monster/party/rest.go` — populate the `members` relationship
- `services/atlas-monster-death/atlas.com/monster/party/builder.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/party/rest_test.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/party/requests.go` — read-only; the URL already matches the channel service's and needs no change
- `services/atlas-monster-death/atlas.com/monster/party/processor.go` — read-only; `GetByMemberId` already returns the whole party

Module root: `services/atlas-monster-death/atlas.com/monster`.

Patterns to copy: `services/atlas-channel/atlas.com/channel/party/rest.go` (port verbatim — `GetReferencedIDs`, `GetReferencedStructs`, `SetToManyReferenceIDs`, `SetReferencedStructs`, `MemberRestModel`, `Extract`, `ExtractMember`) and `services/atlas-channel/atlas.com/channel/party/model.go:13-80` (the `Model`/`MemberModel` shape). One deliberate divergence from the channel copy: its `SetToManyReferenceIDs` seed literal omits `Instance`; include `Instance: uuid.Nil` in ours so every field of the seeded struct is written explicitly.

- [x] **Step 1: Write the failing test**

New file `party/rest_test.go`, package `party`. Setup is plain struct literals plus `jsonapi` — no builder needed for the REST assertions themselves.

`TestExtract_MapsMembers` — build a `RestModel` literal:

```
RestModel{Id: 7, LeaderId: 11, Members: []MemberRestModel{
  {Id: 11, Name: "Leader", Level: 120, JobId: 112, WorldId: 0, ChannelId: 1, MapId: 100000000, Instance: uuid.Nil, Online: true},
  {Id: 12, Name: "Member", Level: 30,  JobId: 100, WorldId: 0, ChannelId: 1, MapId: 100000000, Instance: uuid.Nil, Online: false},
}}
```

Call `Extract(rm)` and assert: `m.Id() == 7`, `m.LeaderId() == 11`, `len(m.Members()) == 2`, and per member `Id()`, `Name()`, `Level()`, `JobId()`, `Online()`, plus `Members()[0].Field().MapId() == _map.Id(100000000)` and `Members()[0].Field().ChannelId() == channel.Id(1)`.

`TestSetToManyReferenceIDs_SeedsMembers` — on a `&RestModel{}`, call `SetToManyReferenceIDs("members", []string{"11", "12"})`, assert `len(r.Members) == 2` and `r.Members[0].Id == 11`, `r.Members[1].Id == 12`. Then call `SetToManyReferenceIDs("other", []string{"99"})` and assert `len(r.Members)` is still 2.

`TestGetReferencedIDs_And_Structs` — on a `RestModel` with the two members above, assert `GetReferencedIDs()` has length 2 with `Type == "members"`, `Name == "members"`, `ID` of `"11"` then `"12"`; and `GetReferencedStructs()` has length 2.

`TestBuilders_ProduceReadableModel` — `NewBuilder(7).SetLeaderId(11).AddMember(NewMemberBuilder(11).SetLevel(120).SetName("Leader").SetOnline(true).Build()).Build()`, assert `Id()`, `LeaderId()`, `len(Members()) == 1`, `Members()[0].Level() == 120`.

- [x] **Step 2: Run test to verify it fails**

```
go test ./party/ -v
```

Expected: FAIL to compile — `RestModel` has no `LeaderId`/`Members` field, `Model` has no `Members`, `NewBuilder` undefined.

- [x] **Step 3: Write the implementation**

`party/model.go`:

```go
package party

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

type Model struct {
	id       uint32
	leaderId uint32
	members  []MemberModel
}

func (m Model) Id() uint32              { return m.id }
func (m Model) LeaderId() uint32        { return m.leaderId }
func (m Model) Members() []MemberModel  { return m.members }

type MemberModel struct {
	id     uint32
	name   string
	level  byte
	jobId  job.Id
	field  field.Model
	online bool
}

func (m MemberModel) Id() uint32          { return m.id }
func (m MemberModel) Name() string        { return m.name }
func (m MemberModel) Level() byte         { return m.level }
func (m MemberModel) JobId() job.Id       { return m.jobId }
func (m MemberModel) Field() field.Model  { return m.field }
func (m MemberModel) Online() bool        { return m.online }
```

`party/rest.go`: port the channel service's file. `RestModel` gains `LeaderId uint32 \`json:"leaderId"\`` and `Members []MemberRestModel \`json:"-"\``. `MemberRestModel` carries `Id uint32 \`json:"-"\``, `Name string`, `Level byte`, `JobId job.Id`, `WorldId world.Id`, `ChannelId channel.Id`, `MapId _map.Id`, `Instance uuid.UUID`, `Online bool` with the channel service's JSON tags, plus `GetName() string` returning `"members"`, `GetID()`, `SetID()`. `ExtractMember` builds the field via `field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build()`. Keep `Extract`'s signature `func Extract(rm RestModel) (Model, error)`.

`party/builder.go` (new): `Builder{id, leaderId uint32; members []MemberModel}` with `NewBuilder(id uint32) *Builder`, `SetLeaderId(uint32) *Builder`, `AddMember(MemberModel) *Builder`, `Build() Model`; and `MemberBuilder{...}` with `NewMemberBuilder(id uint32) *MemberBuilder`, `SetName(string)`, `SetLevel(byte)`, `SetJobId(job.Id)`, `SetField(field.Model)`, `SetOnline(bool)`, `Build() MemberModel`. Builders return `Model`/`MemberModel` directly (no error) — this package has no validation to perform.

- [x] **Step 4: Run tests**

```
go build ./... && go test ./party/... -v
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/party/
git commit -m "feat(atlas-monster-death): deserialize party members relationship (FR-5.4)"
```

---

### Task 3: `atlas-monster-death` — monster information gains `Level`/`Name` and a `Processor`

The level gate needs the mob's level (FR-6.6) and the hint text needs its name (FR-6.8); the REST payload already carries both. The free function `GetById` gains a `Processor` wrapper so tests can inject it.

### Interfaces

- Consumes: nothing from earlier tasks.
- Produces: `information.Model.Level() uint32`, `information.Model.Name() string`; `information.Processor` interface with `GetById(monsterId uint32) (Model, error)`; `information.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`; `informationmock.ProcessorMock{GetByIdFunc func(monsterId uint32) (information.Model, error)}` in `monster/information/mock`.

### Files

- `services/atlas-monster-death/atlas.com/monster/monster/information/model.go` — add `level`, `name` + accessors
- `services/atlas-monster-death/atlas.com/monster/monster/information/builder.go` — add `SetLevel`, `SetName`
- `services/atlas-monster-death/atlas.com/monster/monster/information/rest.go` — `Extract` maps `Level` and `Name` (the `RestModel` fields at lines 7 and 11 already exist)
- `services/atlas-monster-death/atlas.com/monster/monster/information/processor.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/monster/information/mock/processor.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/monster/information/rest_test.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/monster/information/provider.go` — read-only; the existing free function stays and the processor delegates to it

Module root: `services/atlas-monster-death/atlas.com/monster`.

Patterns to copy: `services/atlas-monster-death/atlas.com/monster/party/processor.go` (Processor/ProcessorImpl/NewProcessor + `var _ Processor = (*ProcessorImpl)(nil)`) and `services/atlas-monster-death/atlas.com/monster/party/mock/processor.go` (nil-func fallthrough to a zero value).

- [x] **Step 1: Write the failing test**

New file `monster/information/rest_test.go`, package `information`.

`TestExtract_CarriesLevelAndName` — `Extract(RestModel{Id: 100100, Name: "Blue Snail", Hp: 8, Experience: 3, Level: 2})`, assert `Hp() == 8`, `Experience() == 3`, `Level() == 2`, `Name() == "Blue Snail"`.

`TestBuilder_SetsLevelAndName` — `NewBuilder().SetHp(1000).SetExperience(500).SetLevel(125).SetName("Zakum").Build()`, assert all four accessors.

- [x] **Step 2: Run test to verify it fails**

```
go test ./monster/information/ -v
```

Expected: FAIL to compile — `Model` has no `Level`/`Name`, `Builder` has no `SetLevel`/`SetName`.

- [x] **Step 3: Write the implementation**

`model.go` gains `level uint32` and `name string` with `Level() uint32` and `Name() string` value-receiver accessors. `builder.go` gains matching fields (zero defaults — do not invent a non-zero level or name) and `SetLevel(level uint32) *Builder` / `SetName(name string) *Builder`, both wired into `Build()`. `rest.go`'s `Extract` gains `level: rm.Level, name: rm.Name`.

`monster/information/processor.go` (new):

```go
package information

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(monsterId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(monsterId uint32) (Model, error) {
	return GetById(p.l)(p.ctx)(monsterId)
}
```

`monster/information/mock/processor.go` (new): package `mock`, `ProcessorMock{GetByIdFunc func(monsterId uint32) (information.Model, error)}`, `var _ information.Processor = (*ProcessorMock)(nil)`, nil-func fallthrough returning `information.Model{}, nil`.

- [x] **Step 4: Run tests**

```
go build ./... && go test ./monster/information/... -v
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/monster/information/
git commit -m "feat(atlas-monster-death): expose monster level and name, add information processor (FR-6.6)"
```

---

### Task 4: `atlas-monster-death` — `rates` and `map` processors, `rates` builder

Two more free functions gain the `Processor` + `mock` shape, so the distributor's rate-lookup count and co-location set become assertable. The `rates` package also gains a builder, because `rates.Default()` pins every multiplier at 1.0 and the party tests need a non-1.0 EXP rate.

### Interfaces

- Produces: `rates.Processor` with `GetForCharacter(ch channel.Model, characterId uint32) Model`; `rates.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`; `rates.NewBuilder() *Builder` with `SetExpRate/SetMesoRate/SetItemDropRate/SetQuestExpRate(float64) *Builder` and `Build() Model`; `_map.Processor` with `CharacterIdsInField(f field.Model) ([]uint32, error)`; `_map.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`; matching `ProcessorMock`s in `rates/mock` and `map/mock`.

### Files

- `services/atlas-monster-death/atlas.com/monster/rates/processor.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/rates/mock/processor.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/rates/builder.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/map/processor.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/map/mock/processor.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/rates/builder_test.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/rates/provider.go` — read-only; `GetForCharacter` stays as the free function the processor delegates to
- `services/atlas-monster-death/atlas.com/monster/map/provider.go` — read-only; `CharacterIdsInFieldModelProvider` stays; `CreateDrops` and the drop path keep calling the free functions unchanged

Module root: `services/atlas-monster-death/atlas.com/monster`. The `map` package's Go package name is `_map` (see `map/provider.go:1`); its mock subpackage is package `mock` importing `_map "atlas-monster-death/map"`.

Patterns to copy: `services/atlas-monster-death/atlas.com/monster/party/processor.go` and `.../party/mock/processor.go`. Builder pattern: `services/atlas-monster-death/atlas.com/monster/monster/information/builder.go`.

- [x] **Step 1: Write the failing test**

New file `rates/builder_test.go`, package `rates`.

`TestNewBuilder_DefaultsToUnitRates` — `NewBuilder().Build()`, assert all four accessors are `1.0` (the builder defaults match `Default()`, so a test that forgets to set a rate behaves like today's code).

`TestBuilder_SetsEachRate` — `NewBuilder().SetExpRate(2.5).SetMesoRate(3.0).SetItemDropRate(4.0).SetQuestExpRate(5.0).Build()`, assert `ExpRate() == 2.5`, `MesoRate() == 3.0`, `ItemDropRate() == 4.0`, `QuestExpRate() == 5.0`.

- [x] **Step 2: Run test to verify it fails**

```
go test ./rates/ -v
```

Expected: FAIL to compile — `NewBuilder` undefined.

- [x] **Step 3: Write the implementation**

`rates/builder.go` (new): `Builder` with the four `float64` fields, `NewBuilder()` seeding all four to `1.0`, the four `Set*` methods, and `Build() Model` (no error — nothing to validate).

`rates/processor.go` (new):

```go
package rates

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

type Processor interface {
	GetForCharacter(ch channel.Model, characterId uint32) Model
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetForCharacter(ch channel.Model, characterId uint32) Model {
	return GetForCharacter(p.l)(p.ctx)(ch, characterId)
}
```

`rates/mock/processor.go` (new): `ProcessorMock{GetForCharacterFunc func(ch channel.Model, characterId uint32) rates.Model}`, nil-func fallthrough returning `rates.Default()` (**not** the zero `rates.Model{}` — a zero EXP rate would silently award nothing in any test that forgets to set the func).

`map/processor.go` (new): package `_map`, `Processor` with `CharacterIdsInField(f field.Model) ([]uint32, error)`, `ProcessorImpl{l, ctx}`, `NewProcessor`, `var _ Processor = (*ProcessorImpl)(nil)`, and

```go
func (p *ProcessorImpl) CharacterIdsInField(f field.Model) ([]uint32, error) {
	return CharacterIdsInFieldModelProvider(p.l)(p.ctx)(f)()
}
```

`map/mock/processor.go` (new): `ProcessorMock{CharacterIdsInFieldFunc func(f field.Model) ([]uint32, error)}`, nil-func fallthrough returning `[]uint32{}, nil`.

- [x] **Step 4: Run tests**

```
go build ./... && go test ./rates/... ./map/... -v
```

Expected: PASS. `map/provider_drain_test.go` must stay green untouched.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/rates/ services/atlas-monster-death/atlas.com/monster/map/
git commit -m "feat(atlas-monster-death): add rates and map processors with mocks"
```

---

### Task 5: `atlas-monster-death` — `system_message` producer package

A new root-level `system_message` package producing `SHOW_HINT` on `COMMAND_TOPIC_SYSTEM_MESSAGE`. The command structs are **copied**, not imported: reaching into `atlas-channel`'s or `atlas-saga-orchestrator`'s internals across a service boundary is forbidden, and every other cross-service contract in this repo is duplicated the same way.

The deployment already provides the topic — `deploy/k8s/base/atlas-monster-death.yaml:20-22` uses `envFrom: configMapRef: atlas-env` and `deploy/k8s/base/env-configmap.yaml:92` already defines `COMMAND_TOPIC_SYSTEM_MESSAGE`. No deployment change here.

### Interfaces

- Produces: `system_message.Processor` with `ShowHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error`; `system_message.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`; `system_message.EnvCommandTopic`, `system_message.CommandShowHint`, `system_message.Command[E]`, `system_message.ShowHintBody`; `systemmessagemock.ProcessorMock{ShowHintFunc ...}` in `system_message/mock`.

### Files

- `services/atlas-monster-death/atlas.com/monster/system_message/kafka.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/system_message/producer.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/system_message/processor.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/system_message/mock/processor.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/system_message/producer_test.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/character/producer.go` — read-only; the local producer shape to mirror
- `services/atlas-monster-death/atlas.com/monster/character/processor.go` — read-only; `producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(...)` is the call shape that carries tenant and span headers

Module root: `services/atlas-monster-death/atlas.com/monster`.

Patterns to copy: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/system_message/producer.go:94-110` (`ShowHintCommandProvider` — the exact wire shape) and `.../system_message/processor.go:1-60` (Processor/NewProcessor). Struct definitions: `services/atlas-channel/atlas.com/channel/kafka/message/system_message/kafka.go:11-33` and `:62-67`. Only the `SHOW_HINT` command is needed — do **not** port the other ten command types.

- [x] **Step 1: Write the failing test**

New file `system_message/producer_test.go`, package `system_message`.

`TestShowHintCommandProvider_WireShape` — build `ch := channel.NewModel(world.Id(0), channel.Id(1))` (`libs/atlas-constants/channel/model.go:22`).

Then:

```
tid := uuid.MustParse("11111111-2222-3333-4444-555555555555")
msgs, err := ShowHintCommandProvider(tid, ch, 12345, "hint text", 0, 0)()
```

Assert `err == nil`, `len(msgs) == 1`, `bytes.Equal(msgs[0].Key, producer.CreateKey(12345))`, and that `json.Unmarshal(msgs[0].Value, &Command[ShowHintBody]{})` yields exactly:

| field | expected |
|---|---|
| `transactionId` | `11111111-2222-3333-4444-555555555555` |
| `worldId` | `0` |
| `channelId` | `1` |
| `characterId` | `12345` |
| `type` | `"SHOW_HINT"` |
| `body.hint` | `"hint text"` |
| `body.width` | `0` |
| `body.height` | `0` |

Also assert the raw JSON key names by unmarshalling into `map[string]any` and checking the exact keys `transactionId`, `worldId`, `channelId`, `characterId`, `type`, `body`, and inside `body`: `hint`, `width`, `height`. This is the contract `atlas-channel` consumes; a renamed tag is a silent break.

- [x] **Step 2: Run test to verify it fails**

```
go test ./system_message/ -v
```

Expected: FAIL to compile — package does not exist.

- [x] **Step 3: Write the implementation**

`system_message/kafka.go`:

```go
package system_message

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_SYSTEM_MESSAGE"

	CommandShowHint = "SHOW_HINT"
)

// Command mirrors atlas-channel's system_message command envelope. It is
// duplicated rather than imported: reaching across a service boundary for
// another service's internals is forbidden, and every cross-service contract
// in this repo is carried as a local copy.
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

// ShowHintBody is the body for showing a hint box to a character.
type ShowHintBody struct {
	Hint   string `json:"hint"`   // Hint text to display
	Width  uint16 `json:"width"`  // Width of the hint box (0 for auto-calculation)
	Height uint16 `json:"height"` // Height of the hint box (0 for auto-calculation)
}
```

`system_message/producer.go`: `ShowHintCommandProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) model.Provider[[]kafka.Message]` built with `producer.CreateKey(int(characterId))` and `producer.SingleMessageProvider(key, value)`, exactly as the saga-orchestrator reference.

`system_message/processor.go`: `Processor` interface with only `ShowHint(...)`, `ProcessorImpl{l, ctx}`, `NewProcessor`, `var _ Processor = (*ProcessorImpl)(nil)`, and

```go
func (p *ProcessorImpl) ShowHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(ShowHintCommandProvider(transactionId, ch, characterId, hint, width, height))
}
```

`system_message/mock/processor.go`: `ProcessorMock{ShowHintFunc func(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error}`, `var _ system_message.Processor = (*ProcessorMock)(nil)`, nil-func fallthrough returning `nil`.

- [x] **Step 4: Run tests**

```
go build ./... && go test ./system_message/... -v
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/system_message/
git commit -m "feat(atlas-monster-death): add SHOW_HINT system message producer (FR-6.7)"
```

---

### Task 6: `atlas-monster-death` — per-character hint throttle

D10: an in-process, per-replica throttle keyed by `(tenantId, characterId)` with a one-minute window. `atlas-monster-death` runs `replicas: 2` (`deploy/k8s/base/atlas-monster-death.yaml:6`), so the worst case is one hint per replica per minute per character — bounded, and far better than one per kill.

### Interfaces

- Consumes: nothing (the throttle is independent of `Processor`).
- Produces: `system_message.Throttle` with `Allow(tenantId uuid.UUID, characterId uint32) bool`; `system_message.NewThrottle(window time.Duration, capacity int, now func() time.Time) *Throttle`; `system_message.GetHintThrottle() *Throttle` (process singleton: one-minute window, capacity 4096, `time.Now`).

### Files

- `services/atlas-monster-death/atlas.com/monster/system_message/throttle.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/system_message/throttle_test.go` — **new file**

Module root: `services/atlas-monster-death/atlas.com/monster`.

- [x] **Step 1: Write the failing test**

New file `system_message/throttle_test.go`, package `system_message`. Every case drives a fake clock through a captured `time.Time` variable and `func() time.Time { return clock }` — no `time.Sleep`, no real clock.

`TestThrottle_Allow` — construct `NewThrottle(time.Minute, 4096, clockFn)` per case; `tA` and `tB` are `uuid.New()` values fixed per test.

| case | sequence | expected results |
|---|---|---|
| `first call allowed` | `Allow(tA, 1)` | `true` |
| `second call inside window denied` | `Allow(tA, 1)`; advance 30s; `Allow(tA, 1)` | `true`, `false` |
| `call after window allowed` | `Allow(tA, 1)`; advance 61s; `Allow(tA, 1)` | `true`, `true` |
| `boundary: exactly the window is allowed` | `Allow(tA, 1)`; advance exactly 60s; `Allow(tA, 1)` | `true`, `true` (the check is `elapsed < window`) |
| `distinct characters are independent` | `Allow(tA, 1)`; `Allow(tA, 2)` at the same instant | `true`, `true` |
| `distinct tenants are independent` | `Allow(tA, 1)`; `Allow(tB, 1)` at the same instant | `true`, `true` |

`TestThrottle_SweepsWhenOverCapacity` — `NewThrottle(time.Minute, 4, clockFn)`; `Allow(tA, 1..4)` at t0 (all `true`); advance 61s; `Allow(tA, 5)` → `true`, and assert `len(th.last) <= 4` (in-package access) proving the four stale entries were swept rather than the map growing unbounded. Then, without advancing, `Allow(tA, 5)` → `false`.

`TestThrottle_ConcurrentAllowIsRaceFree` — 50 goroutines each calling `Allow(tA, uint32(i))` on a shared throttle with `time.Now`, joined by a `sync.WaitGroup`. Assert no panic; the value assertion is that exactly 50 distinct characters were admitted. This exists so `go test -race` covers the mutex.

- [x] **Step 2: Run test to verify it fails**

```
go test ./system_message/ -run TestThrottle -v
```

Expected: FAIL to compile — `NewThrottle` undefined.

- [x] **Step 3: Write the implementation**

`system_message/throttle.go`:

```go
package system_message

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type throttleKey struct {
	tenantId    uuid.UUID
	characterId uint32
}

// Throttle bounds how often a given (tenant, character) pair may be sent a
// hint. State is per-process: atlas-monster-death runs multiple replicas, so
// the effective bound is one hint per replica per window (D10). That is an
// approximation of Cosmic's per-character nextWarningTime, chosen over shared
// state in Redis because a cosmetic notice does not justify a new
// infrastructure dependency.
type Throttle struct {
	mu       sync.Mutex
	window   time.Duration
	capacity int
	now      func() time.Time
	last     map[throttleKey]time.Time
}

func NewThrottle(window time.Duration, capacity int, now func() time.Time) *Throttle

// GetHintThrottle returns the process-wide hint throttle: a one-minute window,
// a 4096-key cap, and the real clock.
func GetHintThrottle() *Throttle

// Allow reports whether a hint may be emitted now, recording the emission when
// it returns true.
func (t *Throttle) Allow(tenantId uuid.UUID, characterId uint32) bool
```

`Allow` body: lock; `n := t.now()`; if a prior time exists and `n.Sub(prior) < t.window`, return `false`; otherwise, if `len(t.last) >= t.capacity`, delete every key whose recorded time is older than `n.Add(-t.window)`; then record `t.last[k] = n` and return `true`.

`GetHintThrottle` uses a package-level `sync.Once` plus a package-level `*Throttle`, mirroring how `GetMonsterRegistry()` is done in `atlas-monsters` (`services/atlas-monsters/atlas.com/monsters/monster/registry.go`).

- [x] **Step 4: Run tests**

```
go build ./... && go test ./system_message/... -race -v
```

Expected: PASS with no race reports.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/system_message/throttle.go services/atlas-monster-death/atlas.com/monster/system_message/throttle_test.go
git commit -m "feat(atlas-monster-death): throttle level-gate hints per tenant and character (D10)"
```

---

### Task 7: `atlas-monster-death` — `intervalSet` (level-gate interval union)

A port of `<cosmic>/src/main/java/tools/IntervalBuilder.java`. FR-6.2 is explicit that this is a **union of possibly-overlapping ranges**, not a single `[min−5, max+5]` band; an implementation that takes the min and max is incorrect.

Arithmetic is done in `int` with a `lo < 0 → 0` clamp, because mob level is `uint32` and member level is `byte`: `5 - 5 - 1` under either unsigned type wraps to an enormous value.

### Interfaces

- Produces: unexported `intervalSet` in package `monster` with `add(lo, hi int)`, `build() intervalSet`, `contains(v int) bool`.

### Files

- `services/atlas-monster-death/atlas.com/monster/monster/interval.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/monster/interval_test.go` — **new file**

Module root: `services/atlas-monster-death/atlas.com/monster`. Package `monster` (the same package as `processor.go`), so `calculateExperienceStandardDeviationThreshold` and `isWhiteExperienceGain` stay reachable without export churn.

- [x] **Step 1: Write the failing test**

New file `monster/interval_test.go`, package `monster`.

`TestIntervalSet_Contains` — each case builds a fresh `intervalSet`, calls `add` for each listed range in the listed order, calls `build()`, then asserts `contains(v)` for each probe.

| case | adds | probes → expected |
|---|---|---|
| `single interval` | (25,35) | 24→false, 25→true, 30→true, 35→true, 36→false |
| `PRD worked example` | (120,130) mob 125±5, then (25,35) contributor 30±5, then (115,125) contributor 120±5 | 32→true, 70→false, 25→true, 35→true, 115→true, 130→true, 36→false, 114→false |
| `adjacent intervals merge` | (0,10), (11,20) | 10→true, 11→true, 20→true, 21→false |
| `overlapping intervals merge` | (0,10), (5,20) | 0→true, 15→true, 20→true, 21→false |
| `disjoint intervals stay disjoint` | (0,5), (100,105) | 5→true, 6→false, 99→false, 100→true |
| `negative lo clamps to zero` | (-5, 3) | 0→true, 3→true, 4→false |
| `unsorted adds` | (100,110), (0,10), (50,60) | 5→true, 55→true, 105→true, 30→false, 70→false |
| `empty set contains nothing` | — | 0→false, 50→false |

The `PRD worked example` row is FR-6.2 verbatim (contributors at 30 and 120, mob at 125, `LEVEL_INTERVAL = LEACH_INTERVAL = 5`): a `[min−5, max+5]` implementation would admit 70 and fail this row.

`TestIntervalSet_BuildMergesRanges` — after `add(0,10); add(5,20); add(100,105)` and `build()`, assert `len(s.ivs) == 2` and that the merged elements are exactly `{lo:0, hi:20}` and `{lo:100, hi:105}` in that order. This pins the merge itself rather than only its observable effect.

`TestIntervalSet_BuildIsIdempotent` — `build()` on an already-built set returns an equal set (`reflect.DeepEqual`).

- [x] **Step 2: Run test to verify it fails**

```
go test ./monster/ -run TestIntervalSet -v
```

Expected: FAIL to compile — `intervalSet` undefined.

- [x] **Step 3: Write the implementation**

`monster/interval.go`:

```go
package monster

import "sort"

type interval struct {
	lo int
	hi int
}

// intervalSet is a union of possibly-overlapping level bands, ported from
// <cosmic> tools/IntervalBuilder.java. FR-6.2 is explicit that the level gate
// is a union and NOT a single [min-5, max+5] band: contributors at levels 30
// and 120 against a level-125 mob must admit a level-32 member and reject a
// level-70 one.
type intervalSet struct {
	ivs []interval
}

// add records one band. lo is clamped at 0 because callers do unsigned level
// arithmetic in int and a low-level mob can produce a negative lower bound.
// Merging is deferred to build() so add stays O(1).
func (s *intervalSet) add(lo, hi int) {
	if lo < 0 {
		lo = 0
	}
	if hi < lo {
		return
	}
	s.ivs = append(s.ivs, interval{lo: lo, hi: hi})
}

// build returns a copy with the bands sorted by lo and overlapping or
// adjacent bands merged.
func (s intervalSet) build() intervalSet { /* sort by lo, then single merge pass; [a,b] and [c,d] merge when c <= b+1 */ }

// contains reports whether v falls in any band. A linear scan is correct and
// clearer than a binary search at these sizes: one mob band plus at most six
// contributor bands.
func (s intervalSet) contains(v int) bool { /* linear scan over s.ivs */ }
```

Fill the two elided bodies. `build` must not mutate the receiver's backing array — copy `s.ivs` into a fresh slice before sorting, so `TestIntervalSet_BuildIsIdempotent` holds and a caller that keeps the unbuilt set is not surprised.

- [x] **Step 4: Run tests**

```
go test ./monster/ -run TestIntervalSet -v && go build ./...
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/monster/interval.go services/atlas-monster-death/atlas.com/monster/monster/interval_test.go
git commit -m "feat(atlas-monster-death): add interval-union level gate primitive (FR-6.2)"
```

---

### Task 8: `atlas-monster-death` — `ExperienceConfig` and its env loader

D9: a struct built from Go defaults, with the three gate settings overridable by env var. An absent or unparseable key is not an error — it falls back to the default. The last three settings deliberately have **no** env key: they are game-balance constants, not deployment knobs, and giving them one invites drift between replicas of the same world.

### Interfaces

- Produces: `monster.ExperienceConfig{EnforceMobLevelRange bool; LevelInterval uint32; LeachInterval uint32; SplitCommonMod float64; MvpMod float64; PartyBonusPerMember float64}` (exported fields — this is a config value object, not a domain model); `monster.DefaultExperienceConfig() ExperienceConfig`; `monster.LoadExperienceConfig() ExperienceConfig`.

### Files

- `services/atlas-monster-death/atlas.com/monster/monster/config.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/monster/config_test.go` — **new file**
- `deploy/k8s/base/env-configmap.yaml` — add the three keys
- `deploy/k8s/base/atlas-monster-death.yaml` — read-only; already `envFrom: configMapRef: atlas-env`, so no change

Module root: `services/atlas-monster-death/atlas.com/monster`. Package `monster`.

- [x] **Step 1: Write the failing test**

New file `monster/config_test.go`, package `monster`. Each case sets env with `t.Setenv` (which restores automatically) and calls `LoadExperienceConfig()`.

`TestDefaultExperienceConfig` — assert every field: `EnforceMobLevelRange == true`, `LevelInterval == 5`, `LeachInterval == 5`, `SplitCommonMod == 0.8`, `MvpMod == 0.2`, `PartyBonusPerMember == 0.05`.

`TestLoadExperienceConfig` — table-driven:

| case | env set | expected |
|---|---|---|
| `no env is defaults` | (all three explicitly `t.Setenv(k, "")`) | equal to `DefaultExperienceConfig()` |
| `gate disabled` | `USE_ENFORCE_MOB_LEVEL_RANGE=false` | `EnforceMobLevelRange == false`, other five at defaults |
| `gate explicitly enabled` | `USE_ENFORCE_MOB_LEVEL_RANGE=true` | `EnforceMobLevelRange == true` |
| `intervals overridden` | `LEVEL_INTERVAL=10`, `LEACH_INTERVAL=3` | `LevelInterval == 10`, `LeachInterval == 3` |
| `unparseable bool falls back` | `USE_ENFORCE_MOB_LEVEL_RANGE=maybe` | `EnforceMobLevelRange == true` |
| `unparseable interval falls back` | `LEVEL_INTERVAL=abc` | `LevelInterval == 5` |
| `zero interval is honoured` | `LEVEL_INTERVAL=0` | `LevelInterval == 0` (0 is a valid tightening, not a parse failure) |
| `balance constants are never env-driven` | `SPLIT_COMMON_MOD=9`, `MVP_MOD=9`, `PARTY_BONUS_PER_MEMBER=9` | `SplitCommonMod == 0.8`, `MvpMod == 0.2`, `PartyBonusPerMember == 0.05` |

- [x] **Step 2: Run test to verify it fails**

```
go test ./monster/ -run TestLoadExperienceConfig -v
```

Expected: FAIL to compile — `LoadExperienceConfig` undefined.

- [x] **Step 3: Write the implementation**

`monster/config.go`:

```go
package monster

import (
	"os"
	"strconv"
)

const (
	EnvEnforceMobLevelRange = "USE_ENFORCE_MOB_LEVEL_RANGE"
	EnvLevelInterval        = "LEVEL_INTERVAL"
	EnvLeachInterval        = "LEACH_INTERVAL"
)

// ExperienceConfig carries the EXP-distribution tunables. The three gate
// settings mirror <cosmic>/config.yaml:243 and :293-294 and are overridable
// per deployment. The three split mods are game-balance constants with no env
// key on purpose: an env-tunable split would let two replicas of the same
// world award different EXP for the same kill.
type ExperienceConfig struct {
	EnforceMobLevelRange bool
	LevelInterval        uint32
	LeachInterval        uint32
	SplitCommonMod       float64
	MvpMod               float64
	PartyBonusPerMember  float64
}

func DefaultExperienceConfig() ExperienceConfig {
	return ExperienceConfig{
		EnforceMobLevelRange: true,
		LevelInterval:        5,
		LeachInterval:        5,
		SplitCommonMod:       0.8,
		MvpMod:               0.2,
		PartyBonusPerMember:  0.05,
	}
}

// LoadExperienceConfig layers env overrides over the defaults. An absent or
// unparseable key keeps the default rather than failing the service: EXP
// tuning must never be able to prevent a replica from starting.
func LoadExperienceConfig() ExperienceConfig { /* strconv.ParseBool / strconv.ParseUint(v, 10, 32) with default fallthrough */ }
```

Fill `LoadExperienceConfig`.

Then add to `deploy/k8s/base/env-configmap.yaml`, keeping the file's alphabetical grouping: `LEACH_INTERVAL: "5"` and `LEVEL_INTERVAL: "5"` immediately before `REDIS_URL` (currently line 193), and `USE_ENFORCE_MOB_LEVEL_RANGE: "true"` after `TRACE_SAMPLING_RATIO` (currently line 196).

- [x] **Step 4: Run tests**

```
go build ./... && go test ./monster/ -run 'TestDefaultExperienceConfig|TestLoadExperienceConfig' -v
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/monster/config.go services/atlas-monster-death/atlas.com/monster/monster/config_test.go deploy/k8s/base/env-configmap.yaml
git commit -m "feat(atlas-monster-death): add ExperienceConfig with env-backed level gate settings (D9)"
```

---

### Task 9: `atlas-monster-death` — plan value types, `computeAward`, hint text

The pure award arithmetic and the value types the planner will produce. `planDistribution` itself is Task 10; this task lands the types it returns so the two can be reviewed separately.

`computeAward` preserves today's ordering — the rate is applied to the pooled EXP **before** the split (`processor.go:135-137` then `:230-240`), which is what makes FR-8.2 (bonus is also rate-multiplied) fall out for free.

### Interfaces

- Consumes: `ExperienceConfig` from Task 8.
- Produces: `Recipient`, `Exclusion`, `ExperiencePlan`, `DamageInput`, `SoloInput`, `MemberInput`, `PartyInput`, `ExperienceInput` (all in package `monster`); `computeAward(r Recipient, expRate float64, cfg ExperienceConfig) (personal uint32, bonus uint32, guarded bool)`; `levelGateHintText(name string, level uint32) string`.

### Files

- `services/atlas-monster-death/atlas.com/monster/monster/experience.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/monster/experience_test.go` — **new file**

Module root: `services/atlas-monster-death/atlas.com/monster`. Package `monster`.

- [x] **Step 1: Write the failing test**

New file `monster/experience_test.go`, package `monster`. `cfg := DefaultExperienceConfig()` unless a row says otherwise. Compare floats-turned-uint32 exactly; compare no floats.

`TestComputeAward` — table-driven; each row constructs a `Recipient` literal and asserts `(personal, bonus, guarded)`.

| case | Recipient (Level, PooledExp, TotalPartyLevel, PartyBonusMod, IsMvp) | expRate | expect personal | expect bonus | guarded |
|---|---|---|---|---|---|
| `solo identity` | 50, 1000, 50, 0.0, true | 1.0 | 1000 | 0 | false |
| `solo with rate` | 50, 1000, 50, 0.0, true | 2.0 | 2000 | 0 | false |
| `one-member party equals solo (FR-5.11)` | 50, 1000, 50, 0.0, true | 1.0 | 1000 | 0 | false |
| `two-member party, MVP` | 50, 1000, 100, 0.10, true | 1.0 | 600 | 60 | false |
| `two-member party, non-MVP` | 50, 1000, 100, 0.10, false | 1.0 | 400 | 40 | false |
| `unequal levels, MVP` | 100, 1000, 150, 0.10, true | 1.0 | 733 | 73 | false |
| `unequal levels, non-MVP` | 50, 1000, 150, 0.10, false | 1.0 | 266 | 26 | false |
| `bonus is rate-multiplied (FR-8.2)` | 50, 1000, 100, 0.10, true | 3.0 | 1800 | 180 | false |
| `four-member party bonus` | 50, 1000, 200, 0.20, false | 1.0 | 200 | 40 | false |
| `zero pooled exp` | 50, 0, 100, 0.10, true | 1.0 | 0 | 0 | false |
| `zero rate` | 50, 1000, 100, 0.10, true | 0.0 | 0 | 0 | false |
| `zero totalPartyLevel is guarded (FR-8.6)` | 50, 1000, 0, 0.10, true | 1.0 | 0 | 0 | **true** |
| `zero level, non-zero party level` | 0, 1000, 100, 0.0, false | 1.0 | 0 | 0 | false |
| `overflow clamps` | 50, 1e18, 50, 0.0, true | 1.0 | 4294967295 | 0 | **true** |

Derivations for the reviewer, using `share = 0.8·L/ΣL (+0.2 if MVP)`, `personal = share · pooled · rate`, `bonus = partyBonusMod · personal`:
- `two-member party, MVP`: `0.8·50/100 + 0.2 = 0.6` → `600`; `0.10·600 = 60`.
- `two-member party, non-MVP`: `0.8·50/100 = 0.4` → `400`; `0.10·400 = 40`.
- `unequal levels, MVP`: `0.8·100/150 + 0.2 = 0.7333…` → `733.33…` → truncates to `733`; `0.10·733.33… = 73.33…` → `73`.
- `unequal levels, non-MVP`: `0.8·50/150 = 0.26666…` → `266.66…` → `266`; `0.10·266.66… = 26.66…` → `26`.
- `four-member party bonus`: `0.8·50/200 = 0.2` → `200`; `0.20·200 = 40`.

`TestComputeAward_NonFiniteIsGuarded` — separate from the table because the inputs cannot be written as plain literals. Three cases, each asserting `(0, 0, true)`: `PooledExp: math.NaN()`; `PooledExp: math.Inf(1)`; `expRate: math.NaN()`. Nothing non-finite may be cast to `uint32` — the conversion is implementation-defined in Go and must never reach the wire.

`TestLevelGateHintText` — `levelGateHintText("Blue Snail", 2)` equals exactly:

```
You have gained #rno experience#k from defeating #e#bBlue Snail#k#n (lv. #b2#k)! Take note you must have around the same level as the mob to start earning EXP from it.
```

Assert with `!=` on the whole string, not `strings.Contains`.

- [x] **Step 2: Run test to verify it fails**

```
go test ./monster/ -run 'TestComputeAward|TestLevelGateHintText' -v
```

Expected: FAIL to compile — `Recipient` and `computeAward` undefined.

- [x] **Step 3: Write the implementation**

`monster/experience.go`:

```go
package monster

import (
	"fmt"
	"math"
)

// DamageInput is one aggregated damage entry: exactly one per damaging
// character, in-field or not.
type DamageInput struct {
	CharacterId uint32
	Damage      uint32
}

// SoloInput is an in-field damager with no party.
type SoloInput struct {
	CharacterId uint32
	Level       byte
}

// MemberInput is one co-located party member. Membership here already means
// "in the field where the kill happened" -- co-location comes from the
// atlas-maps character list, never from the party service's member location
// fields, which are only eventually consistent (D11).
type MemberInput struct {
	CharacterId uint32
	Level       byte
}

// PartyInput is one participating party and its co-located members, whether or
// not they dealt damage.
type PartyInput struct {
	PartyId uint32
	Members []MemberInput
}

// ExperienceInput is everything the planner needs. Damages carries EVERY
// damager including those who left the field, because out-of-field damagers
// still count toward totalDamage and totalEntries even though they are never
// party-resolved and receive nothing (D12).
type ExperienceInput struct {
	MonsterExperience uint32
	MonsterLevel      uint32
	Damages           []DamageInput
	Solos             []SoloInput
	Parties           []PartyInput
}

// Recipient is one character who will receive an award. PooledExp is the
// party's whole participation EXP, not this member's share -- the split is
// applied by computeAward.
type Recipient struct {
	CharacterId     uint32
	Level           byte
	PartyId         uint32
	PooledExp       float64
	TotalPartyLevel uint32
	PartyBonusMod   float64
	IsMvp           bool
	White           bool
}

// Exclusion is a co-located party member the level gate kept out.
type Exclusion struct {
	CharacterId uint32
}

type ExperiencePlan struct {
	Recipients             []Recipient
	Exclusions             []Exclusion
	TotalDamage            uint32
	TotalEntries           int
	ExperiencePerDamage    float64
	StandardDeviationRatio float64
}

// computeAward applies the character's EXP rate to the pooled figure BEFORE
// the split, which is the existing ordering and is what makes FR-8.2 (party
// bonus is rate-multiplied too) hold without a second multiplication.
//
// guarded reports that a value was not representable and was replaced: a
// non-finite intermediate becomes 0, and a value at or above MaxUint32 is
// clamped. uint32(NaN) is implementation-defined in Go and must never reach
// the wire (FR-8.6).
func computeAward(r Recipient, expRate float64, cfg ExperienceConfig) (uint32, uint32, bool) {
	if r.TotalPartyLevel == 0 {
		return 0, 0, true
	}
	exp := r.PooledExp * expRate
	share := cfg.SplitCommonMod * float64(r.Level) / float64(r.TotalPartyLevel)
	if r.IsMvp {
		share += cfg.MvpMod
	}
	personalF := share * exp
	bonusF := r.PartyBonusMod * personalF

	personal, pg := toAwardAmount(personalF)
	bonus, bg := toAwardAmount(bonusF)
	return personal, bonus, pg || bg
}

// toAwardAmount converts a computed EXP figure to the uint32 the award command
// carries, reporting whether the value had to be replaced.
func toAwardAmount(v float64) (uint32, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, true
	}
	if v < 0 {
		return 0, true
	}
	if v >= math.MaxUint32 {
		return math.MaxUint32, true
	}
	return uint32(v), false
}

// levelGateHintText renders Cosmic's level-gate notice (Character.java:9246).
func levelGateHintText(name string, level uint32) string {
	return fmt.Sprintf("You have gained #rno experience#k from defeating #e#b%s#k#n (lv. #b%d#k)! Take note you must have around the same level as the mob to start earning EXP from it.", name, level)
}
```

- [x] **Step 4: Run tests**

```
go build ./... && go test ./monster/ -run 'TestComputeAward|TestLevelGateHintText' -v
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/monster/experience.go services/atlas-monster-death/atlas.com/monster/monster/experience_test.go
git commit -m "feat(atlas-monster-death): add EXP plan value types and award computation (FR-8)"
```

---

### Task 10: `atlas-monster-death` — `planDistribution`

The pure planner. This is where every party rule lives, and it takes no logger, no context, no clock, and no collaborator — so every numeric acceptance criterion in the PRD becomes a mock-free table test.

Gate ordering is fixed and matters: **the gate selects recipients only.** `partyDamage`, `participationExp`, the contributor list, and the interval set itself are all computed *before* the gate, from the ungated contributor list. A gated-out member who dealt damage still widens the interval for everyone else and still contributes to a pool they do not share in — that is Cosmic's behaviour (`Monster.java:549-600`) and is what makes the interval union meaningful (D14).

### Interfaces

- Consumes: `ExperienceConfig` (Task 8), `intervalSet` (Task 7), all plan value types plus `levelGateHintText` (Task 9), and the untouched `calculateExperienceStandardDeviationThreshold` / `isWhiteExperienceGain` in `processor.go:207-229` (D4).
- Produces: `planDistribution(in ExperienceInput, cfg ExperienceConfig) ExperiencePlan`.

### Files

- `services/atlas-monster-death/atlas.com/monster/monster/experience.go` — append `planDistribution`
- `services/atlas-monster-death/atlas.com/monster/monster/experience_test.go` — append the planner tests
- `services/atlas-monster-death/atlas.com/monster/monster/processor.go` — read-only in this task; `calculateExperienceStandardDeviationThreshold` (207-221) and `isWhiteExperienceGain` (223-229) are called from `planDistribution` and must not be edited

Module root: `services/atlas-monster-death/atlas.com/monster`. Package `monster`.

- [x] **Step 1: Write the failing test**

Append to `monster/experience_test.go`.

`TestPlanDistribution` — table-driven. Each row builds an `ExperienceInput` and asserts `TotalDamage`, `TotalEntries`, the full `Recipients` slice field-by-field in order, and the full `Exclusions` slice. `cfg := DefaultExperienceConfig()` unless the row says otherwise. Every input uses `MonsterExperience: 1000` and `MonsterLevel: 100` unless stated.

| case | input | expected |
|---|---|---|
| `solo single damager` | Damages `[{1,500}]`, Solos `[{1,50}]`, Parties none | TotalDamage 500, TotalEntries 1, epd `1000/500 = 2.0`; one Recipient `{CharacterId:1, Level:50, PartyId:0, PooledExp:1000, TotalPartyLevel:50, PartyBonusMod:0, IsMvp:true, White:true}`; no Exclusions |
| `two solo damagers` | Damages `[{1,750},{2,250}]`, Solos `[{1,50},{2,50}]` | TotalDamage 1000, TotalEntries 2; Recipients ascending id: `{1, PooledExp:750, TotalPartyLevel:50, IsMvp:true, White:true}`, `{2, PooledExp:250, TotalPartyLevel:50, IsMvp:true, White:false}` (ratios 0.75/0.25, mean 0.5, sd 0.25, threshold 0.75 — 0.75 ≥ 0.75 is white, 0.25 is not) |
| `zero total damage` | Damages `[{1,0}]`, Solos `[{1,50}]` | TotalDamage 0, TotalEntries 1, **no Recipients**, no Exclusions, `ExperiencePerDamage == 0` — the caller warns and returns; nothing may divide by zero (FR-3.3) |
| `empty input` | all empty | TotalDamage 0, TotalEntries 0, no Recipients, no Exclusions |
| `two-member party, one damager (FR-5.3)` | Damages `[{1,1000}]`, Parties `[{PartyId:9, Members:[{1,50},{2,50}]}]` | TotalDamage 1000, TotalEntries 1; Recipients `{1, PartyId:9, PooledExp:1000, TotalPartyLevel:100, PartyBonusMod:0.10, IsMvp:true, White:true}`, `{2, PartyId:9, PooledExp:1000, TotalPartyLevel:100, PartyBonusMod:0.10, IsMvp:false, White:false}` — the non-damager is a recipient with **yellow** EXP (FR-4.3) |
| `one-member party equals solo (FR-5.11)` | Damages `[{1,1000}]`, Parties `[{PartyId:9, Members:[{1,50}]}]` | one Recipient `{1, PooledExp:1000, TotalPartyLevel:50, PartyBonusMod:0.0, IsMvp:true, White:true}` — field-for-field identical to the `solo single damager` recipient except `PartyId` |
| `four-member party bonus (FR-5.8)` | Damages `[{1,1000}]`, Parties `[{9, [{1,50},{2,50},{3,50},{4,50}]}]` | every Recipient has `PartyBonusMod == 0.20`, `TotalPartyLevel == 200` |
| `MVP is highest damager in expMembers` | Damages `[{1,100},{2,900}]`, Parties `[{9, [{1,50},{2,50}]}]` | `IsMvp` true for 2, false for 1; both `PooledExp == 1000` |
| `MVP tie breaks to lowest characterId (D13)` | Damages `[{2,500},{1,500}]`, Parties `[{9, [{1,50},{2,50}]}]` | `IsMvp` true for 1, false for 2 |
| `MVP falls to a non-damager when no member damaged` | MonsterLevel **50**; Damages `[{7,1000}]` (7 is out of field), Parties `[{9, [{1,50},{2,50}]}]` | partyDamage 0 → `PooledExp == 0` for both; `IsMvp` true for 1 only; TotalEntries 2 (one party + one out-of-field damager). MonsterLevel is 50 here because with no contributors the interval set is the mob band alone, and a level-100 mob would gate both level-50 members out |
| `out-of-field damager counts but receives nothing (D12)` | Damages `[{1,500},{7,500}]`, Solos `[{1,50}]` (7 absent from Solos and every party) | TotalDamage 1000, TotalEntries 2; exactly one Recipient (id 1) with `PooledExp == 500` |
| `level gate excludes and does not count (FR-6.1)` | MonsterLevel 125; Damages `[{1,1000}]`, Parties `[{9, [{1,120},{2,32},{3,70}]}]` | intervals `{[120,130] mob, [115,125] contributor}` merge to `[115,130]`; the only Recipient is 1 (`Level:120`, `TotalPartyLevel == 120`, `PartyBonusMod == 0.0` — a single eligible member is not a sharer); Exclusions `[{2},{3}]`. Levels 32 and 70 are neither counted in `TotalPartyLevel` nor allowed to raise `PartyBonusMod` |
| `interval union admits and rejects (FR-6.2)` | MonsterLevel 125; Damages `[{1,600},{2,400}]`, Parties `[{9, [{1,120},{2,30},{3,32},{4,70}]}]` | merged set `{[25,35],[115,130]}`; Recipients 1, 2, 3; Exclusions `[{4}]`; `TotalPartyLevel == 182`; `PartyBonusMod == 0.15` |
| `gate disabled admits everyone` | same input as the row above, `cfg.EnforceMobLevelRange = false` | Recipients 1, 2, 3, 4; no Exclusions; `TotalPartyLevel == 252`; `PartyBonusMod == 0.20` |
| `gate never applies to solo (FR-6.3)` | MonsterLevel 125; Damages `[{1,1000}]`, Solos `[{1,5}]` | one Recipient (id 1) — a level-5 solo killer of a level-125 mob is never gated; no Exclusions |
| `a contributor's band widens the set and their damage feeds the pool (D14)` | MonsterLevel 200; Damages `[{1,500},{2,500}]`, Parties `[{9, [{1,30},{2,199},{3,32},{4,100}]}]` | merged set `{[25,35],[194,205]}`; Recipients 1, 2, 3; Exclusions `[{4}]`; `TotalPartyLevel == 261`; every recipient's `PooledExp == 1000·epd`. Member 3 (level 32) is eligible **only** because contributor 1 (level 30) widened the set to a band nowhere near the mob's, and contributor 1's 500 damage is pooled for everyone. A contributor is always admitted by its own band, so the gate can never shrink `partyDamage` |
| `party with no eligible members is skipped (FR-5.10)` | MonsterLevel 200; Damages `[{1,1000}]` where 1 is out of field; Parties `[{9, [{5,10}]}]` | no Recipients; Exclusions `[{5}]`; TotalEntries 2 |
| `zero totalPartyLevel yields no recipients (FR-5.6)` | Damages `[{1,1000}]`, Parties `[{9, [{1,0}]}]`, gate disabled | no Recipients, no Exclusions — a zero divisor never reaches computeAward |
| `mixed solo and party` | Damages `[{1,600},{2,400}]`, Solos `[{1,50}]`, Parties `[{9, [{2,50},{3,50}]}]` | TotalDamage 1000, TotalEntries 2; Recipients ascending id 1, 2, 3; 1 has `PartyId 0`, `PooledExp 600`; 2 and 3 have `PartyId 9`, `PooledExp 400`, `TotalPartyLevel 100`, `PartyBonusMod 0.10`; `IsMvp` true for 1 and 2 |
| `parties are ordered by partyId, members by characterId` | Damages `[{4,100},{2,100}]`, Parties `[{PartyId:9, Members:[{4,50},{3,50}]}, {PartyId:2, Members:[{2,50},{1,50}]}]` (both slices deliberately out of order) | Recipients in exactly `[1, 2, 3, 4]` order |

`TestPlanDistribution_IsDeterministicUnderShuffledInput` (NFR-5) — take the `mixed solo and party` input, produce `want := planDistribution(in, cfg)`, then for 20 iterations build a fresh input with `Damages`, `Solos`, `Parties`, and each `PartyInput.Members` reversed, and assert `reflect.DeepEqual(planDistribution(shuffled, cfg), want)`. Reversal rather than randomisation keeps the test reproducible; the point is that the planner sorts rather than trusting its caller.

`TestPlanDistribution_TotalEntriesComposition` (PRD white/yellow acceptance) — one input with 2 solo damagers, 2 participating parties, and 3 out-of-field damagers; assert `TotalEntries == 7` directly. `totalEntries` is otherwise observable only through the stddev threshold.

- [x] **Step 2: Run test to verify it fails**

```
go test ./monster/ -run TestPlanDistribution -v
```

Expected: FAIL to compile — `planDistribution` undefined.

- [x] **Step 3: Write the implementation**

Append `planDistribution` to `monster/experience.go`. The algorithm, in order:

1. Copy and sort: `Damages` ascending `CharacterId`, `Solos` ascending `CharacterId`, `Parties` ascending `PartyId`, and each party's `Members` ascending `CharacterId`. Sort copies — never mutate the caller's slices.
2. `damageOf := map[uint32]uint32` from `Damages`; `totalDamage := Σ Damages[i].Damage`.
3. If `totalDamage == 0`: return `ExperiencePlan{TotalDamage: 0, TotalEntries: len(sorted Damages), Recipients: nil, Exclusions: nil}`. Nothing else is computed — `epd` would divide by zero.
4. `epd := float64(in.MonsterExperience) / float64(totalDamage)`.
5. `inFieldDamagers := 0`; count each solo, plus each party member with a non-zero entry in `damageOf`. `outOfField := len(Damages) - inFieldDamagers`. `totalEntries := len(Solos) + len(Parties) + outOfField`.
6. `personalRatio := map[uint32]float64{}`; `entryRatios := []float64{}`. For each solo: `ratio := float64(damageOf[id]) / float64(totalDamage)`; record in both. For each party: sum its contributors' individual ratios into `personalRatio` and append the **sum** as one element of `entryRatios` (FR-4.2). A party member with no damage gets no `personalRatio` entry, which is what makes them yellow (FR-4.3).
7. `sort.Float64s(entryRatios)` before the threshold call. Float summation is associativity-sensitive; sorting is what makes the threshold byte-reproducible rather than merely statistically stable.
8. `stdr := calculateExperienceStandardDeviationThreshold(entryRatios, totalEntries)`. Leave that function untouched (D4).
9. Solo recipients: one per solo, `PooledExp: float64(damageOf[id]) * epd`, `TotalPartyLevel: uint32(level)`, `PartyBonusMod: 0`, `IsMvp: true`, `PartyId: 0`, `White: isWhiteExperienceGain(id, personalRatio, stdr)`. Skip a solo whose `TotalPartyLevel` would be 0.
10. Party recipients, per party in ascending `PartyId`:
    - `contributors` = members with `damageOf[id] > 0`, ascending `CharacterId`.
    - `partyDamage` = Σ contributor damage. **Not** gated (D14).
    - `participationExp := float64(partyDamage) * epd`.
    - If `cfg.EnforceMobLevelRange`: build the `intervalSet` — `add(int(in.MonsterLevel)-int(cfg.LevelInterval), int(in.MonsterLevel)+int(cfg.LevelInterval))`, then for each contributor `add(int(c.Level)-int(cfg.LeachInterval), int(c.Level)+int(cfg.LeachInterval))` — `build()`, then partition members into `expMembers` (`contains(int(m.Level))`) and `excluded`. Otherwise `expMembers = members`, `excluded` empty.
    - Append every `excluded` member to the plan's `Exclusions` (they are notified even when the party ends up with no recipients).
    - `totalPartyLevel := Σ uint32(m.Level)` over `expMembers`. If `len(expMembers) == 0` or `totalPartyLevel == 0`, emit no recipients for this party and continue.
    - `mvpId`: iterate `expMembers` ascending and keep the first strict maximum of `damageOf[id]` (absent = 0). The strict `>` is what makes ties resolve to the lowest `characterId` (D13).
    - `hasPartySharers := len(expMembers) > 1`; `partyBonusMod := cfg.PartyBonusPerMember * float64(len(expMembers))` when `hasPartySharers`, else `0.0`.
    - One `Recipient` per `expMember` with `PooledExp: participationExp`, `TotalPartyLevel: totalPartyLevel`, `PartyBonusMod: partyBonusMod`, `IsMvp: m.CharacterId == mvpId`, `PartyId: p.PartyId`, `White: isWhiteExperienceGain(m.CharacterId, personalRatio, stdr)`.
11. Sort `Recipients` and `Exclusions` ascending `CharacterId` and return the plan with `TotalDamage`, `TotalEntries`, `ExperiencePerDamage: epd`, `StandardDeviationRatio: stdr`.

- [x] **Step 4: Run tests**

```
go build ./... && go test ./monster/ -v
```

Expected: PASS, including the pre-existing `processor_test.go` and `characterization_test.go`, which must not need any edit.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/monster/experience.go services/atlas-monster-death/atlas.com/monster/monster/experience_test.go
git commit -m "feat(atlas-monster-death): add pure party EXP distribution planner (FR-5, FR-6)"
```

---

### Task 11: `atlas-monster-death` — inject collaborators into `monster.ProcessorImpl`

Pure plumbing, no behaviour change: `ProcessorImpl` gains collaborator fields bound at construction, plus an `atlas-pets`-style `With(...)`. `CreateDrops` switches to the injected processors too, so there is one construction path rather than two.

The `atlas-pets` comment at `processor.go:106-114` documents the one hazard — never bind a *method value* at construction time, or a `With` clone dispatches to the original receiver. We bind processor **interfaces**, not method values, so the hazard does not apply here; do not introduce a `func` field.

### Interfaces

- Consumes: `information.Processor` + `NewProcessor` (Task 3), `rates.Processor` (Task 4), `_map.Processor` (Task 4), `system_message.Processor` + `GetHintThrottle` (Tasks 5-6), `ExperienceConfig` + `LoadExperienceConfig` (Task 8), plus the existing `character.Processor` and `party.Processor`.
- Produces: `monster.ProcessorOption`; `WithCharacterProcessor`, `WithPartyProcessor`, `WithRatesProcessor`, `WithInformationProcessor`, `WithFieldProcessor`, `WithSystemMessageProcessor`, `WithHintThrottle`, `WithExperienceConfig`; `(*ProcessorImpl).With(opts ...ProcessorOption) Processor`.

### Files

- `services/atlas-monster-death/atlas.com/monster/monster/processor.go` — add fields, bind them in `NewProcessor`, add the options and `With`, and route `CreateDrops`/`DistributeExperience` through the fields
- `services/atlas-monster-death/atlas.com/monster/monster/processor_di_test.go` — **new file**
- `services/atlas-pets/atlas.com/pets/pet/processor.go` — read-only reference: `ProcessorOption` at line 161, the `With*` functions at 163-208, `With` at 211-217

Module root: `services/atlas-monster-death/atlas.com/monster`. Package `monster`.

- [x] **Step 1: Write the failing test**

New file `monster/processor_di_test.go`, package `monster`. Imports the mocks: `charactermock "atlas-monster-death/character/mock"`, `partymock "atlas-monster-death/party/mock"`, `ratesmock "atlas-monster-death/rates/mock"`, `informationmock "atlas-monster-death/monster/information/mock"`, `mapmock "atlas-monster-death/map/mock"`, `systemmessagemock "atlas-monster-death/system_message/mock"`. Build the logger with `logrus.New()` and the context with `tenant.WithContext(context.Background(), ten)` where `ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)` — the same setup `atlas-monsters`' `processor_test.go:1904-1905` uses.

`TestWith_ReturnsCloneAndDoesNotMutateOriginal` — construct `base := NewProcessor(l, ctx).(*ProcessorImpl)`; capture `base.rp`; call `clone := base.With(WithRatesProcessor(&ratesmock.ProcessorMock{})).(*ProcessorImpl)`; assert `clone != base`, `clone.rp != base.rp`, and `base.rp` is the value captured before the call.

`TestWith_AppliesEveryOption` — apply all eight options with distinct mock instances and `WithExperienceConfig(ExperienceConfig{LevelInterval: 42})`, then assert each field on the clone is the injected value (`clone.cfg.LevelInterval == 42`).

`TestNewProcessor_BindsProductionDefaults` — assert every collaborator field on a bare `NewProcessor(l, ctx).(*ProcessorImpl)` is non-nil, and that `cfg` equals `LoadExperienceConfig()`.

- [x] **Step 2: Run test to verify it fails**

```
go test ./monster/ -run 'TestWith_|TestNewProcessor_' -v
```

Expected: FAIL to compile — `ProcessorOption` and the `With*` functions are undefined.

- [x] **Step 3: Write the implementation**

In `processor.go`, extend the struct and constructor:

```go
type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	cp   character.Processor
	pp   party.Processor
	rp   rates.Processor
	ip   information.Processor
	fp   _map.Processor
	smp  system_message.Processor
	ht   *system_message.Throttle
	cfg  ExperienceConfig
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		cp:  character.NewProcessor(l, ctx),
		pp:  party.NewProcessor(l, ctx),
		rp:  rates.NewProcessor(l, ctx),
		ip:  information.NewProcessor(l, ctx),
		fp:  _map.NewProcessor(l, ctx),
		smp: system_message.NewProcessor(l, ctx),
		ht:  system_message.GetHintThrottle(),
		cfg: LoadExperienceConfig(),
	}
}
```

Then one `ProcessorOption` setter per field, following `atlas-pets` exactly, and:

```go
func (p *ProcessorImpl) With(opts ...ProcessorOption) Processor {
	clone := *p
	cp := &clone
	for _, opt := range opts {
		opt(cp)
	}
	return cp
}
```

Finally, replace the inline constructions in the existing bodies with the fields: in `CreateDrops`, `rates.GetForCharacter(p.l)(p.ctx)(f.Channel(), killerId)` → `p.rp.GetForCharacter(f.Channel(), killerId)` and `party.NewProcessor(p.l, p.ctx).GetByMemberId(killerId)` → `p.pp.GetByMemberId(killerId)`; in `DistributeExperience` and `produceDistribution`, `character.NewProcessor(p.l, p.ctx)` → `p.cp`, `rates.GetForCharacter(...)` → `p.rp.GetForCharacter(...)`, `information.GetById(p.l)(p.ctx)(monsterId)` → `p.ip.GetById(monsterId)`, and `_map.CharacterIdsInFieldModelProvider(p.l)(p.ctx)(f)` → a `p.fp.CharacterIdsInField(f)` call. `drop.NewProcessor` and `quest.GetStartedQuestIds` stay as they are — the drop path is out of scope.

**Behaviour must be identical after this task.** The `DistributeExperience` rewrite is Task 12.

- [x] **Step 4: Run tests**

```
go build ./... && go test ./monster/... -v
```

Expected: PASS, including every pre-existing test in the package.

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/monster/processor.go services/atlas-monster-death/atlas.com/monster/monster/processor_di_test.go
git commit -m "refactor(atlas-monster-death): inject monster processor collaborators (D2)"
```

---

### Task 12: `atlas-monster-death` — rewrite `DistributeExperience` as resolve → plan → award

The behavioural change. `produceDistribution` and `DamageDistributionModel` are retired; `distributeCharacterExperience` is replaced by `computeAward` plus the award loop, which also removes the `totalPartyLevel byte` overflow (D6 — six level-200 members sum to 1200, which wraps to 176 in a `byte` and inflates every member's share ~6.8×).

Both `TODO parties` and `TODO account for healing` markers are deleted here, and `totalDamage` becomes the sum of the aggregated entries rather than `mi.Hp()` (FR-3.1). The rationale is **normalisation**, not heal absorption: `totalDamage` is only ever the denominator of `personalRatio` and `experiencePerDamage`, so `Σ entries` makes the participants' shares sum to exactly 1.0, while `mi.Hp()` over- or under-distributes whenever the monster was healed or entries decayed.

Not changed here: the `AWARD_EXPERIENCE` contract, including the fact that `awardExperienceCommandProvider` emits no `transactionId` and no `showEffect` and appends a `PARTY` distribution with `Amount: 0` on solo awards. PRD §2 forbids touching it.

### Interfaces

- Consumes: everything from Tasks 3-11.
- Produces: no new exported surface. `DamageEntryModel` and its `NewDamageEntryModel` constructor keep their current shape — the consumer at `kafka/consumer/monster/consumer.go:64-66` is unchanged.

### Files

- `services/atlas-monster-death/atlas.com/monster/monster/processor.go` — rewrite `DistributeExperience`, delete `produceDistribution` and `distributeCharacterExperience`
- `services/atlas-monster-death/atlas.com/monster/monster/model.go` — delete `DamageDistributionModel` and its four accessors; keep `DamageEntryModel`
- `services/atlas-monster-death/atlas.com/monster/monster/experience.go` — read-only; `planDistribution` and `computeAward` are called from here
- `services/atlas-monster-death/atlas.com/monster/kafka/consumer/monster/consumer.go` — read-only; it already calls `monster.NewProcessor(l, ctx).DistributeExperience(...)` and needs no change, because `NewProcessor` now binds the config itself

Module root: `services/atlas-monster-death/atlas.com/monster`.

The mock-driven tests for this behaviour are Task 13. This task's own gate is that the package still builds and every pre-existing test stays green.

- [x] **Step 1: Write the failing test**

Append to `monster/experience_test.go` the one piece of this task that is pure and therefore testable without mocks:

`TestAggregateDamageEntries` — table-driven over `aggregateDamageEntries([]DamageEntryModel) []DamageInput`, which is the FR-1.4 service-boundary defence.

| case | input (`NewDamageEntryModel` calls in order) | expected output |
|---|---|---|
| `already aggregated` | (1,100), (2,200) | `[{1,100},{2,200}]` |
| `accumulates, does not assign` | (1,100), (1,100), (1,50) | `[{1,250}]` |
| `sorts ascending by characterId` | (9,10), (2,20), (5,30) | `[{2,20},{5,30},{9,10}]` |
| `interleaved duplicates` | (2,10), (1,5), (2,10), (1,5) | `[{1,10},{2,20}]` |
| `empty` | none | empty slice, not nil-dereferencing |
| `zero damage preserved` | (1,0) | `[{1,0}]` |

The `accumulates, does not assign` row is the PRD acceptance bullet: today's `soloDistribution[de.characterId] = de.damage` (`processor.go:167`) assigns, so an un-aggregated entry list silently drops all but the last hit.

- [x] **Step 2: Run test to verify it fails**

```
go test ./monster/ -run TestAggregateDamageEntries -v
```

Expected: FAIL to compile — `aggregateDamageEntries` undefined.

- [x] **Step 3: Write the implementation**

Add to `experience.go`:

```go
// aggregateDamageEntries folds a KILLED event's damage entries into one entry
// per character, ascending by characterId. atlas-monsters already aggregates
// on both its write paths, so this is the service-boundary defence FR-1.4
// asks for: it accumulates rather than assigns, so an un-aggregated list from
// any future producer cannot silently drop all but the last hit.
func aggregateDamageEntries(des []DamageEntryModel) []DamageInput
```

Then rewrite `DistributeExperience` in `processor.go`:

```go
func (p *ProcessorImpl) DistributeExperience(f field.Model, monsterId uint32, damageEntries []DamageEntryModel) error {
	// 1. RESOLVE
	damages := aggregateDamageEntries(damageEntries)
	var totalDamage uint32
	for _, d := range damages {
		totalDamage += d.Damage
	}
	if totalDamage == 0 {
		p.l.Warnf("Monster [%d] died with no recorded damage. No experience distributed.", monsterId)
		return nil
	}
	// monster information and the field character list are independent; issue
	// them together.
	// ... sync.WaitGroup over two closures, capturing (mi, miErr) and (ids, idsErr)
	// ... return the first non-nil error
	// 2. resolve parties for in-field damagers, memoising by partyId
	// 3. PLAN
	plan := planDistribution(ExperienceInput{...}, p.cfg)
	// 4. AWARD, then hints
	return nil
}
```

Fill it in as follows.

*Resolve, monster and field.* Run `p.ip.GetById(monsterId)` and `p.fp.CharacterIdsInField(f)` concurrently under a `sync.WaitGroup`; on either error, log at error with `monsterId` and return the error. Build `inField map[uint32]bool` from the id slice.

*Resolve, parties.* Walk `damages` in order (already ascending `characterId`). Skip any character not in `inField` — an out-of-field damager is never party-resolved, which is both what FR-2.1 says and what closes the tag-and-walk-away leech vector (D12). For an in-field character:

- If `partyOf[characterId]` is already set from a previously fetched party, it is already accounted for; continue.
- Otherwise `pt, err := p.pp.GetByMemberId(characterId)`. On error, log at **warn** and treat the character as solo (FR-2.3 — a party-service outage must degrade to today's solo behaviour, never to zero EXP). On `pt.Id() == 0`, treat as solo.
- On success with a real party: record `parties[pt.Id()] = pt` and set `partyOf[m.Id()] = pt.Id()` for **every** member of `pt`. Because `GetByMemberId` returns the whole party, this collapses an N-member party to one lookup (NFR-1).

For each solo character, fetch the level with `p.cp.GetById(characterId)`; on error log at error and skip that character only (this matches today's behaviour at `processor.go:130-132`), never aborting the rest.

Build `[]PartyInput` from `parties`: for each, `Members` = `pt.Members()` filtered to `inField[m.Id()]`, carrying `m.Id()` and `m.Level()`. Co-location comes from the `atlas-maps` set and **never** from `MemberModel.Field()` or `MemberModel.Online()`, which are only eventually consistent (D11) — an offline character is not in the field set anyway, so FR-5.5 holds a fortiori.

*Plan.* `plan := planDistribution(ExperienceInput{MonsterExperience: mi.Experience(), MonsterLevel: mi.Level(), Damages: damages, Solos: solos, Parties: partyInputs}, p.cfg)`.

*Award.* For each `r` in `plan.Recipients` (already ascending `characterId`): `rate := p.rp.GetForCharacter(f.Channel(), r.CharacterId)` — exactly one rate lookup per recipient (FR-8.4); `personal, bonus, guarded := computeAward(r, rate.ExpRate(), p.cfg)`; if `guarded`, log at warn with `monsterId` and `r.CharacterId`; then `if err := p.cp.AwardExperience(f.Channel(), r.CharacterId, r.White, personal, bonus); err != nil` log at warn and **continue** — one recipient's failure must not abort the others (FR-9.2).

*Hints.* After every award, for each `e` in `plan.Exclusions`: `if !p.ht.Allow(tenant.MustFromContext(p.ctx).Id(), e.CharacterId) { continue }`; then `p.smp.ShowHint(uuid.New(), f.Channel(), e.CharacterId, levelGateHintText(mi.Name(), mi.Level()), 0, 0)`; log a publish error at warn and continue (FR-6.10). Hints run last so a hint failure cannot affect EXP at all.

Delete `produceDistribution` and `distributeCharacterExperience` entirely. In `model.go`, delete `DamageDistributionModel` and its `Solo`, `ExperiencePerDamage`, `PersonalRatio`, and `StandardDeviationRatio` accessors — `ExperiencePlan` carries all of it now, including the `TotalDamage` the PRD asked for. Keep `DamageEntryModel`, `NewDamageEntryModel`, and its two accessors.

- [x] **Step 4: Run tests**

```
go build ./... && go test ./... -v
```

Expected: PASS. Confirm with `grep -rn 'TODO parties\|TODO account for healing' services/atlas-monster-death/` that neither comment survives, and with `grep -n 'totalPartyLevel byte' services/atlas-monster-death/atlas.com/monster/monster/processor.go` that the `byte` signature is gone (both greps must print nothing).

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/monster/processor.go services/atlas-monster-death/atlas.com/monster/monster/model.go services/atlas-monster-death/atlas.com/monster/monster/experience.go services/atlas-monster-death/atlas.com/monster/monster/experience_test.go
git commit -m "feat(atlas-monster-death): distribute party experience via resolve-plan-award (FR-2, FR-3, FR-5, FR-6)"
```

---

### Task 13: `atlas-monster-death` — processor tests with mocks

The I/O-shaped acceptance criteria: lookup counts, tenant propagation, failure isolation, throttling. Everything numeric is already covered mock-free in Tasks 9-10 — do not re-test arithmetic here.

### Interfaces

- Consumes: `With(...)` and all mocks from Tasks 3-6 and 11; the rewritten `DistributeExperience` from Task 12.

### Files

- `services/atlas-monster-death/atlas.com/monster/monster/processor_experience_test.go` — **new file**
- `services/atlas-monster-death/atlas.com/monster/monster/processor_di_test.go` — read-only reference for the logger/tenant/context setup

Module root: `services/atlas-monster-death/atlas.com/monster`. Package `monster`.

- [x] **Step 1: Write the failing test**

New file `monster/processor_experience_test.go`, package `monster`. Shared setup per test: `l := logrus.New()` with `l.SetOutput(io.Discard)`; `ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)`; `ctx := tenant.WithContext(context.Background(), ten)`; `f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()`. Mocks are wired through `NewProcessor(l, ctx).(*ProcessorImpl).With(...)`. Counters are plain `int` variables closed over by the mock funcs — no mocking library.

Each test injects `WithHintThrottle(system_message.NewThrottle(time.Minute, 4096, func() time.Time { return clock }))` with an explicit fake clock, so throttle state never leaks between tests via the process singleton.

| test | setup | assertion |
|---|---|---|
| `TestDistributeExperience_OnePartyLookupPerParty` | 4 in-field damagers, all in party 9 (`GetByMemberIdFunc` returns a 4-member party) | `getByMemberIdCalls == 1` (FR-2.4, NFR-1) |
| `TestDistributeExperience_OneRateLookupPerRecipient` | 2-member party, one damager | `getForCharacterCalls == 2`, and the recorded `characterId` arguments are `[1, 2]` (FR-8.4) |
| `TestDistributeExperience_PartyLookupErrorFallsBackToSolo` | single damager, `GetByMemberIdFunc` returns an error | one `AwardExperience` call, with `party == 0`; no panic (FR-2.3) |
| `TestDistributeExperience_PartyRecipientsCarryNonZeroParty` | 2-member party, 1000 damage, both level 50, mob exp 1000 level 50 | both `AwardExperience` calls have `party > 0` and `party == uint32(0.10*float64(amount))` (FR-8.5) |
| `TestDistributeExperience_SoloRecipientCarriesZeroParty` | single partyless damager (`GetByMemberIdFunc` returns `party.Model{}`) | one `AwardExperience` call with `party == 0` and `amount == monsterExp * expRate` (PRD solo-path acceptance) |
| `TestDistributeExperience_ZeroDamageAwardsNothing` | damage entries `[{1,0}]` | zero `AwardExperience` calls, zero `ShowHint` calls, `err == nil` (FR-3.3) |
| `TestDistributeExperience_OutOfFieldDamagerReceivesNothing` | damagers 1 and 7; `CharacterIdsInFieldFunc` returns `[1]` only | exactly one `AwardExperience` call, for character 1; `getByMemberIdCalls == 1` — character 7 is never party-resolved (D12) |
| `TestDistributeExperience_ExcludedMemberGetsExactlyOneHint` | mob level 125 name "Zakum"; party of a level-120 contributor and a level-70 member | exactly one `ShowHint` call, `characterId == 70's id`, `width == 0`, `height == 0`, `hint == levelGateHintText("Zakum", 125)` (FR-6.7, FR-6.8) |
| `TestDistributeExperience_HintFailureDoesNotAbortAwards` | as above but with three excluded members and `ShowHintFunc` returning an error for the first | three `ShowHint` calls attempted, all recipients still awarded, `err == nil` (FR-6.10) |
| `TestDistributeExperience_AwardFailureDoesNotAbortOthers` | 3-member party, `AwardExperienceFunc` errors on the first `characterId` | three `AwardExperience` calls, `err == nil` (FR-9.2) |
| `TestDistributeExperience_HintIsThrottledAcrossKills` | same excluded member, `DistributeExperience` called twice with the clock advanced 30s between | exactly one `ShowHint` call total; advancing the clock 61s and calling a third time yields a second call (D10) |
| `TestDistributeExperience_GateDisabledEmitsNoHint` | the exclusion setup, plus `WithExperienceConfig` with `EnforceMobLevelRange: false` | zero `ShowHint` calls, every member awarded (FR-6.5 acceptance) |
| `TestDistributeExperience_InformationErrorReturnsError` | `GetByIdFunc` returns an error | `DistributeExperience` returns that error, zero `AwardExperience` calls |
| `TestDistributeExperience_FieldErrorReturnsError` | `CharacterIdsInFieldFunc` returns an error | `DistributeExperience` returns that error, zero `AwardExperience` calls |
| `TestDistributeExperience_AwardOrderIsAscendingCharacterId` | party members 3, 1, 2 returned in that order by the party mock | the recorded `AwardExperience` `characterId` sequence is exactly `[1, 2, 3]` (FR-9.1) |

Tenant/span header propagation (the remaining cross-cutting acceptance bullet) is covered structurally: `system_message.ProcessorImpl.ShowHint` uses the same `producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(...)` call as `character.ProcessorImpl.AwardExperience`, and the headers are attached by `libs/atlas-kafka`, not by this code. Assert that equivalence by reading both bodies rather than by writing a broker-dependent test.

- [x] **Step 2: Run test to verify it fails**

```
go test ./monster/ -run TestDistributeExperience -v
```

Expected: FAIL — the mocks compile but assertions fail until Task 12's rewrite is in place. If Task 12 already landed, these should be written and then confirmed one at a time.

- [x] **Step 3: Fix whatever the tests catch**

These tests are written against the Task 12 implementation. If any fails, fix `processor.go` — not the test — unless the test's own expectation contradicts a requirement quoted in this plan.

- [x] **Step 4: Run the full suite with the race detector**

```
go build ./... && go test ./... -race
```

Expected: PASS with no race reports (the resolve phase's concurrent monster/field fetch and the hint throttle's mutex are both exercised).

- [x] **Step 5: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/monster/processor_experience_test.go
git commit -m "test(atlas-monster-death): cover party EXP distribution I/O behaviour"
```

---

## Final gate (controller, not an implementer task)

- [x] Flagless `tools/verify.sh` exits 0. Only the flagless invocation counts — `--quick`/`--no-docker` skip the bake and `-race`.
- [x] `backend-guidelines-reviewer` passes on both `services/atlas-monster-death` and `services/atlas-monsters`.
- [x] `grep -rn 'TODO parties\|TODO account for healing' services/atlas-monster-death/` prints nothing.
- [x] No `*_testhelpers.go` file was introduced anywhere in the branch.
