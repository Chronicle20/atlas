# Deployment & Wiring Patterns

This file documents two conventions that are easy to break when adding a new
shared library or a new Kafka topic. Both are enforced as audit checks by the
`backend-guidelines-reviewer` (DOM-22, DOM-23; defined in
[audit-checklist.md](audit-checklist.md)). Read once before adding a new lib or
a new topic.

---

## The container build is service-agnostic (DOM-22)

There is **no per-service Dockerfile** for Go services. The repo-root
`Dockerfile` builds every service in `.github/config/services.json`
(`.services[] | select(.type=="go-service")`), parameterized by `ARG SERVICE`:

```bash
docker buildx bake atlas-<svc>       # preferred
docker build -f Dockerfile --build-arg SERVICE=atlas-<svc> .
```

It copies **every** `libs/atlas-*` module and synthesizes a minimal `go.work`
listing all of them plus the target service's module. Adding a new `libs/*`
dependency to a service's `go.mod` therefore requires **no** Dockerfile change
— that was the old per-service-Dockerfile trap, and it is gone.

What still requires a manual edit is adding a **new shared library**:

| # | Where | What to add |
|---|---|---|
| 1 | Root `Dockerfile`, mod-only block | `COPY libs/atlas-X/go.mod libs/atlas-X/go.sum libs/atlas-X/` (omit `go.sum` if the lib has none — `atlas-retry` and `atlas-service` are the current examples) |
| 2 | Root `Dockerfile`, source block | `COPY libs/atlas-X libs/atlas-X` |
| 3 | Root `Dockerfile`, synthesized `go.work` loop | add `atlas-X` to the `for L in ...` list |
| 4 | Repo-root `go.work` | add `./libs/atlas-X` to `use()` |

### Why `go build ./...` is not enough

Local `go build ./...` runs against the repo-root `go.work`, which lists every
lib and every service. The image builds against the freshly-synthesized minimal
`go.work` inside the container. A new lib that local builds resolve happily
breaks the image build the moment the deploy runs — which is why the flagless
`tools/verify.sh` includes the `docker buildx bake` stage and `--no-docker` /
`--quick` do not.

### Audit verification — DOM-22

Triggers when the diff adds a module under `libs/`.

```bash
# For each libs/atlas-X added in the diff:
grep -c 'libs/atlas-X' Dockerfile      # expect >= 2 (mod-only COPY + source COPY)
grep -n 'atlas-X' Dockerfile           # confirm presence in the `for L in ...` go.work loop
grep -n './libs/atlas-X' go.work       # confirm the workspace use() entry
```

**Pass criteria:** the new lib appears in both `COPY` blocks, in the
synthesized `go.work` loop, and in the repo-root `go.work`. Missing any one of
them fails the image build even when local `go build ./...` succeeds. A diff
that only adds an *existing* lib to a service's `go.mod` is `N/A` for DOM-22.

---

## Kafka topic naming (DOM-23)

Every Kafka topic in this project follows a single rigid convention:

1. **The env-var name and the literal topic name are identical**, in
   SHOUTY_SNAKE_CASE prefixed with `COMMAND_TOPIC_` or `EVENT_TOPIC_`.

   Examples (from `deploy/k8s/base/env-configmap.yaml`):
   ```yaml
   COMMAND_TOPIC_DATA: "COMMAND_TOPIC_DATA"
   COMMAND_TOPIC_MONSTER: "COMMAND_TOPIC_MONSTER"
   EVENT_TOPIC_CHARACTER_STATUS: "EVENT_TOPIC_CHARACTER_STATUS"
   ```

   **Anti-patterns:** `command.monster`, `monster-events`, `topic_monster`,
   `monster.command.v1`. Anything dotted, lowercase, hyphenated, or versioned
   diverges from the convention and breaks configmap-driven topic discovery.

2. **All topics live in `deploy/k8s/base/env-configmap.yaml`** under the
   `# Command Topics` or `# Event Topics` sections, alphabetically ordered.

3. **Services consume topic names via `envFrom: configMapRef: atlas-env`**, NOT
   via hand-written `env: - name: COMMAND_TOPIC_X / value: ...` blocks in the
   service's deployment manifest. The configmap is the single source of truth;
   duplicating the value in a service manifest invites drift.

   The service's Go code reads `os.Getenv("COMMAND_TOPIC_X")` — typically via
   `topic.EnvProvider(l)("COMMAND_TOPIC_X")()` — and uses the result verbatim.

4. **Both overlays must re-list the key.** The main and pr overlays use
   `behavior: replace` on the `atlas-env` `configMapGenerator`, so a base key
   that an overlay does not re-list is simply **absent** in that environment.
   See `docs/adding-a-new-service.md` §3.4 and §4.3 — the pr overlay's literals
   are generator-owned (`deploy/k8s/overlays/pr/scripts/gen-topic-config.sh`),
   the main overlay's are hand-maintained.

### Checklist when adding a new topic

- [ ] Name is `COMMAND_TOPIC_<DOMAIN>` or `EVENT_TOPIC_<DOMAIN>_STATUS`.
- [ ] Entry added to `deploy/k8s/base/env-configmap.yaml` as `KEY: "KEY"`, in
      alphabetical order under the appropriate section.
- [ ] Main-overlay `configMapGenerator` literal added (`KEY=KEY-main`).
- [ ] PR-overlay literals **regenerated** with `gen-topic-config.sh`, not
      hand-edited.
- [ ] No literal `env: - name: COMMAND_TOPIC_X / value: ...` block added to the
      service's deployment manifest.
- [ ] Go code references the env var as a constant (e.g.
      `EnvCommandTopic = "COMMAND_TOPIC_X"`) resolved via
      `topic.EnvProvider(l)(EnvCommandTopic)()`.
- [ ] The actual Kafka topic is provisioned out-of-band (this repo has no
      `KafkaTopic` CRs); document partition/replication needs in the service's
      `docs/kafka.md`.

### Why this convention exists

`libs/atlas-kafka` reads topic names directly from env vars and uses them
verbatim — there is no name-transformation layer. The configmap-as-source-of-
truth pattern lets an operator rename a topic in one place and roll the whole
platform without rebuilding images. Service-local `env:` overrides defeat that.

### Audit verification — DOM-23

Triggers when the diff adds or renames a topic env var.

```bash
# 1. Every topic env var the service consumes (from Go source).
# 2. Each must exist in the base configmap as KEY: "KEY", and must NOT be
#    redeclared as a literal env value in the service's base manifest.
grep -rohE 'COMMAND_TOPIC_[A-Z_]+|EVENT_TOPIC_[A-Z_]+' \
  services/<svc>/atlas.com/<svc> | sort -u \
  | while read -r T; do
      cfg=$(grep -q "  $T: \"$T\"$" deploy/k8s/base/env-configmap.yaml \
              && echo PRESENT || echo MISSING)
      mani=$(grep -A1 "name: $T$" deploy/k8s/base/atlas-<svc>.yaml | grep -q "value:" \
              && echo DUPLICATE || echo CLEAN)
      printf "%-32s configmap=%s manifest_literal=%s\n" "$T" "$cfg" "$mani"
    done
```

Any `configmap=MISSING` or `manifest_literal=DUPLICATE` line is a FAIL.
Overlay key parity is machine-checked by `tools/service-registration-guard.sh`
— run it and cite its exit status rather than re-deriving the overlay diff by
hand.
