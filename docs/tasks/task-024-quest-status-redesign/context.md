# Task 024 — Quest Status Redesign — Execution Context

Quick-reference companion to `plan.md`. Use this to orient a fresh session
before touching code.

## Scope at a glance

| | |
| --- | --- |
| Service | `services/atlas-ui` only |
| Backend changes | **None** |
| Production files edited | 3 |
| New test files | 2 |
| Estimated granularity | 7 tasks, TDD per step |

## Files to create

- `services/atlas-ui/src/components/features/quests/__tests__/QuestName.test.tsx`
- `services/atlas-ui/src/components/features/quests/__tests__/QuestStatusTabs.test.tsx`

## Files to modify

- `services/atlas-ui/src/components/features/quests/EntityName.tsx`
  - Add `QuestName` function at the end (mirrors `NpcName` / `ItemName` shape
    but reads from `useQuest` + `useTenant` instead of a unified data hook).
- `services/atlas-ui/src/components/features/quests/index.ts`
  - Add `QuestName` to the existing `EntityName.tsx` barrel line.
- `services/atlas-ui/src/components/features/quests/QuestStatusTabs.tsx`
  - Replace inner `QuestStatusCard` with a new inner `QuestStatusWidget`.
  - Swap each `TabsContent` inner container from a vertical stack
    (`space-y-3`) to a responsive grid.
  - Drop `ExternalLink` from `lucide-react` imports; drop `Link` wrapping the
    trailing icon button; drop the `showProgress` prop path.
  - Remove `expirationTime` rendering entirely.

## Design decisions already made (don't re-open)

- **Tenant source** — `QuestName` reads `useTenant()` internally; parent does
  NOT thread tenant as a prop (deviation from PRD sketch, aligned with rest
  of `*Name` family in-repo; see design §3.2).
- **Name primitive location** — added to the existing `EntityName.tsx` barrel;
  NOT a new file (design §3.1).
- **Widget location** — stays inline in `QuestStatusTabs.tsx`; NOT a new
  file (design §3.3).
- **Truncation tooltip** — `title` attribute on `QuestName`'s inner `<span>`
  only; parent widget does not re-query for the name (design §3.4).
- **`EntityWidget.tsx` left alone** — its `QuestWidget` shares the React
  Query cache with `QuestName`, so there is no duplication at runtime
  (design §3.5). Out of scope.

## Key types

```ts
// src/types/models/quest.ts
export interface CharacterQuestStatus {
    id: string
    type: "quest-status"
    attributes: CharacterQuestStatusAttributes
}

export interface CharacterQuestStatusAttributes {
    characterId: number
    questId: number
    state: QuestState
    startedAt: string
    completedAt?: string
    expirationTime?: string       // RENDERING DROPPED
    completedCount: number
    forfeitCount: number
    progress: QuestProgress[]     // RENDERING DROPPED
}

export interface QuestDefinition {
    id: string
    type: "quests"
    attributes: QuestAttributes   // .name is what QuestName renders
}
```

## Hook shapes to remember

```ts
// src/lib/hooks/api/useQuests.ts
useQuest(tenant: Tenant | null, id: string): UseQueryResult<QuestDefinition, Error>
// → returns { data, isLoading, isError, ... }
// → enabled: !!tenant?.id && !!id (so no-tenant is a safe no-op)
// → staleTime / gcTime: 10 minutes
```

```ts
// src/context/tenant-context.tsx
useTenant(): {
    activeTenant: Tenant | null
    // ...other fields not needed here
}
```

**Note:** the `*Name` family previously in `EntityName.tsx` (Npc, Item, Mob,
Skill) uses `useNpcData` / `useItemData` / etc., which already flatten to
`{ name, isLoading, hasError }`. `useQuest` is a raw React Query result and
does NOT flatten. `QuestName` dereferences `data?.attributes.name` itself.

## Testing setup

- Vitest + `@testing-library/react` + `jsdom`.
- Global test APIs (`describe`, `it`, `expect`, `vi`) enabled via
  `src/test/setup.ts`.
- Jest-era tests (`jest.fn`/`jest.mock`) exist elsewhere in the repo but are
  excluded from `tsc -b`; write new tests with `vi.*` only.
- Mock examples to imitate: `src/pages/__tests__/TenantsPage.test.tsx`
  (shows `vi.mock` + `MemoryRouter` + `useTenantMock` pattern).

## Verification commands

```bash
cd services/atlas-ui
npm run lint                 # ESLint must pass
npm run test -- --run \
    src/components/features/quests/__tests__   # New tests must pass
npm run build                # tsc -b + vite build must pass
```

Optional smoke test: `npm run dev`, open a character detail page, switch
between Started and Completed tabs, confirm grid layout and whole-card link
at narrow/`sm`/`lg` viewport widths.

## Out of scope (do NOT touch)

- `services/atlas-ui/src/components/features/quests/EntityWidget.tsx`
- `services/atlas-quest/atlas.com/quest/quest/rest.go` — the
  `ExpirationTime` zero-date serialization bug stays; we simply stop
  rendering the field.
- Any page other than the Character Detail Quest Status section.
- Any hook, service, or route file.
