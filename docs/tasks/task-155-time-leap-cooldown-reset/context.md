# Time Leap Cooldown Reset (task-155) — Implementation Context

Companion to `plan.md`. Summarizes key files, decisions, and dependencies so an
implementer (or reviewer) can orient without re-reading the whole PRD/design.

## What this task does

Casting Buccaneer Time Leap (skill 5121010) clears every active skill cooldown
except Time Leap's own, for the caster and for in-range party members, and the
cleared timers disappear on each affected client. Reference behavior: Cosmic
`StatEffect.removeAllCooldownsExcept(TIME_LEAP)`.

Data flow: atlas-channel `timeleap.Apply` (per-skill handler, dispatched from
`UseSkill` after MP charge + Time Leap's own `SET_COOLDOWN`) resolves recipients
and emits one `RESET_COOLDOWNS` command per recipient → atlas-skills consumer →
`ResetCooldownsAndEmit` enumerates + clears registry cooldowns → one
`COOLDOWN_EXPIRED` status event per cleared skill → existing atlas-channel
`handleCooldownExpired` writes the cooldown-clear packet (skill id, time 0).

## Key files

### atlas-skills (`services/atlas-skills/atlas.com/skills`)
- `skill/cooldown_registry.go` — Redis-backed cooldown registry
  (`TenantRegistry`, composite key `<charId>:<skillId>`). Gains
  `GetAllForCharacter(ctx, characterId) (map[uint32]time.Time, error)` — client-side
  filter over `GetAllEntries` with the trailing-colon prefix invariant from
  `ClearAll` (lines 59-64).
- `skill/processor.go` — gains `ResetCooldowns(mb)` (buffer form, returns cleared
  skill ids) + `ResetCooldownsAndEmit`. Registry + events only; never touches the
  DB. Per-skill `Clear` failure → log + continue (partial success; re-delivery is
  a harmless no-op).
- `skill/producer.go` — `statusEventCooldownExpiredProvider` (line 129) reused
  as-is, now with real transactionId/worldId (the expiry ticker passes zeros).
- `kafka/message/skill/kafka.go` — gains `CommandTypeResetCooldowns` +
  `ResetCooldownsBody{ExceptSkillIds []uint32; SourceSkillId uint32}`.
- `kafka/consumer/skill/consumer.go` — gains `handleCommandResetCooldowns`,
  registered in `InitHandlers`; logs transaction_id/character_id/source_skill_id
  on failure.
- `skill/mock/processor.go` — gains the two new methods only. NOTE: the existing
  mock methods use a stale curried shape that predates transactionId/worldId in
  the interface — do not rewrite them in this task.

### atlas-channel (`services/atlas-channel/atlas.com/channel`)
- `kafka/message/skill/kafka.go` — mirror `Command[E]` gains
  `TransactionId uuid.UUID` + `WorldId world.Id` (design §3.4). Wire-compatible
  both directions: the skills-side decoder already has these fields, and the old
  field-less `SET_COOLDOWN` JSON decoded to the same zero values.
- `data/skill/producer.go` — gains `ResetCooldownsCommandProvider(...)`, key =
  `producer.CreateKey(int(characterId))` (per-character partition ordering with
  `SET_COOLDOWN`).
- `character/skill/processor.go` — gains
  `ResetCooldowns(transactionId, f, exceptSkillIds, sourceSkillId) model.Operator[uint32]`
  mirroring `ApplyCooldown`'s operator shape.
- `skill/handler/timeleap/timeleap.go` (new) — handler registered via `init()`
  for `skill2.BuccaneerTimeLeapId`; blank import added to
  `skill/handler/registrations/registrations.go`. Three test seams
  (`loadCaster`, `selectParty`, `emitReset`), pattern: `mysticdoor`.
- `skill/handler/recipients.go:94` — `SelectInRangePartyMembers` (existing,
  reused): returns nil on zero LT/RB rect before any I/O; bitmap-driven.
- `kafka/consumer/skill/consumer.go:143` — `handleCooldownExpired` (existing,
  UNCHANGED): status topic is consumed by every channel pod, so cross-pod party
  members are covered; each pod writes only to sessions it owns.

### Untouched (by design)
- Logout/death `ClearAll` path (`kafka/consumer/character/consumer.go:48,61` in
  atlas-skills) — stays event-less.
- `libs/atlas-redis`, `libs/atlas-packet`, `libs/atlas-constants` — no changes.
  `BuccaneerTimeLeapId = Id(5121010)` at `libs/atlas-constants/skill/constants.go:3212`;
  the party bitmap for 5121010 is already decoded
  (`libs/atlas-packet/model/skill_usage_info.go:171,221,272`).
- atlas-buffs — Time Leap has no statups (WZ v83 `Skill.wz/512.img.xml`:
  `lt=(-400,-300)`, `rb=(400,300)`, `cooltime` 2940→2700s, `mpCon` only).

## Key decisions (from design.md)

1. **Fan-out in atlas-channel** — one `RESET_COOLDOWNS` per recipient, keyed per
   character. atlas-skills stays party-agnostic (§3.1).
2. **Reuse `COOLDOWN_EXPIRED`** — one existing-shape event per cleared skill; no
   new event type, no new writer, no new channel consumer (§3.2, FR-11).
3. **Client-side prefix filter** over `GetAllEntries` — no `libs/atlas-redis`
   change; the expiry ticker already runs the same SCAN every tick (§3.3).
4. **Envelope upgrade** — channel's `Command[E]` mirror gains
   TransactionId/WorldId to match skills field-for-field; one `uuid.New()` per
   cast shared across the fan-out; `SetCooldownCommandProvider` call site left
   emitting zero values (§3.4).
5. **No per-member visual effect** — the socket handler already broadcasts the
   cast; cooldown timers clearing is the feedback (§3.5, resolves PRD OQ-1).
6. **`sourceSkillId` included** in the body, observability-only, never enters the
   skills processor signature; generic senders pass 0 (§3.5, resolves PRD OQ-2).
7. **Exclusion applies to every recipient** — a party-member Buccaneer keeps
   their own Time Leap cooldown (FR-3); correctness never depends on
   SET/RESET ordering (the exclusion list guarantees survival either way).

## Dependencies & gotchas

- Task order matters: 1 → 2 → 3 within atlas-skills, 4 → 5 within atlas-channel;
  the two services are otherwise independent (3 and 4 could swap).
- Consumer happy-path test asserts registry state only: `ResetCooldownsAndEmit`
  mutates the registry before the Kafka emit, and the emit errors in tests (no
  broker) — documented partial-success behavior, not a bug.
- Test-file style is mixed on purpose: `cooldown_registry_test.go` uses testify,
  `processor_test.go`/`consumer` tests use plain `t.Fatalf` — match each file.
- The new consumer test file must be internal (`package skill`) because handler
  funcs are unexported; the existing external `consumer_test.go` stays untouched.
- Builder pattern for test fixtures; no `*_testhelpers.go` (project rule).
- Verification gate: `go test -race`/`go vet`/`go build` in BOTH modules,
  `docker buildx bake atlas-skills atlas-channel` from the worktree root,
  `tools/redis-key-guard.sh` from the repo root (no `GOWORK=off` prefix).

## Verification commands

```bash
# module gates
(cd services/atlas-skills/atlas.com/skills   && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
# image gate (from worktree root)
docker buildx bake atlas-skills atlas-channel
# redis guard (from repo root)
tools/redis-key-guard.sh
```
