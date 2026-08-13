# Template / Tenant Configuration JSON Export — Design

Task: task-199-config-json-export
PRD: [`prd.md`](./prd.md)
Status: Draft
Created: 2026-08-07

---

## 1. Summary

Add an **Export** button to the shared header of `TemplateDetailLayout` and
`TenantDetailLayout`. Clicking it serialises the currently-viewed configuration's
JSON:API `attributes` object to a pretty-printed `.json` file and hands it to the
browser download mechanism. No backend change; no new network path.

The design is four small units, each independently testable:

| Unit | File | Purpose |
|---|---|---|
| `downloadJson` | `src/lib/utils/download-json.ts` | Blob → object URL → synthetic anchor → revoke. Knows nothing about configurations. |
| `configExportFilename` + `toConfigExportPayload` | `src/lib/utils/config-export.ts` | Pure projection + naming. Knows nothing about React or the DOM. |
| `useDetailActionBarState` | `src/components/DetailActionBarContext.tsx` (new export) | Read-only accessor for the shared bar's `dirty` flag. |
| `ConfigExportButton` | `src/components/features/config/ConfigExportButton.tsx` | Wires the above to React Query + toast + shadcn `Button`/`Tooltip`. |

Only `ConfigExportButton` touches React; only `downloadJson` touches the DOM. The
payload projection — the part with real correctness risk — is a pure function
over a plain object and is unit-tested without a renderer.

---

## 2. Verified ground truth

Facts established by reading the code during this design pass. These resolve or
correct several PRD assumptions.

### 2.1 Server key order equals seed-file key order

`services/atlas-configurations/atlas.com/configurations/templates/rest.go:11-22`
and `.../tenants/rest.go:11-22` declare **byte-identical** field sets in the same
order:

```go
Id           string               `json:"-"`
Region       string               `json:"region"`
MajorVersion uint16               `json:"majorVersion"`
MinorVersion uint16               `json:"minorVersion"`
UsesPin      bool                 `json:"usesPin"`
Socket       socket.RestModel     `json:"socket"`
Characters   characters.RestModel `json:"characters"`
NPCs         []npcs.RestModel     `json:"npcs"`
Worlds       []worlds.RestModel   `json:"worlds"`
CashShop     cashshop.RestModel   `json:"cashShop"`
```

The checked-in seeds carry exactly that key order:
`['region','majorVersion','minorVersion','usesPin','socket','characters','npcs','worlds','cashShop']`
(verified across all 11 files under
`services/atlas-configurations/seed-data/templates/`; the five seeds with no NPC
bindings simply omit the `npcs` key on disk).

`JSON.parse` preserves insertion order for non-integer-like keys, and JS object
spread preserves the position of keys that already exist. So an export built by
spreading `attributes` reproduces the seed key order **for free** — no explicit
key ordering step is needed, and none will be added.

### 2.2 Templates DO carry `cashShop` — PRD FR-2.2 is incomplete

PRD FR-2.2 lists the template key set without `cashShop`. That is wrong:
`templates.RestModel.CashShop` has no `omitempty`, `TemplateAttributes.cashShop`
exists in `src/types/models/template.ts:96`, and every one of the 11 seed files
has a top-level `cashShop` key.

**Resolution:** FR-2.7 governs — the export ships whatever `attributes` holds and
enumerates no key list. The projection is key-agnostic, so template and tenant
payloads are structurally identical, which is what FR-2.3 already asserts. No
code change follows from this; it only removes a false expectation from the
acceptance criteria.

### 2.3 Seed files DO end with a trailing newline — PRD open question 3 resolved

`tail -c 4 template_gms_83_1.json | od -c` → `} \n } \n`. FR-2.4's trailing
newline stands.

### 2.4 The tenant service does NOT normalise `null` collections

