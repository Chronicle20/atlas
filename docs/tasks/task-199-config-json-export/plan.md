# Template / Tenant Configuration JSON Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an **Export** button to the Template Details and Tenant Details page headers that downloads the viewed configuration as a pretty-printed, seed-template-shaped `.json` file.

**Architecture:** Four units, each independently testable — a DOM-only `downloadJson` helper, a pure `config-export` projection/naming module, a new read accessor on the existing `DetailActionBarContext`, and a shared `ConfigExportButton` that wires them to React Query, `sonner` toasts and shadcn `Button`/`Tooltip`. The button is mounted once per layout, so it is present on every sub-tab with no per-page wiring. No backend change and no new network path.

**Tech Stack:** TypeScript, React 19, React Router, TanStack React Query v5, shadcn/ui (Radix), lucide-react, sonner, Vitest + @testing-library/react (jsdom).

## Global Constraints

- All changes live in `services/atlas-ui/`. No Go file, no seed template, no k8s manifest, no `services.json` entry. The branch diff must contain only `services/atlas-ui/**` and `docs/**`.
- No new endpoint, no new service method, no change to `templates.service.ts` / `tenants.service.ts` behaviour.
- Frontend conventions (`services/atlas-ui/CLAUDE.md`): named exports only, `@/` alias imports, no `next/*` imports, shadcn `Button` + lucide icons, `sonner` `toast`.
- Tests use `vi.*` (never `jest.*`). Vitest runs with `globals: true`, `environment: "jsdom"`, and only collects `src/**/*.test.{ts,tsx}` (`vitest.config.ts:27-35`).
- Exported JSON: `JSON.stringify(payload, null, 2)` + a single trailing `"\n"`, MIME `application/json`.
- Exported filenames: `template_<region>_<major>_<minor>.json` / `tenant_<region>_<major>_<minor>.json`, region lowercased with every character outside `[a-z0-9]` replaced by `_`; fallback `template_<id>.json` / `tenant_<id>.json`.
- Gate commands, run from the worktree root at the end (Task 5):
  - `cd services/atlas-ui && npm run test`
  - `cd services/atlas-ui && npm run build` (this is the type-check; it also type-checks test files)
  - `tools/lint.sh --check` (needs nvm 22 on PATH)

## Deviations from design.md (verified during planning — implement as written here)

1. **No extra `TooltipProvider`.** design.md §2.6 says `ConfigExportButton` must mount its own `TooltipProvider`. The project's `Tooltip` wrapper already renders a `TooltipProvider` internally (`src/components/ui/tooltip.tsx:20-28`), so `<Tooltip>` alone is sufficient and a second provider would be dead code.
2. **Explicit filename meta at the call site.** design.md §3.4 sketches `configExportFilename(kind, { id: data.id, ...data.attributes })`. The plan passes `region` / `majorVersion` / `minorVersion` explicitly instead, so the argument is a plain literal with no reliance on TypeScript's excess-property rules for spreads.
3. **`toConfigExportPayload` builds through a `Record<string, unknown>` and casts once on return.** Assigning to a property of a generic `T` is a TypeScript error; building a record and asserting `as T` at the boundary is the only way to keep the generic signature the design specifies.

Everything else follows design.md exactly.

---

## File Structure

| File | Responsibility |
|---|---|
| `services/atlas-ui/src/lib/utils/download-json.ts` (new) | Serialise + hand a Blob to the browser download mechanism. Knows nothing about configurations. |
| `services/atlas-ui/src/lib/utils/config-export.ts` (new) | Pure payload projection (`toConfigExportPayload`) and filename derivation (`configExportFilename`). No React, no DOM. |
| `services/atlas-ui/src/components/DetailActionBarContext.tsx` (modify) | Add `useDetailActionBarState()` read accessor. |
| `services/atlas-ui/src/components/features/config/ConfigExportButton.tsx` (new) | The button: query wiring, disabled predicate, dirty tooltip, toasts. |
| `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx` (modify) | Header flex row + `<ConfigExportButton kind="template" …/>`. |
| `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx` (modify) | Header flex row + `<ConfigExportButton kind="tenant" …/>`. |
| `.../src/lib/utils/__tests__/download-json.test.ts` (new) | Blob body, MIME, filename, revoke, no-leak-on-throw. |
| `.../src/lib/utils/__tests__/config-export.test.ts` (new) | Envelope, normalisation, sort, key order, filename cases. |
| `.../src/components/features/config/__tests__/ConfigExportButton.test.tsx` (new) | Disabled state, click path, toasts, no-refetch-on-click. |
| `.../src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx` (new) | Header renders an `Export` control. |
| `.../src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx` (new) | Header renders an `Export` control. |

All paths below are relative to `services/atlas-ui/` unless stated otherwise. Run `npm` commands from `services/atlas-ui/`; run `git` commands from the worktree root.

---

### Task 1: `downloadJson` helper

