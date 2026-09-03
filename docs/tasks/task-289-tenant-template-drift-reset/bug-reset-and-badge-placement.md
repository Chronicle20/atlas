# bug-reset-and-badge-placement

UI polish feedback on the task-289 tenant drift/reset surfaces. Not a functional
defect — layout inconsistency reported after implementation.

## Reproduced

Tenant detail pages on branch `task-289-tenant-template-drift-reset`
(`/tenants/:id/...`), any tenant whose configuration reports drift.

## Observed

1. The "Differs from template: ..." badge renders inside the right-hand action
   cluster of `TenantDetailLayout`, crowded against `ConfigExportButton` and the
   whole-document `TenantResetButton`
   (`services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx:63-76`).
2. Per-section reset buttons sit in different places per sub-section:
   - Global Properties: bottom of the form, in the same row as **Save**
     (`services/atlas-ui/src/pages/tenants-properties-form.tsx:186-192`).
   - Character Templates / Character Presets / Maple Life / Socket Handlers /
     Socket Writers: a bare `<div className="flex justify-end">` at the top of
     the sub-section content, with no spacing between it and the editor below.

## Expected

1. The drift badge belongs on the left of the page header, with the "Tenant
   Details" heading and the tenant id — not in the action cluster.
2. Every sub-section that offers a scoped reset places that button in the same
   spot with the same spacing. Top-right of the sub-section content, above the
   editor, with padding separating it from the content below; Global Properties
   moves to match (its footer keeps only **Save**).

## Root cause

Each sub-section page grew its own inline reset row during Phase 4; there is no
shared placement primitive, and the properties form placed its reset next to the
existing submit button instead. The badge was appended to the action cluster
because that div already existed.

## Fix

Introduce one shared placement component and route every scoped reset through it.

- `services/atlas-ui/src/components/features/tenants/TenantSectionResetBar.tsx`
  (new): thin wrapper — `<div className="flex justify-end pb-4">` around
  `TenantResetButton`, same props (`id`, `sections`, `sectionLabel`). This is the
  single place section-reset placement is defined.
- `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx`:
  move the drift `Badge` (with its `Tooltip`) out of the right-hand action
  cluster into the left header block, beneath the tenant id `<p>`. Keep the
  `driftedSections` derivation, the strict `=== true` gating, and the tooltip
  copy exactly as-is. The action cluster keeps `ConfigExportButton` and the
  whole-document `TenantResetButton`.
- `services/atlas-ui/src/pages/TenantsCharacterTemplatesPage.tsx`,
  `TenantsCharacterPresetsPage.tsx`, `TenantsMapleLifePage.tsx`,
  `TenantsHandlersPage.tsx`, `TenantsWritersPage.tsx`: replace the inline
  `<div className="flex justify-end"><TenantResetButton .../></div>` with
  `<TenantSectionResetBar ... />`, unchanged `sections`/`sectionLabel` values.
- `services/atlas-ui/src/pages/tenants-properties-form.tsx`: remove the
  `TenantResetButton` from the submit row (leaving `Save` alone in
  `flex flex-row gap-2 justify-end`) and render
  `<TenantSectionResetBar id={id} sections={["properties"]}
  sectionLabel="global properties" />` above the `<Form>` so both
  `TenantsPropertiesPage` and `TenantDetailPage` pick it up from one place.
- Tests to update/extend:
  `services/atlas-ui/src/pages/__tests__/TenantsSectionReset.test.tsx`,
  `services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx`,
  `services/atlas-ui/src/pages/__tests__/TenantsCharacterTemplatesPage.test.tsx`,
  `services/atlas-ui/src/pages/__tests__/TenantsCharacterPresetsPage.test.tsx`.
  Assertions that depend on the old placement must assert the new one; add
  coverage that the properties sub-section still renders a scoped reset and that
  the drift badge is inside the header block.

Behaviour, copy, section scoping, and the reset mutation are unchanged — this is
placement only.

## Not yet answered

- None blocking. Badge placement chosen as "beneath the tenant id in the left
  header block"; reset bar placement chosen as "top-right of sub-section content
  with `pb-4`" — both are direct readings of the feedback.

## Resolution

Fixed in `32723e0d7` (branch `task-289-tenant-template-drift-reset`).
`tools/verify.sh --quick --base c2c720c5f` exited 0; the full atlas-ui vitest
suite (283 files / 2400 tests) passed separately, covering what `--quick` skips.
The flagless gate still owes a pre-PR run. Not yet confirmed by live testing in
the browser.
