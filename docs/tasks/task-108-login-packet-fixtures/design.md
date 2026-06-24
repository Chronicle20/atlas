# Login Packet-Fixture Verification Campaign — Design

Task: task-108-login-packet-fixtures
Phase: 2 (Design)
Created: 2026-06-23
Status: Draft for review

---

## 1. Problem & Framing

The `login` family in the coverage matrix (`docs/packets/audits/STATUS.md`) sits at
~74% verified with **20 `incomplete` cells across 12 rows** (confirmed from
`status.json`, not estimated). The goal is to drive every `incomplete` login cell to
`verified` (✅) — or to a *justified* `n-a` where the opcode is version-absent — landing
each promotion as the coupled artifacts the playbook
(`docs/packets/audits/VERIFYING_A_PACKET.md`) requires.

The PRD framed this as a *scattered-holes, port-a-verified-sibling* campaign. That
framing is correct in spirit, but investigation surfaced three facts that materially
reshape the work and are documented below before any cell is touched:

1. **The work is serverbound-dominant** (16 of 20 cells), so the §9–10 serverbound rules
   — REPORT + evidence + marker, all three agreeing, op routed in the template — govern
   most of the campaign, not the simpler clientbound shape that drove task-107 (door).
2. **The dominant missing artifact is the per-version audit report**, not a fixture and
   not an IDA harvest. Unlike door, the client functions are *present* in the committed
   exports for almost every cell, so most reports regenerate deterministically with **no
   live IDA** (§4.1, §5).
3. **One client send-function fans out to several matrix rows** via the
   `candidatesFromFName` `#suffix` mechanism, and **every login case is already wired**
   in `tools/packet-audit/cmd/run.go` (§3). No new `candidatesFromFName` case is needed —
   a real simplification versus the playbook's general "add a case for a new serverbound
   op" rule.

### 1.1 Corrections to PRD assumptions (resolved from source)

- **Writer/codec ownership (PRD §7 said "atlas-login").** The login wire codecs and
  their fixture tests live in **`libs/atlas-packet/login/{clientbound,serverbound}/`**,
  not in `services/atlas-login`. The serverbound decoders
  (`character_select.go`, `all_character_list_select*.go`, `server_status_request.go`,
  `server_list_request.go`, …) and their `*_test.go` fixtures are the files this campaign
  touches. `services/atlas-login` (and `atlas-channel`) only *consume* these codecs. The
  changed Go **module is `libs/atlas-packet`**.
- **PRD `ServerListRequest — [v83, v84]` is a `sub-struct` row, not an `op` row.** In
  `status.json` its `kind` is `sub-struct` with `opcode: -1` and no `fnames`; it links via
  `CLogin::ChangeStepImmediate` → `ServerListRequest` (run.go). It still needs the same
  three serverbound artifacts; it simply has no dispatch opcode of its own.
- **PRD `AllCharacterListRequest (T1)` — `status.json` confirms `tier1: true`** for that
  row (op `VIEW_ALL_CHAR`, `CLogin::SendViewAllCharPacket`). It is the only `tier1:true`
  serverbound login row in scope; the rest are `tier1:false`. `status.json` is
  authoritative wherever it and the PRD disagree.

---

## 2. The 20 cells, enumerated by (packet, op, fname)

Every cell is keyed by its **distinct op** even where the packet path repeats — this
directly answers PRD Open Question 1 (the duplicated `CharacterSelect` /
`AllCharacterListSelect` rows are *distinct ops sharing one packet path and one client
send-function*, each a separate matrix row needing its own fixture/evidence/report).

### Clientbound (4 cells)

| Packet | op | fname (`#`-split) | Incomplete versions | status.json note |
|---|---|---|---|---|
| `AuthLoginFailed` | `LOGIN_STATUS` | `OnCheckPasswordResult#AuthLoginFailed` | v83, v84 | v83 "marker present, no fresh evidence"; v84 "verdict ❌" |
| `ServerStatus` | `SERVERSTATUS` | `OnCheckUserLimitResult` | v84 | "verdict ❌" (jms = n-a) |
| `ServerListEnd` | `WORLD_INFORMATION` | `OnWorldInformation#ServerListEnd` | jms | "verdict ❌" |

### Serverbound (16 cells)