**Files:**
- Create: `services/atlas-ui/src/lib/utils/download-json.ts`
- Test: `services/atlas-ui/src/lib/utils/__tests__/download-json.test.ts`

**Interfaces:**
- Consumes: nothing.
- Produces: `export function downloadJson(filename: string, payload: unknown): void` — serialises `payload` as `JSON.stringify(payload, null, 2)` plus a trailing `"\n"`, wraps it in an `application/json` Blob, and triggers a download named `filename`. Throws whatever `JSON.stringify` throws, without having created an object URL.

- [x] **Step 1: Write the failing test**

Create `src/lib/utils/__tests__/download-json.test.ts`:

```ts
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { downloadJson } from "@/lib/utils/download-json";

describe("downloadJson", () => {
  let createObjectURL: ReturnType<typeof vi.fn>;
  let revokeObjectURL: ReturnType<typeof vi.fn>;
  let blobs: Blob[];
  let clickSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    // jsdom implements neither of these (src/test/setup.ts stubs only
    // matchMedia and ResizeObserver), so they are stubbed per-suite rather
    // than globally - a global stub would silently mask their absence in
    // unrelated suites.
    blobs = [];
    createObjectURL = vi.fn((blob: Blob) => {
      blobs.push(blob);
      return "blob:mock-url";
    });
    revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL,
      revokeObjectURL,
    });
    clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, "click")
      .mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    clickSpy.mockRestore();
  });

  it("writes a pretty-printed JSON blob with a trailing newline", async () => {
    downloadJson("template_gms_83_1.json", { region: "GMS", n: [1, 2] });

    expect(createObjectURL).toHaveBeenCalledTimes(1);
    expect(blobs).toHaveLength(1);
    expect(blobs[0].type).toBe("application/json");
    await expect(blobs[0].text()).resolves.toBe(
      `${JSON.stringify({ region: "GMS", n: [1, 2] }, null, 2)}\n`,
    );
  });

  it("clicks an anchor carrying the requested filename", () => {
    let downloadAttr: string | null = null;
    clickSpy.mockImplementation(function (this: HTMLAnchorElement) {
      downloadAttr = this.download;
    });

    downloadJson("tenant_gms_83_1.json", {});

    expect(clickSpy).toHaveBeenCalledTimes(1);
    expect(downloadAttr).toBe("tenant_gms_83_1.json");
  });

  it("revokes the object URL and leaves no anchor behind", () => {
    downloadJson("template_gms_83_1.json", {});

    expect(revokeObjectURL).toHaveBeenCalledTimes(1);
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
    expect(document.body.querySelector("a")).toBeNull();
  });

  it("propagates a serialisation failure without creating an object URL", () => {
    const cyclic: Record<string, unknown> = {};
    cyclic.self = cyclic;

    expect(() => downloadJson("x.json", cyclic)).toThrow();
    expect(createObjectURL).not.toHaveBeenCalled();
    expect(revokeObjectURL).not.toHaveBeenCalled();
    expect(document.body.querySelector("a")).toBeNull();
  });
});
```

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/utils/__tests__/download-json.test.ts`
Expected: FAIL — cannot resolve `@/lib/utils/download-json`.

- [x] **Step 3: Write the implementation**

Create `src/lib/utils/download-json.ts`:

```ts
/**
 * Hand a JSON payload to the browser's download mechanism.
 *
 * Serialisation happens BEFORE createObjectURL, so a throw (cyclic structure,
 * BigInt) cannot leak an object URL - there is nothing to revoke yet. The
 * anchor teardown and the revoke live in a `finally` so a throw from click()
 * cannot leak either.
 *
 * The trailing newline matches the checked-in seed files under
 * services/atlas-configurations/seed-data/templates/, which all end with one.
 */
export function downloadJson(filename: string, payload: unknown): void {
  const body = `${JSON.stringify(payload, null, 2)}\n`;
  const url = URL.createObjectURL(
    new Blob([body], { type: "application/json" }),
  );
  const anchor = document.createElement("a");
  try {
    anchor.href = url;
    anchor.download = filename;
    document.body.appendChild(anchor);
    anchor.click();
  } finally {
    // remove() rather than removeChild() - a no-op when the append never
    // happened, which keeps this block total.
    anchor.remove();
    URL.revokeObjectURL(url);
  }
}
```

- [x] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/lib/utils/__tests__/download-json.test.ts`
Expected: PASS — 4 tests.

- [x] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/utils/download-json.ts \
        services/atlas-ui/src/lib/utils/__tests__/download-json.test.ts
