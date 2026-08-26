# task-267 — Implementation Context

Companion to `plan.md`. Everything here is what an implementer would otherwise
have to rediscover.

## Key files

| Path | Role |
|---|---|
| `libs/atlas-kafka/consumer/group.go` | `Group`/`Generation` interfaces, the producer seams, `groupConfig()`. New seam lands here. |
| `libs/atlas-kafka/consumer/engine_group.go` | 109 lines. `startGroupEngine` (join/consume/backoff loop) and `runGenerations` (the empty-assignment branch). Both halves of the fix land here. |
| `libs/atlas-kafka/consumer/manager.go` | 691 lines. `Manager`/`Consumer` structs, the `Manager` → `Consumer` seam plumbing in `AddConsumer`, all `record*` state recorders, `Snapshot`. |
| `libs/atlas-kafka/consumer/debug.go` | 117 lines. `GET /api/debug/consumers` JSON:API attributes. Field names are lowerCamelCase. |
| `libs/atlas-kafka/consumer/fakegroup_test.go` | The broker-free harness: `fakeGroup`, `fakeGeneration`, `silentLogger`, `waitFor`, `hasLogContaining`. New `fakePartitionCounter` joins it. |
| `libs/atlas-kafka/consumer/engine_group_test.go` | The engine tests. `newGroupConsumer` (line 18) is the standard scaffold. |
| `libs/atlas-kafka/consumer/state_test.go` | `newTestConsumer` (line 8) — the minimal struct-literal `Consumer`. |
| `services/atlas-character-factory/.../factory/resource.go` | 122 lines. `handleCreateFromPreset` (line 57) swallowed its error; the sibling seed handler at line 110 shows the intended shape. |

Module roots (the `go build`/`go test` cwd): `libs/atlas-kafka`,
`services/atlas-character-factory/atlas.com/character-factory`.

## Decisions carried from design.md

1. **The watch alone is not the fix.** kafka-go v0.4.51's `partitionWatcher`
   does its first `readPartitions` outside the ticker loop and returns on *any*
   error (`consumergroup.go:512-518`), including the `UnknownTopicOrPartition`
   its own ticker branch tolerates (`:527-529`). A returning Start'd fn ends the
   generation (`:394-411`), `run` sees a nil error and takes the
   `err == nil: continue` arm without `JoinGroupBackoff` (`:713-770`). So
   `WatchPartitionChanges: true` against a missing topic is a rebalance storm.
   Hence three parts, and hence Task 5 (the flag) lands **after** Tasks 3 and 4.
2. **Seam is a producer function, not a `Group` method.** `Group` is documented
   as a pure subset of `*kafka.ConsumerGroup` (`group.go:9-11`), and the gate
   needs the lookup before a `Group` exists.
3. **`kafka.Client.Metadata`, never `kafka.Conn.ReadPartitions`.** The latter
   sends `AllowAutoTopicCreation: true` (`conn.go:984-986`); a consumer must not
   create a topic as a side effect of asking whether it exists.
4. **Snapshot is a counter+timestamp pair, superseded not cleared** — matching
   `IdleTicks`/`LastIdleTickAt`. The counter surviving recovery *is* the
   post-mortem evidence the incident lacked.
5. **Nil `pcp` is inert.** Gate disabled, classification indeterminate. This is
   what leaves every existing struct-literal-`Consumer` test unchanged; the plan
   edits only `group_test.go` (Task 1 and Task 5) and `debug_test.go` (Task 2)
   among existing tests.

## Deviations from design.md

**One, deliberate.** design §3.3 says the `errTopicMissing` sentinel arm in
`startGroupEngine` "falls through to the top of the loop" with no backoff. The
plan (Task 4, Step 3) applies the loop's existing `backoff.next()` wait — still
with **no** `recordError` and **no** Error log. Reason: a bare `continue` pairs
a definite `ErrTopicNotFound` at classification with whatever `awaitTopic`
answers a microsecond later; if that second lookup is *indeterminate* (a broker
flap), the gate returns true immediately and the engine spins leave→join with no
pacing. The backoff bounds it at 10s and costs at most 10s of recovery latency,
attributed through `recordBackoff` into `totalBackoffNs` like every other wait.

## Task sizing

Seven tasks, none over 4 files. F4 (>6 files / >1 service) should not fire.
Task 7 is verification only and touches no source.

Ordering is a hard dependency chain for Tasks 1 → 3 → 4 → 5 (seam, then gate,
then classification, then the flag that is only safe with both guards present).
Task 2 must precede Tasks 3 and 4 (both call `recordTopicMissing`). Task 6 is
fully independent and may run at any point.

## Verification gate

`tools/verify.sh` flagless must exit 0 before PR (CLAUDE.md "Done means
verified"). `--quick` is a fast intermediate check only; it skips the bake and
`-race` and does not count. The library change is consumed by all 14+ services,
so the repo-wide run is not optional here.

`tools/task-facts.sh` reports `kafka_surface=false` / `applicable_guards=none`
for the branch as it stood at plan time, because only docs had changed. Once the
library edits land, the changed-package set covers `libs/atlas-kafka/consumer`
and `services/atlas-character-factory/.../factory`; re-run `task-facts.sh` at
review time rather than trusting the plan-time snapshot.

## Symbols confirmed against the pinned dependency

Verified in `$(go env GOMODCACHE)/github.com/segmentio/kafka-go@v0.4.51`
(pinned at `libs/atlas-kafka/go.mod:9`):

- `kafka.Client{Addr net.Addr, Timeout time.Duration}` — `client.go:28-45`
- `func (c *Client) Metadata(ctx, *MetadataRequest) (*MetadataResponse, error)`, and its request carries `TopicNames` only — `metadata.go:40-44`
- `kafka.MetadataRequest{Topics []string}`, `kafka.MetadataResponse{Topics []Topic}` — `metadata.go:12-38`
- `kafka.Topic{Name string, Partitions []Partition, Error error}` — `kafka.go:14-30`; `Error` is `makeError(code, "")`, which is `nil` for code 0 — `error.go:667-675`
- `kafka.UnknownTopicOrPartition Error = 3` — `error.go:18`
- `func kafka.TCP(address ...string) net.Addr` — `address.go:9`

Atlas-side signatures confirmed: `server.NewHandlerDependency(logrus.FieldLogger, context.Context)` and `server.NewHandlerContext(jsonapi.ServerInformation)` — `libs/atlas-rest/server/context.go:17,33`.
