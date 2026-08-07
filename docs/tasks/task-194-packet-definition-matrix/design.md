# Packet Definition Matrix — Design

Version: v1
Status: Draft
Created: 2026-08-05
Input: [`prd.md`](./prd.md) (approved), [`prototype.html`](./prototype.html)
---

## 1. Summary

Three deliverables, in dependency order:

1. **atlas-configurations** gains two additive REST fields — `socket.unsupported.{handlers,writers}` and a per-entry `fname` — plus a socket validator shared by the template and tenant trees. No entity, migration, provider, processor-flow or Kafka change.
2. **A re-runnable `packet-audit seed-fname` subcommand** backfills `fname` into the eleven seed templates by joining `(direction, opcode)` against `docs/packets/registry/<version>.yaml`.
3. **atlas-ui** gains a pure, React-free socket-domain library; a single pivot-grid component driven by it; a drawer, six dialogs and a bulk-copy flow on top; a new `/packet-matrix` route; and the deletion of the four stacked-card `useFieldArray` forms.

The architectural centre of gravity is §5 — the domain model. Everything the grid, the drawer, the dialogs and the ancestry comparison do is a pure function over a normalized `SocketObject`. The React layer renders and dispatches; it holds no protocol semantics.

**§2 records seven findings, measured against the repository, that change what the PRD asks for.** One of them (F1) invalidates the PRD's stated identity model and must be resolved before implementation. They are summarized as explicit PRD deviations in §12.

## 2. Findings that change the PRD

Every number below was measured against the current worktree, not recalled.

### F1 — The definition name is **not** unique within a collection (blocking)

PRD §4.1 states "Its identity is its implementation name." The seed corpus contradicts this in every version:

| Template | Collection | Name | Bindings |
|---|---|---|---|
| gms_95_1 | handlers | `NoOpHandler` | `0x17`, `0x19`, `0x22`, `0x24` |
| gms_95_1 | handlers | `ServerListRequestHandle` | `0x04`, `0x0B` |
| gms_95_1 | writers | `CharacterEffect` | `0xE0`, `0xE9` |
| gms_95_1 | writers | `MiniRoom` | `0xB8`, `0x0B8` |
| gms_83_1 | handlers | `ServerListRequestHandle` | `0x04`, `0x0B` |
| jms_185_1 | writers | `MiniRoom` | `0xA3`, `0x0A3` |

`ServerListRequestHandle` is duplicated in 9 of 11 templates; `NoOpHandler` in 3; `MiniRoom` in 5; `CharacterEffect` in 2.

These are two different phenomena and must be handled differently:

- **Legitimate multi-binding.** `NoOpHandler` at four opcodes is a deliberate sink for known-but-ignored packets. `ServerListRequestHandle` at `0x04` and `0x0B`, `CharacterEffect` at `0xE0` and `0xE9` — one implementation, several wire opcodes. This is normal and permanent.
- **Data defect.** `MiniRoom` at `0xB8` **and** `0x0B8` — the same numeric opcode (184) written twice with different leading-zero padding. This is the duplicate-opcode smell the PRD lists as a non-goal.

Consequences, both of which are data-loss bugs if unaddressed:

- A cell that renders "the opcode" (FR-2.6) cannot render four.
- A dialog that splices "the definition named X" out of the array would collapse `NoOpHandler`'s four routes into one and **silently delete three live routes** on save.

Resolved in §5.1: a row is keyed by name, a cell holds a **set of bindings**, and every mutation is keyed by `(name, normalized opcode)` — never by name alone. This is the single most important decision in this document.

### F2 — The writers row count is 219, not 259

Union of distinct implementation names across all eleven seed templates: **141 handlers** (matches the PRD) and **219 writers** (the PRD says 259 in §2 and in the acceptance criteria). The corpus totals 1,146 handler and 1,717 writer entries; the largest single template is gms_95_1 at 129 handlers / 215 writers.

### F3 — The fname coverage table is measurably wrong; the real coverage is better

PRD §7.3 claims 2,169 / 2,179. The corpus is 2,863 entries, not 2,179. Measured join coverage:

