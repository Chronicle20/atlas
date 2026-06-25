# Character Packet-Fixture Verification Campaign — Design

Task: task-109-character-packet-fixtures
Phase: 2 (Design)
Created: 2026-06-23
Status: Draft for review

---

## 1. Problem & Framing

The `character` family is the **largest implemented family** in the coverage matrix
(`docs/packets/audits/STATUS.md`): 71 rows, of which **47 `incomplete` cells across 21
rows** remain (confirmed by enumerating `status.json`, not estimated). The goal is to
drive every `incomplete` character cell to `verified` (✅) — or to a *justified* `n-a`
where a function is genuinely version-absent — landing each promotion as the coupled
artifacts the playbook (`docs/packets/audits/VERIFYING_A_PACKET.md`) requires.

The PRD framed this as a *mixed* campaign: some single-version holes that port from a
verified sibling, several packets unverified on *every* version needing fresh
decompilation. That framing is correct, but investigation surfaced four facts that
materially reshape the work and are recorded here before any cell is touched:

1. **Three packets have no byte-test file at all** — `clientbound/list.go`,
   `clientbound/appearance_update.go`, `clientbound/effect_quest.go` (verified by
   directory listing: `*_test.go` absent for each). These are the *fully-unverified T1
   group*; they need **brand-new cross-version byte-fixtures written from fresh
   per-version decompiles**, not a marker added to an existing test. This is the
   highest-effort slice and is sequenced first (§8 Phase A).
2. **A real Class-E cluster exists** (unlike login, where only one jms function was
   absent). Four functions are absent from older/jms exports yet **verified on v87/v95** —
   i.e. present-but-unnamed in those IDBs, not genuinely missing (§5). These are
   producible by name + surgical splice, **not** `n-a`.
3. **The bulk is deterministic report-gen** ("no audit report" cells whose client
   function is present in the export) — same cheap path as login Stage 1 (§4 Class A).
4. **Every character op is already wired** in `tools/packet-audit/cmd/run.go`
   `candidatesFromFName` (§7) — no new linkage case is expected.

### 1.1 Corrections to PRD assumptions (resolved from source)

- **Codec ownership (PRD §7 said "atlas-login / atlas-channel").** All character wire
  codecs and their fixture tests live in **`libs/atlas-packet/character/{clientbound,
  serverbound}/`** — a single Go module, `libs/atlas-packet`. `services/atlas-login`
  (lifecycle: view-all/list/create/delete/check-name) and `services/atlas-channel`
  (in-field: spawn/move/buff/expression/chair/appearance/effect/keymap/ap/heal) only
  *consume* these codecs. The login-vs-channel split the PRD asks about is a **consumer
  split, not a codec-location split**; the files this campaign touches are all under
  `libs/atlas-packet/character/`. **The only changed Go module is `libs/atlas-packet`.**
- **`CharacterList` is wired and emitted, not latent (PRD Open Question 2, resolved).**
  `services/atlas-login/.../socket/writer/character_list.go` and
  `socket/handler/character_list_world.go` exist and emit it via `CLogin::OnSelectWorldResult`.
  It is `incomplete` on all five versions purely because `list.go` has **no byte-fixture**
  — it belongs in the Phase-A fresh-decompile group, not flagged as dead code.
- **Duplicated rows enumerate by op, not by packet path (PRD Open Question 3).**
  `status.json` carries `EffectQuest` as **two op-rows** (`SHOW_FOREIGN_EFFECT`,
  `SHOW_ITEM_GAIN_INCHAT`, both via `CUser::OnEffect`), `AutoDistributeAp` as **two
  op-rows** (`DISTRIBUTE_AP`, `AUTO_DISTRIBUTE_AP`, both via `CWvsContext::SendAbilityUpRequest`),
  and `KeyMapChange` as an **op-row** (`CHANGE_KEYMAP`) **plus a `sub-struct` row**
  (v84-only `no audit report`). Each is a distinct matrix cell needing its own
  fixture/evidence/report; §2 keys every cell by op so none collapses.

