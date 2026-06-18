# Mystic Door — Party-State Reconciliation Redesign

- **Task:** task-093-mystic-door (follow-up fix on the existing branch)
- **Date:** 2026-06-18
- **Status:** Design — approved direction, pending spec review
- **Supersedes:** the per-event delta handlers added across commits `10d5036d2` (surgical town-portal update), `00165e7f5` (leave→solo re-key), `82c2fd591` (rejoin adopt), `429164618`/`c9c340733` (recast suppression), `b25dc7dfd` (expel/disband teardown). Those remain as reference for the behaviors the reconciler must reproduce.

## 1. Problem

Reported reproduction (two characters, same field map `240011000`, town `240000000`):

| Step | Symptom |
|------|---------|
| Bishop casts, Chronicle casts, Chronicle expels Bishop | works |
| Chronicle reinvites Bishop | Bishop's own door **recasts to a platform below**; otherwise OK |
| Chronicle (leader) leaves party | **Chronicle's door remains** on Bishop's client; **Bishop's door remains** on Chronicle's client |
| Chronicle recreates party, Bishop joins | both stale doors persist |
| any later party-update tick (no recast) | **both clients crash** |

### 1.1 Ground-truth evidence (Loki, `atlas-pr-769`, run 10:58–11:01)

Actors decoded from `door_action` logs: **char 5 = Bishop** (casts first, expelled), **char 1 = Chronicle** (leader, slot 0).

- `10:58:05 spawn owner=5 area_x=688 area_y=-478` … door `1000090`.
- The door is re-keyed across every transition (`leave_party` → `join_party_rekey` → `disband_party` → `join_party_rekey`), all via `Clone` — **`area_y` never changes in the data**. So "platform below" is a *client render artifact*, not data drift.
- `10:59:36 disband_party areaDoor=1000090 (owner 5) party=1000000008 members=1` — when Chronicle (leader) left, `DisbandPartyDoors` received a member list of **only `[5]`**; Chronicle's own door (owner 1) was in no list and was never re-keyed/removed → **orphaned, tagged to dead party `1000000008`**.
- `11:01:07 TownPortal: unable to resolve party [1000000008] for door owner` — confirms the orphan: the channel can no longer resolve the dead party to broadcast a removal, so the door lingers on the other client.

### 1.2 Two root causes + one secondary defect

1. **Incomplete disband member list (immediate bug).** `services/atlas-parties/atlas.com/parties/party/processor.go:429` removes the departing leader (`RemoveMember`) *before* `:462` reads `party.Members()` for the DISBAND event body — so the leader is excluded. `atlas-doors` `handleDisband` (`kafka/consumer/party/consumer.go:165`) trusts that list, and its own comment wrongly assumes it is "the full former member list."

2. **Delta-application architecture (systemic).** Door↔party association is maintained by hand-coded per-event-type deltas (`JoinPartyDoor`, `LeavePartyDoor`, `DisbandPartyDoors`, `ShowPartyDoorsToCharacter`, `HidePartyDoorsFromCharacter`, `ReslotParty`), each driven by a partial member list. This is incomplete (anyone not in the list is skipped), order-dependent, and **non-convergent** — errors accumulate (orphaned doors) and never self-heal. Six fix commits each closed one transition and exposed the next; disband-leader is simply the next gap.

3. **Redundant area re-send (the "platform below" flicker).** Every party re-key broadcasts a full `CREATED` with `forCharacterId=0`, which the channel turns into an area `SpawnDoor` to **all** eligible viewers — including the owner, who already renders the door. Verified at `10:58:58`: `Broadcast [SpawnDoor] … to [2] session(s)` after a re-key, re-spawning Bishop's own door. The owner never needs an area packet for a party-scope change.

## 2. Verified client behavior (v95 IDB, `GMS_v95.0_U_DEVM.exe`, port 13340)

A Mystic Door reaches the client over **two render surfaces**, and the town surface has **two mutually-exclusive paths gated by the client's own party state**.

