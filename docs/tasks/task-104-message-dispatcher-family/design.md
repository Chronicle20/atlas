# CWvsContext::OnMessage Dispatcher Family Migration — Design

Task: task-104-message-dispatcher-family
Status: Draft for review
Created: 2026-06-18
Companion to: `prd.md` (approved), `docs/packets/DISPATCHER_FAMILY.md` (governing pattern),
task-103 guild migration (exemplar).

---

## 1. Problem & Current State (grounded)

`CWvsContext::OnMessage` (the `MESSAGE` opcode, `CWvsContext::OnPacket` case 0x26) is a
mode-prefix dispatcher: the client reads a leading mode byte and routes to a per-mode
sub-handler, two of which (`OnDropPickUpMessage`, `OnQuestRecordMessage`) fan out further on
an inner discriminator. Atlas implements the family with **24 `StatusMessage*` structs** in
`libs/atlas-packet/character/clientbound/status_message.go`.

Unlike the guild family (task-103), the **codec + config-driven body layer is already built**:

- All 24 structs already take `mode byte` via constructor (no `mode: 0x` literals).
- `libs/atlas-packet/character/status_message_body.go` already exposes **23 body funcs**, each
  resolving the outer mode through `WithResolvedCode("operations", "<KEY>", func(mode byte) …)`
  with a **fixed string-literal key** and **no caller-supplied selector** — already
  footgun-free (INV-2/INV-3 clean). 23 funcs cover 24 structs because
  `CharacterStatusMessageOperationDropLossItemBody(itemId, quantity)` constructs *either*
  `StatusMessageDropLossStackableItem` *or* `StatusMessageDropLossUnStackableItem` by
  inventory type (`status_message_body.go:58-66`). Every struct is constructed → no orphan
  (INV-5 already satisfied).
- The ~8 atlas-channel consumers (drop, compartment, quest, asset, character,
  system_message, conversation_reward_notice, + main) already call these body funcs.

What is **not** done is exactly the dispatcher-audit half:

| Gap | Evidence | Requirement |
|---|---|---|
| `run.go` maps the whole family to ONE candidate (`StatusMessageDropPickUpInventoryFull`) | `tools/packet-audit/cmd/run.go:392-398` | FR-5 |
| Matrix has a single op-row (`SHOW_STATUS_INFO`) graded ✅ on v83/v84/v87/v95 from mode 0 alone; ❌ on jms | `docs/packets/audits/STATUS.md:61` | FR-7 |
| Only **4** `// packet-audit:verify` markers exist across all 24 arms | `status_message_test.go` (grep) | FR-7.1 |
| jms export models the 16 per-mode delegates 0 deep; GMS exports carry a flat `OnMessage` | PRD §1, FR-6 | FR-6 |
| **The seeded per-version mode table is WRONG for v83** (see §2 — the triggering bug) | `docs/packets/dispatchers/character_status_message.yaml`; IDA | FR-2/FR-9 |

So this task is **not** a codec/config refactor like guild — it is (a) fix the v83 mode-table
correctness bug, (b) enroll the family in the dispatcher-audit tooling so per-arm verification
*proves* the fix and prevents recurrence, and (c) resolve the jms ❌ via export decomposition.

**Governing rule (`DISPATCHER_FAMILY.md`):** `matrix ✅` means codec byte-correct for the one
arm that was sampled — nothing more. The single-representative ✅ on v83 is precisely the
"passes on one byte" false pass: it sampled mode 0 (drop/pickup), which is identical across
all versions, so the upper-arm shift was invisible.

## 2. The triggering bug (IDA-grounded, this session)

The committed `character_status_message.yaml` asserts the 15-mode table (keys 0–14) is
"version-STABLE … byte-identical in v83 and v95 (IDA-verified)." **That claim is false.**
Decompiling `CWvsContext::OnMessage` in every IDB this session:

- **v83** (`MapleStory_dump.exe`, port 13342, `0xA209D4`): **14 cases (0–0xD)**, and there is
  **no `OnIncSPMessage` arm**. Order: 0 DropPickUp · 1 QuestRecord · 2 CashItemExpire ·
  3 IncEXP · 4 Inc**POP**(fame) · 5 Inc**Money**(meso) · 6 IncGP · 7 GiveBuff ·
  8 GeneralItemExpire · 9 SystemMessage · 0xA QuestRecordEx · 0xB ItemProtectExpire ·
  0xC ItemExpireReplace · 0xD SkillExpire.
- **v95** (port 13340, `0xA06C90`): **15 cases (0–0xE)**, `OnIncSPMessage` at case **4**,
  pushing fame→5, meso→6, …, SkillExpire→14. Named symbols — authoritative.
