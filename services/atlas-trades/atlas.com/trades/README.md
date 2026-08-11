# atlas-trades

Owns 1:1 player-to-player trading: the trade room lifecycle and the durable
ledger of completed trades. Live room state lives in a process-local
in-memory registry, not the database, so the service pins `replicas: 1` and
carries no HPA — that is a correctness constraint, not a capacity choice.

The database backs only the completed-trade ledger and the transactional
outbox (`libs/atlas-outbox`), which is drained to Kafka by the outbox drainer
booted in `main.go`.

## Boundaries

- atlas-trades never writes a packet — atlas-channel owns the wire.
- atlas-channel never mutates inventory or meso — atlas-trades drives
  settlement through the saga orchestrator.

## Configuration

Standard Atlas service environment (see `deploy/k8s/base/atlas-trades.yaml`):

| Variable | Purpose |
|---|---|
| `REST_PORT` | HTTP listen port (`8080` in-cluster). |
| `DB_NAME`, `DB_USER`, `DB_PASSWORD` | PostgreSQL connection (`atlas-trades`, env-suffixed by the kustomize overlays). |
| `BOOTSTRAP_SERVERS` | Kafka brokers. |
| `COMMAND_TOPIC_TRADE` | Inbound trade commands. |
| `EVENT_TOPIC_TRADE_STATUS` | Outbound trade status events. |
| `LOG_LEVEL` | Logrus level. |

## REST endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/readyz` | Readiness probe. |

Ingress routes `/api/trades/...` to this service (`deploy/shared/routes.conf`).
