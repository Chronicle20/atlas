# Code-Derived Kafka Topic Manifest — Design

Task: task-276-kafka-topic-manifest
PRD: [prd.md](prd.md) (approved, v1)
Status: Draft
Created: 2026-08-28

---

## 1. What the investigation changed about the PRD's picture

The PRD's diagnosis is correct and its two live defects are confirmed exactly. Its
*sizing* is not: three numbers are an order of magnitude off, and two deployment
surfaces are missing from §7. Everything below is measured on this branch.

| PRD claim | Measured | Where |
|---|---|---|
| "~163 declarations" to retype | **517** non-test declaration lines, **159** distinct tokens | `grep -rn --include='*.go' -E '[A-Za-z_]+\s*=\s*"[A-Z0-9_]*TOPIC[A-Z0-9_]*"' services/ libs/ \| grep -v _test.go` |
| "~30 discarded `EnvProvider` errors" | **336** call sites, **325** discard the error | `grep -rn 'topic.EnvProvider' services/ libs/` |
| 17 orphan ConfigMap keys | **17**, confirmed exactly | `comm -13` of code tokens vs `env-configmap.yaml` keys |
| 2 unmanaged `STATUS_*` tokens | **2**, confirmed exactly | same `comm`, other direction |
| §7 service-impact table | **misses `libs/atlas-outbox` and `deploy/compose/.env.example`** | below |

Four findings drive the architecture:

**F1 — a defined type does not close the hole.** Go assigns an *untyped* string
constant to any string-kind named type. After `type Token string`,
`buf.Put("EVENT_TOPIC_MARRIAGE_STATUS", p)` still compiles unchanged. FR-1.4 is
therefore **not** enforceable by the type system, and the generator (which finds
tokens *by type*) cannot see such a literal at all. There is exactly one such site
today — `services/atlas-marriages/atlas.com/marriages/kafka/consumer/character/consumer.go:78`
— and four more that evade the mechanism by a different route:
`libs/atlas-service/envregistry.go:52,59` and `libs/atlas-service/projection.go:67,68`
call `os.Getenv("EVENT_TOPIC_CONFIGURATION_*")` directly, bypassing
`topic.EnvProvider` entirely. Closing FR-1.4 requires a **go/analysis analyzer**,
not a type.

**F2 — the analyzer is also the cheap gate, which resolves Open Question 3.**
`tools/go-analyzer-guards.sh` already builds one `unitchecker` binary carrying five
analyzers and type-checks `services/` and `libs/` **once** per verify run; its header
records the measurement that motivated the consolidation (46.4s / 47.5s per *standalone*
driver pass, of which the analyzer build was ~10s). A sixth analyzer joining that sweep
costs the analyzer's own walk over already-built syntax trees — effectively free. A
standalone `go/packages` `NeedTypes` load in `libs/atlas-kafka/gen` would re-pay that
whole ~46s on the common Go-change path and blow NFR §8's 30s budget. So the design
**splits the two jobs**: the analyzer is the gate on `.go` changes; the generator is the
writer, gated on a narrow non-Go trigger. See §4.

**F3 — the token flows further than §7 says.** The producer path is
`Buffer.Put(token) → Buffer.GetAll() map[token][]kafka.Message → producer.Provider(token)
→ Manager.Writer(l, token) → topic.EnvProvider`. There are **41** per-service `Buffer`
types with an identical `Put(t string, …)` and **827** `Put` call sites. The transactional
half of that path lands in `libs/atlas-outbox`: `EnqueueBuffer(…, contents map[string][]kafka.Message)`
(`bridge.go:21`) keys by token and resolves at `bridge.go:37`. `libs/atlas-outbox` must be
retyped and is absent from PRD §7. (It already propagates the `EnvProvider` error
correctly — one of the 11 sites that does.)

**F4 — there are four deployment surfaces, not two.**
`deploy/compose/.env.example` is a live developer surface with the same identity mapping
(`COMMAND_TOPIC_ACCOUNT=COMMAND_TOPIC_ACCOUNT`) and it is badly stale: **89** tokens
against 159 in code — it carries 15 of the 17 orphans and is missing ~72 real tokens.
Leaving it hand-maintained keeps a live instance of the exact defect class this task
exists to remove, on a surface no gate watches. It is generated here (§8), and this is
the one deliberate scope addition beyond the PRD.

---

## 2. Architecture

Five components, three of which are new.

