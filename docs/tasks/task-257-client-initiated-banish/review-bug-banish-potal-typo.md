# Review: `potal` typo fallback — commit `e68ef7313`

**Range reviewed:** `1c42f52e0..e68ef7313` (single commit `e68ef7313`). Commit
`1c42f52e0` was reviewed separately (`review-bug-banish-map-id-always-zero.md`) and
is out of scope here.

**Brief:** `docs/tasks/task-257-client-initiated-banish/bug-banish-map-id-always-zero.md`,
section "Follow-up: `potal` typo — IN SCOPE".

## Scope

Files touched: `services/atlas-data/atlas.com/data/monster/reader.go` (+13/-1),
`services/atlas-data/atlas.com/data/monster/reader_test.go` (+68 new test), and the
bug file itself (documentation only, +229). `xml/model.go` is unmodified by this
commit (confirmed by diff — see finding 2 below).

## Findings

### 1. `getBanish` portal resolution order and the changed first-call default — PASS

`reader.go:132-138` (post-fix):

```go
portal := b.GetString("banMap/0/portal", "")
if portal == "" {
    portal = b.GetString("banMap/0/potal", "")
}
if portal == "" {
    portal = "sp"
}
```

vs. pre-fix (`reader.go:127` at `1c42f52e0`): `portal := b.GetString("banMap/0/portal", "sp")`.

`xml.Node.GetString` (`services/atlas-data/atlas.com/data/xml/model.go:82-90`) returns
the node's actual `Value` whenever a matching `<string>` child exists, regardless of
whether that value is empty — the `def` parameter is only used when no matching node
is found at all. So the changed default (`""` instead of `"sp"`) only changes behavior
for a WZ node where `portal` is *present* with an empty string value; in every other
case (present-with-value, or absent) the two forms are equivalent.

