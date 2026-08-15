# task-230 — Implementation Context

Companion to [`plan.md`](plan.md). Everything here was read from the working tree at plan time.
Nothing is asserted from memory.

---

## 1. Verified facts that de-risk the plan

### 1.1 All 17 target opcode slots are FREE

Checked every template's `socket.handlers` array for a collision at the design's target opcode:

| Template | `ScriptedItemHandle` | `NpcItemUseHandle` |
|---|---|---|
| `template_gms_61_1.json` | — | `0x66` FREE |
| `template_gms_72_1.json` | `0x4D` FREE | `0x6E` FREE |
| `template_gms_79_1.json` | `0x4C` FREE | `0x6D` FREE |
| `template_gms_83_1.json` | `0x4E` FREE | `0x6F` FREE |
| `template_gms_84_1.json` | `0x4E` FREE | `0x6F` FREE |
| `template_gms_87_1.json` | `0x51` FREE | `0x72` FREE |
| `template_gms_92_1.json` | `0x55` FREE | `0x7A` FREE |
| `template_gms_95_1.json` | `0x54` FREE | `0x7B` FREE |
| `template_jms_185_1.json` | `0x46` FREE | `0x6A` FREE |

No template edit displaces an existing binding.

### 1.2 Template handler entry shape

The array is at `socket.handlers` (140 entries in v83), and the key is `handler`, **not**
`implementation`:

```json
{
  "opCode": "0x53",
  "validator": "LoggedInValidator",
  "handler": "ShopScannerItemUseHandle",
  "fname": "CWvsContext::SendShopScannerItemUseRequest",
  "services": ["channel"]
}
```

`opCode` is a **hex string**, `validator` is mandatory
(`bug_socket_handler_missing_validator_silently_dropped`), and entries are sorted strictly
ascending by numeric opcode.

### 1.3 Registry entry shape

`docs/packets/registry/gms_v83.yaml:2245`:

```yaml
- op: SCRIPTED_ITEM
  direction: serverbound
  opcode: 78
  fname: CWvsContext::SendScriptRunItemRequest
  provenance: csv-import
```

`opcode` is **decimal**. Entries are sorted ascending by opcode within the file.

Registry state today: `SCRIPTED_ITEM` present in v83/v84/v87/v92/v95/jms; `NPC_ITEM_USE_REQUEST`
present in the same six. Both **absent** from `gms_v61.yaml`, `gms_v72.yaml`, `gms_v79.yaml` —
which is the design §4.2 gap.

### 1.4 Matrix rows today

```
STATUS.md:612  SCRIPTED_ITEM        … v48 ⬜ v61 ⬜ v72 ⬜ v79 ⬜ | v83 ❌ v84 ❌ v87 ❌ v92 ❌ v95 ❌ jms ❌
STATUS.md:653  NPC_ITEM_USE_REQUEST … v48 ⬜ v61 ⬜ v72 ⬜ v79 ⬜ | v83 ❌ v84 ❌ v87 ❌ v92 ❌ v95 ❌ jms ❌
```

There is **no `gms_v12` column** — the matrix runs v48…jms_v185. Design F-4 is confirmed by
direct inspection of the row.

---

## 2. Codec placement — settled

Design §4.1 deferred the package choice to plan time. **Settled: `libs/atlas-packet/inventory/serverbound/`.**

That directory already holds the sibling item-use routes:

- `item_use.go` — `ItemUse` (`SendStatChangeItemUseRequest`, `updateTime`+`slot`+`itemId`)
- `lottery_item_use.go` — `LotteryItemUse` (`SendLotteryItemUseRequest`, `slot`+`itemId`, **no**
  `updateTime`)
- `scroll_use.go`, `compartment_merge.go`, `compartment_sort.go`, `move.go`

`LotteryItemUse` is the closest structural twin to `NpcItemUse` (identical 2-field body) and
`ItemUse` to `ScriptedItem` (identical 3-field body). Both new codecs sit beside them.

**Naming consequence.** `qualifiedWriterName(pkg, name)` = TitleCase(pkg) + struct name, so the
matrix/evidence paths are:

- `inventory/serverbound/InventoryScriptedItem`
- `inventory/serverbound/InventoryNpcItemUse`

### 2.1 The `candidatesFromFName` linkage is mandatory

