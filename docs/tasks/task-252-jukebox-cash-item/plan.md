# Jukebox Cash Item Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Using a classification-510 (song player / jukebox) cash item consumes one item and changes the background music for every character in the user's field for the song's own length, replaying to late arrivals and stopping on expiry.

**Architecture:** The weather cash-item pipeline with the BGM leg removed and a client-supplied duration added — channel arm → saga (`DestroyAsset` + `play_jukebox`) → atlas-saga-orchestrator → `PLAY_JUKEBOX` on `COMMAND_TOPIC_MAP` → atlas-maps jukebox registry with a capped TTL → `JUKEBOX_START` / `JUKEBOX_END` on `EVENT_TOPIC_MAP_STATUS` → atlas-channel broadcasts `PlayJukebox` to the field. The client resolves the song from the item's own WZ `info/path` node and restores the map BGM itself, so no `FieldEffect` BGM packet and no atlas-data change are involved.

**Tech Stack:** Go (5 modules), Kafka (segmentio), JSON:API REST (api2go/jsonapi), gorilla/mux, logrus, testify/plain-Go table tests.

**Spec:** [design.md](design.md) (PRD for reference: [prd.md](prd.md))

## Global Constraints

- **The stop signal is exactly `-1`**, never "some negative id". Any other negative value spins the client's `CMapLoadable::Update` loop forever (design §3.2).
- **No `FieldEffect` BGM packet anywhere in this feature.** Sending one sets `m_sChangedBgmUOL` and permanently breaks `RestoreBGM` (design §3.1).
- **No change to `libs/atlas-packet/field/clientbound/play_jukebox.go`** or to its pinned evidence records for `gms_v95` / `jms_v185`.
- **No atlas-data change.** The client reads the WZ node itself.
- Duration is the client's `IWzSound::length` in **milliseconds**, capped server-side at `maxJukeboxDuration = 10 * time.Minute`. Never converted to seconds.
- Every registry key, Kafka message, and REST lookup is tenant-scoped via `tenant.MustFromContext(ctx)` / `FieldKey{Tenant, Field}`.
- The arm is keyed on the cash-slot **type byte 20**, never on a hard-coded item id.
- No `EnableActions` / unlock packet from the success path — the non-silent inventory operation from the consume commit clears the client's exclusive-request lock.
- Implementers run module-local `go build ./... && go test ./...` only. Repo-wide `tools/verify.sh` is the controller's gate, not the implementer's.

---

## Task 1: `ItemUseSongPlayer` serverbound codec

### Files

- `libs/atlas-packet/cash/serverbound/item_use_song_player.go` — **new file**; the type-20 sub-body codec
- `libs/atlas-packet/cash/serverbound/item_use_song_player_test.go` — **new file**; round-trip tests
- `libs/atlas-packet/cash/serverbound/item_use_field_effect.go` — read-only; the structural template
- `libs/atlas-packet/cash/serverbound/item_use_morph_coupon.go` — read-only; the `packet-audit:fname` + evidence-comment template
- `libs/atlas-packet/test/tenant.go` — read-only; `pt.Variants`, `pt.CreateContext`
- `libs/atlas-packet/test/roundtrip.go` — read-only; `pt.RoundTrip`

Module root for `go build`/`go test`: `libs/atlas-packet`.

Patterns to copy: `libs/atlas-packet/cash/serverbound/item_use_field_effect.go:13-50` (struct + Encode/Decode shape), `libs/atlas-packet/cash/serverbound/item_use_field_effect_test.go:9-38` (both round-trip tests).

### Interfaces

- Produces: `serverbound.ItemUseSongPlayer` with `NewItemUseSongPlayer(updateTimeFirst bool) *ItemUseSongPlayer`, accessors `SoundLengthMs() uint32` and `UpdateTime() uint32`, plus `Operation() string`, `String() string`, `Encode`, `Decode`. Task 7 consumes all of these.

- [ ] **Step 1: Write the failing tests**

Create `item_use_song_player_test.go`. Two test functions, each ranging over `pt.Variants` with `t.Run(v.Name, ...)`, exactly like `item_use_field_effect_test.go` (imports, subtest wrapper, and `pt.CreateContext` call copied from there).

| test func | struct literal under test | `New...` arg | assertions |
|---|---|---|---|
| `TestItemUseSongPlayerUpdateTimeFirstRoundTrip` | `ItemUseSongPlayer{soundLengthMs: 123456, updateTimeFirst: true}` | `true` | `output.SoundLengthMs() == 123456` |
| `TestItemUseSongPlayerNoUpdateTimeFirstRoundTrip` | `ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false}` | `false` | `output.SoundLengthMs() == 123456` **and** `output.UpdateTime() == 77777` |

Both call `pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)` — which itself asserts zero unconsumed bytes, so the `updateTimeFirst == true` case proves no trailing int is written.

Add a third, byte-exact test that pins the wire order (the round-trip alone cannot catch a swapped field order when both fields are int32):

```go
func TestItemUseSongPlayerWireOrder(t *testing.T) {
	// soundLengthMs first, then the trailing updateTime on the versions that
	// trail it (GMS <= v84). Little-endian int32 each.
	// 0x0001E240 == 123456, 0x00012FD1 == 77777.
}
```

Case table for `TestItemUseSongPlayerWireOrder`:

| subtest | struct | expected bytes (hex) |
|---|---|---|
| `updateTimeFirst` | `ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: true}` | `40 e2 01 00` |
| `updateTimeTrails` | `ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false}` | `40 e2 01 00 d1 2f 01 00` |

Use `pt.Encode(t, ctx, input.Encode, nil)` (`libs/atlas-packet/test/roundtrip.go:16`) to get the bytes, and compare with `bytes.Equal`. Use `pt.CreateContext("GMS", 83, 1)` for the context; the codec has no tenant-dependent branch, so one variant suffices here.

- [ ] **Step 2: Run the tests to verify they fail**

```
cd libs/atlas-packet && go test ./cash/serverbound/ -run ItemUseSongPlayer -v
```

Expected: FAIL — `undefined: ItemUseSongPlayer`.

- [ ] **Step 3: Write the codec**

Create `item_use_song_player.go` in `package serverbound`, imports `context`, `fmt`, `github.com/sirupsen/logrus`, and the socket `request`/`response` packages (copy the import block from `item_use_field_effect.go`).

Doc comment (required — it carries the derivation evidence and the audit marker):

```go
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest
//
// ItemUseSongPlayer is the USE_CASH_ITEM sub-body for a song player (jukebox)
// cash item, item classification 510 — cash-slot type 20 on every version
// examined (get_cashslot_item_type @0x488c70 on GMS v95.0: `case 510: result = 20`).
//
// The sub-body is exactly one int32: the WZ sound's own IWzSound::length, in
// milliseconds. IDA-verified on two builds that bracket the supported range:
//   GMS v95.0 (GMS_v95.0_U_DEVM.exe) case-20 arm @0x9ed51e — reads the item's
//     info/path node (StringPool 0x734), resolves it via IWzResMan::GetObjectA
//     @0x9ed75a, casts to IWzSound @0x9ed773, calls IWzSound::Getlength
//     @0x9ed7af and passes the result straight to COutPacket::Encode4 @0x9ed7b9.
//   GMS v83 (MapleStory_dump.exe) case-20 arm @0xa0c1a2 — identical sequence,
//     Getlength via the vtable+56 getter sub_644DCF @0xa0c3ed then Encode4
//     @0xa0c3f6.
// Exactly one Encode4 in the arm on both.
//
// The trailing updateTime is NOT part of this arm: it comes from the shared
// send tail on the versions that trail it (GMS <= v84), exactly as documented
// on ItemUseMorphCoupon. cashsb.UpdateTimeFirst(tenant) selects which.
//
// The server never resolves the BGM. The client reads the item's own
// info/path node in CMapLoadable::PlayNextMusic @0x61dab0 and hands it to
// CSoundMan::PlayBGM, so no BGM name crosses the wire in either direction.
type ItemUseSongPlayer struct {
	soundLengthMs   uint32
	updateTime      uint32
	updateTimeFirst bool
}
```