---

## 2. The 47 cells, enumerated by (packet, op, fname, versions)

Column order matches the matrix: **v83 · v84 · v87 · v95 · jms_v185**. Every cell is keyed
by its distinct op even where the packet path repeats. "note" is the verbatim
`status.json` note (the verdict symbol is the audit report's verdict, *not* the matrix
cell state — the executor re-adjudicates it per cell, never trusts it blind).

### Clientbound (34 cells)

| Packet | op | fname | Incomplete versions | note |
|---|---|---|---|---|
| `CharacterViewAllCharacters` | `VIEW_ALL_CHAR` | `CLogin::OnViewAllCharResult` | v84, jms | v84 "tier-1 without fixture; 🚫"; jms "no audit report" |
| `CharacterList` | `CHARLIST` | `CLogin::OnSelectWorldResult` | v83, v84, v87, v95, jms | v83/v87/v95 "verdict ❌"; v84 "🔍"; jms "no report" — **no test file (§8 Phase A)** |
| `AddCharacterEntry` | `ADD_NEW_CHAR_ENTRY` | `CLogin::OnCreateNewCharacterResult` | jms | "no audit report" |
| `BuffGive` | `GIVE_BUFF` | `CWvsContext::OnTemporaryStatSet` | jms | "no audit report" |
| `CharacterInfo` | `CHAR_INFO` | `CWvsContext::OnCharacterInfo` | jms | "no audit report" |
| `CharacterSpawn` | `SPAWN_PLAYER` | `CUserPool::OnUserEnterField` | jms | "no audit report" |
| `CharacterMovement` | `MOVE_PLAYER` | `CUserRemote::OnMove` | v84, jms | both "tier-1 without fixture; verdict ❌" |
| `CharacterExpression` | `FACIAL_EXPRESSION` | `CAvatar::SetEmotion`, `CUser::OnEmotion` | v83, v84 | both "no audit report" — **Class E (§5)** |
| `CharacterChairShow` | `SHOW_CHAIR` | `CUserRemote::OnSetActivePortableChair` | v83, v84, jms | all "no audit report" — **Class E (§5)** |
| `CharacterAppearanceUpdate` | `UPDATE_CHAR_LOOK` | `CUserRemote::OnAvatarModified` | v83, v84, v87, v95, jms | all "tier-1 without fixture; 🔍" — **no test file (§8 Phase A)** |
| `EffectQuest` | `SHOW_FOREIGN_EFFECT` | `CUser::OnEffect` | v83, v84, v87, v95, jms | all "🔍" — **no test file (§8 Phase A)** |
| `EffectQuest` | `SHOW_ITEM_GAIN_INCHAT` | `CUser::OnEffect` | v83, v84, v87, v95, jms | all "🔍" — **no test file; distinct op (§8 Phase A)** |
| `BuffGiveForeign` | `GIVE_FOREIGN_BUFF` | `CUserRemote::OnSetTemporaryStat` | jms | "no audit report" |

### Serverbound (13 cells)

| Packet | op | fname | Incomplete | note |
|---|---|---|---|---|
| `CheckName` | `CHECK_CHAR_NAME` | `CCashShop::SendCheckDuplicateIDPacket`, `CLogin::SendCheckDuplicateIDPacket` | v83, v84, jms | all "no audit report" — v83/v84 **Class E**, jms Class A (§5) |
| `CreateCharacter` | `CREATE_CHAR` | `CLogin::SendNewCharPacket` | jms | "no audit report" |
| `DeleteCharacter` | `DELETE_CHAR` | `CLogin::SendDeleteCharPacket` | jms | "no audit report" |
| `AutoDistributeAp` | `DISTRIBUTE_AP` | `CWvsContext::SendAbilityUpRequest#DistributeAp` | v84 | "tier-1 without fixture; verdict ❌" |
| `AutoDistributeAp` | `AUTO_DISTRIBUTE_AP` | `CWvsContext::SendAbilityUpRequest#AutoDistributeAp` | v84 | "tier-1 without fixture; verdict ❌" |
| `HealOverTime` | `HEAL_OVER_TIME` | `CWvsContext::SendStatChangeRequest` | jms | "no audit report" (function present in jms export → Class A) |
| `KeyMapChange` | `CHANGE_KEYMAP` | `CFuncKeyMappedMan::SaveFuncKeyMap` (+ pet-consume variants) | v83, v87, v95, jms | all "no audit report" (v84 op-row already ✅) |
| `KeyMapChange` (sub-struct) | — | (links via `SaveFuncKeyMap`) | v84 | "no audit report" |

