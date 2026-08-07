# fix(portal): stop a single portal touch from executing the rule twice (task-184)

Fixes the double-execute reported in #1193.

## Root cause

The GMS v83 client re-checks portal collision every frame while the player
overlaps a portal's rect (`CUserLocal::CheckPortal_Collision`), and the scripted-portal
path has no minimum re-send interval. The only thing stopping a re-send is the
client's `m_bExclRequestSent` flag — which the server clears by emitting
`EnableActions`.

`handleEnterCommand` emitted `EnableActions` unconditionally, including when the
matched outcome had just dispatched a warp. The flag therefore cleared while the
warp was still in flight and the player still stood inside the portal's collision
rect, so the client legitimately re-fired the ENTER request and the whole rule ran
a second time.

A second defect compounded it: `WarpToPortal` / `WarpToSavedLocation` saga steps
were marked self-completing, so **every** portal warp — including successful ones —
ran to a 30 s `SAGA_TIMEOUT`. That made `FAILED` worthless as a "the warp did not
land" signal. The acceptance-table comment asserting no `MAP_CHANGED` fires for a
same-map warp was false against `atlas-maps`, which emits it unconditionally.

## What changed — three independent layers

**Layer A — `atlas-saga-orchestrator`.** Both warp actions now complete on the
observed `MAP_CHANGED` instead of timing out, guarded by a new opt-in character-id
constraint on `AcceptEvent` (`ForCharacter`) so the `WarpPartyQuestMembersToMap`
fan-out — N warps stamped with one transaction id — cannot complete a step
belonging to a different character. The false comment is replaced with an accurate
one that records the cross-service coupling.

**Layer B — `atlas-portal-actions`.** Portal operations are classified moving/static
in a table whose zero value is invalid, so a new operation cannot silently default
to "does not move". `ExecuteOperations` reports whether a moving operation was
*successfully dispatched*, and the ENTER handler suppresses `EnableActions` on that
signal — backed by a `PendingAction` registration and a 5 s saga timeout so a
genuinely failed warp still frees the player.

**Layer C — `atlas-portal-actions`.** A fail-open Redis `SET NX` + 2 s TTL gate drops
duplicate ENTER commands before any rule evaluation. Defence in depth: the client is
*designed* to re-fire while a player stands in a portal rect, so any future
unlock-shaped regression re-opens the same hole.

## Rollout ordering

Deploy **`atlas-saga-orchestrator` first, then `atlas-portal-actions`** (design §8).
Layer B suppresses the eager unlock and relies on the saga failing to release a
player whose warp did not land; the orchestrator must already be acknowledging
`MAP_CHANGED` before that suppression goes live.

## Behaviour beyond the literal PRD wording

Three deviations, all documented in design.md §10, plus one surfaced in review:

1. **FR-2.3/2.4** condition is "a moving operation was *successfully dispatched*",
   not merely "declared", and it applies to the error branch too. Strictly safer: a
   warp that failed before creating its saga still unlocks the client, because there
   is no saga in flight to fail and release them.
2. **FR-2.5**'s `PendingAction` registration happens inside the two executor methods,
   not `handleEnterCommand` — the saga transaction id is minted there and the
   consumer never sees it.
3. **FR-3.3**'s tenant scoping is composed from `redis.TenantKey` + `redis.CompositeKey`
   because `atlas.Lock` is not tenant-aware on its own.
4. The old independent `!Allow` unlock branch is gone; the handler now unlocks purely
   on `!CharacterMoved` regardless of `Allow`. This is strictly more correct — a
   deny-rule that still dispatches a redirect warp would otherwise have been
   under-suppressed — but it fixes a case the PRD did not explicitly enumerate.

## Verification

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in both changed modules.
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` exit 0.
- `tools/lint.sh --check`: 0 issues on every Go target.
- No `go.mod` / `go.sum` delta, so no `docker buildx bake` is required.
- Code review: plan-adherence (full adherence, 0 gaps across all 24 FR/NFR entries)
  and backend-guidelines. The latter found one blocking issue — the portal action
  registry silently discarded Redis errors, and since this change makes that registry
  the sole recovery path for a suppressed unlock, a dropped write would have frozen a
  player permanently with no log trace. Fixed and re-reviewed; those failures now log
  a `Warn`, a clean miss stays silent, and four tests force real Redis errors and
  assert on captured log output.

### Pending live confirmation

These two acceptance criteria cannot be machine-checked and must be verified on
**GMS 83.1** against the `undodraco` portal from #1193:

- one touch → exactly one portal `ENTER` command executed and exactly one `MAP_CHANGED`
- no `SAGA_TIMEOUT` on a successful portal warp

## Residual risk

**Previously-dead operations become live** (design §3.4). A database-resident NPC
conversation script that batches operations *after* a `warp_to_portal` /
`warp_to_saved_location` step will now execute those trailing operations for the
first time. They are currently dead — the batch stalls at the warp step until the
30 s timeout fails the saga — so this is a bug fix, but it is worth reviewing against
live tenant data before rollout.

**5 s warp-saga timeout.** Against a ~300 ms observed end-to-end warp this is ~16×
headroom, and it is a per-saga in-process timer (not a periodic sweep), so it fires
predictably regardless of load. If warp latency ever regresses past 5 s the timer can
fire before `MAP_CHANGED` arrives, unlocking a client whose warp is about to land —
reproducing the original bug in a much rarer race window. This is observable: every
such case logs at Info with `kind=warp` and `error_code=SAGA_TIMEOUT`. The value is a
named constant (`warpSagaTimeout`), so it is a one-line change if production says
otherwise.
