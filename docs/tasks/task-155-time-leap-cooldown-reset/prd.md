# Time Leap Cooldown Reset (Buccaneer 5121010) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Time Leap (Buccaneer 4th-job skill, id 5121010) is a party utility skill whose entire effect is
server-side: when cast, it removes every active skill cooldown from the caster and from in-range
party members — except Time Leap itself. The reference implementation is Cosmic
`StatEffect.java:1090-1091` (`applyto.removeAllCooldownsExcept(TIME_LEAP)`), applied to the caster
and to each party member the party-buff application reaches.

Atlas currently has no skill-triggered cooldown-reset path at all. atlas-skills owns cooldowns in a
Redis-backed registry and exposes `ClearAll`, but that is only invoked from the character
logout/death Kafka consumers (`services/atlas-skills/atlas.com/skills/kafka/consumer/character/consumer.go:48,61`).
On the atlas-channel side, the skill-use pipeline decodes Time Leap's affected-party bitmap
(`libs/atlas-packet/model/skill_usage_info.go` includes `BuccaneerTimeLeapId` in the party-skill
decode lists) and then discards it — casting Time Leap today charges MP and applies Time Leap's own
cooldown, and nothing else happens.

This task adds the skill-triggered reset path end to end: a per-skill handler in atlas-channel that
resolves recipients (caster + in-range party members), a new `RESET_COOLDOWNS` command in
atlas-skills that clears a character's cooldowns except an exclusion list, and per-skill
`COOLDOWN_EXPIRED` status events so each recipient's client cooldown UI actually clears. Scope: S.

## 2. Goals

Primary goals:

- Casting Time Leap clears all of the caster's active cooldowns except Time Leap's own.
- In-range party members (same world/channel/map, within the skill rectangle, selected by the
  affected-party bitmap) get the same reset, also excepting Time Leap.
- Each cleared cooldown is reflected on the affected client (cooldown timer disappears) via the
  existing `COOLDOWN_EXPIRED` → cooldown-writer path.
- The reset command is generic (`RESET_COOLDOWNS` with an exclusion list), not Time-Leap-specific,
  so future skills or admin tooling can reuse it.

Non-goals:

- No change to the logout/death `ClearAll` path (it stays event-less and prefix-based).
- No client packet additions — the existing cooldown writer (skill id + time 0) is reused.
- No buff/stat application: Time Leap grants no statups; this task does not touch atlas-buffs.
- No handling of other cooldown-adjacent skills (Hero's Will, Battleship cooldown carve-outs —
  separate backlog items).

## 3. User Stories

- As a Buccaneer, I want casting Time Leap to reset my other skills' cooldowns so that I can chain
  long-cooldown skills the way the original game allows.
- As a party member standing near a Buccaneer who casts Time Leap, I want my cooldowns reset so
  that the party benefits as designed.
