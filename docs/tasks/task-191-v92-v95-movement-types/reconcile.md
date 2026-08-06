# task-191 — Live tenant socket-config reconcile (movement `types`)

Record of the FR-6 data operation that brings **existing** tenants up to the five-handler movement-
`types` invariant that the seed templates now satisfy.

Two passes are recorded here:

| Pass | Date (UTC) | Tenants | Scope |
|---|---|---|---|
| 1 | 2026-08-05T03:09Z | GMS 92.1, GMS 95.1 | FR-6 — the task's primary target |
| 2 | 2026-08-05T09:30Z | GMS 48.1 | FR-4's live counterpart, found by the fleet sweep in §11 |

> **Why this is needed at all:** seed templates apply **only at tenant provisioning**. Fixing
> `template_gms_92_1.json` / `template_gms_95_1.json` / `template_gms_48_1.json` does nothing for a
> tenant that already exists — it keeps its stored socket configuration until someone PATCHes it.
> `tools/template-movement-types-guard.sh` keeps the *templates* correct going forward; it cannot see
> live tenants. This document is the repeatable procedure for those.

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
- **The `types` array is version-specific — never copy one version's table to another.** Observed
  lengths across this fleet: 23 (v48/61/72/79/83), 24 (v84), 25 (v87), 33 (jms185), 37 (v92/v95).
  Source the array from the same version, and assert its length before use.
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
| Dates | pass 1 — 2026-08-04 local / `2026-08-05T03:09Z`; pass 2 (v48) — 2026-08-05 local / `2026-08-05T09:30Z`. UTC timestamps below are authoritative. |
| Cluster | shared dev cluster |
| Namespace | `atlas-main` |
| Ingress host | `dev.atlas.home` |
| Load-balancer IP | `192.168.23.230` |
| Service | `atlas-configurations`, path `/api/configurations/...` |
| Tenant headers | **not** required — the configuration-tenants resource is a bootstrap resource |
| Restarted deployment | `deployment/atlas-channel` (once per pass) |

Scratch snapshots were written outside the repo. `$SCRATCH` below stands for that working directory
(e.g. a per-session temp dir); **never** commit snapshots into the repo. Export it once:

```bash
export SCRATCH=/some/scratch/dir && mkdir -p "$SCRATCH"
```

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

10 configuration tenants exist. The three reconciled:

| Tenant id | Region | Version | Pass | What was wrong |
|---|---|---|---|---|
| `db1dbfb3-4345-4731-9223-c40b0c7f6457` | GMS | 92.1 | 1 | no `types` on 4 handlers; `PetMovementHandle` absent; summon block at wrong opcodes |
| `c794c706-aea3-4882-90a6-a3b7ee314f52` | GMS | 95.1 | 1 | no `types` on 4 handlers; `CharacterMoveHandle` absent; 3 empty validators |
| `e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd` | GMS | 48.1 | 2 | no `types` on `SummonMoveHandle` |

The remaining seven were verified clean at the `types` level and left untouched — see §11.

---

## 3. Target state (the five-handler invariant)

Five handlers decode `model.Movement` through the same `options` map and therefore each need a
non-empty `options.types`. The fifth, `NPCActionHandle`, is easy to miss — it decodes movement via
the same path (`libs/atlas-packet/npc/serverbound/action.go:22,62`).

Per-tenant target opcodes (all `validator: LoggedInValidator`, `services: ["channel"]`):

| Handler | v48 (GMS 48.1) | v92 (GMS 92.1) | v95 (GMS 95.1) |
|---|---|---|---|
| `CharacterMoveHandle` | `0x21` | `0x2E` | `0x2C` **(new entry)** |
| `PetMovementHandle` | `0x71` | `0xC4` **(new entry)** | `0xC7` |
| `SummonMoveHandle` | `0x78` | `0xCC` (was `0xC8`) | `0xCF` |
| `MonsterMovementHandle` | `0x81` | `0xDC` | `0xE3` |
| `NPCActionHandle` | `0x8A` | `0xEA` | `0xF1` |
| **`types` length** | **23** | **37** | **37** |

Non-movement summon opcodes corrected on v92 at the same time (they belong to the same relocated
block): `SummonAttackHandle` `0xC9`→`0xCD`, `SummonDamageHandle` `0xCA`→`0xCE`.

### Where the `types` array comes from

- **v92 / v95** — read from the repo, never retyped:
  `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` and
  `template_gms_92_1.json`. Both arrays were verified byte-identical to each other and across all
  five handlers within each template, length 37, before use.
- **v48** — sourced from **that same tenant's own live `CharacterMoveHandle` (`0x21`)**, because the
  other four v48 handlers already carried the correct table and only `SummonMoveHandle` was missing
  it. All four existing v48 arrays were verified byte-identical to each other, length 23, before the
  copy. As an independent cross-check (not the source), `template_gms_48_1.json` was confirmed to
  carry the identical 23-entry array at all five handlers with matching opcodes.

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

