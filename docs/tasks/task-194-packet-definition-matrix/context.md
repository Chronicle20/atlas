# Packet Definition Matrix — Implementation Context

Companion to [`plan.md`](./plan.md). Everything here was measured against this
worktree, not recalled. Read this before starting any task; it is the shortest
path to the decisions and traps the plan encodes.

Inputs: [`prd.md`](./prd.md) (amended in Task 1), [`design.md`](./design.md),
`prototype.html`.

---

## 1. What this task builds

Three deliverables in dependency order:

1. **atlas-configurations** — two additive REST fields (`socket.unsupported.{handlers,writers}`, per-entry `fname`) plus a neutral `socket` validator package shared by the template and tenant trees. No entity, migration, provider, processor-flow or Kafka change.
2. **`packet-audit seed-fname`** — a re-runnable subcommand that backfills `fname` into the eleven seed templates by joining `(direction, opcode)` against `docs/packets/registry/<version>.yaml`.
3. **atlas-ui** — a pure React-free socket-domain library, one pivot-grid component driven by it, a drawer, six dialogs, two bulk flows, a `/packet-matrix` route, and the deletion of four stacked-card `useFieldArray` forms.

---

## 2. Decisions of record

Resolved with the user on 2026-08-05, superseding design.md §10.

| # | Decision | Consequence |
|---|---|---|
| 1 | **Validation is strict.** The server blocks on all of FR-11.1–11.5 at 400, not design.md's recommended two-rule subset. | Task 3 has no warn tier. Task 18 must build the validator remediation, or the live gms_95 tenant deadlocks. |
| 2 | **The padded-opcode duplicates are fixed here**, overriding the PRD non-goal. | Task 2 removes four `MiniRoom` entries. This is what makes decision 1 safe for the seed corpus. |
| 3 | **The PRD's measured numbers are amended.** | 219 writer rows, 2,863 corpus entries (2,859 after Task 2). Task 1 edits `prd.md`. |

### The strict-validation consequence — do not drop Task 18

Saves are whole-document. With FR-11.4 blocking, a configuration carrying **any**
empty handler validator cannot be saved *at all* — so a single-definition edit
can never be the fix. The live gms_95 tenant carries 32 empty validators
(`bug_v95_tenant_empty_validators_and_dup_opcode`), which under strict blocking
is a hard deadlock: the first PATCH that would repair one is itself rejected.

`fillMissingValidators` (Task 12) + `FillMissingValidatorsDialog` (Task 18)
repair the whole document in one write, which is the only way out. The banner
appears on a per-object page only when that document actually has an offender.

---

## 3. Corpus facts (measured, not recalled)

Run against `services/atlas-configurations/seed-data/templates/`:

| Fact | Value |
|---|---|
| Templates | 11 |
| Total socket entries | **2,863** before Task 2, **2,859** after |
| Distinct handler names | **141** (matches the PRD) |
| Distinct writer names | **219** (the PRD said 259 — design F2) |
| Largest template | `gms_95_1` at 129 handlers / 215 writers |
| Service values in use | `login` ×416, `channel` ×2449 — nothing else |
| Validator values in use | `LoggedInValidator` ×1089, `NoOpValidator` ×57 |
| Empty validators in seed data | **0** (the 32 are in a live *tenant*, not the seed templates) |
| Malformed opcodes in seed data | **0** |

### Names bound to more than one opcode — all legitimate except four entries

| Kind | Name | Templates |
|---|---|---|
| handler | `ServerListRequestHandle` | 9 (`gms_48_1` … `gms_95_1`) |
| writer | `MiniRoom` | 5 (`gms_83_1`, `gms_84_1`, `gms_87_1`, `gms_95_1`, `jms_185_1`) |
| handler | `NoOpHandler` | 3 (`gms_87_1`, `gms_92_1`, `gms_95_1`) |
| writer | `CharacterEffect` | 2 (`gms_95_1`, `jms_185_1`) |

`NoOpHandler` in `gms_95_1` is at `0x17`, `0x19`, `0x22`, `0x24` — a deliberate
sink for known-but-ignored packets. `ServerListRequestHandle` is at `0x04` and
`0x0B`. These are normal, permanent, and must never be "fixed".

