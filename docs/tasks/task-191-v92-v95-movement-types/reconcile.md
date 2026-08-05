# task-191 — Live tenant socket-config reconcile (v92 / v95 movement `types`)

Record of the FR-6 data operation that brings **existing** GMS 92.1 and GMS 95.1 tenants up to the
five-handler movement-`types` invariant that the seed templates now satisfy.

> **Why this is needed at all:** seed templates apply **only at tenant provisioning**. Fixing
> `template_gms_92_1.json` / `template_gms_95_1.json` does nothing for a tenant that already exists —
> it keeps its stored socket configuration until someone PATCHes it. This document is the repeatable
> procedure for any environment not covered below.

---

## 0. Read this before you run anything

- **PATCH is a FULL REPLACE** of the tenant configuration JSON
  (`handleUpdateConfigurationTenant` → `UpdateById`). The body must carry the **complete**
  attributes object. Anything you omit is deleted.
- **A wholesale `socket` swap is FORBIDDEN.** Do **not** lift the template's `socket` block into the
  PATCH body. Doing so would also
  - apply unrelated opcode relocations,
  - rewrite the `operations` mode tables (client wire values that carry their own IDA provenance),
  - and silently revert tenant-specific customization.

  Build the body from the **live** config with only the movement-handler entries changed, and prove
  it with a diff before you send it (§5).
- **The PATCH response is NOT evidence.** A handler entry whose `validator` does not resolve is
  accepted at the transport layer and then **silently dropped at load time** —
  `libs/atlas-opcodes/producer.go:65-69` logs `Unable to locate validator [...]` and `continue`s.
  Verify with a fresh `GET` (§7).
- **atlas-channel must be restarted.** Handler/writer maps are built at listener-creation time and the
  configuration projection's `ListenerConfig` diff **excludes** handlers/writers, so a handlers-only
  change does not hot-reload (§8).

---

## 1. Environment

| | |
|---|---|
| Date | 2026-08-04 (log timestamps below are UTC, so they read `2026-08-05T03:…Z`) |
| Cluster | shared dev cluster |
| Namespace | `atlas-main` |
| Ingress host | `dev.atlas.home` |
| Load-balancer IP | `192.168.23.230` |
| Service | `atlas-configurations`, path `/api/configurations/...` |
| Tenant headers | **not** required — the configuration-tenants resource is a bootstrap resource |
| Restarted deployment | `deployment/atlas-channel` |

Scratch snapshots were written outside the repo. `$SCRATCH` below stands for that working directory
(e.g. a per-session temp dir); **never** commit snapshots into the repo.

If the host/IP no longer resolves, rediscover it first:

```bash
kubectl -n atlas-main get ingress,svc -o wide
```

---

## 2. Tenants reconciled

Enumerated with:

```bash
curl -s --resolve dev.atlas.home:80:192.168.23.230 \
  http://dev.atlas.home/api/configurations/tenants | python3 -m json.tool
```

10 configuration tenants exist. The two in scope:

| Tenant id | Region | Version | Note |
|---|---|---|---|
| `db1dbfb3-4345-4731-9223-c40b0c7f6457` | GMS | 92.1 | reconciled |
| `c794c706-aea3-4882-90a6-a3b7ee314f52` | GMS | 95.1 | reconciled |

The other eight (GMS 48.1, 61.1, 72.1, 79.1, 83.1, 84.1, 87.1 and JMS 185.1) were **not** touched —
see §11.

---

## 3. Target state (the five-handler invariant)

Five handlers decode `model.Movement` through the same `options` map and therefore each need a
non-empty `options.types`. The fifth, `NPCActionHandle`, is easy to miss — it decodes movement via
the same path (`libs/atlas-packet/npc/serverbound/action.go:22,62`).

The canonical `types` array is **37 entries** and is read from the repo, never retyped:

```
services/atlas-configurations/seed-data/templates/template_gms_95_1.json
services/atlas-configurations/seed-data/templates/template_gms_92_1.json
```

Both templates' arrays were verified byte-identical to each other and across all five handlers
within each template before use.

Per-tenant target opcodes (all `validator: LoggedInValidator`, `services: ["channel"]`):

| Handler | v92 (GMS 92.1) | v95 (GMS 95.1) |
|---|---|---|
| `CharacterMoveHandle` | `0x2E` | `0x2C` **(new entry)** |
| `PetMovementHandle` | `0xC4` **(new entry)** | `0xC7` |
| `SummonMoveHandle` | `0xCC` (was `0xC8`) | `0xCF` |
| `MonsterMovementHandle` | `0xDC` | `0xE3` |
| `NPCActionHandle` | `0xEA` | `0xF1` |

