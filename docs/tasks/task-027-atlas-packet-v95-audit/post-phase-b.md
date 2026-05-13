# Post-Phase-B Scoping Checkpoint — task-027

This document enumerates the remaining work after Phase A (audit pipeline) +
Phase B (login-domain audit + fixes) shipped on branch
`task-027-atlas-packet-v95-audit`. Use it to scope Phase C–F as sibling tasks.

## Snapshot of audited packets (login domain)

| Packet | Verdict | Notes |
|---|---|---|
| AuthSuccess | ✅ | v95 field-7 width fix shipped (Task 16) |
| ServerListEntry | 🔍 | per-channel world-id fix shipped (Task 17); 🔍 verdict reflects loop-body sub-struct, not a wire bug |
| ServerIP | ✅ | wire-perfect for v95 once analyzer learned `WriteByteArray` (Task 19) |
| CharacterList | 🔍 | sub-struct recurse into CharacterListEntry/CharacterStat/AvatarLook |
| Request (LoginHandle) | ✅ | modified-v95 shape; stock-v95 stubbed in `request_stock.go` (Task 18) — sibling task tracks Nexon-passport validator |
| CharacterSelect | ✅ | no-PIC path wire-correct |

Reports: `docs/packets/audits/gms_v95/`.

## Phase C — Sub-struct audit

Sub-structs marked `🔍 KindRecurse` by the audit pipeline are the highest-leverage
next pass. Per the spike report and SUMMARY snapshot above, these structs are
the gating drift surface for v95:

| Sub-struct | Used by (login + channel) | Drift risk |
|---|---|---|
| `CharacterStat` | CharacterList, channel/spawn writers | high — stat-block layout changes between versions |
| `AvatarLook` | CharacterList, channel/spawn writers | high — equipment slot count, masked-slot rules |
| `ChannelLoad` | ServerListEntry | low — flat 4-field record, already audited at the loop-body level |
| `CharacterListEntry` | CharacterList | medium — composes CharacterStat + AvatarLook + rank block |

Scope: enumerate every type in `libs/atlas-packet/**` whose Encode/Decode is
called as a method on a struct field (the pipeline's `KindRecurse` markers
identify these directly). One sibling task per high-frequency type.

## Phase D — Per-domain clientbound audit

Login is shipped. Channel-domain clientbound writers are the next bulk pass.
One sibling task per domain:

- `libs/atlas-packet/character/clientbound/` — character ops (rename, spawn, equip change, stat update)
- `libs/atlas-packet/monster/clientbound/` — monster spawn/move/damage
- `libs/atlas-packet/drop/clientbound/` — item drops, pick-up packets
- `libs/atlas-packet/field/clientbound/` — map changes, weather, hidden
- `libs/atlas-packet/inventory/clientbound/` — inventory ops, item move/use
- `libs/atlas-packet/pet/clientbound/` — pet ops
- `libs/atlas-packet/reactor/clientbound/` — reactor state, hit
- `libs/atlas-packet/quest/clientbound/` — quest start/complete/update
- `libs/atlas-packet/party/clientbound/` — party invite/join/leave
- `libs/atlas-packet/guild/clientbound/` — guild ops
- `libs/atlas-packet/buddy/clientbound/` — buddy list ops
- `libs/atlas-packet/chat/clientbound/` — chat, whispers
- `libs/atlas-packet/messenger/clientbound/` — messenger
- `libs/atlas-packet/note/clientbound/` — mail notes
- `libs/atlas-packet/merchant/clientbound/` — hired merchant
- `libs/atlas-packet/interaction/clientbound/` — trade, player shops
- `libs/atlas-packet/fame/clientbound/` — fame system
- `libs/atlas-packet/storage/clientbound/` — storage NPC
- `libs/atlas-packet/cashshop/clientbound/` — cash shop
- `libs/atlas-packet/ui/clientbound/` — UI notifications
- `libs/atlas-packet/socket/clientbound/` — keep-alive, error

Prereq: expand `docs/packets/ida-exports/gms_v95.json` to cover the relevant
FNames. The current export covers only 6 spike functions; sibling tasks must
refresh via `packet-audit export --ida-source mcp ...` on a maintainer
workstation, or hand-derive from focused spike sessions.

## Phase E — Per-domain serverbound audit

Symmetric to Phase D; same domain split. Sibling tasks pair (Phase D → Phase E)
per domain since clientbound + serverbound for a feature typically land
together.

## Phase F — Stock-Nexon v95 support (recommendation)

**Recommend splitting Phase F into a sibling task** `task-NNN-atlas-packet-stock-nexon-v95`.

Trigger condition: split as a sibling task as soon as any Phase C/D/E sub-task
is open for review. Rationale: stock-v95 support requires both:
1. Real `decodeStock` implementation for `LoginHandle.Request` (currently stubbed)
2. Server-side Nexon-passport validation (out-of-band integration)

Neither belongs in a per-domain audit pass. Carry the `_pending.md` entry
`CLogin::SendCheckPasswordPacket (stock variant)` forward to that sibling task.

## What this task delivered

- Audit pipeline (`tools/packet-audit/`) — reusable for any GMS/JMS version
- `clientVariant` plumbing — template field, tenant accessor, version helper
- Six spike-confirmed login packets audited; three concrete fixes shipped
- Documented pending IDA exports in `docs/packets/ida-exports/_pending.md`
- This checkpoint document

## Decision point

The user decides whether to:
- Continue inside this task by opening per-domain sub-tasks, or
- Close this task as "Phase A + B shipped; Phase C–F enumerated above" and spec
  sibling tasks via `/spec-task`.

The plan does not prescribe; this document is the artifact for either path.
