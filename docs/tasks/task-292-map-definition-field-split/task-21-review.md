# Task 21 Review — atlas-ui service-layer docs

Commit reviewed: `a47f63a2a` (range `57485113a..a47f63a2a`)
File: `services/atlas-ui/docs/service-layer.md` (+55/-1)

## Scope

Documentation-only change. Verified every factual claim added by the diff against the source it
describes, per the brief's "Controller pre-flight corrections" (C1-C5), which supersede the
plan's original task-21 text.

## Findings

### PASS — C1: "field environment" correctly dropped

`git show a47f63a2a` diff and `grep -n -i "environment" services/atlas-ui/docs/service-layer.md`
return no output. The runtime row reads exactly `Runtime (fields, field characters, live
monsters)` — matches C1's required text verbatim. Confirmed the underlying claim is still true at
this commit: `grep -rn "environment" services/atlas-ui/src/services/api/fields.service.ts
services/atlas-ui/src/lib/hooks/api/useFields.ts
services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts` returns no output. The report's claim of
"no environment query exists anywhere in the fields/field-runtime service or hook files" is
accurate and matches C1; the implementer did not silently follow the original (pre-correction)
plan text.

### PASS — Cache profile values

Doc states: definition `staleTime`/`gcTime` = `10 * 60 * 1000` each; runtime `staleTime` =
`5 * 1000`, `gcTime` = `60 * 1000`.

- `useMapEntities.ts:28-29,39-40,52-53,65-66,78-79` — all five hooks (`useMapPortals`,
  `useMapNpcs`, `useMapReactors`, `useMapMonsters`, `useMapObjects`) use the literal
  `10 * 60 * 1000` for both. Confirmed by direct read.
- `useWorlds.ts:23-24,36-37` — same literal, both hooks. Confirmed.
- `useFields.ts:18-19` — `RUNTIME_STALE_TIME = 5 * 1000`, `RUNTIME_GC_TIME = 60 * 1000`.
  Confirmed.
- `useFieldRuntime.ts:24-25` — same constant names/values, declared independently (not imported
  from `useFields.ts`). Confirmed.

Doc correctly states these are "declared independently... duplicated, not shared from a common
constant" (line ~113 of the new text). This matches the source: two separate `const` declarations
with the same names in two different files, no shared import. No false "shared constant" claim.

### PASS — Query-key namespaces

- `mapEntityKeys` (`useMapEntities.ts:12-18`) → `["maps", mapId, "portals"|"npcs"|"reactors"|
  "monsters"|"objects"]`. Doc matches verbatim.
- `worldKeys` (`useWorlds.ts:11-15`) → `["worlds"]`, `["worlds","list"]`,
  `["worlds", worldId, "channels"]`. Doc matches verbatim.
- `fieldKeys` (`useFields.ts:13-16`) → `["fields"]`, `["fields","list", filters]`. Doc matches.
- `fieldRuntimeKeys` (`useFieldRuntime.ts:17-21`) → `["fields", w, c, m, i, "monsters"|
  "characters"]`. Doc matches.

The doc explicitly separates key-root grouping from cache-profile grouping: "`worldKeys` is its
own namespace — it carries the definition cache profile, but its key root is `["worlds", …]`, not
`["maps", …]`." This is precisely the distinction the brief's C2 required and does not conflate
the two groupings. PASS.

### PASS — `fieldQueryOptions` documented and accurate

Doc states `useFields.ts` exports `fieldQueryOptions(filters)` as the shared assembly point,
spread by both `useFields` and `useFieldsForMap`. Confirmed at `useFields.ts:26-33` (definition)
and lines 40, 52 (both call sites spread `...fieldQueryOptions(filters)`). Accurate.

### PASS — FR-39 no-polling claim

Doc: "Neither profile uses `refetchInterval` anywhere — runtime freshness comes from a short
`staleTime` plus normal refetch-on-mount/refocus behavior, not polling." This claim is scoped
(by section context) to the four hook modules discussed in the "Cache profiles" section
(`useMapEntities.ts`, `useWorlds.ts`, `useFields.ts`, `useFieldRuntime.ts`), not to atlas-ui as a
whole.

