# Mode-byte dispatcher enumerations

Each `<dispatcher>.yaml` here fully enumerates the mode arms of one mode-prefix
dispatcher listed in `docs/packets/evidence/families.yaml`. A dispatcher is a
single clientbound opcode whose body begins with a `Decode1(mode)` byte that
switches to per-mode behaviour; the matrix caps such ops at 🧩 (see
`tools/packet-audit/internal/matrix/grade.go`) until every mode is accounted for.

These files are the **complete mode set** — the input the grader uses to decide
when a dispatcher op may graduate 🧩 → ✅. A family passes when every mode is:

- **verified** — Atlas sends this mode and a per-mode byte-fixture proves its
  wire body (`sends: true` + a verified arm codec), or
- **n/a** — Atlas never sends this mode (`sends: false`), e.g. a client-only
  notice arm (`bodyless: true`) or a feature Atlas does not implement.

Enumerations are derived from the GMS v95 PDB-named build (authoritative), each
arm decompiled and body-verified — not inferred from the handler name. The
`body` field is the ordered client read sequence AFTER the mode byte;
`bodyless: true` marks arms that read nothing off the wire (pure UI/notice).

Per-version opcodes for the parent op live in `docs/packets/registry/`; the mode
values themselves are version-stable for these dispatchers unless noted.