`sortTemplate` (`templates.service.ts:61-79`) coerces `npcs`/`worlds` from `null`
to `[]`. `sortTenantConfig` (`tenants.service.ts:169-187`) sorts handlers and
writers but performs **no** null-normalisation. PRD FR-2.6 assumes the export can
inherit the normalisation from the service layer; on the tenant path there is
nothing to inherit.

**Resolution:** the export performs its own normalisation in
`toConfigExportPayload`, for both kinds. This is deliberate belt-and-braces on
the template path (where the service already did it) and the only correct source
on the tenant path. It also decouples the export's contract from a service-layer
implementation detail that could be refactored away.

Note that normalising a key that is present-but-`null` preserves its position;
normalising a key that is **absent** appends it at the end. The latter cannot
happen against the real API (no `omitempty` on either Go field), but the
projection is written so that the absent case still produces a valid file — it
just orders `npcs`/`worlds` last. Documented, not defended against.

### 2.5 Both services already sort handlers/writers ascending by numeric opCode

`sortTemplate` and `sortTenantConfig` both apply
`parseInt(opCode, 16)` ascending to `socket.handlers` and `socket.writers`.
FR-2.5 is satisfied by consuming the hook data. The projection re-applies the
same sort anyway — same reasoning as 2.4: the export owns its output contract,
and re-sorting an already-sorted array is free at this size.

### 2.6 There is no app-wide `TooltipProvider`

Every consumer (`MapImageOverlay.tsx:110`, `SkillWidget.tsx:42`,
`InventoryGrid.tsx`) mounts its own `TooltipProvider`. `ConfigExportButton` must
do the same, locally, or the Radix tooltip will not render.

### 2.7 `DetailActionBarContext` is module-private — PRD open question 2

The context object at `DetailActionBarContext.tsx:30-31` is a module-level
`const` with no export. The two existing exported hooks are
`useRegisterDetailActionBar` (write) and `DetailActionBar` (the renderer). There
is no read accessor.

**Resolution:** add one three-line named export,
`useDetailActionBarState(): DetailActionBarConfig | null`. The export button
renders inside `DetailActionBarProvider` (the provider wraps the whole layout
including the header row), so the hook resolves. This is strictly additive and
does not change the existing write path.

### 2.8 React Query defaults: `staleTime: 0`, `refetchOnMount: true`

`src/lib/query-client.ts`. Consequences for FR-5.4 are analysed in §4.3.

### 2.9 jsdom does not implement `URL.createObjectURL`

`src/test/setup.ts` stubs `matchMedia` and `ResizeObserver` only. Tests for
`downloadJson` must `vi.stubGlobal`/`vi.spyOn` `URL.createObjectURL` and
`URL.revokeObjectURL` themselves. This is a test-authoring constraint, not a
change to the shared setup file — adding a global stub would silently mask the
absence in unrelated suites.

---

## 3. Architecture

### 3.1 Data flow

```
TemplateDetailLayout (or TenantDetailLayout)
  └─ DetailActionBarProvider
       ├─ header row
       │    └─ <ConfigExportButton kind="template" id={id} />
       │         ├─ useTemplate(id) / useTenantConfiguration(id)   ← React Query
       │         ├─ useDetailActionBarState()                      ← dirty flag
       │         └─ onClick:
       │              toConfigExportPayload(query.data.attributes)
       │              configExportFilename(kind, meta)
       │              downloadJson(filename, payload)
       │              toast.success(...)
       └─ <Outlet> sub-tab page (also observes the same query key)
```

### 3.2 `downloadJson(filename, payload)`

```ts
export function downloadJson(filename: string, payload: unknown): void {
  const body = `${JSON.stringify(payload, null, 2)}\n`;
  const url = URL.createObjectURL(
    new Blob([body], { type: "application/json" }),
  );
  const anchor = document.createElement("a");
  try {
    anchor.href = url;
    anchor.download = filename;
    document.body.appendChild(anchor);
    anchor.click();
  } finally {
    anchor.remove();
    URL.revokeObjectURL(url);
  }
}
```

