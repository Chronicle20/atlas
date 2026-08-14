# Skill Macro Version Coverage — Design

Task: task-226-skill-macro-version-coverage
PRD: [`prd.md`](./prd.md)
Status: Draft for review
Created: 2026-08-13

---

## 1. What the exploration changed

The PRD framed this task as "routing + byte-verification of an already-correct
codec." Reading the code disproved two of its premises. Both findings are load-bearing
for the architecture, so they lead.

### 1.1 There are TWO macro encoders, and they disagree on shout polarity

| Site | Direction | Shout | File |
|---|---|---|---|
| `character.SkillMacro.Encode` | clientbound | `WriteBool(!e.Shout)` | `libs/atlas-packet/character/skill_macro.go:47` |
| `character.SkillMacro.Decode` | serverbound | `shout := !r.ReadBool()` | `libs/atlas-packet/character/skill_macro.go:64` |
| `model.Macro.Encode` | clientbound | `WriteBool(m.shout)` | `libs/atlas-packet/model/macros.go:50` |

`character.SkillMacro.Encode` is **dead code**. Neither clientbound announce site
uses it — both build a `model.Macros` and pass `macros.Encode`:

- `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:368-373` (login)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/macro/consumer.go:82-85` (update)

So production reads the shout flag **inverted** (`SkillMacro.Decode`) and writes it
**upright** (`model.Macro.Encode`). On gms_83 today — a version with both directions
routed — a macro saved with shout enabled is stored with the opposite value of what the
client sent, or is re-displayed with the opposite value of what was stored. Exactly one
of the two polarities is right; the codebase cannot tell you which, and no test can catch
it because each encoder is exercised in isolation. The PRD's open question #2 is not a
curiosity — it is a live defect on the versions that already work.

This is what `bug_matrix_roundtrip_fixture_false_verify` looks like from the inside:
`skill_macro_test.go` round-trips `SkillMacro.Encode` → `SkillMacro.Decode`, and the
double inversion cancels out, so the test is green while the shipped pair is wrong.

### 1.2 The codec cannot be verified where it currently lives

`packet-audit`'s `locateAtlasFile` (`tools/packet-audit/cmd/run.go:3624-3659`) only walks
files whose path contains `/clientbound/` or `/serverbound/`. `libs/atlas-packet/character/skill_macro.go`
is in neither. Combined with `libs/atlas-packet/model/macros.go` (also neither), **no audit
report can ever be generated for either macro op** at their current locations, and §9 of
`docs/packets/audits/VERIFYING_A_PACKET.md` makes an audit report a hard requirement for a
serverbound ✅ ("evidence with no report → `matrix --check` dangling-evidence failure").

Relocation is therefore a precondition of the PRD's acceptance criteria, not a stylistic
preference.

### 1.3 The IDA exports do not carry either function

```
gms_jms_185.json   init=0 flush=0
gms_v48.json       init=1 (unresolved, no address) flush=0
gms_v61.json       init=1 (address 0x849bce, calls: null) flush=0
gms_v72.json       init=1 (address 0x92126b, calls: null) flush=0
gms_v79.json       init=1 (address 0x97311a, calls: null) flush=0
gms_v83/84/87/92/95 init=0 flush=0
```

`CMacroSysMan::FlushToSvr` appears in **zero** exports; `CWvsContext::OnMacroSysDataInit`
appears in four, always as a stub with `calls: null` or `unresolved: true`. Report
generation is deterministic off the export (§9), so every report this task needs is
currently ungeneratable. A harvest + surgical-splice stage (§10 of the playbook —
**never** a wholesale re-export, which drifts ~150 unrelated function keys) is the
critical path, and it is by far the largest cost in the task. The template edits, which
the PRD leads with, are the cheapest part.

### 1.4 The v72 registry fname is a v79 carryover

`docs/packets/registry/gms_v72.yaml:2562-2569` records `SKILL_MACRO` with
`fname: sub_6022DB` and `ida.address: 6175200` (= `0x5E39E0`). `0x5E39E0` is the
COutPacket-ctor harvest site named in that entry's own note, not the sender.
`0x6022DB` is the **v79** address (`gms_v79.yaml:3049-3056`, `ida.address: 6300379` =
`0x6022DB`). The v72 entry's fname was copied from v79 and does not name a v72 function.
The PRD's open question #4 resolves to: yes, this is a registry data error, and it is
in scope to correct.

### 1.5 Neither op is linked in `candidatesFromFName`

`grep -n Macro tools/packet-audit/cmd/run.go` → no hits. Per playbook §9 ("to verify a
NEW serverbound op you must add its primary fname as a `candidatesFromFName` case"),
both ops need cases added, and the two directions would collide on the flat,
writerName-keyed audit directory. There is precedent for the fix: the `reportName`
override field (`run.go:208-215`), added for exactly this — summon clientbound
`SummonMove` vs serverbound `Move` both deriving `SummonMove`.

---

## 2. Architecture

### 2.1 Codec placement — one struct per direction

```
libs/atlas-packet/character/clientbound/skill_macro.go   type SkillMacro  (Encode)
libs/atlas-packet/character/serverbound/skill_macro.go   type SkillMacro  (Decode)
```

Both derive `qualifiedWriterName("character", "SkillMacro")` = `CharacterSkillMacro`, so
the serverbound candidate sets `reportName: "CharacterSkillMacroHandle"` — matching the
template's handler binding name and mirroring the `SummonMoveHandle` precedent. Resulting
identities:

| | marker `packet=` | report file |
|---|---|---|
| clientbound | `character/clientbound/CharacterSkillMacro` | `CharacterSkillMacro.json` |
| serverbound | `character/serverbound/CharacterSkillMacroHandle` | `CharacterSkillMacroHandle.json` |

`libs/atlas-packet/character/skill_macro.go` is deleted. The two exported name constants
(`CharacterSkillMacroHandle`, `CharacterSkillMacroWriter`) move with their direction —
`CharacterSkillMacroHandle` to `character/serverbound`, `CharacterSkillMacroWriter` to
`character/clientbound` — matching how the other split packages carry their binding names.

### 2.2 `model.Macros` is absorbed, not wrapped

`libs/atlas-packet/model/macros.go` is deleted and its two call sites in atlas-channel
re-pointed at `character/clientbound.SkillMacro`. It has no other consumer in the tree
(`grep -rn "model.Macros" libs/ services/` returns only its own definition).

The alternative — keep `model.Macros` as the production encoder and have the clientbound
struct delegate to it — is rejected. It preserves the ability for the audited codec and
the shipped codec to drift apart, which is the precise failure mode this task exists to
close. A verified ✅ must sit on the bytes the server actually sends.

**This overrides the PRD's Service Impact row** ("`services/atlas-channel` — No source
change expected"). Two files change:
`kafka/consumer/session/consumer.go` and `kafka/consumer/macro/consumer.go`, each swapping
`packetmodel.NewMacros(...)` for the clientbound constructor. `socket/handler/character_skill_macro.go`
changes its import to `character/serverbound`. No `macro.Model` change, no REST/Kafka
contract change — Data Model section of the PRD holds.

### 2.3 Version gating

Gates live inline in the two files, expressed as
`t.IsRegion("GMS") && t.MajorAtLeast(N)` / `t.Region() == "JMS"` against the tenant from
context — the idiom at `libs/atlas-packet/field/serverbound/general.go:47`. Raw
`MajorVersion() > N` is banned (`bug_majorversion_gt83_is_off_by_one_v87`).

Per-version files (the `chat/clientbound/v79.go` pattern) are held in reserve: adopt them
only if FR-1 finds a version whose read order differs structurally, not merely in a field
width. A packet this small should not be split across files for a one-field delta.

The fields FR-1.2 must resolve, and the current (unverified) hypothesis for each:

| Field | Hypothesis | Source |
|---|---|---|
| count | `byte`, client-capped | `skill_macro.go:42` |
| name | length-prefixed ASCII | `skill_macro.go:45` |
| shout | 1 byte — **polarity contested** | §1.1 |
| skillId1..3 | `uint32` ×3 | `skill_macro.go:48-50` |

Every one is a hypothesis to confirm per version. Anything the decompile does not settle
is written **unknown** in `layout-derivation.md` and the corresponding cell does not
promote.

### 2.4 Decode bound

FR-1 will establish the client's macro capacity. The decoder clamps its loop to that
capacity rather than trusting the wire count — a hostile or corrupt count byte of 255
currently allocates and parses 255 entries. This is the NFR's "bound the decode loop"
made concrete; it costs one comparison.

### 2.5 n-a becomes machine-enforced

Add to `docs/packets/feature-families.yaml`:

```yaml
  skill_macro:
    - SKILL_MACRO
    - MACRO_SYS_DATA_INIT
