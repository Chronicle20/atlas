# Git Workflow

This document owns the mechanics of branch safety, pushing after history
rewrites, what triggers a PR build, and `gh` authentication in this repo.

## Branch safety

Never commit or push directly to `main`. Branch protection blocks the push,
so a commit made on local `main` is stranded and never reaches the remote.
Check the branch before every `git commit`.

Setup work that must precede a feature branch still goes *on* the feature
branch — create it first; it branches from the same HEAD.

Recovery from a stray `main` commit: preserve the content on a branch
(cherry-pick if needed), then:

```sh
git fetch origin main && git reset --hard origin/main
```

## Pushing and history rewrites

After completing a rebase/merge/history-rewrite, always push (force-push
when history was rewritten) so the PR reflects the resolved state. Do not
stop at local-only completion — a rebase resolved only locally leaves the PR
still showing conflicts.

## PR body: the live smoke test

Every PR body opened from this repo ends with a **Live smoke test** section —
the steps a human runs against this PR's own ephemeral environment to watch
the change actually work. `superpowers:finishing-a-development-branch`
Step 5 Option 2 defers to "the repo's PR template and conventions"; this
section is that convention.

The steps are specific to the change and nothing else. Do not write a
generic "env came up / UI loaded / a client logged in" checklist — the
nightly `pr-env-smoke.yml` run already proves that, and repeating it hides
the one thing a reader needs. If a change is genuinely unobservable at
runtime (docs, CI config, a pure refactor with no behavior delta), write
`Live smoke test: n/a — <reason>` and stop.

Rules for the steps:

- **Derive them from the diff.** Every step traces to a hunk in this branch.
  Never invent an endpoint, a screen, an opcode, or an NPC that the change
  does not touch.
- **Each step stands alone and is literal.** The PR number's host
  (`<N>.atlas.home`), the actual client port for the version under test
  (login = `majorVersion × 100`, channel = `+1` —
  `services/atlas-pr-bootstrap/scripts/version-ports.sh`), the exact request,
  the exact in-game action. Never "navigate to the relevant page" or "test
  the new behavior."
- **Every step names what proves it passed** — the response field, the log
  line, the on-screen result. A step with no expected observation is not a
  test.
- **2–5 steps.** Cover the change's happy path plus whatever it regressed
  risk onto; leave the exhaustive matrix to the test suite.
- **Step 0 is the label** whenever the PR is not yet labeled: the
  environment only exists while the PR carries `deploy-env`.
- **Call out sparse mode** when it matters. In sparse mode every unchanged
  service is served by `main` (see §9.15 of the ephemeral-PR runbook). If a
  step depends on a service outside the changed set, say so, or say the PR
  needs `gh pr edit <N> --add-label atlas:isolated`.

Shape:

```markdown
## Live smoke test

0. Add the `deploy-env` label; wait for the `atlas-pr-<N>` rollout.
1. `curl -H 'ENVIRONMENT: pr-<N>' https://<N>.atlas.home/api/…` →
   expect `attributes.<field> == <value>` (was absent before this branch).
2. Connect a v83 client to `<N>.atlas.home:8300` (channel `8301`), …
   → expect <specific observable result>.
```

## Build triggering and the conflict exception

A plain push to a task branch DOES trigger the PR workflows and the
`atlas-pr-<N>` ephemeral rollout. Do not merge `origin/main` as a routine
build-triggering ritual.

The one exception: when the branch conflicts with `main`, the push does not
start the build — merge `origin/main`, resolve, push the merge commit. The
merge is the conflict resolution, not the trigger.

See [`docs/runbooks/ephemeral-pr-deployments.md`](runbooks/ephemeral-pr-deployments.md)
for the full `atlas-pr-<N>` environment lifecycle.

## `gh` authentication

Run `gh` with the token env explicitly cleared so it uses the stored
`hosts.yml` account:

```sh
env -u GH_TOKEN -u GITHUB_TOKEN gh …
```

Do NOT source `~/.config/atlas/gh.env` — its `GH_TOKEN` is expired and takes
precedence, causing 401s. Never echo the token.