Design notes:

- `JSON.stringify` runs **before** `createObjectURL`, so a serialisation throw
  (a cyclic structure, a `BigInt`) cannot leak an object URL at all — there is
  nothing to revoke yet. FR-4.2 is satisfied structurally rather than by
  defensive cleanup.
- The trailing newline (FR-2.4, confirmed §2.3) lives here, not in the
  projection, so the projection stays a pure object→object function and the
  string-shaping concern stays in one place.
- `anchor.remove()` rather than `document.body.removeChild(anchor)` — it is a
  no-op if the append never happened, which keeps the `finally` block total.
- This helper is deliberately generic (`payload: unknown`). It does not import
  anything from the configuration domain, and `NpcShopCard`'s inline copy
  (`NpcShopCard.tsx:263-274`) could be migrated onto it later. **That migration
  is not in this task's scope** — it would change `NpcShopCard`'s output by
  adding a trailing newline, which is a behaviour change to an unrelated feature
  and belongs in its own change.

### 3.3 `config-export.ts` — pure projection and naming

```ts
export type ConfigExportKind = "template" | "tenant";

export function toConfigExportPayload<T extends ExportableConfigAttributes>(
  attributes: T,
): T;

export function configExportFilename(
  kind: ConfigExportKind,
  meta: {
    id: string;
    region?: string;
    majorVersion?: number;
    minorVersion?: number;
  },
): string;
```

`toConfigExportPayload` responsibilities, in order:

1. Spread `attributes` (drops nothing, reorders nothing — §2.1).
2. `npcs: attributes.npcs ?? []`, `worlds: attributes.worlds ?? []` (FR-2.6, §2.4).
3. If `attributes.socket` is present, replace it with a copy whose `handlers`
   and `writers` are sorted ascending by `parseInt(opCode, 16)` (FR-2.5, §2.5).
   If `socket` is absent or falsy, leave it untouched — mirroring both services'
   existing `if (!…socket) return …` guard rather than inventing a new failure
   mode.
4. Return. No `id`, `type`, or `data` key is ever introduced, because none is
   ever read (FR-2.1) — the hooks already hand back `{ id, attributes }` and the
   projection only ever receives `.attributes`.

`configExportFilename` responsibilities:

- Sanitise: `region.toLowerCase().replace(/[^a-z0-9]/g, "_")` (FR-3.3).
- If the sanitised region is empty, or `majorVersion`/`minorVersion` is not a
  finite number, fall back to `<kind>_<id>.json` (FR-3.4). The `id` goes through
  the same sanitiser — a UUID is already `[a-z0-9-]`, and `-` would otherwise
  survive into the name inconsistently with FR-3.3's rule.
- Otherwise `<kind>_<region>_<major>_<minor>.json`, giving
  `template_gms_83_1.json` / `tenant_gms_83_1.json` (FR-3.1, FR-3.2).

Both functions are synchronous, dependency-free, and exported by name. They are
**not** added to the `src/lib/utils/index.ts` barrel: that barrel currently
re-exports only `toast` and `maplestory`, and pulling a DOM-touching module into
it would widen the import graph for every barrel consumer. Call sites import
from the module path directly, matching how `asset-url`, `clock`, and
`reward-pool-chance` are consumed.

### 3.4 `ConfigExportButton`

One shared component parameterised by `kind`, not two per-feature wrappers. See
§5.1 for why.

```tsx
interface ConfigExportButtonProps {
  kind: ConfigExportKind;
  id: string | undefined;
}
```

Hook-ordering constraint: React forbids conditional hooks, so the component
calls **both** query hooks and disables the irrelevant one by passing an empty
id. Both `useTemplate` and `useTenantConfiguration` already guard with
`enabled: !!id`, so the disabled hook issues no request and stays in `pending`
with `fetchStatus: "idle"`. This is an established pattern in this codebase —
`DefinitionGridPage.tsx:62-63` does exactly this for the same two hooks:

