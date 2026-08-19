# Review: fix-whisper-send-result-presence (2633d67d3)

**Range reviewed:** `62439e69b..2633d67d3` (one code commit, `2633d67d3`; `7ecf2681c` is
docs-only and not reviewed as code). Base `62439e69b` is task-238's own last commit on
this stacked branch — task-238's content (findDecision, location package, presence.go)
is treated as read-only context, not as part of this unit.

**Scope confirmed:** the diff touches exactly the two files the brief names —
`character_chat_whisper.go` (+87/-5) and `character_chat_whisper_test.go` (+163) — plus
the bug doc itself. `kafka/consumer/message/consumer.go` is untouched (verified via
`git diff --stat` against that path — no output). No scope drift.

## 1. Reachable case not caught by the new decision table

`whisperDecision` (character_chat_whisper.go:183-206) switches on `loc.State()`, which
is a `characterconst.PresenceState` with exactly three possible values, confirmed by
reading `libs/atlas-constants/character/presence.go:8-23`: `PresenceStateOffline`,
`PresenceStateInField`, `PresenceStateInCashShop`. `ParsePresenceState` collapses
anything else (including absent) to `Offline`. The switch has an explicit case for
`InField` (deliverable: true) and `InCashShop` (deliverable: false), with `default`
correctly landing on `Offline` (deliverable: false) — there is no fourth wire value that
could fall through unhandled. **No reachable target is incorrectly marked
undeliverable.** PASS.

One design point, not a defect: unlike `findDecision` (character_chat_whisper.go:97-166),
which consults `findLocalSessionFunc` before falling back to `findCharacterLocationFunc`,
`whisperDecision` goes straight to the location lookup with no local-session
short-circuit. This is exactly what the brief specifies ("Resolve presence through the
existing seam `findCharacterLocationFunc` ... Reuse it as-is" — no mention of
`findLocalSessionFunc`), and it is correct: whisper delivery in `handleWhisperChat`
already works across channels via Kafka, so there is no cross-channel restriction to
encode here, only a liveness check. Confirmed no false negative from this omission.

## 2. Infrastructure-error row fails open — verified in code, not just test name

`whisperDecision` (character_chat_whisper.go:189-196):
```go
loc, err := findCharacterLocationFunc(l, ctx, tc.Id())
if err != nil {
    if errors.Is(err, location.ErrNotFound) {
        return whisperOutcome{deliverable: false, branch: "never-logged-in"}
    }
    // Infrastructure failure: fail open.
    return whisperOutcome{deliverable: true, branch: "lookup-failed", err: err}
}
```
`deliverable: true` is the field `produceWhisperChatResult` branches on
(character_chat_whisper.go:232): `if !o.deliverable { announce false; return }` —
for `lookup-failed`, `o.deliverable` is `true`, so this branch is skipped and
`produceWhisperChatFunc` is called at line 236, i.e. the chat command **is** produced.
No `WhisperSendResult(false)` is announced on this path. Test
`TestWhisperChat_Decision/infrastructure_error_fails_open` (whisper test file,
"infrastructure error fails open" case) sets `e.locErr = errors.New(...)` (a non-
`ErrNotFound` error) and asserts `wantAnnounced: false`, `wantProduced: true`,
`wantErrorLevel: true`. Ran it directly (`go test -run TestWhisperChat -v`): PASS.
Confirmed at the code level, not just by the test/case name. PASS.

## 3. `whisperDecision` reuses the existing seams, not parallel lookups

`whisperDecision` calls `findCharacterByNameFunc` (character_chat_whisper.go:184) and
`findCharacterLocationFunc` (character_chat_whisper.go:189) — both are the same
package-level vars `findDecision` already uses (lines 98 and 129/143), declared once
at lines 45-47 and 65-67. No second lookup helper was added. `git diff --stat` confirms
`maps/location/*.go` is unchanged in this commit (it's task-238's prior work).

Test-side seam swapping: `newFindEnv` (shared by both `/find` and whisper tests) swaps
`findCharacterByNameFunc` and `findCharacterLocationFunc` and restores each via
`t.Cleanup` (character_chat_whisper_test.go:112-128) — this setup is reused verbatim by
the whisper tests through `newFindEnv(t)`, so the same cleanup applies. PASS.

## 4. `produceWhisperChatFunc` seam

