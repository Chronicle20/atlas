# Scripted Items — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-14

---

## 1. Overview

MapleStory's `Consume/0243.img` family contains items whose entire purpose is to open an NPC
conversation when used. The client gates them structurally: `CWvsContext::SendScriptRunItemRequest`
refuses to emit anything unless `itemId / 10000 == 243`. Each such item carries two WZ `spec`
fields — `script` (a script name) and `npc` (the NPC template whose avatar the dialogue renders
with). In Cosmic this is `ScriptedItemHandler.java`, which looks up the item script and runs it.

Atlas already ingests both fields. `services/atlas-data/atlas.com/data/consumable/reader.go:75`
parses `npc`, `:162` parses `spec/script`, and both are exposed over REST
(`consumable/rest.go:74-75`). Every downstream service that mirrors the consumable RestModel —
`atlas-consumables`, `atlas-inventory`, `atlas-npc-shops` — carries the fields through its local
model and then drops them. A grep for `.Script()` across `services/` returns zero consumers. The
data has been sitting there unused.

The wire side is entirely absent: `SCRIPTED_ITEM` / `CWvsContext::SendScriptRunItemRequest` has no
codec in `libs/atlas-packet`, no binding in any of the eleven seed socket-config templates, and is
`❌` in every column of the coverage matrix (`docs/packets/audits/STATUS.md:612`). `atlas-npc-conversations`
has no item-triggered entry point at all — conversations resolve exclusively through
`ByNpcIdProvider` (`conversation/model.go:30`) and, for quests, through the parallel
`conversation/quest/` family.

This task closes the loop: implement the serverbound codec across every client version that
actually has the function, add an item-keyed conversation family to `atlas-npc-conversations`
mirroring the existing quest family, and route the channel handler through a saga so the item is
consumed and the conversation opened as one unit.

---

## 2. Goals

Primary goals:

- Implement the `SCRIPTED_ITEM` serverbound codec in `libs/atlas-packet` across all versions whose
  client contains `CWvsContext::SendScriptRunItemRequest`, and promote the matrix row.
- Correct the coverage matrix, which currently records `n-a` for `gms_v72` and `gms_v79` even though
  both binaries contain the function (see §4.1 — this is a matrix defect, not a scope choice).
- Add an item-triggered conversation entry point to `atlas-npc-conversations`, keyed by item id,
  structurally parallel to the existing `conversation/quest/` family.
- Consume the scripted item up-front in the channel handler, coordinated with conversation start via
  a new saga action, following the `remote_merchant` precedent.
- Seed one or two reference conversations across every applicable version directory to prove the
  path end to end and give a play-testable target.

Non-goals:

- A Cosmic-style JavaScript scripting runtime. Atlas conversations are declarative JSON state
  machines and stay that way.
- Authoring conversation JSON for the full `0243.img` item set. Two reference conversations only;
  the rest is content work for a later pass.
- `NPC_ITEM_USE_REQUEST` / `CWvsContext::SendSelectNpcItemUseRequest` — a second, distinct opcode in
  the same feature neighbourhood. See §9 O-1; this is an explicit open decision, currently out.
- Any change to `atlas-data` WZ parsing. The fields are already parsed correctly.

---

## 3. User Stories

- As a player, I want to use a scripted item from my inventory and have an NPC dialogue open, so that
  the item does what its description says instead of silently doing nothing.
- As a player, I want a scripted item consumed exactly once when I use it, so that a dialogue I close
  or a server hiccup never duplicates or silently eats my item.
- As a content author, I want to attach a conversation to an item id in seed JSON the same way I
  already attach one to a quest, so that I do not have to learn a second authoring model.
- As a server operator, I want scripted items to work identically on every client version we serve
  that supports them, so that version parity does not become a support burden.
- As a maintainer, I want the coverage matrix to reflect what the client binaries actually contain,
  so that `n-a` remains a trustworthy signal.

---

## 4. Functional Requirements

### 4.1 Version scope and the matrix correction

