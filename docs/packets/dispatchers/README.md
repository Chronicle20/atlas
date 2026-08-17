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

## Aliases (`alias_of`)

An entry in an `operations`/`errors` table may carry `alias_of: <ANCHOR_KEY>`
instead of `modes:`. The alias is an **Atlas-side key** — a server reason or
taxonomy name the code resolves through `ResolveCode` — bound to the byte an
IDA-verified anchor key in the same table already owns. It exists so a template
can name a server concept without that name implying a *new* client arm.

Rules, enforced by `packet-audit operations`:

- An alias declares `alias_of` and **never** `modes`; the byte always comes
  from the anchor, so an alias can never drift away from a verified value.
- The anchor must be a non-alias key in the same table (no chains, no dangling
  references) — a violation exits 3 as a malformed doc.
- The alias is emitted **only** on versions where the anchor has a mode. On a
  version whose client switch has no case for the anchor, neither key is
  written; the gap is left to the runtime's logged fallback rather than filled
  with an invented code.

These files remain the source of truth in both directions: `operations`
regenerates each template's tables wholesale from the YAML, so a key that lives
only in a template — alias or not — is deleted on the next regeneration and
reported as `EXTRA` by `--check` in the meantime.
