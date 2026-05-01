# Self-hosted character render service — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-01

---

## 1. Overview

atlas-ui currently renders character images by hitting `https://maplestory.io/api/...`, an external service over which we have no control. The dependency has three concrete operational problems: (a) maplestory.io's `/character/center/...` endpoint returns a tight bounding box that varies by equipment, which forced the JS pixel-scan workaround in `CharacterRenderer.frameMode='platform'`; (b) every render is a 302 redirect, adding latency on cold loads; (c) deployments without internet egress to `maplestory.io` cannot render characters at all.

This task replaces the dependency with an in-cluster render service powered by atlas-wz-extractor. The same Character.wz that's already extracted to `atlas-assets-pvc` becomes the source for compositing. The existing `mapimage` package in atlas-wz-extractor (layered sprite blit + z-sort + origin alignment) provides the compositing primitives. atlas-assets keeps serving PNGs as static files. atlas-ui swaps its URL builder and retires the pixel-scan.

The win is operational (no external dep, faster cold loads, works offline) and architectural (we control canvas dimensions, can extend to walk/jump animations, can cache aggressively per tenant/region/version).

## 2. Goals

Primary:

- Eliminate the runtime dependency on `maplestory.io` from atlas-ui.
- Provide deterministic-canvas character renders for any `(skin, hair, face, equipment[], stance, frame, resize)` tuple supported by the tenant's Character.wz extract.
- Cache renders to atlas-assets-pvc keyed by a hash of the loadout, served by existing atlas-assets nginx.
- Expose the API for `stand1`, `stand2`, `walk1`, `alert`, `jump` stances and arbitrary frame indices, even if the UI initially only exercises `stand1/0` and `stand2/0`.
- Wipe the character cache on every wz-extractor extraction run for the affected `(tenant, region, version)` triple.

Non-goals:

- Pet rendering, mount rendering. Mount slots in a render request are silently dropped (only the rider is composited).
- Cash equipment slot rendering (slots -101 through -114).
- Animation interpolation, GIFs, or multi-frame outputs in a single response.
- Replacing or restructuring `CharacterRenderer.tsx` beyond switching the URL source.
- Backwards compatibility with the maplestory.io URL path. Hard cutover.

## 3. User Stories

- As an admin viewing AccountDetailPage, I want character tiles to render at uniform scale and foot alignment without needing JS pixel-scans, so the slot grid reads as a clean roster.
- As an admin previewing presets in ApplyPresetDialog, I want preset cards to render quickly even on a cold cache, with no external service hop.
- As an operator running atlas in a network without egress to `maplestory.io`, I want characters to render entirely from local assets.
- As a future contributor adding walk/jump animation to a character detail view, I want the render endpoint to already accept stance + frame parameters so I only have to wire the UI.

## 4. Functional Requirements

**Asset extraction (atlas-wz-extractor)**

- FR-1: The Character.wz dispatch in `image/extract.go` is extended beyond `info/icon` extraction to emit *worn-sprite* assets and per-sprite metadata for the slot categories listed in §FR-3.
- FR-2: Each emitted sprite asset is paired with a JSON metadata sidecar carrying `origin`, `map` (joint points: neck, navel, hand, etc.), `z` (layer string), `group`, `delay`, and the canvas's `vslot`/`islot` from the parent `info` block.
- FR-3: The covered equipment slot categories are: **Cap, Face Accessory (Accessory), Eye Accessory, Earrings, Top (Coat), Overall (Longcoat), Bottom (Pants), Shoes, Glove, Shield, Cape, Weapon, Hair, Face, Body skin imgs (`00002000.img` … `00002013.img` and `00012000.img` … `00012013.img` for both genders)**.
- FR-4: The smap (z-order resolution table) source is identified, parsed, and emitted as a single JSON file per `(region, version)` for use by the renderer at composition time.
- FR-5: Stale character cache is wiped at the start of every extraction run, scoped to the `(tenant, region, version)` triple about to be re-extracted.

**Render endpoint (atlas-wz-extractor)**

