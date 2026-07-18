# Teleport Rocks (Regular + VIP) — Design

Task: task-124-teleport-rocks
Status: Approved PRD → this design resolves PRD §9 and fixes the architecture.
Sources of truth used: IDA v83 (`MapleStory_dump.exe`, port 13342) and v95 (`GMS_v95.0_U_DEVM.exe`, port 13341) live decompiles; local GMS 83.1 WZ dump (`tmp/ec876921-c363-4cc6-9c51-5bb8d57f9553/GMS/83.1/` at the main repo root); repo source cited as `file:line`.

---

## 1. Resolved Open Questions (PRD §9)

All client-side answers below are IDA-verified on **both** v83 and v95; the two versions agree on every layout and semantic, which brackets v84/v87. jms/v84/v87 byte-fixture confirmation happens at execution time per §10 (their IDBs are not currently loaded).

During this design pass the four previously-unnamed v83 functions were named in the shared v83 IDB and the IDB was saved: `CWvsContext::SendMapTransferRequest` (0xA261BC), `CWvsContext::RunMapTransferItem` (0xA0A4AA), `CUIMapTransfer::OnRegister` (0x83A084), `CUIMapTransfer::OnDelete` (0x83A392).

### Q1 — Request layouts (VERIFIED)

**`USE_TELEPORT_ROCK` — `CWvsContext::SendMapTransferItemUseRequest`** (v83 `0xA0A3BB` op 0x54; v95 `0x9E6020` op 0x5B — both match the registry):

```
short  nPOS          // USE-inventory slot of the rock
int    nItemID       // client guard: nItemID/10000 == 232 (regular rock ONLY on this op)
<target payload — see RunMapTransferItem below>
int    updateTime    // trailing, both versions (no leading updateTime on this op, even v95)
```

**Target payload — `CWvsContext::RunMapTransferItem`** (v83 `0xA0A4AA`, v95 `0x9E11C0`), shared by both the regular-rock op and the cash path:

```
byte bByName
  1 → string targetName   // length-prefixed ASCII (EncodeStr)
  0 → int dwTargetField    // saved map id (only encoded when selection != 999999999)
```

Caveat (both versions): if the dialog closes OK with neither a name nor a valid map, the client sends the packet with **no target payload at all**. The decoder must treat a truncated/absent payload as a validation failure (warn + drop), not crash.

**Cash rocks do NOT use `USE_TELEPORT_ROCK`.** They ride the existing cash-item-use op (`CWvsContext::SendConsumeCashItemUseRequest`, v83 op 0x4F): common prefix (v95+ leading updateTime — already modeled by `updateTimeFirst` in `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:32`), `short nEPOS`, `int nItemID`, then for the teleport-rock case the same `RunMapTransferItem` payload, then trailing `int updateTime` (v83 tail at `0xA0EA53`; v95 case at `0x9EE059`). This is exactly the `CashSlotItemType(12)` branch that currently falls through to warn-and-drop.

**`TROCK_ADD_MAP` — `CWvsContext::SendMapTransferRequest`** (v83 `0xA261BC` op 0x66; v95 `0x9F3B90` op 0x72, signature `(long nType, ulong dwTargetField, unsigned char bCanTransferContinent)`):

```
byte nType                  // 1 = register, 0 = delete
byte bCanTransferContinent  // list selector: 0 = regular (5), 1 = VIP (10)
if nType == 0:
  int dwTargetField         // map to delete
```

**Critical correction to PRD FR-5:** on register (`nType=1`) the client sends **no map id** (v95 `CUIMapTransfer::OnRegister` `0x7DFE30` calls `SendMapTransferRequest(1, 0, m_bCanTransferContinent)`; v83 identical at `0x83A084`). The server must take the character's **current map** from server-side session state, not from the packet. There is nothing to "validate equals current map" — the client never claims one.

### Q2 — Field-limit semantics (VERIFIED, two different mechanisms)