Non-movement summon opcodes corrected on v92 at the same time (they belong to the same relocated
block): `SummonAttackHandle` `0xC9`→`0xCD`, `SummonDamageHandle` `0xCA`→`0xCE`.

---

## 4. Before state

### v92 — `db1dbfb3-4345-4731-9223-c40b0c7f6457`

46 handlers, 60 writers, already ascending. All validators populated.

```
0x2E CharacterMoveHandle    validator=LoggedInValidator  types=None
0xC8 SummonMoveHandle       validator=LoggedInValidator  types=None
0xC9 SummonAttackHandle     validator=LoggedInValidator  types=None
0xCA SummonDamageHandle     validator=LoggedInValidator  types=None
0xDC MonsterMovementHandle  validator=LoggedInValidator  types=None
0xEA NPCActionHandle        validator=LoggedInValidator  types=None
     PetMovementHandle      ABSENT
```

### v95 — `c794c706-aea3-4882-90a6-a3b7ee314f52`

128 handlers, 215 writers, already ascending.

```
0xC7 PetMovementHandle      validator=''                 types=None
0xCF SummonMoveHandle       validator=LoggedInValidator  types=None
0xD0 SummonAttackHandle     validator=LoggedInValidator  types=None
0xD1 SummonDamageHandle     validator=LoggedInValidator  types=None
0xE3 MonsterMovementHandle  validator=''                 types=None
0xF1 NPCActionHandle        validator=''                 types=None
     CharacterMoveHandle    ABSENT
```

> **Divergence found on v95 that the plan did not anticipate.** 35 of the 128 v95 handlers carried an
> **empty `validator`** — including three of the five movement handlers. Per
> `libs/atlas-opcodes/producer.go:65-69` an empty validator does not resolve, so those handlers were
> being **dropped entirely** at load time. Adding `options.types` to a dropped handler is a no-op, so
> the reconcile could not have taken effect without also setting the validator.
>
> `validator: "LoggedInValidator"` was therefore set on the v95 movement handlers **only** — the value
> is taken from the v95 seed template, not invented. The other 32 empty-validator entries were left
> untouched as out of scope; see §10.

---

## 5. Build the body surgically, then prove the diff

Snapshot first — always, and keep the snapshots:

```bash
mkdir -p "$SCRATCH"
for T in db1dbfb3-4345-4731-9223-c40b0c7f6457 c794c706-aea3-4882-90a6-a3b7ee314f52; do
  curl -s --resolve dev.atlas.home:80:192.168.23.230 \
    "http://dev.atlas.home/api/configurations/tenants/$T" > "$SCRATCH/$T.before.json"
done
```

The build script deep-copies `data.attributes` from the snapshot and mutates only the movement
entries, then re-sorts `handlers` by ascending `opCode`. It **aborts** unless every one of these
holds, so the live shape can never be assumed:

- each template contains exactly one of each of the five handlers, all `types` arrays byte-identical,
  length 37;
- every handler expected to be present is present **at the expected opCode**;
- every handler expected to be absent is absent;
- the destination opCode slot is free (checked against the full handler list);
- the entry does not already carry `options` (refuses to overwrite);
- the existing validator is `''` or `LoggedInValidator` (refuses to change anything else);
- the edit introduces **no new** numeric opCode collision, and the relative order of any
  pre-existing colliding pair survives the re-sort.

Diff the built body against the snapshot **before** PATCHing. Note the diff groups entries by handler
name into lists, because a handler name may legitimately map to several opcodes
(`ServerListRequestHandle`, `NoOpHandler`) — a naive name-keyed dict silently collapses those.

```bash
python3 diff.py "$SCRATCH/$T.before.json" "$SCRATCH/$T.after.json"
```

### v92 diff-proof

```
top-level keys added/removed: set() set()
socket sub-keys added/removed: set() set()
writers unchanged: True
handler count: 46 -> 47
handlers added: ['PetMovementHandle']
handlers removed: []
  handler changed: CharacterMoveHandle | options None->types[37]
  handler changed: MonsterMovementHandle | options None->types[37]
  handler changed: NPCActionHandle | options None->types[37]
  handler changed: SummonAttackHandle | opCode 0xC9->0xCD
  handler changed: SummonDamageHandle | opCode 0xCA->0xCE
  handler changed: SummonMoveHandle | opCode 0xC8->0xCC; options None->types[37]
changed handler names: ['CharacterMoveHandle', 'MonsterMovementHandle', 'NPCActionHandle', 'SummonAttackHandle', 'SummonDamageHandle', 'SummonMoveHandle']
ascending: True
```

