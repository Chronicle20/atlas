# Task 108 — Execution Audit & Plan Corrections

## Material plan correction: demux op-rows are graded worst-of-candidates

**Discovered during execution (Task 1).** The plan's per-cell recipes and its
§C "verdict-clean rule" misdiagnose how the coverage matrix grades the login
op-rows. Verified against `tools/packet-audit/internal/matrix/{build,grade}.go`:

1. **Op-rows aggregate all writers sharing the op's base FName**
   (`worstCandidateCell`, build.go:261) and take the **worst** by state severity.
   The login op-rows are demux families:
   - `LOGIN_STATUS` → `CLogin::OnCheckPasswordResult` → **4 writers**:
     `AuthLoginFailed`, `AuthSuccess`, `AuthPermanentBan`, `AuthTemporaryBan`.
   - `PICK_ALL_CHAR` / `VIEW_ALL_WITH_PIC` / `VIEW_ALL_PIC_REGISTER`
     → `CLogin::SendSelectCharPacketByVAC` → **3 writers** (the three
     `AllCharacterListSelect*`). All three op-rows grade worst-of the same 3.
   - `CHAR_SELECT` / `REGISTER_PIC` / `CHAR_SELECT_WITH_PIC`
     → `CLogin::SendSelectCharPacket` → **3 writers** (the three
     `CharacterSelect*`). All three op-rows grade worst-of the same 3.

   **Consequence:** ALL writers in a family must individually grade Verified
   before ANY of the family's op-rows flips. The displayed cell *note* is the
   worst arm's note, not necessarily the writer the cell is named after. The
   plan named cells after one writer and prescribed fixing that writer; the real
   blocker is often a *sibling* arm.

2. **For a tier-1 / FlatInvalid writer the diff verdict is ADVISORY**
   (grade.go:195-211). Such a cell promotes on **marker + fresh evidence**, NOT
   on "FlatInvalid:false and every Verdict==0". The plan §C verdict-clean rule is
   wrong for the branchy writers (`AuthSuccess`, base `CharacterSelect`, etc.),
   whose flat positional diff is a known modeling limitation resolved by
   byte-level fixtures. A real `Verdict != 0` on a **tier-0** (FlatInvalid=false)
   report IS a real wire delta → decompile / fix-first, never pin over it.

### Corrected grading rule applied for the rest of the campaign

> An in-scope login op-row is Verified iff **every** writer sharing the op's base
> FName is individually Verified for that version. Per-writer: tier-0 needs
> `toolPass(verdict 0) + marker`; tier-1 (FlatInvalid OR tier1 packet) needs
> `marker + fresh evidence` (verdict advisory). `matrix --check` (baseline exit 0,
> 0 conflicts, 0 login lines) is the arbiter.

### Task-structure deviations from plan.md

- **Plan Tasks 3 + 4 merged** into one "v84 `SendSelectCharPacket` family" unit:
  the three CHAR_SELECT/REGISTER_PIC/CHAR_SELECT_WITH_PIC op-rows are worst-of the
  same 3 writers, so none flips until all three writers are verified. RegisterPic
  and WithPic already carry v84 marker+evidence; only the base `CharacterSelect`
  needs work (byte-fixture v84 row + marker + evidence).
- **Cell #1 (AuthLoginFailed gms_v83)** was promoted by pinning a fresh
  `AuthSuccess` **gms_v83** evidence record (the FlatInvalid sibling arm that
  was dragging the op-row), NOT by touching AuthLoginFailed. AuthSuccess maps to
  the base `CLogin::OnCheckPasswordResult` key (mirrors the existing gms_v84
  AuthSuccess record). Committed `925554c`.

## Export / fname deviations (grounded, documented)

