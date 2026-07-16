# atlas-mini-games

Owns the mini-room mini-games (Omok and Match Cards): room lifecycle
(create/visit/leave/chat/expel), gameplay (ready/start/move-stone/flip-card/
tie/retreat/give-up/skip/exit-after-game), and each character's persistent
win/tie/loss record per game type. Live room state lives in a process-wide
in-memory registry (`game.Registry`, not Redis); only the win/tie/loss records
are persisted (PostgreSQL, `game_records`).

Because room state is in-memory, the service pins `replicas: 1`.

## Domain

- **game** — the room model + `game.Registry` (tenant-partitioned, RWMutex),
  the Omok / Match Cards engines (`game/omok`, `game/matchcards`), and the
  `game.Processor` that handles every lifecycle/gameplay command and emits the
  resulting status events. Session score (per-room, never persisted) and the
  tie / forfeit-farm cooldowns live here.
- **record** — the immutable win/tie/loss `Model`, its GORM `Entity`
  (`game_records`, surrogate uuid PK + `(tenant_id, character_id, game_type)`
  unique index), and `record.Processor` / `ApplyResult` (two-row atomic upsert
  on game end via `db.Transaction`).
- **data/** — thin REST clients the validation ladder reads through:
  `character` (alive check), `map` (fieldLimit "can't start game here"),
  `inventory` (owns the Omok/Match-Cards set item), `chalkboard` (blocks
  opening a room while a chalkboard is open).

## REST endpoints

Collection endpoints return a JSON:API paginated envelope (`meta.page` + `first`/`prev`/`next`/`last` links); page with `page[number]`/`page[size]` (default & max page size 250 — a field/character holds few rooms/records).

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/characters/{characterId}/game-records` | Win/tie/loss record per game type (zero-filled for never-played types). Paginated. |
| GET | `/api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/games` | Every mini-game room registered in the field (for atlas-channel's map-entry balloon reconcile). Paginated. |

## Kafka topics

| Direction | Topic env | Notes |
|-----------|-----------|-------|
| Consumes | `COMMAND_TOPIC_MINI_GAME` | Lifecycle + gameplay commands. |
| Consumes | `EVENT_TOPIC_SESSION_STATUS` | Session destroy → membership teardown. |
| Consumes | `EVENT_TOPIC_CHARACTER_STATUS` | Logout / map-change / channel-change → teardown. |
| Produces | `EVENT_TOPIC_MINI_GAME_STATUS` | Room + gameplay status events consumed by atlas-channel. |

## External dependencies

- **PostgreSQL** — persists `game_records` only.
- **Kafka** — command/event transport (topics above).
- **atlas-character** / **atlas-data** (maps) / **atlas-inventory** /
  **atlas-chalkboards** — REST reads for the create/visit validation ladder.

## Runtime configuration

| Variable | Description |
|----------|-------------|
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | PostgreSQL connection. |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers. |
| `REST_PORT` | REST server port. |
| `COMMAND_TOPIC_MINI_GAME` / `EVENT_TOPIC_MINI_GAME_STATUS` | Mini-game command / status topic names. |
| `EVENT_TOPIC_SESSION_STATUS` / `EVENT_TOPIC_CHARACTER_STATUS` | Consumed teardown topics. |
| `CHARACTERS_SERVICE_URL` / `DATA_SERVICE_URL` / `INVENTORY_SERVICE_URL` / `CHALKBOARDS_SERVICE_URL` | REST-client base URLs (fall back to `BASE_SERVICE_URL`). |
| `TRACE_ENDPOINT` | OpenTelemetry collector gRPC endpoint. |
| `LOG_LEVEL` | Log level (e.g. `debug`, `info`). |
