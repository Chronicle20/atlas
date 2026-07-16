# Random Reward Items — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Fifty-six v83 Consume items carry a `reward` node in WZ data (e.g. 2022309 Blue Gift Box-style boxes, 2022323/2022324, 2022336): using one consumes the box and grants a prob-weighted random item from its reward table. This is the reference behavior implemented by Cosmic's `ItemRewardHandler.java` (verified against the local checkout at `~/source/Cosmic`).

Atlas has the data layer on both ends but nothing in between. `atlas-data` parses the reward table (`services/atlas-data/atlas.com/data/consumable/reader.go:164-172` — `item`/`count`/`prob` per entry) and `atlas-consumables` already mirrors it into its domain model (`data/consumable/model.go:106` `rewards []RewardModel`). But no socket handler decodes the use request, no opcode is routed, and no consumer arm rolls or grants — using a reward box in-game today is a dead click.

This task implements the flow end-to-end: decode the serverbound packet, validate the box, roll a single weighted pick over the reward table, grant the rolled item (with expiration for period-limited equips), consume the box atomically, honor the reward entry's `Effect`/`worldMsg` presentation fields, and give correct client feedback for the inventory-full case — across all supported GMS tenant versions.

## 2. Goals

Primary goals:
- A player can use any reward-node Consume item and receive a prob-weighted random item from its reward table.
- The box and the granted reward move atomically — no box-consumed-without-reward and no reward-without-box-consumed outcomes.
- The reward entry's `worldMsg` (world-wide announce with `/name` and `/item` substitution), `period` (expiration on the granted item), and `Effect` (use effect shown to the player) are honored. All three fields are parsed by `atlas-data` and carried through `atlas-consumables`.
- Inventory-full is detected **before** the box is consumed; the player is notified and keeps the box.
- The flow works on all supported GMS tenant versions (v83, v84, v87, v92, v95).
- The new serverbound packet codec is byte-fixture verified per `docs/packets/audits/VERIFYING_A_PACKET.md`, promoting the `LOTTERY_ITEM_USE_REQUEST` matrix cells.

Non-goals:
- Gachapon machines (`USE_GACHAPON_BOX_ITEM`, atlas-gachapons) — a separate flow and opcode. The weighted roll stays local to atlas-consumables; per owner decision this feature does not belong in atlas-gachapons.
- Monster catch items (backlog §13) and Solomon/gach-EXP items (backlog §14).
- Scripted items (`SCRIPTED_ITEM` opcode) — different mechanism.
- Any atlas-ui work.
- JMS support is not a v1 requirement. The registry has a jms opcode (`0x06B`), so design should note whether the same body applies, but implementation and fixtures for jms are included only if the design-phase IDA check is cheap and confirms parity (jms IDB availability rotates; see project memory).

## 3. User Stories

- As a player, I want to use a reward box and receive a random item from its reward pool so that opening boxes works as it did on the reference server.
- As a player, I want the box preserved and a clear "inventory full" message when the rolled item cannot fit, so I never lose the box to a full inventory.
- As a player, I want rare wins announced to the world (for reward entries that carry `worldMsg`) with my name and the item name filled in.
- As a player, I want time-limited rewards (entries with `period`) to arrive with the correct expiration set.
- As an operator, I want the roll to be honestly random (CSPRNG, prob-weighted) and observable in logs so reward economics can be audited.

## 4. Functional Requirements

### 4.1 Client protocol (design-phase IDA verification required)

