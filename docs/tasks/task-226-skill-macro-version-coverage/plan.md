# Skill Macro Version Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the skill-macro codec into one struct per direction under `libs/atlas-packet/character/{clientbound,serverbound}/`, derive its per-version wire layout from the client IDBs, byte-verify both matrix rows to ✅, and bind the four missing handler opcodes plus one missing writer opcode in the seed templates so gms_87/92/95 and jms_185 players can save macros.

**Architecture:** Today two contradictory encoders exist (`character.SkillMacro.Encode` — dead, inverted shout — and `model.Macros.Encode` — shipped, upright shout), and neither lives on a path `packet-audit`'s `locateAtlasFile` will walk, so neither row can ever verify where it stands. The fix is structural: one clientbound struct that IS the shipped encoder, one serverbound struct that IS the shipped decoder, both under audited paths, both carrying gates derived from the IDBs rather than assumed. Verification and template routing follow the derived layout, never precede it.

**Tech Stack:** Go 1.24 workspace (`go.work`), `libs/atlas-packet` (immutable structs + `Encode`/`Decode` closures), `libs/atlas-socket` (`response.Writer` / `request.Reader`), `libs/atlas-tenant` (`tenant.MustFromContext`), `tools/packet-audit` (matrix/evidence/report generation), ida-pro-mcp for decompiles, JSON seed templates in `services/atlas-configurations/seed-data/templates/`.

**Spec:** [`design.md`](./design.md) (PRD: [`prd.md`](./prd.md))

## Global Constraints

- **Worktree.** All work happens in `.worktrees/task-226-skill-macro-version-coverage` on branch `task-226-skill-macro-version-coverage`. Never edit files in the main repo. Verify with `git rev-parse --show-toplevel` before any commit.
- **Grounding.** Every byte in a fixture, every opcode, every field width traces to a decompile line or a repo `file:line`. Anything unresolved is written **unknown** in `layout-derivation.md` and its cell does **not** promote. Never fill a gap with a plausible value (CLAUDE.md "Grounding & Honesty").
- **Version gates.** Use `t.IsRegion("GMS") && t.MajorAtLeast(N)` / `t.Region() == "JMS"` against `tenant.MustFromContext(ctx)`. Raw `MajorVersion() > N` is banned (`bug_majorversion_gt83_is_off_by_one_v87`). Reference idiom: `libs/atlas-packet/field/serverbound/general.go:44-58`.
- **Export hygiene.** Never overwrite a committed IDA export wholesale. Harvest to a temp file with `-prior-export "" -pending <roster> -descent-depth 12`, then splice ONLY the needed function entries (`VERIFYING_A_PACKET.md` §10).
- **IDA session resolution.** Resolve the session from `mcp__ida-pro__idb_list` by binary **name** and pass it as the `database` parameter. `select_instance(port)` is dead since task-138.
- **Version set (10 columns):** `gms_v48`, `gms_v61`, `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`. Exports live at `docs/packets/ida-exports/gms_v<NN>.json` and `gms_jms_185.json`.
- **Registry opcodes (authority — `feedback_verify_packets_not_cross_version_opcodes`):**
  - `SKILL_MACRO` serverbound: v72=109 (`0x6D`), v79=108 (`0x6C`), v83=110 (`0x6E`), v84=110 (`0x6E`), v87=113 (`0x71`), v92=121 (`0x79`), v95=122 (`0x7A`), jms185=105 (`0x69`). v48/v61 absent (⬜).
  - `MACRO_SYS_DATA_INIT` clientbound: v61=91 (`0x5B`), v72=113 (`0x71`), v79=117 (`0x75`), v83=124 (`0x7C`), v84=127 (`0x7F`), v87=132 (`0x84`), v92=139 (`0x8B`), v95=140 (`0x8C`), jms185=122 (`0x7A`). v48 absent (⬜).
- **`pt.Variants` indices** (`libs/atlas-packet/test/context.go:18-40`, positional — never insert, only append): `[0]`=GMS v28, `[1]`=v83, `[2]`=v87, `[3]`=v95, `[4]`=JMS v185, `[5]`=v84, `[6]`=v86, `[7]`=v48, `[8]`=v61, `[9]`=v72, `[10]`=v79, `[11]`=v92.
- **Guards that must be green before the PR** (run from the worktree root): `tools/lint.sh --check`, `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `go run ./tools/packet-audit matrix --check`, plus `go test -race ./...`, `go vet ./...`, `go build ./...` in `libs/atlas-packet`, `services/atlas-channel`, `services/atlas-configurations`.
- **No `go.mod` changes are expected.** If one is touched, `docker buildx bake atlas-<svc>` becomes mandatory for that service.
- **Stop-and-ask triggers** (do NOT guess past these): (a) shout polarity unresolvable from the decompile; (b) the client's macro-count capacity unresolvable; (c) IDA access unavailable for a whole IDB, which would force the design's Alternative C fallback — a user decision, never a unilateral downgrade.

---

## Task 1: Pin the pre-change byte baseline

Capture what the two *shipped* encoders emit today, before anything moves. This is the regression anchor for gms_83/84 (PRD acceptance criterion "byte-identical to `main`"), and it is also the artifact that makes the §1.1 polarity contradiction concrete rather than asserted.

**Files:**
- Create: `docs/tasks/task-226-skill-macro-version-coverage/baseline-bytes.md`
- Read only: `libs/atlas-packet/model/macros.go`, `libs/atlas-packet/character/skill_macro.go`

- [ ] **Step 1: Write a throwaway baseline capture test**

Create `libs/atlas-packet/model/macros_baseline_capture_test.go` (deleted again in Step 4 — it exists only to print bytes):

```go
package model

