# Starting a new packet-audit version pass

> Orchestration playbook for adding a new client version column to the coverage
> matrix (`docs/packets/audits/STATUS.md`). Written against the task-085
> workflow; replaces the pre-matrix invocations. The old §2–§8 content has been
> subsumed by `VERIFYING_A_PACKET.md` (single-cell procedure) and by the matrix
> subcommand itself.

Baseline versions (existing columns):

| Version key | Region / major.minor | Template | IDA export |
|---|---|---|---|
| `gms_v48`  | GMS 48.1  | `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`  | `docs/packets/ida-exports/gms_v48.json`  |
| `gms_v61`  | GMS 61.1  | `services/atlas-configurations/seed-data/templates/template_gms_61_1.json`  | `docs/packets/ida-exports/gms_v61.json`  |
| `gms_v72`  | GMS 72.1  | `services/atlas-configurations/seed-data/templates/template_gms_72_1.json`  | `docs/packets/ida-exports/gms_v72.json`  |
| `gms_v79`  | GMS 79.1  | `services/atlas-configurations/seed-data/templates/template_gms_79_1.json`  | `docs/packets/ida-exports/gms_v79.json`  |
| `gms_v83`  | GMS 83.1  | `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`  | `docs/packets/ida-exports/gms_v83.json`  |
| `gms_v84`  | GMS 84.1  | `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`  | `docs/packets/ida-exports/gms_v84.json`  |
| `gms_v87`  | GMS 87.1  | `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`  | `docs/packets/ida-exports/gms_v87.json`  |
| `gms_v95`  | GMS 95.1  | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`  | `docs/packets/ida-exports/gms_v95.json`  |
| `jms_v185` | JMS 185.1 | `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` | `docs/packets/ida-exports/gms_jms_185.json` |

Adding a new version (e.g. `gms_v92`) means running through §1 once, then
iterating on §3 until the declared scope is satisfied.

---

## 1. Set up the column

Four artefacts must exist before `matrix` can emit cells for the new version:
a registry file, a tenant template, an IDA export, and at least one completed
audit pass.

### 1.1 Registry file — `discover-ops`

The operation registry (`docs/packets/registry/<version>.yaml`) is the
authoritative list of opcodes the client handles for that version. It determines
applicability for every cell: present → applicable; absent → `⬜`. Without it
every cell is `incomplete` with an "applicability unknown" note.

**Step A: seed from CSVs**

```bash
go run ./tools/packet-audit registry seed \
  --clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  --serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  --out docs/packets/registry
```

This writes one YAML per version key it finds column headers for. For a version
the CSVs have no column for (e.g. `gms_v84`), copy the nearest version's YAML
manually and annotate with a note explaining the provenance.

**Step B: run `discover-ops` against the IDB**

`CClientSocket::ProcessPacket` is a shim, not the primary dispatcher — it is
the correct entry point for IDA but it routes internally to ~40 `*::OnPacket`
dispatcher functions. The v87 run identified 40 dispatchers; expect a similar
count for any GMS version (see `docs/packets/registry/discover_gms_v87.md` as
the reference worklist).

```bash
go run ./tools/packet-audit discover-ops \
  --version <version-key> \
  --ida-port <port> \
  --ida-url http://192.168.20.3:13337/mcp \
  --out /tmp/<version>_discover.md
```

Flags:

```
  -apply
        when true, append discovered ops to the registry YAML
  -dispatcher string
        comma-separated list of dispatcher function names and/or hex addresses
        (default "CClientSocket::ProcessPacket")
  -ida-port int
        IDA-MCP instance port to select (0 = default active instance)
  -ida-url string
        IDA-MCP HTTP endpoint (default "http://192.168.20.3:13337/mcp")
  -out string
        worklist markdown output path (default: docs/packets/registry/discover_<version>.md)
  -registry-dir string
        directory containing <version>.yaml registry files (default "docs/packets/registry")
  -version string
        target version key, e.g. gms_v83 (required)
