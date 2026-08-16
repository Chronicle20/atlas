# The name-availability check was never wired (found testing on atlas-pr-1370)

## Symptom

```
"message":"Read a unhandled message with op 0x15.","service.name":"atlas-channel",
"ms.version":"83.1","region":"GMS"
```

logged every time the player typed a candidate name into the cash-shop rename
dialog. The dialog blocked forever because nothing answered.

## Cause

The client has ONE serverbound opcode for "check this character name" —
`CHECK_CHAR_NAME`, gms_v83 `0x15`. `docs/packets/MapleStory Ops -
ServerBound.csv` row 33 lists **two senders** against that single opcode
column, and `docs/packets/registry/gms_v83.yaml:1891` carries the second as an
`fname_alt`:

```
CLogin::SendCheckDuplicateIDPacket      (character creation, login socket)
CCashShop::SendCheckDuplicateIDPacket   (cash shop rename, channel socket)
```

Every template bound that opcode with `"services": ["login"]` only, so
`opcodes.BuildHandlerMap(l, ServiceChannel, …)` skipped it and the channel had
no handler. The clientbound half — `cash/clientbound.CheckNameChange`
(`CCashShop::OnCheckDuplicatedIDResult`) — had been fully implemented and
verified by Task 19, and its writer was configured in all nine GMS templates,
but **no handler and no `main.go` writer registration ever referenced it**:
`grep -rn CashShopCheckNameChangeWriter --include='*.go' services/` returned
nothing before this fix.

Plan Task 26 ("The availability-check handlers") named
`cash_shop_check_name_change.go` in its Files list, but its Step 5 registered
`NAME_TRANSFER` / `WORLD_TRANSFER` instead — the two `*_POSSIBLE` credential
ops. Those two shipped (`cash_shop_check_name_change_possible.go`,
`cash_shop_check_transfer_world_possible.go`); the name-availability probe did
not. The task's own verification could not catch it: nothing in the gate
asserts that a configured writer has a caller.

## Fix

Commit `dbe555d1c`:

- `libs/atlas-packet/cash/serverbound/check_name_change.go` —
  `CheckNameChangeRequest`, a single `EncodeStr(sCharName)`. Its own type
  rather than a reuse of `charsb.CheckName` because the handler NAME is the
  template binding key and the two sockets answer with different clientbound
  ops. Named `…Request`, not `CheckNameChange`, to avoid the task-226
  `Title(family)+Struct` collision in `packet-audit fname-doc` with the
  clientbound codec of the same name.
- `services/atlas-channel/.../socket/handler/cash_shop_check_name_change.go` —
  calls atlas-character `GET /characters/name-validity` at **TENANT** scope
  (FR-3.2) and maps its four reasons onto the client's three-way signed
  branch. Only `duplicate` has a distinct client arm; the other three land on
  the generic error arm, which is a fact about every GMS build examined (see
  `cashcb.CheckNameChange`'s doc comment), not a shortcut here.
- `services/atlas-channel/.../character/name_validity_requests.go` — the
  plain-JSON client for that endpoint (it is not JSON:API, so
  `requests.GetRequest[T]` cannot be used), mirroring
  atlas-character-factory's.
- Nine GMS templates gain the channel-scoped handler entry at the
  `CHECK_CHAR_NAME` opcode: gms_v48 `0x11`, the other eight `0x15`. jms_v185
  has no name-change feature at all (derivation.md §1.5) and is untouched.

## What the body shape rests on

`CLogin::SendCheckDuplicateIDPacket` is a single `EncodeStr` in the checked-in
IDA exports for v61/v72/v79/v83/v84/v87/v95, and both receivers
(`CLogin::OnCheckDuplicatedIDResult`, `CCashShop::OnCheckDuplicatedIDResult`)
decode the same `DecodeStr + Decode1` on all nine versions — i.e. the two
halves of this exchange are the same packets throughout.
`CCashShop::SendCheckDuplicateIDPacket` itself is **not named in any
checked-in IDB export**, so the cash-shop sender's body is asserted from the
shared opcode plus the identical result codec, not from a direct decompile. A
trailing field present only on the cash-shop variant would not be visible in
that evidence. No `packet-audit:verify` marker was added for this reason — the
matrix cells for `CHECK_CHAR_NAME` stay owned by
`character/serverbound/CheckName`'s test.

## Deploy note

The handler entries are **seed data**. An ephemeral env whose tenant socket
config already exists will not be re-seeded, so a running `atlas-pr-*`
environment needs the two new bindings PATCHed into its tenant configuration
(or a fresh env) before the fix is observable there.
