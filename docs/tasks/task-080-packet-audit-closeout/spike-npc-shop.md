# Spike: NPC Shop packet verdicts (task-080 B3.3 + B3.4)

Two verification spikes for the npc-shop packet pair. IDB loaded: **JMS v185.1**
(`MapleStory_dump_SCY.exe`). Plan addresses in the PRD are GMS; JMS addresses
resolved by name below. GMS v83 corroboration from `docs/packets/ida-exports/gms_v83.json`
(the source for `template_gms_83_1.json`).

**Both verdicts: VERIFIED — NO FIX.** No emitter / op-byte / body divergence found
on the wired (GMS) config. Notes on the unwired JMS template are recorded but are
out of scope for these two spikes.

---

## B3.3 — NPC shop-operation CLIENTBOUND mode enum

### IDA (JMS185)
`CShopDlg::OnPacket` @ **0x7cb04e** (`?OnPacket@CShopDlg@@SAXJAAVCInPacket@@@Z`).
The shop-operation packet (`nType == 0x14B`) reads the mode with `CInPacket::Decode1`
@0x7cb0e5 and switches. Full case map:

| JMS mode | Client behaviour | Body after mode byte |
|---|---|---|
| 0 | refresh inventory / scroll pos (success) | none |
| 1, 5, 9 | StringPool Notice 893 | none |
| 2, 0xA (10) | StringPool Notice 257 | none |
| 3 | StringPool Notice 894 | none |
| **4, 8, 0x13 (19)** | **`return` — no-op** | none |
| 0xD (13) | StringPool Notice 259 | none |
| 0xE (14) | `Decode4` levelLimit → formatted Notice 0x375 | int |
| 0xF (15) | `Decode4` levelLimit → formatted Notice 0x374 | int |
| 0x10 (16) | StringPool Notice 5177 | none |
| 0x11 (17) | StringPool Notice 5179 | none (JMS: mode-only) |
| 0x14 (20) | StringPool Notice 895 | none |
| default (incl. 6,7,0xB,0xC) | StringPool Notice 900 (generic) | none |

### Atlas emitters (`libs/atlas-packet/npc/clientbound/shop_operation*.go`)
Resolver `operations` (GMS template @ line 2272, `template_gms_83_1.json`) maps the
codes Atlas can emit: OK=0, OUT_OF_STOCK=1, NOT_ENOUGH_MONEY=2, INVENTORY_FULL=3,
OUT_OF_STOCK_2=5, OUT_OF_STOCK_3=9, NOT_ENOUGH_MONEY_2=10, NEED_MORE_ITEMS=13,
OVER_LEVEL_REQUIREMENT=14, UNDER_LEVEL_REQUIREMENT=15, TRADE_LIMIT=16,
GENERIC_ERROR=17, GENERIC_ERROR_WITH_REASON=17.

Body encoders:
- `ShopOperationSimple` → mode only (modes 0,1,2,3,5,9,10,13,16,17).
- `ShopOperationLevelRequirement` → mode + int levelLimit (modes 14,15).
- `ShopOperationGenericError` → mode + bool hasReason + optional ascii reason (mode 17).

### Comparison / verdict
- **Every mode Atlas emits is a handled, semantically-matching case in the JMS185
  switch.** Modes 0,1,2,3,5,9,10,13,16 take the mode-only `Notice` arms (match
  `ShopOperationSimple`). Modes 14/15 take the `Decode4` levelLimit arms (match
  `ShopOperationLevelRequirement`). Mode 17 is the generic-error arm.
- **Modes Atlas does NOT emit are exactly the client no-op / generic-default arms.**
  JMS cases 4, 8, 0x13 are explicit `return` (no-op); cases 6,7,0xB,0xC fall to the
  generic `default` (Notice 900). The plan's specific "4/6/7/0xB/0xC carry no
  emitter" claim holds — none of those has an Atlas emitter (the exact no-op set is
  4/8/19 in JMS, but the conclusion — Atlas emits a correct subset and skips the
  client-unhandled modes — is confirmed).
- **GENERIC_ERROR body shape:** GMS v83 export (`OnPacket#GenericError`) reads
  `Decode1 mode + Decode1 hasReason + optional DecodeStr reason` — exactly the
  Atlas `ShopOperationGenericError` shape. (JMS185 case 0x11 is mode-only Notice
  5179 with no hasReason, but JMS shop is not wired — see note.)

**Verdict B3.3: VERIFIED — no missing emitter, no wrong body for any emitted mode.
No fix.**

---

## B3.4 — NPC shop SERVERBOUND op-byte values (esp. LEAVE)