### v48 — `e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd`

81 handlers, 62 writers, already ascending, no duplicate opcodes, **zero** empty validators.

```
0x21 CharacterMoveHandle    validator=LoggedInValidator  types=23
0x71 PetMovementHandle      validator=LoggedInValidator  types=23
0x78 SummonMoveHandle       validator=LoggedInValidator  types=None   <-- the only gap
0x79 SummonAttackHandle     validator=LoggedInValidator  types=None   (not a movement handler)
0x7A SummonDamageHandle     validator=LoggedInValidator  types=None   (not a movement handler)
0x81 MonsterMovementHandle  validator=LoggedInValidator  types=23
0x8A NPCActionHandle        validator=LoggedInValidator  types=23
```

A single missing table. No opcode changes, no new entries, no validator changes were needed.

---

## 5. Build the body surgically, then prove the diff

Snapshot first — always, and keep the snapshots. **Note the analysis runs *inside* the loop**: after a
`for` loop terminates, `$T` holds only the last id, so a copy-paste that analyses outside the loop
silently checks one tenant and skips the rest.

```bash
TENANTS="db1dbfb3-4345-4731-9223-c40b0c7f6457 c794c706-aea3-4882-90a6-a3b7ee314f52"

for T in $TENANTS; do
  curl -s --resolve dev.atlas.home:80:192.168.23.230 \
    "http://dev.atlas.home/api/configurations/tenants/$T" > "$SCRATCH/$T.before.json"
done

python3 build.py            # writes $SCRATCH/<id>.after.json for both tenants

for T in $TENANTS; do
  echo "=== $T ==="
  python3 diff.py "$SCRATCH/$T.before.json" "$SCRATCH/$T.after.json"
done
```

For the v48 pass, substitute `TENANTS=e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd` and `build_v48.py`.
Full source of `build.py`, `build_v48.py` and `diff.py` is embedded in **Appendix A** — they are the
safety-critical core of this procedure and must not be reconstructed from memory.

The build scripts **abort** unless every one of these holds, so the live shape is never assumed:

- the source `types` array is the expected length for that version (37 for v92/v95, 23 for v48), and
  every handler that already has a table agrees with it byte-for-byte;
- every handler expected to be present is present **at the expected opCode**;
- every handler expected to be absent is absent;
- the destination opCode slot is free (checked against the full handler list);
- the entry does not already carry `options` (refuses to overwrite);
- the existing validator is `''` or `LoggedInValidator` (refuses to change anything else);
- the edit introduces **no new** numeric opCode collision, and the relative order of any pre-existing
  colliding pair survives the re-sort;
- (v48 only) the opCode list is completely unchanged — that edit must not reorder or relocate anything.

The diff groups entries by handler name into **lists**, because a handler name may legitimately map to
several opcodes (`ServerListRequestHandle` at `0x04`/`0x0B`, `NoOpHandler` at several) — a naive
name-keyed dict silently collapses those.

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

### v48 diff-proof

```
top-level keys added/removed: set() set()
socket sub-keys added/removed: set() set()
writers unchanged: True
handler count: 81 -> 81
handlers added: []
handlers removed: []
  handler changed: SummonMoveHandle | options None->types[23]
changed handler names: ['SummonMoveHandle']
ascending: True
```

All three match the intent exactly: nothing outside `socket`, `writers unchanged: True`, only the
intended entries changed, ascending. v48 additionally added **zero** handlers, as required.
**If a diff shows anything else, STOP — do not PATCH.**

---

## 6. PATCH

Body shape is the JSON:API envelope `{"data":{"type":"tenants","id":"…","attributes":{…}}}`.

```bash
for T in $TENANTS; do
  echo "=== $T ==="
  curl -s -X PATCH --resolve dev.atlas.home:80:192.168.23.230 \
    -H 'Content-Type: application/vnd.api+json' \
    --data @"$SCRATCH/$T.after.json" \
    "http://dev.atlas.home/api/configurations/tenants/$T" \
    -w '\nHTTP %{http_code}\n'
done
```

Result: **HTTP 200** for all three tenants. Empty response body.

---

## 7. Read back and verify (the PATCH response is not evidence)

Again, the verification runs **inside** the loop. `EXPECT_LEN` is the version's array length — 37 for
v92/v95, 23 for v48.

