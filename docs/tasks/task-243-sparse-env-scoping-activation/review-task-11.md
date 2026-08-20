# Review — Task 11: pr-sparse overlay SERVICE_ID/precreate-groups placeholder tokens

Commit range: `100aaae68..e014e2f03` (single commit `e014e2f03`)
Brief: `.superpowers/sdd/plan/task-11-brief.md`
Report: `.superpowers/sdd/plan/task-11-report.md`

## Scope

Reviewed the diff of `e014e2f03` (21 lines inserted into
`deploy/k8s/overlays/pr-sparse/kustomization.yaml`) plus the sed substitution
pass in `.github/workflows/pr-validation.yml` (~lines 939-1043) that Task 12
will extend, since correctness of this task's placement depends on that
mechanism's contract. `scope_confirmed`: diff matches the brief (two
`#PLACEHOLDER_` tokens added to the `patches:` list, nothing else touched).

## Findings

### BLOCKING: self-referential prose comments collide with the unanchored sed substitution and will corrupt the file / break YAML once Task 12 fills them

`deploy/k8s/overlays/pr-sparse/kustomization.yaml:257` and `:268` place an
explanatory comment block **directly above** each token, and that comment's
first line repeats the token name verbatim:

```
257:  # PLACEHOLDER_SERVICE_ID_BLOCK — one inline strategic-merge patch per
...
267:  #PLACEHOLDER_SERVICE_ID_BLOCK
268:  # PLACEHOLDER_PRECREATE_GROUPS_BLOCK — a JSON-6902 patch adding a
...
277:  #PLACEHOLDER_PRECREATE_GROUPS_BLOCK
```

The existing CI substitution pass (`.github/workflows/pr-validation.yml:1024-1029`)
uses a plain, **unanchored** `sed` substring match with the `g` flag against
every `*.yaml`/`*.yml` file under `$OVERLAY_DIR`:

```
sed -i -e "s${D}PLACEHOLDER_DELETE_BLOCK${D}${DELETE_BLOCK}${D}g" ...
```

