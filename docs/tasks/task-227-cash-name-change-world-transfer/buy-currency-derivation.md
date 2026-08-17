# task-227 — What currency does `cashshop.RequestPurchase` need for BUY_NAME_CHANGE / BUY_WORLD_TRANSFER?

## The question

`cashshop.RequestPurchase(characterId, serialNumber, isPoints, currency, zero)`
(`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:98-127`) treats
`currency` as a client-declared wallet selector (`wallet.Model.Balance`/`.Purchase`,
`services/atlas-cashshop/atlas.com/cashshop/wallet/model.go:37-58`: `1` → credit/NX
Cash, `2` → maple points, anything else → NX Prepaid). The client's
`BUY_NAME_CHANGE` (mode 46) and `BUY_WORLD_TRANSFER` (mode 49) ops decode to no
`currency`/`isPoints` field in `libs/atlas-packet/cash/serverbound/shop_operation_buy_name_change.go`
and `.../shop_operation_buy_world_transfer.go`, and `commodity.Model` carries only
`Price`, no wallet type. Task: determine from IDA evidence what wallet the real
client intends, so a future fulfillment/charge path for these two ops knows what to
pass.

**Repo-state note, found during this pass:** as currently implemented,
`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`
`handleBuyNameChange`/`handleBuyWorldTransfer` do **not** call `RequestPurchase` at
all — the doc comments at lines 206-217 and 267-273 already state the packet
carries no currency and defer charging to "the pending-change's own execution
saga" (not yet built; no `PENDING_CHANGE_CREATED` consumer exists anywhere under
`services/atlas-cashshop/`, confirmed by `grep -rln PENDING_CHANGE_CREATED
services/ libs/` returning only `atlas-character` producer/model files). This
finding is for whoever builds that saga.

## Evidence

### 0. IDA sessions used

| Version | Binary | Session | Confirmed |
|---|---|---|---|
| gms_v83 | `MapleStory_dump.exe.i64` | `41f13e0d` | baseline, primary evidence |
| gms_v87 | `GMSv87_4GB.exe.i64` | `d51ecbd3` | cross-check |
| gms_v95 | `GMS_v95.0_U_DEVM.exe.i64` | `79906a1e` | cross-check, PDB-backed |

All three resolved from `idb_list` by binary name, per `input_path`, matching
`docs/tasks/task-227-cash-name-change-world-transfer/derivation.md` §0.

### 1. The wire layout carries no currency field (repo codec vs. client send, byte for byte)

**v83 client send functions** (`CCashShop::SendBuyNameChangeItemPacket` @`0x47342f`,
`CCashShop::SendBuyTransferWorldItemPacket` @`0x473601`, decompiled directly):

```c
// SendBuyNameChangeItemPacket @0x47342f
COutPacket::COutPacket(&v6, 229);
COutPacket::Encode1(&v6, 0x2Eu);        // mode 46 = BUY_NAME_CHANGE
COutPacket::Encode4(&v6, a2);           // serial number
COutPacket::EncodeStr(&v6, ... a3);     // old name
COutPacket::EncodeStr(&v6, ... a4);     // new name
CClientSocket::SendPacket(...);
```

```c
// SendBuyTransferWorldItemPacket @0x473601
COutPacket::COutPacket(&v4, 0xE5);      // 229, same opcode as above
COutPacket::Encode1(&v4, 0x31u);        // mode 49 = BUY_WORLD_TRANSFER
COutPacket::Encode4(&v4, a2);           // serial number
COutPacket::Encode4(&v4, a3);           // target world
CClientSocket::SendPacket(...);
```

Field-for-field this matches the repo decoder exactly — `ShopOperationBuyNameChange`
(`libs/atlas-packet/cash/serverbound/shop_operation_buy_name_change.go:45-51`:
`serialNumber`, `oldName`, `newName`) and `ShopOperationBuyWorldTransfer`
(`.../shop_operation_buy_world_transfer.go:42-47`: `serialNumber`, `targetWorld`).
There is no third/fourth encoded field on either send, at any position, that could
be a hidden currency selector. **No selector exists on the wire for either op, on
v83** — the client literally does not have the value to send.

### 2. The client hard-codes the wallet to NX Prepaid — confirmed on 3 versions

The button handlers that build the confirmation dialog **before** calling the two
send functions above hard-code the wallet in the confirmation string and in which
balance field they debit-check against.

**v83** — `CCashShop::OnBuyNameChange` @`0x47031e` (name-change arm; calls
`SendCheckNameChangePossiblePacket`) and `CCashShop::OnBuyTransferWorld` @`0x470480`
(world-transfer arm; calls `CheckTransferWorldPossible` then
`SendCheckTransferWorldPossiblePacket`) both contain the **identical literal
string**, `aYouWillSpendDN` @`0xaf20c8`:

```c
v5 = _ZtlSecureFuse<long>(this + 52, v14);   // compare balance field this+52 vs price
...
if ( v5 >= v4 )
  ZXString<char>::Format(&a2, "You will spend %d NX Prepaid.\r\nWould you like to proceed?", v4);
else
  StringPool::GetString(Instance, &v10, SP_593_YOU_DONT_HAVE_ENOUGH_NX_PREPAID);
```

