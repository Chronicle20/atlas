# Whisper `/find` — Accurate Target Location — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-18
---

## 1. Overview

The whisper `/find` command (and its buddy-window twin) lets a player ask the server where
another character currently is. The client renders one of four mutually exclusive answers,
selected by a `findMode` discriminator byte: the target is on a map (`findMode 1`), in the
cash shop (`findMode 2`), on a different channel (`findMode 3`), or not findable at all
(the error shape). All four wire encodings already exist and are exercised in
`libs/atlas-packet/field/clientbound/whisper.go`.

The server-side selection between those four shapes is what is broken. Two of the branches
in `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper.go`
are hard-coded stubs left behind by earlier work, and two further defects fall out of the
same function's control flow. The net effect is that `/find` gives a confidently wrong
answer for every target who is not standing on a map on the requester's own channel: a
player in the cash shop is reported as if they were reachable elsewhere, a player on
channel 7 is reported as being on channel 1, and a player who is logged off entirely is
also reported as being on channel 1.

This task replaces the stubs with real lookups. The cross-channel half is nearly free —
`atlas-maps` already serves the target's full field (world, channel, map, instance) and the
handler already calls that endpoint but discards everything except the map id. The cash-shop
half needs a new piece of infrastructure: today no service records that a character is in
the cash shop, because the event that announces it is consumed only for its side effect of
removing the character's map location. This task gives `atlas-cashshop` a presence store fed
by that same event, and a read endpoint the channel can consult.

## 2. Goals

Primary goals:

- `/find` reports a cash-shop occupant with the cash-shop result shape, not a fabricated channel.
- `/find` reports an off-channel target's **real** channel number, not a hard-coded `0`.
- `/find` reports an offline target as not-findable instead of claiming they are on channel 1.
- `/find` does not disclose the location of a character in a different world.
- `/find` does not disclose the location of a GM character to a non-GM requester.
- Both find arms — `WhisperModeFind` (0x09) and `WhisperModeBuddyWindowFind` (0x48) — are
  specified and tested, not merely sharing an untested code path.
- Every find-result wire shape is pinned by a byte-level fixture test.

Non-goals:

- Whisper **chat** delivery. The same-world gate, command-registry passthrough, and
  failure result already work (`atlas-messages/.../message/processor.go:85`, handler lines 29–36).
  This task does not touch the `WhisperModeChat` branch.
- Buddy-list online/offline status, party member channel display, or any other presence
  consumer. The presence store this task adds is available to them, but wiring them is out of scope.
- A distinct "in MTS" find result. See §4.3 for why MTS is deliberately folded into the
  cash-shop answer.
- Any change to the four find-result encodings in `libs/atlas-packet`. The encodings are
  correct; only their selection is wrong. Tests are added, structs are not modified.
- A general-purpose presence service. Presence is added to `atlas-cashshop` because
  `atlas-cashshop` owns the cash-shop domain, not as the first step of a broader refactor.

## 3. User Stories

- As a player, I want `/find <name>` to tell me the actual channel a friend is on, so that
  I can change to that channel and meet them.
- As a player, I want `/find <name>` to tell me my friend is in the cash shop, so that I
  know to wait rather than hunting channels for them.
- As a player, I want `/find <name>` to tell me plainly that a character is not findable
  when they are logged off, so that I do not change to channel 1 and search an empty map.
- As a player, I want `/find` on a character in another world to fail rather than report a
  channel I cannot reach from where I am.
- As a GM, I want my location to be hidden from ordinary players' `/find`, so that I can
  observe without being tracked.
- As a player using the buddy window's find button, I want the same correctness guarantees
  as the chat-box `/find` command.

## 4. Functional Requirements

### 4.1 Find-result selection

`produceFindResultBody` in
`services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper.go`
resolves the answer by evaluating the following ordered rules. The first matching rule wins;
evaluation stops there.