```bash
EXPECT_LEN=37   # 23 for the v48 pass

for T in $TENANTS; do
  curl -s --resolve dev.atlas.home:80:192.168.23.230 \
    "http://dev.atlas.home/api/configurations/tenants/$T" > "$SCRATCH/$T.readback.json"

  echo "=== $T ==="
  python3 -c "
import json,sys
d=json.load(open(sys.argv[1])); want=int(sys.argv[2])
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
print('all types length %d:' % want, all(v[2]==want for v in found.values()))
" "$SCRATCH/$T.readback.json" "$EXPECT_LEN"
done
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

### v48 read-back — quoted output

```
CharacterMoveHandle ('0x21', 'LoggedInValidator', 23)
MonsterMovementHandle ('0x81', 'LoggedInValidator', 23)
NPCActionHandle ('0x8A', 'LoggedInValidator', 23)
PetMovementHandle ('0x71', 'LoggedInValidator', 23)
SummonMoveHandle ('0x78', 'LoggedInValidator', 23)
ascending: True
all five present with non-empty types: True
all validators set: True
all validators == LoggedInValidator: True
all types length 23: True
```

Round-trip integrity also checked — the read-back `attributes` are **identical** to the bytes sent,
and re-running the §5 diff against the read-back reproduces the same intended-changes-only result:

```
db1dbfb3-4345-4731-9223-c40b0c7f6457 : attributes identical to what was sent: True
c794c706-aea3-4882-90a6-a3b7ee314f52 : attributes identical to what was sent: True
e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd : attributes identical to what was sent: True

db1dbfb3-4345-4731-9223-c40b0c7f6457 writers identical before vs live-readback: True (count 60)
c794c706-aea3-4882-90a6-a3b7ee314f52 writers identical before vs live-readback: True (count 215)
e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd writers identical before vs live-readback: True (count 62)
                                     handler count before -> readback: 81 -> 81
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

**Pass 1 (v92/v95)** — initiated `2026-08-05T03:09:46Z`, new pod `atlas-channel-78774898bf-kvgdt`:

```
deployment.apps/atlas-channel restarted
Waiting for deployment "atlas-channel" rollout to finish: 1 old replicas are pending termination...
deployment "atlas-channel" successfully rolled out
```

**Pass 2 (v48)** — initiated `2026-08-05T09:30:23Z`, new pod `atlas-channel-557479645c-9ztb4`:

```
deployment.apps/atlas-channel restarted
Waiting for deployment "atlas-channel" rollout to finish: 1 old replicas are pending termination...
deployment "atlas-channel" successfully rolled out
```

> Startup logs roll off within hours on this cluster — capture the evidence below immediately after
> the rollout, and give the pod a few seconds before grepping or you will race the config load and see
> an empty result.

### `Configuring opcode` evidence (the positive signal)

`libs/atlas-opcodes/producer.go:83`. Each reconciled tenant registers all five movement handlers plus
the summon block, each with a resolved validator:

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