The opcode was resolved directly from each client binary by decompiling
`CWvsContext::SendScriptRunItemRequest` and reading the `COutPacket::COutPacket(&buf, <opcode>)`
constructor argument. `gms_v83` was decompiled first as a control and yielded `0x4E`, matching the
value already in `docs/packets/registry/gms_v83.yaml` — confirming the method before it was applied
to unmapped versions.

| Template | Function present | Opcode | Current matrix | Required |
|---|---|---|---|---|
| `gms_v12` | unverified — no IDB session available | — | `n-a` | See FR-1.1 |
| `gms_v48` | **no** (`93cc947e`) | — | `n-a` | unchanged, `n-a` |
| `gms_v61` | **no** (`415bf585`) | — | `n-a` | unchanged, `n-a` |
| `gms_v72` | **yes** @ `0x9044d8` (`c8acae95`) | **`0x04D` (77)** | `n-a` ❌ **wrong** | implement + verify |
| `gms_v79` | **yes** @ `0x955840` (`1438cecd`) | **`0x04C` (76)** | `n-a` ❌ **wrong** | implement + verify |
| `gms_v83` | yes @ `0xa09b26` (`41f13e0d`) | `0x04E` (78) | `❌` | implement + verify |
| `gms_v84` | yes (registry) | `0x04E` | `❌` | implement + verify |
| `gms_v87` | yes (registry) | `0x051` | `❌` | implement + verify |
| `gms_v92` | yes (registry) | `0x055` | `❌` | implement + verify |
| `gms_v95` | yes (registry) | `0x054` | `❌` | implement + verify |
| `jms_v185` | yes (registry) | `0x046` | `❌` | implement + verify |

**FR-1.1** — `gms_v12` MUST be resolved during design by opening the v12 IDB and querying for
`SendScriptRunItemRequest`. Record the result either way. A negative result MUST be recorded as a
deliberate, evidence-backed `n-a` in the version support doc, not left as an unexamined default.

**FR-1.2** — `gms_v48` and `gms_v61` remain `n-a`. Both binaries expose a dense, fully-symbolised
`Send*ItemUseRequest` family (`SendStatChangeItemUseRequest`, `SendMobSummonItemUseRequest`,
`SendBridleItemUseRequest`, `SendSkillLearnItemUseRequest`, and others) with no
`SendScriptRunItemRequest` among them. Absence in a densely named export set is meaningful evidence,
not an unnamed-symbol artifact.

**FR-1.3** — `gms_v72` and `gms_v79` MUST have registry entries added
(`docs/packets/registry/gms_v72.yaml`, `gms_v79.yaml`) with the opcodes in the table above, and their
version support docs MUST be corrected from `n-a` to a real coverage state.

**FR-1.4** — No wire change may be made to an already-verified op on any version as a side effect of
this work.

### 4.2 Packet codec

The client body is byte-identical across every version inspected (v72, v79, v83):

```
Encode4(get_update_time())   // uint32 update time
Encode2(slot)                // int16  source inventory slot
Encode4(itemId)              // int32  item template id
```

**FR-2.1** — A serverbound codec MUST be added to `libs/atlas-packet` decoding exactly those three
fields in that order, with both `Encode` and `Decode` implemented, following the existing
`inventory/serverbound` item-use codecs as the structural model.

**FR-2.2** — No version-conditional fields are required by current evidence. If design finds a
divergence on a version not yet decompiled (v84, v87, v92, v95, jms_v185), it MUST be gated with the
`MajorAtLeast` idiom, never a raw `> N` comparison.

**FR-2.3** — Each version's opcode MUST be resolved from tenant socket configuration, never
hard-coded (DOM-25).

**FR-2.4** — The handler MUST be bound in the seed socket-config template for every in-scope version.
Template edits MUST satisfy `tools/template-opcode-order-guard.sh` (strictly ascending `opCode`) and
`tools/template-duplicate-binding-guard.sh`.

### 4.3 Client-side gates the server must mirror

**FR-3.1** — The client refuses to send unless `itemId / 10000 == 243`. The server MUST independently
validate that the item id falls in `2430000`–`2439999` and reject anything outside it. A request
outside that range is impossible from a legitimate client and MUST be logged and dropped without
consuming, matching the rejection style of
`character_cash_item_use_remote_merchant.go:87` for an impossible cash slot type.

