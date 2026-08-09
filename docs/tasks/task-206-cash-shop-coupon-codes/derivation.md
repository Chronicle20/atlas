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

**Reserved reason bytes that are NOT notices** (added task-206 Task 9 — this
section was the only GMS version without the subsection its siblings carry;
nothing mis-sends today because none of the three is mapped, but the record
had no in-place warning against a future hand-edit). The implied trio from
v83's 162/164/177 plus the proven +15 is 177/179/192, and decompiling
`CCashShop::OnCashItemResUseCouponFailed` @ `0x485f93` **confirms all three
exactly**:

| Byte | Behaviour | Evidence |
|---|---|---|
| 177 (0xB1) | `NoticeFailReason` (default text — 177 is in the default-case set) **then** `CCashShop::SendTransferFieldPacket` — kicks the player out of the cash shop | `if ( v3 == 177 \|\| v3 == 179 )` @ `0x485fb9`; `NoticeFailReason` @ `0x485fc8`, `SendTransferFieldPacket` @ `0x485fcf` |
| 179 (0xB3) | same as 177 | `0x485fb9` |
| 192 (0xC0) | notice StringPool **558** **then** `CCashShop::OnStatusExit` — the coupon "wrong number → disconnect" byte | `if ( v4 == 192 )` @ `0x485fda`; `GetString(&v6, 558)` @ `0x485ff3`, `CUtilDlg::Notice` @ `0x485ff8`, `OnStatusExit` @ `0x486002` |

192 = v83's 177 + 15 ✅ and both 177 and 179 are in v87's default-case set, so
none of the three collides with a mapped key. None is mapped in
`cash_shop_operation.yaml`.

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
statement at the end of this section. The clientbound 57-arm table **was**
derived in full by task-206 Task 9 — see
"### gms_v92 clientbound arm table" below.

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

Opcode **269 / `0x10D`**, built by `CCashShop::OnStatusCoupon` @ `0x484430`
(**named in task-4's export pass**; was `sub_484430` — an unnamed symbol blocked
`packet-audit export`'s by-name harvest, so the evidence citation could not be
pinned until it was named). No mode byte.

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

### gms_v92 clientbound arm table

**DERIVED IN FULL this pass (task-206 Task 9) — the earlier note above is
superseded and its premise was wrong.** `docs/tasks/task-183-cashshop-result-
family/arm-catalog.md` does **not** carry a v92 column: its nine per-arm columns
are v48 / v61 / v72 / v79 / v83 / v84 / v87 / v95 / jms — v92 is not among them
(`grep -c v92 arm-catalog.md` → 0). There was nothing to port, so the whole arm
table was enumerated directly rather than spot-checked.

**Why this mattered.** `cash_shop_operation.yaml` had no `gms_v92` key anywhere,
and `packet-audit operations` short-circuits on `len(expected) == 0`, so
`template_gms_92_1.json`'s 57 `CashShopOperation` `operations` keys were
validated by **nothing** while `--check` still reported OK.

**Jump-table header.** `CCashShop::OnCashItemResult` @ `0x495300`
(session `acdfccff`): `CInPacket::Decode1` @ `0x49530a`,
`add eax, 0FFFFFFADh` (bias −83) @ `0x495312`, `cmp eax, 66h` @ `0x495315`,
`ja def_495325` @ `0x495318`, annotated **"switch 103 cases"** with
**"default case, cases 84-86,93-96,102,105,125-146,157,158,161-166,168-171,
174,175,177"**. Domain **83…185** = 103 values; the default set has 46 members
(3+4+1+1+22+2+6+4+2+1); 103 − 46 = **57 explicit cases**, matching the 57
distinct jump targets IDA lists at `jmp ds:jpt_495325` and the first arm's own
annotation `jumptable 00495325 case 83` @ `0x49532c`.

**This is not an ordinal alignment.** All 57 handlers carry their real mangled
symbols in this IDB, so each mode byte is attached to its Atlas key through the
`handler:` fname already recorded in `cash_shop_operation.yaml` — every row is
`verified`, none is `aligned`. The five arms the brief asked to be re-derived
by hand (including both coupon arms) are marked ★.

| Atlas key | mode | handler fname | dispatch call addr |
|---|---|---|---|
| ★ LIMIT_GOODS_COUNT_CHANGED | 83 | `CCashShop::OnCashItemResLimitGoodsCountChanged` | `0x49532f` |
| LOAD_INVENTORY_SUCCESS | 87 | `CCashShop::OnCashItemResLoadLockerDone` | `0x49533c` |
| LOAD_INVENTORY_FAILURE | 88 | `CCashShop::OnCashItemResLoadLockerFailed` | `0x495349` |
| LOAD_GIFT_SUCCESS | 89 | `CCashShop::OnCashItemResLoadGiftDone` | `0x495356` |
| LOAD_GIFT_FAILED | 90 | `CCashShop::OnCashItemResLoadGiftFailed` | `0x495363` |
| LOAD_WISHLIST | 91 | `CCashShop::OnCashItemResLoadWishDone` | `0x495370` |
| LOAD_WISH_FAILED | 92 | `CCashShop::OnCashItemResLoadWishFailed` | `0x49537d` |
| UPDATE_WISHLIST | 97 | `CCashShop::OnCashItemResSetWishDone` | `0x49538a` |
| SET_WISH_FAILED | 98 | `CCashShop::OnCashItemResSetWishFailed` | `0x495397` |
| PURCHASE_SUCCESS | 99 | `CCashShop::OnCashItemResBuyDone` | `0x4953a4` |
| BUY_FAILED | 100 | `CCashShop::OnCashItemResBuyFailed` | `0x4953b1` |
| ★ USE_COUPON_SUCCESS | 101 | `CCashShop::OnCashItemResUseCouponDone` | `0x4953f2` |
| GIFT_COUPON_SUCCESS | 103 | `CCashShop::OnCashItemResGiftCouponDone` | `0x4953ff` |
| ★ USE_COUPON_FAILED | 104 | `CCashShop::OnCashItemResUseCouponFailed` | `0x49540c` |
| GIFT_SUCCESS | 106 | `CCashShop::OnCashItemResGiftDone` | `0x495419` |
| GIFT_FAILED | 107 | `CCashShop::OnCashItemResGiftFailed` | `0x495426` |
| INVENTORY_CAPACITY_INCREASE_SUCCESS | 108 | `CCashShop::OnCashItemResIncSlotCountDone` | `0x495433` |
| INVENTORY_CAPACITY_INCREASE_FAILED | 109 | `CCashShop::OnCashItemResIncSlotCountFailed` | `0x495440` |
| INC_TRUNK_COUNT_SUCCESS | 110 | `CCashShop::OnCashItemResIncTrunkCountDone` | `0x49544d` |
| INC_TRUNK_COUNT_FAILED | 111 | `CCashShop::OnCashItemResIncTrunkCountFailed` | `0x49545a` |
| INC_CHARACTER_SLOT_COUNT_SUCCESS | 112 | `CCashShop::OnCashItemResIncCharacterSlotCountDone` | `0x495467` |
| INC_CHARACTER_SLOT_COUNT_FAILED | 113 | `CCashShop::OnCashItemResIncCharacterSlotCountFailed` | `0x495474` |
| INC_BUY_CHARACTER_COUNT_SUCCESS | 114 | `CCashShop::OnCashItemResIncBuyCharacterCountDone` | `0x495481` |
| INC_BUY_CHARACTER_COUNT_FAILED | 115 | `CCashShop::OnCashItemResIncBuyCharacterCountFailed` | `0x49548e` |
| ENABLE_EQUIP_SLOT_EXT_SUCCESS | 116 | `CCashShop::OnCashItemResEnableEquipSlotExtDone` | `0x49549b` |
| ENABLE_EQUIP_SLOT_EXT_FAILED | 117 | `CCashShop::OnCashItemResEnableEquipSlotExtFailed` | `0x4954a8` |
| CASH_ITEM_MOVED_TO_INVENTORY | 118 | `CCashShop::OnCashItemResMoveLtoSDone` | `0x4954b5` |
| MOVE_L_TO_S_FAILED | 119 | `CCashShop::OnCashItemResMoveLtoSFailed` | `0x4954c2` |
| CASH_ITEM_MOVED_TO_CASH_INVENTORY | 120 | `CCashShop::OnCashItemResMoveStoLDone` | `0x4954cf` |
| MOVE_S_TO_L_FAILED | 121 | `CCashShop::OnCashItemResMoveStoLFailed` | `0x4954dc` |
| DESTROY_SUCCESS | 122 | `CCashShop::OnCashItemResDestroyDone` | `0x4954e9` |
| DESTROY_FAILED | 123 | `CCashShop::OnCashItemResDestroyFailed` | `0x4954f6` |
| EXPIRE_DONE | 124 | `CCashShop::OnCashItemResExpireDone` | `0x49555e` |
| REBATE_SUCCESS | 147 | `CCashShop::OnCashItemResRebateDone` | `0x495503` |
| REBATE_FAILED | 148 | `CCashShop::OnCashItemResRebateFailed` | `0x495510` |
| COUPLE_SUCCESS | 149 | `CCashShop::OnCashItemResCoupleDone` | `0x49551d` |
| COUPLE_FAILED | 150 | `CCashShop::OnCashItemResCoupleFailed` | `0x49552a` |
| BUY_PACKAGE_SUCCESS | 151 | `CCashShop::OnCashItemResBuyPackageDone` | `0x4953be` |
| BUY_PACKAGE_FAILED | 152 | `CCashShop::OnCashItemResBuyPackageFailed` | `0x4953cb` |
| GIFT_PACKAGE_SUCCESS | 153 | `CCashShop::OnCashItemResGiftPackageDone` | `0x4953d8` |
| GIFT_PACKAGE_FAILED | 154 | `CCashShop::OnCashItemResGiftPackageFailed` | `0x4953e5` |
| ★ BUY_NORMAL_SUCCESS | 155 | `CCashShop::OnCashItemResBuyNormalDone` | `0x49556b` |
| BUY_NORMAL_FAILED | 156 | `CCashShop::OnCashItemResBuyNormalFailed` | `0x495578` |
| FRIENDSHIP_SUCCESS | 159 | `CCashShop::OnCashItemResFriendShipDone` | `0x495537` |
| FRIENDSHIP_FAILED | 160 | `CCashShop::OnCashItemResFriendShipFailed` | `0x495544` |
| FREE_CASH_ITEM_DONE | 167 | `CCashShop::OnCashItemResFreeCashItemDone` | `0x495551` |
| PURCHASE_RECORD | 172 | `CCashShop::OnCashItemResPurchaseRecord` | `0x4955ac` |
| PURCHASE_RECORD_FAILED | 173 | `CCashShop::OnCashItemResPurchaseRecordFailed` | `0x4955b9` |
| NAME_CHANGE_BUY_DONE | 176 | `CCashShop::OnCashItemNameChangeResBuyDone` | `0x495585` |
| TRANSFER_WORLD_SUCCESS | 178 | `CCashShop::OnCashItemResTransferWorldDone` | `0x495592` |
| TRANSFER_WORLD_FAILED | 179 | `CCashShop::OnCashItemResTransferWorldFailed` | `0x49559f` |
| GACHAPON_OPEN_SUCCESS | 180 | `CCashShop::OnCashItemResCashGachaponOpenDone` | `0x4955c6` |
| GACHAPON_OPEN_FAILED | 181 | `CCashShop::OnCashItemResCashGachaponOpenFailed` | `0x4955d3` |
| GACHAPON_COPY_SUCCESS | 182 | `CCashShop::OnCashItemResCashGachaponCopyDone` | `0x4955e0` |
| GACHAPON_COPY_FAILED | 183 | `CCashShop::OnCashItemResCashGachaponCopyFailed` | `0x4955ed` |
| ★ CHANGE_MAPLE_POINT_SUCCESS | 184 | `CCashShop::OnCashItemResChangeMaplePointDone` | `0x4955fa` |
| CHANGE_MAPLE_POINT_FAILED | 185 | `CCashShop::OnCashItemResChangeMaplePointFailed` | `0x495607` |

