# RE findings — the memo `nFlag` and its client-side render branches

Task: task-240-cash-shop-stub-operations
Question asked: is there a flag in the clientbound note write that gates a
"you'll receive fame" style notice?
Answer: **yes — `nFlag == 1` renders exactly the gift + fame notice.** The two
strings are now decoded out of the v83 in-binary StringPool:

- `"<memo sender> has received a gift."`
- `"<local character name>'s fame has gone up +1."`

See "Resolved" below for the decode and the per-version sweep.

IDBs read: GMS v48, v61, v72, v79, v83 (`MapleStory_dump.exe.i64`), v84.1, v87,
v92.1, v95.0 (`GMS_v95.0_U_DEVM.exe.i64`), and JMS v185.

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

## Resolved — the strings, decoded from the v83 pool

Closed by decoding the v83 (target-version) StringPool rather than v95's. The
v95 attempt failed on index→pointer arithmetic; v83's pool is a simple
pointer table and decodes cleanly.

### v83 decode mechanics (all addresses from `MapleStory_dump.exe.i64`)

| Symbol | Address | Role |
|---|---|---|
| `StringPool::GetBSTR` | `0x406292` | public entry; index is the `StringPoolStrings` ordinal |
| `StringPool::GetString` (private) | `0x79e993` | lazy decode + cache |
| `off_BDC9D4` | `0xBDC9D4` | `ms_aString` — flat `char*` table, `[idx]` at `0xBDC9D4 + 4*idx` |
| `dword_B001EC` | `0xB001EC` | 16-byte key `d6 de 75 86 46 64 a3 71 e8 e6 7b d3 33 30 e7 2e` |
| `dword_B001FC` | `0xB001FC` | key length = `16` |
| `sub_79EBF3` | `0x79EBF3` | rotate-left the key by the seed: byte-rotate `(seed>>3) % 16`, then bit-rotate `seed & 7` |
| `sub_79E7E8` | `0x79E7E8` | keystream byte `= rotatedKey[i % 16]` |
| `sub_79ECDE` | `0x79ECDE` | XOR loop |

Blob layout is `[seed:u8][encoded chars…][NUL]`. The XOR loop is
`k = rotatedKey[i % 16]; plain = (enc == k) ? k : enc ^ k` — the `enc == k`
case is what keeps NULs out of the encoded blob.

### The two entries

| idx | blob addr | seed | decoded |
|---|---|---|---|
| 3366 | `0xB18108` | `0x84` | `has received a gift.` |
| 3367 | `0xB180EC` | `0x85` | `'s fame has gone up +1.` |

(20 and 23 bytes — matching the blob lengths exactly. Neighbouring indices
decode to English UI text as well, so the table base is right.)

### How they are assembled — v83 `sub_64B1A5` (DrawMemo, `0x64B1A5`)

```c
v47 = *(a3 + 6);            // MEMO.nFlag  (CMemoListDlg::MEMO + 0x18)
if ( v47 > 0 && v47 <= 2 )  // nFlag == 1 or 2
{
    v108 = MEMO.bsSender          + StringPool::GetBSTR(3366);   // line 1
    v109 = CWvsContext.name       + StringPool::GetBSTR(3367);   // line 2
    ... DrawTextA(v108) ...
    if ( *(a3 + 6) == 1 )
        ... DrawTextA(v109) ...                                  // line 2 only on nFlag == 1
}
```

The line-2 subject is confirmed to be the **local character's name**, not a
guess: v61 reads the same field as `*(g_pWvsContext + 0x2098)`, and
`CWvsContext::GetCharacterName` (v61 `0x484B82`) is literally
`mov eax, [ecx+2098h]`.

Line 1 renders `"<sender> has received a gift."` with no inserted space — which
is exactly why the clientbound wire writes `senderName + " "`
(`libs/atlas-packet/note/entry.go:26`). Line 2 renders
`"<my name>'s fame has gone up +1."`.

So `nFlag` is: `1` = gift delivered **and** gifter famed +1; `2` = gift
delivered, no fame; `3` (or `2` on v48/v61) = wedding invitation.

### Per-version sweep (all 10 templates that have an IDB)

Probe: the `MEMO.nFlag` (`+0x18`) tests inside each build's DrawMemo.

| Template | DrawMemo | wedding arm | gift-block guard | fame-line guard |
|---|---|---|---|---|
| gms_v48 | `sub_53555E` | `[esi+18h] == 2` `0x535868` | `[edi+18h]; dec eax; jnz` `0x535A58` | same test (both strings in one block) |
| gms_v61 | `sub_5ADC52` | `a3[6] != 2` | `a3[6] == 1` | same test (both strings in one block) |
| gms_v72 | `sub_5FBB91` | `[esi+18h] == 3` `0x5FBE97` | `[ecx+18h]` `0x5FC091` | `[eax+18h] == 1` `0x5FC360` |
| gms_v79 | `sub_61A680` | `0x61A986` | `0x61AB80` | `0x61AE4F` |
| gms_v83 | `sub_64B1A5` | `!= 3` | `>0 && <=2` | `== 1` |
| gms_v84 | `sub_660DEE` | `0x6610F4` | `0x6612EE` | `0x6615BD` |
| gms_v87 | `sub_684F9C` | `0x6852A2` | `0x68549C` | (same +0x7CF offset) |
| gms_v92 | `sub_6185A0` | `[edx+18h] == 3` `0x618A5E` | `dec eax; cmp eax,1; ja` `0x618DC0` | `[edx+18h] == 1` `0x61922B` |
| gms_v95 | `CMemoListDlg::DrawMemo` `0x6247B0` | `!= 3` | `== 1` block | `== 1` |
| jms_v185 | `sub_6C3446` | `[esi+18h] == 3` `0x6C374C` | `0x6C3946` | `[eax+18h] == 1` `0x6C3C15` |

v48 and v61 collapse the two lines into a single `nFlag == 1` block (no
`nFlag == 2` "gift without fame" variant) and use `2` for the wedding arm —
the same enum renumber `discardSpecialFlag` already models. Every other build
matches v83 exactly.

**Not checked:** `gms_v12` — there is no v12 IDB in `IDBs_v9`. The cash-shop
gift flow is not exercised on that template.
