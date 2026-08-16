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
