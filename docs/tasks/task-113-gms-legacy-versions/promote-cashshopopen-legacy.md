# Promote `cash/clientbound/CashShopOpen` to ✅ for gms_v48 + gms_v61 (legacy body)

The 7 modern versions (v72/v79/v83/v84/v87/v95/jms) were promoted in the prior pass
(`promote-cashshopopen.md`). v48 and v61 were FLAGGED there because their clients read
a **legacy** Cash Shop body and the modern writer over-emitted 6 bytes. This change
version-gates those 6 bytes, registers the op, adds golden fixtures + markers, pins
fresh evidence, and honestly promotes both cells.

## The legacy body (IDA body-verified, not assumed)

Both v48 and v61 read a body that OMITS exactly two modern-GMS fields:
**`DecodeZeroGoods` (2 bytes)** and the **trailing `Decode4` nHighest (4 bytes)**.
Everything else — migrate-in CharacterData, `Decode1` auth, `DecodeStr` account,
`SetSaleInfo` (Decode4 nNotSaleCount + Decode2 special + Decode1 discounts),
`DecodeBuffer(1080)` Best block, `DecodeStock`, `DecodeLimitGoods`, `Decode1`
bEventOn — is byte-identical to the modern GMS body.

### gms_v48 (IDA port 13337, `GMS_v48_1_DEVM.exe`)

- Dispatch: `sub_5C45ED` routes **opcode 74** → `sub_5C4D9C` (`0x5c4d9c`,
  CStage::OnSetCashShop equivalent). Opcode 73 → SET_ITC. IDA-confirmed at the
  `sub_5C45ED` switch (`if (a1 == 74) return sub_5C4D9C(a2);`).
- Handler `sub_5C4D9C 0x5c4d9c`: reads CharacterData (`sub_49D320` @0x5c4dc2),
  then constructs `CCashShop` via `sub_447122(v4, a1)` @0x5c4e58.
- Ctor `sub_447122 0x447122`: calls body reader `sub_44E1E5(a2)` @**0x447239**,
  then `this[316] = Decode1(a2)` @**0x447249** — a SINGLE trailing `Decode1`
  (bEventOn). NO `Decode4` nHighest.
- Body reader `sub_44E1E5 0x44e1e5` (LoadData): `Decode1` auth @0x44e20c →
  `DecodeStr` account @0x44e226 → SetSaleInfo `sub_71E3E4` @0x44e473 (verified:
  `Decode4` nNotSaleCount @0x71e48b + `Decode2` special @0x71e4ba + `Decode1`
  discounts @0x71e670) → `DecodeBuffer(this+20, 0x438=1080)` @**0x44e993** →
  `sub_44F10B` DecodeStock @**0x44e99d** (Decode2 count + DecodeBuffer(8·n)) →
  `sub_44F152` DecodeLimitGoods @**0x44e9a7** (Decode2 count + DecodeBuffer(104·n)).
  There is **NO third post-buffer decoder** — no `DecodeZeroGoods`.

### gms_v61 (IDA port 13338, `GMS_v61.1_U_DEVM.exe`, symbolicated)

- **opcode 94** (`0x5E`), `CStage::OnSetCashShop 0x65a973` (CStage::OnPacket case
  '^'(94)). Reads `CharacterData::Decode` then `CCashShop::CCashShop 0x453549`.
- Ctor `CCashShop::CCashShop 0x453549`: `CCashShop::LoadData(pExceptionObject)`
  @0x4536c1, then `this[317] = Decode1` @**0x4536d4** — SINGLE trailing `Decode1`
  (bEventOn). NO `Decode4` nHighest.
- `CCashShop::LoadData 0x45b539`: `Decode1` auth → `DecodeStr` account →
  `CWvsContext::SetSaleInfo 0x8474e6` → `DecodeBuffer(this+20, 1080)` →
  `sub_45C497` DecodeStock → `sub_45C4DE` DecodeLimitGoods. **No `DecodeZeroGoods`.**

### Boundary proof