import (
	"encoding/hex"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCaptureMacrosBaseline(t *testing.T) {
	for _, i := range []int{1, 5} { // pt.Variants[1]=GMS v83, [5]=GMS v84
		v := pt.Variants[i]
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		m := NewMacros(
			NewMacro("Buff", true, skill.Id(1001003), skill.Id(1001004), skill.Id(0)),
			NewMacro("Attack", false, skill.Id(1001005), skill.Id(0), skill.Id(0)),
		)
		got := m.Encode(nil, ctx)(nil)
		t.Logf("%s len=%d bytes=%s", v.Name, len(got), hex.EncodeToString(got))
	}
}
```

- [ ] **Step 2: Run it and record the output**

Run: `cd libs/atlas-packet && go test ./model/ -run TestCaptureMacrosBaseline -v`
Expected: PASS, with two `t.Logf` lines printing a length and a hex string.

- [ ] **Step 3: Write `baseline-bytes.md`**

Record, verbatim from the test output, one section per variant. Structure:

```markdown
# Pre-change byte baseline (captured on branch task-226 before any codec move)

Fixture model (identical in every later fixture in this task):
  entry[0] = name "Buff",   shout true,  skillIds 1001003 / 1001004 / 0
  entry[1] = name "Attack", shout false, skillIds 1001005 / 0       / 0

## Shipped clientbound encoder — `model.Macros.Encode` (`libs/atlas-packet/model/macros.go:20`)
| variant | len | hex |
|---|---|---|
| GMS v83 | <from test output> | <from test output> |
| GMS v84 | <from test output> | <from test output> |

Shout is written UPRIGHT here (`macros.go:50` `w.WriteBool(m.shout)`).

## Dead clientbound encoder — `character.SkillMacro.Encode` (`libs/atlas-packet/character/skill_macro.go:39`)
Writes shout INVERTED (`skill_macro.go:47` `w.WriteBool(!e.Shout)`), so its byte 7
differs from the shipped encoder's for the same model. Not referenced by any
production announce site (see design.md §1.1); superseded by Task 6.

## Shipped serverbound decoder — `character.SkillMacro.Decode` (`libs/atlas-packet/character/skill_macro.go:54`)
Reads shout INVERTED (`skill_macro.go:64` `shout := !r.ReadBool()`), i.e. the
opposite polarity from the shipped encoder above. Exactly one of the two is
correct; Task 3 decides which from the IDB.
```

Fill every `<from test output>` with the actual value the test printed. Do not paraphrase or recompute by hand.

- [ ] **Step 4: Delete the capture test**

```bash
rm libs/atlas-packet/model/macros_baseline_capture_test.go
```

- [ ] **Step 5: Verify the tree builds clean**

Run: `cd libs/atlas-packet && go build ./... && go vet ./...`
Expected: no output, exit 0.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-226-skill-macro-version-coverage/baseline-bytes.md
git commit -m "docs(task-226): pin pre-change skill-macro byte baseline"
```

---

## Task 2: Harvest and splice the IDA exports

Neither function is usable in any committed export: `CMacroSysMan::FlushToSvr` appears in zero, `CWvsContext::OnMacroSysDataInit` appears in four as a stub with `calls: null` (design §1.3). Report generation is deterministic off the export, so this is the critical path — no report, no serverbound ✅ (`VERIFYING_A_PACKET.md` §9).

**Files:**
- Modify: `docs/packets/ida-exports/gms_v48.json`, `gms_v61.json`, `gms_v72.json`, `gms_v79.json`, `gms_v83.json`, `gms_v84.json`, `gms_v87.json`, `gms_v92.json`, `gms_v95.json`, `gms_jms_185.json`
- Create: `docs/tasks/task-226-skill-macro-version-coverage/harvest-log.md`
- Side effect: renamed symbols saved into the IDBs themselves (`idb_save`)

**Interfaces:**
- Produces: for every version, an export entry under `functions` keyed by the registry `fname` with real (non-null) `calls`, resolvable by `packet-audit evidence pin --ida "<FName>"` and by report generation.

- [ ] **Step 1: Enumerate the live IDA sessions**

Call `mcp__ida-pro__idb_list`. Record the exact binary **name** for each of the ten versions in `harvest-log.md`. Never address a session by port (`reference_packet_audit_select_instance_dead`).

If an IDB for an in-scope version is not loaded, STOP and ask the user to load it — a missing IDB is one of the playbook's genuine blockers (`VERIFYING_A_PACKET.md` "Producible prerequisite vs genuine blocker").

- [ ] **Step 2: Locate both functions in each IDB**

Per version, in this order:

1. `mcp__ida-pro__func_query` with `name_regex` `MacroSysMan|OnMacroSysDataInit|FlushToSvr` and the version's `database`. This is the documented lookup method — do not improvise (CLAUDE.md "Reverse Engineering / IDA").
2. If the sender is unnamed, find it by the send-site byte signature from `VERIFYING_A_PACKET.md` §10: `6A <op> 8D 8D ?? ?? ?? ?? E8` (push opcode; lea ecx; call `COutPacket` ctor), with `<op>` the version's registry opcode from Global Constraints. Use `mcp__ida-pro__find_bytes`.
3. Confirm the hit by decompiling it (`mcp__ida-pro__decompile`) and reading the literal integer in `COutPacket::COutPacket(&pkt, OPCODE)`. The integer is ground truth; the symbol is not (§10 "Distrust IDB function names"). It must equal the registry opcode.
4. If it does not match, that is a registry data error — record it in `harvest-log.md` and fix it in Task 5, do not bend the fixture to it.
5. Rename the function to the canonical name via `mcp__ida-pro__rename` (`CMacroSysMan::FlushToSvr` / `CWvsContext::OnMacroSysDataInit`), then `mcp__ida-pro__idb_save` (`feedback_name_idb_symbols_while_reversing`).

An unnamed function is not an absent function (`feedback_unnamed_idb_function_still_exists`). Only after step 2 fails a **binary-wide** search may absence even be considered — and then it is Task 4's problem, not this task's.

- [ ] **Step 3: Harvest to a temp file, one IDB at a time**

For each version, write a pending roster listing exactly the two function names, then:

```bash
go run ./tools/packet-audit export \
  -prior-export "" \
  -pending /tmp/claude-macro-roster-<version>.md \
  -descent-depth 12 \
  -ida-database "<binary name from idb_list>" \
  -output /tmp/claude-macro-harvest-<version>.json
```

The `-ida-database` flag is required (`reference_packet_audit_ida_database_flag`). Do **not** pass `-ida-url` with its default — the default port is dead and silently targets the wrong binary (`bug_packet_audit_ida_url_default_stale`). Serialize: one IDB at a time.

- [ ] **Step 4: Splice, surgically**

For each version, edit the committed export at `docs/packets/ida-exports/<export>.json` and:
- **overwrite** the two target entries with the harvested versions (a committed stub with `calls: null` is exactly what must be replaced);
- **add only-if-absent** any helper functions the harvest pulled in that the two entries' `calls` reference.

Do not touch any other key. Re-running `packet-audit export` wholesale drifts ~150 unrelated function keys and degrades unrelated cells (§10). If a harvested entry records `{op: Delegate, ref: COutPacket}`, strip that one call — it is the packet constructor, not a wire read (§10 "COutPacket-delegate harvest artifact").

- [ ] **Step 5: Verify the splice did not drift the export**

```bash
git diff --stat docs/packets/ida-exports/
```
Expected: only the ten export files changed, and `git diff docs/packets/ida-exports/gms_v83.json` shows changes confined to the `CWvsContext::OnMacroSysDataInit` / `CMacroSysMan::FlushToSvr` entries and newly-added helper entries. Any other function key appearing in the diff is drift — revert and re-splice by hand.

- [ ] **Step 6: Prove both entries are now resolvable**

```bash
go run ./tools/packet-audit matrix --check
```
Expected: exit 0, or failures unrelated to macros. This will NOT yet promote anything; it confirms the exports are still internally consistent.

- [ ] **Step 7: Write `harvest-log.md`**

One row per version: binary name from `idb_list`, the two function addresses, whether each was already named or renamed by this task, the `COutPacket` opcode integer read from the decompile, and whether it matched the registry. Mark any version where a function was **not found after a binary-wide search** as `NOT FOUND — see Task 4`.

- [ ] **Step 8: Commit**

```bash
git add docs/packets/ida-exports/ docs/tasks/task-226-skill-macro-version-coverage/harvest-log.md
git commit -m "chore(task-226): splice macro send/receive functions into IDA exports"
```

---

## Task 3: Derive the per-version layout

The decision task. Everything downstream — the polarity, the decode bound, the gates, the fixtures — reads its inputs from the document this task produces.

**Files:**
- Create: `docs/tasks/task-226-skill-macro-version-coverage/layout-derivation.md`

**Interfaces:**
- Produces: `layout-derivation.md` with (a) a per-version × per-field table, (b) a **Shout polarity** verdict, (c) a **Capacity** verdict (the client's maximum macro count), (d) a **Divergences** list naming every version/field pair needing a gate. Tasks 6, 7, 10 and 11 consume all four.

- [ ] **Step 1: Decompile both functions on every version**

For each of the ten versions, `mcp__ida-pro__decompile` both functions at the addresses recorded in `harvest-log.md`, descending into helper reads (address-based descent — `VERIFYING_A_PACKET.md` §3). Write down the **full ordered** read/write list, including guards and loop bounds.

- [ ] **Step 2: Resolve the count field and the capacity**

In `CWvsContext::OnMacroSysDataInit`, the loop bound over the entries answers both questions: the width of the leading count read (`Decode1` vs `Decode2` vs `Decode4`) and whether the client clamps it to a constant. Quote the actual decompiled line for each — do not paraphrase the number (CLAUDE.md: "quote the actual values before drawing a conclusion").

If the capacity is a compile-time constant compared against the decoded count, that constant is the Capacity verdict. If the client loops unbounded on the wire count, record Capacity as **unbounded** and say so.

- [ ] **Step 3: Resolve the shout polarity**

This is the load-bearing question (design §1.1) — production currently writes it upright and reads it inverted, so one of the two live versions is corrupting saved macros today.

Read both sides:
- `CWvsContext::OnMacroSysDataInit`: what does the client store from the 1-byte read, and is the stored value negated or compared against 0 before being used as "shout"?
- `CMacroSysMan::FlushToSvr`: what does the client write for a macro whose shout checkbox is ticked?

The verdict must be one of: `UPRIGHT` (1 = shout on), `INVERTED` (0 = shout on), or `UNKNOWN`. If `UNKNOWN`, **STOP and ask the user** — the affected cells do not promote and a coin flip here silently corrupts every saved macro (design §7).

- [ ] **Step 4: Resolve the name encoding and the skill-id widths**

Per version: is the name a length-prefixed string (`DecodeStr` → 2-byte length + bytes) or fixed-width? Are the three skill ids `Decode4` each, and are there exactly three? A fixed-width name on any version is a **structural** divergence and triggers the per-version-file option in design §2.3 — flag it explicitly.

- [ ] **Step 5: Write `layout-derivation.md`**

Structure:

```markdown
# Skill macro — per-version layout derivation (task-226)

Sources: docs/packets/ida-exports/ (spliced in Task 2) and live decompiles via
ida-pro-mcp. Addresses and binary names: see harvest-log.md.

## Verdicts

- **Shout polarity:** UPRIGHT | INVERTED | UNKNOWN — <one sentence, citing the
  decompiled line on the version that settled it, plus every version checked>
- **Capacity:** <integer> | unbounded — <citing the loop-bound line>
- **Count width:** <byte|uint16|uint32> — <citing the Decode line>

## Field table

| version | fn (clientbound) | fn (serverbound) | count | name | shout | skillId1..3 |
|---|---|---|---|---|---|---|
| gms_v48 | … | … | … | … | … | … |
| … one row per version … |

Each cell is either a concrete width/encoding with the decompiled line it came
from, or the literal word **unknown**.

## Divergences requiring a gate

| version(s) | field | divergence | gate expression |
|---|---|---|---|
| … | … | … | `t.IsRegion("GMS") && t.MajorAtLeast(N)` |

(If the table is empty, say so explicitly: "No divergence found across
v48..jms185; the layout is uniform and the codec carries no version gate."
An empty table is a finding, not an omission.)

## Unresolved

<Every field marked **unknown** above, restated with what would be needed to
resolve it. If none: "None.">
```

Every row cites an IDB function and a decompiled line. Every unresolved field is the literal word **unknown** — never a plausible value (FR-1.5).

- [ ] **Step 6: Gate-check the verdicts**

Re-read the three verdicts. If Shout polarity is `UNKNOWN`, or Capacity is needed by Task 7 and is `UNKNOWN`, stop here and raise it with the user before proceeding. Otherwise continue.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-226-skill-macro-version-coverage/layout-derivation.md
git commit -m "docs(task-226): per-version skill-macro layout derivation"
```

---

## Task 4: Resolve the v48/v61 `n-a` question

Three cells are `⬜` today: `SKILL_MACRO` on gms_v48 and gms_v61, and `MACRO_SYS_DATA_INIT` on gms_v48. `⬜` is a claim held to the same evidentiary bar as a verification (`VERIFYING_A_PACKET.md` "Is this cell `n-a`?"). Note that v61 already routes the clientbound writer (`template_gms_61_1.json:2524`, opcode `0x5B`) and carries the op in its registry (`gms_v61.yaml:613`) — so on v61 the receive side is present, which is positive pressure on the send side existing too.

**Files:**
- Modify (conditionally): `docs/packets/feature-na-evidence.yaml`, `docs/packets/registry/gms_v48.yaml`, `docs/packets/registry/gms_v61.yaml`
- Create: `docs/tasks/task-226-skill-macro-version-coverage/na-recheck.md`

**Interfaces:**
- Consumes: `harvest-log.md` (Task 2), `layout-derivation.md` (Task 3).
- Produces: `na-recheck.md` with a per-cell verdict of `CONFIRMED-NA` or `CORRECTED`. Tasks 10–12 read it to decide whether v48/v61 columns are in their sweep.

- [ ] **Step 1: Run the mandatory sibling cross-check**

For each of the three cells, decompile the **same-family opposite-direction** handler on that same version (playbook step 3: "the receive side proves the send side"):
- v61 `SKILL_MACRO` (send) → decompile v61 `CWvsContext::OnMacroSysDataInit` @ the address in `gms_v61.yaml:619`. If it decodes and stores macro state, the send side exists somewhere — keep looking.
- v48 both directions → there is no v48 registry entry for either, so search for the feature itself, not for a name.

- [ ] **Step 2: Anchor on invariants, not on names**

Per playbook step 2, search for the things that must exist if the feature exists:
- the opcode-construction site: scan **all** `COutPacket::COutPacket(&pkt, <int>)` call sites in `.text` for one whose body shape is `Encode1(count)` + per-entry `EncodeStr(name)` + `Encode1(shout)` + 3×`Encode4(skillId)`. Do not scope the search to an address range — the teleport-rock v48 case (task-124) failed exactly that way.
- the receive dispatcher's opcode-to-handler switch, for the clientbound side.
- the `CMacroSysMan` data structure or the Skill-window UI that would read/write it.

Record the search extent as a number ("all N `COutPacket(long)` call sites scanned"), not as an adjective.

- [ ] **Step 3: Record the verdict per cell in `na-recheck.md`**

```markdown
# v48 / v61 n-a re-check (task-226, FR-2)

## SKILL_MACRO × gms_v48 — <CONFIRMED-NA | CORRECTED>
<evidence: what was searched, how exhaustively, with counts and addresses>

## SKILL_MACRO × gms_v61 — <CONFIRMED-NA | CORRECTED>
<evidence, including the sibling cross-check result on OnMacroSysDataInit>

## MACRO_SYS_DATA_INIT × gms_v48 — <CONFIRMED-NA | CORRECTED>
<evidence>
```

- [ ] **Step 4a (only if a cell is CONFIRMED-NA): add positive-absence evidence**

Append to `docs/packets/feature-na-evidence.yaml` under `entries:`, matching the existing shape (`feature-na-evidence.yaml:9-24`):

```yaml
  - op: SKILL_MACRO
    version: gms_v61
    evidence: >
      <the positive proof from Steps 1-2, verbatim and specific: the exhaustive
      scan and its count, the addresses checked, and the sibling cross-check
      outcome. Blank or hand-wavy evidence does not count — the gate requires
      non-empty text and a reviewer requires it to be checkable.>
```

- [ ] **Step 4b (only if a cell is CORRECTED): add the registry op**

Insert the op into the version's registry file at its opcode-sorted position, matching the shape at `docs/packets/registry/gms_v79.yaml:3049-3056`:

```yaml
- op: SKILL_MACRO
  direction: serverbound
  opcode: <the integer read from COutPacket::COutPacket in Step 2>
  fname: <the canonical or renamed symbol>
  provenance: ida-discovered
  ida:
    address: <decimal address>
  note: 'task-226 FR-2.3: corrected from n-a. <one sentence of the evidence>'
```

A corrected cell is then in scope for Tasks 6–12 exactly like any other version: it needs a layout row in `layout-derivation.md` (go back and add it), a template binding, a fixture, and evidence. Correcting a wrong `n-a` is in scope, not a follow-up (FR-2.3, CLAUDE.md "No Deferring Producible Work").

- [ ] **Step 5: Verify the registry/evidence files still parse**

```bash
go run ./tools/packet-audit matrix --check
```
Expected: exit 0, or a failure that names only the macro rows. A YAML parse error here is a hard stop.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-226-skill-macro-version-coverage/na-recheck.md docs/packets/feature-na-evidence.yaml docs/packets/registry/
git commit -m "fix(task-226): re-check v48/v61 skill-macro n-a claims against the IDBs"
```

---

## Task 5: Correct the v72 registry entry

`docs/packets/registry/gms_v72.yaml:2562-2569` records `SKILL_MACRO` with `fname: sub_6022DB` and `ida.address: 6175200` (= `0x5E39E0`). `0x6022DB` is the **v79** address (`gms_v79.yaml:3049-3056`, `ida.address: 6300379`); `0x5E39E0` is the `COutPacket`-ctor harvest site named in the v72 entry's own note, not the sender. The fname was copied from v79 (design §1.4).

**Files:**
- Modify: `docs/packets/registry/gms_v72.yaml:2562-2569`
- Modify: `docs/packets/registry/gms_v79.yaml:3049-3056` (fname only, if Task 2 renamed the v79 symbol)
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_72_1.json:804-810`, `template_gms_79_1.json:804-810` (the `fname` field of the existing handler bindings)

- [ ] **Step 1: Rewrite the v72 entry from the harvest**

Using the v72 address and canonical name recorded in `harvest-log.md`:

```yaml
- op: SKILL_MACRO
  direction: serverbound
  opcode: 109
  fname: CMacroSysMan::FlushToSvr
  provenance: ida-discovered
  ida:
    address: <decimal of the v72 sender address from harvest-log.md>
  note: 'task-226: fname/address corrected — the prior entry carried v79''s sub_6022DB and the 0x5e39e0 COutPacket-ctor harvest site. Renamed in the v72 IDB and spliced into the export.'
```

Keep `opcode: 109` unless Step 2 of Task 2 read a different integer from `COutPacket::COutPacket` — in which case use the read integer and note the correction.

- [ ] **Step 2: Align the v79 entry's fname if it was renamed**

If Task 2 renamed the v79 sender from `sub_6022DB` to `CMacroSysMan::FlushToSvr`, update `gms_v79.yaml:3052` to match. A registry fname edit stales the matrix (`bug_registry_fname_change_stales_packet_matrix`) — Step 4 regenerates it.

- [ ] **Step 3: Align the two template `fname` fields**

`template_gms_72_1.json:807` and `template_gms_79_1.json:807` both carry `"fname": "sub_6022DB"`. Set each to the canonical name now recorded in that version's registry. Change only the `fname` value — leave `opCode`, `validator`, `handler`, and `services` untouched, and do not move the entries (the opcode-order guard reads position).

- [ ] **Step 4: Regenerate and check the matrix**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```
Expected: exit 0. The macro rows still read ❌ — nothing has been verified yet; this only proves the registry/template/export now agree on names.

- [ ] **Step 5: Run the template guards**

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
```
Expected: both exit 0 with no findings.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/registry/ services/atlas-configurations/seed-data/templates/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "fix(task-226): correct v72 SKILL_MACRO fname/address (v79 carryover)"
```

---

## Task 6: Clientbound codec — `character/clientbound/SkillMacro`

Absorb `model.Macros` into an audited path. This struct becomes THE production clientbound encoder; `model/macros.go` is deleted in Task 8, so nothing can drift from what gets verified (design §2.2).

**Files:**
- Create: `libs/atlas-packet/character/clientbound/skill_macro.go`
- Test: `libs/atlas-packet/character/clientbound/skill_macro_test.go`

**Interfaces:**
- Consumes: `layout-derivation.md` verdicts (Task 3).
- Produces:
  - `const CharacterSkillMacroWriter = "CharacterSkillMacro"`
  - `func NewSkillMacroEntry(name string, shout bool, skillId1, skillId2, skillId3 skill.Id) SkillMacroEntry`
  - `func NewSkillMacro(macros ...SkillMacroEntry) SkillMacro`
  - `func (m SkillMacro) Macros() []SkillMacroEntry`
  - `func (m SkillMacro) Operation() string`
  - `func (m SkillMacro) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte`
  - `func (m *SkillMacro) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{})`
  - entry getters: `Name() string`, `Shout() bool`, `SkillId1() skill.Id`, `SkillId2() skill.Id`, `SkillId3() skill.Id`
  - Tasks 8 (rewiring), 9 (`candidatesFromFName`), 10 (verification) all depend on these exact names.

- [ ] **Step 1: Write the failing byte test**

Create `libs/atlas-packet/character/clientbound/skill_macro_test.go`:

```go
package clientbound

import (
	"encoding/hex"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// fixture builds the shared two-entry model used by every macro fixture in
// task-226. Kept identical to baseline-bytes.md's fixture so the v83/v84 rows
// are directly comparable to the pre-change capture.
func fixture() SkillMacro {
	return NewSkillMacro(
		NewSkillMacroEntry("Buff", true, skill.Id(1001003), skill.Id(1001004), skill.Id(0)),
		NewSkillMacroEntry("Attack", false, skill.Id(1001005), skill.Id(0), skill.Id(0)),
	)
}

func TestSkillMacroByteOutput(t *testing.T) {
	cases := []struct {
		variant pt.TenantVariant
		want    string
	}{
		{pt.Variants[8], wantUniform},  // GMS v61
		{pt.Variants[9], wantUniform},  // GMS v72
		{pt.Variants[10], wantUniform}, // GMS v79
		{pt.Variants[1], wantUniform},  // GMS v83
		{pt.Variants[5], wantUniform},  // GMS v84
		{pt.Variants[2], wantUniform},  // GMS v87
		{pt.Variants[11], wantUniform}, // GMS v92
		{pt.Variants[3], wantUniform},  // GMS v95
		{pt.Variants[4], wantUniform},  // JMS v185
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			got := hex.EncodeToString(fixture().Encode(nil, ctx)(nil))
			if got != tc.want {
				t.Errorf("bytes:\n got %s\nwant %s", got, tc.want)
			}
		})
	}
}
```

Above `TestSkillMacroByteOutput`, add the IDA-evidence comment block and one marker line per version, taking each address from `harvest-log.md`:

```go
// TestSkillMacroByteOutput verifies MACRO_SYS_DATA_INIT byte output against the
// client's CWvsContext::OnMacroSysDataInit read order (see layout-derivation.md).
//
// Wire layout: count(1) + per entry { name(2+len) + shout(1) + skillId1(4) +
// skillId2(4) + skillId3(4) }.
// Fixture: 1 + (2+4+1+12) + (2+6+1+12) = 41 bytes.
//
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v61 ida=0x…
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v72 ida=0x…
// … one line per version in scope, address from harvest-log.md …
```

Define `wantUniform` at the top of the file. Its value is derived from the layout in `layout-derivation.md`, and under the uniform hypothesis (count `byte`; name `WriteAsciiString` = LE `uint16` length + ASCII bytes, `libs/atlas-socket/response/writer.go:82-93`; shout 1 byte; three LE `uint32` skill ids, `writer.go:36`) with **UPRIGHT** polarity it is:

```go
// count=2
// entry0: len=0x0004 "Buff"   shout=01 1001003=0x000f462b 1001004=0x000f462c 0
// entry1: len=0x0006 "Attack" shout=00 1001005=0x000f462d 0 0
const wantUniform = "02" +
	"0400" + "42756666" + "01" + "2b460f00" + "2c460f00" + "00000000" +
	"0600" + "41747461636b" + "00" + "2d460f00" + "00000000" + "00000000"
```

That is 41 hex-encoded bytes (82 hex chars): `1 + (2+4+1+12) + (2+6+1+12)`. The line above is the value to *start* from, and Step 4 replaces the whole constant with the value the run actually produced, only after each byte is accounted for by a row in `layout-derivation.md`. Never keep a hand-typed expectation the run disagrees with; and never paste a run's output in without checking it field by field against the derivation.

If `layout-derivation.md` records the polarity as **INVERTED**, byte 7 is `00` and the second entry's shout byte is `01`; if it records a per-version divergence, replace the single `wantUniform` with one constant per divergent group and update the case table accordingly. If a version's layout row is **unknown**, omit that version from the case table and from the marker list — an unknown cell does not promote.

- [ ] **Step 2: Run the test to see it fail**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run TestSkillMacroByteOutput -v`
Expected: FAIL — `undefined: NewSkillMacro` / `undefined: NewSkillMacroEntry`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/character/clientbound/skill_macro.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterSkillMacroWriter = "CharacterSkillMacro"

// SkillMacroEntry is one row of the client's Skill-window macro list: a name,
// the "shout the macro name in chat" flag, and up to three bound skill ids.
type SkillMacroEntry struct {
	name     string
	shout    bool
	skillId1 skill.Id
	skillId2 skill.Id
	skillId3 skill.Id
}

func NewSkillMacroEntry(name string, shout bool, skillId1 skill.Id, skillId2 skill.Id, skillId3 skill.Id) SkillMacroEntry {
	return SkillMacroEntry{name: name, shout: shout, skillId1: skillId1, skillId2: skillId2, skillId3: skillId3}
}

func (e SkillMacroEntry) Name() string       { return e.name }
func (e SkillMacroEntry) Shout() bool        { return e.shout }
func (e SkillMacroEntry) SkillId1() skill.Id { return e.skillId1 }
func (e SkillMacroEntry) SkillId2() skill.Id { return e.skillId2 }
func (e SkillMacroEntry) SkillId3() skill.Id { return e.skillId3 }

// SkillMacro is the clientbound MACRO_SYS_DATA_INIT packet
// (CWvsContext::OnMacroSysDataInit): the server hands the client the character's
// whole macro list, at login and after every macro update.
//
// Byte layout: see docs/tasks/task-226-skill-macro-version-coverage/layout-derivation.md.
//
// packet-audit:fname CWvsContext::OnMacroSysDataInit
type SkillMacro struct {
	macros []SkillMacroEntry
}

func NewSkillMacro(macros ...SkillMacroEntry) SkillMacro {
	return SkillMacro{macros: macros}
}

func (m SkillMacro) Macros() []SkillMacroEntry { return m.macros }

func (m SkillMacro) Operation() string { return CharacterSkillMacroWriter }

func (m SkillMacro) String() string {
	return fmt.Sprintf("macros [%d]", len(m.macros))
}

func (m SkillMacro) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.macros)))
		for _, e := range m.macros {
			w.WriteAsciiString(e.name)
			w.WriteBool(e.shout)
			w.WriteInt(uint32(e.skillId1))
			w.WriteInt(uint32(e.skillId2))
			w.WriteInt(uint32(e.skillId3))
		}
		return w.Bytes()
	}
}

func (m *SkillMacro) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadByte()
		m.macros = make([]SkillMacroEntry, 0, count)
		for range count {
			name := r.ReadAsciiString()
			shout := r.ReadBool()
			skillId1 := skill.Id(r.ReadUint32())
			skillId2 := skill.Id(r.ReadUint32())
			skillId3 := skill.Id(r.ReadUint32())
			m.macros = append(m.macros, NewSkillMacroEntry(name, shout, skillId1, skillId2, skillId3))
		}
	}
}
```

Then apply `layout-derivation.md`:
- **Polarity.** The code above writes and reads `shout` **upright**. If the derivation's verdict is `INVERTED`, change *both* `w.WriteBool(e.shout)` → `w.WriteBool(!e.shout)` and `shout := r.ReadBool()` → `shout := !r.ReadBool()`, and say so in the struct doc comment citing the decompiled line. Both sides must move together — a one-sided flip re-creates the §1.1 defect.
- **Gates.** For every row in the derivation's "Divergences requiring a gate" table, add `t := tenant.MustFromContext(ctx)` at the top of `Encode` (changing the `_ context.Context` parameter to `ctx context.Context`) and wrap the divergent write in the gate expression the table names, with a comment citing the decompiled line — following `libs/atlas-packet/field/serverbound/general.go:44-58`. If the table is empty, leave the signature as `_ context.Context` and note in the doc comment that the layout is uniform across v48..jms185 per the derivation.

- [ ] **Step 4: Run the test and read the actual bytes**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run TestSkillMacroByteOutput -v`
Expected: FAIL, with `got <hex>` printed for each variant.

Now check the printed `got` against the layout table field by field: count byte, name length prefix, name bytes, shout byte, three skill ids. Only after each byte is accounted for by a derivation row, paste the value into `wantUniform`.

Cross-check the v83 and v84 rows against `baseline-bytes.md`'s **shipped clientbound encoder** row. They must be identical unless the derivation's polarity verdict is `INVERTED`, in which case exactly the two shout bytes differ — that deviation is expected, is recorded in `layout-derivation.md`, and must be called out in the PR description (design §5, superseding the PRD's "byte-identical to `main`" criterion).

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run TestSkillMacroByteOutput -v`
Expected: PASS for every case.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/clientbound/skill_macro.go libs/atlas-packet/character/clientbound/skill_macro_test.go
git commit -m "feat(task-226): clientbound SkillMacro codec with IDA-derived layout"
```

---

## Task 7: Serverbound codec — `character/serverbound/SkillMacro`

**Files:**
- Create: `libs/atlas-packet/character/serverbound/skill_macro.go`
- Test: `libs/atlas-packet/character/serverbound/skill_macro_test.go`

**Interfaces:**
- Consumes: `layout-derivation.md` verdicts, including **Capacity** (Task 3).
- Produces:
  - `const CharacterSkillMacroHandle = "CharacterSkillMacroHandle"`
  - `func NewSkillMacroEntry(name string, shout bool, skillId1, skillId2, skillId3 skill.Id) SkillMacroEntry`
  - `func (m SkillMacro) Macros() []SkillMacroEntry` + the same five entry getters as Task 6
  - `func (m *SkillMacro) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{})`
  - `func (m SkillMacro) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte`
  - Task 8 (handler rewiring) and Task 11 (verification) depend on these exact names.

- [ ] **Step 1: Write the failing decode test**

Create `libs/atlas-packet/character/serverbound/skill_macro_test.go`:

```go
package serverbound

import (
	"encoding/hex"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func TestSkillMacroDecode(t *testing.T) {
	cases := []struct{ variant pt.TenantVariant }{
		{pt.Variants[9]},  // GMS v72
		{pt.Variants[10]}, // GMS v79
		{pt.Variants[1]},  // GMS v83
		{pt.Variants[5]},  // GMS v84
		{pt.Variants[2]},  // GMS v87
		{pt.Variants[11]}, // GMS v92
		{pt.Variants[3]},  // GMS v95
		{pt.Variants[4]},  // JMS v185
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			raw, err := hex.DecodeString(wireUniform)
			if err != nil {
				t.Fatalf("fixture hex: %v", err)
			}
			m := SkillMacro{}
			m.Decode(nil, ctx)(request.NewReader(raw), nil)

			got := m.Macros()
			if len(got) != 2 {
				t.Fatalf("count: got %d, want 2", len(got))
			}
			if got[0].Name() != "Buff" || !got[0].Shout() ||
				got[0].SkillId1() != skill.Id(1001003) ||
				got[0].SkillId2() != skill.Id(1001004) ||
				got[0].SkillId3() != skill.Id(0) {
				t.Errorf("entry 0: got %+v", got[0])
			}
			if got[1].Name() != "Attack" || got[1].Shout() ||
				got[1].SkillId1() != skill.Id(1001005) ||
				got[1].SkillId2() != skill.Id(0) ||
				got[1].SkillId3() != skill.Id(0) {
				t.Errorf("entry 1: got %+v", got[1])
			}
		})
	}
}

