# Context — World-Domain Packet Audit (task-068)

Companion to `plan.md`. Captures the key files, decisions, and dependencies the implementer needs without re-reading the full PRD/design.

---

## Source artifacts (read these first)

- `docs/tasks/task-068-world-domain-packet-audit/prd.md` — full requirements, §4.1 coverage matrix, §4.5 conversation per-type policy, §4.6 set_field 3-deep exception, §4.8 cross-version cadence, §10 acceptance criteria.
- `docs/tasks/task-068-world-domain-packet-audit/design.md` — analyzer reuse policy (§3 — DO NOT extend), registry triage (§4), set_field nesting policy (§5), conversation per-type audit shape (§6), NPC dispatcher offset (§7), field-effect sub-op pattern (§8), cross-version phasing (§9), flat report layout (§10), phasing (§11), PRD-question resolutions (§12), risks (§13), out of scope (§14), reference points (§15).
- `docs/tasks/task-027-atlas-packet-v95-audit/post-phase-b.md` — first-domain closeout precedent.
- `docs/tasks/task-028-character-domain-audit/post-phase-b.md` — second-domain closeout precedent and template for `post-phase-b.md` Phase 4.
- `docs/tasks/task-028-character-domain-audit/plan.md` — style anchor for plan structure and per-bucket workflow.
- `docs/packets/audits/gms_v95/SUMMARY.md` — current login + character verdict matrix. Snapshot at Phase 0; world rows append flat alongside.
- `docs/packets/MapleStory Ops - ClientBound.csv`, `... - ServerBound.csv` — FName ↔ opcode mappings; consumed by the audit CLI.
- `docs/packets/ida-exports/gms_v95.json` — login + character FName entries; world entries append during Phase 2.
- `docs/packets/ida-exports/_pending.md` — bare-handler / sub-op / tool-limitation ledger; world sections created in Phase 2h sweep.

---

## Key existing code references

| Concern | File | Notes |
|---|---|---|
| Audit CLI | `tools/packet-audit/main.go`, `tools/packet-audit/cmd/` | `go run ./tools/packet-audit ...` is the canonical invocation. Flag list in `README.md`. |
| TypeRegistry (read-only for this task) | `tools/packet-audit/internal/atlaspacket/registry.go` | Pass-2 method discovery handles `Encode`, `Write`, and (from task-028) `EncodeForeign`. Per design §3, this task does NOT edit `registry.go` — only the `_test.go`. |
| TypeRegistry fixtures | `tools/packet-audit/internal/atlaspacket/registry_test.go` | Phase 1 Tasks 2–3 append `TestRegistryRegistersNpcShopItem`, `TestRegistryRegistersNpcConversation`. Mirror existing fixture shape (`TestRegistryFindsCharacterListEntry` etc.). |
| Analyzer (DO NOT TOUCH) | `tools/packet-audit/internal/atlaspacket/analyzer.go` | Per design §3 mandate. If a panic/cycle surfaces, STOP and spin a sibling task. |
| Round-trip test harness | `libs/atlas-packet/test/roundtrip.go` | Asserts `reader.Available() == 0` after decode. Every serverbound fix test uses this. |
| Variant fixtures | `libs/atlas-packet/test/context.go` | Source of `pt.GMSv83()` / `pt.GMSv87()` / `pt.GMSv95()` / `pt.JMSv185()` used by the 4-variant sweep. |
| Existing 4-variant pattern | `libs/atlas-packet/login/clientbound/auth_success_test.go:9-37` | `for _, v := range pt.Variants { t.Run(v.Name, ...) }` — copy for new tests. |
| Templates | `services/atlas-configurations/seed-data/templates/template_gms_{12,83,87,92,95}_1.json`, `template_jms_185_1.json` | At plan-task time, none of these contain world-domain `writer` entries (verified: `grep -o '"writer": "[^"]*"' template_gms_95_1.json` shows only login + character entries). Phase 2 sub-tasks append world writers as they audit each sub-domain. |
| World-domain Operation strings | various | `SetField` (used by both `SetField` and `WarpToMap` — see `warp_to_map.go:16-40`), `NPCConversation` (`conversation.go:13`), plus per-file `Operation()` constants. Read each file to determine the canonical string. |
| Set-field envelope | `libs/atlas-packet/field/clientbound/set_field.go` | 4 sibling guards at depth 1 (encode + decode mirrors). Allowed 3-deep nesting per PRD §4.6. Envelope-only audit per design §3 last paragraph — `CharacterData.Encode` is already audited under character domain. |
| Warp-to-map | `libs/atlas-packet/field/clientbound/warp_to_map.go` | Uses `Operation() = SetFieldWriter` (shared with `SetField`). Cross-version-likely. |
| Transport | `libs/atlas-packet/field/clientbound/transport.go` | Recently merged from `instance-based-transports` (commits `456ec8717`, `05bdfd7b0`). In-place audit. |
| Field effect (good form) | `libs/atlas-packet/field/clientbound/effect.go` | 5+ separate Go encoder structs (`EffectSummon`, `EffectTremble`, `EffectString`, `EffectBossHp`, `EffectRewardRullet`). Each gets its own audit row. |
| Field effect (bad form) | `libs/atlas-packet/field/clientbound/effect_weather.go` | One struct, mode byte in constructor. Tool-limitation verdict + manual sub-op annotation per design §8. |
| Clock (bad form) | `libs/atlas-packet/field/clientbound/clock.go` | Two modes (wall-clock vs countdown). Same treatment as `effect_weather.go`. |
| Conversation (monolithic) | `libs/atlas-packet/npc/clientbound/conversation.go` | 360 lines; 8 dialog-type sub-encoders behind a leading text-type byte. Per PRD §4.5 + design §6, audited as ONE report with per-type sub-sections; NOT refactored. |
| Shop list (loop-count) | `libs/atlas-packet/npc/clientbound/shop_list.go` | Commodity-array loop; flattens incorrectly through the analyzer. Manual bounds verification against IDA `CUserShopDlg::SendShopList`. |
| Existing audit reports (flat) | `docs/packets/audits/gms_v95/*.{md,json}` | 57 files at plan-task time (login + character). World rows interleave flat per design §10. |

