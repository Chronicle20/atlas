# Tenant socket-template conventions

The tenant seed templates under
`services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`
drive per-version packet routing. Two arrays live under `socket`:

- `handlers` — serverbound: `opCode` → `handler` (+ `validator`, optional
  `services`, `options`).
- `writers` — clientbound: `opCode` → `writer` (+ optional `services`,
  `options`).

## Rule: ascending opcode order (enforced)

**Both `handlers` and `writers` MUST be listed in strictly ascending `opCode`
order within each template.** When you add a handler or writer, insert it at its
sorted position — never append it next to a semantically-related entry (e.g. do
not drop the portable-chair handler right after the heal-over-time handler just
because they are both recovery packets; place it by its opcode).

Why this is a rule and not just a nicety:

- The arrays are loaded into an **opcode-keyed dispatch map**, so order is
  functionally irrelevant to the running server. That is exactly why it drifts —
  nothing at runtime notices when it is wrong.
- Sorted arrays are **diffable and mergeable**: a new entry shows up as a single
  localized insertion, and two branches adding different opcodes do not fight
  over the same append point.
- Sorted arrays are **auditable by eye**: "is opcode `0xNN` already routed on
  this version?" is answerable by scanning, and `verify-serverbound` / template
  cross-checks read cleanly.

## Guard

`tools/template-opcode-order-guard.sh` (repo root) checks every template's
`handlers` and `writers` for ascending `opCode` order and exits non-zero on any
descent. It runs in CI as the **Template Opcode Order Guard** job in
`.github/workflows/pr-validation.yml` and is listed in the CLAUDE.md
Build & Verification checklist.

Run it locally before committing template edits:

```sh
tools/template-opcode-order-guard.sh
```

If it fails, move the offending entry to its sorted position (the guard prints
the exact `0xNN (handler) follows 0xMM (handler)` pair). Re-sorting is safe —
it changes only array order, never behavior.
