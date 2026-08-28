# Review — bug-gift-note-nflag fix (commit `76a7ab60e`)

Task: task-240-cash-shop-stub-operations
Range reviewed: `906439e31..76a7ab60e` (single commit `76a7ab60e`)
Brief: `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-note-nflag.md`
RE authority: `docs/tasks/task-240-cash-shop-stub-operations/re-memo-nflag.md`
Report: `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-note-nflag-report.md`

## Scope

```
git diff --stat 906439e31..76a7ab60e
```
touches exactly:
- `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward_test.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/note_send.go`
- `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-note-nflag.md` (new, doc)
- `docs/tasks/task-240-cash-shop-stub-operations/re-memo-nflag.md` (doc, "Resolved" section added)

Matches the brief's three-file "Fix" inventory exactly; nothing else in the
tree changed. `scope_confirmed`: the review covers the three code-file
changes plus a hand trace of the payload's cross-service path through
saga-orchestrator, atlas-notes, and back into atlas-channel's clientbound
codec (files outside the diff, read only because correctness genuinely
depends on their contract, per the review brief's explicit instruction).

## Findings — requirement by requirement

### 1. `buildGiftForwardSaga` sends `Flag: 1`, `buildNoteSendSaga` still sends `0`

PASS.
- `note_gift_forward.go` diff: `Flag: 0` → `Flag: 1`
  (`services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go:53`
  in the post-fix file), with the comment replaced by real semantics citing
  StringPool 3366/3367 and `re-memo-nflag.md`, and stating why 1 (not 2):
  Atlas awards the fame itself via `buildGiftFameSaga`.
- `note_send.go` diff: `Flag: 0` value **unchanged**; only the comment was
  rewritten to state the now-known concrete semantics (1/2 gift block, 3
  wedding invite) and cite `re-memo-nflag.md`. Confirmed by reading the
  hunk — no `Flag:` value line appears in the note_send.go diff, only the
  comment block above it.

### 2. Flag survives end-to-end: saga payload → saga-orchestrator `CreateNote` → atlas-notes `Model.Flag()` → note status event → channel clientbound `NoteEntry.Flag`

PASS, verified by hand-tracing every hop (all outside the diff, read per the
review brief's explicit escalation for this exact question):

1. `services/atlas-channel/.../saga/model.go` — `CreateNotePayload.Flag byte`, set to 1 by the fixed code.
2. `services/atlas-saga-orchestrator/.../saga/handler.go:3790` —
   `handleCreateNote` calls `h.noteP.CreateNote(..., payload.Flag, payload.GiftNote)` — passthrough.
3. `services/atlas-saga-orchestrator/.../note/producer.go:25` —
   `CreateNoteCommandProvider` puts `Flag: flag` verbatim into the Kafka
   `CommandCreateBody`.
4. `services/atlas-notes/.../kafka/consumer/note/consumer.go:51` —
   `CreateAndEmit(..., c.Body.Flag, ...)` — passthrough.
5. `services/atlas-notes/.../note/processor.go:116-121` — `CreateAndEmit`
   threads `flag` through the curried `Create` builder chain unmodified,
   then (`processor.go:101`) emits
   `CreateNoteStatusEventProvider(..., m.Flag(), ...)` — reads the flag back
   off the persisted `Model`, not off the original input, but
   `note/builder.go:57 SetFlag` / `note/model.go:47 Flag()` do a bare
   store/return of the `byte`, no clamp.
6. `services/atlas-notes/.../note/producer.go:16-24` —
   `CreateNoteStatusEventProvider` puts `Flag: flag` verbatim into
   `StatusEventCreatedBody`.
7. Client re-reads the note list rather than the create event carrying the
   entry directly: `services/atlas-channel/.../kafka/consumer/note/consumer.go`
   `handleNoteCreated` only triggers `NoteOperationWriter(NoteRefreshBody())`
   (a "go re-fetch" signal), not the flag itself. The flag reaches the wire
   on the client's follow-up `NoteOperationRequest`, handled in
   `services/atlas-channel/.../socket/handler/note_operation.go:111-150`:
   `note.NewProcessor(...).GetByCharacter(...)` (a REST call into
   atlas-notes) returns `note.Model`, and line 143
   `Flag: m.Flag()` / line 149
   `notepkt.NoteEntry{..., Flag: n.Flag}` — passthrough.
8. `services/atlas-channel/.../note/rest.go:19,50,62` — the REST DTO
   round-trips `Flag byte` verbatim (`n.Flag()` in, `SetFlag(r.Flag)` out);
   no transform.
9. `libs/atlas-packet/note/entry.go:29` — `NoteEntry.Encode` does
   `w.WriteByte(n.Flag)` — a raw byte write, no clamp, no lookup table.

No hop clamps, drops, or hard-codes the value anywhere on this path. This
part of the review brief's explicit ask ("test asserts the value the client
actually receives is 1, not merely that the saga payload says 1") is only
**partially** satisfied by tests, see finding below.

**Non-blocking gap**: no single test exercises this full pipeline and
asserts a wire byte of `1`. What exists:
- `note_gift_forward_test.go:57` (this commit) pins the saga *payload*
  `np.Flag != 1` — one hop.
- `libs/atlas-packet/note/clientbound/display_test.go:20,41-42` (pre-existing,
  not touched by this commit, "tier-1 fixtures" per the brief's "Do not
  touch") already asserts a `NoteEntry{Flag: 1}` decodes back to `Flag == 1`
  through `Encode`/`Decode` — proving the codec's terminal hop is
  flag-preserving, generically, not specifically for the gift path.
No test connects the two — i.e. no test proves `buildGiftForwardSaga`'s
`Flag: 1` is the same `1` that reaches `NoteEntry.Encode`. Given the brief
explicitly states "No codec change is needed" and lists the codec under "Do
not touch," and given the intervening hops (saga-orchestrator, atlas-notes)
are genuinely a different service with their own test suites not in this
commit's scope, this is not a defect in the commit under review — it is an
inherent limit of single-commit scope for a value that crosses three
services. Flagging as non-blocking / not fully evaluable within this
commit's diff; the manual trace above is the closest available substitute
evidence.

### 3. No unintended change to `DiscardEntry` / `discardSpecialFlag` for `flag == 1`

PASS. `discardSpecialFlag`
(`libs/atlas-packet/note/serverbound/operation_discard.go:22-27`) returns
only `2` (GMS ≤ v61) or `3` (all others) — never `1` — and is untouched by
this diff (not in `git diff --stat` output). A `flag == 1` entry cannot
match `discardSpecialFlag` on any version, so `DiscardEntry` decodes it as
an ordinary two-field entry exactly as before this change, matching the
brief's "Do not touch" instruction and its claim in `bug-gift-note-nflag.md`
line 62-66.

### 4. `GiftNote: true` sender-fame suppression in atlas-notes unaffected

PASS. `services/atlas-notes/atlas.com/notes/note/processor.go:340`
(`if m.GiftNote() { ... }`, the discard-time fame-suppression check) branches
on the `GiftNote` bool, not on `Flag`, and atlas-notes is entirely outside
this commit's diff. `note_gift_forward.go`'s `GiftNote: true` line
(immediately below the changed `Flag: 1` line per the diff context) was not
touched.

## TDD honesty check

Confirmed directly, not merely by the report's claim: `git show
906439e31:.../note_gift_forward.go` shows the pre-fix file still has
`Flag: 0` at that line. The flipped assertion `np.Flag != 1`
(`note_gift_forward_test.go:57`) therefore fails against the base commit
(`np.Flag` would be `0`) and passes against `76a7ab60e`. This is a genuine
regression pin, not a vacuous test.

## Test execution

```
cd services/atlas-channel/atlas.com/channel
go test ./socket/handler/... -run 'TestBuildGiftForwardSaga|TestBuildNoteSendSaga' -v
```
```
=== RUN   TestBuildGiftForwardSaga
--- PASS: TestBuildGiftForwardSaga (0.00s)
=== RUN   TestBuildNoteSendSaga
--- PASS: TestBuildNoteSendSaga (0.00s)
PASS
ok  	atlas-channel/socket/handler	0.023s
```

## Not evaluable

- **Live client confirmation.** The brief itself scopes this out ("Not yet
  answered" §1) — no running client was available to this review either.
  Consistent with the brief; not a defect of this commit.
- **Full cross-service wire assertion.** As noted above, no test spans
  saga-orchestrator + atlas-notes + atlas-channel to assert the client
  literally receives byte `1`. Substituted with a manual, cited hop-by-hop
  trace; counted as not evaluable via automated test within this commit's
  scope, not as a blocking defect (the intervening services' own test
  suites are outside this diff).

## Verdict rationale

Every requirement in the brief's `## Fix` section is implemented exactly as
specified, confined to the three named files, with a genuine (not vacuous)
regression-pinning test, and no adjacent behavior (`DiscardEntry`,
`GiftNote` fame suppression, `note_send.go`'s value) was disturbed. The one
gap — no single test asserting the terminal wire byte for the gift path
specifically — is a pre-existing structural limit (the value crosses three
services) rather than something this commit did wrong, and is disclosed
here rather than silently absorbed into approval.
