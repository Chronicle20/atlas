# Review — Task 9: `bootstrap.sh` upsert on the derived id, over the override set

Commit reviewed: `100764219` only (`e014e2f03`/`e1317e3ad` = Task 11, `e32c9ef15` = Task 13 — excluded).

Brief: `.superpowers/sdd/plan/task-9-brief.md`
Report: `.superpowers/sdd/plan/task-9-report.md`

## Scope

Reviewed the full diff of `100764219`:
- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` (+128/-72)
- `services/atlas-pr-bootstrap/test/bootstrap_test.bats` (+70/-25)
- `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml` (+6)
- `deploy/k8s/overlays/pr/sync-bootstrap.yaml` (+6)

Plus read-only reference to `services/atlas-pr-bootstrap/scripts/service-config.sh`
(`build_service_config`/`build_login_entry`/`build_channel_entry`/`merge_tenant_entry`,
landed by Task 8) since `upsert_sparse_service_config` depends on its exact
signature, and `tools/derive-service-id.sh` for the Task 1 contract this task
is required to pin.

`scope_confirmed`: matches the brief's stated Files list plus the
brief-mandated mirror-pair RBAC comment edit; no code outside this surface
was needed to evaluate correctness.

## 1. Deviation from the brief's six-row test table — verified, not a defect

Confirmed directly:

```
$ ls services/atlas-pr-bootstrap/canonical/services/
channel-service.json  drops-service.json  login-service.json
```

Only three canonical templates exist; `world-service.json`,
`character-factory.json`, `drops-information-service.json` do not exist
anywhere in git history for that directory (`git log --oneline --all -- .../canonical/services/`
shows only the two commits that created the three files that exist). The
brief's own Step 1 text anticipated exactly this and instructed: "if the
other three have no canonical template, the table carries only the three
that do... Do not fabricate a template file."

`svc_table_lookup` (`bootstrap.sh:527-536`) implements a 3-row table
(`atlas-login`, `atlas-channel`, `atlas-drops`) and returns non-zero for
anything else; `upsert_sparse_service_config`'s "unmapped -> log info,
return 0" branch (`bootstrap.sh:552-555`) is the single code path that
handles both "not in the table" and "no template," exactly as the brief's
prose describes. The bats test (`bootstrap_test.bats:106-131`) asserts the
triple for the three mapped deployments and non-zero status for the three
unmapped ones. This is the correct call — the premise holds, and grounding
the test in what exists rather than an invented value is what the repo's
"never fabricate" convention requires. Not a defect.

## 2. Scope of the two `sync-bootstrap.yaml` touches — in scope, correctly mirrored

The brief's Files list names only `bootstrap.sh`/`bootstrap_test.bats`, but
its prose explicitly requires the RBAC comment edit as part of this task:
"the NFR 'Security' clause asks for an explicit re-examination... Record
that conclusion in a comment on the Role's `verbs` line... This file is
mirror-locked... make the identical comment edit in
`deploy/k8s/overlays/pr/sync-bootstrap.yaml` in the same commit, or the
guard fails." So the two yaml touches are not scope creep — they are a
named requirement the "Files" header simply under-listed.

Both files carry byte-identical added comment blocks
(`deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml:15-20` and
`deploy/k8s/overlays/pr/sync-bootstrap.yaml:15-20`, confirmed via `git show`
on both hunks — identical text). `tools/pr-sparse-mirror-guard.sh` passes:

```
pr-sparse-mirror-guard: up to date
```

Not a defect.

## 3. Task 1 contract (`derive-service-id.sh` no-trailing-newline) — BLOCKING gap

`tools/derive-service-id.sh:11` states the contract explicitly: "Prints the
derived UUID to stdout with no trailing newline." The brief requires Task 9,
as "the first real call site," to carry a test asserting this at the call
boundary, and requires the shell not to assume a trailing newline.

The actual call boundary is `bootstrap.sh:558-563`
(`upsert_sparse_service_config`):

```sh
svc_id_var=$(svc_id_var_name "$type")
svc_id="${!svc_id_var:-}"
if [ -z "$svc_id" ]; then
```

`svc_id="${!svc_id_var:-}"` is direct indirect parameter expansion, not a
`$(...)` command substitution — it does not trim or otherwise assume
anything about trailing bytes, so the code itself does not misbehave. But
the new test that the report and the implementer's self-review claim "pins
the no-trailing-newline contract... at this call boundary"
(`bootstrap_test.bats:141-152`, `svc_id_var_name does not assume or append a
trailing newline`) does not exercise that boundary at all:

```sh
@test "svc_id_var_name does not assume or append a trailing newline" {
    load_fn svc_id_var_name
    local out
    out=$(svc_id_var_name login-service | wc -c)
    [ "$out" -eq 24 ]
}
```

`svc_id_var_name` (`bootstrap.sh:539-541`) takes a **service type** string
(`"login-service"`) and returns a **variable name** string
(`"SERVICE_ID_LOGIN_SERVICE"`). It never reads, touches, or passes through
the derived UUID that `tools/derive-service-id.sh` produces. The test
asserts that `svc_id_var_name`'s own return value (the 24-byte variable
*name*, built entirely from `printf`/`tr` on the caller-supplied type
string) carries no stray newline — a true statement, but about the wrong
value. It provides zero coverage of what happens when
`SERVICE_ID_LOGIN_SERVICE` (the actual env var carrying
`derive-service-id.sh`'s UUID output) is read via `${!svc_id_var:-}` and
flows into `svc_id`, `build_service_config`'s id-format regex
(`service-config.sh:87`), the GET URL, or the POST/PATCH body.

No test in this commit sets `SERVICE_ID_LOGIN_SERVICE` (or any
`SERVICE_ID_<TYPE>`) to a well-formed UUID and asserts it flows through
`upsert_sparse_service_config` byte-identically — the only test that
exercises that variable at all is the "absent" case
(`bootstrap_test.bats:154-165`), which unsets it and asserts failure, not
the presence/pass-through path this brief clause is about.

This is a genuine gap between what the brief required and what the report
claims was delivered ("Confirmed `derive-service-id.sh`'s no-trailing-newline
contract is pinned at the `svc_id_var_name` call boundary via a dedicated
test" — task-9-report.md self-review section — is incorrect: that function
is not the call boundary for the UUID value at all). Blocking: either
retarget the existing test at the real boundary (assert
`${!svc_id_var:-}` with a no-trailing-newline UUID set in the env produces
an unmodified `svc_id`, e.g. via `wc -c` on the value or a regex-pass
assertion through `build_service_config`), or add a new one; the current
test does not satisfy the brief clause it is credited with satisfying.

## Correctness of the change itself

- `svc_table_lookup` (`bootstrap.sh:527-536`): pure case-statement lookup,
  correct triples for the three mapped deployments, confirmed by test.
- `svc_id_var_name` (`bootstrap.sh:539-541`): `printf`+`tr` uppercase/`-`→`_`
  conversion, verified correct for all three brief examples
  (`bootstrap_test.bats:134-138`), and no trailing newline on its own output
  (though, per finding 3, that isn't the contract that needed pinning).
- `upsert_sparse_service_config` (`bootstrap.sh:557-621`):
  - id-absent guard runs before any I/O — confirmed by test
    (`bootstrap_test.bats:154-165`) and code order (`bootstrap.sh:559-563`
    precedes the first `curl` at `bootstrap.sh:565`).
  - Existing-row branch reuses the same `jq -cS` idempotency comparison as
    `upsert_service_config` (verified textually identical pattern at
    `bootstrap.sh:490` vs `bootstrap.sh:583`), correctly dodging the
    PATCH-handler panic on tenant-agnostic (`shape=none`) configs by passing
    `new_attrs="$live_attrs"` unchanged when `entry=""`.
  - Absent-row branch uses `"${ENV_HEADER[@]}"` (built by `env_header_init`,
    `bootstrap.sh:55-62`) instead of the old hardcoded `-H "ENVIRONMENT: ..."`
    — correct, and matches the brief's POST instruction verbatim.
  - `build_service_config "$shape" "$tmpl" "$svc_id"` call signature matches
    Task 8's landed `build_service_config() { local shape="$1" tmpl="$2"
    id="${3-}" ...}` (`service-config.sh:80`) exactly — confirmed by reading
    that function.
- Loop replacing the three hard-coded calls (`bootstrap.sh:631-632`) matches
  the brief's pseudocode exactly: `env_record_get | jq -r
  '.data.attributes.overrides // {} | keys[]'` piped into `for d in
  $overrides; do upsert_sparse_service_config "$d" || exit 1; done`.
- `kubectl set env deployment/... SERVICE_ID=...` and its
  "SERVICE_ID routing (task-47)" comment block are fully deleted (confirmed:
  no `kubectl set env` remains anywhere in the diff or resulting file); a
  new comment block explains why sparse mode no longer needs it
  (`bootstrap.sh:625-632`) — matches FR-1.2 as the brief specifies.
- Restart loop: `atlas-login`/`atlas-channel`'s sparse-only restart
  conditional (the whole `if [ "${ATLAS_MODE...}" = "sparse" ]; then
  restart_targets="atlas-login atlas-channel $restart_targets"; fi` block)
  is deleted entirely, leaving the base `restart_targets="atlas-drops
  atlas-character-factory atlas-world"` unconditional for both modes — this
  matches the brief ("remove them from `restart_targets`' sparse branch...
  leave the isolated-mode `restart_targets` untouched"; isolated mode never
  had a branch, so its behavior is unaffected by this deletion).
