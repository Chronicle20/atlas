# task-276 — implementation context

Companion to [plan.md](plan.md). Records the decisions, measurements, and
sizing judgements the plan's task bodies assume.

## Key files

| File | Role |
|---|---|
| `libs/atlas-kafka/topic/provider.go:14-24` | `EnvProvider`; the fallback at lines 17-21 is what makes an unmanaged token invisible |
| `libs/atlas-kafka/producer/manager.go:66,133` | `Manager.Writer` / `ManagerWriterProvider` — the token-carrying producer seam |
| `libs/atlas-kafka/producer/provider.go:10` | `producer.Provider` — the seam every service `Emit` calls |
| `libs/atlas-outbox/bridge.go:21,37,42` | `EnqueueBuffer`; keys by token, resolves at 37, persists the **resolved** name at 42 |
| `libs/atlas-service/envregistry.go:52,59`, `projection.go:67,68` | the four raw `os.Getenv` topic reads — a third token shape neither the declaration sweep nor the `EnvProvider` sweep captures |
| `services/atlas-kafka-precreate/main.go:57` | the single `discover.FromEnviron` seam |
| `services/atlas-kafka-precreate/internal/discover/discover.go:12-30` | prefixes + `compactVars` + the compaction rationale comment moved to `policies.yaml` |
| `deploy/k8s/base/env-configmap.yaml` | 174 topic keys, lines 21-196, split by `DB_HOST`/`DB_PORT` at 103-104 |
| `deploy/k8s/overlays/{main,pr,pr-sparse}/kustomization.yaml` | 174 literals each, at 60-233 / 180-353 / 343-516 |
| `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh` | 25 lines; its `^(COMMAND\|EVENT)_TOPIC_` selector is what silently drops the two `STATUS_*` tokens |
| `deploy/k8s/base/atlas-kafka-precreate.yaml:55-60` | the envFrom-only comment — the Job must never grow an `env:` key |
| `tools/atlasguards/guards.go:36-64` | the five-analyzer registration `topicguard` joins |
| `tools/go-analyzer-guards.sh:51-58,62` | `GUARD_SRCS` and `SELFTEST_GUARDS` |
| `tools/verify.sh:374` (`.go` block), `:566-580` (drift-step pattern) | the two gate insertion points |

## Decisions carried from design.md

1. **The type alone cannot enforce FR-1.4.** Go assigns an untyped string
   constant to any string-kind named type, so `put("EVENT_TOPIC_X")` still
   compiles after `type Token string`, and the generator — which finds tokens
   by type — cannot see it. Hence `tools/topicguard`.
2. **Scan and gate are split.** The generator is the writer, on a narrow
   non-Go trigger; the analyzer is the gate, riding the vet sweep that already
   type-checks `services/` and `libs/` once per run. A standalone
   `go/packages NeedTypes` load on every `.go` change would re-pay ~46s and
   blow the 30s NFR budget.
3. **Cleanup policy lives in `policies.yaml`, not a second type.** Policy is a
   property of the Kafka topic, not of a Go declaration; two services
   declaring the same token are naming one topic, and letting each declare a
   policy creates a conflict with no correct resolution. This makes FR-2.6
   vacuous — the plan deliberately carries **no code** for it.
4. **`libs/atlas-kafka/consumer.NewConfig` is NOT retyped**, contrary to
   FR-1.3's literal text. It takes the *resolved* topic name, not a token.
   Retyping it would make `Token` mean two things and destroy the analyzer's
   ability to tell them apart.
5. **`verify.sh`'s drift step is NOT triggered on `\.go$`**, contrary to
   FR-5.2's literal text. `topicguard`'s `token-not-in-manifest` diagnostic
   covers the same failure inside the already-running sweep, satisfying
   FR-5.2's own escape clause ("gate the expensive load behind a cheap
   pre-check") with something strictly stronger than a grep.
6. **`deploy/compose/.env.example` becomes generated** — the one true scope
   addition beyond PRD §7. It is a live surface carrying 15 of the 17 orphans
   and missing ~72 real tokens, with no gate watching it. Dropping it costs
   one renderer and one path in the verify trigger; nothing else changes.
7. **`gen-topic-config.sh` is deleted, not widened.** FR-3.4 permits either.
   Its 2026-08-20 `atlas-login` crash-loop rationale is carried verbatim into
   the generator's `overlaySuffixes` table — that comment is the only record
   of why `pr-sparse` must use the baseline's suffix rather than no suffix.

## Sizing decisions