| Packet | op | fname (`#`-split) | Incomplete | tier1 | note |
|---|---|---|---|---|---|
| `ServerStatusRequest` | `SERVERSTATUS_REQUEST` | `SendCheckUserLimitPacket` | jms | F | "no audit report" — **see §6 (n-a question)** |
| `AllCharacterListRequest` | `VIEW_ALL_CHAR` | `SendViewAllCharPacket` | v83 | **T** | "marker present, no fresh evidence" |
| `ServerListRequest` (sub-struct) | — | `ChangeStepImmediate` | v83, v84 | F | "no audit report" |
| `CharacterSelect` | `CHAR_SELECT` | `SendSelectCharPacket` | v84, jms | F | "tier-1 without fixture; 🔍" |
| `CharacterSelect` | `REGISTER_PIC` | `SendSelectCharPacket#CharacterSelectRegisterPic` | v84, jms | F | "🔍" |
| `CharacterSelect` | `CHAR_SELECT_WITH_PIC` | `SendSelectCharPacket#CharacterSelectWithPic` | v84, jms | F | "🔍" |
| `AllCharacterListSelect` | `PICK_ALL_CHAR` | `SendSelectCharPacketByVAC#AllCharacterListSelect` | v84, v87 | F | v84 "❌"; v87 "no report" (jms n-a) |
| `AllCharacterListSelect` | `VIEW_ALL_PIC_REGISTER` | `SendSelectCharPacketByVAC#AllCharacterListSelectWithPicRegister` | v84, v87 | F | v84 "❌"; v87 "no report" |
| `AllCharacterListSelect` | `VIEW_ALL_WITH_PIC` | `SendSelectCharPacketByVAC#AllCharacterListSelectWithPic` | v84, v87 | F | v84 "❌"; v87 "no report" |

