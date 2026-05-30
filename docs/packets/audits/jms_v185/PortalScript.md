# PortalScript (← `CUserLocal::CheckPortal_Collision`)

- **IDA:** 0xa0dde7
- **Atlas file:** `../../libs/atlas-packet/portal/serverbound/script.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (get_field()->m_bFieldKey @line75)` | ✅ |  |
| 1 | string | string `portalName (@line79)` | ✅ |  |
| 2 | int16 | int16 `x (current user X @line81)` | ✅ |  |
| 3 | int16 | int16 `y (current user Y @line83)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 field+portal on 2026-05-28
