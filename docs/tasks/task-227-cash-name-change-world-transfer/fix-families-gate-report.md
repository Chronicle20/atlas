# Step 2 — atlas-families deployment wiring + `inFamily` fix

Scope: step 2 only of the "Suggested fix order" in
`bug-world-transfer-eligibility-reasons.md` ("Families → wire up the
deployment" + `inFamily`'s two logic bugs, ruling 3). Steps 1, 3, 4, 5 were
explicitly out of scope and not touched.

## Half A — atlas-families deployment

Verified absent, as the brief stated: no `deploy/k8s/base/atlas-families.yaml`,
no `/api/families` route, and (separately discovered) `atlas-families` was
sitting in `tools/service-registration-guard.sh`'s `ALLOW_NO_DEPLOYMENT`
allowlist with the comment "scaffolded service, not yet deployed".

Already wired (verified before touching, not re-done):
- `.github/config/services.json`, `docker-bake.hcl`, `go.work`,
  `tools/db-bootstrap.sh` — all had the entry already (§1, §6.2 of
  `docs/adding-a-new-service.md`).
- `deploy/k8s/overlays/{main,pr}/kustomization.yaml` `images:` lists —
  already pinned (§3.3/4.2).
- `deploy/k8s/base/env-configmap.yaml` and both overlays'
  `configMapGenerator` literals for `COMMAND_TOPIC_FAMILY` /
  `EVENT_TOPIC_FAMILY_{STATUS,ERRORS,REPUTATION}` / (consumed)
  `EVENT_TOPIC_CHARACTER_STATUS` — already present, all three tiers in
  parity (§2.3/3.4/4.3).

Added:
- `deploy/k8s/base/atlas-families.yaml` — Deployment (2 replicas, container
  name `families`, `DB_NAME=atlas-families`, `DB_USER`/`DB_PASSWORD` from
  `db-credentials`, `envFrom: atlas-env`) + Service, modeled on
  `atlas-guilds.yaml` (nearest DB-backed sibling with the same
  envFrom+DB_NAME+secretKeyRef shape). `atlas-families` is DB-backed and runs
  Kafka consumers/a producer per its `main.go`
  (`services/atlas-families/atlas.com/family/main.go`), matching the
  guilds/fame class of service, not a stateless one.
- `deploy/k8s/base/kustomization.yaml` — `atlas-families.yaml` resource entry
  (alphabetical slot).
- `deploy/k8s/overlays/main/patches/db-name-suffix.yaml` and
  `patches/atlas-env-env.yaml` — new patch documents for
  `DB_NAME: "atlas-families-main"` and `ATLAS_ENV: "main"` (hand-maintained
  per §3, no generator for the main overlay).
- `deploy/k8s/overlays/pr/kustomization.yaml` — added `atlas-families` to the
  `ATLAS_DB_NAMES` literal (§4.1).
- `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml` and
  `patches/consumer-group-env.yaml` — regenerated via
  `deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh` and
  `gen-consumer-group-patch.sh` (both generator-owned, §4.4/4.5). The
  consumer-group literal picked up correctly from `main.go`'s
  `consumergroup.Resolve("Family Service")`.
- `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` —
  regenerated via `gen-cleanup-env.sh` after the `ATLAS_DB_NAMES` change
  (§4.6); `ATLAS_SERVICES` already included `atlas-families` from
  `services.json`.
- `deploy/shared/routes.conf` — new `/api/families` location block, proxying
  to `atlas-families:8080`, placed next to `/api/guilds` (§5.1).
- `deploy/k8s/base/routes.conf.template.generated` — regenerated via
  `tools/gen-routes.sh` (§5.2).
- `tools/service-registration-guard.sh` — removed the stale
  `'atlas-families'` entry from `ALLOW_NO_DEPLOYMENT` now that it has a base
  manifest.

**`versions.json` — not touched, and the brief/diagnosis's claim about it is
wrong.** `deploy/k8s/base/versions.json` is the client
game-version/LB-port declaration list
(`region`/`majorVersion`/`minorVersion`, `additionalProperties: false` per
`versions.schema.json`), consumed only by `tools/gen-lb-ports.sh` for the
login/channel LoadBalancers. It has no concept of a per-service entry at
all — there is nothing shaped like a "families row" that could exist there.
`git log` confirms it has only ever held client-version rows since task-084.
I did not add anything to it; adding an entry would have violated its own
schema. This is flagged as a correction to the diagnosis doc, not silently
skipped.

**Concurrent-worktree note:** this worktree had another implementer agent
(step 1, the storage-warning move) committing in parallel. My staged
`deploy/shared/routes.conf` +
`deploy/k8s/base/routes.conf.template.generated` changes were swept into
their commit `c642cda59` ("fix(task-227): move storage-stranding warning
from CHECK to BUY_WORLD_TRANSFER") rather than mine, because `git commit`
commits the whole index at commit time and the two agents share one working
tree. My own `feat(deploy):` commit similarly picked up one file
(`world_transfer_alias_test.go`) staged by a step-5 agent between my `git
add` and `git commit`. Content-wise everything is correct and nothing was
lost or reverted; only commit attribution is mixed. I did not attempt to
unwind this with `reset`/`rebase` per the git safety protocol — that risks
destroying concurrent in-flight work. Flagging for the controller's
awareness when reviewing per-commit diffs.

### Verification (Half A)

```
bash tools/service-registration-guard.sh
  -> service-registration-guard: clean

bash tools/gen-routes.sh && git add deploy/shared/routes.conf deploy/k8s/base/routes.conf.template.generated
bash deploy/shared/test/routes_nginxt.sh
  -> nginx: configuration file /etc/nginx/nginx.conf test is successful
  -> routes.conf MinIO upstream cross-namespace check: OK
  -> routes.conf atlas-renders tenant header check: OK
  -> routes.conf character-render hash-length check: OK
  -> routes drift check (shared vs k8s-generated): OK

kubectl kustomize deploy/k8s/overlays/main | grep -A40 "name: atlas-families$"
  -> DB_NAME: atlas-families-main, ATLAS_ENV: main,
     image: ghcr.io/chronicle20/atlas-families/atlas-families:main-543a88d (pinned, not :latest)

kubectl kustomize deploy/k8s/overlays/pr > /dev/null
  -> exit 0, renders clean; atlas-families Deployment carries
     KAFKA_CONSUMER_GROUP="Family Service [PLACEHOLDER_ATLAS_ENV]",
     ATLAS_ENV/DB_NAME placeholders, replicas: 1 (PR override)

services/atlas-families/atlas.com/family: go build ./... && go vet ./...
  -> clean (no Go code was touched in this module; confirms nothing broke)
```

Not done (out of my ability to produce, per the doc's own §6.1/§6b):
creating `atlas-families-main` on postgres.home, and flipping the GHCR
package to public after the first image push. Both are manual/out-of-repo
steps documented in `docs/adding-a-new-service.md` §6.1/§6b.

## Half B — `inFamily` fix

`services/atlas-character/atlas.com/character/pending_change/requests.go:113`
(`inFamily`), two bugs fixed:

1. Non-404 errors now propagate instead of being folded into `true`. Per the
   brief, no `check_unavailable` reason was invented here — that taxonomy is
   step 3's job; `processor_eligibility.go` was not touched.
2. Success now requires `len(members) > 1`. Confirmed the tree shape by
   reading `getFamilyTreeHandler`
   (`services/atlas-families/atlas.com/family/family/resource.go`): the
   route materializes self + senior + juniors + siblings in one array, so a
   character with no relatives still gets 200 with a one-element (self-only)
   array. This matches `familyMemberRestModel`'s own doc comment in
   `pending_change/rest.go` ("even a solo tree of just itself").

### Tests added

`TestInFamily` (table test) in
`services/atlas-character/atlas.com/character/pending_change/requests_test.go`,
covering:
- 404 → `false, nil`
- 500 (stand-in for a transport/decode failure) → error propagated, `false`
- 200 with a self-only tree (`["500500"]`) → `false, nil`
- 200 with a relative present (`["500500","500501"]`) → `true, nil`

### Verification (Half B)

```
cd services/atlas-character/atlas.com/character
go build ./... && go test ./pending_change/...
  -> ok  atlas-character/pending_change  137.142s
```

Output pristine (no warnings), all pending_change package tests pass
including the new `TestInFamily` subtests and the pre-existing gate-11 MTS
tests.

## Files changed

- `deploy/k8s/base/atlas-families.yaml` (new)
- `deploy/k8s/base/kustomization.yaml`
- `deploy/k8s/overlays/main/patches/atlas-env-env.yaml`
- `deploy/k8s/overlays/main/patches/db-name-suffix.yaml`
- `deploy/k8s/overlays/pr/kustomization.yaml`
- `deploy/k8s/overlays/pr/patches/consumer-group-env.yaml`
- `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml`
- `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`
- `tools/service-registration-guard.sh`
- `deploy/shared/routes.conf` (content correct, landed in commit
  `c642cda59` due to the shared-worktree race above)
- `deploy/k8s/base/routes.conf.template.generated` (same as above)
- `services/atlas-character/atlas.com/character/pending_change/requests.go`
- `services/atlas-character/atlas.com/character/pending_change/requests_test.go`

## Self-review

- Half A: every deliverable the brief listed is present and verified against
  the doc's own gates (`service-registration-guard.sh`,
  `routes_nginxt.sh`, `kubectl kustomize` render). The `versions.json` item
  was investigated and found to be a diagnosis error rather than a real gap;
  documented rather than silently dropped or blindly obeyed.
- Half B: predicate now honest per both stated bugs; tests are table-driven
  and hit the real HTTP boundary (httptest server), not mocks of the request
  layer, matching the existing `mtsMux` pattern in the same file.
- No `check_unavailable` reason invented; `processor_eligibility.go`
  untouched, as instructed.
- Did not touch steps 1/3/4/5 — no template JSON, no
  `cash_shop_check_transfer_world_possible*`, no `processor_eligibility.go`
  edits from this task.

## Concerns

- The commit-attribution mixing described above (shared worktree with a
  concurrent implementer) means `git log --follow` on
  `deploy/shared/routes.conf` won't show a `feat(deploy):`-titled commit for
  the families route — it's inside `c642cda59`. Content is correct; only
  worth the controller/reviewer knowing before assuming a route commit is
  missing.