| # | Condition | Result shape |
|---|---|---|
| FR-1 | Target name does not resolve to a character | `WhisperFindResultError` |
| FR-2 | Target's world ≠ requester's world | `WhisperFindResultError` |
| FR-3 | Target is a GM and requester is not a GM | `WhisperFindResultError` |
| FR-4 | Target is present in the cash shop (or MTS) | `WhisperFindResultCashShop` |
| FR-5 | Target holds a live session on the requester's channel | `WhisperFindResultMap` (see FR-9) |
| FR-6 | Target has a resolvable location on another channel | `WhisperFindResultChannel` with that channel id |
| FR-7 | None of the above (target is offline) | `WhisperFindResultError` |

**FR-1 — unresolvable name.** Unchanged from today: a `GetByName` failure yields
`NewWhisperFindResultError(resultMode, targetName)`. Note the current code passes the
*requested* name here, not a canonical one, which is correct — there is no character to
canonicalise against.

**FR-2 — cross-world.** `character.ProcessorImpl.GetByName`
(`services/atlas-channel/atlas.com/channel/character/processor.go:236`) filters by name
within the tenant only; it applies no world filter. The handler MUST compare the resolved
character's world against `s.Field().WorldId()` and return the error shape when they differ.
This branch is wire-identical to FR-1 and FR-7 but MUST be a distinct code branch with its
own log line (see FR-13) and its own test, so that a cross-world probe is distinguishable
in operations from an ordinary miss.

*Rationale for wire-identical treatment:* none of the four client shapes carries a world
field. `WhisperFindResultChannel` writes a bare channel int
(`whisper.go:227`), so "channel 3, world 1" and "channel 3, world 0" are indistinguishable
to the requester's client, and the requester cannot reach another world's channel without a
full world transfer. Reporting a channel would therefore be actively misleading. The error
shape is the only honest answer the protocol can express.

**FR-3 — GM concealment.** The channel's character model already carries the flag
(`character/model.go:64`, `Gm() bool`, backed by `gm int`). When the resolved target is a GM
and the requesting session is not, the handler MUST return the error shape. A GM requester
sees GM targets normally. A GM finding a non-GM is unaffected. Requester GM status is read
from the session (`session.Model`), not re-fetched.

**FR-4 — cash shop / MTS.** Determined in two ways depending on where the target is:

- *Same channel:* a cash-shop or MTS occupant still holds a live session on the channel;
  they are merely filtered out of map queries by the `CashScene() == CashSceneNone` guard
  (`session/processor.go:115,135`). When the local session lookup succeeds, the handler MUST
  read `CashScene()` (`session/model.go:225`; constants `CashSceneNone=0`,
  `CashSceneCashShop=1`, `CashSceneMts=2` at `model.go:20-22`) and return the cash-shop shape
  for any non-`None` value. **No service call is required for this case.**
- *Different channel:* the handler queries the new `atlas-cashshop` presence endpoint (§5.1).

**FR-5 — same-channel map.** Unchanged in substance from today, but now gated behind FR-4:
only a session whose `CashScene()` is `CashSceneNone` produces a map result.

**FR-6 — remote channel.** The handler MUST call `location.GetField`
(`services/atlas-channel/atlas.com/channel/maps/location/requests.go:75`) rather than
`location.ResolveMapId`. `GetField` returns a full `field.Model` built from the
`atlas-maps` `GET /characters/{id}/location` response, which already carries `WorldId`,
`ChannelId`, `MapId`, and `Instance` (`RestModel`, `requests.go:33`). The channel id from
that field is passed to `NewWhisperFindResultChannel`.

The channel id is written to the wire **0-based** (`w.WriteInt(m.channelId)`,
`whisper.go:227`); the client adds one for display. `channel.Id` is already the 0-based
internal representation, so it is passed through with a numeric conversion only — no `+1`
or `-1` adjustment. The existing hard-coded `0` is exactly why every off-channel target
currently reads as "channel 1".

