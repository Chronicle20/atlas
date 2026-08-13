# Summon Temporary Stats (PUPPET/SUMMON) Wire Handling — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-07-10
Updated: 2026-08-07 — rebased onto main; evidence and coverage widened from the
original 6-version list to the full supported-tenant set (12 versions).
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
shows it is not a display gap**: no supported client has a PUPPET or SUMMON
SecondaryStat bit, so there is nothing the wire *could* carry.

### 1.1 Evidence sweep (all supported versions)

The original pass (2026-07-10) covered six versions. It was re-run and widened on
**2026-08-07** across every IDB Atlas holds for a supported tenant version — ten live
sessions. Queries: all IDB names matching `CTS_`, all matching `CTS_.*(summon|puppet)`,
and all matching `puppet`.

**Tier A — complete CTS symbol table, direct negative (re-verified 2026-08-07):**

| Version | IDB | `CTS_*` names | Hits for `CTS_.*(summon\|puppet)` | Hits for `puppet` |
|---|---|---|---|---|
| GMS v83 | `MapleStory_dump.exe.i64` | 256 | 2 — both `CTS_SummonBomb` (initializer @0x799C9A, thunk @0x799C95) | 0 |
| GMS v95 | `GMS_v95.0_U_DEVM.exe.i64` | 828 | 6 — all `CTS_SummonBomb` (initializers @0xB0AC20 / @0xB0CE20, `$initializer$` @0xB25C90 / @0xB28B00, data @0xC6D130 / @0xC6DD48) | 1 — `?LetMobChasePuppet@CMobPool@@…` @0x656670, a mob-AI targeting method, **not** a stat |

Both enumerations are complete sets, so these are true negatives, not absent-symbol
guesses. `CTS_SummonBomb` is a mob-skill stat and is unreceivable regardless: it is
`UINT128(1) << 0x80` = bit **128** on v83 and bit **129** on v95 (per the IDA-verified
table in `docs/tasks/task-086-mount-system/v95_secondarystat_table.md`, bits 0–129, zero
collisions), both of which overflow the 128-bit GIVE_BUFF mask (bits 0–127).

**Tier B — no CTS symbol table; absence of any `puppet` symbol (verified 2026-08-07):**

GMS **v48, v61, v72, v79, v84, v87, v92**. These builds carry zero `CTS_*` names, so no
direct enumeration is possible. Each returned **zero** names matching `puppet`. Each does
expose the `SecondaryStat` decoder the registry is modelled on, which is what pins the
bit layout Atlas encodes:

- v61 `SecondaryStat::DecodeForLocal` @0x663665, `::DecodeForRemote` @0x667C5F, `::Reset` @0x662704
- v72 `::DecodeForLocal` @0x6CB87B, `::DecodeForRemote` @0x6CFE78, `::Reset` @0x6CA91A
- v79 `::DecodeForLocal` @0x6FBCBA, `::DecodeForRemote` @0x701539, `::Reset` @0x6FA9BD
- v87 `::DecodeForLocal` @0x7D1EF5, `::DecodeForRemote` @0x7D8533, `::Reset` @0x7D089E
- v92 `::DecodeForLocal` @0x71A9F0, `::DecodeForRemote` @0x711240, `::Reset` @0x70F320
- v84 exposes only `SecondaryStat::DecodeForRemote` @0x7AC409 (local decoder unnamed)
- v48 has **no** named `SecondaryStat` decoder; its three decode sites are identified by
  address in `libs/atlas-packet/model/character_temporary_stat.go` (`legacyGmsMask` doc
  comment): local `CWvsContext::OnTemporaryStatSet` @0x71AF4B → `sub_5CA524`
  `DecodeBuffer(8)` @0x5CA539; reset @0x71B054; foreign `sub_5CBA1F` @0x5CBA33. All three
  test bits 0–46 only.

**Tier C — prior-pass evidence, not re-confirmed on 2026-08-07:**

