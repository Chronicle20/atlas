# Bug — the gift acknowledgement note never renders the gift/fame notice

Task: task-240-cash-shop-stub-operations
Branch: `task-240-cash-shop-stub-operations`
Base for the fix range: `906439e31`

## Reproduced

Not a live reproduction — a static one, and decisive. Every clientbound memo
Atlas writes carries `nFlag = 0`
(`services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go:48`),
and the client's `CMemoListDlg::DrawMemo` renders the gift/fame block only for
`nFlag ∈ {1, 2}`. `0` is not in that set on any of the ten client builds swept,
so the block is unreachable today.

## Observed

The cash-shop gift acknowledgement note shows only the recipient's typed
message. Nothing tells the gifter that the gift landed, and nothing tells them
they were famed — even though Atlas *does* fame them (`buildGiftFameSaga`,
`note_gift_forward.go`, awards +1 to the gifter when the gift is acknowledged).

## Expected

The gifter's memo shows two extra lines under the message:

```
<recipient> has received a gift.
<gifter>'s fame has gone up +1.
```

## Root cause

`buildGiftForwardSaga` sets `Flag: 0`. The correct value is `1`.

`nFlag` semantics, decoded from the v83 in-binary StringPool and confirmed
across every template with an IDB — full evidence, addresses and per-version
table in **`re-memo-nflag.md`** (§"Resolved"), which is the authority for this
fix:

| nFlag | meaning |
|---|---|
| 0 | plain note — no extra block |
| 1 | gift delivered **and** gifter famed +1 (both lines) |
| 2 | gift delivered, no fame line (v72+ only; v48/v61 have no such value) |
| 3 (2 on v48/v61) | wedding invitation — already modelled by `discardSpecialFlag` |

Line 1 concatenates `MEMO.bsSender` with StringPool 3366 `has received a gift.`
Line 2 concatenates `CWvsContext`'s character name (offset `+0x2098`, per
`CWvsContext::GetCharacterName`) with StringPool 3367 `'s fame has gone up +1.`

`1` — not `2` — is correct because Atlas really does award the +1: the gift
acknowledgement fires `buildGiftFameSaga` alongside the note saga. The two
lines match the two things that actually happened.

Sender/receiver mapping already lines up: the gift-forward note's *sender* is
the gift recipient and its *receiver* is the gifter, so line 1 names the
recipient and line 2 names the gifter viewing the memo. The `senderName + " "`
the codec already writes (`libs/atlas-packet/note/entry.go:26`) is exactly the
separator line 1 needs — the client inserts none.

No codec change is needed. `DiscardEntry`
(`libs/atlas-packet/note/serverbound/operation_discard.go`) branches only on
`discardSpecialFlag` (3, or 2 pre-v72), so a `flag == 1` entry decodes as an
ordinary two-field entry on every version. `atlas-notes`' sender-fame-on-discard
is already suppressed for this note by the existing `GiftNote: true`.

## Fix

1. `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`
   — in `buildGiftForwardSaga` (~line 45-48), change `Flag: 0` to `Flag: 1` and
   replace the "Flag 0 = plain note" comment with the real semantics: 1 = the
   client's gift-received + fame-gained memo block, cite `re-memo-nflag.md`, and
   state why 1 rather than 2 (Atlas awards the +1 via `buildGiftFameSaga`).

2. `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward_test.go`
   — line 34 comment and line 57 assertion currently pin `np.Flag != 0`. Flip
   the assertion to `1` and update the comment. This is the regression pin: it
   must fail before the change in (1) and pass after.

3. `services/atlas-channel/atlas.com/channel/socket/handler/note_send.go`
   — ordinary notes stay `Flag: 0`. Its comment (~line 47) says non-zero flags
   are "reserved for other memo render templates"; make it concrete now that
   they are known (1/2 = gift block, 3 = wedding invite) and point at
   `re-memo-nflag.md`. **Do not change its value.**
   `note_send_test.go:46-48` keeps asserting `0`.

### Do not touch

- The `NoteEntry` codec (`libs/atlas-packet/note/entry.go`) — `Flag` is already
  a plain byte, and its tier-1 fixtures are verified.
- `discardSpecialFlag` / `DiscardEntry` — flag 1 needs no special handling.
- `GiftNote` / `buildGiftFameSaga` — the fame award itself is correct as-is.
- Any seed template.

## Not yet answered

- **Live confirmation.** Nothing here has been seen on a running client. The
  string decode is unambiguous (clean English at exactly the blob lengths, with
  neighbouring indices decoding to English too), but a tester gifting an item
  and reading the resulting memo is the only thing that closes it.
- **Fame-award failure skew.** The memo's line 2 is fixed at note-creation time
  while the fame award is a separate saga. If that saga fails, the note would
  claim a +1 that did not land. Pre-existing to this fix (the same divergence
  exists today between `buildGiftFameSaga` and the silent note); flagged, not
  addressed here.
- **gms_v12.** No IDB exists for it, so its DrawMemo was not swept. The
  cash-shop gift flow is not exercised on that template.

## Resolution

- **Fixed by `76a7ab60e`** — `fix(atlas-channel): set nFlag=1 on cash-shop gift
  acknowledgement notes`. Touches exactly the three files in the `## Fix`
  inventory, plus this file and `re-memo-nflag.md`.

- **Review:** `review-bug-gift-note-nflag.md` — **APPROVED_WITH_FINDINGS**,
  0 blocking, 1 non-blocking. The reviewer hand-traced `Flag` across the seam
  (channel saga payload → saga-orchestrator `CreateNote` → atlas-notes
  `Model.Flag()` → note status event → clientbound `NoteEntry.Flag`) and found
  no hop that clamps, drops, or hard-codes it. It confirmed the regression pin
  genuinely fails on `906439e31` and passes after, that `discardSpecialFlag`
  (`operation_discard.go:22-27`, returns only 2 or 3) is unaffected for
  `flag == 1`, and that the `GiftNote` sender-fame suppression
  (`services/atlas-notes/atlas.com/notes/note/processor.go:340`) is untouched.

  The non-blocking finding is a **structural test gap, not a defect introduced
  by this commit**: no single automated test connects `buildGiftForwardSaga`'s
  `Flag: 1` to the terminal `NoteEntry.Encode` wire byte for the gift path.
  `note_gift_forward_test.go:57` pins the saga-payload end and
  `libs/atlas-packet/note/clientbound/display_test.go:20,41-42` pins the codec
  end; the two intervening services are covered by the manual trace only.
  Deliberately left open — closing it means a cross-service integration test,
  which is wider than this fix.

- **Gate:** flagless `tools/verify.sh` over the whole branch — **exit 0**,
  `All checks passed.` 91 Go modules built/vetted/tested with `-race`, all
  analyzer and template guards green, lint & format green over both the Go
  modules and atlas-ui, atlas-ui tests + build green.
  One caveat worth recording: the log line is
  `− docker buildx bake (no go.mod touched)` — the bake was skipped by the
  script's own no-go.mod-change condition, not by a `--quick` flag. This was a
  genuinely flagless run; the bake simply had nothing to do.

- **Live re-test: NOT yet performed.** The `nFlag == 1` render is established
  from the client binaries (string decode + a sweep of all ten templates with
  an IDB — see `re-memo-nflag.md`), but no one has yet gifted an item on a
  running client and read the resulting memo. Until that happens this is
  fixed-in-principle. The two lines to look for are:
  `"<recipient> has received a gift."` and
  `"<gifter>'s fame has gone up +1."`
