# RpsOperation (← `CRPSGameDlg::Update`)

- **IDA:** 0x760e64
- **Atlas file:** `libs/atlas-packet/rps/serverbound/operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op byte (0=OnBtStart/2=Update-timeout/3=OnBtContinue/4=OnBtExit/5=OnBtRetry; bodyless arm, no further fields — docs/tasks/task-132-rps-npc-game/ida-rps-serverbound.md §0/§1-§5)` | ✅ |  |

