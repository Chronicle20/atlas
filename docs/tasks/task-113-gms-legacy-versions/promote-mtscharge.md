# Promote `field/clientbound/MtsChargeParamResult` to ✅ across all implemented versions

MTS_CHARGE_PARAM_RESULT (`CITC::OnChargeParamResult`) is the bodiless "charge
parameter result" the client expects after ITC_STATUS_CHARGE (the MTS "Charge"
button). It had NO byte-fixture test and graded ❌/⬜ everywhere. This change
supplies the three artifacts the report-less byte-fixture promotion path (commit
6c202cb7) needs per version — a registry `packet:` field, a `packet-audit:verify`
marker, and a fresh evidence record — and it normalises the op name so all
implemented versions land on one matrix row.

## IDA-verified wire (all versions)

Every implemented client's `CITC::OnChargeParamResult` handler was decompiled and
reads **NOTHING** from the `CInPacket`: it clears the ITC request latch
(`this[6]=0` / `m_bITCRequestSent=0`), looks up the billing web URL from the
StringPool, and calls `open_web_site`. The opcode alone is the signal, so the wire
body is **empty and version-stable** — no codec version-gating needed (the writer
was already a correct empty encoder).

| version | handler entry (read-site) | dispatch opcode | body |
|---------|---------------------------|-----------------|------|
| gms_v61 | 0x52d691 | 0x111 (273) | empty (bodiless) |
| gms_v72 | 0x566768 | 0x135 (309) | empty |
| gms_v79 | 0x57f3d7 | 0x142 (322) | empty |
| gms_v83 | 0x5a4241 | 0x15A (346) | empty |
| gms_v84 | 0x5b46f8 (sub_5B46BC case 0x164) | 0x164 (356) | empty |
| gms_v87 | 0x5d4300 | 0x16F (367) | empty |
| gms_v95 | 0x575bc0 | 0x19A (410) | empty |

v84 dispatch was IDA-confirmed via the CITC sub-dispatcher `sub_5B46BC`
(`sub eax,164h; jz -> sub_5B46F8`), so charge-param = **0x164 (356)**, matching the
tenant template writer route. (The v84 registry's stale `MTS_OPERATION2`@347 /
`MTS_OPERATION`@348 are a pre-existing, non-colliding issue left untouched.)

## Registry work

The op was named inconsistently across versions. Normalised so all seven land on
the single `MTS_CHARGE_PARAM_RESULT` matrix row, and added the `packet:` link:

- **v61 / v72 / v79** — already `MTS_CHARGE_PARAM_RESULT`; added `packet:` field.
- **v83 / v87 / v95** — **renamed** placeholder `IDA_0X15A` / `IDA_0X16F` /
  `IDA_0X19A` → `MTS_CHARGE_PARAM_RESULT`; added `packet:` field + handler-entry
  note. (This also disambiguates the v87 `IDA_0X16F` row from an unrelated jms op
  `sub_48F168` that coincidentally sits at 0x16F — they were falsely unified.)
- **v84** — **added** a new `MTS_CHARGE_PARAM_RESULT` entry (opcode 356, fname
  `CITC::OnChargeParamResult`, ida 0x5b46f8, `packet:` field, IDA note). None
  existed before.

## Byte-test / markers

`libs/atlas-packet/field/clientbound/mts_charge_param_result_test.go` (new):
- `TestMtsChargeParamResultGolden` — asserts the empty (bodiless) body across all
  seven implemented variants; carries the seven `packet-audit:verify` markers.
- `TestMtsChargeParamResultRoundTrip` — bodiless round-trip across the variants.

## Export splices

`CITC::OnChargeParamResult` was absent from **every** export, so one read-site
entry was surgically inserted into each of the 7 exports (first key under
`functions`, `calls: null`, `direction: clientbound`, address = the IDA-verified
handler entry). Per-file indentation preserved (1-space unit for v61/72/79,
2-space for v83/84/87/95); 5 lines each, all other bytes byte-unchanged. No full
re-export.

## Evidence

Seven records pinned (`category: TIER1-FIXTURE`, `ida.function:
CITC::OnChargeParamResult`, sha computed by the pin tool from the spliced entry,
`verifies:` → `TestMtsChargeParamResultGolden`). Marker addr = export entry addr =
evidence addr per version.

## Per-version outcome

| version | packet: field | marker (ida=) | evidence | export splice | Result |
|---------|---------------|---------------|----------|---------------|--------|
| gms_v61 | yes | 0x52d691 | yes | spliced | ✅ |
| gms_v72 | yes | 0x566768 | yes | spliced | ✅ |
| gms_v79 | yes | 0x57f3d7 | yes | spliced | ✅ |
| gms_v83 | yes (renamed op) | 0x5a4241 | yes | spliced | ✅ |
| gms_v84 | yes (new op) | 0x5b46f8 | yes | spliced | ✅ |
| gms_v87 | yes (renamed op) | 0x5d4300 | yes | spliced | ✅ |
| gms_v95 | yes (renamed op) | 0x575bc0 | yes | spliced | ✅ |

## Flagged / out of scope

- **jms_v185** — the jms tenant template does **not** wire the `MtsChargeParamResult`
  writer (it wires `SetItc` and the ITC serverbound handlers, but no clientbound
  charge-param-result writer), so the packet is **not implemented** there. Left
  untouched; jms stays at 384 verified. Not a flag of an unresolved read-site — the
  handler is simply unrouted in jms.
- **gms_v48** — MTS is v53+; version-absent. Untouched (stays 165).

None flagged unresolvable; no fabricated sha/address (all seven read-sites resolved
and were body-verified in their IDBs).

## Verification

- `go test -race ./field/clientbound/` — green (incl. both new tests).
- `go test ./...` (atlas-packet) — green.
- `go vet ./...` (atlas-packet) — clean.
- `packet-audit matrix --check` — exit 0 (no conflicts, no dangling, no orphan).
- status.json conflict scan — 0.

## Verified-cell count (✅) before → after

| version | before | after | Δ |
|---------|--------|-------|---|
| v48 | 165 | 165 | 0 |
| gms_v61 | 230 | 231 | +1 |
| gms_v72 | 238 | 239 | +1 |
| gms_v79 | 250 | 251 | +1 |
| gms_v83 | 390 | 391 | +1 |
| gms_v84 | 367 | 368 | +1 |
| gms_v87 | 401 | 402 | +1 |
| gms_v95 | 421 | 422 | +1 |
| jms_v185 | 384 | 384 | 0 (unimplemented) |

No version dropped; every delta is the MTS_CHARGE_PARAM_RESULT ❌/⬜→✅ flip.
