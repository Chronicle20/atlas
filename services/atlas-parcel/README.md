# atlas-parcel

## Responsibility

The atlas-parcel service is Duey the mailman's custody store. It holds items and mesos sent between characters (including across worlds and across servers) while they are in transit, notifies the recipient once a parcel becomes receivable, and sweeps unclaimed parcels for expiry — returning a returnable parcel to its sender or discarding a non-returnable one. Custody transfer (accept/release/restore/remove) is driven by the send/withdraw sagas over Kafka; discard and the read API are reached directly over REST.

## External Dependencies

- **PostgreSQL**: Persistent storage for parcels in custody
- **Kafka**: Custody command consumption, custody status ack production, and arrival status event production

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `REST_PORT` | HTTP server port |
| `DATABASE_*` | PostgreSQL connection configuration |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers |
| `COMMAND_TOPIC_PARCEL_CUSTODY` | Kafka topic for parcel custody commands |
| `EVENT_TOPIC_PARCEL_CUSTODY_STATUS` | Kafka topic for parcel custody status (ack) events |
| `EVENT_TOPIC_PARCEL_STATUS` | Kafka topic for parcel arrival status events |
| `PARCEL_EXPIRY_INTERVAL_SECONDS` | Expiry sweep cadence in seconds (default 3600 — 1 hour) |
| `PARCEL_NOTIFICATION_INTERVAL_SECONDS` | Arrival-notification sweep cadence in seconds (default 300 — 5 minutes) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Parcel](docs/parcel.md)