**FR-3.2** — The server MUST validate that the item actually occupies the claimed slot and that its
template id matches, before consuming.

**FR-3.3** — The handler participates in the excl-request / `EnableActions` contract. v83 wraps the
send in `CanSendExclRequest(500, 0)` and sets `SetExclRequestSent`; v72 and v79 do the same via the
equivalent unnamed helper. Every early-return rejection path MUST unlock the client. Per the
project's excl-request contract, an outcome that warps MUST NOT be unlocked — if a seeded reference
conversation warps the character, the warp path owns the unlock.

**FR-3.4** — v83 additionally guards on `CWvsContext::IsAbleToConsume(itemId, 1)`, which v72 and v79
do not. This is a client-side convenience check only; the server MUST NOT rely on it and MUST perform
its own ownership and quantity validation on all versions (FR-3.2).

### 4.4 Conversation dispatch

**FR-4.1** — A new `conversation/item/` family MUST be added to `atlas-npc-conversations`, keyed by
**item id**, structurally mirroring the existing `conversation/quest/` package (`administrator.go`,
`entity.go`, `model.go`, `processor.go`, `provider.go`, `resource.go`, `rest.go`, `subdomain.go`).

**FR-4.2** — Resolution is by item id. The item's WZ `npc` field selects the avatar the dialogue
renders with and is carried on the conversation start command; it does **not** select the
conversation. The WZ `script` field is recorded for authoring traceability but is not the lookup key.

**FR-4.3** — A new command type MUST be added to `COMMAND_TOPIC_NPC` for starting an item
conversation, carrying at minimum: item id, npc template id, character id, account id, world, channel,
map, instance, and the source inventory slot.

**FR-4.4** — If no conversation is authored for a given scripted item id, the server MUST log at warn
and MUST NOT consume the item, and MUST unlock the client. A missing conversation is a content gap,
not a reason to destroy the player's item.

### 4.5 Consumption and saga

The item is consumed **up-front**, before the conversation opens. This is a deliberate divergence
from Cosmic, where `ScriptedItemHandler` does not consume and the item script calls
`gainItem(id, -1)` itself. Atlas conversations have no equivalent escape hatch today, and the
up-front model matches the established `remote_merchant` shape.

**FR-5.1** — A new saga action MUST be added to `libs/atlas-saga` for starting an item-triggered NPC
conversation. No existing action covers it — the full action list in `libs/atlas-saga/model.go:67-189`
contains `OpenNpcShop` but nothing for conversations.

**FR-5.2** — The saga MUST order the steps so the item is destroyed and the conversation started
together, following the two-step shape of the remote-merchant saga
(`character_cash_item_use_remote_merchant.go:118-150`): `DestroyAssetFromSlot` for the consumed item
and the new conversation-start action.

**FR-5.3** — The saga MUST be compensatable. If conversation start fails after the item is destroyed,
the item MUST be restored. If the item destroy fails, no conversation may open. `compensator.go` MUST
gain a reverse-walk arm for the new action, as it has for `OpenNpcShop` (`compensator.go:1508`).

**FR-5.4** — `event_acceptance.go` MUST declare the accepted event kinds for the new action, and
`character_extractor.go` MUST extract the character id from the new payload type.

**FR-5.5** — The handler MUST be idempotent against Kafka redelivery. A redelivered command MUST NOT
consume the item twice.

### 4.6 Reference content

**FR-6.1** — One or two scripted items MUST be selected as reference implementations, chosen from the
live `0243.img` set (see FR-6.2), and a conversation authored for each.

**FR-6.2** — The concrete `0243.img` item list — every item id in the family with its `script` and
`npc` values — MUST be pulled from live `atlas-data` during design and recorded in the task folder.
No item ids may be asserted from memory or from general MapleStory knowledge. The research note that
seeded this task claims "24 items in `Consume/0243.img.xml`, e.g. 2430000-2430005"; that count and
those ids are **unverified** — no WZ XML exists on the working filesystem
(`find / -name 0243.img.xml` returned nothing) and the WZ corpus lives in MinIO.

