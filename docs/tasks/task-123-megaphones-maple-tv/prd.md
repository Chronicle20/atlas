# Megaphones (All Tiers) & Maple TV — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Megaphones are the primary player-to-world communication items in MapleStory: a player uses a cash item and a message is broadcast beyond their map — channel-wide for the basic Megaphone, world-wide (all channels) for Super/Item/Triple/Avatar Megaphones, and via the on-screen Maple TV overlay for TV items. They are a core social and economy feature (trade advertising, guild recruitment) present in every version Atlas targets.

Atlas currently has **no player-initiated megaphone path**. The serverbound `USE_CASH_ITEM` handler (`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`) classifies megaphone items (`ClassificationMegaphones` = 507 → cash-slot item types 12–15/45–52; `ClassificationAvatarMegaphone` = 539 → type 42/43) but has no branch for any of them — they fall through to the warn log at `character_cash_item_use.go:110` and the item is never consumed, nothing is broadcast. The clientbound side is partial: the `WorldMessage` writer (`CWvsContext::OnBroadcastMsg`) exists with mode constants for `MEGAPHONE`, `SUPER_MEGAPHONE`, `ITEM_MEGAPHONE`, `MULTI_MEGAPHONE`, etc., and is emitted by several **server-initiated** consumers (system_message, mount, party_quest, message, session, gachapon), but the item-megaphone body (attached-item serialization) and the entire Avatar Megaphone (`SET_AVATAR_MEGAPHONE`, `CLEAR_AVATAR_MEGAPHONE`, `AVATAR_MEGAPHONE_RESULT`) and Maple TV (`SEND_TV`, `REMOVE_TV`, `ENABLE_TV`) packet families are unimplemented (STATUS.md: ❌ for all versions).

This task implements the full player flow end-to-end for **all currently supported tenant versions**: serverbound decode of each megaphone sub-body, item consumption via saga, world-wide broadcast fan-out over Kafka (gachapon announce pattern), the missing clientbound writers with byte-fixture verification, the Maple TV broadcast queue, and seed-template/live-config wiring.

## 2. Goals

Primary goals:
- A player using any megaphone tier (Megaphone 5071xxx, Super 5072xxx, Item 5076xxx, Triple/Multi 5077xxx, Avatar 539xxxx) or a Maple TV item (5075xxx) sees the correct broadcast on every affected client, and the item is consumed exactly once.
- Broadcast reach is correct per tier: channel-wide for basic Megaphone; world-wide (all channels of the world) for Super/Item/Triple/Avatar and Maple TV.
- All new packets (serverbound sub-bodies and clientbound writers) are byte-fixture verified against the client per `docs/packets/audits/VERIFYING_A_PACKET.md`, for every version with an IDB; the coverage matrix (STATUS.md) is regenerated.
- New opcodes/handlers/writers/operations-table entries are added to **every** supported seed template and the live-tenant patch runbook is documented (seed templates only apply at tenant creation).

Non-goals:
- Moderation guards (mute checks, minimum level) — explicitly out of scope per product decision.
- GM/server-initiated notices (already implemented via existing consumers).
- Cash shop purchase flow for these items (items are assumed present in inventory).
- Persisting megaphone/TV history (no DB entities; broadcasts are fire-and-forget, TV queue is runtime state).
- Chat-log / audit tooling.

## 3. User Stories

- As a player, I want to use a Megaphone (5071000) so that everyone on my channel sees my message.
- As a player, I want to use a Super Megaphone so that everyone in my world sees my message with my name and channel, and whisper-availability flag.
- As a player, I want to use an Item Megaphone with an item attached so that everyone in my world can see the exact item I'm advertising.
- As a player, I want to use a Triple/Multi Megaphone so that I can broadcast up to three lines at once world-wide.
- As a player, I want to use an Avatar Megaphone so that my character's appearance is shown alongside my message on every client in the world.
- As a player, I want to send a Maple TV message (optionally featuring a partner character) so that it plays on the TV overlay for everyone in the world, queued behind other TV messages.
- As a player receiving a broadcast, I want megaphone messages to render with the sender's name (and medal decoration where applicable, matching the existing gachapon/notice conventions).

## 4. Functional Requirements

### FR-1 Serverbound decode (`USE_CASH_ITEM` megaphone branches)

1.1. Extend `CharacterCashItemUseHandleFunc` with branches for each megaphone cash-slot item type currently classified by `GetCashSlotItemType` (types 12, 13, 14, 15, the TV group 45–52, and the avatar-megaphone type 42/43). Each branch decodes its type-specific sub-body via a new `cashsb.ItemUse*` packet struct in `libs/atlas-packet/cash/serverbound`.