Then, mirroring `ItemUseFieldEffect` exactly:

```go
func NewItemUseSongPlayer(updateTimeFirst bool) *ItemUseSongPlayer {
	return &ItemUseSongPlayer{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseSongPlayer) SoundLengthMs() uint32 { return m.soundLengthMs }
func (m ItemUseSongPlayer) UpdateTime() uint32    { return m.updateTime }

func (m ItemUseSongPlayer) Operation() string { return "ItemUseSongPlayer" }

func (m ItemUseSongPlayer) String() string {
	return fmt.Sprintf("soundLengthMs [%d] updateTime [%d]", m.soundLengthMs, m.updateTime)
}

func (m ItemUseSongPlayer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.soundLengthMs)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseSongPlayer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.soundLengthMs = r.ReadUint32()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```
cd libs/atlas-packet && go build ./... && go test ./cash/serverbound/ -run ItemUseSongPlayer -v
```

Expected: PASS (all variants).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_use_song_player.go libs/atlas-packet/cash/serverbound/item_use_song_player_test.go
git commit -m "feat(packet): add ItemUseSongPlayer type-20 cash item use sub-body"
```

---

## Task 2: `play_jukebox` saga action in `libs/atlas-saga`

### Files

- `libs/atlas-saga/model.go` — add the `PlayJukebox` Action constant (the `// Field effect actions` block, line ~259)
- `libs/atlas-saga/payloads.go` — add `PlayJukeboxPayload` next to `FieldEffectWeatherPayload` (line ~1172)
- `libs/atlas-saga/unmarshal.go` — add the `case PlayJukebox:` arm next to `case FieldEffectWeather:` (line ~606)
- `libs/atlas-saga/unmarshal_test.go` — add the decode test
- `libs/atlas-saga/world_transfer_test.go` — add `PlayJukebox` to the `otherActions` slice (line ~126, next to `FieldEffectWeather`) so the collision test covers it

Module root for `go build`/`go test`: `libs/atlas-saga`.

Patterns to copy: `libs/atlas-saga/payloads.go:1172-1181` (payload struct + field comments), `libs/atlas-saga/unmarshal.go:606-611` (switch arm), `libs/atlas-saga/unmarshal_test.go:786-799` (decode test shape).

### Interfaces

