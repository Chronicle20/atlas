# Kites (Cash Item Category 508) — Design

Task: task-211-kite-cash-item
Status: Draft for review
Created: 2026-08-10
Input: [`prd.md`](prd.md) (approved)

---

## 1. Scope of this document

The PRD left six open questions (§9 Q1–Q6), three of which block functional
requirements. This design closes all six with cited evidence, then records the
architecture decisions for the new `atlas-kites` service, the `atlas-channel`
integration, and the packet-layer corrections.

Nothing below is asserted from memory or from general MapleStory knowledge.
Every wire claim carries an IDB address or a repo `file:line`.

---

## 2. Open questions — resolved

### Q1 — Which `fieldLimit` bit forbids kites? → **None. There is no bit.**

The client's placement gate is not a `FieldLimit` mask at all. It is a
hard-coded field-id range test inside the type-18 arm of
`CWvsContext::SendConsumeCashItemUseRequest` (GMS v95.0 IDB, function
`0x9eb3e0`; arm entry `$LN136_30` at `0x9ecfa2`, labelled
`jumptable 009EB50A case 18`):

```
0x9ecfa2  cmp  TSingleton<CUniqueModeless>::ms_pInstance, ebx   ; case 18 entry
0x9ecfa8  jz   loc_9ECFEF                                       ; no modeless dialog open -> proceed
          ...                                                    ; else CHATLOG_ADD(StringPool 0x98), abort

0x9ed017  call CWvsContext::GetCurFieldID
0x9ed01e  mov  eax, 6B5FCA6Bh
0x9ed023  imul this
0x9ed025  sar  edx, 16h
0x9ed028  mov  eax, edx
0x9ed02a  shr  eax, 1Fh
0x9ed02d  add  eax, edx                                         ; eax = GetCurFieldID() / 10000000
0x9ed02f  cmp  eax, 5Bh                                         ; 91
0x9ed032  jnz  loc_9ED082                                       ; NOT 91 -> proceed to the input dialog
0x9ed034  ...  StringPool::GetString(0xE62); CHATLOG_ADD        ; == 91 -> refuse, no packet sent
```

`0x6B5FCA6B` with `sar 22` is the canonical MSVC signed divide-by-10,000,000
magic (`ceil(2^54 / 10^7) = 1801439851 = 0x6B5FCA6B`). So the client's own rule
is: **a kite may not be placed on a field whose id divided by 10,000,000 is 91**
— the Free Market range (`910000000`–`919999999`). Everywhere else is allowed.

Two corollaries:

- `libs/atlas-constants/map/field_limit.go` gets **no new bit**. The PRD's
  fallback branch (FR-5.3 → tenant-configured map policy) is the branch that
  applies, and it now has a concrete, evidenced default rather than an
  arbitrary one.
- The constant `0x6B5FCA6B` does not appear **anywhere** in the GMS v83 dump
  (`find_bytes "6B CA 5F 6B"` over `MapleStory_dump.exe.i64` → 0 matches), so
  the Free-Market ban is a later-version addition. Atlas will enforce it
  uniformly on all versions anyway: it is a server policy default, not a
  client-parity requirement, and blocking FM banners on v83 is the desired
  behaviour regardless.

**Decision:** FR-5.3 is implemented as a tenant-configurable map policy in
`atlas-kites` (see ADR-6), defaulting to "deny fields where
`uint32(mapId) / 10000000 == 91`". Reason code `MAP_FORBIDDEN`.

### Q2 — Destroy animation type mapping → **0 = animated, non-zero = silent.**

`CMessageBoxPool::OnMessageBoxLeaveField` (GMS v95.0 `0x635d60`):

```c
v3   = CInPacket::Decode1(iPacket);      // the leading byte
dwID = CInPacket::Decode4(iPacket);
if ( ZMap<...>::GetAt(&this->m_mMessageBox, &dwID, &pr) )
{
  CMessageBoxPool::RemoveMessageBox(this, v25);   // always removed
  if ( !v3 )                                      // ONLY when the byte is 0
  {
    ... IWzVector2D::RelMove(...);
    ... IWzGr2DLayer::Getcanvas / RemoveCanvas / InsertCanvas / Animate(GA_STOP)
    CAnimationDisplayer::RegisterOneTimeAnimation(...);   // the despawn animation
  }
}
```

The byte is not a *selector* between two animations; it is a **suppress-animation
flag**. Zero plays the one-shot despawn animation; any non-zero value removes the
box instantly with no visual. The repo's own audit records already say exactly
this and were simply never wired up —
`docs/packets/audits/gms_v95/FieldKiteDestroy.json` row 0:
`"bAnimation (leave animation type; 0 = play despawn animation)"`, and the v83
record at `docs/packets/audits/gms_v83/FieldKiteDestroy.json` matches.

**Decision:** both PRD reasons (`OWNER_LEFT`, `OWNER_LOGGED_OUT`) map to **0**
(animated). There is currently no reason that wants the silent path; introducing
one is a future concern, not this task's.