48.1 e1f06ae2 Configuring opcode [0x21] with validator [LoggedInValidator] and handler [CharacterMoveHandle].
48.1 e1f06ae2 Configuring opcode [0x71] with validator [LoggedInValidator] and handler [PetMovementHandle].
48.1 e1f06ae2 Configuring opcode [0x78] with validator [LoggedInValidator] and handler [SummonMoveHandle].
48.1 e1f06ae2 Configuring opcode [0x79] with validator [LoggedInValidator] and handler [SummonAttackHandle].
48.1 e1f06ae2 Configuring opcode [0x7A] with validator [LoggedInValidator] and handler [SummonDamageHandle].
48.1 e1f06ae2 Configuring opcode [0x81] with validator [LoggedInValidator] and handler [MonsterMovementHandle].
48.1 e1f06ae2 Configuring opcode [0x8A] with validator [LoggedInValidator] and handler [NPCActionHandle].
```

For v95 this is the load-time proof that the three previously-empty validators now resolve — before
the PATCH those three handlers would have been dropped instead of configured.

After pass 2, all three reconciled tenants were re-confirmed in the same pod's logs:

```
48.1: 5/5 movement handlers registered
92.1: 5/5 movement handlers registered
95.1: 5/5 movement handlers registered
```

### `listener.added` evidence

All 10 tenants got a listener in both passes. Pass 2, all ten:

```
ec876921-c363-4cc6-9c51-5bb8d57f9553 2026-08-05T09:30:31.231Z
92adbe47-5ada-4f3b-8224-f58c80a4a2d5 2026-08-05T09:30:31.245Z
48d415ca-59de-4953-9aed-0c4156a09bc9 2026-08-05T09:30:31.252Z
0d250dc9-64c4-45ae-8bc2-fc0a9cdb5578 2026-08-05T09:30:31.280Z
db1dbfb3-4345-4731-9223-c40b0c7f6457 2026-08-05T09:30:31.289Z
86da65d2-b9fa-4176-985a-6a5df586220c 2026-08-05T09:30:31.294Z
4936dff2-7121-4f46-b9eb-1ae541f4a85f 2026-08-05T09:30:31.304Z
c794c706-aea3-4882-90a6-a3b7ee314f52 2026-08-05T09:30:31.311Z
abedf3b4-1d7c-4b3b-bc52-70f62ab09418 2026-08-05T09:30:31.318Z
e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd 2026-08-05T09:30:31.336Z
```

---

## 9. Negative signal — the error line is gone

`movementPathAttrFromOptions` logs `Code [%d] not configured for use in movement…` at **error** level,
once per fragment, when a movement handler has no `types` table. After each reconcile:

```bash
kubectl -n atlas-main logs deployment/atlas-channel --since=10m | grep -c "not configured for use in movement" || echo 0
```

```
0
```

Both passes: `0`. Total error-level lines since each restart: `0`.

> **Honest caveat on the strength of this signal — applies to v92, v95 and v48 alike.** Both
> post-restart windows had **zero client sessions** on the cluster, so `0` here is consistent with the
> fix but is not *by itself* proof that the decode path is exercised — absence of traffic also
> produces `0`. The load-bearing evidence is the §7 read-back (correct-length `types` on all five
> handlers) plus the §8 `Configuring opcode` lines showing all five handlers actually registered. A
> live client session on the affected version remains the strongest confirmation and should be
> re-checked with the same grep next time one connects.

---

## 10. Pre-existing defects observed and deliberately NOT fixed

These were found while reading the live configs. None is caused by these PATCHes; all are out of scope
for task-191 and are recorded here so they are not lost.

1. **v95: 32 handlers still have an empty `validator`** and are dropped at load time
   (`libs/atlas-opcodes/producer.go:65-69`). 31 of them are `channel`-scoped and produce one warning
   each; the 32nd (`StartErrorHandle`) is `login`-scoped so atlas-channel never evaluates it —
   31 warnings observed, which reconciles exactly. Affected handlers include
   `CharacterLoggedInHandle`, `ChannelChangeHandle`, `CharacterChatGeneralHandle`,
   `CharacterInventoryMoveHandle`, `CharacterItemUseHandle`, `DropPickUpHandle`, `PartyOperationHandle`,
   `QuestActionHandle`, `ReactorHitHandle`, and most of the `Pet*` family. **v92 and v48 have zero.**
   This is a real functional gap on the v95 tenant and deserves its own task.

2. **v95: duplicate numeric opCode `0x9C`** — `"0x09C"` (`EnterDoorHandle`) and `"0x9C"` (`UseDoor`)
   parse to the same `uint16`, so `BuildHandlerMap` writes both into `result[0x9C]` and the later one
   wins. Left untouched; the build script explicitly asserts the re-sort preserves their relative
   order so behaviour is unchanged.

3. **Benign, fleet-wide:** `Service declares writer [...] but tenant config has no opcode mapping for it.`
   — 515 distinct instances across all 10 tenants. These are template-completeness gaps on the
   *writers* side. `writers` were verified byte-identical before and after on every tenant patched
   (§7), so the PATCHes cannot have caused them. Movement-related instances: `CharacterMovement` and
   `PetMovement` on GMS 48.1, and `PetMovement` on GMS 92.1 — these are **clientbound writer** opcode
   gaps, unrelated to the serverbound handler `types` tables this task fixes.

4. Duplicate handler **names** at different opcodes (`ServerListRequestHandle` at `0x04`/`0x0B`,
   `NoOpHandler` at several) are legitimate multi-opcode mappings, not defects. They matter only
   because a name-keyed diff would hide them.

---

## 11. Fleet-wide sweep, and what was NOT reconciled

An earlier revision of this document asserted that the non-target tenants were unaffected. **That
claim was wrong** — it was based on `Configuring opcode` log lines, which only prove a handler is
*registered*, not that it has a `types` table. The two are different, and conflating them is exactly
the sort of overclaim this project forbids. It hid a real gap on GMS 48.1.

The claim has been replaced with an actual `types`-level sweep of all 10 tenants. Reproduce it with
`sweep.sh` (Appendix A):

```bash
SCRATCH=/some/scratch/dir sh sweep.sh
```

Result **after** both reconcile passes:

```
REG  VER    TENANT                                 PRESENT  TYPES-LENS  GAPS
GMS  48.0   e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd   5/5      [23]        none
GMS  61.0   0d250dc9-64c4-45ae-8bc2-fc0a9cdb5578   5/5      [23]        none
GMS  72.0   48d415ca-59de-4953-9aed-0c4156a09bc9   5/5      [23]        none
GMS  79.0   92adbe47-5ada-4f3b-8224-f58c80a4a2d5   5/5      [23]        none
GMS  83.0   ec876921-c363-4cc6-9c51-5bb8d57f9553   5/5      [23]        none
GMS  84.0   4936dff2-7121-4f46-b9eb-1ae541f4a85f   5/5      [24]        none
GMS  87.0   86da65d2-b9fa-4176-985a-6a5df586220c   5/5      [25]        none
GMS  92.0   db1dbfb3-4345-4731-9223-c40b0c7f6457   5/5      [37]        none
GMS  95.0   c794c706-aea3-4882-90a6-a3b7ee314f52   5/5      [37]        none
JMS  185.0  abedf3b4-1d7c-4b3b-bc52-70f62ab09418   5/5      [33]        none

