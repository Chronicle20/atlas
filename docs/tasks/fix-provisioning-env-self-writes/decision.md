# Decision record: admitting PROVISIONING environments at the REST gate

Companion to `brief.md`. Records what was decided, what review changed, and
what is knowingly left open.

## Why the change was needed

`ParseEnvironment` (`libs/atlas-rest/server/handler.go`) rejected any
`ENVIRONMENT` header naming a non-ACTIVE environment with 400. That made
task-232's lifecycle unsatisfiable:

- `deploy/k8s/overlays/pr-sparse/environment-record.yaml:1-6` — the record is
  created in `PROVISIONING` and "the phase is flipped to ACTIVE **last**".
- atlas-pr-bootstrap must write its service-config rows *during* provisioning;
  those writes **are** provisioning.

So the write could never succeed: ACTIVE required a completed bootstrap, and
bootstrap required ACTIVE. Observed live on PR #1411 as a 400 with
`"Request names an unknown or inactive environment."`

## What was decided

Admit known environments in `PROVISIONING` or `ACTIVE`; keep rejecting
unknown, `DEACTIVATING`, and `DELETED`. `IsActive` keeps its exact meaning —
`IsOwner` depends on it, and that is what enforces FR-5.2.

Alternative rejected: dropping the `ENVIRONMENT` header from bootstrap. It
would unblock immediately but reintroduce the `environment=''` row leak that
#1418 fixed, since the column is server-owned and a header-less caller is
stamped legacy.

## What review changed

The original justification — mine, carried into the implementer's doc comment
— claimed confinement is "enforced downstream by the scope layer". Review
proved that false as a blanket statement, and I confirmed it independently:

```
$ grep -rln 'scope\.Strict\|scope\.AuthorizeWrite' --include='*.go' services/
services/atlas-configurations
services/atlas-tenants
```

Only **two** services implement that layer. `ParseEnvironment` guards roughly
forty. For the rest, admitting the header means the request is served as any
untagged request would be. The doc comment now states this explicitly rather
than claiming a confinement that does not exist.

Why this is still acceptable: in sparse mode the non-overridden services *are*
the baseline's, so answering a `pr-<N>`-tagged request from baseline data is
the intended behaviour, not a leak across environments. The gate was never
what separated environments — `scope` does that where it exists, and
`IsOwner` governs who serves traffic. What the gate did was reject unknown
environments, which it still does.

## Knowingly left open

1. **FR-3.6 is now literally out of date.**
   `docs/tasks/task-232-sparse-ephemeral-environments/prd.md:348,543` still
   reads that a request naming an "unknown or inactive" environment is
   rejected. `PROVISIONING` is not Active, so the PRD text no longer matches
   the code. This is a sanctioned decision, not drift — but the PRD should be
   amended by whoever owns task-232 rather than silently diverging. Flagged,
   not edited here.
2. **~39 services behind the gate have no scope layer.** Whether that matters
   in practice depends on per-service deployment topology (namespace/DB per
   environment), which was out of scope to audit. If environment isolation is
   ever expected to be enforced *in-process* rather than by topology, this is
   where that gap lives.
3. **`TestParseEnvironmentRejectsADeletedEnvironment` does not exercise a
   distinct DELETED path.** `MapRegistry.Apply` removes the record on
   `PhaseDeleted`, so the test re-enters the unknown-id branch. Disclosed in
   the test's own comment; not fixable through the registry's public API.
4. **Nothing ever transitions an environment to ACTIVE.** Tracked as issue
   \#1420, together with the baseline having no environment record of its own.
   Explicitly out of scope here.

## Review outcomes

- `atlas-reviewer`: CHANGES_REQUIRED, 1 blocking — the false confinement claim
  above. Addressed by rewriting the doc comment.
- `backend-guidelines-reviewer`: CHANGES_REQUIRED, 1 blocking — DOM-20,
  `TestRecordProvisionableAcrossPhases` iterated its case table without
  `t.Run`. Addressed; subtests now run per phase. It disposed of `SEC-*` as
  N/A on the grounds that neither module is an auth service, which is a
  correct reading of the rule's trigger but means the checklist did not
  examine the widened gate — that is what `atlas-reviewer` covered instead.