JMS v185 (`MapleStory_dump_SCY.exe.i64`, session `b6864e54`). The 2026-07-10 pass read
its CTS symbols directly and found no `CTS_*Summon*` / `CTS_*Puppet*` initializers. On
2026-08-07 that session was wedged — `server_health`, `func_query`, and `entity_query`
all timed out — so the finding is carried forward, **not** re-run. This is the one cell
in the sweep resting on prior evidence.

**Tier D — no IDB, bracketed:**

GMS **v28** (< v48, inherits the legacy pre-v61 8-byte-mask path; round-trip-verified
only) and GMS **v86** (between v84 and v87, which agree). Neither has a binary to read.

**Out of scope — GMS v12.** A seed template exists
(`services/atlas-configurations/seed-data/templates/template_gms_12_1.json`) but v12 is
not a packet-matrix column, is not in `libs/atlas-packet/test.Variants`, and its template
declares zero buff writers — it has no CTS surface to be wrong about.

### 1.2 What the evidence supports

Every version with a readable CTS table says the same thing, and no version at any tier
produced a puppet- or summon-named stat symbol. Puppet is a **`CSummoned` display
object** client-side, not a secondary stat — v95's only `puppet` symbol is
`CMobPool::LetMobChasePuppet`, i.e. mob AI reacting to a spawned summon object.

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
- FR-3: `CharacterTemporaryStat.AddStat` (character_temporary_stat.go:580) MUST skip
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

- FR-7: Behavior MUST be identical across **every** supported tenant version — the
  canonical list is `libs/atlas-packet/test.Variants`: GMS **v28, v48, v61, v72, v79,
  v83, v84, v86, v87, v92, v95** and **JMS v185**. Server-only classification is
  version-independent because no client version has these bits (§1.1). No per-version
  registry shift changes of any kind.
- FR-7.1: Coverage MUST span every distinct CTS registry/mask class the packet layer
  builds, not merely a sample of versions. The gates in
  `libs/atlas-packet/model/character_temporary_stat.go` produce **seven** classes:

  | Class | Gate | Versions in `test.Variants` |
  |---|---|---|
  | Legacy 8-byte mask | `legacyGmsMask`: GMS `< 61` | v28, v48 |
  | v61 special | GMS `== 61` | v61 |
  | Mid GMS | GMS `61 < v < 84` | v72, v79, v83 |
  | v84+ | GMS `MajorAtLeast(84)` | v84, v86 |
  | v87+ (post-SoulStone) | GMS `MajorAtLeast(87)` | v87, v92 |
  | v95+ | GMS `MajorAtLeast(95)` | v95 |
  | JMS | `Region() == "JMS"` | JMS v185 |

  Iterating `test.Variants` satisfies this by construction. The legacy class matters
  most: pre-v61 clients receive an **8-byte** mask (`WriteLong(mask.L)`), so any
  assertion written against a 16-byte mask layout is wrong there.

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
- **Multi-tenancy:** classification is tenant-independent; tests MUST exercise **all**
  `test.Variants` versions (FR-7/FR-7.1), matching the idiom of the existing CTS suite
  (`TestCTSForeignEmptyRoundTrip` and siblings already loop `pt.Variants`). Two anchor
  versions are no longer sufficient — they miss the legacy 8-byte-mask class entirely.
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
- [ ] Unit tests in `libs/atlas-packet` prove, **for every version in
      `test.Variants`** (all 12, covering all seven registry classes — FR-7.1):
      (a) PUPPET/SUMMON changes never set a mask bit or write payload bytes;
      (b) a mixed buff (wire stat + PUPPET/SUMMON) encodes byte-identically to the same
      buff without the server-only stats; (c) a pure PUPPET/SUMMON buff yields an
      empty-mask packet, still emitted; (d) an unknown stat name still logs the error
      and is dropped.
- [ ] Byte fixtures confirm existing stats' mask positions/shapes are unchanged across
      the whole existing CTS suite, which already pins legacy (v48/v61), mid-GMS (v83),
      v95, and JMS layouts (FR-8).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
      `libs/atlas-packet` (and atlas-channel if touched); `tools/redis-key-guard.sh`
      clean from repo root.
- [ ] No changes landed in atlas-data/atlas-buffs/atlas-summons statup or lifecycle
      code paths.
