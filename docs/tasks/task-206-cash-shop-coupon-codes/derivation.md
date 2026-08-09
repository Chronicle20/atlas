# task-206 — IDA derivation record

Every wire value used by this task originates here. A value absent from this
file may not appear in a codec, a dispatcher YAML, or a tenant template.

Method: `docs/packets/audits/VERIFYING_A_PACKET.md`. Sessions resolved from
`idb_list` by binary NAME and passed as the `database` parameter.

| version | IDB binary name | session id at derivation time |
|---|---|---|
| gms_v83 | `MapleStory_dump.exe.i64` | `41f13e0d` |
| gms_v84 | `GMS_v84.1_U_DEVM.i64` | `5881cf84` |
| gms_v87 | `GMSv87_4GB.exe.i64` | `d51ecbd3` |
| gms_v92 | `GMS_v92_1_DEVM.exe.i64` | `acdfccff` |
| gms_v95 | `GMS_v95.0_U_DEVM.exe.i64` | `79906a1e` |
| jms_v185 | `MapleStory_dump_SCY.exe.i64` | `b6864e54` |

---

## Structural finding that reframes the whole task (READ FIRST)

**There is no `CCashShop::OnCashItemRequest` function in the client, and
`USE_COUPON` is not an arm of the serverbound `CASHSHOP_OPERATION` mode
switch.**

Two independent facts, both read out of the gms_v83 IDB and re-confirmed on
v84/v87/v92/v95:

1. The client has no single serverbound cash-shop request builder. Each UI
   action constructs its own `COutPacket(&pkt, <CASHSHOP_OPERATION opcode>)`
   and writes its own leading `Encode1(mode)`. The "mode switch" is a
   *server-side* construct; the client-side ground truth is the complete set
   of `COutPacket` construction sites for that opcode. This record enumerates
   those sites exhaustively via a byte search for the opcode-push
   (`68 <op> 00 00 00` / `6A <op>`), so the arm table is complete by
   construction, not by name search.

2. The coupon submission is a **separate opcode** — the registry's
   `COUPON_CODE` serverbound op — built by `CCashShop::OnStatusCoupon`. It has
   **no mode byte at all**; the body starts immediately with strings.

Consequence for downstream tasks: `USE_COUPON` must **not** be added to
`CashShopOperationHandle.options.operations`, and the new codec is not a
`cash/serverbound/ShopOperationUseCoupon` sub-arm of `CASHSHOP_OPERATION` — it
is a codec for the standalone `COUPON_CODE` op. Tasks 5, 7, 8, 9 need to be
re-read against this before they are executed.

---

## gms_v83

- Serverbound `CASHSHOP_OPERATION` opcode: **229 / `0xE5`**
  (`docs/packets/registry/gms_v83.yaml`).
- Serverbound `COUPON_CODE` opcode: **230 / `0xE6`**.
- No serverbound dispatcher function exists (see structural finding). Send
  sites enumerated by byte search for `68 E5 00 00 00` — 24 hits, all inside
  `CCashShop` methods, all accounted for below.
- Clientbound dispatcher: `CCashShop::OnPacket` @ `0x478e2b` →
  `CCashShop::OnCashItemResult` @ `0x47915f`.
- Failure-reason sink: `CCashShop::NoticeFailReason` @ `0x47c17a`.

### Serverbound `operations` (CashShopOperationHandle)

24 `COutPacket(…, 0xE5)` sites → **19 distinct mode bytes**, which is exactly
the 19 keys already in `template_gms_83_1.json`.

| Key | Mode | Evidence |
|---|---|---|
| BUY | 3 | `CCashShop::SendBuyAvatarPacket` ctor @ `0x46bc85`, `push 3` @ `0x46bc95`; also `CCashShop::OnBuy` ctor @ `0x46dfc7`, `push 3` @ `0x46dfd4` |
| GIFT | 4 | `CCashShop::SendGiftsPacket` ctor @ `0x46fb76`, `push 4` @ `0x46fb8b`; `CCashShop::OnGiftMateInfoResult` `push 4; lea ecx,[ebp-50h]; call Encode1` @ `0x46f19c`; `CCashShop::GiftWishItem` (`push 4; pop edi` @ `0x472fc6`) |
| SET_WISHLIST | 5 | `CCashShop::OnSetWish` ctor @ `0x470e4e`, `push 5` @ `0x470e5f`; `CCashShop::OnRemoveWish` ctor @ `0x470f6d`, `push 5` @ `0x470f7e` |
| INCREASE_INVENTORY | 6 | `CCashShop::OnExItemSlot` ctor @ `0x46c308`, `push 6` @ `0x46c315`; also the computed arm in `CCashShop::OnBuySlotInc` (`cmp eax,2396h; sete al; add eax,6` @ `0x470988`-`0x470990` → 6 when itemId/1000 != 9110) |
| INCREASE_STORAGE | 7 | `CCashShop::OnIncTrunkCount` ctor @ `0x46c681`, `push 7` @ `0x46c68e`; also `OnBuySlotInc` computed arm (→ 7 when itemId/1000 == 9110) |
| INCREASE_CHARACTER_SLOT | 8 | `CCashShop::OnIncCharacterSlotCount` ctor @ `0x46c854`, `push 8` @ `0x46c861` |
| ENABLE_EQUIP_SLOT | 9 | `CCashShop::OnEnableEquipSlotExt` ctor @ `0x46c9f5`, `push 9` @ `0x46ca02` |
| MOVE_FROM_CASH_INVENTORY | 13 | `CCashShop::OnMoveCashItemLtoS` ctor @ `0x4727a8`, `push 0Dh` @ `0x4727b5` |
| MOVE_TO_CASH_INVENTORY | 14 | `CCashShop::OnMoveCashItemStoL` ctor @ `0x4728f1`, `push 0Eh` @ `0x472901` |
| REBATE_LOCKER_ITEM | 26 | `CCashShop::OnRebateLockerItem` ctor @ `0x46bf74`, `push 1Ah` @ `0x46bf81` |
| BUY_COUPLE | 29 | `CCashShop::OnBuyCouple` ctor @ `0x47021f`, `push 1Dh` @ `0x47022c`; also `GiftWishItem` `push 1Dh` @ `0x473000` |
| BUY_PACKAGE | 30 | `CCashShop::OnBuyPackage` ctor @ `0x46e50b`, `push 1Eh` @ `0x46e518` |
| BUY_OTHER_PACKAGE | 31 | `CCashShop::OnGiftPackage` (named this pass; was `sub_46FC3B`) ctor @ `0x46fef1`, `push 1Fh` @ `0x46fefe` |
| BUY_NORMAL | **32** | `CCashShop::OnBuyNormal` ctor @ `0x46e79e`, `push 20h` @ `0x46e7ab` — **TEMPLATE BUG**, see below |
| APPLY_WISHLIST | 33 | `CCashShop::ApplyWishListEvent` ctor @ `0x47315f`, `push 21h` @ `0x473170` |
| BUY_FRIENDSHIP | 35 | `CCashShop::OnBuyFriendship` ctor @ `0x470c7e`, `push 23h` @ `0x470c8b`; also `GiftWishItem` `push 23h` @ `0x472feb` |
| GET_PURCHASE_RECORD | 40 | `CCashShop::RequestCashPurchaseRecord` ctor @ `0x46bd1b`, `push 28h` @ `0x46bd2c` |
| BUY_NAME_CHANGE | 46 | `CCashShop::SendBuyNameChangeItemPacket` ctor @ `0x473443`, `push 2Eh` @ `0x473453` |
| BUY_WORLD_TRANSFER | 49 | `CCashShop::SendBuyTransferWorldItemPacket` ctor @ `0x473611`, `push 31h` @ `0x473622` |

**Template bug — `BUY_NORMAL`.**
`services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
(`CashShopOperationHandle` → `options.operations`) declares `"BUY_NORMAL": 20`.
The decompile pushes `20h` = **32**. Both values recorded; the template value
is a hex-digits-read-as-decimal transcription error. `template_gms_84_1.json`
carries the identical `20` and the identical bug. Do not silently prefer
either — fix the template in the task that owns the `operations` YAML.

**Row-count self-check:** 24 opcode-construction sites → 19 distinct mode
bytes → 19 rows. Matches the template's 19 keys one-for-one on names.

### `errors` enum (reason byte accepted by every `*Failed` arm)

Every failure arm reads a single `Decode1` and passes it **unmodified** to
`CCashShop::NoticeFailReason(this, reason)` — e.g.
`CCashShop::OnCashItemResUseCouponFailed` @ `0x47a7db`:
`v3 = CInPacket::Decode1(a2)` @ `0x47a7eb`, then
`CCashShop::NoticeFailReason(this, v3)` @ `0x47a806`. So the wire reason byte
*is* the `NoticeFailReason` switch key.

`NoticeFailReason` @ `0x47c17a` is a jump-table switch:
`add eax, 0FFFFFF5Dh` (bias −163) @ `0x47c180`, `cmp eax, 46h` @ `0x47c185`,
`ja def_47C18E` @ `0x47c188`. IDA's own jump-table annotation reads
**"switch 71 cases"** with **"default case, cases 164,174,175,177,197,198,
201-204,206,207,209,225-229"**. So the accepted domain is 163…233
(`0xA3`…`0xE9`) minus those 18 → **53 explicit reason values**, everything
else falling to the default notice (StringPool id 557).

**What is directly verified vs. what is aligned.** The switch *cases* and each
case's StringPool id are read straight out of the decompile (addresses below).
The client's id → message text lives in the StringPool data file loaded at
runtime, **not in the IDB**, so per-key text confirmation is not available
from this session. The key names below are assigned by ordinal alignment of
the 53 cases (ascending) against the 53 non-`UNKNOWN_ERROR`
`CashShopOperationError*` constants in
`libs/atlas-packet/cash/clientbound/shop_operation_body.go:80-125`, in their
declared order. That alignment is pinned by three independent anchors:

- **Anchor A (exact).** Case `0xD2` @ `0x47c42b` does not use the StringPool
  at all — it builds an inline literal `"The coupon system will be available
  soon."` (`aTheCouponSyste` @ `0xaf2328`). Ordinal position 35 of the case
  list. Atlas key #36 (counting `UNKNOWN_ERROR` as #1) is
  `COUPON_SYSTEM_AVAILABLE_SOON`. Exact match.
- **Anchor B.** `CCashShop::OnStatusCoupon` @ `0x4710e8` calls
  `NoticeFailReason(this, 0C3h)` @ `0x471100` when the cash-shop-enabled flag
  `[this+40h]` is zero. Ordinal position 29 → Atlas key #30
  `CASH_SHOP_NOT_AVAILABLE_DURING_BETA`. Consistent.
- **Anchor C (structural).** The ten coupon-specific Atlas keys (#12–#21) land
  on the contiguous block `0xB0`–`0xBA`, and the one hole in that block,
  `0xB1`, is exactly the value `OnCashItemResUseCouponFailed` special-cases
  @ `0x47a822` to show `SP_548_YOU_ARE_DISCONNECTED_FROM_CASH_SHOP_BECAUSE_
  YOU_PUT_WRONG_COUPON_NUM` and then `OnStatusExit`. A coupon-reason block
  sitting exactly where the coupon keys fall is not a coincidence.

Rows are marked `verified` (anchored) or `aligned` (ordinal inference).
An `aligned` row is safe to *encode* — the byte is real and the client will
render *a* notice for it — but the specific English wording is unconfirmed.

**WZ/StringPool cross-check (this pass).** Attempted to close the "specific
English wording is unconfirmed" gap for `aligned` rows using local WZ/String
data instead of decompilation. Three routes were checked:

1. **Local extracted XML dump.** No `tmp/<tenant-uuid>/GMS/83.1/` tree exists
   in this worktree or environment (the memory note describing this path
   referenced an ephemeral dump from an unrelated past session that was not
   persisted). Other MapleStory client source checkouts present on this
   machine (`~/source/Cosmic`, `~/source/ms_1172`, etc., unrelated to this
   Atlas task) do have an extracted `String.wz/Cash.img.xml`, but its
   structure is item-**id**-keyed (`<imgdir name="5000011"><string
   name="name" value="Monkey"/>`) — cash-shop item catalog names/descriptions,
   not a generic small-integer string-pool table. It cannot answer "what text
   does string-pool id 535 map to" at all; the id space it uses (7-digit cash
   item ids) is structurally disjoint from the 3–4 digit `StringPool` ids
   `NoticeFailReason` passes to `StringPool::GetString`. Dead end.
