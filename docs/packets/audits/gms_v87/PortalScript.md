# PortalScript (← `CUserLocal::CheckPortal_Collision`)

- **IDA:** 0x9c8832
- **Atlas file:** `libs/atlas-packet/portal/serverbound/script.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (get_field()->m_bFieldKey, @0x9c88ce)` | ✅ |  |
| 1 | string | string `portalName (PORTAL::sName, @0x9c88f0)` | ✅ |  |
| 2 | int16 | int16 `x (GetPos().x, @0x9c8907)` | ✅ |  |
| 3 | int16 | int16 `y (GetPos().y, @0x9c891f)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
