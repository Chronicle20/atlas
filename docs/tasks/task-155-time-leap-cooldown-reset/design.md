# Time Leap Cooldown Reset (Buccaneer 5121010) — Design

Task: task-155-time-leap-cooldown-reset
Status: Approved PRD → design
Date: 2026-07-10

## 1. Summary

Casting Time Leap must clear every active skill cooldown — except Time Leap's own — for the
caster and for in-range party members, with each cleared cooldown reflected on the affected
client. The design adds:

1. **atlas-channel**: a per-skill handler package `skill/handler/timeleap` that resolves
   recipients (caster + in-range party members) and emits one generic `RESET_COOLDOWNS`
   command per recipient on the existing skill command topic.
2. **atlas-skills**: a `RESET_COOLDOWNS` command type, a consumer handler, a processor method
   `ResetCooldowns` / `ResetCooldownsAndEmit`, and a per-character cooldown enumeration on the
   registry. Each cleared cooldown emits one existing-shape `COOLDOWN_EXPIRED` status event.
3. **No new client packets, no REST changes, no lib changes.** The existing
   `handleCooldownExpired` → `CharacterSkillCooldownWriter` (skill id, time 0) path clears the
   client UI.

Grounding verified during design:

- WZ data (v83 `Skill.wz/512.img.xml`, skill `5121010`): every level carries
  `lt=(-400,-300)` / `rb=(400,300)` and a `cooltime` (2940s at level 1 down to 2700s at
  level 5), `mpCon` only — no `statUps`, no `duration`. So the generic buff path in
  `UseSkill` never fires for Time Leap, and the rect-based party selector is the correct
  recipient filter.
- `libs/atlas-packet/model/skill_usage_info.go` already lists `skill.BuccaneerTimeLeapId` in
  the party-bitmap decode lists (lines 171, 221, 272), so `AffectedPartyMemberBitmap()` is
  populated for this skill on all supported versions.
- `UseSkill` (`services/atlas-channel/atlas.com/channel/skill/handler/common.go:93`) already
  charges MP and applies Time Leap's own cooldown via `ApplyCooldown` → `SET_COOLDOWN` before
  the per-skill dispatcher runs (line 117). The socket handler
  (`socket/handler/character_skill_use.go:108-110`) already broadcasts the cast effect to self
  and other sessions after `UseSkill` returns. The Time Leap handler therefore adds **only**
  the reset emission (PRD FR-4).

## 2. Architecture and Data Flow

```
client USE_SKILL(5121010, partyBitmap)
  └─ atlas-channel socket handler character_skill_use.go
       └─ UseSkill (common.go)
            ├─ MP charge, SET_COOLDOWN(5121010) [existing, unchanged]
            └─ per-skill dispatcher → timeleap.Apply
                 ├─ load caster (X, Y)
                 ├─ SelectInRangePartyMembers(bitmap, LT/RB rect)
                 └─ per recipient (caster + members):
                      emit Command RESET_COOLDOWNS{exceptSkillIds:[5121010], sourceSkillId:5121010}
                      (key = recipient characterId, shared transactionId per cast)
                          │  COMMAND_TOPIC_SKILL
                          ▼
atlas-skills consumer handleCommandResetCooldowns
  └─ Processor.ResetCooldownsAndEmit
       ├─ Registry.GetAllForCharacter (GetAllEntries + "<charId>:" prefix filter)
       ├─ per cooldown not in exceptSkillIds: Registry.Clear
       └─ buffer 1 COOLDOWN_EXPIRED status event per cleared skill
          (message.Buffer → message.Emit, atomic per recipient)
                          │  EVENT_TOPIC_SKILL_STATUS
                          ▼
atlas-channel handleCooldownExpired (existing, unchanged)
  └─ per session-present recipient: CharacterSkillCooldownWriter(skillId, 0)
```

Ordering note (PRD NFR): the caster's `SET_COOLDOWN` and `RESET_COOLDOWNS` both use
`producer.CreateKey(characterId)` on the same topic, so for the caster they are
partition-ordered (SET first). Correctness does not depend on this — the exclusion list
guarantees Time Leap's own cooldown survives in either order.

## 3. Alternatives Considered

### 3.1 Where the party fan-out happens

