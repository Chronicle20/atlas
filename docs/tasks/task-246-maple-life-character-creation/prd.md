# Maple Life — In-Game Character Creation (`Cash/0543`) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

`Cash/0543` ("Maple Life") is a cash-item family that lets a player create a new
character on their account **from inside the game world**, without logging back
out to the character-select screen. The client surface is
`CUICharacterSaleDlg` — a dialog that collects a candidate name, checks it for
duplicates against the server, then submits a full character-creation payload
(look, starting equipment, job) and renders the result.

Atlas currently has no dispatch arm for the family.
`GetCashSlotItemType` already classifies it — `item.ClassificationCharacterCreation`
(543) maps to cash-slot types 57/58 or 65/66 depending on item id and version
(`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1457-1469`)
— but `CharacterCashItemUseHandleFunc` has no branch for any of those four
values, so a player using one of the three 543 items falls through to the
handler's warn fallthrough: the packet is dropped, the dialog never advances,
and the item is neither consumed nor refunded.

The three ops that back the flow are all present in the packet registry and all
uncovered:

| Op | Direction | Client fname | Coverage today |
|---|---|---|---|
| `USE_CASH_ITEM` | serverbound | `CUICharacterSaleDlg::SendCreateNewCharacter` | opcode routed; **this sub-body arm missing** |
| `USE_MAPLELIFE` | serverbound | — | `incomplete`, **gms_v95 only** (opcode 303); `n-a` on v48–v92 and jms_v185 |
| `MAPLELIFE_RESULT` | clientbound | `CUICharacterSaleDlg::OnCheckDuplicatedIDResult` | ❌ on gms_v83/84/87/92/95 |
| `MAPLELIFE_ERROR` | clientbound | `CUICharacterSaleDlg::OnCreateNewCharacterResult` | ❌ on gms_v83/84/87/92/95 |

(Source: `docs/packets/audits/status.json`, `docs/packets/audits/STATUS.md:489-491,590`,
`docs/packets/MapleStory Ops - ClientBound.csv:566-567`,
`docs/packets/MapleStory Ops - ServerBound.csv:174`.)

The character-creation seam already exists and does not need to be built.
`atlas-character-factory` owns character seeding and exposes
`POST characters/seed`; `atlas-login`'s `character/factory` package is a thin
REST client over it
(`services/atlas-login/atlas.com/login/character/factory/{processor,requests}.go`).
`atlas-channel` needs the equivalent client, resolved through
`requests.RootUrlFor(ctx, "CHARACTER_FACTORY")` exactly as
`atlas-channel/character/name_validity_requests.go` already resolves the
character service for name checks.

## 2. Goals

Primary goals:

- A player who uses a `Cash/0543` item in the game world sees
  `CUICharacterSaleDlg` complete end to end: name check answers, character is
  created on their account, and the client renders a real result.
- The three Maple Life packet cells (`USE_MAPLELIFE` serverbound on gms_v95,
  `MAPLELIFE_RESULT` and `MAPLELIFE_ERROR` clientbound on gms_v83/84/87/92/95)
  are implemented, byte-fixture verified, and promoted in the coverage matrix.
- The 543 dispatch arm is version-scoped and classification-first so it cannot
  collide with the cash-slot types it shares numeric values with.
- Character creation from the channel goes through `atlas-character-factory`'s
  existing `POST characters/seed` — no duplicated seeding logic, no new
  character-creation implementation.
- Every terminal outcome (success, duplicate name, invalid look, factory
  failure, slot-limit failure) is either a client-rendered result or a
  deliberate, logged rejection. No silent drop.

Non-goals:

- Cash-shop commodity/pricing authoring for the 543 item ids (that is seed data,
  authored separately).
- The other unimplemented cash-slot types (cube/potential family, jukebox
  trigger, cosmetic coupons, expression items) — each is its own backlog row.
- Non-GMS versions. `USE_MAPLELIFE`, `MAPLELIFE_RESULT`, and `MAPLELIFE_ERROR`
  are all `n-a` on `jms_v185` and on gms_v48/61/72/79; those versions are out of
  scope and must remain untouched.