1.2. Sub-body fields must be derived from the client (`CWvsContext::SendConsumeCashItemUseRequest` and the per-dialog senders listed in STATUS.md row `USE_CASH_ITEM`, e.g. `CItemSpeakerDlg::_SendConsumeCashItemUseRequest` for the item megaphone) via IDA for each version with an IDB — **not** from Cosmic alone. Known shape expectations to verify: message string(s) (3 for triple), whisper flag, attached-item inventory-type + slot reference (item megaphone), TV type/partner-character name/durations (TV), message for avatar megaphone. The v83 trailing `updateTime` noted at `character_cash_item_use.go:108` must be resolved as part of this decode work.

1.3. Decode failures or item/slot mismatches log a warn and return without consuming the item (matching the existing guard at `character_cash_item_use.go:38-41`).

1.4. For the item megaphone, the referenced item is resolved from the character's inventory at decode time and **snapshotted** into the broadcast payload (no persistence, no later re-resolution). If the referenced slot is empty or mismatched, the use is rejected (no consume, no broadcast).

### FR-2 Orchestration (consume + broadcast, channel-initiated)

2.1. Each accepted use creates a saga (pattern: the existing `FieldEffectUse` saga in the same handler): step 1 `DestroyAsset` (consume exactly 1 of the cash item), step 2 emit the broadcast event. The broadcast must not happen if the consume fails (compensation per existing saga semantics).

2.2. Broadcast fan-out follows the gachapon `RewardWonEvent` pattern: a Kafka event carrying tenant headers, world id, sender character id/name/medal, message payload, and tier-specific fields; every atlas-channel instance consumes it, filters `sc.IsWorld(...)`, and announces to all sessions in its channel.

2.3. Reach rules: basic Megaphone (type 12) announces to the sender's **channel** only (filter world+channel); Super/Item/Triple/Avatar announce to **all channels of the world**. Maple TV announces world-wide via the TV packet family.

2.4. Sender name decoration reuses the existing medal conventions in `socket/writer/world_message.go` (`decorateNameForMessage` / `decorateMegaphoneMessage`).

### FR-3 Clientbound writers — WorldMessage modes

