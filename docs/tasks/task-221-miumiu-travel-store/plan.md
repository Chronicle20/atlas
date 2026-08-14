# MiuMiu Travel Store (cash item category 545) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Using *Miu Miu the Traveling Merchant* (5450000) opens NPC 9090000's shop from any map and consumes exactly one copy only after the shop actually opened, on every GMS tenant whose client can send the op.

**Architecture:** A classification-545 arm in `atlas-channel`'s `CharacterCashItemUseHandleFunc` resolves the target NPC from `atlas-data`, records a pending-unlock entry in a new in-memory `remotemerchant` registry, and creates a two-step saga (`open_npc_shop` → `destroy_asset_from_slot`). `atlas-saga-orchestrator` dispatches `COMMAND_TOPIC_NPC_SHOP`/`ENTER` carrying the saga's transaction id, and a new consumer on `EVENT_TOPIC_NPC_SHOP_STATUS` completes or fails step 1 on `ENTERED`/`ENTER_ERROR`. `atlas-channel`'s existing npc-shop consumer announces the shop packet and — only when the registry has an entry for that character — sends `EnableActions`, so the ordinary NPC-talk path is byte-identical to today. Supporting work: `atlas-data` exposes the cash item's `info/npc`; `atlas-npc-shops` learns to report enter failures instead of silently emitting nothing; ten shop seed files; and the missing `NPCShopHandle`/`NPCShop`/`NPCShopOperation` template registrations that make NPC shops transactable at all on gms_48/87/92/95.

**Tech Stack:** Go 1.x monorepo (`go.work`), Kafka (`libs/atlas-kafka`), JSON:API REST (api2go), GORM, `libs/atlas-saga` shared contract, `libs/atlas-constants`, `libs/atlas-packet`, `deploy/seed` JSON catalog, `services/atlas-configurations/seed-data/templates/*.json`.

**Repo-root note:** every command below that says `cd "$(git rev-parse --show-toplevel)"` means the task worktree root. All paths in this document are repo-relative.

---

## Global Constraints

These apply to **every** task. They are not restated per task.

- **Multi-tenancy.** Every code path resolves the tenant via `tenant.MustFromContext(ctx)`. No cross-tenant state; registry keys include `tenant.Model`.
- **Version gating uses the tenant idiom, never a raw comparison.** Use `t.IsRegion("GMS")` and `t.MajorAtLeast(n)` (`libs/atlas-tenant/tenant.go`). A bare `t.MajorVersion() > 83` is the bug recorded in `[[bug_majorversion_gt83_is_off_by_one_v87]]` and is forbidden in new code.
- **No wire change to an already-verified version.** v61/72/79/83/84/87/95 NPC-shop packet bytes must be identical after this task; their `OPEN_NPC_SHOP` / `CONFIRM_SHOP_TRANSACTION` matrix cells stay ✅.
- **Never emit `NPCShopOperation` (CONFIRM_SHOP_TRANSACTION) outside a buy/sell/recharge round trip.** `CShopDlg::OnPacket` @ `0x756da7` throws `CDisconnectException` when that packet arrives with no request outstanding — recorded in `services/atlas-npc-shops/atlas.com/npc/shops/producer.go:36-51`. This is why Task 4 introduces a *separate* enter-failure event type rather than reusing `ERROR`.
- **No hard-coded client wire values.** The target NPC id comes from WZ via `atlas-data` (DOM-25, `[[feedback_client_wire_values_config_resolved]]`).
- **Shared types first (DOM-21).** `world.Id`, `channel.Id`, `item.Id`, `slot.Position`, `item.Classification` come from `libs/atlas-constants`. Do not redeclare.
- **Immutable models + Builder for tests.** No `*_testhelpers.go` files with test-only constructors.
- **No `// TODO`, no stubs, no 501s in landed commits.**
- **Goroutines only via `routine.Go`** (`tools/goroutine-guard.sh`).
- **Repo-relative paths only in committed files** — never a literal `/home/<user>/…`.
- **Commit after every task.** Conventional-commit subject prefixed `feat(task-221):` / `fix(task-221):` / `chore(task-221):`.

### Fixed values used across tasks

| Name | Value | Source |
|---|---|---|
| Remote-merchant classification | `item.ClassificationRemoteMerchant` = 545 | `libs/atlas-constants/item/constants.go:106` |
| Miu Miu item id | 5450000 | WZ `Item.wz/Cash/0545.img.xml` |
| Remote Gachapon Ticket item id | 5451000 (unreachable — no client emits it) | design §1.2 |
| Miu Miu target NPC | 9090000 (from `info/npc`) | WZ `Item.wz/Cash/0545.img.xml` |
| Cash-slot type (store) | 37 (GMS < 95) / 38 (GMS ≥ 95) | `character_cash_item_use.go:999-1005` |
| Cash inventory type byte | 5 | `libs/atlas-saga/payloads.go:109` |
| Enabled GMS majors | 72, 79, 83, 84, 87, 92, 95 (i.e. `IsRegion("GMS") && MajorAtLeast(72)`) | design §1.3 |
| Disabled | gms_12, gms_48, gms_61, JMS | design §1.3, §7.3 |

---

## File Structure

**Created**

| File | Responsibility |
|---|---|
| `services/atlas-channel/atlas.com/channel/remotemerchant/registry.go` | Per-(tenant, character) pending-unlock entry for a remote-initiated shop open; TTL sweep. |
| `services/atlas-channel/atlas.com/channel/remotemerchant/registry_test.go` | Registry unit tests. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_remote_merchant.go` | The classification-545 arm. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_remote_merchant_test.go` | Arm tests. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npcshop/kafka.go` | Orchestrator's mirror of the npc-shop Kafka contract. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/consumer.go` | `EVENT_TOPIC_NPC_SHOP_STATUS` consumer. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/consumer_test.go` | Consumer tests. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/testmain_test.go` | Tenant-context test main (copy of the storage one). |
| `tools/npc-shop-contract-mirror-guard.sh` | Keeps the three copies of the npc-shop contract identical. |
| `deploy/seed/gms/{12,48,61,72,79,83,84,87,92,95}_1/npc-shops/shops/shop-9090000.json` | Shop data (10 files). |
| `docs/tasks/task-221-miumiu-travel-store/commodity-existence-sweep.md` | Per-version existence results for the 26 commodities. |
| `docs/tasks/task-221-miumiu-travel-store/gms48-shop-operations.md` | IDB derivation of the v48 `NPCShopOperation` table. |
| `docs/tasks/task-221-miumiu-travel-store/live-config-reconciliation.md` | Read-back evidence for FR-6.6. |

**Modified**

| File | Change |
|---|---|
| `services/atlas-data/atlas.com/data/cash/rest.go` | `Npc uint32` field. |
| `services/atlas-data/atlas.com/data/cash/reader.go` | Parse `info/npc`. |
| `services/atlas-channel/atlas.com/channel/data/cash/rest.go` | `Npc uint32` field. |
| `services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go` | `TransactionId` on `Command`/`StatusEvent`; `ENTER_ERROR` type + body. |
| `services/atlas-npc-shops/atlas.com/npc/shops/producer.go` | Thread transaction id; new enter-error provider. |
| `services/atlas-npc-shops/atlas.com/npc/shops/processor.go` | `Enter` takes a transaction id, emits `ENTER_ERROR` on shop-missing and already-in-shop. |
| `services/atlas-npc-shops/atlas.com/npc/kafka/consumer/shops/consumer.go` | Pass `e.TransactionId` into `EnterAndEmit`. |
| `services/atlas-channel/atlas.com/channel/kafka/message/npc/shop/kafka.go` | Mirror of the above. |
| `services/atlas-channel/atlas.com/channel/npc/shops/producer.go` | Thread transaction id. |
| `services/atlas-channel/atlas.com/channel/npc/shops/processor.go` | `EnterShop` takes a transaction id. |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop/consumer.go` | Registry-conditional `EnableActions`; `ENTER_ERROR` handler; TTL sweep. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` | Route classification 545 into the new arm. |
| `services/atlas-channel/atlas.com/channel/socket/init.go` | Clear the registry on session destroy. |
| `libs/atlas-saga/model.go` | `OpenNpcShop` action, `RemoteMerchant` saga type. |
| `libs/atlas-saga/payloads.go` | `OpenNpcShopPayload`. |
| `libs/atlas-saga/unmarshal.go` | `OpenNpcShop` case. |
| `services/atlas-saga-orchestrator/.../saga/model.go` | Aliases + unmarshal case. |
| `services/atlas-saga-orchestrator/.../saga/character_extractor.go` | `OpenNpcShopPayload` case. |
| `services/atlas-saga-orchestrator/.../saga/handler.go` | `handleOpenNpcShop` + dispatch case. |
| `services/atlas-saga-orchestrator/.../saga/producer.go` | `NpcShopEnterCommandProvider`, `NpcShopExitCommandProvider`, `EmitNpcShopExit`. |
| `services/atlas-saga-orchestrator/.../saga/producer_testseam.go` | `SetEmitNpcShopExitForTest`. |
| `services/atlas-saga-orchestrator/.../saga/event_acceptance.go` | Two `EventKind`s + `acceptanceTable` entry. |
| `services/atlas-saga-orchestrator/.../saga/event_acceptance_test.go` | `allActions` gains `OpenNpcShop`. |
| `services/atlas-saga-orchestrator/.../saga/compensator.go` | `RemoteMerchant` reverse-walk + `OpenNpcShop` → `EXIT`. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go` | Register the new consumer. |
| `services/atlas-configurations/seed-data/templates/template_gms_{48,87,92,95}_1.json` | Handler/writer registrations. |
| `.github/workflows/pr-validation.yml` | CI job for the new mirror guard. |
| `CLAUDE.md` | Document the new guard. |

---

## Design deltas discovered while planning

These correct or extend `design.md`. Each is implemented by the task named.

| # | Design said | Reality (verified) | Task |
|---|---|---|---|
| D1 | §3.1 step 4: the arm re-validates cash-slot ownership via `cashItemInSlotFunc`. | `CharacterCashItemUseHandleFunc` **already** does this for every arm at `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:54-58` and returns early on mismatch. A second call would be dead code. | 12 |
| D2 | §3.3: the orchestrator consumes `EVENT_TOPIC_NPC_SHOP_STATUS` and completes the step. | `Processor.AcceptEvent(transactionId uuid.UUID, kind EventKind, …)` (`saga/processor.go:86`) correlates **only** by transaction id, and the npc-shop `Command`/`StatusEvent` structs (`services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go:12-16,63-67`) carry **no** transaction id. Threading one through the contract is mandatory and touches three modules. | 3 |
| D3 | §4.2: a missing shop yields `ERROR`, which fails the step. | `ProcessorImpl.Enter` (`services/atlas-npc-shops/atlas.com/npc/shops/processor.go:275-288`) returns the error and emits **nothing**. The saga step would hang until the saga timer expires and the client would never unlock. | 4 |
| D4 | §4.3: the plan phase should confirm whether `atlas-npc-shops` replies `ERROR` when the character is already in a shop. | It does not — `GetRegistry().AddCharacter` at `processor.go:284` overwrites unconditionally and emits `ENTERED` again. Confirmed: the fix belongs in `atlas-npc-shops`. | 4 |
| D5 | (not addressed) An enter failure could reuse the existing `ERROR` status event. | It must not. `handleErrorStatusEvent` (`services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop/consumer.go:88`) writes `NPCShopOperation`, and that packet with no outstanding request throws `CDisconnectException` (repo-recorded, `producer.go:36-51`). A distinct `ENTER_ERROR` type is required. | 3, 4, 11 |
| D6 | §3.3: `handleOpenNpcShop` "mirrors `handleShowStorage`". | `handleShowStorage` uses `h.storageP`, a `HandlerImpl` field threaded through ~15 `With*` builder methods. The lighter, equally-idiomatic precedent is `handleIncubatorResult` (`handler.go:1157-1172`), which emits directly via `producer.ProviderImpl` + a provider in `saga/producer.go`. Use that; no new `HandlerImpl` field. | 7 |
| D7 | §3.3: "the step's compensator emits `EXIT`". | The repo has an existing reverse-walk for consume-after-effect cash-item sagas: `compensateCashItemUse` + `DispatchCashItemUseRollbacks` (`compensator.go:1371,1428`), selected by saga type at `compensator.go:266-268`. Reuse it — add `RemoteMerchant` to the type list and an `OpenNpcShop` case to the rollback switch. No bespoke compensator. | 9 |
| D8 | (not addressed) The npc-shop contract now has three copies across three Go modules. | The repo already treats this exact failure mode as guard-worthy for trades (`tools/trade-contract-mirror-guard.sh`). Add the equivalent for npc-shop. | 3 |

---

## Task 1: `atlas-data` exposes the cash item's `info/npc`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/cash/rest.go:41-52`
- Modify: `services/atlas-data/atlas.com/data/cash/reader.go:76-80`
- Test: `services/atlas-data/atlas.com/data/cash/reader_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `cash.RestModel` gains exported field `Npc uint32` with json tag `"npc"`. Task 2 mirrors this on the channel side.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/cash/reader_test.go`. Match the existing tests' XML-fixture style in that file (they build an `xml.Node` tree and call `Read(l)(model.FixedProvider(node))()`); copy the nearest existing test's fixture helper rather than inventing a new one.

```go
func TestRead_ParsesInfoNpcForRemoteMerchantItems(t *testing.T) {
	l, _ := test.NewNullLogger()

	root := xml.Node{
		Name: "0545.img",
		ChildNodes: []xml.Node{
			{
				Name: "5450000",
				ChildNodes: []xml.Node{
					{
						Name: "info",
						IntegerNodes: []xml.IntegerNode{
							{Name: "cash", Value: 1},
							{Name: "npc", Value: 9090000},
							{Name: "slotMax", Value: 100},
						},
					},
				},
			},
			{
				Name: "5451000",
				ChildNodes: []xml.Node{
					{
						Name: "info",
						IntegerNodes: []xml.IntegerNode{
							{Name: "cash", Value: 1},
							{Name: "slotMax", Value: 100},
						},
					},
				},
			},
		},
	}

	res, err := Read(l)(model.FixedProvider(root))()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, want 2", len(res))
	}

	byId := make(map[uint32]RestModel, len(res))
	for _, m := range res {
		byId[m.Id] = m
	}

	if got := byId[5450000].Npc; got != 9090000 {
		t.Errorf("5450000 Npc = %d, want 9090000", got)
	}
	if got := byId[5451000].Npc; got != 0 {
		t.Errorf("5451000 Npc = %d, want 0 (item has no info/npc)", got)
	}
}
```

If the fixture shape above does not compile against `atlas-data/xml`, adapt the node literal to whatever the neighbouring tests in `reader_test.go` already use — the assertions are what matter.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-data/atlas.com/data && go test ./cash/ -run TestRead_ParsesInfoNpcForRemoteMerchantItems -v
```

Expected: FAIL — `byId[5450000].Npc` undefined (`RestModel` has no field `Npc`).

- [ ] **Step 3: Add the field**

In `services/atlas-data/atlas.com/data/cash/rest.go`, add to `RestModel` immediately after `StateChangeItem`:

```go
	// Npc is the WZ info/npc value: the NPC template a remote-merchant cash
	// item (classification 545) opens. 0 when the item targets no NPC.
	Npc uint32 `json:"npc,omitempty"`
