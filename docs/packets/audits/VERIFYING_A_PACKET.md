# Verifying a packet (single packet × single version)

The canonical procedure for promoting a coverage-matrix cell
(`docs/packets/audits/STATUS.md`) to ✅. Written for a human or an agent.
Hard rule (CLAUDE.md "Verification Over Memory"): every byte in a fixture must
trace to a decompile line — never to MapleStory knowledge from memory.

## 0. Prerequisites
- The five registry files in `docs/packets/registry/`.
- The version's IDA export in `docs/packets/ida-exports/` (jms_v185 uses
  `gms_jms_185.json`).
- For fresh decompiles: a live ida-pro-mcp instance with the version's IDB.

## 1. Resolve scope
Look the op up in `docs/packets/registry/<version>.yaml`. If absent there:
your job is confirming `n-a` (verify the template doesn't route the opcode in
`services/atlas-configurations/seed-data/templates/` and no Atlas struct
claims it) or filing a 🟥 conflict — then stop.

## 2. Check current state
- The cell in STATUS.md and `status.json`.
- Any evidence record: `docs/packets/evidence/<version>/<packet dots>.yaml`.
- The latest audit report: `docs/packets/audits/<version>/<Writer>.{json,md}`.

## 3. Decompile the client side
- Enumerate live instances (`mcp__ida-pro__list_instances`) and
  `select_instance` the one whose loaded IDB matches the target version —
  ports vary by IDA launch order, NEVER hardcode them.
- Decompile the registry entry's `fname` (batch `decompile`); descend into
  helper reads (address-based descent, same rule as the exporter).
- Write down the full ordered read/write list including guards and loop bounds.

## 4. Compare against Atlas
The encoder/decoder in `libs/atlas-packet/<pkg>/`, including version gates
(`MajorVersion()` comparisons — beware the v84 off-by-one class: `>83` must be
`>=87` when v84 matches v83). Divergence ⇒ wire fix FIRST (own commit, own
review), then continue.

## 5. Derive expected bytes
Concrete model fixture; hand-compute the byte sequence from the client read
order. One fixture per mode for mode-driven packets. Cite the decompile line
for every field in a comment.

## 6. Write the byte-test
With the marker above the function:

    // packet-audit:verify packet=<pkg/dir/Struct> version=<key> ida=<0xaddr>

Use the existing `pt.Variants` table pattern
(`libs/atlas-packet/party/clientbound/invite_test.go` is the reference).
The table is a `[]struct{ variant pt.TenantVariant; ... }` slice that
accesses `pt.Variants` by index; see `TestInviteByteOutput` for a complete
example with per-variant byte counts and IDA evidence comments.

## 7. Pin evidence
    go run ./tools/packet-audit evidence pin --packet <id> --version <key> \
      --ida "<FName>" --category TIER1-FIXTURE

This writes `docs/packets/evidence/<version>/<packet dots>.yaml`. After the
command succeeds, open the YAML and add the `verifies:` field manually:

    verifies:
      - <test file>#<TestName>

The `--ida` argument is the function name exactly as it appears as a key in
the IDA export's `functions` map (e.g. `CWvsContext::OnPartyResult` — fully
qualified, including any `#case` suffix the export uses), not a hex address.
The tool resolves the address from the export and embeds it in the record
automatically.

## 8. Regenerate and verify promotion
    go run ./tools/packet-audit matrix
    go run ./tools/packet-audit matrix --check
The cell must now be ✅. Commit test + evidence + STATUS.md/status.json
together.

Note on `--check` exit codes: until the registry-seed conflict backlog is
burned down (task-085 Phase 5), `matrix --check` exits 1 from pre-existing 🟥
conflicts unrelated to your cell. Your bar is: the run must introduce **no
new problems** — zero orphan/dangling/stale/drift lines mentioning your
packet, and the conflict count must not increase. Once conflicts reach zero,
the bar becomes a clean exit 0.

## Failure modes (design §13)
- `evidence pin` fails "not in export" → the citation is unresolvable; the
  export needs re-harvest (task-081 playbook) before this cell can verify.
- `matrix --check` reports an orphan marker → the marker's ida= address
  matches neither the evidence record nor the audit report; fix the address,
  never delete the check.
- Hash drift on an existing record → see STATUS degradation paths
  (design §10.2): cosmetic decompile churn → re-pin; material change →
  re-verify from step 3.
