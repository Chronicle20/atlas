# task-122 — Adversarial Review: Attack-Path Snapshots

Scope: `main...HEAD` in this worktree. Reviewed the snapshot registry/processor/shadow
(`services/atlas-channel/atlas.com/channel/character/snapshot/*.go`), the attack-path
consumer (`socket/handler/character_attack_common.go` + friends), position-feed call
sites (`movement/processor.go`, `portal/processor.go`, `kafka/consumer/character/consumer.go`),
the skill-data TTL cache (`data/skill/cache.go`, replacing `data/skill/registry.go`), and the
buff snapshot maintenance (`kafka/consumer/buff/consumer.go`). Read-only; no code changed.
Gap classes hunted: attack-path semantic equivalence, position feed, shadow verification,
skill cache.

## 1. Semantic equivalence (composed snapshot vs `GetById(InventoryDecorator, SkillModelDecorator)`)

Independently re-derived the read-set from `character_attack_projectile.go` and
`socket/writer/character_attack_common.go` (grepped every `.Method()` call on the character
model) and it matches design.md §2.1 exactly: `Id, Level, JobId, Skills, X, Y,
Equipment().Get("weapon"), Inventory().Consumable/Cash().Assets(), Hp()` (Sacrifice cost,
`character_attack_common.go:1184`). No field read by the attack path, projectile planner, or
either writer body falls outside `statValueKeys` (`character/snapshot/registry.go:246-265`) or
the inventory/skills full-replace event handlers — I did not find a decorator-populated field
(`Pets, Quests, Party, MonsterBook`, etc.) that the attack path reads; those are also never
decorated onto the model by today's `cp.GetById(InventoryDecorator, SkillModelDecorator)` call,
so there is no divergence introduced by the snapshot on that front (confirmed
`ProcessorImpl.GetById` at `character/processor.go:76-79` applies only the decorators passed in).

Wire equivalence is not just claimed — `TestWriterEquivalence_SnapshotComposedModel`
(`socket/handler/character_attack_common_test.go:615-651`) builds the same logical model two
ways (decorator chain vs snapshot `BackfillCore/Skills/Inventory` + `Get`) and asserts
byte-identical `CharacterAttackRangedBody` output. PASS.

`stat.TypeAvailableSP` has no case in `applyStat` (`registry.go:271-311`) — intentional
(comment: "AVAILABLE_SP is a per-book string table on the model"), degrades to
`InvalidateCore`. Correct, not a gap.

**No blocking finding under this heading.**

## 2. Position-feed completeness

Enumerated every writer of `position.GetRegistry().Put` (the pre-existing local position
mirror) and every caller of `movement.NewProcessor(...)`:

- `movement/processor.go:84` (`ForCharacter`, normal movement fold incl. in-map
  `TeleportElement` frames) → feeds `snapshot.SetPosition` synchronously.
- `movement/processor.go:114` (`TeleportCharacter`) → feeds `snapshot.SetPosition`
  synchronously. This is the commit-1c5431d40 fix cited in the brief; grepping today's tree
  shows exactly two `position.GetRegistry().Put` call sites and both are now paired with
  `snapshot.GetRegistry().SetPosition` — the sibling-gap vector (a `position.Put` without a
  matching `snapshot.SetPosition`) is exhausted; I found no third.
- `portal.EnterInner` (`portal/processor.go:149`, inner-portal same-map teleport) routes
  through `movement.TeleportCharacter` — covered.