★ spot re-derivations (done first, before the full sweep, as the catalog-
agreement gate the brief specified): `USE_COUPON_SUCCESS` = **101** @ `0x4953f2`,
`USE_COUPON_FAILED` = **104** @ `0x49540c`, `LIMIT_GOODS_COUNT_CHANGED` = **83**
@ `0x49532f` (independently confirmed by the disassembler comment
`jumptable 00495325 case 83`), `BUY_NORMAL_SUCCESS` = **155** @ `0x49556b`,
`CHANGE_MAPLE_POINT_SUCCESS` = **184** @ `0x4955fa`. All five equal the shipped
template values (101 and 104 for the two coupon arms, as the brief predicted).

**Diff against `template_gms_92_1.json` (57 existing keys): ZERO differences.**
Same key set (no template-only key, no derived-only key) and the same value for
all 57. So Step 4's "treat disagreement as a template bug" branch did not fire:
**no v92 wire byte changed**, and `packet-audit operations` regenerates the file
byte-identically. The only change is that the column now EXISTS in the YAML, so
`--check` actually validates those 57 keys instead of skipping them.

Ten of the YAML's 67 `operations` keys have no v92 arm (absent by full case
enumeration, so `n-a`, recorded as key-omission per this file's convention).
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

## gms_v48

IDB `GMS_v48_1_DEVM.exe.i64`, session `93cc947e`. Failure-reason sink:
`CCashShop::NoticeFailReason` @ `0x456073` (already named in this IDB).

### `errors` enum

**Jump-table header (read, not inferred).** `add eax, 0FFFFFF8Fh` (bias −113) @ `0x456079`,
`cmp eax, 2Ah` @ `0x45607c`, IDA annotation **"switch 43 cases"** with
**"default case, cases 114,139,140"**. Domain **113…155** (43 values) minus the
3 default-set values ⇒ **40 explicit cases**, which equals the 40 distinct
jump-table targets IDA lists at the `jmp ds:jpt_…` line.

**The constant-offset hypothesis FAILS — nothing here is offset-derived.**
The test the v84/v87/v92/v95 passes used is exact set-equality of the
default-case set under a constant offset. v83's domain is 163…233 (71 cases,
18-member default set); this version's is 113…155 (43 cases, 3-member default
set). Neither the case count, the range, nor the default set maps under any
offset — in particular the offset the `OnStatusCoupon` gate would imply
(137 − 195 = -58) predicts a domain starting at 105, not 113. **Rejected.**
Per the stop rule, every case body was instead decompiled individually and
its `StringPool` id read (addresses in the table below).

**How the keys were assigned.** The 40 explicit cases were aligned ordinally
against v83's 53-case list. Four independent constraints pin that alignment:

- **Anchor A (exact, direct).** Exactly one case in this version does not use
  the StringPool — it builds the inline literal `"The coupon system will be
  available soon."` (`aTheCouponSyste`). It is case **143** @ `0x4562de`, at
  ordinal position 28 of this version's case list. v83's inline-literal case
  (`0xD2`) is at ordinal position 35 of its 53. ⇒ `COUPON_SYSTEM_AVAILABLE_SOON`.
