# cash/serverbound/CashItemUseSongPlayer — GMS/JMS coverage sweep (task-252 Task 9)

Adds the packet coverage-matrix cell for the USE_CASH_ITEM cash-slot type-20
(song player / jukebox, item classification 510) sub-body,
`libs/atlas-packet/cash/serverbound/item_use_song_player.go` (Task 1 of this
branch). The codec was already IDA-derived on two builds (gms_v83 @0xa0c1a2,
gms_v95 @0x9ed51e) before this task; this pass re-confirms those two live and
sweeps the remaining eight in-scope versions read-only (no `rename`,
`set_comments`, or `idb_save`).

## Per-version evidence table

| Version | Function @ addr (session) | Case-20 dispatch | What it does | Decision |
|---|---|---|---|---|
| gms_v48 | `SendConsumeCashItemUseRequest` @0x70e495 (session 12a398ce) | Unique arm @0x70ff9c (`jumptable 0070E53C case 20`) | `CUser::GetDragon`@0x53174c, a CDragon-null-check guard (`sub_711EB7`), a field-existence guard trio (`sub_711EFE`/`sub_711EF2`), `eax=[arg_4]/1000` compared against `0x13B0` (5040) via `sub_4025A5`, then on the non-default path `CWvsContext::RunMapTransferItem`@0x614da6 — the same helper `cash/serverbound/ItemUseTeleportRock`'s arm delegates to (task-124). No `IWzResMan::GetObjectA`/`IWzSound`/`Encode4` anywhere in the arm. `get_consume_cash_item_type` on this build is `sub_47742E`@0x47742e, a WZ-driven `ZMap<long,ZRef<CItemInfo::LOTTERYITEM>,long>::GetAt` lookup — NOT the hardcoded `get_cashslot_item_type` switch v83+ use — so the case-number-to-feature mapping is not assumed stable across versions; confirmed different here from the arm body itself, not from the number. `SendCashSlotItemUseRequest`@0x719901 (task-139's structural analog) was also checked: dispatches only types 8/9/11 to pet-activate/effect-item-change/entrusted-shop-check; no song-player arm there either. | **n-a** — case dispatched, different feature (teleport rock) |
| gms_v61 | `SendConsumeCashItemUseRequest` @0x832a5d (session 921fdbb5) | **No arm** — enumerated every `jumptable 00832B01` case comment across the function's full body: cases 12,13,14,15,16,17,18,19,21,22,23,24,25,26,28,29,30,31,32,33,36,39,41,45,46,47,48,49,50,51,52 plus the compiler default-case list (27,34,35,37,38,40,42-44). 20 appears in neither. | n/a | **n-a** — case absent from the switch entirely |
| gms_v72 | `SendConsumeCashItemUseRequest` @0x904fe2 (session 99e435d8) | Unique arm @0x906966 | `GetObjectA`@0x906b4e (resolves the item's info/path sound node) → `Encode4`@0x906bba (single Encode4 in the arm) | **verified** |
| gms_v79 | `SendConsumeCashItemUseRequest` @0x95634a (session 5a1cd4f3) | Unique arm @0x957d8e | `GetObjectA`@0x957f76 → `Encode4`@0x957fe2 | **verified** |
| gms_v83 | `SendConsumeCashItemUseRequest` @0xa0a63f (session 754107bf) | Unique arm @0xa0c1a2 (already cited on the codec's doc comment; re-confirmed live this pass) | `GetObjectA`@0xa0c391 → sound-length getter `sub_644DCF` (vtable+56)@0xa0c3ed → `Encode4`@0xa0c3f6 | **verified** (re-confirmed) |
| gms_v84 | `SendConsumeCashItemUseRequest` @0xa54a2f (session 46c2a2eb) | Unique arm @0xa5656d | `GetObjectA`@0xa56755 → `Encode4`@0xa567c1 | **verified** |
| gms_v87 | `SendConsumeCashItemUseRequest` @0xa9fef9 (session c0829805) | Unique arm @0xaa1ae1 | `GetObjectA`@0xaa1ccb → `Encode4`@0xaa1d30 | **verified** |
| gms_v92 | `SendConsumeCashItemUseRequest` @0x9bfe10 (session 019cd393) | Unique arm @0x9c2029 — structural clone of the v95 arm (`TransientLayer_Exist`-style gate `sub_9A47F0` first) | Same `GetObjectA`/`IWzSound`/`Getlength` chain → `Encode4`@0x9c22c4 | **verified** |
| gms_v95 | `SendConsumeCashItemUseRequest` @0x9eb3e0 (session ecc757f4) | Unique arm @0x9ed51e (already cited on the codec's doc comment; re-confirmed live this pass) | `GetObjectA`@0x9ed75a → cast to `IWzSound`@0x9ed773 → `Getlength`@0x9ed7af → `Encode4`@0x9ed7b9 | **verified** (re-confirmed) |
| jms_v185 | `SendConsumeCashItemUseRequest` @0xaef2f5 (session a977912e) | Unique arm @0xaf0b7b | `GetObjectA`@0xaf0d80 → `Encode4`@0xaf0de5 | **verified** |

Every verified cell traces to a single `Encode4` of the WZ sound's
`IWzSound::Getlength` result, reached via
`IWzResMan::GetObjectA`→cast-to-`IWzSound`→`Getlength`, on every version from
gms_v72 through jms_v185. `cashsb.UpdateTimeFirst(tenant)` (GMS<=v84 trails,
GMS>=87/JMS leads) governs whether the shared dispatcher tail's trailing
`update_time` follows this arm's own payload — that gate was verified
pre-existing (task-1/task-7) and is unchanged by this task.

## The v48 finding, in more detail

The critical distinguishing fact: v83+'s cash-slot type mapping
(`get_consume_cash_item_type` → `get_cashslot_item_type`) is a **hardcoded**
switch on item classification — `case 510: result = 20;` on gms_v95.0
@0x488c70. v48's equivalent, `sub_47742E`@0x47742e, is instead a lookup into
`ZMap<long,ZRef<CItemInfo::LOTTERYITEM>,long>` — a WZ-populated per-item cache.
That means the same numeric case value is **not guaranteed to carry the same
feature identity** between the two schemes. This sweep did not assume it did:
it decompiled v48's case-20 arm directly and found teleport-rock logic
(`RunMapTransferItem`), not the sound-length Encode4 shape every other version
carries. Whether song player exists under some *other* case number on v48 (or
doesn't exist as a distinct cash-slot feature at all — plausible, since jukebox
items may postdate this early build) was not resolved; it doesn't need to be
for this cell's disposition, since the question this task answers is "does
CASE 20 encode the song length on v48", and the answer is demonstrably no.

## Mechanism note

Both n-a dispositions were recorded in `docs/packets/audits/<version>/_unimplemented.json`
(`packet: cash/serverbound/CashItemUseSongPlayer`), the mechanism that actually
grades `RowSubStruct` cells (`tools/packet-audit/internal/matrix/build.go`
`gradeSubStructCell` reads `in.Unimplemented[vk][pkt]`) — not
`docs/packets/feature-na-evidence.yaml`, which only participates in the RowOp
family-consistency gate.

## Prerequisite wiring

`tools/packet-audit/cmd/run.go`'s `candidatesFromFName` switch for
`CWvsContext::SendConsumeCashItemUseRequest` gained an `ItemUseSongPlayer`
(pkg `cash`, dir `serverbound`) entry, modelled on the existing
`ItemUsePetSkill`/`ItemUsePetNameTag` entries in the same list. Without it no
audit report could ever be generated for this struct — the evidence would
have stayed permanently dangling. `qualifiedWriterName("cash",
"ItemUseSongPlayer")` = `CashItemUseSongPlayer`, which is why the matrix row,
the `packet-audit:verify` markers, and the evidence file paths all read
`cash/serverbound/CashItemUseSongPlayer` (the struct's own Go type name has no
`Cash` prefix; the prefix is a matrix/report naming convention, not part of
the codec).

## Verification

```
go run ./tools/packet-audit matrix              # regenerates STATUS.md/status.json
go run ./tools/packet-audit matrix --check       # exit 0
go run ./tools/packet-audit fname-doc --check    # exit 0
go run ./tools/packet-audit operations --check   # exit 0
go run ./tools/packet-audit dispatcher-lint      # clean
```

`go build ./... && go test ./...` clean in `libs/atlas-packet` and
`tools/packet-audit`.
