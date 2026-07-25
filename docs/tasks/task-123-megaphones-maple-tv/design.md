# Megaphones (All Tiers) & Maple TV — Design

Task: task-123-megaphones-maple-tv
Status: Proposed
PRD: `docs/tasks/task-123-megaphones-maple-tv/prd.md`

---

## 1. Design-phase evidence (verified, with sources)

Everything in this section was verified during design; nothing is from memory. Sources: the
v95 IDB (`GMS_v95.0_U_DEVM.exe`), a named v83 IDB (`MapleStory_dump.exe`, v83_Me memory dump —
both currently open on the IDA host), the Cosmic reference server, v83 WZ XML
(Cosmic checkout `wz/`), and the repo itself (file:line cites).

### 1.1 Item inventory (v83 WZ: `Item.wz/Cash/0507.img.xml`, `0539.img.xml`, names from `String.wz/Cash.img.xml`)

| Item | Name | Cash-slot type (<v95 / GMS≥95) per landed classifier |
|---|---|---|
| 5070000 | Cheap Megaphone | 0 / 0 — no send path |
| 5071000 | Megaphone | 12 / 12 |
| 5072000 | Super Megaphone | 13 / 13 |
| 5073000 | Heart Megaphone | 0 / 0 — no send path |
| 5074000 | Skull Megaphone | 0 / 45 |
| 5075000–5075002 | MapleTV / Star / Heart Messenger | 46/47/48 → +1 on GMS≥95 |
| 5075003–5075005 | Megassenger / Star / Heart | 49/50/51 → +1 on GMS≥95 |
| 5076000 | Item Megaphone | 14 / 14 |
| 5077000 | Triple Megaphone | 60 / 61 |
| 507x8xxx | (no such item in v83 WZ) | 15 |
| 5390000/1/2/5/6 | Avatar megaphones (Diablo, Cloud 9, Loveholic, Cute/Roaring Tiger) | 42 / 43 |

