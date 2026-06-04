# Character Render — Default Clothing Fallback — Design

Task: task-078-render-default-clothing
Status: Draft (design phase)
PRD: `docs/tasks/task-078-render-default-clothing/prd.md`
Date: 2026-06-03

---

## 1. Summary

`atlas-renders` composites a character PNG from body skin + head + hair/face +
equipped items. When no `Coat`/`Pants`/`Longcoat` is equipped, the compositor
skips those slots and renders a bare body. This task injects **gender-specific
beginner clothing** into any empty clothing slot so a character is never nude,
threads a new optional `gender` query parameter through the render contract, and
folds the resolved gender into the loadout hash on both the Go service and the
atlas-ui producer so the cache stays correct.

The change is deliberately small and additive. It reuses the existing
equipment-injection path end-to-end: a default item is dropped into the same
`slot→id` map an equipped item would occupy, so it flows through `partClassFor`,
`fetchAtlas`, joint solving, vslot occlusion, and z-ordering with **zero**
special-casing in the compositor's hot loop.

## 2. Scope and boundaries

In scope:
- `atlas-renders` (Go): query parsing, gender resolution, default injection,
  canonical-hash extension, handler hash recomputation, tests.
- `atlas-ui` (TypeScript): loadout/gender plumbing, canonical-string mirror,
  `gender` query param emission, the three inline loadout builders in
  `useCharacterImage.ts`, and the cross-language hash fixture.

Out of scope (per PRD non-goals): no new underwear sprite, no bare-body opt-out,
no character-creation/inventory/storage changes, no cache purge/migration, no
nginx route change (the route regex keys on the 16-char hash and the query
string is forwarded verbatim via `$is_args$args` — `gender` is a query param,
not a path component, so `deploy/shared/routes.conf:214` is untouched).

## 3. Component boundaries

The feature decomposes into four tightly-scoped units, each independently
testable:

| Unit | File | Responsibility | Depends on |
|------|------|----------------|------------|
| Gender resolution | `character/gender.go` (new) | Map `(genderParam, face) → 0\|1`; own the default-clothing constants and `defaultCoat`/`defaultPants` lookups | nothing (pure) |
| Default injection | `character/composite.go` | After the equipment slot-map is built, fill empty clothing slots per FR-2 | `gender.go`, `libs/atlas-constants/item` |
| Query parsing | `character/query.go` | Parse optional `gender` into a sentinel-bearing `RenderQuery.Gender` | nothing |
| Canonical hash | `character/hash.go` + `character/handler.go` | Append resolved gender to the canonical string; handler resolves once for the URL-hash check | `gender.go` |

The TS side mirrors units 1, 3, and 4 inside
`characterRender.service.ts` (a single module), with the inline loadout builders
in `useCharacterImage.ts` updated to pass `gender` through.

A new file `gender.go` (rather than appending to `composite.go`) keeps the
gender/inference/constants concern isolated and easy to point a unit test at; it
mirrors how the TS logic is a discrete set of helpers.

## 4. Key design decisions

### 4.1 Gender resolution is a single pure, idempotent function

```go
// gender.go
const (
    GenderMale   = 0
    GenderFemale = 1
    GenderUnspecified = -1 // RenderQuery sentinel: param absent

    DefaultCoatMale    = 1040036
    DefaultPantsMale   = 1060026
    DefaultCoatFemale  = 1041046
    DefaultPantsFemale = 1061039
)

// ResolveGender maps an optional gender selector plus a face id to a concrete
// 0 (male) / 1 (female) value. Precedence: an explicit 0/1 wins; otherwise the
// v83 face convention (faceId/1000)%10 == 1 ⇒ female; anything else ⇒ male.
func ResolveGender(genderParam, face int) int {
    if genderParam == GenderMale || genderParam == GenderFemale {
        return genderParam
    }
    if face > 0 && (face/1000)%10 == 1 {
        return GenderFemale
    }
    return GenderMale
}
```

**Idempotence is the linchpin.** `ResolveGender(0, face) == 0` and
`ResolveGender(1, face) == 1` for any face. This lets us resolve gender once in
the handler (for the hash) and *again* in `Composite` (for injection) and be
guaranteed the same answer without threading a resolved value through the call
graph. `Composite` stays callable in isolation in tests (set `q.Gender`
directly), and the handler doesn't need a new out-parameter on `Composite`.

Hair is deliberately **not** consulted (PRD FR-3): later female hair ranges break
the clean modulo; face (`200xx` male / `210xx` female per the char-creation
template) is stable.