(The plan phase MUST enumerate each row by op-keyed fname so none of the three
`CharacterSelect` / three `AllCharacterListSelect` rows is silently collapsed. The exact
`qualifiedWriterName` — and therefore the report filename and marker `packet=` path — is
`TitleCase(pkg)+structName`, e.g. struct `AllCharacterListSelectWithPic` in pkg `login`
→ report `LoginAllCharacterListSelectWithPic`, per run.go's `qualifiedWriterName`.)

---

## 3. The `candidatesFromFName` fan-out is already wired (PRD Open Question 1, resolved)

`tools/packet-audit/cmd/run.go` (≈L520–575) already contains a `candidatesFromFName`
case for **every** login op in scope, including the `#suffix` splits that map one client
send-function to N distinct ops:

- `CLogin::SendSelectCharPacket` → `CharacterSelect` (base, `CHAR_SELECT`)
- `CLogin::SendSelectCharPacket#CharacterSelectRegisterPic` → `REGISTER_PIC`
- `CLogin::SendSelectCharPacket#CharacterSelectWithPic` → `CHAR_SELECT_WITH_PIC`
- `CLogin::SendSelectCharPacketByVAC#AllCharacterListSelect{,WithPic,WithPicRegister}` → the three VAC ops
- `CLogin::OnCheckPasswordResult#{AuthLoginFailed,AuthTemporaryBan,AuthPermanentBan}` (clientbound)
- `CLogin::OnWorldInformation#ServerListEnd`, `CLogin::SendCheckUserLimitPacket`,
  `CLogin::OnCheckUserLimitResult`, `CLogin::ChangeStepImmediate`, `CLogin::SendViewAllCharPacket`

The existing v83/v87/v95 markers on these tests (e.g. `character_select_byte_test.go`,
`all_character_list_select_test.go`) confirm the linkage works end-to-end today. So **no
`run.go` change is expected** — the campaign produces reports/evidence/markers against an
already-correct linkage table. (If the plan finds a missing `#suffix` case it is a wiring
bug to fix first, but the audit above found none missing for the 20 cells.)

The send function emits *different opcodes for the same call site* depending on
PIN/PIC/VAC client state; the `#suffix` is the disambiguator the analyzer uses to descend
to the right branch of the one decompiled function. Each branch = one packet/op = one
fixture + one evidence record + one report.

---

## 4. Promotion mechanism & per-cell work-classes

A login cell promotes when, for that `packet × version`, **all** of these exist and agree
(serverbound: §9; clientbound tier-0: marker + report; tier-1: + evidence):

1. A `// packet-audit:verify packet=login/<dir>/<Qualified> version=<v> ida=0x<addr>`
   marker stacked above the byte-test.
2. A per-version **audit report** `docs/packets/audits/<version>/Login<Struct>.{json,md}`
   — generated deterministically by the **root** `packet-audit` command against the
   committed export (§9 step 3; no live IDA). Its absence is the literal
   `"note":"no audit report"` / dangling-evidence failure on most incomplete cells.
3. A pinned **evidence record**
   `docs/packets/evidence/<version>/login.<dir>.<Struct>.yaml` (serverbound: always;
   clientbound: tier-1 only) with a `verifies:` line and an `ida.function` that may carry
   the `#suffix` (template: existing
   `gms_v84/login.serverbound.CharacterSelectWithPic.yaml`, whose function is
   `CLogin::SendSelectCharPacket#CharacterSelectWithPic`).
4. (Serverbound) the op is **routed** in that version's seed template
   (`services/atlas-configurations/seed-data/templates/template_<v>.json` —
   note the jms file is `template_jms_185_1.json`, not `template_gms_jms_185.json`; the
   plan must resolve the exact filename per version, not assume).

### 4.1 The four work-classes (drives plan ordering)

Cells fall into four classes by *what is actually missing*. Class A is the bulk and needs
no live IDA; classes B/D/E need a decompile and are grouped by IDB.

- **Class A — report-gen only (no live IDA).** Function present in the committed export,
  marker (and often evidence) already present; the cell is `incomplete` purely because the
  per-version **report** was never copied in. Action: run root report-gen, copy
  `Login<Struct>.{json,md}` into `docs/packets/audits/<v>/`, add marker/evidence if absent,
  regen matrix. **Candidates:** the v87 `AllCharacterListSelect` ×3 ("no report"); the v84
  `CharacterSelect` ×3 and `AllCharacterListSelect` ×3 where v84 evidence already exists;
  `ServerListRequest` v83/v84; `AllCharacterListRequest` v83; `AuthLoginFailed` v83.
  Confirm export-presence per cell before assuming Class A.
- **Class B — ❌ verdict, needs fresh decompile to adjudicate.** A `verdict ❌` means the
  generated report's read order *disagrees with the Atlas codec*. Decompile that version's
  client function and decide: **(i) stale/cosmetic** (codec actually matches → the ❌ was a
  pre-fix artifact; regenerate the report and re-pin), or **(ii) real wire delta** → take
  the wire-fix-first path (§7). **Candidates:** `AuthLoginFailed` v84, `ServerStatus` v84,
  `ServerListEnd` jms, `AllCharacterListSelect` v84 ×3.
- **Class D — 🔍 tier-1-without-fixture.** Marker/verdict exists but the byte-fixture is
  missing for that version. Write/extend the byte-test with the `#suffix` branch, add
  marker + evidence + report. **Candidates:** `CharacterSelect` v84/jms ×3.
- **Class E — function absent from export (n-a vs name-and-splice).** The fname is not in
  that version's export. Confirm in the *IDB* whether the function exists (unnamed →
  name + surgical absent-only splice per §10, then Class A) or is genuinely absent (→ the
  cell is **`n-a`**, recorded with justification, not forced to ✅). **Candidate:**
  `ServerStatusRequest` jms — see §6.

A cell's class is a *hypothesis from the note*; the plan/executor confirms it per cell
(export grep → report-gen → only decompile when the report says ❌ or the function is
absent). We do not pre-commit to a class without the export/report check.

---

## 5. Why login is cheaper than door (export presence)

`grep` of `docs/packets/ida-exports/` for the login send/recv functions:

| fname | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|
| `OnCheckPasswordResult` (AuthLoginFailed) | ✓ | ✓ | ✓ | ✓ | ✓ |
| `OnCheckUserLimitResult` (ServerStatus) | ✓ | ✓ | ✓ | ✓ | **absent** |
| `OnWorldInformation` (ServerListEnd) | ✓ | ✓ | ✓ | ✓ | ✓ |
| `SendCheckUserLimitPacket` (ServerStatusRequest) | ✓ | ✓ | ✓ | ✓ | **absent** |
| `SendViewAllCharPacket` (AllCharacterListRequest) | ✓ | ✓ | ✓ | ✓ | ✓ |
| `SendSelectCharPacketByVAC` (AllCharacterListSelect ×3) | ✓ | ✓ | ✓ | ✓ | **absent** |
| `SendSelectCharPacket` (CharacterSelect ×3) | ✓ | ✓ | ✓ | ✓ | ✓ |

