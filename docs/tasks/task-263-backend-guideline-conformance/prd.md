# Backend Guideline Conformance Sweep — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-26
Source: [Chronicle20/atlas#1498](https://github.com/Chronicle20/atlas/issues/1498)
---

## 1. Overview

The `backend-guidelines-reviewer` audit run during #1497 (task-261, the Go 1.27.0
migration) surfaced seven structural deviations from the backend-dev-guidelines
checklist. None were introduced by that PR — each sits on a line the migration's
gofumpt sweep whitespace-touched, so they are pre-existing and became visible
incidentally. They were recorded non-blocking for the migration and split out into
#1498 under the audit protocol's "prevalence is not compliance" rule.

This task does **not** stop at those seven. A repo-wide sweep (`sweep.sh` in this
folder) shows the same three checklist rows are violated far more broadly than the
issue implies: 361 `rest.go` files lack `Transform`, 100 `Builder` structs are
declared outside `builder.go`, and 59 builder constructors are named
`NewModelBuilder` against 132 named `NewBuilder`. Fixing only the seven named files
would leave the checklist rows still failing everywhere else and would guarantee the
next audit re-raises them. The decision taken at spec time is to converge the whole
repository onto the checklist in this task.

The work is mechanical and behavior-preserving by construction: it adds conversion
functions, moves type declarations between files in the same package, and renames
constructors. No wire format, no packet, no Kafka topic, no HTTP route, and no
database schema changes. The risk is not correctness but volume — roughly 520 change
sites across ~40 services — so the plan must be codemod-first, with hand work
reserved for the cases a codemod cannot safely reach.

## 2. Goals

Primary goals:

- Close checklist row **DOM-04** (`Transform(Model) (RestModel, error)` defined in
  `rest.go`) for every package that can host it.
- Close checklist rows **DOM-01 / FILE-05** (builder declared in `builder.go`, with a
  `NewBuilder()` constructor) repo-wide.
- Leave a re-runnable inventory (`sweep.sh`) so a future audit can confirm the rows
  stay closed rather than re-deriving them by hand.
- Produce a documented, evidence-backed exemption list for the packages where a
  checklist row is structurally impossible to satisfy, so those are recorded as
  decided rather than as outstanding gaps.

Non-goals:

- Any behavior change. If a diff hunk changes what a service does at runtime, it is
  out of scope and must be reverted.
- Rewriting `atlas-data`'s storage model. Its packages persist `RestModel` directly
  and declare no domain `Model` (§4.1); introducing one is a separate design task.
- Changing the checklist itself. DOM-01/DOM-04/FILE-05 are taken as given.
- Frontend (`atlas-ui`) or `libs/atlas-packet` codec structure.
- Any other checklist row the task-261 audit did not flag (DOM-05 `TransformSlice`,
  EXT-*, SEC-*, and the rest).

## 3. User Stories

- As a backend developer, I want every `rest.go` to expose `Transform` in the same
  place with the same signature, so that I can move between services without
  re-learning each package's conversion convention.
- As a reviewer running `backend-guidelines-reviewer`, I want DOM-01, DOM-04 and
  FILE-05 to pass repo-wide, so that a fired finding means a real new deviation
  rather than pre-existing noise I have to triage every time.
- As a developer adding a new package, I want `NewBuilder()` to be the unambiguous
  convention, so that I am not choosing between two live naming patterns.
- As a future auditor, I want the structurally-exempt packages listed with evidence,
  so that I do not re-open a decision that was already made.

## 4. Functional Requirements

### 4.1 DOM-04 — `Transform` in `rest.go`

**Current state (from `sweep.sh`, run at `eaa5ce6f7`):**

| Population | Count | Inventory file |
|---|---|---|
| `rest.go` files with no `func Transform(` | 361 | (union of the two below) |
| — package serves a resource (`resource.go` present) | 35 | `inventory-dom04-outbound.txt` |
| — package is an inbound-only read client | 326 | `inventory-dom04-inbound.txt` |
| `rest.go` files that already define `Transform` | 165 | — |

Cross-cut by whether a domain `Model` type exists to transform *from*:

| Population | Count | Inventory file |
|---|---|---|
| Package declares `type Model` | 185 | `inventory-dom04-has-model.txt` |
| Package declares no `type Model` | 176 | `inventory-dom04-no-model.txt` |

**FR-1.** Every package in `inventory-dom04-has-model.txt` MUST define, in its own
`rest.go`:

```go
func Transform(m Model) (RestModel, error)
```

The function maps each exported `RestModel` field from the corresponding `Model`
accessor. It returns a non-nil error only where a field conversion can genuinely
fail; otherwise it returns `nil`. It MUST be the exact inverse of the package's
existing `Extract`/`extract` where one exists.

**FR-2.** Where a package's domain type is not literally named `Model` (e.g.
`channel/monsterbook` declares `Collection` and `Card`), the function MUST follow the
package's existing `Extract` naming — `Transform` for the primary type, `Transform<X>`
for each additional one, mirroring `ExtractCard` → `TransformCard`.

**FR-3.** Where a package declares multiple `RestModel` types over one `Model` (e.g.
the five-compartment `data/tradeability` readers), one `Transform` per RestModel is
required, named for the wire type: `TransformEquipment`, `TransformConsumable`,
`TransformSetup`, `TransformEtc`, `TransformCash`. A shared generic helper mirroring
the existing `extract[R]` is permitted and preferred.

**FR-4.** Every new `Transform` MUST have a round-trip test asserting
`Extract(Transform(m)) == m` for a fully-populated `Model`. This test is what makes
the function live code rather than an unreachable stub, and it is the only mechanism
that will catch a field added to `Model` but forgotten in `Transform`.

**FR-5.** Packages in `inventory-dom04-no-model.txt` are **structurally exempt** and
MUST NOT be given a `Transform`. There is no `Model` type to accept as a parameter.
Writing one would require inventing a domain model, which is a behavior-bearing
design change and explicitly a non-goal (§2). Evidence for the largest cluster
(`atlas-data`, 21 of the 35 outbound files): `services/atlas-data/atlas.com/data/monster/`
has no `model.go`, its `reader.go` produces `RestModel` directly, and
`resource.go:163` marshals `server.MarshalResponse[RestModel]` — the domain model
and the wire model are the same type by design.

**FR-6.** The exemption list MUST be written to
`docs/tasks/task-263-backend-guideline-conformance/exemptions.md`, one entry per
package, each citing the `file:line` evidence that no `Model` exists. This file is
the artifact a future audit reads to classify those rows `n-a` rather than `FAIL`.

**FR-7.** The four packages named in #1498 MUST be included; the sweep confirms all
four fall in the `has-model` population, so none is exempt:

- `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go`
- `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go`
- `services/atlas-channel/atlas.com/channel/monsterbook/rest.go`
- `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest.go`

### 4.2 DOM-01 / FILE-05 — builder placement

**Current state:** 100 `type <X>Builder struct` declarations sit outside
`builder.go` (`inventory-file05-builders.txt`). Concentration:

| File | Builders |
|---|---|
| `services/atlas-npc-conversations/atlas.com/npc/conversation/model.go` | 20 |
| `services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go` | 7 |
| `services/atlas-pets/atlas.com/pets/data/pet/model.go` | 2 |
| `services/atlas-npc-conversations/.../conversation/quest/model.go` | 2 |
| 69 further files | 1 each |

**FR-8.** Each builder struct, its constructor, its `Clone*`, its fluent setters, and
its `Build()` MUST be moved into a `builder.go` in the same package, created if
absent. The move is a pure relocation: no signature, field, or body change.

**FR-9.** Where a package already has a `builder.go` holding some builders and others
live elsewhere, all of them MUST end up in `builder.go`.

**FR-10.** `entity_builder.go` files (`atlas-quest` ×2, `atlas-tenants` ×2) hold
*entity* builders, not domain-model builders. These MUST be resolved explicitly: the
implementer determines from the code whether each is a domain builder subject to
FILE-05, and either moves it into `builder.go` or records it in `exemptions.md` with
evidence. It MUST NOT be left unexamined.

**FR-11.** `services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go` (1038
lines, 7 builders) MUST be split so that the seven reference-data builders move to
`builder.go` and the reference-data models remain in `reference_data.go`.