- **v84 `ServerStatus` annotation fix (controller-verified, not a wire delta).**
  `CLogin::OnCheckUserLimitResult` reads 2×`Decode1` on BOTH v83 (@0x5f92ae) and
  v84 (@0x60e275); atlas writes one `WriteShort` = 2 bytes → wire-equivalent. The
  v84 export entry listed two literal `Decode1` ops (→ "width mismatch"); the
  verified v83 entry collapses them to one `Decode2` with a wire-equivalence
  comment. Fixed the v84 entry to mirror v83. Grounded by decompiling both.
- **v84 `ServerListRequest` — same inline as v83.** No discrete `ChangeStep`/
  `ChangeStepImmediate` symbol in v84; the bodyless `COutPacket(4)+SendPacket`
  lives in `sub_609165` (the v84 step-machine analog), in the `*(CWvsContext+8228)
  ==1` block. Spliced under the canonical name with the real v84 address + note;
  controller re-decompiled and confirmed bodyless opcode-4.
- **v83 `ServerListRequest` — `CLogin::ChangeStepImmediate` does not exist as a
  discrete symbol in the v83 IDB** (unlike v87/v95/jms). The immediate
  server-list-request send is **inlined into `CLogin::ChangeStep` @0x5f53c0**
  (`m_nBaseStep==1` block: `COutPacket(opcode 4)+SendPacket`, no Encode calls →
  bodyless). The export key was spliced under the canonical matrix logical name
  `CLogin::ChangeStepImmediate` (the `candidatesFromFName` mapping at run.go:546)
  with the **real** inline address and a `note` documenting the inline.
  Independently re-decompiled and confirmed (controller). This is a grounded
  unblock per "No Deferring Producible Work" — the fname was *found and verified*
  inlined, not fabricated — NOT a faked-hash escalation case.

## Wire deltas found

- **none through v83.** v83 `AllCharacterListRequest` (`SendViewAllCharPacket`
  @0x5fac34) sends opcodes 0xC/0xD with zero Encode calls → bodyless on v83; the
  atlas decoder gates all 5 field reads behind `MajorAtLeast(87)` → reads nothing
  on v83. The report's 5 "atlas: extra" verdict-2 rows are a flat-diff modeling
  artifact (analyzer models the v87 fields statically), confirmed by decompile —
  NOT a real over-read. Tier-1 advisory → promoted via marker + fresh evidence.
- jms `ServerListEnd` remains the prime real-delta suspect (Task 8).

## jms ServerListEnd / OnWorldInformation — resolved (not a wire delta)

The `WORLD_INFORMATION` op grades worst-of-writers under base FName
`CLogin::OnWorldInformation` (ServerListEnd + ServerListEntry). The blocker was
NOT the `#ServerListEnd` sentinel (that matched immediately — single 0xFF) but a
**mismodeled base export entry**: it had been authored "Wire layout identical to
GMS v95" and carried a spurious `Decode1(nBlockCharCreation)` between the two
`Decode2`s and `nChannelCount`. The jms decompile @0x66f107 reads, after the two
`Decode2`s, `v8 = Decode1` **directly as nChannelCount** — there is NO
blockCharCreation byte in jms. The atlas `server_list_entry.go` JMS encode
already omits blockCharCreation, so the atlas writer was correct; the export was
wrong. Removed the spurious Decode1 and corrected the notes → ServerListEntry
flips ❌→✅, ServerListEnd already ✅, op verified.

## jms ServerStatusRequest n-a (IDB-confirmed no SendCheckUserLimitPacket)

`func_query name_regex "SendCheckUserLimit|CheckUserLimit"` on the jms IDB
(MapleStory_dump_SCY.exe, port 13338) returns **zero matches** — there is no
`CLogin::SendCheckUserLimitPacket` send function, and `OnCheckUserLimitResult`
is absent too (`UserLimit|UserNumber|WorldStatus|ServerStatus` regex also
returns zero). The whole user-limit / world-population request-response flow is
absent in jms. The clientbound sibling `login/clientbound/ServerStatus`
(`CLogin::OnCheckUserLimitResult`) is already `n-a` for jms. The serverbound
`SERVERSTATUS_REQUEST` op was `Present` in the registry ONLY via a spurious
`provenance: csv-import` entry (opcode 5, fname `CLogin::SendCheckUserLimitPacket`).
Removed that entry from `docs/packets/registry/jms_v185.yaml` so the op is
`Absent` → `n-a` (opcode -1), mirroring the ServerStatus/VAC n-a siblings.
`matrix --check` stays exit 0 (no new dangling-opcode/conflict line).