| Source | Resolved |
|---|---|
| Direct `(direction, opcode)` join, 9 versions with a registry | 2,674 / 2,685 |
| gms_92_1 via implementation-name match against v87 + v95 | 112 / 112 |
| gms_12_1 via implementation-name match against v48 + v61 | 63 / 66 |
| **Total** | **2,849 / 2,863** |

The three gms_12_1 misses are `WorldSelectHandle` (`0x03`), `ServerLoad` (`0x02`) and `CashShopCashQueryResult` (`0xBD`). Two opcodes across the nine registry versions carry more than one distinct `fname`; §7 specifies the tie-break.

The acceptance criterion becomes "the generator's reported coverage matches the run recorded in this task folder", not a hard-coded absolute.

### F4 — Opcode format must accept 1–4 hex digits

`template_jms_185_1.json` contains `"opCode": "0x9"`. A `^0x[0-9A-Fa-f]{2}$` regex would reject existing valid data. The accepted format is `^0[xX][0-9A-Fa-f]{1,4}$`.

### F5 — Sparse fieldsets are verified working

`libs/atlas-rest/server/response.go:23` and `libs/atlas-rest/server/paginated_response.go:33` both call `jsonapi.FilterSparseFields(d, queryParams)` after marshalling, on the single and paginated paths respectively. `paginate.DefaultPageSize` is 50 (`libs/atlas-rest/server/paginate/params.go:18`), so eleven templates fit one page. PRD §5.3's verification item is closed: **no backend work required.**

### F6 — Seeded `fname` reaches fresh databases only

`seeder.go` `importTemplate` skips any template whose `(region, majorVersion, minorVersion)` row already exists. Backfilled seed files therefore reach new clusters and CI, never an existing deployment. Existing installs acquire `fname` through the UI, a baseline republish, or not at all. This is acceptable — `fname` is informational (FR-10.4) — but must be stated so nobody reports it as a bug.

### F7 — `options` round-trips from absent to `null`

Seed entries omit `options` entirely when unset. `handler.RestModel.Options` is `map[string]interface{}` with no `omitempty`, so any document that round-trips through a PATCH gains `"options": null` on every entry. Harmless semantically (both decode to a nil map, and both mean "not supplied" per FR-3.2), but it turns the first save of any template into a 200-line diff. §4.1 adds `omitempty`.

### F8 — Additive fields are safe for every consumer

No `DisallowUnknownFields` call exists anywhere in `services/` or `libs/`. `atlas-channel` decodes the socket document into `configuration/tenant/socket/rest.go`, which declares only `Handlers` and `Writers`; `unsupported` and `fname` are ignored by construction. No consumer change, no `libs/atlas-opcodes` change.

## 3. Architecture overview

```
atlas-configurations (Go)                  atlas-ui (TypeScript)
─────────────────────────                  ───────────────────────────────────
templates/socket/rest.go  ─┐               src/lib/socket/          (pure, no React)
tenants/socket/rest.go    ─┼─ additive     ├── model.ts       types + states
  handler/rest.go  +fname  │   fields      ├── normalize.ts   Template|Tenant → SocketObject
  writer/rest.go   +fname  │               ├── opcode.ts      parse / format / compare
                           │               ├── matrix.ts      objects → rows × cells
socket/ (new, shared)  ────┘               ├── options.ts     classify + nested matrix
  Validate(Input) []Issue                  ├── ancestry.ts    infer + classify
    ├─ adapters in templates/socket        └── mutate.ts      pure splice functions
    └─ adapters in tenants/socket                    │
                                           src/lib/hooks/api/useSocketObjects.ts
tools/packet-audit                                   │
  cmd/seed_fname.go  (new subcommand)      src/components/features/socket/
    registry ⋈ seed → fname                ├── PacketGrid.tsx      + Row, Cell
                                           ├── DefinitionDrawer.tsx
                                           ├── OptionsMatrix.tsx
                                           ├── dialogs/*.tsx       (6)
                                           └── CopyFromAncestorFlow.tsx
                                                     │
                                           src/pages/PacketMatrixPage.tsx
                                           src/pages/Te{mplates,nants}{Handlers,Writers}Page.tsx
```