2. **Live atlas-data REST.** Confirmed by direct code read: atlas-data's
   String.wz ingester (`item/string_registry.go`) only walks numeric-named
   *item-id* child nodes of `Cash.img`/`Consume.img`/etc. and captures a
   single `name` leaf keyed by that item id; its only string route
   (`/data/item-strings/{itemId}`) explicitly filters `item_id >= 1000000`.
   There is no route, and no ingested table, that accepts an arbitrary small
   `StringPool` id (534, 535, 547, …) and returns text. Dead end.
3. **IDA `StringPoolStrings` enum (not raw WZ, but the closest available
   substitute).** The gms_v83 IDB (session `41f13e0d`) has a local enum type
   `StringPoolStrings` whose members are named `SP_<id>_<ABBREVIATED_TEXT>`
   (e.g. `SP_587_D_MAPLEPOINTS`) — Hex-Rays auto-resolves this name whenever a
   raw integer literal is passed **directly** as the id argument to
   `StringPool::GetString`/`GetStringW` (whose prototype parameter is typed
   `StringPoolStrings`), even though the enum's ~6000+ members cannot be bulk
   -enumerated through the available type-query API (confirmed: `type_query`
   returns `member_count: 0` even for a control COM enum, `tagBINDSTRING`,
   that is known to have members — an API limitation, not evidence the enum
   is empty). `NoticeFailReason` itself stores each case's id into a variable
   before calling `GetString`, so Hex-Rays can't resolve a name there — the
   raw literals in the table below are unaffected. But `find(type=immediate)`
   over the whole binary for each target id, followed by decompiling any
   second occurrence, surfaced two cases where the *same* id is passed as a
   **direct literal** to `GetString` elsewhere in the client, which forced
   Hex-Rays to resolve the friendly name — see the `verified (cross-decompile)`
   rows below. This is real, citable evidence (not ordinal inference), so
   those two rows are promoted. The other 49 `aligned` ids either have no
   second literal occurrence anywhere in the 10MB binary, or their only other
   occurrences are unrelated numeric coincidences (checked and rejected:
   536/540/544/552/555/557/583/etc. recur as item-category ids, UI window
   coordinates, or unrelated mode-byte comparisons, not `GetString` calls) —
   those rows are left `aligned`, unchanged.

| Key | Byte | StringPool id | Conf. | Evidence (case body addr) |
|---|---|---|---|---|
| REQUEST_TIMED_OUT | 0xA3 | 534 | aligned | `0x47c195` |
| NOT_ENOUGH_CASH | 0xA5 | 535 | **verified (cross-decompile)** | `0x47c1ab`; independently confirmed at `0x5c3813` — `CITCBidAuctionDlg` bid-confirm handler (`sub_5C373E`, gms_v83) calls `StringPool::GetString(Instance, &v15, SP_535_YOU_DONT_HAVE_ENOUGH_CASH)` directly when the account balance is insufficient to place the bid. The enum member name `SP_535_YOU_DONT_HAVE_ENOUGH_CASH` matches `NOT_ENOUGH_CASH` |
| CANNOT_GIFT_WHEN_UNDERAGE | 0xA6 | 536 | aligned | `0x47c1c1` |
| EXCEEDED_GIFT_LIMIT | 0xA7 | 537 | aligned | `0x47c1d7` |
| CANNOT_GIFT_TO_OWN_ACCOUNT | 0xA8 | 4537 | aligned | `0x47c57a` |
| INCORRECT_NAME | 0xA9 | 4538 | **verified (cross-decompile)** | `0x47c590`; independently confirmed at `0x46ecc4` — `CCashShop::OnGiftMateInfoResult` (gms_v83) calls `StringPool::GetString(Instance, &v42, SP_4538_PLEASE_CONFIRM_WHETHER_R_NTHE_CHARACTERS_NAME_IS_CORRECT)` directly when the gift recipient lookup returns nothing (`CInPacket::Decode1(arg0) == 0`). The enum member name matches `INCORRECT_NAME`; notably this second occurrence is inside `CCashShop` itself, not an unrelated class |
| CANNOT_GIFT_GENDER_RESTRICTION | 0xAA | 4539 | aligned | `0x47c5a6` |
| CANNOT_GIFT_RECIPIENT_INVENTORY_FULL | 0xAB | 4540 | aligned | `0x47c5b9` |
| EXCEEDED_CASH_ITEM_LIMIT | 0xAC | 538 | aligned | `0x47c1ed` |
| INCORRECT_NAME_OR_GENDER_RESTRICTION | 0xAD | 539 | aligned | `0x47c203` |
| INVALID_COUPON_CODE | 0xB0 | 547 | aligned (block-anchored, C) | `0x47c2b3` |
| COUPON_EXPIRED | 0xB2 | 543 | aligned (block-anchored, C) | `0x47c25b` |
| COUPON_ALREADY_USED | 0xB3 | 542 | aligned (block-anchored, C) | `0x47c245` |
| COUPON_INTERNET_CAFE_RESTRICTION | 0xB4 | 544 | aligned (block-anchored, C) | `0x47c271` |
| INTERNET_CAFE_COUPON_ALREADY_USED | 0xB5 | 545 | aligned (block-anchored, C) | `0x47c287` |
| INTERNET_CAFE_COUPON_EXPIRED | 0xB6 | 546 | aligned (block-anchored, C) | `0x47c29d` |
| COUPON_NOT_REGISTERED | 0xB7 | 3274 | aligned (block-anchored, C) | `0x47c3d1` |
| COUPON_GENDER_RESTRICTION | 0xB8 | 541 | aligned (block-anchored, C) | `0x47c22f` |
| COUPON_CANNOT_BE_GIFTED | 0xB9 | 551 | aligned (block-anchored, C) | `0x47c2c9` |
| COUPON_ONLY_FOR_MAPLE_STORY | 0xBA | 552 | aligned (block-anchored, C) | `0x47c2df` |
| INVENTORY_FULL | 0xBB | 853 | aligned | `0x47c2f5` |
| NOT_AVAILABLE_FOR_PURCHASE | 0xBC | 553 | aligned | `0x47c30b` |
| CANNOT_GIFT_INVALID_NAME_OR_GENDER | 0xBD | 554 | aligned | `0x47c321` |
| CHECK_NAME_OF_RECEIVER | 0xBE | 583 | aligned | `0x47c337` |
| NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR | 0xBF | 2505 | aligned | `0x47c34d` |
| OUT_OF_STOCK | 0xC0 | 2506 | aligned | `0x47c363` |
| EXCEEDED_SPENDING_LIMIT | 0xC1 | 555 | aligned | `0x47c379` |
| NOT_ENOUGH_MESOS | 0xC2 | 5599 | aligned | `0x47c38f` |
| CASH_SHOP_NOT_AVAILABLE_DURING_BETA | 0xC3 | 556 | **verified (anchor B)** | `0x47c3a5` |
| INVALID_BIRTHDAY | 0xC4 | 514 | aligned | `0x47c3bb` |
| ONLY_AVAILABLE_TO_USERS_BUYING | 0xC7 | 3275 | aligned | `0x47c3e7` |
| ALREADY_APPLIED | 0xC8 | 3558 | aligned | `0x47c3fd` |
| DAILY_PURCHASE_LIMIT | 0xCD | 540 | aligned | `0x47c219` |
| COUPON_USAGE_LIMIT | 0xD0 | 4457 | aligned | `0x47c413` |
| COUPON_SYSTEM_AVAILABLE_SOON | 0xD2 | inline literal | **verified (anchor A)** | `0x47c429`, string `aTheCouponSyste` @ `0xaf2328` |
| FIFTEEN_DAY_LIMIT | 0xD3 | 3749 | aligned | `0x47c446` |
| NOT_ENOUGH_GIFT_TOKENS | 0xD4 | 3992 | aligned | `0x47c45c` |
| CANNOT_SEND_TECHNICAL_DIFFICULTIES | 0xD5 | 3993 | aligned | `0x47c472` |
| CANNOT_GIFT_ACCOUNT_AGE | 0xD6 | 3994 | aligned | `0x47c488` |
| CANNOT_GIFT_PREVIOUS_INFRACTIONS | 0xD7 | 3995 | aligned | `0x47c49e` |
| CANNOT_GIFT_AT_THIS_TIME | 0xD8 | 3996 | aligned | `0x47c4b4` |
| CANNOT_GIFT_LIMIT | 0xD9 | 3997 | aligned | `0x47c4ca` |
| CANNOT_GIFT_TECHNICAL_DIFFICULTIES | 0xDA | 3998 | aligned | `0x47c4e0` |
| CANNOT_TRANSFER_UNDER_LEVEL_TWENTY | 0xDB | 4002 | aligned | `0x47c4f6` |
| CANNOT_TRANSFER_TO_SAME_WORLD | 0xDC | 4005 | aligned | `0x47c50c` |
| CANNOT_TRANSFER_TO_NEW_WORLD | 0xDD | 4006 | aligned | `0x47c522` |
| CANNOT_TRANSFER_OUT | 0xDE | 4007 | aligned | `0x47c538` |
| CANNOT_TRANSFER_NO_EMPTY_SLOTS | 0xDF | 4008 | aligned | `0x47c54e` |
| EVENT_ENDED_OR_CANT_BE_FREELY_TESTED | 0xE0 | 4020 | aligned | `0x47c564` |
| CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS | 0xE6 | 5063 | aligned | `0x47c5cc` |
| PLEASE_TRY_AGAIN | 0xE7 | 5064 | aligned | `0x47c5df` |
| CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN | 0xE8 | 5573 | aligned | `0x47c5f2` |
| CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN | 0xE9 | 5574 | aligned | `0x47c605` |
| UNKNOWN_ERROR | *(any unlisted byte)* | 557 | verified | default case `0x47c618` |

**Row-count self-check:** 53 explicit table rows + `UNKNOWN_ERROR` = 54,
equal to the 54 `CashShopOperationError*` constants and to the 53 explicit
switch cases + default. ✅

**Reserved reason bytes that are NOT notices** (must not be sent as a generic
coupon error — they change client state):

| Byte | Behaviour | Evidence |
|---|---|---|
| 0xA2 (162) | `NoticeFailReason` (default text) **then** `CCashShop::SendTransferFieldPacket` — kicks the player out of the cash shop | `OnCashItemResUseCouponFailed` @ `0x47a801`, `0x47a810`, `0x47a817` |
| 0xA4 (164) | same as 0xA2 | `0x47a801` |
| 0xB1 (177) | notice `SP_548_YOU_ARE_DISCONNECTED…WRONG_COUPON_NUM` **then** `CCashShop::OnStatusExit` | `0x47a822`-`0x47a84a` |

### `USE_COUPON` request body

