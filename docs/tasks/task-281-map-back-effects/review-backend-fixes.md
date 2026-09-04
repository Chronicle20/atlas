# Review — backend-guidelines fix rounds (findings B1/B2/B3)

- **Range reviewed:** `797d1e0cf..318b1c0c0` (`3375aa52b`, `318b1c0c0`)
- **Brief:** `docs/tasks/task-281-map-back-effects/audit-backend.md` blocking findings 1–3
  (DOM-25 consolidated finding, FILE-05/06, DOM-05)
- **Implementer reports:** `.superpowers/sdd/plan/backend-fix-b2-b3-report.md`,
  `.superpowers/sdd/plan/backend-fix-b1-report.md`
- **Verdict:** APPROVED_WITH_FINDINGS

## Scope confirmed

`git diff --stat 797d1e0cf..318b1c0c0` matches the two commits described: the
`backeffect` package split + `TransformSlice` (3375aa52b, B2/B3) and the
semantic-effect-key refactor across atlas-messages → atlas-maps →
atlas-saga-orchestrator → atlas-channel (318b1c0c0, B1). No unrelated files
touched. `services/atlas-channel/.../socket/writer/set_back_effect.go` and
`saga/handler.go` are **not** in this diff (both pre-date it, from the
earlier Task 8 work) — I traced them anyway per Question 1 since correctness
of the B1 seam depends on their contract.

## Q1 — is the clientbound send path actually wired? (priority)

**Finding: not blocking. The send path is intact; the flagged file is
genuinely dead but is a redundant duplicate, not the live path.**

Traced by hand: Kafka status event → atlas-channel consumer → packet struct →
socket.

- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:1181-1199`
  (`handleStatusEventBackEffectSet`) and `:1201-1219`
  (`handleStatusEventBackEffectClear`) are the live consumers, registered at
  `consumer.go:125,130`.
- Both call `doorAnnounce(l, ctx, wp, fieldcb.SetBackEffectWriter,
  fieldcb.NewSetBackEffect(backEffectWireByte(e.Body.Effect), ...).Encode, s)`
  (`consumer.go:1193`) / `fieldcb.NewClearBackEffect().Encode`
  (`consumer.go:1213`) — calling the `libs/atlas-packet` codec constructor
  **directly**, not through `socket/writer/set_back_effect.go`'s
  `SetBackEffectBody(effect byte, ...)` wrapper.
- `doorAnnounce` (`consumer.go:790`) wraps `session.Announce`, which is the
  real socket write path.
- `fieldcb.SetBackEffectWriter` / `fieldcb.ClearBackEffectWriter`
  (`libs/atlas-packet/field/clientbound/set_back_effect.go:13`,
  `clear_back_effect.go:12`) are operation-name string constants registered
  in `atlas-channel/main.go:873-874`'s `produceWriters()` list — this is the
  opcode-name registration table, unrelated to the `socket/writer` package.

Confirmed `services/atlas-channel/atlas.com/channel/socket/writer/set_back_effect.go`
(`SetBackEffectBody(effect byte, ...)`) and its `clear_back_effect.go` sibling
have **zero callers** anywhere in the atlas-channel module (`grep -rn
"writer\.SetBackEffectBody\|writer\.ClearBackEffectBody" .` — no hits outside
the two definitions). This is a real dead-code file. But:

1. It is **not part of this diff** — `git log --follow` shows it was added in
   `057060bee` ("feat(atlas-channel): broadcast back-effect set and clear
   packets"), predating `797d1e0cf`. Neither B1/B2/B3 fix commit touches it.
2. It is not a lone anomaly: `socket/writer/play_jukebox.go`'s
   `PlayJukeboxBody(itemId int32, playerName string)` has the exact same
   shape and the exact same zero-caller status — `consumer.go` calls
   `fieldcb.NewPlayJukebox(...).Encode` directly at
   `consumer.go:1137,1161,1229,1243` instead of `writer.PlayJukeboxBody(...)`.
   This is a pre-existing, tolerated convention in this package (a thin
   `socket/writer` wrapper coexisting with a direct `fieldcb.New*(...).Encode`
   call site), not something task-281 introduced or broke.
3. `TestHandleStatusEventBackEffectSet_*` /
   `TestAnnounceActiveBackEffects_*` (`consumer_test.go:837-1046`) exercise
   the real `doorAnnounce`/codec path end-to-end (decode the captured wire
   bytes via `fieldcb.SetBackEffect`'s own decoder) and pass, which would not
   be possible if the send path were broken.

Feature is **not** broken end-to-end. The B1 implementer's own report
(`.superpowers/sdd/plan/backend-fix-b1-report.md:173-179`) already surfaces
this exact fact accurately ("dead code, no callers, left unchanged to avoid
touching unrelated dead code outside the brief's inventory") — that
self-report is correct and does not need correction.

**Non-blocking cleanup note:** `set_back_effect.go`'s signature
(`effect byte`) is now the only place in the codebase still describing the
value as a bare byte with no semantic type at the call boundary (compare
`backEffectWireByte(e beconst.Effect) byte` in `consumer.go:1172`, which *is*
the resolution point). Since it has no callers it cannot desync from
anything today, but if a future change reintroduces a caller without
noticing `backEffectWireByte` exists, DOM-25 could regress silently. Worth a
follow-up ticket to delete both dead wrapper files (`set_back_effect.go`,
`clear_back_effect.go`, and while at it `play_jukebox.go`) rather than leave
three files that look load-bearing but aren't. Not blocking this review.

## Q2 — B1 correctness

**PASS on every point checked.**

- `git diff 797d1e0cf..318b1c0c0 -- libs/atlas-packet` — genuinely empty
  (verified directly; no output). `go test ./...` in `libs/atlas-packet`
  green, `go run ./tools/packet-audit matrix --check` clean (only two
  pre-existing n/a-evidence notes, unrelated).
- Kafka command topic (`COMMAND_TOPIC_MAP`) bodies match byte-for-byte across
  all three producers/consumers:
  - `services/atlas-messages/atlas.com/messages/kafka/message/map/kafka.go:14-16,35-42`
  - `services/atlas-maps/atlas.com/maps/kafka/message/map/command.go:16-17,42-49`
  - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/map/kafka.go:19-20,45-52`
  - All three declare `SetBackEffectCommandBody{ Effect backeffect.Effect;
    FieldId uint32; PageId uint8; Duration uint32 }` and
    `ClearBackEffectCommandBody{}` identically.