## Backend-guidelines review

**Verdict: PASS.** Scope is genuinely test-only and clean. No DOM-* domain-logic
checks apply (no `model.go`/`processor.go`/`resource.go`/Kafka/REST changes); the
diff is confined to `libs/atlas-packet/login/{clientbound,serverbound}/*_test.go`.

### Scope confirmation
- `git diff --name-only main..HEAD -- '*.go' | grep -v _test.go` → **NO non-test
  Go files changed.** Every Go edit is a `_test.go` byte-fixture / marker-comment
  addition. The production codecs were not touched, so the immutable-model,
  processor, administrator, REST/JSON:API, and Kafka portions of the checklist
  are not triggered.
- No `*_testhelpers.go` / test-only constructor files introduced
  (`find login -name '*testhelper*'` → none). Test setup uses the existing
  `pt.CreateContext` / `pt.Variants` / `pt.RoundTrip` idiom — consistent with the
  Builder/shared-helper convention.

### Build / vet / race gate (run in worktree module)
- `go vet ./login/...` → clean.
- `go test -race ./login/... -count=1` → `ok clientbound`, `ok serverbound`.

### New JMS v185 byte-fixtures are genuine (not tautological)
The three new `JMS v185` cases in
`libs/atlas-packet/login/serverbound/character_select_byte_test.go`
(lines 80-86, 124-129, 167-178) assert JMS-specific `want` bytes that omit
mac/hwid, distinct from the shared GMS `want`:
- `CharacterSelect`  JMS want = `le4(charId)` only — matches codec
  `character_select.go:47` (`if t.Region() == "GMS" && MajorVersion() > 12`),
  skipped for JMS.
- `CharacterSelectWithPic` JMS want = `lp("PIC") + le4(charId)` — matches
  `character_select_with_pic.go:53` (`if t.Region() == "GMS"`), skipped for JMS.
- `CharacterSelectRegisterPic` JMS want = `{0x01} + le4(charId) + lp("PIC")` —
  matches `character_select_register_pic.go:58` (GMS-gated mac/hwid; pic written
  unconditionally), skipped for JMS.

Each JMS case calls the **real production** `in.Encode(l, pt.CreateContext("JMS",
185, 1))(nil)`. `pt.CreateContext` (test/context.go:34) builds a real
`tenant.Create(...)` stored via `tenant.WithContext`; the codec reads it through
`tenant.MustFromContext` / `t.Region()`. Not a mock.

**Mutation check (empirical, reverted):** temporarily relaxing the `Encode`
GMS guard at `character_select.go:47` so the JMS path would emit mac/hwid caused
**only** `TestCharacterSelectByteOutput/JMS_v185` to FAIL while every GMS subtest
still PASSED — proving the fixture pins the JMS-specific wire shape and would
catch a codec regression. File restored; `git status` on the codec is clean.

### Marker-only additions
The remaining nine edited files add `// packet-audit:verify ... version=...`
marker comments (plus, in `character_select_byte_test.go`, a `GMS v84` row and
the three JMS subtests) to round-trip tests that already iterate `pt.Variants`.
`pt.Variants` (test/context.go:18-32) includes `JMS v185`, `GMS v84`, `GMS v87`,
so the newly-marked versions actually execute. `pt.RoundTrip`
(test/roundtrip.go:21-33) is a real encode→decode round-trip under a real tenant
context that fails on unconsumed bytes — not a no-op.

