# FieldTournament (← `CField_Tournament::OnTournament`)

- **IDA:** 0x57b61a
- **Atlas file:** `libs/atlas-packet/field/clientbound/tournament.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode; consumed inside the leading branch condition itself (C \|\| short-circuit against a client-local TSecType flag, never a wire read) -- selects one of two mutually-exclusive UI arms` | ✅ |  |
| 1 | byte | byte `value; read unconditionally by whichever arm is selected -- champion/finalist/round-N notice in one arm, prize-not-set/insufficient-users notice in the other. Both arms terminate after this byte.` | ✅ |  |