The dependency arrow runs one way: components import from `lib/socket`, never the reverse. `lib/socket` imports no React, no React Query and no service module — it is a function of its arguments, which makes the entire protocol semantics unit-testable without rendering and is where the test weight sits (§8).

## 4. Backend design — atlas-configurations

### 4.1 REST model changes

Mirrored across `templates/socket/` and `tenants/socket/` (the two trees are already parallel; keeping them so is cheaper than an aliasing indirection and matches the existing service boundary):

```go
// {templates,tenants}/socket/rest.go
type RestModel struct {
    Handlers    []handler.RestModel `json:"handlers"`
    Writers     []writer.RestModel  `json:"writers"`
    Unsupported UnsupportedRestModel `json:"unsupported"`
}

type UnsupportedRestModel struct {
    Handlers []string `json:"handlers"`
    Writers  []string `json:"writers"`
}

// {templates,tenants}/socket/handler/rest.go
type RestModel struct {
    OpCode    string                 `json:"opCode"`
    Validator string                 `json:"validator"`
    Handler   string                 `json:"handler"`
    FName     string                 `json:"fname,omitempty"`
    Options   map[string]interface{} `json:"options,omitempty"`   // F7
    Services  []string               `json:"services,omitempty"`
}
```

`writer/rest.go` takes the same treatment with `Writer` in place of `Validator`/`Handler`.

### 4.2 The normalization invariant

`unsupported` is a value type, so an absent key decodes to a struct with two nil slices, and a nil slice marshals as `null` rather than `[]`. Both PRD acceptance criteria ("loads with both lists empty", "carries an empty `unsupported` object") are satisfied by one function:

```go
// {templates,tenants}/socket/rest.go
func Normalize(rm RestModel) RestModel   // nil slices → empty slices; entries left untouched
```

Called in exactly two places per tree: `Make` (every read path funnels through it) and the top of `Create`/`UpdateById` (every write path). The invariant to test: **for any input document, the marshalled output always contains `"unsupported": {"handlers": [...], "writers": [...]}` with non-nil arrays.**

### 4.3 Validation — what blocks, and why most rules do not

The governing precedent is `bug_legacy_tenant_preset_blocks_config_patch`: a blocking validation rule added for good reasons made every legacy tenant's configuration un-PATCHable, including the PATCH that would have fixed it. F1 shows the socket corpus is in exactly that condition today.

**Principle: the server blocks only on rules that no existing document can violate. Every other rule is a client-side dialog constraint plus a grid-visible data-issue marker.**

| Rule | PRD | Server | Rationale |
|---|---|---|---|
| Name in both a collection and its `unsupported` list | FR-11.3 | **400** | `unsupported` is a new field; no stored document can violate it. |
| Opcode format `^0[xX][0-9A-Fa-f]{1,4}$` | FR-11.2 | **400** | Verified against all 2,863 seed entries (F4). A malformed opcode already breaks routing. |
| Duplicate `(name, normalized opcode)` in one collection | FR-11.1 | warn | Violated today by `MiniRoom` in 5 templates (F1). Fixing that data is a PRD non-goal. |
| Empty handler `validator` | FR-11.4 | warn | Seed data is clean, but the live v95 tenant carries 32 empty validators (`bug_v95_tenant_empty_validators_and_dup_opcode`). Blocking would strand it. |
| Unknown service name | FR-11.5 | warn | Same exposure; the dialog offers a picker sourced from the services resource instead. |
| Duplicate opcode across different names | FR-11.6 | accepted | Explicitly legal. Never reported. |

Warnings are computed by the same shared validator and returned to the UI **only** via its own client-side evaluation of the fetched document — no new endpoint, no response envelope change. The Go validator's warn-level issues exist so the rules live in one place and are exercised by Go tests; the server discards them.

If the user prefers strict enforcement, the alternative is grandfathering-by-diff — load the stored document in `UpdateById`, and reject only violations the incoming write *introduces*. That is implementable (the processor already has DB access) but adds a read to every write and cannot help `Create`, which is how the UI clones a tenant from a template. Recommended against for v1; recorded in §10 as the user's call.

