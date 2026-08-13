# Summon Temporary Stats (PUPPET/SUMMON) Wire Handling — Design

Version: v2
Status: Approved for planning
Created: 2026-07-10
Updated: 2026-08-07 — rebased onto main; line references re-verified; version
matrix widened to every supported tenant version (PRD §1.1, FR-7/FR-7.1).
PRD: `docs/tasks/task-164-summon-temp-stats/prd.md`

---

## 1. Problem Recap (context for the plan phase)

atlas-data emits `PUPPET`/`SUMMON` statups for summon-type skills; they flow through
atlas-buffs into atlas-channel's four CTS encode paths (local give, foreign give,
cancel local/foreign, character-spawn foreign block). All four paths delegate to
`CharacterTemporaryStat.AddStat` in
`libs/atlas-packet/model/character_temporary_stat.go:580`, whose registry lookup
fails for these two names and logs
`Attempting to add buff [PUPPET], but cannot find it.` at ERROR.

IDA verification (PRD §1.1) established there is no client-side SecondaryStat bit
for Puppet or Summon on any supported version — a 10-IDB sweep re-run 2026-08-07,
with JMS v185 alone resting on the prior pass (its instance is wedged). These
statups are server-side lifecycle bookkeeping only. The correct behavior is:
classify them as **server-only**, skip them silently at the packet layer, keep
every other stat's wire bytes identical, and keep the ERROR for genuinely unknown
names.

Verified code facts the design relies on (line numbers re-checked against main at
`e0f5bd01d`, 2026-08-07):

- All four writer bodies (`character_buff_give.go:22,37`,
  `character_buff_cancel.go:22,37`, `character_spawn.go:34`) build a fresh
  `CharacterTemporaryStat` and call `AddStat` per change. None consult the registry
  directly. A fix inside `AddStat` therefore covers FR-4 with **zero
  atlas-channel changes**.
- A repo-wide `\.AddStat\(` sweep finds exactly nine call sites. The five above are
  the only `CharacterTemporaryStat` ones; the other four
  (`monster_spawn.go:20`, `kafka/consumer/monster/consumer.go:475,509,543`) are
  `MonsterTemporaryStat.AddStat` — a different receiver with its own registry, and
  explicitly out of scope (PRD §2 non-goals). The skip must therefore live on the
  character receiver only, and cannot leak to mob stats.
- `buildCharacterTemporaryStatRegistry` (`character_temporary_stat.go:63`) is a
  shift-ordered enumeration; adding/removing/reordering entries moves every
  subsequent mask bit. Any design that touches the registry risks FR-8.
- The registry is version-gated at six boundaries — `legacyGmsMask` (GMS `< 61`,
  line 625), GMS `== 61` (line 1010), `MajorAtLeast(84)` (775), `MajorAtLeast(87)`
  (107, 178), `MajorAtLeast(95)` (785, 1030), and `Region() == "JMS"` (177) —
  yielding the seven registry classes PRD FR-7.1 enumerates. Critically,
  `EncodeMask` writes an **8-byte** mask on the legacy class (`WriteLong(mask.L)`,
  line ~672) versus 16 bytes elsewhere, so no test may hard-code a 16-byte-mask
  byte offset if it iterates all versions.
- Writers already emit the packet regardless of how many stats survived `AddStat`
  (they call `NewBuffGive(*cts)` unconditionally), so FR-5's "still emit with empty
  mask" is the existing behavior once the stats are skipped — no writer logic
  changes.
- The file already uses a package-level name-set idiom for stat classification:
  `baseStatNames` (`character_temporary_stat.go:971`).

## 2. Chosen Approach: server-only name set consulted in `AddStat`

Add a package-level set in `libs/atlas-packet/model/character_temporary_stat.go`,
mirroring the existing `baseStatNames` idiom:

```go
// serverOnlyStatNames are temporary stats that exist only for server-side
// lifecycle bookkeeping (Odin lineage). No supported client has a
// SecondaryStat bit for them — IDA-verified across every version Atlas holds
// a binary for (GMS v48/v61/v72/v79/v83/v84/v87/v92/v95, JMS v185), see
// docs/tasks/task-164-summon-temp-stats/prd.md §1.1 — so they are never
// encoded into any CTS mask or payload, on any tenant version.
var serverOnlyStatNames = map[character.TemporaryStatType]bool{
	character.TemporaryStatTypePuppet: true,
	character.TemporaryStatTypeSummon: true,
}
```

`AddStat` checks the set before the registry lookup:

```go
name := character.TemporaryStatType(n)
if serverOnlyStatNames[name] {
	l.Debugf("Skipping server-only temporary stat [%s]; it has no client wire representation.", name)
	return
}
st, err := CharacterTemporaryStatTypeByName(t)(name)
if err != nil {
	l.WithError(err).Errorf("Attempting to add buff [%s], but cannot find it.", name)
	return
}
```

That is the entire production change. Properties, mapped to the PRD:

