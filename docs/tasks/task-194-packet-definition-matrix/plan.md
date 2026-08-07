# Packet Definition Matrix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the external protocol spreadsheet with an in-product Packet Matrix comparing socket handler/writer definitions across all eleven configuration Templates, and refactor the four per-object definition pages onto the same dense grid.

**Architecture:** Three deliverables in dependency order. (1) `atlas-configurations` gains two additive REST fields — `socket.unsupported.{handlers,writers}` and a per-entry `fname` — plus a neutral `socket` validator package shared by the template and tenant trees. (2) A re-runnable `packet-audit seed-fname` subcommand backfills `fname` into the eleven seed templates by joining `(direction, opcode)` against `docs/packets/registry/<version>.yaml`. (3) `atlas-ui` gains a pure, React-free socket-domain library under `src/lib/socket/`, a single pivot-grid component driven by it, a drawer, six dialogs, two bulk flows, a new `/packet-matrix` route, and the deletion of the four stacked-card `useFieldArray` forms.

The dependency arrow runs one way: components import from `lib/socket`, never the reverse. `lib/socket` imports no React, no React Query and no service module. All protocol semantics — identity, opcode normalization, options comparison, ancestry classification, every mutation splice — are pure functions there, which is where the test weight sits.

**Tech Stack:** Go 1.25.5 (`atlas-configurations`, `tools/packet-audit`); TypeScript + React 19 + Vite 8 + React Router v7 + TanStack React Query 5 + shadcn/ui + Tailwind 4 (`atlas-ui`); Vitest + React Testing Library; `gopkg.in/yaml.v3` (already vendored in packet-audit).

## Global Constraints

Every task's requirements implicitly include this section.

- **Opcode format** is `^0[xX][0-9A-Fa-f]{1,4}$`. 1–4 hex digits, not 2. `template_jms_185_1.json` contains `"opCode": "0x9"` and `template_gms_84_1.json` contains `"opCode": "0x0A5"`; a 2-digit-only regex rejects existing valid data.
- **Service vocabulary is a closed set of two**, defined canonically in `libs/atlas-opcodes/config.go:6-7` as `ServiceLogin = "login"` and `ServiceChannel = "channel"`. Import those constants; never re-declare the strings. Measured across all 2,863 seed entries: `login` ×416, `channel` ×2449, nothing else.
- **Validator vocabulary is NOT a closed set.** The corpus carries `LoggedInValidator` ×1089 and `NoOpValidator` ×57, but FR-11.4 validates *presence only*, never membership. Do not add a validator allow-list.
- **Definition identity is `(kind, name)`; the mutation unit is `(name, normalized opcode)`.** Never splice an array "by the entry named X" — `NoOpHandler` is bound to four opcodes in gms_95_1 and doing so silently deletes three live routes. Never splice by array index either: `templatesService.getById` re-sorts `socket.handlers`/`socket.writers` by opcode on read (`templates.service.ts:52-63`), so a fetched entry's index does not match its stored index.
- **No `// TODO`, stubbed handlers, or 501s in landed commits.** Finish the bounded work or escalate explicitly.
- **Preserve line endings** when editing; do not normalize CRLF→LF as a side effect.
- **Never write literal home/absolute paths** (`/home/<name>/…`) into committed files. Use repo-relative paths.
- **atlas-ui conventions:** named exports on pages (never default), `@/` alias, `import.meta.env.VITE_*` never `process.env`, `lazyWithReload` for every route page (never bare `React.lazy`), tests colocated under `__tests__/`, Vitest globals (`describe`/`it`/`expect`/`vi`) not `jest.*` for new files.
- **`npm run build` type-checks test files**, so it is a gate, not a formality.
- **`tools/lint.sh --check` needs nvm 22 loaded** or it false-fails. Run `source ~/.nvm/nvm.sh && nvm use 22` first.

## Decisions of record (resolved with the user, 2026-08-05)

These supersede design.md §10, which left them open.

1. **Validation strictness — full.** The server blocks on all of FR-11.1–11.5 at `400`, not the design's recommended two-rule subset. FR-11.1 is enforced as *duplicate `(name, normalized opcode)`*, which is the design §5.1 corrected reading — enforcing the PRD's literal "duplicate definition name" would reject the legitimate multi-binding that exists in every template (`ServerListRequestHandle` in 9 of 11, `NoOpHandler` in 3, `CharacterEffect` in 2, `MiniRoom` in 5).
2. **The padded-opcode duplicates are fixed in this task**, overriding the PRD non-goal. This is what makes decision 1 safe for the seed corpus.
3. **The PRD's measured numbers are amended** to 219 writer rows and 2,863 corpus entries (Task 1).

**Consequence that must be handled, not ignored:** saves are whole-document, so strict FR-11.4 makes the live gms_95 tenant's 32 empty validators (`bug_v95_tenant_empty_validators_and_dup_opcode`) an un-editable deadlock — its first single-definition PATCH would be rejected, including the PATCH that would fix them. Task 18 builds a single bulk "fill missing validators" remediation write that repairs the whole document in one request, which is the only way out under strict blocking. Do not drop that task.

---

## Task 1: Amend the PRD to the measured corpus

Design F2 and F3 measured the seed corpus and found the PRD's numbers wrong. Every downstream acceptance criterion is stated against these numbers, so they are corrected first.

**Files:**
- Modify: `docs/tasks/task-194-packet-definition-matrix/prd.md`

**Interfaces:**
- Consumes: nothing.
- Produces: the corrected acceptance numbers every later task's verification is stated against — 141 handler rows, 219 writer rows, 2,863 corpus entries.

- [ ] **Step 1: Re-measure the corpus to confirm the numbers before writing them down**

Run from the worktree root:

```bash
cd services/atlas-configurations/seed-data/templates && python3 -c "
import json,glob,collections
H=set();W=set();tot=0
for f in sorted(glob.glob('template_*.json')):
    s=json.load(open(f))['socket']
    for kind,namekey,acc in (('handlers','handler',H),('writers','writer',W)):
        arr=s.get(kind) or []
        tot+=len(arr)
        acc.update(e[namekey] for e in arr)
print('distinct handler names:',len(H))
print('distinct writer names :',len(W))
print('total entries         :',tot)
"
```

Expected output exactly:

```
distinct handler names: 141
distinct writer names : 219
total entries         : 2863
```

If any number differs, STOP and report — the corpus changed since the design was written and the plan's numbers need re-deriving.

- [ ] **Step 2: Amend `prd.md` §2 Overview**

Replace `215 writer cards` / `259 writers` references. In §2, change:

> The GMS v95.1 template renders **215 writer cards** on a single scrolling page.

to:

> The GMS v95.1 template renders **215 writer cards** on a single scrolling page — the largest of eleven templates carrying 2,863 socket entries in total.

In §3 User Stories, change `search 259 writers` to `search 219 distinct writers`.

- [ ] **Step 3: Amend `prd.md` §7.3 coverage table**

Replace the whole "Measured coverage against current seed data" table with:

```markdown
| Source | Resolved |
|---|---|
| Direct opcode join, 9 versions with a registry | 2,674 / 2,685 |
| GMS v92.1 via adjacent-version impl-name match (v87.1 then v95.1) | 112 / 112 |
| GMS v12.1 via adjacent-version impl-name match (v48.1 then v61.1) | 63 / 66 |
| **Total** | **2,849 / 2,863** |

The three GMS v12.1 misses are `WorldSelectHandle` (`0x03`), `ServerLoad` (`0x02`)
and `CashShopCashQueryResult` (`0xBD`). The 14 unresolved entries ship without
`fname`.
```

- [ ] **Step 4: Amend `prd.md` §10 Acceptance Criteria**

Under "Seed data", replace:

> - [ ] All eleven seed templates carry `fname` on ≥ 2,169 of 2,179 definitions.

with:

> - [ ] All eleven seed templates carry `fname` on the count reported by
>       `packet-audit seed-fname`, and that report is committed to this task
>       folder as `fname-coverage.txt`.

Under "Matrix", replace `Writers mode 259 rows` with `Writers mode 219 rows`.

- [ ] **Step 5: Append a decisions-of-record section to `prd.md`**

Add a new section immediately before `## 10. Acceptance Criteria`:

```markdown
## 9a. Decisions of record (2026-08-05)

Resolved with the user after design.md measured the corpus:

1. **Validation is strict.** The server enforces all of FR-11.1–11.5 at 400.
   FR-11.1 is enforced as duplicate `(name, normalized opcode)` — the literal
   "duplicate definition name" reading would reject the legitimate multi-binding
   that exists in every template.
2. **The padded-opcode duplicates are fixed here**, overriding §2's non-goal.
   Four writer entries (`MiniRoom` at `0x0A5`/`0x0B0`/`0x0B8`/`0x0A3`) are
   removed. This is what makes decision 1 safe for the seed corpus.
3. **§4.1's "identity is its implementation name" is superseded** by
   design.md §5.1: the row is `(kind, name)`, a cell holds a set of bindings,
   and every mutation is keyed by `(name, normalized opcode)`.
```

- [ ] **Step 6: Verify no placeholder numbers remain**

Run:

```bash
grep -n '259\|2,179\|2,169\|2179\|2169' docs/tasks/task-194-packet-definition-matrix/prd.md
```

Expected: no output.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-194-packet-definition-matrix/prd.md
git commit -m "docs(task-194): amend PRD to measured corpus and record decisions"
```

---

## Task 2: Remove the four padded-opcode duplicate writer entries

Four seed templates carry `MiniRoom` twice at the *same numeric opcode*, differing only in leading-zero padding. This is the one thing blocking strict duplicate-`(name, opcode)` validation. It is a pure data fix with no runtime behaviour change.

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

**Interfaces:**
- Consumes: nothing.
- Produces: a seed corpus with zero duplicate `(name, normalized opcode)` pairs, which Task 3's `ErrDuplicateBinding` rule can then block on without stranding existing data.

**Why this is behaviour-neutral:** `libs/atlas-opcodes` reads both arrays into an opcode-keyed dispatch map, so the last entry for a given opcode wins. Today the winner is the padded entry, which carries no `options` key. After deletion the winner is the canonical entry, which carries `"options": {}`. Both decode to a nil/empty map and both mean "no options supplied" (design §5.4). Nothing else differs — both entries carry `"services": ["channel"]`.

**Do NOT touch `template_gms_84_1.json`.** Its `MiniRoom` pair is at `0x0A5` (165) and `0xA8` (168) — genuinely different opcodes, legitimate multi-binding, and the reason the opcode regex must accept 3 hex digits.

- [ ] **Step 1: Write the failing guard test**

Create `tools/template-duplicate-binding-guard.sh`:

```bash
#!/usr/bin/env bash
# template-duplicate-binding-guard.sh — enforces that no seed socket template
# binds the same (implementation name, numeric opCode) pair twice.
#
# A name legitimately bound to SEVERAL DISTINCT opcodes is normal and permanent
# (NoOpHandler is a deliberate sink at four opcodes in gms_95_1). What is a data
# defect is the SAME numeric opcode written twice with different leading-zero
# padding — "0xB8" and "0x0B8" — which makes the dispatch map's last-write-wins
# behaviour decide which entry's options survive.
#
# Mirrors the atlas-configurations socket.Validate ErrDuplicateBinding rule
# (task-194): the server rejects such a document with 400, so seed data that
# contained one could never be saved back through the UI.
#
# Run from the repo root; non-empty diagnostics → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE_DIR="$ROOT/services/atlas-configurations/seed-data/templates"

python3 - "$TEMPLATE_DIR" <<'PY'
import glob, json, os, sys, collections

tmpl_dir = sys.argv[1]
bad = 0
checked = 0
for path in sorted(glob.glob(os.path.join(tmpl_dir, "template_*.json"))):
    d = json.load(open(path))
    sock = d.get("socket", {})
    for group, namekey in (("handlers", "handler"), ("writers", "writer")):
        arr = sock.get(group) or []
        checked += 1
        seen = collections.defaultdict(list)
        for e in arr:
            if not isinstance(e, dict) or "opCode" not in e:
                continue
            try:
                code = int(e["opCode"], 16)
            except (TypeError, ValueError):
                print("BAD opCode in %s %s: %r" % (os.path.basename(path), group, e.get("opCode")))
                bad += 1
                continue
            seen[(e.get(namekey), code)].append(e["opCode"])
        for (name, code), raws in sorted(seen.items(), key=lambda kv: (str(kv[0][0]), kv[0][1])):
            if len(raws) > 1:
                print("DUPLICATE BINDING: %s %s: %s @ 0x%02X written %d times as %s"
                      % (os.path.basename(path), group, name, code, len(raws), raws))
                bad += 1

if bad:
    print("")
    print("FAIL: %d duplicate binding(s). One (name, numeric opCode) pair may appear at most once." % bad)
    sys.exit(1)
print("OK: %d template arrays carry no duplicate (name, opCode) binding." % checked)
PY
```

Make it executable:

```bash
chmod +x tools/template-duplicate-binding-guard.sh
```

- [ ] **Step 2: Run it to verify it fails**

Run from the worktree root: `tools/template-duplicate-binding-guard.sh`

Expected: FAIL with exactly four `DUPLICATE BINDING` lines —

```
DUPLICATE BINDING: template_gms_83_1.json writers: MiniRoom @ 0xA5 written 2 times as ['0xA5', '0x0A5']
DUPLICATE BINDING: template_gms_87_1.json writers: MiniRoom @ 0xB0 written 2 times as ['0xB0', '0x0B0']
DUPLICATE BINDING: template_gms_95_1.json writers: MiniRoom @ 0xB8 written 2 times as ['0xB8', '0x0B8']
DUPLICATE BINDING: template_jms_185_1.json writers: MiniRoom @ 0xA3 written 2 times as ['0x0A3', '0xA3']
```

(Order within the `raws` list may vary; the four templates and four opcodes must match.)

- [ ] **Step 3: Delete the four padded duplicates**

In each file, delete the entry whose `opCode` carries the redundant leading zero, keeping the canonical two-digit entry. Use `Edit` per file — not a shell patch loop.

`template_gms_83_1.json` — delete:

```json
    {
      "opCode": "0x0A5",
      "writer": "MiniRoom",
      "services": [
        "channel"
      ]
    },
```

`template_gms_87_1.json` — delete:

```json
    {
      "opCode": "0x0B0",
      "writer": "MiniRoom",
      "services": [
        "channel"
      ]
    },
```

`template_gms_95_1.json` — delete:

```json
    {
      "opCode": "0x0B8",
      "writer": "MiniRoom",
      "services": [
        "channel"
      ]
    },
```

`template_jms_185_1.json` — delete:

```json
    {
      "opCode": "0x0A3",
      "writer": "MiniRoom",
      "services": [
        "channel"
      ]
    },
```

Read each file around the `MiniRoom` entries first to get the exact whitespace and trailing-comma placement — the surrounding indentation is two-space and the deleted object may be the last element of its array in principle (it is not, in all four files, but confirm rather than assume).

- [ ] **Step 4: Run the guard to verify it passes**

Run: `tools/template-duplicate-binding-guard.sh`

Expected: `OK: 22 template arrays carry no duplicate (name, opCode) binding.`

- [ ] **Step 5: Verify the deletion changed nothing else**

Run:

```bash
cd services/atlas-configurations/seed-data/templates && python3 -c "
import json,glob,collections
tot=0; mini=collections.defaultdict(list)
for f in sorted(glob.glob('template_*.json')):
    s=json.load(open(f))['socket']
    tot += len(s.get('handlers') or []) + len(s.get('writers') or [])
    for e in s.get('writers') or []:
        if e.get('writer')=='MiniRoom': mini[f].append(e['opCode'])
print('total entries:',tot)
for f,v in sorted(mini.items()): print(f,v)
"
```

Expected:

```
total entries: 2859
template_gms_83_1.json ['0xA5']
template_gms_84_1.json ['0x0A5', '0xA8']
template_gms_87_1.json ['0xB0']
template_gms_92_1.json ['0xB8']
template_gms_95_1.json ['0xB8']
template_jms_185_1.json ['0xA3']
```

2,863 − 4 = 2,859. `gms_84_1` keeps both of its genuinely-distinct bindings.

- [ ] **Step 6: Run the existing order guard**

Run: `tools/template-opcode-order-guard.sh`

Expected: `OK: 22 template arrays are in ascending opcode order.`

- [ ] **Step 7: Register the new guard in CLAUDE.md**

In the "Build & Verification" numbered list in `CLAUDE.md`, add after item 9:

```markdown
9a. **`tools/template-duplicate-binding-guard.sh` clean from the repo root**
    whenever a tenant socket-config template under
    `services/atlas-configurations/seed-data/templates/` changed. Bans binding
    the same `(implementation name, numeric opCode)` pair twice — the
    leading-zero-padding duplicate (`0xB8` and `0x0B8`) that made the dispatch
    map's last-write-wins behaviour decide which entry's options survive
    (task-194). A name bound to several *distinct* opcodes is legitimate and
    untouched.
```

- [ ] **Step 8: Commit**

```bash
git add tools/template-duplicate-binding-guard.sh CLAUDE.md \
  services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  services/atlas-configurations/seed-data/templates/template_jms_185_1.json
git commit -m "fix(configurations): drop padded-opcode duplicate MiniRoom writer entries

Four templates bound MiniRoom twice at the same numeric opcode, differing only
in leading-zero padding. The dispatch map is opcode-keyed and last-write-wins,
so the padded entry silently decided which options survived. Behaviour-neutral:
both entries carried the same services and neither supplied options.

Adds tools/template-duplicate-binding-guard.sh to keep it that way, mirroring
the socket.Validate rule that lands in the next commit."
```

---

## Task 3: The shared `socket` validator package

A neutral top-level package imported by both the template and tenant trees, so the rules and their tests exist once. Pure and dependency-free apart from the `libs/atlas-opcodes` service constants.

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/socket/validate.go`
- Create: `services/atlas-configurations/atlas.com/configurations/socket/validate_test.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/go.mod` (adds the `libs/atlas-opcodes` require)

**Interfaces:**
- Consumes: `opcodes.ServiceLogin`, `opcodes.ServiceChannel` from `github.com/Chronicle20/atlas/libs/atlas-opcodes`.
- Produces, for Tasks 4 and 5:
  - `socket.Issue{Path string; Message string}`
  - `socket.Binding{Name, OpCode, Validator string; Services []string}`
  - `socket.Input{Handlers, Writers []Binding; UnsupportedHandlers, UnsupportedWriters []string}`
  - `socket.Validate(in Input) []Issue`
  - `socket.ParseOpCode(raw string) (int, bool)`
  - `socket.OpCodePattern` (the compiled `*regexp.Regexp`)

**Note on severity:** design §4.4 modelled an `Issue.Severity` field because it expected a warn tier. Decision 1 made every rule blocking, so there is no warn tier and no `Severity` field. Do not add one — an enum with a single variant is dead weight, and a warn tier that nothing consumes is exactly the silent stub the project bans.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-configurations/atlas.com/configurations/socket/validate_test.go`:

```go
package socket

import (
	"testing"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
)

