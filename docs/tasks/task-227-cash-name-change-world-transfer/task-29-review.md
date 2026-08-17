# Task 29 review: atlas-guilds consumes `NAME_CHANGED`

Commit reviewed: `41a2d7e98` (5 files, +197/-0)

Verdict: **PASS** — no blocking findings.

## 1. Producer/consumer contract

**PASS.** Diffed field-by-field:

- `services/atlas-character/.../kafka.go:235` — `StatusEventTypeNameChanged = "NAME_CHANGED"`
  vs. `services/atlas-guilds/atlas.com/guilds/kafka/message/character/kafka.go:17` —
  `EventCharacterStatusTypeNameChanged = "NAME_CHANGED"`. Literal string matches;
  only the Go identifier differs, per the controller addenda's explicit instruction
  to follow this service's local `EventCharacterStatusType*` naming convention (the
  three pre-existing sibling constants — `Deleted`, `Login`, `Logout`,
  `ChannelChanged` — already use that prefix).
- `services/atlas-character/.../kafka.go:359` — `StatusEventNameChangedBody{OldName
  string \`json:"oldName"\`; NewName string \`json:"newName"\`}` vs.
  `kafka/message/character/kafka.go:48-51` — identical field names, types, and json
  tags.
- Envelope (`StatusEvent[E]`) was already present and unchanged in the consumer
  file; its fields (`TransactionId`, `WorldId`, `CharacterId`, `Type`, `Body`)
  already matched the producer's envelope shape before this commit.

**Topic env var**: both sides use `EVENT_TOPIC_CHARACTER_STATUS`
(`EnvEventTopicCharacterStatus` producer-side at kafka.go:223; same constant name
and value consumer-side at kafka.go:12). `InitConsumers` in
`kafka/consumer/character/consumer.go:24-28` (unchanged by this commit) already
subscribes using this env var.

**Deploy config declaration** — checked, this is NOT the earlier-task gap the brief
warned about. `EVENT_TOPIC_CHARACTER_STATUS` is declared as a **global** topic env
var, not per-service, in all four deploy files:
- `deploy/compose/.env.example:78`
- `deploy/k8s/base/env-configmap.yaml:105`
- `deploy/k8s/overlays/pr/kustomization.yaml:266`
- `deploy/k8s/overlays/main/kustomization.yaml:142`

atlas-guilds picks it up via `env_file: .env` on the shared `x-atlas-infra` anchor
in `deploy/compose/docker-compose.core.yml` (no per-service topic env block exists
for any service using this global topic — atlas-guilds' own compose block only
lists `LOG_LEVEL`/`DB_NAME`). This is consistent with the topic already being
consumed by three pre-existing handlers in this same file before this task, so
there is no new deploy gap introduced here.

## 2. Handler registration

**PASS.** `kafka/consumer/character/consumer.go:44-46` registers
`handleCharacterNameChanged(db)` inside the existing `InitHandlers` body,
alongside the three pre-existing `rf(t, message.AdaptHandler(message.PersistentConfig(...)))`
calls (Deleted, Login, Logout) — same topic variable `t`, same wrapping pattern.
The type guard is present at consumer.go:100 (`if e.Type !=
character2.EventCharacterStatusTypeNameChanged { return }`), matching the shape of
the three sibling handlers exactly (e.g. `handleStatusEventLogin` at consumer.go:73).

Controller addenda point 3 (no `main.go` change needed) is also correct —
`main.go` is untouched in this commit, and `InitConsumers`/`InitHandlers` wiring
was already in place from before this task.

## 3. Tests

**PASS.** `kafka/consumer/character/consumer_test.go` (new, 150 lines):

- `TestNameChangedUpdatesTheGuildRosterName` — seeds a `member.Entity` row directly
  via `db.Create`, invokes the handler, asserts the persisted row's `Name` column
  via a direct `db.Where(...).First(&e)` query (not a mock, not "handler was
  called"). Would fail if the handler were deleted (compile break) or if
  `UpdateName`/`updateName` silently no-opped.
- `TestNameChangedIsIdempotent` — invokes the handler twice with the identical
  event, asserts the row still reads `"Zulu"`. The UPDATE is idempotent by
  construction (no INSERT, no counter), so this is a meaningful redelivery check —
  it would catch a hypothetical append-only or counter-based implementation.
- `TestNameChangedForANonMemberIsANoOp` — invokes the handler for a character with
  no seeded row, asserts `guildMemberRowCount(t, db) == 0`. Confirms the UPDATE
  does not error and does not fall back to an INSERT/upsert for a non-member.

All three assert on real DB state read back through a fresh query, not on
call-was-made spies. No `newTestDB` helper (that name was the plan snippet's
invention, per the brief) — the test file builds its fixtures directly following
`guild/processor_test.go`'s pattern (`setupTestTenant`/`setupTestContext`/
`setupTestLogger` + local `setupTestDatabase`), with `member.Migration(db)` run
against an in-memory sqlite DB. No `*_testhelpers.go` file was created — the
helpers live in `consumer_test.go` itself, consistent with CLAUDE.md's Test
Helper Pattern guidance.