### 4.4 Shared validator and error plumbing

A new neutral package `atlas-configurations/socket` (top level, imported by both trees):

```go
package socket

type Severity string   // "error" | "warning"

type Issue struct {
    Severity Severity
    Path     string   // "socket.handlers[12].opCode"
    Message  string
}

type Binding struct{ Name, OpCode, Validator string; Services []string }

type Input struct {
    Handlers, Writers                []Binding
    UnsupportedHandlers, UnsupportedWriters []string
}

func Validate(in Input) []Issue
```

`templates/socket` and `tenants/socket` each contribute a ~15-line adapter building `Input` from their `RestModel`. The rules themselves — and their table-driven tests — exist once.

Two existing pieces need generalizing, both small:

- `templates/validation_error.go` and `tenants/validation_error.go` currently hold `[]preset.ValidationError`. They become `[]Issue`-shaped, with the preset validator's errors adapted through a converter (`presets[<id>].<field>` keeps its existing `meta.path`). One error type per tree, one `errors.As` branch, unchanged JSON:API error shape.
- `Create` currently runs no validation at all and its resource handler has no `errors.As` branch. Socket validation must run on both `Create` and `UpdateById`, so `handleCreateConfigurationTemplate` / `handleCreateConfigurationTenant` gain the same branch `handleUpdate*` already has (`templates/resource.go:130-140`).

Socket validation is pure and dependency-free, so it runs unconditionally inside the processor — it is **not** routed through the `WithValidator` injection point, which exists because preset validation needs an atlas-data client. That seam stays as-is.

### 4.5 Explicitly unchanged

`Entity`, `AutoMigrate`, `provider.go`, `administrator.go`, the outbox/backfill path, the tenant configuration projection, `libs/atlas-opcodes`, and every atlas-channel decode path (F8).

## 5. Domain model

### 5.1 Definition identity and bindings

The corrected model, replacing PRD §4.1:

- A **Definition** is identified by `(kind, name)` where `kind ∈ {handler, writer}`. This is the row key and the unit of Unsupported marking, ancestry classification and copy-from-ancestor.
- Within one object, a Definition holds one or more **Bindings**. A Binding is one array entry: `{opCode, opCodeValue, validator?, services, options, fname?, index}`. `opCodeValue` is the parsed integer; `index` is the entry's position in the object's own array.
- **A Binding, not a Definition, is the unit of Add / Edit / Delete.**

Rendering (FR-2.6 refined): a cell shows the lowest binding's opcode, and `+N` when `N` further bindings exist. The drawer lists every binding for the scoped object with its own action row.

Two bindings of one name whose `opCodeValue` are equal (`0xB8` / `0x0B8`) render as a single opcode carrying a duplicate marker, with both underlying entries visible and independently deletable in the drawer.

### 5.2 States

```ts
type DefinitionState = "defined" | "unsupported" | "undefined";
```

`defined` ⇔ at least one binding. The FR-1.1 invariant (never both `defined` and `unsupported`) is enforced by construction in `mutate.ts`: `addBinding` clears the name from the unsupported list; `markUnsupported` removes **every** binding of that name — necessarily, because `unsupported` is name-scoped while bindings are opcode-scoped. The dialog states this in as many words before confirming (FR-6.4 already requires it; F1 makes the plural matter).

### 5.3 Opcode normalization

`opcode.ts` is the only place a hex string is interpreted:

```ts
parseOpcode(raw: string): number | null   // "0x2A" | "2A" | "0X2a" → 42; "42" → 42 only in search
formatOpcode(n: number): string           // canonical "0x2A"
```

FR-4.3's three-way search match (`0x2A`, `2A`, `42`) is a search-input concern: a numeric-looking query is matched against `opCodeValue` under both hex and decimal interpretations, so `42` matches both `0x42` and `0x2A`. Stored values are never rewritten — canonicalization is display-only, because rewriting `0x0B8` to `0xB8` on save is the PRD's out-of-scope data fix.

### 5.4 Options

```ts
type OptionsShape = "absent" | "empty" | "map" | "list";
```