git commit -m "feat(ui): add downloadJson helper for client-side JSON export"
```

---

### Task 2: `config-export` projection and filename

**Files:**
- Create: `services/atlas-ui/src/lib/utils/config-export.ts`
- Test: `services/atlas-ui/src/lib/utils/__tests__/config-export.test.ts`

**Interfaces:**
- Consumes: nothing (pure module — no React, no DOM, no imports from the configuration domain beyond types).
- Produces:
  - `export type ConfigExportKind = "template" | "tenant"`
  - `export interface ConfigExportMeta { id: string; region?: string; majorVersion?: number; minorVersion?: number }`
  - `export interface ExportableConfigAttributes { region?: string; majorVersion?: number; minorVersion?: number; npcs?: unknown[] | null; worlds?: unknown[] | null; socket?: { handlers?: { opCode: string }[]; writers?: { opCode: string }[] } | null }`
  - `export function toConfigExportPayload<T extends ExportableConfigAttributes>(attributes: T): T`
  - `export function configExportFilename(kind: ConfigExportKind, meta: ConfigExportMeta): string`

Not added to the `src/lib/utils/index.ts` barrel — that barrel re-exports only `toast` and `maplestory`; call sites import from the module path directly, matching `asset-url`, `clock`, `reward-pool-chance`.

- [x] **Step 1: Write the failing test**

Create `src/lib/utils/__tests__/config-export.test.ts`:

```ts
import { describe, expect, it } from "vitest";

import {
  configExportFilename,
  toConfigExportPayload,
} from "@/lib/utils/config-export";

// Built in the on-the-wire key order that atlas-configurations emits
// (templates/rest.go and tenants/rest.go declare byte-identical field sets in
// this order), which is also the key order of the checked-in seed files.
function fixture(overrides: Record<string, unknown> = {}) {
  return {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    socket: {
      handlers: [
        { opCode: "0x0B8", validator: "v2", handler: "h2" },
        { opCode: "0x01", validator: "v1", handler: "h1" },
      ],
      writers: [
        { opCode: "0x1A", writer: "w2" },
        { opCode: "0x0F", writer: "w1" },
      ],
    },
    characters: { templates: [], presets: [] },
    npcs: [{ npcId: 9000000, impl: "shop" }],
    worlds: [{ name: "Scania" }],
    cashShop: { commodities: {} },
    ...overrides,
  };
}

describe("toConfigExportPayload", () => {
  it("emits no JSON:API envelope keys", () => {
    const out = toConfigExportPayload(fixture());

    expect(out).not.toHaveProperty("id");
    expect(out).not.toHaveProperty("type");
    expect(out).not.toHaveProperty("data");
  });

  it("preserves the seed-file key order", () => {
    const out = toConfigExportPayload(fixture());

    expect(Object.keys(out)).toEqual([
      "region",
      "majorVersion",
      "minorVersion",
      "usesPin",
      "socket",
      "characters",
      "npcs",
      "worlds",
      "cashShop",
    ]);
  });

  it("normalises null collections to empty arrays in place", () => {
    const out = toConfigExportPayload(fixture({ npcs: null, worlds: null }));

    expect(out.npcs).toEqual([]);
    expect(out.worlds).toEqual([]);
    expect(Object.keys(out).indexOf("npcs")).toBe(6);
    expect(Object.keys(out).indexOf("worlds")).toBe(7);
  });

  it("passes present collections through by value", () => {
    const out = toConfigExportPayload(fixture());

    expect(out.npcs).toEqual([{ npcId: 9000000, impl: "shop" }]);
    expect(out.worlds).toEqual([{ name: "Scania" }]);
  });

  it("sorts handlers and writers ascending by numeric opCode", () => {
    const out = toConfigExportPayload(fixture());

    expect(out.socket?.handlers?.map((h) => h.opCode)).toEqual([
      "0x01",
      "0x0B8",
    ]);
    expect(out.socket?.writers?.map((w) => w.opCode)).toEqual(["0x0F", "0x1A"]);
  });

  it("keeps both entries when two opCodes compare numerically equal", () => {
    const out = toConfigExportPayload(
      fixture({
        socket: {
          handlers: [
            { opCode: "0xB8", handler: "padded-off" },
            { opCode: "0x0B8", handler: "padded-on" },
          ],
          writers: [],
        },
      }),
    );

    expect(out.socket?.handlers).toHaveLength(2);
  });

  it("does not mutate the input", () => {
    const input = fixture();
    const before = JSON.stringify(input);

    toConfigExportPayload(input);

    expect(JSON.stringify(input)).toBe(before);
  });

  it("returns an absent socket untouched", () => {
    const out = toConfigExportPayload(fixture({ socket: null }));

    expect(out.socket).toBeNull();
  });
});

