# bug: sparse envs subscribe to unsuffixed topics the baseline never publishes to

**Observed** (`atlas-pr-1411`, 2026-08-20): `atlas-login` crash-loops with

```
"error":{"message":"configuration: projection snapshot not yet published"}
"message":"Unable to find task [timeout]." log.level=fatal
```

`atlas-channel` in the same namespace is `1/1 Running` but functionally dead —
every consumer logs `Fetch deadline expired on idle topic [...]` forever.

## Root cause

The sparse overlay points the Kafka data plane at **unsuffixed** topic names.
The baseline does not publish there — `overlays/main` suffixes **all 170**
`COMMAND_TOPIC_*` / `EVENT_TOPIC_*` values with `-main`.

Live evidence:

| | value |
|---|---|
| `cm/atlas-env` in `atlas-main` | `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS = ...SERVICE_STATUS-main` |
| `cm/atlas-env` in `atlas-pr-1411` | `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS = ...SERVICE_STATUS` |

```
$ kafka-get-offsets.sh --topic EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS
EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS:0:0            # empty
$ kafka-get-offsets.sh --topic EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS-main
EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS-main:0:90      # where the data is
```

0 of 170 topic keys in the PR-env ConfigMap carry any suffix.

The wrong premise is stated verbatim in the task artifacts:

- `prd.md:383` FR-4.8 — "Sparse environments consume the **unsuffixed**
  baseline topics."
- `design.md:484`, `plan.md:219`, `plan.md:6313` — same wording.
- `deploy/k8s/overlays/pr-sparse/kustomization.yaml:247-256` — `behavior: merge`
  is chosen *specifically* so base's unsuffixed topic literals survive: "That is
  what makes topic suffixing disappear (D1, FR-4.8)".

"Unsuffixed" was treated as a synonym for "the baseline's". It is not: the
baseline's topics are suffixed `-main`. REST got this right (`ns-overrides.yaml`
resolves `NS_*` to `PLACEHOLDER_BASELINE_NAMESPACE`, verified live: 59 `NS_*`
vars = `atlas-main` on the PR ingress). Kafka has no equivalent redirect.

## Failure path in atlas-login

1. `Subscriber.Start` reads end offsets for the empty unsuffixed topic →
   `{}` → `CaughtUp` has no bars to clear and fires immediately.
   Log confirms `projection.caughtup` at `10:23:13.603`, *before* the
   service-status consumer is even assigned a partition (`10:23:16.614`).
2. `main.go:93 publishSnapshot()` runs with `state.Snapshot()` → `svc == nil`,
   so `configuration.PublishSnapshot` never closes `readyCh`
   (`configuration/registry.go:104`).
3. `main.go:155 GetServiceConfig()` blocks on `waitReady()` for the
   60s `readyTimeout` and returns `ErrNotReady`.
4. `main.go:165 l.Fatalf("Unable to find task [%s]")` → CrashLoopBackOff.

Nothing ever arrives to close the gate, because the config that atlas-login
needs lives on the `-main` topic. Both required service records **do** exist
there:

```
service:e7fb1d7e-47b8-46bd-97dc-867d93530856   # atlas-login  (SERVICE_ID in the PR pod)
service:e7fb1d7e-47b8-46bd-97dc-867d93530000   # atlas-channel
```


## The same error in two more places

Topics were the half that crashed. The same premise — "sparse uses base's
unsuffixed defaults" instead of "sparse uses the baseline's names" — was
applied to the other two shared substrates:

**Postgres.** `pr-sparse/kustomization.yaml` omitted `patches/db-name-suffix.yaml`
("D1: shared databases — no per-environment `DB_NAME` suffix"), leaving base's
unsuffixed `DB_NAME` values. Queried live:

```
$ psql -h postgres.home -c "select datname from pg_database where datname like 'atlas-%'"
atlas-accounts-main
atlas-characters-main
atlas-configurations-main
...                       # 45 rows; every one suffixed. No unsuffixed database exists.
```

Latent, not live, in `atlas-pr-1411`: neither atlas-login nor atlas-channel has
a `DB_NAME`. Any DB-backed override service — `pr-sparse/README.md`'s own
worked example is atlas-character — would have connected to a database that
does not exist.