- FR-6: A new HTTP endpoint composites a character from inputs and returns `image/png`. Exact path/query shape is detailed in `api-contracts.md`.
- FR-7: The endpoint accepts `stance ∈ {stand1, stand2, walk1, alert, jump}` and `frame ∈ {0..N}` where N depends on the stance + body sprite. Invalid combinations return 400 with a JSON error body.
- FR-8: The endpoint accepts a `resize ∈ {1, 2, 3, 4}` integer scale factor. Default 2.
- FR-9: Mount slots (-18 saddle, -19 mount, -20 mob equip), pet slots (-14, -21..-30), and cash slots (-101..-114) in the request are silently dropped from the equipment map before compositing.
- FR-10: Unknown templateIds (not present in extracted Character.wz) return 400 with the offending id named.
- FR-11: First request for a unique loadout renders synchronously, persists the PNG to `atlas-assets-pvc` under a deterministic path, and returns it in the same response. Concurrent requests for the same loadout are deduplicated (single render, multiple readers).
- FR-12: Subsequent requests for the same loadout are served by atlas-assets nginx as static files; atlas-wz-extractor is not contacted on cache hits.

**Compositor (atlas-wz-extractor)**

- FR-13: The compositor produces fixed-canvas output: **96×128 native pixels** with the character's foot row at row 124 (4-pixel ground padding). Scaled by `resize` for output dims (e.g. resize=2 → 192×256).
- FR-14: Layering uses the smap-resolved z-order from FR-4. Sprites of the same z value tie-break by extraction order.
- FR-15: Joint alignment uses the `origin` + `map` system: every sprite's anchor is mapped to the parent body's complementary joint (e.g. hair's `neck` map point aligns with body's `neck`).
- FR-16: Two-handed weapons force `stance=stand2` regardless of the request's stance value, matching MapleStory client behavior. (Already partially handled in atlas-ui via `isTwoHandedWeapon`; the canonical decision moves server-side.)
- FR-17: When a requested loadout includes a mount slot, the rider is rendered standing on the standard ground line. The mount sprite itself is dropped — see Non-goals.

**Cutover (atlas-ui)**

- FR-18: `mapleStoryService.generateCharacterUrl()` is replaced with a builder that produces URLs against the new endpoint. The builder takes the same input shape so call sites are unchanged.
- FR-19: `useCharacterImage` and `CharacterRenderer` are updated for the new URL contract. The `frameMode='platform'` pixel-scan is removed entirely; foot alignment is now guaranteed by the canvas contract.
- FR-20: The maplestory.io constants in `services/api/maplestory.service.ts` (apiBaseUrl, apiVersion, etc.) are deleted along with all dead code paths that referenced them.
- FR-21: The hard cutover means once this lands, atlas-ui sessions issue zero requests to `maplestory.io`. Verifiable in the browser Network tab.

**Operations**

- FR-22: Deploy adds an ingress route mapping `/api/character/render` (or chosen path) to atlas-wz-extractor.
- FR-23: A K8s readiness gate ensures the wz-extractor pod is ready before atlas-ui pods accept traffic on first deploy.

## 5. API Surface

See `api-contracts.md` for the detailed endpoint contract. In summary:

- **POST** (or **GET** with deterministic param ordering) on atlas-wz-extractor returns `image/png` for a `(tenant, region, version, skin, hair, face, items, stance, frame, resize)` tuple.
- **GET** on atlas-assets returns the same PNG from cache for any subsequent requestor.
- A loadout-hash function maps the input tuple to a stable filename. The same hash is used both as the cache filename and as the uncached endpoint's redirect target.

## 6. Data Model

See `data-model.md` for the Character.wz subset structure and the sprite metadata schema. In summary:

- Character.wz body sprites live at `0000{skin}.img` / `0001{skin}.img` (gendered).
- Equipment subdirs: Cap, Cape, Coat, Glove, Hair, Pants, Shoes, Shield, Weapon, Face, Accessory, Longcoat (overall), Earrings.
- Each `.img` has an `info/{islot, vslot, cash}` block plus per-stance/frame canvases.
- Each canvas declares an `origin` (anchor point on the sprite) and a `map` directory of joint points (`neck`, `navel`, `hand`, `head`, etc.).
- Layer order uses string `z` values resolved against a global smap.
- A single render walks: pick body skin frames → join head, face, hair → join equipped items via their respective joints → sort by z → blit in order onto the fixed canvas.

## 7. Service Impact

- **atlas-wz-extractor** — major. New extraction phase, new HTTP endpoint, new compositor wiring around `mapimage` primitives, new cache management. Estimated 1500–2500 LOC including tests.
- **atlas-assets** — none. Existing nginx serves the new `character/{hash}.png` paths the same way it serves icons today.
- **atlas-ui** — small. URL builder swap (~20 LOC), pixel-scan retirement (~50 LOC removed), constants cleanup. Tests updated for new mocks.
- **atlas-ingress / deploy** — small. Route registration in `deploy/k8s/ingress.yaml` for the dynamic render path; static path is already covered.
- **CI** — small. New Go tests for compositor + extractor extension.