**FR-7 — offline.** `location.GetField` returns `location.ErrNotFound` (mapped from HTTP 404
at `requests.go:78-80`) when the character has no location row. `atlas-maps` removes that row
on logout (`maps/kafka/consumer/character/consumer.go:182`, `ExitAll`). Because FR-4 has
already ruled out cash-shop presence by this point, an `ErrNotFound` here means the character
is genuinely offline, and the handler MUST return the error shape.

An infrastructure error from `GetField` (5xx, network, timeout) is **not** `ErrNotFound` and
MUST be distinguished: log at error level and return the error shape. The handler MUST NOT
reuse `location.ResolveMapId`'s behaviour of collapsing every failure to map id 0 — that
collapse is what allows a transport failure to be rendered as a real location today.

### 4.2 Cash-shop presence in atlas-cashshop

**FR-8 — presence store.** `atlas-cashshop` gains a presence record keyed by character,
scoped by tenant, recording that a character is currently inside the cash-shop stage and the
world/channel they entered from. It is maintained by a new consumer on
`EVENT_TOPIC_CASH_SHOP_STATUS` (`cashshop.EnvEventTopicStatus`), which `atlas-cashshop` does
not consume today — its existing cash-shop consumer subscribes to the *command* topic only
(`kafka/consumer/cashshop/consumer.go:31`).

Transitions:

| Event | Source | Effect |
|---|---|---|
| `CHARACTER_ENTER` on `EVENT_TOPIC_CASH_SHOP_STATUS` | `channel/cashshop/producer.go:14` | Upsert presence for the character with the event's `WorldId` / `ChannelId` |
| `CHARACTER_EXIT` on `EVENT_TOPIC_CASH_SHOP_STATUS` | `channel/cashshop/producer.go:27` | Delete presence |
| Character `LOGOUT` on `EVENT_TOPIC_CHARACTER_STATUS` | atlas-character | Delete presence |
| Character `DELETED` on `EVENT_TOPIC_CHARACTER_STATUS` | atlas-character | Delete presence |

**FR-9 — the logout transition is mandatory, not defensive.** `CharacterExitCashShopStatusEventProvider`
is emitted from exactly one call site: `cashshop.NewProcessor(...).Exit(...)` in
`services/atlas-channel/atlas.com/channel/socket/handler/map_change.go:43`, i.e. only when the
player leaves the cash shop *back into a map*. A player who disconnects while still inside the
cash shop never produces a `CHARACTER_EXIT`. Without the logout transition, presence would
leak permanently and `/find` would report a long-logged-off character as "in cash shop"
forever — trading one wrong answer for another. `atlas-cashshop` already consumes
`character.EnvEventTopicStatus` (`kafka/consumer/character/consumer.go:22`) but currently
handles only `StatusEventTypeDeleted` (line 42), so this is a new handler on an existing
consumer.

**FR-10 — presence records are ephemeral.** Presence describes a live connection, not durable
game state. A record whose owning channel is no longer running is meaningless. Startup and
recovery behaviour for stale records is an open question (§9, OQ-1).

### 4.3 MTS is reported as cash shop

**FR-11.** `MtsEntryHandleFunc` renders the ITC inside the cash-shop `CStage` and therefore
emits the *same* cash-shop `CHARACTER_ENTER` event that the cash shop itself emits
(`socket/handler/mts_entry.go:103`). The only thing distinguishing the two scenes is
`session.SetCashScene(s.SessionId(), session.CashSceneMts)` — a **local, unpublished**
session flag (`mts_entry.go:104`).

Consequently an MTS occupant lands in the cash-shop presence store automatically, and
`/find` reports them with `WhisperFindResultCashShop`. This is intentional and requires no
event-schema change, no new topic, and no change to `atlas-mts`. The client has no distinct
"in MTS" find shape to render in any case. Making the scenes separately observable would
require adding a discriminator to `CharacterMovementBody` or emitting a second event; that is
explicitly deferred.

