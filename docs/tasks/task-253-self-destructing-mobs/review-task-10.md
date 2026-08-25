# Review: Task 10 — atlas-monsters SELF_DESTRUCT command arm

Commit range: `9babf80fe..9c07358` (single commit `9c07358`,
"feat(atlas-monsters): consume the SELF_DESTRUCT monster command")

Brief: `.superpowers/sdd/plan/task-10-brief.md`
Report: `.superpowers/sdd/plan/task-10-report.md`

## Scope

Diff touches exactly the three files the brief named:

- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` (+15)
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go` (+11)
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go` (+23)

No other files changed. Scope matches the brief exactly.

## Checklist findings

### 1. Detonation routes through Task 7's `Processor.SelfDestruct` with `TriggerContact` — no second path

PASS. `handleSelfDestructCommand` (consumer.go:198-204) is a thin adapter:

```go
func handleSelfDestructCommand(l logrus.FieldLogger, ctx context.Context, c command[selfDestructCommandBody]) {
	if c.Type != CommandTypeSelfDestruct {
		return
	}
	monster.NewProcessor(l, ctx).SelfDestruct(c.MonsterId, c.Body.CharacterId, monster.TriggerContact)
}
```

Verified `Processor.SelfDestruct` (monster/processor.go:1847-1866) is the single
authoritative entry point Task 7 built: it re-derives monster existence,
aliveness, and `SelfDestruction()` presence itself, then hands off to the
shared `selfDestructFrom` epilogue (processor.go:1873-1899), which owns the
exactly-once transition via `Registry.SelfDestruct` and the kill bookkeeping
(`finalizeKill`). No detonation logic, no kill-epilogue duplication, and no
alternate path was introduced by this commit — the handler only supplies
`(monsterId, characterId, TriggerContact)`. `TriggerContact` constant confirmed
at processor.go:96, pre-existing (Task 7).

### 2. Command type/body/handler shape follows sibling arms; handler is registered and reachable

PASS. `CommandTypeSelfDestruct = "SELF_DESTRUCT"` added to the same const
block as every other command type (kafka.go:31). `selfDestructCommandBody`
matches the `CharacterId uint32` shape used by `killCommandBody` and
`forceControlCommandBody` (kafka.go:150-161), same doc-comment convention as
neighboring types.

`handleSelfDestructCommand` placed immediately after `handleKillCommand`
(consumer.go:198), matching the brief's "patterns to copy" instruction and the
`Type != Const { return }` early-return convention used by every sibling
handler (`handleKillCommand`, `handleCatchCommand`, `handleDrainMpCommand`,
etc. — consumer.go:189-260).

Registration confirmed present and reachable: `InitHandlers` registers
`rf(t, message.AdaptHandler(message.PersistentConfig(handleSelfDestructCommand)))`
on `t` derived from `topic.EnvProvider(l)(EnvCommandTopic)()` (consumer.go:57-59),
same topic-resolution call used by every sibling registration in the same
function. Not a declared-but-orphaned handler — it is wired into the dispatch
chain identically to its neighbors.

### 3. Tenant/context propagation matches sibling handlers

PASS. `monster.NewProcessor(l, ctx)` is the identical construction used by
every sibling handler (`handleKillCommand`, `handleDrainMpCommand`,
`handleCatchCommand`, `handleClearAggroCommand` all call
`monster.NewProcessor(l, ctx)` — consumer.go:189-230). `ctx` is the handler's
own parameter, populated upstream by the kafka consumer's tenant header
parser (`consumer.TenantHeaderParser`, consumer.go:24, unchanged by this
commit) exactly as for every other command on this topic. No divergence.

### 4. Unknown/absent monster and unmarshal-failure paths — no panic, no silent success

PASS. Every handler on `EnvCommandTopic` receives every message and
type-filters on `c.Type`; a `SELF_DESTRUCT` message unmarshals harmlessly into
unrelated `command[T]` instantiations for other handlers (zero-value fields,
no panic) exactly as pre-existing sibling command types already behave on this
shared topic — this is the topic's existing dispatch model, not something this
commit changed.

Absent/unknown monster: `Processor.SelfDestruct` (processor.go:1847-1851)
looks up the monster via `GetMonsterRegistry().GetMonster`, and on error logs
at Debug and returns — no panic, no error propagated up, matches the "every
rejection is a silent debug-level drop" contract documented directly above the
function (processor.go:1840-1845, Task 7). This is identical in shape to
`handleKillCommand`'s downstream `Kill` and `handleClearAggroCommand`'s
downstream `ClearAggro` (which logs at Error rather than swallowing — a
Task-7-owned design choice, not something this task altered).

### 5. Tests assert real behaviour, not mock call counts

PASS. Both new tests are pure JSON-unmarshal/constant-value assertions in the
same style as every pre-existing test in `kafka_test.go`
(`TestDamageCommandBody_DecodeNewShape`, etc.) — no mocks, no processor
invocation, no call-count assertions:

- `TestSelfDestructCommandUnmarshal` (kafka_test.go:80-95) unmarshals the
  brief's exact fixture and asserts `Type`, `MonsterId`, `Body.CharacterId`.
- `TestSelfDestructCommandTypeValue` (kafka_test.go:97-101) pins the wire
  constant `"SELF_DESTRUCT"` against silent drift from the channel-side mirror.

Confirmed genuine RED→GREEN: `git show 9babf80fe:.../kafka.go` contains no
`SelfDestruct` reference, so the test necessarily failed to compile before
this commit (matches the report's captured RED output, which shows
`undefined: selfDestructCommandBody` / `undefined: CommandTypeSelfDestruct`).

## Verification run (module-local, this task's surface only)

```
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./kafka/consumer/monster/...
```

Result: build succeeds, `ok atlas-monsters/kafka/consumer/monster`.

## Not evaluable

None — the full unit (contract, handler, registration, tests) sits inside a
single small, self-contained diff, and its dependency (`Processor.SelfDestruct`
/ `Registry.SelfDestruct`, Task 7/5) was already reviewable in-repo and traced
above.

## Verdict

APPROVED. No blocking or non-blocking findings.
