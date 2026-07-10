# Summon Temporary Stats (PUPPET/SUMMON) Wire Handling — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Summon-type skills produce `PUPPET`/`SUMMON` temporary-stat statups in atlas-data's
skill reader (`services/atlas-data/atlas.com/data/skill/reader.go:274,320-327`). These
flow through the buff pipeline (atlas-buffs → buff status events → atlas-channel) into
the packet layer's character-temporary-stat (CTS) encoding. The CTS registry
(`libs/atlas-packet/model/character_temporary_stat.go`) has no Puppet/Summon entries, so
every encode attempt logs `Attempting to add buff [PUPPET], but cannot find it.` at
ERROR level and drops the change. This fires on four paths in atlas-channel:
local buff give, foreign buff give, buff cancel, and the character-spawn buff block
(`services/atlas-channel/atlas.com/channel/socket/writer/character_buff_give.go`,
`character_buff_cancel.go`, `character_spawn.go`).

The backlog item (P11) framed this as a display gap for observers. **IDA verification
(2026-07-10) shows it is not a display gap**: no supported client has a PUPPET or SUMMON
SecondaryStat bit, so there is nothing the wire *could* carry. Evidence:

- **GMS v83** (symbol dump, IDA port 13342): the complete `_dynamic_initializer_for__CTS_*__`
  enumeration contains no `CTS_Puppet` / `CTS_Summon`. The only summon-adjacent stat,
  `CTS_SummonBomb` (a mob-skill stat, initializer @0x799C9A), is `UINT128(1) << 0x80` =
  bit **128**, which overflows the 128-bit GIVE_BUFF mask (bits 0–127) — unreceivable.
- **GMS v95**: the complete IDA-verified table in
  `docs/tasks/task-086-mount-system/v95_secondarystat_table.md` (bits 0–129, zero
  collisions) has no Puppet/Summon; `CTS_SummonBomb` = 129, also overflow.
- **JMS v185** (has CTS symbols, port 13344): no `CTS_*Summon*` / `CTS_*Puppet*` initializers.
- **GMS v87 / v84**: those IDBs carry no CTS symbol names (retail/unnamed binaries), so
  no direct enumeration; both are bracketed by v83 and v95, which agree. v84 is
  structurally identical to v83 (task-083 audit).

Summon visibility for observers is already delivered by the summon object packets
(spawn/attack/remove — task-088, fixtures task-106), not by a buff. `PUPPET`/`SUMMON`
statups are Odin-lineage **server-side lifecycle bookkeeping** (e.g. summon
cancel-on-buff-expiry). The correct product outcome is therefore: classify these two
stats as server-only in the packet layer, skip them silently in all wire-encode paths,
and stop logging errors — while leaving server-side emission and lifecycle behavior
completely untouched.

## 2. Goals

Primary goals:
- Eliminate the `cannot find it` ERROR log spam for `PUPPET`/`SUMMON` on every
  atlas-channel CTS encode path (local give, foreign give, cancel, character spawn).
- Make the exclusion of these stats from the wire an explicit, documented, tested
  decision (server-only classification) instead of an accidental lookup failure.
- Preserve byte-identical wire output for all other stats, on all supported tenants.

Non-goals:
- Adding PUPPET/SUMMON mask bits to the CTS registry (verified: no client reads them).
- Changing summon object visibility (task-088/106 own that; it works via summon packets).
- Changing atlas-data statup production or atlas-buffs lifecycle behavior.
- Touching the monster (mob) temporary-stat registry's equivalent lookup-failure log
  (`libs/atlas-packet/model/monster.go:434`) or `CTS_SummonBomb`/mob-skill stats.
- Auditing other potentially-missing CTS entries beyond PUPPET/SUMMON.

## 3. User Stories

- As a server operator, I want summon/puppet usage to produce zero ERROR-level log
  entries in atlas-channel so that real encode failures aren't buried in noise.
- As a developer reading the CTS registry, I want PUPPET/SUMMON to be explicitly
  modeled as server-only stats so I don't re-investigate this "gap" or mistakenly add
  invented mask bits later.
- As a player observing a summoner, I continue to see their summon/puppet via the
  summon object packets, and their other buffs encode unchanged even when granted by
  the same skill.

## 4. Functional Requirements

### 4.1 Server-only stat classification (packet layer)

- FR-1: `libs/atlas-packet` MUST recognize `character.TemporaryStatTypePuppet` and
  `character.TemporaryStatTypeSummon` as **server-only** temporary stats: valid domain
  values that are never encoded into any CTS wire mask or payload, on any tenant
  version.
- FR-2: The classification MUST live in the packet layer (registry/`AddStat` level),
  not upstream — atlas-data statup production and atlas-buffs behavior are unchanged
  (chosen over upstream filtering to avoid disturbing summon lifecycle tracking).
