# Task 17 review — `atlas-maker` `reagent` seeded table

Range: `1c61312..b16ff729e` (3 commits: `1c61312` derivation doc, `062373736` domain package,
`b16ff729e` routing fix). Reviewed against `.superpowers/sdd/plan/task-17-brief.md` (controller
addendum authoritative), `task-17-brief-fix1.md`, `task-17-report.md`, and
`docs/tasks/task-285-maker-skill-crafting/reagent-derivation.md`.

## Verdict: APPROVED

## Seed fidelity (highest-value check)

Compared all 45 files under `deploy/seed/gms/83_1/reagents/reagent-*.json` against
`reagent-derivation.md` §3.1 row by row (id, stat, value) — **exact match, all 45 rows, no
invented/dropped/altered row.**

- **Full item ids, not truncated.** Every seed file's `"id"` is the full decimal item id
  (`"4250000"` … `"4251402"`), matching the derivation's finding-1 caveat about the Hex-Rays
  `char` artifact. `libs/atlas-constants/item.Id` is `uint32` (`libs/atlas-constants/item/constants.go:5`),
  and `reagent/entity.go:19` stores `ReagentItemId uint32` — no narrowing anywhere in the read or
  write path. `builder_test.go`'s `TestBuilderKeepsTheFullItemId` (`item.Id(4251402)`) pins this.
- **Stat names carried verbatim.** `builder.go`'s `ValidStats` slice reproduces the derivation's
  §3.2 15-name list character-for-character, including case (`incPAD`, `incMAD`, …, `incReqLevel`,
  `randOption`, `randStat`); `builder_test.go:TestBuilderRejectsUnknownStat` explicitly rejects
  `"incpad"` (wrong case) and `"incPAD "` (trailing space), proving no normalization occurs.
- **`randOption`/`randStat` in the valid set, not additive.** Both are members of `ValidStats`
  (`reagent/builder.go:23-38`) so the seed round-trips (rows 40-45), and `builder.go:14-17` /
  `model.go` document explicitly that they are equip variance keys, not `stat += value` —
  matching addendum finding 3. Nothing in this change applies them as additive stats (no
  application logic exists yet; that is Task 22/23's job per the addendum).
- **One `(stat, value)` pair per gem.** `Model` (`reagent/model.go:14-20`) carries exactly one
  `stat`/`value` field pair, not a collection — matches the derivation's first-non-zero,
  never-accumulate selection rule (addendum finding 2, derivation §1.4).
- **`int16` is lossless and not misdescribed.** `model.go:40-42` and `entity.go:20-22` both
  document `Value` as a signed delta and note `incReqLevel` is negative; grepped the whole
  package for `int32`/`32-bit`/`16-bit` comments — none exist. No comment claims `int16` is the
  client's width (the client's `long` claim lives only in the derivation doc, correctly).

## Interfaces contract (Tasks 22-23 dependency)

Verified against the brief's line 58 and the addendum:

