# Skill Macro Version Coverage — Implementation Context

Companion to [`plan.md`](./plan.md). Everything an executor needs that the plan's
task list assumes rather than restates.

Task: `task-226-skill-macro-version-coverage`
Worktree: `.worktrees/task-226-skill-macro-version-coverage`
Branch: `task-226-skill-macro-version-coverage`

---

## 1. The problem in one paragraph

Skill macros are fully implemented server-side but unroutable on half the version
columns, and the codec that would be verified is not the codec that ships. Four
tenant versions (gms_87, gms_92, gms_95, jms_185) never bind the serverbound
handler opcode, so every macro save is dropped at dispatch; gms_92 also never
binds the clientbound writer, so its macro list never loads. Meanwhile two
different clientbound encoders exist in the tree with *opposite* shout polarity,
only one is called, and the only test round-trips the uncalled one against a
decoder with the same inversion — so the test is green while the shipped pair
disagrees. Both codecs also sit on paths `packet-audit` will not walk, so
neither matrix row can promote where it stands.

## 2. Key files

### Codecs (before this task)

| File | Role | Note |
|---|---|---|
| `libs/atlas-packet/character/skill_macro.go` | `SkillMacro` — `Encode` (dead) + `Decode` (shipped) | shout inverted on both sides; deleted in Task 8 |
| `libs/atlas-packet/model/macros.go` | `model.Macros` / `model.Macro` — the **shipped** clientbound encoder | shout upright; deleted in Task 8 |
| `libs/atlas-packet/character/skill_macro_test.go` | round-trip only | the double inversion cancels; deleted in Task 8 |

### Codecs (after this task)

| File | Role |
|---|---|
| `libs/atlas-packet/character/clientbound/skill_macro.go` | `SkillMacro` — the one clientbound encoder, audited path |
| `libs/atlas-packet/character/serverbound/skill_macro.go` | `SkillMacro` — the one serverbound decoder, audited path, bounded loop |

### atlas-channel call sites

| File:line | What changes |
|---|---|
| `main.go:91` | drop the `character2` import |
| `main.go:730` | writer registration → `charcb.CharacterSkillMacroWriter` |
| `main.go:943` | handler map key → `charsb.CharacterSkillMacroHandle` |
| `kafka/consumer/session/consumer.go:368-373` | login announce → `charcb.NewSkillMacro` |
| `kafka/consumer/macro/consumer.go:80-85` | update announce → `charcb.NewSkillMacro` |
| `socket/handler/character_skill_macro.go` | decode via `charsb.SkillMacro`, getters instead of exported fields |

`macro.Model` (`services/atlas-channel/atlas.com/channel/macro/model.go:40`) and the
`skill_macro_status_event` Kafka contract are **unchanged**. No REST change, no
schema change.

### Packet-audit surface

| File | Role |
|---|---|
| `tools/packet-audit/cmd/run.go` — `candidatesFromFName` | maps IDA fname → codec; both macro ops absent today (Task 9) |
| `tools/packet-audit/cmd/run.go:208-215` | the `reportName` override field and why it exists |
| `tools/packet-audit/cmd/na_consistency.go:15-24` | the feature-family n-a gate |
| `docs/packets/feature-families.yaml` | family declarations (Task 10 adds `skill_macro`) |
| `docs/packets/feature-na-evidence.yaml` | positive-absence records (Task 4 may add) |
| `docs/packets/registry/*.yaml` | opcode + fname authority per version |
| `docs/packets/ida-exports/*.json` | report-generation input; both functions missing/stubbed (Task 2) |
| `docs/packets/audits/STATUS.md:177,650` | the two rows this task promotes |

### Templates

`services/atlas-configurations/seed-data/templates/`. Current macro bindings,
audited at plan time:

| Template | handler | writer |
|---|---|---|
| `template_gms_12_1.json` | — | — (out of scope, not a matrix column) |
| `template_gms_48_1.json` | — | — |
| `template_gms_61_1.json` | — | `0x5B` (`:2524`) |
| `template_gms_72_1.json` | `0x6D` (`:806`) | `0x71` (`:2674`) |
| `template_gms_79_1.json` | `0x6C` (`:806`) | `0x75` (`:2714`) |
| `template_gms_83_1.json` | `0x6E` (`:878`) | `0x7C` (`:2842`) |
| `template_gms_84_1.json` | `0x6E` (`:882`) | `0x7F` (`:2874`) |
| `template_gms_87_1.json` | **missing** | `0x84` (`:2626`) |
| `template_gms_92_1.json` | **missing** | **missing** |
| `template_gms_95_1.json` | **missing** | `0x8C` (`:2933`) |
| `template_jms_185_1.json` | **missing** | `0x7A` (`:2537`) |

## 3. Decisions already made (do not relitigate)

