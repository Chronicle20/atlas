# Character Presets — UX Flow

Companion to `prd.md`. Describes the two operator surfaces exposed by the Atlas UI.

---

## A. Apply Preset to existing account

**Entry**: `AccountDetailPage` → header action **Add character from preset**. Visible only when the active tenant has at least one preset (FR-23).

**Dialog steps** (single shadcn `<Dialog>`, no multi-step wizard):

1. **Preset picker** — searchable combobox listing the active tenant's presets by `name` with secondary text `tags` + `jobId → job.Name`.
2. **World picker** — drop-down of worlds for the active tenant (uses existing world hook).
3. **Name input** — pre-filled with the preset's `defaultName`. As the operator types, the UI debounces a call to `GET /factory/characters/name-validity?name=…&worldId=…`; valid → submit button enabled, invalid → inline error with the helper's `detail` string.
4. **Submit** — `POST /factory/characters/from-preset`.

**Outcome**:

- `202 Accepted` → close dialog, `toast.success("Creating character… this may take a moment.")`, `queryClient.invalidateQueries({ queryKey: characterKeys.list(tenant) })`. The character appears once the saga completes (existing list refresh is sufficient — no manual polling needed for this surface).
- `400` / `409` → inline error in the dialog, dialog stays open so the operator can adjust.
- `404` (preset gone) / `5xx` → toast + dialog stays open.

---

## B. Admin Bootstrap wizard

**Entry**: `AccountsPage` → header action **Bootstrap Admin Account**.

**Wizard steps** (single shadcn `<Dialog>` with internal `step` state; cancellable at any time before step 4):

### Step 1 — Account credentials

Form fields:

- `name` (required; surface uniqueness check via `GET /accounts/?name=`)
- `password` (required, plain text input — atlas-account hashes it)

"Next" disabled until both fields valid.

### Step 2 — World + tag filter

- World drop-down.
- Tag selector populated from the union of tags across the active tenant's presets. Default selection: `4th-job`.
- Live preview list below: every preset whose `tags` intersects the selected tags is shown with: `name`, `jobId → job.Name`, `level`, `defaultName`, and a checkbox (default checked).

"Next" disabled if zero presets selected.

### Step 3 — Per-preset name overrides

Table of selected presets. For each row:

- **Name input** — pre-filled with `defaultName` (or empty + required if `defaultName` is blank).
- Live name-validity check via `GET /factory/characters/name-validity` against the chosen world.
- Conflict resolution: if two rows share the same name, both rows show a "duplicate within wizard" warning and the operator must change one. Server-side duplicate check is in addition to this client-side check.

"Apply" disabled until every name is valid and unique within the wizard.

### Step 4 — Apply

Disabled state for the wizard while the apply runs. Sequential pipeline:

1. **Create account** — `POST /accounts/` with `{ name, password }`. Response is `202 Accepted` (no body).
2. **Wait for account** — poll `GET /accounts/?name=<name>` every 1 s, up to 30 s. Surface a clear timeout error if the account does not appear; offer **Retry**.
3. **Apply each preset** — for each row in selected order:
   - Render the row with status `pending` → `applying…` → `success` (with character name) | `failed` (with error detail and per-row **Retry** button).
   - `POST /factory/characters/from-preset` with `{ presetId, accountId, worldId, name }`.
   - On `202`, optimistically mark `success`. The wizard does not wait for the saga to actually complete — the toast on the AccountsPage will refresh once the saga lands. Failures are wizard-row failures only when the synchronous response itself is non-2xx; saga-time compensation is invisible to the wizard (a follow-up improvement could add `transactionId` polling, but that is out of scope per FR-19).

After step 4 completes (all rows are `success` or the operator has dismissed), the wizard offers a **Done** button which closes the dialog and navigates to the new account's detail page.

### Cancellation semantics

- Cancellable freely in steps 1–3.
- Step 4 cannot be cancelled mid-flight per row; rows already submitted will continue server-side. Pending rows remain not-applied if the operator closes the wizard. No server-side resume — the operator can re-run the wizard with only the missing presets selected.

---

## C. Preset catalog editor (template + tenant)

Two near-identical pages following the existing form pattern (`templates-character-templates-form.tsx` is the closest precedent).

**Layout** (both `TemplatesCharacterPresetsPage` and `TenantsCharacterPresetsPage`):

- Top: **Add preset** button.
- Below: a list/accordion of preset cards. Each card has:
  - Header row: `name`, `tags` chips, `jobId → job.Name`, **Delete** icon (`X`).
  - Identity section: `name`, `description`, `tags` chip editor.
  - Character section: `jobId`, `gender`, `face`, `hair`, `hairColor`, `skinColor`, `mapId`, `level`, `meso`, `gm`, `defaultName`.
  - Stats section: 6 numeric inputs (`str`, `dex`, `int`, `luk`, `hp`, `mp`).
  - **Equipment** subsection: list of `{ templateId, useAverageStats }` rows with add/remove.
  - **Inventory** subsection: list of `{ templateId, quantity }` rows with add/remove.
  - **Skills** subsection: list of `{ skillId, level }` rows with add/remove.
- Bottom: **Save** button — submits the entire list via the corresponding `PUT` endpoint.

**Item / skill picker**: where the existing item search component is available, use it inline; otherwise free-text uint32 input with optional "lookup name" link to the existing `ItemDetailPage`/`SkillDetailPage`. Don't block the task on building a new picker.

**Validation feedback**: server-side validation errors from `400` responses are rendered per-preset and per-field using the JSON:API `errors[]` `meta.path` convention used by the existing forms.

---

## D. Navigation

- `TenantDetailPage` sidebar gains a **Character Presets** entry next to the existing **Character Templates**.
- `TemplateDetailPage` sidebar gains the same.
- Routes registered in `App.tsx`, using the same `/character/<thing>` namespace as the existing template routes (`/templates/:id/character/templates`, `/tenants/:id/character/templates`):
  - `/templates/:id/character/presets`
  - `/tenants/:id/character/presets`
- Breadcrumb integration via the existing `lib/breadcrumbs/` registry.
