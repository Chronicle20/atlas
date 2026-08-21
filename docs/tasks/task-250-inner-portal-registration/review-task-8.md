# Review: Task 8 — portal.EnterInner (inner-portal validation and adoption)

Commit range: `a1fa2fcfb..bb38fa567`
- `b92ef4b37` feat(atlas-channel): add `portal.EnterInner` for inner-portal validation
- `bb38fa567` test(atlas-channel): assert Fh=0 on the teleport command wire

Brief: `.superpowers/sdd/plan/task-8-brief.md` (CONTROLLER AMENDMENTS authoritative)
Report: `.superpowers/sdd/plan/reports/task-8-report.md`

## Scope confirmed

Diff touches exactly the four files the brief named:
`portal/processor.go`, `portal/mock/processor.go`, `portal/processor_test.go` (new),
`movement/teleport_test.go`. `git diff --stat` shows +374/-0 across those four
files, no other files touched. Matches the report. No scope mismatch.

## Directed checks

1. **`maxPortalEntryDistance == 81`, comment cites the derivation doc.**
   PASS. `portal/processor.go:19-28` — constant is `81`, comment cites
   `docs/tasks/task-250-inner-portal-registration/structures/gms_v95.md`
   `"## Threshold derivation"` verbatim. Read that section
   (`structures/gms_v95.md:62-113`): 71 (diagonal bound of the 100×100
   default collision rect) + 10 (the client's own `ptPos.y - 10` foothold
   probe literal) = 81. Matches the controller amendment exactly.

2. **Registry MISS must not refuse; code and test must pin this.**
   PARTIAL FAIL — code is correct, test does not actually exercise a miss.
   Code: `processor.go:131-139` — `if last, ok := ...Lookup(...); ok { ... }`,
   with the distance check entirely inside the `ok` branch; a miss (`ok ==
   false`) falls straight through to the claimed-target check with no
   refusal. Correct per design §5.5.
   Test: `processor_test.go:157-158` defines the `"last-position registry
   miss"` row with neither `mutate` nor `seedRegistry` set. But the harness's
   default (`processor_test.go:192-196`) is:
   ```go
   if tc.seedRegistry != nil {
       tc.seedRegistry(t, ctx)
   } else {
       position.GetRegistry().Put(tenant.MustFromContext(ctx), enterInnerTestCharacterId, position.Position{X: 100, Y: 200})
   }
   ```
   Every row that does not supply `seedRegistry` gets a `Put` at `(100,200)`
   — exactly on the source portal — as its fallback. There is no
   "opt out of seeding" flag on the row struct, so the `"last-position
   registry miss"` row is **not** a registry miss: it seeds a real entry at
   distance 0, which happens to pass the proximity check for an unrelated
   reason (distance under threshold, not the miss short-circuit). If the
   `ok`-gated skip in `processor.go:131` were deleted, or its `ok` branch
   inverted to refuse on a genuine miss, this test would still pass, because
   it never presents the code with an actual registry miss for character 42.
   The report's claim ("verified explicitly by the last-position registry
   miss subtest") and the self-review's claim ("verified explicitly...") are
   both incorrect — no test in this diff currently proves design §5.5.
   **This is the one blocking finding.**

3. **Every refusal path logs at warning and `return nil`.**
   PASS. All five refusal sites (`processor.go:110-113`, `115-118`,
   `120-123`, `126-129`, `134-138`, `141-145` — six sites, seven counting the
   error-wrapped ones) call `p.l.Warnf` (or `WithError(err).Warnf`) followed
   immediately by `return nil`. No refusal path returns a non-nil error; the
   only path that can return non-nil is the accept path's
   `TeleportCharacter(...)` call (`processor.go:149`), matching the interface
   doc comment and PRD FR-4.6.

4. **Adopted coordinates only ever come from the server's own portal data.**
   PASS. The only call into `TeleportCharacter` is `processor.go:149`:
   `newMovementProcessor(p.l, p.ctx).TeleportCharacter(f, characterId, dp.X(), dp.Y())`
   — `dp` is the destination portal resolved from `p.pd.GetInMapByName`
   (server-side portal data), never the packet's `claimedX/Y/TargetX/Y`
   parameters. The claimed values are used only in the comparison at
   `processor.go:141` and in `Warnf`/`Debugf` log strings. Confirmed by the
   `"happy path adopts server coordinates"` test row
   (`processor_test.go:161-168`), which sends `claimedX=9999, claimedY=9999`
   and asserts the teleport target is still `(300, -50)` — the destination
   portal's coordinates.

5. **Proximity math widens to int32 before squaring.**
   PASS. `processor.go:132`: `dx, dy := int32(last.X)-int32(sp.X()), int32(last.Y)-int32(sp.Y())`, then
   `distSq := dx*dx + dy*dy` (line 133) — both operands already `int32` at
   the point of multiplication, so no `int16` overflow.

