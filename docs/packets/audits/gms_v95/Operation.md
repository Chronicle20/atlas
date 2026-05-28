# Operation (← `CTrunkDlg::OnPacket#Operation`)

- **IDA:** 0x76a990
- **Atlas file:** `../../libs/atlas-packet/storage/serverbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op-byte (4=get,5=put,7=meso; supplied by caller)` | ✅ |  |


> ack: dispatcher — op-byte supplied by caller; field payloads live in the
> sibling sub-bodies (OperationRetrieveAsset/StoreAsset/Meso). See
> `docs/packets/ida-exports/_pending.md` → "OP-FAMILY-storage".
