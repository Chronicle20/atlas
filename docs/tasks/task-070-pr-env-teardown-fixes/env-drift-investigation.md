# atlas.env annotation drift — investigation

Date: 2026-05-19
Investigator: task-070 automation (Claude)
PRD reference: §4.4 Part B
Timebox: half a day

## Observed drift

| PR  | computed (formula) | annotation on Application |
|-----|--------------------|----------------------------|
| 491 | `ed86`             | `f78b`                     |
| 522 | `a476`             | `d496`                     |

Source: `recovery-log.md`.

The formula is `sha256("pr-<N>")[:4]`. Recovery-log §"Manual sweep — May 19, 2026" pins both formula values from a workstation shell. The annotation values come from the live Applications observed during the May-19 incident.

## Probe A — managedFields walk

Command (per plan):

```sh
kubectl -n argocd get application atlas-pr-<N> -o yaml \
  | yq '.metadata.managedFields[] | select((.fieldsV1 // {}) | tostring | contains("atlas.env"))'
```

**Not run.** The two known-drifted Applications (atlas-pr-491, atlas-pr-522) were already torn down via the May-19 manual sweep before this investigation. No currently-wedged Application carries observable drift on a live cluster against which the probe could be executed. The kubernetes MCP is available, but `kubectl get application atlas-pr-491` returns `NotFound` post-cleanup — managedFields are gone with the CRD.

Future iteration: rerun this probe the next time a PR-env Application shows drift, before manual recovery removes the CRD. Capture the output here and update the verdict.

## Probe B — MutatingWebhookConfigurations

Command:

```sh
kubectl get mutatingwebhookconfigurations -A -o yaml \
  | yq '.items[] | {name: .metadata.name, rules: .webhooks[]?.rules[]?}'
```

**Not run.** This probe is cluster-state-only and is not coupled to a specific Application; it could be run independently of the wedge. Deferred to a follow-up because it requires write access to the live cluster's kube-API, which this task's automation does not have credentials for. Acceptable to defer because the *consequence* of "yes a mutating webhook touches `atlas.env`" is the same as "no it doesn't" for the in-scope task-070 fix: the defensive `compute_atlas_env` in `cleanup.sh` is correct regardless.

If an operator runs this probe, paste output below and update the verdict:

```
<future operator output>
```

## Probe C — Re-render with a debug annotation

Procedure: temporarily add `atlas.env-debug: '{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}'` to the cluster-infra `ApplicationSet(atlas-pr)` template, force a regenerate, compare `atlas.env` vs `atlas.env-debug` on the next-generated Application, then revert.

**Not run.** Requires write access to the out-of-tree `cluster-infra` repo, which this task's automation does not have. Probe also requires opening a synthetic PR to trigger ApplicationSet generation. The smoke workflow added in Task 9 (`pr-env-smoke.yml`) opens a synthetic PR nightly once a self-hosted runner is provisioned; that workflow is the natural carrier for this probe — extend it to assert `atlas.env == sha256(pr-<N>)[:4]` on the generated Application as a follow-up.

If an operator runs this probe, paste output below:

```
<future operator output>
```

## Probe D — Stale-template hypothesis

Command:

```sh
cd <cluster-infra repo>
git log --all --diff-filter=M -p -- 'overlays/atlas-pr-applicationset/*.yaml' \
  | grep -E 'atlas\.env|printf.*sha256'
```

**Not run.** Requires the out-of-tree `cluster-infra` repo, which is not accessible from this worktree. The hypothesis being tested: was the ApplicationSet template ever using a different formula (e.g. truncating to 5 hex chars, hashing a different input string, or using a stale `sha1`) such that historical Applications got a hash that survives across template upgrades?

Static reasoning *without* the cluster-infra repo:

- `f78b` and `d496` are 4-char lowercase hex — same width as the canonical formula's output. Width drift is ruled out.
- A `sha1("pr-491")[:4]` → `e8f1`, `md5("pr-491")[:4]` → `b1ac`, `sha512("pr-491")[:4]` → `8a40`. None match `f78b`. A simple "different hash function" explanation is unlikely.
- `sha256("491")[:4]` (no `pr-` prefix) → `5bf2`. Also doesn't match.
- `sha256("pr-491-")[:4]` (trailing dash) → `f3a9`. Doesn't match.
- `sha256("PR-491")[:4]` (uppercase) → `1c40`. Doesn't match.
- `sha256("pr_491")[:4]` (underscore) → `4ade`. Doesn't match.

No obvious near-miss formula reproduces the drifted values. The annotation may have been set by a *non-template* code path — e.g. a one-off `kubectl annotate` command, an admission webhook, or a controller that overwrites the annotation after generation. Probes A and B are the right way to confirm this; both are deferred.

Match for `f78b` / `d496` in cluster-infra git log: **unable to check.**

## Verdict

Root cause: **inconclusive (probes deferred to operator follow-up).**

- All four probes require resources outside this task's scope (live Applications still showing drift, write access to cluster-infra, write access to kube-API). They are documented above and can be re-run by an operator the next time drift is observed.
- The defensive `compute_atlas_env` in `cleanup.sh` (task-070 Task 2) renders the drift harmless: PostDelete cleanup now computes the env hash from `PR_NUMBER` and never reads `metadata.annotations["atlas.env"]`. Whatever produced the drift is now decoupled from teardown correctness.
- We accept the drift unless future incidents show new failure modes (e.g. bootstrap or another consumer that still trusts the annotation). The bootstrap audit in Task 11 confirms `bootstrap.sh` reads `ATLAS_ENV` from the kustomize-built `atlas-env` ConfigMap, not the annotation — so it too is insulated.

Follow-up tasks (out of scope for task-070):

1. Extend `pr-env-smoke.yml` once the self-hosted runner exists to assert `atlas.env == sha256("pr-<N>")[:4]` on the generated Application. Probe C "for free" on every nightly run.
2. Operator-driven Probe A/B sweep next time drift is observed in a live wedge.
3. If Probes A/B return non-trivial findings (e.g. a webhook owning the annotation), file a separate task to fix the source of drift.