func TestSkillMacroDecodeClampsCount(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	// A count byte of 0xFF with no entry bytes behind it: the decoder must stop
	// at the client's capacity (layout-derivation.md "Capacity"), not allocate
	// and parse 255 entries off an exhausted reader.
	m := SkillMacro{}
	m.Decode(nil, ctx)(request.NewReader([]byte{0xFF}), nil)
	if len(m.Macros()) > maxMacroCount {
		t.Errorf("decoded %d entries, want at most %d", len(m.Macros()), maxMacroCount)
	}
}
```

Define `wireUniform` in the test file as the **same hex string** the clientbound test settled on in Task 6 Step 4 (the client's send order and its read order are the same field order unless `layout-derivation.md` says otherwise; if it does say otherwise, use the serverbound order and note the difference in a comment). Add the marker block above `TestSkillMacroDecode`:

```go
// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=gms_v72 ida=0x…
// … one line per version in scope, address from harvest-log.md …
```

Verify `request.NewReader`'s exact constructor signature at `libs/atlas-socket/request/reader.go` before running; if it takes more than a `[]byte`, adjust the two call sites.

If `layout-derivation.md` marks Capacity **unbounded**, replace `TestSkillMacroDecodeClampsCount` with a test that asserts the decoder stops cleanly at the end of the reader rather than at a constant, and drop `maxMacroCount` from the codec — do not invent a cap the client does not have.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/serverbound/ -run TestSkillMacro -v`
Expected: FAIL — `undefined: SkillMacro` / `undefined: maxMacroCount`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/character/serverbound/skill_macro.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterSkillMacroHandle = "CharacterSkillMacroHandle"

