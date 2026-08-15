# Scripted Items — Design

Task: task-230
Status: Approved
Created: 2026-08-14
Inputs: [`prd.md`](prd.md), [`version-evidence.md`](version-evidence.md)

---

## 0. What changed from the PRD

Five findings from design-phase verification materially alter the PRD. Each is evidence-backed
below; the PRD sections they supersede are named.

| # | Finding | Supersedes |
|---|---|---|
| F-1 | `atlas-data` reads `npc` from the **`info`** node, but every `0243` item carries it under **`spec`**. `Npc` is `0` for every scripted item today. `runOnPickup` has the identical defect. | PRD §1 ("Atlas already ingests both fields"), §2 non-goal ("no change to atlas-data WZ parsing"), §7 |
| F-2 | `remote_merchant` orders `OpenNpcShop` **first** and consumes **after**; `OpenNpcShop` is deliberately non-self-completing. The PRD's "consume up-front, matching the `remote_merchant` shape" misreads the precedent. | PRD §4.5, FR-5.2, FR-5.3 |
| F-3 | v95 gates on `nItemID / 10000 == 243 **|| nItemID == 3994225**`. A flat range check rejects a legitimate v95 request. | PRD FR-3.1 |
| F-4 | `gms_v12` is **not a matrix column** — the matrix runs v48…jms_v185. FR-1.1 has no matrix action; it is an evidence record only. `SendScriptRunItemRequest` is absent from the v12 binary regardless. | PRD FR-1.1, §9 O-2 |
| F-5 | `NPC_ITEM_USE_REQUEST` is **not** a `spec/script` route. It gates on item classifications **545 and 239**, carries a **different 2-field body** (no `updateTime`), and spans **v61…jms_v185**. It is now in scope (O-1 decision: fold in). | PRD §2 non-goal, §9 O-1 |

Decisions taken at the design gate:

- **D-1** — Conversation-first, then consume. Requires a new status topic (§6.2).
- **D-2** — Fix `spec/npc` **and** `spec/runOnPickup` in the consumable reader, with an `info` fallback.
- **D-3** — Exclude item `3994225`. Documented gap, not a silent omission (§3.3).
- **D-4** — Fold `NPC_ITEM_USE_REQUEST` into task-230.

---

## 1. Client evidence

All ten IDBs were swept — this is a full sweep, not a spot-check. Method per
[`version-evidence.md`](version-evidence.md): resolve the function, decompile, read the
`COutPacket::COutPacket` constructor argument and the `Encode*` sequence. Where the function was
unnamed, it was located by scanning for the `cmp reg, 0F3h` / `cmp reg, 221h` classification gate
and then named in the IDB.

### 1.1 `SCRIPTED_ITEM` — `CWvsContext::SendScriptRunItemRequest`

| Template | Address | Opcode | Registry | Extra gate |
|---|---|---|---|---|
| `gms_v12` | **absent** | — | — | — |
| `gms_v48` | **absent** | — | — | — |
| `gms_v61` | **absent** | — | — | — |
| `gms_v72` | `0x9044d8` | `0x04D` (77) | **missing** | — |
| `gms_v79` | `0x955840` | `0x04C` (76) | **missing** | — |
| `gms_v83` | `0xa09b26` | `0x04E` (78) | ✅ matches | `IsAbleToConsume` |
| `gms_v84` | `0xa53f08` | `0x04E` (78) | ✅ matches | `IsAbleToConsume` |
| `gms_v87` | `0xa9f3d2` | `0x051` (81) | ✅ matches | `IsAbleToConsume` |
| `gms_v92` | `0x9b3da0` | `0x055` (85) | ✅ matches | `IsAbleToConsume` |
| `gms_v95` | `0x9de7a0` | `0x054` (84) | ✅ matches | `IsAbleToConsume`; **`|| nItemID == 3994225`** |
| `jms_v185` | `0xaee7ce` | `0x046` (70) | ✅ matches | `IsAbleToConsume` |

Note `0xa9f3d2` (v87 `SCRIPTED_ITEM`) and `0xaa5a85` (v87 `NPC_ITEM_USE_REQUEST`, §1.2) are
adjacent and easily transposed; §1.3 is the authoritative list.

Body, **byte-identical on all eight versions that have it**:

```
Encode4(get_update_time())   // uint32 update time
Encode2(nPOS)                // int16  source inventory slot
Encode4(nItemID)             // int32  item template id
```

`gms_v84` was unnamed; located at `0xa53f08` via its `cmp eax, 0F3h` gate and renamed to
`CWvsContext__SendScriptRunItemRequest` in the IDB.

**Absence evidence.** `gms_v12` contains exactly one `cmp ..., 0F3h` in the whole image
(`0x4dba45`, a `cmp al, 0F3h` byte compare inside an unrelated 0x641-byte function) — not a
32-bit `itemId / 10000` test. Its `Send*ItemUseRequest` family stops at `SendPetFoodItemUseRequest`.
`gms_v48` and `gms_v61` are covered by the dense-naming argument already recorded in
[`version-evidence.md`](version-evidence.md). Since v48 and v61 both lack the function, v12 —
which predates both — cannot have it.

### 1.2 `NPC_ITEM_USE_REQUEST` — `CWvsContext::SendSelectNpcItemUseRequest`