6. **Registry lookup is tenant-scoped from ctx.**
   PASS. `processor.go:131`: `position.GetRegistry().Lookup(tenant.MustFromContext(p.ctx), characterId)`.
   `position.Registry.Lookup` keys on `Key{Tenant, CharacterId}`
   (`position/registry.go:60-66`), so cross-tenant collisions on the same
   character id are not possible.

7. **`bb38fa567` asserts `Fh == 0` on the real wire body, tracing into the
   atlas-character consumer.**
   PASS. `movement/teleport_test.go`'s new `TestTeleportCharacter_EmitsFhZeroOnWire`
   uses `producertest.InstallCapturing()` (a real producer capture, not a
   mock of `TeleportCharacter` itself), polls
   `capture.Messages(movement2.EnvCommandCharacterMovement)`, JSON-decodes
   the actual `kafka.Message.Value` into `movement2.Command[any]`, and
   asserts `cmd.Fh != 0` fails the test (`teleport_test.go:127-129`), plus
   `X`, `Y`, `ObjectId`. This is the genuine wire body, not a call-arguments
   assertion against a mock. Traced the producer side:
   `movement/processor.go:98` — `CommandProducer(f, uint64(characterId),
   characterId, x, y, 0, 0)` — `fh` is the hardcoded literal `0`, exactly
   what the test pins. Traced the consumer side: `git show b1ddb4db8` in the
   main repo (`services/atlas-character/.../character/processor.go`,
   `temporal_data.go`) — `Move` routes `fh == 0` through the new
   `UpdatePosition`, which preserves the existing stored foothold, instead of
   `Update`, which would overwrite it. The test's assertion is exactly the
   value that consumer branches on. Note: `TeleportCharacter`'s `fh=0`
   literal predates this diff (landed in Task 6, per the controller
   amendment); this commit only adds the missing wire-level pin the CLAUDE.md
   cross-service-seam rule requires — consistent with the brief's Step 4
   framing as a "carried-forward follow-up," not a behavior change.

## Other findings

- `portal/mock/processor.go` gained `EnterInnerFunc` + nil-guarded method
  (`git diff a1fa2fcfb..b92ef4b37 -- .../mock/processor.go`); `var _
  portal.Processor = (*ProcessorMock)(nil)` still compiles. `WarpToPortal` is
  confirmed not part of the `Processor` interface (`processor.go:30-46`), so
  the report's claim that it needed no mock update is correct.
- `go build ./... && go test ./portal/... ./movement/...` from
  `services/atlas-channel/atlas.com/channel` passes clean (verified in this
  review, not re-trusting the implementer's report).
- "at the threshold" / "beyond threshold" arithmetic double-checked by hand:
  81² = 6561; seeded `(181,200)` → dx=81 → distSq=6561, not `>` 6561, accepted
  (correct, matches row's `expectTeleport: true`); seeded `(182,200)` → dx=82
  → distSq=6724 `> 6561`, refused (correct).
- `EnterInner` is already called from `socket/handler/portal_inner.go`
  (pre-existing, outside this diff's scope) — not reviewed here as it is not
  part of the commit range.

## Not evaluable

- None. All directed checks and the diff surface were within reach of the
  four changed files plus their two read-only contracts (`position/registry.go`,
  `structures/gms_v95.md`) and one out-of-tree consumer commit
  (`b1ddb4db8`, read via `git show` for the seam trace).

## Verdict rationale

Six of seven directed checks pass cleanly with cited evidence. Directed check
2 is a genuine test-honesty gap: the code correctly implements the
registry-miss-must-not-refuse rule, but the test the brief specified and the
report claims covers it does not actually construct a miss, due to the test
harness's blanket default seeding. This is exactly the class of defect §4
("Test honesty... A test that passes either way is a finding, not coverage")
calls out, and it is directly on the design §5.5 requirement the task brief
flagged as consequential enough to name in its own row description
("**accepted** (design §5.5: a miss must never refuse)"). It is a small,
mechanical fix — introduce an explicit `skipSeed bool` (or equivalent) on the
row struct for this one row, or clear the registry entry after the default
Put for this row — but it must land before this task is considered to satisfy
the brief.

---

verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-250-inner-portal-registration/review-task-8.md
scope_confirmed: portal.EnterInner (processor.go, mock, processor_test.go) + the carried-forward Fh=0 wire test in movement/teleport_test.go; both commits in a1fa2fcfb..bb38fa567
blocking: 1
  - services/atlas-channel/atlas.com/channel/portal/processor_test.go:155-158 — the "last-position registry miss" subtest does not produce an actual registry miss (the harness's default `seedRegistry` branch at line 192-196 always `Put`s an entry at (100,200) when the row omits `seedRegistry`), so design §5.5's "a miss must never refuse" rule is asserted by the code but pinned by no test; the row would still pass if the miss-handling branch were removed or inverted.
non_blocking: 0
not_evaluable: 0
