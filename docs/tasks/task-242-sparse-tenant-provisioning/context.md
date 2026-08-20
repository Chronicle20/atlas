# task-242 — Implementation Context

Companion to `plan.md`. Everything an implementer or reviewer needs that is
not a step: the file map, the decisions the plan inherited, the two it
amended, and the traps that are easy to walk into.

---

## Task → files → surface

| Task | Surface | Files | Service |
|---|---|---|---|
| 1 | Extract `env-record.sh`; cleanup.sh delegates | 5 | atlas-pr-bootstrap |
| 2 | Environment-scoped tenant lookup/create | 2 | atlas-pr-bootstrap |
| 3 | PATCH the tenant onto the environment record | 2 | atlas-pr-bootstrap |
| 4 | `ATLAS_ENVIRONMENT` literals + main's env record Job | 4 | deploy/k8s |
| 5 | Ingress `ENVIRONMENT` default (nginx `map`) | 3 | deploy/k8s |
| 6 | `tools/overlay-env-guard.sh` + verify.sh wiring | 2 | tools |
| 7 | Runbook section | 1 | docs |
| 8 | Flagless `tools/verify.sh` | 0 | — |

No task exceeds 6 files, and none crosses a service boundary. Tasks 1→2→3
are strictly sequential (each depends on the previous one's helpers); 4→5
are sequential (5's `configMapKeyRef` needs 4's ConfigMap key); 6 depends on
4 and 5; 7 and 8 come last. No task was left deliberately large.

---

## Key files, and what each one actually is

**`libs/atlas-env/env.go:29`** — `SelfVar = "ATLAS_ENVIRONMENT"`. This is the
variable, and it is a *different* variable from `ATLAS_ENV` (the env hash
used for consumer groups and pod labels, set by
`deploy/k8s/overlays/main/patches/atlas-env-env.yaml`). Confusing the two is
the single easiest way to break this task.

**`libs/atlas-env/registry.go:216-247`** — `IsOwner` ends
`return rec.Baseline == r.self`; `EnvironmentsOwnedBy` returns the legacy
`[""]` **only** while `len(r.records) == 0`. Together those two lines are
why an empty `self` on main's pods is not merely a sparse-mode bug: the
moment any sparse PR registers a record, main's own periodic loops iterate
an empty slice. Task 4 fixes both halves at once — the literal *and* the
`main` record — because either alone is insufficient.

**`deploy/k8s/base/env-configmap.yaml`** — plain ConfigMap resource, not
generated. `overlays/main` and `overlays/pr` both replace it wholesale
(`behavior: replace`); `overlays/pr-sparse` merges into it
(`behavior: merge`). That asymmetry is why Task 4 touches three files to add
one key.

**`deploy/k8s/base/atlas-ingress.yaml`** — `nginx.conf` lives in the plain
`atlas-ingress-configmap` and is mounted by `subPath`, so it is **not**
envsubst'd and editing it does **not** roll the pods.
`routes.conf.template` lives in the hashed `atlas-ingress-routes` generator
and **is** envsubst'd. Task 5's new template joins the second one, so the
generator hash changes and the rollout does happen — but the `nginx.conf`
half still needs the explicit restart noted in the runbook.

**`services/atlas-pr-bootstrap/scripts/bootstrap.sh`** — runs its whole flow
at top level and is **not sourceable**. Its bats suites extract helpers with
`sed -n '/^name()/,/^}/p'`. Every helper this plan adds is therefore at
column 0, and `ENV_HEADER` is populated by an extractable `env_header_init`
function rather than by a bare top-level assignment, precisely so the
ATLAS_MODE gate is testable.

**`services/atlas-pr-bootstrap/test/cleanup_test.bats`** — runs `cleanup.sh`
as a whole script through `bash`, with stubs on `PATH`. That is why Task 1's
extraction is safe and why the suite passing *unmodified* is the acceptance
test for it.

---

## Decisions inherited from design.md

- **D1** — main's environment record is created by a deploy-time Argo Job at
  sync-wave 11, POSTing to the ClusterIP (never the ingress, which would
  400 its own creating request once the default lands). The bring-up window
  where the ingress stamps `main` before the record exists is accepted;
  §9's alternative (seed it from `atlas-configurations`' migration) stays in
  reserve and is a one-file swap if the window proves unacceptable.
- **D2** — one ConfigMap literal reaches all 64 deployments, because every
  base Deployment consumes `atlas-env` via `envFrom`.
- **D3** — the ingress default is an envsubst'd nginx `map` sourced from the
  same `ATLAS_ENVIRONMENT` key `env.Self()` reads, so the two cannot drift.
  Zero per-overlay patches; nothing mirrored by
  `tools/pr-sparse-mirror-guard.sh` is touched.
- **D4** — bootstrap gates on `ATLAS_MODE`, not on a live record check,
  because it is *establishing* state and must send the header on its first
  call. `ENV_HEADER` is an array, not a string, because a string
  word-splits on the space in `ENVIRONMENT: pr-1411`.
- **D5** — the PATCH carries all five attributes **and** the record's
  *current* phase. A phase-less body 400s (`validatePhase` runs before the
  backfill); a same-phase transition is legal, which is what makes it
  idempotent. It runs *before* the tenant-config clone so a mid-run failure
  still leaves a reclaimable record.