## 8. Non-Functional Requirements

**Performance**

- Cold render p95 < 500ms for typical loadouts (≤ 8 equipped slots, resize=2).
- Cache hit p95 < 50ms (served by nginx static).
- Concurrent requests for the same loadout deduplicate to a single render.

**Storage**

- Character cache size on PVC is bounded by extraction-cycle wipe (FR-5). Between wipes, growth is unbounded but proportional to unique loadouts requested.
- A future iteration may add LRU eviction; out of scope for v1.

**Security**

- Endpoint validates all inputs (skin range, item id existence, stance/frame validity). No path traversal via templateId or stance values.
- Tenant scoping enforced via the URL path — a request for tenant A cannot render assets from tenant B's extract.
- No authentication on the render endpoint itself (consistent with current atlas-assets, which is also unauth).

**Observability**

- Render endpoint emits OTel spans (`character.render`) with attributes: tenant, region, version, stance, frame, equipped-slot-count, cache-hit, render-ms.
- Counters: `character_render_total{cache_hit, stance}`, `character_render_errors_total{reason}`.
- The "no progress indicator" gap from §wz-extractor status discussion is acknowledged but not blocking — future task.

**Multi-tenancy**

- All cache paths and render assets are namespaced by `(tenant, region, version)`. No cross-tenant cache pollution.
- A tenant whose Character.wz hasn't been extracted yet returns 404 with a clear error message.

## 9. Open Questions

- **Smap location.** The z-order table — is it `Base.wz/zmap.img`, hardcoded in the client, or somewhere else? Needs a probe before implementation begins. Falls under the design phase.
- **Two-handed determination.** The atlas-ui has `TWO_HANDED_WEAPON_RANGES` hardcoded. Move to atlas-constants? Derive from Character.wz weapon `info`? Need a single source of truth.
- **Skin color mapping.** atlas-ui's `SKIN_COLOR_MAPPING` translates internal 0–10 to WZ-named 2000–2013. The new endpoint should accept the internal value and map it server-side, OR the atlas-ui keeps converting. Pick one.
- **Walk/jump frame counts.** Each stance has variable frame counts in the WZ. Endpoint should validate per-stance, but the per-stance frame max isn't known until the extractor populates metadata. Bootstrapping question.
- **Ear / face features.** maplestory.io has flags `showEars`, `showLefEars`, `showHighLefEars` for Lef ears. Are these needed for atlas? If not, skip; if yes, add to API.

## 10. Acceptance Criteria

- [ ] atlas-wz-extractor's Character.wz dispatch emits worn-sprite assets + metadata for all FR-3 slot categories.
- [ ] Smap resolution table is extracted and consumed by the compositor.
- [ ] New render endpoint returns deterministic 96×128 (resize=1) / 192×256 (resize=2) PNGs for the test loadouts: bare body, equipped warrior, mage with tall hat, archer with long hair, polearm wielder.
- [ ] Cold render p95 < 500ms across the test loadout suite (measured locally).
- [ ] Cache hit serves the PNG via atlas-assets nginx; atlas-wz-extractor receives no traffic for the second request of the same loadout.
- [ ] Character cache directory is wiped at the start of every extraction run, scoped to the affected `(tenant, region, version)`.
- [ ] atlas-ui's `mapleStoryService.generateCharacterUrl` is replaced; no source file references `maplestory.io`.
- [ ] `CharacterRenderer.frameMode='platform'` pixel-scan code is deleted.
- [ ] Browser Network tab shows zero requests to `maplestory.io` during a full atlas-ui session that views accounts, characters, and presets.
- [ ] An atlas deployment with no internet egress can render every test loadout end-to-end.
- [ ] Endpoint accepts `stance ∈ {stand1, stand2, walk1, alert, jump}` and rejects unknowns with 400.
- [ ] Endpoint validates `frame` per stance and rejects out-of-range with 400.
- [ ] Mount/pet/cash slots in the request are silently dropped (no error, no render of those layers).
- [ ] Two-handed weapon overrides stance to `stand2` regardless of request.
- [ ] OTel spans are emitted with the documented attributes; counters increment correctly.
- [ ] Tenant isolation: requests for tenant A cannot render using tenant B's assets.
