# RE findings — the memo `nFlag` and its client-side render branches

Task: task-240-cash-shop-stub-operations
Question asked: is there a flag in the clientbound note write that gates a
"you'll receive fame" style notice?
Answer: **yes, `nFlag == 1` gates an extra rendered block** — but the block's
text is NOT yet resolved. See "Unresolved" below before acting on this.

IDBs read: GMS v83 (`MapleStory_dump.exe.i64`) and GMS v95.0
(`GMS_v95.0_U_DEVM.exe.i64`).

## The wire field

`NoteEntry` (`libs/atlas-packet/note/entry.go`) writes
`id / senderName+" " / message / int64 time / flag`. That maps exactly onto the
client's `GW_Memo`, whose v95 layout (from the IDB's own type, PDB-derived) is:

| member | offset | type |
|---|---|---|
| `dwSN` | 0x00 | unsigned int |
| `sSender` | 0x04 | char[13] |
| `sContent` | 0x11 | char[201] |
| `dateSent` | 0xDA | _FILETIME |
| `nFlag` | 0xE2 | int |

`nFlag` is the **only** flag on a memo. There is no separate fame field, and no
fame-related member anywhere in the struct.

The per-entry decode confirms the same order — v83 `sub_4E4ADB` (called once per
item from `CWvsContext::OnMemoResult` mode 3 / DISPLAY) reads
`Decode4` → `DecodeStr` → `DecodeStr` → `DecodeBuffer(8)` → `Decode1`, storing
the final byte at `+226*4`.

`nFlag` is copied verbatim into the UI's own record,
`CMemoListDlg::MEMO` (v95 IDB type), at `+0x18`:

| member | offset |
|---|---|
| `bCheck` | 0x00 |
| `nYPos` | 0x04 |
| `dwSN` | 0x08 |
| `bsSender` | 0x0C |
| `sTime` | 0x10 |
| `absContent` | 0x14 |
| `nFlag` | 0x18 |

## The branches on nFlag

Only **two** values are special-cased anywhere in the memo UI:

### `nFlag == 3` — wedding invitation (already known)

`CMemoListDlg::SetRet` (v83 `0x64aa57`) counts entries with
`*(v9 + i + 24) == 3`, sends that count, and for those entries — if an ETC slot
is free — parses the marriage number out of the message and appends it.
Otherwise it shows `SP_4261_YOUR_ETC_SLOT_IS_FULL...TO_RECEIVE_WEDDING_INVITES`.
This is what `discardSpecialFlag` in
`libs/atlas-packet/note/serverbound/operation_discard.go:22` already models
(2 on v48/v61, 3 from v72 on).

v95 `CMemoListDlg::DrawMemo` (`0x6247b0`) has the matching render arm:
`if (*(v121 + 24) != 3)` takes the ordinary content path; the `== 3` path
`Find`s `"_"` in the content and draws only `Mid(...)` after it.

### `nFlag == 1` — an EXTRA two-line block is drawn

This is the branch that answers the question.

**v95, `CMemoListDlg::DrawMemo`** — after the ordinary content lines, inside a
guard, the decompiler shows:

```c
v69 = *(v119 + 24) == 1;
...
if ( v69 )
{
    // Copy() a canvas, then a second DrawTextA of a string built earlier
}
```

The two strings drawn in that block are built as:

- `Ztl_bstr_t::operator+( <MEMO.bsSender>, StringPool::GetBSTR(3399) )`
  — the note's **sender name** concatenated with a string-pool entry.
- `Ztl_bstr_t::operator+( <value at CWvsContext + 2099*4>, StringPool::GetBSTR(3400) )`
  — a CWvsContext-held string (very likely the local character's name)
  concatenated with a second string-pool entry.

**v83, `CMemoListDlg::OnCreate`** (`sub_649FBD`) — the same flag reserves the
vertical space for that block when the entry list is built:

```c
v62[6] = *(v60 + 226);   // MEMO.nFlag <- GW_Memo.nFlag
...
v82 = v62[6] - 1;        // nFlag - 1
...
if ( !v82 )              // nFlag == 1
{
    v84 = sub_40B947(*(this + 156));
    v119 = v84 + v83 + 2 * v84;   // three extra line-heights for this entry
}
```

So the behaviour is present on both the v83 target and v95, and it is gated
purely by `nFlag == 1`.

## What Atlas does today

`buildGiftForwardSaga`
(`services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`)
sets `Flag: 0` on the gift-forward note, so this block never renders for a
cash-shop gift thank-you note. `buildNoteSendSaga` (`note_send.go`) also sends 0.
Atlas never emits `nFlag == 1` anywhere.

## Unresolved — do not act on this without closing it

The **text** of string-pool entries 3399 and 3400 is not established, so what
the extra block actually *says* is unknown. It was not possible to read them
reliably:

- `StringPool::GetString` (v95 `0x746750`) decodes lazily from
  `StringPool::ms_aString` (`0xc5a878`) with a 16-byte key at
  `StringPool::ms_aKey` (`0xb98830`), rotated left by a per-string seed
  (`rotatel<unsigned char>`, `0x746270`) and XORed
  (`Decode<char>`, `0x746520`, with the `enc == key → plain = key` quirk that
  avoids embedding NULs).
- Reimplementing that decode and indexing `ms_aString[3399]` / `[3400]`
  yields **WZ resource paths** — `"Letter"`, `"Mark.img/Mark/%s/%08d/%d"`,
  and neighbours like `"/BackGround/%08d/%d"`, `"Info/BtDelete"` — not UI
  sentences. The per-string seed also did not sit at a consistent offset
  across the blobs that were checked. Both signs point at the index→pointer
  arithmetic being off by some amount, so **these decoded values must not be
  quoted as the memo strings.**

Consequently: it is confirmed that `nFlag == 1` shows an extra
sender-name-plus-text / character-name-plus-text block, and it is NOT confirmed
that the block is a fame notice. Setting `Flag: 1` on gift-forward notes is a
plausible next step but would be a guess until the strings are read.

Ways to close it, cheapest first:

1. Read the memo strings out of the client's `String.wz` / `UI.wz`
   (`UI.wz/UIWindow.img/Memo`) rather than the in-binary obfuscated pool.
2. Fix the `ms_aString` base/stride and re-decode 3399 / 3400.
3. Set `Flag: 1` on a gift note on a live client and read the rendered block
   directly — decisive, and cheap if a test client is already up.