Classifier: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:176-246`
(megaphones), `:351-357` (avatar megaphone). **Type-number collisions exist**: 12 is also
teleport rock (`:137-139`, task-124 in flight) and 42 is also pet evolution on GMS≥95
(`:344-349`). Dispatch therefore keys on `item.GetClassification` (507/539) first, then
cash-slot type — never on the numeric type alone.

### 1.2 Client behavior (IDA-verified this phase)

- **SEND_TV read order (v83 ≡ v95, verified in both IDBs)** — `CMapleTVMan::OnSetMessage`
  (v95 `0x60f870`, v83 dump `0x6371c1`): `byte flag` (bit 1 unused-by-us, bit 2 = has
  receiver look), `byte messageType` (0 normal / 1 star / 2 heart), `AvatarLook sender`,
  `str senderName`, `str receiverName` (always present; empty when none), `str ×5 message
  lines`, `int totalWaitTime` (seconds — stored as `m_nTotalWaitTime`), then
  `AvatarLook receiver` iff `flag & 2`.
- **ENABLE_TV read order (v83 ≡ v95)** — `CMapleTVMan::OnSendMessageResult` (v95 `0x60f5f0`,
  v83 `0x6373a0`): `byte hasError`; if nonzero, `byte code`: 1 = "non-GM character tried to
  send GM message", 2 = "the waiting line is longer than an hour, please try later",
  3 = "you've entered the wrong user name". `byte 0` = silent success. **This is
  sender-feedback, not a broadcast** (Cosmic broadcasts a zero-filled one to everyone —
  a sloppiness we do not copy).
- **REMOVE_TV** — `CMapleTVMan::OnClearMessage` reads nothing (0x14/0x1a bytes; clears display).
- **TV wait-time UX** — `CMapleTVMan::ConfirmTimeRemaining` (called 6× inside
  `CWvsContext::SendConsumeCashItemUseRequest`) shows "your message will air in %d seconds"
  from `m_nTotalWaitTime` before the client even sends. Real GMS therefore ran a **queue**
  with server-computed wait times — the PRD's queue requirement is client-corroborated.
- **Avatar megaphone auto-clear = client-side, 10 000 ms** — `CWvsContext::Update` (v95
  `0x9ea7f0`: `tCurTime - m_tAM_LastUpdate > 10000 → ByeAvatarMegaphone`; v83 dump
  `0xa03350`, `mov edi, 2710h` at `0xa037a9` gating the same call). `OnClearAvatarMegaphone`
  (v83+v95) is a guarded early-clear: `if (m_bAvatarMegaphone) ByeAvatarMegaphone()` —
  idempotent, safe to send redundantly.
- **SET_AVATAR_MEGAPHONE read order (v83 ≡ v95)** — `CWvsContext::OnSetAvatarMegaphone`
  (v95 `0xa017e0`, v83 `0xa2a486`): `int itemId`, `str name`, `str ×4 message lines`,
  `int channel`, `byte whisper`, `AvatarLook sender`.
- **AVATAR_MEGAPHONE_RESULT (v83 `0xa2a3bc`)** — `byte code`: 83 = "the waiting line is
  longer than 15 seconds, please try later", 84 = level-10 gate (moderation — out of scope),
  anything else = `str` shown as a notice dialog. **Real GMS serialized avatar megaphones
  world-scoped with a 15-second wait cap** — same shape as the TV queue, shorter horizon.

### 1.3 Cosmic reference behavior (local Cosmic checkout, cited as reference only — wire shapes get per-version IDA verification at implementation)

- `UseCashItemHandler.java:290-372` — serverbound sub-bodies: basic mega `str`; super
  `str + byte whisper`; TV `[type-dependent: byte, byte ear, str partnerName] + 5×str + int`;
  item mega `str + byte whisper + byte hasItem + int invType + int slot`; triple
  `byte lines(1..3) + str×lines + byte whisper`; avatar (539) `4×str + byte whisper`.
- `MapleTVEffect.java` — durations 15 s base, 30 s (tvType 4), 60 s (tvType 5); **reject-if-active**
  (no queue — a Cosmic simplification, contradicted by the client's wait-time strings);
  server-scheduled `removeTV`; **no late-joiner replay**.
- Avatar megaphone: server-scheduled clear at 10 s (redundant with the client timer we verified).
- `PacketCreator.java` — `sendTV` (matches the IDA read order above), `itemMegaphone`
  (mode 8 body: `str msg, byte channel, byte whisper, byte slotPos, [GW_ItemSlotBase]`),
  `getMultiMegaphone` (mode 0x0A: first str, byte count, remaining strs, **channel byte ×10**,
  byte ear, byte 1 — the trailing pad needs IDA confirmation), `getAvatarMega`, `byeAvatarMega`
  (`byte 1`).

### 1.4 Repo state (explored this phase)

- **Handler**: megaphone/TV/avatar types are classified but unhandled — they fall through to
  the warn at `character_cash_item_use.go:110`. The `FieldEffectUse` branch (`:60-106`) is the
  saga template: decode sub-body → 2-step saga (`DestroyAsset` → `FieldEffectWeather`) →
  `saga.NewProcessor(l, ctx).Create(...)` via Kafka `COMMAND_TOPIC_SAGA`.
- **Slot validation**: `character.Processor.GetItemInSlot` (`character/processor.go:208-217`,
  REST to atlas-inventory) already guards the handler (`:37-41`).
- **Saga orchestrator**: `DestroyAsset` is event-driven (completes on the asset DELETED event,
  `consumer/asset/consumer.go:220,260`); `FieldEffectWeather` completes synchronously after
  producing a map command (`handler.go:2853-2880`). **No compensation exists for
  DestroyAsset/FieldEffectWeather** (`compensator.go` reverse-walks only CharacterCreation and
  PetEvolution); flat 30 s default timeout (`libs/atlas-saga/model.go:168`).
- **Broadcast fan-out**: gachapon consumer (`kafka/consumer/gachapon/consumer.go:51-89`) is the
  world-broadcast reference: `LastOffset`, `TenantHeaderParser`, `sc.IsWorld(...)` filter,
  `session.AllInChannelProvider` + `session.Announce`. Channel pods host many
  (tenant, world, channel) listeners from config projection — **multiple pods can host the same
  world**, so world state cannot live in channel memory (`deploy/k8s/base/atlas-channel.yaml:5-10`,
  single Deployment, scalable replicas).
- **WorldMessage writer** (`socket/writer/world_message.go`): all 19 mode keys exist, mode byte
  resolved from the tenant `operations` table via `ResolveCode` — never hard-coded. The packet
  structs are already discrete-per-mode (`libs/atlas-packet/chat/clientbound/world_message.go`),
  but **`WorldMessageItemMegaphone` writes only `bool + int32 slot` (`:159-202`) instead of the
  `GW_ItemSlotBase` block the client reads** — it must be re-cut around `model.Asset`
  (`libs/atlas-packet/model/asset.go`, the shared item-block encoder used by storage/trade/MTS).
  Tests are round-trip-only; **no `packet-audit:verify` markers, no byte fixtures** for any
  WorldMessage mode.
- **Avatar look**: `socket/model/avatar.go NewFromCharacter(c, mega)` →
  `packetmodel.Avatar` (encoded via `Avatar.Encode`, version-aware) — the only look builder;
  reuse it (PRD FR-4.2 satisfied, no duplication).
- **Templates** (`services/atlas-configurations/seed-data/templates/`): WorldMessage writer +
  full `operations` table present in **all five channel templates** (83:`0x44`, 84:`0x44`,
  87:`0x46`, 95:`0x47`, jms:`0x3E`) with identical mode values 0-18 (v83-derived; per-version
  verification pending). **`CharacterCashItemUseHandle` is wired only in gms_83/84** — v87/v95/jms
  templates lack the USE_CASH_ITEM handler entry entirely (opcodes known: `0x052`/`0x055`/`0x047`).
  **gms_12 and gms_92 are login-only stubs** (no channel handlers/writers, no operations tables).
- **Packet-audit state**: no evidence records, audit reports, or ida-export read orders exist for
  any of these families (greenfield). `SERVERMESSAGE`/`OnBroadcastMsg` is **not** a registered
  dispatcher family (no `dispatchers/worldmessage.yaml`, not in `families.yaml`, baseline empty).
  STATUS.md opcodes: SERVERMESSAGE v83/84 `0x044`, v87 `0x046`, v95 `0x047`, jms `0x03E`;
  avatar family v83 `0x06E-0x070`, v84/v87 `0x071-0x073`, v95 `0x072-0x074`, jms SET/CLEAR
  `0x05A/0x05B` (RESULT absent in jms); TV v83 `0x155-0x157`, v84 `0x15F-0x161`, v87
  `0x16A-0x16C`, v95 `0x195-0x197`, jms `0x17A-0x17C`. All ❌ today.
- **Libs audit (per the audit-existing-libs rule)**: `libs/atlas-redis` offers
  `TenantRegistry` (WATCH/CAS `Update`), `TTLRegistry` (`PopExpired`), `TenantKeyedSortedSet`,
  `Lock`; `libs/atlas-lock` offers `LeaderElection` (used by summons/doors/monsters sweeps;
  1–5 s split-brain caveat documented in its doc.go). **No new lib is needed** — the TV queue
  composes from `TenantRegistry` CAS + `LeaderElection`. atlas-world already runs the generic
  ticker pattern (`tasks/task.go`, channel-expiry sweep at `main.go:147`) and owns Redis-backed
  per-world registries, but runs `replicas: 2` **without** leader election today.

---

## 2. Architecture overview

Two broadcast paths, split by whether the family has serialization semantics:

```
                            ┌──────────────────────────────────────────────────────┐
 client ──USE_CASH_ITEM──▶  │ atlas-channel handler                                │
                            │  decode sub-body → validate slot (+ snapshot item /  │
                            │  looks) → [TV/avatar: REST wait-check → maybe reject]│
                            │  → saga: DestroyAsset → emit step                    │
                            └──────────┬───────────────────────────────────────────┘
                                       │ COMMAND_TOPIC_SAGA
                            ┌──────────▼───────────────┐
                            │ atlas-saga-orchestrator  │
                            │  step1 DestroyAsset      │ (event-driven, existing)
                            │  step2a EmitMegaphone ───┼──▶ EVENT_TOPIC_MEGAPHONE ──────────┐
                            │  step2b EnqueueWorldBroadcast ─▶ COMMAND_TOPIC_WORLD_BROADCAST│
                            └──────────────────────────┘                │                   │
                                                            ┌───────────▼──────────┐        │
                                                            │ atlas-world          │        │
                                                            │  broadcast coordinator│       │
                                                            │  (Redis CAS queue,   │        │
                                                            │  leader-gated sweep) │        │
                                                            └───────────┬──────────┘        │
                                                    EVENT_TOPIC_WORLD_BROADCAST_STATUS      │
                                                     (QUEUED/STARTED/ENDED)                  │
                                       ┌─────────────────────┴───────────────┬──────────────┘
                            ┌──────────▼───────────────────────────────────  ▼ ─────────────┐
                            │ atlas-channel consumers (every pod, LastOffset, IsWorld/Is)   │
                            │  megaphone event → WorldMessage mode writers                  │
                            │  STARTED → SEND_TV / SET_AVATAR_MEGAPHONE                     │
                            │  ENDED   → REMOVE_TV / CLEAR_AVATAR_MEGAPHONE                 │
                            │  QUEUED  → ENABLE_TV success ack to the sender's session      │
                            └───────────────────────────────────────────────────────────────┘
