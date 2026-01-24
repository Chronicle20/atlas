# atlas-parties

A RESTful microservice that manages party membership and leadership for game characters.

## Overview

This service maintains in-memory party state and coordinates party operations through Kafka messaging. It tracks character membership across parties, handles party lifecycle events, and responds to character status changes from external services.

## External Dependencies

- Kafka (message broker)
- Jaeger (distributed tracing)
- atlas-character service (foreign character data via REST)

## Environment Variables

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger endpoint in host:port format |
| LOG_LEVEL | Logging verbosity (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BASE_SERVICE_URL | Base URL for REST API in scheme://host:port/api/ format |
| BOOTSTRAP_SERVERS | Kafka broker address in host:port format |
| COMMAND_TOPIC_PARTY | Kafka topic for party commands |
| EVENT_TOPIC_PARTY_STATUS | Kafka topic for party status events |
| EVENT_TOPIC_PARTY_MEMBER_STATUS | Kafka topic for party member status events |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events |
| EVENT_TOPIC_INVITE_STATUS | Kafka topic for invite status events |
| COMMAND_TOPIC_INVITE | Kafka topic for invite commands |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
