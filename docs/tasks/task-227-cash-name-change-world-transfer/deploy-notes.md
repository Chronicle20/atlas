# task-227 — deploy notes

Operational steps this feature needs that code alone does not perform. Everything
here is a **live-tenant action**; none of it is exercised by CI or by
`tools/verify.sh`.

Opcode values below are transcribed from `docs/packets/audits/STATUS.md` (the
generated matrix), not from any task report. Where a report's prose and STATUS.md
disagree, STATUS.md wins — the task-18 progress note abbreviates its row with the
v48/v61 columns mislabelled as v72/v79, and the matrix is the authority.

---

## 1. Tenant socket-config templates — opcode table

Nine GMS versions plus JMS185. `—` means the op does not exist on that client and
must **not** be bound; binding it would either shadow a real op or register a
writer that can never fire.

### Clientbound

| Op | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | JMS185 |
|---|---|---|---|---|---|---|---|---|---|---|
| `CASHSHOP_CHECK_NAME_CHANGE` | 0x102 | 0x101 | 0x125 | 0x131 | 0x148 | 0x14F | 0x159 | 0x17B | 0x183 | — |
| `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT` | — | — | — | 0x132 | 0x149 | 0x150 | 0x15A | 0x17C | 0x184 | — |
| `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` | 0x105 | 0x104 | 0x128 | 0x134 | 0x14B | 0x152 | 0x15C | 0x17E | 0x186 | 0x16C |
| `CANCEL_NAME_CHANGE_RESULT` | — | 0x056 | 0x069 | 0x06B | 0x071 | 0x074 | 0x074 | 0x076 | 0x075 | — |
| `CANCEL_TRANSFER_WORLD_RESULT` | — | 0x057 | 0x06A | 0x06C | 0x072 | 0x075 | 0x075 | 0x077 | 0x076 | — |
| `CANCEL_NAME_CHANGE_BY_OTHER` | — | — | 0x070 | 0x072 | 0x078 | 0x07B | 0x07B | 0x07D | 0x07C | — |

### Serverbound

| Op | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | JMS185 |
|---|---|---|---|---|---|---|---|---|---|---|
| `NAME_TRANSFER` | 0x012 | 0x010 | 0x010 | 0x010 | 0x010 | 0x010 | 0x010 | 0x010 | 0x010 | — |
| `WORLD_TRANSFER` | 0x014 | 0x012 | 0x012 | 0x012 | 0x012 | 0x012 | 0x012 | 0x012 | 0x012 | 0x009 |

### Traps in this table

- **v48 is not v61+.** `NAME_TRANSFER`/`WORLD_TRANSFER` bind `0x012`/`0x014` on
  v48 but `0x010`/`0x012` from v61 on. Copying the v61 constants into v48 binds
  both ops one slot low.
- **v48 and v61 run *higher* to *lower* on the two cash-check rows**
  (`0x102`→`0x101`, `0x105`→`0x104`). That is a genuine decrement, not a
  transcription slip.
- **v84 and v87 legitimately share a value** on `CANCEL_NAME_CHANGE_RESULT`
  (`0x074`), `CANCEL_TRANSFER_WORLD_RESULT` (`0x075`) and
  `CANCEL_NAME_CHANGE_BY_OTHER` (`0x07B`). Each was derived from its own IDB
  (provenance confirmed in the task-20/21/22 reviews). Do not "fix" the
  collision.
- **JMS185 has no name-change feature at all** — only `WORLD_TRANSFER` (0x009)
  and `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` (0x16C). The exclusion was
  verified against the JMS IDB, not inferred.

---

## 2. Pending-change expiry config is **not** auto-seeded

`imprint-configs` on atlas-tenants is reachable only through an explicit
`POST .../imprint-configs/seed`. There is no reconciliation on tenant creation —
this mirrors `trade-configs`, which behaves identically and is likewise absent
from the auto-seed `Groups()` list.

A live tenant therefore keeps the **168h** built-in default until someone POSTs
the seed or PATCHes the config. The default is safe, so this is a deployment
note rather than a defect, but it means a tenant wanting a different expiry
window gets the default silently until acted on.

---

## 3. Reason taxonomy is a server record, not a wire contract

The four rejection reasons (`name_taken`, `name_reserved`,
`name_invalid_length`, `name_invalid_charset`) are persisted in
`pending_change.Reason` and surfaced through REST and the operator panel.

Only `name_taken` has a faithful client arm. The real client — verified by
decompile on v48, v61, v72, v83 and v95 — renders name-change results
three-way (taken / available / generic unknown-error) and has **no arm** for
reserved, too-short or bad-charset. Those three collapse to a generic in-client
error on every version.

This is GMS-authentic behaviour. Operators diagnosing "player says they got a
generic error" should read the reason off the pending-change record or the
operator panel, which is the only surface that distinguishes the four.

---

## 4. Pre-existing defects found in passing (not introduced here)

Recorded because they were confirmed during this task and would otherwise be
lost:

- **`RemoteMerchant` is missing from `saga/timer.go`'s `allSagaTypes`** — the
  exact defect class that list exists to guard against. `WorldTransfer` was
  missing too and was added on this branch; `RemoteMerchant` was left alone as
  out of scope, but it means remote-merchant sagas never time out.
- **`character/administrator.go`'s `update()` switches over column names by
  hand**, so any future `SetX` that lacks a matching `case` silently no-ops the
  DB write. This bit the world-change arm on this branch (proved red→green). No
  lint or CI check guards it.
- **`packet-audit`'s `-ida-url` default points at a dead port** (`:13337/mcp`;
  live is `:8745/mcp`). It fails loudly against a dead port, but against a
  permissive server on the default it would silently harvest the wrong binary.
- **`packet-audit export -splice` returns HTTP 400** against this environment's
  IDA MCP endpoint (schema mismatch on the `database` param), which forced
  hand-splicing of five export entries during task 22. The hashes were verified
  genuine, but the tool defect stands.
