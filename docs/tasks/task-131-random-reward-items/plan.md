# Random Reward Items Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a player use a `reward`-node Consume item and atomically receive one prob-weighted random reward (with expiration, effect, and world-announce where the data defines them), across GMS v83/v84/v87/v95.

**Architecture:** New serverbound codec (`libs/atlas-packet`) + atlas-channel handler emits a `REQUEST_ITEM_REWARD` command; atlas-consumables validates → reserves the box → weighted-rolls (crypto/rand) → emits `CREATE_ASSET` for the reward → on `CREATED` commits the box consume and emits presentation events, on `CREATION_FAILED` cancels the reservation (box kept). atlas-data parses three new per-reward-entry fields. Presentation reuses the already-implemented-but-unwired `EffectLotteryUse` codecs plus the existing world-message and status-message writers.

**Tech Stack:** Go (DDD immutable models + Builder, JSON:API REST, segmentio Kafka, functional `model.Provider`), the shared `libs/atlas-*` libraries, JSON seed templates in atlas-configurations.

## Global Constraints

- **Versions in scope (expanded 2026-07-15, post-`main`-merge — design v2 §2.1/§2.6/§2.7):**
  dedicated lottery opcode in v72 (0x6F), v79 (0x6E), v83 (0x70), v84 (0x70),
  v87 (0x73), v92 (0x7B), v95 (0x7C), jms (0x6B) — v72/v79/jms IDA-verified live,
  v84/v87/v92 by no-IDB registry/CSV lineage. v48/v61 have NO dedicated opcode
  (introduced at v72); reward boxes there use the generic item-use path, with the
  server detecting the reward table in `RequestItemConsume` (§2.7). Do NOT invent
  a lottery opcode for v48/v61.
- **Never hardcode a version branch** keyed on `>83`. Per-version behavior comes
  from tenant config tables (opcodes, effect/world-message/status modes) only.
- **Randomness: `crypto/rand` only** (`rand.Int(rand.Reader, big.NewInt(...))`).
  No `math/rand` anywhere in the reward path.
- **Immutable models:** private fields + getters + Builder; no test-only
  constructors / `*_testhelpers.go` (use the Builder in tests).
- **Before defining any new type/alias/constant, check `libs/atlas-constants`** for
  an existing equivalent (DOM-21). No new numeric types expected here.
- **`period` unit is MINUTES:** `expiration = now + period*time.Minute` when
  `period > 0`; `period <= 0` → zero `time.Time` (no expiration).
