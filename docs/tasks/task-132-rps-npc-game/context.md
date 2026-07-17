# RPS NPC Game — Implementation Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer needs before touching code. All paths repo-relative to the worktree root `.worktrees/task-132-rps-npc-game/`.

## Decisions locked during planning (refine the design)

1. **`StartRPSGame` dispatch is synchronous REST, not an async Kafka reply.** Design §9 framed it as "dispatch a command and await a saga-status reply, exactly as gachapon." Exploration showed the gachapon precedent is actually a **synchronous REST call** from `atlas-saga-orchestrator` into the owning service, with the orchestrator self-completing the step (`StepCompleted(true)`) and an **empty `acceptanceTable` entry**. RPS follows that: the orchestrator POSTs `{RPS_URL}rps/games` and self-completes. (Plan Task 13.) This is simpler and matches the real code.

2. **Entry cost lives on the `rpsAction` state JSON (`entryCostMeso`), not fetched from config by npc-conversations.** Design §8 suggested a runtime config fetch or a `local:` helper. To avoid adding a config client to `atlas-npc-conversations`, the entry cost is a field on the `rpsAction` state (mirroring `gachaponAction.ticketItemId`), seeded per version. It stays tenant-tunable and matches the `rps-rewards` config default (1000) by convention. `atlas-rps`'s own copy of the entry cost (for the `GET` response) comes from `rps-rewards` config. (Plan Tasks 23, 25.)

3. **Session store is Redis TTL (design D2), keyed `(tenant, characterId)`.** No Postgres, no migration. Abandoned sessions are swept with no payout; the bet is already spent and prizes only pay on explicit collect, so there is no economic-integrity reason to persist.

4. **Entry saga = `[AwardMesos(−entryCost), StartRPSGame]`; payout saga = `[AwardMesos(+meso)?, AwardAsset(item,qty)?]`.** `AwardMesosPayload.Amount int32` explicitly supports negative for deduction (verified `libs/atlas-saga/payloads.go:70`). The `NOT_ENOUGH_MESO` failure of step 1 gives FR-1.3 for free (routes to the failure dialogue). Payout is built and submitted by `atlas-rps` on `Collect`.

5. **Version set: v83, v84, v87, v95, jms_v185; v92 parked.** The PRD's "v92" was really the v95/jms185 columns. `gms_92_1` template is a stub with no `operations` table and there is no v92 IDB — implementing it would require inventing bytes (forbidden). Parked exactly like task-086 mount-food.

## The two IDA/WZ hard gates (do these before the codecs/ladder)

- **Packet wire formats** (Plan Tasks 14, 16): mode-byte tables and per-frame field layouts for `RPS_GAME` (`CRPSGameDlg::OnPacket`) and `RPS_ACTION` (the six `CRPSGameDlg::` senders) are decompiled per version and written to `ida-rps-clientbound.md` / `ida-rps-serverbound.md`. Codecs (Tasks 15, 17) and seed `operations` tables (Task 20) cite those notes. If a version's IDB is not loaded (per memory `reference_ida_instance_ports_shifted_idbs_v9`, the loaded set rotates and v84/v87/jms may be absent), mark those cells **blocked-pending-IDB** and proceed with the loaded versions — do not invent.
- **Reward-ladder item ids** (Plan Task 26): sourced from Cosmic `9000019.js`, every item id verified against local WZ / atlas-data. The default seed ships a meso-only tunable ladder as a placeholder until this task fills verified content.

## Reference implementations to clone (exact templates)

