# Packet Definition Matrix — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-05
---

## 1. Overview

Socket handler and writer definitions are the routing contract between the game
client and Atlas. Each Template and Tenant carries two arrays under
`socket` — `handlers` (serverbound: opcode → validator → handler impl) and
`writers` (clientbound: opcode → writer impl) — plus a free-form `options`
object per entry that supplies wire tables the codecs read at runtime
(movement type arrays, operation-mode maps, failure-reason codes).

Today these are edited through a stacked-card `useFieldArray` form, one card per
definition, with no search, no sort, no filter and no cross-object comparison.
The GMS v95.1 template renders **215 writer cards** on a single scrolling page — the largest of eleven templates carrying 2,863 socket entries in total.
There is no way to answer "does v87 route this yet?" inside the product at all;
that question is answered by an external spreadsheet that is maintained by hand
and drifts from the live configuration.

This task replaces the spreadsheet with a **Packet Matrix** page under
Deployment, and refactors the four per-object definition pages into the same
dense grid. Definitions gain an explicit `Unsupported` state so "audited, this
version does not have this packet" stops being indistinguishable from "nobody
has looked yet", and gain an `fname` field carrying the client-side function
name so the matrix reproduces the spreadsheet's reference column.

## 2. Goals

Primary goals:

- Replace the external protocol spreadsheet with a first-class Packet Matrix
  page comparing socket definitions across Templates.
- Make the four per-object definition pages usable at 141 handler / 219 writer
  scale: dense rows, search, sort, filter, dialog-based mutation.
- Introduce an explicit `Unsupported` state, stored separately from the
  definition arrays, so absence and audited-absence are distinguishable.
- Surface `options` divergence — specifically **absence** of options where
  sibling Templates supply them — which is currently invisible.
- Let a Tenant be compared against its inferred ancestor Template, and let
  missing definitions be copied from that ancestor in bulk.

Non-goals:

- Optimistic concurrency or a definition-scoped PATCH endpoint. Saves remain
  whole-document PUTs and last-write-wins. Explicitly accepted for v1.
- Fixing the duplicate-`0x00` opcode data smell in existing tenant configs.
- Generating audit registries for GMS v12.1 and v92.1.
- Changing any codec, packet, or runtime decode behaviour. This task is
  configuration management only.
- Editing `options` values through a structured editor beyond what the existing
  `OptionsField` component already provides.

## 3. User Stories

- As a protocol maintainer, I want to see one definition's opcode across every
  Template at once, so I can tell which versions still need it without opening
  eleven pages.
- As a protocol maintainer, I want to search 219 distinct writers by name or opcode, so I
  stop scrolling a 215-card form.
- As a protocol maintainer, I want to mark a definition Unsupported for a
  version I have audited, so the next person does not re-investigate it.
- As a protocol maintainer, I want to see when a Template supplies no `options`
  for a definition that its neighbours do supply, so I catch omissions like an
  empty movement `types` table before they reach a client.
- As a deployment operator, I want to see how a Tenant differs from its ancestor
  Template, so I know what has been hand-modified in the live runtime.
- As a deployment operator, I want to copy every definition my Tenant is missing
  from its ancestor Template in one reviewed operation, so I stop doing it one
  dialog at a time.
- As a protocol maintainer, I want to jump from a matrix cell straight to that
  Template's definition page with the row already selected, so I can edit
  without re-finding the row.

## 4. Functional Requirements

### 4.1 Domain model

A **Definition** is one packet implementation. Its identity is its
implementation name (`LoginHandle`, `AuthLoginFailed`) — **not** its opcode,
which is version-specific, and not the client op name.

A Definition is either a **Handler** or a **Writer**. The two collections are
never mixed in a single view.

For any Template or Tenant, a Definition is in exactly one state:

| State | Meaning | Storage |
|---|---|---|
| Defined | A complete definition exists | entry in `socket.handlers` / `socket.writers` |
| Unsupported | Audited and confirmed absent for this Region/Version | name in `socket.unsupported.handlers` / `.writers` |
| Undefined | Neither of the above | inferred from absence |

FR-1.1 A name MUST NOT be both Defined and Unsupported in the same object.
FR-1.2 Adding a Definition MUST remove any Unsupported marker for that name.
FR-1.3 Removing an Unsupported marker MUST return the Definition to Undefined.
FR-1.4 Deleting a Definition MUST leave it Undefined unless the user also elects
to mark it Unsupported in the same dialog.

### 4.2 Packet Matrix page

