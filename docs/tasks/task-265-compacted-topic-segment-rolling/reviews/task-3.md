# Review — Task 3: Alter-phase log field and README documentation

Range reviewed: `d673eed8f..55283df28` (single commit `55283df28`,
"docs(kafka-precreate): document the compacted topic configuration").

## Scope confirmation

`git log --oneline d673eed8f..55283df28` shows exactly one commit. `git diff
--stat` confirms exactly three files touched, all under
`services/atlas-kafka-precreate/`:

- `services/atlas-kafka-precreate/main.go` (+5/-4)
- `services/atlas-kafka-precreate/README.md` (+56/-4)
- `services/atlas-kafka-precreate/internal/topics/topics.go` (+3/-2, doc
  comment only)

No deploy/overlay change, no `libs/` change. Matches the brief's stated
scope exactly. `topics_test.go` (out of scope, concurrently edited by
another implementer) was not touched by this commit and was excluded from
this review per instructions.

## Findings

### PASS — Alter-phase log field (`main.go:72-80`)

```go
logrus.WithFields(logrus.Fields{
    "phase":   "alter",
    "topics":  len(t.Compact),
    "policy":  "compact",
    "configs": topics.CompactConfigNames(),
}).Info("compacted topic configuration applied")
```

Matches the brief's Step 1 block verbatim. `phase`, `topics`, `policy`
fields preserved unchanged (no parser of this log line loses a field).
`configs` is sourced from `topics.CompactConfigNames()` — not a
hand-written literal list, satisfying the controller's explicit
requirement to avoid duplicating the config list. `topics` package was
already imported (for `topics.Ensure`); no new import added, matching the
brief's constraint.

### PASS — `topics.go` is a doc-comment-only change

`git diff` shows only the `Ensure` doc comment changed (lines 127-130,
formerly "applies cleanup.policy=compact...", now "applies the compacted
topic configuration (cleanup.policy, max.compaction.lag.ms, segment.ms,
min.cleanable.dirty.ratio)..."). No executable code in `topics.go` is
touched. The comment accurately reflects what `Ensure` does: it delegates
to `compactAlterConfigs()`, which projects `compactTopicConfigs` (all four
entries) onto the `IncrementalAlterConfigsRequestConfig` slice — confirmed
by reading `topics.go:74-79` (`compactTopicConfigs` declaration) and
`:95-103` (`compactAlterConfigs`).

### PASS — README env-var row (`README.md:21`)

Full compacted-topic configuration named with a link to the new
`#compacted-topics` anchor, verbatim match to the brief's Step 3 text.

### PASS — README Create/Alter phase bullet (`README.md:27-38`)

Verbatim match to the brief's Step 4 text, including the convergence-sweep
explanation and the "cleanup.policy=compact on its own is inert" framing
that correctly attributes the load-bearing role to `max.compaction.lag.ms`
rather than presenting the four configs as equally necessary.

### PASS — `## Compacted topics` section (`README.md:63-107`)

Verbatim match to the brief's Step 5 text, including the config table
(declaration order `cleanup.policy`, `max.compaction.lag.ms`,
`segment.ms`, `min.cleanable.dirty.ratio` — matches the code's declared
order, confirmed against `compactTopicConfigs` in `topics.go:74-79`) and
the two accuracy requirements the controller supplied:

1. **Log-start-offset is not a compaction signal.** README states this
   explicitly: "Log-start-offset is **not** the signal — compaction
   rewrites a segment in place, keeping surviving records at their
   original offsets, so a fully compacted partition can still report a log
   start offset of `0`." The sharpest signal listed is
   `cleaner-offset-checkpoint` (list item 1), matching the controller's
   guidance.
2. **`max.compaction.lag.ms` is the load-bearing knob, not all four
   equally.** The section states this in bold: "`max.compaction.lag.ms` is
   load-bearing," and explicitly demotes the other two: `segment.ms` is
   "exactly redundant given the `min()` rule" (kept for readability/
   defense-in-depth, not necessity), and `min.cleanable.dirty.ratio` "was
   not isolated by that experiment... it is cheap steady-state insurance,
   not a load-bearing knob." The README does not present the four as
   equally load-bearing anywhere.

### PASS — Anchor resolves

`## Compacted topics` heading exists at `README.md:63`; the env-var row's
`[Compacted topics](#compacted-topics)` link at `README.md:21` points at
it (grep-verified, matching heading slug convention).

### PASS — Config values verbatim

Cross-checked against `topics.go` constants:
`compactCleanupPolicy = "compact"` (topics.go:23),
`compactMaxCompactionLagMs = "600000"` (topics.go:36),
`compactSegmentMs = "600000"` (topics.go:47),
`compactMinCleanableDirtyRatio = "0.01"` (topics.go:55). README table and
env-var row values match exactly.

### PASS — Build

`go build ./...` succeeds from the module root. (`go vet ./...` reports an
unused `reflect` import in `topics_test.go`, which is outside this
commit's diff and outside this review's scope per the task instructions —
noted, not a finding against this unit.)

## Not evaluable

None. All artifacts in scope (main.go, README.md, topics.go doc comment)
were reviewed directly against the brief and cross-checked against the
existing `compactTopicConfigs` declaration.

## Verdict

APPROVED. No blocking or non-blocking findings — the commit matches the
brief's verbatim text and declaration order exactly, sources the log field
from `CompactConfigNames()` as required, and both controller-supplied
accuracy requirements (log-start-offset is not the signal;
`max.compaction.lag.ms` alone is load-bearing) are correctly reflected in
the README.