### v95 diff-proof

```
top-level keys added/removed: set() set()
socket sub-keys added/removed: set() set()
writers unchanged: True
handler count: 128 -> 129
handlers added: ['CharacterMoveHandle']
handlers removed: []
  handler changed: MonsterMovementHandle | validator ''->'LoggedInValidator'; options None->types[37]
  handler changed: NPCActionHandle | validator ''->'LoggedInValidator'; options None->types[37]
  handler changed: PetMovementHandle | validator ''->'LoggedInValidator'; options None->types[37]
  handler changed: SummonMoveHandle | options None->types[37]
changed handler names: ['MonsterMovementHandle', 'NPCActionHandle', 'PetMovementHandle', 'SummonMoveHandle']
ascending: True
```

Both match the intent exactly: nothing outside `socket`, `writers unchanged: True`, exactly one
handler added, zero removed, only the intended entries changed, ascending. **If a diff shows anything
else, STOP — do not PATCH.**

---

## 6. PATCH

Body shape is the JSON:API envelope `{"data":{"type":"tenants","id":"…","attributes":{…}}}`.

```bash
# v92
curl -s -X PATCH --resolve dev.atlas.home:80:192.168.23.230 \
  -H 'Content-Type: application/vnd.api+json' \
  --data @"$SCRATCH/db1dbfb3-4345-4731-9223-c40b0c7f6457.after.json" \
  "http://dev.atlas.home/api/configurations/tenants/db1dbfb3-4345-4731-9223-c40b0c7f6457" \
  -w '\nHTTP %{http_code}\n'

# v95
curl -s -X PATCH --resolve dev.atlas.home:80:192.168.23.230 \
  -H 'Content-Type: application/vnd.api+json' \
  --data @"$SCRATCH/c794c706-aea3-4882-90a6-a3b7ee314f52.after.json" \
  "http://dev.atlas.home/api/configurations/tenants/c794c706-aea3-4882-90a6-a3b7ee314f52" \
  -w '\nHTTP %{http_code}\n'
```

Result: **HTTP 200** for both. Empty response body.

---

## 7. Read back and verify (the PATCH response is not evidence)

```bash
for T in db1dbfb3-4345-4731-9223-c40b0c7f6457 c794c706-aea3-4882-90a6-a3b7ee314f52; do
  curl -s --resolve dev.atlas.home:80:192.168.23.230 \
    "http://dev.atlas.home/api/configurations/tenants/$T" > "$SCRATCH/$T.readback.json"
done

python3 -c "
import json,sys
d=json.load(open(sys.argv[1]))
hs=d['data']['attributes']['socket']['handlers']
mv={'CharacterMoveHandle','MonsterMovementHandle','PetMovementHandle','SummonMoveHandle','NPCActionHandle'}
found={}
for e in hs:
    if e.get('handler') in mv:
        t=(e.get('options') or {}).get('types')
        found[e['handler']]=(e['opCode'], e.get('validator'), len(t) if t else None)
for k in sorted(mv): print(k, found.get(k,'ABSENT'))
codes=[int(e['opCode'],16) for e in hs]
print('ascending:', codes==sorted(codes))
print('all five present with non-empty types:', len(found)==5 and all(v[2] for v in found.values()))
print('all validators set:', all(v[1] for v in found.values()))
print('all validators == LoggedInValidator:', all(v[1]=='LoggedInValidator' for v in found.values()))
print('all types length 37:', all(v[2]==37 for v in found.values()))
" "$SCRATCH/$T.readback.json"
```

### v92 read-back — quoted output

```
CharacterMoveHandle ('0x2E', 'LoggedInValidator', 37)
MonsterMovementHandle ('0xDC', 'LoggedInValidator', 37)
NPCActionHandle ('0xEA', 'LoggedInValidator', 37)
PetMovementHandle ('0xC4', 'LoggedInValidator', 37)
SummonMoveHandle ('0xCC', 'LoggedInValidator', 37)
ascending: True
all five present with non-empty types: True
all validators set: True
all validators == LoggedInValidator: True
all types length 37: True
```

### v95 read-back — quoted output

```
CharacterMoveHandle ('0x2C', 'LoggedInValidator', 37)
MonsterMovementHandle ('0xE3', 'LoggedInValidator', 37)
NPCActionHandle ('0xF1', 'LoggedInValidator', 37)
PetMovementHandle ('0xC7', 'LoggedInValidator', 37)
SummonMoveHandle ('0xCF', 'LoggedInValidator', 37)
ascending: True
all five present with non-empty types: True
all validators set: True
all validators == LoggedInValidator: True
all types length 37: True
```

