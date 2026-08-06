# Backend Audit — task-191-v92-v95-movement-types (Go surface only)

- **Worktree:** `.worktrees/task-191-v92-v95-movement-types` (repo root)
- **Branch:** `task-191-v92-v95-movement-types`
- **Merge-base:** `999e48a2a`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-05
- **Build:** PASS
- **Vet:** PASS
- **Tests (race):** all packages `ok` (no FAIL)
- **Overall:** PASS

## Scope

Three Go files changed, all in `libs/atlas-packet`, a shared wire-codec library — not a service with a DDD `internal/<domain>` layout:

- `libs/atlas-packet/model/movement.go`
- `libs/atlas-packet/model/movement_test.go`
- `libs/atlas-packet/character/serverbound/version_bounds_test.go`

No `services/*` code, no `go.mod`, no k8s/deploy manifests, no `model.go`/`processor.go`/`entity.go`/`rest.go` DDD scaffolding touched. Confirmed via `.claude/skills/backend-dev-guidelines/resources/*.md`: none of them scope `libs/atlas-packet` codec files into the Domain/Sub-Domain/File-Responsibilities checklists (those target service `internal/` packages with GORM entities, JSON:API resources, and Kafka messaging). Accordingly:

- **DOM-01 through DOM-20, DOM-22 through DOM-24, DOM-26 through DOM-28, all SUB-*, all FILE-*, all EXT-*, all SCAFFOLD-*, all SEC-*:** N/A — no domain package, no service, no HTTP client, no new service scaffolding, no new atlas-channel `Writer`/`Handler` constant registered (this modifies an existing codec's field layout, it doesn't add a new packet type), no auth surface touched.
- **DOM-21** (no duplication of atlas-constants types): applies — checked below, PASS.
- **DOM-25** (client wire values config-resolved): considered and ruled not applicable — `StartVx`/`StartVy` are raw physics deltas (`int16`), not client dispatch/classification bytes routed through a lookup table; nothing here is a `Writer`/`Handler` opcode or a mode-prefix byte.

## Project-Specific Invariant Checks (per the audit brief)

| # | Check | Result | Evidence |
|---|-------|--------|----------|
| 1 | Encode/Decode gate textually identical | **PASS** | `Decode` gate: `libs/atlas-packet/model/movement.go:61` `if t.IsRegion("GMS") && t.MajorAtLeast(88) {`. `Encode` gate: `libs/atlas-packet/model/movement.go:224` `if t.IsRegion("GMS") && t.MajorAtLeast(88) {`. Identical condition text, both immediately followed by two `int16` reads/writes of `StartVx`/`StartVy` (`movement.go:62-63` vs `movement.go:225-226`). |
| 2 | New gate is `&&`-shaped (excludes JMS); `XOffset`/`YOffset` gate untouched | **PASS** | New gate at `movement.go:61` and `movement.go:224` is `t.IsRegion("GMS") && t.MajorAtLeast(88)` — excludes JMS as required. `XOffset`/`YOffset` gate remains `!t.IsRegion("GMS") || t.MajorAtLeast(88)` at `movement.go:160` (`NormalElement.Decode`) and `movement.go:260` (`NormalElement.Encode`) — confirmed byte-for-byte identical to pre-change via `git diff 999e48a2a..HEAD -- libs/atlas-packet/model/movement.go` (no hunk touches those lines; diff only adds the `StartVx`/`StartVy` field, the new gated block in `Decode`/`Encode`, and the explanatory comments). |
| 3 | Tenant resolved outside the closure, both directions | **PASS** | `Decode`: `t := tenant.MustFromContext(ctx)` at `movement.go:37`, before `return func(...)` at `movement.go:38` — matches `NormalElement.Decode`'s existing shape (`movement.go:147`). `Encode`: `t := tenant.MustFromContext(ctx)` at `movement.go:216`, before `return func(...)` at `movement.go:217`. |
| 4 | Import alias preserved | **PASS** | `movement.go:11`: `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`. |
| 5 | DOM-21 — no duplicated atlas-constants type | **PASS** | `git diff 999e48a2a..HEAD -- '*.go' \| grep -E '^\+\s*(type \|const )'` returns no matches — no new `type`/`const` declarations anywhere in the diff. The only new symbols are two struct fields (`StartVx`, `StartVy int16`) on the existing `Movement` struct; `int16` is a primitive, not a shared-lib type. |
| 6 | Repo guards (goroutine/redis) unaffected | **PASS** | `git diff 999e48a2a..HEAD -- '*.go' \| grep -E '^\+.*\bgo (func\|[A-Za-z_])'` → no matches. `git diff 999e48a2a..HEAD -- '*.go' \| grep -E '^\+.*redis'` → no matches. No bare `go` statement or keyed redis call introduced. |

