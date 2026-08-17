# World-transfer (5401000) crashes the v83 client after the birthday dialog

Reported: 2026-08-16, testing task-227 on `atlas-pr-1370` (tenant
`d606f1cb-ba79-45ca-a989-cf0dc956fee7`, GMS 83.1). Name change (5400000)
works; world transfer crashes the client immediately after the birthday
confirmation is submitted.

## What the logs show

`atlas-channel-7bb98687b5-chvz7`, trace `055e0faaf4a9c2952bcd7e688b60459d`,
session `c07185fe-3912-4d35-9ce7-5053e9ffdaf2` (the cash-shop session):

```
19:20:12.046  [CashShopCheckTransferWorldPossibleHandle] read [characterId [1], credential [REDACTED]]
19:20:12.050  GET  /api/accounts/1                 -> birthDate 19900912
19:20:12.054  POST /api/accounts/1/pic-attempts    -> {"attempts":0,"limitReached":false}
19:20:12.054  GET  /api/characters?accountId=1&worldId=0&page[number]=1&page[size]=100
19:20:16.562  SESSION DESTROYED (issuer CHANNEL)   <- client gone
```

No error, warn or panic anywhere in the pod's logs. The missing "Printing
request." line for the last GET is **not** a hang: `PagedGetRequest`
(`libs/atlas-rest/requests/paged.go:50`) calls `getBody` directly and never
logs a response — only the non-paged `get()` in `requests/get.go:138` does.
The endpoint answers fine when queried directly.

So the crash happens at the *answer to the credential check*, before the
client sends anything else — there is no `CashShopOperationHandle` op 49
(`BUY_WORLD_TRANSFER`) in the log. This matches the reported symptom exactly:
birthday submitted → server answers → client dies.

Routing is correct and ruled out: the live tenant config binds
`CashShopCheckTransferWorldPossibleResult` at `0x14B` with
`operations: {ALLOWED: 0, …}`, and `docs/packets/MapleStory Ops - ClientBound.csv:539`
gives `331 / 0x14B` for v83.

## The two things the server sent in that window

`socket/handler/cash_shop_check_transfer_world_possible.go:112,120`:

1. **`CheckTransferWorldPossibleResult` ALLOWED with an empty world list.**
   `worldNames` is passed as `nil`, so `hasWorldList = false` and the client's
   `CCashShop::m_asWorldName` stays empty while arm 0 opens
   `CUITransferWorldLicenseNotice`. The handler's own doc comment (lines 65-69)
   declares this a deliberate gap: atlas-channel's `world` package has only
   `GetById` (`world/processor.go:37`), no list-all.
2. **A pink-text `WorldMessage` delivered to a client sitting in the Cash
   Shop** — `warnIfStrandingStorage` (FR-4.7). Verified it fired: account 1 has
   exactly one character in world 0 (`GET /api/characters?accountId=1&worldId=0`
   → `total: 1`, character 1 "Hard"), so `isLast` is true.

Both are unique to the world-transfer path; the name-change flow that works
sends neither (`CheckNameChangePossibleResult` has no list field, and the
pending-change pink text at `kafka/consumer/pendingchange/consumer.go:130`
only fires on *resolution*, not on purchase).

## Root cause — CONFIRMED against the v83 IDB

Candidate 1. The client crashes on a **null dereference** when the world-name
list is empty. Chain, all from `MapleStory_dump.exe` (GMS v83):

1. `CCashShop::OnCheckTransferWorldPossibleResult` @`0x47bd9b` — arm 0 opens
   `CUITransferWorldLicenseNotice` and `DoModal`s it. The list decode is
   guarded by `bHasWorldList`, so with `false` the array `this+329`
   (`m_asWorldName`) is simply never populated. No crash yet.
2. `CUITransferWorldLicenseNotice::OnCreate` @`0x7ef03a` — builds its text from
   a WZ property; never touches `m_asWorldName`. The notice renders fine.
3. `CUITransferWorldLicenseNotice::OnButtonClicked` @`0x7ef6e3`, `a2 == 1`
   (OK) — constructs `CUITransferWorldSelectDlg` (`sub_7EFA98` @`0x7efa98`,
   which zeroes `*(this+140)`, the initial combo selection) and `DoModal`s it.
4. `CUITransferWorldSelectDlg::OnCreate` @`0x7efc22` — the fill loop IS
   guarded:
   ```c
   for ( i = 0; ; ++i ) {
     v18 = *(*(this + 128) + 1316);          // CCashShop::m_asWorldName (329*4)
     if ( !v18 || i >= *(v18 - 4) ) break;   // null data ptr or count -> exit
     CCtrlComboBox::AddItem(*(this + 172), *(v18 + 4 * i), i);
   }
   sub_4C738B(*(this + 172), *(this + 140));  // SetSelected(0) on an EMPTY combo
   ```