```

**Dispatcher curation (mandatory before `--apply`)**

`CClientSocket::ProcessPacket` IS a shim; its internal routing fans to the real
dispatcher set. The v87 run found ~40 dispatcher addresses this way. Before
`--apply`, review the discover worklist against this curation checklist:

- **Include**: every `*::OnPacket` function whose switch branches on the
  top-level game opcode (e.g. `CWvsContext::OnPacket`, `CField::OnPacket`,
  `CLogin::OnPacket`, all `CField_*` subclass overrides, `CCashShop::OnPacket`,
  all pool dispatchers `CMobPool::OnPacket`, `CNpcPool::OnPacket`, etc.).
- **Exclude** (body-mode demuxers whose internal switch is NOT the top-level
  opcode): `CScriptMan::OnPacket`, `CShopDlg::OnPacket`, `CAdminShopDlg::OnPacket`,
  `CStoreBankDlg::OnPacket`, `CMiniRoomBaseDlg::OnPacketBase`, `CTrunkDlg`,
  `CRPSGameDlg`, `CUIMessenger`, and similar single-opcode family handlers.
- **Per-dispatcher decompile-check**: decompile each candidate address and
  confirm the switch scrutinee is the opcode integer, not a secondary mode byte.
- **Collision resolution**: if two dispatchers claim the same opcode with
  different handlers, `discover-ops` refuses `--apply` until the collision is
  manually resolved and recorded with `provenance: manual`.

Once curation is done, re-run with `--apply`:

```bash
go run ./tools/packet-audit discover-ops \
  --version <version-key> \
  --ida-port <port> \
  --dispatcher 0xADDR1,0xADDR2,...   \
  --apply \
  --out docs/packets/registry/discover_<version>.md
```

This appends new ops with `provenance: ida-discovered` and their handler
address. Registry entries not found by discovery are flagged in the worklist
under "Missing at discovery" — resolve each as a CSV transcription error
(correct the entry) or a discovery blind spot (record as `provenance: manual`
with an IDA citation). See `discover_gms_v87.md` for a completed example.

**Instance selection**: multiple IDBs can be loaded simultaneously; never
hardcode port numbers. Use `mcp__ida-pro__list_instances` to enumerate loaded
IDBs, then pass the matching port as `--ida-port`. The current four-version
convention assigns ports 13337–13340 by convention but port assignment depends
on IDA launch order — always confirm via `list_instances`.

### 1.2 Tenant template

Add a seed template at
`services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`.
The version key derives from the template's `region` + `major` fields (e.g.
region `"GMS"`, major `84` → version key `gms_v84`).

Template opcodes must match the registry: ops the registry marks present should
be routed; ops the registry marks absent should not appear. Disagreement becomes
a 🟥 conflict cell. Resolve via §5.1 (three-way arbiter: IDB first, then fix
whichever leg is wrong).

**Live-tenant warning**: seed templates apply only at tenant creation. Fixing a
template opcode does NOT automatically update existing tenants — you must patch
the live tenant's config and restart the channel (handler/writer projections
don't hot-reload).

### 1.3 IDA export

The export is a machine-harvested JSON of the IDB's `Encode`/`Decode` read-order
per handler function. It feeds the static audit pass and the evidence freshness
checks.

**Roster bootstrap**: the exporter builds its function roster from
`candidatesFromFName` (derived from the template opcodes). A first export for a
new version needs a seeded roster — copy the nearest version's export JSON as
the starting point and purge any cross-IDB coincidentals (functions that appear
in both versions only because the nearest version shared a binary segment, not
because the target version actually has them). Then run:

```bash
go run ./tools/packet-audit export \
  --version <version-key> \
  --ida-port <port> \
  --ida-url http://192.168.20.3:13337/mcp \
  --output docs/packets/ida-exports/<version>.json
```

Flags:

```
  -descent-depth int
        max helper-descent recursion depth (default 6)
  -generated-at string
        fixed provenance timestamp (default: now / $PACKET_AUDIT_GENERATED_AT)
  -ida-port int
        IDA-MCP instance port to select (0 = default active instance)
  -ida-timeout duration
        per-call IDA-MCP timeout (default 1m0s)
  -ida-url string
        IDA-MCP HTTP endpoint (default "http://192.168.20.3:13337/mcp")
  -output string
        output JSON path (required)
  -version string
        target version key, e.g. gms_v95 (required)