Consequently the two constants in
`libs/atlas-packet/field/clientbound/kite_destroy.go:15-20` are renamed to say
what they mean — `KiteDestroyAnimated = 0`, `KiteDestroySilent = 1` — a
byte-neutral rename with the same discipline as FR-2.1 (see ADR-7).

### Q3 — Serverbound sub-body shape → **one length-prefixed ASCII string.**

Same case-18 arm. The full flow after the field check:

1. `CUIHope` dialog is constructed (`??0CUIHope@@QAE@J@Z` called at `0x9ed0b6`)
   and run modally (`CDialog::DoModal` at `0x9ed0d9`); a return other than 1
   aborts with no packet.
2. `CUIHope::GetText(&s1, &s2, &s3)` at `0x9ed0f8` reads the **three** edit
   controls (`CUIHope::GetText` `0x782a10` → `m_pEditInput1/2/3`).
3. The three lines are joined with `'\n'`:
   `s4 = s1 + '\n' + s2 + '\n' + s3` (`ZXString<char>::operator+` chain
   `0x9ed107`–`0x9ed163`).
4. `CCurseProcess::ProcessString(s4)` at `0x9ed1b7`; a non-zero result raises
   `CUtilDlg::Notice(StringPool 0x11D)` and **aborts without sending**.
5. `COutPacket::EncodeStr(oPacket, s4)` at `0x9ed271` — **the only encode in the
   arm** — then it jumps to the shared send tail `loc_9F0637`.

So the sub-body is exactly `message: string`, as the PRD expected. FR-1.3's gate
is satisfied for GMS v95.

**One correction to FR-1.1:** the sub-body must also carry the *trailing*
`updateTime` for the versions that trail it. `ItemUse.UpdateTimeFirst`
(`libs/atlas-packet/cash/serverbound/item_use.go:23`) reports GMS ≤ 84 as
trailing; the sibling `ItemUseChalkboard`
(`item_use_chalkboard.go:42-49`) shows the established shape:

```go
func (m *ItemUseChalkboard) Decode(...) {
    m.message = r.ReadAsciiString()
    if !m.updateTimeFirst { m.updateTime = r.ReadUint32() }
}
```

`ItemUseKite` copies this exactly. Omitting it would mis-decode every GMS
version at or below v84 — which is six of the ten in scope.

**Older versions.** The type-18 arm demonstrably exists on GMS v83:
`get_cashslot_item_type` (v83 `0x48645b`) contains `case 508: return 18`, and
v83's case-18 jumptable target (`0xa0bd2b`) opens with the identical
`CUniqueModeless` guard (`sub_A14A1C` at `0xa14a1c` returns
`TSingleton<CUniqueModeless>::ms_pInstance != 0`). The v83 dump has no `CUIHope`
symbol and its wide-string data is not reachable by `find_bytes`, so the v83
*body* layout is **not** verified here. It is treated as message-only by
inheritance from v95 and **must be pinned per version by byte fixtures during
implementation** — the same per-cell discipline every other cash sub-body in
this package already follows (`v48_test.go`, `v61_test.go`, `v72_test.go`,
`v79_test.go`).

### Q4 — Message length bound → **182 bytes.**

`CUIHope::OnCreate` (`0x7824f0`) creates three `CCtrlEdit` controls, each with
`CREATEPARAM` field `+0x30` set to `60`:

```c
v25 = 60;  ZXString<char>::operator=(paramEdit, sStrDefault);
this->m_pEditInput1.p->CreateCtrl(..., 15, 57, 314, 16, paramEdit);
v25 = 60;  ... m_pEditInput2 ... (15, 74, 314, 16, paramEdit);
v25 = 60;  ... m_pEditInput3 ... (15, 91, 314, 16, paramEdit);
```

`CCtrlEdit::CREATEPARAM+0x30` is `nHorzMax` (`type_inspect` on
`CCtrlEdit::CREATEPARAM`, size 60). `CCtrlEdit::InsertString` (`0x4e2a10`)
compares `nHorzMax` against **`ZXString` byte lengths**, not pixel widths:

```c
if ( v14 + v16 + v18 > this->m_nHorzMax )   // len(prefix)+len(suffix)+len(insert)
```

So each line is capped at 60 bytes, and the joined payload is at most
`60 + 1 + 60 + 1 + 60 = 182` bytes.

**Decision:** `atlas-kites` rejects `len(message) > 182` with reason
`MESSAGE_TOO_LONG`. The bound lives in the authoritative service (not the
socket decoder) so a crafted packet cannot bypass it, per the PRD's §8 security
requirement.

### Q5 — gms v12 applicability → **out of scope; not an `n-a` claim.**

- `docs/packets/audits/STATUS.md:22` shows the matrix column set:
  v48, v61, v72, v79, v83, v84, v87, v92, v95, JMS185. **There is no v12
  column** — the three kite rows (`STATUS.md:330,332,336`) have exactly ten
  cells each, matching the PRD's FR-9.1 table.