Round-trip integrity also checked — the read-back `attributes` are **identical** to the bytes sent,
and re-running the §5 diff against the read-back reproduces the same intended-changes-only result:

```
db1dbfb3-4345-4731-9223-c40b0c7f6457 : attributes identical to what was sent: True
c794c706-aea3-4882-90a6-a3b7ee314f52 : attributes identical to what was sent: True
db1dbfb3-4345-4731-9223-c40b0c7f6457 writers identical before vs live-readback: True (count 60)
c794c706-aea3-4882-90a6-a3b7ee314f52 writers identical before vs live-readback: True (count 215)
```

Relocated v92 summon block, confirmed live:

```
0xCC SummonMoveHandle LoggedInValidator
0xCD SummonAttackHandle LoggedInValidator
0xCE SummonDamageHandle LoggedInValidator
```

---

## 8. Restart atlas-channel

```bash
kubectl -n atlas-main rollout restart deployment/atlas-channel
kubectl -n atlas-main rollout status deployment/atlas-channel
```

```
restart initiated (UTC): 2026-08-05T03:09:46Z
deployment.apps/atlas-channel restarted
Waiting for deployment "atlas-channel" rollout to finish: 1 old replicas are pending termination...
deployment "atlas-channel" successfully rolled out
```

New pod `atlas-channel-78774898bf-kvgdt`, `1/1 Running`.

### `Configuring opcode` evidence (the positive signal)

`libs/atlas-opcodes/producer.go:83`. Both target tenants register all five movement handlers plus the
relocated summon block, each with a resolved validator:

```
92.1 db1dbfb3 Configuring opcode [0x2E] with validator [LoggedInValidator] and handler [CharacterMoveHandle].
92.1 db1dbfb3 Configuring opcode [0xC4] with validator [LoggedInValidator] and handler [PetMovementHandle].
92.1 db1dbfb3 Configuring opcode [0xCC] with validator [LoggedInValidator] and handler [SummonMoveHandle].
92.1 db1dbfb3 Configuring opcode [0xCD] with validator [LoggedInValidator] and handler [SummonAttackHandle].
92.1 db1dbfb3 Configuring opcode [0xCE] with validator [LoggedInValidator] and handler [SummonDamageHandle].
92.1 db1dbfb3 Configuring opcode [0xDC] with validator [LoggedInValidator] and handler [MonsterMovementHandle].
92.1 db1dbfb3 Configuring opcode [0xEA] with validator [LoggedInValidator] and handler [NPCActionHandle].

95.1 c794c706 Configuring opcode [0x2C] with validator [LoggedInValidator] and handler [CharacterMoveHandle].
95.1 c794c706 Configuring opcode [0xC7] with validator [LoggedInValidator] and handler [PetMovementHandle].
95.1 c794c706 Configuring opcode [0xCF] with validator [LoggedInValidator] and handler [SummonMoveHandle].
95.1 c794c706 Configuring opcode [0xD0] with validator [LoggedInValidator] and handler [SummonAttackHandle].
95.1 c794c706 Configuring opcode [0xD1] with validator [LoggedInValidator] and handler [SummonDamageHandle].
95.1 c794c706 Configuring opcode [0xE3] with validator [LoggedInValidator] and handler [MonsterMovementHandle].
95.1 c794c706 Configuring opcode [0xF1] with validator [LoggedInValidator] and handler [NPCActionHandle].
```

This is the load-time proof that the three previously-empty v95 validators now resolve — before the
PATCH those three handlers would have been dropped instead of configured.

### `listener.added` evidence

All 10 tenants got a listener, including both targets:

```
{"@timestamp":"2026-08-05T03:09:59.723Z","key":{"TenantId":"c794c706-aea3-4882-90a6-a3b7ee314f52","WorldId":0,"ChannelId":0},"log.level":"info","message":"listener.added","service.name":"atlas-channel"}
{"@timestamp":"2026-08-05T03:09:59.859Z","key":{"TenantId":"db1dbfb3-4345-4731-9223-c40b0c7f6457","WorldId":0,"ChannelId":0},"log.level":"info","message":"listener.added","service.name":"atlas-channel"}
```

---

## 9. Negative signal — the error line is gone

`movementPathAttrFromOptions` logs `Code [%d] not configured for use in movement…` at **error** level,
once per fragment, when a movement handler has no `types` table. After the reconcile:

```bash
kubectl -n atlas-main logs deployment/atlas-channel --since=10m | grep -c "not configured for use in movement" || echo 0
```

