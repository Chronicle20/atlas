# FreeFormNotice (← `CWvsContext::OnEntrustedShopCheckResult#FreeFormNotice`)

- **IDA:** 0xb0ee59
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual analysis

**JMS-specific absent mode.** Mode 0x12 (`FreeFormNotice`) is absent from the JMS v185 `OnEntrustedShopCheckResult` switch (@ 0xb0ee59). The JMS switch has cases 7/8/9/10/11/13/14/15/16/17 — case 0x12 falls to `default: return`. The JMS client silently ignores this packet.

Atlas `FreeFormNotice` encoder writes mode 0x12 + flag + string. If a JMS tenant has a server that emits `FreeFormNotice`, the JMS client would just ignore the packet (no parse, no state change). This is a JMS-client-side limitation, not a server-side wire bug.

**JMS vs GMS: absent in JMS ⚠️ (out of scope).** Atlas encodes mode 0x12 which JMS does not handle. Since the JMS client ignores unknown modes (default return), no atlas code change is needed. The atlas `FreeFormNotice` struct is correct for GMS; JMS simply doesn't implement it. Noted as a JMS capability gap.

Ack: misc-audit Phase 3 JMS185 on 2026-06-03