- Two stale in-file comment references to `create_service_config` (near
  `env_header_init` and `record_environment_tenant`) were updated to
  `upsert_sparse_service_config` — confirmed at `bootstrap.sh:50` and
  `bootstrap.sh:358`.

## Test honesty

Ran the actual suite (not just trusting the report's numbers):

```
$ bats services/atlas-pr-bootstrap/test/bootstrap_test.bats
1..8
ok 1 bootstrap.sh fails without ATLAS_ENV
ok 2 bootstrap.sh fails without ATLAS_UI_BASE
ok 3 bootstrap.sh fails fast when no canonical baseline (404)
ok 4 bootstrap.sh reports MinIO-unreachable distinctly (000)
ok 5 sparse service table maps every SERVICE_ID-carrying deployment
ok 6 service id env var name is derived from the service type
ok 7 svc_id_var_name does not assume or append a trailing newline
ok 8 sparse service-config step fails when the CI-rendered id is absent
```

All 4 new tests pass and are meaningful for what they assert (each fails if
you revert the corresponding code change), except test 7, whose assertion —
while true — does not test the property the brief and report claim it
tests (see finding 3).

`tools/shell-guard.sh --require-shellcheck` →
`shell-guard: 71 script(s) OK (syntax + shellcheck -S error).` — reproduced,
passes.

`tools/pr-sparse-mirror-guard.sh` → `pr-sparse-mirror-guard: up to date` —
reproduced, passes.

## Not evaluable

- The network-touching GET/PATCH/POST paths inside `upsert_sparse_service_config`
  (`bootstrap.sh:565-621`) are untested by bats (out of scope per the
  brief: "The network-touching parts are out of scope for bats"). Correctness
  of the merge/PATCH/POST logic against a live `atlas-configurations` server
  is not evaluable from this review surface. Not counted as a defect since
  the brief explicitly scoped it out, but noted for completeness.

## Verdict rationale

Two of the three requested judgment calls (the six-row table deviation, and
the yaml scope) resolve in the implementer's favor — both are correct,
brief-conforming decisions. The third (the Task 1 no-trailing-newline
contract) does not: the brief's specific ask — a test pinning that contract
at the real call boundary — is not met; the delivered test exercises an
unrelated function and the report/self-review incorrectly credits it as
satisfying the requirement. The underlying shell code itself is safe (direct
parameter expansion, no `$()` truncation), so this is a missing/misdirected
test, not a runtime bug — but it is a brief requirement not delivered, and
the report's claim to have delivered it is inaccurate, so it is blocking
rather than a note.
