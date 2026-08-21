# task-231 — code review audit

Task 40, step 3. Three modular reviewers ran in parallel against the whole-branch
diff (67 commits, merge base `bb2ac767a` → `bf5f1e446`). Each wrote its own file
so the three could run concurrently without clobbering a shared one; this page
consolidates them and records the disposition of every finding.

| Reviewer | Verdict | Detail |
|---|---|---|
| `plan-adherence-reviewer` | NEEDS_REVIEW — 39/40 IMPLEMENTED, Task 39 PARTIAL | [audit-plan-adherence.md](audit-plan-adherence.md) |
| `backend-guidelines-reviewer` | NEEDS-WORK — 6 Important, 5 Minor | [audit-backend.md](audit-backend.md) |
| `frontend-guidelines-reviewer` | NEEDS-WORK — 3 blocking, 4 non-blocking | [audit-frontend.md](audit-frontend.md) |

Beyond the three audits, Task 40 step 2 traced the nine cross-service seams by
hand — the check `tools/verify.sh` structurally cannot perform, because every
service can build, vet, test and bake clean while a seam is broken. **All nine
verified, zero gaps**, each with a producer↔consumer field-by-field comparison
across module boundaries rather than a "both tasks reported DONE" inference.

## Findings and disposition

### Fixed on this branch

| # | Finding | Where |
|---|---|---|
| B2 | **`BulkCreate` skips `ValidateConfiguration`.** The seed loader is the only production path that creates definitions, and it never validates; `Processor.Create`, which does, is wired to no route. FR-D6's documented invariant — "no path by which an unvalidatable definition reaches the table" — was false. The one correctness bug in the set. | `event/definition/subdomain.go:58-75` |
| B1 | The definition PATCH handler hand-parses the JSON:API envelope instead of using `RegisterInputHandler[T]`. | `event/definition/resource.go:38,155-214` |
| B3 | `ObserveMonsterSpawned` / `ObserveMonsterGone` / `MonsterTally` issue GORM calls directly in the processor rather than through administrator/provider. | `event/occurrence/processor.go:219-258` |
| B4 | `event/transition` has `model.go` but no processor/administrator, and is called straight from an HTTP handler. | `event/occurrence/resource.go:177` |
| B5 | DOM-21: `MonsterId uint32` reinvents `monster.Id` from `libs/atlas-constants`. | `events/crimsonbalrog/{config,evaluate,producer}.go` |
| B6 | ContiMove `state`/`subState` client wire bytes flow as free-form seed config instead of through the tenant writer-options table — the anti-pattern already litigated in task-102/task-103. Both call sites were introduced by this branch. | `atlas-channel` event + map consumers, `crimsonbalrog/config.go:41-45` |
| T39 | The boundary test caught only string literals, so a generic package importing `events/crimsonbalrog` and naming its constant passed the guard untouched — the more realistic violation shape, and the boundary the test claims to pin. | `event/boundary_test.go` |
| FE-09 | No `enabled: !!activeTenant` guard on any of the four Phase G query call sites. `TenantProvider` skips its push effect while the tenant is null, so a direct navigation fires a request with no tenant headers. | three Phase G pages |
| FE-10 | Tenant id absent from all four query keys, so a tenant switch serves the previous tenant's cached rows. | same |
| FE-14 | `as const` missing from three of five inline query keys. | same |

### Parked for a decision, not defects to fix silently

- **The occurrence wire model does not serialize `worldId`/`channelId`**, though
  the domain model and the DB carry them and the list endpoint filters on them,
  so the UI infers scope from the opaque `context`. Fixing it properly is not
  additive: "unscoped" is unrepresentable today — the entity writer always
  stores a value and `world.Id` is a byte where 0 is a valid world — so naively
  promoting the fields would render a tenant-wide event as `w0 ch0`, which is
  confidently wrong rather than merely absent. This is a wire-contract design
  decision. Currently hypothetical: CRIMSON_BALROG is the only scoped event type
  and its context does carry the keys.

### Deferred minors, recorded rather than discarded

- Task 34: `login_test.go` fixtures never distinguish `ExpMultiplier` from
  `DropMultiplier`, so a swap of `EXP_BUFF_RATE`/`ITEM_UP_BY_ITEM` would not be
  caught. The mapping was verified correct by hand; one asymmetric fixture
  closes it.
- Task 33: `main.go` wires via `init()` rather than `main()` — a new pattern for
  this repo, reviewed as sound and narrowly scoped.
- Task 36: one `getOccurrences` call per definition (N+1), forced by the absence
  of a bulk correlation path; disclosed by the implementer, not hidden.
- Task 37: no `mapId`/`voyageId` filter though the client serializes both
  (FR-UI6 does not require them); no debounce on the filter inputs.
- `registry.Handler` has drifted from Task 11's description (gained
  `ConcurrencyKeyIsConstant`, lost `Complete`) via Task 33's fix round — not a
  defect; both completion paths use `occurrence.Processor.Complete` directly.
- Backend minors: missing `TransformSlice` in three generic-layer `rest.go`
  files; `ToEntity` as a free function rather than a `Model` method;
  `events/anniversary` has no `producer.go`; both event handlers bake in
  `logrus.StandardLogger()` at startup instead of a threaded field logger; an
  `integration`-tagged test opens Postgres without registering tenant callbacks.
- Frontend minor: no centralized `eventKeys` factory (five ad-hoc keys across
  three files) — a refactor, deliberately kept out of the fix diff.

### Noted, not ours

- The `frontend-dev-guidelines` skill's `resources/*.md` describe a Next.js App
  Router architecture this repo does not use (atlas-ui is Vite + React Router).
  The reviewer followed the repo where the two disagreed.
- 5 of 11 socket-config templates (`gms_12`, `gms_48`, `gms_61`, `gms_72`,
  `gms_92`) do not route `ContiMove` at all — a pre-existing version-bring-up
  gap, evidenced in `template-routing.md`, out of scope here.

## What the reviewers confirmed clean

Worth recording, because these are the areas where a defect would have been
expensive: Kafka consumer idempotency is test-covered throughout (this repo has
a known redelivery dupe class); multi-tenancy scoping holds, including the
scheduling poller's documented cross-tenant exception; schema evolution is
backward-compatible across all five touched sibling services; the hand-rolled
JSON:API relationship/`included` extraction in the UI client is properly
narrowed with typed guards; and there are no stubs, `// TODO`s or 501s anywhere
in the diff.

The branch's central architectural claim — that a third event type needs no edit
to `event/…` — held up under adversarial reading, and is now pinned
mechanically by the Task 39 boundary test rather than by prose.