// maxMacroCount is the client's macro capacity, read out of the loop bound in
// CWvsContext::OnMacroSysDataInit — see layout-derivation.md "Capacity". The
// decoder clamps to it rather than trusting the wire count, so a corrupt or
// hostile count byte cannot drive an unbounded parse (design §2.4).
const maxMacroCount = <capacity from layout-derivation.md>

// SkillMacroEntry is one row of the macro list as the client sends it.
type SkillMacroEntry struct {
	name     string
	shout    bool
	skillId1 skill.Id
	skillId2 skill.Id
	skillId3 skill.Id
}

func NewSkillMacroEntry(name string, shout bool, skillId1 skill.Id, skillId2 skill.Id, skillId3 skill.Id) SkillMacroEntry {
	return SkillMacroEntry{name: name, shout: shout, skillId1: skillId1, skillId2: skillId2, skillId3: skillId3}
}

func (e SkillMacroEntry) Name() string       { return e.name }
func (e SkillMacroEntry) Shout() bool        { return e.shout }
func (e SkillMacroEntry) SkillId1() skill.Id { return e.skillId1 }
func (e SkillMacroEntry) SkillId2() skill.Id { return e.skillId2 }
func (e SkillMacroEntry) SkillId3() skill.Id { return e.skillId3 }

