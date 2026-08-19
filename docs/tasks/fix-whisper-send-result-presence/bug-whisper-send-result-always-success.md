# bug: whispering an offline or non-existent character silently reports success

**Reproduced:** not reproduced live; established by reading the code path on
`main` and on `task-238-whisper-find-location`. The mechanism below is traced
end to end, not inferred.

**Observed (by inspection):** whichever way a whisper to an unreachable
recipient fails, the sender is told nothing or told it worked.

- `services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go`
  — `handleWhisperChat` emits
  `fieldcb.NewWhisperSendResult(0x0A, tc.Name(), true)`. The `true` is a
  literal. It is emitted once the chat event round-trips, which says only that
  the event was published — nothing about whether any session received it.
- If the recipient name does not resolve to a character at all,
  `services/atlas-messages/atlas.com/messages/message/processor.go`
  `HandleWhisper` logs `Unable to locate recipient [%s]` and returns an error.
  That error terminates the Kafka handler; **no event is published and no
  packet is ever sent back to the sender.** The sender's client sits with no
  feedback whatsoever.
- If the recipient exists but is logged out, `HandleWhisper` succeeds (it only
  checks that the character record exists and that worlds match), the event is
  published, no channel holds a session for them, and the sender receives
  `success: true`.

**Expected:** the sender receives `WhisperSendResult` with `success: false`
when the target cannot receive the message — unknown name, logged out, or in
the cash shop — and `success: true` only when a live field session exists to
deliver to.

**Root cause:** there is no world-wide liveness lookup on `main`.
`services/atlas-maps/.../character/location/model.go` on `main` stores
world/channel/map with no liveness discriminator, so a stale row for a
logged-out character is byte-identical to a live one. The channel service can
only see its own channel's session registry. That gap is why the literal
`true` was written in the first place, and why `character_chat_whisper.go` on
`main` carries `// TODO find a way to look up remote channel.`

**This branch is stacked on `task-238-whisper-find-location` precisely because
task-238 closes that gap.** It adds:

- `libs/atlas-constants/character/presence.go` — `PresenceState` with
  `PresenceStateOffline` / `PresenceStateInField` / `PresenceStateInCashShop`,
  and `ParsePresenceState`, which resolves anything unrecognised (including
  absent) to `OFFLINE`.
- `services/atlas-channel/atlas.com/channel/maps/location` — `Model.State()`
  and `location.Get(l, ctx, characterId) (Model, error)`, returning
  `location.ErrNotFound` on HTTP 404 and the underlying error otherwise.

## Fix

Put the decision in the **serverbound handler**, not in the Kafka consumer.

- `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper.go`
  — the `p.Mode() == chat.WhisperModeChat` branch. Today it produces the chat
  command unconditionally and only reports `false` when the *producer* itself
  errors. Change it to resolve the target's reachability first:
  - Resolve the target character by name through the existing package-level
    seam `findCharacterByNameFunc` (already defined in this file for the
    `/find` path — reuse it, do not add a second lookup helper).
  - Resolve presence through the existing seam `findCharacterLocationFunc`
    (`location.Get`, already defined in this file). Reuse it as-is.
  - Decision table:
    - character name does not resolve → announce
      `fieldcb.NewWhisperSendResult(0x0A, p.TargetName(), false)` and return
      **without** producing the chat command.
    - `location.ErrNotFound` (no location row — never logged in) → same:
      `false`, do not produce.
    - `PresenceStateOffline` → same: `false`, do not produce.
    - `PresenceStateInCashShop` → same: `false`, do not produce. A cash-shop
      session lives in atlas-cashshop's socket, not in any channel's session
      registry, so `handleWhisperChat` could not deliver to it — reporting
      `true` would be a lie. This is a deliberate ruling; keep it.
    - `PresenceStateInField` → produce the chat command exactly as today. The
      `success: true` continues to come from the Kafka consumer's
      `handleWhisperChat` on the round trip. Do not move or duplicate it.
    - **Infrastructure error** from `location.Get` (anything that is not
      `ErrNotFound`) → log at error level and **fail open**: produce as today.
      A transport blip must not turn a deliverable whisper into a reported
      failure. This mirrors the reasoning task-238 applied to `/find`, but the
      opposite direction is correct here, because unlike `/find` this path has
      a real side effect to preserve.
  - Keep the existing producer-error branch (`false` on produce failure) as it
    is.
  - Follow this file's established idiom: task-238 extracted `/find`'s decision
    into a pure, table-tested function with named branches
    (`findOutcome` / `findOutcomeKind`). Do the same here — a pure function
    returning the decision plus a branch name, so each rule stays separable in
    logs and directly unit-testable. Do not grow an inline `if/else` ladder in
    the handler body.

- `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper_test.go`
  (extend; task-238 already added it) — one case per row of the decision table
  above, asserting both the emitted packet and whether the chat command was
  produced. Swap the seams and restore with `t.Cleanup`, exactly as the
  existing `/find` tests in that file do. The infrastructure-error case must
  assert **fail-open** (command produced, no `false` packet).

- Do **not** touch
  `services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go`.
  A sibling branch (`fix-whisper-cross-channel-delivery`) is rewriting
  `handleWhisperChat` there; keeping this change out of that file is what keeps
  the two branches conflict-free.

## Not yet answered

- Whether an offline recipient should get the whisper *queued* (as the Notes
  system does) rather than dropped. Out of scope — do not build it.
- `atlas-messages`' `HandleWhisper` still returns an error for an unresolvable
  recipient, which now becomes unreachable in practice for whispers originating
  from this handler. Leave it in place as a defensive check; do not remove it.

## Resolution

_(pending)_
