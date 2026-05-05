# Monster-Buff Trust-but-Verify (Doom Handler Removal) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-04
---

## 1. Overview

The atlas-channel skill pipeline currently has two parallel paths that can both apply
monster status effects for the same skill cast:

1. The generic `applyToMobs` path in `services/atlas-channel/atlas.com/channel/skill/handler/common.go`,
   which iterates `SkillUsageInfo.AffectedMobIds()` (populated when the client packet is
   classified as a mob-affecting buff via `libs/atlas-packet/model/skill_usage_info.go:isMobAffectingBuff`).
2. The per-skill Doom handler in `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go`,
   which performs a server-authoritative `GetInMapRect` query, rolls `prop` per target, skips
   magic-reflect mobs, and emits `ApplyStatus` per survivor.

The Doom handler was built in task-047 because, at the time, the client packet did **not**
carry the affected mob IDs (Doom was not in `isMobAffectingBuff`). The server therefore had
to do its own bbox selection. That gap has since been closed (Doom is now in the
`isMobAffectingBuff` allowlist), so both paths fire on every Doom cast. The pod logs taken
during this task's interview confirm the duplication: three mobs in the rect, six
`Applying status to monster [...]` log lines per cast (path A before the rect query, path B
after), six byte-identical `MonsterStatSet` packets to the client. This is the proximate
source of the dual-apply bug.

This task removes the per-skill Doom handler entirely and lifts the server-authority
guarantees it provided (rect verification, mobCount cap, prop roll, kind-aware reflect
skip) into the generic `applyToMobs` path, so they apply uniformly to every mob-affecting
buff skill — present and future. Whenever the client packet diverges from server
expectations (mob ID outside the verified rect, count exceeds the skill's `mobCount`),
the channel emits a structured warn-level log with `tenant`, `world.id`, `channel.id`,
`character_id`, `skill_id`, `expected`, `received` so a future auto-ban subsystem can
consume the signal. The per-skill handler subpackage and its blank-import line in
`registrations.go` are deleted as part of this change.

## 2. Goals

Primary goals:

- Eliminate the dual-apply path for Doom (and any other skill listed in
  `isMobAffectingBuff`) by making `applyToMobs` the single emitter of mob-status applies
  for buff-classified skills.
- Move the four server-authority guarantees from the Doom handler into the generic path:
  (a) rect verification of `affectedMobIds`, (b) per-skill `mobCount` cap enforcement,
  (c) per-target `prop` roll, (d) kind-aware reflect skip.
- Surface client-server divergence as structured warn logs with sufficient fields for an
  auto-ban system to act on later.
- Delete the per-skill Doom handler subpackage and its registration without losing test
  coverage of the equivalent behaviors.

Non-goals:

- Building or designing the auto-ban subsystem itself. This task only emits the signals.
- Expanding `isMobAffectingBuff` to additional skills. Doom is the sole entry today; that
  list grows in future tasks as more skills are wired into the buff-stat path.
- Changing the per-skill dispatcher pattern (`Lookup` registry) for skills that legitimately
  need it (e.g., Heal, Cure, MPEater, Drain). Only the Doom handler is removed.
- Modifying atlas-monsters' apply-side filters (boss / elemental immunity stay where they
  live in `services/atlas-monsters/.../monster/processor.go`).
- Any client-side change. The v83 client behavior is treated as fixed.
- Reworking Crash / Dispel cancel semantics beyond what the trust-but-verify pass touches
  (rect verify, count cap, kind-aware reflect skip apply; prop roll for cancels is opt-in
  per skill — see FR-4.5).

## 3. User Stories

- As a channel server operator, I want a single, server-authoritative path for monster-status
  applies so that one cast produces exactly one `MonsterStatSet` per affected mob.
- As a future auto-ban subsystem, I want structured warn logs whenever the client claims to
  have hit a mob that the server's rect query rejects, so I can correlate divergence over
  time and flag accounts.
