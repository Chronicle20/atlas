# task-086 Mount System — Deployment Notes

Operational steps required to roll out the mount feature. Read alongside `plan.md` and `context.md`.

## 1. New service: atlas-mounts

- New Go service `services/atlas-mounts` (REST + Kafka only; **no LB socket ports**). Registered in
  `.github/config/services.json`, `docker-bake.hcl` (`go_services`), `go.work`, and
  `deploy/k8s/base/atlas-mounts.yaml` (+ `kustomization.yaml`).
- **Database:** `DB_NAME: atlas-mounts` (own Postgres DB; GORM auto-migrates `character_mounts`).
- Consumes: `EVENT_TOPIC_CHARACTER_BUFF_STATUS` (registry + SET), `EVENT_TOPIC_CHARACTER_STATUS`
  (logout cleanup + job-change is handled via the cancel-all path, see §4), `EVENT_TOPIC_TAMING_MOB_FOOD` (feed).
- Produces: `EVENT_TOPIC_MOUNT_STATUS`.

## 2. Live tenant config patch (KNOWN PITFALL — applies to EXISTING tenants)

Seed templates apply only at tenant **creation**. Existing tenants will NOT get the new opcodes
automatically (ref memory `bug_new_opcodes_not_in_live_tenant_config`). Per channel, patch live config:

- **Inbound handler:** bind opcode **`0x4D`** → handle `MountFoodHandle` in the live `Socket.Handlers`.
- **Outbound writer:** bind the per-version opcode for writer name `SetTamingMobInfo` in the live `Socket.Writers`.
- **Then restart the channel** — the projection does NOT hot-reload handlers/writers.

Symptom if skipped: feeding the mount no-ops (client → "unhandled message op 0x4D" at info), and the
SET_TAMING_MOB_INFO broadcast silently drops (no level/exp/tiredness UI).

> The `0x4D` inbound opcode and the `SetTamingMobInfo` outbound opcode are the v83 baseline. Confirm
> the per-version opcode values for v87/v92/v95/JMS when patching those tenants (see §5).

## 3. New Kafka topics (env vars)

| Topic | Producer | Consumer |
|---|---|---|
| `COMMAND_TOPIC_TAMING_MOB_FOOD` | atlas-channel | atlas-consumables |
| `EVENT_TOPIC_TAMING_MOB_FOOD` | atlas-consumables | atlas-mounts |
| `EVENT_TOPIC_MOUNT_STATUS` | atlas-mounts | atlas-channel |

Ensure these topics exist / are auto-created per the cluster's Kafka topic policy.

## 4. Behavior change — job change now cancels ALL buffs (FR-4.2, user-approved)

The job-change saga (NPC `change_job` op in atlas-npc-conversations + the GM `@change … job` command in
atlas-messages) now appends a `cancel_all_buffs` step. **Every job change clears all active buffs**
server-wide (matches MapleStory job-advancement behavior). This is the mechanism that dismounts the
MaxInt32-duration mount buff on job change. Operators should be aware this affects all buffs, not just mounts.

## 5. Multi-version (PRE-DEPLOY GATE — Task 41b)

The mount packet **body layouts** are IDA-confirmed for **v83 only**. Before enabling the feature on any
non-v83 tenant (GMS 12/87/92/95, JMS 185), run the cross-version IDA verification (plan Task 41b) on the
v87/v95/JMS IDBs and add version branches where they differ:
1. Monster Riding two-state CTS stat encoding (`character_temporary_stat.go getBaseTemporaryStats`).
2. `SET_TAMING_MOB_INFO` body (`characterId,level,exp,tiredness,levelUp`).
3. Food request `0x4D` body (`ts,slot,itemId`).
Also re-confirm the mount skill ids (1004/1013/1017/1018/1019/1031 + Noblesse/Legend) in JMS skill data.

The architecture is otherwise version-agnostic (opcodes per-tenant config; CTS encoder already version-branched;
skill/item/quest data read per-tenant from each version's WZ).

## 6. FR-9 questline (Riding Mimiana)

- Quest **20523** already exists in WZ (GMS v83) and its `endActions` auto-award saddle **1912005**
  (class 191), taming-mob **1902005** (class 190), and skill **10001004** (MonsterRider), consuming
  quest item **4032117**, on NPC **1102002** (prereq quest **20522**).
- The new NPC conversation `deploy/seed/gms/83_1/npc-conversations/quests/quest-20523.json` starts/completes
  the quest; rewards flow from the WZ `EndActions` via atlas-quest `processEndActions` (no manual award —
  the `suppressAwardAssetByCompleteQuest` dedup would otherwise double-grant).
- **Per-version:** quest-20523 conversation is authored for v83 only. Other tenants that carry quest 20523
  in their WZ need their own per-version conversation seed (deferred to the multi-version pre-deploy pass).
- The mount **engine** works independently of the questline for any character with the (innate) MonsterRider
  skill + an equipped saddle (slot -19) + taming-mob (slot -18).

## 7. Caches / data redeploy

- atlas-data was changed (skill reader emits vehicle ids for skill-only mounts). If atlas-data is
  redeployed, no mount-specific cache clear is required (skill effects are not spawn-cached), but follow the
  standard atlas-data redeploy procedure.

## 8. Verification performed (this branch)

- `go build` / `go vet` / `go test -race` clean across all 8 changed modules (atlas-constants, atlas-packet,
  atlas-channel, atlas-consumables, atlas-data, atlas-messages, atlas-mounts, atlas-npc-conversations).
- `tools/redis-key-guard.sh` (workspace mode) clean (exit 0). Note: `GOWORK=off` variant fails on ~53
  modules with "matched no packages" — a **pre-existing** environmental go.mod-staleness issue present on
  `main`, NOT introduced by this branch; atlas-mounts passes the guard under both the direct analyzer and
  workspace mode.
- `docker buildx bake` clean for all 6 affected services: atlas-mounts, atlas-channel, atlas-data,
  atlas-consumables, atlas-npc-conversations, atlas-messages.