**Task 4 is deliberately large.** It touches ~390 files across 64 service
modules plus four library seams — far past the >6-file / >1-service split
rule. Splitting it is not possible: `topic.Token` is a defined type, so the
declarations and the seams that consume them must change in one compile unit
or the workspace does not build. Splitting it per service would leave the
tree red across every intermediate task, which is worse than one large task.

It is sized to the implementer budget anyway, because it is **tool-driven**:
two `topicmod` invocations, four hand edits at named `file:line`s, one build
sweep. The discovery that would normally inflate such a task was done in
Tasks 2 and 3, where the rules were written and tested against fixtures.

Tasks 2 and 3 exist because of `docs/codemod-vs-agents.md`: 346 declaration
files, 42 buffer types, 61 `NewConfig` wrappers, and 336 error sites are the
same templated transformation repeated, which clears the second-dispatch
threshold decisively. The rewriter honours both of that document's contracts —
every site is rewritten or listed in a residue report, never silently skipped,
and `-check` exits non-zero while any un-migrated site remains, so it becomes
the regression guard afterward.

### plan-lint F4 warnings, deliberately accepted

`tools/plan-lint.sh` reports three advisory F4 warnings; all three are
accepted, and none is a hidden `PARTIAL` risk:

- **Task 2 "spans 2 services"** — a false positive. The task creates only
  `tools/topicmod`; the two service paths in its `Files` block are read-only
  references naming the canonical R1 and R2 *input* shapes so the implementer
  does not have to hunt for a fixture source.
- **Task 7 lists 7 files** — five are the four deploy surfaces plus the
  deleted `gen-topic-config.sh`, which cannot be split from the renderers that
  subsume it: deleting the script and generating what replaced it is one
  reviewable unit. The other two are read-only guard scripts confirmed
  unaffected.
- **Task 9 lists 7 files** — four are `testdata/` fixtures for a single
  `analysistest` run plus the analyzer and its test. The `atlasguards` /
  `go-analyzer-guards.sh` registration is two small edits that must land with
  the analyzer or the sweep does not run it.

## Known intervals and residual risks

- **Between Task 1 and Task 4 the branch is runtime-red.** `EnvProvider`
  starts returning errors in Task 1; the 325 call sites that discard them are
  not fixed until Task 4. Every task in between compiles and tests green —
  this is a within-branch interval, never a merged state.
- **The `EnvProvider` error contract is a live behaviour change.** Two tokens
  (`STATUS_TOPIC_CASH_ITEM`, `STATUS_EVENT_TOPIC_SKILL_MACRO`) are the only
  ones absent from `env-configmap.yaml` today, so removing the fallback
  changes observable behaviour for those two only — and Task 6 adds them to
  the ConfigMap in the same branch.
- **`libs/atlas-service`'s four reads stay optional.** Unset means "degrade to
  legacy single-environment mode", not fatal, so they keep `os.LookupEnv`
  rather than moving to the now-fatal `EnvProvider`. Declaring the tokens is
  what satisfies `raw-env-topic-read`, which fires on a *literal* argument.
  Changing those four to fatal would be a behaviour change outside this
  task's scope.
- **`outbox_entries` in-flight window.** Unsent rows name the *resolved* topic
  and would drain to the old name. `AllowAutoTopicCreation: true` means they
  drain rather than error, and neither renamed token is produced through the
  outbox bridge today, so the window is empty in practice. Recorded in
  `migration.md`, not mitigated.
- **Orphan names are never transcribed.** design.md gives counts (17 removed,
  2 added), not names. Task 12 recovers the exact lists with `comm` against
  the merge-base and `topics.yaml`; nothing in this plan names an orphan.
- **`-check` wall time is unmeasured.** The design budgets 30s (NFR §8) and
  the trigger is narrow, but no measurement exists yet. Task 8 Step 5 records
  the number; if it is over budget, the fallback (hash the Go token
  declarations before loading) is designed but deliberately not built —
  measure first.

## Dependencies between tasks

```
1 (Token + EnvProvider)
  └─ 2 (topicmod: type rules) ─ 3 (topicmod: error rules) ─ 4 (run it)
       └─ 5 (gen: scan + topics.yaml)
            ├─ 6 (render base ConfigMaps) ─ 7 (render overlays + compose)
            │                                  └─ 8 (gen-topics.sh + verify.sh)
            ├─ 9 (topicguard; needs topics.yaml to exist)
            └─ 10 (precreate manifest pkg) ─ 11 (rewire + mount)
                                                └─ 12 (migration.md + full verify)
```

Task 5 depends on Task 4: the generator collects tokens **by type**, so it
finds nothing until the retype has landed. Task 9's third diagnostic depends
on Task 5 for the same reason. Tasks 6/7 and 10/11 are independent of each
other and could run in parallel if desired.