**The four genuine defects** (same numeric opcode, different zero-padding), all
removed in Task 2:

| Template | Kept | Removed |
|---|---|---|
| `template_gms_83_1.json` | `0xA5` | `0x0A5` |
| `template_gms_87_1.json` | `0xB0` | `0x0B0` |
| `template_gms_95_1.json` | `0xB8` | `0x0B8` |
| `template_jms_185_1.json` | `0xA3` | `0x0A3` |

In each case the removed entry is the second of the pair, has no `options` key,
and is otherwise identical (`"services": ["channel"]`). Behaviour-neutral: the
dispatch map is opcode-keyed and last-write-wins, so today the padded entry
wins; after removal the canonical one does, and both mean "no options supplied".

**`template_gms_84_1.json` keeps both its `MiniRoom` bindings** — `0x0A5` (165)
and `0xA8` (168) are genuinely different opcodes. It is also why the opcode
regex must accept three hex digits.

### Registry ambiguity — exactly one

`gms_v61.yaml` clientbound opcode 242 carries two distinct `fname` values:
`STORAGE` → `CTrunkDlg::OnPacket` and `RPS_GAME` → `CRPSGameDlg::OnPacket`. The
tie-break is the lexicographically-first `op` name, so `RPS_GAME` wins.

---

## 4. Key files

### atlas-configurations

| File | Role |
|---|---|
| `templates/socket/rest.go`, `tenants/socket/rest.go` | Gain `Unsupported` + `Normalize` (Task 4) |
| `templates/socket/{handler,writer}/rest.go` + tenant mirrors | Gain `FName`; `Options` gains `omitempty` (Task 4) |
| `socket/validate.go` *(new)* | The shared rules, imported by both trees (Task 3) |
| `templates/socket/adapter.go`, `tenants/socket/adapter.go` *(new)* | `ToValidationInput` — ~20 mechanical lines each (Task 5) |
| `templates/validation_error.go`, `tenants/validation_error.go` | Gain `socketIssues` alongside the existing preset errors (Task 5) |
| `templates/processor.go:83,116,65`, `tenants/processor.go:151,115` | `Create`, `UpdateById`, `Make` — validate + normalize (Task 5) |
| `templates/resource.go:35`, `tenants/resource.go:114` | `Create` gains the `errors.As` branch `Update` already has (Task 5) |
| `seed-data/templates/template_*.json` | 11 files; Tasks 2 and 7 |

### tools/packet-audit

| File | Role |
|---|---|
| `cmd/root.go` | `Run` is a flat list of `if args[0] == "…"` clauses; add one |
| `cmd/seed_fname.go` *(new)* | The generator (Task 6) |
| `internal/opregistry/opregistry.go:64` | `LoadVersion(path) (*VersionFile, error)`; `Entry{Op, Direction, Opcode, FName}` |

### atlas-ui

| File | Role |
|---|---|
| `src/types/models/socket.ts` *(new)* | The shared wire shape (Task 8) |
| `src/lib/socket/{model,opcode,normalize,matrix,options,ancestry,mutate}.ts` *(new)* | The pure domain library (Tasks 8–12) |
| `src/lib/hooks/api/useSocketObjects.ts` *(new)* | Sparse reads + the single write path (Task 13) |
| `src/components/features/socket/**` *(new)* | Grid, toolbar, drawer, dialogs, bulk flows (Tasks 14–18) |
| `src/pages/PacketMatrixPage.tsx` *(new)* | The matrix route (Task 19) |
| `src/components/app-sidebar-items.ts:64-70` | Deployment children — triple-sync |
| `src/lib/deployment-routes.ts:6-11` | `DEPLOYMENT_ROUTE_PREFIXES` — triple-sync |
| `src/components/__tests__/app-sidebar.test.tsx:47-54` | The assertion that forces the sync |
| `src/pages/{templates,tenants}-{handlers,writers}-form.tsx` | **Deleted** in Task 19 |
| `src/components/unknown-options.tsx` | `OptionsField<T>({form, path})` — **survives**, embedded by the dialogs |

---

## 5. Traps

### 5.1 Identity: never splice by name alone, never by index