Every **target** cell's function is present in the export **except the jms user-limit
pair** (both absent, both consistent with jms `ServerStatus` already being `n-a`). So,
unlike door (functions absent from four of five exports → harvest+splice was the dominant
cost), the login campaign is mostly **deterministic report-gen** (§9 step 3) with live IDA
reserved for the Class B ❌ adjudications and the single Class E jms question. This is the
central architectural difference from task-107 and it makes the live-IDA surface small
(one IDB session for the v84 ❌ cells, one for jms).

---

## 6. The jms `ServerStatusRequest` question (PRD Open Question 3 → likely `n-a`)

`grep -c` of `gms_jms_185.json` returns **0** for both `SendCheckUserLimitPacket` and
`OnCheckUserLimitResult`. The clientbound twin `ServerStatus` is **already recorded `n-a`
for jms**. Together this strongly suggests jms has no user-limit/server-status handshake
on the wire, which would make `ServerStatusRequest jms` an **`n-a`, not a real gap** —
the acceptance criterion explicitly allows `n-a` as a terminal state.

This is a *producible* check, not a documented gap: before recording `n-a` the executor
**must** open the jms `*_U_DEVM` IDB (§10: the retail jms dump is SMC; use the dev build)
and confirm `CLogin::SendCheckUserLimitPacket` is genuinely absent (not merely unnamed).
- If absent → record `ServerStatusRequest jms = n-a` with the IDB-confirmed justification
  in the evidence/status note, and the cell is *done*.
- If present-but-unnamed → name it (§10 byte-signature `6A <op> … E8`), surgically splice
  it into the jms export, then it becomes Class A.

This is the one cell with a genuine fork that the design cannot pre-decide without the
IDB; it is surfaced here and resolved by *trying*, per the "verify the blocker by trying"
rule.

---

## 7. If a Class B decompile reveals a real wire delta (fix-first path)

