# Task 124 — Teleport Rocks: Execution Context

Companion to `plan.md`. Key files, locked decisions, and dependencies an implementer (or reviewer) needs without re-reading the whole design.

## Documents

- PRD: `docs/tasks/task-124-teleport-rocks/prd.md`
- Design (resolves all PRD §9 open questions, IDA-verified v83+v95): `design.md`
- Plan: `plan.md` (23 tasks)

## Locked decisions (do not relitigate during execution)

1. **Register sends no map id.** `TROCK_ADD_MAP` with `nType=1` carries only the list flag; the server uses the session's current map. PRD FR-5's "validate equals current map" wording is superseded by design §1 Q1.
2. **Persistence lives in atlas-character** (`teleport_rock` domain, table `teleport_rock_maps`, slot-per-row, unique `(tenant_id, character_id, list_type, slot)`). Mutations ride Kafka commands; REST is read-only (`GET /characters/{id}/teleport-rock-maps`).
3. **List rewrite, not slot surgery.** Delete/add rewrites the whole ≤10-row list (`replaceList`) inside one transaction — compaction is free and slot-uniqueness conflicts are impossible.
4. **VIP selector is `itemId/1000 == 5041`.** 2320000, 5040000, 5040001 → regular list; 5041000 → VIP. Client-verified (design §1 Q5).
5. **Field-limit bits 0x40|0x02 bar rock use** on source AND target (target-side is server policy). The save-side bar is the numeric rule `mapId/100000000 != 0 && (mapId/1000000)%100 != 9`, NOT fieldLimit (design §1 Q2).
6. **Continent restriction is server policy** (design §1 Q3): non-VIP rocks reject `continent(src) != continent(dst)` with mode 8, where `continent = mapId/100000000`.
7. **Warp-to-player is same tenant+world+channel only** (design §1 Q6); any miss → mode 6. No cross-channel flow.
8. **Saga = WarpToRandomPortal [+ DestroyAsset]**, new SagaType `teleport_rock_use`, no orchestrator changes (dispatch is per-action). Warp before destroy (FR-2). No success MAP_TRANSFER_RESULT — SetField is the success signal.
9. **Mode bytes are config-resolved only** (`WithResolvedCode("operations", key)`); the nine-key table ships with the writer row in the same commit, per version.
10. **Error mapping**: LIST_FULL/DUPLICATE/MAP_NOT_ALLOWED → MAP_NOT_AVAILABLE(10); NOT_FOUND → CANNOT_GO(5); modes 7 and 11 seeded but unemitted.
11. **Cash entry point**: `character_cash_item_use.go` enum-12 branch gated by `item.GetClassification == ClassificationTeleportRock` (504) — megaphone aliases of enum 12 keep warn-and-drop.
12. **Character-data threading is fail-open**: fetch failure at login sends empty lists, never blocks.
13. **gms_92 stays unverified** (no IDB); v84/v87/jms verification needs IDBs loaded — unresolvable fname = stop-and-ask (never substitute).

## Key pattern files (copy these shapes)

