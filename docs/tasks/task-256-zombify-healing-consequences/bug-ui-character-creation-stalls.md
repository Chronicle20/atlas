# Bug: character creation from the UI never completes in the PR env

- **Task**: task-256-zombify-healing-consequences
- **PR / env**: PR 1449, namespace `atlas-pr-1449` (ATLAS_ENV `1450`)
- **Tenant**: `5ae4fd69-6971-4c88-aa5c-d55a8c861cd2`, region GMS, version 83.1
- **Reported**: 2026-08-26 — "I cannot create a character via the ui."
- **Status**: root cause established; environment unblocked by a restart. No fix
  committed on the task-256 branch — the defect is not in this PR's diff.

## Reproduced

Yes, twice.

1. The user's UI attempts at 18:06:25, 18:07:00 and 18:08:37 UTC
   (`POST /api/factory/characters/from-preset`, preset = Marksman — 4th job,
   name `Atlas`, account 1, world 0).
2. Direct repro at 18:12:59 UTC from inside the namespace:

```sh
curl -X POST -H 'Content-Type: application/vnd.api+json' \
  -H 'TENANT_ID: 5ae4fd69-6971-4c88-aa5c-d55a8c861cd2' -H 'REGION: GMS' \
  -H 'MAJOR_VERSION: 83' -H 'MINOR_VERSION: 1' \
  -d '{"data":{"type":"preset-create","attributes":{"presetId":"aa6cbb45-1ef9-4bb0-9368-21297edad68c","accountId":1,"worldId":0,"name":"Zorbo"}}}' \
  http://atlas-ingress.atlas-pr-1449.svc.cluster.local/api/factory/characters/from-preset
# HTTP 202, transactionId 334c95f4-453f-4cef-be3e-2cbf2cb10c58
```

The character did not appear.

## Observed

- `atlas-character-factory` accepted the request and returned **202 Accepted**
  with a transaction id. It logged the preset lookup, the name-validity check,
  the nine `GET /api/data/equipment/{id}` calls and the batched
  `GET /api/data/skills?ids=...` (28 skills returned). No error at any level.
- `atlas-saga-orchestrator` logged **nothing** for the transaction. Its only
  lines were startup and per-topic idle chatter.
- `COMMAND_TOPIC_SAGA-1450` had a log-end offset of **4** — the factory's
  produce succeeded; the messages sat unconsumed.
- The orchestrator's consumers were stuck:

```
18:02:58.233 debug Consumer for topic [COMMAND_TOPIC_SAGA-1450]
  (group [Saga Orchestrator Service [1450]]) holds no partition assignment
  in generation 1; healthy-idle.
```

  22 of ~24 topic consumers in that group logged this and never logged an
  assignment. The only two that did hold partition 0 were
  `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS-1450` and
  `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-1450`.
- `kubectl rollout restart deploy/atlas-saga-orchestrator -n atlas-pr-1449` at
  18:15:29 fixed it immediately. The new pod took `partition: 0` on
  `COMMAND_TOPIC_SAGA-1450` and drained the 4 backlogged sagas at 18:15:30,
  including transaction `334c95f4-…`. `GET /api/characters/` then returned
  characters `Atlas` (id 1) and `Zorbo` (id 2), both jobId 322, level 200 —
  the user's attempt and the repro.

## Expected

The saga command produced by `atlas-character-factory` is consumed by
`atlas-saga-orchestrator` within seconds, the 43-step `character_creation` saga
runs, and the character appears.

## Root cause

The orchestrator's Kafka consumer group joined **before its topics existed** and
never re-joined once they were created.

- The service pods came up at ~18:02:52. The `atlas-kafka-precreate` job for
  this env reconciled its 170 topics at **18:12:41** — ten minutes *after* the
  consumers joined (`[2026-08-26T18:11:07Z] creating 170 topics`,
  `[2026-08-26T18:12:41Z] reconciled 170 topics`). The two topics that did get
  an assignment are the two that already existed at join time.
