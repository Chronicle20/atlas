# Missing features audit: NPC content

Theme: NPC-driven content that reads or writes free-form player input (`cm.sendGetText`
in Cosmic terms), and the subsystems that content depends on.

**How this was audited.** This document does not re-run a fresh WZ/packet sweep; it
consolidates the findings already produced and reviewed during task-284 (`askText`
free-text conversation state), specifically `docs/tasks/task-284-npc-ask-text-state/prd.md`
§4.9 (the 14-script `sendGetText` disposition table) and
`docs/tasks/task-284-npc-ask-text-state/conversion-notes.md` (per-conversion deviations
recorded while authoring the seed content in Tasks 16–20). Script names and dispositions
are taken from those documents; scripts themselves live in an external Cosmic checkout not
present in this repository, per the citations in `conversion-notes.md`.

Legend for scope: S = one handler/branch, M = handler + service logic + events, L = new
subsystem/multi-service.

---

## 1. Scope and method

Cosmic's `cm.sendGetText(prompt[, defaultText[, minLength, maxLength]])` opens a
keyboard-input box on the client and blocks the script until the player submits text (or
cancels). Atlas's NPC conversation engine gained a matching state type, `askText`
(`services/atlas-npc-conversations`, task-284), that sends the equivalent client request,
stores the trimmed answer in conversation context, and branches on an ordered,
first-match-wins comparison table (see `services/atlas-npc-conversations/docs/domain.md`
and `docs/npc_conversation_conversion_spec.md`). This document tracks which Cosmic
`sendGetText` call sites that capability unblocks, and which remain out of reach because
the *subsystem the script drives*, not free-text input itself, is missing.

## 2. Wholly missing (subsystem, not input-handling)

The following `sendGetText` call sites are blocked on a subsystem Atlas does not have,
independent of `askText`'s existence:

### Guild creation via NPC name entry
- **Player experience:** `2010009.js` (Guild Union) prompts for a guild name via
  `sendGetText` and creates the guild on submission.
- **Atlas absence:** guild creation is out of scope for `atlas-npc-conversations`
  entirely — no guild-system operation exists to name and create a guild from a
  conversation context value. Scope: **L**.

### Character rename via NPC
- **Player experience:** `changeName.js` prompts for a new character name via
  `sendGetText`.
- **Atlas absence:** `atlas-character` has no rename capability; there is nothing for a
  conversation operation to call. Scope: **M/L**.

### NLC vending machine ticket quantity
- **Player experience:** `1052014.js` prompts for a ticket quantity via `sendGetText`
  as part of the NLC vending-machine flow.
- **Atlas absence:** the vending-machine subsystem itself is not implemented. Scope: **L**.

### Ariant PQ / expedition participant limit
- **Player experience:** `2101014.js` prompts for a participant limit via `sendGetText`
  as part of registering an Ariant PQ expedition.
- **Atlas absence:** the expedition system is not implemented. Scope: **L**.

## 3. Dead scripts (skip, not blocked)

- `2030013_old.js` (Zakum instance password) — the `_old` suffix marks this a dead
  script in Cosmic; it is not reachable and was not converted. Not a gap.

## 4. Already covered before task-284

- `2090004.js` (craft quantity prompt) uses `sendGetText` for a bounded numeric range,
  which Atlas already modeled as `askNumber` (seeded as `npc-2090004.json`, all
  versions) prior to this work. No `askText` conversion was needed or made.

## 5. `sendGetText` → `askText` conversion status

Of the 14 Cosmic scripts identified that call `sendGetText`, 8 required and received a
free-text `askText` conversion under task-284 (Tasks 16–20); 4 remain blocked on the
subsystems in §2; 1 was already covered by `askNumber` (§4); 1 is dead (§3).

**Converted** (seeded under `deploy/seed/gms/{48,61,72,79,83,84,87,92,95}_1/npc-conversations/npc/`
and `deploy/seed/jms/185_1/...`; `gms/12_1` excluded — no NPC conversation wiring):

| Script | Seed file | NPC / purpose |
|---|---|---|
| `MagatiaPassword.js` | `npc-2111024.json` | Magatia lab door, password vs. quest 3360 progress |
| `2111017.js` | `npc-2111017.json` | Magatia lab pipe, progress 0 → 1 |
| `2111018.js` | `npc-2111018.json` | Magatia lab pipe, progress 2 → 3 (also opens `askPassword` in the same turn) |
| `2111019.js` | `npc-2111019.json` | Magatia lab pipe, progress 1 → 2 |
| `ThiefPassword.js` | `npc-1063011.json` (merged with `PupeteerPassword.js`, see below) | "Open Sesame", gated on quest 3925 completed |
| `PupeteerPassword.js` | `npc-1063011.json` (merged with `ThiefPassword.js`, see below) | "Francis is a genius Puppeteer!", quest-gated warps |
| `2091009.js` | `npc-2091009.json` | Sealed Shrine entrance, "Actions speak louder than words" |
| `1092019.js` | `npc-1092019.json` | Nautilus seagull quiz, answer vs. question index |

**`ThiefPassword.js` / `PupeteerPassword.js` merge.** Both scripts are opened by NPC
template 1063011 (from two different portals — `thief_in1.js` and `enterDollcave.js`
respectively), and Atlas conversations are keyed by NPC id with no portal-side "open a
named script" concept. They were converted into a single `npc-1063011.json`, sharing one
`askText` prompt with two `matches` entries (one per password) and a common
`wrongPassword` fallback. See `conversion-notes.md` Task 18 for the full reasoning and its
implication for any future portal-action work touching those two portals.

**The `1092019.js` nine-Barts arm is omitted, not stubbed.** `1092019.js`'s
`seagullProgress == 1` branch calls `cm.getEventManager("4jaerial").startInstance(…)`,
starting the nine-Barts instance. Atlas has no instance/event-manager capability reachable
from an NPC conversation — this is the same class of blocker as §2, not a prerequisite this
conversion could produce itself. `npc-1092019.json`'s `branchProgress` outcome list
therefore only covers progress `0` and progress `2`; the default outcome silently falls
through to ending the conversation, with no placeholder dialogue standing in for the
omitted case. See `conversion-notes.md` Task 20.

**Blocked** (subsystem gap, tracked in §2 — no seed file):

| Script | Purpose | Blocked on |
|---|---|---|
| `2101014.js` | Ariant PQ participant limit | Expedition system |
| `1052014.js` | NLC vending machine ticket quantity | Vending-machine subsystem |
| `2010009.js` | Guild Union name entry | Guild creation flow |
| `changeName.js` | Character rename | Rename capability in `atlas-character` |

**Skipped** (dead script, tracked in §3): `2030013_old.js`.

**Already covered** (tracked in §4): `2090004.js` (as `askNumber`, unchanged by task-284).
