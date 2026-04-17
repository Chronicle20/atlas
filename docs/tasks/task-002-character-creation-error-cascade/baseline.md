---
name: Phase 0 Baseline — Character Creation Error Cascade
description: Build and test pass list captured at task-002 Phase 0 so later regressions are attributable.
type: baseline
task: task-002-character-creation-error-cascade
---

# Phase 0 Baseline

Captured: 2026-04-17 on branch `deploy-reorg`.

All five services build and test green. `go build ./...` exit 0 for each; `go test ./...` exit 0 for each. Packages listed below are the ones with test files ("ok" lines); everything else was `[no test files]`.

## atlas-saga-orchestrator

```
ok  atlas-saga-orchestrator/kafka/message/compartment    0.005s
ok  atlas-saga-orchestrator/party_quest                  0.010s
ok  atlas-saga-orchestrator/saga                         0.019s
ok  atlas-saga-orchestrator/saga/mock                    0.007s
ok  atlas-saga-orchestrator/validation/mock              0.004s
```

## atlas-character-factory

```
ok  atlas-character-factory/factory                      20.717s
```

## atlas-character

```
ok  atlas-character/character                            59.406s
ok  atlas-character/session                              0.038s
```

## atlas-skills

```
ok  atlas-skills/kafka/consumer/character                0.027s
ok  atlas-skills/kafka/consumer/macro                    0.008s
ok  atlas-skills/kafka/consumer/skill                    0.027s
ok  atlas-skills/macro                                   0.019s
ok  atlas-skills/skill                                   0.055s
```

## atlas-login

```
ok  atlas-login/world                                    0.006s
```

## Notes

- No in-flight edits observed to `saga/`, `compensator.go`, `producer.go`, or the three error-swallow files (saga consumer.go:49, atlas-character consumer.go:352, atlas-login seed consumer.go).
- Working branch is `deploy-reorg` (not a dedicated task branch, by user preference).
- `go.work.sum` has uncommitted baseline-unrelated changes; untracked `docs/tasks/task-002-character-creation-error-cascade/` holds the PRD/plan/tasks/context that drive this task.
