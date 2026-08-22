# bug: clicking Duey opens nothing — "Parcel" writer never registered in atlas-channel

**Status:** fixed by `ce894c841` ("fix(atlas-channel): register
parcelcb.ParcelWriter in produceWriters()"); **live re-test pending**. Found in
live testing on `atlas-pr-1434` (task-241 branch head `f2ce001f8`).

## Resolution

- `ce894c841` adds the `parcelcb` import and `parcelcb.ParcelWriter` to
  `produceWriters()`, plus the regression test in `writers_test.go`. The test
  was confirmed RED before the `main.go` change and GREEN after.
- `tools/verify.sh --quick --base f2ce001f8` exits **0** — 90 changed Go
  modules, all guards passed (build/vet, analyzer, skill/job id, scope,
  producer seam, service registration, env domain, env bootstrap, lint/format).
- Live confirmation on `atlas-pr-1434` still outstanding: it requires the
  rebuilt `atlas-channel` image to roll out, then a fresh Duey click. Until
  that is done, this bug is fixed-but-unconfirmed. Per the "Not yet answered"
  note below, the first re-test will exercise only the empty-mailbox OPEN body.
**Environment:** tenant `a049bb75-1ccc-4cb8-ac6a-bd604dfbbe5b`, GMS **83.1**
(`atlas-pr-bootstrap-tenant` → `MAJOR_VERSION=83, MINOR_VERSION=1, REGION=GMS`),
map 240000000, character 1.

## Reproduced

Yes. Clicking NPC 9010009 (Duey) in map 240000000. No dialog appears; the client
is otherwise responsive.

Note: an unrelated cluster outage (nodes `eos`/`ananke` NotReady, the single
Kafka broker stranded, ~10 services crashlooped) masked this at first. The
outage was resolved before this reproduction — every pod in the namespace was
`Running` and Kafka healthy at the time of the capture below.

## Observed

The whole chain works up to the final packet write. From one click
(trace `bec1f587fa810ff2f115b47c7f8576ff`, transaction
`8b352c2e-710b-4e22-aac4-ee965b369ab9`):

`atlas-channel` 17:13:33.474 →
```
[NPCStartConversationHandle] read [oid [8] x [-1362] y [293]]
Starting NPC [9010009] conversation for character [1].
```

`atlas-npc-conversations` → `Processing state [openDuey]` → `Executing 1
operations`, emitting on `COMMAND_TOPIC_SAGA-237a`:
```json
{"transactionId":"8b352c2e-...","sagaType":"inventory_transaction",
 "initiatedBy":"npc-conversation-batch",
 "steps":[{"stepId":"open_duey-1","status":"pending","action":"show_parcel",
 "payload":{"characterId":1,"npcId":9010009,"worldId":0,"channelId":0,"quick":false}}]}
```

`atlas-saga-orchestrator` → `Handling saga command` → `Progressing saga step
[open_duey-1]` → `Marked earliest pending step as [completed]`, and produces on
`COMMAND_TOPIC_PARCEL-237a`.

`atlas-channel` 17:13:33.659 receives it:
```json
{"transactionId":"8b352c2e-...","worldId":0,"channelId":0,"characterId":1,
 "npcId":9010009,"quick":false,"type":"SHOW_PARCEL"}
```
fetches the mailbox successfully
(`GET /api/parcels?filter[recipientId]=1&filter[worldId]=0&filter[status]=pending`
→ `response: null`, i.e. an empty but valid mailbox), then at 17:13:33.663:

```json
{"error":{"message":"writer not found"},"log.level":"error",
 "message":"Unable to show parcel dialog to character [1].",
 "originator":"COMMAND_TOPIC_PARCEL-237a","service.name":"atlas-channel"}
```

The saga still reports `COMPLETED` — `handleShowParcel`
(`saga/handler.go:2623`) self-completes on a successful *produce*, so the
failure downstream in the channel is invisible to the saga.

## Expected

`session.Announce(...)(parcelcb.ParcelWriter)(parcelcb.ParcelOpenBody(...))`
writes the Duey OPEN packet and the parcel dialog appears.

## Root cause

`produceWriters()` in `services/atlas-channel/atlas.com/channel/main.go`
(lines 678–900) never lists `parcelcb.ParcelWriter` (`"Parcel"`,
`libs/atlas-packet/parcel/clientbound/parcel.go:16`). `main.go` does not import
`parcelcb` at all — `grep -c "Parcel" main.go` returns **0**.

`BuildWriterProducer` registers a writer only when the name is in BOTH the
tenant config AND the code-side `produceWriters()` list, and it warns only for
the opposite mismatch. Every seed template *does* declare the opcode —
`grep -rln '"Parcel"' services/atlas-configurations/seed-data/templates/`
matches all eight (`gms_72_1`, `79_1`, `83_1`, `84_1`, `87_1`, `92_1`, `95_1`,
`jms_185_1`) — so the config half is fine and the code half is missing.

This is the identical failure class documented in
`services/atlas-channel/atlas.com/channel/writers_test.go:11-15` for the MTS
"Charge" button: a handler announces, the opcode is mapped, the writer name is
absent from `produceWriters()`, and it surfaces only as a runtime
`writer not found`.

**Not version-specific.** The missing registration is unconditional, so every
tenant and every version is affected, not just GMS 83.1.

A repo sweep of announced-vs-registered writer names in atlas-channel found
`parcelcb.ParcelWriter` to be the only genuine omission
(`chatcb.WorldMessageWriter` is a false positive — registered at `main.go:775`
under the `chatCB` alias).

## Fix

- `services/atlas-channel/atlas.com/channel/main.go` — add the `parcelcb`
  import (`github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound`)
  and add `parcelcb.ParcelWriter` to the `produceWriters()` slice (678–900),
  placed with the other feature-writer groups.
- `services/atlas-channel/atlas.com/channel/writers_test.go` — add a regression
  test in the shape of `TestProduceWriters_RegistersMtsWriters` asserting
  `parcelcb.ParcelWriter` is registered. This is the guard that would have
  caught it; task-241 shipped the announce without it.

## Not yet answered

- Whether any *other* task-241 clientbound packet is announced through a writer
  that is registered but whose per-version opcode is absent from a given
  template — the sweep above covered the code-side list only, not per-version
  opcode coverage. Verified only that `"Parcel"` is present in all eight
  templates.
- Live confirmation that the dialog renders correctly with a **non-empty**
  mailbox. This reproduction had `response: null` (empty mailbox), so only the
  empty-list OPEN body will be exercised by the first re-test.