- **Anchor B (exact, call site).** `CCashShop::OnStatusCoupon` @ `0x44d304`
  calls `NoticeFailReason(this, 137)` when the cash-shop-loaded flag
  `[this+0x40]` is clear (address from §"Legacy versions — COUPON_CODE
  applicability"). Case 137 sits at ordinal position 24 here; v83's gate (195)
  sits at ordinal 29 of its 53. ⇒ `CASH_SHOP_NOT_AVAILABLE_DURING_BETA`.
- **Anchor C (structural, id-block deltas).** StringPool ids are not stable
  across GMS versions (this version's default-case id is 534; v83's is 557), so
  id *equality* proves nothing. What is stable is the *internal delta pattern*
  of each contiguous run. Every multi-member run in this version's id list has
  a byte-identical delta pattern to the correspondingly-positioned run in v83's
  (see the `v83 id` column below) — a run inserted or deleted anywhere would
  break it.
- **Anchor D (paired singleton).** v83 uses two consecutive ids at two far-apart
  ordinals — 3274 at ordinal 17 (`COUPON_NOT_REGISTERED`) and 3275 at ordinal 31
  (`ONLY_AVAILABLE_TO_USERS_BUYING`). This version reproduces that signature
  exactly with 2938 / 2939 at the correspondingly-mapped ordinals.

Anchors A and B independently fix how many keys are missing *before* each of
them, and the id-delta alignment fixes *which*. The two counts agree, which
is the consistency check — see the row-count self-check below.

Rows are `aligned` (ordinal inference, same evidence class as the v83 `aligned`
rows — the byte is real and the client renders *a* notice for it, but the exact
English wording is unconfirmed) or `verified` where an anchor names the row
directly. No `aligned` row is restated as `verified` anywhere.

| Key | Byte | StringPool id | v83 id | Conf. | Evidence (case body addr) |
|---|---|---|---|---|---|
| REQUEST_TIMED_OUT | 0x71 (113) | 514 | 534 | aligned | `0x45608e` |
| NOT_ENOUGH_CASH | 0x73 (115) | 515 | 535 | aligned | `0x4560a4` |
| CANNOT_GIFT_WHEN_UNDERAGE | 0x74 (116) | 516 | 536 | aligned | `0x4560ba` |
| EXCEEDED_GIFT_LIMIT | 0x75 (117) | 517 | 537 | aligned | `0x4560d0` |
| EXCEEDED_CASH_ITEM_LIMIT | 0x76 (118) | 518 | 538 | aligned | `0x4560e6` |
| INCORRECT_NAME_OR_GENDER_RESTRICTION | 0x77 (119) | 519 | 539 | aligned | `0x4560fc` |
| INVALID_COUPON_CODE | 0x78 (120) | 526 | 547 | aligned | `0x456196` |
| COUPON_EXPIRED | 0x79 (121) | 522 | 543 | aligned | `0x45613e` |
| COUPON_ALREADY_USED | 0x7A (122) | 521 | 542 | aligned | `0x456128` |
| COUPON_INTERNET_CAFE_RESTRICTION | 0x7B (123) | 523 | 544 | aligned | `0x456154` |
| INTERNET_CAFE_COUPON_ALREADY_USED | 0x7C (124) | 524 | 545 | aligned | `0x45616a` |
| INTERNET_CAFE_COUPON_EXPIRED | 0x7D (125) | 525 | 546 | aligned | `0x456180` |
| COUPON_NOT_REGISTERED | 0x7E (126) | 2938 | 3274 | aligned | `0x45629e` |
| COUPON_GENDER_RESTRICTION | 0x7F (127) | 520 | 541 | aligned | `0x456112` |
| COUPON_ONLY_FOR_MAPLE_STORY | 0x80 (128) | 529 | 552 | aligned | `0x4561ac` |
| INVENTORY_FULL | 0x81 (129) | 775 | 853 | aligned | `0x4561c2` |
| NOT_AVAILABLE_FOR_PURCHASE | 0x82 (130) | 530 | 553 | aligned | `0x4561d8` |
| CANNOT_GIFT_INVALID_NAME_OR_GENDER | 0x83 (131) | 531 | 554 | aligned | `0x4561ee` |
| CHECK_NAME_OF_RECEIVER | 0x84 (132) | 554 | 583 | aligned | `0x456204` |
| NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR | 0x85 (133) | 2264 | 2505 | aligned | `0x45621a` |
| OUT_OF_STOCK | 0x86 (134) | 2265 | 2506 | aligned | `0x456230` |
| EXCEEDED_SPENDING_LIMIT | 0x87 (135) | 532 | 555 | aligned | `0x456246` |
| NOT_ENOUGH_MESOS | 0x88 (136) | 263 | 5599 | aligned | `0x45625c` |
| CASH_SHOP_NOT_AVAILABLE_DURING_BETA | 0x89 (137) | 533 | 556 | **verified (anchor B)** | `0x456272` |
| INVALID_BIRTHDAY | 0x8A (138) | 495 | 514 | aligned | `0x456288` |
| ONLY_AVAILABLE_TO_USERS_BUYING | 0x8D (141) | 2939 | 3275 | aligned | `0x4562b4` |
| ALREADY_APPLIED | 0x8E (142) | 3210 | 3558 | aligned | `0x4562ca` |
| COUPON_SYSTEM_AVAILABLE_SOON | 0x8F (143) | inline literal | inline | **verified (anchor A)** | `0x4562de` |
| FIFTEEN_DAY_LIMIT | 0x90 (144) | 3449 | 3749 | aligned | `0x4562fd` |
| NOT_ENOUGH_GIFT_TOKENS | 0x91 (145) | 3643 | 3992 | aligned | `0x456313` |
| CANNOT_SEND_TECHNICAL_DIFFICULTIES | 0x92 (146) | 3644 | 3993 | aligned | `0x456329` |
| CANNOT_GIFT_ACCOUNT_AGE | 0x93 (147) | 3645 | 3994 | aligned | `0x45633f` |
| CANNOT_GIFT_PREVIOUS_INFRACTIONS | 0x94 (148) | 3646 | 3995 | aligned | `0x456355` |
| CANNOT_GIFT_AT_THIS_TIME | 0x95 (149) | 3647 | 3996 | aligned | `0x45636b` |
| CANNOT_GIFT_LIMIT | 0x96 (150) | 3648 | 3997 | aligned | `0x456381` |
| CANNOT_GIFT_TECHNICAL_DIFFICULTIES | 0x97 (151) | 3649 | 3998 | aligned | `0x456394` |
| CANNOT_TRANSFER_UNDER_LEVEL_TWENTY | 0x98 (152) | 3655 | 4002 | aligned | `0x4563a7` |
| CANNOT_TRANSFER_TO_SAME_WORLD | 0x99 (153) | 3658 | 4005 | aligned | `0x4563ba` |
| CANNOT_TRANSFER_TO_NEW_WORLD | 0x9A (154) | 3659 | 4006 | aligned | `0x4563cd` |
| CANNOT_TRANSFER_OUT | 0x9B (155) | 3660 | 4007 | aligned | `0x4563e0` |
| UNKNOWN_ERROR | *(any unlisted byte)* | 534 | 557 | verified | default case `0x4563f3` |

#### Atlas keys ABSENT from this version (13) — no case exists

These are absent-by-enumeration, not unknown: the full case list above is
complete for the switch's whole domain, so a key with no row has no byte.

- `CANNOT_GIFT_TO_OWN_ACCOUNT`
- `INCORRECT_NAME`
- `CANNOT_GIFT_GENDER_RESTRICTION`
- `CANNOT_GIFT_RECIPIENT_INVENTORY_FULL`
- `COUPON_CANNOT_BE_GIFTED`
- `DAILY_PURCHASE_LIMIT`
- `COUPON_USAGE_LIMIT`
- `CANNOT_TRANSFER_NO_EMPTY_SLOTS`
- `EVENT_ENDED_OR_CANT_BE_FREELY_TESTED`
- `CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS`
- `PLEASE_TRY_AGAIN`
- `CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN`
- `CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN`

**Row-count self-check:** 40 mapped rows + 13 absent = 53 Atlas keys;
40 explicit cases = 43 domain values − 3 default-set values. ✅

**Reserved reason bytes that are NOT notices** (must never be mapped to an
Atlas key — they change client state):

| Byte | Behaviour | Evidence |
|---|---|---|
| 112 (0x70) | `NoticeFailReason` (default text) **then** `CCashShop::SendTransferFieldPacket` — kicks the player out of the cash shop | `CCashShop::OnCashItemResUseCouponFailed` @ `0x454cb9`, branch `0x454cd4` |
| 114 (0x72) | same as 112 | `0x454cd4` |

Neither byte is an explicit case (112 is below the switch domain / in the
default set, 114 is in the default set), so neither can collide with a mapped
key — machine-checked. **This version has NO coupon-specific
"wrong number → `OnStatusExit`" byte**: unlike v83 (`0xB1`) / v84 (186) /
v87 (192) / v92 (15), the `else` arm of `OnCashItemResUseCouponFailed` is a
plain `NoticeFailReason` call with no follow-up — same shape as jms_v185.

---

## gms_v61

IDB `GMS_v61.1_U_DEVM.exe.i64`, session `415bf585`. Failure-reason sink:
`CCashShop::NoticeFailReason` @ `0x463ef4` (already named in this IDB).

### `errors` enum

**Jump-table header (read, not inferred).** `add eax, 0FFFFFF80h` (bias −128) @ `0x463efa`,
`cmp eax, 33h` @ `0x463efd`, IDA annotation **"switch 52 cases"** with
**"default case, cases 129,155,156,159-162,164,165"**. Domain **128…179** (52 values) minus the
9 default-set values ⇒ **43 explicit cases**, which equals the 43 distinct
jump-table targets IDA lists at the `jmp ds:jpt_…` line.

**The constant-offset hypothesis FAILS — nothing here is offset-derived.**
The test the v84/v87/v92/v95 passes used is exact set-equality of the
default-case set under a constant offset. v83's domain is 163…233 (71 cases,
18-member default set); this version's is 128…179 (52 cases, 9-member default
set). Neither the case count, the range, nor the default set maps under any
offset — in particular the offset the `OnStatusCoupon` gate would imply
(153 − 195 = -42) predicts a domain starting at 121, not 128. **Rejected.**
Per the stop rule, every case body was instead decompiled individually and
its `StringPool` id read (addresses in the table below).

**How the keys were assigned.** The 43 explicit cases were aligned ordinally
against v83's 53-case list. Four independent constraints pin that alignment:

- **Anchor A (exact, direct).** Exactly one case in this version does not use
  the StringPool — it builds the inline literal `"The coupon system will be
  available soon."` (`aTheCouponSyste`). It is case **167** @ `0x4641a3`, at
  ordinal position 31 of this version's case list. v83's inline-literal case
  (`0xD2`) is at ordinal position 35 of its 53. ⇒ `COUPON_SYSTEM_AVAILABLE_SOON`.
- **Anchor B (exact, call site).** `CCashShop::OnStatusCoupon` @ `0x45a6d2`
  calls `NoticeFailReason(this, 153)` when the cash-shop-loaded flag
  `[this+0x40]` is clear (address from §"Legacy versions — COUPON_CODE
  applicability"). Case 153 sits at ordinal position 25 here; v83's gate (195)
  sits at ordinal 29 of its 53. ⇒ `CASH_SHOP_NOT_AVAILABLE_DURING_BETA`.
- **Anchor C (structural, id-block deltas).** StringPool ids are not stable
  across GMS versions (this version's default-case id is 569; v83's is 557), so
  id *equality* proves nothing. What is stable is the *internal delta pattern*
  of each contiguous run. Every multi-member run in this version's id list has
  a byte-identical delta pattern to the correspondingly-positioned run in v83's
  (see the `v83 id` column below) — a run inserted or deleted anywhere would
  break it.
- **Anchor D (paired singleton).** v83 uses two consecutive ids at two far-apart
  ordinals — 3274 at ordinal 17 (`COUPON_NOT_REGISTERED`) and 3275 at ordinal 31
  (`ONLY_AVAILABLE_TO_USERS_BUYING`). This version reproduces that signature
  exactly with 3194 / 3195 at the correspondingly-mapped ordinals.

Anchors A and B independently fix how many keys are missing *before* each of
them, and the id-delta alignment fixes *which*. The two counts agree, which
is the consistency check — see the row-count self-check below.

Rows are `aligned` (ordinal inference, same evidence class as the v83 `aligned`
rows — the byte is real and the client renders *a* notice for it, but the exact
English wording is unconfirmed) or `verified` where an anchor names the row
directly. No `aligned` row is restated as `verified` anywhere.

| Key | Byte | StringPool id | v83 id | Conf. | Evidence (case body addr) |
|---|---|---|---|---|---|
| REQUEST_TIMED_OUT | 0x80 (128) | 547 | 534 | aligned | `0x463f0f` |
| NOT_ENOUGH_CASH | 0x82 (130) | 548 | 535 | aligned | `0x463f25` |
| CANNOT_GIFT_WHEN_UNDERAGE | 0x83 (131) | 549 | 536 | aligned | `0x463f3b` |
| EXCEEDED_GIFT_LIMIT | 0x84 (132) | 550 | 537 | aligned | `0x463f51` |
| EXCEEDED_CASH_ITEM_LIMIT | 0x85 (133) | 551 | 538 | aligned | `0x463f67` |
| INCORRECT_NAME_OR_GENDER_RESTRICTION | 0x86 (134) | 552 | 539 | aligned | `0x463f7d` |
| INVALID_COUPON_CODE | 0x87 (135) | 560 | 547 | aligned | `0x46402d` |
| COUPON_EXPIRED | 0x88 (136) | 556 | 543 | aligned | `0x463fd5` |
| COUPON_ALREADY_USED | 0x89 (137) | 555 | 542 | aligned | `0x463fbf` |
| COUPON_INTERNET_CAFE_RESTRICTION | 0x8A (138) | 557 | 544 | aligned | `0x463feb` |
| INTERNET_CAFE_COUPON_ALREADY_USED | 0x8B (139) | 558 | 545 | aligned | `0x464001` |
| INTERNET_CAFE_COUPON_EXPIRED | 0x8C (140) | 559 | 546 | aligned | `0x464017` |
| COUPON_NOT_REGISTERED | 0x8D (141) | 3194 | 3274 | aligned | `0x46414b` |
| COUPON_GENDER_RESTRICTION | 0x8E (142) | 554 | 541 | aligned | `0x463fa9` |
| COUPON_CANNOT_BE_GIFTED | 0x8F (143) | 563 | 551 | aligned | `0x464043` |
| COUPON_ONLY_FOR_MAPLE_STORY | 0x90 (144) | 564 | 552 | aligned | `0x464059` |
| INVENTORY_FULL | 0x91 (145) | 836 | 853 | aligned | `0x46406f` |
| NOT_AVAILABLE_FOR_PURCHASE | 0x92 (146) | 565 | 553 | aligned | `0x464085` |
| CANNOT_GIFT_INVALID_NAME_OR_GENDER | 0x93 (147) | 566 | 554 | aligned | `0x46409b` |
| CHECK_NAME_OF_RECEIVER | 0x94 (148) | 590 | 583 | aligned | `0x4640b1` |
| NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR | 0x95 (149) | 2465 | 2505 | aligned | `0x4640c7` |
| OUT_OF_STOCK | 0x96 (150) | 2466 | 2506 | aligned | `0x4640dd` |
| EXCEEDED_SPENDING_LIMIT | 0x97 (151) | 567 | 555 | aligned | `0x4640f3` |
| NOT_ENOUGH_MESOS | 0x98 (152) | 278 | 5599 | aligned | `0x464109` |
| CASH_SHOP_NOT_AVAILABLE_DURING_BETA | 0x99 (153) | 568 | 556 | **verified (anchor B)** | `0x46411f` |
| INVALID_BIRTHDAY | 0x9A (154) | 527 | 514 | aligned | `0x464135` |
| ONLY_AVAILABLE_TO_USERS_BUYING | 0x9D (157) | 3195 | 3275 | aligned | `0x464161` |
| ALREADY_APPLIED | 0x9E (158) | 3480 | 3558 | aligned | `0x464177` |
| DAILY_PURCHASE_LIMIT | 0xA3 (163) | 553 | 540 | aligned | `0x463f93` |
| COUPON_USAGE_LIMIT | 0xA6 (166) | 4380 | 4457 | aligned | `0x46418d` |
| COUPON_SYSTEM_AVAILABLE_SOON | 0xA7 (167) | inline literal | inline | **verified (anchor A)** | `0x4641a3` |
| FIFTEEN_DAY_LIMIT | 0xA8 (168) | 3670 | 3749 | aligned | `0x4641c0` |
| NOT_ENOUGH_GIFT_TOKENS | 0xA9 (169) | 3917 | 3992 | aligned | `0x4641d6` |
| CANNOT_SEND_TECHNICAL_DIFFICULTIES | 0xAA (170) | 3918 | 3993 | aligned | `0x4641ec` |
| CANNOT_GIFT_ACCOUNT_AGE | 0xAB (171) | 3919 | 3994 | aligned | `0x464202` |
| CANNOT_GIFT_PREVIOUS_INFRACTIONS | 0xAC (172) | 3920 | 3995 | aligned | `0x464218` |
| CANNOT_GIFT_AT_THIS_TIME | 0xAD (173) | 3921 | 3996 | aligned | `0x46422e` |
| CANNOT_GIFT_LIMIT | 0xAE (174) | 3922 | 3997 | aligned | `0x464244` |
| CANNOT_GIFT_TECHNICAL_DIFFICULTIES | 0xAF (175) | 3923 | 3998 | aligned | `0x464257` |
| CANNOT_TRANSFER_UNDER_LEVEL_TWENTY | 0xB0 (176) | 3927 | 4002 | aligned | `0x46426a` |
| CANNOT_TRANSFER_TO_SAME_WORLD | 0xB1 (177) | 3930 | 4005 | aligned | `0x46427d` |
| CANNOT_TRANSFER_TO_NEW_WORLD | 0xB2 (178) | 3931 | 4006 | aligned | `0x464290` |
| CANNOT_TRANSFER_OUT | 0xB3 (179) | 3932 | 4007 | aligned | `0x4642a3` |
| UNKNOWN_ERROR | *(any unlisted byte)* | 569 | 557 | verified | default case `0x4642b6` |

#### Atlas keys ABSENT from this version (10) — no case exists

These are absent-by-enumeration, not unknown: the full case list above is
complete for the switch's whole domain, so a key with no row has no byte.

- `CANNOT_GIFT_TO_OWN_ACCOUNT`
- `INCORRECT_NAME`
- `CANNOT_GIFT_GENDER_RESTRICTION`
- `CANNOT_GIFT_RECIPIENT_INVENTORY_FULL`
- `CANNOT_TRANSFER_NO_EMPTY_SLOTS`
- `EVENT_ENDED_OR_CANT_BE_FREELY_TESTED`
- `CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS`
- `PLEASE_TRY_AGAIN`
- `CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN`
- `CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN`

**Row-count self-check:** 43 mapped rows + 10 absent = 53 Atlas keys;
43 explicit cases = 52 domain values − 9 default-set values. ✅

**Reserved reason bytes that are NOT notices** (must never be mapped to an
Atlas key — they change client state):

| Byte | Behaviour | Evidence |
|---|---|---|
| 127 (0x7F) | `NoticeFailReason` (default text) **then** `CCashShop::SendTransferFieldPacket` — kicks the player out of the cash shop | `CCashShop::OnCashItemResUseCouponFailed` @ `0x462936`, branch `0x462953` |
| 129 (0x81) | same as 127 | `0x462953` |

Neither byte is an explicit case (127 is below the switch domain / in the
default set, 129 is in the default set), so neither can collide with a mapped
key — machine-checked. **This version has NO coupon-specific
"wrong number → `OnStatusExit`" byte**: unlike v83 (`0xB1`) / v84 (186) /
v87 (192) / v92 (15), the `else` arm of `OnCashItemResUseCouponFailed` is a
plain `NoticeFailReason` call with no follow-up — same shape as jms_v185.

---

## gms_v72

IDB `GMS_v72.1_U_DEVM.exe.i64`, session `c8acae95`. Failure-reason sink:
`CCashShop::NoticeFailReason` @ `0x473ba9` (already named in this IDB).

### `errors` enum

**Jump-table header (read, not inferred).** `add eax, 0FFFFFF73h` (bias −141) @ `0x473baf`,
`cmp eax, 3Ah` @ `0x473bb4`, IDA annotation **"switch 59 cases"** with
**"default case, cases 142,172,173,176-179,181,182,184"**. Domain **141…199** (59 values) minus the
10 default-set values ⇒ **49 explicit cases**, which equals the 49 distinct
jump-table targets IDA lists at the `jmp ds:jpt_…` line.

**The constant-offset hypothesis FAILS — nothing here is offset-derived.**
The test the v84/v87/v92/v95 passes used is exact set-equality of the
default-case set under a constant offset. v83's domain is 163…233 (71 cases,
18-member default set); this version's is 141…199 (59 cases, 10-member default
set). Neither the case count, the range, nor the default set maps under any
offset — in particular the offset the `OnStatusCoupon` gate would imply
(170 − 195 = -25) predicts a domain starting at 138, not 141. **Rejected.**
Per the stop rule, every case body was instead decompiled individually and
its `StringPool` id read (addresses in the table below).