```
0
```

Total error-level lines since the restart: `0`.

> **Honest caveat on the strength of this signal.** The post-restart window had **zero client sessions**
> on the cluster, so `0` here is consistent with the fix but is not *by itself* proof that the decode
> path is exercised — absence of traffic also produces `0`. The load-bearing evidence is the §7
> read-back (37-entry `types` on all five handlers) plus the §8 `Configuring opcode` lines showing all
> five handlers actually registered. A live v92/v95 client session remains the strongest confirmation
> and should be re-checked with the same grep next time one connects.

---

## 10. Pre-existing defects observed and deliberately NOT fixed

These were found while reading the live config. None is caused by this PATCH; all are out of scope for
task-191 and are recorded here so they are not lost.

1. **v95: 32 handlers still have an empty `validator`** and are dropped at load time
   (`libs/atlas-opcodes/producer.go:65-69`). 31 of them are `channel`-scoped and produce one warning
   each; the 32nd (`StartErrorHandle`) is `login`-scoped so atlas-channel never evaluates it —
   31 warnings observed, which reconciles exactly. Affected handlers include
   `CharacterLoggedInHandle`, `ChannelChangeHandle`, `CharacterChatGeneralHandle`,
   `CharacterInventoryMoveHandle`, `CharacterItemUseHandle`, `DropPickUpHandle`, `PartyOperationHandle`,
   `QuestActionHandle`, `ReactorHitHandle`, and most of the `Pet*` family. **v92 has zero.**
   This is a real functional gap on the v95 tenant and deserves its own task.

2. **v95: duplicate numeric opCode `0x9C`** — `"0x09C"` (`EnterDoorHandle`) and `"0x9C"` (`UseDoor`)
   parse to the same `uint16`, so `BuildHandlerMap` writes both into `result[0x9C]` and the later one
   wins. Left untouched; the build script explicitly asserts the re-sort preserves their relative
   order so behaviour is unchanged.

3. **Benign, fleet-wide:** `Service declares writer [...] but tenant config has no opcode mapping for it.`
   — 515 distinct instances across all 10 tenants. These are template-completeness gaps on the
   *writers* side. `writers` were verified byte-identical before and after on both tenants (§7), so the
   PATCH cannot have caused them. Movement-related instances: `CharacterMovement` and `PetMovement` on
   GMS 48.1, and `PetMovement` on GMS 92.1.

4. Duplicate handler **names** at different opcodes (`ServerListRequestHandle` at `0x04`/`0x0B`,
   `NoOpHandler` at several) are legitimate multi-opcode mappings, not defects. They matter only
   because a name-keyed diff would hide them.

---

## 11. Environments deliberately NOT reconciled

| Scope | Why not |
|---|---|
| The other 8 tenants on this cluster (GMS 48.1, 61.1, 72.1, 79.1, 83.1, 84.1, 87.1; JMS 185.1) | Out of scope for task-191, which covers v92 and v95 only. The post-restart logs confirm all eight already register all five movement handlers with `LoggedInValidator`, so none is affected by the bug this task fixes. |
| Any other cluster / environment | Not reachable from this session and not enumerated. Re-run §2 → §9 against that environment's ingress. Any environment whose v92/v95 tenants were provisioned **before** the template fix landed needs this same reconcile. |
| Newly provisioned tenants | Not needed — they seed from the fixed templates and satisfy the invariant at creation. |

---

## 12. Repeating this elsewhere — checklist

1. Enumerate tenants; record the ids you actually observe (§2). Do not trust remembered ids.
2. Snapshot every tenant you intend to touch, and keep the snapshots (§5).
3. Read the **live** handler set. Do not assume it matches the template — on this cluster v92 was
   missing `PetMovementHandle`, v95 was missing `CharacterMoveHandle`, v92's summon block was at the
   wrong opcodes, and v95 had empty validators. If a handler you expect is missing or an opcode
   differs, stop and investigate rather than inventing an entry.
4. Build the body from the live config, changing only movement entries; re-sort by ascending `opCode`.
5. Diff against the snapshot and confirm: nothing outside `socket`, `writers unchanged: True`,
   exactly the intended handlers changed, ascending. **Anything else → stop.**
6. PATCH (expect HTTP 200). The response is not evidence.
7. Fresh `GET` read-back; require all five handlers present, `validator=LoggedInValidator`,
   `types` length 37, ascending.
8. `kubectl -n <ns> rollout restart deployment/atlas-channel` and confirm the `Configuring opcode`
   lines for all five handlers.
9. Grep for `not configured for use in movement` — expect `0`, and note whether there was live traffic.