- `docs/packets/audits/` has ten version directories; no `gms_v12`.
- No v12 IDB is present in the `idb_list` session set, so no evidence could be
  produced for it even in principle.
- `template_gms_12_1.json` is a skeletal bring-up: 24 handlers, 44 writers,
  covering login/character-list/field/monster/summon only. It has **no**
  `CharacterCashItemUseHandle` binding (the other ten templates each have
  exactly one), no chalkboard, no drops, no reactors.

**Decision:** the correct scope is **ten templates**, not eleven. v12 gets no
kite bindings and no `n-a` matrix entry (there is no cell to mark). The PRD's
"all eleven" in FR-9.1/§10 is amended to "all ten matrix versions"; the
acceptance criterion becomes `grep -i kite` returning bindings in the ten
non-v12 templates. This is not a deferral — v12 cannot host the feature because
it cannot even receive the serverbound request.

### Q6 — Per-map cap per instance? → **Yes, per `field.Model`, and this fixes a latent bug.**

`field.Model` includes `instanceId`, and every sibling in-map registry route is
`.../maps/{mapId}/instances/{instanceId}/<resource>`. Kites follow that.

Worth stating explicitly because the template we are copying gets it wrong:
`atlas-chalkboards`' character-in-map index builds its field key **without** the
instance —
`services/atlas-chalkboards/atlas.com/chalkboards/kafka/consumer/character/consumer.go`
uses `field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).Build()` on
login/logout/map-change, leaving `Instance` at `uuid.Nil`, while
`chalkboard/resource.go:63` looks the set up with `SetInstance(instanceId)`.
Any non-nil instance therefore returns an empty set — instanced maps have never
replayed chalkboards. The character status events *do* carry the instance
(`StatusEventLoginBody.Instance`, `StatusEventLogoutBody.Instance`,
`StatusEventMapChangedBody.OldInstance`/`TargetInstance`,
`ChangeChannelEventLoginBody.Instance`), so it is purely a dropped field.

**Decision:** `atlas-kites` threads the instance through on every transition.
The chalkboards bug is *not* fixed here — different service, different task, and
touching it would widen this branch's blast radius. It is called out so the
copy does not inherit it.

---

## 3. Architecture decisions

### ADR-1 — A new `atlas-kites` service, modelled on `atlas-chalkboards`

**Alternatives considered**

| Option | Verdict |
|---|---|
| **A. New `atlas-kites` service** (PRD) | **Chosen.** |
| B. Fold kites into `atlas-chalkboards` | Rejected. |
| C. Fold kites into `atlas-maps` (alongside `mist`) | Rejected. |

**B** is genuinely tempting: `atlas-chalkboards` already owns "a player message
pinned to a map, owner-bound, Redis-only, replayed on map entry, destroyed when
the owner leaves", and already runs the character-in-field index and the
character-status consumers that kites need. Folding in would cost zero service
registrations. It is rejected because the service name would stop describing its
contents, and the project's precedent for that situation is a rename
(atlas-gachapons → atlas-reward-pools), which is strictly more disruptive than a
new service. Domain-per-service is the house style; two message-pinning domains
in one binary is not.

**C** is rejected for the reason the PRD already gives: `mist` has no REST
resource, so it is never replayed on map entry, and map-entry replay is half of
this feature.

**What A actually costs, and what it reuses.** The reuse is large and should be
treated as a copy-with-corrections, not a from-scratch build. `atlas-chalkboards`
is only ~15 files. `atlas-kites` mirrors its package layout one-for-one:

```
services/atlas-kites/atlas.com/kites/
  main.go
  kite/        model.go builder.go registry.go processor.go producer.go
               resource.go rest.go
  character/   model.go registry.go processor.go        # in-field index (instance-correct)
  configuration/ model.go rest.go requests.go registry.go   # kite-configs (ADR-6)
  kafka/consumer/{kite,character}/consumer.go
  kafka/message/{kite,character}/kafka.go
  rest/
```

Registration follows [`docs/adding-a-new-service.md`](../../adding-a-new-service.md)
§1–§5 and §7 is N/A (not socket-exposing). **§6 Databases is skipped entirely**
— confirmed against the precedent: `atlas-chalkboards` appears in
`.github/config/services.json:65` with no database, and `grep chalkboard
tools/db-bootstrap.sh` returns nothing. `tools/service-registration-guard.sh`
does not demand a `DB_NAME` for services that declare none. This resolves the
PRD's Service Impact "confirm during design" item.

### ADR-2 — State layout: character-keyed registry + character-in-field set

Because FR-5.2 makes a kite **one per character**, `characterId` is a natural
primary key and the "index by field" requirement in PRD §6 dissolves: the
field→kites lookup is the intersection of *characters currently in that field*
with *characters that own a kite*. That is precisely how
`atlas-chalkboards/chalkboard/resource.go:76-84` serves `chalkboards_in_map`,
and it needs no second index to maintain and no write-time consistency between
two Redis structures.

Three Redis structures, all via `libs/atlas-redis` (so
`tools/redis-key-guard.sh` stays clean):

