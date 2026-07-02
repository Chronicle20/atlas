# task-123 megaphones-maple-tv — Implementation Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer (or reviewer) needs, without re-reading the whole PRD/design.

## What this task builds

Player-initiated megaphones (basic / super / item / triple / avatar) and Maple TV, end-to-end: serverbound decode → saga (consume item, then broadcast) → Kafka fan-out → clientbound rendering on every affected client, plus a world-scoped serialized queue for TV/avatar in atlas-world, seed-template wiring for 5 versions, and per-version byte-fixture verification.

## Authoritative documents

| Doc | Role |
|---|---|
| `docs/tasks/task-123-megaphones-maple-tv/prd.md` | Requirements (FR-1…FR-8, acceptance criteria) |
| `docs/tasks/task-123-megaphones-maple-tv/design.md` | Decisions D1–D11 + design-phase IDA/Cosmic/WZ evidence (§1) |
| `docs/tasks/task-123-megaphones-maple-tv/plan.md` | Task-by-task implementation plan (21 tasks) |
| `docs/packets/DISPATCHER_FAMILY.md` | Mandatory recipe for the WorldMessage family enrollment (Task 18) |
| `docs/packets/audits/VERIFYING_A_PACKET.md` | Mandatory procedure for every verification cell (Tasks 19–20) |

## Key decisions (from design.md, binding)

- **D1 TV/avatar queue owner = atlas-world**: new `broadcast` domain; Redis `TenantRegistry` CAS (namespace `world-broadcast`, key `<worldId>:<family>`); 1 s sweep gated by `atlas-lock` `LeaderElection` (lease `world-broadcast-sweep`). No new lib.
- **D2 Kafka taxonomy**: `EVENT_TOPIC_MEGAPHONE` (tier-discriminated event, stateless tiers) + `COMMAND_TOPIC_WORLD_BROADCAST` / `EVENT_TOPIC_WORLD_BROADCAST_STATUS` (coordinator pair). Status types `QUEUED|STARTED|ENDED`. All `LastOffset` on the channel side; the world-side command consumer is NOT LastOffset.
- **D3 saga shape**: saga type `megaphone_use`; steps `[DestroyAsset, EmitMegaphone]` or `[DestroyAsset, EnqueueWorldBroadcast]` (+ third `EmitMegaphone` step for Megassenger TV tiers ≥3 — Cosmic parity, plan Task 12). Wait caps enforced at decode time via REST to atlas-world (TV > 3600 s, avatar > 15 s → rejection packet, item untouched); REST failure = conservative reject. No refund after consume (matches FieldEffectUse; no DestroyAsset compensation exists).
- **D4 item megaphone**: `WorldMessageItemMegaphone` re-cut around `model.Asset` (`GW_ItemSlotBase` block); asset snapshotted at decode into the saga/event payload, never re-resolved.
- **D5 dispatcher family**: WorldMessage enrolled with exactly 4 arms (MEGAPHONE — new discrete struct, SUPER_MEGAPHONE, ITEM_MEGAPHONE, MULTI_MEGAPHONE); `docs/packets/dispatchers/worldmessage.yaml` authored from each version's `OnBroadcastMsg` switch via IDA; op row stays 🧩 until all arms verify; NOT in `families.yaml`, NOT baselined.
- **D6/D8 durations**: TV 15/30/60 s (tvType 4→30, 5→60; Cosmic-sourced server policy); avatar 10 s (client auto-clears at 10 000 ms — server ENDED clear is idempotent redundancy). SEND_TV `totalWaitTime` computed by the coordinator.
- **D7**: no late-joiner replay.
- **D9 version scope**: gms_83/84/87/95 + jms_185 full; **gms_12 and gms_92 excluded** (login-only templates — flagged PRD deviation); jms has **no AVATAR_MEGAPHONE_RESULT** (rejection is silent for JMS).
- **D10**: `SenderMedal` plumbed everywhere, always `""` today (no medal source in atlas-channel).
- **D11**: `MAPLE_TV_USE_RES` stays unimplemented; 5070000/5073000 have no send path; type-15 (507x8xxx) ships no arm.

