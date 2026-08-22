# bug: PARCEL message is 4 bytes early — +29 is a hasMessage flag, not the message

Follow-on to `bug-duey-receive-list-item-slot-desync.md`, which explicitly
deferred this: "If, after this fix, rows 2 and 3 still show empty Message, that
is a SEPARATE bug — the message field offset/encoding — and needs its own
investigation." They do. This is that investigation.

## Reproduced

Same environment and data as the slot-desync bug: ephemeral env
`atlas-pr-1434`, GMS v83 client, Duey NPC 9010009 → Receive tab, three pending
parcels from sender `Atlas` with messages `""`, `"lulnub"`, `"lulul"`.

## Observed (client, after the slot-desync fix)

Fixed by the previous commit: expiry, sender, mesos, and the attached item
(icon + quantity) now render correctly on all three rows — the block no longer
desyncs.

Still wrong:

| # | message sent | row "Name" column | note field |
|---|---|---|---|
| 1 | `""`       | `Atlas`                   | (empty) — correct |
| 2 | `"lulnub"` | `Atlas` + random ASCII    | `ub` |
| 3 | `"lulul"`  | `Atlas` + random ASCII    | `l`  |

The note text is the message with its **first 4 bytes cut off**, on both
samples. The "random ASCII" beside the name appears on exactly the two rows
that carry a message.

## Expected

Rows 2 and 3 show their full messages (`lulnub`, `lulul`), and the row's
note-marker column renders the client's own marker string rather than being
driven by message bytes.

## Root cause — established, IDA-confirmed

GMS v83 IDB (session `754107bf`, `MapleStory_dump.exe.i64`). The renderer is
`CTabReceive::Draw` @0x6EFA1F — it is the ONLY consumer that reads the tail of
PARCEL's 234-byte block, which is why the previous investigation (which looked
only at `CTabReceive::SetParcel` @0x6EF69C and `CParcelDlg::OnPacket`
@0x6F56EA) found no field-by-field reader and had to model +29..233 as a
design-level inference.

`CTabReceive::Draw` reads the block as:

- **Row list**, per visible row (loop @0x6EFD70..0x6F019B), three columns:
  - x=13: `_bstr_t::_bstr_t(&v150, parcel + 4)` @0x6EFED5 — the sender name,
    a NUL-terminated C string at +4. Confirms senderName[13] at +4.
  - x=113: `if ( *(parcel + 29) )` @0x6EFF31 → `push offset dword_AFCDB0;
    call _bstr_t::_bstr_t` @0x6EFF78 — a **static** string constant, drawn
    only when the +29 field is non-zero. `dword_AFCDB0` holds the bytes
    `C4 FC B9 E8 BC DB 00` — EUC-KR Korean (a three-syllable word), which a
    non-Korean client renders as mojibake. **That is the "random ASCII" next
    to the name**: it is the client's own note-present marker, drawn because
    our message's first byte (`'l'`, 0x6C) currently lands at +29.
  - x=188: the +21 expiry countdown (unchanged, already correct).
- **Detail pane for the selected parcel** (@0x6F07AB), verbatim disassembly:

  ```
  6f07ab  cmp  [eax+1Dh], edi        ; edi == 0 — 32-BIT compare at +29
  6f07ae  jz   short loc_6F0816      ; no note -> clear the note control
  6f07b0  add  eax, 21h              ; +33
  6f07b3  push eax                   ; Str
  6f07b4  call _strlen
  6f07bc  jnz  short loc_6F07EB      ; non-empty -> use it as-is
  6f07c1  push 0F2Eh                 ; StringPool 3886 -> strcpy into +33
  ```

  then `ZXString<char>::GetBuffer(..., parcel + 33, -1)` @0x6F0801 and
  `sub_6F5D37(dlg, buf)` @0x6F080C set the note text control from the
  NUL-terminated string at **+33**.

So the real layout of the tail is:

```
+29  int32  hasMessage   cmp [eax+1Dh], edi is a 32-bit compare (@0x6F07AB);
                          gates both the row marker and the note control
+33  char[201] message   NUL-terminated ASCII; 234-33 = 201
```

Our encoder (`libs/atlas-packet/parcel/parcel.go`, `Encode`) writes
`writeFixedAscii(w, p.message, 205)` starting at +29 with no flag field.
Verified by dumping the real bytes for `NewParcel(7,"Atlas",5000,<future>,
"lulnub")`:

```
00000010  00 88 13 00 00 00 8c 84  c6 f3 32 dd 01 6c 75 6c  |..........2..lul|
00000020  6e 75 62 00 ...                                    |nub...          |
```

`6c` ('l') sits at +29 — the flag slot — and the client therefore reads the
note from +33, i.e. `"ub"`. `"lulul"` likewise yields `"l"`. Both observed
samples match exactly. Row 1's empty message writes 0 at +29, so its flag is
correctly clear and its note correctly empty — which is why row 1 looked fine
and gave no signal.

The total stays 234 either way: 4 + 13 + 4 + 8 + **4 + 201** = 234.

## Fix

- `libs/atlas-packet/parcel/parcel.go`
  - Replace `parcelMessageWidth = 205` with a `+29` int32 flag plus
    `parcelMessageWidth = 201`.
  - `Encode`: after the 8-byte expiry, write `uint32(1)` when `p.message != ""`
    and `uint32(0)` otherwise, then `writeFixedAscii(w, p.message, 201)`.
    Use the existing writer idiom for a 4-byte little-endian int (`w.WriteInt`,
    as used for `p.id`/`p.mesos`).
  - Replace the `+29..233 message + padding (205 bytes)` doc block on the
    `Parcel` struct comment with the two real fields, citing
    `CTabReceive::Draw` @0x6EFA1F, the marker draw @0x6EFF31/@0x6EFF78, the
    32-bit flag compare @0x6F07AB (`cmp [eax+1Dh], edi`), and the `+33` note
    read @0x6F0801. This span is no longer a design-level inference — remove
    the "NOT independently decompile-confirmed" caveat and say what the
    evidence is.
  - Note in the comment that when the flag is set but +33 is empty the client
    substitutes StringPool 3886 (@0x6F07C1), so the flag must track "message
    is non-empty", not "a message field exists".
- `libs/atlas-packet/parcel/clientbound/v72_test.go`, `v79_test.go`,
  `v83_test.go`, `v84_test.go`, `v87_test.go`, `v92_test.go`, `v95_test.go`,
  `v185_test.go`, and `libs/atlas-packet/parcel/parcel_test.go` — every fixture
  that builds the 234-byte block with `msg := make([]byte, 205)` must become a
  4-byte flag (matching the fixture's message) plus `make([]byte, 201)`. Update
  the accompanying comments.
- `docs/packets/audits/STATUS.md` and `docs/packets/audits/status.json` —
  regenerate with `go run ./tools/packet-audit matrix` AFTER the fixture edits,
  and commit.

Verification: module-local `go build ./... && go test ./...` in
`libs/atlas-packet` and `services/atlas-channel`, plus
`go run ./tools/packet-audit matrix --check` exiting 0.

## Not yet answered

- **Other versions.** The 234-byte block and the +29/+33 split are read from
  the v83 IDB. `Parcel` is a single shared struct with no version gates, and
  the previous investigation established the 234-byte `DecodeBuffer` shape
  holds across the family, so the fix applies uniformly. If a later version
  turns out to move the boundary, that is a separate gate, not a change here.
- **`itemType` is always 0** (`atlas-parcel` never calls
  `Builder.SetItemType`) — still out of scope, carried over from
  `bug-duey-receive-list-item-slot-desync.md`.

## Outcome

_(to be filled in by the fix)_
