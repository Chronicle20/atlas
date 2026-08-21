# Review — Task 11: atlas-channel SELF_DESTRUCT command + deathType passthrough

Commit range: `9c07358..61457cf1a` (single commit `61457cf1a`)
Brief: `.superpowers/sdd/plan/task-11-brief.md`
Report: `.superpowers/sdd/plan/task-11-report.md`

## Scope

Diff touches exactly the 7 files the brief named:

- `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` (+17)
- `services/atlas-channel/atlas.com/channel/monster/producer.go` (+19)
- `services/atlas-channel/atlas.com/channel/monster/processor.go` (+7)
- `services/atlas-channel/atlas.com/channel/monster/mock/processor.go` (+8)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go` (+23/-12)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go` (+49, new tests)
- `services/atlas-channel/atlas.com/channel/monster/producer_test.go` (+47, new tests)

No scope creep. Matches the brief 1:1.

## 1. Cross-service contract match (highest priority)

Checked `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` and `consumer.go` (Task 10, already landed):

- `CommandTypeSelfDestruct = "SELF_DESTRUCT"` — identical string on both sides
  (channel: `kafka.go:23`; monsters: `kafka.go:32`).
- Body shape: channel's `SelfDestructCommandBody{CharacterId uint32 \`json:"characterId"\`}`
  (`kafka.go:127-130`) matches monsters' unexported `selfDestructCommandBody{CharacterId uint32 \`json:"characterId"\`}`
  (`kafka.go:159-161`) — same field name, same type, same JSON tag. Byte-for-byte match confirmed
  by reading both struct definitions directly, not by trusting identifier names.
- `StatusEventKilledBody.DeathType byte \`json:"deathType"\`` and
  `StatusEventDestroyedBody.DeathType byte \`json:"deathType"\`` (channel `kafka.go:225-228,232-235`)
  match the producer side in `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:105,148`
  (`DeathType byte \`json:"deathType"\`` on both `statusEventDestroyedBody` and `statusEventKilledBody`).
- `DestroyType` enum values (`libs/atlas-packet/monster/clientbound/destroy.go:24-38`:
  Disappear=0, FadeOut=1, Bomb=2, DestructByMiss=3, Swallow=4, SelfDestruct=5) match the
  `destroyTypeFor` table exactly, and match `TestDestroyTypeFor`'s expected values.

PASS — no drift found anywhere in the wire contract.

## 2. `deathType` passthrough

`destroyTypeFor` (`consumer.go:214-220`):
```go
func destroyTypeFor(deathType byte) monsterpkt.DestroyType {
	if deathType == 0 {
		return monsterpkt.DestroyTypeFadeOut
	}
	return monsterpkt.DestroyType(deathType)
}
```
Both `handleStatusEventDestroyed` (`consumer.go:198`) and `handleStatusEventKilled`
(`consumer.go:309`) now call `destroyTypeFor(e.Body.DeathType)` and pass the result through
`destroyForSession`/`killForSession` into `monsterpkt.NewMonsterDestroy(uniqueId, dt)`. The
prior hardcoded `monsterpkt.DestroyTypeFadeOut` argument is gone from both call sites — confirmed
by diff, no remaining reference to the old hardcode.

Default/unset case (`deathType == 0`): resolves to `DestroyTypeFadeOut`, exactly the value the
pre-commit hardcode always sent. Byte-identical to pre-task-253 behaviour for every ordinary kill
where an old (or unmodified) atlas-monsters build doesn't set the field — no regression.
`TestStatusEventKilledBodyDecodesDeathType` independently pins that an omitted `deathType` JSON
key decodes to Go zero value `0` (`consumer_test.go`), and `TestDestroyTypeFor`'s "producer
omitted the field" case pins `0 → DestroyTypeFadeOut`.

PASS.

## 3. D2 (no `action != 0` pattern-match)

`git show 61457cf1a | grep -n "action != 0"` — no hits. `destroyTypeFor`'s only conditional is
`deathType == 0`, a distinct default-value check for the rolling-deploy gap, not the D2 discard
pattern. PASS.

## 4. Mock wiring

