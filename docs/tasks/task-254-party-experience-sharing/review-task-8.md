# Review: Task 8 — `ExperienceConfig` env-backed loader

Commit range: `e22deb1cb..b1a00f68f` (single commit `b1a00f68f`, "feat(atlas-monster-death): add ExperienceConfig with env-backed level gate settings (D9)")

Brief: `.superpowers/sdd/plan/task-8-brief.md`
Report: `.superpowers/sdd/plan/task-8-report.md`

## Scope

```
deploy/k8s/base/env-configmap.yaml                                        |   3 +
services/atlas-monster-death/.../monster/monster/config.go (new)          |  64 +
services/atlas-monster-death/.../monster/monster/config_test.go (new)     | 160 +
3 files changed, 227 insertions(+)
```

Matches the brief's file list exactly (`config.go`, `config_test.go`, `env-configmap.yaml`). `deploy/k8s/base/atlas-monster-death.yaml` correctly untouched (already `envFrom: configMapRef: atlas-env`).

## Findings

### 1. `ExperienceConfig` struct and `DefaultExperienceConfig()` — PASS

`services/atlas-monster-death/atlas.com/monster/monster/config.go:16-31` — struct fields and types (`bool`, two `uint32`, three `float64`) match the brief's interface list verbatim. Defaults match D9 exactly: `EnforceMobLevelRange=true, LevelInterval=5, LeachInterval=5, SplitCommonMod=0.8, MvpMod=0.2, PartyBonusPerMember=0.05` (config.go:24-30). Confirmed by `TestDefaultExperienceConfig` (config_test.go:5-23), which asserts all six fields and passes (`go test -run TestDefaultExperienceConfig -v` → PASS, verified live).

Exported fields with no `Builder`/`Extract` — correct per brief, which explicitly calls this out as "a config value object, not a domain model," not subject to the immutable-model convention.

### 2. `LoadExperienceConfig()` — parse/fallback behaviour — PASS

`config.go:36-58`. Only the three gate keys are read (`USE_ENFORCE_MOB_LEVEL_RANGE`, `LEVEL_INTERVAL`, `LEACH_INTERVAL`, constants at config.go:9-13). Each uses `os.LookupEnv` + `strconv.ParseBool`/`strconv.ParseUint(v, 10, 32)`, only overwriting the default on `err == nil`. Absent key: `LookupEnv` returns `ok=false`, default is kept. Unparseable value: `err != nil`, default is kept. Zero (`"0"`) parses successfully and is honoured, not treated as "unset." This is exactly the behaviour the brief specifies ("absent or unparseable key is not an error — it falls back to the default"; "0 is a valid tightening, not a parse failure").

`SplitCommonMod`, `MvpMod`, `PartyBonusPerMember` are never referenced by `os.LookupEnv` anywhere in `config.go` — confirmed by reading the full 64-line file; they are immune to env override by construction, not by convention.

### 3. Test coverage — PASS, and the tests are honest

`config_test.go` covers every row of the brief's table (no-env-is-defaults, gate disabled, gate explicitly enabled, intervals overridden, unparseable bool falls back, unparseable interval falls back, zero interval honoured, balance constants never env-driven). Ran live:

```
go test ./monster/ -run 'TestDefaultExperienceConfig|TestLoadExperienceConfig' -v
```
All 1+8 sub-tests PASS (verified in this review session, not taken from the implementer's report).

These tests are not vacuous: `TestLoadExperienceConfig` would fail without the implementation (compile error, per Step 2 of the plan), and each fallback case sets an env var to a value that would visibly diverge from the default if the fallback logic were wrong (e.g. `"maybe"` for a bool, `"abc"` for a uint) — genuinely pins the fallback-not-error contract, not just a happy path.

### 4. `deploy/k8s/base/env-configmap.yaml` — PASS

```
LEACH_INTERVAL: "5"
LEVEL_INTERVAL: "5"
REDIS_URL: "redis.home:6379"
...
TRACE_SAMPLING_RATIO: "1.0"
USE_ENFORCE_MOB_LEVEL_RANGE: "true"
```

Three new keys, all `"5"`/`"true"` matching `DefaultExperienceConfig()` and matching the `EnvEnforceMobLevelRange`/`EnvLevelInterval`/`EnvLeachInterval` constants in `config.go` verbatim (name-for-name). Alphabetical placement preserved: `LEACH_INTERVAL`/`LEVEL_INTERVAL` sort correctly between `EVENT_TOPIC_WORLD_RATE` and `REDIS_URL`; `USE_ENFORCE_MOB_LEVEL_RANGE` sorts correctly after `TRACE_SAMPLING_RATIO` (there is no later key in the file, so this is also the correct tail position). Placement matches the brief's explicit line-number instructions.

Only the three gate keys got configmap entries — `SPLIT_COMMON_MOD`, `MVP_MOD`, `PARTY_BONUS_PER_MEMBER` are correctly absent, consistent with D9's "only the first three have env keys."

### 5. env-domain guard — not implicated

`tools/env-domain-guard.sh` (envguard) forbids a domain package importing `libs/atlas-env` outside `main.go`/`kafka/`/`rest/`. `config.go` imports only `os` and `strconv` — it does not touch `libs/atlas-env` at all, so this guard does not apply to this file and is not violated.

### 6. Build/verify — PASS (module-local, per implementer scope)

```
cd services/atlas-monster-death/atlas.com/monster && go build ./... && go test ./monster/ -run 'TestDefaultExperienceConfig|TestLoadExperienceConfig' -v
```
Ran live in this review: build succeeds, all tests PASS. (Full `go test ./...` for the module was reported by the implementer as green; not independently re-run here since it is outside this task's touched surface — module-local unit build/test for the touched package is the review-relevant slice and was reproduced.)

## Not evaluable

- Nothing in scope was left unevaluated. `LoadExperienceConfig()` is not yet wired into `NewProcessor` (Task 11, out of scope for this commit per both the brief's file list and the implementer's report) — correctly not attempted here.

## Verdict rationale

Every brief requirement (struct shape, defaults, env keys, fallback semantics, configmap keys/placement, no env key for the three balance constants, test table) is satisfied and independently reproduced by running the actual tests, not by trusting the implementer's report. No blocking or non-blocking defects found.
