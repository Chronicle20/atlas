# Pet Evolution Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let Dragon/Robo pets hatch from eggs automatically on summon and evolve/re-evolve into random adult forms via an NPC, mutating the pet and its inventory asset in place so the pet record and cash linkage survive.

**Architecture:** WZ evolution data is parsed by `atlas-data` and exposed on the pet REST resource. `atlas-pets` owns the random outcome roll and the in-place pet-row mutation, then cascades a new `CHANGE_TEMPLATE` command to `atlas-inventory` (which swaps the asset's `templateId` in place and emits the existing `UPDATED` event — never `DELETED`, so the pet is not destroyed). Egg hatching runs inside the `SPAWN` path (no saga). NPC-driven evolution is a `PetEvolution` saga `[destroy_item → award_mesos → evolve_pet]` with reverse-walk compensation that refunds the Rock + mesos if the pet became ineligible. `atlas-channel` needs no code change: a summoned pet's appearance refreshes via re-emitted `DESPAWNED`/`SPAWNED` events.

**Tech Stack:** Go 1.2x, GORM, Kafka (segmentio), `api2go/jsonapi`, `libs/atlas-model` functional providers, `libs/atlas-saga`, immutable Builder models, `message.Buffer`/`message.Emit`.

**Confirmed WZ data (verified against `tmp/<uuid>/GMS/83.1/Item.wz/Pet/`):**
- Dragon Egg `5000028`: `evol1=5000029`, `evolNo=1`, `evolProb1=100`, `evolReqItemID=0`, no `evolReqPetLvl` → **egg**.
- Baby Dragon `5000029`: `evol1..4=5000030,5000031,5000032,5000033`, `evolNo=4`, `evolProb=33,33,33,1`, `evolReqItemID=5380000`, `evolReqPetLvl=15`.
- Robo Egg `5000047`: `evol1=5000048`, `evolNo=1`, `evolProb1=1000`, `evolReqItemID=0`, no `evolReqPetLvl` → **egg**.
- Baby Robo `5000048`: `evol1..5=5000049..5000053`, `evolNo=5`, `evolProb=330,330,330,9,1`, `evolReqItemID=5380000`, `evolReqPetLvl=15`.

> ⚠️ Probability weights are **relative**, not percentages (robo uses a 1000-base). The roll MUST sum the weights and pick proportionally — never assume they total 100.

**Egg discriminator (data-driven, no hard-coded ids):**
- **Egg** = evolution data present AND `reqItemId == 0` AND `reqPetLevel == 0` AND `evolNo == 1`.
- **Evolvable (baby/adult)** = evolution data present AND `reqItemId != 0` (and usually `reqPetLevel > 0`, `evolNo > 1`).
- **Non-evolvable** = no evolution data (empty `evolutions`).

---

## File Structure

**atlas-data** (`services/atlas-data/atlas.com/data/pet/`)
- Modify `rest.go` — add `ReqPetLevel`, `ReqItemId`, `Evolutions []EvolutionRestModel` attributes + `EvolutionRestModel`.
- Modify `reader.go` — parse `evol*` nodes from `info/`.
- Modify `reader_test.go` / add `rest_test.go` coverage.

**atlas-pets** (`services/atlas-pets/atlas.com/pets/`)
- Modify `data/pet/model.go`, `data/pet/rest.go` — mirror the new evolution fields.
- Modify `pet/builder.go` — add `SetTemplateId`.
- Modify `pet/administrator.go` — add `updateOnEvolve`.
- Modify `kafka/message/pet/kafka.go` — `EVOLVE` command + `EVOLVED` event body.
- Modify `pet/producer.go` — `evolvedEventProvider`.
- Create `kafka/message/compartment/kafka.go` + `kafka/producer`-side provider — outbound `CHANGE_TEMPLATE` command.
- Create `inventory/command.go` (or extend `inventory/processor.go`) — emit `CHANGE_TEMPLATE`.
- Modify `pet/processor.go` — `Evolve`/`EvolveAndEmit`, egg-hatch branch in `Spawn`, weighted-roll injection.
- Modify `kafka/consumer/pet/consumer.go` — `handleEvolveCommand`.

**atlas-inventory** (`services/atlas-inventory/atlas.com/inventory/`)
- Modify `asset/administrator.go` — `updateTemplate`.
- Modify `asset/processor.go` — `ChangeTemplate`.
- Modify `kafka/message/compartment/kafka.go` — `CHANGE_TEMPLATE` command + body.
- Modify `kafka/consumer/compartment/consumer.go` — `handleChangeTemplateCommand`.
- Modify `compartment/processor.go` — `ChangeTemplate`/`ChangeTemplateAndEmit` (resolve asset by `petId`).

**libs/atlas-saga**
- Modify `model.go` — `EvolvePet` action, `PetEvolution` type.
- Modify `payloads.go` — `EvolvePetPayload`.
- Modify `unmarshal.go` — `EvolvePet` case.

**atlas-saga-orchestrator** (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`)
- Modify `pet/processor.go` — `EvolveAndEmit`/`Evolve` + command provider.
- Modify `kafka/message/pet/kafka.go` + producer — outbound `EVOLVE`.
- Modify `saga/handler.go` — `handleEvolvePet` + `GetHandler` case.
- Modify `saga/compensator.go` — `PetEvolution` reverse-walk + new `DestroyAsset`/`AwardMesos` inverse cases.

**atlas-npc-conversations** (`services/atlas-npc-conversations/atlas.com/npc/`)
- Modify `pet/model.go`, `pet/rest.go` — add `templateId`, `level`.
- Create `petdata/{model,rest,requests,processor}.go` — thin atlas-data evolution client.
- Modify `conversation/operation_executor.go` — `local:enumerate_evolvable_pets`, `evolve_pet`, `PetEvolution` saga-type override.
- Create `deploy/seed/.../npc-conversations/npc/npc-1032102.json` — reference Garnox conversation.

**atlas-channel** — verify-only (no code change).

---

# Phase 1 — atlas-data: parse & expose evolution data (FR-1.1–1.3)

### Task 1: Add evolution attributes to the pet REST model

**Files:**
- Modify: `services/atlas-data/atlas.com/data/pet/rest.go:9-16`
- Modify: `services/atlas-data/atlas.com/data/pet/rest.go` (append `EvolutionRestModel`)

- [ ] **Step 1: Add the fields and nested struct**

In `rest.go`, change the `RestModel` struct (lines 9-16) to:

```go
type RestModel struct {
	Id          uint32               `json:"-"`
	Name        string               `json:"name"`
	Hungry      uint32               `json:"hungry"`
	Cash        bool                 `json:"cash"`
	Life        uint32               `json:"life"`
	ReqPetLevel uint32               `json:"reqPetLevel"`
	ReqItemId   uint32               `json:"reqItemId"`
	Evolutions  []EvolutionRestModel `json:"evolutions"`
	Skills      []SkillRestModel     `json:"-"`
}
```

Append at the end of `rest.go`:

```go
type EvolutionRestModel struct {
	TemplateId  uint32 `json:"templateId"`
	Probability uint32 `json:"probability"`
}
```

`Evolutions` is a plain attribute slice (no JSON:API relationship), so `GetReferences`/`SetReferencedStructs` stay unchanged — `api2go/jsonapi` marshals a json-tagged nested slice as a normal attribute array. Initialize `Evolutions` to a non-nil empty slice in the reader so non-evolvable pets serialize `"evolutions": []`.

- [ ] **Step 2: Build to verify it compiles**

Run: `cd services/atlas-data/atlas.com/data && go build ./...`
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-data/atlas.com/data/pet/rest.go
git commit -m "feat(atlas-data): add evolution attributes to pet REST model"
```

### Task 2: Parse `evol*` nodes in the WZ reader

**Files:**
- Modify: `services/atlas-data/atlas.com/data/pet/reader.go:55-57`
- Test: `services/atlas-data/atlas.com/data/pet/reader_test.go`

- [ ] **Step 1: Write the failing test**

Add to `reader_test.go` a table test feeding the reader an `xml.Node` for an evolvable pet. Use the existing test's node-construction helper/pattern (mirror the `info`/`interact` fixture already in that file). Assert:

```go
func TestReadEvolutionData(t *testing.T) {
	// Build an info node with evol fields (mirror existing fixture construction).
	// info children: hungry=2, cash=1, life=90, evol=1, evolNo=4,
	//   evol1=5000030, evol2=5000031, evol3=5000032, evol4=5000033,
	//   evolProb1=33, evolProb2=33, evolProb3=33, evolProb4=1,
	//   evolReqPetLvl=15, evolReqItemID=5380000
	// (plus an empty "interact" node so ChildByName("interact") succeeds)
	rm, err := Read(testLogger())(testCtx(t))(model.FixedProvider(node))()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rm.ReqPetLevel != 15 {
		t.Errorf("ReqPetLevel = %d, want 15", rm.ReqPetLevel)
	}
	if rm.ReqItemId != 5380000 {
		t.Errorf("ReqItemId = %d, want 5380000", rm.ReqItemId)
	}
	if len(rm.Evolutions) != 4 {
		t.Fatalf("Evolutions len = %d, want 4", len(rm.Evolutions))
	}
	if rm.Evolutions[0].TemplateId != 5000030 || rm.Evolutions[0].Probability != 33 {
		t.Errorf("Evolutions[0] = %+v, want {5000030 33}", rm.Evolutions[0])
	}
	if rm.Evolutions[3].TemplateId != 5000033 || rm.Evolutions[3].Probability != 1 {
		t.Errorf("Evolutions[3] = %+v, want {5000033 1}", rm.Evolutions[3])
	}
}

func TestReadNonEvolvablePet(t *testing.T) {
	// info node WITHOUT any evol fields, plus empty interact node.
	rm, err := Read(testLogger())(testCtx(t))(model.FixedProvider(node))()
	if err != nil {
		t.Fatalf("non-evolvable pet must read without error: %v", err)
	}
	if rm.ReqPetLevel != 0 || rm.ReqItemId != 0 || len(rm.Evolutions) != 0 {
		t.Errorf("non-evolvable pet must have zero evolution data, got level=%d item=%d evols=%d",
			rm.ReqPetLevel, rm.ReqItemId, len(rm.Evolutions))
	}
}
```

> If `reader_test.go` has no existing logger/ctx/node helpers, mirror the construction already used by the file's current test(s) — do not add `*_testhelpers.go`.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test ./pet/ -run TestReadEvolution -v`
Expected: FAIL (`Evolutions len = 0, want 4`).

- [ ] **Step 3: Implement the parsing**

In `reader.go`, replace lines 55-57 (the `hungry`/`cash`/`life` block) with:

```go
				m.Hungry = uint32(i.GetIntegerWithDefault("hungry", 0))
				m.Cash = i.GetBool("cash", true)
				m.Life = uint32(i.GetIntegerWithDefault("life", 0))

				m.ReqPetLevel = uint32(i.GetIntegerWithDefault("evolReqPetLvl", 0))
				m.ReqItemId = uint32(i.GetIntegerWithDefault("evolReqItemID", 0))
				m.Evolutions = make([]EvolutionRestModel, 0)
				evolNo := int(i.GetIntegerWithDefault("evolNo", 0))
				for n := 1; n <= evolNo; n++ {
					tid := uint32(i.GetIntegerWithDefault(fmt.Sprintf("evol%d", n), 0))
					if tid == 0 {
						continue // tolerate gaps
					}
					m.Evolutions = append(m.Evolutions, EvolutionRestModel{
						TemplateId:  tid,
						Probability: uint32(i.GetIntegerWithDefault(fmt.Sprintf("evolProb%d", n), 0)),
					})
				}
```

`fmt` and `strconv` are already imported in `reader.go`.

> Also set `m.Evolutions = make([]EvolutionRestModel, 0)` is done above unconditionally, so a non-evolvable pet (evolNo absent → 0) yields an empty slice and the loop body never runs.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test ./pet/ -run TestRead -v`
Expected: PASS for both `TestReadEvolutionData` and `TestReadNonEvolvablePet`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/pet/reader.go services/atlas-data/atlas.com/data/pet/reader_test.go
git commit -m "feat(atlas-data): parse evol* WZ nodes for pet evolution"
```

---

# Phase 2 — atlas-pets: read evolution data from atlas-data

### Task 3: Extend the atlas-pets data/pet client

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/data/pet/model.go`
- Modify: `services/atlas-pets/atlas.com/pets/data/pet/rest.go`

- [ ] **Step 1: Add evolution to the domain model**

In `data/pet/model.go`, add to `Model` (after `life`) and provide getters:

```go
type Model struct {
	id          uint32
	hunger      uint32
	cash        bool
	life        uint32
	reqPetLevel uint32
	reqItemId   uint32
	evolutions  []EvolutionModel
	skills      []SkillModel
}

func (m Model) ReqPetLevel() uint32        { return m.reqPetLevel }
func (m Model) ReqItemId() uint32          { return m.reqItemId }
func (m Model) Evolutions() []EvolutionModel { return m.evolutions }

// IsEgg reports an auto-hatch egg: evolution data with no requirements and a
// single deterministic target.
func (m Model) IsEgg() bool {
	return len(m.evolutions) == 1 && m.reqItemId == 0 && m.reqPetLevel == 0
}

// IsEvolvable reports an NPC-evolvable baby/adult: evolution data gated by a
// required item.
func (m Model) IsEvolvable() bool {
	return len(m.evolutions) > 0 && m.reqItemId != 0
}

type EvolutionModel struct {
	templateId  uint32
	probability uint32
}

func (e EvolutionModel) TemplateId() uint32  { return e.templateId }
func (e EvolutionModel) Probability() uint32 { return e.probability }
```

Add builder setters mirroring the existing ones:

```go
func (b *ModelBuilder) SetReqPetLevel(v uint32) *ModelBuilder { b.reqPetLevel = v; return b }
func (b *ModelBuilder) SetReqItemId(v uint32) *ModelBuilder   { b.reqItemId = v; return b }
func (b *ModelBuilder) SetEvolutions(v []EvolutionModel) *ModelBuilder { b.evolutions = v; return b }
```

Add the matching fields to `ModelBuilder` and to its `Build()` return, mirroring the existing `id/hunger/cash/life/skills` wiring.

- [ ] **Step 2: Add evolution to the REST model + Extract**

In `data/pet/rest.go`, add to `RestModel` (after `Life`):

```go
	ReqPetLevel uint32               `json:"reqPetLevel"`
	ReqItemId   uint32               `json:"reqItemId"`
	Evolutions  []EvolutionRestModel `json:"evolutions"`
```

Append:

```go
type EvolutionRestModel struct {
	TemplateId  uint32 `json:"templateId"`
	Probability uint32 `json:"probability"`
}
```

Extend `Extract` (lines 115-127) to populate the new fields:

```go
func Extract(rm RestModel) (Model, error) {
	sms, err := model.SliceMap(ExtractSkill)(model.FixedProvider(rm.Skills))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}
	evos := make([]EvolutionModel, 0, len(rm.Evolutions))
	for _, e := range rm.Evolutions {
		evos = append(evos, EvolutionModel{templateId: e.TemplateId, probability: e.Probability})
	}
	return Model{
		id:          rm.Id,
		hunger:      rm.Hungry,
		cash:        rm.Cash,
		life:        rm.Life,
		reqPetLevel: rm.ReqPetLevel,
		reqItemId:   rm.ReqItemId,
		evolutions:  evos,
		skills:      sms,
	}, nil
}
```

(`Transform` may stay as-is — atlas-pets only consumes this client.)

- [ ] **Step 2b: Update the data/pet mock if the interface grew**

If `data/pet/processor.go`'s `Processor` interface is unchanged (it returns `Model`), the mock in `data/pet/mock/` needs no change. Verify with build below; update the mock only if compilation requires it.

- [ ] **Step 3: Build**

Run: `cd services/atlas-pets/atlas.com/pets && go build ./...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/data/pet/
git commit -m "feat(atlas-pets): read pet evolution data from atlas-data"
```

---

# Phase 3 — atlas-inventory: in-place template swap (FR-4.1–4.4)

### Task 4: Asset administrator `updateTemplate`

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/administrator.go:54-60`

- [ ] **Step 1: Add the update function**

After `updateSlot` (line 56), add:

```go
func updateTemplate(db *gorm.DB, id uint32, templateId uint32) error {
	return db.Model(&Entity{Id: id}).Select("TemplateId").Updates(&Entity{TemplateId: templateId}).Error
}
```

This mirrors `updateSlot`/`updateQuantity` and updates **only** `TemplateId` (slot, compartment, `cashId`, `petId`, expiration, quantity preserved).

- [ ] **Step 2: Build**

Run: `cd services/atlas-inventory/atlas.com/inventory && go build ./asset/...`
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/asset/administrator.go
git commit -m "feat(atlas-inventory): add updateTemplate column updater"
```

### Task 5: Asset processor `ChangeTemplate`

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/processor.go` (after `UpdateEquipmentStats`, ~line 266)
- Test: `services/atlas-inventory/atlas.com/inventory/asset/processor_test.go` (create or extend)

- [ ] **Step 1: Write the failing test**

Add a test that creates a pet asset, calls `ChangeTemplate`, and asserts the row's `TemplateId` changed while slot/`petId`/`cashId`/expiration are preserved and an `UPDATED` event is buffered. Mirror the existing asset processor test setup (in-memory sqlite + Builder). Skeleton:

```go
func TestChangeTemplatePreservesIdentity(t *testing.T) {
	db := testDB(t) // existing helper / pattern in this package's tests
	p := NewProcessor(testLogger(), testCtx(t), db)
	// create a pet asset via the package Builder + create()
	created, _ := p.Create(...) // use the existing create path used by sibling tests
	var buf *message.Buffer
	err := message.Emit(testProducer())(func(mb *message.Buffer) error {
		buf = mb
		return p.ChangeTemplate(mb)(uuid.New(), created.OwnerId(), created.Id(), 5000029)
	})
	if err != nil {
		t.Fatalf("ChangeTemplate: %v", err)
	}
	got, _ := p.GetById(created.Id())
	if got.TemplateId() != 5000029 {
		t.Errorf("TemplateId = %d, want 5000029", got.TemplateId())
	}
	if got.Slot() != created.Slot() || got.PetId() != created.PetId() || got.CashId() != created.CashId() {
		t.Errorf("identity not preserved: %+v vs %+v", got, created)
	}
}
```

> Use the exact create/test harness the existing `asset` tests use. If none exists, follow the project Builder pattern (no `*_testhelpers.go`).

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./asset/ -run TestChangeTemplate -v`
Expected: FAIL (`ChangeTemplate` undefined).

- [ ] **Step 3: Implement `ChangeTemplate`**

In `asset/processor.go`, after `UpdateEquipmentStats` (line 266):

```go
func (p *Processor) ChangeTemplateAndEmit(transactionId uuid.UUID, characterId uint32, assetId uint32, newTemplateId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		return p.ChangeTemplate(mb)(transactionId, characterId, assetId, newTemplateId)
	})
}