```

- [ ] **Step 4: Parse it**

In `services/atlas-data/atlas.com/data/cash/reader.go`, immediately after the `m.StateChangeItem = …` line:

```go
			// info/npc — the shop NPC a remote-merchant item (0545.img) opens.
			// Mirrors consumable/reader.go's identical read.
			m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test ./cash/... -v
```

Expected: PASS, including the pre-existing `reader_test.go`, `rest_test.go` and `resource_test.go` cases.

- [ ] **Step 6: Module verification**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go vet ./... && go test -race ./...
```

Expected: all clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-data/atlas.com/data/cash/rest.go services/atlas-data/atlas.com/data/cash/reader.go services/atlas-data/atlas.com/data/cash/reader_test.go
git commit -m "feat(task-221): expose cash item info/npc in atlas-data"
```

---

## Task 2: `atlas-channel` reads the cash item's `npc`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/cash/rest.go:7-12`
- Test: `services/atlas-channel/atlas.com/channel/data/cash/rest_test.go` (create)

**Interfaces:**
- Consumes: Task 1's `"npc"` json field on the `cash_items` resource.
- Produces: `cash.RestModel.Npc uint32`, reachable from the handler as `cash.NewProcessor(l, ctx).GetById(itemId)` → `.Npc`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/data/cash/rest_test.go`:

```go
package cash

import (
	"encoding/json"
	"testing"
)

