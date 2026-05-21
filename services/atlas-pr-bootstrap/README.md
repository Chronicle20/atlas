# atlas-pr-bootstrap

Image used by ephemeral per-PR environments for bootstrap (PostSync hook)
and cleanup (PostDelete hook). Two entrypoints share one image:

- `/atlas/bootstrap.sh` — uploads the canonical WZ zip and seeds every
  domain via the existing atlas-ui SetupPage endpoints.
- `/atlas/cleanup.sh` — drops per-env Postgres DBs, Kafka topics, Kafka
  consumer groups, Redis keys, ghcr image tags, and Pi-hole A records.

Both scripts read `ATLAS_ENV` and emit JSON-line logs to stdout for Loki.

See `docs/runbooks/ephemeral-pr-deployments.md` for operational docs.

## Runtime dependencies

The image is single-stage Alpine 3.23 and contains:

- apk: `bash`, `curl`, `jq`, `postgresql-client`, `redis`, `ca-certificates`, `github-cli`, `kubectl`, `unzip` (build-time only)
- `rpk` (Redpanda CLI; Kafka admin protocol compatible) — vendored as a
  static binary from `redpanda-data/redpanda` releases. Used by
  `cleanup.sh` for topic and consumer-group list/delete. Pin via the
  `RPK_VERSION` Dockerfile build arg.

Earlier revisions of the image baked the Apache Kafka tarball + an
OpenJDK 17 JRE for the same operations. `rpk` is a single ~30 MB static
Go binary and removes the JVM startup latency from every cleanup call.