func (p *Processor) ChangeTemplate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, assetId uint32, newTemplateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, assetId uint32, newTemplateId uint32) error {
		a, err := p.GetById(assetId)
		if err != nil {
			return err
		}
		if !a.IsPet() {
			return errors.New("change template only supported for pet assets")
		}
		updated := Clone(a).SetTemplateId(newTemplateId).Build()
		if err = updateTemplate(p.db.WithContext(p.ctx), assetId, newTemplateId); err != nil {
			return err
		}
		return mb.Put(asset.EnvEventTopicStatus, UpdatedEventStatusProvider(transactionId, characterId, updated))
	}
}
```

Add `"errors"` to the imports if not already present.

- [ ] **Step 4: Run to verify it passes**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./asset/ -run TestChangeTemplate -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/asset/
git commit -m "feat(atlas-inventory): asset ChangeTemplate emits UPDATED in place"
```

### Task 6: `CHANGE_TEMPLATE` compartment command + consumer + processor

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go:13-32, 141`
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:84, 333`
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` (interface + impl)

- [ ] **Step 1: Add the command constant + body**

In `kafka/message/compartment/kafka.go`, add to the command const block (after line 31):

```go
	CommandChangeTemplate    = "CHANGE_TEMPLATE"
```

Add the body struct (near `ModifyEquipmentCommandBody`, after line 166):

```go
// ChangeTemplateCommandBody changes an existing pet asset's templateId in place.
// The asset is resolved by (CharacterId, PetId); the cash reference, slot, and
// expiration are preserved.
type ChangeTemplateCommandBody struct {
	PetId         uint32 `json:"petId"`
	NewTemplateId uint32 `json:"newTemplateId"`
}
```

(`CharacterId` and `TransactionId` ride the `Command[E]` envelope.)

- [ ] **Step 2: Register the consumer handler**

In `kafka/consumer/compartment/consumer.go`, register alongside the others (mirror line 84):

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeTemplateCommand(db)))); err != nil {
				return err
			}
```

