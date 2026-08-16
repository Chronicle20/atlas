# The dried-up state on the wire — findings and fixes

Follow-up to a live test on `atlas-pr-1360` (v83 tenant). The tester backdated
`atlas-pets.pets.expiration` to simulate a dried-up pet. The pet stayed
summonable, and clicking it froze the client. Backdating the matching
`atlas-inventory.assets.expiration` row changed nothing.

Both observations were correct, and both are explained by one defect: **Atlas
had no way to express "this pet has dried up" on the wire.**

## 1. What the client actually reads

`GW_ItemSlotPet::dateDead` — the 8-byte FILETIME at struct offset +89 — and
nothing else. It is NOT `dateExpire` (the cash item's own lifetime, which is
what `atlas-inventory.assets.expiration` feeds), so editing the inventory row
was never going to matter. `dateDead` is not stored anywhere either: it is
derived at send time from `atlas-pets.pets.expiration`
(`character/processor.go:140`, `kafka/consumer/asset/consumer.go:118` →
`SetPetDeadDate(pm.Expiration())`).

The predicate is a **threshold, not a clock**. GMS v95 (symbols present):

```
GW_ItemSlotPet::IsDead      @0x4f6c30  -> IsLimitedLifePet ? nRemainLife <= 0
                                                           : IsDeadByDate()
GW_ItemSlotPet::IsDeadByDate@0x4f1be0  -> CompareFileTime(dateDead,
                                             DB_DATE_20790101) >= 0
```

`DB_DATE_20790101` = 150842304000000000 (2079-01-01). GMS v83 is the same code
unnamed: `sub_4E4003` → `sub_4E4044`, comparing against `dword_AF30B0`, whose
value reads back as exactly 150842304000000000. `CompareFileTime` is unsigned.

So a pet is dried up **iff `dateDead >= 2079-01-01`**. The client never compares
`dateDead` against the current time. An elapsed date — yesterday, 2026 — encodes
to ≈1.34e17, below the threshold, and the client reads the pet as alive and
renders SP_679 "Water of life dries up on \<yesterday\>".

`encodePetDeadDate` (`libs/atlas-packet/model/asset.go`) sent exactly that:
`MsTime(expiration)`. Nothing in Atlas ever emitted a value on the dead side of
the threshold, so the dried-up state was unreachable.

### Why this looked like it used to work