tenants with a movement-types gap: 0
```

The same sweep run **before** the v48 pass reported `[23, None]` with
`GAPS ['SummonMoveHandle']` on GMS 48.1 and `none` for the other nine — that one gap is what pass 2
fixed.

| Scope | Status |
|---|---|
| GMS 61.1, 72.1, 79.1, 83.1, 84.1, 87.1, JMS 185.1 | **Not reconciled — verified not to need it.** Each has all five movement handlers present, with a validator and a non-empty version-appropriate `types` array (23/24/25/33 as shown). Verified at the `types` level by the sweep above, not inferred from log lines. |
| Any other cluster / environment | **Not reconciled — not reachable from this session and not enumerated.** Re-run §2 → §9 against that environment's ingress. Any environment whose tenants were provisioned **before** the corresponding template fix landed needs this same reconcile; run `sweep.sh` there first to find out which. |
| Newly provisioned tenants | **Not needed** — they seed from the fixed templates and satisfy the invariant at creation. |

---

## 12. Repeating this elsewhere — checklist

1. Enumerate tenants; record the ids you actually observe (§2). Do not trust remembered ids.
2. Run `sweep.sh` to find which tenants actually have a gap. Check `types`, not log lines — a handler
   can be registered and still have no table.
3. Snapshot every tenant you intend to touch, and keep the snapshots (§5).
4. Read the **live** handler set. Do not assume it matches the template — on this cluster v92 was
   missing `PetMovementHandle`, v95 was missing `CharacterMoveHandle`, v92's summon block was at the
   wrong opcodes, and v95 had empty validators. If a handler you expect is missing or an opcode
   differs, stop and investigate rather than inventing an entry.
5. Source the `types` array from the **same version** and assert its length (§3). Never reuse another
   version's table.
6. Build the body from the live config, changing only movement entries; re-sort by ascending `opCode`.
7. Diff against the snapshot and confirm: nothing outside `socket`, `writers unchanged: True`,
   exactly the intended handlers changed, ascending. **Anything else → stop.**
8. PATCH (expect HTTP 200). The response is not evidence.
9. Fresh `GET` read-back; require all five handlers present, `validator=LoggedInValidator`,
   correct `types` length, ascending.
10. `kubectl -n <ns> rollout restart deployment/atlas-channel` and confirm the `Configuring opcode`
    lines for all five handlers.
11. Grep for `not configured for use in movement` — expect `0`, and record whether there was live
    traffic, because with none the `0` is weak evidence.
12. Re-run `sweep.sh` to confirm the fleet is clean.

---

## Appendix A — scripts

These are the scripts that were actually run. The **only** modification from the versions executed is
that the hard-coded scratch-path constant now reads `$SCRATCH` from the environment, since the
original session's scratch directory is ephemeral.

### `build.py` — v92 / v95

```python
#!/usr/bin/env python3
"""Build the reconciled v92/v95 tenant socket configs surgically.

Reads the LIVE snapshot, mutates ONLY the five movement-carrying handler entries,
re-sorts handlers by ascending opCode, writes the JSON:API PATCH body.
Aborts loudly on any deviation from the expected live shape.
"""
import copy
import json
import os
import sys

TPL = "services/atlas-configurations/seed-data/templates/template_gms_%s_1.json"
SNAP = os.environ["SCRATCH"] + "/%s.%s.json"

V92 = "db1dbfb3-4345-4731-9223-c40b0c7f6457"
V95 = "c794c706-aea3-4882-90a6-a3b7ee314f52"

MOVE = ("CharacterMoveHandle", "MonsterMovementHandle", "PetMovementHandle",
        "SummonMoveHandle", "NPCActionHandle")


def die(msg):
    print("ABORT: %s" % msg)
    sys.exit(1)


def canonical_types():
    arrs = []
    for v in ("92", "95"):
        t = json.load(open(TPL % v))
        for h in MOVE:
            got = [e for e in t["socket"]["handlers"] if e.get("handler") == h]
            if len(got) != 1:
                die("template gms_%s_1: expected exactly 1 %s, got %d" % (v, h, len(got)))
            arrs.append(got[0]["options"]["types"])
    if not all(a == arrs[0] for a in arrs):
        die("template types arrays are not byte-identical")
    if len(arrs[0]) != 37:
        die("expected 37 types entries, got %d" % len(arrs[0]))
    return arrs[0]


RELEVANT = set(MOVE) | {"SummonAttackHandle", "SummonDamageHandle"}


def index(handlers):
    """Map handler-name -> entry. Duplicate names are legitimate for multi-opcode
    handlers (ServerListRequestHandle, NoOpHandler); only the entries this task
    touches must be unique."""
    idx = {}
    for e in handlers:
        h = e.get("handler")
        if h in idx:
            if h in RELEVANT:
                die("movement-relevant handler %s appears more than once" % h)
            continue
        idx[h] = e
    return idx


def expect_present(idx, handler, opcode):
    e = idx.get(handler)
    if e is None:
        die("expected handler %s to be present on the live tenant, it is ABSENT" % handler)
    if e["opCode"].lower() != opcode.lower():
        die("handler %s: expected live opCode %s, found %s" % (handler, opcode, e["opCode"]))
    return e