A `❌` that turns out to be a genuine read-order divergence (an inserted field, a changed
width, a different guard) is a **wire bug in the login codec for that version**, not a
verification-only change. Per PRD non-goals ("no behavior changes; surface, don't silently
patch") and playbook §4/§6:

1. **Surface it** in the design log / PR description — do not silently re-pin a ❌ as ✅.
2. Land the **codec fix first** as its own commit in `libs/atlas-packet/login/` (add the
   version branch to the decoder, update the cross-version byte-test to expect the
   divergence), with its own review.
3. Then run the verification pipeline for that cell against the corrected codec.

No delta is *expected* for v84 (memory: v84 packet structure is byte-identical to v83 —
`bug_majorversion_gt83_is_off_by_one_v87`, `bug_v84_opcode_table_shifted_vs_v83` affects
*opcodes* not *body layout* below ~0x3D), so the v84 ❌ cells are most likely **stale
verdicts** (Class B-i) that clear on report regeneration. The jms `ServerListEnd` ❌ is the
most plausible *real* delta (jms login structure genuinely differs) and should be read
carefully. The pipeline must not *assume* identity — it reads each client.

---

## 8. Architecture: two-stage pipeline

Because the live-IDA surface is small (§5), the campaign splits cleanly into a
deterministic stage and an IDA stage, rather than door's uniform per-IDB loop.

**Stage 1 — deterministic report-gen sweep (no IDA).** For every cell whose function is
present in the export (all but the jms user-limit pair), per version V:

1. Confirm the export contains the fname (`grep`); confirm the op is routed in
   `template_<V>.json` (else §7 / playbook §10 "wire the route").
2. Run the **root** `packet-audit` for V (csv-clientbound + csv-serverbound +
   `template_<V>.json` + `-ida-source ida-exports/<V export>.json`) to a temp `-output`.
3. Read the generated `Login<Struct>.json` verdict.
   - **✅** → copy `Login<Struct>.{json,md}` into `docs/packets/audits/<V>/`; ensure the
     marker exists (add if absent); pin/refresh the evidence record (serverbound always;
     clientbound tier-1 only — only `AllCharacterListRequest` is tier-1 here).
   - **❌** → escalate to Stage 2 (this cell needs a decompile, §4.1 Class B / §7).
4. Regenerate `STATUS.md`/`status.json`; confirm the cell flipped to ✅.

**Stage 2 — live-IDA cells, grouped by IDB.** Only the Class B ❌ cells that survived
Stage 1 and the Class E jms cell. Group by IDB (IDA `select_instance` is shared global
state — never interleave two versions, playbook §10):

- **v84 IDB** (confirm the loaded version via `list_instances`, never hardcode the port —
  the PRD's port list is a hint): decompile `OnCheckPasswordResult` (AuthLoginFailed),
  `OnCheckUserLimitResult` (ServerStatus), `SendSelectCharPacketByVAC` (the three VAC ops),
  and the `CharacterSelect` `#`-branches for the Class D v84 fixtures. Adjudicate
  stale-vs-real (§7). Splice only if an entry is missing/stale (absent-only; never
  overwrite a good entry).
- **jms `*_U_DEVM` IDB**: `OnWorldInformation#ServerListEnd` (❌ adjudication),
  `SendSelectCharPacket` `#`-branches (Class D jms fixtures), and the §6
  `SendCheckUserLimitPacket` n-a confirmation.

**Commit granularity.** One commit per `packet × version` cell carrying its coupled
artifacts (report + marker + evidence, plus any export splice or codec fix), plus the
regenerated `STATUS.md`/`status.json`. Sequence the commits so each `status.json` regen
reflects exactly the cells landed so far (the matrix files are shared state). This matches
the playbook's "commit the artifacts together" rule and keeps each promotion atomic and
reviewable.

---

## 9. `matrix --check` exit-code bar (same as door §7)

Per playbook §8, `matrix --check` exits 1 from a pre-existing 🟥 registry-seed conflict
backlog unrelated to login. The acceptance bar is therefore **"no new problems," not a
clean exit 0**:

- Zero orphan / dangling / stale / drift lines mentioning any `login/*` packet.
- The global conflict count must **not increase**.
- Every login cell in scope reads ✅ (or justified `n-a`) after regen.
- `fname-doc --check` and `operations --check` introduce no new failures.

(If the campaign happens to clear login-specific lines that were contributing to the
count, that is fine — the bar is "no new," and a net decrease is a bonus, not required.)

---

## 10. Service / module impact

- **`libs/atlas-packet`** — the only Go module touched. `*_test.go` files gain markers and
  (Class D) new `#suffix` byte-fixtures; a `*.go` codec changes **only if** a Class B
  decompile proves a real wire delta (§7). Verification: `go test -race ./...`,
  `go vet ./...`, `go build ./...` clean in `libs/atlas-packet`.
- **No `go.mod` is touched** → per CLAUDE.md the `docker buildx bake` gate is **conditional
  on a `go.mod` change and does not apply**. (`tools/redis-key-guard.sh` from repo root is
  unaffected and stays clean — login codecs touch no Redis.)
- **`docs/packets/`** — per-version audit reports, evidence records, the jms export splice
  (only if §6 finds the function present-but-unnamed), and regenerated
  `STATUS.md`/`status.json`.
- **`tools/packet-audit`** — used as-is; **no `run.go` change expected** (§3: every login
  `candidatesFromFName` case is already wired).
- **Production handlers** (`atlas-login`, `atlas-channel`) — unchanged unless §7 fires.

---

## 11. Verification plan (acceptance mapping)

| PRD Acceptance Criterion | How this design satisfies it |
|---|---|
| Every `login` row ✅ or justified `n-a` for all five versions; no `incomplete` | §2 enumerates all 20 cells; §8 pipeline promotes each; §6 handles the one likely `n-a` |
| Each duplicated-path row verified per distinct fname | §2 op-keyed table + §3 `#suffix` fan-out — three `CharacterSelect` + three `AllCharacterListSelect` rows each get their own fixture/evidence/report |
| Each promoted cell has a `packet-audit:verify` fixture + fresh evidence (serverbound: + REPORT) committed together | §4 promotion mechanism; §8 commit granularity (one cell = one commit, coupled artifacts) |
| `matrix --check` / `fname-doc` / `operations` exit cleanly | §9 "no new problems" bar |
| Affected module test/vet/build clean; bake for touched `go.mod` | §10 — only `libs/atlas-packet` test files change; no `go.mod` touched → no bake required |

**Definition of done:** all 20 login cells read ✅ (or IDB-justified `n-a` for jms
`ServerStatusRequest`) in regenerated `STATUS.md`; each promotion's report + marker +
evidence committed atomically; `libs/atlas-packet` green on `go test -race`/`vet`/`build`;
`matrix --check` introduces no new login-related problems and the conflict count does not
increase.

---

## 12. Alternatives considered

1. **Treat every cell as a fresh decompile (rejected — wasteful).** Most cells are
   missing only the report, whose generation is deterministic from the present export with
   no IDA. Decompiling all 20 would burn the shared IDA instance for cells that don't need
   it. Stage 1 / Stage 2 split (§8) reserves IDA for the ❌ and absent-fname cells.
2. **Force the jms `ServerStatusRequest` cell to ✅ (rejected).** Its function is absent
   from the export and its clientbound twin is already `n-a`; fabricating a fixture would
   be a false pass. The honest outcomes are IDB-confirmed `n-a` or name-and-splice (§6).
3. **Silently re-pin a ❌ as ✅ after regen (rejected).** A ❌ may encode a real wire delta
   (jms `ServerListEnd` is the prime suspect). Each ❌ is adjudicated by decompile; a real
   delta takes the fix-first path (§7), surfaced not hidden.
4. **One test function per version vs stacked markers (chosen: stacked markers).** The repo
   idiom (existing `character_select*` / `all_character_list_select*` tests) stacks
   per-version `packet-audit:verify` lines above a single table-driven test that already
   asserts cross-version byte-equality. Adding per-version test functions duplicates the
   loop for no grading benefit. New Class D fixtures extend the existing `#suffix` test,
   they don't fork it.
5. **Add `candidatesFromFName` cases defensively (rejected — unnecessary).** §3 confirmed
   every login op is already wired; adding cases would be dead code. If the plan finds a
   genuine gap it's a bug fixed in its own commit, not a blanket addition.

---

## 13. Risks & mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| A v84 ❌ is a real wire delta, not stale | Low (v84 body ≡ v83) | §7 fix-first path; decompile before re-pinning; cross-version test updated to expect any divergence |
| jms `ServerListEnd` genuinely diverges | Medium | §8 Stage 2 reads the jms client carefully; §7 if real |
| jms `SendCheckUserLimitPacket` is present-but-unnamed (not n-a) | Medium | §6 — name + surgical absent-only splice, then Class A; don't record `n-a` without the IDB check |
| Export splice corrupts a committed export | Low (only jms §6 may splice) | Absent-only surgical splice; never run full `export` over a committed file; diff before commit (playbook §10) |
| One of the three `CharacterSelect`/`AllCharacterListSelect` rows silently collapsed | Medium | §2 op-keyed enumeration; plan must list each `#suffix` row; per-row report filename via `qualifiedWriterName` |
| `COutPacket`-delegate harvest artifact blocks a report | Low | Strip the delegate ctor call from the spliced entry (playbook §10) |
| jms retail IDB is SMC / undecompilable | Low (login recv is simple) | Use the clean `*_U_DEVM` jms build (playbook §10) |
| `matrix --check` pre-existing conflicts mask a new login regression | Low | §9 bar checks login-specific lines + conflict-count delta, not raw exit code |
| Template filename/route assumption wrong (jms is `template_jms_185_1.json`) | Low | §4 step 4 — resolve the exact template filename per version from disk, don't assume |

---

## 14. Open questions — resolved or scoped for the plan

- **Duplicated `CharacterSelect`/`AllCharacterListSelect` rows** → distinct ops sharing one
  packet path and one client send-function, split by `candidatesFromFName` `#suffix`;
  already wired in run.go (§2, §3). Each needs its own fixture/evidence/report.
- **`AllCharacterListRequest` v83 / `ServerListRequest` v83-v84 — verified sibling to port
  from?** → Yes: both have ✅ siblings (v87/v95/jms for `AllCharacterListRequest`;
  v87/v95/jms for `ServerListRequest`), functions present in the v83/v84 exports → Class A
  report-gen (§4.1, §5). No fresh decompile expected.
- **Genuine `n-a` vs real gap for jms** → `ServerStatus` already `n-a`; `ServerStatusRequest
  jms` is the one cell whose `n-a`-vs-gap fork the design cannot pre-decide — resolved by
  the §6 IDB check during execution (the only genuine stop-and-ask candidate, and only if
  the IDB itself is missing).

No blocking open questions remain for the planning phase.