describe("configExportFilename", () => {
  const meta = {
    id: "8b1d4c4e-0000-4000-8000-000000000000",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  };

  it("names a template export after the seed convention", () => {
    expect(configExportFilename("template", meta)).toBe(
      "template_gms_83_1.json",
    );
  });

  it("prefixes a tenant export with tenant_", () => {
    expect(configExportFilename("tenant", meta)).toBe("tenant_gms_83_1.json");
  });

  it("sanitises characters outside [a-z0-9] in the region", () => {
    expect(configExportFilename("template", { ...meta, region: "GMS-Beta" })).toBe(
      "template_gms_beta_83_1.json",
    );
  });

  it("falls back to the id when the region is missing or empty", () => {
    expect(configExportFilename("template", { ...meta, region: undefined })).toBe(
      "template_8b1d4c4e_0000_4000_8000_000000000000.json",
    );
    expect(configExportFilename("tenant", { ...meta, region: "   " })).toBe(
      "tenant_8b1d4c4e_0000_4000_8000_000000000000.json",
    );
  });

  it("falls back to the id when a version is not a finite number", () => {
    expect(
      configExportFilename("template", { ...meta, majorVersion: undefined }),
    ).toBe("template_8b1d4c4e_0000_4000_8000_000000000000.json");
    expect(
      configExportFilename("tenant", { ...meta, minorVersion: Number.NaN }),
    ).toBe("tenant_8b1d4c4e_0000_4000_8000_000000000000.json");
  });
});
```

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/utils/__tests__/config-export.test.ts`
Expected: FAIL — cannot resolve `@/lib/utils/config-export`.

- [x] **Step 3: Write the implementation**

Create `src/lib/utils/config-export.ts`:

```ts
/**
 * Pure projection + naming for the Template / Tenant configuration export.
 *
 * The export owns its own output contract rather than inheriting it from the
 * service layer: templates.service's sortTemplate normalises null collections
 * and sorts the socket tables, but tenants.service's sortTenantConfig only
 * sorts - so on the tenant path there is nothing to inherit. Doing both here
 * keeps the exported document's shape independent of a service-layer detail
 * that could be refactored away.
 */

export type ConfigExportKind = "template" | "tenant";

/** One socket table entry, reduced to the field the export orders by. */
interface OpCodedEntry {
  opCode: string;
}

/**
 * The structural subset of TemplateAttributes / TenantConfigAttributes the
 * projection reads. Deliberately loose: everything else is passed through
 * untouched (FR-2.7), so this module never has to track a key list.
 */
export interface ExportableConfigAttributes {
  region?: string;
  majorVersion?: number;
  minorVersion?: number;
  npcs?: unknown[] | null;
  worlds?: unknown[] | null;
  socket?: {
    handlers?: OpCodedEntry[];
    writers?: OpCodedEntry[];
  } | null;
}

export interface ConfigExportMeta {
  id: string;
  region?: string;
  majorVersion?: number;
  minorVersion?: number;
}

function byOpCode(a: OpCodedEntry, b: OpCodedEntry): number {
  return parseInt(a.opCode, 16) - parseInt(b.opCode, 16);
}

/**
 * Project a JSON:API resource's `attributes` into the exported document.
 *
 * Spreading reproduces the seed-file key order for free: JSON.parse preserves
 * insertion order for non-integer-like keys, the server emits them in the
 * order templates/rest.go declares them, and re-assigning an existing key
 * keeps its position. No explicit key-order table is introduced - that would
 * be a second source of truth that goes stale the next time RestModel gains a
 * field.
 *
 * Note: normalising a key that is present-but-null keeps its position;
 * normalising an ABSENT key appends it at the end. The API never omits npcs
 * or worlds (neither Go field carries omitempty), so that case cannot arise
 * against the real server; the output is still valid if it ever does.
 */
export function toConfigExportPayload<T extends ExportableConfigAttributes>(
  attributes: T,
): T {
  // Built as a record because assigning to a property of a generic T is not
  // expressible in TypeScript; the assertion is confined to the return.
  const out: Record<string, unknown> = { ...attributes };
  out.npcs = attributes.npcs ?? [];
  out.worlds = attributes.worlds ?? [];

  const socket = attributes.socket;
  if (socket) {
    out.socket = {
      ...socket,
      handlers: [...(socket.handlers ?? [])].sort(byOpCode),
      writers: [...(socket.writers ?? [])].sort(byOpCode),
    };
  }

  return out as T;
}

function sanitise(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]/g, "_");
}

/**
 * `template_gms_83_1.json` / `tenant_gms_83_1.json`, matching the seed-data
 * naming convention. Falls back to `<kind>_<id>.json` whenever the
 * region/version metadata is unusable, so the name is never malformed.
 */
export function configExportFilename(
  kind: ConfigExportKind,
  meta: ConfigExportMeta,
): string {
  const region = meta.region ? sanitise(meta.region.trim()) : "";
  const major = meta.majorVersion;
  const minor = meta.minorVersion;
  const versioned =
    region.replace(/_/g, "") !== "" &&
    typeof major === "number" &&
    Number.isFinite(major) &&
    typeof minor === "number" &&
    Number.isFinite(minor);

  if (!versioned) {
    return `${kind}_${sanitise(meta.id)}.json`;
  }
  return `${kind}_${region}_${major}_${minor}.json`;
}
```

- [x] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/lib/utils/__tests__/config-export.test.ts`
Expected: PASS — 14 tests.

- [x] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/utils/config-export.ts \
        services/atlas-ui/src/lib/utils/__tests__/config-export.test.ts
git commit -m "feat(ui): add config export payload projection and filename derivation"
```

---

### Task 3: `useDetailActionBarState` + `ConfigExportButton`