Cell count: 2+5+1+1+1+1+2+2+3+5+5+5+1 (clientbound = 34) + 3+1+1+1+1+1+4+1 (serverbound
= 13) = **47**. The plan MUST list each op-keyed row so the two `EffectQuest`, two
`AutoDistributeAp`, and the KeyMapChange op-vs-sub-struct rows each get their own
fixture/evidence/report; the report filename and marker `packet=` path derive from the
struct's `qualifiedWriterName` (`TitleCase(pkg)+Struct`, e.g. `CharacterEffectQuest`),
per `run.go`.

---

## 3. Promotion mechanism (per playbook)

A character cell promotes when, for that `packet × version`, **all** of these exist and
agree:

1. A `// packet-audit:verify packet=character/<dir>/<Struct> version=<v> ida=0x<addr>`
   marker stacked above the byte-test (confirmed format from existing
   `buff_give_test.go` / `create_test.go`; jms token is `jms_v185`, e.g.
   `chat/.../version=jms_v185`).
2. A per-version **audit report** `docs/packets/audits/<version>/Character<Struct>.{json,md}`
   — generated deterministically by the **root** `packet-audit` command against the
   committed export when the client function is present (no live IDA); its absence is the
   literal `"note":"no audit report"` on most incomplete cells.
3. A pinned **evidence record** `docs/packets/evidence/<version>/character.<dir>.<Struct>.yaml`
   with a `verifies:` line and an `ida.function` (carrying the `#suffix` for the split ops).
   Serverbound: always; clientbound: tier-1 — and **every** in-scope row is `tier1: true`,
   so evidence is required for all 47 cells.
4. (Serverbound) the op is **routed** in that version's seed template
   (`services/atlas-configurations/seed-data/templates/template_<v>.json`; the jms file is
   **`template_jms_185_1.json`** — resolve the exact filename per version, never assume).

The byte-fixture itself is the load-bearing artifact: a full-body byte test, not a mode
or length enumeration. The large bodies (`CharacterList`, `CharacterAppearanceUpdate`,
`CharacterInfo`, `CharacterSpawn`) must exercise the nested avatar/look block end to end.

---

## 4. Work-classes (drives plan ordering)

Cells fall into four classes by *what is actually missing*. The class is a hypothesis
from the note + export grep; the executor confirms it per cell (export grep → report-gen
→ decompile only when the report says ❌ or the function is absent).

- **Class A — report-gen only (no live IDA).** Client function present in the committed
  export; the cell is `incomplete` only because the per-version report was never copied
  in. Action: run root report-gen → copy `Character<Struct>.{json,md}` into
  `docs/packets/audits/<v>/` → add marker + evidence if absent → regen matrix.
  **Candidates (function present, "no audit report"):** all jms holes whose function is
  present (`AddCharacterEntry`, `BuffGive`, `CharacterInfo`, `CharacterSpawn`,
  `BuffGiveForeign`, `CreateCharacter`, `DeleteCharacter`, `HealOverTime`,
  `CheckName` jms), `KeyMapChange` v83/v87/v95/jms + v84 sub-struct,
  `CharacterViewAllCharacters` jms.
- **Class B — ❌ verdict, needs a decompile to adjudicate.** The generated report's read
  order disagrees with the Atlas codec. Decompile that version's client function and
  decide: **(i) stale/cosmetic** (codec actually matches → regenerate + re-pin) or
  **(ii) real wire delta** → fix-first path (§9). **Candidates:** `CharacterMovement`
  v84/jms, `AutoDistributeAp` v84 ×2, `CharacterViewAllCharacters` v84 (🚫), plus the
  `CharacterList` v83/v87/v95 ❌ verdicts surfaced inside Phase A.