```

- **Stateless tiers** (basic/super/item/triple megaphone): instantaneous, overlap freely in real
  GMS → orchestrator emits one broadcast event; channel pods render. No coordinator involvement.
- **Serialized tiers** (Maple TV, avatar megaphone): world-scoped one-at-a-time playback with
  client-corroborated wait caps (1 h TV, 15 s avatar) → a **world broadcast coordinator** in
  atlas-world owns the queue and drives STARTED/ENDED transitions.

---

## 3. Key decisions

### D1 — TV/avatar queue owner: atlas-world (PRD Q2)

**Chosen**: a `broadcast` domain in atlas-world. State per (tenant, world, family) in Redis via
the existing `atlas-redis` `TenantRegistry` (single JSON queue value, optimistic WATCH/CAS
`Update`); queue advance by a 1 s ticker (existing `tasks.Register` pattern) gated by
`atlas-lock` `LeaderElection` (lease `world-broadcast-sweep`), which atlas-world adopts the same
way atlas-summons/doors/monsters already do.

Alternatives considered:
1. **Redis-composed queue advanced by channel pods** (every pod sweeps; `Lock` arbitrates each
   advance). Rejected: N channel pods × T tenants competing for locks every second; the winner
   still has to publish a Kafka event for the other pods, so it saves nothing over a coordinator
   and adds lock churn + split-brain surface in the busiest service.
2. **Saga-orchestrator-owned timers**. Rejected: the orchestrator's timers are per-transaction
   backstops; a 60 s TV slot would hold sagas open and its timer registry is not world-scoped
   state ownership.
3. **New service**. Rejected: YAGNI; atlas-world is the world orchestrator, already owns
   world-scoped Redis registries, tickers, and Kafka status fan-out.

Exactly-once ordering: enqueue commands are keyed by `tenant+world` (single-partition ordering);
every state transition goes through CAS `Update`, so even during a 1–5 s leader split-brain a
double-sweep resolves to one winning transition; only the CAS winner emits events. Residual risk:
a duplicate STARTED/ENDED emission if a leader dies mid-emit after CAS — consumers tolerate this
(re-sending SEND_TV restarts the same message; OnClearAvatarMegaphone is guarded client-side).
Queue drain on pod restart: state is in Redis; the new leader's sweep resumes it.

### D2 — Kafka taxonomy (PRD Q3): one stateless event topic + one coordinator command/status pair

- `EVENT_TOPIC_MEGAPHONE` — `MegaphoneBroadcastEvent` with a `Tier` discriminator
  (`MEGAPHONE|SUPER|ITEM|TRIPLE`), `Scope` (`CHANNEL|WORLD`), world/channel ids, sender
  id/name/medal, message line(s), whisper flag, optional item snapshot. Produced by the
  orchestrator; consumed by every channel pod (`LastOffset`); basic megaphone filtered with
  `sc.Is(t, world, channel)`, the rest with `sc.IsWorld(t, world)`.
- `COMMAND_TOPIC_WORLD_BROADCAST` — `EnqueueCommand` (family `TV|AVATAR`, full render payload,
  duration). Produced by the orchestrator, consumed by atlas-world, keyed `tenant+world`.
- `EVENT_TOPIC_WORLD_BROADCAST_STATUS` — `QUEUED` (characterId, waitSeconds), `STARTED`
  (payload + totalWaitSeconds for SEND_TV), `ENDED`. Produced by atlas-world, consumed by
  channel pods.

One tier-discriminated megaphone event (not per-tier topics) because every consumer does the
same thing and the payload delta is small; the coordinator pair is separate because its
consumers, ordering, and producer differ. All carry tenant headers, `LastOffset`
(fire-and-forget, per PRD §5).

### D3 — Saga shape: consume first, then emit; capacity rejected before consuming

- New `SagaType`: `MegaphoneUse = "megaphone_use"` (all six tiers).
- New `Action`s in `libs/atlas-saga` + orchestrator handlers (both complete synchronously in
  the handler, exactly like `FieldEffectWeather`):
  - `EmitMegaphone` → produce `EVENT_TOPIC_MEGAPHONE`.
  - `EnqueueWorldBroadcast` → produce `COMMAND_TOPIC_WORLD_BROADCAST`.
- Steps: `[DestroyAsset{qty 1}, EmitMegaphone{...}]` or `[DestroyAsset, EnqueueWorldBroadcast]`.
  Consume-fails → broadcast step never runs (PRD FR-2.1). Broadcast-step failure after consume =
  Kafka-produce failure only; no refund, matching the existing `FieldEffectUse` semantics (no
  compensation path exists for DestroyAsset — documented, accepted). No explicit `Timeout` —
  2 steps fit the flat 30 s default (the preset-scaling pattern is for data-driven step counts).
- **TV/avatar wait-cap enforcement happens at decode time in the handler**, before the saga:
  REST `GET` to atlas-world for the queue's current wait; over cap (TV > 3600 s, avatar > 15 s) →
  write the family's rejection packet to the sender (`ENABLE_TV` `01 02`;
  `AVATAR_MEGAPHONE_RESULT` `83`) and return without a saga — item untouched. This mirrors the
  existing decode-time `GetItemInSlot` REST guard. The check-then-enqueue race (two simultaneous
  sends both passing) only overshoots the cap marginally and is accepted; the alternative
  (event-driven enqueue-accept step before DestroyAsset, with dequeue compensation) adds an
  orchestrator await path + compensation for a race that has no user-visible harm.

### D4 — Item megaphone: re-cut `WorldMessageItemMegaphone` around `model.Asset`; snapshot at decode

The existing struct's `int32 slot` body cannot match `CWvsContext::OnBroadcastMsg` mode 8 (the
client reads a `GW_ItemSlotBase` block; Cosmic concurs: `byte slotPos + addItemInfo`). Replace its
body with `mode, message, channel, whisper, byte hasItem/slotPos, [model.Asset block]`
(exact read order IDA-verified per version at implementation), embedding via
`w.WriteByteArray(asset.Encode(l, ctx)(options))` — the `character/clientbound/spawn.go:77`
pattern. The handler resolves the referenced slot via `GetItemInSlot` (empty/mismatched → reject,
no consume, per FR-1.4), converts the channel `asset.Model` into an `AssetSnapshot` DTO carried in
the saga payload and Kafka event, and consumers rebuild `model.Asset` from the snapshot — never
re-resolved (PRD Q6). The channel-side `asset.Model → packetmodel.Asset` mapping used by the
storage/trade writer paths is reused/extracted (exact helper pinned at plan time).

### D5 — Formalize WorldMessage as a dispatcher family; ship 4 arms fully

Per FR-3.3 and the dispatcher false-pass rule:
- Author `docs/packets/dispatchers/worldmessage.yaml` enumerating **all** modes per version from
  each version's `OnBroadcastMsg` switch (IDA), making `operations --check` enforce the five
  templates' tables (today they're unenforced, identical v83-derived copies — per-version drift
  gets caught or corrected here).
- Add `CWvsContext::OnBroadcastMsg#<Mode>` `candidatesFromFName` entries and per-mode body
  functions with fixed operation keys + `WithResolvedCode`, per `DISPATCHER_FAMILY.md`.