- **v87** (port 13341, `0xAB8076`): 15 cases, identical layout to v95.
- **v84** (port 13337, `0xA6BDD9`): 15 cases (0–0xE), so SP is present (v84 ≠ v83 for this
  packet). Sub-handlers are `sub_*` (unnamed in the export); the *count* and SP-present
  boundary are proven, but the exact per-arm semantic order must be confirmed by decompiling
  each `sub_` during enumeration (FR-1.2 — do not fold from v83/v95 by assumption).
- **jms** (port 13339, `0xB078F3`): **16 cases (0–0xF)** — the 15 v95 arms plus
  `case 0xF → sub_B0931C`, an arm with **no** corresponding Atlas struct (PRD open-Q2).

The resulting **IDA-grounded per-version outer-mode table** is the table in `prd.md`'s
companion message and §4 below. Consequence of the bug on a live v83 tenant today: Atlas
emits FAME at mode 5, but the v83 client's mode 5 is `OnIncMoneyMessage` → a fame gain renders
as meso; GP renders as buff; every arm from fame upward is off by one; and SKILL_EXPIRE (sent
at 14) exceeds v83's `default` boundary (0xD) and is silently dropped. `INCREASE_SKILL_POINT`
(sent at 4) hits v83's fame handler.

This bug is **why the audit exists** and is the design's center of gravity.

## 3. Goal / Definition of Done

Migrate `CWvsContext::OnMessage` to the canonical discrete-per-mode audit shape, **correct the
v83 (and any v84) mode table**, drive every supported arm to ✅ across all five versions
(version-absent → ⬜, never ❌, never fabricated ✅), resolve the jms ❌, and keep the family
`dispatcher-lint`-clean with **no** baseline entry. All gates in §9 exit 0.

Per-version arm verdicts at done:
- **v83:** 14 arms (DROP_PICK_UP through SKILL_EXPIRE at the v83 modes) ✅;
  `INCREASE_SKILL_POINT` ⬜ (genuinely absent).
- **v84/v87/v95:** all 15 arms ✅.
- **jms:** all 15 v95 arms ✅; the 16th arm (`sub_B0931C`, mode 0xF) — see D7 (stop-and-ask).

## 4. The grounded per-version mode table (source of truth)

This table replaces the incorrect "version-stable" table in
`character_status_message.yaml`. v83 ≠ v84+ (SP insertion at case 4):

| key (outer mode) | gms_v83 | gms_v84 | gms_v87 | gms_v95 | jms_v185 |
|---|---|---|---|---|---|
| DROP_PICK_UP | 0 | 0 | 0 | 0 | 0 |
| QUEST_RECORD | 1 | 1 | 1 | 1 | 1 |
| CASH_ITEM_EXPIRE | 2 | 2 | 2 | 2 | 2 |
| INCREASE_EXPERIENCE | 3 | 3 | 3 | 3 | 3 |
| INCREASE_SKILL_POINT | — (absent) | 4 | 4 | 4 | 4 |
| INCREASE_FAME | 4 | 5 | 5 | 5 | 5 |
| INCREASE_MESO | 5 | 6 | 6 | 6 | 6 |
| INCREASE_GUILD_POINT | 6 | 7 | 7 | 7 | 7 |
| GIVE_BUFF | 7 | 8 | 8 | 8 | 8 |
| GENERAL_ITEM_EXPIRE | 8 | 9 | 9 | 9 | 9 |
| SYSTEM_MESSAGE | 9 | 10 | 10 | 10 | 10 |
| QUEST_RECORD_EX | 10 | 11 | 11 | 11 | 11 |
| ITEM_PROTECT_EXPIRE | 11 | 12 | 12 | 12 | 12 |
| ITEM_EXPIRE_REPLACE | 12 | 13 | 13 | 13 | 13 |
| SKILL_EXPIRE | 13 | 14 | 14 | 14 | 14 |

v84 values are pending the per-arm `sub_` confirmation (D8); the SP-at-4 layout is the
strong prior but is verified, not assumed, in enumeration.

## 5. Architecture — two-level dispatcher

The family has **two** discriminator levels; only the **outer** is a config-resolved mode.

```
outer mode byte (0..0xE, config-resolved from the operations table)  ← the dispatcher mode
   └─ mode 0  DROP_PICK_UP → inner int8 discriminator (-2/-1/-3/0/1/2 + sign)  ← structural
   └─ mode 1  QUEST_RECORD → inner byte after questId (0/1/2)                  ← structural
   └─ modes 2..N  one arm each (no inner fan-out)
```