def expect_absent(idx, handler):
    if handler in idx:
        die("expected handler %s to be ABSENT on the live tenant, it is present at %s"
            % (handler, idx[handler]["opCode"]))


def set_types(e, types):
    opts = e.get("options")
    if opts not in (None, {}):
        die("handler %s already carries options %r - refusing to overwrite"
            % (e.get("handler"), opts))
    e["options"] = {"types": copy.deepcopy(types)}


def set_validator(e):
    """Movement handlers must resolve a validator or BuildHandlerMap drops them
    (libs/atlas-opcodes/producer.go:65-69). Template value is LoggedInValidator."""
    cur = e.get("validator")
    if cur not in ("", "LoggedInValidator"):
        die("handler %s has unexpected validator %r - refusing to change"
            % (e.get("handler"), cur))
    e["validator"] = "LoggedInValidator"


def new_entry(handler, opcode, types):
    return {
        "opCode": opcode,
        "validator": "LoggedInValidator",
        "handler": handler,
        "services": ["channel"],
        "options": {"types": copy.deepcopy(types)},
    }


def free_slot(handlers, opcode):
    """Check the FULL handler list, not the de-duplicated index."""
    for e in handlers:
        if e["opCode"].lower() == opcode.lower():
            die("opCode %s already occupied by %s" % (opcode, e.get("handler")))


def build(tenant, version, types):
    doc = json.load(open(SNAP % (tenant, "before")))
    if doc["data"]["id"] != tenant or doc["data"]["type"] != "tenants":
        die("snapshot envelope mismatch for %s" % tenant)
    attrs = copy.deepcopy(doc["data"]["attributes"])
    if str(attrs.get("majorVersion")) != version or attrs.get("region") != "GMS":
        die("tenant %s is not GMS %s (got %s %s)"
            % (tenant, version, attrs.get("region"), attrs.get("majorVersion")))

    hs = attrs["socket"]["handlers"]
    idx = index(hs)
    before_codes = [int(e["opCode"], 16) for e in hs]
    if before_codes != sorted(before_codes):
        die("live handlers are not already ascending - unexpected")
    # Pre-existing numeric opCode collisions (e.g. v95 "0x09C" vs "0x9C") are NOT
    # in scope; record them so the post-edit check only flags NEW ones.
    pre_dupes = {c for c in before_codes if before_codes.count(c) > 1}
    if pre_dupes:
        print("  note: pre-existing numeric opCode collisions (untouched): %s"
              % sorted(hex(c) for c in pre_dupes))
    # relative order of colliding entries must survive the re-sort (last one wins
    # in BuildHandlerMap, so flipping them would change behaviour)
    pre_order = [e["handler"] for e in hs if int(e["opCode"], 16) in pre_dupes]

    if version == "92":
        # 1. types onto the four existing movement-carrying handlers
        for h, op in (("CharacterMoveHandle", "0x2E"),
                      ("MonsterMovementHandle", "0xDC"),
                      ("SummonMoveHandle", "0xC8"),
                      ("NPCActionHandle", "0xEA")):
            e = expect_present(idx, h, op)
            set_types(e, types)
            set_validator(e)
        # 2. new PetMovementHandle at 0xC4
        expect_absent(idx, "PetMovementHandle")
        free_slot(hs, "0xC4")
        hs.append(new_entry("PetMovementHandle", "0xC4", types))
        # 3. summon opcode corrections (SummonMove already has types from step 1)
        for h, old, new in (("SummonMoveHandle", "0xC8", "0xCC"),
                            ("SummonAttackHandle", "0xC9", "0xCD"),
                            ("SummonDamageHandle", "0xCA", "0xCE")):
            e = expect_present(idx, h, old)
            free_slot(hs, new)
            e["opCode"] = new
    elif version == "95":
        for h, op in (("PetMovementHandle", "0xC7"),
                      ("MonsterMovementHandle", "0xE3"),
                      ("SummonMoveHandle", "0xCF"),
                      ("NPCActionHandle", "0xF1")):
            e = expect_present(idx, h, op)
            set_types(e, types)
            set_validator(e)
        expect_absent(idx, "CharacterMoveHandle")
        free_slot(hs, "0x2C")
        hs.append(new_entry("CharacterMoveHandle", "0x2C", types))
    else:
        die("unknown version %s" % version)

    # 4. re-sort ascending
    hs.sort(key=lambda e: int(e["opCode"], 16))
    codes = [int(e["opCode"], 16) for e in hs]
    if codes != sorted(codes):
        die("post-sort not ascending")
    post_dupes = {c for c in codes if codes.count(c) > 1}
    new_dupes = post_dupes - pre_dupes
    if new_dupes:
        die("edit introduced NEW duplicate opCodes: %s" % sorted(hex(c) for c in new_dupes))
    post_order = [e["handler"] for e in hs if int(e["opCode"], 16) in pre_dupes]
    if post_order != pre_order:
        die("re-sort changed the relative order of colliding entries: %s -> %s"
            % (pre_order, post_order))

    out = {"data": {"type": "tenants", "id": tenant, "attributes": attrs}}
    path = SNAP % (tenant, "after")
    with open(path, "w") as f:
        json.dump(out, f, indent=2)
    print("wrote %s (handlers=%d)" % (path, len(hs)))