Before task-139 (`0ad029a28`, PR #1253) the field was written as
`MsTime(m.expiration)` — sourced from the item expiration, and `MsTime` returns
`-1` for a zero time, which as an unsigned FILETIME is the maximum and sits
ABOVE the threshold. Every pet with an unset expiration therefore rendered as
dried up. Task-139 fixed that false positive (zero → 0) but never added the true
positive, trading a false "dried up" for a permanent "alive".

## 2. What the client does with a dried-up pet

`CWvsContext::SendActivatePetRequest` (v95 @0x9f6980 named; v83 @0xa240a2
identical) is ONE function with three outcomes on double-click:

| Pet state | Client behavior |
|---|---|
| alive | sends SPAWN_PET — tick + slot + bBossPet — and latches the exclusive-request lock |
| dried up, revivable | `CUtilDlg::Notice` SP_378 "The time has run out so it can't move." **No packet, no lock.** |
| dried up, WZ `noRevive` | sends DESTROY_PET_ITEM_REQUEST — tick + 8-byte `liCashItemSN` — latches the lock, then offers the Cash Shop (SP_379) |

Two consequences for this branch:

- The Water of Life case is the **middle** row. Once the dried-up state reaches
  the client, a dried-up revivable pet is not summonable and cannot hang —
  the client refuses locally. design.md §3 (D2) assumed "the client has no
  concept" of expiry; it does, and this is where the gate lives.
- The freeze the tester hit was the **first** row: the client thought the pet was
  alive, sent SPAWN_PET, atlas-pets rejected with `ErrPetExpired`, the
  transactional emit discarded the buffer so no status event was produced, and
  nothing ever released the lock.

## 3. Changes

### FIX-1 — encode the dried-up state (`libs/atlas-packet/model/asset.go`)

`encodePetDeadDate` now maps an already-elapsed dead date to the new
`PetDriedUpFileTime` constant (150842304000000000) instead of its own `MsTime`.
Zero (permanent) still encodes 0; a future date still encodes its own `MsTime`.
Encoding is therefore time-dependent — inherent, because the wire value is a
state and which state a stored timestamp denotes changes as the clock passes it.

`TestAssetPetCashItemDeadDate` gains the elapsed case and switches its dates to
now-relative ones; the old literal (2026-11-06) would have silently changed
meaning the day it elapsed.

### FIX-2 — implement DESTROY_PET_ITEM_REQUEST

The op was in every registry and in the matrix, but had **no codec, no handler
and no template route** on any version. New codec
`libs/atlas-packet/pet/serverbound/destroy_item.go` (`DestroyItem`): tick u32 +
`liCashItemSN` u64, no version gate. Body confirmed identical at every send site
that has the op:

| Version | Send site | Opcode | Destroy arm |
|---|---|---|---|
| gms_v83 | 0xa240a2 (unnamed) | 0x50 | `COutPacket(0x50)` Encode4 EncodeBuffer(8) |
| gms_v84 | unnamed, not in export | 0x50 | registry (discover-ops); body = v83 |
| gms_v87 | 0xabbb70 | 0x53 | Encode4@0xabbc7f EncodeBuffer@0xabbc90 |
| gms_v92 | 0x9cb540 | 0x57 | COutPacket@0x9cb6db Encode4@0x9cb6f2 EncodeBuffer@0x9cb701 |
| gms_v95 | 0x9f6980 | 0x56 | `EncodeBuffer(&v9->liCashItemSN, 8)` |
| jms_v185 | 0xb0b40b | 0x48 | COutPacket@0xb0b50f Encode4@0xb0b524 EncodeBuffer@0xb0b532 |

v48/v61/v72/v79 have no opcode for it (registry: `n-a`) and are not routed.

Handler `PetDestroyItemHandleFunc` resolves the pet by the serial the client
sent (with the same pet-id fallback `model.Asset.PetSerialNumber` uses),
re-checks that it really has dried up, and runs a one-step
`InventoryTransaction` saga with `DestroyAssetFromSlot` — by slot, because a
character can hold several pets of one template and only the clicked one is
dead. Deleting the asset is the whole cascade: atlas-pets already drops the pet
record on the asset-deleted event.

**Matrix note.** `DESTROY_PET_ITEM_REQUEST` was reading ✅ on v87/v95/jms
against `pet/serverbound/PetSpawn` — a false verify. The op shares its FName
with SPAWN_PET, and `findReport` resolves reports by FName alone, so the second
op silently inherited the first op's report and fixture. `findReport` now
rejects a report whose packet id contradicts an explicit registry `packet:`
declaration (guarded by `isPacketID`, since some entries — gms_v79
MTS_OPERATION — carry prose there instead). The six registries now declare
`packet: pet/serverbound/PetDestroyItem`, and evidence is pinned for v87, v92,
v95 and jms_v185; v92 promotes from ❌.

**Left open:** gms_v83 and gms_v84 cells stay ❌. Their send site is unnamed in
both IDBs, so it is absent from the checked-in export and `evidence pin` cannot
resolve a citation. The bytes are pinned in the test for both versions;
promoting the cells needs the function named and the column's export
re-harvested (`RE_AUDITING_A_COLUMN.md`). The STATUS.md row still *labels*
itself `pet/serverbound/PetSpawn` — cosmetic only (the label comes from the
FName-matched report; the cells grade against the declared packet). Making the
label follow the registry declaration relabels 12 unrelated rows, so it was left
alone.

### FIX-3 — release the lock on every spawn bail-out

`PetSpawnHandleFunc` discarded its result and returned silently on four paths
(character load failure, empty slot, non-pet item, and now dried-up), each of
which leaves the client latched. All four now call `session.EnableActions`. The
dried-up gate is belt-and-braces: an up-to-date client takes the IsDead arm and
never sends SPAWN_PET for a doll, but a client whose item block predates the
expiry still can.

## 4. Reproducing a dried-up pet, after these changes

Set `atlas-pets.pets.expiration` to a past timestamp. That is the whole
procedure — the inventory row is unrelated, and `dateDead` is derived. Relog (or
change channel) so the client re-reads the inventory; the tooltip then reads
"The water of life has dried up" and double-clicking gives SP_378 rather than a
freeze.
