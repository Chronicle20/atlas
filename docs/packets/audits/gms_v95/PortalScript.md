# PortalScript (← `CUserLocal::CheckPortal_Collision`)

- **IDA:** 0x919a10
- **Atlas file:** `../../libs/atlas-packet/portal/serverbound/script.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (get_field()->m_bFieldKey)` | ✅ |  |
| 1 | string | string `portalName (PORTAL::sName)` | ✅ |  |
| 2 | int16 | int16 `x (GetPos().x)` | ✅ |  |
| 3 | int16 | int16 `y (GetPos().y)` | ✅ |  |

