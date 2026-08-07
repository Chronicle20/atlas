# v92/v95 Movement `types` Configuration — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-04
Tracking issue: #1054
---

## 1. Overview

Atlas decodes MapleStory movement paths generically. `libs/atlas-packet/model/movement.go` reads a
fragment count, then for each fragment reads a one-byte element type and looks that byte up — as an
**array index** — in the socket handler's `options.types` configuration. The looked-up entry yields a
`Name` (e.g. `FALL_DOWN`) and a `Type` (`NORMAL` / `JUMP` / `TELEPORT` / `START_FALL_DOWN` /
`FLYING_BLOCK` / `STAT_CHANGE` / `DEFAULT`), and the `Type` selects which concrete element decoder
runs. The wire layout of a movement fragment is entirely version-specific, so this mapping is
tenant configuration rather than Go code — one `types` array per move handler, per client version.

The `template_gms_92_1.json` and `template_gms_95_1.json` seed templates carry **no `types` option on
any move handler**. When `options["types"]` is absent, `movementPathAttrFromOptions`
(`libs/atlas-packet/model/movement.go:284-312`) logs an error and returns `("NOT_FOUND", "DEFAULT")`.
No branch in `Movement.Decode` matches `DEFAULT`, so every fragment falls through to the bare
`Element` decoder, which reads only `BMoveAction` (byte) + `TElapse` (int16) — three bytes. X, Y, Vx,
Vy, Fh, and the derived stance are never read, and the reader desynchronizes against a fragment that
is 9–15 bytes wide on the wire.

