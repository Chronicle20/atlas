# Third-event walkthrough (task-39, FR-20.9)

This file copies design.md §16 verbatim, then checks each claim against the
code as it stands after 38 landed tasks, not as designed. Divergences are
called out inline rather than silently corrected — that comparison is the
deliverable acceptance 20.9 asks for, not the walkthrough text itself.

---

## §16 as designed (verbatim copy)

> ## 16. Architecture validation: adding a third event (FR-20.9)
>
> The walkthrough PRD §20.9 requires. Third event: **Mysterious Merchant** — a
> travelling NPC that appears in one randomly chosen town map for a bounded window,
> several times a day. It is deliberately unlike both shipped events: recurring
> rather than one-shot or externally triggered, spatially scoped to a map it picks
> itself rather than one it is handed, and it commands a domain (`atlas-npcs`)
> neither shipped event touches.
>
> What it needs:
>
> 1. `events/mysteriousmerchant/config.go` — `candidateMapIds`, `appearancesPerDay`,
>    `duration`, `npcId`.
> 2. `events/mysteriousmerchant/handler.go` implementing `registry.Handler`:
>    - `ValidateConfiguration` — non-empty candidates, positive duration.
>    - `ConcurrencyKey` — a constant, so at most one merchant is abroad at a time.
>    - `Evaluate` — pick a map, return an `occurrence.Seed` with it; also schedule
>      the day's next `TRIGGER_EVALUATION`, which is how "recurring" is expressed
>      without any new scheduling primitive.
>    - `Start` — emit an NPC-spawn command; return `Progress` with
>      `nextTransitionAt = now + duration`.
>    - `Advance` — terminal; complete with `WINDOW_ELAPSED`.
>    - `Complete` — emit an NPC-despawn command.
> 3. One line in `main.go`: `registry.Register(mysteriousmerchant.NewHandler())`.
> 4. A seed JSON file.
> 5. A new NPC command in `kafka/message/npc/` — a domain command, owned by the
>    event package.
>
> What it does **not** need, which is the actual claim: no change to
> `event/definition`, `event/occurrence`, `event/transition`, `event/scheduling`,
> the poller, the REST resources, the UI page structure, the database schema, or
> any gameplay service other than the one it commands. The occurrence's map scope
> rides `event_occurrence_map`; its recurrence rides `TRIGGER_EVALUATION`; its
> window rides `OCCURRENCE_TRANSITION`; its exclusivity rides `concurrency_key`.
> Each is a generic mechanism the shipped events already exercise from a different
> angle.
>
> The one thing it would add to the generic layer is a UI detail panel, and that is
> by design a per-type component (§13) rather than an edit to a shared switch.

---

## Verification against the code as it now stands

### Item 1 — `events/mysteriousmerchant/config.go`

Confirmed pattern still holds. Both shipped events follow this exact shape:
`events/crimsonbalrog/config.go` and `events/anniversary/config.go` each hold a
`Config` struct, a `DecodeConfig(json.RawMessage) (Config, error)`, and a
`Validate()` method. A `mysteriousmerchant/config.go` with
`candidateMapIds`/`appearancesPerDay`/`duration`/`npcId` fits without
modification to any generic type.

### Item 2 — `events/mysteriousmerchant/handler.go` implementing `registry.Handler`

**Diverges from the design as written.** `event/registry/handler.go`'s current
`Handler` interface (see doc comment on the `Advance` method, and the package
comment at the top of the file) has **no separate `Complete` method**:

```go
type Handler interface {
    Type() string
    ValidateConfiguration(raw json.RawMessage) error
    ConcurrencyKey(ctx context.Context, workContext json.RawMessage) (string, error)
    ConcurrencyKeyIsConstant() bool
    Evaluate(ctx context.Context, d Definition, w Work) (*Seed, error)
    Start(ctx context.Context, o Occurrence) (Progress, error)
    Advance(ctx context.Context, o Occurrence, w Work) (Progress, error)
}
```