- `portal.Warp` / `WarpToPosition` (owl warp `owl_warp.go:112`, generic map-change
  `map_change.go:72`, mystic door `mystic_door_enter.go:51`, resurrection chase-warp
  `skill/handler/resurrection/resurrection.go:54`) all go through an async Kafka command to
  atlas-portals → atlas-maps `ChangeMap`, which is confirmed (comment at
  `atlas-maps/.../character/warp/processor.go:112`: "atlas-maps is the sole emitter of
  MAP_CHANGED") to be the **only** source of `MAP_CHANGED`. `handleSnapshotMapChanged`
  (`kafka/consumer/character/consumer.go:600-617`) sets position from `TargetX/TargetY` when
  `UseTargetPosition`, else invalidates position **and** core so the next attack refetches
  fresh REST X/Y — exactly matches design §10.4's disposition and event-coverage.md §2.
- No GM-warp / rush / knockback / mystic-summon position-changing command exists in this
  service outside the above call sites (grepped `Rush|Knockback|FlashJump|Teleport(` and found
  only decode-only stubs); nothing to feed.
- Death/respawn (`respawn/processor.go`) is out of the diff's touched surface (unchanged file)
  and — as far as traced — dispatches through a saga that ultimately reaches atlas-maps
  `ChangeMap`, so it is covered by the same MAP_CHANGED path; not independently verified
  end-to-end (flagged under Not evaluable).

**No blocking finding under this heading** — the position-feed sibling-gap class the brief
warns about (a `TeleportCharacter`-style miss) does not recur elsewhere in this diff.

## 3. Shadow verification — blind spots

`shadow.go` samples at `CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE` (default unset → rate 0, off in
prod; `shadowRate()` `shadow.go:42-56`), bounded to 4 concurrent comparisons
(`shadowMaxInFlight`), only on a **full-hit** read (`processor.go:57-63` fast path and
`processor.go:150-152` slow-path `fullHit`). It compares: `Level/JobId`, X/Y with a ±100
tolerance band, weapon templateId, consumable+cash asset quantities (by slot+templateId),
skill id→level set, and SOUL_ARROW/SHADOW_PARTNER buff presence
(`compareProjection`, `shadow.go:119-146`).

**Blocking: the buff-divergence branch is dead code.** Both call sites of `maybeShadow`
(`processor.go:61` and `processor.go:151`) pass `nil` for `servedBuffs`, and
`compareProjection` explicitly skips the buffs comparison whenever `snapBuffs == nil`
(`shadow.go:139`, "controller ruling R9"). There is no code path in this diff that ever
supplies a non-nil `servedBuffs` to `maybeShadow`. Design §8 explicitly lists "active
projectile-gate buffs" as part of what shadow verification compares — as landed, it never
does. `atlas_channel_char_snapshot_divergence_total{component="buffs"}` can never increment,
regardless of whether the buff snapshot (event-driven, per §6 below) actually diverges from
REST truth. This silently defeats exactly the safety net the brief asks about for the one
component (buffs) that has the documented "atlas-buffs pod-restart drops buffs silently"
residual risk (event-coverage.md §5) — the one place shadow verification would have the most
value, it cannot fire.

Other things shadow verification cannot see, by construction (not necessarily wrong, but
worth being explicit about since the brief asks): Hp/Mp (not compared — Sacrifice/heal math
reads Hp but design never claimed to cover it), inventory contents beyond
consumable/cash-slot quantities (e.g., equip stat drift on the equipped weapon itself, only
templateId is compared), and any read on a fallback/degraded path (shadow only samples
full-hit reads, by design — reasonable, since a fallback read is already REST-fresh).

Metrics: `snapshotDivergenceTotal`/`snapshotReadsTotal`/`snapshotUpdatesTotal` are registered
via `promauto` at package init (`character/snapshot/metrics.go:29-51`) and `/metrics` is
mounted in `main.go:370` via `promhttp.Handler()` — confirmed live, contradicting design.md
§8's claim of an `/api/metrics` mount path (doc says `/api/metrics`, landed code mounts
`/metrics`; cosmetic doc drift, non-blocking).

`CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE` is not set anywhere in this repository (searched the whole
tree); deployment/staging wiring lives in a separate images/manifests repo this diff does not
touch — **not evaluable** from this unit.

## 4. Fallback behavior

`Processor.Get` (`processor.go:78-88`): a **core** REST fallback failure returns the error to
`processAttack`, which returns it at `character_attack_common.go:876-879` — the swing is
aborted before any damage/broadcast/projectile side effect (all of those run after the `c, err
:= sp.Get(...)` call at the top of the closure). Covered directly by
`TestProcessor_FallbackFailureSurfacesError` (`processor_test.go:177`).

Inventory/skills REST fallback failures **degrade** (serve the model without that component
set) rather than fail the swing, matching today's `InventoryDecorator`/`SkillModelDecorator`
behavior exactly (comment at `processor.go:98-100,120-122`). Covered by
`TestProcessor_InventoryFallbackFailureServesDegradedModel` /
`TestProcessor_SkillsFallbackFailureServesDegradedModel`.

No test exercises `processAttack` itself end-to-end with a failing `sp.Get` (the handler
closure is largely untested as a whole, consistent with this codebase's convention of testing
extracted pure/deps-injected helpers rather than the full closure) — the propagation is a
one-line `if err != nil { return err }`, low-risk, and the unit-level fallback-failure test
above already proves the underlying contract. Non-blocking.

