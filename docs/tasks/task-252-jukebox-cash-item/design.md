# Jukebox Cash Item — Design

Version: v1
Status: Draft
Created: 2026-08-21
PRD: [prd.md](prd.md)

---

## 0. Summary of what the client already does

The design phase resolved all four PRD open questions against the client
binaries. Two of the answers materially shrink the feature relative to the PRD:

1. **The client resolves the song itself.** `CMapLoadable::PlayNextMusic` reads
   the used item's own WZ `info/path` node and hands it straight to
   `CSoundMan::PlayBGM`. The server never names the BGM. This deletes the
   entire atlas-data `info/path` workstream (PRD FR-2.1 – FR-2.4) and both
   `FieldEffect` BGM broadcasts (FR-5.1.2, FR-5.2's restore).
2. **The client tells the server the song length.** The type-20 arm of
   `CWvsContext::SendConsumeCashItemUseRequest` encodes one int32: the WZ sound's
   own `IWzSound::length`. Duration is therefore client-supplied and
   server-capped, not a server constant (PRD FR-4.2 / OQ-3).

Everything else in the PRD stands. §6 records the exact evidence; §10 lists the
PRD requirements this design supersedes so the acceptance criteria can be
amended before `/plan-task`.

---

## 1. Architecture

The feature is the weather cash item's pipeline with the BGM leg removed and a
client-supplied duration added.

```
client                  atlas-channel            saga-orch        atlas-maps
------                  -------------            ---------        ----------
USE_CASH_ITEM (type 20)
  body: int32 soundLenMs
      ────────────────►  CharacterCashItemUseHandleFunc
                         · cashItemInSlotFunc ownership check
                         · decode ItemUseSongPlayer
                         · resolve character name
                         · reject if soundLenMs == 0
                         · Saga{FieldEffectUse}:
                             1. DestroyAsset(itemId, qty 1)
                             2. PlayJukebox{field, itemId,
                                  playerName, durationMs}
                                        ─────────►  handlePlayJukebox
                                                    map_command.PlayJukebox
                                                        ─── PLAY_JUKEBOX ───►
                                                                     jukebox.Start
                                                                     (registry,
                                                                      capped TTL)
                                                        ◄── JUKEBOX_START ──
                         handleStatusEventJukeboxStart
                         · ForSessionsInMap → PlayJukebox(itemId, playerName)
      ◄────────────────
                                                             tasks.Jukebox sweep
                                                        ◄─── JUKEBOX_END ───
                         handleStatusEventJukeboxEnd
                         · ForSessionsInMap → PlayJukebox(-1, "")
      ◄────────────────
```

On map enter, the channel's `CHARACTER_ENTER` handler queries
`GET .../instances/{id}/jukebox` and announces `PlayJukebox(itemId, playerName)`
to that one session.

### Why this shape

Every hop already exists for weather and is the only path in the repo by which
a cash-item use can consume an asset *and* mutate durable field state
atomically. The saga is not ceremony: it is what guarantees the item is gone
before the song starts, and that a failed consume leaves no song playing
(PRD FR-3.4). A direct channel→maps command would consume-and-broadcast
independently and could play a song for an item that was never destroyed.

---

## 2. Component design

### 2.1 `libs/atlas-packet/cash/serverbound/item_use_song_player.go` (new)

```go
type ItemUseSongPlayer struct {
    soundLengthMs   uint32
    updateTime      uint32
    updateTimeFirst bool
}
```

Wire order, per §6.1: `Encode4(soundLengthMs)`, then — only when
`updateTimeFirst == false` (GMS ≤ v84) — the shared send tail's
`Encode4(updateTime)`. Both `Encode` and `Decode`, constructed as
`NewItemUseSongPlayer(updateTimeFirst)`, exactly like `ItemUseFieldEffect` and
`ItemUseMorphCoupon`. Carries the `packet-audit:fname
CWvsContext::SendConsumeCashItemUseRequest` marker and a doc comment citing the
two addresses in §6.1.

The clientbound `PlayJukebox` codec is **unchanged**. Its evidence records for
`gms_v95` and `jms_v185` stay pinned as-is.

### 2.2 `atlas-channel` — the type-20 arm

`CashSlotItemTypeSongPlayer = CashSlotItemType(20)` joins the constant block.
Gated on the type byte, not the item id: `get_cashslot_item_type` maps
`nItemID / 10000 == 510 → 20` on every version examined (§6.4), and 20 is not
shared with another classification in that function, so the type-keyed arm is
stable across a version bump — unlike the 530 morph coupon, which had to be
classification-keyed.

Arm body, after the shared ownership preamble:

1. Decode `ItemUseSongPlayer`; if `!updateTimeFirst`, overwrite `updateTime`.
2. `character.NewProcessor(l, ctx).GetById()(s.CharacterId())` → `c.Name()`.
   Failure ⇒ log at debug, return, consume nothing. (Same shape as the kite arm.)
3. If `soundLengthMs == 0` ⇒ warn, return, consume nothing. A zero-length song
   is either a broken client or a spoofed body; either way it would produce a
   song that ends the instant it starts.
4. Create `saga.Saga{SagaType: saga.FieldEffectUse, InitiatedBy: "CASH_ITEM_USE"}`
   with the two steps in PRD FR-1.5 order, the second carrying
   `DurationMs: soundLengthMs`.
5. No `EnableActions` — the non-silent inventory operation from the consume
   commit clears the client's exclusive-request lock, the reasoning already
   recorded on the field-effect and morph-coupon arms.

**Client-side pre-gates worth knowing (they are not server obligations).** The
arm at `0x9ed51e` refuses to send at all when `m_bJukeBoxPlaying != 0` or a
transient layer exists, and puts up a `CUtilDlg::YesNo` confirmation first. So a
player whose client has already received a `JUKEBOX_START` cannot use a second
jukebox. The server still handles replacement (§2.4) because that guard lives
only in the sender's client.

### 2.3 `libs/atlas-saga` + `atlas-saga-orchestrator`

- `model.go`: `PlayJukebox Action = "play_jukebox"`.
- `payloads.go`: `PlayJukeboxPayload{WorldId, ChannelId, MapId, Instance,
  ItemId, PlayerName, DurationMs}`. Note `DurationMs`, not weather's seconds —
  the client's value is milliseconds and converting to seconds would silently
  truncate.
- `unmarshal.go`: a case mirroring `FieldEffectWeather`.
- orchestrator `saga/model.go`: the two re-export aliases and the local
  `unmarshal` case.
- `saga/event_acceptance.go`: `sharedsaga.PlayJukebox: {}` in the
  fire-and-forget block — nothing Kafka-side advances this step.
- `saga/handler.go`: `handlePlayJukebox` added to the `Handler` interface, the
  action switch, and implemented as a near-copy of `handleFieldEffectWeather`
  (validate payload → build `field.Model` → call the processor → `StepCompleted`).
- `map_command/processor.go` + `producer.go`: `PlayJukebox(transactionId, f,
  itemId, playerName, durationMs)` producing `CommandTypePlayJukebox` onto
  `EnvCommandTopicMap`.

### 2.4 `atlas-maps` — `map/jukebox/`

A new package structurally parallel to `map/weather/`:

| File | Content |
|---|---|
| `registry.go` | `FieldKey{Tenant, Field}` → `JukeboxEntry{ItemId, PlayerName, ExpiresAt}`; `Set`/`Get`/`Delete`/`GetExpired`, package singleton behind `sync.Once`, same as weather. |
| `processor.go` | `Start(f, itemId, playerName, duration)`, `GetActive(f)`. |
| `producer.go` | `JukeboxStartEventProvider`, `JukeboxEndEventProvider`. |
| `rest.go` | `RestModel{Id, ItemId, PlayerName}`, `GetName() == "jukebox"`. |
| `resource.go` | `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/jukebox`, 404 when absent. |

`kafka/consumer/map/consumer.go` gains `handlePlayJukeboxCommand`, registered
alongside the weather handler on the same topic. Duration cap:

```go
const maxJukeboxDuration = 10 * time.Minute
```

Longer ⇒ capped and logged at warn, exactly as weather does at
`consumer.go:51-55`. Ten minutes is chosen as an order of magnitude above any
real WZ sound while still bounding a crafted command; the constant is one named
value, trivially retunable.

`tasks/jukebox.go` mirrors `tasks/weather.go` verbatim in structure, including
the `envContext` origination and the injected `emit` seam that makes the sweep
unit-testable — that seam exists because an empty `ENVIRONMENT` header would
make every live deployment react to one pod's expiry. Registered in `main.go`
next to `tasks.NewWeather`, one-second interval.

**Replacement (PRD FR-4.3)** is `registry.Set` overwriting the key. The prior
`ExpiresAt` disappears with the entry, so the sweep can never emit a
`JUKEBOX_END` for the replaced song and cut the new one short.

### 2.5 `atlas-channel` — broadcast, replay, REST client

- New `jukebox/` package (`processor.go`, `requests.go`, `rest.go`, `mock/`),
  a direct analogue of `weather/`.
- `handleStatusEventJukeboxStart`: tenant/world/channel guard, then
  `_map.NewProcessor(...).ForSessionsInMap(f, session.Announce(...)(
  fieldcb.PlayJukeboxWriter)(fieldcb.NewPlayJukebox(int32(itemId),
  playerName).Encode))`. Broadcast to **everyone in the field including the
  user** — the sending client does not apply the effect locally (§6.3).
- `handleStatusEventJukeboxEnd`: same fan-out with
  `fieldcb.NewPlayJukebox(-1, "")`. Exactly `-1`; see §3.2.
- Map-enter replay: one more `routine.Go` block beside the weather block at
  `consumer.go:346`, querying `jukebox.NewProcessor(l, ctx).GetActive(f)` and
  announcing `PlayJukebox` to `s` alone. Returns silently on error, so an
  unreachable atlas-maps costs the song, not the map entry.

---

## 3. Key decisions

### 3.1 No `FieldEffect` BGM packet — and why sending one would be a bug

Sending `FieldEffectBackgroundMusicBody` alongside the jukebox is not merely
redundant, it is actively wrong. That packet sets `CMapLoadable::m_sChangedBgmUOL`,
and `CMapLoadable::RestoreBGM` (§6.2) restores *that* string when it is set,
falling back to `PlayBGMFromMapInfo` only when it is empty. A jukebox that also
sent a BGM override would leave the field permanently playing the jukebox track
after the song "ended" — the exact failure PRD FR-5.2 is trying to prevent.

The client's own path is complete: `OnPlayJukeBox` → `PrepareNextBGM` (fade out,
schedule +2500 ms) → `CMapLoadable::Update` → `PlayNextMusic` → read the item's
`info/path` → `PlayBGM`. Nothing in it needs the server.

