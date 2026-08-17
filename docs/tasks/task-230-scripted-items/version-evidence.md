# task-230 — client evidence for `SCRIPTED_ITEM`

Derived 2026-08-14 via ida-pro-mcp against the open IDB session set. Recorded here so design does
not re-derive it, and so the matrix corrections in the PRD (§4.1) are traceable to specific
addresses.

## Method

For each version, `func_query` with `name_regex: "ScriptRunItem|ScriptItem|RunItem"`, then
`decompile` on any hit. The opcode is the second argument to the `COutPacket::COutPacket`
constructor.

**Control:** `gms_v83` was decompiled first. It yielded `COutPacket::COutPacket(&v6, 0x4E)`, which
matches the `0x04E` already recorded in `docs/packets/registry/gms_v83.yaml`. The method is therefore
validated before being applied to versions with no registry entry.

## Results

| Binary | Session | `SendScriptRunItemRequest` | Opcode |
|---|---|---|---|
| `MapleStory_dump.exe` (v83) | `41f13e0d` | `0xa09b26` | `0x4E` (78) — matches registry ✅ |
| `GMS_v79_1_DEVM.exe` | `1438cecd` | `0x955840` | **`0x4C` (76)** — new |
| `GMS_v72.1_U_DEVM.exe` | `c8acae95` | `0x9044d8` | **`0x4D` (77)** — new |
| `GMS_v61.1_U_DEVM.exe` | `415bf585` | **absent** | — |
| `GMS_v48_1_DEVM.exe` | `93cc947e` | **absent** | — |
| `gms_v12` | no session open | unverified | — |

Note the non-monotonic opcode ordering: v72 = `0x4D`, v79 = `0x4C`, v83 = `0x4E`. This is not a
transcription error — both were read from their own decompilation. Databases were cross-checked
against the `func_query` addresses before decompiling.

### Absence evidence for v48 / v61

Neither binary is merely missing a *name*. Both expose a densely symbolised `Send*ItemUseRequest`
family with `SendScriptRunItemRequest` absent from it:

- **v61** (`415bf585`) — `SendUpgradeItemUseRequest` `0x8317a4`, `SendStatChangeItemUseRequest`
  `0x831880`, `SendMobSummonItemUseRequest` `0x831c83`, `SendPetFoodItemUseRequest` `0x831de9`,
  `SendTamingMobFoodItemUseRequest` `0x831f44`, `SendBridleItemUseRequest` `0x832005`,
  `SendSkillLearnItemUseRequest` `0x8325d2`, `SendShopScannerItemUseRequest` `0x832680`,
  `SendMapTransferItemUseRequest` `0x8327db`, `SendSelectNpcItemUseRequest` `0x83778d`,
  `SendLotteryItemUseRequest` `0x839b0f`.
- **v48** (`93cc947e`) — `SendUpgradeItemUseRequest` `0x70da60`, `SendStatChangeItemUseRequest`
  `0x70db3c`, `SendMobSummonItemUseRequest` `0x70ddaa`, `SendPetFoodItemUseRequest` `0x70df2f`,
  `SendTamingMobFoodItemUseRequest` `0x70e00b`, `SendBridleItemUseRequest` `0x70e0c5`,
  `SendSkillLearnItemUseRequest` `0x70e3e7`, `SendEtcCashItemUseRequest` `0x7198bb`.

Absence from a fully-named export set is meaningful. This is distinct from the usual
"unnamed ≠ absent" caution, which applies to sparsely-named IDBs.

Note v61 **has** `SendSelectNpcItemUseRequest` while lacking `SendScriptRunItemRequest` — the two
ops have different version spans. See PRD §9 O-1.

## Wire body

Identical across v72, v79, v83:

```
Encode4(get_update_time())   // uint32
Encode2(slot)                // int16  — a2, source inventory slot
Encode4(itemId)              // int32  — a3, item template id
```

## Client-side gates

- **All three versions:** hard guard `itemId / 10000 == 243`. The client cannot send this op for any
  item outside `2430000`–`2439999`. This structurally confirms the `0243.img` family without needing
  the WZ corpus.
- **v83 only:** `CWvsContext::CanSendExclRequest(this, 500, 0)` and
  `CWvsContext::IsAbleToConsume(this, itemId, 1)` (`0xa10579`). v72/v79 call the equivalent
  excl-request helper (`sub_4DBE16(500, 0)` on v72) but have **no** `IsAbleToConsume` check.
- **All three:** `SetExclRequestSent` / equivalent is set after send — the standard excl-request
  unlock contract applies.

## Not yet decompiled

v84, v87, v92, v95, jms_v185 — opcodes taken from their existing registry entries
(`0x04E`, `0x051`, `0x055`, `0x054`, `0x046`). Bodies assumed identical. PRD §9 O-5 asks design to
spot-check at least v95 and jms_v185.