| New code | Clone from |
|---|---|
| `atlas-rps` main/registry/model/processor/kafka/sweeper spine | `services/atlas-expressions/atlas.com/expressions/` |
| `atlas-rps` REST read layer | `services/atlas-chalkboards/atlas.com/chalkboards/` |
| `atlas-rps` config-read client | `services/atlas-transports/atlas.com/transports/transport/config/` |
| Redis TTL registry + tenant Set | `libs/atlas-redis` `atlas.NewTTLRegistry` / `atlas.NewSet`; usage in `.../expressions/expression/registry.go` |
| Packet dispatcher family (clientbound + serverbound + body funcs + fixtures) | `libs/atlas-packet/storage/` (`operation_body.go`, `clientbound/{error,error_modes,show}.go`, `serverbound/*`, `clientbound/show_test.go`) |
| `candidatesFromFName` cases | `tools/packet-audit/cmd/run.go:1786-1813` (storage) |
| Channel serverbound dispatcher handler | `services/atlas-channel/.../socket/handler/storage_operation.go` (`isStorageOperation`) |
| Channel clientbound writer registration | `services/atlas-channel/.../main.go` `produceWriters()` (storage writer ~line 767); handler map ~894; validators ~904 |
| Channel event consumer → `session.Announce` | `services/atlas-channel/.../kafka/consumer/storage/consumer.go` |
| Saga action + payload + unmarshal | `libs/atlas-saga/{model.go,payloads.go,unmarshal.go}` (gachapon/mesos entries) |
| Orchestrator REST-dispatch handler + client + acceptance entry | `services/atlas-saga-orchestrator/.../saga/handler.go` (`handleSelectGachaponReward` ~2378), `gachapon/` client, `saga/event_acceptance.go` |
| NPC dedicated-action state + pending-saga park | `services/atlas-npc-conversations/.../conversation/processor.go` (`processGachaponActionState` ~934), `model.go` (`GachaponActionModel` ~1358), `model_json.go` (~279/~427), `kafka/consumer/saga/consumer.go` (gachapon arm ~99) |
| NPC saga re-export shim | `services/atlas-npc-conversations/.../npc/saga/model.go` |
| tenants config resource (full recipe) | `services/atlas-tenants/atlas.com/tenants/configuration/*` + `rest/handler.go` (the "vessels" resource) |
| tenants seed data file shape | `services/atlas-tenants/configurations/vessels/*.json` |
| NPC conversation JSON seed | `deploy/seed/gms/83_1/npc-conversations/npc/npc-9100100.json` (gachapon) |

## Service registration checklist (new Go service — 5 edit points)

1. `go.work` — add `./services/atlas-rps/atlas.com/rps`.
2. `.github/config/services.json` — add the go-service entry.
3. `docker-bake.hcl` — add `"atlas-rps"` to the **hand-synced** `go_services` list (HCL can't read JSON — memory `reference_docker_bake_hand_synced`).
4. `deploy/k8s/base/atlas-rps.yaml` + add to `base/kustomization.yaml` resources.
5. Image entries in `overlays/pr/kustomization.yaml` and `overlays/main/kustomization.yaml`.

No Dockerfile edit (parameterized by `ARG SERVICE`; no new shared lib). No LB socket port (REST+Kafka only).

## Opcode reference (from STATUS.md — verified)

| Packet | Dir | v83 | v84 | v87 | v95 | jms185 |
|---|---|---|---|---|---|---|
| RPS_GAME | clientbound | 0x138 | 0x13F | 0x149 | 0x173 | 0x151 |
| RPS_ACTION | serverbound | 0x088 | 0x08C | 0x090 | 0x0A0 | 0x08B |

Mode bytes within each packet are NOT in STATUS.md — derive per version (Tasks 14/16).

## Relevant memory / known-bug guards

- `bug_socket_handler_missing_validator_silently_dropped` — the RPS_ACTION seed handler entry MUST carry `"validator": "LoggedInValidator"` or `BuildHandlerMap` drops it silently.
- `bug_new_opcodes_not_in_live_tenant_config` — seed templates apply only at tenant creation; live tenants need a PATCH + channel restart (Task 20's `live-config-patch.md`).
- `bug_operations_mode_tables_missing_v87_v95_jms` — mode tables are version-dependent; populate each version's from its own IDA switch.
- `feedback_dispatcher_mode_byte_is_false_pass` / `feedback_dispatcher_config_drive_all_modes` — every arm with a body is fully encoded + byte-fixtured; modes always resolved via the `operations` table, never literals; enumerating mode bytes is NOT verification.
- `reference_rediskeyguard_invariant` — run `tools/redis-key-guard.sh` with `GOWORK=off` from repo root; all Redis access through lib types.
- `reference_ida_instance_ports_shifted_idbs_v9` — `list_instances` + match binary NAME before reading; the loaded set rotates.
- `bug_readiness_probe_path_under_api_basepath` — if a readiness probe is added, path is `/api/readyz` (under the REST base path). (Chalkboards/expressions declare none — match them unless the base server auto-mounts readiness.)

## Verification gate (every changed module)

`go test -race ./...`, `go vet ./...`, `go build ./...`; `docker buildx bake atlas-<svc>` per touched `go.mod`; `tools/redis-key-guard.sh` (repo root); `packet-audit dispatcher-lint / matrix --check / fname-doc --check / operations --check` all exit 0; `kustomize build` both overlays. Then `superpowers:requesting-code-review` before PR.

## Changed modules (for the final gate)

`libs/atlas-saga`, `libs/atlas-packet`, `tools/packet-audit`, `services/atlas-rps` (new), `services/atlas-saga-orchestrator`, `services/atlas-channel`, `services/atlas-configurations`, `services/atlas-tenants`, `services/atlas-npc-conversations`. Plus non-code: `go.work`, `.github/config/services.json`, `docker-bake.hcl`, `deploy/k8s/*`, `deploy/seed/*`, `docs/packets/*`.
