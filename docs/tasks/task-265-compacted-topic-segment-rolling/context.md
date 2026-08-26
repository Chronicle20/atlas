# task-265 — Implementation Context

## What this task is

The three config-status topics (`EVENT_TOPIC_CONFIGURATION_{ENVIRONMENT,SERVICE,TENANT}_STATUS-main`) carry `cleanup.policy=compact` and nothing else, so their single base-0 segment never rolls and the log cleaner never touches them — the environment topic has grown to 9.8 MB / 36 095 records in one segment. The fix is three more topic-level configs applied on both the create and the alter path.

## Key files

| Path | Role |
| --- | --- |
| `services/atlas-kafka-precreate/internal/topics/topics.go` | All behaviour. `compactCleanupPolicy` const (line 23) → four-const block + `compactTopicConfigs` + two projections + `CompactConfigNames`. `Ensure`'s create loop is at `:54-63`, its alter resource loop at `:100-113`, its two error wraps at `:116` and `:122`. |
| `services/atlas-kafka-precreate/internal/topics/topics_test.go` | All tests. `stubClient` at `:24-80` (records every request, delegates to a `*Fn` field, panics on the three group-coordinator methods). `TestEnsure_SingleCreateRequest` `:126`, `TestEnsure_AlterConfigs` `:282`, `..._NoCompactTopics` `:332`, `..._ResourceError` `:349`. |
| `services/atlas-kafka-precreate/main.go` | Alter-phase log block at `:73-79`. Already imports the `topics` package. |
| `services/atlas-kafka-precreate/README.md` | Env-var row `:21`, Create/Alter phase bullet `:29-32`. |

Module root for every `go build` / `go test`: `services/atlas-kafka-precreate` (module `atlas.com/kafka-precreate`). Nothing outside that directory is touched — no deploy overlay, no `libs/`, no `discover` change.

## Decisions carried from the design

- **The declaration is one slice, two projections.** The defect class is "a config present in one request builder and absent from the other," so the design makes that unrepresentable rather than fixing the instance. `compactTopicConfig` is a neutral local pair type, not `[]kafka.ConfigEntry`, because the alter direction needs `ConfigOperation` and `kafka.ConfigEntry` has no counterpart — one direction is a projection either way.
- **Both projections allocate per call.** `ConfigEntries` / `Configs` are per-topic fields kafka-go reads but does not document as immutable; a shared backing array across 3 topics × 4 configs buys nothing and risks silent aliasing.
- **`max.compaction.lag.ms` is the load-bearing knob; the other two are not, and the comments say so.** Live verification against `apache/kafka:4.1.1` (design §2.2) showed that config alone rolls and compacts. `segment.ms=600000` is exactly redundant given the broker's `min(segment.ms, max.compaction.lag.ms)` roll rule; `min.cleanable.dirty.ratio` was never isolated and the forced-cleaning path bypasses the ratio check. FR-4.2 requires a future reader be able to tell which is which, so the constants' doc comments state it explicitly.
- **No remediation phase.** Design §2.3 verified live that a segment whose first-message timestamp already predates the newly-applied roll bound rolls on the very next append, with the cleaner collapsing it ~30 s later. The existing unconditional alter over the whole compacted set is the entire self-healing mechanism; the plan flags it as must-not-change in Task 1 Step 4.
- **No swallowed alter errors.** No allowlist of tolerable broker rejections, no partial-application fallback. A silently-uncompacted compacted topic is the bug; it must not be reachable through a caught error.
- **Tests assert literal values, not the constants.** A test that reads the constant cannot catch someone changing the constant. Task 2 Step 3 additionally requires proving the agreement test fails when a config is removed — a vacuous set comparison would be worse than no test.
- **Log-start-offset is not a compaction signal.** The PRD's final acceptance criterion was wrong; compaction preserves surviving records' original offsets. Corrected in both the plan's Task 4 and the README's verification subsection: the cleaner-offset checkpoint is the sharp signal.

## Task shape

Four tasks. Task 1 (behaviour) touches 2 files; Task 2 (anti-regression tests) 1; Task 3 (log field + README) 2; Task 4 is the verification gate. None exceeds the 6-file / 1-service sizing bound, and none was deliberately left large.

Task 2 has no red step — its two tests guard Task 1's already-correct implementation, so they pass on first run. Step 3 of that task is the substitute: temporarily delete a config from `compactTopicConfigs`, confirm the test fails, restore.

## Verification

`tools/verify.sh` flagless must exit 0 before the branch is called done; `--quick` / `--no-docker` skip the bake and `-race` and do not count. Post-deploy checks on the live cluster are an operator step outside this branch.