- Produces, consumed by Tasks 3 and 7:
  - `saga.PlayJukebox Action = "play_jukebox"`
  - `saga.PlayJukeboxPayload{WorldId world.Id; ChannelId channel.Id; MapId _map.Id; Instance uuid.UUID; ItemId uint32; PlayerName string; DurationMs uint32}` with JSON tags `worldId`, `channelId`, `mapId`, `instance`, `itemId`, `playerName`, `durationMs`.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-saga/unmarshal_test.go` (imports `encoding/json` and `testing` are already present in that file):

```go
func TestUnmarshalPlayJukeboxStep(t *testing.T) {
	data := []byte(`{"stepId":"s1","status":"pending","action":"play_jukebox","payload":{"worldId":0,"channelId":1,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000","itemId":5100000,"playerName":"Chronicle","durationMs":45000},"createdAt":"2026-08-21T00:00:00Z","updatedAt":"2026-08-21T00:00:00Z"}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Payload.(PlayJukeboxPayload)
	if !ok {
		t.Fatalf("payload type = %T", s.Payload)
	}
	if p.ItemId != 5100000 || p.PlayerName != "Chronicle" || p.DurationMs != 45000 {
		t.Fatalf("payload = %+v", p)
	}
	if p.WorldId != 0 || p.ChannelId != 1 || p.MapId != 100000000 {
		t.Fatalf("field coordinates = %+v", p)
	}
}
```

Also add `PlayJukebox,` to the `otherActions` slice in `world_transfer_test.go`, on the line immediately after `FieldEffectWeather,`.

- [ ] **Step 2: Run the test to verify it fails**

```
cd libs/atlas-saga && go test ./... -run PlayJukebox -v
```

Expected: FAIL — `undefined: PlayJukeboxPayload`.

- [ ] **Step 3: Implement**

In `model.go`, in the `// Field effect actions` block:

```go
	// Field effect actions
	FieldEffectWeather Action = "field_effect_weather"
	// PlayJukebox starts a song in one field. DurationMs is the client's own
	// IWzSound::length; atlas-maps caps it. The BGM name is never carried --
	// the client resolves it from the item's WZ info/path node itself.
	PlayJukebox Action = "play_jukebox"
```

In `payloads.go`, immediately after `FieldEffectWeatherPayload`:

```go
// PlayJukeboxPayload represents the payload for starting a jukebox song in a field.
type PlayJukeboxPayload struct {
	WorldId    world.Id   `json:"worldId"`    // WorldId of the field
	ChannelId  channel.Id `json:"channelId"`  // ChannelId of the field
	MapId      _map.Id    `json:"mapId"`      // MapId of the field
	Instance   uuid.UUID  `json:"instance"`   // Instance UUID of the field
	ItemId     uint32     `json:"itemId"`     // Cash song-player item ID
	PlayerName string     `json:"playerName"` // Character who started the song
	DurationMs uint32     `json:"durationMs"` // Song length in MILLISECONDS (client-supplied, server-capped)
}
```

In `unmarshal.go`, immediately after the `case FieldEffectWeather:` arm:

```go
	case PlayJukebox:
		var payload PlayJukeboxPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 4: Run the tests to verify they pass**

```
cd libs/atlas-saga && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga/model.go libs/atlas-saga/payloads.go libs/atlas-saga/unmarshal.go libs/atlas-saga/unmarshal_test.go libs/atlas-saga/world_transfer_test.go
git commit -m "feat(saga): add play_jukebox action and payload"
```

---

## Task 3: `PLAY_JUKEBOX` command in atlas-saga-orchestrator

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/map/kafka.go` — add `CommandTypePlayJukebox` and `PlayJukeboxCommandBody`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer.go` — add `PlayJukeboxCommandProvider`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/processor.go` — add `PlayJukebox` to the `Processor` interface and `ProcessorImpl`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — re-export alias (line ~233), payload alias (line ~367), unmarshal case (line ~1655)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — `sharedsaga.PlayJukebox: {}` in the fire-and-forget block (line ~308)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — `Handler` interface method (line ~184), action-switch case (line ~1003), `handlePlayJukebox` implementation (next to `handleFieldEffectWeather` at line ~3578)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go` — add `sharedsaga.PlayJukebox` to `allActions` (line ~48)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go` — add the invalid-payload test

Module root for `go build`/`go test`: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

Patterns to copy: `map_command/producer.go:14-30` (provider), `map_command/processor.go:33-35` (processor method), `saga/handler.go:3578-3605` (`handleFieldEffectWeather`), `saga/handler_test.go:1422-1441` (invalid-payload test shape).

### Interfaces

- Consumes from Task 2: `sharedsaga.PlayJukebox`, `sharedsaga.PlayJukeboxPayload`.
- Produces, consumed by Task 5 (atlas-maps reads the same JSON off Kafka): command type string `"PLAY_JUKEBOX"` on `COMMAND_TOPIC_MAP`, body `{"itemId":<uint32>,"playerName":<string>,"durationMs":<uint32>}`, envelope `{transactionId, worldId, channelId, mapId, instance, type, body}`.
- Produces for the orchestrator's own packages: `map_command.Processor.PlayJukebox(transactionId uuid.UUID, f field.Model, itemId uint32, playerName string, durationMs uint32) error`.

- [ ] **Step 1: Write the failing tests**

Add `sharedsaga.PlayJukebox,` to the `allActions` slice in `saga/event_acceptance_test.go`, on the line immediately after `sharedsaga.FieldEffectWeather,`. That single edit arms two existing completeness tests — `TestAcceptanceTable_EveryActionRepresented` (event_acceptance_test.go) and `TestStepUnmarshal_EveryActionRepresented` (unmarshal_completeness_test.go) — which will both fail until the acceptance-table entry and the `saga/model.go` unmarshal case exist.

Append to `saga/handler_test.go` (its imports already carry `logrus`, `logrus/hooks/test`, `uuid`, `testify/assert`):

```go
// TestHandlePlayJukebox_InvalidPayload proves handlePlayJukebox rejects a step
// whose payload is not a PlayJukeboxPayload before touching Kafka. The happy
// path is not covered here for the same reason handleFieldEffectWeather is
// not: no fixture in this package stubs the atlas-kafka producer's
// WriterFactory/env-topic resolution. Message-shape coverage lives in
// TestPlayJukeboxCommandProvider (map_command/producer_test.go).
func TestHandlePlayJukebox_InvalidPayload(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	_, ctx := setupContext()

	saga, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(QuestReward). // any type; this test never reaches the saga body
		SetInitiatedBy("test").
		Build()
	assert.NoError(t, err)

	step := NewStep[any]("play-jukebox-step", Pending, PlayJukebox, "invalid-payload-type")

	err = NewHandler(logger, ctx).handlePlayJukebox(saga, step)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid payload")
}
```

Create `map_command/producer_test.go` (new file, `package map_command`) pinning the command's wire shape:

```go
func TestPlayJukeboxCommandProvider(t *testing.T) { ... }
```

Case: build `f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(instanceId).Build()` for a fixed `instanceId := uuid.New()`, call `PlayJukeboxCommandProvider(transactionId, f, 5100000, "Chronicle", 45000)()`, assert exactly one message, `json.Unmarshal` its `Value` into `mapKafka.Command[mapKafka.PlayJukeboxCommandBody]`, and assert every field:

| field | expected |
|---|---|
| `Type` | `mapKafka.CommandTypePlayJukebox` (`"PLAY_JUKEBOX"`) |
| `TransactionId` | the `transactionId` passed in |
| `WorldId` / `ChannelId` / `MapId` | `0` / `1` / `100000000` |
| `Instance` | the `instanceId` passed in |
| `Body.ItemId` | `5100000` |
| `Body.PlayerName` | `"Chronicle"` |
| `Body.DurationMs` | `45000` |

- [ ] **Step 2: Run the tests to verify they fail**

```
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... ./map_command/... 2>&1 | head -40
```

Expected: FAIL — `undefined: PlayJukeboxCommandProvider`, `undefined: handlePlayJukebox`, and the two completeness tests reporting a missing `play_jukebox` entry.

- [ ] **Step 3: Implement**

`kafka/message/map/kafka.go` — add the constant and body next to the weather pair:

```go
	CommandTypePlayJukebox = "PLAY_JUKEBOX"
```

```go
type PlayJukeboxCommandBody struct {
	ItemId     uint32 `json:"itemId"`
	PlayerName string `json:"playerName"`
	DurationMs uint32 `json:"durationMs"`
}
```

`map_command/producer.go` — a direct analogue of `WeatherStartCommandProvider`:

```go
func PlayJukeboxCommandProvider(transactionId uuid.UUID, f field.Model, itemId uint32, playerName string, durationMs uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.PlayJukeboxCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypePlayJukebox,
		Body: mapKafka.PlayJukeboxCommandBody{
			ItemId:     itemId,
			PlayerName: playerName,
			DurationMs: durationMs,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`map_command/processor.go` — add to the `Processor` interface and implement:

```go
	PlayJukebox(transactionId uuid.UUID, f field.Model, itemId uint32, playerName string, durationMs uint32) error
```

```go
func (p *ProcessorImpl) PlayJukebox(transactionId uuid.UUID, f field.Model, itemId uint32, playerName string, durationMs uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(PlayJukeboxCommandProvider(transactionId, f, itemId, playerName, durationMs))
}
```

`saga/model.go` — three edits, each immediately after its `FieldEffectWeather` neighbour:

```go
	PlayJukebox = sharedsaga.PlayJukebox
```
```go
	PlayJukeboxPayload                  = sharedsaga.PlayJukeboxPayload
```
```go
	case PlayJukebox:
		var payload PlayJukeboxPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
```

`saga/event_acceptance.go` — in the fire-and-forget block, after `sharedsaga.FieldEffectWeather: {},`:

```go
	sharedsaga.PlayJukebox:                {},
```

`saga/handler.go` — interface method after `handleFieldEffectWeather`:

```go
	handlePlayJukebox(s Saga, st Step[any]) error
```

switch case after `case FieldEffectWeather:`:

```go
	case PlayJukebox:
		return h.handlePlayJukebox, true
```

implementation immediately after `handleFieldEffectWeather`:

```go
// handlePlayJukebox handles the PlayJukebox action.
// Produces a PLAY_JUKEBOX command to COMMAND_TOPIC_MAP. DurationMs is passed
// through in MILLISECONDS -- unlike FieldEffectWeather, whose payload carries
// seconds. The value is the client's own IWzSound::length; atlas-maps caps it.
func (h *HandlerImpl) handlePlayJukebox(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(PlayJukeboxPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	h.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"map_id":         payload.MapId,
		"item_id":        payload.ItemId,
	}).Debug("Starting jukebox")

	f := field.NewBuilder(payload.WorldId, payload.ChannelId, payload.MapId).
		SetInstance(payload.Instance).
		Build()

	err := h.mapCommandP.PlayJukebox(s.TransactionId(), f, payload.ItemId, payload.PlayerName, payload.DurationMs)
	if err != nil {
		h.logActionError(s, st, err, "Unable to start jukebox.")
		return err
	}

	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)
	return nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator
git commit -m "feat(saga-orchestrator): emit PLAY_JUKEBOX map command for the play_jukebox action"
```

---

## Task 4: atlas-maps jukebox registry, processor, and REST model

### Files

- `services/atlas-maps/atlas.com/maps/kafka/message/map/command.go` — add `CommandTypePlayJukebox` and `PlayJukeboxCommandBody`
- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go` — add the `JUKEBOX_START` / `JUKEBOX_END` status-event types and bodies
- `services/atlas-maps/atlas.com/maps/map/jukebox/registry.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/jukebox/processor.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/jukebox/rest.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/jukebox/registry_test.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/weather/registry.go` — read-only; the structural template
- `services/atlas-maps/atlas.com/maps/map/weather/processor.go` — read-only; the structural template
- `services/atlas-maps/atlas.com/maps/map/weather/rest.go` — read-only; the structural template

Module root for `go build`/`go test`: `services/atlas-maps/atlas.com/maps`.

Patterns to copy: `map/weather/registry.go:1-84` (whole file, `Message` field → `PlayerName`), `map/weather/processor.go:1-45`, `map/weather/rest.go:1-30`.

### Interfaces

- Consumes from Task 3: the `PLAY_JUKEBOX` command JSON shape.
- Produces, consumed by Task 5:
  - `jukebox.JukeboxEntry{ItemId uint32; PlayerName string; ExpiresAt time.Time}`
  - `jukebox.FieldKey{Tenant tenant.Model; Field field.Model}`
  - `jukebox.ExpiredEntry{Key FieldKey; Entry JukeboxEntry}`
  - `jukebox.NewProcessor(l, ctx) Processor` with `Start(f field.Model, itemId uint32, playerName string, duration time.Duration)` and `GetActive(f field.Model) (JukeboxEntry, bool)`
  - package funcs `jukebox.GetExpired() []ExpiredEntry` and `jukebox.DeleteEntry(key FieldKey)`
  - `jukebox.RestModel{Id string; ItemId uint32; PlayerName string}` (`GetName() == "jukebox"`) and `jukebox.Transform(e JukeboxEntry) (RestModel, error)`
  - `mapKafka.CommandTypePlayJukebox`, `mapKafka.PlayJukeboxCommandBody`, `mapKafka.EventTopicMapStatusTypeJukeboxStart`, `mapKafka.EventTopicMapStatusTypeJukeboxEnd`, `mapKafka.JukeboxStart`, `mapKafka.JukeboxEnd`

- [ ] **Step 1: Write the failing test**

Create `map/jukebox/registry_test.go` (`package jukebox`). Four test functions covering the registry contract the design's §7 table names. Build tenants with `tenant.Create(uuid.New(), "GMS", 83, 1)` and fields with `field.NewBuilder(0, 1, 100000000).Build()` — the same construction `tasks/weather_test.go:31-33` uses.

Note the registry is a package singleton, so each test must delete the keys it created (`DeleteEntry`) at the end, or use distinct tenants per test. Prefer distinct tenants per test — one `tenant.Create(uuid.New(), ...)` per test gives a disjoint key space with no cleanup coupling.

| test func | setup | assertion |
|---|---|---|
| `TestJukeboxStartThenGetActive` | `NewProcessor(l, ctx).Start(f, 5100000, "Chronicle", time.Minute)` | `GetActive(f)` returns `ok == true`, `ItemId == 5100000`, `PlayerName == "Chronicle"`, and `ExpiresAt.After(time.Now())` |
| `TestJukeboxStartReplacesActiveEntry` | `Start(f, 5100000, "Chronicle", time.Hour)` then `Start(f, 5100001, "Other", time.Minute)` | `GetActive(f)` returns exactly one entry with `ItemId == 5100001`, `PlayerName == "Other"`, and `ExpiresAt.Before(time.Now().Add(2*time.Minute))` — proving the replaced entry's one-hour expiry is gone |
| `TestJukeboxGetExpiredReturnsOnlyExpired` | `Start(f, 5100000, "Chronicle", -time.Second)` (already expired) on tenant A; `Start(f, 5100001, "Other", time.Hour)` on tenant B | `GetExpired()` contains the tenant-A key with `Entry.ItemId == 5100000` and does **not** contain the tenant-B key |
| `TestJukeboxIsTenantIsolated` | two tenants A and B, same `f`; `Start` only under A's ctx | `GetActive(f)` under B's ctx returns `ok == false` |

`GetActive(f)` on a field with nothing playing must return `ok == false` — assert that as the first line of `TestJukeboxIsTenantIsolated` before the `Start` call.

- [ ] **Step 2: Run the test to verify it fails**

```
cd services/atlas-maps/atlas.com/maps && go test ./map/jukebox/... -v
```

Expected: FAIL — the package does not exist.

- [ ] **Step 3: Implement**

`kafka/message/map/command.go` — add to the const block and a new body type:

```go
	CommandTypePlayJukebox  = "PLAY_JUKEBOX"
```

```go
type PlayJukeboxCommandBody struct {
	ItemId     uint32 `json:"itemId"`
	PlayerName string `json:"playerName"`
	DurationMs uint32 `json:"durationMs"`
}
```

`kafka/message/map/kafka.go` — add to the const block and two new body types:

```go
	EventTopicMapStatusTypeJukeboxStart    = "JUKEBOX_START"
	EventTopicMapStatusTypeJukeboxEnd      = "JUKEBOX_END"
```

```go
type JukeboxStart struct {
	ItemId     uint32 `json:"itemId"`
	PlayerName string `json:"playerName"`
}

type JukeboxEnd struct {
	ItemId uint32 `json:"itemId"`
}
```

`map/jukebox/registry.go` — `package jukebox`, a line-for-line transcription of `map/weather/registry.go` with `WeatherEntry` → `JukeboxEntry` and the `Message string` field → `PlayerName string`. Same `FieldKey`, `Registry`, `registry`/`once` singleton, `getRegistry`, `Set`, `Get`, `Delete`, `ExpiredEntry`, `GetExpired` (method and package func), `DeleteEntry`.

`map/jukebox/processor.go` — transcription of `map/weather/processor.go`:

```go
type Processor interface {
	Start(f field.Model, itemId uint32, playerName string, duration time.Duration)
	GetActive(f field.Model) (JukeboxEntry, bool)
}
```

`Start` builds `JukeboxEntry{ItemId: itemId, PlayerName: playerName, ExpiresAt: time.Now().Add(duration)}`, calls `getRegistry().Set(key, entry)`, and logs:

```go
	p.l.Debugf("Jukebox started in map [%d] instance [%s] with item [%d] by [%s] for [%s].", f.MapId(), f.Instance(), itemId, playerName, duration)
```

`map/jukebox/rest.go` — transcription of `map/weather/rest.go`:

```go
type RestModel struct {
	Id         string `json:"-"`
	ItemId     uint32 `json:"itemId"`
	PlayerName string `json:"playerName"`
}
```

`GetName()` returns `"jukebox"`; `Transform` sets `Id: strconv.Itoa(int(e.ItemId))`.

- [ ] **Step 4: Run the tests to verify they pass**

```
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./map/jukebox/... ./kafka/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/jukebox services/atlas-maps/atlas.com/maps/kafka/message/map
git commit -m "feat(maps): add jukebox field registry, processor, and REST model"
```

---

## Task 5: atlas-maps jukebox command consumer, expiry sweep, and REST resource

### Files

- `services/atlas-maps/atlas.com/maps/map/jukebox/producer.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/jukebox/resource.go` — **new file**
- `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go` — add `handlePlayJukeboxCommand` and register it in `InitHandlers`
- `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer_test.go` — **new file**; the duration-cap / wrong-type tests
- `services/atlas-maps/atlas.com/maps/kafka/consumer/map/testmain_test.go` — **new file**; `producertest.InstallNoop()`
- `services/atlas-maps/atlas.com/maps/tasks/jukebox.go` — **new file**
- `services/atlas-maps/atlas.com/maps/tasks/jukebox_test.go` — **new file**
- `services/atlas-maps/atlas.com/maps/main.go` — register the task (next to `tasks.NewWeather`, line ~132) and the route initializer (next to `weather.InitResource`, line ~144)
- `services/atlas-maps/atlas.com/maps/map/weather/producer.go` — read-only; template
- `services/atlas-maps/atlas.com/maps/map/weather/resource.go` — read-only; template
- `services/atlas-maps/atlas.com/maps/tasks/weather.go` — read-only; template
- `services/atlas-maps/atlas.com/maps/tasks/weather_test.go` — read-only; template
- `services/atlas-maps/atlas.com/maps/kafka/consumer/character/testmain_test.go` — read-only; the `producertest.InstallNoop()` template

Module root for `go build`/`go test`: `services/atlas-maps/atlas.com/maps`.

Patterns to copy: `map/weather/producer.go:14-45`, `map/weather/resource.go:19-57`, `kafka/consumer/map/consumer.go:31-65`, `tasks/weather.go:18-66`, `tasks/weather_test.go:17-51`.

### Interfaces

- Consumes from Task 4: everything that task's Interfaces block lists.
- Produces, consumed by Task 8 (over Kafka) and Task 6 (over REST):
  - `JUKEBOX_START` on `EVENT_TOPIC_MAP_STATUS`, body `{"itemId":<uint32>,"playerName":<string>}`
  - `JUKEBOX_END` on `EVENT_TOPIC_MAP_STATUS`, body `{"itemId":<uint32>}`
  - `GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/jukebox` → `200` with a JSON:API `jukebox` resource, or `404` with no body

- [ ] **Step 1: Write the failing test**

Create `tasks/jukebox_test.go` (`package tasks`), the direct analogue of `tasks/weather_test.go`. Declare a test-local context key type `jukeboxEnvMarkerKey string` (do **not** import `libs/atlas-env` — `tasks` sits outside env-domain-guard's permitted import list; `tasks/weather_test.go:17-21` records why).

```go
func TestProcessExpiredJukebox_AppliesEnvContextToEmit(t *testing.T) { ... }
func TestProcessExpiredJukebox_DeletesTheEntry(t *testing.T) { ... }
```

| test | setup | assertion |
|---|---|---|
| `TestProcessExpiredJukebox_AppliesEnvContextToEmit` | one `jukebox.ExpiredEntry{Key: {Tenant: ten, Field: f}, Entry: jukebox.JukeboxEntry{ItemId: 5100000, PlayerName: "Chronicle", ExpiresAt: time.Now().Add(-time.Second)}}`; `envContext` stamps `jukeboxEnvMarkerKey("marker") = "stamped"`; spy `emit` records `ctx.Value(...)` | the recorded marker equals `"stamped"` — an identity `envContext` would still pass if the wiring were dropped, so this must use a *stamping* one |
| `TestProcessExpiredJukebox_DeletesTheEntry` | `jukebox.NewProcessor(l, tenant.WithContext(ctx, ten)).Start(f, 5100000, "Chronicle", -time.Second)`, then `processExpiredJukebox(l, ctx, jukebox.GetExpired(), noopEmit, identityEnvContext)` | `jukebox.NewProcessor(l, tctx).GetActive(f)` returns `ok == false` afterwards |

Both call `processExpiredJukebox(l, context.Background(), entries, emit, envContext)` directly — the sweep's pure core, never `Run()`.

Also create `kafka/consumer/map/testmain_test.go` (`package _map`), copied verbatim from `kafka/consumer/character/testmain_test.go` with the package name changed — `producertest.InstallNoop()` in `TestMain` so the handler's Kafka emit is inert. Then create `kafka/consumer/map/consumer_test.go` (`package _map`) covering the command handler's three behaviours. Build the command with `mapKafka.Command[mapKafka.PlayJukeboxCommandBody]{TransactionId: uuid.New(), WorldId: 0, ChannelId: 1, MapId: 100000000, Instance: uuid.Nil, Type: ..., Body: ...}`, invoke `handlePlayJukeboxCommand()(l, ctx, cmd)`, and read the result out of the registry with `jukebox.NewProcessor(l, ctx).GetActive(f)`. Use a fresh `tenant.Create(uuid.New(), "GMS", 83, 1)` per test so the registry singleton's key spaces are disjoint.

| test func | `Type` | `Body.DurationMs` | assertion |
|---|---|---|---|
| `TestHandlePlayJukeboxCommand_StartsWithTheCommandDuration` | `mapKafka.CommandTypePlayJukebox` | `45000` | `GetActive(f)` → `ok == true`, `ItemId == 5100000`, `PlayerName == "Chronicle"`, and `ExpiresAt` between `time.Now().Add(40*time.Second)` and `time.Now().Add(50*time.Second)` |
| `TestHandlePlayJukeboxCommand_CapsExcessiveDuration` | `mapKafka.CommandTypePlayJukebox` | `3600000` (one hour) | `ExpiresAt` is **not** after `time.Now().Add(maxJukeboxDuration + time.Second)` — i.e. capped at ten minutes, not an hour |
| `TestHandlePlayJukeboxCommand_IgnoresOtherCommandTypes` | `mapKafka.CommandTypeWeatherStart` | `45000` | `GetActive(f)` → `ok == false` |

- [ ] **Step 2: Run the test to verify it fails**

```
cd services/atlas-maps/atlas.com/maps && go test ./tasks/... ./kafka/consumer/map/... -run Jukebox -v
```

Expected: FAIL — `undefined: processExpiredJukebox`, `undefined: handlePlayJukeboxCommand`.

- [ ] **Step 3: Implement**

`map/jukebox/producer.go` — transcription of `map/weather/producer.go`:

```go
func JukeboxStartEventProvider(transactionId uuid.UUID, f field.Model, itemId uint32, playerName string) model.Provider[[]kafka.Message]
func JukeboxEndEventProvider(transactionId uuid.UUID, f field.Model, itemId uint32) model.Provider[[]kafka.Message]
```

using `mapKafka.StatusEvent[mapKafka.JukeboxStart]` / `[mapKafka.JukeboxEnd]`, `Type: mapKafka.EventTopicMapStatusTypeJukeboxStart` / `...JukeboxEnd`, key `producer.CreateKey(int(f.MapId()))`, and the same envelope-from-field assignments.

`map/jukebox/resource.go` — transcription of `map/weather/resource.go` with:

```go
const (
	getJukeboxInMap = "get_jukebox_in_map"
)
```

route `"/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/jukebox"`, handler `handleGetJukeboxInMap`, `w.WriteHeader(http.StatusNotFound)` and return when `GetActive` reports `!ok`.

`kafka/consumer/map/consumer.go` — register the new handler in `InitHandlers` on the same topic, immediately after the weather registration:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handlePlayJukeboxCommand()))); err != nil {
			return err
		}