## Test-Quality Review

### `TestMovementHeaderVersionBoundary` (`movement_test.go:83-112`)

- Short-header expectation (`movement_test.go:93`): `StartX=100 (0x0064 LE → 0x64,0x00)`, `StartY=200 (0x00C8 LE → 0xC8,0x00)`, count byte `0x00` → `{0x64,0x00,0xC8,0x00,0x00}`. Verified correct by hand.
- Long-header expectation (`movement_test.go:106`): adds `StartVx=5 (0x0005 LE → 0x05,0x00)` and `StartVy=-3 (int16 -3 = 0xFFFD LE → 0xFD,0xFF)` between `StartY` and the count byte → `{0x64,0x00,0xC8,0x00,0x05,0x00,0xFD,0xFF,0x00}`. Verified correct: little-endian two's-complement encoding of `-3` is `0xFD,0xFF`, matching the expected bytes exactly.
- Covers both sides of the boundary (`{GMS,83}`, `{GMS,84}`, `{GMS,87}`, `{JMS,185}` → short; `{GMS,92}`, `{GMS,95}` → long) so a broken/missing gate on either the version-check or the region-check would fail the test.

### `TestMovementHeaderRoundTrip` (`movement_test.go:116-143`)

- Exercises `Encode` then `Decode` symmetrically per `test.RoundTrip` (`libs/atlas-packet/test` helper), which fails on any unconsumed trailing bytes — this is the mechanism that would catch a one-sided gate (e.g., `Encode` writes 9 bytes but `Decode` only reads 5, or vice versa).
- Table covers GMS 87 (drop), JMS 185 (drop), GMS 92 (carry), GMS 95 (carry) and asserts both `StartX`/`StartY` preservation and the exact `StartVx`/`StartVy` round-tripped values, including the negative `-3` case. Not a vacuous "no panic" test — it asserts on real decoded field values.

### `TestMoveVersionBoundary` rewrite (`libs/atlas-packet/character/serverbound/version_bounds_test.go:75-105`)

- **v83–v87/v84-86 dr-block assertions untouched**: `version_bounds_test.go:81-94` is identical to the pre-change logic (confirmed via `git diff`, no hunk in that range) — still asserts v87 differs from v83, and v84-86 match v87, not v83.
- **`packet-audit:verify` marker line intact**: `version_bounds_test.go:74` — `// packet-audit:verify packet=character/serverbound/Move version=gms_v83 ida=0x9cb992` — present and unmodified in the diff (outside the added-lines hunk).
- **Genuinely a strengthening, not a relaxation**: the old assertion was `bytes.Equal(v95, v87)` (whole-packet equality) — per the diff removed hunk (`git diff` shows `-\tif v95 := encode(95); !bytes.Equal(v95, v87) {`). That assertion is *incompatible* with a correct fix (post-fix, v95 must legitimately differ from v87 by 4 bytes because of the new `StartVx`/`StartVy` header), so it had to change. The replacement (`version_bounds_test.go:95-104`) asserts two independent, more specific facts instead of one coarser (and now-false) one:
  1. `len(got) != len(v87)+4` — exact length delta, not just "not equal."
  2. `!bytes.Equal(got[:len(v87)], v87)` — the dr-block *prefix* still matches v87 byte-for-byte.

  A test that merely replaced `bytes.Equal` with `!bytes.Equal(v95, v87)` (accepting any difference) would have been a relaxation/disguised weakening — masking a completely wrong length or a corrupted dr-block as "pass." This rewrite does not do that: it pins the exact expected shape (unchanged dr-block + exactly 4 new bytes), so a bug that corrupted the dr-block itself, or added the wrong number of header bytes, would still fail the test. It also cross-references (`version_bounds_test.go:71-73`) `TestMovementHeaderVersionBoundary`/`TestMovementHeaderRoundTrip` in `model/movement_test.go` for the semantic (non-zero-value) verification of what those 4 bytes actually are, rather than re-deriving it — appropriate given `Move.Encode` composes `model.Movement.Encode` and doesn't own that layout itself.
  - Verdict: **genuine strengthening**, not a disguised relaxation.

