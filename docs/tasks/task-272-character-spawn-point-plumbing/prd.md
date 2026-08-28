# Character Spawn Point Plumbing — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-27
Issue: https://github.com/Chronicle20/atlas/issues/1528
---

## 1. Overview

The `spawnPoint` field on the character model is broken at the REST decode seam in every service
that mirrors `atlas-character`'s character model. Eight services define a `Model.SpawnPoint()`
accessor that is a hardcoded `return 0` stub, ignoring the `spawnPoint` field it names. Four of
those eight additionally drop the field entirely in `Extract`, so inbound JSON never reaches the
model. Two of the stubs are consumed on player-facing packet paths, and a third launders the
hardcoded zero back into a JSON:API response that downstream services read as if it were real data.

This task fixes the plumbing: every `Extract` decodes `spawnPoint`, and every stubbed accessor
returns the field it names. It also fixes the sibling defect in `atlas-npc-shops`, whose `Extract`
drops `x` / `y` / `stance`.

**This task does not make the value non-zero.** `atlas-character` is the system of record, and its
`SpawnPoint` column (`services/atlas-character/atlas.com/character/character/entity.go:59`,
`uint32`, `default=0`) is never written: the only mutator, `UpdateSpawnPoint`
(`services/atlas-character/atlas.com/character/character/administrator.go:284`), has zero callers
repo-wide. The persisted spawn point is therefore always `0` at the source. After this task the
field propagates faithfully end to end and the observable wire output is unchanged — that
byte-identical outcome is the primary safety property, not an incidental one. Introducing a
producer that records the portal a character last used is explicitly out of scope and is not being
filed as a follow-up (a deliberate decision, see §2 Non-goals).

Note that live map transitions are unaffected: they already carry a real portal id through
`WarpToMapBody` (`services/atlas-channel/atlas.com/channel/socket/writer/set_field.go:18`). The
broken field is the *persisted* spawn point used in `CHARACTER_DATA` and `CHARACTER_LIST`, a
separate mechanism from in-session warping.

## 2. Goals

Primary goals:

- Every character-model `Extract` in the eight affected services decodes `spawnPoint` from its
  `RestModel`.
- No `Model.SpawnPoint()` accessor is a hardcoded stub; each returns its backing field.
- Resolve the `uint32` (storage/REST) vs `byte` (wire) width mismatch deliberately, with narrowing
  placed at the wire boundary rather than inside the accessor.
- `atlas-query-aggregator` re-serves the real `spawnPoint` value rather than a laundered `0`, at
  full `uint32` fidelity.
- `atlas-npc-shops`' `Extract` populates `x`, `y`, and `stance`.
- Wire output is provably unchanged: with the source column at `0`, `CHARACTER_DATA` and
  `CHARACTER_LIST` encode byte-identically to today.

Non-goals:

- **Introducing a producer for `spawnPoint`.** Nothing will call `UpdateSpawnPoint` as a result of
  this task, and no follow-up task is being filed for it. The always-zero persisted value is
  accepted as current intended behavior. (Explicit user decision.)
- Deduplicating the eight near-identical copies of the character model into a shared library. The
  fix is applied in place, eight times. (Explicit user decision.)
- Fixing the `Rank()` / `RankMove()` / `JobRank()` / `JobRankMove()` `return 0` stubs, which follow
  the same shape in seven of the eight services (see §9). Different field family, different
  question.
- Fixing `atlas-pets`' broader `Extract` gap beyond `spawnPoint` (see §4.3 and §9).
- Any change to `libs/atlas-packet` codecs, opcodes, or version gates. The wire format is correct
  as-is; only the value feeding it is wrong.
- Any change to `atlas-character` itself. Its model, entity, and REST layer already handle
  `spawnPoint` correctly.

## 3. User Stories

- As a service developer, I want `Extract` to decode every field the `RestModel` declares, so that
  I can trust a decoded model rather than grepping to find out which fields silently drop.
- As a service developer, I want an accessor named `SpawnPoint()` to return the spawn point, so
  that reading the call site tells me what the code does.