```tsx
const templateQuery = useTemplate(kind === "template" ? (id ?? "") : "");
const tenantQuery = useTenantConfiguration(kind === "tenant" ? (id ?? "") : "");
const query = kind === "template" ? templateQuery : tenantQuery;
```

Render:

- shadcn `Button`, `variant="outline"`, `size="sm"`, lucide `Download` with
  `aria-hidden`, visible text `Export` (FR-1.5, accessibility NFR — the visible
  label is the accessible name, so no extra `aria-label` is needed).
- `disabled={!query.data}` — covers loading, error, and the
  no-id/disabled-hook case in one predicate (FR-5.2). Deriving from `data`
  presence rather than from `isLoading || isError` also handles the
  refetch-after-error state without a second condition.
- Wrapped in a local `TooltipProvider` + `Tooltip` (§2.6). Tooltip content is
  `Exports the last saved configuration`, rendered whenever
  `useDetailActionBarState()?.dirty === true` (FR-5.3). When not dirty, no
  tooltip is rendered at all — an always-on tooltip on a self-explanatory button
  is noise. A disabled Radix `TooltipTrigger` does not fire pointer events, but
  the dirty state and the disabled state are mutually exclusive in practice (a
  page cannot be dirty before its query resolves), so no `asChild`/wrapper-span
  workaround is warranted.

Click handler:

```tsx
const onExport = () => {
  const data = query.data;
  if (!data) return;
  try {
    downloadJson(
      configExportFilename(kind, { id: data.id, ...data.attributes }),
      toConfigExportPayload(data.attributes),
    );
    toast.success(kind === "template" ? "Template exported" : "Tenant exported");
  } catch {
    toast.error("Export failed");
  }
};
```

`toast` is imported from `sonner` directly, matching the sibling socket dialogs
(`DefinitionActionDialogs.tsx:2`, `EditDefinitionDialog.tsx:4`) and
`NpcShopCard`. The richer `@/lib/utils/toast` wrapper exists for API-error
transformation, which does not apply to a client-side serialisation failure.

No navigation, route change, or state mutation occurs anywhere in this path
(FR-6.3) — the handler is `void` and touches nothing outside the DOM anchor.

### 3.5 Layout wiring

Both layouts change identically: the header `div.space-y-0.5` becomes a
flex row with the heading block on the left and the button on the right.

```tsx
<div className="flex items-start justify-between gap-4">
  <div className="space-y-0.5">
    <h2 className="text-2xl font-bold tracking-tight">Template Details</h2>
    <p className="text-muted-foreground">{id}</p>
  </div>
  <ConfigExportButton kind="template" id={id} />
</div>
```

Because the button lives in the layout — which persists across sub-route
navigation — FR-1.3 is satisfied structurally for all six template tabs and all
seven tenant tabs, with no per-page wiring and nothing to forget when a tab is
added later. The `DetailActionBar` at the bottom of the layout is untouched
(FR-1.4); the two controls share a provider but not a render tree branch.

---

## 4. Key decisions and their consequences

### 4.1 Payload source: the React Query hook, not a fresh fetch

FR-5.1 requires the export to reflect *persisted server state*, not unsaved form
edits. Reading `query.data` gives exactly that: the detail pages hold their edits
in local form state and only write back through mutations, so the cached resource
is by construction the last-saved document.

The alternative — reading the page's in-progress form state — was rejected
outright: it would produce a file that does not correspond to anything the server
has, which is the opposite of the reconcile-against-seed workflow the feature
exists for.

### 4.2 Observer vs. cache-peek

Three options for getting `data` into the button:

| | Mechanism | Disabled state | Extra request | Verdict |
|---|---|---|---|---|
| **A** | `useTemplate(id)` observer in the layout | Derived from `query.data` — accurate | See §4.3 | **Chosen** |
| B | `queryClient.getQueryData(templateKeys.detail(id))` on click | None available without subscribing; button would be enabled while data is absent | Zero | Rejected — cannot satisfy FR-5.2 |
| C | `queryClient.fetchQuery(...)` on click | Accurate | One per click, always | Rejected — violates FR-5.4 |

B is tempting because it is genuinely zero-cost, but FR-5.2's disabled-while-
loading requirement needs a subscription, and re-deriving one from
`queryClient.getQueryState` in a `useSyncExternalStore` is strictly more code
than just using the hook. A wins on both correctness and size.

### 4.3 FR-5.4, stated honestly

FR-5.4 says export "MUST NOT trigger a new network request if the query cache
already holds the resource." With `staleTime: 0` and `refetchOnMount: true`
(§2.8), a newly-mounting observer *does* trigger a background refetch. The
accurate claim is narrower and still satisfies the requirement's intent:

- **The click handler issues no request.** It is pure cache read + serialisation.
  This is the literal requirement and it holds unconditionally.
- **Mounting the button adds no request beyond what the page already made.** The
  layout and its first child page mount in the same commit and observe the same
  query key, so React Query deduplicates the two into one in-flight fetch.
- **Sub-tab navigation is unaffected.** The layout persists across sub-routes, so
  its observer mounts once per detail-page entry. The refetch on each sub-tab
  change is caused by the *child page's* observer remounting and is pre-existing
  behaviour, unchanged by this feature.

The acceptance criterion "Exporting fires no additional network request when the
resource is already cached" is therefore tested at the **click** level: render
with a pre-populated `QueryClient`, click, assert the fetch spy count is
unchanged.

### 4.4 Normalisation lives in the export, not the services

Adding null-normalisation to `sortTenantConfig` to mirror `sortTemplate` was
considered and rejected. `sortTenantConfig` feeds every tenant-config consumer in
the app (`tenant-context.tsx:177`, the socket grids, `CharactersPanel`,
`ApplyPresetDialog`, …); changing its output shape to fix an export-only concern
is a wide blast radius for a narrow requirement, and it would be an unrequested
behaviour change to code this task otherwise does not touch. The export owns its
own output contract instead (§2.4).

### 4.5 No `id`-stripping logic

FR-2.1 forbids `data`/`type`/`id` in the output. There is nothing to strip: both
service layers already project the JSON:API document down to
`{ id, attributes }`, and the export only ever passes `.attributes` to the
projection. The requirement is met by never picking the key up, which is a
stronger guarantee than deleting it afterwards. The test asserts the absence
anyway, so a future service-layer change that starts nesting `id` inside
`attributes` would be caught.

---

## 5. Alternatives considered

### 5.1 Component factoring — PRD open question 1

| Option | Assessment |
|---|---|
| **One shared `ConfigExportButton` with a `kind` prop** | **Chosen.** The two payload shapes are identical (§2.1, §2.2) and the only differences are which hook to read, the filename prefix, and one toast string — three small branches. Cost: both hooks are called and one is inert, which is already the codebase's own pattern (`DefinitionGridPage`). |
| Two thin per-feature components over a shared `useConfigExport(kind)` hook | Avoids the inert hook call, but the hook still has to call both query hooks for the same React reason, so the inert call does not actually go away — it just moves. Net result is one extra file and one extra indirection for zero benefit. Rejected. |
| Inline the button in each layout | Duplicates the click handler, the disabled predicate, the tooltip, and the toast across two files that already drifted apart once (different sidebar item counts). Rejected. |

### 5.2 Where the JSON is assembled

Considered adding `templatesService.exportOne(id)` / a tenant equivalent
alongside the existing list-level `templatesService.export()`
(`templates.service.ts:449`). Rejected: that method fetches (`getAll`) and
returns a `Blob`, so it is a network+encoding function, whereas this feature must
*not* fetch (FR-5.4) and needs the payload as an object for the projection to be
unit-testable. Reusing it would mean fetching the whole template list to export
one template. The existing method is left untouched, per the PRD's non-goals.

