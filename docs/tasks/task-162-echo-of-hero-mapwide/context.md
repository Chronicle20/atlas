# task-162 Echo of Hero Map-Wide — Execution Context

Updated: 2026-08-07 — rebased onto main (254 commits). Decisions D1/D2/D3/D5 were
revised; the pre-rebase versions of this file are obsolete, do not follow them.

## What this task does

The four X005 Echo of Hero skills currently buff only the caster. This task adds a
per-skill registry handler that fans the buff out to every live-session character in
the caster's field, excluding the caster (already buffed by the generic step), dead
characters (HP 0), and hidden GMs. Server-side routing only — no packet, data, buff-service,
or template changes, and **no `common.go` change**.

## Why this got smaller on rebase

Three tasks landed while this one sat:

- **task-187** — `UseSkill` resolves the wire id to a `skill.Identity` through the
  tenant's version set, then dispatches the registry on the Identity. Raw wire-id
  compares are banned by `tools/skill-job-id-guard.sh`.
- **task-156** — added `SelectAllCharactersInMap`, the `healdispel` handler package
  (the template for this task), and `character/buff.IsGmHidden` — the canonical
  **version-aware** hidden-GM check.
- **the per-skill handler registry** — seven handler subpackages already exist.

So the recipient selector and the hidden predicate already exist, and version
correctness falls out of Identity keying instead of needing per-version logic.

## Key files

| File | Why it matters |
|---|---|
| `skill/handler/healdispel/healdispel.go` | **The template.** Same map-wide shape, same `isGmHidden` seam, same deps-struct pattern. Production wiring at lines 160-206; copy its structure. |
| `skill/handler/registry.go` | `Register(id skill2.Identity, h Handler)` / `Lookup`. Keyed on Identity, not wire id — this is the whole version story. |
| `skill/handler/recipients.go` | `SelectAllCharactersInMap` (l.204) — reuse as-is. **Do not modify this file.** |
| `skill/handler/common.go` | `UseSkill`: generic buff step (l.175-179) buffs the caster; registry dispatch (l.194-201) runs after. **Do not modify this file.** |
| `character/buff/hidden.go` | `IsGmHidden(ctx, bs)` (l.21) — version-aware hide detection. Use this, never a `SourceId` literal compare. |
| `skill/handler/registrations/registrations.go` | One blank import to add. |
| `libs/atlas-constants/skill/identities_gen.go` | `BeginnerEchoOfHero` (l.11) + the other three identities. DOM-21: define nothing new. |
| `libs/atlas-constants/skill/version_*_gen.go` | Per-version `Id → Identity` maps — the source of truth for the availability table. |

## Locked decisions (design.md §2)

- **D1** Routing is a **registry handler** (`echoofhero/` package), not an inline
  `common.go` branch. The generic step buffs the caster only (X005 is not an
  `isPartyBuff`, so its bitmap is 0 and `applyToParty` no-ops on
  `recipients.go:236-238`); the handler covers everyone else. `common.go` diff must
  be empty.
- **D2** Reuse `SelectAllCharactersInMap`; filter caster/dead/hidden in the handler.
  No new selector.
- **D3** Hidden-GM check is `buff.IsGmHidden(ctx, bs)`. A raw
  `SourceId() == 9101004` compare is **version-incorrect** (hide is `5101004` at
  v0.48) and is the single largest correctness fix in the rebase.
- **D4** Uniform skip-and-continue on any per-recipient failure. Never abort the cast.
- **D5** No `isEchoOfHero` id predicate — registration is on the four Identity
  constants. An id predicate is both redundant and the banned idiom.
- **D6** Concurrent id sweep (inside the reused selector), sequential fetch/filter,
  recipients sorted ascending.
- **D7** `echoDeps` function-seam struct + pure `applyEchoOfHero` core, mirroring
  `healDispelDeps`.

## Version scope (11 versions — FR-5)

X005 availability is **not** version-stable (the v1 docs claimed it was):

| Version | Beginner | Noblesse | Legend | Evan |
|---|:---:|:---:|:---:|:---:|
| `gms_v12`, `gms_v48` | — | — | — | — |
| `gms_v61` | ✓ | — | — | — |
| `gms_v72` | ✓ | ✓ | — | — |
| `gms_v79`, `gms_v83` | ✓ | ✓ | ✓ | — |
| `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185` | ✓ | ✓ | ✓ | ✓ |

This is the **tenant version set (11)**, not the packet matrix's 9 columns (which
omit `gms_v12`/`gms_v92`) — irrelevant here because this is a zero-packet change.

**No per-version code is written.** On `gms_v48` wire `1005` is unbound, so
`set.Skill.Resolve` returns `ok == false`, `Lookup` is never called, and the handler
is inert. Registering all four identities is therefore correct on all 11 versions.
Task 2's four resolution tests pin this mechanically.

## Guards this task must satisfy

`tools/skill-job-id-guard.sh` (no version-divergent wire literals — the important
one), `tools/goroutine-guard.sh`, `tools/redis-key-guard.sh`, `tools/lint.sh --check`
(needs nvm 22 on PATH or it false-fails). No template guards apply — no template
changes.

## Test conventions in this package

- Deps-struct seams + `t.Cleanup` restore; no `*_testhelpers.go` files.
- `channelhandler.NewPartyRecipientBuilder().SetId(..).SetHp(..)...Build()` for
  recipients; `character.NewModelBuilder()...MustBuild()` for characters.
- `healdispel_test.go`'s `capture` struct + `newDeps(...)` helper is the shape to copy.
- `registry_test.go`'s `TestDispatch_v48HideNotCorkscrew` is the version-correctness
  test precedent for Task 2's resolution tests.

## Dependencies

- No new module deps — `go.mod` untouched, so `docker buildx bake atlas-channel` is
  not required (re-verify in Task 3 Step 5).
- Code review (`superpowers:requesting-code-review`) is mandatory before any PR;
  reviewers must run inside this worktree, pinned to the cheaper model.