- Rewriting the ~40 sibling `MajorVersion() >= 95` branches in
  `GetCashSlotItemType`. New code uses `MajorAtLeast`; the existing block is not
  in this task's blast radius (same boundary task-227 drew).
- Any atlas-ui surface. Maple Life is entirely player-facing and in-client.
- Deleting or "selling" characters. Despite the `CharacterSale` fname, the two
  covered ops are *check-duplicate-id* and *create-new-character*; anything
  beyond creation is out of scope unless FR-1.1's derivation proves the same
  sub-body carries it.

## 3. User Stories

- As a player, I want to use a Maple Life item in the field and have the
  character-creation dialog open, so that I can make a new character without
  returning to character select.
- As a player, I want the dialog to tell me immediately whether my candidate
  name is taken, so that I do not waste the item on a name I cannot have.
- As a player, I want my new character to exist on my account and appear in my
  character list the next time I reach character select, so that the item
  delivered what it promised.
- As a player whose creation failed (name taken at submit time, no free
  character slot, invalid look), I want the client to show me a real error, so
  that I know why nothing happened and that my item was not silently eaten.
- As an operator, I want a failed Maple Life creation to leave the item intact
  in the player's inventory, so that a transient factory outage is not a support
  ticket about a lost cash item.

## 4. Functional Requirements

### 4.1 Wire derivation (prerequisite for everything below)

- **FR-1.1** — The exact serverbound sub-body of
  `CUICharacterSaleDlg::SendCreateNewCharacter` MUST be derived by direct
  disassembly of the GMS IDBs (per `docs/packets/IMPLEMENTING_A_PACKET.md`),
  not inferred from `charsb.CreateCharacter` (the login-socket creation packet).
  The two are different code paths and MUST NOT be assumed identical. The
  derivation is recorded in `derivation.md` in this task folder with addresses
  per version.
- **FR-1.2** — The item-id → cash-slot-type split
  (`itemId/1000-5431 > 1` → 57/58 else 65/66) MUST be re-derived from the
  client's own `get_cashslot_item_type` and its dispatcher, on both a pre-95 and
  a v95 IDB. `Cash/0543` contains only three items (5430xxx/5431xxx/5432xxx), so
  the 57/58 branch may be unreachable with shipped data; the derivation decides
  whether an arm is written for it or whether it is documented as unreachable.
  **No arm may be written on a guess.**
- **FR-1.3** — The `USE_MAPLELIFE` (gms_v95, opcode 303) serverbound body MUST be
  derived and its relationship to the `USE_CASH_ITEM` path settled: whether v95
  replaces the cash-item sub-body with a dedicated op, supplements it, or uses
  both in sequence. The answer determines whether v95 has one dispatch entry
  point or two.
- **FR-1.4** — `MAPLELIFE_RESULT` (`OnCheckDuplicatedIDResult`) and
  `MAPLELIFE_ERROR` (`OnCreateNewCharacterResult`) bodies MUST be derived per
  version for gms_v83/84/87/92/95, including the full result/error code
  enumeration each dialog branch renders.

### 4.2 Cash-item dispatch

- **FR-2.1** — `CharacterCashItemUseHandleFunc` gains a branch for the Maple
  Life family that routes the `SendCreateNewCharacter` sub-body to a dedicated
  handler.
- **FR-2.2** — The branch MUST be **classification-first** (`category ==
  item.ClassificationCharacterCreation`), following the precedent set by the
  megaphone and remote-merchant branches
  (`character_cash_item_use.go:786-793`), because the cash-slot type values
  collide:
  - 57/58 is also `ClassificationPetMultiConsumable` (`:1485-1490`);
  - 65 is `CashSlotItemTypeSealTimedV95` (`:970`);
  - 66 is `CashSlotItemTypeViciousHammer`, GMS < 95 (`:991`).
  A bare `it == CashSlotItemType(65)` comparison is **forbidden**.
