# Review: Task 4 — Thread `duration` / `byItemOption` through atlas-channel

Commit range reviewed: `6c6aac7f9..676e5e6db` (single commit `676e5e6db`).

## Scope confirmation

`git diff --stat 6c6aac7f9..676e5e6db` shows exactly the six files the brief's Files
block names:

- `character/expression/processor.go`
- `character/expression/producer.go`
- `character/expression/producer_test.go` (new)
- `kafka/consumer/expression/consumer.go`
- `kafka/message/expression/kafka.go`
- `socket/handler/character_expression.go`

No extra files, no scope creep. `git log --oneline 676e5e6db..HEAD` shows Task 5
(`a9f733017`, range/ownership guards) is already committed on top in this worktree;
`socket/handler/character_expression_test.go` (which exercises the range/ownership gate)
belongs to that later commit, not this one — confirmed it doesn't exist at either
`6c6aac7f9` or `676e5e6db` (`git show 6c6aac7f9:...` → "exists on disk, but not in
6c6aac7f9"). Not a Task 4 scope violation.

Scope confirmed: matches the brief.

## Requirement-by-requirement

1. **`kafka.go` — `Duration int32`/`ByItemOption bool` added to both `Command` and
   `Event`, after `Expression`.** Confirmed at
   `services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go` — both
   structs gained the two fields with identical tags `json:"duration"` /
   `json:"byItemOption"`, matching Task 3's `atlas-expressions` `StatusEvent`/`Command`
   tags exactly (`services/atlas-expressions/atlas.com/expressions/kafka/message/expression/kafka.go`).
   PASS.

2. **`SetCommandProvider` widened, no clamp.**
   `character/expression/producer.go` — signature gained `duration int32, byItemOption bool`,
   both forwarded verbatim into `expression2.Command{Duration: duration, ByItemOption: byItemOption}`.
   No `if duration < 0` anywhere. PASS.

3. **`Processor.Change` widened (interface + impl), Debugf arg fixed.**
   `character/expression/processor.go` — interface and impl both gained
   `duration int32, byItemOption bool`; the log line's second `Debugf` arg changed from
   `f.MapId()` to `expression` per brief's incidental fix. PASS.

4. **New `producer_test.go`, no test-helper file, mirrors `atlas-expressions/expression/producer.go` shape.**
   Table-driven `TestSetCommandProviderCarriesDurationAndByItemOption` with the three
   cases from the brief (`-1`/false, `3000`/true, `0`/false), asserting
   `CharacterId`/`Expression`/`MapId`/`Duration`/`ByItemOption` via `json.Unmarshal` of
   the produced Kafka message. Ran it:
   ```
   go test ./character/expression/... -v
   --- PASS: TestSetCommandProviderCarriesDurationAndByItemOption (0.00s)
       --- PASS: .../v95_extra_expression (0.00s)
       --- PASS: .../item_option_set (0.00s)
       --- PASS: .../pre-v95_zero_values (0.00s)
   ```
   Not a `*_testhelpers.go` file — it's a genuine `_test.go`. PASS.

5. **`consumer.go` — TODO block deleted, `false`/`0` literal replaced by real event
   fields, no clamp.** The five-line `task-028 follow-up` comment is gone (confirmed:
   `grep -rn "task-028 follow-up" kafka/consumer/expression/` → no output). Replaced with
   a grounding comment citing the IDA evidence for `nDuration = -1` and explaining the
   deliberate bit-pattern-preserving narrowing. The call site now reads
   `charpkt.NewCharacterExpression(e.CharacterId, e.Expression, uint32(e.Duration), e.ByItemOption)`
   — both `e.Duration` and `e.ByItemOption` are the real event fields, no more
   placeholder literals. PASS.

6. **`character_expression.go` — only the `Change` call changed.**
   `git diff` for this file shows exactly one line changed (the last line), from
   `.Change(s.CharacterId(), s.Field(), p.Emote())` to
   `.Change(s.CharacterId(), s.Field(), p.Emote(), p.Duration(), p.ByItemOption())`.
   No guard logic (range check / ownership lookup) added here — that's Task 5's
   commit, confirmed above to be a separate later commit. PASS.

7. **`character_cash_item_use.go` untouched.** Not present in `git diff --stat` for
   this range. PASS.

8. **No wire-format change.** Only JSON message-envelope structs (`kafka.go`) and
   internal Go signatures changed; `libs/atlas-packet/character/clientbound/expression.go`
   and `serverbound/expression.go` (the `Encode`/`Decode` version gates) are untouched by
   this diff. PASS.

## Deferred item 1 — is `consumer.go`'s `uint32(e.Duration)` the only narrowing site?

Grepped the full expression-command/event surface on both sides of the seam
(`atlas-channel/character/expression`, `atlas-channel/kafka/message/expression`,
`atlas-channel/socket/handler/character_expression.go`, `atlas-expressions/expression`,
`atlas-expressions/kafka/message/expression`) for `uint32(`. The only duration-related
hit is `kafka/consumer/expression/consumer.go:63`:
```go
charpkt.NewCharacterExpression(e.CharacterId, e.Expression, uint32(e.Duration), e.ByItemOption)
```
Every other `uint32(...)` hit in those directories is an unrelated `CharacterId`/
`Expression`/index conversion in existing tests, not `Duration`. `libs/atlas-packet`'s
`NewCharacterExpression` constructor takes `duration uint32` (confirmed in
`character/clientbound/expression.go`), so this is the correct — and only — place the
signed/unsigned boundary is crossed. **Confirmed: single narrowing site, correct.**

## Deferred item 2 — does the JSON contract round-trip `-1` as `-1`, not `4294967295`?

Compared struct field types on both sides of the Kafka seam:

- `atlas-expressions` producer (`services/atlas-expressions/atlas.com/expressions/kafka/message/expression/kafka.go`):
  `StatusEvent.Duration int32 \`json:"duration"\`` and `Command.Duration int32`.
- `atlas-channel` consumer (`services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go`):
  `Event.Duration int32 \`json:"duration"\`` and `Command.Duration int32`.

Both sides use `int32`, so Go's `encoding/json` marshals `-1` as the JSON number
literal `-1` and unmarshals it back into an `int32` field as `-1` — no intermediate
`uint32` anywhere in the JSON path. The only place `uint32` appears is the wire-packet
constructor call in `consumer.go` (deferred item 1), which is downstream of the
JSON decode, not part of it. **Confirmed: the contract round-trips `-1` as `-1` before
narrowing; no premature/incorrect narrowing on either side.**

## Build/test verification (module-local, as an implementer would run)

```
cd services/atlas-channel/atlas.com/channel
go build ./...                                                   # exit 0
go test ./character/expression/... ./kafka/consumer/expression/... ./socket/handler/...
ok  	atlas-channel/character/expression	(cached)
?   	atlas-channel/kafka/consumer/expression	[no test files]
ok  	atlas-channel/socket/handler	(cached)
```
Matches the report's claimed GREEN state.

## Not evaluable

- Wire-packet `Encode`/`Decode` byte-for-byte stability (`v61_test.go`/`v72_test.go`/
  `v79_test.go` in `libs/atlas-packet`) — those files are untouched by this diff and
  belong to an earlier task's surface (Task 2); not re-verified here since correctness
  of this task does not depend on them (this task changes JSON-message structs and
  Go signatures only, never the packet codec).
- `atlas-expressions` producer's own behavior — reviewed and approved in Task 3's
  review; this task only consumes its JSON contract.

## Verdict

All Task 4 requirements are met, both items deferred from Task 3's review are
resolved with evidence, no clamp exists anywhere on the `duration` path, and the
build/tests pass locally. No blocking findings.