- **FR-1/FR-7 by construction**: the set is consulted before any tenant-dependent
  lookup, so classification is version-independent.
- **FR-2**: lives entirely in `libs/atlas-packet`; atlas-data/atlas-buffs/
  atlas-summons and all atlas-channel writers untouched.
- **FR-3**: the two names skip at DEBUG (PRD open question resolved: plain
  `Debugf` on every skip — it is cheap, and "first-add-only" would need state for
  no benefit). Unknown names still hit the existing `Errorf` path unchanged.
- **FR-4**: `AddStat` is the single chokepoint for all four encode paths
  (verified above).
- **FR-5/FR-6**: writers emit unconditionally today; a pure PUPPET/SUMMON buff
  yields a `CharacterTemporaryStat` with zero entries, which encodes exactly as
  the current empty CTS (mask from `twoStateBaseStats` only, `nDefenseAtt`/
  `nDefenseState` bytes, base-stat blocks). No format change.
- **FR-8**: `buildCharacterTemporaryStatRegistry` is not touched at all — no
  shift can move. Byte fixtures still prove it (see §5).

## 3. Alternatives Considered

### 3.1 `serverOnly` flag on registry entries (rejected)

Register PUPPET/SUMMON in `buildCharacterTemporaryStatRegistry` as entries with a
`serverOnly: true` field and no shift increment (or a zero mask), then have
`AddStat`/encode loops filter on the flag.

- Pro: classification is visible in the same table as every other stat; lookup
  succeeds so "known but server-only" is expressible.
- Con: the registry builder's whole design is "position in the call sequence ==
  mask bit". Entries that don't consume a shift need a second constructor path
  (`NewCharacterTemporaryStatType` computes `mask = 1 << shift` unconditionally,
  `character_temporary_stat.go:44`), and every consumer of `inOrder`/`byName`
  (EncodeMask, Encode, EncodeForeign, Decode, DecodeForeign) must learn to skip
  them. Five filter sites versus one, all in FR-8-critical code, to express two
  names. Rejected: highest blast radius for zero functional gain.

### 3.2 Classification in `libs/atlas-constants` (`TemporaryStatType.ServerOnly()`) (rejected)

A method on `character.TemporaryStatType` next to the constants.

- Pro: single authoritative home for the domain fact; other services could reuse it.
- Con: PRD FR-2 explicitly places the classification in the packet layer — the
  fact being encoded is "no client wire representation", a packet-layer concern,
  not a domain property (the domain deliberately keeps producing and tracking
  these statups). Exposing it in atlas-constants invites upstream filtering, which
  the PRD forbids (it would disturb summon lifecycle bookkeeping). Rejected on
  PRD grounds; noted here so a future audit doesn't "helpfully" move it.

### 3.3 Writer-side filtering in atlas-channel (rejected)

Filter `c.Type()` in the four writer bodies before calling `AddStat`.

- Con: four copies of the rule, misses any future caller of `AddStat`, and the
  PRD already chose the lib (FR-2, Service Impact §7). Rejected.

## 4. Error Handling & Logging

- Server-only skip: `l.Debugf` per occurrence. Zero ERROR-level output across
  cast, expiry, death, observer spawn (NFR "logging hygiene").
- Unknown stat name: existing `l.WithError(err).Errorf` untouched — the
  acceptance criterion (d) explicitly re-tests it.
- No new error returns; `AddStat`'s signature is unchanged, so no caller churn.

## 5. Testing Strategy

All tests live in `libs/atlas-packet/model/character_temporary_stat_test.go`,
following the existing patterns (`tenant.Create([16]byte{}, region, major, minor)`,
byte-level assertions; e.g. `TestCTSEncodeBuffPerStatLayout`). TDD order: write
each test against current code, watch the PUPPET/SUMMON ones fail (ERROR fires /
lookup fails), then add the set + skip.

**Version matrix (revised v2).** Every new test loops `pt.Variants` — all 12
supported tenant versions, which is exactly the idiom the existing suite already
uses (`TestCTSForeignEmptyRoundTrip`, `TestCTSForeignSingleStatRoundTrip`,
`TestCTSForeignMultiStatRoundTrip`). This satisfies FR-7/FR-7.1 by covering all
seven registry classes, including the legacy 8-byte-mask class (v28, v48) that the
v1 two-anchor matrix missed entirely. The v1 rationale — "FR-7 holds by
construction because the skip precedes tenant-dependent code" — is still true, but
it is an argument, not a test; looping `pt.Variants` costs nothing and turns it
into evidence.

1. **Server-only never reaches the wire** (acceptance (a)): for every
   `pt.Variants` version, `AddStat` with `PUPPET` and `SUMMON` → `Encode` and
   `EncodeForeign` output byte-identical to a freshly-constructed empty CTS.
   Assert via `logrus/hooks/test` that no ERROR-level entry was recorded and that
   both adds logged exactly one DEBUG entry each.
