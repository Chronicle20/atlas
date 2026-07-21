# FieldTournamentMatchTable (← `CField_Tournament::OnTournamentMatchTable`)

- **IDA:** 0x5a9ed7
- **Atlas file:** `libs/atlas-packet/field/clientbound/tournament_match_table.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | bytes | bytes `match table (m_aaMatch; 0x300=768-byte raw blob, PDB-typed unsigned int[32][6]; single bulk memcpy, not per-field reads; read inside CMatchTableDlg::CMatchTableDlg ctor @0x83517f)` | ✅ |  |
| 1 | byte | byte `state (m_nState)` | ✅ |  |

