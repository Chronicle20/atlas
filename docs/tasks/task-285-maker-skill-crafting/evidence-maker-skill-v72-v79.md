# Evidence — `MAKER_SKILL` is NOT `n-a` on gms_v72 / gms_v79

Date: 2026-08-28
Method: ida-pro-mcp against the checked-in IDBs.

## Finding

`docs/packets/audits/status.json` row 324 records `MAKER_SKILL` (serverbound) as
`{"state": "n-a", "opcode": -1}` for `gms_v72` and `gms_v79`. **That is incorrect.** Both
clients ship the complete Item Maker UI and both send the request.

| Version | IDB | `CUIItemMaker::RequestItemMake` | Opcode |
|---|---|---|---|
| `gms_v72` | `GMS_v72.1_U_DEVM.exe.i64` (session `99e435d8`) | `?RequestItemMake@CUIItemMaker@@IAEHXZ` @ `0x760cc3` | `COutPacket::COutPacket(v16, 0x70)` → **112 / 0x70** |
| `gms_v79` | `GMS_v79_1_DEVM.exe.i64` (session `5a1cd4f3`) | `?RequestItemMake@CUIItemMaker@@IAEHXZ` @ `0x795dc3` | `COutPacket::COutPacket(v16, 111)` → **111 / 0x6F** |

Both send via `CClientSocket::SendPacket(g_pClientSocketInstance, v16)`.

## Supporting symbols (v72)

The whole feature is present, not a stub:

- `?OnCreate@CUIItemMaker@@UAEXPAX@Z` @ `0x75c079`
- `?IsAbleToMake@CUIItemMaker@@IAEHXZ` @ `0x75c914`
- `?DoesSatisfyPreCondition@CUIItemMaker@@IAEHJ@Z` @ `0x75c7e1`
- `?DrawRecipe@CUIItemMaker@@IAEX...@Z` @ `0x75eebd`
- `?DrawGem@CUIItemMaker@@IAEX...@Z` @ `0x75f426`
- `?DrawCatalyst@CUIItemMaker@@IAEX...@Z` @ `0x75f60f`
- `?ConfirmMake@CUIItemMaker@@IAEHXZ` @ `0x7609f2`
- `?StartItemMake@CUIItemMaker@@IAEHXZ` @ `0x760b58`
- `?IsEnoughMeso@CUIItemMaker@@IAEHXZ` @ `0x760be0`
- `?OnItemMakeResult@CUIItemMaker@@QAEXJJJJ@Z` @ `0x760e76`
- `?Load_ItemMakeInfo@CItemMakerInfo@@QAEHXZ` @ `0x5a2802`
- `?Load_GemEffect@CItemMakerInfo@@QAEHXZ` @ `0x5a2cf5`
- `?Load_MonsterCrystalLevel@CItemMakerInfo@@QAEHXZ` @ `0x5a3033`
- `?Load_MonsterTrophy@CItemMakerInfo@@QAEHXZ` @ `0x5a3430`

v79 mirrors these (`?Load_ItemMakeInfo@CItemMakerInfo@@QAEHXZ` @ `0x5bca77`,
`?Load_MonsterCrystalLevel@CItemMakerInfo@@QAEHXZ` @ `0x5bd2a8`, …).

The string `"ItemMake.img"` is present in both v72 (`0xa5bf5c`) and v83 (`0xaf67c8`).

## Structural corroboration

The derived opcodes land exactly in the registry's existing gaps, and the serverbound
neighbourhood aligns one-for-one with v83:

| | v72 | v79 | v83 |
|---|---|---|---|
| `LOTTERY_ITEM_USE_REQUEST` | 111 | 110 | 112 |
| `MAKER_SKILL` | **112 (was a gap)** | **111 (was a gap)** | 113 |
| `SUE_CHARACTER` | 113 | 112 | 114 |

## Root cause of the bad cell

`MAKER_SKILL` carries `provenance: csv-import` in `gms_v83.yaml:2543`, `gms_v84.yaml:2545`,
and `gms_v87.yaml:2658` — it was seeded from `docs/packets/MapleStory Ops - ServerBound.csv`,
which has no v72/v79 column. The op therefore never entered `gms_v72.yaml` / `gms_v79.yaml` at
all, and the matrix rendered "absent from the registry" as `n-a` rather than as undiscovered.
This is a discovery gap, not a client capability gap.

## Request layout (derived, v72 and v79 identical)

Field order read straight off both decompilations. Mode is the leading `int`.

```
Encode4  nMode

nMode 1..2  (create item)
  Encode4  nMode                 // echoed
  Encode4  nTargetItemID
  Encode1  bUseCatalyst
  Encode4  nGemCount             // count of non-null reagent slots
  Encode4  nGemItemID            // repeated, one per non-null slot

nMode 3     (monster crystal from leftover)
  Encode4  3
  Encode4  nLeftoverItemID

nMode 4     (disassemble)
  Encode4  4
  Encode4  nItemID
  Encode4  nInventoryType
  Encode4  nSlotPos
```

This matches the reference server's read order in `Cosmic MakerProcessor.makerAction`,
including the otherwise-unexplained extra `int` before the slot position in mode 4
(commented there as `// 1... probably inventory type`).

## gms_v48 / gms_v61 — `n-a` appears correct

The only `CUIItemMaker`-named function in either IDB is
`?OnButtonClicked@CUIItemMaker@@UAEXI@Z` (v48 `0x608a0b`, v61 `0x6baddf`). Decompiling the v61
one shows it is **mislabelled**: its body dispatches to `CUIGuildBBS::OnDelete`,
`CUIGuildBBS::OnRegister`, `CUIGuildBBS::OnComment`, and `CUIGuildBBS::OnCommentDelete`. It is
the Guild BBS handler wearing the wrong name.

Neither IDB contains `CItemMakerInfo` or `RequestItemMake`. `n-a` on v48/v61 stands. The
mislabelled symbol is a separate IDB-hygiene issue, out of scope here.

## Required correction

Before any codec work:

1. Add `MAKER_SKILL` serverbound to `docs/packets/registry/gms_v72.yaml` (opcode 112,
   `fname: CUIItemMaker::RequestItemMake`, `provenance: ida-discovered`, `ida.address: 0x760cc3`).
2. Add the same to `docs/packets/registry/gms_v79.yaml` (opcode 111, `ida.address: 0x795dc3`).
3. Regenerate `status.json` / `STATUS.md` and confirm both cells flip from `n-a` to
   `incomplete`, and that `gms_v48` / `gms_v61` remain `n-a`.
4. Re-run `packet-audit matrix`, `fname-doc`, and `operations --check`.