## Build & Test Results (verbatim, from `libs/atlas-packet`)

```
$ go build ./...
(no output — clean)

$ go vet ./...
(no output — clean)
VET_EXIT=0

$ go test -race ./... -count=1
ok  	github.com/Chronicle20/atlas/libs/atlas-packet	1.031s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/account/serverbound	1.041s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/buddy	1.022s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/buddy/clientbound	1.040s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/buddy/serverbound	1.021s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound	1.130s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound	1.125s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/channel/clientbound	1.015s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/channel/serverbound	1.018s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/character	1.034s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound	1.212s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound/monsterbook	1.016s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound	1.066s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound/monsterbook	1.014s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/chat	1.015s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound	1.055s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/chat/serverbound	1.026s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/door/clientbound	1.019s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/door/serverbound	1.013s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/drop/clientbound	1.020s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/drop/serverbound	1.012s
?   	github.com/Chronicle20/atlas/libs/atlas-packet/fame	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/fame/clientbound	1.020s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/fame/serverbound	1.013s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/field	1.011s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound	1.164s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound	1.070s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/guild	1.010s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound	1.066s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/guild/serverbound	1.042s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/incubator/clientbound	1.010s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/interaction	1.024s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound	1.071s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/interaction/serverbound	1.072s
?   	github.com/Chronicle20/atlas/libs/atlas-packet/inventory	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/inventory/clientbound	1.035s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/inventory/serverbound	1.018s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/login/clientbound	1.048s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/login/serverbound	1.046s
?   	github.com/Chronicle20/atlas/libs/atlas-packet/merchant	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound	1.058s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound	1.021s
?   	github.com/Chronicle20/atlas/libs/atlas-packet/messenger	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/messenger/clientbound	1.031s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/messenger/serverbound	1.021s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/model	1.079s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/clientbound	1.027s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/serverbound	1.022s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound	1.063s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound	1.027s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/mount/serverbound	1.011s
?   	github.com/Chronicle20/atlas/libs/atlas-packet/note	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/note/clientbound	1.018s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/note/serverbound	1.024s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/npc	1.015s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound	1.077s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/npc/serverbound	1.035s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/party	1.017s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/party/clientbound	1.059s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/party/serverbound	1.025s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/pet/clientbound	1.028s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound	1.032s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/portal/serverbound	1.016s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/quest/clientbound	1.021s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/quest/serverbound	1.026s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/reactor/clientbound	1.020s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/reactor/serverbound	1.016s
?   	github.com/Chronicle20/atlas/libs/atlas-packet/rps	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/rps/clientbound	1.020s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/rps/serverbound	1.029s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/socket/clientbound	1.016s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/socket/serverbound	1.016s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound	1.017s
?   	github.com/Chronicle20/atlas/libs/atlas-packet/storage	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/storage/clientbound	1.030s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/storage/serverbound	1.016s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/summon/clientbound	1.023s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/summon/serverbound	1.020s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock	1.012s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/clientbound	1.013s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/serverbound	1.015s
?   	github.com/Chronicle20/atlas/libs/atlas-packet/test	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/tool	1.319s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/tv	1.013s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/tv/clientbound	1.028s
?   	github.com/Chronicle20/atlas/libs/atlas-packet/ui	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/ui/clientbound	1.019s
TEST_EXIT=0
```

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

None.

### Notes for the record

- Nothing under `services/atlas-channel/` was touched, matching the brief's stated expectation (it consumes `model.Movement` as-is and needs no change).
- No `go.mod` changed; `docker buildx bake` / service-registration guards are therefore out of scope for this diff.
- This audit only covers the Go surface. JSON templates, the bash guard, and CI YAML are explicitly out of remit per the audit brief and are covered by the parallel plan-adherence review (`docs/tasks/task-191-v92-v95-movement-types/audit.md`).