### DOM-21 (shared-constant reuse)
No new `type`/`const` declarations in the test diff
(`git diff ... | grep -E '^\+.*\b(type|const)\b '` → none). The `le4`/`lp`
helpers are local LE byte-builders for fixtures, not redeclarations of any
`libs/atlas-constants` type. No violation.

### Blocking
- None.

### Non-blocking
- None.

## Plan-adherence review

**Verdict: PASS — READY_TO_MERGE.** All 20 in-scope cells reached `verified`
or justified `n-a`; every documented deviation is grounded in real decompiled
behavior; no production codec was changed; every gate passes. The material
"demux worst-of-candidates" plan deviation is correct and well-evidenced.

### 1. All 20 in-scope cells reached verified / justified n-a (verified from status.json)
Read `docs/packets/audits/status.json` directly (not prose). Login family totals:
**112 verified, 13 n-a, 0 incomplete.** Each of the 20 §C cells confirmed by
(packet, op, version) lookup:
- 19 → `verified` (the AuthLoginFailed×2, ServerStatus v84, ServerListEnd jms,
  AllCharacterListRequest v83, ServerListRequest v83/v84, AllCharacterListSelect
  ×3 v84 + ×3 v87, CharacterSelect ×3 v84 + ×3 jms).
- 1 → `n-a` (`serverbound/ServerStatusRequest jms_v185`), IDB-justified.

### 2. Each deviation in audit.md is grounded (no faked hashes / fabricated fnames)
- **v83 `ServerListRequest` inline splice** — export adds
  `CLogin::ChangeStepImmediate @0x5f53c0` with `calls: []` and a note that v83
  inlines the bodyless `COutPacket(4)+SendPacket` into `CLogin::ChangeStep`
  (no discrete symbol, unlike v87/v95/jms). Grounded: the marker
  `version=gms_v83 ida=0x5f53c0` matches the spliced address; the production
  `server_list_request.go` decoder is bodyless, consistent with the inline.
  This is a *found-and-verified* inline, not a fabricated fname — correct per
  "No Deferring Producible Work."
- **v84 `ServerListRequest`** — spliced `@0x609165` (`sub_609165`); marker
  `version=gms_v84 ida=0x609165` matches. Consistent.
- **v84 `ServerStatus` annotation fix** — export entry changed from two literal
  `Decode1` to one `Decode2` w/ wire-equivalence note, mirroring the verified v83
  entry. atlas writes one `WriteShort` (2 bytes) → wire-equivalent. Not a wire
  delta; export annotation corrected. Grounded by the cited v83@0x5f92ae /
  v84@0x60e275 addresses.
- **v87 VAC splice** — adds the three `SendSelectCharPacketByVAC#AllCharacterListSelect{,WithPic,WithPicRegister}`
  suffix keys with dispatch discriminators (switch case 3/1/0) @0x62ee37; markers
  match. Surgical absent-only (git diff shows only the 3 keys + nothing else).
- **jms `ServerListEnd`/`ServerListEntry`** — export base entry corrected to
  REMOVE a spurious `Decode1(nBlockCharCreation)` (jms @0x66f107 reads the second
  `Decode1` directly as `nChannelCount`). Cross-checked production
  `server_list_entry.go`: it has an explicit `else if t.Region() == "JMS"` branch
  (lines 64/105) that already omits blockCharCreation → the atlas writer was
  right, the export was wrong. ServerListEntry flips ❌→✅; codec untouched.
- **jms `CharacterSelect#WithPic`/`#RegisterPic` splices** — notes assert JMS
  `NO mac/hwid (differs from GMS)`. Cross-checked the byte-fixtures (below) and
  the production codec's `if t.Region() == "GMS"` guards — they agree exactly.