- **Class C — fully-unverified, no test file (fresh per-version decompile).** No
  `*_test.go` exists; the read order must be derived per version and a brand-new
  cross-version byte-fixture written. **Candidates:** `CharacterList`,
  `CharacterAppearanceUpdate`, `EffectQuest` (both ops). This is Phase A (§8).
- **Class E — function absent from that version's export.** Confirm in the *IDB* whether
  the function exists (unnamed → name it + surgical **absent-only** splice per playbook
  §10, then Class A) or is genuinely absent (→ justified `n-a`). **Candidates (export
  grep = 0):** `CheckName` v83/v84, `CharacterExpression` v83/v84
  (`OnEmotion`), `CharacterChairShow` v83/v84/jms (`OnSetActivePortableChair`). All three
  are **verified on v87/v95**, proving the feature exists in the client → the v83/v84/jms
  IDBs hold them unnamed; name-and-splice, not `n-a` (§5).

---

## 5. Export-presence evidence (why Class E is name-and-splice, not n-a)

`grep` of `docs/packets/ida-exports/` for the in-scope functions (counts; `Y`=present):

| fname (packet) | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|
| `OnSelectWorldResult` (CharacterList) | Y | Y | Y | Y | Y |
| `OnViewAllCharResult` (ViewAll) | Y | Y | Y | Y | Y |
| `OnCreateNewCharacterResult` (AddEntry) | Y | Y | Y | Y | Y |
| `OnTemporaryStatSet` / `OnCharacterInfo` / `OnUserEnterField` / `OnMove` / `OnAvatarModified` / `OnEffect` / `OnSetTemporaryStat` | Y | Y | Y | Y | Y |
| `SendNewCharPacket` / `SendDeleteCharPacket` / `SendAbilityUpRequest` / `SaveFuncKeyMap` | Y | Y | Y | Y | Y |
| `SendStatChangeRequest` (HealOverTime) | Y | Y | Y | Y | **Y(1)** |
| `SendCheckDuplicateIDPacket` (CheckName) | **0** | **0** | Y | Y | Y |
| `OnEmotion` (CharacterExpression) | **0** | **0** | Y | Y | Y |
| `OnSetActivePortableChair` (CharacterChairShow) | **0** | **0** | Y | Y | **0** |

Every in-scope function is present **except** the three Class-E functions on v83/v84 (and
chair on jms). Because each of those packets is **already ✅ on v87/v95**, the feature
demonstrably exists in the client; their absence is from the *export*, not the binary.
Per playbook §10 and the login sibling's Class-E handling, the executor opens the version
IDB, confirms the function is present-but-unnamed, **names it** (byte-signature anchored)
and **surgically splices** the single absent entry into the export (never overwrites the
file), after which the cell becomes Class A. Only if the IDB genuinely lacks the function
is `n-a` recorded with the IDB-confirmed justification — the bar is "produce it if I can,"
not "document a gap."

---

## 6. The `candidatesFromFName` fan-out is already wired

`tools/packet-audit/cmd/run.go` already contains a case for every in-scope character op,
including the `#suffix` splits (verified by grep):

- `CUser::OnEffect` (EffectQuest), `CUserRemote::OnAvatarModified` (AppearanceUpdate)
- `CUserRemote::OnSetActivePortableChair` (chair delegate → Decode4 chairId),
  `CUser::OnEmotion` (expression delegate → Decode4 expressionId)
- `CLogin::OnSelectWorldResult` (CharacterList),
  `CLogin::OnViewAllCharResult#{CharacterViewAllCount,CharacterViewAllCharacters,CharacterViewAllSearchFailed}`
