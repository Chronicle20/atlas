# Review: JMS 185 KEYMAP entry-count fix (round 7)

**Range reviewed:** `a7c8768be..550ec76d0` (single commit `550ec76d0`)
**Files touched:** `libs/atlas-packet/character/clientbound/keymap.go`, `libs/atlas-packet/character/clientbound/keymap_test.go`
**Requirement:** `docs/tasks/task-273-jms185-channel-enter-crash/bug-jms185-channel-enter-crash.md`, "## Round 7" section

## Scope confirmed

The diff matches the "Fix" prescribed at the end of Round 7 exactly: a JMS arm
added to `keyMapEntryCount` returning 94, plus a byte-length-pinning test.
Nothing else in the diff. No scope drift.

## Findings

### 1. PASS — JMS arm changes only JMS-region tenants; GMS arms byte-identical

`libs/atlas-packet/character/clientbound/keymap.go:30-39`:

```go
func keyMapEntryCount(ctx context.Context) int32 {
	t := tenant.MustFromContext(ctx)
	if t.Region() == "GMS" && t.MajorVersion() < 83 {
		return 89
	}
	if t.Region() == "JMS" {
		return 94
	}
	return 90
}
```

The new `if t.Region() == "JMS"` block is inserted between the pre-existing
GMS<83 arm and the pre-existing `return 90` default — it does not modify
either existing line, and `GMS` vs `JMS` are mutually exclusive region
strings, so insertion order is immaterial. Confirmed empirically: the
pre-existing GMS fixture tests (`TestCharacterKeyMapByteOutputV61`,
`TestCharacterKeyMapByteOutputV72`, `TestCharacterKeyMapByteOutputV79`,
`character/clientbound/v61_test.go:542`, `v72_test.go:609`,
`v79_test.go:634` — none touched by this diff) and the full
`TestCharacterKeyMap`/`TestCharacterKeyMapResetToDefault` GMS-variant sweep
all pass unmodified against the new code:

```
--- PASS: TestCharacterKeyMap/GMS_v72 (0.00s)
--- PASS: TestCharacterKeyMap/GMS_v79 (0.00s)
--- PASS: TestCharacterKeyMap/GMS_v92 (0.00s)
--- PASS: TestCharacterKeyMapByteOutputV61 (0.00s)
--- PASS: TestCharacterKeyMapByteOutputV72 (0.00s)
--- PASS: TestCharacterKeyMapByteOutputV79 (0.00s)
```

Both `Encode` (`keymap.go:71`) and `Decode` (`keymap.go:92`) call the same
`keyMapEntryCount(ctx)` helper, so this is the only place the count can
diverge — verified.

### 2. PASS — gate idiom is consistent with repo convention; region-only is correct here

The task brief asked to judge whether a bare `t.Region() == "JMS"` (no
`MajorAtLeast`) is acceptable given `jms_v185` is the only JMS template.
Checked the cited precedent, `libs/atlas-packet/reactor/serverbound/hit.go:45`:

```go
if (t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.Region() == "JMS" {
```

`hit.go` uses exactly the same pattern — a version-gated GMS arm alongside a
bare, version-unqualified JMS arm — for the identical reason: there is
currently only one JMS variant registered in
`libs/atlas-packet/test/context.go:23` (`{Name: "JMS v185", Region: "JMS",
MajorVersion: 185, ...}`, confirmed by grep — no other `JMS` entry exists in
that file). A `MajorAtLeast` bound would be indistinguishable from a bare
region check today and would only matter once a second JMS template lands,
at which point the existing precedent (`hit.go`) is the place this will need
to be revisited too — not a defect unique to this commit. The idiom used
(`t.Region() == "JMS"`, not `t.MajorVersion() > 185` or similar) matches the
repo convention of gating on symbolic region checks, not raw magic-number
version comparisons. No violation.

### 3. PASS — new test pins the wire length, not just a round-trip

`libs/atlas-packet/character/clientbound/keymap_test.go:19-35`
(`TestCharacterKeyMapJMS185ByteLength`) asserts `len(got) != 471` before
doing the round-trip. Verified this is load-bearing, not decorative, by
reverting only `keymap.go` to its pre-fix (`a7c8768be`) state and re-running
this test against the new test file:

```
=== RUN   TestCharacterKeyMapJMS185ByteLength
    keymap_test.go:31: jms_v185 keymap encode: got 451 bytes, want 471 (1 flag byte + 94*5 entries)
--- FAIL: TestCharacterKeyMapJMS185ByteLength (0.00s)
```

and passing again once `keymap.go` is restored to `550ec76d0`. The test
would have caught the exact bug this commit fixes (451 vs. 471 = the
documented 20-byte shortfall). This is a genuine regression pin, not a test
that passes either way.

Repo state note: restoring `keymap.go` to the pre-fix commit for this check
inadvertently triggered `git stash pop` against an unrelated pre-existing
stash entry (`task-263` WIP), producing conflicts in ~18 unrelated files.
These were resolved by `git checkout HEAD -- <paths>` and the pre-existing
stash was left untouched and intact (`git stash list` confirms it is still
present as `stash@{0}`, unmodified). `git status` and `git diff --stat`
against `HEAD` are now clean; `go test` at `HEAD` (`550ec76d0`) passes. This
was reviewer tooling friction, not a defect in the commit under review, but
is recorded here for transparency.

### 4. PASS — Decode stays symmetric with Encode for JMS

`Decode` (`keymap.go:91-108`) calls `count := keyMapEntryCount(ctx)` at
`keymap.go:92`, the same helper `Encode` uses at `keymap.go:71`, and loops
`for i := int32(0); i < count; i++` reading one `ReadInt8`/`ReadInt32` pair
per entry — matching `Encode`'s `WriteInt8`/`WriteInt32` pair per entry
exactly. `TestCharacterKeyMapJMS185ByteLength` calls
`test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)` at line 34, which
passed. No asymmetry.

## Not evaluable

- **Live client verification.** The bug doc's own "Resolution (round 7)"
  section states: "Live re-test on the JMS 185 client is still outstanding;
  the crash is not confirmed gone until a channel enter completes without
  the AV at 0x44F5E2." This is explicitly out of reach for a code review —
  it requires a live JMS 185 client session — and is called out by name in
  the source document itself rather than silently assumed passing.
- **Whether other JMS-185 divergences remain later in the burst** (0x7A,
  0x3E, 0x1D) — the bug doc's own "Not yet answered (round 7)" section
  defers this to a post-fix live re-test; it is out of scope for this
  keymap-only diff.

## Verdict rationale

All four focus questions resolve to PASS with direct evidence (file:line,
and for #1 and #3, executed test output). No blocking defects found in the
diff itself. The only open item — live-client confirmation — is explicitly
flagged as outstanding by the author in the same document, not concealed,
so it is recorded as not-evaluable rather than a blocking finding against
this commit.