- A kafka-go group member subscribed to a topic with zero partitions receives an
  empty assignment. `libs/atlas-kafka/consumer/group.go:115` sets
  `WatchPartitionChanges: false`, so the member is never told the topic later
  gained a partition, and nothing else forces a rebalance.
- `libs/atlas-kafka/consumer/engine_group.go:83` classifies the empty assignment
  as **healthy-idle** and logs it at debug. That classification is correct for
  its designed case (replicas: 2 against a single-partition topic, so one member
  legitimately holds nothing) but it also silently absorbs this case, where
  *every* member of a single-replica deployment holds nothing because the topics
  did not exist yet. There is no warn, no metric, no self-heal.

Net effect: any Atlas env whose services start before `atlas-kafka-precreate`
finishes has permanently deaf consumers until something restarts the pods. The
UI symptom is only the most visible instance — every saga-driven flow in that
env was equally dead.

**This is not caused by the task-256 diff.** `atlas-character-factory`,
`atlas-saga-orchestrator` and `libs/atlas-kafka` are untouched by PR 1449; the
branch changes `atlas-channel`, `atlas-consumables` and `atlas-messages` only.
The env was mid-resync when the bug was reported (`atlas-pr-create-dbs` and
`atlas-kafka-precreate` both ran minutes before the failing attempts).

## Fix

No fix has been committed. Two candidate surfaces, neither on the task-256
branch — both need their own task/branch:

1. `libs/atlas-kafka/consumer/group.go:108-115` — `WatchPartitionChanges: false`.
   Setting it true makes kafka-go watch partition counts and trigger a rebalance
   when a subscribed topic appears or grows. Guarded by
   `libs/atlas-kafka/consumer/group_test.go:37-42`, which asserts it stays
   false to match the legacy engine; that assertion would have to be revisited
   deliberately.
2. `libs/atlas-kafka/consumer/engine_group.go:64-87` — the healthy-idle branch.
   It cannot distinguish "another member holds this partition" from "this topic
   has no partitions at all". Reading the topic's partition count on entry
   would separate the two and let the second case warn (or re-create the
   consumer) instead of parking forever.
3. Deploy ordering (repo unknown — not yet located): `atlas-kafka-precreate`
   should complete before the service deployments roll, e.g. as an Argo sync
   wave / pre-sync hook.

Secondary, unrelated to the outage but found on the way:

4. `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:57-67`
   — `handleCreateFromPreset` swallows the error entirely: it maps it to a status
   code and writes the header with no log line and no response body. Its sibling
   `handleCreateCharacter` (same file, line 104) logs. Every preset-creation
   failure is invisible in the logs.
5. `atlas-saga-orchestrator` subscribes to `EVENT_TOPIC_PARCEL_CUSTODY_STATUS`
   with **no env suffix** while every other topic in the same group is suffixed
   `-1450`. Cross-env bleed / dead subscription; worth a separate look.

## Not yet answered

- Why the PR-env sync started the service deployments before the topic
  precreate job. Is this ordering guaranteed anywhere, or incidental?
- Whether the same window exists in `atlas-main` on a cold start (main's
  orchestrator currently holds partition 0 on `COMMAND_TOPIC_SAGA-main`, so it
  is healthy right now — that says nothing about its startup ordering).
- Whether `WatchPartitionChanges: true` is safe for the other engines/groups, or
  whether the rebalance churn it introduces is why it was pinned false.
- Whether `EVENT_TOPIC_PARCEL_CUSTODY_STATUS` is intentionally unsuffixed.

## Resolution

- Environment `atlas-pr-1449` unblocked at 18:15:30 UTC by restarting
  `atlas-saga-orchestrator`. Character creation from the UI verified working
  after the restart (characters `Atlas` and `Zorbo` created from the backlog).
- No commit on `task-256-zombify-healing-consequences`. PR 1449 is not
  implicated.