**Consequence for atlas-data:** no change at all. The PRD's new `info/path`
attribute is not built. This is the single largest scope reduction in the design
and the one most worth a second look before `/plan-task`.

### 3.2 The stop signal is exactly `-1`, not "a negative id"

`PlayNextMusic` branches on `m_nJukeBoxItemID == -1` for the restore path. Any
*other* non-zero value falls into the else branch, calls
`CItemInfo::GetItemInfo(negativeId)`, gets a null interface, and returns
**without clearing `m_nJukeBoxItemID`** — so `CMapLoadable::Update`'s
`m_nJukeBoxItemID != 0 && now > m_tNextMusic` gate stays true and the client
re-enters `PlayNextMusic` every frame, forever. `-1` is a correctness
requirement, not a convention.

Note also that `OnPlayJukeBox` stores the decoded id *before* its
`get_consume_cash_item_type(itemId) == 20` gate, and `-1 / 10000 == 0` fails
that gate. The stop packet therefore produces no chat line and no
`PrepareNextBGM` call; the restore is driven entirely by `Update` seeing the
stored `-1` against an already-elapsed `m_tNextMusic`. That works because
`PlayNextMusic` never resets `m_tNextMusic` — it clears `m_nJukeBoxItemID`
instead — so the timestamp is always in the past by the time a stop arrives.

