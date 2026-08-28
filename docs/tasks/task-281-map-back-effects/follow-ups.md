# Follow-ups from task-281 (map back-effect packets)

## SET_MAP_OBJECT_VISIBLE (out of scope, cheap to pick up next)

`SET_MAP_OBJECT_VISIBLE` — client handler `CMapLoadable::OnSetMapObjectVisible`,
tracked in `docs/packets/audits/STATUS.md:227` — is **case 145 in the same
`CMapLoadable::OnPacket` switch** that this task (task-281) decompiled for
`SET_BACK_EFFECT` / `CLEAR_BACK_EFFECT`. It is currently ❌ (unsupported) on
every version where the client exposes it:

| Version | Opcode | Router address of `OnSetMapObjectVisible` |
|---|---|---|
| gms_v48 | — (no arm in this version's field router) | not present |
| gms_v61 | — (no arm; explicitly absent — see `docs/tasks/task-281-map-back-effects/structures/gms_v61.md:58`) | not present |
| gms_v72 | — (no arm recorded in this task's switch transcription) | not present |
| gms_v79 | — (no arm recorded in this task's switch transcription) | not present |
| gms_v83 | `0x081` (case 129) | `0x64446c` (`docs/tasks/task-281-map-back-effects/structures/gms_v83.md:18`) |
| gms_v84 | `0x084` (case 132) | `0x65a249` (`docs/tasks/task-281-map-back-effects/structures/gms_v84.md:20`) |
| gms_v87 | `0x089` (case `0x89`) | `0x67db82` (`docs/tasks/task-281-map-back-effects/structures/gms_v87.md:18`) |
| gms_v92 | `0x090` (case 144) | `0x613bcd` (`docs/tasks/task-281-map-back-effects/structures/gms_v92.md:23`) |
| gms_v95 | `0x091` (case 145) | address not annotated in this task's transcription of the design record (`docs/tasks/task-281-map-back-effects/structures/gms_v95.md:20`) |
| jms_v185 | `0x07F` (case `0x7F`) | `0x6ba126` (`docs/tasks/task-281-map-back-effects/structures/jms_v185.md:19`) |

Per design §2.4, `SET_MAP_OBJECT_VISIBLE` was explicitly excluded from task-281's
scope, which covered only `SET_BACK_EFFECT` and `CLEAR_BACK_EFFECT`. It is called
out here as the next-cheapest adjacent packet to implement: the router switch it
lives in (`CMapLoadable::OnPacket`) is already decompiled and cross-checked
against the opcode registries for eight of the ten versions in this task's
structures records (`docs/tasks/task-281-map-back-effects/structures/`), so a
follow-up task can reuse that router work directly instead of re-deriving it.

Note: the brief for this follow-up cited `docs/packets/audits/STATUS.md:211` as
the `SET_MAP_OBJECT_VISIBLE` row; at the time this file was written the row had
moved to line 227 (confirmed by `grep -n SET_MAP_OBJECT_VISIBLE
docs/packets/audits/STATUS.md`). The row content (opcode/status table) is
otherwise unchanged from what the brief described.