- `reagent.Model.ReagentItemId() item.Id` — `model.go:29`
- `reagent.Model.Stat() string` — `model.go:35`
- `reagent.Model.Value() int16` — `model.go:40`
- `reagent.Processor.GetByItemId(itemId item.Id) (Model, error)` — `processor.go:26`, returning
  `ErrNotFound` (wrapped with the id) on a genuine miss, distinguishable via `errors.Is` — pinned
  by `processor_test.go:TestGetByItemIdNotFound` and `TestGetByItemIdNotFoundAcrossTenants`. This
  is exactly what FR-3.2 (drop unheld reagent, don't fail the craft) needs.
- `reagent.Processor.GetAll() ([]Model, error)` — `processor.go:24`, tenant-scoped, pinned by
  `TestGetAllReturnsEveryReagentForTheTenant`.
- `reagent/mock/processor.go` implements the same interface (`var _ reagent.Processor = ...`
  compile-time assertion at `mock/processor.go:14`) and is ready for Task 22/23 to consume.

## Tenant scoping

`entity.go:18-19` places `uniqueIndex:idx_reagents_tenant_item` on `(TenantId, ReagentItemId)`
across both columns — correct composite key, and all 45 seed ids are distinct so the index can
never collide within one tenant. Reads go through `database.Query`/`WithContext` following the
exact `gachapon/provider.go` pattern (context-scoped tenant filtering is the shared library's job,
not this package's) — same shape the codebase already trusts for `gachapon`.
`processor_test.go:TestGetByItemIdIsTenantScoped` and `TestGetByItemIdNotFoundAcrossTenants` seed
the same item id under two tenants with different values and assert each tenant reads only its
own row, and that tenant B gets `ErrNotFound` for a row that exists only under tenant A. No
cross-tenant read is reachable in the code path exercised.

## Routing fix (`b16ff729e`)

- `deploy/shared/routes.conf:696-699` adds `location ~ ^/api/reagents(/.*)?$` pointing at
  `atlas-maker:8080`, placed immediately after the existing `/api/maker` block — domain-named,
  not re-prefixed under `/maker`, consistent with the `/api/gachapons` precedent at line ~465 and
  with the controller's ruling (concern 1, already settled — not re-litigated here).
- `deploy/k8s/base/routes.conf.template.generated:696-699` was regenerated (not hand-edited) and
  templates the new block identically to its `/api/maker` neighbour:
  `atlas-maker.${NS_ATLAS_MAKER}.svc.cluster.local:8080` — confirmed `NS_ATLAS_MAKER` is reused,
  not a new/invented namespace variable.
- `tools/gen-routes.sh --check` → `gen-routes: up to date`.
- `tools/overlay-env-guard.sh` → all `PASS`/expected `SKIP`, no `FAIL`.
- `ns-vars.generated.yaml` correctly has no diff (`NS_ATLAS_MAKER` pre-existed) — verified with
  `git diff --stat 1c61312..b16ff729e -- deploy/k8s/base/ns-vars.generated.yaml` (empty).
- Fix round stayed in scope: only `deploy/shared/routes.conf` and
  `deploy/k8s/base/routes.conf.template.generated` changed in `b16ff729e`; the `reagent` package,
  seed files, and tests are untouched by that commit.

## Read-only enforcement (FR-2.3)

`resource.go` registers explicit 405 handlers for `POST/PUT/PATCH/DELETE` on both the collection
and `{itemId:[0-9]+}` routes, rather than relying on gorilla/mux's implicit method-mismatch
behaviour — the report's documented reasoning (mux only surfaces the last-tried route's mismatch,
which degrades to 404 once the seed routes mount under the same `/reagents` prefix) is corroborated
by `resource_test.go:TestResourceStaysReadOnlyBesideTheSeedRoutes`, which composes the router the
way `main.go` actually does (reagent routes + seed routes on the same prefix) and asserts both the
405s and the seed route's reachability. This is a materially better test than one that only
exercises `reagent.InitResource` in isolation, since the in-isolation test would not have caught
the composition bug the report describes.

## Backend conventions

- Builder pattern used throughout tests (`builder_test.go`, `processor_test.go`,
  `resource_test.go`, `subdomain_test.go`) — no `*_testhelpers.go` file exists in the package.
- `model.go`'s `Model` is immutable: unexported fields, no setters, built only through `Builder`.
- `libs/atlas-constants/item.Id` is reused for the item-id type; no new domain type/alias/constant
  was introduced for it. The gem stat names are the client's own WZ field-name strings (`incPAD`
  etc.), which is a different kind of value than `libs/atlas-constants/stat.Type`'s enum
  (character-stat display names like `"STRENGTH"`) — checked `libs/atlas-constants/stat/*.go` and
  confirmed there is no existing constant set this change should have reused instead.
- File-for-file structure matches `gachapon/`'s 12-file shape plus the two prerequisite files
  (`rest/handler.go`, `seed/groups.go`) already accepted by the controller as necessary, not
  re-litigated here.

## Build/test verification (informational — not a substitute for the controller's `verify.sh` gate)

- `go build ./...` and `go test ./... -count=1` from `services/atlas-maker/atlas.com/maker`:
  all packages pass (`atlas-maker`, `atlas-maker/reagent`).
- `tools/catalog-lint`: `go test ./...` passes; `go run . deploy/seed` exits 0 with the new
  `reagents` subdomain recognized (rule added at `tools/catalog-lint/subdomains.go`).

## Not evaluable

- The client-side "no read site found" claim in derivation §1.7 (gem-effect map is write-only in
  both binaries) is outside this diff's surface — it is a finding from Step 1 (already committed,
  out of scope for a Step 2-7 + routing review) and the derivation doc itself flags it as an
  honest limit rather than a verified behaviour. Not re-investigated, per the controller's
  instruction.
- Whether Tasks 22-23 will correctly refrain from treating `randOption`/`randStat` as additive is
  not evaluable here — this package only carries the constraint in a comment; enforcement is a
  future task's responsibility.

## Non-blocking notes

- None beyond what is already recorded as an accepted external blocker (GMS/83.1-only seed,
  ruled on) or accepted prerequisite (extra files, ruled on).

## Findings already ruled on — checked, not contradicted

1. `/api/reagents` domain-named routing — code matches the ruling; no `/maker` re-prefix found.
2. `gms/83_1`-only seed — confirmed no `72_1` seed was invented; only `deploy/seed/gms/83_1/reagents/`
   exists in this diff.
3. `rest/handler.go`, `seed/groups.go`, 45 seed files, `catalog-lint` rule — all present and
   consistent with being necessary prerequisites; no unrelated scope creep found alongside them.