---

## Critical decisions locked in design

- **Analyzer reuse, no changes.** Design §3 — this task does NOT touch `tools/packet-audit/internal/atlaspacket/analyzer.go`. If a panic or new cycle surfaces, STOP and split a sibling task. Sub-op dispatch, loop-count modeling, and deep sub-struct descent remain pipeline limitations — defer per the established pattern (acknowledge in audit report, optionally `_pending.md`).
- **TypeRegistry scope.** Only register types the world domain *actually calls*. High-confidence Phase 1 batch: `NpcShopItem`, `NpcConversation`. Medium-confidence (`SetField` map-header, `WarpToMap` coord block, Clock per-mode) deferred to Phase 2 on first analyzer-surfaced need.
- **Cross-domain ripple guardrail.** Every registry addition triggers a re-run of the v95 audit; login + character SUMMARY rows must stay byte-identical. Phase 0 (Task 1) sets the baseline; every Phase 1 / Phase 2 step that registers a type re-diffs against `/tmp/summary-pre-task068.md`. Any drift → STOP and roll back the registry entry.
- **set_field.go nesting.** 3-deep nested guards allowed for `set_field.go` ONLY (PRD §4.6). All other encoders capped at 2-deep. 4+ → STOP, defer to `_pending.md`. The current `set_field.go` is at 4 sibling guards at depth 1 (NOT nested) per design §5; the 3-deep allowance is a future ceiling, not a current overrun.
- **set_field envelope-only audit.** The embedded `m.characterData.Encode(...)` call descends into `CharacterData`, which is already audited under the character domain (task-028). Audit only the bytes around the `CharacterData.Encode` call; annotate `🔍 envelope-only` for the inner shape.
- **Conversation per-type audit shape.** One report file (`NPCConversation.md`) with 8 per-type sub-sections; SUMMARY row = worst-of-8 verdict; per-section fix commits + per-section 4-variant tests. NOT a refactor of `conversation.go`.
- **NPC dispatcher offset.** `CUserPool::OnPacket`-equivalent prepends `characterId` (4 bytes). Atlas decoders must either consistently include `characterId` at offset 0 OR consistently treat it as already-consumed. Audit's job is to document the boundary in each report header + verify the post-prepend payload against IDA per-handler decoder.
- **Cross-version phasing.** v95 complete → v83 → v87 → JMS v185 (PRD §4.8). One IDA database per Phase 3 task; user-driven swap. Batches IDA context-switching to three discrete windows.
- **Audit-report ack footer.** ALWAYS the LAST line written. To re-run a single report: `git checkout HEAD -- <report.md>` first.
- **Sub-op enum drift, loop-count, deep sub-struct descent** — known pipeline limitations from task-028. Documented in `_pending.md`, NOT fixed in this task.
- **Bare handlers** — deferred to `_pending.md` (`## Still pending — world domain`). No descent into `services/atlas-channel/` or `services/atlas-npcs/`.
- **Tracking sub-tasks ≠ single PR commits.** Phase 2 / Phase 3 sub-tasks produce 1–N fix commits + 1 bucket commit each. The sub-task is the tracking unit.
- **No `reflect`, no `interface{}` params, no benchmarks** in encoder fixes.