**Files:**
- Modify: `services/atlas-ui/src/components/DetailActionBarContext.tsx` (append one exported hook)
- Create: `services/atlas-ui/src/components/features/config/ConfigExportButton.tsx`
- Test: `services/atlas-ui/src/components/features/config/__tests__/ConfigExportButton.test.tsx`

**Interfaces:**
- Consumes: `downloadJson(filename, payload)` (Task 1); `toConfigExportPayload`, `configExportFilename`, `ConfigExportKind` (Task 2); the existing `useTemplate(id)` (`src/lib/hooks/api/useTemplates.ts:113`) and `useTenantConfiguration(id)` (`src/lib/hooks/api/useTenants.ts:228`), both already guarded by `enabled: !!id`.
- Produces:
  - `export function useDetailActionBarState(): DetailActionBarConfig | null` in `DetailActionBarContext.tsx`
  - `export interface ConfigExportButtonProps { kind: ConfigExportKind; id: string | undefined }`
  - `export function ConfigExportButton(props: ConfigExportButtonProps): JSX.Element`

- [x] **Step 1: Write the failing test**

Create `src/components/features/config/__tests__/ConfigExportButton.test.tsx`:

```tsx
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { ConfigExportButton } from "@/components/features/config/ConfigExportButton";
import { DetailActionBarProvider } from "@/components/DetailActionBarContext";
import { templateKeys } from "@/lib/hooks/api/useTemplates";
import { tenantKeys } from "@/lib/hooks/api/useTenants";
import { downloadJson } from "@/lib/utils/download-json";
import { templatesService } from "@/services/api/templates.service";
import { tenantsService } from "@/services/api/tenants.service";
import { toast } from "sonner";

vi.mock("@/lib/utils/download-json", () => ({ downloadJson: vi.fn() }));
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));
vi.mock("@/services/api/templates.service", () => ({
  templatesService: { getById: vi.fn() },
}));
vi.mock("@/services/api/tenants.service", () => ({
  tenantsService: { getTenantConfigurationById: vi.fn() },
}));

const ID = "8b1d4c4e-0000-4000-8000-000000000000";

const attributes = {
  region: "GMS",
  majorVersion: 83,
  minorVersion: 1,
  usesPin: false,
  socket: { handlers: [], writers: [] },
  characters: { templates: [], presets: [] },
  npcs: null,
  worlds: null,
};

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
}

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        <DetailActionBarProvider>{children}</DetailActionBarProvider>
      </QueryClientProvider>
    );
  };
}

describe("ConfigExportButton", () => {
  beforeEach(() => {
    vi.mocked(templatesService.getById).mockResolvedValue({
      id: ID,
      attributes,
    } as never);
    vi.mocked(tenantsService.getTenantConfigurationById).mockResolvedValue({
      id: ID,
      attributes,
    } as never);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("is disabled until the query has data", async () => {
    const client = makeClient();
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    expect(button).toBeDisabled();
    await waitFor(() => expect(button).toBeEnabled());
  });

  it("stays disabled when there is no id", () => {
    const client = makeClient();
    render(<ConfigExportButton kind="template" id={undefined} />, {
      wrapper: wrapper(client),
    });

    expect(screen.getByRole("button", { name: "Export" })).toBeDisabled();
  });

  it("stays disabled when the query errors", async () => {
    vi.mocked(templatesService.getById).mockRejectedValue(new Error("boom"));
    const client = makeClient();
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() =>
      expect(vi.mocked(templatesService.getById)).toHaveBeenCalled(),
    );
    expect(button).toBeDisabled();
  });

  it("downloads the projected attributes under the seed-style filename", async () => {
    const client = makeClient();
    client.setQueryData(templateKeys.detail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() => expect(button).toBeEnabled());
    await userEvent.click(button);

    expect(vi.mocked(downloadJson)).toHaveBeenCalledTimes(1);
    const [filename, payload] = vi.mocked(downloadJson).mock.calls[0];
    expect(filename).toBe("template_gms_83_1.json");
    expect(payload).not.toHaveProperty("id");
    expect(payload).toMatchObject({ region: "GMS", npcs: [], worlds: [] });
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Template exported");
  });

  it("fires no additional request when clicked with the resource cached", async () => {
    const client = makeClient();
    client.setQueryData(templateKeys.detail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() => expect(button).toBeEnabled());
    await waitFor(() =>
      expect(vi.mocked(templatesService.getById)).toHaveBeenCalledTimes(1),
    );
    const before = vi.mocked(templatesService.getById).mock.calls.length;

    await userEvent.click(button);

    expect(vi.mocked(templatesService.getById).mock.calls.length).toBe(before);
  });

  it("exports a tenant configuration with the tenant_ prefix and toast", async () => {
    const client = makeClient();
    client.setQueryData(tenantKeys.configDetail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="tenant" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() => expect(button).toBeEnabled());
    await userEvent.click(button);

    expect(vi.mocked(downloadJson).mock.calls[0][0]).toBe(
      "tenant_gms_83_1.json",
    );
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("Tenant exported");
    expect(vi.mocked(templatesService.getById)).not.toHaveBeenCalled();
  });

  it("shows an error toast when the download throws", async () => {
    vi.mocked(downloadJson).mockImplementation(() => {
      throw new Error("nope");
    });
    const client = makeClient();
    client.setQueryData(templateKeys.detail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    await waitFor(() => expect(button).toBeEnabled());
    await userEvent.click(button);

    expect(vi.mocked(toast.error)).toHaveBeenCalledWith("Export failed");
    expect(vi.mocked(toast.success)).not.toHaveBeenCalled();
  });

  it("marks the icon aria-hidden so the accessible name is just Export", async () => {
    const client = makeClient();
    client.setQueryData(templateKeys.detail(ID), { id: ID, attributes });
    render(<ConfigExportButton kind="template" id={ID} />, {
      wrapper: wrapper(client),
    });

    const button = screen.getByRole("button", { name: "Export" });
    expect(button.querySelector("svg")).toHaveAttribute("aria-hidden", "true");
  });
});
```

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/components/features/config/__tests__/ConfigExportButton.test.tsx`
Expected: FAIL — cannot resolve `@/components/features/config/ConfigExportButton`.

- [x] **Step 3: Add the `useDetailActionBarState` accessor**

In `src/components/DetailActionBarContext.tsx`, append this export at the end of the file (after `DetailActionBar`):

```tsx
/**
 * Read the shared action bar's current state without registering anything.
 * Used by controls that render beside the bar's owner and need to know whether
 * the page has unsaved edits - e.g. the header Export button, which exports the
 * last SAVED document and says so while the page is dirty. Returns null outside
 * a DetailActionBarProvider, or when no page has registered.
 */
