# task-225 Evan Dragon — Whole-Branch Correctness Review

Scope: read-only cross-task review after all 17 plan tasks landed and passed
their individual task-scoped reviews. This review reads the diff
`.superpowers/sdd/plan/review-723519dc4..f71298987.diff` (588KB / 14422 lines)
in full — grepped for the changed-file list up front, then read each new/
touched source file directly from the worktree (not diff hunks) for
`libs/atlas-packet/dragon/**`, `services/atlas-dragons/**`,
`services/atlas-channel/**/dragon*`, the six seed templates' Dragon bindings,
and `docs/tasks/task-225-evan-dragon/{design,plan}.md` for cross-references.
Verification steps already run (build/vet/race/guards/matrix/bake) were not
re-run; only two targeted `grep`/`diff` spot-checks were used to confirm
factual claims (go.sum version pins, kafka contract mirror byte-for-byte,
template opcode bindings across all six templates).

---

## Critical

None found.

## Important

### 1. `MOVE_DRAGON` silently zeroes the stored `stance` on every move — a late-entering viewer's replay spawn shows the wrong pose

- `services/atlas-channel/atlas.com/channel/socket/handler/dragon_move.go:29`
  hardcodes the `stance` argument to `Move` as literal `0`:
  ```go
  _ = dragoncmd.NewProcessor(l, ctx).Move(s.Field(), s.CharacterId(), p.StartX(), p.StartY(), 0, p.RawMovement())
  ```
  `libs/atlas-packet/dragon/serverbound/move.go` confirms `serverbound.Move`
  genuinely has no stance accessor — `CVecCtrlDragon::EndUpdateActive` sends
  only the opaque `CMovePath` blob, so there is no real value to pass. That
  part is correct and matches the client evidence.
- The bug is downstream: `services/atlas-dragons/atlas.com/dragons/dragon/processor.go:148-150`
  threads that `0` straight into the registry:
  ```go
  m, err := GetRegistry().Update(p.ctx, p.t, characterId, func(cur Model) Model {
      return cur.Move(int32(startX), int32(startY), stance)   // stance == 0, always
  })
  ```
  `Model.Move` (`services/atlas-dragons/atlas.com/dragons/dragon/model.go:35-37`)
  unconditionally overwrites the stored `stance` field. So after a dragon's
  **first** move, ever, the registry's `stance` is permanently pinned to `0`
  regardless of the dragon's real facing/pose.
- Consequence: `spawnDragonForSession`
  (`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:464-478`)
  replays `m.Stance()` from the registry to any player entering the map later
  — after that first move, every late entrant sees the dragon spawn with
  stance `0` instead of its true last stance, until the dragon's next MOVED
  broadcast (which doesn't touch stance visually anyway — `DragonMoveBody`
  only relays the raw blob) corrects the client's own rendering. It never
  self-heals in the *stored* value; it degrades permanently to `0` after the
  first move and stays there for the dragon's whole lifetime.
- This is exactly a seam bug: Task 10's `Processor.Move(... stance byte ...)`
  signature and Task 12's handler wiring were each individually correct
  against their own contracts (the codec genuinely has no stance; the
  processor genuinely accepts a stance parameter), but composed, the
  parameter is dead weight that quietly corrupts persisted state. The plan
  itself (`docs/tasks/task-225-evan-dragon/plan.md:3379`) specifies passing
  literal `0` with no comment explaining why, which is a sign this wasn't a
  deliberate, reviewed tradeoff so much as an unexamined consequence of
  keeping the `Move` signature symmetric with `Create`.
- Fix options, in order of cleanliness: (a) drop `stance` from
  `Processor.Move`/`Registry.Update`'s closure entirely and leave the stored
  stance untouched by `Move` (since MOVE never legitimately changes it), or
  (b) if a real stance signal is ever decoded from the movement blob later,
  wire it through then. Given the wire genuinely carries nothing, (a) is
  correct today. This is cosmetic-only (wrong pose for one spawn frame, no
  desync, no crash) but is a real, reachable defect, not a hypothetical —
  recommend fixing before merge since the fix is a few lines.

