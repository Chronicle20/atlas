# Social-Domain Packet Audit — Design

Version: v1
Status: Proposed
Created: 2026-05-15
PRD: `prd.md`
Prior art:
- `../../task-027-atlas-packet-v95-audit/{design,plan,post-phase-b}.md` (pipeline, analyzer, registry, ack pattern)
- `../../task-028-character-domain-audit/{design,plan,post-phase-b}.md` (scaling the pipeline to a larger domain, multi-version pass, hot-path discipline)

---

## 1. Design Goals

This is the third audit task to use the `tools/packet-audit/` pipeline. Both the pipeline and its scaling pattern are now mature. This design is therefore not architectural — it is **scope, sequencing, and the social-domain-specific risks the prior tasks did not face**.

Constraints driving the decisions below:

- **The pipeline already exists and works.** Do not re-design it. Don't touch the analyzer unless a concrete social-domain finding forces a fix (and even then, prefer `_pending.md` deferral over a tooling excursion — see §4).
- **The PRD's scope numbers are wrong.** PRD §4.1 claims 147 packets (70 cb + 77 sb). Actual src-file count is **76 packets** (37 cb + 39 sb). The PRD's table double-counted by including `_test.go` files. This delta is decisive for sub-task sizing (PRD §10 acceptance covers "all 147 listed packet files"; the correct denominator is 76). §3 below restates the matrix.
- **Sub-op dispatch is the dominant social pattern.** Both `guild/serverbound/` and `party/serverbound/` use a `operation.go` dispatcher that decodes only the leading op byte, with each sub-op as its own file (`operation_invite.go`, `operation_join.go`, …). This is *good news* — the dispatcher pattern explodes the sub-op family into independently-auditable files, so the analyzer addresses them naturally. **Most of the PRD's "sub-op deferral risk" dissolves once you read the directory**; the residual risk is concentrated in chat (`general.go`, `multi.go`, `whisper.go`, `world_message*.go`) where a single file's `Encode` body still branches by leading byte internally.
- **No retroactive scope creep.** Task-027 ballooned 4×; task-028 explicitly disallowed mid-flight pivots. This task inherits that discipline: stop at the social domain, defer everything else.
- **Bare-handler exclusion stays.** Task-027/028 deferred handlers without atlas-packet decoders to `_pending.md`. Same here. Atlas-side service-code descent (atlas-guild, atlas-party business logic) is not part of a wire-shape audit.
- **task-028's analyzer fix is already shipped.** Early-return walker handles `if a { return } else { ... }` correctly. The character pass exercised it on hot packets. Social packets are simpler control-flow than spawn/attack/damage, so no further analyzer work is anticipated.

---

## 2. Architecture Overview

No architectural change. The data flow established by task-027 §2 is unchanged:

```
CSV ─→ template ─→ IDA source ─→ atlas-packet analyzer ─→ diff engine ─→ report writer
                                                ↑
                                                │
                                          TypeRegistry
```

What changes for this task is **what the pieces ingest**:

| Piece                | task-028 input                                          | task-066 input                                                                |
|----------------------|---------------------------------------------------------|-------------------------------------------------------------------------------|
| Atlas source         | `libs/atlas-packet/character/{cb,sb}/`                  | `libs/atlas-packet/{guild,party,buddy,messenger,note,chat}/{cb,sb}/`           |
| IDA exports          | `gms_v95.json` (character append)                       | `gms_v95.json` (social append)                                                |
| IDA exports (cross)  | `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json`      | Same four — **append** social rows                                            |
| Templates            | Character opcodes/sub-ops                               | Social opcodes/sub-ops only                                                   |
| TypeRegistry         | `+AttackInfo`, `+CharacterTemporaryStat::EncodeForeign`, `+Pet`, `+MovementInfo`, `+DamageTakenInfo` | `+GuildMemberEntry`, `+PartyMemberHPBarEntry`, `+BuddyListEntry`, `+MessengerChatEntry` (verify symbol names during execution; PRD §4.5 lists the same set) |
| Analyzer             | Early-return fix landed                                 | No analyzer change expected                                                   |

