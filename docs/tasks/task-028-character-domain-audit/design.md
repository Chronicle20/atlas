# Character-Domain Packet Audit — Design

Version: v1
Status: Proposed
Created: 2026-05-14
PRD: `prd.md`
Prior art: `../task-027-atlas-packet-v95-audit/{design,plan,post-phase-b}.md`

---

## 1. Design Goals

This task takes a tool that has already shipped (`tools/packet-audit/`) and pointed it at a domain ~1.7× larger and qualitatively hotter than login. Architecture decisions follow from that delta, not from green-field analysis. The task-027 design doc is the architectural baseline; this doc only enumerates the deltas that the character domain forces.

Constraints driving the decisions below:

- **The pipeline already exists and works.** Don't re-design it. Extend the `TypeRegistry`, fix the documented `CharacterList ❌` false positive, and ship audit reports. Anything else is scope creep.
- **Character packets are hot.** A wrong byte width in `attack.go` or `damage.go` corrupts gameplay for every character every fight, every session. Login bugs hit once per session and got 12 commits to fix; character bugs will be invisible in QA logs because the symptoms ("client desync", "damage looks weird") are easy to dismiss. The audit pipeline's job is to make them undismissable.
- **The domain has multi-version branches everywhere.** `spawn.go` alone has 7 distinct `if t.Region()/t.MajorVersion()` branches. The analyzer's early-return modeling is load-bearing for the verdicts on these packets — task-027 left the bug in because login only had one offender (`CharacterList`); character will have many.
- **No retroactive scope creep.** Task-027 expanded mid-flight from "spike replication" to "full login audit + balloon support + sub-struct descent". This task explicitly bounds scope: 48 packets, 4 versions, one analyzer fix. Phase F-style "while we're here, let's also fix…" excursions get split into sibling tasks.
- **Bare-handler exclusion stays.** Task-027 deferred handlers without atlas-packet decoders to `_pending.md`. Same treatment here, same rationale: descending into atlas-channel/atlas-character service code dilutes the wire-shape audit with service-logic auditing.

---

## 2. Architecture Overview

No new architecture. The data flow is identical to task-027 §2:

```
CSV ─→ template ─→ IDA source ─→ atlas-packet analyzer ─→ diff engine ─→ report writer
                                                ↑
                                                │
                                          TypeRegistry
```

What changes for this task is **what the pieces ingest**:

| Piece                | task-027 input            | task-028 input                                              |
|----------------------|---------------------------|-------------------------------------------------------------|
| Atlas source         | `libs/atlas-packet/login` | `libs/atlas-packet/character/{clientbound,serverbound}/`    |
| IDA exports          | `gms_v95.json` (login)    | `gms_v95.json` (character, **append** to existing)         |
| IDA exports (cross)  | n/a (login was v95-only)  | `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` (new)    |
| Templates            | 6 templates, 8 opcode fixes | Same 6 templates, character-domain opcodes/sub-ops only   |
| TypeRegistry         | `CharacterStat`, `AvatarLook`, `Asset`, `ChannelLoad` | `+AttackInfo`, `+CharacterTemporaryStat::EncodeForeign`, `+Pet`, `+Avatar` (verify during execution; not exhaustive) |
| Analyzer             | early-return = unsupported (caused `CharacterList ❌`) | early-return = supported (this task fixes it)              |

The audit pipeline is **read-only** against `libs/atlas-packet/`; the only writes are to `tools/packet-audit/` (analyzer + registry), `libs/atlas-packet/character/` (wire-bug fixes), `services/atlas-configurations/seed-data/templates/template_*.json` (opcode/enum fixes), and `docs/packets/` (audit reports + IDA exports).

---

## 3. The hard part #1: early-return modeling in the analyzer

This is the analyzer fix called out in PRD §4.4. It is structurally identical to a problem the task-027 spike already documented — `CharacterList ❌` flags as a false positive because the analyzer treats sibling branches as cumulative when they are mutually exclusive via `return`.

### 3.1 Current behaviour

`tools/packet-audit/internal/atlaspacket/analyzer.go` walks the encoder body, collecting `Write*` calls into a flat list annotated with the conjunction of enclosing `if`-guards. The walk does not model control-flow exits. Consider:

```go
if a {
    w.WriteByte(1)
    return
}
w.WriteInt(2)
```