The `PlayJukebox` codec already models this: `itemId` is `int32` and the
trailing name is written only when `itemId >= 0`. No codec change.

### 3.3 Duration: client-supplied, server-capped

Alternatives considered:

| Option | Verdict |
|---|---|
| Server constant (PRD FR-4.2) | Rejected. The client already sends the true length; a constant would cut songs off or leave silence. |
| Trust the client value outright | Rejected. A crafted body could pin a field's BGM indefinitely (PRD §8 "bounded effect"). |
| **Client value, capped at `maxJukeboxDuration`, zero rejected** | **Chosen.** Correct for real clients, bounded for hostile ones, one named constant. |

The zero check lives in the channel arm rather than atlas-maps so that a
zero-length request consumes nothing — by the time the command reaches
atlas-maps the `DestroyAsset` step has already committed.

### 3.4 A parallel `jukebox` package rather than generalising `weather`

Weather and jukebox share a registry shape, an expiry sweep, a REST resource,
and a status-event pair. Extracting a generic `field_effect_registry` would
remove roughly 150 lines of duplication across atlas-maps and atlas-channel.

Rejected for this task. The two differ in entry payload (`Message` vs
`PlayerName`), in duration source (server constant vs client value), in what the
end event carries, and in whether a companion `FieldEffect` packet accompanies
the broadcast. A generic abstraction would be parameterised on all four axes
after exactly two instances — the classic premature-abstraction shape — and it
would put a refactor of an already-verified weather path inside a
new-feature task. If a third such effect arrives, extract then, with three real
call sites to design against.