types = canonical_types()
print("canonical types: %d entries" % len(types))
build(V92, "92", types)
build(V95, "95", types)
print("OK")
```

### `build_v48.py` — v48

```python
#!/usr/bin/env python3
"""Build the reconciled GMS 48.1 tenant socket config.

Single edit: give SummonMoveHandle the movement `types` table it is missing.
The table is sourced from THIS SAME TENANT's live CharacterMoveHandle (0x21) --
not from a seed template and not from another tenant. v48's table is 23 entries;
the v92/v95 table is 37 and must never be used here.

No opcode changes, no new entries, no validator changes.
"""
import copy
import json
import os
import sys

SNAP = os.environ["SCRATCH"] + "/%s.%s.json"
V48 = "e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd"
EXPECTED_LEN = 23

# handler -> expected live opCode. All five must already exist with a validator.
EXPECTED = {
    "CharacterMoveHandle": "0x21",
    "PetMovementHandle": "0x71",
    "SummonMoveHandle": "0x78",
    "MonsterMovementHandle": "0x81",
    "NPCActionHandle": "0x8A",
}
TARGET = "SummonMoveHandle"


def die(msg):
    print("ABORT: %s" % msg)
    sys.exit(1)


doc = json.load(open(SNAP % (V48, "before")))
if doc["data"]["id"] != V48 or doc["data"]["type"] != "tenants":
    die("snapshot envelope mismatch")
attrs = copy.deepcopy(doc["data"]["attributes"])
if attrs.get("region") != "GMS" or str(attrs.get("majorVersion")) != "48":
    die("tenant is not GMS 48 (got %s %s)" % (attrs.get("region"), attrs.get("majorVersion")))

hs = attrs["socket"]["handlers"]

codes = [int(e["opCode"], 16) for e in hs]
if codes != sorted(codes):
    die("live handlers are not ascending")
if len(codes) != len(set(codes)):
    die("live config has duplicate numeric opCodes: %s"
        % sorted(hex(c) for c in codes if codes.count(c) > 1))

# locate the five movement handlers; each must be unique and at its expected opCode
found = {}
for e in hs:
    h = e.get("handler")
    if h in EXPECTED:
        if h in found:
            die("movement handler %s appears more than once" % h)
        found[h] = e
for h, op in EXPECTED.items():
    e = found.get(h)
    if e is None:
        die("expected movement handler %s to be present, it is ABSENT" % h)
    if e["opCode"].lower() != op.lower():
        die("handler %s: expected live opCode %s, found %s" % (h, op, e["opCode"]))
    if not e.get("validator"):
        die("handler %s has an empty validator - out of scope for this edit" % h)

# source the table from this tenant's own CharacterMoveHandle
src = (found["CharacterMoveHandle"].get("options") or {}).get("types")
if not src:
    die("CharacterMoveHandle carries no types table to copy from")
if len(src) != EXPECTED_LEN:
    die("expected a %d-entry v48 table, found %d - refusing (v92/v95 use 37)"
        % (EXPECTED_LEN, len(src)))

# every other movement handler that already has a table must agree with it
for h, e in found.items():
    if h == TARGET:
        continue
    t = (e.get("options") or {}).get("types")
    if t != src:
        die("handler %s types table differs from CharacterMoveHandle - ambiguous source" % h)

# the single edit
tgt = found[TARGET]
opts = tgt.get("options")
if opts not in (None, {}):
    die("%s already carries options %r - refusing to overwrite" % (TARGET, opts))
tgt["options"] = {"types": copy.deepcopy(src)}

# nothing else may have moved: handler count, order, opcodes, validators
after_codes = [int(e["opCode"], 16) for e in hs]
if after_codes != codes:
    die("opCode list changed - this edit must not reorder or relocate anything")
if len(hs) != len(doc["data"]["attributes"]["socket"]["handlers"]):
    die("handler count changed")

out = {"data": {"type": "tenants", "id": V48, "attributes": attrs}}
path = SNAP % (V48, "after")
with open(path, "w") as f:
    json.dump(out, f, indent=2)
print("wrote %s" % path)
print("edit: %s @%s options.types = <%d entries copied from CharacterMoveHandle @0x21>"
      % (TARGET, EXPECTED["SummonMoveHandle"], len(src)))
print("OK")
```

### `diff.py` — the pre-PATCH proof

```python
#!/usr/bin/env python3
"""Prove the built PATCH body differs from the live snapshot ONLY in the
intended movement-handler entries."""
import json
import sys
from collections import defaultdict

a = json.load(open(sys.argv[1]))["data"]["attributes"]
b = json.load(open(sys.argv[2]))["data"]["attributes"]

