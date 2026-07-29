# v92 CashShopOperation template wiring (out-of-matrix)

## Why this is a hand-edit

`gms_v92` is a **template-only** client version: it ships a tenant seed template
(`template_gms_92_1.json`) but is **not** one of the 9 `matrix.VersionKeys`
(`tools/packet-audit/internal/matrix/model.go`). `packet-audit operations`
iterates only `matrix.VersionKeys`, so it never emits the v92 template's
`CashShopOperation` `operations` map — the map was empty, which drops every
`CCashShop::OnCashItemResult` arm at emit time (the "MODE resolves wrong /
packet dropped" failure).

Promoting v92 into `matrix.VersionKeys` would make it a full matrix column (a
version bring-up — `STARTING_A_NEW_VERSION_PASS`), which is out of scope. Per
the task-138 precedent (v92 mount-food handler wired directly in the v92
template), the `operations` map is wired directly here, with **IDB-verified**
byte values.

## IDB evidence (GMS_v92_1_DEVM)

`CCashShop::OnCashItemResult` @ `0x495300`. The dispatcher normalizes the mode
byte before switching:

```c
v4 = (unsigned __int8)CInPacket::Decode1(iPacket) - 83;   // wire byte = case + 83
switch (v4) { case 0: ...LimitGoodsCountChanged; case 4: ...LoadLockerDone; ... }
```

So the **wire mode byte = switch-case value + 83**.

Contrast with v95 (`?OnCashItemResult@CCashShop@@IAEXAAVCInPacket@@@Z`, session
`e4abcb98`), which switches on the **raw** byte (`case 0x54u:
LimitGoodsCountChanged`, `case 0x58u: LoadLockerDone`, …). The gap pattern
differs between the two builds, so **v92 is NOT a constant offset from v95**:

| operation (v95 handler)            | v95 wire | v92 case | v92 wire |
|------------------------------------|---------:|---------:|---------:|
| LIMIT_GOODS_COUNT_CHANGED          | 84 (0x54)|        0 |       83 |
| LOAD_INVENTORY_SUCCESS (LoadLocker)| 88 (0x58)|        4 |       87 |
| CHANGE_MAPLE_POINT_FAILED          |188 (0xBC)|      102 |      185 |

(diff is 1 for early arms, 3 for the last — a naive `v95-1` copy would be wrong.)

Both switches enumerate the **same 57 handlers** (verified by handler-name set
equality), so no arm is `n-a` for v92. The full key→byte map was produced by
mapping each v95 `operations` key → its v95 handler → the v92 case for that same
handler → `case + 83`, and written into the v92 template
`CashShopOperation` writer `options.operations` (57 entries, ascending by byte,
matching the v95 template's ordering).

## Deployment note

Live v92 tenants still need a socket-config PATCH to pick up the seed change
(the seed template is the source of record; live configs are not auto-migrated).