### 4.4 Both find arms

**FR-12.** The two arms differ only in the leading mode byte echoed back to the client:
`0x09` for `WhisperModeFind`, `0x48` for `WhisperModeBuddyWindowFind` (handler lines 46–51).
Every rule in §4.1 applies identically to both. The one existing behavioural difference is
preserved: the `0x09` arm sends `WhisperFindResultMapWithXY` (map id plus x/y), the `0x48`
arm sends `WhisperFindResultMap` (map id only). Each of FR-1 through FR-7 MUST have test
coverage on both arms, or a test that asserts the arms differ only in the mode byte and the
x/y presence.

### 4.5 Observability

**FR-13.** Each terminal branch logs at debug level with the requester id, target name, and
the branch taken, so an operator can tell from logs which of the three wire-identical error
branches (FR-1 unresolvable, FR-2 cross-world, FR-3 GM-concealed, FR-7 offline) produced a
given error result. Infrastructure failures (FR-6/FR-7 non-`ErrNotFound`) log at error level
with the underlying error attached.

## 5. API Surface

### 5.1 New: atlas-cashshop character presence

```
GET /characters/{characterId}/cash-shop-presence
```

JSON:API, following the conventions already used by `atlas-maps`'s
`GET /characters/{id}/location` (which this endpoint deliberately mirrors, since the channel
consumes them as a pair).

Success (`200`) — the character is in the cash-shop stage:

```json
{
  "data": {
    "type": "cash-shop-presences",
    "id": "12345",
    "attributes": {
      "worldId": 0,
      "channelId": 3
    }
  }
}
```

Not present (`404`) — the character is not in the cash-shop stage. This is the ordinary,
expected response for the large majority of calls and MUST NOT be logged as an error by
either side.

The resource type name is `cash-shop-presences`. The required api2go no-op relationship
stubs (`SetToOneReferenceID`, `SetToManyReferenceIDs`) are implemented per the
`libs/atlas-rest` contract, as `location.RestModel` does at `requests.go:55-56`.

### 5.2 New: atlas-channel client for the above

A `presence` package under `services/atlas-channel/atlas.com/channel/cashshop/` providing:

- `GetPresence(l, ctx, characterId) (Model, error)` returning world + channel.
- `ErrNotFound`, mapped from `requests.ErrNotFound` on 404, mirroring
  `location.ErrNotFound` (`maps/location/requests.go:78-80`) so callers can distinguish
  "not in cash shop" from "lookup failed".
- `SetBaseURLForTest` for httptest-driven tests, mirroring
  `location.SetBaseURLForTest` (`requests.go:88`).

Base URL resolves via `requests.RootUrlFor(ctx, "CASHSHOP")`.

### 5.3 Modified: none

No existing endpoint changes shape. No packet struct changes.

## 6. Data Model

### 6.1 atlas-cashshop — cash-shop presence

| Column | Type | Notes |
|---|---|---|
| `tenant_id` | uuid | Multi-tenancy scope; part of the uniqueness constraint |
| `character_id` | uint32 | Unique per tenant — a character is in at most one cash shop |
| `world_id` | byte | World the character entered from |
| `channel_id` | byte | Channel the character entered from |
| `entered_at` | timestamp | Entry time; supports the staleness question in OQ-1 |

Constraints:

- Unique on (`tenant_id`, `character_id`).
- Lookup by (`tenant_id`, `character_id`) is the only read path.

`CHARACTER_ENTER` is an upsert, not an insert: a duplicate or replayed enter for a character
already present MUST overwrite rather than fail, since Kafka delivery is at-least-once and
the consumer is registered with `message.PersistentConfig` semantics on the producing side.

