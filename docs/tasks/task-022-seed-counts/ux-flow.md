# Seed Counts — UX Flow

Companion to `prd.md`. Describes the user-visible behavior of the Seed Data panel on `/setup` after this feature lands.

## Layout

The Seed Data panel adopts the same row primitive as the Game Data panel above it. Each row has four slots:

```
[icon] [label + badge]                                                   [Seed button]
```

Where:

- **icon** — existing per-row lucide icon (`Database`, `Package`, `MessageSquare`, …).
- **label** — existing row title (e.g. "Monster & Reactor Drops").
- **badge** — new live-polled count string. Rendered inside the same `aria-live="polite"` `<p>` the Game Data rows use.
- **Seed button** — existing button. Disabled only while that row's own mutation is in flight.

Ordering and icons are unchanged from the current `SetupPage.tsx`.

## Badge formats

| Row | Example badge |
|---|---|
| Monster & Reactor Drops | `12,040 monster drops / 48 continent drops / 6,116 reactor drops` |
| Gachapons | `17 gachapons / 842 items / 60 global items` |
| NPC Conversations | `1,284 conversations` |
| Quest Conversations | `517 conversations` |
| NPC Shops | `148 shops / 2,341 commodities` |
| Portal Scripts | `61 scripts` |
| Reactor Scripts | `89 scripts` |
| Map Action Scripts | `210 scripts` |

Singular form kicks in at `1` via the existing `pluralize(n, singular, plural)` helper — `1 script`, `2 scripts`.

## States

### Initial load

1. User navigates to `/setup`.
2. Every row's badge reads `"—"` for ~a few hundred ms while the first poll flies.
3. First successful response paints the live count.

### Active tenant switch

1. `TenantProvider` fires `queryClient.clear()` as per existing contract.
2. Every row's badge reverts to `"—"` for a moment.
3. Counts repaint with the new tenant's numbers on the next poll tick (5s worst case, but the initial fetch fires immediately on cache miss because `staleTime: 0`).

### Clicking Seed on a sync row (e.g. NPC Shops)

1. User clicks Seed. Button disables, shows spinner.
2. Sync POST runs for a few hundred ms to a few seconds depending on file count.
3. `onSuccess` invalidates the row's status query.
4. Button re-enables, toast says "Seeded NPC Shops".
5. Badge repaints with the new numbers on the next poll tick (typically within ~1s of invalidation because React Query refetches on invalidation).

### Clicking Seed on an async row (e.g. Monster & Reactor Drops)

1. User clicks Seed. Button disables, spinner shown.
2. POST returns `202 Accepted` almost immediately.
3. `onSuccess` fires, button re-enables, toast says "Seeding Drops…".
4. The async goroutine in atlas-drop-information begins writing rows.
5. Badge reflects the growing counts on each 5s poll: `0 monster drops / 0 continent drops / 0 reactor drops` → `4,000 / 12 / 2,100` → … → the final numbers.
6. User sees the counts stop changing — that is the implicit "done" signal. There is no explicit success banner for async seeds (matching the current UX).

### Service unavailable

1. A count endpoint returns `500` or the fetch fails (service is down, network blip).
2. React Query marks the query as errored. Badge renders `"—"`.
3. No toast. No banner. Next 5s poll retries silently.
4. When the service recovers, the next successful poll repaints the badge.

### Stale / lagged count

If a poll fires while the async seed is mid-transaction, `COUNT(*)` reflects only the committed rows. The badge may jump in large chunks between polls depending on transaction granularity. This is acceptable and matches the observability model of Game Data's XML count during extraction.

## Non-states

- No progress bar, no percent indicator.
- No "seeded 3m ago" timestamp in v1 (the endpoint returns `updatedAt` but the UI ignores it).
- No "reset" or "clear" action — operators re-seed to reset.
- No per-sub-resource seed button — Seed still fires the existing compound seed.