**How the keys were assigned.** The 49 explicit cases were aligned ordinally
against v83's 53-case list. Four independent constraints pin that alignment:

- **Anchor A (exact, direct).** Exactly one case in this version does not use
  the StringPool — it builds the inline literal `"The coupon system will be
  available soon."` (`aTheCouponSyste`). It is case **185** @ `0x473e5a`, at
  ordinal position 35 of this version's case list. v83's inline-literal case
  (`0xD2`) is at ordinal position 35 of its 53. ⇒ `COUPON_SYSTEM_AVAILABLE_SOON`.
- **Anchor B (exact, call site).** `CCashShop::OnStatusCoupon` @ `0x4698f5`
  calls `NoticeFailReason(this, 170)` when the cash-shop-loaded flag
  `[this+0x40]` is clear (address from §"Legacy versions — COUPON_CODE
  applicability"). Case 170 sits at ordinal position 29 here; v83's gate (195)
  sits at ordinal 29 of its 53. ⇒ `CASH_SHOP_NOT_AVAILABLE_DURING_BETA`.
- **Anchor C (structural, id-block deltas).** StringPool ids are not stable
  across GMS versions (this version's default-case id is 558; v83's is 557), so
  id *equality* proves nothing. What is stable is the *internal delta pattern*
  of each contiguous run. Every multi-member run in this version's id list has
  a byte-identical delta pattern to the correspondingly-positioned run in v83's
  (see the `v83 id` column below) — a run inserted or deleted anywhere would
  break it.
- **Anchor D (paired singleton).** v83 uses two consecutive ids at two far-apart
  ordinals — 3274 at ordinal 17 (`COUPON_NOT_REGISTERED`) and 3275 at ordinal 31
  (`ONLY_AVAILABLE_TO_USERS_BUYING`). This version reproduces that signature
  exactly with 3249 / 3250 at the correspondingly-mapped ordinals.

Anchors A and B independently fix how many keys are missing *before* each of
them, and the id-delta alignment fixes *which*. The two counts agree, which
is the consistency check — see the row-count self-check below.

Rows are `aligned` (ordinal inference, same evidence class as the v83 `aligned`
rows — the byte is real and the client renders *a* notice for it, but the exact
English wording is unconfirmed) or `verified` where an anchor names the row
directly. No `aligned` row is restated as `verified` anywhere.

| Key | Byte | StringPool id | v83 id | Conf. | Evidence (case body addr) |
|---|---|---|---|---|---|
| REQUEST_TIMED_OUT | 0x8D (141) | 536 | 534 | aligned | `0x473bc6` |
| NOT_ENOUGH_CASH | 0x8F (143) | 537 | 535 | aligned | `0x473bdc` |
| CANNOT_GIFT_WHEN_UNDERAGE | 0x90 (144) | 538 | 536 | aligned | `0x473bf2` |
| EXCEEDED_GIFT_LIMIT | 0x91 (145) | 539 | 537 | aligned | `0x473c08` |
| CANNOT_GIFT_TO_OWN_ACCOUNT | 0x92 (146) | 4509 | 4537 | aligned | `0x473fa5` |
| INCORRECT_NAME | 0x93 (147) | 4510 | 4538 | aligned | `0x473fb8` |
| CANNOT_GIFT_GENDER_RESTRICTION | 0x94 (148) | 4511 | 4539 | aligned | `0x473fcb` |
| CANNOT_GIFT_RECIPIENT_INVENTORY_FULL | 0x95 (149) | 4512 | 4540 | aligned | `0x473fde` |
| EXCEEDED_CASH_ITEM_LIMIT | 0x96 (150) | 540 | 538 | aligned | `0x473c1e` |
| INCORRECT_NAME_OR_GENDER_RESTRICTION | 0x97 (151) | 541 | 539 | aligned | `0x473c34` |
| INVALID_COUPON_CODE | 0x98 (152) | 549 | 547 | aligned | `0x473ce4` |
| COUPON_EXPIRED | 0x99 (153) | 545 | 543 | aligned | `0x473c8c` |
| COUPON_ALREADY_USED | 0x9A (154) | 544 | 542 | aligned | `0x473c76` |
| COUPON_INTERNET_CAFE_RESTRICTION | 0x9B (155) | 546 | 544 | aligned | `0x473ca2` |
| INTERNET_CAFE_COUPON_ALREADY_USED | 0x9C (156) | 547 | 545 | aligned | `0x473cb8` |
| INTERNET_CAFE_COUPON_EXPIRED | 0x9D (157) | 548 | 546 | aligned | `0x473cce` |
| COUPON_NOT_REGISTERED | 0x9E (158) | 3249 | 3274 | aligned | `0x473e02` |
| COUPON_GENDER_RESTRICTION | 0x9F (159) | 543 | 541 | aligned | `0x473c60` |
| COUPON_CANNOT_BE_GIFTED | 0xA0 (160) | 552 | 551 | aligned | `0x473cfa` |
| COUPON_ONLY_FOR_MAPLE_STORY | 0xA1 (161) | 553 | 552 | aligned | `0x473d10` |
| INVENTORY_FULL | 0xA2 (162) | 854 | 853 | aligned | `0x473d26` |
| NOT_AVAILABLE_FOR_PURCHASE | 0xA3 (163) | 554 | 553 | aligned | `0x473d3c` |
| CANNOT_GIFT_INVALID_NAME_OR_GENDER | 0xA4 (164) | 555 | 554 | aligned | `0x473d52` |
| CHECK_NAME_OF_RECEIVER | 0xA5 (165) | 584 | 583 | aligned | `0x473d68` |
| NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR | 0xA6 (166) | 2488 | 2505 | aligned | `0x473d7e` |
| OUT_OF_STOCK | 0xA7 (167) | 2489 | 2506 | aligned | `0x473d94` |
| EXCEEDED_SPENDING_LIMIT | 0xA8 (168) | 556 | 555 | aligned | `0x473daa` |
| NOT_ENOUGH_MESOS | 0xA9 (169) | 5092 | 5599 | aligned | `0x473dc0` |
| CASH_SHOP_NOT_AVAILABLE_DURING_BETA | 0xAA (170) | 557 | 556 | **verified (anchor B)** | `0x473dd6` |
| INVALID_BIRTHDAY | 0xAB (171) | 516 | 514 | aligned | `0x473dec` |
| ONLY_AVAILABLE_TO_USERS_BUYING | 0xAE (174) | 3250 | 3275 | aligned | `0x473e18` |
| ALREADY_APPLIED | 0xAF (175) | 3534 | 3558 | aligned | `0x473e2e` |
| DAILY_PURCHASE_LIMIT | 0xB4 (180) | 542 | 540 | aligned | `0x473c4a` |
| COUPON_USAGE_LIMIT | 0xB7 (183) | 4436 | 4457 | aligned | `0x473e44` |
| COUPON_SYSTEM_AVAILABLE_SOON | 0xB9 (185) | inline literal | inline | **verified (anchor A)** | `0x473e5a` |
| FIFTEEN_DAY_LIMIT | 0xBA (186) | 3724 | 3749 | aligned | `0x473e77` |
| NOT_ENOUGH_GIFT_TOKENS | 0xBB (187) | 3968 | 3992 | aligned | `0x473e8d` |
| CANNOT_SEND_TECHNICAL_DIFFICULTIES | 0xBC (188) | 3969 | 3993 | aligned | `0x473ea3` |
| CANNOT_GIFT_ACCOUNT_AGE | 0xBD (189) | 3970 | 3994 | aligned | `0x473eb9` |
| CANNOT_GIFT_PREVIOUS_INFRACTIONS | 0xBE (190) | 3971 | 3995 | aligned | `0x473ecf` |
| CANNOT_GIFT_AT_THIS_TIME | 0xBF (191) | 3972 | 3996 | aligned | `0x473ee5` |
| CANNOT_GIFT_LIMIT | 0xC0 (192) | 3973 | 3997 | aligned | `0x473efb` |
| CANNOT_GIFT_TECHNICAL_DIFFICULTIES | 0xC1 (193) | 3974 | 3998 | aligned | `0x473f11` |
| CANNOT_TRANSFER_UNDER_LEVEL_TWENTY | 0xC2 (194) | 3978 | 4002 | aligned | `0x473f27` |
| CANNOT_TRANSFER_TO_SAME_WORLD | 0xC3 (195) | 3981 | 4005 | aligned | `0x473f3d` |
| CANNOT_TRANSFER_TO_NEW_WORLD | 0xC4 (196) | 3982 | 4006 | aligned | `0x473f53` |
| CANNOT_TRANSFER_OUT | 0xC5 (197) | 3983 | 4007 | aligned | `0x473f69` |
| CANNOT_TRANSFER_NO_EMPTY_SLOTS | 0xC6 (198) | 3984 | 4008 | aligned | `0x473f7f` |
| EVENT_ENDED_OR_CANT_BE_FREELY_TESTED | 0xC7 (199) | 3996 | 4020 | aligned | `0x473f92` |
| UNKNOWN_ERROR | *(any unlisted byte)* | 558 | 557 | verified | default case `0x473ff1` |

#### Atlas keys ABSENT from this version (4) — no case exists

These are absent-by-enumeration, not unknown: the full case list above is
complete for the switch's whole domain, so a key with no row has no byte.

- `CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS`
- `PLEASE_TRY_AGAIN`
- `CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN`
- `CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN`

**Row-count self-check:** 49 mapped rows + 4 absent = 53 Atlas keys;
49 explicit cases = 59 domain values − 10 default-set values. ✅

**Reserved reason bytes that are NOT notices** (must never be mapped to an
Atlas key — they change client state):

| Byte | Behaviour | Evidence |
|---|---|---|
| 140 (0x8C) | `NoticeFailReason` (default text) **then** `CCashShop::SendTransferFieldPacket` — kicks the player out of the cash shop | `CCashShop::OnCashItemResUseCouponFailed` @ `0x4724a4`, branch `0x4724c3` |
| 142 (0x8E) | same as 140 | `0x4724c3` |

Neither byte is an explicit case (140 is below the switch domain / in the
default set, 142 is in the default set), so neither can collide with a mapped
key — machine-checked. **This version has NO coupon-specific
"wrong number → `OnStatusExit`" byte**: unlike v83 (`0xB1`) / v84 (186) /
v87 (192) / v92 (15), the `else` arm of `OnCashItemResUseCouponFailed` is a
plain `NoticeFailReason` call with no follow-up — same shape as jms_v185.

---

## gms_v79

IDB `GMS_v79_1_DEVM.exe.i64`, session `1438cecd`. Failure-reason sink:
`CCashShop::NoticeFailReason` @ `0x475075` (already named in this IDB).

### `errors` enum

**Jump-table header (read, not inferred).** `add eax, 0FFFFFF65h` (bias −155) @ `0x47507b`,
`cmp eax, 43h` @ `0x475080`, IDA annotation **"switch 68 cases"** with
**"default case, cases 156,186,187,190-193,195,196,198,214-218"**. Domain **155…222** (68 values) minus the
15 default-set values ⇒ **53 explicit cases**, which equals the 53 distinct
jump-table targets IDA lists at the `jmp ds:jpt_…` line.

**The constant-offset hypothesis FAILS — nothing here is offset-derived.**
The test the v84/v87/v92/v95 passes used is exact set-equality of the
default-case set under a constant offset. v83's domain is 163…233 (71 cases,
18-member default set); this version's is 155…222 (68 cases, 15-member default
set). Neither the case count, the range, nor the default set maps under any
offset — in particular the offset the `OnStatusCoupon` gate would imply
(184 − 195 = -11) predicts a domain starting at 152, not 155. **Rejected.**
Per the stop rule, every case body was instead decompiled individually and
its `StringPool` id read (addresses in the table below).

