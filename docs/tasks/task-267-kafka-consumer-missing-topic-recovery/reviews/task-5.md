# Review: Task 5 — Enable `WatchPartitionChanges` on the consumergroup engine

Commit range: `b77baa543..ccbac99cd` (single commit `ccbac99cd`,
`feat(atlas-kafka): enable WatchPartitionChanges on the consumergroup engine`)

Brief: `.superpowers/sdd/plan/task-5-brief.md`
Report: `.superpowers/sdd/plan/task-5-report.md`

## Scope

`git diff --stat` confirms exactly the three files the brief names, nothing more:

```
libs/atlas-kafka/README.md              | 56 +++++++++++++++++++++++++++------
libs/atlas-kafka/consumer/group.go      | 24 +++++++++++---
libs/atlas-kafka/consumer/group_test.go | 17 ++++++----
3 files changed, 76 insertions(+), 21 deletions(-)
```

## Findings

### PASS — `groupConfig()` flag flip and comment (group.go)

`libs/atlas-kafka/consumer/group.go` diff shows only the doc comment above
`groupConfig()` and the `WatchPartitionChanges` field changed; `ID`, `Brokers`,
`Topics`, and `StartOffset` lines are untouched context. `WatchPartitionChanges`
goes `false` -> `true`. The new comment cites `FR-1.4` for the topology mirror,
`FR-1.1` for the deliberate divergence, `FR-1.5` for `PartitionWatchInterval`
staying at kafka-go's default, and explicitly ties safety to `awaitTopic` +
`classifyEmptyAssignment`, matching the brief's Step 3 block verbatim.

Verified the cited kafka-go source lines against the vendored module actually
pinned by `libs/atlas-kafka/go.mod` (`go list -m github.com/segmentio/kafka-go`
-> `v0.4.51`, matching one of three cached module versions on disk — checked
the right one):

- `consumergroup.go:203-205` — `PartitionWatchInterval` default assignment: line 204 is
  `config.PartitionWatchInterval = defaultPartitionWatchTime`, guarded by the
  `== 0` check on line 203. Accurate.
- `consumergroup.go:512-518` — startup `readPartitions` call (line 512) and its
  error-log branch (515-518), which returns unconditionally on *any* error.
  Accurate.
- `consumergroup.go:527-529` — the ticker branch's `case err == nil,
  errors.Is(err, UnknownTopicOrPartition):` tolerance. Accurate, and correctly
  identifies the asymmetry between the startup read (which does not tolerate
  this error) and the ticker read (which does).

No other line of `groupConfig()` or the surrounding file changed. The Task 1
nil-`pcp`-is-inert property is not referenced anywhere in this diff and remains
untouched (confirmed no other hunks in the file).

### PASS — `TestGroupConfigMirrorsTodaysTopology` inversion (group_test.go)

The assertion is correctly inverted: `if !cfg.WatchPartitionChanges { t.Fatal(...) }`,
replacing the old `if cfg.WatchPartitionChanges { t.Fatal(...) }`. The comment
now states the deliberate divergence and explains why it's only safe with the
two guards in place, matching the brief's Step 1 block verbatim. The adjacent
`cfg.Validate()` check (FR-1.3) is untouched, per the brief's explicit
instruction to leave it alone — confirmed in the diff (it's outside the
+/- hunk).

Ran the test locally to confirm it currently passes:

```
=== RUN   TestGroupConfigMirrorsTodaysTopology
--- PASS: TestGroupConfigMirrorsTodaysTopology (0.00s)
=== RUN   TestGroupConfigHonoursLastOffset
--- PASS: TestGroupConfigHonoursLastOffset (0.00s)
PASS
```

Test honesty: the implementer's report includes a RED run
(`WatchPartitionChanges = false, want true (task-267 FR-1.1)`) taken before the
flag flip, which is the correct failure message for the new polarity — this is
not a test that passes either way.

### PASS — README rewrite describes NEW behaviour, not the old caveat

`libs/atlas-kafka/README.md`'s "Partition-count changes" section is fully
replaced by "Missing topics and partition-count changes". The old text ("Neither
engine watches for partition-count changes... deliberate opt-in, not something
to enable as a side effect") is gone entirely — the new text states the
opposite of the caveat that used to gate the flag ("`WatchPartitionChanges` is
`true` on the `consumergroup` engine, so a topic repartitioned from 1 to N
partitions is picked up... with no restart"). It correctly retains a similar
warning shape but now conditioned on the two guards rather than "never enable
without opting in."

The two Task 2 snapshot fields are both documented, and the field names/JSON
tags match the actual struct in `libs/atlas-kafka/consumer/debug.go` exactly:

```
68:	IdleTicks           int       `json:"idleTicks"`
73:	AssignedPartitions []int     `json:"assignedPartitions"`
75:	LastAssignmentAt   time.Time `json:"lastAssignmentAt"`
78:	TopicMissingObservations int       `json:"topicMissingObservations"`
79:	LastTopicMissingAt       time.Time `json:"lastTopicMissingAt"`
```

README's health-read formula (`assignedPartitions non-empty, or
lastAssignmentAt after lastTopicMissingAt`) references only fields that exist
with those exact names.

The incident narrative in the README (`atlas-pr-1449`, "Twenty-two consumers in
one pod") is background color from the brief text, copied verbatim per the
brief's Step 5 instruction — reviewed as a documentation-accuracy matter only
to the extent it's internally consistent with the kafka-go line citations
above, which check out.

### PASS — `awaitTopic`/`classifyEmptyAssignment`/`errTopicMissing` preconditions already present

Confirmed these are already on the branch (from Tasks 3/4, prior commits, not
part of this diff) so the premise of this task — "only safe once both have
landed" — actually holds:

```
libs/atlas-kafka/consumer/engine_group.go:50:   if errors.Is(err, errTopicMissing) {
libs/atlas-kafka/consumer/engine_group.go:146:  // errTopicMissing is a deliberate withdrawal from the group, not a failure:
libs/atlas-kafka/consumer/engine_group.go:158:  var errTopicMissing = errors.New(...)
libs/atlas-kafka/consumer/engine_group.go:222:  return errTopicMissing
```

## Not evaluable

- The correctness of `awaitTopic`/`classifyEmptyAssignment` themselves is out
  of scope for this review (Task 3/4's surface, already landed and presumably
  separately reviewed); this review only confirmed their presence, not their
  internal correctness, since Task 5's diff does not touch them.
- Did not run `tools/verify.sh`, docker, or a repo-wide build, per
  instructions (a background run is already in progress on this tree).

## Verdict

All three files match the brief's prescribed content verbatim, the test
inversion has correct polarity and honest RED/GREEN evidence, the doc-comment
and README kafka-go line citations were independently checked against the
pinned `v0.4.51` module and are accurate, and the safety precondition
(Tasks 3/4 already landed) is confirmed present on the branch.