### 5.3 Making the export byte-identical to a seed file

Considered emitting the file through a canonicalising serialiser that forces the
seed key order explicitly, so a `diff` against the seed is guaranteed clean.
Rejected as unnecessary: §2.1 establishes that spread-order already reproduces
seed order because the Go struct order and the seed order are the same. Adding an
explicit key-order table would create a second source of truth that silently goes
stale the next time a field is added to `RestModel` — the exact failure mode this
repo's guard scripts exist to prevent. The acceptance criterion is "shows only
intentional drift", not byte-identity, and spread-order meets it.

---

## 6. Error handling

| Failure | Behaviour |
|---|---|
| Query loading / errored / no id | Button disabled (`!query.data`). No click path reachable. |
| `JSON.stringify` throws | Caught in `onExport` → `toast.error("Export failed")`. No object URL was created (§3.2), so nothing leaks. |
| `createObjectURL` / `click` throws | `finally` in `downloadJson` removes the anchor and revokes the URL; the throw propagates to `onExport`'s catch → `toast.error`. |
| Tenant switched mid-view | `TenantProvider` clears the React Query cache; `query.data` becomes undefined; button returns to disabled until refetch (FR-7.2). No cross-tenant data can be exported because the payload is read synchronously from the current `query.data` inside the handler. |
| Browser blocks the programmatic download | Outside our control and not detectable from JS; the success toast will have fired. Accepted — this is the same contract `NpcShopCard` has shipped with. |

No `errorLogger` call is added: a client-side `JSON.stringify` failure is not an
API error and the shared error pipeline (`@/lib/api/errors`) transforms API
shapes. The toast is the whole of the observability surface, as the PRD's
Observability NFR anticipated.

---

## 7. Testing

### 7.1 `src/lib/utils/__tests__/download-json.test.ts`

Stub `URL.createObjectURL` / `URL.revokeObjectURL` via `vi.stubGlobal` (§2.9) and
spy on `HTMLAnchorElement.prototype.click`.

- Blob body is `JSON.stringify(payload, null, 2)` + `"\n"`, MIME
  `application/json`.
- `anchor.download` equals the filename argument.
- `revokeObjectURL` called exactly once with the created URL.
- No anchor remains in `document.body` afterwards.
- A payload that throws on serialise (cyclic object) propagates, and
  `createObjectURL` was never called.

### 7.2 `src/lib/utils/__tests__/config-export.test.ts`

Pure, no renderer.

- **Envelope:** output has no `id`, `type`, or `data` key.
- **Normalisation:** `npcs: null` / `worlds: null` → `[]`; a present array is
  passed through by value.
- **Sort:** handlers and writers given out of order come back ascending by
  `parseInt(opCode, 16)`, including a mixed-width case (`0x0B8` vs `0xB8`
  compare equal numerically and must not crash or drop an entry).
- **Key order:** `Object.keys(output)` equals the seed order
  `['region','majorVersion','minorVersion','usesPin','socket','characters','npcs','worlds','cashShop']`
  for a fixture built in that order. This is the regression test for §2.1 — if a
  future refactor rebuilds the object key-by-key, this fails.
- **Missing `socket`:** returned untouched, no throw.
- **Filename:** `template_gms_83_1.json`; `tenant_gms_83_1.json`; region
  `"GMS-Beta"` → `gms_beta`; missing region → `template_<id>.json`;
  non-numeric major/minor → fallback.

### 7.3 `src/components/features/config/__tests__/ConfigExportButton.test.tsx`

Rendered with a `QueryClientProvider` over a `QueryClient` seeded via
`setQueryData`, and a `MemoryRouter` if the tree needs one.

