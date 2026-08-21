# Bug: sparse ephemeral environments never deploy — Argo renders an unresolved overlay

**Status:** root-caused, fixed on `fix-sparse-overlay-argo-path`
**Found on:** PR #1411 (`fix-whisper-cross-channel-delivery`), the first PR to
select task-232's `sparse` mode.
**Severity:** sparse is the default mode, so this broke the ephemeral
environment for every PR not labelled `atlas:isolated`.

## Symptom

`atlas-pr-1411` sat `OutOfSync` / `Missing` with the sync operation
retry-looping. All 26 PR-validation checks were green, and the mode report
comment was correct (`sparse`, 2 of 76 workloads, override set
`atlas-channel` + `atlas-login`).

```
$ kubectl get application atlas-pr-1411 -n argocd \
    -o jsonpath='{.status.operationState.message}'
one or more objects failed to apply, reason: Namespace
"atlas-pr-PLACEHOLDER_PR_NUMBER" is invalid: metadata.name: Invalid value:
"atlas-pr-PLACEHOLDER_PR_NUMBER": a lowercase RFC 1123 label must consist of
lower case alphanumeric characters or '-' [...]
namespaces "atlas-pr-PLACEHOLDER_PR_NUMBER" not found [...]
Retrying attempt #4 at 3:32PM.
```

The `atlas-pr-1411` namespace itself existed — Argo's `CreateNamespace=true`
creates it from `spec.destination.namespace`, which is correct. It is the
*rendered manifests* that carried the invalid namespace, so nothing was ever
created inside it.

## Root cause

Task-232 made the producer mode-aware and left the consumer hard-coded.

`.github/workflows/pr-validation.yml` branches on mode when choosing which
overlay to resolve:

```sh
if [ "$MODE" = "sparse" ]; then
  OVERLAY_DIR=deploy/k8s/overlays/pr-sparse
else
  OVERLAY_DIR=deploy/k8s/overlays/pr
fi
```

Substitution, image-tag bumping, and the `git add` that builds
`bot/pr-<N>-resolved` all follow that variable. In sparse mode
`deploy/k8s/overlays/pr/` is never touched.

But `ApplicationSet(atlas-pr)` pins the path literally:

```json
{"path": "deploy/k8s/overlays/pr",
 "targetRevision": "bot/pr-{{.number}}-resolved"}
```

So on `bot/pr-1411-resolved`, `pr-cleanup/` and `pr-sparse/` were both fully
resolved, while `pr/` — the only directory Argo reads — still contained
`namespace: atlas-pr-PLACEHOLDER_PR_NUMBER`
(`deploy/k8s/overlays/pr/kustomization.yaml:25`).

## Why every guard missed it

1. **The ApplicationSet is out of repo.** It lives in cluster-infra; `deploy/`
   contains no `kind: ApplicationSet`. No in-repo guard can see which path it
   pins, and `verify.sh` structurally cannot cover the seam.
2. **The leak guard scanned the wrong directory.** The substitution step's
   `PLACEHOLDER_` check globs `$OVERLAY_DIR` + `pr-cleanup` — that is, the
   directory it just resolved. A clean `pr-sparse/` reported green while the
   published `pr/` was untouched.
3. **Task-232 never modelled it.** `design.md` does not mention ApplicationSet
   at all, and `plan.md` has no task covering the Argo-side path.

## Fix

Publish the mode-selected overlay *at the fixed path the ApplicationSet
already expects*, rather than teaching the out-of-repo ApplicationSet about
modes. In the `Force-push bot/pr-<N>-resolved` step:

```sh
if [ "$MODE" = "sparse" ]; then
  rm -rf deploy/k8s/overlays/pr
  mv deploy/k8s/overlays/pr-sparse deploy/k8s/overlays/pr
fi
```

- The bot branch is generated and disposable, so relocating a directory on it
  costs nothing and keeps the mode decision entirely inside this workflow.
- `pr-sparse/` and `pr/` sit at the same depth, so the shared `../../base`
  resource reference resolves identically after the move.
- The move must run **after** `gen-cleanup-env.sh`, which reads
  `deploy/k8s/overlays/pr/scripts/` — a directory sparse mode does not carry.
  All other `overlays/pr/scripts` references are source-branch CI/dev
  generators, never read from the bot branch at sync time.
- Staging becomes `git add -A deploy/k8s/overlays` so the removal of
  `pr-sparse/` is staged alongside the rewritten `pr/`.
- A second leak guard re-checks the published path after the move. That is
  the check that would have caught this.

### Alternative considered and rejected

Making the ApplicationSet path mode-aware. Argo's PR generator exposes only
labels, branch, and number — not the mode, which `tools/mode-select.sh`
derives from changed files. CI would have to round-trip the mode back as a PR
label, plus a coordinated cluster-infra change. More moving parts for the
same result.

## Verification

Replayed the workflow's sparse resolve against a copy of the tree and rendered
`deploy/k8s/overlays/pr` with kustomize — i.e. exactly what Argo reads.

| | Namespace | Placeholders | Deployments |
|---|---|---|---|
| before | `atlas-pr-PLACEHOLDER_PR_NUMBER` | present | 67 (full set) |
| after | `atlas-pr-1411` | none | 3 (`atlas-channel`, `atlas-ingress`, `atlas-login`) |

The "after" row is the sparse override set plus `atlas-ingress`, which matches
the mode report comment on PR #1411.

Flagless `tools/verify.sh` exits 0. Note that it selects almost no checks for
this change — the diff is workflow-only, so the `pr-sparse mirror drift` and
`mode select decision table` guards both report "unchanged" and skip. The
kustomize render above is the real evidence, not the gate.

## Reconciling PR #1411

The stuck app recovers on its own once a corrected `bot/pr-1411-resolved` is
pushed; no manual namespace surgery is needed.

1. Merge this fix to `main`.
2. Re-run PR #1411's validation. `pull_request` workflows execute from the
   merge of head and base, so #1411 picks up the corrected step without
   needing a rebase.
3. `update-pr-overlay` force-pushes a `bot/pr-1411-resolved` whose `pr/` is
   the resolved sparse overlay; Argo's retry loop then applies it.

If the operation is still wedged, terminate it so the next auto-sync starts
clean rather than resuming the failed one.