The doc comment on `Advance` explains why: "there is no separate `Complete`
method: both real implementations converge on `occurrence.Processor.Complete`
directly, and a `Handler.Complete` would have zero production callers." So a
real Mysterious Merchant handler would express "terminal, complete with
`WINDOW_ELAPSED`, emit an NPC-despawn command" as the **terminal branch of
`Advance`** (returning `Progress{Terminal: true, CompletionReason:
"WINDOW_ELAPSED"}` after emitting the despawn command), the same way
`events/crimsonbalrog/arrival.go` and `events/anniversary` fold their
completion side effects into the method that observes termination rather than
a dedicated `Complete` hook. The design's item-2 bullet list (`Start` /
`Advance` / `Complete` as three separate lifecycle points) does not match the
interface that was actually built; `ConcurrencyKeyIsConstant()` — also absent
from the design's bullet list — is a required method too (added per task-231
R33-4, documented in the interface's doc comment), and Mysterious Merchant
would return `true` since its concurrency key is the constant the design
already calls for.

Everything else in item 2 holds: `ValidateConfiguration`, `ConcurrencyKey`,
`Evaluate` (returning a `*registry.Seed`, not `*occurrence.Seed` — `Seed` is
defined in `event/registry/handler.go`, not in `event/occurrence`; the design's
prose says "`occurrence.Seed`" but the actual type lives in the `registry`
package) all exist on the interface exactly as described.

### Item 3 — one line in `main.go`

**Confirmed, with a caveat that only applies if the event needs domain
consumers.** `main.go` today registers both shipped handlers as single lines:

```go
registry.Register(crimsonbalrog.NewHandler())
...
registry.Register(anniversary.NewHandler(db))
```

For a handler with no constructor dependencies beyond `db` (or none), one line
suffices — confirmed. However, Mysterious Merchant's own design description
(§16 body: "it commands a domain neither shipped event touches") implies it
only *emits* commands and does not *consume* any status events, so no
additional `kafka/consumer/<domain>` wiring is implied by §16 itself. Worth
noting for the record: both shipped events that DO react to external domain
events (`crimsonbalrog` to monster status, `anniversary` to character login)
each required a **second** kind of `main.go` change beyond `registry.Register`
— a pair of `InitConsumers`/`InitHandlers` calls against a new top-level
`kafka/consumer/<domain>` package (`kafka/consumer/monsterstatus`,
`kafka/consumer/characterstatus`), not something owned inside
`events/crimsonbalrog` or `events/anniversary` themselves. Mysterious
Merchant, as designed, does not need this because it only emits an NPC-spawn/
despawn command — it never reacts to an NPC-domain event coming back. If a
future event needed such a reaction, the true cost is "one line plus one new
consumer package plus a `main.go` wiring block," not just "one line."

### Item 4 — a seed JSON file

Confirmed the mechanism exists, at a path the design leaves unstated. The
shipped events' seed files live at repo root:
`deploy/seed/shared/all/events/definitions/event-crimson-balrog.json` and
`deploy/seed/shared/all/events/definitions/event-anniversary.json`, resolved
by `event/definition/subdomain.go`'s `InitSeedResource` via
`seeder.NewFilesystemCatalogSourceWithShared("SEED_CATALOG_ROOT",
"./deploy/seed", "shared/all")`. A `mysteriousmerchant` seed file would land at
`deploy/seed/shared/all/events/definitions/event-mysterious-merchant.json`,
outside `services/atlas-events/` entirely (under the monorepo-root `deploy/`
tree). This does not contradict the design's claim ("a seed JSON file" is
accurate) but the design does not say where; recorded here so the fourth item
is unambiguous rather than merely "believed true."

### Item 5 — a new NPC command in `kafka/message/npc/`

Confirmed the pattern. `events/crimsonbalrog/producer.go` builds its
`SpawnFieldCommandBody`/`DestroyBySourceCommandBody` messages from
`kafka/message/monster`, a sibling top-level message package, and calls
`producer.SingleMessageProvider` directly from inside the event package — no
generic-layer involvement. A `kafka/message/npc` package with
`SpawnCommandBody`/`DespawnCommandBody`, built and emitted the same way from
inside `events/mysteriousmerchant/`, fits the established shape exactly.