// SkillMacro is the serverbound SKILL_MACRO packet (CMacroSysMan::FlushToSvr):
// the client flushes its whole macro list whenever the Skill window's macro
// editor is confirmed.
//
// Byte layout: see docs/tasks/task-226-skill-macro-version-coverage/layout-derivation.md.
//
// packet-audit:fname CMacroSysMan::FlushToSvr
type SkillMacro struct {
	macros []SkillMacroEntry
}

func NewSkillMacro(macros ...SkillMacroEntry) SkillMacro {
	return SkillMacro{macros: macros}
}

func (m SkillMacro) Macros() []SkillMacroEntry { return m.macros }

func (m SkillMacro) Operation() string { return CharacterSkillMacroHandle }

func (m SkillMacro) String() string {
	return fmt.Sprintf("macros [%d]", len(m.macros))
}

func (m SkillMacro) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.macros)))
		for _, e := range m.macros {
			w.WriteAsciiString(e.name)
			w.WriteBool(e.shout)
			w.WriteInt(uint32(e.skillId1))
			w.WriteInt(uint32(e.skillId2))
			w.WriteInt(uint32(e.skillId3))
		}
		return w.Bytes()
	}
}

func (m *SkillMacro) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := int(r.ReadByte())
		if count > maxMacroCount {
			count = maxMacroCount
		}
		m.macros = make([]SkillMacroEntry, 0, count)
		for range count {
			name := r.ReadAsciiString()
			shout := r.ReadBool()
			skillId1 := skill.Id(r.ReadUint32())
			skillId2 := skill.Id(r.ReadUint32())
			skillId3 := skill.Id(r.ReadUint32())
			m.macros = append(m.macros, NewSkillMacroEntry(name, shout, skillId1, skillId2, skillId3))
		}
	}
}
```

Replace `<capacity from layout-derivation.md>` with the integer that document records. **Do not guess it** — if the derivation says `unknown`, stop and ask (Global Constraints stop-and-ask trigger (b)).

Apply the same polarity verdict as Task 6 — both files must express shout identically — and the same gate rows from the derivation's divergence table.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./character/serverbound/ -run TestSkillMacro -v`
Expected: PASS for every case, including `TestSkillMacroDecodeClampsCount`.

