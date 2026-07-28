# task-151-brawler-mp-recovery — Execution Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer
needs beyond the task steps themselves.

## What this task is

Brawler MP Recovery (skill 5101005) casts today but does nothing — the per-skill
registry lookup misses. This task adds the handler: lose `floor(MaxHP / x)` HP,
gain `floor(hpLost * y / 100)` MP, values from the tenant version's skill effect
data. Entirely server-authoritative (IDA-verified in the PRD — the v83 client
applies no local HP/MP change and has no low-HP gate).

## Key files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/registry.go` | `Register`/`Lookup` registry + the `Handler` type every per-skill handler implements (lines 18–24). |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | `UseSkill` generic path; per-skill dispatch at the end (line 117). **Unchanged by this task** — 5101005 has no hpCon/mpCon/duration, cooldown already applies via `e.Cooldown()`. |
| `services/atlas-channel/atlas.com/channel/skill/handler/mysticdoor/` | The structural template: `init()` registration + package-level `var` seams overridden in tests (`mysticdoor_test.go` `invokeApply` pattern). |
| `services/atlas-channel/atlas.com/channel/skill/handler/heal/` | Formula-in-own-file precedent (`formula.go`) and the registry `Lookup` registration test (`heal_test.go:12`). |
| `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` | Blank-import list that makes handler `init()`s run in production (`main.go:58` imports it). |
| `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` | Effect model. `y int16` field exists, populated from REST (`rest.go:115`); only the `Y()` getter is missing. `X()` exists (line 144). |
| `services/atlas-channel/atlas.com/channel/data/skill/effect/rest.go` | `RestModel` (`X`/`Y int16`, json `"x"`/`"y"`) and `Extract(RestModel) (Model, error)` — the only cross-package way to build a `Model` with x/y set (no builder exists); tests use it. |
| `services/atlas-channel/atlas.com/channel/character/processor.go` | `ChangeHP(f, characterId, amount int16) error` (line 271), `ChangeMP` (line 275), `GetById()(id)`. Fire-and-forget Kafka command emissions to atlas-character. |
| `services/atlas-channel/atlas.com/channel/character/model.go` | `MaxHp() uint16` (line 135). |
| `libs/atlas-constants/skill/constants.go` | `BrawlerMPRecoveryId = Id(5101005)` (line 3194) — already exists, do not redefine. |

## Locked decisions (do not relitigate)

- **No low-HP guard** (owner decision, PRD FR-3): handler emits the full
  `-hpLost` with no knowledge of current HP; atlas-character's ChangeHP owns
  the 0-floor/death path. No clamp, no rejection.
- **mpGain from intended loss** (PRD FR-2 / design alt D): computed from the
  full `hpLost`, never a post-clamp actual delta.
- **Mysticdoor-style seams, not heal-style direct calls** (design §2.2/alt B):
  three `var` seams (`loadCaster`, `changeHP`, `changeMP`), test overrides with
  `t.Cleanup` restore.
- **Caster load failure returns the error with zero emits** (FR-5) — this
  deliberately differs from Heal, which swallows the load error. `UseSkill`
  logs-and-swallows handler errors (`common.go:118-120`), so returning is
  observability-only; nothing is rolled back (cooldown already applied,
  matching Cosmic).
- **ChangeHP error skips ChangeMP** (forbidden partial = gain without cost).
  ChangeMP error after a successful ChangeHP request is an acceptable partial
  (cost-without-gain), just logged + returned.
- **No AnnounceSkillUse/foreign broadcast** in the handler — the generic path
  already covers the plain stat-change cast broadcast (design §2.2).
- **No new packets/REST/Kafka topics/migrations; atlas-channel only.**

## Formula ground truth (tests only — never hardcode in production code)

WZ v83 `Skill.wz/510.img.xml`, verified during PRD: max level 10; `x = 10` at
every level; `y` = 55 (L1), 75 (L5), 100 (L10); cooltime 70s→25s. Worked
example: MaxHP 1234 → hpLost 123; L1 mpGain 67, L5 92, L10 123.

## Dependencies between tasks

- Task 1 (`Y()` getter) and Task 2 (`Amounts`) are independent of each other.
- Task 3 (handler) needs both: calls `Amounts(maxHp, e.X(), e.Y())`.
- Task 4 (blank import + gates) needs Task 3's package to exist.

## Verification

From `services/atlas-channel/atlas.com/channel/`: `go build ./...`,
`go vet ./...`, `go test -race ./...` — all clean.
From the worktree root: `tools/redis-key-guard.sh` (no `GOWORK=off` prefix).
`docker buildx bake` not required — no `go.mod` change.

After implementation: run `superpowers:requesting-code-review` before any PR
(mandatory; reviewers write to `docs/tasks/task-151-brawler-mp-recovery/audit.md`).

## Gotchas

- Test files must not be `*_testhelpers.go`; in-package tests may set
  unexported fields directly (Task 1 does `Model{y: 55}`), cross-package tests
  build effect models via `effect.Extract(effect.RestModel{...})`.
- `registrations.go` import block is alphabetical and gofmt-aligned — run
  `gofmt -w` after adding the mprecovery line.
- The registration test lives in package `mprecovery`, so `init()` runs for the
  test binary without the blank import; the Task 4 blank import is what makes
  it live in *production* (`main.go` → registrations).
- Handler errors returned to `UseSkill` are logged there and swallowed — do not
  add retry or compensation logic.