## 5. Skill-data TTL cache (`data/skill/cache.go`)

Negative caching is `errors.Is(err, requests.ErrNotFound)`-gated only
(`cache.go:193-196`); transient errors are never cached — correct. TTL clamps
(`[1s,24h]` positive, `[0s,5m]` negative) and kill-switch (`SKILL_DATA_CACHE_ENABLED`,
default true) are read once via `sync.Once` at first use (`getSkillCache`, `cache.go:109-117`)
— consistent with the rest of the codebase's config-at-boot convention. Cache key is
`skillId` only (not skillId+level) — correct, because `Model` carries the full per-level
effect table and `GetEffect` indexes into it after the cache lookup
(`processor.go:35-47`), so level-freshness is not a concern.

**Design-doc inaccuracy (non-blocking):** design.md §2.2 and event-coverage.md §7 both
describe the pre-task-122 skill-data path as "uncached REST on every lookup." That is
incorrect: `git show main:.../data/skill/registry.go` shows an existing unbounded, no-TTL,
positive-only, per-tenant `Cache` that `ProcessorImpl.GetById` already checked before every
REST call (`git show main:.../data/skill/processor.go:35-38`). The new TTL cache is a strict
improvement (adds negative caching + bounded TTL + a kill switch + tenant eviction, none of
which existed before), so this does not change the correctness verdict, but the design's
stated rationale/baseline for this change is wrong and should be corrected if this doc is
reused for future audits.

## 6. Buff snapshot (apply/expire/cancel reaching the snapshot; staleness across expiry)

`handleSnapshotBuffApplied`/`handleSnapshotBuffExpired`
(`kafka/consumer/buff/consumer.go:636-668`) are wired as additive handlers on the same
`character_buff_status_event` topic/consumer as the existing broadcast handlers — confirmed
registered in `InitHandlers` (`consumer.go:89-98`). `APPLY`/`CANCEL`/`CANCEL_ALL`/
`CANCEL_BY_TYPES` all route through atlas-buffs to `EXPIRED` (event-coverage.md §5, verified
against atlas-buffs `character/processor.go:80,95,122`), so all three of "apply, expiry,
cancel" reach the snapshot via the same two handlers — there is no separate CANCEL event type
to miss. `UpsertBuff`/`RemoveBuff` key by `SourceId` (`registry.go:595-634`), matching one
buff.Model per skill source with its own `Changes` slice — consistent with the event payload
shape.

Staleness across expiry: `GetBuffs` self-filters `Expired()` entries at read time
(`processor.go:186-195`, `filterActive`), and `buff.Model.Expired()` is computed from the
event's `ExpiresAt` compared against wall-clock `time.Now()` (not from an atlas-buffs push) —
so even a lost `EXPIRED` event bounds staleness to the buff's own remaining duration, matching
the documented residual risk in event-coverage.md §5. HP/MP watk/matk-affecting buff reads
are not on the attack path directly (effective stats, which include buffs, stay on REST per
event-coverage.md §6 — an explicit, audited non-goal, not a gap introduced by this diff).

**No blocking finding under this heading**, other than the shadow-verification interaction
already raised in §3 (the one place a stale/lost buff event would show up as a divergence
metric, it structurally cannot).

## Minor / non-blocking observations

- `Registry.ComposedIfValid` (`registry.go:135-160`) gates the fast composed-model cache on
  `buffsValid` even though buffs are not part of the returned `character.Model` (`Get()` never
  applies buffs to `m`). Any buff mutation invalidates `buffsValid` and starves the fast-path
  composed cache for that character until the next `GetBuffs` lazy-seed, even though the
  slower `View`-based path in `Get()` still serves zero-REST (`fullHit` there does not require
  `buffsValid`). Performance-only, not a correctness issue.
- design.md §8 says the metrics mount is at `/api/metrics`; landed code mounts `/metrics`
  (`main.go:370`). Cosmetic.

## Not evaluable

- `CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE` deployment wiring (staging soak) — lives in a
  separate images/manifests repo not part of this diff.
- Full end-to-end trace of character death/respawn → atlas-maps `ChangeMap` (touches
  `respawn/processor.go` and the saga orchestrator, neither modified by this diff) — traced to
  the saga dispatch point only, not confirmed to terminate in `ChangeMap` every time.