- **Serverbound:** `LOTTERY_ITEM_USE_REQUEST` (fname `CWvsContext::SendLotteryItemUseRequest`) — ❌ unimplemented in every version (STATUS.md:606; v83 `0x070`, v84 `0x070`, v87 `0x073`, v95 `0x07C`, jms `0x06B`). Cosmic registers its handler to the same v83 opcode (`RecvOpcode.USE_ITEM_REWARD(0x70)`), confirming this is the reward-item op. Cosmic's read order is `short slot, int itemId`; whether the real client writes a leading `updateTime` (as sibling item-use packets do) and the exact per-version body MUST be IDA-verified during design — Cosmic read order is a hint, not evidence.
- **Clientbound:** no new writer is expected. Verified existing packet families cover all feedback:
  - Inventory-full: Cosmic sends `SHOW_STATUS_INFO` mode-0 sub-mode `0xFF` (`getShowInventoryStatus`, PacketCreator.java:3573). Atlas's status-message family (`SHOW_STATUS_INFO` / `CWvsContext::OnMessage`, ✅ all versions, STATUS.md:61) is the target; design confirms the exact Atlas writer/mode mapping (beware the v83/v84 status-message operations-table history — see project memory).
  - Enable-actions / unstick: the existing consume-error path (`ConsumeError`, `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:280` emitting `ErrorEventProvider`) already unsticks the client; the success path unsticks via the normal inventory-update flow. Design verifies nothing extra is needed.
  - `worldMsg` announce: Cosmic uses server notice type 6 (light-blue, world-wide). Design maps this to the existing Atlas world-message machinery (`socket/writer/world_message.go` and whatever service-level broadcast command feeds it), including confirming the per-version operations-table entries exist for the chosen mode.
  - `Effect` presentation: design determines the client mechanism — Cosmic loads the field but never uses it server-side, so it may be client-automatic or may require an item-effect packet. If a packet is required, use an existing verified effect writer; the requirement is that the effect visibly plays for the using player where WZ defines one.
- v92 has no IDB. Its opcode/body derive from the template lineage; mark v92 cells accordingly and flag any guesswork explicitly (sibling-task convention).

### 4.2 Request routing

1. New atlas-channel socket handler for `LOTTERY_ITEM_USE_REQUEST` decoding the verified body and emitting a Kafka command (new command type on the consumable command topic, or a sibling topic — design decides, following the `RequestItemConsume` precedent at `services/atlas-channel/atlas.com/channel/consumable/processor.go:28`).
2. atlas-consumables consumes the command and routes to a new `ConsumeReward` item-consumer arm alongside `ConsumeStandard`/`ConsumeTownScroll`/etc. (`consumable/processor.go:184` dispatch). Reward-node presence — not classification — is the routing key on the consumables side; the serverbound opcode already disambiguates intent.

### 4.3 Validation (before any mutation)

On a reward-use request, atlas-consumables must validate:

1. The character owns the claimed item at the claimed slot (existing reservation flow: reserve the box via the compartment reservation machinery the other consume arms use).
2. The item's consumable data actually has a non-empty reward table (a reward request for a non-reward item is dropped with a warn log and reservation cancel).
3. The reward table's total probability is > 0.

A request failing validation cancels the reservation (box untouched) and emits the consume-error event.

### 4.4 Roll semantics (clean weighted pick — deliberate deviation from Cosmic)

1. Roll ONE entry with probability `entry.prob / sum(all probs)`, using CSPRNG (`crypto/rand`) — precedent: `services/atlas-gachapons/atlas.com/gachapons/reward/processor.go:121` `selectTier`.
2. This deviates from Cosmic's algorithm (iterate entries in order, each independently rolling `nextInt(totalprob) < prob`, first win taken — order-biased, and can end with **no** win, leaving the box unconsumed). Owner decision: clean single weighted pick. Consequence: every use yields exactly one reward.
3. **Design-phase check:** sample the 56 v83 reward tables from local WZ data and confirm prob sums are sane as pure weights (i.e. no table where the authored intent is clearly "sum < denominator = chance of nothing"). If any table looks intent-percentage-based, surface it in design.md before implementation — do not silently change item economics.

### 4.5 Grant

