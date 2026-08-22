# Review: Task 2 — `group_state` probe, `seed_group` message classification, skip-and-record seeding pass

Range reviewed: `6415c5ce6..073117395` (single commit `073117395`, message
`feat(kafka-precreate): probe group state and skip active groups when
seeding offsets`).

Note on process: the worktree's HEAD has since moved to `457365ea0`
("fix(kafka-precreate): warn instead of failing the Job for skipped active
groups" — evidently Task 3). I confirmed by extracting
`git show 073117395:deploy/k8s/base/kafka-precreate.sh` into a scratch copy
and diffing it against both the pre-image (`6415c5ce6`) and the live worktree
file, so every finding below is scoped to the exact `073117395` snapshot, not
to later work that has already landed on the branch.

## Scope confirmed

Single file touched: `deploy/k8s/base/kafka-precreate.sh` (`git diff --stat
6415c5ce6..073117395` → `1 file changed, 124 insertions(+), 8 deletions(-)`).
Matches the brief's file list exactly (`group_state` added; `seed_group` and
`seed_override_offsets` rewritten). `deploy/k8s/base/atlas-kafka-precreate_test.sh`
is not touched by this commit — consistent with the brief, which only asks
this task to run the existing suite, not extend it.

## Findings

### PASS — `group_state` matches the brief's provided code verbatim

`deploy/k8s/base/kafka-precreate.sh:142-149` (at `073117395`) is a
byte-for-byte match of the brief's Step 1 code block: `set +e` / capture /
`set -e` region spans exactly the one command substitution (no `$?` read,
which the brief's own comment explains — the probe "never fails, never exits
non-zero" and callers only consume stdout). Uses `"$KAFKA_BIN/kafka-consumer-groups.sh"`
full path (satisfies the `$KAFKA_BIN`-on-every-invocation constraint). The
`awk 'NF>=6 && $(NF-1)!="STATE" { print $(NF-1); exit }'` idiom matches the
measured-facts requirement: header excluded by `$(NF-1)!="STATE"`, never by
line number or field count.

### PASS — `seed_group` classifies on message before exit code, correct return contract

`deploy/k8s/base/kafka-precreate.sh:210-236`. `set +e` region spans exactly
one command substitution (`seed_out=$(...)`) plus its `$?` read
(`seed_rc=$?`) — matches the NFR. `case "$seed_out" in *"Assignments can only
be reset if the group"*) return 2 ;; esac` runs unconditionally, before the
`[ "$seed_rc" -ne 0 ]` check — matches "message case must run BEFORE the
exit-code check." No blanket `|| true` anywhere on the reset call; the
`else` branch (`[ "$seed_rc" -ne 0 ]`) prints the captured output to stderr
and `return`s the real `$seed_rc`, so broker-unreachable / authorization /
malformed-argument failures still propagate. `2>&1` capture used in place of
the old `>/dev/null`, so the previously-silent refusal is now visible on the
fatal path. `$KAFKA_BIN` full path used on the one Kafka CLI invocation.

### PASS — `seed_override_offsets` probes, skips, records, and preserves the NG6 early return

`deploy/k8s/base/kafka-precreate.sh:280-330`. The
`KAFKA_CONSUMER_GROUP`-unset early return (lines 281-284) is verified
byte-identical to the pre-image (`git diff` shows zero changes to those four
lines) and is still the first statement in the function, ahead of any Kafka
call — matches the binding constraint. The per-group loop probes
`group_state`, gates on `state_is_seedable`, and on a seedable group calls
`seed_group … || seed_rc=$?` (the required pattern so `set -e` does not abort
on a deliberate `2`). Branches: `0` → increment `seeded_count`; `2` → append
to `$skipped_groups`, increment `skipped_count`, log; anything else → `exit
"$seed_rc"` (fatal, unchanged from the brief). An already-active group (per
`state_is_seedable`) is skipped the same way without ever calling
`seed_group`. `$skipped_groups` / `$seeded_count` / `$skipped_count` are
produced as documented interfaces for Task 3 to consume. Body is a verbatim
match of the brief's Step 3 code block.

### PASS — dual-use portability constraints held

`bash -n`, `sh -n`, and `shellcheck -S error` all exit 0 against the exact
`073117395` snapshot of both `kafka-precreate.sh` and
`atlas-kafka-precreate_test.sh` (verified by extracting the commit blob to a
scratch copy, not the live worktree, since the worktree has since advanced to
Task 3's commit). `sh atlas-kafka-precreate_test.sh` prints the NG6 PASS, the
`state_is_seedable` PASS, then `SKIP: BOOTSTRAP_SERVERS unset`, exit 0 —
matches the baseline exactly. Grepped the diff region for `local`, `[[`,
`+=`, `$'...'`, and arrays — none introduced. No CR bytes in the diff (line
endings preserved).

### PASS — `verify_group_offsets` correctly left untouched

Confirmed via the commit-blob diff against the pre-image: zero changes to
`verify_group_offsets` in this commit. This is the plan's explicit
intermediate state (skip set produced but not yet consumed) and is correct
per the task brief's scope note — not flagged as a defect.

### Non-blocking — `seed_rc` global-variable name shared between `seed_group` and its caller

`seed_group` assigns to the global `seed_rc` internally
(`kafka-precreate.sh:225`, no `local` available under the dual-use
constraint), and `seed_override_offsets`'s loop also uses a variable named
`seed_rc` for the same purpose (`kafka-precreate.sh:303-305`). I traced all
three return paths (`return 2`, `return "$seed_rc"`, `return 0`) and
confirmed the caller's `seed_rc=$?` (fired via `||` on any non-zero return)
always overwrites the shared variable with the function's actual return
value, and the `return 0` path is only reachable when the internal `seed_rc`
was already `0`, so the shared name does not currently produce an observable
bug. This exact naming choice is dictated by the brief's own Step 3 code
block, not an implementer deviation — but it is a fragile pattern (a future
edit that lets `seed_group` return early on a different internal `seed_rc`
value, or that reorders the caller's `seed_rc=0` initialization, would
silently regress). Worth a rename in a follow-up, not blocking this task.

### Not evaluable

- The real `--describe --state` and `--reset-offsets --execute` output
  against an actual Kafka broker was not exercised (`BOOTSTRAP_SERVERS`
  unset in this environment — the same `SKIP` the test suite itself reports).
  The measured-facts constraints (blank-line/header/five-token row, message
  text, exit-0-on-refusal) are taken as given per the plan's instruction not
  to re-derive them, and the code matches those facts as documented; live
  verification against a broker is out of this review's reach.

## Verdict rationale

All binding constraints for this task are met: dual-use portability, full
`$KAFKA_BIN` paths, unchanged line endings, byte-identical/first NG6 early
return, narrowed `set +e` regions, no blanket `|| true`, correct `seed_group`
return contract with the caller using `rc=0; … || rc=$?`, and the baseline
shellcheck/test-suite output unchanged. The one item raised is a latent
fragility inherited from the brief's own specified code, not a functional
defect, and does not block.
