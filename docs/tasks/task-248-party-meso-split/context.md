# task-248-party-meso-split — Implementation Context

Companion to [plan.md](./plan.md). Everything here was verified against the tree
at the branch head; nothing is remembered or inferred.

---

## Key files

### `atlas-drops` (module root `services/atlas-drops/atlas.com/drops`)

| Path | Role |
|---|---|
| `drop/processor.go:135-147` | `Reserve` — the single evaluation point. `GetRegistry().ReserveDrop` already serializes per drop, so the split runs at most once. |
| `drop/producer.go:100-122` | `reservedEventStatusProvider` — the template for `mesoAwardedEventStatusProvider`, including `producer.CreateKey(int(d.Id()))`. |
| `kafka/message/drop/kafka.go:57-66` | `StatusEvent[E]` — already carries `TransactionId`, `Instance`, `DropId`. No envelope change needed on the producing side. |
| `drop/model.go:74,149` | `Meso()` and `Field()` — the two inputs `Reserve` reads off the reserved drop. |
| `drop/processor_test.go:21-40` | miniredis registry + tenant-context harness reused by the new `Reserve` tests. |
| `kafka/message/message.go:11-32` | `Buffer.GetAll() map[string][]kafka.Message` — how the tests read back what `Reserve` put. |

### `atlas-character` (module root `services/atlas-character/atlas.com/character`)

| Path | Role |
|---|---|
| `character/processor.go:928-950` | `AttemptMesoPickUp` — removed. Its `txErr != nil` early return is the bug FR-18 fixes. |
| `character/processor.go:880-925` | `RequestChangeMeso` — the transaction + `outbox.EmitProvider` shape the new method copies, and the source of the deliberate-asymmetry comment style. |
| `character/producer.go:151-166` | `mesoChangedStatusEventProvider(transactionId, characterId, worldId, amount int32, actorId uint32, actorType string, showEffect bool)`. |
| `kafka/consumer/drop/consumer.go:42-52` | `handleDropReservation` — deleted. Note it currently fabricates a transaction id with `uuid.New()` and builds the field without `Instance`. |
| `kafka/message/drop/kafka.go:57-65` | The mirror `StatusEvent[E]`, currently missing `TransactionId`. |
| `character/meso_outbox_test.go:20-40` | `outboxTestDb`, `createTestCharacter`, `outboxRowCount`. |
| `character/meso_overflow_test.go:24-43` | `producertest.InstallCapturing` + the balance setup that actually crosses `MaxUint32`. |
| `character/processor_test.go:24,58,63` | `testDatabase`, `testTenant`, `testLogger`. |

### Copy sources in other services (read-only)

| Path | What to copy |
|---|---|
| `services/atlas-monster-death/atlas.com/monster/party/{requests,processor}.go` + `mock/processor.go` | The minimal read-only `PARTIES` client. `atlas-drops → atlas-parties` is the same edge monster-death already has. |
| `services/atlas-channel/atlas.com/channel/party/rest.go:16-170` | JSON:API `members` relationship plumbing and `ExtractMember`. |
| `services/atlas-channel/atlas.com/channel/party/processor.go:99-104` | `MemberInMap` — the four-dimension + online predicate `splitMeso` re-expresses. |
| `services/atlas-mounts/atlas.com/mounts/mount/processor.go:48-63` | `ProcessorOption` / `With`. |
| `services/atlas-parties/atlas.com/parties/party/rest.go:95-103` | The authoritative member field names on the wire (`worldId`, `channelId`, `mapId`, `instance`, `online`). |

---

## Decisions made at plan time

### 1. No deploy change is required — design §9's manifest note is wrong

Design §9 says `atlas-drops` needs an `atlas-parties` service-discovery entry in
the compose and k8s manifests. It does not. Verified:

- `requests.RootUrlFor` (`libs/atlas-rest/requests/url.go:34-41`) falls back to
  the shared `BASE_SERVICE_URL` when no `PARTIES_SERVICE_URL` override is set.
- `deploy/k8s/base/atlas-drops.yaml` has no per-domain URL env at all — it takes
  the shared `atlas-env` configMap, exactly like every other service.
- `deploy/k8s/base/atlas-monster-death.yaml` likewise carries no `PARTIES` entry,
  and monster-death already makes this exact call.
- The shared ingress already routes it: `deploy/shared/routes.conf:16-17`,
  `location ~ ^/api/parties(/.*)?$ → atlas-parties:8080`.

