# Ring Pair Field Behavior — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-26

Source issue: [Chronicle20/atlas#1514](https://github.com/Chronicle20/atlas/issues/1514)

---

## 1. Overview

task-240 (PR #1426, branch `task-240-cash-shop-stub-operations`) delivered the *purchase* of a
cash-shop ring pair. It did not deliver the *behavior* of one. `atlas-cashshop` persists a
`cash_rings` row per half — two rows sharing a `PairId` — and emits a `RING_PURCHASED` status
event carrying that `pairId`. Nothing, anywhere, reads it. A tester on `atlas-pr-1426`
(tenant `f3fc852d-555a-45b1-80d8-578ea3b9f401`, GMS client 83.1) bought both a FRIENDSHIP and a
COUPLE pair, saw both halves land in both lockers, and observed no in-game effect and no
indication of who the ring was paired with.

This is a feature gap, not a regression: nothing that shipped is broken. What is missing is the
*read* side. The v83 wire already has the hooks, stubbed out:
`libs/atlas-packet/character/clientbound/spawn.go:174-176` writes three hard-coded zero bytes
labeled `// couple ring`, `// friendship ring`, `// marriage ring`, and
`libs/atlas-packet/character/clientbound/info.go` writes a hard-coded `WriteBool(false)` labeled
`// marriage ring`. Those four literals are the entire surface area of the gap on the wire.

This task fills them. It makes a purchased ring pair observable in the field: the partner's
identity surfaces where the v83 client expects to render it, and the couple/friendship proximity
effect fires because the client has the data it needs to fire it. Ownership of pair state stays
with `atlas-cashshop`; `atlas-channel` becomes a reader.

## 2. Goals

Primary goals:

- Fill the three stubbed ring blocks in `CharacterSpawn` (`spawn.go:174-176`) with real,
  IDA-derived content for every applicable client version, so a spawning character carries their
  ring pair onto the field.
- Fill the stubbed marriage-ring flag in `CharacterInfo` (`info.go`) and whatever partner block
  the v83 client reads behind it, so the character info panel names the partner.
- Give `atlas-channel` a read path to a character's active ring pairs, sourced from
  `atlas-cashshop`'s existing `GET /rings?filter[characterId]` REST surface.
- Link an equipped ring asset to its `cash_rings` half, so the encoder knows *which* equipped
  item is a ring and which pair it belongs to.
- Promote the affected packet coverage-matrix cells to verified with byte fixtures, per
  `docs/packets/audits/VERIFYING_A_PACKET.md`.

Non-goals:

- Wedding and engagement ceremony flow (`CField_Wedding::OnWeddingProgress`,
  `OnWeddingCeremonyEnd`, `NOTIFY_MARRIAGE`, `NOTIFY_MARRIED_PARTNER_MAP_TRANSFER`).
- Marriage rings acquired from NPCs, and the `MARRIAGE_REQUEST` / `MARRIAGE_RESULT` /
  `RING_ACTION` serverbound flow. The marriage-ring *block* is filled on the wire (it is
  structurally adjacent and cannot be left divergent), but no marriage ring is obtainable by this
  task; it is fed by the same `cash_rings` read path and will be empty in practice.
- Spouse chat (`SPOUSE_CHAT`, `CField::OnCoupleMessage`, `CUIStatusBar::SendCoupleMessage`).
- Ring breaking, expiry, trade restrictions, or drop rules. `ring.State` already carries
  `BROKEN`/`EXPIRED`; this task reads `ACTIVE` and ignores the rest.
- The cash-shop purchase flow itself. task-240 owns it.
- Server-side proximity evaluation. See §4.3.

## 3. User Stories

- As a player who bought a couple ring, I want the ring effect to appear when my partner and I
  stand near each other, so that the item I paid for does something.
- As a player who bought a friendship ring, I want the same, so that the friendship ring is
  distinguishable from a plain equip.
- As a player, I want to see who my ring is paired with when I inspect my own character, so I can
  confirm the pairing took.
- As a player, I want to see who *another* player's ring is paired with when I inspect them, so
  the pairing is socially visible.
- As a player, I want the ring's partner name on the item itself in my equip inventory, so I can
  tell two rings apart without inspecting anyone.
- As an operator, I want ring-pair reads to fail soft, so that an `atlas-cashshop` outage
  degrades ring rendering rather than blocking character spawn.

## 4. Functional Requirements

### 4.1 Ring pair read model (`atlas-channel`)

- **FR-1.** `atlas-channel` gains a `ring` package with a read-only processor that fetches a
  character's ring pair halves from `atlas-cashshop` via `GET /rings?filter[characterId]=<id>`,
  the route task-240 registers at
  `services/atlas-cashshop/atlas.com/cashshop/ring/resource.go`. It follows the existing
  channel-side REST-consumer pattern (see `services/atlas-channel/atlas.com/channel/door/rest.go`
  for the shape).
- **FR-2.** The read model exposes, per half: `pairId`, `characterId`, `partnerCharacterId`,
  `assetId`, `itemTemplateId`, `ringType` (`COUPLE` | `FRIENDSHIP`), and `state`.
- **FR-3.** Only halves with `state == ACTIVE` are rendered. `BROKEN` and `EXPIRED` halves are
  fetched and discarded, not filtered server-side, so the filter lives in one place.
- **FR-4.** Ring pair lookups are cached per character for the duration of the character's
  presence on the channel. The cache is invalidated on the `RING_PURCHASED` status event
  (`cashshop2.StatusEventTypeRingPurchased`) for either the buyer or — if resolvable — the
  partner, and on character map/channel transfer.
- **FR-5.** A failed or timed-out ring lookup MUST NOT fail the spawn. The encoder falls back to
  writing the current stub bytes (all-zero ring blocks) and logs at warn level.

### 4.2 Wire encoding

- **FR-6.** `CharacterSpawn` gains ring-pair inputs and encodes the couple, friendship, and
  marriage blocks at `spawn.go:174-176` from them. The exact layout behind each `true` flag MUST
  be derived from the client in design phase (see §9 OQ-1); it is not specified here and MUST NOT
  be assumed from remembered MapleStory behavior.
- **FR-7.** The blocks are version-gated with the `MajorAtLeast` idiom per
  `docs/packets/IMPLEMENTING_A_PACKET.md`. The three flags exist across the whole supported
  range: `spawn.go`'s own v48 comment records "the ring tail is exactly six Decode1 flags
  (miniroom/adboard/couple/friend/marriage/final-effect)". Whether the *body* behind a set flag
  diverges by version is an open derivation.
- **FR-8.** `CharacterInfo` encodes the marriage-ring flag (currently `WriteBool(false)`) and any
  block the client reads behind it from the same read model. The legacy GMS v29..v60 arm of
  `info.go` has no marriage-ring bool at all and MUST remain untouched.
- **FR-9.** No wire change may alter the bytes emitted for an already-verified matrix cell when
  the character has no ring pair. The all-zero path must be byte-identical to today's output, and
  a regression test MUST assert this for every version currently verified for
  `character/clientbound/CharacterSpawn` and `character/clientbound/CharacterInfo`.

### 4.3 Proximity effect

- **FR-10.** Proximity is evaluated **client-side**. The server's obligation is to put correct
  ring data in the spawn packet for every character visible on the map; the v83 client decides
  when two ring wearers are near enough to render the effect.
- **FR-11.** This is contingent on design-phase derivation confirming the client actually does
  this (§9 OQ-2). If derivation shows the client requires a server-sent effect packet, design
  phase MUST surface that as a scope change before planning, not silently add server-side
  proximity ticking.
- **FR-12.** Because the effect rides on spawn data, ring state changes that occur while a
  character is already on a map require a re-spawn or an appearance update to take effect. This
  task does NOT add a mid-map ring update path; a ring purchased while both parties stand on the
  same map is permitted to require a map change before the effect appears. This limitation MUST
  be documented in the service docs.

### 4.4 Equip linkage

- **FR-13.** The encoder must determine which of a character's *equipped* items is a ring half.
  `cash_rings.AssetId` names the cash asset created at purchase; the linkage is
  `equipped asset id → cash_rings row`. Design phase MUST confirm this id is the same identifier
  the channel sees on the equipped-inventory side, and specify the join if it is not.
- **FR-14.** If a character owns a ring pair half but does not have it equipped, the ring blocks
  are not written for them. Owning is not wearing.
- **FR-15.** A character may hold multiple active halves (multiple friendship rings are
  purchasable). Where the wire admits only one, the design MUST specify a deterministic selection
  rule (e.g. lowest equipped slot) rather than an arbitrary one.

### 4.5 Item tooltip

- **FR-16.** The partner's name is available on the ring item in the equip inventory. Design
  phase determines whether this requires any packet work at all — the v83 client may derive it
  from the spawn ring block or from a per-item field — and scopes accordingly. If the client
  renders it with no additional server data, this requirement is satisfied by FR-6 and MUST be
  recorded as such rather than padded with speculative work.

### 4.6 Coverage

- **FR-17.** Every packet × version cell touched by FR-6 and FR-8 is verified with a byte fixture
  carrying a `packet-audit:verify` marker, its evidence record pinned, and the matrix regenerated
  — per `docs/packets/audits/VERIFYING_A_PACKET.md`.
- **FR-18.** A `coverage-manifest.yaml` is written to this task folder declaring every codec and
  gate the task intends to touch, so `packet-completeness-critic` can diff it against the branch.

## 5. API Surface

No new REST endpoints. This task is a consumer of the surface task-240 already registers:

**`GET /rings?filter[characterId]=<uint32>`** — `atlas-cashshop`, JSON:API, paginated,
`filter[characterId]` required. Returns `RestModel`:

```
id, pairId, characterId, partnerCharacterId, assetId, itemTemplateId, ringType, state, createdAt
```

**`GET /rings/{ringId}`** — single half by id. Not expected to be used by this task; listed for
completeness.

Both are tenant-scoped by the standard header middleware. `atlas-channel` consumes them through
its usual `atlas-rest` client with the request context propagated.

### 5.1 Kafka

Consumed, not produced. `atlas-channel`'s existing `handleStatusEventRingPurchased`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:461`) gains cache
invalidation (FR-4). Its doc comment currently records that the partner's half is deliberately not
announced because there is no live session correlation on the event (OQ-R1 from task-240); this
task does not change that decision — it makes the partner's half visible on the *field*, not in
the cash-shop result packet.

No new topics, no new event types.

## 6. Data Model

No new persisted entities. The owning entity ships with task-240:

```
cash_rings
  Id                 uuid  PK
  TenantId           uuid  NOT NULL, index
  PairId             uuid  NOT NULL, index      -- two rows share this
  CharacterId        uint32 NOT NULL, index
  PartnerCharacterId uint32 NOT NULL
  AssetId            uint32 NOT NULL             -- the cash asset in this character's locker
  ItemTemplateId     uint32 NOT NULL
  RingType           string NOT NULL             -- COUPLE | FRIENDSHIP
  State              string NOT NULL             -- ACTIVE | BROKEN | EXPIRED
  CreatedAt          timestamp NOT NULL
```

`atlas-channel` holds an in-memory, per-character read cache only (FR-4). No migration in this
task.

`ring.Type` and `ring.State` are service-local string types in `atlas-cashshop`
(`ring/model.go`), with a documented reason: `libs/atlas-constants`' `ClassificationRing` is an
item *classification*, not a pairing type. If `atlas-channel` needs the same values, design phase
decides between promoting them to `libs/atlas-constants` and re-declaring them channel-side.
Per repository convention, check `libs/atlas-constants/` before defining anything new.

## 7. Service Impact

| Service | Change |
|---|---|
| `libs/atlas-packet` | `character/clientbound/spawn.go` — ring blocks at 174-176 become data-driven; `character/clientbound/info.go` — marriage flag becomes data-driven; new byte fixtures and evidence records for both across affected versions. |
| `atlas-channel` | New read-only `ring` package (REST consumer + per-character cache); spawn/appearance encode sites pass ring data through; `kafka/consumer/cashshop` invalidates the cache on `RING_PURCHASED`. |
| `atlas-cashshop` | None expected. `GET /rings` already exists on the task-240 branch. If design phase finds the response insufficient for the encoder (e.g. it needs the partner's *name*, which the REST model does not carry), a read-side addition here is in scope. |
| `atlas-character` | Possibly none. Touched only if FR-13 shows the equipped-asset id must be joined through the character/inventory service to reach `cash_rings.AssetId`. |
| `atlas-ui` | None. |
| `docs/packets` | Matrix rows for the affected cells; `dispatchers/`, `registry/`, and `evidence/` updated per the verification playbook. |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every ring read is tenant-scoped via the request context. `cash_rings` is
  already `TenantId`-indexed. No cross-tenant pair is representable, and a test MUST assert the
  channel-side cache is keyed by tenant as well as character.
- **Performance.** Spawn is a hot path: every character entering a map triggers a spawn encode
  for every observer. Ring data MUST be resolved from the per-character cache, never with a
  synchronous REST call inside the encode. Cache population happens on character load, not on
  encode.
- **Availability.** `atlas-cashshop` being down degrades ring rendering to the current all-zero
  behavior (FR-5). It never blocks or fails a spawn.
- **Observability.** Log at warn on ring lookup failure with character id and error; log at debug
  on cache population and invalidation. No new metrics required.
- **Security.** Ring data is not sensitive, but partner *character ids* are only ever resolved to
  names server-side — the client is sent names, not internal ids, unless the derived wire format
  requires ids.
- **Wire safety.** FR-9 is the hard constraint: a character without a ring pair must produce
  byte-identical output to today on every currently-verified cell.

## 9. Open Questions

These are genuine external derivations, not deferred work. Each MUST be answered in design phase
from the client binary, per `docs/reverse-engineering.md` and `docs/packets/PROCESS.md`.

- **OQ-1 — What does the client read behind each ring flag?** `spawn.go:174-176` writes three
  zero bytes; nothing in the repo records the layout when the flag is set. The curated v83 export
  (`docs/packets/ida-exports/gms_v83.json`, 791 functions) names nine ring/couple functions, all
  of them cash-shop purchase, wedding-field, or spouse-chat — **none** covers the equip-side ring
  block. Derivation must go through `CUserRemote::Init` (v83 @0x97f55d) directly, likely
  requiring an IDA session rather than the checked-in export.
- **OQ-2 — Does the v83 client evaluate couple-ring proximity itself?** FR-10 assumes yes. If the
  client instead waits on a server-sent effect packet, §4.3 changes shape and the task grows a
  proximity evaluator. This must be settled before planning.
- **OQ-3 — Does `CharacterInfo` carry a partner block behind the marriage flag?** `info.go`'s
  `WriteBool(false)` is followed immediately by the guild name in the current encoder. Whether the
  client reads more when the bool is `true` is unknown.
- **OQ-4 — Is `cash_rings.AssetId` the same identifier the channel sees on an equipped item?**
  (FR-13.) If not, the join must be specified.
- **OQ-5 — Does the ring block need the partner's name or only their character id?** The `GET
  /rings` REST model carries `partnerCharacterId` but not a name; `RingPurchasedBody` carries
  `PartnerName` but the REST read model does not. If the wire needs a name, either the channel
  resolves it or `atlas-cashshop`'s read model gains it.
- **OQ-6 — Which versions beyond v83 are in scope?** The tester's client is GMS 83.1. `spawn.go`
  has distinct legacy arms at `< 61` and `< 83`, plus v84-87, v95+, and JMS paths. Design phase
  decides whether to fill all of them or gate ring content to a derived subset.

## 10. Acceptance Criteria

- [ ] `atlas-channel` has a read-only `ring` package that fetches a character's pair halves from
      `atlas-cashshop` and caches them per tenant + character.
- [ ] `RING_PURCHASED` invalidates that cache.
- [ ] `spawn.go:174-176` no longer contains three hard-coded `WriteByte(0)` ring literals; the
      blocks are driven by ring data, with the layout traced to a cited IDA address.
- [ ] `info.go`'s `WriteBool(false) // marriage ring` is likewise data-driven, with the v29..v60
      legacy arm untouched.
- [ ] A test asserts byte-identical spawn and info output versus current `main` for a character
      with no ring pair, on every currently-verified version.
- [ ] A test asserts non-zero, correctly-shaped ring blocks for a character wearing an `ACTIVE`
      COUPLE half and for one wearing an `ACTIVE` FRIENDSHIP half.
- [ ] `BROKEN` and `EXPIRED` halves produce no ring block.
- [ ] An owned-but-not-equipped half produces no ring block.
- [ ] An `atlas-cashshop` failure produces the all-zero fallback and a warn log, not a spawn error.
- [ ] Every touched packet × version cell is verified with a `packet-audit:verify` fixture, its
      evidence record pinned, and `docs/packets/audits/STATUS.md` regenerated.
- [ ] `coverage-manifest.yaml` exists in this task folder and `packet-completeness-critic` reports
      no CHANGED-BUT-UNCLAIMED or CLAIMED-BUT-UNVERIFIED findings.
- [ ] Service docs for `atlas-channel` record the FR-12 limitation (ring changes take effect on
      next spawn).
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Live-tested on GMS client 83.1: two characters wearing paired couple rings stand adjacent
      and the effect renders; each can see the other's partner name in the character info panel
      and on the ring item in the equip inventory.

## 11. Dependency

**This task depends on task-240 (PR #1426) landing on `main`.** The `cash_rings` entity, the
`ring` package, the `GET /rings` route, and the `RING_PURCHASED` event all live on branch
`task-240-cash-shop-stub-operations` and exist nowhere on `main` — a repo-wide grep for `pairId`
on `main` returns only `atlas-doors`' unrelated door pairing. This worktree branches from `main`,
so the ring consumer cannot build until #1426 merges. Design phase may proceed against the
task-240 branch as a reference; planning and execution MUST NOT begin until `main` carries the
ring package.