Data flow per arm (already in place; unchanged):

```
consumer → CharacterStatusMessage…Body(arm data)                 (status_message_body.go)
   → WithResolvedCode("operations", "<FIXED_KEY>",               resolves OUTER mode from
        func(mode byte) → clientbound.New<Arm>(mode, data))      the tenant operations table
   → struct.Encode writes outer mode + inner discriminator + body (status_message.go)
```

The **24 `#`-entries** in `run.go` are a finer granularity than the **15 operations keys**:
the 8 drop arms all resolve key `DROP_PICK_UP` (one outer mode), the 3 quest-record arms all
resolve `QUEST_RECORD`. Each struct still gets exactly one `#`-entry → INV-1 holds. The
`#`-suffix is the audit candidate selector, not the operations key.

## 6. Key design decisions (alternatives + tradeoffs)

### D1 — The inner discriminator is a structural constant, NOT a second config-resolved mode

Resolves PRD open-Q4. The drop/pickup inner int8 (`-2` item-unavailable, `-1` inventory-full,
`-3` game-file-damaged, `0` stackable, `1` meso, `2` unstackable) and the quest-record inner
byte (`0` forfeit, `1` update, `2` complete) are values the client's `OnDropPickUpMessage` /
`OnQuestRecordMessage` switch compares against **literal constants** that map to string-pool
ids — they do not shift per version and are not table-driven on the client.

- **Chosen:** keep the inner discriminator baked into each arm's `Encode` (as it is today,
  e.g. `w.WriteInt8(-2)`), cited to the inner sub-handler decompile per version. Only the
  outer mode is config-resolved. This does **not** trip INV-2: INV-2 bans `mode:\s*0x` literals
  in a struct *constructor* and `func(_ byte)` in body files — an inner body write is neither.