- **Use-side (warp) bar, current field**: `RunMapTransferItem` refuses to open the dialog when `CField::IsEventMap(field)` **or** `fieldLimit & 0x40` **or** `fieldLimit & 0x02` (v83 `0xA0A4CF-0xA0A55E` reads `field+324`; v95 reads the same member with `& 0x40` / `& 2` explicitly). So the rock-use ban bits are **0x40** and **0x02**.
- **Save-side (register) bar is NOT fieldLimit.** Both clients use a numeric rule in `CUIMapTransfer::OnRegister`: a map may be saved iff `mapId / 100000000 != 0 && (mapId / 1000000) % 100 != 9` (bars all sub-9-digit maps — Maple Island, Masteria, GM maps — and every `x09xxxxxxx` event block). Violation → the mode-10 string.
- WZ cross-check (GMS 83.1 `Map.wz`): the 0x40 bit is set exactly where expected — Free Market Entrance 910000000 `fieldLimit=8428` (0x20EC), Zakum's Altar 280030000 `fieldLimit=2564732`, Horntail cave 240060200 `fieldLimit=2433148` — all have `&0x40 != 0`.
- Atlas plumbing already exists: atlas-data parses `fieldLimit` (`services/atlas-data/atlas.com/data/map/reader.go:81`), atlas-channel exposes it (`services/atlas-channel/atlas.com/channel/data/map/model.go:25`), and the mystic-door handler is the precedent for channel-side fieldLimit validation (`services/atlas-channel/atlas.com/channel/skill/handler/mysticdoor/mysticdoor.go:103-110`). `libs/atlas-constants/map/field_limit.go` needs one new constant for `0x40` (`FieldLimitNoTeleportItem`); `0x02` already exists as `FieldLimitNoMysticDoor` (Atlas's name for the bit — the client uses the same bit for both bans).
- Server policy addition: the client can only check the **source** field; the server also applies `0x40|0x02` to the **target** map (mode 5 on violation). This half is policy, not client-verifiable.

### Q3 — Regular-rock region restriction (client does NOT enforce; server policy)

Verified: neither client pre-filters warp targets by region — the only regular-vs-VIP distinctions client-side are which list is shown and the `bCanTransferContinent` flag (official symbol name, v95). The restriction is therefore server-side by contract. **Policy (design decision):** `continent(mapId) := mapId / 100000000`; a regular rock (and 5040000/5040001) rejects targets on a different continent with **mode 8**; VIP (5041000) skips the check. This is grounded in the flag's official name but is not client-verifiable — marked as policy, not IDA fact.

### Q4 — `MAP_TRANSFER_RESULT` mode set (VERIFIED, identical v83 `0xA25268` ↔ v95 `0x9F9F90`)

Wire: `byte mode`, `byte targetList` (0 = regular, 1 = VIP; always present), then for modes 2/3 exactly 5 (regular) or 10 (VIP) × `int mapId` (pad with 999999999). Client behavior per mode:

| mode | client action (both versions) |
|---|---|
| 0, 1, 4 | nothing (default arm) |
| 2, 3 | reload the full list into `adwMapTransfer[5]` / `adwMapTransferEx[10]` + refresh UI — **2 and 3 are handled identically** |
| 5 | "You cannot go to that place." |
| 6, 7 | "%s is currently difficult to locate, so the teleport will not take place." (target name) — 6 and 7 identical |
| 8 | "You cannot go to that place." (same string as 5, distinct case) |
| 9 | "It's the map you're currently on." |
| 10 | "This map is not available to enter for the list." |
| 11 | "Users below level 7 are not allowed to go out from Maple Island." |

Since 2/3 (and 6/7) are client-identical, Atlas's assignment — `DELETE_LIST=2`, `REGISTER_LIST=3`, `UNABLE_TO_LOCATE=6` — is safe regardless of what the original server sent. The full operations table (all nine keys) ships per version (§8).

### Q5 — Teleport Coke (VERIFIED)

Both clients compute the list selector as `bCanTransferContinent = (nItemID / 1000 != 5040)` (v83 `0xA0CAB0`: `cmp eax, 13B0h`; v95 `0x9EE059`). So **5040000 and 5040001 are regular-list**, 5041xxx is VIP. PRD assumption confirmed.

### Q6 — Warp-to-player cross-channel (server policy)

The packet carries only a name; the client has no channel semantics here. **Policy:** resolve the target within the same tenant + world + **channel** (session registry lookup). Offline, other-channel, cash-shop, or nonexistent targets all reject with mode 6. Cross-channel warp is out of scope (no channel-change flow is triggered). Mode 7 is left unemitted (client-identical spare).

### Item data (WZ-verified)

`Item.wz/Consume/0232.img.xml` entry 02320000 has **info only** (`slotMax=1, only=1, timeLimited=1, tradeBlock=1, notSale=1`) — no `spec` node. atlas-data needs **no item-side change**; PRD §7's "no change expected" holds.

---

## 2. Architecture Overview

Three cooperating changes, following the PRD's service split:

```
client ──USE_TELEPORT_ROCK / CASH_ITEM_USE(rock)──▶ atlas-channel handler
                                                        │ validate (item, source/target fieldLimit,
                                                        │ list membership via REST, continent, same-map,
                                                        │ player lookup)
                                                        ├─ fail → MapTransferResult error (inline write)
                                                        └─ ok  → saga: WarpToRandomPortal → DestroyAsset(all rocks: 2320000 + 504xxxx)

client ──TROCK_ADD_MAP──▶ atlas-channel handler ──Kafka COMMAND_TOPIC_TELEPORT_ROCK──▶ atlas-character
                                                        (register: mapId = session field, server-derived)
        atlas-character validates (capacity, duplicate, eligibility) + persists
                 └─Kafka EVENT_TOPIC_TELEPORT_ROCK_STATUS─▶ atlas-channel consumer
                            └─ session.Announce → MapTransferResult (mode 2/3 refresh or error)

login/field-enter: atlas-channel GET /characters/{id}/teleport-rock-maps → CharacterData codec fields
```

Alternatives considered:

- **Persistence in atlas-channel (Redis or local DB)** — rejected: lists are durable character state (survive relog/channel change), belong to the character-state owner; Redis would also route through `libs/atlas-redis` for no gain.
- **REST mutation from channel instead of Kafka commands** — rejected: game-flow mutations in this codebase ride Kafka (PRD §5 contract); REST is read-side only here, matching `saved_location`'s read pattern while grafting the command-consumer pattern already used by the character command topic.
- **Synchronous validate-in-channel for add/remove** — rejected: capacity/duplicate checks belong with the data owner (atlas-character); the channel would race its own cache. Channel supplies only the server-derived current map id.
- **One row per list (array column / JSON)** — rejected in favor of the PRD's slot-per-row schema: matches `saved_location`'s composite-unique-index pattern, keeps mutations single-row, and makes capacity a processor concern.

## 3. atlas-character — new `teleport_rock` domain

Copy the `saved_location` package shape wholesale (`services/atlas-character/atlas.com/character/saved_location/*` — entity/model/builder/provider/administrator/processor/rest/resource, `entity.go:13-24` for the struct pattern), then graft the Kafka command/status patterns from the character domain.

**Entity** (table `teleport_rock_maps`, per PRD §6):

```go
type entity struct {
    ID          uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
    TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:1"`
    CharacterId uint32    `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:2"`
    ListType    string    `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:3"` // "regular" | "vip"
    Slot        int       `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:4"`
    MapId       _map.Id   `gorm:"not null"`
}
```

Capacity (5/10) enforced in the processor. **Delete compacts slots** (remaining entries shift down) so the wire list is always a contiguous prefix + `EmptyMapId` padding — matches how the client renders a full-list refresh.

**Processor** (`Interface + Impl`, `NewProcessor(l, ctx, db)`, tenant via `tenant.MustFromContext`): pure `AddMap(mb)(characterId, mapId, vip)` / `RemoveMap(mb)(characterId, mapId, vip)` + `AndEmit` variants; `GetByCharacterId(characterId) (Model, error)` returning both lists. Validations (each buffers an ERROR status event instead of mutating):

- ADD: eligibility `mapId >= 100000000 && (mapId/1000000)%100 != 9` → `MAP_NOT_ALLOWED`; list full → `LIST_FULL`; already present in that list → `DUPLICATE`.
- REMOVE: not present → `NOT_FOUND`.

**Kafka** (new files `kafka/message/teleportrock/kafka.go`, `kafka/consumer/teleportrock/consumer.go`, wired in `main.go` next to the existing consumers at `services/atlas-character/atlas.com/character/main.go:73-87`):

- `COMMAND_TOPIC_TELEPORT_ROCK`: envelope `Command[E]{TransactionId, WorldId, CharacterId, Type, Body}`; `ADD_MAP{MapId, Vip}`, `REMOVE_MAP{MapId, Vip}`.
- `EVENT_TOPIC_TELEPORT_ROCK_STATUS`: `LIST_UPDATED{Vip bool, Maps []uint32}` (full post-mutation list for the affected list only) and `ERROR{Vip bool, Reason string}` with reasons `LIST_FULL | DUPLICATE | MAP_NOT_ALLOWED | NOT_FOUND`.

**REST**: `GET /characters/{characterId}/teleport-rock-maps` — single resource, `GetName() = "teleport-rock-maps"`, attributes `{regular: [mapId…], vip: [mapId…]}` (unpadded; padding is the codec's job). Registered via `InitResource` in `main.go` beside `saved_location.InitResource` (`main.go:100`).

**Deletion cleanup**: `character.Delete`'s transaction (`services/atlas-character/atlas.com/character/character/processor.go:304-325`) gains a delete of `teleport_rock_maps` rows for the character — atlas-character has **no** sub-domain cleanup today (verified gap), so this is the first; same-transaction is the correct lifecycle (option (a); a status-event consumer would be self-consumption).

**Mock**: `teleport_rock/mock/processor.go`, `ProcessorMock` func-field struct per the fame/notes convention (atlas-character has no mocks yet; this introduces the standard shape).

**Migration**: append to `database.SetMigrations(...)` (`main.go:68`). Net-new table; no backfill.

## 4. atlas-channel — handlers, use flow, status projection

### 4.1 `TROCK_ADD_MAP` handler (new `socket/handler/teleport_rock_add_map.go`)

Decode per Q1. `nType=1` → emit `ADD_MAP` with `MapId: s.Field().MapId()` (server-derived current map — never from the packet). `nType=0` → `REMOVE_MAP` with the decoded map id. `Vip` from `bCanTransferContinent`. Fire-and-forget; all feedback arrives via the status consumer.

### 4.2 Status consumer (new `kafka/consumer/teleportrock/consumer.go`)

Follow the messenger consumer end-to-end shape (`kafka/consumer/messenger/consumer.go:27-107`): `InitConsumers`/`InitHandlers` + `session.NewProcessor(...).IfPresentByCharacterId(...)` + `session.Announce(...)(MapTransferResultWriter)(body)`.

- `LIST_UPDATED` → mode `REGISTER_LIST` (add) / `DELETE_LIST` (remove), targetList flag, list padded to 5/10 with `EmptyMapId`. (FR-7: the client only updates its UI from this packet.)
- `ERROR` → reason → mode: `LIST_FULL | DUPLICATE | MAP_NOT_ALLOWED → MAP_NOT_AVAILABLE(10)`, `NOT_FOUND → CANNOT_GO(5)`. (Client prechecks full/duplicate itself; these fire only for bypassed clients, and mode 10's string is the closest faithful message.)

### 4.3 Use flow (shared by both entry ops)

New `teleportrock` channel package owning `UseRock(l, ctx, wp)(s, rock, target)`:

1. **Item check**: regular path — `GetItemInSlot(TypeValueUse, slot)` template id must be 232xxxx and equal the decoded item id; cash path — existing cash-slot check in `character_cash_item_use.go:37-41`. VIP := `itemId/1000 != 5040` (mirrors the client exactly).
2. **Source field**: `fieldLimit & (FieldLimitNoTeleportItem|FieldLimitNoMysticDoor) != 0` → mode 5. (Data via the existing `data/map` processor, mystic-door precedent.)
3. **Target resolution**:
   - By map: fetch `/characters/{id}/teleport-rock-maps`; target must be in the list matching the rock (regular list for 2320000/5040000/5040001, VIP list for 5041000) → else mode 5. Same map as current → mode 9. Target `fieldLimit & (0x40|0x02)` → mode 5. Regular rocks: `continent(source) != continent(target)` → mode 8.
   - By name: same-tenant/world/channel session lookup → miss → mode 6; then the target-map checks above against the player's current field (membership check does not apply).
4. **Failure**: write `MapTransferResult` error inline from the handler (the `BlockedMapWriter` announce pattern, `socket/handler/mystic_door_enter.go:47-49`). Nothing is consumed (FR-1).
5. **Success**: one saga, new `SagaType` `teleport_rock_use` (constant added to `libs/atlas-saga`; the orchestrator dispatches by step *action*, `services/atlas-saga-orchestrator/.../saga/handler.go:77-84`, so no orchestrator logic changes):
   - Step 1 `WarpToRandomPortal` (existing action, `libs/atlas-saga/model.go:57`) to the target field. Random spawn portal is the faithful era behavior for rock warps and avoids inventing a new action (PRD §7 constraint satisfied).
   - Step 2 (**all teleport rocks** — 2320000 and cash 5040000/5040001/5041000; see PRD correction) `DestroyAsset{Quantity:1}`. Warp-before-destroy ordering guarantees a failed warp never consumes (FR-2); a destroy failure after warp is benign in the fail-open direction (item survives).
   - Cash rocks: step 1 only. No `MAP_TRANSFER_RESULT` is sent on success — the warp's SetField is the success signal (client sets no pending state that needs clearing; verified: `m_sMapTransferTargetUserName` is cleared on any result *or* overwritten next use).
6. **Cash entry point**: `character_cash_item_use.go` type-12 branch — decode the rock payload and delegate to `UseRock`. Two wrinkles: the handler currently discards its `writer.Producer` (`:25`) — un-discard it; and `GetCashSlotItemType` also maps some megaphones to enum 12 (`:181`) — branch on `item.GetClassification(itemId) == item.ClassificationTeleportRock`, keeping the warn-and-drop fallthrough for the megaphone alias.

### 4.4 Character-data codec threading (FR-15/16)

- `libs/atlas-packet/character/data.go`: `CharacterData` gains `TeleportMaps []_map.Id` (regular) and `VipTeleportMaps []_map.Id`; `encodeTeleports`/`decodeTeleports` (`data.go:700-720`) read/write them, padding to 5/10 with `EmptyMapId`, preserving the `(GMS && MajorVersion() > 28) || JMS` gate. Round-trip tests stay green by symmetry; add a golden fixture pinning a populated teleport region (none exists today, so nothing breaks — verified).
- `services/atlas-channel/.../socket/writer/character_data.go:16` `BuildCharacterData` gains the lists as a parameter (like `bl buddylist.Model`); both call sites (`set_field.go:31`, `cash_shop_open.go:19`) fetch via a new channel-side `character/teleportrock` requests+processor (`GET characters/%d/teleport-rock-maps` against `RootUrl("CHARACTERS")`, mirroring `character/key/requests.go:10-20`). Fetch failure logs warn and threads empty lists (fail-open — a missing list must not block login).

## 5. libs/atlas-packet — new packet artifacts

New `teleportrock` package:

- `teleportrock/serverbound/use.go` — `Use{Slot int16, ItemId uint32, ByName bool, TargetName string, TargetMap uint32, UpdateTime uint32}`, `Operation() = "TeleportRockUseHandle"`; decoder tolerates the absent-target-payload edge (Q1 caveat) by flagging invalid rather than erroring the session.
- `teleportrock/serverbound/add_map.go` — `AddMap{Register bool, Vip bool, MapId uint32}`, `Operation() = "TeleportRockAddMapHandle"`.
- Cash payload: a thin shared codec for the `RunMapTransferItem` target payload, reused by `cash/serverbound` for the type-12 branch (shared-model wrapper pattern, per the AttackInfo/TOUCH precedent).
- `teleportrock/clientbound/result.go` — writer name const `MapTransferResultWriter = "MapTransferResult"`; bodies via `atlas_packet.WithResolvedCode("operations", KEY, ...)` (`libs/atlas-packet/resolve.go:13`), messenger-style (`libs/atlas-packet/messenger/operation_body.go:24-28`):
  - `MapTransferResultListBody(vip bool, maps []_map.Id)` for `REGISTER_LIST`/`DELETE_LIST` (writes mode, flag, padded 5/10 ints),
  - `MapTransferResultErrorBody(key, vip)` for the error modes (mode, flag byte).
- Byte-fixture tests with `packet-audit:verify` markers for all three ops (door `remove_test.go` is the template).

Channel registration: `handlerMap` entries in `main.go` (~`:798-891`) and the writer const in `produceWriters()` (~`:698`).

## 6. Kafka / saga contracts (summary)

| Topic env | Direction | Types |
|---|---|---|
| `COMMAND_TOPIC_TELEPORT_ROCK` | channel → character | `ADD_MAP{MapId,Vip}`, `REMOVE_MAP{MapId,Vip}` |
| `EVENT_TOPIC_TELEPORT_ROCK_STATUS` | character → channel | `LIST_UPDATED{Vip,Maps[]}`, `ERROR{Vip,Reason}` |
| `COMMAND_TOPIC_SAGA` (existing) | channel → orchestrator | new `SagaType: teleport_rock_use`; steps `WarpToRandomPortal` [+ `DestroyAsset`] |

Deploy manifests: add the two new topic env vars to atlas-character and atlas-channel k8s bases (same places the existing `COMMAND_TOPIC_CHARACTER`/status topics are declared).

## 7. Error-mode mapping (server emission policy)

| Condition | Mode key | Byte |
|---|---|---|
| add/remove success | `REGISTER_LIST` / `DELETE_LIST` | 3 / 2 |
| source or target field barred; target not in list; player target's map barred; NOT_FOUND on remove | `CANNOT_GO` | 5 |
| player target unresolvable (offline / other channel / cash shop / nonexistent) | `UNABLE_TO_LOCATE` | 6 |
| regular-rock continent mismatch | `CANNOT_GO_CONTINENT` | 8 |
| target is current map | `CURRENT_MAP` | 9 |
| add-map rejected (eligibility, full, duplicate) | `MAP_NOT_AVAILABLE` | 10 |
| reserved, unemitted (client-verified values) | `UNABLE_TO_LOCATE_2`=7, `MAPLE_ISLAND_LEVEL7`=11 | 7 / 11 |

Mode 11 is table-seeded but unemitted: sub-level-7 characters on Maple Island cannot legitimately hold a rock, and every Maple Island map fails the save-eligibility rule anyway.

## 8. Seed templates + live rollout (FR-17/18/19)

Per-version wiring, all six in-scope templates (`services/atlas-configurations/seed-data/templates/template_gms_{83,84,87,92,95}_1.json`, `template_jms_185_1.json`; gms_12 excluded per PRD scope):

| Version | USE_TELEPORT_ROCK handler | TROCK_ADD_MAP handler | MAP_TRANSFER_RESULT writer |
|---|---|---|---|
| gms_83 | 0x54 | 0x66 | 0x2A |
| gms_84 | 0x54 | 0x66 | 0x2A |
| gms_87 | 0x57 | 0x69 | 0x2A |
| gms_92 | 0x5B | 0x71 | 0x2B |
| gms_95 | 0x5B | 0x72 | 0x29 |
| jms_185 | 0x4C | 0x61 | 0x27 |

Sources: registry YAMLs (`docs/packets/registry/*.yaml`, csv-import provenance) + CSV columns (`docs/packets/MapleStory Ops - *.csv`); v83/v95 serverbound opcodes additionally IDA-confirmed this pass (0x54/0x66 and 0x5B/0x72 read directly from the `COutPacket` constructors). v84 clientbound 0x2A is below the known ≥0x3D v84 shift region. gms_92 has no IDB — template-lineage values, cells stay unverified (PRD FR-20 exception).

- Both handler rows get `"validator": "LoggedInValidator"` (a validator-less entry is silently dropped — known failure class).
- The writer row carries the full nine-key `operations` map (values from §1 Q4, identical v83↔v95; v84/v87/jms re-confirmed at execution). Missing keys are a client-crash class (`ResolveCode` returns 99 on miss, `libs/atlas-packet/resolve.go:27`), so the table ships in the same commit as the writer row.
- **Live tenants do not re-seed.** Rollout step (documented in plan): PATCH each live tenant's socket config with the two handler rows + writer row (+ operations), then restart atlas-channel (handlers/writers don't hot-reload).

## 9. Data flow / consistency notes

- Add/remove is one transaction in atlas-character (single list mutation + compaction); the status event carries the authoritative post-mutation list, so the client can never render drift.
- The channel never caches lists; it fetches at login/field-enter (codec) and per use (membership check). Single indexed lookup each — no caching layer (NFR).
- Warp+consume atomicity is the saga's ordering; validation failures never reach the saga (FR-1/FR-2).
- Multi-tenancy: every row keyed by `tenant_id`; consumers get tenant via header parsers; REST via `tenant.MustFromContext`.

## 10. Packet-verification plan (FR-20/21)

Per `docs/packets/audits/VERIFYING_A_PACKET.md`, one packet-verifier pass per op × version:

- Evidence anchors already pinned this pass: v83 `SendMapTransferItemUseRequest 0xA0A3BB`, `SendMapTransferRequest 0xA261BC`, `RunMapTransferItem 0xA0A4AA`, `OnMapTransferResult 0xA25268`; v95 `0x9E6020`, `0x9F3B90`, `0x9E11C0`, `0x9F9F90`. (v83 names applied + IDB saved.)
- v84/v87/jms: the three fnames are absent from the checked-in ida-exports (verified), so execution needs those IDBs loaded (`list_instances` first; the loaded set rotates). If an IDB is unavailable or an fname doesn't resolve, that cell is a stop-and-ask — never substituted.
- gms_92: remains at the unverified designation (no IDB), template values from lineage.
- Deliverables per verified cell: byte-fixture test with `packet-audit:verify` marker, evidence YAML under `docs/packets/evidence/<version>/`, STATUS.md regeneration; `packet-audit matrix --check` and `operations --check` exit 0. MAP_TRANSFER_RESULT is a single-writer mode packet (not a dispatcher family) — `dispatcher-lint` not applicable, but every mode arm gets a body fixture (list-refresh both flags, plus one error mode), per the no-mode-byte-only-verification rule.

## 11. Testing

- **atlas-character**: processor unit tests (capacity, duplicate, eligibility, compaction, not-found; Builder-pattern setup, no test-helper files); consumer handler tests per existing command-consumer tests; REST transform tests.
- **libs/atlas-packet**: round-trips via `pt.Variants` for all new codecs; golden fixtures for the result bodies and the populated CharacterData teleport region; the absent-target-payload edge decodes as invalid, not panic.
- **atlas-channel**: use-flow validation table tests (each rejection → expected mode) with mocked character/map/session processors; cash-branch disambiguation test (megaphone id still warn-drops, 504x routes).
- **Verification gates** (CLAUDE.md): `go test -race`, `go vet`, `go build` per changed module; `docker buildx bake atlas-character atlas-channel` (+ any module whose go.mod changes — atlas-saga bump ripples to the orchestrator image too); `tools/redis-key-guard.sh` (no new Redis usage expected).

## 12. Risks / edge cases

- **Missing operations key = client crash (99)** — the writer row and its operations table land atomically per template; `operations --check` gates.
- **Megaphone/rock enum-12 alias** — classification check in the cash branch (explicitly tested).
- **Client sends register with no map id** — server-derived map closes the spoof hole the PRD's original FR-5 wording implied.
- **Empty target payload on use** — decoder-tolerated, handler warn-drops (no result packet; the client sent a malformed request).
- **Live-tenant patch forgotten** — symptom would be "unhandled message op" at info; rollout step is explicit in the plan (known failure class).
- **`ExecuteTransaction` no-op caveat** — the add/remove mutation is single-statement-set inside one `tx`; even under the known no-op-transaction bug the worst case is a partial slot-compaction, which the next full-list event repairs. No dependency on cross-service atomicity.

## 13. Out of scope (unchanged from PRD)

Married-partner transfer, Dojo/other `OnTeleport` family, Hyper Rock, cash-shop purchase flow, atlas-ui.
