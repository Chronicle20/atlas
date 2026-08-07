# task-199 — Implementation Context

Companion to [`plan.md`](./plan.md). Everything here was read from the branch
during the planning pass; line numbers are from
`.worktrees/task-199-config-json-export` at plan time.

## Scope in one line

Add an **Export** button to the Template/Tenant detail-page headers that
downloads the viewed configuration's JSON:API `attributes` as a seed-shaped
`.json` file. Frontend only.

## Key files

### Touched

| File | Current state |
|---|---|
| `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx` | 6 sidebar items; header is a bare `div.space-y-0.5` with `<h2>Template Details</h2>` + `<p>{id}</p>`; wrapped in `DetailActionBarProvider`; `DetailActionBar` pinned below the scroll area. |
| `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx` | Structurally identical, 7 sidebar items (adds MTS Configuration), heading `Tenant Details`. |
| `services/atlas-ui/src/components/DetailActionBarContext.tsx` | Exports `DetailActionBarConfig`, `DetailActionBarProvider`, `useRegisterDetailActionBar` (write), `DetailActionBar` (renderer). The context object itself (line 30) is a module-level `const` with **no** export — hence the new `useDetailActionBarState` read accessor. |

### Read, not touched

| File | What matters |
|---|---|
| `src/lib/hooks/api/useTemplates.ts:113` | `useTemplate(id)` → `UseQueryResult<Template, Error>`, key `templateKeys.detail(id)` (line 36), `enabled: !!id`, `gcTime` 10 min. |
| `src/lib/hooks/api/useTenants.ts:228` | `useTenantConfiguration(id)` → `UseQueryResult<TenantConfig, Error>`, key `tenantKeys.configDetail(id)` (line 42), `enabled: !!id`. |
| `src/services/api/templates.service.ts:61-79` | `sortTemplate` normalises `npcs`/`worlds` `null → []` **and** sorts handlers/writers by `parseInt(opCode, 16)`. |
| `src/services/api/tenants.service.ts:169-187` | `sortTenantConfig` sorts only — **no** null normalisation. This is why the export normalises for itself. |
| `src/types/models/template.ts:69-120` | `TemplateAttributes` (with optional `cashShop`), `Template = { id, attributes }`. |
| `src/services/api/tenants.service.ts:75-131,320-325` | `TenantConfigAttributes` / `TenantConfig`, both re-exported from the `export type { … }` block at line 320. Structurally identical to `TemplateAttributes`. |
| `src/types/models/socket.ts` | `SocketConfig { handlers: SocketHandlerEntry[]; writers: SocketWriterEntry[]; unsupported?: SocketUnsupported }`. |
| `src/components/ui/tooltip.tsx:20-28` | **`Tooltip` already renders its own `TooltipProvider`.** No extra provider is needed (corrects design.md §2.6). |
| `src/components/features/socket/DefinitionGridPage.tsx:62-65` | The established call-both-hooks-with-an-empty-id pattern the export button copies. |
| `src/components/features/npc/NpcShopCard.tsx:263-274` | The inline Blob/anchor download this task generalises. **Left untouched** — migrating it would add a trailing newline to an unrelated feature's output. |
| `src/lib/utils/index.ts` | Barrel re-exports only `./toast` and `./maplestory`. The two new util modules are deliberately **not** added to it. |
| `src/test/setup.ts` | Stubs `matchMedia` and `ResizeObserver` only — **not** `URL.createObjectURL`. Stub it per-suite. |
| `vitest.config.ts:27-35` | `environment: "jsdom"`, `globals: true`, `include: ["src/**/*.test.{ts,tsx}"]`. |
| `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` | The target file shape: 2-space indent, keys `region, majorVersion, minorVersion, usesPin, socket, characters, npcs, worlds, cashShop`, trailing newline. |

## Decisions carried in from design.md

1. **One shared `ConfigExportButton` with a `kind` prop**, not two per-feature
   components — the two payload shapes are identical, and a per-feature split
   would still have to call both query hooks for Rules-of-Hooks reasons.
2. **Payload comes from the React Query cache**, never from page form state.
   The export is defined as "the last saved document"; that is what makes it
   diffable against a seed template.
3. **The export owns its output contract** — it re-sorts and re-normalises even
   where `sortTemplate` already did, rather than widening `sortTenantConfig`
   (which feeds every tenant-config consumer in the app).
4. **No key-order table.** Spread order reproduces seed order because the Go
   `RestModel` field order and the seed key order are the same; an explicit
   table would be a second source of truth that silently goes stale.
5. **No `id`-stripping code.** Both service layers already project down to
   `{ id, attributes }`, and only `.attributes` is ever passed to the
   projection. The test asserts the absence anyway.

## Plan-time corrections to design.md

Recorded in plan.md's "Deviations" section; repeated here so they are not lost:

1. No extra `TooltipProvider` (the shadcn `Tooltip` wrapper mounts one).
2. `configExportFilename` is called with explicit `region`/`majorVersion`/
   `minorVersion` rather than a spread of `attributes`.
3. `toConfigExportPayload` builds a `Record<string, unknown>` and asserts
   `as T` once on return — assigning to a property of a generic `T` is not
   expressible in TypeScript.

## Honest scoping notes

- **FR-5.4** ("no additional network request") is satisfiable only at the
  *click* level. React Query's defaults are `staleTime: 0` /
  `refetchOnMount: true` (`src/lib/query-client.ts`), so a mounting observer
  does background-refetch. The layout and its first child page mount in the
  same commit and observe the same key, so React Query dedupes them into one
  in-flight fetch — the button adds no request beyond what the page already
  made. The test asserts the service call count is unchanged **across the
  click**.
- **`parseInt(opCode, 16)` yields `NaN` on a malformed opCode**, making the
  comparator non-deterministic. Pre-existing in both services; inherited
  rather than diverged from. Not fixed here — fixing it would change existing
  display ordering.

## Verification

No Go module changes, so no `go test` / `go vet` / `docker buildx bake` target
is in scope. From the worktree root:

```
cd services/atlas-ui && npm run test
cd services/atlas-ui && npm run build      # this is the type-check
tools/lint.sh --check                      # needs nvm 22 on PATH
```

`tools/lint.sh --check` false-fails without nvm on PATH — if it errors before
linting anything, load nvm 22 first rather than treating it as a code failure.

## Dependencies

None outside this branch. No other in-flight task touches
`TemplateDetailLayout.tsx`, `TenantDetailLayout.tsx`, or
`DetailActionBarContext.tsx`.