**FR-6.3** — Reference conversations MUST be seeded under `deploy/seed/<region>/<version>/` for every
in-scope version, alongside the existing `npc/` and `quests/` directories.

---

## 5. API Surface

New REST resources on `atlas-npc-conversations`, following the JSON:API conventions and the shape of
the existing quest conversation resource (`conversation/quest/resource.go`):

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/item-conversations` | List authored item conversations for the tenant |
| `GET` | `/item-conversations/{itemId}` | Fetch one item's conversation state machine |
| `POST` | `/item-conversations` | Create |
| `PATCH` | `/item-conversations/{itemId}` | Update |
| `DELETE` | `/item-conversations/{itemId}` | Delete |

Resource type name and exact path MUST be confirmed against the existing quest resource's
`GetName()` during design so the two families stay symmetric.

Error cases:

- `404` — no conversation authored for the requested item id.
- `400` — item id outside `2430000`–`2439999`.
- `409` — a conversation already exists for that item id on create.

No changes to `atlas-data`'s REST surface; `script` and `npc` are already exposed.

---

## 6. Data Model

New entity in `atlas-npc-conversations`, mirroring the quest conversation entity:

| Field | Type | Notes |
|---|---|---|
| `tenant_id` | uuid | Tenant scoping, required on every query |
| `item_id` | uint32 | Lookup key; constrained to `2430000`–`2439999` |
| `npc_id` | uint32 | Avatar the dialogue renders with |
| `script_name` | string | WZ `spec/script`, authoring traceability only |
| `states` | jsonb | State machine, identical schema to NPC/quest conversations |

Constraints:

- Unique on `(tenant_id, item_id)`.
- The state machine reuses the existing `StateModel` / `StateType` vocabulary
  (`conversation/model.go:37-49`) unchanged. No new state types are introduced.

Migration: additive only — one new table. No changes to existing conversation tables.

New saga payload type in `libs/atlas-saga` for the conversation-start action, carrying character id,
item id, npc id, world, channel, map, instance, and source slot.

---

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-packet` | New serverbound `SCRIPTED_ITEM` codec (§4.2) |
| `libs/atlas-opcodes` | Op registration if required by the registry conventions |
| `libs/atlas-saga` | New action const + payload type + unmarshal arm |
| `atlas-channel` | New socket handler; conversation-start producer; saga creation |
| `atlas-npc-conversations` | New `conversation/item/` family, REST resource, Kafka command arm |
| `atlas-saga-orchestrator` | Handler arm, compensator arm, event acceptance, character extractor |
| `atlas-consumables` / `atlas-inventory` | Surface `Script` / `Npc` on their consumable models if the validation path needs them |
| `atlas-configurations` | Handler binding in every in-scope seed template |
| `docs/packets/` | Registry entries for v72/v79; support doc corrections; matrix regeneration |
| `deploy/seed/` | Reference conversation JSON per version |

---

## 8. Non-Functional Requirements

- **Multi-tenancy** — every query tenant-scoped via `tenant.MustFromContext(ctx)`; the new table
  carries `tenant_id`; unique constraint is `(tenant_id, item_id)`.
- **Observability** — structured log fields matching the remote-merchant handler's style
  (`character_id`, `item_id`, `npc_template_id`, `transaction_id`) on both the success and every
  rejection path, so a play-test session can be traced end to end in Loki. Loki's service label is
  `service_name`, not `app`.
- **Security** — the server never trusts the client's item id or slot; FR-3.1 through FR-3.4 are the
  validation contract. An out-of-range item id is treated as a protocol violation.
- **Correctness under redelivery** — Kafka is at-least-once; FR-5.5 is not optional.
- **Performance** — no new hot paths. Conversation lookup is a single tenant-scoped indexed read on
  item use.
- **Version parity** — behaviour must be identical across all in-scope versions; the only permitted
  divergence is the opcode value, which is config-resolved.

---

## 9. Open Questions