- Status-event topic (`EVENT_TOPIC_MAP_STATUS`) bodies match byte-for-byte
  atlas-maps → atlas-channel:
  - `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go:66-73`
  - `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go:66-73`
  - Identical `BackEffectSet{Effect backeffect.Effect; FieldId uint32; PageId
    uint8; Duration uint32}` / `BackEffectClear{}`.
- `libs/atlas-saga/payloads.go:1323-1336` (`SetBackEffectPayload`) also
  carries `backeffect.Effect`, and `unmarshal_test.go:1667-1691` was updated
  to encode `"effect":"SHOW"` and assert
  `p.Effect != backeffect.EffectShow` — a genuine behavior-changing test (the
  old fixture literal `"effect":0` would now fail to unmarshal into a
  `string`-typed `Effect`, so this test would fail without the fix, not just
  pass either way).
- Semantic-to-wire resolution happens in **exactly one place**:
  `backEffectWireByte(e beconst.Effect) byte` at
  `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:1172-1178`,
  called from both `handleStatusEventBackEffectSet` (`:1193`) and
  `announceActiveBackEffects` (`:1243`). No other cast/switch on `Effect`
  exists in the diff.
- Semantics preserved, not inverted:
  `libs/atlas-packet/field/clientbound/set_back_effect.go:21-22` —
  `BackEffectShow byte = 0`, `BackEffectHide byte = 1`, unchanged (packet lib
  frozen). `backEffectWireByte` maps `EffectHide → BackEffectHide`,
  else `→ BackEffectShow` — correct, non-inverted. Confirmed by
  `consumer_test.go:871-872` (`EffectShow → BackEffectShow`) and
  `consumer_test.go:994,1002` (first entry `EffectShow → BackEffectShow`,
  second `EffectHide → BackEffectHide`) — a genuine round-trip assertion
  through the real codec decoder (`decodeSetBackEffect`), not a mock.
- New domain type location: `libs/atlas-constants/backeffect/effect.go`
  (type `Effect string`, consts `EffectShow`/`EffectHide`) — correctly placed
  under `libs/atlas-constants` per CLAUDE.md, not duplicated in a service
  module.
- GM command syntax unchanged:
  `services/atlas-messages/atlas.com/messages/command/map/back_effect.go:41`
  regex `^@backeffect\s+(\d+)\s+([01])(?:\s+(\d+))?$` —
  `@backeffect <pageId> <effect> [durationMs]`, `0`/`1` digit still the
  user-facing syntax (0=show/1=hide), resolved to the semantic type by
  `parseBackEffect` (`back_effect.go:26-34`) immediately at the chat-command
  boundary — this is the single other resolution point mentioned in the
  brief (GM digit → semantic), separate from and consistent with the
  channel-side wire resolution.