// TestRestModel_DecodesNpc locks the channel-side mirror of atlas-data's
// cash_items "npc" attribute (task-221). A missing field here decodes to 0
// silently and the remote-merchant arm would reject every use.
func TestRestModel_DecodesNpc(t *testing.T) {
	var m RestModel
	if err := json.Unmarshal([]byte(`{"npc":9090000,"protectTime":0}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Npc != 9090000 {
		t.Errorf("Npc = %d, want 9090000", m.Npc)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./data/cash/ -run TestRestModel_DecodesNpc -v
```

Expected: FAIL — `m.Npc` undefined.

- [ ] **Step 3: Add the field**

In `services/atlas-channel/atlas.com/channel/data/cash/rest.go`, inside `RestModel`:

```go
	// Npc is the WZ info/npc value served by atlas-data: the NPC template a
	// remote-merchant cash item (classification 545) opens. 0 when none.
	Npc uint32 `json:"npc"`
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./data/cash/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/cash/rest.go services/atlas-channel/atlas.com/channel/data/cash/rest_test.go
git commit -m "feat(task-221): expose cash item npc on the channel data model"
```

---

## Task 3: Thread a transaction id (and an `ENTER_ERROR` type) through the npc-shop Kafka contract

This is design delta **D2** and **D5**. The contract lives in three Go modules that the compiler does not link; Task 3 also adds the mechanical guard (**D8**).

**Files:**
- Modify: `services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/npc/shop/kafka.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npcshop/kafka.go`
- Modify: `services/atlas-npc-shops/atlas.com/npc/shops/producer.go`
- Modify: `services/atlas-npc-shops/atlas.com/npc/shops/processor.go`
- Modify: `services/atlas-npc-shops/atlas.com/npc/kafka/consumer/shops/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/npc/shops/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/npc/shops/processor.go`
- Create: `tools/npc-shop-contract-mirror-guard.sh`
- Modify: `.github/workflows/pr-validation.yml`
- Modify: `CLAUDE.md`
- Test: `services/atlas-npc-shops/atlas.com/npc/kafka/consumer/shops/consumer_test.go`

**Interfaces:**
- Produces, in all three copies:
  - `Command[E]` gains `TransactionId uuid.UUID` with json tag `transactionId` as its **first** field.
  - `StatusEvent[E]` gains `TransactionId uuid.UUID` with json tag `transactionId` as its **first** field.
  - `StatusEventTypeEnterError = "ENTER_ERROR"`, `EnterErrorShopNotFound = "SHOP_NOT_FOUND"`, `EnterErrorAlreadyInShop = "ALREADY_IN_SHOP"`.
  - `type StatusEventEnterErrorBody struct { NpcTemplateId uint32; Reason string }` with json tags `npcTemplateId`, `reason`.
- Produces (atlas-npc-shops): `Processor.EnterAndEmit(transactionId uuid.UUID, characterId uint32, npcId uint32) error` and `Processor.Enter(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(npcId uint32) error`.
- Produces (atlas-channel): `shops.Processor.EnterShop(transactionId uuid.UUID, characterId uint32, npcTemplateId uint32) error`.
- Consumed by: Tasks 4, 7, 8, 11.

Compatibility note: `uuid.UUID` unmarshals from an absent JSON field as `uuid.Nil`, so an in-flight message produced by an old pod decodes cleanly. The ordinary NPC-talk path passes `uuid.Nil`, which never matches a saga.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-npc-shops/atlas.com/npc/kafka/consumer/shops/consumer_test.go` (follow the file's existing `shop.Command[...]` construction style):

```go
// TestHandleEnterCommand_PropagatesTransactionId asserts the enter command's
// transaction id survives the wire round trip. It is the only correlation key
// the saga orchestrator has (task-221 design delta D2).
func TestHandleEnterCommand_PropagatesTransactionId(t *testing.T) {
	txn := uuid.New()
	cmd := shop.Command[shop.CommandShopEnterBody]{
		TransactionId: txn,
		CharacterId:   1234,
		Type:          shop.CommandShopEnter,
		Body:          shop.CommandShopEnterBody{NpcTemplateId: 9090000},
	}

	b, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round shop.Command[shop.CommandShopEnterBody]
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.TransactionId != txn {
		t.Errorf("TransactionId = %s, want %s", round.TransactionId, txn)
	}
}

// TestStatusEvent_EnterErrorRoundTrips locks the ENTER_ERROR shape. It must be
// distinct from StatusEventTypeError: the channel writes NPCShopOperation for
// ERROR, and that packet with no outstanding request disconnects the client
// (producer.go:36-51).
func TestStatusEvent_EnterErrorRoundTrips(t *testing.T) {
	txn := uuid.New()
	e := shop.StatusEvent[shop.StatusEventEnterErrorBody]{
		TransactionId: txn,
		CharacterId:   1234,
		Type:          shop.StatusEventTypeEnterError,
		Body:          shop.StatusEventEnterErrorBody{NpcTemplateId: 9090000, Reason: shop.EnterErrorShopNotFound},
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round shop.StatusEvent[shop.StatusEventEnterErrorBody]
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.TransactionId != txn || round.Type != "ENTER_ERROR" || round.Body.Reason != "SHOP_NOT_FOUND" {
		t.Errorf("round-trip mismatch: %+v", round)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go test ./kafka/consumer/shops/ -run 'TransactionId|EnterError' -v
```

Expected: FAIL — unknown fields `TransactionId`, `StatusEventTypeEnterError`, `StatusEventEnterErrorBody`.

- [ ] **Step 3: Edit the owning contract**

In `services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go`, replace the header, add the import, and update the two structs:

```go
// Package shops owns the COMMAND_TOPIC_NPC_SHOP / EVENT_TOPIC_NPC_SHOP_STATUS
// contract. atlas-channel and atlas-saga-orchestrator carry byte-identical
// mirrors of everything below the `package` clause because the three services
// are separate Go modules and nothing in the compiler links them: a field name
// or json tag changed here and not there decodes into a zero-valued body at
// runtime, silently. tools/npc-shop-contract-mirror-guard.sh checks this
// mechanically.
package shops

import "github.com/google/uuid"
```

```go
type Command[E any] struct {
	// TransactionId correlates a command with the saga step that issued it.
	// uuid.Nil for the ordinary NPC-talk path, which has no saga.
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}
```

```go
type StatusEvent[E any] struct {
	// TransactionId echoes the originating command's id so a saga can accept
	// the event. uuid.Nil when the command carried none.
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}
```

In the status-event const block, add alongside `StatusEventTypeError`:

```go
	// StatusEventTypeEnterError reports that an ENTER command failed. It is
	// deliberately NOT StatusEventTypeError: the channel renders that one as a
	// CONFIRM_SHOP_TRANSACTION packet, and CShopDlg::OnPacket @0x756da7 throws
	// CDisconnectException when that packet arrives with no buy/sell/recharge
	// request outstanding. An enter failure has none.
	StatusEventTypeEnterError = "ENTER_ERROR"

	// Reasons carried by StatusEventEnterErrorBody.
	EnterErrorShopNotFound  = "SHOP_NOT_FOUND"
	EnterErrorAlreadyInShop = "ALREADY_IN_SHOP"
```

And after `StatusEventErrorBody`:

```go
type StatusEventEnterErrorBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
	Reason        string `json:"reason"`
}
```

- [ ] **Step 4: Mirror into the other two modules**

The guard in Step 8 compares from the `package ` clause onward, so the `package` line itself must match across all three. Name all three packages `shops`:

- **atlas-npc-shops** — already `package shops`. This is the owner.
- **atlas-channel** — `services/atlas-channel/atlas.com/channel/kafka/message/npc/shop/kafka.go`. Change its package clause from `shop` to `shops` and copy the owner's body verbatim below it. Its importers already alias it (`shops2 "atlas-channel/kafka/message/npc/shop"` in `npc/shops/producer.go` and `kafka/consumer/npc/shop/consumer.go`), so only the clause changes — but re-run `go build ./...` to catch any unaliased importer. Give it this doc comment: `// Package shops mirrors atlas-npc-shops' COMMAND_TOPIC_NPC_SHOP / EVENT_TOPIC_NPC_SHOP_STATUS contract. atlas-npc-shops owns it; see tools/npc-shop-contract-mirror-guard.sh.`
- **atlas-saga-orchestrator** — new file `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npcshop/kafka.go`, `package shops`, same body, same style of mirror doc comment. Import it aliased: `npcshop "atlas-saga-orchestrator/kafka/message/npcshop"`.

- [ ] **Step 5: Thread the id through atlas-npc-shops**

`services/atlas-npc-shops/atlas.com/npc/shops/producer.go` — the enter providers take a transaction id; the buy/sell/recharge providers keep their signatures and set `TransactionId: uuid.Nil`:

```go
func enteredEventProvider(transactionId uuid.UUID, characterId uint32, npcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops.StatusEvent[shops.StatusEventEnteredBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          shops.StatusEventTypeEntered,
		Body: shops.StatusEventEnteredBody{
			NpcTemplateId: npcId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// enterErrorEventProvider reports a failed ENTER. See StatusEventTypeEnterError
// for why this is not errorEventProvider.
func enterErrorEventProvider(transactionId uuid.UUID, characterId uint32, npcId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops.StatusEvent[shops.StatusEventEnterErrorBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          shops.StatusEventTypeEnterError,
		Body: shops.StatusEventEnterErrorBody{
			NpcTemplateId: npcId,
			Reason:        reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`services/atlas-npc-shops/atlas.com/npc/shops/processor.go` — update the `Processor` interface entries and the impl. This step only threads the id; the behaviour change is Task 4:

```go
func (p *ProcessorImpl) EnterAndEmit(transactionId uuid.UUID, characterId uint32, npcId uint32) error {
	return message.Emit(p.kp)(func(mb *message.Buffer) error {
		return p.Enter(mb)(transactionId)(characterId)(npcId)
	})
}

func (p *ProcessorImpl) Enter(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(npcId uint32) error {
	return func(transactionId uuid.UUID) func(characterId uint32) func(npcId uint32) error {
		return func(characterId uint32) func(npcId uint32) error {
			return func(npcId uint32) error {
				p.l.Debugf("Character [%d] attempting to enter shop [%d].", characterId, npcId)
				_, err := p.GetByNpcId(p.CommodityDecorator)(npcId)
				if err != nil {
					p.l.WithError(err).Errorf("Cannot locate shop [%d] character [%d] is attempting to enter.", npcId, characterId)
					return err
				}
				GetRegistry().AddCharacter(p.ctx, characterId, npcId)
				return mb.Put(shops.EnvStatusEventTopic, enteredEventProvider(transactionId, characterId, npcId))
			}
		}
	}
}
```

`services/atlas-npc-shops/atlas.com/npc/kafka/consumer/shops/consumer.go` — in `handleEnterCommand`:

```go
		err := shops.NewProcessor(l, ctx, db).EnterAndEmit(e.TransactionId, e.CharacterId, e.Body.NpcTemplateId)
```

Update any mock implementing `Processor` (`grep -rn 'EnterAndEmit' services/atlas-npc-shops --include='*.go'`) with the identical signature.

- [ ] **Step 6: Thread the id through atlas-channel**

`services/atlas-channel/atlas.com/channel/npc/shops/producer.go`:

```go
func ShopEnterCommandProvider(transactionId uuid.UUID, characterId uint32, npcTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops2.Command[shops2.CommandShopEnterBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          shops2.CommandShopEnter,
		Body: shops2.CommandShopEnterBody{
			NpcTemplateId: npcTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`services/atlas-channel/atlas.com/channel/npc/shops/processor.go` — update the `Processor` interface and impl:

```go
// EnterShop issues an ENTER command. transactionId is uuid.Nil for the
// ordinary NPC-talk path; the remote-merchant flow drives ENTER from a saga
// step instead and never calls this.
func (p *ProcessorImpl) EnterShop(transactionId uuid.UUID, characterId uint32, npcTemplateId uint32) error {
	p.l.Debugf("Character [%d] is entering NPC shop [%d].", characterId, npcTemplateId)
	return producer.ProviderImpl(p.l)(p.ctx)(shops2.EnvCommandTopic)(ShopEnterCommandProvider(transactionId, characterId, npcTemplateId))
}
```

Update every caller (`grep -rn 'EnterShop(' services/atlas-channel --include='*.go'`) to pass `uuid.Nil` first, and the mock under `services/atlas-channel/atlas.com/channel/npc/shops/mock/` identically.

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go build ./... && go test ./... 2>&1 | tail -30
```

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./npc/... ./kafka/... 2>&1 | tail -30
```

Expected: PASS.

- [ ] **Step 8: Add the mirror guard**

Create `tools/npc-shop-contract-mirror-guard.sh`. Read `tools/trade-contract-mirror-guard.sh` first and follow its structure exactly:

```bash
#!/usr/bin/env bash
# npc-shop-contract-mirror-guard.sh — enforces that the COMMAND_TOPIC_NPC_SHOP /
# EVENT_TOPIC_NPC_SHOP_STATUS contract is identical in its three copies.
#
# atlas-npc-shops owns the contract; atlas-channel and atlas-saga-orchestrator
# carry mirrors because the three services live in separate Go modules and
# nothing in the compiler links them. A field name or json tag changed in one
# copy and not the others does not fail any build — it decodes into a
# zero-valued body at runtime, silently. task-221.
#
# The files are compared from their `package` clause onward: the only permitted
# difference is the leading doc comment, which names the mirror direction.
#
# Run from the repo root; drift → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OWNER="$ROOT/services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go"
CHANNEL_MIRROR="$ROOT/services/atlas-channel/atlas.com/channel/kafka/message/npc/shop/kafka.go"
SAGA_MIRROR="$ROOT/services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npcshop/kafka.go"

rc=0
for f in "$OWNER" "$CHANNEL_MIRROR" "$SAGA_MIRROR"; do
    if [ ! -f "$f" ]; then
        echo "npc-shop-contract-mirror-guard: FAIL — missing contract file: ${f#"$ROOT"/}"
        rc=1
    fi
done
[ "$rc" -ne 0 ] && exit "$rc"

body() { awk '/^package /{p=1} p' "$1"; }

check_pair() {
    local owner="$1" mirror="$2" label="$3"
    if ! diff -u <(body "$owner") <(body "$mirror"); then
        echo "npc-shop-contract-mirror-guard: FAIL — $label drift (diff above)."
        return 1
    fi
    return 0
}

check_pair "$OWNER" "$CHANNEL_MIRROR" "atlas-channel mirror" || rc=1
check_pair "$OWNER" "$SAGA_MIRROR" "atlas-saga-orchestrator mirror" || rc=1

if [ "$rc" -eq 0 ]; then
    echo "npc-shop-contract-mirror-guard: OK — all three copies identical."
fi
exit "$rc"
```

Then:

```bash
cd "$(git rev-parse --show-toplevel)" && chmod +x tools/npc-shop-contract-mirror-guard.sh && ./tools/npc-shop-contract-mirror-guard.sh
```

Expected: `OK — all three copies identical.` If it reports drift, fix the mirrors until it passes — do not weaken the guard.

- [ ] **Step 9: Wire the guard into CI and CLAUDE.md**

In `.github/workflows/pr-validation.yml`, add a job modelled on the `redis-key-guard` job at line 85 (same `needs`/`if`/checkout shape; no Go setup, the guard is pure shell):

```yaml
  npc-shop-contract-mirror-guard:
    name: NPC Shop Contract Mirror Guard
    needs: detect-changes
    if: needs.detect-changes.outputs.deploy-only != 'true'
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: npc-shop contract mirror guard
        run: ./tools/npc-shop-contract-mirror-guard.sh
```

In `CLAUDE.md`, append to the numbered "Build & Verification" list, after item 13:

```markdown
14. **`tools/npc-shop-contract-mirror-guard.sh` clean from the repo root** whenever
    any copy of the npc-shop Kafka contract changed. atlas-npc-shops owns
    `kafka/message/shops/kafka.go`; atlas-channel and atlas-saga-orchestrator
    carry mirrors in separate Go modules, so a field name or json tag changed in
    one and not the others fails no build — it decodes into a zero-valued body at
    runtime, silently. The guard diffs all three from their `package` clause
    onward; only the leading doc comment, which names the mirror direction, may
    differ.
```

- [ ] **Step 10: Commit**

```bash
git add services/atlas-npc-shops services/atlas-channel/atlas.com/channel/kafka/message/npc services/atlas-channel/atlas.com/channel/npc/shops services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npcshop tools/npc-shop-contract-mirror-guard.sh .github/workflows/pr-validation.yml CLAUDE.md
git commit -m "feat(task-221): carry a transaction id and an ENTER_ERROR type on the npc-shop contract"
```

---

## Task 4: `atlas-npc-shops` reports enter failures

Design deltas **D3** and **D4**. Today a missing shop emits nothing and an already-in-shop character silently re-enters.

**Files:**
- Modify: `services/atlas-npc-shops/atlas.com/npc/shops/processor.go` (`Enter`)
- Test: `services/atlas-npc-shops/atlas.com/npc/shops/processor_test.go` (create if absent)

**Interfaces:**
- Consumes: Task 3's `enterErrorEventProvider`, `EnterErrorShopNotFound`, `EnterErrorAlreadyInShop`.
- Produces: `Enter` now always emits exactly one status event — `ENTERED` on success, `ENTER_ERROR` on either failure — and returns `nil` on the `ENTER_ERROR` paths (the failure is reported on the topic, not by returning an error that only gets logged).

- [ ] **Step 1: Write the failing tests**

Create or extend `services/atlas-npc-shops/atlas.com/npc/shops/processor_test.go`. Drive `Enter` through a `message.Buffer` and assert on the buffered messages — that is the seam the repo uses for `*AndEmit`/pure-method pairs. Before writing, read `services/atlas-npc-shops/atlas.com/npc/kafka/consumer/shops/consumer_test.go` and any `testmain_test.go` in the module and reuse their logger/context/db fixtures rather than inventing new ones. If `message.Buffer` exposes its contents under a different accessor than `GetAll()`, use whatever `libs/atlas-kafka/message` actually provides.

```go
// TestEnter_ShopNotFound_EmitsEnterError: without this the saga step hangs
// until the saga timer expires and the player's client never unlocks
// (task-221 design delta D3).
func TestEnter_ShopNotFound_EmitsEnterError(t *testing.T) {
	// db seeded with NO shop for npc 9090000
	p := NewProcessor(logger, ctx, db)
	mb := message.NewBuffer()
	txn := uuid.New()

	if err := p.Enter(mb)(txn)(1234)(9090000); err != nil {
		t.Fatalf("Enter returned err %v; the failure must be reported on the topic, not returned", err)
	}

	msgs := mb.GetAll()[shops.EnvStatusEventTopic]
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	var e shops.StatusEvent[shops.StatusEventEnterErrorBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Type != shops.StatusEventTypeEnterError {
		t.Errorf("Type = %q, want %q", e.Type, shops.StatusEventTypeEnterError)
	}
	if e.TransactionId != txn {
		t.Errorf("TransactionId = %s, want %s", e.TransactionId, txn)
	}
	if e.Body.Reason != shops.EnterErrorShopNotFound {
		t.Errorf("Reason = %q, want %q", e.Body.Reason, shops.EnterErrorShopNotFound)
	}
}

// TestEnter_AlreadyInShop_EmitsEnterError: AddCharacter used to overwrite
// unconditionally, so a second ENTER silently re-entered and a remote-merchant
// saga would consume the item for a shop the player was already standing in
// (task-221 design delta D4, PRD FR-2.3).
func TestEnter_AlreadyInShop_EmitsEnterError(t *testing.T) {
	// db seeded WITH a shop for npc 9090000
	p := NewProcessor(logger, ctx, db)
	GetRegistry().AddCharacter(ctx, 1234, 9090000)
	t.Cleanup(func() { GetRegistry().RemoveCharacter(ctx, 1234) })

	mb := message.NewBuffer()
	txn := uuid.New()
	if err := p.Enter(mb)(txn)(1234)(9090000); err != nil {
		t.Fatalf("Enter: %v", err)
	}

	msgs := mb.GetAll()[shops.EnvStatusEventTopic]
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	var e shops.StatusEvent[shops.StatusEventEnterErrorBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Body.Reason != shops.EnterErrorAlreadyInShop {
		t.Errorf("Reason = %q, want %q", e.Body.Reason, shops.EnterErrorAlreadyInShop)
	}
}

// TestEnter_Success_EmitsEnteredWithTransactionId
func TestEnter_Success_EmitsEnteredWithTransactionId(t *testing.T) {
	// db seeded WITH a shop for npc 9090000, character not in any shop
	p := NewProcessor(logger, ctx, db)
	mb := message.NewBuffer()
	txn := uuid.New()
	if err := p.Enter(mb)(txn)(1234)(9090000); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	t.Cleanup(func() { GetRegistry().RemoveCharacter(ctx, 1234) })

	msgs := mb.GetAll()[shops.EnvStatusEventTopic]
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	var e shops.StatusEvent[shops.StatusEventEnteredBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Type != shops.StatusEventTypeEntered || e.TransactionId != txn || e.Body.NpcTemplateId != 9090000 {
		t.Errorf("unexpected entered event: %+v", e)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go test ./shops/ -run 'TestEnter_' -v
```

Expected: FAIL — shop-not-found emits nothing; already-in-shop emits `ENTERED`.

- [ ] **Step 3: Implement**

Replace the innermost body of `Enter` in `services/atlas-npc-shops/atlas.com/npc/shops/processor.go`:

```go
			return func(npcId uint32) error {
				p.l.Debugf("Character [%d] attempting to enter shop [%d].", characterId, npcId)

				// The shop must exist. Reporting this on the topic rather than
				// returning it is what lets a saga step fail (and a remote
				// merchant item survive) instead of hanging until the saga
				// timer expires — task-221 design delta D3.
				_, err := p.GetByNpcId(p.CommodityDecorator)(npcId)
				if err != nil {
					p.l.WithError(err).Errorf("Cannot locate shop [%d] character [%d] is attempting to enter.", npcId, characterId)
					return mb.Put(shops.EnvStatusEventTopic, enterErrorEventProvider(transactionId, characterId, npcId, shops.EnterErrorShopNotFound))
				}

				// One exclusive dialog at a time. AddCharacter overwrites, so
				// without this guard a second ENTER silently re-enters and a
				// remote-merchant saga would consume the item for a shop the
				// player is already standing in (PRD FR-2.3, delta D4).
				if existing, inShop := GetRegistry().GetShop(p.ctx, characterId); inShop {
					p.l.Warnf("Character [%d] attempted to enter shop [%d] while already in shop [%d]; rejecting.", characterId, npcId, existing)
					return mb.Put(shops.EnvStatusEventTopic, enterErrorEventProvider(transactionId, characterId, npcId, shops.EnterErrorAlreadyInShop))
				}

				GetRegistry().AddCharacter(p.ctx, characterId, npcId)
				return mb.Put(shops.EnvStatusEventTopic, enteredEventProvider(transactionId, characterId, npcId))
			}
```

Confirm `GetRegistry().GetShop(ctx, characterId) (uint32, bool)` is the actual signature — `Exit` at `processor.go:297` already uses it that way.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go test -race ./... 2>&1 | tail -20
```

Expected: PASS.

- [ ] **Step 5: Module verification**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go build ./... && go vet ./...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-npc-shops
git commit -m "fix(task-221): report npc-shop enter failures instead of emitting nothing"
```

---

## Task 5: `libs/atlas-saga` — `open_npc_shop` action and payload

**Files:**
- Modify: `libs/atlas-saga/model.go:13-44` (saga types), `:57-…` (actions)
- Modify: `libs/atlas-saga/payloads.go`
- Modify: `libs/atlas-saga/unmarshal.go`
- Test: `libs/atlas-saga/unmarshal_test.go`

**Interfaces:**
- Produces:
  - `saga.RemoteMerchant Type = "remote_merchant"`
  - `saga.OpenNpcShop Action = "open_npc_shop"`
  - `saga.OpenNpcShopPayload{CharacterId uint32; WorldId world.Id; ChannelId channel.Id; NpcTemplateId uint32}`
- Consumed by: Tasks 6, 7, 9, 12.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-saga/unmarshal_test.go`. The file has ~36KB of per-action precedent — copy the nearest case verbatim and change only the action and payload:

```go
func TestUnmarshalStep_OpenNpcShop(t *testing.T) {
	raw := []byte(`{
		"stepId": "open_npc_shop",
		"status": "pending",
		"action": "open_npc_shop",
		"payload": {"characterId": 1234, "worldId": 0, "channelId": 1, "npcTemplateId": 9090000}
	}`)

	var s Step[any]
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := s.Payload.(OpenNpcShopPayload)
	if !ok {
		t.Fatalf("payload type = %T, want OpenNpcShopPayload", s.Payload)
	}
	if p.CharacterId != 1234 || p.NpcTemplateId != 9090000 || p.ChannelId != channel.Id(1) {
		t.Errorf("payload = %+v", p)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-saga && go test ./... -run TestUnmarshalStep_OpenNpcShop -v
```

Expected: FAIL — `OpenNpcShopPayload` undefined.

- [ ] **Step 3: Add the saga type**

In `libs/atlas-saga/model.go`, append to the `Type` const block (after `MegaphoneUse`):

```go
	// RemoteMerchant is the classification-545 cash item flow: open an NPC's
	// shop from anywhere, then consume the item — never the other way round
	// (task-221).
	RemoteMerchant Type = "remote_merchant"
```

- [ ] **Step 4: Add the action**

In the same file's `Action` const block, add a new group after the storage block:

```go
	// NPC shop actions
	OpenNpcShop Action = "open_npc_shop"
```

- [ ] **Step 5: Add the payload**

In `libs/atlas-saga/payloads.go`, next to `ShowStoragePayload` (line 496):

```go
// OpenNpcShopPayload represents the payload required to open an NPC's shop for
// a character. Unlike ShowStorage this step is NOT self-completing: it waits
// for the npc-shop status topic to report ENTERED or ENTER_ERROR, which is what
// lets a following destroy step consume the cash item only once the shop
// actually opened (task-221 FR-4.3).
type OpenNpcShopPayload struct {
	CharacterId   uint32     `json:"characterId"`   // CharacterId the shop is opened for
	WorldId       world.Id   `json:"worldId"`       // WorldId associated with the action
	ChannelId     channel.Id `json:"channelId"`     // ChannelId associated with the action
	NpcTemplateId uint32     `json:"npcTemplateId"` // NPC template whose shop to open
}
```

- [ ] **Step 6: Add the unmarshal case**

In `libs/atlas-saga/unmarshal.go`, next to the `case ShowStorage:` arm (line 312):

```go
	case OpenNpcShop:
		var payload OpenNpcShopPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd libs/atlas-saga && go build ./... && go vet ./... && go test -race ./...
```

Expected: PASS, including `payloads_test.go`.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-saga
git commit -m "feat(task-221): add open_npc_shop saga action and remote_merchant saga type"
```

---

## Task 6: Orchestrator aliases the new action, payload and saga type

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go:131`, `:275`, `:1315`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor.go`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor_test.go`

**Interfaces:**
- Consumes: Task 5's `sharedsaga.OpenNpcShop`, `sharedsaga.OpenNpcShopPayload`, `sharedsaga.RemoteMerchant`.
- Produces: package-local aliases `OpenNpcShop`, `OpenNpcShopPayload`, `RemoteMerchant`; `ExtractCharacterId` returns the character id for an `OpenNpcShopPayload` step.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor_test.go`, using whatever step constructor the neighbouring test in that file already uses:

```go
func TestExtractCharacterId_OpenNpcShop(t *testing.T) {
	st := NewStep[any]("open_npc_shop", Pending, OpenNpcShop, OpenNpcShopPayload{
		CharacterId:   4242,
		NpcTemplateId: 9090000,
	})
	if got := ExtractCharacterId(st); got != 4242 {
		t.Errorf("ExtractCharacterId = %d, want 4242", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestExtractCharacterId_OpenNpcShop -v
```

Expected: FAIL — `OpenNpcShop` / `OpenNpcShopPayload` undefined.

- [ ] **Step 3: Add the aliases**

In `saga/model.go`, next to `ShowStorage = sharedsaga.ShowStorage` (line 131):

```go
	OpenNpcShop = sharedsaga.OpenNpcShop
```

Next to `ShowStoragePayload = sharedsaga.ShowStoragePayload` (line 275):

```go
	OpenNpcShopPayload = sharedsaga.OpenNpcShopPayload
```

And in the saga-`Type` alias block (find `MegaphoneUse = sharedsaga.MegaphoneUse` and follow its form):

```go
	RemoteMerchant = sharedsaga.RemoteMerchant
```

- [ ] **Step 4: Add the unmarshal case**

In `saga/model.go`, next to `case ShowStorage:` (line 1315). Copy the `ShowStorage` arm and change only the action and type — the orchestrator's private model uses different field names (`s.action`/`s.payload`) than the shared lib, so do not transplant the shared-lib arm:

```go
	case OpenNpcShop:
		var payload OpenNpcShopPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
```

- [ ] **Step 5: Add the character extractor case**

In `saga/character_extractor.go`, add to the type switch:

```go
	case OpenNpcShopPayload:
		return p.CharacterId
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./saga/ -run 'TestExtractCharacterId|Completeness' -v
```

Expected: PASS, including `unmarshal_completeness_test.go`.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor_test.go
git commit -m "feat(task-221): alias open_npc_shop into the saga orchestrator model"
```

---

## Task 7: Orchestrator handler, producer and acceptance table

Design delta **D6**: emit directly via `producer.ProviderImpl`, following `handleIncubatorResult` — no new `HandlerImpl` field.

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` (dispatch switch near line 849; new func after `handleIncubatorResult` line 1172)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go:108-112`, `:169`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go:14-52`, `.../producer_test.go`

**Interfaces:**
- Consumes: Task 3's orchestrator mirror package (`npcshop "atlas-saga-orchestrator/kafka/message/npcshop"`, package name `shops`), Task 6's aliases.
- Produces:
  - `NpcShopEnterCommandProvider(transactionId uuid.UUID, payload OpenNpcShopPayload) model.Provider[[]kafka.Message]`
  - `NpcShopExitCommandProvider(transactionId uuid.UUID, characterId uint32) model.Provider[[]kafka.Message]`
  - `EventKindNpcShopEntered EventKind = "npcshop.entered"`, `EventKindNpcShopError EventKind = "npcshop.error"`
  - `acceptanceTable[sharedsaga.OpenNpcShop] = {EventKindNpcShopEntered, EventKindNpcShopError}`
  - `(*HandlerImpl).handleOpenNpcShop(s Saga, st Step[any]) error` — leaves the step `Pending`.
- Consumed by: Tasks 8, 9.

- [ ] **Step 1: Write the failing tests**

In `saga/event_acceptance_test.go`, add `sharedsaga.OpenNpcShop` to the `allActions` slice (append after the line containing `sharedsaga.EmitMegaphone, sharedsaga.EnqueueWorldBroadcast,`):

```go
	sharedsaga.OpenNpcShop,
```

and add to the `TestStepAcceptsEvent_KnownSuccessKinds` case table:

```go
		{sharedsaga.OpenNpcShop, EventKindNpcShopEntered},
```

Append to `saga/producer_test.go`:

```go
// TestNpcShopEnterCommandProvider asserts the ENTER command carries the saga's
// transaction id — the orchestrator's only correlation key when the ENTERED
// event comes back (task-221 design delta D2).
func TestNpcShopEnterCommandProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := NpcShopEnterCommandProvider(txn, OpenNpcShopPayload{
		CharacterId:   1234,
		WorldId:       world.Id(0),
		ChannelId:     channel.Id(1),
		NpcTemplateId: 9090000,
	})()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	var cmd npcshop.Command[npcshop.CommandShopEnterBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.TransactionId != txn {
		t.Errorf("TransactionId = %s, want %s", cmd.TransactionId, txn)
	}
	if cmd.Type != npcshop.CommandShopEnter {
		t.Errorf("Type = %q, want %q", cmd.Type, npcshop.CommandShopEnter)
	}
	if cmd.CharacterId != 1234 || cmd.Body.NpcTemplateId != 9090000 {
		t.Errorf("unexpected command: %+v", cmd)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run 'EveryActionRepresented|NpcShopEnterCommandProvider' -v
```

Expected: FAIL — missing `acceptanceTable` entry; `NpcShopEnterCommandProvider` undefined.

- [ ] **Step 3: Add the event kinds and acceptance entry**

In `saga/event_acceptance.go`, append to the `EventKind` const block after the `Note` group:

```go
	// NPC shop (atlas-npc-shops acks on EVENT_TOPIC_NPC_SHOP_STATUS, task-221).
	EventKindNpcShopEntered EventKind = "npcshop.entered"
	EventKindNpcShopError   EventKind = "npcshop.error"
```

In `acceptanceTable`, next to the `sharedsaga.ShowStorage: {}` entry (line 169):

```go
	// Unlike ShowStorage this is NOT self-completing: the item is consumed by a
	// following step, so the shop must be confirmed open first (task-221 FR-4.3).
	sharedsaga.OpenNpcShop: {EventKindNpcShopEntered, EventKindNpcShopError},
```

- [ ] **Step 4: Add the producers**

In `saga/producer.go`, after `IncubatorResultEventProvider`, and add the import `npcshop "atlas-saga-orchestrator/kafka/message/npcshop"`:

```go
// NpcShopEnterCommandProvider builds the COMMAND_TOPIC_NPC_SHOP ENTER command
// for an open_npc_shop step. Keyed by character id, matching every other
// producer on this topic (atlas-channel's npc/shops/producer.go).
func NpcShopEnterCommandProvider(transactionId uuid.UUID, payload OpenNpcShopPayload) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(payload.CharacterId))
	value := &npcshop.Command[npcshop.CommandShopEnterBody]{
		TransactionId: transactionId,
		CharacterId:   payload.CharacterId,
		Type:          npcshop.CommandShopEnter,
		Body: npcshop.CommandShopEnterBody{
			NpcTemplateId: payload.NpcTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// NpcShopExitCommandProvider builds the COMMAND_TOPIC_NPC_SHOP EXIT command
// used to compensate an open_npc_shop step whose saga later failed, so the
// player is not left standing in a shop they did not pay for (FR-4.5).
func NpcShopExitCommandProvider(transactionId uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npcshop.Command[npcshop.CommandShopExitBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          npcshop.CommandShopExit,
		Body:          npcshop.CommandShopExitBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Add the handler**

In `saga/handler.go`, after `handleIncubatorResult` (line 1172):

```go
// handleOpenNpcShop handles the OpenNpcShop action.
//
// Deliberately NOT self-completing (contrast handleShowStorage): the step stays
// Pending until the npc-shop status consumer reports ENTERED or ENTER_ERROR.
// That is the whole point of the remote-merchant saga — the following
// destroy_asset_from_slot step must not run unless the shop actually opened
// (task-221 FR-4.3, FR-4.4).
func (h *HandlerImpl) handleOpenNpcShop(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(OpenNpcShopPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := producer.ProviderImpl(h.l)(h.ctx)(npcshop.EnvCommandTopic)(NpcShopEnterCommandProvider(s.TransactionId(), payload))
	if err != nil {
		h.logActionError(s, st, err, "Unable to emit npc shop enter command.")
		return err
	}

	h.l.WithFields(logrus.Fields{
		"transaction_id":  s.TransactionId().String(),
		"character_id":    payload.CharacterId,
		"npc_template_id": payload.NpcTemplateId,
	}).Debug("Dispatched npc shop ENTER; awaiting ENTERED/ENTER_ERROR.")

	return nil
}
```

And in the dispatch switch (near `case ShowStorage:` at line 849), matching the surrounding arms' exact return form:

```go
	case OpenNpcShop:
		return h.handleOpenNpcShop(s, st)
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./saga/ -run 'AcceptanceTable|NpcShop' -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga
git commit -m "feat(task-221): dispatch open_npc_shop and accept npc-shop status events"
```

---

## Task 8: Orchestrator consumes `EVENT_TOPIC_NPC_SHOP_STATUS`

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/consumer.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/consumer_test.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/testmain_test.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go` (imports; `InitConsumers` near line 117; `InitHandlers` near line 167)

**Interfaces:**
- Consumes: Task 7's `EventKindNpcShopEntered`, `EventKindNpcShopError`; Task 3's mirror package.
- Produces: `npcshop.InitConsumers(l)(cmf)(consumerGroupId)` and `npcshop.InitHandlers(l)(rf) error`, same curried shapes as the `storage` consumer.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/testmain_test.go` as a copy of `../storage/testmain_test.go` with the package renamed to `npcshop`.

Create `consumer_test.go`, modelled on `../storage/consumer_test.go`. Reuse that file's tenant-scoped context helper rather than writing a new one; if it has none, build the context with `tenant.WithContext(context.Background(), <tenant.Create(...)>)` the way `testmain_test.go` does:

```go
package npcshop

import (
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	npcshop "atlas-saga-orchestrator/kafka/message/npcshop"
)

// TestHandleEnteredEvent_IgnoresWrongType guards the shared-topic fan-out: every
// handler registered on EVENT_TOPIC_NPC_SHOP_STATUS sees every event, so the
// type check is what stops an EXITED from completing an open_npc_shop step.
func TestHandleEnteredEvent_IgnoresWrongType(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleEnteredEvent(l, ctx, npcshop.StatusEvent[npcshop.StatusEventEnteredBody]{
		TransactionId: uuid.New(),
		CharacterId:   1234,
		Type:          npcshop.StatusEventTypeExited,
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a non-ENTERED event: %v", hook.Entries)
	}
}

// TestHandleEnteredEvent_NilTransactionIgnored: the ordinary NPC-talk path
// produces ENTER with uuid.Nil, and every one of those events lands here. It
// must never advance a saga.
func TestHandleEnteredEvent_NilTransactionIgnored(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleEnteredEvent(l, ctx, npcshop.StatusEvent[npcshop.StatusEventEnteredBody]{
		TransactionId: uuid.Nil,
		CharacterId:   1234,
		Type:          npcshop.StatusEventTypeEntered,
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a uuid.Nil transaction: %v", hook.Entries)
	}
}

// TestHandleEnterErrorEvent_NilTransactionIgnored
func TestHandleEnterErrorEvent_NilTransactionIgnored(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleEnterErrorEvent(l, ctx, npcshop.StatusEvent[npcshop.StatusEventEnterErrorBody]{
		TransactionId: uuid.Nil,
		CharacterId:   1234,
		Type:          npcshop.StatusEventTypeEnterError,
		Body:          npcshop.StatusEventEnterErrorBody{Reason: npcshop.EnterErrorShopNotFound},
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a uuid.Nil transaction: %v", hook.Entries)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./kafka/consumer/npcshop/ -v
```

Expected: FAIL — package has no `handleEnteredEvent`.

- [ ] **Step 3: Write the consumer**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/consumer.go`:

```go
package npcshop

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	npcshop "atlas-saga-orchestrator/kafka/message/npcshop"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("npc_shop_status_event")(npcshop.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(npcshop.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnteredEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterErrorEvent))); err != nil {
			return err
		}
		return nil
	}
}

// handleEnteredEvent completes a pending open_npc_shop step. The ordinary
// NPC-talk path produces ENTER with uuid.Nil, so most events reaching this
// handler match no saga — the nil check and AcceptEvent both decline them.
//
// A redelivered ENTERED for an already-completed step is also declined by
// AcceptEvent, which is the idempotency guarantee the at-least-once topic needs
// (task-221 NFR Idempotency).
func handleEnteredEvent(l logrus.FieldLogger, ctx context.Context, e npcshop.StatusEvent[npcshop.StatusEventEnteredBody]) {
	if e.Type != npcshop.StatusEventTypeEntered {
		return
	}
	if e.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNpcShopEntered); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":  e.TransactionId.String(),
		"character_id":    e.CharacterId,
		"npc_template_id": e.Body.NpcTemplateId,
	}).Debug("NPC shop entered; completing open_npc_shop step.")

	_ = p.StepCompleted(e.TransactionId, true)
}

// handleEnterErrorEvent fails the step, which fails the saga, which means the
// following destroy_asset_from_slot never runs — the cash item survives
// (task-221 FR-4.4).
func handleEnterErrorEvent(l logrus.FieldLogger, ctx context.Context, e npcshop.StatusEvent[npcshop.StatusEventEnterErrorBody]) {
	if e.Type != npcshop.StatusEventTypeEnterError {
		return
	}
	if e.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNpcShopError); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":  e.TransactionId.String(),
		"character_id":    e.CharacterId,
		"npc_template_id": e.Body.NpcTemplateId,
		"reason":          e.Body.Reason,
	}).Error("NPC shop enter failed; failing open_npc_shop step.")

	_ = p.StepCompleted(e.TransactionId, false)
}
```

- [ ] **Step 4: Register in main**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go`:

- add the import `npcshopconsumer "atlas-saga-orchestrator/kafka/consumer/npcshop"` alongside the other consumer imports;
- next to `storage.InitConsumers(l)(cmf)(consumerGroupId)` (line 117): `npcshopconsumer.InitConsumers(l)(cmf)(consumerGroupId)`;
- next to the `storage.InitHandlers` block (line 167), copying its exact error-handling form and log level:

```go
	if err := npcshopconsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register npc shop status handlers.")
	}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./kafka/consumer/npcshop/ -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go
git commit -m "feat(task-221): consume npc-shop status events in the saga orchestrator"
```

---

## Task 9: Compensate a failed remote-merchant saga with `EXIT`

Design delta **D7**: reuse `compensateCashItemUse` / `DispatchCashItemUseRollbacks`.

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go` (emit + seam)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer_testseam.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go:266-268`, `:1428` (rollback switch), plus the doc comments at `:109-114`, `:1361`, `:1411-1427`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/remote_merchant_compensation_test.go` (create)

**Build tag:** `producer_testseam.go` is `//go:build test`, and the test files that use its setters (`late_event_integration_test.go`) carry the same tag. The new test file must therefore also start with `//go:build test`, and is only run by `go test -tags=test`.

**Interfaces:**
- Consumes: Task 5's `RemoteMerchant`, `OpenNpcShop`, `OpenNpcShopPayload`; Task 7's `NpcShopExitCommandProvider`.
- Produces: `EmitNpcShopExit(l, ctx, transactionId, characterId) error` and `SetEmitNpcShopExitForTest(fn) func(...)`. A failed `RemoteMerchant` saga reverse-walks its completed steps, emitting `EXIT` for a completed `open_npc_shop` and re-creating a destroyed asset for a completed `destroy_asset_from_slot`.

- [ ] **Step 1: Add the emitter and its seam**

The compensator must be testable without a broker. In `saga/producer.go`, next to the other `…Fn` seams (`emitConversationRewardNoticeFn` etc.):

Note the arrangement the module already uses: the emit function and its `…Fn` package var live in the **untagged** `producer.go`; only the setter goes in the `//go:build test` `producer_testseam.go`.

```go
// EmitNpcShopExit emits the EXIT compensation command for an open_npc_shop
// step. Indirected through a package var so compensator tests can observe it
// without a broker (same seam shape as emitConversationRewardNoticeFn).
func EmitNpcShopExit(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, characterId uint32) error {
	return emitNpcShopExitFn(l, ctx, transactionId, characterId)
}

var emitNpcShopExitFn = emitNpcShopExitImpl

func emitNpcShopExitImpl(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, characterId uint32) error {
	return producer.ProviderImpl(l)(ctx)(npcshop.EnvCommandTopic)(NpcShopExitCommandProvider(transactionId, characterId))
}
```

In `saga/producer_testseam.go` (already `//go:build test`):

```go
// SetEmitNpcShopExitForTest swaps the underlying EXIT-emission function and
// returns the previous one for restoration. Compiled only with -tags=test.
func SetEmitNpcShopExitForTest(fn func(logrus.FieldLogger, context.Context, uuid.UUID, uint32) error) func(logrus.FieldLogger, context.Context, uuid.UUID, uint32) error {
	prev := emitNpcShopExitFn
	emitNpcShopExitFn = fn
	return prev
}
```

- [ ] **Step 2: Write the failing test**

Create `saga/remote_merchant_compensation_test.go`. This mirrors `point_reset_compensation_test.go:34-97` exactly — the same `NewBuilder()` saga construction, the same `NewCompensator(logger, testTenantContext())`, and the same "exercise `Dispatch*Rollbacks` directly to avoid the `EmitSagaFailed` Kafka path" rationale (no broker runs in the test environment). `testTenantContext()` already exists in the package at `point_reset_compensation_test.go:21`; reuse it, do not redeclare it.

```go
//go:build test

package saga

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	compmock "atlas-saga-orchestrator/compartment/mock"
)

// TestRemoteMerchantCompensationEmitsShopExit verifies the remote_merchant
// reverse-walk (task-221 FR-4.5): when consume_remote_merchant_item fails, the
// already-completed open_npc_shop is inverted with an EXIT command so the
// player is not left standing in a shop they did not pay for.
//
// DispatchCashItemUseRollbacks is exercised directly (mirroring the point-reset
// and pet-evolution compensation tests) to avoid the EmitSagaFailed Kafka path.
func TestRemoteMerchantCompensationEmitsShopExit(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	const (
		testCharId = uint32(88221)
		miuMiuItem = uint32(5450000)
		miuMiuNpc  = uint32(9090000)
	)

	var exitCalls []uint32
	origExit := SetEmitNpcShopExitForTest(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, characterId uint32) error {
		exitCalls = append(exitCalls, characterId)
		return nil
	})
	t.Cleanup(func() { SetEmitNpcShopExitForTest(origExit) })

	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			return nil
		},
	}

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(RemoteMerchant).
		SetInitiatedBy("remote-merchant-compensation-test").
		AddStep("open_npc_shop", Completed, OpenNpcShop, OpenNpcShopPayload{
			CharacterId:   testCharId,
			NpcTemplateId: miuMiuNpc,
		}).
		AddStep("consume_remote_merchant_item", Failed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId:   testCharId,
			InventoryType: 5,
			Slot:          3,
			Quantity:      1,
			TemplateId:    miuMiuItem,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP)

	compensator.DispatchCashItemUseRollbacks(s)

	assert.Equal(t, 1, len(exitCalls), "the opened shop should be closed exactly once")
	if len(exitCalls) == 1 {
		assert.Equal(t, testCharId, exitCalls[0], "EXIT must target the test character")
	}
}

// TestRemoteMerchantCompensationSkipsUncompletedShopOpen verifies that a shop
// open that never completed is NOT closed — the reverse-walk only inverts
// Completed mutations.
func TestRemoteMerchantCompensationSkipsUncompletedShopOpen(t *testing.T) {
	logger, _ := test.NewNullLogger()

	var exitCalls []uint32
	origExit := SetEmitNpcShopExitForTest(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, characterId uint32) error {
		exitCalls = append(exitCalls, characterId)
		return nil
	})
	t.Cleanup(func() { SetEmitNpcShopExitForTest(origExit) })

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(RemoteMerchant).
		SetInitiatedBy("remote-merchant-compensation-test").
		AddStep("open_npc_shop", Failed, OpenNpcShop, OpenNpcShopPayload{
			CharacterId:   88222,
			NpcTemplateId: 9090000,
		}).
		Build()
	assert.NoError(t, err)

	NewCompensator(logger, testTenantContext()).DispatchCashItemUseRollbacks(s)

	assert.Equal(t, 0, len(exitCalls), "a shop that never opened must not be closed")
}
```

Confirm `compmock`'s import path and mock type name against `point_reset_compensation_test.go`'s import block before running.

- [ ] **Step 3: Run test to verify it fails**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test ./saga/ -run TestDispatchCashItemUseRollbacks_RemoteMerchant -v
```

Expected: FAIL — no EXIT emitted (the rollback switch has no `OpenNpcShop` case).

- [ ] **Step 4: Route `RemoteMerchant` into the cash-item reverse-walk**

In `saga/compensator.go`, extend the saga-type arm at line 266:

```go
	// Cash-item-use reverse-walk (Task 10). A failed item_tag_use /
	// sealing_lock_use / incubator_use / remote_merchant must refund the
	// already-completed consume steps and undo any awarded result rather than
	// only compensating the failed step.
	if s.SagaType() == ItemTagUse || s.SagaType() == SealingLockUse || s.SagaType() == IncubatorUse || s.SagaType() == RemoteMerchant {
		return c.compensateCashItemUse(s, failedStep)
	}
```

Update the doc comments at `:109-114`, `:1361` and `:1411-1427` to name `remote_merchant` alongside the other three and to document the new `OpenNpcShop` inverse.

- [ ] **Step 5: Add the `OpenNpcShop` rollback case**

In `DispatchCashItemUseRollbacks` (line 1428), add a case to the switch:

```go
		case OpenNpcShop:
			// The inverse of "opened a shop" is "close it". Without this a
			// failed destroy step leaves the player standing in a shop they
			// did not pay for (task-221 FR-4.5).
			if payload, ok := step.Payload().(OpenNpcShopPayload); ok {
				if err := EmitNpcShopExit(c.l, c.ctx, s.TransactionId(), payload.CharacterId); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"character_id":   payload.CharacterId,
					}).Error("Reverse-walk: OpenNpcShop -> EXIT dispatch failed; continuing chain.")
				}
			}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test -race ./saga/ 2>&1 | tail -20
```

Expected: PASS, including the pre-existing compensator tests.

- [ ] **Step 7: Module verification**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go vet ./... && go test -race ./...
```

- [ ] **Step 8: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga
git commit -m "feat(task-221): compensate a failed remote-merchant saga with an npc-shop EXIT"
```

---

## Task 10: `atlas-channel` remote-merchant unlock registry

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/remotemerchant/registry.go`
- Create: `services/atlas-channel/atlas.com/channel/remotemerchant/registry_test.go`

**Interfaces:**
- Produces:
  - `remotemerchant.TTL = 30 * time.Second`
  - `remotemerchant.Entry{ItemId item.Id; Slot slot.Position; At time.Time}`
  - `remotemerchant.Expired{Tenant tenant.Model; CharacterId uint32; Entry Entry}`
  - `remotemerchant.GetRegistry() *Registry`
  - `(*Registry).Put(t tenant.Model, characterId uint32, e Entry)`
  - `(*Registry).Take(t tenant.Model, characterId uint32) (Entry, bool)` — returns and **removes** in one lock, so an `ENTERED` and an `ENTER_ERROR` racing on the same character unlock exactly once.
  - `(*Registry).ClearCharacter(t tenant.Model, characterId uint32)`
  - `(*Registry).Sweep(now time.Time) []Expired`
- Consumed by: Tasks 11, 12.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/remotemerchant/registry_test.go`:

```go
package remotemerchant

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mustTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return m
}

func TestTake_ReturnsAndRemovesOnce(t *testing.T) {
	ten := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearCharacter(ten, 1234) })

	r.Put(ten, 1234, Entry{ItemId: item.Id(5450000), Slot: slot.Position(3), At: time.Now()})

	e, ok := r.Take(ten, 1234)
	if !ok {
		t.Fatal("Take: want hit")
	}
	if e.ItemId != item.Id(5450000) || e.Slot != slot.Position(3) {
		t.Errorf("entry = %+v", e)
	}
	if _, ok := r.Take(ten, 1234); ok {
		t.Error("second Take: want miss — an entry must unlock exactly once")
	}
}

func TestTake_MissForUnknownCharacter(t *testing.T) {
	ten := mustTenant(t)
	if _, ok := GetRegistry().Take(ten, 999999); ok {
		t.Error("Take on an unregistered character: want miss (this is what keeps the NPC-talk path byte-identical)")
	}
}

func TestTake_TenantScoped(t *testing.T) {
	a, b := mustTenant(t), mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearCharacter(a, 1234) })

	r.Put(a, 1234, Entry{ItemId: item.Id(5450000), Slot: slot.Position(3), At: time.Now()})
	if _, ok := r.Take(b, 1234); ok {
		t.Error("Take from another tenant: want miss")
	}
}

func TestSweep_EvictsExpiredOnly(t *testing.T) {
	ten := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() {
		r.ClearCharacter(ten, 1)
		r.ClearCharacter(ten, 2)
	})

	now := time.Now()
	r.Put(ten, 1, Entry{ItemId: item.Id(5450000), Slot: slot.Position(1), At: now.Add(-2 * TTL)})
	r.Put(ten, 2, Entry{ItemId: item.Id(5450000), Slot: slot.Position(2), At: now})

	expired := r.Sweep(now)
	if len(expired) != 1 || expired[0].CharacterId != 1 {
		t.Fatalf("Sweep = %+v, want exactly character 1", expired)
	}
	if _, ok := r.Take(ten, 1); ok {
		t.Error("swept entry still present")
	}
	if _, ok := r.Take(ten, 2); !ok {
		t.Error("fresh entry was swept")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./remotemerchant/ -v
```

Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the registry**

Create `services/atlas-channel/atlas.com/channel/remotemerchant/registry.go`:

```go
// Package remotemerchant tracks which characters opened an NPC shop by using a
// classification-545 cash item (Miu Miu the Traveling Merchant) rather than by
// talking to the NPC.
//
// Why it exists: the client sets m_bExclRequestSent when it sends CASH_ITEM_USE
// and CShopDlg::SetShopDlg (@0x7529ad on v83) never clears it — decompiled in
// full during task-221's design phase (§1.2, OQ-2). So the server must send
// EnableActions. But the ordinary "talk to an NPC" shop path must stay
// byte-identical on the versions whose OPEN_NPC_SHOP matrix cells are already
// verified, so the unlock cannot be unconditional in the shop consumer. An
// entry here is the condition.
//
// This is presentation state only. Losing it (pod restart, dropped event) costs
// one EnableActions, never an item — the item's fate belongs entirely to the
// saga.
//
// Why in-process state is the whole view rather than a shard: a character's
// socket session lives on exactly one atlas-channel pod, so the pod that wrote
// the entry is the pod that owns the session that needs unlocking.
package remotemerchant

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TTL bounds how long a pending unlock waits for its status event. A dropped or
// lost event would otherwise leave the character locked until they reconnect;
// the sweep unlocks them instead. 30s is well above the ENTER→ENTERED round
// trip and well below any tolerable stuck-client window.
const TTL = 30 * time.Second

type Key struct {
	Tenant      tenant.Model
	CharacterId uint32
}

// Entry is the pending unlock for one remote-initiated shop open.
type Entry struct {
	ItemId item.Id
	Slot   slot.Position
	At     time.Time
}

// Expired is one swept entry, carrying enough context to unlock its session.
type Expired struct {
	Tenant      tenant.Model
	CharacterId uint32
	Entry       Entry
}

type Registry struct {
	mutex   sync.RWMutex
	pending map[Key]Entry
}

var (
	registry *Registry
	once     sync.Once
)

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{pending: make(map[Key]Entry)}
	})
	return registry
}

// Put records a pending unlock. Called before the saga is created so a very
// fast ENTERED cannot arrive before the registry write.
func (r *Registry) Put(t tenant.Model, characterId uint32, e Entry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pending[Key{Tenant: t, CharacterId: characterId}] = e
}

// Take returns and removes the pending unlock in one lock, so an ENTERED and an
// ENTER_ERROR racing on the same character unlock exactly once.
func (r *Registry) Take(t tenant.Model, characterId uint32) (Entry, bool) {
	k := Key{Tenant: t, CharacterId: characterId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	e, ok := r.pending[k]
	if ok {
		delete(r.pending, k)
	}
	return e, ok
}

// ClearCharacter drops the pending unlock for a character (session destroy).
// Without this the map leaks one entry per character ever seen by this pod.
func (r *Registry) ClearCharacter(t tenant.Model, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.pending, Key{Tenant: t, CharacterId: characterId})
}

// Sweep removes and returns every entry older than TTL. The caller sends
// EnableActions for each so a lost status event cannot leave a character
// permanently locked.
func (r *Registry) Sweep(now time.Time) []Expired {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var out []Expired
	for k, e := range r.pending {
		if now.Sub(e.At) >= TTL {
			out = append(out, Expired{Tenant: k.Tenant, CharacterId: k.CharacterId, Entry: e})
			delete(r.pending, k)
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./remotemerchant/ -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/remotemerchant
git commit -m "feat(task-221): add the remote-merchant pending-unlock registry"
```

---

## Task 11: Unlock the client when a remote-initiated shop opens or fails

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop/consumer.go:35-56` (register a third handler + start the sweep), `:59-86` (entered path), plus a new `handleEnterErrorStatusEvent`
- Modify: `services/atlas-channel/atlas.com/channel/socket/init.go:56-66`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop/consumer_test.go` (create)

**Interfaces:**
- Consumes: Task 10's registry; Task 3's `StatusEventTypeEnterError` / `StatusEventEnterErrorBody`.
- Produces: `unlockPendingRemoteMerchant(l logrus.FieldLogger, t tenant.Model, characterId uint32, unlock func())`. Behaviour contract: `EnableActions` is sent **iff** the registry had an entry for that character.

Ordering matters: announce the shop packet **first**, then `EnableActions`. `EnableActions` is the client's duplicate-request gate (`[[reference_exclrequest_unlock_contract]]`); unlocking before the dialog exists would let the client fire another request into a half-built state.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop/consumer_test.go`:

```go
package shop

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"atlas-channel/remotemerchant"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mustTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return m
}

func nullLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

func TestUnlockPending_HitUnlocksAndClears(t *testing.T) {
	ten := mustTenant(t)
	remotemerchant.GetRegistry().Put(ten, 1234, remotemerchant.Entry{
		ItemId: item.Id(5450000), Slot: slot.Position(3), At: time.Now(),
	})
	t.Cleanup(func() { remotemerchant.GetRegistry().ClearCharacter(ten, 1234) })

	var unlocked int
	unlockPendingRemoteMerchant(nullLogger(t), ten, 1234, func() { unlocked++ })

	if unlocked != 1 {
		t.Errorf("unlock calls = %d, want 1", unlocked)
	}
	if _, ok := remotemerchant.GetRegistry().Take(ten, 1234); ok {
		t.Error("registry entry survived the unlock")
	}
}

// TestUnlockPending_MissDoesNotUnlock protects the ordinary NPC-talk path:
// v61/72/79/83/84/87/95 OPEN_NPC_SHOP cells are verified and must stay
// byte-identical, so no unconditional EnableActions may be added here.
func TestUnlockPending_MissDoesNotUnlock(t *testing.T) {
	ten := mustTenant(t)

	var unlocked int
	unlockPendingRemoteMerchant(nullLogger(t), ten, 999999, func() { unlocked++ })

	if unlocked != 0 {
		t.Errorf("unlock calls = %d, want 0", unlocked)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/npc/shop/ -v
```

Expected: FAIL — `unlockPendingRemoteMerchant` undefined.

- [ ] **Step 3: Add the helper and rewrite the entered path**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop/consumer.go`, add the imports `"atlas-channel/remotemerchant"` and `statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"`, then:

```go
// unlockPendingRemoteMerchant sends EnableActions if and only if this character
// reached the shop by using a classification-545 cash item. The client sets
// m_bExclRequestSent when it sends CASH_ITEM_USE and CShopDlg::SetShopDlg never
// clears it (task-221 design §1.2 OQ-2), so the server must unlock — but only
// for that path. Unlocking the ordinary NPC-talk path would change the bytes on
// versions whose OPEN_NPC_SHOP cells are already verified.
func unlockPendingRemoteMerchant(l logrus.FieldLogger, t tenant.Model, characterId uint32, unlock func()) {
	e, ok := remotemerchant.GetRegistry().Take(t, characterId)
	if !ok {
		return
	}
	l.WithFields(logrus.Fields{
		"character_id": characterId,
		"item_id":      uint32(e.ItemId),
		"slot":         int16(e.Slot),
	}).Debug("Unlocking client after a remote-merchant shop open.")
	unlock()
}
```

Rewrite `handleEnteredStatusEvent` so the unlock runs on every exit path, after any shop announce:

```go
func handleEnteredStatusEvent(sc server.Model, wp writer.Producer) message.Handler[shops2.StatusEvent[shops2.StatusEventEnteredBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shops2.StatusEvent[shops2.StatusEventEnteredBody]) {
		if e.Type != shops2.StatusEventTypeEntered {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.CharacterId)
		if err != nil {
			return
		}

		// EnableActions is the client's duplicate-request gate, so it goes
		// AFTER the shop packet on the success path — but it must still fire if
		// the announce below bails out, or the player is locked for nothing.
		defer unlockPendingRemoteMerchant(l, t, e.CharacterId, func() {
			_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		})

		sms, err := skill.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get skills for character [%d].", s.CharacterId())
			return
		}

		nsm, err := shops.NewProcessor(l, ctx).GetShop(e.Body.NpcTemplateId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get shop for NPC [%d].", e.Body.NpcTemplateId)
			return
		}
		set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
		bp := writer.NPCShopBody(e.Body.NpcTemplateId, nsm.Commodities(), sms, set.Skill)
		_ = session.Announce(l)(ctx)(wp)(npcpkt.NPCShopWriter)(bp)(s)
	}
}
```

- [ ] **Step 4: Add the enter-error handler**

Still in `consumer.go`:

```go
// handleEnterErrorStatusEvent unlocks the client after a failed remote-merchant
// shop open. It deliberately writes NO packet: NPCShopOperation with no
// outstanding buy/sell/recharge request throws CDisconnectException in
// CShopDlg::OnPacket (@0x756da7), which is exactly why ENTER_ERROR is a
// separate status type from ERROR (task-221 design delta D5).
func handleEnterErrorStatusEvent(sc server.Model, wp writer.Producer) message.Handler[shops2.StatusEvent[shops2.StatusEventEnterErrorBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shops2.StatusEvent[shops2.StatusEventEnterErrorBody]) {
		if e.Type != shops2.StatusEventTypeEnterError {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.CharacterId)
		if err != nil {
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":    e.CharacterId,
			"npc_template_id": e.Body.NpcTemplateId,
			"reason":          e.Body.Reason,
		}).Warn("NPC shop enter failed.")

		unlockPendingRemoteMerchant(l, t, e.CharacterId, func() {
			_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		})
	}
}
```

Register it in `InitHandlers` alongside the other two, following the same `id, err = rf(...)` / `handles = append(...)` shape.

- [ ] **Step 5: Add the TTL sweep**

Still in the same package. Use `routine.Go` — a bare `go` statement fails `tools/goroutine-guard.sh`:

```go
// startRemoteMerchantSweep evicts pending unlocks whose status event never
// arrived and unlocks those clients, so a dropped event cannot leave a
// character permanently locked (task-221 design §2.3).
func startRemoteMerchantSweep(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer) {
	routine.Go(l, ctx, func(c context.Context) {
		ticker := time.NewTicker(remotemerchant.TTL)
		defer ticker.Stop()
		for {
			select {
			case <-c.Done():
				return
			case now := <-ticker.C:
				for _, ex := range remotemerchant.GetRegistry().Sweep(now) {
					if !ex.Tenant.Is(sc.Tenant()) {
						continue
					}
					s, err := session.NewProcessor(l, c).GetByCharacterId(sc.Channel())(ex.CharacterId)
					if err != nil {
						continue
					}
					l.WithFields(logrus.Fields{
						"character_id": ex.CharacterId,
						"item_id":      uint32(ex.Entry.ItemId),
					}).Warn("Remote-merchant shop open timed out with no status event; unlocking client.")
					_ = session.Announce(l)(c)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
				}
			}
		}
	})
}
```

Call it once from `InitHandlers` before returning `handles`. `InitHandlers` currently has no `ctx` in scope — thread the one its caller already holds by adding a `ctx context.Context` parameter to the curried chain, mirroring how the neighbouring channel consumers that need a context do it. Do not fabricate a `context.Background()`: the tenant must come from the real context.

- [ ] **Step 6: Clear on session destroy**

In `services/atlas-channel/atlas.com/channel/socket/init.go`, inside the `SetDestroyer` closure alongside the existing clears, and add the `"atlas-channel/remotemerchant"` import:

```go
							// Channel change and disconnect both destroy the
							// session; without this the pending-unlock map
							// leaks one entry per character ever seen by this
							// pod (task-221).
							remotemerchant.GetRegistry().ClearCharacter(t, s.CharacterId())
```

- [ ] **Step 7: Run tests and the goroutine guard**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./kafka/consumer/npc/shop/ ./remotemerchant/ -v
```

```bash
cd "$(git rev-parse --show-toplevel)" && ./tools/goroutine-guard.sh
```

Expected: tests PASS, guard exits 0.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop services/atlas-channel/atlas.com/channel/socket/init.go
git commit -m "feat(task-221): unlock the client only for remote-initiated npc shop opens"
```

---

## Task 12: The classification-545 handler arm

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_remote_merchant.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_remote_merchant_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (insert the branch after line 504, before the `ClassificationMegaphones` branch)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go` (add `newCashItemUseTestSessionForVersion`)

**Interfaces:**
- Consumes: Task 2's `cash.RestModel.Npc`; Task 5's `saga.OpenNpcShop`/`RemoteMerchant`/`OpenNpcShopPayload`; Task 10's registry.
- Produces:
  - `handleRemoteMerchantUse(l, ctx, wp)(s session.Model, t tenant.Model, itemId item.Id, source slot.Position, it CashSlotItemType)`
  - `remoteMerchantEnabled(t tenant.Model) bool`
  - `remoteMerchantCashSlotType(t tenant.Model) CashSlotItemType`
  - test seams `cashItemDataFunc`, `remoteMerchantSagaCreateFunc`

**No `Decode` call.** Design §1.2 established the sub-body is empty on every version that has an arm: the store case falls straight into the dispatcher's shared encode-and-send tail with no `Encode*` of its own (v83 `0xa0cda7`, v95 `0x9ee50a`).

**No ownership re-check** (design delta D1): `CharacterCashItemUseHandleFunc` already validated `cashItemInSlotFunc(...) == itemId` at `character_cash_item_use.go:54-58` before any arm runs.

- [ ] **Step 1: Add the version-parameterised session helper**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go`, add next to `newCashItemUseTestSession` — same body, parameterised region/major, and have the existing helper delegate to it with `("GMS", 83)`:

```go
// newCashItemUseTestSessionForVersion is newCashItemUseTestSession with the
// tenant version parameterised, for the remote-merchant version-gate tests.
func newCashItemUseTestSessionForVersion(t *testing.T, characterId uint32, region string, major uint16) (session.Model, context.Context, func()) {
	t.Helper()
	ten := mustTenant(t, region, major, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)

	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, characterId)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	return updated, ctx, func() { session.ClearRegistryForTenant(ten.Id()) }
}
```

- [ ] **Step 2: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_remote_merchant_test.go`. It reuses `installCashItemInSlotSeam`, `mustTenant`, `cashItemUsePrefix` and the helper above (same package):

```go
package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	"atlas-channel/data/cash"
	"atlas-channel/remotemerchant"
	"atlas-channel/saga"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// installCashItemDataSeam swaps the atlas-data cash-item read.
func installCashItemDataSeam(t *testing.T, m cash.RestModel, err error) func() {
	t.Helper()
	orig := cashItemDataFunc
	cashItemDataFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (cash.RestModel, error) {
		return m, err
	}
	return func() { cashItemDataFunc = orig }
}

// installRemoteMerchantSagaSeam records created sagas instead of producing.
func installRemoteMerchantSagaSeam(t *testing.T) (*[]saga.Saga, func()) {
	t.Helper()
	var got []saga.Saga
	orig := remoteMerchantSagaCreateFunc
	remoteMerchantSagaCreateFunc = func(_ logrus.FieldLogger, _ context.Context, s saga.Saga) error {
		got = append(got, s)
		return nil
	}
	return &got, func() { remoteMerchantSagaCreateFunc = orig }
}

func TestRemoteMerchantEnabled_MatchesTheIdbEvidenceMatrix(t *testing.T) {
	cases := []struct {
		region string
		major  uint16
		want   bool
	}{
		// Design §1.3. The three disabled GMS builds have no send arm for the
		// cash-slot type, so a request can only come from a crafted packet.
		{"GMS", 12, false},
		{"GMS", 48, false},
		{"GMS", 61, false},
		{"GMS", 72, true},
		{"GMS", 79, true},
		{"GMS", 83, true},
		{"GMS", 84, true},
		{"GMS", 87, true},
		{"GMS", 92, true},
		{"GMS", 95, true},
		// JMS maps classification 545 to cash-slot type 36, not 37/38, and this
		// task does not enable it (design §7.3).
		{"JMS", 185, false},
	}
	for _, c := range cases {
		ten := mustTenant(t, c.region, c.major, 1)
		if got := remoteMerchantEnabled(ten); got != c.want {
			t.Errorf("remoteMerchantEnabled(%s %d) = %v, want %v", c.region, c.major, got, c.want)
		}
	}
}

func TestRemoteMerchantCashSlotType_MatchesGetCashSlotItemType(t *testing.T) {
	if got := remoteMerchantCashSlotType(mustTenant(t, "GMS", 83, 1)); got != CashSlotItemType(37) {
		t.Errorf("GMS 83 = %d, want 37", got)
	}
	if got := remoteMerchantCashSlotType(mustTenant(t, "GMS", 95, 1)); got != CashSlotItemType(38) {
		t.Errorf("GMS 95 = %d, want 38", got)
	}
}

func TestHandleRemoteMerchantUse_HappyPathRegistersAndCreatesSaga(t *testing.T) {
	const itemId = uint32(5450000)
	const charId = uint32(555)
	const srcSlot = int16(3)

	restoreSlot := installCashItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installCashItemDataSeam(t, cash.RestModel{Id: itemId, Npc: 9090000}, nil)
	defer restoreData()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, charId)
	defer cleanup()

	r := request.NewReader(0, cashItemUsePrefix(srcSlot, itemId), 0, false)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, r, map[string]interface{}{})

	if len(*sagas) != 1 {
		t.Fatalf("sagas created = %d, want 1", len(*sagas))
	}
	sg := (*sagas)[0]
	if sg.SagaType != saga.RemoteMerchant {
		t.Errorf("SagaType = %q, want %q", sg.SagaType, saga.RemoteMerchant)
	}
	if len(sg.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(sg.Steps))
	}
	if sg.Steps[0].Action != saga.OpenNpcShop {
		t.Errorf("step 0 action = %q, want open_npc_shop", sg.Steps[0].Action)
	}
	op, ok := sg.Steps[0].Payload.(saga.OpenNpcShopPayload)
	if !ok || op.NpcTemplateId != 9090000 || op.CharacterId != charId {
		t.Errorf("step 0 payload = %+v", sg.Steps[0].Payload)
	}
	if sg.Steps[1].Action != saga.DestroyAssetFromSlot {
		t.Errorf("step 1 action = %q, want destroy_asset_from_slot", sg.Steps[1].Action)
	}
	dp, ok := sg.Steps[1].Payload.(saga.DestroyAssetFromSlotPayload)
	if !ok || dp.InventoryType != 5 || dp.Slot != srcSlot || dp.Quantity != 1 || dp.TemplateId != itemId {
		t.Errorf("step 1 payload = %+v", sg.Steps[1].Payload)
	}

	// The pending unlock must be registered BEFORE the saga is created, or a
	// fast ENTERED could arrive with nothing to match.
	ten := tenant.MustFromContext(ctx)
	if _, ok := remotemerchant.GetRegistry().Take(ten, charId); !ok {
		t.Error("no pending unlock registered")
	}
}

func TestHandleRemoteMerchantUse_DisabledVersionDoesNotCreateSaga(t *testing.T) {
	const itemId = uint32(5450000)
	restoreSlot := installCashItemInSlotSeam(t, 3, itemId)
	defer restoreSlot()
	restoreData := installCashItemDataSeam(t, cash.RestModel{Id: itemId, Npc: 9090000}, nil)
	defer restoreData()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	// gms_61: case 37 sits in SendConsumeCashItemUseRequest's default list
	// (@0x832af3) — no client emits this.
	s, ctx, cleanup := newCashItemUseTestSessionForVersion(t, 777, "GMS", 61)
	defer cleanup()

	r := request.NewReader(0, cashItemUsePrefix(3, itemId), 0, false)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, r, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 on a disabled version", len(*sagas))
	}
}

func TestHandleRemoteMerchantUse_TicketNeverConsumes(t *testing.T) {
	const itemId = uint32(5451000)
	restoreSlot := installCashItemInSlotSeam(t, 3, itemId)
	defer restoreSlot()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 888)
	defer cleanup()

	r := request.NewReader(0, cashItemUsePrefix(3, itemId), 0, false)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, r, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 — no client build emits the remote gachapon ticket", len(*sagas))
	}
}

func TestHandleRemoteMerchantUse_DataErrorDoesNotCreateSaga(t *testing.T) {
	const itemId = uint32(5450000)
	restoreSlot := installCashItemInSlotSeam(t, 3, itemId)
	defer restoreSlot()
	restoreData := installCashItemDataSeam(t, cash.RestModel{}, errors.New("boom"))
	defer restoreData()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 999)
	defer cleanup()

	r := request.NewReader(0, cashItemUsePrefix(3, itemId), 0, false)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, r, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 when atlas-data fails", len(*sagas))
	}
}

func TestHandleRemoteMerchantUse_ZeroNpcDoesNotCreateSaga(t *testing.T) {
	const itemId = uint32(5450000)
	restoreSlot := installCashItemInSlotSeam(t, 3, itemId)
	defer restoreSlot()
	restoreData := installCashItemDataSeam(t, cash.RestModel{Id: itemId, Npc: 0}, nil)
	defer restoreData()
	sagas, restoreSaga := installRemoteMerchantSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 1001)
	defer cleanup()

	r := request.NewReader(0, cashItemUsePrefix(3, itemId), 0, false)
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, r, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 when the item resolves to npc 0", len(*sagas))
	}
}
```

Two notes for the implementer: `wp` is `nil` in these tests because every path either creates a saga or logs — if a rejection path panics on a nil `writer.Producer`, pass whatever no-op producer the neighbouring `character_cash_item_use_test.go` cases already use. And the v61 test reuses `cashItemUsePrefix` because `cashsb.UpdateTimeFirst(t)` is false below v87; if the reader errors on that build, adjust the fixture bytes, never the assertion.

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run RemoteMerchant -v
```

