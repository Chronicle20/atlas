# FieldTournamentMatchTable (← `CField_Tournament::OnTournamentMatchTable`)

- **IDA:** 0x58b29b
- **Atlas file:** `libs/atlas-packet/field/clientbound/tournament_match_table.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | bytes | bytes `match table (m_aaMatch; 0x300=768-byte raw blob, PDB-typed unsigned int[32][6]; single bulk memcpy, not per-field reads; byte-identical to v83 ctor helper)` | ✅ |  |
| 1 | byte | byte `state (m_nState)` | ✅ |  |