- [ ] **Step 5: Add the cross-seam round-trip test**

Append to `libs/atlas-packet/character/serverbound/skill_macro_test.go`:

```go
// TestSkillMacroCrossSeamRoundTrip encodes with the SERVERBOUND struct and
// decodes with the SERVERBOUND struct — the intra-file agreement check. The
// cross-DIRECTION check (clientbound Encode → serverbound Decode) is not a
// round trip at all on the wire and is deliberately not asserted here: the two
// directions are proven separately against absolute bytes, which is what
// stopped the double-inversion in the old shared struct from cancelling out
// (design §1.1, bug_matrix_roundtrip_fixture_false_verify).
func TestSkillMacroCrossSeamRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewSkillMacro(
				NewSkillMacroEntry("Buff", true, skill.Id(1001003), skill.Id(1001004), skill.Id(0)),
				NewSkillMacroEntry("Attack", false, skill.Id(1001005), skill.Id(0), skill.Id(0)),
			)
			output := SkillMacro{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Macros()) != len(input.Macros()) {
				t.Fatalf("count: got %d, want %d", len(output.Macros()), len(input.Macros()))
			}
			for i, e := range output.Macros() {
				if e.Name() != input.Macros()[i].Name() {
					t.Errorf("macros[%d].Name: got %v, want %v", i, e.Name(), input.Macros()[i].Name())
				}
				if e.Shout() != input.Macros()[i].Shout() {
					t.Errorf("macros[%d].Shout: got %v, want %v", i, e.Shout(), input.Macros()[i].Shout())
				}
				if e.SkillId1() != input.Macros()[i].SkillId1() {
					t.Errorf("macros[%d].SkillId1: got %v, want %v", i, e.SkillId1(), input.Macros()[i].SkillId1())
				}
			}
		})
	}
}
```

- [ ] **Step 6: Run the full package tests**

Run: `cd libs/atlas-packet && go test -race ./character/... && go vet ./...`
Expected: PASS, no vet findings.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/character/serverbound/skill_macro.go libs/atlas-packet/character/serverbound/skill_macro_test.go
git commit -m "feat(task-226): serverbound SkillMacro codec with bounded decode"
```

---

## Task 8: Re-point atlas-channel and delete the legacy codecs

Only after both new codecs are green does production switch over. This is the commit that makes the audited codec and the shipped codec the same object.

**Files:**
- Delete: `libs/atlas-packet/character/skill_macro.go`, `libs/atlas-packet/character/skill_macro_test.go`, `libs/atlas-packet/model/macros.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go:91` (import), `:730`, `:943`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:36` (import), `:368-373`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/macro/consumer.go:22` (import), `:80-85`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_macro.go`

**Interfaces:**
- Consumes: `charcb.NewSkillMacro`, `charcb.NewSkillMacroEntry`, `charcb.CharacterSkillMacroWriter` (Task 6); `charsb.SkillMacro`, `charsb.CharacterSkillMacroHandle` (Task 7).

- [ ] **Step 1: Confirm nothing else depends on the deleted symbols**

```bash
grep -rn "model.Macros\|packetmodel.NewMacro\|NewMacros\|character2.CharacterSkillMacro\|charpkt.CharacterSkillMacro" libs services --include='*.go'
```
Expected: exactly the six call sites listed under **Files** above (main.go ×2, session consumer ×3, macro consumer ×3) plus the definitions being deleted. Any other hit means this task's file list is incomplete — extend it before proceeding.

- [ ] **Step 2: Update the serverbound handler**

Rewrite `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_macro.go`:

```go
package handler

import (
	"atlas-channel/macro"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func CharacterSkillMacroHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.SkillMacro{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		macros := make([]macro.Model, 0)
		for i, e := range p.Macros() {
			m := macro.NewModel(uint32(i), e.Name(), e.Shout(), e.SkillId1(), e.SkillId2(), e.SkillId3())
			macros = append(macros, m)
		}
		err := macro.NewProcessor(l, ctx).Update(s.CharacterId(), macros)
		if err != nil {
			l.WithError(err).Errorf("Unable to update skill macros for character [%d].", s.CharacterId())
		}
	}
}
```

The `skill.Id(...)` conversions and the `atlas-constants/skill` import are gone — `SkillMacroEntry` already carries `skill.Id` and `macro.NewModel` (`services/atlas-channel/atlas.com/channel/macro/model.go:40`) already takes `skill.Id`.

- [ ] **Step 3: Update the login announce site**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go`, replace lines 368-373 with:

```go
						ems := make([]charcb.SkillMacroEntry, 0)
						for _, sm := range sms {
							ems = append(ems, charcb.NewSkillMacroEntry(sm.Name(), sm.Shout(), sm.SkillId1(), sm.SkillId2(), sm.SkillId3()))
						}
						macros := charcb.NewSkillMacro(ems...)
						err = session.Announce(l)(ctx)(wp)(charcb.CharacterSkillMacroWriter)(macros.Encode)(s)
```

`charcb` is already imported at line 37. Delete the now-unused `charpkt` import (line 36) and the `packetmodel` import (confirm no other `packetmodel.` reference survives in the file with the Step 1 grep).

- [ ] **Step 4: Update the macro-update announce site**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/macro/consumer.go`, replace lines 80-85 with:

```go
				ems := make([]charcb.SkillMacroEntry, 0, len(sorted))
				for _, sm := range sorted {
					ems = append(ems, charcb.NewSkillMacroEntry(sm.Name, sm.Shout, skill2.Id(sm.SkillId1), skill2.Id(sm.SkillId2), skill2.Id(sm.SkillId3)))
				}
				macros := charcb.NewSkillMacro(ems...)
				return session.Announce(l)(ctx)(wp)(charcb.CharacterSkillMacroWriter)(macros.Encode)
```

Add `charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"` to the import block; remove the `charpkt` (line 22) and `packetmodel` imports. `skill2` stays — `macro2.MacroBody`'s skill fields are raw `uint32`.

- [ ] **Step 5: Update main.go**

- Line 730: `character2.CharacterSkillMacroWriter` → `charcb.CharacterSkillMacroWriter`
- Line 943: `handlerMap[character2.CharacterSkillMacroHandle]` → `handlerMap[charsb.CharacterSkillMacroHandle]`
- Line 91: delete the `character2 "github.com/Chronicle20/atlas/libs/atlas-packet/character"` import — the Step 1 grep proved these were its only two uses. `charcb` (line 92) and `charsb` (line 94) are already imported.

- [ ] **Step 6: Delete the legacy files**

```bash
rm libs/atlas-packet/character/skill_macro.go \
   libs/atlas-packet/character/skill_macro_test.go \
   libs/atlas-packet/model/macros.go
```

- [ ] **Step 7: Build and test both modules**

```bash
cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./...
cd ../../services/atlas-channel && go build ./... && go vet ./... && go test -race ./...
```
Expected: all clean. An "imported and not used" error here means a Step 3–5 import removal was missed.

