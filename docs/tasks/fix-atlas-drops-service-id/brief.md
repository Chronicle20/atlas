# Brief — fix the two defects behind the atlas-drops outage

Worktree: `<repo-root>/.worktrees/fix-atlas-drops-service-id`
Branch: `fix-atlas-drops-service-id` (from `origin/main` @ `0b6917b6e`)
Module: `services/atlas-configurations/atlas.com/configurations`

**Read `docs/tasks/fix-atlas-drops-service-id/bug-drops-canonical-row-deleted.md`
first** — it carries the full incident evidence, the `service_history` rows, and
the reasoning for both required changes. Do not re-derive it.

The production database has ALREADY been repaired by hand. Do not write
migrations or scripts to fix data. This is a code-only change.

## Task 1 — `resolveGroup` must fail loudly instead of silently deleting

In `servicesuniq/migration.go`, `resolveGroup` currently falls back to
newest-`service_history` and then lowest-id when no candidate matches the
derived id. That fallback deleted the canonical `drops-service` row and took
production down.

Change: when NO candidate matches the derived
`uuid5(atlasServiceNS, type + "/" + environment)`, return the existing
unresolvable error naming every candidate. Remove the newest-history and
lowest-id fallbacks as automatic deleters.

Keep the derived-id rule exactly as it is. Keep `Preflight` read-only and
unchanged. Do not change the unique-index creation.

Update the existing tests that assert the removed fallbacks — they encode the
behaviour we are deliberately removing, so they must become assertions that the
ambiguous group now errors. Add a regression test named for this incident: a
group of one canonical-id row plus two newer-history rows, none matching the
derived id, must error and delete NOTHING.

## Task 2 — never insert a service-config row with an empty environment

In `services/administrator.go`, `create` inserts
`string(env.MustFromContext(ctx))`. When the `ENVIRONMENT` header is absent that
is `""`, producing a row invisible to every scoped consumer.

Change: a create must never land an empty environment. Resolve the environment
from the request context as now; if the result is empty, fall back to the
service's own configured environment, and if that is also empty, reject the
create rather than inserting an unscoped row. Return a 4xx for the rejection —
match how the package already surfaces `ErrInvalidPhase` / invalid-service-type
style validation failures rather than inventing a new pattern.

Add tests: create with the header stamps that environment; create without the
header falls back to the service's own environment; create with neither is
rejected and inserts nothing.

## Files

- `services/atlas-configurations/atlas.com/configurations/servicesuniq/migration.go` — `resolveGroup` (Task 1)
- `services/atlas-configurations/atlas.com/configurations/servicesuniq/migration_test.go` — update fallback tests, add regression test (Task 1)
- `services/atlas-configurations/atlas.com/configurations/services/administrator.go` — `create` at :16-24 (Task 2)
- `services/atlas-configurations/atlas.com/configurations/services/processor.go` — `Create` at :152 (Task 2, if the rejection belongs here)
- `services/atlas-configurations/atlas.com/configurations/services/resource.go` — `handleCreateServiceConfiguration` at :73 (Task 2, status-code mapping)
- Test file(s) beside the above for Task 2 coverage.

Patterns to copy: `services/atlas-configurations/atlas.com/configurations/environments/processor.go:43,52`
(`ErrInvalidPhase` / `ErrIllegalPhaseTransition`) for the sentinel-error +
handler-status shape.

## Verification

Module-local only: `go build ./... && go test ./...` from
`services/atlas-configurations/atlas.com/configurations`. Do NOT run
`tools/verify.sh`, `-race`, lint, or docker bake — those run outside this agent.

Do not commit to `main`. Do not use `git add -A` / `git add .`. Prefix every
Bash call with `cd <worktree> && ...` and verify the branch after committing.
