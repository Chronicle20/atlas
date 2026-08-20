# Review: Task 12 — CI renders SERVICE_ID and resolved consumer-group names

Commit reviewed: `7d8bb365f` (`.github/workflows/pr-validation.yml`, +141/-0)
Brief: `.superpowers/sdd/plan/task-12-brief.md`
Report: `.superpowers/sdd/plan/task-12-report.md`

## Scope

Single-commit diff to `.github/workflows/pr-validation.yml`'s `update-pr-overlay`
job, `Substitute placeholders in PR overlay` step: a new loop that fills
`PLACEHOLDER_SERVICE_ID_BLOCK` / `PLACEHOLDER_PRECREATE_GROUPS_BLOCK` (both
introduced by Task 11 in `deploy/k8s/overlays/pr-sparse/kustomization.yaml`),
plus an offline `kustomize build` verification step. Matches the brief's file
scope exactly (`git show --stat` confirms one file). No other file touched by
this commit.

## Findings

### 1. The `GROUPS` rename — verified, correct, and complete

Independently reproduced the claim in a clean bash 5.2.37 shell:

```
$ bash -c 'GROUPS=""; GROUPS="${GROUPS}hello\n"; echo "len=${#GROUPS}"; declare -p GROUPS'
len=4
declare -a GROUPS=([0]="1000" [1]="24" ...)
```

`GROUPS` is bash's own special array (list of the caller's numeric group
IDs). Assignment to it is silently a no-op — the array snaps back to the
group-membership list on the very next reference, and `${#GROUPS}` reports
the *string length of element 0*, not "did my assignment take." Using the
brief's literal `GROUPS` variable name would have produced an empty
`PRECREATE_GROUPS_BLOCK` while every intermediate echo/log line looked
plausible. This is a real, high-value catch, and the implementer's report
documents the exact repro command that proves it (`.github/workflows/pr-validation.yml:1029-1032`, comment before `GROUP_LIST=""`).

Swept the rest of the block this commit adds
(`.github/workflows/pr-validation.yml:1033-1102`) for any other bash special
name used as an assignment target: `SERVICE_ID_BLOCK`, `JOB_ENV_OPS`,
`GROUP_LIST`, `svc`, `base`, `stype`, `cname`, `sid`, `var`, `grp`, `RENDERED`,
`base_sid`, `want_sid`, `rendered_sid`, `job_sid`, `precreate_groups`. None of
these collide with any bash special/reserved variable (`REPLY`, `IFS`,
`PS1..4`, `OPTIND`, `RANDOM`, `SECONDS`, `LINENO`, `PIPESTATUS`, `BASH_*`,
`UID`/`EUID`/`PPID`/`HOSTNAME`/`GROUPS`). No partial fix — the fix is complete
for this commit's surface.

**PASS.**

### 2. Substring-collision hazard — no collision found

`grep -rn "PLACEHOLDER_SERVICE_ID_BLOCK\|PLACEHOLDER_PRECREATE_GROUPS_BLOCK" deploy/k8s/overlays/pr-sparse/ deploy/k8s/overlays/pr-cleanup/`
returns exactly the two anchor lines this commit targets
(`kustomization.yaml:269` and `:279`) — no README/prose collision remains, and
none was introduced by this commit's own doc comments (the new block comment
at `:1009-1032` lives in the workflow file, which is not itself subject to
this `sed` pass — the pass only walks `$OVERLAY_DIR` and `pr-cleanup`).

The `sed -i ... -e "s${D}PLACEHOLDER_SERVICE_ID_BLOCK${D}...${D}g"` uses the
same `\x01` control-character delimiter as the pre-existing `DELETE_BLOCK` /
`NS_OVERRIDES` substitutions, for the same documented reason (payload
contains YAML `|-` block-scalar markers). Payload content (`$sid`, `$grp`,
`$var`, `$cname`, `$svc`) is entirely data-derived (UUIDs, service names,
resolved group strings) — none of it can contain the literal string
`PLACEHOLDER_SERVICE_ID_BLOCK` or `PLACEHOLDER_PRECREATE_GROUPS_BLOCK`, so the
payload cannot reintroduce a matchable token substring for a second pass over
the same files (this `sed` invocation is a single pass over both new `-e`
expressions and the three pre-existing ones, run once — not iterated).