- As a consumer of `atlas-query-aggregator`'s character endpoint, I want the `spawnPoint` it serves
  to reflect the upstream value, so that I am not consuming a fabricated constant as data.
- As a maintainer, I want the latent copies in `atlas-cashshop` and `atlas-pets` fixed while the
  context is loaded, so that wiring a writer to them later does not silently reintroduce a
  player-facing bug.

## 4. Functional Requirements

### 4.1 Accessor width and narrowing (design decision)

The backing field is `uint32` in all eight services, matching `atlas-character`'s
`Model.SpawnPoint() uint32` (`model.go:137`) and its `uint32` entity column. The wire field is a
single byte (`libs/atlas-packet/character/data.go:41` `SpawnPoint byte`, written via
`w.WriteByte` at `data.go:333`).

**FR-1.** `Model.SpawnPoint()` MUST return `uint32` in all eight services, matching the backing
field and the upstream source of truth. The current `byte` return type MUST NOT be retained.

**FR-2.** Narrowing to `byte` MUST occur at the wire boundary, at each packet call site:

- `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:47` →
  `SpawnPoint: byte(c.SpawnPoint())`
- `services/atlas-login/atlas.com/login/socket/writer/character_list.go:56` →
  `byte(c.SpawnPoint())`

Rationale: narrowing inside the accessor would silently truncate any value above 255 for *every*
consumer, including the non-wire REST path in `atlas-query-aggregator`. Placing the cast at the two
wire call sites keeps the model type-honest and confines truncation to the one layer that genuinely
requires a byte.

**FR-3.** `services/atlas-query-aggregator/atlas.com/query-aggregator/character/rest.go:128` MUST
drop its `uint32(...)` cast and assign `m.SpawnPoint()` directly, preserving full `uint32` fidelity
on the REST re-serve path.

### 4.2 Extract must decode `spawnPoint`

**FR-4.** The following four services drop `spawnPoint` in `Extract` and MUST decode it:

| Service | `Extract` | Style |
|---|---|---|
| `atlas-channel` | `atlas.com/channel/character/rest.go:125` | struct literal |
| `atlas-login` | `atlas.com/login/character/rest.go:129` | builder chain |
| `atlas-cashshop` | `atlas.com/cashshop/character/rest.go:128` | struct literal |
| `atlas-pets` | `atlas.com/pets/character/rest.go:97` | struct literal |

Struct-literal `Extract`s add `spawnPoint: m.SpawnPoint`. The builder-chain `Extract` in
`atlas-login` adds `.SetSpawnPoint(m.SpawnPoint)`; the setter already exists at
`atlas.com/login/character/builder.go:82`.

**FR-5.** The following four services already decode `spawnPoint` correctly and MUST NOT have their
`Extract` changed: `atlas-npc-shops` (`rest.go:160`), `atlas-consumables` (`rest.go:124`),
`atlas-messages` (`rest.go:163`), `atlas-query-aggregator` (`rest.go:166`).

### 4.3 Stubbed accessors

**FR-6.** All eight `SpawnPoint()` stubs MUST be replaced with `return m.spawnPoint`:

| Service | Accessor | `Extract` decodes today? | Consumer |
|---|---|---|---|
| `atlas-channel` | `character/model.go:240` | ❌ | `character_data.go:47` (CHARACTER_DATA) |
| `atlas-login` | `character/model.go:222` | ❌ | `character_list.go:56` (CHARACTER_LIST) |
| `atlas-query-aggregator` | `character/model.go:224` | ✅ | `rest.go:128` (REST re-serve) |
| `atlas-cashshop` | `character/model.go:211` | ❌ | none |
| `atlas-npc-shops` | `character/model.go:208` | ✅ | none |
| `atlas-pets` | `character/model.go:207` | ❌ | none |
| `atlas-consumables` | `character/model.go:213` | ✅ | none |
| `atlas-messages` | `character/model.go:205` | ✅ | none |

