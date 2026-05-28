# Context — Social-Domain Packet Audit (task-066)

Companion to `plan.md`. Captures the key files, decisions, and dependencies the implementer needs without re-reading the full PRD/design.

---

## Source artifacts (read these first)

- `docs/tasks/task-066-social-domain-packet-audit/prd.md` — full requirements, §4 functional reqs, §10 acceptance criteria. **Note:** PRD §4.1 packet count is wrong (147 includes test files); the corrected denominator is **76**, established in design §3.
- `docs/tasks/task-066-social-domain-packet-audit/design.md` — phasing (§8), chat sub-mode dispatch (§4), operation-dispatcher families (§5), party hot-path discipline (§6), TypeRegistry batch (§7), risks (§10), out-of-scope (§11), file enumeration (§12), reference points (§13).
- `docs/tasks/task-027-atlas-packet-v95-audit/post-phase-b.md` — template for `post-phase-b.md` in Phase 4.
- `docs/tasks/task-027-atlas-packet-v95-audit/plan.md` — style anchor for plan structure (this plan inherits its conventions).
- (If task-028 has merged to main when this work begins:) `docs/tasks/task-028-character-domain-audit/{plan,post-phase-b}.md` — second-iteration anchor; the per-fix recipe and bucket-commit cadence in this plan derive from its Phase 2.
- `docs/packets/audits/gms_v95/SUMMARY.md` — current audit verdict matrix (login + possibly character if merged). Phase 1 appends 76 social rows; Phase 3 asserts no regression on the existing rows.
- `docs/packets/MapleStory Ops - ClientBound.csv`, `... - ServerBound.csv` — FName ↔ opcode mappings; consumed by the audit CLI.

---

## Key existing code references