FR-2.1 A new route `/packet-matrix` is added, reachable from the Deployment
sidebar group positioned between **Tenants** and **Services**.
FR-2.2 The page is **Templates-only**. Tenants never appear as columns.
FR-2.3 Handlers and Writers are separate modes, selected by a segmented control.
FR-2.4 Rows are Definitions; columns are selected Templates in version order.
FR-2.5 The row set is the **union** of Defined and Unsupported names across the
selected Templates, for the active mode. A Definition that is neither Defined
nor Unsupported anywhere cannot appear; this is accepted.
FR-2.6 A cell renders:
  - Defined → the opcode, on the Defined background
  - Unsupported → an explicit `n/a` marker, visually distinct
  - Undefined → an empty cell
FR-2.7 Cells are not directly editable.
FR-2.8 The **definition name column is frozen**; it is the only frozen column.
FR-2.9 One selected Template is the **baseline**, which determines row ordering.
It is **not** rendered as a separate column — the baseline Template's own column
is marked in place (distinct header treatment plus a column outline).
FR-2.10 Default baseline is the highest-version selected Template. The user may
select a different baseline.
FR-2.11 Definitions not present in the baseline sort after baseline-defined
entries.
FR-2.12 Users may choose which Templates appear as columns, filtered by Region
and Version.

### 4.3 Options surfacing

`options` is a free-form object. Two structural families occur in practice:

- **Maps** keyed by name, value is the wire number — `operations`,
  `failedReasonCodes`, `codes`.
- **Ordered lists** where the *array index* is the wire value and the name is
  not unique — `types`. GMS v95.1 `CharacterMovement` carries `UNKNOWN` at six
  separate indices.

FR-3.1 There MUST NOT be a per-row options summary column. Structural
divergence between versions is the expected state — GMS v12.1 supports 9
movement types and jms185 supports 33 — and marking it on every row is noise.
FR-3.2 Exactly one options marker appears in the grid: a cell whose object
supplies **no** options for a Definition that at least one other selected object
does supply options for. This indicates an omission rather than version drift.
FR-3.3 Selecting a row opens a detail drawer with an **Options** tab rendering a
nested matrix: rows are option entries (array index for lists, key name for
maps), columns are the same selected objects.
FR-3.4 The nested matrix MUST mark entries that differ from the baseline at that
index/key, and entries absent for that object.
FR-3.5 Ordered lists MUST be compared positionally. Position is the identity;
the name is not.

### 4.4 Sorting, searching, filtering

FR-4.1 Default sort is ascending baseline opcode.
FR-4.2 Sorting is also supported by definition name, and by state.
FR-4.3 Search matches definition name, `fname`, and opcode. Opcode matching MUST
normalise numerically, so `0x2A`, `2A` and `42` all match the same cell.
FR-4.4 Filters: state (Defined / Unsupported / Undefined), has options, options
not supplied, and service. Every filter is evaluated **across the whole row** —
a row survives when ANY visualized object satisfies it, never only the
baseline (amended by §14 item 7).
FR-4.5 On per-object Tenant pages, an additional filter for difference-from-
ancestor state.

### 4.5 Row and cell selection

FR-5.1 Selecting a row opens a detail drawer showing, per selected object:
state, opcode and validator (handlers only). Services and options shape are
NOT repeated here — each owns its own tab (amended by §14 item 11).
FR-5.2 Clicking a **cell** — including an Undefined or Unsupported one, which
is where a definition gets created or audited-absent (amended by §14 item 8) —
rather than the definition name scopes the drawer's actions to that object. The
scoped object is indicated visually and every action's ACCESSIBLE NAME names it
(`Edit in v87.1…`); the visible label is the short verb, since the drawer header
already states the scope once (amended by §14 item 9).
FR-5.3 Clicking the definition name leaves the scope on the baseline.
FR-5.4 Actions targeting an object where the Definition is Undefined MUST be
disabled where they have no meaning (e.g. Open, Edit, Undefine). Define, Copy
and Mark unsupported stay enabled — an Undefined scope is precisely when they
apply.

### 4.6 Dialogs

All mutations occur through dialogs. Each dialog reads the current
configuration, splices the affected definition, and writes the whole document
back.