- **Arms shipped with full bodies + byte fixtures in this task**: `MEGAPHONE`,
  `SUPER_MEGAPHONE`, `ITEM_MEGAPHONE`, `MULTI_MEGAPHONE` (plus fixtures for arms the codebase
  already emits where the audit requires them for family registration — scoped at plan time).
  The op row stays capped (🧩) until every arm verifies — honest, it is ❌ today.
- Not added to `families.yaml` (FIELD_EFFECT model); never added to the empty lint baseline.

### D6 — Avatar megaphone: coordinator-serialized, all three writers runtime-wired

- 10 s slot duration (client auto-clear constant, IDA-verified v83+v95); 15 s wait cap
  (client string, `SP_3972`). Effectively one active + at most one pending.
- STARTED → `SET_AVATAR_MEGAPHONE` (itemId, decorated name, 4 lines, channel, whisper, look
  snapshot from `socketmodel.NewFromCharacter`). ENDED → `CLEAR_AVATAR_MEGAPHONE` (`byte 1`) —
  belt-and-braces on top of the client's own 10 s timer; the client guard makes it a no-op when
  already cleared. Decode-time rejection → `AVATAR_MEGAPHONE_RESULT` code 83. All three writers
  have real emitters; no orphan codecs.
- No server scheduler beyond the coordinator sweep; FR-4.3 resolved: clear is client-driven,
  server ENDED is a redundancy, both verified.

