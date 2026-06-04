# Lock (← `CUserLocal::OnSetDirectionMode`)

- **IDA:** 0xa2cd83
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/lock.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bSet (direction/lock flag)` | ✅ |  |

