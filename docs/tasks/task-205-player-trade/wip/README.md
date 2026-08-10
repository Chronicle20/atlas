# WIP: the escrow settlement/unwind pass (Tasks 27–29)

`escrow-settlement-orchestrator.patch` is a **working, fully green** pass over
`libs/atlas-saga` and `atlas-saga-orchestrator` — plus the `settlement` package
rename inside atlas-trades — that was set aside because the atlas-trades
`trade` package half of the same change could not be finished in the same
sitting. It is kept because re-deriving it costs more than reading it.

Apply with `git apply docs/tasks/task-205-player-trade/wip/escrow-settlement-orchestrator.patch`
and add `expansion_stub_test.go.new` as
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/expansion_stub_test.go`.

**DO NOT apply it alone.** With the patch on and the atlas-trades `trade`
package unchanged, everything still compiles but `settlementPayload` leaves
`EscrowId` zero, so every settlement releases the nil escrow row. A red build is
recoverable; that is not.

## What the patch already does, and why

- **`TradeSettlementItem` becomes `TradeEscrowItem`**, carrying `EscrowId` plus
  the full stat snapshot. The orchestrator can no longer look an escrowed asset
  up — there is no compartment row to read — so the snapshot has to travel on the
  payload.
- **`expandTradeSettlement` releases from escrow and never debits meso.** The
  debit happened at stage time (design §5A.5); the old negative leg would charge
  the giver twice. `TestExpandTradeSettlementEmitsNoNegativeAward` is the guard.
- **`trade_unwind` is a new composite** for teardown: every item back to its
  owner, every escrowed meso refunded IN FULL. It is deliberately separate from
  settlement rather than "settlement with recipient == owner", because a refund
  is untaxed and a delivery is not — fusing them would put an "is this a refund?"
  branch inside the expander.
- **The trade reverse-walk's release inverse is now a custody RESTORE**, not a
  re-grant. That is not just a rename: restoring the row cannot race an accept
  that may already have delivered the same item to the counterparty.
- **Deleted, not ported:** the asset-id step-id pairing (`tradeStepAssetId`,
  `tradeAcceptSnapshots`) and the settlement expander's slot / instance /
  unparseable-id checks. All of them guarded against the staged asset changing
  underneath the settlement, which escrow makes impossible.

## What still has to be written (the atlas-trades `trade` package)

1. `StagedItem.reservationId` → `escrowId`, plus a `pending` flag. Pending items
   HOLD their dialog slot but are NOT announced: the release step unlocks the
   client before the escrow row exists, so without the hold the client can stage
   twice into one slot, and without the silence a failed saga shows both dialogs
   an item that was never escrowed (design §5A.4).
2. `putItem` submits `transfer_to_trade` (saga transactionId == escrowId, so a
   SAGA_FAILED maps straight back to the row) instead of reserving.
3. The custody consumer confirms the stage — clears `pending`, emits
   `ITEM_STAGED` — since the escrow row already carries roomId/tradeSlot/ownerId
   and is therefore the correlation handle.
4. `AddMeso` debits the DELTA against `escrow.MesoByOwner` and upserts the row.
5. Teardown submits `trade_unwind`; `claimedRoom.releases` disappears entirely
   (the DB is the source of truth, so `withLateStages` has nothing to catch).
6. `settlementPayload` becomes a method that reads `escrow.ItemsByRoom`.
7. **Delete:** `stagedRelease`, `resolveStagedReleases`, `withLateStages`,
   `emitStagedReleases`, `resolveStagedSlot`, `compartmentCache`,
   `stagedSlotCorrections`, `releasesFor`, `RefreshReservations`, and the whole
   `compartment` package.
8. `ITEM_REFUSED` in both trade-contract copies + the atlas-channel arm (the
   `MESO_REFUSED` half already landed in dc1e679f3).
