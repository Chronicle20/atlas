# Task 13 — scoped re-review

Range: `47fdbf191..6e557896d` (1 commit, `6e557896d`)
Diff source: `.superpowers/sdd/plan/review-47fdbf191..6e557896d.diff`
Report: `.superpowers/sdd/plan/task-13-report.md` ("Fix round 1" section)

## Scope

Single commit, comment-only per the implementer's account. Verified against
the actual diff (`git diff --stat 47fdbf191..6e557896d`):

```
 .../kafka/message/parcel/custody/kafka.go | 7 +++++--
 1 file changed, 5 insertions(+), 2 deletions(-)
```

One file touched, all five added/changed lines are inside `//` doc comments
(const-block comments on `CommandRestoreParcel`/`CommandRemoveParcel`, and the
struct doc comments on `RestoreParcelCommandBody`/`RemoveParcelCommandBody`).
No identifier, type, field, string literal, or code statement changed.
`scope_confirmed`: matches — this is exactly and only the comment fix the
report describes.

## Finding under re-review

**Original finding:** `CommandRestoreParcel`/`CommandRemoveParcel` doc
comments did not state idempotency (0 rows affected = success), though the
brief required it.

**Verdict: ADDRESSED.**

Evidence:

- Brief text (`.superpowers/sdd/plan/task-13-brief.md:107`): "Both are
  idempotent: 0 rows affected is success."
- Current source
  (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/parcel/custody/kafka.go`):
  - Line 26: `// Idempotent: 0 rows affected is success.` directly above
    `CommandRestoreParcel = "RESTORE_PARCEL"` (line 27).
  - Line 30: `// Idempotent: 0 rows affected is success.` directly above
    `CommandRemoveParcel = "REMOVE_PARCEL"` (line 31).
  - `RestoreParcelCommandBody` struct doc comment (diff hunk 2): "... the
    compensating inverse of RELEASE_FROM_PARCEL). Idempotent: 0 rows affected
    is success."
  - `RemoveParcelCommandBody` struct doc comment (diff hunk 2): "...
    compensating inverse of ACCEPT_TO_PARCEL). Idempotent: 0 rows affected is
    success."
- Wording check: all four instances say "0 rows affected is success," not
  merely "is idempotent." This is the unambiguous form the finding demanded —
  a later reader cannot mistake it for "0 rows means the target row wasn't
  found, treat as an error."
- Comment-only confirmed: diff contains no non-comment line changes (checked
  the full diff hunks above; every added `+` line is either a `//` comment
  line or a comment continuation, and one deleted `-` line per struct that is
  the pre-fix single-line version of the same comment, replaced by the
  extended two-line version).

## Sweep claim spot-check

The report claims a sweep found no other brief Step-3 requirement landed only
in the report and not in the source. Checked directly against brief Step 3
(`.superpowers/sdd/plan/task-13-brief.md:98-107`):

- "why RESTORE_PARCEL exists" (un-resolves a parcel released by a
  `withdraw_from_parcel` whose `accept_to_character` then failed) — present
  verbatim in the `CommandRestoreParcel` const comment
  (kafka.go:23-25) and in the `RestoreParcelCommandBody` struct comment.
- "why REMOVE_PARCEL exists" (hard-deletes a still-pending row from a late
  `accept_to_parcel` after the saga already compensated) — present verbatim
  in the `CommandRemoveParcel` const comment (kafka.go:28-30) and in
  `RemoveParcelCommandBody`'s struct comment.
- Idempotency clause — now present on all four, as verified above.

No other Step-3 clause (envelope copied verbatim, renamed from the MTS twin)
is comment-text-bearing beyond what's already in the file header
("mirror atlas-parcel's kafka/message/custody/kafka.go byte-for-byte..."),
which predates this commit and was not part of the finding. Sweep claim
holds for the parts checkable from Step 3's text.

## Divergence check (parcel file vs. MTS twin)

Read `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/mts/custody/kafka.go`
directly: `CommandRestoreMtsHolding`, `CommandRemoveMtsListing`, and their
body structs (`RestoreMtsHoldingCommandBody` inferred not shown in the
excerpt but the const-block comments for `CommandRemoveMtsListing` at
lines 24-25 of that file) carry no idempotency statement — the doc comments
only explain existence/purpose, not rows-affected semantics.

The parcel file's added text is additive only: it states a fact (idempotency)
that the MTS twin's comments are silent on, and does not restate or alter any
of the "why this command exists" prose that the two files share in parallel
form. No contradiction — the MTS twin doesn't claim non-idempotency, it simply
doesn't discuss idempotency at all. Confirmed correct as the report describes.

## Disposition

- PASS — idempotency statement present, unambiguous ("0 rows affected is
  success"), on both commands' const comments and both bodies' struct
  comments.
- PASS — diff is comment-only; no behavioral/code change slipped in.
- PASS — sweep claim holds against brief Step 3's full text.
- PASS — MTS-twin divergence is additive only, not contradictory.

## Not evaluable

None — the full scope of this fix (a 5-line comment-only diff against a
single, precisely-worded brief requirement) was directly checkable from the
diff, the brief, the current source file, and the MTS twin file.
