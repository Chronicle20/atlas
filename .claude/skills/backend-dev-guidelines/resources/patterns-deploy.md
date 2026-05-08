# Deployment & Wiring Patterns

This file documents two conventions that are easy to break when adding new dependencies or new Kafka topics to a service. Both are enforced as audit checks by the `backend-guidelines-reviewer` (DOM-22, DOM-23). Read once before adding a new lib dep or a new topic.

---

## Dockerfile must match service `go.mod` (DOM-22)

Each service has its own multi-stage `Dockerfile` at `services/<svc>/Dockerfile`. The build stage hand-rolls a minimal `go.work` and `go mod edit -replace` block that lists every `Chronicle20/atlas/libs/*` module the service depends on. **Adding a new `libs/atlas-X` dep to the service's `go.mod` is not enough — the Dockerfile must be updated in four places, or the docker build fails on `go build` with a missing-replace error.**

The four places (template using `atlas-redis` as the new dep):

1. **Module manifests COPY** (top of build stage) — copy `go.mod` and `go.sum` so `go mod download` can resolve:
   ```dockerfile
   COPY libs/atlas-redis/go.mod libs/atlas-redis/go.sum libs/atlas-redis/
   ```
   (If the lib has no `go.sum`, omit it — see other libs in the file for the pattern.)

2. **Embedded `go.work`** — add the lib to the `use (...)` block:
   ```dockerfile
   echo '    ./libs/atlas-redis' >> go.work && \
   ```

3. **Source COPY** — copy the actual source after the `go mod download` step:
   ```dockerfile
   COPY libs/atlas-redis libs/atlas-redis
   ```

4. **`go mod edit -replace`** — pin the in-image path so the service's go.mod replace directive works inside the container:
   ```dockerfile
   -replace=github.com/Chronicle20/atlas/libs/atlas-redis=/app/libs/atlas-redis \
   ```

### Checklist when adding a new `libs/atlas-X` dep

- [ ] `go get` (or `import` + `go mod tidy`) added it to the service's `go.mod`.
- [ ] All four blocks in `services/<svc>/Dockerfile` updated.
- [ ] Verify with: `docker build -f services/<svc>/Dockerfile -t scratch:test .` from the repo root.

If the new dep transitively pulls in further `libs/atlas-Y` libs, those need the same treatment. Inspect the new lib's `go.mod` for `Chronicle20/atlas/libs/*` entries and add any that aren't already in the service Dockerfile.

### Why `go build ./...` is not enough

Local `go build ./...` runs against the root `go.work` which lists every lib. The Dockerfile builds against a freshly-constructed minimal `go.work` that lists only the libs the author remembered to add. A new dep that local builds resolve happily can break the docker build the moment the deploy runs.

**Verification commands** the reviewer uses:

```bash
# Find DIRECT-require libs only (skip `replace`-only lines and `// indirect`),
# then check each appears ≥4 times in the Dockerfile.
# Pipe to `while read` so this works in both bash and zsh without word-split quirks.
awk '/^require \(/{flag=1; next} /^\)/{flag=0} flag && !/\/\/ indirect/' \
  services/<svc>/atlas.com/<svc>/go.mod \
  | grep -oE "Chronicle20/atlas/libs/atlas-[a-zA-Z0-9-]+" | sort -u \
  | while read -r L; do
      short=${L##*/}
      count=$(grep -c "$short" services/<svc>/Dockerfile)
      verdict=$([ "$count" -ge 4 ] && echo OK || echo FAIL)
      printf "%-22s -> %d mentions [%s]\n" "$short" "$count" "$verdict"
    done
```

Any direct-require lib with fewer than 4 mentions is a FAIL. (Libs listed only in `replace` directives are legacy/cleanup material — out of scope for this check; libs marked `// indirect` are pulled in transitively and don't need explicit Dockerfile blocks because their parent already does.)

---

## Kafka topic naming (DOM-23)

Every Kafka topic in this project follows a single rigid convention:

1. **The env-var name and the literal topic name are identical** and use SHOUTY_SNAKE_CASE prefixed with `COMMAND_TOPIC_` or `EVENT_TOPIC_`.

   Examples (from `deploy/k8s/env-configmap.yaml`):
   ```yaml
   COMMAND_TOPIC_DATA: "COMMAND_TOPIC_DATA"
   COMMAND_TOPIC_MONSTER: "COMMAND_TOPIC_MONSTER"
   EVENT_TOPIC_CHARACTER_STATUS: "EVENT_TOPIC_CHARACTER_STATUS"
   ```

   **Anti-patterns:** `command.monster`, `monster-events`, `topic_monster`, `monster.command.v1`. Anything dotted, lowercase, hyphenated, or versioned diverges from the convention and breaks the configmap-driven topic discovery.

2. **All topics live in `deploy/k8s/env-configmap.yaml`** under the `# Command Topics` or `# Event Topics` sections, kept alphabetically ordered.

3. **Services consume topic names via `envFrom: configMapRef: atlas-env`**, NOT via hand-written `env: - name: COMMAND_TOPIC_X / value: ...` blocks in the service's deployment manifest. The configmap is the single source of truth — duplicating the value in a service manifest invites drift.

   The service Go code reads `os.Getenv("COMMAND_TOPIC_X")` (typically via `topic.EnvProvider(l)("COMMAND_TOPIC_X")()`) and uses the resulting value as the literal topic name with no transformation.

### Checklist when adding a new topic

- [ ] Picked a name in `COMMAND_TOPIC_<DOMAIN>` or `EVENT_TOPIC_<DOMAIN>_STATUS` form.
- [ ] Added an entry to `deploy/k8s/env-configmap.yaml` of the form `KEY: "KEY"` (key and value identical), in alphabetical order under the appropriate section.
- [ ] Did **not** add a literal `env: - name: COMMAND_TOPIC_X / value: ...` block to the service's deployment manifest. The service's `envFrom: configMapRef: atlas-env` pulls it in automatically.
- [ ] Service Go code references the env var as a constant (e.g. `EnvCommandTopic = "COMMAND_TOPIC_X"`) and resolves it via `topic.EnvProvider(l)(EnvCommandTopic)()`.
- [ ] The actual Kafka topic is provisioned out-of-band (this repo has no `KafkaTopic` CRs); document partition/replication requirements in the service's `docs/kafka.md`.

### Why this convention exists

Atlas's `libs/atlas-kafka` reads topic names directly from env vars and uses them verbatim. There is no name-transformation layer (no `service-name + "/" + topic`). The configmap-as-source-of-truth pattern lets operators rename a topic in exactly one place and roll the whole platform without rebuilding service images. Service-local `env:` overrides defeat that.

**Verification commands** the reviewer uses:

```bash
# 1. Find every topic env var the service consumes (from Go source).
# 2. Each must exist in env-configmap.yaml with KEY: "KEY" shape AND must NOT
#    be redeclared as a literal env value in the service's deployment manifest.
grep -rohE 'COMMAND_TOPIC_[A-Z_]+|EVENT_TOPIC_[A-Z_]+' \
  services/<svc>/atlas.com/<svc> | sort -u \
  | while read -r T; do
      cfg=$(grep -q "  $T: \"$T\"$" deploy/k8s/env-configmap.yaml \
              && echo PRESENT || echo MISSING)
      mani=$(grep -A1 "name: $T$" deploy/k8s/<svc>.yaml | grep -q "value:" \
              && echo DUPLICATE || echo CLEAN)
      printf "%-32s configmap=%s manifest_literal=%s\n" "$T" "$cfg" "$mani"
    done
```

Any `configmap=MISSING` or `manifest_literal=DUPLICATE` line is a FAIL.