Checked this directly against the confirmed-available serialized dump at
`tmp/ec876921-c363-4cc6-9c51-5bb8d57f9553/GMS/83.1/Mob.wz` (1564 `*.img.xml`
images, 26 with an `info/ban` node — file count and node-count match the bug
file's table exactly):

```
$ grep -rn 'name="portal" value=""' Mob.wz/*.img.xml   # no match
$ grep -rn 'name="potal" value=""'  Mob.wz/*.img.xml   # no match
```

Every `ban` node in the dump is one of: `portal` present with a real value (19),
`potal` present with a real value — always `out00` (3: `9500194`, `9500303`,
`9500304`), or neither key present at all (4: `9300139`, `9300140`, `9300151`,
`9300152` — verified their `banMap/0` subtree literally contains only `<int
name="field">`, no portal/potal child). No node anywhere in this scope has an
empty-string `portal` value, so the changed default introduces no observable
regression in the one scope I could ground against real data. The three-scan
fallback chain still terminates at `"sp"` for the no-portal-child mobs exactly as
before.

### 2. Fallback stayed local to `reader.go` — PASS

`git diff 1c42f52e0..e68ef7313 -- services/atlas-data/atlas.com/data/xml/model.go
services/atlas-data/atlas.com/data/xml/model_test.go` produces no output — `xml.Node`
is untouched by this commit. `git grep -n "potal"` across `services/atlas-data`
outside the two touched files returns nothing. The fallback is confined to
`getBanish` in `reader.go`, matching the operator ruling in the bug file ("Do NOT
make the fallback generic in `xml.Node`").

### 3. `TestReaderBanishPortalTypo` — PASS, genuinely pins the fix

Ran the new test against the fixed `reader.go` (all three subtests pass):

```
--- PASS: TestReaderBanishPortalTypo (0.00s)
    --- PASS: .../portal_wins_when_both_spellings_present (0.00s)
    --- PASS: .../potal_resolves_when_portal_absent (0.00s)
    --- PASS: .../sp_default_when_neither_spelling_present (0.00s)
```

Then checked out `reader.go` at `1c42f52e0` (pre-fix) with the new test file
overlaid, same package, same test:

```
--- FAIL: TestReaderBanishPortalTypo (0.00s)
    --- PASS: .../portal_wins_when_both_spellings_present (0.00s)
    --- FAIL: .../potal_resolves_when_portal_absent (0.00s)
        reader_test.go:1642: PortalName="sp", want "out00"
    --- PASS: .../sp_default_when_neither_spelling_present (0.00s)
```

This is the expected shape: the "portal wins" and "sp default" subtests pass
either way (they don't exercise the typo path — the former because `portal` is
correctly read even pre-fix, the latter because no-portal-child already defaulted
to `"sp"`), but the "potal resolves when portal absent" subtest — the one that
actually needs the fallback — fails before the fix and passes after. The brief's
requirement ("must fail before and pass after") is met at the subtest level, and
the "both spellings present → `portal` wins" case is explicitly covered as required.
Worktree was restored to a clean `e68ef7313` state after this check (`git status`
showed no diff on the touched files).

### 4. Per-version sweep table — internally consistent, coherent

Table at `bug-banish-map-id-always-zero.md:203-214`. For every row, `portal + potal
+ neither` sums to the reported `ban nodes` / `banMap/0/field` count:

- GMS 61.1: 15+3+4=22 ✓; GMS 72.1: 18+3+4=25 ✓; GMS 79.1/83.1/84.1/87.1:
  19+3+4=26 ✓; GMS 92.1: 23+3+4=30 ✓; GMS 95.1: 14+3+4=21 ✓; JMS 185.1:
  19+3+4=26 ✓.
- The GMS 83.1 row (26/26/0/19/3/4) matches what I independently counted from the
  locally-available serialized dump (finding 1), which is the one row I could
  ground-truth directly.
- `potal` count is a stable 3 and `neither` a stable 4 across every non-zero row,
  consistent with the claim that these are fixed WZ authoring artifacts that
  persist unchanged once a template is introduced, rather than something that
  varies by client version.
- The zero row (GMS 48.1: all columns 0) is explained by "none of the 26 templates
  that ever carry a `ban` node exist yet at that version — 687 images total vs.
  1564 at 83.1," with the confirming method stated (name-searching the parsed
  image list for `5090000`/`9500194`/`9300139` and finding none). I could not
  independently verify the 48.1 counts — no GMS 48.1 dump exists locally
  (`find tmp -ipath "*GMS/48.1*Mob.wz*"` returns nothing) and the scratch
  MinIO-walking tool was explicitly not committed. Per the task instructions I am
  not re-running the MinIO sweep; the row is coherent on its face (a total-zero
  row for an early version with roughly half the image count of 83.1 is a
  plausible and internally non-contradictory claim) but is flagged below as not
  independently re-verified by this review.
- No scope reported a third spelling, multi-entry `banMap`, or `field` stored as
  a string, which is the condition the brief said would require stopping rather
  than widening the fallback. Not contradicted by anything in the table.

## Not evaluable

- The per-version sweep for GMS 48/61/72/79/84/87/92/95.1 and JMS 185.1 was
  produced by a scratch, uncommitted tool against MinIO; I have no local dump for
  any of those scopes and did not re-run the MinIO sweep (out of scope per the
  reviewing brief). Judged for internal coherence only (see finding 4).
- The live re-test and the client-initiated-banish trigger question are explicitly
  marked "NOT DONE" / "not resolved" in the bug file's own Resolution section and
  are not part of this commit's diff.

## Verdict rationale

The fix does exactly what the brief's "Follow-up" section specified: `portal`
first, `potal` fallback, `"sp"` final default, comment naming the three affected
template ids, fallback kept local to `reader.go`, and a three-case test that
distinguishes the fix from the pre-fix behavior at the subtest granularity that
matters. The one behavior-changing detail (first-call default flipped from `"sp"`
to `""`) does not change any observable output against the one scope I could
ground-truth, and is logically inert given how `GetString` treats present-vs-absent
distinct from empty-vs-nonempty values.

```text
verdict: APPROVED
artifact: docs/tasks/task-257-client-initiated-banish/review-bug-banish-potal-typo.md
scope_confirmed: reviewed only commit e68ef7313 (reader.go potal fallback + reader_test.go + bug file); 1c42f52e0 excluded per instruction
blocking: 0
non_blocking: 0
not_evaluable: 2
```