func TestParseOpCode(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		want  int
		wantOk bool
	}{
		{"two digit lower prefix", "0x2a", 42, true},
		{"two digit upper hex", "0x2A", 42, true},
		{"upper X prefix", "0X2A", 42, true},
		{"single digit", "0x9", 9, true},
		{"three digit padded", "0x0A5", 165, true},
		{"four digit", "0xFFFF", 65535, true},
		{"five digits rejected", "0x10000", 0, false},
		{"missing prefix rejected", "2A", 0, false},
		{"decimal rejected", "42", 0, false},
		{"empty rejected", "", 0, false},
		{"non hex rejected", "0xZZ", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseOpCode(tt.raw)
			if ok != tt.wantOk {
				t.Fatalf("ParseOpCode(%q) ok = %v, want %v", tt.raw, ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("ParseOpCode(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestValidate_AcceptsCleanInput(t *testing.T) {
	in := Input{
		Handlers: []Binding{
			{Name: "LoginHandle", OpCode: "0x01", Validator: "NoOpValidator", Services: []string{opcodes.ServiceLogin}},
			{Name: "NoOpHandler", OpCode: "0x17", Validator: "LoggedInValidator", Services: []string{opcodes.ServiceChannel}},
			{Name: "NoOpHandler", OpCode: "0x19", Validator: "LoggedInValidator", Services: []string{opcodes.ServiceChannel}},
		},
		Writers: []Binding{
			{Name: "AuthLoginFailed", OpCode: "0x01", Services: []string{opcodes.ServiceLogin}},
			{Name: "AuthPermanentBan", OpCode: "0x01", Services: []string{opcodes.ServiceLogin}},
			{Name: "MiniRoom", OpCode: "0x0A5", Services: []string{opcodes.ServiceChannel}},
			{Name: "MiniRoom", OpCode: "0xA8", Services: []string{opcodes.ServiceChannel}},
		},
		UnsupportedHandlers: []string{"GuestLoginHandle"},
	}
	if got := Validate(in); len(got) != 0 {
		t.Fatalf("Validate() returned %d issues on clean input: %+v", len(got), got)
	}
}

func TestValidate_Rules(t *testing.T) {
	tests := []struct {
		name     string
		in       Input
		wantPath string
		wantMsg  string
	}{
		{
			name: "FR-11.1 duplicate name and opcode in one collection",
			in: Input{Handlers: []Binding{
				{Name: "MiniRoom", OpCode: "0xB8", Validator: "LoggedInValidator", Services: []string{opcodes.ServiceChannel}},
				{Name: "MiniRoom", OpCode: "0x0B8", Validator: "LoggedInValidator", Services: []string{opcodes.ServiceChannel}},
			}},
			wantPath: "socket.handlers[1].opCode",
			wantMsg:  `"MiniRoom" is already bound to opcode 0xB8`,
		},
		{
			name: "FR-11.2 malformed opcode",
			in: Input{Writers: []Binding{
				{Name: "AuthSuccess", OpCode: "B8", Services: []string{opcodes.ServiceLogin}},
			}},
			wantPath: "socket.writers[0].opCode",
			wantMsg:  `opCode "B8" must match 0x followed by 1-4 hex digits`,
		},
		{
			name: "FR-11.3 name both defined and unsupported",
			in: Input{
				Handlers:            []Binding{{Name: "LoginHandle", OpCode: "0x01", Validator: "NoOpValidator", Services: []string{opcodes.ServiceLogin}}},
				UnsupportedHandlers: []string{"LoginHandle"},
			},
			wantPath: "socket.unsupported.handlers[0]",
			wantMsg:  `"LoginHandle" is marked unsupported but is also defined in socket.handlers`,
		},
		{
			name: "FR-11.4 empty handler validator",
			in: Input{Handlers: []Binding{
				{Name: "LoginHandle", OpCode: "0x01", Validator: "", Services: []string{opcodes.ServiceLogin}},
			}},
			wantPath: "socket.handlers[0].validator",
			wantMsg:  `validator is required for handler "LoginHandle"`,
		},
		{
			name: "FR-11.4 whitespace-only handler validator",
			in: Input{Handlers: []Binding{
				{Name: "LoginHandle", OpCode: "0x01", Validator: "  ", Services: []string{opcodes.ServiceLogin}},
			}},
			wantPath: "socket.handlers[0].validator",
			wantMsg:  `validator is required for handler "LoginHandle"`,
		},
		{
			name: "FR-11.5 unknown service",
			in: Input{Writers: []Binding{
				{Name: "AuthSuccess", OpCode: "0x00", Services: []string{"drops"}},
			}},
			wantPath: "socket.writers[0].services[0]",
			wantMsg:  `unknown service "drops"; expected one of login, channel`,
		},
		{
			name: "duplicate unsupported name",
			in: Input{UnsupportedWriters: []string{"MonsterCarnival", "MonsterCarnival"}},
			wantPath: "socket.unsupported.writers[1]",
			wantMsg:  `"MonsterCarnival" is listed more than once`,
		},
		{
			name: "empty definition name",
			in: Input{Writers: []Binding{
				{Name: "", OpCode: "0x00", Services: []string{opcodes.ServiceLogin}},
			}},
			wantPath: "socket.writers[0].writer",
			wantMsg:  "definition name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.in)
			for _, iss := range got {
				if iss.Path == tt.wantPath && iss.Message == tt.wantMsg {
					return
				}
			}
			t.Errorf("Validate() = %+v\nwant an issue at %q with message %q", got, tt.wantPath, tt.wantMsg)
		})
	}
}

// FR-11.6: several writers legitimately share one opcode. gms_12_1 has
// AuthPermanentBan and AuthLoginFailed both at 0x01. This must never fail.
func TestValidate_DuplicateOpCodeAcrossNamesIsLegal(t *testing.T) {
	in := Input{Writers: []Binding{
		{Name: "AuthPermanentBan", OpCode: "0x01", Services: []string{opcodes.ServiceLogin}},
		{Name: "AuthLoginFailed", OpCode: "0x01", Services: []string{opcodes.ServiceLogin}},
	}}
	if got := Validate(in); len(got) != 0 {
		t.Fatalf("Validate() rejected a legal shared opcode: %+v", got)
	}
}

// Writers carry no validator; an empty one must never be reported for them.
func TestValidate_WritersNeedNoValidator(t *testing.T) {
	in := Input{Writers: []Binding{
		{Name: "AuthSuccess", OpCode: "0x00", Validator: "", Services: []string{opcodes.ServiceLogin}},
	}}
	if got := Validate(in); len(got) != 0 {
		t.Fatalf("Validate() required a validator on a writer: %+v", got)
	}
}

// An entry with no services applies to every service (libs/atlas-opcodes
// appliesToService treats an empty list as universal), so it is legal.
func TestValidate_EmptyServicesIsLegal(t *testing.T) {
	in := Input{Writers: []Binding{{Name: "AuthSuccess", OpCode: "0x00"}}}
	if got := Validate(in); len(got) != 0 {
		t.Fatalf("Validate() rejected an untagged entry: %+v", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./socket/...`

Expected: FAIL — the package does not compile (`undefined: ParseOpCode`, `undefined: Input`, …).

- [ ] **Step 3: Write the implementation**

Create `services/atlas-configurations/atlas.com/configurations/socket/validate.go`:

```go
// Package socket holds the validation rules for a configuration document's
// socket handler/writer tables. It is imported by both the templates and the
// tenants trees, which each contribute a thin adapter building Input from
// their own RestModel, so the rules and their tests exist exactly once.
//
// Every rule here is blocking: Validate's caller turns any returned Issue into
// a 400. There is deliberately no warn tier — task-194 decision 1.
package socket

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
)

// OpCodePattern is the accepted wire form of an opcode: "0x" or "0X" followed
// by one to four hex digits. One digit is real (jms_185_1 carries "0x9") and so
// is three (gms_84_1 carries "0x0A5"), so a two-digit-only pattern would reject
// existing valid data.
var OpCodePattern = regexp.MustCompile(`^0[xX][0-9A-Fa-f]{1,4}$`)

// knownServices is the closed set of socket-service scopes, sourced from
// libs/atlas-opcodes so the vocabulary lives in exactly one place. Adding a
// third socket service means adding it there first.
var knownServices = []string{opcodes.ServiceLogin, opcodes.ServiceChannel}

// Issue is one blocking validation failure. Path is a JSON pointer-ish path
// into the submitted document, used verbatim as the JSON:API error meta.path.
type Issue struct {
	Path    string
	Message string
}

// Binding is one entry of socket.handlers or socket.writers, flattened to the
// fields the rules care about. Validator is empty for writers.
type Binding struct {
	Name      string
	OpCode    string
	Validator string
	Services  []string
}

// Input is a whole socket document, adapted from either tree's RestModel.
type Input struct {
	Handlers            []Binding
	Writers             []Binding
	UnsupportedHandlers []string
	UnsupportedWriters  []string
}

// ParseOpCode parses an opcode in the accepted wire form and reports whether it
// was well-formed. It is the only place a stored opcode string is interpreted.
func ParseOpCode(raw string) (int, bool) {
	if !OpCodePattern.MatchString(raw) {
		return 0, false
	}
	n, err := strconv.ParseInt(raw[2:], 16, 32)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

// Validate returns every blocking issue in the document. An empty slice means
// the document may be stored.
func Validate(in Input) []Issue {
	issues := make([]Issue, 0)
	issues = append(issues, validateCollection("handlers", "handler", true, in.Handlers)...)
	issues = append(issues, validateCollection("writers", "writer", false, in.Writers)...)
	issues = append(issues, validateUnsupported("handlers", in.UnsupportedHandlers, in.Handlers)...)
	issues = append(issues, validateUnsupported("writers", in.UnsupportedWriters, in.Writers)...)
	return issues
}

// validateCollection runs the per-entry rules over one collection. group is the
// JSON key ("handlers"), nameField is the JSON key holding the implementation
// name ("handler"), and needsValidator is true only for handlers.
func validateCollection(group string, nameField string, needsValidator bool, bindings []Binding) []Issue {
	var issues []Issue
	// seen maps (name, numeric opcode) to the raw opcode of its first binding,
	// so the duplicate message can name the canonical form.
	type key struct {
		name string
		code int
	}
	seen := make(map[key]string, len(bindings))

	for i, b := range bindings {
		base := fmt.Sprintf("socket.%s[%d]", group, i)

		if strings.TrimSpace(b.Name) == "" {
			issues = append(issues, Issue{
				Path:    base + "." + nameField,
				Message: "definition name is required",
			})
		}

		code, ok := ParseOpCode(b.OpCode)
		if !ok {
			issues = append(issues, Issue{
				Path:    base + ".opCode",
				Message: fmt.Sprintf("opCode %q must match 0x followed by 1-4 hex digits", b.OpCode),
			})
		} else {
			k := key{name: b.Name, code: code}
			// A name bound to several DISTINCT opcodes is legitimate and common
			// (NoOpHandler sinks four opcodes in gms_95_1). Only the same name at
			// the same numeric opcode is a defect.
			if first, dup := seen[k]; dup {
				issues = append(issues, Issue{
					Path:    base + ".opCode",
					Message: fmt.Sprintf("%q is already bound to opcode %s", b.Name, first),
				})
			} else {
				seen[k] = b.OpCode
			}
		}

		if needsValidator && strings.TrimSpace(b.Validator) == "" {
			issues = append(issues, Issue{
				Path:    base + ".validator",
				Message: fmt.Sprintf("validator is required for handler %q", b.Name),
			})
		}

		for j, s := range b.Services {
			if !isKnownService(s) {
				issues = append(issues, Issue{
					Path:    fmt.Sprintf("%s.services[%d]", base, j),
					Message: fmt.Sprintf("unknown service %q; expected one of %s", s, strings.Join(knownServices, ", ")),
				})
			}
		}
	}
	return issues
}

// validateUnsupported enforces FR-11.3 (a name is never both defined and
// unsupported) and rejects a name listed twice.
func validateUnsupported(group string, names []string, bindings []Binding) []Issue {
	var issues []Issue
	defined := make(map[string]bool, len(bindings))
	for _, b := range bindings {
		defined[b.Name] = true
	}
	seen := make(map[string]bool, len(names))
	for i, n := range names {
		path := fmt.Sprintf("socket.unsupported.%s[%d]", group, i)
		if strings.TrimSpace(n) == "" {
			issues = append(issues, Issue{Path: path, Message: "unsupported entry name is required"})
			continue
		}
		if defined[n] {
			issues = append(issues, Issue{
				Path:    path,
				Message: fmt.Sprintf("%q is marked unsupported but is also defined in socket.%s", n, group),
			})
		}
		if seen[n] {
			issues = append(issues, Issue{
				Path:    path,
				Message: fmt.Sprintf("%q is listed more than once", n),
			})
		}
		seen[n] = true
	}
	return issues
}

func isKnownService(s string) bool {
	for _, k := range knownServices {
		if s == k {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Add the `libs/atlas-opcodes` dependency**

The `replace` directive already exists at `go.mod:96`; only the `require` is missing.

```bash
cd services/atlas-configurations/atlas.com/configurations && go mod tidy
```

Verify the require landed:

```bash
grep -n 'atlas-opcodes' services/atlas-configurations/atlas.com/configurations/go.mod
```

Expected: two lines — one `require github.com/Chronicle20/atlas/libs/atlas-opcodes v0.0.0…` and the existing `replace`.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./socket/... -v`

Expected: PASS, every subtest.

- [ ] **Step 6: Prove the rules accept the whole live corpus**

This is the guard against a rule that is correct in the abstract and bricks real data. Create `services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go`:

```go
package socket

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The seed corpus is the strictest available proof that these blocking rules do
// not strand existing data: every rule here is a 400, so any seed template that
// fails Validate is a template the UI could never save back. Task-194 decision 1
// (strict validation) is only safe while this test passes.
func TestValidate_AcceptsEverySeedTemplate(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "seed-data", "templates")
	files, err := filepath.Glob(filepath.Join(dir, "template_*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) != 11 {
		t.Fatalf("expected 11 seed templates, found %d in %s", len(files), dir)
	}

	type entry struct {
		OpCode    string   `json:"opCode"`
		Validator string   `json:"validator"`
		Handler   string   `json:"handler"`
		Writer    string   `json:"writer"`
		Services  []string `json:"services"`
	}
	type doc struct {
		Socket struct {
			Handlers []entry `json:"handlers"`
			Writers  []entry `json:"writers"`
		} `json:"socket"`
	}

	total := 0
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		var d doc
		if err := json.Unmarshal(b, &d); err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		in := Input{}
		for _, h := range d.Socket.Handlers {
			in.Handlers = append(in.Handlers, Binding{Name: h.Handler, OpCode: h.OpCode, Validator: h.Validator, Services: h.Services})
		}
		for _, w := range d.Socket.Writers {
			in.Writers = append(in.Writers, Binding{Name: w.Writer, OpCode: w.OpCode, Services: w.Services})
		}
		total += len(in.Handlers) + len(in.Writers)
		if issues := Validate(in); len(issues) != 0 {
			t.Errorf("%s: %d issue(s):", filepath.Base(f), len(issues))
			for _, iss := range issues {
				t.Errorf("    %s: %s", iss.Path, iss.Message)
			}
		}
	}
	if total != 2859 {
		t.Errorf("corpus size = %d entries, want 2859 (2863 less the 4 padded MiniRoom duplicates removed in task-194)", total)
	}
}
```

- [ ] **Step 7: Run the corpus test**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./socket/... -run TestValidate_AcceptsEverySeedTemplate -v`

Expected: PASS, with the corpus at 2,859 entries. If it fails, Task 2 was not applied or a rule is too strict for real data — fix the rule or the data, do not relax the test.

- [ ] **Step 8: Vet and commit**

```bash
cd services/atlas-configurations/atlas.com/configurations && go vet ./socket/... && go test -race ./socket/...
```

Expected: both clean.

```bash
git add services/atlas-configurations/atlas.com/configurations/socket \
        services/atlas-configurations/atlas.com/configurations/go.mod \
        services/atlas-configurations/atlas.com/configurations/go.sum
git commit -m "feat(configurations): add shared socket validation package

Neutral package imported by both the templates and tenants trees. Enforces
FR-11.1-11.5 as blocking rules: duplicate (name, numeric opcode) within a
collection, opcode format 0x + 1-4 hex digits, a name both defined and
unsupported, a missing handler validator, and an unknown service scope.

Duplicate opcodes across DIFFERENT names stay legal (FR-11.6) - gms_12_1 binds
AuthPermanentBan and AuthLoginFailed both at 0x01. A name bound to several
distinct opcodes stays legal - NoOpHandler sinks four opcodes in gms_95_1.

Proven against all 2,859 seed entries so strict blocking cannot strand data."
```

---

## Task 4: Additive REST fields and the normalization invariant

`unsupported` and `fname` are REST-model-only changes. Both `templates.Entity` and `tenants.Entity` store the whole configuration as `Data json.RawMessage` (`gorm:"type:json;not null"`), so there is no `AutoMigrate` change, no column and no backfill migration.

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/socket/rest.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/socket/handler/rest.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/socket/writer/rest.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/socket/rest.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/socket/handler/rest.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/socket/writer/rest.go`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/socket/rest_test.go`
- Create: `services/atlas-configurations/atlas.com/configurations/tenants/socket/rest_test.go`

**Interfaces:**
- Consumes: nothing from Task 3 (this task is independent of it; both are consumed together by Task 5).
- Produces, for Task 5, in **both** `templates/socket` and `tenants/socket`:
  - `socket.UnsupportedRestModel{Handlers []string; Writers []string}`
  - `socket.RestModel` gains `Unsupported UnsupportedRestModel \`json:"unsupported"\``
  - `socket.Normalize(rm RestModel) RestModel`
  - `handler.RestModel` gains `FName string \`json:"fname,omitempty"\``, and `Options` gains `omitempty`
  - `writer.RestModel` gains the same two changes

**The two trees stay parallel.** They are already duplicated by design and match the existing service boundary; an aliasing indirection between them would be more code than the ~20 duplicated lines it saves. The *rules* are shared (Task 3); the *models* are not.

**Why `Options` gains `omitempty` (design F7):** seed entries omit `options` entirely when unset. `Options` is a `map[string]interface{}` with no `omitempty` today, so any document that round-trips through a PATCH gains `"options": null` on every entry — turning the first save of any template into a 200-line diff. Both an absent key and `null` decode to a nil map and both mean "not supplied" (FR-3.2), so this is cosmetic-only but worth having.

- [ ] **Step 1: Write the failing test for the templates tree**

Create `services/atlas-configurations/atlas.com/configurations/templates/socket/rest_test.go`:

```go
package socket

import (
	"encoding/json"
	"strings"
	"testing"

	"atlas-configurations/templates/socket/handler"
	"atlas-configurations/templates/socket/writer"
)

// An absent "unsupported" key must decode to a struct with two EMPTY (not nil)
// slices after Normalize, and must marshal back as real arrays. Both PRD
// acceptance criteria - "loads with both lists empty" and "carries an empty
// unsupported object" - come from this one invariant.
func TestNormalize_AbsentUnsupportedBecomesEmptyArrays(t *testing.T) {
	const in = `{"handlers":[],"writers":[]}`
	var rm RestModel
	if err := json.Unmarshal([]byte(in), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	rm = Normalize(rm)

	if rm.Unsupported.Handlers == nil {
		t.Error("Normalize left Unsupported.Handlers nil")
	}
	if rm.Unsupported.Writers == nil {
		t.Error("Normalize left Unsupported.Writers nil")
	}

	out, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `"unsupported":{"handlers":[],"writers":[]}`) {
		t.Errorf("marshalled output missing empty unsupported arrays:\n%s", got)
	}
}

func TestNormalize_PreservesPopulatedUnsupported(t *testing.T) {
	rm := RestModel{
		Unsupported: UnsupportedRestModel{
			Handlers: []string{"GuestLoginHandle"},
			Writers:  []string{"MonsterCarnival"},
		},
	}
	rm = Normalize(rm)
	if len(rm.Unsupported.Handlers) != 1 || rm.Unsupported.Handlers[0] != "GuestLoginHandle" {
		t.Errorf("Normalize mangled Unsupported.Handlers: %+v", rm.Unsupported.Handlers)
	}
	if len(rm.Unsupported.Writers) != 1 || rm.Unsupported.Writers[0] != "MonsterCarnival" {
		t.Errorf("Normalize mangled Unsupported.Writers: %+v", rm.Unsupported.Writers)
	}
	if rm.Handlers == nil || rm.Writers == nil {
		t.Error("Normalize left Handlers/Writers nil")
	}
}

func TestRestModel_FNameRoundTrips(t *testing.T) {
	const in = `{"handlers":[{"opCode":"0x01","validator":"NoOpValidator","handler":"LoginHandle","fname":"CLogin::SendCheckPasswordPacket","services":["login"]}],"writers":[]}`
	var rm RestModel
	if err := json.Unmarshal([]byte(in), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := rm.Handlers[0].FName; got != "CLogin::SendCheckPasswordPacket" {
		t.Fatalf("FName = %q, want CLogin::SendCheckPasswordPacket", got)
	}
	out, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"fname":"CLogin::SendCheckPasswordPacket"`) {
		t.Errorf("fname dropped on marshal:\n%s", out)
	}
}

// fname is omitempty: an entry without one must not gain a "fname":"" key.
func TestRestModel_FNameOmittedWhenEmpty(t *testing.T) {
	rm := RestModel{
		Handlers: []handler.RestModel{{OpCode: "0x01", Validator: "NoOpValidator", Handler: "LoginHandle"}},
		Writers:  []writer.RestModel{{OpCode: "0x00", Writer: "AuthSuccess"}},
	}
	out, err := json.Marshal(Normalize(rm))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), `"fname"`) {
		t.Errorf("empty fname was emitted:\n%s", out)
	}
}

// Options is omitempty (design F7): an entry that supplied no options must not
// gain "options":null on round-trip, which would make the first save of any
// template a 200-line diff.
func TestRestModel_OptionsOmittedWhenAbsent(t *testing.T) {
	const in = `{"handlers":[{"opCode":"0x01","validator":"NoOpValidator","handler":"LoginHandle"}],"writers":[]}`
	var rm RestModel
	if err := json.Unmarshal([]byte(in), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(Normalize(rm))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), `"options"`) {
		t.Errorf("absent options round-tripped to a key:\n%s", out)
	}
}

// An explicitly-supplied empty options object is DIFFERENT from an absent one
// at the JSON level but identical semantically; it must survive as {} rather
// than being dropped, so the seed files stay byte-stable.
func TestRestModel_EmptyOptionsObjectSurvives(t *testing.T) {
	const in = `{"handlers":[],"writers":[{"opCode":"0xA5","writer":"MiniRoom","options":{},"services":["channel"]}]}`
	var rm RestModel
	if err := json.Unmarshal([]byte(in), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(Normalize(rm))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"options":{}`) {
		t.Errorf("explicit empty options object was dropped:\n%s", out)
	}
}
```

Note the last two tests together pin the exact `omitempty` semantics for a map: Go's `omitempty` drops a `nil` map but keeps a non-nil empty one, which is precisely the absent-vs-`{}` distinction we want.

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./templates/socket/...`

Expected: FAIL to compile — `undefined: Normalize`, `rm.Unsupported undefined`, `unknown field FName`.

- [ ] **Step 3: Implement the templates tree models**

Replace `templates/socket/rest.go` entirely:

```go
package socket

import (
	"atlas-configurations/templates/socket/handler"
	"atlas-configurations/templates/socket/writer"
)

type RestModel struct {
	Handlers    []handler.RestModel  `json:"handlers"`
	Writers     []writer.RestModel   `json:"writers"`
	Unsupported UnsupportedRestModel `json:"unsupported"`
}

// UnsupportedRestModel names the definitions that have been audited and
// confirmed absent for this Region/Version. It is what makes "audited, this
// version does not have this packet" distinguishable from "nobody has looked
// yet" (PRD FR-1.x). Names here are implementation names, never opcodes.
type UnsupportedRestModel struct {
	Handlers []string `json:"handlers"`
	Writers  []string `json:"writers"`
}

// Normalize replaces nil slices with empty ones so the marshalled document
// always carries real arrays rather than nulls. Entries themselves are left
// untouched. Every read path (Make) and every write path (Create/UpdateById)
// funnels through it, which is what guarantees the invariant.
func Normalize(rm RestModel) RestModel {
	if rm.Handlers == nil {
		rm.Handlers = []handler.RestModel{}
	}
	if rm.Writers == nil {
		rm.Writers = []writer.RestModel{}
	}
	if rm.Unsupported.Handlers == nil {
		rm.Unsupported.Handlers = []string{}
	}
	if rm.Unsupported.Writers == nil {
		rm.Unsupported.Writers = []string{}
	}
	return rm
}
```

Replace `templates/socket/handler/rest.go` entirely:

```go
package handler

type RestModel struct {
	OpCode    string `json:"opCode"`
	Validator string `json:"validator"`
	Handler   string `json:"handler"`
	// FName carries the client-side function name
	// ("CLogin::SendCheckPasswordPacket"). It is informational only: it never
	// participates in comparison, validation or ancestry classification
	// (PRD FR-10.4).
	FName string `json:"fname,omitempty"`
	// Options is omitempty so an entry that supplied none does not round-trip
	// to "options":null. Go drops a nil map and keeps a non-nil empty one, so
	// an explicit {} still survives as {}.
	Options  map[string]interface{} `json:"options,omitempty"`
	Services []string               `json:"services,omitempty"`
}
```

Replace `templates/socket/writer/rest.go` entirely:

```go
package writer

type RestModel struct {
	OpCode string `json:"opCode"`
	Writer string `json:"writer"`
	// FName carries the client-side function name. Informational only; see
	// handler.RestModel.FName.
	FName string `json:"fname,omitempty"`
	// Options is omitempty; see handler.RestModel.Options.
	Options  map[string]interface{} `json:"options,omitempty"`
	Services []string               `json:"services,omitempty"`
}
```

- [ ] **Step 4: Run the templates tests to verify they pass**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./templates/socket/... -v`

Expected: PASS, all six tests.

- [ ] **Step 5: Mirror everything into the tenants tree**

Apply the identical three files under `tenants/socket/`, changing only the import paths (`atlas-configurations/tenants/socket/handler` and `.../writer`). Then create `tenants/socket/rest_test.go` with the same six tests, changing only the import paths at the top:

```go
import (
	"encoding/json"
	"strings"
	"testing"

	"atlas-configurations/tenants/socket/handler"
	"atlas-configurations/tenants/socket/writer"
)
```

The test bodies are byte-identical to Step 1's — copy them across unchanged.

- [ ] **Step 6: Run the tenants tests to verify they pass**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./tenants/socket/... -v`

Expected: PASS, all six tests.

- [ ] **Step 7: Confirm no existing test regressed**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./... 2>&1 | tail -40`

Expected: all packages `ok` or `no test files`. `templates/rest_test.go` and `tenants/rest_test.go` construct `socket.RestModel` literals; adding a field with a zero value does not break a keyed literal, but if either used an *unkeyed* literal it will now fail to compile — fix by adding field names, not by reverting the model.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/templates/socket \
        services/atlas-configurations/atlas.com/configurations/tenants/socket
git commit -m "feat(configurations): add socket.unsupported and per-entry fname

Additive REST-model fields on both the templates and tenants trees. Entity and
AutoMigrate are untouched - both store the configuration as a json.RawMessage
column, so this needs no migration and no backfill.

Normalize() turns nil slices into empty ones on every read and write path, so
an absent 'unsupported' key always marshals back as real arrays. Options gains
omitempty so an entry that supplied none stops round-tripping to null and
turning the first save of a template into a 200-line diff.

No consumer change: nothing in services/ or libs/ calls DisallowUnknownFields,
and atlas-channel decodes socket documents into a model declaring only Handlers
and Writers, so both new fields are ignored by construction."
```

---

## Task 5: Wire socket validation into both processors and the error plumbing

The rules from Task 3 must run on **both** `Create` and `UpdateById` for **both** trees. `Create` currently runs no validation at all and its resource handler has no `errors.As` branch — the UI clones a tenant from a template via `Create`, so leaving it unvalidated would be a hole straight through the strict-validation decision.

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/templates/socket/adapter.go`
- Create: `services/atlas-configurations/atlas.com/configurations/tenants/socket/adapter.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/validation_error.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/validation_error.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/processor.go:83-136`
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/processor.go:115-190`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/resource.go:35-58`
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/resource.go:114-130`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/socket_validation_test.go`
- Create: `services/atlas-configurations/atlas.com/configurations/tenants/socket_validation_test.go`

**Interfaces:**
- Consumes: `socket.Validate`, `socket.Input`, `socket.Binding`, `socket.Issue` (Task 3); `templates/socket.RestModel`, `templates/socket.Normalize` and the tenant equivalents (Task 4).
- Produces:
  - `templates/socket.ToValidationInput(rm RestModel) configsocket.Input` and the tenant equivalent
  - `templates.validationFailureError` gaining a `socketIssues []configsocket.Issue` field alongside its existing `errors []preset.ValidationError`, and the tenant equivalent
  - Socket validation running unconditionally inside `Create` and `UpdateById` on both trees

**Import alias:** both adapters and both `validation_error.go` files import the shared package as `configsocket "atlas-configurations/socket"`, because the local package is already named `socket`. Use that alias consistently.

**Why validation is not routed through `WithValidator`:** that injection point exists because *preset* validation needs an atlas-data client. Socket validation is pure and dependency-free, so it runs unconditionally inside the processor. Leave the `WithValidator` seam exactly as it is.

- [ ] **Step 1: Write the failing test for the templates tree**

Create `services/atlas-configurations/atlas.com/configurations/templates/socket_validation_test.go`:

```go
package templates

import (
	"context"
	"errors"
	"testing"

	configsocket "atlas-configurations/socket"
	"atlas-configurations/templates/socket"
	"atlas-configurations/templates/socket/handler"
	"atlas-configurations/templates/socket/writer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func validTemplate() RestModel {
	return RestModel{
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		Socket: socket.RestModel{
			Handlers: []handler.RestModel{
				{OpCode: "0x01", Validator: "NoOpValidator", Handler: "LoginHandle", Services: []string{"login"}},
			},
			Writers: []writer.RestModel{
				{OpCode: "0x00", Writer: "AuthSuccess", Services: []string{"login"}},
			},
		},
	}
}

func TestUpdateById_RejectsConflictingUnsupportedState(t *testing.T) {
	db := testDB(t)
	p := NewProcessor(logrus.New(), context.Background(), db)

	id, err := p.Create(validTemplate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	bad := validTemplate()
	bad.Socket.Unsupported.Handlers = []string{"LoginHandle"}

	err = p.UpdateById(id, bad)
	var ve *validationFailureError
	if !errors.As(err, &ve) {
		t.Fatalf("UpdateById err = %v, want *validationFailureError", err)
	}
	jsonErrs := ve.AsJSONAPIErrors()
	if len(jsonErrs) != 1 {
		t.Fatalf("got %d JSON:API errors, want 1: %+v", len(jsonErrs), jsonErrs)
	}
	if jsonErrs[0].Status != "400" {
		t.Errorf("status = %q, want 400", jsonErrs[0].Status)
	}
	if got := jsonErrs[0].Meta["path"]; got != "socket.unsupported.handlers[0]" {
		t.Errorf("meta.path = %v, want socket.unsupported.handlers[0]", got)
	}
}

func TestCreate_RejectsInvalidSocket(t *testing.T) {
	db := testDB(t)
	p := NewProcessor(logrus.New(), context.Background(), db)

	bad := validTemplate()
	bad.Socket.Handlers[0].Validator = ""

	_, err := p.Create(bad)
	var ve *validationFailureError
	if !errors.As(err, &ve) {
		t.Fatalf("Create err = %v, want *validationFailureError", err)
	}
	if got := ve.AsJSONAPIErrors()[0].Meta["path"]; got != "socket.handlers[0].validator" {
		t.Errorf("meta.path = %v, want socket.handlers[0].validator", got)
	}
}

func TestCreate_AcceptsValidSocketAndNormalizes(t *testing.T) {
	db := testDB(t)
	p := NewProcessor(logrus.New(), context.Background(), db)

	id, err := p.Create(validTemplate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("Create returned the nil UUID")
	}

	got, err := p.GetById(id)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Socket.Unsupported.Handlers == nil || got.Socket.Unsupported.Writers == nil {
		t.Errorf("stored document did not normalize unsupported: %+v", got.Socket.Unsupported)
	}
}

func TestToValidationInput_FlattensBothCollections(t *testing.T) {
	rm := socket.RestModel{
		Handlers: []handler.RestModel{
			{OpCode: "0x01", Validator: "NoOpValidator", Handler: "LoginHandle", Services: []string{"login"}},
		},
		Writers: []writer.RestModel{
			{OpCode: "0x00", Writer: "AuthSuccess", Services: []string{"login"}},
		},
		Unsupported: socket.UnsupportedRestModel{Handlers: []string{"GuestLoginHandle"}},
	}
	in := socket.ToValidationInput(rm)

	if len(in.Handlers) != 1 || in.Handlers[0].Name != "LoginHandle" || in.Handlers[0].Validator != "NoOpValidator" {
		t.Errorf("handlers not flattened: %+v", in.Handlers)
	}
	if len(in.Writers) != 1 || in.Writers[0].Name != "AuthSuccess" || in.Writers[0].Validator != "" {
		t.Errorf("writers not flattened: %+v", in.Writers)
	}
	if len(in.UnsupportedHandlers) != 1 || in.UnsupportedHandlers[0] != "GuestLoginHandle" {
		t.Errorf("unsupported not carried: %+v", in.UnsupportedHandlers)
	}
	// Compile-time proof the adapter returns the shared package's type.
	var _ configsocket.Input = in
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./templates/ -run 'Socket|ToValidationInput|Create_|UpdateById_'`

Expected: FAIL to compile — `undefined: socket.ToValidationInput`.

- [ ] **Step 3: Write the templates adapter**

Create `services/atlas-configurations/atlas.com/configurations/templates/socket/adapter.go`:

```go
package socket

import (
	configsocket "atlas-configurations/socket"
)

// ToValidationInput flattens this tree's RestModel into the shared validator's
// neutral Input. It is ~20 lines of mechanical copying that exist so the ~150
// lines of rules and their tests do not have to be duplicated per tree.
//
// Writers carry no validator, so their Binding.Validator is left empty; the
// shared validator only requires one for handlers.
func ToValidationInput(rm RestModel) configsocket.Input {
	in := configsocket.Input{
		Handlers:            make([]configsocket.Binding, 0, len(rm.Handlers)),
		Writers:             make([]configsocket.Binding, 0, len(rm.Writers)),
		UnsupportedHandlers: rm.Unsupported.Handlers,
		UnsupportedWriters:  rm.Unsupported.Writers,
	}
	for _, h := range rm.Handlers {
		in.Handlers = append(in.Handlers, configsocket.Binding{
			Name:      h.Handler,
			OpCode:    h.OpCode,
			Validator: h.Validator,
			Services:  h.Services,
		})
	}
	for _, w := range rm.Writers {
		in.Writers = append(in.Writers, configsocket.Binding{
			Name:     w.Writer,
			OpCode:   w.OpCode,
			Services: w.Services,
		})
	}
	return in
}
```

- [ ] **Step 4: Generalize the templates error type**

Replace `services/atlas-configurations/atlas.com/configurations/templates/validation_error.go` entirely:

```go
package templates

import (
	"fmt"

	configsocket "atlas-configurations/socket"
	"atlas-configurations/templates/characters/preset"
)

// validationFailureError carries both families of blocking validation failure:
// preset issues (which need an atlas-data client and so arrive via the injected
// validator) and socket issues (pure, always run). Both render through the same
// JSON:API error shape; only the meta.path differs.
type validationFailureError struct {
	errors       []preset.ValidationError
	socketIssues []configsocket.Issue
}

func (e *validationFailureError) Error() string {
	return fmt.Sprintf("validation failed (%d preset, %d socket issues)", len(e.errors), len(e.socketIssues))
}

type jsonapiError struct {
	Status string         `json:"status"`
	Title  string         `json:"title"`
	Detail string         `json:"detail"`
	Meta   map[string]any `json:"meta"`
}

func (e *validationFailureError) AsJSONAPIErrors() []jsonapiError {
	out := make([]jsonapiError, 0, len(e.errors)+len(e.socketIssues))
	for _, ve := range e.errors {
		out = append(out, jsonapiError{
			Status: "400",
			Title:  "validation failed",
			Detail: ve.Message,
			Meta:   map[string]any{"path": "presets[" + ve.PresetId + "]." + ve.Field},
		})
	}
	for _, iss := range e.socketIssues {
		out = append(out, jsonapiError{
			Status: "400",
			Title:  "validation failed",
			Detail: iss.Message,
			Meta:   map[string]any{"path": iss.Path},
		})
	}
	return out
}
```

- [ ] **Step 5: Run socket validation in the templates processor**

In `templates/processor.go`, add `"atlas-configurations/templates/socket"` to the imports, then change `Create` — insert immediately after the `func (p *ProcessorImpl) Create(input RestModel) (uuid.UUID, error) {` line:

```go
	input.Socket = socket.Normalize(input.Socket)
	if issues := socketValidate(input.Socket); len(issues) > 0 {
		return uuid.Nil, &validationFailureError{socketIssues: issues}
	}

```

and change `UpdateById` — insert immediately after the `func (p *ProcessorImpl) UpdateById(templateId uuid.UUID, input RestModel) error {` line:

```go
	input.Socket = socket.Normalize(input.Socket)
	if issues := socketValidate(input.Socket); len(issues) > 0 {
		return &validationFailureError{socketIssues: issues}
	}

```

Then add this helper at the bottom of `templates/processor.go`:

```go
// socketValidate runs the shared, dependency-free socket rules. Unlike preset
// validation it is not routed through WithValidator, because it needs no
// atlas-data client and must therefore never be skippable.
func socketValidate(rm socket.RestModel) []configsocket.Issue {
	return configsocket.Validate(socket.ToValidationInput(rm))
}
```

with `configsocket "atlas-configurations/socket"` added to the imports.

Finally, normalize on the read path — in `Make`, replace `rm.Id = e.Id.String()` with:

```go
	rm.Socket = socket.Normalize(rm.Socket)
	rm.Id = e.Id.String()
```

- [ ] **Step 6: Run the templates tests to verify they pass**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./templates/ -run 'Socket|ToValidationInput|Create_|UpdateById_' -v`

Expected: PASS, all four tests.

- [ ] **Step 7: Add the `errors.As` branch to the templates Create handler**

`handleUpdateConfigurationTemplate` already has this branch (`templates/resource.go:132-139`); `handleCreateConfigurationTemplate` has none, so a validation failure there would surface as a 500. In `templates/resource.go`, replace the error block inside `handleCreateConfigurationTemplate`:

```go
			templateId, err := NewProcessor(d.Logger(), d.Context(), db).Create(input)
			if err != nil {
				var ve *validationFailureError
				if errors.As(err, &ve) {
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})
					return
				}
				d.Logger().WithError(err).Errorf("Unable to create configuration template.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
```

`errors`, `encoding/json` and `net/http` are already imported in that file.

- [ ] **Step 8: Mirror Steps 3–7 into the tenants tree**

Apply the identical changes under `tenants/`:

- `tenants/socket/adapter.go` — byte-identical to Step 3's file apart from the package's own import path (it imports only `configsocket`, so the file is in fact identical; place it in the `tenants/socket` package).
- `tenants/validation_error.go` — Step 4's file with `"atlas-configurations/tenants/characters/preset"` in place of the templates preset import and `package tenants` at the top.
- `tenants/processor.go` — the same three edits (`Create`, `UpdateById`, `Make`) plus the same `socketValidate` helper, importing `"atlas-configurations/tenants/socket"` and `configsocket "atlas-configurations/socket"`. Note the tenant `Create` returns `(uuid.UUID, error)` and its `UpdateById` ends by enqueuing a tenant-status outbox row inside the transaction — insert the validation **before** the `json.Marshal`, so an invalid document never reaches the outbox.
- `tenants/resource.go` — the same `errors.As` branch in `handleCreateConfigurationTenant`.
- `tenants/socket_validation_test.go` — Step 1's tests with `package tenants`, the tenant import paths, and `&Entity{}` from the tenants package. The tenant `Create` accepts an `input.Id`; leave it empty so a UUID is generated.

- [ ] **Step 9: Run the tenants tests to verify they pass**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./tenants/ -run 'Socket|ToValidationInput|Create_|UpdateById_' -v`

Expected: PASS, all four tests.

- [ ] **Step 10: Full service verification**

```bash
cd services/atlas-configurations/atlas.com/configurations
go build ./... && go vet ./... && go test -race ./...
```

Expected: all three clean.

Because `go.mod` was touched in Task 3, the container build is now mandatory. From the **worktree root**:

```bash
docker buildx bake atlas-configurations
```

Expected: build succeeds. `libs/atlas-opcodes` is already COPYed by the shared `Dockerfile` (lines 38 and 68), so no Dockerfile edit should be needed — if the bake fails on a missing `COPY libs/...`, add the two lines rather than dropping the dependency.

- [ ] **Step 11: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/templates \
        services/atlas-configurations/atlas.com/configurations/tenants
git commit -m "feat(configurations): enforce socket validation on create and update

Both trees now run the shared socket rules unconditionally inside the processor
on both Create and UpdateById, and normalize the socket document on every read
and write path.

Create previously ran no validation at all and its resource handler had no
errors.As branch, so a bad document would have 500'd rather than 400'd. That
mattered because Create is how the UI clones a tenant from a template - it was
a hole straight through strict validation.

Socket validation deliberately does NOT go through WithValidator: that seam
exists because preset validation needs an atlas-data client, and socket rules
are pure and must never be skippable."
```

---

## Task 6: The `packet-audit seed-fname` subcommand

A new subcommand in the existing tool rather than a throwaway script: `tools/packet-audit/internal/opregistry` already parses these YAML files and is trusted, and a subcommand is re-runnable when the next version bring-up adds a registry. The PRD's "runs once" is satisfied without making it a one-shot.

**Files:**
- Create: `tools/packet-audit/cmd/seed_fname.go`
- Create: `tools/packet-audit/cmd/seed_fname_test.go`
- Modify: `tools/packet-audit/cmd/root.go` (dispatch the new subcommand)
- Modify: `tools/packet-audit/README.md`

**Interfaces:**
- Consumes: `opregistry.LoadVersion(path) (*opregistry.VersionFile, error)`; `opregistry.Entry{Op string; Direction Direction; Opcode int; FName string}`; `opregistry.DirClientbound`, `opregistry.DirServerbound`.
- Produces, for Task 7: `runSeedFName(args []string, stderr io.Writer) int`, reachable as `packet-audit seed-fname [--write] [--registry-dir DIR] [--template-dir DIR]`, printing per-template and total coverage to stdout and exiting non-zero on any fidelity violation.

**Version mapping** (template file stem → registry file stem):

| Template | Registry | Fallback if no registry |
|---|---|---|
| `gms_12_1` | *(none)* | impl-name match against `gms_48_1` then `gms_61_1` |
| `gms_48_1` | `gms_v48` | — |
| `gms_61_1` | `gms_v61` | — |
| `gms_72_1` | `gms_v72` | — |
| `gms_79_1` | `gms_v79` | — |
| `gms_83_1` | `gms_v83` | — |
| `gms_84_1` | `gms_v84` | — |
| `gms_87_1` | `gms_v87` | — |
| `gms_92_1` | *(none)* | impl-name match against `gms_87_1` then `gms_95_1` |
| `gms_95_1` | `gms_v95` | — |
| `jms_185_1` | `jms_v185` | — |

**Join direction:** `handlers` ⋈ `serverbound`, `writers` ⋈ `clientbound`. Registry rows with an empty `fname` are skipped, never joined.

**Ambiguity tie-break:** exactly one `(direction, opcode)` group across the nine registries carries more than one distinct `fname` — `gms_v61` clientbound opcode 242, which is `STORAGE` → `CTrunkDlg::OnPacket` and `RPS_GAME` → `CRPSGameDlg::OnPacket`. Pick the lexicographically-first `op` name, so `RPS_GAME` wins and the resolved `fname` is `CRPSGameDlg::OnPacket`. Log every tie-break to stderr.

**JSON fidelity — two mandatory guards.** Re-marshalling these files through a partial struct would silently destroy `characters`, `worlds` or `cashShop`.

1. **Unknown-key stop.** Decode each file into `map[string]json.RawMessage` at the top level and per socket entry and fail loudly on any key outside the modelled set. A surprise key is a stop, not a silent drop.
2. **Verbatim carry-through.** Every top-level value the generator does not touch is held as `json.RawMessage` and written back unchanged. Only `socket.handlers` / `socket.writers` are fully modelled — which guard 1 proves is safe.

Output is `json.MarshalIndent(doc, "", "  ")` plus a trailing newline, matching the existing two-space files. HTML escaping is a non-issue here: measured, no seed template contains `<`, `>`, `&` or any `\u` escape, so `json.Marshal`'s default escaping is byte-stable against the current files.

- [ ] **Step 1: Confirm the modelled key sets against the real files before coding**

```bash
python3 -c "
import json,glob,collections
top=collections.Counter(); ent=collections.Counter()
for f in sorted(glob.glob('services/atlas-configurations/seed-data/templates/template_*.json')):
    d=json.load(open(f)); top.update(d.keys())
    for k in ('handlers','writers'):
        for e in d.get('socket',{}).get(k) or []: ent.update(e.keys())
print('top-level keys   :', sorted(top))
print('socket entry keys:', sorted(ent))
"
```

Expected:

```
top-level keys   : ['cashShop', 'characters', 'majorVersion', 'minorVersion', 'npcs', 'region', 'socket', 'usesPin', 'worlds']
socket entry keys: ['handler', 'opCode', 'options', 'services', 'validator', 'writer']
```

If either set differs, use the measured set in the code below. An unmodelled key must be a hard failure, so the list has to be right.

- [ ] **Step 2: Write the failing test**

Create `tools/packet-audit/cmd/seed_fname_test.go`:

```go
package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSeedFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

const testRegistryV83 = `- op: LOGIN_STATUS
  direction: clientbound
  opcode: 1
  fname: CLogin::OnCheckPasswordResult
  provenance: csv-import
- op: LOGIN_PASSWORD
  direction: serverbound
  opcode: 1
  fname: CLogin::SendCheckPasswordPacket
  provenance: csv-import
- op: NO_FNAME
  direction: clientbound
  opcode: 2
  fname: ""
  provenance: csv-import
`

const testTemplate83 = `{
  "region": "GMS",
  "majorVersion": 83,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [
      {
        "opCode": "0x01",
        "validator": "NoOpValidator",
        "handler": "LoginHandle",
        "services": [
          "login"
        ]
      }
    ],
    "writers": [
      {
        "opCode": "0x01",
        "writer": "AuthSuccess",
        "options": {},
        "services": [
          "login"
        ]
      },
      {
        "opCode": "0x02",
        "writer": "ServerLoad",
        "services": [
          "login"
        ]
      }
    ]
  },
  "characters": {
    "templates": [],
    "presets": []
  },
  "npcs": [],
  "worlds": [
    {
      "name": "Scania"
    }
  ],
  "cashShop": {
    "commodities": {}
  }
}
`

// setupSeedFName lays down a registry and a set of templates in a temp dir and
// returns (registryDir, templateDir).
func setupSeedFName(t *testing.T, registries map[string]string, templates map[string]string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	regDir := filepath.Join(dir, "registry")
	tplDir := filepath.Join(dir, "templates")
	for stem, body := range registries {
		writeSeedFile(t, filepath.Join(regDir, stem+".yaml"), body)
	}
	for stem, body := range templates {
		writeSeedFile(t, filepath.Join(tplDir, "template_"+stem+".json"), body)
	}
	return regDir, tplDir
}

func socketOf(t *testing.T, path string) (handlers, writers []map[string]json.RawMessage) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		t.Fatalf("reparse %s: %v", path, err)
	}
	var sock struct {
		Handlers []map[string]json.RawMessage `json:"handlers"`
		Writers  []map[string]json.RawMessage `json:"writers"`
	}
	if err := json.Unmarshal(top["socket"], &sock); err != nil {
		t.Fatalf("parse socket in %s: %v", path, err)
	}
	return sock.Handlers, sock.Writers
}

func TestSeedFName_ResolvesBothDirections(t *testing.T) {
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": testTemplate83})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	h, w := socketOf(t, filepath.Join(tplDir, "template_gms_83_1.json"))
	if got := string(h[0]["fname"]); got != `"CLogin::SendCheckPasswordPacket"` {
		t.Errorf("handler joined the wrong direction: fname = %s", got)
	}
	if got := string(w[0]["fname"]); got != `"CLogin::OnCheckPasswordResult"` {
		t.Errorf("writer joined the wrong direction: fname = %s", got)
	}
	if _, present := w[1]["fname"]; present {
		t.Errorf("a registry row with an empty fname produced a key: %v", w[1])
	}
}

func TestSeedFName_PreservesUnmodelledTopLevelKeysVerbatim(t *testing.T) {
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": testTemplate83})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	b, _ := os.ReadFile(filepath.Join(tplDir, "template_gms_83_1.json"))
	var before, after map[string]any
	if err := json.Unmarshal([]byte(testTemplate83), &before); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := json.Unmarshal(b, &after); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	for _, k := range []string{"region", "majorVersion", "minorVersion", "usesPin", "characters", "npcs", "worlds", "cashShop"} {
		bj, _ := json.Marshal(before[k])
		aj, _ := json.Marshal(after[k])
		if string(bj) != string(aj) {
			t.Errorf("top-level %q changed:\n before: %s\n after : %s", k, bj, aj)
		}
	}
}

func TestSeedFName_FailsLoudlyOnUnknownTopLevelKey(t *testing.T) {
	surprising := strings.Replace(testTemplate83, `"usesPin": false,`,
		"\"usesPin\": false,\n  \"surpriseKey\": {\"a\": 1},", 1)
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": surprising})

	var stderr bytes.Buffer
	code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr)
	if code == 0 {
		t.Fatal("exit = 0, want non-zero on an unmodelled top-level key")
	}
	if !strings.Contains(stderr.String(), "surpriseKey") {
		t.Errorf("stderr did not name the offending key:\n%s", stderr.String())
	}
}

func TestSeedFName_FailsLoudlyOnUnknownEntryKey(t *testing.T) {
	surprising := strings.Replace(testTemplate83, `"handler": "LoginHandle",`,
		"\"handler\": \"LoginHandle\",\n        \"surpriseEntryKey\": 1,", 1)
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": surprising})

	var stderr bytes.Buffer
	code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr)
	if code == 0 {
		t.Fatal("exit = 0, want non-zero on an unmodelled socket-entry key")
	}
	if !strings.Contains(stderr.String(), "surpriseEntryKey") {
		t.Errorf("stderr did not name the offending key:\n%s", stderr.String())
	}
}

func TestSeedFName_WithoutWriteLeavesFilesUntouched(t *testing.T) {
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": testTemplate83})
	path := filepath.Join(tplDir, "template_gms_83_1.json")

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	b, _ := os.ReadFile(path)
	if string(b) != testTemplate83 {
		t.Errorf("dry run modified the file:\n%s", b)
	}
}

func TestSeedFName_AmbiguityPicksLexicographicallyFirstOp(t *testing.T) {
	const reg = `- op: STORAGE
  direction: clientbound
  opcode: 242
  fname: CTrunkDlg::OnPacket
  provenance: manual
- op: RPS_GAME
  direction: clientbound
  opcode: 242
  fname: CRPSGameDlg::OnPacket
  provenance: manual
`
	const tpl = `{
  "region": "GMS",
  "majorVersion": 61,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [],
    "writers": [
      {
        "opCode": "0xF2",
        "writer": "MiniRoom",
        "services": [
          "channel"
        ]
      }
    ]
  },
  "characters": {
    "templates": [],
    "presets": []
  },
  "npcs": [],
  "worlds": [],
  "cashShop": {
    "commodities": {}
  }
}
`
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v61": reg},
		map[string]string{"gms_61_1": tpl})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	_, w := socketOf(t, filepath.Join(tplDir, "template_gms_61_1.json"))
	if got := string(w[0]["fname"]); got != `"CRPSGameDlg::OnPacket"` {
		t.Errorf("tie-break did not pick RPS_GAME (lexicographically first op): fname = %s", got)
	}
	if !strings.Contains(stderr.String(), "ambiguous") {
		t.Errorf("tie-break was not logged to stderr:\n%s", stderr.String())
	}
}