## Services & files touched

| Surface | Where | Plan tasks |
|---|---|---|
| Serverbound sub-body codecs | `libs/atlas-packet/cash/serverbound/item_use_{megaphone,super_megaphone,item_megaphone,triple_megaphone,maple_tv,avatar_megaphone}.go` | 1 |
| Saga lib | `libs/atlas-saga/{model,payloads,unmarshal}.go` — `MegaphoneUse`, `EmitMegaphone`, `EnqueueWorldBroadcast`, `AssetSnapshot`, `AvatarSnapshot` (snapshot DTOs are the single source of truth, reused by all kafka message structs) | 2 |
| WorldMessage arms | `libs/atlas-packet/chat/clientbound/world_message.go` (new `WorldMessageMegaphone`; re-cut `WorldMessageItemMegaphone`), `libs/atlas-packet/chat/world_message_body.go` (new package `chat`, `field_effect_body.go` model) | 3, 4 |
| Avatar family codecs | `libs/atlas-packet/chat/clientbound/avatar_megaphone.go` (Set/Clear/Result) | 5 |
| TV family codecs | `libs/atlas-packet/tv/clientbound/` (new package: SetMessage/ClearMessage/SendMessageResult) | 6 |
| World coordinator | `services/atlas-world/atlas.com/world/broadcast/` (model/registry/processor/resource/rest/task) + `kafka/{message,producer,consumer}/broadcast/` + `main.go`; go.mod adds atlas-saga, atlas-lock | 7–9 |
| Orchestrator | `saga/handler.go` (two fire-and-forget handlers, `handleFieldEffectWeather` model at :2855), `saga/producer.go`, `kafka/message/{megaphone,broadcast}/` | 10 |
| Channel | `kafka/message/{megaphone,worldbroadcast}/`, `worldbroadcast/` REST client, `socket/handler/character_cash_item_use{,_megaphone}.go`, `socket/model/snapshot.go`, `kafka/consumer/{megaphone,worldbroadcast}/`, `socket/writer/world_message.go`, `main.go` | 11–15 |
| Config/deploy | 5 seed templates (writers + USE_CASH_ITEM handlers), `deploy/k8s/base/env-configmap.yaml` + pr/main overlays (3 new topic vars) | 15, 16 |
| Packet audit | `docs/packets/dispatchers/worldmessage.yaml`, `tools/packet-audit/cmd/run.go` `#`-entries, evidence/fixtures/STATUS.md | 18–20 |

## Reference implementations to copy (do not invent new patterns)

- Handler saga branch: `character_cash_item_use.go:60-106` (FieldEffectUse).
- World-broadcast consumer skeleton: `services/atlas-channel/.../kafka/consumer/gachapon/consumer.go` (LastOffset, TenantHeaderParser, `sc.IsWorld`, `AllInChannelProvider` + `session.Announce`).
- Single-session ack: `kafka/consumer/buddylist/consumer.go:87` (`IfPresentByCharacterId(sc.Channel())`).
- Redis registry + tenant tracking: `services/atlas-world/.../channel/registry.go` (TenantRegistry + `atlas.Set` tenants, `trackTenant`).
- Leader-gated sweep: `services/atlas-summons/.../main.go:98-121`; ticker: `services/atlas-world/.../tasks/task.go` + `channel/task.go`.
- Orchestrator fire-and-forget handler: `saga/handler.go:2855-2880` (`handleFieldEffectWeather`); event provider: `saga/producer.go:148` (`GachaponRewardWonEventProvider`, `kproducer.CreateKey` + `SingleMessageProvider`).
- Body functions with resolved mode: `libs/atlas-packet/field/field_effect_body.go`; dispatcher `#`-entries: `tools/packet-audit/cmd/run.go:2051-2072`.
- Asset → packet conversion: `services/atlas-channel/.../socket/model/asset.go:13` (`NewAsset`); look builder: `socket/model/avatar.go:10` (`NewFromCharacter`).
- Fixture marker shape: `libs/atlas-packet/chat/clientbound/general_test.go:9-11`.
- Template entry shapes: `template_gms_83_1.json` → handler `{"opCode":"0x4F","validator":"LoggedInValidator","handler":"CharacterCashItemUseHandle"}`, writer `{"opCode":"0x44","writer":"WorldMessage","options":{"operations":{…19 keys…}}}`.