`absent` (key missing or `null`) and `empty` (`{}`) both satisfy FR-3.2's "supplies no options". Classification: a value whose sole key holds a JSON array (`types`) is a `list`; a flat object of name → number (`operations`, `failedReasonCodes`, `codes`) is a `map`. Anything else falls back to `map` over its top-level keys and renders read-only.

The cell marker (FR-3.2) fires when this object's shape is `absent`/`empty` **and** at least one other selected object supplies a non-empty shape for the same Definition. It is the only options signal in the grid (FR-3.1).

The nested matrix (FR-3.3–3.5) is built by `buildOptionsMatrix(defn, objects, baselineKey)`:

- **Lists** compare positionally — row key is the array index, which *is* the wire value. A name that shifts index between versions is a diagonal, not a match.
- **Maps** compare by key. Row key is the option name; the cell is the wire number.
- A cell is `differs` when the baseline has a value at that row key and this object's differs; `missing` when the baseline has one and this object has none; `extra` when this object has one past the baseline's extent.

### 5.5 Ancestry

`inferAncestor(tenant, templates)` matches on exact `(region, majorVersion, minorVersion)` (FR-8.1). Zero matches → `null`, and the Tenant page renders one column with ancestry features absent (FR-8.2) — not a disabled control, not an empty column.

`classifyAgainstAncestor(tenantDefn, templateDefn)` returns `same | modified | tenant-only | missing | unsupported`, comparing the **binding set**: opcodes numerically, validator and services by value, options structurally per §5.4, and `fname` never (FR-10.4). Differing binding *counts* for one name is `modified`.

## 6. Frontend design

### 6.1 Rendering strategy

A plain semantic `<table>` with `position: sticky` on the header row and on the first column, rows wrapped in `React.memo` keyed by definition name.

Worst case is 219 rows × 12 columns ≈ 2,600 cells. That renders in one pass well inside a frame budget, and because the row component is memoized over a precomputed row object, filtering and search re-render only the rows whose membership changed. Sorting, filtering and search all operate on the derived model in a `useMemo`, not on the DOM.

Rejected alternatives:

- **`@tanstack/react-table`** (already a dependency). Its column model is built for static column definitions with per-column accessors; here columns are dynamic objects and every sort/filter predicate is cross-column. It would add a layer that owns none of the semantics.
- **`@tanstack/react-virtual`** (new dependency). Virtualization fights a frozen first column and a sticky header, and it breaks the deep-link "scroll to and select this row" path (FR-12.2) without extra scroll-to-index plumbing. Held as the escape hatch if measurement shows jank; not taken up front.

### 6.2 Module layout

```
src/lib/socket/
  model.ts      SocketObject, Definition, Binding, DefinitionState, OptionsShape
  opcode.ts     parseOpcode, formatOpcode, matchesQuery
  normalize.ts  fromTemplate(), fromTenantConfig() → SocketObject
  matrix.ts     buildRows({objects, kind, baselineKey}) → Row[]; sort/filter/search predicates
  options.ts    classifyOptions, buildOptionsMatrix
  ancestry.ts   inferAncestor, classifyAgainstAncestor
  mutate.ts     addBinding, editBinding, deleteBinding, markUnsupported,
                clearUnsupported, copyBindings, copyMissingFromAncestor
  __tests__/    one spec per module
src/types/models/socket.ts   shared SocketConfig / HandlerEntry / WriterEntry types
```

`src/types/models/socket.ts` resolves a live duplication: the socket shape is declared inline in both `src/types/models/template.ts:84` and `src/services/api/tenants.service.ts:74`. Both must gain `unsupported` and `fname`; extracting the shape once is cheaper than editing it twice and is squarely in scope.

Components:

```
src/components/features/socket/
  PacketGrid.tsx           table shell, sticky/frozen layout, selection, keyboard nav
  PacketGridRow.tsx        memoized row
  PacketGridCell.tsx       state background, opcode/+N, n/a marker, ⌀ glyph
  GridToolbar.tsx          mode switch, search, column picker, baseline picker, filters
  DefinitionDrawer.tsx     per-object detail; Fields / Options / Services tabs; action rail
  OptionsMatrix.tsx        nested per-entry matrix
  CopyFromAncestorFlow.tsx candidate list → per-definition review → single write
  dialogs/AddDefinitionDialog.tsx
  dialogs/EditDefinitionDialog.tsx
  dialogs/DeleteDefinitionDialog.tsx
  dialogs/MarkUnsupportedDialog.tsx
  dialogs/CopyDefinitionDialog.tsx
  dialogs/ResetToAncestorDialog.tsx
```

Pages: new `PacketMatrixPage.tsx`; the four existing page wrappers swap their form import for `<DefinitionGridPage kind={…} scope={…} />`.

### 6.3 Data flow, and the sparse-cache write hazard

The matrix reads eleven templates with sparse fieldsets (F5):

```
GET /api/configurations/templates?fields[templates]=region,majorVersion,minorVersion,socket
```

**Hazard.** `tenants.service.ts:303` builds its PATCH body as `{...tenant.attributes, ...updatedAttributes}`. If that spread ever runs over a sparsely-fetched document, the write silently erases `characters`, `worlds` and `cashShop`. The same shape exists on the template path.

**Rule, enforced by construction:** sparse reads live under their own query key and are never a mutation input.

- `templateKeys.socketMatrix()` / a matching tenant key hold the sparse documents. They feed the grid only.
- Every mutation re-fetches the **full** document by id inside its `mutationFn`, applies the pure `mutate.ts` function to that fresh document, and PATCHes the whole thing. This also narrows the last-write-wins window the PRD accepts.
- Binding resolution against the fresh document is by `(name, opCodeValue)`. If that resolves to a count other than 1, the mutation aborts with an error toast rather than guessing — concurrent edits fail loudly instead of clobbering (F1 makes index-based splicing unsafe).
- `onSuccess` invalidates both the sparse key and the detail key.

### 6.4 Routing, deep links, and the sidebar triple-sync

`/packet-matrix` requires three files to change together, because a test asserts they agree:

1. `src/components/app-sidebar-items.ts` — Deployment children, between Tenants and Services.
2. `src/lib/deployment-routes.ts` — add `/packet-matrix` to `DEPLOYMENT_ROUTE_PREFIXES`, which drives both the inert tenant switcher and the deployment-scope banner.
3. `src/components/__tests__/app-sidebar.test.tsx` — the expected children array at line 51.

Plus the route itself in `App.tsx`, using `lazyWithReload` per the atlas-ui guide.

Deep links are query parameters via `useSearchParams`, so they survive a reload and are copy-pasteable (FR-12.1/12.2):

- `/packet-matrix?mode=writers&baseline=<templateId>&cols=<id,id,…>&def=<name>`
- `/templates/:id/handlers?def=<name>` — grid filtered to that definition, row selected.

"Open in <object>" from a matrix cell navigates to the second form, which is why cell scope must carry the object id.

### 6.5 Accessibility

`role="grid"` with `aria-rowindex`/`aria-colindex`; rows focusable with arrow-key movement and Enter to open the drawer; cells are buttons so cell-scoping is reachable without a mouse. State is never colour-only: Unsupported renders the literal `n/a`, options-absence renders `⌀` with an `aria-label`, and the baseline column carries a header badge in addition to its outline.

### 6.6 Removals

`templates-handlers-form.tsx`, `templates-writers-form.tsx`, `tenants-handlers-form.tsx`, `tenants-writers-form.tsx` are deleted (FR-7.4). `OptionsField` in `src/components/unknown-options.tsx` survives — it is the options editor the Add/Edit dialogs embed (PRD non-goal: no structured options editor beyond what exists).

## 7. The fname generator

A new subcommand in the existing tool: `packet-audit seed-fname [--write] [--registry-dir …] [--template-dir …]`.

Placed there rather than in a throwaway script because `tools/packet-audit/internal/opregistry` already parses these YAML files and is trusted, and because a subcommand is re-runnable when the next version bring-up adds a registry — the PRD's "runs once" is satisfied without making it a one-shot.

Algorithm:

1. For each of the nine versions with a registry: join every seed entry on `(direction, int(opCode, 16))` — `handlers` ⋈ `serverbound`, `writers` ⋈ `clientbound`. Skip registry rows with an empty `fname`.
2. Ambiguity (two rows, same direction+opcode, different `fname` — 2 cases measured): pick the lexicographically first `op` name, deterministically, and log the choice to stderr.
3. gms_92_1 and gms_12_1 have no registry. Resolve by implementation name against the already-resolved adjacent versions — v87 then v95 for v92, v48 then v61 for v12 — first hit wins. Valid because the implementation name is the definition identity within a direction.
4. Entries that resolve to nothing are written without `fname` (the field is `omitempty`).
5. Report per-template and total coverage on stdout; `--write` is required to touch files.

**JSON fidelity.** Re-marshalling through a Go struct risks dropping keys the struct does not model. Two guards, both mandatory:

- The generator first decodes each file into `map[string]json.RawMessage` (top level) and `map[string]json.RawMessage` per socket entry, and **fails loudly** on any key outside the known set (`region`, `majorVersion`, `minorVersion`, `usesPin`, `socket`, `characters`, `worlds`, `cashShop`; and `opCode`, `validator`, `handler`, `writer`, `fname`, `options`, `services`). A surprise key is a stop, not a silent drop.
- It then writes through an ordered struct that carries `characters`, `worlds` and `cashShop` as `json.RawMessage` verbatim, with `json.MarshalIndent(v, "", "  ")` to match the existing two-space files.

Verification after the run, in this order: a semantic deep-compare of every file before/after ignoring added `fname` keys; `tools/template-opcode-order-guard.sh`; a spot-read of one diff hunk.

The generator writes `fname` only. Empty `unsupported` objects are added by the same pass (they are part of the same seed edit) but carry no derived data.

## 8. Testing

**Go (`atlas-configurations`).** Table-driven tests on `socket.Validate` covering each rule and severity, including the F1 corpus as fixtures (a `MiniRoom` duplicate must produce a warning, not an error). Round-trip tests on `Normalize`: absent `unsupported` → empty arrays; `fname` omitted when empty; a name in both `handlers` and `unsupported.handlers` → 400 through the resource layer on both PATCH and POST. Existing `rest_test.go` / `processor_test.go` extend rather than duplicate.

**TypeScript.** The weight is on `src/lib/socket/__tests__/` — pure functions, no rendering:

- `opcode` — the FR-4.3 three-way match, `0x9`, malformed input.
- `matrix` — row union across objects, baseline ordering with non-baseline entries last, multi-binding cells, numerically-equal duplicate bindings.
- `options` — list-vs-map classification, positional list comparison, the FR-3.2 marker firing on absence and *not* on structural divergence.
- `ancestry` — all five classifications, and the no-ancestor path.
- `mutate` — every dialog's splice, with the invariants asserted: adding clears `unsupported`; `markUnsupported` removes all bindings; delete removes exactly one binding of a multi-binding name; copy-missing never touches a defined entry and produces one document.

RTL covers the interaction contract only: cell-scoping relabels actions and disables where Undefined; the drawer's Options tab renders the nested matrix; the sidebar sync test passes with the new entry.

`npm run build` type-checks tests, so it is a gate, not a formality.

## 9. Alternatives considered