| Surface | Path | Packet (channel writer) | Recipients |
|---------|------|--------------------------|------------|
| Field/area door | single | `SpawnDoor(owner)` + area `SpawnPortal` | owner + party co-members in the field |
| Town door — **solo** | only when *recipient not in a party* | `SpawnPortal(town→area)` / `RemoveTownDoor` | the owner only |
| Town door — **party** | only when *recipient in a party* | `PARTY_OPERATION` town-portal slot update | all party members |

Evidence:

- **Mode = party membership.** `CWvsContext::OnPartyResult` @ `0xa10ab0`, tail `LABEL_142`, calls `CField::OnTownPortalChanged(get_field(), bParty, &m_party, &m_townPortal, …)` with `bParty = (m_nPartyID != 0)`, after **every** party operation.
- **Party town-render is an array-driven reconciler.** `CField::OnTownPortalChanged` @ `0x5381f0`, `bParty` branch, loops slots `i = 0..5` over PARTYDATA `aTownPortal[i]`: renders slot `i` iff `adwCharacterID[i] != 0 && adwFieldID[i] != 999999999 && aTownPortal[i].m_dwTownID == currentField`; **otherwise tears down that slot's existing layers**. The client renders exactly what the server places in the per-slot array and clears a slot only when the server makes it inactive/different-town.
- **The array is written two ways:** a per-slot update (`OnPartyResult case 46`) and full `PARTYDATA::Decode` (full snapshots — `case 7/8/12/15/38`). Both the surgical update and any full party-update tick rewrite it.
- **Hard client-kill guard.** `OnPartyResult case 46`: `slot = Decode1(); if (slot > 5) throw CDisconnectException; TOWNPORTAL::Set(&m_party.aTownPortal[slot], …)`. A town-portal slot ≥ 6 disconnects the client by design. The **operation code is version-shifted: v95 = `46`, v83 = `0x25`** — consistent with the known per-version party operations-table reshift; the channel must emit the per-version code (it already does, via `PartyOperationWriter`).
- **Solo shares slot 0.** With `bParty = 0` the same function renders the single `m_townPortal` into `m_apLayerTownPortal[0]` and the tail loop clears slots 1–5. A party→solo flip self-clears slots 1–5 *when a solo town-portal change is processed*; slot 0 is shared and is the server's responsibility.

**Design consequences (now first-class):**

- The party town-portal array is **authoritative client state**. The server must reconcile it as a bounded `[0..5]` vector: *set* occupied slots and *explicitly clear* every vacated slot. A door left tagged to a dead party = a populated slot = a ghost.
- Every emitted town-portal slot must be provably **≤ 5**; a `> 5` send is a guaranteed disconnect.

## 3. Design — reconciliation, not deltas

### 3.1 Core principle

A door has two independent facets the current code conflates:

1. **Area-door lifecycle** — the physical door + portal in the field. Created on cast; destroyed on recast / expiry / leave-field. **Party membership changes never touch this.**
2. **Town/party projection** — town-minimap portal slot, party town-portal array entry, and *which characters may see the area door*. This is a **pure function of `(owner, current authoritative party membership)`**.

Party transitions recompute facet 2 only, as a reconciliation: compute desired → diff against current → emit the minimal targeted deltas, using the **existing** `CREATED` / `REMOVED` / `SLOT_CHANGED` + `forCharacterId` status-event vocabulary. **atlas-channel and the Kafka event schema are unchanged.**

### 3.2 The reconciler

One function replaces the five party methods and `ReslotParty`:

```
ReconcileParty(ctx, partyId, members []character.Id)
```

`members` is the authoritative post-change ordered member list (leader at index 0), fetched from atlas-parties for join/left/expel/change-leader, or carried on the (now-complete) DISBAND event body when the party is gone (`members` empty ⇒ everyone drops to solo).

Algorithm:

1. **Candidate owners** = `members` ∪ `{ leaver/expellee/former-member ids from the event }`. A door's `partyId` is only ever set to a party its owner belonged to, so this set is exactly the owners whose door could still be tagged `partyId` — no "doors by partyId" scan/index is needed. Each candidate's door(s) are loaded via `GetByOwner`; see §3.5 for the leaver source per event type.
2. For each candidate's door(s), compute **desired scope**:
   - owner ∈ `members` → `partyId`, `slot = ComputeSlot(members, owner)` (≤ 5, asserted), town portal resolved for that slot.
   - owner ∉ `members` → solo: `partyId 0`, `slot 0`, solo town portal.