This is the same defect class task-179 (PR #1036) fixed for v48/61/72/79, but those versions could be
repaired by copying each template's own `CharacterMoveHandle.types` into its Monster/Pet/Summon
handlers. v92 and v95 have no such source array — v92's `CharacterMoveHandle` is itself untyped and
v95 has no `CharacterMoveHandle` entry at all — so the arrays must be derived from the clients. The
consequence is broader than task-179's: on v92/v95 tenants it breaks **character** movement decode in
addition to the mob drift/fall-through and frozen-stance symptoms.

## 2. Goals

Primary goals:

- Derive the correct per-index movement `types` array for GMS v92 and GMS v95 from their respective
  client binaries, with recorded evidence.
- Populate `types` on every move handler in `template_gms_92_1.json` and `template_gms_95_1.json`.
- Add the two move handler entries that are missing outright: `CharacterMoveHandle` in
  `template_gms_95_1.json` and `PetMovementHandle` in `template_gms_92_1.json`.
- Close the task-179 leftover: `SummonMoveHandle` (opCode `0x78`) in `template_gms_48_1.json` still
  has no `types`.
- Reconcile the live v92 and v95 tenant socket configurations so existing tenants pick the fix up,
  not only freshly-seeded or restored ones.

Non-goals:

- Any change to the decode/encode logic in `libs/atlas-packet/model/movement.go`. The generic
  lookup is correct; only its configuration input is missing.
- Any change to the v88+ `XOffset`/`YOffset` gate on `NormalElement`
  (`movement.go:131-137`) — that boundary is already correct and covers v92/v95.
- Promoting packet coverage-matrix cells (`docs/packets/audits/STATUS.md`) for the movement ops. This
  task fixes configuration, not codecs; no byte-fixture campaign is in scope.
- Changing `types` on any template other than `gms_92_1`, `gms_95_1`, and the single v48 Summon
  handler named above.
- Live play-testing. The requester will play-test v92 and v95 separately; this task's verification
  bar is derivation evidence plus static invariants (see §10).

## 3. User Stories

- As a player on a v92 or v95 tenant, I want my character's movement to be received correctly by the
  server so that other players see me where I actually am and my position is not corrupted.
- As a player on a v92 or v95 tenant, I want monsters to move and animate normally rather than
  drifting off their spawn foothold and falling through the map on re-entry.
- As a player on a v92 or v95 tenant, I want my pet and my summons to follow me correctly.
- As an operator provisioning a new v92 or v95 tenant, I want the seed template to produce a working
  movement configuration without a manual post-provision patch.
- As an operator running an existing v92 or v95 tenant, I want the fix applied to my live tenant
  configuration without re-provisioning from scratch.

## 4. Functional Requirements

### FR-1 — Derive the v92 and v95 movement type arrays

**FR-1.1** For GMS v92 and GMS v95 independently, derive the ordered array of movement element types
from the client. The array index is the element-type byte read off the wire; the entry describes how
the remainder of that fragment is laid out. Derivation must come from the client binary — the v92 and
v95 IDBs are both packet-named (see project memory `reference_v92_idb_named_from_v95`,
`reference_v95_idb_pdb_backed`) — following the same approach task-179 used to validate v79: read the
`CMovePath::Encode` / movement-fragment dispatch and group indices by which fields each writes.

**FR-1.2** Each derived entry maps to exactly one of the `Type` values the decoder branches on, and
the derivation must justify the choice by the field set the client writes for that index:

| `Type` | Decoder | Fields read after the type byte |
|---|---|---|
| `NORMAL` | `NormalElement` | X, Y, Vx, Vy, Fh, [FhFallStart if `Name == "FALL_DOWN"`], XOffset, YOffset (v88+), BMoveAction, TElapse |
| `TELEPORT` | `TeleportElement` | X, Y, Fh, BMoveAction, TElapse |
| `START_FALL_DOWN` | `StartFallDownElement` | Vx, Vy, FhFallStart, BMoveAction, TElapse |
| `FLYING_BLOCK` | `FlyingBlockElement` | X, Y, Vx, Vy, BMoveAction, TElapse |
| `JUMP` | `JumpElement` | Vx, Vy, BMoveAction, TElapse |
| `STAT_CHANGE` | `StatChangeElement` | BStat only |
| `DEFAULT` | `Element` | BMoveAction, TElapse |

**FR-1.3** The `Name` field is load-bearing in exactly one place: `NormalElement` reads an extra
`FhFallStart` int16 when `Name == "FALL_DOWN"` (`movement.go:126-128`, mirrored in `Encode`).
Derivation must identify which index (if any) is `FALL_DOWN` on each version and name it exactly
`FALL_DOWN`. Indices with no meaningful name use `UNKNOWN`, matching the existing templates.

**FR-1.4** Record the derivation as a committed evidence document under
`docs/tasks/task-191-v92-v95-movement-types/` — per version, per index: the client function and
address consulted, the field set observed, and the resulting `Name`/`Type`. An index whose semantics
cannot be established from the client is recorded as unresolved and escalated (see FR-1.6), never
guessed.

**FR-1.5** Cross-version continuity is a **check**, not a source. The existing templates share an
identical `Name`/`Type` prefix for indices 0–21 across v83, v84, v87, and jms_185, with v84 inserting
`FLYING_BLOCK` at index 23 and lengths growing 23 → 24 → 25 → 33. A v92/v95 derivation that
contradicts this prefix is a signal to re-check the derivation, and a derivation that merely matches
it is not thereby verified. Neighbouring templates must not be copied — that is precisely the
shortcut the tracking issue rules out.

**FR-1.6** If any index cannot be resolved from the client, stop and escalate with the specific
unresolved index and what evidence would settle it. Do not substitute a plausible value.

### FR-2 — Populate `types` on the v92 and v95 move handlers

**FR-2.1** In `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`, set
`options.types` to the derived v92 array on:
- `CharacterMoveHandle` (opCode `0x2E`)
- `MonsterMovementHandle` (opCode `0xDC`)
- `SummonMoveHandle` (opCode `0xC8`)
- `PetMovementHandle` (added per FR-3.2)

**FR-2.2** In `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`, set
`options.types` to the derived v95 array on:
- `CharacterMoveHandle` (added per FR-3.1)
- `MonsterMovementHandle` (opCode `0xE3`)
- `PetMovementHandle` (opCode `0xC7`)
- `SummonMoveHandle` (opCode `0xCF`)

**FR-2.3** Within a single template, all four handlers carry a byte-identical `types` array. This is
the invariant that holds in every already-correct template (v61/72/79/83/84/87/jms) and is asserted
in FR-5.1.

**FR-2.4** `CharacterInventoryMoveHandle` is not a movement handler despite the name — it is item
movement within an inventory and takes no `types`. It must remain untouched in both templates.

### FR-3 — Add the missing handler entries

**FR-3.1** `template_gms_95_1.json` has no `CharacterMoveHandle` entry at all; serverbound character
movement is currently unrouted on v95. Add it with `validator: "LoggedInValidator"`,
`services: ["channel"]`, and the derived `types`, matching the shape of every other template's entry.

The opCode must be re-derived from the v95 client, not taken from the registry at face value.
`docs/packets/registry/gms_v95.yaml:2290-2294` records serverbound `MOVE_PLAYER` at opcode **44
(`0x2C`)** with `fname: CUserLocal::OnKey` plus `fname_alts` — that fname is not a movement handler
name, and `0x2C` sits *below* v92's `0x2E`, breaking the otherwise monotonic upward drift of this
opcode across versions (`0x21` v48 → `0x26` v61 → `0x28` v72 → `0x27` v79 → `0x29` v83/84 → `0x2B`
v87 → `0x2E` v92). Treat the registry row as suspect until the IDB confirms or corrects it; if the
IDB shows the registry is wrong, correct the registry row in the same change.

**FR-3.2** `template_gms_92_1.json` has no `PetMovementHandle` entry. Add it, with the derived opCode,
`validator: "LoggedInValidator"`, `services: ["channel"]`, and the derived `types`. As with FR-3.1,
the opCode is derived from the v92 client.

**FR-3.3** Every added handler entry must have a `validator`. A socket handler entry with a missing
validator is silently dropped at load time with no error — a handler that appears configured but
never fires.

**FR-3.4** Added entries must be inserted at their sorted position in the `handlers` array, not
appended next to a semantically-related entry.
`tools/template-opcode-order-guard.sh` enforces strictly ascending `opCode` order (see
`docs/packets/TEMPLATE_CONVENTIONS.md`).

### FR-4 — Close the v48 Summon leftover

**FR-4.1** `template_gms_48_1.json`'s `SummonMoveHandle` (opCode `0x78`) has no `options.types`, while
that template's `CharacterMoveHandle` (`0x21`), `PetMovementHandle` (`0x71`), and
`MonsterMovementHandle` (`0x81`) all carry an identical 23-entry array. task-179 claimed
Monster/Pet/Summon coverage but this handler was missed. Copy that template's own
`CharacterMoveHandle.types` into it verbatim — unlike v92/v95, v48 has a valid in-template source, so
no derivation is required.

**FR-4.2** No other template is modified under this requirement. `template_gms_12_1.json` has a
9-entry array present and consistent across its three move handlers (it has no pet handler), and is
correct as-is.

### FR-5 — Static invariants

**FR-5.1** For every template in `services/atlas-configurations/seed-data/templates/`, every move
handler present (`CharacterMoveHandle`, `MonsterMovementHandle`, `PetMovementHandle`,
`SummonMoveHandle`) has a non-empty `options.types`, and all such arrays within one template are
byte-identical. This must be checked mechanically over all 11 templates, not by inspection of the
three touched files.

**FR-5.2** Every `types` entry has both a `Name` (string) and a `Type` (string), and every `Type`
value is one of the seven the decoder recognizes (`NORMAL`, `JUMP`, `TELEPORT`, `START_FALL_DOWN`,
`FLYING_BLOCK`, `STAT_CHANGE`, `DEFAULT`). A typo'd `Type` degrades silently to the 3-byte generic
decode for that one index — the same bug in miniature.

**FR-5.3** At most one entry per array is named `FALL_DOWN`, since that name selects the extra
`FhFallStart` read.

**FR-5.4** `tools/template-opcode-order-guard.sh` exits 0.

### FR-6 — Reconcile live tenant configurations

**FR-6.1** Seed templates apply only at tenant provisioning (fresh or restored). Existing v92 and v95
tenants keep their stored socket configuration and will not pick up the fix. Reconcile each live v92
and v95 tenant's socket configuration to the corrected template via the atlas-configurations tenant
API (`GET` the tenant config, swap the affected handler entries, `PATCH` it back — see §5).

**FR-6.2** Reconciliation is per-environment and must be verified after the fact by re-reading each
patched tenant's configuration and confirming the four move handlers are present with a non-empty
`types` — not by trusting the PATCH response.

**FR-6.3** Document the exact reconcile procedure and which environments/tenants were patched in the
task folder, so the operation is repeatable for any environment not covered during this task.

## 5. API Surface

No new or modified endpoints. FR-6 uses the existing atlas-configurations tenant resource
(`services/atlas-configurations/atlas.com/configurations/tenants/resource.go:25-32`):

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/configurations/tenants` | List configuration tenants (paginated) |
| `GET` | `/configurations/tenants/{tenantId}` | Read one tenant's stored configuration |
| `PATCH` | `/configurations/tenants/{tenantId}` | Write the reconciled configuration back |

JSON:API conventions apply: the `PATCH` body is a `{"data": {"type": "tenants", "id": "...",
"attributes": {...}}}` envelope. Requests carry the standard tenant headers.

Error cases relevant to FR-6: a `PATCH` that omits required socket-config structure, or that
introduces a handler without a validator, is accepted at the transport layer but produces a silently
non-functional handler at load time — hence the read-back verification in FR-6.2.

## 6. Data Model

No schema change. The affected structure is the JSONB socket configuration stored per tenant and
mirrored by the seed templates.

A move handler entry:

```json
{
  "opCode": "0xDC",
  "validator": "LoggedInValidator",
  "handler": "MonsterMovementHandle",
  "services": ["channel"],
  "options": {
    "types": [
      { "Name": "NORMAL", "Type": "NORMAL" },
      { "Name": "JUMP",   "Type": "JUMP"   }
    ]
  }
}
```

Constraints:

- `types` is an ordered array; **position is the wire value**. Entries are never reordered, and gaps
  are filled with `{"Name": "UNKNOWN", "Type": "<derived>"}` rather than omitted.
- `Name` is free-form except for the reserved `FALL_DOWN`, which triggers the extra `FhFallStart`
  read/write in `NormalElement`.
- `Type` is one of the seven values enumerated in FR-5.2.
- Array length is version-specific (currently 9 at v12, 23 at v48–v83, 24 at v84, 25 at v87, 33 at
  jms_185). v92 and v95 lengths are an output of FR-1, not an input.

Migration: none for the seed templates (they are files). Live tenants are migrated by the FR-6
reconcile, which is a data operation, not a schema migration.

## 7. Service Impact

| Service / module | Change |
|---|---|
| `services/atlas-configurations` | Seed data only: `template_gms_92_1.json`, `template_gms_95_1.json`, `template_gms_48_1.json`. No Go code change. |
| `libs/atlas-packet` | **No change.** `model/movement.go` is the consumer of this configuration and is already correct. |
| `services/atlas-channel` | **No code change.** It is the runtime beneficiary — `CharacterMoveHandleFunc` (`socket/handler/character_move.go:15`) and the mob/pet/summon equivalents begin receiving fully-decoded movement on v92/v95 once configuration is fixed. Registration in `main.go:816` already exists. |
| `docs/packets/registry/gms_v95.yaml` | Possibly corrected, if FR-3.1's IDB derivation shows the recorded serverbound `MOVE_PLAYER` opcode/fname is wrong. |
| Live v92/v95 tenants | Data reconcile per FR-6. |

## 8. Non-Functional Requirements

- **Correctness over completeness.** An unresolved index escalates (FR-1.6). A wrong `types` entry is
  worse than an absent one in one respect: absence logs a loud error per fragment, while a wrong
  entry decodes silently and corrupts position state.
- **Multi-tenancy.** `types` is per-tenant configuration resolved from the tenant's socket config;
  nothing in this change introduces a version-conditional code path or a shared/global default. The
  existing v88+ `XOffset`/`YOffset` gate remains the only version branch in movement decode.
- **Observability.** The absence of `types` is already loud — `movementPathAttrFromOptions` logs
  `"Code [%d] not configured for use in movement..."` at error level per fragment. After the fix,
  those log lines must be absent for v92/v95 channels; their continued presence is the primary
  regression signal.
- **No regression for other versions.** Templates other than the three named in §2 are byte-unchanged.
  The diff must show changes confined to `template_gms_92_1.json`, `template_gms_95_1.json`, and the
  single `SummonMoveHandle` entry in `template_gms_48_1.json` (plus task docs and, conditionally, the
  v95 registry row).
- **Grounding.** Every derived value cites a client function/address. No value is carried over from
  general MapleStory knowledge or from a neighbouring template.

## 9. Open Questions

1. **v95 `CharacterMoveHandle` opCode.** `docs/packets/registry/gms_v95.yaml` says serverbound
   `MOVE_PLAYER` = 44 (`0x2C`) with `fname: CUserLocal::OnKey`. Both the value (below v92's `0x2E`)
   and the fname look wrong. Resolved during design by IDB derivation; if the registry is wrong it is
   corrected in this change (FR-3.1).
2. **v92 `PetMovementHandle` opCode.** Not currently present in any v92 artifact in-repo; derived from
   the v92 IDB during design (FR-3.2).
3. **Array lengths.** Whether v92 and v95 both land at 25 (matching v87) or grow further is an output
   of FR-1. The v92→v95 span crosses no known GMS movement rework, but that is an expectation to test,
   not an assumption to encode.
4. **Which environments to reconcile.** FR-6 covers live v92/v95 tenants; the specific environment
   list (local / ephemeral / any long-lived) is confirmed at execution time and recorded per FR-6.3.

## 10. Acceptance Criteria

Derivation:

- [ ] A committed evidence document under `docs/tasks/task-191-v92-v95-movement-types/` records, per
      version (v92, v95) and per index, the client function/address consulted, the observed field
      set, and the resulting `Name`/`Type`.
- [ ] No index is recorded as guessed or inferred-from-a-neighbouring-template. Any unresolved index
      was escalated rather than filled in.
- [ ] The v95 `CharacterMoveHandle` opCode and the v92 `PetMovementHandle` opCode are each backed by
      a cited client address.

Templates:

- [ ] `template_gms_92_1.json`: `CharacterMoveHandle` (`0x2E`), `MonsterMovementHandle` (`0xDC`),
      `SummonMoveHandle` (`0xC8`), and a newly added `PetMovementHandle` each carry the derived v92
      `types`.
- [ ] `template_gms_95_1.json`: a newly added `CharacterMoveHandle`, plus `MonsterMovementHandle`
      (`0xE3`), `PetMovementHandle` (`0xC7`), and `SummonMoveHandle` (`0xCF`) each carry the derived
      v95 `types`.
- [ ] `template_gms_48_1.json`: `SummonMoveHandle` (`0x78`) carries that template's existing 23-entry
      array, byte-identical to its `CharacterMoveHandle.types`.
- [ ] Both added handler entries specify `validator: "LoggedInValidator"` and `services: ["channel"]`.
- [ ] `CharacterInventoryMoveHandle` is unchanged in every template.

Invariants (checked mechanically across all 11 templates, output shown):

- [ ] Every move handler present in every template has a non-empty `options.types`.
- [ ] Within each template, all move-handler `types` arrays are byte-identical.
- [ ] Every `Type` value is one of the seven the decoder recognizes.
- [ ] At most one `FALL_DOWN`-named entry per array.
- [ ] `tools/template-opcode-order-guard.sh` exits 0.
- [ ] `tools/lint.sh --check` exits 0.

Scope containment:

- [ ] `git diff --stat` against `main` shows changes only in the three named template files, the
      task docs folder, and — only if FR-3.1's derivation required it —
      `docs/packets/registry/gms_v95.yaml`.
- [ ] No file under `libs/atlas-packet/` or `services/atlas-channel/` is modified.

Live reconcile:

- [ ] Each live v92 and v95 tenant's stored socket configuration has been PATCHed to include the four
      move handlers with the derived `types`.
- [ ] Post-PATCH read-back of each patched tenant confirms the handlers are present and `types` is
      non-empty (FR-6.2), with the actual response quoted.
- [ ] The reconcile procedure and the list of patched environments/tenants are documented in the task
      folder.

Out of band (not gating this task): the requester play-tests v92 and v95 — character movement
decodes, mobs hold their foothold and animate, pets and summons follow.
