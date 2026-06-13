# Packet operation registry

One YAML per client version: the authoritative per-version operation universe
(rows + applicability) for the coverage matrix (`packet-audit matrix`).

- Seeded once from `docs/packets/MapleStory Ops - {ClientBound,ServerBound}.csv`
  via `packet-audit registry seed` (provenance: `csv-import`). **The CSVs are
  frozen as historical reference** — corrections and additions land here, not
  there.
- Grown/corrected by `packet-audit discover-ops` against the version's IDA
  database (provenance: `ida-discovered`, with the handler/site address).
- Human adjudications (CSV transcription error vs discovery blind spot) are
  recorded as `provenance: manual` with an IDA citation in `note`.
- `gms_v84.yaml` was seeded as a copy of the v83 column (no v84 CSV column;
  task-083 found v84 byte-identical to v83) and is corrected by discovery.

Schema per entry: `op`, `direction`, `opcode`, `fname`, optional `fname_alts`,
`provenance`, optional `ida.address`, optional `note`. Uniqueness:
(op, direction) per file. See task-085 design §5.1–5.2.