`docs/packets/audits/VERIFYING_A_PACKET.md:130-137`: *"To verify a NEW serverbound op you must add
its primary fname as a `candidatesFromFName` case."* The switch lives at
`tools/packet-audit/cmd/run.go` (the `SendLotteryItemUseRequest` case is at `run.go:2205`). Without
the case, every one of the 17 verify attempts fails to resolve a codec and no cell promotes.

---

## 3. The Kafka contract decisions the design left open

### 3.1 The start command rides the existing `COMMAND_TOPIC_NPC`

`services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go` today:

```go
const (
    EnvCommandTopic                 = "COMMAND_TOPIC_NPC"
    CommandTypeStartConversation    = "START_CONVERSATION"
    CommandTypeContinueConversation = "CONTINUE_CONVERSATION"
    CommandTypeEndConversation      = "END_CONVERSATION"
)

type Command[E any] struct {
    NpcId       uint32 `json:"npcId"`
    CharacterId uint32 `json:"characterId"`
    Type        string `json:"type"`
    Body        E      `json:"body"`
}
```

`Command[E]` carries **no `TransactionId`**. The plan adds one (`omitempty`, additive and
backward-compatible) plus one new type `START_ITEM_CONVERSATION`.

**The `239` path deliberately reuses the existing `START_CONVERSATION` type** rather than adding a
`START_NPC_CONVERSATION` command type. The saga action `StartNpcConversation` is still distinct
(design §6.3's argument about the orchestrator's per-action dispatch switch is about *actions*, not
*command types*), but the wire command is the one that already exists — now carrying a
transactionId.

This is the exact npc-shop precedent. `kafka/consumer/npcshop/consumer.go:44-47`:

> *"the ordinary NPC-talk path produces ENTER with `uuid.Nil`, so most events reaching this handler
> match no saga — the nil check and `AcceptEvent` both decline them."*

Same shape here: `TransactionId == uuid.Nil` ⇒ ordinary NPC talk, emit no status. Non-nil ⇒
saga-awaited, emit `STARTED`/`START_ERROR`.

### 3.2 The new status topic and its mirror

`atlas-npc-conversations` currently **produces no status topic** — it only consumes
`EVENT_TOPIC_SAGA_STATUS` (`kafka/message/saga/kafka.go:7`) for sagas a conversation initiates.
The awaited-step design needs the opposite direction.