- `CFuncKeyMappedMan::SaveFuncKeyMap` (+ pet-consume variants on the same op 0x9F),
  `CWvsContext::SendAbilityUpRequest#{DistributeAp,AutoDistributeAp}`,
  `CLogin::SendCheckDuplicateIDPacket`

So **no `run.go` change is expected** — the campaign produces reports/evidence/markers
against an already-correct linkage table. If the plan finds a missing `#suffix` case it is
a wiring bug fixed in its own commit; the audit found none missing for the 47 cells.

---

## 7. Architecture: phased multi-stage pipeline

Because the live-IDA surface is mixed (a fresh-decompile group + a Class-E cluster + ❌
adjudications, with a Class-A bulk that needs no IDA), the campaign runs in three internal
phases, each a fan-out of one-cell-per-agent verification, grouped by IDB so the shared
global IDA `select_instance` is never interleaved across versions.

**Phase A — fully-unverified T1 group (Class C, highest risk first per PRD).**
`CharacterList`, `CharacterAppearanceUpdate`, `EffectQuest` (both ops). For each, per
version: decompile the client read order at the per-version opcode, write a brand-new
`*_test.go` cross-version byte-fixture (stacked per-version markers over a table-driven
test, matching the repo idiom), pin evidence, generate the report, regen matrix. These
exercise the full nested avatar/look + effect bodies. `CharacterList`'s ❌ verdicts
(v83/v87/v95) are adjudicated here (stale-vs-real, §9). Grouped by IDB: do all of a
version's Phase-A functions in one IDB session.

**Phase B — Class-E name-and-splice cluster.** `CheckName` v83/v84,
`CharacterExpression` v83/v84, `CharacterChairShow` v83/v84/jms. Per IDB: confirm the
function present-but-unnamed → name (byte-signature anchored) → absent-only splice into
the export → then Class-A report-gen + fixture/marker/evidence. The v87/v95 verified
fixtures give the expected read order. Only record `n-a` if the IDB genuinely lacks it.

**Phase C — deterministic report-gen sweep (Class A, no IDA) + Class-B ❌ adjudication.**
The jms holes and the remaining `KeyMapChange` versions whose functions are present:
run root report-gen, copy `Character<Struct>.{json,md}`, ensure marker/evidence, regen
matrix. The Class-B ❌ cells (`CharacterMovement` v84/jms, `AutoDistributeAp` v84 ×2,
`CharacterViewAllCharacters` v84 🚫) are decompiled in their version IDB and adjudicated;
v84 body is byte-identical to v83 below ~0x3D (project memory:
`bug_majorversion_gt83_is_off_by_one_v87`), so v84 ❌s are most likely stale verdicts that
clear on regeneration — but the executor reads each client, never assumes identity.

