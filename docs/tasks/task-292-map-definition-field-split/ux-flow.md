# UX Flow — Map Definition / Field Separation

A clickable prototype of every screen and empty state in this PRD lives next to this file at
`ux-prototype.html`. It is a single self-contained HTML file — open it directly in a browser, no
server or build step. All data in it is mocked; it exists to fix layout, terminology, and navigation,
not to specify implementation.

## Prototype routes

| Prototype route | PRD section | What it demonstrates |
|---|---|---|
| `#/maps` | §4.2 | Map list with a live-field count column |
| `#/maps/910340000` | §4.2 | Map Definition with three live fields |
| `#/maps/610030300` | FR-8 | Map Definition, no-live-fields empty state |
| `#/maps/610030300/objects` | FR-6 | Map Objects definition tab, `OBSTACLE` kind |
| `#/fields` | §4.3 | Fields locator with world/channel/map filters |
| `#/fields` → Channel 3 | FR-15 | No-matching-fields empty state with filter echo |
| `#/fields/0/1/910340000/1` | §4.4 | Field detail: overview, live summary, static context |
| `#/fields/0/1/910340000/1/objects` | §4.7 | Map Objects tab, tracked + untracked merge, Set/Reset |

## Navigation model

```
                    ┌────────────────┐
                    │ Map Definition │  configuration
                    └───────┬────────┘
                            │ Live Fields
                            ▼
                       ┌─────────┐
                       │  Field  │       observation
                       └────┬────┘
          ┌─────────────────┼──────────────────┐
          ▼                 ▼                  ▼
     Characters         Monsters          Map Objects
          │                 │                  │
          ▼                 ▼                  ▼
 Character Detail   Monster Definition   Map Definition
                    + Map Spawn Definition
```

Both entry directions are supported: "tell me about this map" (Maps → Map Definition) and "tell me
what is happening right now" (Fields → Field Detail).

## Known divergences from the prototype

- The prototype's Live Fields table shows `Age` and `State` columns. These are **not** in the PRD —
  see Open Question 1: the non-empty liveness rule gives no `createdAt` and no lifecycle state to
  report. Ignore those two columns.
- The prototype applies Set immediately with only a toast. The PRD requires a confirmation dialog on
  both Set and Reset (FR-36).
- The prototype's monster columns are illustrative; FR-29 requires dropping any column the
  `atlas-monsters` payload does not actually provide (Open Question 3).
- Pin placement on the map image is decorative in the prototype. FR-19 requires the component accept
  positioned runtime entities, but does not require accurate placement in this task.
