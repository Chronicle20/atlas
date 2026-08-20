# Review: Task 7 — readinessProbe on atlas-login and atlas-channel

Commit under review: `f6e33c8b7` (parent `e96378598`)
Brief: `.superpowers/sdd/plan/task-7-brief.md`
Report: `.superpowers/sdd/plan/task-7-report.md`

## Scope

Reviewed the commit diff (`git show f6e33c8b7`, `git diff e96378598 f6e33c8b7`)
against the brief, plus the Go handler chain the probe targets:
`libs/atlas-service/bootstrap.go` (`Runtime.Ready`), `libs/atlas-rest/server/server.go`
(`MountReadiness`), and the route-registration call sites in
`services/atlas-login/atlas.com/login/main.go` and
`services/atlas-channel/atlas.com/channel/main.go`. Also ran the guard scripts
named in the brief's verification section and rendered `overlays/pr-sparse` to
confirm the probe survives kustomize overlay composition.

`scope_confirmed`: matches the brief exactly — two `readinessProbe` block
insertions in `deploy/k8s/base/atlas-login.yaml` and
`deploy/k8s/base/atlas-channel.yaml`, nothing else in the commit.

## Findings

### 1. Manifest-to-handler wiring — PASS (closes Task 5 review's not-evaluable item)

Traced end to end, not taken on the manifest's word:

- Probe: `httpGet: path: /api/readyz, port: 8080` (`deploy/k8s/base/atlas-login.yaml:39-51`,
  `deploy/k8s/base/atlas-channel.yaml:39-51`).
- Both `atlas-login/main.go:179-181` and `atlas-channel/main.go:357-360` call
  `restserver.New(l).SetBasePath("/api/").SetPort(os.Getenv("REST_PORT")).
  AddRouteInitializer(restserver.MountReadiness("/readyz", rt.Ready))` — base
  path + mount path compose to exactly `/api/readyz`.
- `REST_PORT: "8080"` in `deploy/k8s/base/env-configmap.yaml:194` — matches
  the probe's `port: 8080`.
- `MountReadiness` (`libs/atlas-rest/server/server.go:37-48`) registers a
  `GET`-only handler (`.Methods(http.MethodGet)`) that writes `200` when
  `fn()` is true and `503` when false — correct kubelet readinessProbe
  semantics (only 2xx-399 counts as success).
- `rt.Ready` is `Runtime.Ready()` (`libs/atlas-service/bootstrap.go:100-112`):
  false while `shuttingDown` is set, otherwise ANDs every registered gate.
- Both services register `service.WithReadinessGate(state.HasService)`
  (`atlas-login/main.go:72`, `atlas-channel/main.go:198`) alongside a
  `caughtUp.CaughtUpNow` gate — so the probe is gated on
  `projection.State.HasService`, exactly as the brief and D6 require.
- No auth middleware wraps the readiness route — `restserver.Builder` only
  applies `CommonHeader`/`LoggingMiddleware` (`libs/atlas-rest/server/server.go:70-71`),
  so the probe reaches the handler unauthenticated as a kubelet probe must.

Verdict: the manifest's `path`/`port` match the actual registered handler
exactly. The item Task 5's review left not-evaluable is closed.

### 2. Line-ending preservation — PASS (independently verified, not just trusted)

Compared committed blobs, not working-tree files, against the parent commit's
blobs:

- `git show e96378598:deploy/k8s/base/atlas-login.yaml` (108 lines, 52 CRLF)
  vs `git show f6e33c8b7:...` (120 lines, 64 CRLF) — the +12 line delta
  exactly accounts for the +12 CRLF-line delta.
- Byte-for-byte `cmp` of the untouched head (lines 1-33) and tail (parent
  lines 34-108 vs child lines 46-120) regions of both files returned
  identical — the *only* bytes that differ between parent and child blobs
  are the inserted block itself.
- `cat -A` on the inserted 12-line block in both files shows every line
  ending `^M$` (CRLF), consistent with the surrounding file, including the
  em-dash comment line.
- Same result for `atlas-channel.yaml` (head/tail `cmp` identical).

The implementer's claimed fix is verified, not just trusted: the committed
blobs preserve CRLF throughout, with no normalization side effect.

### 3. Diff scope — PASS

`git diff e96378598 f6e33c8b7 --stat` shows exactly:
```
deploy/k8s/base/atlas-channel.yaml | 12 ++++++++++++
deploy/k8s/base/atlas-login.yaml   | 12 ++++++++++++
2 files changed, 24 insertions(+)
```
No `go.work.sum`, no other file. The reported stash incident left no trace in
this commit.

### 4. Guard scripts and overlay rendering — PASS

- `tools/service-name-guard.sh` → `service-name-guard: clean` (exit 0).
- `tools/sparse-baseline-scoping-guard.sh` → all 4 checks PASS (exit 0).
- `kustomize build deploy/k8s/overlays/pr-sparse` renders both probes intact
  (`failureThreshold: 30`, `initialDelaySeconds: 10`, `periodSeconds: 10`,
  `path: /api/readyz`, `port: 8080` for atlas-channel; matching values for
  atlas-login save for the pre-existing `atlas-rankings`-style probe
  elsewhere in the same render, unrelated to this change).

### 5. Probe timing and crash-loop risk — PASS

`initialDelaySeconds: 10, periodSeconds: 10, failureThreshold: 30` gives a
5-minute catch-up budget, as the brief states, before the pod is marked
NotReady (not before — it's already excluded from Service endpoints once
first-probe fails, per k8s readinessProbe semantics: a pod starts as
NotReady until its first successful probe).

Crash-loop check: neither `atlas-login.yaml` nor `atlas-channel.yaml` (nor
the `atlas-rankings.yaml` pattern they copy) defines a `livenessProbe`
(`grep livenessProbe` across all three returns nothing). A readinessProbe
failure alone never restarts a container — it only removes the pod from
Service endpoints. So a pod whose service-config row never arrives
(`HasService` never true) sits NotReady indefinitely rather than
crash-looping, which is the desired behavior described in the brief/design.

### 6. Comment text — non-blocking, inherited from the brief

Both inserted blocks use identical comment wording, including the literal
sentence *"atlas-login is projection-driven — with no row it binds no
socket..."* — this sentence is copy-pasted into `atlas-channel.yaml`
(`deploy/k8s/base/atlas-channel.yaml:34-38`) verbatim, naming the wrong
service. This is not an implementer deviation — the brief's `### Step 1`
code block specifies this exact text "In each file" with no per-file
variation, and the implementer followed it exactly (confirmed: the report
explicitly notes matching the brief's wording "literally for both"). Flagging
because it ships a misleading comment in the channel manifest (a future
reader will wonder why an atlas-channel probe's rationale talks about
atlas-login), but the defect originates in the brief, not in this
implementer's execution of it — not blocking for this review, but worth a
one-line fix (e.g. "atlas-channel is projection-driven" in that file) in a
follow-up or at PR review time.

## Not evaluable

None. All items in the review brief (manifest-to-handler tracing, CRLF
preservation, diff-scope isolation, probe timing/crash-loop semantics) were
directly verifiable within this commit's scope.

## Verdict rationale

All five required checks pass with direct evidence; the only finding is a
brief-inherited comment-text inaccuracy that does not affect behavior or
correctness. This warrants APPROVED_WITH_FINDINGS rather than a fully clean
APPROVED, since the misleading comment will ship to `atlas-channel.yaml` as
committed.