```

Run a small smoke-test roster first (a handful of known-good FNames) before
committing to the full run. The exporter tolerates decompile failures
(they become `Unresolved` entries and BFS continues); a genuine transport error
aborts loudly.

### 1.4 Static audit pass

After the registry, template, and export exist, run the static audit for the new
version. From the worktree root:

```bash
go run ./tools/packet-audit \
  --csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  --atlas-packet    libs/atlas-packet \
  --template        services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json \
  --ida-source      docs/packets/ida-exports/<version>.json \
  --output          docs/packets/audits
```

Run from the worktree root. This writes `docs/packets/audits/<version>/SUMMARY.md`
plus per-packet `.md` detail files; `--output` is the parent directory, not the
versioned subdirectory.

**Live IDB validation pass** (optional but strongly recommended): after the
static pass, run `validate` to cross-check the hand-authored baseline against
the open IDB:

```bash
go run ./tools/packet-audit validate \
  --version <version-key> \
  --ida-port <port> \
  --report /tmp/<version>_validate.md
```

Flags:

```
  -allowlist string
        unimplemented-case allowlist (default: docs/packets/audits/<auditdir>/_unimplemented.json)
  -baseline string
        baseline export JSON path (default: docs/packets/ida-exports/<version>.json)
  -descent-depth int
        max helper-descent recursion depth (default 6)
  -ida-port int
        IDA-MCP instance port to select (0 = default active instance)
  -ida-timeout duration
        per-call IDA-MCP timeout (default 1m0s)
  -ida-url string
        IDA-MCP HTTP endpoint (default "http://192.168.20.3:13337/mcp")
  -report string
        output markdown report path (required)
  -version string
        target version key, e.g. gms_v95 (required)
```

Triage `divergent` entries with `diff-shape`; allowlist genuine `missing-mode`
cases into `docs/packets/audits/<version>/_unimplemented.json`.

**`decompose` — extend the baseline with live IDA reads for every exported entry**

```bash
go run ./tools/packet-audit decompose \
  --version   <version-key> \
  --ida-port  <port> \
  --out       /tmp/<version>_extended.json \
  --report    /tmp/<version>_decompose.md
```

Flags:

```
  -audit-dir string
        committed audit dir (default: docs/packets/audits/<version>)
  -baseline string
        baseline export JSON path (default: docs/packets/ida-exports/<version>.json)
  -descent-depth int
        max helper-descent recursion depth (default 6)
  -ida-port int
        IDA-MCP instance port to select (0 = default active instance)
  -ida-timeout duration
        per-call IDA-MCP timeout (default 1m0s)
  -ida-url string
        IDA-MCP HTTP endpoint (default "http://192.168.20.3:13337/mcp")
  -out string
        output extended baseline JSON path (required)
  -report string
        output markdown report path (required)
  -version string
        target version key, e.g. gms_v83 (required)
```

> **JMS quirk**: `--version gms_jms_185` defaults `--audit-dir` to
> `docs/packets/audits/gms_jms_185`, which does not exist; the actual audit dir
> is `docs/packets/audits/jms_v185`. Always pass `--audit-dir
> docs/packets/audits/jms_v185` explicitly for JMS passes.

**`triage` — produce a divergent-entry worklist from the extended baseline**

```bash
go run ./tools/packet-audit triage \
  --version   <version-key> \
  --ida-port  <port> \
  --report    /tmp/<version>_triage.md
```

Flags:

```
  -audit-dir string
        committed audit dir (default: docs/packets/audits/<version>)
  -baseline string
        baseline export JSON path (default: docs/packets/ida-exports/<version>.json)
  -descent-depth int
        max helper-descent recursion depth (default 6)
  -ida-port int
        IDA-MCP instance port to select (0 = default active instance)
  -ida-timeout duration
        per-call IDA-MCP timeout (default 1m0s)
  -ida-url string
        IDA-MCP HTTP endpoint (default "http://192.168.20.3:13337/mcp")
  -report string
        output markdown worklist path (required)
  -version string
        target version key, e.g. gms_v95 (required)