### D7 — No late-joiner replay (PRD Q5b)

Broadcasts ride `LastOffset` fire-and-forget consumers; a session created mid-TV-message does not
receive the active SEND_TV. Matches Cosmic; nothing in the client requires it; keeps the
session-start path untouched. Documented as matching reference behavior.

### D8 — Durations and caps (PRD Q5a)

TV: 15 s base, 30 s (tvType 4), 60 s (tvType 5) — source: Cosmic `MapleTVEffect.java:56-61`. The
client renders until REMOVE_TV (no client-side display constant found), so these are server
policy values; the client-verified part is the queue/wait mechanics (`m_nTotalWaitTime`,
"longer than an hour" cap). Avatar: 10 s / 15 s cap (IDA, §1.2). `totalWaitTime` in SEND_TV is
computed by the coordinator (remaining active + pending durations), not Cosmic's `1337` filler.

### D9 — Version scope (PRD Q1 + FR-7)

- **Full implementation + byte fixtures**: gms_83, gms_84 (v83 structure + v84 opcode table,
  already reflected in STATUS.md rows), gms_87, gms_95, jms_185. jms ships **no**
  `AVATAR_MEGAPHONE_RESULT` (op absent in jms per STATUS.md — the decode-time avatar rejection
  for jms is scoped at plan time against the jms IDB; if jms has no result packet, rejection is
  silent-return).
