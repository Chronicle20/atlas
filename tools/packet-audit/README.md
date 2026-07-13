# packet-audit

Audits `libs/atlas-packet` encoder/decoder wire shapes against IDA-decompiled
client functions. Produces per-packet markdown + JSON reports under
`docs/packets/audits/<version>/` and the coverage matrix
(`docs/packets/audits/STATUS.md` / `status.json`).

Start at `docs/packets/PROCESS.md` for the task-type → entry-point → playbook
index and the current version set / CI gate list.

## Root pipeline (report generation)

Invoked with no subcommand — generates per-packet audit reports for one version
from the CSVs, tenant template, and IDA export:

    packet-audit \
      --csv-clientbound  docs/packets/MapleStory\ Ops\ -\ ClientBound.csv \
      --csv-serverbound  docs/packets/MapleStory\ Ops\ -\ ServerBound.csv \
      --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
      --atlas-packet     libs/atlas-packet \
      --ida-source       docs/packets/ida-exports/gms_v95.json \
      --output           docs/packets/audits

Writes `<output>/<version>/<Writer>.{json,md}`. Required flags:
`--csv-clientbound`, `--csv-serverbound`, `--template`. Defaults:
`--atlas-packet libs/atlas-packet`, `--ida-source mcp`,
`--output docs/packets/audits`. Exit codes: 0 clean, 1 blocker, 2 warnings only,
3 runtime error.

## Subcommands

Dispatched by `cmd.Run` (`tools/packet-audit/cmd/root.go`) — 15 subcommands plus
the root pipeline above:

| Subcommand | Purpose |
|---|---|
| `export` | Harvest a per-version IDA export JSON from a live IDA-MCP instance. |
| `validate` | Cross-check the committed export/baseline against live IDA reads. |
| `infer` | Propose high-confidence dispatch selectors (roll-up proposal JSON). |
| `decompose` | Extend the baseline with live IDA reads for every exported entry. |
| `triage` | Produce a divergent-entry worklist from the extended baseline. |
| `registry` | `registry seed` — seed registry YAMLs from the ops CSVs. |
| `matrix` | Build (and `--check`) the coverage matrix STATUS.md / status.json. |
| `dispatcher-lint` | Enforce the dispatcher-family invariants INV-1..INV-5. |
| `doc-freshness` | `doc-freshness --check` — assert PROCESS.md packet-process-facts match the tool's ground truth (CI-gated). |
| `gate-lint` | Report raw `MajorVersion()` boundary comparisons that should use `MajorAtLeast(N)` (report-only; `--check` to fail). |
| `fname-doc` | Check/regenerate `// packet-audit:fname` struct comments. |
| `operations` | Check/regenerate per-tenant `operations` mode tables. |
| `evidence` | Pin/manage evidence records (`evidence pin ...`). |
| `resolve-dispatch` | Auto-write high-confidence selectors into the baseline. |
| `diff-shape` | Read-only shape diff diagnostic (export vs live IDA). |
| `discover-ops` | Generate a per-version registry/op worklist from templates + CSVs. |
| `verify-serverbound` | Produce a serverbound send-site verification worklist. |

## Refreshing the IDA export

The export at `docs/packets/ida-exports/<version>.json` is the canonical
artifact for CI runs (no IDA Pro dependency at check time). Regenerate from a
connected IDA-MCP session with the matching binary loaded:

    packet-audit export \
      --version   gms_v95 \
      --output    docs/packets/ida-exports/gms_v95.json \
      --ida-url   http://<host>:<port>/mcp \
      --ida-port  13337

`export` flags (source: `runExport` in `cmd/root.go`): `--version` (required),
`--output` (required), `--ida-url` (default `http://192.168.20.3:13337/mcp`),
`--ida-port` (0 = default active instance), `--ida-timeout` (default 60s),
`--descent-depth` (default 6), `--generated-at` (fixed provenance timestamp;
default now / `$PACKET_AUDIT_GENERATED_AT`), `--prior-export` (default
`docs/packets/ida-exports/<version>.json`; pass `""` for a targeted harvest),
`--pending` (default `docs/packets/ida-exports/_pending.md`), `--force`
(overwrite an existing, differing `--output` — off by default), `--splice
<FName>` (merge a single harvested entry into the existing `--output`).

**The export is not idempotent — never overwrite a committed export wholesale.**
Re-running drifts existing function keys (Hex-Rays variance). The tool enforces
this (task-169 FR-3.2): by **default** `export` refuses to overwrite an existing
`--output` when the fresh harvest differs — it writes `<output>.new` plus an
added/removed/changed function-key summary to stderr and exits non-zero, leaving
the committed file untouched. To add or fix one function, harvest a targeted
roster (`--prior-export "" --pending <roster.md>`) and surgically merge exactly
that entry with `--splice <FName>` (all other entries are preserved
byte-for-byte). Pass `--force` only to deliberately regenerate the whole file.
See `docs/packets/audits/VERIFYING_A_PACKET.md` §10.