Add the handler (mirror `handleModifyEquipmentCommand`, line 333):

```go
func handleChangeTemplateCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.ChangeTemplateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.ChangeTemplateCommandBody]) {
		if c.Type != compartment2.CommandChangeTemplate {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).ChangeTemplateAndEmit(c.TransactionId, c.CharacterId, c.Body.PetId, c.Body.NewTemplateId)
	}
}
```

- [ ] **Step 3: Add the compartment processor method**

In `compartment/processor.go`, add to the `Processor` interface (near the other command methods) and implement (mirror `ModifyEquipment`, line 1752). The method resolves the cash-compartment asset whose `petId` matches, then delegates to the asset processor:

```go
func (p *Processor) ChangeTemplateAndEmit(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		return p.ChangeTemplate(mb)(transactionId, characterId, petId, newTemplateId)
	})
}

func (p *Processor) ChangeTemplate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
		p.l.Debugf("Character [%d] changing template of pet [%d] asset to [%d].", characterId, petId, newTemplateId)
		invLock := LockRegistry().Get(characterId, inventory.TypeValueCash)
		invLock.Lock()
		defer invLock.Unlock()

		return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			cp := p.WithTransaction(tx).WithAssetProcessor(asset.NewProcessor(p.l, p.ctx, tx))
			c, err := cp.GetByCharacterAndType(characterId)(inventory.TypeValueCash)
			if err != nil {
				return err
			}
			c, err = cp.DecorateAsset(c)
			if err != nil {
				return err
			}
			for _, a := range c.Assets() {
				if a.IsPet() && a.PetId() == petId {
					return cp.assetProcessor.ChangeTemplate(mb)(transactionId, characterId, a.Id(), newTemplateId)
				}
			}
			return fmt.Errorf("pet [%d] asset not found in cash compartment for character [%d]", petId, characterId)
		})
	}
}
```

> Confirm during implementation that `DecorateAsset` populates `Assets()` for the cash compartment (it is the same decorator used by other read paths). If a more direct asset-by-petId query is preferred, use the asset processor's compartment-scoped provider — but the in-memory filter above is sufficient and matches existing patterns.

- [ ] **Step 4: Build + vet**

Run: `cd services/atlas-inventory/atlas.com/inventory && go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/kafka/ services/atlas-inventory/atlas.com/inventory/compartment/processor.go
git commit -m "feat(atlas-inventory): CHANGE_TEMPLATE command swaps pet asset template in place"
```

---

# Phase 4 — atlas-pets: EVOLVE, egg hatch, inventory cascade (FR-2, FR-3)

### Task 7: `SetTemplateId` on the pet builder

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/builder.go`

- [ ] **Step 1: Add the setter**

After `SetLevel` (line 55), add:

```go
func (b *ModelBuilder) SetTemplateId(templateId uint32) *ModelBuilder {
	b.templateId = templateId
	return b
}
```

- [ ] **Step 2: Build**

Run: `cd services/atlas-pets/atlas.com/pets && go build ./pet/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/pet/builder.go
git commit -m "feat(atlas-pets): add SetTemplateId to pet builder"
```

### Task 8: `EVOLVE` command + `EVOLVED` event definitions

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/kafka/message/pet/kafka.go:10-19, 73-89`

- [ ] **Step 1: Add command + event constants and bodies**

Add to the command block (after line 18):

```go
	CommandPetEvolve         = "EVOLVE"
```

Add the (empty) command body near the others (after `SetExcludeCommandBody`, line 55):

```go
type EvolveCommandBody struct {
}
```

Add to the status-event block (after line 84):

```go
	StatusEventTypeEvolved          = "EVOLVED"
```

Add the event body (after `ExcludeChangedStatusEventBody`, line 162):

```go
type EvolvedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	OldTemplateId uint32    `json:"oldTemplateId"`
	NewTemplateId uint32    `json:"newTemplateId"`
	TransactionId uuid.UUID `json:"transactionId"`
}
```

- [ ] **Step 2: Build**

Run: `cd services/atlas-pets/atlas.com/pets && go build ./kafka/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/kafka/message/pet/kafka.go
git commit -m "feat(atlas-pets): define EVOLVE command and EVOLVED status event"
```