| Concern | File | Notes |
|---|---|---|
| Audit CLI entry-point | `tools/packet-audit/` | `go run ./tools/packet-audit ...`. No code change needed in this task. |
| TypeRegistry — pass-2 (method discovery) | `tools/packet-audit/internal/atlaspacket/registry.go:81-117` | Auto-discovers receiver `Encode`/`Write`/`EncodeForeign` methods (`EncodeForeign` added by task-028). Phase 0 adds fixtures only — no walker change. |
| TypeRegistry — existing fixtures | `tools/packet-audit/internal/atlaspacket/registry_test.go` | Tasks 4–6 from task-028 added character-domain fixtures (`AttackInfo`, `Pet`, `DamageTakenInfo`, `Movement`+elements, `CharacterTemporaryStat::EncodeForeign`). Task 1 here adds `GuildMember`, `Buddy`, `Avatar` in the same shape. |
| Analyzer walker — `*ast.IfStmt` arm | `tools/packet-audit/internal/atlaspacket/analyzer.go:191-243` | Early-return suffix-taint shipped by task-028. No further analyzer change expected (design §1, §2). |
| Sub-struct: GuildMember | `libs/atlas-packet/model/guild_member.go:21` | `Encode` method on value receiver. Used by `guild/clientbound/info.go`, `operation.go`. |
| Sub-struct: Buddy | `libs/atlas-packet/model/buddy.go:19` | `Encode` method on value receiver. Used by `buddy/clientbound/list_update.go`, `update.go`. |
| Sub-struct: Avatar | `libs/atlas-packet/model/avatar.go` | Already in registry (used by character/spawn). Used by `messenger/clientbound/add.go`, `update.go`. |
| Package-level helper: `WritePartyData` | `libs/atlas-packet/party/member_data.go:19` | `func WritePartyData(w *response.Writer, members []PartyMember, leaderId uint32)` — flattens 6 fixed-size column slices. **Not a receiver method**, so the registry's pass-2 cannot model it. Task 1 documents this in `_pending.md`. Affected packets: `party/clientbound/{update,join,left}.go`. |
| Hot-path encoder: party HP broadcast | `libs/atlas-packet/party/clientbound/member_hp.go` | 3 `WriteInt`s (characterId, hp, maxHp). Hot path per design §6 — 4-variant byte-output sweep mandatory for any fix. |
| Hot-path encoder: party update | `libs/atlas-packet/party/clientbound/update.go` | Calls `party.WritePartyData(w, m.members, m.leaderId)`. Same hot-path discipline. |
| Operation dispatcher (op-byte only): guild | `libs/atlas-packet/guild/serverbound/operation.go:30-40` | `Encode` body is one `WriteByte(m.op)`. Confirms design §5 reading. OP-FAMILY-guild row in `_pending.md`. |
| Operation dispatcher (op-byte only): bbs | `libs/atlas-packet/guild/serverbound/bbs_operation.go` | Same shape. OP-FAMILY-bbs row. |
| Operation dispatcher (op-byte only): party | `libs/atlas-packet/party/serverbound/operation.go` | Same shape. OP-FAMILY-party row. |
| Operation dispatcher (op-byte only): messenger | `libs/atlas-packet/messenger/serverbound/operation.go` | Same shape. OP-FAMILY-messenger row. |
| Operation dispatcher (op-byte only): note | `libs/atlas-packet/note/serverbound/operation.go` | Same shape. OP-FAMILY-note row. |
| Chat sub-mode dispatch (parameterised) | `libs/atlas-packet/chat/clientbound/general.go` | First `WriteByte` in `Encode` body — value depends on hardcoded literal vs `m.<field>`. Verify at audit time per Task 5 Step 3 dichotomy. |
| Chat hard-coded sub-mode | `libs/atlas-packet/chat/clientbound/whisper.go` | `WhisperSendResult.Encode` writes `m.mode` (parameterised) — sub-mode goes to consolidated `_pending.md` row. Verify against IDA dispatcher case-statement. |
| Round-trip harness | `libs/atlas-packet/test/roundtrip.go:12-24` | Asserts `reader.Available() == 0` after decode. Every serverbound fix test uses this. |
| Tenant variants for tests | `libs/atlas-packet/test/context.go` | Source of `pt.GMSv83()` / `pt.GMSv95()` / `pt.JMSv185()` (and possibly `pt.GMSv87()` if task-028 added it). 4-variant sweep iterates `pt.Variants`. |
| Existing roundtrip test pattern | `libs/atlas-packet/login/clientbound/auth_success_test.go:9-37` | `for _, v := range pt.Variants { t.Run(v.Name, ...) }` — copy for new tests. |
| Templates ship | `services/atlas-configurations/seed-data/templates/template_gms_{12,83,87,92,95}_1.json`, `template_jms_185_1.json` | Six templates total. Social-domain template fixes touch the relevant subset. |
| IDA exports (current) | `docs/packets/ida-exports/gms_v83.json`, `gms_v95.json`, `_pending.md` | `gms_v87.json` and `gms_jms_185.json` do NOT exist yet — Tasks 9 and 10 create them. |
| `_pending.md` ledger | `docs/packets/ida-exports/_pending.md` | Existing login section + possibly character section. Task 1 creates "Sub-op enum / sub-struct deferrals — social domain (task-066)" heading; later tasks append OP-FAMILY rows and bare-handler rows under their own headings. |

---

## Critical decisions locked in design