- **Rejected — config-resolve the inner discriminator too** (a `drop.yaml` / sub-operations
  table, mirroring guild's BBS peer-dispatcher in task-103): guild's BBS is a genuinely
  version-variable *opcode* sub-dispatcher; these inner bytes are semantic string selectors
  that are stable across versions. A second operations table would add ceremony with no
  correctness benefit and no precedent for non-opcode inner enums. Tradeoff: the uniformity
  rule in `DISPATCHER_FAMILY.md` is about the *mode byte*; we honor it for the outer mode and
  document the inner bytes as structural, cited to IDA.
- **Verification still per-inner-arm:** each of the 8 drop and 3 quest arms gets its own
  discrete struct, `#`-entry, and byte fixture, so the inner discriminator is byte-verified
  per arm regardless — no "passes on the outer byte" false pass.

### D2 — `run.go`: 24 `#`-entries replacing the single representative; no phantom root

Replace `case "CWvsContext::OnMessage": return {StatusMessageDropPickUpInventoryFull}`
(run.go:392-398) with one `case "CWvsContext::OnMessage#<Arm>":` per supported arm → its
discrete struct (`dir: clientbound`). The bare root returns **no representative** (mirror the
`OnFieldEffect` / guild `OnGuildResult` root handling confirmed during execution). 24 `#`-entries
(DropLoss is two: `#DropLossStackableItem`, `#DropLossUnStackableItem`). FR-5.2/FR-5.3:
remove the single-rep phantom; comments carry current per-version verdicts, no stale
"deferred to _pending.md".

### D3 — Body layer already config-driven & footgun-free → no `dispatcher-lint-baseline` entry

`OnMessage` is **not** in `dispatcher-lint-baseline.yaml` today (only party/guild/buddy). It is
invisible to the linter only because it is a single-representative, not a `#`-family. Once D2
adds the `#`-entries the linter scans it; because the body layer is already INV-2/3/5-clean,
it should pass **without** a baseline entry (FR-10.1). Keep the fixed string-literal keys
("DROP_PICK_UP", …) — INV-3 explicitly permits a string literal as the key; promoting to typed
consts is optional polish, not required, and the keys must continue to match the yaml exactly.

### D4 — Fix the v83 mode table; make the operations table genuinely per-version

Correct `character_status_message.yaml` to §4's table: drop the false "version-stable" banner;
set `INCREASE_SKILL_POINT` absent on v83 (omit/null, not a fabricated byte) and FAME→SKILL_EXPIRE
to the v83 values; keep v84/v87/v95/jms at the SP-at-4 layout. Reconcile every seed template's
`CharacterStatusMessage` operations map to this table (`gms_83`, `gms_84`, `gms_87`, `gms_95`,
`jms`). The gate is `packet-audit operations --check`. The version gate where code branches is
`>= 84` for SP presence (FR-1.4: not `>83` loosely — the SP boundary is exactly v83→v84).

### D5 — `INCREASE_SKILL_POINT` is ⬜ on v83, never ❌, never faked

v83 has no SP arm. Its row is **version-absent (⬜)** for v83 and ✅ for v84/v87/v95/jms. The
`StatusMessageIncreaseSkillPoint` struct + body func stay (used by v84+); on v83 the body func
must **not** emit (resolving `INCREASE_SKILL_POINT` against a v83 operations table that lacks the
key returns no code — confirm the resolve path no-ops/errors safely rather than sending mode 99).
Confirm v83 consumer behavior during execution: an SP gain on v83 has no client packet, matching
the client.

### D6 — jms export decomposition (FR-6) is the bulk of the new tooling work

The jms export must gain the 16 per-mode delegate sub-functions (the 15 named `On*Message`
handlers + `sub_B0931C`) with **real** addresses from the jms IDB (decompiled this session:
`OnDropPickUpMessage 0xB07A01` … `OnSkillExpireMessage 0xB088A4`, `sub_B0931C 0xB0931C`), plus
any inner fan-out sub-functions. The GMS exports (v83/v87/v95) gain the same per-mode delegate
structure (Decode1 + guarded delegate refs) so the audit decomposes the family on GMS too.
Surgical JSON splice, never overwrite (per packet-audit reference notes); no `address: "0x0"` /
`ida=0x0` placeholders; evidence records pin a real `decompile_sha256`.

### D7 — jms mode 0xF `sub_B0931C` is a stop-and-ask, not an invention

`sub_B0931C` (jms case 0xF) has no Atlas struct. Per FR-1.3 and PRD open-Q2: resolve its real
name/arm from the jms IDB during enumeration. If it maps to an existing Atlas arm, wire it; if
it is a jms-only arm with no Atlas equivalent, **stop and ask** — do not invent a struct, fake a
fname, or grade it. jms reaching ✅ for the 15 shared arms does not depend on 0xF; 0xF is tracked
explicitly (⬜/escalated) so the jms row is honest either way.

### D8 — v84 verified directly, not folded (FR-1.2)

v84 has a real IDB (port 13337). Its 15-case count is proven; its per-arm semantic order
(`sub_*`) is confirmed by decompiling each delegate (read order = SP body `short jobId + byte
amount` at case 4, etc.), not assumed equal to v95. task-103 found a real v84≠v83 guild
divergence; this packet already shows a v84≠v83 divergence (the SP insertion), so v84 gets its
own per-arm fixtures, not a v83 or v95 copy.

### D9 — Keep `character_status_message.yaml` as the source of truth (naming)

PRD acceptance §10 names `docs/packets/dispatchers/message.yaml`. The canonical file already
exists as `character_status_message.yaml`, named for the writer and consistent with its siblings
(`cash_shop_operation.yaml`, `npc_shop_operation.yaml`, …). **Decision:** correct and extend the
existing `character_status_message.yaml` rather than create a redundant `message.yaml`; treat the
PRD's filename as descriptive, not prescriptive. (Flagged so a reviewer can object; the
sibling-consistent name is the better choice.)

### D10 — Call sites are verified, not rewritten (FR-8)

Because the body-func layer already exists and the consumers already call it, FR-8 is mostly a
**verification** step: confirm each of the ~8 consumers routes through the per-mode body funcs
(no leftover direct struct construction with a literal mode), and confirm every touched
serverbound/clientbound handler/writer seed entry keeps a non-empty validator/opcode (FR-8.2,
the silently-dropped trap). Any v83 SP emission path is made a no-op (D5). No business-logic
change.

## 7. Component-by-component scope

- **`libs/atlas-packet/character/clientbound/status_message.go`** — 24 structs; verify each
  `Encode` writes outer mode + inner discriminator + full body; add decompile citations
  (function + per-version address) in struct comments where missing. No structural rewrite
  expected (codecs already correct); changes are citation + any read-order fix surfaced by a
  fixture.
- **`libs/atlas-packet/character/status_message_body.go`** — verify fixed-key bodies; ensure the
  v83-absent `INCREASE_SKILL_POINT` path no-ops safely (D5).
- **`libs/atlas-packet/character/clientbound/status_message_test.go`** — per-arm byte fixtures
  with `// packet-audit:verify` markers + IDA citations for **every supported version** (24
  arms × applicable versions; v83 omits SP).
- **`tools/packet-audit/cmd/run.go`** — 24 `#`-entries; single representative + phantom removed
  (D2).
- **`docs/packets/dispatchers/character_status_message.yaml`** — corrected per-version table
  (§4), false "version-stable" banner replaced with the IDA-grounded per-version note (D4/D9).