## 4. The two recorded decisions

**(a) Tenant-scoped UPDATE — PASS, and siblings untouched.**
`guild/member/administrator.go:38-46` — `updateName` filters
`WHERE tenant_id = ? AND character_id = ?`, unlike `updateStatus` (line 26-29) and
`updateTitle` (line 31-35), which filter on `character_id` alone. A `git diff`
confirms `updateStatus`/`updateTitle` are byte-for-byte unmodified in this commit
— only `updateName` was added, with an explanatory comment pointing at the
divergence so a future reader won't "fix" it to match the siblings. Comment
references "task-227 Phase F decision log," which is a real section
(`docs/tasks/task-227-cash-name-change-world-transfer/plan.md:2587`, "Phase F —
`NAME_CHANGED` consumers"; `RESUME.md:231`, "Phase F warning (Tasks 29–32)").

**(b) Nothing emitted to Kafka — PASS.** Grepped
`services/atlas-guilds/atlas.com/guilds/kafka/message/guild/kafka.go`'s full
`StatusEventType*` const block (lines 106-117): `Created`, `Disbanded`,
`EmblemUpdated`, `RequestAgreement`, `MemberStatusUpdated`, `MemberTitleUpdated`,
`MemberLeft`, `MemberJoined`, `NoticeUpdated`, `CapacityUpdated`, `TitlesUpdated`,
`Error`. No `MemberNameUpdated`/`MEMBER_NAME_UPDATED` exists, confirming the
report's claim that emitting one would require inventing a new event type. The
handler at consumer.go:94-113 does not call any `...AndEmit` method or touch the
guild package's Kafka producer path — it only calls
`member.NewProcessor(l, ctx, db).UpdateName(...)`, which is a plain DB update with
no Kafka production. Consistent with design.md's Phase F description (only a
consumer table is listed for atlas-guilds, no new produced event).

## 5. DOM-* guideline conformance

**PASS**, with one minor non-blocking observation.

- **Layering**: `handleCharacterNameChanged` → `member.NewProcessor(l, ctx,
  db).UpdateName` → package-private `updateName(db, tenantId, characterId, name)`
  in `administrator.go`. This is the identical shape to the pre-existing
  `UpdateStatus`/`updateStatus` and `UpdateTitle`/`updateTitle` pairs
  (`processor.go:79-85`, `administrator.go:26-35`) — consumer → processor →
  administrator, no layer-skipping.
- **Tenant handling from context**: `ProcessorImpl.t` is populated from
  `tenant.MustFromContext(ctx)` in `NewProcessor` (processor.go:33), and
  `UpdateName` passes `p.t.Id()` through to the tenant-scoped `updateName` query.
  No tenant value is taken from the event body or any other untrusted source.
- **Immutable models**: no `Model` mutation added; `updateName` operates on
  `Entity` via a scoped GORM `Update`, matching the existing sibling functions'
  pattern in this file. `member/entity.go`'s `Entity`/`Model` split is unchanged.
- **No reinvented constants**: `world.Id` is used from
  `github.com/Chronicle20/atlas/libs/atlas-constants/world` in the test file
  (consumer_test.go:18), not a locally reinvented type.
- **No test-only constructors / `*_testhelpers.go`**: confirmed above, no such
  file was added.
- **Minor, non-blocking**: `handleCharacterNameChanged` calls
  `tenant.MustFromContext(ctx)` a second time (consumer.go:106) purely for the
  success-path log fields, when `member.NewProcessor` already resolved the same
  tenant internally one line earlier. This duplicates a context lookup but is
  consistent with how `handleStatusEventDeleted`/`Login`/`Logout` are written in
  this same file (none of them thread tenant through for logging, so this
  handler's logging is actually slightly more thorough than its siblings, not
  less careful). Not a correctness issue; flagging only as something a future
  refactor could tidy.

## Files reviewed

- `services/atlas-guilds/atlas.com/guilds/kafka/message/character/kafka.go`
- `services/atlas-guilds/atlas.com/guilds/kafka/consumer/character/consumer.go`
- `services/atlas-guilds/atlas.com/guilds/kafka/consumer/character/consumer_test.go`
- `services/atlas-guilds/atlas.com/guilds/guild/member/administrator.go`
- `services/atlas-guilds/atlas.com/guilds/guild/member/processor.go`
- (cross-referenced) `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go`
- (cross-referenced) `services/atlas-guilds/atlas.com/guilds/kafka/message/guild/kafka.go`
- (cross-referenced) `deploy/compose/.env.example`, `deploy/compose/docker-compose.core.yml`,
  `deploy/k8s/base/env-configmap.yaml`, `deploy/k8s/overlays/{pr,main}/kustomization.yaml`
- (cross-referenced) `docs/tasks/task-227-cash-name-change-world-transfer/plan.md` (Phase F),
  `RESUME.md`, `design.md`
