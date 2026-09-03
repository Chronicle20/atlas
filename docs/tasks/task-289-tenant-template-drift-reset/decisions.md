# task-289 — Scoping Decisions

Decisions taken during the spec interview (2026-09-02). The PRD references these
by number.

## D1 — Baseline is the live template row, not a clone-time snapshot

A tenant's baseline is the `templates` row matching its
`(region, majorVersion, minorVersion)` **as it stands at read time**.

Considered and rejected: pinning the template content (or its hash) onto the
tenant row at creation. That is a truer reading of "the template it was cloned
from", but it requires a new column plus a migration, and every tenant that
already exists has no snapshot to pin — they would report an unknown baseline
until manually re-pinned, which is exactly the population this feature exists to
serve.

Also rejected: live comparison plus a recorded `template_id`. The extra column
only pays off if a tenant's region/version can drift away from its template's,
which is not a scenario the UI produces today.

Consequence, accepted: editing a template retroactively makes its tenants report
drift with no tenant-side change. This is tolerable because it is *true* — the
tenant is genuinely behind its blueprint — and because task-201 already
established that a drift flag here is advisory and image-relative rather than
alertable (carried forward as NFR-4).

## D2 — Drift is reported per section, not as one boolean

The template resource reports a single `seedDrift`. The tenant resource reports a
flag per comparable section plus an aggregate.

The asymmetry is deliberate. A template's divergence from its shipped file is a
single question with a single answer ("is this row stale?"). A tenant's
divergence is not: a diverged `socket` section means a live client packet is
being dropped by the channel dispatcher, while a diverged `characters` section is
usually an operator's deliberate work. Collapsing both into one boolean produces
an indicator the operator cannot act on without diffing two JSON documents by
hand.

Rejected for now: a field-level diff preview. Section granularity is the
resolution that makes the reset decision, and a diff renderer for a document this
deeply nested is a separate piece of work.

## D3 — `worlds` is tenant-owned: excluded from both drift and reset

World configuration carries the server message, event message, and the exp / meso
/ item-drop / quest-exp rate multipliers. These are the fields an operator most
expects to differ from the blueprint, and the ones whose loss in a reset would be
most damaging.

Considered and rejected: reporting `worlds` drift but excluding it from reset.
Honest, but it means a permanently-red section on every real deployment, which
trains the operator to ignore the indicator — the failure mode NFR-5 exists to
prevent.

Also rejected: including it in both. Simplest rule, but it makes "reset to
baseline" an action that silently wipes live rates.

`diagnostics` (`tracePackets`) is excluded on different grounds: it is
tenant-only and has no template counterpart at all, so there is nothing to
compare it against.

## D4 — Reset is available both whole-document and per-section

`POST /configurations/tenants/{id}/reset` with no body resets every comparable
section; with a `sections` array it resets exactly those.

The whole-document form is the common case and must stay one click. The
per-section form exists for the specific operational need that motivated D2:
pulling in a newly-added socket opcode without discarding hand-authored character
presets.

Rejected: per-section only. It makes "get me back to baseline" six actions
instead of one, for a safety gain already provided by the confirm dialog and the
history record.

## D5 — The lossy clone is fixed as part of this task

`onboarding.service.ts` currently copies `region`, `majorVersion`,
`minorVersion`, `usesPin`, `characters`, `npcs`, `socket`, `worlds` and a
conditional `cashShop` from the template — it drops `mapleLife` entirely.

Left alone, every tenant created through the UI would report `mapleLife` drift
the moment it was created, violating NFR-5 on day one. Fixing the clone is a
prerequisite for the feature being usable, not a scope expansion, so it is
FR-5.1 rather than a follow-up.