### The "does not need" claim

Confirmed for the generic layer proper. Nothing under `event/definition`,
`event/occurrence`, `event/transition`, `event/scheduling`,
`event/orchestration`, or `event/registry` would need to change to add
Mysterious Merchant — this is exactly the property `event/boundary_test.go`
(this task) now pins mechanically: the generic layer contains no literal
naming `CRIMSON_BALROG` or `ANNIVERSARY` in a non-comment position today, and
the same AST walk would catch a `MYSTERIOUS_MERCHANT` literal landing there in
the future.

Two items the design's "does not need" list omits, found while checking the
code:

- **The poller, REST resources, and schema are untouched, confirmed** — no
  divergence there.
- **A new top-level `kafka/consumer/<domain>` package is a real, recurring
  cost the design's item list doesn't surface**, as detailed under Item 3
  above. It happens not to apply to Mysterious Merchant specifically (it has
  no inbound domain reaction), but the walkthrough's own framing ("what it
  does not need... any gameplay service other than the one it commands")
  undersells how much wiring a *reactive* third event (one that listens for a
  domain status event, the way `crimsonbalrog` and `anniversary` both do)
  would actually require. A future event needing NPC-domain feedback (e.g.
  "despawn early if the NPC is killed") would need that extra package plus a
  second `main.go` wiring block, not "no change... to any gameplay service
  other than the one it commands."

### Divergences summary

| Design claim | Status | Note |
|---|---|---|
| `Handler` has `Start`/`Advance`/`Complete` as three methods | **Diverged** | No `Complete` method exists; terminal completion is the terminal branch of `Advance`. `ConcurrencyKeyIsConstant()` is also a required method the design's bullet list omits. |
| `Evaluate` returns `*occurrence.Seed` | **Diverged (naming only)** | The type is `registry.Seed`, defined in `event/registry/handler.go`. |
| One line in `main.go` registers the handler | **Confirmed** | True for `registry.Register`; a handler with inbound domain consumption needs a second, undescribed wiring cost (new `kafka/consumer/<domain>` package + `main.go` block). |
| A seed JSON file | **Confirmed, path unstated** | `deploy/seed/shared/all/events/definitions/event-<slug>.json`, at the monorepo root, outside `services/atlas-events/`. |
| A new NPC command in `kafka/message/npc/`, owned by the event package | **Confirmed** | Matches `kafka/message/monster` + `events/crimsonbalrog/producer.go` exactly. |
| No change to `event/definition`, `event/occurrence`, `event/transition`, `event/scheduling`, the poller, REST resources, UI page structure, or the schema | **Confirmed** | Mechanically pinned by `event/boundary_test.go` (this task) for the type-literal dimension. |
| No change needed to "any gameplay service other than the one it commands" | **True for Mysterious Merchant specifically, overstated as a general claim** | Both shipped events that react to a domain (not just command one) each required a new top-level `kafka/consumer/<domain>` package. |

### Files inspected for this walkthrough

- `services/atlas-events/atlas.com/events/event/registry/handler.go`
- `services/atlas-events/atlas.com/events/event/registry/registry.go`
- `services/atlas-events/atlas.com/events/main.go`
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/handler.go`
- `services/atlas-events/atlas.com/events/events/crimsonbalrog/producer.go`
- `services/atlas-events/atlas.com/events/kafka/consumer/monsterstatus/consumer.go`
- `services/atlas-events/atlas.com/events/kafka/consumer/characterstatus/consumer.go`
- `services/atlas-events/atlas.com/events/event/definition/subdomain.go`
- `deploy/seed/shared/all/events/definitions/event-crimson-balrog.json` (existence confirmed via `find`; contents not opened — path only)
- `deploy/seed/shared/all/events/definitions/event-anniversary.json` (existence confirmed via `find`; contents not opened — path only)