**Alternative considered — resolve once, thread the value into `Composite`:**
add a `resolvedGender int` parameter to `Composite`. Rejected: it widens an
already-long signature and creates two ways to learn the gender (param vs.
threaded), inviting drift. Idempotent re-resolution is cheaper to reason about
and is O(1).

### 4.2 `RenderQuery.Gender` uses a `-1` sentinel for "unspecified"

`query.go` gains:

```go
type RenderQuery struct {
    Skin, Hair, Face int
    Stance           string
    Frame, Resize    int
    Items            []int
    Gender           int // 0, 1, or GenderUnspecified(-1) when absent
}
```

Parsing rules (FR-4):
- `gender` absent → `GenderUnspecified`.
- `gender` == `"0"`/`"1"` → `0`/`1`.
- `gender` present but anything else → `error` (handler maps to `400
  invalid-input`, consistent with the existing `ParseRenderQuery` malformed-input
  path).

**Alternative considered — `Gender *int`:** a nil pointer is the "natural" Go
optional. Rejected: every existing `RenderQuery` field is a value, the struct is
passed by value through `Composite`, and a sentinel int keeps the canonical-hash
arithmetic and test construction trivial (no `&x` dance). `-1` is unambiguous
because the only valid wire values are `0`/`1`.

### 4.3 Default injection: a small helper called after the slot-map is built

In `Composite`, immediately after:

```go
equipment := FilterEquipment(ItemsToSlotMap(q.Items))
```

insert:

```go
applyDefaultClothing(equipment, ResolveGender(q.Gender, q.Face))
```

with:

```go
// applyDefaultClothing fills empty clothing slots with the gender's beginner
// coat/pants so a character is never rendered bare. An equipped Overall in the
// top slot (-5) covers both halves and suppresses both defaults.
func applyDefaultClothing(equipment map[int]int, gender int) {
    if id, ok := equipment[topSlot]; ok &&
        item.GetClassification(item.Id(uint32(id))) == item.ClassificationOverall {
        return // overall covers top + bottom
    }
    if _, ok := equipment[topSlot]; !ok {
        equipment[topSlot] = defaultCoat(gender)
    }
    if _, ok := equipment[bottomSlot]; !ok {
        equipment[bottomSlot] = defaultPants(gender)
    }
}
```

where `topSlot = -5`, `bottomSlot = -6` (named constants, mirroring the existing
`slotForItemID` numbering), and `defaultCoat`/`defaultPants` are trivial
gender switches over the constants in §4.1.

This satisfies FR-2's independence exactly:
- Overall in `-5` → both checks suppressed (`return`).
- Real top in `-5`, empty `-6` → coat skipped (slot occupied), pants injected.
- Empty `-5`, real bottom in `-6` → coat injected, pants skipped.
- Both empty → both injected.

The injected id is an ordinary map entry, so step 4 of `Composite` (the
equipment loop) fetches its atlas via `partClassFor` → `Coat`/`Pants` →
`fetchAtlas` and composites it like any equip. No change to joint solving,
vslot occlusion, or z-order. **This is the whole point of injecting into the
slot map rather than adding a parallel "defaults" list.**

Note on slot semantics: `slotForItemID` already maps both `ClassificationTop`
and `ClassificationOverall` to `-5`, so "does an overall occupy the top slot" is
answered by classifying whatever sits in `-5` — no separate overall slot exists.

### 4.4 Canonical hash gains a trailing resolved-gender field

`CanonicalLoadoutString` gains a final `gender int` parameter and appends it last:

```
tenant|region|maj.min|skin|hair|face|stance|frame|resize|items|gender
```

Example: `tenant-a|GMS|83.1|0|30030|20000|stand1|0|2|` becomes
`tenant-a|GMS|83.1|0|30030|20000|stand1|0|2||1` for a resolved-female empty
loadout (note the empty items field then `|1`).

The handler resolves gender once and feeds the **resolved** value (never the raw
`-1` sentinel) into the canonical string:

```go
g := ResolveGender(q.Gender, q.Face)
canonical := CanonicalLoadoutString(
    urlTenant, urlRegion, t.MajorVersion(), t.MinorVersion(),
    q.Skin, q.Hair, q.Face, q.Stance, q.Frame, q.Resize, q.Items, g,
)
```

Appending at the **end** (vs. inserting next to skin/hair/face) means the field
ordering of every pre-gender position is unchanged, which keeps the diff minimal
and the format easy to audit against the TS mirror. Every existing hash changes
regardless of insertion position (PRD §6 lazy-regeneration is accepted), so the
choice is purely about diff legibility — end-append wins.

