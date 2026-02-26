# atlas-merchant

The merchant service manages character shops and hired merchants. It handles the full shop lifecycle — creation, opening, maintenance, closing — along with item listing management, visitor tracking, purchase transactions with fee calculation, and post-closure item/meso storage via Frederick (the hired merchant NPC).

Character shops close automatically on owner disconnect. Hired merchants operate independently of the owner's session, expire after 24 hours, and store unsold items and accumulated mesos at Frederick for later retrieval. A tiered notification scheduler reminds owners to collect stored goods.

## External Dependencies

- **PostgreSQL** — shops, listings, messages, Frederick items/mesos/notifications
- **Redis** — active shop registry (TenantRegistry), map placement index, visitor tracking (forward/reverse Index)
- **Kafka** — command ingestion, status/listing event production, inventory (compartment) and meso (character) integration
- **OpenTelemetry** — distributed tracing via OTLP/gRPC
- **atlas-data** — portal position data for placement validation (REST)

## Runtime Configuration

| Variable | Purpose |
|---|---|
| `BOOTSTRAP_SERVERS` | Kafka broker address |
| `COMMAND_TOPIC_MERCHANT` | Merchant command topic |
| `EVENT_TOPIC_MERCHANT_STATUS` | Merchant status event topic |
| `EVENT_TOPIC_MERCHANT_LISTING` | Merchant listing event topic |
| `COMMAND_TOPIC_COMPARTMENT` | Compartment (inventory) command topic |
| `EVENT_TOPIC_COMPARTMENT_STATUS` | Compartment status event topic |
| `COMMAND_TOPIC_CHARACTER` | Character command topic |
| `EVENT_TOPIC_CHARACTER_STATUS` | Character status event topic |
| `REST_PORT` | HTTP server port |
| `TRACE_ENDPOINT` | OTLP trace collector endpoint |
| `LOG_LEVEL` | Logging level |
| `DATA_ROOT_URL` | Base URL for atlas-data REST service |

## REST Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/merchants?mapId={id}` | List shops on a map |
| GET | `/merchants/{shopId}` | Get shop details with listings |
| GET | `/merchants/{shopId}/relationships/listings` | Get shop listings |
| GET | `/characters/{characterId}/merchants` | Get shops owned by character |

## Kafka Commands Consumed

| Topic | Command | Description |
|---|---|---|
| `COMMAND_TOPIC_MERCHANT` | `PLACE_SHOP` | Create a new shop |
| `COMMAND_TOPIC_MERCHANT` | `OPEN_SHOP` | Transition shop from Draft to Open |
| `COMMAND_TOPIC_MERCHANT` | `CLOSE_SHOP` | Manually close a shop |
| `COMMAND_TOPIC_MERCHANT` | `ENTER_MAINTENANCE` | Enter maintenance mode |
| `COMMAND_TOPIC_MERCHANT` | `EXIT_MAINTENANCE` | Exit maintenance mode |
| `COMMAND_TOPIC_MERCHANT` | `ADD_LISTING` | Add an item listing |
| `COMMAND_TOPIC_MERCHANT` | `REMOVE_LISTING` | Remove an item listing |
| `COMMAND_TOPIC_MERCHANT` | `UPDATE_LISTING` | Update listing price/bundles |
| `COMMAND_TOPIC_MERCHANT` | `PURCHASE_BUNDLE` | Purchase bundles from a listing |
| `COMMAND_TOPIC_MERCHANT` | `ENTER_SHOP` | Visitor enters shop |
| `COMMAND_TOPIC_MERCHANT` | `EXIT_SHOP` | Visitor exits shop |
| `COMMAND_TOPIC_MERCHANT` | `SEND_MESSAGE` | Send chat message in shop |
| `COMMAND_TOPIC_MERCHANT` | `RETRIEVE_FREDERICK` | Retrieve items/mesos from Frederick |

## Kafka Events Produced

| Topic | Event | Description |
|---|---|---|
| `EVENT_TOPIC_MERCHANT_STATUS` | `SHOP_OPENED` | Shop transitioned to Open |
| `EVENT_TOPIC_MERCHANT_STATUS` | `SHOP_CLOSED` | Shop closed (manual, expired, sold out, disconnect, empty) |
| `EVENT_TOPIC_MERCHANT_STATUS` | `MAINTENANCE_ENTERED` | Shop entered maintenance |
| `EVENT_TOPIC_MERCHANT_STATUS` | `MAINTENANCE_EXITED` | Shop exited maintenance |
| `EVENT_TOPIC_MERCHANT_STATUS` | `VISITOR_ENTERED` | Visitor entered shop |
| `EVENT_TOPIC_MERCHANT_STATUS` | `VISITOR_EXITED` | Visitor exited shop |
| `EVENT_TOPIC_MERCHANT_STATUS` | `CAPACITY_FULL` | Shop at visitor capacity |
| `EVENT_TOPIC_MERCHANT_STATUS` | `PURCHASE_FAILED` | Purchase attempt failed |
| `EVENT_TOPIC_MERCHANT_STATUS` | `FREDERICK_NOTIFICATION` | Frederick retrieval reminder |
| `EVENT_TOPIC_MERCHANT_LISTING` | `LISTING_PURCHASED` | Listing bundles purchased |
| `COMMAND_TOPIC_COMPARTMENT` | `RELEASE_ASSET` / `ACCEPT_ASSET` | Inventory integration commands |
| `COMMAND_TOPIC_CHARACTER` | `REQUEST_CHANGE_MESO` | Meso transfer commands |

## Kafka Events Consumed

| Topic | Event | Description |
|---|---|---|
| `EVENT_TOPIC_COMPARTMENT_STATUS` | `ACCEPTED` / `RELEASED` / `ERROR` | Inventory operation confirmations |
| `EVENT_TOPIC_CHARACTER_STATUS` | `LOGOUT` | Character disconnect — closes character shops |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