| Template | Address | Opcode | Registry | Matrix now |
|---|---|---|---|---|
| `gms_v12` | **absent** | — | — | not a column |
| `gms_v48` | **absent** | — | — | ⬜ (correct) |
| `gms_v61` | `0x83778d` | **`0x066` (102)** | **missing** | ⬜ **wrong** |
| `gms_v72` | `0x90a5ac` | **`0x06E` (110)** | **missing** | ⬜ **wrong** |
| `gms_v79` | `0x95b96c` | **`0x06D` (109)** | **missing** | ⬜ **wrong** |
| `gms_v83` | `0xa10075` | `0x06F` (111) | ✅ matches | ❌ |
| `gms_v84` | `0xa5a4b2` | `0x06F` (111) | ✅ matches | ❌ |
| `gms_v87` | `0xaa5a85` | `0x072` (114) | ✅ matches | ❌ |
| `gms_v92` | `0x9aff40` | `0x07A` (122) | ✅ matches | ❌ |
| `gms_v95` | `0x9da430` | `0x07B` (123) | ✅ matches | ❌ |
| `jms_v185` | `0xaf43ee` | `0x06A` (106) | ✅ matches | ❌ |

Body, **byte-identical on all nine versions that have it** — note there is **no `updateTime`**:

```
Encode2(nPOS)                // int16  source inventory slot
Encode4(nItemID)             // int32  item template id
```

Client gate on every version:

```
(nItemID / 10000 == 545 || nItemID / 10000 == 239) && CanSendExclRequest(200, 0)
```

plus two refusal arms that emit a chat message and send nothing:

- field flag bit 18 (`0x40000`) set → `SP_276_YOU_CANT_USE_IT_HERE_IN_THIS_MAP` (v83 naming; the
  string id shifts per version: 275 on v72/v79, 279 on v84, 289 on v61/v92, `0x11E` on v87,
  `0x10D` on jms).
- a `CUniqueModeless` dialog already open → `SP_150_NOT_AVAILABLE` (150 on most; 152 on v92,
  `0x88` on jms).

`gms_v84` was unnamed; located at `0xa5a4b2` via its `cmp eax, 221h` gate and renamed to
`CWvsContext__SendSelectNpcItemUseRequest`. `gms_v48` and `gms_v12` contain **no** `cmp ,545` and
no `cmp ,239` anywhere in their images — absence confirmed by instruction scan, not by missing
symbol.

`gms_v12` was opened for this pass (`GMS_v12_1_DEVM.exe.i64`) specifically to discharge FR-1.1.

### 1.3 Authoritative address list

Use this, not the inline tables, when pinning evidence:

```
SCRIPTED_ITEM              NPC_ITEM_USE_REQUEST
gms_v61  —                 gms_v61  0x83778d  0x066
gms_v72  0x9044d8  0x04D   gms_v72  0x90a5ac  0x06E
gms_v79  0x955840  0x04C   gms_v79  0x95b96c  0x06D
gms_v83  0xa09b26  0x04E   gms_v83  0xa10075  0x06F
gms_v84  0xa53f08  0x04E   gms_v84  0xa5a4b2  0x06F
gms_v87  0xa9f3d2  0x051   gms_v87  0xaa5a85  0x072
gms_v92  0x9b3da0  0x055   gms_v92  0x9aff40  0x07A
gms_v95  0x9de7a0  0x054   gms_v95  0x9da430  0x07B
jms_v185 0xaee7ce  0x046   jms_v185 0xaf43ee  0x06A
```

---

## 2. The two features are not one feature

The PRD treated `spec/script` and `spec/npc` as a single feature. The client disagrees, and the WZ
data confirms the split:

| | `SCRIPTED_ITEM` | `NPC_ITEM_USE_REQUEST` |
|---|---|---|
| Item families | `243xxxx` | `239xxxx`, `545xxxx` |
| Lookup key | the item's own **script** | the item's **npc** |
| WZ node holding `npc` | `spec/npc` (**not parsed today**) | `info/npc` (parsed correctly today) |
| Body | 3 fields, leads with `updateTime` | 2 fields, no `updateTime` |
| Excl-request window | 500 ms | 200 ms |
| Version span | v72…jms_v185 | v61…jms_v185 |
| What it opens | a conversation authored **for the item** | the NPC's own shop or conversation |