```

and add the handler, mirroring `handleWeatherStartCommand`:

```go
// maxJukeboxDuration bounds a crafted or buggy PLAY_JUKEBOX command. The
// duration is the client's own IWzSound::length, so a real track is well
// under this; ten minutes is an order of magnitude above any real WZ sound
// while still preventing a field's BGM from being pinned indefinitely.
const maxJukeboxDuration = 10 * time.Minute

func handlePlayJukeboxCommand() func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.PlayJukeboxCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.PlayJukeboxCommandBody]) {
		if c.Type != mapKafka.CommandTypePlayJukebox {
			return
		}

		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
		duration := time.Duration(c.Body.DurationMs) * time.Millisecond

		if duration > maxJukeboxDuration {
			l.Warnf("Jukebox duration [%s] for map [%d] instance [%s] exceeds maximum, capping at [%s].", duration, c.MapId, c.Instance, maxJukeboxDuration)
			duration = maxJukeboxDuration
		}

		l.Debugf("Received play jukebox command for map [%d] instance [%s] item [%d] duration [%s].", c.MapId, c.Instance, c.Body.ItemId, duration)

		jukebox.NewProcessor(l, ctx).Start(f, c.Body.ItemId, c.Body.PlayerName, duration)

		err := producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(jukebox.JukeboxStartEventProvider(c.TransactionId, f, c.Body.ItemId, c.Body.PlayerName))
		if err != nil {
			l.WithError(err).Errorf("Unable to produce jukebox start event for map [%d] instance [%s].", c.MapId, c.Instance)
		}
	}
}
```

Move `const maxJukeboxDuration` to package scope (as written above) rather than inside the closure, so the tasks test and any future caller can reference it; note that `maxWeatherDuration` is function-scoped and stays as it is.

`tasks/jukebox.go` — transcription of `tasks/weather.go`:

```go
const JukeboxTask = "jukebox_task"

