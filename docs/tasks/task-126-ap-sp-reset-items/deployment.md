# Task 126 (AP/SP reset items) — Deployment / Park Notes

## Scope of this change

This task wires the `CharacterCashItemUseHandle` serverbound handler at opcode `0x55` onto
the **gms_95** seed template only (`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`).

This is a controller-directed scope reduction (Option A) from the original Task 16 plan, which
called for wiring gms_87 (0x52), gms_95 (0x55), and jms_185 (0x47). Reason: the serverbound
`ItemUsePointReset` codec (the body layout the AP/SP-reset items rely on) is IDA-verified only
for gms_83 and gms_95. gms_87 and jms_185 have no IDB and the sender fname
(`CItemSpeakerDlg::_SendConsumeCashItemUseRequest` per `docs/packets/registry/gms_v87.yaml` and
`docs/packets/registry/jms_v185.yaml`) is absent from all checked-in IDA exports for those
versions, so their wire layout for the point-reset request body is unverified. Wiring an
unverified codec into a live handler path would violate the project's verify-don't-invent rule
(see repo `CLAUDE.md` "Grounding & Honesty (No Inventing)").

## 1. Live tenant patch — gms_95 only

Existing gms_95 tenants do **not** re-seed from the updated template; the new handler row must
be applied to each live gms_95 tenant's socket configuration via the atlas-tenants
configurations REST surface.

Row to append (identical shape to the seed-template entry added in this task):

```json
{
  "opCode": "0x55",
  "validator": "LoggedInValidator",
  "handler": "CharacterCashItemUseHandle"
}
```

Procedure per live gms_95 tenant:

1. `GET /tenants/{tenantId}/configurations/{resourceName}` (the socket/handlers configuration
   resource — same resource the seed templates populate at tenant creation) to fetch the
   tenant's current handler list.
2. Append the row above to the `handlers` array, preserving all existing entries and the
   array's ordering convention (numerically adjacent to the existing `0x53`/`0x5D` entries, as
   in the updated `template_gms_95_1.json`).
3. `PATCH /tenants/{tenantId}/configurations/{resourceName}` with the updated document.
4. **Restart atlas-channel** for that tenant/cluster. Handler wiring is resolved once at
   config load; it does not hot-reload on a live PATCH (see the project's documented pattern:
   new opcodes missing from a live tenant config are silently unroutable until the service
   restarts).
5. Confirm via atlas-channel startup logs that the handler map includes opcode `0x55` for the
   tenant (absence of a "validator not found" / missing-handler warning at startup).

Do not patch gms_92 tenants — see the park section below. gms_87 and jms_185 are now
**SHIPPED** (IDA-verified) — see section 2A.

## 2A. SHIPPED versions — gms_87 (0x52) and jms_185 (0x47) — IDA-verified (task-16 unblock)

The v87 and jms_185 IDBs are now loaded, so both versions were IDA-verified and wired in this
unblock pass (superseding the earlier controller-directed park). Both are seeded automatically
onto new tenants and must be PATCHed onto live tenants (procedure below).

| Version | Handler opcode | Sender fname / IDA addr | Codec (sub-body) | Macro writer opcode / IDA addr |
|---|---|---|---|---|
| gms_87 | `0x52` | `CWvsContext::SendConsumeCashItemUseRequest` @`0xa9fef9` | `Encode4(to) + Encode4(from)`; **update_time in header (first)** | `0x84` — `case 0x84`→`OnMacroSysDataInit` @`0xac0d6e` in `OnPacket` @`0xa9d011`; `ProcessPacket` @`0x4a8622` passes raw `Decode2` opcode (no offset) |
| jms_185 | `0x47` | `CWvsContext::SendConsumeCashItemUseRequest` @`0xaef2f5` | `Encode4(to) + Encode4(from)`; **update_time in header (first)** | `0x7A` — `case 0x7A`→`OnMacroSysDataInit` @`0xb10384` in `OnPacket` @`0xaebfe7`; `ProcessPacket` @`0x4b17eb` passes raw `Decode2` opcode (no offset) |

### KEY FINDING — hypothesis overturned: v87 and jms_185 are update_time-FIRST, not trailing