### Task 9: `evolvedEventProvider`

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/producer.go`

- [ ] **Step 1: Add the provider**

After `excludeChangedEventProvider` (line 173):

```go
func evolvedEventProvider(m Model, oldTemplateId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.EvolvedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeEvolved,
		Body: pet.EvolvedStatusEventBody{
			Slot:          m.Slot(),
			OldTemplateId: oldTemplateId,
			NewTemplateId: m.TemplateId(),
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 2: Build**

Run: `cd services/atlas-pets/atlas.com/pets && go build ./pet/...`
Expected: clean (unused function is fine until Task 12 wires it; if the build flags it, this step lands together with Task 12 — keep them in one commit if needed).

- [ ] **Step 3: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/pet/producer.go
git commit -m "feat(atlas-pets): add evolvedEventProvider"
```

### Task 10: `updateOnEvolve` administrator

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/administrator.go`

- [ ] **Step 1: Add the updater**

After `updateLevel` (line 89), add:

```go
func updateOnEvolve(db *gorm.DB) func(petId uint32, templateId uint32, expiration time.Time) error {
	return func(petId uint32, templateId uint32, expiration time.Time) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Updates(map[string]interface{}{
				"template_id": templateId,
				"expiration":  expiration,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("no entity found to evolve")
		}
		return nil
	}
}
```

Add `"time"` to the administrator imports.

> A `map[string]interface{}` update is used (not the column-`Select` form) so a zero-value field is never silently skipped; both columns always written.

- [ ] **Step 2: Build**

Run: `cd services/atlas-pets/atlas.com/pets && go build ./pet/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/pet/administrator.go
git commit -m "feat(atlas-pets): add updateOnEvolve administrator"
```

### Task 11: Outbound `CHANGE_TEMPLATE` cascade producer

**Files:**
- Create: `services/atlas-pets/atlas.com/pets/kafka/message/compartment/kafka.go`
- Create: `services/atlas-pets/atlas.com/pets/inventory/command.go`

- [ ] **Step 1: Define the outbound command contract**

Create `kafka/message/compartment/kafka.go` (mirrors atlas-inventory's command schema — each service owns its copy):

```go
package compartment

import "github.com/google/uuid"

const (
	EnvCommandTopic       = "COMMAND_TOPIC_COMPARTMENT"
	CommandChangeTemplate = "CHANGE_TEMPLATE"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	InventoryType byte      `json:"inventoryType"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type ChangeTemplateCommandBody struct {
	PetId         uint32 `json:"petId"`
	NewTemplateId uint32 `json:"newTemplateId"`
}
```

> `InventoryType` is part of the shared envelope; the cash type (value `5`) is set by the emitter. Confirm `inventory.TypeValueCash` byte value via `libs/atlas-constants/inventory` and pass it through.

- [ ] **Step 2: Add the emitter**

Create `inventory/command.go`:

```go
package inventory

import (
	compartmentmsg "atlas-pets/kafka/message/compartment"
	"atlas-pets/kafka/message"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func changeTemplateCommandProvider(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartmentmsg.Command[compartmentmsg.ChangeTemplateCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueCash),
		Type:          compartmentmsg.CommandChangeTemplate,
		Body: compartmentmsg.ChangeTemplateCommandBody{
			PetId:         petId,
			NewTemplateId: newTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ChangeTemplate buffers a CHANGE_TEMPLATE command to atlas-inventory.
func (p *ProcessorImpl) ChangeTemplate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
		return mb.Put(compartmentmsg.EnvCommandTopic, changeTemplateCommandProvider(transactionId, characterId, petId, newTemplateId))
	}
}
```

Add `ChangeTemplate(mb *message.Buffer) func(...) error` to the `inventory.Processor` interface and the inventory mock (`inventory/` mock if one exists). Verify `inventory.ProcessorImpl` has access to a producer; if not, the pet processor can buffer the command directly instead (see Task 12 note).

> **Simpler alternative:** if wiring a producer into `inventory.ProcessorImpl` is awkward, skip `inventory/command.go` and put `changeTemplateCommandProvider` + the `mb.Put(...)` directly in `pet/producer.go` / `pet/processor.go`. The pet processor already buffers events via `mb.Put(pet.EnvStatusEventTopic, ...)`; emitting one more topic from the same buffer is consistent. Choose whichever keeps the producer dependency clean — the cascade just needs the command on `COMMAND_TOPIC_COMPARTMENT` within the same `message.Emit`.

- [ ] **Step 3: Build**

Run: `cd services/atlas-pets/atlas.com/pets && go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/kafka/message/compartment/ services/atlas-pets/atlas.com/pets/inventory/
git commit -m "feat(atlas-pets): emit CHANGE_TEMPLATE cascade to atlas-inventory"
```

### Task 12: `Evolve` / `EvolveAndEmit` processor with injectable weighted roll

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/processor.go` (interface lines 35-71, struct 73-85, constructor 87-102, options, new methods near `AwardCloseness`)
- Test: `services/atlas-pets/atlas.com/pets/pet/processor_test.go`

- [ ] **Step 1: Add the injectable roll to the processor**

Add a field to `ProcessorImpl` (after `Despawner`, line 84):

```go
	// rollEvolution picks an index into the weighted candidate list. Injectable
	// for deterministic tests; defaults to a weighted-random pick.
	rollEvolution func(weights []uint32) int
```

In `NewProcessor` (after line 100), set the default:

```go
	p.rollEvolution = weightedRoll
```

Add the default implementation and the option near the other options (after `WithSkillProcessor`, line 134):

```go
// weightedRoll picks an index proportional to the given relative weights.
// Weights are NOT assumed to sum to 100 (WZ uses arbitrary bases, e.g. 1000).
func weightedRoll(weights []uint32) int {
	var total uint32
	for _, w := range weights {
		total += w
	}
	if total == 0 {
		return 0
	}
	r := uint32(rand.Intn(int(total)))
	var acc uint32
	for i, w := range weights {
		acc += w
		if r < acc {
			return i
		}
	}
	return len(weights) - 1
}

func WithRollEvolution(f func(weights []uint32) int) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.rollEvolution = f
	}
}
```

`math/rand` is already imported.

- [ ] **Step 2: Write the failing test**

In `processor_test.go`, add a test using an injected roll. Mirror the existing processor tests' DB + producer harness (Builder pattern). Assert: in-place identity preservation, templateId becomes the rolled candidate, expiration reset, `EVOLVED` + `CHANGE_TEMPLATE` emitted; and a level-gate rejection test.

```go
func TestEvolveRollsAndPreservesIdentity(t *testing.T) {
	db := testDB(t)
	// stub data processor returning evolvable data (reqItemId=5380000, reqPetLevel=15,
	// evolutions = [{5000030,33},{5000031,33},{5000032,33},{5000033,1}])
	dp := &dataPetMock{model: evolvableDragonData()}
	p := NewProcessor(testLogger(), testCtx(t), db).
		With(WithDataProcessor(dp), WithRollEvolution(func(_ []uint32) int { return 2 })) // pick 5000032
	// create a baby dragon pet at level 20, summoned slot 0
	pe := createPet(t, p, 5000029, 20, 0)
	oldExpiration := pe.Expiration()

	err := p.EvolveAndEmit(uuid.New(), pe.Id())
	if err != nil {
		t.Fatalf("EvolveAndEmit: %v", err)
	}
	got, _ := p.GetById(pe.Id())
	if got.TemplateId() != 5000032 {
		t.Errorf("TemplateId = %d, want 5000032", got.TemplateId())
	}
	if got.Id() != pe.Id() || got.CashId() != pe.CashId() || got.Level() != pe.Level() ||
		got.Closeness() != pe.Closeness() || got.Name() != pe.Name() || got.Slot() != pe.Slot() {
		t.Errorf("identity/stats not preserved: %+v vs %+v", got, pe)
	}
	if !got.Expiration().After(oldExpiration) {
		t.Errorf("expiration not reset: got %v, old %v", got.Expiration(), oldExpiration)
	}
}

func TestEvolveRejectsUnderLevel(t *testing.T) {
	db := testDB(t)
	dp := &dataPetMock{model: evolvableDragonData()} // reqPetLevel=15
	p := NewProcessor(testLogger(), testCtx(t), db).With(WithDataProcessor(dp))
	pe := createPet(t, p, 5000029, 10, -1) // level 10 < 15
	err := p.EvolveAndEmit(uuid.New(), pe.Id())
	if err == nil {
		t.Fatal("expected error for under-level pet")
	}
	got, _ := p.GetById(pe.Id())
	if got.TemplateId() != 5000029 {
		t.Errorf("pet must not change: got %d", got.TemplateId())
	}
}
```

> Use the package's existing test DB/producer helpers and `data/pet/mock`. Define `evolvableDragonData()`/`createPet` inline in the test file via the Builders.

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run TestEvolve -v`
Expected: FAIL (`EvolveAndEmit` undefined).

- [ ] **Step 4: Implement `Evolve` / `EvolveAndEmit`**

Add to the `Processor` interface (after the `AwardLevel` pair, line 68):

```go
	EvolveAndEmit(transactionId uuid.UUID, petId uint32) error
	Evolve(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error
```

Add the implementation near `AwardCloseness` (after line 714):

```go
func (p *ProcessorImpl) EvolveAndEmit(transactionId uuid.UUID, petId uint32) error {
	return message.Emit(p.kp)(func(mb *message.Buffer) error {
		return p.Evolve(mb)(transactionId, petId)
	})
}

func (p *ProcessorImpl) Evolve(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error {
	return func(transactionId uuid.UUID, petId uint32) error {
		p.l.Debugf("Evolving pet [%d].", petId)
		var oldTemplateId uint32
		var newTemplateId uint32
		var ownerId uint32
		var wasSummoned bool
		var summonedSlot int8

		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			pe, err := p.With(WithTransaction(tx)).GetById(petId)
			if err != nil {
				return err
			}
			oldTemplateId = pe.TemplateId()
			ownerId = pe.OwnerId()
			wasSummoned = pe.Slot() >= 0
			summonedSlot = pe.Slot()

			d, err := p.dp.GetById(pe.TemplateId())
			if err != nil {
				return err
			}
			if !d.IsEvolvable() {
				return fmt.Errorf("pet template [%d] is not evolvable", pe.TemplateId())
			}
			if uint32(pe.Level()) < d.ReqPetLevel() {
				return fmt.Errorf("pet [%d] level [%d] below required [%d]", petId, pe.Level(), d.ReqPetLevel())
			}

			evos := d.Evolutions()
			weights := make([]uint32, len(evos))
			for i, e := range evos {
				weights[i] = e.Probability()
			}
			idx := p.rollEvolution(weights)
			newTemplateId = evos[idx].TemplateId()

			updated, err := Clone(pe).
				SetTemplateId(newTemplateId).
				SetExpiration(time.Now().Add(2160 * time.Hour)).
				Build()
			if err != nil {
				return err
			}
			if err = updateOnEvolve(tx)(petId, newTemplateId, updated.Expiration()); err != nil {
				return err
			}

			// Cascade the in-place inventory asset swap (keyed by petId).
			if err = p.ip.ChangeTemplate(mb)(transactionId, ownerId, petId, newTemplateId); err != nil {
				return err
			}
			return mb.Put(pet.EnvStatusEventTopic, evolvedEventProvider(updated, oldTemplateId, transactionId))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to evolve pet [%d].", petId)
			return txErr
		}

		// Refresh appearance for a summoned pet via despawn+respawn (FR-3.4).
		if wasSummoned {
			if err := p.Despawn(mb)(petId)(ownerId)(pet.DespawnReasonNormal); err != nil {
				p.l.WithError(err).Warnf("Unable to despawn evolved pet [%d] for appearance refresh.", petId)
			} else if err := p.Spawn(mb)(petId)(ownerId)(summonedSlot == 0); err != nil {
				p.l.WithError(err).Warnf("Unable to respawn evolved pet [%d] for appearance refresh.", petId)
			}
		}
		p.l.Infof("Evolved pet [%d]: [%d] -> [%d].", petId, oldTemplateId, newTemplateId)
		return nil
	}
}
```

Add the `ip inventory.Processor` field to `ProcessorImpl` and wire it in `NewProcessor` (mirror the existing `cp`/`dp` wiring): `ip: inventory.NewProcessor(l, ctx)`. Add `"time"` to imports. Import `"atlas-pets/inventory"` (alias to avoid clashing with the constants `inventory` package — e.g. `inv "atlas-pets/inventory"` and field type `inv.Processor`).

> **Re-spawn note:** the `Despawn`/`Spawn` refresh re-runs slot assignment. Because the slot is unchanged (same pet, same lead status), this re-emits `DESPAWNED` then `SPAWNED` with the new `templateId`. If re-running `Spawn`'s lead/slot logic proves fragile here, the simpler equivalent is to emit `despawnEventProvider` + `spawnEventProvider` directly from the buffer using the current `TemporalData` (`p.tr.GetById`). Prefer the direct event emission if `Spawn` side-effects (slot shuffles) are undesirable. Decide during implementation and keep the appearance-refresh test asserting both events fire.

- [ ] **Step 5: Run to verify it passes**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run TestEvolve -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/pet/processor.go services/atlas-pets/atlas.com/pets/pet/processor_test.go
git commit -m "feat(atlas-pets): EVOLVE rolls outcome and mutates pet in place"
```

### Task 13: `handleEvolveCommand` consumer

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/kafka/consumer/pet/consumer.go`

- [ ] **Step 1: Register + handle**

Register the handler alongside the others (mirror `handleAwardClosenessCommand` registration ~line 43):

```go
			_, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleEvolveCommand(db))))
			if err != nil {
				return err
			}
```

Add the handler (mirror `handleAwardClosenessCommand`, which threads `c.TransactionId`):

```go
func handleEvolveCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.EvolveCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.EvolveCommandBody]) {
		if c.Type != pet2.CommandPetEvolve {
			return
		}
		err := pet.NewProcessor(l, ctx, db).EvolveAndEmit(c.TransactionId, c.PetId)
		if err != nil {
			l.WithError(err).Errorf("Unable to evolve pet [%d].", c.PetId)
		}
	}
}
```

> Match the exact registration idiom used by the surrounding handlers in this file (the `rf`/`InitHandlers` shape).

- [ ] **Step 2: Build + test**

Run: `cd services/atlas-pets/atlas.com/pets && go build ./... && go test ./kafka/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/kafka/consumer/pet/consumer.go
git commit -m "feat(atlas-pets): register EVOLVE command handler"
```

### Task 14: Egg hatch-on-spawn branch (FR-2.1–2.4)

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/processor.go` (`Spawn`, lines 337-432)
- Test: `services/atlas-pets/atlas.com/pets/pet/processor_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSpawnHatchesEgg(t *testing.T) {
	db := testDB(t)
	dp := &dataPetMock{byId: map[uint32]data.Model{
		5000028: eggDragonData(),     // IsEgg(): evolutions=[{5000029,100}], reqItemId=0, reqPetLevel=0
	}}
	// character owns the egg in cash inventory but NOT the baby
	cp := characterMockOwningCash(t, /* templates */ []uint32{5000028})
	p := NewProcessor(testLogger(), testCtx(t), db).With(WithDataProcessor(dp), WithCharacterProcessor(cp))
	pe := createPet(t, p, 5000028, 1, -1)

	err := p.SpawnAndEmit(pe.Id(), pe.OwnerId(), true)
	if err != nil {
		t.Fatalf("SpawnAndEmit: %v", err)
	}
	got, _ := p.GetById(pe.Id())
	if got.TemplateId() != 5000029 {
		t.Errorf("egg did not hatch: templateId = %d, want 5000029", got.TemplateId())
	}
	if got.Slot() != -1 {
		t.Errorf("egg must not spawn: slot = %d, want -1", got.Slot())
	}
}

func TestSpawnHatchRefusedWhenBabyOwned(t *testing.T) {
	db := testDB(t)
	dp := &dataPetMock{byId: map[uint32]data.Model{5000028: eggDragonData()}}
	cp := characterMockOwningCash(t, []uint32{5000028, 5000029}) // already owns baby
	p := NewProcessor(testLogger(), testCtx(t), db).With(WithDataProcessor(dp), WithCharacterProcessor(cp))
	pe := createPet(t, p, 5000028, 1, -1)
	_ = p.SpawnAndEmit(pe.Id(), pe.OwnerId(), true)
	got, _ := p.GetById(pe.Id())
	if got.TemplateId() != 5000028 {
		t.Errorf("egg must not hatch when baby owned: got %d", got.TemplateId())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run TestSpawnHatch -v`