Both the success-path confirmation text *and* the insufficient-funds StringPool
entry (`SP_593_YOU_DONT_HAVE_ENOUGH_NX_PREPAID`) name NX Prepaid explicitly, not a
generic "cash"/"points" wording. There is no branch on any client-side "which
wallet did the user pick" state — the field compared (`this + 52`) is fixed, and
the string is a `Format()` of a fixed literal, not a table indexed by a wallet
enum.

**v87** — `CCashShop::OnBuyNameChange` @`0x47ab57`, freshly decompiled this pass,
same shape: `_ZtlSecureFuse<long>((this + 14), v14) >= v5` gates the identical
`"You will spend %d NX Prepaid.\r\nWould you like to proceed?"` format string
(`aYouWillSpendDN` @`0xb958b8`), else `StringPool::GetString(&v10, 602)` (the
insufficient-NX-Prepaid string at this version's numbering).

**v95** — `CCashShop::OnBuyNameChange` @`0x491200`, freshly decompiled this pass,
PDB-backed (member names resolved, not `sub_*`/offsets). This is the decisive
cross-check: the balance field compared against price is not just offset-52-shaped
but **named by the PDB**:

```c
v19[0] = this->_ZtlSecureTear_m_nPrepaidNXCash_CS;
ZtlSecureTear_m_nPrepaidNXCash = this->_ZtlSecureTear_m_nPrepaidNXCash;
...
if ( _ZtlSecureFuse<long>(ZtlSecureTear_m_nPrepaidNXCash, v19[0]._m_pStr) >= v8 )
    ZXString<char>::Format(&nCommSN, "You will spend %d NX Prepaid.\r\nWould you like to proceed?", v8);
```

`CCashShop::m_nPrepaidNXCash` (the field, per its symbol name) is the balance the
client checks and implicitly debits for both name-change and world-transfer
purchases. This is a named CCashShop member, not an inferred offset — it settles
that the field is specifically the NX-Prepaid balance, not a generic "selected
wallet" cache.

`CCashShop::OnBuyTransferWorld` was not independently re-decompiled on v87/v95 in
this pass (budget), but its v83 body (§ above) shares the identical
`aYouWillSpendDN` string and `_ZtlSecureFuse` gate pattern as `OnBuyNameChange`
within the same binary — both are literal-for-literal copies of the same
confirmation-dialog idiom, addressed at `derivation.md` §1.1 ("byte-identical in
shape"). Given v83's OnBuyTransferWorld already reuses the exact same
`aYouWillSpendDN` string constant as OnBuyNameChange in the same binary (not a
separate string), and v87/v95 confirm OnBuyNameChange's wording and field-name are
stable across an 8-version-table span (v83→v87→v95, GMS major 83/87/95), there is
no plausible branch point where OnBuyTransferWorld would diverge to a different
wallet on those same two versions.

### 3. Contrast: the generic BUY op DOES carry a client-chosen selector

`ShopOperationBuy` (`libs/atlas-packet/cash/serverbound/shop_operation_buy.go:17-33`,
`packet-audit:fname CCashShop::OnBuy`) decodes `isPoints bool` + `currency uint32`
+ `serialNumber` (+ trailing version-gated fields), sourced from prior IDA passes
cited in the file's own comments (`CCashShop::OnBuy` v61 `@0x457ea4`, v48
`@0x44b0cf`/send `@0x44b38a`) — i.e. a real per-purchase wallet selector chosen in
the UI and placed on the wire before the serial number. `RequestPurchase`'s
`currency uint32` parameter is that field, verbatim, for the generic `BUY` op.
`BUY_NAME_CHANGE`/`BUY_WORLD_TRANSFER` have no analogous field anywhere in their
send functions (§1) — the two op families are genuinely different: one op lets the
player pick a wallet in the UI and puts the pick on the wire, the other two never
show a wallet choice at all and always debit NX Prepaid.

## VERDICT: DERIVED

**The client always intends NX Prepaid for both `BUY_NAME_CHANGE` and
`BUY_WORLD_TRANSFER`.** There is no wallet-selector UI, no wire field, and no
client-side branch on wallet type for either op — the confirmation text
("You will spend %d NX Prepaid...") and the balance field checked
(`CCashShop::m_nPrepaidNXCash`, PDB-named on v95) are both hard-coded, confirmed
identically on v83, v87, and v95.

**What Atlas should pass to `RequestPurchase`:** any `currency` value that is
*not* `1` (credit/NX Cash) and *not* `2` (points) — per
`wallet.Model.Balance`/`.Purchase` (`services/atlas-cashshop/atlas.com/cashshop/wallet/model.go:37-58`),
every other value falls through to the `prepaid` branch. `currency = 0` is the
natural choice (it is also what a zero-valued/absent field defaults to), but any
non-1/non-2 sentinel is equivalent given the current three-way switch — there is
no dedicated named constant for "prepaid" in `wallet` today, only the fallthrough
`else`. Whoever wires the fulfillment/charge saga should pass `isPoints = false`
and `currency` = that non-1/non-2 sentinel (0 is fine), not attempt to read a
currency off the `ShopOperationBuyNameChange`/`ShopOperationBuyWorldTransfer`
structs — those structs correctly have no such field, and none should be added.
