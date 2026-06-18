# CField Map/Field Packet Family — Byte-Plumbing Batch 2 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drive every applicable `CField*` map/field coverage-matrix cell to ✅ **verified** (or ⬜ **n/a** with IDB-evidenced version-absence) across gms_v83/v84/v87/v95 + jms_v185 — for all 75 ops (45 core `CField::` + 30 `CField_*` minigame) — by triaging each op verify-vs-implement, linking/adding a byte-exact codec, wiring it into `atlas-channel` + the five seed templates, and pinning tier-1 evidence.

**Architecture:** Same four-layer recipe as task-092, codified in `docs/packets/IMPLEMENTING_A_PACKET.md` / `docs/packets/VERIFYING_A_PACKET.md` — task-096 **follows** those docs, does not re-author them. Three stages: **Stage 0 (triage)** produces the committed A/B/C classification artifact (`structures/triage.md`) plus the chat-relocation (D3), resolving every C-row fname against the IDB *before* any codec is written — the load-bearing anti-duplication gate. **Stage 1 (IDA-bound)** harvests the byte layout of every B-row / unverified-A-row op from each version IDB — one IDB at a time — into per-version `structures/*.md` notes and fixes registry gaps. **Stage 2 (pure-Go)** transcribes each layout into an immutable codec + round-trip/golden test + verify marker + pinned evidence, registers the writer/handler in `atlas-channel`, and routes the per-version opcode in all five seed templates, cluster by cluster. The matrix grader (`tools/packet-audit`) is the burndown gate.

**Tech Stack:** Go 1.2x, `libs/atlas-packet` (codec, under `field/` — a tier-1 prefix), `libs/atlas-socket` (Reader/Writer), `libs/atlas-tenant` (version from ctx), `tools/packet-audit` (matrix/evidence), atlas-configurations seed-template JSON, IDA-Pro MCP (Stage 1 only).

**Read first:** `docs/tasks/task-096-cfield-packet-family/context.md` — it holds the wiring sites, the test-harness API, the registry/export mechanics, the IDA ports, and the packet-audit command surface. Task steps reference it by section (e.g. `context.md §1`) instead of repeating data. Also read the design (`design.md`) decisions D1–D4 and §5 (chat relocation) and §6 (cluster sequencing).

**Why codec bodies are not pre-written here:** CLAUDE.md forbids citing packet bytes from memory — the field order of an op is unknown until its IDB function is decompiled. Stage 1 produces that field order as a concrete artifact (`structures/<version>.md#<OP>`). Stage-2 codec steps transcribe from that artifact. This is a deliberate three-stage structure, not a placeholder: every non-byte detail (paths, names, markers, commands, template JSON shape, classification) is fully specified below.

---

## Conventions used by every Stage-2 op

Four reusable recipes. Each op task names which recipe(s) it uses and supplies its own data; the full code lives here so no task is "similar to" another. All four are the task-092 recipes, with `<pkg>` = `field` for this family.

### Recipe R-CB — new clientbound codec (server→client)

**Files:** Create `libs/atlas-packet/field/clientbound/<op>.go`; create `libs/atlas-packet/field/clientbound/<op>_test.go`; modify `services/atlas-channel/atlas.com/channel/main.go` (`produceWriters`); create `services/atlas-channel/atlas.com/channel/socket/writer/<op>.go`; modify the 5 templates.

- [ ] **R-CB.1 — Write the failing test.** In `<op>_test.go`, golden-byte for v83 + round-trip across all variants. Field values are illustrative; byte expectations come from `structures/gms_v83.md#<OP>`.

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// (markers added in R-CB.5, not yet)
func Test<Op>(t *testing.T) {
	input := New<Op>(/* concrete fixture args */)
	// golden byte check (v83 baseline) — bytes transcribed from structures/gms_v83.md#<OP>
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{ /* per-field bytes with // comments citing the decompile line */ }
	if !bytes.Equal(got, want) {
		t.Fatalf("<OP> layout mismatch\n got % x\nwant % x", got, want)
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
```

- [ ] **R-CB.2 — Run, expect FAIL** (`New<Op>` undefined): `go test ./libs/atlas-packet/field/clientbound/ -run Test<Op>`.
- [ ] **R-CB.3 — Write the codec** `<op>.go`, modeled on `field/clientbound/clock.go` / `field/clientbound/set_field.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const <Op>Writer = "<WriterName>"   // see op data table

type <Op> struct { /* private fields, types per structures/<version>.md */ }
func New<Op>(/* args */) <Op> { return <Op>{ /* … */ } }
/* getters … */
func (m <Op>) Operation() string { return <Op>Writer }
func (m <Op>) String() string { return fmt.Sprintf("…", /* fields */) }

func (m <Op>) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		_ = t // drop if no version branch
		// fields in client read-order from structures/<version>.md#<OP>;
		// version-branch with t.MajorAtLeast(87) / t.Region()=="JMS" — NEVER >83
		return w.Bytes()
	}
}

func (m *<Op>) Decode(l logrus.FieldLogger, ctx context.Context) func(*request.Reader, map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		_ = t
		// exact mirror of Encode read-order
	}
}
```

- [ ] **R-CB.4 — Run, expect PASS:** `go test ./libs/atlas-packet/field/clientbound/ -run Test<Op>`.
- [ ] **R-CB.5 — Add verify markers** above `Test<Op>`, one per applicable version, `ida=` = the export address for the op's fname (from `structures/<version>.md#<OP>`):

```go
// packet-audit:verify packet=field/clientbound/<Op> version=gms_v83 ida=0x<addr>
// packet-audit:verify packet=field/clientbound/<Op> version=gms_v84 ida=0x<addr>
// packet-audit:verify packet=field/clientbound/<Op> version=gms_v87 ida=0x<addr>
// packet-audit:verify packet=field/clientbound/<Op> version=gms_v95 ida=0x<addr>
// packet-audit:verify packet=field/clientbound/<Op> version=jms_v185 ida=0x<addr>
```

