# Task 5 Report — W2: disambiguate `atlas-character/character`'s two builders

## Summary

Renamed `character.NewModelBuilder()` (the zero-value model-reconstruction
builder in `model.go`) to `character.NewEmptyBuilder()` across
`services/atlas-character`, per the plan's ruling: `NewBuilder` in
`builder.go:91` is already taken by the genuinely distinct
character-creation builder, so FR-12's usual rename target collides.
`type modelBuilder` and `CloneModel` were left unchanged (exempt from
FR-13; to be recorded in `exemptions.md` by Task 25).

## Step 1 — call-site count

```
grep -rn --include='*.go' 'NewModelBuilder(' services/atlas-character | wc -l
```
Result: `88` — matches the plan's expected count exactly, in 19 files:

```
services/atlas-character/atlas.com/character/kafka/consumer/character/pending_change_applier_test.go
services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go
services/atlas-character/atlas.com/character/pending_change/refund_idempotency_test.go
services/atlas-character/atlas.com/character/pending_change/task_test.go
services/atlas-character/atlas.com/character/pending_change/processor_eligibility_test.go
services/atlas-character/atlas.com/character/character/rest_test.go
services/atlas-character/atlas.com/character/character/hp_mp_gain_test.go
services/atlas-character/atlas.com/character/character/meso_outbox_test.go
services/atlas-character/atlas.com/character/character/rest.go
services/atlas-character/atlas.com/character/character/processor_test.go
services/atlas-character/atlas.com/character/character/kafka_integration_test.go
services/atlas-character/atlas.com/character/character/provider.go
services/atlas-character/atlas.com/character/character/model_test.go
services/atlas-character/atlas.com/character/character/patch_integration_test.go
services/atlas-character/atlas.com/character/character/name_validity_resource_test.go
services/atlas-character/atlas.com/character/character/administrator_test.go
services/atlas-character/atlas.com/character/character/model.go
services/atlas-character/atlas.com/character/character/producer_test.go
services/atlas-character/atlas.com/character/character/login_logout_channel_override_test.go
```

## Step 2 — rename the declaration

`model.go:242` now reads:

```go
// NewEmptyBuilder creates a zero-valued builder for reconstructing a Model.
// The creation-flow builder is NewBuilder in builder.go; the two are distinct.
func NewEmptyBuilder() *modelBuilder {
	return &modelBuilder{}
}
```

`type modelBuilder` (model.go:211) and `CloneModel` (model.go:248, unchanged
signature) were left as-is.

## Step 3 — rewrite call sites

Applied `sed -i 's/NewModelBuilder(/NewEmptyBuilder(/g'` across the 19 files
enumerated in Step 1 (this is a repository-mechanical sweep confined to one
service, consistent with CLAUDE.md's exception for genuinely mechanical
sweeps). `character.NewModelBuilder()` and bare `NewModelBuilder()` call
sites were both covered since the pattern matches `NewModelBuilder(`
regardless of a package qualifier prefix.

## Step 4 — verify no remaining references

```
grep -rn --include='*.go' 'NewModelBuilder' services/atlas-character
```
No output (grep exit 1 / no matches). Confirmed via a fresh `cat -n` read
of `model.go` and a follow-up `git diff --stat` (19 files, 90 insertions /
88 deletions — 88 renames + 2 added comment lines).

Note: an earlier `grep` invocation during this session returned stale
results (apparently a cached hit from the `rtk` proxy) showing 78 leftover
matches including in `model.go`. A direct `cat -n` of the file and a
re-run of the same `grep` immediately after showed the rename was in fact
already fully applied; the stale output was not real. Flagging this here in
case it recurs for a reviewer re-running the same check.

## Step 5 — build and test

From `services/atlas-character/atlas.com/character`:

```
go build ./...
go vet ./...
```
Both exit 0, no output.

```
go test ./...
```
All packages with tests pass:

```
ok  	atlas-character	0.019s
ok  	atlas-character/character	7.202s
ok  	atlas-character/configuration	0.009s
ok  	atlas-character/kafka/consumer/character	18.960s
ok  	atlas-character/kafka/consumer/drop	0.021s
ok  	atlas-character/kafka/consumer/teleportrock	0.017s
ok  	atlas-character/kafka/message/character	0.005s
ok  	atlas-character/location	0.033s
ok  	atlas-character/pending_change	236.079s
ok  	atlas-character/session	0.054s
ok  	atlas-character/session/history	0.036s
ok  	atlas-character/skill	0.036s
ok  	atlas-character/teleport_rock	0.064s
[exited with code 0]
```

Re-run after adding the doc comment (cached, same result, exit code 0) to
confirm the comment-only follow-up edit did not affect behavior.

## Files changed

- `services/atlas-character/atlas.com/character/character/model.go` — renamed `NewModelBuilder` to `NewEmptyBuilder`, added doc comment; `type modelBuilder` and `CloneModel` unchanged
- `services/atlas-character/atlas.com/character/character/provider.go` — call site updated
- `services/atlas-character/atlas.com/character/character/rest.go` — call site updated
- `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` — call site updated
- Remaining 15 files (all `_test.go`) — call sites updated per the Step 1 enumeration above

`services/atlas-character/atlas.com/character/character/builder.go` was read-only reference material (the `NewBuilder`/`Builder` creation-flow builder); not modified.

## Self-review

- `type modelBuilder` and `CloneModel` confirmed unchanged (`grep -n "type modelBuilder\|type Builder\b"` shows `modelBuilder` in model.go:211 and `Builder` in builder.go:30 — no collision).
- Diff on `provider.go` and `consumer.go` spot-checked: clean 1-line renames, no unrelated changes.
- No `NewModelBuilder` references remain anywhere in `services/atlas-character` (verified twice, see Step 4 note on stale cache).
- Commit scoped to `services/atlas-character` only, staged by explicit path (no `git add -A`/`.`).
- Branch and worktree confirmed unchanged after commit: `task-263-backend-guideline-conformance` at `<repo-root>/.worktrees/task-263-backend-guideline-conformance`.

## Issues or concerns

None. The task completed cleanly within budget. One process note: a stale/cached `grep` result appeared mid-session (likely from the `rtk` token-killer proxy); re-running the same command and cross-checking with `cat -n` resolved the discrepancy and confirmed the rename was correctly and fully applied.

## Commit

`c861e29b4` — `refactor(atlas-character): rename NewModelBuilder to NewEmptyBuilder`
