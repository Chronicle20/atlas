# Fix H — second post-merge flagless gate (verify-final5.log, VERIFY EXIT=1)

Branch merged `origin/main` cleanly at `598cc6c27` (2 commits: a `libs/atlas-kafka`
consumer WaitGroup fix, an overlay bump). The flagless gate reports **7 failed checks over
4 root causes**.

**The G5 fix worked and immediately earned its keep.** The lint guard now reports
`93 module(s)` instead of 92, and `libs/atlas-kafka/gen` is being built, tested and linted
**for the first time since it was created** — it had been silently invisible to the gate.
Two of the four root causes below (H1's test, H3) are pre-existing defects in that module
that no gate had ever been able to see.

## H1 — `topics.yaml` is stale (4 of the 7 failing checks)

```
scan_test.go:116: topics.yaml is stale relative to the workspace scan -- run `go run .` to regenerate
```

Fails: `go build/vet/test -race libs/atlas-kafka/gen`, `gen-topics_test.sh`
(`gen-topics.sh --check exits 0 on a clean tree` — want 0, got 1), `topic manifest drift`,
and `topic generator tests`. All four are the same staleness.

Cause: main landed new Kafka topics after the manifest was last generated — the
`atlas-player-npcs` service and now an `atlas-channel` player-npc consumer. Regenerating is
the fix, but it must happen **after H2**, because typing H2's constant as `topic.Token` is
what makes the scanner see that topic at all. Regenerating first would bake in a manifest
that is missing it.

## H2 — `atlas-channel`'s new player-npc consumer uses an untyped topic constant

```
kafka/consumer/playernpc/consumer.go:53:57: bare topic literal "EVENT_TOPIC_PLAYER_NPC_STATUS"
  reaching a topic.Token parameter; declare it as a topic.Token constant
kafka/consumer/playernpc/consumer.go:64:33: (same)
```

Identical in shape to Fix G's G2, in a third service. The declaration is
`services/atlas-channel/atlas.com/channel/kafka/consumer/playernpc/kafka.go:19`:

```go
const EnvEventTopicStatus = "EVENT_TOPIC_PLAYER_NPC_STATUS"
```

— an untyped string. The analyzer flags the *reference* reaching a `topic.Token` parameter,
not a literal at the call site (see `fix-G-review.md`, which established this reading).
Retyping the constant to `topic.Token` is the fix. Note this constant lives in the
`kafka/consumer/playernpc` package, not a `kafka/message/...` package as in the other two
services.

## H3 — staticcheck S1016 in `libs/atlas-kafka/gen`

```
libs/atlas-kafka/gen/manifest.go:50:33: S1016: should convert e (type Entry) to yamlEntry
  instead of using struct literal (staticcheck)
```

A genuine pre-existing lint defect in the gen module, surfaced only because G5 put the
module in `go.work` and therefore in the lint guard's scope for the first time.

## H4 — `verify_test.sh`: 3 bake-target assertions regressed, cause NOT yet established

```
FAIL - two changed go.mods select two bake targets   want: atlas-account,atlas-ban   got: (empty)
FAIL - two bake targets produce exactly one bake gate   want: 1   got: 0
FAIL - the gate names the target count   want: docker buildx bake (2 target(s))   got: (empty)
```

These three **passed in the previous gate** (`verify-final4.log`, where `verify_test.sh`'s
only two failures were the module count and the broken-probe assertion, both of which G5
fixed). So they are new since G5, and the empty `got` values point at a `--facts`
invocation that exited non-zero and produced no output — the signature of G5's new
fail-loudly drift check firing during the test's probe setup.

**This is a hypothesis, not a finding.** It must be confirmed by reading
`tools/verify_test.sh`'s bake-target section and G5's check in `tools/lib/go-work.sh`
before anything is changed. Two possibilities with very different fixes:

- The drift check is correct and the test's probe setup needs to keep `go.work` consistent; or
- The drift check is too eager — e.g. it should not fire on `--facts`, which executes no
  gate and has no business failing on workspace hygiene.

The second would be a real design flaw in G5 worth fixing properly, since `--facts` is
documented as the cheap "ask the gate what it selected" path.

## Caveat on this log

Stray `verify_test.sh` runs from two runaway subagents overlapped this gate's early phase
(see progress.md). They mutate `go.work.sum` and plant probe modules. H1/H2/H3 are
independently confirmed from source and are not race artifacts. **H4 is the one finding
that could plausibly be a race artifact** — confirm it reproduces on a clean tree before
treating it as real.