**Determinism contract (NFR):** the producer (atlas-ui) always emits the
**resolved** gender both as the trailing canonical field *and* as the `gender`
query param. The service therefore receives `gender=0|1` explicitly, and
`ResolveGender(0|1, face)` returns it verbatim — UI and service canonical
strings are byte-identical by construction. Direct callers that omit `gender`
get face inference on the service side; their URL hash must have been computed
the same way (this is the existing "callers must compute the hash correctly"
contract, now with one more field).

## 5. TypeScript mirror (atlas-ui)

`src/services/api/characterRender.service.ts`:

- `CharacterLoadout` gains `gender?: number`.
- New `resolveGender(gender: number | undefined, face: number): 0 | 1`
  mirroring §4.1 byte-for-byte (absent/`undefined` → infer from face;
  `(face/1000)%10 === 1` ⇒ 1; else 0; non-positive face ⇒ 0).
- `canonicalLoadoutString(...)` gains a trailing `gender: number` argument
  appended as the final `|`-joined field (mirrors §4.4).
- `generateCharacterUrl` resolves gender from `loadout.gender` (→ face
  inference fallback), appends it to the canonical string, and adds
  `gender: String(resolved)` to the `URLSearchParams`.
- `characterToLoadout` populates `gender` from `character.attributes.gender`.

`src/lib/hooks/useCharacterImage.ts` has **three** inline loadout/canonical
builders that must each pass gender to keep the React-Query key hash equal to the
URL hash:
1. `generateQueryKey` → its `canonicalLoadoutString(...)` call.
2. The main `queryFn` → its `generateCharacterUrl(...)` loadout object.
3. `prefetchVariants` / `preloadImages` / `warmCache` → their
   `generateCharacterUrl(...)` loadout objects.

`MapleStoryCharacterData` already carries `gender` (`maplestory.ts:106`), so
every call site has the value in hand — the work is purely passing it through.
`character-cache-sw.ts` keys off the produced URL string (not a recomputed
hash), so it needs no logic change; it benefits automatically from the new URLs.

**Cross-language parity fixture.** `__tests__/loadout-hashes.json` (consumed by
`characterRender.service.test.ts`) gains a `gender` field on every row and its
`canonical`/`expectedHash` values are regenerated to include the trailing field.
To harden the Go↔TS determinism guarantee that this task most depends on, the Go
`handler_test.go` gets matching gender cases whose canonical strings and hashes
are computed the same way (a male-face vs. female-face empty loadout producing
**different** hashes — directly covering the PRD's "no collision" acceptance
criterion). We keep the two suites hand-mirrored rather than introducing a
cross-module file read (the fixture lives under the UI tree; a Go test reaching
across module boundaries is more fragile than two small, explicit case tables).

## 6. Data flow

```
UI (characterToLoadout) ── gender from character.attributes.gender
   │
   ├─ resolveGender(gender, face) ─► resolved 0|1
   ├─ canonicalLoadoutString(..., resolved) ─► hash (16 hex)
   └─ generateCharacterUrl ─► /api/assets/.../<hash>.png?...&gender=<resolved>
        │
        ▼ (nginx forwards query verbatim via $is_args$args)
atlas-renders Handler
   ├─ ParseRenderQuery ─► q.Gender ∈ {0,1,-1}
   ├─ g = ResolveGender(q.Gender, q.Face)
   ├─ CanonicalLoadoutString(..., g) ─► expected hash; compare to URL hash
   ├─ cache hit? stream PNG
   └─ cache miss ─► Composite
        ├─ equipment = FilterEquipment(ItemsToSlotMap(q.Items))
        ├─ applyDefaultClothing(equipment, ResolveGender(q.Gender, q.Face))
        └─ existing compositing path (atlas fetch, joints, vslot, z-order, blit)
```

## 7. Error handling and degradation

- `gender` present but not `0`/`1` → `400 invalid-input` (FR-4), surfaced from
  `ParseRenderQuery` through the existing handler error envelope.
- URL hash mismatch (now also a function of resolved gender) → `400
  hash-mismatch`, unchanged mechanism.
- **Missing default atlas (FR-7):** a default id is injected exactly like an
  equip, so the existing step-4 path applies: `fetchAtlas` fails →
  `l.Warnf("missing atlas: partClass=%s id=%d (skipping)")` → the slot renders
  bare. This is the same warn-and-skip behavior as any missing equip — no new
  error path, no hard failure. It is also the failure mode the §9 asset
  verification (below) exists to catch before release.

## 8. Asset verification (PRD §9 — the one release-blocking item)

The feature is only *visibly* effective if the four default atlases are present
in the renders bucket for the target region/version:

