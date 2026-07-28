# Post-merge delta review — task-113 (a19c39486d..HEAD)

**Verdict: PASS.** Read-only adversarial review of the 22-commit delta since the
merge. Tool-integrity holds; no version regressed; modern CashShopOpen byte-identical;
no new fname-doc drift; no TODO/stub/abspath. All bars green.

## 1. Tool integrity (grade.go + matrix.go) — PASS, cannot green without byte-verification

**6c202cb7 (no-report byte-fixture promotion):** `grade.go:188-211`. The no-report
branch promotes to ✅ **only** when `a.marker.Found && a.hasEvidence && a.evidence.Fresh`.
- Stale fixture (`!Fresh`) → `StateIncomplete` (line 201-207). Verified by
  `TestGradeByteFixtureNoReportStaleEvidenceIncomplete`.
- Marker present but no evidence → `StateIncomplete` (line 208-210). Verified by
  `TestGradeByteFixtureNoReportNoMarkerIncomplete`.
- No `packet:` link → falls through to `"no audit report"` Incomplete (line 211).
  Verified by `TestGradeNoReportNoPacketLinkStaysIncomplete` (asserts exact note).
- Family/dispatcher capping preserved: `a.family` → `StateFamily`, never ✅ (line
  196-198). Verified by `TestGradeByteFixtureNoReportFamilyCapped`.
- The gate is identical to the pre-existing tier-1 path (`grade.go:233`); the change
  extends the same marker+fresh-evidence trust model to report-less packets, it does
  not weaken it.

**01e5fb30 (dangling-evidence --check exemption):** `matrix.go:166-172` +
`registryDeclaresPacket` (line 344-357). Exemption fires **only** when the same
version's registry has an op whose `packet:` == the evidence key. Genuinely dangling
evidence (no report, no `packet:`) still fails --check — the pre-existing
`TestMatrixDanglingEvidenceFailsCheck` still passes, and the new
`TestMatrixPacketLinkedEvidenceExemptFromDangling` asserts the narrow exemption.

**Could anything green without byte-verification? NO.** Marker scan + fresh
decompile-hash-matched evidence + registry `packet:` link are all required.

## 2. No regression — PASS

Computed verified counts from status.json, base vs HEAD:

| version | base | head | Δ |
|---|---|---|---|
| gms_v48 | 165 | 166 | +1 |
| gms_v61 | 208 | 232 | +24 |
| gms_v72 | 216 | 240 | +24 |
| gms_v79 | 228 | 252 | +24 |
| gms_v83 | 389 | 392 | +3 |
| gms_v84 | 366 | 369 | +3 |
| gms_v87 | 400 | 403 | +3 |
| gms_v95 | 420 | 423 | +3 |
| jms_v185 | 383 | 385 | +2 |

Every delta positive; HEAD totals match the stated targets exactly. SET_ITC ✅ on
v61–jms, ⬜ on v48 (n-a). MTS_CHARGE_PARAM_RESULT ✅ v61–v95, ⬜ v48 and ⬜ jms
(not wired — jms registry has no `packet:` for it). SET_CASH_SHOP ✅ all 9.

## 3. CashShopOpen legacy gate — PASS (modern byte-identical)

`shop_open.go` gate changed `MajorVersion() > 12` → `MajorAtLeast(72)` for
DecodeZeroGoods, and nHighest inner `Region()=="GMS"` → `MajorAtLeast(72)`.
- Modern GMS (72/79/83/84/87/95): `>12` and `>=72` both true; nHighest GMS both true
  → unchanged.
- JMS: ZeroGoods and nHighest both already false (GMS-only) → unchanged.
- Only v48/v61 (<72) lose ZeroGoods(2B) + nHighest(4B) — IDA body-verified in commit.

## 4. MTS extension — PASS (re-derived per IDB, not copied)

Evidence for FieldItcOperation* across v61/v72/v79 carries **distinct addresses AND
distinct decompile hashes** per version (e.g. OnBuy 0x529964/0x562009/0x57ac34).
matrix --check validated all hashes fresh. v61 test file markers cite per-mode v61
IDA send-site addresses (RegisterSale 0x528f35 a2==0, RegisterAuction 0x528f35 a2==1,
StatusCharge 0x528ed7). v48 MTS/ITC n-a is evidence-backed: "MTS added GMS v53; v48
(<53) has only CITCWnd UI shell, not CITC packet class" with IDA citations.

## 5. Hygiene — PASS

No TODO/FIXME/stub/501 in delta Go (the one grep hit was item id `50100008`
containing "501"). No `/home/`/`/Users/` abs paths in added lines. `git status` clean.

## Bars

- `go build ./...` (libs/atlas-packet, tools/packet-audit): PASS (exit 0)
- `go vet ./...` both modules: PASS (clean)
- `go test ./libs/atlas-packet/... ./tools/packet-audit/...`: PASS
- `matrix --check`: PASS (exit 0)
- `fname-doc --check`: 4 missing (buddy OperationAdd, chat WorldMessageSimple, chat
  Multi, field FieldObstacleOnOffList) — **pre-existing/known, no new MTS drift**.
