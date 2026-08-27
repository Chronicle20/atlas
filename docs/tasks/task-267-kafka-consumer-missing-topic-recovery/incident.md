# Incident: `atlas-pr-1449` saga orchestrator deaf on every topic

Source evidence for task-267. Recorded 2026-08-26. Everything here was observed
live in the namespace; nothing is inferred.

## Environment

- PR 1449, namespace `atlas-pr-1449`, `ATLAS_ENV` `1450`
- Tenant `5ae4fd69-6971-4c88-aa5c-d55a8c861cd2`, region GMS, version 83.1
- Reported by the user as "I cannot create a character via the ui."
- Discovered while working `task-256-zombify-healing-consequences`. **PR 1449 is
  not implicated** — that branch touches `atlas-channel`, `atlas-consumables`
  and `atlas-messages` only; `atlas-character-factory`,
  `atlas-saga-orchestrator` and `libs/atlas-kafka` are untouched by its diff.

## Timeline (UTC)

| Time | Event |
| --- | --- |
| ~18:02:52 | Service pods come up. |
| 18:02:58.233 | 22 of ~24 topic consumers in group `Saga Orchestrator Service [1450]` log `holds no partition assignment in generation 1; healthy-idle` at debug. Only `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS-1450` and `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-1450` — the two topics that already existed — take partition 0. |
| 18:06:25, 18:07:00, 18:08:37 | User's UI attempts: `POST /api/factory/characters/from-preset`, preset Marksman (4th job), name `Atlas`, account 1, world 0. All accepted. |
| 18:11:07 | `atlas-kafka-precreate` logs `creating 170 topics`. |
| 18:12:41 | `atlas-kafka-precreate` logs `reconciled 170 topics` — ten minutes after the consumers joined. |
| 18:12:59 | Direct repro from inside the namespace (below). HTTP 202, `transactionId 334c95f4-453f-4cef-be3e-2cbf2cb10c58`. Character does not appear. |
| 18:15:29 | `kubectl rollout restart deploy/atlas-saga-orchestrator -n atlas-pr-1449`. |
| 18:15:30 | New pod takes `partition: 0` on `COMMAND_TOPIC_SAGA-1450` and drains all four backlogged sagas, including `334c95f4-…`. |
| after | `GET /api/characters/` returns `Atlas` (id 1) and `Zorbo` (id 2), both jobId 322, level 200 — the user's attempt and the repro. |

## Repro

```sh
curl -X POST -H 'Content-Type: application/vnd.api+json' \
  -H 'TENANT_ID: 5ae4fd69-6971-4c88-aa5c-d55a8c861cd2' -H 'REGION: GMS' \
  -H 'MAJOR_VERSION: 83' -H 'MINOR_VERSION: 1' \
  -d '{"data":{"type":"preset-create","attributes":{"presetId":"aa6cbb45-1ef9-4bb0-9368-21297edad68c","accountId":1,"worldId":0,"name":"Zorbo"}}}' \
  http://atlas-ingress.atlas-pr-1449.svc.cluster.local/api/factory/characters/from-preset
```

## Observed

- `atlas-character-factory` returned **202 Accepted** with a transaction id. It
  logged the preset lookup, the name-validity check, nine
  `GET /api/data/equipment/{id}` calls and the batched `GET /api/data/skills?ids=…`
  (28 skills). No error at any level.
- `atlas-saga-orchestrator` logged **nothing** for any of the transactions — only
  startup and per-topic idle chatter.
- `COMMAND_TOPIC_SAGA-1450` log-end offset was **4**: the factory's produce
  succeeded, the messages sat unconsumed.

Net effect: every saga-driven flow in the environment was dead, not just
character creation. The UI symptom was the most visible instance.

## Root cause

The consumer group joined before its topics existed and never re-joined once
they were created.

- A kafka-go group member subscribed to a zero-partition topic receives an empty
  assignment.
- `libs/atlas-kafka/consumer/group.go:115` sets `WatchPartitionChanges: false`,
  so the member is never told the topic later gained a partition, and nothing
  else forces a rebalance.
- `libs/atlas-kafka/consumer/engine_group.go:71-86` classifies the empty
  assignment as *healthy-idle*, logs it at debug, and `continue`s into a `Next`
  that parks until the generation ends. Nothing ends it.

The healthy-idle classification is correct for its designed case (`replicas: 2`
against a single-partition topic — one member legitimately holds nothing) but it
also absorbs this case, where *every* member holds nothing because the topics did
not exist. No warn, no metric, no self-heal.

## Notes on the originally-suspected deploy-ordering cause

The bug report listed deploy ordering as a candidate fix with "repo unknown".
The gate already exists in this repository:

- `deploy/k8s/base/atlas-kafka-precreate.yaml:38` — Job at
  `argocd.argoproj.io/sync-wave: "0"`.
- `deploy/k8s/base/kustomization.yaml:89-102` — every `Deployment` patched to
  `argocd.argoproj.io/sync-wave: "10"`, explicitly so Argo waits for the Job.

So ordering *is* guaranteed within a sync that rolls the Deployments. The
observed inversion means the pods came from an earlier sync and Argo skipped the
already-healthy wave-10 Deployments in the sync that recreated the Job. A wave
gate structurally cannot cover "topics recreated under live pods," which is why
task-267 fixes the library instead. Deploy-side hardening, if any, is a separate
task.

## False alarm: `EVENT_TOPIC_PARCEL_CUSTODY_STATUS`

The report flagged this topic as subscribed without the `-1450` env suffix. It is
suffixed in all three overlays —
`deploy/k8s/overlays/pr/kustomization.yaml:323`,
`deploy/k8s/overlays/main/kustomization.yaml:203`,
`deploy/k8s/overlays/pr-sparse/kustomization.yaml:489`. The unsuffixed literal at
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/parcel/custody/kafka.go:118`
is the environment-*variable* name, not the topic value, exactly like every other
topic in that service. No action.

## Resolution

Environment unblocked at 18:15:30 by restarting `atlas-saga-orchestrator`.
Character creation from the UI verified working after the restart. No commit was
made on `task-256-zombify-healing-consequences`.