export function useDetailActionBarState(): DetailActionBarConfig | null {
  const ctx = useContext(DetailActionBarContext);
  return ctx?.config ?? null;
}
```

- [x] **Step 4: Write the `ConfigExportButton` implementation**

Create `src/components/features/config/ConfigExportButton.tsx`:

```tsx
import { Download } from "lucide-react";
import { toast } from "sonner";

import { useDetailActionBarState } from "@/components/DetailActionBarContext";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useTemplate } from "@/lib/hooks/api/useTemplates";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import {
  configExportFilename,
  toConfigExportPayload,
  type ConfigExportKind,
} from "@/lib/utils/config-export";
import { downloadJson } from "@/lib/utils/download-json";

export interface ConfigExportButtonProps {
  kind: ConfigExportKind;
  id: string | undefined;
}

/**
 * Downloads the viewed Template / Tenant configuration as a seed-shaped JSON
 * file. Lives in the detail LAYOUT header, so it is present on every sub-tab
 * without per-page wiring.
 *
 * The payload is read from the React Query cache, so it is the last PERSISTED
 * document - never the page's unsaved form state. That is the point: the file
 * exists to be diffed against, or promoted into, a checked-in seed template.
 */
export function ConfigExportButton({ kind, id }: ConfigExportButtonProps) {
  // Rules of Hooks forbid calling one hook conditionally, so both are called
  // and the irrelevant one is disabled with an empty id (both guard with
  // `enabled: !!id`, so it issues no request). Same pattern as
  // DefinitionGridPage.tsx.
  const templateQuery = useTemplate(kind === "template" ? (id ?? "") : "");
  const tenantQuery = useTenantConfiguration(
    kind === "tenant" ? (id ?? "") : "",
  );
  const query = kind === "template" ? templateQuery : tenantQuery;
  const actionBar = useDetailActionBarState();

  const onExport = () => {
    const data = query.data;
    if (!data) return;
    try {
      downloadJson(
        configExportFilename(kind, {
          id: data.id,
          region: data.attributes.region,
          majorVersion: data.attributes.majorVersion,
          minorVersion: data.attributes.minorVersion,
        }),
        toConfigExportPayload(data.attributes),
      );
      toast.success(
        kind === "template" ? "Template exported" : "Tenant exported",
      );
    } catch {
      toast.error("Export failed");
    }
  };

  // Deriving from data presence covers loading, error, refetch-after-error and
  // the no-id case in one predicate.
  const button = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={!query.data}
      onClick={onExport}
    >
      <Download className="h-4 w-4" aria-hidden="true" />
      Export
    </Button>
  );

  // The tooltip earns its place only while the page has unsaved edits - an
  // always-on tooltip on a self-explanatory button is noise. `Tooltip` mounts
  // its own TooltipProvider (components/ui/tooltip.tsx), so none is added here.
  if (!actionBar?.dirty) return button;

  return (
    <Tooltip>
      <TooltipTrigger asChild>{button}</TooltipTrigger>
      <TooltipContent>Exports the last saved configuration</TooltipContent>
    </Tooltip>
  );
}
```

- [x] **Step 5: Run the test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/components/features/config/__tests__/ConfigExportButton.test.tsx`
Expected: PASS — 8 tests.