- **Serverbound body is invariant:** `slot int16, itemId int32`, no updateTime.
- **Every terminal path must unstick the client** (success via inventory events;
  every failure via the ERROR arm's `StatChanged([], true)`).
- **Verification gate (per CLAUDE.md), run at the end and after risky tasks:**
  `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed
  module; `docker buildx bake atlas-{data,consumables,channel,configurations}`
  from the worktree root; `tools/redis-key-guard.sh` clean from repo root
  (`GOWORK=off`). Changed modules: `libs/atlas-packet`,
  `services/atlas-data/atlas.com/data`,
  `services/atlas-consumables/atlas.com/consumables`,
  `services/atlas-channel/atlas.com/channel`, `services/atlas-configurations`.
- All commands run from the worktree root (`.worktrees/task-131-random-reward-items`).
  Every commit stays on branch `task-131-random-reward-items`.

---

## Task 1: atlas-data — parse per-entry `Effect` / `worldMsg` / `period`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/consumable/rest.go:125-129` (`RewardRestModel`)
- Modify: `services/atlas-data/atlas.com/data/consumable/reader.go:164-172` (reward loop)
- Test: `services/atlas-data/atlas.com/data/consumable/reader_test.go`

**Interfaces:**
- Produces: `RewardRestModel{ItemId uint32; Count uint32; Prob uint32; Effect string; WorldMsg string; Period int32}` serialized as JSON keys `itemId,count,prob,effect,worldMsg,period`.

- [ ] **Step 1: Write the failing test.** Append to `reader_test.go`:

```go
func TestReaderRewardFields(t *testing.T) {
	l, _ := test.NewNullLogger()

	const xmlData = `
<imgdir name="0202.img">
  <imgdir name="02022309">
    <imgdir name="info">
      <int name="price" value="0"/>
    </imgdir>
    <imgdir name="reward">
      <imgdir name="0">
        <int name="item" value="1132010"/>
        <int name="count" value="1"/>
        <int name="prob" value="100"/>
        <string name="Effect" value="Effect/BasicEff/Event1/Good"/>
        <string name="worldMsg" value="/name got /item"/>
        <int name="period" value="7200"/>
      </imgdir>
      <imgdir name="1">
        <int name="item" value="2000000"/>
        <int name="count" value="5"/>
        <int name="prob" value="900"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>
`
	rms := Read(l)(xml.FromByteArrayProvider([]byte(xmlData)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	box, ok := rmm[strconv.Itoa(2022309)]
	if !ok {
		t.Fatalf("rmm[2022309] does not exist.")
	}
	if len(box.Rewards) != 2 {
		t.Fatalf("len(box.Rewards) = %d, want 2", len(box.Rewards))
	}
	r0 := box.Rewards[0]
	if r0.ItemId != 1132010 || r0.Count != 1 || r0.Prob != 100 {
		t.Fatalf("r0 base = {%d,%d,%d}, want {1132010,1,100}", r0.ItemId, r0.Count, r0.Prob)
	}
	if r0.Effect != "Effect/BasicEff/Event1/Good" {
		t.Errorf("r0.Effect = %q, want the WZ effect path", r0.Effect)
	}
	if r0.WorldMsg != "/name got /item" {
		t.Errorf("r0.WorldMsg = %q, want the announce template", r0.WorldMsg)
	}
	if r0.Period != 7200 {
		t.Errorf("r0.Period = %d, want 7200", r0.Period)
	}
	// Entry with no Effect/worldMsg/period must default cleanly.
	r1 := box.Rewards[1]
	if r1.Effect != "" || r1.WorldMsg != "" {
		t.Errorf("r1 Effect/WorldMsg = %q/%q, want empty", r1.Effect, r1.WorldMsg)
	}
	if r1.Period != -1 {
		t.Errorf("r1.Period = %d, want -1 (default)", r1.Period)
	}
}
```

- [ ] **Step 2: Run it, confirm it fails.**

Run: `cd services/atlas-data/atlas.com/data && go test ./consumable/ -run TestReaderRewardFields -v`
Expected: FAIL — `RewardRestModel` has no `Effect`/`WorldMsg`/`Period` fields (compile error), or Period defaults to 0 not -1.

- [ ] **Step 3: Extend `RewardRestModel`.** In `rest.go`, replace the struct at lines 125-129:

```go
type RewardRestModel struct {
	ItemId   uint32 `json:"itemId"`
	Count    uint32 `json:"count"`
	Prob     uint32 `json:"prob"`
	Effect   string `json:"effect"`
	WorldMsg string `json:"worldMsg"`
	Period   int32  `json:"period"`
}
```

- [ ] **Step 4: Parse the fields in `reader.go`.** Replace the reward loop body at lines 164-172:

```go
			r, err := cxml.ChildByName("reward")
			if err == nil && r != nil {
				for _, ro := range r.ChildNodes {
					// Per-entry reward fields. Note capital "Effect" (verified WZ
					// casing) — distinct from the item-level lowercase "effect"
					// parsed at reader.go ~L80. period defaults to -1 (= no
					// expiration); Effect/worldMsg default to "".
					m.Rewards = append(m.Rewards, RewardRestModel{
						ItemId:   uint32(ro.GetIntegerWithDefault("item", 0)),
						Count:    uint32(ro.GetIntegerWithDefault("count", 0)),
						Prob:     uint32(ro.GetIntegerWithDefault("prob", 0)),
						Effect:   ro.GetString("Effect", ""),
						WorldMsg: ro.GetString("worldMsg", ""),
						Period:   int32(ro.GetIntegerWithDefault("period", -1)),
					})
				}
			}
```

- [ ] **Step 5: Run the test, confirm it passes.**

Run: `cd services/atlas-data/atlas.com/data && go test ./consumable/ -run TestReaderRewardFields -v`
Expected: PASS. Also run the existing `TestReader` to confirm no regression: `go test ./consumable/ -v`.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-data/atlas.com/data/consumable/rest.go services/atlas-data/atlas.com/data/consumable/reader.go services/atlas-data/atlas.com/data/consumable/reader_test.go
git commit -m "feat(task-131): parse per-entry reward Effect/worldMsg/period in atlas-data"
```

---

## Task 2: atlas-consumables — mirror reward fields + add getters

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go:190-194` (`RewardModel`) and add `Rewards()` getter to `Model`
- Modify: `services/atlas-consumables/atlas.com/consumables/data/consumable/rest.go:169-181` (`RewardRestModel`, `ExtractReward`)
- Test: `services/atlas-consumables/atlas.com/consumables/data/consumable/rest_test.go` (create)

**Interfaces:**
- Produces:
  - `RewardModel` getters `ItemId() uint32`, `Count() uint32`, `Prob() uint32`, `Effect() string`, `WorldMsg() string`, `Period() int32`.
  - `Model.Rewards() []RewardModel`.
  - `RewardRestModel{ItemId,Count,Prob uint32; Effect,WorldMsg string; Period int32}`.

- [ ] **Step 1: Write the failing test.** Create `rest_test.go`:

```go
package consumable

import "testing"

func TestExtractRewardFields(t *testing.T) {
	rm := RewardRestModel{ItemId: 1132010, Count: 1, Prob: 100, Effect: "Effect/BasicEff/Event1/Good", WorldMsg: "/name got /item", Period: 7200}
	got, err := ExtractReward(rm)
	if err != nil {
		t.Fatal(err)
	}
	if got.ItemId() != 1132010 || got.Count() != 1 || got.Prob() != 100 {
		t.Fatalf("base = {%d,%d,%d}", got.ItemId(), got.Count(), got.Prob())
	}
	if got.Effect() != "Effect/BasicEff/Event1/Good" {
		t.Errorf("Effect() = %q", got.Effect())
	}
	if got.WorldMsg() != "/name got /item" {
		t.Errorf("WorldMsg() = %q", got.WorldMsg())
	}
	if got.Period() != 7200 {
		t.Errorf("Period() = %d", got.Period())
	}
}

func TestExtractPropagatesRewardsToModel(t *testing.T) {
	rm := RestModel{Id: 2022309, Rewards: []RewardRestModel{{ItemId: 1, Count: 1, Prob: 10, Period: -1}}}
	m, err := Extract(rm)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rewards()) != 1 {
		t.Fatalf("len(m.Rewards()) = %d, want 1", len(m.Rewards()))
	}
	if m.Rewards()[0].Prob() != 10 || m.Rewards()[0].Period() != -1 {
		t.Errorf("reward = prob %d period %d", m.Rewards()[0].Prob(), m.Rewards()[0].Period())
	}
}
```

- [ ] **Step 2: Run it, confirm it fails.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./data/consumable/ -run TestExtract -v`
Expected: FAIL — `RewardModel` has no getters/fields; `Model` has no `Rewards()`.

- [ ] **Step 3: Extend `RewardModel` + add getters in `model.go`.** Replace the struct at lines 190-194:

```go
type RewardModel struct {
	itemId   uint32
	count    uint32
	prob     uint32
	effect   string
	worldMsg string
	period   int32
}

func (m RewardModel) ItemId() uint32  { return m.itemId }
func (m RewardModel) Count() uint32    { return m.count }
func (m RewardModel) Prob() uint32     { return m.prob }
func (m RewardModel) Effect() string   { return m.effect }
func (m RewardModel) WorldMsg() string { return m.worldMsg }
func (m RewardModel) Period() int32    { return m.period }
```

Add a `Rewards()` getter to `Model` (place it next to the other `Model` getters, e.g. after `MonsterSummons()` around line 188):

```go
func (m Model) Rewards() []RewardModel {
	return m.rewards
}
```

- [ ] **Step 4: Extend `RewardRestModel` + `ExtractReward` in `rest.go`.** Replace lines 169-181:

```go
type RewardRestModel struct {
	ItemId   uint32 `json:"itemId"`
	Count    uint32 `json:"count"`
	Prob     uint32 `json:"prob"`
	Effect   string `json:"effect"`
	WorldMsg string `json:"worldMsg"`
	Period   int32  `json:"period"`
}

func ExtractReward(rm RewardRestModel) (RewardModel, error) {
	return RewardModel{
		itemId:   rm.ItemId,
		count:    rm.Count,
		prob:     rm.Prob,
		effect:   rm.Effect,
		worldMsg: rm.WorldMsg,
		period:   rm.Period,
	}, nil
}
```

- [ ] **Step 5: Run the test, confirm it passes.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./data/consumable/ -run TestExtract -v`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/data/consumable/model.go services/atlas-consumables/atlas.com/consumables/data/consumable/rest.go services/atlas-consumables/atlas.com/consumables/data/consumable/rest_test.go
git commit -m "feat(task-131): mirror reward Effect/worldMsg/period + getters in atlas-consumables"
```

---

## Task 3: atlas-packet — `LotteryItemUse` serverbound codec + fixtures

**Files:**
- Create: `libs/atlas-packet/inventory/serverbound/lottery_item_use.go`
- Create: `libs/atlas-packet/inventory/serverbound/lottery_item_use_test.go`

**Interfaces:**
- Produces: const `CharacterItemUseLotteryHandle = "CharacterItemUseLotteryHandle"`; type `LotteryItemUse{source int16; itemId uint32}` with `NewLotteryItemUse() LotteryItemUse`, getters `Source() int16`, `ItemId() uint32`, `Operation() string`, and `Encode`/`Decode` matching the sibling `ItemUse` shape minus `updateTime`.

- [ ] **Step 1: Write the failing test.** Create `lottery_item_use_test.go` (mirrors `item_use_test.go`; the package alias is `test`, not `pt`):

```go
package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/serverbound/LotteryItemUse version=gms_v83 ida=0xa1249f
// packet-audit:verify packet=inventory/serverbound/LotteryItemUse version=gms_v95 ida=0x9d6c50
func TestLotteryItemUseRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := LotteryItemUse{source: 5, itemId: 2022309}
			output := LotteryItemUse{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}
```

- [ ] **Step 2: Run it, confirm it fails.**

Run: `cd libs/atlas-packet && go test ./inventory/serverbound/ -run TestLotteryItemUse -v`
Expected: FAIL — `LotteryItemUse` undefined (compile error).

- [ ] **Step 3: Create the codec.** Write `lottery_item_use.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterItemUseLotteryHandle = "CharacterItemUseLotteryHandle"

// LotteryItemUse - reward-box ("lottery") use request.
// packet-audit:fname CWvsContext::SendLotteryItemUseRequest
// Body is invariant across GMS v83-v95: slot int16, itemId int32. There is no
// leading updateTime (unlike CUser::SendStatChangeItemUseRequest). IDA-verified
// v83 fn 0xa1249f, v95 fn 0x9d6c50 (design task-131 §2.1).
type LotteryItemUse struct {
	source int16
	itemId uint32
}

func NewLotteryItemUse() LotteryItemUse {
	return LotteryItemUse{}
}

func (m LotteryItemUse) Source() int16 { return m.source }
func (m LotteryItemUse) ItemId() uint32 { return m.itemId }

func (m LotteryItemUse) Operation() string {
	return CharacterItemUseLotteryHandle
}

func (m LotteryItemUse) String() string {
	return fmt.Sprintf("source [%d], itemId [%d]", m.source, m.itemId)
}

func (m LotteryItemUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *LotteryItemUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Run the test, confirm it passes.**

Run: `cd libs/atlas-packet && go test ./inventory/serverbound/ -run TestLotteryItemUse -v`
Expected: PASS across all `test.Variants` (GMS v28/v83/v84/v86/v87/v95, JMS v185).

- [ ] **Step 5: Commit.**

```bash
git add libs/atlas-packet/inventory/serverbound/lottery_item_use.go libs/atlas-packet/inventory/serverbound/lottery_item_use_test.go
git commit -m "feat(task-131): add LotteryItemUse serverbound codec with byte fixtures"
```

> Matrix promotion / evidence records for the STATUS.md `LOTTERY_ITEM_USE_REQUEST`
> row (including v84/v87 no-IDB lineage cells) are handled in Task 12 via the
> verify-packet playbook — do NOT invent IDA addresses for v84/v87 here.

---

## Task 4: atlas-consumables — CREATE_ASSET command + CREATED/CREATION_FAILED contract

Mirror atlas-inventory's contract into consumables so the reward grant can issue
`CREATE_ASSET` and correlate on `CREATED`/`CREATION_FAILED`. See `context.md`
"CREATE_ASSET contract gap".

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/compartment/kafka.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/compartment/producer.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/compartment/processor.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/once/compartment/once.go`
- Test: `services/atlas-consumables/atlas.com/consumables/kafka/message/compartment/kafka_test.go` (create)

**Interfaces:**
- Produces:
  - Constants `CommandCreateAsset = "CREATE_ASSET"`, `StatusEventTypeCreated = "CREATED"`, `StatusEventTypeCreationFailed = "CREATION_FAILED"`, `CreateAssetInventoryFull = "CREATE_ASSET_INVENTORY_FULL"`, `CreateAssetTemplateNotFound = "CREATE_ASSET_TEMPLATE_NOT_FOUND"`, `CreateAssetUnknownError = "CREATE_ASSET_UNKNOWN_ERROR"`.
  - `CreateAssetCommandBody{TemplateId,Quantity uint32; Expiration time.Time; OwnerId uint32; Flag uint16; Rechargeable uint64; UseAverageStats bool}`.
  - `StatusEvent[E]` gains top-level `TransactionId uuid.UUID`.
  - `CreateResultEventBody{Type byte; Capacity uint32; ErrorCode string; Message string}` (combined body that deserializes both CREATED and CREATION_FAILED).
  - `(*compartment.Processor).RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error`.
  - `once.CreationValidator(transactionId uuid.UUID) message.Validator[compartment.StatusEvent[compartment.CreateResultEventBody]]`.

- [ ] **Step 1: Write the failing test.** Create `kafka/message/compartment/kafka_test.go`:

```go
package compartment

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestCreateAssetCommandBodyRoundTrip(t *testing.T) {
	in := Command[CreateAssetCommandBody]{
		TransactionId: uuid.New(),
		CharacterId:   42,
		InventoryType: 2,
		Type:          CommandCreateAsset,
		Body:          CreateAssetCommandBody{TemplateId: 1132010, Quantity: 1},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Command[CreateAssetCommandBody]
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Type != CommandCreateAsset || out.Body.TemplateId != 1132010 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestCreateResultEventDeserializesBothShapes(t *testing.T) {
	tid := uuid.New()
	created := `{"transactionId":"` + tid.String() + `","characterId":1,"type":"CREATED","body":{"type":2,"capacity":24}}`
	failed := `{"transactionId":"` + tid.String() + `","characterId":1,"type":"CREATION_FAILED","body":{"errorCode":"CREATE_ASSET_INVENTORY_FULL","message":"full"}}`

	var ce StatusEvent[CreateResultEventBody]
	if err := json.Unmarshal([]byte(created), &ce); err != nil {
		t.Fatal(err)
	}
	if ce.Type != StatusEventTypeCreated || ce.TransactionId != tid || ce.Body.Capacity != 24 {
		t.Fatalf("created parse: %+v", ce)
	}
	var fe StatusEvent[CreateResultEventBody]
	if err := json.Unmarshal([]byte(failed), &fe); err != nil {
		t.Fatal(err)
	}
	if fe.Type != StatusEventTypeCreationFailed || fe.Body.ErrorCode != CreateAssetInventoryFull {
		t.Fatalf("failed parse: %+v", fe)
	}
}
```

- [ ] **Step 2: Run it, confirm it fails.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./kafka/message/compartment/ -v`
Expected: FAIL — undefined `CommandCreateAsset`, `CreateAssetCommandBody`, `CreateResultEventBody`, and `StatusEvent` has no `TransactionId`.

- [ ] **Step 3: Extend the message contract.** In `kafka/message/compartment/kafka.go`:

Add to the command const block (after `CommandModifyEquipment`):

```go
	CommandCreateAsset       = "CREATE_ASSET"
```

Add the create-asset command body (after `CancelReservationCommandBody`):

```go
type CreateAssetCommandBody struct {
	TemplateId      uint32    `json:"templateId"`
	Quantity        uint32    `json:"quantity"`
	Expiration      time.Time `json:"expiration"`
	OwnerId         uint32    `json:"ownerId"`
	Flag            uint16    `json:"flag"`
	Rechargeable    uint64    `json:"rechargeable"`
	UseAverageStats bool      `json:"useAverageStats,omitempty"`
}
```

Add to the status const block (after `StatusEventTypeReservationCancelled`):

```go
	StatusEventTypeCreated        = "CREATED"
	StatusEventTypeCreationFailed = "CREATION_FAILED"

	CreateAssetTemplateNotFound = "CREATE_ASSET_TEMPLATE_NOT_FOUND"
	CreateAssetInventoryFull    = "CREATE_ASSET_INVENTORY_FULL"
	CreateAssetUnknownError     = "CREATE_ASSET_UNKNOWN_ERROR"
```

Add a top-level `TransactionId` to `StatusEvent` (the `CREATED`/`CREATION_FAILED`
events carry it there, not in the body; RESERVED correlation is unaffected because
it reads `e.Body.TransactionId`):

```go
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}
```

Add the combined result body (after `ReservationCancelledEventBody`):

```go
// CreateResultEventBody is a union over the CREATED ({type,capacity}) and
// CREATION_FAILED ({errorCode,message}) event bodies so a single once-handler
// can await either; branch on StatusEvent.Type.
type CreateResultEventBody struct {
	Type      byte   `json:"type"`
	Capacity  uint32 `json:"capacity"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}
```

(`time` and `uuid` are already imported in this file.)

- [ ] **Step 4: Run the message test, confirm it passes.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./kafka/message/compartment/ -v`
Expected: PASS.

- [ ] **Step 5: Add the producer.** In `compartment/producer.go`, add `"time"` to imports, then:

```go
func requestCreateAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.CreateAssetCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandCreateAsset,
		Body: compartment.CreateAssetCommandBody{
			TemplateId: templateId,
			Quantity:   quantity,
			Expiration: expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 6: Add the processor method.** In `compartment/processor.go`, add `"errors"`, `"time"`, and `item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"` to imports, then:

```go
func (p *Processor) RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error {
	it, ok := inventory.TypeFromItemId(item2.Id(templateId))
	if !ok {
		return errors.New("invalid templateId")
	}
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(requestCreateAssetCommandProvider(transactionId, characterId, it, templateId, quantity, expiration))
}
```

- [ ] **Step 7: Add the creation once-validator.** In `kafka/once/compartment/once.go`, add:

```go
func CreationValidator(transactionId uuid.UUID) message.Validator[compartment.StatusEvent[compartment.CreateResultEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.CreateResultEventBody]) bool {
		return e.TransactionId == transactionId &&
			(e.Type == compartment.StatusEventTypeCreated || e.Type == compartment.StatusEventTypeCreationFailed)
	}
}
```

- [ ] **Step 8: Build the module, confirm it compiles.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go test ./kafka/... -v`
Expected: builds clean; message tests PASS.

- [ ] **Step 9: Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/message/compartment/ services/atlas-consumables/atlas.com/consumables/compartment/ services/atlas-consumables/atlas.com/consumables/kafka/once/compartment/once.go
git commit -m "feat(task-131): add CREATE_ASSET command + CREATED/CREATION_FAILED contract to atlas-consumables"
```

---

## Task 5: atlas-consumables — item-string data client (for worldMsg `/item`)

Mirror the existing `data/consumable` REST client for `GET /data/item-strings/{itemId}`.

**Files:**
- Create: `services/atlas-consumables/atlas.com/consumables/data/itemstring/rest.go`
- Create: `services/atlas-consumables/atlas.com/consumables/data/itemstring/requests.go`
- Create: `services/atlas-consumables/atlas.com/consumables/data/itemstring/processor.go`

**Interfaces:**
- Produces: `(*itemstring.Processor).GetName(itemId uint32) (string, error)` returning the item's display name (or error on lookup failure).

- [ ] **Step 1: Create `rest.go`.**

```go
package itemstring

type RestModel struct {
	Id   string `json:"-"`
	Name string `json:"name"`
}

func (r RestModel) GetName() string { return "item-strings" }

func (r RestModel) GetID() string { return r.Id }

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
```

- [ ] **Step 2: Create `requests.go`** (mirror `data/consumable/requests.go`; the DATA root already includes the service base):

```go
package itemstring

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "data/item-strings"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
```

- [ ] **Step 3: Create `processor.go`.**

```go
package itemstring

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

func (p *Processor) GetName(itemId uint32) (string, error) {
	rm, err := requests.Provider[RestModel, RestModel](p.l, p.ctx)(requestById(itemId), model.Identity[RestModel])()
	if err != nil {
		return "", err
	}
	return rm.Name, nil
}
```

> If `model.Identity` does not exist in the `atlas-model/model` package, define a
> local `func identity(r RestModel) (RestModel, error) { return r, nil }` and pass
> it instead. Verify by grepping `libs/atlas-model/model` for `func Identity`.

- [ ] **Step 4: Build, confirm it compiles.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./data/itemstring/...`
Expected: builds clean.

- [ ] **Step 5: Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/data/itemstring/
git commit -m "feat(task-131): add item-string data client to atlas-consumables"
```

---

## Task 6: atlas-consumables — pure reward helpers (roll, expiration, substitution)

**Files:**
- Create: `services/atlas-consumables/atlas.com/consumables/consumable/reward.go`
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/reward_test.go`

**Interfaces:**
- Produces (package `consumable`):
  - `rollReward(rewards []consumable3.RewardModel) (consumable3.RewardModel, error)` — one crypto/rand weighted pick; skips zero-prob entries; errors when total prob is 0.
  - `rewardExpiration(period int32, now time.Time) time.Time` — `now + period*minute` if `period > 0`, else zero time.
  - `substituteWorldMsg(template, characterName, itemName string) string` — replaces every `/name` and `/item` token.

  (`consumable3` is the existing import alias for `atlas-consumables/data/consumable` used in `processor.go`.)

- [ ] **Step 1: Write the failing test.** Create `reward_test.go`:

```go
package consumable

import (
	"testing"
	"time"

	consumable3 "atlas-consumables/data/consumable"
)

func rw(itemId, count, prob uint32) consumable3.RewardModel {
	return consumable3.RewardModelBuilder().SetItemId(itemId).SetCount(count).SetProb(prob).Build()
}

func TestRollRewardSingleEntry(t *testing.T) {
	got, err := rollReward([]consumable3.RewardModel{rw(2000000, 1, 100)})
	if err != nil {
		t.Fatal(err)
	}
	if got.ItemId() != 2000000 {
		t.Fatalf("got %d, want 2000000", got.ItemId())
	}
}

func TestRollRewardSkipsZeroProb(t *testing.T) {
	// Only the second entry has weight; it must always win.
	for i := 0; i < 200; i++ {
		got, err := rollReward([]consumable3.RewardModel{rw(111, 1, 0), rw(222, 1, 5)})
		if err != nil {
			t.Fatal(err)
		}
		if got.ItemId() != 222 {
			t.Fatalf("iteration %d: got %d, want 222 (zero-prob entry must never win)", i, got.ItemId())
		}
	}
}

func TestRollRewardTotalZeroErrors(t *testing.T) {
	if _, err := rollReward([]consumable3.RewardModel{rw(1, 1, 0), rw(2, 1, 0)}); err == nil {
		t.Fatal("expected error when total prob is 0")
	}
	if _, err := rollReward(nil); err == nil {
		t.Fatal("expected error for empty reward table")
	}
}

func TestRollRewardDistribution(t *testing.T) {
	// 10:90 split over 10k rolls; the rare entry should land roughly in-band.
	const n = 10000
	rare := 0
	for i := 0; i < n; i++ {
		got, err := rollReward([]consumable3.RewardModel{rw(1, 1, 100), rw(2, 1, 900)})
		if err != nil {
			t.Fatal(err)
		}
		if got.ItemId() == 1 {
			rare++
		}
	}
	// Expected ~1000; allow a wide band to avoid flakiness.
	if rare < 700 || rare > 1300 {
		t.Fatalf("rare count %d out of expected ~1000 band [700,1300]", rare)
	}
}

func TestRewardExpiration(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// 7200 minutes = exactly 5 days.
	got := rewardExpiration(7200, now)
	if !got.Equal(now.Add(5 * 24 * time.Hour)) {
		t.Fatalf("period=7200 → %v, want now+5d", got)
	}
	if !rewardExpiration(-1, now).IsZero() {
		t.Fatalf("period=-1 must yield zero time")
	}
	if !rewardExpiration(0, now).IsZero() {
		t.Fatalf("period=0 must yield zero time")
	}
}

func TestSubstituteWorldMsg(t *testing.T) {
	got := substituteWorldMsg("/name has obtained /item from a box! /name is lucky.", "Hero", "Golden Apple")
	want := "Hero has obtained Golden Apple from a box! Hero is lucky."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
```

> This test assumes `consumable3.RewardModelBuilder()` exists. If it does not, add a
> minimal Builder to `data/consumable/model.go` in this task (immutable-model rule —
> no exported struct literal): `RewardModelBuilder()` with `SetItemId/SetCount/
> SetProb/SetEffect/SetWorldMsg/SetPeriod` and `Build()`. Do NOT add a
> `*_testhelpers.go`.

- [ ] **Step 2: Run it, confirm it fails.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run 'TestRoll|TestReward|TestSubstitute' -v`
Expected: FAIL — `rollReward`/`rewardExpiration`/`substituteWorldMsg` (and possibly `RewardModelBuilder`) undefined.

- [ ] **Step 3: Add the Builder if missing.** If the compile shows `RewardModelBuilder` undefined, append to `data/consumable/model.go`:

```go
type RewardModelBuilderType struct {
	m RewardModel
}

func RewardModelBuilder() *RewardModelBuilderType { return &RewardModelBuilderType{} }

func (b *RewardModelBuilderType) SetItemId(v uint32) *RewardModelBuilderType   { b.m.itemId = v; return b }
func (b *RewardModelBuilderType) SetCount(v uint32) *RewardModelBuilderType    { b.m.count = v; return b }
func (b *RewardModelBuilderType) SetProb(v uint32) *RewardModelBuilderType     { b.m.prob = v; return b }
func (b *RewardModelBuilderType) SetEffect(v string) *RewardModelBuilderType   { b.m.effect = v; return b }
func (b *RewardModelBuilderType) SetWorldMsg(v string) *RewardModelBuilderType { b.m.worldMsg = v; return b }
func (b *RewardModelBuilderType) SetPeriod(v int32) *RewardModelBuilderType    { b.m.period = v; return b }
func (b *RewardModelBuilderType) Build() RewardModel                           { return b.m }
```

- [ ] **Step 4: Implement the helpers.** Create `consumable/reward.go`:

```go
package consumable

import (
	consumable3 "atlas-consumables/data/consumable"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"
)

// rollReward performs one clean prob-weighted pick over the reward table using a
// CSPRNG (design task-131 §2.4 — deliberate deviation from Cosmic's order-biased
// iterate-and-maybe-nothing algorithm). Zero-prob entries are skipped naturally.
// Errors when the summed weight is zero (defense in depth; callers validate first).
func rollReward(rewards []consumable3.RewardModel) (consumable3.RewardModel, error) {
	var total uint32
	for _, r := range rewards {
		total += r.Prob()
	}
	if total == 0 {
		return consumable3.RewardModel{}, errors.New("reward table has zero total probability")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(total)))
	if err != nil {
		return consumable3.RewardModel{}, err
	}
	roll := uint32(n.Int64())

	var cumulative uint32
	for _, r := range rewards {
		cumulative += r.Prob()
		if roll < cumulative {
			return r, nil
		}
	}
	// Unreachable given total>0, but return the last entry defensively.
	return rewards[len(rewards)-1], nil
}

// rewardExpiration converts a reward entry's period (MINUTES; design §2.3) to an
// absolute expiration timestamp. period <= 0 (default -1) means no expiration.
func rewardExpiration(period int32, now time.Time) time.Time {
	if period <= 0 {
		return time.Time{}
	}
	return now.Add(time.Duration(period) * time.Minute)
}

// substituteWorldMsg fills the reward worldMsg template's /name and /item tokens.
// Applied here, once, in one place (design §4.2 — Cosmic's replaceAll was a no-op).
func substituteWorldMsg(template, characterName, itemName string) string {
	s := strings.ReplaceAll(template, "/name", characterName)
	s = strings.ReplaceAll(s, "/item", itemName)
	return s
}
```

- [ ] **Step 5: Run the tests, confirm they pass.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run 'TestRoll|TestReward|TestSubstitute' -v && go test ./data/consumable/ -v`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/reward.go services/atlas-consumables/atlas.com/consumables/consumable/reward_test.go services/atlas-consumables/atlas.com/consumables/data/consumable/model.go
git commit -m "feat(task-131): weighted roll + expiration + worldMsg substitution helpers"
```

---

## Task 7: atlas-consumables — reward presentation event contract

Add the consumables-side event types the channel will consume.

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/producer.go`
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/producer_reward_test.go` (create)

**Interfaces:**
- Produces:
  - Constants `EventTypeRewardEffect = "REWARD_EFFECT"`, `EventTypeRewardWon = "REWARD_WON"`, `ErrorTypeInventoryFull = "INVENTORY_FULL"`.
  - Bodies `RewardEffectBody{BoxItemId uint32; Effect string}`, `RewardWonBody{BoxItemId uint32; ItemId uint32; Message string}`.
  - Providers `RewardEffectEventProvider(characterId character.Id, boxItemId uint32, effect string)`, `RewardWonEventProvider(characterId character.Id, boxItemId uint32, itemId uint32, message string)`, and reuse of `ErrorEventProvider(characterId, ErrorTypeInventoryFull)`.

- [ ] **Step 1: Write the failing test.** Create `producer_reward_test.go`:

```go
package consumable

import (
	"testing"

	consumable2 "atlas-consumables/kafka/message/consumable"
)

func TestRewardEventTypeConstants(t *testing.T) {
	if consumable2.EventTypeRewardEffect != "REWARD_EFFECT" {
		t.Errorf("EventTypeRewardEffect = %q", consumable2.EventTypeRewardEffect)
	}
	if consumable2.EventTypeRewardWon != "REWARD_WON" {
		t.Errorf("EventTypeRewardWon = %q", consumable2.EventTypeRewardWon)
	}
	if consumable2.ErrorTypeInventoryFull != "INVENTORY_FULL" {
		t.Errorf("ErrorTypeInventoryFull = %q", consumable2.ErrorTypeInventoryFull)
	}
}

func TestRewardEventProvidersProduceOneMessage(t *testing.T) {
	if msgs, err := RewardEffectEventProvider(7, 2022309, "Effect/BasicEff/Event1/Good")(); err != nil || len(msgs) != 1 {
		t.Fatalf("RewardEffectEventProvider: msgs=%d err=%v", len(msgs), err)
	}
	if msgs, err := RewardWonEventProvider(7, 2022309, 1132010, "Hero got Belt")(); err != nil || len(msgs) != 1 {
		t.Fatalf("RewardWonEventProvider: msgs=%d err=%v", len(msgs), err)
	}
}
```

- [ ] **Step 2: Run it, confirm it fails.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run TestReward.*Event -v`
Expected: FAIL — undefined constants / providers.

- [ ] **Step 3: Extend the event contract.** In `kafka/message/consumable/kafka.go`, add to the event const block (after `EventTypeEffectApplied`) and error-type list:

```go
	EventTypeRewardEffect  = "REWARD_EFFECT"
	EventTypeRewardWon     = "REWARD_WON"

	ErrorTypeInventoryFull = "INVENTORY_FULL"
```

Add the bodies (after `EffectAppliedBody`):

```go
type RewardEffectBody struct {
	BoxItemId uint32 `json:"boxItemId"`
	Effect    string `json:"effect"`
}

type RewardWonBody struct {
	BoxItemId uint32 `json:"boxItemId"`
	ItemId    uint32 `json:"itemId"`
	Message   string `json:"message"`
}
```

- [ ] **Step 4: Add the providers.** In `consumable/producer.go`:

```go
func RewardEffectEventProvider(characterId character.Id, boxItemId uint32, effect string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Event[consumable.RewardEffectBody]{
		CharacterId: characterId,
		Type:        consumable.EventTypeRewardEffect,
		Body:        consumable.RewardEffectBody{BoxItemId: boxItemId, Effect: effect},
	}
	return producer.SingleMessageProvider(key, value)
}

func RewardWonEventProvider(characterId character.Id, boxItemId uint32, itemId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Event[consumable.RewardWonBody]{
		CharacterId: characterId,
		Type:        consumable.EventTypeRewardWon,
		Body:        consumable.RewardWonBody{BoxItemId: boxItemId, ItemId: itemId, Message: message},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Run the test, confirm it passes.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run TestReward.*Event -v`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go services/atlas-consumables/atlas.com/consumables/consumable/producer.go services/atlas-consumables/atlas.com/consumables/consumable/producer_reward_test.go
git commit -m "feat(task-131): add REWARD_EFFECT/REWARD_WON/INVENTORY_FULL events to atlas-consumables"
```

---

## Task 8: atlas-consumables — `RequestItemReward` + `ConsumeReward` flow

The integration task. Wires reserve → roll → create-asset → commit/cancel.

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/reward.go` (small pure helpers)
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/processor_reward_test.go` (create)

**Interfaces:**
- Consumes: Task 2 (`RewardModel` getters, `Model.Rewards()`), Task 4 (`RequestCreateItem`, `once.CreationValidator`, `CreateResultEventBody`, status/error constants), Task 5 (`itemstring.Processor`), Task 6 (`rollReward`, `rewardExpiration`, `substituteWorldMsg`), Task 7 (event providers).
- Produces: `(*Processor).RequestItemReward(f field.Model, characterId uint32, itemId item2.Id, source int16) error`; pure helpers `validateRewardTable`, `grantQuantity`.

- [ ] **Step 1: Write the failing test.** Create `processor_reward_test.go`:

```go
package consumable

import (
	"testing"

	consumable3 "atlas-consumables/data/consumable"
)

// validateRewardTable is the pre-reserve guard used by RequestItemReward.
func TestValidateRewardTable(t *testing.T) {
	if err := validateRewardTable(nil); err == nil {
		t.Fatal("empty table must be rejected")
	}
	if err := validateRewardTable([]consumable3.RewardModel{rw(1, 1, 0)}); err == nil {
		t.Fatal("zero total prob must be rejected")
	}
	if err := validateRewardTable([]consumable3.RewardModel{rw(1, 1, 5)}); err != nil {
		t.Fatalf("valid table rejected: %v", err)
	}
}

// grantQuantity clamps count=0 up to 1 (design §5.4).
func TestGrantQuantity(t *testing.T) {
	if grantQuantity(0) != 1 {
		t.Fatalf("count 0 → %d, want 1", grantQuantity(0))
	}
	if grantQuantity(5) != 5 {
		t.Fatalf("count 5 → %d, want 5", grantQuantity(5))
	}
}
```

- [ ] **Step 2: Run it, confirm it fails.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run 'TestValidateRewardTable|TestGrantQuantity' -v`
Expected: FAIL — `validateRewardTable` / `grantQuantity` undefined.

- [ ] **Step 3: Add the small pure helpers to `reward.go`.**

```go
func validateRewardTable(rewards []consumable3.RewardModel) error {
	if len(rewards) == 0 {
		return errors.New("item has no reward table")
	}
	var total uint32
	for _, r := range rewards {
		total += r.Prob()
	}
	if total == 0 {
		return errors.New("reward table has zero total probability")
	}
	return nil
}

func grantQuantity(count uint32) uint32 {
	if count == 0 {
		return 1
	}
	return count
}
```

- [ ] **Step 4: Run the helper test, confirm it passes.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run 'TestValidateRewardTable|TestGrantQuantity' -v`
Expected: PASS.

- [ ] **Step 5: Implement `RequestItemReward` + `ConsumeReward`.** Append to `consumable/processor.go`. Add these imports if not present: `"time"`, `itemstring "atlas-consumables/data/itemstring"`; confirm `once "atlas-consumables/kafka/once/compartment"`, `compartment2 "atlas-consumables/kafka/message/compartment"`, `compartment "atlas-consumables/compartment"` are imported (they are, under existing aliases — reuse them).

```go
// RequestItemReward begins the reward-box flow: validate the reward table, reserve
// the box, and on RESERVED run ConsumeReward. Mirrors RequestScroll's structure
// (processor.go:515). One transactionId spans reserve → create → commit.
func (p *Processor) RequestItemReward(f field.Model, characterId uint32, itemId item2.Id, source int16) error {
	transactionId := uuid.New()

	ci, err := p.cdp.GetById(uint32(itemId))
	if err != nil {
		// Nothing reserved yet; just unstick the client.
		return p.rewardError(characterId, err)
	}
	if err = validateRewardTable(ci.Rewards()); err != nil {
		p.l.Warnf("Character [%d] requested reward-use of item [%d] with no usable reward table: %v", characterId, itemId, err)
		return p.rewardError(characterId, err)
	}

	p.l.Debugf("Creating OneTime consumer for reward transaction [%s].", transactionId.String())
	t, _ := topic.EnvProvider(p.l)(compartment2.EnvEventTopicStatus)()
	validator := once.ReservationValidator(transactionId, uint32(itemId))
	handler := compartment.Consume(ConsumeReward(transactionId, f, characterId, source, itemId, ci.Rewards()))
	if _, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler))); err != nil {
		return p.rewardError(characterId, err)
	}

	err = p.cpp.RequestReserve(transactionId, characterId, inventory2.TypeValueUse, []compartment.Reserves{{Slot: source, ItemId: uint32(itemId), Quantity: 1}})
	if err != nil {
		return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, source, err)
	}
	return nil
}

// rewardError unsticks the client without a reservation to cancel (pre-reserve
// failure path). Emits the generic consumable ERROR event.
func (p *Processor) rewardError(characterId uint32, err error) error {
	p.l.Debugf("Character [%d] reward request failed pre-reserve: [%v]", characterId, err)
	if cErr := producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(ErrorEventProvider(ts.Id(characterId), "")); cErr != nil {
		p.l.WithError(cErr).Errorf("Unable to emit reward pre-reserve error for character [%d]; client may be stuck.", characterId)
	}
	return err
}

// ConsumeReward fires on RESERVED. It rolls one reward, requests its creation,
// and registers a once-handler that commits the box on CREATED or cancels the
// reservation on CREATION_FAILED (box preserved).
func ConsumeReward(transactionId uuid.UUID, f field.Model, characterId uint32, slot int16, boxItemId item2.Id, rewards []consumable3.RewardModel) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)

			won, err := rollReward(rewards)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			l.Debugf("Character [%d] rolled reward item [%d] x[%d] (prob [%d]) from box [%d] (transaction [%s]).", characterId, won.ItemId(), won.Count(), won.Prob(), boxItemId, transactionId.String())

			// Register the creation once-handler BEFORE emitting CREATE_ASSET.
			t, _ := topic.EnvProvider(l)(compartment2.EnvEventTopicStatus)()
			cv := once.CreationValidator(transactionId)
			ch := grantReward(transactionId, f, characterId, slot, boxItemId, won)
			if _, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(cv, ch))); err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}

			expiration := rewardExpiration(won.Period(), time.Now())
			if err = p.cpp.RequestCreateItem(transactionId, characterId, won.ItemId(), grantQuantity(won.Count()), expiration); err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			return nil
		}
	}
}

// grantReward is the CREATED/CREATION_FAILED once-handler. On success it commits
// the box consume and emits presentation events; on failure it cancels the box
// reservation (box preserved) and emits the appropriate ERROR.
func grantReward(transactionId uuid.UUID, f field.Model, characterId uint32, slot int16, boxItemId item2.Id, won consumable3.RewardModel) message.Handler[compartment2.StatusEvent[compartment2.CreateResultEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment2.StatusEvent[compartment2.CreateResultEventBody]) {
		p := NewProcessor(l, ctx)

		if e.Type == compartment2.StatusEventTypeCreationFailed {
			if cErr := p.cpp.CancelItemReservation(characterId, inventory2.TypeValueUse, transactionId, slot); cErr != nil {
				l.WithError(cErr).Errorf("Unable to cancel box reservation after reward creation failure for character [%d].", characterId)
			}
			errorType := ""
			if e.Body.ErrorCode == compartment2.CreateAssetInventoryFull {
				errorType = consumable.ErrorTypeInventoryFull
			}
			if cErr := producer.ProviderImpl(l)(ctx)(consumable.EnvEventTopic)(ErrorEventProvider(ts.Id(characterId), errorType)); cErr != nil {
				l.WithError(cErr).Errorf("Unable to emit reward creation-failed error for character [%d].", characterId)
			}
			return
		}

		// CREATED: commit the box.
		if cErr := p.cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot); cErr != nil {
			l.WithError(cErr).Errorf("Reward granted but box consume failed for character [%d] (transaction [%s]); box release needs ops intervention.", characterId, transactionId.String())
		}
		l.Debugf("Character [%d] reward granted: box [%d] consumed, item [%d] created (transaction [%s]).", characterId, boxItemId, won.ItemId(), transactionId.String())

		p.emitRewardPresentation(f, characterId, boxItemId, won)
	}
}

// emitRewardPresentation emits REWARD_EFFECT (if the entry has an Effect path) and
// REWARD_WON (if the entry has a worldMsg, after /name and /item substitution).
// Presentation-only: every failure warn-logs and is swallowed (never blocks grant).
func (p *Processor) emitRewardPresentation(_ field.Model, characterId uint32, boxItemId item2.Id, won consumable3.RewardModel) {
	if won.Effect() != "" {
		if err := producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(RewardEffectEventProvider(ts.Id(characterId), uint32(boxItemId), won.Effect())); err != nil {
			p.l.WithError(err).Warnf("Unable to emit reward effect for character [%d].", characterId)
		}
	}
	if won.WorldMsg() != "" {
		name := ""
		if c, err := p.cp.GetById()(characterId); err == nil {
			name = c.Name()
		} else {
			p.l.WithError(err).Warnf("Unable to resolve name for reward announce (character [%d]); skipping /name.", characterId)
		}
		itemName, err := itemstring.NewProcessor(p.l, p.ctx).GetName(won.ItemId())
		if err != nil {
			p.l.WithError(err).Warnf("Unable to resolve item name [%d] for reward announce; skipping announce.", won.ItemId())
			return
		}
		msg := substituteWorldMsg(won.WorldMsg(), name, itemName)
		if err := producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(RewardWonEventProvider(ts.Id(characterId), uint32(boxItemId), won.ItemId(), msg)); err != nil {
			p.l.WithError(err).Warnf("Unable to emit reward-won announce for character [%d].", characterId)
		}
	}
}
```

> `p.cp` is the `*character.Processor` field already on `Processor`
> (`processor.go:54`); `p.cdp` is the consumable-data processor field
> (`processor.go:57`); `p.cpp` is the compartment processor. `character.Model`
> exposes `Name()` (`character/model.go:93`). The `field.Model` param on
> `emitRewardPresentation` is currently unused (`_`) — the map/session context is
> resolved channel-side; keep the param so the signature stays stable if a
> consumables-side field lookup is later needed. If `go vet` flags the unused
> import of `field` from this addition, note `field` is already imported in
> processor.go.

- [ ] **Step 6: Run tests + build, confirm green.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go test ./consumable/ -race -v`
Expected: builds clean; all consumable tests PASS.

- [ ] **Step 7: Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go services/atlas-consumables/atlas.com/consumables/consumable/reward.go services/atlas-consumables/atlas.com/consumables/consumable/processor_reward_test.go
git commit -m "feat(task-131): RequestItemReward + ConsumeReward reserve/roll/grant/commit flow"
```

---

## Task 9: atlas-consumables — command consumer arm for `REQUEST_ITEM_REWARD`

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go` (command type + body)
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go`

**Interfaces:**
- Consumes: Task 8 (`RequestItemReward`).
- Produces: command `CommandRequestItemReward = "REQUEST_ITEM_REWARD"`, body `RequestItemRewardBody{Source slot.Position; ItemId item.Id}`; consumer arm `handleRequestItemReward`.

- [ ] **Step 1: Add the command type + body.** In `kafka/message/consumable/kafka.go`, add to the command const block:

```go
	CommandRequestItemReward      = "REQUEST_ITEM_REWARD"
```

Add the body (after `RequestItemConsumeBody`):

```go
type RequestItemRewardBody struct {
	Source slot.Position `json:"source"`
	ItemId item.Id       `json:"itemId"`
}
```

- [ ] **Step 2: Register + implement the consumer arm.** In `kafka/consumer/consumable/consumer.go`, add to `InitHandlers` (after the `handleRequestItemConsume` registration):

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestItemReward))); err != nil {
				return err
			}
```

Add the handler (mirrors `handleRequestItemConsume`; builds a `field.Model` from the command envelope):

```go
func handleRequestItemReward(l logrus.FieldLogger, ctx context.Context, c consumable2.Command[consumable2.RequestItemRewardBody]) {
	if c.Type != consumable2.CommandRequestItemReward {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	err := consumable.NewProcessor(l, ctx).RequestItemReward(f, uint32(c.CharacterId), c.Body.ItemId, int16(c.Body.Source))
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to use reward box in slot [%d].", c.CharacterId, c.Body.Source)
	}
}
```

`field` is already imported in this consumer (used by `handleCancelConsumableEffect`).

- [ ] **Step 3: Build + test, confirm green.**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go vet ./... && go test -race ./...`
Expected: clean.

- [ ] **Step 4: Commit.**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go
git commit -m "feat(task-131): consume REQUEST_ITEM_REWARD command in atlas-consumables"
```

---

## Task 10: atlas-channel — serverbound handler + `REQUEST_ITEM_REWARD` emit

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go` (command type + body)
- Modify: `services/atlas-channel/atlas.com/channel/consumable/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/consumable/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (handlerMap)

**Interfaces:**
- Consumes: Task 3 (`invsb.CharacterItemUseLotteryHandle`, `LotteryItemUse`).
- Produces: channel command `CommandRequestItemReward = "REQUEST_ITEM_REWARD"`, body `RequestItemRewardBody{Source slot.Position; ItemId item.Id}`; `(*Processor).RequestItemReward(...)`; handler `CharacterItemUseLotteryHandleFunc`.

- [ ] **Step 1: Add the channel-side command contract.** In `kafka/message/consumable/kafka.go`, add to the command const block:

```go
	CommandRequestItemReward  = "REQUEST_ITEM_REWARD"
```

Add the body (after `RequestItemConsumeBody`):

```go
type RequestItemRewardBody struct {
	Source slot.Position `json:"source"`
	ItemId item.Id       `json:"itemId"`
}
```

- [ ] **Step 2: Add the producer.** In `consumable/producer.go`:

```go
func RequestItemRewardCommandProvider(f field.Model, characterId character.Id, source slot.Position, itemId item.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestItemRewardBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        consumable.CommandRequestItemReward,
		Body: consumable.RequestItemRewardBody{
			Source: source,
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 3: Add the processor method.** In `consumable/processor.go`:

```go
func (p *Processor) RequestItemReward(f field.Model, characterId character.Id, itemId item.Id, source slot.Position) error {
	p.l.Debugf("Character [%d] using reward box [%d] from slot [%d].", characterId, itemId, source)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemRewardCommandProvider(f, characterId, source, itemId))
}
```

- [ ] **Step 4: Add the handler.** In `socket/handler/character_item_use.go`:

```go
func CharacterItemUseLotteryHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := inventory2.NewLotteryItemUse()
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = consumable.NewProcessor(l, ctx).RequestItemReward(s.Field(), character.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Source()))
	}
}
```

- [ ] **Step 5: Register the handler.** In `main.go`, in the `handlerMap` block near line 891 (next to the other item-use handlers), add:

```go
	handlerMap[invsb.CharacterItemUseLotteryHandle] = handler.CharacterItemUseLotteryHandleFunc
```

- [ ] **Step 6: Build, confirm it compiles.**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 7: Commit.**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go services/atlas-channel/atlas.com/channel/consumable/ services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-131): route LotteryItemUse handler + REQUEST_ITEM_REWARD emit in atlas-channel"
```

---

## Task 11: atlas-channel — presentation consumer arms (inventory-full, effect, world-won)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go` (mirror event types + bodies)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go`

**Interfaces:**
- Consumes: Task 7 event JSON (`REWARD_EFFECT`, `REWARD_WON`, ERROR/`INVENTORY_FULL`).
- Produces: three consumer arms — inventory-full status message, self+foreign lottery effect, channel-wide world announce.

- [ ] **Step 1: Mirror the event contract on the channel side.** In `kafka/message/consumable/kafka.go`, add to the event const block:

```go
	EventTypeRewardEffect  = "REWARD_EFFECT"
	EventTypeRewardWon     = "REWARD_WON"

	ErrorTypeInventoryFull = "INVENTORY_FULL"
```

Add the bodies (after `ScrollBody`):

```go
type RewardEffectBody struct {
	BoxItemId uint32 `json:"boxItemId"`
	Effect    string `json:"effect"`
}

type RewardWonBody struct {
	BoxItemId uint32 `json:"boxItemId"`
	ItemId    uint32 `json:"itemId"`
	Message   string `json:"message"`
}
```

- [ ] **Step 2: Add the inventory-full arm.** In `kafka/consumer/consumable/consumer.go`, extend `handleErrorConsumableEvent` — insert an arm BEFORE the generic StatChanged unstick (after the `ErrorTypePetCannotConsume` block). The **writer const** `CharacterStatusMessageWriter` is in `.../character/clientbound` (aliased `charcb`); the **body factory** `CharacterStatusMessageDropPickUpInventoryFullBody()` is in `.../character` (aliased `charpkt`). Match the aliases used in `kafka/consumer/compartment/consumer.go`; add the imports if the file does not already have them:

```go
		if e.Body.Error == consumable2.ErrorTypeInventoryFull {
			err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
				if aerr := session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageDropPickUpInventoryFullBody())(s); aerr != nil {
					return aerr
				}
				return session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
			})
			if err != nil {
				l.WithError(err).Errorf("Unable to process inventory-full event for character [%d].", e.CharacterId)
			}
			return
		}
```

> The existing file imports `charpkt` as `.../character/clientbound`. There is a
> naming clash: the writer const lives in `.../character/clientbound` and the body
> factory in `.../character`. Resolve by importing `.../character/clientbound` as
> `charcb` and `.../character` as `charpkt` (the convention in
> `compartment/consumer.go`), updating the two existing `charpkt.` references in
> this file (the scroll handler's `charpkt.CharacterItemUpgradeWriter` /
> `charpkt.NewItemUpgrade` — those are in `.../character/clientbound`, so they
> become `charcb.`). Grep + fix in this step; `go build` confirms.

- [ ] **Step 3: Add the reward-effect + reward-won handlers.** In the same file:

```go
func handleRewardEffectConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.RewardEffectBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.RewardEffectBody]) {
		if e.Type != consumable2.EventTypeRewardEffect {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
			// Self: the user sees the lottery-use effect.
			if aerr := session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterLotteryUseEffectBody(e.Body.BoxItemId, true, e.Body.Effect))(s); aerr != nil {
				l.WithError(aerr).Warnf("Unable to send lottery effect to character [%d].", e.CharacterId)
			}
			// Others in the map see the foreign effect.
			return _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterLotteryUseEffectForeignBody(s.CharacterId(), e.Body.BoxItemId, true, e.Body.Effect)))
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to process reward-effect event for character [%d].", e.CharacterId)
		}
	}
}

func handleRewardWonConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.RewardWonBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.RewardWonBody]) {
		if e.Type != consumable2.EventTypeRewardWon {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		sessions, err := session.NewProcessor(l, ctx).AllInChannelProvider(sc.WorldId(), sc.ChannelId())
		if err != nil {
			l.WithError(err).Error("Unable to get sessions for reward-won broadcast.")
			return
		}
		announceOp := session.Announce(l)(ctx)(wp)(chatcb.WorldMessageWriter)(writer.WorldMessageBlueTextBody("", "", e.Body.Message))
		for _, s := range sessions {
			if aerr := announceOp(s); aerr != nil {
				l.WithError(aerr).Warnf("Unable to send reward-won announce to session.")
			}
		}
	}
}
```

> `writer.WorldMessageBlueTextBody(medal, characterName, message)` resolves the
> `BLUE_TEXT` mode from the tenant `operations` table and encodes the blue-text
> packet with `itemId=0`. The item name is already substituted into `message`
> (Task 8), so the trailing packet itemId is cosmetic and 0 is fine — this matches
> how `system_message/consumer.go:131` uses the same body. Import the channel
> `writer` package and `chatcb "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"`
> for the `WorldMessageWriter` const; `_map "atlas-channel/map"` for the foreign
> fan-out (match existing aliases in sibling consumers).

- [ ] **Step 4: Register the two new handlers.** In `InitHandlers`, after the scroll handler registration:

```go
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRewardEffectConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRewardWonConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

