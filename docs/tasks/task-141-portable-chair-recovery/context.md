# Portable Chair Recovery — Execution Context

Task: task-141-portable-chair-recovery
Companion to: `plan.md` (implementation plan), `design.md` (IDA-verified architecture), `prd.md` (original requirements — note the design **corrected the PRD's premise**; design §1–3 is authoritative).

## The one-paragraph story

Row 562 (`STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST` / `CWvsContext::TryRecovery`) is an **empty, once-per-sit notification** — not the recovery tick. Chair recovery amounts ride the already-implemented `HEAL_OVER_TIME` packet (row 577). So the task is: (a) implement 562 faithfully as an empty codec + logging handler and promote the matrix cell ×5; (b) make chair recovery server-authoritative by rerouting `HEAL_OVER_TIME` from atlas-channel's direct `ChangeHP/MP` calls into a new `RECOVERY` command consumed by atlas-chairs (sit-state check → item-data amounts → 4000 ms/stat rate limit → `CHANGE_HP`/`CHANGE_MP` emission); (c) add `recoveryMP` to atlas-data setups; (d) fix the gms_95 seed template's validator-less handler entries (silently dropped by `BuildHandlerMap`).

## Planning-phase finding (deviation from design, deliberate)

Design §2.4 flagged one validator-less gms_95 entry (`0x64 CharacterHealOverTimeHandle`). Source inspection found **35** validator-less entries in `template_gms_95_1.json` (including `0x14 CharacterLoggedInHandle`). Plan Task 4 fixes all 35 (convention: `CharacterLoggedInHandle`/`StartErrorHandle`/`PongHandle` → `NoOpValidator`, rest → `LoggedInValidator`, verified against gms_83/jms_185 templates). The other four templates have zero missing validators (verified during planning). Template-only effect at tenant creation; live-tenant impact is handled in the runbook.

## Key files (verified during planning)

| What | Where |
|---|---|
| Codec pattern to mirror | `libs/atlas-packet/character/serverbound/chair_portable.go` (+ `_test.go` for marker/fixture style) |
| Heal packet + current handler | `libs/atlas-packet/character/serverbound/heal_over_time.go`; `services/atlas-channel/atlas.com/channel/socket/handler/character_heal_over_time.go` (currently calls `character.NewProcessor(...).ChangeHP/MP` directly — this is what gets rerouted) |
| Channel handler registration | `services/atlas-channel/atlas.com/channel/main.go` ~line 853 (`handlerMap[charsb.CharacterChairPortableHandle] = ...`) |
| Channel chair processor/producer (Use/Cancel pattern for the new Recover) | `services/atlas-channel/atlas.com/channel/chair/{processor,producer}.go` |
| CHANGE_HP/MP command shape to mirror byte-for-byte | `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go` (envelope `{WorldId, CharacterId, Type, Body{ChannelId, Amount}}`) and `character/producer.go` `ChangeHPCommandProvider` |
| Chairs registry model + registry | `services/atlas-chairs/atlas.com/chairs/chair/{model,registry,processor,producer}.go`; registry is `atlas-redis` `TenantRegistry[uint32, Model]`, JSON marshal is hand-written in model.go |
| Chairs consumer to extend | `services/atlas-chairs/atlas.com/chairs/kafka/consumer/chair/consumer.go` (USE/CANCEL arms; add RECOVERY) |
| Chairs message buffer (`message.Emit` pattern) | `services/atlas-chairs/atlas.com/chairs/kafka/message/message.go` |
| REST client to mirror for data/setup | `services/atlas-chairs/atlas.com/chairs/data/map/` (`requests.RootUrl("DATA")`, env `DATA_SERVICE_URL`) |
| atlas-data setup reader/REST | `services/atlas-data/atlas.com/data/setup/{reader,rest}.go` (RecoveryHP exists; add RecoveryMP next to it) |
| Seed templates | `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95,jms_185}_1.json`, path `socket.handlers` |
| packet-audit fname→codec linkage | `tools/packet-audit/cmd/run.go` `candidatesFromFName` (add `case "CWvsContext::TryRecovery"` near the chair bucket, ~line 720) |
| Verification playbook | `docs/packets/audits/VERIFYING_A_PACKET.md` §§6–10 (export flags verified in `tools/packet-audit/cmd/root.go`: `export --version --ida-port --descent-depth --prior-export "" --pending <roster> --output`) |

## IDA ground truth (from design §2.1 — do not re-derive)

| Version | Export key | IDA port | TryRecovery addr (= test marker `ida=`) | Opcode |
|---|---|---|---|---|
| gms_v83 | gms_v83 | 13342 | 0xa02e34 | 0x4A |
| gms_v84 | gms_v84 | 13345 | 0xa4d05a (**unnamed `sub_A4D05A` in IDB — rename first**) | 0x4A |
| gms_v87 | gms_v87 | 13343 | 0xa97e50 | 0x4D |
| gms_v95 | gms_v95 | 13341 | 0x9d4020 | 0x50 |
| jms_v185 | gms_jms_185 (audits dir: **jms_v185**) | 13344 | 0xae6f5a | 0x42 |

- **None of the five committed exports contain `CWvsContext::TryRecovery`** (verified by grep during planning) — evidence pinning fails until each export gets a targeted-harvest surgical splice (plan Task 5 Step 3). Never regenerate a full export.
- Body is empty in all five (COutPacket ctor → SendPacket, zero Encode calls). No clientbound response exists.
- Rate-limit ground truth: HEAL_OVER_TIME cadence is frame-paced (accumulator +30/frame, threshold 10000) → ~5.6–11 s wall-clock; server floor 4000 ms/stat (design §7). Server-internal policy, not DOM-25 territory.

## Architecture decisions already made (do not relitigate)

- **atlas-chairs owns validation** (design §4, FR-5.1). Channel's heal handler emits `RECOVERY` on `COMMAND_TOPIC_CHAIR`; chairs emits `CHANGE_HP`/`CHANGE_MP` on `COMMAND_TOPIC_CHARACTER`. Both env vars already exist in the shared `atlas-env` configmap (`deploy/k8s/base/env-configmap.yaml` — verified; no deploy changes needed).
- **Natural-regen ticks pass through chairs unchanged** (claimed values, `!= 0`, including negative jms clamp corrections) — preserves today's behavior byte-for-byte. Validating natural regen is explicitly out of scope (design §10).
- **562 handler is decode + debug log, no emission** (design §4) — that IS the packet's complete verified semantics, not a stub.
- **Rate-limit timestamps live inside the existing chair registry Model** — `Clear` (stand up) discards them for free (FR-4.5).
- **Fail-closed**: setup-data lookup failure drops the seated tick; never falls back to the claimed value (design §9).

## Gotchas the executor must respect

- Sub-agents must `cd` into this worktree (`.worktrees/task-141-portable-chair-recovery`) first and verify `git branch --show-current` = `task-141-portable-chair-recovery` after each commit.
- Missing `validator` in a socket-handler config entry = handler silently dropped (warning only). Every new/edited entry needs one.
- IDA export files are non-idempotent: splice single entries; verify with `git diff --stat` that only the new entry's lines changed; strip a `COutPacket` delegate call if harvested.
- `matrix --check` currently exits 1 from a pre-existing conflict backlog — the bar is zero NEW problems mentioning StateChangeByPortableChair and no conflict-count increase.
- `docker buildx bake atlas-data atlas-chairs atlas-channel` from the worktree root is mandatory before claiming done; `go build` cannot catch Dockerfile COPY gaps.
- `tools/redis-key-guard.sh` from repo root WITHOUT a `GOWORK=off` prefix.
- Chairs `Clear` in unit tests hits the real Kafka producer (returns error without a broker); tests only depend on registry removal — see plan Task 8 Step 3 note.
- Never `go work sync`; run `go mod tidy` only after imports exist.

## Dependencies between plan tasks

```
Task 1 (data recoveryMP)  ──────────────┐
Task 2 (lib codec) → Task 3 (562 handler) → Task 4 (templates) → Task 5 (audit artifacts; needs live IDA)
Task 6 (registry model) ┐
Task 7 (setup client)   ├→ Task 8 (chairs RECOVERY) → Task 9 (channel reroute) → Task 10 (gates)
Task 1 ─────────────────┘
```

Task 5 is the only task requiring live IDA instances; if one is unreachable it blocks (stop and report — never fake evidence). All other tasks are self-contained in the repo.

## Post-merge rollout (summary; full runbook at the end of plan.md)

data (deploy + canonical re-ingest + baseline re-publish + per-tenant re-ingest where tenant rows exist) → chairs → channel → live tenant config PATCH (`/api/configurations/tenants/{tenantId}` on atlas-configurations, JSON:API envelope) + channel restart → live acceptance on v83 (The Relaxer 3010000 HP; 3010136 HP+MP; stand-up stops; natural regen intact) and v95 (regen restored — intended behavior change).