```
                 libs/atlas-kafka/topic
                 └── type Token string                     (D1)
                            │
        declarations        │        call sites
   517 × `X topic.Token = "…"`      Put/Provider/Writer/EnqueueBuffer
            │                               │
            │                               │
   ┌────────▼────────┐            ┌─────────▼──────────┐
   │ libs/atlas-kafka│            │ tools/topicguard   │  (D2b) NEW
   │ /gen  (WRITER)  │            │ joins atlasguards  │
   │ go/packages     │            │ shared vet sweep   │
   │ NeedTypes load  │            │  • no bare literal │
   │      (D2a) NEW  │            │  • no raw os.Getenv│
   └────────┬────────┘            │  • token ∈ manifest│
            │                     └─────────┬──────────┘
            │ writes                        │ reads
            ▼                               │
   libs/atlas-kafka/gen/topics.yaml ◄───────┘
   libs/atlas-kafka/gen/policies.yaml  (hand-authored)
            │
            │ renders
            ├─► deploy/k8s/base/env-configmap.yaml         (marked block)
            ├─► deploy/k8s/base/kafka-topics-configmap.yaml (whole file)
            ├─► deploy/k8s/overlays/{main,pr,pr-sparse}/kustomization.yaml (marked blocks)
            └─► deploy/compose/.env.example                 (marked block)
                            │
                            ▼ mounted
                 services/atlas-kafka-precreate
                 internal/manifest (NEW) → discover.Topics  (D8)
```

Component ownership:

| Component | New? | Owns |
|---|---|---|
| `libs/atlas-kafka/topic.Token` | new type | the declaration shape |
| `tools/topicguard` | **new module** | FR-1.4 + manifest-membership gate, on every `.go` change |
| `libs/atlas-kafka/gen` | **new module** | the scan, `topics.yaml`, and all four rendered artifacts |
| `libs/atlas-kafka/gen/policies.yaml` | new file | cleanup policy, hand-authored |
| `services/atlas-kafka-precreate/internal/manifest` | **new package** | mounted-manifest load + env resolution |

---

## 3. D1 — the token type

```go
// libs/atlas-kafka/topic/token.go
package topic

// Token is the NAME OF THE ENVIRONMENT VARIABLE that carries a topic's
// per-environment name — never the topic name itself. The distinction is
// load-bearing: overlays suffix the name, the manifest carries only the token.
type Token string
```

Declaration form, applied to all 517 sites:

```go
const (
	EnvCommandTopic     topic.Token = "COMMAND_TOPIC_SKILL_MACRO"
	EnvStatusEventTopic topic.Token = "STATUS_EVENT_TOPIC_SKILL_MACRO"
)
```

Signatures retyped `string → topic.Token` (the token-carrying ones only):

| Symbol | File |
|---|---|
| `topic.EnvProvider(l) func(Token) Provider` | `libs/atlas-kafka/topic/provider.go:13` |
| `producer.Provider func(Token) MessageProducer` | `libs/atlas-kafka/producer/provider.go:9` |
| `(*producer.Manager).Writer(l, Token)` | `libs/atlas-kafka/producer/manager.go:67` |
| `producer.ManagerWriterProvider(l)(Token)` | `libs/atlas-kafka/producer/manager.go` |
| `outbox.EnqueueBuffer(…, map[Token][]kafka.Message)` | `libs/atlas-outbox/bridge.go:21` |
| 41 × `(*Buffer).Put(Token, …)` / `GetAll() map[Token][]kafka.Message` / `Emit` | `services/*/kafka/message/message.go` |
| 61 × per-service `NewConfig(l)(name)(Token)(groupId)` | `services/*/kafka/consumer/consumer.go` |

**Explicitly NOT retyped:** `consumer.NewConfig(brokers, name, topic string, groupId)`
(`libs/atlas-kafka/consumer/config.go:28`), `Config.topic`, `producer.WriterFactory`,
`Manager.writers map[string]Writer`, `outbox.Message.Topic`, and every `rf(t, handler)`
registration function. **These take the RESOLVED topic name, not a token.** PRD FR-1.3
lists "`libs/atlas-kafka/consumer`'s config constructors" among the retype targets; that
is a misreading of the seam — the library constructor never sees a token, only the
per-service wrapper does. Retyping it would make `Token` mean two things and destroy the
analyzer's ability to tell them apart. This is a deliberate, narrow deviation from FR-1.3.

---

## 4. D2 — split the scan from the gate (resolves Open Question 3)

### D2a — `libs/atlas-kafka/gen`, the writer

