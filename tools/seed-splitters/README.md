# seed-splitters

One-shot Go programs that produce the initial `deploy/seed/<region>/<version>/`
catalog content from existing bundled JSON files. Each is deterministic — rerunning
produces byte-identical output — and is NOT run by CI. They are committed for
reproducibility and to bootstrap new region/version directories from v83 content.

Programs:

- `split-monster-drops/`  — splits `monster_drops.json` (array) into one JSON:API file per monster.
- `split-continent-drops/` — same for `continent_drops.json`.
- `split-gachapons/`      — merges `gachapons.json` + `gachapon_items.json` into one combined file per gachapon, plus `_global/items.json`.
- `wrap-jsonapi/`         — generic wrapper for files that already exist per-entity but lack the JSON:API envelope.
