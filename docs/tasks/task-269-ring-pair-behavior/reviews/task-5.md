# Review: Task 5 — Wire sites B and D (`CharacterSpawn`, `CharacterInfo`)

Range: `eddb64bb2..940fd0ae4`
Brief: `.superpowers/sdd/plan/task-5-brief.md`
Report: `.superpowers/sdd/plan/task-5-report.md`

## Scope

`git diff --stat eddb64bb2..940fd0ae4` shows exactly 8 files, all in
`libs/atlas-packet/character/clientbound/`:

- `spawn.go`, `spawn_test.go`, `info.go`, `info_test.go` — the brief's file
  inventory.
- `v61_test.go`, `v72_test.go`, `v79_test.go`, `version_bounds_test.go` — not
  in the brief's inventory, flagged by the implementer as call-site fixups.

No other files touched. `libs/atlas-packet/model/ring.go` (Task 2/3's codec,
the binding authority for this task) is unmodified — confirmed by `git diff
--stat` showing no `model/` entry.

## Findings

### 1. Signature changes — PASS

`spawn.go`: `rings model.RingSet` appended as the last parameter of
`NewCharacterSpawn` (`spawn.go:48-57` region, confirmed in diff), struct field
added, `Encode`/`Decode` mirror `m.rings.EncodeField(w, t)` /
`m.rings.DecodeField(r, t)` replacing the three `WriteByte(0)`/`ReadByte()`
lines, `t` already in scope as the brief predicted. `Rings() model.RingSet`
getter added beside the other getters (`spawn.go:219`).

`info.go`: `hasMarriageRing bool` appended as the last parameter of
`NewCharacterInfo`, struct field added, `Encode`/`Decode` mirror
`w.WriteBool(m.hasMarriageRing)` / `m.hasMarriageRing = r.ReadBool()`
replacing the `WriteBool(false)` / `_ = r.ReadBool()` lines.
`HasMarriageRing() bool` getter added beside the other getters
(`info.go:213`).

Both new parameters are last, matching the binding constraint that Task 7's
call-site update at atlas-channel be purely additive.

Version gate at `spawn.go:181` (`t.Region() == "GMS" && t.MajorVersion() >=
61 && t.MajorVersion() < 95`) and `info.go:90` (`t.Region() == "GMS" &&
t.MajorVersion() > 28 && t.MajorVersion() < 61`) are both pre-existing,
untouched by this diff, and already use the repo's comparison idiom — no new
raw `> N` gate was introduced by this task.

### 2. Test honesty — PASS (verified by fault injection)

Ran the module's test suite after manually reverting just the `Encode` line
in each file (leaving everything else — struct field, constructor, getters —
intact) to confirm the new tests catch a real regression, not merely a
compile-time signature mismatch:

- `spawn.go`: replaced `m.rings.EncodeField(w, t)` with three
  `WriteByte(0)` calls → `TestCharacterSpawnRingBlocks/couple_populated`
  fails: `got 123 want 143 (empty 123 + 20)`. The `empty_is_unchanged`
  sub-tests still pass (correctly — that case does not distinguish rings-off
  from the reverted encoder).
- `info.go`: replaced `w.WriteBool(m.hasMarriageRing)` with
  `w.WriteBool(false)` → `TestCharacterInfoMarriageFlag/true,_modern` fails:
  `true marriage-flag byte: got 0x0 want 0x01` and the "differs in exactly
  one byte" assertion reports 0 diffs.

Both reverts were restored from `/tmp` backups immediately after; `git
status --porcelain` on the worktree shows no residual change to `spawn.go`
or `info.go`. Test files genuinely pin the new behavior, not just the new
signature.

### 3. +20 vs the brief's +18 — brief was wrong, implementer's correction is verified