Expected: FAIL (egg spawns normally instead of hatching).

- [ ] **Step 3: Implement the hatch branch**

In `Spawn`, immediately after the ownership check (after line 349, before the `SpawnedByOwnerProvider` call) insert:

```go
						// Egg hatch-on-summon (FR-2.1..2.4): if the template is an
						// egg, hatch into its single target instead of spawning.
						d, derr := p.dp.GetById(pe.TemplateId())
						if derr == nil && d.IsEgg() {
							baby := d.Evolutions()[0].TemplateId()

							// FR-2.2: refuse if the character already owns the baby.
							c, cerr := p.cp.GetById(p.cp.InventoryDecorator)(actorId)
							if cerr != nil {
								return cerr
							}
							if _, owned := c.Inventory().Cash().FindFirstByItemId(baby); owned {
								p.l.Infof("Refusing to hatch egg [%d] for character [%d]: baby [%d] already owned.", petId, actorId, baby)
								return mb.Put(pet.EnvStatusEventTopic, commandResponseEventProvider(pe, 0, false))
							}

							// Mutate the pet row in place: templateId egg->baby,
							// reset stats to defaults (level 1, closeness 0, full).
							hatched, herr := Clone(pe).
								SetTemplateId(baby).
								SetLevel(1).
								SetCloseness(0).
								SetFullness(MaxFullness).
								SetExpiration(pe.Expiration()). // preserve egg expiration
								Build()
							if herr != nil {
								return herr
							}
							if herr = updateOnEvolve(tx)(petId, baby, hatched.Expiration()); herr != nil {
								return herr
							}
							// Cascade the in-place inventory asset swap.
							if herr = p.ip.ChangeTemplate(mb)(uuid.Nil, actorId, petId, baby); herr != nil {
								return herr
							}
							// Egg is consumed; do NOT spawn. Player re-summons the baby.
							return nil
						}
```

> `updateOnEvolve` also resets expiration; here we pass the preserved egg expiration so the baby keeps the egg's remaining lifespan (FR-2.3). Default stats come from the explicit `SetLevel(1)/SetCloseness(0)/SetFullness(MaxFullness)`.