3. Compute **desired viewer set** per door = owner + current co-members. Compare to the **previous** viewer set implied by the door's stored `partyId`.
4. Emit minimal deltas:
   - **Scope change** (party/slot/portal differs): persist the re-keyed door, emit `SLOT_CHANGED` — **town-portal move only, never an area packet**.
   - **Viewer gained** (a character who could not see this door now can): targeted `CREATED` (`forCharacterId = newViewer`). This is the **only** path that sends an area `SpawnDoor` on a party change, and only to someone who lacked it.
   - **Viewer lost**: targeted `REMOVED` (`forCharacterId = lostViewer`).
   - **No change**: emit nothing (kills the flicker).
5. The party town-portal array is reconciled as a bounded vector: for `slot 0..len(members)-1` set the occupant's entry; for `slot len(members)..5` (and any slot a departed member vacated) emit an explicit clear. Never emit a slot > 5.

The function is **idempotent** (a second call emits nothing) and **convergent** (a door tagged to an absent party always resolves to solo, so orphans self-correct).

### 3.3 How each symptom is fixed

- **Orphaned "Chronicle's door remains":** on disband, Chronicle ∈ candidate owners (he owns a door tagged `partyId`) and ∉ `members` (empty) → his door reconciles to solo, is removed from Bishop (viewer lost), and its party-array slot is cleared. Requires §3.5.
- **"Platform below" flicker:** the owner is never a *viewer gained* on a party change, so he never receives a redundant area `SpawnDoor`.
- **Both-clients-crash:** no door is ever left tagged to a dead party, and the town-portal array is always reconciled with bounded, cleared slots — removing every stale-array precondition. (See §6 for the honest scope note on the exact trigger.)

### 3.4 Components changed

- **atlas-doors `door` package:** add `ReconcileParty` (+ a small `desiredDoorState` / diff helper). Remove `JoinPartyDoor`, `LeavePartyDoor`, `DisbandPartyDoors`, `ShowPartyDoorsToCharacter`, `HidePartyDoorsFromCharacter`, `ReslotParty`. Keep `Spawn`, `RemoveByOwner`, `RemoveByOwnerIfLeftField`, `Reslot` (the area-door lifecycle + the low-level reslot primitive the reconciler calls).
- **atlas-doors `kafka/consumer/party`:** the five handlers (`handleJoined/Left/Expel/Disband/ChangeLeader`) collapse to: resolve authoritative `members` → call `ReconcileParty`. Disband passes the event body's (now-complete) member list.
- **atlas-parties `party/processor.go`:** capture the full member list **before** `RemoveMember` (`:429`) and emit it in the DISBAND body (`:462`). This is the one cross-service change.
- **atlas-channel:** unchanged. The status-event handlers (`CREATED` / `REMOVED` / `SLOT_CHANGED` with `forCharacterId`) and `PartyOperationWriter` already express every delta the reconciler needs, and the slot-≥6 guard at `kafka/consumer/door/consumer.go:139` stays as defence-in-depth.

### 3.5 Sourcing the leaver(s)

The reconciler must reach a door whose owner just left the party. Two cases:

- **Join / left / expel / change-leader:** the party still exists; `members` is fetched from atlas-parties. The leaver/expellee id is on the event (`ActorId` / `Body.CharacterId`); include it in the candidate set so its door is reconciled to solo even though it is no longer in `members`.
- **Disband:** the party is gone; the **DISBAND event body must carry the complete former member list** (the atlas-parties fix). Every former member becomes a candidate; `members` is empty so all reconcile to solo.

No new Redis index is required — candidates are reached via `GetByOwner` over the known member + leaver ids. (A "doors by partyId" index is intentionally *out of scope*; see §5.)

## 4. Data flow — reported scenario under the new design