There is no `^`/`$` anchor and no requirement that the match be the sole
content of the line. Task 12 (per this task's own brief, §"Filled by
`.github/workflows/pr-validation.yml`'s `update-pr-overlay` job") will add an
analogous `-e "s${D}PLACEHOLDER_SERVICE_ID_BLOCK${D}${SERVICE_ID_BLOCK}${D}g"`
line to that same `sed` invocation. Because line 257 also contains the
literal substring `PLACEHOLDER_SERVICE_ID_BLOCK`, that substitution will
**also** fire on the descriptive comment line, not just on the intended
token line 267.

I reproduced this directly (not merely inferred it) by copying the real file
into scratch and running a `sed` invocation shaped exactly like the existing
`DELETE_BLOCK` call, with a synthetic multi-line strategic-merge-patch
payload built the same way `DELETE_BLOCK` itself is built (`printf` with
literal `\n` escapes that `sed`'s replacement text turns into real
newlines — the same technique `pr-validation.yml:1006` already uses and that
the brief says `PLACEHOLDER_SERVICE_ID_BLOCK`'s real payload will need, being
"one inline strategic-merge patch per override-set Deployment"). Result:

```
  # Empty, as checked in here (a lone comment — ignored by the YAML
  # parser, so this file still parses raw), this overlay builds every
  # base Deployment, same as overlays/pr — the smoke-test shape exercised
  # by Task 44 Step 4.
  # 
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: atlas-channel
      spec:
        template:
          spec:
            containers:
              - name: atlas-channel
                env:
                  - name: SERVICE_ID
                    value: abc123 — one inline strategic-merge patch per
  # override-set Deployment that carries a SERVICE_ID in base, setting that
  ...
```

The synthetic patch's tail (`value: abc123`) is spliced onto the leftover
tail of the original comment sentence (`— one inline strategic-merge patch
per`), producing an invalid YAML scalar, and the injected `- patch: |-` /
`apiVersion:` / … lines are emitted **without** a leading `#` — they become
live YAML list items rather than a commented block, injected at the wrong
list position (above the real token site, still 10 lines further down). This
is not a cosmetic corruption; it is a shape that `kustomize build` is likely
to reject or, worse, silently accept as a malformed/duplicated patch entry.

This is a materially different situation from the pre-existing precedent the
implementer's report cites confidence from
(`grep -rl` isolation check, `#PLACEHOLDER_DELETE_BLOCK` indentation match).
The one existing case of a token name appearing in prose within a swept file —
`PLACEHOLDER_OVERRIDES_JSON` in `deploy/k8s/overlays/pr-sparse/environment-record.yaml:23`
— is safe only because `OVERRIDES_JSON`'s payload is a single-line JSON
object with no embedded literal newlines; splicing it into a comment line
still yields one syntactically valid `#`-prefixed line. `DELETE_BLOCK` and
`NS_OVERRIDES`, the two multi-line-payload tokens, avoid this collision
entirely because their explanatory prose lives in `pr-validation.yml` itself
(outside `$OVERLAY_DIR`, never swept) — not in the target YAML file. Task 11
is the first place a *multi-line-payload* token's explanatory comment was
put in the swept file, and it reproduces the token name in that comment,
which the sed-based DELETE_BLOCK precedent never had to survive.

This is squarely within the review's flagged risk ("a token that ... sed
cannot address as written, is a blocking defect") and it will not surface
until Task 12 attempts to fill these tokens, at which point it manifests as
either a `kustomize build` failure or a corrupted, ambiguously-placed patch
in sparse-mode PRs — i.e. exactly the failure mode Task 12's own brief
(unread here, out of scope) would need to design around or this task would
need to avoid by construction.

**Fix options** (not adjudicating which — flagging for Task 12 or a Task 11
follow-up, not implementing): reword the explanatory comment to not repeat
the bare token substring (e.g. break it up, or refer to it as "the
`SERVICE_ID_BLOCK` token" without the `PLACEHOLDER_` prefix that the sed
pattern matches on), or move the explanation to `pr-validation.yml`/README.md
matching the `DELETE_BLOCK` precedent, or have Task 12 anchor its `sed`
patterns to lines containing *only* the token (e.g. `^  #PLACEHOLDER_X$`).

## What passes

- **Token wording and placement** (`deploy/k8s/overlays/pr-sparse/kustomization.yaml:257-277`) — verbatim match to the brief's specified text, placed immediately before `#PLACEHOLDER_DELETE_BLOCK` (line 278) as instructed. Confirmed by direct read.
- **Indentation/convention** — both token lines use 2-space indent, no leading `- `, matching `#PLACEHOLDER_DELETE_BLOCK`'s exact styling (verified with `cat -A`: `  #PLACEHOLDER_SERVICE_ID_BLOCK$`, `  #PLACEHOLDER_PRECREATE_GROUPS_BLOCK$`, `  #PLACEHOLDER_DELETE_BLOCK$`).
- **Isolation from `overlays/pr`** — `grep -rn 'PLACEHOLDER_SERVICE_ID_BLOCK\|PLACEHOLDER_PRECREATE_GROUPS_BLOCK' deploy/k8s/overlays/` returns exactly one file, `deploy/k8s/overlays/pr-sparse/kustomization.yaml`. Confirmed directly, matching the report's claim.
- **Both overlays still build unfilled**: `kustomize build deploy/k8s/overlays/pr-sparse` and `kustomize build deploy/k8s/overlays/pr` both exit 0 (only the pre-existing `commonLabels` deprecation warning, unrelated to this change). Confirmed directly.
- **Guard suite**: `tools/overlay-env-guard.sh`, `tools/sparse-baseline-scoping-guard.sh`, `tools/pr-sparse-mirror-guard.sh` all ran clean (all `PASS`/`SKIP`, no `FAIL`), matching the report's claimed output. Confirmed directly.
- **Diff shape matches the report**: 21 insertions, 0 deletions, single file — no incidental edits, no touching of `deploy/k8s/base/atlas-channel.yaml` or any out-of-scope file.
- **Scope discipline**: no edits to `services/atlas-pr-bootstrap/scripts/bootstrap.sh`, the `servicesuniq` migration test, or `docs/runbooks/sparse-environments.md` — correctly left to the concurrent implementers noted in the task brief.

## Not evaluable

- Whether Task 12's actual `sed` construction avoids the collision described above cannot be evaluated here — Task 12 is not yet implemented in this worktree (`.superpowers/sdd/plan/task-12-brief.md` exists but no corresponding commit). The finding above is a forward-looking defect in the placement this task chose, verified by reproducing the mechanism Task 12 is committed (by this task's own brief) to use, not a defect in code that exists yet.

## Verdict rationale

The diff, taken purely as "did two comment tokens land in the right file at
the right place, matching the brief's wording, without breaking today's
unfilled build" — is correct and verified by direct reproduction of every
check the report claims. But the task's own review brief explicitly asks
whether these tokens will be *reachable* by the sed pass Task 12 extends, and
direct reproduction shows the surrounding prose (which the brief itself
specified verbatim) will not survive that sed pass intact — it will corrupt
adjacent YAML. That is a blocking defect in the unit as delivered, even
though it was mandated by the brief's exact wording; the implementer
followed the brief faithfully but did not catch (nor did the report's
self-review catch) that the brief's own comment wording sets up a
foreseeable collision with the substitution mechanism explicitly named in
that same comment block.