## Minor

### 2. `atlas-dragons/go.sum` reverses the just-landed repo-wide dependency bump

Confirmed by direct read, not just citing the deferred list:
```
services/atlas-dragons/atlas.com/dragons/go.sum:195: golang.org/x/mod v0.38.0 ...
services/atlas-dragons/atlas.com/dragons/go.sum:206: golang.org/x/tools v0.47.0 ...
```
while `go.work.sum` (bumped by the merge-base commit itself,
`723519dc4 fix(deps): update module golang.org/x/mod to v0.39.0 (#1335)`)
carries `v0.39.0`/`v0.48.0`. `x/mod`/`x/tools` are indirect-only in
`atlas-dragons/go.mod` (not a direct `require`), which is why `go build`/
`docker buildx bake atlas-dragons` are unaffected — the workspace-level
`go.work.sum` supplies the checksum either way. This is a real, verified
drift, not just a deferred-list restatement: the new module's `go.sum` was
generated (via `go mod tidy`, presumably at scaffolding time within this
branch) before picking up the version the rest of the fleet had just
standardized on at the merge base. **Triage: fix before merge, trivially** —
`cd services/atlas-dragons/atlas.com/dragons && go get golang.org/x/mod@v0.39.0 golang.org/x/tools@v0.48.0 && go mod tidy`,
or let the next repo-wide bump correct it. Not a build blocker either way.

---

## Seam checks performed (composition correctness) — no additional defects found

- **CREATE broadcast (map-wide incl. owner) vs. replay-on-entry (skips self)
  compose correctly.** `handleStatusEventCreated`
  (`services/atlas-channel/atlas.com/channel/kafka/consumer/dragon/consumer.go:85-102`)
  broadcasts `ForSessionsInMap` including the owner — needed because the owner
  has not locally rendered their own dragon. `spawnDragonForSession`
  (`kafka/consumer/map/consumer.go:469`) explicitly skips
  `m.OwnerCharacterId() == s.CharacterId()`. Traced both call graphs: a
  fresh-login dragon owner gets exactly one SPAWN_DRAGON (via CREATED
  broadcast); a player entering an existing dragon's map later gets exactly
  one SPAWN_DRAGON for each pre-existing dragon (via replay, self excluded)
  plus the CREATED broadcast for their *own* dragon if they are themselves
  an Evan logging in. No duplicate, no gap, in either direction.

- **`JOB_CHANGED` DESTROY-half (atlas-dragons) / CREATE-half (atlas-channel)
  do not race into a double-create or lost destroy.** They consume the same
  `EVENT_TOPIC_CHARACTER_STATUS` topic in different consumer groups with no
  ordering guarantee between them, but the composition is race-safe because
  both halves re-derive ground truth rather than trusting event-body state:
  atlas-dragons' `handleJobChanged`
  (`services/atlas-dragons/atlas.com/dragons/kafka/consumer/character/consumer.go:64-74`)
  destroys only if the event's own `JobId` resolves to non-dragon-bearing;
  atlas-channel's `handleStatusEventJobChanged`
  (`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:511-523`)
  fires CREATE *unconditionally*, but `Processor.Create`
  (`services/atlas-dragons/.../dragon/processor.go:73-111`) re-fetches the
  character fresh via REST and re-evaluates `HasDragon` at execution time —
  it never trusts the triggering event's job id. So regardless of which
  handler runs first, Create's own gate reflects the character's *current*
  (post-change) job, making the outcome order-independent: entering the
  range creates once (emit-on-absent→present prevents a dup even under
  redelivery race, see below); leaving the range destroys once and any
  in-flight/redelivered Create is a no-op against the now-non-Evan job.
  One residual, narrow gap noted but not flagged as a defect: `Create`'s
  field argument comes from the *session's* field at JOB_CHANGED-processing
  time (`s.Field()`), so a JOB_CHANGED that lands mid-warp (extremely rare —
  job change and map change essentially never coincide) could seed a stale
  field. This is a pre-existing class of hazard for any Kafka-driven
  cross-service command with a field argument (shared with summons/pets),
  not something task-225 introduced or needs to solve alone.