```

The gate (`tools/packet-audit/cmd/na_consistency.go:15-24`) then refuses `SKILL_MACRO ⬜`
on any version where `MACRO_SYS_DATA_INIT` is ✅ unless a positive-absence entry exists in
`docs/packets/feature-na-evidence.yaml`. This converts FR-2.2 from a prose promise into a
CI failure — v61 in particular already has `MACRO_SYS_DATA_INIT` in its registry
(`gms_v61.yaml:613`, opcode 91) and a routed writer (`template_gms_61_1.json:2524`), so
once that cell verifies, the v61 `SKILL_MACRO ⬜` must either be proven or corrected. That
is the outcome the PRD wants; the family entry makes it non-optional.

Registering the family before the cells promote is deliberate: it will fail
`matrix --check` mid-task, which is the signal that the v48/v61 question is still open.

---

## 3. Alternatives considered

### A. Split + absorb + verify (recommended — the design above)

One struct per direction, one encoder, gates derived from IDA, templates last.
Cost: highest. Buys the only outcome where a ✅ means the shipped bytes were checked.

### B. Minimal-churn audit wrappers

Leave `character/skill_macro.go` and `model/macros.go` where they are; add thin wrapper
structs under `character/{client,server}bound/` purely so `locateAtlasFile` finds
something. The playbook explicitly permits an uncalled audit codec for shared-model
serverbound ops (§9, "the wrapper may be an uncalled audit codec").

Rejected. That allowance exists for a *shared* decoder with one production implementation.
Here there are two contradictory implementations, and a wrapper would verify one while
production runs the other — manufacturing precisely the false ✅ the PRD's fourth user
story is written against. It also leaves the shout defect in place on gms_83/84.

### C. Route first, verify later

Add the five template bindings, ship the player-visible fix in a day, verify in a
follow-up.

Rejected as the primary plan: it widens an unverified — and, per §1.1, internally
inconsistent — decoder from 4 tenant versions to 8. It stays on the table as a **fallback
only if IDA access blocks FR-1 entirely**, in which case the fallback ships the templates
plus a resolved shout polarity and leaves the matrix rows at ❌ with a recorded reason.
Taking that path is a stop-and-ask, not a unilateral downgrade.

### D. Per-version codec files vs. inline gates

Covered in §2.3: inline gates by default, per-version files only on structural divergence.

---

## 4. Work sequence

Ordering is driven by one rule from the playbook (§4): **divergence ⇒ wire fix first, own
commit.** Templates land last so no new tenant is exposed to a decoder that is still
under investigation.

1. **Harvest + splice the exports.** Per IDB, harvest `CMacroSysMan::FlushToSvr` and
   `CWvsContext::OnMacroSysDataInit` with `-prior-export "" -pending <roster> -descent-depth 12`
   to a temp file; splice only those entries (and absent-only helpers) into the committed
   export. Name unnamed senders in the IDB and `idb_save`
   (`feedback_name_idb_symbols_while_reversing`). Ten IDBs; resolve each session from
   `idb_list` by binary **name**, never by port.
2. **Derive layouts** → `layout-derivation.md`, one row per version × field, each citing
   an IDB function and decompiled line; unresolved fields marked **unknown**.
3. **Resolve the v48/v61 question** (FR-2) using the same decompiles. Outcome is either a
   `feature-na-evidence.yaml` entry (absent symbol *and* absent dispatch entry) or a
   registry op + template binding + full verification.
4. **Fix the registry**: v72 `SKILL_MACRO` fname/address (§1.4).
5. **Wire fix + relocation, own commit**: split the codec, absorb `model.Macros`, apply
   gates, apply the decode bound, re-point the three atlas-channel files.
6. **Link the tooling**: two `candidatesFromFName` cases, serverbound with
   `reportName: "CharacterSkillMacroHandle"`.
7. **Verify cell by cell** (`/verify-packet` per op × version): byte fixture with marker,
   evidence pin, generated audit report copied in, `matrix` + `matrix --check`.
8. **Route the templates** (FR-5): four handler entries + one writer entry, sorted
   insertion, `LoggedInValidator` + `fname` + `services`, matching the shape at
   `template_gms_83_1.json:875-883` / `:2840-2847`.
9. **Reconciliation doc** + guards + review.

Steps 1–3 are sequential; step 7 fans out per cell once its layout row exists.

---

## 5. Testing

**Byte fixtures** replace the round-trip as the unit of proof. Pattern:
`libs/atlas-packet/party/clientbound/invite_test.go` (`TestInviteByteOutput`) — a table
indexing `pt.Variants`, with per-variant expected bytes and an IDA-evidence comment above
each, and the marker:

```
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v83 ida=0x…
```

`pt.Variants` already covers every version this task needs: v48[7], v61[8], v72[9],
v79[10], v83[1], v84[5], v87[2], v92[11], v95[3], JMS185[4]. No table change required.

**Serverbound fixtures** feed hand-computed bytes into `Decode` and assert the resulting
model — not `Encode`-produced bytes, which is what makes the current test worthless.

**The round-trip test** may survive alongside the fixtures (FR-3.2), but only after the
double-inversion that currently masks the polarity bug is gone. Since the two directions
become two structs, a round-trip test now genuinely crosses the seam it is meant to guard.

**Regression pinning.** Capture v83/v84 bytes from `main` before step 5 and assert them
after. Caveat, stated plainly because it contradicts a PRD acceptance criterion: if IDA
proves production's clientbound polarity wrong, the v83/v84 clientbound bytes **will**
change, by design. The PRD's "byte-identical to `main`" criterion yields to the IDA
finding — that is the whole point of deriving from the client. Any such deviation is
recorded in `layout-derivation.md` and called out in the PR description, and the pinned
fixture is updated to the IDA-derived value rather than deleted.

---

## 6. Scope boundaries

- **v92 is not repaired broadly.** Its template carries 70 handlers against gms_95's 137,
  and its evidence directory holds 23 records against gms_95's 670. This task adds exactly
  the two macro bindings and verifies exactly the two macro cells. The wider v92 gap is
  real, is not this task, and is not created by this task.
- `template_gms_12_1.json` — untouched (not a matrix column).
- `ANTI_MACRO_*` — unrelated; the exports' `CWvsContext::OnAntiMacroResult` /
  `CUIAntiMacro::SetRet` entries are noise for this task, not targets.
- Live-tenant PATCH — out of branch. `live-tenant-reconciliation.md` is the deliverable.

## 7. Risks

| Risk | Mitigation |
|---|---|
| Export harvest drifts unrelated function keys | Surgical splice only, per playbook §10; never re-export wholesale |
| `CMacroSysMan::FlushToSvr` unnamed on legacy IDBs | Resolve from the opcode dispatch table, name it, `idb_save`; unnamed ≠ absent |
| Shout polarity unresolvable from decompile | Record **unknown**, do not promote the affected cells, stop and ask — a coin-flip here silently corrupts saved macros |
| Family entry fails `matrix --check` mid-task | Intended; it is the open-question tracker. Must be green before PR |
| `reportName` collision assumption wrong | Verified against `run.go:208-215` and the SummonMove precedent before the fixtures are written |

## 8. Open questions carried forward

All four of the PRD's open questions remain open and are answered by FR-1, with two
sharpened by exploration: question #2 (shout polarity) is now known to be a live
contradiction rather than a possible one (§1.1), and question #4 (the `sub_6022DB` fname)
is resolved as a confirmed registry error on v72 (§1.4).

One new question, for FR-1: does any version encode the macro name as fixed-width rather
than length-prefixed? `WriteAsciiString` is length-prefixed; a fixed-width version would
be a structural divergence and would trigger the per-version-file option in §2.3.
