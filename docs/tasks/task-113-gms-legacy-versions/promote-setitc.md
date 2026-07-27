# Promote `field/clientbound/SetItc` to ✅ across all implemented versions

SET_ITC (`CStage::OnSetITC`) had real golden byte-tests but graded ❌ everywhere
because main cited the client read-site addresses in plain comments (not
`packet-audit:verify` markers) and carried no evidence records, so grading had no
op→struct join key. The no-report byte-fixture promotion path (commit 6c202cb7)
promotes such a cell on a marker + fresh evidence when the registry op declares a
`packet:` field. This change supplied all three per version.

## Per-version outcome

| Version | packet: field | marker (ida=) | evidence pinned | export splice | Result |
|---------|---------------|---------------|-----------------|---------------|--------|
| gms_v61 | yes | 0x65b3b4 (TestSetItcLegacyGolden) | yes (fn in export) | not needed | ✅ |
| gms_v72 | yes | 0x6c2145 (TestSetItcLegacyGolden) | yes (fn in export) | not needed | ✅ |
| gms_v79 | yes | 0x6f1c4a (TestSetItcLegacyGolden) | yes (fn in export) | not needed | ✅ |
| gms_v83 | yes | 0x7774d1 (TestSetItcRoundTrip) | yes | spliced | ✅ |
| gms_v84 | yes | 0x799e7a (TestSetItcRoundTrip) | yes | spliced | ✅ |
| gms_v87 | yes | 0x7c57d0 (TestSetItcRoundTrip) | yes | spliced | ✅ |
| gms_v95 | yes | 0x71af60 (TestSetItcRoundTrip) | yes | spliced | ✅ |
| jms_v185 | yes | 0x7ef6fa (TestSetItcRoundTrip) | yes | spliced | ✅ |

All 8 evidence records: `category: TIER1-FIXTURE`, `ida.function: CStage::OnSetITC`,
`verifies:` → the golden test. None flagged unresolvable; no fabricated sha/address.

## Byte-tests / markers

- **v83/v84/v87/v95/jms** — markers placed above `TestSetItcRoundTrip`, which
  round-trips the full body across `pt.Variants` (v83/84/87/95/jms). The
  tail-anchored `TestSetItcDefaultsGolden`/`ExplicitConfigGolden` (v95) also stand.
- **v79/v72/v61** — `pt.Variants` does not enumerate these, so a new tail-anchored
  `TestSetItcLegacyGolden` (loops GMS 79/72/61) was added; the ≥v61 SET_ITC body
  (DecodeStr account + 5×Decode4 + 8-byte DecodeBuffer) is version-stable, so the
  trailing 28 bytes equal v83's. Markers sit above it.
- Header comment rewritten: addresses are now pinned machine markers + evidence,
  and the address table was extended with the v61/v72/v79 read-sites.

## Export splices

`CStage::OnSetITC` was already present in the v61/v72/v79 exports (added by the MTS
agents) — pin worked directly. It was **absent** from v83/v84/v87/v95/jms exports;
one read-site entry each was surgically inserted (first key under `functions`,
`calls: null`, `direction: clientbound`) using the IDA-verified read-site addresses
from `set_itc.go` / `set_itc_test.go`. No full re-export — existing per-function
entries (and thus every other pinned evidence hash) are byte-unchanged.

## Tool fix (required for `matrix --check` exit 0)

The design §13 dangling-evidence `--check` guard predated 6c202cb7 and flagged
every report-less evidence record as "has no audit report", so `--check` failed the
moment any registry used `packet:`. Added `registryDeclaresPacket` exemption in
`tools/packet-audit/cmd/matrix.go`: evidence whose (packet, version) is declared via
a registry op's `packet:` field is the intended no-report promotion path, not
dangling. Regression test `TestMatrixPacketLinkedEvidenceExemptFromDangling`;
existing `TestMatrixDanglingEvidenceFailsCheck` still passes.

## Flagged unresolvable

None.

## Verification

- `go test ./libs/atlas-packet/...` — green.
- `go test ./tools/packet-audit/...` — green (incl. new + existing dangling tests).
- `go vet ./...` — clean in both changed modules.
- `packet-audit matrix --check` — exit 0.

## Verified-cell count (✅) before → after

| Version | before | after | Δ |
|---------|--------|-------|---|
| v48 | 88 | 88 | 0 (SET_ITC stays n-a; untouched) |
| gms_v61 | 153 | 154 | +1 |
| gms_v72 | 163 | 164 | +1 |
| gms_v79 | 167 | 168 | +1 |
| gms_v83 | 278 | 279 | +1 |
| gms_v84 | 270 | 271 | +1 |
| gms_v87 | 287 | 288 | +1 |
| gms_v95 | 306 | 307 | +1 |
| jms_v185 | 270 | 271 | +1 |

No version dropped; every delta is the SET_ITC ❌→✅ flip.

## Commits (branch `task-113-gms-legacy-versions`)

1. `01e5fb307` fix(packet-audit): exempt packet-linked report-less evidence from dangling --check
2. `9e80289f0` feat(packet-audit): verify SET_ITC (field/clientbound/SetItc) across all 8 versions