- **gms_12: excluded.** Login-only template (24 handlers/42 writers, no channel section); no
  megaphone surface to wire. PRD Q1 answered: exclude.
- **gms_92: excluded — PRD deviation, flagged.** The PRD assumed v92 gets template wiring
  (FR-7.2), but discovery shows `template_gms_92_1.json` is also a **login-only stub** — no
  WorldMessage writer, no USE_CASH_ITEM handler, no operations tables. Wiring six new writers
  into a template that lacks the base channel config is incoherent; bootstrapping the whole v92
  channel template is task-113-adjacent work, not this task. Bannered honestly per FR-7.2's own
  spirit.
- Implementation prerequisite (environment): the IDA host currently has v48/v61/v72/v79/v95.0
  and the v83 dump open — **the v84/v87/jms IDBs must be (re)opened for per-version
  verification**; both v83 instances available today are usable (the v83_Me dump is fully named
  for these families).

### D10 — Medal decoration

Event payloads carry `SenderMedal string`, threaded through `decorateNameForMessage` /
`decorateMegaphoneMessage` exactly like gachapon does today — which passes `""` because
atlas-channel has no medal source (verified: no `Medal` on the channel character model). This
matches "the existing gachapon/notice conventions" the PRD points at; the field is plumbed so a
future medal source drops in without protocol changes.

### D11 — Out of scope, noted

`MAPLE_TV_USE_RES` (`CWvsContext::OnMapleTVUseRes`, v83 `0x06D`) is an adjacent clientbound op
not listed in the PRD and not needed for the Cosmic-parity flow; it stays unimplemented (still ❌
in the matrix, untouched by this task). Cheap/Heart Megaphone (5070000/5073000) have no client
send path per the classifier (type 0) — no arms. Type 15 (507x8xxx) has no item in v83 WZ; per
PRD Q6 ("only ship arms for items that exist") no arm ships unless a supported version's WZ
proves the item at implementation time.

---

## 4. Component design

### 4.1 `libs/atlas-packet` — serverbound (`cash/serverbound`)

New sub-body structs following the `ItemUseFieldEffect(updateTimeFirst bool)` pattern (which
also resolves the v83 trailing-`updateTime` TODO at `character_cash_item_use.go:108` for every
branch — `updateTime` is read before the body on GMS≥95 and after it otherwise):

| Struct | Fields (Cosmic-derived; IDA-verified per version at implementation) |
|---|---|
| `ItemUseMegaphone` | message |
| `ItemUseSuperMegaphone` | message, whisper |
| `ItemUseItemMegaphone` | message, whisper, hasItem, invType(int32), slot(int32) |
| `ItemUseTripleMegaphone` | lineCount(byte, 1–3), lines, whisper |
| `ItemUseMapleTV` | tvType-dependent prefix (byte, ear, partnerName), 5 lines, trailing int32 |
| `ItemUseAvatarMegaphone` | 4 lines, whisper |

Each gets encode+decode, round-trip tests, and serverbound verification per
`VERIFYING_A_PACKET.md` against `CWvsContext::SendConsumeCashItemUseRequest` (+
`CItemSpeakerDlg::_SendConsumeCashItemUseRequest` for the item megaphone) per version.

### 4.2 `libs/atlas-packet` — clientbound

- `chat/clientbound`: re-cut `WorldMessageItemMegaphone` (D4); verify/fixture
  `WorldMessageSimple`(MEGAPHONE arm), `WorldMessageSuperMegaphone`, `WorldMessageMultiMegaphone`
  (Cosmic's trailing `channel×10 + ear + 1` vs the struct's current body reconciled via IDA).
  New: `SetAvatarMegaphone` (itemId, name, 4 lines, channel, whisper, `model.Avatar`),
  `ClearAvatarMegaphone` (byte), `AvatarMegaphoneResult` (code byte + optional string).
- New `tv/clientbound` package (client subsystem `CMapleTVMan`): `TvSetMessage` (flag, type,
  sender `model.Avatar`, names, 5 lines, totalWaitTime, optional receiver `model.Avatar`),
  `TvClearMessage` (empty), `TvSendMessageResult` (hasError byte + code byte).
- All new structs: `packet-audit:fname` comments, byte fixtures with `packet-audit:verify`
  markers per IDB-backed version, `pt.Variants` round-trips.

### 4.3 `atlas-channel`

- **Handler branches** in `CharacterCashItemUseHandleFunc`, keyed classification-first (§1.1
  collisions): decode → (item mega: slot resolve + snapshot; TV/avatar: look snapshots +
  optional partner lookup by name; TV/avatar: REST wait-check → maybe reject) → build
  `MegaphoneUse` saga → `saga.Create`. Decode failure / slot mismatch → warn + return, no
  consume (FR-1.3).