### 3.5 Registry stays in-process, unpersisted

Same reasoning as weather, restated because it is a deliberate acceptance: an
`atlas-maps` restart silently drops active songs and clients keep playing until
they change maps. For a cosmetic effect bounded at ten minutes this is cheaper
than a persistence story, and it matches the behaviour operators already expect
from weather.

---

## 4. Wire and API contract

### 4.1 Serverbound sub-body (type 20)

| Order | Field | Type | Present when |
|---|---|---|---|
| 1 | `soundLengthMs` | int32 | always |
| 2 | `updateTime` | int32 | `updateTimeFirst == false` (GMS ≤ v84) |

### 4.2 Clientbound

`PlayJukebox` only. Start: `(int32 itemId, string playerName)`. Stop:
`(int32 -1)`, no name.

### 4.3 Kafka

`PLAY_JUKEBOX` on `EnvCommandTopicMap` — body `{itemId, playerName, durationMs}`;
envelope carries `transactionId, worldId, channelId, mapId, instance`.

`JUKEBOX_START` on `EnvEventTopicMapStatus` — body `{itemId, playerName}`.
`JUKEBOX_END` — body `{itemId}`.

Message types are declared per-service in each `kafka/message/map` package, per
the existing duplication convention.

### 4.4 REST

```
GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/jukebox
```

```json
{"data":{"type":"jukebox","id":"5100000",
 "attributes":{"itemId":5100000,"playerName":"Chronicle"}}}
```

`404` with no body when nothing is playing. Tenant from request context.
No `bgmPath` attribute — see §3.1.

---

## 5. Error handling

| Condition | Behaviour |
|---|---|
| Slot does not hold the claimed template | Rejected by the shared preamble; nothing consumed. |
| Character lookup fails | Debug log, return; nothing consumed. |
| `soundLengthMs == 0` | Warn, return; nothing consumed, nothing broadcast. |
| `DestroyAsset` step fails | Saga machinery stops; the jukebox step never runs. |
| `durationMs > maxJukeboxDuration` | Capped, warned, accepted. |
| Song already active in the field | Entry replaced; new song takes effect for everyone; old expiry discarded. |
| atlas-maps unreachable on map enter | Replay block returns silently; map entry unaffected. |
| Broadcast fan-out error | Logged at error; no retry, matching weather. |

---

## 6. Evidence

IDBs used: `GMS_v95.0_U_DEVM.exe.i64` (session `ecc757f4`) and
`MapleStory_dump.exe.i64` GMS v83 (session `754107bf`). No v95.1 IDB is open in
this environment; v95.0 and v83 agree on every point below, which brackets the
supported range.

### 6.1 OQ-1 — the type-20 sub-body

**GMS v95.0**, `CWvsContext::SendConsumeCashItemUseRequest` @`0x9eb3e0`, case-20
arm entered at `0x9ed51e` (`jumptable 009EB50A case 20`):

- `0x9ed51e` `cmp [edi+20h], ebx` — `CMapLoadable::m_bJukeBoxPlaying` (offset
  `0x20`); non-zero ⇒ notice `0x11F`, no send.
- `0x9ed525` `CMapLoadable::TransientLayer_Exist` ⇒ same notice path.
- `0x9ed63e` `CUtilDlg::YesNo` — confirmation dialog.
- `0x9ed678` `push 734h` → `StringPool::GetBSTR` → `0x9ed6ad`
  `IWzProperty::Getitem` on the item-info property → `get_string`. `0x734` is
  the pool index for `"path"`.
- `0x9ed731` `Ztl_bstr_t(path)` → `0x9ed75a` `IWzResMan::GetObjectA(sUOL)` →
  `0x9ed773` cast to `IWzSound`.