Its own module (not in `go.work` — same posture as `tools/atlasguards`, built with
`GOWORK=off`), depending on `golang.org/x/tools v0.49.0`, the version already pinned by
`tools/scopeguard` and `tools/atlasguards`.

```
cd libs/atlas-kafka/gen
go run .           # write topics.yaml + all four rendered artifacts
go run . -check    # exit 1 with a unified diff; writes nothing
```

Loads every `use` directive in `go.work` with
`packages.NeedTypes|NeedTypesInfo|NeedSyntax|NeedName|NeedFiles`, collects every constant
whose type is `github.com/Chronicle20/atlas/libs/atlas-kafka/topic.Token`, and folds its
value. `packages.Load` errors, or any package with `len(pkg.Errors) > 0`, is a hard
failure naming the package (FR-2.3) — a partial load is the silent gap this task removes.
`_test.go` files are excluded by loading without `Tests: true`, which satisfies FR-1.7
structurally rather than by denylist (76 test-file token lines drop out).

### D2b — `tools/topicguard`, the gate

A go/analysis analyzer registered as the **sixth** analyzer in `tools/atlasguards`,
appended to `GUARD_SRCS` in `tools/go-analyzer-guards.sh:50-57` so an edit to it rebuilds
the shared binaries. Runs in **both** the services and libs passes (the `libs/atlas-service`
`os.Getenv` violations are in `libs/`). Three diagnostics:

1. **`bare-token-literal`** — a string literal or untyped string constant, whose value
   matches `^[A-Z0-9_]*TOPIC[A-Z0-9_]*$`, converted to `topic.Token` at a call site
   rather than referenced through a `topic.Token`-typed constant. Closes F1/FR-1.4.
   Current violation: `services/atlas-marriages/.../consumer/character/consumer.go:78`.
2. **`raw-env-topic-read`** — `os.Getenv` / `os.LookupEnv` with a literal argument matching
   the same pattern, outside `libs/atlas-kafka/topic`. Current violations:
   `libs/atlas-service/envregistry.go:52,59`, `libs/atlas-service/projection.go:67,68`.
3. **`token-not-in-manifest`** — a `topic.Token` constant whose value is absent from the
   checked-in `topics.yaml`. The analyzer reads `topics.yaml` once via an
   `-topicguard.manifest` flag defaulting to a repo-relative path.

Diagnostic 3 is what makes the *common path* cheap: adding a token in Go without
regenerating fails the already-running vet sweep by name, at ~zero marginal cost. The
expensive `-check` never has to run on a Go change.

### D2c — `tools/verify.sh` wiring (FR-5.1, FR-5.2, FR-5.3)

The analyzer needs no new step — it rides the existing
`if [ "$ALL" -eq 1 ] || touched '\.go$'` block at `tools/verify.sh:374`.

One new step, on a **narrow** trigger:

```sh
if [ "$ALL" -eq 1 ] || touched '^(libs/atlas-kafka/gen/|deploy/k8s/base/env-configmap\.yaml|deploy/k8s/base/kafka-topics-configmap\.yaml|deploy/k8s/overlays/[a-z-]+/kustomization\.yaml|deploy/compose/\.env\.example)'; then
    step "topic manifest drift" ./tools/gen-topics.sh --check
else
    skip "topic manifest drift (no manifest or deploy surface changed)"
fi
```

`tools/gen-topics.sh` is a thin `cd libs/atlas-kafka/gen && GOWORK=off go run . "$@"` wrapper,
matching the `gen-lb-ports.sh --check` / `gen-routes.sh --check` step shape at
`tools/verify.sh:566-569`. No Docker, no broker (FR-5.3).

**The `.go` trigger FR-5.2 asks for is deliberately not used**, because diagnostic 3 covers
the same failure at a fraction of the cost. FR-5.2's own escape clause ("gate the expensive
load behind a cheap pre-check") is satisfied — the analyzer *is* the pre-check, and it is
strictly stronger than a grep because it is type-aware.

---

## 5. D3 — manifest schema

`libs/atlas-kafka/gen/topics.yaml` (generated, sorted by token — FR-2.4/2.5):

```yaml
# GENERATED by libs/atlas-kafka/gen — do not edit.
# Source of truth: topic.Token constants in services/ and libs/.
# Regenerate: tools/gen-topics.sh    Drift check: tools/gen-topics.sh --check
topics:
  - token: COMMAND_TOPIC_CHARACTER
    cleanup: delete
    packages:
      - atlas.com/buffs/kafka/message/character
      - atlas.com/maps/tasks
      # …every declaring package, sorted
  - token: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS
    cleanup: compact
    packages:
      - atlas.com/configurations/tenants
      - atlas.com/configurations/servicesuniq
```