Migration: additive — one new table, no changes to existing tables, no backfill. An empty
presence table is a correct cold-start state (it means "nobody is in the cash shop", which
becomes true within seconds as sessions turn over).

Whether this is a database table or an in-memory registry is a design-phase decision; the
columns above describe the information required either way. See OQ-1.

## 7. Service Impact

**`atlas-channel`** — the bulk of the change.

- `socket/handler/character_chat_whisper.go`: `produceFindResultBody` rewritten to the FR-1…FR-7
  ordered rules. Both `TODO` comments (lines 59, 74) removed.
- `cashshop/presence/`: new REST client package (§5.2).
- Reads `session.CashScene()` for the same-channel cash-shop case; swaps
  `location.ResolveMapId` for `location.GetField` for the remote case.
- No new Kafka producer or consumer.

**`atlas-cashshop`** — new presence capability.

- New consumer on `EVENT_TOPIC_CASH_SHOP_STATUS` handling `CHARACTER_ENTER` / `CHARACTER_EXIT`.
- New handler on the existing character-status consumer
  (`kafka/consumer/character/consumer.go`) for `LOGOUT`, alongside the existing `DELETED` handler.
- New presence store and processor.
- New REST resource registered in `main.go` alongside the existing
  `AddRouteInitializer(...)` chain (lines 126–133).

**`atlas-maps`** — no change. Its cash-shop consumer
(`kafka/consumer/cashshop/consumer.go:45`) keeps removing the map location on cash-shop entry;
that behaviour is correct and is precisely what makes a separate presence store necessary.

**`atlas-mts`** — no change (see FR-11).

**`libs/atlas-packet`** — tests only. New byte-fixture tests in
`field/clientbound/whisper_test.go`; no struct or encoding changes.

**`atlas-character`** — no change. Existing name and GM-flag lookups suffice.

## 8. Non-Functional Requirements

**Performance.** `/find` is a low-frequency, human-initiated command. The rule ordering in
§4.1 keeps the common cases cheap: FR-5 (same channel, on a map) costs one name lookup plus
one location call, and the cash-shop presence call is reached only for a target who is *not*
on the requester's channel. Worst case is three service calls (name, presence, location).
No caching is required at expected `/find` volumes; introducing one would risk reporting a
stale channel, which is the class of bug this task exists to remove.

**Security and information disclosure.** FR-2 and FR-3 are the reasons this section exists.
Today `/find` leaks the map of any character in the tenant, across worlds, including GMs.
Both leaks are closed by returning the error shape — the same response an unknown name
produces — so a probe cannot distinguish "no such character", "different world", and
"GM" by response shape or by timing beyond ordinary lookup variance.

**Multi-tenancy.** The presence table is tenant-scoped and every query filters on
`tenant_id`. The consumer reads tenant from the Kafka header via the standard
`consumer.TenantHeaderParser` already used by every consumer in the service. The REST
endpoint derives tenant from request context per the existing `atlas-cashshop` resource pattern.

**Observability.** Per FR-13. The four wire-identical error branches must be separable in logs.

**Failure behaviour.** A presence-lookup failure MUST NOT cause `/find` to report a wrong
location. If the presence call fails with anything other than `ErrNotFound`, the handler
logs at error level and continues to the location lookup — a target genuinely on a map is
still answered correctly, and a cash-shop occupant degrades to the offline error shape
rather than to a fabricated channel. `atlas-cashshop` being down must not make `/find`
answer worse than "not findable".

## 9. Open Questions

**OQ-1 — Presence storage: table or in-memory registry?** §6.1 specifies the information,
not the mechanism. A GORM table matches every other `atlas-cashshop` store and survives a
service restart, but then a service restart leaves stale rows for characters whose channel
also restarted. An in-memory registry is self-clearing on restart but loses presence if
`atlas-cashshop` restarts while players are shopping (reporting them as offline until they
leave the shop). If a table is chosen, decide whether `entered_at` drives a sweeper and what
its TTL is. **Design phase decides.**