### 4.3 DOM-01 — builder constructor naming

**Current state:**

| Constructor name | Count (non-test) |
|---|---|
| `NewBuilder` | 132 |
| `NewModelBuilder` | 59 |
| other `New<X>Builder` | 69 |

**FR-12.** All 59 `NewModelBuilder` constructors
(`inventory-dom01-newmodelbuilder.txt`) MUST be renamed to `NewBuilder`, with every
call site updated. Test call sites are included.

**FR-13.** The associated `ModelBuilder` / `modelBuilder` type MUST be renamed to
`Builder` / `builder` in the same change, so the type and its constructor agree. Where
the type is unexported (`*modelBuilder`), it stays unexported (`*builder`).

**FR-14.** `CloneModelBuilder` MUST be renamed `CloneBuilder` where it accompanies a
renamed constructor.

**FR-15.** The 69 `other New<X>Builder` constructors
(`inventory-dom01-other.txt`) name a type *other* than the package's `Model` — e.g.
`NewProposalBuilder`, `NewCeremonyBuilder`, `NewDSNBuilder`,
`NewEquipableReferenceDataBuilder`. Where a package has exactly one builder and it
builds the package `Model`, it MUST be renamed to `NewBuilder`. Where a package has
several builders over distinct types, `New<Type>Builder` is the only name that can
disambiguate them; those MUST be recorded in `exemptions.md` with the sibling
builders listed as evidence. The implementer classifies each of the 69 — none may be
skipped silently.

**FR-16.** After the rename, `sweep.sh` MUST report zero entries in
`inventory-dom01-newmodelbuilder.txt`.

### 4.4 Behavior preservation

**FR-17.** No change in this task may alter runtime behavior. Concretely: no
`RestModel` JSON tag, no `GetName()` return value, no route, no Kafka topic, no
struct field type, and no `Build()` validation rule may change.

**FR-18.** `git diff` for the branch MUST contain no edit to a `.sql` migration, a
`docker-compose*.yml`, a `.tpl`/template file, or anything under
`libs/atlas-packet/`.

## 5. API Surface

None. This task introduces no endpoint, modifies no request or response shape, and
changes no error case. Every `RestModel` keeps its existing JSON tags, `GetName()`
value, and `GetID()` format, so no serialized payload changes on the wire.

`Transform` functions added under FR-1 are new *exported Go symbols*, not new API
surface. They are consumed by their round-trip tests (FR-4) and are available to
future callers that need outbound serialization.

## 6. Data Model

No new entity, no field change, no relationship change, no constraint change, no
migration. No `tenant_id` scoping question arises: the task adds no persisted data.

Type renames under FR-13 (`ModelBuilder` → `Builder`) are compile-time only and touch
no GORM entity, no column, and no serialized form.

## 7. Service Impact

Change sites by checklist row:

| Row | Sites | Nature |
|---|---|---|
| DOM-04 (FR-1) | ~185 packages | new function + test per package |
| FILE-05 (FR-8) | 100 declarations | intra-package file moves |
| DOM-01 (FR-12) | 59 constructors + call sites | rename |
| DOM-01 (FR-15) | 69 constructors, triaged | rename or exempt |

Services with the heaviest DOM-04 load (`inventory-dom04-inbound.txt`, by service):

| Service | Files |
|---|---|
| atlas-channel | 68 |
| atlas-configurations | 24 |
| atlas-saga-orchestrator | 17 |
| atlas-login | 14 |
| atlas-messages | 13 |
| atlas-consumables | 11 |
| atlas-monster-death | 9 |
| atlas-character-factory | 9 |
| atlas-cashshop | 9 |
| atlas-world | 8 |
| atlas-query-aggregator | 8 |
| atlas-npc-conversations | 8 |

Roughly 40 services are touched in total. `libs/atlas-database`,
`libs/atlas-constants`, `libs/atlas-script-core`, and `libs/atlas-packet/model` each
appear in the FILE-05 inventory and are in scope for FR-8 except where FR-18 excludes
them — `libs/atlas-packet` is excluded, so
`libs/atlas-packet/model/skill_usage_info.go:143` is out of scope and MUST be recorded
in `exemptions.md` on that basis.