Two forward-compatibility decisions:

- The root is a **mapping**, not a sequence, so Open Question 5's follow-up can add a
  sibling `groups:` key purely additively. The `libs/atlas-kafka/consumergroup` work stays
  out of scope; the format leaves room for it.
- `cleanup` is an explicit enum (`delete` | `compact`) written on **every** entry rather
  than omitted-means-default, so a policy change is a one-line diff rather than an
  added/removed line.

`kafka-topics-configmap.yaml` carries the same document verbatim under one key, so
precreate and the generator parse one schema with one struct:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: atlas-kafka-topics
data:
  topics.yaml: |
    topics:
      - token: COMMAND_TOPIC_CHARACTER
        cleanup: delete
      # packages[] omitted — ownership metadata is a review aid, not runtime input
```

`packages[]` is stripped from the mounted copy: it is ~159 × N lines of provenance that
precreate never reads, and a ConfigMap has a 1 MiB ceiling.

---

## 6. D4 — cleanup policy lives in `policies.yaml` (resolves Open Question 2)

**Decision: `policies.yaml`, as PRD FR-2.7 proposes. Reject `topic.CompactedToken`.**

The measured reason: the three compact tokens are declared at **3, 3, and 2** sites
respectively (`EVENT_TOPIC_CONFIGURATION_{TENANT,SERVICE}_STATUS` at 3 each,
`…_ENVIRONMENT_STATUS` at 2). A second type would have to be applied consistently at
every one of those sites forever, across service boundaries, and FR-2.6's conflict check
would fire on any single miss. More fundamentally: **cleanup policy is a property of the
Kafka topic, not of a Go declaration.** Two services declaring the same token are naming
one topic; letting each declare a policy for it creates a conflict that has no correct
resolution. A single file keyed by token has exactly one place to state the answer.

```yaml
# libs/atlas-kafka/gen/policies.yaml — HAND-AUTHORED. Not generated.
#
# Tokens whose topic must carry cleanup.policy=compact.
#
# Their consumers replay from first-offset at every boot to rebuild
# tenant/service config state and the outbox never re-emits a (topic, key) it
# already delivered, so under the default DELETE cleanup retention empties the
# topic ~7 days after the last config change and every later projection boot has
# nothing to replay. Events are keyed, so compaction retains the latest snapshot
# per key forever.
#   (carried verbatim from services/atlas-kafka-precreate/internal/discover/discover.go:20-27)
compact:
  - EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS
  - EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS
  - EVENT_TOPIC_CONFIGURATION_TENANT_STATUS
```

A `compact:` entry naming a token the scan did not find is a hard failure (FR-2.7) — a
stale policy is drift. FR-2.6's "same token, conflicting policy" case becomes structurally
impossible under this decision, so it is satisfied vacuously and the generator carries no
code for it.

---

## 7. D5 — generating `env-configmap.yaml`

`deploy/k8s/base/env-configmap.yaml` today is **globally alphabetical**, which puts the
topics in **two** non-contiguous regions (lines 21-102 `COMMAND_TOPIC_*`, 105-196
`EVENT_TOPIC_*`) with `DB_HOST`/`DB_PORT` wedged between them at 103-104 and
`LEACH_INTERVAL`…`USE_ENFORCE_MOB_LEVEL_RANGE` after at 197-203. Adding the two `STATUS_*`
tokens would open a **third** region between `REST_PORT` and `TRACE_ENDPOINT`.

**Decision: one generated block, moved to the tail of `data:`.** Hand-written keys
(`ATLAS_ENVIRONMENT` and its 12-line rationale comment, `BASE_SERVICE_URL`,
`BOOTSTRAP_SERVERS`, `DB_*`, `LEACH_INTERVAL`, `LEVEL_INTERVAL`, `REDIS_URL`, `REST_PORT`,
`TRACE_*`, `USE_ENFORCE_MOB_LEVEL_RANGE`) move above it and are preserved byte-for-byte
(FR-3.2); the topic block itself stays internally sorted.

```yaml
  USE_ENFORCE_MOB_LEVEL_RANGE: "true"
  # BEGIN GENERATED TOPICS — libs/atlas-kafka/gen. Do not edit; run tools/gen-topics.sh.
  COMMAND_TOPIC_ACCOUNT: "COMMAND_TOPIC_ACCOUNT"
  …
  STATUS_TOPIC_CASH_ITEM: "STATUS_TOPIC_CASH_ITEM"
  # END GENERATED TOPICS
