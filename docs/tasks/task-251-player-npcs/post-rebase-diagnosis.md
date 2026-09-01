# Post-rebase diagnosis (branch task-251-player-npcs @ 36a16d39d)

## What happened

The branch was rebased onto `origin/main` (95 commits; `70ddc5ca0` → `7e833b2b0`),
then a second session merged `origin/main` in again and pushed
`78f07b126 fix(task-251): repair PR gate failures` plus merge commit `c0042ee65`.
Current tip is `36a16d39d`.

Conflicts resolved by hand during the rebase:

- `docs/packets/audits/STATUS.md` / `status.json` — regenerated with `packet-audit matrix`.
- `docs/packets/ida-exports/*.json` — key-level union (only new key: `CNpcPool::OnNpcImitateData`).
- `services/atlas-configurations/.../socket/corpus_test.go` — corpus total 3384 + 19 = 3403.
- `validation/{model,context}.go` — upstream (`b284bcebf`, the guidelines sweep) moved
  `ValidationContextBuilder` and `ConditionBuilder` out of these files into `builder.go`.
  Upstream's deletion was accepted and task-251's additions (`playerNpcP` field, initializers,
  `SetPlayerNpcProcessor`, `Build()` wiring, `CanSpawnPlayerNpcCondition` in `SetType` and in
  both referenceId-requirement switches) were ported into `builder.go`.
- `saga/handler.go`, `saga/handler_test.go`, saga `main.go` — union.
- `saga/event_acceptance.go` — dropped the self-completing `DeployPlayerNpc` entry; Task 23b's
  event-driven entry (`{EventKindPlayerNpcCommandSucceeded, EventKindPlayerNpcCommandFailed}`) supersedes it.
- deploy overlays + `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` — union of
  upstream's `atlas-parcel`/`atlas-kafka-precreate` with task-251's `atlas-player-npcs`; the new
  image is pinned to the same tag as its siblings in the main overlay.

## What it means

The recurring failure class is an API drift the rebase pulled in, not a bad conflict resolution:
upstream's guidelines sweep made `character.NewBuilder().Build()` return `(Model, error)` and
renamed `NewModelBuilder` → `NewBuilder`. Every task-251 test written against the old
single-value signature now fails typecheck.

Fixed so far:

- `fix(task-251): backfill fname on the ImitatedNPCData/RemoveNPC seed writers` (b8158354e) —
  `TestSeedFName_RealTemplatesInsertionCoverage` (new from upstream) requires an `fname` on every
  registry-resolvable seed writer binding; task-251's 19 bindings had none. Values are exactly what
  `packet-audit seed-fname` derives, inserted in place so the templates keep their committed key order.
- `fix(task-251): adopt character.Builder's two-value Build in the playerNpc test` (36a16d39d) —
  `validation/model_test.go` `TestCanSpawnPlayerNpcCondition`.

## Next step

`tools/verify.sh --quick` at the pre-fix tree reported 9 failing checks. Confirmed still failing at
36a16d39d:

- `go build/vet services/atlas-messages/atlas.com/messages` —
  `command/playernpc/commands_test.go:24`: `too many return values: have (character.Model, error), want (character.Model)`.
  Same `Build()` drift; apply the same fix.

The other 7 (`go analyzer guards`, `scope guard`, `producer seam guard`, `service registration guard`,
`toolchain pin guard`, `env domain guard`, `lint & format guard`) were reported on the pre-fix,
pre-merge tree and have NOT been re-confirmed at 36a16d39d — several may be cascade failures of the two
build breaks, and the merge commit may have addressed others. Re-run the gate after fixing
atlas-messages rather than assuming the list is current.

Verification note: both `verify.sh --quick` runs were invoked without `--base`, so a `libs/` change
fanned them out to all 92 modules. Pass `--base <last-gated-commit>` for iteration; the flagless run
is still required before PR.