`NoOpHandler` is four array entries. A dialog that removes "the entry named
`NoOpHandler`" deletes three live routes on save. Every mutation is keyed by
`(name, normalized opcode)` and throws `MutationError` if that does not resolve
to exactly one entry.

Index is equally unsafe: `templatesService.getById` re-sorts both arrays by
opcode on read (`templates.service.ts:52-63`), so a fetched entry's index does
not match its stored index.

### 5.2 The sparse-cache write hazard

`tenants.service.ts:303` builds its PATCH body as
`{ ...tenant.attributes, ...updatedAttributes }`. A sparsely-fetched document
reaching that spread silently erases `characters`, `worlds` and `cashShop`. The
same shape exists on the template path.

**Rule:** sparse reads live under `socketKeys.matrix()` / `socketKeys.tenantMatrix()`
and are never a mutation input. Every mutation re-fetches the full document by
id inside its `mutationFn`.

### 5.3 Two live bugs in the template save path

Both are fixed in Task 13; do not build on top of them.

- `templatesService.update` calls `api.put`, but `atlas-configurations` registers **no** PUT route — `templates/resource.go:29` binds `/{templateId}` to `http.MethodPatch` only, and `grep -rn 'MethodPut' services/atlas-configurations/` returns nothing. A PUT there is a 405.
- The four stacked-card forms pass `updates: { socket: {...} }`, a *partial* attribute object, which `throwIfInvalid` → `validateTemplate` rejects for missing `region`/`majorVersion`/`usesPin`/`characters` before the transport is even reached.

The tenant path is fine — it uses `api.patch` and spreads the full attributes.

### 5.4 The sidebar triple-sync

`/packet-matrix` needs `app-sidebar-items.ts`, `deployment-routes.ts` **and**
`app-sidebar.test.tsx` changed together. The test asserts every Deployment child
satisfies `isDeploymentRoute`, so changing any two leaves the suite red.

### 5.5 Options: positional vs keyed comparison

Ordered lists (`types`) compare **positionally** — the array index *is* the wire
value. `gms_95_1` `CharacterMovement` carries `UNKNOWN` at six separate indices,
so a name-keyed comparison would match six unrelated slots. Maps (`operations`,
`failedReasonCodes`, `codes`) compare by key.

An **empty** `types` array classifies as a *list with zero entries*, not as
"supplies no options" — the distinction is what finally makes the empty
`gms_92_1`/`gms_95_1` `CharacterMovement` and `gms_87_1`/`gms_95_1`/`jms_185_1`
`PetMovement` tables visible. This task makes that state visible; it does **not**
judge it (carried forward from PRD §9, still out of scope).

### 5.6 The options marker fires on absence only

FR-3.2's marker means "this object supplies no options where a sibling does". It
never fires on structural divergence between versions — that is the expected
state (`gms_12_1` supports 9 movement types, `jms_185_1` supports 33) and
marking it on every row is noise. It also never fires on an *undefined* cell:
there is no definition there for options to be missing from.

### 5.7 `omitempty` on a Go map

`Options map[string]interface{} \`json:"options,omitempty"\`` drops a **nil** map
but keeps a **non-nil empty** one. That is exactly the absent-vs-`{}`
distinction wanted: an entry that supplied none stops round-tripping to
`"options": null` (which would make the first save of any template a 200-line
diff), while an explicit `{}` survives as `{}` so the seed files stay stable.

### 5.8 Seeded `fname` reaches fresh databases only

`seeder.go`'s `importTemplate` skips any template whose
`(region, majorVersion, minorVersion)` row already exists. Backfilled seed files
therefore reach new clusters and CI, **never an existing deployment**. Existing
installs acquire `fname` through the UI, a baseline republish, or not at all.
Acceptable — `fname` is informational (FR-10.4) — but say it in the commit
message so nobody reports it as a bug.

### 5.9 The order guard is non-strict

`tools/template-opcode-order-guard.sh` compares with `code < prev`, so equal
adjacent opcodes pass despite the docstring claiming "strictly ascending". That
is why the padded duplicates survived, and why Task 2 adds a separate
`template-duplicate-binding-guard.sh` rather than tightening the existing one
(tightening it would also reject the legitimate `AuthPermanentBan` /
`AuthLoginFailed` shared opcode).