```

> **JMS quirk**: same as `decompose` — pass `--audit-dir
> docs/packets/audits/jms_v185` explicitly for `--version gms_jms_185`.

**`resolve-dispatch` — auto-write high-confidence selectors into the baseline**

```bash
go run ./tools/packet-audit resolve-dispatch \
  --version   <version-key> \
  --ida-port  <port> \
  --worklist  /tmp/<version>_resolve.md
```

Flags:

```
  -baseline string
        baseline export JSON path (default: docs/packets/ida-exports/<version>.json)
  -descent-depth int
        max helper-descent recursion depth (default 6)
  -ida-port int
        IDA-MCP instance port to select (0 = default active instance)
  -ida-timeout duration
        per-call IDA-MCP timeout (default 1m0s)
  -ida-url string
        IDA-MCP HTTP endpoint (default "http://192.168.20.3:13337/mcp")
  -min-confidence float
        auto-accept threshold (default 0.6)
  -version string
        target version key, e.g. gms_v95 (required)
  -worklist string
        output confirmation worklist markdown path (required)
```

Review the worklist for low-confidence picks before committing the mutated baseline.

### 1.5 Serverbound opcode verification — `verify-serverbound`

The serverbound half of the registry gets its own bulk verification pass:
for every serverbound registry entry, decompile the client **send function**
(the address comes from the committed audit reports under
`docs/packets/audits/<version>/`, keyed by the entry's `fname`) and check that
the registry opcode appears among the literals passed to
`COutPacket::COutPacket`. Run it once the registry (§1.1) and the static audit
pass (§1.4) exist — the audit reports are where it finds the send-site
addresses — with the version's IDB open:

```bash
go run ./tools/packet-audit verify-serverbound \
  --version  <version-key> \
  --ida-port <port>
```

Flags:

```
  -audits-dir string
        parent directory containing per-version audit report dirs (default "docs/packets/audits")
  -ida-port int
        IDA-MCP instance port to select (0 = default active instance)
  -ida-url string
        IDA-MCP HTTP endpoint (default "http://192.168.20.3:13337/mcp")
  -out string
        worklist markdown output path (default: docs/packets/registry/verify_serverbound_<version>.md)
  -registry-dir string
        directory containing <version>.yaml registry files (default "docs/packets/registry")
  -version string
        target version key, e.g. gms_v83 (required)
```

Output is a committed worklist
(`docs/packets/registry/verify_serverbound_<version>.md` — see the
`gms_v87` / `gms_v95` / `jms_v185` ones for completed examples) with three
buckets. The command exits 0 regardless of bucket counts — it is a worklist
generator, not a CI gate; the worklist is what you burn down:

- **Confirmed** — registry opcode found in the send function's `COutPacket`
  constructor call set. No action.
- **Mismatch — REVIEW** — the registry opcode is NOT in the decompiled send
  function's opcode set (the found set is listed). Either the `fname` is wrong
  (opcode-cluster off-by-one — the registry-fname mislabel trap in
  `IMPLEMENTING_A_PACKET.md` Step 1) or the opcode assignment is wrong.
  Arbitrate against the IDB and fix the registry row in the same change
  (`provenance: manual`, IDA citation in `note`).
- **Unresolved** — could not verify, with the reason per row: no `fname` in the
  registry (derive the send-site and populate it, `provenance: ida-discovered`),
  no audit-report address for the fname (complete the §1.4 audit pass first),
  decompile error, or a send site with no opcode literal (dynamic opcodes).

---

## 2. Regenerate the matrix

Once all four artefacts are in place, regenerate:

```bash
go run ./tools/packet-audit matrix
```

Flags:

```
  -audits-dir string
        audit reports parent dir (default "docs/packets/audits")
  -check
        CI mode: verify committed outputs are current; fail on conflicts/drift
  -evidence-dir string
        evidence ledger dir (default "docs/packets/evidence")
  -exports-dir string
        IDA export JSON dir (default "docs/packets/ida-exports")
  -out-dir string
        output dir for STATUS.md/status.json (default "docs/packets/audits")
  -packet-lib string
        atlas-packet root for marker scanning (default "libs/atlas-packet")
  -registry-dir string
        registry YAML dir (default "docs/packets/registry")
  -templates-dir string
        tenant seed templates dir (default "services/atlas-configurations/seed-data/templates")
  -tiers string
        tier-1 membership YAML (default "docs/packets/evidence/tiers.yaml")
  -versions string
        comma-separated version keys (default "gms_v48,gms_v61,gms_v72,gms_v79,gms_v83,gms_v84,gms_v87,gms_v95,jms_v185")