- **FR-2.3** — Any version-scoped value MUST come from a named helper using the
  `t.IsRegion("GMS") && t.MajorAtLeast(95)` idiom (as
  `nameChangeCashSlotItemType`/`worldTransferCashSlotItemType` do at `:1103-1120`),
  never a raw `MajorVersion() >= 95` in new code and never a hard-coded literal.
- **FR-2.4** — On a version where the family is `n-a` (gms_v48/61/72/79,
  jms_v185), the handler MUST NOT decode a body or produce a wire change. The
  wire behaviour of those versions is unchanged by this task.

### 4.3 Name-duplicate check

- **FR-3.1** — The channel MUST answer the dialog's duplicate-name check with
  `MAPLELIFE_RESULT`. The request arrives via
  `CUICharacterSaleDlg::SendCheckDuplicateIDPacket`, which on gms_v83 is
  `CHECK_CHAR_NAME` (opcode 0x100, `EncodeStr(sCharName)` — see
  `docs/packets/evidence/gms_v83/character.serverbound.CheckName.yaml`). The
  per-version opcode and sender MUST be confirmed for v84/87/92/95 as part of
  FR-1.4; the v83 evidence record is not evidence for the other four.
- **FR-3.2** — The uniqueness scope is `WORLD`, matching character creation, not
  `TENANT` (which is the cash-shop rename's deliberately stricter scope —
  `services/atlas-channel/atlas.com/channel/character/name_validity_requests.go:19-27`).
  Reuse the existing `checkNameValidity` client; do not add a second one.
- **FR-3.3** — The four `NameReason*` values atlas-character returns
  (`length`, `regex`, `duplicate`, `reserved`) MUST each map to a defined
  `MAPLELIFE_RESULT` code via a named table, following the pattern in
  `cash_shop_check_name_change.go:82-104`. An unmapped reason maps to the
  dialog's generic-failure code and is logged at error level — never dropped.
- **FR-3.4** — If the channel socket does not already route the check-name
  opcode, it MUST be routed in every in-scope template. The check is answered by
  the channel, not by atlas-login.

### 4.4 Character creation

- **FR-4.1** — On a valid submit, the channel calls
  `atlas-character-factory`'s `POST characters/seed` through a new
  `atlas-channel` REST client resolved via
  `requests.RootUrlFor(ctx, "CHARACTER_FACTORY")`, mirroring
  `services/atlas-login/atlas.com/login/character/factory/requests.go`.
  **No character-seeding logic is reimplemented in atlas-channel, and
  atlas-login's existing path is not modified.**
- **FR-4.2** — The account id and world id come from the **session**
  (`s.AccountId()`, `s.WorldId()`), never from the client packet, so a crafted
  request cannot create a character on another account or in another world.
- **FR-4.3** — Every look/equipment field submitted by the client (face, hair,
  hair colour, skin colour, gender, top, bottom, shoes, weapon, job index, sub-job
  index) MUST be validated against the same rules the factory applies before the
  call, so a crafted packet cannot seed an out-of-range look. Where
  `atlas-character-factory` already validates (`.../character-factory/validation`),
  a factory rejection MUST be surfaced as a client-rendered `MAPLELIFE_ERROR`,
  not swallowed.
- **FR-4.4** — Creation MUST respect the account's character-slot limit. A
  slot-limit rejection is a distinct, client-rendered error code.
- **FR-4.5** — The name is re-checked at submit time. Passing FR-3.1 earlier in
  the dialog is not sufficient; another player may have taken the name in
  between. A submit-time duplicate is a client-rendered `MAPLELIFE_ERROR`.

### 4.5 Item consumption and failure semantics

- **FR-5.1** — The Maple Life item is consumed **only on a confirmed successful
  creation**. Every failure path (name duplicate, validation rejection, slot
  limit, factory unreachable, factory error) leaves the item in the player's
  cash inventory.
- **FR-5.2** — If the item is consumed and the creation is subsequently found to
  have failed, the item MUST be restored. The ordering that makes this
  unnecessary (create first, consume second) is preferred; if the client's
  protocol forces consume-first, the compensating restore is required.
