# SetGender (← `CLogin::SendSetGenderPacket`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/account/serverbound/set_gender.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual analysis

**Function absent from JMS v185.** `CLogin::SendSetGenderPacket` is not present in the JMS v185 IDB. JMS login flow differs; no named gender-setting function was found. The IDA source address is empty, so the audit tool produced an artifact ❌.

No atlas code change needed. SetGender may be a GMS-specific or early-version flow not present in JMS v185.

**JMS vs GMS: absent in JMS (function not found).** Out of scope for JMS gate verification.

Ack: misc-audit Phase 3 JMS185 on 2026-06-03
