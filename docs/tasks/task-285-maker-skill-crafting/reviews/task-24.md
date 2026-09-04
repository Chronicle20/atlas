# Task 24 Review — `atlas-maker` REST surface and error codes

Commit range: `534212bfc..3d30bf023` (single commit `3d30bf023`, "feat(maker): expose maker recipe and craft REST endpoints")

## Scope

`git diff --stat 534212bfc..3d30bf023`:

```
services/atlas-maker/README.md                                          |  30 +-
.../atlas-maker/atlas.com/maker/craft/emitter.go                        |  36 ++
.../atlas-maker/atlas.com/maker/craft/errors.go                         |  98 ++++
.../atlas-maker/atlas.com/maker/craft/processor.go                      |  62 --
.../atlas-maker/atlas.com/maker/craft/resource.go                       | 237 ++++++++
.../atlas.com/maker/craft/resource_test.go                              | 643 +++++++++++++++++++++
.../atlas-maker/atlas.com/maker/craft/rest.go                           | 143 +++++
services/atlas-maker/atlas.com/maker/main.go                            |   2 +
.../atlas-maker/atlas.com/maker/rest/handler.go                         |  44 ++
```

All changes land inside `services/atlas-maker/`. Two files beyond the brief's
literal list (`craft/emitter.go`, additions to `rest/handler.go`) — both are
required for `main.go`'s wiring to be real rather than stubbed, and both are
justified in the report. No scope creep found.

`go build ./...` and `go vet ./...` from `services/atlas-maker/atlas.com/maker`
are clean; `go test ./... -count=1` passes (all 12 packages).

## Focus area 1 — all eleven error-code rows tested

Read (not reasoned about) `craft/resource_test.go:398-529`,
`TestEveryErrorCodeIsReturnedByItsOwnCondition`. Ran it directly:

```
$ go test ./craft/... -run TestEveryErrorCodeIsReturnedByItsOwnCondition -v -count=1
--- PASS: TestEveryErrorCodeIsReturnedByItsOwnCondition (0.01s)
    --- PASS: .../recipe_not_found
    --- PASS: .../level_too_low
    --- PASS: .../skill_level_too_low
    --- PASS: .../insufficient_materials
    --- PASS: .../missing_prerequisite_item
    --- PASS: .../missing_prerequisite_quest
    --- PASS: .../insufficient_mesos
    --- PASS: .../inventory_full
    --- PASS: .../equip_not_found
    --- PASS: .../no_crystal_mapping
    --- PASS: .../craft_in_progress
```

Each sub-test provokes a distinct condition (a distinct harness mutation or
mock error, `resource_test.go:426-498`) and `assertCraftError` (line 531)
asserts both `resp.StatusCode` and the JSON:API `code` field. All eleven
brief rows are present, each provoking exactly its own condition. **PASS.**

## Focus area 2 — `TestFailureLeavesStateUnchanged`

`resource_test.go:579-616`. It captures `h.character`/`h.etc`/`h.equip`
before the request, issues it, and asserts `assert.Equal` (struct equality,
field-for-field) after, plus `assert.Empty(t, d.em.calls, ...)` — a real
assertion, not a comment reasoning about the property. **PASS on the
"asserted not reasoned about" requirement.**

However, it only exercises 4 of the 11 rejection rows: `level_too_low`,
`insufficient_mesos`, `inventory_full`, `missing_prerequisite_quest`
(`resource_test.go:585-590`). The brief's literal text says "for every
rejection condition." `skill_level_too_low`, `insufficient_materials`,
`missing_prerequisite_item`, `equip_not_found`, `no_crystal_mapping`, and
`craft_in_progress` are not exercised by this test (their zero-emission
property is separately covered by
`TestEveryErrorCodeIsReturnedByItsOwnCondition`'s `d.em.calls` assertion at
`resource_test.go:526`, but that test does not assert
character/etc/equip-field equality). **Non-blocking finding** —
`resource_test.go:582-590` covers a representative subset, not literally
every row the brief names.

## Focus area 3 — `TestRecipeRoutesAreReadOnly`

`resource_test.go:618-643`. Confirmed by direct run:

```
$ go test ./craft/... -run TestRecipeRoutesAreReadOnly -v -count=1
--- PASS: TestRecipeRoutesAreReadOnly
    --- PASS: POST/.../recipes, PUT/.../recipes, PATCH/.../recipes, DELETE/.../recipes
    --- PASS: POST/.../recipes/1082002, PUT/.../recipes/1082002, PATCH/.../recipes/1082002, DELETE/.../recipes/1082002
```