---

## Decisions deferred to execution time

- **Exact world-domain writer names + opcodes for the v95 template.** Read each file's `Operation()` to get the writer string; opcodes from IDA dispatcher case-statements. Land template additions in the same commit as the first audited packet that needs them.
- **Per-version audit-output directory layout.** v83/v87/JMS-185 reports go under `docs/packets/audits/gms_v83/`, `gms_v87/`, `jms_v185/` respectively (assumption baked into Tasks 12–14 paths, mirroring task-028 precedent). If task-028's Phase 3 chose a flat layout, match that.
- **Whether `gms_v87.json` exists.** Task-028 plan claimed it would create one in its Phase 3; verify on Task 13 start. If missing, Task 13 originates it.
- **Whether `NpcShopItem` is a named type or inline.** Task 2 step 1 inspects; the fixture shape depends on the finding.
- **Whether `conversation.go` ships per-type structs or one struct with constructor methods.** Task 3 step 1 inspects; the fixture shape depends.
- **Conversation text-type byte resolvability.** Plan-task assumption: ≥ 6 of 8 branches are statically resolvable (literal mode byte per `if branch`). Up to 2 may require `_pending.md` deferral. More than 2 → file's SUMMARY row goes ⚠️ overall (PRD §4.5).
- **set_field JMS sub-major-version split (Phase 3 Task 14).** If v185 IDA reveals an early-JMS vs 185+ divergence inside `Region() == "JMS"`, that's a structural rewrite candidate. Hard cap from §5: 3-deep total. 4+ → defer to `_pending.md` as a follow-up task.
- **Effect.go sub-struct count.** Design §8 says "5+ separate Go encoder structs". Verify by reading `effect.go` in Task 7; each gets its own audit row.
- **TypeRegistry medium-confidence entries.** `SetField` map-header, `WarpToMap` coord block, Clock per-mode — register on first analyzer-surfaced need during Phase 2.

---

## Workflow notes for the implementer