type Jukebox struct {
	l          logrus.FieldLogger
	interval   time.Duration
	envContext func(context.Context) context.Context
}

func NewJukebox(l logrus.FieldLogger, interval time.Duration, envContext func(context.Context) context.Context) *Jukebox
func (w *Jukebox) Run()
func emitJukeboxEnd(l logrus.FieldLogger) func(ctx context.Context, e jukebox.ExpiredEntry) error
func processExpiredJukebox(l logrus.FieldLogger, ctx context.Context, expired []jukebox.ExpiredEntry, emit func(ctx context.Context, e jukebox.ExpiredEntry) error, envContext func(context.Context) context.Context)
func (w *Jukebox) SleepTime() time.Duration
```

`Run()` opens the `otel` span named `JukeboxTask` and calls `processExpiredJukebox(w.l, ctx, jukebox.GetExpired(), emitJukeboxEnd(w.l), w.envContext)`. `emitJukeboxEnd` produces `jukebox.JukeboxEndEventProvider(uuid.New(), f, e.Entry.ItemId)` onto `mapKafka.EnvEventTopicMapStatus`. `processExpiredJukebox` applies `envContext(tenant.WithContext(ctx, e.Key.Tenant))` before each emit and calls `jukebox.DeleteEntry(e.Key)` after — carry over `tasks/weather.go`'s comment explaining why the env context must be stamped.

`main.go` — two additions:

```go
	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(tasks.NewJukebox(l, time.Second, envContext))
	})
```

```go
		AddRouteInitializer(jukebox.InitResource(GetServer())).