- As a channel-server developer adding a new monster-status skill, I want the generic
  `applyToMobs` path to enforce mobCount, prop, rect, and reflect uniformly, so I do not
  have to write a per-skill handler for each one.
- As a player, I want Doom to keep working at L30 (and other levels) with the same
  on-screen behavior I observed before the consolidation — namely, atlas-monsters has DOOM
  applied to every targeted in-rect mob, and the polymorph rendering is the same as today.

## 4. Functional Requirements

### 4.1 Rect verification (FR-4.1)

`applyToMobs` MUST construct the verification rect using the same formula the Doom
handler currently uses (`services/atlas-channel/.../skill/handler/doom/bbox.go`):

- Load the caster's `(X, Y, Stance)`.
- Derive `facingLeft = (stance & 1) == 1` (OdinMS convention).
- Compute `(x1, y1, x2, y2)` from `caster.X/Y`, `facingLeft`, `e.LT()`, `e.RB()`.
- Issue `monster.NewProcessor(l, ctx).GetInMapRect(f, x1, y1, x2, y2, e.MobCount())`
  to atlas-monsters and collect the resulting unique-id set.

`applyToMobs` MUST then compare the client-provided `affectedMobIds` against the
server-returned in-rect set:

- The intersection (mob IDs present in both lists) is the **applied set**.
- The exclusive client side (mob IDs the client sent but the server did NOT return) is
  the **anomaly set**; presence of any anomaly mob triggers the warn-log of FR-4.7.1
  but does NOT abort the cast — the applied set still proceeds.
- The exclusive server side (mob IDs in the rect that the client did NOT send) is
  intentionally ignored — the client's omission is treated as authoritative for "did
  not target".

### 4.2 Skills without an effect bbox (FR-4.2)

If `e.LT()` and `e.RB()` are both zero-valued (no bbox in the WZ effect), the verification
rect is undefined. In this case `applyToMobs` MUST fall back to "trust the client unmodified"
for the rect check — i.e., the applied set equals `affectedMobIds`. The mobCount cap
(FR-4.3), prop roll (FR-4.5), and reflect skip (FR-4.6) still apply. A debug-level log
SHOULD note the bbox-fallback so future auditing can identify these skills, but no warn
log is emitted (the client is not anomalous; the skill simply has no rect contract).

### 4.3 mobCount cap (FR-4.3)

If `len(affectedMobIds) > e.MobCount()`, `applyToMobs` MUST drop the entire cast
(no `ApplyStatus` or `CancelStatus` calls for any mob in the list) and emit the
warn-log of FR-4.7.2. This is treated as anomalous client behavior worthy of a hard stop
because no legitimate v83 client should send more targets than the skill's `mobCount`
permits.

This check runs **before** rect intersection so that an over-count cast is rejected
even if every claimed mob ID happens to be in-rect.

### 4.4 Applied-set ordering (FR-4.4)

The order of `ApplyStatus` / `CancelStatus` emissions inside the applied set MUST follow
the order of `affectedMobIds` in the client packet. We do not re-sort by distance or
unique id; preserving client order keeps the wire trace easy to correlate with packet logs.
The `mobCount` cap is enforced in FR-4.3 so this ordering does not interact with truncation.

### 4.5 Prop roll (FR-4.5)

For each mob in the applied set, `applyToMobs` MUST roll `e.Prop()` and only emit
`ApplyStatus` (or `CancelStatus`) if the roll succeeds. Roll mechanics mirror the
existing Doom handler:

- `prop <= 0` → always skip (effectively "off")
- `prop >= 1` → always pass
- otherwise `rand.Float64() <= prop` → pass

