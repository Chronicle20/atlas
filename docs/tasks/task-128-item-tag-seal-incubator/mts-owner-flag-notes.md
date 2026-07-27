# Task 10 — MTS listing snapshot capture site (owner/flag)

Investigation only, read-only trace. No Go code changed.

## Answer (short form)

- **Capture site**: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:1600`, inside `expandTransferToMts` (function starts `:1521`), in the construction of the `AcceptToMtsListingPayload` literal (`:1570`–`:1614`).
  ```go
  Flags:         foundAsset.Flag,   // processor.go:1600
  ```
- **Flags-carry answer**: YES. `flags` on the listing IS sourced from the seller's equip asset's flag bitfield at this exact site. `foundAsset` is a `compartment.AssetRestModel` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/rest.go:20`, `Flag uint16 \`json:"flag"\``) fetched live from the character's inventory compartment during saga expansion (`compartment.RequestCompartment`, `processor.go:1530`). `asset.FlagLock = 0x01` (`libs/atlas-constants/asset/flag.go:6`) is a bit within that same `uint16`, so the seal bit already carries end-to-end from the seller's live asset into the listing row's `flags` column (`listing/entity.go:77`) via `processor_custody.go:161` (`SetFlags(b.Flags)`). **The seal side is already correct; only `owner` needs adding, at the SAME site and the same downstream chain.**
- **Owner is currently NOT threaded anywhere in this path.** `compartment.AssetRestModel` already carries `Owner string` (`compartment/rest.go:19`, sibling of `Flag` on line 20) — the source data exists — but `expandTransferToMts` never copies `foundAsset.Owner` onto the payload, and no downstream struct in the chain (`AcceptToMtsListingPayload`, the custody command body on either side, `listing.AcceptRequest`, `listing.Builder`, `listing.Model`, `listing.Entity`, `listing/rest.go`) has an `owner` field today. Confirmed via `grep -n "Owner\|owner" listing/model.go listing/entity.go listing/builder.go listing/rest.go` — the only "owner" hits are `OfferWishOwnerId` (an unrelated want-ad field), not an item-tag owner.

## Full trace (seller lists item → listing row created)

1. **Seller-initiated saga step build** (in atlas-mts, when the seller calls `POST /worlds/{worldId}/listings`):
   `services/atlas-mts/atlas.com/mts/listing/processor.go:805` — `builder.AddStep("transfer_to_mts", saga.Pending, saga.TransferToMts, saga.TransferToMtsPayload{...})`. This payload carries only identity + sale params (`libs/atlas-saga/payloads.go:593-613`); it explicitly does NOT carry the item snapshot ("The item snapshot itself is looked up from inventory during expansion (NOT carried here)" — payloads.go:591-592).

