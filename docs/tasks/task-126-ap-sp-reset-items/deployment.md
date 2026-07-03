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

Do not patch gms_87, jms_185, or gms_92 tenants — see the park section below.

## 2. Parked versions — gms_87 (0x52), jms_185 (0x47), gms_92 (no opcode)

| Version | Registry opcode | Status | Reason |
|---|---|---|---|
| gms_87 | `0x52` (`USE_CASH_ITEM`, csv-import provenance, `docs/packets/registry/gms_v87.yaml`) | Parked | No v87 IDB checked in; sender fname `CItemSpeakerDlg::_SendConsumeCashItemUseRequest` cannot be located in any checked-in export, so the point-reset request body layout is unverified for this version. |
| jms_185 | `0x47` (`USE_CASH_ITEM`, csv-import provenance, `docs/packets/registry/jms_v185.yaml`) | Parked | Same as gms_87 — no jms IDB checked in, sender fname unresolved in checked-in exports. |
| gms_92 | none | Parked | No IDB and no `USE_CASH_ITEM` registry row exists for gms_92 at all; there is no opcode value to wire, verified or otherwise. |

On these three versions, the AP/SP-reset cash items stay inert exactly as they do today: the
client can send the request, but no handler is wired for the opcode on these tenants (gms_92
because the opcode itself is unknown), so the request is dropped server-side with no effect —
no crash, no silent partial application.

Each park unblocks the same way the pre-existing v92 mount-food park does (see project memory
`bug_v92_mount_food_parked`): once a corresponding IDB exists for that version and the
`ItemUsePointReset` codec is IDA-verified against it (handler wiring + byte-level fixture),
these rows can be added to their respective seed templates and PATCHed onto any live tenants.

`gms_83` and `gms_84` are not part of this park list: `gms_83`'s handler is IDA-verified and was
wired before this feature (pre-existing `template_gms_83_1.json` row, opcode `0x4F`). `gms_84`
is byte-identical to `gms_83` for this packet family per the task-083 structural audit, and its
handler is also already wired pre-feature (`template_gms_84_1.json`, opcode `0x4F`) — no change
needed for either version in this task.

## 3. New-tenant behavior

- New **gms_95** tenants seed the `CharacterCashItemUseHandle` handler automatically from the
  updated `template_gms_95_1.json` — no manual step required.
- New **gms_87**, **jms_185**, and **gms_92** tenants do **not** get this handler seeded — they
  remain parked per section 2 until their respective codecs are IDA-verified.
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

### Parked versions — FR-18 macro refresh still degraded

On the parked versions (`gms_87`, `jms_185`, `gms_92`) the `CharacterSkillMacro` writer is
**likewise still absent** and cannot be added without an IDB to IDA-verify the version-specific
opcode (the opcode is version-shifted, as proven above). FR-18's live macro refresh and the
login-time macro push therefore remain degraded on those versions — consistent with, and for the
same reason as, the point-reset handler park in section 2. They unblock together: once an IDB
exists for that version, the macro writer opcode can be IDA-verified and wired alongside the
point-reset handler. (`gms_83`/`gms_84` already carry the writer — `0x7C`/`0x7F` respectively.)

With the writer now wired on gms_95, AP reset and SP reset both complete correctly *and* the
client's macro list refreshes live on gms_95 (via the FR-18 `UPDATED` push and at login), in
addition to the stat/skill/pink-text writers already confirmed present in
`template_gms_95_1.json` (`StatChanged 0x1E`, `WorldMessage 0x47`, `CharacterSkillChange 0x23`).