FR-6.1 **Add Definition** — required: name, opcode, services, plus validator for
handlers. Optional: options. Validates required fields, opcode format,
duplicate name within the target collection, and conflicting Unsupported state.
FR-6.2 **Edit Definition** — edits opcode, validator, services, options. The
name is the identity and is not editable; renaming is unsupported.
FR-6.3 **Delete Definition** — distinguishes *Remove definition* (→ Undefined)
from *Remove and mark Unsupported*.
FR-6.4 **Mark Unsupported** — identifies the Definition and the target
Region/Version, requires confirmation, and states explicitly that an existing
definition will be removed.
FR-6.5 **Copy Definition** — choose source object, choose source Definition,
load its values, edit before applying, confirm target. The result is
independent of the source.
FR-6.6 **Reset to Ancestor** (Tenant only) — shows current Tenant values,
ancestor Template values, and the fields that will change.

### 4.7 Per-object pages

FR-7.1 The four existing routes — `/templates/:id/handlers`,
`/templates/:id/writers`, `/tenants/:id/handlers`, `/tenants/:id/writers` —
render the same grid component as the matrix, locked to one object.
FR-7.2 Tenant pages additionally render the inferred ancestor Template as a
second, read-only column.
FR-7.3 The mode switch, column picker and baseline selector are absent on these
pages.
FR-7.4 The stacked-card `useFieldArray` forms
(`tenants-handlers-form.tsx`, `tenants-writers-form.tsx`,
`templates-handlers-form.tsx`, `templates-writers-form.tsx`) are removed.

### 4.8 Tenant ancestry

FR-8.1 A Tenant's ancestor Template is inferred by exact match on Region, Major
Version and Minor Version. No Template id is stored.
FR-8.2 If no exact Template exists, the Tenant page drops to a **single column**
and ancestry-dependent features are absent. No manual Template substitution is
offered. This is not expected to occur.
FR-8.3 With an ancestor present, each Definition is classified as: Same as
Template, Modified, Tenant-only, Missing from Tenant, or Unsupported in Tenant.
FR-8.4 Comparison covers opcode, validator, services, and options. Maps compare
by key/value; lists compare positionally; opcodes normalise numerically before
comparison.

### 4.9 Bulk copy from ancestor

FR-9.1 A Tenant page offers **Copy missing from ancestor**, which identifies
Definitions that are Defined in the ancestor Template and Undefined in the
Tenant.
FR-9.2 The flow shows candidates, allows per-definition selection, allows
per-definition review and adjustment of opcode and configuration, then applies.
FR-9.3 Before applying, the review step MUST show per definition: name, source
opcode, target opcode, validator, services, option differences, and current
target state.
FR-9.4 The operation MUST NOT overwrite any existing Tenant Definition. Only
Undefined entries are affected.
FR-9.5 Unsupported Definitions are excluded unless explicitly selected and
changed by the user.
FR-9.6 The whole selection applies as a single configuration write.

### 4.10 FName metadata

FR-10.1 Handler and Writer definitions gain an optional `fname` string carrying
the client-side function name (`CLogin::SendCheckPasswordPacket`).
FR-10.2 `fname` is stored in the configuration JSON, not derived at runtime.
FR-10.3 The matrix renders `fname` as a toggleable column and includes it in
search.
FR-10.4 `fname` is informational. It MUST NOT participate in comparison,
validation, or ancestry classification.
FR-10.5 The client op name (`LOGIN_PASSWORD`) is deliberately **not** carried.

### 4.11 Validation

FR-11.1 Validate duplicate definition names within the same collection.
FR-11.2 Validate opcode format.
FR-11.3 Validate conflicting Defined / Unsupported state.
FR-11.4 Validate missing handler validators.
FR-11.5 Validate service values.
FR-11.6 Duplicate opcodes MUST NOT fail validation. Several Writers legitimately
share an opcode — GMS v12.1 has `AuthPermanentBan` and `AuthLoginFailed` both at
`0x01`.

### 4.12 Deep linking

FR-12.1 Direct navigation is supported to a Template or Tenant handler/writer
page, optionally to a specific Definition within it, and to the Matrix filtered
to a specific Definition.
FR-12.2 Opening a Definition from the matrix navigates to the scoped object's
page with the grid filtered to that Definition and its row selected.

## 5. API Surface

No new endpoints. Existing endpoints gain fields.

### 5.1 Modified resources

`GET|POST|PATCH /api/configurations/templates[/{id}]`
`GET|POST|PATCH /api/configurations/tenants[/{id}]`
`GET /api/configurations/tenants/{id}/configurations/{resource}`

The `socket` attribute gains an `unsupported` object, and each handler/writer
entry gains an optional `fname`:

```json
{
  "socket": {
    "handlers": [
      {
        "opCode": "0x01",
        "validator": "NoOpValidator",
        "handler": "LoginHandle",
        "fname": "CLogin::SendCheckPasswordPacket",
        "options": {},
        "services": []
      }
    ],
    "writers": [],
    "unsupported": {
      "handlers": ["GuestLoginHandle"],
      "writers": []
    }
  }
}
```

### 5.2 Compatibility

- `unsupported` is optional. Absent → both lists empty.
- `fname` is optional and omitempty.
- Existing consumers of `socket.handlers` / `socket.writers` are unaffected;
  no field is removed or renamed.

### 5.3 Sparse fieldsets

The matrix requests only what it needs:

```
GET /api/configurations/templates?fields[templates]=socket,region,majorVersion,minorVersion
```

`templates/resource.go` and `tenants/resource.go` already parse
`jsonapi.ParseQueryFields` and thread it into `MarshalResponse` on both list and
detail handlers. This requires verification that the filter reaches the
attribute marshaller, not implementation.

### 5.4 Validation errors

Validation failures return the existing JSON:API error shape via the services'
`validation_error.go`. New error cases: conflicting Defined/Unsupported state,
and duplicate definition name within a collection.

## 6. Data Model

### 6.1 No migration required

Both `templates.Entity` and `tenants.Entity` store the whole configuration as
`Data json.RawMessage` (`gorm:"type:json;not null"`). Adding `unsupported` and
`fname` is a **REST-model change only**. No `AutoMigrate` change, no column, no
backfill migration.

### 6.2 Go REST model changes

`services/atlas-configurations/atlas.com/configurations/`:

- `templates/socket/rest.go`, `tenants/socket/rest.go` — add
  `Unsupported UnsupportedRestModel \`json:"unsupported"\``
- new `Unsupported` model with `Handlers []string` and `Writers []string`
- `templates/socket/handler/rest.go`, `tenants/socket/handler/rest.go` — add
  `FName string \`json:"fname,omitempty"\``
- `templates/socket/writer/rest.go`, `tenants/socket/writer/rest.go` — same

### 6.3 Seed data

- `unsupported` ships **empty** on all seed templates. It is populated as audits
  land, not backfilled now.
- `fname` **is** backfilled across all eleven seed templates by a one-time
  generator (see §7.3).

### 6.4 TypeScript types

`services/atlas-ui/src/types/models/template.ts` — the `socket` shape gains
`unsupported` and per-entry `fname`. The Tenant configuration type mirrors it.

## 7. Service Impact

### 7.1 atlas-configurations

REST-model additions only (§6.2). Validation for the new conflicting-state rule.
Seed templates updated with `fname` and empty `unsupported`. No entity,
migration, processor, provider, or Kafka change.

### 7.2 atlas-ui

The bulk of the work.

- New route `/packet-matrix` and sidebar entry in `app-sidebar-items.ts` under
  the Deployment group, between Tenants and Services. The sidebar sync test
  (`src/components/__tests__/app-sidebar.test.tsx`) asserts Deployment children
  agree with `isDeploymentRoute`; both must be updated together.
- A shared definition-grid component driving both the matrix and all four
  per-object pages.
- Definition dialogs (add / edit / delete / mark-unsupported / copy / reset).
- The bulk copy-missing-from-ancestor flow.
- Ancestry inference and comparison logic, including structural options
  comparison with positional list semantics.
- Removal of the four stacked-card forms (§7.1 FR-7.4).
- Query hooks and service-layer updates for the new fields and sparse fieldsets.

### 7.3 One-time fname generator

A tool that writes `fname` into the eleven seed templates by joining each
definition's opcode against `docs/packets/registry/<version>.yaml` on
`(direction, opcode)` — handlers against `serverbound`, writers against
`clientbound`.

| Source | Resolved |
|---|---|
| Direct opcode join, 9 versions with a registry | 2,674 / 2,685 |
| GMS v92.1 via adjacent-version impl-name match (v87.1 then v95.1) | 112 / 112 |
| GMS v12.1 via adjacent-version impl-name match (v48.1 then v61.1) | 63 / 66 |
| **Total** | **2,849 / 2,863** |

The three GMS v12.1 misses are `WorldSelectHandle` (`0x03`), `ServerLoad` (`0x02`)
and `CashShopCashQueryResult` (`0xBD`). The 14 unresolved entries ship without
`fname`.

This generator runs once to produce the seed data. It is not a build step and
not a runtime dependency; `fname` is thereafter maintained as ordinary
configuration.

### 7.4 Unaffected

atlas-channel and every other consumer of socket configuration are unaffected.
No codec, opcode table, or decode path changes.