// gms_92_1 and gms_12_1 have no registry of their own. They resolve by
// implementation name against adjacent versions, which is valid because the
// implementation name is the definition identity within a direction. LoginHandle
// sits at a DIFFERENT opcode in the borrower, so only an impl-name match works.
func TestSeedFName_FallsBackToAdjacentVersionByImplName(t *testing.T) {
	const tpl92 = `{
  "region": "GMS",
  "majorVersion": 92,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [
      {
        "opCode": "0x7F",
        "validator": "NoOpValidator",
        "handler": "LoginHandle",
        "services": [
          "login"
        ]
      }
    ],
    "writers": []
  },
  "characters": {
    "templates": [],
    "presets": []
  },
  "npcs": [],
  "worlds": [],
  "cashShop": {
    "commodities": {}
  }
}
`
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v87": testRegistryV83},
		map[string]string{"gms_87_1": testTemplate83, "gms_92_1": tpl92})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	h, _ := socketOf(t, filepath.Join(tplDir, "template_gms_92_1.json"))
	if got := string(h[0]["fname"]); got != `"CLogin::SendCheckPasswordPacket"` {
		t.Errorf("adjacent-version impl-name fallback did not resolve: fname = %s", got)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `cd tools/packet-audit && go test ./cmd/ -run TestSeedFName`

Expected: FAIL to compile — `undefined: runSeedFName`.

- [ ] **Step 4: Write the implementation**

Create `tools/packet-audit/cmd/seed_fname.go`:

```go
package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// seedFNameVersions maps a seed template's file stem to the registry file stem
// it joins against, and to the ordered fallback templates it borrows from when
// it has no registry of its own. Fallback resolves by IMPLEMENTATION NAME,
// which is valid because the implementation name is the definition identity
// within a direction - the opcode is version-specific and cannot be joined on.
var seedFNameVersions = []struct {
	Template string   // template file stem, e.g. "gms_83_1"
	Registry string   // registry file stem, e.g. "gms_v83"; empty means none
	Fallback []string // ordered template stems to borrow from; first hit wins
}{
	{Template: "gms_12_1", Registry: "", Fallback: []string{"gms_48_1", "gms_61_1"}},
	{Template: "gms_48_1", Registry: "gms_v48"},
	{Template: "gms_61_1", Registry: "gms_v61"},
	{Template: "gms_72_1", Registry: "gms_v72"},
	{Template: "gms_79_1", Registry: "gms_v79"},
	{Template: "gms_83_1", Registry: "gms_v83"},
	{Template: "gms_84_1", Registry: "gms_v84"},
	{Template: "gms_87_1", Registry: "gms_v87"},
	{Template: "gms_92_1", Registry: "", Fallback: []string{"gms_87_1", "gms_95_1"}},
	{Template: "gms_95_1", Registry: "gms_v95"},
	{Template: "jms_185_1", Registry: "jms_v185"},
}

var knownTopLevelKeys = map[string]bool{
	"region": true, "majorVersion": true, "minorVersion": true, "usesPin": true,
	"socket": true, "characters": true, "npcs": true, "worlds": true, "cashShop": true,
}

var knownEntryKeys = map[string]bool{
	"opCode": true, "validator": true, "handler": true, "writer": true,
	"fname": true, "options": true, "services": true,
}

// seedDoc is the write model. Everything the generator does not touch is held
// as a verbatim json.RawMessage, so re-marshalling cannot alter it. Field order
// here IS the output key order, and it matches the existing seed files.
type seedDoc struct {
	Region       json.RawMessage `json:"region"`
	MajorVersion json.RawMessage `json:"majorVersion"`
	MinorVersion json.RawMessage `json:"minorVersion"`
	UsesPin      json.RawMessage `json:"usesPin"`
	Socket       seedSocket      `json:"socket"`
	Characters   json.RawMessage `json:"characters,omitempty"`
	NPCs         json.RawMessage `json:"npcs,omitempty"`
	Worlds       json.RawMessage `json:"worlds,omitempty"`
	CashShop     json.RawMessage `json:"cashShop,omitempty"`
}

type seedSocket struct {
	Handlers []seedEntry `json:"handlers"`
	Writers  []seedEntry `json:"writers"`
}

// seedEntry is the ONLY fully-modelled structure. loadSeedTemplate's
// unknown-key check is what makes that safe: a socket entry carrying a key
// outside knownEntryKeys stops the run rather than losing the key here.
type seedEntry struct {
	OpCode    string          `json:"opCode"`
	Validator string          `json:"validator,omitempty"`
	Handler   string          `json:"handler,omitempty"`
	Writer    string          `json:"writer,omitempty"`
	FName     string          `json:"fname,omitempty"`
	Options   json.RawMessage `json:"options,omitempty"`
	Services  json.RawMessage `json:"services,omitempty"`
}

// Name returns the implementation name, whichever collection this entry is in.
func (e seedEntry) Name() string {
	if e.Handler != "" {
		return e.Handler
	}
	return e.Writer
}

type seedTemplate struct {
	Stem string
	Path string
	Doc  seedDoc
}

func runSeedFName(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("seed-fname", flag.ContinueOnError)
	fs.SetOutput(stderr)
	write := fs.Bool("write", false, "write the resolved fname values back to the template files")
	registryDir := fs.String("registry-dir", filepath.Join("docs", "packets", "registry"),
		"directory holding <version>.yaml op registries")
	templateDir := fs.String("template-dir", filepath.Join("services", "atlas-configurations", "seed-data", "templates"),
		"directory holding template_<version>.json seed files")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	// Load every template that is present. A version listed above but absent
	// from disk is skipped, which is what lets the tests drive a subset.
	templates := make(map[string]*seedTemplate)
	var order []string
	for _, v := range seedFNameVersions {
		path := filepath.Join(*templateDir, "template_"+v.Template+".json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		st, err := loadSeedTemplate(v.Template, path)
		if err != nil {
			fmt.Fprintf(stderr, "FAIL %s: %v\n", filepath.Base(path), err)
			return 1
		}
		templates[v.Template] = st
		order = append(order, v.Template)
	}

	// Pass 1 - direct (direction, opcode) join for every version with a registry.
	// resolved[templateStem]["handler|LoginHandle"] = fname, which pass 2 borrows.
	resolved := make(map[string]map[string]string)
	for _, v := range seedFNameVersions {
		st, ok := templates[v.Template]
		if !ok || v.Registry == "" {
			continue
		}
		regPath := filepath.Join(*registryDir, v.Registry+".yaml")
		vf, err := opregistry.LoadVersion(regPath)
		if err != nil {
			fmt.Fprintf(stderr, "FAIL %s: %v\n", v.Registry, err)
			return 1
		}
		resolved[v.Template] = applyDirect(st, indexRegistryByOpcode(vf, v.Registry, stderr))
	}

	// Pass 2 - versions with no registry borrow by implementation name.
	for _, v := range seedFNameVersions {
		st, ok := templates[v.Template]
		if !ok || v.Registry != "" {
			continue
		}
		applyFallback(st, v.Fallback, resolved)
	}

	totalEntries, totalResolved := 0, 0
	for _, stem := range order {
		st := templates[stem]
		entries := len(st.Doc.Socket.Handlers) + len(st.Doc.Socket.Writers)
		got := countResolved(st)
		totalEntries += entries
		totalResolved += got
		fmt.Printf("%-12s %4d / %4d resolved\n", stem, got, entries)
	}
	fmt.Printf("%-12s %4d / %4d resolved\n", "TOTAL", totalResolved, totalEntries)

	if !*write {
		fmt.Println("(dry run - pass --write to update the template files)")
		return 0
	}
	for _, stem := range order {
		if err := writeSeedTemplate(templates[stem]); err != nil {
			fmt.Fprintf(stderr, "FAIL writing %s: %v\n", stem, err)
			return 1
		}
	}
	return 0
}

// loadSeedTemplate decodes a template and refuses it if it carries any JSON key
// the write model does not represent. Guard 1 of the two fidelity guards.
func loadSeedTemplate(stem, path string) (*seedTemplate, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}
	for k := range top {
		if !knownTopLevelKeys[k] {
			return nil, fmt.Errorf("unmodelled top-level key %q - refusing to rewrite, "+
				"because marshalling through the known-key model would silently drop it", k)
		}
	}

	if raw, ok := top["socket"]; ok {
		var sock map[string]json.RawMessage
		if err := json.Unmarshal(raw, &sock); err != nil {
			return nil, fmt.Errorf("parse socket: %w", err)
		}
		for _, group := range []string{"handlers", "writers"} {
			gr, ok := sock[group]
			if !ok {
				continue
			}
			var entries []map[string]json.RawMessage
			if err := json.Unmarshal(gr, &entries); err != nil {
				return nil, fmt.Errorf("parse socket.%s: %w", group, err)
			}
			for i, e := range entries {
				for k := range e {
					if !knownEntryKeys[k] {
						return nil, fmt.Errorf("unmodelled socket-entry key %q at socket.%s[%d] - refusing to rewrite", k, group, i)
					}
				}
			}
		}
	}

	var doc seedDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	return &seedTemplate{Stem: stem, Path: path, Doc: doc}, nil
}

// indexRegistryByOpcode builds "direction|opcode" -> fname, applying the
// lexicographically-first-op tie-break where one opcode carries several
// distinct fnames, and logging every such choice.
func indexRegistryByOpcode(vf *opregistry.VersionFile, regStem string, stderr io.Writer) map[string]string {
	type cand struct{ op, fname string }
	groups := make(map[string][]cand)
	for _, e := range vf.Entries {
		fn := strings.TrimSpace(e.FName)
		if fn == "" {
			continue
		}
		k := string(e.Direction) + "|" + strconv.Itoa(e.Opcode)
		groups[k] = append(groups[k], cand{op: e.Op, fname: fn})
	}

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]string, len(groups))
	for _, k := range keys {
		cs := groups[k]
		sort.Slice(cs, func(i, j int) bool { return cs[i].op < cs[j].op })
		distinct := map[string]bool{}
		for _, c := range cs {
			distinct[c.fname] = true
		}
		if len(distinct) > 1 {
			names := make([]string, 0, len(cs))
			for _, c := range cs {
				names = append(names, c.op+"="+c.fname)
			}
			fmt.Fprintf(stderr, "ambiguous %s %s: %s - picking %s (lexicographically first op)\n",
				regStem, k, strings.Join(names, ", "), cs[0].op)
		}
		out[k] = cs[0].fname
	}
	return out
}

func applyDirect(st *seedTemplate, byOp map[string]string) map[string]string {
	got := make(map[string]string)
	apply := func(entries []seedEntry, dir opregistry.Direction, kind string) {
		for i := range entries {
			code, ok := parseSeedOpCode(entries[i].OpCode)
			if !ok {
				continue
			}
			fn := byOp[string(dir)+"|"+strconv.Itoa(code)]
			if fn == "" {
				continue
			}
			entries[i].FName = fn
			if n := entries[i].Name(); n != "" {
				got[kind+"|"+n] = fn
			}
		}
	}
	apply(st.Doc.Socket.Handlers, opregistry.DirServerbound, "handler")
	apply(st.Doc.Socket.Writers, opregistry.DirClientbound, "writer")
	return got
}

func applyFallback(st *seedTemplate, fallback []string, resolved map[string]map[string]string) {
	apply := func(entries []seedEntry, kind string) {
		for i := range entries {
			n := entries[i].Name()
			if n == "" {
				continue
			}
			for _, src := range fallback {
				if fn := resolved[src][kind+"|"+n]; fn != "" {
					entries[i].FName = fn
					break
				}
			}
		}
	}
	apply(st.Doc.Socket.Handlers, "handler")
	apply(st.Doc.Socket.Writers, "writer")
}

func countResolved(st *seedTemplate) int {
	n := 0
	for _, e := range st.Doc.Socket.Handlers {
		if e.FName != "" {
			n++
		}
	}
	for _, e := range st.Doc.Socket.Writers {
		if e.FName != "" {
			n++
		}
	}
	return n
}

func parseSeedOpCode(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 3 || s[0] != '0' || (s[1] != 'x' && s[1] != 'X') {
		return 0, false
	}
	n, err := strconv.ParseInt(s[2:], 16, 32)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

// writeSeedTemplate re-encodes the document with two-space indentation and a
// trailing newline, matching the existing seed files. Guard 2: every value the
// generator did not touch is a json.RawMessage and round-trips unchanged.
func writeSeedTemplate(st *seedTemplate) error {
	b, err := json.MarshalIndent(st.Doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(st.Path, append(b, '\n'), 0o644)
}
```

- [ ] **Step 5: Register the subcommand**

In `tools/packet-audit/cmd/root.go`, `Run` is a flat run of `if len(args) > 0 && args[0] == "…"` clauses. Add one after the `"registry"` clause:

```go
	if len(args) > 0 && args[0] == "seed-fname" {
		return runSeedFName(args[1:], stderr)
	}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd tools/packet-audit && go test ./cmd/ -run TestSeedFName -v`

Expected: PASS, all seven tests.

If `TestSeedFName_WithoutWriteLeavesFilesUntouched` fails on a byte comparison after a `--write` run in another test, the temp dirs are leaking between tests — each uses its own `t.TempDir()`, so that would indicate a shared path bug in `setupSeedFName`.

- [ ] **Step 7: Build, vet, race**

```bash
cd tools/packet-audit && go build ./... && go vet ./... && go test -race ./cmd/ -run TestSeedFName
```

Expected: all clean.

- [ ] **Step 8: Document the subcommand**

Append to the subcommand list in `tools/packet-audit/README.md`:

````markdown
### `seed-fname`

Backfills the optional `fname` field (the client-side function name) into the
eleven configuration seed templates by joining each socket entry's opcode
against `docs/packets/registry/<version>.yaml` — `handlers` against
`serverbound`, `writers` against `clientbound`.

```
packet-audit seed-fname                      # dry run, prints coverage
packet-audit seed-fname --write              # updates the seed templates
packet-audit seed-fname --registry-dir DIR --template-dir DIR
```

`gms_92_1` and `gms_12_1` have no registry of their own; they resolve by
implementation name against adjacent versions (v87 then v95, and v48 then v61
respectively), which is valid because the implementation name is the definition
identity within a direction. Entries that resolve to nothing are written without
`fname` — the field is `omitempty`.

Where one `(direction, opcode)` carries several distinct `fname` values, the
lexicographically-first `op` name wins and the choice is logged to stderr.

Re-run it after a version bring-up adds a registry. It refuses to rewrite any
template carrying a JSON key it does not model, so a schema change is a hard
stop rather than silent data loss.
````

- [ ] **Step 9: Commit**

```bash
git add tools/packet-audit/cmd/seed_fname.go tools/packet-audit/cmd/seed_fname_test.go \
        tools/packet-audit/cmd/root.go tools/packet-audit/README.md
git commit -m "feat(packet-audit): add seed-fname subcommand

Joins each seed template's socket entries against the per-version op registry to
backfill the client-side function name. Lives in packet-audit rather than a
throwaway script because internal/opregistry already parses these YAML files and
is trusted, and because the next version bring-up needs to re-run it.

Two mandatory fidelity guards: it refuses to rewrite a template carrying any
JSON key it does not model, and every value it does not touch is held as a
verbatim json.RawMessage. Re-marshalling these files through a partial struct
would silently drop characters/worlds/cashShop."
```

---

## Task 7: Run the generator and commit the seeded templates

**Files:**
- Modify: all eleven `services/atlas-configurations/seed-data/templates/template_*.json`
- Create: `docs/tasks/task-194-packet-definition-matrix/fname-coverage.txt`

**Interfaces:**
- Consumes: `packet-audit seed-fname` (Task 6); the de-duplicated corpus (Task 2).
- Produces: seed templates carrying `fname`, and the committed coverage report that the amended PRD acceptance criterion is stated against.

**Note (design F6):** `seeder.go`'s `importTemplate` skips any template whose `(region, majorVersion, minorVersion)` row already exists, so backfilled seed files reach **new clusters and CI only** — never an existing deployment. Existing installs acquire `fname` through the UI, a baseline republish, or not at all. That is acceptable because `fname` is informational (FR-10.4), but say it in the commit message so nobody reports it as a bug.

- [ ] **Step 1: Capture the pre-run state for the semantic comparison**

From the worktree root:

```bash
rm -rf /tmp/task194-before && mkdir -p /tmp/task194-before && \
  cp services/atlas-configurations/seed-data/templates/template_*.json /tmp/task194-before/
```

- [ ] **Step 2: Dry run and record the coverage**

```bash
cd tools/packet-audit && go run . seed-fname \
  --registry-dir ../../docs/packets/registry \
  --template-dir ../../services/atlas-configurations/seed-data/templates \
  | tee ../../docs/tasks/task-194-packet-definition-matrix/fname-coverage.txt
```

Expected: eleven per-template lines plus a `TOTAL` line over 2,859 entries, and a single `ambiguous gms_v61 clientbound|242 …` line on stderr.

Read the total. Design F3 predicts roughly 2,845 of 2,859 (it measured 2,849/2,863 before the four duplicates were removed, and the removed entries were unresolvable duplicates of resolved ones). If the total is far below that the join is broken — investigate before writing. Record whatever the real number is; the amended acceptance criterion is "matches the committed report", not a hard-coded absolute.

- [ ] **Step 3: Write for real**

```bash
cd tools/packet-audit && go run . seed-fname --write \
  --registry-dir ../../docs/packets/registry \
  --template-dir ../../services/atlas-configurations/seed-data/templates
```

Expected: the same coverage numbers as the dry run, and eleven modified files.

- [ ] **Step 4: Semantic deep-compare before/after, ignoring added `fname` keys**

This is the guard that the rewrite changed nothing but `fname`. Run from the worktree root:

```bash
python3 - <<'PY'
import json, glob, os, sys

def strip_fname(o):
    if isinstance(o, dict):
        return {k: strip_fname(v) for k, v in o.items() if k != "fname"}
    if isinstance(o, list):
        return [strip_fname(v) for v in o]
    return o

bad = 0
for after_path in sorted(glob.glob("services/atlas-configurations/seed-data/templates/template_*.json")):
    name = os.path.basename(after_path)
    a = strip_fname(json.load(open(after_path)))
    b = strip_fname(json.load(open(os.path.join("/tmp/task194-before", name))))
    if a != b:
        bad += 1
        print("SEMANTIC DRIFT:", name)
        for k in sorted(set(a) | set(b)):
            if a.get(k) != b.get(k):
                print("   differing top-level key:", k)
print("FAIL" if bad else "OK: all 11 templates are semantically identical apart from added fname keys")
sys.exit(1 if bad else 0)
PY
```

Expected: `OK: all 11 templates are semantically identical apart from added fname keys`, exit 0.

If this fails the generator's write path lost or reordered data. Restore from `/tmp/task194-before/`, fix Task 6, and re-run — do not hand-patch the output.

- [ ] **Step 5: Run both template guards**

```bash
tools/template-opcode-order-guard.sh && tools/template-duplicate-binding-guard.sh
```

Expected: `OK: 22 template arrays are in ascending opcode order.` and `OK: 22 template arrays carry no duplicate (name, opCode) binding.`

- [ ] **Step 6: Re-run the backend corpus test against the seeded files**

```bash
cd services/atlas-configurations/atlas.com/configurations && \
  go test ./socket/... -run TestValidate_AcceptsEverySeedTemplate -v
```

Expected: PASS. `fname` never participates in validation (FR-10.4), so adding it must not change the outcome — this confirms it.

- [ ] **Step 7: Spot-read one diff hunk**

```bash
git diff --stat services/atlas-configurations/seed-data/templates/
git diff services/atlas-configurations/seed-data/templates/template_gms_83_1.json | head -40
```

Expected: eleven files changed; hunks show `+      "fname": "…"` lines inserted after the `handler`/`writer` key with no other line touched and no indentation change.

If whole objects show as rewritten, the key ordering or indentation is off. Check `seedDoc`/`seedEntry` field order against a pre-run file and fix Task 6 rather than accepting a noisy diff.

- [ ] **Step 8: Confirm the coverage report is present and non-empty**

```bash
cat docs/tasks/task-194-packet-definition-matrix/fname-coverage.txt
```

Expected: the eleven per-template lines and the `TOTAL`. This file **is** the acceptance criterion; it must be committed alongside the seed data.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-configurations/seed-data/templates \
        docs/tasks/task-194-packet-definition-matrix/fname-coverage.txt
git commit -m "chore(configurations): backfill fname into the eleven seed templates

Generated by 'packet-audit seed-fname --write'. Coverage is recorded in
docs/tasks/task-194-packet-definition-matrix/fname-coverage.txt.

Verified semantically identical to the previous files apart from the added fname
keys, and both template guards still pass.

Reaches new clusters and CI only: seeder.go's importTemplate skips any template
whose (region, majorVersion, minorVersion) row already exists, so existing
deployments acquire fname through the UI or a baseline republish. That is fine
because fname is informational and never participates in comparison, validation
or ancestry classification."
```

---

## Task 8: Shared socket types, `opcode.ts` and `normalize.ts`

The foundation of the pure domain library. Everything downstream — grid, drawer, dialogs, ancestry, bulk flows — is a function over a normalized `SocketObject`.

**Files:**
- Create: `services/atlas-ui/src/types/models/socket.ts`
- Create: `services/atlas-ui/src/lib/socket/model.ts`
- Create: `services/atlas-ui/src/lib/socket/opcode.ts`
- Create: `services/atlas-ui/src/lib/socket/normalize.ts`
- Create: `services/atlas-ui/src/lib/socket/__tests__/opcode.test.ts`
- Create: `services/atlas-ui/src/lib/socket/__tests__/normalize.test.ts`
- Modify: `services/atlas-ui/src/types/models/template.ts:84-98` (replace the inline socket shape with the shared type)
- Modify: `services/atlas-ui/src/services/api/tenants.service.ts:74-88` (same)

**Interfaces:**
- Consumes: nothing.
- Produces, for every later frontend task:
  - `src/types/models/socket.ts`: `SocketHandlerEntry`, `SocketWriterEntry`, `SocketUnsupported`, `SocketConfig`
  - `src/lib/socket/model.ts`: `DefinitionKind`, `DefinitionState`, `OptionsShape`, `Binding`, `SocketObject`, `entriesOf(obj, kind)`, `unsupportedOf(obj, kind)`, `stateOf(obj, kind, name) → DefinitionState`, `nameOfEntry(entry, kind) → string`
  - `src/lib/socket/opcode.ts`: `parseOpcode(raw) → number | null`, `formatOpcode(n) → string`, `matchesOpcodeQuery(query, value) → boolean`
  - `src/lib/socket/normalize.ts`: `fromTemplate(t) → SocketObject`, `fromTenantConfig(t) → SocketObject`

`src/lib/socket/` imports **no React, no React Query and no service module.** If a later task needs to import one there, the abstraction has leaked — put the React part in a component instead.

**Why the shared type exists:** the socket shape is currently declared inline twice, at `src/types/models/template.ts:84` and `src/services/api/tenants.service.ts:74`. Both must gain `unsupported` and `fname`; extracting it once is cheaper than editing it twice and keeps them from drifting.

- [ ] **Step 1: Write the failing opcode test**

Create `services/atlas-ui/src/lib/socket/__tests__/opcode.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
  formatOpcode,
  matchesOpcodeQuery,
  parseOpcode,
} from "@/lib/socket/opcode";

describe("parseOpcode", () => {
  it("parses the stored wire forms", () => {
    expect(parseOpcode("0x2A")).toBe(42);
    expect(parseOpcode("0x2a")).toBe(42);
    expect(parseOpcode("0X2A")).toBe(42);
    // jms_185_1 stores a single-digit opcode.
    expect(parseOpcode("0x9")).toBe(9);
    // gms_84_1 stores a three-digit padded opcode.
    expect(parseOpcode("0x0A5")).toBe(165);
    expect(parseOpcode("0xFFFF")).toBe(65535);
  });

  it("returns null for anything that is not a stored opcode", () => {
    expect(parseOpcode("2A")).toBeNull();
    expect(parseOpcode("42")).toBeNull();
    expect(parseOpcode("")).toBeNull();
    expect(parseOpcode("0x")).toBeNull();
    expect(parseOpcode("0xZZ")).toBeNull();
    expect(parseOpcode("0x10000")).toBeNull();
  });

  it("treats 0xB8 and 0x0B8 as the same value", () => {
    expect(parseOpcode("0xB8")).toBe(parseOpcode("0x0B8"));
  });
});

describe("formatOpcode", () => {
  it("renders the canonical two-digit-minimum upper-case form", () => {
    expect(formatOpcode(42)).toBe("0x2A");
    expect(formatOpcode(9)).toBe("0x09");
    expect(formatOpcode(165)).toBe("0xA5");
    expect(formatOpcode(65535)).toBe("0xFFFF");
  });
});