The task hypothesis assumed both versions carried a **trailing** update_time in the point-reset
sub-body (matching gms_83). IDA disproved this: on **both** gms_87 (@`0xa9fef9`) and jms_185
(@`0xaef2f5`), `CWvsContext::SendConsumeCashItemUseRequest` encodes
`Encode4(get_update_time())` in the packet **header** (before the `get_consume_cash_item_type`
switch), then the AP-reset (`case 0x17`) and SP-reset (`case 0x18`) arms encode only
`Encode4(to) + Encode4(from)`. The send tail (gms_87 `LABEL_41`; jms_185 `LABEL_528`) contains
**no** trailing `Encode4(update_time)`. This is the same header-first layout already verified for
gms_95. Only gms_83/gms_84 keep update_time in the send tail (trailing).

Consequently the shared header gate was **corrected**: the update_time-first predicate in
`libs/atlas-packet/cash/serverbound/item_use.go` (`ItemUse.Encode`/`ItemUse.Decode`) and
`services/atlas-channel/.../socket/handler/character_cash_item_use.go` (`updateTimeFirst`) was
changed from `Region()=="GMS" && MajorVersion()>=95` to **`MajorVersion() >= 87`**. This yields:
gms_83/84 → trailing (unchanged); gms_87/95 → header-first; jms_185 (185) → header-first. Without
this fix a v87/jms tenant would misparse the header (reading the 4 update_time bytes as
source+itemId) and never reach the point-reset logic — so the fix is required to ship, not
cosmetic. The change only flips versions ≥ 87 that had no cash-item-use handler wired before
(v87 newly wired here; v92 parked; v95 already header-first), so no working path regresses.

### Coverage-matrix note (v87/jms cells stay ❌ — same tooling gap as v95)

Because both versions are update_time-first, their `cash/serverbound/CashItemUsePointReset` cells
CANNOT be promoted via a `packet-audit:verify` marker: the codec gates the trailing write on the
runtime bool `updateTimeFirst`, which the version-based (not value-based) analyzer cannot
evaluate — it statically counts three writes and grades the report FlatInvalid. This is the exact
pre-existing tooling gap the gms_95 fixture documents, so the v87/jms cells remain `❌` alongside
v95 in `docs/packets/audits/STATUS.md`. The read order is nonetheless IDA-verified and pinned by
byte-exact fixtures (`TestItemUsePointResetBytesV87`, `TestItemUsePointResetBytesJMS185`).
`matrix --check` stays exit 0 (no drift).

### Live tenant patch — gms_87 and jms_185 (required)

Existing v87/jms tenants do **not** re-seed. Per live tenant, append BOTH rows and restart
atlas-channel (handler + writer wiring resolve once at config load; no hot-reload):

- gms_87 handler: `{"opCode": "0x52", "validator": "LoggedInValidator", "handler": "CharacterCashItemUseHandle"}`
- gms_87 writer: `{"opCode": "0x84", "writer": "CharacterSkillMacro"}`
- jms_185 handler: `{"opCode": "0x47", "validator": "LoggedInValidator", "handler": "CharacterCashItemUseHandle"}`
- jms_185 writer: `{"opCode": "0x7A", "writer": "CharacterSkillMacro"}`

Procedure is identical to section 1 (GET → append to `handlers`/`writers` arrays → PATCH →
**restart atlas-channel**).

## 2. Parked version — gms_92 (no opcode)

| Version | Registry opcode | Status | Reason |
|---|---|---|---|
| gms_92 | none | Parked | No IDB and no `USE_CASH_ITEM` registry row exists for gms_92 at all; there is no opcode value to wire, verified or otherwise. |

On gms_92, the AP/SP-reset cash items stay inert exactly as they do today: no handler is wired
for the opcode (the opcode itself is unknown), so the request is dropped server-side with no
effect — no crash, no silent partial application.

gms_92 unblocks the same way the pre-existing v92 mount-food park does (see project memory
`bug_v92_mount_food_parked`): once a gms_92 IDB exists and the `ItemUsePointReset` codec + macro
writer opcode are IDA-verified against it, the rows can be added to `template_gms_92_1.json` and
PATCHed onto any live tenants. Note the header-gate fix already treats gms_92 (92 ≥ 87) as
update_time-first, matching the v87→v95 trend, but this is unverified for v92 and has no runtime
effect while v92 stays unwired.