- **Packet count is 76, not 147** (design §3). PRD §4.1's 147 came from counting `_test.go` files. Phase 1 sub-task exit gates use 76 as the denominator.
- **No analyzer surgery in this task** (design §1, §2). The early-return walker (task-028) and `EncodeForeign` registry support (task-028) are inherited as-is. Phase 0 only adds registry *fixtures* — no walker code changes.
- **TypeRegistry registers only what social packets reference** (design §7). Predicted batch is 3 types (`GuildMember`, `Buddy`, `Avatar`). Mid-audit additions are cheap (one fixture). Do NOT pre-register the rest of `libs/atlas-packet/model/`.
- **`party.WritePartyData` is a known tool-limitation** documented in `_pending.md` rather than fixed (Task 1 Step 4). Affected packets get ⚠️ verdicts with IDA cross-check footers.
- **Chat sub-mode dispatch — single deferral row, not per-file** (design §4.1). One bullet under "Sub-op enum / sub-struct deferrals — social domain" listing all parameterised chat files. Hard-coded sub-mode literals audit normally.
- **Operation dispatchers are good news** (design §5). `guild/serverbound/operation.go` etc. only emit the op byte; sub-op payload files audit individually. One OP-FAMILY-* row per family in `_pending.md`.
- **Bare handlers** (anything without an atlas-packet decoder) stay deferred to `_pending.md` per PRD §3 non-goal; no descent into atlas-channel/atlas-guild/atlas-party/atlas-buddies service code.
- **JMS hard cap** — no atlas-packet encoder may grow beyond 2 nested region/version guards from this task's fixes. 3+ → log to `_pending.md` and stop (design §8 Phase 2; carries from task-028 §7).
- **No `reflect`, no new `interface{}` params, no benchmarks** (design §6, inherited from task-028 §8). Encoder fixes change which `WriteX` runs under which guard; encoder structure stays as-is.
- **Tracking sub-tasks ≠ single PRs.** Each Phase 1/2 sub-task contains independent fix commits (one fix = one commit); the sub-task itself is the tracking unit, not a single commit/PR. Bucket commits land *after* per-fix commits so the SUMMARY snapshot reflects post-fix state.
- **Sub-phase ordering — warm-up to hot-path** (design §8 Phase 1, with explicit confirmation in design §14). Default order: note → buddy → messenger → chat → party → guild. Can be overridden to hot-path-first (party before chat) at execution time if the implementer prefers, but plan defaults to warm-up.

---

## Decisions deferred to execution time

- **Per-version audit-output directory layout.** v83/v87/JMS-185 reports go under `docs/packets/audits/gms_v83/`, `gms_v87/`, `jms_v185/` respectively — that's the assumption baked into Tasks 8–10 paths. The tool will create the directory; if reviewers prefer a flat layout, restructure on the first cross-version run, not later (consistency matters more than aesthetic).
- **Bucket commit cadence** — Tasks 2–7 may produce 1–N fix commits before the bucket commit. Maintain ordering: fixes first, audit-report bucket commit last.
- **Whether to widen or narrow a multi-version gate** — case-by-case, driven by IDA evidence. Default is "widen only when v83/v87/JMS-185 IDA confirms shared behaviour"; default is "narrow only when one of those versions diverges and atlas's current code is the wrong shape for it".
- **`_pending.md` section headings** — the file currently has a "## Still pending — login domain" heading and (if task-028 merged) a "## Still pending — character domain" heading. Task 1 creates "Sub-op enum / sub-struct deferrals — social domain (task-066)". Tasks 2–7 add OP-FAMILY-* rows under that same heading or under a sibling "## Bare handlers — social domain (task-066)" heading on first hit.
- **cb/sb filename collision in audit reports.** Several sub-domains have an `operation.go` in both clientbound and serverbound. If the audit tool produces colliding `operation.md` filenames, hand-rename to `operation_cb.md` / `operation_sb.md` consistently within the affected sub-task and re-stage. Confirm at first occurrence (likely Task 2 — note has `operation.go` in both directions).
- **Login/character regression budget surfaced by Phase 3** — Task 11 caps at 2 new ❌s. 3+ means STOP and split a sibling task; do not absorb into this plan.
- **Whether to add OP-FAMILY-bbs as a separate row from OP-FAMILY-guild** — yes, default; but if at audit time the `bbs_operation.go` op-byte space turns out to be a sub-namespace of the guild op-byte space (i.e. one dispatcher with bbs ops nested), collapse to one row.

---

## Workflow notes for the implementer

