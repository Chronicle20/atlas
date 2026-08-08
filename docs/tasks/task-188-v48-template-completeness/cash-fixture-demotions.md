# gms_v48 cash fixture demotions

Why several `gms_v48` cash cells lost a `✅` on this branch, and which of them
came back. Written at the pre-PR review step: the facts were already recorded in
commit messages, but nothing in the task folder surfaced them, so a reviewer
reading only the matrix saw an unexplained coverage drop.

## What happened

`ef950fdb1` fixed a real bug — `shop_operation_increase_storage.go` decoded a
4-byte currency field that the v48 client never sends
(`CCashShop::OnIncTrunkCount @0x44aad1` encodes three bytes: mode `@0x44abf4`,
`isPoints` `@0x44ac04`, `Encode1(0)` `@0x44ac0d`).

Fixing it meant seeding five `CCashShop` FNames into the v48 export. That
reclassified the whole `CashShopOperation` family as **tier-1**, and tier-1 cells
require a byte fixture to grade `verified`. Seven cells had been graded `✅` on a
round-trip test only — the exact false-verify the packet docs warn about — so
they dropped to `partial: tier-1 needs byte-fixture test to verify`.

That demotion is the honest grade, not a break. Reverting the export addition
would have restored a false `✅`. From `ef950fdb1`'s message:

> KNOWN SIDE EFFECT, recorded rather than papered over […] those cells were
> exactly the round-trip-only ✅ that docs/packets warns about. Reverting the
> export addition would restore a false ✅, so it stands; the fixtures are the
> next batch.

## Current state

| Arm | Fixture | Cell |
|---|---|---|
| `BuyNormal` | re-fixtured in `79a0d16e6` (`OnBuyNormal @0x44cbb2`) | verified |
| `SetWishlist` | re-fixtured in `79a0d16e6` (`OnSetWish @0x44ce9b`) | verified |
| `Buy` | owed | partial |
| `BuyCouple` | owed | partial |
| `BuyFriendship` | owed | partial |
| `BuyPackage` | owed | partial |
| `Gift` | owed | partial |

The five `owed` arms are carried in `coverage-manifest.yaml` under
`out_of_scope` so they are tracked rather than forgotten. Their codecs already
carry the legacy gates (`buyOmitsCurrency`, `legacyGMS`) and need
*verification*, not changes.

## The one that was collateral damage

`ef950fdb1` rewrote `cash/serverbound/v48_test.go` wholesale and, in the
process, also deleted two fixtures that had nothing to do with the
`CashShopOperation` family:

- `TestItemUseMegaphoneBytesV48`
- `TestItemUseSuperMegaphoneBytesV48`

Both pin `USE_CASH_ITEM` (`CWvsContext::SendConsumeCashItemUseRequest
@0x70e495`), not `CASHSHOP_OPERATION`. That commit's message accounts for the
seven `CashShopOperation` arms and never mentions these two, so the resulting
`USE_CASH_ITEM × gms_v48` demotion `verified -> partial` was unexplained and
looks unintended.

**Restored.** Both tests were re-added verbatim from `origin/main` — the
increase-storage fix touched no megaphone codec, and both pass unchanged.
`USE_CASH_ITEM × gms_v48` is back to `verified`, and the matrix regen moved
exactly that one cell.

## Net effect on the gms_v48 column vs `origin/main`

| Grade | main | now |
|---|---|---|
| verified | 182 | 196 |
| partial | 2 | 19 |
| incomplete | 203 | 195 |
| n-a | 628 | 605 |

The `partial` growth is the tier-1 reclassification above plus newly-wired ops
that are routed but not yet fixtured. The only `verified -> n-a` movement is
`FIELD_OBSTACLE_ONOFF_LIST`, which is the deliberate consequence of the opcode
correction in `5270b70fe` — the op does not exist on v48.