`produceWhisperChatFunc` (character_chat_whisper.go:52-54) wraps
`message.NewProcessor(l, ctx).WhisperChat(...)` — the exact call the old inline code made
(compare to the diff's removed line: `message.NewProcessor(l, ctx).WhisperChat(s.Field(),
s.CharacterId(), p.Msg(), p.TargetName())`). Argument order and values match
(`f, actorId, msg, recipientName` ↔ `s.Field(), s.CharacterId(), p.Msg(),
p.TargetName()`). No behavioural change to the production call itself. PASS.

Both new tests (`TestWhisperChat_Decision`, `TestWhisperChat_ProduceFailure`) swap
`produceWhisperChatFunc` and restore it via `t.Cleanup` (test file lines ~585-590 and
~629-633). Ran `go test ./socket/handler/... -run TestWhisperChat -v`: both tests and all
six `TestWhisperChat_Decision` subtests pass; ran the full `socket/handler` package after
(`go test ./socket/handler/...`): `ok`, no seam left swapped and bleeding into other
tests. PASS.

## 5. Per-message REST cost — explicit assessment

Every `WhisperModeChat` dispatch now performs two synchronous REST round trips before
producing the Kafka command: `findCharacterByNameFunc` (atlas-character `GetByName`) and
`findCharacterLocationFunc` (atlas-maps `location.Get`) — visible at
character_chat_whisper.go:184 and :189, both unconditional. This is a new fixed latency
cost added to every chat whisper, not just to `/find`.

Assessment: acceptable. This mirrors the /find path's own cost exactly (same two calls,
same seams), whisper-chat is a low-frequency, human-typed action (not a hot per-tick
path), and the alternative — the literal `true` this task fixes — is a correctness bug,
not a performance one. The brief explicitly mandates reusing these seams "as-is" rather
than adding caching or a cheaper check, and no other code in this diff introduces
avoidable duplication (single call each, no per-branch re-lookup). Flagging this
explicitly per the review brief, not as a blocking finding.

## 6. Tests assert the NEW contract, not the old always-true behaviour

The old code (removed lines in the diff) never inspected reachability at all — it called
`WhisperChat` unconditionally and only announced `false` if the producer itself errored;
it never announced `false` for an unknown/offline/cash-shop target, and never withheld
production for those cases. The new `TestWhisperChat_Decision` subtests
`unresolvable name`, `never logged in`, `offline`, and `in cash shop` all assert
`wantProduced: false` and `wantAnnounced: true` (i.e., `WhisperSendResult(false)` is
announced and the chat command is *not* produced) — behaviour the old code could never
exhibit, since it always produced and never announced `false` on these paths. These are
genuine failing-without-the-fix tests, not vacuous coverage. `TestWhisperChat_ProduceFailure`
preserves an existing behaviour (announce `false` on producer error) rather than asserting
anything new, and is correctly framed as such in its own doc comment. PASS.

## Other observations (non-blocking)

- `character_chat_whisper.go:198-205`: the `switch loc.State()` correctly treats every
  unmapped/future state as `offline` via `default`, consistent with `ParsePresenceState`'s
  own fail-safe default. If atlas-maps ever adds a fourth presence state without a
  corresponding channel-side deploy, this degrades to "undeliverable" rather than
  silently treating it as deliverable — the safer failure direction. Noted, not required
  by the brief, and correct as written.
- Log line at character_chat_whisper.go:221-230 includes `requester_id`, `target_name`,
  `branch`, and error detail/level exactly mirroring `produceFindResultBody`'s FR-13
  logging idiom — consistent with the established convention in this file.

## Verification performed

- `git diff --stat 62439e69b..2633d67d3` and full hunk read of both changed Go files.
- Read `libs/atlas-constants/character/presence.go` and
  `services/atlas-channel/atlas.com/channel/maps/location/{model,resolve}.go` to confirm
  `PresenceState` exhaustiveness and `location.Get`/`ErrNotFound` semantics (task-238's
  code, read as a dependency contract only, not re-reviewed).
- `cd services/atlas-channel/atlas.com/channel && go build ./...` — clean.
- `go test ./socket/handler/... -run TestWhisperChat -v` — all 8 sub/tests PASS.
- `go test ./socket/handler/...` (full package) — `ok`, confirming no seam left swapped
  between tests.
- Confirmed `kafka/consumer/message/consumer.go` has zero diff in this range.

## Not evaluable

None. The unit is self-contained within the two files reviewed plus the location/
presence contracts it depends on, both of which were read to confirm the contract holds.

## Verdict

APPROVED. All six weighted checks pass with code-level evidence; tests genuinely pin the
new contract; no scope drift; seams are reused, not duplicated, and correctly restored.