- [ ] **Step 4: Run to verify it passes**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run TestSpawn -v`
Expected: PASS (hatch + refusal + existing spawn tests still green).

- [ ] **Step 5: Full module test/vet/build**

Run: `cd services/atlas-pets/atlas.com/pets && go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/pet/processor.go services/atlas-pets/atlas.com/pets/pet/processor_test.go
git commit -m "feat(atlas-pets): hatch eggs in place on summon"
```

---

# Phase 5 — libs/atlas-saga + atlas-saga-orchestrator (FR-3.1, FR-5.4)

### Task 15: `EvolvePet` action, `PetEvolution` type, payload, unmarshal

**Files:**
- Modify: `libs/atlas-saga/model.go:14-25, 73`
- Modify: `libs/atlas-saga/payloads.go` (after `GainClosenessPayload`, line 255)
- Modify: `libs/atlas-saga/unmarshal.go` (after `GainCloseness` case, line 185)

- [ ] **Step 1: Add the constants**

In `model.go`, add to the `Type` block (after line 25):

```go
	PetEvolution         Type = "pet_evolution"
```

In `model.go`, add to the `Action` block (after `GainCloseness`, line 73):

```go
	EvolvePet              Action = "evolve_pet"
```

- [ ] **Step 2: Add the payload**

In `payloads.go`, after `GainClosenessPayload` (line 255):

```go
// EvolvePetPayload drives an NPC pet evolution. The outcome roll is owned by
// atlas-pets; this payload only identifies the pet.
type EvolvePetPayload struct {
	CharacterId uint32 `json:"characterId"`
	PetId       uint32 `json:"petId"`
}
```

- [ ] **Step 3: Add the unmarshal case**

In `unmarshal.go`, after the `GainCloseness` case (line 185):

```go
	case EvolvePet:
		var payload EvolvePetPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 4: Add an unmarshal round-trip test**

In `unmarshal_test.go`, add a case mirroring an existing one (e.g. `GainCloseness`) that marshals a `Step` with `EvolvePet`/`EvolvePetPayload` and asserts it unmarshals back to the typed payload.

- [ ] **Step 5: Build + test**

Run: `cd libs/atlas-saga && go test ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(atlas-saga): add EvolvePet action and PetEvolution saga type"
```

### Task 16: Orchestrator pet processor `EvolveAndEmit`

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/processor.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/` provider file (where `AwardClosenessProvider` lives)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/pet/kafka.go`

- [ ] **Step 1: Add the EVOLVE command contract (outbound)**

In the orchestrator's `kafka/message/pet/kafka.go`, add (mirror its existing command constants/bodies that match atlas-pets):

```go
	CommandPetEvolve = "EVOLVE"
```
```go
type EvolveCommandBody struct {
}
```

- [ ] **Step 2: Add the provider**

Where `AwardClosenessProvider` is defined, add:

```go
func EvolveProvider(transactionId uuid.UUID, petId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.EvolveCommandBody]{
		TransactionId: transactionId,
		PetId:         petId,
		Type:          pet2.CommandPetEvolve,
		Body:          pet2.EvolveCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
```

Match the field names of the orchestrator's `pet2.Command[E]` envelope (mirror `AwardClosenessProvider`).

- [ ] **Step 3: Add the processor methods**

In `pet/processor.go`, add to the `Processor` interface:

```go
	EvolveAndEmit(transactionId uuid.UUID, petId uint32) error
	Evolve(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error
```

Implement:

```go
func (p *ProcessorImpl) EvolveAndEmit(transactionId uuid.UUID, petId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Evolve(mb)(transactionId, petId)
	})
}

func (p *ProcessorImpl) Evolve(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error {
	return func(transactionId uuid.UUID, petId uint32) error {
		return mb.Put(pet2.EnvCommandTopic, EvolveProvider(transactionId, petId))
	}
}
```

Update the orchestrator's pet mock (if any) to satisfy the grown interface.

- [ ] **Step 4: Build**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/ services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/pet/
git commit -m "feat(saga-orchestrator): pet processor emits EVOLVE command"
```

### Task 17: `handleEvolvePet` handler + `GetHandler` case

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` (interface ~line 103, `GetHandler` switch, handler near `handleGainCloseness` ~line 1205)

- [ ] **Step 1: Add to the Handler interface**

After `handleGainCloseness(s Saga, st Step[any]) error` (line 103):

```go
	handleEvolvePet(s Saga, st Step[any]) error
```

- [ ] **Step 2: Add the GetHandler case**

In the `GetHandler` switch, add:

```go
	case EvolvePet:
		return h.handleEvolvePet, true
```

- [ ] **Step 3: Implement the handler**

After `handleGainCloseness` (mirror it, line 1205):

```go
func (h *HandlerImpl) handleEvolvePet(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(EvolvePetPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	err := h.petP.EvolveAndEmit(s.TransactionId(), payload.PetId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to evolve pet.")
		return err
	}
	return nil
}
```

- [ ] **Step 4: Build + test**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./saga/ -run TestHandle -v`
Expected: clean (add a `handleEvolvePet` dispatch test mirroring an existing handler test if the suite covers handlers).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go
git commit -m "feat(saga-orchestrator): wire handleEvolvePet"
```

### Task 18: `PetEvolution` reverse-walk compensation (FR-5.4)

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` (CompensateFailedStep ~line 180, new dispatch func)
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go`

**Why this is needed:** the default `CompensateFailedStep` only compensates the *failed* step. For the evolution saga `[destroy_item, award_mesos, evolve_pet]`, a failed `evolve_pet` must reverse-walk the **completed** `destroy_item` (refund the Rock) and `award_mesos` (refund the mesos). The existing `CharacterCreation` reverse-walk does not handle `DestroyAsset`/`AwardMesos` inverses — add a `PetEvolution` path that does.

- [ ] **Step 1: Write the failing test**

```go
func TestPetEvolutionCompensationRefundsResources(t *testing.T) {
	// Build a PetEvolution saga with completed destroy_item (Rock 5380000) +
	// completed award_mesos (-50000) + FAILED evolve_pet.
	// Use a spy compartment/character processor capturing refund calls.
	spyComp := &spyCompartmentProcessor{}
	spyChar := &spyCharacterProcessor{}
	c := NewCompensator(...).WithCompartmentProcessor(spyComp).WithCharacterProcessor(spyChar)
	s := petEvolutionSagaWithFailedEvolve(t)

	if err := c.CompensateFailedStep(s); err != nil {
		t.Fatalf("CompensateFailedStep: %v", err)
	}
	if spyComp.createItemCalls != 1 {
		t.Errorf("Rock not refunded: RequestCreateItem calls = %d, want 1", spyComp.createItemCalls)
	}
	if spyChar.awardMesosCalls != 1 || spyChar.lastMesosAmount <= 0 {
		t.Errorf("mesos not refunded: calls=%d amount=%d", spyChar.awardMesosCalls, spyChar.lastMesosAmount)
	}
}
```

> Mirror existing compensator tests' construction of `Saga`/`Step` and the spy/mocks already used in `compensator_test.go`. Use the saga builder to set type `PetEvolution`.

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestPetEvolutionCompensation -v`
Expected: FAIL (no refund dispatched).

- [ ] **Step 3: Add the PetEvolution branch + dispatcher**

In `CompensateFailedStep`, after the `CharacterCreation` branch (line 181):

```go
	if s.SagaType() == PetEvolution {
		return c.compensatePetEvolution(s, failedStep)
	}
```

Add (mirror `compensateCharacterCreation` + `DispatchCharacterCreationRollbacks`, lines 908-1027):

```go
func (c *CompensatorImpl) compensatePetEvolution(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("PetEvolution saga failing — dispatching reverse-walk compensation.")

	c.DispatchPetEvolutionRollbacks(s)

	if !GetCache().TryTransition(c.ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed) {
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())
		return nil
	}
	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	reason := fmt.Sprintf("Pet evolution failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithField("transaction_id", s.TransactionId().String()).Error("Failed to emit saga failed event after pet-evolution compensation.")
		return err
	}
	return nil
}

