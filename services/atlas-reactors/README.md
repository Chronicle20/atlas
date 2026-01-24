# atlas-reactors

Manages reactor instances as in-memory volatile game objects. Reactors are interactive objects within maps that respond to player actions, transitioning through states when hit until reaching a terminal state where they trigger and are destroyed.

This service uses an in-memory registry pattern. Reactor instances exist only during active game sessions and are not persisted across service restarts.

## External Dependencies

- Kafka (BOOTSTRAP_SERVERS)
- Jaeger (JAEGER_HOST)
- atlas-data service (DATA root URL)

## Runtime Configuration

| Variable                       | Description                              |
|--------------------------------|------------------------------------------|
| BOOTSTRAP_SERVERS              | Kafka broker addresses                   |
| JAEGER_HOST                    | Jaeger tracing host:port                 |
| LOG_LEVEL                      | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| COMMAND_TOPIC_REACTOR          | Kafka topic for reactor commands         |
| EVENT_TOPIC_REACTOR_STATUS     | Kafka topic for reactor status events    |
| COMMAND_TOPIC_REACTOR_ACTIONS  | Kafka topic for reactor action commands  |
| DATA                           | Root URL for atlas-data service          |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
