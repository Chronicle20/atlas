# Fix report: PARCEL message is 4 bytes early — +29 is a hasMessage flag, not the message

## What was implemented

Exactly the fix described in `bug-duey-parcel-message-offset.md`:

- `libs/atlas-packet/parcel/parcel.go`
  - `parcelMessageWidth` changed from `205` to `201`.
  - `Encode` now writes a 4-byte little-endian `hasMessage` flag
    (`w.WriteInt(uint32(1))` when `p.message != ""`, else `w.WriteInt(uint32(0))`)
    immediately after the 8-byte expiry, before the fixed message buffer.
    Total block size is unchanged at 234 bytes (4 + 13 + 4 + 8 + 4 + 201).
  - Replaced the `+29..233 message + padding (205 bytes)` struct-comment
    block with two real fields, `+29 uint32 hasMessage (flag)` and
    `+33 char[201] message`, citing `CTabReceive::Draw` @0x6EFA1F, the row
    marker draw @0x6EFF31/@0x6EFF78, the 32-bit flag compare @0x6F07AB
    (`cmp [eax+1Dh], edi`), and the `+33` note read via
    `ZXString<char>::GetBuffer` @0x6F0801 / `sub_6F5D37` @0x6F080C. The old
    "NOT independently decompile-confirmed" caveat is removed since this
    span is now IDA-confirmed. Also noted (per the bug file) that the flag
    must track "message is non-empty," not "a message field exists,"
    because when the flag is set but +33 is empty the client substitutes
    StringPool 3886 (@0x6F07C1).

- Test fixtures — every place that built the 234-byte block by hand with
  `msg := make([]byte, 205)` was changed to `make([]byte, 201)`, and every
  `want`/`pBytes` byte-slice construction now inserts
  `0x01, 0x00, 0x00, 0x00` (the hasMessage flag, LE) immediately after the
  `filetime[:]` bytes and before `msg`. All fixtures in this package use
  message `"hi"` (non-empty), so the flag is always `1` — no fixture needed
  a `0` flag case.
  - `libs/atlas-packet/parcel/clientbound/parcel_test.go` (also updated the
    doc-comment wire-shape description from `message/padding[205]` to
    `hasMessage flag[4] + message[201]`)
  - `libs/atlas-packet/parcel/clientbound/v72_test.go`
  - `libs/atlas-packet/parcel/clientbound/v79_test.go`
  - `libs/atlas-packet/parcel/clientbound/v83_test.go`
  - `libs/atlas-packet/parcel/clientbound/v84_test.go`
  - `libs/atlas-packet/parcel/clientbound/v87_test.go`
  - `libs/atlas-packet/parcel/clientbound/v92_test.go`
  - `libs/atlas-packet/parcel/clientbound/v95_test.go`
  - `libs/atlas-packet/parcel/clientbound/v185_test.go`

  Note: the brief's Files section names `libs/atlas-packet/parcel/parcel_test.go`;
  the actual fixture file that builds the 234-byte block is
  `libs/atlas-packet/parcel/clientbound/parcel_test.go` (there is no
  `parcel_test.go` directly under `parcel/`). Updated the file that actually
  exists and contains the described fixtures.

- `docs/packets/audits/STATUS.md` / `docs/packets/audits/status.json` —
  regenerated with `go run ./tools/packet-audit matrix` after the fixture
  edits. The regeneration produced no diff (the matrix does not track
  message-field offsets), so there is nothing to commit for these two files.
  `go run ./tools/packet-audit matrix --check` exits 0.

## Testing

```
cd libs/atlas-packet && go build ./... && go test ./...
```
All packages `ok`, including `parcel/clientbound 0.008s`. No failures, no skips.

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
All packages `ok`, including `atlas-channel/parcel 0.009s` and
`atlas-channel/socket/handler 0.884s`. No failures.

```
go run ./tools/packet-audit matrix --check
```
Exit 0 (only informational `note` lines for unrelated n-a evidence entries).

## Files changed

- `libs/atlas-packet/parcel/parcel.go`
- `libs/atlas-packet/parcel/clientbound/parcel_test.go`
- `libs/atlas-packet/parcel/clientbound/v72_test.go`
- `libs/atlas-packet/parcel/clientbound/v79_test.go`
- `libs/atlas-packet/parcel/clientbound/v83_test.go`
- `libs/atlas-packet/parcel/clientbound/v84_test.go`
- `libs/atlas-packet/parcel/clientbound/v87_test.go`
- `libs/atlas-packet/parcel/clientbound/v92_test.go`
- `libs/atlas-packet/parcel/clientbound/v95_test.go`
- `libs/atlas-packet/parcel/clientbound/v185_test.go`
- `docs/tasks/task-241-duey-parcel-delivery/bug-duey-parcel-message-offset.md`
  (added — was present but untracked in the worktree; committed as part of
  this fix so the diagnosis lands with the code that fixes it)

Commit: `b21783437 fix(duey): encode PARCEL's message as a hasMessage flag + text, not raw padding`

## Self-review

- Verified the total block size is still 234 bytes (4+13+4+8+4+201) in both
  the doc comment and the `Encode` implementation.
- Verified every fixture that had `msg := make([]byte, 205)` was updated to
  201, paired with a flag insertion at the correct position (right after
  `filetime[:]`, right before `msg`) — checked via `grep -rn` before and
  after editing, all eight version files plus the shared `parcel_test.go`
  used identical variable names (`pBytes`/`want`, `msg`, `filetime`), so the
  same mechanical edit was correct everywhere.
- Confirmed `w.WriteInt` takes `uint32`, matching the existing idiom used for
  `p.id`/`p.mesos` in the same `Encode` function.
- No new domain type/constant was introduced beyond the local
  `parcelMessageWidth`; the flag is written inline with the existing
  `WriteInt` idiom rather than a new named constant, matching how the
  neighboring `hasItem` bool is written.
- Doc comment for the struct now cites concrete addresses (@0x6EFA1F,
  @0x6EFF31, @0x6EFF78, @0x6F07AB, @0x6F0801, @0x6F080C) rather than a
  design-level inference, matching the brief's instruction to remove the "NOT
  independently decompile-confirmed" caveat.

## Issues or concerns

- None. The one deviation from the brief's literal file list
  (`libs/atlas-packet/parcel/parcel_test.go` vs. the actual
  `libs/atlas-packet/parcel/clientbound/parcel_test.go`) is noted above; the
  file described by the brief (the one with the `msg := make([]byte, 205)`
  fixtures and 234-byte block wire-shape comment) was found and updated.
