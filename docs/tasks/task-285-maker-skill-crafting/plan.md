# Maker Skill / Item Maker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the MapleStory Item Maker across five surfaces — recipe ingestion in `atlas-data`, a new `atlas-maker` domain service, `MAKER_SKILL`/`MAKER_RESULT` codecs in `libs/atlas-packet`, a handler and writer in `atlas-channel`, and a compensable craft saga in `atlas-saga-orchestrator`.

**Architecture:** `atlas-data` ingests `Etc.wz/ItemMake.img.xml` into the shared `documents` table under type `ITEM_MAKE`. `atlas-maker` is the only component that knows what a recipe *means*: it reads recipes, character level/mesos, Maker skill level, inventory snapshots and quest state from five upstreams, computes an exact consumption plan of concrete `(compartment, slot, quantity)` tuples, and emits a craft saga. `atlas-channel` decodes `MAKER_SKILL`, calls `atlas-maker`, and writes `MAKER_RESULT` only after the saga reaches a terminal state — so the create-arm body can enumerate the consumption that actually happened. One new saga action, `AwardCraftedAsset`, exists because no current creation payload can express "an equip with `tuc` upgrade slots and explicit reagent-adjusted stats".

**Tech Stack:** Go 1.27 across nine modules (`libs/atlas-packet`, `libs/atlas-saga`, `libs/atlas-constants`, `tools/packet-audit`, `services/atlas-data/atlas.com/data`, `services/atlas-inventory/atlas.com/inventory`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-channel/atlas.com/channel`, and the new `services/atlas-maker/atlas.com/maker`); GORM + Postgres; Kafka; JSON:API via `libs/atlas-rest`; logrus; `crypto/rand` for weighted draws.

**Spec:** `docs/tasks/task-285-maker-skill-crafting/design.md` (PRD at `docs/tasks/task-285-maker-skill-crafting/prd.md`, wire evidence at `docs/tasks/task-285-maker-skill-crafting/evidence-maker-skill-v72-v79.md`)

## Global Constraints

- **The design's §2 corrections are binding and override the PRD.** In particular: `MAKER_RESULT` is result-code-prefixed (`Encode4 nResult`, then `Encode4 nMode`), not mode-prefixed (C-1); the wire layout is version-invariant so **no `MajorAtLeast` gate is written unless Task 6 finds a divergence** (C-2); `MAKER_SKILL` encodes its mode **once** (C-3); `atlas-data` gets **no new table and no migration** (C-4); `reqQuest` is ingested *and* enforced (C-5); the top-level group digit is persisted (C-6).
- Applicable versions are exactly eight: `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`. `gms_v48` and `gms_v61` are genuinely `n-a` for both ops and MUST stay `⬜`. `template_gms_12_1.json`, `template_gms_48_1.json`, `template_gms_61_1.json` are never touched.
- Opcodes and dispatcher mode bytes are resolved from config, **never hard-coded**. `dispatcher-lint` INV-2 fails on any `mode:\s*0x` literal in a dispatcher struct constructor and on any `func(_ byte)` in a body-function file; INV-3 fails on any exported `*Body(` function that lets the caller pick the operation.
- **No wire change to an already-verified version.** `packet-audit matrix --check` is a hard gate: any non-zero exit fails CI.
- Never trust client-supplied quantities, slots, item ids, or reagent lists. Every one is re-validated server-side against the character's actual inventory (PRD §8 Security).
- A rejected craft MUST leave materials, mesos and equips byte-identical to their pre-request state (FR-3.7), and MUST still produce a `MAKER_RESULT` so the client UI unlocks (FR-5.2).
- Every step of every craft saga MUST use a **compensable** action. `DestroyAllAssets` is explicitly excluded because it is not compensable. Material consumption uses `DestroyAssetFromSlot` (never `DestroyAsset`, which resolves a template to the *first* matching slot only and under-consumes a multi-stack material).
- `randomReward` sampling is server-side only, drawn from `crypto/rand`, never `math/rand`. The weight table is never sent to the client.
- Collection GET routes paginate (`page[number]`/`page[size]`) — never a bare `GetAll` + `MarshalResponse[[]...]`. See `docs/rest-pagination.md`.
- Background goroutines are spawned via `routine.Go(l, ctx, fn)` from `libs/atlas-routine` — never a bare `go` statement (DOM-26, enforced by `tools/goroutine-guard.sh`).
- Check `libs/atlas-constants/` before defining any new domain type, alias, or numeric constant. `item.Id`, `inventory.Type`, `inventory.TypeFromItemId`, and the four Maker skill identities already exist (see Task 21).
- Never land a placeholder comment, a stubbed handler, or an unimplemented status response.
- Use the project's Builder pattern for test setup; no `*_testhelpers.go` test-only constructors.
- Preserve existing line endings; never normalize CRLF→LF as a side effect.
- Interfaces and their mocks change together. Adding a method to a `Processor` interface without updating its `mock/processor.go` breaks the build.

---

## Task 1: Registry correction — `MAKER_SKILL` on `gms_v72` and `gms_v79`

FR-4.0. `status.json` row 324 records `MAKER_SKILL` as `n-a` on `gms_v72`/`gms_v79`. That is a *discovery* gap, not a client-capability gap: the op carries `provenance: csv-import` on v83/v84/v87, and the source CSV has no v72/v79 column, so the op never entered those two registry files at all. Both clients ship the full `CUIItemMaker` UI and send the request.

This task is standalone and mergeable on its own.

### Files

- `docs/packets/registry/gms_v72.yaml` — add the `MAKER_SKILL` serverbound entry
- `docs/packets/registry/gms_v79.yaml` — add the `MAKER_SKILL` serverbound entry
- `docs/packets/audits/status.json` — regenerated, not hand-edited
- `docs/packets/audits/STATUS.md` — regenerated, not hand-edited

Patterns to copy: `docs/packets/registry/gms_v72.yaml:2634-2641` (the `LOTTERY_ITEM_USE_REQUEST` entry — same `ida-discovered` + `note:` shape, and the immediate predecessor by opcode).

Commands run from the repo root (`tools/packet-audit` is its own module but is reached via `go run ./tools/packet-audit` through `go.work`).

### Interfaces

- Consumes: nothing.
- Produces: the `MAKER_SKILL` × `gms_v72` and `MAKER_SKILL` × `gms_v79` matrix cells, flipped from `⬜ n-a` to `❌ incomplete`. Task 7 promotes them to `✅`.

### Verified facts this task rests on

`ida.address` in these files is stored as a **decimal integer**, not a hex string (`gms_v72.yaml:2640` reads `address: 9488698`). The two addresses from the evidence doc convert as:

| Version | fname | opcode | `0x` address | decimal to write |
|---|---|---|---|---|
| `gms_v72` | `CUIItemMaker::RequestItemMake` | 112 (`0x70`) | `0x760cc3` | `7736515` |
| `gms_v79` | `CUIItemMaker::RequestItemMake` | 111 (`0x6F`) | `0x795dc3` | `7953859` |

Insertion points, both between `LOTTERY_ITEM_USE_REQUEST` and `SUE_CHARACTER` so the file stays opcode-ordered:

| File | insert immediately before | current line |
|---|---|---|
| `docs/packets/registry/gms_v72.yaml` | `- op: SUE_CHARACTER` | 2642 |
| `docs/packets/registry/gms_v79.yaml` | `- op: SUE_CHARACTER` | 3138 |

The opcode neighbourhood corroborates both values one-for-one against v83:

| | v72 | v79 | v83 |
|---|---|---|---|
| `LOTTERY_ITEM_USE_REQUEST` | 111 | 110 | 112 |
| `MAKER_SKILL` | **112** | **111** | 113 |
| `SUE_CHARACTER` | 113 | 112 | 114 |

- [ ] **Step 1: Confirm the starting state**

Run from the repo root:

```
grep -n "op: MAKER_SKILL" docs/packets/registry/gms_v83.yaml docs/packets/registry/gms_v72.yaml docs/packets/registry/gms_v79.yaml
```

Expected: **exactly one** hit, from `gms_v83.yaml` (around line 2539). `gms_v72.yaml` and `gms_v79.yaml` must not appear — their absence is the defect this task fixes. Including v83 in the command proves the pattern itself matches, so an empty result cannot be misread as "already correct". If v72 or v79 appears, STOP: the task has already been done.

- [ ] **Step 2: Add the `gms_v72` entry**

Insert immediately before `- op: SUE_CHARACTER` in `docs/packets/registry/gms_v72.yaml`:

```yaml
- op: MAKER_SKILL
  direction: serverbound
  opcode: 112
  fname: CUIItemMaker::RequestItemMake
  provenance: ida-discovered
  ida:
    address: 7736515
  note: 'task-285: v72 COutPacket(0x70) = 112; ?RequestItemMake@CUIItemMaker@@IAEHXZ @0x760cc3 (GMS_v72.1_U_DEVM.exe.i64), sent via CClientSocket::SendPacket. Previously absent because MAKER_SKILL carries provenance csv-import on v83/v84/v87 and the source CSV has no v72 column, so the matrix rendered registry-absence as n-a. Opcode lands in the existing gap between LOTTERY_ITEM_USE_REQUEST (111) and SUE_CHARACTER (113), matching the v83 neighbourhood one-for-one.'
```

- [ ] **Step 3: Add the `gms_v79` entry**

Insert immediately before `- op: SUE_CHARACTER` in `docs/packets/registry/gms_v79.yaml`:

```yaml
- op: MAKER_SKILL
  direction: serverbound
  opcode: 111
  fname: CUIItemMaker::RequestItemMake
  provenance: ida-discovered
  ida:
    address: 7953859
  note: 'task-285: v79 COutPacket(111) = 0x6F; ?RequestItemMake@CUIItemMaker@@IAEHXZ @0x795dc3 (GMS_v79_1_DEVM.exe.i64), sent via CClientSocket::SendPacket. Previously absent for the same csv-import reason as v72. Opcode lands in the existing gap between LOTTERY_ITEM_USE_REQUEST (110) and SUE_CHARACTER (112).'
```

- [ ] **Step 4: Regenerate the matrix**

Run from the repo root:

```
go run ./tools/packet-audit matrix
```

This rewrites `docs/packets/audits/status.json` and `docs/packets/audits/STATUS.md`. Do not hand-edit either file.

- [ ] **Step 5: Assert the two cells moved and nothing else regressed**

Run from the repo root:

```
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
```

All four MUST exit 0.

Then confirm the specific cells. `MAKER_SKILL` is `status.json` row 324; read the row and assert:

- `gms_v72` and `gms_v79` are now `incomplete` (`❌`) with opcodes `112` and `111` — **not** `n-a`, and **not** `-1`.
- `gms_v48` and `gms_v61` are still `n-a` with opcode `-1`. Neither IDB contains `CItemMakerInfo` or `RequestItemMake`; the only `CUIItemMaker`-named symbol in either is a mislabelled Guild BBS handler. These two cells MUST NOT move.
- The six already-registered versions are unchanged at 113/113/116/124/125/108 (v83/v84/v87/v92/v95/jms_v185).

- [ ] **Step 6: Commit**

```bash
git add docs/packets/registry/gms_v72.yaml docs/packets/registry/gms_v79.yaml docs/packets/audits/status.json docs/packets/audits/STATUS.md
git commit -m "fix(packets): register MAKER_SKILL on gms_v72 and gms_v79"
```

---

## Task 2: `atlas-data` `itemmake` — RestModel and registry

Per C-4 the `itemmake` domain adds **no migration and no new table**. Every WZ domain in `atlas-data` persists into the shared `documents` table (`services/atlas-data/atlas.com/data/document/entity.go:15-26`), upserting on `(tenant_id, type, document_id)`. Idempotency (FR-1.6) is therefore structural and free.

This task establishes the shape the reader (Task 3) fills and the resource (Task 4) serves.

### Files

- `services/atlas-data/atlas.com/data/itemmake/rest.go` — new file; `RestModel` and its three child models
- `services/atlas-data/atlas.com/data/itemmake/registry.go` — new file; the `sync.Once` document registry singleton
- `services/atlas-data/atlas.com/data/itemmake/rest_test.go` — new file; round-trip and ordering tests

Module root for `go build`/`go test`: `services/atlas-data/atlas.com/data`.

Patterns to copy: `services/atlas-data/atlas.com/data/commodity/rest.go:1-33` (flat `RestModel` + `GetName`/`GetID`/`SetID` via `strconv`) and `services/atlas-data/atlas.com/data/commodity/registry.go:13-18` (the `sync.Once`-guarded package-level `document.Registry`). Every domain writes its own registry singleton — there is no shared generic getter.

### Interfaces

- Consumes: `document.Registry` from `services/atlas-data/atlas.com/data/document`.
- Produces, for Tasks 3 and 4:
  - `itemmake.RestModel` with fields `Id uint32`, `Group uint32`, `ReqLevel uint32`, `ReqSkillLevel uint32`, `ItemNum uint32`, `Tuc uint32`, `Meso uint32`, `Catalyst uint32`, `ReqItem uint32`, `ReqEquip uint32`, `Recipe []MaterialRestModel`, `RandomReward []RewardRestModel`, `ReqQuest []QuestReqRestModel`
  - `itemmake.MaterialRestModel{ItemId uint32; Count uint32}`
  - `itemmake.RewardRestModel{ItemId uint32; ItemNum uint32; Prob uint32}`
  - `itemmake.QuestReqRestModel{QuestId uint32; State uint32}`
  - `func itemmake.GetModelRegistry() *document.Registry[string, RestModel]`
  - `RestModel.GetName() string` returning `"itemMakes"`

### Why `Group` exists (C-6)

The archive's top level is keyed by the created item's leading digit (`0`, `1`, `2`, `4`, `8`, `16`). Mode 3's leftover → crystal lookup must be scoped to group `0`, otherwise an arbitrary recipe that happens to list the leftover as a material would match. Discarding the digit at ingestion makes that lookup impossible downstream, so it is persisted.

### Why `ReqQuest` exists (C-5)

A field-name sweep of the reference `Etc.wz/ItemMake.img.xml` yields exactly: `catalyst`, `count`, `item`, `itemNum`, `meso`, `prob`, `randomReward`, `recipe`, `reqEquip`, `reqItem`, `reqLevel`, `reqQuest`, `reqSkillLevel`, `tuc`. FR-1.2 omits `reqQuest`. It is an `imgdir` mapping quest id → required state, not a scalar:

```xml
<imgdir name="reqQuest"><int name="21614" value="3"/></imgdir>
```

Occurrence counts in the reference archive: `catalyst` 772, `reqItem` 14, `reqEquip` 2, `reqQuest` 2. Two recipes are affected. It is ingested *and* enforced (Task 21) — ingesting a field then ignoring it is the "documented gap" the repo forbids when the prerequisite is producible, and it is: `atlas-quests` already answers per-character quest state.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-data/atlas.com/data/itemmake/rest_test.go`, `package itemmake`. Setup shape copied from `services/atlas-data/atlas.com/data/commodity/rest_test.go`.

`TestRestModelGetName` — assert `RestModel{}.GetName() == "itemMakes"`.

`TestRestModelIdRoundTrip` — table-driven, `cases := []struct{ name string; id uint32 }`:

| subtest name | id | expected `GetID()` |
|---|---|---|
| `group zero crystal` | `4260000` | `"4260000"` |
| `equip recipe` | `1082002` | `"1082002"` |
| `zero` | `0` | `"0"` |

For each: build `RestModel{Id: tc.id}`, assert `GetID()` equals the expected string, then `var out RestModel; out.SetID(tc.expected)` and assert `out.Id == tc.id`.

`TestRestModelJSONPreservesListOrder` — the ordering regression FR-1.3/FR-1.4 would otherwise hide. Build:

```go
in := RestModel{
    Id:    1082002,
    Group: 1,
    Recipe: []MaterialRestModel{
        {ItemId: 4011001, Count: 5},
        {ItemId: 4011002, Count: 3},
        {ItemId: 4021007, Count: 1},
    },
    RandomReward: []RewardRestModel{
        {ItemId: 4260000, ItemNum: 1, Prob: 70},
        {ItemId: 4260001, ItemNum: 1, Prob: 25},
        {ItemId: 4260002, ItemNum: 1, Prob: 5},
    },
    ReqQuest: []QuestReqRestModel{{QuestId: 21614, State: 3}},
}
```

`json.Marshal` then `json.Unmarshal` into a fresh `RestModel`, and assert element-by-element that all three slices come back in the **same order** with the same values. Assert `len(out.Recipe) == 3`, `out.Recipe[0].ItemId == 4011001`, `out.Recipe[2].ItemId == 4021007`, and likewise for `RandomReward` (`out.RandomReward[0].Prob == 70`, `out.RandomReward[2].Prob == 5`).

`TestRestModelAbsentListsAreEmptyNotNilOnRoundTrip` — unmarshal the literal JSON `{"group":0,"reqLevel":0,"reqSkillLevel":0,"itemNum":0,"tuc":0,"meso":0,"catalyst":0,"reqItem":0,"reqEquip":0}` into a `RestModel` and assert every scalar is `0` and `len(Recipe) == 0`, `len(RandomReward) == 0`, `len(ReqQuest) == 0`. This pins the PRD's "0 when absent" convention (FR-1.5).

`TestGetModelRegistryIsSingleton` — call `GetModelRegistry()` twice and assert both calls return the identical pointer.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-data/atlas.com/data`:

```
go test ./itemmake/... -count=1
```

Expected: FAIL — no non-test Go files / undefined `RestModel`, `MaterialRestModel`, `RewardRestModel`, `QuestReqRestModel`, `GetModelRegistry`.

- [ ] **Step 3: Write `rest.go`**

```go
package itemmake

import "strconv"

type RestModel struct {
	Id            uint32              `json:"-"`
	Group         uint32              `json:"group"`
	ReqLevel      uint32              `json:"reqLevel"`
	ReqSkillLevel uint32              `json:"reqSkillLevel"`
	ItemNum       uint32              `json:"itemNum"`
	Tuc           uint32              `json:"tuc"`
	Meso          uint32              `json:"meso"`
	Catalyst      uint32              `json:"catalyst"`
	ReqItem       uint32              `json:"reqItem"`
	ReqEquip      uint32              `json:"reqEquip"`
	Recipe        []MaterialRestModel `json:"recipe"`
	RandomReward  []RewardRestModel   `json:"randomReward"`
	ReqQuest      []QuestReqRestModel `json:"reqQuest"`
}

// MaterialRestModel is one ordered (item, count) entry of a recipe's `recipe`
// child list (FR-1.3). Order is document order and is load-bearing.
type MaterialRestModel struct {
	ItemId uint32 `json:"itemId"`
	Count  uint32 `json:"count"`
}

// RewardRestModel is one ordered (item, itemNum, prob) entry of a recipe's
// optional `randomReward` child list (FR-1.4). Prob is a relative weight, not a
// percentage; it is never sent to the client.
type RewardRestModel struct {
	ItemId  uint32 `json:"itemId"`
	ItemNum uint32 `json:"itemNum"`
	Prob    uint32 `json:"prob"`
}

// QuestReqRestModel is one (questId, state) entry of a recipe's optional
// `reqQuest` child list (design C-5). Only two recipes in the reference archive
// carry one.
type QuestReqRestModel struct {
	QuestId uint32 `json:"questId"`
	State   uint32 `json:"state"`
}

func (r RestModel) GetName() string { return "itemMakes" }

func (r RestModel) GetID() string { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(id string) error {
	v, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}
```

Match `commodity/rest.go`'s exact `GetID`/`SetID` idiom rather than inventing a variant; if that file uses `strconv.FormatUint`/`ParseUint`, mirror it instead.

- [ ] **Step 4: Write `registry.go`**

Mirror `commodity/registry.go:13-18` exactly, substituting the type:

```go
package itemmake

import (
	"sync"

	"atlas-data/document"
)

var registryOnce sync.Once
var modelRegistry *document.Registry[string, RestModel]

func GetModelRegistry() *document.Registry[string, RestModel] {
	registryOnce.Do(func() {
		modelRegistry = document.NewRegistry[string, RestModel]()
	})
	return modelRegistry
}
```

Read `commodity/registry.go` first and copy its import path and constructor call verbatim — the module is named `atlas-data`, and the registry constructor's exact name and signature must match what that file uses.

- [ ] **Step 5: Run the tests to verify they pass**

Run from `services/atlas-data/atlas.com/data`:

```
go test ./itemmake/... -count=1
```

Expected: PASS, all five tests.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/itemmake/
git commit -m "feat(data): add itemmake RestModel and document registry"
```

---

## Task 3: `atlas-data` `itemmake` — the archive reader

FR-1.1 through FR-1.5. `Etc.wz/ItemMake.img.xml` is a two-level tree: six top-level `imgdir`s named `0`, `1`, `2`, `4`, `8`, `16` (the created item's leading digit), each containing entries keyed by the zero-padded 8-digit created-item id.

### Files

- `services/atlas-data/atlas.com/data/itemmake/reader.go` — new file; `Read`
- `services/atlas-data/atlas.com/data/itemmake/reader_test.go` — new file; fixture-driven tests

Module root for `go build`/`go test`: `services/atlas-data/atlas.com/data`.

Patterns to copy:
- `services/atlas-data/atlas.com/data/commodity/reader.go:1-35` — the `Read(l) func(model.Provider[xml.Node]) model.Provider[[]RestModel]` signature and the flat per-entry loop.
- `services/atlas-data/atlas.com/data/quest/reader.go:132-229` (`readRequirements`) — the **ordered child-list idiom**: `n.ChildByName("recipe")` then range `recipeNode.ChildNodes`, appending in document order and never sorting.

### Interfaces

- Consumes: `itemmake.RestModel` and its child models (Task 2); `xml.Node` from `services/atlas-data/atlas.com/data/xml`.
- Produces, for Task 4: `func itemmake.Read(l logrus.FieldLogger) func(model.Provider[xml.Node]) model.Provider[[]RestModel]` — match `commodity/reader.go`'s exact signature.

### The `xml.Node` API this reader uses

All defined in `services/atlas-data/atlas.com/data/xml/model.go`:

| Member | Signature | Line |
|---|---|---|
| `Name` | `string` (the `name` attribute) | struct field |
| `ChildNodes` | `[]Node` (preserves XML document order) | struct field |
| `ChildByName` | `func (n *Node) ChildByName(name string) (*Node, error)` | `:21` |
| `IntegerNodes` | `[]IntegerNode` | struct field |
| `GetIntegerWithDefault` | `func (n *Node) GetIntegerWithDefault(name string, def int32) int32` | `:124` |

`GetIntegerWithDefault(name, 0)` is what satisfies FR-1.5's default-don't-fail rule for every scalar.

### Two structural rules

**Group digit.** The reader records each top-level node's `Name` as the entry's `Group`. `strconv.Atoi` on `Name` gives the digit; a top-level node whose name is not numeric is logged and skipped.

**Malformed entries are skipped, not fatal.** A per-entry `continue`, never an error return. `RegisterFileData` (`services/atlas-data/atlas.com/data/data/processor.go:302-307`) discards the `RegisterFunc` error entirely:

```go
func (p *ProcessorImpl) RegisterFileData(rootDir string, wzFileName string, rf RegisterFunc) Worker {
	return func() error {
		rf(filepath.Join(rootDir, wzFileName))
		return nil
	}
}
```

A returned error would be silently swallowed and the operator would see nothing, so the log line is the only signal that matters.

**`reqQuest` keys are quest ids, not field names.** Unlike `recipe`/`randomReward` whose children are indexed `imgdir`s containing `item`/`count` ints, `reqQuest` is a single `imgdir` whose `IntegerNodes` are `name=<questId> value=<state>`. It is read by ranging `reqQuestNode.IntegerNodes` and `strconv.Atoi`-ing each node's `Name`, not via `GetIntegerWithDefault`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-data/atlas.com/data/itemmake/reader_test.go`, `package itemmake`.

**Fixture mechanism — read this before writing.** `commodity/reader_test.go` does **not** build `xml.Node` struct literals. It declares a raw XML string constant and parses it at test time through `xml.FromByteArrayProvider([]byte(testXML))` (`services/atlas-data/atlas.com/data/xml/reader.go:51`), feeding the resulting provider into `Read(l)(...)`. Copy that mechanism; the test body at `commodity/reader_test.go:2627-2666` (`TestReader`) is the template. (The design's phrase "synthetic `xml.Node` fixtures built inline in Go" means an inline XML-text literal, not Go struct construction.)

Declare one shared fixture covering every case:

```go
const testXML = `<imgdir name="ItemMake.img">
  <imgdir name="0">
    <imgdir name="04260000">
      <int name="reqLevel" value="0"/>
      <int name="reqSkillLevel" value="0"/>
      <int name="itemNum" value="1"/>
      <int name="tuc" value="0"/>
      <int name="meso" value="0"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4000000"/>
          <int name="count" value="1"/>
        </imgdir>
      </imgdir>
      <imgdir name="randomReward">
        <imgdir name="0">
          <int name="item" value="4260000"/>
          <int name="itemNum" value="1"/>
          <int name="prob" value="70"/>
        </imgdir>
        <imgdir name="1">
          <int name="item" value="4260001"/>
          <int name="itemNum" value="1"/>
          <int name="prob" value="25"/>
        </imgdir>
        <imgdir name="2">
          <int name="item" value="4260002"/>
          <int name="itemNum" value="1"/>
          <int name="prob" value="5"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="1">
    <imgdir name="01082002">
      <int name="reqLevel" value="30"/>
      <int name="reqSkillLevel" value="2"/>
      <int name="itemNum" value="1"/>
      <int name="tuc" value="7"/>
      <int name="meso" value="1200"/>
      <int name="catalyst" value="4130000"/>
      <int name="reqItem" value="4000021"/>
      <int name="reqEquip" value="1002419"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4011001"/>
          <int name="count" value="5"/>
        </imgdir>
        <imgdir name="1">
          <int name="item" value="4011002"/>
          <int name="count" value="3"/>
        </imgdir>
        <imgdir name="2">
          <int name="item" value="4021007"/>
          <int name="count" value="1"/>
        </imgdir>
      </imgdir>
      <imgdir name="reqQuest">
        <int name="21614" value="3"/>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="2">
    <imgdir name="02020000">
      <int name="reqLevel" value="10"/>
      <int name="itemNum" value="3"/>
      <int name="meso" value="500"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4000001"/>
          <int name="count" value="2"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="4">
    <imgdir name="04030000">
      <int name="reqLevel" value="15"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="800"/>
    </imgdir>
  </imgdir>
  <imgdir name="8">
    <imgdir name="08000000">
      <int name="reqLevel" value="20"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="900"/>
    </imgdir>
  </imgdir>
  <imgdir name="16">
    <imgdir name="16000000">
      <int name="reqLevel" value="25"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="1000"/>
    </imgdir>
  </imgdir>
  <imgdir name="1">
    <imgdir name="NOT_A_NUMBER">
      <int name="reqLevel" value="1"/>
    </imgdir>
  </imgdir>
</imgdir>`
```

`TestReadCoversEveryTopLevelGroup` — parse the fixture, collect results into a `map[uint32]RestModel` keyed by `Id` (use `model.CollectToMap` as `commodity/reader_test.go:2627-2666` does). Assert exactly these `(id → group)` pairs are present:

| id | group |
|---|---|
| `4260000` | `0` |
| `1082002` | `1` |
| `2020000` | `2` |
| `4030000` | `4` |
| `8000000` | `8` |
| `16000000` | `16` |

This is the FR-1.1 acceptance criterion and the C-6 `Group` requirement in one assertion.

`TestReadScalars` — assert on the `1082002` entry, field by field: `ReqLevel == 30`, `ReqSkillLevel == 2`, `ItemNum == 1`, `Tuc == 7`, `Meso == 1200`, `Catalyst == 4130000`, `ReqItem == 4000021`, `ReqEquip == 1002419`.

`TestReadAbsentScalarsDefaultToZero` — assert on the `2020000` entry (which declares no `tuc`, `catalyst`, `reqItem`, `reqEquip`, `reqSkillLevel`): `Tuc == 0`, `Catalyst == 0`, `ReqItem == 0`, `ReqEquip == 0`, `ReqSkillLevel == 0`, and that the entry is nonetheless present. FR-1.5.

`TestReadRecipeOrder` — assert `1082002`'s `Recipe` is exactly, in this order:

| index | ItemId | Count |
|---|---|---|
| 0 | `4011001` | `5` |
| 1 | `4011002` | `3` |
| 2 | `4021007` | `1` |

`TestReadRandomRewardOrder` — assert `4260000`'s `RandomReward` is exactly, in this order:

| index | ItemId | ItemNum | Prob |
|---|---|---|---|
| 0 | `4260000` | `1` | `70` |
| 1 | `4260001` | `1` | `25` |
| 2 | `4260002` | `1` | `5` |

`TestReadRandomRewardAbsentIsEmpty` — assert `len(m[1082002].RandomReward) == 0`.

`TestReadReqQuest` — assert `len(m[1082002].ReqQuest) == 1`, `m[1082002].ReqQuest[0].QuestId == 21614`, `m[1082002].ReqQuest[0].State == 3`. Assert `len(m[4260000].ReqQuest) == 0`.

`TestReadRecipeAbsentIsEmpty` — assert `len(m[4030000].Recipe) == 0` (the group-`4` entry declares no `recipe`).

`TestReadSkipsMalformedEntryWithoutAborting` — assert `4260000`, `1082002`, `2020000`, `4030000`, `8000000` and `16000000` are ALL present despite the trailing `NOT_A_NUMBER` entry, and that no entry with a zero id was produced. The malformed entry must not abort the archive (FR-1.5).

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-data/atlas.com/data`:

```
go test ./itemmake/... -count=1 -run TestRead
```

Expected: FAIL — undefined `Read`.

- [ ] **Step 3: Write `reader.go`**

Structure, following `commodity/reader.go`'s signature and `quest/reader.go:132-229`'s ordered-child idiom:

1. `Read(l logrus.FieldLogger) func(model.Provider[xml.Node]) model.Provider[[]RestModel]`.
2. Range the root node's `ChildNodes` — these are the six group directories. `strconv.Atoi(groupNode.Name)`; on error, `l.Warnf` and `continue`.
3. Range each group node's `ChildNodes` — these are the recipe entries. `strconv.Atoi(entryNode.Name)`; on error, `l.Warnf` naming the group and the bad key, and `continue`. (`Atoi` handles the zero-padded 8-digit keys; `04260000` parses to `4260000`.)
4. Build the `RestModel`: `Id` from the key, `Group` from the group digit, every scalar via `entryNode.GetIntegerWithDefault("<name>", 0)` cast to `uint32`.
5. `recipe`: `entryNode.ChildByName("recipe")`; on error leave the slice empty. Otherwise range `recipeNode.ChildNodes` **in order**, appending `MaterialRestModel{ItemId: uint32(c.GetIntegerWithDefault("item", 0)), Count: uint32(c.GetIntegerWithDefault("count", 0))}`. Do not sort.
6. `randomReward`: same shape, appending `RewardRestModel{ItemId, ItemNum, Prob}` from `item`/`itemNum`/`prob`. Do not sort.
7. `reqQuest`: `entryNode.ChildByName("reqQuest")`; range `reqQuestNode.IntegerNodes`, `strconv.Atoi(in.Name)` for the quest id (log-and-skip on error) and the node's value for the state. Do not sort.

Take the loop-variable capture and provider-construction details from `commodity/reader.go` verbatim rather than improvising — that file is 35 lines and is the whole contract.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-data/atlas.com/data`:

```
go test ./itemmake/... -count=1
```

Expected: PASS, all nine `TestRead*` tests plus Task 2's five.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/itemmake/reader.go services/atlas-data/atlas.com/data/itemmake/reader_test.go
git commit -m "feat(data): read Etc.wz/ItemMake.img.xml into itemmake models"
```

---

## Task 4: `atlas-data` `itemmake` — processor, storage, REST resource

FR-1.6 and PRD §5's two `atlas-data` endpoints.

### Files

- `services/atlas-data/atlas.com/data/itemmake/processor.go` — new file; `Processor`, `NewProcessor`, `NewStorage`, `Register`, `RegisterItemMake`
- `services/atlas-data/atlas.com/data/itemmake/mock/processor.go` — new file; `ProcessorMock`
- `services/atlas-data/atlas.com/data/itemmake/resource.go` — new file; `InitResource` and the handlers
- `services/atlas-data/atlas.com/data/itemmake/resource_test.go` — new file
- `services/atlas-data/atlas.com/data/main.go` — add one `AddRouteInitializer` line

Module root for `go build`/`go test`: `services/atlas-data/atlas.com/data`.

Patterns to copy:
- `services/atlas-data/atlas.com/data/commodity/processor.go:1-61` — the whole file, including the `NewStorage` wrapper and the per-item `s.Add` loop in `Register` (note the comment at `:39-45` explaining why there is no outer transaction — task-076).
- `services/atlas-data/atlas.com/data/commodity/resource.go` lines 18-31 — `InitResource` route registration; the same shape appears at lines 17-27 of `services/atlas-data/atlas.com/data/etc/resource.go`.
- `services/atlas-data/atlas.com/data/commodity/mock/processor.go` lines 1-29 — the mock shape.
- `services/atlas-data/atlas.com/data/commodity/resource_test.go` and `resource_pagination_test.go` — handler test setup.

### Interfaces

- Consumes: `itemmake.RestModel`, `itemmake.GetModelRegistry` (Task 2), `itemmake.Read` (Task 3), `document.NewStorage`.
- Produces, for Task 5: `func itemmake.NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` with a method `RegisterItemMake(filePath string) error` matching `data.RegisterFunc` (`services/atlas-data/atlas.com/data/data/processor.go:226`, `RegisterFunc func(filePath string) error`).

### The storage call

`document.NewStorage` is declared at `services/atlas-data/atlas.com/data/document/storage.go:20`:

```go
func NewStorage[I string, M Identifier[I]](l logrus.FieldLogger, db *gorm.DB, r *Registry[I, M], docType string) *Storage[I, M]
```

`commodity/processor.go:35-37` calls it as:

```go
func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "COMMODITY")
}
```

The `itemmake` equivalent uses document type `"ITEM_MAKE"`, keyed by the created-item id. Writes upsert on `(tenant_id, type, document_id)` via `clause.OnConflict` (`services/atlas-data/atlas.com/data/document/db_storage.go:120-157`), which is what makes FR-1.6 structural.

### Routes

| Method | Path | Handler |
|---|---|---|
| `GET` | `/data/item-makes` | paginated list |
| `GET` | `/data/item-makes/{itemId}` | one recipe by created-item id |

The list route **must paginate** (`page[number]`/`page[size]`) — never a bare `GetAll` + `MarshalResponse[[]RestModel]`. See `docs/rest-pagination.md` and copy `commodity/resource.go`'s list handler, which already does this.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-data/atlas.com/data/itemmake/resource_test.go`, `package itemmake`. Setup — router construction, tenant context, in-memory DB — copied from `services/atlas-data/atlas.com/data/commodity/resource_test.go`.

`TestGetItemMakeById` — seed the storage with the `1082002` model from Task 3's fixture, `GET /data/item-makes/1082002`, assert HTTP 200 and that the JSON:API `data.attributes` round-trips every scalar (`group` 1, `reqLevel` 30, `reqSkillLevel` 2, `itemNum` 1, `tuc` 7, `meso` 1200, `catalyst` 4130000, `reqItem` 4000021, `reqEquip` 1002419) **and both list orders**: `recipe` is `[{4011001,5},{4011002,3},{4021007,1}]` and `reqQuest` is `[{21614,3}]`, element by element.

`TestGetItemMakeByIdFromEachGroup` — seed one entry from each of the six groups (ids `4260000`, `1082002`, `2020000`, `4030000`, `8000000`, `16000000`) and `GET` each, asserting HTTP 200 and the correct `group` attribute. This is the PRD §10 acceptance criterion "at least one entry from each top-level directory".

`TestGetItemMakeByIdNotFound` — `GET /data/item-makes/9999999` against empty storage, assert HTTP 404.

`TestGetItemMakeRandomRewardOrderSurvivesREST` — seed `4260000`, `GET` it, assert `randomReward` comes back as `[{4260000,1,70},{4260001,1,25},{4260002,1,5}]` in that exact order. FR-1.4.

`TestListItemMakesPaginates` — seed 25 entries, `GET /data/item-makes?page[number]=2&page[size]=10`, assert exactly 10 returned and that the JSON:API pagination links are present.

`TestRegisterIsIdempotent` — FR-1.6. Register the same parsed model set twice through the processor, then assert the stored document count is unchanged after the second pass and that `GET /data/item-makes/1082002` still returns exactly one resource with identical attributes. Structural under C-4, but pinned so a future storage change cannot regress it unnoticed.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-data/atlas.com/data`:

```
go test ./itemmake/... -count=1
```

Expected: FAIL — undefined `InitResource`, `NewProcessor`, `NewStorage`.

- [ ] **Step 3: Write `processor.go` and `mock/processor.go`**

Copy `commodity/processor.go` wholesale and substitute: the document type string `"ITEM_MAKE"`, the model type `RestModel`, the reader call `Read`, and the exported registration method name `RegisterItemMake`. Keep the per-item `s.Add` loop and the no-outer-transaction comment — the reason it exists (task-076) applies identically here.

`RegisterItemMake` wires `Read` to `xml.FromPathProvider` exactly as `commodity/processor.go`'s `RegisterCommodity` does, and has signature `func (p *ProcessorImpl) RegisterItemMake(filePath string) error` so it satisfies `data.RegisterFunc`.

Write `mock/processor.go` in the same pass — a nil-checked function field per interface method, mirroring `commodity/mock/processor.go`. The interface and its mock change together.

- [ ] **Step 4: Write `resource.go` and wire `main.go`**

`InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer`, registering the two routes above. Copy `commodity/resource.go:18-31` for the registration block and its list handler for the pagination handling.

In `services/atlas-data/atlas.com/data/main.go`, add one line to the `AddRouteInitializer` chain immediately after the `commodity`/`cashpackage` pair (currently `commodity` at line 193, `cashpackage` at 194, `etc` at 195):

```go
		AddRouteInitializer(itemmake.InitResource(db)(GetServer())).
```

Add the `itemmake` import. Confirm the exact line numbers at edit time — the chain has drifted by a line or two since the design was written.

- [ ] **Step 5: Run the tests to verify they pass**

Run from `services/atlas-data/atlas.com/data`:

```
go build ./...
go test ./itemmake/... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/itemmake/ services/atlas-data/atlas.com/data/main.go
git commit -m "feat(data): expose item-makes REST resource backed by document storage"
```

---

## Task 5: `atlas-data` — `ITEM_MAKE` worker registration

FR-1.1's "established per-archive worker pattern".

### Files

- `services/atlas-data/atlas.com/data/data/processor.go` — add the `WorkerItemMake` const, extend the `Workers` slice, add the dispatch branch
- `services/atlas-data/atlas.com/data/data/processor_test.go` — assert the worker is enumerated and dispatches
- `services/atlas-data/atlas.com/data/README.md` — document the new worker and the two endpoints

Module root for `go build`/`go test`: `services/atlas-data/atlas.com/data`.

Patterns to copy: `services/atlas-data/atlas.com/data/data/processor.go:193-194` (the `WorkerCharacterCreation` branch — the same single-`RegisterFileData` shape, since `MakeCharInfo.img.xml` is also a one-file `Etc.wz` archive).

### Interfaces

- Consumes: `itemmake.NewProcessor(...).RegisterItemMake` (Task 4).
- Produces: the `ITEM_MAKE` worker, runnable via the existing worker-instruction path.

### Exact current state

Verified in the tree right now:

- Const block: **lines 42-59**, last entry `WorkerMobSkill = "MOB_SKILL"` on line 59, closing paren on line 60.
- `Workers` slice: **line 62**, currently ending `..., WorkerFace, WorkerHair, WorkerMobSkill}`.
- The branch to mirror, **lines 193-194**:

```go
} else if name == WorkerCharacterCreation {
    err = p.RegisterFileData(path, filepath.Join("Etc.wz", "MakeCharInfo.img.xml"), templates.NewProcessor(p.l, p.ctx, p.db).RegisterCharacterTemplate)()
}
```

### Why its own worker, not chained onto `WorkerCommodity`

`WorkerCommodity` already chains `CashPackage` behind `Commodity`. Chaining makes the second archive's ingestion conditional on the first's success for no reason, and `ItemMake.img` is unrelated to `Commodity.img`. It gets its own worker name.

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-data/atlas.com/data/data/processor_test.go`, following the file's existing setup shape.

`TestWorkersIncludesItemMake` — assert `WorkerItemMake == "ITEM_MAKE"` and that `Workers` contains it exactly once. Also assert `len(Workers) == 18` so a future addition that forgets the slice is caught.

`TestStartWorkerDispatchesItemMake` — call `StartWorker(WorkerItemMake, <temp path>)` and assert it does not return the "unknown worker" error the default branch produces. Read the existing dispatch chain's final `else` to get that error's exact text and assert against it by identity, not by substring guess.

- [ ] **Step 2: Run the test to verify it fails**

Run from `services/atlas-data/atlas.com/data`:

```
go test ./data/... -count=1 -run ItemMake
```

Expected: FAIL — undefined `WorkerItemMake`.

- [ ] **Step 3: Add the const, the slice entry, and the branch**

Append to the const block (after `WorkerMobSkill` on line 59):

```go
	WorkerItemMake          = "ITEM_MAKE"
```

Append `WorkerItemMake` to the `Workers` slice on line 62.

Add the dispatch branch alongside the other `Etc.wz` single-file workers:

```go
} else if name == WorkerItemMake {
    err = p.RegisterFileData(path, filepath.Join("Etc.wz", "ItemMake.img.xml"), itemmake.NewProcessor(p.l, p.ctx, p.db).RegisterItemMake)()
}
```

Add the `itemmake` import.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-data/atlas.com/data`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Update the README**

In `services/atlas-data/atlas.com/data/README.md`, add `ITEM_MAKE` to the worker list and add the two `/data/item-makes` routes to the REST endpoints table, matching the table's existing column layout.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/data/ services/atlas-data/atlas.com/data/README.md
git commit -m "feat(data): register ITEM_MAKE worker for Etc.wz/ItemMake.img.xml"
```

---

## Task 6: Per-version wire derivation for both ops (all eight versions)

This is design risk **R-1** discharged, and it is the prerequisite for Tasks 7-9. The design decompiled only four IDBs (`gms_v72` @ `0x86a152`, `gms_v83` @ `0x95dad3`, `gms_v95` @ `0x9102f0`, `jms_v185` @ `0xa29527`) and concluded from those four that the layout is version-invariant (C-2). Four unsampled versions remain: `gms_v79`, `gms_v84`, `gms_v87`, `gms_v92`.

**Dispatch this task with `model: opus`.** It is derivation-heavy; every other task in this plan is Sonnet work.

### Files

- `docs/tasks/task-285-maker-skill-crafting/wire-derivation.md` — new file; the per-version evidence record
- `docs/tasks/task-285-maker-skill-crafting/coverage-manifest.yaml` — new file; the packet task's declared scope
- `docs/packets/registry/gms_v72.yaml` — correct the `MAKER_RESULT` `ida.address`
- `docs/packets/registry/gms_v79.yaml` — correct the `MAKER_RESULT` `ida.address`

Read-only references: `docs/packets/PROCESS.md`, `docs/packets/audits/VERIFYING_A_PACKET.md`, `docs/tasks/task-285-maker-skill-crafting/evidence-maker-skill-v72-v79.md`.

### Interfaces

- Consumes: Task 1's registry entries.
- Produces, for Tasks 7-9: a per-version, per-arm read order with a cited address for each of the eight versions, and a verdict on whether any `MajorAtLeast` gate is required.

### Why the registry addresses need correcting

`docs/packets/registry/gms_v72.yaml`'s `MAKER_RESULT` entry carries `ida.address: 8772770`, and the **same** decimal value appears verbatim on the immediately preceding (`PLAY_MINI_GAME_SOUND`) and following (`KOREAN_EVENT`) unrelated entries. That is a bulk-import placeholder, not a per-op verified address, and it disagrees with the design's own C-1 citation of `CUserLocal::OnMakerResult @ 0x86a152` for `gms_v72`. `gms_v79`'s entry (`ida.address: 9080139`) is suspect for the same reason. Correct both to the addresses this task derives. Do **not** touch the other six versions' `MAKER_RESULT` addresses unless this task's own decompilation contradicts them; record any such contradiction in `wire-derivation.md` and correct it there too.

- [ ] **Step 1: Write the coverage manifest**

`docs/packets/PROCESS.md` requires a packet task to declare its scope up front, so `packet-completeness-critic` can diff it against the branch's actual git + matrix delta. Create `docs/tasks/task-285-maker-skill-crafting/coverage-manifest.yaml`:

```yaml
# coverage-manifest
ops:
  - MAKER_SKILL
  - MAKER_RESULT
versions:
  - gms_v72
  - gms_v79
  - gms_v83
  - gms_v84
  - gms_v87
  - gms_v92
  - gms_v95
  - jms_v185
fields:
  - "MAKER_SKILL: no version gate expected (design C-2); confirm per version"
  - "MAKER_RESULT: result-code prefix precedes the mode (design C-1)"
out_of_scope:
  - model/asset
```

- [ ] **Step 2: Derive `CUserLocal::OnMakerResult` on all eight versions**

Follow `docs/packets/PROCESS.md` and `docs/packets/audits/VERIFYING_A_PACKET.md`. For each of the eight versions, decompile `CUserLocal::OnMakerResult` and record the exact `CInPacket::Decode*` read order per arm.

The four already-decompiled versions and their addresses, to be re-confirmed and carried forward:

| Version | IDB session | Address |
|---|---|---|
| `gms_v72` | `99e435d8` | `0x86a152` |
| `gms_v83` | `754107bf` | `0x95dad3` |
| `gms_v95` | `ecc757f4` | `0x9102f0` |
| `jms_v185` | `a977912e` | `0xa29527` |

Derive `gms_v79`, `gms_v84`, `gms_v87`, `gms_v92` from scratch.

The shape all four sampled versions share, which the remaining four are being tested against:

```c
v53 = CInPacket::Decode4(a2);     // nResult
if ( v53 <= 1 )
{
  v3 = CInPacket::Decode4(v2);    // nMode
  switch ( v3 ) { case 1: case 2: ... case 3: ... case 4: ... }
}
```

Record, per version, the read order of each of the four arms plus the bodyless `nResult > 1` form. Note that function-size differences between versions (`0x660` v72, `0x6df` v83, `0x8a0` v95, `0x633` jms) come from chat-log string handling and `CUIStatusBar::ChatLogAdd` inlining, **not** from packet fields — so a size difference alone is not evidence of a wire divergence.

- [ ] **Step 3: Derive `CUIItemMaker::RequestItemMake` on all eight versions**

Same procedure. `gms_v72` @ `0x760cc3` and `gms_v79` @ `0x795dc3` are pinned by the evidence doc; `gms_v95` @ `0x7d58d0` was confirmed during design. Derive the other five.

Confirm C-3 specifically on every version: there is **one** mode encode, inside the switch, not a pre-switch encode followed by a per-arm echo. The v95 decompilation reads:

```c
COutPacket::COutPacket(&oPacket, 125);
switch ( this->m_nRecipeClass ) {
  case 1u: case 2u:
    COutPacket::Encode4(&oPacket, m_nRecipeClass);   // the ONLY mode encode
    COutPacket::Encode4(&oPacket, this->m_nTargetItem);
    ...
```

The double-encode rendering in `evidence-maker-skill-v72-v79.md` and in PRD FR-4.3 is a transcription artefact of showing each arm's first field. If any version genuinely double-encodes, that is a real divergence — record it.

- [ ] **Step 4: Write `wire-derivation.md`**

For each op × version, record: the IDB session id, the function address, the ordered `Decode`/`Encode` list per arm, and an explicit `IDENTICAL` or `DIVERGENT` verdict against the reference shape. Include a summary table:

| op | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms185 |
|---|---|---|---|---|---|---|---|---|
| `MAKER_SKILL` | | | | | | | | |
| `MAKER_RESULT` | | | | | | | | |

State the C-2 verdict in one sentence: either "no version gate is required for either op" or an explicit enumeration of the diverging fields and their boundary versions.

- [ ] **Step 5: If a divergence was found, record the gate**

Only if Step 4 found one. A divergence is gated with the `MajorAtLeast` idiom, **never** a raw `> N` comparison, and it must be registered in `docs/packets/gates.yaml` — `packet-audit gate-check --check` is a blocking CI gate that asserts every gated divergence has a verified byte fixture on **both** adjacent straddling versions. Add the entry and note in `wire-derivation.md` which two versions the boundary fixtures must cover.

If no divergence was found, state that explicitly and make no `gates.yaml` change.

- [ ] **Step 6: Correct the `MAKER_RESULT` registry addresses**

Update `ida.address` on the `MAKER_RESULT` entry in `docs/packets/registry/gms_v72.yaml` (line ~1407) and `docs/packets/registry/gms_v79.yaml` (line ~1424) to the decimal form of the addresses derived in Step 2, and append a `note:` citing this task. Leave `opcode`, `direction`, `fname` and `provenance` untouched.

Then run from the repo root:

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
```

All must exit 0.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-285-maker-skill-crafting/wire-derivation.md docs/tasks/task-285-maker-skill-crafting/coverage-manifest.yaml docs/packets/registry/gms_v72.yaml docs/packets/registry/gms_v79.yaml docs/packets/audits/
git commit -m "docs(packets): derive MAKER_SKILL and MAKER_RESULT wire layout across all eight versions"
```

---

## Task 7: `MAKER_SKILL` serverbound codec

FR-4.1. Decodes the request on all eight versions.

### Files

- `libs/atlas-packet/character/serverbound/maker_skill.go` — new file; the codec
- `libs/atlas-packet/character/serverbound/maker_skill_test.go` — new file; byte fixtures
- `tools/packet-audit/cmd/run.go` — add the `candidatesFromFName` case

Module roots: `libs/atlas-packet` for the codec and its test; `tools/packet-audit` for `run.go`.

Patterns to copy: `libs/atlas-packet/character/clientbound/show_combo.go` for the struct/constructor/`Encode`/`Decode` plumbing and the `packet-audit:fname` marker placement (it is the other `CUserLocal::*` op in this package). `libs/atlas-packet/report/clientbound/sue_character_result_test.go:1-50` for the byte-fixture test shape.

**This is new-pattern work, not a copy.** No existing serverbound codec in `libs/atlas-packet` decodes its own dispatch mode off the wire and then reads a *different field set per mode value* inside one `Decode`. The nearest neighbours all fall short: `libs/atlas-packet/rps/serverbound/operation.go` is mode-only with no branching; `libs/atlas-packet/character/clientbound/attack.go` branches on a discriminant that is **caller-injected via the constructor**, not self-decoded; `libs/atlas-packet/cash/serverbound/item_use.go` branches on a version gate. Budget accordingly.

### Interfaces

- Consumes: Task 6's derived read order.
- Produces, for Task 25: `serverbound.MakerSkill` with accessors `Mode() uint32`, `TargetItemId() uint32`, `UseCatalyst() bool`, `GemItemIds() []uint32`, `LeftoverItemId() uint32`, `ItemId() uint32`, `InventoryType() uint32`, `SlotPos() uint32`, and `func serverbound.NewMakerSkill(...)`.

### Corrected layout (design §5.1, C-3)

The mode is encoded **once**. There is no pre-switch encode.

```
Decode4  nMode

mode 1|2   Decode4 nTargetItemID
           Decode1 bUseCatalyst
           Decode4 nGemCount
           nGemCount × Decode4 nGemItemID

mode 3     Decode4 nLeftoverItemID

mode 4     Decode4 nItemID
           Decode4 nInventoryType
           Decode4 nSlotPos
```

`nInventoryType` in mode 4 is a real field — the reference server's own read order carries it, commented there as "probably inventory type". It is decoded and then **re-validated**, never trusted (PRD §8 Security).

### The mode is NOT a dispatcher family

`docs/packets/DISPATCHER_FAMILY.md` scopes families to **clientbound** ops, and `TestFamilyCapServerboundSkipped` in `tools/packet-audit/cmd/family_cap_test.go` guards this. Adding a serverbound fname to `families.yaml` would demote already-verified cells: task-206 did exactly that and demoted 17. Serverbound mode dispatch instead becomes the handler's `options.operations` routing table in each seed template (Task 25).

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-packet/character/serverbound/maker_skill_test.go`, `package serverbound`. Use `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"` and the `pt.Variants[i]` index guard idiom from `libs/atlas-packet/report/clientbound/sue_character_result_test.go:37-44` — assert `v.Name` before use so an index drift fails loudly.

Variant indices, confirmed in `libs/atlas-packet/test/context.go:19-40`:

| version | index | `v.Name` |
|---|---|---|
| `gms_v83` | `1` | `"GMS v83"` |
| `gms_v87` | `2` | `"GMS v87"` |
| `gms_v95` | `3` | `"GMS v95"` |
| `jms_v185` | `4` | `"JMS v185"` |
| `gms_v84` | `5` | `"GMS v84"` |
| `gms_v72` | `9` | `"GMS v72"` |
| `gms_v79` | `10` | `"GMS v79"` |
| `gms_v92` | `11` | `"GMS v92"` |

`TestMakerSkillDecodeCreate` — decode this byte sequence and assert every field. Two gems, catalyst used:

```
01 00 00 00        nMode = 1
02 08 10 00        nTargetItemID = 1082002
01                 bUseCatalyst = true
02 00 00 00        nGemCount = 2
41 5F 3D 00        nGemItemID[0] = 4021313
42 5F 3D 00        nGemItemID[1] = 4021314
```

Assert `Mode() == 1`, `TargetItemId() == 1082002`, `UseCatalyst() == true`, `len(GemItemIds()) == 2`, `GemItemIds()[0] == 4021313`, `GemItemIds()[1] == 4021314`.

`TestMakerSkillDecodeCreateWithUpgrade` — same shape with `nMode = 2`, zero gems, no catalyst:

```
02 00 00 00        nMode = 2
02 08 10 00        nTargetItemID = 1082002
00                 bUseCatalyst = false
00 00 00 00        nGemCount = 0
```

Assert `Mode() == 2`, `UseCatalyst() == false`, `len(GemItemIds()) == 0`.

`TestMakerSkillDecodeMonsterCrystal`:

```
03 00 00 00        nMode = 3
00 06 3D 00        nLeftoverItemID = 4000000
```

Assert `Mode() == 3`, `LeftoverItemId() == 4000000`. Assert `len(GemItemIds()) == 0` and `TargetItemId() == 0` — no mode-1 fields leak.

`TestMakerSkillDecodeDisassemble`:

```
04 00 00 00        nMode = 4
02 08 10 00        nItemID = 1082002
01 00 00 00        nInventoryType = 1
05 00 00 00        nSlotPos = 5
```

Assert `Mode() == 4`, `ItemId() == 1082002`, `InventoryType() == 1`, `SlotPos() == 5`.

`TestMakerSkillEncodeDecodeRoundTripPerMode` — table-driven over all four modes; construct via `NewMakerSkill`, `Encode`, `Decode` into a fresh struct, assert field-for-field equality. This is what proves `Encode` and `Decode` agree.

`TestMakerSkillDecodeIsVersionInvariant` — range **all eight** applicable variants; for each, decode the mode-1 fixture above under `pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)` and assert identical field values. This is the executable form of C-2. **If Task 6 found a divergence, replace this test with per-version expectations and a `MajorAtLeast` gate — do not delete it.**

Add a `// packet-audit:verify packet=character/serverbound/MakerSkill version=<key> ida=0x<addr>` marker above one byte-output test per version, with the address taken from Task 6's `wire-derivation.md`. The marker format is exactly as at `libs/atlas-packet/report/clientbound/sue_character_result_test.go:36`.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `libs/atlas-packet`:

```
go test ./character/serverbound/... -count=1 -run MakerSkill
```

Expected: FAIL — undefined `NewMakerSkill`, `MakerSkill`.

- [ ] **Step 3: Write `maker_skill.go`**

An immutable struct with unexported fields, a constructor, accessors, `Operation() string`, `String() string`, `Encode`, and `Decode` — matching the shape of `libs/atlas-packet/character/clientbound/show_combo.go`.

`Decode` reads `nMode` first, then switches on it to read that arm's fields, leaving the other arms' fields zero. `Encode` mirrors it exactly: write `nMode`, then switch and write only that arm's fields — one mode encode, per C-3.

Carry a `// packet-audit:fname CUIItemMaker::RequestItemMake` marker above the struct, and cite the per-version addresses from `wire-derivation.md` in the doc comment.

Add `const MakerSkillHandle = "MakerSkillHandle"` for the handler name Task 25 registers, following the `Writer`/`Handle` const convention used elsewhere in the package (e.g. `libs/atlas-packet/character/clientbound/buff_cancel.go:15`).

- [ ] **Step 4: Run the tests to verify they pass**

Run from `libs/atlas-packet`:

```
go build ./...
go test ./character/serverbound/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Register the candidate in `run.go`**

In `tools/packet-audit/cmd/run.go`'s `candidatesFromFName` (function starts at line 293), add — placed with the other serverbound entries, and **without** a `#` suffix, because this is not a dispatcher family:

```go
	case "CUIItemMaker::RequestItemMake":
		return []candidate{{name: "MakerSkill", pkg: "character", dir: csvpkg.DirServerbound}}
```

Follow the surrounding style: `libs/atlas-packet/character/serverbound` is reached via `pkg: "character"` + `dir: csvpkg.DirServerbound`. Confirm against a neighbouring serverbound entry that already resolves into a `pkg`-qualified directory.

- [ ] **Step 6: Run the packet gates**

Run from the repo root:

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit dispatcher-lint
cd tools/packet-audit && go test ./... -count=1
```

All must exit 0. Confirm the `MAKER_SKILL` row now reads `✅` on all eight applicable versions and `⬜` on `gms_v48`/`gms_v61`.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/character/serverbound/ tools/packet-audit/cmd/run.go docs/packets/audits/
git commit -m "feat(packet): add MAKER_SKILL serverbound codec across all eight versions"
```

---

## Task 8: `MAKER_RESULT` dispatcher family — arm structs and body functions

FR-4.2, FR-4.3, and design §4.3.2. Five discrete arms in one consolidated clientbound file, each with a config-resolved mode.

### Files

- `libs/atlas-packet/character/clientbound/maker_result.go` — new file; the five arm structs
- `libs/atlas-packet/character/maker_result_body.go` — new file; the operation-key consts and the five body functions
- `tools/packet-audit/cmd/run.go` — add five `#`-suffixed `candidatesFromFName` cases

Module roots: `libs/atlas-packet`; `tools/packet-audit` for `run.go`.

Patterns to copy: `libs/atlas-packet/cash/clientbound/shop_operation_result.go` (the canonical consolidated per-arm file) and `libs/atlas-packet/cash/clientbound/shop_operation_body.go:13-149` (the fixed operation-key const block) with its call sites at `:165-169`. `libs/atlas-packet/field/clientbound/mts_operation.go:55-105` shows a conditional-tail arm — the closest analogue to the create arm's two conditional pairs.

### Interfaces

- Consumes: `atlas_packet.WithResolvedCode` (`libs/atlas-packet/resolve.go:41`).
- Produces, for Tasks 9 and 26:
  - `const clientbound.MakerResultWriter = "MakerResult"`
  - structs `MakerResultCreate`, `MakerResultCreateWithUpgrade`, `MakerResultMonsterCrystal`, `MakerResultDisassemble`, `MakerResultFailed`, each with a `New*(mode byte, ...)` constructor
  - body functions `character.MakerResultCreateBody(...)`, `MakerResultCreateWithUpgradeBody(...)`, `MakerResultMonsterCrystalBody(...)`, `MakerResultDisassembleBody(...)`, `MakerResultFailedBody(...)`

### Derived arm bodies (design §4.3.2)

`nResult` is a shared **header** field on every struct, not an arm selector. A value `> 1` means the body stops there — that is the `FAILED` arm.

```
Encode4  nResult                       // > 1 ⇒ stop; this is the FAILED arm
Encode4  nMode

CREATE / CREATE_WITH_UPGRADE  (modes 1, 2 — identical bodies)
  Encode1  bNoItemAwarded              // when 0, the pair below follows
    Encode4  nTargetItemID
    Encode4  nItemNum
  Encode4  nMaterialCount
    nMaterialCount × { Encode4 nItemID; Encode4 nCount }
  Encode4  nGemCount
    nGemCount × { Encode4 nItemID }
  Encode1  bCatalystUsed
    Encode4  nCatalystItemID           // only when bCatalystUsed
  Encode4  nMesoCost

MONSTER_CRYSTAL  (mode 3)
  Encode4  nCrystalItemID
  Encode4  nLeftoverItemID             // no meso field on the wire

DISASSEMBLE  (mode 4)
  Encode4  nDisassembledItemID
  Encode4  nCrystalCount
    nCrystalCount × { Encode4 nItemID; Encode4 nCount }
  Encode4  nMesoCost

FAILED
  (nResult only)
```

### Three details that are easy to get wrong

**`bNoItemAwarded` is inverted.** The client reads `if (!Decode1()) { id = Decode4; num = Decode4; }`. A **truthy** byte *suppresses* the pair. Name the field for what the byte means on the wire, not for what it feels like it should mean.

**The meso field is a cost, not a refund.** The client renders it as `Format(SP_292_YOU_HAVE_LOST_MESOS_D, -v37)` in both mode 1|2 and mode 4 — a positive wire value displayed as a loss. PRD FR-3.4's "meso refund" wording refers to a *separate* meso award, not this field. Disassembly's wire meso is what the operation **charged**.

**The mode is `Encode4`, but the constructor takes `mode byte`.** `WithResolvedCode` resolves the operations-table value as a `byte` (`libs/atlas-packet/resolve.go:41` — `factory func(byte) packet.Encoder`), and INV-2 requires the constructor's first parameter to be that resolved `mode byte`. The struct widens it on write. Do **not** "fix" this by hard-coding a `uint32` mode literal — that is anti-pattern AP-2 and `dispatcher-lint` fails on it.

### Why `CREATE` and `CREATE_WITH_UPGRADE` are separate structs

Their bodies are identical, but INV-1 forbids one struct mapped by more than one `#`-entry, and the two modes are genuinely distinct operations to the client. Discrete means discrete.

- [ ] **Step 1: Write the arm structs**

Create `libs/atlas-packet/character/clientbound/maker_result.go`, `package clientbound`.

Declare `const MakerResultWriter = "MakerResult"` — the registry writer name shared by every arm, mirroring `MtsOperationWriter` at `libs/atlas-packet/field/clientbound/mts_operation.go:39`.

For each of the five arms declare: an immutable struct with unexported fields (`mode byte` first, then `result uint32`, then the arm's body fields), a `New<Arm>(mode byte, ...)` constructor, accessors, `Operation() string { return MakerResultWriter }`, `String() string`, `Encode`, and `Decode`.

Every `Encode` writes, in order: `nResult` as a 4-byte int, then `uint32(m.mode)` as a 4-byte int, then that arm's body. `MakerResultFailed.Encode` writes `nResult` only — no mode, because the client stops reading at `nResult > 1`.

Carry a `// packet-audit:fname CUserLocal::OnMakerResult#<Arm>` marker above each struct, with `<Arm>` exactly `Create`, `CreateWithUpgrade`, `MonsterCrystal`, `Disassemble`, `Failed`. Cite the per-version addresses from Task 6's `wire-derivation.md` in each doc comment.

Model the two conditional tails on `MtsResultRegisterSaleEntryFailed` (`libs/atlas-packet/field/clientbound/mts_operation.go:85-105`), whose `Encode`/`Decode` pair gates a trailing read on a decoded field — the same shape `bNoItemAwarded` and `bCatalystUsed` need.

The five constructor signatures, which Step 2's body functions call and Task 26 depends on. `mode byte` is first on every one (INV-2); `result` is the shared `nResult` header field:

```go
func NewMakerResultCreate(mode byte, result uint32, noItemAwarded bool, targetItemId uint32, itemNum uint32, materials []MakerMaterial, gemItemIds []uint32, catalystUsed bool, catalystItemId uint32, mesoCost uint32) MakerResultCreate

func NewMakerResultCreateWithUpgrade(mode byte, result uint32, noItemAwarded bool, targetItemId uint32, itemNum uint32, materials []MakerMaterial, gemItemIds []uint32, catalystUsed bool, catalystItemId uint32, mesoCost uint32) MakerResultCreateWithUpgrade

func NewMakerResultMonsterCrystal(mode byte, result uint32, crystalItemId uint32, leftoverItemId uint32) MakerResultMonsterCrystal

func NewMakerResultDisassemble(mode byte, result uint32, disassembledItemId uint32, crystals []MakerMaterial, mesoCost uint32) MakerResultDisassemble

func NewMakerResultFailed(result uint32) MakerResultFailed
```

`MakerResultFailed` takes no `mode` — the client stops reading at `nResult > 1`, so that arm writes no mode field. It is still a discrete struct with its own `#`-entry.

`MakerMaterial` is the shared `(itemId, count)` pair the create and disassemble arms both repeat:

```go
type MakerMaterial struct {
	itemId uint32
	count  uint32
}

func NewMakerMaterial(itemId uint32, count uint32) MakerMaterial
```

- [ ] **Step 2: Write the body functions**

Create `libs/atlas-packet/character/maker_result_body.go`, `package character`.

Declare the fixed operation keys, mirroring `libs/atlas-packet/cash/clientbound/shop_operation_body.go:13-149`:

```go
const (
	MakerResultOperationCreate            = "CREATE"
	MakerResultOperationCreateWithUpgrade = "CREATE_WITH_UPGRADE"
	MakerResultOperationMonsterCrystal    = "MONSTER_CRYSTAL"
	MakerResultOperationDisassemble       = "DISASSEMBLE"
	MakerResultOperationFailed            = "FAILED"
)
```

One body function per arm, each resolving its **own fixed const** — never a parameter:

```go
func MakerResultMonsterCrystalBody(result uint32, crystalItemId uint32, leftoverItemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MakerResultOperationMonsterCrystal, func(mode byte) packet.Encoder {
		return clientbound.NewMakerResultMonsterCrystal(mode, result, crystalItemId, leftoverItemId)
	})
}
```

INV-3 constraints, both by-name and semantic: **no** body function may take a parameter named `op`, `code`, `mode` or `key`, and **no** parameter of any name may flow into the `WithResolvedCode` key position. The key is always a hard-coded const. `func(_ byte)` is banned — the resolved mode must be passed through.

- [ ] **Step 3: Register the five candidates in `run.go`**

In `tools/packet-audit/cmd/run.go`'s `candidatesFromFName`, add the five `#`-suffixed entries, following `:705-711`'s shape:

```go
	case "CUserLocal::OnMakerResult#Create":
		return []candidate{{name: "MakerResultCreate", pkg: "character", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnMakerResult#CreateWithUpgrade":
		return []candidate{{name: "MakerResultCreateWithUpgrade", pkg: "character", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnMakerResult#MonsterCrystal":
		return []candidate{{name: "MakerResultMonsterCrystal", pkg: "character", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnMakerResult#Disassemble":
		return []candidate{{name: "MakerResultDisassemble", pkg: "character", dir: csvpkg.DirClientbound}}
	case "CUserLocal::OnMakerResult#Failed":
		return []candidate{{name: "MakerResultFailed", pkg: "character", dir: csvpkg.DirClientbound}}
```

Precede them with a comment explaining the disposition, as the `OnClaimResult`/`OnSueCharacterResult` block at `:679-704` does: this is a genuine multi-arm client dispatcher, authored discrete-per-mode from the start, so it goes in neither `dispatcher-lint-baseline.yaml` (which is empty and only shrinks) nor `families.yaml` (adding it would cap the op at `🧩`; the FIELD_EFFECT model of aggregating worst-of-all-arms applies instead).

- [ ] **Step 4: Verify it builds and the linter is satisfied**

Run from `libs/atlas-packet`:

```
go build ./...
go vet ./...
```

Then from the repo root:

```
go run ./tools/packet-audit dispatcher-lint
```

Expected: exit 0. If INV-2 fires, a `mode: 0x` literal or a `func(_ byte)` survived. If INV-3 fires, a body function is letting its caller pick the operation. If INV-5 fires, an arm struct has no body function constructing it — all five must be reachable.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/clientbound/maker_result.go libs/atlas-packet/character/maker_result_body.go tools/packet-audit/cmd/run.go
git commit -m "feat(packet): add MAKER_RESULT dispatcher family with five discrete arms"
```

---

## Task 9: `MAKER_RESULT` — dispatcher YAML, operations tables, byte fixtures

Completes the family: the mode-table source of truth, the generated per-version `operations` tables, and per-arm per-version verification.

### Files

- `docs/packets/dispatchers/maker_result.yaml` — new file; the complete arm enumeration
- `libs/atlas-packet/character/clientbound/maker_result_test.go` — new file; byte fixtures with `packet-audit:verify` markers
- `services/atlas-configurations/seed-data/templates/` — the eight applicable templates, **regenerated** by `packet-audit operations`, never hand-edited

Module root for the fixture test: `libs/atlas-packet`.

Patterns to copy: `docs/packets/dispatchers/sue_character_result.yaml` (full file, 34 lines — the `writer`/`fname`/`op`/`direction`/`opcodes`/`operations` schema) and `libs/atlas-packet/report/clientbound/sue_character_result_test.go:1-50` (a complete `packet-audit:verify` fixture test).

### Interfaces

- Consumes: Task 6's `wire-derivation.md`, Task 8's arm structs and body functions.
- Produces: `MAKER_RESULT` at `✅` on all eight applicable versions, and the `operations` mode table in each of the eight seed templates that Task 26's writer resolves against.

### The `operations` table is generated, not hand-written

`docs/packets/dispatchers/README.md` is explicit: `packet-audit operations` **regenerates each template's tables wholesale from the YAML**, and a key that lives only in a template is deleted on the next regeneration and reported as `EXTRA` by `--check` in the meantime. So the YAML is authored, and the templates are produced from it. Never edit a template's `operations` block by hand.

- [ ] **Step 1: Write `docs/packets/dispatchers/maker_result.yaml`**

Schema per `sue_character_result.yaml`. The `opcodes` block carries the parent op's per-version opcode from the registry; `operations` enumerates the five arms with their per-version mode values, taken from Task 6's derivation.

```yaml
# MakerResult — CUserLocal::OnMakerResult item-maker result dispatcher.
# NOTE (design C-1): this op is RESULT-CODE-prefixed, not mode-prefixed. The
# client reads Encode4(nResult) FIRST and only reads Encode4(nMode) when
# nResult <= 1; a result > 1 ends the body (the FAILED arm). The mode is
# therefore the SECOND field, and nResult is a shared header field carried by
# every arm struct, not an arm selector. DISPATCHER_FAMILY.md requires a
# config-resolved mode, not that the mode be the first field, so this is still
# a legitimate family.
# Modes 1 and 2 are body-identical but are distinct client operations and get
# discrete structs per INV-1.
# Addresses and per-version confirmation: docs/tasks/task-285-maker-skill-crafting/wire-derivation.md
writer: MakerResult
fname: CUserLocal::OnMakerResult
op: MAKER_RESULT
direction: clientbound
opcodes:
  gms_v72: "0x0C7"
  gms_v79: "0x0CB"
  gms_v83: "0x0D9"
  gms_v84: "0x0DD"
  gms_v87: "0x0E6"
  gms_v92: "0x0FA"
  gms_v95: "0x0F8"
  jms_v185: "0x0E2"
operations:
  - { key: CREATE,              modes: { gms_v72: 1, gms_v79: 1, gms_v83: 1, gms_v84: 1, gms_v87: 1, gms_v92: 1, gms_v95: 1, jms_v185: 1 } }
  - { key: CREATE_WITH_UPGRADE, modes: { gms_v72: 2, gms_v79: 2, gms_v83: 2, gms_v84: 2, gms_v87: 2, gms_v92: 2, gms_v95: 2, jms_v185: 2 } }
  - { key: MONSTER_CRYSTAL,     modes: { gms_v72: 3, gms_v79: 3, gms_v83: 3, gms_v84: 3, gms_v87: 3, gms_v92: 3, gms_v95: 3, jms_v185: 3 } }
  - { key: DISASSEMBLE,         modes: { gms_v72: 4, gms_v79: 4, gms_v83: 4, gms_v84: 4, gms_v87: 4, gms_v92: 4, gms_v95: 4, jms_v185: 4 } }
  - { key: FAILED,              modes: { gms_v72: 0, gms_v79: 0, gms_v83: 0, gms_v84: 0, gms_v87: 0, gms_v92: 0, gms_v95: 0, jms_v185: 0 } }
```

**Verify the opcodes and every mode value against the registry and `wire-derivation.md` before committing — do not accept the table above on faith.** The opcodes are transcribed from PRD FR-4.2 and must match `docs/packets/registry/<key>.yaml`'s `MAKER_RESULT` entry for each of the eight versions. The `FAILED` mode value of `0` is a placeholder: that arm writes no mode byte at all, so pick the value the client's own switch leaves unused and record in the file comment why it was chosen.

- [ ] **Step 2: Generate the templates' operations tables**

Run from the repo root:

```
go run ./tools/packet-audit operations
go run ./tools/packet-audit operations --check
```

The first regenerates the `operations` tables in the seed templates from the YAML; the second must exit 0. Confirm by inspection that the eight applicable templates gained a `MakerResult` writer entry with the five keys, and that `template_gms_12_1.json`, `template_gms_48_1.json` and `template_gms_61_1.json` were **not** touched.

- [ ] **Step 3: Write the byte fixtures**

Create `libs/atlas-packet/character/clientbound/maker_result_test.go`, `package clientbound`. Use `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"` and the `pt.Variants[i]` + `v.Name` guard idiom; the index table is in Task 7 Step 1.

One test function per arm per version, each preceded by a doc comment carrying the IDA evidence and a marker of exactly this form (see `libs/atlas-packet/report/clientbound/sue_character_result_test.go:36`):

```
// packet-audit:verify packet=character/clientbound/MakerResultCreate version=gms_v83 ida=0x95dad3
```

Expected bytes per arm. All integers little-endian; `nResult = 0` (success) except in `Failed`.

`MakerResultCreate` — mode 1, item awarded, two materials, one gem, catalyst used, 1200 meso:

```
00 00 00 00        nResult = 0
01 00 00 00        nMode = 1
00                 bNoItemAwarded = 0  -> the pair follows
02 08 10 00        nTargetItemID = 1082002
01 00 00 00        nItemNum = 1
02 00 00 00        nMaterialCount = 2
D9 2D 3D 00        nItemID  = 4011001
05 00 00 00        nCount   = 5
DA 2D 3D 00        nItemID  = 4011002
03 00 00 00        nCount   = 3
01 00 00 00        nGemCount = 1
41 5F 3D 00        nItemID  = 4021313
01                 bCatalystUsed = 1
80 03 3F 00        nCatalystItemID = 4130944
B0 04 00 00        nMesoCost = 1200
```

`MakerResultCreate` no-item variant — assert that `bNoItemAwarded = 1` **suppresses** the `nTargetItemID`/`nItemNum` pair, so the byte after the flag is `nMaterialCount`. This is the inverted-flag regression test and it must exist.

`MakerResultCreate` no-catalyst variant — assert `bCatalystUsed = 0` suppresses `nCatalystItemID`, so `nMesoCost` follows the flag directly.

`MakerResultCreateWithUpgrade` — byte-identical to `MakerResultCreate` except `nMode = 2`. Assert the full byte string, not just the mode, so a future edit to one arm that forgets the other is caught.

`MakerResultMonsterCrystal` — mode 3, no meso field:

```
00 00 00 00        nResult = 0
03 00 00 00        nMode = 3
00 06 3D 00        nCrystalItemID  = 4000000
01 06 3D 00        nLeftoverItemID = 4000001
```

Assert the encoded length is exactly 16 bytes — this is what proves no meso field leaked onto the wire.

`MakerResultDisassemble` — mode 4, two crystal stacks, 500 meso charged:

```
00 00 00 00        nResult = 0
04 00 00 00        nMode = 4
02 08 10 00        nDisassembledItemID = 1082002
02 00 00 00        nCrystalCount = 2
00 06 3D 00        nItemID = 4000000
03 00 00 00        nCount  = 3
01 06 3D 00        nItemID = 4000001
01 00 00 00        nCount  = 1
F4 01 00 00        nMesoCost = 500
```

`MakerResultFailed` — `nResult = 2`, nothing else:

```
02 00 00 00        nResult = 2
```

Assert the encoded length is exactly 4 bytes. The client stops reading at `nResult > 1`, so writing a mode here would desynchronise it.

`TestMakerResultBodyFunctionsResolveMode` — for each of the five body functions, invoke it with an `options` map containing an `operations` table that maps the arm's key to a **non-default** byte (e.g. `CREATE → 7`), and assert the encoded mode field is `7`, not `1`. This is the executable proof that the mode is config-resolved rather than hard-coded, and it is the test that would have caught AP-2 and AP-3.

Cover all eight versions for every arm. Where Task 6 confirmed the layout identical, a table-driven test ranging the eight variants with one shared expectation is correct and preferred over eight copy-pasted functions — but each version still needs its own `packet-audit:verify` marker with its own address, since the matrix grades per cell.

- [ ] **Step 4: Run the tests**

Run from `libs/atlas-packet`:

```
go build ./...
go test ./character/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Run every packet gate**

Run from the repo root:

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit doc-freshness --check
go run ./tools/packet-audit gate-check --check
cd tools/packet-audit && go test ./... -count=1
```

Every one must exit 0. Then confirm in `docs/packets/audits/STATUS.md` that `MAKER_RESULT` reads `✅` — not `🧩` — on all eight applicable versions and `⬜` on `gms_v48`/`gms_v61`. A `🧩` means an arm is unverified; a `🟥` means a conflict. Neither is acceptable.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/dispatchers/maker_result.yaml libs/atlas-packet/character/clientbound/maker_result_test.go services/atlas-configurations/seed-data/templates/ docs/packets/audits/
git commit -m "feat(packet): verify MAKER_RESULT arms across all eight versions"
```

---

## Task 10: `libs/atlas-saga` — the `AwardCraftedAsset` action

Design §4.5.1. One new saga action, because no existing creation payload can express "an equip with `tuc` upgrade slots and explicit reagent-adjusted stats".

### Files

- `libs/atlas-saga/model.go` — add the `AwardCraftedAsset` action constant
- `libs/atlas-saga/payloads.go` — add `AwardCraftedAssetPayload`
- `libs/atlas-saga/unmarshal.go` — add the `Step[T].UnmarshalJSON` case
- `libs/atlas-saga/payloads_test.go` — round-trip and back-compat tests

Module root for `go build`/`go test`: `libs/atlas-saga`.

Patterns to copy: the `AcceptToParcel` action added in commit `9486b6088` ("Duey parcel delivery", #1434) — constant at `libs/atlas-saga/model.go:241`, payload at `payloads.go:1017`, unmarshal case at `unmarshal.go:480-499`. **Do not use the most recent action (`MapleLifeUse`, `model.go:85`, commit `7f157fb03`) as the template** — it is deliberately single-step and non-compensable, so it touched none of `unmarshal.go`, `event_acceptance.go`, or `compensator.go`, and following it would silently omit four of this action's five wiring points.

`AcceptToMtsListingPayload` (`libs/atlas-saga/payloads.go:880-915`) is the shape to clone: it already carries the full explicit-stat block **and** `Slots uint16` at `:907`.

### Interfaces

- Consumes: nothing.
- Produces, for Tasks 12-14 and 23:
  - `saga.AwardCraftedAsset Action = "award_crafted_asset"`
  - `saga.AwardCraftedAssetPayload` (fields below)
  - a `Step[T].UnmarshalJSON` case decoding it

### Why not widen `AwardAsset`

`AwardAsset` carries `ItemPayload{TemplateId, Quantity, Period, Expiration}` (`payloads.go:20-32`) and its compensation path is `RequestDestroyItem(templateId, quantity)`. It is used by dozens of call sites. Widening its payload widens the blast radius of every one of them for a field only maker sets, and blurs the compensation semantics of both. `CreateAndEquipAsset` adds only `UseAverageStats bool` — a toggle into randomized stat rolling, which cannot express "set these exact stats". Explicit per-stat fields exist today **only** on *snapshot* payloads that move an already-existing asset between custodies (`AcceptToMtsListingPayload`, `AcceptToParcelPayload`), never on a creation payload. A discrete action keeps both narrow.

- [ ] **Step 1: Write the failing tests**

Add to `libs/atlas-saga/payloads_test.go`, `package saga`. The file's idiom is plain `encoding/json` marshal/unmarshal assertions — see `TestDestroyAssetFromSlotPayloadTemplateIdRoundTrip` at `:9-37` for the exact shape, including its legacy-JSON back-compat half.

`TestAwardCraftedAssetActionConstant` — assert `string(AwardCraftedAsset) == "award_crafted_asset"`. Mirrors `TestSkillBookUseSagaType` at `:39`.

`TestAwardCraftedAssetPayloadRoundTrip` — build a fully-populated payload:

```go
in := AwardCraftedAssetPayload{
	CharacterId:   1,
	TemplateId:    1082002,
	Quantity:      1,
	Slots:         7,
	Strength:      3,
	Dexterity:     2,
	Intelligence:  0,
	Luck:          0,
	HP:            15,
	MP:            0,
	WeaponAttack:  4,
	MagicAttack:   0,
	WeaponDefense: 6,
	MagicDefense:  1,
	Accuracy:      2,
	Avoidability:  1,
	Hands:         0,
	Speed:         0,
	Jump:          0,
}
```

`json.Marshal`, `json.Unmarshal` into a fresh value, assert field-for-field equality across **all** of them. A stat silently dropped by a missing JSON tag is exactly the defect this catches.

`TestAwardCraftedAssetPayloadSlotsSurvivesZero` — set `Slots: 0` explicitly, marshal, and assert the JSON **contains** `"slots":0`. `Slots` must not carry `omitempty`: a zero-slot craft (a non-upgradeable equip) is meaningful and must be distinguishable from an absent field downstream.

`TestAwardCraftedAssetStepUnmarshal` — the `unmarshal.go` case. Unmarshal this literal into a `Step[any]`:

```json
{"stepId":"award","status":"pending","action":"award_crafted_asset","payload":{"characterId":1,"templateId":1082002,"quantity":1,"slots":7,"strength":3,"weaponAttack":4}}
```

Assert `step.Action == AwardCraftedAsset` and that `step.Payload` type-asserts to `AwardCraftedAssetPayload` with `TemplateId == 1082002`, `Slots == 7`, `Strength == 3`, `WeaponAttack == 4`. Untyped-payload fallthrough is the failure mode here — assert the concrete type, not just the field values.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `libs/atlas-saga`:

```
go test ./... -count=1 -run AwardCraftedAsset
```

Expected: FAIL — undefined `AwardCraftedAsset`, `AwardCraftedAssetPayload`.

- [ ] **Step 3: Add the action constant**

In `libs/atlas-saga/model.go`, add to the `Action` const block (the block containing `AcceptToParcel` at `:241`):

```go
	AwardCraftedAsset Action = "award_crafted_asset"
```

- [ ] **Step 4: Add the payload**

In `libs/atlas-saga/payloads.go`, alongside the other award payloads:

```go
// AwardCraftedAssetPayload creates an equip with EXPLICIT stats and an explicit
// upgrade-slot count. It exists because neither AwardAsset (ItemPayload only)
// nor CreateAndEquipAsset (which adds only UseAverageStats, a toggle into
// randomized rolling) can express "an equip with tuc upgrade slots and
// reagent-adjusted stats" — FR-3.1 and FR-3.2. The stat block mirrors
// AcceptToMtsListingPayload's; Slots carries the recipe's `tuc`.
//
// Slots deliberately has no omitempty: a zero-slot craft is meaningful and must
// be distinguishable from an absent field.
type AwardCraftedAssetPayload struct {
	CharacterId   uint32 `json:"characterId"`
	TemplateId    uint32 `json:"templateId"`
	Quantity      uint32 `json:"quantity"`
	Slots         uint16 `json:"slots"`
	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	HP            uint16 `json:"hp"`
	MP            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
	ShowEffect    bool   `json:"showEffect"`
}
```

- [ ] **Step 5: Add the unmarshal case**

In `libs/atlas-saga/unmarshal.go`'s `Step[T].UnmarshalJSON`, add a `case AwardCraftedAsset:` arm following the exact shape of the `case AcceptToParcel:` arm at `:480-499`.

- [ ] **Step 6: Run the tests to verify they pass**

Run from `libs/atlas-saga`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(saga): add AwardCraftedAsset action for explicit-stat equip creation"
```

---

## Task 11: `atlas-inventory` — explicit-stat asset creation

**This is the cross-service seam.** Per CLAUDE.md it is the part a green `verify.sh` cannot see, so this task's deliverable is a test in `atlas-inventory` asserting the **new** contract, not the old silent drop.

### Files

- `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go` — extend `CreateAssetCommandBody`
- `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka_test.go` — assert the new wire contract
- `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go` — thread the new fields through `handleCreateAssetCommand`
- `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` — thread them into `asset.CreateOptions`
- `services/atlas-inventory/atlas.com/inventory/asset/processor.go` — extend `CreateOptions` and honour it in `Create`
- `services/atlas-inventory/atlas.com/inventory/asset/processor_test.go` — assert stats and slots land on the created asset

Module root for `go build`/`go test`: `services/atlas-inventory/atlas.com/inventory`.

### Interfaces

- Consumes: nothing from earlier tasks — this task is independently mergeable.
- Produces, for Task 12: a `CreateAssetCommandBody` that carries explicit stats and a slot count, and an `asset.CreateOptions` that honours them.

### The exact gap, verified

The full hop chain, and the fact that closes the design's argument:

| # | Hop | Location |
|---|---|---|
| 1 | `RequestCreateItem` → `RequestCreateItemWithStats` | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor.go:65-75` |
| 2 | command provider | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/producer.go:16-34` |
| 3 | command body (orchestrator copy) | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go:127-135` |
| 3' | command body (**inventory copy**) | `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go:109` |
| 4 | consumer | `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:232-243` |
| 5 | `CreateAssetAndEmit` → `CreateAssetAndLock` → `CreateAsset` | `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:1246-1358` |
| 6 | `asset.CreateOptions` | `services/atlas-inventory/atlas.com/inventory/asset/processor.go:510-517` |

`asset.CreateOptions` today is exactly:

```go
type CreateOptions struct {
	Quantity        uint32
	Expiration      time.Time
	OwnerId         uint32
	Flag            uint16
	Rechargeable    uint64
	UseAverageStats bool
}
```

There is **no** field for explicit per-stat values and none for an upgrade-slot count, anywhere in the chain. `UseAverageStats` is a boolean toggle into randomized/average stat rolling — it cannot express "set these exact stats". This is the strongest confirmation that `AwardCraftedAsset` is not avoidable by widening `AwardAsset`.

### The two-copy trap

`CreateAssetCommandBody` is declared **independently in both services** — there is no shared type for this message. `services/atlas-saga-orchestrator/.../kafka/message/compartment/kafka.go:127-135` and `services/atlas-inventory/.../kafka/message/compartment/kafka.go:109` must be extended in lockstep with identical JSON tags. A field added to one and not the other decodes as its zero value with no error — a silent drop. This task extends the inventory copy; Task 12 extends the orchestrator copy; Task 12's final step asserts the two agree.

Every new field carries `omitempty` **except** `Slots`, and every existing producer keeps working unchanged: absent → zero → current behaviour.

- [ ] **Step 1: Write the failing tests**

Add to `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka_test.go`. The existing `TestCreateAssetCommandBody_UseAverageStats_RoundTrip` at `:10` and `..._OmitEmpty` at `:28` are the template.

`TestCreateAssetCommandBody_ExplicitStats_RoundTrip` — build a body with `Slots: 7`, `Strength: 3`, `WeaponAttack: 4`, `WeaponDefense: 6`, `HP: 15`, marshal, unmarshal, assert each survives.

`TestCreateAssetCommandBody_LegacyPayloadDecodesWithZeroStats` — unmarshal the legacy literal `{"templateId":1082002,"quantity":1,"ownerId":0,"flag":0,"rechargeable":0}` and assert `Slots == 0`, `Strength == 0`, `WeaponAttack == 0`. This pins that the extension is additive and cannot break existing producers.

Add to `services/atlas-inventory/atlas.com/inventory/asset/processor_test.go`, using the package's existing Builder-based setup (no `*_testhelpers.go`):

`TestCreateAssetAppliesExplicitStats` — call `Create` with `CreateOptions{Quantity: 1, Slots: 7, Strength: 3, WeaponAttack: 4, WeaponDefense: 6, HP: 15, UseAverageStats: false}` for equip template `1082002`, then assert the persisted asset's stats are **exactly** those values — not rolled, not averaged. Assert `Slots == 7`.

`TestCreateAssetExplicitStatsTakePrecedenceOverAverage` — same call with `UseAverageStats: true` **and** explicit stats set. Assert the explicit values win. Decide and document this precedence here rather than leaving it implicit; explicit-wins is the only rule under which a craft's reagent adjustment is reproducible.

`TestCreateAssetWithoutExplicitStatsIsUnchanged` — call `Create` with `CreateOptions{Quantity: 1, UseAverageStats: true}` and no stat fields, and assert the result matches the pre-change behaviour. This is the regression guard for every existing caller.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-inventory/atlas.com/inventory`:

```
go test ./kafka/message/compartment/... ./asset/... -count=1
```

Expected: FAIL — unknown fields `Slots`, `Strength`, `WeaponAttack` on `CreateAssetCommandBody` and on `CreateOptions`.

- [ ] **Step 3: Extend `CreateAssetCommandBody`**

In `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go`, append to the struct at `:109`:

```go
	Slots         uint16 `json:"slots"`
	Strength      uint16 `json:"strength,omitempty"`
	Dexterity     uint16 `json:"dexterity,omitempty"`
	Intelligence  uint16 `json:"intelligence,omitempty"`
	Luck          uint16 `json:"luck,omitempty"`
	HP            uint16 `json:"hp,omitempty"`
	MP            uint16 `json:"mp,omitempty"`
	WeaponAttack  uint16 `json:"weaponAttack,omitempty"`
	MagicAttack   uint16 `json:"magicAttack,omitempty"`
	WeaponDefense uint16 `json:"weaponDefense,omitempty"`
	MagicDefense  uint16 `json:"magicDefense,omitempty"`
	Accuracy      uint16 `json:"accuracy,omitempty"`
	Avoidability  uint16 `json:"avoidability,omitempty"`
	Hands         uint16 `json:"hands,omitempty"`
	Speed         uint16 `json:"speed,omitempty"`
	Jump          uint16 `json:"jump,omitempty"`
```

The JSON tags must match `AwardCraftedAssetPayload`'s (Task 10) character for character.

- [ ] **Step 4: Extend `asset.CreateOptions` and honour it**

In `services/atlas-inventory/atlas.com/inventory/asset/processor.go`, add the same stat fields plus `Slots uint16` to `CreateOptions` at `:510-517`.

In the equip-creation path that `Create` runs, apply the explicit values when any is non-zero or `Slots` is set, taking precedence over `UseAverageStats`. Read the current stat-rolling branch before editing so the explicit path sets exactly the fields the rolled path sets — a stat the rolled path assigns but the explicit path forgets is a silent zero on every crafted equip.

- [ ] **Step 5: Thread the fields through the consumer and compartment processor**

In `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go`'s `handleCreateAssetCommand` (`:232-243`), pass the new body fields into the `CreateAssetAndEmit` call. Preserve the `database.ApplyOnce` idempotency guard (task-208) exactly as it stands — do not restructure it.

In `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`, widen `CreateAssetAndEmit` / `CreateAssetAndLock` / `CreateAsset` to carry the values into `asset.CreateOptions` at `:1327-1334` and `:1346-1353`.

If any of these is an interface method, update its mock in the same step — the interface and its mock change together.

- [ ] **Step 6: Run the tests to verify they pass**

Run from `services/atlas-inventory/atlas.com/inventory`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS, including every pre-existing test. A failure in an unrelated package means a widened signature was not propagated to a mock.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/
git commit -m "feat(inventory): accept explicit stats and upgrade slots on asset creation"
```

---

## Task 12: `atlas-saga-orchestrator` — explicit-stat creation command

The producer half of Task 11's seam.

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go` — extend `CreateAssetCommandBody` to match the inventory copy
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/producer.go` — carry the new fields into the command
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor.go` — add `RequestCreateItemWithExplicitStats`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/mock/processor.go` — mirror the new interface method

Module root for `go build`/`go test`: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

Patterns to copy: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor.go:65-75` — `RequestCreateItem` delegating to `RequestCreateItemWithStats`, which is the existing precedent for exactly this kind of widening:

```go
func (p *ProcessorImpl) RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error {
	return p.RequestCreateItemWithStats(transactionId, characterId, templateId, quantity, expiration, false)
}
```

### Interfaces

- Consumes: Task 11's extended `CreateAssetCommandBody` JSON contract.
- Produces, for Task 13: `compartment.Processor.RequestCreateItemWithExplicitStats(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time, stats saga.AwardCraftedAssetPayload) error`.

Passing the payload struct itself, rather than eighteen positional `uint16` parameters, is deliberate: a positional list of same-typed stats is the shape where a transposition (accuracy into avoidability) compiles cleanly and is invisible until a player notices.

- [ ] **Step 1: Write the failing test**

Add to the `compartment` package's existing producer test file, using the package's current test setup.

`TestRequestCreateItemWithExplicitStatsCarriesEveryField` — invoke the provider with a fully-populated stat set, decode the emitted `CreateAssetCommandBody` from the produced message, and assert every stat and `Slots` survived. Assert `Slots == 7` specifically, since it is the one field without `omitempty`.

`TestRequestCreateItemIsUnchanged` — invoke the existing `RequestCreateItem` and assert the emitted body has all-zero stats and `Slots == 0`, i.e. the legacy path is byte-identical to before.

- [ ] **Step 2: Run the test to verify it fails**

Run from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`:

```
go test ./compartment/... -count=1
```

Expected: FAIL — undefined `RequestCreateItemWithExplicitStats`.

- [ ] **Step 3: Extend the orchestrator's `CreateAssetCommandBody`**

Append the identical block from Task 11 Step 3 to the struct at `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go:127-135`. The JSON tags must match the inventory copy character for character.

- [ ] **Step 4: Add the producer and processor method**

Extend `RequestCreateAssetCommandProvider` (`compartment/producer.go:16-34`) to accept and set the new fields.

Add `RequestCreateItemWithExplicitStats` to the `Processor` interface (`compartment/processor.go:33-48`) and implement it alongside `RequestCreateItemWithStats` at `:65-75`, reusing the same `inventory.TypeFromItemId(item.Id(templateId))` guard and the same `producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)` emission.

Update `compartment/mock/processor.go` in the same step.

- [ ] **Step 5: Assert the two command-body copies agree**

This is the check that makes the silent-drop failure mode impossible to reintroduce. Add a test asserting that the JSON produced by the orchestrator's `CreateAssetCommandBody` unmarshals into the inventory's copy with every field preserved.

Because the two services are separate modules, the practical form is a golden-JSON assertion in each: marshal a fully-populated body and compare against one shared literal committed in both test files. State in a comment in both that the literal is shared and must be updated in both places together.

- [ ] **Step 6: Run the tests to verify they pass**

Run from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/ services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/
git commit -m "feat(saga-orchestrator): emit explicit-stat asset creation commands"
```

---

## Task 13: `atlas-saga-orchestrator` — `AwardCraftedAsset` handler and step completion

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — the type alias re-export **and** the local unmarshal case
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — interface method, `GetHandler` case, and the handler body
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — the completion/error event pair
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go` — handler and acceptance tests

Module root for `go build`/`go test`: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

Patterns to copy: `AcceptToParcel`'s wiring — alias at `saga/model.go:212`, payload alias at `:356`, local unmarshal case at `:1575-1580`, interface method at `handler.go:152`, `GetHandler` case at `:944-945`, handler body at `:2537-2560`, acceptance entry at `event_acceptance.go:259`.

### Interfaces

- Consumes: `saga.AwardCraftedAsset` and `saga.AwardCraftedAssetPayload` (Task 10); `compartment.Processor.RequestCreateItemWithExplicitStats` (Task 12).
- Produces, for Task 14: a dispatched `AwardCraftedAsset` step that completes on the inventory's asset-created event.

### `saga/model.go` needs TWO changes, not one

The design's §4.5.1 table lists a single "local re-export" row for this file. That undercounts it. The orchestrator maintains its **own** `Step[T].UnmarshalJSON` implementation alongside the shared library's, and both must learn the new action:

1. The type-alias re-exports — `AwardCraftedAsset = sharedsaga.AwardCraftedAsset` near `:212`, and the payload alias near `:356`.
2. An independent `case AwardCraftedAsset:` arm inside this file's own `Step[T].UnmarshalJSON`, near the `case AcceptToParcel:` arm at `:1575-1580`, duplicating the same `json.Unmarshal` logic `libs/atlas-saga/unmarshal.go` already performs for the shared type.

Omitting (2) yields an untyped payload at dispatch time with no compile error — the handler receives a `map[string]interface{}` and the type assertion fails at runtime.

### `saga/rest.go` is NOT required

The design's table lists a `saga/rest.go` "REST payload unmarshaller" row. The evidence says it is optional. `payloadUnmarshalers` (`saga/rest.go:75-85`) covers only nine actions total — `AwardAsset`, `AwardExperience`, `AwardLevel`, `AwardMesos`, `WarpToRandomPortal`, `WarpToPortal`, `DestroyAsset`, `DestroyAllAssets`, `DestroyAssetFromSlot` — and `AcceptToParcel`, a full multi-step compensable custody action, was **not** added to it; it falls through to the untyped default at `rest.go:96-104`.

**Do not add an entry here.** That map only affects typed decoding for the orchestrator's saga-inspection REST endpoint, and nothing in this feature reads a craft saga back through it. If a later task needs typed inspection, add it then, with a test that exercises the endpoint.

- [ ] **Step 1: Write the failing tests**

Add to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go`, using the file's existing setup.

`TestGetHandlerResolvesAwardCraftedAsset` — assert `GetHandler(AwardCraftedAsset)` returns a non-nil handler. A missing `GetHandler` case is what leaves a saga stuck forever with no error.

`TestStepUnmarshalAwardCraftedAssetLocal` — the orchestrator's own `Step[T].UnmarshalJSON`. Unmarshal the JSON literal from Task 10 Step 1 through **this package's** `Step` type and assert `step.Payload` type-asserts to `AwardCraftedAssetPayload` with `TemplateId == 1082002` and `Slots == 7`. This is the test that catches the omitted second `model.go` change.

`TestHandleAwardCraftedAssetRequestsCreationWithStats` — with a mocked `compartment.Processor`, dispatch an `AwardCraftedAsset` step and assert `RequestCreateItemWithExplicitStats` was called once with the payload's template id, quantity and full stat block, and with the step's transaction id.

`TestAwardCraftedAssetEventAcceptance` — assert the acceptance table maps `AwardCraftedAsset` to the same asset-created / asset-error event kinds `AwardAsset` uses. Read the `AwardAsset` entry and assert against those exact constants; do not invent new event kinds.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`:

```
go test ./saga/... -count=1 -run AwardCraftedAsset
```

Expected: FAIL — undefined `AwardCraftedAsset` in this package.

- [ ] **Step 3: Wire `saga/model.go` — both changes**

Add the two type aliases and the local `Step[T].UnmarshalJSON` case, exactly as described above.

- [ ] **Step 4: Wire `saga/handler.go`**

Add `handleAwardCraftedAsset` to the handler interface near `:152`, the `case AwardCraftedAsset:` arm to `GetHandler` near `:944-945`, and the handler body near `:2537-2560`. The body extracts the typed payload and calls `RequestCreateItemWithExplicitStats`.

- [ ] **Step 5: Wire `saga/event_acceptance.go`**

Add the `AwardCraftedAsset` entry near `:259`, mapping to the same event-kind pair `AwardAsset` uses. **An async action with no acceptance entry never completes** — the saga hangs until timeout.

- [ ] **Step 6: Run the tests to verify they pass**

Run from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/
git commit -m "feat(saga-orchestrator): handle AwardCraftedAsset steps"
```

---

## Task 14: `atlas-saga-orchestrator` — `AwardCraftedAsset` compensation

FR-3.7 and PRD §8 Atomicity: a crash mid-craft resolves to fully applied or fully compensated, never partial.

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` — reverse-walk case, `lateCompensableActions` entry, `dispatchLateInverse` case
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go` — compensation tests

Module root for `go build`/`go test`: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

Patterns to copy: `compensator.go:1267-1276` (the `case AwardAsset:` reverse-walk arm — the closest analogue, since the inverse of creating an asset is destroying it), `:2977-3003` (`lateCompensableActions`, with `AcceptToParcel`/`ReleaseFromParcel` at `:3001-3002`), and `:3303-3311` (the `case AcceptToParcel:` arm of `dispatchLateInverse`, reached from `:3139`). `CompensateFailedStep` begins at `:331`.

### Interfaces

- Consumes: Task 13's dispatched step.
- Produces: a compensable `AwardCraftedAsset` — the last piece that makes every step of every craft saga reversible.

### Why this task exists separately

Every step in every sequence in design §4.5.2 must use a compensable action; that is why `DestroyAllAssets` is excluded from the material-consumption path. `AwardCraftedAsset` is the only *new* action in the feature, so it is the only one whose inverse does not already exist. Its inverse is the same as `AwardAsset`'s: destroy the created asset by template and quantity.

- [ ] **Step 1: Write the failing tests**

Add to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go`, using the file's existing setup.

`TestCompensateAwardCraftedAssetDestroysTheAsset` — build a saga whose `AwardCraftedAsset` step completed and whose next step failed; run the reverse walk; assert a destroy was requested for the crafted template at the crafted quantity.

`TestAwardCraftedAssetIsLateCompensable` — assert `AwardCraftedAsset` is present in `lateCompensableActions`. Read the map at `:2977-3003` and assert membership directly.

`TestDispatchLateInverseAwardCraftedAsset` — assert `dispatchLateInverse` handles `AwardCraftedAsset` and issues the destroy. A missing arm here means the action is *declared* late-compensable but has no inverse to dispatch, which fails at runtime rather than at build time.

`TestCraftSagaFullyCompensatesOnFinalStepFailure` — the FR-3.7 acceptance test at saga level. Build the mode-1 sequence from design §4.5.2 — `AwardMesos` (negative) → `DestroyAssetFromSlot` per material slot → `AwardCraftedAsset` — fail the last step, and assert the compensation dispatches: a re-create for every destroyed slot (via `DestroyAssetFromSlotPayload.TemplateId`, which exists at `libs/atlas-saga/payloads.go:141` precisely to enable this) and a positive `AwardMesos` reversing the charge. Assert the count of compensating dispatches equals the count of completed steps — a partial compensation is the failure this test exists to catch.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`:

```
go test ./saga/... -count=1 -run Compensate
```

Expected: FAIL — no compensation path for `AwardCraftedAsset`.

- [ ] **Step 3: Add the three compensator wirings**

1. A `case AwardCraftedAsset:` arm in the reverse walk reached from `CompensateFailedStep` (`:331`), modelled on `case AwardAsset:` at `:1267-1276`, requesting destruction of the crafted template at the crafted quantity.
2. `AwardCraftedAsset: {}` in `lateCompensableActions` (`:2977-3003`).
3. A `case AwardCraftedAsset:` arm in `dispatchLateInverse` (`:3139` onward), modelled on `:3303-3311`.

All three are required. Adding only (1) leaves late compensation broken; adding only (2) declares an inverse that does not exist.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/
git commit -m "feat(saga-orchestrator): compensate AwardCraftedAsset steps"
```

---

## Task 15: `atlas-maker` — service skeleton

Design §4.2. A new domain service structured after `atlas-reward-pools`, the closest existing shape: a domain service with seeded reference data, a compute-only subdomain, and no ownership of mutable game state.

### Files

- `services/atlas-maker/atlas.com/maker/go.mod` — new file
- `services/atlas-maker/atlas.com/maker/main.go` — new file; bootstrap
- `services/atlas-maker/atlas.com/maker/wiring_test.go` — new file; route-wiring smoke test
- `services/atlas-maker/atlas.com/maker/README.md` — new file
- `go.work` — add the new module to `use()`

Module root for `go build`/`go test`: `services/atlas-maker/atlas.com/maker`.

Patterns to copy: `services/atlas-reward-pools/atlas.com/reward-pools/main.go:41-74` (`func main()` starts exactly at line 41) and `services/atlas-reward-pools/atlas.com/reward-pools/wiring_test.go`.

`atlas-maker` is a free service name — verified against the full `services/` listing.

### Interfaces

- Consumes: nothing from earlier tasks.
- Produces, for Tasks 16-24: a buildable module with a `main.go` whose `database.Connect` migration list and `AddRouteInitializer` chain later tasks append to.

### Why a database is still required

The service stores no craft rows (design §4.2.6 — OQ-4 resolved as "no audit table"; the saga record in `atlas-saga-orchestrator` is the durable history). But the `reagent` and crystal-band tables (Tasks 17-18) are real tenant-owned tables, so the service takes the standard `database.Connect` + `seeder.SeedState` bootstrap from `reward-pools/main.go:41-74`.

### Upstream service tokens

`atlas-maker` reads from five upstreams. Cross-service clients resolve their base URL via `requests.RootUrlFor(ctx, "<TOKEN>")`, and the tokens this service needs are:

| Upstream | Token | Used for |
|---|---|---|
| `atlas-data` | `DATA` | recipes (`data/item-makes`), equip `reqLevel` (`data/equipment/{id}`) |
| `atlas-character` | `CHARACTERS` | level, mesos |
| `atlas-skills` | `SKILLS` | Maker skill level |
| `atlas-inventory` | `INVENTORY` | compartment snapshots, accommodation check |
| `atlas-quests` | `QUESTS` | `reqQuest` state |

- [ ] **Step 1: Write the failing test**

Create `services/atlas-maker/atlas.com/maker/wiring_test.go`, copying the shape of `services/atlas-reward-pools/atlas.com/reward-pools/wiring_test.go`.

`TestServiceBootstraps` — construct the router the way `main.go` does and assert it builds without panicking and that the service's base route prefix resolves. Keep it a smoke test; the domain tests live with their domains.

- [ ] **Step 2: Run the test to verify it fails**

Run from `services/atlas-maker/atlas.com/maker`:

```
go test ./... -count=1
```

Expected: FAIL — no such module / no Go files.

- [ ] **Step 3: Write `go.mod` and add the module to `go.work`**

Copy `services/atlas-reward-pools/atlas.com/reward-pools/go.mod`, changing the module name to `atlas-maker`. Keep the `replace` directives for the `libs/*` modules identical to reward-pools'; the Go version must match the repo's `go1.27.0` toolchain.

Add to `go.work`'s `use()` block:

```
	./services/atlas-maker/atlas.com/maker
```

- [ ] **Step 4: Write `main.go`**

Copy `services/atlas-reward-pools/atlas.com/reward-pools/main.go:41-74` and adapt: the service name, the `database.Connect(l, database.SetMigrations(...))` migration list (empty for now; Tasks 17-18 append), the `service.Bootstrap` + `server.New(l)` chain, and the `AddRouteInitializer` chain (empty for now; Tasks 17, 18 and 24 append).

Declare the `consumerGroupId` literal that `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh` derives the PR-env `KAFKA_CONSUMER_GROUP` value from, matching reward-pools' declaration site.

- [ ] **Step 5: Write the README**

`services/atlas-maker/atlas.com/maker/README.md`, following the structure of reward-pools' README: purpose, REST endpoints table (populate as Tasks 17-24 add them), Kafka commands/events table, and the upstream dependency list from the table above.

- [ ] **Step 6: Run the test to verify it passes**

Run from `services/atlas-maker/atlas.com/maker`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maker/ go.work
git commit -m "feat(maker): scaffold atlas-maker service"
```

---

## Task 16: `atlas-maker` — build, deploy, and ingress registration

Design risk **R-3**. `docs/adding-a-new-service.md` exists because atlas-mts (task-121) was wired into CI, the k8s base and the PR overlay but missed all four main-overlay enumerations, producing crash-looping pods, an unpinnable `:latest` image, and silently unsuffixed Kafka topics. **None of these lists are derived from each other**, and several fail silently.

This task is deliberately larger than the ~6-file guideline: it is the same mechanical registration repeated across independent hand-maintained lists, which batches well and does not decompose into meaningfully reviewable halves. Splitting it would create tasks that are individually green and jointly broken.

### Files

- `.github/config/services.json` — add the service entry
- `docker-bake.hcl` — add `"atlas-maker"` to `go_services`
- `deploy/k8s/base/atlas-maker.yaml` — new file; Deployment + Service
- `deploy/k8s/base/kustomization.yaml` — add the manifest to `resources:`
- `deploy/k8s/overlays/main/kustomization.yaml` — `images:` pin
- `deploy/k8s/overlays/pr/kustomization.yaml` — `images:` pin and `ATLAS_DB_NAMES`
- `deploy/shared/routes.conf` — the nginx location block
- `tools/db-bootstrap.sh` — add the unsuffixed DB name

Also touched, all **generator-owned — never hand-edited**: `deploy/k8s/overlays/main/patches/db-name-suffix.yaml`, `deploy/k8s/overlays/main/patches/atlas-env-env.yaml`, `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml`, `deploy/k8s/overlays/pr/patches/consumer-group-env.yaml`, `deploy/k8s/base/routes.conf.template.generated`, `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`.

Read-only reference: `docs/adding-a-new-service.md` — work through every applicable row.

### Interfaces

- Consumes: Task 15's module and its `consumerGroupId` literal.
- Produces: a service that builds in CI, renders in both overlays with correct values, and is reachable through the ingress.

### The four silent-failure traps

1. **`images:` bump is a no-op for a missing entry.** The main-publish workflow runs `yq '(.images[] | select(.name == …) | .newTag) = …'`. No entry means nothing is written and **no error is raised** — the service runs `:latest` forever.
2. **`configMapGenerator` with `behavior: replace` drops unlisted keys.** The overlay replaces `env-configmap.yaml` rather than merging with it. A topic var in base but not in the overlay literals simply does not exist in that environment.
3. **Missing topic env vars don't crash.** `libs/atlas-kafka/topic/provider.go` falls back to the token itself with only a warn log, so the service silently produces/consumes on the unsuffixed topic. It "works" until one side gets the var.
4. **`DB_NAME` without the env suffix crash-loops** (`SQLSTATE 3D000`) — the only loud one.

- [ ] **Step 1: Build and CI**

`.github/config/services.json` — add to `services[]` with `name`, `type: go-service`, `path`, `module_path`, `docker_image`, `docker_context: "."`. Both `main-publish.yml` and `pr-validation.yml` read this dynamically.

`docker-bake.hcl` — add `"atlas-maker"` to the hardcoded `go_services` list. It is **hand-synced** with `services.json`; adding to one does not add to the other.

`go.work` was done in Task 15.

Verify the image target builds:

```
docker buildx bake atlas-maker
```

- [ ] **Step 2: Kubernetes base**

`deploy/k8s/base/atlas-maker.yaml` — Deployment + Service, copied from an existing DB-backed service's manifest. No `namespace:` (overlays set it). `DB_NAME` gets the **unsuffixed** base value `atlas-maker`. The container `name:` is the short name `maker` — the overlay patches match on it.

`deploy/k8s/base/kustomization.yaml` — add `atlas-maker.yaml` to `resources:`.

`deploy/k8s/base/env-configmap.yaml` — add any **new** Kafka topic env var this service introduces as `KEY: "KEY"`. The guard cannot check this (it only enforces parity of keys already present), so confirm by hand against `main.go`'s topic references. If the service introduces no new topic, make no change and say so in the commit message.

Because the service consumes seed data (Tasks 17-18), add the `atlas.seed-catalog: "true"` label to the Deployment — the `components/seed-catalog` kustomize component injects the git-sync sidecar and `SEED_CATALOG_ROOT` automatically. The guard cannot check this either.

- [ ] **Step 3: Main overlay**

`deploy/k8s/overlays/main/patches/db-name-suffix.yaml` — a new patch document setting `DB_NAME: "atlas-maker-main"`, targeting container `maker`.

`deploy/k8s/overlays/main/patches/atlas-env-env.yaml` — a new patch document setting `ATLAS_ENV: "main"`.

`deploy/k8s/overlays/main/kustomization.yaml` — add to `images:`:

```yaml
  - name: ghcr.io/chronicle20/atlas-maker/atlas-maker
    newTag: main-<sha>
```

Set `newTag` to the current fleet tag and confirm it exists on GHCR before committing (`docker manifest inspect`).

Add every topic var from Step 2 to the `configMapGenerator` literals as `KEY=KEY-main`. Do **not** add `KAFKA_CONSUMER_GROUP` on main — it is intentionally not injected there; see the comment at the top of the main kustomization.

- [ ] **Step 4: PR overlay**

`deploy/k8s/overlays/pr/kustomization.yaml` — add `atlas-maker` to the `ATLAS_DB_NAMES` literal (this one list drives both the wave-0 create-DBs job and teardown), and add the same `images:` entry shape as Step 3.

Then **re-run the generators** rather than hand-editing their output:

```
deploy/k8s/overlays/pr/scripts/gen-topic-config.sh
deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh
deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
```

Paste `gen-topic-config.sh`'s output into the atlas-env generator block. The other three rewrite their own files. `pr-validation.yml` **hard-fails the PR** when the committed `atlas-pr-cleanup-env.example.yaml` is stale.

- [ ] **Step 5: Ingress**

`deploy/shared/routes.conf` — add an nginx location block, alphabetically placed, using the bare container name:

```nginx
location ~ ^/api/maker(/.*)?$ {
  proxy_pass http://atlas-maker:8080;
}
```

Then regenerate and commit both files:

```
tools/gen-routes.sh
```

`deploy/shared/test/routes_nginxt.sh` drift-checks the pair but is docker-based and **operator-run — nothing in CI invokes it**, so a stale generated file will not fail the PR. Run `tools/gen-routes.sh` and commit its output.

- [ ] **Step 6: Databases**

`tools/db-bootstrap.sh` — add the **unsuffixed** name `atlas-maker` to the hand-edited `DBS` list.

PR envs need nothing beyond Step 4's `ATLAS_DB_NAMES`.

- [ ] **Step 7: Run the guard and the render checks**

Run from the repo root:

```
tools/service-registration-guard.sh
```

Expected: exit 0. It machine-checks §1 (docker-bake, go.work), §2.1-2.2 (base manifest, kustomization resources), §3 (main overlay images pin, `ATLAS_ENV=main`, `DB_NAME=atlas-maker-main` — values, not just presence), §4 (pr images, per-doc `DB_NAME`, `ATLAS_DB_NAMES`, consumer-group doc), §6.2 (db-bootstrap list), atlas-env configmap key parity, and patch-doc container names against the base manifest.

Then the checks it cannot do:

```
kubectl kustomize deploy/k8s/overlays/main | grep -B2 -A6 "name: atlas-maker$"
kubectl kustomize deploy/k8s/overlays/pr > /dev/null
docker buildx bake atlas-maker
```

The main render must show `DB_NAME=atlas-maker-main`, `ATLAS_ENV=main`, and the image pinned to `main-<sha>` — not `:latest`.

- [ ] **Step 8: Record the two out-of-repo manual steps**

Neither can be done from this branch, and both cause runtime-only failures. Record them in `docs/tasks/task-285-maker-skill-crafting/context.md` under a "Rollout" heading and repeat them in the PR description:

1. **Create `atlas-maker-main` on postgres.home before merging.** Main has no wave-0 create job; the pods crash-loop on `SQLSTATE 3D000` until the database exists. Owner = the app role; `uuid-ossp` is inherited from `template1`.
2. **Flip the new GHCR package to public after the first image push.** The first `docker buildx bake` push creates `ghcr.io/chronicle20/atlas-maker/atlas-maker` as **private**, and the cluster pulls anonymously — no `imagePullSecrets` on any Deployment. CI reports a clean build while the pod sits in `ImagePullBackOff` against a 401. Verify with:

```bash
curl -s -o /dev/null -w '%{http_code}\n' \
  'https://ghcr.io/token?scope=repository%3Achronicle20%2Fatlas-maker%2Fatlas-maker%3Apull&service=ghcr.io'
```

200 means public; 401 means still private.

- [ ] **Step 9: Commit**

```bash
git add .github/config/services.json docker-bake.hcl deploy/ tools/db-bootstrap.sh dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml docs/tasks/task-285-maker-skill-crafting/context.md
git commit -m "chore(maker): register atlas-maker across CI, k8s, and ingress"
```

---

## Task 17: `atlas-maker` — the `reagent` seeded table

Design §4.2.3, resolving **OQ-5**. The gem/reagent → `(stat, value)` mapping has no `ItemMake.img` source. Decision: a tenant-owned seeded table in `atlas-maker`, exposed read-only, retunable per tenant through the seed catalog like every other seeded domain.

**Dispatch this task with `model: opus`** — Step 1 is IDA derivation.

### Files

- `services/atlas-maker/atlas.com/maker/reagent/entity.go` — new file; GORM entity and migration
- `services/atlas-maker/atlas.com/maker/reagent/model.go` — new file; immutable model with accessors
- `services/atlas-maker/atlas.com/maker/reagent/builder.go` — new file; fluent builder
- `services/atlas-maker/atlas.com/maker/reagent/administrator.go` — new file; writes
- `services/atlas-maker/atlas.com/maker/reagent/provider.go` — new file; lazy reads
- `services/atlas-maker/atlas.com/maker/reagent/processor.go` — new file; business logic

Continued in the same task (the remaining files of one scaffolded domain package): `rest.go`, `resource.go`, `subdomain.go`, `mock/processor.go`, `builder_test.go`, `processor_test.go`, `resource_test.go`; plus `services/atlas-maker/atlas.com/maker/main.go` for the migration and route registration.

Module root for `go build`/`go test`: `services/atlas-maker/atlas.com/maker`.

Patterns to copy: `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/` — the full 12-file seeded domain package (`administrator.go`, `builder.go`, `builder_test.go`, `entity.go`, `model.go`, `processor.go`, `processor_test.go`, `provider.go`, `resource.go`, `resource_test.go`, `rest.go`, `subdomain.go`). Build in dependency order: `model.go` → `entity.go` → `builder.go` → processor and provider → `rest.go` → `resource.go` → tests.

### Interfaces

- Consumes: Task 15's `main.go`.
- Produces, for Tasks 22-23: `reagent.Model` with `ReagentItemId() item.Id`, `Stat() string`, `Value() int16`; and `reagent.Processor.GetByItemId(item.Id) (Model, error)` plus a bulk `GetAll()`.

### Schema

```
reagents(tenant_id, reagent_item_id, stat, value)
```

`stat` is the affected equip stat; `value` its delta. Tenant-scoped through the standard tenant context — no cross-tenant read is possible.

### The seed content is derived, not invented

The client owns the authoritative mapping. `CItemMakerInfo::Load_GemEffect` reads it out of the WZ archive:

| Version | Address |
|---|---|
| `gms_v72` | `0x5a2cf5` |
| `gms_v83` | `0x5e6f4c` |

Derive the source node and its field names from that function, then generate the seed from the archive. This is the same standard applied to every other game-data value in this repo — a hand-written table of plausible stat deltas is exactly the invented value CLAUDE.md forbids.

- [ ] **Step 1: Derive the reagent mapping from the client (opus)**

Decompile `CItemMakerInfo::Load_GemEffect` at the two addresses above. Record: the WZ node path it reads, the child field names it pulls per gem, and the value semantics (absolute vs delta, signed vs unsigned).

Write the findings to `docs/tasks/task-285-maker-skill-crafting/reagent-derivation.md`, including the full derived `(reagentItemId, stat, value)` table and the address and node path each row came from.

If the node the function reads is **not** present in the local reference dump, stop and report that specifically — it is an external blocker of the same class as R-2, not something to fill with plausible values.

- [ ] **Step 2: Write the failing tests**

Create `services/atlas-maker/atlas.com/maker/reagent/builder_test.go` and `processor_test.go`, copying the setup shape from `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/builder_test.go` and `processor_test.go`. Use the Builder pattern for fixtures — no `*_testhelpers.go`.

`TestBuilderRejectsUnknownStat` — assert the builder returns an error for a stat name outside the derived set. Enumerate the valid stat names as a package-level slice so the test and the validation share one source.

`TestBuilderRoundTrip` — build a model, assert every accessor returns what was set.

`TestGetByItemIdReturnsSeededReagent` — seed one row from Step 1's derived table, assert `GetByItemId` returns it with the exact stat and value.

`TestGetByItemIdIsTenantScoped` — seed the same `reagent_item_id` under two tenants with different values; assert each tenant's context reads only its own row. This is the PRD §8 multi-tenancy requirement made executable.

`TestGetByItemIdNotFound` — assert a distinguishable not-found error for an unseeded id. Task 23 relies on this to drop unheld reagents rather than fail the craft (FR-3.2).

`TestResourceIsReadOnly` — assert `POST`, `PUT`, `PATCH` and `DELETE` against the reagent routes return 405. FR-2.3's read-only rule applies to reference data generally; a writable reagent table would let a client retune its own stat bonuses.

- [ ] **Step 3: Run the tests to verify they fail**

Run from `services/atlas-maker/atlas.com/maker`:

```
go test ./reagent/... -count=1
```

Expected: FAIL — no such package.

- [ ] **Step 4: Write the domain package**

In dependency order, copying `gachapon/`'s file-by-file structure:

`model.go` — immutable `Model` with unexported fields and accessors; no setters.
`entity.go` — the GORM entity with the `reagents` table name, a `uniqueIndex` on `(tenant_id, reagent_item_id)`, and `Migration(db *gorm.DB) error`.
`builder.go` — fluent builder enforcing the stat-name invariant.
`provider.go` — lazy tenant-scoped reads.
`administrator.go` — writes, used only by the seeder.
`processor.go` — `Processor` interface, `NewProcessor`, `GetByItemId`, `GetAll`.
`mock/processor.go` — written in the same step as the interface.
`rest.go` — `RestModel` with `GetName() string { return "reagents" }`.
`resource.go` — read-only routes; the collection route paginates.
`subdomain.go` — the seed-catalog subdomain registration, mirroring `gachapon/subdomain.go`.

- [ ] **Step 5: Wire `main.go`**

Add `reagent.Migration` to the `database.Connect(l, database.SetMigrations(...))` list and `AddRouteInitializer(reagent.InitResource(db)(GetServer()))` to the chain.

- [ ] **Step 6: Run the tests to verify they pass**

Run from `services/atlas-maker/atlas.com/maker`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maker/atlas.com/maker/ docs/tasks/task-285-maker-skill-crafting/reagent-derivation.md
git commit -m "feat(maker): add tenant-owned reagent stat table seeded from client data"
```

---

## Task 18: `atlas-maker` — the crystal level-band table and seed groups

Design §4.2.4, resolving **OQ-2**. `ItemMake.img` does not encode disassembly. Decision: derive the crystal yield from the equip's level requirement, with the band table itself derived from the client.

**Dispatch this task with `model: opus`** — Step 1 is IDA derivation.

### Files

- `services/atlas-maker/atlas.com/maker/crystalband/entity.go` — new file; entity and migration
- `services/atlas-maker/atlas.com/maker/crystalband/model.go` — new file
- `services/atlas-maker/atlas.com/maker/crystalband/builder.go` — new file
- `services/atlas-maker/atlas.com/maker/crystalband/processor.go` — new file; `CrystalForLevel`
- `services/atlas-maker/atlas.com/maker/crystalband/processor_test.go` — new file
- `services/atlas-maker/atlas.com/maker/seed/groups.go` — new file; seed-catalog group registration

Continued in the same task: `crystalband/provider.go`, `administrator.go`, `rest.go`, `resource.go`, `subdomain.go`, `mock/processor.go`, `builder_test.go`; plus `seed/groups_test.go` and `main.go`.

Module root for `go build`/`go test`: `services/atlas-maker/atlas.com/maker`.

Patterns to copy: the same `gachapon/` package shape as Task 17, and `services/atlas-reward-pools/atlas.com/reward-pools/seed/groups.go` (32 lines) for the seed-catalog group registration — it registers exactly one group via `seeder.RegisterRoutes(router, db, l, src, seeder.Group{Name: ..., URLPrefix: ..., Subdomains: [...]})`.

### Interfaces

- Consumes: Task 17's package layout conventions.
- Produces, for Task 23: `crystalband.Processor.CrystalForLevel(reqLevel uint32) (item.Id, uint32, error)` returning the crystal template and the count for an equip of that level requirement.

### No new `atlas-data` surface is needed

`GET /data/equipment/{equipmentId}` already exposes `reqLevel` (`equipment/rest.go:33`, populated at `equipment/reader.go:103`), so disassembly reads the level requirement from an endpoint that already exists.

### The band table is derived, not invented

`CItemMakerInfo::Load_MonsterCrystalLevel` is the client's own loader for exactly this table:

| Version | Address |
|---|---|
| `gms_v72` | `0x5a3033` |
| `gms_v83` | `0x5e728a` |

- [ ] **Step 1: Derive the level-band table from the client (opus)**

Decompile `Load_MonsterCrystalLevel` at both addresses. Record the band boundaries and the crystal item id each band maps to, plus the WZ node path the function reads.

Append the findings to `docs/tasks/task-285-maker-skill-crafting/reagent-derivation.md` under a "Monster crystal level bands" heading, with the derived table and the address each row came from. As in Task 17, if the source node is absent from the local dump, report that rather than inventing boundaries.

- [ ] **Step 2: Write the failing tests**

Create `services/atlas-maker/atlas.com/maker/crystalband/processor_test.go`.

`TestCrystalForLevelAtEachBand` — table-driven, one case per band from Step 1's derived table, asserting the exact crystal id and count.

`TestCrystalForLevelAtBandBoundaries` — for every boundary `n` in the derived table, assert `CrystalForLevel(n-1)`, `CrystalForLevel(n)` and `CrystalForLevel(n+1)` land in the expected bands. Off-by-one at a band edge is the defect this table's shape invites.

`TestCrystalForLevelBelowLowestBand` — assert the behaviour the client exhibits for an equip below the lowest band. Take this from the decompilation, not from a guess; record which it is in the test comment.

`TestCrystalForLevelIsTenantScoped` — seed differing bands for two tenants and assert each reads its own.

Create `services/atlas-maker/atlas.com/maker/seed/groups_test.go`, copying `reward-pools/seed/groups_test.go`. Assert both the `reagents` and `crystalBands` groups are registered with the expected URL prefixes and subdomains.

- [ ] **Step 3: Run the tests to verify they fail**

Run from `services/atlas-maker/atlas.com/maker`:

```
go test ./crystalband/... ./seed/... -count=1
```

Expected: FAIL — no such packages.

- [ ] **Step 4: Write the `crystalband` package**

Same file set and dependency order as Task 17. The entity's table is `crystal_bands(tenant_id, min_level, max_level, crystal_item_id, count)` with a `uniqueIndex` on `(tenant_id, min_level)`.

`CrystalForLevel` selects the band containing the level. Load the bands once per tenant and resolve in memory; do not issue a query per lookup.

- [ ] **Step 5: Write `seed/groups.go`**

Register both seeded domains, mirroring `reward-pools/seed/groups.go`'s single-group call:

```go
seeder.RegisterRoutes(router, db, l, src,
	seeder.Group{Name: "reagents", URLPrefix: "/reagents", Subdomains: []seeder.Subdomain{reagent.Subdomain()}},
	seeder.Group{Name: "crystalBands", URLPrefix: "/crystal-bands", Subdomains: []seeder.Subdomain{crystalband.Subdomain()}},
)
```

Read `reward-pools/seed/groups.go` first and match its exact call signature — the `RegisterRoutes` arity and the `Group`/`Subdomain` field names must come from that file, not from this sketch.

- [ ] **Step 6: Wire `main.go`**

Add `crystalband.Migration` to the migration list, `AddRouteInitializer(crystalband.InitResource(db)(GetServer()))` to the chain, and `seed.InitResource` following reward-pools' placement.

- [ ] **Step 7: Run the tests to verify they pass**

Run from `services/atlas-maker/atlas.com/maker`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-maker/atlas.com/maker/ docs/tasks/task-285-maker-skill-crafting/reagent-derivation.md
git commit -m "feat(maker): derive monster-crystal level bands and register seed groups"
```

---

## Task 19: `atlas-maker` — upstream REST clients

Design §4.2's `data/…` row. Six per-upstream clients following the repo's per-service `requests.go` + `rest.go` convention.

This task is deliberately larger than the ~6-file guideline: it is one mechanical two-file pattern repeated six times, with no shared state between the pairs.

### Files

- `services/atlas-maker/atlas.com/maker/data/itemmake/requests.go` + `rest.go` + `processor.go` — new files; `atlas-data` recipes
- `services/atlas-maker/atlas.com/maker/data/equipment/requests.go` + `rest.go` + `processor.go` — new files; equip `reqLevel`
- `services/atlas-maker/atlas.com/maker/character/requests.go` + `rest.go` + `processor.go` — new files; level and mesos
- `services/atlas-maker/atlas.com/maker/skill/requests.go` + `rest.go` + `processor.go` — new files; Maker skill level
- `services/atlas-maker/atlas.com/maker/compartment/requests.go` + `rest.go` + `processor.go` — new files; inventory snapshot and accommodation
- `services/atlas-maker/atlas.com/maker/quest/requests.go` + `rest.go` + `processor.go` — new files; `reqQuest` state

Plus a `mock/processor.go` and a `processor_test.go` per package.

Module root for `go build`/`go test`: `services/atlas-maker/atlas.com/maker`.

Patterns to copy, one per client — all six already exist in `atlas-channel` and are the exact shape to mirror:

| New package | Copy from | Token |
|---|---|---|
| `data/equipment` | `services/atlas-channel/atlas.com/channel/data/equipment/requests.go` | `DATA` |
| `data/itemmake` | `services/atlas-channel/atlas.com/channel/data/commodity/requests.go` | `DATA` |
| `character` | `services/atlas-channel/atlas.com/channel/character/requests.go` | `CHARACTERS` |
| `skill` | `services/atlas-consumables/atlas.com/consumables/skill/requests.go` | `SKILLS` |
| `compartment` | `services/atlas-channel/atlas.com/channel/compartment/requests.go` | `INVENTORY` |
| `quest` | `services/atlas-channel/atlas.com/channel/quest/requests.go` | `QUESTS` |

### Interfaces

- Consumes: Task 4's `/data/item-makes` routes.
- Produces, for Tasks 20-23:
  - `itemmake.Processor.GetAll() ([]Model, error)` and `GetById(item.Id) (Model, error)`
  - `equipment.Processor.GetById(item.Id) (Model, error)` exposing `ReqLevel() uint32`
  - `character.Processor.GetById(uint32) (Model, error)` exposing `Level() byte` and `Meso() uint32`
  - `skill.Processor.GetByCharacterId(uint32) ([]Model, error)`
  - `compartment.Processor.GetByType(uint32, inventory.Type) (Model, error)` and `CanAccommodate(uint32, []AccommodationItem) (bool, error)`
  - `quest.Processor.GetByCharacterId(uint32) ([]Model, error)`

### Two paginated upstreams

`atlas-skills`' and `atlas-quests`' list endpoints are **paginated** (task-117). Both `atlas-channel` clients handle this by exposing a bare URL rather than a `requests.Request` and draining it with `requests.DrainProvider`, which appends its own `page[number]`/`page[size]` params per request:

```go
// characterSkillsUrl returns the list URL for a character's skills. The
// atlas-skills list endpoint is paginated (task-117); it is consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func characterSkillsUrl(ctx context.Context, characterId uint32) (string, error) {
```

Copy that shape for both. A single `GetRequest` against either returns page one only and silently truncates — which for skills means a character's Maker skill can go missing and every craft is rejected with `skill_level_too_low`.

### The accommodation endpoint already exists

`atlas-inventory` exposes `POST characters/{characterId}/inventory/accommodation`, which takes a **list** of `{itemId, quantity}` and returns an overall verdict plus a per-item breakdown:

```go
type accommodationOutputRestModel struct {
	Id           string                         `json:"-"`
	Accommodated bool                           `json:"accommodated"`
	Results      []accommodationResultRestModel `json:"results"`
}

type accommodationResultRestModel struct {
	ItemId       uint32 `json:"itemId"`
	Quantity     uint32 `json:"quantity"`
	Accommodated bool   `json:"accommodated"`
}
```

This answers FR-3.6 directly and in one call for the whole award set. `atlas-maker` uses it rather than reimplementing free-slot arithmetic against a compartment snapshot — the repo already owns this computation, and duplicating it would drift. The design's §4.2.2 step 6 ("free-slot capacity for every award, computed before any mutation") is satisfied by this call. Copy the request and both rest models from `services/atlas-channel/atlas.com/channel/compartment/requests.go:28-39` and `rest.go:131-170`.

### Compartment reads are per-type

`atlas-inventory`'s `handleGetCompartmentByType` **requires** the `type` query param and returns 400 without it (`services/atlas-inventory/atlas.com/inventory/compartment/resource.go:67-72`). There is no batched all-compartments endpoint. The snapshot is therefore three calls — EQUIP, USE, ETC — not one. The client exposes:

```go
const (
	Resource = "characters/%d/inventory/compartments"
	ByType   = Resource + "?type=%d"
)
```

`inventory.TypeValueEquip` is `1`, `TypeValueUse` is `2`, `TypeValueETC` is `4` (`libs/atlas-constants/inventory/constants.go:12-19`). Use the constants, never the literals.

- [ ] **Step 1: Write the failing tests**

One `processor_test.go` per package, each stubbing the upstream with an `httptest` server and asserting the client parses the JSON:API response into the model. Follow the test shape used by an existing service's client tests.

Per package, at minimum:

- a happy-path parse asserting every field the consumer uses
- a 404 producing a distinguishable not-found error
- for `skill` and `quest`: a **two-page** stub asserting the drain collects entries from both pages. A test that stubs only one page proves nothing about the pagination handling and is the defect this test exists to prevent.
- for `compartment`: a `CanAccommodate` test asserting a multi-item request round-trips and that `Accommodated == false` is surfaced, plus a per-item `Results` assertion.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-maker/atlas.com/maker`:

```
go test ./data/... ./character/... ./skill/... ./compartment/... ./quest/... -count=1
```

Expected: FAIL — no such packages.

- [ ] **Step 3: Write the six client packages**

For each: `requests.go` (the `getBaseRequest` + per-route request functions), `rest.go` (the `RestModel` mirroring the upstream's, with only the fields this service consumes), `processor.go` (the `Processor` interface, `NewProcessor`, and the model transform), and `mock/processor.go`.

Mirror only the upstream fields actually used. A `RestModel` that mirrors an upstream wholesale becomes a maintenance liability the first time that upstream adds a field.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-maker/atlas.com/maker`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maker/atlas.com/maker/
git commit -m "feat(maker): add upstream REST clients for data, character, skills, inventory, quests"
```

---

## Task 20: `atlas-maker` — the `recipe` cache and its indexes

Design §4.2.1, resolving **OQ-3**.

### Files

- `services/atlas-maker/atlas.com/maker/recipe/model.go` — new file; the recipe model
- `services/atlas-maker/atlas.com/maker/recipe/processor.go` — new file; the cache and both indexes
- `services/atlas-maker/atlas.com/maker/recipe/processor_test.go` — new file
- `services/atlas-maker/atlas.com/maker/recipe/mock/processor.go` — new file

Module root for `go build`/`go test`: `services/atlas-maker/atlas.com/maker`.

Patterns to copy: `services/atlas-reward-pools/atlas.com/reward-pools/reward/` — a compute-only domain with no entity, no administrator and no table (`builder.go` 50, `model.go` 37, `processor.go` 277, `resource.go` 116, `rest.go` 38).

### Interfaces

- Consumes: `data/itemmake.Processor` (Task 19).
- Produces, for Tasks 21-23:
  - `recipe.Model` with `Id() item.Id`, `Group() uint32`, `ReqLevel() uint32`, `ReqSkillLevel() uint32`, `ItemNum() uint32`, `Tuc() uint32`, `Meso() uint32`, `Catalyst() item.Id`, `ReqItem() item.Id`, `ReqEquip() item.Id`, `Materials() []Material`, `RandomRewards() []Reward`, `QuestRequirements() []QuestRequirement`
  - `recipe.Processor.GetById(item.Id) (Model, error)`
  - `recipe.Processor.GetByLeftover(item.Id) (Model, error)` — **group `0` only**
  - `recipe.Processor.GetAll() ([]Model, error)`

### The two indexes

The recipe set is a few thousand immutable rows per tenant, built lazily per tenant and invalidated on the seed/ingestion signal:

- `byItemId map[item.Id]Model` — mode 1/2 and the eligibility listing.
- `byLeftover map[item.Id]Model` — **restricted to group `0`**. This is why C-6 persists the group digit: without it, an arbitrary recipe that happens to list the leftover as a material would match, and a player could disassemble-to-crystal through an unrelated equip recipe.

### OQ-3 is resolved as "no drop-table join"

The `ItemMake.img` group `0` directory alone is sufficient. Its entries already pair the leftover as the sole `recipe` material with the crystal tiers as `randomReward` outcomes. The reference server maps leftover → crystal via `drop_data`, but Atlas does not need to: `GET /items/{itemId}/drops` exists at `services/atlas-drop-information/atlas.com/dis/monster/drop/resource.go:26-27` if a future cross-check is ever wanted, **but it is not a dependency of this design** and must not be called here.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-maker/atlas.com/maker/recipe/processor_test.go`. Stub `data/itemmake.Processor` with its mock and seed the six-group fixture from Task 3.

`TestGetByIdReturnsRecipe` — assert `GetById(1082002)` returns a model whose scalars and ordered lists match the fixture, element by element.

`TestGetByIdNotFound` — assert a distinguishable not-found error, which Task 24 maps to `recipe_not_found` (404).

`TestGetByLeftoverResolvesGroupZero` — seed the group-`0` entry `4260000` whose sole material is leftover `4000000`; assert `GetByLeftover(4000000)` returns recipe `4260000`.

`TestGetByLeftoverIgnoresNonZeroGroups` — seed a **group-1** recipe that lists `4000000` among its materials, and assert `GetByLeftover(4000000)` still returns only the group-`0` recipe. Then remove the group-`0` entry and assert `GetByLeftover(4000000)` returns not-found rather than the group-1 recipe. This is the C-6 requirement made executable and is the single most important test in this task.

`TestGetByLeftoverNotFound` — assert an unmapped leftover returns a distinguishable error, which Task 24 maps to `no_crystal_mapping` (422).

`TestIndexesAreTenantScoped` — build indexes under two tenant contexts with different recipe sets; assert neither sees the other's.

`TestIndexIsBuiltOnceUntilInvalidated` — count upstream calls across repeated lookups; assert the upstream is read once, then again after invalidation.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-maker/atlas.com/maker`:

```
go test ./recipe/... -count=1
```

Expected: FAIL — no such package.

- [ ] **Step 3: Write the package**

`model.go` — the immutable model and its `Material{ItemId item.Id; Count uint32}`, `Reward{ItemId item.Id; ItemNum uint32; Prob uint32}`, `QuestRequirement{QuestId uint32; State uint32}` value types. Preserve list order from the REST model; never sort.

`processor.go` — the lazy per-tenant index build, guarded by a `sync.RWMutex` (`sync.Once` is wrong here: the cache must be invalidatable). `GetByLeftover` consults only entries with `Group() == 0`.

`mock/processor.go` — in the same step as the interface.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-maker/atlas.com/maker`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maker/atlas.com/maker/recipe/
git commit -m "feat(maker): index recipes by item id and by group-zero leftover"
```

---

## Task 21: `atlas-maker` — eligibility evaluation

FR-2.1, FR-2.2, FR-3.5, and design §4.2.2.

### Files

- `services/atlas-maker/atlas.com/maker/craft/eligibility.go` — new file; the check order and its result type
- `services/atlas-maker/atlas.com/maker/craft/snapshot.go` — new file; the inventory snapshot
- `services/atlas-maker/atlas.com/maker/craft/eligibility_test.go` — new file
- `services/atlas-maker/atlas.com/maker/craft/snapshot_test.go` — new file

Module root for `go build`/`go test`: `services/atlas-maker/atlas.com/maker`.

### Interfaces

- Consumes: `recipe.Processor` (Task 20); `character`, `skill`, `compartment`, `quest` processors (Task 19).
- Produces, for Tasks 23-24: `craft.Eligibility` carrying an `Eligible bool` and, when false, a `Reason` that maps 1:1 onto PRD §5's error codes; and `craft.Snapshot` with `Held(item.Id) uint32` and `Slots(item.Id) []SlotHolding`.

### The Maker skill identities, verified

All four exist in `libs/atlas-constants/skill/identities_gen.go` with these exact names and values:

| Constant | Value | Line |
|---|---|---|
| `skill.BeginnerMaker` | `1007` | `:13` |
| `skill.NoblesseMaker` | `10001007` | `:400` |
| `skill.LegendMaker` | `20001007` | `:520` |
| `skill.EvanMaker` | `20011007` | `:536` |

Use these constants. Do not define a maker-skill list in the service — DOM-21 requires checking `libs/atlas-constants/` first, and this is exactly what it holds.

### Check order — cheapest first

Ordered so the expensive reads are skipped for most recipes:

1. `reqLevel` vs the character's level; `reqSkillLevel` vs the Maker variant's level. FR-3.5 requires level ≥ 1 for **any** craft, checked once up front before per-recipe work.
2. `reqItem` / `reqEquip` against the snapshot.
3. `reqQuest` against `atlas-quests` — **only** for the few recipes that carry one (C-5; two in the reference archive).
4. Every `recipe` material at its `count`, summed across slots in the snapshot.
5. `meso` vs the character's mesos.
6. Award accommodation via `compartment.Processor.CanAccommodate` (FR-3.6).

### The snapshot satisfies the batched-read NFR

`atlas-inventory` has no batched all-compartments endpoint, so the NFR's "single batched inventory read, not one read per recipe" is satisfied by reading each *inventory type* once (EQUIP, USE, ETC) into a snapshot, then evaluating every candidate recipe against that snapshot **in memory**. Three upstream calls total, not one per recipe.

- [ ] **Step 1: Write the failing tests**

Create both test files, mocking all five upstream processors.

`TestSnapshotSumsQuantityAcrossSlots` — an ETC compartment holding item `4011001` as 3 in slot 1 and 4 in slot 7; assert `Held(4011001) == 7` and that `Slots(4011001)` returns both slots in slot order with their individual quantities. The per-slot detail is what Task 23's consumption plan consumes.

`TestSnapshotReadsEachTypeExactlyOnce` — assert exactly three upstream calls for a snapshot build, one per type, regardless of how many recipes are then evaluated. This is the NFR made executable.

`TestEligibilityOnePerExclusionReason` — table-driven, **one case per exclusion reason**, as PRD §10 requires. Each case starts from a fully-eligible fixture and breaks exactly one condition:

| subtest name | broken condition | expected `Reason` |
|---|---|---|
| `level too low` | character level 29 vs `reqLevel` 30 | `level_too_low` |
| `no maker skill` | character has no Maker variant at all | `skill_level_too_low` |
| `maker skill level zero` | Maker present at level 0 | `skill_level_too_low` |
| `maker skill below recipe` | Maker level 1 vs `reqSkillLevel` 2 | `skill_level_too_low` |
| `missing material` | holds 4 of `4011001`, needs 5 | `insufficient_materials` |
| `missing reqItem` | `reqItem` 4000021 absent | `missing_prerequisite_item` |
| `missing reqEquip` | `reqEquip` 1002419 not equipped | `missing_prerequisite_item` |
| `missing reqQuest` | quest 21614 at state 1, needs 3 | `missing_prerequisite_quest` |
| `insufficient mesos` | 1199 mesos vs `meso` 1200 | `insufficient_mesos` |
| `inventory full` | `CanAccommodate` returns false | `inventory_full` |

`TestEligibilityAllConditionsMetIsEligible` — the fully-eligible fixture returns `Eligible == true` and an empty `Reason`.

`TestAnyMakerVariantSatisfiesTheSkillGate` — four subtests, one per identity (`BeginnerMaker`, `NoblesseMaker`, `LegendMaker`, `EvanMaker`), each the character's only Maker skill at level 2; assert all four satisfy a `reqSkillLevel` of 2.

`TestReqQuestIsOnlyReadWhenTheRecipeCarriesOne` — evaluate a recipe with no `reqQuest` and assert the quest processor was **not** called. Two recipes in the archive carry one; reading quests for all of them is the cost C-5 explicitly avoids.

`TestMaterialCountSumsAcrossStacks` — a 5-count material held as 3+2 across two slots is sufficient. This is the case `DestroyAsset` would silently under-consume and is why Task 23 uses per-slot destruction.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-maker/atlas.com/maker`:

```
go test ./craft/... -count=1
```

Expected: FAIL — no such package.

- [ ] **Step 3: Write `snapshot.go` and `eligibility.go`**

`snapshot.go` — build from three `compartment.GetByType` calls using `inventory.TypeValueEquip`, `TypeValueUse` and `TypeValueETC`. Expose `Held(item.Id) uint32` (summed) and `Slots(item.Id) []SlotHolding` (ordered by slot), plus an `Equipped(item.Id) bool` for `reqEquip`.

`eligibility.go` — the ordered check chain above, returning on the first failure with the matching reason. Resolve the Maker level as the maximum level across the four identities the character actually has.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-maker/atlas.com/maker`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maker/atlas.com/maker/craft/
git commit -m "feat(maker): evaluate per-character recipe eligibility"
```

---

## Task 22: `atlas-maker` — the weighted reward draw

FR-1.4, FR-3.1, and the PRD's Randomness NFR.

### Files

- `services/atlas-maker/atlas.com/maker/craft/draw.go` — new file; `totalWeight`, `selectWeightedIndex`, `Draw`
- `services/atlas-maker/atlas.com/maker/craft/draw_test.go` — new file

Module root for `go build`/`go test`: `services/atlas-maker/atlas.com/maker`.

Patterns to copy: `services/atlas-reward-pools/atlas.com/reward-pools/reward/processor.go:225-254`, which is the proven implementation of exactly this:

```go
func totalWeight(pool []poolItem) uint32 {
	var total uint32
	for _, pi := range pool { total += pi.Weight }
	return total
}

func selectWeightedIndex(pool []poolItem, roll uint32) int {
	var cumulative uint32
	for i, pi := range pool {
		cumulative += pi.Weight
		if roll < cumulative { return i }
	}
	return len(pool) - 1
}
```

That file draws its roll from **`crypto/rand`** — `rand.Int(rand.Reader, big.NewInt(int64(total)))` at `:262` — and falls back to a uniform `rand.Int(..., len(pool))` pick for a zero-weight pool at `:269-276`. Mirror both.

### Interfaces

- Consumes: `recipe.Reward` (Task 20).
- Produces, for Task 23: `craft.Draw(rewards []recipe.Reward) (recipe.Reward, error)`, plus the unexported `totalWeight` and `selectWeightedIndex` the tests exercise directly.

### Constraints

`math/rand` is banned here — the draw decides item value and a predictable stream is exploitable. The weight table is never sent to the client (PRD §8 Randomness); nothing in this package may reach a REST response.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-maker/atlas.com/maker/craft/draw_test.go`.

`TestSelectWeightedIndexAcrossCumulativeRanges` — the pure function, tested directly across every boundary. Pool weights `[70, 25, 5]`, cumulative `[70, 95, 100]`:

| subtest name | roll | expected index |
|---|---|---|
| `first weight lower bound` | `0` | `0` |
| `first weight upper bound` | `69` | `0` |
| `second weight lower bound` | `70` | `1` |
| `second weight upper bound` | `94` | `1` |
| `third weight lower bound` | `95` | `2` |
| `third weight upper bound` | `99` | `2` |
| `roll at total falls to last` | `100` | `2` |

The three lower-bound cases are the off-by-one the cumulative form invites.

`TestTotalWeight` — `[70, 25, 5]` sums to `100`; the empty pool sums to `0`.

`TestSelectWeightedIndexSingleEntry` — a one-entry pool returns index `0` for roll `0`.

`TestSelectWeightedIndexZeroWeightEntriesAreNeverSelected` — pool `[50, 0, 50]`: assert no roll in `0..99` yields index `1`.

`TestDrawReturnsAnEntryFromThePool` — call `Draw` 100 times over the three-entry pool and assert every result is one of the three rewards. Do not assert a distribution — that is a flaky test, and `crypto/rand` is not stubbed here.

`TestDrawEmptyPoolReturnsError` — assert a distinguishable error rather than a panic or a zero-value reward.

`TestDrawAllZeroWeightsSelectsUniformly` — assert `Draw` returns some entry rather than erroring, matching reward-pools' zero-weight fallback.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-maker/atlas.com/maker`:

```
go test ./craft/... -count=1 -run "Draw|WeightedIndex|TotalWeight"
```

Expected: FAIL — undefined `Draw`, `selectWeightedIndex`, `totalWeight`.

- [ ] **Step 3: Write `draw.go`**

Port `reward/processor.go:225-276` against `recipe.Reward`, keeping `totalWeight` and `selectWeightedIndex` as pure unexported functions so they stay directly testable, and drawing the roll from `crypto/rand` exactly as that file does.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-maker/atlas.com/maker`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Assert `math/rand` is absent**

Run `grep -rn "math/rand" services/atlas-maker` from the repo root. Expected: no output. If anything matches, replace it with `crypto/rand` before committing — a predictable roll here decides item value and is exploitable.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maker/atlas.com/maker/craft/
git commit -m "feat(maker): draw random rewards by weight from crypto/rand"
```

---

## Task 23: `atlas-maker` — craft validation, consumption plan, and saga emission

The heart of the service. Design §3.2, §4.5.2, and §7.

### Files

- `services/atlas-maker/atlas.com/maker/craft/plan.go` — new file; the consumption plan
- `services/atlas-maker/atlas.com/maker/craft/processor.go` — new file; validation and saga construction
- `services/atlas-maker/atlas.com/maker/craft/inflight.go` — new file; the duplicate-suppression guard
- `services/atlas-maker/atlas.com/maker/craft/plan_test.go` — new file
- `services/atlas-maker/atlas.com/maker/craft/processor_test.go` — new file
- `services/atlas-maker/atlas.com/maker/craft/inflight_test.go` — new file

Module root for `go build`/`go test`: `services/atlas-maker/atlas.com/maker`.

### Interfaces

- Consumes: everything from Tasks 17-22.
- Produces, for Task 24: `craft.Processor.Create(characterId uint32, req Request) (uuid.UUID, error)` returning the saga transaction id, and a typed error whose code maps onto PRD §5.

### The saga builder has no typed helpers

`saga.NewBuilder()` (`libs/atlas-saga/builder.go:19-24`) exposes only `SetTransactionId`, `SetSagaType`, `SetInitiatedBy`, `SetTimeout`, one generic `AddStep(stepId string, status Status, action Action, payload any) *Builder` (`:52-64`), and `Build() Saga` (`:67-79`). There are **no** per-action `Add*` convenience methods. Every step is a bare `.AddStep(id, saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{...})` call.

### The three step sequences (design §4.5.2)

**Mode 1|2 (create).** `AwardMesos` (negative `Amount`, the recipe's `meso`) → one `DestroyAssetFromSlot` **per resolved slot** for each material, gem and the catalyst → `AwardCraftedAsset` for an equip output, or plain `AwardAsset` for a non-equip output.

**Mode 3 (crystal).** `AwardMesos` (negative, recipe `meso`) → `DestroyAssetFromSlot` for the leftover → `AwardAsset` for the weighted `randomReward` draw.

**Mode 4 (disassemble).** `DestroyAssetFromSlot` at the client-supplied slot — *after* verifying that slot actually holds the claimed equip → `AwardAsset` per derived crystal → `AwardMesos` for the charge.

### Why `DestroyAssetFromSlot` and not the alternatives

`DestroyAsset` resolves a template to the **first** matching slot only, so a 5-count material spanning two stacks silently under-consumes. `DestroyAllAssets` is explicitly **not compensable**. `DestroyAssetFromSlotPayload` carries an optional `TemplateId` (`libs/atlas-saga/payloads.go:141`) precisely so the compensator can re-create the asset — always set it. `atlas-maker` supplies the exact slots from the Task 21 snapshot, which is also what makes the "never trust client quantities" NFR true.

### OQ-7 — mode 3 consumes 100 of the leftover

All four decompiled clients render the mode-3 consumption log as `Format(SP_293_YOU_HAVE_LOST_ITEMS_S_D, <name>, 100)` — the literal `100` is hard-coded in the client. The reference archive's group-`0` recipe lists its leftover material with `count: 1`. The wire carries only the item id, so the discrepancy is invisible to the protocol.

**Decision: consume 100.** A client whose chat log says "lost 100" while the server removed 1 is a visible, exploitable inconsistency, and 100 is the quantity every client build agrees on. The recipe's `count` is ignored for group `0`.

**Before implementing, confirm this against the reference server's crystal path and reverse the decision if it disagrees** — the check is cheap and the design records it precisely so it is not left as an unowned gap. Record the outcome in a comment at the constant's declaration.

### Replay suppression (§7)

A per-`(tenant, characterId)` in-flight guard, taken when a craft saga is emitted and released on its terminal event. A second `MAKER_SKILL` arriving while one is in flight returns `craft_in_progress` (409).

This is **in-memory**, consistent with §4.2.6's "no persistent craft state". It is a duplicate-suppression window, not durable state: a restart losing it degrades to the ordinary validation path, which is still server-authoritative and cannot double-spend a material that is no longer there.

- [ ] **Step 1: Write the failing tests**

`plan_test.go`:

`TestPlanResolvesMaterialAcrossMultipleSlots` — a 5-count material held as 3 in slot 1 and 2 in slot 7 produces exactly two `DestroyAssetFromSlot` entries: `(slot 1, qty 3)` and `(slot 7, qty 2)`, both carrying `TemplateId`. Assert the quantities sum to exactly 5 — never 6, and never one entry for 5 against a slot holding 3.

`TestPlanConsumesFromLowestSlotFirst` — with holdings in slots 7, 1 and 4, assert the plan consumes in ascending slot order. Deterministic ordering is what makes the plan reproducible and its tests non-flaky.

`TestPlanDropsUnheldReagents` — FR-3.2. A request naming three gems of which the character holds two produces destroy steps for the two held gems only, and the craft is **not** rejected.

`TestPlanIncludesCatalystWhenFlagSetAndHeld` — assert the recipe's `catalyst` template is consumed once when `bUseCatalyst` is set and held.

`TestPlanOmitsCatalystWhenNotHeld` — assert the craft proceeds without it rather than failing.

`TestPlanNeverTrustsClientQuantities` — submit a request whose gem list names a template the character does not hold at all, plus a duplicate of one they hold once; assert the plan destroys exactly one of the held gem and nothing of the unheld one.

`processor_test.go`:

`TestCreateModeOneBuildsSequence` — assert the built saga's steps are, in order: `AwardMesos` with a **negative** `Amount` equal to the recipe's `meso`; one `DestroyAssetFromSlot` per resolved slot; `AwardCraftedAsset` with `Slots` equal to the recipe's `tuc` and the reagent-adjusted stat block. Assert the step count exactly.

`TestCreateModeOneNonEquipUsesAwardAsset` — a non-equip output produces `AwardAsset`, not `AwardCraftedAsset`.

`TestCreateModeThreeConsumesOneHundredLeftover` — OQ-7. Assert the leftover destroy steps sum to exactly `100`, not the recipe's `count`.

`TestCreateModeThreeAwardsOneWeightedDraw` — assert exactly one `AwardAsset` step, and that its template is one of the recipe's `randomReward` entries.

`TestCreateModeFourVerifiesSlotBeforeDestroying` — a request claiming equip `1082002` in EQUIP slot 5 when that slot holds something else is rejected with `equip_not_found` and **emits no saga at all**.

`TestCreateModeFourChargesMeso` — assert the mode-4 sequence ends with an `AwardMesos` step whose amount is the derived charge, and that the crystals come from `crystalband.CrystalForLevel` against the equip's `reqLevel`.

`TestEveryRejectionEmitsNoSaga` — table-driven over **every** PRD §5 error condition. For each: assert the returned error's code, and assert the saga emitter was called **zero** times. This is FR-3.7's "rejection is pre-mutation" made executable — a rejected craft cannot have mutated anything because nothing was emitted.

`TestEveryStepUsesACompensableAction` — walk the built saga for all three modes and assert no step uses `DestroyAllAssets`. Assert positively that every action appears in the compensable set.

`inflight_test.go`:

`TestSecondCraftWhileInFlightIsRejected` — assert the second call returns `craft_in_progress` and emits no saga.
`TestGuardReleasesOnTerminalEvent` — after release, a second craft succeeds.
`TestGuardIsPerCharacterAndPerTenant` — a craft in flight for character A does not block character B, nor the same character id under a different tenant.
`TestGuardIsConcurrencySafe` — fire N concurrent `Create` calls for one character and assert exactly one emits.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-maker/atlas.com/maker`:

```
go test ./craft/... -count=1
```

Expected: FAIL — undefined `Plan`, `Processor.Create`, the in-flight guard.

- [ ] **Step 3: Write `plan.go`**

Resolve each required template to concrete `(inventoryType, slot, quantity)` tuples against the Task 21 snapshot, consuming slots in ascending order and setting `TemplateId` on every entry. Drop unheld reagents. Never read a quantity from the request.

- [ ] **Step 4: Write `inflight.go`**

A `sync.Mutex`-guarded `map[inflightKey]struct{}` keyed by tenant id and character id, with `TryAcquire` and `Release`. Take the guard **before** emitting and release it on the terminal event.

- [ ] **Step 5: Write `processor.go`**

Validate via Task 21's eligibility, build the plan, apply reagent stat adjustments from Task 17 for an equip output, draw the reward via Task 22 when the recipe has one, and build the saga with `saga.NewBuilder()` and bare `AddStep` calls in the sequences above.

Emit the PRD §8 Observability log line at info level for each accepted craft: tenant, character, mode, recipe id, materials consumed, meso delta, produced item, correlated by saga id.

Declare the mode-3 leftover quantity as a named constant with the OQ-7 rationale and the Step-1 confirmation outcome in its doc comment — not as a bare `100`.

- [ ] **Step 6: Run the tests to verify they pass**

Run from `services/atlas-maker/atlas.com/maker`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maker/atlas.com/maker/craft/
git commit -m "feat(maker): validate crafts, plan consumption, and emit the craft saga"
```

---

## Task 24: `atlas-maker` — REST surface and error codes

PRD §5's `atlas-maker` endpoints, plus design §4.2.5's two additions.

### Files

- `services/atlas-maker/atlas.com/maker/craft/resource.go` — new file; the three routes
- `services/atlas-maker/atlas.com/maker/craft/rest.go` — new file; request and response models
- `services/atlas-maker/atlas.com/maker/craft/errors.go` — new file; the stable error codes
- `services/atlas-maker/atlas.com/maker/craft/resource_test.go` — new file
- `services/atlas-maker/atlas.com/maker/main.go` — new file as of Task 15; register the routes
- `services/atlas-maker/atlas.com/maker/README.md` — new file as of Task 15; document the endpoints

Module root for `go build`/`go test`: `services/atlas-maker/atlas.com/maker`.

Patterns to copy: `services/atlas-reward-pools/atlas.com/reward-pools/reward/resource.go` (116 lines) for a compute-only domain's route registration and handler shape.

### Interfaces

- Consumes: `craft.Processor` (Task 23), `recipe.Processor` (Task 20).
- Produces, for Task 25: the three HTTP routes `atlas-channel` calls.

### Routes

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/characters/{characterId}/maker/recipes` | recipes the character currently qualifies for |
| `GET` | `/characters/{characterId}/maker/recipes/{itemId}` | one recipe with per-character eligibility and the computed material/meso cost |
| `POST` | `/characters/{characterId}/maker/crafts` | validate and, on success, emit the craft saga; returns the saga id |

The list route **accepts the standard pagination params** (design §4.2.5): the eligible set is small but the full set is not.

Recipe data is read-only — no route permits recipe mutation (FR-2.3).

### Error codes

Every error is a JSON:API error carrying a stable `code`. PRD §5's nine, plus the two the design adds:

| Condition | Status | Code |
|---|---|---|
| Recipe id not found | 404 | `recipe_not_found` |
| Character below `reqLevel` | 422 | `level_too_low` |
| Maker skill absent or below `reqSkillLevel` | 422 | `skill_level_too_low` |
| Missing a `recipe` material | 422 | `insufficient_materials` |
| Missing `reqItem` / `reqEquip` | 422 | `missing_prerequisite_item` |
| Missing a required quest state | 422 | `missing_prerequisite_quest` |
| Insufficient mesos | 422 | `insufficient_mesos` |
| No free inventory slot for an award | 422 | `inventory_full` |
| Equip not in the named slot (disassemble) | 422 | `equip_not_found` |
| Leftover has no crystal mapping | 422 | `no_crystal_mapping` |
| A craft is already in flight | 409 | `craft_in_progress` |

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-maker/atlas.com/maker/craft/resource_test.go`.

`TestListRecipesReturnsOnlyEligible` — seed a character eligible for two of five recipes; assert exactly those two are returned.

`TestListRecipesPaginates` — seed 25 eligible recipes; `GET ...?page[number]=2&page[size]=10`; assert 10 returned and the pagination links present.

`TestGetRecipeIncludesEligibilityAndCost` — assert the single-recipe response carries the eligibility verdict and the computed material and meso cost.

`TestEveryErrorCodeIsReturnedByItsOwnCondition` — table-driven, **one case per row of the table above**, each provoking exactly that condition and asserting both the HTTP status and the JSON:API `code`. This is the PRD §10 acceptance criterion "every error code in §5 is returned by a test that provokes exactly that condition"; all eleven rows must be present.

`TestPostCraftReturnsSagaId` — a valid craft returns 200/202 with the saga transaction id.

`TestFailureLeavesStateUnchanged` — FR-3.7's acceptance criterion, **asserted rather than reasoned about**. For every rejection condition: capture the character's materials, mesos and equips before the request, issue it, and assert the post-request state is identical field-for-field. Assert the saga emitter was called zero times.

`TestRecipeRoutesAreReadOnly` — assert `POST`, `PUT`, `PATCH`, `DELETE` on both recipe routes return 405 (FR-2.3).

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-maker/atlas.com/maker`:

```
go test ./craft/... -count=1 -run Resource
```

Expected: FAIL — undefined `InitResource`.

- [ ] **Step 3: Write `errors.go`, `rest.go` and `resource.go`**

`errors.go` — a typed error carrying the stable code and HTTP status, with one constructor per row. Task 21's eligibility reasons map onto these 1:1.

`rest.go` — the craft request model (mode, target item, catalyst flag, gem list, leftover id, disassemble slot) and the response models. `GetName()` returns `"makerCrafts"` and `"makerRecipes"` respectively.

`resource.go` — the three routes; the list route paginates.

- [ ] **Step 4: Wire `main.go` and the README**

Add `AddRouteInitializer(craft.InitResource(db)(GetServer()))` to the chain, and populate the README's REST endpoints table with all three routes and the full error-code table.

- [ ] **Step 5: Run the tests to verify they pass**

Run from `services/atlas-maker/atlas.com/maker`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maker/atlas.com/maker/
git commit -m "feat(maker): expose maker recipe and craft REST endpoints"
```

---

## Task 25: `atlas-channel` — the `MAKER_SKILL` handler

FR-5.1.

### Files

- `services/atlas-channel/atlas.com/channel/handler/maker_skill.go` — new file; the handler
- `services/atlas-channel/atlas.com/channel/handler/maker_skill_test.go` — new file
- `services/atlas-channel/atlas.com/channel/maker/requests.go` — new file; the `atlas-maker` client
- `services/atlas-channel/atlas.com/channel/maker/processor.go` — new file
- `services/atlas-channel/atlas.com/channel/main.go` — register the handler
- `services/atlas-configurations/seed-data/templates/` — the eight applicable templates gain the handler entry

Module root for `go build`/`go test`: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: the `ItcOperationHandle` registration at `services/atlas-channel/atlas.com/channel/main.go:1020`, in this block:

```go
handlerMap[fieldsb.EnterMtsHandle] = handler.EnterMtsHandleFunc
handlerMap[fieldsb.ItcStatusChargeHandle] = handler.ItcStatusChargeHandleFunc
handlerMap[fieldsb.ItcQueryCashRequestHandle] = handler.ItcQueryCashRequestHandleFunc
handlerMap[fieldsb.ItcOperationHandle] = handler.ItcOperationHandleFunc
handlerMap[petsb.PetMovementHandle] = handler.PetMovementHandleFunc
```

For the REST client, copy `services/atlas-channel/atlas.com/channel/compartment/requests.go`'s `getBaseRequest` + `requests.PostRequest` shape; the new token is `MAKER`.

### Interfaces

- Consumes: `serverbound.MakerSkill` and `serverbound.MakerSkillHandle` (Task 7); `atlas-maker`'s `POST /characters/{id}/maker/crafts` (Task 24).
- Produces, for Task 26: an emitted craft saga whose terminal event drives the result write.

### Handler contract

1. Decode the mode and arm body.
2. Resolve the character from the session.
3. `POST` to `atlas-maker` `/characters/{id}/maker/crafts` with the untrusted request **verbatim** — the channel does not validate; `atlas-maker` does.
4. On rejection, map the returned `code` to a `MAKER_RESULT` failure and write it (Task 26 supplies the writer).
5. On acceptance, write **nothing** — the result is written when the saga reaches a terminal state (design §3.3).

The handler is `LoggedInValidator`-gated, like every other in-field op.

### The serverbound mode table

The mode is **not** a dispatcher family (see Task 7). Serverbound mode dispatch is the handler's `options.operations` routing table in each seed template, which is the documented mechanism. The handler entry's shape, from `services/atlas-configurations/seed-data/templates/template_gms_95_1.json:2557-2563`:

```json
"handler": "ItcOperationHandle",
"fname": "CITC::OnRegisterSaleEntry",
"options": { "operations": { "REGISTER_SALE": 2, "SALE_CURRENT_ITEM": 3, ... } }
```

The `MakerSkillHandle` entry maps `CREATE: 1`, `CREATE_WITH_UPGRADE: 2`, `MONSTER_CRYSTAL: 3`, `DISASSEMBLE: 4` — confirm each against Task 6's derivation before writing.

Add it to exactly the eight applicable templates. `template_gms_12_1.json`, `template_gms_48_1.json` and `template_gms_61_1.json` are not touched.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/handler/maker_skill_test.go`, following the setup shape of an existing handler test in the same package.

`TestMakerSkillHandlerForwardsEachModeVerbatim` — table-driven over the four modes. For each, feed the corresponding byte fixture from Task 7 and assert the outbound `atlas-maker` request body carries the decoded fields **unchanged** — including a gem list the character does not hold. The channel must not filter; that is `atlas-maker`'s job and filtering here would mask a validation bug.

`TestMakerSkillHandlerWritesFailureOnRejection` — table-driven over every PRD §5 error code. Stub `atlas-maker` to return each, and assert a `MAKER_RESULT` failure is written for each one. FR-5.2: no path may leave the client UI locked.

`TestMakerSkillHandlerWritesNothingOnAcceptance` — stub acceptance and assert **no** packet is written. The result comes from the saga's terminal event.

`TestMakerSkillHandlerRequiresLogin` — assert the handler is registered behind `LoggedInValidator`.

`TestMakerSkillHandlerWritesFailureWhenMakerIsUnreachable` — stub a transport error and assert a failure result is still written. An upstream outage must not lock the UI either.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-channel/atlas.com/channel`:

```
go test ./handler/... -count=1 -run MakerSkill
```

Expected: FAIL — undefined `MakerSkillHandleFunc`.

- [ ] **Step 3: Write the `maker` client**

`services/atlas-channel/atlas.com/channel/maker/requests.go`:

```go
const (
	craftsResource = "characters/%d/maker/crafts"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAKER")
}
```

Plus a `requests.PostRequest` for the craft, and `processor.go` wrapping it with the typed error-code mapping.

`MAKER` must be added to `deploy/k8s/base/env-configmap.yaml` and both overlays' generators if service base URLs are configured that way — check how `INVENTORY` is provisioned for `atlas-channel` and mirror it exactly. A missing token resolves to the bare string and the call fails at runtime with only a warn log.

- [ ] **Step 4: Write the handler and register it**

`services/atlas-channel/atlas.com/channel/handler/maker_skill.go` implementing the contract above.

In `main.go`'s `produceHandlers()`, alongside the block at `:1017-1023`:

```go
	handlerMap[charactersb.MakerSkillHandle] = handler.MakerSkillHandleFunc
```

Use whatever import alias `main.go` already binds to `libs/atlas-packet/character/serverbound`; add one if none exists.

- [ ] **Step 5: Add the handler to the eight seed templates**

Add the `MakerSkillHandle` entry with its `options.operations` mode table to each of `template_gms_72_1.json`, `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`, using each version's own opcode from the registry.

Then run from the repo root:

```
go run ./tools/packet-audit operations --check
```

Expected: exit 0.

- [ ] **Step 6: Run the tests to verify they pass**

Run from `services/atlas-channel/atlas.com/channel`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/ services/atlas-configurations/seed-data/templates/
git commit -m "feat(channel): handle MAKER_SKILL and forward crafts to atlas-maker"
```

---

## Task 26: `atlas-channel` — the `MAKER_RESULT` writer and terminal-event consumer

FR-5.2 and design §3.2 step 5.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/maker/consumer.go` — new file; the saga terminal-event consumer
- `services/atlas-channel/atlas.com/channel/kafka/consumer/maker/consumer_test.go` — new file
- `services/atlas-channel/atlas.com/channel/main.go` — register the writer and the consumer
- `services/atlas-configurations/seed-data/templates/` — the eight applicable templates gain the writer entry

Module root for `go build`/`go test`: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `produceWriters()` at `services/atlas-channel/atlas.com/channel/main.go:702` for the writer-name list; the writer entry's template shape from `services/atlas-configurations/seed-data/templates/template_gms_95_1.json:3082-3090`:

```json
"writer": "ClaimResult",
"fname": "CWvsContext::OnClaimResult",
"options": { "operations": { "SUCCESS": 2, "REPORTED_NOTICE": 3, ... } }
```

For the soft-resolve fallback idiom, copy `services/atlas-channel/atlas.com/channel/kafka/consumer/mts/consumer.go`'s `failNoticeOr`.

### Interfaces

- Consumes: the five body functions from Task 8; the craft saga's terminal events.
- Produces: a `MAKER_RESULT` on every terminal path.

### Why the result is written after the saga

The alternative — write the success result on validation and let the saga run behind it — is simpler and one round-trip faster, and it is what the reference server effectively does. It is rejected because **the create-arm body must enumerate the actual materials and gems consumed**. Reporting a consumption that a later compensation reverses puts the client's chat log permanently out of step with the inventory, and FR-3.7 requires a rejected craft to leave state byte-identical. The latency is already tolerated: the client UI is locked from `StartItemMake` until `OnMakerResult`.

### Every path emits exactly one result

| Terminal condition | Arm |
|---|---|
| saga completed, mode 1 | `CREATE` |
| saga completed, mode 2 | `CREATE_WITH_UPGRADE` |
| saga completed, mode 3 | `MONSTER_CRYSTAL` |
| saga completed, mode 4 | `DISASSEMBLE` |
| saga compensated | `FAILED` |
| saga timed out | `FAILED` |
| synchronous rejection (Task 25) | `FAILED` |

There is no third path. The client re-enables its UI on **every** `OnMakerResult` regardless of `nResult`, so no terminal state leaves it locked.

The consumer must also **release Task 23's in-flight guard** on every terminal event, including timeout and compensation. A guard that leaks on the failure path locks the character out of crafting until the pod restarts.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/maker/consumer_test.go`.

`TestTerminalEventWritesCorrectArm` — table-driven, one case per row of the table above; assert the correct arm's body function was invoked.

`TestCompletedCreateEnumeratesActualConsumption` — assert the `CREATE` arm's material list, gem list, catalyst flag and meso cost match what the **saga actually consumed**, not what the client requested. Include a case where the request named an unheld gem and assert it is absent from the result.

`TestCompensatedSagaWritesFailed` — assert a `FAILED` arm with `nResult > 1` and a 4-byte body.

`TestTimedOutSagaWritesFailed` — same, for the timeout path.

`TestEveryTerminalPathReleasesTheInFlightGuard` — assert the guard is released on completion, compensation **and** timeout. A test that covers only the success path would miss the lockout.

`TestUnknownTerminalStateStillWritesAResult` — feed an unrecognised terminal state and assert a `FAILED` is written rather than nothing. FR-5.2 admits no silent path.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-channel/atlas.com/channel`:

```
go test ./kafka/consumer/maker/... -count=1
```

Expected: FAIL — no such package.

- [ ] **Step 3: Write the consumer**

Consume the craft saga's terminal events, resolve the arm from the saga type and terminal state, call the matching body function from Task 8, write to the session, and release the in-flight guard.

Where an `operations` table or key could be missing, **soft-resolve with a fallback to the `FAILED` arm** — never let `ResolveCode`'s 99-on-miss reach the client. Copy `mts/consumer.go`'s `failNoticeOr` idiom.

- [ ] **Step 4: Register the writer and the consumer**

Add `charactercb.MakerResultWriter` to the `produceWriters()` slice at `main.go:702`, and register the consumer alongside the other Kafka consumers.

Add any new topic env var to `deploy/k8s/base/env-configmap.yaml` and both overlays' `configMapGenerator` literals — the `behavior: replace` trap means an unlisted key simply does not exist in that environment.

- [ ] **Step 5: Add the writer to the eight seed templates**

The `operations` block for the `MakerResult` writer is **generated** from `docs/packets/dispatchers/maker_result.yaml` by `packet-audit operations` (Task 9) — do not hand-write it. Add the writer's `fname` and per-version opcode entry, then run from the repo root:

```
go run ./tools/packet-audit operations
go run ./tools/packet-audit operations --check
```

Expected: exit 0, and the three excluded templates untouched.

- [ ] **Step 6: Run the tests to verify they pass**

Run from `services/atlas-channel/atlas.com/channel`:

```
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/ services/atlas-configurations/seed-data/templates/
git commit -m "feat(channel): write MAKER_RESULT on every craft saga terminal state"
```

---

## Task 27: Gates and pre-PR review

Nothing new is implemented here. This is the gate that decides the branch is done.

### Files

- `docs/tasks/task-285-maker-skill-crafting/context.md` — record the gate outcomes
- `docs/tasks/task-285-maker-skill-crafting/coverage-manifest.yaml` — new file as of Task 6; reconcile against what the branch actually changed

Read-only references: `tools/verify.sh`, `tools/service-registration-guard.sh`, `docs/review-protocol.md`.

### Interfaces

- Consumes: every prior task.
- Produces: a branch that is genuinely ready for a PR.

- [ ] **Step 1: Run every packet gate**

Run from the repo root:

```
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit doc-freshness --check
go run ./tools/packet-audit gate-check --check
```

Every one must exit 0. Then confirm in `docs/packets/audits/STATUS.md`:

- `MAKER_SKILL` reads `✅` on all eight applicable versions, `⬜` on `gms_v48`/`gms_v61`.
- `MAKER_RESULT` reads `✅` — not `🧩` — on all eight, `⬜` on `gms_v48`/`gms_v61`.
- No cell anywhere in the matrix reads `🟥`.

- [ ] **Step 2: Run the service registration guard**

Run from the repo root:

```
tools/service-registration-guard.sh
```

Expected: exit 0.

Then the checks it cannot perform:

```
kubectl kustomize deploy/k8s/overlays/main | grep -B2 -A6 "name: atlas-maker$"
kubectl kustomize deploy/k8s/overlays/pr > /dev/null
docker buildx bake atlas-maker
```

Confirm by hand: the new `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` keys and the `MAKER` service token exist in base `env-configmap.yaml` **and** in both overlays' generators; the `atlas.seed-catalog` label is on the Deployment; the ingress route is present in both `deploy/shared/routes.conf` and the regenerated template.

- [ ] **Step 3: Run the full verification gate**

Run from the repo root:

```
tools/verify.sh
```

**Flagless.** `--quick`/`--no-docker` also exit 0 but skip the bake and `-race`; they do not count. Per CLAUDE.md, the flagless run must exit 0 before this branch may be called done.

If a guard fails, consult `docs/verification.md` before changing the script.

- [ ] **Step 4: Reconcile the coverage manifest**

Run `packet-completeness-critic` against the branch. It diffs `coverage-manifest.yaml` against the branch's actual git delta over `libs/atlas-packet` structs and version gates, and the matrix delta over `status.json`, reporting:

- **CHANGED-BUT-UNCLAIMED** — a codec or gate moved but the task never declared it.
- **CLAIMED-BUT-UNVERIFIED** — a manifest `op × version` with no verified cell.

Both must be empty. If a codec was legitimately touched incidentally, add it to `out_of_scope` with a justification — an `out_of_scope` entry is a claim that the touch is intentional and needs no verification, not a way to silence the critic.

- [ ] **Step 5: Trace the cross-service seam by hand**

A green `verify.sh` cannot see a cross-service seam defect. Trace the explicit-stat contract end to end and confirm a test asserts the **new** contract at each hop:

`craft.Processor` → `AwardCraftedAssetPayload` → orchestrator `saga/handler.go` → `RequestCreateItemWithExplicitStats` → `CreateAssetCommandBody` (orchestrator copy) → Kafka → `CreateAssetCommandBody` (inventory copy) → `handleCreateAssetCommand` → `asset.CreateOptions` → the persisted asset.

Specifically confirm the two independently-declared `CreateAssetCommandBody` structs still agree field-for-field and tag-for-tag. A field added to one and not the other decodes as its zero value with **no error** — every crafted equip would silently get zero stats.

- [ ] **Step 6: Run code review**

Per CLAUDE.md, always run code review before opening a PR, even when the plan looks complete. Per `docs/review-protocol.md`, dispatch:

- `backend-guidelines-reviewer` over the changed Go packages — `atlas-maker` is a whole new service and gets the full `SCAFFOLD-*` treatment.
- `plan-adherence-reviewer` against this plan.
- `packet-completeness-critic` (Step 4) as the packet-specific companion.

`frontend-guidelines-reviewer` is not applicable — no TypeScript changed.

- [ ] **Step 7: Record the outcomes and the rollout steps**

Append to `docs/tasks/task-285-maker-skill-crafting/context.md`: each gate's command and its exit status, and the review verdicts.

Repeat in the PR description the two out-of-repo manual steps from Task 16 Step 8 — **create `atlas-maker-main` on postgres.home before merging**, and **flip the GHCR package to public after the first image push**. Both fail only at runtime, and both are invisible to CI.

Also carry forward the unresolved risk **R-2 (OQ-1)**: only a GMS 83.1 XML dump exists on this machine, so "does every target client ship `ItemMake.img` with the same field set?" cannot be answered here. The mitigation is structural — FR-1.5's default-don't-fail reading means an archive missing a field ingests with zeros, and a tenant whose archive lacks the file ingests an empty recipe set rather than failing startup. This is a genuine external blocker and is surfaced, not guessed at.
