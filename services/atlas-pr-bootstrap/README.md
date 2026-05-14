# atlas-pr-bootstrap

Image used by ephemeral per-PR environments for bootstrap (PostSync hook)
and cleanup (PostDelete hook). Two entrypoints share one image:

- `/atlas/bootstrap.sh` — uploads the canonical WZ zip and seeds every
  domain via the existing atlas-ui SetupPage endpoints.
- `/atlas/cleanup.sh` — drops per-env Postgres DBs, Kafka topics, Kafka
  consumer groups, Redis keys, ghcr image tags, and Pi-hole A records.

Both scripts read `ATLAS_ENV` and emit JSON-line logs to stdout for Loki.

See `docs/runbooks/ephemeral-pr-deployments.md` for operational docs.
