# Character Render — Default Clothing Fallback — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-03
---

## 1. Overview

The `atlas-renders` service composites a character image from a base body skin
(WZ ids 2000–2013), a head, hair, face, and the character's equipped items.
Clothing slots are filled only when a corresponding item is equipped: `Coat`
(Top), `Pants` (Bottom), or `Longcoat` (Overall). When none of those are
equipped, `Composite()` simply skips them and renders the bare body + head —
i.e. a nude character. This is visible anywhere a character with empty clothing
slots is rendered (UI character previews, account/character lists, etc.).

There is **no separate "underwear" sprite** in the Character.wz data; verified
against `libs/atlas-wz/charparts` extraction and the body part set (a body
atlas exposes `body`/`arm`/etc. parts under a single `Bd` occlusion slot — no
garment layer). MapleStory's canonical "basic" appearance is simply the
**beginner clothing items** assigned at character creation (a plain
shirt + shorts). The fix is therefore to inject default beginner clothing for
any empty clothing slot so a character is never rendered nude.

The default clothing is gender-specific, but the render endpoint currently only
receives `skin/hair/face/items` — not gender. This PRD introduces an optional
`gender` query parameter (the UI supplies the character's real stored gender),
with a deterministic face-id fallback for direct API callers. Because the
render is cached by a strict loadout hash, gender becomes part of the canonical
hash input on both the Go service and the atlas-ui producer.

## 2. Goals

Primary goals:
- A character with no Top, Bottom, or Overall equipped renders wearing
  gender-appropriate beginner clothing instead of a bare body.
- The fallback is applied per clothing slot independently (a character missing
  only a bottom still gets default pants, etc.).
- The UI uses the character's authoritative stored gender; direct API callers
  that omit gender still get a clothed, non-colliding render via face inference.
- The render cache remains correct — no two distinct rendered images share a
  cache key.

Non-goals:
- No new "underwear" sprite asset is created or ingested.
- No opt-out / "render bare body" mode (decided: always clothe).
- No change to character creation, inventory, or how `atlas-character` stores
  gender.
- No proactive purge/migration of the existing render cache (decided: lazy
  regeneration; see §6).
- No change to hair, face, hat, weapon, or any non-clothing slot behavior.

## 3. User Stories

- As a player viewing my character in the UI, I want my character to wear basic
  clothes even before I equip anything, so it never appears nude.
- As an operator/admin browsing character lists, I want every character thumbnail
  to be presentable regardless of equipped items.
- As a developer calling the render endpoint directly (without a gender param),
  I want the service to still produce a clothed image rather than a nude one.

## 4. Functional Requirements

### FR-1 Default clothing item ids

The default beginner clothing item ids (user-confirmed) are:

| Gender | Coat (Top) | Pants (Bottom) |
|--------|-----------|----------------|
| Male (0)   | `1040036` | `1060026` |
| Female (1) | `1041046` | `1061039` |

These ids MUST be defined as named constants in `atlas-renders` (single source
of truth), not scattered literals.

### FR-2 Per-slot independent fallback

After the equipped-item slot map is built (in `Composite()`), apply defaults:

- Let `hasOverall` = an Overall (item `Classification == ClassificationOverall`)
  occupies the top slot (`-5`).
- If `hasOverall` is true → inject nothing (an overall covers both top and
  bottom).
- Else:
  - If the top slot (`-5`) is empty → inject the gender's default **coat** at `-5`.
  - If the bottom slot (`-6`) is empty → inject the gender's default **pants** at `-6`.

The two checks are independent: a character with a real top but no bottom gets
default pants; a character with a real bottom but no top gets a default coat.

A default is injected into the slot map exactly like an equipped item, so it
flows through the existing compositing, z-ordering, and vslot-occlusion paths
with no special-casing.

### FR-3 Gender resolution

Gender for default selection is resolved as:

1. If the `gender` query param is present and is `0` or `1` → use it.
2. Otherwise (absent / unspecified) → infer from the **face** id using the v83
   convention `(faceId / 1000) % 10`: `1` ⇒ female, anything else ⇒ male. A
   non-positive / unknown face id resolves to male (`0`).

Hair MUST NOT be used for inference (later female hair ranges break the clean
modulo; face is stable). Repo-verified against the char-creation template
(`CharMale` faces `200xx`, `CharFemale` faces `210xx`).

Resolution is deterministic and identical on both the Go service and the TS
producer so the loadout hash matches (see FR-5).

### FR-4 `gender` query parameter

The render endpoint accepts an **optional** `gender` query parameter:

- Allowed values: `0` (male) or `1` (female).
- Absent → treated as "unspecified" and resolved via face inference (FR-3).
- Present but not `0`/`1` → `400 invalid-input` (consistent with existing query
  validation in `ParseRenderQuery`).

### FR-5 Loadout hash includes gender

The render is cached by a 16-hex-char loadout hash, and `handler.go` rejects any
request whose URL hash does not match `CanonicalLoadoutString(...)`. Because
gender now changes the rendered pixels, the **resolved** gender MUST be appended
to the canonical loadout string on both sides:

- Go: `services/atlas-renders/.../character/hash.go#CanonicalLoadoutString`.
- TS: `services/atlas-ui/src/services/api/characterRender.service.ts#canonicalLoadoutString`.

Both sides append the resolved gender value in the same position and format.
The UI also sends `gender` as a query parameter so the service recomputes the
same canonical string. This is a one-time change to every existing hash (see
§6 for cache impact).

### FR-6 Always clothe (no opt-out)

There is no parameter or mode to render the bare body. The fallback always
applies when a clothing slot is empty.

### FR-7 Missing-atlas behavior (degradation)

If a default clothing atlas is not present in MinIO for the target
region/version, the existing "missing atlas → log warning → skip" path applies
and the slot renders bare (same as today for any missing equip). This is a
known degradation, not a hard failure — see the verification item in §9.

## 5. API Surface

Endpoint (unchanged path):

```
GET /api/wz/character/render/{tenant}/{region}/{version}/{hash}.png
    ?skin=&hair=&face=&stance=&frame=&resize=&items=&gender=
```

New query parameter:

| Param  | Type | Required | Values | Default |
|--------|------|----------|--------|---------|
| gender | int  | no       | 0, 1   | unspecified → inferred from face |

Error cases:
- `gender` present but not `0`/`1` → `400` `invalid-input`.
- URL hash mismatch (now also a function of resolved gender) → `400`
  `hash-mismatch` (unchanged mechanism).

No response shape changes (still `image/png`).

## 6. Data Model

No persistent data model changes. No new tables, columns, or migrations.

Cache impact (MinIO renders bucket): adding gender to the canonical string
changes every character render's hash/key. Existing cached PNGs become orphaned
under their old keys and are never read again. **Decision: lazy regeneration** —
new requests miss the cache and recomposite on first hit; stale objects are left
to a future bucket lifecycle/cleanup. No deploy-time purge is required.

## 7. Service Impact

### atlas-renders (Go) — primary
- `character/query.go` — parse optional `gender` (sentinel for "unspecified").
- `character/composite.go` — resolve gender (param → face inference), inject
  per-slot defaults; define default-clothing constants and the gender/inference
  helpers.
- `character/hash.go` — `CanonicalLoadoutString` gains a resolved-gender field.
- `character/handler.go` — resolve gender and feed it into the canonical hash
  recomputation used for URL-hash validation.
- Corresponding `*_test.go` updates (query parsing, hash canonical, composite
  injection, handler hash validation).

### atlas-ui (TypeScript)
- `src/services/api/characterRender.service.ts`:
  - `CharacterLoadout` gains `gender?: number`.
  - `canonicalLoadoutString` gains the resolved-gender field (mirrors Go).
  - `generateCharacterUrl` resolves gender (loadout gender → face inference),
    adds it to the canonical string and as a `gender` query param.
  - `characterToLoadout` populates `gender` from `character.attributes.gender`.
  - Add gender/face-inference helpers mirroring the Go logic.
- Update `__tests__/characterRender.service.test.ts` and any hash-dependent
  callers (`lib/hooks/useCharacterImage.ts`,
  `lib/utils/character-cache-sw.ts`) if signatures or hash inputs change.

### atlas-character
- No code change. Already exposes `gender` in its REST resource
  (`character/entity.go:52`, `character/rest.go:38`); the UI already receives it.

## 8. Non-Functional Requirements

- **Correctness / caching:** No two distinct rendered images may share a cache
  key. Gender is part of the canonical hash on both producer and service (FR-5).
- **Determinism:** Gender resolution and canonical-string construction are
  byte-for-byte identical between Go and TS so UI-produced hashes always pass
  service validation.
- **Multi-tenancy:** Unchanged — render scope remains tenant/region/version
  keyed; gender adds no tenant surface.
- **Performance:** Negligible — at most two extra atlas fetches (coat/pants),
  already LRU-cached like any equip; injection is O(1) map work.
- **Observability:** A missing default-clothing atlas logs at the existing
  warn level (same path as any missing equip).
- **Backward compatibility:** Direct callers that omit `gender` keep working and
  now render clothed (via face inference) instead of nude.

## 9. Open Questions

- **Asset verification (must confirm during design/execution):** Are the four
  default atlases — `Coat/1040036`, `Pants/1060026`, `Coat/1041046`,
  `Pants/1061039` — actually extracted to MinIO for the target region/version?
  If any are absent, FR-7 degradation leaves that slot bare. If they are not
  ingested, design must decide whether to (a) trigger/await ingestion, or
  (b) pick alternate ids known to exist. This is the one item that can block the
  feature from being visibly effective.
- Confirm `character.attributes.gender` is reliably populated on the UI Character
  model for all code paths that build a loadout (not just the detail page).

## 10. Acceptance Criteria

- [ ] A male character with no Top/Bottom/Overall renders wearing coat `1040036`
      and pants `1060026`.
- [ ] A female character with no Top/Bottom/Overall renders wearing coat
      `1041046` and pants `1061039`.
- [ ] A character with an Overall and no separate Top/Bottom renders with the
      overall only — no default coat or pants injected.
- [ ] A character with a real Top but no Bottom renders with the real top and
      default pants (and the symmetric case).
- [ ] The `gender` query param (0/1) selects the male/female default set; an
      invalid value returns `400`.
- [ ] With `gender` omitted, the service infers from face id and still renders
      clothed; a male-face and a female-face empty-loadout produce **different**
      cache hashes (no collision).
- [ ] UI-produced render URLs include `gender`, and the service's URL-hash
      validation passes (UI and service canonical strings match).
- [ ] `go test -race ./...` and `go vet ./...` clean in atlas-renders;
      `docker buildx bake atlas-renders` succeeds.
- [ ] atlas-ui `npm run test`, `npm run lint`, and `npm run build` clean.
- [ ] Asset-verification open question (§9) is resolved with evidence before the
      branch is called done.