Owner file: `services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go`.
Mirror: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go`
— a **new package**; the orchestrator's `kafka/message/` has no `npc/` today (it has `npcshop/`).

Two Go modules, no compiler linkage → CLAUDE.md's items 13/14/15 failure class exactly. The plan
adds `tools/npc-conversation-contract-mirror-guard.sh` modelled on
`tools/npc-shop-contract-mirror-guard.sh`.

Topic env registration (mirroring `EVENT_TOPIC_NPC_SHOP_STATUS`):

- `deploy/k8s/base/env-configmap.yaml:147`
- `deploy/k8s/overlays/pr/kustomization.yaml:308` — `…-PLACEHOLDER_ATLAS_ENV`
- `deploy/k8s/overlays/main/kustomization.yaml:184` — `…-main`

An unsuffixed name cross-talks between environments.

---

## 4. `atlas-npc-conversations` — the shapes to mirror

### 4.1 The `243` family is packaged like `quest`, shaped like `npc`

`conversation/quest/Model` carries a **dual** state machine (`startStateMachine` +
`endStateMachine *StateMachine`) because a quest has accept and complete dialogues.
`conversation/npc/Model` carries `startState string` + `states []conversation.StateModel` and
implements `FindState` directly — that is the `StateContainer` interface at
`conversation/model.go:14`.

An item has one entry point, so `item.Model` copies **`npc`'s** shape and **`quest`'s** packaging
(own table, own REST resource, own seeder subdomain, own `MigrateTable`).

### 4.2 Files in `conversation/quest/` to mirror

`administrator.go` (create/update/delete/deleteAll closures over `*gorm.DB`), `entity.go`
(GORM entity + `Make`/`ToEntity`/`MigrateTable`), `groups.go` (`InitSeedResource` →
`seeder.RegisterRoutes` with `seeder.AdaptSubdomain`), `model.go`, `processor.go`
(Interface+Impl, `NewProcessor(l, ctx, db)`), `provider.go` (`getByIdProvider` /
`getByQuestIdProvider` / `getAllPagedProvider`), `resource.go`, `rest.go`, `subdomain.go`.

### 4.3 REST route naming — the design's `/item-conversations` is not the house style

PRD §5 proposed `/item-conversations`. The actual quest routes are:

```
/quests/conversations
/quests/conversations/{conversationId}
```

with `Resource = "quest-conversations"` (`quest/rest.go:13`) as the JSON:API **type**. So the item
family gets routes `/items/conversations` + `Resource = "item-conversations"`. The design's
resource-type name survives; the path follows the house pattern.

Note the path parameter is `{conversationId}` (a **uuid**), not the item id — `ByItemIdProvider`
is an internal lookup, not a route.

### 4.4 Seeder registration is a three-place change

1. `item/subdomain.go` — `Name() "item.conversation"`, `Path() "npc-conversations/items"`,
   `Type() "item-conversation"`, `EntityIDPattern() ^item-(\d+)\.json$`.
2. `item/groups.go` — `seeder.RegisterRoutes(..., URLPrefix: "/items/conversations", ...)`.
3. `tools/catalog-lint/subdomains.go:12-35` — a **hand-maintained mirror** of every per-service
   Subdomain. Missing entry ⇒ catalog-lint failure. Existing line to copy:
   `{path: "npc-conversations/quests", typ: "quest-conversation", pattern: regexp.MustCompile(`^quest-(\d+)\.json$`)}`.

### 4.5 `ConversationContext` is Redis-persisted — adding a field is a 4-place change

`conversation/registry.go` stores it via `atlas.TenantRegistry[uint32, ConversationContext]`, so it
round-trips through JSON. Adding `originTransactionId` (design §6.5) touches:

- the struct + getter (`conversation/model.go:2223-2232`, ~2287)
- the builder + setter (`model.go:2300-2380`)
- `MarshalJSON` (`model_json.go:497-517`)
- `UnmarshalJSON` (`model_json.go:520-542`)

Design §6.5 explicitly warns against folding this into the existing `sagaIndex`: that index is
keyed by *sagas the conversation initiated*, the opposite direction. Keep it separate.

### 4.6 `StartQuest` is the template for `StartItem`

`conversation/processor.go:165-210`. The shape: guard on `GetPreviousContext` returning nil error
("another conversation exists"), build the context with `SetConversationType` + `SetSourceId`, add
context values, `SetContext`, then loop `ProcessState` until it returns `cont == false`.

`processor.go:104-108` is the redelivery hazard design §6.5 names: a second delivery of the same
start command hits "another conversation exists" and would emit `START_ERROR` for a saga that
already succeeded.

---

## 5. Saga wiring — the five touch points `OpenNpcShop` occupies

| File | Line | What |
|---|---|---|
| `libs/atlas-saga/model.go` | 192-193 | `OpenNpcShop Action = "open_npc_shop"` under `// NPC shop actions` |
| `libs/atlas-saga/model.go` | 46-49 | `RemoteMerchant Type = "remote_merchant"` |
| `libs/atlas-saga/payloads.go` | 504-514 | `OpenNpcShopPayload` + the "NOT self-completing" doc comment |
| `libs/atlas-saga/unmarshal.go` | 319 | decode arm |
| orchestrator `saga/model.go` | 285, 1333 | payload alias + unmarshal arm |
| orchestrator `saga/handler.go` | 178, 854-855, 1195-1222 | interface method, dispatch case, impl |
| orchestrator `saga/producer.go` | 371-383 | `NpcShopEnterCommandProvider` |
| orchestrator `saga/producer.go` | 386-398 | `NpcShopExitCommandProvider` + `EmitNpcShopExit` seam |
| orchestrator `saga/event_acceptance.go` | 112-113, 177, 424-425 | EventKinds, action→kinds map, kind→outcome map |
| orchestrator `saga/character_extractor.go` | 65-66 | `case OpenNpcShopPayload: return p.CharacterId` |
| orchestrator `saga/compensator.go` | 1508-1520 | reverse-walk `OpenNpcShop → EXIT` |
| orchestrator `kafka/consumer/npcshop/consumer.go` | whole file | status consumer: `AcceptEvent` + `StepCompleted` |
| channel `saga/model.go` | 54, 74 | payload alias + saga type re-export |