ka, kb = set(a), set(b)
print("top-level keys added/removed:", kb - ka, ka - kb)
for k in sorted(ka & kb):
    if k != "socket" and a[k] != b[k]:
        print("CHANGED OUTSIDE SOCKET:", k)

print("socket sub-keys added/removed:",
      set(b["socket"]) - set(a["socket"]), set(a["socket"]) - set(b["socket"]))
print("writers unchanged:", a["socket"].get("writers") == b["socket"].get("writers"))


def group(hs):
    g = defaultdict(list)
    for e in hs:
        g[e.get("handler")].append(e)
    for v in g.values():
        v.sort(key=lambda e: int(e["opCode"], 16))
    return g


ga, gb = group(a["socket"]["handlers"]), group(b["socket"]["handlers"])
print("handler count: %d -> %d" % (len(a["socket"]["handlers"]), len(b["socket"]["handlers"])))
print("handlers added:", sorted(set(gb) - set(ga)))
print("handlers removed:", sorted(set(ga) - set(gb)))

changed = []
for h in sorted(set(ga) & set(gb)):
    if ga[h] != gb[h]:
        changed.append(h)
        for old, new in zip(ga[h], gb[h]):
            deltas = []
            if old.get("opCode") != new.get("opCode"):
                deltas.append("opCode %s->%s" % (old.get("opCode"), new.get("opCode")))
            if old.get("validator") != new.get("validator"):
                deltas.append("validator %r->%r" % (old.get("validator"), new.get("validator")))
            if old.get("services") != new.get("services"):
                deltas.append("services %r->%r" % (old.get("services"), new.get("services")))
            oo, no = old.get("options"), new.get("options")
            if oo != no:
                deltas.append("options %s->types[%d]"
                              % (oo, len(no["types"]) if no and no.get("types") else 0))
            for k in set(old) | set(new):
                if k not in ("opCode", "validator", "services", "options") and old.get(k) != new.get(k):
                    deltas.append("%s %r->%r" % (k, old.get(k), new.get(k)))
            print("  handler changed:", h, "|", "; ".join(deltas))
print("changed handler names:", changed)

codes = [int(e["opCode"], 16) for e in b["socket"]["handlers"]]
print("ascending:", codes == sorted(codes))
```

### `sweep.sh` — fleet-wide `types`-level audit

```sh
#!/bin/sh
# Fleet-wide movement-`types` sweep. Checks every configuration tenant at the
# TYPES level -- not merely "is the handler registered", which is a weaker claim.
# Usage: SCRATCH=/some/tmp/dir sh sweep.sh
set -eu
: "${SCRATCH:?set SCRATCH to a scratch directory}"
RESOLVE='--resolve dev.atlas.home:80:192.168.23.230'
BASE='http://dev.atlas.home/api/configurations'

curl -s $RESOLVE "$BASE/tenants" > "$SCRATCH/tenants.json"

python3 -c "
import json
d=json.load(open('$SCRATCH/tenants.json'))
print('\n'.join(e['id'] for e in d['data']))
" > "$SCRATCH/ids.txt"

while read -r T; do
  curl -s $RESOLVE "$BASE/tenants/$T" > "$SCRATCH/sweep.$T.json"
done < "$SCRATCH/ids.txt"

python3 - "$SCRATCH" <<'PY'
import json, glob, sys
S = sys.argv[1]
MOVE = ['CharacterMoveHandle', 'PetMovementHandle', 'SummonMoveHandle',
        'MonsterMovementHandle', 'NPCActionHandle']
rows = []
for f in sorted(glob.glob('%s/sweep.*.json' % S)):
    d = json.load(open(f)); a = d['data']['attributes']
    got = {}
    for e in a['socket']['handlers']:
        if e.get('handler') in MOVE:
            t = (e.get('options') or {}).get('types')
            got[e['handler']] = (e['opCode'], e.get('validator'), len(t) if t else None)
    gaps = [h for h in MOVE if h not in got or not got[h][2]]
    novalid = [h for h, v in got.items() if not v[1]]
    lens = sorted({v[2] for v in got.values()}, key=lambda x: (x is None, x))
    rows.append((a['region'], float(a['majorVersion']), d['data']['id'],
                 len(got), lens, gaps, novalid))
rows.sort(key=lambda r: (r[0], r[1]))
print('%-4s %-6s %-38s %-8s %-11s %s' %
      ('REG', 'VER', 'TENANT', 'PRESENT', 'TYPES-LENS', 'GAPS'))
bad = 0
for reg, ver, tid, n, lens, gaps, nv in rows:
    if gaps or nv or n != 5:
        bad += 1
    print('%-4s %-6s %-38s %-8s %-11s %s%s' %
          (reg, ver, tid, '%d/5' % n, lens, gaps or 'none',
           ('  NO-VALIDATOR:%s' % nv) if nv else ''))
print()
print('tenants with a movement-types gap: %d' % bad)
PY
```