| Structure | Type | Namespace | Purpose |
|---|---|---|---|
| kite state | `atlas.TenantRegistry[uint32, Model]` keyed by `characterId` | `kite` | the kite itself |
| character-in-field | `atlas.TenantKeyedSet[field.Model]` | `kite-char` | replay + per-map cap denominator |
| id counter | `atlas.NewIDGenerator(client, "kite")` | `kite` | wire id (ADR-3) |

The stored value is the full immutable `Model` (JSON-serialised by
`TenantRegistry`), not a bare string as chalkboards does — kites carry nine
fields, not one.

**Rejected alternative:** a dedicated `atlas.Index` from `field` → kite ids
(`libs/atlas-redis/index.go`). It works, but it is a second structure that must
stay consistent with the primary registry across create/destroy/owner-departure,
and it buys nothing that the character-in-field set (which we need anyway, for
the owner-departure teardown) does not already give.

**Cardinality note.** The intersection scan is bounded by the number of
characters in one map, not by tenant-wide kite count, and the result is bounded
by the per-map cap (default 10). The REST list stays paginated to match the
`chalkboards_in_map` contract.

### ADR-3 — Wire id from a tenant-scoped Redis counter

`libs/atlas-redis/id.go` already provides exactly what FR-3.3 asks for:
`atlas.NewIDGenerator(client, namespace).NextID(ctx, t) (uint32, error)`, a
tenant-keyed `INCR`. No process-local counter, no collision across replicas.
`REMOVE_KITE` addresses by this id alone, and the id is stable for the kite's
lifetime because the registry row is written once and never re-keyed.

The id is *not* the `characterId`. Chalkboards conflates the two (its
`RestModel.Id` is the character id); kites must not, because the wire id is a
distinct object id in `MESSAGEBOX+16` and the client keys its
`ZMap<unsigned long, ZRef<MESSAGEBOX>>` on it.

### ADR-4 — Cap enforcement under concurrency: one Redis lock per field

FR-5.1 (per-map cap) and FR-5.2 (one per character) are registry invariants and
must hold when two `CREATE` commands race. FR-5.2 alone is safe by construction
— the command topic is keyed on `characterId`
(`producer.CreateKey(int(characterId))`, matching
`atlas-channel/chalkboard/producer.go:15`), so one character's commands are
totally ordered within a partition. FR-5.1 is not: two *different* characters
placing on the same full-but-for-one map land on different partitions.

`Create` therefore takes `atlas.NewLock(client, "kite-cap").Acquire(ctx,
fieldKey)` around {count-in-field → validate → allocate id → registry insert}
and releases it after the insert (before the Kafka emit, which is not part of
the invariant). Lock acquisition failure is a `MAP_FULL`-class refusal rather
than a retry loop; a lost race on a full map is the correct outcome anyway.

**Rejected alternative:** a Lua CAS on a per-field counter key. Faster, but it
introduces a fourth structure that can drift from the registry, and the
contention here is negligible (a handful of kite placements per map per minute).

### ADR-5 — Kafka topology

Exactly as PRD §5, with the following made explicit.

**`COMMAND_TOPIC_KITE`** (channel → kites), keyed `characterId`:

| Command | Body |
|---|---|
| `CREATE` | field scope, `characterId`, `name`, `templateId`, `message`, `x`, `y` |
| `DESTROY` | field scope, `kiteId` |

`name` is on the command, not fetched by `atlas-kites`. `atlas-channel` must
load the character anyway to get `x`/`y` (ADR-7 below), so the name comes free
and `atlas-kites` avoids a REST dependency on the character service for a value
it would fetch on every create. It is server-side state either way — never
client-supplied — satisfying PRD §8.

**`EVENT_TOPIC_KITE_STATUS`** (kites → channel), keyed `mapId`
(`producer.CreateKey(int(field.MapId()))`, matching
`atlas-chalkboards/chalkboard/producer.go:15`):

| Event | Body |
|---|---|
| `CREATED` | full projection: `kiteId`, `characterId`, `name`, `templateId`, `message`, `x`, `y`, field scope |
| `DESTROYED` | `kiteId`, field scope, `reason` |
| `CREATION_FAILED` | `characterId`, field scope, `reason` |

Reasons: `OWNER_LEFT`, `OWNER_LOGGED_OUT` (destroy); `MAP_FULL`,
`ALREADY_PLACED`, `MAP_FORBIDDEN`, `MESSAGE_TOO_LONG` (failure). `FieldKiteError`
has an empty body (`kite_error.go:15-19`), so the reason is server-side
diagnostics only — which is exactly why it must be logged at the point of
refusal (PRD §8 Observability).

`atlas-kites` additionally consumes the existing character status topic (no new
topic) for login/logout/map-change/channel-change, driving both the
character-in-field index and the FR-6 teardown — the same handler set as
`atlas-chalkboards/kafka/consumer/character/consumer.go`, with the ADR-2
instance correction.