| Decision | Where | Why |
|---|---|---|
| One struct per direction under `character/{clientbound,serverbound}/` | design §2.1 | `locateAtlasFile` only walks `/clientbound/` and `/serverbound/` paths (`run.go:3624-3659`); no relocation → no report → no ✅ |
| `model.Macros` is **absorbed and deleted**, not wrapped | design §2.2 | a wrapper preserves the ability for audited and shipped codecs to drift — the exact failure this task exists to close |
| Serverbound `reportName: "CharacterSkillMacroHandle"` | design §2.1 | both directions derive `CharacterSkillMacro`; the audit dir is flat and writerName-keyed. Precedent: `run.go:1137` `SummonMoveHandle` |
| Inline version gates, not per-version files | design §2.3 | per-version files only on a *structural* divergence (e.g. fixed-width name), not a field-width delta |
| Templates land **last** | design §4 | playbook §4: divergence ⇒ wire fix first; don't widen an unverified decoder from 4 tenants to 8 |
| `feature-families.yaml` entry added mid-task even though it reds the gate | design §2.5 | it is the open-question tracker for v48/v61; green is a Task 13 requirement |
| PRD's "atlas-channel: no source change expected" is **overridden** | design §2.2 | three atlas-channel files change |
| PRD's "v83/v84 byte-identical to main" **yields** to the IDA polarity verdict | design §5 | deriving from the client is the point; any deviation is recorded and called out in the PR |

## 4. The three verdicts everything hangs on

Produced by Task 3 in `layout-derivation.md`; consumed by Tasks 6, 7, 10, 11.

1. **Shout polarity** — `UPRIGHT` / `INVERTED` / `UNKNOWN`. Production writes it
   upright (`model/macros.go:50`) and reads it inverted
   (`character/skill_macro.go:64`); one of those is corrupting saved macros on
   gms_83/84 *today*. `UNKNOWN` ⇒ stop and ask; a coin flip silently corrupts
   every saved macro.
2. **Capacity** — the client's maximum macro count, read from the loop bound in
   `CWvsContext::OnMacroSysDataInit`. Becomes `maxMacroCount` in the serverbound
   decoder. `UNKNOWN` ⇒ stop and ask; do not invent a cap.
3. **Divergences** — the per-version/per-field list that becomes the gate set. An
   *empty* list is a legitimate finding and must be stated explicitly, not left
   implicit.

## 5. Dependency order

```
1 baseline  ──▶ (independent, but must run before Task 8 deletes model/macros.go)

2 harvest/splice exports
      │
      ▼
3 layout derivation ──┬──▶ 4 v48/v61 n-a re-check
      │               └──▶ 5 v72 registry fix
      ▼
6 clientbound codec ──┐
7 serverbound codec ──┼──▶ 8 rewire atlas-channel + delete legacy
                      │
                      ▼
                  9 packet-audit linkage + reports
                      │
                ┌─────┴─────┐
                ▼           ▼
        10 verify CB   11 verify SB
                └─────┬─────┘
                      ▼
              12 template routing  ──▶  13 reconciliation + guards + review
```

Tasks 1–5 are sequential-ish and IDA-bound; 6 and 7 are independent of each other;
10 and 11 are independent of each other. Tasks 10/11 legitimately leave some cells
red until Task 12 routes the opcodes — that specific red (`routedElsewhere &&
!routed`) is expected and named in each task.

## 6. Environment and tooling gotchas

- **IDA sessions.** `mcp__ida-pro__idb_list`, then pass the binary **name** as
  `database`. `select_instance(port)` is dead. Ports rotate between launches.
- **packet-audit + IDA.** `-ida-database "<binary name>"` is required. The
  `-ida-url` default points at a dead port and silently targets the wrong binary.
- **Export splicing.** Never re-export wholesale — it drifts ~150 unrelated
  function keys. Harvest to `/tmp/claude-…`, splice by hand, and check
  `git diff --stat docs/packets/ida-exports/`.
- **Registry fname edits stale the matrix.** Always follow with
  `go run ./tools/packet-audit matrix`.
- **`tools/lint.sh --check` false-fails without nvm** on PATH, and contends on a
  cross-worktree golangci-lint lock. If it reports atlas-ui findings on a branch
  that touched no TS, check the environment before treating it as real.
- **`pt.Variants` is positional.** Append only; existing indices are referenced
  by index across the whole packet library.
- **`WriteAsciiString`** = LE `uint16` length + ShiftJIS-encoded bytes
  (`libs/atlas-socket/response/writer.go:82-93`). ASCII-only fixture names keep
  that a no-op. **`WriteInt`** is LE `uint32` (`writer.go:36`).
- **Scratchpad** for all temp files:
  `/tmp/claude-…` paths as used in the plan; never write temp artifacts into the
  repo.

## 7. Artifacts this task produces

| Path | Produced by |
|---|---|
| `baseline-bytes.md` | Task 1 |
| `harvest-log.md` | Task 2 |
| `layout-derivation.md` | Task 3 (FR-1.5) |
| `na-recheck.md` | Task 4 (FR-2) |
| `live-tenant-reconciliation.md` | Task 13 (FR-5.4) |
| `audit.md` | Task 13 (reviewer agents) |
| `completeness-critic.md` | Task 13 (`packet-completeness-critic`) |

All live under `docs/tasks/task-226-skill-macro-version-coverage/`.

## 8. Out of scope

- Macro persistence semantics, the `macro` REST/Kafka contract, `atlas-character`
  storage.
- The `ANTI_MACRO_*` family — unrelated anti-botting, shares only a name fragment.
- `template_gms_12_1.json` — v12 is not a coverage-matrix column.
- Live-tenant PATCHes — post-merge, user-run; this branch delivers the input doc.
- Any atlas-ui surface.
- The wider gms_92 template gap (70 handlers vs gms_95's 137). This task adds
  exactly the two macro bindings. That gap is real, pre-existing, and not created
  here.