```

plus the `"atlas-maps/map/jukebox"` import.

- [ ] **Step 4: Run the tests to verify they pass**

```
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps
git commit -m "feat(maps): consume PLAY_JUKEBOX, expire songs, and serve the jukebox resource"
```

---

## Task 6: atlas-channel jukebox REST client

### Files

- `services/atlas-channel/atlas.com/channel/jukebox/rest.go` — **new file**
- `services/atlas-channel/atlas.com/channel/jukebox/requests.go` — **new file**
- `services/atlas-channel/atlas.com/channel/jukebox/processor.go` — **new file**
- `services/atlas-channel/atlas.com/channel/jukebox/mock/processor.go` — **new file**
- `services/atlas-channel/atlas.com/channel/jukebox/requests_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/weather/rest.go` — read-only; template
- `services/atlas-channel/atlas.com/channel/weather/requests.go` — read-only; template
- `services/atlas-channel/atlas.com/channel/weather/processor.go` — read-only; template
- `services/atlas-channel/atlas.com/channel/weather/mock/processor.go` — read-only; template

Module root for `go build`/`go test`: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `weather/rest.go:1-20`, `weather/requests.go:11-26`, `weather/processor.go:12-33`, `weather/mock/processor.go:1-20`.

### Interfaces

- Consumes from Task 5: the `GET .../instances/{instanceId}/jukebox` endpoint and its `jukebox` JSON:API resource shape (`itemId`, `playerName`).
- Produces, consumed by Task 8:
  - `jukebox.RestModel{Id string; ItemId uint32; PlayerName string}` (`GetName() == "jukebox"`)
  - `jukebox.NewProcessor(l, ctx) Processor` with `GetActive(f field.Model) (RestModel, error)`
  - `mock.ProcessorMock{GetActiveFunc func(f field.Model) (jukebox.RestModel, error)}`

- [ ] **Step 1: Write the failing test**

Create `jukebox/requests_test.go` (`package jukebox`). Stand up an `httptest` server, point `MAPS_SERVICE_URL` at it with `t.Setenv`, and assert both the URL the client builds and the model it decodes.

| test func | server behaviour | assertion |
|---|---|---|
| `TestGetActiveDecodesTheJukeboxResource` | responds `200` with `{"data":{"type":"jukebox","id":"5100000","attributes":{"itemId":5100000,"playerName":"Chronicle"}}}` and `Content-Type: application/vnd.api+json` | returned `RestModel.ItemId == 5100000`, `PlayerName == "Chronicle"`, no error |
| `TestGetActiveRequestsTheInstanceScopedPath` | records `r.URL.Path`, responds as above | the recorded path ends with `/worlds/0/channels/1/maps/100000000/instances/<instanceId>/jukebox`, where `<instanceId>` is the exact `uuid` string the field was built with |
| `TestGetActiveReturnsErrorWhenNothingPlaying` | responds `404` with no body | `GetActive` returns a non-nil error (the caller in Task 8 treats any error as "no song") |

Build the field with `field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(instanceId).Build()` and the context with `tenant.WithContext(context.Background(), ten)` from `tenant.Create(uuid.New(), "GMS", 83, 1)`.

- [ ] **Step 2: Run the test to verify it fails**

```
cd services/atlas-channel/atlas.com/channel && go test ./jukebox/... -v
```

Expected: FAIL — the package does not exist.

- [ ] **Step 3: Implement**

`jukebox/rest.go`:

```go
package jukebox

type RestModel struct {
	Id         string `json:"-"`
	ItemId     uint32 `json:"itemId"`
	PlayerName string `json:"playerName"`
}

func (m RestModel) GetID() string   { return m.Id }
func (m RestModel) GetName() string { return "jukebox" }

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}
```

`jukebox/requests.go` — transcription of `weather/requests.go` with:

```go
const (
	mapInstanceResource        = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapInstanceJukeboxResource = mapInstanceResource + "/jukebox"
)

func requestJukeboxInMap(ctx context.Context, f field.Model) requests.Request[RestModel]
```

`getBaseRequest` resolves `requests.RootUrlFor(ctx, "MAPS")`, exactly as weather's does.

`jukebox/processor.go` — transcription of `weather/processor.go`:

```go
type Processor interface {
	GetActive(f field.Model) (RestModel, error)
}
```

with `ProcessorImpl{l, ctx}`, `NewProcessor`, `var _ Processor = (*ProcessorImpl)(nil)`, `GetActive` calling `requests.Provider[RestModel, RestModel](p.l, p.ctx)(requestJukeboxInMap(p.ctx, f), Extract)()`, and `Extract(m RestModel) (RestModel, error)` returning `m, nil`.

`jukebox/mock/processor.go` — transcription of `weather/mock/processor.go` with `jukebox.RestModel` in place of `weather.RestModel`.

- [ ] **Step 4: Run the tests to verify they pass**

```
cd services/atlas-channel/atlas.com/channel && go build ./jukebox/... && go test ./jukebox/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/jukebox
git commit -m "feat(channel): add jukebox REST client for the atlas-maps jukebox resource"
```

---

## Task 7: atlas-channel type-20 item-use arm

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — add `CashSlotItemTypeSongPlayer` to the const block (line ~959) and the arm (immediately after the `CashSlotItemTypeFieldEffect` arm, which ends at line ~181)
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_jukebox_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/saga/model.go` — add the `PlayJukebox` const alias (line ~112, next to `FieldEffectWeather`) and the `PlayJukeboxPayload` type alias (line ~28, next to `FieldEffectWeatherPayload`)
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go` — read-only; `installCashItemInSlotSeam`, `newCashItemUseTestSession`, `cashItemUsePrefix`
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_kite_test.go` — read-only; `newKiteCharacterServer` (the character-name fixture this arm reuses)
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_item_gachapon_test.go` — read-only; `installCapturingProducer`
- `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go` — read-only; `gaugeProducerRecorder`

Module root for `go build`/`go test`: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `character_cash_item_use.go:136-181` (the field-effect arm's two-step saga), `character_cash_item_use.go:107-134` (the kite arm's character-name lookup and its failure handling), `character_cash_item_use_karma_test.go:336-398` (saga-emission assertions), `character_cash_item_use_kite_test.go:57-63` (building the sub-body wire bytes with `response.NewWriter`).

### Interfaces

- Consumes from Task 1: `cashsb.NewItemUseSongPlayer(updateTimeFirst)`, `sp.SoundLengthMs()`, `sp.UpdateTime()`.
- Consumes from Task 2 (via the channel's re-export): `saga.PlayJukebox`, `saga.PlayJukeboxPayload`.
- Produces: a `saga.FieldEffectUse` saga on `COMMAND_TOPIC_SAGA` with steps `["consume_song_player_item" → saga.DestroyAsset, "play_jukebox" → saga.PlayJukebox]`.

- [ ] **Step 1: Write the failing test**

Create `character_cash_item_use_jukebox_test.go` (`package handler`).

Helper — the type-20 request builder, modelled on `kiteUseRequest`:

```go
// jukeboxUseRequest builds the real wire payload for a category-510 (song
// player) cash item use on GMS v83: the common cashsb.ItemUse prefix (slot,
// itemId -- no leading updateTime at v83) followed by the type-20 sub-body,
// which per ItemUseSongPlayer is one int32 sound length plus a trailing
// updateTime on GMS <= v84. Built with the same response.Writer primitives
// ItemUseSongPlayer.Encode uses internally, because its fields are private
// with no public constructor argument for them.
func jukeboxUseRequest(l logrus.FieldLogger, source int16, itemId uint32, soundLengthMs uint32) *request.Reader {
	w := response.NewWriter(l)
	w.WriteInt(soundLengthMs)
	w.WriteInt(0) // trailing updateTime, GMS v83 (updateTimeFirst == false)
	body := append(cashItemUsePrefix(source, itemId), w.Bytes()...)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	return &reader
}
```

Fixed values shared by every case: `const jukeboxItemId = uint32(5100000)`, `const jukeboxSlot = int16(4)`, `const jukeboxCharacterId = uint32(777)`, `const jukeboxSoundLengthMs = uint32(45000)`.

Four test functions:

| test func | setup | expected |
|---|---|---|
| `TestJukeboxArmSuccessCreatesTwoStepSaga` | `installCashItemInSlotSeam(t, jukeboxSlot, jukeboxItemId)`; `newKiteCharacterServer(t, jukeboxCharacterId, "Chronicle", 0, 0)`; `installCapturingProducer()`; `newCashItemUseTestSession(t, jukeboxCharacterId)`; request with `soundLengthMs = 45000` | exactly one message on `sagaMsg.EnvCommandTopic`; decoded `saga.Saga` has `SagaType == saga.FieldEffectUse`, `InitiatedBy == "CASH_ITEM_USE"`, `len(Steps) == 2`; step 0 `Action == saga.DestroyAsset` with `DestroyAssetPayload{CharacterId: 777, TemplateId: 5100000, Quantity: 1, RemoveAll: false}`; step 1 `Action == saga.PlayJukebox` with `PlayJukeboxPayload{WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil, ItemId: 5100000, PlayerName: "Chronicle", DurationMs: 45000}`; `rec.calls == 0` (no announce on the success path) |
| `TestJukeboxArmRejectsSlotTemplateMismatch` | `installCashItemInSlotSeam(t, jukeboxSlot, 5100001)` — the slot holds a *different* template than the request claims | zero messages on every captured topic; `rec.calls == 0` |
| `TestJukeboxArmRejectsZeroSoundLength` | as the success case but `soundLengthMs = 0` | zero messages on every captured topic; `rec.calls == 0` |
| `TestJukeboxArmRejectsUnresolvableCharacter` | as the success case but the character server responds `404` (`httptest` handler writing `http.StatusNotFound`, `t.Setenv("CHARACTERS_SERVICE_URL", ...)`) | zero messages on every captured topic; `rec.calls == 0` |

Each test invokes `CharacterCashItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, r, map[string]interface{}{})` with `rec := &gaugeProducerRecorder{}`. Step-1 payload is asserted after a type switch on `got.Steps[1].Payload.(saga.PlayJukeboxPayload)` — the same `json.Unmarshal` into `saga.Saga` then typed-payload assertion that `TestKarmaArmSuccessCreatesTwoStepSaga` uses.

For the "zero messages" assertions, iterate every topic in `*captured` and require `len(msgs) == 0`, as `character_cash_item_use_expiration_extender_test.go:223-228` does.

- [ ] **Step 2: Run the tests to verify they fail**

```
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run Jukebox -v
```

Expected: FAIL — `undefined: saga.PlayJukebox` / the request falls through to the warn-and-drop and emits no saga.

- [ ] **Step 3: Implement**

`saga/model.go` — two aliases:

```go
	PlayJukeboxPayload           = sharedsaga.PlayJukeboxPayload