- **Kafka contract mirror is byte-for-byte identical today.** Diffed
  `services/atlas-channel/atlas.com/channel/kafka/message/dragon/kafka.go`
  against `services/atlas-dragons/atlas.com/dragons/{dragon/kafka.go,
  kafka/consumer/dragon/kafka.go}` field-by-field (types + json tags): the
  `Command[E]`/`CreateCommandBody`/`DestroyCommandBody`/`MoveCommandBody` and
  `StatusEvent[E]`/`StatusEventCreatedBody`/`StatusEventMovedBody`/
  `StatusEventDestroyedBody` definitions match exactly, differing only in
  doc comments (expected — each file's comment names the mirror direction).
  `kafka_test.go` on the channel side pins this from literal JSON, giving the
  regression net the doc comment promises.

- **Idempotency under at-least-once redelivery is handled, with a
  documented, evidenced client-side backstop.** `Create`'s
  `Exists`-then-`Put` is not atomic (two racing deliveries could both read
  `existed==false` and both emit CREATED), but (a) `Put` itself is an
  unconditional upsert so the stored state converges correctly either way,
  and (b) the design's own citation of `CUser::OnDragonPacket`'s spawn arm
  (`design.md` §2.4: `if (nType==206) { ...release existing...; OnCreated(...); }`)
  is a real decompiled fact, not a hand-wave — the client releases any
  existing dragon object before creating a new one, so a duplicate
  SPAWN_DRAGON self-heals visually. This mirrors how the rest of the
  Kafka-driven entity registries in this codebase (summons, pets) accept the
  same non-atomic-upsert tradeoff; not a new risk introduced here.

- **Field-index (`fieldIdx`) contract is respected by all callers.** Grepped
  every call site of `Registry.Update` — the only caller is `Processor.Move`,
  which changes only x/y/stance, never `Field()`, so the documented
  "Update never migrates the field index" contract is not violated anywhere
  in this branch. `Put` correctly detects and migrates a field change
  (`registry.go:128-138`); `Remove` and job-change/map-change/channel-change
  handlers all route field changes through Destroy+Create (a Remove+Put
  pair), never through Update.

- **`int32` coordinates are consistent end-to-end with zero narrowing.**
  Traced the full pipeline: `dragon.Model` (int32) →
  `storedDragon`/Redis JSON (int32) → `StatusEventCreatedBody` (int32, both
  mirrored copies) → `libs/atlas-packet/dragon/clientbound/spawn.go`
  (`WriteInt32`/`ReadInt32`, matching the confirmed 4-byte `Decode4 x, Decode4 y`
  in `CDragon::OnCreated`) → both REST models (atlas-dragons' `RestModel` and
  atlas-channel's mirror `RestModel`, both `int32` with matching json tags).
  The only place a 16-bit type appears is `character.Model.X()/Y()` (int16,
  correctly — that's the *character's* position, which really is 2-byte on
  the wire) and `serverbound.Move.StartX()/StartY()` (int16, correctly —
  lifted from the `CMovePath` blob's first 4 bytes, which really are 2-byte
  fields). `Create` casts `int32(c.X())`/`int32(c.Y())` at the one legitimate
  widening point. No narrowing conversion exists anywhere in the pipeline.

- **`jobId` version gate is correctly localized to the one codec that needs
  it.** `spawnHasJobId` in `libs/atlas-packet/dragon/clientbound/spawn.go:74-77`
  is the only version-conditional in the whole feature. `StatusEventCreatedBody.JobId`,
  the channel-side `Model.JobId()`, and `writer.DragonSpawnBody`'s `jobId`
  parameter are all unconditionally present in the internal
  model/event/REST/writer-argument pipeline — jobId is simply omitted from
  the *wire encode* on gated versions, never assumed present downstream of
  that boundary. Nothing else in the pipeline reads or requires `jobId` to
  have been transmitted.

- **Template routing matches Go registration exactly, across all six
  templates.** Parsed `handlers`/`writers` in
  `template_gms_{83,84,87,92,95}_1.json` and `template_jms_185_1.json`
  directly (not by grep on the diff): each binds handler name
  `DragonMoveHandle` and writer names `DragonSpawn`/`DragonMove`/
  `DragonRemove`, exactly matching the Go constants
  `serverbound.DragonMoveHandle`, `clientbound.DragonSpawnWriter`,
  `clientbound.DragonMoveWriter`, `clientbound.DragonRemoveWriter`, and
  `main.go`'s registration of the same four strings
  (`services/atlas-channel/atlas.com/channel/main.go:661-663,869`). No
  numeric opcode leaked into Go; opcodes differ per version as expected
  (e.g. v83 `0xB5`/`0xB6`/`0xB7` vs v95 `0xCE`/`0xCF`/`0xD0`/`0xD6`), names do
  not.

---

## Triage of the 11 deferred Minor findings

1. **`Destroy`/`Move`'s first `Get` swallows its error** (`processor.go:121-124`,
   `:144-147`) — **genuinely fine to ship.** Confirmed by reading both call
   sites: the error is deliberately mapped to "no dragon, no-op" per the
   FR-1.6 idempotency requirement, and a real Redis outage surfacing as "no
   dragon" is a false-negative, not data corruption — the next successful
   call re-converges. Would be nicer with a distinguishing log line, but not
   a correctness bug.

2. **`fieldIdx.Remove` errors silently discarded in `Put`/`Remove`** — **fine
   to ship; verified against precedent.** Grepped `atlas-summons/summon/registry.go:266`
   and confirmed the identical `_ = r.fieldIdx.Remove(...)` pattern already
   exists there. This is established codebase convention, not a new
   deviation.

3. **Tenant scoping hand-formats the suffix instead of
   `TenantRegistry`/`TenantKeyedSet`** — **fine to ship.** Read
   `libs/atlas-redis/tenant_registry.go`: it does not offer a field-index
   companion type matching `KeyedSet`'s shape, and — per the registry.go
   comment — the hand-formatted-suffix approach mirrors `storedSummon` in
   atlas-summons, which predates `TenantRegistry`. Functionally correct (no
   cross-tenant leakage: every key is prefixed with `t.Id().String()`
   consistently in `storeSuffix`/`fieldSuffix`); a real but pre-existing
   inconsistency in the codebase, not one task-225 introduced.

4. **`handleGetDragonByCharacterId` maps ANY error to 404, no log** — **ship,
   matches the documented tradeoff.** The doc comment directly above it
   explains the design intent: 404-for-no-dragon is the overwhelming-majority
   case (every non-Evan character), and logging it would be one error line
   per non-Evan lookup. The cost is that a genuine Redis outage is
   indistinguishable from "no dragon" at this one endpoint — acceptable
   given `Processor.GetByCharacterId`'s error-swallowing is already accepted
   in finding 1 above; this is the same tradeoff one layer up, not a new one.

5. **Channel-side move handler discards the processor's error with `_ =`,
   no log** — **ship.** `dragon_move.go:29`. Matches the pattern used by
   sibling move handlers elsewhere in the codebase for fire-and-forget
   Kafka-command production; a failed Move here just means one movement
   frame doesn't propagate, self-correcting on the dragon's next move. Not
   worth blocking on.

6. **`InitConsumers` discards the inner `rf(...)` return in both
   atlas-dragons consumer files** — **ship.** `rf` here is
   `consumer.GetManager().RegisterHandler`-shaped registration; the pattern
   of ignoring its return in `InitConsumers` (as opposed to `InitHandlers`,
   which correctly checks handler-registration errors) is standard across
   the codebase's consumer bootstrap code, not specific to this branch.

7. **No direct test coverage for `services/atlas-channel/.../dragon/`** —
   **should note as a real gap, but not merge-blocking.** This package (the
   channel-side `Processor`/`Model`/producer/requests) has no `_test.go`
   file; its correctness is exercised only indirectly through the
   `kafka/consumer/dragon` and `kafka/consumer/map` tests and the codec
   tests. Given the package is thin (mostly wiring — Kafka provider calls
   and a REST drain), the risk is low, but it is the one honestly-uncovered
   seam in the branch. Recommend a follow-up, not a blocker.

8. **`kafka/consumer/dragon/consumer_test.go` tests `excludesOwner`/`handles`,
   which the broadcast handlers never call** — **confirmed exactly as
   described; ship, but the gap is real and worth closing cheaply.** Read
   `consumer.go:85-143`: `handleStatusEventCreated`/`handleStatusEventMoved`/
   `handleStatusEventDestroyed` each hardcode their own
   `ForSessionsInMap`/`ForOtherSessionsInMap` choice directly; `excludesOwner`
   and `handles` (defined at the top of the same file) are dead from the
   handlers' perspective — `consumer_test.go` verifies the pure functions,
   not the wiring. A future edit that swapped `ForSessionsInMap` for
   `ForOtherSessionsInMap` in one handler would compile and pass CI with zero
   red tests. Manual review today confirms the three handlers are correct
   (cross-checked against FR-3.1/FR-3.3/FR-4.3 in this review's "Seam checks"
   section above), so there is no live bug — but there is genuinely no
   regression net. Cheap fix for a fast-follow: have each handler call
   `handles`/`excludesOwner` internally rather than re-encoding the policy
   ad hoc, which would make the existing unit tests actually cover the
   dispatch logic they're named for.

9. **`ProcessorImpl.t` (tenant) stored but never read in atlas-dragons'
   character processor** — **ship.** Confirmed: `character/processor.go`
   never reads `p.t` after construction. `tenant.MustFromContext(ctx)` at
   construction time is doing useful work regardless (it panics fast on a
   missing tenant rather than failing deep inside a request), so the call is
   worth keeping even though storing its result is dead. Cosmetic only.

10. **Doc comment says the envelope "mirrors atlas-character," but
    MAP_CHANGED/CHANNEL_CHANGED are actually produced by atlas-channel** —
    **confirmed factually wrong; doc-only, not merge-blocking.** Grepped for
    `StatusEventTypeMapChanged =` across both services: only
    `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go`
    defines/produces it; atlas-character does not. The
    `kafka/consumer/character/kafka.go:21-23` comment in atlas-dragons is
    therefore inaccurate for two of its five event types (LOGIN/LOGOUT/
    JOB_CHANGED do originate in atlas-character; MAP_CHANGED/CHANNEL_CHANGED
    do not). Worth a one-line comment fix so a future maintainer chasing a
    field down doesn't start at the wrong service, but has zero runtime
    effect.

11. **`go.sum` pins `x/mod`/`x/tools` behind the just-landed repo-wide bump**
    — see Minor finding 2 above. Upgraded from "deferred" to a concrete,
    independently-verified finding with an exact fix command; recommend
    actually applying it before merge since it's a two-command fix and
    leaving it means the very next repo-wide dependency PR will show an
    unexplained "already bumped, why is atlas-dragons behind" diff.

---

## Verdict

The branch is **merge-ready modulo one small fix**: resolve finding 1 (the
`stance` zeroing on move) before merge — it's a genuine, reachable defect
with a small fix. Finding 2 / deferred-11 (the go.sum pin) is worth folding
into the same PR since it's a two-command fix, but is not itself blocking.
Nothing else found in this whole-branch pass rises above Minor, and every
deferred Minor from the execution ledger has been independently re-verified
against source (not merely restated) and triaged above — none of them
warrant blocking. The lifecycle composition (CREATE/replay/JOB_CHANGED
split/idempotency/field-index contract/coordinate width/version
gate/template↔Go name binding) all check out under direct tracing through
the actual code, not just the doc comments describing it.

Task 17's live steps (tenant socket-config reconcile, live behavioural
verification against a running client) remain a genuine post-deploy
checklist item, not something this static review can discharge — noted as
already known/accepted per the task brief.