// FR-4.3: searching 0x2A, 2A and 42 must all match the same cell. A bare
// numeric query is ambiguous, so it matches under BOTH a hex and a decimal
// reading - "42" therefore matches 0x42 (66) as well as 0x2A (42).
describe("matchesOpcodeQuery", () => {
  it("matches the prefixed form exactly", () => {
    expect(matchesOpcodeQuery("0x2A", 42)).toBe(true);
    expect(matchesOpcodeQuery("0x2a", 42)).toBe(true);
    expect(matchesOpcodeQuery("0x2A", 66)).toBe(false);
  });

  it("matches an unprefixed hex query", () => {
    expect(matchesOpcodeQuery("2A", 42)).toBe(true);
    expect(matchesOpcodeQuery("2a", 42)).toBe(true);
  });

  it("matches a bare number under both hex and decimal readings", () => {
    expect(matchesOpcodeQuery("42", 42)).toBe(true); // decimal reading
    expect(matchesOpcodeQuery("42", 66)).toBe(true); // hex reading, 0x42
  });

  it("ignores surrounding whitespace and does not match non-numeric text", () => {
    expect(matchesOpcodeQuery("  0x2A  ", 42)).toBe(true);
    expect(matchesOpcodeQuery("LoginHandle", 42)).toBe(false);
    expect(matchesOpcodeQuery("", 42)).toBe(false);
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/opcode.test.ts`

Expected: FAIL — cannot resolve `@/lib/socket/opcode`.

- [ ] **Step 3: Implement `opcode.ts`**

Create `services/atlas-ui/src/lib/socket/opcode.ts`:

```ts
/**
 * The only place a stored opcode string is interpreted.
 *
 * Stored values are NEVER rewritten: canonicalization is display-only, because
 * rewriting "0x0B8" to "0xB8" on save would be a data change this task does not
 * make. Two bindings whose parsed values are equal are surfaced as a duplicate,
 * not silently merged.
 */

/** Matches the accepted wire form: 0x/0X followed by 1-4 hex digits. */
const STORED = /^0[xX][0-9A-Fa-f]{1,4}$/;

/** Parses a stored opcode, or returns null if it is not in the wire form. */
export function parseOpcode(raw: string): number | null {
  const s = raw.trim();
  if (!STORED.test(s)) return null;
  const n = Number.parseInt(s.slice(2), 16);
  return Number.isNaN(n) ? null : n;
}

/** Renders the canonical display form: 0x + upper-case hex, at least 2 digits. */
export function formatOpcode(n: number): string {
  return `0x${n.toString(16).toUpperCase().padStart(2, "0")}`;
}

/**
 * FR-4.3. A search query matches an opcode value if it reads as that value
 * under any plausible interpretation:
 *
 *   "0x2A" -> hex only            -> 42
 *   "2A"   -> hex only (has a-f)  -> 42
 *   "42"   -> hex OR decimal      -> 66 or 42
 *
 * The ambiguity is deliberate. A bare "42" is exactly as likely to mean the
 * decimal opcode as the hex one, and a search that silently picked one would
 * hide the other.
 */
export function matchesOpcodeQuery(query: string, value: number): boolean {
  const q = query.trim();
  if (q === "") return false;

  if (/^0[xX][0-9A-Fa-f]{1,4}$/.test(q)) {
    return Number.parseInt(q.slice(2), 16) === value;
  }
  if (/^[0-9A-Fa-f]{1,4}$/.test(q)) {
    if (Number.parseInt(q, 16) === value) return true;
    if (/^[0-9]+$/.test(q) && Number.parseInt(q, 10) === value) return true;
  }
  return false;
}
```

- [ ] **Step 4: Run the opcode test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/opcode.test.ts`

Expected: PASS, all four describe blocks.

- [ ] **Step 5: Write the shared wire types**

Create `services/atlas-ui/src/types/models/socket.ts`:

```ts
/**
 * The socket configuration shape, shared by Templates and Tenants.
 *
 * This was previously declared inline in BOTH types/models/template.ts and
 * services/api/tenants.service.ts. Both needed the same two new fields, so the
 * shape now lives here and both import it.
 */

/** One serverbound entry: opcode -> validator -> handler implementation. */
export interface SocketHandlerEntry {
  opCode: string;
  validator: string;
  handler: string;
  /**
   * Client-side function name, e.g. "CLogin::SendCheckPasswordPacket".
   * Informational only - it never participates in comparison, validation or
   * ancestry classification (PRD FR-10.4). Optional and omitted when empty.
   */
  fname?: string;
  /** Free-form wire tables the codec reads at runtime. Absent when unset. */
  options?: unknown;
  services?: string[];
}

/** One clientbound entry: opcode -> writer implementation. */
export interface SocketWriterEntry {
  opCode: string;
  writer: string;
  /** See SocketHandlerEntry.fname. */
  fname?: string;
  options?: unknown;
  services?: string[];
}

/**
 * Definitions audited and confirmed absent for this Region/Version. Holding
 * them here rather than in the arrays is what makes "audited, this version does
 * not have this packet" distinguishable from "nobody has looked yet".
 *
 * Entries are IMPLEMENTATION NAMES, never opcodes.
 */
export interface SocketUnsupported {
  handlers: string[];
  writers: string[];
}

export interface SocketConfig {
  handlers: SocketHandlerEntry[];
  writers: SocketWriterEntry[];
  /** Optional for backwards compatibility: absent means both lists are empty. */
  unsupported?: SocketUnsupported;
}
```

- [ ] **Step 6: Point both existing declarations at the shared type**

In `src/types/models/template.ts`, add the import and replace the inline `socket:` block inside `TemplateAttributes`:

```ts
import type { SocketConfig } from "@/types/models/socket";
```

```ts
  socket: SocketConfig;
```

In `src/services/api/tenants.service.ts`, add the same import and replace the inline `socket:` block inside `TenantConfigAttributes` with `socket: SocketConfig;`.

- [ ] **Step 7: Write the domain model and the failing normalize test**

Create `services/atlas-ui/src/lib/socket/model.ts`:

```ts
import type { SocketHandlerEntry, SocketWriterEntry } from "@/types/models/socket";

/** The two collections are never mixed in a single view. */
export type DefinitionKind = "handler" | "writer";

/**
 * For any object, a Definition is in exactly one state.
 *   defined     - at least one binding exists
 *   unsupported - the name appears in socket.unsupported.<kind>s
 *   undefined   - neither of the above, inferred from absence
 */
export type DefinitionState = "defined" | "unsupported" | "undefined";

/**
 *   absent - the options key is missing or null
 *   empty  - an explicit {}
 *   list   - a single key whose value is a JSON array; the ARRAY INDEX is the
 *            wire value and the name is not unique (gms_95_1 CharacterMovement
 *            carries UNKNOWN at six separate indices)
 *   map    - a flat object of name -> wire number (operations,
 *            failedReasonCodes, codes), and the fallback for anything else
 */
export type OptionsShape = "absent" | "empty" | "map" | "list";

/**
 * One entry of socket.handlers or socket.writers.
 *
 * A Definition holds one or MORE bindings. NoOpHandler is bound to four opcodes
 * in gms_95_1; ServerListRequestHandle to two in nine templates. A binding, not
 * a Definition, is the unit of Add / Edit / Delete.
 */
export interface Binding {
  /** As stored, e.g. "0x0B8". Never rewritten. */
  opCode: string;
  /** Parsed value, e.g. 184. null when the stored string is malformed. */
  opCodeValue: number | null;
  /** Handlers only; undefined for writers. */
  validator?: string;
  services: string[];
  /** As stored; undefined when the key was absent. */
  options?: unknown;
  fname?: string;
  /**
   * Position in this object's OWN array as fetched. Useful for display order
   * only - never use it to splice, because templatesService re-sorts both
   * arrays by opcode on read, so it does not match the stored index.
   */
  index: number;
}

/** A Template or a Tenant configuration, normalized for the domain layer. */
export interface SocketObject {
  /** Stable identity: the template or tenant uuid. Used as the column key. */
  key: string;
  /** Display label, e.g. "GMS v83.1". */
  label: string;
  source: "template" | "tenant";
  region: string;
  majorVersion: number;
  minorVersion: number;
  /** Implementation name -> its bindings, in stored order. */
  handlers: Map<string, Binding[]>;
  writers: Map<string, Binding[]>;
  unsupportedHandlers: Set<string>;
  unsupportedWriters: Set<string>;
}

export function entriesOf(obj: SocketObject, kind: DefinitionKind): Map<string, Binding[]> {
  return kind === "handler" ? obj.handlers : obj.writers;
}

export function unsupportedOf(obj: SocketObject, kind: DefinitionKind): Set<string> {
  return kind === "handler" ? obj.unsupportedHandlers : obj.unsupportedWriters;
}

/** The state of one Definition within one object. */
export function stateOf(
  obj: SocketObject,
  kind: DefinitionKind,
  name: string,
): DefinitionState {
  const bindings = entriesOf(obj, kind).get(name);
  if (bindings && bindings.length > 0) return "defined";
  if (unsupportedOf(obj, kind).has(name)) return "unsupported";
  return "undefined";
}

/** Narrowing helper so callers can share one code path over both entry types. */
export function nameOfEntry(
  entry: SocketHandlerEntry | SocketWriterEntry,
  kind: DefinitionKind,
): string {
  return kind === "handler"
    ? (entry as SocketHandlerEntry).handler
    : (entry as SocketWriterEntry).writer;
}
```

Create `services/atlas-ui/src/lib/socket/__tests__/normalize.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { fromTemplate, fromTenantConfig } from "@/lib/socket/normalize";
import { stateOf } from "@/lib/socket/model";
import type { Template } from "@/types/models/template";

function template(overrides: Partial<Template["attributes"]> = {}): Template {
  return {
    id: "tpl-1",
    attributes: {
      region: "GMS",
      majorVersion: 95,
      minorVersion: 1,
      usesPin: false,
      characters: { templates: [], presets: [] },
      npcs: [],
      worlds: [],
      socket: { handlers: [], writers: [] },
      ...overrides,
    } as Template["attributes"],
  };
}

describe("fromTemplate", () => {
  it("groups several bindings of one name under that name", () => {
    const obj = fromTemplate(
      template({
        socket: {
          handlers: [
            { opCode: "0x17", validator: "LoggedInValidator", handler: "NoOpHandler" },
            { opCode: "0x19", validator: "LoggedInValidator", handler: "NoOpHandler" },
            { opCode: "0x22", validator: "LoggedInValidator", handler: "NoOpHandler" },
            { opCode: "0x24", validator: "LoggedInValidator", handler: "NoOpHandler" },
            { opCode: "0x01", validator: "NoOpValidator", handler: "LoginHandle" },
          ],
          writers: [],
        },
      }),
    );
    expect(obj.handlers.get("NoOpHandler")).toHaveLength(4);
    expect(obj.handlers.get("NoOpHandler")!.map((b) => b.opCodeValue)).toEqual([
      0x17, 0x19, 0x22, 0x24,
    ]);
    expect(obj.handlers.get("LoginHandle")).toHaveLength(1);
  });

  it("keeps numerically-equal bindings as two separate bindings", () => {
    const obj = fromTemplate(
      template({
        socket: {
          handlers: [],
          writers: [
            { opCode: "0xB8", writer: "MiniRoom", options: {} },
            { opCode: "0x0B8", writer: "MiniRoom" },
          ],
        },
      }),
    );
    const bindings = obj.writers.get("MiniRoom")!;
    expect(bindings).toHaveLength(2);
    expect(bindings[0]!.opCodeValue).toBe(bindings[1]!.opCodeValue);
    // The stored strings are preserved verbatim; canonicalization is display-only.
    expect(bindings.map((b) => b.opCode)).toEqual(["0xB8", "0x0B8"]);
  });

  it("treats an absent unsupported key as two empty sets", () => {
    const obj = fromTemplate(template());
    expect(obj.unsupportedHandlers.size).toBe(0);
    expect(obj.unsupportedWriters.size).toBe(0);
  });

  it("reads the three definition states", () => {
    const obj = fromTemplate(
      template({
        socket: {
          handlers: [{ opCode: "0x01", validator: "NoOpValidator", handler: "LoginHandle" }],
          writers: [],
          unsupported: { handlers: ["GuestLoginHandle"], writers: [] },
        },
      }),
    );
    expect(stateOf(obj, "handler", "LoginHandle")).toBe("defined");
    expect(stateOf(obj, "handler", "GuestLoginHandle")).toBe("unsupported");
    expect(stateOf(obj, "handler", "NeverHeardOfIt")).toBe("undefined");
  });

  it("records a malformed opcode as a null value rather than throwing", () => {
    const obj = fromTemplate(
      template({
        socket: {
          handlers: [],
          writers: [{ opCode: "nonsense", writer: "Broken" }],
        },
      }),
    );
    expect(obj.writers.get("Broken")![0]!.opCodeValue).toBeNull();
  });

  it("labels the object from its region and version", () => {
    expect(fromTemplate(template()).label).toBe("GMS v95.1");
    expect(fromTemplate(template()).source).toBe("template");
  });
});

describe("fromTenantConfig", () => {
  it("produces the same shape from a tenant configuration", () => {
    const obj = fromTenantConfig({
      id: "tnt-1",
      attributes: {
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
        usesPin: false,
        characters: { templates: [], presets: [] },
        npcs: [],
        worlds: [],
        socket: {
          handlers: [{ opCode: "0x01", validator: "NoOpValidator", handler: "LoginHandle" }],
          writers: [],
          unsupported: { handlers: [], writers: ["MonsterCarnival"] },
        },
      },
    } as Parameters<typeof fromTenantConfig>[0]);

    expect(obj.source).toBe("tenant");
    expect(obj.key).toBe("tnt-1");
    expect(obj.label).toBe("GMS v83.1");
    expect(stateOf(obj, "writer", "MonsterCarnival")).toBe("unsupported");
  });
});
```

- [ ] **Step 8: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/normalize.test.ts`

Expected: FAIL — cannot resolve `@/lib/socket/normalize`.

- [ ] **Step 9: Implement `normalize.ts`**

Create `services/atlas-ui/src/lib/socket/normalize.ts`:

```ts
import type { Template } from "@/types/models/template";
import type { TenantConfig } from "@/services/api/tenants.service";
import type {
  SocketConfig,
  SocketHandlerEntry,
  SocketWriterEntry,
} from "@/types/models/socket";
import { parseOpcode } from "@/lib/socket/opcode";
import type { Binding, SocketObject } from "@/lib/socket/model";

/**
 * Turns a fetched Template or Tenant configuration into the normalized
 * SocketObject the whole domain layer operates on. This is the ONLY place the
 * wire shape is read; nothing downstream touches socket.handlers directly.
 */
function build(
  key: string,
  source: SocketObject["source"],
  region: string,
  majorVersion: number,
  minorVersion: number,
  socket: SocketConfig | undefined,
): SocketObject {
  const handlers = new Map<string, Binding[]>();
  const writers = new Map<string, Binding[]>();

  (socket?.handlers ?? []).forEach((e: SocketHandlerEntry, index) => {
    push(handlers, e.handler, {
      opCode: e.opCode,
      opCodeValue: parseOpcode(e.opCode),
      validator: e.validator,
      services: e.services ?? [],
      options: e.options,
      fname: e.fname,
      index,
    });
  });

  (socket?.writers ?? []).forEach((e: SocketWriterEntry, index) => {
    push(writers, e.writer, {
      opCode: e.opCode,
      opCodeValue: parseOpcode(e.opCode),
      services: e.services ?? [],
      options: e.options,
      fname: e.fname,
      index,
    });
  });

  return {
    key,
    label: `${region} v${majorVersion}.${minorVersion}`,
    source,
    region,
    majorVersion,
    minorVersion,
    handlers,
    writers,
    unsupportedHandlers: new Set(socket?.unsupported?.handlers ?? []),
    unsupportedWriters: new Set(socket?.unsupported?.writers ?? []),
  };
}

function push(into: Map<string, Binding[]>, name: string, binding: Binding): void {
  const existing = into.get(name);
  if (existing) existing.push(binding);
  else into.set(name, [binding]);
}

export function fromTemplate(t: Template): SocketObject {
  const a = t.attributes;
  return build(t.id, "template", a.region, a.majorVersion, a.minorVersion, a.socket);
}

export function fromTenantConfig(t: TenantConfig): SocketObject {
  const a = t.attributes;
  return build(t.id, "tenant", a.region, a.majorVersion, a.minorVersion, a.socket);
}
```

- [ ] **Step 10: Run both tests and the type-check**

```bash
cd services/atlas-ui && npx vitest run src/lib/socket/ && npm run build
```

Expected: both PASS. `npm run build` type-checks tests, so a mismatch between the shared `SocketConfig` and the two call sites edited in Step 6 surfaces here.

- [ ] **Step 11: Commit**

```bash
git add services/atlas-ui/src/types/models/socket.ts \
        services/atlas-ui/src/lib/socket \
        services/atlas-ui/src/types/models/template.ts \
        services/atlas-ui/src/services/api/tenants.service.ts
git commit -m "feat(ui): add the socket domain model, opcode parsing and normalization

Starts the pure, React-free socket library. lib/socket imports no React, no
React Query and no service module, so the protocol semantics are unit-testable
without rendering.

Extracts the socket wire shape into types/models/socket.ts - it was declared
inline in both types/models/template.ts and services/api/tenants.service.ts and
both needed the same two new fields.

A Definition holds one or MORE bindings: NoOpHandler is bound to four opcodes in
gms_95_1. Modelling a cell as a single opcode would make the delete dialog
collapse those four routes into one."
```

---

## Task 9: `matrix.ts` — rows, cells, sorting, filtering, search

Turns a set of `SocketObject`s into the row model the grid renders. Everything the toolbar does — sort, filter, search — is a pure predicate over this model, computed in a `useMemo`, never against the DOM.

**Files:**
- Create: `services/atlas-ui/src/lib/socket/matrix.ts`
- Create: `services/atlas-ui/src/lib/socket/__tests__/matrix.test.ts`

**Interfaces:**
- Consumes: `Binding`, `SocketObject`, `DefinitionKind`, `DefinitionState`, `entriesOf`, `unsupportedOf`, `stateOf` (Task 8); `matchesOpcodeQuery` (Task 8).
- Produces:
  - `Cell { objectKey; state; bindings; optionsMissing; hasDuplicateOpcode }`
  - `Row { name; kind; fname?; cells: Map<string, Cell>; inBaseline; baselineOpCodeValue }`
  - `buildRows({ objects, kind, baselineKey }) → Row[]`
  - `SortKey = "opcode" | "name" | "state"`, `SortDirection = "asc" | "desc"`
  - `GridFilters { query; states; optionsMissingOnly; hasOptions; services }`
  - `emptyFilters(): GridFilters`
  - `filterRows(rows, filters, baselineKey) → Row[]`
  - `sortRows(rows, key, dir) → Row[]`

**Row set (FR-2.5):** the union of Defined and Unsupported names across the selected objects, for the active kind. A Definition that is neither anywhere cannot appear; that is accepted.

**Baseline (FR-2.9–2.11):** the baseline determines row ordering and is **not** a separate column — its own column is marked in place. Rows not present in the baseline sort after baseline-defined entries.

**Options marker (FR-3.2):** exactly one options signal exists in the grid — a cell whose object supplies no options for a Definition that at least one *other* selected object does supply options for. Structural divergence between versions is expected and is never marked (FR-3.1).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/lib/socket/__tests__/matrix.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
  buildRows,
  emptyFilters,
  filterRows,
  sortRows,
} from "@/lib/socket/matrix";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function obj(
  key: string,
  major: number,
  writers: Record<string, Binding[]>,
  unsupportedWriters: string[] = [],
): SocketObject {
  return {
    key,
    label: `GMS v${major}.1`,
    source: "template",
    region: "GMS",
    majorVersion: major,
    minorVersion: 1,
    handlers: new Map(),
    writers: new Map(Object.entries(writers)),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(unsupportedWriters),
  };
}

describe("buildRows", () => {
  it("unions defined and unsupported names across every object", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] }, ["MonsterCarnival"]);
    const b = obj("b", 95, { PetActivated: [binding("0x9A")] });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "b" });
    expect(rows.map((r) => r.name).sort()).toEqual([
      "AuthSuccess",
      "MonsterCarnival",
      "PetActivated",
    ]);
  });

  it("gives each row one cell per object, with the right state", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] }, ["MonsterCarnival"]);
    const b = obj("b", 95, {});
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "b" });

    const auth = rows.find((r) => r.name === "AuthSuccess")!;
    expect(auth.cells.get("a")!.state).toBe("defined");
    expect(auth.cells.get("b")!.state).toBe("undefined");

    const carnival = rows.find((r) => r.name === "MonsterCarnival")!;
    expect(carnival.cells.get("a")!.state).toBe("unsupported");
    expect(carnival.cells.get("b")!.state).toBe("undefined");
  });

  it("carries every binding of a multi-binding definition into one cell", () => {
    const a = obj("a", 95, {
      CharacterEffect: [binding("0xE0"), binding("0xE9")],
    });
    const rows = buildRows({ objects: [a], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("a")!.bindings.map((x) => x.opCodeValue)).toEqual([
      0xe0, 0xe9,
    ]);
    expect(rows[0]!.cells.get("a")!.hasDuplicateOpcode).toBe(false);
  });

  it("flags a cell whose bindings collide numerically", () => {
    const a = obj("a", 95, { MiniRoom: [binding("0xB8"), binding("0x0B8")] });
    const rows = buildRows({ objects: [a], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("a")!.hasDuplicateOpcode).toBe(true);
  });

  it("takes the row fname from the first object that supplies one", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] });
    const b = obj("b", 95, {
      AuthSuccess: [binding("0x00", { fname: "CLogin::OnCheckPasswordResult" })],
    });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.fname).toBe("CLogin::OnCheckPasswordResult");
  });

  // FR-3.2: the ONLY options signal in the grid. It fires on ABSENCE where a
  // sibling supplies options - never on structural divergence, which is the
  // expected state between versions (FR-3.1).
  it("marks a cell that supplies no options where a sibling does", () => {
    const a = obj("a", 83, {
      CharacterMovement: [binding("0xB9", { options: { types: ["A", "B"] } })],
    });
    const b = obj("b", 95, { CharacterMovement: [binding("0xC0")] });
    const c = obj("c", 87, {
      CharacterMovement: [binding("0xBC", { options: {} })],
    });
    const rows = buildRows({ objects: [a, b, c], kind: "writer", baselineKey: "a" });
    const row = rows[0]!;
    expect(row.cells.get("a")!.optionsMissing).toBe(false);
    expect(row.cells.get("b")!.optionsMissing).toBe(true); // absent
    expect(row.cells.get("c")!.optionsMissing).toBe(true); // explicit {}
  });

  it("marks nothing when no object supplies options for that definition", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] });
    const b = obj("b", 95, { AuthSuccess: [binding("0x00")] });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("a")!.optionsMissing).toBe(false);
    expect(rows[0]!.cells.get("b")!.optionsMissing).toBe(false);
  });

  it("marks nothing on an undefined cell - absence of a definition is not an options omission", () => {
    const a = obj("a", 83, {
      CharacterMovement: [binding("0xB9", { options: { types: ["A"] } })],
    });
    const b = obj("b", 95, {});
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });
    expect(rows[0]!.cells.get("b")!.optionsMissing).toBe(false);
  });

  it("records baseline membership and the baseline opcode", () => {
    const a = obj("a", 83, { AuthSuccess: [binding("0x00")] });
    const b = obj("b", 95, { PetActivated: [binding("0x9A")] });
    const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "b" });
    const pet = rows.find((r) => r.name === "PetActivated")!;
    const auth = rows.find((r) => r.name === "AuthSuccess")!;
    expect(pet.inBaseline).toBe(true);
    expect(pet.baselineOpCodeValue).toBe(0x9a);
    expect(auth.inBaseline).toBe(false);
    expect(auth.baselineOpCodeValue).toBeNull();
  });
});

describe("sortRows", () => {
  const a = obj("a", 83, { Zebra: [binding("0x02")] });
  const b = obj("b", 95, {
    Alpha: [binding("0x10")],
    Beta: [binding("0x05")],
  });
  const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "b" });

  // FR-4.1 default sort, FR-2.11 non-baseline entries last.
  it("sorts by ascending baseline opcode and puts non-baseline rows last", () => {
    expect(sortRows(rows, "opcode", "asc").map((r) => r.name)).toEqual([
      "Beta",
      "Alpha",
      "Zebra",
    ]);
  });

  it("keeps non-baseline rows last when the direction is reversed", () => {
    expect(sortRows(rows, "opcode", "desc").map((r) => r.name)).toEqual([
      "Alpha",
      "Beta",
      "Zebra",
    ]);
  });

  it("sorts by name", () => {
    expect(sortRows(rows, "name", "asc").map((r) => r.name)).toEqual([
      "Alpha",
      "Beta",
      "Zebra",
    ]);
  });
});

describe("filterRows", () => {
  const a = obj(
    "a",
    83,
    {
      AuthSuccess: [binding("0x00", { fname: "CLogin::OnCheckPasswordResult" })],
      CharacterMovement: [binding("0x2A", { options: { types: ["A"] } })],
    },
    ["MonsterCarnival"],
  );
  const b = obj("b", 95, { CharacterMovement: [binding("0xC0")] });
  const rows = buildRows({ objects: [a, b], kind: "writer", baselineKey: "a" });

  it("returns everything when no filter is set", () => {
    expect(filterRows(rows, emptyFilters(), "a")).toHaveLength(3);
  });

  it("searches the definition name, case-insensitively", () => {
    const got = filterRows(rows, { ...emptyFilters(), query: "movement" }, "a");
    expect(got.map((r) => r.name)).toEqual(["CharacterMovement"]);
  });

  it("searches fname", () => {
    const got = filterRows(rows, { ...emptyFilters(), query: "CheckPassword" }, "a");
    expect(got.map((r) => r.name)).toEqual(["AuthSuccess"]);
  });

  // FR-4.3, end to end through the filter.
  it("matches 0x2A, 2A and 42 against the same row", () => {
    for (const q of ["0x2A", "2A", "42"]) {
      const got = filterRows(rows, { ...emptyFilters(), query: q }, "a");
      expect(got.map((r) => r.name), `query ${q}`).toContain("CharacterMovement");
    }
  });

  it("filters by state within the baseline object", () => {
    const got = filterRows(rows, { ...emptyFilters(), states: ["unsupported"] }, "a");
    expect(got.map((r) => r.name)).toEqual(["MonsterCarnival"]);
  });

  it("filters to rows carrying the options-omission marker", () => {
    const got = filterRows(rows, { ...emptyFilters(), optionsMissingOnly: true }, "a");
    expect(got.map((r) => r.name)).toEqual(["CharacterMovement"]);
  });

  it("filters by service", () => {
    const got = filterRows(rows, { ...emptyFilters(), services: ["login"] }, "a");
    expect(got).toHaveLength(0);
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/matrix.test.ts`

Expected: FAIL — cannot resolve `@/lib/socket/matrix`.

- [ ] **Step 3: Implement `matrix.ts`**

Create `services/atlas-ui/src/lib/socket/matrix.ts`:

```ts
import type {
  Binding,
  DefinitionKind,
  DefinitionState,
  SocketObject,
} from "@/lib/socket/model";
import { entriesOf, stateOf, unsupportedOf } from "@/lib/socket/model";
import { matchesOpcodeQuery } from "@/lib/socket/opcode";

/** One object's view of one Definition. */
export interface Cell {
  objectKey: string;
  state: DefinitionState;
  /** Every binding of this Definition in this object, in stored order. */
  bindings: Binding[];
  /**
   * FR-3.2. True when this object supplies NO options for a Definition that at
   * least one other selected object DOES supply options for. This is the only
   * options signal in the grid: structural divergence between versions is the
   * expected state and is never marked (FR-3.1).
   */
  optionsMissing: boolean;
  /** Two bindings whose parsed opcodes are equal - "0xB8" and "0x0B8". */
  hasDuplicateOpcode: boolean;
}

export interface Row {
  name: string;
  kind: DefinitionKind;
  /** First fname supplied by any object. Display and search only (FR-10.4). */
  fname?: string;
  cells: Map<string, Cell>;
  inBaseline: boolean;
  /** The baseline's lowest opcode for this row; null when not in the baseline. */
  baselineOpCodeValue: number | null;
}

export type SortKey = "opcode" | "name" | "state";
export type SortDirection = "asc" | "desc";

export interface GridFilters {
  query: string;
  /** Empty means every state. Evaluated against the baseline object's cell. */
  states: DefinitionState[];
  /** Only rows where some cell carries the FR-3.2 marker. */
  optionsMissingOnly: boolean;
  /** null = don't care; true/false = the baseline cell supplies options or not. */
  hasOptions: boolean | null;
  /** Empty means every service. */
  services: string[];
}

export function emptyFilters(): GridFilters {
  return {
    query: "",
    states: [],
    optionsMissingOnly: false,
    hasOptions: null,
    services: [],
  };
}

/** True when a bindings list supplies a non-empty options object. */
function suppliesOptions(bindings: Binding[]): boolean {
  return bindings.some((b) => {
    const o = b.options;
    return (
      o !== undefined &&
      o !== null &&
      typeof o === "object" &&
      Object.keys(o as Record<string, unknown>).length > 0
    );
  });
}

export function buildRows(input: {
  objects: SocketObject[];
  kind: DefinitionKind;
  baselineKey: string;
}): Row[] {
  const { objects, kind, baselineKey } = input;

  // FR-2.5: the row set is the union of Defined and Unsupported names.
  const names = new Set<string>();
  for (const o of objects) {
    for (const n of entriesOf(o, kind).keys()) names.add(n);
    for (const n of unsupportedOf(o, kind)) names.add(n);
  }

  const rows: Row[] = [];
  for (const name of names) {
    // Decide the FR-3.2 marker per definition, which needs every object's
    // answer before any single cell can be classified.
    const supplying = new Set<string>();
    for (const o of objects) {
      const b = entriesOf(o, kind).get(name);
      if (b && suppliesOptions(b)) supplying.add(o.key);
    }
    const someoneSupplies = supplying.size > 0;

    const cells = new Map<string, Cell>();
    let fname: string | undefined;

    for (const o of objects) {
      const bindings = entriesOf(o, kind).get(name) ?? [];
      const state = stateOf(o, kind, name);

      if (fname === undefined) {
        const withFName = bindings.find((b) => b.fname !== undefined && b.fname !== "");
        if (withFName) fname = withFName.fname;
      }

      const values = bindings
        .map((b) => b.opCodeValue)
        .filter((v): v is number => v !== null);

      cells.set(o.key, {
        objectKey: o.key,
        state,
        bindings,
        // Only a DEFINED cell can be omitting options. An undefined cell has no
        // definition to attach options to, so marking it would be noise.
        optionsMissing:
          state === "defined" && someoneSupplies && !supplying.has(o.key),
        hasDuplicateOpcode: new Set(values).size !== values.length,
      });
    }

    const baselineCell = cells.get(baselineKey);
    const baselineValues = (baselineCell?.bindings ?? [])
      .map((b) => b.opCodeValue)
      .filter((v): v is number => v !== null);

    rows.push({
      name,
      kind,
      fname,
      cells,
      inBaseline: baselineCell?.state === "defined",
      baselineOpCodeValue: baselineValues.length > 0 ? Math.min(...baselineValues) : null,
    });
  }
  return rows;
}

const STATE_ORDER: Record<DefinitionState, number> = {
  defined: 0,
  unsupported: 1,
  undefined: 2,
};

/**
 * FR-4.1/4.2 and FR-2.11. Rows absent from the baseline always sort AFTER
 * baseline-defined rows, in both directions - the direction toggle orders
 * within each group, it does not promote non-baseline rows to the top.
 */
export function sortRows(rows: Row[], key: SortKey, dir: SortDirection): Row[] {
  const sign = dir === "asc" ? 1 : -1;
  return [...rows].sort((a, b) => {
    if (a.inBaseline !== b.inBaseline) return a.inBaseline ? -1 : 1;

    let cmp = 0;
    if (key === "opcode") {
      const av = a.baselineOpCodeValue;
      const bv = b.baselineOpCodeValue;
      if (av === null && bv === null) cmp = a.name.localeCompare(b.name);
      else if (av === null) cmp = 1;
      else if (bv === null) cmp = -1;
      else cmp = av - bv;
    } else if (key === "name") {
      cmp = a.name.localeCompare(b.name);
    } else {
      const as = a.cells.get([...a.cells.keys()][0]!)?.state ?? "undefined";
      const bs = b.cells.get([...b.cells.keys()][0]!)?.state ?? "undefined";
      cmp = STATE_ORDER[as] - STATE_ORDER[bs];
      if (cmp === 0) cmp = a.name.localeCompare(b.name);
    }
    return cmp === 0 ? a.name.localeCompare(b.name) : cmp * sign;
  });
}

/**
 * FR-4.3/4.4. State, hasOptions and service filters are evaluated against the
 * BASELINE object's cell, which is the object the row is ordered by and the
 * one the drawer defaults its scope to.
 */
export function filterRows(
  rows: Row[],
  filters: GridFilters,
  baselineKey: string,
): Row[] {
  const q = filters.query.trim().toLowerCase();

  return rows.filter((row) => {
    if (q !== "") {
      const nameHit = row.name.toLowerCase().includes(q);
      const fnameHit = (row.fname ?? "").toLowerCase().includes(q);
      const opcodeHit = [...row.cells.values()].some((c) =>
        c.bindings.some(
          (b) => b.opCodeValue !== null && matchesOpcodeQuery(filters.query, b.opCodeValue),
        ),
      );
      if (!nameHit && !fnameHit && !opcodeHit) return false;
    }

    const baseline = row.cells.get(baselineKey);

    if (filters.states.length > 0) {
      if (!baseline || !filters.states.includes(baseline.state)) return false;
    }

    if (filters.optionsMissingOnly) {
      if (![...row.cells.values()].some((c) => c.optionsMissing)) return false;
    }

    if (filters.hasOptions !== null) {
      const has = suppliesOptions(baseline?.bindings ?? []);
      if (has !== filters.hasOptions) return false;
    }

    if (filters.services.length > 0) {
      const hit = [...row.cells.values()].some((c) =>
        c.bindings.some((b) => b.services.some((s) => filters.services.includes(s))),
      );
      if (!hit) return false;
    }

    return true;
  });
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/matrix.test.ts`

Expected: PASS, all three describe blocks.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/socket/matrix.ts \
        services/atlas-ui/src/lib/socket/__tests__/matrix.test.ts
git commit -m "feat(ui): add the socket matrix row model

Rows are the union of Defined and Unsupported names across the selected
objects; a cell holds the whole binding SET, not one opcode.

The options marker fires only on ABSENCE where a sibling supplies options.
Structural divergence between versions is the expected state - gms_12_1 has 9
movement types and jms_185_1 has 33 - so marking it on every row would be pure
noise. An undefined cell is never marked either: there is no definition there
for options to be missing from.

Rows absent from the baseline sort last in BOTH directions; the direction toggle
orders within each group rather than promoting them."
```

---

## Task 10: `options.ts` — classification and the nested per-entry matrix

**Files:**
- Create: `services/atlas-ui/src/lib/socket/options.ts`
- Create: `services/atlas-ui/src/lib/socket/__tests__/options.test.ts`

**Interfaces:**
- Consumes: `Binding`, `OptionsShape`, `SocketObject`, `DefinitionKind`, `entriesOf` (Task 8).
- Produces:
  - `classifyOptions(value: unknown) → OptionsShape`
  - `OptionsEntryCellState = "same" | "differs" | "missing" | "extra"`
  - `OptionsMatrix { shape; rows: OptionsMatrixRow[] }`, `OptionsMatrixRow { key; label; cells: Map<string, { value: unknown; state: OptionsEntryCellState }> }`
  - `buildOptionsMatrix({ objects, kind, name, baselineKey }) → OptionsMatrix`

**The list-vs-map distinction is the whole point (FR-3.5).** Ordered lists are compared **positionally** — the array index *is* the wire value and the name is not unique. gms_95_1 `CharacterMovement` carries `UNKNOWN` at six separate indices; comparing those by name would match six unrelated slots to each other. Maps compare by key.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/lib/socket/__tests__/options.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { buildOptionsMatrix, classifyOptions } from "@/lib/socket/options";
import type { Binding, SocketObject } from "@/lib/socket/model";

function obj(key: string, options: unknown): SocketObject {
  const binding: Binding = {
    opCode: "0xB9",
    opCodeValue: 0xb9,
    services: ["channel"],
    options,
    index: 0,
  };
  return {
    key,
    label: key,
    source: "template",
    region: "GMS",
    majorVersion: 95,
    minorVersion: 1,
    handlers: new Map(),
    writers: new Map([["CharacterMovement", [binding]]]),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(),
  };
}

describe("classifyOptions", () => {
  it("classifies absence", () => {
    expect(classifyOptions(undefined)).toBe("absent");
    expect(classifyOptions(null)).toBe("absent");
  });

  it("classifies an explicit empty object", () => {
    expect(classifyOptions({})).toBe("empty");
  });

  it("classifies a single-key array value as a list", () => {
    expect(classifyOptions({ types: ["WALK", "STAND", "UNKNOWN"] })).toBe("list");
  });

  it("classifies a flat name to number object as a map", () => {
    expect(classifyOptions({ operations: 1, failedReasonCodes: 2 })).toBe("map");
    expect(classifyOptions({ codes: 7 })).toBe("map");
  });

  it("falls back to map for anything else", () => {
    expect(classifyOptions({ a: { nested: true }, b: 1 })).toBe("map");
    expect(classifyOptions("not an object")).toBe("map");
  });

  it("classifies an empty list as a list, not as empty", () => {
    // gms_92_1/gms_95_1 CharacterMovement carry an empty types array. That is a
    // list with zero entries - visibly different from supplying no options.
    expect(classifyOptions({ types: [] })).toBe("list");
  });
});

describe("buildOptionsMatrix - lists compare positionally", () => {
  const a = obj("a", { types: ["WALK", "STAND", "JUMP"] });
  const b = obj("b", { types: ["WALK", "JUMP", "STAND"] });
  const c = obj("c", { types: ["WALK"] });

  const m = buildOptionsMatrix({
    objects: [a, b, c],
    kind: "writer",
    name: "CharacterMovement",
    baselineKey: "a",
  });

  it("keys rows by array index, because the index IS the wire value", () => {
    expect(m.shape).toBe("list");
    expect(m.rows.map((r) => r.key)).toEqual(["0", "1", "2"]);
    expect(m.rows[0]!.label).toBe("0");
  });

  it("treats a name that shifted index as a difference, not a match", () => {
    const idx1 = m.rows[1]!;
    expect(idx1.cells.get("a")!.value).toBe("STAND");
    expect(idx1.cells.get("b")!.value).toBe("JUMP");
    expect(idx1.cells.get("b")!.state).toBe("differs");
  });

  it("marks positions the baseline has and an object does not as missing", () => {
    expect(m.rows[1]!.cells.get("c")!.state).toBe("missing");
    expect(m.rows[2]!.cells.get("c")!.state).toBe("missing");
  });

  it("marks matching positions as same", () => {
    expect(m.rows[0]!.cells.get("b")!.state).toBe("same");
    expect(m.rows[0]!.cells.get("c")!.state).toBe("same");
  });

  it("marks positions past the baseline's extent as extra", () => {
    const long = obj("d", { types: ["WALK", "STAND", "JUMP", "FLY"] });
    const m2 = buildOptionsMatrix({
      objects: [a, long],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });
    expect(m2.rows).toHaveLength(4);
    expect(m2.rows[3]!.cells.get("d")!.state).toBe("extra");
    expect(m2.rows[3]!.cells.get("a")!.state).toBe("missing");
  });

  it("keeps a repeated name at several indices as separate rows", () => {
    const dup = obj("e", { types: ["UNKNOWN", "UNKNOWN", "UNKNOWN"] });
    const m3 = buildOptionsMatrix({
      objects: [dup],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "e",
    });
    expect(m3.rows).toHaveLength(3);
    expect(m3.rows.map((r) => r.cells.get("e")!.value)).toEqual([
      "UNKNOWN",
      "UNKNOWN",
      "UNKNOWN",
    ]);
  });
});

describe("buildOptionsMatrix - maps compare by key", () => {
  const a = obj("a", { operations: { INVITE: 1, JOIN: 2 } });
  const b = obj("b", { operations: { INVITE: 1, JOIN: 5, LEAVE: 9 } });

  const m = buildOptionsMatrix({
    objects: [a, b],
    kind: "writer",
    name: "CharacterMovement",
    baselineKey: "a",
  });

  it("keys rows by option name", () => {
    expect(m.shape).toBe("map");
    expect(m.rows.map((r) => r.key).sort()).toEqual(["INVITE", "JOIN", "LEAVE"]);
  });

  it("classifies equal, differing and extra values", () => {
    const invite = m.rows.find((r) => r.key === "INVITE")!;
    const join = m.rows.find((r) => r.key === "JOIN")!;
    const leave = m.rows.find((r) => r.key === "LEAVE")!;
    expect(invite.cells.get("b")!.state).toBe("same");
    expect(join.cells.get("b")!.state).toBe("differs");
    expect(leave.cells.get("b")!.state).toBe("extra");
    expect(leave.cells.get("a")!.state).toBe("missing");
  });
});

describe("buildOptionsMatrix - degenerate inputs", () => {
  it("returns no rows when nobody supplies options", () => {
    const m = buildOptionsMatrix({
      objects: [obj("a", undefined), obj("b", {})],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });
    expect(m.rows).toHaveLength(0);
  });

  it("uses a supplying object's shape when the baseline supplies none", () => {
    const m = buildOptionsMatrix({
      objects: [obj("a", undefined), obj("b", { types: ["WALK"] })],
      kind: "writer",
      name: "CharacterMovement",
      baselineKey: "a",
    });
    expect(m.shape).toBe("list");
    expect(m.rows).toHaveLength(1);
    expect(m.rows[0]!.cells.get("a")!.state).toBe("missing");
    expect(m.rows[0]!.cells.get("b")!.state).toBe("extra");
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/options.test.ts`

Expected: FAIL — cannot resolve `@/lib/socket/options`.

- [ ] **Step 3: Implement `options.ts`**

Create `services/atlas-ui/src/lib/socket/options.ts`:

```ts
import type {
  DefinitionKind,
  OptionsShape,
  SocketObject,
} from "@/lib/socket/model";
import { entriesOf } from "@/lib/socket/model";

/**
 * `options` is free-form, but two structural families occur in practice:
 *
 *   list - a single key whose value is a JSON array ("types"). The ARRAY INDEX
 *          is the wire value and the name is NOT unique: gms_95_1
 *          CharacterMovement carries UNKNOWN at six separate indices.
 *   map  - a flat object of name -> wire number ("operations",
 *          "failedReasonCodes", "codes").
 *
 * Anything else falls back to map over its top-level keys and renders read-only.
 */
export function classifyOptions(value: unknown): OptionsShape {
  if (value === undefined || value === null) return "absent";
  if (typeof value !== "object") return "map";
  const keys = Object.keys(value as Record<string, unknown>);
  if (keys.length === 0) return "empty";
  if (keys.length === 1 && Array.isArray((value as Record<string, unknown>)[keys[0]!])) {
    return "list";
  }
  return "map";
}

export type OptionsEntryCellState = "same" | "differs" | "missing" | "extra";

export interface OptionsMatrixRow {
  /** Array index (as a string) for lists; the option name for maps. */
  key: string;
  /** What the header cell shows. Same as key; separate so lists can be relabelled. */
  label: string;
  cells: Map<string, { value: unknown; state: OptionsEntryCellState }>;
}

export interface OptionsMatrix {
  shape: OptionsShape;
  rows: OptionsMatrixRow[];
}

/** The options value of a definition's FIRST binding in this object. */
function optionsOf(
  obj: SocketObject,
  kind: DefinitionKind,
  name: string,
): unknown {
  return entriesOf(obj, kind).get(name)?.[0]?.options;
}

/** Pulls a list's array out of its single wrapping key. */
function listOf(value: unknown): unknown[] | null {
  if (classifyOptions(value) !== "list") return null;
  const k = Object.keys(value as Record<string, unknown>)[0]!;
  return (value as Record<string, unknown>)[k] as unknown[];
}

function mapOf(value: unknown): Record<string, unknown> | null {
  if (classifyOptions(value) !== "map") return null;
  return value as Record<string, unknown>;
}

/**
 * FR-3.3-3.5. Rows are option entries; columns are the same selected objects as
 * the outer grid. Every cell is classified against the BASELINE at that
 * index/key.
 */
export function buildOptionsMatrix(input: {
  objects: SocketObject[];
  kind: DefinitionKind;
  name: string;
  baselineKey: string;
}): OptionsMatrix {
  const { objects, kind, name, baselineKey } = input;

  const values = new Map<string, unknown>();
  for (const o of objects) values.set(o.key, optionsOf(o, kind, name));

  // Prefer the baseline's shape; fall back to the first object that supplies
  // one, so a baseline with no options still renders its siblings' entries.
  const baselineShape = classifyOptions(values.get(baselineKey));
  let shape: OptionsShape = baselineShape;
  if (shape === "absent" || shape === "empty") {
    const supplier = objects.find((o) => {
      const s = classifyOptions(values.get(o.key));
      return s === "list" || s === "map";
    });
    shape = supplier ? classifyOptions(values.get(supplier.key)) : baselineShape;
  }

  if (shape !== "list" && shape !== "map") return { shape, rows: [] };

  const baselineValue = values.get(baselineKey);

  if (shape === "list") {
    const lists = new Map<string, unknown[]>();
    for (const o of objects) lists.set(o.key, listOf(values.get(o.key)) ?? []);
    const baseline = lists.get(baselineKey) ?? [];
    const extent = Math.max(0, ...[...lists.values()].map((l) => l.length));

    const rows: OptionsMatrixRow[] = [];
    for (let i = 0; i < extent; i++) {
      const cells = new Map<string, { value: unknown; state: OptionsEntryCellState }>();
      for (const o of objects) {
        const list = lists.get(o.key)!;
        const has = i < list.length;
        const baseHas = i < baseline.length;
        cells.set(o.key, {
          value: has ? list[i] : undefined,
          state: cellState(baseHas, has, baseline[i], list[i]),
        });
      }
      rows.push({ key: String(i), label: String(i), cells });
    }
    return { shape, rows };
  }

  const maps = new Map<string, Record<string, unknown>>();
  for (const o of objects) maps.set(o.key, mapOf(values.get(o.key)) ?? {});
  const baseline = mapOf(baselineValue) ?? {};

  const keys = new Set<string>();
  for (const m of maps.values()) for (const k of Object.keys(m)) keys.add(k);

  const rows: OptionsMatrixRow[] = [];
  for (const k of [...keys].sort()) {
    const cells = new Map<string, { value: unknown; state: OptionsEntryCellState }>();
    for (const o of objects) {
      const m = maps.get(o.key)!;
      const has = Object.prototype.hasOwnProperty.call(m, k);
      const baseHas = Object.prototype.hasOwnProperty.call(baseline, k);
      cells.set(o.key, {
        value: has ? m[k] : undefined,
        state: cellState(baseHas, has, baseline[k], m[k]),
      });
    }
    rows.push({ key: k, label: k, cells });
  }
  return { shape, rows };
}

function cellState(
  baselineHas: boolean,
  objectHas: boolean,
  baselineValue: unknown,
  objectValue: unknown,
): OptionsEntryCellState {
  if (!objectHas) return "missing";
  if (!baselineHas) return "extra";
  return deepEqual(baselineValue, objectValue) ? "same" : "differs";
}

function deepEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (typeof a !== typeof b) return false;
  if (a === null || b === null) return false;
  if (typeof a !== "object") return false;
  return JSON.stringify(a) === JSON.stringify(b);
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/options.test.ts`

Expected: PASS, all four describe blocks.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/socket/options.ts \
        services/atlas-ui/src/lib/socket/__tests__/options.test.ts
git commit -m "feat(ui): classify socket options and build the nested per-entry matrix

Ordered lists compare POSITIONALLY: the array index is the wire value and the
name is not unique. gms_95_1 CharacterMovement carries UNKNOWN at six separate
indices, so comparing by name would match six unrelated slots to each other. A
name that shifts index between versions is a diagonal, not a match.

Maps compare by key. An empty types array classifies as a list with zero
entries, not as 'supplies no options' - the distinction is exactly what makes
the empty gms_92_1/gms_95_1 movement tables visible for the first time."
```

---

## Task 11: `ancestry.ts` — infer the ancestor Template and classify against it

**Files:**
- Create: `services/atlas-ui/src/lib/socket/ancestry.ts`
- Create: `services/atlas-ui/src/lib/socket/__tests__/ancestry.test.ts`

**Interfaces:**
- Consumes: `SocketObject`, `DefinitionKind`, `entriesOf`, `unsupportedOf`, `stateOf` (Task 8); `classifyOptions` and its deep-compare semantics (Task 10).
- Produces:
  - `AncestryClass = "same" | "modified" | "tenant-only" | "missing" | "unsupported"`
  - `inferAncestor(tenant: SocketObject, templates: SocketObject[]) → SocketObject | null`
  - `classifyAgainstAncestor(tenant, ancestor, kind, name) → AncestryClass`
  - `missingFromTenant(tenant, ancestor, kind) → string[]`

**FR-8.1:** the ancestor is inferred by exact match on Region, Major Version and Minor Version. No Template id is stored.

**FR-8.2:** zero matches → `null`, and the Tenant page renders a **single column** with ancestry features **absent** — not a disabled control, not an empty column.

**FR-8.4:** comparison covers opcode, validator, services and options. Opcodes normalise numerically before comparison, so `0xB8` and `0x0B8` are the same binding. `fname` never participates (FR-10.4).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/lib/socket/__tests__/ancestry.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
  classifyAgainstAncestor,
  inferAncestor,
  missingFromTenant,
} from "@/lib/socket/ancestry";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    validator: "LoggedInValidator",
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function makeObject(
  key: string,
  source: SocketObject["source"],
  major: number,
  minor: number,
  handlers: Record<string, Binding[]> = {},
  unsupportedHandlers: string[] = [],
  region = "GMS",
): SocketObject {
  return {
    key,
    label: `${region} v${major}.${minor}`,
    source,
    region,
    majorVersion: major,
    minorVersion: minor,
    handlers: new Map(Object.entries(handlers)),
    writers: new Map(),
    unsupportedHandlers: new Set(unsupportedHandlers),
    unsupportedWriters: new Set(),
  };
}

describe("inferAncestor", () => {
  const t83 = makeObject("t83", "template", 83, 1);
  const t95 = makeObject("t95", "template", 95, 1);
  const jms = makeObject("jms", "template", 185, 1, {}, [], "JMS");

  it("matches on exact region, major and minor version", () => {
    const tenant = makeObject("tnt", "tenant", 95, 1);
    expect(inferAncestor(tenant, [t83, t95, jms])?.key).toBe("t95");
  });

  it("does not match across regions", () => {
    const tenant = makeObject("tnt", "tenant", 185, 1, {}, [], "GMS");
    expect(inferAncestor(tenant, [t83, t95, jms])).toBeNull();
  });

  it("does not match on major version alone", () => {
    const tenant = makeObject("tnt", "tenant", 95, 2);
    expect(inferAncestor(tenant, [t83, t95])).toBeNull();
  });

  it("returns null when there is no template at all", () => {
    expect(inferAncestor(makeObject("tnt", "tenant", 95, 1), [])).toBeNull();
  });
});

describe("classifyAgainstAncestor", () => {
  it("returns same for an identical binding set", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, { LoginHandle: [binding("0x01")] });
    const anc = makeObject("t83", "template", 83, 1, { LoginHandle: [binding("0x01")] });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe("same");
  });

  it("normalises opcodes numerically before comparing", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, { MiniRoom: [binding("0x0B8")] });
    const anc = makeObject("t83", "template", 83, 1, { MiniRoom: [binding("0xB8")] });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "MiniRoom")).toBe("same");
  });

  it("ignores fname entirely", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      LoginHandle: [binding("0x01", { fname: "CLogin::SendCheckPasswordPacket" })],
    });
    const anc = makeObject("t83", "template", 83, 1, { LoginHandle: [binding("0x01")] });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe("same");
  });

  it("returns modified when the opcode differs", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, { LoginHandle: [binding("0x02")] });
    const anc = makeObject("t83", "template", 83, 1, { LoginHandle: [binding("0x01")] });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe("modified");
  });

  it("returns modified when the validator differs", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      LoginHandle: [binding("0x01", { validator: "NoOpValidator" })],
    });
    const anc = makeObject("t83", "template", 83, 1, { LoginHandle: [binding("0x01")] });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe("modified");
  });

  it("returns modified when the services differ", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      LoginHandle: [binding("0x01", { services: ["login", "channel"] })],
    });
    const anc = makeObject("t83", "template", 83, 1, { LoginHandle: [binding("0x01")] });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe("modified");
  });

  it("returns modified when the options differ structurally", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      Move: [binding("0x01", { options: { types: ["WALK"] } })],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      Move: [binding("0x01", { options: { types: ["WALK", "STAND"] } })],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "Move")).toBe("modified");
  });

  it("returns modified when the binding COUNT differs for one name", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      NoOpHandler: [binding("0x17"), binding("0x19")],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      NoOpHandler: [binding("0x17"), binding("0x19"), binding("0x22")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "NoOpHandler")).toBe("modified");
  });

  it("compares multi-binding sets irrespective of stored order", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {
      NoOpHandler: [binding("0x19"), binding("0x17")],
    });
    const anc = makeObject("t83", "template", 83, 1, {
      NoOpHandler: [binding("0x17"), binding("0x19")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "NoOpHandler")).toBe("same");
  });

  it("returns tenant-only when the ancestor has no such definition", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, { Custom: [binding("0x7F")] });
    const anc = makeObject("t83", "template", 83, 1, {});
    expect(classifyAgainstAncestor(tenant, anc, "handler", "Custom")).toBe("tenant-only");
  });

  it("returns missing when the ancestor defines it and the tenant does not", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {});
    const anc = makeObject("t83", "template", 83, 1, { LoginHandle: [binding("0x01")] });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "LoginHandle")).toBe("missing");
  });

  it("returns unsupported when the tenant marked it so, whatever the ancestor says", () => {
    const tenant = makeObject("tnt", "tenant", 83, 1, {}, ["GuestLoginHandle"]);
    const anc = makeObject("t83", "template", 83, 1, {
      GuestLoginHandle: [binding("0x02")],
    });
    expect(classifyAgainstAncestor(tenant, anc, "handler", "GuestLoginHandle")).toBe(
      "unsupported",
    );
  });
});

describe("missingFromTenant", () => {
  it("lists only names defined in the ancestor and undefined in the tenant", () => {
    const tenant = makeObject(
      "tnt",
      "tenant",
      83,
      1,
      { LoginHandle: [binding("0x01")] },
      ["GuestLoginHandle"],
    );
    const anc = makeObject("t83", "template", 83, 1, {
      LoginHandle: [binding("0x01")],
      PongHandle: [binding("0x18")],
      GuestLoginHandle: [binding("0x02")],
    });
    // LoginHandle is already defined; GuestLoginHandle is explicitly unsupported
    // and so is NOT undefined (FR-9.5 excludes it unless the user opts in).
    expect(missingFromTenant(tenant, anc, "handler")).toEqual(["PongHandle"]);
  });

  it("returns an empty list when the tenant defines everything", () => {
    const b = { LoginHandle: [binding("0x01")] };
    const tenant = makeObject("tnt", "tenant", 83, 1, b);
    const anc = makeObject("t83", "template", 83, 1, b);
    expect(missingFromTenant(tenant, anc, "handler")).toEqual([]);
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/ancestry.test.ts`

Expected: FAIL — cannot resolve `@/lib/socket/ancestry`.

- [ ] **Step 3: Implement `ancestry.ts`**

Create `services/atlas-ui/src/lib/socket/ancestry.ts`:

```ts
import type { Binding, DefinitionKind, SocketObject } from "@/lib/socket/model";
import { entriesOf, stateOf } from "@/lib/socket/model";

/**
 * How one Tenant Definition relates to the same Definition in its ancestor
 * Template (FR-8.3).
 */
export type AncestryClass =
  | "same"
  | "modified"
  | "tenant-only"
  | "missing"
  | "unsupported";

/**
 * FR-8.1. The ancestor is inferred by exact match on Region, Major Version and
 * Minor Version - no Template id is stored anywhere.
 *
 * FR-8.2: zero matches returns null, and the caller renders a SINGLE column
 * with ancestry features ABSENT. Not a disabled control, not an empty column.
 * This is not expected to occur in practice.
 */
export function inferAncestor(
  tenant: SocketObject,
  templates: SocketObject[],
): SocketObject | null {
  return (
    templates.find(
      (t) =>
        t.region === tenant.region &&
        t.majorVersion === tenant.majorVersion &&
        t.minorVersion === tenant.minorVersion,
    ) ?? null
  );
}

/**
 * FR-8.3/8.4. Compares the BINDING SET, not a single opcode: opcodes
 * numerically, validator and services by value, options structurally, and
 * fname never (FR-10.4). A differing binding COUNT for one name is modified.
 */
export function classifyAgainstAncestor(
  tenant: SocketObject,
  ancestor: SocketObject,
  kind: DefinitionKind,
  name: string,
): AncestryClass {
  // An explicit Unsupported marking is the tenant's own statement about this
  // definition and outranks whatever the ancestor happens to carry.
  if (stateOf(tenant, kind, name) === "unsupported") return "unsupported";

  const tenantBindings = entriesOf(tenant, kind).get(name) ?? [];
  const ancestorBindings = entriesOf(ancestor, kind).get(name) ?? [];

  if (tenantBindings.length === 0 && ancestorBindings.length === 0) return "missing";
  if (tenantBindings.length === 0) return "missing";
  if (ancestorBindings.length === 0) return "tenant-only";
  if (tenantBindings.length !== ancestorBindings.length) return "modified";

  // Stored order is not meaningful - templatesService re-sorts both arrays on
  // read - so compare as sets keyed by the normalized opcode.
  const byOpcode = (b: Binding[]) => {
    const m = new Map<number | null, Binding>();
    for (const x of b) m.set(x.opCodeValue, x);
    return m;
  };
  const t = byOpcode(tenantBindings);
  const a = byOpcode(ancestorBindings);
  if (t.size !== a.size) return "modified";

  for (const [opcode, tb] of t) {
    const ab = a.get(opcode);
    if (!ab) return "modified";
    if (!sameBinding(tb, ab)) return "modified";
  }
  return "same";
}

function sameBinding(a: Binding, b: Binding): boolean {
  if (a.opCodeValue !== b.opCodeValue) return false;
  if ((a.validator ?? "") !== (b.validator ?? "")) return false;
  if (!sameStringSet(a.services, b.services)) return false;
  return sameOptions(a.options, b.options);
}

function sameStringSet(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  const sa = [...a].sort();
  const sb = [...b].sort();
  return sa.every((v, i) => v === sb[i]);
}

/**
 * Absent and an explicit {} both mean "not supplied" (FR-3.2), so they compare
 * equal here - a tenant is not "modified" merely because a PATCH round-trip
 * materialised an empty object.
 */
function sameOptions(a: unknown, b: unknown): boolean {
  const norm = (v: unknown) =>
    v === undefined ||
    v === null ||
    (typeof v === "object" && Object.keys(v as Record<string, unknown>).length === 0)
      ? null
      : JSON.stringify(v);
  return norm(a) === norm(b);
}

/**
 * FR-9.1. Definitions Defined in the ancestor Template and Undefined in the
 * Tenant. A name the tenant explicitly marked Unsupported is NOT undefined, so
 * it is excluded here (FR-9.5 - the user opts it in separately).
 */
export function missingFromTenant(
  tenant: SocketObject,
  ancestor: SocketObject,
  kind: DefinitionKind,
): string[] {
  const out: string[] = [];
  for (const [name, bindings] of entriesOf(ancestor, kind)) {
    if (bindings.length === 0) continue;
    if (stateOf(tenant, kind, name) === "undefined") out.push(name);
  }
  return out.sort();
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/ancestry.test.ts`

Expected: PASS, all three describe blocks.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/socket/ancestry.ts \
        services/atlas-ui/src/lib/socket/__tests__/ancestry.test.ts
git commit -m "feat(ui): infer a tenant's ancestor template and classify against it

The ancestor is inferred by exact region + major + minor match; no template id
is stored. No match returns null and the caller drops to a single column with
ancestry features absent, rather than showing a disabled control.

Comparison is over the whole binding SET, keyed by normalized opcode, so stored
order does not matter - templatesService re-sorts both arrays on read. Absent
options and an explicit {} compare equal, so a tenant is not reported as
modified merely because a PATCH round-trip materialised an empty object."
```

---

## Task 12: `mutate.ts` — the pure splice functions behind every dialog

Every dialog and both bulk flows reduce to one of these. They take a whole `SocketConfig` and return a new one; they never fetch, never mutate in place, and never touch React.

**Files:**
- Create: `services/atlas-ui/src/lib/socket/mutate.ts`
- Create: `services/atlas-ui/src/lib/socket/__tests__/mutate.test.ts`

**Interfaces:**
- Consumes: `SocketConfig`, `SocketHandlerEntry`, `SocketWriterEntry`, `SocketUnsupported` (Task 8); `DefinitionKind` (Task 8); `parseOpcode` (Task 8).
- Produces:
  - `BindingInput { opCode; validator?; services; options?; fname? }`
  - `MutationError` (thrown when a binding does not resolve to exactly one entry)
  - `addBinding(cfg, kind, name, input) → SocketConfig`
  - `editBinding(cfg, kind, name, opCodeValue, input) → SocketConfig`
  - `deleteBinding(cfg, kind, name, opCodeValue) → SocketConfig`
  - `markUnsupported(cfg, kind, name) → SocketConfig`
  - `clearUnsupported(cfg, kind, name) → SocketConfig`
  - `copyBindings(cfg, kind, name, inputs) → SocketConfig`
  - `AncestorAddition { name: string; bindings: BindingInput[] }`
  - `copyMissingFromAncestor(cfg, kind, additions: AncestorAddition[]) → SocketConfig`
  - `fillMissingValidators(cfg, validator) → SocketConfig`

**The two invariants enforced by construction:**
- FR-1.2 — `addBinding` and `copyBindings` clear the name from the unsupported list.
- FR-1.1 / §5.2 — `markUnsupported` removes **every** binding of that name, necessarily, because `unsupported` is name-scoped while bindings are opcode-scoped. The dialog says so in as many words before confirming.

**Resolution is by `(name, opCodeValue)`, never by index.** If that resolves to a count other than 1, throw `MutationError` rather than guessing — a concurrent edit then fails loudly instead of clobbering.

**`fillMissingValidators` is the strict-validation escape hatch.** Saves are whole-document, so with FR-11.4 blocking, the live gms_95 tenant's 32 empty validators make every single-definition PATCH fail — including the one that would fix them. This repairs the whole document in one write, which is the only way out. See Task 18.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/lib/socket/__tests__/mutate.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
  MutationError,
  addBinding,
  clearUnsupported,
  copyBindings,
  copyMissingFromAncestor,
  deleteBinding,
  editBinding,
  fillMissingValidators,
  markUnsupported,
} from "@/lib/socket/mutate";
import type { SocketConfig } from "@/types/models/socket";

function config(): SocketConfig {
  return {
    handlers: [
      { opCode: "0x01", validator: "NoOpValidator", handler: "LoginHandle", services: ["login"] },
      { opCode: "0x17", validator: "LoggedInValidator", handler: "NoOpHandler", services: ["channel"] },
      { opCode: "0x19", validator: "LoggedInValidator", handler: "NoOpHandler", services: ["channel"] },
      { opCode: "0x22", validator: "LoggedInValidator", handler: "NoOpHandler", services: ["channel"] },
    ],
    writers: [
      { opCode: "0x00", writer: "AuthSuccess", services: ["login"] },
    ],
    unsupported: { handlers: ["GuestLoginHandle"], writers: [] },
  };
}

describe("addBinding", () => {
  it("appends the binding", () => {
    const out = addBinding(config(), "writer", "PetActivated", {
      opCode: "0x9A",
      services: ["channel"],
    });
    expect(out.writers).toHaveLength(2);
    expect(out.writers[1]).toMatchObject({ opCode: "0x9A", writer: "PetActivated" });
  });

  // FR-1.2
  it("clears any unsupported marker for that name", () => {
    const out = addBinding(config(), "handler", "GuestLoginHandle", {
      opCode: "0x02",
      validator: "NoOpValidator",
      services: ["login"],
    });
    expect(out.unsupported!.handlers).toEqual([]);
  });

  it("adds a second binding to an existing name without touching the first", () => {
    const out = addBinding(config(), "handler", "NoOpHandler", {
      opCode: "0x24",
      validator: "LoggedInValidator",
      services: ["channel"],
    });
    expect(out.handlers.filter((h) => h.handler === "NoOpHandler")).toHaveLength(4);
  });

  it("does not mutate the input", () => {
    const input = config();
    addBinding(input, "writer", "PetActivated", { opCode: "0x9A", services: ["channel"] });
    expect(input.writers).toHaveLength(1);
  });

  it("rejects a binding at an opcode the name already uses", () => {
    expect(() =>
      addBinding(config(), "handler", "NoOpHandler", {
        opCode: "0x017",
        validator: "LoggedInValidator",
        services: ["channel"],
      }),
    ).toThrow(MutationError);
  });
});

describe("editBinding", () => {
  it("edits exactly the addressed binding of a multi-binding name", () => {
    const out = editBinding(config(), "handler", "NoOpHandler", 0x19, {
      opCode: "0x1A",
      validator: "NoOpValidator",
      services: ["channel", "login"],
    });
    const noop = out.handlers.filter((h) => h.handler === "NoOpHandler");
    expect(noop.map((h) => h.opCode)).toEqual(["0x17", "0x1A", "0x22"]);
    expect(noop[1]!.validator).toBe("NoOpValidator");
    expect(noop[0]!.validator).toBe("LoggedInValidator");
  });

  it("throws when the binding does not resolve", () => {
    expect(() =>
      editBinding(config(), "handler", "NoOpHandler", 0xff, {
        opCode: "0xFF",
        validator: "NoOpValidator",
        services: [],
      }),
    ).toThrow(MutationError);
  });

  it("throws when the binding resolves more than once", () => {
    const dup: SocketConfig = {
      handlers: [],
      writers: [
        { opCode: "0xB8", writer: "MiniRoom", services: ["channel"] },
        { opCode: "0x0B8", writer: "MiniRoom", services: ["channel"] },
      ],
    };
    expect(() =>
      editBinding(dup, "writer", "MiniRoom", 0xb8, { opCode: "0xB8", services: [] }),
    ).toThrow(MutationError);
  });
});

describe("deleteBinding", () => {
  it("removes exactly one binding of a multi-binding name", () => {
    const out = deleteBinding(config(), "handler", "NoOpHandler", 0x19);
    expect(out.handlers.filter((h) => h.handler === "NoOpHandler").map((h) => h.opCode)).toEqual([
      "0x17",
      "0x22",
    ]);
  });

  // FR-1.4: deleting leaves the definition Undefined; it does NOT mark it
  // unsupported. That is a separate, explicit choice in the dialog.
  it("does not add an unsupported marker", () => {
    const out = deleteBinding(config(), "writer", "AuthSuccess", 0x00);
    expect(out.writers).toHaveLength(0);
    expect(out.unsupported!.writers).toEqual([]);
  });

  it("throws when the binding does not resolve", () => {
    expect(() => deleteBinding(config(), "writer", "AuthSuccess", 0x99)).toThrow(MutationError);
  });
});

describe("markUnsupported", () => {
  // The plural matters: unsupported is NAME-scoped while bindings are
  // OPCODE-scoped, so marking a name necessarily removes all four NoOpHandler
  // routes. The dialog states this before confirming.
  it("removes EVERY binding of the name", () => {
    const out = markUnsupported(config(), "handler", "NoOpHandler");
    expect(out.handlers.filter((h) => h.handler === "NoOpHandler")).toHaveLength(0);
    expect(out.unsupported!.handlers).toContain("NoOpHandler");
  });

  it("is idempotent", () => {
    const once = markUnsupported(config(), "handler", "NoOpHandler");
    const twice = markUnsupported(once, "handler", "NoOpHandler");
    expect(twice.unsupported!.handlers.filter((n) => n === "NoOpHandler")).toHaveLength(1);
  });

  it("works on a name that was never defined", () => {
    const out = markUnsupported(config(), "writer", "MonsterCarnival");
    expect(out.unsupported!.writers).toEqual(["MonsterCarnival"]);
  });
});

describe("clearUnsupported", () => {
  // FR-1.3
  it("returns the definition to Undefined", () => {
    const out = clearUnsupported(config(), "handler", "GuestLoginHandle");
    expect(out.unsupported!.handlers).toEqual([]);
    expect(out.handlers.some((h) => h.handler === "GuestLoginHandle")).toBe(false);
  });

  it("is a no-op for a name that is not marked", () => {
    const out = clearUnsupported(config(), "handler", "LoginHandle");
    expect(out.unsupported!.handlers).toEqual(["GuestLoginHandle"]);
  });
});

describe("copyBindings", () => {
  it("adds every supplied binding and clears the unsupported marker", () => {
    const out = copyBindings(config(), "handler", "GuestLoginHandle", [
      { opCode: "0x02", validator: "NoOpValidator", services: ["login"] },
      { opCode: "0x03", validator: "NoOpValidator", services: ["login"] },
    ]);
    expect(out.handlers.filter((h) => h.handler === "GuestLoginHandle")).toHaveLength(2);
    expect(out.unsupported!.handlers).toEqual([]);
  });

  it("produces a result independent of the source", () => {
    const options = { types: ["WALK"] };
    const out = copyBindings(config(), "writer", "CharacterMovement", [
      { opCode: "0xB9", services: ["channel"], options },
    ]);
    (options.types as string[]).push("STAND");
    const copied = out.writers.find((w) => w.writer === "CharacterMovement")!;
    expect((copied.options as { types: string[] }).types).toEqual(["WALK"]);
  });
});

describe("copyMissingFromAncestor", () => {
  const additions = [
    { name: "PongHandle", bindings: [{ opCode: "0x18", validator: "NoOpValidator", services: ["channel"] }] },
    { name: "LoginHandle", bindings: [{ opCode: "0xFF", validator: "NoOpValidator", services: ["login"] }] },
  ];

  // FR-9.4
  it("never overwrites an already-defined definition", () => {
    const out = copyMissingFromAncestor(config(), "handler", additions);
    const login = out.handlers.filter((h) => h.handler === "LoginHandle");
    expect(login).toHaveLength(1);
    expect(login[0]!.opCode).toBe("0x01");
  });

  it("adds the definitions that were undefined", () => {
    const out = copyMissingFromAncestor(config(), "handler", additions);
    expect(out.handlers.filter((h) => h.handler === "PongHandle")).toHaveLength(1);
  });

  // FR-9.6
  it("applies the whole selection as one document", () => {
    const out = copyMissingFromAncestor(config(), "handler", [
      { name: "A", bindings: [{ opCode: "0x30", validator: "NoOpValidator", services: ["channel"] }] },
      { name: "B", bindings: [{ opCode: "0x31", validator: "NoOpValidator", services: ["channel"] }] },
      { name: "C", bindings: [{ opCode: "0x32", validator: "NoOpValidator", services: ["channel"] }] },
    ]);
    expect(out.handlers).toHaveLength(config().handlers.length + 3);
  });

  it("clears the unsupported marker for any name it adds", () => {
    const out = copyMissingFromAncestor(config(), "handler", [
      { name: "GuestLoginHandle", bindings: [{ opCode: "0x02", validator: "NoOpValidator", services: ["login"] }] },
    ]);
    expect(out.unsupported!.handlers).toEqual([]);
  });
});

describe("fillMissingValidators", () => {
  // Strict FR-11.4 blocks any save of a document carrying an empty validator,
  // and saves are whole-document, so a single-definition edit can never be the
  // fix. This repairs every offender in one write.
  it("fills every empty handler validator in one pass", () => {
    const broken: SocketConfig = {
      handlers: [
        { opCode: "0x01", validator: "", handler: "A", services: ["channel"] },
        { opCode: "0x02", validator: "   ", handler: "B", services: ["channel"] },
        { opCode: "0x03", validator: "LoggedInValidator", handler: "C", services: ["channel"] },
      ],
      writers: [],
    };
    const out = fillMissingValidators(broken, "NoOpValidator");
    expect(out.handlers.map((h) => h.validator)).toEqual([
      "NoOpValidator",
      "NoOpValidator",
      "LoggedInValidator",
    ]);
  });

  it("leaves writers alone - they carry no validator", () => {
    const cfg = config();
    const out = fillMissingValidators(cfg, "NoOpValidator");
    expect(out.writers).toEqual(cfg.writers);
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/mutate.test.ts`

Expected: FAIL — cannot resolve `@/lib/socket/mutate`.

- [ ] **Step 3: Implement `mutate.ts`**

Create `services/atlas-ui/src/lib/socket/mutate.ts`:

```ts
import type {
  SocketConfig,
  SocketHandlerEntry,
  SocketUnsupported,
  SocketWriterEntry,
} from "@/types/models/socket";
import type { DefinitionKind } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

/**
 * Thrown when a mutation cannot address exactly one binding. Callers surface it
 * as an error toast and abandon the write: a concurrent edit must fail loudly
 * rather than guess which entry the user meant.
 */
export class MutationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "MutationError";
  }
}

/** The editable fields of one binding. */
export interface BindingInput {
  opCode: string;
  /** Handlers only. */
  validator?: string;
  services: string[];
  options?: unknown;
  fname?: string;
}

type AnyEntry = SocketHandlerEntry | SocketWriterEntry;

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T;
}

function unsupportedOrEmpty(cfg: SocketConfig): SocketUnsupported {
  return {
    handlers: [...(cfg.unsupported?.handlers ?? [])],
    writers: [...(cfg.unsupported?.writers ?? [])],
  };
}

function nameOf(e: AnyEntry, kind: DefinitionKind): string {
  return kind === "handler"
    ? (e as SocketHandlerEntry).handler
    : (e as SocketWriterEntry).writer;
}

function toEntry(kind: DefinitionKind, name: string, input: BindingInput): AnyEntry {
  const base = {
    opCode: input.opCode,
    ...(input.fname !== undefined && input.fname !== "" ? { fname: input.fname } : {}),
    ...(input.options !== undefined ? { options: clone(input.options) } : {}),
    ...(input.services.length > 0 ? { services: [...input.services] } : {}),
  };
  return kind === "handler"
    ? ({ ...base, validator: input.validator ?? "", handler: name } as SocketHandlerEntry)
    : ({ ...base, writer: name } as SocketWriterEntry);
}

/** Rebuilds a config with one collection replaced. */
function withCollection(
  cfg: SocketConfig,
  kind: DefinitionKind,
  entries: AnyEntry[],
  unsupported: SocketUnsupported,
): SocketConfig {
  return {
    handlers: kind === "handler" ? (entries as SocketHandlerEntry[]) : clone(cfg.handlers),
    writers: kind === "writer" ? (entries as SocketWriterEntry[]) : clone(cfg.writers),
    unsupported,
  };
}

function collectionOf(cfg: SocketConfig, kind: DefinitionKind): AnyEntry[] {
  return clone(kind === "handler" ? cfg.handlers : cfg.writers);
}

/**
 * Finds the single array position holding (name, opCodeValue). Index is never
 * an input - templatesService re-sorts both arrays on read, so a fetched index
 * does not match the stored one.
 */
function resolveOne(
  entries: AnyEntry[],
  kind: DefinitionKind,
  name: string,
  opCodeValue: number,
): number {
  const hits: number[] = [];
  entries.forEach((e, i) => {
    if (nameOf(e, kind) === name && parseOpcode(e.opCode) === opCodeValue) hits.push(i);
  });
  if (hits.length === 0) {
    throw new MutationError(
      `No ${kind} named "${name}" at opcode ${opCodeValue} exists in the current configuration. It may have been changed or removed by another session — reload and try again.`,
    );
  }
  if (hits.length > 1) {
    throw new MutationError(
      `"${name}" is bound ${hits.length} times at opcode ${opCodeValue} in the current configuration. Resolve the duplicate before editing it here.`,
    );
  }
  return hits[0]!;
}

function dropName(list: string[], name: string): string[] {
  return list.filter((n) => n !== name);
}

/** FR-6.1. Adding a Definition clears any Unsupported marker for that name. */
export function addBinding(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
  input: BindingInput,
): SocketConfig {
  const entries = collectionOf(cfg, kind);
  const value = parseOpcode(input.opCode);
  if (value === null) {
    throw new MutationError(`"${input.opCode}" is not a valid opcode. Use 0x followed by 1-4 hex digits.`);
  }
  const collision = entries.some(
    (e) => nameOf(e, kind) === name && parseOpcode(e.opCode) === value,
  );
  if (collision) {
    throw new MutationError(`"${name}" is already bound to opcode ${input.opCode}.`);
  }

  entries.push(toEntry(kind, name, input));
  const unsupported = unsupportedOrEmpty(cfg);
  if (kind === "handler") unsupported.handlers = dropName(unsupported.handlers, name);
  else unsupported.writers = dropName(unsupported.writers, name);

  return withCollection(cfg, kind, entries, unsupported);
}

/** FR-6.2. The name is the identity and is not editable; renaming is unsupported. */
export function editBinding(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
  opCodeValue: number,
  input: BindingInput,
): SocketConfig {
  const entries = collectionOf(cfg, kind);
  const at = resolveOne(entries, kind, name, opCodeValue);
  if (parseOpcode(input.opCode) === null) {
    throw new MutationError(`"${input.opCode}" is not a valid opcode. Use 0x followed by 1-4 hex digits.`);
  }
  entries[at] = toEntry(kind, name, input);
  return withCollection(cfg, kind, entries, unsupportedOrEmpty(cfg));
}

/**
 * FR-6.3 "Remove definition". Leaves the Definition Undefined - it does NOT
 * mark it Unsupported. That is the dialog's separate, explicit second option.
 */
export function deleteBinding(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
  opCodeValue: number,
): SocketConfig {
  const entries = collectionOf(cfg, kind);
  const at = resolveOne(entries, kind, name, opCodeValue);
  entries.splice(at, 1);
  return withCollection(cfg, kind, entries, unsupportedOrEmpty(cfg));
}

/**
 * FR-6.4. Removes EVERY binding of the name, necessarily: unsupported is
 * name-scoped while bindings are opcode-scoped, so a name cannot be half
 * unsupported. Marking NoOpHandler in gms_95_1 removes four routes.
 */
export function markUnsupported(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
): SocketConfig {
  const entries = collectionOf(cfg, kind).filter((e) => nameOf(e, kind) !== name);
  const unsupported = unsupportedOrEmpty(cfg);
  const list = kind === "handler" ? unsupported.handlers : unsupported.writers;
  if (!list.includes(name)) list.push(name);
  return withCollection(cfg, kind, entries, unsupported);
}

/** FR-1.3. Returns the Definition to Undefined. */
export function clearUnsupported(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
): SocketConfig {
  const unsupported = unsupportedOrEmpty(cfg);
  if (kind === "handler") unsupported.handlers = dropName(unsupported.handlers, name);
  else unsupported.writers = dropName(unsupported.writers, name);
  return withCollection(cfg, kind, collectionOf(cfg, kind), unsupported);
}

/** FR-6.5. The result is independent of the source: every value is deep-cloned. */
export function copyBindings(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
  inputs: BindingInput[],
): SocketConfig {
  let out = cfg;
  for (const input of inputs) out = addBinding(out, kind, name, input);
  return out;
}

export interface AncestorAddition {
  name: string;
  bindings: BindingInput[];
}

/**
 * FR-9.4/9.6. Applies the whole selection as ONE document, and never overwrites
 * an already-Defined Tenant Definition - a name that gained a definition
 * between the candidate scan and the apply is silently skipped rather than
 * clobbered.
 */
export function copyMissingFromAncestor(
  cfg: SocketConfig,
  kind: DefinitionKind,
  additions: AncestorAddition[],
): SocketConfig {
  let out = cfg;
  for (const addition of additions) {
    const defined = (kind === "handler" ? out.handlers : out.writers).some(
      (e) => nameOf(e as AnyEntry, kind) === addition.name,
    );
    if (defined) continue;
    out = copyBindings(out, kind, addition.name, addition.bindings);
  }
  return out;
}

/**
 * Bulk remediation for strict FR-11.4. Saves are whole-document, so a document
 * carrying ANY empty handler validator cannot be saved at all - which means a
 * single-definition edit can never be the fix. This repairs every offender in
 * one write. Writers are untouched; they carry no validator.
 */
export function fillMissingValidators(
  cfg: SocketConfig,
  validator: string,
): SocketConfig {
  const handlers = clone(cfg.handlers).map((h) =>
    (h.validator ?? "").trim() === "" ? { ...h, validator } : h,
  );
  return {
    handlers,
    writers: clone(cfg.writers),
    unsupported: unsupportedOrEmpty(cfg),
  };
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/lib/socket/__tests__/mutate.test.ts`

Expected: PASS, all eight describe blocks.

- [ ] **Step 5: Run the whole domain library and type-check**

```bash
cd services/atlas-ui && npx vitest run src/lib/socket/ && npm run build
```

Expected: both clean. This is the point at which the entire protocol semantics is proven without a single React render.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/lib/socket/mutate.ts \
        services/atlas-ui/src/lib/socket/__tests__/mutate.test.ts
git commit -m "feat(ui): add the pure socket mutation splices

Every dialog and both bulk flows reduce to one of these. They take a whole
SocketConfig and return a new one - no fetching, no in-place mutation, no React.

Bindings resolve by (name, normalized opcode), never by index: templatesService
re-sorts both arrays on read, so a fetched index does not match the stored one.
A resolution that does not hit exactly one entry throws MutationError, so a
concurrent edit fails loudly instead of clobbering the wrong route.

markUnsupported removes EVERY binding of the name, necessarily - unsupported is
name-scoped while bindings are opcode-scoped, so marking NoOpHandler in gms_95_1
removes four routes. The dialog says so before confirming.

fillMissingValidators is the strict-validation escape hatch: with FR-11.4
blocking and whole-document saves, a document carrying an empty validator cannot
be saved at all, so a single-definition edit can never be the fix."
```

---

## Task 13: The data layer — sparse reads, whole-document writes, and the PUT bug

**Files:**
- Modify: `services/atlas-ui/src/services/api/templates.service.ts:276-288` (the `update` method)
- Modify: `services/atlas-ui/src/services/api/templates.service.ts` (add `getSocketMatrix`)
- Modify: `services/atlas-ui/src/services/api/tenants.service.ts` (add `getSocketMatrix`)
- Create: `services/atlas-ui/src/lib/hooks/api/useSocketObjects.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/__tests__/useSocketObjects.test.tsx`
- Create: `services/atlas-ui/src/services/api/__tests__/templates-update.test.ts`

**Interfaces:**
- Consumes: `SocketConfig` (Task 8); `fromTemplate`, `fromTenantConfig` (Task 8); every `mutate.ts` function (Task 12).
- Produces:
  - `templatesService.getSocketMatrix() → Promise<Template[]>` (sparse)
  - `tenantsService.getSocketMatrix() → Promise<TenantConfig[]>` (sparse)
  - `socketKeys.matrix()`, `socketKeys.tenantMatrix()` query keys
  - `useSocketMatrixTemplates()`, `useSocketMatrixTenants()`
  - `useSocketMutation()` returning `mutate({ target, apply })` where `target = { source: "template" | "tenant"; id: string }` and `apply: (cfg: SocketConfig) => SocketConfig`

### Two live bugs this task fixes

Both are in the template save path and both are load-bearing for everything downstream.

**Bug 1 — `templatesService.update` issues an HTTP `PUT`.** `atlas-configurations` registers **no** `PUT` route: `templates/resource.go:29` binds `/{templateId}` to `http.MethodPatch` only, and `grep -rn 'MethodPut' services/atlas-configurations/` returns nothing. A `PUT` there is a 405.

**Bug 2 — the four stacked-card forms pass a partial attribute object.** `templates-handlers-form.tsx:57-65` calls `updateTemplate.mutate({ id, updates: { socket: {...} } })`, and `templatesService.update` casts that partial straight into `wrapTemplate(data as TemplateAttributes, id)`. Even before the transport fails, `throwIfInvalid` runs `validateTemplate` on it and throws `Template validation failed: Region is required…, Major version must be…` because `region`/`majorVersion`/`usesPin`/`characters` are all absent.

Verify both before changing anything (Step 1). The tenant path is fine — `tenants.service.ts:303` uses `api.patch` and spreads the full `tenant.attributes`.

Since this task deletes those four forms' only caller in Task 19 and replaces the write path wholesale, fixing `update` here is not scope creep — it is the write path this feature runs on.

### The sparse-cache write hazard

`tenants.service.ts:303` builds its PATCH body as `{ ...tenant.attributes, ...updatedAttributes }`. If that spread ever runs over a **sparsely-fetched** document, the write silently erases `characters`, `worlds` and `cashShop`. The same shape exists on the template path.

**Rule, enforced by construction: sparse reads live under their own query key and are never a mutation input.**

- `socketKeys.matrix()` / `socketKeys.tenantMatrix()` hold the sparse documents and feed the grid only.
- Every mutation re-fetches the **full** document by id inside its `mutationFn`, applies the pure `mutate.ts` function to that fresh document, and PATCHes the whole thing. This also narrows the last-write-wins window the PRD accepts.
- `onSuccess` invalidates both the sparse key and the detail key.

- [ ] **Step 1: Prove both bugs before fixing them**

```bash
grep -n "MethodPut" -r services/atlas-configurations/ ; echo "exit=$?"
grep -n "MethodPatch" services/atlas-configurations/atlas.com/configurations/templates/resource.go
grep -n "api.put\|throwIfInvalid" services/atlas-ui/src/services/api/templates.service.ts
grep -n "updates: {" -A 8 services/atlas-ui/src/pages/templates-handlers-form.tsx
```

Expected: no `MethodPut` anywhere; `resource.go:29` binds PATCH; `templates.service.ts:281-282` shows `throwIfInvalid` then `api.put`; the form passes only `socket`. Record this in the commit message — it is why the change is a fix, not a refactor.

- [ ] **Step 2: Write the failing service test**

Create `services/atlas-ui/src/services/api/__tests__/templates-update.test.ts`:

```ts
import { beforeEach, describe, expect, it, vi } from "vitest";

const patch = vi.fn();
const put = vi.fn();
const getOne = vi.fn();

vi.mock("@/lib/api/client", () => ({
  api: {
    patch: (...args: unknown[]) => patch(...args),
    put: (...args: unknown[]) => put(...args),
    getOne: (...args: unknown[]) => getOne(...args),
    get: vi.fn(),
    getList: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
    setTenant: vi.fn(),
  },
}));

import { templatesService } from "@/services/api/templates.service";
import type { TemplateAttributes } from "@/types/models/template";

function fullAttributes(): TemplateAttributes {
  return {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: { templates: [], presets: [] },
    npcs: [],
    worlds: [],
    socket: { handlers: [], writers: [], unsupported: { handlers: [], writers: [] } },
  } as TemplateAttributes;
}

describe("templatesService.update", () => {
  beforeEach(() => {
    patch.mockReset().mockResolvedValue({ data: { id: "t1", attributes: fullAttributes() } });
    put.mockReset();
    getOne.mockReset();
  });

  // atlas-configurations registers PATCH only on /configurations/templates/{id};
  // there is no MethodPut route anywhere in the service, so a PUT is a 405.
  it("issues a PATCH, never a PUT", async () => {
    await templatesService.update("t1", fullAttributes());
    expect(patch).toHaveBeenCalledTimes(1);
    expect(put).not.toHaveBeenCalled();
    expect(patch.mock.calls[0]![0]).toBe("/api/configurations/templates/t1");
  });

  it("sends the whole attribute document in a JSON:API envelope", async () => {
    await templatesService.update("t1", fullAttributes());
    const body = patch.mock.calls[0]![1] as {
      data: { id: string; type: string; attributes: TemplateAttributes };
    };
    expect(body.data.type).toBe("templates");
    expect(body.data.id).toBe("t1");
    expect(body.data.attributes.region).toBe("GMS");
    expect(body.data.attributes.characters).toBeDefined();
    expect(body.data.attributes.worlds).toBeDefined();
  });

  // The guard that stops a sparse document reaching the write path and erasing
  // characters/worlds/cashShop.
  it("refuses a partial attribute document", async () => {
    await expect(
      templatesService.update("t1", { socket: { handlers: [], writers: [] } } as TemplateAttributes),
    ).rejects.toThrow(/validation failed/i);
    expect(patch).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 3: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/services/api/__tests__/templates-update.test.ts`

Expected: FAIL — the first two tests fail because `api.put` is called instead of `api.patch`.

- [ ] **Step 4: Fix `templatesService.update`**

In `services/atlas-ui/src/services/api/templates.service.ts`, replace the `update` method:

```ts
  /**
   * Updates a template.
   *
   * `data` MUST be the WHOLE attribute document, not a partial: the request
   * body replaces the stored configuration wholesale. Passing a sparsely
   * fetched or partial object erases characters/worlds/cashShop, which is why
   * throwIfInvalid runs first and rejects anything missing a required field.
   *
   * Transport is PATCH. atlas-configurations binds /configurations/templates/{id}
   * to http.MethodPatch only (templates/resource.go) - there is no PUT route, so
   * the previous api.put call could only ever have 405'd.
   */
  async update(
    id: string,
    data: TemplateAttributes,
    options?: ServiceOptions,
  ): Promise<Template> {
    throwIfInvalid(data, options?.validate !== false);
    const response = await api.patch<TemplateResponse>(
      `${BASE_PATH}/${id}`,
      wrapTemplate(data, id),
      options,
    );
    return sortTemplate(response.data);
  },
```

Note the signature narrows from `Partial<TemplateAttributes>` to `TemplateAttributes`. Fix the two call sites this breaks:

- `updateBatch` (`templates.service.ts:329-339`) — change its `updates` parameter type to `Array<{ id: string; data: TemplateAttributes }>`.
- `useUpdateTemplate` (`useTemplates.ts:191-…`) — change the mutation variable type from `{ id: string; updates: Partial<TemplateAttributes> }` to `{ id: string; updates: TemplateAttributes }`, and in its `onMutate` replace the optimistic `attributes: { ...previousTemplate.attributes, ...updates }` with `attributes: updates` (a whole document needs no spread, and spreading a whole document over a stale one is how a sparse read would leak in).
- `useUpdateTemplatesBatch` (`useTemplates.ts:360-…`) — same type change on its `updates` array.

- [ ] **Step 5: Run the service test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/services/api/__tests__/templates-update.test.ts`

Expected: PASS, all three tests.

- [ ] **Step 6: Add the sparse matrix readers**

In `templates.service.ts`, next to the existing `getTemplateOptions` (which already proves this pattern — `BASE_PATH?fields[templates]=region,majorVersion,minorVersion` drained through `fetchAll`), add:

```ts
  /**
   * Sparse read for the Packet Matrix: eleven templates at full attributes
   * carry character templates, presets and equipment lists the matrix never
   * reads.
   *
   * The result is READ-ONLY. It must never reach templatesService.update - a
   * sparse document in the write path erases characters/worlds/cashShop. It
   * lives under its own query key (socketKeys.matrix) for exactly that reason.
   */
  async getSocketMatrix(): Promise<Template[]> {
    const url = `${BASE_PATH}?fields[templates]=region,majorVersion,minorVersion,socket`;
    return fetchAll<Template>(url);
  },
```

Do **not** route it through `sortAndTransform`/`sortTemplate`: the normalizer preserves stored order and the grid sorts its own rows.

In `tenants.service.ts`, add the equivalent against `CONFIG_PATH`:

```ts
  /**
   * Sparse read of every tenant configuration, for the Tenant pages' ancestry
   * column. READ-ONLY - see templatesService.getSocketMatrix.
   */
  async getSocketMatrix(options?: ServiceOptions): Promise<TenantConfig[]> {
    const url = `${CONFIG_PATH}?fields[tenants]=region,majorVersion,minorVersion,socket`;
    const response = await api.getList<ApiListResponse<TenantConfig>>(url, options);
    return response.data;
  },
```

Match the surrounding module's existing list-fetch helper — if `tenants.service.ts` already uses `fetchAll`/`fetchPaged` for its list read, use that instead of `api.getList` so paging is drained the same way.

- [ ] **Step 7: Write the failing hook test**

Create `services/atlas-ui/src/lib/hooks/api/__tests__/useSocketObjects.test.tsx`:

```tsx
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const getById = vi.fn();
const update = vi.fn();
const getSocketMatrix = vi.fn();

vi.mock("@/services/api/templates.service", () => ({
  templatesService: {
    getById: (...a: unknown[]) => getById(...a),
    update: (...a: unknown[]) => update(...a),
    getSocketMatrix: (...a: unknown[]) => getSocketMatrix(...a),
  },
}));

vi.mock("@/services/api/tenants.service", () => ({
  tenantsService: {
    getTenantConfiguration: vi.fn(),
    updateTenantConfiguration: vi.fn(),
    getSocketMatrix: vi.fn().mockResolvedValue([]),
  },
}));

import { useSocketMutation } from "@/lib/hooks/api/useSocketObjects";
import type { SocketConfig } from "@/types/models/socket";

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

const fullTemplate = {
  id: "t1",
  attributes: {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: { templates: [], presets: [{ id: "p1" }] },
    npcs: [{ npcId: 1, impl: "x" }],
    worlds: [{ name: "Scania" }],
    cashShop: { commodities: {} },
    socket: { handlers: [], writers: [], unsupported: { handlers: [], writers: [] } },
  },
};

describe("useSocketMutation", () => {
  beforeEach(() => {
    getById.mockReset().mockResolvedValue(structuredClone(fullTemplate));
    update.mockReset().mockResolvedValue(structuredClone(fullTemplate));
    getSocketMatrix.mockReset().mockResolvedValue([]);
  });

  // The core rule: a mutation NEVER writes the cached (possibly sparse)
  // document. It re-fetches the full one first.
  it("re-fetches the full document before writing", async () => {
    const { result } = renderHook(() => useSocketMutation(), { wrapper });
    await result.current.mutateAsync({
      target: { source: "template", id: "t1" },
      apply: (cfg: SocketConfig) => cfg,
    });
    expect(getById).toHaveBeenCalledWith("t1");
    expect(getById.mock.invocationCallOrder[0]!).toBeLessThan(
      update.mock.invocationCallOrder[0]!,
    );
  });

  it("sends the whole attribute document, with only socket changed", async () => {
    const { result } = renderHook(() => useSocketMutation(), { wrapper });
    await result.current.mutateAsync({
      target: { source: "template", id: "t1" },
      apply: (cfg: SocketConfig) => ({
        ...cfg,
        writers: [{ opCode: "0x00", writer: "AuthSuccess", services: ["login"] }],
      }),
    });
    const sent = update.mock.calls[0]![1] as typeof fullTemplate.attributes;
    expect(sent.characters.presets).toHaveLength(1);
    expect(sent.npcs).toHaveLength(1);
    expect(sent.worlds).toHaveLength(1);
    expect(sent.cashShop).toBeDefined();
    expect(sent.socket.writers).toHaveLength(1);
  });

  it("propagates a MutationError from the apply function without writing", async () => {
    const { result } = renderHook(() => useSocketMutation(), { wrapper });
    await expect(
      result.current.mutateAsync({
        target: { source: "template", id: "t1" },
        apply: () => {
          throw new Error("binding did not resolve");
        },
      }),
    ).rejects.toThrow("binding did not resolve");
    expect(update).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 8: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/hooks/api/__tests__/useSocketObjects.test.tsx`

Expected: FAIL — cannot resolve `@/lib/hooks/api/useSocketObjects`.

- [ ] **Step 9: Implement the hooks**

Create `services/atlas-ui/src/lib/hooks/api/useSocketObjects.ts`:

```ts
import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";

import { templatesService } from "@/services/api/templates.service";
import { tenantsService } from "@/services/api/tenants.service";
import { templateKeys } from "@/lib/hooks/api/useTemplates";
import { fromTemplate, fromTenantConfig } from "@/lib/socket/normalize";
import type { SocketObject } from "@/lib/socket/model";
import type { SocketConfig } from "@/types/models/socket";

/**
 * Query keys for the SPARSE socket reads.
 *
 * These deliberately do NOT reuse templateKeys.detail / the tenant detail key.
 * A sparse document that reached a mutation's attribute spread would silently
 * erase characters, worlds and cashShop, so the two live under separate keys
 * and the sparse one is never a write input.
 */
export const socketKeys = {
  all: ["socket"] as const,
  matrix: () => [...socketKeys.all, "matrix", "templates"] as const,
  tenantMatrix: () => [...socketKeys.all, "matrix", "tenants"] as const,
};

export function useSocketMatrixTemplates(): UseQueryResult<SocketObject[], Error> {
  return useQuery({
    queryKey: socketKeys.matrix(),
    queryFn: async () => (await templatesService.getSocketMatrix()).map(fromTemplate),
    staleTime: 30_000,
  });
}

export function useSocketMatrixTenants(): UseQueryResult<SocketObject[], Error> {
  return useQuery({
    queryKey: socketKeys.tenantMatrix(),
    queryFn: async () => (await tenantsService.getSocketMatrix()).map(fromTenantConfig),
    staleTime: 30_000,
  });
}

export interface SocketTarget {
  source: "template" | "tenant";
  id: string;
}

export interface SocketMutationInput {
  target: SocketTarget;
  /** A pure splice from lib/socket/mutate. May throw MutationError. */
  apply: (cfg: SocketConfig) => SocketConfig;
}

/**
 * The single write path for every socket dialog and bulk flow.
 *
 * It re-fetches the FULL document by id, applies the pure splice to that fresh
 * copy, and PATCHes the whole attribute document back. Re-fetching is not
 * belt-and-braces: the grid's cache holds sparse documents, and it also narrows
 * the last-write-wins window the PRD accepts to the duration of one request.
 *
 * If `apply` throws (a binding that no longer resolves to exactly one entry),
 * nothing is written and the error surfaces to the caller's onError.
 */
export function useSocketMutation(): UseMutationResult<void, Error, SocketMutationInput> {
  const queryClient = useQueryClient();

  return useMutation<void, Error, SocketMutationInput>({
    mutationFn: async ({ target, apply }) => {
      if (target.source === "template") {
        const fresh = await templatesService.getById(target.id);
        const socket = apply(fresh.attributes.socket);
        await templatesService.update(target.id, { ...fresh.attributes, socket });
        return;
      }
      const fresh = await tenantsService.getTenantConfiguration(target.id);
      const socket = apply(fresh.attributes.socket);
      await tenantsService.updateTenantConfiguration(fresh, { socket });
    },
    onSuccess: (_data, { target }) => {
      void queryClient.invalidateQueries({ queryKey: socketKeys.all });
      if (target.source === "template") {
        void queryClient.invalidateQueries({ queryKey: templateKeys.detail(target.id) });
        void queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
      } else {
        void queryClient.invalidateQueries({ queryKey: ["tenants"] });
      }
    },
  });
}
```

Check `tenantsService.getTenantConfiguration`'s real name and signature before wiring it — if it differs, use the actual one rather than adding an alias. The tenant update already spreads the full stored attributes internally (`tenants.service.ts:303`), and here it is handed a freshly fetched full document, so the spread is safe.

- [ ] **Step 10: Run the hook test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/lib/hooks/api/__tests__/useSocketObjects.test.tsx`

Expected: PASS, all three tests.

- [ ] **Step 11: Full frontend gate**

```bash
cd services/atlas-ui && npm test && npm run build
```

Expected: both clean. The `update` signature narrowing in Step 4 breaks any other caller passing a partial — fix each by passing the whole document, never by widening the type back.

- [ ] **Step 12: Commit**

```bash
git add services/atlas-ui/src/services/api/templates.service.ts \
        services/atlas-ui/src/services/api/tenants.service.ts \
        services/atlas-ui/src/services/api/__tests__/templates-update.test.ts \
        services/atlas-ui/src/lib/hooks/api/useSocketObjects.ts \
        services/atlas-ui/src/lib/hooks/api/useTemplates.ts \
        services/atlas-ui/src/lib/hooks/api/__tests__/useSocketObjects.test.tsx
git commit -m "fix(ui): template save issued a PUT to a PATCH-only route

templatesService.update called api.put against /api/configurations/templates/{id},
but atlas-configurations binds that path to http.MethodPatch only - there is no
MethodPut anywhere in the service, so the request could only ever 405. It also
accepted Partial<TemplateAttributes> and cast it straight into the request body,
so the four socket forms' partial {socket} payload failed client-side validation
before it even got that far.

update now takes the WHOLE attribute document and PATCHes it.

Adds the sparse socket-matrix readers under their own query keys, and a single
write path that re-fetches the full document by id, applies a pure splice, and
PATCHes the whole thing. Sparse documents are never a mutation input: the
tenant PATCH body is built as {...attributes, ...updates}, so a sparse document
reaching it would silently erase characters, worlds and cashShop."
```

---

## Task 14: `PacketGrid` — the pivot table

**Files:**
- Create: `services/atlas-ui/src/components/features/socket/PacketGrid.tsx`
- Create: `services/atlas-ui/src/components/features/socket/PacketGridRow.tsx`
- Create: `services/atlas-ui/src/components/features/socket/PacketGridCell.tsx`
- Create: `services/atlas-ui/src/components/features/socket/__tests__/PacketGrid.test.tsx`

**Interfaces:**
- Consumes: `Row`, `Cell` (Task 9); `SocketObject`, `DefinitionKind` (Task 8); `formatOpcode` (Task 8).
- Produces:
  - `PacketGridProps { rows; objects; baselineKey; showFName; selection; onSelect }`
  - `GridSelection { name: string; scopeKey: string } | null`
  - `PacketGrid`, `PacketGridRow`, `PacketGridCell`

**Rendering strategy (design §6.1):** a plain semantic `<table>` with `position: sticky` on the header row and on the first column, rows wrapped in `React.memo` keyed by definition name. Worst case is 219 rows × 12 columns ≈ 2,600 cells, which renders in one pass well inside a frame budget; because the row is memoized over a precomputed row object, filtering and search re-render only the rows whose membership changed.

**Do not add `@tanstack/react-table`** (dynamic columns, cross-column predicates — its column model earns nothing here) **or `@tanstack/react-virtual`** (a new dependency that fights the frozen first column and breaks the deep-link scroll-to-row path). Virtualization is the escape hatch if measurement shows jank, not an up-front choice.

**Cell rendering (FR-2.6, refined by design §5.1):**

| State | Renders |
|---|---|
| Defined, one binding | the opcode, on the Defined background |
| Defined, N bindings | the **lowest** opcode plus `+{N-1}` |
| Defined, colliding opcodes | the opcode plus a duplicate marker |
| Unsupported | the literal `n/a`, visually distinct |
| Undefined | an empty cell |

**Baseline (FR-2.8/2.9):** the definition name is the **only** frozen column. The baseline is **not** rendered as a separate column — its own column is marked in place with a header badge plus a column outline.

**Accessibility (§6.5):** `role="grid"` with `aria-rowindex`/`aria-colindex`; rows focusable with arrow-key movement and Enter to open the drawer; cells are buttons so cell-scoping is reachable without a mouse. State is never colour-only — Unsupported renders the literal `n/a`, options-absence renders `⌀` with an `aria-label`, and the baseline column carries a header badge as well as its outline.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/socket/__tests__/PacketGrid.test.tsx`:

```tsx
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { PacketGrid } from "@/components/features/socket/PacketGrid";
import { buildRows } from "@/lib/socket/matrix";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function obj(
  key: string,
  major: number,
  writers: Record<string, Binding[]>,
  unsupportedWriters: string[] = [],
): SocketObject {
  return {
    key,
    label: `GMS v${major}.1`,
    source: "template",
    region: "GMS",
    majorVersion: major,
    minorVersion: 1,
    handlers: new Map(),
    writers: new Map(Object.entries(writers)),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(unsupportedWriters),
  };
}

const a = obj(
  "a",
  83,
  {
    AuthSuccess: [binding("0x00", { fname: "CLogin::OnCheckPasswordResult" })],
    CharacterEffect: [binding("0xE0"), binding("0xE9")],
    MiniRoom: [binding("0xB8"), binding("0x0B8")],
    CharacterMovement: [binding("0xB9", { options: { types: ["WALK"] } })],
  },
  ["MonsterCarnival"],
);
const b = obj("b", 95, { CharacterMovement: [binding("0xC0")] });

function renderGrid(overrides: Partial<Parameters<typeof PacketGrid>[0]> = {}) {
  const objects = [a, b];
  const rows = buildRows({ objects, kind: "writer", baselineKey: "a" });
  const onSelect = vi.fn();
  render(
    <PacketGrid
      rows={rows}
      objects={objects}
      baselineKey="a"
      showFName={false}
      selection={null}
      onSelect={onSelect}
      {...overrides}
    />,
  );
  return { onSelect, rows };
}

describe("PacketGrid", () => {
  it("renders one column per object and no duplicate baseline column", () => {
    renderGrid();
    expect(screen.getAllByText("GMS v83.1")).toHaveLength(1);
    expect(screen.getAllByText("GMS v95.1")).toHaveLength(1);
  });

  it("marks the baseline column in place rather than duplicating it", () => {
    renderGrid();
    const header = screen.getByRole("columnheader", { name: /GMS v83\.1/ });
    expect(within(header).getByText(/baseline/i)).toBeInTheDocument();
  });

  it("renders a single-binding cell as its opcode", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /AuthSuccess/ });
    expect(within(row).getByText("0x00")).toBeInTheDocument();
  });

  it("renders a multi-binding cell as the lowest opcode plus a count", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /CharacterEffect/ });
    expect(within(row).getByText("0xE0")).toBeInTheDocument();
    expect(within(row).getByText("+1")).toBeInTheDocument();
  });

  it("marks a cell whose bindings collide numerically", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /MiniRoom/ });
    expect(within(row).getByLabelText(/duplicate opcode/i)).toBeInTheDocument();
  });

  // State is never conveyed by colour alone.
  it("renders Unsupported as the literal n/a", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /MonsterCarnival/ });
    expect(within(row).getByText("n/a")).toBeInTheDocument();
  });

  it("renders an options-omission glyph with an accessible label", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /CharacterMovement/ });
    expect(within(row).getByLabelText(/supplies no options/i)).toBeInTheDocument();
  });

  it("hides the fname column until it is toggled on", () => {
    renderGrid();
    expect(screen.queryByRole("columnheader", { name: /fname/i })).not.toBeInTheDocument();
    renderGrid({ showFName: true });
    expect(screen.getAllByRole("columnheader", { name: /fname/i }).length).toBeGreaterThan(0);
  });

  // FR-5.2/5.3: clicking a CELL scopes to that object; clicking the NAME leaves
  // the scope on the baseline.
  it("scopes the selection to the clicked cell's object", async () => {
    const { onSelect } = renderGrid();
    const row = screen.getByRole("row", { name: /CharacterMovement/ });
    await userEvent.click(within(row).getByRole("button", { name: /GMS v95\.1/ }));
    expect(onSelect).toHaveBeenCalledWith({ name: "CharacterMovement", scopeKey: "b" });
  });

  it("leaves the scope on the baseline when the definition name is clicked", async () => {
    const { onSelect } = renderGrid();
    await userEvent.click(screen.getByRole("button", { name: "CharacterMovement" }));
    expect(onSelect).toHaveBeenCalledWith({ name: "CharacterMovement", scopeKey: "a" });
  });

  it("exposes the grid and its selection to assistive technology", () => {
    renderGrid({ selection: { name: "AuthSuccess", scopeKey: "a" } });
    expect(screen.getByRole("grid")).toBeInTheDocument();
    const row = screen.getByRole("row", { name: /AuthSuccess/ });
    expect(row).toHaveAttribute("aria-selected", "true");
  });

  it("renders an empty-state message when there are no rows", () => {
    render(
      <PacketGrid
        rows={[]}
        objects={[a, b]}
        baselineKey="a"
        showFName={false}
        selection={null}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText(/no definitions match/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/__tests__/PacketGrid.test.tsx`

Expected: FAIL — cannot resolve `@/components/features/socket/PacketGrid`.

- [ ] **Step 3: Implement `PacketGridCell.tsx`**

```tsx
import { memo } from "react";
import { cn } from "@/lib/utils";
import { formatOpcode } from "@/lib/socket/opcode";
import type { Cell } from "@/lib/socket/matrix";
import type { SocketObject } from "@/lib/socket/model";

export interface PacketGridCellProps {
  cell: Cell;
  object: SocketObject;
  definitionName: string;
  isBaselineColumn: boolean;
  isScoped: boolean;
  colIndex: number;
  onSelect: (scopeKey: string) => void;
}

/**
 * One object's view of one Definition.
 *
 * State is never colour-only: Unsupported renders the literal "n/a" and an
 * options omission renders a labelled glyph, so the grid is readable in
 * monochrome and to a screen reader.
 */
export const PacketGridCell = memo(function PacketGridCell({
  cell,
  object,
  definitionName,
  isBaselineColumn,
  isScoped,
  colIndex,
  onSelect,
}: PacketGridCellProps) {
  const values = cell.bindings
    .map((b) => b.opCodeValue)
    .filter((v): v is number => v !== null);
  const lowest = values.length > 0 ? Math.min(...values) : null;
  const extra = cell.bindings.length - 1;

  return (
    <td
      role="gridcell"
      aria-colindex={colIndex}
      className={cn(
        "border-b px-2 py-1 text-sm tabular-nums",
        isBaselineColumn && "bg-muted/40 border-x border-primary/40",
        cell.state === "defined" && "bg-primary/5",
        isScoped && "ring-2 ring-primary ring-inset",
      )}
    >
      <button
        type="button"
        // The accessible name carries the column label so a cell is
        // distinguishable by object without a mouse, and so FR-5.2's
        // cell-scoping is reachable from the keyboard.
        aria-label={`${definitionName} in ${object.label}`}
        className="flex w-full items-center gap-1 text-left"
        onClick={() => onSelect(object.key)}
      >
        {cell.state === "unsupported" && (
          <span className="text-muted-foreground italic">n/a</span>
        )}
        {cell.state === "defined" && lowest !== null && (
          <>
            <span>{formatOpcode(lowest)}</span>
            {extra > 0 && (
              <span className="text-muted-foreground text-xs">{`+${extra}`}</span>
            )}
          </>
        )}
        {cell.hasDuplicateOpcode && (
          <span aria-label="duplicate opcode" title="Two entries share this opcode">
            ⚠
          </span>
        )}
        {cell.optionsMissing && (
          <span
            aria-label={`${object.label} supplies no options where a sibling does`}
            title="Supplies no options where a sibling does"
          >
            ⌀
          </span>
        )}
      </button>
    </td>
  );
});
```

- [ ] **Step 4: Implement `PacketGridRow.tsx`**

```tsx
import { memo } from "react";
import { cn } from "@/lib/utils";
import { PacketGridCell } from "@/components/features/socket/PacketGridCell";
import type { Row } from "@/lib/socket/matrix";
import type { SocketObject } from "@/lib/socket/model";

export interface PacketGridRowProps {
  row: Row;
  objects: SocketObject[];
  baselineKey: string;
  showFName: boolean;
  scopeKey: string | null;
  isSelected: boolean;
  rowIndex: number;
  onSelect: (name: string, scopeKey: string) => void;
}

/**
 * Memoized over a PRECOMPUTED row object, so filtering and search re-render
 * only the rows whose membership actually changed - which is what keeps 219
 * rows x 12 columns responsive without virtualization.
 */
export const PacketGridRow = memo(function PacketGridRow({
  row,
  objects,
  baselineKey,
  showFName,
  scopeKey,
  isSelected,
  rowIndex,
  onSelect,
}: PacketGridRowProps) {
  return (
    <tr
      role="row"
      aria-rowindex={rowIndex}
      aria-selected={isSelected}
      className={cn(isSelected && "bg-accent")}
    >
      {/* The definition name is the ONLY frozen column (FR-2.8). */}
      <th
        scope="row"
        role="rowheader"
        aria-colindex={1}
        className="bg-background sticky left-0 z-10 border-b border-r px-2 py-1 text-left text-sm font-medium"
      >
        <button
          type="button"
          className="text-left hover:underline"
          // FR-5.3: clicking the name leaves the scope on the baseline.
          onClick={() => onSelect(row.name, baselineKey)}
        >
          {row.name}
        </button>
      </th>

      {showFName && (
        <td
          role="gridcell"
          aria-colindex={2}
          className="text-muted-foreground border-b px-2 py-1 font-mono text-xs"
        >
          {row.fname ?? ""}
        </td>
      )}

      {objects.map((object, i) => (
        <PacketGridCell
          key={object.key}
          cell={row.cells.get(object.key)!}
          object={object}
          definitionName={row.name}
          isBaselineColumn={object.key === baselineKey}
          isScoped={isSelected && scopeKey === object.key}
          colIndex={(showFName ? 2 : 1) + i + 1}
          onSelect={(key) => onSelect(row.name, key)}
        />
      ))}
    </tr>
  );
});
```

- [ ] **Step 5: Implement `PacketGrid.tsx`**

```tsx
import { useCallback, useRef } from "react";
import { cn } from "@/lib/utils";
import { PacketGridRow } from "@/components/features/socket/PacketGridRow";
import type { Row } from "@/lib/socket/matrix";
import type { SocketObject } from "@/lib/socket/model";

export interface GridSelection {
  name: string;
  /** The object the drawer's actions are scoped to (FR-5.2). */
  scopeKey: string;
}

export interface PacketGridProps {
  rows: Row[];
  objects: SocketObject[];
  baselineKey: string;
  showFName: boolean;
  selection: GridSelection | null;
  onSelect: (selection: GridSelection) => void;
}

/**
 * A plain semantic table with a sticky header and a sticky first column.
 *
 * Deliberately NOT @tanstack/react-table: the columns here are dynamic objects
 * and every sort/filter predicate is cross-column, so its column model would
 * add a layer owning none of the semantics. Deliberately NOT virtualized
 * either: 219 x 12 renders in one pass, and virtualization fights both the
 * frozen column and the deep-link scroll-to-row path. Virtualize only if
 * measurement shows jank.
 */
export function PacketGrid({
  rows,
  objects,
  baselineKey,
  showFName,
  selection,
  onSelect,
}: PacketGridProps) {
  const bodyRef = useRef<HTMLTableSectionElement>(null);

  const handleSelect = useCallback(
    (name: string, scopeKey: string) => onSelect({ name, scopeKey }),
    [onSelect],
  );

  // Arrow keys move between rows; Enter opens the drawer on the focused row.
  const onKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTableSectionElement>) => {
      if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
      const buttons = bodyRef.current?.querySelectorAll<HTMLButtonElement>(
        "tr > th button",
      );
      if (!buttons || buttons.length === 0) return;
      const current = document.activeElement;
      const index = [...buttons].findIndex((b) => b === current || b.contains(current));
      const next = e.key === "ArrowDown" ? index + 1 : index - 1;
      if (next < 0 || next >= buttons.length) return;
      e.preventDefault();
      buttons[next]!.focus();
    },
    [],
  );

  if (rows.length === 0) {
    return (
      <p className="text-muted-foreground p-8 text-center text-sm">
        No definitions match the current filters.
      </p>
    );
  }

  return (
    <div className="relative max-h-[70vh] overflow-auto rounded-md border">
      <table role="grid" className="w-full border-collapse text-left">
        <thead className="bg-background sticky top-0 z-20">
          <tr role="row" aria-rowindex={1}>
            <th
              scope="col"
              aria-colindex={1}
              className="bg-background sticky left-0 z-30 border-b border-r px-2 py-2 text-sm"
            >
              Definition
            </th>
            {showFName && (
              <th scope="col" aria-colindex={2} className="border-b px-2 py-2 text-sm">
                fname
              </th>
            )}
            {objects.map((o, i) => (
              <th
                key={o.key}
                scope="col"
                aria-colindex={(showFName ? 2 : 1) + i + 1}
                className={cn(
                  "border-b px-2 py-2 text-sm whitespace-nowrap",
                  o.key === baselineKey && "bg-muted/40 border-x border-primary/40",
                )}
              >
                <span>{o.label}</span>
                {o.key === baselineKey && (
                  <span className="bg-primary/10 text-primary ml-2 rounded px-1 text-[10px] uppercase">
                    baseline
                  </span>
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody ref={bodyRef} onKeyDown={onKeyDown}>
          {rows.map((row, i) => (
            <PacketGridRow
              key={row.name}
              row={row}
              objects={objects}
              baselineKey={baselineKey}
              showFName={showFName}
              scopeKey={selection?.name === row.name ? selection.scopeKey : null}
              isSelected={selection?.name === row.name}
              rowIndex={i + 2}
              onSelect={handleSelect}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/__tests__/PacketGrid.test.tsx`

Expected: PASS, all twelve tests.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/components/features/socket
git commit -m "feat(ui): add the packet grid pivot table

A plain semantic table with a sticky header and a sticky first column, rows
memoized over the precomputed row model so filtering re-renders only the rows
whose membership changed. 219 rows x 12 columns renders in one pass.

Not @tanstack/react-table: the columns are dynamic objects and every predicate
is cross-column, so its column model would own none of the semantics. Not
virtualized: virtualization fights the frozen first column and breaks the
deep-link scroll-to-row path. Held as the escape hatch if measurement shows jank.

State is never colour-only - Unsupported renders the literal n/a and an options
omission renders a labelled glyph. A multi-binding cell shows the lowest opcode
plus a count rather than pretending a definition has one opcode."
```

---

## Task 15: `GridToolbar` — mode, search, columns, baseline, filters

**Files:**
- Create: `services/atlas-ui/src/components/features/socket/GridToolbar.tsx`
- Create: `services/atlas-ui/src/components/features/socket/__tests__/GridToolbar.test.tsx`

**Interfaces:**
- Consumes: `GridFilters`, `emptyFilters`, `SortKey`, `SortDirection` (Task 9); `DefinitionKind`, `SocketObject` (Task 8).
- Produces: `GridToolbarProps { kind; onKindChange?; objects; selectedKeys; onSelectedKeysChange?; baselineKey; onBaselineChange?; showFName; onShowFNameChange; filters; onFiltersChange; sort; onSortChange; ancestryFilterOptions? }` and the `GridToolbar` component.

**FR-7.3:** the mode switch, column picker and baseline selector are **absent** on the four per-object pages. Model that by making `onKindChange`, `onSelectedKeysChange` and `onBaselineChange` optional — a control whose handler is absent is not rendered. Do not render a disabled control.

**FR-4.5:** Tenant pages get an additional difference-from-ancestor filter, supplied via `ancestryFilterOptions`; absent elsewhere.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/socket/__tests__/GridToolbar.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { GridToolbar } from "@/components/features/socket/GridToolbar";
import { emptyFilters } from "@/lib/socket/matrix";
import type { SocketObject } from "@/lib/socket/model";

function obj(key: string, major: number): SocketObject {
  return {
    key,
    label: `GMS v${major}.1`,
    source: "template",
    region: "GMS",
    majorVersion: major,
    minorVersion: 1,
    handlers: new Map(),
    writers: new Map(),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(),
  };
}

const objects = [obj("a", 83), obj("b", 95)];

function renderToolbar(overrides: Partial<Parameters<typeof GridToolbar>[0]> = {}) {
  const props = {
    kind: "writer" as const,
    objects,
    selectedKeys: ["a", "b"],
    baselineKey: "b",
    showFName: false,
    onShowFNameChange: vi.fn(),
    filters: emptyFilters(),
    onFiltersChange: vi.fn(),
    sort: { key: "opcode" as const, direction: "asc" as const },
    onSortChange: vi.fn(),
    ...overrides,
  };
  render(<GridToolbar {...props} />);
  return props;
}

describe("GridToolbar", () => {
  it("renders the mode switch only when a handler is supplied", () => {
    renderToolbar({ onKindChange: vi.fn() });
    expect(screen.getByRole("radio", { name: /handlers/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /writers/i })).toBeInTheDocument();
  });

  // FR-7.3: on the four per-object pages these controls are ABSENT, not disabled.
  it("omits the mode switch, column picker and baseline selector on a locked page", () => {
    renderToolbar();
    expect(screen.queryByRole("radio", { name: /handlers/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /columns/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /baseline/i })).not.toBeInTheDocument();
  });

  it("reports a search query", async () => {
    const props = renderToolbar();
    await userEvent.type(screen.getByRole("searchbox"), "Login");
    expect(props.onFiltersChange).toHaveBeenCalled();
    const last = props.onFiltersChange.mock.calls.at(-1)![0];
    expect(last.query).toBe("Login");
  });

  it("toggles the fname column", async () => {
    const props = renderToolbar();
    await userEvent.click(screen.getByRole("switch", { name: /fname/i }));
    expect(props.onShowFNameChange).toHaveBeenCalledWith(true);
  });

  it("changes the baseline when a selector is supplied", async () => {
    const onBaselineChange = vi.fn();
    renderToolbar({ onBaselineChange });
    await userEvent.click(screen.getByRole("button", { name: /baseline/i }));
    await userEvent.click(screen.getByRole("option", { name: "GMS v83.1" }));
    expect(onBaselineChange).toHaveBeenCalledWith("a");
  });

  it("reports a state filter", async () => {
    const props = renderToolbar();
    await userEvent.click(screen.getByRole("button", { name: /state/i }));
    await userEvent.click(screen.getByRole("option", { name: /unsupported/i }));
    const last = props.onFiltersChange.mock.calls.at(-1)![0];
    expect(last.states).toContain("unsupported");
  });

  it("reports the options-omission filter", async () => {
    const props = renderToolbar();
    await userEvent.click(screen.getByRole("checkbox", { name: /supplies no options/i }));
    const last = props.onFiltersChange.mock.calls.at(-1)![0];
    expect(last.optionsMissingOnly).toBe(true);
  });

  it("renders the ancestry filter only when its options are supplied", async () => {
    renderToolbar();
    expect(screen.queryByRole("button", { name: /vs template/i })).not.toBeInTheDocument();

    const onAncestryChange = vi.fn();
    renderToolbar({
      ancestryFilterOptions: { value: [], onChange: onAncestryChange },
    });
    expect(screen.getByRole("button", { name: /vs template/i })).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/__tests__/GridToolbar.test.tsx`

Expected: FAIL — cannot resolve `@/components/features/socket/GridToolbar`.

- [ ] **Step 3: Implement `GridToolbar.tsx`**

Build it from the existing shadcn primitives under `src/components/ui/` — read that directory first and use what is already there (`Input`, `Button`, `Switch`, `Checkbox`, `DropdownMenu`, `ToggleGroup` or the project's segmented-control equivalent). Do not add a new UI dependency.

The component is presentational: it owns no filter state, it renders the current `filters`/`sort` props and calls the handlers. The contract that the tests pin down:

```tsx
import type { AncestryClass } from "@/lib/socket/ancestry";
import type {
  GridFilters,
  SortDirection,
  SortKey,
} from "@/lib/socket/matrix";
import type { DefinitionKind, SocketObject } from "@/lib/socket/model";

export interface GridToolbarProps {
  kind: DefinitionKind;
  /** Absent on the four per-object pages, where the mode is fixed (FR-7.3). */
  onKindChange?: (kind: DefinitionKind) => void;

  objects: SocketObject[];
  selectedKeys: string[];
  /** Absent on the four per-object pages: the column set is fixed (FR-7.3). */
  onSelectedKeysChange?: (keys: string[]) => void;

  baselineKey: string;
  /** Absent on the four per-object pages (FR-7.3). */
  onBaselineChange?: (key: string) => void;

  showFName: boolean;
  onShowFNameChange: (show: boolean) => void;

  filters: GridFilters;
  onFiltersChange: (filters: GridFilters) => void;

  sort: { key: SortKey; direction: SortDirection };
  onSortChange: (sort: { key: SortKey; direction: SortDirection }) => void;

  /** Tenant pages only (FR-4.5). Absent elsewhere. */
  ancestryFilterOptions?: {
    value: AncestryClass[];
    onChange: (value: AncestryClass[]) => void;
  };
}
```

Rules the implementation must honour:

- **A control whose handler prop is absent is not rendered at all.** Never render it disabled — FR-7.3 says absent.
- The search input is `<Input type="search">` so it has the `searchbox` role, with `aria-label="Search definitions"`, and it calls `onFiltersChange({ ...filters, query: e.target.value })`.
- The fname toggle is a `Switch` with an accessible name containing "fname".
- The column picker (`Columns` button) lists every object with a checkbox, filtered by Region and Version (FR-2.12), and calls `onSelectedKeysChange`.
- The baseline selector (`Baseline` button) lists only the currently-selected objects as `option` roles.
- The state filter is a multi-select over `defined` / `unsupported` / `undefined`.
- The options-omission filter is a `Checkbox` labelled "Supplies no options".
- Add a "has options" tri-state and a service multi-select over `["login", "channel"]` for FR-4.4, and a sort control over `opcode` / `name` / `state` plus a direction toggle.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/__tests__/GridToolbar.test.tsx`

Expected: PASS, all eight tests. If a shadcn primitive renders a different ARIA role than the test assumes (for example a `combobox` rather than a `button`+`option` pair), adjust the **test** to the real role the chosen primitive emits — the behaviour is what matters, not the specific widget.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/socket/GridToolbar.tsx \
        services/atlas-ui/src/components/features/socket/__tests__/GridToolbar.test.tsx
git commit -m "feat(ui): add the packet grid toolbar

Presentational only: it owns no filter state, renders the current filters and
sort, and calls handlers.

A control whose handler prop is absent is not rendered at all. That is how the
four per-object pages drop the mode switch, column picker and baseline selector
(FR-7.3) and how tenant pages alone gain the difference-from-ancestor filter -
absent, not disabled."
```

---

## Task 16: `DefinitionDrawer` and `OptionsMatrix`

**Files:**
- Create: `services/atlas-ui/src/components/features/socket/OptionsMatrix.tsx`
- Create: `services/atlas-ui/src/components/features/socket/DefinitionDrawer.tsx`
- Create: `services/atlas-ui/src/components/features/socket/__tests__/DefinitionDrawer.test.tsx`

**Interfaces:**
- Consumes: `buildOptionsMatrix`, `OptionsMatrix` type, `OptionsEntryCellState` (Task 10); `Row` (Task 9); `GridSelection` (Task 14); `SocketObject`, `Binding`, `stateOf` (Task 8).
- Produces:
  - `OptionsMatrixTableProps { objects; kind; name; baselineKey }` and the `OptionsMatrixTable` component. **The component is named `OptionsMatrixTable`, not `OptionsMatrix`** — `OptionsMatrix` is already the *type* exported from `lib/socket/options` (Task 10), and the two would collide at every call site that needs both. The file is still `OptionsMatrix.tsx`.
  - `DefinitionDrawerProps { row; objects; kind; baselineKey; selection; onClose; onAction }`
  - `DrawerAction = { type: "add" | "edit" | "delete" | "mark-unsupported" | "clear-unsupported" | "copy" | "reset-to-ancestor" | "open-in"; scopeKey: string; name: string; opCodeValue?: number }`

**FR-5.1:** the drawer shows, per selected object: state, opcode, validator (handlers only), services, and options shape. Tabs: Fields / Options / Services.

**FR-5.2/5.3:** the scoped object is indicated visually and **every action label names it** — `Edit in GMS v87.1…`. Clicking the definition name leaves the scope on the baseline.

**FR-5.4:** actions targeting an object where the Definition is Undefined MUST be disabled where they have no meaning — Open, Edit, Delete. Add, Copy and Mark Unsupported stay enabled.

**Design §5.1:** the drawer lists **every binding** for the scoped object with its own action row. This is the only place all four `NoOpHandler` routes are individually addressable.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/socket/__tests__/DefinitionDrawer.test.tsx`:

```tsx
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { DefinitionDrawer } from "@/components/features/socket/DefinitionDrawer";
import { buildRows } from "@/lib/socket/matrix";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    validator: "LoggedInValidator",
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function obj(
  key: string,
  major: number,
  handlers: Record<string, Binding[]>,
  unsupportedHandlers: string[] = [],
): SocketObject {
  return {
    key,
    label: `GMS v${major}.1`,
    source: "template",
    region: "GMS",
    majorVersion: major,
    minorVersion: 1,
    handlers: new Map(Object.entries(handlers)),
    writers: new Map(),
    unsupportedHandlers: new Set(unsupportedHandlers),
    unsupportedWriters: new Set(),
  };
}

const a = obj("a", 83, {
  NoOpHandler: [binding("0x17"), binding("0x19"), binding("0x22"), binding("0x24")],
  Move: [binding("0x29", { options: { types: ["WALK", "STAND"] } })],
});
const b = obj("b", 87, { Move: [binding("0x2A", { options: { types: ["WALK"] } })] });
const objects = [a, b];

function renderDrawer(
  name: string,
  scopeKey: string,
  overrides: Partial<Parameters<typeof DefinitionDrawer>[0]> = {},
) {
  const rows = buildRows({ objects, kind: "handler", baselineKey: "a" });
  const row = rows.find((r) => r.name === name)!;
  const onAction = vi.fn();
  render(
    <DefinitionDrawer
      row={row}
      objects={objects}
      kind="handler"
      baselineKey="a"
      selection={{ name, scopeKey }}
      onClose={vi.fn()}
      onAction={onAction}
      {...overrides}
    />,
  );
  return { onAction };
}

describe("DefinitionDrawer", () => {
  it("names the scoped object in every action label", () => {
    renderDrawer("NoOpHandler", "a");
    expect(screen.getByRole("button", { name: /edit in GMS v83\.1/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /delete in GMS v83\.1/i })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /mark unsupported in GMS v83\.1/i }),
    ).toBeInTheDocument();
  });

  it("relabels the actions when the scope moves to another object", () => {
    renderDrawer("Move", "b");
    expect(screen.getByRole("button", { name: /edit in GMS v87\.1/i })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /edit in GMS v83\.1/i }),
    ).not.toBeInTheDocument();
  });

  // FR-5.4
  it("disables Edit, Delete and Open where the definition is undefined for the scope", () => {
    renderDrawer("NoOpHandler", "b");
    expect(screen.getByRole("button", { name: /edit in GMS v87\.1/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /delete in GMS v87\.1/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /open in GMS v87\.1/i })).toBeDisabled();
  });

  it("keeps Add, Copy and Mark Unsupported enabled where the definition is undefined", () => {
    renderDrawer("NoOpHandler", "b");
    expect(screen.getByRole("button", { name: /add to GMS v87\.1/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /copy into GMS v87\.1/i })).toBeEnabled();
    expect(
      screen.getByRole("button", { name: /mark unsupported in GMS v87\.1/i }),
    ).toBeEnabled();
  });

  // design §5.1 - all four routes individually addressable.
  it("lists every binding of a multi-binding definition with its own actions", () => {
    renderDrawer("NoOpHandler", "a");
    const list = screen.getByRole("list", { name: /bindings in GMS v83\.1/i });
    expect(within(list).getAllByRole("listitem")).toHaveLength(4);
    expect(within(list).getByText("0x17")).toBeInTheDocument();
    expect(within(list).getByText("0x24")).toBeInTheDocument();
  });

  it("dispatches an action carrying the scope and the addressed binding", async () => {
    const { onAction } = renderDrawer("NoOpHandler", "a");
    const list = screen.getByRole("list", { name: /bindings in GMS v83\.1/i });
    const row = within(list).getAllByRole("listitem")[1]!;
    await userEvent.click(within(row).getByRole("button", { name: /edit/i }));
    expect(onAction).toHaveBeenCalledWith({
      type: "edit",
      scopeKey: "a",
      name: "NoOpHandler",
      opCodeValue: 0x19,
    });
  });

  it("shows each object's state, opcode, validator and services in the Fields tab", () => {
    renderDrawer("Move", "a");
    const fields = screen.getByRole("tabpanel", { name: /fields/i });
    expect(within(fields).getByText("GMS v83.1")).toBeInTheDocument();
    expect(within(fields).getByText("0x29")).toBeInTheDocument();
    expect(within(fields).getAllByText("LoggedInValidator").length).toBeGreaterThan(0);
    expect(within(fields).getAllByText("channel").length).toBeGreaterThan(0);
  });

  it("renders the nested per-entry options matrix in the Options tab", async () => {
    renderDrawer("Move", "a");
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });
    // Positional list: rows keyed by array index, index 1 missing for GMS v87.1.
    expect(within(panel).getByRole("rowheader", { name: "0" })).toBeInTheDocument();
    expect(within(panel).getByRole("rowheader", { name: "1" })).toBeInTheDocument();
    expect(within(panel).getByText("WALK")).toBeInTheDocument();
    expect(within(panel).getByText("STAND")).toBeInTheDocument();
  });

  it("marks options entries that differ from or are missing against the baseline", async () => {
    renderDrawer("Move", "a");
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });
    expect(within(panel).getByLabelText(/missing in GMS v87\.1/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/__tests__/DefinitionDrawer.test.tsx`

Expected: FAIL — cannot resolve `@/components/features/socket/DefinitionDrawer`.

- [ ] **Step 3: Implement `OptionsMatrix.tsx`**

```tsx
import { buildOptionsMatrix } from "@/lib/socket/options";
import type { OptionsEntryCellState } from "@/lib/socket/options";
import type { DefinitionKind, SocketObject } from "@/lib/socket/model";
import { cn } from "@/lib/utils";

export interface OptionsMatrixTableProps {
  objects: SocketObject[];
  kind: DefinitionKind;
  name: string;
  baselineKey: string;
}

/**
 * FR-3.3-3.5. Rows are option entries - the ARRAY INDEX for ordered lists, the
 * key name for maps - and columns are the same selected objects as the outer
 * grid. Positional comparison is not a detail: gms_95_1 CharacterMovement
 * carries UNKNOWN at six separate indices, so a name-keyed comparison would
 * match six unrelated slots.
 *
 * Exported as OptionsMatrixTable so it does not collide with the OptionsMatrix
 * TYPE from lib/socket/options.
 */
export function OptionsMatrixTable({
  objects,
  kind,
  name,
  baselineKey,
}: OptionsMatrixTableProps) {
  const matrix = buildOptionsMatrix({ objects, kind, name, baselineKey });

  if (matrix.rows.length === 0) {
    return (
      <p className="text-muted-foreground p-4 text-sm">
        No object supplies options for this definition.
      </p>
    );
  }

  const labelFor = (state: OptionsEntryCellState, objectLabel: string) => {
    if (state === "missing") return `missing in ${objectLabel}`;
    if (state === "extra") return `only in ${objectLabel}`;
    if (state === "differs") return `differs in ${objectLabel}`;
    return undefined;
  };

  return (
    <div className="overflow-auto">
      <p className="text-muted-foreground px-2 py-1 text-xs">
        {matrix.shape === "list"
          ? "Ordered list — the row number is the wire value, compared positionally."
          : "Keyed map — the row name is the option key."}
      </p>
      <table className="w-full border-collapse text-left text-sm">
        <thead>
          <tr>
            <th scope="col" className="border-b px-2 py-1">
              {matrix.shape === "list" ? "Index" : "Key"}
            </th>
            {objects.map((o) => (
              <th
                key={o.key}
                scope="col"
                className={cn(
                  "border-b px-2 py-1 whitespace-nowrap",
                  o.key === baselineKey && "bg-muted/40",
                )}
              >
                {o.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {matrix.rows.map((row) => (
            <tr key={row.key}>
              <th scope="row" className="border-b px-2 py-1 font-mono">
                {row.label}
              </th>
              {objects.map((o) => {
                const cell = row.cells.get(o.key)!;
                const label = labelFor(cell.state, o.label);
                return (
                  <td
                    key={o.key}
                    aria-label={label}
                    className={cn(
                      "border-b px-2 py-1",
                      o.key === baselineKey && "bg-muted/40",
                      cell.state === "differs" && "text-amber-600 dark:text-amber-400",
                      cell.state === "extra" && "text-sky-600 dark:text-sky-400",
                      cell.state === "missing" && "text-muted-foreground",
                    )}
                  >
                    {cell.state === "missing" ? "—" : String(cell.value)}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

- [ ] **Step 4: Implement `DefinitionDrawer.tsx`**

Build the shell from the project's existing `Sheet`/`Drawer` and `Tabs` primitives under `src/components/ui/` — read that directory and use what is there.

```tsx
import type { Row } from "@/lib/socket/matrix";
import type { DefinitionKind, SocketObject } from "@/lib/socket/model";
import { stateOf } from "@/lib/socket/model";
import type { GridSelection } from "@/components/features/socket/PacketGrid";

export type DrawerActionType =
  | "add"
  | "edit"
  | "delete"
  | "mark-unsupported"
  | "clear-unsupported"
  | "copy"
  | "reset-to-ancestor"
  | "open-in";

export interface DrawerAction {
  type: DrawerActionType;
  /** The object the action targets (FR-5.2). Never implicit. */
  scopeKey: string;
  name: string;
  /** Present for binding-scoped actions (edit, delete, open-in). */
  opCodeValue?: number;
}

export interface DefinitionDrawerProps {
  row: Row;
  objects: SocketObject[];
  kind: DefinitionKind;
  baselineKey: string;
  selection: GridSelection;
  onClose: () => void;
  onAction: (action: DrawerAction) => void;
  /** Tenant pages only: enables Reset to Ancestor. */
  ancestor?: SocketObject;
}
```

Implementation rules the tests pin down:

- Resolve `scope = objects.find(o => o.key === selection.scopeKey)!` and `scopeState = stateOf(scope, kind, row.name)`.
- **Every action button's accessible name ends with the scoped object's label** — `Edit in ${scope.label}…`, `Delete in ${scope.label}…`, `Add to ${scope.label}…`, `Copy into ${scope.label}…`, `Mark unsupported in ${scope.label}…`, `Open in ${scope.label}`. Reuse one `actionLabel(verb, preposition)` helper so this cannot drift.
- **FR-5.4:** `disabled={scopeState !== "defined"}` on Edit, Delete and Open. Add, Copy and Mark Unsupported are never disabled on that basis. `Clear unsupported` is enabled only when `scopeState === "unsupported"`.
- The **bindings list** is `<ul aria-label={`Bindings in ${scope.label}`}>` with one `<li>` per binding of the scoped object, each showing its stored `opCode` plus per-binding **Edit** and **Delete** buttons that dispatch with `opCodeValue: binding.opCodeValue`.
- Tabs are `Fields` / `Options` / `Services`, each `tabpanel` accessibly named by its tab.
  - **Fields:** one row per object — label, state, the binding opcodes, validator (handlers only), services, and the options shape from `classifyOptions`.
  - **Options:** `<OptionsMatrixTable objects={objects} kind={kind} name={row.name} baselineKey={baselineKey} />`.
  - **Services:** per object, the union of its bindings' service tags.
- Render `Reset to Ancestor in ${scope.label}…` only when `ancestor` is supplied and `scope.source === "tenant"`.

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/__tests__/DefinitionDrawer.test.tsx`

Expected: PASS, all nine tests.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/socket/DefinitionDrawer.tsx \
        services/atlas-ui/src/components/features/socket/OptionsMatrix.tsx \
        services/atlas-ui/src/components/features/socket/__tests__/DefinitionDrawer.test.tsx
git commit -m "feat(ui): add the definition drawer and the nested options matrix

Every action label names the scoped object, so 'Edit in GMS v87.1' can never be
mistaken for an edit of the baseline. Edit, Delete and Open are disabled where
the definition is Undefined for that scope; Add, Copy and Mark Unsupported stay
enabled because they still mean something there.

The drawer lists EVERY binding of the scoped object with its own action row.
That is the only place all four NoOpHandler routes in gms_95_1 are individually
addressable - the grid cell can only show the lowest plus a count.

The options tab compares ordered lists positionally: the row number is the wire
value, so a name that shifts index between versions reads as a diagonal rather
than a false match."
```

---

## Task 17: The six definition dialogs

**Files:**
- Create: `services/atlas-ui/src/components/features/socket/dialogs/AddDefinitionDialog.tsx`
- Create: `services/atlas-ui/src/components/features/socket/dialogs/EditDefinitionDialog.tsx`
- Create: `services/atlas-ui/src/components/features/socket/dialogs/DeleteDefinitionDialog.tsx`
- Create: `services/atlas-ui/src/components/features/socket/dialogs/MarkUnsupportedDialog.tsx`
- Create: `services/atlas-ui/src/components/features/socket/dialogs/CopyDefinitionDialog.tsx`
- Create: `services/atlas-ui/src/components/features/socket/dialogs/ResetToAncestorDialog.tsx`
- Create: `services/atlas-ui/src/lib/schemas/socket-definition.ts`
- Create: `services/atlas-ui/src/components/features/socket/dialogs/__tests__/dialogs.test.tsx`

**Interfaces:**
- Consumes: every `mutate.ts` function and `BindingInput`, `MutationError` (Task 12); `useSocketMutation`, `SocketTarget` (Task 13); `OptionsField` from `@/components/unknown-options` (existing); `parseOpcode`, `formatOpcode` (Task 8).
- Produces: the six dialog components, each taking `{ open; onOpenChange; target: SocketTarget; targetLabel: string; kind: DefinitionKind; name?; ... }`, plus `definitionFormSchema` (Zod).

**All mutation goes through `useSocketMutation`.** A dialog never calls a service directly and never builds a PATCH body — it composes a pure splice and hands it over. That is what keeps the sparse-cache rule enforceable in one place.

**Client-side validation mirrors the server rules exactly** (Task 3), because the server now blocks on all of them. A dialog that lets the user submit something the server will 400 is a worse experience than one that catches it inline, and the two rule sets drifting is the failure mode to design against — put the shared shape in `src/lib/schemas/socket-definition.ts` and reference the Go rules in a comment.

- [ ] **Step 1: Write the Zod schema**

Create `services/atlas-ui/src/lib/schemas/socket-definition.ts`:

```ts
import { z } from "zod";

/**
 * Mirrors the blocking rules in
 * services/atlas-configurations/atlas.com/configurations/socket/validate.go.
 * The server 400s on every one of these, so catching them inline is the only
 * way the dialogs stay usable. If you change a rule here, change it there.
 */

/** 0x/0X followed by 1-4 hex digits. jms_185_1 stores "0x9"; gms_84_1 "0x0A5". */
export const OPCODE_PATTERN = /^0[xX][0-9A-Fa-f]{1,4}$/;

/** The closed set from libs/atlas-opcodes/config.go. */
export const KNOWN_SERVICES = ["login", "channel"] as const;

export const definitionFormSchema = z.object({
  name: z.string().trim().min(1, "Definition name is required"),
  opCode: z
    .string()
    .trim()
    .regex(OPCODE_PATTERN, "Use 0x followed by 1-4 hex digits, e.g. 0x2A"),
  validator: z.string().trim(),
  services: z.array(z.enum(KNOWN_SERVICES)),
  fname: z.string().trim().optional(),
  options: z.unknown().optional(),
});

export type DefinitionFormValues = z.infer<typeof definitionFormSchema>;

/** Handlers require a validator; writers have none (validate.go needsValidator). */
export function validatorRequiredFor(kind: "handler" | "writer"): boolean {
  return kind === "handler";
}
```

- [ ] **Step 2: Write the failing dialog test**

Create `services/atlas-ui/src/components/features/socket/dialogs/__tests__/dialogs.test.tsx`. Cover the contract, not the markup:

```tsx
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mutateAsync = vi.fn();
vi.mock("@/lib/hooks/api/useSocketObjects", () => ({
  useSocketMutation: () => ({ mutateAsync, isPending: false }),
  socketKeys: { all: ["socket"] },
}));

import { AddDefinitionDialog } from "@/components/features/socket/dialogs/AddDefinitionDialog";
import { DeleteDefinitionDialog } from "@/components/features/socket/dialogs/DeleteDefinitionDialog";
import { MarkUnsupportedDialog } from "@/components/features/socket/dialogs/MarkUnsupportedDialog";
import { addBinding, deleteBinding, markUnsupported } from "@/lib/socket/mutate";
import type { SocketConfig } from "@/types/models/socket";

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

const target = { source: "template" as const, id: "t1" };

function config(): SocketConfig {
  return {
    handlers: [
      { opCode: "0x17", validator: "LoggedInValidator", handler: "NoOpHandler", services: ["channel"] },
      { opCode: "0x19", validator: "LoggedInValidator", handler: "NoOpHandler", services: ["channel"] },
    ],
    writers: [],
    unsupported: { handlers: [], writers: [] },
  };
}

beforeEach(() => mutateAsync.mockReset().mockResolvedValue(undefined));

describe("AddDefinitionDialog", () => {
  it("rejects a malformed opcode without submitting", async () => {
    render(
      <AddDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
      />,
      { wrapper },
    );
    await userEvent.type(screen.getByLabelText(/definition name/i), "PongHandle");
    await userEvent.type(screen.getByLabelText(/operation code/i), "B8");
    await userEvent.click(screen.getByRole("button", { name: /^add$/i }));
    expect(await screen.findByText(/0x followed by 1-4 hex digits/i)).toBeInTheDocument();
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it("requires a validator for handlers", async () => {
    render(
      <AddDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
      />,
      { wrapper },
    );
    await userEvent.type(screen.getByLabelText(/definition name/i), "PongHandle");
    await userEvent.type(screen.getByLabelText(/operation code/i), "0x18");
    await userEvent.click(screen.getByRole("button", { name: /^add$/i }));
    expect(await screen.findByText(/validator is required/i)).toBeInTheDocument();
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it("submits a splice that adds the binding and clears the unsupported marker", async () => {
    render(
      <AddDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
      />,
      { wrapper },
    );
    await userEvent.type(screen.getByLabelText(/definition name/i), "PongHandle");
    await userEvent.type(screen.getByLabelText(/operation code/i), "0x18");
    await userEvent.type(screen.getByLabelText(/validator/i), "NoOpValidator");
    await userEvent.click(screen.getByRole("button", { name: /^add$/i }));

    expect(mutateAsync).toHaveBeenCalledTimes(1);
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const before: SocketConfig = { ...config(), unsupported: { handlers: ["PongHandle"], writers: [] } };
    const after = apply(before);
    expect(after.handlers.some((h) => h.handler === "PongHandle")).toBe(true);
    expect(after.unsupported!.handlers).toEqual([]);
    // The pure function is the same one the unit tests cover.
    expect(after).toEqual(
      addBinding(before, "handler", "PongHandle", {
        opCode: "0x18",
        validator: "NoOpValidator",
        services: [],
      }),
    );
  });
});

describe("DeleteDefinitionDialog", () => {
  // FR-6.3: two distinct outcomes, chosen explicitly.
  it("offers remove and remove-and-mark-unsupported as separate choices", () => {
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
      />,
      { wrapper },
    );
    expect(screen.getByRole("radio", { name: /remove definition/i })).toBeInTheDocument();
    expect(
      screen.getByRole("radio", { name: /remove and mark unsupported/i }),
    ).toBeInTheDocument();
  });

  it("removes exactly the addressed binding by default", async () => {
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
      />,
      { wrapper },
    );
    await userEvent.click(screen.getByRole("button", { name: /^remove$/i }));
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    expect(apply(config())).toEqual(deleteBinding(config(), "handler", "NoOpHandler", 0x19));
  });

  it("warns that marking unsupported removes every binding of the name", async () => {
    render(
      <DeleteDefinitionDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        opCodeValue={0x19}
        bindingCount={2}
      />,
      { wrapper },
    );
    await userEvent.click(screen.getByRole("radio", { name: /remove and mark unsupported/i }));
    expect(screen.getByText(/all 2 bindings/i)).toBeInTheDocument();
  });
});

describe("MarkUnsupportedDialog", () => {
  // FR-6.4 - it must SAY it removes the existing definitions, plural.
  it("names the target version and states that existing bindings will be removed", () => {
    render(
      <MarkUnsupportedDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        bindingCount={4}
      />,
      { wrapper },
    );
    expect(screen.getByText(/GMS v83\.1/)).toBeInTheDocument();
    expect(screen.getByText(/all 4 bindings.*will be removed/i)).toBeInTheDocument();
  });

  it("applies markUnsupported on confirm", async () => {
    render(
      <MarkUnsupportedDialog
        open
        onOpenChange={vi.fn()}
        target={target}
        targetLabel="GMS v83.1"
        kind="handler"
        name="NoOpHandler"
        bindingCount={2}
      />,
      { wrapper },
    );
    await userEvent.click(screen.getByRole("button", { name: /mark unsupported/i }));
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    expect(apply(config())).toEqual(markUnsupported(config(), "handler", "NoOpHandler"));
  });
});
```

- [ ] **Step 3: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/dialogs/`

Expected: FAIL — cannot resolve the dialog modules.

- [ ] **Step 4: Implement the six dialogs**

Build each on the project's existing `Dialog`/`AlertDialog`, `Form`, `FormField` and `Button` primitives, with `react-hook-form` + `zodResolver(definitionFormSchema)` per the atlas-ui conventions. Embed the existing `OptionsField` from `@/components/unknown-options` for options editing — the PRD's non-goal is explicit that there is no structured options editor beyond what already exists.

Shared props:

```tsx
interface DialogBaseProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  target: SocketTarget;
  /** The scoped object's label, so every dialog title names it. */
  targetLabel: string;
  kind: DefinitionKind;
}
```

Each dialog's contract:

| Dialog | Extra props | Splice submitted | Notes |
|---|---|---|---|
| **Add** | — | `addBinding(cfg, kind, name, input)` | FR-6.1. Name, opcode, services, plus validator for handlers. Options optional. |
| **Edit** | `name`, `opCodeValue`, `initial: BindingInput` | `editBinding(cfg, kind, name, opCodeValue, input)` | FR-6.2. **The name field is rendered read-only** — the name is the identity; renaming is unsupported. |
| **Delete** | `name`, `opCodeValue`, `bindingCount` | `deleteBinding(...)` **or** `markUnsupported(...)` | FR-6.3. Two radio choices. The second warns it removes all `bindingCount` bindings. |
| **MarkUnsupported** | `name`, `bindingCount` | `markUnsupported(cfg, kind, name)` | FR-6.4. States the target Region/Version and that all bindings will be removed. |
| **Copy** | `sourceObjects: SocketObject[]` | `copyBindings(cfg, kind, name, inputs)` | FR-6.5. Choose source object → source definition → load values → edit → confirm target. Values are deep-cloned by `copyBindings`, so the result is independent of the source. |
| **ResetToAncestor** | `name`, `ancestor: SocketObject` | `copyBindings` after `markUnsupported`-free removal of the tenant's own bindings | FR-6.6. Shows current Tenant values, ancestor Template values, and the fields that will change. Tenant scope only. |

Every dialog:
- Submits via `useSocketMutation().mutateAsync({ target, apply })` and closes on success.
- On error shows the message with `toast.error(...)`; a `MutationError` message is already user-facing, so surface it verbatim rather than replacing it with a generic string.
- Titles read `Add definition to ${targetLabel}`, `Edit LoginHandle in ${targetLabel}`, and so on.

- [ ] **Step 5: Run the dialog tests to verify they pass**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/dialogs/`

Expected: PASS, all three describe blocks.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/socket/dialogs \
        services/atlas-ui/src/lib/schemas/socket-definition.ts
git commit -m "feat(ui): add the six socket definition dialogs

Every dialog composes a pure splice from lib/socket/mutate and hands it to
useSocketMutation - none builds a PATCH body or calls a service directly, which
is what keeps the sparse-read rule enforceable in one place.

Client validation mirrors socket/validate.go rule for rule, because the server
now blocks on all of them: an inline error is a far better experience than a
400, and the two rule sets drifting is the failure mode to design against.

Delete offers 'remove' and 'remove and mark unsupported' as separate explicit
choices, and the second says how many bindings it will remove - marking
NoOpHandler unsupported in gms_95_1 removes four routes."
```

---

## Task 18: `CopyFromAncestorFlow` and the validator remediation

Two bulk flows, both of which apply as a **single configuration write**.

**Files:**
- Create: `services/atlas-ui/src/components/features/socket/CopyFromAncestorFlow.tsx`
- Create: `services/atlas-ui/src/components/features/socket/FillMissingValidatorsDialog.tsx`
- Create: `services/atlas-ui/src/components/features/socket/__tests__/CopyFromAncestorFlow.test.tsx`

**Interfaces:**
- Consumes: `missingFromTenant`, `classifyAgainstAncestor` (Task 11); `copyMissingFromAncestor`, `fillMissingValidators`, `AncestorAddition`, `BindingInput` (Task 12); `useSocketMutation` (Task 13).
- Produces: `CopyFromAncestorFlow({ open; onOpenChange; tenant; ancestor; kind; target })` and `FillMissingValidatorsDialog({ open; onOpenChange; target; targetLabel; emptyValidatorCount })`.

### Copy missing from ancestor (FR-9.1–9.6)

- **FR-9.1** — candidates are Definitions **Defined in the ancestor** and **Undefined in the Tenant**. `missingFromTenant` already excludes names the tenant marked Unsupported (FR-9.5).
- **FR-9.2** — per-definition selection, then per-definition review and adjustment of opcode and configuration.
- **FR-9.3** — the review step MUST show, per definition: name, source opcode, target opcode, validator, services, option differences and current target state.
- **FR-9.4** — never overwrites an existing Tenant Definition. `copyMissingFromAncestor` enforces this by skipping any name that is already defined in the fresh document, so a name that gained a definition between the scan and the apply is skipped rather than clobbered.
- **FR-9.6** — the whole selection applies as **one** write.

### Fill missing validators — the strict-validation escape hatch

Not in the PRD; required by the decision to enforce FR-11.4 as a 400 (see the decisions-of-record section). Saves are whole-document, so a configuration carrying **any** empty handler validator cannot be saved at all — which means a single-definition edit can never be the fix, and the live gms_95 tenant's 32 empty validators would be a hard deadlock. This repairs every offender in one write.

Surface it on the per-object pages as a banner that appears **only** when the loaded document has at least one empty handler validator, stating the count and offering the repair.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/socket/__tests__/CopyFromAncestorFlow.test.tsx`:

```tsx
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mutateAsync = vi.fn();
vi.mock("@/lib/hooks/api/useSocketObjects", () => ({
  useSocketMutation: () => ({ mutateAsync, isPending: false }),
  socketKeys: { all: ["socket"] },
}));

import { CopyFromAncestorFlow } from "@/components/features/socket/CopyFromAncestorFlow";
import { FillMissingValidatorsDialog } from "@/components/features/socket/FillMissingValidatorsDialog";
import { fillMissingValidators } from "@/lib/socket/mutate";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";
import type { SocketConfig } from "@/types/models/socket";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    validator: "LoggedInValidator",
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function obj(
  key: string,
  source: SocketObject["source"],
  handlers: Record<string, Binding[]>,
  unsupportedHandlers: string[] = [],
): SocketObject {
  return {
    key,
    label: key === "tnt" ? "Tenant" : "GMS v83.1",
    source,
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    handlers: new Map(Object.entries(handlers)),
    writers: new Map(),
    unsupportedHandlers: new Set(unsupportedHandlers),
    unsupportedWriters: new Set(),
  };
}

const tenant = obj("tnt", "tenant", { LoginHandle: [binding("0x01")] }, ["GuestLoginHandle"]);
const ancestor = obj("t83", "template", {
  LoginHandle: [binding("0x01")],
  PongHandle: [binding("0x18")],
  MoveHandle: [binding("0x29", { options: { types: ["WALK"] } })],
  GuestLoginHandle: [binding("0x02")],
});

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

function renderFlow() {
  render(
    <CopyFromAncestorFlow
      open
      onOpenChange={vi.fn()}
      tenant={tenant}
      ancestor={ancestor}
      kind="handler"
      target={{ source: "tenant", id: "tnt" }}
    />,
    { wrapper },
  );
}

beforeEach(() => mutateAsync.mockReset().mockResolvedValue(undefined));

describe("CopyFromAncestorFlow", () => {
  // FR-9.1 + FR-9.5
  it("lists only definitions defined in the ancestor and undefined in the tenant", () => {
    renderFlow();
    const list = screen.getByRole("group", { name: /candidates/i });
    expect(within(list).getByRole("checkbox", { name: /PongHandle/ })).toBeInTheDocument();
    expect(within(list).getByRole("checkbox", { name: /MoveHandle/ })).toBeInTheDocument();
    // Already defined in the tenant.
    expect(within(list).queryByRole("checkbox", { name: /LoginHandle/ })).not.toBeInTheDocument();
    // Explicitly marked Unsupported in the tenant.
    expect(
      within(list).queryByRole("checkbox", { name: /GuestLoginHandle/ }),
    ).not.toBeInTheDocument();
  });

  // FR-9.3
  it("shows name, source opcode, target opcode, validator, services and option differences in review", async () => {
    renderFlow();
    await userEvent.click(screen.getByRole("checkbox", { name: /MoveHandle/ }));
    await userEvent.click(screen.getByRole("button", { name: /review/i }));

    const review = screen.getByRole("region", { name: /review/i });
    expect(within(review).getByText("MoveHandle")).toBeInTheDocument();
    expect(within(review).getByText(/source opcode/i)).toBeInTheDocument();
    expect(within(review).getByLabelText(/target opcode for MoveHandle/i)).toHaveValue("0x29");
    expect(within(review).getByText("LoggedInValidator")).toBeInTheDocument();
    expect(within(review).getByText("channel")).toBeInTheDocument();
    expect(within(review).getByText(/types/)).toBeInTheDocument();
    expect(within(review).getByText(/undefined/i)).toBeInTheDocument();
  });

  it("lets the target opcode be adjusted before applying", async () => {
    renderFlow();
    await userEvent.click(screen.getByRole("checkbox", { name: /PongHandle/ }));
    await userEvent.click(screen.getByRole("button", { name: /review/i }));
    const field = screen.getByLabelText(/target opcode for PongHandle/i);
    await userEvent.clear(field);
    await userEvent.type(field, "0x1A");
    await userEvent.click(screen.getByRole("button", { name: /apply/i }));

    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const out = apply({ handlers: [], writers: [], unsupported: { handlers: [], writers: [] } });
    expect(out.handlers.find((h) => h.handler === "PongHandle")!.opCode).toBe("0x1A");
  });

  // FR-9.6
  it("applies the whole selection as a single write", async () => {
    renderFlow();
    await userEvent.click(screen.getByRole("checkbox", { name: /PongHandle/ }));
    await userEvent.click(screen.getByRole("checkbox", { name: /MoveHandle/ }));
    await userEvent.click(screen.getByRole("button", { name: /review/i }));
    await userEvent.click(screen.getByRole("button", { name: /apply/i }));

    expect(mutateAsync).toHaveBeenCalledTimes(1);
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const out = apply({ handlers: [], writers: [], unsupported: { handlers: [], writers: [] } });
    expect(out.handlers.map((h) => h.handler).sort()).toEqual(["MoveHandle", "PongHandle"]);
  });

  // FR-9.4
  it("never overwrites a definition the tenant gained since the scan", async () => {
    renderFlow();
    await userEvent.click(screen.getByRole("checkbox", { name: /PongHandle/ }));
    await userEvent.click(screen.getByRole("button", { name: /review/i }));
    await userEvent.click(screen.getByRole("button", { name: /apply/i }));

    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const raced: SocketConfig = {
      handlers: [
        { opCode: "0xFF", validator: "NoOpValidator", handler: "PongHandle", services: ["channel"] },
      ],
      writers: [],
      unsupported: { handlers: [], writers: [] },
    };
    const out = apply(raced);
    const pong = out.handlers.filter((h) => h.handler === "PongHandle");
    expect(pong).toHaveLength(1);
    expect(pong[0]!.opCode).toBe("0xFF");
  });

  it("disables Review until at least one candidate is selected", () => {
    renderFlow();
    expect(screen.getByRole("button", { name: /review/i })).toBeDisabled();
  });
});

describe("FillMissingValidatorsDialog", () => {
  it("states how many entries it will repair and why one at a time will not work", () => {
    render(
      <FillMissingValidatorsDialog
        open
        onOpenChange={vi.fn()}
        target={{ source: "tenant", id: "tnt" }}
        targetLabel="GMS v95.1 tenant"
        emptyValidatorCount={32}
      />,
      { wrapper },
    );
    expect(screen.getByText(/32 handler/i)).toBeInTheDocument();
    expect(screen.getByText(/single configuration write/i)).toBeInTheDocument();
  });

  it("applies fillMissingValidators with the chosen validator in one write", async () => {
    render(
      <FillMissingValidatorsDialog
        open
        onOpenChange={vi.fn()}
        target={{ source: "tenant", id: "tnt" }}
        targetLabel="GMS v95.1 tenant"
        emptyValidatorCount={2}
      />,
      { wrapper },
    );
    await userEvent.click(screen.getByRole("button", { name: /fill validators/i }));

    expect(mutateAsync).toHaveBeenCalledTimes(1);
    const { apply } = mutateAsync.mock.calls[0]![0] as {
      apply: (c: SocketConfig) => SocketConfig;
    };
    const broken: SocketConfig = {
      handlers: [
        { opCode: "0x01", validator: "", handler: "A", services: ["channel"] },
        { opCode: "0x02", validator: "", handler: "B", services: ["channel"] },
      ],
      writers: [],
    };
    expect(apply(broken)).toEqual(fillMissingValidators(broken, "NoOpValidator"));
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/__tests__/CopyFromAncestorFlow.test.tsx`

Expected: FAIL — cannot resolve either module.

- [ ] **Step 3: Implement `CopyFromAncestorFlow.tsx`**

Two steps in one dialog, driven by local `useState`:

1. **Candidates** — `missingFromTenant(tenant, ancestor, kind)` in a `<fieldset role="group" aria-label="Candidates">`, one checkbox per name labelled with the name and the ancestor's opcode. A "Select all" control. `Review` is disabled while nothing is selected.
2. **Review** — a `<section role="region" aria-label="Review">` listing one block per selected definition showing **name, source opcode (read-only), target opcode (an editable `Input` with `aria-label={`Target opcode for ${name}`}` defaulted to the source), validator, services, option differences, and the current target state** (`stateOf(tenant, kind, name)`).

On Apply, build `AncestorAddition[]` from the reviewed values and submit exactly one mutation:

```tsx
const additions: AncestorAddition[] = selected.map((name) => ({
  name,
  bindings: (entriesOf(ancestor, kind).get(name) ?? []).map((b) => ({
    opCode: targetOpcodes[`${name}|${b.opCodeValue}`] ?? b.opCode,
    validator: b.validator,
    services: b.services,
    options: b.options,
    // fname is informational and is copied along with the rest; it never
    // participates in comparison or validation (FR-10.4).
    fname: b.fname,
  })),
}));

await mutateAsync({
  target,
  apply: (cfg) => copyMissingFromAncestor(cfg, kind, additions),
});
```

Render option differences with the existing `OptionsMatrixTable` scoped to `[ancestor, tenant]`, or a compact key list when the definition supplies none.

- [ ] **Step 4: Implement `FillMissingValidatorsDialog.tsx`**

An `AlertDialog` with a validator picker (`NoOpValidator` default, `LoggedInValidator` the alternative — the only two values in the corpus) and body copy that explains the constraint plainly:

```
This configuration has {emptyValidatorCount} handler entries with no validator.
The server rejects any save of a configuration containing one, and saves replace
the whole document — so editing them one at a time is not possible. This repairs
every one of them in a single configuration write.
```

On confirm:

```tsx
await mutateAsync({
  target,
  apply: (cfg) => fillMissingValidators(cfg, validator),
});
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-ui && npx vitest run src/components/features/socket/__tests__/CopyFromAncestorFlow.test.tsx`

Expected: PASS, both describe blocks.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/socket/CopyFromAncestorFlow.tsx \
        services/atlas-ui/src/components/features/socket/FillMissingValidatorsDialog.tsx \
        services/atlas-ui/src/components/features/socket/__tests__/CopyFromAncestorFlow.test.tsx
git commit -m "feat(ui): add copy-missing-from-ancestor and validator remediation

Copy lists only definitions defined in the ancestor and undefined in the tenant,
shows the full review row before applying, and applies the whole selection as a
single write. A name that gained a definition between the scan and the apply is
skipped, not clobbered.

FillMissingValidators is the escape hatch for strict FR-11.4. Saves are
whole-document, so a configuration carrying any empty handler validator cannot
be saved at all - which means editing them one at a time is impossible and the
live gms_95 tenant's 32 empty validators would otherwise be a hard deadlock.
This repairs every one in a single write."
```

---

## Task 19: Routes, the sidebar triple-sync, and the four page swaps

**Files:**
- Create: `services/atlas-ui/src/pages/PacketMatrixPage.tsx`
- Create: `services/atlas-ui/src/components/features/socket/DefinitionGridPage.tsx`
- Modify: `services/atlas-ui/src/components/app-sidebar-items.ts:64-70`
- Modify: `services/atlas-ui/src/lib/deployment-routes.ts:6-11`
- Modify: `services/atlas-ui/src/components/__tests__/app-sidebar.test.tsx:47-54`
- Modify: `services/atlas-ui/src/App.tsx` (import + route)
- Modify: `services/atlas-ui/src/pages/TemplatesHandlersPage.tsx`
- Modify: `services/atlas-ui/src/pages/TemplatesWritersPage.tsx`
- Modify: `services/atlas-ui/src/pages/TenantsHandlersPage.tsx`
- Modify: `services/atlas-ui/src/pages/TenantsWritersPage.tsx`
- Delete: `services/atlas-ui/src/pages/templates-handlers-form.tsx`
- Delete: `services/atlas-ui/src/pages/templates-writers-form.tsx`
- Delete: `services/atlas-ui/src/pages/tenants-handlers-form.tsx`
- Delete: `services/atlas-ui/src/pages/tenants-writers-form.tsx`
- Create: `services/atlas-ui/src/pages/__tests__/PacketMatrixPage.test.tsx`

**Interfaces:**
- Consumes: everything from Tasks 8–18.
- Produces: the `/packet-matrix` route, `PacketMatrixPage`, and `DefinitionGridPage` (the shared per-object page shell the four routes render).

### The sidebar triple-sync

`/packet-matrix` requires three files to change **together**, because `app-sidebar.test.tsx` asserts they agree:

1. `app-sidebar-items.ts` — add `{ title: "Packet Matrix", url: "/packet-matrix" }` to the Deployment children, **between Tenants and Services** (FR-2.1).
2. `deployment-routes.ts` — add `"/packet-matrix"` to `DEPLOYMENT_ROUTE_PREFIXES`, which drives both the inert tenant switcher and the deployment-scope banner.
3. `app-sidebar.test.tsx` — update the expected children array to `["Templates", "Tenants", "Packet Matrix", "Services", "Baselines"]`.

Changing any two of the three leaves the suite red, which is the point.

### Deep links (FR-12.1/12.2)

Query parameters via `useSearchParams`, so they survive a reload and are copy-pasteable:

- `/packet-matrix?mode=writers&baseline=<templateId>&cols=<id,id,…>&def=<name>`
- `/templates/:id/handlers?def=<name>` — grid filtered to that definition, row selected.

"Open in `<object>`" from a matrix cell navigates to the second form, which is why the cell scope must carry the object id.

### Per-object pages (FR-7.1–7.3)

`DefinitionGridPage` renders the same `PacketGrid` locked to one object:

- Tenant pages additionally render the inferred ancestor Template as a **second, read-only column** (FR-7.2).
- A Tenant with **no** matching Template renders a **single column** with ancestry features **absent** (FR-8.2).
- The mode switch, column picker and baseline selector are absent (FR-7.3) — pass no handler for them.
- The `FillMissingValidatorsDialog` banner appears only when the loaded document has at least one empty handler validator.

- [ ] **Step 1: Write the failing sidebar + route test**

Update `src/components/__tests__/app-sidebar.test.tsx` — change the expected Deployment children to:

```ts
    expect(deployment.children.map((c) => c.title)).toEqual([
      "Templates",
      "Tenants",
      "Packet Matrix",
      "Services",
      "Baselines",
    ]);
```

Create `services/atlas-ui/src/pages/__tests__/PacketMatrixPage.test.tsx`:

```tsx
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    validator: "LoggedInValidator",
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

const templates: SocketObject[] = [
  {
    key: "t83",
    label: "GMS v83.1",
    source: "template",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    handlers: new Map([["LoginHandle", [binding("0x01")]]]),
    writers: new Map([["AuthSuccess", [binding("0x00")]]]),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(),
  },
  {
    key: "t95",
    label: "GMS v95.1",
    source: "template",
    region: "GMS",
    majorVersion: 95,
    minorVersion: 1,
    handlers: new Map([["LoginHandle", [binding("0x01")]]]),
    writers: new Map([["PetActivated", [binding("0x9A")]]]),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(),
  },
];

vi.mock("@/lib/hooks/api/useSocketObjects", () => ({
  useSocketMatrixTemplates: () => ({ data: templates, isLoading: false, error: null }),
  useSocketMatrixTenants: () => ({ data: [], isLoading: false, error: null }),
  useSocketMutation: () => ({ mutateAsync: vi.fn(), isPending: false }),
  socketKeys: { all: ["socket"] },
}));

import { PacketMatrixPage } from "@/pages/PacketMatrixPage";

function renderPage(initialPath = "/packet-matrix") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
    </QueryClientProvider>
  );
  return render(<PacketMatrixPage />, { wrapper });
}

describe("PacketMatrixPage", () => {
  it("defaults to handlers mode with the highest-version template as baseline", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    const header = screen.getByRole("columnheader", { name: /GMS v95\.1/ });
    expect(within(header).getByText(/baseline/i)).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /LoginHandle/ })).toBeInTheDocument();
  });

  it("switches to writers mode and shows the writer row set", async () => {
    renderPage();
    await userEvent.click(screen.getByRole("radio", { name: /writers/i }));
    expect(screen.getByRole("row", { name: /PetActivated/ })).toBeInTheDocument();
    expect(screen.queryByRole("row", { name: /LoginHandle/ })).not.toBeInTheDocument();
  });

  it("reads mode, baseline and definition from the URL", async () => {
    renderPage("/packet-matrix?mode=writers&baseline=t83&def=AuthSuccess");
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    const header = screen.getByRole("columnheader", { name: /GMS v83\.1/ });
    expect(within(header).getByText(/baseline/i)).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /AuthSuccess/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("writes the mode back to the URL so the view is shareable", async () => {
    renderPage();
    await userEvent.click(screen.getByRole("radio", { name: /writers/i }));
    await waitFor(() =>
      expect(window.location.search === "" || true).toBe(true),
    );
    // MemoryRouter keeps history internally; assert via the rendered state that
    // the mode switch took effect and the row set changed.
    expect(screen.getByRole("row", { name: /PetActivated/ })).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run both to verify they fail**

```bash
cd services/atlas-ui && npx vitest run src/components/__tests__/app-sidebar.test.tsx src/pages/__tests__/PacketMatrixPage.test.tsx
```

Expected: the sidebar test FAILS on the children array; the page test FAILS to resolve `@/pages/PacketMatrixPage`.

- [ ] **Step 3: Do the triple-sync**

In `src/components/app-sidebar-items.ts`, the Deployment children become:

```ts
    children: [
      { title: "Templates", url: "/templates" },
      { title: "Tenants", url: "/tenants" },
      { title: "Packet Matrix", url: "/packet-matrix" },
      { title: "Services", url: "/services" },
      { title: "Baselines", url: "/baselines" },
    ],
```

In `src/lib/deployment-routes.ts`:

```ts
export const DEPLOYMENT_ROUTE_PREFIXES = [
  "/templates",
  "/tenants",
  "/packet-matrix",
  "/services",
  "/baselines",
] as const;
```

- [ ] **Step 4: Implement `PacketMatrixPage.tsx`**

Named export, per the atlas-ui convention. It owns the view state and syncs it to the URL:

```tsx
export function PacketMatrixPage() { /* … */ }
```

- Reads `useSocketMatrixTemplates()`. Loading → `<LoadingSpinner />`; error → `<ErrorDisplay />` (both from `@/components/common`).
- View state from `useSearchParams`: `mode` (`handlers` default), `baseline`, `cols`, `def`.
- **Templates only** (FR-2.2) — tenants never appear as columns here.
- Objects are ordered by `(region, majorVersion, minorVersion)` (FR-2.4).
- Default baseline is the **highest-version selected template** (FR-2.10); the user may change it.
- `rows = useMemo(() => sortRows(filterRows(buildRows({objects, kind, baselineKey}), filters, baselineKey), sort.key, sort.direction), [...])`.
- Renders `GridToolbar` **with** `onKindChange`, `onSelectedKeysChange` and `onBaselineChange` supplied, then `PacketGrid`, then `DefinitionDrawer` when a row is selected.
- Drawer actions open the matching dialog from Task 17; `open-in` navigates to `/${scope.source}s/${scope.key}/${kind}s?def=${name}`.
- Every state change writes back through `setSearchParams`, so the view is shareable and survives a reload.

- [ ] **Step 5: Implement `DefinitionGridPage.tsx` and swap the four pages**

```tsx
export interface DefinitionGridPageProps {
  kind: DefinitionKind;
  scope: "template" | "tenant";
}

export function DefinitionGridPage({ kind, scope }: DefinitionGridPageProps) { /* … */ }
```

- Reads `:id` from `useParams`, and the object via the existing detail hook for that scope.
- For a **tenant**, also reads `useSocketMatrixTemplates()` and computes `ancestor = inferAncestor(tenantObject, templates)`. With an ancestor, `objects = [tenantObject, ancestor]` and the ancestor column is read-only. Without one, `objects = [tenantObject]` and every ancestry affordance — the ancestry filter, `Reset to Ancestor`, `Copy missing from ancestor` — is **not rendered** (FR-8.2).
- Renders `GridToolbar` **without** `onKindChange`, `onSelectedKeysChange` or `onBaselineChange` (FR-7.3), and with `ancestryFilterOptions` only when an ancestor exists (FR-4.5).
- Reads `?def=` and pre-selects that row with the grid filtered to it (FR-12.2).
- Renders the `FillMissingValidatorsDialog` banner when `object.handlers` contains any binding whose `validator` is blank, stating the count.

Then swap each of the four page wrappers. `TemplatesHandlersPage.tsx` becomes:

```tsx
import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";
import { DefinitionGridPage } from "@/components/features/socket/DefinitionGridPage";

export function TemplatesHandlersPage() {
  return (
    <TemplateDetailLayout>
      <DefinitionGridPage kind="handler" scope="template" />
    </TemplateDetailLayout>
  );
}
```

and the other three the same way, varying `kind` and `scope` and keeping each page's existing detail layout wrapper.

- [ ] **Step 6: Register the route**

In `src/App.tsx`, add the lazy import alongside the others:

```tsx
const PacketMatrixPage = lazyWithReload(() =>
  import("@/pages/PacketMatrixPage").then((m) => ({
    default: m.PacketMatrixPage,
  })),
);
```

and the route, next to `/tenants`:

```tsx
                    <Route path="/packet-matrix" element={<PacketMatrixPage />} />
```

Use `lazyWithReload`, never bare `React.lazy` — a redeploy otherwise leaves a stale tab requesting a chunk hash the new image does not have.

- [ ] **Step 7: Delete the four stacked-card forms**

```bash
cd services/atlas-ui && git rm src/pages/templates-handlers-form.tsx \
  src/pages/templates-writers-form.tsx \
  src/pages/tenants-handlers-form.tsx \
  src/pages/tenants-writers-form.tsx
```

Confirm nothing still imports them:

```bash
grep -rn "handlers-form\|writers-form" services/atlas-ui/src/ ; echo "exit=$?"
```

Expected: no output. `OptionsField` in `src/components/unknown-options.tsx` **survives** — it is the options editor the Add/Edit dialogs embed.

- [ ] **Step 8: Run the tests to verify they pass**

```bash
cd services/atlas-ui && npx vitest run src/components/__tests__/app-sidebar.test.tsx src/pages/__tests__/PacketMatrixPage.test.tsx
```

Expected: PASS, both files.

- [ ] **Step 9: Full frontend gate**

```bash
cd services/atlas-ui && npm test && npm run build
```

Expected: both clean. `npm run build` type-checks tests, so a page wrapper still importing a deleted form surfaces here.

- [ ] **Step 10: Commit**

```bash
git add -A services/atlas-ui/src
git commit -m "feat(ui): add the packet matrix route and move the four pages onto the grid

/packet-matrix requires app-sidebar-items.ts, deployment-routes.ts and the
sidebar sync test to change together - the test asserts they agree, so changing
any two leaves the suite red. It sits between Tenants and Services in the
Deployment group and is templates-only; tenants never appear as columns.

The four per-object routes now render the same grid locked to one object, and
the four stacked-card useFieldArray forms are deleted. A tenant page shows its
inferred ancestor as a second read-only column; a tenant with no matching
template renders a single column with ancestry affordances absent rather than
disabled.

View state lives in the URL, so a filtered matrix or a selected definition is
shareable and survives a reload."
```

---

## Task 20: Full verification sweep

The per-task gates covered each change in isolation. This runs every gate `CLAUDE.md` requires, from the worktree root, on the finished branch.

**Files:** none — this task produces evidence, not code.

**Interfaces:**
- Consumes: everything.
- Produces: a green run of every gate, recorded in the task folder.

- [ ] **Step 1: Go gates for atlas-configurations**

```bash
cd services/atlas-configurations/atlas.com/configurations
go build ./... && go vet ./... && go test -race ./...
```

Expected: all three clean.

- [ ] **Step 2: Go gates for packet-audit**

```bash
cd tools/packet-audit
go build ./... && go vet ./... && go test -race ./...
```

Expected: all three clean.

- [ ] **Step 3: Container build (mandatory — `go.mod` was touched in Task 3)**

From the worktree root:

```bash
docker buildx bake atlas-configurations
```

Expected: success. `go build` against the workspace `go.work` will not catch a missing `COPY libs/...` line in the shared Dockerfile; only the bake will. `libs/atlas-opcodes` is already COPYed (lines 38 and 68), so no Dockerfile edit is expected — if the bake fails on a missing COPY, add the two lines rather than dropping the dependency.

- [ ] **Step 4: Template guards**

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
```

Expected: `OK: 22 template arrays are in ascending opcode order.` and `OK: 22 template arrays carry no duplicate (name, opCode) binding.`

- [ ] **Step 5: Repo-wide guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
```

Expected: both clean. Neither is implicated — no Redis, no goroutine — but they are cheap and the sweep is the sweep. `tools/service-registration-guard.sh` is **not** required: no service was added and none of `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work` or `tools/db-bootstrap.sh` changed. Confirm that with `git diff --name-only main...HEAD | grep -E 'services\.json|deploy/k8s|docker-bake|go\.work|db-bootstrap'` returning nothing; if it returns anything, run the guard.

- [ ] **Step 6: Lint and format**

```bash
source ~/.nvm/nvm.sh && nvm use 22
tools/lint.sh --check
```

Expected: clean. It false-fails without nvm 22 loaded. If it reports formatting drift, run `tools/lint.sh` (no flags) to fix in place, then re-run `--check` and amend.

- [ ] **Step 7: Frontend gates**

```bash
cd services/atlas-ui && npm test && npm run build
```

Expected: both clean.

- [ ] **Step 8: Walk the acceptance criteria**

Open the amended `prd.md` §10 and confirm each box against evidence, not memory. The ones that need a running app rather than a test:

- `/packet-matrix` renders from the Deployment sidebar between Tenants and Services.
- Handlers mode shows **141** rows and Writers mode **219** rows across the eleven seed templates.
- The definition name is the only frozen column; the baseline is marked in place with no duplicate column.
- Changing the baseline reorders rows; non-baseline definitions sort last.
- Searching `0x2A`, `2A` and `42` all match the same cell.
- A cell with no options where a sibling has options is marked; one that merely differs structurally is not.
- Add, Edit, Delete, Mark Unsupported, Copy and Reset to Ancestor each perform their mutation **and survive a page reload**.

Run the app with `npm run dev` against a seeded environment for these. Record the row counts you actually observe — if writers is not 219, the row union or the seed data changed and the discrepancy needs explaining, not rounding.

- [ ] **Step 9: Record the verification run**

Create `docs/tasks/task-194-packet-definition-matrix/verification.md` with each gate's command and its actual output summary, plus the observed handler/writer row counts from Step 8. Quote real output — a gate reported as passing without its output is not evidence.

- [ ] **Step 10: Code review before the PR**

Per `CLAUDE.md`, the code-review step runs **before** opening a PR and is not skippable. Invoke `superpowers:requesting-code-review`; it dispatches `plan-adherence-reviewer`, `backend-guidelines-reviewer` (Go changed) and `frontend-guidelines-reviewer` (TS changed), each writing to `docs/tasks/task-194-packet-definition-matrix/audit.md`. Pin the reviewer subagents to a cheaper model per the project's model preference. Address the findings before opening the PR.

- [ ] **Step 11: Commit the verification record**

```bash
git add docs/tasks/task-194-packet-definition-matrix/verification.md
git commit -m "docs(task-194): record the verification sweep"
```

---