```
```go
	PlayJukebox           = sharedsaga.PlayJukebox
```

`character_cash_item_use.go` — add to the `CashSlotItemType` const block:

```go
	// CashSlotItemTypeSongPlayer is classification 510 (jukebox). Unlike the
	// 530 morph coupon, which had to be classification-keyed because its type
	// byte moves across versions, 20 is stable: get_cashslot_item_type maps
	// 510 -> 20 on every version examined (GMS v95.0 @0x488c70, GMS v83) and
	// no other classification yields 20 in that function.
	CashSlotItemTypeSongPlayer = CashSlotItemType(20)
```

and the arm, immediately after the `CashSlotItemTypeFieldEffect` arm's closing brace:

```go
		if it == CashSlotItemTypeSongPlayer {
			sp := cashsb.NewItemUseSongPlayer(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			if !updateTimeFirst {
				updateTime = sp.UpdateTime()
			}

			// The client sends the song's own length (IWzSound::length) and
			// resolves the BGM itself from the item's WZ info/path node
			// (CMapLoadable::PlayNextMusic). The server carries neither the BGM
			// name nor a duration constant -- only the client's value, which
			// atlas-maps caps.
			//
			// A zero length is a broken or spoofed client: the song would end
			// the instant it started. Reject before consuming anything.
			if sp.SoundLengthMs() == 0 {
				l.Warnf("Character [%d] sent song player [%d] use with a zero sound length; ignoring without consuming.", s.CharacterId(), itemId)
				return
			}

			// PlayJukebox carries the starting player's name, which is not on
			// the wire -- resolve it from server-side character state, the same
			// source the kite arm uses.
			c, cerr := character2.NewProcessor(l, ctx).GetById()(s.CharacterId())
			if cerr != nil {
				l.WithError(cerr).Debugf("Unable to resolve character [%d] for jukebox use.", s.CharacterId())
				return
			}

			// No EnableActions: the non-silent INVENTORY_OPERATION emitted by
			// the consume commit already clears the client's exclusive-request
			// lock -- the same reasoning recorded on the field-effect and
			// morph-coupon arms.
			transactionId := uuid.New()
			now := time.Now()
			f := s.Field()
			steps := []saga.Step{
				{
					StepId: "consume_song_player_item",
					Status: saga.Pending,
					Action: saga.DestroyAsset,
					Payload: saga.DestroyAssetPayload{
						CharacterId: s.CharacterId(),
						TemplateId:  uint32(itemId),
						Quantity:    1,
						RemoveAll:   false,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					StepId: "play_jukebox",
					Status: saga.Pending,
					Action: saga.PlayJukebox,
					Payload: saga.PlayJukeboxPayload{
						WorldId:    f.WorldId(),
						ChannelId:  f.ChannelId(),
						MapId:      f.MapId(),
						Instance:   f.Instance(),
						ItemId:     uint32(itemId),
						PlayerName: c.Name(),
						DurationMs: sp.SoundLengthMs(),
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			}
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.FieldEffectUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps:         steps,
			})
			return
		}
```

- [ ] **Step 4: Run the tests to verify they pass**

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/ ./saga/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler services/atlas-channel/atlas.com/channel/saga
git commit -m "feat(channel): route cash slot type 20 song player use into a play_jukebox saga"
```

---

## Task 8: atlas-channel jukebox broadcast and map-enter replay

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go` — add the `JUKEBOX_START` / `JUKEBOX_END` type constants and their bodies
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` — add `handleStatusEventJukeboxStart` / `handleStatusEventJukeboxEnd` (next to the weather pair at lines ~974 and ~1062), register both in `InitHandlers` (line ~97-105), and add the map-enter replay `routine.Go` block (next to the weather block at line ~346)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go` — add the broadcast and replay tests
- `libs/atlas-packet/field/clientbound/play_jukebox.go` — **read-only, must not change**; `PlayJukeboxWriter`, `NewPlayJukebox`
- `services/atlas-channel/atlas.com/channel/main.go` — read-only; `fieldcb.PlayJukeboxWriter` is already registered at line ~842

Module root for `go build`/`go test`: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `kafka/consumer/map/consumer.go:974-995` and `:1062-1080` (the weather status handlers), `:754-761` (the `doorAnnounce` seam), `:346-361` (the weather map-enter replay block), `kafka/consumer/map/consumer_test.go:28-63` (`newTestCtx`, `newTestField`, `addFieldSession`), `:352-383` (`stubDoorAnnounceForVisuals`, the announce-stub pattern).

### Interfaces

- Consumes from Task 5: the `JUKEBOX_START` / `JUKEBOX_END` status events.
- Consumes from Task 6: `jukebox.NewProcessor(l, ctx).GetActive(f)`.
- Produces: `PlayJukebox` packets on the wire. Nothing downstream consumes this task.

- [ ] **Step 1: Write the failing tests**

Add to `kafka/consumer/map/consumer_test.go`. Sessions in this package are built with a `nil` conn (`addFieldSession`), so the handlers must announce through the package's existing `doorAnnounce` seam rather than calling `session.Announce` directly — the stub is what makes them testable. Reuse a local stub in the shape of `stubDoorAnnounceForVisuals`, recording `(writerName, encodedBytes)` per call:

```go
// stubDoorAnnounceForJukebox records every announce the jukebox handlers make,
// capturing the encoded PlayJukebox body so the exact item id on the wire can
// be asserted -- the -1 stop signal is a correctness requirement, not a
// convention (design §3.2).
func stubDoorAnnounceForJukebox(t *testing.T) (restore func(), calls *[]jukeboxAnnounce)
```

where `type jukeboxAnnounce struct { Writer string; Body []byte }`.

Test table:

| test func | setup | assertion |
|---|---|---|
| `TestHandleStatusEventJukeboxStart_BroadcastsToEverySessionInField` | two sessions added to `f` via `addFieldSession`; event `StatusEvent[JukeboxStart]{Type: EventTopicMapStatusTypeJukeboxStart, WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil, Body: {ItemId: 5100000, PlayerName: "Chronicle"}}` | exactly 2 announces, both `Writer == fieldcb.PlayJukeboxWriter`; each `Body` decodes (via `fieldcb.PlayJukebox{}.Decode`) to `ItemId() == 5100000` and `PlayerName() == "Chronicle"` |
| `TestHandleStatusEventJukeboxStart_IgnoresOtherEventTypes` | same sessions; event with `Type: "SOMETHING_ELSE"` | zero announces |
| `TestHandleStatusEventJukeboxStart_IgnoresOtherWorldChannel` | same sessions; event with `WorldId: 1` (a world the `server.Model` under test is not) | zero announces |
| `TestHandleStatusEventJukeboxEnd_BroadcastsExactlyMinusOne` | two sessions; event `StatusEvent[JukeboxEnd]{Type: EventTopicMapStatusTypeJukeboxEnd, ..., Body: {ItemId: 5100000}}` | exactly 2 announces, both `fieldcb.PlayJukeboxWriter`; each `Body` decodes to `ItemId() == -1` **and** `PlayerName() == ""` — decoding the raw first four bytes as little-endian int32 must yield exactly `-1` (`ff ff ff ff`), never any other negative value |
| `TestHandleStatusEventJukeboxEnd_IgnoresOtherWorldChannel` | event with `ChannelId: 3` | zero announces |
| `TestAnnounceActiveJukebox_ReplaysToTheEnteringSession` | `jukebox` REST fixture server returning the active song (`httptest` + `t.Setenv("MAPS_SERVICE_URL", ...)`); one entering session | exactly 1 announce, `fieldcb.PlayJukeboxWriter`, decoding to `ItemId() == 5100000` / `PlayerName() == "Chronicle"` |
| `TestAnnounceActiveJukebox_FailsOpenWhenMapsUnreachable` | fixture server returning `404` | zero announces and no panic — the map entry must not be blocked |

Build the `server.Model` for the two handler-constructor arguments the same way the weather handlers are constructed in production (`sc server.Model, wp writer.Producer`); pass `nil` for `wp` since the `doorAnnounce` stub never reaches it.

To make the replay path directly testable, extract it into a named function rather than leaving it inline in the `routine.Go` block:

```go
func announceActiveJukebox(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, s session.Model)
```

— the same extraction `announceActiveVisuals` already represents in this file, and the reason its own tests can exist.

- [ ] **Step 2: Run the tests to verify they fail**

```
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/map/ -run Jukebox -v
```

Expected: FAIL — `undefined: handleStatusEventJukeboxStart`.

- [ ] **Step 3: Implement**

`kafka/message/map/kafka.go` — add next to the weather pair:

```go
	EventTopicMapStatusTypeJukeboxStart    = "JUKEBOX_START"
	EventTopicMapStatusTypeJukeboxEnd      = "JUKEBOX_END"
```

```go
type JukeboxStart struct {
	ItemId     uint32 `json:"itemId"`
	PlayerName string `json:"playerName"`
}

type JukeboxEnd struct {
	ItemId uint32 `json:"itemId"`
}
```

`kafka/consumer/map/consumer.go` — the stop-signal constant:

```go
// jukeboxStopItemId is the EXACT value CMapLoadable::PlayNextMusic branches on
// to restore the map's own BGM (@0x61dab0: `if (m_nJukeBoxItemID == -1)`). Any
// other negative id falls into the else branch, fails CItemInfo::GetItemInfo,
// and returns WITHOUT clearing m_nJukeBoxItemID -- so CMapLoadable::Update
// re-enters PlayNextMusic every frame, forever. This is a correctness
// requirement, not a convention.
const jukeboxStopItemId = int32(-1)
```

the two handlers, mirroring the weather pair but announcing through `doorAnnounce`:

```go
func handleStatusEventJukeboxStart(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.JukeboxStart]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.JukeboxStart]) {
		if e.Type != _map3.EventTopicMapStatusTypeJukeboxStart {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Jukebox started in map [%d] instance [%s] with item [%d] by [%s].", e.MapId, e.Instance, e.Body.ItemId, e.Body.PlayerName)
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		// Broadcast to everyone in the field INCLUDING the user: the sending
		// client applies nothing locally (m_bJukeBoxPlaying is written only
		// inside PlayNextMusic, reachable only from OnPlayJukeBox), and its own
		// m_bJukeBoxPlaying pre-gate prevents a second send, so no double-apply
		// is possible.
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
			return doorAnnounce(l, ctx, wp, fieldcb.PlayJukeboxWriter, fieldcb.NewPlayJukebox(int32(e.Body.ItemId), e.Body.PlayerName).Encode, s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast jukebox start to map [%d] instance [%s].", e.MapId, e.Instance)
		}
	}
}