2. **Saga expansion** (atlas-saga-orchestrator, `TransferToMts` → 2 steps):
   `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:1521` `expandTransferToMts`.
   - `:1530` looks up the seller's inventory compartment via `compartment.RequestCompartment`.
   - `:1536-1549` finds `foundAsset *compartment.AssetRestModel` by `AssetId` — **this is the seller's live asset, the ground truth for stats/flags/owner**.
   - `:1553-1616` builds two steps: `release_from_character` (removes the item from the seller's inventory) then `accept_to_mts_listing` carrying an `AcceptToMtsListingPayload` literal whose item-snapshot fields (`:1580-1600`) are copied straight off `foundAsset` — **`Flags: foundAsset.Flag` at `:1600` is the flags capture site**. `foundAsset.Owner` (available on the struct, `compartment/rest.go:19`) is never referenced here — that's the owner capture GAP.

3. **Payload struct definition** (shared lib):
   `libs/atlas-saga/payloads.go:629` `AcceptToMtsListingPayload` — the item-snapshot fields run `:639-662`; `Flags uint16` is the last one (`:662`). No `Owner` field exists on this struct.

4. **Saga step handler dispatches the atlas-mts command** (atlas-saga-orchestrator):
   `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:2019` `handleAcceptToMtsListing` reads the `AcceptToMtsListingPayload` off the step (`:2020`) and calls `h.mtsP.AcceptToMtsListingAndEmit(payload.TransactionId, mts.AcceptToMtsListingParams{...})` (`:2028-2068`), copying `Flags: payload.Flags` at `:2058`. No `Owner` passthrough (payload has none to pass).

5. **Params struct + producer** (atlas-saga-orchestrator, `mts` package):
   - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/mts/processor.go:21` `AcceptToMtsListingParams` — item-snapshot fields `:29-53`, `Flags uint16` last (`:53`). No `Owner`.
   - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/mts/producer.go:17-61` `AcceptToMtsListingProvider` builds the wire `mtsCustody.Command[mtsCustody.AcceptToMtsListingCommandBody]`, copying `Flags: params.Flags` at `:50`. No `Owner`.

6. **Wire command body (producer side)**:
   `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/mts/custody/kafka.go:42` `AcceptToMtsListingCommandBody` — mirrors the atlas-mts-side struct field-for-field (JSON over `COMMAND_TOPIC_MTS_CUSTODY`). No `Owner` field.

7. **Wire command body (consumer side, atlas-mts)**:
   `services/atlas-mts/atlas.com/mts/kafka/message/custody/kafka.go:61` `AcceptToMtsListingCommandBody` — item-snapshot fields `:69-95`, `Flags uint16 \`json:"flags"\`` last (`:95`). No `Owner` field. **These two command-body structs (steps 6 and 7) must stay in lockstep since there is no shared type between the two services — both need the new field added together.**

8. **Consumer handler maps the command body to the listing processor's request**:
   `services/atlas-mts/atlas.com/mts/kafka/consumer/custody/consumer.go:78` `handleAcceptToMtsListing`, body `:96-136` — maps `custody.AcceptToMtsListingCommandBody` fields onto `listing.AcceptRequest{...}`, `Flags: b.Flags` at `:126`. No `Owner`.

9. **AcceptRequest struct + the actual DB-row build (THE terminal capture into the listing entity)**:
   `services/atlas-mts/atlas.com/mts/listing/processor_custody.go:25` `AcceptRequest` — item-snapshot fields `:37-59`, `Flags uint16` last (`:59`). No `Owner`.
   `processor_custody.go:80` `Accept()` builds the row via `NewBuilder(...).SetFlags(b.Flags)...Build()` — the `SetFlags(b.Flags)` call is at `:161`. This is the terminal write into the `listing.Builder`, which is what ultimately produces the GORM entity (`CreateListing(tx, m)` at `:176`).

10. **Model / Builder / Entity / Provider / Administrator / REST** (all in `services/atlas-mts/atlas.com/mts/listing/`) each mirror `flags` and would need an `owner` sibling:
    - `builder.go`: field `flags uint16` (`:49`), `SetFlags` (`:224-227`), assembled into the model at `:336` (`flags: b.flags`).
    - `model.go`: field `flags uint16` (`:80`), getter `Flags()` (`:137`).
    - `entity.go`: GORM column `Flags uint16 \`gorm:"column:flags;not null"\`` (`:77`) — needs a migration for the new `owner` column.
    - `provider.go` (entity→model on read): `SetFlags(e.Flags)` (`:276`).
    - `administrator.go` (model→entity on create/update): `Flags: m.Flags()` (`:126`).
    - `rest.go` (model→REST DTO): `Flags uint16 \`json:"flags"\`` field (`:45`), populated `Flags: m.Flags()` (`:157`). This is the field the brief says already exists with no `Owner` — confirmed.

## Secondary capture sites that ALSO copy the snapshot (same `flags` pattern, same gap) — Tasks 11-14 must cover these too if item-tag owner must survive every custody hop, not just the initial list:

- **Cancel/Expire → seller holding**: `services/atlas-mts/atlas.com/mts/listing/processor.go:587` `transitionToSellerHolding`, builder call `:606-632`, `SetFlags(lm.Flags())` at `:631`. Mirrors the same `holding.Builder`/`holding.Model`/`holding.Entity`/`holding.Provider`/`holding.Administrator`/`holding.rest.go` chain (flags sites found at `holding/builder.go:190-191,234`, `holding/model.go:59,93`, `holding/entity.go:74`, `holding/provider.go:138`, `holding/administrator.go:107`, `holding/rest.go:41,113`) — every one of those needs an `owner` sibling too.
- **Buy settle → buyer holding**: `services/atlas-mts/atlas.com/mts/listing/processor_custody.go:227` `SettleMove`, builder call `:305-332`, `SetFlags(lm.Flags())` at `:331`. Same `holding.*` chain as above.
- **Saga compensation (re-grant item to seller on a failed TransferToMts)**: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go:1556` `assetDataFromMtsListingSnapshot(p AcceptToMtsListingPayload) asset2.AssetData` sets `Flag: p.Flags` (`:1578`) but not `Owner`/`OwnerId`, even though the target `asset2.AssetData` struct (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/asset/kafka.go:29`) already has both `OwnerId uint32` (`:33`) and `Owner string` (`:34`) fields sitting unused right next to `Flag` (`:35`). Once `AcceptToMtsListingPayload.Owner` exists, this reconstruction function should also set `Owner: p.Owner` so a compensated (failed-list) re-grant restores the item-tag owner, not just its stats.