| Gender | Coat partClass/id | Pants partClass/id |
|--------|-------------------|--------------------|
| Male   | `Coat/1040036`    | `Pants/1060026`    |
| Female | `Coat/1041046`    | `Pants/1061039`    |

Atlas object keys (per `storage/atlas.go`):
`<scope>/regions/<region>/versions/<version>/atlases/Coat/1040036.png` (+`.json`),
and likewise for the other three.

**Verification procedure (execution gate, run before the branch is called
done):**
1. Functional check — request an empty-loadout render for a male face and a
   female face against a live/dev atlas-renders and confirm (a) the produced PNG
   shows clothing and (b) the service logs contain **no** `missing atlas:
   partClass=Coat|Pants` warning for the four ids.
2. If a warning fires for any id, the atlas is absent. Resolve by either:
   - (a) trigger/await ingestion of the missing Coat/Pants id for that
     region/version, or
   - (b) substitute an alternate beginner id confirmed present in the bucket and
     update the constants in `gender.go` (and the PRD table).

   The decision between (a) and (b) is deferred to execution because it depends
   on what the probe finds; the design only fixes the *constants are the single
   source of truth* invariant so a swap is a one-line change in `gender.go`
   (and its TS counterpart is not needed — the ids live only on the Go side;
   the UI never names item ids, it only passes gender).

This design does **not** itself create or ingest assets (per non-goal "no new
underwear sprite"); it only fixes the verification gate and keeps the ids
swappable.

## 9. Testing strategy

Go (`atlas-renders`):
- `ResolveGender`: table — explicit 0/1 wins over face; female face `21xxx`→1;
  male face `20xxx`→0; face `0`/negative→0; param `-1`→infer.
- `applyDefaultClothing`: the four FR-2 cases (overall suppresses both; real top
  + empty bottom; empty top + real bottom; both empty) plus female ids.
- `ParseRenderQuery`: `gender` absent→`-1`; `"0"`/`"1"`→0/1; `"2"`/`"x"`→error.
- `CanonicalLoadoutString`: trailing gender changes the string/hash; male vs.
  female empty loadout produce **different** hashes (no collision).
- Handler: a UI-style URL with `gender` query param passes hash validation;
  mismatched/absent gender produces the expected 400/recomputation behavior.

TypeScript (`atlas-ui`):
- `resolveGender` parity table (same vectors as Go).
- `canonicalLoadoutString` includes trailing gender.
- `generateCharacterUrl` emits `gender` query param and a hash matching the
  canonical string.
- Regenerated `loadout-hashes.json` rows (now with `gender`) pass.
- `useCharacterImage` query-key hash equals the URL hash for a gendered loadout
  (guards the three inline builders).

Build/verify gates (CLAUDE.md): `go test -race ./...`, `go vet ./...`,
`tools/redis-key-guard.sh`, `docker buildx bake atlas-renders`; atlas-ui `npm run
test|lint|build`.

## 10. Risks

- **Inline-builder drift (highest):** `useCharacterImage.ts` rebuilds the
  loadout/canonical in three places; missing one yields a query-key↔URL hash
  mismatch (silent cache misses / wrong key). Mitigated by the query-key==URL
  test above and by enumerating all three sites in §5.
- **Go↔TS format skew:** any difference in field order/format breaks every
  UI-produced URL. Mitigated by the mirrored fixtures/cases in §5/§9.
- **Absent assets:** handled by §8; bounded to "slot renders bare" (no crash).
- **Cache churn:** every existing hash changes once (accepted, PRD §6 lazy
  regeneration; no purge).

## 11. Files touched

atlas-renders (Go):
- `character/gender.go` — **new**: constants, `ResolveGender`,
  `defaultCoat`/`defaultPants`.
- `character/query.go` — parse optional `gender` into `RenderQuery.Gender`.
- `character/composite.go` — `applyDefaultClothing`, call after slot-map build.
- `character/hash.go` — trailing `gender` param on `CanonicalLoadoutString`.
- `character/handler.go` — resolve gender once, feed into canonical recompute.
- `character/*_test.go` — gender/query/composite/handler cases.

atlas-ui (TypeScript):
- `services/api/characterRender.service.ts` — gender on loadout, resolve helper,
  canonical mirror, query-param emission, `characterToLoadout`.
- `lib/hooks/useCharacterImage.ts` — pass gender through the three builders.
- `services/api/__tests__/characterRender.service.test.ts` +
  `__tests__/loadout-hashes.json` — regenerated with gender.

No nginx, deploy, schema, or `atlas-character` changes.
