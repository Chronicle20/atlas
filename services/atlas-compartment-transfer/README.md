# atlas-compartment-transfer

A Kafka-based microservice that orchestrates the transfer of assets between different compartments (character inventory, cash shop, storage) in the Atlas system.

## Overview

This service acts as a saga orchestrator for compartment transfers. It receives transfer commands and coordinates the accept/release workflow between source and destination compartments using a two-phase approach.

## External Dependencies

- Kafka (message broker)
- Jaeger (distributed tracing)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers |
| `JAEGER_HOST_PORT` | Jaeger host and port for distributed tracing |
| `LOG_LEVEL` | Logging level (`panic`, `fatal`, `error`, `warn`, `info`, `debug`, `trace`) |
| `COMMAND_TOPIC_COMPARTMENT_TRANSFER` | Topic for compartment transfer commands |
| `COMMAND_TOPIC_COMPARTMENT` | Topic for character compartment commands |
| `COMMAND_TOPIC_CASH_COMPARTMENT` | Topic for cash shop compartment commands |
| `COMMAND_TOPIC_STORAGE_COMPARTMENT` | Topic for storage compartment commands |
| `EVENT_TOPIC_COMPARTMENT_STATUS` | Topic for character compartment status events |
| `EVENT_TOPIC_CASH_COMPARTMENT_STATUS` | Topic for cash shop compartment status events |
| `EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS` | Topic for storage compartment status events |
| `EVENT_TOPIC_COMPARTMENT_TRANSFER_STATUS` | Topic for compartment transfer status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [Saga](docs/saga.md)
