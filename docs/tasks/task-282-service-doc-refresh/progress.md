# task-282: service documentation refresh

## What was done

A full `/service-doc` sweep across all 68 Go services (every `services/*/`
containing a `go.mod`). One `service-documentation` agent per service, pinned
to Sonnet, run in the worktree `.worktrees/task-282-service-doc-refresh`.
Runner caps concurrency at 20, so the queue was fed as slots freed.

Result: 49 services took changes, 19 were verified accurate and untouched.
Committed as `fc3388510`, 182 files, +11485/-475. Documentation only — no
code, config, or behavior changed.

Excluded as non-Go-services (agent does not apply): atlas-ui (Next.js
frontend), atlas-gachapons (canonical data + scripts), atlas-pr-bootstrap
(Dockerfile/scripts), atlas-wz-extractor (only a `.env`).

## Known-incomplete work (two agents hit their budget and said so)

1. **atlas-channel** — added 17 previously undocumented domains, 18 topics,
   5 external services and 5 registries, but did NOT re-audit the ~50
   pre-existing `docs/domain.md` sections for line-level drift against the 47
   commits since the last doc pass. Those sections may still be stale.
2. **atlas-saga-orchestrator** — rebuilt the saga-type table (8 -> 28) and
   action table (~90 -> 123), but did NOT audit the "Compensation strategies
   by action type" list in `docs/domain.md` against the now-123 actions.
   `compensator.go` is 3471 lines with ~56 methods; the list is curated and
   may omit strategies for newly documented actions.

Both are follow-up passes, each scoped to one file in one service.

## Cross-cutting observations (not acted on)

- Storage docs repo-wide abbreviate Redis keys as `{tenantId}`, while
  `libs/atlas-redis` actually embeds `{region}:{major}.{minor}` in that
  segment. Consistent everywhere; a repo-wide convention, not a defect.
- Stray module-root `README.md` files sit inside the Go module rather than at
  the service root the DOCS.md contract specifies: `atlas-mini-games` and
  `atlas-trades` (`services/*/atlas.com/*/README.md`). Left untouched.
- `atlas-merchant/docs/rest.md` and `atlas-renders/docs/rest.md` document
  `/debug/consumers` and `/readyz` as endpoints. Both predate this branch and
  are unchanged by it; the convention applied here is that operational routes
  are not the public HTTP interface.
- Code-level (not doc) inconsistency found by the atlas-query-aggregator
  agent: `validation/rest.go`'s `questStatus` error message labels the enum
  `UNDEFINED=0, NOT_STARTED=1, STARTED=2, COMPLETED=3`, but `quest/model.go`
  defines `NotStarted=0, Started=1, Completed=2`. Docs match the code; the
  error string does not.

## Next step

Code review, then PR. Nothing else is pending.