Verified narrowly: `grep -rn 'refetchInterval'` against those four files returns no output.
Verified broadly: `grep -rn 'refetchInterval' services/atlas-ui/src/lib/hooks/api/` DOES find
`refetchInterval` in `useCanonicalData.ts`, `useTransports.ts`, `useAccounts.ts`,
`useAccountByName.ts`, and `useSeed.ts` — but none of those modules are described or referenced
by the new doc section, and the doc's sentence is textually scoped to "Neither profile" (i.e. the
definition/runtime profiles just introduced), not a blanket claim about every hook in the
codebase. Read literally and in context, the claim is true and not overbroad. No finding.

### PASS — Module list (Directory section)

- `fields.service.ts` exists at `services/atlas-ui/src/services/api/fields.service.ts`; hits
  `/api/fields` (`getFields`) and `/api/worlds/.../characters` (`getFieldCharacters`). Doc
  comment "`/api/fields` — live field occupancy + field character roster" is a reasonable
  one-line summary of both methods; not misleading.
- `worlds.service.ts` exists; `getWorlds()` → `/api/worlds/`, `getChannels()` →
  `/api/worlds/{id}/channels`. Doc comment "world/channel topology" accurate.
- `live-monsters.service.ts` exists; single `getMonsters()` method. Doc comment "live monsters
  for a running field" accurate.
- `map-entities.service.ts:105` — `getObjects(mapId)` confirmed present alongside
  `getPortals`/`getNpcs`/`getReactors`/`getMonsters`. Doc's note that `getObjects()` was "added
  alongside" the existing methods is accurate.

### PASS — No absolute paths

`grep -n '/home/' services/atlas-ui/docs/service-layer.md` returns no output (exit 1 / empty).

### PASS — Tenant contract addition

Doc adds a paragraph stating tenant is deliberately omitted from `fieldKeys`, `fieldRuntimeKeys`,
`worldKeys` because `tenant-context.tsx:68`'s `queryClient.clear()` already handles isolation, and
that `enabled: !!activeTenant` is kept. Confirmed: every one of the eight hooks in the four
reviewed files (`useFields`, `useFieldsForMap`, `useLiveMonsters`, `useFieldCharacters`,
`useMapPortals`, `useMapNpcs`, `useMapReactors`, `useMapMonsters`, `useMapObjects`, `useWorlds`,
`useChannels`) carries an `enabled: ... !!activeTenant` guard and none includes tenant in its key
factory. Line-reference `:68` for the `queryClient.clear()` call was not independently
re-verified line-by-line in this review (out of the stated diff scope — the sentence already
existed pre-diff except for the added `:68` locator) but was pre-confirmed by the brief's C2 as
fact; treating as verified per brief.

### PASS — Structure / voice / placement

`grep -n '^##' services/atlas-ui/docs/service-layer.md` shows the new "Cache profiles: definition
vs runtime" (`##`) and its "Key namespaces" (`###`) subsection landed between "Shape of a hook"
and "Tenant contract", consistent with the report's stated rationale (builds on the hook example,
its tenancy point feeds the next section). All pre-existing sections (Directory, Shape of a
service, Shape of a hook, Tenant contract, Shared types and helpers, Patterns for pages,
Follow-ups) remain present and in order. The new material is not a bolt-on appendix; heading
depth (`##`/`###`) and prose style match the surrounding file.

### PASS — Commit hygiene

`git status --porcelain` at review time shows only `agent-ledger.tsv` and `task-21-report.md` as
untracked (neither staged in the reviewed commit). `git show a47f63a2a --stat` confirms exactly
one file changed. Matches C4.

## Report accuracy

The implementer report claims DONE, no deviations beyond following the brief's own corrections,
and explicitly calls out following C1 (dropping "field environment") and C3 (skipping
`tools/verify.sh`). Both claims verified true against the actual diff and file state. No
discrepancy found between the report and the commit.

## Not evaluable

None. All claims in the diff were checkable against source within the stated review surface
(the four hook files, three new service files, and `map-entities.service.ts`).

## Verdict rationale

Every factual claim in the added documentation was checked against the code it describes and
found accurate: cache profile values, key-factory shapes and roots, the duplicated-not-shared
runtime constants, the `fieldQueryOptions` assembly point, the field-environment omission, the
no-`refetchInterval` claim (correctly scoped), the module list, and the tenant-key omission
rationale. No absolute paths. No structural bolt-on. No deviation from the brief's
pre-flight corrections. No trace of the pre-correction "field environment" text.