- **(chosen) atlas-channel emits one `RESET_COOLDOWNS` per recipient.** Recipient resolution
  needs party membership, the affected-member bitmap, live-session presence, and the LT/RB
  rectangle — all channel-local concerns already encapsulated in
  `SelectInRangePartyMembers`. atlas-skills stays party-agnostic. Per-recipient commands keep
  the per-character Kafka partition key (`CreateKey(characterId)`), preserving per-character
  ordering with `SET_COOLDOWN` and every other skill command.
- Rejected: one command carrying a recipient list. Fewer messages (≤6 saved per cast), but it
  breaks the one-character-per-command envelope every other skill command uses, breaks
  per-character partition keying, and forces atlas-skills to loop characters inside one
  message handler. Party casts are rare; the saving is negligible.
- Rejected: atlas-skills resolves the party itself. atlas-skills has no party, map, session,
  or skill-effect knowledge; importing those dependencies inverts service boundaries.

### 3.2 How clients learn about cleared cooldowns

- **(chosen) One existing-shape `COOLDOWN_EXPIRED` status event per cleared skill.**
  `handleCooldownExpired` (`services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer.go:143`)
  already turns each into a `CharacterSkillCooldownWriter(skillId, 0)` packet, and the status
  topic is consumed by every channel pod, so party members on other pods are covered with zero
  new consumer code. A Buccaneer with N cooldowns yields N small events — bounded by the
  character's skill count, same shape the background expiry ticker already produces in bulk.
- Rejected: a new batch `COOLDOWNS_RESET` event. Saves a few messages but requires a new event
  type mirrored in two services plus a new channel consumer handler that loops the same writer
  N times anyway (there is no batch client packet). New surface, no behavioral gain.

### 3.3 Registry enumeration

- **(chosen) Client-side prefix filter over the existing `TenantRegistry.GetAllEntries`.**
  New method `Registry.GetAllForCharacter(ctx, characterId) (map[uint32]time.Time, error)` in
  `services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go`: call
  `r.reg.GetAllEntries(ctx, t)` and keep suffixes with the `"<charId>:"` prefix (trailing
  colon — same safe-prefix invariant as `ClearAll`, `cooldown_registry.go:59-64`, so charId
  100 never matches 1000). No `libs/atlas-redis` change, no Dockerfile/CI surface.
- Rejected: a new `GetEntriesByPrefix` in `libs/atlas-redis`. Server-side MATCH is more
  efficient, but `GetAllEntries` is already a tenant-scoped SCAN the expiry ticker runs every
  tick over the same data; one extra scan per cast is negligible. A lib change costs a
  workspace-wide dependency bump for no measurable win. If a future feature needs hot-path
  prefix reads, promote it then.

### 3.4 Command envelope: transactionId / worldId from atlas-channel

atlas-channel's message mirror (`services/atlas-channel/atlas.com/channel/kafka/message/skill/kafka.go`)
predates the skills-side envelope: its `Command[E]` has **no** `TransactionId`/`WorldId`
fields, so today's `SET_COOLDOWN` arrives at atlas-skills with `uuid.Nil` / world 0 (Go zero
values on decode). The PRD's observability NFR requires transactionId propagation for the new
command.

- **(chosen) Add `TransactionId uuid.UUID` and `WorldId world.Id` to the channel mirror's
  `Command[E]` envelope**, matching the skills-side struct field-for-field. The Time Leap
  handler generates **one `uuid.New()` per cast** and passes it to every recipient's command,
  so a single cast's fan-out is correlatable across services. `WorldId` comes from
  `f.WorldId()`. The existing `SetCooldownCommandProvider` call site is left emitting zero
  values (unchanged behavior, now just explicit) — upgrading it is out of scope.
- Rejected: a RESET-only command struct outside the generic envelope. Two envelope shapes on
  one topic is exactly the kind of drift that produced the current mismatch.

### 3.5 PRD open questions — resolved

1. **Per-member visual effect: none.** The socket handler already broadcasts
   `AnnounceSkillUse` / `AnnounceForeignSkillUse` for every cast
   (`character_skill_use.go:108-110`); FR-4 forbids the handler duplicating that. The
   cooldown timers visibly clearing on each recipient's client is the essential feedback,
   delivered by the `COOLDOWN_EXPIRED` path. (Heal's internal re-broadcast is a heal-local
   quirk this handler must not copy.)
2. **Source skill id: yes, include it.** `ResetCooldownsBody` carries
   `sourceSkillId uint32` (5121010 here), used only for logging/observability in
   atlas-skills. One field, makes every reset attributable; generic consumers (admin tooling)
   may send 0.