- `0x9ed7af` `IWzSound::Getlength` → `0x9ed7b4 push eax` → `0x9ed7b9`
  **`COutPacket::Encode4`**.

**GMS v83**, same function @`0xa0a63f`, case-20 arm at `0xa0c1a2`
(`jumptable 00A0A6E6 case 20`): identical sequence —
`GetItemInfo` @`0xa0c1da`, `GetBSTR` @`0xa0c2ec`, `IWzProperty::GetItem`
@`0xa0c306`, `IWzResMan::GetObjectA` @`0xa0c391`, `sub_644DCF` @`0xa0c3ed`
(a COM property getter at vtable+56 on the `IWzSound`, i.e. `Getlength`), then
`COutPacket::Encode4` @`0xa0c3f6`.

Exactly one `Encode4` in the arm. The trailing `updateTime` on v83/v84 comes
from the shared send tail, as already documented on `ItemUseMorphCoupon`.

### 6.2 OQ-2 — what restores the map BGM

`CField::OnPlayJukeBox` @`0x537940` (v95.0):

```c
v4 = CInPacket::Decode4(iPacket);
this->m_nJukeBoxItemID = v4;
if ( get_consume_cash_item_type(v4) == 20 ) {
    if ( v4 >= 0 ) { /* StringPool 0x1AC3 + item name + DecodeStr(name) → ChatLogAdd */ }
    CMapLoadable::PrepareNextBGM(this);
}
```

`CMapLoadable::PrepareNextBGM` @`0x610040` fades the BGM to zero over 1500 ms and
sets `m_tNextMusic = timeGetTime() + 2500`.

`CMapLoadable::Update` @`0x61dfc0`:

```c
if ( !<ctx flag> && this->m_nJukeBoxItemID && v9 - this->m_tNextMusic > 0 )
    CMapLoadable::PlayNextMusic(this);
```

`CMapLoadable::PlayNextMusic` @`0x61dab0`:

```c
if ( this->m_nJukeBoxItemID == -1 ) {
    CMapLoadable::RestoreBGM(this);
    this->m_bJukeBoxPlaying = 0;
    this->m_nJukeBoxItemID  = 0;
} else {
    CItemInfo::GetItemInfo(..., this->m_nJukeBoxItemID);
    ... StringPool::GetBSTR(0x734) → IWzProperty::Getitem → sPath ...
    CSoundMan::PlayBGM(..., sPath, 1, 0x3E8, 0x3E8, 1);
    this->m_bJukeBoxPlaying = 1;
    this->m_nJukeBoxItemID  = 0;
}
```

The v83 build of the same function @`0x641db2` is identical and names the pool
constant outright: `StringPool::GetBSTR(Instance, &v8, SP_1803_PATH)`. That is
what fixes `0x734`/`0x1803` as `"path"`.

`CMapLoadable::RestoreBGM` @`0x61a4f0`:

```c
if ( Compare(&this->m_sChangedBgmUOL, &sDefault) )
    CSoundMan::PlayBGM(..., m_sChangedBgmUOL, 1, 0x3E8, 0x3E8, 1);
else
    CMapLoadable::PlayBGMFromMapInfo(this);
```

Two conclusions: the client plays the song from the item's own WZ node, and the
client restores the map's own BGM — unless a `FieldEffect` BGM override is
present, in which case it restores *that* instead. §3.1.

### 6.3 OQ-4 — broadcast scope

The sender's client sets no jukebox state in the send arm; `m_bJukeBoxPlaying`
is written only inside `PlayNextMusic`, which is reachable only from
`OnPlayJukeBox` → `PrepareNextBGM`. So the user's own client applies nothing
until the broadcast returns. Broadcast to the whole field, including the user.
No double-apply is possible: the arm's own `m_bJukeBoxPlaying != 0` pre-gate
prevents a second send.

### 6.4 The type byte

`get_cashslot_item_type` @`0x488c70` (v95.0): `case 510: result = 20;`, reached
via `switch (nItemID / 10000)`. `get_consume_cash_item_type` @`0x49c700`
whitelists 20. No other classification yields 20 in that function, so keying the
handler arm on the type byte is safe across versions.

---

## 7. Testing

