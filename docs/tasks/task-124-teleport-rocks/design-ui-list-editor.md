# Design — Teleport-Rock List Editor (Character Detail Page)

Date: 2026-07-18
Status: Approved (follow-on to the task-124 teleport-rock feature)

## 1. Purpose & Scope

Give the atlas-ui **Character Detail** page admin CRUD over a character's two
teleport-rock destination lists (**Regular**, capacity 5; **VIP**, capacity 10).
Each list is presented as its own card showing the rock item icon, a
`used of capacity` count, the saved maps (resolved to names), an **Add** action
(searchable map picker), and per-row **Delete**.

Validation is faithful to the in-game flow: a map must pass
`EligibleForRegistration`, the list must not be full, and duplicates are
rejected. Admin edits write to atlas-character (the source of truth) and, when
the character is online, refresh the in-game list via the existing
`LIST_UPDATED` status event.

Out of scope: changing the in-game `TROCK_ADD_MAP` packet flow, cross-channel
concerns, and any new map-picker primitive (built from existing UI components).

## 2. Backend — atlas-character

### 2.1 Processor refactor (`teleport_rock/processor.go`)

Today `AddMap`/`RemoveMap` handle a validation failure by buffering an ERROR
status event and returning `nil`. That is correct for the game (the client
renders from the result packet) but hides success/failure from a REST caller.
Refactor so **validation returns a typed error**, emitted before anything is
buffered:

- Introduce typed/sentinel errors for the four reasons:
  `ErrMapNotAllowed`, `ErrListFull`, `ErrDuplicate`, `ErrNotFound`.
- The shared validate → mutate → buffer-`LIST_UPDATED` logic returns the typed
  error on validation failure (buffering nothing) and buffers `LIST_UPDATED`
  on success (unchanged).
- **Game/async path** — `AddMapAndEmit` / `RemoveMapAndEmit` (used by the
  command consumer) catch the typed error inside the `message.Emit` closure,
  buffer the matching `errorEventProvider(...)` ERROR status event, and return
  `nil`. In-game behavior is preserved byte-for-byte; the ERROR-event mapping
  simply moves from inside the validation branch to the wrapper.
- **REST/sync path** — new `Add(transactionId, worldId, characterId, mapId, vip)
  (Model, error)` and `Remove(...) (Model, error)`. On validation failure the
  typed error propagates (no event). On success `LIST_UPDATED` is flushed and
  the updated `Model` is returned.

Because validation runs before any buffering, a failure never emits a partial
event regardless of `message.Emit` flush semantics.

### 2.2 REST endpoints (`teleport_rock/resource.go`, `rest.go`)

Extend the existing `/characters/{characterId}/teleport-rock-maps` resource
(currently GET-only). No new ingress route is required — the path already
routes to atlas-character; only the method set grows.

- `POST /characters/{characterId}/teleport-rock-maps`
  JSON:API body `{data:{type:"teleport-rock-maps", attributes:{list:"regular"|"vip", mapId:N}}}`
  → add. Returns the updated aggregate `RestModel` (200).
- `DELETE /characters/{characterId}/teleport-rock-maps/{list}/{mapId}`
  → remove. Returns the updated aggregate `RestModel` (200).

Both handlers:
1. Resolve the character's `worldId` from the character record (for the
   `LIST_UPDATED` event envelope) and mint a fresh `transactionId`.
2. Call the synchronous `Add`/`Remove`.
3. Map a typed error to an HTTP status; otherwise marshal the updated model.

**Error → HTTP mapping**

| Typed error        | HTTP | Case                         |
|--------------------|------|------------------------------|
| `ErrMapNotAllowed` | 400  | map fails eligibility        |
| `ErrListFull`      | 409  | list at capacity (5 / 10)    |
| `ErrDuplicate`     | 409  | map already in the list      |
| `ErrNotFound`      | 404  | remove a map not in the list |

### 2.3 Response shape (`RestModel`)

Add capacity to the read model so the UI renders `used of capacity` from a
single source of truth (no hardcoded 5/10):

```go
type RestModel struct {
    Id              string    `json:"-"`
    Regular         []_map.Id `json:"regular"`
    Vip             []_map.Id `json:"vip"`
    RegularCapacity int       `json:"regularCapacity"` // model.RegularCapacity
    VipCapacity     int       `json:"vipCapacity"`     // model.VipCapacity
}
```

`Transform` populates the capacities from `model.Capacity(false/true)`.

### 2.4 Live refresh (no longer a limitation)

