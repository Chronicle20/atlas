# Task 164 — Summon Temporary Stats (PUPPET/SUMMON) — Execution Context

Companion to `plan.md`. Key files, decisions, and dependencies for implementers.

**v2 (2026-08-07)** — rebased onto main at `e0f5bd01d`; line numbers re-verified;
version coverage widened to all 12 supported tenant versions.

## What this task is

`PUPPET`/`SUMMON` statups (produced by atlas-data for summon-type skills) reach
`CharacterTemporaryStat.AddStat` in `libs/atlas-packet`, fail the registry lookup, and
log `Attempting to add buff [PUPPET], but cannot find it.` at ERROR on every CTS encode
path in atlas-channel. IDA verification (PRD §1.1 — a 10-IDB sweep re-run 2026-08-07)
proved no supported client has a SecondaryStat bit for these stats — they are
server-side lifecycle bookkeeping, and summon visibility is carried by the summon
object packets (task-088/106). The fix: classify them as **server-only** in the packet
layer and skip them silently (DEBUG).

## Key files

| File | Role |
|---|---|
| `libs/atlas-packet/model/character_temporary_stat.go` | The only production file changed. `AddStat` (line 580) gets a `serverOnlyStatNames` check before the registry lookup; the set is declared above it, mirroring the `baseStatNames` idiom (line 971). Registry builder at line 63 — do not touch. |
| `libs/atlas-packet/model/character_temporary_stat_test.go` | All new tests land here (4 new test functions). Existing byte fixtures in this file are the FR-8 registry-invariance proof and must pass unmodified. |
| `libs/atlas-constants/character/temporary_stat.go:123-124` | `TemporaryStatTypeSummon` (`"SUMMON"`) / `TemporaryStatTypePuppet` (`"PUPPET"`) constants — consumed, not changed. |
| `libs/atlas-packet/test/context.go` | `pt.Variants` — **the canonical supported-version list (12 entries)** every new test must loop: GMS v28, v48, v61, v72, v79, v83, v84, v86, v87, v92, v95, JMS v185. Also `pt.CreateContext(region, major, minor)`. `testlog "github.com/sirupsen/logrus/hooks/test"` is already a module dependency (used in `roundtrip.go`). |
| `services/atlas-channel/.../socket/writer/character_buff_give.go:22,37`, `character_buff_cancel.go:22,37`, `character_spawn.go:34` | The four encode paths. **Read-only context** — they all delegate to `AddStat` and emit unconditionally, which is why zero atlas-channel changes are needed. |
| `services/atlas-channel/.../socket/writer/monster_spawn.go:20`, `.../kafka/consumer/monster/consumer.go:475,509,543` | The other four `AddStat` call sites repo-wide. These are `MonsterTemporaryStat.AddStat` — a **different receiver with its own registry**, explicitly out of scope. Do not let the skip reach them. |

## Load-bearing decisions (from design.md — do not relitigate)

1. **Name set in `AddStat`, not a registry flag** (design §2 vs §3.1): `buildCharacterTemporaryStatRegistry` is a shift-ordered enumeration — position == mask bit. It must not be touched; a registry-flag approach needs five filter sites in FR-8-critical code.
2. **Packet layer, not atlas-constants and not writers** (design §3.2/§3.3): FR-2 places the classification in `libs/atlas-packet`. Exposing a `ServerOnly()` domain method would invite upstream filtering, which would disturb summon lifecycle bookkeeping.
3. **Skip logs `Debugf` on every occurrence** — no first-add-only state.
4. **Emission is NOT skipped** (FR-5, owner decision 2026-07-10): a pure PUPPET/SUMMON buff still emits an empty-mask packet. This is already writer behavior; the lib change does not alter it.
5. **Unknown-name ERROR stays intact** (FR-3): only the two named stats skip.

## Test gotchas (verified against current code)

- **Nil loggers:** existing tests pass `nil` to `AddStat` — safe only because the happy
  path never logs. The skip path calls `l.Debugf`, so server-only tests must pass a real
  logger (`testlog.NewNullLogger()`), with `l.SetLevel(logrus.DebugLevel)` when asserting
  the DEBUG skip entries (null logger defaults to Info, which would swallow them).
- **Duration bytes are wall-clock-dependent — use `time.Time{}`, not a real expiry:**
  the self `Encode` path derives each per-stat duration from `expiresAt` at encode time,
  so two `Encode` calls on equivalent CTS can differ. Passing the **zero time** saturates
  that field to a constant on both the modern and legacy writers (pinned by
  `TestNoExpiryStatEncodesSaturatedDuration` / `TestLegacyDurationUnitsNoExpirySaturates`),
  making full-slice byte comparison deterministic on every version.
  **Plan v1 instead skipped bytes 22..25 — that is superseded and must not be
  reintroduced:** the offset assumes a 16-byte mask and is wrong on the legacy class
  (GMS `< 61`), where `EncodeMask` writes only `WriteLong(mask.L)` — 8 bytes.
- **Mask width is version-dependent.** Seven registry classes exist (PRD FR-7.1);
  gate boundaries in `character_temporary_stat.go`: `legacyGmsMask` GMS `< 61` (625),
  GMS `== 61` (1010), `MajorAtLeast(84)` (775), `MajorAtLeast(87)` (107, 178),
  `MajorAtLeast(95)` (785, 1030), `Region() == "JMS"` (177). Never hard-code a byte
  offset in a test that loops `pt.Variants`.
- **Booster is the wire-stat probe** in the mixed-buff test because it is registered
  unconditionally (shift 11, ahead of every gate), so it exists in all seven classes and
  falls inside the legacy mask's bits 0-46.
- **Byte-equality alone is not the failing TDD signal:** pre-change, the failed lookup
  already drops the stats, so encodes are already byte-identical. The failing-first
  assertions are the log-level ones (ERROR fires today, DEBUG expected after).
- Existing fixtures that must pass unmodified (FR-8) are tabulated by registry class in
  `plan.md` Task 1 Step 4 — they span legacy/v61, mid-GMS, v95, and all-variant
  round-trips.

## Dependencies & scope guard

- Changed module: `libs/atlas-packet` only. No service `go.mod` touched → no
  `docker buildx bake` required. If any `services/` file changes, the scope audit in
  plan Task 3 fails the branch.
- Explicitly untouched: `services/atlas-data` (statup production),
  `services/atlas-buffs` (lifecycle), `services/atlas-summons`, all atlas-channel writers,
  `buildCharacterTemporaryStatRegistry`, the mob temporary-stat registry
  (`libs/atlas-packet/model/monster.go` lookup-failure log is out of scope), and
  `CTS_SummonBomb`/mob-skill stats.
- No REST/Kafka/tenant-config surface changes; nothing config-resolved changes (DOM-25
  not implicated).

## Verification bar (plan Task 3)

`go test -race ./...`, `go vet ./...`, `go build ./...` clean in `libs/atlas-packet`;
`tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and `tools/lint.sh --check`
clean from the worktree root (no `GOWORK=off` prefix — and source nvm/Node 22 before
`lint.sh` or its atlas-ui half false-fails); scope audit via
`git diff --stat main...HEAD -- . ':!docs/tasks'` shows exactly the two lib files.
Runtime acceptance (live-log grep on a v83 tenant) is post-merge, out of the automated
plan.