Per the interview, the prop roll applies to **both** the apply path
(`mp.ApplyStatus`) and the cancel path (`mp.CancelStatus`) by default, but the
implementation MUST allow a per-skill carve-out: a small in-package allow/deny table
keyed on `skill.Id` so that future skills whose WZ data prescribes "prop only on apply,
not cancel" (or vice versa) can be configured without re-architecting the path.
The initial table for the skills currently in `applyToMobs`
(Doom apply; Crash, Magic Crash, Power Crash, Priest Dispel cancels) is:

- DOOM apply → prop applies
- Crash family cancels → prop applies
- Priest Dispel cancel → prop applies

If a future skill needs a carve-out, the allow/deny table is the contract.

A prop-skipped target is not anomalous and does NOT emit a warn log; it is logged at
debug level only and counted in the per-cast summary (FR-4.8).

### 4.6 Kind-aware reflect skip (FR-4.6)

For each mob in the applied set, `applyToMobs` MUST consult
`monster.GetStatusMirror().GetReflect(t, mobId, kind)` (mirror seam already used by
`doom.go:113`) and skip the mob if a reflect of the matching kind is active. The skill's
"kind" is determined by:

- If the skill is in `isCrashOrDispel` (today: Crusader Armor Crash, White Knight Magic
  Crash, Dragon Knight Power Crash → `PHYSICAL`; Priest Dispel → `MAGICAL`), use the kind
  returned by `dispelSkillClass`.
- For mob-affecting buff applies that are NOT crashes/dispels (today: Doom), the kind is
  `MAGICAL`. A small `mobBuffApplyKind(skill.Id) string` helper MUST be introduced that
  returns `"MAGICAL"` for the current Doom entry; future apply-style status skills are
  added to this helper as they are wired in. If the helper returns `""` (unknown skill)
  the reflect check is skipped — the cast still proceeds — and a debug log notes the
  unclassified kind.

A reflect-skipped target is not anomalous and does NOT emit a warn log; it is logged at
debug level and counted in the per-cast summary (FR-4.8).

### 4.7 Anomaly warn logs (FR-4.7)

`applyToMobs` MUST emit a warn-level log entry on each of the following anomalies. All
warn logs MUST include the standard channel structured fields already present on
`logrus.FieldLogger` from the parent context (`tenant`, `world.id`, `channel.id`) plus
the anomaly-specific fields below.

#### 4.7.1 Client mob outside server rect

Triggered when one or more entries in `affectedMobIds` are NOT in the server's
`GetInMapRect` result. Fields:

- `event = "monster_buff_anomaly_out_of_rect"`
- `character_id`
- `skill_id`
- `skill_level`
- `rect = {x1, y1, x2, y2}`
- `mob_count_cap = e.MobCount()`
- `client_mob_ids = [...]` (entire client list, not just anomaly subset)
- `server_mob_ids = [...]` (entire server in-rect list)
- `anomaly_mob_ids = [...]` (set difference: client minus server)

Message: `"client_targeted_mob_outside_server_rect"`. Issued **once per cast** even if
multiple anomaly mobs are present.

#### 4.7.2 Client count exceeds mobCount cap

Triggered when `len(affectedMobIds) > e.MobCount()`. Fields:

- `event = "monster_buff_anomaly_over_cap"`
- `character_id`
- `skill_id`
- `skill_level`
- `mob_count_cap = e.MobCount()`
- `client_mob_count = len(affectedMobIds)`
- `client_mob_ids = [...]`

Message: `"client_target_count_exceeds_skill_cap"`. The cast is dropped (FR-4.3).

#### 4.7.3 No other anomaly cases warn

Empty `affectedMobIds`, empty server in-rect, prop misses, reflect skips, and
zero-intersection (when client list IS contained by server list but the intersection is
empty because the client sent zero IDs) are all NOT anomalies. Per the interview, the
specific signal to warn on is "client list is not contained by server query"; everything
else is normal.

### 4.8 Per-cast summary log (FR-4.8)

`applyToMobs` MUST emit one debug-level summary line per cast (mirroring the existing
`Doom: caster=... mobsInRect=... applied=... reflectSkipped=... propSkipped=...`
format). Fields (debug only):

