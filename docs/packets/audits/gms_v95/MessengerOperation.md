# MessengerOperation (← `CUIMessenger::OnDestroy`)

- **IDA:** 0x7f03f0
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op (=2, LEAVE) — only field; atlas Operation struct reads this as mode byte` | ✅ |  |