### 5.10 `go.mod` is touched → the container build is mandatory

Task 3 adds a `require` for `libs/atlas-opcodes` (the `replace` already exists at
`go.mod:96`). Per `CLAUDE.md`, `docker buildx bake atlas-configurations` is then
mandatory — `go build` against the workspace `go.work` will not catch a missing
`COPY libs/...` line in the shared Dockerfile. `libs/atlas-opcodes` is already
COPYed at `Dockerfile:38` and `:68`, so no Dockerfile edit is expected.

---

## 6. Architectural rules

- **`src/lib/socket/` imports no React, no React Query and no service module.** If a task needs one there, the abstraction has leaked — put the React part in a component. This is what makes the entire protocol semantics unit-testable without rendering, and it is where the test weight sits.
- **The two backend trees stay parallel.** `templates/socket` and `tenants/socket` are duplicated by design and match the existing service boundary. The *rules* are shared via the neutral `socket` package; the *models* are not. An aliasing indirection would be more code than the ~20 lines it saves.
- **Socket validation is not routed through `WithValidator`.** That seam exists because *preset* validation needs an atlas-data client. Socket rules are pure and must never be skippable, so they run unconditionally inside the processor.
- **A control whose handler prop is absent is not rendered.** That is how the four per-object pages drop the mode switch, column picker and baseline selector (FR-7.3), and how a tenant with no ancestor drops every ancestry affordance (FR-8.2) — absent, never disabled.
- **No new UI dependency.** Not `@tanstack/react-table` (dynamic columns, cross-column predicates — its column model earns nothing) and not `@tanstack/react-virtual` (fights the frozen first column and breaks the deep-link scroll-to-row path). Virtualization is the escape hatch if measurement shows jank, not an up-front choice.

---

## 7. Verification gates

From the worktree root, per `CLAUDE.md`:

| Gate | When |
|---|---|
| `go build ./... && go vet ./... && go test -race ./...` in `services/atlas-configurations/atlas.com/configurations` | Every backend task |
| same in `tools/packet-audit` | Tasks 6–7 |
| `docker buildx bake atlas-configurations` | **Mandatory** — `go.mod` touched in Task 3 |
| `tools/template-opcode-order-guard.sh` | Tasks 2, 7 |
| `tools/template-duplicate-binding-guard.sh` *(new in Task 2)* | Tasks 2, 7 |
| `tools/lint.sh --check` | Final sweep — needs `nvm use 22` or it false-fails |
| `npm test && npm run build` in `services/atlas-ui` | Every frontend task; `build` type-checks tests |

Not implicated: `service-registration-guard.sh` (no new service; none of
`services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, `db-bootstrap.sh`
changed), `redis-key-guard.sh`, `goroutine-guard.sh`.

**Code review runs before the PR**, not after — `superpowers:requesting-code-review`
dispatches `plan-adherence-reviewer`, `backend-guidelines-reviewer` and
`frontend-guidelines-reviewer`. Pin them to a cheaper model per the project's
model preference.

---

## 8. Explicitly out of scope

- Optimistic concurrency or a definition-scoped PATCH endpoint. Saves remain whole-document, last-write-wins. Re-fetching inside `mutationFn` narrows the window but does not close it.
- Whether the empty `types` arrays on `gms_92_1`/`gms_95_1` `CharacterMovement` and `gms_87_1`/`gms_95_1`/`jms_185_1` `PetMovement` are intentional. This task makes the state visible for the first time; it does not change it. Worth a separate investigation — an empty movement table is the same shape as the confirmed v79 monster-movement defect where moves never decoded.
- Sourcing `Unsupported` from the packet audit matrix (`docs/packets/audits/status.json`, which already tracks `n-a` per op × version). The field ships empty and hand-populated.
- Generating audit registries for `gms_12_1` and `gms_92_1`; they resolve `fname` by adjacent-version impl-name match instead.
- Any codec, packet, opcode-table or runtime decode change. `atlas-channel` is untouched: it decodes socket documents into a model declaring only `Handlers` and `Writers`, and no `DisallowUnknownFields` call exists anywhere in `services/` or `libs/`, so both new fields are ignored by construction.