1. **Bishop casts, Chronicle casts:** unchanged area-door spawns; each `ReconcileParty(p, [1,5])` sets slots 0/1 and party-array entries.
2. **Chronicle expels Bishop:** `ReconcileParty(p=1000000008, members=[1])`, candidates `{1,5}`. Bishop(5) ∉ members → solo (door removed from Chronicle, array slot 1 cleared, solo `SpawnPortal` to Bishop). Chronicle(1) unchanged. Bishop's own area door is *not* re-sent.
3. **Chronicle reinvites Bishop:** `ReconcileParty(p, members=[1,5])`, candidates `{1,5}`. Bishop(5) → party slot 1 (door newly visible to Chronicle = targeted `CREATED` to Chronicle only; Bishop gets a `SLOT_CHANGED` + array entry, **no area re-send** → no "platform below"). Chronicle unchanged.
4. **Chronicle (leader) leaves → disband:** DISBAND body now `[1,5]`. `ReconcileParty(p=…008, members=[])`, candidates `{1,5}`. Both → solo: each door removed from the *other* member, array slots cleared, solo `SpawnPortal` to each owner. No orphan.
5. **Chronicle recreates party, Bishop joins:** `ReconcileParty(p=1000000009, members=[1,5])`. Both doors (currently solo) → the new party, correct slots, array set. Convergent.
6. **Any later tick:** reconcile is idempotent; the array holds only valid, bounded, occupied slots → no crash precondition.

## 5. Out of scope

- Area-door spawn / recast / expiry / leave-field logic (unchanged).
- Kafka event schema and atlas-channel handlers (unchanged).
- Redis registry schema; no "doors by partyId" index (candidates reached via member+leaver `GetByOwner`).
- Migrating the party packet **dispatcher family** to discrete-per-mode (`docs/packets/DISPATCHER_FAMILY.md`). We rely on the existing `PartyOperationWriter` for the version-resolved town-portal op code; the family migration is a separate task.

## 6. Error handling, edge cases, honesty

- **Idempotency / dropped or reordered events:** every reconcile recomputes from authoritative state and diffs, so a duplicate or out-of-order party event converges rather than corrupts.
- **Orphan self-heal:** any door whose `partyId` names a party not returned by atlas-parties reconciles to solo on the next reconcile touching that owner.
- **Slot bound:** `ComputeSlot` caps at `maxPartySize-1 = 5`; the reconciler asserts `slot ≤ 5` before emitting and the channel guard rejects `≥ 6`. Belt and suspenders against the verified `CDisconnectException`.
- **Party resolve failure** (atlas-parties REST error during reconcile): treat as "membership unknown," skip the destructive half (do not mass-drop to solo on a transient error); log and rely on the next event. Reslot/visibility errors are per-door and logged, never fatal.
- **Honest crash note:** static analysis confirms one hard crash vector (town-portal slot > 5 → `CDisconnectException`) and the ghost mechanism (dead-party door cannot be resolved to the co-member, so its `RemoveDoor` never arrives). With 2-member parties the slot stays 0/1, so slot > 5 is *not* obviously the exact trigger of the reported "tick" crash; pinning it precisely needs a client exception log or a packet capture. The redesign does not depend on pinning it — it removes every orphaned/stale-array state, the common precondition of all crash hypotheses.

## 7. Testing

- **`ReconcileParty` table tests** over the in-memory registry seam (already used by the door processor tests): for cast, expel, reinvite, leader-leave-disband, recreate+join, and the **full reported scenario end-to-end**, assert (a) the desired door records (party/slot/portal) and (b) the exact emitted status-event set (type, `forCharacterId`, slot).
- **Idempotency:** a second `ReconcileParty` with identical membership emits zero events.
- **Orphan self-heal:** seed a door tagged to an absent party → reconcile resolves it to solo and clears its array slot.
- **Flicker regression:** assert **no** area `SpawnDoor` / owner-targeted `CREATED` on a pure party-scope change (the "platform below" guard).
- **Slot bound:** assert no emitted town-portal slot exceeds 5 for any membership permutation.
- Delete tests for the five removed methods; retarget any still-valid assertions onto `ReconcileParty`.
- **atlas-parties:** unit-assert the DISBAND event body carries the full former member list including the departing leader.
- Standard gate: `go test -race ./...`, `go vet ./...`, `go build ./...`, `docker buildx bake atlas-doors atlas-parties`, `tools/redis-key-guard.sh`.