## Protocol evidence (verified during design — cite, don't re-derive)

- Opcode table (STATUS.md lines 88, 142–145, 446–448) is reproduced in plan.md "Key protocol facts"; jms lacks AVATAR_MEGAPHONE_RESULT.
- Client read orders (v83≡v95, IDA): SEND_TV / ENABLE_TV / REMOVE_TV / SET_AVATAR_MEGAPHONE / AVATAR_MEGAPHONE_RESULT — full field lists in design §1.2 and plan "Key protocol facts". ENABLE_TV is sender-feedback only (never broadcast — Cosmic's zero-filled broadcast is a known sloppiness we do not copy).
- Serverbound sub-bodies are **Cosmic-derived** (`UseCashItemHandler.java:290-372`, checkout at the sibling `Cosmic` repo) until Tasks 19–20 IDA-verify each version. TV sub-body conditionals (tvType gates) and the Megassenger double-broadcast come from Cosmic case 5; `sendTV` flag byte = `partner != nil ? 3 : 1`, messageType = `tvType <= 2 ? tvType : tvType - 3` (`PacketCreator.java:943-966`); durations from `MapleTVEffect.java:56-61`.
- Item classification: dispatch classification-first (`item.GetClassification` 507/539) then `(itemId/1000)%10` — cash-slot type 12 collides with teleport rock (task-124 in flight) and 42 with pet evolution. 5074000 Skull is TV-family **only** on GMS≥95.
- The `updateTimeFirst` convention (GMS≥95 reads updateTime before the sub-body; earlier versions after) resolves the `character_cash_item_use.go:108` TODO — the TODO comment must be deleted, not carried.

## Environment prerequisites / gotchas

- **IDA host**: Tasks 18–20 need v83(dump)/v84/v87/v95/jms IDBs open; as of 2026-07-02 only v95 + v83-dump were loaded. Always `list_instances` and match binary NAME; `select_instance(port)`. Use `func_query` with `name_regex`.
- **Kafka topic env vars** must be added in three deploy files (base configmap + pr/main overlays) or the services silently get empty topics.
- **Seed templates apply only at tenant creation** — live tenants need the PATCH + channel-restart runbook (Task 17). Handlers without validators are silently dropped.
- **WorldMessage operations tables** in v84/87/95/jms templates are unverified v83 copies today; Task 18 verifies per-version and `operations --check` begins enforcing them via `worldmessage.yaml`.
- **redis-key-guard**: all new Redis access must go through `libs/atlas-redis` types (`TenantRegistry.Update` is WATCH/CAS; `Update` returns `ErrNotFound` on missing key — the registry `Upsert` handles create-on-missing).
- **atlas-world runs replicas:2 without leader election today** — the broadcast sweep must be leader-gated from day one (summons pattern) or STARTED/ENDED events double-fire constantly.
- **`docker buildx bake`** for atlas-channel/world/saga-orchestrator/configurations is mandatory before claiming done; `go build` won't catch Dockerfile COPY gaps (no new libs added, so no Dockerfile edits expected).
- **jms rejection path**: avatar-megaphone over-cap rejection sends nothing on JMS (no result op).

## Dependency order

Packet/saga libs (1–6) → world coordinator (7–9) ∥ orchestrator (10) ∥ channel messages (11) → handler (12) → consumers (13–14) → wiring/deploy (15) → templates/runbook (16–17) → IDA enrollment + verification (18–20) → final gates + live acceptance (21). Full graph at the end of plan.md.
