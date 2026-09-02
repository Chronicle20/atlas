# task-278 — progress / handoff

## Where things stand

The feature is implemented and was verified end-to-end against a live client on
2026-09-02 (pr-1566, tenant `d1defcd0-9994-4ee2-bcad-9eaaa1377839`, map 103000800,
object `gate`). Details and log timestamps are in `manual-test-notes.md`.

Live-verified before the reset fix: state change → visible client change,
reset-on-empty, replay-on-enter. The `DELETE` reset was verified as **broken**,
and that fix is the last three commits.

## Commits on `task-278-map-environment-object-state`

| Commit | Change |
|---|---|
| `e050686d4` | ingress route for `.../environment` (endpoint previously fell through the `^/api/worlds(/.*)?$` catch-all to atlas-world) |
| `da20524d3` | atlas-data parses the WZ `obj` node and serves `GET /data/maps/{mapId}/objects` |
| `b3706768d` | `EnvironmentObject` gains `state`; atlas-maps resolves each object's default on first track; atlas-channel announces it instead of hardcoded `0` |
| `5efddd01b` | errcheck fix in the new atlas-data endpoint test |

## The defect that drove the last three commits

`handleStatusEventEnvironmentReset` restored a hardcoded `0` for every cleared
object. For map 103000800's `gate` the visible state is `0` (observed in-client —
do NOT re-derive this from the WZ tree; an earlier pass read
`Obj/effect.img/quest/gate/0` as "invisible" and got the polarity backwards) and
the map declares default `l2=1`, so `DELETE` left the gate visible — a PQ stage
resetting into its completed look.

`0` remains correct for `ObjectKindObstacle`, where `FieldObstacleAllReset`
genuinely means "all off"; that path is preserved.

The blocker was that no service knew an object's declared default —
`atlas-data/.../map/reader.go` never parsed the `obj` node (only `shipObj`). That
is what `da20524d3` adds.

Note: this widened task-278 into atlas-data. It was done in this worktree by
explicit user direction rather than spun out as a separate task, so
`followup-reset-default-brief.md` describes work that is now IMPLEMENTED HERE.

## Cross-service seam

`EnvironmentReset.Cleared[]` gained a `state` field. atlas-channel is the sole
consumer (`consumer.go:134`). `TestHandleStatusEventEnvironmentReset_RestoresCarriedDefaultState`
asserts the new contract and was shown RED against the old hardcoded `0`:

```
captured announcements = [{SetObjectState gate 0} {SetObjectState obs3 0}],
                    want [{SetObjectState gate 1} {SetObjectState obs3 0}]
```

It decodes the real `fieldcb.SetObjectState` wire body, not just writer names.

## Review rounds after the post-merge commits

| Round | Agent | Verdict | Artifact |
|---|---|---|---|
| Post-merge commits `9b7817a13..5efddd01b` | `task-reviewer` | APPROVED | `reviews/post-merge-fix-round.md` |
| Post-merge backend guidelines | `backend-guidelines-reviewer` | CHANGES_REQUIRED (5 blocking) | `audit-postmerge.md` |
| Fix round 2 → `6c9acd433` | `task-implementer` + `task-reviewer` | APPROVED | `reviews/fix-round-2.md` |
| Fix round 3 → `0054591a5` | `task-implementer` | DONE (2-line lint fix) | `fix-round-3-report.md` |

Four of the five blocking findings were fixed in `6c9acd433`: DOM-01 (`builder.go`
for `atlas-maps/.../data/map/object/`), DOM-04 (`Transform` in that package's
`rest.go`), EXT-02 (httptest-backed drain test for the new atlas-data client),
DOM-20 (`TestGetObjects` rewritten table-driven).

**The fifth was ruled out by the user, deliberately.** DOM-04 against
`services/atlas-data/atlas.com/data/map/object/rest.go` is a documented
exception: that package has no `Model` type — `reader.go` builds `RestModel`
straight from the WZ node — and all four siblings (`map/monster`, `map/npc`,
`map/portal`, `map/reactor`) are likewise `rest.go`-only. Satisfying the rule
would have made the newest of five packages the only one shaped differently.
Do not "fix" this in a later pass without revisiting that ruling.

## Remaining work

1. ~~Flagless `tools/verify.sh` must exit 0.~~ **DONE — PASS, exit 0** at branch
   head `0054591a5`. Full flagless run (69 Go modules built/vetted/tested with
   `-race`, plus every guard including lint & format): `All checks passed.`
   Log: `/tmp/t278/verify4.log`. The run at `6c9acd433` failed on a single
   staticcheck QF1012 in the new `processor_test.go`, fixed in `0054591a5`.
2. ~~**Code review**~~ **DONE** — see the table above. All rounds settled;
   nothing outstanding.
3. **Live re-test of reset.** Needs a NEW image; deployed `pr-1566-e050686`
   predates `da20524d3`/`b3706768d`. Procedure: set `gate` to a non-default state,
   confirm visually, `DELETE`, confirm the gate returns to the map's initial
   appearance instead of staying set.

## Unrelated latent hazard

atlas-channel's service-config row (`e7fb1d7e-47b8-46bd-97dc-867d93530000`) in
pr-1566 still carries `environment: 'main'` while the namespace is env `6a89`
with an empty `environments` table. It is healthy only because it holds a
snapshot fetched earlier and will likely crash-loop on its next restart, as
atlas-login did. Repair is DELETE + POST of the config row with the id preserved
(no `ENVIRONMENT` header). See `manual-test-notes.md` for the full incident.