- [x] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/DetailActionBarContext.tsx \
        services/atlas-ui/src/components/features/config/ConfigExportButton.tsx \
        services/atlas-ui/src/components/features/config/__tests__/ConfigExportButton.test.tsx
git commit -m "feat(ui): add ConfigExportButton and DetailActionBar state accessor"
```

---

### Task 4: Wire the button into both detail layouts

**Files:**
- Modify: `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx`
- Modify: `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx`
- Test: `services/atlas-ui/src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx` (new)
- Test: `services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx` (new)

**Interfaces:**
- Consumes: `ConfigExportButton` (Task 3).
- Produces: nothing new — the layouts' props are unchanged.

- [x] **Step 1: Write the failing tests**

Create `src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx`:

```tsx
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { TemplateDetailLayout } from "@/components/features/templates/TemplateDetailLayout";

vi.mock("@/components/features/config/ConfigExportButton", () => ({
  ConfigExportButton: ({ kind, id }: { kind: string; id?: string }) => (
    <button type="button" data-kind={kind} data-id={id}>
      Export
    </button>
  ),
}));

describe("TemplateDetailLayout", () => {
  it("renders an Export control in the header for the routed template", () => {
    render(
      <MemoryRouter initialEntries={["/templates/tpl-1/properties"]}>
        <Routes>
          <Route
            path="/templates/:id/properties"
            element={
              <TemplateDetailLayout>
                <div>child</div>
              </TemplateDetailLayout>
            }
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("Template Details")).toBeInTheDocument();
    const button = screen.getByRole("button", { name: "Export" });
    expect(button).toHaveAttribute("data-kind", "template");
    expect(button).toHaveAttribute("data-id", "tpl-1");
  });
});
```

Create `src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx`:

```tsx
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";

vi.mock("@/components/features/config/ConfigExportButton", () => ({
  ConfigExportButton: ({ kind, id }: { kind: string; id?: string }) => (
    <button type="button" data-kind={kind} data-id={id}>
      Export
    </button>
  ),
}));

describe("TenantDetailLayout", () => {
  it("renders an Export control in the header for the routed tenant", () => {
    render(
      <MemoryRouter initialEntries={["/tenants/tnt-1/properties"]}>
        <Routes>
          <Route
            path="/tenants/:id/properties"
            element={
              <TenantDetailLayout>
                <div>child</div>
              </TenantDetailLayout>
            }
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("Tenant Details")).toBeInTheDocument();
    const button = screen.getByRole("button", { name: "Export" });
    expect(button).toHaveAttribute("data-kind", "tenant");
    expect(button).toHaveAttribute("data-id", "tnt-1");
  });
});
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-ui && npx vitest run src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx`
Expected: FAIL — `Unable to find an accessible element with the role "button" and name "Export"`.

- [x] **Step 3: Wire the template layout**

In `src/components/features/templates/TemplateDetailLayout.tsx`, add the import:

```tsx
import { ConfigExportButton } from "@/components/features/config/ConfigExportButton";
```

and replace the header block

```tsx
        <div className="space-y-0.5">
          <h2 className="text-2xl font-bold tracking-tight">
            Template Details
          </h2>
          <p className="text-muted-foreground">{id}</p>
        </div>
```

with

```tsx
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-0.5">
            <h2 className="text-2xl font-bold tracking-tight">
              Template Details
            </h2>
            <p className="text-muted-foreground">{id}</p>
          </div>
          <ConfigExportButton kind="template" id={id} />
        </div>
```

- [x] **Step 4: Wire the tenant layout**

In `src/components/features/tenants/TenantDetailLayout.tsx`, add the import:

```tsx
import { ConfigExportButton } from "@/components/features/config/ConfigExportButton";
```

and replace the header block

```tsx
        <div className="space-y-0.5">
          <h2 className="text-2xl font-bold tracking-tight">Tenant Details</h2>
          <p className="text-muted-foreground">{id}</p>
        </div>
```

with

```tsx
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-0.5">
            <h2 className="text-2xl font-bold tracking-tight">Tenant Details</h2>
            <p className="text-muted-foreground">{id}</p>
          </div>
          <ConfigExportButton kind="tenant" id={id} />
        </div>
```

The `DetailActionBar` at the bottom of each layout and the `DetailActionBarProvider` wrapper are untouched — the export button renders inside the provider (so `useDetailActionBarState` resolves) but in a different branch of the tree from the save bar.

- [x] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-ui && npx vitest run src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx`
Expected: PASS — 2 tests.

- [x] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx \
        services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx \
        services/atlas-ui/src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx \
        services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx
git commit -m "feat(ui): render the config Export button in both detail layouts"
```

---

### Task 5: Full gate run

**Files:**
- Modify: whatever the gates report (formatting fixes only, if any).

**Interfaces:**
- Consumes: everything from Tasks 1–4.
- Produces: a branch that satisfies every acceptance criterion in `prd.md` §10.

- [x] **Step 1: Run the full unit suite**

Run: `cd services/atlas-ui && npm run test`
Expected: PASS, no new failures. If a pre-existing failure appears, capture its output and confirm it also fails on `main` before treating it as unrelated — do not silence it.

- [x] **Step 2: Type-check via the build**

Run: `cd services/atlas-ui && npm run build`
Expected: `tsc -b` clean (this is the type-check, and it covers the new test files), then a successful `vite build`.

- [x] **Step 3: Run the shared lint & format guard**

Run (from the worktree root, with nvm 22 on PATH): `tools/lint.sh --check`
Expected: exit 0. If it reports formatting drift, run `tools/lint.sh` (no flags) to fix in place, then re-run `--check`.

- [x] **Step 4: Confirm the diff containment**

Run (from the worktree root):

```bash
git diff --name-only main...HEAD | grep -v '^services/atlas-ui/' | grep -v '^docs/' || true
```

Expected: no output. Any path printed is a violation of the Global Constraints — no Go module may be touched, so no `docker buildx bake` target is in scope.

- [ ] **Step 5: Manually verify the exported file against a seed** — NOT PERFORMED. Deferred to the human partner by explicit decision: this step needs a real browser (dev server + click + inspect the downloaded file), which the execution environment cannot drive. Everything it would cover at the data level is covered by automated fixtures (key order, sorting, null normalisation, trailing newline); what remains uncovered is the browser's actual download behaviour — notably the synchronous `URL.revokeObjectURL` in `download-json.ts`, which jsdom cannot exercise.

Run the dev server (`cd services/atlas-ui && npm run dev`), open a Template Details page for a seeded version, click **Export**, and check the downloaded file:

```bash
head -6 ~/Downloads/template_gms_83_1.json
python3 -c "import json,sys; print(list(json.load(open(sys.argv[1])).keys()))" ~/Downloads/template_gms_83_1.json
tail -c 2 ~/Downloads/template_gms_83_1.json | od -c
```

Expected: keys in the order `['region','majorVersion','minorVersion','usesPin','socket','characters','npcs','worlds','cashShop']` (a live tenant may carry a subset), the file ending in `}\n`, and a diff against `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` showing only intentional drift. Record what you observed — do not claim this step passed without the actual output.

- [x] **Step 6: Commit any gate fixes**

```bash
git add -A services/atlas-ui
git commit -m "chore(ui): satisfy lint/format gates for config export"
```

(Skip if the gates required no changes.)

---

## Self-Review

**Spec coverage** — every PRD functional requirement maps to a task:

| Requirement | Where |
|---|---|
| FR-1.1 / FR-1.2 / FR-1.3 / FR-1.4 | Task 4 (button in the layout header, `DetailActionBar` untouched) |
| FR-1.5 | Task 3 (`variant="outline"`, `size="sm"`, `Download` icon, `Export` label) |
| FR-2.1 / FR-2.7 | Task 2 (`toConfigExportPayload` receives only `.attributes`; envelope test) |
| FR-2.2 / FR-2.3 | Task 2 — key-agnostic projection; design.md §2.2 corrects FR-2.2's key list (templates DO carry `cashShop`), so no key list is enumerated in code |
| FR-2.4 | Task 1 (`JSON.stringify(payload, null, 2)` + `"\n"`) |
| FR-2.5 | Task 2 (`byOpCode` sort) |
| FR-2.6 | Task 2 (`?? []` normalisation, both kinds) |
| FR-3.1 – FR-3.4 | Task 2 (`configExportFilename` + its five naming tests) |
| FR-4.1 – FR-4.4 | Task 1 (Blob + object URL + synthetic anchor, `finally` teardown, `application/json`, single shared helper) |
| FR-5.1 / FR-5.2 / FR-5.4 | Task 3 (`query.data` source, `disabled={!query.data}`, no-additional-request test) |
| FR-5.3 | Task 3 (`useDetailActionBarState()?.dirty` → tooltip) |
| FR-6.1 / FR-6.2 / FR-6.3 | Task 3 (success/error toasts; the handler is `void` and touches nothing outside the DOM anchor) |
| FR-7.1 / FR-7.2 | Task 3 — inherited: the export adds no request path, and `query.data` is read synchronously inside the handler, so a tenant switch that clears the cache disables the button rather than exporting stale data |
| PRD §10 gates | Task 5 |

**Placeholder scan:** no TBD/TODO, no "add error handling", no "similar to Task N". Every code step carries the literal code.

**Type consistency:** `ConfigExportKind`, `ConfigExportMeta`, `ExportableConfigAttributes`, `toConfigExportPayload`, `configExportFilename`, `downloadJson`, `useDetailActionBarState`, `ConfigExportButtonProps` are each defined once (Tasks 1–3) and referenced under exactly those names in Tasks 3–4. Query keys `templateKeys.detail(id)` and `tenantKeys.configDetail(id)` match `useTemplates.ts:36` and `useTenants.ts:42`.
