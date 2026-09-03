# Review: revert l2-as-default-state (6396adae4)

Brief: `docs/tasks/task-278-map-environment-object-state/diagnosis-l2-is-not-a-state.md`,
`# Fix` section.

## Scope

Single commit `6396adae4`, 27 files changed (+284/-865). Reviewed the full diff
(`git show --stat` / per-file `git show`), plus the pre-revert state of
`consumer.go` (`6396adae4^`) to confirm the new test is not vacuous, and ran
`go build`/`go test` for the three touched modules (atlas-channel, atlas-maps,
atlas-data).

`scope_confirmed`: the commit does exactly what the brief's `# Fix` section
enumerates — no extra files touched, no brief item skipped.

## 1. `EnvironmentObject` shape parity (producer vs. consumer)

- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go:75-78` and
  `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go:75-78`
  both now declare `EnvironmentObject{Kind string \`json:"kind"\`; Name string
  \`json:"name"\`}` — byte-identical struct body, tags, and updated doc
  comments on both sides. **PASS.**
- Producer: `services/atlas-maps/atlas.com/maps/map/environment/producer.go:36`
  emits `mapKafka.EnvironmentObject{Kind: string(e.Kind), Name: e.Name}` — no
  `State` field written. **PASS.**
- Consumer: `services/atlas-channel/.../consumer/map/consumer.go:1300-1310`
  reads only `o.Kind` / `o.Name` off `EnvironmentObject`; no reference to a
  `State` field remains anywhere in atlas-channel's `_map3` usage. **PASS** —
  no stale producer field, no stale consumer read.

## 2. Consumer test asserts the NEW contract, on the real wire body, non-vacuously

`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go`,
`TestHandleStatusEventEnvironmentReset_ObstacleOnlyRestore` (renamed from
`_RestoresCarriedDefaultState`):

- Decodes via the real `fieldcb.SetObjectState.Decode` codec
  (`stubDoorAnnounceForObjectState`, lines ~1210-1226), not writer names alone.
  **PASS** on "real wire body" requirement.
- Fixture clears `{ENVIRONMENT, gate}` and `{OBSTACLE, obs3}`; assertion is
  exactly one captured announce, `{SetObjectStateWriter, obs3, 0}` — i.e.
  ENVIRONMENT produces no `SetObjectState` and OBSTACLE still produces state
  0. **PASS** on contract match.
- `FieldObstacleAllReset` is still sent unconditionally ahead of the loop
  (`consumer.go:1288-1290`, unchanged by this commit) and two sibling tests
  (`_AllResetRouted`, `_AllResetUnrouted`) were updated in the same commit to
  drop the now-absent `gate` announcement from their `want` slices. **PASS.**
- Non-vacuousness confirmed by diffing against the pre-revert handler
  (`6396adae4^:.../consumer.go:1296-1306`): with the new fixture shape
  (`State` field absent ⇒ Go zero value 0), the *old* handler's
  `state := o.State; if kind == Obstacle { state = 0 }` would still call
  `announceObjectState` for the ENVIRONMENT entry (with state 0), producing
  **two** `SetObjectState` announces where the new test expects **one**. The
  test would fail against the pre-revert handler. **PASS.**

## 3. No dangling references to deleted atlas-data endpoint / atlas-maps object package

- `grep -rn "maps/data/map/object\|data/map/object"` across `services/`:
  zero hits.
- `grep -rn "mapId}/objects\|GetObjects\|objectProvider"` across `services/`:
  zero hits.
- `services/atlas-data/docs/rest.md`: the deleted `/objects` endpoint section
  is gone; the one remaining `objects` grep hit (line 831, "Array of lose item
  objects") is unrelated inventory-endpoint prose.
- `services/atlas-maps/atlas.com/maps/data/map/` directory listing after the
  commit contains only `info/ monster/ reactor/ script/` — no `object/`.
- `go build ./...` succeeds clean for `atlas-channel`, `atlas-maps`, and
  `atlas-data` module roots (no compile-time stragglers, e.g. in generated
  route config or mocks). **PASS.**
- `mock/processor.go` diff (`-9` lines, part of `git show --stat`) drops
  `GetObjects`; grep confirms no remaining caller. **PASS.**

## 4. Ingress route (`e050686d4`) and adjacent paths unaffected

- `git diff e050686d4 6396adae4 -- deploy/k8s/base/routes.conf.template.generated deploy/shared/routes.conf`
  is empty — the `.../environment` ingress location is byte-identical between
  the two commits. **PASS.**
- `handleStatusEventEnvironmentStateChanged` (`consumer.go:1249-1272`) is
  untouched by this diff; it uses `EnvironmentStateChanged`, a distinct wire
  type from `EnvironmentReset`/`EnvironmentObject`, and is not part of the
  revert's blast radius per the diagnosis. **PASS.**
- `environment.Registry`/`Processor.Reset`/`Processor.Set` retain their
  tracking/clear-on-empty mechanics; only the `DefaultState` field, method,
  and its atlas-data resolution (`defaultStateOf`) were removed
  (`registry.go`, `processor.go` diffs). Callers of `Reset`
  (`services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go:149`,
  `services/atlas-maps/atlas.com/maps/map/processor.go:115`) are not part of
  this diff. **PASS**, consistent with "reset-on-empty and replay-on-enter
  unaffected."

## Test / build verification

- `go build ./...` clean for atlas-channel, atlas-maps, atlas-data module
  roots.
- `go test ./kafka/consumer/map/...` (atlas-channel): `ok`.
- `go test ./map/environment/... ./kafka/...` (atlas-maps): all `ok` / no test
  files where expected.
- `go test ./map/...` (atlas-data): `ok`.
- `gofmt -l` on every Go file touched in this commit: no output (all
  formatted).

## Docs

- `docs/tasks/task-278-map-environment-object-state/design.md` §1.2 corrected
  in place, points at the diagnosis file, matches brief wording. **PASS.**
- `docs/tasks/task-278-map-environment-object-state/progress.md` records the
  revert and explicitly supersedes the old "Remaining work" item per the
  brief. **PASS.**

## Findings

None blocking. None non-blocking beyond what's already noted above (all
items PASS with cited evidence).

## Not evaluable

- Live-client re-verification of the new "no announce for ENVIRONMENT"
  behavior is asserted by the diagnosis file's evidence table (already
  established per task instructions, not re-derived here) — this review
  covers only the code/test correctness of the revert itself, not a fresh
  live session.

## Verdict

APPROVED. Every brief item in the diagnosis's `# Fix` section is present in
the commit, the two `EnvironmentObject` copies stay identical, the rewritten
consumer test asserts the new contract against the real wire codec and is
demonstrably non-vacuous, no dangling references to the deleted atlas-data
endpoint or atlas-maps object package remain, and the ingress route plus
adjacent ENVIRONMENT_STATE_CHANGED/reset/replay paths are confirmed
byte-identical or untouched.