| What | Where |
|---|---|
| Domain package shape (entity/model/builder/administrator/provider/rest/resource) | `services/atlas-character/atlas.com/character/saved_location/` |
| Command consumer (character service) | `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` |
| Status-event producer providers | `services/atlas-character/atlas.com/character/character/producer.go` |
| sqlite test harness | `services/atlas-character/atlas.com/character/character/processor_test.go:20-48` |
| Channel status consumer → session.Announce | `services/atlas-channel/atlas.com/channel/kafka/consumer/messenger/consumer.go` |
| Channel REST read package | `services/atlas-channel/atlas.com/channel/character/key/` |
| Package-var test seams in handlers | `services/atlas-channel/atlas.com/channel/socket/handler/mystic_door_enter.go:25-51` |
| fieldLimit validation precedent | `services/atlas-channel/atlas.com/channel/skill/handler/mysticdoor/mysticdoor.go:103-110` |
| Saga construction from a handler | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:60-105` |
| Config-resolved mode bodies | `libs/atlas-packet/messenger/operation_body.go:24-28` + `libs/atlas-packet/resolve.go` |
| Byte-fixture test with verify markers | `libs/atlas-packet/door/clientbound/remove_test.go` |
| Codec round-trip helper | `libs/atlas-packet/character/data_test.go:14-54` (`pt.RoundTrip`, `pt.Variants`) |

## Integration points being modified

- `libs/atlas-packet/character/data.go:700-720` — `encodeTeleports`/`decodeTeleports` stop hardcoding `EmptyMapId`.
- `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:16` — `BuildCharacterData` gains a 4th param; call sites: `set_field.go:31`, `cash_shop_open.go:19`, `character_data_test.go:26`.
- `services/atlas-channel/atlas.com/channel/main.go` — handlerMap (~:798), produceWriters (~:608/788), InitConsumers (~:198), InitHandlers register chain (~:416-437).
- `services/atlas-character/atlas.com/character/main.go` — migrations (:68), consumers (:71-88), routes (:100).
- `services/atlas-character/atlas.com/character/character/processor.go:304-325` — delete tx gains `teleport_rock.DeleteForCharacter` (first sub-domain cleanup in this service — intentional).
- Seed templates ×6 + `deploy/k8s/base/env-configmap.yaml` + pr/main overlays + `deploy/compose/.env.example`.

## Wire contracts (authoritative summary)

- `USE_TELEPORT_ROCK` (sb): `short slot, int itemId, Target, int updateTime` (trailing on ALL versions). Target may be absent → decode invalid → handler warn-drop.
- `TROCK_ADD_MAP` (sb): `byte register, byte vip, [int mapId if delete]`.
- Cash branch (sb, after ItemUse prefix): `Target, int updateTime`.
- `MAP_TRANSFER_RESULT` (cb): `byte mode, byte vipFlag, [5|10 × int mapId for modes 2/3]`.
- Kafka: `COMMAND_TOPIC_TELEPORT_ROCK` (`ADD_MAP`/`REMOVE_MAP` {MapId, Vip}), `EVENT_TOPIC_TELEPORT_ROCK_STATUS` (`LIST_UPDATED` {Vip, Registered, Maps[]}, `ERROR` {Vip, Reason}). Envelopes carry `TransactionId, WorldId, CharacterId, Type`.

## Opcode table (Task 20 / rollout)

| Version | USE | ADD_MAP | RESULT |
|---|---|---|---|
| gms_83 | 0x54 | 0x66 | 0x2A |
| gms_84 | 0x54 | 0x66 | 0x2A |
| gms_87 | 0x57 | 0x69 | 0x2A |
| gms_92 | 0x5B | 0x71 | 0x2B |
| gms_95 | 0x5B | 0x72 | 0x29 |
| jms_185 | 0x4C | 0x61 | 0x27 |

Operations table (all versions, v84/v87/jms re-confirmed at Task 22): DELETE_LIST=2, REGISTER_LIST=3, CANNOT_GO=5, UNABLE_TO_LOCATE=6, UNABLE_TO_LOCATE_2=7, CANNOT_GO_CONTINENT=8, CURRENT_MAP=9, MAP_NOT_AVAILABLE=10, MAPLE_ISLAND_LEVEL7=11.

## IDA anchors (Task 22 evidence)

- v83 (`MapleStory_dump.exe`): SendMapTransferItemUseRequest `0xA0A3BB`, SendMapTransferRequest `0xA261BC`, RunMapTransferItem `0xA0A4AA`, OnMapTransferResult `0xA25268` (names applied + IDB saved during design).
- v95 (`GMS_v95.0_U_DEVM.exe`): `0x9E6020`, `0x9F3B90`, `0x9E11C0`, `0x9F9F90`.
- v84/v87/jms: fnames absent from checked-in exports; needs IDBs loaded (`list_instances`, match by binary NAME — the loaded set rotates).

## External dependencies / risks

- Task 22 needs live IDA instances (only external dependency). Stop-and-ask on unavailable IDB or unresolvable fname.
- `ExecuteTransaction` no-op caveat (known bug): worst case here is a partial list rewrite, repaired by the next full-list event — acceptable per design §12.
- Live tenants need the config PATCH + channel restart post-merge (rollout section in plan.md) or the ops silently no-op.
- Bake set: atlas-character, atlas-channel, atlas-saga-orchestrator, atlas-login (lib ripple).