Expected: FAIL — `remoteMerchantEnabled`, `remoteMerchantCashSlotType`, `cashItemDataFunc`, `remoteMerchantSagaCreateFunc` undefined.

- [ ] **Step 4: Write the arm**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_remote_merchant.go`. If `saga.Saga`/`saga.Step` field names differ from the literal below, copy them from `character_cash_item_use_point_reset.go:57-77`, which builds the same shape:

```go
package handler

import (
	cashData "atlas-channel/data/cash"
	"atlas-channel/remotemerchant"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// cashItemDataFunc is a test seam for the atlas-data cash-item read
// (package-var injection precedent: cashItemInSlotFunc in
// character_cash_item_use.go).
var cashItemDataFunc = func(l logrus.FieldLogger, ctx context.Context, itemId uint32) (cashData.RestModel, error) {
	return cashData.NewProcessor(l, ctx).GetById(itemId)
}

// remoteMerchantSagaCreateFunc is a test seam for saga creation.
var remoteMerchantSagaCreateFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

// remoteMerchantEnabled reports whether this tenant's client can send
// CASH_ITEM_USE for classification 545 at all.
//
// Derived from CWvsContext::SendConsumeCashItemUseRequest's jump table in every
// GMS IDB (task-221 design §1.3): gms_12 registers no CharacterCashItemUseHandle
// at all; gms_48 predates get_cashslot_item_type entirely; gms_61 lists case 37
// in the dispatcher's default arm (@0x832af3), so it computes the type and sends
// nothing. v72 (@0x907472) through v95 (@0x9ee50a) all have a real arm.
//
// JMS maps classification 545 to cash-slot type 36 rather than 37/38
// (get_cashslot_item_type @0x49a1ee) and this task seeds no JMS shops or
// templates, so it stays off — design §7.3 records the bounded follow-up.
func remoteMerchantEnabled(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(72)
}

// remoteMerchantCashSlotType returns the cash-slot type this tenant's client
// computes for a remote store, mirroring GetCashSlotItemType's 545 branch.
func remoteMerchantCashSlotType(t tenant.Model) CashSlotItemType {
	if t.IsRegion("GMS") && t.MajorAtLeast(95) {
		return CashSlotItemType(38)
	}
	return CashSlotItemType(37)
}

// handleRemoteMerchantUse implements classification 545 (remote merchant):
// Miu Miu the Traveling Merchant (5450000) opens NPC 9090000's shop from
// anywhere, and the item is consumed only once the shop is confirmed open.
//
// Dispatch is classification-first, not cash-slot-type-first, for the reason
// character_cash_item_use.go:503-507 already documents: the type byte collides
// (37 is also the wedding-ticket bucket, 59/60 are also triple-megaphone
// buckets). The type is validated here, never used to choose a path.
//
// There is no sub-body to decode. Every version's arm falls straight into the
// dispatcher's shared encode-and-send tail with no Encode* of its own — v83
// @0xa0cda7, v95 @0x9ee50a (design §1.2, OQ-1).
//
// Ownership was already re-validated for every arm in
// CharacterCashItemUseHandleFunc (character_cash_item_use.go:54-58); this arm
// deliberately does not repeat it.
func handleRemoteMerchantUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, t tenant.Model, itemId item.Id, source slot.Position, it CashSlotItemType) {
	return func(s session.Model, t tenant.Model, itemId item.Id, source slot.Position, it CashSlotItemType) {
		enableActions := func() {
			_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		}

		// 5451xxx is the Remote Gachapon Ticket. No audited client build emits
		// CASH_ITEM_USE for its cash-slot type — 59/60 sit in every build's
		// default arm (design §1.2, OQ-3) — so reaching this branch means a
		// crafted packet. Never consume.
		if uint32(itemId)/1000 == 5451 {
			l.Warnf("Character [%d] attempted remote gachapon ticket [%d]; no client build emits this op — ignoring without consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		if !remoteMerchantEnabled(t) {
			l.Warnf("Character [%d] attempted remote merchant item [%d] on unsupported version [region %s major %d]; ignoring without consuming.", s.CharacterId(), itemId, t.Region(), t.MajorVersion())
			enableActions()
			return
		}

		if expected := remoteMerchantCashSlotType(t); it != expected {
			l.Warnf("Character [%d] remote merchant item [%d] arrived with cash slot type [%d], expected [%d]. Impossible from a legit client. Rejecting.", s.CharacterId(), itemId, it, expected)
			enableActions()
			return
		}

		ci, err := cashItemDataFunc(l, ctx, uint32(itemId))
		if err != nil {
			l.WithError(err).Errorf("Character [%d] remote merchant item [%d]: unable to read cash item data.", s.CharacterId(), itemId)
			enableActions()
			return
		}
		if ci.Npc == 0 {
			l.Warnf("Character [%d] remote merchant item [%d] resolves to npc 0; no shop to open.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		f := s.Field()
		now := time.Now()
		transactionId := uuid.New()

		// Registry first: a very fast ENTERED must not arrive before the entry
		// that tells the shop consumer to unlock this client.
		remotemerchant.GetRegistry().Put(t, s.CharacterId(), remotemerchant.Entry{
			ItemId: itemId,
			Slot:   source,
			At:     now,
		})

		sg := saga.Saga{
			TransactionId: transactionId,
			SagaType:      saga.RemoteMerchant,
			InitiatedBy:   "CASH_ITEM_USE",
			Steps: []saga.Step{
				{
					StepId: "open_npc_shop",
					Status: saga.Pending,
					Action: saga.OpenNpcShop,
					Payload: saga.OpenNpcShopPayload{
						CharacterId:   s.CharacterId(),
						WorldId:       f.WorldId(),
						ChannelId:     f.ChannelId(),
						NpcTemplateId: ci.Npc,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					StepId: "consume_remote_merchant_item",
					Status: saga.Pending,
					Action: saga.DestroyAssetFromSlot,
					Payload: saga.DestroyAssetFromSlotPayload{
						CharacterId:   s.CharacterId(),
						InventoryType: 5, // cash
						Slot:          int16(source),
						Quantity:      1,
						ShowEffect:    false,
						TemplateId:    uint32(itemId),
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		}

		if err := remoteMerchantSagaCreateFunc(l, ctx, sg); err != nil {
			l.WithError(err).Errorf("Character [%d] remote merchant item [%d]: unable to create saga.", s.CharacterId(), itemId)
			remotemerchant.GetRegistry().ClearCharacter(t, s.CharacterId())
			enableActions()
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":    s.CharacterId(),
			"item_id":         uint32(itemId),
			"cash_slot_type":  uint32(it),
			"npc_template_id": ci.Npc,
			"transaction_id":  transactionId.String(),
		}).Info("Remote merchant shop open requested.")
	}
}
```

- [ ] **Step 5: Route into the arm**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`, immediately after `category := item.GetClassification(itemId)` (line 504) and **before** the megaphone branch:

```go
		// Classification-FIRST, same reason as the megaphone branch below: the
		// cash-slot type byte collides (37 is also the wedding-ticket bucket,
		// 59/60 are also triple-megaphone buckets — GetCashSlotItemType).
		if category == item.ClassificationRemoteMerchant {
			handleRemoteMerchantUse(l, ctx, wp)(s, t, itemId, source, it)
			return
		}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -v 2>&1 | tail -40
```

Expected: PASS, including every pre-existing cash-item-use test. The new branch sits ahead of the megaphone branch and must not shadow it — megaphones are a different classification, never 545.

- [ ] **Step 7: Module verification**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./... 2>&1 | tail -20
```

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler
git commit -m "feat(task-221): handle classification-545 remote merchant cash items"
```

---

## Task 13: Shop data for NPC 9090000

**Files:**
- Create: `docs/tasks/task-221-miumiu-travel-store/commodity-existence-sweep.md`
- Create: `deploy/seed/gms/{12,48,61,72,79,83,84,87,92,95}_1/npc-shops/shops/shop-9090000.json`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: ten seed files consumed by `atlas-npc-shops`' `ShopSubdomain` (`services/atlas-npc-shops/atlas.com/npc/shops/subdomain.go:28-29`, path `npc-shops/shops`, filename pattern `^shop-(\d+)\.json$`).

The 26 commodities (PRD FR-5.2), all with `discountRate: 0`, `levelLimit: 0`, `period: 0`, `tokenPrice: 0`, `tokenTemplateId: 0`:

| templateId | mesoPrice | | templateId | mesoPrice |
|---|---|---|---|---|
| 2010003 | 100 | | 2022191 | 1000 |
| 2061000 | 1 | | 2022189 | 1000 |
| 2060000 | 1 | | 2010004 | 310 |
| 2030000 | 400 | | 2010001 | 106 |
| 2022195 | 15000 | | 2010002 | 50 |
| 2020015 | 10200 | | 2010000 | 30 |
| 2020014 | 8100 | | 2002025 | 1500 |
| 2020013 | 5600 | | 2002024 | 1500 |
| 2020012 | 4500 | | 2002023 | 3800 |
| 2022190 | 3000 | | 5041000 | 1500000 |
| 2001002 | 4000 | | 2022000 | 1650 |
| 2001001 | 2300 | | 2022003 | 1100 |
| 2001000 | 3200 | | 2022192 | 600 |

- [ ] **Step 1: Run the per-version existence sweep (OQ-5)**

No row ships unverified. `atlas-data` serves per-tenant (therefore per-version) item data:

- `20xxxxx` ids → `GET {DATA}/data/consumables/{id}`
- `5041000` → `GET {DATA}/data/cash/items/{id}`

For each of the ten GMS versions, resolve that version's tenant id and query all 26 ids. Use the tenant header convention the repo already uses for `atlas-data` reads (`[[reference_query_atlas_data_skill_per_version]]`, `[[reference_atlas_data_wz_inspection]]`); if the exact header set is unclear, read `libs/atlas-rest`'s tenant decorator rather than guessing.

Record the result in `docs/tasks/task-221-miumiu-travel-store/commodity-existence-sweep.md`: a table with one row per templateId and one column per version holding `present` / `absent` / `not-ingested`, plus the exact command used and the date.

Rules for the outcome:
- `absent` for a version → **drop** that commodity from that version's file.
- `not-ingested` (that version's WZ set is not loaded in the environment queried) → do **not** guess. Record it, seed the full 26 rows for that version, and name the un-checked versions in the sweep doc's "unverified" section. An extra row for a nonexistent item is inert in `atlas-npc-shops`; a *missing* row is a visible content gap.

Candidates most likely to be absent pre-v83, per design §3.5: `5041000` (Teleport Rock), `2022189`/`2022190`/`2022191`/`2022195`, `2002023`-`2002025`.

- [ ] **Step 2: Write one file and validate its shape**

Create `deploy/seed/gms/83_1/npc-shops/shops/shop-9090000.json` first, using the envelope verified against `deploy/seed/gms/83_1/npc-shops/shops/shop-1001000.json`. Keys are alphabetically ordered within each object, matching the existing files:

```json
{
  "data": {
    "attributes": {
      "commodities": [
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 100,
          "period": 0,
          "templateId": 2010003,
          "tokenPrice": 0,
          "tokenTemplateId": 0
        }
      ],
      "npcId": 9090000,
      "recharger": true
    },
    "id": "9090000",
    "type": "npc-shop"
  }
}
```

Expand `commodities` to that version's full surviving list, in the table's order.

`"recharger": true` is what makes `atlas-npc-shops` append the Redis-backed rechargeable star/bullet listings (`shops/cache.go`, `data/consumable/processor.go:GetRechargeable`) — the item description promises star and bullet recharge, and the 2070xxx ids are appended, never seeded.

- [ ] **Step 3: Validate**

```bash
python3 -m json.tool deploy/seed/gms/83_1/npc-shops/shops/shop-9090000.json > /dev/null && echo OK
```

```bash
cd "$(git rev-parse --show-toplevel)" && go run ./tools/catalog-lint 2>&1 | tail -20
```

If `tools/catalog-lint` needs arguments, read `tools/catalog-lint/subdomains.go` and its entry point for the invocation. Expected: valid JSON, lint clean.

- [ ] **Step 4: Write the other nine**

Repeat for `12_1, 48_1, 61_1, 72_1, 79_1, 84_1, 87_1, 92_1, 95_1`, each with its own sweep-filtered commodity list.

Seeding the three versions where the item cannot be used (12, 48, 61) is deliberate: the shop belongs to NPC 9090000, not to the item, and a future NPC-talk route to the same merchant should find it. An unreachable shop row is inert.

- [ ] **Step 5: Validate all ten**

```bash
for f in deploy/seed/gms/*/npc-shops/shops/shop-9090000.json; do python3 -m json.tool "$f" > /dev/null || echo "BAD: $f"; done
```

```bash
ls deploy/seed/gms/*/npc-shops/shops/shop-9090000.json | wc -l
```

Expected: no `BAD:` lines, count `10`.

- [ ] **Step 6: Commit**

```bash
git add deploy/seed/gms/*/npc-shops/shops/shop-9090000.json docs/tasks/task-221-miumiu-travel-store/commodity-existence-sweep.md
git commit -m "feat(task-221): seed NPC 9090000's shop in all ten GMS versions"
```

---

## Task 14: Register `NPCShopHandle` on gms_87/92/95 and the gms_92 writers

Current state, re-counted in this worktree (matches the PRD's table):

| Template | `NPCShopHandle` | `NPCShop` writer | `NPCShopOperation` writer |
|---|---|---|---|
| gms_12 | ✗ (also no `CharacterCashItemUseHandle` — out of scope) | ✗ | ✗ |
| gms_48 | ✓ | ✗ | ✗ (Task 15) |
| gms_61/72/79/83/84 | ✓ | ✓ | ✓ |
| gms_87 | ✗ | ✓ | ✓ |
| gms_92 | ✗ | ✗ | ✗ |
| gms_95 | ✗ | ✓ | ✓ |

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`

**Interfaces:** none (data only). Opcodes come from `docs/packets/audits/STATUS.md` — serverbound `NPC_SHOP` at line 572, clientbound `OPEN_NPC_SHOP` at 381 and `CONFIRM_SHOP_TRANSACTION` at 383, read against the column header at line 22 (`v48 # | v48 | v61 # | v61 | …`).

| Template | Add | opCode |
|---|---|---|
| gms_87 | `NPCShopHandle` | `0x040` |
| gms_92 | `NPCShopHandle` | `0x043` |
| gms_92 | `NPCShop` writer | `0x164` |
| gms_92 | `NPCShopOperation` writer | `0x165` |
| gms_95 | `NPCShopHandle` | `0x042` |

- [ ] **Step 1: Add the handler entries**

Into each of the three templates' `handlers` array, **at its sorted `opCode` position** — not appended next to a semantically related entry. Copy gms_83's block verbatim (`template_gms_83_1.json:520-536`), changing only `opCode`:

```json
      {
        "opCode": "0x040",
        "validator": "LoggedInValidator",
        "handler": "NPCShopHandle",
        "fname": "CShopDlg::SetRet",
        "options": {
          "operations": {
            "BUY": 0,
            "SELL": 1,
            "RECHARGE": 2,
            "LEAVE": 3
          }
        },
        "services": [
          "channel"
        ]
      },
```

The non-empty `"validator"` is mandatory — a handler with a missing validator is silently dropped at load (`[[bug_socket_handler_missing_validator_silently_dropped]]`).

- [ ] **Step 2: Add the gms_92 writer entries**

Into `template_gms_92_1.json`'s `writers` array at their sorted positions, copying gms_83's `0x131`/`0x132` blocks (`template_gms_83_1.json:4425-4457`) including the full `NPCShopOperation` operations table, changing only `opCode`:

```json
      {
        "opCode": "0x164",
        "writer": "NPCShop",
        "fname": "CShopDlg::SetShopDlg",
        "services": [
          "channel"
        ]
      },
      {
        "opCode": "0x165",
        "writer": "NPCShopOperation",
        "fname": "CShopDlg::OnPacket",
        "options": {
          "operations": {
            "OK": 0,
            "OUT_OF_STOCK": 1,
            "NOT_ENOUGH_MONEY": 2,
            "INVENTORY_FULL": 3,
            "OUT_OF_STOCK_2": 5,
            "OUT_OF_STOCK_3": 9,
            "NOT_ENOUGH_MONEY_2": 10,
            "NEED_MORE_ITEMS": 13,
            "OVER_LEVEL_REQUIREMENT": 14,
            "UNDER_LEVEL_REQUIREMENT": 15,
            "TRADE_LIMIT": 16,
            "GENERIC_ERROR": 17,
            "GENERIC_ERROR_WITH_REASON": 17
          }
        },
        "services": [
          "channel"
        ]
      },
```

Before accepting gms_83's operations table for v92, cross-check it against the gms_87 and gms_95 templates' existing `NPCShopOperation` blocks — v92 sits between them, so if 87 and 95 both agree with 83, v92 inherits the same table with high confidence. If 87 and 95 **disagree** with each other, derive v92's table from the v92 IDB's `CShopDlg::OnPacket` `Decode1` switch instead of copying either. Operations tables are exactly the thing that drifts across generations (`[[bug_gms_61_72_79_interaction_operations_wrong]]`, `[[bug_operations_mode_tables_missing_v87_v95_jms]]`).

- [ ] **Step 3: Run the template guards**

```bash
cd "$(git rev-parse --show-toplevel)" && ./tools/template-opcode-order-guard.sh && ./tools/template-duplicate-binding-guard.sh && ./tools/template-movement-types-guard.sh
```

```bash
for v in 87 92 95; do python3 -m json.tool "services/atlas-configurations/seed-data/templates/template_gms_${v}_1.json" > /dev/null || echo "BAD: $v"; done
```

Expected: all three guards exit 0, no `BAD:` lines. An out-of-order insertion fails the first guard — move the entry, do not silence the guard.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_87_1.json services/atlas-configurations/seed-data/templates/template_gms_92_1.json services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(task-221): register NPCShopHandle on gms_87/92/95 and the gms_92 shop writers"
```

---

## Task 15: Register the gms_48 shop writers (OQ-4)

Design §1.2 resolved OQ-4 from the v48 IDB: `?OnPacket@CShopDlg@@SAXJAAVCInPacket@@@Z` @ `0x5b7a38` branches on `nType == 229` → allocate `CShopDlg` + `SetShopDlg`, and `nType == 230` → the transaction-result switch. So `OPEN_NPC_SHOP` = `0xE5`, `CONFIRM_SHOP_TRANSACTION` = `0xE6`. Both matrix cells are currently ⬜ (unknown), not ❌.

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`
- Create: `docs/tasks/task-221-miumiu-travel-store/gms48-shop-operations.md`

**Interfaces:** none (data only).

- [ ] **Step 1: Derive the v48 `NPCShopOperation` operations table from the IDB**

Do **not** copy gms_83's table. Open the v48 IDB — resolve the session from `idb_list` by binary **name** and pass it as the `database` parameter; `select_instance` is dead (`[[reference_packet_audit_select_instance_dead]]`) — and decompile `CShopDlg::OnPacket` @ `0x5b7a38`. Read the `nType == 230` arm's `Decode1` result switch arm by arm, mapping each numeric case to the StringPool notice it raises, and match those notices to the thirteen `NPCShopOperation` names.

Record the derivation in `docs/tasks/task-221-miumiu-travel-store/gms48-shop-operations.md`: address, case number, notice string id, resulting name, one row each. A table copied without this derivation reproduces `[[bug_gms_61_72_79_interaction_operations_wrong]]`.

If one of the thirteen names has no v48 case, omit it from the v48 table rather than inventing a value, and say so in the derivation doc.

- [ ] **Step 2: Add the writer entries**

Into `template_gms_48_1.json`'s `writers` array at their sorted `opCode` positions:

```json
      {
        "opCode": "0xE5",
        "writer": "NPCShop",
        "fname": "CShopDlg::SetShopDlg",
        "services": [
          "channel"
        ]
      },
      {
        "opCode": "0xE6",
        "writer": "NPCShopOperation",
        "fname": "CShopDlg::OnPacket",
        "options": {
          "operations": {
            "REPLACE_WITH_THE_TABLE_DERIVED_IN_STEP_1": 0
          }
        },
        "services": [
          "channel"
        ]
      },
```

Replace the `REPLACE_WITH_…` key with the derived table before committing — a placeholder left in a landed template is a silent config break.

Match the opcode-string casing the rest of `template_gms_48_1.json` uses (`0xE5` vs `0x0E5`). A leading-zero variant of an opcode already bound elsewhere is exactly the duplicate-binding bug `tools/template-duplicate-binding-guard.sh` exists to catch.

- [ ] **Step 3: Run the guards**

```bash
cd "$(git rev-parse --show-toplevel)" && ./tools/template-opcode-order-guard.sh && ./tools/template-duplicate-binding-guard.sh
```

```bash
python3 -m json.tool services/atlas-configurations/seed-data/templates/template_gms_48_1.json > /dev/null && echo OK
```

Expected: exit 0, `OK`.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_48_1.json docs/tasks/task-221-miumiu-travel-store/gms48-shop-operations.md
git commit -m "feat(task-221): register the gms_48 shop writers at IDB-derived 0xE5/0xE6"
```

---

## Task 16: Promote the packet coverage cells

A registration change with no byte fixture is not a verified cell (PRD FR-7.1).

**Cells to promote:**

| Op | Direction | Versions | Current |
|---|---|---|---|
| `NPC_SHOP` | serverbound | v87 (`0x040`), v92 (`0x043`), v95 (`0x042`) | ❌ |
| `OPEN_NPC_SHOP` | clientbound | v92 (`0x164`) | ❌ |
| `CONFIRM_SHOP_TRANSACTION` | clientbound | v92 (`0x165`) | ❌ |
| `OPEN_NPC_SHOP` | clientbound | v48 (`0xE5`) | ⬜ |
| `CONFIRM_SHOP_TRANSACTION` | clientbound | v48 (`0xE6`) | ⬜ |

No new serverbound codec is added — the cash-item sub-body is empty (design §1.2), so FR-7.2 has nothing to write.

**Files:** whatever `/verify-packet` touches per cell — a byte-fixture test under `libs/atlas-packet`, an evidence record, and the regenerated matrix. Three artifacts committed together per cell.

- [ ] **Step 1: Verify each cell**

Run the single-cell verify procedure once per row above, driving `docs/packets/audits/VERIFYING_A_PACKET.md`. Read `docs/packets/audits/STATUS.md` lines 381, 383 and 572 and take the canonical packet path from the `Packet` column before invoking — the names below are indicative:

```
/verify-packet npc/serverbound/NpcShop gms_v87
/verify-packet npc/serverbound/NpcShop gms_v92
/verify-packet npc/serverbound/NpcShop gms_v95
/verify-packet npc/clientbound/NpcShop gms_v92
/verify-packet npc/clientbound/NpcShopOperation gms_v92
/verify-packet npc/clientbound/NpcShop gms_v48
/verify-packet npc/clientbound/NpcShopOperation gms_v48
```

Two constraints on the v48 cells: they are ⬜, so the verify pass establishes the opcode as well as the layout. If v48's writer body layout differs from v83's, that is a codec-level version gate discovered during verification — add the gate with the `MajorAtLeast` idiom, do not skip the cell. And a cell that does not promote is a failure to report, never a prose claim (`[[bug_matrix_roundtrip_fixture_false_verify]]`).

- [ ] **Step 2: Check the matrix**

```bash
cd "$(git rev-parse --show-toplevel)" && packet-audit matrix --check && packet-audit fname-doc --check && packet-audit operations --check
```

Expected: exit 0. If `packet-audit` is not on PATH, build/run it from its source directory as `docs/packets/PROCESS.md` describes.

- [ ] **Step 3: Confirm the promotions landed**

```bash
grep -n "^| NPC_SHOP \|^| OPEN_NPC_SHOP \|^| CONFIRM_SHOP_TRANSACTION " docs/packets/audits/STATUS.md
```

Expected: the seven cells above now read ✅. Quote the actual row in the task notes — do not assert promotion without reading it back.

- [ ] **Step 4: Commit**

`/verify-packet` commits its own three artifacts per cell. If any remain unstaged:

```bash
git add libs/atlas-packet docs/packets
git commit -m "test(task-221): verify the npc-shop packet cells opened by this task"
```

---

## Task 17: Reconcile the live tenant socket configurations

A template-only change does nothing for an already-provisioned tenant (`[[bug_new_opcodes_not_in_live_tenant_config]]`). PRD FR-6.6.

**Files:**
- Create: `docs/tasks/task-221-miumiu-travel-store/live-config-reconciliation.md`

- [ ] **Step 1: Reconcile**

Follow `[[reference_reconcile_live_tenant_socket_to_template]]` for each tenant whose template changed in Tasks 14 and 15: gms_48, gms_87, gms_92, gms_95.

- [ ] **Step 2: Verify by reading back the live configuration**

Read the live tenant socket configuration and confirm the new bindings are present. Do **not** verify by asserting the template file changed — that is the exact failure the memory note records.

For each of the four tenants, capture the actual response showing:
- gms_87: `NPCShopHandle` at `0x040`
- gms_92: `NPCShopHandle` at `0x043`, `NPCShop` at `0x164`, `NPCShopOperation` at `0x165`
- gms_95: `NPCShopHandle` at `0x042`
- gms_48: `NPCShop` at `0xE5`, `NPCShopOperation` at `0xE6`

- [ ] **Step 3: Record the evidence**

Write `docs/tasks/task-221-miumiu-travel-store/live-config-reconciliation.md` with, per tenant: the tenant id, the commands run, the quoted read-back showing each new binding, and the date. Quote the values — a paraphrase is not evidence.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-221-miumiu-travel-store/live-config-reconciliation.md
git commit -m "docs(task-221): record live tenant socket-config reconciliation"
```

---

## Task 18: Full verification and code review

- [ ] **Step 1: Per-module Go verification**

Run in each changed module — `libs/atlas-saga`, `services/atlas-channel/atlas.com/channel`, `services/atlas-npc-shops/atlas.com/npc`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-data/atlas.com/data`:

```bash
go build ./... && go vet ./... && go test -race ./...
```

Expected: clean in all five. Report failures with their output — a skipped module is not a passing module.

The orchestrator additionally has `//go:build test`-tagged tests (`producer_testseam.go` and its users, including Task 9's new file), which the untagged run silently skips. Run it a second time:

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test -race ./...
```

Expected: clean, and the run must include `TestRemoteMerchantCompensationEmitsShopExit`.

- [ ] **Step 2: Repo-root guards**

```bash
cd "$(git rev-parse --show-toplevel)" && ./tools/redis-key-guard.sh && ./tools/goroutine-guard.sh && ./tools/template-opcode-order-guard.sh && ./tools/template-duplicate-binding-guard.sh && ./tools/template-movement-types-guard.sh && ./tools/skill-job-id-guard.sh && ./tools/buff-duration-guard.sh && ./tools/trade-contract-mirror-guard.sh && ./tools/npc-shop-contract-mirror-guard.sh && ./tools/service-registration-guard.sh
```

Expected: every script exits 0. `service-registration-guard.sh` is only strictly required if `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work` or `tools/db-bootstrap.sh` changed — this task touches none of them, so run it to confirm it stays a no-op.

- [ ] **Step 3: Lint**

```bash
cd "$(git rev-parse --show-toplevel)" && ./tools/lint.sh --check
```

If it fails only on formatting, run `./tools/lint.sh` (no flags) to rewrite in place, then re-run `--check`. If it false-fails on the atlas-ui half, load nvm first (`[[bug_lint_check_false_fails_without_nvm]]`); this task changes no TypeScript.

- [ ] **Step 4: Docker bake**

Mandatory for every service whose `go.mod` was touched, and cheap insurance for the rest. `go build` against `go.work` cannot catch a missing `COPY libs/...` in the shared root `Dockerfile`.

```bash
cd "$(git rev-parse --show-toplevel)" && docker buildx bake atlas-channel atlas-saga-orchestrator atlas-npc-shops atlas-data
```

Expected: all four targets build. This task adds no new shared lib, so no `Dockerfile`/`go.work` edit should be needed — if a bake fails on a missing `COPY`, that is the signal to add one.

- [ ] **Step 5: Packet audit checks**

```bash
cd "$(git rev-parse --show-toplevel)" && packet-audit matrix --check && packet-audit fname-doc --check && packet-audit operations --check
```

Expected: exit 0.

- [ ] **Step 6: Worktree hygiene**

```bash
git status --porcelain
```

```bash
git rev-parse --show-toplevel
```

```bash
git branch --show-current
```

Expected: clean tree; toplevel ends with `/.worktrees/task-221-miumiu-travel-store`; branch is `task-221-miumiu-travel-store`.

- [ ] **Step 7: Code review**

Run `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (Go changed; no atlas-ui TypeScript changed, so the frontend reviewer is not needed). Findings go to `docs/tasks/task-221-miumiu-travel-store/audit.md`.

Pin the reviewer subagents to Sonnet/Haiku (`[[feedback_review_workflows_use_cheaper_model]]`), and ensure each operates inside this worktree — never the main repo. Verify the tree is clean after the subagent runs.

- [ ] **Step 8: Resolve findings and re-verify**

Apply the review's findings using `superpowers:receiving-code-review` — verify each before implementing; a reviewer can be wrong. Re-run Steps 1–5 after any code change.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "chore(task-221): resolve code review findings"
```

Only open the PR after every command in Steps 1–5 has been run and observed to pass.

---

## Acceptance criteria trace

Every PRD §10 checkbox, mapped to the task that satisfies it.

| PRD acceptance criterion | Task(s) |
|---|---|
| 5450000 on gms_83 opens NPC 9090000's shop and removes one copy | 1, 2, 3, 5–13 |
| Buy/sell/recharge work on gms_83 **and** gms_87/92/95 | 14, 16 |
| `atlas-npc-shops` failure → item kept, client unlocked, reason logged | 4, 8, 11 |
| 5451000 runs a gachapon roll | **Removed** by design §0/§6 — no client build emits the op. Task 12 logs a distinct warn and unlocks; nothing is consumed. |
| `shop-9090000.json` in all ten dirs, `recharger: true`, version-filtered ids | 13 |
| `NPCShopHandle` on gms_87/92/95; gms_92 writers; sorted opcodes; non-empty validators | 14 |
| OQ-4 resolved for gms_48 | 15 (resolved as "register at `0xE5`/`0xE6`", not `n-a`) |
| Live tenant configs reconciled, verified by read-back | 17 |
| Every touched packet cell promoted; `matrix --check` / `fname-doc --check` exit 0 | 16, 18 |
| `go test -race`, `go vet`, `go build` clean in every changed module | 18 |
| `docker buildx bake atlas-channel atlas-saga-orchestrator atlas-npc-shops atlas-data` | 18 |
| All guard scripts clean | 18 |
| Code review run and findings resolved before the PR | 18 |