## 4. Component Design

### 4.1 atlas-channel — `skill/handler/timeleap`

New package `services/atlas-channel/atlas.com/channel/skill/handler/timeleap/timeleap.go`,
registered via `init()` + a blank import added to
`skill/handler/registrations/registrations.go` (pattern: `heal`, `mysticdoor`).

`Apply` lifecycle (mirrors heal's shape, minus HP/XP/broadcast):

1. Load caster via `character.NewProcessor(l, ctx).GetById()(characterId)` — only X/Y are
   needed for the rectangle. On failure: log error, return `nil` (cast continues; per-step
   failures never abort the cast, same policy as heal).
2. Warn if the effect rectangle is missing (defensive only — WZ has it at every level):
   package-local `warnIfMissingRectangle` equivalent (the heal one is package-private to
   `heal`; duplicate the 6-line guard rather than widening `handler`'s API for two users —
   if a third handler needs it, hoist it to `handler` then).
3. `party := handler.SelectInRangePartyMembers(l, ctx, f, characterId, c.X(), c.Y(), e, info.AffectedPartyMemberBitmap())`
   — missing rect or bitmap 0 yields empty slice → caster-only (FR-5 satisfied for solo
   casts: the caster is always a recipient regardless of party state).
4. `transactionId := uuid.New()` (one per cast). `except := []uint32{uint32(skill2.BuccaneerTimeLeapId)}`
   — derived from the constant, applied to every recipient (FR-3: a party-member Buccaneer
   keeps their own Time Leap cooldown).
5. `reset := skillproc.NewProcessor(l, ctx).ResetCooldowns(transactionId, f, except, uint32(skill2.BuccaneerTimeLeapId))`
   then `_ = reset(characterId)` and `_ = reset(r.Id())` per party recipient. Emission
   failures are logged per recipient and do not abort remaining recipients.

### 4.2 atlas-channel — command plumbing

- `kafka/message/skill/kafka.go`: add `TransactionId`/`WorldId` to `Command[E]` (§3.4); add
  `CommandTypeResetCooldowns = "RESET_COOLDOWNS"` and
  `ResetCooldownsBody{ ExceptSkillIds []uint32; SourceSkillId uint32 }`.
- `data/skill/producer.go`: `ResetCooldownsCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32, sourceSkillId uint32)`
  next to `SetCooldownCommandProvider`, key = `producer.CreateKey(int(characterId))`.
- `character/skill/processor.go`: interface + impl method
  `ResetCooldowns(transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32) model.Operator[uint32]`
  mirroring `ApplyCooldown`'s operator shape (`processor.go:45-49`), emitting on
  `skill2.EnvCommandTopic`.

### 4.3 atlas-skills — message, consumer, processor, registry

- `kafka/message/skill/kafka.go`: `CommandTypeResetCooldowns = "RESET_COOLDOWNS"`,
  `ResetCooldownsBody{ ExceptSkillIds []uint32 "json:\"exceptSkillIds\""; SourceSkillId uint32 "json:\"sourceSkillId\"" }`.
- `kafka/consumer/skill/consumer.go`: `handleCommandResetCooldowns(db)` guarded on
  `c.Type != CommandTypeResetCooldowns` (pattern: `handleCommandSetCooldown`), registered in
  `InitHandlers`. Calls `ResetCooldownsAndEmit`; error path logs with
  transaction_id/character_id/source_skill_id fields.
- `skill/cooldown_registry.go`: `GetAllForCharacter` per §3.3. Returns
  `map[uint32]time.Time` (skillId → expiresAt); malformed suffixes are skipped (same
  tolerance as `GetAll`).
- `skill/processor.go`:
  - `ResetCooldowns(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32) ([]uint32, error)`
    — enumerate via `GetAllForCharacter`; for each skillId not in the exception set, call
    `GetRegistry().Clear`; on a per-skill clear error, log and continue (partial success
    beats none; re-delivery is a harmless no-op per PRD idempotency NFR). Buffer one
    `statusEventCooldownExpiredProvider(transactionId, worldId, characterId, skillId)`
    (existing provider, `producer.go:129`) **only for successfully cleared** skills. Returns
    the cleared skill ids. Empty enumeration or all-excepted → `([]uint32{}, nil)`, zero
    events (FR-9).
  - `ResetCooldownsAndEmit(...)` wraps it in `message.Emit(producer.ProviderImpl(...))`
    (pattern: `SetCooldownAndEmit`). Logs at info: character, cleared count, excepted ids
    (PRD observability NFR). `sourceSkillId` stays a body field logged by the consumer
    handler — it is observability-only and never enters the processor signature.
  - Note: unlike `SetCooldown`, this method never touches the DB (`db`/skill rows are not
    involved) — it is registry + events only. It still lives on the standard `Processor` so
    the consumer wiring stays uniform.
- `skill/mock/processor.go`: add both methods (FR-10).

### 4.4 Explicitly unchanged

- Logout/death `ClearAll` path (event-less, prefix-based) — untouched.
- `handleCooldownExpired` / `CharacterSkillCooldownWriter` — untouched, reused as-is.
- `libs/atlas-redis`, `libs/atlas-packet`, `libs/atlas-constants` — untouched.
- atlas-buffs — Time Leap has no statups (WZ-verified), nothing to do.

## 5. Error Handling

| Failure | Behavior |
|---|---|
| Caster load fails in handler | Log error, skip reset entirely, cast continues (heal policy) |
| Party/member load fails | `SelectInRangePartyMembers` already degrades to fewer/zero members |
| Command emission fails for one recipient | Log, continue with remaining recipients |
| Registry enumeration fails | Return error → consumer logs; no partial events emitted |
| Per-skill `Clear` fails | Log, skip that skill's event, continue with the rest |
| No active cooldowns / all excepted | Successful no-op: no events, no error (FR-9) |
| Malformed command | Existing consumer decode/discard behavior |

Multi-tenancy: consumer config already sets `TenantHeaderParser`; registry ops derive tenant
from context (`tenant.MustFromContext`) as today. The `COOLDOWN_EXPIRED` events carry the
command's real transactionId/worldId — an improvement over the expiry ticker's zero values,
using the same event shape.

## 6. Testing Strategy

Builder-pattern setup, no `*_testhelpers.go` (project rule).

**atlas-skills**
- `cooldown_registry_test.go` (miniredis, existing harness): `GetAllForCharacter` returns
  only the character's entries; prefix safety charId 100 vs 1000 vs 1001; skips malformed
  suffixes; empty result for unknown character.
- `processor_test.go`: `ResetCooldowns` with buffer — clears all but excepted, buffers one
  `COOLDOWN_EXPIRED` per cleared skill with correct transactionId/worldId/skillId; exclusion
  keeps 5121010; no-op case buffers nothing and returns no error; multiple exceptions.
- `consumer_test.go`: `handleCommandResetCooldowns` type-guard (wrong type → untouched
  registry), happy path against miniredis-backed registry.
- Regression: existing `ClearAll` consumer tests must still pass unchanged.

**atlas-channel**
- `timeleap_test.go`: seam-based (package-level func vars, pattern used throughout
  `skill/handler`): caster-load seam, party-selection seam, and an emission seam capturing
  `(characterId, exceptSkillIds, sourceSkillId, transactionId)` per call. Cases: solo cast →
  exactly one command for the caster; party cast → one command per in-range member + caster,
  all sharing one transactionId, all carrying `except=[5121010]`; caster-load failure → zero
  commands, no panic; missing rect → caster-only.
- Registry lookup test: `Lookup(skill.BuccaneerTimeLeapId)` resolves after blank import
  (pattern: existing `registry_test.go`).

**Verification gate (PRD acceptance):** `go test -race ./...`, `go vet ./...`,
`go build ./...` in both modules; `docker buildx bake atlas-skills atlas-channel`;
`tools/redis-key-guard.sh` clean from repo root (all Redis access stays inside the existing
registry wrapper).

## 7. Risks

- **Envelope change ripple (§3.4):** adding fields to the channel's `Command[E]` changes the
  JSON of existing `SET_COOLDOWN` messages (two new zero-valued fields). The skills-side
  decoder already has these fields, so this is wire-compatible in both directions during a
  mixed-version deploy.
- **Event volume:** worst case ≈ one event per skill on cooldown per recipient (practically
  < 10 per recipient; Time Leap's own ~45–49 min cooldown gates cast frequency). The expiry
  ticker already emits the same event shape in bulk.
- **Cross-pod party members:** covered — commands are keyed per recipient and the status
  topic is consumed by every channel pod; each pod's `handleCooldownExpired` writes only to
  sessions it owns (`IfPresentByCharacterId`).