`handler.go:1195-1201`'s doc comment is the canonical statement of the non-self-completing
contract and should be echoed (not copied verbatim) on the new arms.

`EmitNpcShopExit` is indirected through a package var specifically so compensator tests can observe
it without a broker — the new compensation arm needs the same seam.

---

## 6. `atlas-channel` — what does not exist yet

### 6.1 The channel's consumable model carries only `spec`

`services/atlas-channel/atlas.com/channel/data/consumable/rest.go`:

```go
type RestModel struct {
    Id   uint32             `json:"-"`
    Spec map[SpecType]int32 `json:"spec"`
}
```

`Npc`, `Script`, and `RunOnPickup` are **not** on it, even though `atlas-data` serves them
(`services/atlas-data/atlas.com/data/consumable/rest.go:74-76`). Design §7.1's `cd.Npc == 0` guard
needs them added to `RestModel`, `Model` (with getters), and `Extract`.

PRD §7 pointed at `atlas-consumables` / `atlas-inventory` for this; the handler actually reads
through `atlas-channel/data/consumable`, so that is the one that must change.

### 6.2 Handler registration

`services/atlas-channel/atlas.com/channel/main.go:963-966`:

```go
handlerMap[merchantsb.HiredMerchantOperationHandle] = handler.HiredMerchantOperationHandleFunc
handlerMap[merchantsb.OwlActionHandle]              = handler.OwlActionHandleFunc
handlerMap[merchantsb.OwlWarpHandle]                = handler.OwlWarpHandleFunc
handlerMap[merchantsb.ShopScannerItemUseHandle]     = handler.ShopScannerItemUseHandleFunc
```

### 6.3 Handler structural models

- **Decode → classify → validate slot → act:**
  `socket/handler/shop_scanner_item_use.go` (~40 lines, the whole shape).
- **Saga construction + logging + `enableActions` closure:**
  `socket/handler/character_cash_item_use_remote_merchant.go`. Note its `enableActions := func() {
  _ = session.EnableActions(l)(ctx)(wp)(s) }` idiom and the `remoteMerchantSagaCreateFunc` package
  var test seam (`:20-22`).
- **Shop-vs-conversation probe:** `socket/handler/npc_start_conversation.go:31-42` —
  `shops.NewProcessor(l, ctx).GetShop(template)`; `err == nil` ⇒ shop, else conversation.

### 6.4 `remoteMerchantEnabled` and the v61 gain

`character_cash_item_use_remote_merchant.go:36-38`:

```go
func remoteMerchantEnabled(t tenant.Model) bool {
    return t.IsRegion("GMS") && t.MajorAtLeast(72)
}
```

v61 is excluded because 545 sits in that dispatcher's **default arm** for `CASH_ITEM_USE`. Design
§7.2 notes `NPC_ITEM_USE_REQUEST` is therefore the *only* remote-merchant route on v61 — a real
behavioural gain that must be play-tested, not assumed. The new handler must **not** reuse
`remoteMerchantEnabled`; that predicate is about the *other* route.

---

## 7. `atlas-data` — the `spec` node defect

`services/atlas-data/atlas.com/data/consumable/reader.go`:

- `:36` binds `i, err := cxml.ChildByName("info")`
- `:75` `m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))` ← wrong node for `0243`
- `:76` `m.RunOnPickup = i.GetBool("runOnPickup", false)` ← same defect
- `:151-153` `m.ConsumeOnPickup = s.GetBool("consumeOnPickup", false)` ← the already-fixed
  precedent, with a comment recording the identical lesson
- `:162` `m.Script = s.GetString("script", "")` ← correct, reads from `spec`

The `spec` node is optional; the reader already guards it with `err == nil && s != nil`. The fix
reads spec-first with an `info` fallback because `0239` genuinely authors under `info` (verified on
`02390001`) while all 23 `0243` items author under `spec`
([`item-inventory.md`](item-inventory.md)).

**Re-ingest is a deployment step, not a code step** (`bug_atlas_data_effects_ingested_not_reparsed`).
A parser fix with no re-ingest leaves every tenant at `npc = 0` and the feature dead — which is why
the handler's `Npc == 0` log line must name re-ingest as the likely cause.

---

## 8. Seed content

Layout: `deploy/seed/<region>/<version>/npc-conversations/{npc,quests}/`. Files are JSON:API
envelopes:

