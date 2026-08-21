# Fix report: corpus guard count for task-250 InnerPortalHandle bindings

## What I implemented

Updated `services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go`
to account for task-250's 10 new `InnerPortalHandle` handler bindings (one on every
seed template except `gms_12`, which has no `gms_v12.yaml` registry at all — it is
structurally unrouted, not an oversight).

Two changes, both confirmed present:

1. The condition on line 63: `if total != 3317 {` → `if total != 3327 {`
2. The `want 3317` text inside the `t.Errorf` format string on line 64 → `want 3327`,
   plus a new clause appended to the end of the running narrative (still inside the
   existing parenthesization), matching the established prose style of prior clauses:

   > `... the six templates whose client has a CDragon) — plus task-250's 10
   > InnerPortalHandle handler bindings, one on every template but gms_12 (which has
   > no gms_v12.yaml registry at all, so it is structurally unrouted rather than an
   > oversight), carrying the intra-map portal entry request, USE_INNER_PORTAL)`

**Confirmation: both the condition (`total != 3327`) and the message text
(`want 3327`) were changed** — verified by re-reading the diff below; no stale
`want 3317` remains.

## Diff

```
-	if total != 3317 {
-		t.Errorf("corpus size = %d entries, want 3317 (3052 before task-206,
+	if total != 3327 {
+		t.Errorf("corpus size = %d entries, want 3327 (3052 before task-206,
...
-the six templates whose client has a CDragon))", total)
+the six templates whose client has a CDragon) — plus task-250's 10 InnerPortalHandle
+handler bindings, one on every template but gms_12 (which has no gms_v12.yaml
+registry at all, so it is structurally unrouted rather than an oversight), carrying
+the intra-map portal entry request, USE_INNER_PORTAL)", total)
```

## Testing

```
cd services/atlas-configurations/atlas.com/configurations && go test ./socket/...
```
Output:
```
ok  	atlas-configurations/socket	0.021s
```

Module-local full build/test:
```
cd services/atlas-configurations/atlas.com/configurations && go build ./... && go test ./...
```
Output: all packages `ok` or `[no test files]`; no failures. (Full output captured in
session — `atlas-configurations/socket` reported `ok` alongside every other package in
the module.)

## Files changed

- `services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go` — the
  only file touched, per the brief.

## Self-review

- Both the numeric condition and the message text were updated in lockstep — no drift
  between assertion and narrative.
- The new clause follows the existing voice: states the count, the binding target,
  the exclusion (`gms_12`) with its structural reason, and what the binding carries
  (the opcode/feature), closing inside the same outer parenthesis as the rest of the
  narrative.
- No other files were modified for this fix. Pre-existing unstaged changes to
  `docs/tasks/task-250-inner-portal-registration/agent-ledger.tsv` and `progress.md`
  and other untracked docs in that folder were left alone (not part of this fix's
  scope, not committed by me).

## Issues or concerns

None. This was a single, well-scoped, already-diagnosed bookkeeping fix.
