# re-cashshop-v72 — GMS v72 CashShop (0xDB / 219) deep resolution

Binary: `GMS_v72.1_U_DEVM.exe` (ida-pro session `eb2a156e`). Read-only.
Opcode 0xDB = 219. All evidence quotes the real `COutPacket::COutPacket(v,219)` +
`COutPacket::Encode1(mode)` per key. Completeness: all **29** `COutPacket::COutPacket`
construction sites in the CCashShop code range `0x461000–0x474000` were enumerated
(`search_text COutPacket::COutPacket`) and each decompiled — this resolves the open
items from `re-v72.md` and closes the enum.

## Resolved open items

### INCREASE_CHARACTER_SLOT — ABSENT as a distinct mode (folded into 6/7)
`CCashShop::OnIncCharacterSlotCount` @0x469159 is a general slot-expansion sender,
NOT a "flag-only char-slot" op. Item-id drives the mode:
```
COutPacket::COutPacket((COutPacket *)v31, 219);                       /*0x469159*/
COutPacket::Encode1((COutPacket *)v31, (v19 / 1000 == 9110) + 6);     /*0x469184*/  // mode 6 or 7
COutPacket::Encode1((COutPacket *)v31, v45 == 2);                     /*0x469194*/  // storage-target flag
COutPacket::Encode4((COutPacket *)v31, v45);                          /*0x46919f*/  // target-inv mask
COutPacket::Encode1((COutPacket *)v31, 1u);                           /*0x4691a8*/
COutPacket::Encode4((COutPacket *)v31, (unsigned int)a2);             /*0x4691b3*/  // sn
```
`v19 = TSecType<long>::GetData(...)` = item id. Guard at 0x468ea8 requires
`Data/10000==911 || Data/1000==5430`. So `item/1000==9110` → **mode 7**
(INCREASE_STORAGE lane); everything else (incl. character-slot items) → **mode 6**
(INCREASE_INVENTORY lane). The body is the inventory/storage-expand shape, not the
v83 "flag only" char-slot body. **v72 has no dedicated INCREASE_CHARACTER_SLOT mode
(v83 mode 8 does not exist); it reuses mode 6.**