- [ ] **Step 8: Run the repo-root guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```
Expected: exit 0 each. `tools/lint.sh --check` needs nvm on PATH or it false-fails (`bug_lint_check_false_fails_without_nvm`); if it reports an atlas-ui failure with no atlas-ui change in this branch, confirm the nvm environment before treating it as real.

- [ ] **Step 9: Commit**

```bash
git add -A libs/atlas-packet services/atlas-channel
git commit -m "refactor(task-226): route production macro announce/handle through the audited codecs"
```

---

## Task 9: Link both codecs into packet-audit

`grep -n Macro tools/packet-audit/cmd/run.go` returns no hits today — neither op maps to a codec, so no report can be generated for either (design §1.5, `VERIFYING_A_PACKET.md` §9 "Linkage").

**Files:**
- Modify: `tools/packet-audit/cmd/run.go` (the `candidatesFromFName` switch, in the `--- Character domain ---` block near `:412-434`)

**Interfaces:**
- Produces: `CWvsContext::OnMacroSysDataInit` → `{name: "SkillMacro", pkg: "character", dir: Clientbound}` → report `CharacterSkillMacro.json`; `CMacroSysMan::FlushToSvr` → `{name: "SkillMacro", pkg: "character", dir: Serverbound, reportName: "CharacterSkillMacroHandle"}` → report `CharacterSkillMacroHandle.json`. Tasks 10 and 11 consume both report names.

- [ ] **Step 1: Add the two cases**

In the character-domain section of `candidatesFromFName`, alongside the existing `BridleMobCatchFail` / `SetCard` cases:

```go
	case "CWvsContext::OnMacroSysDataInit":
		// MACRO_SYS_DATA_INIT — the server hands the client its whole macro list.
		// qualifiedWriterName("character","SkillMacro") = CharacterSkillMacro,
		// matching the template writer binding name.
		return []candidate{{name: "SkillMacro", pkg: "character", dir: csvpkg.DirClientbound}}
	case "CMacroSysMan::FlushToSvr":
		// SKILL_MACRO — the client flushes its macro list. Both directions derive
		// "CharacterSkillMacro" from qualifiedWriterName, which would collide on
		// the flat, writerName-keyed audit dir; the serverbound side overrides to
		// CharacterSkillMacroHandle (the SummonMoveHandle precedent, run.go:1137).
		return []candidate{{name: "SkillMacro", pkg: "character", dir: csvpkg.DirServerbound, reportName: "CharacterSkillMacroHandle"}}
```

If Task 2 could not rename a legacy sender to `CMacroSysMan::FlushToSvr` and that version's registry keeps a `sub_XXXXXX` primary fname, add one extra `case "sub_XXXXXX":` returning the identical serverbound candidate, with a comment naming the version and the `COutPacket` opcode read — the `sub_5D9424` / `sub_5DA381` precedent at `run.go:1146-1160`.

- [ ] **Step 2: Verify the tool still builds**

Run: `cd tools/packet-audit && go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 3: Generate the reports to a temp directory**

For each in-scope version (§9 step 3 of the playbook):

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  -ida-source docs/packets/ida-exports/gms_v83.json \
  -output /tmp/claude-macro-rpt
```

Expected: `/tmp/claude-macro-rpt/gms_v83/CharacterSkillMacro.{json,md}` and `/tmp/claude-macro-rpt/gms_v83/CharacterSkillMacroHandle.{json,md}` both exist. If a report is missing with "delegate to COutPacket: not in export" or "sub_XXXX not in export", that is a Task 2 splice gap — go back and fix the export (§10), do not work around it.

- [ ] **Step 4: Copy the reports in**

For each version, copy only the two macro reports:

```bash
cp /tmp/claude-macro-rpt/<version>/CharacterSkillMacro.{json,md} docs/packets/audits/<version>/
cp /tmp/claude-macro-rpt/<version>/CharacterSkillMacroHandle.{json,md} docs/packets/audits/<version>/
```

Copy the serverbound report only for versions where `SKILL_MACRO` exists in the registry, and the clientbound one only where `MACRO_SYS_DATA_INIT` does (Global Constraints, as amended by Task 4).

- [ ] **Step 5: Read each report before trusting it**

Open each `.md` and check the field list against `layout-derivation.md`. A report that shows fields Atlas does not write, or a mis-alignment, is a real finding — resolve it against the derivation rather than proceeding to pin evidence on top of it.

- [ ] **Step 6: Verify the matrix is still consistent**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```
Expected: exit 0. Cells still ❌ — evidence is not pinned yet.

- [ ] **Step 7: Commit**

```bash
git add tools/packet-audit/cmd/run.go docs/packets/audits/
git commit -m "feat(task-226): link both macro codecs into packet-audit candidate resolution"
```

---

## Task 10: Verify the clientbound row (`MACRO_SYS_DATA_INIT`)

**Files:**
- Create: `docs/packets/evidence/<version>/character.clientbound.CharacterSkillMacro.yaml` (one per in-scope version)
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`
- Modify (conditionally): `docs/packets/feature-families.yaml`

**Interfaces:**
- Consumes: markers written in Task 6, reports copied in Task 9, layout rows from Task 3.

- [ ] **Step 1: Register the feature family**

Add to `docs/packets/feature-families.yaml` under `families:`:

```yaml
  skill_macro:
    - SKILL_MACRO
    - MACRO_SYS_DATA_INIT
```

This is deliberate and will make `matrix --check` fail while any member is `⬜` on a version where the sibling is `verified` — it is the open-question tracker for the v48/v61 question (design §2.5). It must be green by Task 13; Task 4's `feature-na-evidence.yaml` entries are what make it green.

- [ ] **Step 2: Pin evidence, one version at a time**

For each in-scope version:

```bash
go run ./tools/packet-audit evidence pin \
  --packet character/clientbound/CharacterSkillMacro \
  --version <version key> \
  --ida "CWvsContext::OnMacroSysDataInit" \
  --category TIER1-FIXTURE
```

`--ida` takes the function name exactly as it keys the export's `functions` map, not a hex address. If it fails "not in export", that is a Task 2 splice gap.

- [ ] **Step 3: Add the `verifies:` field to each record**

Open each written `docs/packets/evidence/<version>/character.clientbound.CharacterSkillMacro.yaml` and append (matching the shape of `docs/packets/evidence/gms_v83/buddy.clientbound.BuddyAlreadyBuddy.yaml`):

```yaml
verifies:
  - libs/atlas-packet/character/clientbound/skill_macro_test.go#TestSkillMacroByteOutput
```

- [ ] **Step 4: Regenerate the matrix**

```bash
go run ./tools/packet-audit matrix
```

- [ ] **Step 5: Confirm the row promoted**

```bash
grep -n "MACRO_SYS_DATA_INIT" docs/packets/audits/STATUS.md
```
Expected: ✅ in every column where the op exists and a fixture was written. A column that did not promote is a failure to diagnose — check for an orphan marker (the marker's `ida=` matching neither the evidence record nor the report), a dangling evidence record (no report), or a `routedElsewhere && !routed` conflict, per the playbook's "Failure modes". gms_v92 is expected to fail with the routing conflict until Task 12 binds its writer opcode; that is the one acceptable red at this point, and it must be named explicitly in the commit message.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/evidence/ docs/packets/feature-families.yaml docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "test(task-226): verify MACRO_SYS_DATA_INIT byte output per version"
```

---

## Task 11: Verify the serverbound row (`SKILL_MACRO`)

**Files:**
- Create: `docs/packets/evidence/<version>/character.serverbound.CharacterSkillMacroHandle.yaml` (one per in-scope version)
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Interfaces:**
- Consumes: markers written in Task 7, reports copied in Task 9.

- [ ] **Step 1: Pin evidence, one version at a time**

```bash
go run ./tools/packet-audit evidence pin \
  --packet character/serverbound/CharacterSkillMacroHandle \
  --version <version key> \
  --ida "CMacroSysMan::FlushToSvr" \
  --category TIER1-FIXTURE
```

For any version whose registry primary fname is still a `sub_XXXXXX`, pass that name instead — `--ida` must match the export key.

- [ ] **Step 2: Add the `verifies:` field to each record**

```yaml
verifies:
  - libs/atlas-packet/character/serverbound/skill_macro_test.go#TestSkillMacroDecode
```

- [ ] **Step 3: Regenerate the matrix**

```bash
go run ./tools/packet-audit matrix
```

- [ ] **Step 4: Confirm the row promoted**

```bash
grep -n "SKILL_MACRO" docs/packets/audits/STATUS.md
```
Expected: ✅ for v72/v79/v83/v84; the four unrouted versions (v87, v92, v95, jms185) are expected to fail with the playbook's `routedElsewhere && !routed` conflict — "implements this op … but this version's template does not route its opcode, though another version's does" — until Task 12. That is a real template-wiring gap being reported correctly, and Task 12 closes it. Every other column must be ✅.

- [ ] **Step 5: Commit**

```bash
git add docs/packets/evidence/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "test(task-226): verify SKILL_MACRO decode per version"
```

---

## Task 12: Route the missing opcodes in the seed templates