- `caster = characterId`
- `skill_id`
- `skill_level`
- `mobs_in_rect` (server's in-rect count, or `-1` when bbox fallback per FR-4.2)
- `client_mob_count`
- `applied`
- `reflect_skipped`
- `prop_skipped`
- `out_of_rect_dropped`

Message: `"mob_buff_apply_summary"`. This replaces the existing Doom-handler summary.

### 4.9 Status emission (FR-4.9)

After the applied / prop / reflect filters above, `applyToMobs` MUST emit exactly one
of the following per surviving mob, mirroring the current paths in `common.go:75-104`:

- For Crash / Dispel skills (`isCrashOrDispel(sid)`): `mp.CancelStatus(f, mobId,
  nil, characterId, info.SkillId(), dispelSkillClass(sid))`
- For all other mob-affecting buffs: `mp.ApplyStatus(f, mobId, characterId, info.SkillId(),
  uint32(info.SkillLevel()), monsterStatuses, uint32(e.Duration()))`

Where `monsterStatuses` is the int32-cast map from `e.MonsterStatus()`, identical to
today's behavior.

A skill MUST NOT trigger both branches in the same cast. The mutual exclusion is
inherited from the existing `isCrashOrDispel` check.

### 4.10 Per-skill handler removal (FR-4.10)

The `services/atlas-channel/atlas.com/channel/skill/handler/doom/` subpackage MUST be
deleted in its entirety, including:

- `doom.go`
- `bbox.go`
- `doom_test.go`
- `bbox_test.go`

The blank import line in `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`
that reads `_ "atlas-channel/skill/handler/doom" // Priest Doom — task 047` MUST be
removed. The other handler imports (`heal/`) MUST be left intact.

The per-skill `Lookup` registry in `skill/handler/registry.go` is NOT removed — it
continues to serve Heal and any future per-skill handlers. Doom simply no longer
registers itself there because nothing in `doom/init()` survives the deletion.

### 4.11 Test migration (FR-4.11)

Test coverage equivalent to the deleted `doom_test.go` and `bbox_test.go` MUST land in
`services/atlas-channel/atlas.com/channel/skill/handler/common.go` as either:

- A new `common_apply_to_mobs_test.go` that replaces `doom_test.go` / `bbox_test.go`
  one-to-one; OR
- A new package `services/atlas-channel/atlas.com/channel/skill/handler/internal/mobselect/`
  housing the rect, intersection, cap, prop, and reflect helpers as small pure functions,
  each covered by its own `*_test.go`. `applyToMobs` then composes those helpers.

The implementer chooses which structure better matches the existing test seams. Either
way, the following behaviors MUST be pinned:

- Bbox math for left- and right-facing casters (the `calculateBoundingBox` cases).
- Rect intersection logic (anomaly set produces warn log; non-anomaly cast does not).
- mobCount cap enforcement (over-cap drops cast, emits warn log).
- Prop roll (0 → always skip; 1 → always pass; in-between honors injected RNG).
- Kind-aware reflect skip (Doom → MAGICAL; Crash family → PHYSICAL; Priest Dispel →
  MAGICAL).
- Cancel-vs-apply branching (Crash/Dispel hits `CancelStatus`; everything else hits
  `ApplyStatus`).

Test seams MUST be injected via package-level vars (the established Atlas pattern from
the existing `loadCasterFunc`, `propRollFunc`, `rectQueryFunc`, `applyStatusFunc`,
`reflectLookupFunc` in `doom.go`).

### 4.12 Backwards compatibility (FR-4.12)

After this change, casting Doom on a 3-mob pack MUST result in:

- Exactly 3 `ApplyStatus` Kafka commands emitted to atlas-monsters (one per mob).
- Exactly 3 `STATUS_APPLIED` events received back, producing exactly 3 `MonsterStatSet`
  packets to the casting client (one per mob).
- The atlas-monsters registry GET on each mob shows exactly one DOOM entry from the
  Priest skill 2311005 with the WZ-derived duration.

Whether the v83 client visually polymorphs all three is **out of scope** — that is the
known client-side gating issue from this conversation, separate from the dual-apply bug
this task fixes.

## 5. API Surface

This task does not add or modify any external (REST or Kafka) API. All changes are
internal to the atlas-channel skill pipeline.

The changes touch the following internal Go contracts:

- `services/atlas-channel/atlas.com/channel/skill/handler/common.go:applyToMobs` —
  signature unchanged; behavior extended.
- `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` —
  one import line removed.
- New private helpers (location decided by the implementer per FR-4.11) for: rect
  computation, intersection, cap enforcement, prop roll, reflect-kind classification.

The `mobBuffApplyKind(skill.Id) string` helper introduced in FR-4.6 is private to the
`handler` package.

## 6. Data Model

This task does not introduce any new persisted data. Memory-resident state is unchanged:

- `monster.StatusMirror` (singleton, per-tenant projection of monster status events) is
  consulted read-only via `GetReflect`, exactly as the deleted Doom handler did.
- atlas-monsters' status registry is updated via the existing `EnvCommandTopic`
  `ApplyStatus` / `CancelStatus` Kafka commands. No envelope changes.

No migrations are required.

## 7. Service Impact

### atlas-channel (primary)

Files modified:

- `services/atlas-channel/atlas.com/channel/skill/handler/common.go` — `applyToMobs`
  extended with rect verify, cap, prop, kind-aware reflect skip, summary + anomaly logs.
- `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` —
  drop `_ "atlas-channel/skill/handler/doom"` import.

Files deleted:

- `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go`
- `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox.go`
- `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom_test.go`
- `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox_test.go`

Files added:

- Either `services/atlas-channel/atlas.com/channel/skill/handler/common_apply_to_mobs_test.go`
  OR `services/atlas-channel/atlas.com/channel/skill/handler/internal/mobselect/*.go`
  (implementer's choice per FR-4.11).

### atlas-monsters

Unchanged. The `GetInFieldRect` REST endpoint and `ApplyStatus` / `CancelStatus`
Kafka command consumers are already in place from task-047. No filter logic moves
between services.

### libs/atlas-packet

Unchanged. `SkillUsageInfo.AffectedMobIds()` and `isMobAffectingBuff` are already
in place. (The Doom entry in `isMobAffectingBuff` stays — it is the prerequisite
that makes this task possible.)

### libs/atlas-constants

Unchanged.

## 8. Non-Functional Requirements

### 8.1 Performance

- The rect query (`GetInFieldRect`) is the same call the Doom handler already makes;
  total network volume per cast is unchanged (one rect query plus N apply commands).
- The prop roll, intersection, and kind-classification are all in-memory operations
  bounded by `e.MobCount()` (capped at 6 for Doom in v83 WZ data). Per-cast overhead is
  O(N) with a tiny constant.
- After consolidation, the channel emits **half as many** `ApplyStatus` Kafka commands
  per Doom cast as the current bug produces (the dual-apply is gone). Per-cast Kafka
  volume to atlas-monsters drops by 50% for every skill listed in `isMobAffectingBuff`.

### 8.2 Security / abuse signal

Anomaly warn logs (FR-4.7.1, FR-4.7.2) are designed as the input contract for a future
auto-ban system. The structured fields (`character_id`, `skill_id`, `skill_level`,
`rect`, `client_mob_ids`, `server_mob_ids`, `anomaly_mob_ids`) are stable and queryable
in Loki without further parsing.

The cast-drop on FR-4.3 means an over-cap client cannot apply DOOM (or any future buff)
to more mobs than the skill's WZ definition permits, even if the rest of the pipeline
trusts it.

### 8.3 Observability

The per-cast summary (FR-4.8) and anomaly logs (FR-4.7) follow the existing Atlas
ECS-flavored JSON log format and inherit `tenant`, `world.id`, `channel.id`, `service.name`,
`session`, `span.id`, `trace.id` from the request-scoped logger. No new log destination
or pipeline is required; existing Loki/Grafana queries that match `service="atlas-channel"`
will pick up the new fields automatically.

### 8.4 Multi-tenancy

`monster.GetStatusMirror().GetReflect(t, ...)` already takes the tenant model, mirroring
the deleted Doom handler. The `applyToMobs` path receives its tenant via
`tenant.MustFromContext(ctx)` exactly as today. No tenant-scoping regression is possible
because no global state is added.

### 8.5 Backwards compatibility

The wire format on every leg (client → channel; channel → atlas-monsters; atlas-monsters
→ channel; channel → client) is unchanged. The only observable difference for an
operator is: half as many duplicate commands/events/packets per Doom cast, and a new
debug summary + (when triggered) warn lines.

## 9. Open Questions

None left after the interview. The four trust-but-verify guarantees, mobCount overage
handling (drop + warn), prop scope (apply + cancel, with per-skill carve-out support
described in FR-4.5), kind-aware reflect, anomaly-warn condition (client list NOT
contained by server query), and test-migration approach (move-to-common OR extract
helpers) are all decided. The only intentionally deferred item is "future auto-ban
system" — explicitly out of scope.

## 10. Acceptance Criteria

- [ ] `services/atlas-channel/atlas.com/channel/skill/handler/doom/` folder no longer
      exists on the task branch.
- [ ] The `_ "atlas-channel/skill/handler/doom"` line is removed from `registrations.go`;
      the `_ "atlas-channel/skill/handler/heal"` line remains.
- [ ] `applyToMobs` in `common.go` performs rect verification (FR-4.1) using the
      caster-relative bbox formula from the deleted `doom/bbox.go`.
- [ ] `applyToMobs` enforces the `e.MobCount()` cap by **dropping** the cast and emitting
      the FR-4.7.2 warn log when `len(affectedMobIds) > e.MobCount()`.
- [ ] `applyToMobs` rolls `e.Prop()` per target before emitting `ApplyStatus` or
      `CancelStatus`, with per-skill carve-out support described in FR-4.5.
- [ ] `applyToMobs` skips mobs whose active reflect kind matches the cast's classified
      kind (Doom → MAGICAL; Crash family → PHYSICAL; Priest Dispel → MAGICAL).
- [ ] `applyToMobs` emits the FR-4.7.1 warn log once per cast when the client's mob list
      contains any ID not returned by the server's rect query, but proceeds with the
      intersection.
- [ ] `applyToMobs` emits the FR-4.8 debug summary line on every cast that reaches the
      mob-iteration step.
- [ ] Unit tests covering rect math, intersection, cap, prop, reflect kind, and
      apply-vs-cancel branching are present in either `common_apply_to_mobs_test.go` or
      a new `internal/mobselect/` subpackage; equivalent assertions to the deleted
      `doom_test.go` / `bbox_test.go` pass.
- [ ] `go build ./...` and `go test ./...` succeed in `services/atlas-channel/atlas.com/channel/`.
- [ ] Manually casting Doom on a 3-mob pack on the running cluster produces exactly 3
      `Applying status to monster [...]` log lines per cast (down from 6) and one
      `mob_buff_apply_summary` line.
- [ ] Manually crafting a client packet with 8 affected mob IDs on a skill with
      `mobCount=6` produces a `client_target_count_exceeds_skill_cap` warn log and zero
      `ApplyStatus` Kafka commands for that cast.
- [ ] Manually crafting a client packet that includes a mob ID outside the caster's
      effect rect produces a `client_targeted_mob_outside_server_rect` warn log AND
      proceeds with the intersection (so legitimate in-rect targets still take the
      effect).