```

Rejected: three marker pairs preserving global alphabetical order. It encodes today's
prefix set into the file's structure — the very coupling this task removes — and a token
with a new prefix would need a fourth region.

The rewrite is **line-oriented over marker delimiters**, not YAML round-tripping. `yq`
and `gopkg.in/yaml.v3` both drop or relocate comments on re-emit, and this file's value is
substantially in its comments. The generator splices between markers and leaves every
other byte untouched; `-check` compares the spliced result to the file on disk.

---

## 8. D6 — `deploy/compose/.env.example` (scope addition)

Generated with the same marker splice, identity mapping, `KEY=KEY` shell form. This is
**beyond PRD §7**, justified by F4: it is a live surface carrying 15 orphans and missing
~72 tokens, and every argument in the PRD's §1 applies to it verbatim. It costs one more
renderer and one more `--check` comparison over an artifact the generator already holds in
memory. Flagging it explicitly so the scope decision is the user's: if it should be
dropped, remove the renderer and the `.env.example` path from the verify trigger — nothing
else changes.

---

## 9. D7 — overlay generation (FR-3.4, FR-3.5)

All three overlays already carry all 174 topic literals as a **contiguous trailing block**
inside their `configMapGenerator.literals` (`main` at kustomization.yaml:60-233, ending
immediately before `images:`; `pr` from :180; `pr-sparse` from :343). Same marker-splice
treatment, with the per-overlay suffix baked into the rendered line:

| Overlay | Suffix | Line form |
|---|---|---|
| `main` | `-main` | `      - COMMAND_TOPIC_X=COMMAND_TOPIC_X-main` |
| `pr` | `-PLACEHOLDER_ATLAS_ENV` | `      - COMMAND_TOPIC_X=COMMAND_TOPIC_X-PLACEHOLDER_ATLAS_ENV` |
| `pr-sparse` | `-PLACEHOLDER_BASELINE_ENVIRONMENT` | `      - COMMAND_TOPIC_X=COMMAND_TOPIC_X-PLACEHOLDER_BASELINE_ENVIRONMENT` |

The suffix table lives in the generator, which **subsumes**
`deploy/k8s/overlays/pr/scripts/gen-topic-config.sh` — the manifest replaces
`env-configmap.yaml` as its input and the `test("^(COMMAND|EVENT)_TOPIC_")` selector
(the mechanism that silently drops the two `STATUS_*` tokens) disappears rather than being
widened. FR-3.4 permits exactly this ("or replaced by the manifest"). The script is deleted
and its rationale comment — particularly the 2026-08-20 `atlas-login` crash-loop note
explaining why `pr-sparse` must use the baseline's suffix and not "no suffix" — is carried
verbatim into the generator's suffix table.

Two existing guards are checked and unaffected: `tools/pr-sparse-mirror-guard.sh`'s
`MIRRORS` array does not list `kustomization.yaml`, and `tools/overlay-env-guard.sh`
asserts on `ATLAS_ENVIRONMENT` and ingress wiring, not topics.

---

## 10. D8 — precreate consumption (FR-4.1 … FR-4.7)

The seam is clean: `main.go:56` is a single call,
`t := discover.FromEnviron(os.Environ())`, producing `discover.Topics{Plain, Compact}`,
which `topics.Ensure` consumes. Everything downstream — the batched `CreateTopics`,
`errors.Is` on `TopicAlreadyExists`, `Settle`, `EndOffsets`, `groups.Seed`, `Verify`, the
exit-code contract — is untouched (FR-4.6).

New package `internal/manifest`, two pure functions plus one I/O shim, table-testable
without a broker in the same style as `discover`:

```go
func Parse(data []byte) (Manifest, error)                              // malformed / empty / no topics: → error
func Resolve(m Manifest, look func(string) (string, bool)) (discover.Topics, error)
func Load(path string) (Manifest, error)                               // the only I/O
```

`Resolve` walks the manifest in sorted order and returns the **first** unresolvable token
by name (FR-4.4), classifying the rest into `Plain`/`Compact` from the manifest's `cleanup`
field (FR-4.5). It preserves `FromEnviron`'s compaction-wins de-duplication: two tokens
resolving to the same name, one compact, yield one compact topic.

`discover.FromEnviron`, `commandPrefix`, `eventPrefix`, and `compactVars` are **deleted**
(FR-4.1). `discover.Groups` and `discover.StateIsSeedable` and their tests stay untouched
(FR-4.7). `discover.Topics` itself stays where it is — it is the shared vocabulary
`topics.Ensure` already speaks.

`main.go`:

```go
path := os.Getenv("KAFKA_TOPIC_MANIFEST_PATH")
if path == "" {
    path = "/etc/atlas/topics/topics.yaml"
}
m, err := manifest.Load(path)          // fatal, names the path (FR-4.2)
if err != nil { return fmt.Errorf("loading topic manifest from %s: %w", path, err) }
t, err := manifest.Resolve(m, os.LookupEnv)
if err != nil { return err }           // fatal, names the token (FR-4.4)
logrus.WithFields(logrus.Fields{
    "phase": "discover", "manifest": path,
    "tokens": len(m.Topics), "compact": len(t.Compact),
}).Info("topic manifest loaded")       // NFR observability
```

**The `envFrom`-only constraint is respected.** `KAFKA_TOPIC_MANIFEST_PATH` is read via
`os.Getenv` with a hardcoded default and is **not** added to the Job's container `env:`.
`deploy/k8s/base/atlas-kafka-precreate.yaml`'s comment at the container spec is explicit
that `pr-validation.yml`'s JSON-6902 `op: add` on
`/spec/template/spec/containers/0/env` *replaces* an existing `env:` key, so growing one
would make `KAFKA_CONSUMER_GROUP` its only entry. The Job gains only `volumes:` and
`volumeMounts:` — neither is touched by that patch:

```yaml
          volumeMounts:
            - name: topic-manifest
              mountPath: /etc/atlas/topics
              readOnly: true
      volumes:
        - name: topic-manifest
          configMap:
            name: atlas-kafka-topics