**Not a `CASHSHOP_OPERATION` arm.** The coupon submission is opcode
**`0xE6` (230, registry op `COUPON_CODE`)**, built by
`CCashShop::OnStatusCoupon` @ `0x4710e8` (named this pass; was `sub_4710E8` —
the name matches the registry's declared `fname`). There is **no mode byte**.

Gate: `if ([this+40h] == 0) NoticeFailReason(0C3h); return;` @ `0x4710fb`;
`if ([this+18h] != 0) return;` (a send is already in flight) @ `0x47110f`.
The code is read from a modal dialog by
`CCashShop::ShowCouponInputDlg` @ `0x471240` (named this pass; was
`sub_471240`), then `TrimRight`/`TrimLeft`'d @ `0x471145`/`0x47114c`, and an
empty result is rejected client-side with the literal
`"Please enter the coupon code."` @ `0x47116f` — the packet is never sent.

| # | Field | Read | Evidence |
|---|---|---|---|
| 1 | *(secondary string — always empty in v83)* | `COutPacket::EncodeStr` | ctor `0x47118e`; `lea eax,[ebp+var_14]` @ `0x471196` → `EncodeStr` @ `0x4711ab`. `var_14` is zero-initialised @ `0x47111e` and is the second out-parameter of `ShowCouponInputDlg`, which never writes it (only `arg0` receives `CCtrlEdit::GetText`, @ `0x47131a`). It therefore always encodes as a zero-length string. Purpose unknown / unverified. |
| 2 | coupon code (`str`) | `COutPacket::EncodeStr` | `lea eax,[ebp+var_10]` @ `0x4711b3` → `EncodeStr` @ `0x4711c4`. `var_10` is the trimmed dialog text. |
| 3 | *(third string — unreachable in v83)* | `COutPacket::EncodeStr` | guarded by `if (var_14 && *var_14)` @ `0x4711c9`-`0x4711d2`; since field 1 is always empty this branch is dead in v83. Encodes `var_18`, also never written. |

`CClientSocket::SendPacket` @ `0x4711f7`; the in-flight flag `[this+18h]` is
set to 1 @ `0x4711ff`.

**Row-count self-check:** 3 `EncodeStr` call sites in the send path → 3 rows.
No `Encode1`/`Encode4` anywhere in the body. ✅

**Practical wire shape for v83:** `[opcode 0xE6][str ""][str <code>]`.

### Blocking answer 1 — is `UseCouponDone.maplePoint` a DELTA or an ABSOLUTE post-award balance?

**DELTA — the amount awarded by this coupon. It is NOT an absolute balance.**

`CCashShop::OnCashItemResUseCouponDone` @ `0x479d8a`, read order:

| # | Field | Read | Evidence |
|---|---|---|---|
| 1 | cash-item count `n` | `Decode1` | `0x479dc8` |
| 2 | `n` × 55-byte cash-item records (one contiguous blob) | `DecodeBuffer(a2, buf, 55*n)` | `0x479df2` (`55 * v4`), `0x479df6` |
| 3 | **maplePoint** | `Decode4` | `0x479efb` |
| 4 | inventory-item count `m` | `Decode4` | `0x479f03` |
| 5 | `m` × 8-byte records (`{u32 pos/qty, u32 itemId}`) | `DecodeBuffer(a2, buf, 8*m)` | `0x479f2f` (`8 * v18`), `0x479f33` |
| 6 | **mesos** | `Decode4` | `0x47a04e` |

The maplePoint value (`v68`) is used only here:
`if (v68) { ZXString<char>::Format(&v62, StringPool::GetString(…,
SP_587_D_MAPLEPOINTS), v68); … }` @ `0x47a0b6`-`0x47a0df`, and the result is
concatenated into the message headed by `SP_585_YOU_HAVE_RECEIVED`
(@ `0x47a200`) and tailed by `SP_5611__R_NUSING_THE_COUPON` (@ `0x47a1d2` /
`0x47a344`).

Two independent reasons this cannot be an absolute balance: (a) the whole
field is skipped when it is zero (`if (v68)`), which would be wrong for a
balance; (b) it is rendered inside a "You have received … using the coupon"
sentence, alongside `mesos` (`SP_588_D_MESOS`, @ `0x47a179`) which is
unambiguously an award amount. The client never writes this value into any
balance field — the balance is refreshed separately by
`CCashShop::OnQueryCashResult` @ `0x478f81`.

**Action for Task 22:** the design's "absolute post-award balance" assumption
is **wrong**. `CouponRedeemedBody.MaplePoints` must carry the amount granted
by the redemption, and the comment `// absolute post-award balance` in
`design.md` must be corrected.

### Blocking answer 2 — does the client echo the submitted code back on failure?

**No. No extra arm is needed.**

`CCashShop::OnCashItemResUseCouponFailed` @ `0x47a7db` reads exactly one byte
and nothing else: `v3 = CInPacket::Decode1(a2)` @ `0x47a7eb`. There is no
`DecodeStr`, no `Decode4`, no length-prefixed read anywhere in the function.
Everything after the single byte is client-local UI (`NoticeFailReason`,
and for `0xA2`/`0xA4`/`0xB1` the state changes tabled above). The existing
already-verified `USE_COUPON_FAILED` arm (mode `0x5C` on v83 per
`docs/tasks/task-183-cashshop-result-family/arm-catalog.md`) is sufficient.


---

## gms_v84

- Serverbound `CASHSHOP_OPERATION` opcode: **235 / `0xEB`**.
- Serverbound `COUPON_CODE` opcode: **236 / `0xEC`** — **REGISTRY BUG**, see below.
- Send sites enumerated by byte search for `68 EB 00 00 00`: 26 hits, 25 of them
  inside the `CCashShop` code region `0x46e277`-`0x47640d` (the 26th, `0x425e5d`,
  is outside `CCashShop` and is not a packet-ctor site).
- Clientbound dispatcher: `CCashShop::OnCashItemResult` @ `0x47c291`.
- Failure-reason sink: `CCashShop::NoticeFailReason` @ `0x47f318`
  (**named this pass**; was `sub_47F318`).

### Registry bug — `COUPON_CODE` opcode

`docs/packets/registry/gms_v84.yaml` declares
`op: COUPON_CODE / direction: serverbound / opcode: 230`. The only real
coupon send in the v84 client is `COutPacket::COutPacket(&pkt, 236)` @
`0x473c84` inside `CCashShop::OnStatusCoupon` @ `0x473bde` (**named this
pass**; was `sub_473BDE`). 236 = `0xEC` = `CASHSHOP_OPERATION`(235) + 1, which
matches every other version's +1 relation. The `68 E6 00 00 00` byte hit at
`0x473fd1` that would support "230" is inside `sub_473F02`, a dialog-layout
constructor (`230` there is a pixel coordinate passed to a `CreateDlg`-style
vtable call at `0x473fe0`) — it is not a packet. **Fix the registry to 236.**

### Serverbound `operations` (CashShopOperationHandle)

25 `COutPacket(…, 0xEB)` sites → **20 distinct mode bytes**. The 19 v83 modes
are byte-identical; one mode (**72**) is new in v84 and has no Atlas key.

| Key | Mode | Evidence |
|---|---|---|
| BUY | 3 | `CCashShop__OnBuy_send_0xEB_op3` ctor @ `0x46e277`, `push 3` @ `0x46e287`; `CCashShop::OnBuy` ctor @ `0x47098c`, `push 3` @ `0x470999` |
| GIFT | 4 | `CCashShop::SendGiftsPacket` ctor @ `0x47266c`, `push 4` @ `0x47268c`; `CCashShop__OnGiftMateInfoResult_recv_0x14E` ctor @ `0x4718ff`; unnamed GiftWishItem-equivalent ctor @ `0x475aac`, `push 4; pop edi` @ `0x475abc` |
| SET_WISHLIST | 5 | `CCashShop::OnSetWish` ctor @ `0x473944`, `push 5` @ `0x473955`; OnRemoveWish-equivalent ctor @ `0x473a63`, `push 5` @ `0x473a74` |
| INCREASE_INVENTORY | 6 | OnExItemSlot-equivalent ctor @ `0x46e8fd`, `push 6` @ `0x46e90a`; OnBuySlotInc-equivalent computed arm @ `0x473457` |
| INCREASE_STORAGE | 7 | `CCashShop::OnIncTrunkCount` ctor @ `0x46ed56`, `push 7` @ `0x46ed63` |
| INCREASE_CHARACTER_SLOT | 8 | `CCashShop::OnIncCharacterSlotCount` ctor @ `0x46ef29`, `push 8` @ `0x46ef36` |
| ENABLE_EQUIP_SLOT | 9 | `CCashShop::OnEnableEquipSlotExt` ctor @ `0x46f0ca`, `push 9` @ `0x46f0d7` |
| MOVE_FROM_CASH_INVENTORY | 13 | `CCashShop::OnMoveCashItemLtoS` ctor @ `0x47529e`, `push 0Dh` @ `0x4752ab` |
| MOVE_TO_CASH_INVENTORY | 14 | `CCashShop::OnMoveCashItemStoL` ctor @ `0x4753e7`, `push 0Eh` @ `0x4753f6` |
| REBATE_LOCKER_ITEM | 26 | `CCashShop::OnRebateLockerItem` ctor @ `0x46e566`, `push 1Ah` @ `0x46e573` |
| BUY_COUPLE | 29 | `CCashShop::OnBuyCouple` ctor @ `0x472d15`, `push 1Dh` @ `0x472d22` |
| BUY_PACKAGE | 30 | OnBuyPackage-equivalent ctor @ `0x470f93`, `push 1Eh` @ `0x470fa0` |
| BUY_OTHER_PACKAGE | 31 | OnGiftPackage-equivalent ctor @ `0x4729e7`, `push 1Fh` @ `0x4729f4` |
| BUY_NORMAL | **32** | `CCashShop::OnBuyNormal` ctor @ `0x471227`, `push 20h` @ `0x471234` — **TEMPLATE BUG**: `template_gms_84_1.json` says `20` (same transcription error as gms_83) |
| APPLY_WISHLIST | 33 | ApplyWishListEvent-equivalent ctor @ `0x475c55`, `push 21h` @ `0x475c66` |
| BUY_FRIENDSHIP | 35 | `CCashShop::OnBuyFriendship` ctor @ `0x473774`, `push 23h` @ `0x473781` |
| GET_PURCHASE_RECORD | 40 | `CCashShop::RequestCashPurchaseRecord` ctor @ `0x46e30d`, `push 28h` @ `0x46e31e` |
| BUY_NAME_CHANGE | 46 | `CCashShop::SendBuyNameChangeItemPacket` ctor @ `0x475f8d`, `push 2Eh` @ `0x475f9d` |
| BUY_WORLD_TRANSFER | 49 | `CCashShop::SendBuyTransferWorldItemPacket` ctor @ `0x47615b`, `push 31h` @ `0x47616c` |
| *(no Atlas key — see note)* | **72** | `CCashShop__SendCashItemRequest_mode72` @ `0x4762b4` (**named this pass**; was `sub_4762B4`): `COutPacket(…, 235)`, `Encode1(0x48)`, `Encode1(mask == 2)`, `Encode4(mask)`. `mask` is a 3-bit flag set: bit0/1/2 are set when the corresponding cash-locker list already holds ≥ 450 entries (`sub_417012(...) >= 450`, three times). Gated by StringPool 5653/5654 confirmation dialogs. |

**Mode 72 naming.** Atlas has no `operations` key for this arm on any version.
Its semantics ("some locker list is at the 450 cap") are readable but the
matching clientbound arm was not identified in this pass, so no key name is
asserted here — naming it requires pairing it with its result arm in
`docs/tasks/task-183-cashshop-result-family/arm-catalog.md`. Recorded as
**mode 72, key unknown / unverified**.

**Row-count self-check:** 25 in-region opcode-construction sites → 20 distinct
mode bytes → 20 rows. Template declares 19 keys ⇒ the template is missing the
mode-72 arm entirely, in addition to the `BUY_NORMAL` value bug.

### `errors` enum

`CCashShop::NoticeFailReason` @ `0x47f318`:
`add eax, 0FFFFFF54h` (bias −172) @ `0x47f31e`, `cmp eax, 46h` @ `0x47f323`,
`ja def_47F32C` @ `0x47f326`, annotated **"switch 71 cases"** with
**"default case, cases 173,183,184,186,206,207,210-213,215,216,218,234-238"**.

**The v84 enum is v83's shifted by exactly +9, proven structurally, not
assumed.** The bias moved 163 → 172 (+9) and the *default-case set* maps
one-for-one under the same +9:
`{164,174,175,177,197,198,201-204,206,207,209,225-229} + 9 =
{173,183,184,186,206,207,210-213,215,216,218,234-238}` — an exact set
equality, with the same case count (71) and the same range (`0x46`). Three
independent call-site anchors confirm it:

| Anchor | v83 | v84 | Evidence |
|---|---|---|---|
| coupon "wrong number → disconnect" | 177 (`0xB1`) | 186 (`0xBA`) | `OnCashItemResUseCouponFailed` @ `0x47d9c0` (`if (v3 == 186)`), StringPool 551 (v83 used 548) |
| "transfer field" pair | 162, 164 | 171, 173 | same function @ `0x47d99f` (`v2 == 171 \|\| v2 == 173`) |
| cash-shop-disabled gate | 195 (`0xC3`) | 204 (`0xCC`) | `CCashShop::OnStatusCoupon` @ `0x473bfb` (`sub_47F318(204)`) |

| Key | Byte | Evidence |
|---|---|---|
| REQUEST_TIMED_OUT | 0xAC (172) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| NOT_ENOUGH_CASH | 0xAE (174) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_WHEN_UNDERAGE | 0xAF (175) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| EXCEEDED_GIFT_LIMIT | 0xB0 (176) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_TO_OWN_ACCOUNT | 0xB1 (177) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| INCORRECT_NAME | 0xB2 (178) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_GENDER_RESTRICTION | 0xB3 (179) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_RECIPIENT_INVENTORY_FULL | 0xB4 (180) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| EXCEEDED_CASH_ITEM_LIMIT | 0xB5 (181) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| INCORRECT_NAME_OR_GENDER_RESTRICTION | 0xB6 (182) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| INVALID_COUPON_CODE | 0xB9 (185) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| COUPON_EXPIRED | 0xBB (187) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| COUPON_ALREADY_USED | 0xBC (188) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| COUPON_INTERNET_CAFE_RESTRICTION | 0xBD (189) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| INTERNET_CAFE_COUPON_ALREADY_USED | 0xBE (190) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| INTERNET_CAFE_COUPON_EXPIRED | 0xBF (191) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| COUPON_NOT_REGISTERED | 0xC0 (192) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| COUPON_GENDER_RESTRICTION | 0xC1 (193) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| COUPON_CANNOT_BE_GIFTED | 0xC2 (194) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| COUPON_ONLY_FOR_MAPLE_STORY | 0xC3 (195) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| INVENTORY_FULL | 0xC4 (196) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| NOT_AVAILABLE_FOR_PURCHASE | 0xC5 (197) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_INVALID_NAME_OR_GENDER | 0xC6 (198) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CHECK_NAME_OF_RECEIVER | 0xC7 (199) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR | 0xC8 (200) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| OUT_OF_STOCK | 0xC9 (201) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| EXCEEDED_SPENDING_LIMIT | 0xCA (202) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| NOT_ENOUGH_MESOS | 0xCB (203) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CASH_SHOP_NOT_AVAILABLE_DURING_BETA | 0xCC (204) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| INVALID_BIRTHDAY | 0xCD (205) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| ONLY_AVAILABLE_TO_USERS_BUYING | 0xD0 (208) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| ALREADY_APPLIED | 0xD1 (209) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| DAILY_PURCHASE_LIMIT | 0xD6 (214) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| COUPON_USAGE_LIMIT | 0xD9 (217) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| COUPON_SYSTEM_AVAILABLE_SOON | 0xDB (219) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| FIFTEEN_DAY_LIMIT | 0xDC (220) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| NOT_ENOUGH_GIFT_TOKENS | 0xDD (221) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_SEND_TECHNICAL_DIFFICULTIES | 0xDE (222) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_ACCOUNT_AGE | 0xDF (223) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_PREVIOUS_INFRACTIONS | 0xE0 (224) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_AT_THIS_TIME | 0xE1 (225) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_LIMIT | 0xE2 (226) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_TECHNICAL_DIFFICULTIES | 0xE3 (227) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_UNDER_LEVEL_TWENTY | 0xE4 (228) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_TO_SAME_WORLD | 0xE5 (229) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_TO_NEW_WORLD | 0xE6 (230) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_OUT | 0xE7 (231) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_NO_EMPTY_SLOTS | 0xE8 (232) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| EVENT_ENDED_OR_CANT_BE_FREELY_TESTED | 0xE9 (233) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS | 0xEF (239) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| PLEASE_TRY_AGAIN | 0xF0 (240) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN | 0xF1 (241) | v83 case + 9 (jump-table bias/default-set identity, see above) |
| CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN | 0xF2 (242) | v83 case + 9 (jump-table bias/default-set identity, see above) |

**Row-count self-check:** 53 rows + `UNKNOWN_ERROR` (default case @ `0x47f7b6`)
= 54, equal to the 54 Atlas constants and to 71 switch cases − 18 defaults + 1
default. ✅

Reserved non-notice reason bytes: **171** and **173** (notice + kick out of
the cash shop, `CCashShop::SendTransferFieldPacket` @ `0x47f2a6`, named this
pass) and **186** (coupon-specific disconnect + `OnStatusExit`).

### `USE_COUPON` request body

Opcode **236 / `0xEC`** (`COUPON_CODE`), built by `CCashShop::OnStatusCoupon`
@ `0x473bde`. No mode byte. Structurally identical to v83.

| # | Field | Read | Evidence |
|---|---|---|---|
| 1 | target character id (`str`) — empty on the plain-redeem path | `EncodeStr` | ctor `0x473c84`; `0x473c99` → `EncodeStr` @ `0x473ca1` (`v14`, second out-param of `CCashShop::ShowCouponInputDlg` @ `0x473d36`, named this pass) |
| 2 | coupon code (`str`) | `EncodeStr` | `0x473cb2` → `EncodeStr` @ `0x473cba` (`v15`, trimmed @ `0x473c3b`/`0x473c42`) |
| 3 | third string — reachable only when field 1 is non-empty | `EncodeStr` | guard `if (v14 && *v14)` @ `0x473cc4`-`0x473cc6`, `EncodeStr` @ `0x473cde` |

`CClientSocket::SendPacket` @ `0x473ced`. Empty-code rejection with the literal
`"Please enter the coupon code."` @ `0x473c6a`.

**Row-count self-check:** 3 `EncodeStr` sites → 3 rows. ✅

**Prototype-name trap (do not trust the symbol here).** In this IDB the
function at `0x47d979` is listed by `func_query` as
`CCashShop::OnCashItemResUseCouponFailed` but Hex-Rays renders its header as
`CCashShop::OnCashItemResGiftDone` (a stale applied prototype). The *body* is
byte-for-byte the v83 `UseCouponFailed` shape (single `Decode1` @ `0x47d989`,
`171/173` transfer-field pair, `186` disconnect). `OnCashItemResGiftDone`
actually lives at `0x47d9f4`.

---

## gms_v87

- Serverbound `CASHSHOP_OPERATION` opcode: **242 / `0xF2`**.
- Serverbound `COUPON_CODE` opcode: **243 / `0xF3`** (registry correct).
- Send sites enumerated by byte search for `68 F2 00 00 00`: 25 hits inside
  the `CCashShop` region `0x475af7`-`0x47e241`.
- Clientbound dispatcher: `CCashShop::OnCashItemResult` @ `0x4847f4`.
- Failure-reason sink: `CCashShop::NoticeFailReason` @ `0x48795b`.

### Serverbound `operations` (CashShopOperationHandle)

25 sites → **20 distinct mode bytes**. `template_gms_87_1.json` declares
**zero** serverbound operations, so every row below is new to the template.

| Key | Mode | Evidence |
|---|---|---|
| BUY | 3 | `CCashShop::SendBuyAvatarPacket` ctor @ `0x475af7`, `push 3` @ `0x475b07`; `CCashShop::OnBuy` ctor @ `0x47822a`, `push 3` @ `0x478237` |
| GIFT | 4 | `CCashShop::SendGiftsPacket` ctor @ `0x47a39e`, `push 4` @ `0x47a3b2`; `CCashShop::OnGiftMateInfoResult` `push 4; lea ecx,[ebp-4Ch]` @ `0x479950`; `CCashShop::GiftWishItem` `push 4; pop edi` @ `0x47d8f0` |
| SET_WISHLIST | 5 | `CCashShop::OnSetWish` ctor @ `0x47b684`, `push 5` @ `0x47b691`; `CCashShop::OnRemoveWish` ctor @ `0x47b858`, `push 5` @ `0x47b86b` |
| INCREASE_INVENTORY | 6 | `CCashShop::OnExItemSlot` ctor @ `0x47618d`, `push 6` @ `0x47619a`; `CCashShop::OnBuySlotInc` computed arm @ `0x47b19a` |
| INCREASE_STORAGE | 7 | `CCashShop::OnIncTrunkCount` ctor @ `0x476506`, `push 7` @ `0x476513` |
| INCREASE_CHARACTER_SLOT | 8 | `CCashShop::OnIncCharacterSlotCount` ctor @ `0x4766d9`, `push 8` @ `0x4766e6` |
| ENABLE_EQUIP_SLOT | 9 | `CCashShop::OnEnableEquipSlotExt` ctor @ `0x47687a`, `push 9` @ `0x476887` |
| MOVE_FROM_CASH_INVENTORY | 13 | `CCashShop::OnMoveCashItemLtoS` ctor @ `0x47d0ce`, `push 0Dh` @ `0x47d0db` |
| MOVE_TO_CASH_INVENTORY | 14 | `CCashShop::OnMoveCashItemStoL` ctor @ `0x47d217`, `push 0Eh` @ `0x47d226` |
| REBATE_LOCKER_ITEM | 26 | unnamed OnRebateLockerItem-equivalent, ctor @ `0x475df9`, `push 1Ah` @ `0x475e06` |
| BUY_COUPLE | 29 | `CCashShop::OnBuyCouple` ctor @ `0x47aa58`, `push 1Dh` @ `0x47aa65` |
| BUY_PACKAGE | 30 | `CCashShop::OnBuyPackage` ctor @ `0x478b8f`, `push 1Eh` @ `0x478b9c` |
| BUY_OTHER_PACKAGE | 31 | `CCashShop::OnGiftPackage` ctor @ `0x47a72a`, `push 1Fh` @ `0x47a737` |
| BUY_NORMAL | 32 | `CCashShop::OnBuyNormal` ctor @ `0x478eb6`, `push 20h` @ `0x478ec3` |
| APPLY_WISHLIST | 33 | `CCashShop::ApplyWishListEvent` ctor @ `0x47da89`, `push 21h` @ `0x47da9a` |
| BUY_FRIENDSHIP | 35 | `CCashShop::OnBuyFriendship` ctor @ `0x47b4b7`, `push 23h` @ `0x47b4c4` |
| GET_PURCHASE_RECORD | **42** | `CCashShop::RequestCashPurchaseRecord` ctor @ `0x475b9e`, `push 2Ah` @ `0x475baf` — **shifted from 40** on v83/v84 |
| BUY_NAME_CHANGE | **48** | `CCashShop::SendBuyNameChangeItemPacket` ctor @ `0x47ddc1`, `push 30h` @ `0x47ddd1` — **shifted from 46** |
| BUY_WORLD_TRANSFER | **51** | `CCashShop::SendBuyTransferWorldItemPacket` ctor @ `0x47df8f`, `push 33h` @ `0x47dfa0` — **shifted from 49** |
| *(no Atlas key)* | **74** | unnamed function preceding `CCashShop::SendChangeMaplePoint` (`0x47e2b9`); ctor @ `0x47e241`, `push 4Ah` @ `0x47e24e`. The v84 mode-72 arm shifted by +2, same as the 40→42 / 46→48 / 49→51 group. Key unknown / unverified for the same reason as v84. |

**Row-count self-check:** 25 sites → 20 distinct modes → 20 rows. ✅

### `errors` enum

`CCashShop::NoticeFailReason` @ `0x48795b`: `add eax, 0FFFFFF4Eh` (bias −178)
@ `0x487961`, `cmp eax, 47h` @ `0x487966`, annotated **"switch 72 cases"**
with **"default case, cases 179,189,190,192,212,213,216-219,221,222,224,
240-244"**.

**v87 = v83 shifted by exactly +15, plus one extra reason at the top.**
Structural proof: bias 163 → 178 (+15); the default-case set maps one-for-one
under +15 (`{164,174,175,177,197,198,201-204,206,207,209,225-229} + 15 =
{179,189,190,192,212,213,216-219,221,222,224,240-244}`, exact set equality).
The case count rises 71 → 72 and the range `0x46` → `0x47`, i.e. **one new
reason value 249 (`0xF9`)** was appended beyond v83's last (233 + 15 = 248).
Call-site anchor: `CCashShop::OnStatusCoupon` @ `0x47b9f1` calls
`NoticeFailReason(this, 210)`; v83's gate was 195; 195 + 15 = 210. ✅

| Key | Byte | Evidence |
|---|---|---|
| REQUEST_TIMED_OUT | 0xB2 (178) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| NOT_ENOUGH_CASH | 0xB4 (180) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_WHEN_UNDERAGE | 0xB5 (181) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| EXCEEDED_GIFT_LIMIT | 0xB6 (182) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_TO_OWN_ACCOUNT | 0xB7 (183) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| INCORRECT_NAME | 0xB8 (184) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_GENDER_RESTRICTION | 0xB9 (185) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_RECIPIENT_INVENTORY_FULL | 0xBA (186) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| EXCEEDED_CASH_ITEM_LIMIT | 0xBB (187) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| INCORRECT_NAME_OR_GENDER_RESTRICTION | 0xBC (188) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| INVALID_COUPON_CODE | 0xBF (191) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| COUPON_EXPIRED | 0xC1 (193) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| COUPON_ALREADY_USED | 0xC2 (194) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| COUPON_INTERNET_CAFE_RESTRICTION | 0xC3 (195) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| INTERNET_CAFE_COUPON_ALREADY_USED | 0xC4 (196) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| INTERNET_CAFE_COUPON_EXPIRED | 0xC5 (197) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| COUPON_NOT_REGISTERED | 0xC6 (198) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| COUPON_GENDER_RESTRICTION | 0xC7 (199) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| COUPON_CANNOT_BE_GIFTED | 0xC8 (200) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| COUPON_ONLY_FOR_MAPLE_STORY | 0xC9 (201) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| INVENTORY_FULL | 0xCA (202) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| NOT_AVAILABLE_FOR_PURCHASE | 0xCB (203) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_INVALID_NAME_OR_GENDER | 0xCC (204) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CHECK_NAME_OF_RECEIVER | 0xCD (205) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR | 0xCE (206) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| OUT_OF_STOCK | 0xCF (207) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| EXCEEDED_SPENDING_LIMIT | 0xD0 (208) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| NOT_ENOUGH_MESOS | 0xD1 (209) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CASH_SHOP_NOT_AVAILABLE_DURING_BETA | 0xD2 (210) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| INVALID_BIRTHDAY | 0xD3 (211) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| ONLY_AVAILABLE_TO_USERS_BUYING | 0xD6 (214) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| ALREADY_APPLIED | 0xD7 (215) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| DAILY_PURCHASE_LIMIT | 0xDC (220) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| COUPON_USAGE_LIMIT | 0xDF (223) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| COUPON_SYSTEM_AVAILABLE_SOON | 0xE1 (225) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| FIFTEEN_DAY_LIMIT | 0xE2 (226) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| NOT_ENOUGH_GIFT_TOKENS | 0xE3 (227) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_SEND_TECHNICAL_DIFFICULTIES | 0xE4 (228) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_ACCOUNT_AGE | 0xE5 (229) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_PREVIOUS_INFRACTIONS | 0xE6 (230) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_AT_THIS_TIME | 0xE7 (231) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_LIMIT | 0xE8 (232) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_GIFT_TECHNICAL_DIFFICULTIES | 0xE9 (233) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_UNDER_LEVEL_TWENTY | 0xEA (234) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_TO_SAME_WORLD | 0xEB (235) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_TO_NEW_WORLD | 0xEC (236) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_OUT | 0xED (237) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_TRANSFER_NO_EMPTY_SLOTS | 0xEE (238) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| EVENT_ENDED_OR_CANT_BE_FREELY_TESTED | 0xEF (239) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS | 0xF5 (245) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| PLEASE_TRY_AGAIN | 0xF6 (246) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN | 0xF7 (247) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN | 0xF8 (248) | v83 case + 15 (jump-table bias/default-set identity, see above) |
| *(no Atlas key — new in v87)* | 0xF9 (249) | `loc_487DFC`; the 54th explicit case, beyond v83's highest (`0xE9` + 15 = `0xF8`). Key unknown / unverified — Atlas has no 54th non-`UNKNOWN_ERROR` constant. |

**Row-count self-check:** 53 mapped rows + 1 new + `UNKNOWN_ERROR` (default
@ `0x487e10`) = 55, equal to 72 switch cases − 18 defaults + 1 default = 55. ✅
Atlas needs **one additional error constant** to cover v87 fully.

### `USE_COUPON` request body

Opcode **243 / `0xF3`**, built by `CCashShop::OnStatusCoupon` @ `0x47b9d4`.
No mode byte.

| # | Field | Read | Evidence |
|---|---|---|---|
| 1 | target character id (`str`) — empty on the plain-redeem path | `EncodeStr` | ctor `0x47ba7a`; `0x47ba8f` → `EncodeStr` @ `0x47ba97` (`v15`) |
| 2 | coupon code (`str`) | `EncodeStr` | `0x47baa8` → `EncodeStr` @ `0x47bab0` (`v16`, trimmed @ `0x47ba31`/`0x47ba38`) |
| 3 | third string — reachable only when field 1 is non-empty | `EncodeStr` | guard `if (v15 && *v15)` @ `0x47baba`-`0x47babc`, `EncodeStr` @ `0x47bad4` |

`CClientSocket::SendPacket` @ `0x47bae3`; in-flight flag set @ `0x47baeb`.

**Row-count self-check:** 3 `EncodeStr` sites → 3 rows. ✅

---

## gms_v92

**Status: PARTIAL.** The `errors` enum and the `USE_COUPON` body are fully
derived. The serverbound arm table is **not** complete — see the explicit gap
statement at the end of this section. The clientbound 57-arm table was **not**
re-derived in this pass — see the note below.

- Serverbound `CASHSHOP_OPERATION` opcode: **268 / `0x10C`**.
- Serverbound `COUPON_CODE` opcode: **269 / `0x10D`** (registry correct;
  confirmed by `COutPacket::COutPacket(&pkt, 0x10Du)` @ `0x484512`).
- Clientbound dispatcher: `CCashShop::OnCashItemResult` @ `0x495300`.
- Failure-reason sink: `CCashShop::NoticeFailReason` @ `0x491b50`
  (unnamed in this IDB as `sub_491B50`; identified through
  `CCashShop::OnCashItemResUseCouponFailed` @ `0x492f20`).

### `errors` enum — the enum is RENUMBERED to a 1-based scale

This is the single most important v92 finding. `sub_491B50` @ `0x491b50` is
`dec eax` @ `0x491b54` (bias **−1**, not −163), `cmp eax, 44h` @ `0x491b55`,
annotated **"switch 69 cases"** with **"default case, cases 2,12,13,15,35,36,
39-42,44,45,47"**.

`v92 reason = v83 reason − 0xA2 (162)`. Proof: the v83 default-case set
reduced by 162 is `{2,12,13,15,35,36,39-42,44,45,47,63-67}`; v92's default set
is exactly its first 13 members. Three call-site anchors:

| Anchor | v83 | v92 | Evidence |
|---|---|---|---|
| "transfer field" pair | 162, 164 | **0, 2** | `OnCashItemResUseCouponFailed` @ `0x492f43` (`case 0u: case 2u:`) → `CCashShop::SendTransferFieldPacket` @ `0x492f54` |
| coupon "wrong number → disconnect" | 177 | **15** | same function @ `0x492f66` (`if (v4 == 15)`), StringPool 566 |
| cash-shop-disabled gate | 195 | **33** | `sub_484430` (OnStatusCoupon) @ `0x484460` (`sub_491B50(this, 33)`) |

Two structural differences from v83 beyond the shift:
- offsets **63–67** are *explicit* cases in v92 (they are default cases in
  v83/v84/v87) → **five new reason values with no Atlas key**;
- offsets **70, 71** are outside the switch domain (`cmp eax, 44h` caps it at
  69) → `CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN` and
  `CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN` are **n-a on v92**.

| Key | Byte | Evidence |
|---|---|---|
| REQUEST_TIMED_OUT | 0x01 (1) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| NOT_ENOUGH_CASH | 0x03 (3) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_WHEN_UNDERAGE | 0x04 (4) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| EXCEEDED_GIFT_LIMIT | 0x05 (5) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_TO_OWN_ACCOUNT | 0x06 (6) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| INCORRECT_NAME | 0x07 (7) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_GENDER_RESTRICTION | 0x08 (8) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_RECIPIENT_INVENTORY_FULL | 0x09 (9) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| EXCEEDED_CASH_ITEM_LIMIT | 0x0A (10) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| INCORRECT_NAME_OR_GENDER_RESTRICTION | 0x0B (11) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| INVALID_COUPON_CODE | 0x0E (14) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| COUPON_EXPIRED | 0x10 (16) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| COUPON_ALREADY_USED | 0x11 (17) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| COUPON_INTERNET_CAFE_RESTRICTION | 0x12 (18) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| INTERNET_CAFE_COUPON_ALREADY_USED | 0x13 (19) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| INTERNET_CAFE_COUPON_EXPIRED | 0x14 (20) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| COUPON_NOT_REGISTERED | 0x15 (21) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| COUPON_GENDER_RESTRICTION | 0x16 (22) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| COUPON_CANNOT_BE_GIFTED | 0x17 (23) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| COUPON_ONLY_FOR_MAPLE_STORY | 0x18 (24) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| INVENTORY_FULL | 0x19 (25) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| NOT_AVAILABLE_FOR_PURCHASE | 0x1A (26) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_INVALID_NAME_OR_GENDER | 0x1B (27) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CHECK_NAME_OF_RECEIVER | 0x1C (28) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR | 0x1D (29) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| OUT_OF_STOCK | 0x1E (30) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| EXCEEDED_SPENDING_LIMIT | 0x1F (31) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| NOT_ENOUGH_MESOS | 0x20 (32) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CASH_SHOP_NOT_AVAILABLE_DURING_BETA | 0x21 (33) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| INVALID_BIRTHDAY | 0x22 (34) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| ONLY_AVAILABLE_TO_USERS_BUYING | 0x25 (37) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| ALREADY_APPLIED | 0x26 (38) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| DAILY_PURCHASE_LIMIT | 0x2B (43) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| COUPON_USAGE_LIMIT | 0x2E (46) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| COUPON_SYSTEM_AVAILABLE_SOON | 0x30 (48) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| FIFTEEN_DAY_LIMIT | 0x31 (49) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| NOT_ENOUGH_GIFT_TOKENS | 0x32 (50) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_SEND_TECHNICAL_DIFFICULTIES | 0x33 (51) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_ACCOUNT_AGE | 0x34 (52) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_PREVIOUS_INFRACTIONS | 0x35 (53) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_AT_THIS_TIME | 0x36 (54) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_LIMIT | 0x37 (55) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_GIFT_TECHNICAL_DIFFICULTIES | 0x38 (56) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_TRANSFER_UNDER_LEVEL_TWENTY | 0x39 (57) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_TRANSFER_TO_SAME_WORLD | 0x3A (58) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_TRANSFER_TO_NEW_WORLD | 0x3B (59) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_TRANSFER_OUT | 0x3C (60) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_TRANSFER_NO_EMPTY_SLOTS | 0x3D (61) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| EVENT_ENDED_OR_CANT_BE_FREELY_TESTED | 0x3E (62) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS | 0x44 (68) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| PLEASE_TRY_AGAIN | 0x45 (69) | v83 case - 0xA2 (jump-table `dec eax` bias + identical default set, see above) |
| CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN | *(absent)* | offset 70 falls outside this version's switch domain — no case, no default text row |
| CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN | *(absent)* | offset 71 falls outside this version's switch domain — no case, no default text row |
| *(no Atlas key — new in v92)* | 0x3F–0x43 (63–67) | five explicit cases at `loc_49208E`, `loc_4920A5`, `loc_4920BC`, `loc_4920D2`, `loc_4920E9`; default cases on v83/v84/v87. Keys unknown / unverified. |

**Row-count self-check:** 51 mapped rows (53 Atlas keys − 2 out-of-domain) + 5
new + `UNKNOWN_ERROR` (default @ `0x492100`) = 57 = 69 switch cases − 13
defaults + 1 default = 57. ✅

### `USE_COUPON` request body — the third string is GONE

Opcode **269 / `0x10D`**, built by `sub_484430` @ `0x484430` (the v92
`CCashShop::OnStatusCoupon`; unnamed in this IDB). No mode byte.

| # | Field | Read | Evidence |
|---|---|---|---|
| 1 | target character id (`str`) | `EncodeStr` | ctor `0x484512`; `0x48452a` → `EncodeStr` @ `0x484533` (`v9`, second out-param of `sub_480D40` @ `0x4844aa`) |
| 2 | coupon code (`str`) | `EncodeStr` | `0x484546` → `EncodeStr` @ `0x48454f` (`v8`) |

`CClientSocket::SendPacket` @ `0x48455f`. **There is no third `EncodeStr` and
no `if (field1 non-empty)` guard** — the v83/v84/v87 conditional third string
was removed. Empty-code rejection with the literal
`"Please enter the coupon code."` @ `0x4844ee`.

**Row-count self-check:** 2 `EncodeStr` sites → 2 rows. ✅

### Serverbound `operations` — INCOMPLETE (explicit gap)

26 `68 0C 01 00 00` hits fall in the `CCashShop` region
`0x47ece3`-`0x48e809`. One of them (`0x487206`) is **not** a packet site (no
`COutPacket` ctor follows; the bytes at `0x48720b` are
`mov ecx, offset dword_C39830` and a run of colour constants), leaving 25
candidate send sites. Mode bytes read at 19 of them:

`0x47ece3` → 43 · `0x47f38a` → 14 · `0x47f558` → 15 · `0x47f7cc` → 34 ·
`0x47f976` → 52 · `0x4802f0` → 5 · `0x481d16` → 3 · `0x481fbf` → 27 ·
`0x484185` → 4 · `0x4846f6` → 49 · `0x48a034` → 6 · `0x48a29d` → 7 ·
`0x48a4fe` → 8 · `0x48a8db` → 10 · `0x48b07a` → 3 · `0x48bb93` → 33 ·
`0x48d45e` → 30 · `0x48dec8` → 36 · `0x48e809` → 75; plus the
GiftWishItem-equivalent @ `0x48e48c`, which selects among **4 / 36 / 30**
(`mov esi,4` @ `0x48e4a0`, `mov esi,24h` @ `0x48e4b1`, `mov esi,1Eh` @
`0x48e4c9`), and the OnBuySlotInc-equivalent @ `0x48db6a`, which computes
**6 or 7** (`setne cl; add cl,6` @ `0x48db8e`-`0x48db91`).

**Why this is not written as a key → mode table.** Unlike v83/v84/v87/v95, the
v92 IDB has **no names on the request-builder functions** (`func_query`
`CashShop` returns only the `On*Res*` receive handlers plus four senders), so
each mode byte cannot be attributed to a *key* without decompiling every
sender. Three sites (`0x4805a2`, `0x48b7af`, `0x48c56e`, `0x48cead`) push
their mode through a register later in the function and were not resolved.
Emitting a key→mode table from mode bytes alone would be a guess. **This
sub-section is therefore NOT complete and must be finished before Task 5
writes the v92 column of `cash_shop_operation_handle.yaml`.** What remains:
decompile (and name) the 25 v92 sender functions, which is ~25 small
decompiles in session `acdfccff`.

### gms_v92 clientbound template diff

**NOT DERIVED in this pass.** The v92 clientbound `OnCashItemResult` arm table
(`0x495300`) was not enumerated — the pass ran out of budget on the four
serverbound tables. Note for whoever picks this up: an IDA-derived v92 column
**already exists** in
`docs/tasks/task-183-cashshop-result-family/arm-catalog.md` (that table carries
all nine tenant versions, v92 included), while
`docs/packets/dispatchers/cash_shop_operation.yaml` has **no** `gms_v92` key
anywhere in the file (verified by grep). So the correct next step is probably
*not* a fresh decompile but (a) porting the arm-catalog's v92 column into the
YAML and (b) diffing it against `template_gms_92_1.json`'s 57 keys — with a
spot re-derivation from `0x495300` to confirm the catalog.

---

## gms_v95

**Status: SUBSTANTIALLY COMPLETE** — the `errors` enum and the `USE_COUPON`
body are fully derived; three of the 25 serverbound sites are unresolved
(named below).

- Serverbound `CASHSHOP_OPERATION` opcode: **275 / `0x113`**.
- Serverbound `COUPON_CODE` opcode: **276 / `0x114`** (registry correct;
  `COutPacket::COutPacket(&oPacket, 276)` @ `0x487fc2`).
- Clientbound dispatcher: `CCashShop::OnCashItemResult` @ `0x499370`.
- Failure-reason sink: `CCashShop::NoticeFailReason` @ `0x495bc0`.

### `errors` enum — identical to gms_v92

`CCashShop::NoticeFailReason` @ `0x495bc0`: `dec eax` @ `0x495bc4`,
`cmp eax, 44h` @ `0x495bc5`, annotated **"switch 69 cases"** with
**"default case, cases 2,12,13,15,35,36,39-42,44,45,47"** — byte-for-byte the
same bias, count, range and default set as v92's `0x491b50`. Anchor:
`CCashShop::OnStatusCoupon` @ `0x487f10` calls
`CCashShop::NoticeFailReason(this, 33)`, the same disabled-cash-shop reason as
v92. **The v92 table above applies verbatim to gms_v95**, including the five
unnamed new reasons at 63–67 and the two `*_WHEN_UNDER_SEVEN` keys being n-a.

### `USE_COUPON` request body — and what field 1 actually is

Opcode **276 / `0x114`**, built by `CCashShop::OnStatusCoupon` @ `0x487ee0`.
No mode byte. This IDB carries real parameter names, which **retro-names the
mystery first field on every earlier version**:

    if ( CCouponUseSelectDlg::Confirm(&sCouponID, &sCharacterID) == 1 )   /*0x487f5a*/
    ...
    COutPacket::COutPacket(&oPacket, 276);                                /*0x487fc2*/
    ZXString<char>::operator=(v9, &sCharacterID);                         /*0x487fda*/
    COutPacket::EncodeStr(&oPacket, v9[0]);                               /*0x487fe3*/
    ZXString<char>::operator=(v9, &sCouponID);                            /*0x487ff6*/
    COutPacket::EncodeStr(&oPacket, v9[0]);                               /*0x487fff*/

| # | Field | Read | Evidence |
|---|---|---|---|
| 1 | `sCharacterID` (`str`) — the character the coupon's reward is applied to | `EncodeStr` | `0x487fda` → `EncodeStr` @ `0x487fe3` |
| 2 | `sCouponID` (`str`) — the coupon code | `EncodeStr` | `0x487ff6` → `EncodeStr` @ `0x487fff`; trimmed @ `0x487f66`/`0x487f6d`; empty ⇒ `"Please enter the coupon code."` @ `0x487f9e`, packet not sent |

`CClientSocket::SendPacket` @ `0x48800f`; `m_bCashShopRequestSent = 1` @
`0x488018`. Gate `m_bCashShopAuthorized` @ `0x487f09`.

**Row-count self-check:** 2 `EncodeStr` sites → 2 rows. ✅ Same two-field shape
as v92; the v83/v84/v87 conditional third string is absent.

**Retro-application.** The v83/v84/v87 first field — a string the coupon
dialog in those versions never populates — is the **target character id**.
Downstream: for a plain self-redemption Atlas must accept a leading
zero-length string on all five versions, and on v92/v95 must accept exactly
two strings (no optional third).

### Serverbound `operations` (CashShopOperationHandle)

27 `68 13 01 00 00` hits; two (`0x47dc56` inside
`CCashShop::OnChangedCategorySub`, `0x485301` inside `CCashShop::Init`) are
**not** followed by a `COutPacket` ctor on a stack packet (both do
`lea ecx,[esi+…]` into member state) and are treated as non-send sites — 25
candidate send sites, all inside named `CCashShop` methods.

`template_gms_95_1.json` declares **zero** serverbound operations, so every row
below is new to the template.

| Key | Mode | Evidence |
|---|---|---|
| BUY | 3 | `CCashShop::SendBuyAvatarPacket` ctor @ `0x485796`, `push 3` @ `0x4857a6`; `CCashShop::OnBuy` ctor @ `0x48ec48`, `push 3` @ `0x48ec56` |
| GIFT | 4 | `CCashShop::SendGiftsPacket` ctor @ `0x487dd5`, `push 4` @ `0x487df1`; `CCashShop::GiftWishItem` `mov esi,4` @ `0x4924a1` |
| SET_WISHLIST | 5 | `CCashShop::OnSetWish` ctor @ `0x4838d0`, `push 5` @ `0x4838df` |
| INCREASE_INVENTORY | 6 | `CCashShop::OnExItemSlot` ctor @ `0x48dbb8`, `push 6` @ `0x48dbc6`; `CCashShop::OnBuySlotInc` computed arm @ `0x491a47` |
| INCREASE_STORAGE | 7 | `CCashShop::OnIncTrunkCount` ctor @ `0x48de1d`, `push 7` @ `0x48de2b` |
| INCREASE_CHARACTER_SLOT | 8 | `CCashShop::OnIncCharacterSlotCount` ctor @ `0x48e07e`, `push 8` @ `0x48e08c` |
| ENABLE_EQUIP_SLOT | **10** | `CCashShop::OnEnableEquipSlotExt` ctor @ `0x48e45b`, `push 0Ah` @ `0x48e469` — shifted from 9 |
| MOVE_FROM_CASH_INVENTORY | **14** | `CCashShop::OnMoveCashItemLtoS` ctor @ `0x482a8a`, `push 0Eh` @ `0x482a98` — shifted from 13 |
| MOVE_TO_CASH_INVENTORY | **15** | `CCashShop::OnMoveCashItemStoL` ctor @ `0x482c58`, `push 0Fh` @ `0x482c6b` — shifted from 14 |
| REBATE_LOCKER_ITEM | **28** | `CCashShop::OnRebateLockerItem` ctor @ `0x485a9a`, `push 1Ch` @ `0x485aa8` — shifted from 26 |
| BUY_COUPLE | **31** | `CCashShop::OnBuyCouple` ctor @ `0x4910b1`, `push 1Fh` @ `0x4910bf`; `CCashShop::GiftWishItem` `mov esi,1Fh` @ `0x4924c9` |
| BUY_PACKAGE | **32** | `CCashShop::OnBuyPackage` ctor @ `0x48f478`, `push 20h` @ `0x48f486` |
| BUY_NORMAL | **34** | `CCashShop::OnBuyNormal` ctor @ `0x48f753`, `push 22h` @ `0x48f761` |
| APPLY_WISHLIST | **35** | `CCashShop::ApplyWishListEvent` ctor @ `0x482ecc`, `push 23h` @ `0x482eda` |
| BUY_FRIENDSHIP | **37** | `CCashShop::OnBuyFriendship` ctor @ `0x491e50`, `push 25h` @ `0x491e5e`; `CCashShop::GiftWishItem` `mov esi,25h` @ `0x4924b1` |
| GET_PURCHASE_RECORD | **44** | `CCashShop::RequestCashPurchaseRecord` ctor @ `0x4823e3`, `push 2Ch` @ `0x4823f1` |
| BUY_NAME_CHANGE | **50** | `CCashShop::SendBuyNameChangeItemPacket` ctor @ `0x488276`, `push 32h` @ `0x488288` |
| BUY_WORLD_TRANSFER | **53** | `CCashShop::SendBuyTransferWorldItemPacket` ctor @ `0x482f56`, `push 35h` @ `0x482f64` |
| *(no Atlas key)* | **76** | `CCashShop::OnCashGachaponCopy` ctor @ `0x492849`, `push 4Ch` @ `0x492857`. Same arm family as v84's mode 72 / v87's mode 74; here the IDB names the sender, so this one *is* gachapon-copy, not the locker-cap arm — treat v84/72 and v87/74 as still-unnamed. |

**Unresolved v95 sites (3).** `0x483b82` (`CCashShop::OnRemoveWish`),
`0x4901aa` (`CCashShop::OnGiftMateInfoResult`) and `0x490ad8`
(`CCashShop::OnGiftPackage`) push their mode through a register further into
the function and were not read this pass. By structural analogy with v83/v87
they are `SET_WISHLIST`, `GIFT` and `BUY_OTHER_PACKAGE` respectively, but
**that is an inference, not a decompile, and no mode byte is asserted for
them here.** Finish these three before Task 5 writes the v95 column.

**Row-count self-check:** 25 candidate sites, 22 read, 3 unresolved → 19 rows
with a mode byte + 3 named gaps. Not yet equal to the site count; this section
is therefore **not** certified complete.

---

## jms_v185

**Status: body COMPLETE, errors enum PARTIAL (30 of 53 Atlas keys resolved).**
The `COUPON_CODE` body is fully derived and carries a field no GMS version has.
The `NoticeFailReason` enum **fails** the constant-offset set-equality test that
proved v84/v87/v92/v95, so only the sub-domain where equality *does* hold was
mapped; the remaining 23 keys are recorded as unresolved rather than
offset-guessed. See "Why the offset proof stops at 211" below.

- Serverbound `CASHSHOP_OPERATION` opcode: **245 / `0xF5`**
  (`docs/packets/registry/jms_v185.yaml`).
- Serverbound `COUPON_CODE` opcode: **246 / `0xF6`** — **registry value
  CONFIRMED**, see below.
- Request builder: `CCashShop::OnStatusCoupon` @ `0x482450` (already named in
  this IDB; matches the registry's declared `fname`).
- Coupon input dialog: `CCashShop::ShowCouponInputDlg` @ `0x482661`
  (**named this pass**; was `sub_482661`).
- Failure sink: `CCashShop::NoticeFailReason` @ `0x48eaf2`.
- Coupon failure arm: `CCashShop::OnCashItemResUseCouponFailed` @ `0x48d390`.

### Registry confirmation — `COUPON_CODE` opcode 246

Task 1 left `jms_v185.yaml`'s `COUPON_CODE` at 246 with `provenance:
csv-import`, never IDA-confirmed. It is **correct**. The only `COutPacket`
ctor site for 246 inside the `CCashShop` region is
`COutPacket::COutPacket(&oPacket, 0xF6)` @ `0x4825a4`, in
`CCashShop::OnStatusCoupon` @ `0x482450`. Byte search `68 F6 00 00 00` returns
5 hits; the other four are not packet ctors — `0x479519` is inside
`CCashShop::Init` where 246 is the width argument of
`CWnd::CreateWnd(this+4672, 0, 426, 246, 163, …)` @ `0x47952a`, and
`0x565f9b` / `0x90f113` / `0xa7e6fc` are outside the `CCashShop` code region
(`0x478e6f`–`0x48f002`). 246 = `CASHSHOP_OPERATION`(245) + 1, matching every
other version's +1 relation. Registry `provenance` promoted `csv-import` →
`manual` with the address; the numeric value is unchanged.

### `COUPON_CODE` request body — a trailing `Encode1` no GMS version has

No mode byte: after the ctor @ `0x4825a4` the first packet write is an
`EncodeStr` @ `0x4825c1`. There is **no leading `Encode1`**, confirmed at the
instruction level over the whole send path `0x4825a4`–`0x482618`.

The three strings and the byte come from `CCashShop::ShowCouponInputDlg`
@ `0x482661`, which fills three out-parameters from one modal dialog:
`a2` ← `CCtrlEdit::GetText` on edit control #1 @ `0x48273b`, `a3` ←
`CCtrlEdit::GetText` on edit control #2 @ `0x482766`, `a4` ←
`*(v13[1]._m_nRef + 72)` @ `0x482793` (a control-derived integer).
`OnStatusCoupon` calls it as `ShowCouponInputDlg(&v14, &s, &nType)` @
`0x482488`-`0x48248e`, so `v14` = edit #1, `s` = edit #2, `nType` = the control
value.

| # | Field | Read | Evidence |
|---|---|---|---|
| 1 | target character id (`str`) — empty on the plain-redeem path | `COutPacket::EncodeStr` | ctor `0x4825a4`; `ZXString<char>::operator=(&v6, &s)` @ `0x4825b9` → `EncodeStr` @ `0x4825c1`. `s` is `ShowCouponInputDlg`'s second out-param and is the value that gates field 4 — the same structural role field 1 plays on v83/v84/v87, which gms_v95 names `sCharacterID` (`0x487fda`). |
| 2 | coupon code (`str`) | `COutPacket::EncodeStr` | `lea eax,[ebp+var_18]` @ `0x4825c9`, `operator=` @ `0x4825d2` → `EncodeStr` @ `0x4825da`. `var_18` is `v14`, `ShowCouponInputDlg`'s first out-param (edit control #1). |
| 3 | `nType` (`byte`) — **JMS-only field** | `COutPacket::Encode1` | `push [ebp+nType]` @ `0x4825df` → `Encode1` @ `0x4825e5`. Sourced from `*(v13[1]._m_nRef + 72)` @ `0x482793` in `ShowCouponInputDlg` — a value selected in the coupon dialog. **Semantics unknown / unverified**; only its position, width (1 byte) and origin are established. It is unconditionally on the wire. |
| 4 | third string — reachable only when field 1 is non-empty | `COutPacket::EncodeStr` | guard `mov eax,[ebp+s._m_pStr]; cmp eax,ebx; jz` @ `0x4825ea`-`0x4825ef` plus `cmp [eax],bl; jz` @ `0x4825f1`-`0x4825f3` (i.e. `if (s && *s)`); `operator=` @ `0x482601` → `EncodeStr` @ `0x482609`. Encodes `var_14` (`v15`), written by `sub_868B75(&s, &v15)` @ `0x48258b` only inside the same `s`-non-empty branch. |

`CClientSocket::SendPacket` @ `0x482618`; the in-flight flag `[this+1Ch]` is set
to 1 @ `0x482620`. The whole builder is skipped when that flag is already set
(`if (*(this+7)) return;` @ `0x482463`).

**Row-count self-check:** 3 `EncodeStr` call sites (the third guarded) + 1
`Encode1` in the send path → 4 rows. No other `Encode*` call between the ctor
@ `0x4825a4` and `SendPacket` @ `0x482618`. ✅

**Practical wire shape for jms_v185:**
`[opcode 0xF6][str characterId][str couponCode][byte nType]` and, only when
`characterId` is non-empty, a trailing `[str]`.

### `errors` enum

`CCashShop::NoticeFailReason` @ `0x48eaf2` is a jump-table switch:
`add eax, 0FFFFFF4Eh` (bias **−178**) @ `0x48eb03`, `cmp eax, 3Bh` @ `0x48eb08`,
`ja def_48EB13` @ `0x48eb0d`, `jmp jpt_48EB13[eax*4]` @ `0x48eb13`. IDA's own
annotation reads **"switch 60 cases"** with **"default case, cases
179,189,190,213-216,218,219,221,226,234,235"**. Accepted domain 178…237
(`0xB2`…`0xED`) minus those 13 → **47 explicit case labels** (46 distinct
bodies: 217 and 224 share one @ `0x48eec9`), everything else falling to the
default notice (StringPool id 594, `0x48efcd`).

**Anchor A (direct decompile).** `CCashShop::OnCashItemResUseCouponFailed`
@ `0x48d390` reads one `Decode1` @ `0x48d39b` and passes it unmodified to
`NoticeFailReason` @ `0x48d3b4` / `0x48d3be`. Its transfer-field pair is
`if (v3 == 177 || v3 == 179)` @ `0x48d3af` → `SendTransferFieldPacket`
@ `0x48d3c5`. v83's pair is 162, 164. **177 − 162 = 179 − 164 = +15.**

**Anchor B (structural, StringPool run).** v83's four contiguous gift-failure
cases 168–171 carry the contiguous StringPool run 4537, 4538, 4539, 4540
(`0x47c57a`–`0x47c5b9`). jms cases 183–186 — exactly 168–171 + 15 — carry the
contiguous run 4591, 4592, 4593, 4594 (`0x48edab`, `0x48edc1`, `0x48edd7`,
`0x48eded`). A four-long contiguous id run landing on the four +15-corresponding
positions is independent corroboration of the shift.

**Anchor C (the brief's cash-shop-disabled gate) DOES NOT EXIST on jms.**
`CCashShop::OnStatusCoupon` @ `0x482450` contains **no** `NoticeFailReason`
call — its only gate is the in-flight flag `if (*(this+7)) return;` @
`0x482463`. Confirmed by `xrefs_to 0x48eaf2`: 28 call sites, every one an
`On*Failed` / `On*Result` receive handler, none in `OnStatusCoupon`. So the
`X − 195 == offset` check the plan prescribed cannot be run on this version,
and no value is asserted in its place.

#### Why the offset proof stops at 211 (the equality test FAILS globally)

The test the GMS passes used is exact set-equality of the default-case set
under a constant offset. Under +15 it **fails**:

- v83 domain 163…233 (71 cases) → +15 = 178…248. jms domain is 178…**237**
  (60 cases). Same start, different end.
- v83 default set + 15 = `{179,189,190,192,212,213,216-219,221,222,224,
  240-244}` (18 values). jms default set = `{179,189,190,213-216,218,219,221,
  226,234,235}` (13 values). **Not equal.**

Per the plan's stop rule, no table is offset-derived from that. Each of the 47
case bodies was instead read individually for its StringPool id (addresses in
the tables below), and a cross-version id mapping was tested and **rejected**:
the jms id for a +15-corresponding case is not a constant offset of the v83 id
(v83 163→534 vs jms 178→574 is +40; v83 166→536 vs jms 181→577 is +41; v83
178→543 vs jms 193→5919 is unrelated). JMS ships its own string table, so
StringPool ids cannot be used to name jms keys.

What *does* hold is set-equality on a **sub-domain**, and that is what is
tabled. Restricted to v83 `[163,196]` → jms `[178,211]`:

- v83 explicit cases in `[163,196]`: 30 values. +15 → jms `[178,211]`.
- jms explicit cases in `[178,211]`: 31 values.
- The two sets differ by **exactly one member**: jms **192**. 192 = v83 177 + 15,
  and v83 177 (`0xB1`) is the coupon "wrong number → disconnect" reason that
  v83 handles *specially* in `OnCashItemResUseCouponFailed` rather than as a
  notice. jms's `OnCashItemResUseCouponFailed` @ `0x48d390` has **no** such
  special branch (its whole body is the 177/179 pair and one else-arm), so that
  reason became a plain notice case. Coherent divergence, not noise.
- All 30 gap positions inside the sub-domain match. An inserted or deleted
  reason anywhere inside `[178,211]` would break the gap pattern; none does.

Rows in `[178,211]` are therefore marked `aligned (prefix set-equality)` —
the same evidence class as the GMS `aligned` rows: the byte is real and the
client renders *a* notice for it, but the specific Japanese wording is
unconfirmed. Rows above 211 are **not** mapped.

#### Mapped rows — Atlas keys #2–#31

| Key | Byte | StringPool id | Conf. | Evidence (case body addr) |
|---|---|---|---|---|
| REQUEST_TIMED_OUT | 0xB2 (178) | 574 | aligned (prefix set-equality) | `0x48eb1c` |
| NOT_ENOUGH_CASH | 0xB4 (180) | 575 | aligned (prefix set-equality) | `0x48eb32` |
| CANNOT_GIFT_WHEN_UNDERAGE | 0xB5 (181) | 577 | aligned (prefix set-equality) | `0x48eb48` |
| EXCEEDED_GIFT_LIMIT | 0xB6 (182) | 578 | aligned (prefix set-equality) | `0x48eb5e` |
| CANNOT_GIFT_TO_OWN_ACCOUNT | 0xB7 (183) | 4591 | aligned (block-anchored, B) | `0x48edab` |
| INCORRECT_NAME | 0xB8 (184) | 4592 | aligned (block-anchored, B) | `0x48edc1` |
| CANNOT_GIFT_GENDER_RESTRICTION | 0xB9 (185) | 4593 | aligned (block-anchored, B) | `0x48edd7` |
| CANNOT_GIFT_RECIPIENT_INVENTORY_FULL | 0xBA (186) | 4594 | aligned (block-anchored, B) | `0x48eded` |
| EXCEEDED_CASH_ITEM_LIMIT | 0xBB (187) | 579 | aligned (prefix set-equality) | `0x48eb74` |
| INCORRECT_NAME_OR_GENDER_RESTRICTION | 0xBC (188) | 580 | aligned (prefix set-equality) | `0x48eb8a` |
| INVALID_COUPON_CODE | 0xBF (191) | 587 | aligned (prefix set-equality) | `0x48ec35` |
| COUPON_EXPIRED | 0xC1 (193) | 5919 | aligned (prefix set-equality) | `0x48ebc7` |
| COUPON_ALREADY_USED | 0xC2 (194) | 584 | aligned (prefix set-equality) | `0x48ebf3` |
| COUPON_INTERNET_CAFE_RESTRICTION | 0xC3 (195) | 585 | aligned (prefix set-equality) | `0x48ec09` |
| INTERNET_CAFE_COUPON_ALREADY_USED | 0xC4 (196) | 586 | aligned (prefix set-equality) | `0x48ec1f` |
| INTERNET_CAFE_COUPON_EXPIRED | 0xC5 (197) | 3434 | aligned (prefix set-equality) | `0x48ed53` |
| COUPON_NOT_REGISTERED | 0xC6 (198) | 582 | aligned (prefix set-equality) | `0x48ebb1` |
| COUPON_GENDER_RESTRICTION | 0xC7 (199) | 589 | aligned (prefix set-equality) | `0x48ec4b` |
| COUPON_CANNOT_BE_GIFTED | 0xC8 (200) | 590 | aligned (prefix set-equality) | `0x48ec61` |
| COUPON_ONLY_FOR_MAPLE_STORY | 0xC9 (201) | 894 | aligned (prefix set-equality) | `0x48ec77` |
| INVENTORY_FULL | 0xCA (202) | 591 | aligned (prefix set-equality) | `0x48ec8d` |
| NOT_AVAILABLE_FOR_PURCHASE | 0xCB (203) | 592 | aligned (prefix set-equality) | `0x48eca3` |
| CANNOT_GIFT_INVALID_NAME_OR_GENDER | 0xCC (204) | 614 | aligned (prefix set-equality) | `0x48ecb9` |
| CHECK_NAME_OF_RECEIVER | 0xCD (205) | 2708 | aligned (prefix set-equality) | `0x48eccf` |
| NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR | 0xCE (206) | 2709 | aligned (prefix set-equality) | `0x48ece5` |
| OUT_OF_STOCK | 0xCF (207) | 593 | aligned (prefix set-equality) | `0x48ecfb` |
| EXCEEDED_SPENDING_LIMIT | 0xD0 (208) | 257 | aligned (prefix set-equality) | `0x48ed11` |
| NOT_ENOUGH_MESOS | 0xD1 (209) | 5043 | aligned (prefix set-equality) | `0x48ed27` |
| CASH_SHOP_NOT_AVAILABLE_DURING_BETA | 0xD2 (210) | 5522 | aligned (prefix set-equality) | `0x48ed3d` |
| INVALID_BIRTHDAY | 0xD3 (211) | 3435 | aligned (prefix set-equality) | `0x48ed69` |
| UNKNOWN_ERROR | *(any byte outside 178–237, or a default-set byte)* | 594 | verified | default case `0x48efcd` |

#### Unresolved Atlas keys (23) — recorded absent, NOT guessed

Every key from `ONLY_AVAILABLE_TO_USERS_BUYING` onward falls above jms 211,
where the case set diverges from v83+15 (v83+15 predicts 214, 215, 220, 223,
225–239, 245–248; jms actually has 212, 217, 220, 222–225, 227–233, 236, 237).
No byte is asserted for any of them:

`ONLY_AVAILABLE_TO_USERS_BUYING`, `ALREADY_APPLIED`, `DAILY_PURCHASE_LIMIT`,
`COUPON_USAGE_LIMIT`, `COUPON_SYSTEM_AVAILABLE_SOON`, `FIFTEEN_DAY_LIMIT`,
`NOT_ENOUGH_GIFT_TOKENS`, `CANNOT_SEND_TECHNICAL_DIFFICULTIES`,
`CANNOT_GIFT_ACCOUNT_AGE`, `CANNOT_GIFT_PREVIOUS_INFRACTIONS`,
`CANNOT_GIFT_AT_THIS_TIME`, `CANNOT_GIFT_LIMIT`,
`CANNOT_GIFT_TECHNICAL_DIFFICULTIES`, `CANNOT_TRANSFER_UNDER_LEVEL_TWENTY`,
`CANNOT_TRANSFER_TO_SAME_WORLD`, `CANNOT_TRANSFER_TO_NEW_WORLD`,
`CANNOT_TRANSFER_OUT`, `CANNOT_TRANSFER_NO_EMPTY_SLOTS`,
`EVENT_ENDED_OR_CANT_BE_FREELY_TESTED`,
`CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS`, `PLEASE_TRY_AGAIN`,
`CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN`, `CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN`.

Reason: no evidence route survives. Offset derivation is barred by the failed
set-equality; StringPool cross-mapping is barred by the rejected id relation
above; ordinal alignment is barred because the tail has 16 cases against 23
keys. Resolving them requires reading the Japanese text behind each id from a
JMS String.wz / StringPool dump, which is not present in this environment (the
same dead end documented for gms_v83's `aligned` rows).

The 16 unassigned tail cases are recorded here as raw evidence so a future pass
with a JMS string table can finish the mapping without re-decompiling:

| Byte | StringPool id | Evidence (case body addr) |
|---|---|---|
| 0xD4 (212) | 3722 | `0x48ed7f` |
| 0xD9 (217) | 581 | `0x48eec9` (shared body with 224) |
| 0xDC (220) | 4511 | `0x48ed95` |
| 0xDE (222) | 4813 | `0x48ee9d` |
| 0xDF (223) | 4814 | `0x48eeb3` |
| 0xE0 (224) | 581 | `0x48eec9` (shared body with 217) |
| 0xE1 (225) | 4986 | `0x48eedf` |
| 0xE3 (227) | 5473 | `0x48ee2f` |
| 0xE4 (228) | 5452 | `0x48ee45` |
| 0xE5 (229) | 5455 | `0x48ee5b` |
| 0xE6 (230) | 5456 | `0x48ee71` |
| 0xE7 (231) | 5457 | `0x48ee87` |
| 0xE8 (232) | 5786 (`0x169A`), `ZXString<char>::Format` with literal 30 | `0x48eef5`; `GetString` @ `0x48ef0e`, `Format` @ `0x48ef2c` |
| 0xE9 (233) | 5786 (`0x169A`), `ZXString<char>::Format` with literal 70 | `0x48ef5c`; `GetString` @ `0x48ef75`, `Format` @ `0x48ef97` |
| 0xEC (236) | 5664 | `0x48ee03` |
| 0xED (237) | 5890 | `0x48ee19` |

One further case inside the mapped sub-domain has no Atlas key:

| Byte | StringPool id | Evidence | Note |
|---|---|---|---|
| 0xC0 (192) | 583 | `0x48ebdd` | = v83 `0xB1` + 15. A default case on v83/v84/v87 (handled specially in `OnCashItemResUseCouponFailed`); a plain notice on jms. Key unknown / unverified. |

**Row-count self-check:** domain 178–237 = 60 values; 13 default-set values ⇒
**47 explicit case labels**, matching IDA's "switch 60 cases" header. Split:
sub-domain `[178,211]` = 34 values − 3 defaults (179, 189, 190) = 31 explicit
= **30 Atlas-keyed rows + 1 unkeyed (192)**; tail `[212,237]` = 26 values − 10
defaults (213–216, 218, 219, 221, 226, 234, 235) = **16 explicit, all
unkeyed**. 31 + 16 = 47 ✅. Atlas key coverage: **30 of 53** mapped,
**23 recorded absent**, plus `UNKNOWN_ERROR` on the default case.

**Reserved reason bytes that are NOT notices** (must not be sent as a generic
coupon error — they change client state):

| Byte | Behaviour | Evidence |
|---|---|---|
| 0xB1 (177) | `NoticeFailReason` (default text — 177 is in the default-case set) **then** `CCashShop::SendTransferFieldPacket` — kicks the player out of the cash shop | `OnCashItemResUseCouponFailed` @ `0x48d3af`, `0x48d3be`, `0x48d3c5` |
| 0xB3 (179) | same as 0xB1 | `0x48d3af` |

jms has **no** coupon-specific "wrong number → `OnStatusExit`" byte; the v83
`0xB1` / v84 `186` / v92 `15` special branch is absent from
`OnCashItemResUseCouponFailed` @ `0x48d390` (its entire body is the 177/179
pair and one else-arm).

---

## Cross-version summary of the two values the codec needs

| version | `COUPON_CODE` opcode | body |
|---|---|---|
| gms_v83 | 230 / `0xE6` | `str characterId` · `str couponCode` · optional `str` (dead: field 1 always empty) |
| gms_v84 | **236 / `0xEC`** (registry says 230 — bug) | same as v83 |
| gms_v87 | 243 / `0xF3` | same as v83 |
| gms_v92 | 269 / `0x10D` | `str characterId` · `str couponCode` (no optional third) |
| gms_v95 | 276 / `0x114` | `str characterId` · `str couponCode` (no optional third) |
| jms_v185 | 246 / `0xF6` (registry confirmed) | `str characterId` · `str couponCode` · **`byte nType`** · optional `str` (guarded by `characterId` non-empty) |

| version | reason-byte scale | domain | explicit reasons |
|---|---|---|---|
| gms_v83 | v83 baseline | `0xA3`–`0xE9` | 53 + default |
| gms_v84 | v83 **+ 9** | `0xAC`–`0xF2` | 53 + default |
| gms_v87 | v83 **+ 15** | `0xB2`–`0xF9` | 54 (one new) + default |
| gms_v92 | v83 **− 162** (1-based) | 1–69 | 56 (five new, two dropped) + default |
| gms_v95 | identical to gms_v92 | 1–69 | 56 + default |
| jms_v185 | v83 **+ 15** on `[163,196]` **only** — global set-equality FAILS above that | 178–237 (`0xB2`–`0xED`) | 47 explicit + default; **30 of 53 Atlas keys mapped, 23 absent** |