```json
{ "data": { "attributes": { "npcId": 1002000, "startState": "intro", "states": [ … ] } } }
```

Eight in-scope version dirs for `SCRIPTED_ITEM`: `gms/72_1`, `gms/79_1`, `gms/83_1`, `gms/84_1`,
`gms/87_1`, `gms/92_1`, `gms/95_1`, `jms/185_1`. (`gms/61_1` gets `NpcItemUseHandle` but no
scripted-item content — v61 has no `SendScriptRunItemRequest`.)

Reference items (design §8, both verified in [`item-inventory.md`](item-inventory.md)):

- `2430013` — Peng Peng Popsicle, script `item_2430013`, npc `9010000`
- `2430008` — Golden Compass, script `compassUse`, npc `2084002`

Distinct avatars, no warp, no item grant, no quest state — the three selection criteria.
`2430010` (`openTreasure`) is deliberately **not** seeded: it is the only `runOnPickup` item and
that is a pickup trigger this task does not implement.

Each version dir carries a `CATALOG_REVISION` file that must be bumped when its contents change.

---

## 9. Verification gates that apply to this branch

From `CLAUDE.md`, the ones this task actually trips:

| Gate | Why it fires |
|---|---|
| `go test -race ./...`, `go vet ./...`, `go build ./...` | every changed module |
| `docker buildx bake atlas-<svc>` | any service whose `go.mod` moved |
| `tools/lint.sh --check` | tree-wide |
| `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` | tree-wide |
| `tools/template-opcode-order-guard.sh` | 9 templates change |
| `tools/template-duplicate-binding-guard.sh` | 9 templates change |
| `tools/template-movement-types-guard.sh` | any template edit trips its scope |
| `tools/catalog-lint` | new seed subdomain |
| `go run ./tools/packet-audit matrix --check` etc. | registry + matrix change |
| new `tools/npc-conversation-contract-mirror-guard.sh` | the new two-copy contract |

`tools/lint.sh --check` false-fails without nvm (`bug_lint_check_false_fails_without_nvm`) and
contends on a cross-worktree golangci-lint lock — run it from this worktree with nvm active.

The matrix `toolSha` reads git HEAD (`bug_packet_matrix_toolsha_reads_git_head`), so the matrix
regeneration must be the **final** commit on the branch, not a mid-branch one.

---

## 10. Known traps carried into the plan

| Trap | Where it bites |
|---|---|
| Live tenants don't inherit seed-template edits (`bug_new_opcodes_not_in_live_tenant_config`) | Both handlers silently dropped at dispatch until the live socket config is PATCHed |
| Handler with a missing validator is silently dropped (`bug_socket_handler_missing_validator_silently_dropped`) | Every new template entry needs `LoggedInValidator` |
| A round-trip fixture is not a verification (`bug_matrix_roundtrip_fixture_false_verify`) | All 17 cells need real byte fixtures traced to decompile lines |
| A serverbound `❌` can mean an unverified shared codec (`bug_matrix_redx_unverified_shared_codec`) | Design §1 already established neither op has an existing decoder — but Step-0 still runs |
| `NpcItemUse` has **no** `updateTime` | The single most likely defect: pattern-matching against the `ScriptedItem` sibling misaligns every subsequent read |
| v87 spews movement `Code 254` at ~2k/min (`bug_v87_monster_move_decode_code_254_constant`) | A standing confound for any v87 play-test report — play-test on v83 + a legacy column |
| Kafka is at-least-once | The `originTransactionId` idempotency work (§4.5) is not optional |

---

## 11. Open items the plan does **not** close

- **Item `3994225`** (v95-only whitelist) stays out of scope per design D-3. The plan implements the
  *rejection* behaviour — logged at warn, naming the gap explicitly, client unlocked, nothing
  consumed — so the gap is loud rather than silent. Supporting it needs `setup/reader.go` spec
  parsing (it parses no `spec` node at all today) plus a second inventory type on the destroy step.
- **`spec/runOnPickup`** becomes *visible* for the first time via the reader fix, but the pickup
  trigger itself is not implemented. `2430010` is left unseeded so the distinction stays observable.
- **The other 21 `0243` scripts' original behaviour** is unverified — only `killarmush` and
  `removethorns` exist in the local Cosmic tree. The two reference conversations are authored fresh
  as pure dialogue; the WZ `script` value is recorded for traceability only and is not a lookup key.