- As a party member on another map or out of range, I must NOT receive the reset.
- As a Buccaneer, I must not have Time Leap's own just-applied cooldown wiped by its own effect
  (nor a party member's own Time Leap cooldown, if they are also a Buccaneer).

## 4. Functional Requirements

### 4.1 atlas-channel: Time Leap skill handler

- FR-1: Register a handler for `skill.BuccaneerTimeLeapId` in the per-skill handler registry
  (`services/atlas-channel/atlas.com/channel/skill/handler/registry.go`, registered in
  `handler/registrations/registrations.go`), following the existing heal handler layout
  (`handler/heal/heal.go`).
- FR-2: On use, resolve recipients as caster + in-range party members using the existing
  `SelectInRangePartyMembers(l, ctx, f, characterId, x, y, e, info.AffectedPartyMemberBitmap())`
  pattern from the heal handler (same channel + map, LT/RB rectangle from skill effect data,
  affected-party bitmap from the decoded usage info).
- FR-3: For each recipient, emit one `RESET_COOLDOWNS` command to the atlas-skills command topic
  with `exceptSkillIds: [5121010]`. The exclusion applies to every recipient (a party-member
  Buccaneer keeps their own Time Leap cooldown), matching Cosmic's `removeAllCooldownsExcept`.
- FR-4: The handler must not bypass or duplicate the common skill-use pipeline's existing
  responsibilities (MP charge, Time Leap's own cooldown application via `SET_COOLDOWN`, cast-effect
  broadcast). It only adds the reset emission.
- FR-5: Solo cast (no party, or no members in range) still resets the caster's cooldowns.

### 4.2 atlas-skills: RESET_COOLDOWNS command

- FR-6: Add `CommandTypeResetCooldowns = "RESET_COOLDOWNS"` with body
  `{ exceptSkillIds: []uint32 }` to `kafka/message/skill/kafka.go`, mirrored in atlas-channel's
  message definitions.
- FR-7: Add a consumer handler in `kafka/consumer/skill/consumer.go` alongside
  `handleCommandSetCooldown`, dispatching to a new processor method.
- FR-8: Processor method `ResetCooldownsAndEmit(transactionId, worldId, characterId, exceptSkillIds)`:
  - Enumerates the character's active cooldowns. The registry gains a per-character enumeration
    built on the existing `TenantRegistry.GetAllEntries` (filtering suffixes by the
    `"<charId>:"` prefix, same safe-prefix invariant as `ClearAll`). No `libs/atlas-redis` change
    is required.
  - Clears each cooldown not in `exceptSkillIds` via the existing per-skill `Clear`.
  - Emits one `COOLDOWN_EXPIRED` status event per cleared skill (reusing
    `statusEventCooldownExpiredProvider`, with the command's transactionId/worldId rather than the
    background-expiry zero values), batched via `message.Buffer` / `message.Emit` per the project's
    atomic-emission pattern.
- FR-9: A reset with no active cooldowns (or all excepted) is a successful no-op — no events, no
  error.
- FR-10: Update `skill/mock/processor.go` with the new interface method.

### 4.3 Client feedback

- FR-11: No new writers. atlas-channel's existing `handleCooldownExpired` consumer
  (`kafka/consumer/skill/consumer.go:143`) already writes the cooldown-clear packet per skill; the
  per-skill `COOLDOWN_EXPIRED` events from FR-8 drive it for every affected session, including
  party members on other channel pods (the status-event topic is already consumed channel-wide).

## 5. API Surface

No REST changes.

Kafka additions (command topic `EnvCommandTopic` for skills, existing):

```
Command[ResetCooldownsBody] {
  transactionId: uuid,
  worldId:       byte,
  characterId:   uint32,
  type:          "RESET_COOLDOWNS",
  body: {
    exceptSkillIds: []uint32
  }
}
```

Status events: no new types — existing `COOLDOWN_EXPIRED` (`StatusEventCooldownExpiredBody`) is
emitted once per cleared skill.

Error cases: unknown character or empty cooldown set → no-op (FR-9). Malformed command → existing
consumer decode/discard behavior.

## 6. Data Model

No database changes. No Redis schema changes — the cooldown registry keeps its
`<charId>:<skillId>` composite-key layout; this task only adds a read-enumerate-then-clear path
over existing keys. All Redis access stays inside the existing registry wrapper types
(`tools/redis-key-guard.sh` must remain clean).

## 7. Service Impact

- **atlas-skills** — new command type + consumer handler; new processor interface method
  `ResetCooldowns` / `ResetCooldownsAndEmit`; registry per-character enumeration method; mock
  update; unit tests.
- **atlas-channel** — new per-skill handler package `skill/handler/timeleap` (name per existing
  convention, cf. `heal`, `mysticdoor`); registration entry; producer for the new command;
  mirrored message body type; unit tests for recipient selection and command emission.
- **libs/** — none. `skill.BuccaneerTimeLeapId` already exists in `libs/atlas-constants`;
  `skill_usage_info.go` already decodes the party bitmap for 5121010; `libs/atlas-redis` already
  provides `GetAllEntries`.

## 8. Non-Functional Requirements

- **Multi-tenancy:** all paths derive tenant from context (`tenant.MustFromContext`); the consumer
  uses the standard tenant header parser; registry operations are tenant-scoped as today.
- **Ordering/atomicity:** the caster's `SET_COOLDOWN` (Time Leap's own cooldown) and
  `RESET_COOLDOWNS` travel on the same command topic; correctness must not depend on their relative
  order — the exclusion list guarantees Time Leap's cooldown survives regardless.
- **Idempotency:** re-delivering `RESET_COOLDOWNS` is harmless (second pass finds no cooldowns).
- **Observability:** log at info per reset (character, cleared count, excepted ids); propagate
  transactionId through command and status events.
- **Performance:** enumeration is a tenant-scoped Redis SCAN already used by the expiry ticker;
  one reset per recipient per cast is negligible load.

## 9. Open Questions

- Should party recipients see a per-member "affected by skill" visual effect (as some party buffs
  do)? Cosmic's party-buff apply shows one; the heal handler is the local precedent to check during
  design. If heal broadcasts nothing extra, Time Leap follows suit (the cooldown timers clearing is
  the essential feedback).
- Whether `RESET_COOLDOWNS` should carry the source skill id for observability (nice-to-have;
  decide at design time).

## 10. Acceptance Criteria

- [ ] Buccaneer with ≥2 skills on cooldown casts Time Leap: all cooldowns except Time Leap's clear
      server-side (registry) and on the client UI without relog.
- [ ] Time Leap's own cooldown is applied and survives the reset.
- [ ] In-range same-map party member's cooldowns clear (verified with a second character); their
      own Time Leap cooldown (if any) survives.
- [ ] Party member on a different map or outside the skill rectangle is unaffected.
- [ ] Solo cast resets the caster's cooldowns.
- [ ] Logout/death `ClearAll` behavior unchanged (existing consumer tests still pass).
- [ ] Unit tests: processor reset-with-exclusions (incl. no-op case, per-skill event emission),
      registry per-character enumeration prefix safety (charId 100 vs 1000), channel handler
      recipient selection + command emission.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-skills and
      atlas-channel modules; `docker buildx bake atlas-skills atlas-channel` clean;
      `tools/redis-key-guard.sh` clean from repo root.