The modern GMS body (v72+) reads all three post-buffer decoders + the trailing
nHighest (v72 ctor reads `Decode1`+`Decode4`; LoadData has Stock/Limit/**Zero** —
prior-pass verified, and the modern round-trip fixtures exercise it). The legacy body
(v48, v61) reads Stock/Limit only + a single `Decode1`. The break is between v61 and
v72, so the gate is **`GMS && MajorAtLeast(72)`**.

## The gate (`libs/atlas-packet/cash/clientbound/shop_open.go`)

Two spots, both Encode and Decode, changed from `t.MajorVersion() > 12` /
unconditional-GMS to `t.MajorAtLeast(72)`:

- `DecodeZeroGoods` (`w.WriteShort(0)`): `GMS && MajorVersion()>12` →
  `GMS && MajorAtLeast(72)`.
- nHighest (`w.WriteInt(200)`, inside the bEventOn block): `GMS` →
  `GMS && MajorAtLeast(72)`.

`bEventOn` stays gated on `(GMS && >12) || JMS` — v48/v61 (major>12) DO read it
(the ctor's single `Decode1`).

### 7 modern versions unchanged (proof)

- v72/v79/v83/v84/v87/v95: `MajorAtLeast(72)` is `true` exactly where
  `MajorVersion()>12` was `true` → both gated writes still emitted. Byte-identical.
- jms_v185: Region `JMS`, not `GMS` → neither field was emitted before or after.
  Byte-identical.
- Verified by re-running `TestCashShopOpenRoundTrip` (v83/v84/v87/v95/jms +
  v28/v86 variants) and `TestCashShopOpenLegacyGolden` (v79/v72 full-buffer equality
  with v83) — all green after the gate.

## Op registration

- **gms_v48** — new registry entry `SET_CASH_SHOP` opcode 74 (`0x4A`,
  fname `CStage::OnSetCashShop`, address 6049180 = 0x5c4d9c) with
  `packet: cash/clientbound/CashShopOpen`; template writer entry
  `{"opCode":"0x4A","writer":"CashShopOpen"}` added to `template_gms_48_1.json`.
- **gms_v61** — existing `SET_CASH_SHOP` opcode 94 (`0x5E`) already had the writer in
  `template_gms_61_1.json`; added `packet: cash/clientbound/CashShopOpen` to the
  registry entry.

## Fixtures + markers + evidence

- `TestCashShopOpenLegacyBodyV48V61` (`shop_open_test.go`) asserts, for GMS 48 and 61:
  (a) the exact 17-byte legacy tail (last Best triple, Stock 0, Limit 0, bEventOn 0 —
  no ZeroGoods, no nHighest), (b) length == modern v83 length − 6, (c) every byte
  through DecodeLimitGoods byte-identical to the modern v83 encode. Markers:
  `// packet-audit:verify packet=cash/clientbound/CashShopOpen version=gms_v48 ida=0x5c4d9c`
  and `version=gms_v61 ida=0x65a973`.
- Evidence pinned FRESH (`evidence pin --ida CStage::OnSetCashShop --category
  TIER1-FIXTURE`):
  - `docs/packets/evidence/gms_v48/cash.clientbound.CashShopOpen.yaml` —
    address 0x5c4d9c, decompile_sha256 `1081e1a2…b14b3`, verifies the new test.
  - `docs/packets/evidence/gms_v61/cash.clientbound.CashShopOpen.yaml` —
    address 0x65a973, decompile_sha256 `c1527ada…6fc9`, verifies the new test.

## Export splice (§10, one entry, no re-export)

`CStage::OnSetCashShop` was **absent-as-resolved** in `gms_v48.json` (present but
`unresolved: true`, empty address — the IDB names it `sub_5C4D9C`, not the symbol).
Surgically replaced that one entry with the real read-site (`address: 0x5c4d9c`,
`direction: clientbound`, `calls:` documenting Decode→LoadData→bEventOn). No other
entry touched. `gms_v61.json` already carried `CStage::OnSetCashShop` @0x65a973
(`calls: null`) — pinned directly, no splice.

## Verification

- `go test ./libs/atlas-packet/...` — green (incl. new `TestCashShopOpenLegacyBodyV48V61`
  and the re-run modern `TestCashShopOpenRoundTrip` / `TestCashShopOpenLegacyGolden`).
- `go vet ./libs/atlas-packet/...` — clean.
- `go test ./tools/packet-audit/...` — green.
- `packet-audit matrix --check` — **exit 0**. Conflicts: **None**. No Problems section.

## Verified-cell count (✅) before → after

| Version | before | after | Δ |
|---------|--------|-------|---|
| gms_v48 | 165 | 166 | +1 |
| gms_v61 | 231 | 232 | +1 |
| gms_v72 | 240 | 240 | 0 |
| gms_v79 | 252 | 252 | 0 |
| gms_v83 | 392 | 392 | 0 |
| gms_v84 | 369 | 369 | 0 |
| gms_v87 | 403 | 403 | 0 |
| gms_v95 | 423 | 423 | 0 |
| jms_v185 | 385 | 385 | 0 |

`SET_CASH_SHOP` is now ✅ across all 9 versions. No version dropped; only the two
newly-gated legacy cells moved.

## Commits (branch `task-113-gms-legacy-versions`)

1. feat(packet-audit): gate CashShopOpen ZeroGoods/nHighest for legacy GMS (<72); verify v48/v61
2. docs(task-113): CASH_SHOP_OPEN legacy v48/v61 promotion report
