# task-178 — CashShop op semantics (for deep per-version resolution)

The atlas `CashShopOperationHandle` decodes these fields per op AFTER the mode
byte (`Encode1(mode)`). Use the field shape + calling UI + StringPool refs to
match each client send to the correct atlas key and read its mode.

| atlas key | v83 mode | serverbound body after mode byte | disambiguator |
|---|---|---|---|
| BUY | 3 | flag(1) + sn(4) [avatar buy] | `OnBuy`/`SendBuyAvatarPacket` |
| GIFT | 4 | spw(4)+sn(4)+recipient str+msg str | `OnGift`/`SendGiftsPacket` |
| SET_WISHLIST | 5 | 10× sn(4) | `OnSetWish` |
| INCREASE_INVENTORY | 6 | flag(1)+0(1)+tab(1, 1..4) | inventory-tab expand |
| INCREASE_STORAGE | 7 | flag(1)+0(1) | storage/trunk expand |
| INCREASE_CHARACTER_SLOT | 8 | flag(1) [buy char slot] | `OnIncCharacterSlotCount` |
| ENABLE_EQUIP_SLOT | 9 | pointType/flag + sn(4) [item 9110xxx] | `OnEnableEquipSlotExt`; item/1000==9110 |
| MOVE_FROM_CASH_INVENTORY | 13 | sn(4)+inventoryType+**slot** | locker→inv; HAS slot field |
| MOVE_TO_CASH_INVENTORY | 14 | sn(4)+inventoryType | inv→locker; NO slot field |
| BUY_NORMAL | 20 | sn(4) [+ chatlog strings] | `OnBuyNormal` |
| REBATE_LOCKER_ITEM | 26 | unk + birthday | rebate/refund locker |
| BUY_COUPLE | 29 | sn(4)+recipient+msg+option+birthday | `OnBuyCouple` |
| BUY_PACKAGE | 30 | pointType + sn(4) + option | `OnBuyPackage` |
| BUY_OTHER_PACKAGE | 31 | (gift package) | `OnGiftPackage` |
| APPLY_WISHLIST | 33 | **mode only, no body** | mode-only send |
| BUY_FRIENDSHIP | 35 | sn(4)+recipient+msg+option+birthday | `OnBuyFriendship` (like couple) |
| GET_PURCHASE_RECORD | 40 | sn(4) | `RequestCashPurchaseRecord` |
| BUY_NAME_CHANGE | 46 | sn(4)+oldName str+newName str | `sub` with sn+2 strings |
| BUY_WORLD_TRANSFER | 49 | targetWorld + sn(4) — OFTEN A SEPARATE OPCODE in legacy | `OnBuyTransferWorldItem` |

Rules: quote the real `Encode1(mode)` + `COutPacket(op)` line per key. If a key's
send does not exist under the cash-shop opcode, mark ABSENT (with the opcode it
actually uses, if any). If a mode's key identity is uncertain, report the mode +
its body shape and your best atlas-key match with confidence. Never assume v83.
