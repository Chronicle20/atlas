# `LOGIN_AUTH` clientbound v95 resolution

## Seed

`docs/packets/registry/gms_v95.yaml:4` (pre-fix) was an unresolved csv-import
seed:

```yaml
- op: LOGIN_AUTH
  direction: clientbound
  opcode: 0
  fname: ""
  provenance: csv-import
```

`gms_v83/v84/v87` carry this op at opcode 23 as `CLogin::OnEnableSPWResult`
(`jms_v185` calls the same function `CLogin::LoginAuth` at its own opcode 24 —
confirming `LOGIN_AUTH` and `OnEnableSPWResult`/`OnEnableSecondPassword` are
the same semantic op across client families, just renamed).

## Anchor: `CLogin::OnCheckSPWResult`

Database `ecc757f4` (GMS_v95.0_U_DEVM.exe.i64).
`CLogin::OnCheckSPWResult` (`?OnCheckSPWResult@CLogin@@IAEXAAVCInPacket@@@Z`)
is at `0x5d23f0`. `xrefs_to 0x5d23f0` returns exactly one caller:

```
0x5dfa6c -> CLogin::OnPacket (0x5df940)
```

## The v95 login dispatcher — full enumeration

Decompiling `CLogin::OnPacket` (`?OnPacket@CLogin@@UAEXJAAVCInPacket@@@Z`,
`0x5df940`) yields the complete `switch (nType)` for the login-state packet
handler. Every case, verbatim from the decompiler:

| case (opcode) | callee |
|---|---|
| 0 | `CLogin::OnCheckPasswordResult` |
| 1 | `CLogin::OnGuestIDLoginResult` |
| 2 | `CLogin::OnAccountInfoResult` |
| 3 | `CLogin::OnCheckUserLimitResult` |
| 4 | `CLogin::OnSetAccountResult` |
| 5 | `CLogin::OnConfirmEULAResult` |
| 6 | `CLogin::OnCheckPinCodeResult` |
| 7 | `CLogin::OnUpdatePinCodeResult` |
| 8 | `CLogin::OnViewAllCharResult` |
| 9 | `CLogin::OnSelectCharacterByVACResult` |
| 10 | `CLogin::OnWorldInformation` |
| 11 | `CLogin::OnSelectWorldResult` |
| 12 | `CLogin::OnSelectCharacterResult` |
| 13 | `CLogin::OnCheckDuplicatedIDResult` |
| 14 | `CLogin::OnCreateNewCharacterResult` |
| 15 | `CLogin::OnDeleteCharacterResult` |
| **21** | **`CLogin::OnEnableSPWResult`** (`0x5d2290`) |
| 24 | `CLogin::OnLatestConnectedWorld` |
| 25 | `CLogin::OnRecommendWorldMessage` |
| 26 | `CLogin::OnExtraCharInfoResult` |
| **27** | **`CLogin::OnCheckSPWResult`** (`0x5d23f0`) |
| default (16-20, 22-23, and non-`CLogin` ranges 141-146) | falls through to `CStage::OnPacket` / `CMapLoadable::OnPacket` |

Cases 16-20, 22, and 23 are not dispatched by `CLogin::OnPacket` at all — the
switch has no arm for them; `CHECK_CRC_RESULT` (opcode 23,
`CClientSocket::OnCheckCrcResult`) is handled by a different class/dispatcher
entirely, which is unrelated to this op.

There is exactly one arm shaped like `OnEnableSPWResult` in this dispatcher:
**case 21**. It is already present in `gms_v95.yaml` under a different op
name — `HACKSHIELD_REQUEST` (opcode 21, `fname: CLogin::OnEnableSPWResult`,
`provenance: csv-import`) — and, unlike the v83/v84/v87 registries (where the
CSV misfiled `OnEnableSPWResult` at the `HACKSHIELD_REQUEST` slot despite IDA
*not* dispatching it there), the v95 IDB confirms the CSV placement is
correct here: case 21 really does dispatch to `OnEnableSPWResult`. No
correction to `HACKSHIELD_REQUEST` is needed or in scope for this task.

## Error-id cross-check

Per playbook rule 2, cross-checking the `CLoginUtilDlg::Error` ids each
function reads:

- `CLogin::OnCheckSPWResult` (`0x5d23f0`, case 27): unconditionally calls
  `CLoginUtilDlg::Error(93, ...)`. Matches the brief's anchor fact.
- `CLogin::OnEnableSPWResult` (`0x5d2290`, case 21): a multi-branch switch on
  the packet's second decoded byte — `Error(18, ...)` for results 6/9,
  `Error(93, ...)` for result 0x14 (20), `Error(91, ...)` for 0x16 (22),
  `Error(92, ...)` for 0x17 (23), plus a success branch (result 0) that drives
  `CUILoginStart::SetButton`. Distinct behavior from `OnCheckSPWResult`,
  confirming these are two separate result handlers already correctly split
  across `HACKSHIELD_REQUEST` (21) and `CHECK_SPW_RESULT` (27).

## Sibling cross-check (playbook rule 3)

`xrefs_to 0x5d2290` (`OnEnableSPWResult`) returns exactly one caller — the
`CLogin::OnPacket` dispatcher itself (case 21) — confirming the client can
receive this result; there is no other landing site it could also serve.
A name search for `*SPW*` functions in the IDB (`func_query filter=*SPW*`)
returns only three hits: `OnEnableSPWResult`, `OnCheckSPWResult`, and an
unrelated cash-shop `ask_SPW` helper (second-password gate for cash-shop
purchases, called from `CCashShop`/`CPersonalShopDlg`/`CMiniRoomBaseDlg`/
`CWvsContext` sites, not from `CLogin`). No standalone `LOGIN_AUTH`/
`LoginAuth`-named function or dispatch arm exists anywhere in the v95 IDB.

## Outcome: genuinely absent

The v95 `CLogin::OnPacket` switch is fully enumerated above. Every case maps
1:1 either to an already-registered op (`LOGIN_STATUS`=0 ... `HACKSHIELD_REQUEST`=21
... `CHECK_SPW_RESULT`=27) or to a different dispatcher entirely (`CHECK_CRC_RESULT`).
There is no unclaimed opcode left for a distinct `LOGIN_AUTH` entry to
occupy — the functionality the CSV import intended it to represent
(`OnEnableSPWResult`) is already correctly recorded under `HACKSHIELD_REQUEST`
opcode 21.

Per `opregistry.go` `Applicability`: an op not present in a version's
registry file is `Absent` ⇒ graded ⬜ `n-a` on the matrix. Deleted the
`LOGIN_AUTH` clientbound entry from `docs/packets/registry/gms_v95.yaml`
(seed removed, no replacement needed).

`docs/packets/registry/gms_v92.yaml` carries the identical unresolved seed
and is explicitly out of scope for this task (§ Follow-ups) — left untouched.