The analyzer emits `[WriteByte@guard=a, WriteInt@guard=true]`. The IDA decompile for the same shape emits `[Decode1] then [Decode4 if !a]`. The diff engine flattens both for the audit target tenant; for tenants where `a` is true the IDA path has one call and Atlas has two. False ❌.

### 3.2 The fix — taint the post-return suffix with the negated guard

The minimal change: when the walker enters an `if`-block, scan it for `*ast.ReturnStmt` as the terminal statement (recursively descended into nested blocks **only when every branch returns**). If terminal-return holds, every call collected **after** that `if`-block in the enclosing scope picks up an extra implicit guard: `NOT(enclosing-if-condition)`.

This generalises to:

- `if a { ...; return } w.WriteX()` → `WriteX@guard=NOT(a)`.
- `if a { ... } else { ...; return } w.WriteX()` → `WriteX@guard=a` (because the only non-returning branch is the `then` branch).
- `if a { ...; return } else { ...; return } w.WriteX()` → unreachable; emit `🔍 unreachable code` and skip the suffix entirely.

The conjoin logic in `guard.go` already supports `Not(...)`; the fix lives entirely in `analyzer.go` plus tests.

### 3.3 What about early-return inside a `for` loop?

Out of scope. The character domain has no encoders that early-return out of a loop body (verified via `grep -n 'return' libs/atlas-packet/character/**/*.go` during execution; if any exist they get `🔍 manual review`). Modeling loop-internal early-return correctly requires per-iteration guard tracking, which is bigger than this task wants to absorb. The login `CharacterList` case is at function-scope; that's what we fix.

### 3.4 Re-running login to validate the fix

The fix is only "done" when the login `CharacterList.md` verdict flips from ❌ to ✅ in `docs/packets/audits/gms_v95/SUMMARY.md`. Re-running the login audit is mechanical:

```
go run ./tools/packet-audit \
  --csv-clientbound  docs/packets/MapleStory\ Ops\ -\ ClientBound.csv \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

Validation of the fix also covers any other domain that uses the pattern. The cost of doing this re-run as part of *this* task is one command and one commit; the cost of skipping it is that the next domain audit re-discovers the bug.

### 3.5 What if the fix surfaces *new* ❌s in login?

Possible: removing a known false-positive may reveal real bugs masked by the same logic in other login packets. Treat each new ❌ on the relevant login re-run as in-scope for **this** task — every new bug found by the analyzer fix is by definition a bug the analyzer fix found, and triaging it isn't optional. Cap: if more than two new login ❌s appear, stop, document, and split a sibling task. Two is the rough budget the cross-version pass can absorb without missing the character-domain deadline.

---

## 4. The hard part #2: scaling the TypeRegistry without going on a sub-struct tour

Task-027 added 4 sub-structs to the registry. The character domain references at minimum:

- `model.Avatar` — already in registry as `AvatarLook` (confirm names match during execution).
- `model.CharacterTemporaryStat` — registered for login (`Encode`), but **not for `EncodeForeign`** which `spawn.go` and `buff_give.go` use.
- `model.AttackInfo` — not registered; referenced by `attack.go` and `damage.go`.
- `model.Pet` — not registered; referenced by `spawn.go`.
- `model.MovementInfo` — not registered; referenced by `movement.go` and `move.go`.
- `model.DamageTakenInfo` — not registered; referenced by `damage.go`.

That's ~5–6 new registrations on top of an existing pattern. The PRD §4.3 names these as expected; they're not surprises.

### 4.1 Registration discipline

For each new sub-struct:

1. Add the entry to `registry.go` pointing at the actual Go type (`model.AttackInfo`) with the right method name (`Encode` vs `EncodeForeign`).
2. Add a registry_test fixture that asserts the registered type analyzes to the expected primitive-field list. Format mirrors the existing `CharacterStat::Encode` fixture.
3. Don't pre-emptively register every type in `libs/atlas-packet/model/`. Register only what the audited writers actually call. Each registration is a future maintenance liability — every time the model type's wire shape changes, the registry test will catch the drift and the entry must be updated by hand.

Registration order matters operationally: register the sub-struct in the same commit that audits the first packet referencing it. That keeps PR-sized chunks tight ("audit attack + register AttackInfo" is one commit) and makes commit message → code change traceability obvious.

### 4.2 What about types that wrap atlas-channel-side encoders?

The character spawn references `model.Pet.Encode(l, ctx)(options)`. `model.Pet` lives in `libs/atlas-packet/model/`. Fine — register.

But `inventory/` has its own clientbound writers, and `spawn.go` does **not** call into them — it inlines the inventory snapshot via `cts.EncodeForeign`. Verified by reading `spawn.go:75`: `w.WriteByteArray(m.cts.EncodeForeign(...)(options))`. So we do **not** need to register the inventory package's encoders for this task. PRD open question 4 ("Inventory sub-struct ownership") resolves by inspection: spawn's inventory block is delivered via the secondary-stat sub-struct, not a separate inventory encoder. The `inventory` package's writers are out of scope.

### 4.3 Cross-domain ripple

A registry change is technically global — adding `AttackInfo` could affect any packet that uses it (currently only `attack.go` and `damage.go`). The diff engine treats registry additions as additive: any previously-unrecognised `<expr>.Encode(...)` call that now resolves through the registry simply produces a more accurate field list in the next run. Existing reports won't regress. Risk: low.

---

## 5. Phasing — concrete artifacts

Task-027 phasing (A–F) doesn't apply directly. This task has its own phasing, conceptually one level smaller:

### Phase 0 — Analyzer fix + login re-run (gate)

Ship the early-return fix and confirm `CharacterList ✅` before any character-domain work. If the fix surfaces > 2 new login ❌s, stop and split per §3.5.

Artifacts:
- `tools/packet-audit/internal/atlaspacket/analyzer.go` — early-return walker change.
- `tools/packet-audit/internal/atlaspacket/analyzer_test.go` — fixture for `if a { return } else { ... }` shape.
- `docs/packets/audits/gms_v95/CharacterList.{md,json}` — flipped to ✅.
- `docs/packets/audits/gms_v95/SUMMARY.md` — updated counts.

Exit: login `SUMMARY.md` shows 28/28 ✅ (was 27/28).

### Phase 1 — TypeRegistry extension batch

Register the predictable sub-structs up front to avoid death-by-1000-tiny-PRs during Phase 2:

- `model.AttackInfo` (`Encode`).
- `model.CharacterTemporaryStat` (`EncodeForeign`).
- `model.Pet` (`Encode`).
- `model.MovementInfo` (verify exact symbol name during execution).
- `model.DamageTakenInfo` (`Encode`).

Each registration lands with one registry test. The list is **predicted**, not exhaustive — if Phase 2 surfaces more, register them then. Don't register types nothing in the character domain references.

### Phase 2 — v95 character audit (clientbound + serverbound)

Run the audit, triage findings, ship fixes. Sub-phases:

- **2a — Clientbound static run**: produce 30 audit reports. Triage by verdict.
- **2b — Serverbound static run**: produce 18 audit reports. Triage by verdict.
- **2c — Real wire bug fixes**: each ❌ that triages to a real bug gets a fix commit on the task branch. Per PRD §4.6 every fix lands with a 4-variant test sweep and an IDA-citation comment.
- **2d — Template opcode / sub-op fixes**: each ❌ that triages to template drift gets a fix in the relevant `template_*.json`. Per PRD §4.7 every fix lands with the IDA case-statement value in the commit message.
- **2e — `_pending.md` updates**: each bare-handler or out-of-scope writer gets an explicit row.

The audit is "done" when `SUMMARY.md` shows a verdict for every one of the 48 packets *or* an explicit `_pending.md` entry for any that's deferred. No silent skips.

### Phase 3 — Cross-version pass (v83 → v87 → JMS v185)

One binary at a time, user-driven IDA swap. Per PRD §4.5. For each version:

1. User loads the IDA database.
2. Walk the character-domain FName list (already established by Phase 2's v95 export).
3. Populate the matching `gms_v{83,87}.json` / `gms_jms_185.json` for character FNames.
4. Re-run the audit with that version's IDA source and template.
5. For every divergence vs v95 atlas-packet behaviour, either:
   - Existing `Region/MajorVersion` gate already handles it correctly → no change, but the export row captures the IDA evidence.
   - Atlas's gate is wrong → fix the gate, sweep tests across 4 variants, document.
   - Template opcode drift → fix the template, cite case-statement value.

Each version's pass ships as its own commit batch ("phase-3-v83", "phase-3-v87", "phase-3-jms-185") so reviewers can see which IDA findings drove which atlas-packet changes.

### Phase 4 — post-phase-b.md + finishing-a-development-branch

Mirror the task-027 closing pattern:

- Write `docs/tasks/task-028-character-domain-audit/post-phase-b.md` with the same five sections (final state, real wire bugs fixed, template fixes, tooling improvements, remaining work).
- Run `go build`, `go vet`, `go test -race` on `libs/atlas-packet/` and `tools/packet-audit/`.
- Run `docker build -f services/atlas-configurations/Dockerfile .` if templates changed in ways that touch the seed-data structure (key additions are usually fine; new sub-key types aren't — verify per PRD acceptance).
- Run `superpowers:requesting-code-review` (plan-adherence + backend-guidelines).
- Open PR.

---

## 6. v28 coverage (PRD open question 2 resolution)

The PRD asks whether to seek a v28 binary for verification. **Recommendation: no for this task.** Reasons:

- task-027 left v28 unverified for the same reason; the assumption that v28 is structurally close to v83 has held since v28 templates landed.
- The audit pipeline will still produce reports for v28 if pointed at an export — the export just won't exist. Findings would be inference-only ("v83 IDA suggests v28 also writes…").
- The user's binary collection is v83/v87/v95/JMS-185. Adding v28 is a binary-acquisition task, not a packet-audit task.

If a v28 binary surfaces mid-task, defer it to a sibling task; don't pause Phase 3 to integrate it.

If user has updated their binary collection since the task-027 closing memo, treat that as a Phase 3 input change and document, but the planning assumption is unchanged.

---

## 7. JMS divergence (PRD open question 3 resolution)

JMS v185 login used a different opcode space; the same may be true for character. Working policy:

- If JMS opcodes diverge but the wire shape is identical, only the template needs updating. Atlas-packet code stays put.
- If JMS wire shape diverges (different field widths, extra fields, reorderings), the `Region() == "JMS"` branch in the affected encoder gets fixed. Pattern already exists in `spawn.go:79` and `spawn.go:142`.
- If JMS divergence is so extensive that a single encoder becomes a 4-way switch, split into a sibling file `<name>_jms.go` per task-027 §5.3.
- Hard cap: a single character-domain packet's atlas-packet code path may not contain more than two nested `if t.Region()` / `if t.MajorVersion()` levels. If a fix needs 3+, that's a structural rewrite, which is out of scope — log to `_pending.md` and move on.

### 7.1 "JMS bugs become out-of-scope unless atlas-packet writes wrong bytes"

The PRD names this explicitly. Operationalise:

- **In scope**: atlas-packet's encoder writes bytes the JMS v185 client decodes incorrectly (any version).
- **Out of scope**: JMS v185 has functionality the client expects that atlas does not implement at the service layer (e.g. a feature exists but isn't wired through). That's a service-layer gap, not a wire-shape bug.
- **In scope**: JMS v185's IDA shows a field atlas writes as `Decode1` but atlas writes a `WriteInt`. Width mismatch.
- **Out of scope**: JMS v185's IDA shows the same field as v95 IDA and atlas's code is correct — but JMS templates have the wrong opcode. Out of scope only if v95 isn't broken; if both are broken on opcodes, fix both.

---

## 8. Hot-path testing discipline

Login packets were not on hot paths. Character packets are. Specifically:

- `movement.go` — fires every few hundred ms per character in field. ~30× more bytes than a login packet over a session.
- `attack.go`, `damage.go` — fire on every attack swing. Hot, but **per-attack** is still cheap relative to the wire.
- `buff_give.go`, `buff_cancel.go`, `skill_change.go` — fire on every skill/buff state change.
- `spawn.go`, `despawn.go` — fire on every character entering/leaving field.

PRD §8.1 says no perf work; respect that. But the audit fixes need to **not regress** the encoder by adding allocations or extra reflection. Patterns to use:

- Never introduce `reflect.*` in an encoder fix. The encoder dialect (`w.WriteX(...)`) stays exactly as it is; fixes change which `WriteX` is called or under which guard, not how the encoder is structured.
- Never introduce an `interface{}` parameter or option that wasn't there pre-fix. If a fix needs a new variant axis (e.g. "stock-Nexon" was task-027's example), that variant comes from `tenant.Model`, not from an option.
- Tests assert byte output (not just round-trip) per PRD §8.5. Hex strings come from IDA decompiles, captured by hand and verified by re-running the analyzer.

If perf-regression suspicion ever arises (e.g. a fix changes the byte layout of `movement.go`), add a `BenchmarkEncode` to the test file and run before/after. The repo has no benchmark CI gate; this is local maintainer discipline, not enforced.

---

## 9. Template fixes — opcodes vs sub-ops

Task-027 fixed both kinds (7 opcode shifts + 1 enum value). Same dual surface for this task. Important distinction the PRD §4.7 elides:

- **Opcode drift** = the `case N` in the dispatch switch changed between versions. Fix is a single integer in `template_*.json`. Trivial.
- **Sub-op (enum) drift** = a writer that already routes correctly emits a sub-op byte whose value-to-meaning mapping shifted. Example: `DeleteCharacterResponse.NEXON_ID_DIFFERENT_THEN_REGISTERED` shifted from 16 to 26. Fix is a value in the template's enum map. Less trivial — requires reading the IDA function's internal switch table, not just the top-level dispatcher.

Character domain almost certainly has sub-op drift in:

- `status_message.go` (multiple sub-types, dispatched by leading byte).
- `effect.go` / `effect_skill_use.go` / `effect_quest.go` (effect-type byte dispatch).
- `skill_change.go` (result-byte dispatch).

The audit pipeline does not currently model sub-op enums; it audits the wire shape per atlas writer FName. Sub-op drift is caught only when an audit report manually annotates "the sub-op byte for effect type X is value N in atlas but value M in v95 IDA". This is a known pipeline limitation; not in scope to fix here. Document it in `_pending.md` as a follow-up tooling item.

---

## 10. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Analyzer fix overshoots — flips a *real* ❌ to a false ✅ in some packet | Medium | High | Fixture-first TDD: write the test that asserts the new behaviour on the `CharacterList` shape AND a test that asserts a non-early-return ❌ stays ❌. Run full login regression as part of Phase 0 gate. |
| Sub-struct registry blows up — one packet pulls in a chain of 4+ sub-structs none of which are registered | Low-medium | Medium | Phase 1's predicted registration list covers the load-bearing types. Mid-audit additions are cheap (one registry entry + one test). If a chain unexpectedly explodes (5+ unregistered sub-structs in one packet), pause and triage — it usually means the packet is doing something unusual, not that the registry strategy is wrong. |
| Movement/attack/damage hot paths see a wire-bug fix that subtly changes byte layout for v83/v87, breaking existing tenants | Medium | High | 4-variant test sweep is mandatory per fix. Round-trip tests for each variant assert leftover-bytes == 0. Cross-version Phase 3 catches any gate that's narrower or wider than it should be. |
| Bare-handler exclusion turns out to swallow a critical packet (e.g. `create`) | Low | Medium | `_pending.md` row is explicit per excluded packet. If a reviewer flags a deferred packet as critical, the row's IDA address makes the follow-up sibling task one query away. |
| Cross-version pass surfaces a v83 regression introduced by a v95 fix | Medium | High | Phase 2 fixes are gated on `MajorVersion()` correctly. Phase 3 v83 pass re-runs the full audit; v83 regression shows up as a fresh ❌. Plan owner reviews every Phase 3 diff. |
| JMS v185 ends up requiring a whole-encoder rewrite of a hot packet | Low | High | §7's hard cap on nested region/version guards. If hit, log to `_pending.md` and split a sibling task; don't expand scope. |
| Template sub-op enum drift is wide and the pipeline can't model it | High | Low-medium | Document the limitation, accept manual sub-op verification for status-message-style packets, ship the `_pending.md` follow-up tooling row. |
| The retroactive scope expansion that bit task-027 (single-task PR ballooned to 4× its plan) repeats here | Medium | Medium | This design explicitly disallows mid-task pivots into adjacent domains, balloon-equivalent features, or stock-Nexon-style variants. Every "while we're here" gets logged as a future sibling task instead of absorbed. |
| Branch hygiene drift: a fix-test cycle on hot packets generates noisy commit history | Medium | Low | One commit per packet fix (or per template fix group). `superpowers:finishing-a-development-branch` rebase-cleans before PR. |
| gitleaks catches absolute paths in audit reports (task-027 had this) | High | Low | Per PRD §10 acceptance: reviewers scrub their own output. Phase 4 pre-PR check: `grep -r '/home/' docs/packets/audits/gms_v95/character/` must be empty. |

---

## 11. Out of scope (explicit)

To anchor scope and prevent the task-027 ballooning pattern:

- Bare-handler descent into atlas-channel or atlas-character service code (PRD §3 non-goal, PRD §9 q1).
- v28 binary integration (§6).
- Sub-op enum modeling in the audit pipeline (§9 — limitation acknowledged, not fixed).
- Sub-struct registry coverage for any type the character domain doesn't reference (§4.1).
- atlas-channel-side handlers whose decoders live in service code (PRD §3).
- Performance work on hot packets (PRD §8.1 + §8.6).
- NPC, monster, drop, field, inventory, party, guild, buddy, chat domain audits — all are sibling tasks per PRD §3.
- Service-layer adapter changes beyond the minimum needed to wire a fix through (PRD §3).
- Movement-loop early-return modeling in the analyzer (§3.3).
- Generic packet-DSL or schema-first encoder rewrite (task-027 §12).
- New `clientVariant` axes beyond modified/stock (task-027 already shipped this; nothing new here).

---

## 12. Reference points in the existing tree

- `libs/atlas-packet/character/clientbound/spawn.go:79-145` — load-bearing example of multi-version branching in a hot packet. Where the analyzer's verdict matters most.
- `libs/atlas-packet/character/clientbound/attack.go:93-146` — uses `AttackInfo` sub-struct; first registry addition target.
- `libs/atlas-packet/character/clientbound/buff_give.go:26-50` — uses `CharacterTemporaryStat.Encode` (already registered) + `EncodeForeign` (needs registration).
- `libs/atlas-packet/character/clientbound/damage.go` — second `AttackInfo` consumer.
- `libs/atlas-packet/model/attack_info.go`, `pet.go`, `damage_taken_info.go` — sub-struct sources to register.
- `tools/packet-audit/internal/atlaspacket/analyzer.go:115-160` — region where the early-return walker lives (the walker itself is in `collectCallsWithCtx`; the if-handling around line 257-321).
- `tools/packet-audit/internal/atlaspacket/registry.go` — registration site for new sub-structs.
- `docs/packets/audits/gms_v95/SUMMARY.md` — top-level audit index; gets character-domain rows added.
- `docs/packets/audits/gms_v95/CharacterList.md` — current ❌ false-positive report.
- `docs/packets/ida-exports/gms_v95.json` — append character FNames during Phase 2.
- `docs/packets/ida-exports/_pending.md` — append character bare-handler exclusions during Phase 2.
- `docs/tasks/task-027-atlas-packet-v95-audit/post-phase-b.md` — template for this task's closing memo.

---

## 13. What plan-task should do next

The plan should split this design into **explicit, sequenced, small** tasks. Suggested structure:

- **5–8 tasks for Phase 0 + Phase 1** (analyzer fix, login re-run, predicted registry batch + tests).
- **One task per ~6 packets for Phase 2 clientbound** (5 sub-tasks), **one per ~6 packets for serverbound** (3 sub-tasks). Each task ends with a verdict-triaged commit set. Fix commits inside a sub-task are individual; the sub-task is a tracking unit, not a single PR.
- **One task per version for Phase 3** (3 sub-tasks: v83, v87, JMS-185).
- **One task for Phase 4** (post-phase-b.md + verification + code review).

Total target: ~14–18 plan tasks. Anything more is the plan re-deriving the audit per-packet; anything less is hiding scope.

Specifically the plan should answer:

- Which 6 clientbound packets are most likely to surface real wire bugs (suggest: `spawn`, `attack`, `damage`, `buff_give`, `movement`, `skill_change`). Front-load them so the analyzer fix gets exercised on the hot packets before the cold ones.
- The exact fixture inputs for the analyzer's early-return tests, including the negative case (a non-returning `if` should NOT taint the suffix).
- The exact registry test format (existing fixtures are in `registry_test.go`; match them).
- Whether to bundle all v95 fixes into one PR or split per sub-domain (suggestion: split per Phase 2 sub-task — each is naturally a reviewable chunk).
- The Phase 3 commit naming convention so the review history reads cleanly across versions.