**Topic env vars** must be suffixed in base `env-configmap.yaml` *and* mirrored
into both overlay `configMapGenerator` literals — the unsuffixed-fallback trap
(`docs/adding-a-new-service.md` "Silent-failure traps" #3) is silent, and the
guard only checks parity of keys already present, not that the right new ones
exist.

### ADR-6 — Placement policy as a tenant configuration resource

New atlas-tenants configuration resource **`kite-configs`**, consumed exactly
like `mts-configs` — the established precedent is
`services/atlas-channel/atlas.com/channel/mts/configuration/` (`requests.go`
builds `GET {TENANTS}/tenants/{id}/configurations/kite-configs`; `rest.go`'s
`Extract` folds every zero-valued knob back to a compiled default; `registry.go`
caches per tenant). `atlas-kites` gets the same four-file package.

| Knob | Type | Default | Requirement |
|---|---|---|---|
| `maxPerMap` | `int` | `10` | FR-5.1 |
| `maxMessageLength` | `int` | `182` | Q4 / §8 security |
| `blockedMapPrefixes` | `[]uint32` | `[91]` | FR-5.3 / Q1 |

`blockedMapPrefixes` is evaluated as `uint32(f.MapId()) / 10000000`, mirroring
the client's own arithmetic (Q1). A prefix list rather than a map allowlist
because the client's rule is itself a prefix rule, the list is short, and an
explicit per-map denylist for ~900 FM map ids would be unmaintainable. A tenant
that wants a single extra map blocked can be served by a follow-on knob; that is
not this task's problem and no stub is left for it.

Per DOM-25 no client wire value is hard-coded: opcodes resolve through
`opcodes.BuildWriterProducer` (`libs/atlas-opcodes/producer.go:19`) as they
already do for the three kite writers.

### ADR-7 — Packet layer

**7a. New `libs/atlas-packet/cash/serverbound/item_use_kite.go`.** Immutable,
private fields, `NewItemUseKite(updateTimeFirst bool) *ItemUseKite`, both
`Encode` and `Decode`, structurally identical to `ItemUseChalkboard` including
the trailing-`updateTime` branch (Q3). Fixtures round-trip under every
`test.Variants` entry, plus explicit byte fixtures per version in the existing
`v48_test.go`/`v61_test.go`/`v72_test.go`/`v79_test.go` style.

**7b. `FieldKiteSpawn.kiteType` → `y` (FR-2.1/2.2/2.3).** Confirmed on a second
version beyond the PRD's v95 read: GMS v83 `CMessageBoxPool::OnMessageBoxEnterField`
(`0x65acdf`) decodes in the identical order and applies the identical
sprite-anchor offsets —

```c
*(v52 + 16) = CInPacket::Decode4(v3);   // id
*(v4  + 56) = CInPacket::Decode4(v3);   // templateId
ZXString<char>::operator=((v52 +  8), CInPacket::DecodeStr(v3, &result));  // message
ZXString<char>::operator=((v52 + 12), CInPacket::DecodeStr(v3, &result));  // name
*(v7  + 28) = CInPacket::Decode2(v3);   // x
*(v8  + 32) = CInPacket::Decode2(v3);   // y
*(v52 + 36) = *(v52 + 28) - 3;          // renderX
*(v52 + 40) = *(v52 + 32) - 100;        // renderY
```

Both int16s feed one `IWzVector2D::RelMove`. There is no type discriminator on
the wire — appearance comes from `templateId` via `CItemInfo::GetItemProp`.
(v83 additionally seeds the swing angle from `CRand32::Random() % 360` at
`+48` — client-local, never transmitted, and further evidence that nothing in
this packet selects a variant.)

Rename the field, the `NewKiteSpawn` parameter, and the `Decode`/`Encode` bodies.
**No encoded byte changes on any version.** Audit-JSON row-5 `IDAComment` must be
corrected in the four records that currently mislabel it:

| Record | Current row-5 comment |
|---|---|
| `docs/packets/audits/gms_v83/FieldKiteSpawn.json` | `nType/y (spawn y or kite type, +32)` |
| `docs/packets/audits/gms_v87/FieldKiteSpawn.json` | `nType (kite type, +32 @0x694f30)` |
| `docs/packets/audits/gms_v95/FieldKiteSpawn.json` | `nType (kite type)` |
| `docs/packets/audits/jms_v185/FieldKiteSpawn.json` | `y / kiteType (@line94)` |

(v48/v61/v72/v79/v84/v92 rows are empty strings and need no edit.) Then re-pin
`docs/packets/evidence/*/field.clientbound.FieldKiteSpawn.yaml` and regenerate
the matrix; every cell must retain its current status
(`STATUS.md:332` — v48 ✅, v61 🟡ᶠ, v72 🟡ᶠ, v79 🟡ᶠ, v83 ✅, v84 ✅, v87 ✅,
v92 🟡ᶠ, v95 ✅, JMS185 ✅). The existing `packet-audit:verify` markers in
`kite_spawn_test.go:9-13` are untouched apart from the identifier rename.

**7c. `KiteDestroyAnimationType1/2` → `KiteDestroyAnimated/Silent`** (Q2).
Byte-neutral; the audit records already carry the correct semantics so no JSON
edit is needed for `FieldKiteDestroy`.

**7d. Version gating.** None is required. All three clientbound codecs are
version-uniform (identical decode order on v83 and v95; the matrix already
verifies six of ten cells per row against the current single-shape struct), and
the serverbound sub-body's only version axis is the existing
`ItemUse.UpdateTimeFirst` boolean. No `MajorAtLeast` gate is introduced.

### ADR-8 — Version scope

Ten versions: gms 48/61/72/79/83/84/87/92/95 and jms 185. v12 excluded (Q5).

FR-9.4 is confirmed as expected: every one of the ten templates already binds
`CharacterCashItemUseHandle` exactly once (verified by
`grep -c CharacterCashItemUseHandle` across the template set), and the type-18
sub-body rides that existing opcode. **No new handler entry in any template.**
Only the three writers are added, at their sorted `opCode` position, each with
an `fname`, per
[`docs/packets/TEMPLATE_CONVENTIONS.md`](../../packets/TEMPLATE_CONVENTIONS.md).

FR-7.6 is confirmed verify-only: `services/atlas-channel/atlas.com/channel/main.go:724-726`
already lists `fieldcb.KiteSpawnWriter`, `fieldcb.KiteErrorWriter`,
`fieldcb.KiteDestroyWriter` in `produceWriters()`. No change.

---

## 4. Component design

### 4.1 `atlas-kites` — domain

```go
type Model struct {                 // immutable; private fields + getters + Builder
    id          uint32              // wire id (ADR-3)
    f           field.Model
    characterId uint32
    name        string
    templateId  uint32
    message     string
    x           int16
    y           int16
    createdAt   time.Time
}
```

`Builder` is `NewBuilder(id, f, characterId)` with `SetName/SetTemplateId/
SetMessage/SetPosition/SetCreatedAt`, mirroring
`atlas-chalkboards/chalkboard/builder.go`.

### 4.2 `atlas-kites` — processor

Project split: pure `Method(mb *message.Buffer)` composing into a buffer, plus
`MethodAndEmit()` wrappers (`message.Emit(p)`), as in
`services/atlas-maps/atlas.com/maps/mist/processor.go`.

```go
type Processor interface {
    Create(mb *message.Buffer)(cmd CreateCommand) (Model, error)
    CreateAndEmit(cmd CreateCommand) (Model, error)
    Destroy(mb *message.Buffer)(characterId uint32, reason string) (Model, error)
    DestroyAndEmit(characterId uint32, reason string) (Model, error)
    GetByCharacterId(characterId uint32) (Model, error)
    InMapModelProvider(f field.Model) model.Provider[[]Model]
}
```

`Create` under the ADR-4 lock:

1. `blockedMapPrefixes` check → `MAP_FORBIDDEN`.
2. `len(message) > maxMessageLength` → `MESSAGE_TOO_LONG`.
3. registry `Exists(characterId)` → `ALREADY_PLACED`.
4. `len(kites in field) >= maxPerMap` → `MAP_FULL`.
5. `NextID` → build → `registry.Put`.
6. buffer `KITE_CREATED`.

Every refusal buffers `KITE_CREATION_FAILED` instead and returns a typed error;
none of them emits `KITE_CREATED` (FR-3.5). On emit failure after a
successful insert, the insert is rolled back — the explicit precedent is
`mist/processor.go:94-106`:

```go
if err := message.Emit(p.p)(func(buf *message.Buffer) error { ... }); err != nil {
    _, _ = p.r.Remove(p.t, id)   // registry never shows what consumers won't see
    return Model{}, err
}
```

`Destroy` removes first and treats the removal as authoritative; emit failures
are logged, not fatal (same file, `Destroy`).

`DestroyForCharacter` from the PRD collapses into `Destroy(characterId, reason)`
because of the one-per-character invariant — there is no bulk case.

### 4.3 `atlas-kites` — REST

Resource type `kites`, `GetName() == "kites"`. Routes mirror chalkboards':

- `GET /kites/{characterId}` — the owner's kite, 404 if none.
- `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/kites` — paginated
  (`paginate.ParseParams`, `paginate.Slice`, `MarshalPaginatedResponse`),
  sorted by kite `id` before paging so the page boundary is stable.

The PRD's flat `GET /kites?worldId=…` form is replaced by the path-segment form,
because that is what every sibling in-map registry uses and what
`requests.DrainProvider` on the channel side expects.

All handlers are tenant-scoped through `tenant.MustFromContext(ctx)`; keys are
tenant-namespaced by `atlas-redis`, so a cross-tenant read is not expressible.

### 4.4 `atlas-channel` — `kite/` client package

Replaces the dead `channel/kite/model.go` (zero importers, confirmed by
`grep -rn "channel/kite"` → no hits; its `ft`/`Type()` field does not exist on
the wire). The `docs/domain.md:793-803` "Model-only domain" entry is rewritten
to describe the real package.

Files mirror `channel/chalkboard/`: `model.go`, `rest.go` (`Extract`),
`requests.go` (`worlds/%d/channels/%d/maps/%d/instances/%s/kites` — with the
`/instances/` segment present from day one, the bug
`channel/chalkboard/requests.go:9-17` documents having shipped without),
`processor.go` (`InMapModelProvider` via `requests.DrainProvider`,
`ForEachInMap`, `AttemptUse`, `Close`), `producer.go` (commands keyed on
`characterId`).

### 4.5 `atlas-channel` — handler arm

In `CharacterCashItemUseHandleFunc`, alongside the existing
`CashSlotItemTypeChalkboard` arm (`character_cash_item_use.go:93-98`):

```go
const CashSlotItemTypeKite = CashSlotItemType(18)

if it == CashSlotItemTypeKite {
    sp := cashsb.NewItemUseKite(updateTimeFirst)
    sp.Decode(l, ctx)(r, readerOptions)
    c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
    if err != nil { ...; return }
    _ = kite.NewProcessor(l, ctx).AttemptUse(s.Field(), s.CharacterId(), c.Name(),
        uint32(itemId), sp.Message(), c.X(), c.Y())
    return
}
```

Position comes from server-side character state (`Model.X()`/`Model.Y()`,
`character/model.go:244`), the same source
`skill/handler/mysticdoor/mysticdoor.go:47` uses — never from the packet, which
does not carry it anyway (Q3).

Ownership is already verified before this point by the shared
`cashItemInSlotFunc` check (`character_cash_item_use.go:654-661`) that gates the
whole handler (FR-4.2). No item is consumed (FR-4.1): no `saga.DestroyAsset`
step, no inventory mutation, so this is a direct Kafka command and not a saga.

**No `EnableActions`.** The client's kite dialog is modal
(`CDialog::DoModal` at `0x9ed0d9`) and unlocks itself on close; the sibling
chalkboard *use* arm sends none either. Unlocking here would only widen the
client's dup gate.

### 4.6 `atlas-channel` — status consumers

New `kafka/consumer/kite/consumer.go`, structured exactly like
`kafka/consumer/chalkboard/consumer.go` (`consumer.SetStartOffset(kafka.LastOffset)`,
`sc.Is(tenant, worldId, channelId)` guard on every handler):

| Event | Action |
|---|---|
| `CREATED` | `_map.ForSessionsInMap(sc.Field(mapId, instance))` → `KiteSpawnWriter` |
| `DESTROYED` | `_map.ForSessionsInMap(...)` → `KiteDestroyWriter(id, KiteDestroyAnimated)` |
| `CREATION_FAILED` | `session.IfPresentByCharacterId(sc.Channel())(characterId, …)` → `KiteErrorWriter` |

`CREATION_FAILED` is deliberately **not** a map broadcast — it targets the
requesting character only, which is why the event body carries `characterId`.

### 4.7 `atlas-channel` — map-entry replay

One more `routine.Go` block in the map-enter fan-out
(`kafka/consumer/map/consumer.go`, alongside the NPC/monster/summon/drop/
reactor/door/chalkboard/chair passes at :224–:275):

```go
routine.Go(l, ctx, func(_ context.Context) {
    if err := kite.NewProcessor(l, ctx).ForEachInMap(f, spawnKitesForSession(l)(ctx)(wp)(s)); err != nil {
        l.WithError(err).Debugf("SpawnForSelf: unable to spawn kites for character [%d].", s.CharacterId())
    }
})
```

with `spawnKitesForSession` shaped like `spawnChalkboardsForSession`
(:800-810). `ForEachInMap` uses `model.ParallelExecute()`, so the operator must
be free of shared mutable state — the known hazard on this codebase
(PRD §8 Concurrency). The operator here closes over only `s` and `wp` and
constructs a fresh `KiteSpawn` per model, so it is safe by construction; the
plan should carry this as an explicit review point rather than an assumption.

---

## 5. Data flow

**Placement**

```
client USE_CASH_ITEM (type 18, message)
  -> atlas-channel handler arm: decode sub-body, load character for name/x/y
  -> COMMAND_TOPIC_KITE CREATE   (key: characterId)
  -> atlas-kites: lock(field) -> policy -> cap -> one-per-char -> NextID -> Put
  -> EVENT_TOPIC_KITE_STATUS CREATED (key: mapId)   |  CREATION_FAILED
  -> atlas-channel: ForSessionsInMap -> SPAWN_KITE  |  IfPresentByCharacterId -> CANNOT_SPAWN_KITE
```

**Map entry**

```
MAP_CHANGED / login
  -> atlas-channel map consumer SpawnForSelf
  -> GET .../maps/{m}/instances/{i}/kites  (DrainProvider, all pages)
  -> SPAWN_KITE per kite, to the entering session only
```

**Teardown**

```
character status: MapChanged | Logout | ChannelChanged
  -> atlas-kites character consumer: index transition (instance-correct)
  -> Destroy(characterId, OWNER_LEFT | OWNER_LOGGED_OUT)
  -> EVENT_TOPIC_KITE_STATUS DESTROYED (old field)
  -> atlas-channel: ForSessionsInMap(old field) -> REMOVE_KITE(animated)
```

Ordering note: the character index transition must happen **before** the destroy
emit uses the old field, or the DESTROYED event fans out to the wrong map. The
chalkboards consumer gets this right by capturing `of` first; kites does the
same.

---

## 6. Error handling

| Condition | Where | Behaviour |
|---|---|---|
| Sub-body decode short/garbled | channel | logged; no command emitted. The shared `ItemUse` prefix already consumed its bytes, so a malformed tail cannot desync a later handler. |
| Character fetch fails | channel | logged at `Debug`; no command. The player sees nothing, which is the pre-existing behaviour for every failed cash-item arm. |
| Any FR-5 policy refusal | kites | `KITE_CREATION_FAILED` with the specific reason, logged at `Info` with tenant/character/field/reason (the client only sees a generic empty-body error, so the log is the only diagnosis). |
| Lock not acquired | kites | treated as `MAP_FULL`; logged distinctly so contention is separable from a genuinely full map. |
| Registry insert succeeds, emit fails | kites | insert rolled back (`mist/processor.go:94-106` pattern); error returned. |
| Registry remove succeeds, emit fails | kites | logged; removal stands (removal is authoritative). |
| REST list upstream 5xx during replay | channel | logged at `Debug`; the rest of the map-enter fan-out is unaffected (each pass is its own `routine.Go`). |
| Tenant has no `kite-configs` row | kites | `Extract` folds every zero knob to its compiled default (mts-configs precedent). The feature works un-provisioned. |

---

## 7. Testing strategy

**Packet layer** — round-trip tests under every `test.Variants` entry for
`ItemUseKite`; explicit byte fixtures per version in the existing
`v48/v61/v72/v79_test.go` files, covering both the leading- and
trailing-`updateTime` branches. `kite_spawn_test.go` / `kite_v48_test.go` pass
unmodified except for the `kiteType` → `y` identifier rename — that is the
guarantee that FR-2.3's "no encoded bytes changed" holds.

**`atlas-kites`** — table tests over the processor with a real
`TenantRegistry` against a miniredis-style client (the pattern in
`libs/atlas-redis/registry_test.go` and
`atlas-chalkboards/chalkboard/registry_test.go`): each FR-5 refusal reached and
each reason asserted; the rollback path asserted by forcing an emit error and
checking `Exists` is false; a concurrent-create test that runs two goroutines at
a cap boundary and asserts exactly one success.

**`atlas-channel`** — an `httptest`-backed drain test for
`kite.InMapModelProvider` proving multi-page drain and the `/instances/{id}/`
path segment, directly modelled on
`chalkboard/processor_drain_test.go`. A handler test that a type-18 use emits
one `CREATE` command with server-side `x`/`y`/`name` (not packet-derived).

**Test setup uses the project Builder pattern.** No `*_testhelpers.go` files.

---

## 8. Verification gates

Per `CLAUDE.md`, all from the worktree root:

1. `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed
   module (`libs/atlas-packet`, `services/atlas-channel`, `services/atlas-kites`).
2. `docker buildx bake atlas-kites` **and** `docker buildx bake atlas-channel`
   (both `go.mod`s are touched).
3. `tools/service-registration-guard.sh`
4. `tools/redis-key-guard.sh`
5. `tools/goroutine-guard.sh`
6. `tools/lint.sh --check`
7. `tools/template-opcode-order-guard.sh`
8. `tools/template-duplicate-binding-guard.sh`
9. `tools/template-movement-types-guard.sh` (templates changed)
10. `packet-audit` matrix regeneration + `--check`; every `FieldKiteSpawn` cell
    retains its prior status.

---

## 9. Risks and deliberate deviations

**The kite item is not consumed (FR-4.1).** WZ gives these items `slotMax=100`
and `cash=1`, and the retail behaviour is to consume on use. The PRD makes the
opposite call deliberately, gating placement on the per-character cap instead.
This is a coherent product decision and is implemented as specified; noting it
here only so the deviation from retail parity is on the record rather than
discovered later as a bug report.

**No profanity filtering.** The v95 client runs `CCurseProcess::ProcessString`
before sending (`0x9ed1b7`) and refuses locally, so casual cases are already
filtered client-side — but a crafted packet bypasses that entirely, and Atlas
has no server-side equivalent. Out of scope per PRD §2; the 182-byte bound and
the per-character cap are the only server-side griefing limits.

**Pre-v95 serverbound layout is inherited, not verified.** Q3 verifies GMS v95
and establishes that the type-18 arm exists on v83 with the same opening guard.
The six older GMS versions and JMS are covered by per-version byte fixtures
written during implementation; if any of them diverges, the divergence lands as
a gate in `ItemUseKite`, not as a silent mis-decode.

**Chalkboards' instance-blind index is left broken.** Documented in Q6. Fixing
it is a one-line change in another service and belongs to its own change so this
branch's diff stays reviewable.
