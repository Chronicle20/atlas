# Review — Task 19: Saga `ShowParcel` action and the SHOW_PARCEL command

Commit range: `4d75fe7ed..a476d638f` (single commit `a476d638f`).
Brief: `.superpowers/sdd/plan/task-19-brief.md`.
Report: `.superpowers/sdd/plan/task-19-report.md`.

## Scope

`git diff --stat 4d75fe7ed..a476d638f` — 11 files, 212 insertions, 0
deletions, all in `libs/atlas-saga` and
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`. Matches the
brief's stated module roots. `services/atlas-channel` untouched, as required
(a different in-flight fix round is there and out of scope). Scope confirmed.

## Priority 1 — SHOW_PARCEL envelope, field by field

`libs/atlas-saga/payloads.go` `ShowParcelPayload`:
`CharacterId uint32`, `NpcId uint32`, `WorldId world.Id`,
`ChannelId channel.Id`, `Quick bool` — matches the brief's "Interfaces
produced" block verbatim, including JSON tags (`characterId`, `npcId`,
`worldId`, `channelId`, `quick`).

`services/atlas-saga-orchestrator/.../kafka/message/parcel/kafka.go`:
- `EnvCommandTopic = "COMMAND_TOPIC_PARCEL"` — matches brief.
- `CommandTypeShowParcel = "SHOW_PARCEL"` — matches brief.
- `ShowParcelCommand{TransactionId, WorldId, ChannelId, CharacterId, NpcId,
  Quick, Type}` — the brief only names the type, not its field set. I
  compared it against the existing `storage.ShowStorageCommand`
  (`services/atlas-saga-orchestrator/.../kafka/message/storage/kafka.go:110-120`),
  which has the identical shape minus `AccountId` (storage needs it, parcel
  doesn't per the brief) plus `Quick`. Field-for-field consistent with the
  established sibling command and with the payload that produces it
  (`saga/handler.go:2614-2639`, `parcel/producer.go:112-127`).

Topic env-var convention: `EnvCommandTopic = "COMMAND_TOPIC_PARCEL"` follows
the exact naming convention used by every other command topic in this package
(`COMMAND_TOPIC_STORAGE`, `COMMAND_TOPIC_PARCEL_CUSTODY`,
`COMMAND_TOPIC_NPC_SHOP`, etc. — confirmed by grep across
`kafka/message/*/kafka.go`). It is not registered in
`deploy/k8s/base/env-configmap.yaml` yet (that file only has
`COMMAND_TOPIC_PARCEL_CUSTODY`, added in a separate registration commit
`37a0fb601`), and the plan (`plan.md`) never schedules an env-configmap entry
for `COMMAND_TOPIC_PARCEL` either. This is **not a functional defect**: the
resolution path (`atlas-kafka/topic.EnvProvider`) falls back to the literal
token as the topic name when the env var is unset
(`os.LookupEnv` miss → `return token, nil`), which is exactly the value the
configmap's identity mapping (`KEY: "KEY"`) would have supplied anyway. Flagged
as a non-blocking configuration-hygiene note, not a cross-service defect —
it does not change behavior and is not something task 19's brief asked for.

Verdict: no divergence in name, type, or JSON tag found. PASS.

## Priority 2 — additive-only discipline in `libs/atlas-saga`

- `model.go:236` — `ShowParcel Action = "show_parcel"` inserted as a new
  block member; `Action` is a string type (not iota-backed), so insertion
  cannot renumber or reorder any existing constant. Diff shows pure addition,
  no existing line touched.
- `payloads.go` — `ShowParcelPayload` inserted between
  `WithdrawFromParcelPayload` and `RequestGuildNamePayload`; no existing
  struct's fields were touched.
- `unmarshal.go:495-500` — `case ShowParcel:` inserted before `case
  RequestGuildName:`; the switch's existing cases and their bodies are
  untouched (diff shows insertion only, no reflow).
- `unmarshal_test.go` — two new tests appended after the last existing test
  in the file (`TestUnmarshalWithdrawFromParcelStep`); no existing test
  modified.

Verdict: purely additive. PASS.

## Priority 3 — the three files outside the brief's `### Files` list

- **`parcel/producer.go`** — adds `ShowParcelCommandProvider`. The brief's
  "Patterns to copy" section explicitly points at
  `storage/processor.go:103-130` (`ShowStorageAndEmit`), and that pattern's
  provider function (`ShowStorageCommandProvider`) lives in
  `storage/producer.go:83`, not `storage/processor.go`. The parcel package
  mirrors that split exactly (provider in `producer.go`, dispatch in
  `processor.go`). This is the brief's file list being incomplete about a
  companion file the copied pattern requires, not scope creep — confirmed by
  diffing the two packages' shapes side by side.
- **`saga/model.go`** — adds the orchestrator-local aliases
  `ShowParcel = sharedsaga.ShowParcel` and
  `ShowParcelPayload = sharedsaga.ShowParcelPayload`. Every other shared-saga
  action/payload used unqualified in this package's `handler.go` has a
  matching alias here (confirmed: `WithdrawFromParcel`,
  `WithdrawFromParcelPayload` sit immediately above the new lines,
  `saga/model.go:209-210`, `:350-351`). Handler.go references `ShowParcel`
  and `ShowParcelPayload` unqualified — the package would not compile without
  this alias. Required, not scope creep.
- **`saga/parcel_compensation_test.go`** — adds two no-op methods
  (`ShowParcelAndEmit`, `ShowParcel`) to the hand-written `parcelTestMock` so
  it keeps satisfying `parcel.Processor` after the interface grew two new
  methods. Read the full file
  (`services/atlas-saga-orchestrator/.../saga/parcel_compensation_test.go`):
  both new methods are bare `return nil` / a closure returning `nil`, with no
  call counter, no captured argument, and nothing asserted against them
  anywhere in the file — unlike the mock's other methods (e.g.
  `RestoreParcelAndEmit`, `RemoveParcelAndEmit`), which do increment counters
  and are asserted on by the compensation tests. The new methods are inert
  interface-satisfaction fallout exactly as characterized, not a test that
  silently pins a fabricated success.

Verdict: all three are genuinely required, none is scope creep. PASS.

## Priority 4 — `ShowParcel` self-completion

- `saga/event_acceptance.go:258-261` — `sharedsaga.ShowParcel: {}` (empty
  `EventKind` slice), placed identically to `sharedsaga.ShowStorage: {}`
  (`event_acceptance.go:216`). A step whose action maps to an empty
  acceptance list cannot be left waiting for an event that never arrives —
  same shape used by every other self-completing action in this table.
- `saga/handler.go:2614-2639` `handleShowParcel` — byte-for-byte structurally
  identical to `handleShowStorage`
  (`saga/handler.go:2269-2287`): type-assert payload → build `channel.Model`
  → call the `...AndEmit` producer → on success, unconditionally call
  `NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)` (error
  ignored via `_ =`, matching the existing `handleShowStorage`'s exact
  pattern) → `return nil`. On the emit error path, it returns the error
  without completing the step — again identical to `handleShowStorage`, and
  consistent with the design's stated self-completion semantics.
- `GetHandler` dispatch table (`handler.go:944-945`) adds
  `case ShowParcel: return h.handleShowParcel, true`, and the `Handler`
  interface (`handler.go:154`) declares `handleShowParcel`. No other switch
  or table required an entry that was missed (grepped `ShowStorage` across
  the package as the reference action and found the corresponding
  `ShowParcel` entry in each of the same three places: interface, dispatch
  table, event-acceptance table).

Verdict: self-completion treated identically to `ShowStorage`; cannot wedge a
saga. PASS.

## Test honesty

`TestUnmarshalShowParcelStep` / `TestUnmarshalShowParcelStep_Quick`
(`libs/atlas-saga/unmarshal_test.go`) assert `CharacterId == 100`,
`NpcId == 2030`, `WorldId == 0`, `ChannelId == 1`, and `Quick` both `false`
and `true` — these are literal fixture values chosen for the test, not values
read back off the implementation (the implementation has no derived/computed
defaults for these fields; they are pass-through struct fields). The
`Quick: true`/`Quick: false` split directly exercises the two-entry-point
distinction the brief calls out, and both tests would fail against the
pre-change code (compile failure — `ShowParcel`/`ShowParcelPayload`
undefined), matching the brief's stated RED. Not a pinning test.

## Not evaluable

- Whether Task 21's atlas-channel consumer will in fact bind correctly to
  this envelope — that consumer does not exist yet in this range; verified
  the envelope's shape against the brief and the sibling `ShowStorageCommand`
  pattern instead, which is the best available proxy at this point in the
  build order.
- Runtime behavior of the Kafka topic resolution (whether
  `COMMAND_TOPIC_PARCEL` truly falls back correctly in a live cluster) was
  traced through `atlas-kafka/topic.EnvProvider` source but not executed;
  `tools/verify.sh` was not run per instructions.

## Verdict

No blocking defects found. One non-blocking configuration-hygiene note (the
missing `env-configmap.yaml` entry, functionally covered by the library's
env-var fallback, and not scheduled anywhere in the plan — worth a one-line
addition whenever the next task touches that file, but not a functional gap
introduced by this commit).