The player-visible fix. It lands last so no new tenant is exposed to a decoder that was still under investigation (design §4).

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` (handlers)
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` (handlers + writers)
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` (handlers)
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` (handlers)
- Modify (conditionally): `template_gms_48_1.json`, `template_gms_61_1.json` — only if Task 4 returned `CORRECTED` for a cell on that version

- [ ] **Step 1: Add the four handler entries**

Insert each at its **opcode-sorted** position in the template's `handlers` array — never appended next to a semantically-related entry (`docs/packets/TEMPLATE_CONVENTIONS.md`, FR-5.3). The shape matches `template_gms_83_1.json:875-883`:

`template_gms_87_1.json`:
```json
      {
        "opCode": "0x71",
        "validator": "LoggedInValidator",
        "handler": "CharacterSkillMacroHandle",
        "fname": "CMacroSysMan::FlushToSvr",
        "services": [
          "channel"
        ]
      },
```

`template_gms_92_1.json`: identical but `"opCode": "0x79"`.
`template_gms_95_1.json`: identical but `"opCode": "0x7A"`.
`template_jms_185_1.json`: identical but `"opCode": "0x69"`.

Every entry carries `"validator": "LoggedInValidator"` — a handler with a missing or unknown validator is silently dropped at load (`bug_socket_handler_missing_validator_silently_dropped`) — and a `fname`. Use the version's registry `fname` verbatim; if Task 5 changed it for a version, use the changed value.

- [ ] **Step 2: Add the one missing writer entry**

In `template_gms_92_1.json`'s `writers` array, at its sorted position, matching `template_gms_83_1.json:2840-2847`:

```json
      {
        "opCode": "0x8B",
        "writer": "CharacterSkillMacro",
        "fname": "CWvsContext::OnMacroSysDataInit",
        "services": [
          "channel"
        ]
      },
```

A writer without `fname` fails seeding (`bug_seed_template_writers_require_fname`).

- [ ] **Step 3: Verify the JSON still parses and nothing else moved**

```bash
for f in services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_92_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
         services/atlas-configurations/seed-data/templates/template_jms_185_1.json; do
  python3 -m json.tool "$f" > /dev/null && echo "ok $f"
done
git diff --stat services/atlas-configurations/seed-data/templates/
```
Expected: four `ok` lines; the diffstat shows only added lines (5 entries = 5 insertions of 8–9 lines each), no deletions and no reordering.

- [ ] **Step 4: Run the template guards**

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```
Expected: exit 0 each. An opcode-order finding means an entry landed at the wrong index — move it, do not suppress the guard.

- [ ] **Step 5: Build and test atlas-configurations**

```bash
cd services/atlas-configurations && go build ./... && go vet ./... && go test -race ./...
```
Expected: clean.

- [ ] **Step 6: Regenerate the matrix and confirm the routing conflicts clear**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
grep -n "SKILL_MACRO\|MACRO_SYS_DATA_INIT" docs/packets/audits/STATUS.md
```
Expected: `matrix --check` exits 0, and both rows read ✅ in every column where the op exists, `⬜` only where Task 4 returned `CONFIRMED-NA` with a `feature-na-evidence.yaml` entry behind it. Any remaining ❌ is a stop-and-diagnose, not a "documented gap".

- [ ] **Step 7: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "fix(task-226): bind skill-macro handler on v87/v92/v95/jms185 and writer on v92"
```

---

## Task 13: Reconciliation doc, full guard sweep, and code review

**Files:**
- Create: `docs/tasks/task-226-skill-macro-version-coverage/live-tenant-reconciliation.md`
- Create: `docs/tasks/task-226-skill-macro-version-coverage/audit.md` (written by the reviewer agents)

- [ ] **Step 1: Write the live-tenant reconciliation input**

The seed templates only affect newly-seeded tenants; live tenants need a PATCH the user will run post-merge (PRD non-goal, FR-5.4). Write `live-tenant-reconciliation.md`:

```markdown
# Live-tenant reconciliation — skill macros (task-226)

Post-merge input for the live socket-config PATCH. Seed templates changed on
this branch do NOT reach already-provisioned tenants
(bug_new_opcodes_not_in_live_tenant_config). Procedure:
reference_reconcile_live_tenant_socket_to_template.

## Entries to add per version

### gms_87 — `handlers`
| opCode | handler | validator | fname | services |
|---|---|---|---|---|
| 0x71 | CharacterSkillMacroHandle | LoggedInValidator | CMacroSysMan::FlushToSvr | channel |

### gms_92 — `handlers`
| 0x79 | CharacterSkillMacroHandle | LoggedInValidator | CMacroSysMan::FlushToSvr | channel |

### gms_92 — `writers`
| opCode | writer | fname | services |
|---|---|---|---|
| 0x8B | CharacterSkillMacro | CWvsContext::OnMacroSysDataInit | channel |

### gms_95 — `handlers`
| 0x7A | CharacterSkillMacroHandle | LoggedInValidator | CMacroSysMan::FlushToSvr | channel |

### jms_185 — `handlers`
| 0x69 | CharacterSkillMacroHandle | LoggedInValidator | CMacroSysMan::FlushToSvr | channel |

<Add a section for gms_48 / gms_61 only if na-recheck.md returned CORRECTED for
that version. If it returned CONFIRMED-NA for all three cells, say so here
explicitly rather than leaving the reader to infer it.>

## Verification after the PATCH

Log in on each patched tenant, edit a macro in the Skill window, confirm, log
out and back in, and confirm the macro persisted with the shout flag in the
state it was set to. The shout flag specifically: task-226 changed its polarity
handling — see layout-derivation.md.
```

Use only repo-relative paths and placeholders — no `/home/<user>/...` (`feedback_no_home_paths_in_committed_docs`).

- [ ] **Step 2: Run the complete verification sweep**

From the worktree root:

```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-226-skill-macro-version-coverage
git branch --show-current       # must be task-226-skill-macro-version-coverage

(cd libs/atlas-packet          && go build ./... && go vet ./... && go test -race ./...)
(cd services/atlas-channel     && go build ./... && go vet ./... && go test -race ./...)
(cd services/atlas-configurations && go build ./... && go vet ./... && go test -race ./...)
(cd tools/packet-audit         && go build ./... && go vet ./...)

tools/lint.sh --check
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/skill-job-id-guard.sh
go run ./tools/packet-audit matrix --check
```

Expected: every command exits 0. Record the actual output — a claim of "clean" without the run is exactly the false-verified the PRD is written against (`superpowers:verification-before-completion`).

If `git status` shows `go.mod`/`go.sum` changes in any service, add `docker buildx bake atlas-channel` and `docker buildx bake atlas-configurations` to the sweep and run them (CLAUDE.md Build & Verification item 4). None are expected.

- [ ] **Step 3: Confirm the acceptance criteria one by one**

Walk `prd.md` §10 and check each box against a file that exists, quoting the evidence. The one criterion that may legitimately fail is "gms_83 and gms_84 encoded bytes are byte-identical to `main`" — if `layout-derivation.md` returned `INVERTED`, the v83/v84 shout byte changes by design (design §5). Record that deviation, with its IDA citation, in the PR description; do not quietly drop the criterion.

- [ ] **Step 4: Run the code review**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no atlas-ui files changed, so no frontend reviewer). Pin the reviewer subagents to Sonnet (`feedback_review_workflows_use_cheaper_model`) and ensure they run inside this worktree — never the main repo (`feedback_subagent_worktree_cwd`). Findings land in `docs/tasks/task-226-skill-macro-version-coverage/audit.md`.

Also dispatch `packet-completeness-critic` — this is a packet task and it catches the class-8 scope hole (a codec or version gate that moved without being declared).

- [ ] **Step 5: Address the findings**

Apply `superpowers:receiving-code-review`: verify each finding technically before implementing it, and push back on anything wrong rather than performatively agreeing. Re-run the Step 2 sweep after any code change.

- [ ] **Step 6: Confirm the worktree is clean**

```bash
git status --porcelain
```
Expected: empty. A subagent that wrote into the main repo shows up here as an absence — cross-check with `git -C ../.. status --porcelain` in the main repo too.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-226-skill-macro-version-coverage/
git commit -m "docs(task-226): live-tenant reconciliation input and review audit"
```

- [ ] **Step 8: Open the PR**

Push the branch and open the PR (`env -u GH_TOKEN -u GITHUB_TOKEN gh pr create`, per `feedback_gh_token_direnv`). The PR body must state:
- both matrix rows' before/after state,
- the shout-polarity verdict and its IDA citation, plus whether v83/v84 bytes changed as a result,
- the v48/v61 `n-a` outcome,
- the v72 registry correction,
- that live-tenant PATCHes are a post-merge step with `live-tenant-reconciliation.md` as the input.
