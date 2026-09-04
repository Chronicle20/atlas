# Task 21 Report — atlas-ui service-layer docs

## What I implemented

Documentation-only change to `services/atlas-ui/docs/service-layer.md` (the only file touched):

1. **Directory listing** — added `fields.service.ts`, `worlds.service.ts`,
   `live-monsters.service.ts`, and a note on `map-entities.service.ts` gaining `getObjects()`
   alongside its existing portals/npcs/reactors/monsters methods.
2. **New "Cache profiles: definition vs runtime" section**, inserted after "Shape of a hook" and
   before "Tenant contract" (rather than appended at the end), since it builds directly on the
   hook example just shown and its tenancy point feeds into the following section:
   - The two-row cache-profile table (definition `10*60*1000`/`10*60*1000`; runtime
     `5*1000`/`60*1000`), **without** "field environment" in the runtime row per correction C1 —
     confirmed no environment query exists anywhere in the fields/field-runtime service or hook
     files.
   - A note that `RUNTIME_STALE_TIME`/`RUNTIME_GC_TIME` are declared independently in both
     `useFields.ts` and `useFieldRuntime.ts` (duplicated, not shared) — matches C2.
   - A note on `fieldQueryOptions(filters)` as the shared assembly point for the two `useFields.ts`
     query hooks.
   - "No `refetchInterval` anywhere" statement.
   - A "Key namespaces" subsection listing all four key factories verbatim from source
     (`mapEntityKeys`, `worldKeys`, `fieldKeys`, `fieldRuntimeKeys`), explicitly calling out that
     `worldKeys` is a third, independent namespace from `["maps", …]` even though it shares the
     definition cache profile — per C2's precision note.
   - Why the `["maps", …]` / `["fields", …]` roots are disjoint (FR-41: a runtime refetch must not
     invalidate definition data).
3. **Tenant contract section** — added a paragraph stating the new hooks deliberately omit tenant
   from their query keys (because `tenant-context.tsx:68`'s `queryClient.clear()` already handles
   tenant isolation) and instead keep the `enabled: !!activeTenant` guard. Also added the `:68`
   line-reference to the existing sentence about `tenant-context.tsx`.

## Deviation from the brief

None beyond what the brief's own "Controller pre-flight corrections" section directs (dropping
"field environment" per C1, skipping `tools/verify.sh` per C3). I read the corrections and followed
them exactly; I did not add anything the corrections told me to omit.

## What I verified against source (not re-derived, but read to confirm)

- `services/atlas-ui/src/lib/hooks/api/useFields.ts` — confirmed `fieldKeys`, `RUNTIME_STALE_TIME`
  = `5 * 1000`, `RUNTIME_GC_TIME` = `60 * 1000`, `fieldQueryOptions` export, both `useFields` and
  `useFieldsForMap` spreading it, and no `refetchInterval`.
- `services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts` — confirmed `fieldRuntimeKeys` shape
  (`["fields", w, c, m, i, "monsters"|"characters"]`), its own independently-declared
  `RUNTIME_STALE_TIME`/`RUNTIME_GC_TIME`, and no `refetchInterval`.
- `services/atlas-ui/src/lib/hooks/api/useWorlds.ts` — confirmed `worldKeys` shape and the literal
  `10 * 60 * 1000` staleTime/gcTime at both `useWorlds` and `useChannels`.
- `services/atlas-ui/src/lib/hooks/api/useMapEntities.ts` — confirmed `mapEntityKeys` shape
  (`["maps", mapId, "portals"|"npcs"|"reactors"|"monsters"|"objects"]`) and the literal
  `10 * 60 * 1000` at all five query hooks including `useMapObjects`.
- `services/atlas-ui/src/services/api/fields.service.ts`, `worlds.service.ts`,
  `live-monsters.service.ts` — read to confirm the modules exist and their general shape (thin
  `api.getList` adapters) matches the "Shape of a service" section already in the doc, so no new
  example was needed there.

## Testing / checks run (per brief C3 — narrow scope only)

```
$ grep -n '/home/' services/atlas-ui/docs/service-layer.md
(no output — exit 1)

$ grep -rn 'refetchInterval' services/atlas-ui/src/lib/hooks/api/useFields.ts services/atlas-ui/src/lib/hooks/api/useWorlds.ts services/atlas-ui/src/lib/hooks/api/useFieldRuntime.ts services/atlas-ui/src/lib/hooks/api/useMapEntities.ts
(no output — exit 1)
```

Both expected-empty. Did **not** run `tools/verify.sh`, the UI build, `tsc`, or the test suite —
per C3, that's out of my scope and owned by the controller/gate worktree.

## Files changed

- `services/atlas-ui/docs/service-layer.md` (+55/-1)

## Self-review

- Doc voice/heading depth matches the existing file (`##`/`###`, prose paragraphs, inline code for
  identifiers, no bolt-on section — new material placed where it's structurally relevant).
- All paths are repo-relative; grep for `/home/` confirmed.
- No stray `refetchInterval` introduced or claimed to exist; grep confirmed against the four hook
  files named in the brief.
- Commit stages exactly one file (`git status --porcelain` before staging showed only
  `service-layer.md` modified plus an untracked `agent-ledger.tsv`, which was left unstaged per
  C4).

## Issues or concerns

None. Clean, narrow, brief-conformant change.

## Commit

`a47f63a2a docs(atlas-ui): document field service modules and cache profiles`

Confirmed post-commit: worktree root and current branch both resolved correctly via
`git rev-parse --show-toplevel` (repo-root worktree path for this task) and
`git branch --show-current` (`task-292-map-definition-field-split`). Both as expected.