**Redis.** `libs/atlas-redis/keys.go` derives the key prefix from `ATLAS_ENV`.
design §9 held that leaving `ATLAS_ENV` out of the sparse `atlas-env` ConfigMap
made the prefix "inert… `computeKeyPrefix("")` is already the legacy path, so
no code change is needed". Both halves are false:

```
$ kubectl exec -n atlas-pr-1411 atlas-channel-... -- sh -c 'echo $ATLAS_ENV'
a435                       # patches/consumer-group-env.yaml sets it per container
$ kubectl get pod -n atlas-main atlas-channel-... -o json | ... ATLAS_ENV
main                       # the baseline is not the empty-env path either
```

So the sparse pods keyed `a435:atlas:…` while the baseline pods keyed
`main:atlas:…` — and "inert" would have produced a third value, `atlas:`.

`ATLAS_ENV` cannot simply be repointed at the baseline, because
`libs/atlas-lock/leader.go:85` reads the same variable to scope leader-election
leases (`atlas:lock:<ATLAS_ENV>:<name>`) and those MUST stay per-deployment: an
override sharing main's lease runs none of its leader-gated work, which is
exactly the task-200 defect that put the variable there. One variable, two
incompatible jobs.

## Fix

Sparse mode adopts the **baseline's** name for every shared substrate, resolved
from `PLACEHOLDER_BASELINE_ENVIRONMENT` (CI already derives it from the ACTIVE
baseline environment record — `pr-validation.yml`'s "Resolve baseline
environment" step — so nothing hard-codes `main`). This is the symmetric
counterpart of the `NS_*` override that already does it for REST routing.

| | Change |
|---|---|
| Kafka | All 170 topic literals added to `pr-sparse`'s `atlas-env` `configMapGenerator`, suffixed `-PLACEHOLDER_BASELINE_ENVIRONMENT`. `deploy/k8s/base/kafka-precreate.sh` needs no change: it reads topic names from its own `envFrom` ConfigMap, so it becomes an idempotent `--if-not-exists` against existing baseline topics and seeds the override groups' offsets on the correct ones — what design §6.3 intended. |
| Postgres | `pr-sparse/patches/db-name-suffix.yaml` added, suffixing `-PLACEHOLDER_BASELINE_ENVIRONMENT`. `wave0-create-dbs.yaml` stays excluded — the baseline's databases already exist. |
| Redis | `libs/atlas-redis` reads a new `ATLAS_REDIS_ENV` in preference to `ATLAS_ENV`, falling back when unset; `pr-sparse` sets it to the baseline's environment. `ATLAS_ENV` stays per-deployment for consumer groups and `libs/atlas-lock`. |
| Generators | `gen-topic-config.sh` and `gen-db-name-suffix.sh` take the suffix token (and, for the latter, the overlay) as arguments. Default invocations reproduce `overlays/pr`'s checked-in output byte-for-byte. |
| Guard | `tools/sparse-baseline-scoping-guard.sh` renders `pr-sparse` and asserts all four invariants; wired into `tools/verify.sh`. Confirmed it flags the pre-fix tree: 170/170 topics and 36/36 `DB_NAME`s unsuffixed, `ATLAS_REDIS_ENV` unset. |

Rejected alternative: de-suffixing `overlays/main`. That orphans every committed
offset and all retained data on the `-main` topics, and every `-main` database,
for the entire baseline — for no gain.

`kustomize build` output for `overlays/pr` and `overlays/main` is byte-identical
before and after; the `pr-sparse` delta is exactly 485 lines — 170 topics ×2,
36 `DB_NAME`s ×2 (name+value), and the one `ATLAS_REDIS_ENV` line.

## Not investigated

- **MinIO.** `isolation-audit.md` §5 lists it as "currently restored per
  environment" and unaudited. `atlas-minio-init` ran in `atlas-pr-1411`. Whether
  bucket naming has the same baseline/base split was not checked here.
- **Which Redis namespaces are genuinely cross-service.** The prefix is now
  correct regardless, but the audit's §4.2 namespace inventory was taken as
  read rather than re-swept.
