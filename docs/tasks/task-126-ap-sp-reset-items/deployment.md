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

## 4. Known limitation — skill-macro echo writer not wired on gms_95 (pre-existing gap, out of scope)

Part of this feature's SP-reset flow (FR-18, macro cleanup visibility) re-pushes the character's
skill macros to the client via the `CharacterSkillMacro` clientbound writer
(`charpkt.CharacterSkillMacroWriter`) when the macro-status Kafka topic fires an `UPDATED`
event. This writer is **not present** in `template_gms_95_1.json` (nor in
`template_gms_87_1.json`, `template_gms_92_1.json`, or `template_jms_185_1.json`) — it exists
only in `template_gms_83_1.json` (opcode `0x7C`) and `template_gms_84_1.json` (opcode `0x7F`),
confirming the opcode is version-shifted and cannot be guessed for other versions without IDA
verification.

This is a **pre-existing gap**, not introduced by this task: `CharacterSkillMacroWriter` is
already called unconditionally at character login (`kafka/consumer/session/consumer.go:320-339`)
regardless of tenant version, so gms_95 (and gms_87/92/jms) tenants already silently fail to
receive the login-time macro push today, independent of the AP/SP-reset feature. The
`session.Announce` write is non-fatal on a missing writer (same "silently dropped" behavior as
an unrouted handler opcode) — the macro data itself is correctly persisted server-side, but the
client's macro UI does not refresh until the next full login on these versions.

This task does not fix that gap: doing so would require IDA-verifying the `CharacterSkillMacro`
writer opcode for gms_95 (and separately for gms_87/92/jms), which is outside this task's scope
(wiring the point-reset serverbound handler) and outside what can be produced without an IDB.
It is called out here so the AP/SP-reset feature's known behavior on gms_95 is documented
accurately: AP reset and SP reset both complete correctly (stat/skill changes, pink-text
messaging, and action re-enable all use writers confirmed present in `template_gms_95_1.json` —
`StatChanged` opcode `0x1E`, `WorldMessage` opcode `0x47`, `CharacterSkillChange` opcode
`0x23`), but the client's macro list will not visibly refresh until the player's next login if
the reset invalidated a bound macro.
