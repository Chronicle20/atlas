# Mystic Door — Cosmic Visibility Model (rework)

- **Task:** task-093-mystic-door (root-cause rework, on the task branch)
- **Date:** 2026-06-18
- **Status:** Design — approved; supersedes the party-filtered area-door visibility (incl. the `ReconcileParty` area-door re-sends from `design-reconciliation.md`).

## 1. Problem

atlas renders a Mystic Door's **area door** (the physical door, `SpawnDoor`) only to the owner + party members, and **re-sends `SpawnDoor`** whenever party membership changes (the `ReconcileParty` machinery). The v83 client treats a repeat `SpawnDoor` for a door it already has as an open/close **toggle**: the close-toggle swaps the body to the "closing" sprite (which renders well below the platform) and **clears `pLayerFrame` to null**, which then crashes `OnTownPortalRemoved` on the next removal (E_POINTER `-2147467261`, confirmed live: `nState=1, pLayerFrame=0`).

Every rendering symptom we chased — door drawn below the platform, the owner's *own* door breaking on expel, and the expiry crash — is one cause: **re-sending `SpawnDoor` for a door the client already holds.** Walking through the door and back (a single clean re-spawn) fixes all of them, confirming it.

## 2. How Cosmic actually does it (verified in source)

`server/maps/DoorObject.java` and `MapleMap.spawnDoor`:

| Packet | Recipients |
|--------|------------|
| `spawnDoor` (area door) | **everyone in the map** (`MapleMap.spawnDoor` filters only on `mapId == from`) |
| `spawnPortal` (minimap indicator) | **everyone in the map** |
| `partyPortal` (the `0x25` town-portal array) | **owner + party members only** |

- **Entry** (`DoorObject.warp`): `owner || party.getMemberById(ownerId) != null` → play sound + `changeMap`; otherwise `blockedMessage(6)` (the pink "members only" text) + `enableActions`.
- **Party join/leave** (`Character.java` `updatePartyDoors`): walks `party.getDoors()` and updates each door's **town door / `partyPortal`** only. **The area `spawnDoor` is never re-sent.**
- **Mystic Door is a buff** (`StatEffect.isMysticDoor()`, duration `getDuration()`): the caster gets the buff icon counting down, and cancel/expiry runs `cancelMagicDoor()`.

The area door is a **plain ranged map object** — spawned once to everyone, like a monster. Party membership only gates **entry** and the **town-portal array**.

## 3. atlas's deviations

1. Area `spawnDoor`/`spawnPortal` are **party-filtered** (`broadcastDoorToEligible` → `partyMemberSet`; map late-join → `doorPartyMemberSet`). → should be **everyone in the map**.
2. The area door is **re-sent on party changes** by `ReconcileParty` (CREATED/REMOVED with `forCharacterId`). → Cosmic never re-sends it; this is the toggling bug.
3. **No entry gate** — with the door now visible to all, non-party players could *use* it. → must add `blockedMessage(6)`.
4. **No Mystic Door buff** (duration icon + cancellation). → feature gap.

Crucially, atlas **already** has the correct party/town rendering: the channel party-status consumer (`kafka/consumer/party/consumer.go`) rebuilds the PARTYDATA `aTownPortal` array via `toPartyMembers`/`applyMemberDoor` on every Created/Left/Expel/Disband/ChangeLeader and sends it in `PartyJoin/Left/Expel`. That is Cosmic's `partyPortal` path, party-gated, and it does **not** touch the area door. So the door reconcile's party handling is fully **redundant** with this — and is the only thing re-sending the area door.

## 4. The rework

### 4.1 Area door → plain map object (the bug fix)
- **`handleCreated`** (channel door consumer): broadcast `SpawnDoor` + area `SpawnPortal` + town `SpawnPortal` to **all sessions in the map**, not the party-eligible set. (`partyPortal` via `announceTownPortalToParty` stays party-gated — it does not toggle the area door.)
- **`handleRemoved`**: `RemoveDoor` + `RemoveTownDoor` to **all sessions in the map**; `partyPortal` clear to party.
- **Map late-join** (`map/consumer.go` `spawnDoorsForSession`/`spawnTownDoorsForSession`): drop the `doorPartyMemberSet`/`townSpawnPartyMembers` party filter — spawn the door to **every** session entering the map.

### 4.2 Stop re-sending the area door on party changes (delete the reconcile churn)
- The atlas-doors party consumer handlers (`handleJoined/Left/Expel/Disband/ChangeLeader`) no longer call `ReconcileParty`. The party/town rendering is owned by the channel party-status consumer (PARTYDATA rebuild). Remove `ReconcileParty` and its helpers, and the door processor's `Reslot`/party-scope churn used only by it.
- The door's `partyId`/`slot` become vestigial for rendering (the party flow derives a member's slot from their index in the live member list). Keep the fields for now (cast still records them); a later cleanup may drop them. The door registry's by-owner/by-field indices and the `door_status` CREATED/REMOVED lifecycle (cast / expiry / recast / leave-field) are unchanged.

### 4.3 Entry gate (required by 4.1)
- In the door-enter handler (`socket/handler/use_door.go` / `mystic_door_enter.go`): allow the warp only when `enterer == owner` or `enterer` shares a party with the owner; otherwise send the v83 `BLOCKED_MAP` packet with type `6` (Cosmic `blockedMessage(6)`) + re-enable actions, and do **not** warp. (Confirm/define the `BLOCKED_MAP` writer + per-version opcode during implementation.)

### 4.4 Mystic Door buff (follow-up, feature parity)
- Apply a buff to the caster with the door's duration so the client shows the countdown icon and can cancel the door (which then removes it). Separate piece; not required to fix the rendering bugs.

## 5. Why this fixes everything
With the area door spawned exactly once per viewer and **never** re-sent on party changes, the v83 client never gets a second `SpawnDoor` to toggle → no below-platform render, no broken owner door on expel, no null `pLayerFrame`, no expiry crash. The party/town rendering continues to work through the already-correct PARTYDATA path. Net change is mostly **removal** of the reconcile complexity.

## 6. Out of scope / sequencing
1. Revert the snap commit (`cb2e5e37c`) — wrong theory (the cast Y was always correct).
2. §4.1 + §4.2 (the bug fix) + §4.3 (entry gate, required by 4.1).
3. §4.4 buff — follow-up.

## 7. Testing
- Channel door consumer: `handleCreated`/`handleRemoved` broadcast to **all** in-map sessions (assert recipients = full map set, not party subset); `announceTownPortalToParty` still party-only.
- Map late-join: a non-party session entering the map receives the `SpawnDoor`.
- atlas-doors: party events no longer emit door_status CREATED/REMOVED (assert the consumer emits nothing on join/leave/expel/disband); cast/expiry/recast/leave-field lifecycle unchanged.
- Entry gate: owner/party → warp invoked; non-party → `BLOCKED_MAP(6)` sent, no warp.
- Standard gate: `go test -race`/`vet`/`build` for atlas-doors + atlas-channel; `docker buildx bake` both; redis-key-guard.
- Manual on `atlas-pr-769`: full party churn + expiry — no below-platform render, no crash; non-party member sees the door, gets the pink message on entry.