The brief's table says "total length grows by exactly 18" for a
couple-populated spawn. The shipped codec (`model/ring.go`, `encodePair`)
writes, per populated arm: flag(1) + OwnSN(8) + PartnerSN(8) + ItemId(4) = 21
bytes, replacing a single 1-byte flag (not the full 3-byte three-flag span).
21 − 1 = 20. The friendship and marriage flag bytes are untouched and still
follow immediately. `TestCharacterSpawnRingBlocks/couple_populated`
(`spawn_test.go`) asserts `len(got) == len(empty)+20`, cross-checks the
prefix, the shifted suffix, and diffs the replaced span directly against
`model.RingSet{Couple: couple}.EncodeField(...)` output (Task 2/3's own
codec, not a hand-derived literal). Ran this test directly — passes; and (per
§2 above) fails under fault injection. The implementer's arithmetic is
correct and the deviation from the brief's stated number is the right call,
not a defect.

### 4. GMS v28 → GMS v48 fixture substitution — verified, correct

`info.go:90`: `if t.Region() == "GMS" && t.MajorVersion() > 28 &&
t.MajorVersion() < 61`. Read directly — v28 does not satisfy `> 28`, so
GMS v28 falls into the *modern* write path (the one with
`w.WriteBool(m.hasMarriageRing)`), not the legacy no-marriage-byte path the
brief's FR-8 guard is meant to pin. GMS v48 (48 > 28 && 48 < 61) is inside
the legacy arm and is already the module's canonical legacy fixture,
byte-for-byte pinned by the pre-existing `TestCharacterInfoV48Golden`.
`TestCharacterInfoMarriageFlag/legacy_arm_untouched` reuses that same input
with `hasMarriageRing=true` and asserts byte-identical output to the v48
golden. This is the correct FR-8 fixture; using literal GMS v28 as named in
the brief would have exercised the modern path and produced a false-negative
guard (unable to prove the legacy arm is untouched, since it wouldn't have
been the legacy arm at all). Substitution is justified by the gate's own
source, documented inline, and does not touch the gate itself.

### 5. Four "fixup" test files beyond the brief's inventory — necessary, not scope creep

`v61_test.go`, `v72_test.go`, `v79_test.go`, `version_bounds_test.go` each
had exactly one `NewCharacterSpawn(...)` or `NewCharacterInfo(...)` call site
requiring the new trailing argument. Diffed each: every change is a single
added trailing argument (`, model.RingSet{}` or `, false`) with no other
line touched. This is the mechanical consequence of a constructor signature
change in the same package, required for `go build ./...` to succeed
module-wide — not unrequested scope. The brief's own "module-local go
build/test is mine to fix" framing (implicit in the task's Contract 2, cited
in the report) covers this.

### 6. Build/test verification

- `cd libs/atlas-packet && go build ./...` — clean, no errors.
- `cd libs/atlas-packet && go test ./...` — all packages `ok` (some `[no
  test files]`, expected).
- `go test ./character/clientbound/... -run
  'TestCharacterSpawnRingBlocks|TestCharacterInfoMarriageFlag' -v` — both
  tests and all subtests PASS.
- `services/atlas-channel` was not built (expected to fail to compile until
  Task 7 updates call sites — pre-authorized per the task dispatch, not a
  defect of this unit). Attempting a workspace-relative build failed for an
  unrelated reason (go.work module-resolution error, not a compile error
  from this change) — not pursued further as it is out of this unit's scope.

## Not evaluable

- Whether Task 7's atlas-channel call-site update will in fact be purely
  additive is not verifiable from this unit alone (depends on Task 7's
  diff), but the binding constraint (new params last) is satisfied on this
  side of the contract.

## Verdict rationale

Every requirement in the brief's Interfaces section is met exactly. Both
implementer-flagged deviations (+20 vs +18, GMS v48 vs v28) are verified
against primary evidence (the shipped `ring.go` codec and the shipped
`info.go` gate, respectively) and are corrections of brief errors, not
implementer errors. Test honesty is independently confirmed by fault
injection, not just implementer claim. The four extra test files are
in-scope mechanical fixups. No defects found.

---

verdict: APPROVED
artifact: docs/tasks/task-269-ring-pair-behavior/reviews/task-5.md
scope_confirmed: diff eddb64bb2..940fd0ae4 (8 files, libs/atlas-packet/character/clientbound/ only) plus libs/atlas-packet/model/ring.go read as the binding-authority contract the diff depends on
blocking: 0
non_blocking: 0
not_evaluable: 1
