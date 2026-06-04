# RegisterPin (← `CLogin::OnCheckPinCodeResult#RegisterPin`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/account/serverbound/register_pin.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual analysis

**Function absent from JMS v185.** `CLogin::OnCheckPinCodeResult` is not present in the JMS v185 IDB. The JMS template (`template_jms_185_1.json`) has `usesPin: false`, confirming JMS has no PIN flow. The IDA source address is empty, so the audit tool produced an artifact ❌.

No atlas code change needed. RegisterPin is a GMS-only feature.

**JMS vs GMS: absent in JMS (function not found).** Out of scope for JMS gate verification.

Ack: misc-audit Phase 3 JMS185 on 2026-06-03