**IDB grouping (never interleave; confirm the loaded version via `list_instances`, never
hardcode the port — the PRD's port list is a hint):**

- **v83 IDB:** CharacterList, AppearanceUpdate, EffectQuest×2 (Phase A); CheckName,
  Expression, ChairShow name+splice (Phase B); KeyMapChange report-gen.
- **v84 IDB:** same Phase-A set; CheckName/Expression/ChairShow (Phase B);
  CharacterMovement, AutoDistributeAp×2, ViewAll ❌ adjudication (Phase C).
- **v87 / v95 IDB:** CharacterList, AppearanceUpdate, EffectQuest×2 (Phase A) — the only
  in-scope work on these versions (everything else is already ✅).
- **jms IDB (`*_U_DEVM` build, not the SMC retail dump — playbook §10):** CharacterList,
  AppearanceUpdate, EffectQuest×2 (Phase A); ChairShow name+splice (Phase B); the large
  Class-A jms-hole sweep (AddEntry/Buff/Info/Spawn/BuffForeign/Create/Delete/Heal/
  CheckName/ViewAll/Movement/KeyMapChange) (Phase C).

**Commit granularity.** One commit per `packet × version` cell carrying its coupled
artifacts (report + marker + evidence, plus any export splice or codec fix), plus the
regenerated `STATUS.md`/`status.json`. The fully-unverified Phase-A packets may land all
five versions of one packet in a single commit (one new table-driven test file), which is
still atomic per playbook. Sequence commits so each `status.json` regen reflects exactly
the cells landed so far (the matrix files are shared state).

---

## 8. `matrix --check` exit-code bar

Per playbook §8, `matrix --check` exits 1 from a pre-existing 🟥 registry-seed conflict
backlog unrelated to character. The acceptance bar is therefore **"no new problems," not a
clean exit 0**:

- Zero orphan / dangling / stale / drift lines mentioning any `character/*` packet.
- The global conflict count must **not increase**.
- Every character cell in scope reads ✅ (or IDB-justified `n-a`) after regen.
- `fname-doc --check` and `operations --check` introduce no new failures.

A net decrease (clearing character-specific lines) is a bonus, not required.

---

## 9. If a Class-B/C decompile reveals a real wire delta (fix-first path)

A `❌` that turns out to be a genuine read-order divergence (an inserted field, changed
width, different guard) is a **wire bug in the character codec for that version**, not a
verification-only change. Per PRD non-goals ("surface, don't silently patch") and playbook
§4/§6:

1. **Surface it** in the design log / PR description — never silently re-pin ❌ → ✅.
2. Land the **codec fix first** as its own commit in `libs/atlas-packet/character/`
   (add the version branch to the decoder/encoder; update the cross-version byte-test to
   expect the divergence), with its own review.
3. Then run the verification pipeline for that cell against the corrected codec.

No delta is *expected* for the v84 cells (v84 body ≡ v83 below ~0x3D); jms is the most
plausible source of a real delta (jms structure genuinely differs) and is read carefully.
The pipeline must not *assume* identity — it reads each client.

---

## 10. Service / module impact

- **`libs/atlas-packet`** — the only Go module touched. New `*_test.go` files
  (`list_test.go`, `appearance_update_test.go`, `effect_quest_test.go`) and added
  per-version markers/fixtures on existing tests; a `*.go` codec changes **only if** a
  Class-B/C decompile proves a real wire delta (§9). Verification: `go test -race ./...`,
  `go vet ./...`, `go build ./...` clean in `libs/atlas-packet`.
- **No `go.mod` is touched** → per CLAUDE.md the `docker buildx bake` gate is **conditional
  on a `go.mod` change and does not apply**. `tools/redis-key-guard.sh` from repo root
  stays clean (character codecs touch no Redis).
- **`docs/packets/`** — per-version audit reports, evidence records, the Class-E export
  splices (only where §5 finds the function present-but-unnamed), and regenerated
  `STATUS.md`/`status.json`.
- **`tools/packet-audit`** — used as-is; **no `run.go` change expected** (§6).
- **Production handlers** (`atlas-login`, `atlas-channel`) — unchanged unless §9 fires.

---

## 11. Verification plan (acceptance mapping)

| PRD Acceptance Criterion | How this design satisfies it |
|---|---|
| Every `character` row ✅ or justified `n-a` for all five versions; no `incomplete` | §2 enumerates all 47 cells; §7 phases promote each; §5 handles the Class-E cluster as name-and-splice (n-a only if IDB-confirmed absent) |
| Each duplicated-path row verified per distinct fname | §1.1 + §2 op-keyed enumeration — two `EffectQuest` ops, two `AutoDistributeAp` ops, KeyMapChange op + sub-struct each get their own fixture/evidence/report |
| Each promoted cell has a `packet-audit:verify` fixture + fresh evidence (serverbound: + REPORT) committed together | §3 promotion mechanism; §7 commit granularity (coupled artifacts per cell) |
| `matrix --check` / `fname-doc` / `operations` exit cleanly | §8 "no new problems" bar |
| Affected module test/vet/build clean; bake for touched `go.mod` | §10 — only `libs/atlas-packet` test files change; no `go.mod` touched → no bake required |

**Definition of done:** all 47 character cells read ✅ (or IDB-justified `n-a`) in
regenerated `STATUS.md`; each promotion's report + marker + evidence committed atomically;
`libs/atlas-packet` green on `go test -race`/`vet`/`build`; `matrix --check` introduces no
new character-related problems and the conflict count does not increase.

---

## 12. Alternatives considered

1. **Treat every cell as a fresh decompile (rejected — wasteful).** Most jms/keymap cells
   are missing only the report, whose generation is deterministic from the present export
   with no IDA. Phase C reserves IDA for the ❌ and Class-E cells.
2. **Record the Class-E cells as `n-a` (rejected).** They are verified on v87/v95 — the
   feature exists in the client; the function is unnamed in the older/jms IDBs. Forcing
   `n-a` would under-cover a packet that demonstrably ships. §5 names-and-splices; `n-a`
   is reserved for IDB-confirmed genuine absence.
3. **Silently re-pin a ❌ as ✅ after regen (rejected).** A ❌ may encode a real wire delta
   (jms is the prime suspect). Each ❌ is adjudicated by decompile; a real delta takes the
   fix-first path (§9), surfaced not hidden.
4. **One test function per version vs stacked markers (chosen: stacked markers).** The
   repo idiom stacks per-version `packet-audit:verify` lines above one table-driven test
   asserting cross-version byte-equality (existing `buff_give_test.go`, `create_test.go`).
   New Phase-A files follow this; per-version test functions duplicate the loop for no
   grading benefit.
5. **Defer the fully-unverified group to a follow-up task (rejected).** PRD default is one
   task, phased internally; §7 Phase A front-loads it so the highest-risk work is done
   first, not deferred. CLAUDE.md "no deferring producible work" applies.

---

## 13. Risks & mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| A v84 ❌ is a real wire delta, not stale | Low (v84 body ≡ v83 below ~0x3D) | §9 fix-first; decompile before re-pinning; cross-version test updated to expect any divergence |
| A Class-E function is genuinely absent (not unnamed) in an IDB | Low (verified on v87/v95) | §5 — confirm in IDB; record `n-a` with justification only if truly absent |
| Export splice corrupts a committed export | Low (only §5 cells splice) | Absent-only surgical splice; never run full `export` over a committed file; diff before commit (playbook §10) |
| One of the two `EffectQuest`/`AutoDistributeAp` or the KeyMapChange sub-struct rows silently collapsed | Medium | §2 op-keyed enumeration; plan lists each row; report filename via `qualifiedWriterName` |
| Large bodies (List/Appearance/Info/Spawn) verified only on length, not full body | Medium | §3 — full-body byte fixture exercising the nested avatar/look block; no enumeration shortcut |
| jms retail IDB is SMC / undecompilable | Low | Use the clean `*_U_DEVM` jms build (playbook §10) |
| `matrix --check` pre-existing conflicts mask a new character regression | Low | §8 bar checks character-specific lines + conflict-count delta, not raw exit code |
| Template filename assumption wrong (jms is `template_jms_185_1.json`) | Low | §3 step 4 — resolve the exact template filename per version from disk |
| `COutPacket`-delegate harvest artifact blocks a spliced report | Low | Strip the delegate ctor call from the spliced entry (playbook §10) |

---

## 14. Open questions — resolved or scoped for the plan

- **CharacterList latent or emitted?** → Emitted (`writer/character_list.go` +
  `handler/character_list_world.go`); `incomplete` only for lack of a fixture. Phase-A
  fresh decompile (§1.1, §7).
- **Duplicated rows** → distinct ops sharing a packet path; enumerated op-keyed in §2,
  wired in run.go (§6); each gets its own fixture/evidence/report.
- **Which service owns each packet?** → Consumer split only (login lifecycle vs channel
  in-field); all codecs live in `libs/atlas-packet/character/`, the single changed module
  (§1.1).
- **Genuine `n-a` vs real gap** → only the Class-E cluster (§5) could yield `n-a`, and only
  if an IDB confirms genuine absence; the v87/v95 ✅ siblings make name-and-splice the
  expected outcome. No cell is pre-decided as `n-a`.

No blocking open questions remain for the planning phase.