5. `sub_4C738B` @`0x4c738b` (`CCtrlComboBox::SetSelected`) is **not** guarded:
   ```
   4c7399  mov  [esi+68h], eax        ; m_nSelected = 0
   4c73a1  push dword ptr [esi+68h]
   4c73a6  call sub_4C7379
   ```
6. `sub_4C7379` @`0x4c7379`:
   ```
   4c737d  add  ecx, 34h              ; &m_lItem
   4c7380  call sub_4C78DF            ; ZList GetAt(index) -> NULL on empty list
   4c7385  mov  eax, [eax+4]          ; <-- read [0x00000004]
   ```
   `sub_4C78DF` @`0x4c78df` with count 0 and index 0 takes the forward branch
   (`0 <= 0>>1`), returns the null head, and the caller dereferences it.

So: **`hasWorldList = false` → empty combo → `SetSelected(0)` → access
violation at `0x00000004`.** The crash lands on the OK click of the license
notice, which matches the 4.5 s between the response packet (19:20:12.05) and
the session teardown (19:20:16.56).

## Candidate 2 (pink text) — RULED OUT as the crash

`CWvsContext::OnBroadcastMsg` @`0xa22785`, arm 5 (`PINK_TEXT`, per the live
tenant `operations` table) is `CHATLOG_ADD(&msg, 12)`, and `CHATLOG_ADD`
@`0x4906b5` is wrapped in `if ( TSingleton<CUIStatusBar>::ms_pInstance )`. In
the Cash Shop there is no status bar, so the call is a silent no-op — not a
crash.

It is still a **functional** defect: the FR-4.7 storage-stranding warning is
invisible to a player in the Cash Shop, which is the only place it is ever
emitted (`cash_shop_check_transfer_world_possible.go:148`). The warning needs a
delivery mechanism the Cash Shop actually renders.

## Fix — IMPLEMENTED

### 1. Populate `worldNames` (the crash)

`world.Processor` gained `AllProvider()` / `GetAll()`
(`services/atlas-channel/atlas.com/channel/world/processor.go`), draining
atlas-world's paginated `/api/worlds` via `requests.DrainProvider` exactly as
atlas-login does — minus its `?include=channels`, which this consumer does not
need and `Extract` tolerates the absence of.

The handler now fetches the list on the ALLOWED path and, crucially, **answers
`UNKNOWN_ERROR` instead of `ALLOWED` when the list cannot be produced** (lookup
error, or an empty world set). A refusal arm renders a notice and returns the
player to the Cash Shop; an `ALLOWED`-with-no-list kills the client.

### 2. The index==world-id contract

The ordering constraint is stronger than "matching order" — the client sends
back the raw combo **index**, confirmed end-to-end on the v83 IDB this pass:

- `CUITransferWorldSelectDlg::OnCreate` @`0x7efc22` fills the combo with
  `AddItem(name, i)` in list order.
- `sub_7F00D2` @`0x7f00d2` stores `m_pComboBox->m_nSelected` (offset `0x68`)
  into `m_nResult` (offset `0x8C`).
- `CUITransferWorldSelectDlg::GetResult` @`0x7f00e2` returns `m_nResult`
  verbatim.
- `CUITransferWorldLicenseNotice::OnButtonClicked` @`0x7ef6e3` passes it to
  `CCashShop::SendBuyTransferWorldItemPacket` @`0x473601` as `nTargetWorld`.
- `handleBuyWorldTransfer` (`cash_shop_operation.go`) reads it as
  `world.Id(sp.TargetWorld())`.

So `transferWorldNameList` builds a slice of length `max(world id) + 1` and
fills **by id**, not by append order. A gap leaves a blank combo entry, which
`pendingchange.RequestWorldTransfer` already rejects with `world_unknown` —
collapsing the gap instead would shift every later world's index and silently
misroute the purchase.

### 3. Candidate 2 — the invisible warning

The FR-4.7 storage-stranding warning now goes out as **`POP_UP` (arm 1)**, not
`PINK_TEXT` (arm 5). `CWvsContext::OnBroadcastMsg` @`0xa22785` routes arm 1
straight to `CUtilDlg::Notice` with no status-bar guard, while arm 5's
`CHATLOG_ADD` @`0x4906b5` is entirely inside
`if (TSingleton<CUIStatusBar>::ms_pInstance)` — a no-op for a client in the
Cash Shop. `POP_UP` is already the delivery `announceCashShopRejection` uses,
and is configured (`POP_UP: 1`) in every seed template's WORLD_MESSAGE
`operations` table.

### Tests

`cash_shop_check_transfer_world_possible_test.go`:
`TestTransferWorldPossibleAllowedCarriesTheWorldNameList`,
`TestTransferWorldNameListIsIndexedByWorldId` (out-of-order / gap / empty),
`TestTransferWorldPossibleWorldListFailureRefusesRatherThanCrashing`,
`TestWorldTransferStorageWarningUsesPopUpNotPinkText`.
