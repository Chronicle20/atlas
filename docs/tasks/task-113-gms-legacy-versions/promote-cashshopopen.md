# Promote `cash/clientbound/CashShopOpen` to ✅ where the writer is faithful

CASH_SHOP_OPEN (`CStage::OnSetCashShop`) had a real round-trip byte-test but graded
❌ everywhere because the client read-site addresses were never pinned as
`packet-audit:verify` markers and no evidence records existed — the same
golden-test-but-no-machine-marker gap SET_ITC had. This change supplied the
registry `packet:` field + marker + fresh evidence for every version whose client
reads the MODERN Cash Shop body, following the no-report byte-fixture promotion
path (commit 6c202cb7).

**Critical IDA finding (not assumed — body-verified per version):** the Cash Shop
wire body is NOT version-stable the way SET_ITC's was. The `CashShopOpen` writer
targets the modern layout, which the client reads as
`CCashShop::CCashShop -> CCashShop::LoadData`:

```
Decode1  bCashShopAuthorized (GMS; JMS omits and reads account first)
DecodeStr m_sNexonClubID (account)
CWvsContext::SetSaleInfo  Decode4 nNotSaleCount (GMS only) + Decode2 special
                          + [Decode2 extra] (JMS only) + Decode1 discounts
DecodeBuffer(1080)  the "Best" block  ==  writer's 9*2*5 int32 triples (=1080 bytes)
DecodeStock / DecodeLimitGoods (GMS>12||JMS) / DecodeZeroGoods (GMS only)
Decode1  m_bEventOn
Decode4  m_nHighestCharacterLevelInThisAccount (GMS only)
```

The 9×2×5×3×4 = **1080** bytes the writer emits in its "Decode Best" loop is
exactly the `DecodeBuffer(1080)` the client reads — this is the load-bearing
correctness fact and it holds for v72/v79/v83/v84/v87/v95/jms.

## Per-version outcome

| Version | OnSetCashShop | body reader | format | packet:/marker/evidence | Result |
|---------|---------------|-------------|--------|--------------------------|--------|
| gms_v48 | sub_5C4D9C `0x5c4d9c` (opcode **74**) | ctor 0x447122 → sub_44E1E5 | **LEGACY** | — | **FLAGGED (n-a in matrix)** |
| gms_v61 | `0x65a973` | ctor 0x453549 → LoadData 0x45b539 | **LEGACY** | — | **FLAGGED (stays ❌)** |
| gms_v72 | `0x6c16c3` | ctor 0x461fc9 → LoadData 0x46a706 | modern | yes / LegacyGolden / pinned | ✅ |
| gms_v79 | `0x6f11c8` | ctor 0x462e86 → LoadData 0x46b86c | modern | yes / LegacyGolden / pinned | ✅ |
| gms_v83 | `0x776a4f` | ctor 0x468223 → LoadData 0x471f37 | modern | yes / RoundTrip / pinned | ✅ |
| gms_v84 | `0x7993b6` | (byte-identical to v83) | modern | yes / RoundTrip / pinned | ✅ |
| gms_v87 | `0x7c4d0c` | ctor 0x471159 → LoadData 0x47c848 | modern | yes / RoundTrip / pinned | ✅ |
| gms_v95 | `0x71adf0` | ctor 0x4938b0 → LoadData 0x492ea0 | modern | yes / RoundTrip / pinned | ✅ |
| jms_v185| `0x7ef5f2` | ctor 0x47811b → LoadData 0x4839a9 | modern-JMS | yes / RoundTrip / pinned | ✅ |

All 7 evidence records: `category: TIER1-FIXTURE`, `ida.function:
CStage::OnSetCashShop`, `verifies:` → the golden test. None fabricated.

### Body-verification evidence (why v48/v61 are LEGACY, not just "unverified")

- **v83** (reference, fully verified): `CCashShop::LoadData 0x471f37` reads
  `SetSaleInfo` (Decode4 nNotSaleCount + Decode2 special + Decode1 discounts,
  confirmed in `SetSaleInfo 0xa25db4`) → `DecodeBuffer(1080)` → `DecodeStock` →
  `DecodeLimitGoods` → `DecodeZeroGoods`; ctor `0x468223` then reads `Decode1`
  (bEventOn) + `Decode4` (nHighest). Matches the writer's GMS branch byte-for-byte.
- **jms_v185**: `LoadData 0x4839a9` reads account UNCONDITIONALLY (no auth byte),
  `SetSaleInfo 0xb0d593` = Decode2 special + Decode2 extra + Decode1 discounts (NO
  nNotSaleCount), Stock, Limit, **no ZeroGoods**; ctor `0x47811b` reads only
  `Decode1` (no nHighest). Matches the writer's JMS branch exactly.