- **Consumers** (gachapon pattern, per hosted `server.Model`, `LastOffset`, tenant headers):
  - `megaphone`: `EVENT_TOPIC_MEGAPHONE` → scope filter → decorate → announce the matching
    WorldMessage mode body to `AllInChannelProvider` sessions.
  - `world_broadcast`: `EVENT_TOPIC_WORLD_BROADCAST_STATUS` → `IsWorld` filter → STARTED/ENDED →
    TV or avatar writers to all sessions; QUEUED → `TvSendMessageResult{00}` to the sender's
    session only.
- **Writers**: register the six new writer names in `produceWriters()` (`main.go:608+`);
  per-mode WorldMessage body funcs with fixed keys + `WithResolvedCode`.
- Missing-character lookups in consumers: log + skip (gachapon precedent).

### 4.4 `libs/atlas-saga` + `atlas-saga-orchestrator`

- `Type` `MegaphoneUse`; `Action`s `EmitMegaphone`, `EnqueueWorldBroadcast`; payload structs
  carrying the fields in D2 (message lines, whisper, scope, item `AssetSnapshot`, look
  `AvatarSnapshot`s, TV type/durations, partner name/look).
- Orchestrator: two handler cases producing to the new topics and calling `StepCompleted(true)`
  synchronously (`handleFieldEffectWeather` precedent, `handler.go:2853-2880`). Mock updates in
  lockstep.

### 4.5 `atlas-world` — broadcast coordinator

- New `broadcast` domain: `TenantRegistry[string, QueueModel]` namespace `world-broadcast`, key
  `<worldId>:<family>`; `QueueModel{Active *Entry, Pending []Entry}`;
  `Entry{Id uuid, CharacterId, Payload, DurationSeconds, ActivatedAt/ExpiresAt}`.
- Consumer: `COMMAND_TOPIC_WORLD_BROADCAST` → CAS-append; if idle, immediate activate → emit
  STARTED (+ QUEUED ack); else emit QUEUED with computed wait.
- Sweep: `tasks.Register` 1 s ticker inside `LeaderElection.Run` (lease
  `world-broadcast-sweep`; atlas-lock is a new dependency for atlas-world, adopting the
  summons/doors/monsters pattern): expired active → CAS pop + promote → emit ENDED then STARTED.
- REST: `GET /api/worlds/{worldId}/broadcast-queues/{family}` → `{active, pendingCount,
  waitSeconds}` (JSON:API resource; used by the channel decode-time cap check).
- Structured logs on every transition (tenant, world, family, characterId, tier fields).

### 4.6 `atlas-configurations` templates + rollout

- Add to gms_83/84/87/95/jms: six writer entries (opcodes from STATUS.md, §1.4), and the
  USE_CASH_ITEM handler entry (`LoggedInValidator`) to v87 (`0x052`), v95 (`0x055`),
  jms (`0x047`). jms omits `AvatarMegaphoneResult`. Every new handler entry carries a validator
  (FR-6.2). WorldMessage operations tables corrected per-version if `worldmessage.yaml`
  verification finds drift.
- Live-tenant runbook (committed to the task folder per FR-6.3): PATCH each live tenant's socket
  config with the new writer/handler entries + operations deltas, then restart atlas-channel
  (projection does not hot-reload handlers/writers); new-opcode-silently-dropped and
  missing-validator pitfalls called out.

---

## 5. Data flows (condensed)

- **Super megaphone**: decode(`ItemUseSuperMegaphone`) → slot guard → saga
  [DestroyAsset, EmitMegaphone{Tier:SUPER, Scope:WORLD}] → orchestrator emits event → every
  channel pod: IsWorld → `WorldMessageSuperMegaphone(mode, "name : msg", ch, whisper)` to all
  sessions. Basic megaphone identical with Scope:CHANNEL + `sc.Is` filter.
- **Item megaphone**: decode → `GetItemInSlot(referenced invType/slot)`; empty/mismatch → warn,
  return → snapshot asset → saga [DestroyAsset(the megaphone), EmitMegaphone{Tier:ITEM,
  ItemSnapshot}] → consumers rebuild `model.Asset` → mode-8 body with embedded item block.
- **Maple TV**: decode(`ItemUseMapleTV`) → partner lookup (by name, same-world; absent →
  self-message) → look snapshots → REST wait-check (>3600 s → `TvSendMessageResult{01 02}` to
  sender, return) → saga [DestroyAsset, EnqueueWorldBroadcast{family:TV, payload, duration by
  tvType}] → coordinator: idle→STARTED / else QUEUED→…→STARTED → pods send SEND_TV with
  coordinator-computed totalWaitTime; on expiry ENDED → REMOVE_TV → next STARTED.
