# Code-Derived Kafka Topic Manifest — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28
Issue: [#1464](https://github.com/Chronicle20/atlas/issues/1464)
---

## 1. Overview

Atlas's Kafka topic set is never declared anywhere. It is inferred at runtime, twice over. Services name their topics as string constants in Go (`EnvCommandTopic = "COMMAND_TOPIC_FAME"`), pass them to `topic.EnvProvider`, and get back whatever `os.LookupEnv` returns — or, if the variable is unset, the token itself. Independently, the sync-wave-0 `atlas-kafka-precreate` Job scrapes its *own* process environment for variables whose names begin with `COMMAND_TOPIC_` or `EVENT_TOPIC_` (`services/atlas-kafka-precreate/internal/discover/discover.go:14-15`) and pre-creates the resulting names. The only thing joining "topics the code uses" to "topics the environment pre-creates" is a prefix convention enforced by a `strings.HasPrefix` pair and a hand-maintained ConfigMap.

Nothing type-checks that join, and its failure mode is silent. A topic named by a variable that does not match the prefix is simply never pre-created; nothing errors, and the symptom surfaces later as the exact consumer-wedge stampede wave 0 exists to prevent. **This is not hypothetical — the repository has two live instances today.** `STATUS_TOPIC_CASH_ITEM` (`services/atlas-cashshop/atlas.com/cashshop/kafka/message/item/kafka.go:5`, produced to from `cashshop/inventory/asset/processor.go:123,180,232,318`) and `STATUS_EVENT_TOPIC_SKILL_MACRO` (`services/atlas-skills/.../kafka/message/macro/kafka.go:11`, consumed by `services/atlas-channel/.../kafka/consumer/macro/consumer.go:29`) both match neither prefix, appear in **no** ConfigMap in `deploy/`, and therefore resolve through `EnvProvider`'s warn-and-fall-back path to the literal token as the topic name. They are never pre-created, and because the fallback skips the per-environment suffix that `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh` appends to every real topic, every environment on the cluster shares those two topics. The inverse drift exists too: 17 `*_TOPIC_*` keys in `deploy/k8s/base/env-configmap.yaml` (`COMMAND_TOPIC_INVENTORY`, `COMMAND_TOPIC_WZ_EXTRACTION`, `EVENT_TOPIC_SKILL_MACRO`, …) correspond to no topic token in any Go source, and are pre-created in every environment for nothing.

This task makes the topic set a **declared, generated, checked artifact**. A generator statically analyses the topic-token call sites across the `go.work` modules and emits a manifest; the manifest generates `deploy/k8s/base/env-configmap.yaml`'s topic block and a mounted ConfigMap that `atlas-kafka-precreate` consumes instead of scraping `os.Environ()`; and a `--check` drift gate in `tools/verify.sh` turns any divergence between code, manifest, and deployment into a CI failure. Cleanup policy moves from a hardcoded `case`-equivalent (`discover.go:26-30`) into manifest data, and the token type moves from bare `string` to a defined `topic.Token` so an undeclared token at a call site fails the gate rather than the cluster.

The CPU cost of the old shell-driven Job was solved separately by task-260, which ported the scrape faithfully and explicitly deferred this change (task-260 prd.md §2, non-goals). This is the deferred half: removing a silent-failure class, not a performance change.

## 2. Goals

Primary goals:

- Derive the complete Kafka topic-token set from Go source by static analysis of the topic call sites, into a checked-in manifest.
- Generate the topic block of `deploy/k8s/base/env-configmap.yaml` from that manifest, so code and deployment cannot diverge.
- Have `atlas-kafka-precreate` consume the manifest from a mounted ConfigMap instead of scraping `os.Environ()` for name prefixes.
- Carry cleanup policy (`compact` vs. default) as manifest data rather than as a hardcoded name list in Go.
- Introduce a defined `topic.Token` type so a topic token is a declared thing the generator can find, not an arbitrary string appearing at a call site.
- Fail the build (`tools/verify.sh`) on any drift between code, manifest, and rendered deployment.
- Resolve the two live unmanaged tokens (`STATUS_TOPIC_CASH_ITEM`, `STATUS_EVENT_TOPIC_SKILL_MACRO`) and the 17 orphan ConfigMap keys as part of the migration.

Non-goals:

- Changing topic *names* or the per-environment suffix scheme. `gen-topic-config.sh` keeps appending the environment token; the manifest carries tokens, never rendered names.
- Changing consumer-group naming or `libs/atlas-kafka/consumergroup`. The issue notes the related problem; it is deliberately left for a follow-up task (see §9).
- Changing the Kafka admin behaviour delivered by task-260: batch `CreateTopics`, offset seeding, skip rules, exit-code contract, wave-0 position, `activeDeadlineSeconds`, partition count, replication factor.
- Changing `libs/atlas-kafka`'s producer/consumer runtime semantics beyond the token type and the removal of the unset-token fallback.
- Introducing a runtime topic-registration API. Registration was considered and rejected: every service is a separate Go module under `go.work`, so a runtime registry still requires a build-time cross-module scan to collect, while additionally touching every service.

## 3. User Stories

- As a **platform operator**, I want a topic that code publishes to but the environment never pre-created to fail CI, so that I never diagnose it as a consumer-wedge stampede at 2am.
- As a **developer adding a Kafka topic**, I want the ConfigMap entry generated from my token declaration, so that I cannot forget the deploy-side half of the change.
- As a **reviewer**, I want the topic set to appear as a reviewable diff in a manifest file, so that "this PR adds a topic" is visible without reading kustomize output.
- As a **developer**, I want a mistyped or ad-hoc topic token to be a build failure, so the prefix convention stops being load-bearing infrastructure implemented as a grep pattern.
- As an **on-call engineer**, I want a service whose topic variable is unset to fail loudly at boot instead of silently publishing to an unsuffixed topic shared with every other environment.

## 4. Functional Requirements

### 4.1 Token type

- **FR-1.1** `libs/atlas-kafka/topic` MUST define `type Token string`.
- **FR-1.2** Every topic-token constant in `services/` and `libs/` (the `Env*Topic*` consts in `kafka/message/**/kafka.go`, ~163 declarations) MUST be declared with an explicit `topic.Token` type, e.g. `EnvCommandTopic topic.Token = "COMMAND_TOPIC_FAME"`.
- **FR-1.3** The functions that accept a token MUST take `topic.Token` rather than `string`: `topic.EnvProvider`, `libs/atlas-kafka/consumer`'s config constructors, the producer manager (`libs/atlas-kafka/producer/manager.go:67`), and the message-buffer `Put` path. Per-service `NewConfig(l)(name)(token)(groupId)` wrappers MUST be retyped to match.
- **FR-1.4** The generator (FR-2) MUST reject any topic token that reaches one of those functions without originating from a `topic.Token`-typed constant declaration — an inline string literal at a call site is a gate failure, not a manifest entry.
- **FR-1.5** `topic.EnvProvider` MUST NOT fall back to returning the token when the environment variable is unset. It MUST return an error naming the token. (Today's fallback at `libs/atlas-kafka/topic/provider.go:19-21` is what makes an unmanaged token invisible.)
- **FR-1.6** Call sites that currently discard the `EnvProvider` error (`t, _ := topic.EnvProvider(l)(token)()`, ~30 sites) MUST propagate or fatally log it. A service MUST NOT start a consumer or producer against an unresolved topic.
- **FR-1.7** Test-only tokens in `_test.go` files (`EVENT_TOPIC_TEST`, `EVENT_TOPIC_FAKE`, `EVENT_TOPIC_X`, `EVENT_TOPIC_UNSET`, `EVENT_TOPIC_PROVIDER_TEST`, `EVENT_TOPIC_PROVIDER_ENV_TEST`, `ANY_TOPIC`, `MY_TOPIC`, `RACE_TOPIC`, `TEST_TOPIC`) MUST be excluded from the manifest. Exclusion MUST be by build-tag/file-suffix (`_test.go`), not by a name denylist.

### 4.2 Generator

- **FR-2.1** A Go generator MUST live at `libs/atlas-kafka/gen` as its own module, mirroring the established `libs/atlas-constants/gen` pattern (`go run .` to regenerate, `go run . -check` to fail on drift).
- **FR-2.2** The generator MUST load every module in the repository's `go.work` via `golang.org/x/tools/go/packages` with type information, and collect every `topic.Token`-typed constant, resolving its value by constant folding.
- **FR-2.3** The generator MUST fail (non-zero exit, named diagnostic) if any package fails to type-check, rather than emitting a manifest derived from a partial load. A partial load is exactly the silent gap this task removes.
- **FR-2.4** The generator MUST emit a checked-in manifest at `libs/atlas-kafka/gen/topics.yaml` containing, per token: the token name, the declaring package path(s), and the cleanup policy. Tokens MUST be sorted for a stable diff.
- **FR-2.5** The generator MUST record every declaring package for a token shared by multiple services (e.g. `COMMAND_TOPIC_CHARACTER` is declared in many), so the manifest doubles as an ownership map.
- **FR-2.6** The generator MUST warn-and-fail on a token declared with the same name but a conflicting cleanup policy.
- **FR-2.7** Cleanup policy MUST be sourced from a checked-in policy file, `libs/atlas-kafka/gen/policies.yaml`, listing tokens whose topics require `cleanup.policy=compact`. The generator MUST fail if a policy entry names a token the scan did not find (a stale policy is drift too). The three config-projection tokens currently hardcoded at `discover.go:26-30` MUST move here, with their existing rationale comment carried across verbatim.
- **FR-2.8** `-check` MUST exit non-zero with a readable diff when any generated artifact is stale, and MUST NOT write files.

### 4.3 Generated deployment artifacts

- **FR-3.1** The generator MUST render the `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` / `STATUS_*_TOPIC_*` block of `deploy/k8s/base/env-configmap.yaml` from the manifest, preserving today's identity mapping (`KEY: "KEY"` — the per-environment suffix is applied by the overlays, not here).
- **FR-3.2** The generated block MUST be delimited by begin/end marker comments so the generator rewrites only that region; `ATLAS_ENVIRONMENT`, `BASE_SERVICE_URL`, `BOOTSTRAP_SERVERS`, and the hand-written commentary above them MUST be preserved byte-for-byte.
- **FR-3.3** The generator MUST emit a second artifact, `deploy/k8s/base/kafka-topics-configmap.yaml`, defining a ConfigMap (`atlas-kafka-topics`) whose single key carries the manifest — the token list plus per-token cleanup policy — in the form the precreate tool consumes.
- **FR-3.4** `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh` MUST continue to read `env-configmap.yaml` unchanged. Its `test("^(COMMAND|EVENT)_TOPIC_")` selector MUST be widened (or replaced by the manifest) so that a token outside those two prefixes is still suffixed per environment; the two `STATUS_*` tokens are currently dropped by that selector.
- **FR-3.5** The inline topic lists in `deploy/k8s/overlays/{main,pr,pr-sparse}/kustomization.yaml` MUST be regenerated from the manifest as part of the same pass, or verified against it, so an overlay cannot drift from base.

### 4.4 Precreate consumption

- **FR-4.1** `atlas-kafka-precreate` MUST read the topic set from the mounted manifest, not from `os.Environ()` prefix matching. `discover.FromEnviron` and its prefix constants MUST be deleted, not merely bypassed.
- **FR-4.2** The Job spec MUST mount the `atlas-kafka-topics` ConfigMap at a fixed path; the mount path MUST be configurable by environment variable with a default, and an unreadable or malformed manifest MUST be a fatal startup error naming the path.
- **FR-4.3** For each token in the manifest, the tool MUST resolve the topic name via the process environment (still injected by `envFrom: atlas-env`). Resolution stays env-based because names carry the per-environment suffix, which the manifest deliberately does not encode.
- **FR-4.4** A manifest token with no corresponding environment variable, or an empty value, MUST be a **fatal** error naming the token, failing wave 0 before any Deployment starts. Because `env-configmap.yaml` is generated from the same manifest (FR-3.1), this condition can only mean a real deployment defect. *(See §9 — sparse/ephemeral overlays are the case to confirm during design.)*
- **FR-4.5** Cleanup policy MUST be taken from the manifest, per topic. A topic classified compact MUST be created with `ConfigEntries` carrying `cleanup.policy=compact`, exactly as today.
- **FR-4.6** All other task-260 behaviour MUST be preserved bit-for-bit: single batched `CreateTopics`, `TopicAlreadyExists` tolerated via `errors.Is`, de-duplication, compaction-wins classification, the settle pass, consumer-group seeding, skip rules, and the exit-code contract.
- **FR-4.7** `discover.Groups` and `discover.StateIsSeedable` MUST be left unchanged. Only the topic-discovery half of the package is replaced.

### 4.5 Drift gate

- **FR-5.1** `tools/verify.sh` MUST gain a `topic manifest drift` step invoking the generator's `-check`, following the existing `gen-lb-ports.sh --check` / `gen-routes.sh --check` / `gen-tenant-tables.sh --check` step pattern.
- **FR-5.2** The step MUST fire when any of these change: any Go file under `services/` or `libs/`, `libs/atlas-kafka/gen/**`, `deploy/k8s/base/env-configmap.yaml`, `deploy/k8s/base/kafka-topics-configmap.yaml`, or the overlay kustomizations. Given the breadth of the Go trigger, the step MUST be fast enough to run on nearly every branch — the design MUST measure it and, if the full `go/packages` load is too slow for the common path, gate the expensive load behind a cheap pre-check.
- **FR-5.3** The `--check` invocation MUST NOT require Docker or a broker.

### 4.6 Migration

- **FR-6.1** `STATUS_TOPIC_CASH_ITEM` and `STATUS_EVENT_TOPIC_SKILL_MACRO` MUST enter the manifest, gain generated `env-configmap.yaml` entries, and therefore become per-environment-suffixed and pre-created. The change of topic name from the unsuffixed literal to the suffixed name MUST be called out in the PR: in-flight messages on the old unsuffixed topics are abandoned, which is acceptable for a status-event topic read at `LastOffset` but MUST be confirmed for the cashshop producer during design.
- **FR-6.2** The 17 `env-configmap.yaml` keys with no corresponding token in Go source MUST be removed. The design MUST first confirm, per key, that no non-Go consumer (script, job, external tool) references it. Existing topics already created in live environments are left in place, unused.
- **FR-6.3** A short `migration.md` in this task folder MUST record the before/after topic set so a live-environment diff is reviewable.

## 5. API Surface

No REST/JSON:API surface changes. The contracts introduced are build-time and file-level:

| Artifact | Direction | Shape |
|---|---|---|
| `libs/atlas-kafka/gen/topics.yaml` | generated | `topics: [{token, packages[], cleanup}]`, sorted by token |
| `libs/atlas-kafka/gen/policies.yaml` | hand-authored | `compact: [token, …]` with rationale comments |
| `deploy/k8s/base/env-configmap.yaml` (topic block) | generated | `TOKEN: "TOKEN"` lines between markers |
| `deploy/k8s/base/kafka-topics-configmap.yaml` | generated | ConfigMap `atlas-kafka-topics`, one key carrying the token+policy list |
| `libs/atlas-kafka/gen` CLI | — | `go run .` (write), `go run . -check` (exit 1 on drift) |

Go signature changes (breaking within the monorepo, mechanical):

- `topic.EnvProvider(l) func(Token) Provider` — parameter retyped; unset variable now yields an error rather than the token.
- `consumer` config constructors and the producer/message-buffer token parameters retyped from `string` to `topic.Token`.

## 6. Data Model

No database entities, no migrations, no `tenant_id` scoping — topics are an infrastructure-level concern shared across tenants. The "data model" here is the manifest schema in §5. Every token entry is environment-independent by construction; per-environment naming remains the overlays' job.

## 7. Service Impact

| Area | Change |
|---|---|
| `libs/atlas-kafka/topic` | New `Token` type; `EnvProvider` retyped and fallback removed |
| `libs/atlas-kafka/consumer`, `/producer`, message buffer | Token parameters retyped |
| `libs/atlas-kafka/gen` (new module) | Static-analysis generator + `-check` |
| All 14+ services under `services/` | Mechanical: ~163 const declarations retyped; ~30 discarded `EnvProvider` errors handled; per-service `NewConfig` wrappers retyped |
| `services/atlas-kafka-precreate` | `discover.FromEnviron` + prefix consts deleted; manifest reader added; policy sourced from manifest |
| `services/atlas-cashshop`, `atlas-skills`, `atlas-channel` | Pick up the two newly managed `STATUS_*` topics (name change, FR-6.1) |
| `deploy/k8s/base/env-configmap.yaml` | Topic block becomes generated |
| `deploy/k8s/base/kafka-topics-configmap.yaml` | New, generated |
| `deploy/k8s/base/atlas-kafka-precreate.yaml` | Volume + volumeMount for the manifest ConfigMap. **The `envFrom`-only constraint in its comment block must be respected** — the PR-validation JSON-6902 patch does `op: add` on `/spec/template/spec/containers/0/env`, so this Job must not grow an `env:` key; the mount-path override (FR-4.2) must therefore come from `atlas-env` or default |
| `deploy/k8s/overlays/{main,pr,pr-sparse}` | Topic lists regenerated/verified; `gen-topic-config.sh` selector widened |
| `tools/verify.sh` | New drift step |

## 8. Non-Functional Requirements

- **Performance (build):** the `-check` pass must not meaningfully lengthen the common `verify.sh` run — see FR-5.2. Target: under 30s warm, and no worse than the existing `go vet` type-check pass it can potentially share work with.
- **Performance (runtime):** precreate's wave-0 duration must not regress from task-260's steady-state (~seconds). Reading a mounted file is strictly cheaper than the environ scrape it replaces.
- **Correctness over tolerance:** every new failure mode in this task fails *early* (build time, or wave 0 before Deployments start) rather than degrading into a runtime stampede. That is the point of the task.
- **Observability:** precreate must log the manifest path, the token count, and the compact-topic count at startup; a resolution failure must name the offending token.
- **Multi-tenancy:** unaffected — topics are per-environment, not per-tenant.
- **Testing:** the generator gets table-driven unit tests over synthetic packages (declared token, inline literal, conflicting policy, stale policy entry, test-file exclusion); precreate's manifest reader gets table tests for malformed/missing/empty cases; the existing `discover` tests for groups and seedability must keep passing untouched.
- **No placeholders:** the retyping sweep touches every service. It is mechanical and must be complete — a partially retyped tree with `string` call sites still compiling defeats FR-1.4.

## 9. Open Questions

1. **Sparse/ephemeral overlays and FR-4.4's fatal rule.** `overlays/pr-sparse` deliberately shares the baseline's topics and may not define every key. Design must render all three overlays and confirm that "every manifest token has an env value" actually holds before making it fatal; if it does not, the fatal rule needs a documented, narrow exemption rather than a blanket warn.
2. **Where cleanup policy is declared.** This PRD puts it in `policies.yaml` (§4.2, FR-2.7) so the generator stays a pure scan. The alternative — a second defined type, `topic.CompactedToken`, so policy travels with the declaration — is more type-safe but risks two services declaring the same token with different types (FR-2.6 would catch it, loudly). Design decides.
3. **Generator speed vs. `verify.sh` trigger breadth.** A full `go/packages` load over every module on every Go change may be too slow. Options: share the load with the existing `go vet` pass, cache by file hash, or use a cheap grep pre-check that only escalates to the full load when a `topic.Token` declaration line changed.
4. **`EnvProvider` fallback removal blast radius.** ~30 call sites discard the error today. Design must decide, per site, between fatal-at-boot and propagate — and confirm no service currently *relies* on the fallback beyond the two known `STATUS_*` tokens.
5. **Consumer groups.** `libs/atlas-kafka/consumergroup/resolver.go` has the same "declared nowhere, resolved from env at runtime" shape for group names, including `fmt.Sprintf` templating. Explicitly out of scope here; this task should note whether the manifest format leaves room for a `groups:` section so the follow-up is additive.
6. **Cashshop status-topic name change.** FR-6.1 abandons whatever sits on the unsuffixed `STATUS_TOPIC_CASH_ITEM` today. Confirm the producer/consumer offset semantics make this safe.

## 10. Acceptance Criteria

- [ ] `libs/atlas-kafka/topic` defines `Token`; `EnvProvider` takes it and returns an error (never the token) when the variable is unset.
- [ ] Every non-test topic constant in `services/` and `libs/` is declared as `topic.Token`; no call site passes a bare string literal.
- [ ] Every previously discarded `EnvProvider` error is handled; no consumer or producer starts against an unresolved topic.
- [ ] `cd libs/atlas-kafka/gen && go run .` regenerates `topics.yaml`, the `env-configmap.yaml` topic block, and `kafka-topics-configmap.yaml`; re-running produces no diff.
- [ ] `go run . -check` exits 0 on a clean tree, and exits 1 with a named diff when: a token is added in code but not regenerated; a `policies.yaml` entry names an unknown token; a call site passes an undeclared literal.
- [ ] `topics.yaml` contains `STATUS_TOPIC_CASH_ITEM` and `STATUS_EVENT_TOPIC_SKILL_MACRO`, and neither resolves to an unsuffixed name in any rendered overlay.
- [ ] The 17 orphan keys are gone from `env-configmap.yaml` and from all three overlay kustomizations.
- [ ] `kustomize build` succeeds for `overlays/main`, `overlays/pr`, and `overlays/pr-sparse`; the rendered topic set matches the manifest in each.
- [ ] `discover.FromEnviron`, `commandPrefix`, and `eventPrefix` no longer exist in `services/atlas-kafka-precreate`.
- [ ] Precreate reads the mounted manifest, resolves each token through the environment, fails wave 0 by name on an unresolvable token, and applies `cleanup.policy=compact` to exactly the manifest-declared compact set.
- [ ] The `atlas-kafka-precreate` Job mounts the ConfigMap and still carries **no** `env:` key on its container.
- [ ] `tools/verify.sh` gains the drift step, and the flagless run exits 0.
- [ ] `migration.md` records the before/after topic set.