(Only the versions marked "implement" in `structures/applicability.md` get a marker; inapplicable versions get a VERSION-ABSENT evidence pin instead — see R-CB.6 / Cluster-tail note.)

- [ ] **R-CB.6 — Pin evidence**, once per applicable version, then add the `verifies:` list to each generated YAML:

```bash
for V in gms_v83 gms_v84 gms_v87 gms_v95 jms_v185; do \
  go run ./tools/packet-audit evidence pin --packet field/clientbound/<Op> --version $V \
    --ida "<fname>" --category TIER1-FIXTURE ; done
```

(If `pin` reports "function … not in export", STOP and ESCALATE to the user — that version's export lacks the fname. Do not fabricate the hash, auto-re-export, or silently substitute a fname; present it and wait for a decision per context.md §5 and memory `feedback_unresolved_fname_escalate`.)

- [ ] **R-CB.7 — Register the writer.** Add `fieldcb.<Op>Writer` to `produceWriters()` (main.go:594, sorted near sibling writers). Create `socket/writer/<op>.go`:

```go
package writer

import (
	"context"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

func <Op>Body(/* domain args matching New<Op> */) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return fieldcb.New<Op>(/* args */).Encode(l, ctx)(options)
		}
	}
}
```

- [ ] **R-CB.8 — Route in all applicable templates.** Insert a `socket.writers[]` entry `{ "opCode": "0x<hex>", "writer": "<WriterName>" }` in sorted opcode position in each applicable `template_{gms_83,gms_84,gms_87,gms_95}_1.json` + `template_jms_185_1.json`, using the per-version opcode read into `structures/<version>.md` (Stage 1). Skip a template where the op is VERSION-ABSENT.
- [ ] **R-CB.9 — Regenerate + check:** `go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check`; confirm the op's row flipped to ✅ for every applicable version, ⬜ for VERSION-ABSENT, 0 conflicts.
- [ ] **R-CB.10 — Commit** (`go test ./... -race` + `go vet ./...` for libs/atlas-packet + atlas-channel first; verify cwd is the worktree and branch is `task-096-cfield-packet-family` after):

```bash
git add libs/atlas-packet/field/ services/atlas-channel/ services/atlas-configurations/ docs/packets/
git commit -m "feat(task-096): <OP> codec + wiring + verified"
```

### Recipe R-SB — new serverbound codec (client→server)

Identical to R-CB except:
- Codec lives under `field/serverbound/`; const is `<Op>Handle = "<HandlerName>"`; package import alias `fieldsb`.
- No writer Body. Instead, register the handler: add `hm[fieldsb.<Op>Handle] = handler.<Op>HandleFunc` to `produceHandlers()` (main.go:721) and create `socket/handler/<op>.go`:

```go
package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func <Op>HandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) {
	return func(s session.Model, r *request.Reader, ro map[string]interface{}) {
		p := fieldsb.<Op>{}
		p.Decode(l, ctx)(r, ro)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		// behavior: deferred (decode-and-log only — see design D2)
	}
}
```

- Template entry goes in `socket.handlers[]` as `{ "opCode": "0x<hex>", "validator": "LoggedInValidator", "handler": "<HandlerName>" }` (NoOpValidator only for connection-level ops — none expected here). A validator-less handler entry is silently dropped by `BuildHandlerMap` — mandatory, not cosmetic.
- The round-trip test still drives both `Encode` and `Decode` (serverbound models implement both for testability; `Encode` mirrors the client write so RoundTrip closes).

### Recipe R-MARK — verify-only (already-implemented codec, A-row)

For an op whose codec already exists and is correct (e.g. an existing `field/clientbound/*` codec, or a relocated chat codec after Cluster 1): skip codec/wiring creation; do only the linking + verification steps:
- [ ] Confirm the registry op's `fname` maps to the existing struct so `candidatesFromFName` links it (add a thin wrapper only if the registry op uses a shared model with a different name — see R-WRAP).
- [ ] R-CB.5 (markers, one per applicable version) on the existing `*_test.go` if absent.
- [ ] If the existing test lacks a v83 golden-byte assertion for a mask/mode-driven packet, add one (transcribed from `structures/gms_v83.md#<OP>`) before marking.
- [ ] R-CB.6 (evidence pin per applicable version).
- [ ] Ensure each applicable seed template routes the op (R-CB.8 for CB / R-SB template entry for SB) — a missing route on an implemented op is a `conflict`, not a no-op.
- [ ] R-CB.9 (regenerate/check), R-CB.10 (commit).

Confirm in Stage 1 that all applicable per-version layouts equal the existing codec's; if any version differs, that wire-fix is its own commit before marking.

### Recipe R-WRAP — thin wrapper for a shared-model op

When a registry op is served by a shared sub-struct/model rather than a dedicated codec (the task-085 shared-codec pattern, memory `bug_matrix_redx_unverified_shared_codec`): do NOT duplicate the model. Create a thin codec in `field/<dir>/<op>.go` that embeds/delegates to the shared model and exposes `Operation()` = the op's wiring NAME, so `candidatesFromFName` links the op to the wrapper. Then verify it via R-MARK steps. Record the wrapper decision in `structures/triage.md` for that op.

---

## Stage 0 — Triage + chat relocation (the anti-duplication gate, design §3/§5)

> This stage is **design D4**: triage is a committed artifact produced *before* any codec work. No op is implemented without a triage row, and no duplicate codec is created for an A-row — both are acceptance criteria.

### Task 0.1: Capture baseline matrix state

**Files:** Create `structures/baseline.md`.

- [ ] **Step 1:** From the worktree root, run `go run ./tools/packet-audit matrix --check; echo "exit=$?"`. Expected exit 0; if non-zero, record the pre-existing failures in `structures/baseline.md` so they're not attributed to this task.
- [ ] **Step 2:** Record the current ❌/✅/⬜ state of all 75 ops by grepping `docs/packets/audits/STATUS.md` into `structures/baseline.md` (one row per op × version). This is the burndown reference.
- [ ] **Step 3:** Commit: `git add docs/tasks/task-096-cfield-packet-family/structures/baseline.md && git commit -m "chore(task-096): record matrix baseline"`.

### Task 0.2: Codec inventory (what already exists)

**Files:** Create `structures/codec-inventory.md`.

- [ ] **Step 1:** Enumerate existing codecs that could serve a work-list op:
  `find libs/atlas-packet/field libs/atlas-packet/chat -name '*.go' -not -name '*_test.go'`. For each
  file, record its struct name, `Operation()` NAME constant, direction, and whether it has a
  `// packet-audit:verify` marker + an evidence record (`ls docs/packets/evidence/*/field.* docs/packets/evidence/*/chat.*`).
- [ ] **Step 2:** For each of the 75 ops in `structures/cfield-ops.md`, note any existing codec that
  plausibly serves it (by fname / NAME / byte-shape). Mark `PKT` ops from the work-list against
  their files.
- [ ] **Step 3:** Commit `structures/codec-inventory.md`.

### Task 0.3: Resolve C-rows against the IDB (registry corrections)

> The C-rows (design §3) must be resolved to a real op-name before they can be classified A/B. This is IDA-bound — uses the harvest workflow (context.md §4), one IDB at a time. Resolve on **v83** first (the baseline); confirm per-version presence in Stage 1.

**Files:** Modify `docs/packets/registry/<version>.yaml` (fname/op corrections); notes into `structures/triage.md` (C-row section).

- [ ] **Step 1:** `select_instance(13337)` (v83); confirm with `list_instances`.
- [ ] **Step 2: Foothold/stalk cluster.** Decompile `CField::OnStalkResult`, `CField::OnFootHoldInfo`,
  `CField::OnRequestFootHoldInfo`, `CField::OnHontailTimer`. For each `IDA_0X098/09C/09D/0A4/0AA/0AC/0B0/0B1/0X169`
  row, decide: (a) alias of a named CField op already in the list → collapse into it (note the
  mapping); (b) a distinct op → assign a real op-name + provenance `ida-discovered`; (c) genuinely
  version-absent → mark ⬜ candidate with IDB evidence. **No `IDA_0x…` placeholder reaches code.**
- [ ] **Step 3: MTS.** Decompile `CField::OnCharacterSale`. Decide whether `MTS_OPERATION` and
  `MTS_OPERATION2` are two **modes** of one structure (one model, mode-branched, two registry ops →
  two routes) or two **distinct** packets (two models). Record the verdict in `structures/triage.md`.
- [ ] **Step 4: Door / guild.** Decompile `CField::TryEnterTownPortal` (USE_DOOR) and
  `CField::InputGuildName` (GUILD_OPERATION). Resolve direction (the worklist marks both `?`) and
  whether an existing codec already covers GUILD_OPERATION (`PKT` → likely A).
- [ ] **Step 5: Minigame `?` rows.** For `SNOWBALL`/`LEFT_KNOCKBACK`/`COCONUT`/`GUILD_BOSS`/`CONTI_MOVE`
  reached via `Update`/`BasicActionAttack`/`Init`: those fnames are state methods, not send/recv
  sites. Derive the real send-site fname (or confirm serverbound recv) and record it.
- [ ] **Step 6: WHISPER dupe.** Determine whether the two `WHISPER`/`OnWhisper` rows are distinct
  registry rows (mode/reply) or a list dupe; record the resolution.
- [ ] **Step 7:** Patch `docs/packets/registry/gms_v83.yaml` for every resolved row (`provenance:
  manual`/`ida-discovered`, IDA citation in `note`). Commit:
  `git commit -m "fix(task-096): resolve CField C-row fnames in v83 registry"`.

> If any fname cannot be resolved from the v83 IDB (true hard blocker — SMC-only/undecompilable),
> STOP and ESCALATE to the user (memory `feedback_unresolved_fname_escalate`); do not defer a
> producible name. Record the escalation outcome in `structures/triage.md`.

### Task 0.4: Write the committed triage table (D4)

**Files:** Create `structures/triage.md`.

- [ ] **Step 1:** Produce one row per op (all 75), columns: `op | direction (CB/SB) | owner class |
  existing codec path (if any) | classification A/B/C-resolved | resolution note`. Source the
  classification from `structures/codec-inventory.md` (0.2) + the C-row resolutions (0.3).
  - **A** = codec exists and is correct → R-MARK (or R-WRAP for shared-model). No new codec.
  - **B** = no codec → R-CB / R-SB.
  - **C** = was unresolved; now resolved to A or B (no row stays "C").
- [ ] **Step 2:** For the chat-relocation candidates (GENERAL_CHAT, MULTICHAT, WHISPER), record per
  file whether **every** registry op it serves is `CField::`-owned (move) or it is shared with a
  non-CField op (link-in-place exception). This drives Cluster 1.
- [ ] **Step 3:** For every op a version is expected to be absent, note the VERSION-ABSENT candidate
  with the IDB-evidence basis (confirmed in Stage 1).
- [ ] **Step 4:** Commit `structures/triage.md`. **This is the work-list every Stage-2 cluster consumes.**

---

## Stage 1 — IDA derivation harvest (one IDB at a time)

> **Operator note:** Each task requires the matching IDB loaded. The user switches the active IDB; `select_instance` is shared global state — do ALL of a version's derivations before asking to switch (memory `reference_ida_harvest_subagents`). Ports: v83=13337, v84=13341, v87=13338, v95=13339, jms=13340. jms = `*_U_DEVM` build only.

Each Stage-1 task produces `structures/<version>.md` with, per applicable op (every B-row + every A-row whose existing codec lacks an evidence-pinned layout for that version): the demangled fname, the export address, the registry opcode (decimal→hex), and the ordered field list (`name : width : note`) in client read-order, including version guards and loop bounds. It also patches `docs/packets/registry/<version>.yaml` for that version's remaining gaps.

> **Scope-narrowing:** A-rows whose existing codec is already ✅ for a version (per `structures/baseline.md`) need no re-derivation for that version — they are link-only. Stage 1 focuses on B-rows and the unverified cells of A-rows. Use `structures/triage.md` + `structures/baseline.md` to compute the per-version derivation list before starting each version.

### Task 1.A: Harvest gms_v83 (port 13337)

**Files:** Create `structures/gms_v83.md`; modify `docs/packets/registry/gms_v83.yaml`.

- [ ] **Step 1:** `select_instance(13337)`; confirm v83 with `list_instances`.
- [ ] **Step 2:** For each in-scope op present in `gms_v83.yaml`, `decompile` its `fname`, descending
  into helper read/write subs. Record the ordered field list + export address + opcode into
  `structures/gms_v83.md#<OP>`. For serverbound ops, read the opcode from the yaml.
- [ ] **Step 3:** Apply any remaining v83 registry fname corrections surfaced here (beyond the
  Task-0.3 C-rows): confirm the correct fname against the decompile; fix the yaml row
  (`provenance: manual`, IDA citation in `note`).
- [ ] **Step 4:** Note the v84≡v83 expectation for confirmation in Task 1.B. Commit
  `structures/gms_v83.md` + yaml edits: `git commit -m "harvest(task-096): v83 CField layouts"`.

### Task 1.B: Harvest gms_v84 (port 13341)

**Files:** Create `structures/gms_v84.md`; modify `docs/packets/registry/gms_v84.yaml`.

- [ ] **Step 1:** `select_instance(13341)`; confirm v84.
- [ ] **Step 2:** For each in-scope op, confirm the layout matches v83 (expected byte-identical below
  the shifted opcode table — memory `bug_v84_opcode_table_shifted_vs_v83`). Record only the
  **deltas vs v83** + per-op export addresses + the (shifted) v84 opcodes in `structures/gms_v84.md`.
  Flag any op whose body genuinely diverges from v83. Do **not** invent deltas the IDB doesn't show.
- [ ] **Step 3:** Fix v84 fname/opcode mislabels in the yaml as in 1.A.3. Commit.

### Task 1.C: Harvest gms_v87 (port 13338)

**Files:** Create `structures/gms_v87.md`; modify `docs/packets/registry/gms_v87.yaml`.

- [ ] **Step 1:** `select_instance(13338)`; confirm v87.
- [ ] **Step 2:** Derive each in-scope op; record full field list + addresses + opcodes; capture the
  v87+ structural additions (the `MajorAtLeast(87)` branch fields).
- [ ] **Step 3:** Fix any v87 fname mislabels in the yaml. Commit.

### Task 1.D: Harvest gms_v95 (port 13339)

**Files:** Create `structures/gms_v95.md`; modify `docs/packets/registry/gms_v95.yaml`.

- [ ] **Step 1:** `select_instance(13339)`; confirm v95.
- [ ] **Step 2:** Derive each in-scope op; record field lists + addresses + opcodes, including any
  v95-only structural tail.
- [ ] **Step 3:** Add/fix registry rows for any v95-only ops surfaced (`provenance: ida-discovered`);
  if an op is absent from the v95 IDB, document non-existence in `structures/gms_v95.md` as a
  VERSION-ABSENT basis. Commit.

### Task 1.E: Harvest jms_v185 (port 13340)

**Files:** Create `structures/jms_v185.md`; modify `docs/packets/registry/jms_v185.yaml`.

- [ ] **Step 1:** `select_instance(13340)`; confirm jms (the `*_U_DEVM` build).
- [ ] **Step 2:** Derive each applicable op; record field lists + addresses + opcodes + region deltas.
  Several minigames + GMS-event ops are expected JMS-absent — confirm against the dispatcher and
  mark VERSION-ABSENT with the IDB evidence.
- [ ] **Step 3:** Fix any jms fname gaps. Commit.

### Task 1.F: Reconcile applicability matrix

**Files:** Create `structures/applicability.md`.

- [ ] **Step 1:** From the 5 structures docs + registries + `structures/triage.md`, build the
  authoritative (op × version) grid: `implement` / `n-a(VERSION-ABSENT)` / `link-only(already-✅)`.
  This drives which marker/evidence lines each Stage-2 op needs.
- [ ] **Step 2:** For every `n-a` cell, record the `VERSION-ABSENT` justification (IDB dispatcher-
  absence evidence). Never reclassify a live cell to ⬜ to "close" it. Commit.

---

## Stage 2 — Codec + wiring + verification (pure Go)

Cluster order per design §6: 1 (chat relocation) → 2 (transfer/obstacle/quest/clock) → 3 (boss/admin/MTS/door/guild) → 4 (foothold/stalk C-cluster) → 5 (minigames). Each op = one task using R-CB / R-SB / R-MARK / R-WRAP with the data from `structures/triage.md` + `structures/<version>.md` + `structures/applicability.md`. **After each cluster**, run the full module gates (`go test -race`, `go vet`) and `matrix --check`, and confirm the cluster's cells flipped ✅/⬜ with 0 conflicts.

> The op→recipe→struct/NAME/fname mapping below is the **scaffold**; the **authoritative** per-op
> classification and per-version applicability come from `structures/triage.md` (Stage 0) and
> `structures/applicability.md` (Stage 1). Where triage marks an op A, use R-MARK/R-WRAP even if the
> table below lists R-CB/R-SB (the table assumes worst-case implement-new). NAME constants are
> `<Struct>Writer`/`<Struct>Handle` = the struct name (matching the existing `field/` convention).

### Cluster 1 — Chat relocation + new chat ops (design §5)

> Move-not-rewrite, executed before new derivation so regressions are isolated and bisectable. Per
> `structures/triage.md` Task-0.4 Step 2, a codec moves **iff every registry op it serves is
> `CField::`-owned**; a shared file is linked-in-place, not moved.

#### Task 2.1.1: Relocate GENERAL_CHAT (serverbound)

**Files:** `git mv libs/atlas-packet/chat/serverbound/general.go libs/atlas-packet/field/serverbound/general.go` (+ its `_test.go`); modify importers in `services/atlas-channel/atlas.com/channel/`.

- [ ] **Step 1:** Confirm in `structures/triage.md` that `chat/serverbound/general.go` serves **only**
  `CField::SendChatMsg`-owned ops. If shared, STOP — link-in-place instead (skip the move; do R-MARK).
- [ ] **Step 2:** `git mv` the `.go` and `_test.go` together. Change the `package` clause to the
  destination package. Keep the `Operation()` NAME constant byte-identical (the wiring key must not change).
- [ ] **Step 3:** Repoint importers: change `chatSB.<X>` → `fieldsb.<X>` in `main.go`
  (`produceHandlers`) and any `socket/handler/*.go`. Preserve existing `// packet-audit:verify`
  markers + evidence references in the moved test.
- [ ] **Step 4:** `go build ./...` + `go test -race ./...` for `libs/atlas-packet` and
  `services/atlas-channel/atlas.com/channel` — must stay green.
- [ ] **Step 5:** `go run ./tools/packet-audit matrix` — any cell previously ✅ via the `chat/…` path
  must remain ✅ via the `field/…` path. Update the evidence record's `verifies:` packet id to the
  new dotted path (`field.serverbound.General…`) if the matrix flags a drop. A drop-to-❌ is a move
  regression to fix in this commit, never an accepted loss.
- [ ] **Step 6:** `matrix --check` exit 0. Commit:
  `git commit -m "refactor(task-096): relocate GENERAL_CHAT codec chat→field"`.

#### Task 2.1.2: Relocate MULTICHAT (clientbound)

**Files:** `git mv libs/atlas-packet/chat/clientbound/multi.go → field/clientbound/multi.go` (+ `_test.go`); importers.

- [ ] Same six steps as Task 2.1.1, for `chat/clientbound/multi.go` (MULTICHAT, `CField::OnGroupMessage`),
  repointing `chatCB.<X>` → `fieldcb.<X>`. Confirm CField-only ownership first.

#### Task 2.1.3: Relocate WHISPER (clientbound)

**Files:** `git mv libs/atlas-packet/chat/clientbound/whisper.go → field/clientbound/whisper.go` (+ `_test.go`); importers.

- [ ] Same six steps, for `chat/clientbound/whisper.go` (WHISPER, `CField::OnWhisper`), repointing
  `chatCB.<X>` → `fieldcb.<X>`. Resolve the WHISPER-dupe finding from Task 0.3 Step 6 here (one
  codec, one or two routes as resolved).

#### Task 2.1.4: SPOUSE_CHAT (clientbound, new — R-CB)

| Op | Recipe | pkg / Struct | NAME const | fname |
|---|---|---|---|---|
| SPOUSE_CHAT | R-CB | `field/clientbound/SpouseChat` | `SpouseChatWriter="SpouseChat"` | CField::OnCoupleMessage |

- [ ] Execute R-CB.1–R-CB.10. Applicability per `structures/applicability.md` (work-list shows ❌4 —
  one version likely VERSION-ABSENT).

#### Task 2.1.5: ADMIN_CHAT / ADMIN_COMMAND / ADMIN_LOG / MATCH_TABLE / SLIDE_REQUEST / SUE_CHARACTER (serverbound slash family — R-SB)

> These six serverbound ops all share `CField::SendChatMsgSlash`. Task 0.3 + Stage 1 determine
> whether they are one mode-dispatched structure (one codec, mode field, N routes) or distinct
> structures. Model per the recorded verdict.

| Op | Recipe | pkg / Struct | NAME const | fname |
|---|---|---|---|---|
| ADMIN_CHAT | R-SB / R-WRAP | `field/serverbound/AdminChat` (or shared SlashCommand) | `AdminChatHandle="AdminChat"` | CField::SendChatMsgSlash |
| ADMIN_COMMAND | R-SB / R-WRAP | `field/serverbound/AdminCommand` | `AdminCommandHandle="AdminCommand"` | CField::SendChatMsgSlash |
| ADMIN_LOG | R-SB / R-WRAP | `field/serverbound/AdminLog` | `AdminLogHandle="AdminLog"` | CField::SendChatMsgSlash |
| MATCH_TABLE | R-SB / R-WRAP | `field/serverbound/MatchTable` | `MatchTableHandle="MatchTable"` | CField::SendChatMsgSlash |
| SLIDE_REQUEST | R-SB / R-WRAP | `field/serverbound/SlideRequest` | `SlideRequestHandle="SlideRequest"` | CField::SendChatMsgSlash |
| SUE_CHARACTER | R-SB / R-WRAP | `field/serverbound/SueCharacter` | `SueCharacterHandle="SueCharacter"` | CField::SendChatMsgSlash |

- [ ] One task per op, R-SB (or R-WRAP if Stage 1 confirms a single shared `SendChatMsgSlash`
  structure → one model + per-op thin wrappers). Applicability per `structures/applicability.md`.
  `SUE_CHARACTER_RESULT` (`CWvsContext::OnSueCharacterResult`) is **not** in scope and **not** touched.

### Cluster 2 — Core CField: transfer / obstacle / quest / clock / GM events (≈20 ops)

> Mostly fixed-width scalars; expect a high A-row count (many existing `field/clientbound/*` codecs).
> Use `structures/triage.md` to pick R-MARK (existing) vs R-CB (new) per op.

| Op | Dir | pkg / Struct | NAME const | fname |
|---|---|---|---|---|
| BLOCKED_MAP | CB | `field/clientbound/BlockedMap` | `BlockedMapWriter="BlockedMap"` | CField::OnTransferFieldReqIgnored |
| BLOCKED_SERVER | CB | `field/clientbound/BlockedServer` | `BlockedServerWriter="BlockedServer"` | CField::OnTransferChannelReqIgnored |
| FIELD_OBSTACLE_ALL_RESET | CB | `field/clientbound/FieldObstacleAllReset` | `FieldObstacleAllResetWriter="FieldObstacleAllReset"` | CField::OnFieldObstacleAllReset |
| FIELD_OBSTACLE_ONOFF | CB | `field/clientbound/FieldObstacleOnOff` | `FieldObstacleOnOffWriter="FieldObstacleOnOff"` | CField::OnFieldObstacleOnOff |
| FIELD_OBSTACLE_ONOFF_LIST | CB | `field/clientbound/FieldObstacleOnOffList` | `FieldObstacleOnOffListWriter="FieldObstacleOnOffList"` | CField::OnFieldObstacleOnOffStatus |
| SET_OBJECT_STATE | CB | `field/clientbound/SetObjectState` | `SetObjectStateWriter="SetObjectState"` | CField::OnSetObjectState |
| SET_QUEST_CLEAR | CB | `field/clientbound/SetQuestClear` | `SetQuestClearWriter="SetQuestClear"` | CField::OnSetQuestClear |
| SET_QUEST_TIME | CB | `field/clientbound/SetQuestTime` | `SetQuestTimeWriter="SetQuestTime"` | CField::OnSetQuestTime |
| STOP_CLOCK | CB | `field/clientbound/StopClock` | `StopClockWriter="StopClock"` | CField::OnDestroyClock |
| FORCED_MAP_EQUIP | CB | `field/clientbound/ForcedMapEquip` | `ForcedMapEquipWriter="ForcedMapEquip"` | CField::OnFieldSpecificData |
| SUMMON_ITEM_INAVAILABLE | CB | `field/clientbound/SummonItemUnavailable` | `SummonItemUnavailableWriter="SummonItemUnavailable"` | CField::OnSummonItemInavailable |
| FOOTHOLD_INFO | CB | `field/clientbound/FootholdInfo` | `FootholdInfoWriter="FootholdInfo"` | CField::OnRequestFootHoldInfo |
| GMEVENT_INSTRUCTIONS | CB | `field/clientbound/GmEventInstructions` | `GmEventInstructionsWriter="GmEventInstructions"` | CField::OnDesc |
| OX_QUIZ | CB | `field/clientbound/OxQuiz` | `OxQuizWriter="OxQuiz"` | CField::OnQuiz |
| PLAY_JUKEBOX | CB | `field/clientbound/PlayJukebox` | `PlayJukeboxWriter="PlayJukebox"` | CField::OnPlayJukeBox |

- [ ] One task per op (R-CB or R-MARK per triage). Opcodes per version from `structures/<version>.md`;
  codec body from `structures/<version>.md#<OP>`; markers/evidence only for "implement" versions in
  `structures/applicability.md`. `FOOTHOLD_INFO` (`OnRequestFootHoldInfo`) interacts with the
  Cluster-4 C-cluster — if Task 0.3 collapsed an `IDA_0x…` row into it, model once and route both opcodes.

### Cluster 3 — Core CField: boss timers / events / admin / MTS / door / guild (≈10 ops)

| Op | Dir | pkg / Struct | NAME const | fname |
|---|---|---|---|---|
| ZAKUM_SHRINE | CB | `field/clientbound/ZakumShrine` | `ZakumShrineWriter="ZakumShrine"` | CField::OnZakumTimer |
| HORNTAIL_CAVE | CB | `field/clientbound/HorntailCave` | `HorntailCaveWriter="HorntailCave"` | CField::OnHontailTimer |
| WITCH_TOWER_SCORE_UPDATE | CB | `field/clientbound/WitchTowerScoreUpdate` | `WitchTowerScoreUpdateWriter="WitchTowerScoreUpdate"` | CField::OnChaosZakumTimer |
| ARIANT_RESULT | CB | `field/clientbound/AriantResult` | `AriantResultWriter="AriantResult"` | CField::OnWarnMessage |
| ADMIN_RESULT | CB | `field/clientbound/AdminResult` | `AdminResultWriter="AdminResult"` | CField::OnAdminResult |
| VICIOUS_HAMMER | CB | `field/clientbound/ViciousHammer` | `ViciousHammerWriter="ViciousHammer"` | CField::OnItemUpgrade |
| MTS_OPERATION | CB | `field/clientbound/MtsOperation` | `MtsOperationWriter="MtsOperation"` | CField::OnCharacterSale |
| MTS_OPERATION2 | CB | `field/clientbound/MtsOperation2` (or mode of MtsOperation) | `MtsOperation2Writer="MtsOperation2"` | CField::OnCharacterSale |
| USE_DOOR | ? | `field/<dir>/UseDoor` | `UseDoor{Writer\|Handle}` | CField::TryEnterTownPortal |
| GUILD_OPERATION | ? | `field/<dir>/GuildOperation` (likely A — `PKT`) | `GuildOperation{Writer\|Handle}` | CField::InputGuildName |

- [ ] One task per op. **MTS:** model per the Task-0.3 Step-3 verdict (two modes of one struct → one
  model + two routes; two distinct → two structs). **USE_DOOR / GUILD_OPERATION:** direction + A/B
  per Task 0.3 Step 4 — GUILD_OPERATION (`PKT`) is likely R-MARK/R-WRAP.

### Cluster 4 — Foothold/stalk C-cluster

> The `IDA_0X098/09C/09D/0A4/0AA/0AC/0B0/0B1/0X169` rows, **after** their Task-0.3 resolution. By
> design §3 most resolve to (a) collapse-into-a-named-op (already handled in Cluster 2/3 — just
> route the extra opcode + add the marker/evidence for that version) or (c) ⬜ VERSION-ABSENT. Only
> a genuinely-distinct (b) op gets its own R-CB/R-SB task here.

- [ ] **Step 1:** For each `IDA_0x…` row, read its Task-0.3 resolution from `structures/triage.md`.
- [ ] **Step 2 (collapse):** add the version's opcode route + verify marker + evidence pin to the
  already-implemented named op (R-MARK steps); confirm the cell flips ✅.
- [ ] **Step 3 (distinct):** full R-CB/R-SB for the new op using its derived layout.
- [ ] **Step 4 (absent):** pin a `VERSION-ABSENT` evidence record (no test) citing the
  `structures/applicability.md` justification; confirm the cell grades ⬜, not 🟥.
- [ ] **Step 5:** No `IDA_0x…` string reaches code (struct names use the resolved op-name). Commit.

### Cluster 5 — Minigames (the `CField_*` subclasses, ≈30 ops)

> One block, sub-grouped per subclass. Codecs stay under `field/{clientbound,serverbound}/` (keep
> the tier-1 prefix). Expect several IDB-evidenced ⬜ VERSION-ABSENT cells (GMS-event-only; jms may
> lack them). The `?`-direction rows (`SNOWBALL`/`LEFT_KNOCKBACK`/`COCONUT`/`GUILD_BOSS`/`CONTI_MOVE`)
> use the send-site fname resolved in Task 0.3 Step 5.

Sub-groups (op names from `structures/cfield-ops.md`; recipe = R-CB unless triage marks SB/A):

- [ ] **2.5.a — SnowBall (6):** HIT_SNOWBALL, LEFT_KNOCKBACK, LEFT_KNOCK_BACK, SNOWBALL,
  SNOWBALL_MESSAGE, SNOWBALL_STATE. Structs `field/clientbound/SnowBall*` (or `field/serverbound/`
  for the resolved `?` rows). One task per op (R-CB/R-SB per triage).
- [ ] **2.5.b — Tournament (5):** TOURNAMENT, TOURNAMENT_MATCH_TABLE, TOURNAMENT_SET_PRIZE,
  TOURNAMENT_UEW (+ the 5th from `cfield-ops.md`). Structs `field/clientbound/Tournament*`.
- [ ] **2.5.c — Wedding (4):** WEDDING_ACTION, WEDDING_CEREMONY_END, WEDDING_PROGRESS, WEDDING_TALK
  (3 of 4 share `OnWeddingProgress` — confirm modes vs distinct in Stage 1, model accordingly).
  Structs `field/clientbound/Wedding*`.
- [ ] **2.5.d — Coconut (3):** COCONUT (`?`→resolved), COCONUT_HIT, COCONUT_SCORE. Structs
  `field/clientbound/Coconut*`.
- [ ] **2.5.e — GuildBoss (3):** GUILD_BOSS (`?`→resolved), GUILD_BOSS_HEALER_MOVE,
  GUILD_BOSS_PULLEY_STATE_CHANGE. Structs `field/clientbound/GuildBoss*`.
- [ ] **2.5.f — ContiMove (2):** CONTI_MOVE (`Init`→resolved) and CONTI_MOVE (`OnContiMove`) —
  resolve the dupe in triage (likely one CB op; the `Init` row may be ⬜/collapse). Structs
  `field/clientbound/ContiMove`.
- [ ] **2.5.g — AriantArena (2):** ARIANT_ARENA_SHOW_RESULT, ARIANT_ARENA_USER_SCORE. Structs
  `field/clientbound/AriantArena*`.
- [ ] **2.5.h — Battlefield/sheep-ranch (2):** SHEEP_RANCH_CLOTHES, SHEEP_RANCH_INFO. Structs
  `field/clientbound/SheepRanch*` (owner `CField_Battlefield`).
- [ ] **2.5.i — Massacre (2):** PYRAMID_GAUGE (`CField_Massacre::OnMassacreIncGauge`),
  PYRAMID_SCORE (`CField_MassacreResult::OnMassacreResult`). Structs `field/clientbound/Pyramid*`.
- [ ] **2.5.j — Witchtower (1):** ARIANT_SCORE (`CField_Witchtower::OnPacket`). Struct
  `field/clientbound/AriantScore`.

For each minigame op: R-CB (or R-SB/R-MARK per triage). Emit markers/evidence ONLY for the
applicable versions; for inapplicable versions, pin a `VERSION-ABSENT` evidence record (no test) so
the cell grades ⬜, citing `structures/applicability.md`. Confirm with `matrix --check` that
inapplicable cells are `⬜ n/a`, not `🟥 conflict`. Commit per sub-group.

---

## Stage 3 — Documentation

### Task 9.1: `deploy-notes.md`

**Files:** Create `docs/tasks/task-096-cfield-packet-family/deploy-notes.md`.

- [ ] **Step 1:** Per-version opcode table for every new/linked handler + writer (from
  `structures/<version>.md`), in the live-tenant PATCH shape (the `socket.handlers` entry with
  validator, and `socket.writers` entry), grouped by version — the same shape as task-092's
  deploy-notes.
- [ ] **Step 2:** The rollout checklist: PATCH each live v83/v84/v87/v95/jms tenant
  `socket.handlers`/`socket.writers` with the new entries; **restart `atlas-channel`** (the
  handler/writer map is built once at startup; the config projection does not hot-reload them —
  memory `bug_new_opcodes_not_in_live_tenant_config`); post-deploy checks (`grep "Unable to locate
  validator"`==0; no new error/fatal; the new serverbound ops no longer emit "unhandled message op
  0xXX"). Commit.

> Re-authoring `IMPLEMENTING_A_PACKET.md` / `VERIFYING_A_PACKET.md` is **out of scope** — task-092
> produced them; task-096 follows them (design §11). If a CField-specific gotcha surfaces that the
> docs miss, add a short note to the existing doc rather than rewriting it.

---

## Stage 4 — Final verification & handoff

### Task 10.1: Full gates

- [ ] **Step 1:** `go test -race ./...` clean in `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`,
  and `services/atlas-configurations` (+ `tools/packet-audit` if touched).
- [ ] **Step 2:** `go vet ./...` clean in the same; `GOWORK=off ./tools/redis-key-guard.sh` clean from repo root.
- [ ] **Step 3:** `git diff --name-only -- '**/go.mod'` — expected empty (libs/atlas-packet is a
  workspace member; no new lib). If any go.mod changed, `docker buildx bake atlas-<svc>` from the
  worktree root for that service; expect success. (JSON-only template edits → no bake.)
- [ ] **Step 4:** `go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check; echo exit=$?`
  — expect exit 0; spot-check STATUS.md shows every targeted CField op ✅ for applicable versions and
  ⬜ (n/a) with VERSION-ABSENT evidence elsewhere; 0 🟥; no orphan/dangling/stale/drift line
  mentioning a CField packet.
- [ ] **Step 5:** Validate seed-template JSON (each `template_*.json` parses; `handlers`/`writers`
  arrays ascending by opCode; every handler entry has a validator). Final commit of regenerated
  STATUS.md/status.json if not already committed per-cluster.

### Task 10.2: Code review

- [ ] **Step 1:** Invoke `superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` +
  `backend-guidelines-reviewer`). Brief the backend reviewer that uncalled clientbound writer `Body`
  helpers are an intentional seam (design D2 / `IMPLEMENTING_A_PACKET.md`), not dead code, and that
  the chat→field relocation (D3) is a deliberate move-not-rewrite for tier-1-prefix alignment.
- [ ] **Step 2:** Address findings per `superpowers:receiving-code-review`. Re-run Task 10.1 gates after fixes.
- [ ] **Step 3:** Confirm acceptance criteria (PRD §10): every applicable CField cell ✅; genuine
  absences ⬜ with VERSION-ABSENT evidence; 0 conflicts; no duplicate codec for an A-row (triage
  honored); gates green; `deploy-notes.md` present. Then proceed to PR per `superpowers:finishing-a-development-branch`.

---

## Self-Review notes (coverage check vs design)

- Design §1 scope (75 ops, ✅/⬜, verify-existing AND implement-new) → Stage 0 triage + Stage 2 clusters 1–5. ✓
- Design §1 "verified requires codec+test+marker+evidence" (tier-1 `field/`) → R-CB/R-SB steps + R-MARK. ✓
- Design D1 (one branch/PR, cluster-gated commits) → Stage 2 commits per op/sub-group; per-cluster gates. ✓
- Design D2 (codec+route+byte-test only; SB decode-and-log; CB uncalled seam) → R-CB.7/R-SB handler body + Task 10.2 brief. ✓
- Design D3 (relocate CField chat codecs chat→field) → Cluster 1 (Tasks 2.1.1–2.1.5) + context.md §1 tier-1 note. ✓
- Design D4 (triage is a committed artifact; C-rows derived not deferred) → Stage 0 Tasks 0.3/0.4 (`structures/triage.md`). ✓
- Design §3 C-row resolution (foothold/stalk, MTS, door/guild, minigame `?`) → Task 0.3 Steps 2–6 + Cluster 4. ✓
- Design §4 per-op four-step recipe → Stage 1 (derive) + R-CB/R-SB/R-MARK/R-WRAP (model/wire/verify). ✓
- Design §5 move-rule (CField-only ownership; shared = link-in-place; NAME constants unchanged) → Task 0.4 Step 2 + Task 2.1.1 Steps 1–3. ✓
- Design §6 cluster sequencing 0→1→2→3→4→5 → Stage 0 + Stage 2 Clusters 1–5. ✓
- Design §7 version handling (`MajorAtLeast(87)`, VERSION-ABSENT only with evidence) → context.md §1 gate rule + Stage 1 Task 1.F + cluster VERSION-ABSENT steps. ✓
- Design §8 rollout (seed-only, documented PATCH) → Task 9.1 deploy-notes. ✓
- Design §9 gates → Task 10.1. ✓
- Design §10 risks (dup codec, chat-relocation regression, IDA_0x placeholders, MTS modes, false ⬜, v84≡v83, validator-less, export non-idempotency) → triage gate + Cluster-1 matrix re-check + Task 0.3 + applicability + context.md §4/§5. ✓
- Design §11/§12 (out of scope: gameplay, IMPLEMENTING re-author, live PATCH) + open-Q resolutions → Stage 3 note + Task 0.3 + cluster notes. ✓
- PRD §10 acceptance criteria → Task 10.1/10.2 Step 3. ✓
