# Review: Task 6 — CharacterAppearanceUpdate frame correction

Range: `940fd0ae4..3c4d49bcc` (commit `3c4d49bcc` "fix(atlas-packet): correct
CharacterAppearanceUpdate frame and drive its ring blocks")

Brief: `.superpowers/sdd/plan/task-6-brief.md`
Report: `.superpowers/sdd/plan/task-6-report.md`

## Scope reviewed

`git diff --stat 940fd0ae4..3c4d49bcc`:

```
libs/atlas-packet/character/clientbound/appearance_update.go      |  25 +-
libs/atlas-packet/character/clientbound/appearance_update_test.go | 372 ++++++---------------
libs/atlas-packet/character/clientbound/v61_test.go               |  16 +-
libs/atlas-packet/character/clientbound/v72_test.go               |  19 +-
4 files changed, 143 insertions(+), 289 deletions(-)
```

`libs/atlas-packet/model/ring.go` confirmed untouched (`git diff --stat` /
`git log` for the range both empty for that path) — the binding constraint
holds.

## Summary verdict

**CHANGES_REQUIRED.** The core mechanical change (wiring `model.RingSet`
through `Encode`/`Decode`, appending `rings` as the last constructor
parameter, adding tenant plumbing) is correct and matches the brief. However,
the brief's central premise — "no export shows the client reading a trailing
int on this codepath... on any version" — is **false for two of the six
versions this same commit targets**, `gms_v87` and `gms_v95`. The checked-in
IDA exports for those versions show `CUserRemote::OnAvatarModified` reading
an **unconditional, unguarded trailing `Decode4` ("nCompletedSetItemID")**
immediately after the marriage arm — a fact a *prior* task had already IDA-
derived and pinned (with instruction addresses) in the very comment blocks
this commit edited. This commit silently overwrites that correct, previously
-verified fact with the incorrect blanket claim, and unconditionally removes
the trailing `WriteInt(0)` for all tenant versions including v87/v95. The
result is a genuine wire-desync regression for any GMS v87/v95 tenant: the
frame is now emitted 4 bytes short of what those clients unconditionally
read, corrupting the client's read cursor for whatever bytes follow on the
socket.

## Findings

### BLOCKING — trailing int removal breaks GMS v87 and v95, contradicting the brief's own cited export and a pre-existing, address-pinned derivation

- `docs/packets/ida-exports/gms_v87.json:3044-3058` — `CUserRemote::
  OnAvatarModified` (`address: "0xa090f4"`) calls array ends with the
  three guarded `bMarriage` `Decode4`s, followed by an **unguarded**
  `{"op": "Decode4", "comment": "nCompletedSetItemID"}` — no `guard` key at
  all, i.e. read unconditionally regardless of `bMarriage`.
- `docs/packets/ida-exports/gms_v95.json:8816-8829` — identical shape:
  unguarded trailing `Decode4` `"nCompletedSetItemID"` after the marriage
  block, `address: "0x954110"`.
- Contrast `docs/packets/ida-exports/gms_v83.json:7810-7829` (v83, the
  brief's cited authority) and `gms_jms_185.json` and `gms_v84.json`, whose
  `calls` arrays for the same function end cleanly at the marriage arm's
  three `Decode4`s with no trailing read — those three versions genuinely
  have no trailing read, so the frame correction is correct *for them*.
- The pre-task state of the test file (`git show 940fd0ae4:libs/atlas-packet/
  character/clientbound/appearance_update_test.go`, lines ~150-157 and
  ~232-234) already documented this exact per-version split, with specific
  instruction addresses derived by an earlier task:
  ```
  //	completedSet   = Decode4         // read UNCONDITIONALLY in v87 (this[1456]) /*0xa092d5*/
  // Unlike the v83/v84 comment ("trailing int unread when marriage==0"), v87 reads
  // the trailing Decode4 (completed-set-item id) unconditionally...
  ```
  and, for v95:
  ```
  //	completedSet   = Decode4         // read UNCONDITIONALLY in v95 (m_nCompletedSetItemID) /*0x9542ec*/
  ```
  The pre-task `want` byte literals for both `TestCharacterAppearanceUpdate
  ByteOutputV87` and `...V95` explicitly included the trailing
  `0x00, 0x00, 0x00, 0x00, // completed set item id (Decode4, read
  unconditionally)` for exactly this reason.
- This commit (`libs/atlas-packet/character/clientbound/
  appearance_update_test.go:156-159` for v87, `183-184` for v95) **deletes**
  that correct, address-cited fact and replaces it with:
  ```
  // There is no trailing Decode4 read after the marriage arm in any exported
  // version (design.md §3.2's "one int long" defect); the encoder no longer
  // writes one.
  ```
  which is directly contradicted by the very files (`gms_v87.json`,
  `gms_v95.json`) the surrounding `packet-audit:verify ... ida=0xa090f4` /
  `ida=0x954110` markers on the same test cite as authority.
- Net effect: `libs/atlas-packet/character/clientbound/appearance_update.go`
  now writes `characterId + flags + avatar + ringBlock` with no trailing int,
  for **every** tenant, including GMS v87/v95 — but those two clients
  unconditionally consume 4 more bytes than the encoder now sends,
  regardless of whether any ring is populated (the `nCompletedSetItemID`
  read has no guard in either export). Every `CharacterAppearanceUpdate`
  sent to a v87 or v95 client will desync that client's packet-read cursor
  by 4 bytes for whatever data follows on the wire.
- v87 and v95 are not synthetic-only fixtures: `libs/atlas-packet/test/
  context.go:20-21` (`pt.Variants[2]`, `pt.Variants[3]`) are real, generally
  -used supported tenant configurations exercised throughout the packet test
  suite, not placeholders.
- This is not resolvable by the "no version gate on the GMS ring arm" rule
  in `model/ring.go` — that rule is specifically about the ring block's own
  internal shape (JMS entry-count wrapping) being version-independent within
  GMS. The `nCompletedSetItemID` trailing read is a *different*, genuinely
  version-dependent field the brief did not account for and this commit's
  own cited evidence refutes for two of its six target versions.
- Disposition: requires either (a) confirming with a maintainer/IDA reviewer
  whether `gms_v87.json`/`gms_v95.json`'s trailing `nCompletedSetItemID` call
  is itself in error (unlikely given a prior task already independently
  derived and address-pinned the same fact), or (b) restoring a genuinely
  version-gated trailing-int write for v87/v95 (via the package's
  `MajorAtLeast`-style idiom) while keeping the correction for v83/v84/jms.
  Either way the current uniform removal is wrong as shipped.

## Non-blocking observations

1. **`appearance_update_test.go`'s "populated" case does not independently
   transcribe the frame's byte layout; it is built via
   `model.RingSet.EncodeField` instead of a hand-derived literal.**
   `libs/atlas-packet/character/clientbound/appearance_update_test.go:78-92`
   constructs `want` by calling the exact same production
   `model.RingSet.EncodeField` that `Encode()` calls internally, using the
   same op sequence (`WriteInt`/`WriteByte`/`WriteByteArray`/`EncodeField`)
   `Encode()` itself uses. This does verify wiring (call order, no stray
   extra/missing write) but not an independent confirmation of the actual
   ring-block byte layout against the brief's literal spec (`01 <8B ownSN>
   <8B partnerSN> <4B third> 00 00`). This mirrors a precedent set by
   `spawn_test.go`'s "couple populated" case (`libs/atlas-packet/character/
   clientbound/spawn_test.go:166-211`), but that precedent is materially
   stronger: it pins the *entire* surrounding frame via a hand-transcribed
   hex literal (`emptyHex`) and only defers to `EncodeField` for the
   contested span, with an explicit length-delta assertion. The new
   `appearance_update_test.go` has **no hand-transcribed hex literal
   anywhere** across all twelve `empty`/`populated` subtests for v83, v84,
   v87, v95, jms, v79 — every byte of `want` is built from the same
   primitives `Encode()` itself uses. Partial mitigation: `v61_test.go` and
   `v72_test.go` (see below) do carry genuine hand-transcribed hex fixtures
   that independently confirm the trailing-int removal for those two legacy
   versions, and `model/ring_test.go` (untouched, out of this task's scope)
   presumably pins `RingSet.EncodeField`'s byte layout independently. Given
   the blocking finding above, I would want the populated/empty cases to at
   minimum hand-transcribe the couple-populated span the way
   `spawn_test.go` does, so a future regression in the wiring itself (not
   just in `RingSet.EncodeField`) is still caught by an assertion that isn't
   trivially tautological with the code under test.

2. **`v61_test.go` / `v72_test.go` edits are legitimate, in-scope, mandatory
   fallout**, not scope creep. Both files call
   `NewCharacterAppearanceUpdate`; the constructor's arity change makes them
   fail to compile without the update
   (`libs/atlas-packet/character/clientbound/v61_test.go:267`,
   `v72_test.go:271` in the pre-image). The diff
   (`git diff 940fd0ae4..3c4d49bcc -- .../v61_test.go .../v72_test.go`)
   correctly adds `model.RingSet{}` as the third argument and removes the
   `0x00, 0x00, 0x00, 0x00 // completed set item id` line from each
   hand-transcribed `want` array — and, unlike the six new subtests above,
   these two retain real hand-transcribed hex fixtures that independently
   confirm the frame is 4 bytes shorter post-fix for GMS v61/v72. Neither
   v61 nor v72 has a pinned "unconditional trailing read" fact analogous to
   v87/v95 in the pre-task comments, so this removal is not contradicted by
   available evidence for these two versions.

3. `docs/packets/ida-exports/gms_v79.json`'s `CUserRemote::OnAvatarModified`
   ends its marriage arm with a guarded `Delegate` to `sub_8C95A5`
   (`calls: null` — IDA did not trace into it), rather than a clean function
   end. This is unresolved/not evaluable from the export alone: it is
   pre-existing content (the v79 "no trailing Decode4" claim in the test
   file predates this task, unchanged by this diff) and is not something
   Task 6 introduced or was asked to re-litigate, but a reviewer cannot
   independently confirm from this export alone that `sub_8C95A5` performs
   no additional wire read for v79. Flagged as not evaluable, not blocking.

## Constraint checks

- Marriage RECORD (`GroomId`/`BrideId`) vs marriage AVATAR block
  (`MarriageCharacterId`/`MarriagePairCharacterId`) — not touched by this
  diff; `model/ring.go` unmodified. N/A.
- Constructor parameter `rings model.RingSet` appended **last** in
  `NewCharacterAppearanceUpdate` — confirmed (`appearance_update.go:23`).
- No raw `> N` version gate introduced in this file — confirmed; no gating
  logic was added here at all (the ring block's internal JMS branching lives
  in `model/ring.go`, untouched).
- `model/ring.go` not modified by this task — confirmed via
  `git diff --stat` / `git log` for the range.

## Build/test evidence

```
$ cd libs/atlas-packet && go build ./... && go test ./character/...
ok  	.../character
ok  	.../character/clientbound
ok  	.../character/clientbound/monsterbook
ok  	.../character/serverbound
ok  	.../character/serverbound/monsterbook
```
`gofmt -l` and `go vet ./character/clientbound/...` clean on the four
touched files. This is a build/test-green result and does not — cannot —
catch the v87/v95 wire-desync finding above, since both the encoder and the
test's `want` construction share the same (now-wrong) assumption.

## Not evaluable

- `services/atlas-channel` non-compilation and the eight pinned evidence
  records / six `packet-audit:verify` markers invalidated by this change are
  pre-authorized per the task instructions (Task 7 and Task 14 respectively)
  and were not evaluated as defects.
- `gms_v79.json`'s ambiguous trailing `Delegate` (see non-blocking #3) is not
  evaluable from the checked-in export alone.

## Return

```text
verdict: CHANGES_REQUIRED
artifact: docs/tasks/task-269-ring-pair-behavior/reviews/task-6.md
scope_confirmed: reviewed the full diff of 940fd0ae4..3c4d49bcc (appearance_update.go, appearance_update_test.go, v61_test.go, v72_test.go) against docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,gms_jms_185,gms_v79}.json and model/ring.go (read-only, confirmed unmodified)
blocking: 1
  - libs/atlas-packet/character/clientbound/appearance_update.go:39 (Encode) / appearance_update_test.go:156-159,183-184 — the trailing WriteInt(0)/Decode4 removal is correct for gms_v83/v84/jms but wrong for gms_v87 and gms_v95: docs/packets/ida-exports/gms_v87.json:3057 and gms_v95.json:8829 show CUserRemote::OnAvatarModified reading an unconditional, unguarded trailing Decode4 ("nCompletedSetItemID") after the marriage arm — a fact a prior task had already IDA-derived and pinned with instruction addresses (v87 @0xa092d5, v95 @0x9542ec) in the pre-task version of this same test file, which this commit silently overwrote with a false blanket "no export shows a trailing read on any version" claim. As shipped, every CharacterAppearanceUpdate frame sent to a v87 or v95 tenant is 4 bytes short of what the client unconditionally reads, desyncing that client's read cursor.
non_blocking: 2
not_evaluable: 1
```

---

## Fix-round re-review (Task 6, round 1)

Range: `cc49a9fb8..00dc42a52` (commit `00dc42a52` "fix(atlas-packet): restore
gms_v87/v95 trailing completed-set int in CharacterAppearanceUpdate")

Diff stat:

```
libs/atlas-packet/character/clientbound/appearance_update.go      |  23 +++++
libs/atlas-packet/character/clientbound/appearance_update_test.go | 106 +++++++++++++-------
2 files changed, 95 insertions(+), 34 deletions(-)
```

### Open finding: ADDRESSED

`libs/atlas-packet/character/clientbound/appearance_update.go` now gates the
trailing `WriteInt(0)`/`ReadUint32()` behind a new
`hasTrailingCompletedSetItemId(t tenant.Model) bool` helper:

```go
func hasTrailingCompletedSetItemId(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(87)
}
```

used identically in both `Encode` (write) and `Decode` (read). Verified
directly against the exports (`python3`-driven `json.load` walk of each
`functions["CUserRemote::OnAvatarModified"]["calls"]` tail, not the report's
word):

- `gms_v87.json` (`address: "0xa090f4"`) and `gms_v95.json` (`address:
  "0x954110"`) — both end their `calls` array with `{"op": "Decode4",
  "comment": "nCompletedSetItemID"}` and **no `guard` key**, i.e.
  unconditional, immediately after the three `guard: "bMarriage"` Decode4s.
  Confirms the finding's original premise.
- `gms_v83.json`, `gms_v84.json`, `gms_jms_185.json` — all end cleanly at the
  three `bMarriage`-guarded `Decode4`s, no trailing call. Confirms the
  correction is right for these three.
- `gms_v92.json` (not one of the six versions this task's tests exercise, but
  in-region and `MajorAtLeast(87)`) also ends with an unguarded trailing
  `Decode4` (`address: "0x930860"`) — additional, independent confirmation
  that the `MajorAtLeast(87)` floor (not a narrower "only 87 and 95" gate)
  is the right shape, not just curve-fit to the two versions named in the
  brief.
- `gms_v79.json` — major version 79 is `< 87`, so `IsRegion("GMS") &&
  MajorAtLeast(87)` excludes it regardless of the ambiguous trailing
  `Delegate` noted in the original review's non-blocking #3; that ambiguity
  is unaffected by this fix and remains not evaluable from the export alone,
  as before.
- `gms_jms_185.json` (`region: "JMS"`, major 185) — correctly **excluded** by
  the `IsRegion("GMS")` clause despite `MajorAtLeast(87)` being true; a
  version-only gate would have wrongly included it. Confirmed no trailing
  read at jms_185 above, so exclusion is correct on both counts (region and
  content).
- No GMS version `>= 87` in the checked-in exports is wrongly excluded by
  this gate (v87, v92, v95 all included and all three show the unconditional
  trailing read).

`hasTrailingCompletedSetItemId` uses the package's existing `tenant.Model.
IsRegion` / `tenant.Model.MajorAtLeast` idiom (`libs/atlas-tenant/tenant.go:
88,93`) — no raw `> N` comparison introduced, consistent with repo
convention and the original review's constraint check.

The pre-existing, address-cited per-version documentation that this task's
first pass had deleted is restored in the test file: `//	completedSet =
Decode4 // read UNCONDITIONALLY in v87 (this[1456]) /*0xa092d5*/` and the
matching v95 comment `/*0x9542ec*/`. These specific instruction addresses
(as opposed to the function-entry addresses) are not present in the
checked-in export JSON's `calls` array entries (confirmed by grep — the
schema only records function-level `address`, not per-call addresses), so
this specific detail remains not independently re-verifiable from the
export alone; it is a restoration of a prior task's directly-IDA-derived
fact, not a new invention by this fix, so this is unchanged from the
situation at the original review and not a new concern.

### `model/ring.go`

`git diff --stat cc49a9fb8..00dc42a52 -- libs/atlas-packet/model/ring.go` and
`git log --oneline cc49a9fb8..00dc42a52 -- libs/atlas-packet/model/ring.go`
both empty — still unmodified.

### `baseFrameHex` — judged as genuinely pinning the frame

`appearance_update_test.go` adds `const baseFrameHex = "78563412...` (34
bytes: 4-byte characterId + 1-byte flags + 29-byte avatar block) and both
`empty`/`populated` subtests now build `want` via `hex.DecodeString
(baseFrameHex)` + `w.WriteByteArray(base)`, replacing the prior
`w.WriteInt(0x12345678); w.WriteByte(1); w.WriteByteArray(avatarBytes)`
sequence where `avatarBytes := avatar.Encode(nil, ctx)(nil)` called the
production avatar encoder at test time. This is a genuine improvement, not a
restatement of the encoder: the test previously re-derived the avatar
portion by calling into the `avatar` package's live `Encode` at every test
run (so a regression in avatar encoding would silently propagate into both
`got` and `want` and never be caught here); the hex literal decouples the
test from that call entirely, matching the `spawn_test.go` "couple
populated" precedent of pinning the surrounding frame as a literal. The
ring-block span is still built via `model.RingSet.EncodeField(w, tn)` in
both `got` and `want` (unchanged from before, and flagged as the original
review's non-blocking #1, which remains true here) — that specific slice is
still tautological with production code, but the frame prefix that
`baseFrameHex` now covers is not. Net: addresses the note as scoped ("cheap
to do... added a pinned baseFrameHex constant"); does not fully eliminate
non-blocking #1's residual concern about the ring span, which was never
claimed to be in scope for this fix round.

### Build/test evidence (independently re-run, not from the report)

```
$ cd libs/atlas-packet && go build ./... && go test ./character/...
ok  	.../character
ok  	.../character/clientbound
ok  	.../character/clientbound/monsterbook
ok  	.../character/serverbound
ok  	.../character/serverbound/monsterbook
```

`gofmt -l` on both touched files: no output (clean). `go vet
./character/clientbound/...`: no output (clean).

### New breakage in the fix diff

None found. The diff is additive-only (23 lines added to
`appearance_update.go`, no deletions; the test file's net change restores
prior documentation and adds the gate/hex-literal plumbing). No other
production file calls `hasTrailingCompletedSetItemId` or otherwise depends
on the removed/restored byte, so no new call-site risk. `model/ring.go` and
the constructor signature are unchanged from the prior round, so no fresh
arity/compile fallout beyond what round 1 already produced (and that
fallout — `services/atlas-channel` non-compilation — is pre-authorized to
Task 7, unchanged and not re-litigated here).

### Verdict

**Open blocking finding: ADDRESSED.** The gate is correctly shaped
(`IsRegion("GMS") && MajorAtLeast(87)`), matches every checked export
(v83/v84/jms/v79 excluded and correctly so; v87/v92/v95 included and
correctly so), restores the previously-deleted address-pinned
documentation, and is exercised by both `Encode` and `Decode`. The
`baseFrameHex` addition is a genuine, non-tautological improvement over the
prior live-`avatar.Encode()` derivation for the portion of the frame it
covers. No new defects introduced by this fix diff.

## Return (fix-round re-review)

```text
verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-269-ring-pair-behavior/reviews/task-6.md
scope_confirmed: reviewed the full fix diff cc49a9fb8..00dc42a52 (appearance_update.go, appearance_update_test.go) against docs/packets/ida-exports/{gms_v79,gms_v83,gms_v84,gms_v87,gms_v92,gms_v95,gms_jms_185}.json and libs/atlas-tenant/tenant.go; confirmed model/ring.go still unmodified for this range
blocking: 0
non_blocking: 1
  - libs/atlas-packet/character/clientbound/appearance_update_test.go — the ring-block span in both empty/populated subtests is still built via model.RingSet.EncodeField(w, tn) on both the got and want sides (unchanged from the original review's non-blocking #1); a future regression in the wiring itself, not just in RingSet.EncodeField, would still not be caught by an assertion that isn't tautological with the code under test. Not part of this fix round's scope and not blocking.
not_evaluable: 1
  - the exact instruction addresses cited in the restored comments (/*0xa092d5*/ for v87, /*0x9542ec*/ for v95) are not present in the checked-in export JSON's calls array entries (only function-level addresses are recorded in that schema); unchanged from the original review, this is a restoration of a prior task's directly-IDA-derived fact rather than a new claim by this fix.