Read-only against `libs/atlas-packet/`; writes only to:

- `tools/packet-audit/internal/atlaspacket/registry.go` (+ matching `registry_test.go`)
- `libs/atlas-packet/{domain}/{cb,sb}/*.go` (wire-bug fixes only)
- `services/atlas-configurations/seed-data/templates/template_{gms_*,jms_185}_1.json` (opcode/enum fixes)
- `docs/packets/audits/gms_v95/{guild,party,buddy,messenger,note,chat}/` (audit reports)
- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` (IDA evidence appends)
- `docs/packets/audits/gms_v95/_pending.md` (deferral rows)

---

## 3. Actual coverage matrix (PRD §4.1 correction)

Re-enumerated against the worktree (`ls libs/atlas-packet/<d>/<sd>/*.go | grep -v _test.go`):

| Domain    | Clientbound | Serverbound | Total |
|-----------|-------------|-------------|-------|
| guild     | 5           | 19          | 24    |
| party     | 10          | 6           | 16    |
| buddy     | 6           | 3           | 9     |
| messenger | 8           | 5           | 13    |
| note      | 3           | 3           | 6     |
| chat      | 5           | 3           | 8     |
| **Total** | **37**      | **39**      | **76** |

The plan must enumerate the actual filenames so executors don't re-derive from a stale PRD count. §12 lists every file.

### 3.1 PRD §10 acceptance restated

"All 147 listed packet files have audit reports under `docs/packets/audits/gms_v95/`" — operationally, this becomes:

> All 76 social-domain packet src files have a corresponding `.md`/`.json` audit report **OR** a `_pending.md` row citing why a wire-shape audit is not produced.

The denominator is 76. The PRD's 147 came from counting test files; corrected count must propagate to `SUMMARY.md` rows and to the post-phase-b ledger.

---

## 4. The hard part #1: chat sub-type dispatch

Chat is the social-domain analog of task-028's `status_message.go` problem. Specifically:

- `chat/clientbound/general.go` — emits the standard speech bubble. Likely encodes a chat-mode byte (NORMAL=0, WHISPER=1, …) that the v83/v95 client maps differently.
- `chat/clientbound/whisper.go` — whisper-specific shape; sub-mode byte differs from general.
- `chat/clientbound/world_message.go` / `world_message_extra.go` — megaphone variants. Sub-modes here are notorious for drift (item-pop megaphone, avatar megaphone, smega heart, smega skull).
- `chat/clientbound/multi.go` — buddy/party/guild multichat dispatcher.

These are **single files with a leading sub-mode byte literal**. Two operational outcomes:

1. **The sub-mode is hard-coded per file** (e.g. `whisper.go` always writes `0x12` for its sub-mode). In that case the analyzer audits the file normally; the only question is whether the literal matches IDA's case-statement at that version. Treat as a normal audit; verdict drives whether to fix the literal or the template.
2. **The sub-mode is a parameter** (e.g. `general.go` accepts a mode and writes it). The analyzer reports the field as a `Decode1` writer; it cannot statically check the value space. The audit row gets a ⚠️ "manual sub-op review" verdict with a comment naming each known sub-mode and its v95 value. The mapping table moves to `_pending.md` if it crosses ~5 distinct sub-modes.

The PRD §4.7 lists chat sub-ops as deferral candidates. Refinement: **per-file static literals are not deferrals** — they are audit findings. Only the parameterised sub-mode space (general, multi, world_message, world_message_extra) carries the genuine deferral risk. Cap: if a chat file's parameterised sub-mode space exceeds 5 distinct values, defer the enum verification to `_pending.md`; otherwise inline the verdict in the report.

### 4.1 Why this matters more than character did

Task-028 closed the `status_message.go` problem by writing one `_pending.md` row. Chat has four parameterised files. Without the per-file/parameter distinction above, the audit risks four `_pending.md` rows for what is actually one tooling limitation (sub-mode enum modeling). Concentrate the deferral in one row of `_pending.md` keyed by the limitation, not by the file.

---

## 5. The hard part #2: operation-dispatcher families

`guild/serverbound/` and `party/serverbound/` use an explicit dispatcher pattern:

```
operation.go            // reads/writes only the leading op byte
operation_invite.go     // remainder of the invite op
operation_join.go       // remainder of the join op
operation_kick.go       // remainder of the kick op
...
```

The `operation.go` file's `Encode/Decode` body is one `WriteByte` / `ReadByte`. The actual sub-op payloads live as separate files. The analyzer audits each individually — no sub-op modeling needed at the tooling layer.

What the audit must confirm:

- **The op byte's value space matches IDA.** In atlas, the op byte is a field on the struct; the value is supplied by the caller. Audit row for `operation.go` is ⚠️ "tool-limitation: op-byte value supplied by caller; see `_pending.md` row OP-FAMILY-{guild,party}".
- **Each sub-op file's wire shape matches IDA's case-statement body.** Normal audit. ✅ or ❌ per file.
- **The mapping op-byte → sub-handler.** Lives in the IDA dispatcher's switch table. Captured once in `_pending.md` as a static reference table (op value → sub-op file → IDA case offset). The table is documentation, not a tooling gap.

This pattern is the **best-case scenario** for the audit — exploded dispatch is what task-028 wished for. Document it in §1 of the post-phase-b memo as a tooling win the social domain enabled visibility on.

### 5.1 BBS sub-family in guild

`guild/serverbound/bbs_*.go` is the same shape: `bbs_operation.go` is the dispatcher (op byte only), and the 6 individual files (`bbs_create_or_edit_thread`, `bbs_list_threads`, `bbs_display_thread`, `bbs_reply_thread`, `bbs_delete_thread`, `bbs_delete_reply`) carry the sub-op payloads. Audit each individually; one `_pending.md` row for the BBS dispatcher's op value space.

---

## 6. The hard part #3: party HP/MP bar broadcast frequency

`party/clientbound/member_hp.go` broadcasts to every party member on every HP change. This is the social-domain hot path — closer to `attack.go`/`damage.go` than to login packets. A 1-byte wire-shape error here corrupts HP bars for every party member in every fight.

Discipline (mirrors task-028 §8):

- 4-variant byte-output test sweep is mandatory.
- IDA citation in the fix comment names the dispatcher offset.
- No `reflect.*`, no new `interface{}` options. Variant axis from `tenant.Model` if needed.
- If the fix changes layout, add a `BenchmarkEncode` and run before/after locally (no CI gate).

`party/clientbound/update.go` is the second hot file — fires on member join/leave/leader-change. Treat with the same discipline.

---

## 7. TypeRegistry extensions

PRD §4.5 names four expected sub-structs. Refined predictions:

| Sub-struct (predicted symbol)         | Used by                                                                                       | Method      |
|---------------------------------------|-----------------------------------------------------------------------------------------------|-------------|
| `model.GuildMember`                   | `guild/clientbound/info.go`, `bbs.go` (author/replyer entries), `operation.go` (member-list ops) | `Encode`    |
| `model.PartyMemberHPBar` (or similar) | `party/clientbound/member_hp.go`, `update.go`                                                 | `Encode`    |
| `model.BuddyListEntry`                | `buddy/clientbound/list_update.go`, `update.go`                                               | `Encode`    |
| `model.MessengerChatEntry`            | `messenger/clientbound/chat.go`, possibly `update.go`                                         | `Encode`    |

These are **predicted from PRD §4.5 + directory scan**, not confirmed by reading the model package. Phase 1 begins with a 10-minute survey of `libs/atlas-packet/model/` to confirm exact symbol names; if any differ, the plan updates the registry batch accordingly. If a fifth or sixth sub-struct surfaces during Phase 2 (note bodies, guild emblem struct, messenger participant entry), register them as encountered — registration is one entry + one test commit, the marginal cost is small.

### 7.1 Registration discipline (carried from task-028 §4.1)

For each new sub-struct:

1. Add the entry to `registry.go` pointing at the actual Go type with the right method name.
2. Add a `registry_test.go` fixture asserting the analyzed primitive-field list. Match the existing `CharacterStat::Encode` fixture format.
3. Don't pre-emptively register every type in `libs/atlas-packet/model/`. Register only what audited writers actually call.
4. Each registration commits with the first packet that consumes it (commit-message → code-change traceability).

### 7.2 Cross-domain ripple

Same logic as task-028 §4.3: registry additions are additive. Login (task-027) and character (task-028) verdicts must not regress; the Phase 3 closing memo confirms this by re-running both prior audits.

---

## 8. Phasing — concrete artifacts

Phasing follows task-028's pattern, scaled to 76 packets across 6 sub-domains.

### Phase 0 — Survey + registry batch (gate)

One survey commit, then registry registrations.

Artifacts:
- `tools/packet-audit/internal/atlaspacket/registry.go` — additions for the four predicted sub-structs (or however many the model survey confirms).
- `tools/packet-audit/internal/atlaspacket/registry_test.go` — one fixture per registration.
- A short note in `docs/tasks/task-066-social-domain-packet-audit/phase-0-survey.md` (transient; not committed long-term — folded into `post-phase-b.md` at Phase 4) listing confirmed model symbol names and any sub-struct surprises.

Exit: `go test -race ./tools/packet-audit/...` clean; new registry entries each have a passing fixture.

### Phase 1 — v95 audit by sub-domain

Six sub-phases, one per social sub-domain. Each is one tracking unit. Internal commits split clientbound/serverbound and fix-vs-audit naturally.

Suggested ordering (warm-up → hot → tail):

- **1a — note** (6 packets, simplest). Warms up the executor; `note/serverbound/operation.go` confirms the dispatcher pattern matches §5's prediction.
- **1b — buddy** (9 packets). Introduces the BuddyListEntry sub-struct in practice.
- **1c — messenger** (13 packets). Introduces the chat-entry sub-struct.
- **1d — chat** (8 packets, includes §4's sub-mode files). Highest deferral density per-packet.
- **1e — party** (16 packets, includes member_hp/update hot path per §6).
- **1f — guild** (24 packets, largest; includes both operation_* and bbs_* families).

Each sub-phase ends with:

- Audit reports for every packet in that sub-domain.
- `SUMMARY.md` rows added for that sub-domain (verdict + IDA address + notes/citation).
- Real wire-bug fixes committed individually (4-variant test sweep per fix per task-028 §8).
- Template opcode/sub-op fixes committed individually (case-statement value in commit message).
- `_pending.md` rows for bare handlers + sub-op enum spaces.

The audit is "done" for a sub-phase when `SUMMARY.md` shows a verdict or `_pending.md` entry for every src file in that sub-domain — no silent skips.

### Phase 2 — Cross-version pass (v83 → v87 → JMS v185)

One commit batch per version, user-driven IDA swap (PRD §4.6).

For each version, for each social FName:

1. Populate `docs/packets/ida-exports/gms_{v83,v87}.json` or `gms_jms_185.json`.
2. Re-run audit with that version's IDA source + template.
3. For divergences vs v95:
   - Existing `Region/MajorVersion` gate handles it → no code change; export row captures evidence.
   - Atlas gate is wrong → fix the gate, 4-variant sweep, document.
   - Template opcode/enum drift → fix the template, cite case-statement value.
4. Hard cap: 2 nested `if t.Region()` / `if t.MajorVersion()` levels per encoder. 3+ → STOP, `_pending.md`, do not refactor under audit cover (task-028 §7).

Each version is its own commit batch ("phase-2-v83", "phase-2-v87", "phase-2-jms-185") so reviewers can trace per-version drivers.

### Phase 3 — Login + character regression confirm

Mechanical re-run of task-027 and task-028 audits, verifying no verdict regression.

Commands:

```
go run ./tools/packet-audit \
  --csv-clientbound docs/packets/MapleStory\ Ops\ -\ ClientBound.csv \
  --template services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet libs/atlas-packet \
  --ida-source docs/packets/ida-exports/gms_v95.json \
  --output docs/packets/audits/gms_v95
```

Diff `SUMMARY.md` against task-028's closing snapshot. Any verdict flip is in-scope to triage as part of this task. Cap: 2 new ❌s across login + character is the budget before stop-and-split (task-028 §3.5 logic).

### Phase 4 — post-phase-b.md + verification + code review

Mirror the task-028 closing pattern:

- Write `docs/tasks/task-066-social-domain-packet-audit/post-phase-b.md` with the five sections (final state, real wire bugs fixed, template fixes, tooling improvements, remaining work / deferrals).
- Run the four verification commands per PRD §10:
  - `go build ./...`
  - `go vet ./libs/atlas-packet/...`
  - `go test -race ./libs/atlas-packet/...`
  - `go test -race ./tools/packet-audit/...`
- Run `docker build -f services/atlas-configurations/Dockerfile .` only if seed-data structure (not values) changed.
- `gitleaks` scrub: `grep -r '/home/' docs/packets/audits/gms_v95/{guild,party,buddy,messenger,note,chat}/` must be empty.
- Invoke `superpowers:requesting-code-review` (plan-adherence + backend-guidelines).
- Open PR.

---

## 9. Templates — opcode vs sub-op (carried from task-028 §9)

Two surfaces, same as character:

- **Opcode drift** = the `case N` in the dispatch switch changed between versions. Fix is a single integer in `template_*.json`. Trivial.
- **Sub-op (enum) drift** = a writer that routes correctly emits a sub-op byte whose value-to-meaning mapping shifted. Less trivial — requires reading the IDA function's internal switch table.

Social-domain sub-op enum suspects (prioritised):

- `chat/clientbound/general.go`, `multi.go`, `world_message.go`, `world_message_extra.go` — chat-mode bytes (NORMAL/WHISPER/MEGAPHONE/SMEGA/ITEM_POP/AVATAR_MEGAPHONE/HEART/SKULL).
- `guild/clientbound/operation.go` (and clientbound `bbs.go`) — guild operation result codes (invite-result, join-result, kick-result, rank-update-result).
- `party/clientbound/operation_body.go` and `error.go` — party operation results.
- `buddy/clientbound/error.go` — buddy error codes (capacity-full, target-offline, target-blocked, etc.).
- `messenger/clientbound/invite_declined.go` — decline reasons.

Sub-op enum modeling is not in scope to fix at the tooling layer. Document the limitation in one `_pending.md` row keyed by "sub-op enum modeling — social domain" with a sub-list of affected files. Single row, not one per file.

---

## 10. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The PRD-vs-actual scope delta (147 → 76) is missed during plan-task and the executor expects 2× the work | High | Low | This design's §3 is explicit; plan-task should restate it; SUMMARY.md denominator is 76. |
| Sub-struct registry blows up — a guild member-list packet pulls in nested structs (member → avatar → equip slots) | Medium | Medium | Phase 0 survey of `libs/atlas-packet/model/` covers the predicted four. Mid-audit additions are cheap (one entry + one test). Cap: if a single packet's registry chain exceeds 4 unregistered types, pause, triage — likely the packet is doing inventory-style inlining and the chain is the wrong abstraction (task-028 §4.2). |
| Chat sub-mode space is broader than the 4-variant test sweep can express (one whisper variant + one megaphone variant + one smega variant + … explodes test count) | Medium | Medium | §4.1 — concentrate sub-mode enum verification in one `_pending.md` row keyed by limitation, not by file. 4-variant sweep covers wire shape per file; sub-mode value space is a separate dimension acknowledged as deferred. |
| Party member_hp/update hot path sees a wire-bug fix that subtly changes byte layout for v83/v87, breaking existing tenants | Medium | High | §6 discipline + cross-version Phase 2 catches narrow/wide gates. 4-variant byte-output asserts (not just round-trip). |
| Guild operation/bbs dispatcher pattern interpretation is wrong — the dispatcher actually carries payload bytes beyond the op byte (not the case in atlas's current code, but possible if IDA shows otherwise) | Low | Medium | §5's reading is from `operation.go:30-40`: dispatcher writes only one byte. If Phase 1f finds IDA evidence of dispatcher-side payload, that becomes a new finding documented in the report, not a structural design failure. |
| JMS v185 social shape diverges severely (different alliance tiers, different BBS structure per PRD §9 q3) | Medium | High | §8 Phase 2 hard cap on 2 nested guards. If a JMS-only feature requires structural rewrite, `_pending.md` and sibling task. |
| `_pending.md` row inflation — one row per sub-op file + one per bare handler + one per analyzer limitation = noise that obscures real deferrals | Medium | Low | Group deferrals by *cause*, not by *file*. One row per limitation with a sub-list (§9). One row per bare-handler family, not per handler. |
| Cross-version pass surfaces a v83 regression introduced by a v95 fix in a hot social packet (party HP) | Medium | High | §6 fix-discipline + Phase 2 v83 re-run. Plan owner reviews every Phase 2 diff. |
| Login or character regression (Phase 3 finds verdict flip) | Low-medium | Medium | Phase 3 is a gate, not an afterthought. Run before Phase 4. Cap: 2 new ❌s before stop-and-split. |
| Branch hygiene drift across 6 sub-phases × commits per packet | Medium | Low | One commit per packet fix or per template fix group. `superpowers:finishing-a-development-branch` rebase-cleans before PR. |
| gitleaks catches absolute paths in audit reports (task-027/028 both had this) | High | Low | Phase 4 pre-PR check: `grep -r '/home/' docs/packets/audits/gms_v95/<social>/` must be empty. Plan ledger row. |
| Retroactive scope expansion (task-027 ballooning pattern) | Medium | Medium | This design explicitly disallows mid-task pivots into adjacent domains, sub-op enum tooling work, or service-layer fixes beyond minimum ripple. Every "while we're here" gets logged as a future sibling task. |

---

## 11. Out of scope (explicit)

- Bare-handler descent into atlas-channel, atlas-guild, atlas-party, atlas-buddies service code (PRD §3 non-goal).
- Sub-op enum modeling in the audit pipeline (task-028 §9 limitation — acknowledged, not fixed).
- Sub-struct registry coverage for any type the social domain doesn't reference.
- Performance work on hot packets (party HP broadcast). Wire-shape fixes only.
- Analyzer extensions of any kind. Early-return walker is the latest accepted analyzer change (task-028); this task uses it but does not extend it.
- v28 binary integration (task-028 §6 — defer to a sibling task if a binary surfaces).
- Domains outside guild/party/buddy/messenger/note/chat. NPC, monster, drop, field, inventory, miniroom, mini-dungeon, item, quest are all sibling tasks.
- Generic packet-DSL or schema-first encoder rewrite (task-027 §12).
- Stock-Nexon clientVariant axis additions beyond what task-027 already shipped.
- Service-layer logic changes (atlas-guild member list logic, atlas-party member tracking) — fixes are wire-shape only; if a fix requires upstream caller changes, document and split.
- Guild emblem rendering, party UI behavior, buddy-list presence routing — all atlas service concerns, not wire-shape.

---

## 12. File enumeration (canonical input list)

Recorded so plan-task does not re-derive. Excludes `_test.go`.

**guild/clientbound (5):** `bbs.go`, `emblem_changed_foreign.go`, `info.go`, `name_changed_foreign.go`, `operation.go`

**guild/serverbound (19):** `bbs_create_or_edit_thread.go`, `bbs_delete_reply.go`, `bbs_delete_thread.go`, `bbs_display_thread.go`, `bbs_list_threads.go`, `bbs_operation.go`, `bbs_reply_thread.go`, `invite_reject.go`, `operation.go`, `operation_agreement_response.go`, `operation_invite.go`, `operation_join.go`, `operation_kick.go`, `operation_request_create.go`, `operation_set_emblem.go`, `operation_set_member_title.go`, `operation_set_notice.go`, `operation_set_title_names.go`, `operation_withdraw.go`

**party/clientbound (10):** `change_leader.go`, `created.go`, `disband.go`, `error.go`, `invite.go`, `join.go`, `left.go`, `member_hp.go`, `operation_body.go`, `update.go`

**party/serverbound (6):** `invite_reject.go`, `operation.go`, `operation_change_leader.go`, `operation_expel.go`, `operation_invite.go`, `operation_join.go`

**buddy/clientbound (6):** `capacity_update.go`, `channel_change.go`, `error.go`, `invite.go`, `list_update.go`, `update.go`

**buddy/serverbound (3):** (verify in plan; counted 3 src files)

**messenger/clientbound (8):** `add.go`, `chat.go`, `invite_declined.go`, `invite_sent.go`, `join.go`, `remove.go`, `request_invite.go`, `update.go`

**messenger/serverbound (5):** (verify in plan)

**note/clientbound (3):** `display.go`, `operation.go`, `operation_body.go`

**note/serverbound (3):** `operation.go`, `operation_discard.go`, `operation_send.go`

**chat/clientbound (5):** `general.go`, `multi.go`, `whisper.go`, `world_message.go`, `world_message_extra.go`

**chat/serverbound (3):** (verify in plan)

Plan-task is expected to fill the `(verify in plan)` rows during context.md generation.

---

## 13. Reference points in the existing tree

- `tools/packet-audit/internal/atlaspacket/registry.go` — registration site for the four predicted sub-structs.
- `tools/packet-audit/internal/atlaspacket/registry_test.go` — fixture format (match `CharacterStat::Encode` style).
- `tools/packet-audit/internal/atlaspacket/analyzer.go` — early-return walker shipped in task-028; no changes expected.
- `libs/atlas-packet/guild/serverbound/operation.go` — minimal dispatcher pattern (op byte only); confirms §5 reading.
- `libs/atlas-packet/guild/serverbound/operation_invite.go` — typical sub-op file; one field, one Decode.
- `libs/atlas-packet/party/clientbound/member_hp.go` — social hot path (§6).
- `libs/atlas-packet/chat/clientbound/general.go` — chat sub-mode dispatch (§4).
- `libs/atlas-packet/model/` — survey target for Phase 0 (exact sub-struct symbol names).
- `docs/packets/audits/gms_v95/SUMMARY.md` — top-level audit index; receives 76 social rows.
- `docs/packets/audits/gms_v95/_pending.md` — deferral ledger; sub-op enum + bare-handler rows.
- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` — IDA evidence appends.
- `services/atlas-configurations/seed-data/templates/template_gms_{12,28,83,87,92,95}_1.json` + `template_jms_185_1.json` — opcode/enum fixes.
- `docs/tasks/task-028-character-domain-audit/post-phase-b.md` — template for this task's closing memo.

---

## 14. What plan-task should do next

Split this design into explicit, sequenced, small tasks. Suggested structure:

- **2–3 tasks for Phase 0** (model survey + registry batch + tests).
- **One task per sub-phase 1a–1f** (note, buddy, messenger, chat, party, guild). 6 sub-tasks. Each task ends with verdict-triaged commit set; fix commits inside a sub-task are individual.
- **One task per version for Phase 2** (3 sub-tasks: v83, v87, JMS-185).
- **One task for Phase 3** (login + character regression confirm).
- **One task for Phase 4** (post-phase-b.md + verification + code review + PR).

Total target: **13–14 plan tasks**. More than that is the plan re-deriving the audit per-packet; fewer hides scope.

Plan-task should specifically resolve:

- The exact model symbol names confirmed by Phase 0 survey (update §7's table).
- The exact filenames for `buddy/serverbound`, `messenger/serverbound`, `chat/serverbound` (the three rows marked "verify in plan" in §12).
- The IDA function-address mapping for `guild/serverbound/operation.go` and `party/serverbound/operation.go` dispatcher op-byte switch tables (drives the `_pending.md` reference table per §5).
- Whether sub-phase ordering 1a → 1f stands or whether the user prefers hot-path-first (`1e party` before `1d chat`). Default: keep warm-up-to-hot to build executor familiarity before touching member_hp.
- The bundling convention for fix commits within a sub-phase (suggestion: one commit per packet fix; one template-fix commit groups all opcode shifts for that sub-phase).
- The Phase 2 commit naming convention (`phase-2-v83`, `phase-2-v87`, `phase-2-jms-185`) so review history reads cleanly across versions.
- The post-phase-b ledger row format (carry task-028's table headers verbatim).