**OQ-2 — Does the client render `findMode 2` identically for the buddy-window arm?**
`WhisperFindResultCashShop` writes `findMode 2` then `int32 -1` (`whisper.go:119-127`). The
`0x09` path is understood; whether the `0x48` buddy-window handler accepts the same body is
asserted by symmetry, not verified against the client. **Verify against the IDB during
design or implementation** before relying on FR-4 for the buddy arm; if the client diverges,
FR-12 gains an arm-specific exception.

**OQ-3 — GM-to-GM and admin visibility.** FR-3 specifies "GM hidden from non-GM". Whether a
lower-tier GM should see a higher-tier GM is undefined, because the channel character model
exposes GM as a boolean (`gm == 1`, `character/model.go:64-65`) rather than a level. Treating
it as a boolean is assumed. Flag if a GM level exists elsewhere and should be honoured.

**OQ-4 — Instanced maps.** `field.Model` carries an `Instance` uuid, and
`WhisperFindResultMap` carries only a map id. A target inside a map instance the requester
cannot enter is currently reported by map id all the same. This is pre-existing behaviour,
not a regression, and is left unchanged — but it is worth an explicit decision rather than
an accident.

## 10. Acceptance Criteria

Selection logic:

- [ ] Both `TODO` comments are gone from `character_chat_whisper.go` (lines 59 and 74 today),
      and no literal `cs := false` or hard-coded channel argument remains.
- [ ] A target in the cash shop **on the requester's channel** receives
      `WhisperFindResultCashShop`, decided from `session.CashScene()` with no service call.
- [ ] A target in the cash shop **on another channel** receives `WhisperFindResultCashShop`,
      decided from the `atlas-cashshop` presence endpoint.
- [ ] A target in the MTS receives `WhisperFindResultCashShop` (FR-11).
- [ ] A target on channel N (N ≠ requester's channel) receives `WhisperFindResultChannel`
      carrying N, verified for at least one N > 0 — a test that passes with the old
      hard-coded `0` does not count.
- [ ] A logged-off target receives `WhisperFindResultError`, not a channel result.
- [ ] A target in a different world receives `WhisperFindResultError` via a distinct code
      branch with its own test and log line.
- [ ] A GM target receives `WhisperFindResultError` for a non-GM requester, and a normal
      result for a GM requester.
- [ ] A target on the requester's channel and on a map still receives
      `WhisperFindResultMapWithXY` (0x09) / `WhisperFindResultMap` (0x48), with the correct
      map id and coordinates — the one path that works today does not regress.

Presence:

- [ ] `CHARACTER_ENTER` on `EVENT_TOPIC_CASH_SHOP_STATUS` creates presence; a replayed
      duplicate enter upserts rather than erroring.
- [ ] `CHARACTER_EXIT` removes presence.
- [ ] Character `LOGOUT` removes presence — with a test that specifically covers
      *disconnect while inside the cash shop*, the path that emits no `CHARACTER_EXIT` (FR-9).
- [ ] Character `DELETED` removes presence.
- [ ] `GET /characters/{id}/cash-shop-presence` returns 200 with world and channel when
      present, and 404 when not.
- [ ] Presence queries are tenant-scoped; a presence row in tenant A is invisible to tenant B.

Resilience:

- [ ] With `atlas-cashshop` unreachable, `/find` still answers correctly for a target on a
      map, and answers with the error shape (never a fabricated channel) for one who is not.
- [ ] A non-404 failure from the location lookup yields the error shape and an error-level
      log, not map id 0.

Coverage:

- [ ] Byte-fixture tests in `libs/atlas-packet/field/clientbound/whisper_test.go` pin all
      five find-result encodings: map, map-with-xy, channel, cash-shop, and error.
- [ ] Every rule FR-1…FR-7 is covered on both the `0x09` and `0x48` arms.

Gate:

- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.
