# Step 5 — seed world-transfer reason-key aliases into the cash-shop errors tables

## What was implemented

Per `bug-world-transfer-eligibility-reasons.md` "Rulings" ruling 2 and its
alias table (later corrected by the coordinator's live StringPool decryption
— see "Correction" below):

1. **Seeded alias keys** into every seed template whose cash-shop `errors`
   table (found by content — the writer whose `errors` map contains
   `CANNOT_TRANSFER_TO_SAME_WORLD` — not by opCode/index, since the index
   differs per version) carries transfer codes:
   `template_gms_{48,61,72,79,83,84,87,92,95}_1.json`. Each alias resolves to
   *that template's own* numeric code for its anchor name — verified per
   template, never copied across templates (e.g.
   `CANNOT_TRANSFER_TO_SAME_WORLD` is 153/177/195/209/220/229/58/58 depending
   on version).

   Final mapping (after the correction):
   - `world_same` → `CANNOT_TRANSFER_TO_SAME_WORLD`
   - `no_character_slot` → `CANNOT_TRANSFER_NO_EMPTY_SLOTS`
   - `check_unavailable`, `unknown_error`, `world_full`, `world_unknown` →
     `PLEASE_TRY_AGAIN`
   - `name_taken`, `banned`, `is_guild_master`, `is_gm`, `in_family`,
     `trade_open`, `merchant_open`, `mts_listings_open` → `CANNOT_TRANSFER_OUT`

   Where a template's table lacks an anchor entirely, the corresponding
   alias(es) are **not seeded** rather than invented:
   - `gms_48_1`/`gms_61_1` have no `CANNOT_TRANSFER_NO_EMPTY_SLOTS` or
     `PLEASE_TRY_AGAIN` at all, so `no_character_slot`, `check_unavailable`,
     `unknown_error`, `world_full`, `world_unknown` are absent on those two.
   - `gms_72_1` has no `PLEASE_TRY_AGAIN`, so `check_unavailable`,
     `unknown_error`, `world_full`, `world_unknown` are absent there too.

2. **`template_gms_12_1.json`** (no `errors` table at all) and
   **`template_jms_185_1.json`** (an `errors` table with none of the
   `CANNOT_TRANSFER_*` names) are left **untouched** — there is no honest
   code to alias to on either. This is handled at the code level instead
   (see below); the seed-data side of that decision is simply "add nothing".

3. **Explicit, logged fallback in code** (this landed via a concurrent
   step-1 agent's commit that shared this worktree — see "Git note" below —
   but the change is mine and is described here for completeness):
   `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`
   gained `transferFailureReasonConfigured`, which asks the tenant's cash
   shop writer options (`writer.TenantWriterOptions` +
   `atlaspacket.CodeConfigured`, the same pattern `tradeEnterErrorConfigured`
   in `kafka/consumer/trade/consumer.go` and
   `interactionEnterErrorConfigured` in `character_interaction.go` already
   use) whether the tenant's `errors` table binds an alias for the reason
   about to be sent. `announceTransferWorldFailure` now logs a `Warnf`
   naming the reason key, the tenant's region/major/minor (the "which
   template" identity — options carry no literal template file name), and
   the character id when the check misses, **before** falling through to
   `CashShopTransferWorldFailedBody` → `ResolveCode`'s generic-notice path.
   This turns the `gms_12_1`/`jms_185_1` gap (and any future per-key gap)
   into a deliberate, observable decision instead of a silent 99.

4. **Test**: `world_transfer_alias_test.go` (new file, `templates` package)
   — `TestWorldTransferReasonAliasesResolveToAnchorCode` asserts every
   seeded alias resolves to the exact same code as its anchor, for every
   shipped template that carries a `CANNOT_TRANSFER_TO_SAME_WORLD`-bearing
   `errors` table; `TestCannotTransferToNewWorldHasNoAlias` is a regression
   guard that no key in any template's `errors` table shares
   `CANNOT_TRANSFER_TO_NEW_WORLD`'s code (see Correction);
   `TestWorldTransferUnmappableTemplatesCarryNoAliases` pins that
   `gms_12_1`/`jms_185_1` carry no `CANNOT_TRANSFER_TO_SAME_WORLD`-bearing
   table at all, by name.

## Correction applied mid-task

