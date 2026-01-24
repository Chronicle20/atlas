# atlas-portals

Handles portal entry commands and character map transitions. Maintains an in-memory cache of blocked portals per character and coordinates with external services to determine portal behavior.

## External Dependencies

- **Kafka**: Consumes portal commands and character status events; produces character commands and status events
- **DATA Service**: REST client for portal data lookup
- **In-Memory Cache**: Blocked portal state per character (cleared on logout)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `JAEGER_HOST` | Jaeger tracing endpoint |
| `LOG_LEVEL` | Logging level |
| `REST_PORT` | REST server port |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers |
| `DATA_SERVICE_URL` | DATA service base URL |
| `COMMAND_TOPIC_PORTAL` | Portal command topic |
| `COMMAND_TOPIC_PORTAL_ACTIONS` | Portal actions command topic |
| `COMMAND_TOPIC_CHARACTER` | Character command topic |
| `EVENT_TOPIC_CHARACTER_STATUS` | Character status event topic |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