- **`docs/packets/ida-exports/*`** — jms 16-delegate splice + GMS delegate structure + inner
  fan-out (D6); real addresses.
- **`docs/packets/registry/*`, `docs/packets/evidence/*`** — per-arm evidence records, real
  `decompile_sha256`.
- **`docs/packets/audits/STATUS.md` + `status.json`** — regenerated; the single `SHOW_STATUS_INFO`
  row becomes the aggregate of 24 arms (worst-of, FIELD_EFFECT model); jms ❌→✅ for shared arms.
- **`services/atlas-configurations`** — `CharacterStatusMessage` operations maps corrected across
  all five seed templates (D4).
- **`services/atlas-channel`** — call-site verification only (D10).
- **Live config runbook** — post-deploy PATCH of live tenants' operations tables (esp. the v83
  correction) + channel restart; executed after merge/deploy (FR-9, PRD §7/§9).

## 8. Execution phasing (for the plan phase)

1. **Enumerate & ground** — decompile all five `OnMessage` switches (done this session) +
   the two inner fan-out handlers (`OnDropPickUpMessage`, `OnQuestRecordMessage`) per version;
   confirm v84 per-arm order (D8); resolve jms `sub_B0931C` (D7). Author the corrected
   `character_status_message.yaml`.
2. **run.go rewire** — 24 `#`-entries; remove single rep + phantom (D2).
3. **Export decomposition** — jms 16-delegate splice + GMS delegate structure + inner fan-out
   (D6); real addresses.
4. **Fixtures + matrix** — per-arm byte fixtures all applicable versions with markers + IDA
   citations; regenerate STATUS; v83 SP ⬜, jms shared arms ✅.
5. **Seed-template + operations reconcile** — corrected per-version operations maps; `operations
   --check` green (D4).
6. **Lint + de-baseline confirm** — `dispatcher-lint` scans OnMessage clean with no baseline
   entry (D3).
7. **Call-site + validator verification** — D10; v83 SP no-op (D5).
8. **Gates + build** — four packet-audit checks exit 0; `go build/vet/test -race` in
   `libs/atlas-packet`, `tools/packet-audit`, `services/atlas-channel`; `docker buildx bake
   atlas-channel`; `tools/redis-key-guard.sh` (no Redis expected).
9. **Code review** — modular reviewer agents before PR.
10. **Live config** — runbook patch + restart, executed and verified in channel logs (no
    "unhandled message op" for the family; a representative message renders per version, esp.
    v83 fame/meso/skill-expire now correct).

## 9. Acceptance gates (from PRD §10, must all hold)

Discrete-per-mode audit shape (24 `#`-entries, one struct each, no phantom) · footguns absent
(already: zero `mode: 0x` literal, zero `func(_ byte)`, no caller selector) · **v83 mode table
corrected** and per-version operations reconciled · every supported arm ✅ across all five
versions, version-absent → ⬜ (v83 SP), jms shared arms ❌→✅, jms 0xF resolved or escalated ·
jms/GMS exports decomposed with real addresses, no `0x0` placeholders · per-arm fixtures +
markers + IDA citations · `dispatcher-lint` / `matrix --check` / `fname-doc --check` /
`operations --check` exit 0, no baseline entry · `go build/vet/test -race` clean +
`docker buildx bake atlas-channel` · code review before PR · CI green on PR HEAD · live-config
runbook authored (+ executed post-deploy).

## 10. Open questions (resolved during execution, not now)

1. **jms `sub_B0931C` (mode 0xF) identity** — resolve from the jms IDB; stop-and-ask if it has
   no Atlas arm (D7).
2. **v84 per-arm semantic order** — confirm by decompiling each v84 `sub_` (D8); SP-at-4 is the
   strong prior.
3. **v83 inner discriminators** — confirm `OnDropPickUpMessage` / `OnQuestRecordMessage` inner
   read order on v83 matches the structs' baked constants (expected stable; verified per arm).
4. **v83 SP emission no-op** — confirm the resolve path safely no-ops when
   `INCREASE_SKILL_POINT` is absent from the v83 operations table (D5), rather than emitting
   mode 99.
5. **Live tenant/version set** for the post-deploy patch — determined at execution via
   k8s/Grafana MCP; the v83 correction is the priority.

## 11. Out of scope (from PRD §2 non-goals)

No business-logic change to the ~8 consumers (call-site re-route/verify only). No new arms
beyond the existing 24 structs (jms 0xF is escalated, not invented). No DB/REST change. No new
tenant version / LB socket ports. Migrating party/buddy off the baseline is a separate cycle.