- **FR-5.3** — Ownership of the item MUST be verified server-side before any
  creation is attempted: the item id in the request must correspond to a
  `Cash/0543` item the session's character actually holds, in the slot claimed.
- **FR-5.4** — No path may terminate without either a clientbound response or a
  logged, explained rejection. The current warn fallthrough is not an acceptable
  terminal state for any in-scope version.

### 4.6 Packet coverage

- **FR-6.1** — `MAPLELIFE_RESULT` and `MAPLELIFE_ERROR` get immutable codecs in
  `libs/atlas-packet` with both `Encode` and `Decode`, version-gated with the
  `MajorAtLeast` idiom where the layouts diverge.
- **FR-6.2** — `USE_MAPLELIFE` gets a serverbound codec for gms_v95 only.
- **FR-6.3** — Each op × version cell in scope is verified per
  `docs/packets/audits/VERIFYING_A_PACKET.md`: byte-fixture test with a
  `packet-audit:verify` marker, pinned evidence record, regenerated matrix. A
  cell that does not promote in `STATUS.md` is a failure, not a prose claim.
- **FR-6.4** — The writers and handlers are routed in every in-scope template
  (`template_gms_{83,84,87,92,95}_1.json`). `template_gms_{48,61,72,79}_1.json`
  and the jms template are **not** modified.
- **FR-6.5** — No wire change may be made to an already-verified cell of any
  other op.

## 5. API Surface

No new public REST surface is introduced. One existing surface is newly
consumed, and one is reused:

**Consumed (new caller): `atlas-character-factory`**

```
POST {CHARACTER_FACTORY_ROOT}characters/seed
```

Request body is the existing `RestModel` shape used by
`services/atlas-login/atlas.com/login/character/factory/requests.go` —
`accountId`, `worldId`, `name`, `gender`, `jobIndex`, `subJobIndex`, `face`,
`hair`, `hairColor`, `skinColor`, `top`, `bottom`, `shoes`, `weapon`, `level`,
`strength`, `dexterity`, `intelligence`, `luck`, `hp`, `mp`, `mapId`. The exact
field set and any JSON:API framing MUST be read from the factory's resource, not
copied from this document.

Errors: a non-2xx or transport failure is a creation failure under FR-5.1 — the
item survives and the client is told.

**Reused: name validity**

```
GET {CHARACTER_ROOT}characters/name-validity?name=&worldId=&scope=WORLD
```

Already implemented in `atlas-channel/character/name_validity_requests.go`;
returns plain JSON (`{valid, reason, detail, reserved}`), not JSON:API.

**Configuration**

`atlas-channel`'s tenant service configuration must resolve
`CHARACTER_FACTORY`. If the key is absent for the channel service, the
service-config seed data and the ingress route set
(`deploy/k8s/base/routes.conf.template.generated`,
`deploy/k8s/base/ns-vars.generated.yaml`) must be extended — verify before
implementation rather than assuming.

**Wire surface**

| Op | Direction | Versions | Note |
|---|---|---|---|
| `USE_CASH_ITEM` (543 sub-body) | serverbound | gms_v83–v95 | new dispatch arm |
| `USE_MAPLELIFE` (303) | serverbound | gms_v95 only | new codec |
| `MAPLELIFE_RESULT` | clientbound | gms_v83 0x15D, v84 0x15D, v87 0x172, v92 0x194, v95 0x19D | new codec + routing |
| `MAPLELIFE_ERROR` | clientbound | gms_v83 0x15E, v84 0x15E, v87 0x173, v92 0x195, v95 0x19E | new codec + routing |
| `CHECK_CHAR_NAME` | serverbound | gms_v83 0x100; others per FR-3.1 | route on channel if not already |

Opcodes above are transcribed from `docs/packets/audits/status.json` and
`STATUS.md:489-491` and MUST be re-read from the registry at implementation
time, never hard-coded from this table.

## 6. Data Model