Independently reproduced end-to-end: copied `deploy/` and
`tools/derive-service-id.sh` to a scratch dir, extracted the exact `run:`
block via `yaml.safe_load`, and ran it with
`OVERRIDE_SERVICES="atlas-login atlas-channel" PR_NUMBER=1411`. Output matches
the report's exactly, including the `(clean)` leak-sweep and full
`kustomize build` verification pass (see Finding 4).

**PASS.**

### 3. The JSON-6902 `op: add` on the whole `env` path — verified against base

Read `deploy/k8s/base/atlas-kafka-precreate.yaml` directly: the
`kafka-precreate` container has `envFrom: [{configMapRef: {name: atlas-env}}]`
and **no `env:` key at all**. `op: add` at
`/spec/template/spec/containers/0/env` therefore creates the array rather than
replacing an existing one — the implementer's stated reasoning matches the
actual base file. Confirmed further by the independent `kustomize build`
render in the reproduction above: `atlas-kafka-precreate`'s rendered Job
carries exactly the two-line `KAFKA_CONSUMER_GROUP` value with no other `env`
entries lost (there were none to lose).

If a future change ever adds a base `env:` key to this container, whole-path
`add` would then replace it wholesale — worth a one-line comment, but that is
a latent risk against a base file this commit did not touch and does not
currently have, not a defect in this commit. Noted as non-blocking.

**PASS** (with a non-blocking forward-looking note).

### 4. Derived ids — verified independently, no hardcoding

Ran `tools/derive-service-id.sh login-service pr-1411` and
`tools/derive-service-id.sh channel-service pr-1411` directly:

```
6439ca9c-d28d-5db9-821b-8dd93d318a25
5a86d8e6-3167-5e74-9fc5-021d94001da2
```

Both match the brief's and the report's claimed values exactly.
`git show 7d8bb365f | grep -iE "6439ca9c|5a86d8e6"` finds nothing — no
hardcoded UUID literal anywhere in the diff; every id is produced by
`tools/derive-service-id.sh`, the single derivation site.

No-trailing-newline contract: `tools/derive-service-id.sh` (unchanged by this
commit, read for reference) ends its `python3` heredoc with
`sys.stdout.write(...)` — no `\n`. Every use in this commit's new code is
inside `$(...)` command substitution (`sid=$(tools/derive-service-id.sh ...)`),
which strips trailing newlines regardless, so this is belt-and-suspenders
correct either way, consistent with Task 1's contract.

**PASS.**

### 5. Unfilled-`PLACEHOLDER_` sweep and `pr`-overlay non-interference

Reproduced both branches independently in a scratch copy of `deploy/`:

- `MODE=sparse`, `OVERRIDE_SERVICES="atlas-login atlas-channel"`: leak sweep
  reports `(clean)`; the offline `kustomize build` verification step passes
  all three assertions (rendered `SERVICE_ID` per Deployment matches
  derived and differs from base; `atlas-pr-bootstrap` carries a matching
  `SERVICE_ID_<TYPE>`; `atlas-kafka-precreate` carries a non-empty, `%`-free
  `KAFKA_CONSUMER_GROUP`) — output byte-identical to the implementer's
  report.
- `MODE=pr` (non-sparse): the entire `if [ "$MODE" = "sparse" ]` block —
  including this commit's new loop and its offline-render verification step —
  is skipped entirely; leak sweep still reports `(clean)` because
  `deploy/k8s/overlays/pr/` never contains either new token (confirmed by the
  same grep in Finding 2, which found matches only under `pr-sparse/`).

**PASS.**

## Not evaluable

None. All five review questions were answerable from the commit diff plus the
two directly-referenced files (`deploy/k8s/base/atlas-kafka-precreate.yaml`,
`tools/derive-service-id.sh`), and all were independently reproduced rather
than taken on the implementer's word.

## Verdict rationale

No blocking defects found. The `GROUPS`→`GROUP_LIST` deviation from the
brief's literal pseudocode is a correct, well-evidenced, complete fix for a
real silent-failure hazard, not a scope violation — the brief's own intent
(a working `PRECREATE_GROUPS_BLOCK`) is served better by the deviation than by
literal compliance. The `op: add` deviation is verified against the actual
base file. One non-blocking forward-looking note (whole-path `add` would
become destructive if a future commit adds an `env:` key to
`atlas-kafka-precreate`'s base container) is recorded above but does not
block this commit, since no such key exists today.