Note that `atlas-npc-shops`, `atlas-consumables`, `atlas-messages`, and `atlas-query-aggregator`
decode the field correctly but then mask it with the stub — the accessor is the only break in those
four. `atlas-pets`' `Extract` (`rest.go:97`) currently populates only `id`, `x`, `y`, and `stance`
despite its `Model` carrying the full field set (`model.go:14-44`); this task adds `spawnPoint` to
it and does not address the remaining gap (§9).

### 4.4 `atlas-npc-shops` positional fields

**FR-7.** `Extract` at `services/atlas-npc-shops/atlas.com/npc/character/rest.go:134` MUST populate
`x`, `y`, and `stance` from the `RestModel` fields at `rest.go:40-42`. The `Model` already carries
these fields with real accessors (`model.go:224`, `:228`, `:232`), and `Clone`/`Build` already
round-trip them (`builder.go:39-41`, `:148-150`), so this is a three-line addition to the existing
struct literal.

**FR-8.** `SetX` / `SetY` / `SetStance` MUST NOT be added to the `atlas-npc-shops` builder. `Extract`
is a struct literal and does not need them, and no caller does. (`builder.go` currently has no such
setters.)

### 4.5 Behavioral invariance

**FR-9.** Because the source column is always `0`, `CHARACTER_DATA` and `CHARACTER_LIST` MUST encode
byte-identically before and after this change. Any observed difference indicates an unintended
behavior change and MUST block the branch.

## 5. API Surface

No endpoint is added, removed, or has its route changed. No JSON:API request or response *schema*
changes: every affected `RestModel` already declares `SpawnPoint uint32 \`json:"spawnPoint"\``, and
every `Transform` already writes it out.

One response *value* changes:

- `atlas-query-aggregator` character resource, `spawnPoint` attribute — currently always `0`
  regardless of upstream; after this change reflects the value received from `atlas-character`.
  Given the always-zero source column, the emitted value remains `0` in practice today. Type is
  unchanged (`uint32`).

No error cases are added. `Extract` signatures are unchanged and the added assignments cannot fail.

## 6. Data Model

No new entities, no schema changes, no migration. No table, column, index, or constraint is touched
in any service.

The only type change is in-process: `Model.SpawnPoint()` returns `uint32` instead of `byte` in eight
services. Backing fields (`spawnPoint uint32`) and all `RestModel` field types are unchanged.

Multi-tenancy is unaffected — this is a decode-seam and accessor fix within existing tenant-scoped
paths, and no new storage or query path is introduced.

## 7. Service Impact

| Service | Extract | Accessor | Call site | Other |
|---|---|---|---|---|
| `atlas-channel` | add `spawnPoint` | un-stub, `→uint32` | `byte(...)` at `character_data.go:47` | — |
| `atlas-login` | add `.SetSpawnPoint` | un-stub, `→uint32` | `byte(...)` at `character_list.go:56` | — |
| `atlas-cashshop` | add `spawnPoint` | un-stub, `→uint32` | none | — |
| `atlas-pets` | add `spawnPoint` | un-stub, `→uint32` | none | — |
| `atlas-npc-shops` | — | un-stub, `→uint32` | none | add `x`/`y`/`stance` to `Extract` |
| `atlas-consumables` | — | un-stub, `→uint32` | none | — |
| `atlas-messages` | — | un-stub, `→uint32` | none | — |
| `atlas-query-aggregator` | — | un-stub, `→uint32` | drop cast at `rest.go:128` | — |
| `atlas-character` | **no change** | — | — | system of record, already correct |

`libs/atlas-packet` is not modified.

## 8. Non-Functional Requirements

- **Performance:** no measurable impact. The change adds field assignments to existing decode paths
  and removes constant-return accessors; no new allocation, I/O, or query.
- **Security:** none. No new input is trusted and no authorization boundary moves. `spawnPoint` was
  already accepted on the REST seam — it was simply discarded.
- **Observability:** no new logs, metrics, or spans. No existing log line changes.
- **Multi-tenancy:** unchanged; all touched paths are already tenant-scoped by their callers.
- **Compatibility:** the accessor return-type change (`byte` → `uint32`) is a compile-time break
  for any in-repo caller. There are exactly three callers repo-wide (§4.1, §4.3), all updated in
  this task. `go build ./...` across the monorepo is the guard.