- **Avatar megaphone**: decode(`ItemUseAvatarMegaphone`) → look snapshot → REST wait-check
  (>15 s → `AvatarMegaphoneResult{83}`) → saga [DestroyAsset,
  EnqueueWorldBroadcast{family:AVATAR, duration 10}] → STARTED → SET_AVATAR_MEGAPHONE; client
  auto-clears at 10 s; coordinator ENDED → CLEAR_AVATAR_MEGAPHONE (idempotent no-op if already
  cleared).

## 6. Error handling summary

| Failure | Behavior |
|---|---|
| Sub-body decode error / slot mismatch | warn + return; no consume (FR-1.3) |
| Item-megaphone referenced slot empty/mismatched | reject; no consume, no broadcast (FR-1.4) |
| TV/avatar wait over cap (decode-time REST) | family rejection packet to sender; no saga |
| atlas-world REST check unreachable | warn + reject conservatively (no consume) — never consume-then-drop |
| DestroyAsset step fails (item vanished) | saga fails at step 1; no broadcast (FR-2.1) |
| Emit/Enqueue step fails after consume | no refund — matches FieldEffectUse (no DestroyAsset compensation exists); logged at error with transactionId |
| Coordinator leader failover mid-slot | Redis state survives; new leader resumes; rare duplicate STARTED/ENDED tolerated by client (idempotent packets) |
| Character lookup fails in a consumer | log + skip (gachapon precedent) |

## 7. Multi-tenancy

Tenant headers on every event/command; `tenant.MustFromContext` in all processors; coordinator
keys are `TenantRegistry`-scoped (`<env>:atlas:world-broadcast:<tenantKey>:<worldId>:<family>`);
consumers filter on tenant + world (+ channel for basic megaphone). No cross-tenant state or
broadcast leakage; redis-key-guard stays green because all Redis access goes through
`libs/atlas-redis`.

## 8. Testing & verification

- Unit: sub-body round-trips per `pt.Variants`; byte fixtures with `packet-audit:verify`
  markers for every new/changed clientbound body × every IDB-backed version; coordinator
  queue-transition tests (enqueue/activate/expire/promote, CAS-conflict retry) using the
  project Builder pattern (no `*_testhelpers.go`).
- Packet audit: per-mode/per-op evidence records + audit reports; `worldmessage.yaml`;
  regenerate STATUS.md; `dispatcher-lint`, `matrix --check`, `fname-doc --check`,
  `operations --check` all exit 0.
- Build gates (every changed module/service): `go test -race`, `go vet`, `go build`,
  `docker buildx bake` for atlas-channel, atlas-world, atlas-saga-orchestrator,
  atlas-configurations (+ any lib-touch rebuilds), `tools/redis-key-guard.sh`.
- Live acceptance on a v83 tenant per PRD §10 (each tier + queue behavior across ≥2 channels).

## 9. Resolved PRD open questions

| Q | Resolution |
|---|---|
| Q1 gms_12 | Excluded — login-only template (§D9) |
| Q2 TV queue owner | atlas-world coordinator; existing atlas-redis CAS + atlas-lock; no new lib (§D1) |
| Q3 topic taxonomy | One tier-discriminated megaphone event topic + coordinator command/status pair (§D2) |
| Q4 avatar clear | Client-driven auto-clear at 10 000 ms, IDA-verified v83+v95; server ENDED clear kept as idempotent redundancy (§D6) |
| Q5 TV durations / late joiners | 15/30/60 s (Cosmic-sourced server policy), coordinator-computed wait times; no late-joiner replay (§D7, §D8) |
| Q6 heart/skull & item existence | Arms keyed to the landed classifier: 5070000/5073000 have no send path; 5074000 only GMS≥95 (TV family); type 15 has no item — no arm without WZ proof (§1.1, §D11) |

Additional deviation flagged for review: **gms_92 excluded** (login-only template — PRD FR-7.1/7.2
assumed otherwise); see §D9.

## 10. Risks

1. **Serverbound sub-body shapes are Cosmic-derived until per-version IDA verification** —
   mitigated: verification is a first-class implementation phase; v83/v95 IDBs already open,
   v84/v87/jms must be reopened (environment prerequisite, §D9).
2. **WorldMessage operations tables may be version-wrong today** (identical v83-derived copies
   in all templates) — surfaced deliberately by authoring `worldmessage.yaml` from each
   version's switch; fixes ride this task (operations-table backfill precedent, task-112).
3. **Leader split-brain double-emit** — CAS-gated transitions + idempotent client packets;
   documented residual (§D1).
4. **`MULTI_MEGAPHONE` body divergence** (Cosmic's `channel×10` tail vs current struct) —
   resolved by IDA at implementation; the struct is already isolated per-mode.
5. **jms avatar-result absence** — rejection path per-version gated; verified against the jms
   IDB at implementation.