### IDA (JMS185) — op-byte each send writes (`COutPacket` opcode 0x35 = NPC_SHOP)
| Function | Addr | `Encode1` op-byte | Trailing body |
|---|---|---|---|
| `CShopDlg::SendBuyRequest` | 0x7ca2c9 | **0** (@0x7caaf3) | Encode2 slot, Encode4 itemId, Encode2 quantity |
| `CShopDlg::SendSellRequest` | 0x7cacab | **1** (@0x7cae68) | Encode2 slot, Encode4 itemId, Encode2 quantity |
| `CShopDlg::SendRechargeRequest` | 0x7caecf | **2** (@0x7caff8) | Encode2 slot |
| `sub_7CAB93` (bulk buy w/ amount; button 1001) | 0x7cab93 | **3** (@0x7cac6a) | **Encode4 amount** (NOT a bare leave) |
| `CShopDlg::SetRet` (dialog close = LEAVE) | 0x7c6e8d | **4** (@0x7c6eb0) | **none** |

Dispatch confirmed via `CShopDlg::OnButtonClicked` @0x7c7dd7
(1000→Buy, 1001→sub_7CAB93, 1002→Sell, recharge slot buttons→Recharge) and the
ServerBound CSV `NPC_SHOP` row which lists `CShopDlg::SetRet` alongside the
Buy/Sell/Recharge senders under the single 0x35 opcode.

### Atlas / template
- `services/atlas-channel/.../socket/handler/npc_shop.go` resolves op-byte → action
  from the channel `operations` config (BUY/SELL/RECHARGE/LEAVE), then decodes the
  matching body. LEAVE → `ExitShop` with **no body read** (correct).
- `template_gms_83_1.json` @ line 318: `BUY:0, SELL:1, RECHARGE:2, LEAVE:3`.
- Serverbound bodies: `shop_buy.go` (slot/itemId/quantity, + GMS-only trailing
  discountPrice int region-guarded), `shop_sell.go` (slot/itemId/quantity),
  `shop_recharge.go` (slot). `shop.go` reads the bare op-byte.

### Comparison / verdict
- **BUY=0 / SELL=1 / RECHARGE=2: MATCH** the JMS185 client `Encode1` values, and the
  GMS v83 export (`gms_v83.json`) shows the same buy/sell/recharge send shapes. No
  divergence. Verified.
- **`ShopBuy` discountPrice region-guard is correct.** JMS185 `SendBuyRequest` ends
  after `Encode2(quantity)` @0x7cab2b — **no trailing discountPrice**. GMS v83
  `SendBuyRequest` DOES append `Encode4 discountPrice`. The `t.Region()=="GMS"` guard
  in `shop_buy.go` matches both. (The code comment cites the function entry
  @0x7ca2c9; the actual quantity-terminus is @0x7cab2b — annotation is accurate.)
- **LEAVE op-byte — the adversarial point:** In **JMS185** the dialog-close LEAVE is
  `SetRet` → `Encode1(4)`, **no body** — op-byte **4**, not 3. JMS op-byte **3** is a
  *separate* "bulk buy with amount" operation (`sub_7CAB93`, `Encode1(3)+Encode4`),
  so JMS LEAVE is shifted to 4. **However, the GMS template's `LEAVE:3` is correct for
  GMS v83**, which has the classic ShopAction layout (0 buy / 1 sell / 2 recharge /
  3 leave) with **no** intermediate bulk-buy op-3: the v83 export
  (`gms_v83.json`) curates only Simple/LevelRequirement/GenericError OnPacket arms and
  buy/sell/recharge senders — no op-3 bulk-buy send exists in v83 to displace LEAVE.
  The ServerBound CSV confirms `CShopDlg::SetRet` is the NPC_SHOP leave path in GMS.
  LEAVE carries **no trailing body** in both regions — confirmed.

**Verdict B3.4: VERIFIED — BUY/SELL/RECHARGE/LEAVE op-bytes in the wired GMS template
match the GMS client; LEAVE carries no body. No fix.**

---

## Notes (out of scope for these spikes)

- **JMS shop is entirely unwired.** `template_jms_185_1.json` is a partial template
  (25 handlers / 44 writers vs GMS 93 / 112) and defines **neither** `NPCShopHandle`
  **nor** `NPCShopOperation`. So for JMS185 there is currently no shop op-byte map and
  no clientbound shop-operation writer wired. If/when JMS shop is wired, the
  region-correct values are: serverbound **BUY=0 / SELL=1 / RECHARGE=2 / LEAVE=4**
  (op-3 reserved for the JMS bulk-buy-with-amount op), `SendBuyRequest` body **without**
  the trailing discountPrice int (already region-guarded), and clientbound
  GENERIC_ERROR (mode 0x11) is **mode-only** in JMS (no hasReason/reason). These are a
  larger wiring task, not a divergence in the audited GMS path.