1. Grant `count` of the rolled `itemId` to the character via the existing inventory award machinery (atlas-saga-orchestrator already has gachapon award steps — `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/gachapon/` — design picks saga vs. direct inventory command, consistent with how atomicity in 4.7 is achieved).
2. Entries with `period != -1` (equips): the granted item's expiration = now + `period` interpreted per verified semantics. Cosmic computes `period * 60 * 60 * 10` ms with an open "is this a bug?" comment — do NOT copy it. Design verifies the intended unit (expected: hours) against WZ data/other references and documents the decision.
3. Before consuming the box, verify the rolled reward fits the target compartment. If it does not fit: cancel the reservation, send the inventory-full status message, stop. Box preserved.

### 4.6 Consume + presentation

On a successful grant:

1. Commit the box reservation (consume exactly one box).
2. If the rolled entry has `worldMsg`: broadcast the world-wide notice with `/name` → character name and `/item` → item name substitutions applied (actually applied — note Cosmic's `replaceAll` no-op bug; do not reproduce it).
3. If the rolled entry has `Effect`: ensure the effect plays for the player per the design-verified mechanism (4.1).

### 4.7 Atomicity

Consume-box + grant-reward must be atomic in the same sense as the scroll flow: reservation + compensation (saga or buffered-emit pattern; `ConsumeScroll` at `consumable/processor.go:606` is the structural precedent). A failed grant must cancel the reservation; a committed consume must only happen after the grant is assured.

### 4.8 Data layer extension

1. Extend `atlas-data`'s reward parsing (`consumable/reader.go:164-172`) and `RewardRestModel` (`rest.go:125`) with the three per-entry fields Cosmic reads and Atlas drops: `Effect` (string, WZ key `Effect`), `worldMsg` (string), `period` (int, default `-1`). Note these are per-reward-entry fields, distinct from the item-level `effect`/`worldMsg` already parsed at `reader.go:80-82`.
2. Mirror the fields through `atlas-consumables`' `RewardRestModel`/`RewardModel` (`data/consumable/rest.go:169`, `model.go:190`) with getters per the immutable-model pattern.
3. **Rollout note:** atlas-data consumables persist as JSON documents (`document.Storage[string, RestModel]`, `processor.go:14`). Existing tenants' stored documents lack the new fields until re-ingestion / canonical baseline re-publish. The task must document (and if a canonical baseline is in play, perform) the re-publish so live tenants serve the new fields. Missing fields degrade gracefully (empty string / `-1`), never crash.

### 4.9 Configuration & rollout

1. Add the serverbound handler entry (with `LoggedInValidator` — validator-less handler entries are silently dropped) to the tenant seed templates for every supported version.
2. Document the live-tenant config patch + channel restart for existing tenants (seed templates only apply at tenant creation; projection does not hot-reload handlers).
3. If the worldMsg/effect paths use mode-resolved dispatcher packets, confirm the per-version `operations` tables carry the needed modes in every template (missing table entries resolve to 99 and crash the client).

### 4.10 Packet verification

New serverbound codec verified per `docs/packets/audits/VERIFYING_A_PACKET.md`: byte-fixture test with `packet-audit:verify` marker, evidence record, matrix regeneration promoting `LOTTERY_ITEM_USE_REQUEST` cells for every version implemented. v92 cells marked per the no-IDB convention.

## 5. API Surface

No new REST endpoints.

Kafka:
- New command (name/topic per design, e.g. `REQUEST_ITEM_REWARD` on the consumable command topic) carrying tenant headers + world/channel/character context, slot, itemId, and any IDA-verified extras (updateTime). Emitted by atlas-channel, consumed by atlas-consumables.
- Reuse of existing events: consume-error event (unstick), inventory mutation events (grant/consume), world-message broadcast command (worldMsg), saga commands if the saga path is chosen.

REST (read path, existing): `atlas-data` consumables document gains the three per-entry reward fields in its JSON payload; `atlas-consumables` reads them through its existing consumable data client. Backward compatible (additive, omitempty-style defaults).

## 6. Data Model

- No relational migrations. atlas-data consumable storage is a JSON document store; the change is additive fields in the stored `RestModel` (see 4.8 rollout note for re-publish).
- `RewardRestModel` (atlas-data + atlas-consumables mirror): `+ Effect string`, `+ WorldMsg string`, `+ Period int32` (default `-1` = no expiration).
- `RewardModel` (atlas-consumables domain): same three fields, private + getters, Builder-consistent construction.

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-packet` | New serverbound codec for `LOTTERY_ITEM_USE_REQUEST` (per-version body as IDA-verified) + byte fixtures. |
| `atlas-channel` | New socket handler + registration, Kafka command emit; seed-template handler entries for all versions. |
| `atlas-consumables` | New `ConsumeReward` arm: validation, weighted roll (crypto/rand), space pre-check, atomic consume+grant, worldMsg/Effect/period handling; RewardModel field extension. |
| `atlas-data` | Reward-node parsing of `Effect`/`worldMsg`/`period`; RewardRestModel extension; document re-publish note. |
| `atlas-saga-orchestrator` | Only if design picks the saga path for grant atomicity (gachapon-style steps as precedent). |
| `atlas-inventory` | No code change expected (grant via existing commands). |

## 8. Non-Functional Requirements

- **Multi-tenancy:** all Kafka messages carry tenant headers; consumable data reads are tenant-scoped through the existing data client; per-version behavior driven by tenant template config, never hardcoded version branches keyed on `> 83` (use `>= 87` semantics per project convention).
- **Randomness:** CSPRNG (`crypto/rand`), matching the gachapon precedent; no `math/rand`.
- **Observability:** debug-log each roll (characterId, boxId, rolled itemId, prob/total); warn-log validation rejections. No new metrics required.
- **Fail-safety:** any error after reservation cancels the reservation; the client is always unstuck (error event) — never a silently swallowed request leaving the client action-locked.
- **Testing:** unit tests for the weighted pick (distribution + edge cases: single entry, zero-prob entries, total=0), consumer-arm tests per the existing processor_test patterns (Builder-based setup, no `*_testhelpers.go`), byte-fixture tests for the codec.

## 9. Open Questions

1. Exact per-version serverbound body (leading `updateTime`?) — design-phase IDA task.
2. `Effect` delivery mechanism (client-automatic vs. explicit effect packet) — design-phase IDA task.
3. `period` unit (expected hours; Cosmic's math is suspect) — design-phase verification.
4. Prob-sum sanity across the 56 v83 reward tables (any intent-percentage tables where "no win" was the authored behavior?) — design-phase WZ sweep, per 4.4.3.
5. Which grant path (saga vs. direct inventory command + reservation compensation) best satisfies 4.7 — design decision.
6. jms inclusion (registry op exists at `0x06B`; IDB availability rotates) — design-phase call, default out.

## 10. Acceptance Criteria

- [ ] Serverbound `LOTTERY_ITEM_USE_REQUEST` codec implemented for v83/v84/v87/v92/v95 with byte-fixture tests and evidence records; matrix cells promoted (v92 per no-IDB convention).
- [ ] Using a reward-node item in-game consumes exactly one box and grants exactly one prob-weighted reward (single clean weighted pick, crypto/rand).
- [ ] Inventory-full: box preserved, reservation cancelled, inventory-full status message shown, client not stuck.
- [ ] Reward entries with `period` grant items with correct expiration; unit decision documented.
- [ ] Reward entries with `worldMsg` produce a world-wide notice with `/name`/`/item` actually substituted.
- [ ] Reward entries with `Effect` visibly play the effect for the user (mechanism per design).
- [ ] `atlas-data` parses and serves `Effect`/`worldMsg`/`period` per reward entry; re-publish/re-ingestion path for existing tenants documented or performed.
- [ ] Atomicity: no partial outcomes under grant failure (compensation verified by test).
- [ ] Seed templates updated for all supported versions with `LoggedInValidator`; live-tenant patch documented.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` for every touched service; `tools/redis-key-guard.sh` clean.
- [ ] Code review (three-reviewer pattern) run before PR.
