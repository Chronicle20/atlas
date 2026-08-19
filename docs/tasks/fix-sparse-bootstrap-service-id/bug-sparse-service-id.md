# Bug: sparse bootstrap strips SERVICE_ID, crash-looping atlas-channel and atlas-login

**Status:** root-caused, fixed on `fix-sparse-bootstrap-service-id`
**Found on:** PR #1411's sparse environment, immediately after #1416 made the
sparse overlay reach Argo at all. This is the *second* defect on the same
first-ever sparse deploy, not a regression from #1416.

## Symptom

Two pods per workload for `atlas-channel` and `atlas-login` — one Running,
one in `Error`/CrashLoopBackOff:

```
atlas-channel-687b6ccdbf-xwn5g   1/1   Running   0
atlas-channel-7bff746bb9-crrsh   0/1   Error     4
atlas-login-6b5694fb58-5kcgc     0/1   Error     4
atlas-login-947496775-wf4rq      1/1   Running   2
```

Not duplicated workloads: each Deployment has one new ReplicaSet that cannot
become Ready, so the old one is never retired. The "working" pod is the old,
correct revision still serving.

```
panic: uuid: Parse(): invalid UUID length: 0
github.com/google/uuid.MustParse({0x1f5c1d738040, 0x0})
main.main()  /app/services/atlas-channel/atlas.com/channel/main.go:181
```

## Root cause

`uuidgen` is not installed in the atlas-pr-bootstrap image. The Dockerfile's
`apk add` list is `bash curl jq postgresql-client redis ca-certificates
github-cli kubectl unzip` — no `util-linux`.

`build_service_config` calls it **only** in the sparse branch
(`service-config.sh:82`), which is why isolated mode never hit this:

```sh
jq -c --arg id "$(uuidgen)" ...
```

Command substitution swallows the missing-binary failure into an empty
string, so `.data.id` became `""`. From the bootstrap log:

```
/atlas/service-config.sh: line 82: uuidgen: command not found
{"level":"info","msg":"sparse service config  created (type=login, environment=pr-1411)"}
{"level":"warn","msg":"could not set SERVICE_ID= on deployment/atlas-drops"}
```

The chain from there:

1. `svc_id=""`.
2. The POST still succeeds. atlas-configurations mints its own id, because
   `configurations/services/processor.go` falls back to `uuid.New()` when
   `input.Id == ""` — so bootstrap never learns the row's real id.
3. `bootstrap.sh:409` runs `kubectl set env deployment/atlas-channel
   SERVICE_ID=`. That does **not** fail; it writes an env entry with no
   value. Confirmed on the live ReplicaSet: the old one has
   `SERVICE_ID=e7fb1d7e-…`, the new one has bare `{"name": "SERVICE_ID"}`.
4. The new pod panics on `uuid.MustParse("")` and crash-loops forever.

## Second defect, found while tracing the first

`create_service_config`'s POST sends no `ENVIRONMENT` header. The column is
server-owned:

```go
// configurations/services/administrator.go
"INSERT INTO services (id, type, data, environment) VALUES (?, ?, ?, ?)",
    serviceId, serviceType, data, string(env.MustFromContext(ctx)),
```

and `processor.go` states it explicitly — *"Environment is server-owned … the
Entity column always wins over whatever e.Data's JSON blob happened to
contain"*. So the body's `.data.attributes.environment` that
`build_service_config` sets is ignored, and a caller with no header is the
legacy `""` environment (`scope.AuthorizeWrite`: `caller == ""` is always
authorized).

Every sparse row therefore landed with `environment=''` in the **shared
baseline** atlas-configurations (sparse routes non-override services to
`atlas-main`). `cleanup.sh`'s reclaim selects on
`.attributes.environment == $env`, so those rows can never be reclaimed at
teardown — a permanent three-row leak per bootstrap run.

Observed live: 6 orphan rows after 2 bootstrap runs, against main's 4
correctly-stamped rows. They were deleted manually (each re-verified as
`environment == ""` immediately before the DELETE; main's `[main]` rows were
never touched).

## Why the test suite missed it

`service_config_test.bats`'s "sparse mode never reads or writes the pinned
main service row" asserts only that the id `!= pinned` and `!= "null"`. The
empty string satisfies both.

## Fix

1. **Dockerfile** — add `util-linux`. Verified in a real `alpine:3.24`
   container that this package provides `/usr/bin/uuidgen`.
2. **`new_uuid()`** — replaces the inline `$(uuidgen)`. Prefers `uuidgen`,
   falls back to `/proc/sys/kernel/random/uuid` (kernel-provided, no package
   needed), and validates the 8-4-4-4-12 shape, failing loudly instead of
   returning `""`. Uses only bash builtins (`read`, `[[ =~ ]]`, `printf`)
   apart from `uuidgen` itself, so it cannot itself be defeated by a missing
   tool.
3. **`create_service_config`** — refuses to continue on an empty/`null` id
   *before* reaching `kubectl set env`; sends
   `-H "ENVIRONMENT: $ATLAS_ENVIRONMENT"` on the POST; and fails on a failed
   POST rather than proceeding to repoint a Deployment at a row that was
   never created.
4. **Callers** — `|| exit 1` on the three sparse `create_service_config`
   calls. This is belt-and-braces, not the mechanism: `bootstrap.sh:21`
   restores `set -e` after lib.sh relaxes it to `set -uo pipefail`, so a bare
   call to a function returning non-zero already aborts. What actually makes
   a bad row fatal is (3)'s guards returning non-zero at all — previously
   nothing did. The `||` keeps these calls fatal if `set -e` is ever relaxed
   again, which lib.sh has already done once.

   *(An earlier draft of this document and of the code comment claimed lib.sh
   leaves `set -e` off. That was wrong — caught in review, verified against
   `bootstrap.sh:14-21`.)*

## Verification

Replayed the exact production condition — `origin/main`'s script with
`uuidgen` absent from `PATH`:

| | result |
|---|---|
| pre-fix | `uuidgen: command not found`, **exit 0**, `id=[]` |
| post-fix | **exit 1**, `new_uuid: could not generate a UUID (got '')` |

The pre-fix row reproduces the production log line verbatim, including the
silent success.

10 new bats assertions across `service_config_test.bats`,
`dockerfile_test.bats`, and `bootstrap_test.bats`. Note `setup()` in
`service_config_test.bats` now sources `lib.sh` before `service-config.sh`,
mirroring bootstrap.sh's own order — the latter calls `log()` without
sourcing it, so the error paths previously died with `log: command not
found` (127) rather than emitting the asserted message.

## Deliberately not changed

Sparse calls `create_service_config … atlas-drops` and restarts
`atlas-drops atlas-character-factory atlas-world`, none of which are deployed
in sparse mode. That is the source of the benign `could not restart` warnings
in the bootstrap log. It is a task-232 scoping question, not a cause of this
failure, and changing the sparse override set is out of scope here.