- [ ] **Step 5: Build + vet + test, confirm green.**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./kafka/consumer/consumable/...`
Expected: clean. Resolve any alias/const mismatches by grepping the sibling consumers named in `context.md` (`monsterbook`, `gachapon`, `compartment`, `system_message`).

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go
git commit -m "feat(task-131): channel consumer arms for reward effect/announce/inventory-full"
```

---

## Task 12: Packet matrix promotion (verify-packet playbook)

Promote the `LOTTERY_ITEM_USE_REQUEST` STATUS.md row for v83/v84/v87/v95. Do NOT
hand-edit STATUS.md or invent IDA addresses; use the established tooling.

**Files:**
- Modify (via tooling): `docs/packets/audits/STATUS.md` and the evidence records under `docs/packets/audits/`
- The fixture test from Task 3 already carries the v83/v95 `packet-audit:verify` markers.

- [ ] **Step 1: Run the verify-packet playbook.** Invoke the `verify-packet` skill (or dispatch the `packet-verifier` agent) once per version for `inventory/serverbound/LotteryItemUse` × {gms_v83, gms_v84, gms_v87, gms_v95}, following `docs/packets/audits/VERIFYING_A_PACKET.md`. v83 (ida 0xa1249f) and v95 (ida 0x9d6c50) are IDA-verified this task; v84/v87 use the registry-lineage / no-IDB convention per the playbook. The tool regenerates the matrix and pins evidence.

