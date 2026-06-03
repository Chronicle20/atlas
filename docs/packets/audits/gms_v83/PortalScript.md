# PortalScript (← `CUserLocal::CheckPortal_Collision`)

- **IDA:** 0x94dac6
- **Atlas file:** `../../libs/atlas-packet/portal/serverbound/script.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (get_field()+308)` | ✅ |  |
| 1 | string | string `portalName (PORTAL sName)` | ✅ |  |
| 2 | int16 | int16 `x (GetPos().x)` | ✅ |  |
| 3 | int16 | int16 `y (GetPos().y)` | ✅ |  |


Ack: world-audit Phase 3 v83 on 2026-05-28
