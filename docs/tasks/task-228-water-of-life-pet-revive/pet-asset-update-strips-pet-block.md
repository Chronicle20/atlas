# Revived pet: commands dead, despawn closes the client — diagnosis

Reported 2026-08-16 on `atlas-pr-1360` (GMS v83, tenant `14e6eaee-…`, character 1,
pet 1, cash SN `1437245515603478011`). Water of Life revive succeeded; pet
commands did nothing; despawning the pet closed the game.

## Root cause: `handleAssetUpdatedEvent` sends an unenriched pet asset

`services/atlas-channel/.../kafka/consumer/asset/consumer.go:270` builds the
asset from the Kafka body and immediately writes an `InventoryChange` **add**
entry for the slot:

```go
a := buildAssetFromUpdatedBody(e)          // no enrichPetAsset
so := session.Announce(...)(invcb.InventoryChangeWriter)(
        invcb.NewChangeBatch(false, invpkt.NewAddEntry(byte(inventoryType), a.Slot(),
                model2.NewAsset(true, a))).Encode)
```

It is the only asset-add path in that file that skips `enrichPetAsset`
(`handleAssetCreatedEvent` calls it at :259, `handleAssetAcceptedEvent` at :450).
`buildAssetFromUpdatedBody` never sets any pet field — the asset status event
carries none — so for a pet asset the encoder
(`libs/atlas-packet/model/asset.go` → `encodePetCashItemInfo`) writes a
`GW_ItemSlotPet` block of:

| field | sent | should be |
|---|---|---|
| SN (`PetSerialNumber`) | **1** (falls back to `petId`, `asset.go:431-436`) | `1437245515603478011` |
| name | `""` (13-byte pad) | `Pet` |
| level / closeness / fullness | 0 / 0 / 0 | 1 / 0 / 100 |
| dateDead | 0 | 2026-11-14 |

The block is the right *length*, so the packet does not desync — the client
accepts it and silently replaces slot 1's pet record with one whose serial is 1.

## Why this branch triggers it

`ResetPetExpiration` → `assetProcessor.ExtendExpiration`
(`services/atlas-inventory/.../compartment/processor.go:2224`) is new on
task-228, and it is what emits an asset `UPDATED` for a *pet* asset. Nothing
before this branch updated a pet asset in place, so the latent hole in
`handleAssetUpdatedEvent` had no trigger.

## Evidence chain (atlas-channel `pr-1360-668f115`, 11:16–11:18)

```
11:16:48.136  [WaterOfLifeHandle] read []
11:16:48.157  Character [1] reviving pet [1] with Water of Life [5180000] (life [90] days).
11:16:48.626  pet REVIVED  expiration 2026-11-14T11:16:48Z
11:16:48.694  saga e3f5b4c4 COMPLETED (pet_revive)
11:16:48.741  asset UPDATED assetId 14 templateId 5000012 slot 1 petId 1   <-- corrupting packet
11:16:52.055  pet SPAWNED  slot 0  cashId 1437245515603478011
11:16:59–11:17:00  [PetMovementHandle] petId [1437245515603478011]  x4
11:17:16.244  [PetSpawnHandle] slot [1]  -> despawn
11:17:16.383  pet DESPAWNED oldSlot 0 reason NORMAL
11:17:18.156  Read a unhandled message with op 0xDF   <-- PARTY_SEARCH_UPDATE
11:17:18.159  Connection ended.
```

Ruled out along the way:

- **Not the despawn codec.** `PetActivated`/`PetDespawnBody` is untouched by this
  branch, `operations.NORMAL` resolved fine (no "Defaulting to 99" line, no
  "Unable to write" line in the whole 8-minute log).
- **Not a routing gap.** `PetCommandHandle` is at `0xA9`; the client never sent
  `0xA9` (no handler line and no `unhandled message with op 0xA9`) — consistent
  with the client being unable to resolve its own pet item, not with a dropped
  packet. `0xCF`/`0xDF` are `PLAYER_MAP_TRANSFER` and `PARTY_SEARCH_UPDATE`
  (registry `gms_v83.yaml`), both benign and both normally unrouted.
- **Not stale atlas-data.** The re-ingest at 11:00 fixed the earlier `life`
  problem; the revive itself completed.

The client-side consequences (command parse and despawn both resolving the pet
item by SN) are inference from the corrupted SN, not from a decompile; the
corrupted wire value itself is confirmed from the code path above.

## Fix (applied)

`handleAssetUpdatedEvent` now calls `enrichPetAsset`, exactly as the created and
accepted handlers do. `enrichPetAsset` was split so the atlas-pets lookup is
injectable (`enrichPetAssetWith`), which is what makes the enrichment testable
without REST.

Three tests in `kafka/consumer/asset/consumer_pet_enrichment_test.go`:

- `TestAssetUpdatedEventEnrichesPetBlock` — replays the captured `UPDATED` event
  above and asserts the encoded asset carries the cash serial, name, level,
  closeness, fullness and dead date, and that the unenriched one does not.
- `TestEnrichPetAssetLeavesNonPetsAlone` — enrichment stays inert for the
  ordinary items that dominate the `UPDATED` stream.
- `TestEveryAddEntryEnrichesPets` — the guard that would have caught this. It
  walks `consumer.go`'s AST and fails any function building an `InventoryChange`
  add entry without an `enrichPetAsset` call. Verified to fail on the pre-fix
  source (`handleAssetUpdatedEvent builds an InventoryChange add entry without
  enrichPetAsset`).

The sweep of the other emitters is clean: the only `NewAddEntry` sites are the
three handlers here plus `kafka/consumer/pet/consumer.go:269`, which enriches via
`PetAssetEnrichmentDecorator` — and whose comment describes this exact failure
mode, so it is the second time this hole has been hit. Every other
`InventoryChangeWriter` call is a quantity/move/remove entry that does not encode
the asset block.
