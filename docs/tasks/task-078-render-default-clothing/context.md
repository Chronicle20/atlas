# Task 078 — Render Default Clothing — Implementation Context

Companion to `plan.md`. Captures the key files, decisions, and gotchas an executor needs that aren't re-stated inline in every task.

## Goal in one line

Never render a nude character: inject gender-appropriate beginner coat/pants into empty clothing slots, threading a resolved `gender` value through the render contract and into the loadout cache hash on both the Go service and the TS producer.

## Key files (source of truth)

Go — `services/atlas-renders/atlas.com/renders/character/`:
- `composite.go` — `Composite()` builds `equipment := FilterEquipment(ItemsToSlotMap(q.Items))` at line 264; the default injection call goes immediately after. `slotForItemID` already maps both `ClassificationTop` and `ClassificationOverall` to slot `-5` and `ClassificationBottom` to `-6` — there is no separate overall slot, so "is an overall equipped" is answered by classifying whatever sits in `-5`. The injected id is an ordinary map entry, so it flows through `partClassFor` → `fetchAtlas` → joints → vslot occlusion → z-order with no special-casing.
- `query.go` — `RenderQuery` + `ParseRenderQuery`. All fields are values (no pointers); add `Gender int` as a value with a `-1` sentinel, not `*int`.
- `hash.go` — `CanonicalLoadoutString` (11 params today) + `LoadoutHash` (sha256 → first 16 hex). Format string is `"%s|%s|%d.%d|%d|%d|%d|%s|%d|%d|%s"`; gender appends `|%d`.
- `handler.go:81-84` — the canonical recompute used for URL-hash validation. The hash check returns `400 hash-mismatch` **before** any storage access (and `s == nil` → 503 is checked first), which is why the package has no full `Handler()` HTTP test and why Task 3 covers the handler's gender logic via a direct `ResolveGender`+`CanonicalLoadoutString` composition test instead of a storage/tenant harness.

TS — `services/atlas-ui/src/`:
- `services/api/characterRender.service.ts` — the canonical TS module: `CharacterLoadout`, `canonicalLoadoutString`, `loadoutHash` (uses `js-sha256`), `generateCharacterUrl`, `characterToLoadout`. This is the byte-for-byte mirror of the Go hash/gender logic.
- `lib/hooks/useCharacterImage.ts` — has **five** sites that must learn gender: `generateQueryKey` (calls `canonicalLoadoutString`), and four `generateCharacterUrl` loadout literals (`queryFn`, `prefetchVariants`, `preloadImages`, `warmCache`). Miss one → the React-Query key hash diverges from the URL hash → silent cache misses. `generateQueryKey` is currently module-private; Task 5 exports it so the drift-guard test can read it.
- `services/api/__tests__/loadout-hashes.json` — cross-language parity fixture; regenerated (not hand-edited) so `canonical`/`expectedHash` can never drift from the algorithm.

Confirmed present (no code change needed):
- `Character.attributes.gender: number` — `src/types/models/character.ts:28`.
- `MapleStoryCharacterData.gender: number` — `src/types/models/maplestory.ts:106` (every hook call site has it in hand).
- `libs/atlas-constants/item`: `GetClassification(Id) Classification`, `Id = uint32`, `ClassificationTop=104`, `ClassificationOverall=105`, `ClassificationBottom=106` — already imported by `composite.go`.

## Key decisions (from design.md)

1. **`ResolveGender` is one pure idempotent function**, mirrored in Go and TS. Explicit `0|1` wins; else `(face/1000)%10 == 1 ⇒ female`; non-positive face ⇒ male. Idempotence lets the handler resolve once (for the hash) and `Composite` resolve again (for injection) without threading a value — no new `Composite` parameter. **Hair is deliberately not consulted** (later female hair ranges break the clean modulo; face `200xx` male / `210xx` female is stable).
2. **`RenderQuery.Gender` uses `-1` (`GenderUnspecified`) sentinel**, not `*int` — every other field is a value and the struct is passed by value.
3. **Default injection is a slot-map entry**, not a parallel "defaults" list — that's the whole point; zero special-casing downstream. Overall in `-5` suppresses both defaults; the two slots are otherwise independent.
4. **Gender appends to the END of the canonical string** (`...|items|gender`) on both sides — minimal diff, easy to audit against the TS mirror. Every existing hash changes once; PRD §6 accepts lazy regeneration (no cache purge).
5. **Determinism contract:** the UI emits the **resolved** gender both as the trailing canonical field and as the `gender=` query param, so the service receives `gender=0|1` explicitly and `ResolveGender(0|1, face)` returns it verbatim — UI and service canonical strings are byte-identical by construction.

## Gotchas

- **Signature ripple (Go):** extending `CanonicalLoadoutString` breaks `handler.go` and three call sites in `handler_test.go` simultaneously. Task 3 updates all of them in one commit so the package always compiles.
- **Signature ripple (TS):** adding a required trailing `gender` to `canonicalLoadoutString` ripples into `generateCharacterUrl` (same file) and `generateQueryKey` (hook). Task 5 lands all production code in Commit A (so `npm run build` stays green — tests are excluded from `tsc -b`) and fixture+tests in Commit B (so `npm run test` goes green). Between A and B, `npm run test` is expected to be red; that's why build is the A checkpoint and test is the B checkpoint.
- **`Math.floor` in TS:** Go uses integer division (`face/1000`); TS division is float, so the mirror must use `Math.floor(face / 1000) % 10`, guarded by `face > 0`.
- **Fixture hashes are derived, never typed by hand.** The Task 5 generator script computes `canonical` + `expectedHash` with `js-sha256`; do not hand-author hash values.
- **URL path vs route comment:** `handler.go`'s doc comment says `/api/wz/character/render/...` but the TS producer emits `/api/assets/{tenant}/{region}/{maj}.{min}/character/{hash}.png`. nginx forwards the query string verbatim; `gender` is a query param on either path. **No nginx/route change** (`deploy/shared/routes.conf:214` keys on the 16-char hash only).
- **No new shared lib / go.mod change** → no Dockerfile `COPY` edits. `gender.go` is import-free; `composite.go` already imports `item`. The `docker buildx bake atlas-renders` in Task 6 is run to satisfy the PRD §10 acceptance criterion, not because go.mod changed.
- **redis-key-guard:** this change touches no Redis; the guard run in Task 6 is a clean-baseline check (run with `GOWORK=off` if invoked manually, per `reference_rediskeyguard_invariant`).

## Dependencies / ordering

Go Tasks 1→2→3→4 are ordered for compile-green TDD (constants → query field → hash signature+callers → injection). TS Task 5 depends on nothing in the Go tasks (separate module) but is sequenced after for review locality. Tasks 6/7 are verification gates. **Task 8 (asset verification) is the one release-blocking gate** and needs a live/dev atlas-renders + MinIO — it cannot be satisfied from the repo alone; record evidence (verified-present or swapped-id) before calling the branch done.

## Verification commands (full gate)

Go (from `services/atlas-renders/atlas.com/renders`): `go test -race ./...`, `go vet ./...`, `go build ./...`.
Repo root: `tools/redis-key-guard.sh`, `docker buildx bake atlas-renders`.
TS (from `services/atlas-ui`): `npm run lint`, `npm run test`, `npm run build`.

## After execution

Run the code-review step (`superpowers:requesting-code-review` → backend + frontend reviewers, both file groups changed) **before** opening a PR, per CLAUDE.md.