- **v72 / v79**: ctors read `Decode1 + Decode4`; LoadData has all three of
  Stock/Limit/**Zero** (v79 `sub_46C52F`, v72 `sub_46B3C9`). Modern. Since every
  GMS major > 12 takes the identical writer branch, their bytes equal v83's — the
  new `TestCashShopOpenLegacyGolden` asserts both the 23-byte tail AND full-buffer
  equality with the v83 encoding.
- **v61 (LEGACY)**: ctor `0x453549` reads only ONE `Decode1` after LoadData — **no
  `Decode4` nHighest**; `LoadData 0x45b539` after `DecodeBuffer(1080)` has only two
  post-buffer decoders (Stock `sub_45C497`, Limit `sub_45C4DE`) — **no
  DecodeZeroGoods**. The writer's GMS branch unconditionally emits ZeroGoods (2B) +
  nHighest (4B) for every GMS>12, so it over-emits 6 bytes the v61 client never
  reads. Promoting v61 would assert bytes the client does not read → a false ✅.
- **v48 (LEGACY, present)**: the Cash Shop scene-entry EXISTS — `CField::OnPacket
  0x4c66f2` dispatches opcodes 72–75 to `sub_5C45ED`, which routes opcode **74** to
  `sub_5C4D9C 0x5c4d9c` (CStage::OnSetCashShop equivalent: reads CharacterData via
  `sub_49D320`, constructs `CCashShop` via `sub_447122(a1)`, `set_stage`). Its body
  reader `sub_44E1E5` uses the same LEGACY shape as v61 (SetSaleInfo + fixed
  `DecodeBuffer(0x438)`=1080, no ZeroGoods; ctor reads a single `Decode1`, no
  nHighest). Not registered — the current writer cannot faithfully encode v48.

## v48 present-or-n-a determination

**PRESENT** (opcode 74, handler `0x5c4d9c`, body reader `sub_44E1E5`), but the
`cash/clientbound/CashShopOpen` writer does not produce the v48 legacy body.
Left unregistered (matrix cell stays ⬜ / n-a) rather than registered-then-❌ to
avoid a registry↔template conflict; this is a genuine "present but not implementable
with the current writer" flag, not an absence. A future task adding a v48/v61
legacy Cash Shop writer (or gating ZeroGoods/nHighest to `MajorAtLeast(72)`) can
then register + promote v48/v61.

## Export splices

`CStage::OnSetCashShop` was already present in the v72/v79 exports (addresses
0x6c16c3 / 0x6f11c8, matching the registry `ida.address` notes) — pin worked
directly. It was **absent** from v83/v84/v87/v95/jms exports; one read-site entry
each was surgically inserted (`calls: null`, `direction: clientbound`, IDA-verified
addresses above) as the first key under `functions`. No full re-export — every
other pinned evidence hash is byte-unchanged.

## Flagged / not promoted

- **gms_v48**, **gms_v61** — Cash Shop scene-entry present in-client but the wire
  body is the legacy pre-ZeroGoods / pre-nHighest layout the modern writer does not
  emit. Not promoted (would be a false ✅). IDA evidence above.

## Verification

- `go test ./libs/atlas-packet/...` — green (incl. new `TestCashShopOpenLegacyGolden`).
- `go vet ./libs/atlas-packet/...` — clean.
- `go test ./tools/packet-audit/...` — green.
- `packet-audit matrix --check` — exit 0. `operations --check` — exit 0.
- `packet-audit fname-doc --check` — exit 1 with **4 pre-existing** missing entries
  (buddy/chat/field; verified identical on the pre-change tree via `git stash`).
  Unrelated to this change; no cash files involved.

## Verified-cell count (✅) before → after

| Version | before | after | Δ |
|---------|--------|-------|---|
| gms_v48 | 165 | 165 | 0 (legacy; flagged) |
| gms_v61 | 231 | 231 | 0 (legacy; flagged) |
| gms_v72 | 239 | 240 | +1 |
| gms_v79 | 251 | 252 | +1 |
| gms_v83 | 391 | 392 | +1 |
| gms_v84 | 368 | 369 | +1 |
| gms_v87 | 402 | 403 | +1 |
| gms_v95 | 422 | 423 | +1 |
| jms_v185 | 384 | 385 | +1 |

No version dropped; every delta is the CASH_SHOP_OPEN ❌→✅ flip on a
body-verified version.

## Commits (branch `task-113-gms-legacy-versions`)

1. feat(packet-audit): verify CASH_SHOP_OPEN (cash/clientbound/CashShopOpen) across 7 modern versions
2. docs(task-113): CASH_SHOP_OPEN coverage-matrix promotion report
