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
