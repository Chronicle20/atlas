# Character render — risks

## High

### Smap source unidentified

The z-order resolution table — the canonical `string → int` mapping that orders sprite layers — has not yet been confirmed in the WZ files. v83 GMS clients reference a smap from `Base.wz/smap.img` or `Character.wz/smap.img`, but our extracted XML hasn't been verified.

**Impact:** without the smap, renders will have layering bugs (hat behind hair, weapon in front of arm, etc.) — visible artifacts that erode user trust quickly.

**Mitigation:**

1. Probe `tmp/.../Base.wz/` and `Character.wz/` for `smap.img` early in the design phase. If present, parse and emit as a JSON file per `(region, version)`.
2. If absent, ship a hardcoded smap derived from a known-good reference (HaRepacker / open-source MapleStory client work). Document the source in code comments.
3. Add a regression test that renders a known multi-layer loadout and pixel-compares against a fixture.

### Two-handed weapon classification correctness

`stand1` vs `stand2` selection depends on whether the equipped weapon is two-handed. atlas-ui currently has hardcoded `TWO_HANDED_WEAPON_RANGES`. atlas-constants has `item.WeaponType` and `TWO_HANDED_WEAPON_TYPES` — but the source of truth in v83 is the weapon's `info.attackSpeed` / `info.attackType` in Character.wz.

**Impact:** wrong stance picks the wrong body sprite frames, which then propagates to wrong joint positions for the weapon itself. Visible artifacts.

**Mitigation:**

1. The render service derives two-handed-ness from the weapon's `info` block in Character.wz, not from a hardcoded range.
2. atlas-ui's hardcoded range table is removed; the API call doesn't need it (server decides).
3. Cross-check against a known set of weapons (sword 130xxxx, polearm 144xxxx, claw 147xxxx) in tests.

## Medium

### Cache PVC growth

FR-5 wipes the character cache at extraction time. Between extractions, growth is unbounded but proportional to unique loadouts requested. With ~13K possible item combos × ~5 stances × ~4 resize factors, the upper bound is ~250K entries × ~5 KB avg = ~1.2 GB. Realistic working set is 1–10% of that.

**Impact:** PVC fills over time. The cluster's `atlas-assets-pvc` size is currently sized for icon assets, not character renders.

**Mitigation:**

1. Document the projected upper bound in the deploy yaml.
2. Add a metric for cache size.
3. Future iteration: LRU eviction at 80% PVC capacity. Out of scope for v1.

### First-render latency on large loadouts

Compositing a 12-slot character at resize=4 may exceed the 500ms p95 target. Each blit is a per-pixel draw.Over operation, and resize=4 outputs are 384×512 = 196K pixels per layer × ~15 layers = ~3M alpha-composite ops.

**Impact:** UI feels slow on cold cache for fully-decked characters.

**Mitigation:**

1. Pre-scale composited output, not each layer. Composite at native 96×128 then `image.NRGBA.Resize` once at the end.
2. Cache parsed sprite metadata in-process (LRU keyed by templateId) so sidecar JSON parsing is amortized.
3. Run the synchronous render off a worker pool to bound concurrency.
4. Measure on a real cluster pod, not dev laptop, before tuning.

### maplestory.io fallback not retained

Per FR-21, this is a hard cutover. If our renderer has a correctness regression in production (wrong layering, missing equipment), there's no graceful degradation — characters look broken.

**Impact:** rolling back means rolling back atlas-ui too, not just the render service.

**Mitigation:**

1. Stage in a non-prod tenant first; confirm a representative loadout sample before cluster-wide deploy.
2. Keep the maplestory.io URL builder code in git history (deleted, not amended) so a quick revert is possible if needed.
3. The user explicitly chose hard cutover (PRD §3, decision Q6) — accepted risk.

## Low

### Sprite version drift

If the wz-extractor is stale (older Character.wz than the running atlas-channel), a player equipping a brand-new item gets a 400 from the render endpoint because the templateId doesn't exist in the extract.

**Impact:** "broken character" appearance for the small set of newly-introduced items between extractions.

**Mitigation:**

1. Surface `400 unknown-template-id` cleanly in the UI ("character image unavailable; admin needs to re-extract WZ").
2. Recommend the cluster operator runs extraction after every game version bump.

### Region/version mismatch

A tenant configured for `GMS/83.1` requesting renders for items only present in `GMS/214` (a different version's Character.wz extract) returns 400.

**Impact:** consistency edge case during version migrations.

**Mitigation:** validation already covered by FR-10 + the version path component.

### Ear / Lef-ear toggles

maplestory.io exposes `showEars`, `showLefEars`, `showHighLefEars` parameters. atlas hasn't used them so far. If a future use case needs them, the API needs additions.

**Impact:** none for v1, possible API churn later.

**Mitigation:** open question §9 of PRD; decision deferred.

### Tenant cache cross-pollution

Loadout hashes don't include tenant in the *hash function* itself (it's in the path), so two tenants with identical extractions and identical loadouts technically have identical hashes. This is fine because the path differs, but if anyone ever moves cache files between tenants (e.g. for migration scripts), they'd collide.

**Impact:** none under normal ops; latent footgun for migration tooling.

**Mitigation:** documented here; migration tooling should use full paths.