No task touches `deploy/`. The rollout ordering in design §9 still stands.

### 2. An award above `math.MaxInt32` is rejected

Not in the design. `MESO_CHANGED`'s `Amount` is `int32`
(`kafka/message/character/kafka.go:131`), so `int32(meso)` for `meso >
math.MaxInt32` would emit a negative delta while the balance moved up. The plan
adds an explicit guard returning `ErrMesoOverflow` before the `uint32` balance
guard, with a test. Unreachable in practice — no drop carries three billion
meso — but the alternative is a silently wrong event.

### 3. `actorTypeDrop = "DROP"` is a local constant, not a shared one

`libs/atlas-constants/` has no actor-type constants; the existing values
(`"SYSTEM"`, `"CHARACTER"`, `"ITEM"`) are bare literals at their call sites.
Introducing a shared constants home for them is out of scope, so the plan
declares `actorTypeDrop` next to the method that uses it.

### 4. `services/atlas-drops/atlas.com/drops/drop/mock/processor.go` is left alone

Adding `With` to the `drop.Processor` interface would normally require updating
its mock. That file has **no** `var _ drop.Processor` assertion, already omits
`Consume`/`ConsumeAndEmit`, and has zero callers in the repo (`grep -rn
"drop/mock"` under `services/atlas-drops` returns nothing). Touching it would
raise the "why not `Consume` too" question for no benefit. If a reviewer wants
it complete, that is a separate mechanical fix.

### 5. FR-11 / FR-14 resolution is implemented as design §7.1 specifies

Zero-share suppression applies only to non-picker recipients. The picker's
`MESO_AWARDED` is emitted unconditionally whenever a `meso > 0` drop is
reserved, including at `Amount: 0`, because it is what completes the pickup.
`atlas-character` skips the whole transaction on `Amount == 0` so no phantom
"+0 mesos" line is produced.

### 6. The `RESERVED`-no-longer-credits guard is a consumer test, not a processor test

FR-15 removes a handler rather than changing behavior, so there is no behavior
to assert on `character.Processor`. The plan's guard is
`TestHandleMesoAwarded_IgnoresNonMesoAwardedEvents` in the consumer package,
which passes a `nil` `*gorm.DB` — a handler that did not short-circuit on the
type guard would panic.

---

## Dependencies and ordering

- Task 2 needs Task 1's `party.MemberModel` + `party.NewMemberBuilder()`.
- Task 3 needs Task 1 (`party.Processor`, `party/mock`) and Task 2
  (`splitMeso`, `Recipient`).
- Task 4 is independent of Tasks 1–3 at compile time — it consumes the
  `MESO_AWARDED` contract over Kafka, not over an import — but should land after
  them so the branch never has a state where meso is credited nowhere.

No `go.mod` changes anywhere. `github.com/jtumidanski/api2go`,
`github.com/google/uuid`, `atlas-constants`, `atlas-model`, and `atlas-rest` are
already direct dependencies of `atlas-drops`.

---

## Task sizing

No task is deliberately oversized. File counts: Task 1 — six new files, all
mechanical copies of a proven client shape, with one small test; Task 2 — two
new files; Task 3 — four modified files in one service; Task 4 — four modified
plus one new test file in one service. Every task stays inside one service and
one module root, so an implementer's `go build ./... && go test ./...` is a
single `cd`.

---

## Out of scope, recorded so it is not mistaken for an omission

- **`MESO_AWARDED` redelivery double-credits.** Design §8. `atlas-character`
  performs no dedupe on meso credit today either — the existing `RESERVED` path
  has the identical exposure. Making meso credit idempotent is a service-wide
  change touching `RequestChangeMeso`, sack consumption, and saga compensation.
- **Party-member location staleness.** Design §7.4.2. A member mid-map-transfer
  can be mis-classified in either direction; FR-6 bounds the damage to a partner
  occasionally missing one share. No mitigation.
- **Item and equipment party loot rules, party EXP, meso rate multipliers,
  `isSelfLootableOnly`.** PRD §2 non-goals; `drop.Model.CanBeReservedBy`
  (`drop/model.go:98`) is untouched.
- **`atlas-channel`, `atlas-parties`, `atlas-saga-orchestrator`, `atlas-rates`,
  `atlas-reactors`, `atlas-inventory`** — no code change. `atlas-channel`'s
  `MESO_CHANGED` consumer
  (`kafka/consumer/character/consumer.go:452-475`) already routes the gain
  notification by character id, so each recipient sees their own "+N mesos".