| Decision | Chosen | Rejected | Why |
|---|---|---|---|
| Definition identity | `(kind, name)` row + binding set per cell | `(name, opcode)` row | Opcodes differ per version; the row would never span columns — the entire point of the matrix. |
| | | Treat duplicates as invalid data and fix | `NoOpHandler`×4 is intentional; fixing the corpus is a PRD non-goal. |
| Mutation key | `(name, normalized opcode)`, resolved on a fresh fetch | array index | Index shifts under concurrent edits; F1 makes name-only splicing delete live routes. |
| Server validation scope | Block only on rules no stored document can violate | Block on all of FR-11.1–11.5 | Bricks PATCH for the very documents that need fixing (`bug_legacy_tenant_preset_blocks_config_patch`). |
| | | Grandfather by diffing against stored | Adds a read per write, cannot cover `Create`. Available if the user wants strictness (§10). |
| Shared socket validation | One neutral `socket` package + two adapters | Duplicate rules in both trees | ~150 lines of rules and their tests would drift. |
| Grid rendering | Plain sticky `<table>` + memoized rows | `@tanstack/react-table` | Dynamic columns, cross-column predicates — the column model earns nothing. |
| | | `@tanstack/react-virtual` | New dependency; fights frozen column and deep-link scroll. Escape hatch only. |
| Read strategy | Sparse list under a dedicated query key | Reuse the existing detail cache | A sparse document reaching the PATCH spread erases `characters`/`worlds`/`cashShop`. |
| fname generator | `packet-audit` subcommand, re-runnable | One-off script | Registry parser already exists there; next version bring-up re-runs it. |
| fname storage | Stored in configuration JSON (PRD FR-10.2) | Derived at runtime (as `prototype.html` shows) | PRD supersedes the prototype; derivation would need the registry at runtime, which the UI does not have. |

## 10. Open questions for the user

1. **Validation strictness (§4.3).** Recommended: server blocks only the two safe rules; the rest are dialog constraints plus grid markers. The alternative is grandfathering-by-diff. This is the one decision that changes backend scope materially.
2. **The `MiniRoom` / `0x0B8` duplicates (F1).** Recommended: leave them, surface them (PRD non-goal). Fixing them is five entries across five templates and would let duplicate-`(name, opcode)` become a blocking rule — but it is a data change the PRD excluded.
3. **PRD acceptance numbers (F2, F3).** Writers rows become 219, and fname coverage becomes the generator's reported figure against 2,863. Confirm the PRD is amended rather than the design bent to match it.

Carried forward unresolved from the PRD, unchanged and out of scope: whether the empty `types` arrays on gms_92_1 / gms_95_1 `CharacterMovement` and gms_87_1 / gms_95_1 / jms_185_1 `PetMovement` are intentional. This task makes that state visible for the first time; it does not judge it.

## 11. Verification gates

Per `CLAUDE.md`, from the worktree root:

1. `go test -race ./...` and `go vet ./...` in `services/atlas-configurations`.
2. `go build ./...` in `services/atlas-configurations`; `go build ./...` in `tools/packet-audit`.
3. `docker buildx bake atlas-configurations` — only if `go.mod` is touched (no new dependency is planned, so likely not; run it if it is).
4. `tools/template-opcode-order-guard.sh` — mandatory, the seed files change.
5. `tools/lint.sh --check` from the repo root (needs nvm 22 loaded, per `bug_lint_check_false_fails_without_nvm`).
6. `npm test` and `npm run build` in `services/atlas-ui`.
7. Semantic deep-compare of the seed files before/after the fname backfill (§7).

`tools/service-registration-guard.sh`, `redis-key-guard.sh` and `goroutine-guard.sh` are not implicated — no new service, no Redis, no goroutine.

## 12. Deviations from the PRD

| PRD | Change | Source |
|---|---|---|
| §4.1 "identity is its implementation name" | Row identity is `(kind, name)`; the **binding** `(name, opcode)` is the mutation unit. Cells hold binding sets. | F1 |
| §4.2 FR-2.6 "a cell renders the opcode" | Lowest opcode plus `+N`; all bindings in the drawer. | F1 |
| §4.11 FR-11.1/11.4/11.5 | Client-side constraints and grid markers, not 400s. | §4.3 |
| §4.11 FR-11.2 | Opcode regex accepts 1–4 hex digits. | F4 |
| §6.3 / FR-6.3-6.4 | "Remove and mark Unsupported" removes **every** binding of the name. | §5.2 |
| §7.3 coverage table | 2,849 / 2,863, not 2,169 / 2,179. Three named gms_12_1 misses. | F3 |
| §7.3 "runs once… not a build step" | Re-runnable `packet-audit` subcommand. Still not a build step. | §7 |
| §5.3 "requires verification" | Verified working; no backend work. | F5 |
| §10 "Writers mode 259 rows" | 219 rows. | F2 |
| §6.2 (implied) | `options` gains `omitempty`. | F7 |
