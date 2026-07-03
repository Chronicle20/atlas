# task-129 rollout — Vicious Hammer live-tenant configuration patch

Seed templates (`services/atlas-configurations/seed-data/templates/template_gms_*.json`)
already carry the `ItemUpgradeUpdateHandle` handler and the extended `ViciousHammer`
writer `operations` block for gms_v83 / gms_v84 / gms_v87 / gms_v95, so **new**
tenants created after this branch lands get the feature automatically. Seed
templates only apply at tenant **creation** time — existing (already-provisioned)
tenants do not pick up template changes retroactively. Each live tenant needs an
explicit configuration PATCH plus an `atlas-channel` pod restart, because
handlers/writers are read once at startup and do not hot-reload
(see `bug_new_opcodes_not_in_live_tenant_config` in project memory for the same
failure mode on a prior feature).

## Critical correction vs. the original plan

The plan assumed a single uniform opcode/mode table across GMS versions. IDA
verification during task-129 (Tasks 13–15) proved the opcodes AND the
`operations` mode bytes are **version-dependent** — do not copy one version's
values onto another. In particular:

- v84's serverbound/clientbound opcodes (`0x10B` / `0x16C`) are **not** the
  same as v83's (`0x104` / `0x162`) — v84 has a shifted opcode table
  vs v83 (see `bug_v84_opcode_table_shifted_vs_v83` in project memory).
- v87 and v95 use **different SUCCESS/FAILURE mode bytes** (63/64 and 65/66
  respectively) than v83/v84 (61/62) — this mirrors the general pattern that
  dispatcher `operations` tables are per-version, not uniform
  (`feedback_dispatcher_config_drive_all_modes` in project memory).

All values below are taken verbatim from the seed templates already committed
on this branch (verified by direct read of each `template_gms_*_1.json` during
this task's verification pass):

| Version | serverbound `ITEM_UPGRADE_UPDATE` opcode (handler) | clientbound `VICIOUS_HAMMER` opcode (writer) | `operations` OPEN / SUCCESS / FAILURE |
|---|---|---|---|
| gms_v83 | `0x104` | `0x162` | 0 / 61 / 62 |
| gms_v84 | `0x10B` | `0x16C` | 0 / 61 / 62 |
| gms_v87 | `0x112` | `0x177` | 0 / 63 / 64 |
| gms_v95 | `0x128` | `0x1A9` | 0 / 65 / 66 |

## Procedure — for EACH live GMS tenant (v83 / v84 / v87 / v95)

1. **PATCH the tenant's socket configuration** (via atlas-tenants / the config
   UI), using the row from the table above for that tenant's version:
   - Add to `socket.handlers`:
     ```json
     {
       "opCode": "<per-version serverbound opcode from the table>",
       "validator": "LoggedInValidator",
       "handler": "ItemUpgradeUpdateHandle"
     }
     ```
   - Extend the existing `ViciousHammer` writer entry (identified by its
     per-version clientbound opcode from the table) with an `options.operations`
     block using that version's SUCCESS/FAILURE values:
     ```json
     {
       "opCode": "<per-version clientbound opcode from the table>",
       "writer": "ViciousHammer",
       "options": {
         "operations": {
           "OPEN": 0,
           "SUCCESS": <per-version SUCCESS from the table>,
           "FAILURE": <per-version FAILURE from the table>
         }
       }
     }
     ```
   Do **not** reuse v83's `0x104`/`0x162`/61/62 for v84, and do **not** reuse
   61/62 for v87 or v95 — use exactly the row for that tenant's version. v84's
   opcodes are `0x10B`/`0x16C` (not `0x104`/`0x169`); v87 uses SUCCESS/FAILURE
   `63`/`64` (not `61`/`62`); v95 uses `65`/`66` (not `61`/`62`).

2. **Restart the tenant's `atlas-channel` pods.** Handlers and writers are
   resolved once from the loaded tenant configuration at process start; the
   config-status Kafka projection does not push these into a running process.

3. **Smoke test per version, in-game:**
   - Double-click a Vicious Hammer, drop an eligible equip into the target
     slot, click Upgrade → the upgrade gauge fills → a success notice is
     shown.
   - The equip's upgrade-slot count increases by one, visible in the equip
     window immediately, without relogging.
   - Use a third hammer on the same equip (cap is 2 successful uses per
     item) → the client shows "2 upgrade increases have been used already"
     and the hammer is **not** consumed (verify inventory count is
     unchanged).
   - Target the equip with item id `1122000` (Horntail Necklace) → the
     dedicated Horntail refusal notice is shown instead of the generic
     failure notice, and the hammer is not consumed.

## Versions explicitly NOT patched by this rollout

- **jms tenants**: not patched. `VICIOUS_HAMMER` does not exist in the jms
  client's packet registry — the feature is version-absent on jms, not just
  unconfigured. There is no `ItemUpgradeUpdateHandle`/`ViciousHammer` entry in
  `template_jms_185_1.json`, and none should be added; the flow cannot
  complete on that client build.
- **gms_v92 tenants**: not patched. `template_gms_92_1.json` is a
  login-only stub (870 lines vs. 2500+ for v83/v84/v87/v95) with no
  `CASH_ITEM_USE`-family routing at all — there is no existing cash-item-use
  wiring for the Vicious Hammer flow to attach to on this version. This
  matches the documented v92 gap for other cash-item features
  (`project_v92_mount_food_parked` in project memory): v92 stays parked until
  a v92 IDB/template exists to verify against.

## Provenance

The per-version opcode/mode divergence in the table above was IDA-verified
during task-129 (design.md §2.3 "Global Constraints", plan.md Tasks 13–15).
The original plan assumed a single opcode/mode table shared across all GMS
versions; that assumption was wrong and was corrected before templates were
authored. Do not fall back to the uniform-table assumption when patching
live tenants.