- **D7** — sparse environments add **no** `atlas-data` rows. The PRD's
  "~48k documents per environment" premise is wrong post-`c5e88320a`, and
  two of its acceptance criteria were replaced accordingly.

---

## Two amendments this plan makes to design.md

### A. `ATLAS_ENVIRONMENT_DEFAULT` must be defined, never `optional`

D3 specifies `configMapKeyRef … optional: true`. That is unsafe. The nginx
image's `docker-entrypoint.d/20-envsubst-on-templates.sh` builds its
substitution list from variables actually present in the process
environment and passes it to `envsubst` as a SHELL-FORMAT argument;
`envsubst` leaves any variable outside that list **verbatim**. So an
undefined `ATLAS_ENVIRONMENT_DEFAULT` renders the map as

```nginx
map $http_environment $atlas_environment {
    ""      "${ATLAS_ENVIRONMENT_DEFAULT}";
    default $http_environment;
}
```

and every untagged request gets stamped with that literal string, which
`ParseEnvironment` (`libs/atlas-rest/server/handler.go:68`) 400s. Total,
silent outage — in exactly the overlays D3 wanted to leave untouched.

The plan guarantees definedness instead of tolerating absence:
`ATLAS_ENVIRONMENT: ""` in `deploy/k8s/base/env-configmap.yaml` and
`ATLAS_ENVIRONMENT=` in `deploy/k8s/overlays/pr/kustomization.yaml` (needed
separately because that generator is `behavior: replace`), and no
`optional: true` — so a future overlay that drops the key fails loudly at
pod start rather than 400ing every request.

`ATLAS_ENVIRONMENT=""` is indistinguishable from unset for `env.Self()`
(`os.Getenv` returns `""` either way), so FR-1.5 still holds. The manifest
assertion changes from "`overlays/pr` renders no `ATLAS_ENVIRONMENT`" to
"renders it with an empty value" — same semantics, stated accurately.

### B. D6's Go regression pin already exists

Design §7 asks for a new test in
`services/atlas-configurations/atlas.com/configurations/templates/overlay_test.go`
proving a non-baseline environment's template lookup still resolves the
baseline's row. `TestTemplatesFallBackToTheBaselineRow` (line 65) already
does exactly that — seeds only a `main` v83.1 row, reads as `pr-123`,
asserts the baseline row comes back. That is the call `bootstrap.sh:290`
makes and the assertion a `scope.Strict` refactor would break. A second copy
would be duplicate coverage, so the plan adds no Go code and cites the
existing pin in the runbook instead.

---

## Traps

- **`ATLAS_ENV` ≠ `ATLAS_ENVIRONMENT`.** See above.
- **`overlays/pr-sparse` must never contain the literal `main`.** It names
  the baseline only via `PLACEHOLDER_BASELINE_ENVIRONMENT` /
  `PLACEHOLDER_BASELINE_NAMESPACE`. No task in this plan edits `pr-sparse`.
- **Nine mirrored files.** `tools/pr-sparse-mirror-guard.sh:31-41` byte-diffs
  `overlays/pr` against `overlays/pr-sparse` for nine paths, including
  `patches/ingress-host.yaml` — which the PRD flagged as the *wrong* home
  for the ingress default. D3's design avoids all nine; keep it that way.
  `kustomization.yaml` is **not** mirrored, which is why Task 4 may edit
  `overlays/pr/kustomization.yaml`.
- **Bats suites go green for the wrong reason without a `declare -F` guard.**
  A failed sed extraction makes `run` exit 127, which satisfies every "must
  fail" assertion. `data_ingest_test.bats:25-34` is the pattern; every new
  suite must copy it.
- **`set -u` and empty arrays.** `"${ENV_HEADER[@]}"` on an empty array is an
  error under `set -u` in bash < 4.4. The image is `alpine:3.24` (bash 5.x)
  and CI hosts are Linux, so this is safe — but Task 2 pins it with an
  explicit isolated-mode case rather than assuming it.
- **A phase-less environments PATCH is a 400**, not a partial update.
- **`gen-routes.sh` owns `routes.conf.template.generated`.** Never
  hand-edit it, and never put `proxy_set_header ENVIRONMENT` inside it — a
  second same-level directive would accumulate rather than override.

---

## Verification

- `bats services/atlas-pr-bootstrap/test` — runs from `tools/verify.sh:521`
  when the service changes; `cleanup_test.bats` must pass unmodified.
- `./tools/overlay-env-guard.sh` — new in Task 6; wired into `verify.sh`'s
  `deploy/` block.
- `./tools/pr-sparse-mirror-guard.sh`, `./tools/gen-routes.sh --check` — both
  already run from `verify.sh` for this branch's change set.
- Flagless `tools/verify.sh` is the gate. `--quick` / `--no-docker` skip the
  bake and `-race` and do not count.

Live acceptance is a real sparse-environment round trip; the checklist is at
the end of `plan.md`. The defect class this task addresses has escaped four
times on PR #1411 (#1416, #1418, #1421, and this one), so script-level
coverage alone is explicitly not sufficient evidence.
