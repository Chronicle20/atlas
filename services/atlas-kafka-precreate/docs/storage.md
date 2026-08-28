## Tables

None. This service holds no application-owned database. Its only persisted
state is on the Kafka cluster itself: topic existence/configuration and
consumer-group committed offsets, both managed through the Kafka Admin
protocol (see `docs/kafka.md`).

## Relationships

None.

## Indexes

None.

## Migration Rules

None.
