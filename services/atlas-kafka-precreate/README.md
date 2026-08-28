# atlas-kafka-precreate

A sync-wave-0 Kubernetes Job (`deploy/k8s/base/atlas-kafka-precreate.yaml`)
that pre-creates every Kafka topic an Atlas environment declares before any
other service or Job starts, and seeds committed offsets for any override
consumer group so a first (or re-)sync never replays the full retention
window.

It replaces `deploy/k8s/base/kafka-precreate.sh`, which ran the whole pass
through the `apache/kafka:3.7.2` image's JVM CLI — one JVM cold start per
topic. This tool does the same work as a single Go binary talking directly
to the Kafka admin protocol.

See `docs/domain.md` for the discovery/create/settle/seed/verify phases and
their invariants, and `docs/kafka.md` for the Kafka admin surface.

## External dependencies

- A Kafka broker, reachable at the address(es) in `BOOTSTRAP_SERVERS`.

## Runtime configuration overview

| Variable | Required | Meaning |
| --- | --- | --- |
| `BOOTSTRAP_SERVERS` | yes | Comma-separated `host:port` list of Kafka brokers. Empty or unset fails the Job immediately, before any Kafka client is constructed. |
| `COMMAND_TOPIC_*`, `EVENT_TOPIC_*` | no | Every variable with one of these prefixes and a non-empty value names a topic to create. See `docs/domain.md` for the plain/compacted classification and `docs/kafka.md` for the compacted topic configuration. |
| `KAFKA_CONSUMER_GROUP` | no | Newline-delimited list of override consumer group IDs (one per line). Unset or empty means no groups to seed. |

Exit code `0` means every phase that ran completed successfully; exit code
`1` means a phase returned an error. See `docs/domain.md` for the phase
sequence and error conditions.

## Building the image

`tools/verify.sh`'s bake step only builds targets of type `go-service`, and
this tool is not one, so its image is never built by the gate path. Build it
by hand before opening a PR that changes this directory:

```sh
docker build -t atlas-kafka-precreate:local services/atlas-kafka-precreate
```