No cross-service seam changes, so the CLAUDE.md "trace the event into its consumers"
rule has nothing to trace here — but the code-review gate still applies (§10).

## 8. Non-Functional Requirements

- **Performance:** no runtime path is added or removed. `Transform` functions are not
  called by any existing production path.
- **Security:** no token, auth, or redirect handling is touched. SEC-* rows are not
  implicated.
- **Observability:** no logging, tracing, or metric changes.
- **Multi-tenancy:** unaffected. FR-17 forbids adding a `tenantId`/`traceId` field to
  any `RestModel`, which would violate DOM-31.
- **Reviewability:** the diff is large by construction. The plan MUST partition it so
  that no single commit mixes two checklist rows, and so that each commit is
  independently buildable. A reviewer must be able to read one commit and check one
  rule.
- **Codemod-first:** FR-12/FR-13/FR-14 (the 59 renames) and FR-8 (the 100 moves) are
  mechanical transformations over a known file list. Per `docs/codemod-vs-agents.md`,
  these MUST be done by a scripted transformation, not by fanning out implementer
  agents at the same transformation. FR-1 (per-package `Transform` + test) is the
  part that genuinely requires per-package judgment.

## 9. Open Questions

1. **FR-15 classification volume.** The 69 `other New<X>Builder` constructors are
   triaged per-package by the implementer. If that triage finds substantially more
   than expected must be renamed (cascading into many call sites), the design phase
   should re-check whether FR-15 stays in this task or splits out.
2. **`Transform` for multi-type packages.** FR-2 fixes the naming rule, but a package
   like `npc/conversation` with 20+ builder-backed types may need a judgment call on
   which types warrant a `Transform` at all. The design phase should settle whether
   FR-1 applies to every `RestModel` in such a package or only to the ones the
   package's `resource.go`/`requests.go` actually serialize.
3. **Exemption authority.** `exemptions.md` records decisions that a future audit will
   treat as binding. Whether that file is the right durable home, or whether the
   exemptions belong in the checklist resource itself
   (`.claude/skills/backend-dev-guidelines/resources/audit-checklist.md`), is
   unresolved. This PRD assumes the task folder; changing it does not change the work.
4. **Merge-conflict exposure.** A ~520-site branch touching 40 services will conflict
   with any concurrent feature branch. Sequencing against the in-flight worktrees
   (task-240, 241, 246, 250, 251, 254, 256, 259, 262 are open) is a scheduling
   question for the plan phase.

## 10. Acceptance Criteria

- [ ] `bash docs/tasks/task-263-backend-guideline-conformance/sweep.sh` reports
      `inventory-dom01-newmodelbuilder.txt` with **0** entries.
- [ ] `sweep.sh` reports `inventory-file05-builders.txt` with **0** entries, except
      those listed in `exemptions.md`.
- [ ] `sweep.sh`'s DOM-04 inventories contain **only** packages listed in
      `exemptions.md` — i.e. every `has-model` package now defines `Transform`.
- [ ] `exemptions.md` exists, and every package it lists carries `file:line` evidence
      for why the row cannot be satisfied. No entry says "out of scope" without that
      evidence.
- [ ] Every `Transform` added under FR-1 has a round-trip test (FR-4) that fails if a
      `Model` field is added without updating `Transform`.
- [ ] `tools/verify.sh` (flagless — not `--quick`, not `--no-docker`) exits 0.
- [ ] `backend-guidelines-reviewer` reports DOM-01, DOM-04 and FILE-05 as PASS or N/A
      for every service it audits, with no FAIL on those three rows.
- [ ] `git diff main...HEAD` touches no `.sql`, no `docker-compose*.yml`, no template
      file, and nothing under `libs/atlas-packet/` (FR-18).
- [ ] `git diff main...HEAD` contains no change to a JSON tag, a `GetName()` return
      value, a route registration, a Kafka topic, or a `Build()` validation rule
      (FR-17), confirmed by an explicit reviewer pass over the diff.
- [ ] Code review completed before the PR is opened, per CLAUDE.md.