func handleStatusEventJukeboxEnd(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.JukeboxEnd]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.JukeboxEnd]) {
		if e.Type != _map3.EventTopicMapStatusTypeJukeboxEnd {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Jukebox ended in map [%d] instance [%s].", e.MapId, e.Instance)
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		// No BGM FieldEffect on the restore: that packet sets
		// m_sChangedBgmUOL, which CMapLoadable::RestoreBGM prefers over the
		// map's own music -- sending one would leave the field permanently
		// playing the jukebox track. The stop signal alone drives the restore.
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
			return doorAnnounce(l, ctx, wp, fieldcb.PlayJukeboxWriter, fieldcb.NewPlayJukebox(jukeboxStopItemId, "").Encode, s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast jukebox end to map [%d] instance [%s].", e.MapId, e.Instance)
		}
	}
}
```

the replay function and its call site:

```go
// announceActiveJukebox replays an in-progress song to a single entering
// session. Fails open: an unreachable atlas-maps costs the song, not the map
// entry.
func announceActiveJukebox(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, s session.Model) {
	jb, jerr := jukebox.NewProcessor(l, ctx).GetActive(f)
	if jerr != nil {
		return
	}
	_ = doorAnnounce(l, ctx, wp, fieldcb.PlayJukeboxWriter, fieldcb.NewPlayJukebox(int32(jb.ItemId), jb.PlayerName).Encode, s)
}
```

called from a new `routine.Go` block placed immediately after the existing weather replay block:

```go
		routine.Go(l, ctx, func(_ context.Context) {
			announceActiveJukebox(l, ctx, wp, f, s)
		})
```

and both handlers registered in `InitHandlers` after the weather-end registration, each following the existing `id, err = rf(...)` / error-check / `handles = append(...)` triple.

Add the `"atlas-channel/jukebox"` import.

- [ ] **Step 4: Run the tests to verify they pass**

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka
git commit -m "feat(channel): broadcast and replay jukebox songs on map status events"
```

---

## Final gate (controller, not the implementer)

- [ ] Run the flagless verification gate from the worktree root and confirm exit 0:

```
tools/verify.sh
```

- [ ] Confirm the writer is no longer scaffold-only:

```
grep -rn "PlayJukeboxWriter" services/
```

Expected: at least the three invoking call sites added in Task 8, in addition to the registration in `services/atlas-channel/atlas.com/channel/main.go`.

- [ ] Confirm the clientbound codec and its evidence were not touched:

```
git diff --stat main -- libs/atlas-packet/field/clientbound/play_jukebox.go docs/packets/audits
```

Expected: empty.