// DispatchPetEvolutionRollbacks reverse-walks completed steps, refunding the
// destroyed Rock (DestroyAsset) and the deducted mesos (AwardMesos). evolve_pet
// produced no committed mutation on failure, so it has no inverse.
func (c *CompensatorImpl) DispatchPetEvolutionRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		switch step.Action() {
		case DestroyAsset:
			if payload, ok := step.Payload().(DestroyAssetPayload); ok {
				qty := payload.Quantity
				if qty == 0 {
					qty = 1
				}
				if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"template_id":    payload.TemplateId,
					}).Error("Reverse-walk: DestroyAsset → CreateItem dispatch failed; continuing chain.")
				}
			}
		case AwardMesos:
			if payload, ok := step.Payload().(AwardMesosPayload); ok {
				ch := channel.NewModel(payload.WorldId, payload.ChannelId) // mirror handleAwardMesos channel construction
				if err := c.charP.AwardMesosAndEmit(s.TransactionId(), ch, payload.CharacterId, payload.CharacterId, "SYSTEM", -payload.Amount, false); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"amount":         payload.Amount,
					}).Error("Reverse-walk: AwardMesos refund dispatch failed; continuing chain.")
				}
			}
		}
	}
}
```

> **Field names to confirm during implementation:**
> - `DestroyAssetPayload` field names (`CharacterId`, `TemplateId`, `Quantity`) — read `libs/atlas-saga/payloads.go` and match exactly.
> - `AwardMesosPayload` fields (`CharacterId`, `WorldId`, `ChannelId`, `Amount`, and the `Amount` Go type — `int32`). The refund is `-payload.Amount` (the consume step used a negative amount, so negating restores it). Confirm the sign convention against how the npc `award_mesos` op builds the payload and how `handleAwardMesos` applies it; the refund must net the player back to even.
> - Channel model construction: mirror exactly how `handleAwardMesos` (handler.go) builds the `channel.Model` it passes to `AwardMesosAndEmit`. Add `charP` to the compensator if it is not already a field (it already has `compP`; check for a character processor field and add `WithCharacterProcessor` like the orchestrator handler has).

- [ ] **Step 4: Run to verify it passes**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestPetEvolutionCompensation -v`
Expected: PASS.

- [ ] **Step 5: Full module test/vet/build**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go
git commit -m "feat(saga-orchestrator): PetEvolution reverse-walk refunds Rock and mesos"
```

---

# Phase 6 — atlas-npc-conversations (FR-5.1–5.5)

### Task 19: Extend the npc pet client with templateId + level

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/pet/model.go`
- Modify: `services/atlas-npc-conversations/atlas.com/npc/pet/rest.go`

- [ ] **Step 1: Add fields to the model + Extract**

In `pet/model.go`, extend `Model` and add getters:

```go
type Model struct {
	id         uint32
	templateId uint32
	level      byte
	slot       int8
}

func (m Model) TemplateId() uint32 { return m.templateId }
func (m Model) Level() byte        { return m.level }
```

Update the constructor used by `Extract` to set the new fields. In `pet/rest.go`, update `Extract` to populate `templateId` and `level` from the (already-present) `RestModel.TemplateId`/`RestModel.Level` fields.

- [ ] **Step 2: Build**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./pet/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/pet/
git commit -m "feat(npc-conversations): expose pet templateId and level"
```

### Task 20: Thin atlas-data evolution client

**Files:**
- Create: `services/atlas-npc-conversations/atlas.com/npc/petdata/{model,rest,requests,processor}.go`

- [ ] **Step 1: Create the client**

Mirror the existing `pet/` client structure (`requests.go` using `requests.RootUrl("DATA")`, `GET /data/pets/{id}`). The model exposes evolution eligibility:

`petdata/model.go`:
```go
package petdata

type Model struct {
	id          uint32
	reqPetLevel uint32
	reqItemId   uint32
	evolutions  int
}

func (m Model) Id() uint32          { return m.id }
func (m Model) ReqPetLevel() uint32 { return m.reqPetLevel }
func (m Model) ReqItemId() uint32   { return m.reqItemId }

// IsEvolvable reports an NPC-evolvable pet (gated by a required item).
func (m Model) IsEvolvable() bool { return m.evolutions > 0 && m.reqItemId != 0 }
```

`petdata/rest.go`: `RestModel` with `reqPetLevel`, `reqItemId`, `evolutions []EvolutionRestModel` json tags + `Extract` mapping to `Model` (set `evolutions: len(rm.Evolutions)`). Implement the JSON:API `GetName() string { return "pets" }`, `GetID`, `SetID` like the sibling `pet/rest.go`.

`petdata/requests.go`:
```go
package petdata

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const ById = "data/pets/%d"

func requestById(petTemplateId uint32) requests.Request[RestModel] {
	return requests.MakeGetRequest[RestModel](fmt.Sprintf(requests.RootUrl("DATA")+"/"+ById, petTemplateId))
}
```

> Confirm the exact `requests.RootUrl` env key and request helper names against `services/atlas-pets/atlas.com/pets/data/pet/requests.go` (which already calls atlas-data) and mirror them precisely — the env var is the atlas-data service URL key used elsewhere.

`petdata/processor.go`: `Processor` interface with `GetById(petTemplateId uint32) (Model, error)` using `requests.Provider[RestModel, Model](l, ctx)(requestById(id), Extract)()`. Mirror `pet/processor.go`.

- [ ] **Step 2: Build**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./petdata/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/petdata/
git commit -m "feat(npc-conversations): add atlas-data pet evolution client"
```