```

The Job's header comment ("picks out every `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` value")
is rewritten to describe the manifest.

**Ordering hazard.** The ConfigMap is a wave-0 resource consumed by a wave-0 Job. ArgoCD
applies same-wave resources before health-checking any of them, and a `configMap` volume
is resolved at pod start, so a Job pod that starts before the ConfigMap exists fails
`ContainerCreating` and is retried under `backoffLimit: 3` within
`activeDeadlineSeconds: 300`. Plan should give `kafka-topics-configmap.yaml` an explicit
`argocd.argoproj.io/sync-wave: "-1"` to remove the race rather than rely on the retry.

---

## 11. D9 — the `EnvProvider` error sweep (resolves Open Question 4)

336 call sites, 325 discarding. The count is tractable because it is **two shapes**, not
325 individual decisions.

**Shape A — 61 identical per-service `NewConfig` wrappers** (`services/*/kafka/consumer/consumer.go`),
byte-identical across services:

```go
func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return func(name string) func(token string) func(groupId string) consumer.Config {
		return func(token string) func(groupId string) consumer.Config {
			t, _ := topic.EnvProvider(l)(token)()
```

These sit at the bottom of a **non-error-returning** signature; `InitConsumers` chains
`rf(NewConfig(l)(name)(token)(groupId), …)` with no error path, so propagation would
cascade through every consumer registration in every service. **Decision: fatal-at-boot.**
`l.WithError(err).Fatalf("unresolvable topic token [%s]", token)`. This is correct on the
merits: a consumer registered against an unresolved topic is precisely the wedge NFR §8
("fails early rather than degrading into a runtime stampede") targets, and it fires during
`main`'s wiring, before any goroutine starts.

**Shape B — ~264 `InitHandlers` sites**, all inside a function that **already returns
`error`**:

```go
var t string
t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
if _, err := rf(t, …); err != nil { return err }
```

**Decision: propagate.** Mechanically:

```go
t, err := topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
if err != nil { return err }
```

Both shapes are **codemod territory, not agent territory** (`docs/codemod-vs-agents.md`):
one AST-rewrite pass over `topic.EnvProvider` call expressions, dispatching on whether the
enclosing function's result list ends in `error`. Shape B's multi-handler variant (`var t
string` then several `t, _ =` in one body) needs the declaration hoisted to `var t string;
var err error` — a rewrite rule, not 264 judgements. The plan should run the codemod, then
`gofmt`, then a full `go build ./...` per module, and land the residue (any site whose
enclosing function returns neither `error` nor is a `NewConfig` wrapper) as an explicit
enumerated list rather than a silent fallthrough.

**No service relies on the fallback beyond the two known tokens.** The `comm` in §1 shows
exactly two code tokens absent from `env-configmap.yaml`; every other token has a value in
all three overlays (all three carry all 174 keys), so removing the fallback changes
observable behaviour for those two tokens only.

---

## 12. D10 — migration (resolves Open Questions 1 and 6)

**OQ1 — `pr-sparse` and the fatal rule: RESOLVED, the fatal rule holds.**
All three overlays carry the **complete** 174-key topic list inline (`main` 174, `pr` 174,
`pr-sparse` 174 — verified by `grep -c 'TOPIC' deploy/k8s/overlays/*/kustomization.yaml`).
`pr-sparse` shares the *baseline's topic names* via the `-PLACEHOLDER_BASELINE_ENVIRONMENT`
suffix; it does not share by *omitting keys*. Since §9 regenerates all three from the same
manifest, "every manifest token has an env value" is true by construction in every overlay.
FR-4.4 needs no exemption. The plan must still `kustomize build` all three and assert the
rendered key set equals the manifest.

**OQ6 — cashshop name change: RESOLVED, trivially safe. `STATUS_TOPIC_CASH_ITEM` has no
consumer.** A repo-wide search finds one declaration
(`services/atlas-cashshop/atlas.com/cashshop/kafka/message/item/kafka.go:5`), four producer
sites (`cashshop/inventory/asset/processor.go:123,180,232,318`), and one test comment. **No
`InitConsumers`, no `InitHandlers`, no `rf(` registration anywhere in the repo reads it.**
Abandoning whatever sits on the unsuffixed topic abandons nothing anyone reads.

`STATUS_EVENT_TOPIC_SKILL_MACRO` **is** consumed, by
`services/atlas-channel/.../kafka/consumer/macro/consumer.go:29` — and that registration
carries `consumer.SetStartOffset(kafka.LastOffset)`, so the consumer never replays history.
Repointing it at a fresh suffixed topic loses nothing it would have read. FR-6.1's
"acceptable for a status-event topic read at `LastOffset`" is confirmed against the code,
not assumed.

**One residual, benign:** `libs/atlas-outbox` persists the **resolved** topic name into
`outbox_entries.topic` (`entity.go:11`, written at `bridge.go:42`). Unsent rows enqueued
before the change still name the old topic. `TopicWriterPool` sets
`AllowAutoTopicCreation: true` (`publisher.go:86`), so they drain to the old topic rather
than erroring. Neither of the two renamed tokens is produced through the outbox bridge
today (cashshop's four sites use `mb.Put` under the direct `message.Emit` path), so the
window is empty in practice. Recorded, not mitigated.

**The 17 orphans: safe to remove.** Every one was checked for non-Go references. All 17
appear in `deploy/compose/.env.example` (regenerated here, §8); 4 additionally appear only
in historical `docs/tasks/**` prose. `EVENT_TOPIC_SKILL_MACRO` looks like a Go hit but is
a substring match on `STATUS_EVENT_TOPIC_SKILL_MACRO` — it is a genuine orphan. **No
script, Job, or external tool references any of them.** Live-cluster topics already created
under these names are left in place, unused, per FR-6.2.

`migration.md` (FR-6.3) records: 174 keys before → 159 after (−17 orphans, +2 `STATUS_*`),
the two topics whose *rendered name* changes per environment, and the empty in-flight
window above.

---

## 13. Rejected alternatives

| Alternative | Why rejected |
|---|---|
| Generator alone as the drift gate, triggered on `\.go$` (PRD FR-5.2's literal reading) | A standalone `go/packages NeedTypes` load over ~90 modules re-pays the ~46s type-check that `go-analyzer-guards.sh` already performs once per run (its own header documents the measurement). Blows NFR §8's 30s budget on the most common trigger there is. |
| Type alone, no analyzer | F1: Go converts untyped string constants to `Token` implicitly. `Put("EVENT_TOPIC_X")` still compiles, and the generator — which finds tokens by type — cannot see it. FR-1.4 would be unenforced. |
| `topic.CompactedToken` second type (PRD OQ2's alternative) | Policy is a property of the topic, not the declaration. 2-3 declaring sites per compact token across service boundaries, each of which must agree forever; disagreement has no correct resolution. |
| Runtime topic registry | Already rejected in PRD §2 — separate modules under `go.work` still need a build-time cross-module scan to collect. |
| `yq`/`yaml.v3` round-trip of `env-configmap.yaml` | Both drop or relocate comments; this file's `ATLAS_ENVIRONMENT` rationale block is 12 lines of load-bearing comment. Marker splice preserves every non-generated byte. |
| Three marker pairs preserving global alphabetical order | Encodes the current prefix set into the file's structure — the exact coupling this task removes. A new prefix would need a fourth region. |
| Widening `gen-topic-config.sh`'s regex (PRD FR-3.4's first option) | Keeps a prefix regex as load-bearing infrastructure. Replacing the script with the manifest, which FR-3.4 also permits, removes the class. |
| Leaving `deploy/compose/.env.example` hand-maintained | A live instance of this task's defect class (15 orphans, ~72 missing) on a surface no gate watches. |
| Sequence-rooted `topics.yaml` | Blocks Open Question 5's additive `groups:` follow-up. |

---

## 14. Risks

| Risk | Mitigation |
|---|---|
| **517-site retype is a partial sweep** — NFR §8's "no placeholders". Untyped-constant conversion means a half-retyped tree still compiles. | The `token-not-in-manifest` diagnostic fires on *declared* tokens; `bare-token-literal` catches the *undeclared* ones. Both run over `services/` and `libs/` in the shared sweep. A missed declaration surfaces as a manifest gap, not a green build. Plan must run flagless `tools/verify.sh` to exit 0. |
| **Codemod regression across 336 error sites** | Per-module `go build ./...` + `go test ./...` after each codemod pass; residue enumerated explicitly, never silently skipped. |
| **`gofmt` churn hiding a semantic change in review** | Land the mechanical retype and the error-handling codemod as **separate commits**, each `gofmt`-clean, so the diff is reviewable. |
| Generator diverges from analyzer on what counts as a token | Both link the same `topicguard` matcher package; the analyzer is the only implementation of "is this a token literal". |
| ConfigMap/Job wave-0 race | Explicit `sync-wave: "-1"` on `kafka-topics-configmap.yaml` (§10). |
| `-check` slow enough to annoy on deploy-only branches | Trigger is narrow (manifest + 5 deploy paths). If measured over budget, the `-check` can compare a content hash of the Go token declarations before loading — but measure first. |
| Line endings | Marker splice preserves the file's existing endings; never normalize (CLAUDE.md). |

---

## 15. Open-question resolutions

| # | PRD question | Resolution |
|---|---|---|
| 1 | `pr-sparse` vs FR-4.4's fatal rule | **Fatal rule holds unmodified.** All three overlays carry all 174 keys; §9 regenerates all three from one manifest, making the invariant true by construction. |
| 2 | Where cleanup policy is declared | **`policies.yaml`** (§6). Policy is a topic property; 2-3 declaring sites per token make a second type a permanent consistency tax with no correct conflict resolution. |
| 3 | Generator speed vs trigger breadth | **Split** (§4): a sixth analyzer in the existing shared vet sweep is the gate on `.go` changes (~free); the `go/packages` generator is the writer, on a narrow non-Go trigger. |
| 4 | `EnvProvider` fallback blast radius | **336 sites, 325 discarding — 11× the PRD estimate — but two shapes** (§11). 61 `NewConfig` wrappers → fatal-at-boot; ~264 `InitHandlers` → propagate. Codemod, not per-site judgement. No service relies on the fallback beyond the two known tokens. |
| 5 | Consumer groups | Out of scope, confirmed. `topics.yaml` roots at a **mapping** so a `groups:` key is purely additive. |
| 6 | Cashshop status-topic name change | **Safe. `STATUS_TOPIC_CASH_ITEM` has no consumer anywhere in the repo** (§12). `STATUS_EVENT_TOPIC_SKILL_MACRO`'s one consumer reads at `kafka.LastOffset`. |

---

## 16. Scope deltas from the PRD

Surfaced for approval; each is a design decision the PRD left open or got wrong.

1. **`tools/topicguard` is a new component** not in PRD §7. Required by F1; the PRD assumed the generator could enforce FR-1.4 alone.
2. **`libs/atlas-outbox` is retyped.** Missing from PRD §7; it carries the token on the transactional producer path (F3).
3. **`deploy/compose/.env.example` becomes a generated artifact.** Beyond PRD §7 (F4, §8). *The one true scope addition — drop it by deleting one renderer if unwanted.*
4. **`libs/atlas-kafka/consumer`'s `NewConfig` is NOT retyped**, contrary to FR-1.3. It takes a resolved name, not a token (§3).
5. **`gen-topic-config.sh` is deleted, not widened.** FR-3.4 permits either; deleting removes the prefix-regex class (§9).
6. **`verify.sh`'s drift step is NOT triggered on `\.go$`**, contrary to FR-5.2's literal text but satisfying its escape clause (§4c).
7. **Sizing.** 517 declarations (not ~163) and 336 `EnvProvider` sites (not ~30). The plan must size the retype phases against these numbers.