1. **Verify cwd before every commit.** Worktree is `.worktrees/task-068-world-domain-packet-audit/`; branch is `task-068-world-domain-packet-audit`. `git rev-parse --show-toplevel` must end with `/.worktrees/task-068-world-domain-packet-audit`, `git branch --show-current` must be `task-068-world-domain-packet-audit`. The auto-memory pin "Never commit directly to main" is load-bearing — Phase 2 produces many small commits.
2. **Snapshot the prior-domain SUMMARY at Phase 0.** Task 1 step 1 writes `/tmp/summary-pre-task068.md`. This is the regression-diff target for the entire task. If you wipe `/tmp` mid-task, re-snapshot from the merge-base: `git show <task-028-tip>:docs/packets/audits/gms_v95/SUMMARY.md > /tmp/summary-pre-task068.md`.
3. **Run `go test -race ./tools/packet-audit/...` before every registry commit.** The fixtures are the canary; if they break, the registry walker drifted upstream.
4. **Run `go test -race ./libs/atlas-packet/...` before every encoder commit.** A v95 fix that regresses v83 is the worst case; the 4-variant sweep catches it. Encoder fixes that affect downstream consumers (e.g. `set_field` constructor signature changes ripple into `atlas-channel` warp/spawn handlers) need a `go build ./services/atlas-channel/...` sanity check before commit.
5. **Run `go vet ./libs/atlas-packet/...` and `go vet ./tools/packet-audit/...` before each commit.**
6. **Do NOT run `docker build` unless a service Dockerfile or `go.mod` changed.** Template seed-data JSON edits are visible to atlas-configurations at runtime and require no rebuild. Phase 4 (Task 15 step 3) has the explicit gate.
7. **The audit CLI never auto-mutates `.go` files.** Every encoder fix is a hand edit anchored to a freshly-generated audit report. Do not pipeline-rewrite encoders.
8. **IDA MCP needs the right binary loaded.** Phase 0 / Phase 1 / Phase 2 assume v95 IDA. Phase 3 requires the user to swap binaries (the plan calls this out at each Phase 3 task start). If `mcp__ida-pro__get_metadata` returns the wrong binary, ask the user before continuing.
9. **gitleaks risk.** Audit reports sometimes embed absolute paths from `--atlas-packet` flag values — invoke the CLI with relative paths (`--atlas-packet libs/atlas-packet`) from the worktree root and the issue stays absent. Task 15 step 4 is the mandatory pre-PR scrub.
10. **Audit-report ack footer.** The literal line `Ack: world-audit Phase 2<x> on YYYY-MM-DD` is the LAST line of each `.md` report, written after every fix in that sub-phase has landed. Re-running an audit reverts the footer — `git checkout HEAD -- <report.md>` first if you need to re-execute.
11. **Conversation report (Task 9) is the highest-cognitive-load packet.** Plan ≥ 2 sessions: one to draft the per-type breakdown skeleton from IDA, one to verify each sub-section against atlas + run per-type tests. Do not collapse into a single commit.

---

## External dependencies / open questions resolved

- **IDA-MCP availability** — user-driven. Each Phase 3 task starts with a `get_metadata` check.
- **CSV paths** — `docs/packets/MapleStory Ops - ClientBound.csv` and `... - ServerBound.csv` already exist on the branch (shipped with task-027). No new CSVs.
- **v28 binary** — out of scope per design §14 (inherited from task-028 §6). If one surfaces mid-task, defer to a sibling.
- **JMS opcode-space divergence** — task-028 found JMS uses a separate opcode space for login; expect similar shape for world. Tasks 14 step 2 handles this with `"region": "JMS"` annotations in `gms_jms_185.json`.
- **Sub-op enum drift in pipeline** — task-028 §9 / design §3 acknowledges the limitation; documented in `_pending.md`, NOT fixed in this task.
- **Bare handler descent** — design §1 + PRD §3 non-goal; defer to `_pending.md`.
- **`instance-based-transports` coordination** — merged to `main` (commits `456ec8717`, `05bdfd7b0`). No coordination required; `transport.go` is audited in-place in Task 6.

---

## Plan-execution checklist (one-glance summary)

- **15 tasks.** Phase 0 = Task 1 (regression baseline). Phase 1 = Tasks 2–3 (registry fixtures). Phase 2 = Tasks 4–11 (world v95 audit, 8 sub-phases). Phase 3 = Tasks 12–14 (cross-version v83/v87/JMS-185). Phase 4 = Task 15 (closeout).
- **Sub-phase order (Phase 2):** 2a portal/serverbound → 2b field/serverbound → 2c field/clientbound non-effect → 2d field effect cluster → 2e npc/clientbound non-conversation → 2f conversation.go → 2g npc/serverbound → 2h `_pending.md` sweep. Easy wins front-loaded; conversation.go + dispatcher-offset verification deferred until accumulated context is highest.
- **Every Phase 2/3 sub-task is a tracking unit.** Plan checkboxes mark sub-tasks complete only when every bucket packet has a verdict row + every ❌ has a fix commit or `_pending.md` deferral.
- **Final exit gate:** Task 15 steps 2, 4, and 5. `go build`/`go vet`/`go test -race` clean; gitleaks scrub empty; login + character SUMMARY rows byte-identical to `/tmp/summary-pre-task068.md`.
- **No analyzer changes anywhere.** If forced, STOP and split a sibling task.
- **Code review BEFORE PR** — Task 15 step 7 runs `superpowers:requesting-code-review` (plan-adherence + backend-guidelines). Re-run after fix commits land.
