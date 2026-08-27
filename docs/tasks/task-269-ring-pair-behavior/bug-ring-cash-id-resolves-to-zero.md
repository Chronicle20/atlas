# bug: ring pair SNs resolve to 0 once the ring leaves the cash locker, so the ring set never matches and no effect renders

**Reproduced:** tenant `7e2d468e-d80c-4869-a44e-45e10a7b5dbf` (GMS 83.1, environment
`pr-1524`). Characters `Atlas` (id 1, account 1) and `Chronicle` (id 2, account 2),
FRIENDSHIP pair bought fresh at `2026-08-27T21:54:38Z`, ring equipped.

> Note on the previous session's tenant: the namespace redeploy recreated the
> bootstrap tenant, so the earlier id `935188e4-…` (and character ids 38/39) is
> dead. Any finding recorded against it is void — including the "no ring rows
> exist" claim in the previous investigation, which was a query against the wrong
> tenant. The rows do exist and are correct.

**Observed:** `GET /api/rings?filter[characterId]=1` (now reachable — see
`bug-rings-route-missing-from-ingress.md`) returns a well-formed, correctly
paired row, but with **both serial numbers zero**:

```json
{"pairId":"3f2fac1c-1b7b-4e3c-b41a-2d59a7c90b13","characterId":1,
 "partnerCharacterId":2,"assetId":1,"itemTemplateId":1112802,
 "ringType":"FRIENDSHIP","state":"ACTIVE",
 "cashId":0,"partnerCashId":0,"partnerName":"Chronicle"}
```

`partnerName` resolves ("Chronicle" / "Atlas"), so the character-service leg of
`enrich` works. Only the two asset legs fail.

The asset that `cash_rings.asset_id` names is gone:

```
$ curl -D - .../api/cash-shop/assets/1   ->  HTTP/1.1 404 Not Found
$ curl -D - .../api/cash-shop/assets/2   ->  HTTP/1.1 404 Not Found
```

Meanwhile the equipped ring in the character's own inventory carries a real SN —
`GET /api/characters/1/inventory`, slot `-112` (the cash-equip mirror of ring1
`-12`), `templateId 1112802`, `cashId 7454243858150396663`.

**Expected:** PRD FR-13/FR-6 — the ring block on spawn carries the character's own
ring SN and the partner's ring SN, so the client can pair the two `CUser`s in its
user pool and fire the effect.

**Root cause:** `cash_rings.AssetId` stores the **cash-locker asset id** created at
purchase (`cashshop/processor_ring.go:191-193` passes `buyerAsset.Id()` /
`partnerAsset.Id()` into `ring.Half.AssetId`). `ring.ProcessorImpl.enrich`
(`ring/processor.go:120-135`) resolves the SN at read time via
`astP.GetById(half.AssetId())`. Once the ring is taken out of the locker and
equipped, that locker asset row no longer resolves — the 404 above — so the
lookup fails, `enrich`'s deliberate fail-soft leaves `cashId` and
`partnerCashId` at 0, and nothing reports an error anywhere.

Both symptoms follow from the zero SNs:

- `selectPair` (`atlas-channel/ring/processor.go:170-200`) matches
  `h.CashId() != equippedCashId`. `0 != 7454243858150396663` for every half, so
  no pair is ever selected, `GetRingSet` returns an empty `RingSet`, and
  `CharacterSpawn` writes three zero flag bytes. The client's
  `OnFriendRecordAdd` never gets an entry, `CUserPool::Update` never calls
  `CUser::SetCoupleItemEffect` — **no proximity effect**.
- `GetRingRecords` still emits the record (the name is intact), but with
  `OwnSN`/`PairSN` zero. The tooltip line is formatted from
  `sPairCharacterName`, so it may render once the record block reaches the
  client; the SNs are what the user-pool pairing uses.

The stable identifier already exists and already travels with the item:
`cashId` is generated once at asset creation
(`cashshop/inventory/asset/administrator.go:13-35`, `generateUniqueCashId`) and
is the value the inventory side carries on the equipped asset. It is available
on the `Model` returned by `astP.Create` / `CreateGift` at purchase time.

## Fix

- `services/atlas-cashshop/atlas.com/cashshop/ring/entity.go` — add a
  `CashId int64` column to `Entity` (AutoMigrate covers the migration; there is
  no backfill for existing rows and none is needed — a PR-environment re-purchase
  produces correct rows). Keep `AssetId`: it is still the locker provenance.
- `services/atlas-cashshop/atlas.com/cashshop/ring/model.go` /
  `administrator.go` — carry the new field through `Half`, `Model`, `Make`, and
  `ToEntity`.
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_ring.go:191-193`
  — populate `ring.Half.CashId` from `buyerAsset.CashId()` /
  `partnerAsset.CashId()` at purchase, inside the existing transaction.
- `services/atlas-cashshop/atlas.com/cashshop/ring/processor.go:120-135`
  (`enrich`) — prefer the persisted `CashId`; fall back to the existing
  `astP.GetById(half.AssetId())` lookup only when the stored value is 0, so rows
  written before this change still resolve while the ring is in the locker.
  Same for `partnerCashId` via the sibling row.
- Tests: `ring/processor_test.go` — a case proving `enrich` returns the persisted
  SN **when the locker asset no longer exists** (that is the case that fails
  today). `cashshop/processor_ring_test.go` — a case asserting the purchase
  writes a non-zero `CashId` on both halves.

## Not yet answered

- **Whether `atlas-channel`'s equipped-side `CashId()` is the same value.** The
  live inventory shows `7454243858150396663` on the equipped asset and the
  cashshop asset generated its `cashId` the same way, so these should be equal —
  but this has NOT been observed equal end to end, because the cashshop side is
  0 today. After the fix, verify by comparing `GET /api/rings` `cashId` against
  the equipped asset's `cashId` for the same character before declaring it fixed.
- **Do not change `selectPair`.** Its comparison is correct; it was being fed
  zeros. If the two ids turn out to differ after this fix, that is a separate
  bug with its own file.

## Resolution

- **Fixed by:** `0f1f8872b` — "fix(cash-shop): persist ring half CashId so pairs
  resolve after equip". atlas-cashshop only; `atlas-channel` untouched
  (`git diff d8b0ec8d1..HEAD -- services/atlas-channel` is empty).
- **Gate:** `tools/verify.sh --quick --base d8b0ec8d1` → exit 0 (build/vet on
  atlas-cashshop, analyzer guards, scope guard, producer seam guard, env domain
  guard, lint & format all green).
- **Seam review:** APPROVED_WITH_FINDINGS, 0 blocking, 1 non-blocking (a stale
  commit hash in the fix report, corrected in this commit). Artifact:
  `review-ring-cash-id-fix.md`. The reviewer traced all five seam checks by hand,
  including that `ring/rest.go` already carried `CashId`/`PartnerCashId` out to
  the channel so no transformer change was needed.
- **Live re-test:** NOT YET CONFIRMED. Requires a redeploy AND a **fresh ring
  purchase** — existing rows keep `cash_id = 0` and cannot be backfilled.
- **Still open:** the end-to-end numeric equality of the cashshop-persisted
  `cashId` and the channel-side equipped asset's `CashId()` is a deliberate
  deferral, not a silent gap. Confirm it on the re-test by comparing
  `GET /api/rings` `cashId` against the equipped asset's `cashId` for the same
  character before calling this fixed.