- FR-3: `CharacterTemporaryStat.AddStat` (character_temporary_stat.go:531) MUST skip
  server-only stats without emitting an ERROR log. A DEBUG-level trace is acceptable;
  the current `Errorf` for these two names must not fire. Unknown (genuinely
  unregistered) stat names MUST continue to log the existing error.

### 4.2 Encode-path coverage

- FR-4: The skip MUST cover every CTS encode path that feeds from buff changes:
  - local buff give (`CharacterBuffGiveBody`)
  - foreign buff give (`CharacterBuffGiveForeignBody`)
  - buff cancel, local and foreign (`character_buff_cancel.go`)
  - character-spawn foreign buff block (`character_spawn.go:34`)
- FR-5: When a buff's change set reduces to zero wire-encodable stats (e.g. a pure
  puppet/summon skill), the writers MUST still emit the buff packet with the remaining
  (empty) mask — emission is NOT skipped (owner decision, 2026-07-10). The packet's
  trailer/format must remain exactly what the current code produces for an empty CTS.
- FR-6: Cancel behaves symmetrically: summon death/expiry cancels that name only
  PUPPET/SUMMON produce a reset packet with those bits absent (empty mask), and no
  error logs.

### 4.3 Tenant coverage

- FR-7: Behavior MUST be identical across all supported tenants (GMS v83, v84, v87,
  v92, v95, JMS): server-only classification is version-independent because no client
  version has these bits. No per-version registry shift changes of any kind.

### 4.4 Wire-format invariance

- FR-8: The mask bit positions and per-stat wire shapes of every existing registered
  stat MUST be provably unchanged (byte fixtures) — this change must not perturb the
  shift-ordered enumeration in `buildCharacterTemporaryStatRegistry`.

## 5. API Surface

None. No REST endpoints, Kafka message schemas, or tenant configuration resources
change. The buff status event payloads continue to carry PUPPET/SUMMON changes
unchanged; only their wire encoding disposition in the packet layer changes.

## 6. Data Model

None. No entities, migrations, or tenant config changes.

## 7. Service Impact

- `libs/atlas-packet` — **primary**. Server-only classification + silent skip in the
  CTS model; unit tests with byte fixtures.
- `services/atlas-channel` — expected **no code change** if the fix lands entirely in
  the lib (its writers already delegate to `AddStat`). If the design chooses a
  writer-side filter instead, changes are confined to the four writer files above.
- `services/atlas-data`, `services/atlas-buffs`, `services/atlas-summons` — explicitly
  untouched.

## 8. Non-Functional Requirements

- **Logging hygiene:** zero ERROR-level CTS logs during normal summon/puppet cast,
  expiry, death, and observer-spawn flows.
- **Wire stability:** byte-fixture tests prove existing stats' encoding is unchanged
  (FR-8); no tenant template or live-config changes required (nothing config-resolved
  changes — DOM-25 not implicated).
- **Multi-tenancy:** classification is tenant-independent; tests should exercise at
  least v83 and v95 registry builds to prove version invariance.
- **Verification:** standard bar — `go test -race ./...`, `go vet ./...`,
  `go build ./...` in changed modules; `tools/redis-key-guard.sh`; `docker buildx bake`
  for any service whose `go.mod` is touched (none expected for a lib-only change).

## 9. Open Questions

- Log level for the intentional skip: fully silent vs. DEBUG trace (recommend DEBUG on
  first-add only if cheap; design decides).
- FR-5 records the owner's "do not skip emission" decision. If design-phase testing
  shows the v83 client reacts badly to an empty-mask foreign GIVE_BUFF (local
  empty-mask is the established enable-actions shape; foreign is less exercised),
  design should surface that with client evidence before deviating.
- Representation choice inside the lib: a `serverOnly` flag on registry entries vs. a
  separate name set consulted by `AddStat`. Design decides; requirement is only that
  unknown-name errors stay intact (FR-3).

## 10. Acceptance Criteria

- [ ] Casting a puppet skill and a summon skill on a v83 tenant produces no
      `Attempting to add buff [PUPPET|SUMMON], but cannot find it.` logs in
      atlas-channel across cast, observer spawn-in, expiry, and summon death.
- [ ] Unit tests in `libs/atlas-packet` prove: (a) PUPPET/SUMMON changes never set a
      mask bit or write payload bytes on any tenant version; (b) a mixed buff (wire
      stat + PUPPET/SUMMON) encodes byte-identically to the same buff without the
      server-only stats; (c) a pure PUPPET/SUMMON buff yields an empty-mask packet,
      still emitted; (d) an unknown stat name still logs the error and is dropped.
- [ ] Byte fixtures confirm existing stats' mask positions/shapes are unchanged on
      v83 and v95 registry builds (FR-8).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
      `libs/atlas-packet` (and atlas-channel if touched); `tools/redis-key-guard.sh`
      clean from repo root.
- [ ] No changes landed in atlas-data/atlas-buffs/atlas-summons statup or lifecycle
      code paths.
