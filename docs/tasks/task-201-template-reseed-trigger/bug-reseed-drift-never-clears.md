# Bug: re-seeded templates still report `seedDrift` on every row

Task: task-201-template-reseed-trigger (shipped in #1260, `30f795668`)
Branch: `fix/task-201-reseed-drift`
Worktree: `.worktrees/task-201-reseed-drift-fix/`

## Reproduced

Yes — deterministically, in-repo, without a cluster.

Temporary test in `services/atlas-configurations/atlas.com/configurations/templates`:
parse `seed-data/templates/template_gms_83_1.json` into a `RestModel`, take
`Revision(rm)` as the shipped side; then run the exact re-seed write path —
`canonicalBytes(rm)` into an `Entity`, `Make(entity)`, `Revision(...)` — as the
stored side.

```
Environment: "main"  -> shipped=34c8a2b18000 stored=6fd13d0f0b79 equal=false
Environment: ""      -> shipped=34c8a2b18000 stored=34c8a2b18000 equal=true
```

The only variable changed between the two runs is `Entity.Environment`.

## Observed

On `atlas-main`, every template row shows the drift indicator, and
"reset to shipped defaults" does not clear it — for any template, on repeat.

## Expected

Immediately after `ReseedById`, the row's content is byte-identical to the
shipped file, so `storedRevision == shippedRevision` and `seedDrift == false`.

## Root cause

`RestModel.Environment` is a **server-owned** field (see the comment on the
field in `rest.go` and in `Make`) but it carries a JSON tag, so it is included
in the SHA-256 that `Revision` computes. `Revision` clears `Id` and normalizes
`Socket`, but leaves `Environment` alone
(`templates/revision.go:20-30`).

The two sides of the drift comparison then see different values for it:

- **Shipped side** — `LoadCatalog` (`templates/shipped.go:105`) unmarshals the
  seed file. No seed file under `services/atlas-configurations/seed-data/templates/`
  contains an `environment` key (verified across all 11 files), so
  `entry.Model.Environment == ""` and `entry.Revision` hashes `"environment":""`.
- **Stored side** — `makeView` (`templates/processor.go:96-110`) hashes the
  `RestModel` that `Make` produced, and `Make` unconditionally overwrites
  `rm.Environment = e.Environment` from the entity column
  (`templates/processor.go:139`).

On `atlas-main`, `deploy/k8s/overlays/main/kustomization.yaml:48` sets
`ATLAS_ENVIRONMENT=main`, so `Create` stamps `Environment: "main"` into the
column for every row. `"main" != ""` ⇒ the hashes can never agree ⇒
`SeedDrift` is `true` for **every** template, permanently.

`ReseedById` cannot fix this by construction: it writes
`canonicalBytes(entry.Model)`, whose blob contains `"environment":""`, but the
next read goes through `Make`, which replaces it with the column value again.
The write succeeds (the NFR-3 log line is emitted with the correct
`afterRevision`); the badge simply recomputes to the same mismatch.

Why the existing tests missed it: every unit test builds the processor over
`context.Background()`, whose `env.MustFromContext` is the legacy `""`. With
`Environment == ""` on both sides the bug is invisible — the second line of the
repro above is exactly the test suite's condition. The base
`deploy/k8s/base/env-configmap.yaml` also ships `ATLAS_ENVIRONMENT: ""`, so
only the `main` (and `pr-*`) overlays exhibit it.

Ruled out:
- The re-seed write itself. `update(...)` is scoped by the entity's own
  region/version and the row does get rewritten; drift is a **read-path**
  computation.
- Socket nil-vs-empty normalization — `Revision` normalizes both sides.
- `Id` — already cleared in `Revision`.
- Catalog loading / duplicate keys — the catalog resolves correctly; the
  shipped revision is computed, just over a different `environment` value.
- The frontend. `templates-columns.tsx:65` only renders `seedDrift === true`
  straight off the server contract; no client-side comparison exists.

## Fix

Exclude the server-owned `Environment` from the content hash, exactly as `Id`
already is. It is deployment metadata, not template content — two rows that
differ only by which environment owns them are not "different from shipped".

- `services/atlas-configurations/atlas.com/configurations/templates/revision.go`
  — clear `rm.Environment` alongside `rm.Id` in `Revision`, and extend the
  doc comment to say *why* (server-owned, set from `Entity.Environment` by
  `Make`, absent from every shipped file, so hashing it makes drift permanent
  on any non-empty-environment deployment).
- `services/atlas-configurations/atlas.com/configurations/templates/revision_test.go`
  — add a regression test asserting `Revision` is invariant under
  `Environment` (same model, `""` vs `"main"` vs `"pr-123"` ⇒ same hash).
- `services/atlas-configurations/atlas.com/configurations/templates/processor_test.go`
  — add an end-to-end regression over the real re-seed path with a **non-empty**
  `Entity.Environment`: seed a row whose stored bytes are
  `canonicalBytes(shipped)`, stamp `Environment: "main"`, and assert the view
  reports `seedDrift == false` and `storedRevision == shippedRevision`. This is
  the assertion whose absence let the bug ship — existing drift tests all run at
  `Environment: ""`.

Do **not** change `RestModel` (dropping the JSON tag would change the REST
contract and the stored document shape), and do **not** change
`canonicalBytes` or `Make`.

## Not yet answered

- Whether any *other* server-owned or read-time-derived field could enter the
  hash later. The fix should make `Revision`'s contract explicit in its comment
  ("only client-authored content is hashed") so the next such field is caught
  at review rather than in production.

## Outcome

_(to be filled in: fix commit, gate verdict, live re-test on atlas-main)_