Both paths run the same validate → mutate → emit logic; on success the
synchronous REST path flushes `LIST_UPDATED`, which the existing atlas-channel
status consumer turns into a `MAP_TRANSFER_RESULT` list refresh for the online
character (routed by `characterId`). No channel routing lives in the REST
handler.

### 2.5 Docs

Update the atlas-character README REST-endpoints table with the new POST/DELETE
methods and the `regularCapacity`/`vipCapacity` fields. No ingress change.

## 3. Frontend — atlas-ui

### 3.1 Service (`services/api/teleport-rocks.service.ts`)

Mirror the existing per-character service pattern (JSON:API envelope):

- `getByCharacterId(characterId)` → `{ regular, vip, regularCapacity, vipCapacity }`.
- `addMap(characterId, list, mapId)` → POST; returns the updated model.
- `removeMap(characterId, list, mapId)` → DELETE; returns the updated model.
- Surfaces backend error status/detail so the hook can toast a faithful message.

### 3.2 Hooks (`lib/hooks/api/useTeleportRocks.ts`)

- `useTeleportRockMaps(tenant, characterId)` — React Query read.
- `useAddTeleportRockMap()` / `useRemoveTeleportRockMap()` — mutations that
  invalidate the read query on success.

### 3.3 Components (`components/features/characters/`)

- `TeleportRockListCard.tsx` — one card per list. Props: `characterId`, `vip`
  (list discriminator), `maps`, `capacity`. Header: rock icon via
  `useItemData(vip ? 5041000 : 5040000)` + title + `used of capacity` badge.
  Body: `ScrollArea` of rows, each resolving its map name via `useMap` with a
  trash button (immediate delete → mutation + toast; re-addable, no confirm).
  Footer: **Add** button, disabled at capacity.
- `AddTeleportRockMapDialog.tsx` — reuses the `Dialog` primitive (like
  `ChangeMapDialog`). A debounced search `Input` backed by `useMapsByName`,
  results in a `ScrollArea`; selecting a result triggers the add mutation.
  Eligibility/capacity/duplicate rejections surface as an error toast.

### 3.4 Wiring

Render the two cards at the bottom of `pages/CharacterDetailPage.tsx`, gated on
the teleport-rock query having loaded. Reuse the page's existing `sonner`
`Toaster` for success/error toasts.

## 4. Data Flow

```
Admin (Character Detail card)
  └─ Add: pick map ─▶ POST /characters/{id}/teleport-rock-maps {list,mapId}
       atlas-character: worldId lookup ─▶ Add() validate→mutate→emit LIST_UPDATED
         ├─ validation fail ─▶ typed error ─▶ HTTP 400/409 ─▶ error toast
         └─ success ─▶ 200 updated model ─▶ React Query invalidate ─▶ card refresh
                          └─ LIST_UPDATED ─▶ atlas-channel status consumer
                                              └─ online character's in-game list refresh
  └─ Delete (trash): DELETE /characters/{id}/teleport-rock-maps/{list}/{mapId}
       (same path; ErrNotFound ─▶ 404)
```

## 5. Testing

**Backend (atlas-character)** — Builder-pattern setup, no test-helper files:
- Processor `Add`/`Remove`: eligibility reject, list-full, duplicate,
  remove-not-present, and success returns the updated model + buffers
  `LIST_UPDATED`.
- Preserved async behavior: `AddMapAndEmit`/`RemoveMapAndEmit` still emit the
  ERROR status event on each failure reason (existing consumer tests stay green).
- REST handlers: status-code mapping per reason; JSON:API round-trip incl. the
  new capacity fields.

**Frontend (atlas-ui)**:
- Service: envelope shape for add/remove; error propagation from backend status.
- `TeleportRockListCard`: renders list + `used of capacity`; Add disabled at
  capacity; delete calls the mutation.
- `AddTeleportRockMapDialog`: debounced search → select → add mutation; error
  toast on a rejected add.
- Built entirely from existing primitives (`Dialog`/`Input`/`ScrollArea`/
  `Badge`) — no new dependency.

## 6. Decisions & Non-Goals

- **Header icons** use the cash rock ids `5040000` (Regular) / `5041000` (VIP) —
  the matched visual pair.
- **Add semantics**: pick any map (there is no "current map" in admin context),
  but enforce the same in-game rules (`EligibleForRegistration` + capacity +
  duplicate).
- **Delete** is immediate and re-addable (toast, no confirm dialog).
- **No new UI primitive**: the searchable picker is composed from existing
  components rather than adding `cmdk`/`popover`.
- **Non-goal**: altering the in-game `TROCK_ADD_MAP` packet path; cross-channel
  refresh; per-tenant capacity overrides.