3.1. Implement the item-megaphone `WorldMessage` body: message + channel + whisper flag + optional attached-item block (item serialization matching the client's read, `GW_ItemSlotBase`-style). New packet struct in `libs/atlas-packet/chat/clientbound` alongside the existing `WorldMessageSuperMegaphone`.

3.2. Implement/verify the multi-megaphone (triple) body (message lines + channel + whisper flag) and wire the existing `MULTI_MEGAPHONE` / `ITEM_MEGAPHONE` mode constants.

3.3. `WorldMessage` is a mode-prefix dispatcher-family packet (`CWvsContext::OnBroadcastMsg`). Per project policy (dispatcher mode-byte false-pass, `docs/packets/DISPATCHER_FAMILY.md`): every mode arm this task ships must have its full per-mode body implemented and byte-fixtured for every supported version — enumerating mode bytes in the operations table is **not** verification. Mode bytes are resolved from the tenant `operations` table (`WithResolvedCode`), never hard-coded — including version-stable arms.

### FR-4 Clientbound writers — Avatar Megaphone family

4.1. Implement `SET_AVATAR_MEGAPHONE` (`CWvsContext::OnSetAvatarMegaphone`), `CLEAR_AVATAR_MEGAPHONE` (`CWvsContext::OnClearAvatarMegaphone`), and `AVATAR_MEGAPHONE_RESULT` (`CWvsContext::OnAvatarMegaphoneRes`) per the client read order (IDA). The set packet is expected to carry the sender's appearance (avatar look); the exact encoding must be IDA-verified per version.

4.2. Character look data is sourced from the existing character/appearance provider used by other look-encoding writers in atlas-channel (verify at design time which provider owns it; do not duplicate look-encoding logic).

4.3. Avatar megaphone display duration / clear timing must match client behavior (verify whether the client auto-clears or the server must schedule `CLEAR_AVATAR_MEGAPHONE`; if server-scheduled, the same world-scoped scheduler as Maple TV (FR-5) is used).

### FR-5 Maple TV

5.1. Implement `SEND_TV` (`CMapleTVMan::OnSetMessage`), `REMOVE_TV` (`CMapleTVMan::OnClearMessage`), and `ENABLE_TV` (`CMapleTVMan::OnSendMessageResult`) per client read order for each version.

5.2. TV messages form a **serialized queue per world**: one active message at a time; subsequent sends queue and play in order; each TV item tier has a fixed display duration (values verified from client/WZ/Cosmic at design time — do not invent). `REMOVE_TV` (or the client-driven equivalent — verify) ends a message; the next queued message then plays.

5.3. The queue owner must be world-scoped and survive the multi-pod channel topology (one atlas-channel deployment per world×channel — a single channel pod cannot own world state in memory alone). The mechanism is a design decision; candidate options to evaluate in design.md: state in atlas-world, or Redis via `libs/atlas-redis` (respecting the redis-key-guard invariant — no raw keyed go-redis calls outside that lib). Requirement here is only: exactly-once playback ordering per world, no double-play across channel pods, queue drains correctly if a pod restarts.

5.4. If the TV item references a partner character (verify sub-body), the partner's look/name is included per the client read order.

5.5. Whether late joiners (players logging in mid-broadcast) receive the active TV message must be verified against client/Cosmic behavior at design time and matched.

### FR-6 Configuration wiring

6.1. Add the new clientbound writer opcodes (avatar-mega family, TV family) and any new `operations` mode-table entries to **all** supported seed templates in `services/atlas-configurations/seed-data/templates/` (`gms_83`, `gms_84`, `gms_87`, `gms_92`, `gms_95`, `jms_185`; `gms_12` applicability is an open question §9). Per-version opcodes come from STATUS.md/IDA — modes and opcodes are version-dependent and must not be copied across versions unverified.

6.2. Every new `socket.handlers` entry must include a `validator` (missing validator = silently dropped handler). `USE_CASH_ITEM` handling is an existing routed handler, but any new handler entries follow the same rule.

6.3. Document the live-tenant config patch (existing tenants do not receive seed changes; the projection does not hot-reload handlers/writers, so channel restart is required). Include this in the task's rollout notes.

### FR-7 Version support & verification tiers

7.1. In scope: every version with a seed template and provisioned tenants — gms v83, v84, v87, v92, v95, jms v185.

7.2. Versions with an IDB (v83, v84 via table+v83 structure rules, v87, v95, jms) get byte-fixtured verification per `VERIFYING_A_PACKET.md` and matrix promotion. v92 has **no IDB** (established precedent: v92 mount-food parked) — v92 gets template wiring with the structure of the nearest verified version and its matrix cells remain unverified; this must be bannered honestly, not claimed verified.

7.3. Known version pitfalls to respect: `MajorVersion() > 83` off-by-one (use `>= 87`; v84 takes the v83 path structurally), v84 clientbound opcode-table shift ≥ 0x3D, and the missing per-version `operations` tables in v87/v95/jms templates (any WorldMessage mode this task uses must have its operations entry present in every template it ships to, resolved from that version's dispatcher switch).

### FR-8 Documentation & matrix

8.1. Regenerate `docs/packets/audits/STATUS.md` after evidence is pinned; the `USE_CASH_ITEM` serverbound row and the new clientbound rows must reflect actual verification state per version.

8.2. Service docs for atlas-channel updated if its documented capabilities list broadcasts.

## 5. API Surface

No new REST endpoints.

Kafka (names finalized at design time, following existing `EnvEventTopic*` conventions in atlas-channel):
- New broadcast event message type(s) covering: megaphone tier, world id, channel id (sender's), sender character id/name/medal, message line(s), whisper flag, attached-item snapshot (item megaphone), avatar-look reference or payload (avatar megaphone), TV fields (type, partner, durations). One topic with a typed body vs. per-family topics is a design decision.
- All events carry tenant headers (`consumer.TenantHeaderParser` pattern) and are consumed with `kafka.LastOffset` (broadcasts are not replayable state).

Saga:
- Reuses `saga.DestroyAsset`. New saga type / step action(s) for "emit broadcast" analogous to `saga.FieldEffectWeather` — exact step taxonomy is a design decision.

## 6. Data Model

No new database entities and no migrations.

Runtime state only:
- Maple TV queue: world-scoped, tenant-scoped queue of pending TV messages plus the active message with its expiry. Ownership/mechanism per FR-5.3 (design decision). Multi-tenancy: all keys/state scoped by tenant id + world id.
- Item-megaphone attached item: snapshot embedded in the Kafka event at send time; never re-read later (product decision Q6).

## 7. Service Impact

- **atlas-channel** — main surface: new `USE_CASH_ITEM` sub-decode branches + saga creation in `socket/handler/character_cash_item_use.go`; new Kafka consumer(s) for broadcast events (gachapon pattern); writer bodies for item/triple megaphone modes, avatar-mega family, TV family; TV queue participation.
- **libs/atlas-packet** — new serverbound `cash/serverbound` sub-body structs; new clientbound structs (item-megaphone WorldMessage body, avatar-mega family, TV family) + byte-fixture tests.
- **atlas-saga** — new saga type / broadcast step action if the existing step taxonomy doesn't cover "emit broadcast event" (design decision; FieldEffectWeather precedent suggests a small addition).
- **atlas-world** — only if chosen as the TV queue owner (design decision FR-5.3).
- **atlas-configurations** — seed template updates for all supported versions (writers, operations tables, any handler entries).
- **atlas-query / character look provider** — read-only usage for avatar-megaphone look data (no changes expected; verify at design).
- **docs/packets** — evidence records, matrix regeneration, dispatcher family docs if WorldMessage arms are formalized.

## 8. Non-Functional Requirements

- **Multi-tenancy:** every Kafka event carries tenant headers; all processors use `tenant.MustFromContext(ctx)`; TV queue state is tenant+world scoped; no cross-tenant leakage of broadcasts.
- **No hard-coded protocol bytes:** all mode bytes via the tenant `operations` table; all opcodes via tenant templates; inbound decodes reverse-resolve the same tables writers use where applicable.
- **Verification:** `go test -race`, `go vet`, `go build` clean in every changed module; `docker buildx bake atlas-<svc>` for every touched service; `tools/redis-key-guard.sh` clean (mandatory if Redis is chosen for TV state); `packet-audit` matrix/fname-doc/operations `--check` and `dispatcher-lint` exit 0.
- **Resilience:** broadcast consumers tolerate missing character lookups (log + skip, per gachapon consumer); a failed broadcast must not strand the saga (item already consumed → broadcast step failure semantics defined in design).
- **Observability:** broadcast emissions and TV queue transitions logged at info with character/world/tier fields (matching gachapon's structured log fields).
- **Performance:** broadcasts are fan-out writes to in-memory session lists; no per-broadcast DB reads beyond the sender/character lookup already done in the handler. TV queue operations are low-frequency; no polling loops tighter than the queue's duration granularity.

## 9. Open Questions

1. **gms_12 template:** a `template_gms_12_1.json` exists (task-113 legacy versions). Whether a v12-era client supports any of these items/packets is unverified — design phase must check the v12 template/IDB availability and either include or explicitly exclude it.
2. **TV queue owner:** atlas-world vs. Redis (FR-5.3) — decide in design after enumerating existing libs (per the audit-existing-libs rule).
3. **Broadcast topic taxonomy:** one megaphone event topic with a tier discriminator vs. separate topics per family (WorldMessage vs. avatar-mega vs. TV) — design decision.
4. **Avatar megaphone clear timing:** client-driven vs. server-scheduled `CLEAR_AVATAR_MEGAPHONE` (FR-4.3) — IDA verification required.
5. **TV durations & late-joiner sync:** exact per-tier durations and whether the active TV message is replayed to newly connecting sessions (FR-5.2/5.5) — verify from client/Cosmic, do not invent.
6. **Heart/Skull megaphone items (types 45–52 group):** confirm from v83 WZ (`Item.wz/Cash/0507.img.xml`, 13 items) which tiers actually exist per version and which client bodies they use; only ship arms for items that exist.

## 10. Acceptance Criteria

- [ ] Using Megaphone 5071000 broadcasts channel-wide; the item is consumed; observed on a live v83 tenant.
- [ ] Using a Super Megaphone broadcasts world-wide across ≥2 channels with sender name/channel/whisper flag; item consumed.
- [ ] Using an Item Megaphone shows the attached item hover on a receiving client; empty/mismatched slot rejects without consuming.
- [ ] Using a Triple Megaphone broadcasts all three lines world-wide; item consumed.
- [ ] Using an Avatar Megaphone shows the sender's avatar + message on all clients in the world and clears at the correct time; item consumed.
- [ ] Sending a Maple TV message plays it on the TV overlay world-wide; a second send while one is active queues and plays after; items consumed; behavior consistent across channel pods.
- [ ] Failed consume (e.g. item vanished between decode and saga) results in no broadcast; failed broadcast semantics match the design's compensation decision.
- [ ] Every new packet struct has byte-fixture tests with `packet-audit:verify` markers for every IDB-backed version; STATUS.md regenerated; v92 cells honestly unverified.
- [ ] `dispatcher-lint`, `matrix --check`, `fname-doc --check`, `operations --check` all exit 0.
- [ ] All supported seed templates updated (writers, operations entries, validators present on any handler entries); live-tenant patch steps documented in the task folder.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` clean for every touched service; `tools/redis-key-guard.sh` clean.
- [ ] No `// TODO`, stubs, or 501s in landed commits; the existing `TODO` comment at `character_cash_item_use.go:108` is resolved (not carried forward).