### MOVE_FROM / MOVE_TO — resolved by body shape (slot field is the disambiguator)
**MOVE_FROM_CASH_INVENTORY = 12 (0xC)** — `sub_46AEC0` @0x46b03e (locker→inv, HAS slot):
```
COutPacket::COutPacket((COutPacket *)v24, 219);          /*0x46b03e*/
COutPacket::Encode1((COutPacket *)v24, 0xCu);            /*0x46b04c*/  // mode 12
COutPacket::EncodeBuffer((COutPacket *)v24, &Src, 8u);   /*0x46b05a*/  // sn (8-byte)
COutPacket::Encode1((COutPacket *)v24, a4);              /*0x46b065*/  // inventoryType
COutPacket::Encode2((COutPacket *)v24, (unsigned __int16)a5); /*0x46b070*/  // slot  <-- extra field
```
**MOVE_TO_CASH_INVENTORY = 13 (0xD)** — `sub_46B0AE` @0x46b187 (inv→locker, NO slot):
```
COutPacket::COutPacket((COutPacket *)v18, 219);          /*0x46b187*/
COutPacket::Encode1((COutPacket *)v18, 0xDu);            /*0x46b197*/  // mode 13
COutPacket::EncodeBuffer((COutPacket *)v18, &Src, 8u);   /*0x46b1a5*/  // sn (8-byte)
COutPacket::Encode1((COutPacket *)v18, (unsigned __int8)a3); /*0x46b1b0*/  // inventoryType
```
Mode 12 carries the trailing `Encode2(slot)`; mode 13 does not. Per the semantics
disambiguator (FROM = sn+invType+**slot**; TO = sn+invType), this is definitive:
**FROM=12, TO=13** (both shifted down 1 from v83's 13/14).

### BUY_NAME_CHANGE = 41 (0x29) — CONFIRMED
`sub_46BCAC` @0x46bccb:
```
COutPacket::COutPacket((COutPacket *)v6, 219);   /*0x46bccb*/
COutPacket::Encode1((COutPacket *)v6, 0x29u);    /*0x46bcd9*/  // mode 41
COutPacket::Encode4((COutPacket *)v6, a2);       /*0x46bce4*/  // sn
COutPacket::EncodeStr(...);                       /*0x46bcfe*/  // string 1 (old name)
COutPacket::EncodeStr(...);                       /*0x46bd18*/  // string 2 (new name)
```
Caller `sub_72BCBB` @0x72bcbb is the name-entry dialog: length 4–12 check
(`v5 < 4` / `v6 > 12` → reject), `CCurseProcess::ProcessString` profanity filter,
and `_strcmpi` against the current name. Identity confirmed (sn + 2 strings, name
dialog). **High confidence.**

### APPLY_WISHLIST = 32 (0x20) — FOUND (re-v72.md's "absent" is corrected)
`sub_46B9D7` @0x46b9f5 — mode-only, no body:
```
COutPacket::COutPacket((COutPacket *)v3, 219);   /*0x46b9f5*/
COutPacket::Encode1((COutPacket *)v3, 0x20u);    /*0x46ba03*/  // mode 32, then SendPacket (no fields)
```
Caller `sub_4A9C13` @0x4a9c13 iterates the CCashShop wishlist array at
`this[29]+1180 .. +1220` (10 slots), requires all 10 filled (`if (v43 < 10)` →
notice 3535, else confirm YesNo 3536), and on Yes sends mode 32. That array is
provably the wishlist: `CCashShop::OnSetWish` @0x469646 writes/reads the same
`(char *)this + 1180` region across 10 slots. Mode 32 is the **only** mode-only
send in the v72 enum, matching v83's only mode-only send (APPLY_WISHLIST=33).
**Best match — medium-high confidence.** (SET_WISHLIST stays mode 5.)

### ENABLE_EQUIP_SLOT (v83=9) — CONFIRMED ABSENT
No `Encode1(9)` on opcode 219 anywhere in the 29 enumerated sites. Equip-slot-ext
items (`item/1000==9110`) route through OnIncCharacterSlotCount → **mode 7**. No
dedicated mode 9.

### Mode 44 (0x2C) — UNMAPPED (no clean v83 key)
`sub_46BE7E` @0x46be96:
```
COutPacket::COutPacket((COutPacket *)v5, 219);   /*0x46be96*/
COutPacket::Encode1((COutPacket *)v5, 0x2Cu);    /*0x46bea4*/  // mode 44
COutPacket::Encode4((COutPacket *)v5, a2);       /*0x46beaf*/  // = *(CCashShop+1252)
COutPacket::Encode4((COutPacket *)v5, a3);       /*0x46beba*/  // = dialog field [33] (offset+132)
```
Caller `sub_72D47E` @0x72d47e fetches the player's OWN character name
(`CWvsContext::GetCharacterName`), opens a dialog via `sub_72D830` (dialog resource
StringPool 3999, vtable `off_9D6F84`), and on confirm (`sub_4D91E8 == 1`) sends
`Encode4(CCashShop+1252) + Encode4(dialog[33])`. Body = int(4)+int(4). Involves the
own character name but sends two ints (not strings), so it is NOT the mode-41 name
change. No confident v83 atlas-key match — **left unmapped**; best guess a
name/character-related confirm request (distinct dialog from the mode-41 flow).

### BUY_WORLD_TRANSFER — CONFIRMED via separate opcode 18 (0x12)
`sub_46BE19` @0x46be2e (called by `CCashShop::OnBuyTransferWorldItem` @0x468b93):
```
COutPacket::COutPacket((COutPacket *)v5, 18);    /*0x46be2e*/  // opcode 18 (0x12), NOT 219
COutPacket::Encode4((COutPacket *)v5, a2);       /*0x46be3d*/
COutPacket::Encode4((COutPacket *)v5, a3);       /*0x46be48*/
```
Not a mode of 0xDB — a distinct opcode (same as v79).

## Full 0xDB (219) request-mode map (v72), evidence-backed

| atlas key | v72 mode | fn @addr |
|---|---|---|
| BUY | 3 | OnBuy @0x467347 / SendBuyAvatarPacket @0x465936 |
| GIFT | 4 | SendGiftsPacket @0x4681df |
| SET_WISHLIST | 5 | OnSetWish @0x469646 / OnRemoveWish @0x469769 |
| INCREASE_INVENTORY | 6 | sub_465CDC @0x465fb6 |
| INCREASE_STORAGE | 7 | sub_466201 @0x46632f |
| BUY_FRIENDSHIP | 8 | OnBuyFriendship @0x466502 |
| MOVE_FROM_CASH_INVENTORY | 12 (0xC) | sub_46AEC0 @0x46b03e |
| MOVE_TO_CASH_INVENTORY | 13 (0xD) | sub_46B0AE @0x46b187 |
| REBATE_LOCKER_ITEM | 25 (0x19) | sub_465A87 @0x465c22 |
| BUY_COUPLE | 29 (0x1D) | OnBuyCouple @0x467834 |
| BUY_OTHER_PACKAGE | 30 (0x1E) | OnGiftPackage @0x468606 |
| BUY_PACKAGE | 31 (0x1F) | OnBuyPackage @0x467abb |
| APPLY_WISHLIST | 32 (0x20) | sub_46B9D7 @0x46b9f5 (via sub_4A9C13) — best match |
| BUY_NORMAL | 34 (0x22) | OnBuyNormal @0x469476 |
| GET_PURCHASE_RECORD | 39 (0x27) | RequestCashPurchaseRecord @0x4659c9 |
| BUY_NAME_CHANGE | 41 (0x29) | sub_46BCAC @0x46bccb |

Unmapped 0xDB modes (no v83 key): **28 (0x1C)** sub_4686F4 @0x468934
(ask_SPW + EncodeStr×2, package-buy/gift variant); **44 (0x2C)** sub_46BE7E @0x46be96
(Encode4+Encode4, char-name dialog confirm). `sub_46B4F5` @0x46b84c is a shared
locker/gift dispatcher selecting mode ∈ {4,28,30,34} at runtime by item id.

Absent from 0xDB: **INCREASE_CHARACTER_SLOT** (v83=8; folded into 6/7),
**ENABLE_EQUIP_SLOT** (v83=9; folded into 7), **BUY_WORLD_TRANSFER** (v83=49;
separate opcode 18).

Non-0xDB CashShop sends (unchanged from re-v72.md): opcode 21 (name-check,
sub_46BB81), 16 (sub_46BC47), 18 (world transfer), 220/0xDC (coupon,
OnStatusCoupon), 218/0xDA (query cash, TrySendQueryCashRequest), 37
(SendTransferFieldPacket).