`gms_83` and `gms_84` are not part of this park list: `gms_83`'s handler is IDA-verified and was
wired before this feature (pre-existing `template_gms_83_1.json` row, opcode `0x4F`). `gms_84`
is byte-identical to `gms_83` for this packet family per the task-083 structural audit, and its
handler is also already wired pre-feature (`template_gms_84_1.json`, opcode `0x4F`) — no change
needed for either version in this task.

## 3. New-tenant behavior

- New **gms_95** tenants seed the `CharacterCashItemUseHandle` handler automatically from the
  updated `template_gms_95_1.json` — no manual step required.
- New **gms_87** and **jms_185** tenants now seed BOTH the `CharacterCashItemUseHandle` handler
  (0x52 / 0x47) and the `CharacterSkillMacro` writer (0x84 / 0x7A) automatically from their
  updated templates — no manual step required (task-16 unblock, section 2A).
- New **gms_92** tenants do **not** get this handler seeded — parked per section 2 until a gms_92
  IDB exists.
- New **gms_83**/**gms_84** tenants already seed the handler as before (unchanged, pre-existing
  rows).

## 4. CharacterSkillMacro clientbound writer wired on gms_95 (FR-18 macro refresh — RESOLVED)

Part of this feature's SP-reset flow (FR-18, macro cleanup visibility) re-pushes the character's
skill macros to the client via the `CharacterSkillMacro` clientbound writer
(`charpkt.CharacterSkillMacroWriter`, `libs/atlas-packet/character/skill_macro.go` —
`CUser::SendSkillMacroModifiedMessage` family) when the macro-status Kafka topic fires an
`UPDATED` event. It is also called unconditionally at character login
(`kafka/consumer/session/consumer.go:320-339`).

This writer was **previously absent** from `template_gms_95_1.json`, so both FR-18's live
macro refresh and the pre-existing login-time macro push silently no-op'd on gms_95 (the
`session.Announce` write is non-fatal on a missing writer — same "silently dropped" behavior as
an unrouted handler opcode; the macro data itself persists server-side but the client's macro UI
never refreshes). This task **fixes that gap** for gms_95.

**IDA-verified opcode (live `GMS_v95.0_U_DEVM.exe` IDB):** the clientbound macro packet
(`CWvsContext::OnMacroSysDataInit` @`0x9f0c70`, which calls `CMacroSysMan::SetMacro(CInPacket&)`)
is dispatched from `CWvsContext::OnPacket` (@`0x9e5830`) at `case 140:` (dispatch site
@`0x9e5ad6`). `CClientSocket::ProcessPacket` (@`0x4b00f0`) passes the raw wire opcode from
`CInPacket::Decode2` directly into `CWvsContext::OnPacket` with no offset (`v5 = v4;` →
`OnPacket(..., v5, iPacket)`), and the default-branch guard `(v5 - 28) > 0x70` bounds the
CWvsContext range to opcodes 28..140, with macro as the exact upper bound. So the v95 clientbound
macro opcode is **140 = `0x8C`**. Cross-check: `0x8C` falls between `ScriptProgress` (`0x7F`) and
`SetField` (`0x8D`) in the v95 writers array, matching gms_83's ordering (macro sits between
`ScriptProgress 0x7A` and `SetField 0x7D`). The opcode is version-shifted (gms_83 uses `0x7C`),
confirming it could not have been copied and had to be read from the v95 client.

Row added to `template_gms_95_1.json`'s top-level `writers` array:

```json
{
  "opCode": "0x8C",
  "writer": "CharacterSkillMacro"
}
```

### Live gms_95 tenant patch (required)

Existing gms_95 tenants do **not** re-seed from the updated template. This writer row must be
PATCHed into each live gms_95 tenant's socket configuration the same way as the
`CharacterCashItemUseHandle` handler row (section 1): `GET` the socket/writers configuration
resource, append the `{"opCode": "0x8C", "writer": "CharacterSkillMacro"}` entry (preserving
existing entries and numeric ordering), `PATCH` it back, then **restart atlas-channel** for that
tenant/cluster — writer wiring is resolved once at config load and does not hot-reload on a live
PATCH.

### gms_87 / jms_185 macro writer — now SHIPPED (task-16 unblock)

The `CharacterSkillMacro` writer is now IDA-verified and wired for both versions (section 2A):
gms_87 = `0x84` (`case 0x84`→`OnMacroSysDataInit` @`0xac0d6e`), jms_185 = `0x7A`
(`case 0x7A`→`OnMacroSysDataInit` @`0xb10384`). The opcode is version-shifted per version
(gms_83 `0x7C`, gms_95 `0x8C`, gms_87 `0x84`, jms_185 `0x7A`), confirming each had to be read from
its own client. FR-18's live macro refresh and the login-time macro push now work on gms_87 and
jms_185.

### Parked version — FR-18 macro refresh still degraded (gms_92 only)

On the remaining parked version (`gms_92`) the `CharacterSkillMacro` writer is **still absent**
and cannot be added without a gms_92 IDB to IDA-verify the version-specific opcode. FR-18's live
macro refresh and the login-time macro push therefore remain degraded on gms_92 — consistent with
the point-reset handler park in section 2. They unblock together once a gms_92 IDB exists.
(`gms_83`/`gms_84` already carry the writer — `0x7C`/`0x7F` respectively.)

With the writer now wired on gms_95, AP reset and SP reset both complete correctly *and* the
client's macro list refreshes live on gms_95 (via the FR-18 `UPDATED` push and at login), in
addition to the stat/skill/pink-text writers already confirmed present in
`template_gms_95_1.json` (`StatChanged 0x1E`, `WorldMessage 0x47`, `CharacterSkillChange 0x23`).

## 3. Post-ship corrections (client-verified via GMS v83 IDB)

Two follow-ups after reversing the v83 client's AP-reset dialog
(`CWvsContext::SendConsumeCashItemUseRequest` → `case 0x17` arm @`0xa0c427` →
modal stat-reset dialog; button-enable gate `sub_8CBDDB` @`0x8cbddb`):

### 3.1 Magician MP-reset-out loss is INT-scaled (bug fix)

Every branch's HP/MP reset loss matched §4.3 **except magician MP loss**. §4.3
hardcoded a flat `takeMp = 31`; the client (`sub_8CE5BD` @`0x8ce5bd`, branch-2
arm) computes it as:

```
takeMp = 3 * effectiveInt / 40 + 30      (integer division)
```

where `effectiveInt` is the character's total INT (base + equipment), read from
the cached secured field `CWvsContext+0x20F8` — confirmed by the equip-tooltip
renderer `sub_8ED0D2` @`0x8ed0d2`, which reads the STR/DEX/INT/LUK effective-stat
array at `szCookie[156/168/180/192]` (12-byte stride; `szCookie[180]` = `0x20F8`
= INT). `31` is only correct at `effectiveInt ≈ 14`; a higher-INT mage desyncs
MaxMP against the client. Fixed in `character/point_reset.go`
(`pointResetMagicianTakeMp`, `isPointResetMagician`) + `character/processor.go`
(`TransferAP` fetches effective INT via atlas-effective-stats, base-INT fallback).
All gain values and HP loss stay constant (they match the client).

### 3.2 Preset AP spread de-inflated (gms_83 / gms_84)

The 4th-job presets set STR/DEX/INT/LUK = 999 each (sum 3996), which the client
reads as ~4× the AP a level-200 character earns — so its AP-reset dialog disables
HP/MP as a source (gate: `STR+DEX+INT+LUK+AP ≥ 5·level + 20 + v6`, where
`v6 = 5·(job advancement digit)`; for L200 4th-jobs = **1030**). Rewrote each
preset to a realistic spread summing to 1030: secondary stat = max requirement of
the equipset (atlas-data `reqStr/reqDex/reqInt/reqLuk`), the two off-stats = base
4, remainder → primary. HP/MP left unchanged. Only gms_83 and gms_84 carry
presets; the other templates have none.
