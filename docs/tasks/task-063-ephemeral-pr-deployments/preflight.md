# Pre-flight findings (task-063)

## CREATEDB on db-credentials user
- Result: PASS 2026-05-08
- Action taken (if FAIL): n/a

### Evidence
Query against `postgres.home` as the user from `atlas/db-credentials`:

```
 rolname | rolcreatedb
---------+-------------
 atlas   | t
(1 row)
```

### Note on secret hygiene
`kubectl get secret -n atlas db-credentials -o jsonpath='{.data.DB_USER}' | base64 -d` returns `atlas \r\n` (a literal trailing space plus CR+LF). The same applies to `DB_PASSWORD`. Authentication only succeeds after stripping the trailing whitespace (`tr -d ' \r\n'`); the literal-space form fails with `password authentication failed for user "atlas "`. Atlas services in-cluster appear to tolerate this today, but the per-PR overlay tooling introduced in later phases should either strip whitespace defensively or the secret should be re-issued without the trailing whitespace. Tracking this here so a downstream phase can address it if needed.

## Kafka auto.create.topics.enable
- Result: true 2026-05-08
- Action taken (if false): n/a — broker is permissive; per-PR overlay does NOT need a PreSync topic-creation hook for ephemeral envs

### Evidence
The plan's stock label selector returns no pods (`kubectl get pod -A -l app.kubernetes.io/name=kafka` is empty) and `--entity-name 0` is wrong for this cluster (node.id is 1, not 0). Reproducible commands:

Pod selector that worked:
```
kubectl get pod -n kafka -l app=kafka -o name
# pod/kafka-broker-0
```

Broker image is `apache/kafka:4.1.1` (vanilla Apache, not Strimzi/Bitnami/Confluent). `kafka-configs.sh` lives at `/opt/kafka/bin/kafka-configs.sh`. PLAINTEXT listener is on `localhost:9092`.

Two independent confirmations:

1) Container env:
```
$ kubectl exec -n kafka kafka-broker-0 -- env | grep -i auto
KAFKA_AUTO_CREATE_TOPICS_ENABLE=true
```

2) Rendered broker config (`/opt/kafka/config/server.properties`):
```
$ kubectl exec -n kafka kafka-broker-0 -- grep -i auto.create /opt/kafka/config/server.properties
auto.create.topics.enable=true
```

`kafka-configs.sh --describe --entity-name 0` reported "broker '0' doesn't exist and doesn't have dynamic config" — confirmed via `node.id=1` in the same `server.properties`. The plan example's `--entity-name 0` should be parameterised on the actual broker/node id when used elsewhere; here the static property file is authoritative.