## 8. Non-Functional Requirements

**Payload.** Eleven Templates at full attributes carries character templates,
presets and equipment lists the matrix never reads. The matrix MUST request
sparse fieldsets (§5.3).

**Rendering.** The writers matrix is up to 219 rows × 11 columns ≈ 2,409 cells
with a sticky header and a frozen first column. It must remain responsive while
scrolling and filtering.

**Multi-tenancy.** The Packet Matrix is a Deployment-scoped page: it applies to
all tenants and does not follow the tenant switcher, consistent with the other
Deployment routes. Tenant configuration reads and writes continue to carry the
four tenant headers via the existing API client.

**Concurrency.** Saves are whole-document PUTs with last-write-wins. Two
sessions editing different definitions on the same object will clobber each
other. Accepted for v1 and explicitly out of scope.

**Accessibility.** Grid rows and cells are keyboard-reachable, selection state
is exposed to assistive technology, and state is never conveyed by colour alone
— Unsupported carries an `n/a` marker and options-absence carries a glyph.

**Testing.** Vitest with React Testing Library, per the atlas-ui conventions.
Go table-driven tests for the new validation rules. `npm run build` must pass,
since it type-checks tests.

## 9. Open Questions

- Whether the empty `types` arrays observed on GMS v92.1 / v95.1
  `CharacterMovement` and v87.1 / v95.1 / jms185 `PetMovement` are intentional.
  Sibling versions populate 23–33 entries. This has **not** been verified
  against the codecs and is **not** in scope here — this task makes the state
  visible, it does not change it. Worth a separate investigation, as an empty
  movement `types` table is the same shape as the confirmed v79 monster-movement
  defect where moves never decoded.
- Whether `Unsupported` should eventually be sourced from the packet audit
  matrix (`docs/packets/audits/status.json`, which already tracks `n-a` state
  per op × version) rather than hand-maintained. Out of scope for v1; the field
  ships empty and hand-populated.

## 9a. Decisions of record (2026-08-05)

Resolved with the user after design.md measured the corpus:

1. **Validation is strict.** The server enforces all of FR-11.1–11.5 at 400.
   FR-11.1 is enforced as duplicate `(name, normalized opcode)` — the literal
   "duplicate definition name" reading would reject the legitimate multi-binding
   that exists in every template.
2. **The padded-opcode duplicates are fixed here**, overriding §2's non-goal.
   Four writer entries (`MiniRoom` at `0x0A5`/`0x0B0`/`0x0B8`/`0x0A3`) are
   removed. This is what makes decision 1 safe for the seed corpus.
3. **§4.1's "identity is its implementation name" is superseded** by
   design.md §5.1: the row is `(kind, name)`, a cell holds a set of bindings,
   and every mutation is keyed by `(name, normalized opcode)`.

## 10. Acceptance Criteria

Backend:

- [ ] `socket.unsupported.{handlers,writers}` round-trips through both the
      template and tenant configuration APIs.
- [ ] `fname` round-trips on handler and writer entries, omitted when empty.
- [ ] Configuration with no `unsupported` key loads with both lists empty.
- [ ] A payload with a name in both `handlers` and `unsupported.handlers` is
      rejected with a JSON:API validation error.
- [ ] Duplicate opcodes within `writers` are accepted.
- [ ] Duplicate definition names within a collection are rejected.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean, and
      `docker buildx bake atlas-configurations` succeeds.

Seed data:

- [ ] All eleven seed templates carry `fname` on the count reported by
      `packet-audit seed-fname`, and that report is committed to this task
      folder as `fname-coverage.txt`.
- [ ] All eleven seed templates omit the `unsupported` key entirely (no
      audits have landed yet, so there is nothing to record); `Normalize()`
      supplies the empty `{handlers: [], writers: []}` shape on read, per
      §5.2/§6.3.
- [ ] `tools/template-opcode-order-guard.sh` still passes.

Matrix:

- [ ] `/packet-matrix` renders from the Deployment sidebar between Tenants and
      Services, and the sidebar sync test passes.
- [ ] Handlers mode shows 141 rows and Writers mode 219 rows across the eleven
      seed templates.
- [ ] The definition name is the only frozen column; the baseline Template is
      marked in place and has no duplicate column.
- [ ] Changing the baseline reorders rows; non-baseline definitions sort last.
- [ ] Searching `0x2A`, `2A` and `42` all match the same cell.
- [ ] A cell with no options where a sibling Template has options is marked;
      a cell that merely differs structurally is not.