`ProcessorMock.SelfDestructFunc func(f field.Model, monsterId uint32, characterId uint32) error`
(`mock/processor.go:26`) and `(m *ProcessorMock) SelfDestruct(...)` (`mock/processor.go:138-143`)
match the `Processor` interface signature added at `processor.go:31` and the real impl at
`processor.go:161-164`. Nil-safe, mirrors `Kill`'s shape exactly. PASS.

## 5. Test honesty

- `TestDestroyTypeFor` — table-driven, calls the real unexported `destroyTypeFor` and compares
  against real `monsterpkt.DestroyType*` constants. Not mock-based.
- `TestStatusEventKilledBodyDecodesDeathType` — real `json.Unmarshal` into the real
  `monster2.StatusEventKilledBody` struct, asserting the decoded field value for both the
  present and omitted-key cases. Not mock-based.
- `TestSelfDestructCommandProviderShape` — calls the real `SelfDestructCommandProvider`,
  `json.Unmarshal`s the produced Kafka message value into the real `monster2.Command[...]` type,
  and asserts `Type`, `MonsterId`, `Body.CharacterId`, `WorldId`, `ChannelId`, `MapId`, and the
  message key against `producer.CreateKey(7001)`. Not a call-count assertion.

All three tests assert real decoded/produced wire shapes and would fail if the implementation
were wrong (verified by reading assertions against real types, not test doubles). PASS.

Confirmed via `go build ./... && go test ./kafka/consumer/monster/... ./monster/...` in
`services/atlas-channel/atlas.com/channel` — all pass, no `FAIL` in output.

## 6. Disclosed deviation: same-level pair vs. curried step

The brief said "take a `dt monsterpkt.DestroyType` parameter alongside `uniqueId`," which the
implementer read as ambiguous between:
- `func(uniqueId uint32) func(dt monsterpkt.DestroyType) model2.Operator[session.Model]` (extra
  curry level), or
- `func(uniqueId uint32, dt monsterpkt.DestroyType) model2.Operator[session.Model]` (flat pair) —
  what landed.

Checked the surrounding file's convention: `killForSession`/`destroyForSession` are curried
through three prior levels (`l` → `ctx` → `wp` → final step) because each of those levels is
independently supplied at a different call scope (`l` at handler-construction time, `ctx`/`wp`
per-invocation, the final args at the point of use inside `ForSessionsInMap`). Within the file,
`spawnThenControlOperator` (`consumer.go:475`) — a comparably-shaped helper returning
`model2.Operator[session.Model]` — takes multiple flat parameters
(`l, ctx, wp, m, aggro`) rather than currying every argument. There is no existing two-argument
final curry step anywhere in this file to be inconsistent with; the file's convention curries
across *call-scope* boundaries, not merely because a step has more than one logical value.
`uniqueId` and `dt` both come from the exact same call site (`e.UniqueId` and
`destroyTypeFor(e.Body.DeathType)`, both read off the same event) with no independent scoping
need to split them.

**Judgment: acceptable, not a finding.** The flat pair does not break the file's currying
convention — it correctly identifies that `uniqueId`/`dt` share a call scope, unlike the
`l`/`ctx`/`wp` levels above them — and it has precedent (`spawnThenControlOperator`) for a
flat multi-argument final step in this same file. No reshape needed.

## Not evaluable

None — the full contract on both sides of the boundary was directly readable and was read.

## Non-blocking notes

- The implementer's report discloses that Steps 3-5 (implementation) were written before Step 1
  (tests), inverting the brief's prescribed RED→GREEN order, and explains why a literal RED
  capture was skipped (avoiding a stash-based revert in a shared worktree). This is a process
  deviation from the brief's literal step ordering, not a correctness defect — the tests do
  assert real, non-trivial decoded behaviour (see §5) and would fail without the implementation.
  Noting for completeness per the report's own disclosure; not blocking.

## Verdict

APPROVED — all five brief interfaces (command type, command body, producer, processor + mock,
consumer passthrough) present and byte-for-byte consistent with the landed atlas-monsters
consumer; the deathType default is unchanged from the pre-commit hardcode; D2 is honoured; tests
assert real decoded/produced values; the disclosed signature-shape deviation is a reasonable
judgment call consistent with the file's actual (not merely superficial) currying convention.
