# Portal ENTER Double-Execute — Design

Version: v1
Status: Approved for planning
Created: 2026-08-07
PRD: [prd.md](prd.md)
Issue: [#1193](https://github.com/Chronicle20/atlas/issues/1193)

---

## 1. Scope and shape of the change

Three independent mechanisms, deployable in order, each of which stands on its
own:

| Layer | Service | What it does | Failure mode if it alone is deployed |
|---|---|---|---|
| A — warp acknowledgement | `atlas-saga-orchestrator` | `WarpToPortal` / `WarpToSavedLocation` complete on `MAP_CHANGED` instead of hanging to a timeout | None. Strictly removes spurious `SAGA_TIMEOUT`. |
| B — conditional unlock | `atlas-portal-actions` | Outcomes that move the character do not send `EnableActions`; the resulting `SET_FIELD` unlocks the client | Without A, a suppressed unlock leans on a `FAILED` that still fires spuriously at 30 s |
| C — dedupe gate | `atlas-portal-actions` | Redis `SET NX` + 2 s TTL in front of rule evaluation | None. Pure defence in depth. |

**A is the prerequisite for B**, exactly as the PRD argues. Deploy
`atlas-saga-orchestrator` first, then `atlas-portal-actions`. Both directions are
wire-compatible (§8), so the ordering is a quality-of-rollout concern, not a
correctness gate.

B is the actual fix. C exists because the client is *designed* to re-fire while
the player stands in a portal rect, so any future unlock-shaped regression
anywhere in the outcome path re-opens the same hole.

---

## 2. Verification of the PRD's premises

Every load-bearing claim in the PRD was re-read against the worktree at
`a2d00142e`. All hold.

| Claim | Verified at |
|---|---|
| Every portal-enter outcome calls `EnableActions` | `services/atlas-portal-actions/atlas.com/portal/script/consumer.go:88,98,104` — error branch, `!Allow` branch, and unconditional fallthrough |
| `ProcessResult` already carries the matched operations | `portal/script/model.go:75-81` (`Operations []operation.Model`), populated at `portal/script/processor.go:167,176` |
| `handleWarpToPortal` returns without `StepCompleted` | `saga/handler.go:992-1017` — emits `WarpToPortalAndEmit(s.TransactionId(), …)` at :1009 and returns `nil` |
| Both warp actions are `{}` in the acceptance table | `saga/event_acceptance.go:219-220` |
| The acceptance-table comment's same-map claim is false | `atlas-maps/.../character/warp/processor.go:74-96` — `ChangeMap` emits `MapChangedStatusProvider` unconditionally; no `oldField == dest` guard |
| `MAP_CHANGED` carries both correlation fields | `atlas-maps/.../kafka/producer/character.go:40-47` sets `TransactionId` **and** `CharacterId` on the envelope (`saga-orchestrator/.../kafka/message/character/kafka.go:244-250`) |
| `handleWarpPartyQuestMembers` fans out under one tx id then self-completes | `saga/handler.go:2894-2906` — `WarpToPortalAndEmit(s.TransactionId(), member.Id, …)` in a loop, then `StepCompleted` |
| `ExtractCharacterId` covers `WarpToPortalPayload` but not `WarpToSavedLocationPayload` | `saga/character_extractor.go:43-44`, no `WarpToSavedLocationPayload` case |
| `Builder.SetTimeout` exists and 0 means "orchestrator default" | `libs/atlas-saga/builder.go:44-48,66-70`; consumed at `saga-orchestrator/saga/model.go:316-341` (`DefaultSagaTimeout = 30s`), scheduled at `saga/processor.go:302` |
| `Lock.AcquireWithTTL` is `SET NX` + TTL | `libs/atlas-redis/lock.go:60-67` |
| `atlas-portal-actions` already has a Redis client at startup | `portal/main.go:16,51` — `atlas` import and `action.InitRegistry(rc)` |
| `TenantRegistry` supports a TTL'd put | `libs/atlas-redis/tenant_registry.go:117` — `PutWithTTL` |

One PRD detail is **corrected** by this design: `Lock` is *not* tenant-scoped.
`lockKey` is `namespacedKey(namespace, "_lock", key)` (`lock.go:45`) with no
tenant segment. FR-3.3 is satisfied by composing the caller-supplied `key` from
the library's own `redis.TenantKey(t)` and `redis.CompositeKey(...)` helpers
(`libs/atlas-redis/keys.go:33-57`) — still "standard namespacing", not hand-rolled
string concatenation, and no shared-lib change. See §6.2 for the rejected
alternative.

---

## 3. Layer A — warp acknowledgement (`atlas-saga-orchestrator`)

### 3.1 Acceptance table (FR-1.1, FR-1.2)

`WarpToPortal` and `WarpToSavedLocation` move out of the fire-and-forget block
and join `WarpToRandomPortal`:

```go
sharedsaga.WarpToRandomPortal:  {EventKindCharacterMapChanged},
sharedsaga.WarpToPortal:        {EventKindCharacterMapChanged},
sharedsaga.WarpToSavedLocation: {EventKindCharacterMapChanged},
```

The comment block at `event_acceptance.go:202-215` is replaced. The new text must
say, in substance:

> All three warp actions advance on `character.map_changed`, tagged with the saga
> `transactionId`. `atlas-maps` `warp.ProcessorImpl.ChangeMap`
> (`services/atlas-maps/atlas.com/maps/character/warp/processor.go`) emits
> `MAP_CHANGED` **unconditionally** — there is no same-map short-circuit, so a
> portal-to-portal warp within one map acknowledges exactly like a cross-map one.
> An earlier revision of this comment asserted the opposite and left these two
> actions self-completing; that claim was false and every portal warp saga
> consequently ran to `SAGA_TIMEOUT`. If a same-map short-circuit is ever added
> to `ChangeMap`, these entries break silently — the step would hang to timeout
> again.

That last sentence is the whole point of FR-1.2: it names the exact change in
another service that would silently regress this one. The `atlas-maps` side gets
no code change (per PRD §7), so the comment is the only coupling record.

### 3.2 Character-id guard (FR-1.3, FR-1.4, FR-1.5)

`AcceptEvent` currently takes `(transactionId, kind)` and has 63 call sites in the
service. Rather than change every one, extend it with a variadic option:

```go
type acceptOptions struct {
    characterId    uint32
    hasCharacterId bool
}

type AcceptOption func(*acceptOptions)

// ForCharacter constrains acceptance to a step whose payload names this
// character. A step whose payload carries no character id is unconstrained.
func ForCharacter(id uint32) AcceptOption { … }

AcceptEvent(transactionId uuid.UUID, kind EventKind, opts ...AcceptOption) (AcceptDecision, bool)
```

The guard runs as the **last** check in `AcceptEvent`, after the existing
`StepAcceptsEvent` gate (`saga/processor.go:430-439`) and before returning the
decision:

```go
if o.hasCharacterId {
    if want := ExtractCharacterId(step); want != 0 && want != o.characterId {
        LogSkip(p.l, logrus.Fields{
            "transaction_id":     transactionId.String(),
            "step_id":            step.StepId(),
            "step_action":        step.Action(),
            "event_kind":         kind,
            "event_character_id": o.characterId,
            "step_character_id":  want,
        }, SkipReasonCharacterIdMismatch)
        return AcceptDecision{}, false
    }
}
```

- `SkipReasonCharacterIdMismatch = "character_id_mismatch"` joins the constants at
  `event_acceptance.go:385-393` (FR-1.5).
- `want == 0` means "the payload has no character id" — unconstrained, accept.
  This preserves every existing action's behaviour, since `ExtractCharacterId`
  returns 0 for unknown payloads (`character_extractor.go:79-81`).
- `maybeWarnUnmatchedEvent` is **not** called on this path. A cross-character
  event is expected traffic under the party-quest fan-out, not an anomaly; warning
  once per `(tx, kind)` would be noise.
- FR-1.4: `WarpToSavedLocationPayload` is added to `ExtractCharacterId` next to the
  `WarpToPortalPayload` case.

Only one caller passes the option, keeping blast radius minimal:

```go
// kafka/consumer/character/consumer.go, handleCharacterMapChangedEvent
if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterMapChanged,
    saga.ForCharacter(e.CharacterId)); !ok {
    return
}
```

The mechanism is generic; scoping the *use* to `map_changed` means no other event
family's behaviour can shift under this task.

### 3.3 Why the guard is required, not optional

`handleWarpPartyQuestMembers` (`saga/handler.go:2894-2906`) issues N
`WarpToPortalAndEmit` calls stamped with the *saga's* transaction id, then
immediately self-completes its own step. Those N `MAP_CHANGED` events land some
hundreds of milliseconds later, by which time the saga's current step is whatever
came next. Before this task, `map_changed` matched nothing (both warp actions were
`{}`), so the events fell out at `SkipReasonActionMismatch`. After §3.1 they
match — so without §3.2, a party member's arrival would complete a following
`WarpToPortal` step belonging to a different character. The guard is a direct
consequence of §3.1, not a speculative hardening.

### 3.4 Resolution of PRD open question 1 — newly-reachable steps

Enumerated every in-repo construction of a saga step with these two actions:

| Producer | Position | Effect of layer A |
|---|---|---|
| `atlas-channel/.../respawn/processor.go:255-269` | `warp_to_spawn` is appended **last**; the saga is built immediately after (`:271-278`) | Terminal. No newly-reachable step. Saga now completes instead of timing out. |
| `atlas-portal-actions/.../script/executor.go:147-164` (`executeWarp`) | Single-step saga | Terminal. |
| `atlas-portal-actions/.../script/executor.go:573-587` (`executeWarpToSavedLocation`) | Single-step saga | Terminal. |
| `atlas-npc-conversations/.../conversation/operation_executor.go:1362-1371`, `:2481-2488` | Emitted by `createStepForOperation`, consumed by either `createSagaForOperation` (single step, terminal) or `createSagaForOperations` (multi-step batch, `:1013`) | **Only non-terminal case.** |

The multi-step batch path is driven by conversation scripts that live in the
database, not in the repo — a static in-repo enumeration cannot close this. Grep
of all committed JSON found no `warp_to_portal` / `warp_to_saved_location` outside
`atlas-portal-actions/docs/portal_script_schema.json`; the only occurrences in
`atlas-npc-conversations` are in `.bruno` request fixtures.

**Resolution.** Where such a script exists today, the operations after the warp
step are *already dead* — the batch stalls at the warp step and everything after
it never dispatches, until the 30 s timeout fails the saga. Layer A makes them
run, which is what the script author wrote them to do. That is a bug fix, not a
regression, and it is the same fix task-124 applied to `WarpToRandomPortal` for
the teleport-rock `consume_rock` chain.

The residual risk is a script that was authored around the broken behaviour. This
is assessed as negligible: the broken behaviour also produced a `SAGA_TIMEOUT` and
a `FAILED` event on every invocation, which is not a stable base to build on.
Recorded here rather than gated on, and called out in the PR body so it is
reviewable against live tenant data.

### 3.5 Resolution of PRD open question 2 — residual party-quest overlap

**Accepted as a documented residual.** The uncovered case is: one saga in which
`WarpPartyQuestMembersToMap` warps character X, and a *later* step in the same
saga is a `WarpToPortal` also naming X. The character-id guard cannot separate
those two events — they agree on both correlation keys.

Closing it would require qualifying the correlation by step, i.e. adding a
`stepId` to the `CHANGE_MAP` command and echoing it on `MAP_CHANGED`. PRD §5
declares no Kafka field is added or removed, and §2 lists reworking the
correlation model as a non-goal. No such saga exists: `WarpPartyQuestMembersToMap`
is produced only by party-quest handlers, and no in-repo saga pairs it with a
subsequent `WarpToPortal`.

The mitigation, should it ever occur, is already in place and cheap: give the
follow-on warp its own saga. This is noted in the rewritten comment block from
§3.1.

---

## 4. Layer B — conditional unlock (`atlas-portal-actions`)

### 4.1 Operation classification (FR-2.1, FR-2.2)

`ExecuteOperation` is today a 45-line `switch` on `op.Type()`
(`script/executor.go:39-86`) with a `default` that warns and returns `nil`. That
switch is the de-facto registry of known operations, and Go gives no exhaustiveness
check over it — so a literal `movingOps = []string{...}` beside it would drift
silently, which is exactly what FR-2.2 forbids.

Replace the switch with a table that makes classification a required field:

```go
// opClass classifies an operation by whether it moves the character.
// The zero value is deliberately invalid so a new table entry that omits
// the class cannot default to "does not move".
type opClass int

const (
    opClassUnset opClass = iota // invalid — see init()
    opClassStatic               // leaves the character where they are
    opClassMoving               // dispatches a warp / field change
)

type opDef struct {
    class opClass
    run   func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error
}

var opTable = map[string]opDef{
    "play_portal_sound":        {class: opClassStatic, run: …},
    "warp":                     {class: opClassMoving, run: …},
    "drop_message":             {class: opClassStatic, run: …},
    "show_hint":                {class: opClassStatic, run: …},
    "block_portal":             {class: opClassStatic, run: …},
    "create_skill":             {class: opClassStatic, run: …},
    "update_skill":             {class: opClassStatic, run: …},
    "start_instance_transport": {class: opClassMoving, run: …},
    "apply_consumable_effect":  {class: opClassStatic, run: …},
    "cancel_consumable_effect": {class: opClassStatic, run: …},
    "save_location":            {class: opClassStatic, run: …},
    "warp_to_saved_location":   {class: opClassMoving, run: …},
    "start_quest":              {class: opClassStatic, run: …},
}

func init() {
    for name, def := range opTable {
        if def.class == opClassUnset {
            panic("portal operation [" + name + "] has no opClass; classify it as opClassStatic or opClassMoving")
        }
        if def.run == nil {
            panic("portal operation [" + name + "] has no run function")
        }
    }
}
```

The thirteen `execute*` methods keep their current bodies and signatures; only the
dispatch changes. `ExecuteOperation` becomes a map lookup, with the unknown-type
`Warn`-and-`return nil` behaviour preserved for a miss.

**Why `init()` panic and not just a test.** FR-2.2 permits either. A panic in
`init()` is strictly stronger: it fires in *every* test binary in the package, in
`go vet`-adjacent tooling that loads the package, and at service startup. The
condition is statically determined by a composite literal, so it cannot be reached
by any runtime input — it is a compile-and-run-once developer assertion, not a
production failure mode. A dedicated unit test (FR-4.1's sibling) additionally
asserts the table's classification for each of the three moving members, so the
intent is visible in test output and not only in a panic string.

The predicate the consumer needs is exported from the same table — one source of
truth, no second list:

```go
func IsMovingOperation(opType string) bool  // opTable[opType].class == opClassMoving
```

### 4.2 Threading the decision to the consumer (FR-2.3, FR-2.4)

`ProcessResult` already carries `Operations`, so the consumer *could* classify
directly. It should not, for one reason: on the error path
(`processor.go:162-170`) the operations list is present but only a *prefix* of it
actually ran. If a `warp` dispatched successfully and a following `start_quest`
then failed, classifying by the declared list and classifying by what was
dispatched give the same answer — but if the `warp` itself failed before creating
its saga, the list says "moving" while nothing is in flight, and suppressing the
unlock would freeze the player with no saga to fail and release them.

So the decision is reported by the executor as a value, not derived from the
declaration:

```go
// ExecuteOperations reports whether at least one moving operation was
// successfully dispatched, so the caller can decide whether the client is
// already being unlocked by the resulting SET_FIELD.
func (e *OperationExecutor) ExecuteOperations(
    f field.Model, characterId, portalId uint32, ops []operation.Model,
) (movedCharacter bool, err error)
```

`movedCharacter` is set only after the moving operation's `sagaP.Create` returns
`nil`. It is a plain return value — no mutable executor state, so nothing to race
even though `OperationExecutor` is constructed per-request
(`processor.go:50-63`).

`ProcessResult` gains `CharacterMoved bool`, populated on both the success and
the error return in `Process` (`processor.go:162-178`).

`handleEnterCommand` collapses its three unlock sites into one guarded call:

```go
if result.Error != nil {
    l.WithError(result.Error).Errorf(…)
} else {
    l.Debugf(…)
}

// A moving outcome is unlocked by the SET_FIELD its warp produces
// (CWvsContext::OnGameStageChanged clears m_bExclRequestSent). Unlocking here
// while the warp is in flight is what makes the client legitimately re-fire —
// see docs/tasks/task-184-portal-enter-double-execute/prd.md §1.1.
if result.CharacterMoved {
    return
}
character.EnableActions(l)(ctx)(ch, c.Body.CharacterId)
```

This is a **deliberate strengthening of FR-2.3/FR-2.4**: the PRD phrases the
condition as "the matched outcome's operations include at least one moving
operation" and preserves the error branch unchanged. Keying on
*successfully dispatched* instead of *declared*, and applying it to the error
branch too, is strictly safer in both directions — a failed-to-dispatch warp still
unlocks (FR-2.4's intent), and a dispatched warp followed by an unrelated
operation error does not double-unlock (FR-2.3's intent). Flagged explicitly so
the plan-adherence review does not read it as a deviation.

`no_script`, `no_match`, and `!Allow` outcomes all carry `CharacterMoved == false`
and unlock exactly as today.

### 4.3 Pending-action registration and the failure path (FR-2.5, FR-2.6, FR-2.7)

**Where registration happens.** FR-2.5 says `handleEnterCommand` registers the
`PendingAction`. It cannot: the saga transaction id is minted inside
`executeWarp` / `executeWarpToSavedLocation`, and the consumer has no handle on
it. Registration therefore happens in those two executor methods, mirroring
`executeStartInstanceTransport` (`executor.go:447-456`), which already does
exactly this. The FR's intent — "a suppressed unlock is backed by a registered
pending action" — is met; the location differs. Flagged as a deviation.

Both methods take the shape transport already uses:

```go
sagaId := uuid.New()
action.GetRegistry().AddWithTTL(e.ctx, sagaId, action.PendingAction{
    CharacterId: characterId,
    WorldId:     f.WorldId(),
    ChannelId:   f.ChannelId(),
    Kind:        action.KindWarp,
}, pendingActionTTL)

s := saga.NewBuilder().
    SetTransactionId(sagaId).
    SetTimeout(warpSagaTimeout).   // 5s — FR-2.6
    …
```

**`AddWithTTL` (new).** `Registry.Add` uses `TenantRegistry.Put` with no
expiry (`portal/action/registry.go:43-46`), so a dropped `COMPLETED` event leaks
the key forever. Warp registrations use a new `AddWithTTL` over the existing
`TenantRegistry.PutWithTTL` (`libs/atlas-redis/tenant_registry.go:117`), with
`pendingActionTTL = 60 * time.Second`. The TTL must exceed the saga timeout by a
wide margin so `handleStatusEventFailed` can still find the entry; 60 s against a
5 s timeout is ample. `executeStartInstanceTransport` keeps `Add` — changing it is
out of scope, and it is noted as a follow-up candidate rather than silently
altered.

**Resolution of PRD open question 3 — double registration.** There is none.
Each `execute*` method mints its own `uuid.New()` and registers under that id, so
a rule containing both `warp` and `start_instance_transport` produces two sagas
with two distinct transaction ids and two independent registry entries. If both
fail, `EnableActions` is sent twice — harmless (it is an idempotent unlock; the
second arrives with the flag already clear). No conflict, no change needed.

**FR-2.6 — 5 s timeout, and PRD open question 4.** `warpSagaTimeout = 5 *
time.Second` on the two warp sagas only. `start_instance_transport` keeps the 30 s
default: it does strictly more work than a warp (route lookup, capacity check,
instance provisioning) and has its own failure path and messages; shortening it is
a behaviour change to transports that this task has no evidence to justify. 5 s
against the ~300 ms end-to-end warp observed in the issue report is ~16× headroom,
which absorbs a Kafka rebalance or a cold consumer without tripping. Left as a
named constant so it is one edit if production says otherwise.

**FR-2.7 — failure message.** `PendingAction` gains a `Kind` field
(`json:"kind"`, values `"warp"` / `"transport"`). `resolveFailureMessage`
(`kafka/consumer/saga/consumer.go:118-137`) keeps its explicit-message and
error-code branches unchanged and switches only its **default** on `Kind`:

- `KindWarp` → `"You cannot move there right now."`
- `KindTransport` and the empty/unknown value → `"Unable to board transport at
  this time."` (today's text)

Empty defaults to transport so entries written by a pre-deploy replica still
resolve to their current message. The `TRANSPORT_*` error codes are unreachable
for a warp saga, so no reordering is needed. The log line and comments at
`consumer.go:51-52,69,97` that say "transport saga" are corrected to "portal
action saga".

### 4.4 What the client does after the change

For a moving outcome, nothing in `atlas-portal-actions` clears
`m_bExclRequestSent`. `CWvsContext::OnGameStageChanged` @`0xa0400e` clears it and
resets `m_tLastExclRequest` when `set_stage` runs on the `SET_FIELD` produced by
the warp. If the warp never lands, the 5 s saga timeout produces `FAILED`,
`handleStatusEventFailed` finds the registered `PendingAction`, sends the
warp-appropriate message, and calls `EnableActions`. Either way the player regains
control; the flag is never cleared while a warp is in flight.

---

## 5. Layer C — duplicate-command gate (`atlas-portal-actions`)

### 5.1 Placement and shape

A new `portal/dedupe` package, initialised in `main.go` beside
`action.InitRegistry(rc)`:

```go
package dedupe

type Gate interface {
    // Allow reports whether this ENTER should be processed. A duplicate
    // inside the TTL window returns false. Redis errors return true.
    Allow(ctx context.Context, k Key) bool
}

type Key struct {
    CharacterId uint32
    MapId       _map.Id
    Instance    uuid.UUID
    PortalId    uint32
}
```

The implementation wraps `atlas.NewLock(client, "portal-enter")` and calls
`AcquireWithTTL(ctx, redisKey, enterGateTTL)` with

```go
redisKey := atlas.CompositeKey(
    atlas.TenantKey(tenant.MustFromContext(ctx)),
    strconv.FormatUint(uint64(k.CharacterId), 10),
    strconv.FormatUint(uint64(k.MapId), 10),
    k.Instance.String(),
    strconv.FormatUint(uint64(k.PortalId), 10),
)
```

giving a final Redis key of
`<env:>atlas:portal-enter:_lock:<tenantId>:<region>:<maj>.<min>:<char>:<map>:<instance>:<portal>`.
`TenantKey` and `CompositeKey` are the library's own helpers
(`libs/atlas-redis/keys.go:33,55`), satisfying FR-3.3 and FR-8's two-tenant
isolation requirement by construction.

`enterGateTTL = 2 * time.Second` (FR-3.4). The lock is **never released** — TTL
expiry is the release. There is no `Release` call anywhere on this path; a
successful portal entry should keep the gate closed for the full window.

### 5.2 Fail-open (FR-3.6)

`Allow` returns `true` on any Redis error, and `GetGate()` returning `nil` (not
initialised — the case in unit tests) is treated as "allow". A logged `Warn` on
the error path, rate-limited by the fact that portal entry is human-frequency.
FR-3.5's dropped-duplicate log is `Debug` with all five key components plus the
tenant id.

### 5.3 Ordering inside `handleEnterCommand`

The gate is the first statement after the entry `Debugf` and the `field`/`channel`
construction, strictly before `NewProcessor(...).Process(...)` — so a dropped
duplicate performs no script load, no condition evaluation, and no operation
dispatch (FR-3.1, FR-4.2).

### 5.4 Interaction with layer B

They are independent and neither masks the other. B removes the *cause* — with it
in place the client has no reason to re-fire, so C's counter should stay at
approximately zero in a healthy system. A non-zero `Debug` rate on the C log is
therefore itself a signal that some outcome path is still unlocking a moving
character.

---

## 6. Alternatives considered

### 6.1 Instead of layer A: complete the warp step in the handler

`handleWarpToPortal` could call `StepCompleted` itself, like
`handleWarpPartyQuestMembers` does. Rejected: that is precisely today's
fire-and-forget semantics dressed up — `FAILED` would still never mean "the warp
did not land", so layer B's safety net would still be worthless. The PRD's §1.2
argument stands. It would also diverge from `WarpToRandomPortal`, which task-124
already moved to `MAP_CHANGED` acknowledgement.

### 6.2 Instead of composing the tenant into the lock key: a `TenantLock` type

Adding `TenantLock` to `libs/atlas-redis` (mirroring `TenantRegistry`) is the
tidier long-term shape. Rejected for this task: it expands the blast radius from
two services to a shared library consumed by fourteen, for zero behavioural
difference. `TenantKey` + `CompositeKey` are exported for exactly this purpose.
Recorded as a follow-up candidate if a second tenant-scoped lock site appears.

### 6.3 Instead of `opTable`: keep the switch, add a parallel `movingOps` set + a test

Rejected. FR-2.2 requires that omitting a classification fail, and a test that
enumerates the switch's cases cannot exist — Go offers no reflection over switch
arms. Any such test would have to hardcode a third list, giving three things to
drift instead of two. The table makes the classification structurally
non-optional.

### 6.4 Instead of a variadic option on `AcceptEvent`: guard in the consumer handler

Putting the character-id comparison in `handleCharacterMapChangedEvent` after
`AcceptEvent` returns is a two-line change with no signature churn. Rejected: the
doc comment at `saga/processor.go:395-396` declares `AcceptEvent` "the single gate
at which a saga-tagged Kafka event is matched against the saga's pending step",
and the `LogSkip`/`SkipReason*` vocabulary lives with it. Splitting the gate would
leave the next reader with two places to check. The variadic option preserves the
invariant at zero cost to the other 62 call sites.

### 6.5 Instead of layer C: rely on layer B alone

Rejected on the PRD's reasoning, which the IDA evidence supports directly:
`CUserLocal::CheckPortal_Collision` @`0x94dac6` runs the check *every frame* the
player overlaps the rect, and the scripted path — unlike
`CField::SendTransferFieldRequest` @`0x53035d` — has no 500 ms floor. The client
will re-fire the instant anything clears the flag. C is cheap, fails open, and is
the only layer that survives a future regression in an outcome path nobody is
looking at.

---

## 7. Testing

Package-level tests, no new integration harness.

| Req | Test | Package |
|---|---|---|
| FR-4.1 | Outcome with a dispatched `warp` emits no `EnableActions`; outcome with only `play_portal_sound` does. Driven through `handleEnterCommand` with a fake unlocker and a fake saga processor. | `portal/script` |
| FR-2.2 | `opTable` classification asserted for all three moving members and a spot-check of static ones; a table entry with `opClassUnset` panics (`assert.Panics` over the validation func, extracted from `init()` so it is callable). | `portal/script` |
| FR-2.3 (strengthened) | A `warp` whose `sagaP.Create` returns an error yields `CharacterMoved == false` and **does** unlock. | `portal/script` |
| FR-4.2 | Second identical `ENTER` inside the window: fake `Gate` returns `false`; assert the script processor was never invoked and no operation ran. | `portal/script` |
| FR-3.6 | Gate whose Redis call errors → `Allow` returns `true`; nil gate → allow. | `portal/dedupe` |
| FR-2.7 | `resolveFailureMessage` default for `KindWarp` is the warp text; for `KindTransport` and empty it is the transport text; explicit `FailureMessage` still wins. | `portal/kafka/consumer/saga` |
| FR-4.3 | `WarpToPortal` and `WarpToSavedLocation` steps complete on `MAP_CHANGED` carrying the tx id. Extends the existing `accept_event_test.go` fixtures. | `saga` |
| FR-4.4 | `MAP_CHANGED` for character A does not complete a step whose payload names B; assert the `LogSkip` entry's `reason` is `character_id_mismatch` via the existing `logtest.Hook` pattern (`late_event_integration_test.go:98`). | `saga` |
| FR-4.5 | Same-map completion: step payload `MapId == oldMapId`; the acceptance path is map-agnostic, so this asserts no same-map special case was introduced. | `saga` |
| FR-1.4 | `ExtractCharacterId(WarpToSavedLocationPayload{CharacterId: N}) == N`. | `saga` |

Two seams are needed and both follow existing repo precedent
(`saga/processor_testseam.go`):

- `portal/script`: `handleEnterCommand` takes its `Gate`, its `Processor`
  constructor, and its unlock function from package-level vars that a test can
  substitute. No behaviour change in production wiring.
- `portal/dedupe`: `Gate` is an interface, so the fake is trivial.

---

## 8. Rollout, compatibility, and observability

**Order: `atlas-saga-orchestrator`, then `atlas-portal-actions`.** No wire or
schema change in either direction, so any order *works*; this order avoids a
window in which a suppressed unlock is backed by a `FAILED` that still fires
spuriously at 30 s. During that window a player whose warp succeeded would be
sent a failure message and re-unlocked 30 s later — annoying, not stuck.

**Redis compatibility.** `PendingAction` gains `Kind`. Entries written by an old
replica decode with `Kind == ""` and resolve to the current transport message
(§4.3). The new `portal-enter` lock namespace is disjoint from anything existing.

**`atlas-maps` is unchanged and must stay that way.** A same-map short-circuit in
`ChangeMap` would break FR-1 silently — the step would hang to timeout again with
no error anywhere. This is recorded in the code comment from §3.1, which is the
only place a future `atlas-maps` change would plausibly be checked against.

**Observability after the change:**

- `SAGA_TIMEOUT` on a portal warp step becomes a genuine incident signal. It is
  currently emitted on *every* portal warp, success included.
- `character_id_mismatch` skips at `Debug` quantify the party-quest fan-out
  overlap.
- `portal-enter` gate drops at `Debug` should trend to zero once layer B ships;
  a non-zero rate means some outcome path is still unlocking a moving character.

---

## 9. Verification checklist

Run from the worktree root:

1. `go test -race ./...` and `go vet ./...` in `services/atlas-portal-actions` and
   `services/atlas-saga-orchestrator`.
2. `go build ./...` in both services.
3. `tools/redis-key-guard.sh` — the gate uses `atlas.Lock`, not raw `go-redis`, so
   this must stay clean.
4. `tools/goroutine-guard.sh`.
5. `tools/lint.sh --check`.
6. `docker buildx bake atlas-portal-actions` and
   `docker buildx bake atlas-saga-orchestrator` **only if** either `go.mod`
   changed. No new dependency is expected, so no `go.mod` change is expected.
7. Live check on GMS 83.1 with the `undodraco` portal from the issue report: one
   touch → one `ENTER` command executed, one `MAP_CHANGED`, no `SAGA_TIMEOUT`.
8. `superpowers:requesting-code-review` before opening the PR.

No packet, template, or coverage-matrix work: no wire format is touched, so
`tools/template-*-guard.sh` and the packet matrix are not in play.

---

## 10. Deviations from the PRD

Recorded so plan-adherence review reads them as intentional.

| PRD | Design | Why |
|---|---|---|
| FR-2.3/2.4 — condition is "outcome's operations include a moving operation"; error branch preserved verbatim | Condition is "a moving operation was **successfully dispatched**", applied to the error branch too | Safer in both directions: a warp that failed to dispatch still unlocks; a dispatched warp followed by an unrelated operation error does not double-unlock. §4.2 |
| FR-2.5 — `handleEnterCommand` registers the `PendingAction` | `executeWarp` / `executeWarpToSavedLocation` register it | The saga transaction id is minted there and the consumer never sees it. Mirrors `executeStartInstanceTransport`, which already does this. §4.3 |
| FR-3.3 — tenant scoping comes from `libs/atlas-redis` namespacing | Composed from `redis.TenantKey` + `redis.CompositeKey` into the lock key | `Lock` is not tenant-aware (`lock.go:45`); a `TenantLock` type was rejected as out-of-scope shared-lib churn. §5.1, §6.2 |
| Open question 4 — whether `start_instance_transport` needs its own timeout | It keeps the 30 s default; only the two warp sagas get 5 s | It does strictly more work and has its own failure messaging; no evidence justifies shortening it. §4.3 |