2. **Mixed buff byte-equality** (acceptance (b)): CTS built from
   {Booster, PUPPET, SUMMON} encodes byte-identically (Encode and EncodeForeign)
   to CTS built from {Booster}, on every `pt.Variants` version.
3. **Pure server-only buff still emits** (acceptance (c), FR-5/FR-6): CTS built
   from {PUPPET} alone encodes byte-identically to the empty CTS — i.e. the
   packet body writers would still produce a full, well-formed empty-mask packet.
   (Emission itself is writer-unconditional; the lib test proves the body bytes.)
4. **Unknown name still errors** (acceptance (d)): `AddStat` with name `"BOGUS"`
   → stat absent, and `logrus/hooks/test` captured exactly the existing
   `Errorf` entry. Also looped over `pt.Variants`, since the error path is the
   one behavior that must NOT become version-dependent.
5. **Registry invariance** (FR-8, acceptance criterion 3): existing fixture tests
   must pass unmodified — they pin mask positions and per-stat shapes across the
   legacy class (`TestCTSHomingBeaconV61PopulatedBlock`,
   `TestCTSHomingBeaconV61RoundTrip`, `TestCTSHomingBeaconLegacyVersionsHaveNoTrailer`),
   mid-GMS (`TestCTSEncodeSlowDiseasePerStatLayout`, `TestCTSEncodeBuffPerStatLayout`,
   `TestCTSMonsterRidingV83MaskAndNoDoubleEncode`), and v95
   (`TestCTSMonsterRidingV95MaskAndLayout`, `TestCTSHomingBeaconV95MaskAndBlock`,
   `TestCTSPartyBoosterV95Block`), plus the all-variant round-trip tests. Since the
   change adds no registry entry, any fixture diff is a hard failure.

### 5.1 Determinism: use zero-expiry, not byte-offset skipping

The v1 plan compared two `Encode` outputs and skipped bytes 22..25 to dodge the
wall-clock duration field (`int32(expiresAt − time.Now())`, recomputed per encode
call). That offset assumed a 16-byte mask and is **wrong on the legacy class**,
where the mask is 8 bytes — so it cannot survive the widened matrix.

Replace it with `time.Time{}` (zero expiry) for every stat in these tests.
`legacyDurationUnits` and the modern duration writer both saturate a zero expiry to
a constant (`math.MaxInt16` / `math.MaxInt32` — pinned by the existing
`TestNoExpiryStatEncodesSaturatedDuration` and `TestLegacyDurationUnitsNoExpirySaturates`).
That makes both encodes fully deterministic on every version, so the tests compare
**complete byte slices** with no offset arithmetic and no version-specific casing.
This is strictly stronger than the v1 approach: it stops excusing four bytes from
the comparison.

## 6. Impact & Verification

- Changed module: `libs/atlas-packet` only (one prod file, one test file).
  Expected: no atlas-channel diff, no config/template changes (nothing
  config-resolved changes — DOM-25 not implicated).
- Verification bar (PRD §8): `go test -race ./...`, `go vet ./...`,
  `go build ./...` in `libs/atlas-packet`; then from the worktree root
  `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and
  `tools/lint.sh --check`. The latter two were added to CLAUDE.md's bar after
  this design's v1 and are now mandatory; `tools/lint.sh` (no flags) fixes
  formatting in place — run it before committing. The template guards
  (opcode-order, movement-types) and `service-registration-guard.sh` do **not**
  apply: no template, services.json, or deploy file changes. `docker buildx bake`
  is not required for a lib-only change (no service `go.mod` touched), but if any
  service file does end up modified, bake that service per CLAUDE.md.
- Runtime acceptance (criterion 1) is validated post-merge by casting a puppet
  skill (e.g. Sniper's Puppet) and a summon skill on a v83 tenant and grepping
  atlas-channel logs for the ERROR string across cast, observer spawn-in, expiry,
  and summon death.

## 7. Risks

- **Risk: a future stat gets misfiled as server-only.** Mitigated by the set's
  doc comment requiring IDA evidence and by test (2)'s byte-equality framing —
  adding a name to the set without fixtures will still pass, so review discipline
  (DOM checklist) is the real guard. Low likelihood; the set is two entries with
  a paper trail.
- **Risk: some other code path calls `CharacterTemporaryStatTypeByName` with
  PUPPET/SUMMON and now behaves differently.** It doesn't — the only production
  caller is `AddStat` (`character_temporary_stat.go:535`); decode paths iterate
  `reg.inOrder` directly. And this design changes `AddStat`, not the lookup, so
  `CharacterTemporaryStatTypeByName` semantics are untouched anyway.
- **Risk: empty-mask foreign GIVE_BUFF upsets the v83 client** (PRD open
  question). Unchanged from today's behavior: a buff whose only changes fail
  `AddStat` already emits an empty-mask packet (the ERROR log is the only current
  difference). This design does not alter emission, so it introduces no new
  client exposure; if live testing ever shows a problem it is a pre-existing
  writer-level concern, out of scope per FR-5's owner decision.