- Disabled when the query has no data; enabled once `setQueryData` populates
  `templateKeys.detail(id)`.
- Click with seeded cache: `downloadJson` (module-mocked) called once with the
  expected filename and a payload whose top-level keys are the attribute keys;
  the global `fetch` spy count is unchanged (FR-5.4, per §4.3).
- Success toast asserted via a `sonner` module mock — `Template exported` for
  `kind="template"`, `Tenant exported` for `kind="tenant"`.
- `downloadJson` made to throw → `toast.error` asserted, no unhandled rejection.
- Accessible name is `Export`; the lucide icon carries `aria-hidden`.

### 7.4 Layout smoke coverage

`TenantDetailLayout` already has a `__tests__` directory
(`src/components/features/tenants/__tests__/`); `templates/` does not and one is
created. One assertion per layout: the header renders a control with accessible
name `Export`. This is the guard for FR-1.1/FR-1.2 — the per-tab claim (FR-1.3)
follows structurally from the button living in the layout and is not
re-asserted seven times.

### 7.5 Gate commands

Per PRD acceptance criteria, from the worktree root:

```
cd services/atlas-ui && npm run test
cd services/atlas-ui && npm run build      # type-checks, incl. test files
tools/lint.sh --check                      # requires nvm 22 on PATH
```

No Go module is touched, so no `go test` / `go vet` / `docker buildx bake` target
is in scope. The branch diff must contain only `services/atlas-ui/**` and
`docs/**` — that containment is itself an acceptance criterion.

---

## 8. Files touched

| File | Change |
|---|---|
| `services/atlas-ui/src/lib/utils/download-json.ts` | New |
| `services/atlas-ui/src/lib/utils/config-export.ts` | New |
| `services/atlas-ui/src/components/features/config/ConfigExportButton.tsx` | New |
| `services/atlas-ui/src/components/DetailActionBarContext.tsx` | Add `useDetailActionBarState` export |
| `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx` | Header flex row + button |
| `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx` | Header flex row + button |
| `services/atlas-ui/src/lib/utils/__tests__/download-json.test.ts` | New |
| `services/atlas-ui/src/lib/utils/__tests__/config-export.test.ts` | New |
| `services/atlas-ui/src/components/features/config/__tests__/ConfigExportButton.test.tsx` | New |
| `services/atlas-ui/src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx` | New |
| `services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx` | New or extended |

No Go file, no seed template, no k8s manifest, no `services.json` entry.

---

## 9. Risks

| Risk | Mitigation |
|---|---|
| Key order silently drifts from seed order after a future `RestModel` field addition | §7.2's `Object.keys` assertion catches a *client-side* reordering. It cannot catch a server-side field insertion — but neither can anything else in the UI, and such an insertion changes the seed files too, keeping them aligned by construction (§5.3). |
| `parseInt(opCode, 16)` on a malformed opCode yields `NaN`, making the sort comparator non-deterministic | Pre-existing in both services; the export inherits it rather than diverging. A malformed opCode is already rejected upstream by `tools/template-opcode-order-guard.sh` on the seed side. Not fixed here — fixing it would change existing display ordering, which is out of scope. |
| Radix tooltip on a disabled trigger | Dirty and disabled are mutually exclusive (§3.4); no workaround shipped. |
| jsdom lacking `URL.createObjectURL` breaks the suite | Stubbed per-test, not globally (§2.9). |
| Exported file mistaken for a seed file and committed unreviewed | The `tenant_` prefix (FR-3.2) distinguishes a live snapshot at a glance; the template prefix intentionally matches seed naming because that is the stated workflow. No further guard added — this is a human-process concern, not a code one. |

---

## 10. Open questions

None. The PRD's three open questions are resolved in §5.1 (factoring — shared
component), §2.7 (dirty-state source — new `useDetailActionBarState` export), and
§2.3 (trailing newline — confirmed present in the seeds, requirement stands).
