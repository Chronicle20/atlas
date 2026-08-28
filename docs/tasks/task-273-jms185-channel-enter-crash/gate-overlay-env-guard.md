# overlay-env-guard failure triage

## Verdict: NOT REPRODUCIBLE — treat as a transient artifact of the original timed-out `--quick` run, not a defect on this branch

The `overlay-env-guard: FAIL - overlays/pr atlas-ingress-routes-* ConfigMap is
missing the env-default.conf.template key` reported by the earlier
`tools/verify.sh --quick --base 310e5a786` run does **not** reproduce when the
guard is run standalone, on either a clean `origin/main` checkout or the task
worktree itself. Both runs below are clean full passes (`EXIT=0`), with only
the expected `SKIP` line for `overlays/pr-sparse`'s by-design env-wiping
`ns-overrides.yaml` patch (documented in the guard script's own header).

Given that the caller's original run was independently confirmed to have
timed out, the most plausible explanation is that the guard's
`kustomize build overlays/pr` render was truncated mid-write by the timeout
(the guard renders to a `mktemp -d` scratch file with no atomicity guarantee
against an external kill), producing a partial `pr.yaml` that happened to be
cut before the `atlas-ingress-routes-*` ConfigMap's
`env-default.conf.template:` key — hence the single spurious FAIL. This is
consistent with the run being explicitly called out as "a partial run either
way." No commit on this branch, and no file under `deploy/`, was touched to
produce or fix this: `git diff --stat 310e5a786 HEAD -- deploy/` remains
empty (previously established).

## What the guard requires (tools/overlay-env-guard.sh)

Renders `overlays/main`, `overlays/pr`, `overlays/pr-sparse` once each via
`kustomize build` into a scratch dir, then asserts against the rendered YAML
with `awk`/`grep -F`. The specific assertion that failed in the original run
(assertions 11 for `overlays/pr`) extracts the `atlas-ingress-routes-*`
ConfigMap document and requires it to contain the literal line
`env-default.conf.template:`. There is no special-casing for `overlays/pr` on
this assertion (unlike the `pr-sparse` `ns-overrides.yaml` carve-out for
assertions 9-10) — a correctly-rendered `overlays/pr` must always carry the
key.

## Evidence: origin/main (7f1f6dc09, throwaway worktree at /tmp/atlas-main-check)

```
git rev-parse origin/main → 7f1f6dc0915247d4c8bd66ce99447abe8f6904f0
bash tools/overlay-env-guard.sh; echo "EXIT=$?"

overlay-env-guard: PASS - overlays/main atlas-env ConfigMap has ATLAS_ENVIRONMENT: main
overlay-env-guard: PASS - overlays/main ATLAS_ENVIRONMENT (main) matches namespace (atlas-main) with the atlas- prefix stripped
overlay-env-guard: PASS - overlays/main atlas-environment-record Job carries sync-wave 11
overlay-env-guard: PASS - overlays/main atlas-environment-record Job's script targets atlas-configurations directly, bypassing atlas-ingress
overlay-env-guard: PASS - overlays/main atlas-environment-record Job's POST body has phase ACTIVE and name main
overlay-env-guard: PASS - overlays/pr atlas-env ConfigMap has ATLAS_ENVIRONMENT present with an empty value
overlay-env-guard: PASS - overlays/pr renders no atlas-environment-record Job
overlay-env-guard: PASS - overlays/pr-sparse atlas-env ConfigMap has ATLAS_ENVIRONMENT: pr-PLACEHOLDER_PR_NUMBER
overlay-env-guard: PASS - overlays/main atlas-ingress carries ATLAS_ENVIRONMENT_DEFAULT sourced from atlas-env/ATLAS_ENVIRONMENT
overlay-env-guard: PASS - overlays/main atlas-ingress NGINX_ENVSUBST_FILTER includes ATLAS_ENVIRONMENT_DEFAULT
overlay-env-guard: PASS - overlays/pr atlas-ingress carries ATLAS_ENVIRONMENT_DEFAULT sourced from atlas-env/ATLAS_ENVIRONMENT
overlay-env-guard: PASS - overlays/pr atlas-ingress NGINX_ENVSUBST_FILTER includes ATLAS_ENVIRONMENT_DEFAULT
overlay-env-guard: SKIP - overlays/pr-sparse atlas-ingress ATLAS_ENVIRONMENT_DEFAULT/NGINX_ENVSUBST_FILTER (ns-overrides.yaml:38-46's env: #PLACEHOLDER_NS_OVERRIDES YAML-parses to env: null, wiping the container's env list until .github/workflows/pr-validation.yml:1027 substitutes it at PR-apply time; by design, not this guard's concern)
overlay-env-guard: PASS - overlays/main atlas-ingress-routes-* ConfigMap carries env-default.conf.template
overlay-env-guard: PASS - overlays/main atlas-ingress-configmap nginx.conf resolves ENVIRONMENT via $atlas_environment
overlay-env-guard: PASS - overlays/pr atlas-ingress-routes-* ConfigMap carries env-default.conf.template
overlay-env-guard: PASS - overlays/pr atlas-ingress-configmap nginx.conf resolves ENVIRONMENT via $atlas_environment
overlay-env-guard: PASS - overlays/pr-sparse atlas-ingress-routes-* ConfigMap carries env-default.conf.template
overlay-env-guard: PASS - overlays/pr-sparse atlas-ingress-configmap nginx.conf resolves ENVIRONMENT via $atlas_environment
EXIT=0
```

(The throwaway worktree was created via `git worktree add /tmp/atlas-main-check origin/main` and removed via `git worktree remove /tmp/atlas-main-check --force` after the run; it never touched `.worktrees/`.)

## Evidence: task worktree (70efc0dad, task-273-jms185-channel-enter-crash)

```
git rev-parse HEAD → 70efc0dad2b342e484f4f8abd58b783810b3c638
bash tools/overlay-env-guard.sh; echo "EXIT=$?"

overlay-env-guard: PASS - overlays/main atlas-env ConfigMap has ATLAS_ENVIRONMENT: main
overlay-env-guard: PASS - overlays/main ATLAS_ENVIRONMENT (main) matches namespace (atlas-main) with the atlas- prefix stripped
overlay-env-guard: PASS - overlays/main atlas-environment-record Job carries sync-wave 11
overlay-env-guard: PASS - overlays/main atlas-environment-record Job's script targets atlas-configurations directly, bypassing atlas-ingress
overlay-env-guard: PASS - overlays/main atlas-environment-record Job's POST body has phase ACTIVE and name main
overlay-env-guard: PASS - overlays/pr atlas-env ConfigMap has ATLAS_ENVIRONMENT present with an empty value
overlay-env-guard: PASS - overlays/pr renders no atlas-environment-record Job
overlay-env-guard: PASS - overlays/pr-sparse atlas-env ConfigMap has ATLAS_ENVIRONMENT: pr-PLACEHOLDER_PR_NUMBER
overlay-env-guard: PASS - overlays/main atlas-ingress carries ATLAS_ENVIRONMENT_DEFAULT sourced from atlas-env/ATLAS_ENVIRONMENT
overlay-env-guard: PASS - overlays/main atlas-ingress NGINX_ENVSUBST_FILTER includes ATLAS_ENVIRONMENT_DEFAULT
overlay-env-guard: PASS - overlays/pr atlas-ingress carries ATLAS_ENVIRONMENT_DEFAULT sourced from atlas-env/ATLAS_ENVIRONMENT
overlay-env-guard: PASS - overlays/pr atlas-ingress NGINX_ENVSUBST_FILTER includes ATLAS_ENVIRONMENT_DEFAULT
overlay-env-guard: SKIP - overlays/pr-sparse atlas-ingress ATLAS_ENVIRONMENT_DEFAULT/NGINX_ENVSUBST_FILTER (ns-overrides.yaml:38-46's env: #PLACEHOLDER_NS_OVERRIDES YAML-parses to env: null, wiping the container's env list until .github/workflows/pr-validation.yml:1027 substitutes it at PR-apply time; by design, not this guard's concern)
overlay-env-guard: PASS - overlays/main atlas-ingress-routes-* ConfigMap carries env-default.conf.template
overlay-env-guard: PASS - overlays/main atlas-ingress-configmap nginx.conf resolves ENVIRONMENT via $atlas_environment
overlay-env-guard: PASS - overlays/pr atlas-ingress-routes-* ConfigMap carries env-default.conf.template
overlay-env-guard: PASS - overlays/pr atlas-ingress-configmap nginx.conf resolves ENVIRONMENT via $atlas_environment
overlay-env-guard: PASS - overlays/pr-sparse atlas-ingress-routes-* ConfigMap carries env-default.conf.template
overlay-env-guard: PASS - overlays/pr-sparse atlas-ingress-configmap nginx.conf resolves ENVIRONMENT via $atlas_environment
EXIT=0
```

## Conclusion

Both the clean `origin/main` (7f1f6dc09) and the task worktree (70efc0dad)
pass the guard cleanly and identically. The original FAIL does not reproduce
in either state, so it cannot be attributed to this branch's merge — there is
no commit or overlay-file diff to point to (confirmed empty `deploy/` diff).
Most likely cause: the original `--quick` invocation was killed by its own
timeout mid-`kustomize build`, corrupting the scratch render before the
assertion ran. No fix was made or is warranted here; re-run
`tools/overlay-env-guard.sh` (or full `tools/verify.sh`) without truncation
to confirm on a subsequent attempt.
