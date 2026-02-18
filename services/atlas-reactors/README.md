# atlas-reactors

Manages reactor instances as in-memory volatile game objects. Reactors are interactive objects within maps that respond to player actions, transitioning through states when hit until reaching a terminal state where they trigger and are destroyed.

This service uses an in-memory registry pattern. Reactor instances exist only during active game sessions and are not persisted across service restarts.

## External Dependencies

- Kafka (BOOTSTRAP_SERVERS)
- OpenTelemetry Collector (TRACE_ENDPOINT)
- atlas-data service (DATA root URL)

## Runtime Configuration

| Variable                       | Description                              |
|--------------------------------|------------------------------------------|
| BOOTSTRAP_SERVERS              | Kafka broker addresses                   |
| TRACE_ENDPOINT                 | OpenTelemetry gRPC collector endpoint    |
| LOG_LEVEL                      | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT                      | HTTP server port                         |
| COMMAND_TOPIC_REACTOR          | Kafka topic for reactor commands         |
| EVENT_TOPIC_REACTOR_STATUS     | Kafka topic for reactor status events    |
| COMMAND_TOPIC_REACTOR_ACTIONS  | Kafka topic for reactor action commands  |
| EVENT_TOPIC_DROP_STATUS        | Kafka topic for drop status events       |
| COMMAND_TOPIC_DROP             | Kafka topic for drop commands            |
| ITEM_REACTOR_ACTIVATION_DELAY_MS | Item reactor activation delay in milliseconds (default: 5000) |
| DATA                           | Root URL for atlas-data service          |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