The coordinator decrypted the v83 client's StringPool table live and found
that `CANNOT_TRANSFER_TO_NEW_WORLD` (221 on `gms_83_1`, StringPool id 4006)
renders "You cannot transfer a character into the new server world" — the
**newest-world prohibition**, not "destination at capacity". There is no gate
implementing that rule, so nothing may alias it. I had originally aliased
`world_full`/`world_unknown` to it (per the brief's original table); I
removed those two entries from all nine templates and reseeded them as
aliases of `PLEASE_TRY_AGAIN` instead, only on the six templates that carry
`PLEASE_TRY_AGAIN` at all (`gms_79/83/84/87/92/95`) — `gms_48/61/72` correctly
get neither, same as their other missing-anchor gaps. Updated
`world_transfer_alias_test.go`'s alias-group table to match and added the
`TestCannotTransferToNewWorldHasNoAlias` regression guard. This is commit
`94df4ab86`.

## Git note — shared worktree

Per the coordinator's process note, two other agents were committing in this
same worktree concurrently. My first attempt at this task's commit (the
initial seeding, with the *incorrect* `world_full`/`world_unknown` → 
`CANNOT_TRANSFER_TO_NEW_WORLD` mapping, plus the
`cash_shop_operation.go` logging change) was swept into two different
concurrent commits by other agents rather than landing as its own commit —
first `c642cda59` ("fix(task-227): move storage-stranding warning from CHECK
to BUY_WORLD_TRANSFER", which picked up my `cash_shop_operation.go`
`transferFailureReasonConfigured`/`announceTransferWorldFailure` changes)
and then `196cf871d` ("feat(deploy): wire up the atlas-families deployment",
which picked up my seed-template and test-file changes). Both commits'
content is correct as of their landing, and the correction described above
(commit `94df4ab86`) is my own explicit commit, staged file-by-file
(`git add <path>`, never `git add -A`/`git commit -a`), verified with
`git status --short` before committing. `git branch --show-current` and
`git rev-parse --show-toplevel` were checked after each commit and both
report the expected branch/worktree throughout.

I did not attempt to rewrite or split apart the two commits my earlier work
landed in — the other agents were actively committing, and any history
rewrite risked colliding with their concurrent work. All content is present,
correct (after the mid-task correction), and covered by tests at current
HEAD.

## Files changed (by me)

- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`
  — `transferFailureReasonConfigured` + logged fallback in
  `announceTransferWorldFailure` (content landed via commit `c642cda59`,
  authored by this task's work).
- `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_61_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_72_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_79_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- `services/atlas-configurations/atlas.com/configurations/templates/world_transfer_alias_test.go`
  (new)

`template_gms_12_1.json` and `template_jms_185_1.json` are intentionally
untouched.

## What I did not touch

`services/atlas-character/atlas.com/character/pending_change/processor_eligibility.go`
— explicitly out of scope per the brief, owned by another agent.

## Testing

```
cd services/atlas-configurations/atlas.com/configurations
go build ./...
go test ./...
```
All packages `ok` (or `no test files`); `templates` package specifically:

```
=== RUN   TestWorldTransferReasonAliasesResolveToAnchorCode
--- PASS: TestWorldTransferReasonAliasesResolveToAnchorCode (0.03s)
=== RUN   TestCannotTransferToNewWorldHasNoAlias
--- PASS: TestCannotTransferToNewWorldHasNoAlias (0.02s)
=== RUN   TestWorldTransferUnmappableTemplatesCarryNoAliases
--- PASS: TestWorldTransferUnmappableTemplatesCarryNoAliases (0.03s)
PASS
ok  	atlas-configurations/templates	0.085s
```

```
cd services/atlas-channel/atlas.com/channel
go build ./...
go test ./...
```
All packages `ok`.

```
cd libs/atlas-packet
go build ./...
go test ./...
```
All packages `ok` (unchanged; referenced `CodeConfigured`/`ResolveCode` but
did not modify this module).

## Self-review

- No schema or test pinned the `errors` table's entry count or key set
  beyond what `world_transfer_alias_test.go` now covers — checked
  `resource_reseed_test.go`, `processor_test.go`, `socket_validation_test.go`
  in `atlas-configurations`; none reference the cash-shop `errors` map.
- Did not invent any code: every seeded alias points at a name that already
  existed in that template's own `errors` table before this change; every
  gap (missing anchor on a given template, or a whole template with no
  transfer codes) is either omitted or handled by the explicit runtime
  fallback log — never a guessed value.
- Followed the existing `*Configured` predicate pattern
  (`tradeEnterErrorConfigured`, `interactionEnterErrorConfigured`) rather
  than inventing a new mechanism, and used `writer.TenantWriterOptions` +
  `atlaspacket.CodeConfigured`, both already exported for exactly this use.
- JSON formatting preserved (LF line endings, existing indentation); each
  file re-validated with `python3 -c "import json; json.load(...)"` after
  every edit pass.

## Concerns

- The two commits my earlier (pre-correction) work landed in are not my own
  — `c642cda59` and `196cf871d` — because of the shared-worktree race
  described above. Their content is correct as of current HEAD (folded with
  my correction commit `94df4ab86`), and `go build`/`go test` pass at HEAD,
  but the commit boundaries do not cleanly separate "step 1" from "step 5"
  the way the task brief implies they should. Worth flagging to the
  controller in case the PR history needs cleanup before merge.