1. **Verify cwd before every commit.** Worktree is `.worktrees/task-066-social-domain-packet-audit/`; branch is `task-066-social-domain-packet-audit`. `git rev-parse --show-toplevel` must end with `/.worktrees/task-066-social-domain-packet-audit`, `git branch --show-current` must be `task-066-social-domain-packet-audit`. The auto-memory pin "Never commit directly to main" is load-bearing here — Phase 1 produces a lot of small commits and the discipline matters.
2. **Run `go test -race ./tools/packet-audit/...` before every registry commit.** This is the canary for fixture correctness.
3. **Run `go test -race ./libs/atlas-packet/...` before every encoder commit.** A v95 fix that regresses v83 is the worst case; the 4-variant sweep catches it.
4. **Run `go vet ./libs/atlas-packet/...` and `go vet ./tools/packet-audit/...` before each commit.**
5. **Do NOT run `docker build` unless a service Dockerfile or `go.mod` changed.** Template seed-data JSON edits are visible to atlas-configurations at runtime and require no rebuild. Task 12 Step 3 confirms.
6. **The audit CLI never auto-mutates `.go` files.** Every encoder fix is a hand edit anchored to a freshly-generated audit report. Do not pipeline-rewrite encoders.
7. **IDA MCP needs the right binary loaded.** Phase 0 / Phase 1 / Phase 3 assume v95 IDA is open. Phase 2 requires the user to swap binaries explicitly (the plan calls this out at the top of each Phase 2 task). If `mcp__ida-pro__get_metadata` returns the wrong binary, ask the user before continuing.
8. **gitleaks risk.** task-027 had a follow-up to scrub `/home/<user>/` paths out of audit reports. Task 12 Step 4 is the mandatory pre-PR scrub. Audit reports are generated by the tool; the tool sometimes embeds absolute paths from `--atlas-packet` — keep the flag relative (`libs/atlas-packet`) when invoking from worktree root and the issue stays absent.

---

## External dependencies / open questions resolved

- **IDA-MCP availability** — user-driven. Each Phase 2 task starts with a `get_metadata` check.
- **CSV path** — `docs/packets/MapleStory Ops - ClientBound.csv` and `... - ServerBound.csv` already exist on the branch (shipped with task-027). No new CSVs.
- **Sub-struct ownership** — design §7 + Task 1 Step 1 survey. `model.GuildMember` and `model.Buddy` exist; `Avatar` already used by character; `WritePartyData` is package-level and deferred.
- **Operation dispatcher pattern** — design §5; confirmed by `libs/atlas-packet/guild/serverbound/operation.go:30-40`.
- **Chat sub-mode space** — design §4 + Task 5 dichotomy. Hard-coded literals audit; parameterised modes defer.
- **Sub-op enum drift in pipeline** — design §9 acknowledges the limitation; documented, not fixed.
- **Bare handler descent** — design §1 + PRD §3 non-goal; defer to `_pending.md`.
- **JMS opcode-space divergence** — design §8 Phase 2 + design §10 risk row. In-scope vs out-of-scope per Task 10 Step 4.

---

## Plan-execution checklist (one-glance summary)

- 12 tasks. Phase 0 = Task 1 (sub-struct registry fixtures + WritePartyData deferral). Phase 1 = Tasks 2–7 (six sub-domains, warm-up to hot-path). Phase 2 = Tasks 8–10 (cross-version v83/v87/JMS-185). Phase 3 = Task 11 (login + character regression confirm). Phase 4 = Task 12 (closeout).
- Sub-phase ordering 1a → 1f (note → buddy → messenger → chat → party → guild) deliberately escalates analyzer pressure: note exercises the dispatcher pattern; buddy + messenger introduce sub-struct registrations; chat concentrates sub-op deferrals; party introduces hot-path discipline; guild stresses the largest dispatcher families.
- Every Phase 1/2 sub-task is a tracking unit. Plan checkboxes mark sub-tasks complete only when every bucket packet has a verdict row + every ❌ has a fix commit or `_pending.md` deferral.
- Phase 3 (Task 11) is a gate, not an afterthought. Run before Phase 4. Cap: 2 new ❌s across login + character before stop-and-split.
- Final exit gate: Task 12 Steps 2 + 4. `go test -race ./...` + `go vet ./...` clean; gitleaks scrub empty.