## Owner-threading checklist for Tasks 11-14 (in dependency order)

1. `libs/atlas-saga/payloads.go` — add `Owner string` to `AcceptToMtsListingPayload` (near `:662`).
2. `services/atlas-saga-orchestrator/.../saga/processor.go:1600` — add `Owner: foundAsset.Owner,` to the `AcceptToMtsListingPayload{}` literal in `expandTransferToMts`.
3. `services/atlas-saga-orchestrator/.../saga/handler.go:2058` — add `Owner: payload.Owner,` to the `mts.AcceptToMtsListingParams{}` literal.
4. `services/atlas-saga-orchestrator/.../mts/processor.go:53` — add `Owner string` to `AcceptToMtsListingParams`.
5. `services/atlas-saga-orchestrator/.../mts/producer.go:50` — add `Owner: params.Owner,` to the `mtsCustody.AcceptToMtsListingCommandBody{}` literal.
6. `services/atlas-saga-orchestrator/.../kafka/message/mts/custody/kafka.go` — add `Owner string \`json:"owner"\`` to `AcceptToMtsListingCommandBody`.
7. `services/atlas-mts/.../kafka/message/custody/kafka.go:95` — add `Owner string \`json:"owner"\`` to the atlas-mts-side `AcceptToMtsListingCommandBody` (must match step 6's JSON key exactly).
8. `services/atlas-mts/.../kafka/consumer/custody/consumer.go:126` — add `Owner: b.Owner,` to the `listing.AcceptRequest{}` literal.
9. `services/atlas-mts/.../listing/processor_custody.go:59` — add `Owner string` to `AcceptRequest`; `:161` add `.SetOwner(b.Owner)` to the builder chain.
10. `services/atlas-mts/.../listing/builder.go` — add `owner string` field + `SetOwner` + assemble into model (mirror `:49/224/336`).
11. `services/atlas-mts/.../listing/model.go` — add `owner string` field + `Owner()` getter (mirror `:80/137`).
12. `services/atlas-mts/.../listing/entity.go:77` — add `Owner string \`gorm:"column:owner"\`` column + migration.
13. `services/atlas-mts/.../listing/provider.go:276` — add `.SetOwner(e.Owner)`.
14. `services/atlas-mts/.../listing/administrator.go:126` — add `Owner: m.Owner()`.
15. `services/atlas-mts/.../listing/rest.go:45,157` — add `Owner string \`json:"owner"\`` + populate `Owner: m.Owner()` (this is the field Phase C needs for the UI to render item-tag owner on marketplace listings).
16. If holding-stage owner fidelity is in scope: repeat the `holding.*` mirror (item 10-14 above) for `services/atlas-mts/.../holding/{builder,model,entity,provider,administrator,rest}.go`, plus wire `SetOwner(lm.Owner())` into `listing/processor.go:631` and `listing/processor_custody.go:331`.
17. If saga-compensation fidelity is in scope: `services/atlas-saga-orchestrator/.../saga/compensator.go:1578` — add `Owner: p.Owner,` (and `OwnerId` if available) to `assetDataFromMtsListingSnapshot`'s `asset2.AssetData{}` literal.

## Notes / things NOT verified here (out of scope for this investigation)

- Did not trace how `compartment.AssetRestModel.Owner` (`compartment/rest.go:19`) is itself populated upstream (the character-inventory/compartment service) — out of scope per the brief's file list (services/atlas-mts, services/atlas-saga-orchestrator, services/atlas-channel, libs/atlas-saga). The field exists and is already wired into the REST DTO the saga orchestrator consumes, which is sufficient for Tasks 11-14 to consume it.
- `services/atlas-channel` has no direct role in this flow — the list saga is initiated entirely from atlas-mts's own REST handler (`listing/processor.go:805`), not from atlas-channel. No `TransferToMtsPayload{` construction site was found under `services/atlas-channel`.
- `RingId`/`ViciousCount`/`ItemLevel` are also NOT copied in `expandTransferToMts`'s snapshot literal (`processor.go:1580-1600` skips them, though the payload/command structs declare them) — this appears to be a pre-existing gap unrelated to owner/flag and is out of scope for this task, noted here only so Tasks 11-14 don't assume that block is fully wired.