Both recipe routes, all four write methods, all assert `405`. `resource.go:97-100`
registers `handleMethodNotAllowed` explicitly for each write method on both
paths, rather than relying on gorilla/mux's implicit mismatch handling.
**PASS.**

## Focus area 4 — `Request.WorldId`/`ChannelId` populated and reach `craft.Processor`

`rest.go:80-91` (`CraftRequestRestModel`) carries `WorldId byte`/`ChannelId
byte` from the JSON body; `rest.go:107-123` (`ToRequest`) maps them onto
`Request.WorldId`/`Request.ChannelId` unconditionally (no silent drop, no
hardcoded zero). `processor.go:167-168,250-251,353-354` confirm
`req.WorldId`/`req.ChannelId` are read out of `Request` into the
award-payload steps for all three modes. **Code is correct.**

**Non-blocking finding**: no test in this commit exercises a non-zero
`worldId`/`channelId` in the POST body and asserts it reaches the emitted
saga. `craftBody()` (`resource_test.go:391-393`) and every literal JSON body
in the error-table tests omit `worldId`/`channelId` entirely, so every test
in this file exercises the zero-value path only. The wiring is visibly
correct by inspection, but the specific propagation this task was assigned
(per Task 23's reviewer note) has no positive test evidence at the REST
boundary.

## Focus area 5 — `craft/emitter.go`, the Kafka `SagaEmitter`, `COMMAND_TOPIC_SAGA`

Confirmed genuinely pre-provisioned, not a deploy-manifest omission:
- `deploy/k8s/base/env-configmap.yaml:88`: `COMMAND_TOPIC_SAGA:
  "COMMAND_TOPIC_SAGA"`, in the `atlas-env` ConfigMap
  (`deploy/k8s/base/env-configmap.yaml:3-4`).
- `deploy/k8s/base/atlas-maker.yaml:24-26`: the deployment already does
  `envFrom: configMapRef: name: atlas-env`, so every key in that ConfigMap,
  including `COMMAND_TOPIC_SAGA`, is already in atlas-maker's environment
  with no service-specific override needed.

Producer-convention check against `services/atlas-map-actions/atlas.com/map-actions/`
(the pattern the report cites): map-actions declares `EnvCommandTopic =
"COMMAND_TOPIC_SAGA"` (`kafka/message/saga/kafka.go:6`) and calls
`producer.ProviderImpl(l)(ctx)(saga.EnvCommandTopic)(CreateCommandProvider(s))`
(`saga/processor.go:33`), where `CreateCommandProvider` is
`producer.SingleMessageProvider(key, &s)` keyed by `TransactionId`
(`saga/producer.go:11-14`). `craft/emitter.go:34-36`'s `Emit` does the
identical three calls inline (`producer.SingleMessageProvider` then
`producer.ProviderImpl(...)(EnvCommandTopic)(provider)`), same key
derivation. **Matches the cited convention. PASS.**

## Focus area 6 — `craft/errors.go` is a same-semantics move

Diffed `git show 534212bfc:.../craft/processor.go` against
`git show 3d30bf023:.../craft/processor.go`: the removed block (old lines
72-133: `Code`, the `Code` const block, `CraftError`, `Error()`,
`ErrRecipeNotFound`/`ErrCraftInProgress`, `reasonToCraftError`) is
byte-identical to what now lives in `craft/errors.go:12-72` (only the
`errorDocument`/`writeCraftError` addition — new code, not a move — sits
after it). No case in `reasonToCraftError`'s switch changed, no status code
changed, no new default branch. **Confirmed a pure file-organization move,
zero semantic change. PASS.**

Also confirmed `CodeEquipNotFound`/`CodeNoCrystalMapping` (the two codes
`reasonToCraftError`'s switch does not produce) are unchanged Task 23
values, constructed directly in `processor.go:209,305,316` from
`recipe.ErrNoCrystalMapping` and the disassemble-slot check — not touched by
this commit, not silently dropped.

## In-flight guard release — plan gap, verdict

Read `craft/inflight.go:16-28,60-64`: `craftGuard` is an unexported,
process-local `*inflightGuard` package variable inside atlas-maker's
`craft` package — a `sync.Mutex`-guarded Go map with no persistence, no
RPC surface, no Kafka topic of its own.

Read `docs/tasks/task-285-maker-skill-crafting/plan.md:3360-3406` (Task 26,
`atlas-channel` — the `MAKER_RESULT` writer and terminal-event consumer):
its Files section (`plan.md:3364-3369`) places the terminal-event consumer
at `services/atlas-channel/atlas.com/channel/kafka/consumer/maker/consumer.go`
— a different service, a different process, a different Go module than
`atlas-maker`. Line 3406 nonetheless states: "The consumer must also
**release Task 23's in-flight guard** on every terminal event, including
timeout and compensation."

**Verdict: as currently planned, the guard can never be released except by
an atlas-maker pod restart.** `atlas-channel`'s consumer has no mechanism —
no REST call, no shared memory, no message back to atlas-maker — by which
it could reach atlas-maker's in-process `craftGuard` map. Calling
`Processor.ReleaseInFlight` (the method Task 23 built for this) requires
code running inside atlas-maker's own process, consuming the same saga
terminal-event topic atlas-channel's Task 26 consumer reads. No task from
20 through 27 in `plan.md` adds such a consumer to atlas-maker; Task 24's
own Files section (this commit) does not include one either.

This is a genuine plan defect, not something this commit introduced or was
asked to fix — Task 24's brief scoped it to REST files only, and the
implementer correctly declined to invent an unspecified Kafka consumer
(topic contract, dispatch shape) rather than guess. The report's own
escalation (an atlas-maker-side terminal-event consumer, currently unowned
by any numbered task, must be added before Task 27's gate) is accurate and
should be acted on by the controller — either as an addition to Task 24's
own follow-up, a correction to Task 26's brief, or a new task. It does not
block approval of this commit, since this commit does exactly what its own
brief specified and flagged the gap rather than silently landing an
unreviewed guess.

## Other checks

- `rest/handler.go`'s additions (`ParseCharacterId`, `InputHandler[M]`,
  `ParseInput`, `RegisterInputHandler`) mirror `atlas-ban/atlas.com/ban/rest/handler.go`'s
  shape (`server.ParseIntId[uint32]`, `jsonapi.Unmarshal` into a generic
  model, `server.RetrieveSpan`/`server.ParseTenant` chain) — consistent
  with repo convention.
- `main.go:66`: `AddRouteInitializer(craft.InitResource(GetServer())(db))`
  — matches the existing `reagent`/`crystalband`/`seed` call order
  (`si)(db)`, not the brief's literal `(db)(si)`); the report flags this
  explicitly as a documented deviation for internal consistency. Reasonable.
- README (`services/atlas-maker/README.md`) documents all three routes, the
  full eleven-row error table, and the `COMMAND_TOPIC_SAGA` Kafka-produces
  row — matches what the code does.
- Self-disclosed test-package duplication (`rBuildRecipe` etc. duplicating
  `eligibility_test.go`'s `craft_test`-package helpers, because
  `resource_test.go` is internal `package craft` for the
  `processorFactory` injection seam) is a legitimate, disclosed trade-off,
  not a defect.

## Verdict

APPROVED_WITH_FINDINGS. The implementation correctly satisfies the brief's
routes, error codes, read-only enforcement, and the errors.go/emitter.go
additions are sound and verified against real deploy manifests and sibling
service conventions. The two non-blocking findings are test-completeness
gaps (not correctness bugs) against the brief's literal wording. The
in-flight-guard release gap is real and significant to the overall
program but is not this commit's defect — it is correctly escalated rather
than silently absorbed.

---

```
verdict: APPROVED_WITH_FINDINGS
artifact: .superpowers/sdd/plan/task-24-review.md
scope_confirmed: commit 3d30bf023 — craft/errors.go, craft/emitter.go, craft/resource.go, craft/rest.go, craft/resource_test.go, craft/processor.go (Code/CraftError moved out), main.go route registration, rest/handler.go additions, README.md. No files touched outside atlas-maker.
blocking: 0
non_blocking: 3
  - services/atlas-maker/atlas.com/maker/craft/resource_test.go:582-590 — TestFailureLeavesStateUnchanged exercises 4 of the 11 rejection rows, not "every rejection condition" as the brief's Step 1 literally specifies (zero-emission is separately covered for all 11 by TestEveryErrorCodeIsReturnedByItsOwnCondition, but field-for-field state equality is not).
  - services/atlas-maker/atlas.com/maker/craft/resource_test.go:391-393 — no test supplies a non-zero worldId/channelId in the POST body and asserts it reaches the emitted saga; rest.go:107-123's mapping is correct by inspection but has no positive REST-boundary test evidence.
  - docs/tasks/task-285-maker-skill-crafting/plan.md:3406 — Task 26 (atlas-channel) is assigned to release Task 23's in-flight guard, but that guard is a process-local map inside atlas-maker (craft/inflight.go:64); no task in the plan adds the atlas-maker-side consumer needed to actually call ReleaseInFlight. Not a defect in this commit — flag for the controller before Task 27's gate.
not_evaluable: 0
```
