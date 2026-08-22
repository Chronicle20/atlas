# Review: bug-duey-receive-list-item-slot-desync (fix commit)

Range reviewed: `54a968ccb..b5efb76e5` (single fix commit `b5efb76e5`, "fix(atlas-packet):
drop the phantom slot byte from parcel-attached items"). `bcf3a26eb` is docs-only per
instruction and was not reviewed.

Requirement doc: `docs/tasks/task-241-duey-parcel-delivery/bug-duey-receive-list-item-slot-desync.md`

## Scope

`git diff --stat 54a968ccb..b5efb76e5` — 14 files, 83 insertions / 55 deletions:

- `libs/atlas-packet/model/asset.go` (+8)
- `libs/atlas-packet/parcel/parcel.go` (+19/-2 net)
- `services/atlas-channel/atlas.com/channel/parcel/model.go` (+1/-1)
- 9 fixture files under `libs/atlas-packet/parcel/clientbound/*_test.go`
- `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

This matches the "Fix" checklist in the bug file item-for-item. No unrelated files
touched.

## Findings

### 1. `Asset.SetZeroPosition` — additive setter, no other encode path altered — PASS

`libs/atlas-packet/model/asset.go:197-202` adds:

```go
func (m Asset) SetZeroPosition(v bool) Asset {
	m.zeroPosition = v
	return m
}
```

This mirrors every other `Set*` method in the file (immutable value receiver, single
field write). `zeroPosition` was an existing field (`asset.go:20`) already gating the
slot-byte write at `asset.go:370`, `472`, `504`, `520` — the new setter does not touch
any of those gate sites; it only lets a caller flip the flag before `Encode`.

Confirmed no other caller of `zeroPosition` semantics changed: `grep` for `NewAsset(`
across the repo (excluding tests) turns up 19 non-test call sites (inventory, storage,
cashshop, merchant, mts, socket snapshot, character data, interaction/trade, etc.).
None of those files appear in the diff. `grep` for `.SetItem(` (the only call site that
now forces `zeroPosition = true`) turns up exactly one non-test caller:
`services/atlas-channel/atlas.com/channel/parcel/model.go:101`. So the normalization
introduced by this fix is reachable only through the parcel codec path — no inventory,
storage, cashshop, drop, or trade encode path can be affected.

Ran `go test ./...` in `libs/atlas-packet` (all packages, not just parcel) — all pass,
including `inventory/clientbound`, which is the other consumer of `Asset.Encode`/slot
byte logic. This is consistent with (does not merely assert, but empirically confirms)
that no other wire form shifted.

### 2. `Parcel.SetItem` normalization — matches IDA-derived contract — PASS

`libs/atlas-packet/parcel/parcel.go:150-154`:

```go
func (p Parcel) SetItem(a model.Asset) Parcel {
	a = a.SetZeroPosition(true)
	p.item = &a
	return p
}
```

This enforces zero-position at the codec boundary regardless of what the caller
constructed, which is the correct place per the bug file's diagnosis (`GW_ItemSlotBase::Decode
@0x4E33F9` reads the item TYPE byte first with no slot prefix). The `+235..` field
doc comment above the `const` block (parcel.go:99-109) was updated to record the IDA
address and reasoning, matching the "Fix" checklist requirement.

`libs/atlas-packet/parcel/clientbound/parcel_test.go` deliberately passes
`model.NewAsset(false, 1, 1302000, time.Time{})` (non-zero slot) into `SetItem` and then
re-reads via `p.Item()` before encoding (`parcel_test.go:60-68`), which is a real
regression test for the enforcement — it would fail if `SetItem` stopped normalizing.
This is a meaningfully "honest" test, not a token in disguise.

### 3. `atlas-channel` call site — PASS

`services/atlas-channel/atlas.com/channel/parcel/model.go:97`:

```go
item := packetmodel.NewAsset(true, 0, *m.ItemId(), time.Time{})
```

Changed from `false, 0` to `true, 0`, matching the bug file's prescribed one-line fix.
Because `Parcel.SetItem` (finding #2) normalizes regardless, this call-site change is
belt-and-suspenders, not load-bearing by itself — but it is exactly what the bug file
asked for and does no harm.

One observation, non-blocking: there is no new/updated test in
`services/atlas-channel/atlas.com/channel/parcel/model_test.go` asserting
`ToPacket().Item()...ZeroPosition() == true` or the corresponding wire bytes at the
atlas-channel layer. `model_test.go` only has `TestToPacketExpiresAt`. Given the codec
enforces the invariant structurally (finding #2) this is low risk, but it means the
`atlas-channel`-layer assertion added by this commit is not independently pinned by a
test at that layer — only by the codec-level fixtures. Noted under Non-blocking below.

### 4. Nine parcel fixture edits — exactly the leading slot byte removed — PASS

Diffed all nine `libs/atlas-packet/parcel/clientbound/v*_test.go` + `parcel_test.go`
files individually. In every case the only functional change to the expected-byte
builder is the removal of one `append` line:

- v72, v79: `b = append(b, 0x01)` (1-byte slot, sub-83 versions) removed.
- v83, v84, v87, v92, v95, v185: `b = append(b, 0x01, 0x00)` (2-byte short slot,
  MajorAtLeast(83) versions) removed.

No other byte, offset, or field in any fixture builder changed. The `item :=
model.NewAsset(false, 1, ...)` construction calls in the `TestParcel*WithItem` test
bodies (as opposed to the `wantXxxBytes` doc/builder functions) were updated to
`model.NewAsset(true, 0, ...)`, and the header doc comments citing
`model.NewAsset(false, 1, ...)` were updated to `model.NewAsset(true, 0, ...)` with
byte-count corrections (e.g. v72 "77 bytes" → "76 bytes", v79/v83 "81/82 bytes" → "80
bytes each") — consistent with one fewer byte (v72/v79, 1-byte slot removed) or two
fewer bytes (v83+, short slot removed).

This matches the bug file's IDA-confirmed layout: `GW_ItemSlotBase::Decode @0x4E33F9`
reads the item TYPE byte first, with no slot prefix, for every parcel-attached asset
regardless of client version's inventory-context slot width. Removing the leading slot
byte (1 byte pre-83, 2 bytes 83+) from every fixture, and nothing else, is exactly
correct.

`go test ./parcel/...` in `libs/atlas-packet` passes (all 9 version fixtures +
`parcel_test.go`).

### 5. `packet-audit matrix --check` regen — confined to tool hash — PASS

`git diff 54a968ccb..b5efb76e5 -- docs/packets/audits/STATUS.md
docs/packets/audits/status.json` shows the only change in both files is the `Tool:` /
`toolSha` line (source hash of the audit tool itself changed because the parcel
fixtures it scans changed). No `exportHashes`, no per-opcode row, no status glyph in
either file changed — i.e., the regen is confined even more narrowly than "parcel rows
only": no visible-content row changed at all, only the tool fingerprint.

Ran `go run ./tools/packet-audit matrix --check` against the worktree post-checkout:
exits 0 (two informational `note` lines about pre-existing n-a evidence, unrelated to
this diff). Confirms the committed regen is current and not stale.

## Verification run

- `cd libs/atlas-packet && go build ./... && go test ./...` — all packages pass.
- `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./parcel/...`
  — pass.
- `go run ./tools/packet-audit matrix --check` — exit 0.

## Not evaluable

None. The full review surface (codec, model, one call site, nine fixtures, two audit
docs) was covered by direct diff read plus targeted build/test runs; no file the diff
touches was left unexamined.

## Non-blocking

- No atlas-channel-layer test pins `ToPacket()`'s asset as zero-position or asserts the
  resulting wire bytes independently of the codec-level fixtures (see finding #3). Since
  `Parcel.SetItem` enforces the invariant structurally regardless of caller input, this
  is low risk, not a defect — but a future regression at the call site (e.g. someone
  reverting `model.go:97` back to `false`) would only be caught by the codec fixtures,
  not by an atlas-channel-local test. Consider for a follow-up, not blocking this fix.

## Verdict rationale

Every requirement in the bug file's "Fix" section is implemented, in the prescribed
location, with the prescribed semantics. The setter is additive and provably does not
touch any other encode path (grep sweep + full `libs/atlas-packet` test suite green).
The normalization point (`Parcel.SetItem`) is the correct architectural boundary per
the IDA evidence in the bug file. All nine fixture edits are exactly the leading slot
byte and nothing else, cross-checked byte-count-by-byte-count against the updated doc
comments. The STATUS.md/status.json delta is confined to the tool hash — no other
opcode row shifted. `packet-audit matrix --check` is green.
