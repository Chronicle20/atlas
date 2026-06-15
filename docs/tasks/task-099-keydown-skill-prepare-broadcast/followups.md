# task-099 — Follow-ups (operational + validation)

Code is complete, tested, and the coverage matrix is promoted (15 cells ✅). These items are NOT code in this branch — they're deploy-time / runtime-validation steps, plus one parked version.

## 1. Live tenant config patch (REQUIRED before existing tenants get the behavior)
Handlers/writers do NOT hot-reload from the config projection — only NEW tenants pick up seed-template changes at creation. For each EXISTING tenant on a supported version, patch the live channel config to add the new rows, then restart the channel:
- handler: `CharacterSkillPrepareHandle` at the serverbound prepare opcode (v83 `0x5D`, v84 `0x5D`, v87 `0x60`, v95 `0x69`, jms185 `0x58`), validator `LoggedInValidator`.
- writers: `CharacterSkillPrepareForeign` at clientbound prepare (v83 `0xBE`, v84 `0xC2`, v87 `0xCB`, v95 `0xD7`, jms185 `0xC4`) and `CharacterSkillCancelForeign` at clientbound cancel (v83 `0xBF`, v84 `0xC3`, v87 `0xCC`, v95 `0xD9`, jms185 `0xC5`).
- (No new serverbound cancel handler — the keyup rides the existing `CharacterBuffCancel`. On **v95**, note this branch ALSO fixed a missing-validator on that row so it now registers; existing v95 tenants need that patched too or the cancel won't fire.)
See `bug_new_opcodes_not_in_live_tenant_config` for the symptom (client action no-ops, "unhandled message op 0xXX").

## 2. In-map manual validation
Two characters in one map. With the deployed build + patched config:
- Bowmaster casts **Hurricane** → the observer sees the looping cast aura START on keydown and STOP on key release. (This is the original bug.)
- Spot-check across the keydown family: a warrior **Monster Magnet**, a mage **BigBang**, a Corsair **Rapid Fire**. Observer should see the prepare aura start/stop for each.
- Confirm the **arrows/projectiles still render** (no regression to the attack broadcast).

## 3. Death / stun while keydown active (design D6 residual)
D6 implemented the keyup-cancel relay and relies on avatar removal for disconnect/map-leave (no server keydown state). The one unverified path: caster **dies or is stunned mid-keydown while staying in the map**. Expected: the client sends its own cancel (keydown interrupt) → relayed → aura clears. VALIDATE empirically; only if a stuck aura is observed, add server-side cancel synthesis on the death/stun event (do not add speculative state otherwise).

## 4. v92 — PARKED (design D7)
v92 has no client IDB, so its prepare/cancel opcodes/read-order can't be IDB-verified and were intentionally NOT wired (no ported assumptions). v92 keydown skills keep the no-aura bug. Unblocks when a v92 IDB exists: re-run the wire-spec pin + Tasks 5/6 for v92 only.

## 5. Audit-report verdict note (informational)
The generated audit reports for these 3 packets show ❌ static verdicts because the analyzer can't model the serverbound prepare's `swallowMobId` runtime guard (`skillId==33101005`, v95/jms only) nor the dispatcher-consumed `charId` prefix on the clientbound foreign relays. These are tier-1 cells that promote on marker + fresh evidence (the byte-fixture tests are the authoritative verification); the ❌ report verdicts are advisory per the grading rules and do not gate the cell. No action needed.
