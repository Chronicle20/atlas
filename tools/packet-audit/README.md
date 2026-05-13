# packet-audit

Audits `libs/atlas-packet` encoder/decoder wire shapes against IDA-decompiled
client functions. Produces per-packet markdown + JSON reports under
`docs/packets/audits/<region>_v<major>/`.

## Usage

    packet-audit \
      --csv-clientbound  docs/packets/MapleStory\ Ops\ -\ ClientBound.csv \
      --csv-serverbound  docs/packets/MapleStory\ Ops\ -\ ServerBound.csv \
      --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
      --atlas-packet     libs/atlas-packet \
      --ida-source       docs/packets/ida-exports/gms_v95.json \
      --output           docs/packets/audits/gms_v95

Exit codes: 0 clean, 1 blocker, 2 warnings only, 3 runtime error.

See `docs/tasks/task-027-atlas-packet-v95-audit/` for design rationale.

## Refreshing the IDA export

The export at `docs/packets/ida-exports/<region>_v<major>.json` is the canonical
artifact for CI runs (no IDA Pro dependency). To regenerate from a connected
IDA-MCP session:

    packet-audit export \
      --ida-source mcp \
      --csv-clientbound  docs/packets/MapleStory\ Ops\ -\ ClientBound.csv \
      --csv-serverbound  docs/packets/MapleStory\ Ops\ -\ ServerBound.csv \
      --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
      --output           docs/packets/ida-exports/gms_v95.json

The initial v95 export was hand-derived from `docs/packets/spike-login-v95.md`
(six packets); subsequent refreshes use the export subcommand against a live
IDA instance with the matching binary loaded.