`0239` items are remote-NPC summons — verified names "Athena Pierce's Marble" (*"Use this to
communicate with her regardless of the place"*) and "Traveling Tommy's Ticket" (*"Summons Tommy the
traveling merchant"*). `0545` is `ClassificationRemoteMerchant`, already modelled in
`libs/atlas-constants/item/constants.go:106` and already served on the `CASH_ITEM_USE` route by
task-221.

**Consequence for the architecture:** only the `243` family needs the new item-keyed conversation
store. `239` resolves to an ordinary NPC and reuses the **existing** `conversation/npc` family;
`545` reuses the **existing** `OpenNpcShop` saga action. This is a large reduction against the
PRD's implied scope.

---

## 3. Data layer

### 3.1 The `spec` node defect (F-1)

`services/atlas-data/atlas.com/data/consumable/reader.go` binds `i` to the **`info`** child
(`reader.go:36`) and then reads:

- `reader.go:75` — `m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))`
- `reader.go:76` — `m.RunOnPickup = i.GetBool("runOnPickup", false)`

`script` is read correctly from the `spec` node (`reader.go:162`, `s.GetString("script", "")`).

Verified against the `0243.img.xml` corpus: **zero of the 23 items have `info/npc`**; all 23 have
`spec/npc`. Item `02430010` (`openTreasure`) additionally has `spec/runOnPickup = 1`. Meanwhile
`02390001` **does** carry `info/npc`, so the existing read is correct for that family.

This is the same defect class as the `consumeOnPickup` bug already fixed on `reader.go:151-153`,
whose comment records the identical lesson.

**Fix (D-2):** read both fields from `spec` first, falling back to `info`:

```
npcId := s.GetIntegerWithDefault("npc", i.GetIntegerWithDefault("npc", 0))
runOnPickup := s.GetBool("runOnPickup", i.GetBool("runOnPickup", false))
```

The fallback is not defensive padding — `0239` genuinely authors under `info` and `0243` genuinely
authors under `spec`, and both must resolve. The `spec` node is optional, so the read must tolerate
its absence (as `reader.go` already does via the `err == nil && s != nil` guard).

No REST shape change: `Npc`, `Script`, and `RunOnPickup` are already on `RestModel`
(`consumable/rest.go:74-76`) and already mirrored by `atlas-consumables`, `atlas-inventory`, and
`atlas-npc-shops`. Only the values change — from `0` to correct.

**Re-ingest is required.** WZ data is ingested, not re-parsed on read
(`bug_atlas_data_effects_ingested_not_reparsed`). A parser fix with no re-ingest leaves every
tenant with `npc = 0` and the feature dead. This is a deployment step, not a code step, and is
called out in §10.

### 3.2 Item classifications

`libs/atlas-constants/item/constants.go` has no entry for either family. Add:

```go
ClassificationConsumableRemoteNpc    = Classification(239)
ClassificationConsumableScriptedItem = Classification(243)
```

placed in ascending order among the existing `Consumable*` block (between `ClassificationConsumableMonsterCard`
= 238 and the 3xx block). DOM-21 requires the handlers use `item.GetClassification` against these
rather than open-coding `/10000`.

### 3.3 Item `3994225` — the documented gap (D-3)

v95 alone whitelists `nItemID == 3994225`. Verified from WZ: it is **"Evolving Ring Upgrade
Potion"**, an **Install/Setup** item (`3xxxxxx`) carrying `spec/script = consume_3994255` and
`spec/npc = 9000021`.

It is out of scope because it costs three things this task does not otherwise need: spec parsing in
`services/atlas-data/atlas.com/data/setup/reader.go` (which today parses **no** `spec` node at all
and exposes neither `script` nor `npc` on its `RestModel`), a second inventory type on the destroy
step, and a version-gated validation rule.

**Required behaviour, not silence:** a v95 request for `3994225` must be logged at warn with the
same fields as any other rejection, must leave the client unlocked, and must not consume. The
rejection log line must name the gap explicitly so a play-test report is self-explaining. This is
recorded here and in the version support doc; it is a bounded follow-up, not an unknown.

---

## 4. Packet layer

### 4.1 Two codecs, no version gates

Both bodies are byte-identical across every version that has the op (§1), so **no `MajorAtLeast`
gating is required or permitted** — a gate with no divergence to express is noise. FR-2.2 is
satisfied vacuously, and the design deliberately records *why* rather than leaving its absence to
look like an oversight.

Structural model: `libs/atlas-packet/merchant/serverbound/shop_scanner_item_use.go` — immutable
struct, private fields, `New…` constructor, getters, `Operation()`, `String()`, and **both**
`Encode` and `Decode`. Carry the `// packet-audit:fname` marker.

`libs/atlas-packet/item/serverbound/scripted_item.go`:

```go
const ScriptedItemHandle = "ScriptedItemHandle"

// packet-audit:fname CWvsContext::SendScriptRunItemRequest
type ScriptedItem struct {
    updateTime uint32
    source     int16
    itemId     uint32
}
```

Decode order: `updateTime` (uint32) → `source` (int16) → `itemId` (uint32).

`libs/atlas-packet/item/serverbound/npc_item_use.go`:

```go
const NpcItemUseHandle = "NpcItemUseHandle"

// packet-audit:fname CWvsContext::SendSelectNpcItemUseRequest
type NpcItemUse struct {
    source int16
    itemId uint32
}
```

Decode order: `source` (int16) → `itemId` (uint32). **No `updateTime`** — this is the field most
likely to be added by pattern-matching against the sibling codec, and doing so misaligns every
subsequent read.

Exact package placement (`item/serverbound` vs an existing package) is settled at plan time against
the current `libs/atlas-packet` tree; the constraint is that both codecs live together, since they
are sibling item-use routes.

### 4.2 Registry, templates, matrix

**Registry additions** (`docs/packets/registry/`):

| File | Op | Opcode |
|---|---|---|
| `gms_v72.yaml` | `SCRIPTED_ITEM` | `0x04D` |
| `gms_v79.yaml` | `SCRIPTED_ITEM` | `0x04C` |
| `gms_v61.yaml` | `NPC_ITEM_USE_REQUEST` | `0x066` |
| `gms_v72.yaml` | `NPC_ITEM_USE_REQUEST` | `0x06E` |
| `gms_v79.yaml` | `NPC_ITEM_USE_REQUEST` | `0x06D` |

**Template bindings** (`services/atlas-configurations/seed-data/templates/`):

| Template | `ScriptedItemHandle` | `NpcItemUseHandle` |
|---|---|---|
| `template_gms_12_1.json` | — | — |
| `template_gms_48_1.json` | — | — |
| `template_gms_61_1.json` | — | ✔ `0x066` |
| `template_gms_72_1.json` | ✔ `0x04D` | ✔ `0x06E` |
| `template_gms_79_1.json` | ✔ `0x04C` | ✔ `0x06D` |
| `template_gms_83_1.json` | ✔ `0x04E` | ✔ `0x06F` |
| `template_gms_84_1.json` | ✔ `0x04E` | ✔ `0x06F` |
| `template_gms_87_1.json` | ✔ `0x051` | ✔ `0x072` |
| `template_gms_92_1.json` | ✔ `0x055` | ✔ `0x07A` |
| `template_gms_95_1.json` | ✔ `0x054` | ✔ `0x07B` |
| `template_jms_185_1.json` | ✔ `0x046` | ✔ `0x06A` |

Every entry goes at its **sorted position** in the `handlers` array, never appended beside a
semantically-related entry — `tools/template-opcode-order-guard.sh` enforces strictly ascending
`opCode`. `tools/template-duplicate-binding-guard.sh` must also pass; note v83 and v84 bind
`NpcItemUseHandle` to the same numeric opcode in *different* templates, which is legitimate and not
what that guard bans.

Live tenants do not inherit seed-template edits. Per
`bug_new_opcodes_not_in_live_tenant_config`, any tenant already provisioned needs the new handler
entries PATCHed into its live socket configuration, or the opcode is silently dropped at dispatch.
Both handlers also need a validator, or they are dropped silently
(`bug_socket_handler_missing_validator_silently_dropped`).

**Matrix corrections.** `SCRIPTED_ITEM` v72/v79 and `NPC_ITEM_USE_REQUEST` v61/v72/v79 are currently
`⬜` (= n-a) and are wrong. They promote to verified. `v48` stays `⬜` for both, and `v12` is not a
column at all — FR-1.1 is discharged by the evidence in §1.1/§1.2 recorded in the version support
docs, with no matrix edit.

Every promoted cell goes through the single-cell verify procedure
([`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md)) with a
real byte fixture. A round-trip Encode→Decode fixture is **not** a verification
(`bug_matrix_roundtrip_fixture_false_verify`).

---

## 5. Conversation dispatch

### 5.1 New `conversation/item/` family — `243` only

Mirrors `conversation/quest/` file-for-file: `administrator.go`, `entity.go`, `model.go`,
`processor.go`, `provider.go`, `resource.go`, `rest.go`, `subdomain.go`, plus `mock/`.

The `243` family needs a **single** state machine, not the quest family's dual start/end pair — so
`Model` follows the simpler `conversation/npc` shape (`startState` + `[]conversation.StateModel`)
while the *packaging* follows `quest` (own table, own resource, own seeder subdomain).

```go
type Model struct {
    id         uuid.UUID
    itemId     uint32
    npcId      uint32   // avatar the dialogue renders with
    scriptName string   // WZ spec/script — authoring traceability only
    startState string
    states     []conversation.StateModel
    createdAt  time.Time
    updatedAt  time.Time
}
```

`FindState` implements the existing `conversation.StateContainer` interface, so the whole
`ProcessState` / `Continue` / `End` machinery is reused untouched. **No new `StateType` is
introduced** and `conversation/model.go`'s state vocabulary is not extended.

Entity (`item_conversations` table), mirroring `quest/entity.go`:

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `tenant_id` | uuid | composite index priority 1 |
| `item_id` | uint32 | composite index priority 2; unique with `tenant_id` |
| `npc_id` | uint32 | plain index |
| `script_name` | text | traceability |
| `data` | jsonb | serialised `RestModel`, exactly as `quest` stores it |
| `created_at` / `updated_at` / `deleted_at` | | |

Additive migration; `MigrateTable` registered alongside `quest.MigrateTable`
(`main.go:60`). REST resource name `item-conversations`, symmetric with `conversations` and
`quest-conversations` (`conversation/*/rest.go` `Resource` consts). Seeder subdomain
`item.conversation`, path `npc-conversations/items`, entity-id pattern `^item-(\d+)\.json$`.

**`ConversationType`.** `conversation/model.go:2213-2218` defines `npc` and `quest`. Add
`ItemConversationType = "item"`, and start the context with `SetConversationType(ItemConversationType)`
+ `SetSourceId(itemId)`, exactly as `StartQuest` does for quests. `SetNpcId(npcId)` still carries the
avatar so every outbound dialogue command renders the right face.

New processor method on `conversation.Processor`:

```go
StartItem(f field.Model, itemId uint32, npcId uint32, characterId uint32,
          accountId uint32, stateMachine StateContainer) error
```

structurally identical to `StartQuest` (`processor.go:165`), adding `itemId` and `scriptName` as
context values for use in conditions.

### 5.2 `239` reuses what exists

A `239` item resolves its NPC from `info/npc` and then takes **exactly the dispatch
`npc_start_conversation.go:31-42` already performs**: probe `shops.GetShop(template)`; if a shop
exists, open it; otherwise start the NPC conversation. No new store, no new family, no new state
machine. The only new thing is the entry point.

`545` resolves its NPC from cash item data and always takes the shop path — the same
`OpenNpcShop` action task-221 already built.

---

## 6. Saga design

### 6.1 Ordering (D-1)

`remote_merchant` (`character_cash_item_use_remote_merchant.go:118-150`) orders **`OpenNpcShop`
first, `DestroyAssetFromSlot` second**, and `OpenNpcShopPayload`'s own doc comment states why: the
step is *deliberately not self-completing* so "a following destroy step consume[s] the cash item
only once the shop actually opened."

Scripted items adopt the same shape:

```
step 1  start_item_conversation      (awaits STARTED / START_ERROR)
step 2  destroy_asset_from_slot      (Use inventory, qty 1)
```

This makes the two dominant failure modes cost nothing:

| Failure | Result |
|---|---|
| No conversation authored for the item (FR-4.4) | `START_ERROR` → step 2 never runs → item intact |
| Character already in a conversation | `START_ERROR` → item intact |
| Conversation opened but destroy failed | compensate: end the conversation (§6.4) |

The PRD's consume-first ordering would have required compensating a destroy for *both* common
cases. FR-4.4 ("must not consume when no conversation is authored") falls out of the ordering
instead of needing a rollback path — which is the whole argument for this shape.

### 6.2 New status topic

`atlas-npc-conversations` today produces **no** status topic; it only *consumes*
`EVENT_TOPIC_SAGA_STATUS` (`kafka/message/saga/kafka.go:7`) for sagas a conversation initiates. The
awaited-step design needs the opposite direction, so add one, modelled on the npc-shop contract
(`services/atlas-npc-shops/.../shops/kafka.go:55-64`):

```
EnvStatusEventTopic          = "EVENT_TOPIC_NPC_CONVERSATION_STATUS"
StatusEventTypeStarted       = "STARTED"
StatusEventTypeStartError    = "START_ERROR"
```

carrying `transactionId`, `characterId`, and for `START_ERROR` a `reason` discriminating
`no_conversation_authored` from `conversation_in_progress` from `internal_error`. The reason is what
makes a Loki trace of a content gap distinguishable from a real fault without reading code.

`START_ERROR` is deliberately a distinct type rather than a generic `ERROR`, for the same reason
npc-shops split `ENTER_ERROR` from `ERROR`: a generic error type is rendered differently by the
channel and would be ambiguous here.

This is a new Kafka topic and therefore needs the standard registration: env var in the service
configmaps and the topic itself in both kustomize overlays. An unsuffixed topic name silently falls
back and cross-talks between environments — the trap
[`docs/adding-a-new-service.md`](../../adding-a-new-service.md) documents.

### 6.3 New actions and payloads

`libs/atlas-saga/model.go` gains two actions in a new `// NPC conversation actions` block beside
`OpenNpcShop`:

```go
StartItemConversation Action = "start_item_conversation"
StartNpcConversation  Action = "start_npc_conversation"
```

Two explicit actions, not one action with a mode discriminator — the codebase's own convention
(`WarpToPortal` / `WarpToRandomPortal` / `WarpToSavedLocation`) is discrete actions per shape, and a
discriminator inside a payload would defeat the `handler.go` switch that gives each action its own
arm.

```go
// StartItemConversationPayload — 243 family. Like OpenNpcShop this step is NOT
// self-completing: it waits for EVENT_TOPIC_NPC_CONVERSATION_STATUS to report
// STARTED or START_ERROR, which is what lets the following destroy step consume
// the item only once the dialogue actually opened.
type StartItemConversationPayload struct {
    CharacterId uint32     `json:"characterId"`
    AccountId   uint32     `json:"accountId"`
    ItemId      uint32     `json:"itemId"`
    NpcTemplateId uint32   `json:"npcTemplateId"`
    WorldId     world.Id   `json:"worldId"`
    ChannelId   channel.Id `json:"channelId"`
    MapId       _map.Id    `json:"mapId"`
    Instance    uuid.UUID  `json:"instance"`
}

// StartNpcConversationPayload — 239 family, conversation branch.
type StartNpcConversationPayload struct {
    CharacterId   uint32     `json:"characterId"`
    AccountId     uint32     `json:"accountId"`
    NpcTemplateId uint32     `json:"npcTemplateId"`
    WorldId       world.Id   `json:"worldId"`
    ChannelId     channel.Id `json:"channelId"`
    MapId         _map.Id    `json:"mapId"`
    Instance      uuid.UUID  `json:"instance"`
}
```

New saga type `ScriptedItemUse Type = "scripted_item_use"` alongside `RemoteMerchant`.

Orchestrator wiring, one arm each, following `OpenNpcShop`'s five touch points exactly:

| File | Change |
|---|---|
| `saga/model.go` | re-export consts + payload aliases; unmarshal arm (`model.go:1332` pattern) |
| `saga/handler.go` | `handleStartItemConversation` / `handleStartNpcConversation`, non-self-completing, plus the dispatch-map entries |
| `saga/producer.go` | command providers onto `COMMAND_TOPIC_NPC` |
| `saga/event_acceptance.go` | `EventKindNpcConversationStarted` / `…StartError`, mapped for both actions |
| `saga/character_extractor.go` | extract `CharacterId` from both payload types |
| `saga/compensator.go` | reverse-walk arms (§6.4) |

`libs/atlas-saga/unmarshal.go` gains the two decode arms, with tests mirroring
`unmarshal_test.go:1252`.

### 6.4 Compensation

Reverse-walk arms mirror `compensator.go:1508`'s `OpenNpcShop → EXIT`: both new actions compensate
by dispatching `END_CONVERSATION` on `COMMAND_TOPIC_NPC`, so a character is never left standing in a
dialogue for an item they still hold. A dispatch failure logs and continues the chain, as the
`OpenNpcShop` arm does.

Because the destroy is the **last** step, the only path that reaches compensation is
"conversation opened, destroy failed" — genuinely rare, and its compensation is a UI teardown
rather than an item restore. That asymmetry is the point of the ordering.

Compensation must also explicitly unlock the client (§7.3), because on that path the inventory
delta that would otherwise have unlocked it never happened.

### 6.5 Idempotency under redelivery (FR-5.5)

Kafka is at-least-once, and a redelivered start command is the dangerous case: `Processor.Start`
returns `"another conversation exists"` when a context is already live
(`processor.go:104-108`), so a naive handler would emit `START_ERROR` for a conversation that
**it itself** already started, failing a saga that had already succeeded.

Mitigation: stamp the originating `transactionId` into the `ConversationContext` on start. On a
start command whose `transactionId` matches the live context's, re-emit `STARTED` rather than
`START_ERROR`. Only a *different* transaction id against a live context is a genuine
`conversation_in_progress` error.

The registry already carries a `sagaIndex` keyed by saga id
(`conversation/registry.go:17,45-50`), but it serves the opposite direction (sagas a conversation
initiated). Reuse the storage mechanism, not the existing field's semantics — a separate
`originTransactionId` on the context, or the plan may fold it into the existing index if it can show
the two uses cannot collide. Folding without that proof is how the two directions get confused.

The destroy step is idempotent on the orchestrator side already, via step status.

---

## 7. Channel handlers

### 7.1 `ScriptedItemHandleFunc` — 243

Structural model: `shop_scanner_item_use.go` (decode → classify → validate slot → act) with
`character_cash_item_use_remote_merchant.go`'s saga construction and logging.

```
decode ScriptedItem
if GetClassification(itemId) != ClassificationConsumableScriptedItem:
        warn (naming 3994225 explicitly if itemId == 3994225); EnableActions; return
a := GetItemInSlot(characterId, inventory.TypeValueUse, source)
if err or a.TemplateId() != itemId:
        warn; EnableActions; return
cd := consumable data for itemId
if cd.Npc == 0:
        warn "resolves to npc 0 — atlas-data may not have been re-ingested"; EnableActions; return
create saga{ ScriptedItemUse, [ start_item_conversation, destroy_asset_from_slot ] }
```

The `Npc == 0` guard is not paranoia: it is exactly the state every tenant is in until the §3.1
re-ingest lands, and the log line should say so rather than presenting as a mysterious content gap.
It mirrors `character_cash_item_use_remote_merchant.go:104-108`'s `ci.Npc == 0` check.

The "no conversation authored" case (FR-4.4) is **not** checked here — it is the saga's
`START_ERROR`, which is what lets the item survive without a pre-flight round trip and without a
TOCTOU window.

### 7.2 `NpcItemUseHandleFunc` — 239 / 545

```
decode NpcItemUse
switch GetClassification(itemId):
  ClassificationConsumableRemoteNpc (239):
        validate slot in inventory.TypeValueUse
        npcId := consumable data .Npc            // info/npc
        if shop exists for npcId -> saga[ open_npc_shop, destroy_asset_from_slot ]
        else                     -> saga[ start_npc_conversation, destroy_asset_from_slot ]
  ClassificationRemoteMerchant (545):
        validate slot in cash inventory
        npcId := cash item data .Npc
        saga[ open_npc_shop, destroy_asset_from_slot ]
  default:
        warn; EnableActions; return   // impossible from a legitimate client
```

The shop-vs-conversation probe reuses `shops.NewProcessor(l, ctx).GetShop(template)` exactly as
`npc_start_conversation.go:32` does.

**Interaction with task-221.** On v72–v95 both `CASH_ITEM_USE` and `NPC_ITEM_USE_REQUEST` exist, and
which one the client emits for a `545` item is decided by the caller
(`CDraggableItem::OnDoubleClicked`), not by the sender. The server therefore accepts **both** routes
and lets the client choose; neither handler may assume it is the only path. On **v61** — which
`remoteMerchantEnabled` correctly reports as unsupported for the `CASH_ITEM_USE` route, because 545
sits in that dispatcher's default arm — `NPC_ITEM_USE_REQUEST` is the *only* route, so this task
incidentally brings the remote merchant to v61. That is a real behavioural gain and must be
play-tested, not assumed.

Registration: `handlerMap[…Handle] = handler.…HandleFunc` in `atlas-channel/main.go` beside
`merchantsb.ShopScannerItemUseHandle` (`main.go:966`).

### 7.3 The excl-request / `EnableActions` contract (FR-3.3)

Every version sets `m_bExclRequestSent` after sending. The contract
(`reference_exclrequest_unlock_contract`): every path must resolve the lock exactly once, and **an
outcome that warps must not be unlocked** — the warp's `SET_FIELD` unlocks itself.

- **Rejection paths** — explicit `session.EnableActions`. Enumerated: wrong classification;
  `3994225` on v95; slot/template mismatch; `Npc == 0`; saga creation failure; unhandled
  classification in the `NpcItemUse` switch.
- **Success path** — no explicit unlock. The `destroy_asset_from_slot` step produces an inventory
  delta, and that delta is what clears `m_bExclRequestSent` client-side (the same mechanism the
  trade-staging design relies on, `libs/atlas-saga/model.go` `TransferToTrade` commentary). Issuing
  an explicit unlock *as well* would double-resolve the lock.
- **Compensation path** — explicit `EnableActions`, because the destroy that would have unlocked
  never happened (§6.4).
- **A seeded conversation that warps** — the warp owns the unlock; the handler must not add one.
  This constrains reference-content selection (§8).

Note the differing windows: `SCRIPTED_ITEM` uses `CanSendExclRequest(500, 0)`,
`NPC_ITEM_USE_REQUEST` uses `(200, 0)`. These are client-side rate limits only; the server does not
model them, but they matter when reading a play-test capture.

---

## 8. Reference content

The `0243` inventory (23 items, from the v83-era `Item.wz/Consume/0243.img.xml`) is recorded in
[`item-inventory.md`](item-inventory.md) — item id, `script`, `npc`, and `runOnPickup` — discharging
FR-6.2. The PRD's unverified "24 items" was close but wrong, and the ids `2430000`–`2430005` it
guessed do exist.

**The conversation content is ours to author.** Atlas conversations are declarative JSON state
machines, not ported scripts (PRD §2 non-goal), and FR-4.2 already makes the WZ `script` name
traceability-only rather than a lookup key. So the reference conversations do not have to reproduce
any original script's behaviour — the first pass only has to prove the *path*. This dissolves what
would otherwise be a blocking unknown, because the original semantics of these scripts are **not
verified**: of the 23 scripts, only `killarmush` and `removethorns` exist in the local Cosmic tree,
and both are map-gated, quest-touching, and self-consuming (`im.removeAll(...)`) — which is also
direct confirmation of the PRD §4.5 observation that Cosmic scripts consume themselves. The other
21 script bodies are unavailable here, and **no claim is made about what they originally did**.

Selection criterion for the first pass, in priority order:

1. The authored dialogue must **not warp** — §7.3 makes a warping outcome own its own unlock, which
   would test the unlock contract and the dispatch path at once and make any failure ambiguous.
2. It must **not grant items or touch quest state** — those add failure modes unrelated to this task.
3. The two items must have **different `npc` values**, so the play-test proves the avatar is carried
   per-item rather than defaulted.

Two items meeting the criterion, with distinct avatars:

1. **`2430013`** (`item_2430013`, npc `9010000`) — a plain two-node dialogue.
2. **`2430008`** (`compassUse`, npc `2084002`) — a branching dialogue on a different avatar.

Both are authored as pure dialogue for this pass regardless of their historical behaviour; the
`script` value is recorded on the row for traceability only. If a later content pass wants faithful
behaviour, it re-authors the state machine — no schema or code change is implied.

`2430010` (`openTreasure`) is explicitly **not** seeded: it carries `spec/runOnPickup = 1`, which is
a different trigger path (pickup, not use) that this task does not implement. Now that §3.1 makes
the flag visible for the first time, leaving it unseeded keeps that distinction observable rather
than accidentally exercised.

Seeded under `deploy/seed/<region>/<version>/npc-conversations/items/item-<id>.json` for each of the
eight in-scope versions, alongside the existing `npc/` and `quests/` directories.

No `239` reference content is seeded for GMS v83: the v83-era WZ corpus contains **no `0239.img`
at all**, so there is nothing to author against on that version. The `239` path is play-tested on a
version whose corpus has the family, or on `545` (Miu Miu, `5450000`) which task-221 already seeds.

---

## 9. Alternatives considered

**Consume-first with compensation** (the PRD's literal text). Rejected under D-1: it needs the same
status topic *and* a rollback path for the two most common failures, and every rollback is a chance
to lose a player's item. §6.1.

**Pre-flight REST check, no status topic.** `atlas-channel` GETs `/item-conversations/{itemId}`
before creating the saga. Cheapest — no new Kafka topic. Rejected because it cannot see
"character already in a conversation": that state lives in `atlas-npc-conversations`' Redis registry
and has no REST surface, so that case would still consume the item and show no dialogue. It also
introduces a TOCTOU window the awaited-step design does not have.

**One saga action with a mode discriminator** instead of `StartItemConversation` +
`StartNpcConversation`. Rejected: the orchestrator's `handler.go` dispatch switch is per-action, and
a discriminator inside the payload moves branching from a place the compensator and event-acceptance
tables can see into a place they cannot. §6.3.

**Extending `conversation/quest`'s dual state machine to items.** Rejected: `243` items have one
entry point, and the `endStateMachine` would be permanently `nil`, inviting a future reader to
wire it to something.

**Moving `npc` parsing from `info` to `spec` outright** rather than spec-with-info-fallback.
Rejected: `0239` genuinely authors under `info` (verified on `02390001`), and this task needs both
families to resolve. §3.1.

**Splitting `NPC_ITEM_USE_REQUEST` into a sibling task**, as the PRD recommended. Overridden by D-4.
The sweep in §1.2 also showed the PRD's premise for splitting was incomplete — three legacy columns
(v61/v72/v79) are mis-recorded as `n-a` for that op too, and correcting one op's matrix while
leaving the sibling's known-wrong is the kind of half-sweep that makes `n-a` untrustworthy.

---

## 10. Risks and operational notes

| Risk | Mitigation |
|---|---|
| **Parser fix without re-ingest leaves `Npc = 0` everywhere.** The feature is dead and looks like a content gap. | §7.1's explicit `Npc == 0` log names re-ingest as the likely cause. Re-ingest is a required deployment step, verified before play-test. |
| **Live tenants don't inherit seed-template edits.** New opcodes silently dropped. | PATCH live tenant socket configs; verify the handler appears in the live config, not just the template. |
| **Handler bound without a validator is silently dropped.** | Both handlers get validators; verified in the live config. |
| **v87 movement decode already spews `Code 254` at ~2k/min.** A standing confound for any v87 play-test report. | Play-test primarily on v83 and one legacy column; treat v87 anomalies as suspect until reproduced elsewhere. |
| **v61 remote merchant is newly reachable** via the `NPC_ITEM_USE_REQUEST` route. | Explicitly play-tested (§7.2), not assumed. |
| **Both codecs are serverbound `❌` cells** — a `❌` can mean an unverified shared codec rather than a missing one. | The `IMPLEMENTING_A_PACKET.md` Step-0 check runs first for each op; §1 already establishes neither has an existing decoder. |
| **Matrix `toolSha` reads git HEAD.** | Regenerate the matrix after the final commit, not mid-branch. |

---

## 11. Verification plan

Beyond `CLAUDE.md`'s standing gates (`go test -race`, `go vet`, `go build`, `docker buildx bake` for
every service whose `go.mod` moved, `tools/lint.sh --check`, `redis-key-guard`, `goroutine-guard`),
this task specifically requires:

- `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` — nine
  templates change.
- `tools/template-movement-types-guard.sh` — any template edit trips its scope.
- Byte-fixture verification per `VERIFYING_A_PACKET.md` for **17 cells**: `SCRIPTED_ITEM` × 8
  versions, `NPC_ITEM_USE_REQUEST` × 9 versions. A round-trip fixture is not a verification.
- Matrix regenerated; no `❌` in any in-scope column; no residual incorrect `⬜` for
  `SCRIPTED_ITEM` v72/v79 or `NPC_ITEM_USE_REQUEST` v61/v72/v79.
- `atlas-data` re-ingest confirmed: `GET /api/data/consumables/2430004` returns a non-zero `npc`
  and the correct `script`.
- Play-test on v83 **and** a legacy column (v72 or v79 — both newly claimed for both ops), covering:
  dialogue opens with the right avatar and consumes exactly one; an unauthored scripted item logs
  warn, consumes nothing, leaves the client responsive; an out-of-range item id is rejected; a
  slot/template mismatch is rejected; a redelivered command does not double-consume; v61 remote
  merchant opens via the new route.

Code review (`superpowers:requesting-code-review`) runs before the PR, plus
`packet-completeness-critic` against a `coverage-manifest.yaml` declaring both ops across all
in-scope versions.

---

## 12. Traceability

| PRD requirement | Design |
|---|---|
| FR-1.1 `gms_v12` resolved | §1.1, §1.2 — absent in both ops; not a matrix column (F-4) |
| FR-1.2 v48/v61 remain `n-a` | §1.1 v48/v61 for `SCRIPTED_ITEM`; §1.2 **corrects** v61 for `NPC_ITEM_USE_REQUEST` |
| FR-1.3 v72/v79 registry entries | §4.2 |
| FR-1.4 no wire change to verified ops | §4.1 — no version gates, no shared-codec edits |
| FR-2.1 codec with Encode+Decode | §4.1 |
| FR-2.2 `MajorAtLeast` if divergent | §4.1 — no divergence found across the full sweep |
| FR-2.3 opcode config-resolved | §4.2 |
| FR-2.4 template bindings + guards | §4.2 |
| FR-3.1 server-side range validation | §7.1; **amended** by §3.3 for `3994225` |
| FR-3.2 slot/template validation | §7.1, §7.2 |
| FR-3.3 excl-request contract | §7.3 |
| FR-3.4 no reliance on `IsAbleToConsume` | §7.1 — server validates independently |
| FR-4.1 `conversation/item/` family | §5.1 |
| FR-4.2 resolution by item id; npc is avatar | §5.1 |
| FR-4.3 new `COMMAND_TOPIC_NPC` command | §6.3 |
| FR-4.4 no conversation → warn, no consume, unlock | §6.1 — via ordering, not a rollback |
| FR-5.1 new saga action | §6.3 — two actions |
| FR-5.2 step ordering | §6.1 — **inverted** from the PRD (D-1, F-2) |
| FR-5.3 compensatable | §6.4 |
| FR-5.4 event acceptance + character extractor | §6.3 |
| FR-5.5 idempotent under redelivery | §6.5 |
| FR-6.1 reference conversations | §8 |
| FR-6.2 `0243.img` list from real data | §8, [`item-inventory.md`](item-inventory.md) |
| FR-6.3 seeded per version | §8 |
| O-1 `NPC_ITEM_USE_REQUEST` | **In scope** (D-4); §1.2, §2, §7.2 |
| O-2 `gms_v12` | Resolved — absent; §1.1 |
| O-3 `0243` inventory | Resolved; §8 |
| O-4 cancel semantics | Unchanged; §6.1's ordering leaves the terminal-`destroyAsset` option open |
| O-5 v84/v87/v92/v95/jms body confirmation | Resolved — full sweep, all identical; §1 |