- **Guidelines:** the result must satisfy the backend developer guidelines, in particular the
  CLAUDE.md prohibition on landing a stubbed accessor — which these eight stubs are a live instance
  of.

## 9. Open Questions

None blocking. The two design questions the issue raised are resolved:

- **Width mismatch** — resolved in FR-1/FR-2: accessor returns `uint32`, narrowing at the wire
  boundary. Worth explicit confirmation during `/design-task`, since it is the one judgment call
  here rather than a mechanical edit.
- **`atlas-npc-shops` `x`/`y`/`stance`** — resolved by user decision: populate them (FR-7), do not
  delete.

Observations deliberately left out of scope, recorded so they are not lost:

1. **`Rank()` / `RankMove()` / `JobRank()` / `JobRankMove()` are `return 0` stubs** in seven of the
   eight services (`atlas-channel` `model.go:79-91`, `atlas-cashshop` `:55-67`, `atlas-npc-shops`
   `:52-64`, `atlas-pets` `:51-63`, `atlas-consumables` `:57-69`, `atlas-messages` `:49-61`,
   `atlas-query-aggregator` `:64-76`). `atlas-login` is the exception and carries real rank values,
   consistent with it being the only service that displays them. These are plausibly correct
   omissions rather than defects — the services genuinely do not model ranking — but they share the
   stubbed-accessor shape and deserve their own decision.
2. **`atlas-pets`' `Extract` populates only 4 of ~30 fields** (`rest.go:97`), despite the `Model`
   carrying the full set. Beyond `spawnPoint` this task does not address it. It may be intentional
   (a positional-only view for pet movement) or may be the same defect class at larger scale.
3. **`atlas-messages` has a second stub**, `Stance() byte { return 0 }` at `model.go:225`, even
   though its `Extract` decodes `stance`. Same shape as the `spawnPoint` defect but a different
   field; not covered by the issue or this task's agreed scope.
4. **`UpdateSpawnPoint` is dead code** (`services/atlas-character/atlas.com/character/character/administrator.go:284`,
   zero callers repo-wide). Per the scope decision it is left in place and no producer task is filed.

## 10. Acceptance Criteria

- [ ] All eight `Model.SpawnPoint()` accessors return `m.spawnPoint` and are typed `uint32`; a
      repo-wide grep for `func (m Model) SpawnPoint()` shows no `return 0` body.
- [ ] `Extract` in `atlas-channel`, `atlas-login`, `atlas-cashshop`, and `atlas-pets` decodes
      `spawnPoint`; `Extract` in the other four services is unchanged.
- [ ] `atlas-npc-shops` `Extract` populates `x`, `y`, and `stance`; no `SetX`/`SetY`/`SetStance` was
      added to its builder.
- [ ] `atlas-channel` `character_data.go:47` and `atlas-login` `character_list.go:56` narrow with an
      explicit `byte(...)` cast.
- [ ] `atlas-query-aggregator` `rest.go:128` assigns `m.SpawnPoint()` with no cast.
- [ ] `atlas-character` is untouched by the diff.
- [ ] `libs/atlas-packet` is untouched by the diff.
- [ ] A round-trip test per affected service asserts `Extract(Transform(m)).SpawnPoint() ==
      m.SpawnPoint()` for a non-zero spawn point (proving the seam, independent of the always-zero
      source). Test setup uses the project Builder pattern, not test-only constructors.
- [ ] A round-trip test for `atlas-npc-shops` asserts `x`, `y`, and `stance` survive
      `Extract(Transform(m))` with non-zero values.
- [ ] An encoding test demonstrates `CHARACTER_DATA` and `CHARACTER_LIST` output is byte-identical
      to the pre-change bytes when `spawnPoint` is `0` (FR-9).
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `backend-guidelines-reviewer` reports no new findings on the eight touched services.
- [ ] Code review passes before the PR is opened.
- [ ] The PR closes issue #1528, and its description records that items #1 and #2 are fixed as
      plumbing only (value remains `0`, no producer added) and that item #3 was resolved by
      populating rather than deleting.