| Unit | Test |
|---|---|
| `ItemUseSongPlayer` | Byte fixtures for both `updateTimeFirst` values; round-trip Encode/Decode. |
| Type-20 arm | Table test over the existing package-var seams: (a) happy path emits a two-step `FieldEffectUse` saga with `DurationMs` from the wire; (b) slot/template mismatch ⇒ no saga; (c) `soundLengthMs == 0` ⇒ no saga; (d) character lookup error ⇒ no saga. No live Kafka. |
| `map/jukebox` registry | Start, replace (old expiry discarded), expire, tenant isolation on `FieldKey`. |
| `handlePlayJukeboxCommand` | Duration passthrough, cap at `maxJukeboxDuration`, wrong `Type` ignored. |
| `tasks/jukebox.go` | Sweep with a spy `emit`, asserting per-tenant context origination and entry deletion — mirroring `tasks/weather_test.go`. |
| Channel status handlers | `JUKEBOX_START` announces `PlayJukebox(itemId, name)` to every session in the field; `JUKEBOX_END` announces `PlayJukebox(-1, "")`; both drop events for another world/channel/tenant. |
| Regression | Existing `PlayJukebox` codec tests and its `gms_v95` / `jms_v185` evidence records unchanged. |

Flagless `tools/verify.sh` must exit 0.

---

## 8. Risks

- **`-1` exactness.** Any other negative stop value spins the client's `Update`
  loop (§3.2). Guard it with a named constant in the channel handler and a test
  asserting the encoded id.
- **No v95.1 IDB.** Derivation used v95.0 and v83. They agree; a v95.1 spot-check
  during implementation is cheap insurance if an IDB becomes available.
- **`maxJukeboxDuration` is a guess.** Ten minutes bounds abuse without
  truncating any plausible track. If a real WZ sound exceeds it the song is cut
  short — visible only as an early stop, and retunable in one line.
- **Scope reduction.** §3.1 removes work the PRD specified. If a reviewer
  disagrees with the "client resolves the BGM" reading, the atlas-data leg comes
  back — but §6.2's decompilation is unambiguous and the RestoreBGM interaction
  makes the PRD's version incorrect, not merely redundant.

---

## 9. Files touched

| Service / lib | Files |
|---|---|
| `libs/atlas-packet` | `cash/serverbound/item_use_song_player.go` (+test) |
| `libs/atlas-saga` | `model.go`, `payloads.go`, `unmarshal.go` |
| `atlas-saga-orchestrator` | `saga/model.go`, `saga/handler.go`, `saga/event_acceptance.go`, `map_command/processor.go`, `map_command/producer.go`, `kafka/message/map/command.go` |
| `atlas-maps` | `map/jukebox/{registry,processor,producer,rest,resource}.go`, `kafka/consumer/map/consumer.go`, `kafka/message/map/{command,kafka}.go`, `tasks/jukebox.go`, `main.go` |
| `atlas-channel` | `socket/handler/character_cash_item_use.go`, `jukebox/{processor,requests,rest}.go` + `mock/`, `kafka/consumer/map/consumer.go`, `kafka/message/map/kafka.go` |
| `atlas-data` | **none** (§3.1) |
| `libs/atlas-constants` | none |
| `atlas-ui` | none |

---

## 10. PRD requirements this design supersedes

To be reconciled into `prd.md` before `/plan-task`:

- **FR-2.1 – FR-2.4 (atlas-data `info/path`)** — dropped. The client reads the
  WZ node itself (§6.2). No atlas-data or channel-REST-model change.
- **FR-4.2 (server duration constant)** — replaced. Duration is the client's
  `IWzSound::length`, capped server-side at `maxJukeboxDuration` (§3.3).
- **FR-5.1 item 2 (BGM `FieldEffect` on start)** — dropped (§3.1).
- **FR-5.2 (BGM restore on end)** — reduced to `PlayJukebox(-1)` alone; sending a
  BGM packet would break the restore (§3.1, §3.2).
- **FR-4.1 / FR-4.4 / §6.2 `BgmPath` field** — removed from the registry entry,
  the `JUKEBOX_START` body, and the REST resource.
- **FR-5.3 (map-enter replay)** — retained, but announces `PlayJukebox` only.
- **OQ-3** — resolved: no constant song length; only the cap is a constant.
- **Acceptance criteria** — the two clauses requiring a `FieldEffect` BGM packet
  and the one requiring a rejection when `info/path` is absent no longer apply;
  the zero-`soundLengthMs` rejection replaces the latter.