- [ ] The drawer's Options tab renders a nested per-entry matrix, positional for
      `types`, keyed for `operations`.
- [ ] Clicking a cell scopes the drawer's actions to that Template and the
      action labels name it.
- [ ] Actions are disabled where the Definition is Undefined for the scope.

Per-object pages:

- [ ] All four routes render the shared grid; no stacked-card form remains.
- [ ] A Tenant page shows its inferred ancestor as a second column.
- [ ] A Tenant with no matching Template renders a single column with ancestry
      features absent.
- [ ] Modified, Tenant-only, Missing and Unsupported states each render
      distinctly and are filterable.

Dialogs:

- [ ] Add, Edit, Delete, Mark Unsupported, Copy and Reset to Ancestor each
      perform their documented mutation and survive a page reload.
- [ ] Adding a Definition clears any Unsupported marker for that name.
- [ ] Removing an Unsupported marker returns the Definition to Undefined.
- [ ] Delete offers both "remove" and "remove and mark Unsupported".

Bulk copy:

- [ ] Copy missing from ancestor lists only Definitions Defined in the ancestor
      and Undefined in the Tenant.
- [ ] The review step shows name, source opcode, target opcode, validator,
      services, option differences and current target state.
- [ ] Applying never modifies an already-Defined Tenant Definition.
- [ ] The selection applies as a single configuration write.

Frontend gates:

- [ ] `npm test` and `npm run build` clean in atlas-ui.
- [ ] `tools/lint.sh --check` clean from the repo root.

---

## 14. Review feedback — round 1 (2026-08-07)

Twelve items raised against the first implementation, after comparing it with
`prototype.html`. Each is recorded here because several AMEND a requirement
above rather than merely fixing a defect against it; the amendments are
back-annotated inline.

| # | Feedback | Resolution |
|---|---|---|
| 1 | The grid carried a `max-height: 70vh`, floating the frame's bottom border away from a short grid | The grid fills the frame (`min-h-0 flex-1`); the frame is the scroll boundary |
| 2 | No legend below the matrix | `GridLegend` renders below the grid, inside the frame, from the same colour table the cells use |
| 3 | The baseline dropdown did not show the selected value | Trigger reads `Baseline: GMS v95.1` |
| 4 | Filters should be aggregating chips with suggestions on a second header line | Toolbar is two lines: line 1 = what you look at, line 2 = one chip per filter (dashed `+ Service` when inactive, `Service: Login ×` when active) |
| 5 | Sort was visually indistinguishable from the filters | Sort moved into its own bordered group at the end of line 1 |
| 6 | The detail drawer opened on the right, not the bottom | `Sheet side="bottom"`, max 70vh, with the tabs and the scope's binding list side by side |
| 7 | The state filter only inspected the baseline | **Amends FR-4.4**: every filter aggregates across the row. `hasOptions: false` reads as "some DEFINED cell supplies none" |
| 8 | Undefined/Unsupported cells were not clickable | **Amends FR-5.2**: every cell is a real hit area (Undefined renders a `·` placeholder); the drawer's Define / Mark unsupported are how you fill one in |
| 9 | Drawer buttons were verbose and ambiguous | **Amends FR-5.2**: visible label is the short verb, accessible name keeps the full `verb in <scope> (opcode)` phrase, and every button carries a behaviour tooltip. Add relabels to "Define here" on an Undefined scope |
| 10 | "Delete" did not match the grid's vocabulary | Removing the ONLY binding is now "Undefine" (button, dialog title, radio, toast); removing one of several stays "Remove binding" and says what survives |
| 11 | The Fields tab repeated state, services and options | **Amends FR-5.1**: a card per object carrying opcode + validator, tinted by state (state word kept as the card's accessible label); services and options are their own tabs |
| 12 | Contiguous opcode ranges hid the slots where no definition exists | `withOpcodeGaps` interleaves a blank row for every opcode inside the BASELINE's [min, max] that the BASELINE does not bind — a sibling column binding that number for its own definition does not fill the baseline's hole (that definition is a non-baseline row and sorts into the tail). Always on when sorted by opcode and no filter is active |

Two decisions of record from this round:

- **Item 6** is a bottom **Sheet** (still modal), not the prototype's docked
  panel — the focus-trap and close semantics of the existing drawer are kept.
- **Item 12** scans ONE opcode namespace, not one per service. A login handler
  and a channel handler at the same number both suppress that gap, which
  under-reports login-range holes. Accepted deliberately for a quieter grid.