**How the keys were assigned.** The 53 explicit cases were aligned ordinally
against v83's 53-case list. Four independent constraints pin that alignment:

- **Anchor A (exact, direct).** Exactly one case in this version does not use
  the StringPool — it builds the inline literal `"The coupon system will be
  available soon."` (`aTheCouponSyste`). It is case **199** @ `0x475326`, at
  ordinal position 35 of this version's case list. v83's inline-literal case
  (`0xD2`) is at ordinal position 35 of its 53. ⇒ `COUPON_SYSTEM_AVAILABLE_SOON`.
- **Anchor B (exact, call site).** `CCashShop::OnStatusCoupon` @ `0x46aa5b`
  calls `NoticeFailReason(this, 184)` when the cash-shop-loaded flag
  `[this+0x40]` is clear (address from §"Legacy versions — COUPON_CODE
  applicability"). Case 184 sits at ordinal position 29 here; v83's gate (195)
  sits at ordinal 29 of its 53. ⇒ `CASH_SHOP_NOT_AVAILABLE_DURING_BETA`.
- **Anchor C (structural, id-block deltas).** StringPool ids are not stable
  across GMS versions (this version's default-case id is 558; v83's is 557), so
  id *equality* proves nothing. What is stable is the *internal delta pattern*
  of each contiguous run. Every multi-member run in this version's id list has
  a byte-identical delta pattern to the correspondingly-positioned run in v83's
  (see the `v83 id` column below) — a run inserted or deleted anywhere would
  break it.
- **Anchor D (paired singleton).** v83 uses two consecutive ids at two far-apart
  ordinals — 3274 at ordinal 17 (`COUPON_NOT_REGISTERED`) and 3275 at ordinal 31
  (`ONLY_AVAILABLE_TO_USERS_BUYING`). This version reproduces that signature
  exactly with 3254 / 3255 at the correspondingly-mapped ordinals.

Case count 53 equals v83's 53 and both anchors land on their v83 ordinal
unchanged (29 and 35), so the alignment is a straight 1:1 with no holes.

Rows are `aligned` (ordinal inference, same evidence class as the v83 `aligned`
rows — the byte is real and the client renders *a* notice for it, but the exact
English wording is unconfirmed) or `verified` where an anchor names the row
directly. No `aligned` row is restated as `verified` anywhere.

| Key | Byte | StringPool id | v83 id | Conf. | Evidence (case body addr) |
|---|---|---|---|---|---|
| REQUEST_TIMED_OUT | 0x9B (155) | 536 | 534 | aligned | `0x475092` |
| NOT_ENOUGH_CASH | 0x9D (157) | 537 | 535 | aligned | `0x4750a8` |
| CANNOT_GIFT_WHEN_UNDERAGE | 0x9E (158) | 538 | 536 | aligned | `0x4750be` |
| EXCEEDED_GIFT_LIMIT | 0x9F (159) | 539 | 537 | aligned | `0x4750d4` |
| CANNOT_GIFT_TO_OWN_ACCOUNT | 0xA0 (160) | 4499 | 4537 | aligned | `0x475477` |
| INCORRECT_NAME | 0xA1 (161) | 4500 | 4538 | aligned | `0x47548d` |
| CANNOT_GIFT_GENDER_RESTRICTION | 0xA2 (162) | 4501 | 4539 | aligned | `0x4754a3` |
| CANNOT_GIFT_RECIPIENT_INVENTORY_FULL | 0xA3 (163) | 4502 | 4540 | aligned | `0x4754b6` |
| EXCEEDED_CASH_ITEM_LIMIT | 0xA4 (164) | 540 | 538 | aligned | `0x4750ea` |
| INCORRECT_NAME_OR_GENDER_RESTRICTION | 0xA5 (165) | 541 | 539 | aligned | `0x475100` |
| INVALID_COUPON_CODE | 0xA6 (166) | 549 | 547 | aligned | `0x4751b0` |
| COUPON_EXPIRED | 0xA7 (167) | 545 | 543 | aligned | `0x475158` |
| COUPON_ALREADY_USED | 0xA8 (168) | 544 | 542 | aligned | `0x475142` |
| COUPON_INTERNET_CAFE_RESTRICTION | 0xA9 (169) | 546 | 544 | aligned | `0x47516e` |
| INTERNET_CAFE_COUPON_ALREADY_USED | 0xAA (170) | 547 | 545 | aligned | `0x475184` |
| INTERNET_CAFE_COUPON_EXPIRED | 0xAB (171) | 548 | 546 | aligned | `0x47519a` |
| COUPON_NOT_REGISTERED | 0xAC (172) | 3254 | 3274 | aligned | `0x4752ce` |
| COUPON_GENDER_RESTRICTION | 0xAD (173) | 543 | 541 | aligned | `0x47512c` |
| COUPON_CANNOT_BE_GIFTED | 0xAE (174) | 552 | 551 | aligned | `0x4751c6` |
| COUPON_ONLY_FOR_MAPLE_STORY | 0xAF (175) | 553 | 552 | aligned | `0x4751dc` |
| INVENTORY_FULL | 0xB0 (176) | 853 | 853 | aligned | `0x4751f2` |
| NOT_AVAILABLE_FOR_PURCHASE | 0xB1 (177) | 554 | 553 | aligned | `0x475208` |
| CANNOT_GIFT_INVALID_NAME_OR_GENDER | 0xB2 (178) | 555 | 554 | aligned | `0x47521e` |
| CHECK_NAME_OF_RECEIVER | 0xB3 (179) | 584 | 583 | aligned | `0x475234` |
| NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR | 0xB4 (180) | 2490 | 2505 | aligned | `0x47524a` |
| OUT_OF_STOCK | 0xB5 (181) | 2491 | 2506 | aligned | `0x475260` |
| EXCEEDED_SPENDING_LIMIT | 0xB6 (182) | 556 | 555 | aligned | `0x475276` |
| NOT_ENOUGH_MESOS | 0xB7 (183) | 5347 | 5599 | aligned | `0x47528c` |
| CASH_SHOP_NOT_AVAILABLE_DURING_BETA | 0xB8 (184) | 557 | 556 | **verified (anchor B)** | `0x4752a2` |
| INVALID_BIRTHDAY | 0xB9 (185) | 516 | 514 | aligned | `0x4752b8` |
| ONLY_AVAILABLE_TO_USERS_BUYING | 0xBC (188) | 3255 | 3275 | aligned | `0x4752e4` |
| ALREADY_APPLIED | 0xBD (189) | 3538 | 3558 | aligned | `0x4752fa` |
| DAILY_PURCHASE_LIMIT | 0xC2 (194) | 542 | 540 | aligned | `0x475116` |
| COUPON_USAGE_LIMIT | 0xC5 (197) | 4428 | 4457 | aligned | `0x475310` |
| COUPON_SYSTEM_AVAILABLE_SOON | 0xC7 (199) | inline literal | inline | **verified (anchor A)** | `0x475326` |
| FIFTEEN_DAY_LIMIT | 0xC8 (200) | 3728 | 3749 | aligned | `0x475343` |
| NOT_ENOUGH_GIFT_TOKENS | 0xC9 (201) | 3971 | 3992 | aligned | `0x475359` |
| CANNOT_SEND_TECHNICAL_DIFFICULTIES | 0xCA (202) | 3972 | 3993 | aligned | `0x47536f` |
| CANNOT_GIFT_ACCOUNT_AGE | 0xCB (203) | 3973 | 3994 | aligned | `0x475385` |
| CANNOT_GIFT_PREVIOUS_INFRACTIONS | 0xCC (204) | 3974 | 3995 | aligned | `0x47539b` |
| CANNOT_GIFT_AT_THIS_TIME | 0xCD (205) | 3975 | 3996 | aligned | `0x4753b1` |
| CANNOT_GIFT_LIMIT | 0xCE (206) | 3976 | 3997 | aligned | `0x4753c7` |
| CANNOT_GIFT_TECHNICAL_DIFFICULTIES | 0xCF (207) | 3977 | 3998 | aligned | `0x4753dd` |
| CANNOT_TRANSFER_UNDER_LEVEL_TWENTY | 0xD0 (208) | 3981 | 4002 | aligned | `0x4753f3` |
| CANNOT_TRANSFER_TO_SAME_WORLD | 0xD1 (209) | 3984 | 4005 | aligned | `0x475409` |
| CANNOT_TRANSFER_TO_NEW_WORLD | 0xD2 (210) | 3985 | 4006 | aligned | `0x47541f` |
| CANNOT_TRANSFER_OUT | 0xD3 (211) | 3986 | 4007 | aligned | `0x475435` |
| CANNOT_TRANSFER_NO_EMPTY_SLOTS | 0xD4 (212) | 3987 | 4008 | aligned | `0x47544b` |
| EVENT_ENDED_OR_CANT_BE_FREELY_TESTED | 0xD5 (213) | 3999 | 4020 | aligned | `0x475461` |
| CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS | 0xDB (219) | 5025 | 5063 | aligned | `0x4754c9` |
| PLEASE_TRY_AGAIN | 0xDC (220) | 5026 | 5064 | aligned | `0x4754dc` |
| CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN | 0xDD (221) | 5324 | 5573 | aligned | `0x4754ef` |
| CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN | 0xDE (222) | 5325 | 5574 | aligned | `0x475502` |
| UNKNOWN_ERROR | *(any unlisted byte)* | 558 | 557 | verified | default case `0x475515` |

**Row-count self-check:** 53 mapped rows + 0 absent = 53 Atlas keys;
53 explicit cases = 68 domain values − 15 default-set values. ✅

**Reserved reason bytes that are NOT notices** (must never be mapped to an
Atlas key — they change client state):

| Byte | Behaviour | Evidence |
|---|---|---|
| 154 (0x9A) | `NoticeFailReason` (default text) **then** `CCashShop::SendTransferFieldPacket` — kicks the player out of the cash shop | `CCashShop::OnCashItemResUseCouponFailed` @ `0x473769`, branch `0x473788` |
| 156 (0x9C) | same as 154 | `0x473788` |

Neither byte is an explicit case (154 is below the switch domain / in the
default set, 156 is in the default set), so neither can collide with a mapped
key — machine-checked. **This version has NO coupon-specific
"wrong number → `OnStatusExit`" byte**: unlike v83 (`0xB1`) / v84 (186) /
v87 (192) / v92 (15), the `else` arm of `OnCashItemResUseCouponFailed` is a
plain `NoticeFailReason` call with no follow-up — same shape as jms_v185.

---

## Legacy versions — COUPON_CODE applicability

The four legacy templates bind the CLIENTBOUND coupon arms
(USE_COUPON_SUCCESS/USE_COUPON_FAILED at 54/57, 61/64, 69/72, 81/84), so the
receive half exists on all four. This section settles the SEND half, which is
what the registry and the coverage matrix key `n-a` on.

**Verdict: `present` on all four.** The registry `n-a` was "nobody looked", not
"the client cannot send" — every one of the four binaries carries a fully
implemented `CCashShop::OnStatusCoupon` that builds and sends the packet.

| version | IDB binary | session | OnStatusCoupon | "Please enter the coupon code." | opcode+1 push | verdict |
|---|---|---|---|---|---|---|
| gms_v48 | `GMS_v48_1_DEVM.exe.i64` | `93cc947e` | `?OnStatusCoupon@CCashShop@@QAEXXZ` @ `0x44d2e7` (size `0x113`) | **not found** — `find_regex "enter the coupon code"` returned 0 matches; the broader `find_regex "coupon"` returned exactly 3 strings, none of them this one (`0x7faf1c` "The coupon system will be available soon.", `0x7fe000` "…1 [Quick Delivery Coupon]…", `0x7fe0dc` "You do not have the Quick Delivery Coupon."). v48 has no empty-code guard at all — see note below. | `push 0A1h` @ `0x44d340` (= 161 = `CASHSHOP_OPERATION` 160 + 1) | **present** |
| gms_v61 | `GMS_v61.1_U_DEVM.exe.i64` | `415bf585` | `?OnStatusCoupon@CCashShop@@QAEXXZ` @ `0x45a6b5` (size `0x158`) | `aPleaseEnterThe` @ `0x9614e8`, referenced from the empty-code else-arm: `CDialog::DoModal(aPleaseEnterThe, -1)` @ `0x45a741` inside `OnStatusCoupon` | `push 0C5h` @ `0x45a753` (= 197 = `CASHSHOP_OPERATION` 196 + 1) | **present** |
| gms_v72 | `GMS_v72.1_U_DEVM.exe.i64` | `c8acae95` | `?OnStatusCoupon@CCashShop@@QAEXXZ` @ `0x4698d8` (size `0x158`) | `aPleaseEnterThe` @ `0xa5aa6c`, referenced from `CDialog::DoModal(aPleaseEnterThe, -1)` @ `0x469964` inside `OnStatusCoupon` | `push 0DCh` @ `0x469976` (= 220 = `CASHSHOP_OPERATION` 219 + 1) | **present** |
| gms_v79 | `GMS_v79_1_DEVM.exe.i64` | `1438cecd` | `?OnStatusCoupon@CCashShop@@QAEXXZ` @ `0x46aa3e` (size `0x158`) | `aPleaseEnterThe` @ `0xabed1c`, referenced from `CDialog::DoModal(aPleaseEnterThe, -1)` @ `0x46aaca` inside `OnStatusCoupon` | `push 0DEh` @ `0x46aae4` call site, immediate at `0x46aadc` (= 222 = `CASHSHOP_OPERATION` 221 + 1) | **present** |

All four functions were already named in their IDBs — no `rename` was needed.
The serverbound `CASHSHOP_OPERATION` opcodes were read from the registries
(`gms_v48.yaml:1260` = 160, `gms_v61.yaml` = 196, `gms_v72.yaml` = 219,
`gms_v79.yaml` = 221), **not** from the brief's clientbound-flavoured list; the
`+1` relation holds against the serverbound values on all four.

### Request body — identical to the gms_v83 shape on all four

Every legacy send is three `EncodeStr` calls and **no** `Encode1` — so the JMS
`nType` divergence (see `## jms_v185`) does not appear here, and the v92/v95
drop of the third field has not happened yet either. Field order, with the
stack slot that carries each value:

1. `EncodeStr(characterId)` — the *second* argument of
   `CCouponUseSelectDlg::Confirm(ZXString<char>&, ZXString<char>&)`; never
   populated by the dialog, so it is always the empty string on the wire.
2. `EncodeStr(couponCode)` — the *first* `Confirm` argument, the one
   `CCtrlEdit::GetText` writes the typed code into.
3. `EncodeStr(<third string>)` — guarded by `if (characterId && *characterId)`,
   i.e. dead code, exactly as on v83, because field 1 is always empty.

| version | ctor | field 1 `EncodeStr` | field 2 `EncodeStr` | guard | field 3 `EncodeStr` | `SendPacket` |
|---|---|---|---|---|---|---|
| gms_v48 | `COutPacket(0xA1)` @ `0x44d348` | `0x44d365` (`var_10`) | `0x44d37e` (`var_14`) | `cmp eax, ebx` / `cmp [eax], bl` @ `0x44d386`–`0x44d38c` on `var_10` | `0x44d3a2` (`var_18`) | `0x44d3b1` |
| gms_v61 | `COutPacket(0xC5)` @ `0x45a75b` | `0x45a778` (`var_14`) | `0x45a791` (`var_10`) | `0x45a799`–`0x45a79f` on `var_14` | `0x45a7b5` (`var_18`) | `0x45a7c4` |
| gms_v72 | `COutPacket(0xDC)` @ `0x46997e` | `0x46999b` (`var_14`) | `0x4699b4` (`var_10`) | `0x4699bc`–`0x4699c2` on `var_14` | `0x4699d8` (`var_18`) | `0x4699e7` |
| gms_v79 | `COutPacket(0xDE)` @ `0x46aae4` | `0x46ab01` (`var_14`) | `0x46ab1a` (`var_10`) | `0x46ab22`–`0x46ab28` on `var_14` | `0x46ab3e` (`var_18`) | `0x46ab4d` |

The v48 slot assignment is transposed relative to v61/v72/v79 (`var_10` is
characterId there, `var_14` on the later three), but the *wire* order is the
same on all four. v48's `Confirm` call at `0x44d32f` pushes `var_10` then
`var_14`, so `var_14` is argument 1 — and `Confirm` @ `0x44d3fa` assigns
`CCtrlEdit::GetText` (`0x44d4cc`) into argument 1. That fixes `var_14` as the
coupon code and `var_10` as characterId.

### One real behavioural divergence: v48 has no client-side empty-code guard

v61/v72/v79 (and v83+) call `ZXString<char>::TrimRight` / `TrimLeft` on the
typed code, test its length, and on empty show
`"Please enter the coupon code."` via `CDialog::DoModal` + `CUtilDlg::Notice`
*instead of* sending. v48 does none of that: `OnStatusCoupon` @ `0x44d2e7`
goes straight from `Confirm(...) == 1` (`0x44d334`) to
`COutPacket(0xA1)` (`0x44d348`). That is why the literal is absent from the
whole v48 binary — it is not a missing-feature signal, it is a missing-guard
one. **Server consequence:** a v48 client can send an empty / untrimmed coupon
code, so the server must trim and reject empties itself rather than relying on
the client filter that the other five versions apply.

The other guard is shared by all four: the entry `NoticeFailReason` early-out
when the cash-shop-loaded flag (`this+0x40`) is clear — reason `137` on v48
(`0x44d304`), `153` on v61 (`0x45a6d2`), `170` on v72 (`0x4698f5`), `184` on
v79 (`0x46aa5b`). And all four set the in-flight latch `this+0x18 = 1`
after `SendPacket`, so the client self-throttles to one outstanding coupon
request.

---

## Cross-version summary of the two values the codec needs

| version | `COUPON_CODE` opcode | body |
|---|---|---|
| gms_v48 | 161 / `0xA1` | `str characterId` · `str couponCode` · optional `str` (dead: field 1 always empty). No client-side empty-code guard. |
| gms_v61 | 197 / `0xC5` | same as v48 |
| gms_v72 | 220 / `0xDC` | same as v48 |
| gms_v79 | 222 / `0xDE` | same as v48 |
| gms_v83 | 230 / `0xE6` | `str characterId` · `str couponCode` · optional `str` (dead: field 1 always empty) |
| gms_v84 | **236 / `0xEC`** (registry says 230 — bug) | same as v83 |
| gms_v87 | 243 / `0xF3` | same as v83 |
| gms_v92 | 269 / `0x10D` | `str characterId` · `str couponCode` (no optional third) |
| gms_v95 | 276 / `0x114` | `str characterId` · `str couponCode` (no optional third) |
| jms_v185 | 246 / `0xF6` (registry confirmed) | `str characterId` · `str couponCode` · **`byte nType`** · optional `str` (guarded by `characterId` non-empty) |

| version | reason-byte scale | domain | explicit reasons |
|---|---|---|---|
| gms_v48 | **no constant offset** — ordinal alignment, anchored | 113–155 (`0x71`–`0x9B`) | 40 explicit + default; **40 of 53 Atlas keys mapped, 13 absent** |
| gms_v61 | **no constant offset** — ordinal alignment, anchored | 128–179 (`0x80`–`0xB3`) | 43 explicit + default; **43 of 53 mapped, 10 absent** |
| gms_v72 | **no constant offset** — ordinal alignment, anchored | 141–199 (`0x8D`–`0xC7`) | 49 explicit + default; **49 of 53 mapped, 4 absent** |
| gms_v79 | **no constant offset** — ordinal alignment, anchored, 1:1 with v83 | 155–222 (`0x9B`–`0xDE`) | 53 explicit + default; **all 53 mapped** |
| gms_v83 | v83 baseline | `0xA3`–`0xE9` | 53 + default |
| gms_v84 | v83 **+ 9** | `0xAC`–`0xF2` | 53 + default |
| gms_v87 | v83 **+ 15** | `0xB2`–`0xF9` | 54 (one new) + default |
| gms_v92 | v83 **− 162** (1-based) | 1–69 | 56 (five new, two dropped) + default |
| gms_v95 | identical to gms_v92 | 1–69 | 56 + default |
| jms_v185 | v83 **+ 15** on `[163,196]` **only** — global set-equality FAILS above that | 178–237 (`0xB2`–`0xED`) | 47 explicit + default; **30 of 53 Atlas keys mapped, 23 absent** |