- **jms `ServerStatusRequest` n-a** — removed a *spurious* `provenance: csv-import`
  registry entry (opcode 5, `SendCheckUserLimitPacket`) from
  `docs/packets/registry/jms_v185.yaml`; audit.md records `func_query` returned
  zero matches for the user-limit flow in the jms IDB. Mirrors the already-n-a
  clientbound `ServerStatus` jms sibling. `matrix --check` stays exit 0 → no new
  dangling-opcode line, confirming the removal is internally consistent.
- **Evidence `decompile_sha256` hashes** are `evidence pin` outputs (cannot be
  hand-fabricated without the live IDB); the gms_v83 AuthSuccess record mirrors
  the pre-existing gms_v84 AuthSuccess shape exactly. The machine arbiter
  (`matrix --check` exit 0) validates none are stale/dangling.

### 3. No production codec (.go non-test) files changed (verification-only preserved)
`git diff --name-only main..HEAD | grep -E 'login/(client|server)bound/[a-z_]+\.go$' | grep -v _test.go`
→ **empty.** Every Go edit is a `_test.go` marker/fixture addition. Confirmed no
wire deltas required a codec fix — all adjudications resolved as stale-report /
mismodeled-export / inline-symbol, exactly as the plan's "no delta expected"
hypothesis predicted.

### 4. All gates pass (re-run in this audit)
| Gate | Result |
|---|---|
| `go build ./...` (libs/atlas-packet) | PASS (exit 0) |
| `go vet ./...` | PASS (exit 0) |
| `go test -race ./...` | PASS (exit 0) |
| `matrix --check` | **exit 0, zero output, no login lines, 0 conflicts** |
| `fname-doc --check` | exit 0 (`219 structs OK`) |
| `operations --check` | exit 0 (1 pre-existing absent-writer note: jms NoteOperation — unrelated to login) |

The final `matrix --check` is cleaner than the Task-0 baseline expected (the plan
anticipated a residual conflict backlog → exit 1; the actual matrix is fully
clean → exit 0). No new login problem line introduced; conflict count did not
increase. Acceptance bar (§E) met and then some.

### 5. Per-cell artifacts are coupled
Spot-checked commits: the v87 VAC commit (`c5470d2f8`) bundles 3 reports
(.json+.md) + 3 marker test edits + the export splice + regenerated
STATUS.md/status.json in one commit. The jms n-a commit (`6153e0aee`) bundles the
registry edit + status regen + audit.md note together. Coupling is intact.

### False-pass scrutiny (the prompt's specific risk)
The jms `CharacterSelect` cells were the prime false-pass candidate (new fixture
for a version whose wire shape differs). Verified the fixtures are **genuine, not
GMS-shaped copies**: `character_select_byte_test.go` asserts JMS `want` =
`le4(charId)` only (CHAR_SELECT, line 82), `lp(PIC)+le4(charId)` (WITH_PIC, 126),
`{0x01}+le4(charId)+lp(PIC)` (REGISTER_PIC, 172-174) — all omitting mac/hwid,
distinct from the GMS `want`. These exercise the **real** production `Encode`
under a real `pt.CreateContext("JMS",185,1)` tenant; the codec's pre-existing
`if t.Region() == "GMS"` guards produce exactly those bytes; `go test -race`
passes. The export-splice notes and the fixture assertions are mutually
consistent. No false pass found.

### Plan deviations confirmed legitimate
- **Demux worst-of-candidates correction** (audit.md §1) — verified against the
  matrix's `worstCandidateCell` aggregation and the verified-sibling precedent
  (gms_v95 VAC cells carry report+marker and **no** evidence file, yet grade
  verified). The plan §C prescription to pin VAC/AuthLoginFailed evidence was
  unnecessary; omitting it matches the established verified pattern and the
  arbiter passes. Legitimate.
- **Plan Tasks 3+4 merged**, **cell #1 promoted via the AuthSuccess sibling arm**
  — both grounded in the demux-family grading and committed (`925554c`,
  `5c10675d6`). Legitimate.

### Blocking
- None.

### Recommendation
- **Plan Adherence:** FULL.
- **Recommendation:** READY_TO_MERGE.