```

The new version column appears automatically, pre-filled from applicability: ⬜
for ops the registry marks absent, ❌ for ops marked present but unverified. Any
applicability disagreement between the registry and the template prints as a 🟥
conflict and is listed in STATUS.md's conflicts section.

Run `--check` immediately after to capture the baseline conflict and freshness
state:

```bash
go run ./tools/packet-audit matrix --check 2>/tmp/matrix_check.txt; echo "exit=$?"
grep -c '' /tmp/matrix_check.txt   # total findings
grep -ciE 'orphan|dangling|stale|drift|unresolv|malformed' /tmp/matrix_check.txt
```

The second grep must be 0 before committing. `matrix --check` is a hard,
blocking CI gate (`.github/workflows/packet-matrix.yml`) with no
`continue-on-error`: the registry-seed conflict backlog was burned to zero
(task-085), so any 🟥 conflict, fatal finding, or stale committed
STATUS.md/status.json fails CI. Resolve every conflict the pass introduces (or
own it via §5.1) and commit a fresh STATUS.md/status.json before landing.

Commit registry + template + export + audit output + STATUS.md/status.json in a
single PR. The PR description should call out any conflict cells the pass
introduces and their diagnosis.

---

## 3. Promote cells

The pass's job is turning ❌ cells into ✅ or 🟡. Apply
`docs/packets/audits/VERIFYING_A_PACKET.md` per cell, working hottest-tier cells
first (tier-1 packets in `docs/packets/evidence/tiers.yaml` require a byte-fixture
test; tier-0 cells can reach 🟡 from a tool ✅ alone).

Cell states: `✅` verified · `🧩` family (mode-prefix dispatcher; sub-arms
unverified — capped for ops whose registry fname is listed in
`docs/packets/evidence/families.yaml`, currently empty so no op caps) ·
`🟡` partial · `❌` incomplete · `⬜` n-a · `🟥` conflict.

**Fan-out with the packet-verifier agent**: for campaign-scale verification
(verifying a whole version's scope), dispatch the `packet-verifier` agent per
cell family. Each agent invocation follows the `VERIFYING_A_PACKET.md` steps
§0–10; the results are committed as test + evidence + STATUS.md in that agent's
sub-task. Coordinate via a per-version worklist (the `discover-ops` worklist
markdown is a convenient starting point).

**Serverbound worklist — `verify-serverbound`.** For the serverbound half of the
scope, generate the send-site worklist first — it is the serverbound analogue of
the `discover-ops` clientbound worklist and the entry point into the §9
three-artifact verification:

```bash
go run ./tools/packet-audit verify-serverbound --version <version-key> --ida-port <port>
```

Flags (defaults shown): `--registry-dir docs/packets/registry`, `--audits-dir
docs/packets/audits`, `--ida-port 0` (active instance), `--out` (default
`docs/packets/registry/verify_serverbound_<version>.md`). It decompiles each
serverbound send function (address sourced from this version's committed audit
reports) and checks the registry opcode against the `COutPacket::COutPacket`
literal, bucketing every op into **Confirmed** / **Mismatch — REVIEW** /
**Unresolved**:

- **Confirmed** rows are ready for `VERIFYING_A_PACKET.md` §9 (marker + evidence +
  report) → hand them to `packet-verifier`.
- **Mismatch** rows are a wrong registry `fname` or opcode — fix the registry
  (§5.1 leg 1) before verifying.
- **Unresolved** rows have no audit-report address for the fname yet — generate
  the §9 report first (or add the primary fname as a `candidatesFromFName` case),
  then re-run. Unresolved is never a pass.

After each batch of verifications, regenerate:

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

The cell must now be ✅. Commit test + evidence record + STATUS.md/status.json
together. Never hand-edit STATUS.md.

---

## 4. Task-close gate

An audit or version-pass task is done when **every cell in its declared scope**
is ✅, 🟡-with-evidence, or ⬜.

Rules:
- The scope declaration is a list of matrix cells in the task PRD (operation ×
  direction × version). No prose acceptance.
- A cell that remains ❌ in scope at close time blocks the task. Either fix it
  or carve it into a follow-up with its own PRD scope declaration.
- Cell regressions in a PR (any cell degrading from a better state) fail
  `matrix --check` unless the regenerated STATUS.md is committed in the same PR
  and the PR description explicitly owns the regressions with a diagnosis.
- 🟡 cells count as done if and only if they have an evidence record in
  `docs/packets/evidence/<version>/<packet>.yaml` whose `decompile_sha256` still
  matches the current IDA export (no hash drift).

---

## 5. Degradation remediation paths

### 5.1 Conflict cells (🟥)

A 🟥 means three-way disagreement between the operation registry, the version's
template, and Atlas code gates. The IDB is the only neutral arbiter — always
diagnose against it before touching any of the three legs.

**Leg 1 — Registry wrong** (seed transcription error or discovery blind spot):
correct the registry entry. Set `provenance: manual` and add an `ida.address`
citation. This is a doc-only change; the cell re-grades on regeneration.

**Leg 2 — Template wrong** (op unrouted in a version whose client has it, or
routed where the client lacks it):
1. Fix the seed template in `services/atlas-configurations/seed-data/templates/`.
2. **Patch live tenant configs** — seed templates apply only at tenant creation;
   a template-only fix silently does nothing for existing tenants ("unhandled
   message op" bug class).
3. Restart the channel after patching — handler/writer opcode projections don't
   hot-reload.

**Leg 3 — Atlas code wrong** (version gate includes a version whose client lacks
the packet, or excludes one that has it): wire fix through the normal playbook
(code change + byte-test + evidence record via `VERIFYING_A_PACKET.md`).

Conflicts are blockers in `matrix --check`; they cannot be allowlisted or
silently deferred because every conflict is a place where the server can emit
something a client cannot parse, or vice versa.

### 5.2 Degraded verified cells (✅ → ❌)

Three degradation paths, each with its own remediation:

**1. Evidence hash drift** (re-export changed the decompile text):
- Inspect whether the change is material. Cosmetic churn (Hex-Rays variable or
  label renaming with no read-order change) → re-pin via `evidence pin` after
  confirming the read order is unchanged:

  ```bash
  go run ./tools/packet-audit evidence pin \
    --packet  <pkg/dir/Struct> \
    --version <version-key> \
    --ida     "<FName as it appears in the export>" \
    --category <TIER1-FIXTURE|OPAQUE|TRUNCATION|...>
  ```

- Material change (the actual read order differs) → full playbook re-verification
  (`VERIFYING_A_PACKET.md` §3–8). A changed read order is a finding to
  investigate, not a re-pin.

**2. Broken test linkage** (linked test deleted or renamed):
- `matrix --check` emits an `orphan` line. Restore or re-point the
  `packet-audit:verify` marker:

  ```go
  // packet-audit:verify packet=<pkg/dir/Struct> version=<key> ida=<0xaddr>
  ```

  If the test was deleted because the encoder changed, the cell needs full
  re-verification before the marker is re-added.

**3. Tool verdict flip** (tier-0 cells after an analyzer or exporter change):
- Delta triage: diff before/after `grep -rE '\| (❌|🔍) \|' docs/packets/audits/*/SUMMARY.md`.
- Hand-confirm against the IDB which side is right (the IDA trace always wins
  over the analyzer verdict — run the §6 re-audit sequence to re-derive the
  live read orders and compare against the committed baseline).
- Outcome is either an Atlas wire fix or a tool/export correction — never a
  silent re-accept of the old verdict.

**Live-tenant warning (applies to all legs)**: any wire fix or template fix that
changes what Atlas emits or accepts for an existing tenant requires a live-config
patch **and** a channel restart. Template-only or code-only fixes silently do
nothing for tenants that were already provisioned. After the patch + restart,
confirm the previously-"unhandled op" log lines are gone before declaring the
conflict resolved.

---

## 6. Maintenance: re-audit an existing column after export drift

The §1 toolkit is not just for bring-up. Run this sequence over an **existing**
version column whenever its committed baseline may no longer match the client:

- a re-export changed `docs/packets/ida-exports/<version>.json` (evidence hash
  drift — `matrix --check` starts emitting drift/stale lines, §5.2 path 1);
- an analyzer/exporter change flipped tool verdicts (§5.2 path 3);
- an IDB was re-analyzed or renamed at scale (e.g. a naming pass) and you need
  to prove the read orders are unchanged before re-pinning.

Every step below is **read-only** with respect to the committed baseline except
`decompose` (which writes a *separate* extended-baseline JSON you review before
committing). All need the version's IDB open (`list_instances` → `--ida-port`);
all default `--baseline` to `docs/packets/ida-exports/<version>.json`. The JMS
quirk from §1.4 applies throughout: for `--version gms_jms_185` pass
`--audit-dir docs/packets/audits/jms_v185` explicitly where the command takes
an `--audit-dir`.

**Step 1 — scope the drift.** `git diff` the export JSON and capture the
`matrix --check` output. Cosmetic churn (Hex-Rays variable/label renames, no
read-order change) and material change (the read order actually differs) take
different exits at Step 6 — the point of Steps 2–5 is to sort every entry into
one of those two buckets with evidence.

**Step 2 — `validate`: bucket the whole column.** Cross-checks every committed
baseline entry against the live IDB (§1.4 command + flags). The report buckets
each entry: `verified` / `divergent` / `missing-mode` / `extra-mode` /
`unverifiable` / `allowlisted`. A clean maintenance pass is all-`verified` (plus
allowlisted); anything `divergent` proceeds to Step 3.

**Step 3 — `diff-shape`: classify each divergence.** For every DIVERGENT
baseline entry, emits a side-by-side hand-vs-live read list with the divergence
position classified (prefix/suffix match lengths, length delta). Pure
diagnostic — never mutates the baseline or a verdict.

```bash
go run ./tools/packet-audit diff-shape \
  --version  <version-key> \
  --ida-port <port> \
  --report   /tmp/<version>_diffshape.md
```

(Flags mirror `validate`: `--baseline`, `--descent-depth`, `--ida-url`,
`--ida-timeout`.)

**Step 4 — `decompose`: re-derive faithful read orders.** Re-reads every
exported entry live and writes an extended baseline (`--out`) plus a per-fname
classification report (`--report`): `upgraded` (hand order was a truncation;
replaced with the faithful order) / `unchanged` / `divergence` (mid-stream
mismatch — candidate real bug, entry left untouched) / `needs-dispatch` /
`error` / `missing`. §1.4 has the command + flags. Review the report before
committing the extended baseline — `divergence` entries are findings, not
auto-fixes.

**Step 5 — `triage` + `verify-serverbound`: produce the hand-work worklists.**
`triage` (§1.4) turns the divergent set into a deterministic markdown worklist;
`verify-serverbound` (§1.5) re-checks the serverbound half's opcode↔send-site
agreement. Burn both worklists down against the IDB.

**Step 6 — remediate and re-grade.** Per §5.2: cosmetic-only drift → re-pin via
`evidence pin` after confirming the read order is unchanged (Steps 2–4 are that
confirmation); material read-order change → full `VERIFYING_A_PACKET.md`
re-verification of the affected cells; registry/template legs → §5.1. Then
`matrix` + `matrix --check`, and commit the export, evidence records, worklists,
and regenerated STATUS.md/status.json together.
