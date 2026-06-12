# FieldTransport (← `CField_ContiMove::OnContiState`)

- **IDA:** 
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/transport.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