**No new persisted entities.** Character creation writes through
`atlas-character-factory`, which already owns the character record, its
tenant scoping, and its saga. This task adds no table, no column, and no
migration.

Transient state only:

- The dialog is stateless across packets from the server's perspective; the
  candidate name is not persisted between the duplicate check (FR-3.1) and the
  submit (FR-4.5). That is why FR-4.5 mandates a re-check.
- If the derivation in FR-1.1 shows the client expects the server to hold
  per-session dialog state (e.g. a pending creation token), that state lives in
  the channel session and is discarded on disconnect — it is never persisted.

Multi-tenancy: every call carries the tenant via
`requests.TenantHeaderDecorator`, as the existing channel REST clients do. No
tenant id is ever read from the client packet.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-channel` | New `Cash/0543` dispatch arm in `socket/handler/character_cash_item_use.go`; new Maple Life handler file(s); new `character-factory` REST client package (`characters/seed`); channel-side check-name answer if not already routed; writer registrations in `main.go`. |
| `libs/atlas-packet` | New `MAPLELIFE_RESULT` / `MAPLELIFE_ERROR` clientbound codecs; new `USE_MAPLELIFE` serverbound codec (v95); the `SendCreateNewCharacter` cash sub-body model. Byte-fixture tests per version. |
| `atlas-configurations` | Handler/writer routing in `template_gms_{83,84,87,92,95}_1.json`; `CHARACTER_FACTORY` service-config entry for the channel service if absent. |
| `atlas-character-factory` | **No change expected.** Consumed via its existing `POST characters/seed`. If FR-4.3/FR-4.4 reveal the endpoint cannot express a needed rejection (e.g. slot limit) distinctly, a *narrow* error-detail addition is permitted — not a new creation path. |
| `atlas-login` | **No change.** Its existing factory client is the reference implementation, not a refactor target. |
| `deploy/k8s` | Ingress/ns-var route for `CHARACTER_FACTORY` from the channel, only if not already present. |
| `docs/packets` | Registry entries, evidence records, regenerated `STATUS.md`/`status.json`. |

## 8. Non-Functional Requirements

- **Security** — Account id, world id, and character id come from the session,
  never the packet (FR-4.2). Item ownership and slot are verified server-side
  (FR-5.3). All look/equipment fields are range-validated (FR-4.3). A crafted
  packet must not be able to create a character with an arbitrary look, on
  another account, in another world, or without holding the item.
- **Multi-tenancy** — Every outbound REST call carries the tenant header. Name
  uniqueness is scoped per FR-3.2 and never leaks names across tenants.
- **Observability** — Every terminal outcome logs at an appropriate level with
  the item id, character id, and outcome. A rejection is logged with its reason
  code; a factory failure logs the transport error. Fallthrough-without-log is a
  defect.
- **Correctness over convenience** — Version divergence is expressed with
  `MajorAtLeast` helpers, never inline literals. No hard-coded opcodes; all
  opcodes resolve through the tenant's configured template.
- **Verification** — Flagless `tools/verify.sh` must exit 0 before the branch is
  claimed done. `packet-audit` matrix regeneration must show every in-scope cell
  promoted.
- **No regressions** — Existing cash-item arms, especially the 57/58, 65, and 66
  neighbours (`PetMultiConsumable`, `SealTimed`, `ViciousHammer`), must retain
  their behaviour on every version; a test asserting each of those still routes
  correctly is required alongside the new arm.

## 9. Open Questions

1. **Does v95 use `USE_MAPLELIFE` instead of, or in addition to, the
   `USE_CASH_ITEM` sub-body?** `USE_MAPLELIFE` is `n-a` on v83–v92 but
   `incomplete` with a real opcode (303) on v95, while
   `CUICharacterSaleDlg::SendCreateNewCharacter` is listed as a `USE_CASH_ITEM`
   sender across the board. FR-1.3 resolves this; it determines whether v95 has
   one dispatch entry point or two.
2. **Is the 57/58 branch reachable?** `Cash/0543` ships three items
   (5430xxx/5431xxx/5432xxx), all of which satisfy `itemId/1000-5431 <= 1` under
   Go's signed arithmetic and therefore land on 65/66. Whether the client's own
   comparison is unsigned (which would send 5430xxx to 57/58) is an FR-1.2
   derivation question, and the answer decides whether a second arm exists at
   all.
3. **What distinguishes the three 543 items from each other?** They may differ
   only in expiry/quantity, or may gate different creation options (e.g.
   allowed job set). Unknown until the derivation and the `Item.wz` `spec`
   fields are read.
4. **Is `CHECK_CHAR_NAME` already routed on the channel socket for all in-scope
   versions?** The v83 evidence record exists and `cash_shop_check_name_change.go:35`
   implies the cash-shop socket binds it, but the channel socket's routing must
   be confirmed per template (FR-3.4).
5. **Does the channel's tenant service configuration already expose
   `CHARACTER_FACTORY`?** The ingress route exists cluster-wide; whether the
   channel service's config resolves it must be checked before implementation.
6. **Does the newly created character need to be visible without relogging?**
   The character-select list is served by atlas-login, so a channel-created
   character will naturally appear on next login. Whether the client expects any
   in-session refresh is an FR-1.4 derivation question.
7. **Slot-limit rejection distinctness** — whether `POST characters/seed`
   returns a distinguishable error for "no free slot" versus a generic failure
   (FR-4.4). If not, a narrow factory error-detail addition is in scope per §7.

## 10. Acceptance Criteria

- [ ] `derivation.md` records, with per-version IDA addresses, the
      `SendCreateNewCharacter` sub-body, the `USE_MAPLELIFE` v95 body, the
      `MAPLELIFE_RESULT` and `MAPLELIFE_ERROR` bodies and their full code
      enumerations, and the `get_cashslot_item_type` 543 split — for
      gms_v83/84/87/92/95. No field in shipped code is unsourced.
- [ ] A `Cash/0543` dispatch arm exists in `character_cash_item_use.go`, is
      classification-first, and uses a `MajorAtLeast`-based version helper. No
      bare `CashSlotItemType(65)`/`(66)`/`(57)`/`(58)` comparison appears.
- [ ] Regression tests assert that `PetMultiConsumable` (57/58),
      `SealTimed`/`SealTimedV95` (64/65), and `ViciousHammer`/`ViciousHammerV95`
      (66/67) still route to their existing handlers on both a pre-95 and a v95
      tenant after the new arm is added.
- [ ] Using a Maple Life item on a gms_v83 and a gms_v95 tenant opens the dialog,
      answers the duplicate-name check, and creates a character that is present
      in `atlas-character` afterward.
- [ ] The created character's account id and world id match the session's, and a
      packet claiming a different account/world does not create anything.
- [ ] A duplicate name at submit time, a slot-limit failure, an invalid look, and
      an unreachable `atlas-character-factory` each produce a distinct
      client-rendered `MAPLELIFE_ERROR` and leave the item in inventory.
- [ ] The item is consumed exactly once on success and never on failure;
      a test covers the restore path if consume-first ordering is forced.
- [ ] `MAPLELIFE_RESULT` and `MAPLELIFE_ERROR` show ✅ in `STATUS.md` for
      gms_v83, v84, v87, v92, v95; `USE_MAPLELIFE` shows ✅ for gms_v95. Each has
      a byte-fixture test with a `packet-audit:verify` marker and a pinned
      evidence record.
- [ ] `template_gms_{83,84,87,92,95}_1.json` route the new handlers and writers;
      `git diff` shows **no** change to `template_gms_{48,61,72,79}_1.json` or
      the jms template.
- [ ] No previously-verified cell of any other op changed state.
- [ ] `atlas-login` is unmodified; `atlas-channel` reaches character creation
      only through `atlas-character-factory`'s `characters/seed`.
- [ ] No stubbed handler, placeholder comment, or unimplemented-status response
      is left behind on any in-scope version.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.