- [ ] **Step 2: Confirm promotion.** Verify STATUS.md shows the `LOTTERY_ITEM_USE_REQUEST` cells for v83/v84/v87/v95 promoted (no longer ❌). Do NOT touch v92/jms cells (out of scope).

- [ ] **Step 3: Commit** whatever artifacts the playbook produced (fixture already committed in Task 3; here it's the matrix + evidence):

```bash
git add docs/packets/audits/
git commit -m "docs(task-131): promote LOTTERY_ITEM_USE_REQUEST matrix cells (v83/v84/v87/v95)"
```

---

## Task 13: atlas-configurations — seed-template handler entries (v83/v84/v87/v95)

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`

**Interfaces:** Adds one `socket.handlers` entry per template registering the new serverbound handler with a mandatory validator.

- [ ] **Step 1: Add the handler entry to each in-scope template.** In each file's `socket.handlers` array (next to the existing `CharacterItemUseHandle` entry), add — using the per-version opcode:

v83 (`0x070`) and v84 (`0x070`):

```json
      {
        "opCode": "0x70",
        "validator": "LoggedInValidator",
        "handler": "CharacterItemUseLotteryHandle"
      },
```

v87 (`0x073`):

```json
      {
        "opCode": "0x73",
        "validator": "LoggedInValidator",
        "handler": "CharacterItemUseLotteryHandle"
      },
```

v95 (`0x07C`):

```json
      {
        "opCode": "0x7C",
        "validator": "LoggedInValidator",
        "handler": "CharacterItemUseLotteryHandle"
      },
```

Match the exact hex casing/format the file already uses for neighboring entries
(the codebase uses forms like `"0x48"`; use `"0x70"`, `"0x73"`, `"0x7C"`). The
`validator` field is **mandatory** — a validator-less entry is silently dropped
(`libs/atlas-opcodes/producer.go:44-51`; only `NoOpValidator`/`LoggedInValidator`
are registered).

- [ ] **Step 2: Validate JSON.** Run for each file:

Run: `for v in 83 84 87 95; do python3 -m json.tool services/atlas-configurations/seed-data/templates/template_gms_${v}_1.json > /dev/null && echo "gms_${v} ok"; done`
Expected: `ok` for all four (no JSON syntax errors from the inserted comma/object).

- [ ] **Step 3: Confirm the handler string is unique per file** (no duplicate opcode collision):

Run: `for v in 83 84 87 95; do echo "== gms_$v =="; grep -c "CharacterItemUseLotteryHandle" services/atlas-configurations/seed-data/templates/template_gms_${v}_1.json; done`
Expected: `1` per file.

- [ ] **Step 4: Commit.**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_83_1.json services/atlas-configurations/seed-data/templates/template_gms_84_1.json services/atlas-configurations/seed-data/templates/template_gms_87_1.json services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(task-131): register CharacterItemUseLotteryHandle in v83/v84/v87/v95 seed templates"
```

---

## Task 14: Rollout documentation (data re-publish + live-tenant config patch)

**Files:**
- Create: `docs/tasks/task-131-random-reward-items/rollout.md`

**Interfaces:** none (documentation).

- [ ] **Step 1: Write the rollout runbook.** Create `rollout.md` documenting the two rollout steps (design §5.7, §5.8):

  1. **Data fields** — atlas-data consumables are stored JSON documents; existing
     tenants lack `Effect`/`worldMsg`/`period` until re-ingestion. Absent fields
     degrade gracefully (`""`/`-1`). If a canonical baseline is in play:
     `POST /api/data/process` (re-ingest canonical) → `POST /api/data/baseline/publish`
     → per live tenant `POST /api/data/baseline/restore` (tenants on the
     canonical-fallback read path pick the fields up from the publish alone).
  2. **Handler opcode** — seed templates apply only at tenant creation. For each
     existing v83/v84/v87/v95 tenant, PATCH the tenant's socket config to add the
     `CharacterItemUseLotteryHandle` entry (per-version opcode from Task 13) with
     `LoggedInValidator`, then restart atlas-channel (projection does not
     hot-reload handlers — project memory: new-opcodes-not-in-live-tenant-config).
  3. **v92/jms** — explicitly out of scope; do not patch (see `context.md`).

- [ ] **Step 2: Commit.**

```bash
git add docs/tasks/task-131-random-reward-items/rollout.md
git commit -m "docs(task-131): reward-items data re-publish + live-tenant config rollout runbook"
```

---

## Task 15: Full verification gate + code review

**Files:** none (verification only).

- [ ] **Step 1: Per-module Go checks.** For each changed module, run from the worktree root:

```bash
for m in libs/atlas-packet services/atlas-data/atlas.com/data services/atlas-consumables/atlas.com/consumables services/atlas-channel/atlas.com/channel; do
  echo "== $m =="; (cd "$m" && go build ./... && go vet ./... && go test -race ./...) || echo "FAIL $m";
done
```
Expected: every module builds, vets, and tests clean.

- [ ] **Step 2: redis key guard.** From repo root:

Run: `GOWORK=off tools/redis-key-guard.sh`
Expected: clean (this task adds no raw keyed go-redis calls).

- [ ] **Step 3: docker bake the touched services.** From the worktree root:

```bash
docker buildx bake atlas-data
docker buildx bake atlas-consumables
docker buildx bake atlas-channel
docker buildx bake atlas-configurations
```
Expected: all four images build (the new `data/itemstring` package is under an
already-copied module, so no Dockerfile change is expected — the bake confirms it).

- [ ] **Step 4: Code review.** Run `superpowers:requesting-code-review` (dispatches
  `plan-adherence-reviewer` + `backend-guidelines-reviewer` since only Go changed).
  Address findings; re-run the gate. Do this BEFORE opening a PR (CLAUDE.md).

- [ ] **Step 5: Final acceptance sweep against the PRD.** Confirm each §10 AC is met
  for v83/v84/v87/v95 (v92/jms documented out-of-scope), then the branch is
  PR-ready.

---

## Self-review notes (traceability to design/PRD)

- **PRD 4.1 / design §2.1, §5.1** — Task 3 (codec, IDA-verified body). **4.10 /
  §5.1** — Task 12 (matrix).
- **PRD 4.2 / design §5.2, §5.4** — Tasks 8–10 (routing: channel handler → command →
  consumables arm → RequestItemReward).
- **PRD 4.3 / design §5.4** — Task 8 (validation before mutation; reserve).
- **PRD 4.4 / design §2.4, §5.4** — Task 6 (`rollReward`, crypto/rand, single pick).
- **PRD 4.5 / design §2.3, §5.4** — Tasks 6+8 (`rewardExpiration`, grant via
  `RequestCreateItem`, fit-check = grant attempt).
- **PRD 4.6 / design §2.2, §4.2, §5.5** — Tasks 7+11 (REWARD_EFFECT self+foreign,
  REWARD_WON channel-wide, substitution applied once).
- **PRD 4.7 / design §4.1** — Tasks 4+8 (reserve + create-before-commit + cancel
  compensation, one transactionId).
- **PRD 4.8 / design §5.6** — Tasks 1+2 (atlas-data parse + consumables mirror).
- **PRD 4.9 / design §5.8** — Task 13 (seed templates) + Task 14 (live rollout).
- **PRD 4.3 inventory-full / design §4.3** — Task 11 (`INVENTORY_FULL` →
  `DropPickUpInventoryFull` + StatChanged, box preserved via CREATION_FAILED cancel
  in Task 8).
- **Scope deviation:** v92 dropped (skeleton template + no IDB); jms out per §2.6.
  Recorded in `context.md` and Global Constraints; revisit before PR if the owner
  wants v92.
```