### Task 21: `local:enumerate_evolvable_pets` operation

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` (`executeLocalOperation` switch ~line 325; executor struct to add `petdataP`)

- [ ] **Step 1: Write the failing test**

In the operation_executor test file, add a test that stubs `petP.GetPets` (1 summoned baby dragon at level 20, 1 at level 10) and `petdataP.GetById` (5000029 → evolvable, reqPetLevel 15) and asserts the local op writes the eligible pet ids to context and the count. Mirror existing local-op tests.

```go
func TestEnumerateEvolvablePets(t *testing.T) {
	// petP returns two summoned pets: {id:1, template:5000029, level:20}, {id:2, template:5000029, level:10}
	// petdataP returns reqPetLevel=15, evolvable for 5000029
	e := newTestExecutor(t, withPets(...), withPetData(...))
	err := e.executeLocalOperation(testField(), 100, opEnumerate("evolvablePets", "evolvableCount"))
	if err != nil {
		t.Fatalf("op: %v", err)
	}
	// expect context "evolvableCount" == "1" and "evolvablePets" == "1" (only the level-20 pet)
	assertContext(t, e, 100, "evolvableCount", "1")
	assertContext(t, e, 100, "evolvablePets", "1")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestEnumerateEvolvablePets -v`
Expected: FAIL (unknown operation).

- [ ] **Step 3: Implement the local op**

Add `petdataP petdata.Processor` to the executor struct and constructor. In `executeLocalOperation` (after the existing cases, ~line 325 switch), add:

```go
	case "enumerate_evolvable_pets":
		outputKey := operation.Params()["outputContextKey"]
		if outputKey == "" {
			outputKey = "evolvablePets"
		}
		countKey := operation.Params()["countContextKey"]
		if countKey == "" {
			countKey = "evolvableCount"
		}
		pets, err := e.petP.GetPets(characterId)()
		if err != nil {
			return err
		}
		eligible := make([]string, 0)
		for _, pt := range pets {
			if !pt.IsSpawned() {
				continue
			}
			d, derr := e.petdataP.GetById(pt.TemplateId())
			if derr != nil {
				continue
			}
			if d.IsEvolvable() && uint32(pt.Level()) >= d.ReqPetLevel() {
				eligible = append(eligible, strconv.Itoa(int(pt.Id())))
			}
		}
		if err := e.setContextValue(characterId, outputKey, strings.Join(eligible, ",")); err != nil {
			return err
		}
		return e.setContextValue(characterId, countKey, strconv.Itoa(len(eligible)))
```

The operation type string in the conversation JSON is `"local:enumerate_evolvable_pets"` (the `local:` prefix routes it through `executeLocalOperation`, which trims the prefix before the switch — confirm the trim at line ~325).

> When exactly one pet is eligible, the conversation auto-selects by reading the single id from `evolvablePets` into `selectedPetId`. The >1 case drives a `listSelection` state (Task 23 JSON). `enumerate` only computes the list.

- [ ] **Step 4: Run to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestEnumerateEvolvablePets -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go
git commit -m "feat(npc-conversations): enumerate_evolvable_pets local operation"
```

### Task 22: `evolve_pet` remote operation + `PetEvolution` saga-type override

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` (`createStepForOperation` switch ~line 1021; `createSagaForOperations` ~line 823)

- [ ] **Step 1: Write the failing test**

```go
func TestEvolvePetBuildsStep(t *testing.T) {
	e := newTestExecutor(t)
	stepId, status, action, payload, err := e.createStepForOperation(testField(), 100, opEvolvePet("{context.selectedPetId}"))
	if err != nil {
		t.Fatalf("createStepForOperation: %v", err)
	}
	if action != saga.EvolvePet || status != saga.Pending {
		t.Errorf("action/status = %v/%v", action, status)
	}
	p, ok := payload.(saga.EvolvePetPayload)
	if !ok || p.CharacterId != 100 || p.PetId == 0 {
		t.Errorf("payload = %+v", payload)
	}
	_ = stepId
}

func TestEvolutionBatchUsesPetEvolutionSagaType(t *testing.T) {
	e := newTestExecutor(t)
	ops := []OperationModel{opDestroyItem(5380000), opAwardMesos(-50000), opEvolvePet("1")}
	s, err := e.createSagaForOperations(testField(), 100, ops)
	if err != nil {
		t.Fatalf("createSagaForOperations: %v", err)
	}
	if s.SagaType() != saga.PetEvolution {
		t.Errorf("SagaType = %v, want PetEvolution", s.SagaType())
	}
	if len(s.Steps()) != 3 {
		t.Errorf("steps = %d, want 3", len(s.Steps()))
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run "TestEvolvePetBuildsStep|TestEvolutionBatch" -v`
Expected: FAIL.

- [ ] **Step 3: Implement the remote op + saga-type override**

In `createStepForOperation` (switch at line 1021), add:

```go
	case "evolve_pet":
		petSelector, exists := operation.Params()["petId"]
		if !exists {
			return "", "", "", nil, errors.New("missing petId parameter for evolve_pet operation")
		}
		petIdInt, err := e.evaluateContextValueAsInt(characterId, "petId", petSelector)
		if err != nil {
			return "", "", "", nil, err
		}
		payload := saga.EvolvePetPayload{
			CharacterId: characterId,
			PetId:       uint32(petIdInt),
		}
		return stepId, saga.Pending, saga.EvolvePet, payload, nil
```

(`stepId` is already derived at the top of `createStepForOperation` as `"<type>-<characterId>"`.)

In `createSagaForOperations` (line 823), make the saga type conditional on the batch containing an `evolve_pet` step. After the `built` slice is constructed (line 849), before `builder.AddStep`:

```go
	sagaType := saga.InventoryTransaction
	for _, st := range built {
		if st.action == saga.EvolvePet {
			sagaType = saga.PetEvolution
			break
		}
	}
	builder.SetSagaType(sagaType)
```

Change the initial builder construction (lines 825-827) to not hard-set the type, or call `SetSagaType` again here (last write wins). Keep `SetInitiatedBy("npc-conversation-batch")`.

> This is the linchpin that makes the orchestrator's `PetEvolution` reverse-walk (Task 18) fire. Without it, the batch is tagged `InventoryTransaction` and a failed `evolve_pet` would not refund the Rock + mesos.

- [ ] **Step 4: Run to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run "TestEvolvePet|TestEvolutionBatch" -v`
Expected: PASS.

- [ ] **Step 5: Full module test/vet/build**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go
git commit -m "feat(npc-conversations): evolve_pet remote op and PetEvolution saga type"
```

### Task 23: Reference Garnox conversation definition (FR-5.5)

**Files:**
- Create: `deploy/seed/<region>/<version>/npc-conversations/npc/npc-1032102.json`

- [ ] **Step 1: Locate the seed directory + format**

Run: `ls deploy/seed/gms/12_1/npc-conversations/npc/ | head` and open one existing file that uses `listSelection` + a `genericAction` with remote operations (e.g. the skin-coupon NPC) to copy the exact envelope shape (`{"data":{"attributes":{...},"id":"...","type":"npc-conversation"}}`).

- [ ] **Step 2: Author the conversation**

Create `npc-1032102.json` (Garnox) with states:

1. `start` (`genericAction`): operation `local:enumerate_evolvable_pets` → outcomes branch on context `evolvableCount`:
   - `evolvableCount == "0"` → `noEligible`
   - `evolvableCount == "1"` → `confirmOne` (auto-select: a `genericAction` op copies the single id from `evolvablePets` to `selectedPetId`, or use the enumerated single value directly)
   - else → `choosePet`
2. `choosePet` (`listSelection`): present the eligible pets; store choice in `context.selectedPetId`; → `confirm`.
3. `confirmOne`/`confirm` (`listSelection` or `dialogue`): gate on conditions (`item >= 1` for Rock `5380000`; `meso >= <cost>`); "Yes" → `doEvolve`, "No" → end.
4. `doEvolve` (`genericAction`): operations IN ORDER:
   - `{ "type": "destroy_item", "params": { "itemId": "5380000", "quantity": "1" } }`
   - `{ "type": "award_mesos", "params": { "amount": "-<cost>" } }`
   - `{ "type": "evolve_pet", "params": { "petId": "{context.selectedPetId}" } }`
   → `success`.
5. `success` (`dialogue`): confirmation message. `noEligible` (`dialogue`): "come back when your pet is stronger."

> Use a real meso cost consistent with the era (confirm against Cosmic `1032102.js` if available; otherwise pick a reasonable placeholder and note it). The three `doEvolve` operations are all remote, so they batch into one `PetEvolution` saga (Task 22).

- [ ] **Step 3: Validate JSON**

Run: `cat deploy/seed/.../npc-1032102.json | python3 -m json.tool > /dev/null && echo OK`
Expected: `OK`.

- [ ] **Step 4: Commit**

```bash
git add deploy/seed/
git commit -m "feat(npc-conversations): reference Garnox pet evolution conversation"
```

---

# Phase 7 — Verification & docker bake (CLAUDE.md gate)

### Task 24: atlas-channel verify-only confirmation

**Files:** none (verification).

- [ ] **Step 1: Confirm no version branching**

Read `libs/atlas-packet/pet/clientbound/activated.go` and confirm `templateId` is written with no `MajorVersion`/`Region` branch (verified during planning at `activated.go:55`). Read `services/atlas-channel/.../kafka/consumer/pet/consumer.go` and confirm `SPAWNED`/`DESPAWNED` handling, and the asset consumer's `UPDATED` handler. No code change.

- [ ] **Step 2: Document in audit/PR**

Note in the PR description that atlas-channel is verify-only and appearance refresh rides the existing despawn/respawn + asset `UPDATED` events; runtime verification on a v83 and a v95+ tenant is an acceptance step.

### Task 25: Full build/test/vet/bake/redis-guard gate

**Files:** none (verification).

- [ ] **Step 1: Per-module test + vet + build**

For each changed module run from its module root:

```bash
go test -race ./... && go vet ./... && go build ./...
```

Modules: `libs/atlas-saga`, `services/atlas-data/atlas.com/data`, `services/atlas-pets/atlas.com/pets`, `services/atlas-inventory/atlas.com/inventory`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-npc-conversations/atlas.com/npc`.
Expected: all clean.

- [ ] **Step 2: redis-key-guard**

Run from repo root: `GOWORK=off tools/redis-key-guard.sh`
Expected: clean (no new raw keyed go-redis usage was added).

- [ ] **Step 3: docker buildx bake every service whose go.mod was touched**

From the worktree root:

```bash
docker buildx bake atlas-data atlas-pets atlas-inventory atlas-saga-orchestrator atlas-npc-conversations
```

Expected: all targets build. (No `go.mod` was added for a new lib, and no new `libs/` were created, so the root `Dockerfile`/`go.work` need no edits. `libs/atlas-saga` changes are picked up by the services that import it — bake those services, not a lib.)

> If any bake fails on a missing `COPY libs/...`, that means a new lib dependency edge was introduced — fix the root `Dockerfile` + `go.work` per CLAUDE.md. Not expected here.

- [ ] **Step 4: Commit any fixups**

```bash
git add -A
git commit -m "chore(task-089): verification fixups"
```

---

## Acceptance criteria mapping (PRD §10)

| PRD acceptance item | Task(s) |
|---|---|
| atlas-data parses evol nodes; non-evolvable still reads | 1, 2 |
| Egg hatch in place; refused if baby owned | 14 (+11 cascade, 6 inventory) |
| NPC evolves eligible pet; level gate; Rock+mesos; data-weighted roll in atlas-pets | 12, 21, 22, 23 |
| Pet keeps id/cashId/stats; templateId rolled; expiration reset 90d | 7, 10, 12 |
| Asset templateId swapped in place; pet not deleted by asset consumer | 4, 5, 6 (UPDATED, never DELETED) |
| Summoned pet visually becomes new form (v83 + v95) | 12 (despawn/respawn), 24 |
| Re-evolving an adult re-rolls via same path | 12 (data-driven; adult templates carry evolutions) |
| Failed evolution refunds Rock + mesos | 18, 22 |
| >1 evolvable → menu; exactly one → auto-select | 21, 23 |
| EVOLVE/EVOLVED + inventory swap cmd/event exist | 8, 6 |
| No hard-coded probability tables; data-driven | 1, 2, 12 (weighted roll reads WZ weights) |
| Build/test/vet/bake/redis-guard clean | 25 |

## Open items resolved during planning

- **OQ-3 in-place swap feasibility** — confirmed no existing path; `CHANGE_TEMPLATE` added (Tasks 4–6).
- **Compensator gap** — confirmed `CharacterCreation` reverse-walk does NOT cover `DestroyAsset`/`AwardMesos`; new `PetEvolution` path added (Task 18) and the npc batch must tag `PetEvolution` (Task 22).
- **WZ verification gate** — confirmed against local XML; egg discriminator validated; weights are relative (Task 12 roll sums weights).
- **Hatched-baby name/stats** — default stats; expiration preserved (Task 14).