**O-1 — `NPC_ITEM_USE_REQUEST` is a second, distinct opcode.** The research note treats `spec/script`
and `spec/npc` as one feature. The client disagrees: `CWvsContext::SendSelectNpcItemUseRequest` is a
separate function with its own registry entries and its own matrix row
(`docs/packets/audits/STATUS.md:653` — `❌` on v83 `0x06F`, v84 `0x06F`, v87 `0x072`, v92 `0x07A`,
v95 `0x07B`, jms `0x06A`). Notably `gms_v61` **has** `SendSelectNpcItemUseRequest` at `0x83778d`
while having no `SendScriptRunItemRequest` — so the two ops have genuinely different version spans
and are not interchangeable.

This PRD scopes `NPC_ITEM_USE_REQUEST` **out**. It is a second codec, a second set of eleven template
bindings, and a second matrix row. Recommendation: keep it out of task-230 and file it as a sibling
task once the conversation-dispatch machinery built here exists, since it would reuse the same
`conversation/item/` family and the same saga action. **Requires a decision before design closes.**

**O-2 — `gms_v12`** — unresolved pending FR-1.1.

**O-3 — the `0243.img` item inventory** — unresolved pending FR-6.2. Which two items become the
reference implementations depends on what that query returns.

**O-4 — cancel semantics.** With up-front consumption, a player who opens a scripted item's dialogue
and immediately closes it has spent the item for nothing. This matches the chosen model and is
accepted for now, but play-testing may show it is wrong for specific items. If it does, the fix is to
move consumption into the state machine as a terminal `destroyAsset` action — the design should avoid
foreclosing that option.

**O-5 — v84/v87/v92/v95/jms body confirmation.** The three-field body is verified by decompilation on
v72, v79, and v83 only. The remaining five are assumed identical on registry evidence. Design should
spot-check at least v95 and jms_v185 before the codec is written.

---

## 10. Acceptance Criteria

Packet layer:

- [ ] `SCRIPTED_ITEM` codec exists in `libs/atlas-packet` with both `Encode` and `Decode`, decoding
      `update_time:uint32`, `slot:int16`, `itemId:int32`.
- [ ] Registry entries added for `gms_v72` (`0x04D`) and `gms_v79` (`0x04C`).
- [ ] `gms_v12` resolved with recorded evidence (FR-1.1).
- [ ] `gms_v48` / `gms_v61` remain `n-a` with the evidence in FR-1.2 recorded in their support docs.
- [ ] Handler bound in the seed socket-config template for every in-scope version.
- [ ] Matrix row `SCRIPTED_ITEM` shows no `❌` in any in-scope column, and no residual incorrect
      `n-a` for v72/v79.
- [ ] Byte-fixture verification per `docs/packets/audits/VERIFYING_A_PACKET.md` for each promoted
      cell — a round-trip fixture alone is not a verification.

Behaviour:

- [ ] Using a scripted item with an authored conversation opens the dialogue with the correct NPC
      avatar and consumes exactly one item.
- [ ] Using a scripted item with **no** authored conversation logs a warn, consumes nothing, and
      leaves the client unlocked and responsive.
- [ ] An item id outside `2430000`–`2439999` is rejected, logged, and consumes nothing.
- [ ] A slot/template mismatch is rejected and consumes nothing.
- [ ] Saga compensation restores the item when conversation start fails after destroy.
- [ ] A redelivered command does not double-consume.
- [ ] Every rejection path leaves the client unlocked; no unlock is issued on a path that warps.

Content:

- [ ] The `0243.img` item list with `script` and `npc` values is recorded in the task folder, sourced
      from live `atlas-data` (FR-6.2).
- [ ] One or two reference conversations seeded under every in-scope version directory.
- [ ] Manually play-tested on at least two versions, one of them a legacy version (v72 or v79) since
      those columns are newly claimed.

Verification gates (per `CLAUDE.md`):

- [ ] `go test -race ./...` clean in every changed module.
- [ ] `go vet ./...` clean in every changed module.
- [ ] `go build ./...` clean in every changed service.
- [ ] `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` clean.
- [ ] `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` clean.
- [ ] Code review run before PR.