## Q3 — B2/B3 (backeffect package split + TransformSlice)

**PASS.**

- Pure move, no exported name changed: `model.go` now holds `FieldKey` /
  `BackEffectEntry` (unchanged fields/comments verbatim from the old
  `registry.go`), `administrator.go` holds `Set`/`Clear`, `provider.go` holds
  `Get`, and `registry.go` is reduced to the `Registry` struct + singleton
  (`getRegistry`/`sync.Once`). Confirmed no behavior change — `go test
  ./...` in `atlas-maps` module is green including
  `map/backeffect` and `registry_test.go` (unchanged assertions, just moved
  file references).
- `rest.go:42-51` `TransformSlice`:
  ```go
  func TransformSlice(es []BackEffectEntry) ([]RestModel, error) {
      out := make([]RestModel, 0, len(es))
      ...
  }
  ```
  `make([]RestModel, 0, len(es))` guarantees a non-nil, empty slice
  (marshals to JSON `[]`, never `null`) when `es` is empty. Exercised
  end-to-end (through the JSON:API wrapper, not a raw `TransformSlice` unit
  test) by `resource_test.go:100-122`
  (`TestGetBackEffectsInMap_EmptyIsTwoHundred`), which asserts
  `http.StatusOK` and `len(doc.Data.DataArray) == 0`. No dedicated
  `TransformSlice`-level unit test exists (would be a marginal
  belt-and-suspenders addition, not required) — noted as non-blocking.
- `resource.go:38` now calls `TransformSlice(entries)` directly, replacing
  the old inline per-element loop (DOM-05 finding closed).

## Q4 — errcheck

**PASS across both commits, with one pre-existing-pattern note.**

- Searched the diff hunks for the classic misses (unchecked
  `producer.ProviderImpl`, `.Encode(`, `Marshal`/`Unmarshal`, `strconv.*`
  calls) — every one either returns the error directly to its caller
  (`back_effect.go:81`: `return producer.ProviderImpl(...)`) or assigns and
  checks it (`resource.go:38-42`: `res, err := TransformSlice(entries); if
  err != nil {...}`; `rest.go:31`: same pattern for `Transform`;
  `producer_test.go:917`: `msgs, err := SetBackEffectCommandProvider(...)()`
  followed by `require.NoError`).
- One `_ = doorAnnounce(...)` discard exists, newly added at
  `consumer.go:1243` inside `announceActiveBackEffects`. This is not a new
  anti-pattern: it is byte-for-byte the same shape as the pre-existing
  `announceActiveJukebox` (`consumer.go:1229`: `_ = doorAnnounce(...)`), both
  documented as intentional "fails open" behavior (a late-joining session
  missing one background/back-effect announce is not worth failing the
  whole `ForSessionsInMap` fan-out over). `errcheck` under the repo's
  `default: standard` linter set (`.golangci.yml`) does not flag explicit
  blank-identifier discards by default, and this mirrors an already-passing
  sibling function, so it is not a new lint risk introduced by this range.
- `go build ./... && go vet ./... && go test ./...` run clean (no `FAIL`
  lines) in every changed module: `atlas-maps`, `atlas-messages`,
  `atlas-saga-orchestrator`, `atlas-channel`, `libs/atlas-saga`,
  `libs/atlas-constants`, `libs/atlas-packet`.

## Not evaluable

- None. Every check in the brief was settled from the diff plus the traced
  call graph (`socket/writer`, `main.go` writer registration, `saga/handler.go`)
  and module-local build/vet/test runs plus `packet-audit matrix --check`.

## Summary

### Blocking

None.

### Non-blocking (should fix / follow-up)

- `services/atlas-channel/atlas.com/channel/socket/writer/set_back_effect.go`
  and `clear_back_effect.go` (pre-existing, not touched by this range) are
  dead code with zero callers, matching the same pre-existing pattern in
  `play_jukebox.go`. File a follow-up to delete all three rather than leave
  duplicate wrappers around the live `fieldcb.New*(...).Encode` call sites in
  `consumer.go`.
- No dedicated unit test for `TransformSlice`'s empty-slice-not-nil
  guarantee; currently only covered indirectly through the JSON:API-wrapped
  HTTP test. Marginal, not required.
