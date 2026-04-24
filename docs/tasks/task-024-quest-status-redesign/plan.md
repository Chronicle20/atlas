# Quest Status Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the single-column Quest Status list on the Character Detail page with a responsive grid of whole-card-clickable widgets that show the resolved quest name, preserving only the completion-count badge and (on the Completed tab) the completion timestamp.

**Architecture:** atlas-ui only. A new `QuestName` primitive is added to the existing `EntityName.tsx` barrel; it pulls `activeTenant` from `useTenant()` and calls `useQuest` internally, matching the `*Name` family's prop API. `QuestStatusTabs.tsx` drops its inner `QuestStatusCard` in favor of a new inline `QuestStatusWidget` whose root element is a single React Router `<Link>` with no nested interactive children. The progress line and expiration line are removed. `EntityWidget.tsx` is left alone — both callers share the same `questKeys.detail` React Query cache entry.

**Tech Stack:** React 19, TypeScript, React Router v7, TanStack React Query 5, Tailwind 4 (shadcn/ui), Vitest + `@testing-library/react`.

**Reference docs:**
- Spec: `docs/tasks/task-024-quest-status-redesign/design.md`
- PRD: `docs/tasks/task-024-quest-status-redesign/prd.md`
- Execution context cheat-sheet: `docs/tasks/task-024-quest-status-redesign/context.md`

---

## Task 1: Add `QuestName` component and unit test

**Files:**
- Create: `services/atlas-ui/src/components/features/quests/__tests__/QuestName.test.tsx`
- Modify: `services/atlas-ui/src/components/features/quests/EntityName.tsx`

- [ ] **Step 1: Write the failing test file**

Create `services/atlas-ui/src/components/features/quests/__tests__/QuestName.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { UseQueryResult } from "@tanstack/react-query";
import type { QuestDefinition } from "@/types/models/quest";
import type { Tenant } from "@/types/models/tenant";

const useQuestMock = vi.fn();
const useTenantMock = vi.fn();

vi.mock("@/lib/hooks/api/useQuests", () => ({
    useQuest: (...args: unknown[]) => useQuestMock(...args),
}));

vi.mock("@/context/tenant-context", () => ({
    useTenant: () => useTenantMock(),
}));

import { QuestName } from "@/components/features/quests/EntityName";

const fakeTenant = { id: "tenant-1" } as unknown as Tenant;

function mockQuery(
    overrides: Partial<UseQueryResult<QuestDefinition, Error>>,
): UseQueryResult<QuestDefinition, Error> {
    return {
        data: undefined,
        isLoading: false,
        isError: false,
        error: null,
        ...overrides,
    } as UseQueryResult<QuestDefinition, Error>;
}

function makeQuest(name: string): QuestDefinition {
    return {
        id: "42",
        type: "quests",
        attributes: { name } as QuestDefinition["attributes"],
    };
}

describe("QuestName", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        useTenantMock.mockReturnValue({ activeTenant: fakeTenant });
    });

    it("renders a skeleton while loading", () => {
        useQuestMock.mockReturnValue(mockQuery({ isLoading: true }));
        const { container } = render(<QuestName id={42} />);
        // shadcn Skeleton is a div with .animate-pulse in this project
        // (see src/components/ui/skeleton.tsx).
        expect(container.querySelector(".animate-pulse")).not.toBeNull();
        expect(screen.queryByText(/Quest #42/)).toBeNull();
    });

    it("renders the resolved name with a title attribute on success", () => {
        useQuestMock.mockReturnValue(
            mockQuery({ data: makeQuest("Hello Maple") }),
        );
        render(<QuestName id={42} />);
        const span = screen.getByText("Hello Maple");
        expect(span.tagName).toBe("SPAN");
        expect(span.getAttribute("title")).toBe("Hello Maple");
    });

    it("falls back to Quest #<id> on error", () => {
        useQuestMock.mockReturnValue(
            mockQuery({ isError: true, error: new Error("boom") }),
        );
        render(<QuestName id={42} />);
        const span = screen.getByText("Quest #42");
        expect(span.getAttribute("title")).toBe("Quest #42");
    });

    it("falls back to Quest #<id> when data is missing and not loading", () => {
        useQuestMock.mockReturnValue(mockQuery({}));
        render(<QuestName id={42} />);
        expect(screen.getByText("Quest #42")).toBeInTheDocument();
    });

    it("appends muted (#<id>) when showId is true", () => {
        useQuestMock.mockReturnValue(
            mockQuery({ data: makeQuest("Hello Maple") }),
        );
        render(<QuestName id={42} showId />);
        expect(screen.getByText("Hello Maple")).toBeInTheDocument();
        expect(screen.getByText("(#42)")).toBeInTheDocument();
    });

    it("forwards className to the rendered span", () => {
        useQuestMock.mockReturnValue(
            mockQuery({ data: makeQuest("Named") }),
        );
        render(<QuestName id={42} className="font-medium truncate" />);
        const span = screen.getByText("Named");
        expect(span.className).toContain("font-medium");
        expect(span.className).toContain("truncate");
    });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
cd services/atlas-ui && npm run test -- --run src/components/features/quests/__tests__/QuestName.test.tsx
```
Expected: FAIL — `QuestName` is not exported from `@/components/features/quests/EntityName`.

- [ ] **Step 3: Add the `QuestName` implementation**

Open `services/atlas-ui/src/components/features/quests/EntityName.tsx`. Add these two imports to the top of the file (alongside the existing imports):

```tsx
import { useQuest } from "@/lib/hooks/api/useQuests"
import { useTenant } from "@/context/tenant-context"
```

Then append this function at the end of the file (after `JobName`):

```tsx
/**
 * Display Quest name with fallback to ID.
 *
 * Unlike the other *Name components, this pulls `activeTenant` from
 * `useTenant()` and resolves via the React Query `useQuest` hook directly
 * rather than a flattened data hook. Shared `questKeys.detail` cache entries
 * mean repeated renders for the same id are deduplicated.
 */
export function QuestName({ id, showId = false, className }: EntityNameProps) {
    const { activeTenant } = useTenant()
    const { data, isLoading, isError } = useQuest(activeTenant, String(id))

    if (isLoading) {
        return <Skeleton className="h-4 w-16 inline-block" />
    }

    const name = data?.attributes.name
    if (isError || !name) {
        return (
            <span className={className} title={`Quest #${id}`}>
                Quest #{id}
            </span>
        )
    }

    return (
        <span className={className} title={name}>
            {name}
            {showId && <span className="text-muted-foreground ml-1">(#{id})</span>}
        </span>
    )
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
cd services/atlas-ui && npm run test -- --run src/components/features/quests/__tests__/QuestName.test.tsx
```
Expected: PASS — all six cases green.

- [ ] **Step 5: Export `QuestName` from the barrel**

Open `services/atlas-ui/src/components/features/quests/index.ts`. Change:

```ts
export { NpcName, ItemName, MobName, SkillName, JobName } from './EntityName'
```

to:

```ts
export { NpcName, ItemName, MobName, SkillName, JobName, QuestName } from './EntityName'
```

- [ ] **Step 6: Typecheck and lint**

Run:
```bash
cd services/atlas-ui && npm run build && npm run lint
```
Expected: Both succeed. If `tsc -b` complains about an unused `useTenant` or `useQuest` import, recheck Step 3.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/components/features/quests/EntityName.tsx \
        services/atlas-ui/src/components/features/quests/index.ts \
        services/atlas-ui/src/components/features/quests/__tests__/QuestName.test.tsx
git commit -m "feat(atlas-ui): add QuestName resolver component"
```

---

## Task 2: Scaffold `QuestStatusTabs.test.tsx` with a minimum-viable test

This task establishes the test file and a baseline (the Started tab renders by default, shows the loading skeleton correctly, and the existing count description renders). Subsequent tasks extend this file.

**Files:**
- Create: `services/atlas-ui/src/components/features/quests/__tests__/QuestStatusTabs.test.tsx`

- [ ] **Step 1: Write the failing test file**

Create `services/atlas-ui/src/components/features/quests/__tests__/QuestStatusTabs.test.tsx`:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import type { CharacterQuestStatus } from "@/types/models/quest";
import type { Tenant } from "@/types/models/tenant";

const getStartedMock = vi.fn();
const getCompletedMock = vi.fn();
const useTenantMock = vi.fn();

vi.mock("@/services/api/quest-status.service", () => ({
    questStatusService: {
        getStartedQuests: (...args: unknown[]) => getStartedMock(...args),
        getCompletedQuests: (...args: unknown[]) => getCompletedMock(...args),
    },
}));

vi.mock("@/context/tenant-context", () => ({
    useTenant: () => useTenantMock(),
}));

vi.mock("@/components/features/quests/EntityName", () => ({
    QuestName: ({ id, className }: { id: number; className?: string }) => (
        <span className={className} data-testid={`quest-name-${id}`}>
            Quest #{id}
        </span>
    ),
}));

import { QuestStatusTabs } from "@/components/features/quests/QuestStatusTabs";

const fakeTenant = { id: "tenant-1" } as unknown as Tenant;

function makeStatus(
    id: string,
    overrides: Partial<CharacterQuestStatus["attributes"]> = {},
): CharacterQuestStatus {
    return {
        id,
        type: "quest-status",
        attributes: {
            characterId: 1,
            questId: Number(id),
            state: 1,
            startedAt: "2026-04-01T00:00:00Z",
            completedCount: 0,
            forfeitCount: 0,
            progress: [],
            ...overrides,
        },
    };
}

function renderTabs() {
    return render(
        <MemoryRouter>
            <QuestStatusTabs characterId="7" tenant={fakeTenant} />
        </MemoryRouter>,
    );
}

describe("QuestStatusTabs (baseline)", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        useTenantMock.mockReturnValue({ activeTenant: fakeTenant });
        getStartedMock.mockResolvedValue([]);
        getCompletedMock.mockResolvedValue([]);
    });

    it("renders the empty-state copy when no quests are returned", async () => {
        renderTabs();
        expect(
            await screen.findByText(/No quests in progress/i),
        ).toBeInTheDocument();
    });

    it("shows the count description line after fetching", async () => {
        getStartedMock.mockResolvedValue([makeStatus("1001"), makeStatus("1002")]);
        getCompletedMock.mockResolvedValue([makeStatus("2001")]);
        renderTabs();
        expect(
            await screen.findByText(/2 in progress, 1 completed/i),
        ).toBeInTheDocument();
    });

    it("clicking Refresh re-runs both fetchers", async () => {
        renderTabs();
        await screen.findByText(/No quests in progress/i);
        expect(getStartedMock).toHaveBeenCalledTimes(1);
        expect(getCompletedMock).toHaveBeenCalledTimes(1);

        const user = userEvent.setup();
        await user.click(screen.getByRole("button", { name: "" }));
        await waitFor(() => {
            expect(getStartedMock).toHaveBeenCalledTimes(2);
            expect(getCompletedMock).toHaveBeenCalledTimes(2);
        });
    });
});
```

Note: the refresh button is an icon-only button (`variant="outline" size="icon"` with just a `RefreshCw` icon inside). `getByRole("button", { name: "" })` is intentional; if the current source gains an accessible name later, tighten the query.

- [ ] **Step 2: Run the test to verify it passes against the current implementation**

Run:
```bash
cd services/atlas-ui && npm run test -- --run src/components/features/quests/__tests__/QuestStatusTabs.test.tsx
```
Expected: PASS — this exercises behavior that already exists in
`QuestStatusTabs.tsx`. This baseline ensures the test harness works before
Task 3 adds coverage for new behavior.

If the refresh-button query fails due to multiple empty-name buttons,
replace the selector with:

```tsx
const refreshButton = screen
    .getAllByRole("button")
    .find((b) => b.querySelector('svg.lucide-refresh-cw'));
```

- [ ] **Step 3: Commit**

```bash
git add services/atlas-ui/src/components/features/quests/__tests__/QuestStatusTabs.test.tsx
git commit -m "test(atlas-ui): add QuestStatusTabs baseline integration tests"
```

---

## Task 3: Add failing tests for the new grid + widget behavior

These tests describe the PRD/design requirements for the post-refactor state. They MUST fail against the current implementation; Task 4 makes them pass.

**Files:**
- Modify: `services/atlas-ui/src/components/features/quests/__tests__/QuestStatusTabs.test.tsx`

- [ ] **Step 1: Append new describe blocks to the test file**

Append the following to the end of `QuestStatusTabs.test.tsx` (after the
existing `describe("QuestStatusTabs (baseline)", ...)` block, still inside
the same module):

```tsx
describe("QuestStatusTabs (grid + widget behavior)", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        useTenantMock.mockReturnValue({ activeTenant: fakeTenant });
        getStartedMock.mockResolvedValue([]);
        getCompletedMock.mockResolvedValue([]);
    });

    it("renders the Started tab list in a responsive grid container", async () => {
        getStartedMock.mockResolvedValue([makeStatus("1001")]);
        renderTabs();

        const name = await screen.findByTestId("quest-name-1001");
        const grid = name.closest('[data-testid="quest-grid"]');
        expect(grid).not.toBeNull();
        expect(grid!.className).toContain("grid");
        expect(grid!.className).toContain("grid-cols-2");
        expect(grid!.className).toContain("sm:grid-cols-3");
        expect(grid!.className).toContain("lg:grid-cols-4");
        expect(grid!.className).toContain("gap-3");
    });

    it("wraps each widget in a single <a> link to /quests/:questId", async () => {
        getStartedMock.mockResolvedValue([makeStatus("1001")]);
        renderTabs();

        const name = await screen.findByTestId("quest-name-1001");
        const link = name.closest("a");
        expect(link).not.toBeNull();
        expect(link!.getAttribute("href")).toBe("/quests/1001");
        // No nested interactive elements inside the link (no <button>, no
        // other <a>).
        expect(link!.querySelector("button")).toBeNull();
        expect(link!.querySelectorAll("a")).toHaveLength(0);
    });

    it("does NOT render the raw progress line", async () => {
        getStartedMock.mockResolvedValue([
            makeStatus("1001", {
                progress: [{ infoNumber: 5, progress: "10/30" }],
            }),
        ]);
        renderTabs();

        await screen.findByTestId("quest-name-1001");
        expect(screen.queryByText(/#5:/)).toBeNull();
        expect(screen.queryByText(/10\/30/)).toBeNull();
    });

    it("does NOT render the Expires line even when expirationTime is set", async () => {
        getStartedMock.mockResolvedValue([
            makeStatus("1001", { expirationTime: "2030-01-01T00:00:00Z" }),
        ]);
        renderTabs();

        await screen.findByTestId("quest-name-1001");
        expect(screen.queryByText(/Expires:/i)).toBeNull();
    });

    it("does NOT render a separate ExternalLink icon button", async () => {
        getStartedMock.mockResolvedValue([makeStatus("1001")]);
        renderTabs();

        await screen.findByTestId("quest-name-1001");
        // The old icon button used lucide's ExternalLink; the new widget has
        // no <button> inside or alongside the name link.
        const name = screen.getByTestId("quest-name-1001");
        const link = name.closest("a")!;
        expect(link.querySelector(".lucide-external-link")).toBeNull();
    });

    it("shows the x<count> badge only when completedCount > 1", async () => {
        getStartedMock.mockResolvedValue([
            makeStatus("1001", { completedCount: 0 }),
            makeStatus("1002", { completedCount: 1 }),
            makeStatus("1003", { completedCount: 4 }),
        ]);
        renderTabs();

        await screen.findByTestId("quest-name-1003");
        expect(screen.queryByText("x0")).toBeNull();
        expect(screen.queryByText("x1")).toBeNull();
        expect(screen.getByText("x4")).toBeInTheDocument();
    });

    it("shows the completion timestamp on the Completed tab only", async () => {
        getStartedMock.mockResolvedValue([
            makeStatus("1001", { completedAt: "2026-04-01T00:00:00Z" }),
        ]);
        getCompletedMock.mockResolvedValue([
            makeStatus("2001", {
                state: 2,
                completedAt: "2026-04-02T00:00:00Z",
            }),
        ]);
        renderTabs();

        // Started tab is default; completed-at from a started-tab row must not render.
        await screen.findByTestId("quest-name-1001");
        expect(screen.queryByTestId("completion-time")).toBeNull();

        const user = userEvent.setup();
        await user.click(screen.getByRole("tab", { name: /Completed/i }));

        await screen.findByTestId("quest-name-2001");
        const stamp = screen.getByTestId("completion-time");
        expect(stamp).toBeInTheDocument();
        expect(stamp.querySelector(".lucide-clock")).not.toBeNull();
    });

    it("renders the empty-completed message on the Completed tab when the list is empty", async () => {
        getCompletedMock.mockResolvedValue([]);
        renderTabs();

        const user = userEvent.setup();
        await user.click(
            await screen.findByRole("tab", { name: /Completed/i }),
        );
        expect(
            await screen.findByText(/No completed quests/i),
        ).toBeInTheDocument();
    });

    it("surfaces the error card with Retry when the fetch rejects", async () => {
        getStartedMock.mockRejectedValueOnce(new Error("network down"));
        renderTabs();

        expect(await screen.findByText(/network down/i)).toBeInTheDocument();
        const retry = screen.getByRole("button", { name: /Retry/i });

        // Retry must call both fetchers again.
        getStartedMock.mockResolvedValueOnce([]);
        getCompletedMock.mockResolvedValueOnce([]);
        const user = userEvent.setup();
        await user.click(retry);

        await waitFor(() => {
            expect(getStartedMock.mock.calls.length).toBeGreaterThanOrEqual(2);
        });
    });
});
```

- [ ] **Step 2: Run the test file to confirm the new tests fail**

Run:
```bash
cd services/atlas-ui && npm run test -- --run src/components/features/quests/__tests__/QuestStatusTabs.test.tsx
```
Expected: The baseline tests still PASS. The new `grid + widget behavior`
tests FAIL because:
- The grid container is not present (current code uses `space-y-3`).
- The widget still contains an `ExternalLink` icon `<button>` nested next to
  the name link.
- The widget still renders the progress line and the Expires line.
- There is no `data-testid="quest-grid"` or `data-testid="completion-time"`
  in the current markup.

If any new test is green against the old implementation, the assertion is
weaker than it should be — tighten it before continuing.

- [ ] **Step 3: Do NOT commit yet**

Leave the failing tests uncommitted. Task 4 lands them together with the
implementation that makes them pass, so the commit history never contains a
red state.

---

## Task 4: Replace `QuestStatusCard` with `QuestStatusWidget` in a grid

**Files:**
- Modify: `services/atlas-ui/src/components/features/quests/QuestStatusTabs.tsx`

- [ ] **Step 1: Update imports**

In `QuestStatusTabs.tsx`, change the `lucide-react` import line from:

```tsx
import { RefreshCw, Clock, CheckCircle, Play, ExternalLink } from "lucide-react"
```

to:

```tsx
import { RefreshCw, Clock, CheckCircle, Play } from "lucide-react"
```

Then add this new import (grouped with the other `@/` imports):

```tsx
import { QuestName } from "./EntityName"
```

- [ ] **Step 2: Swap the Started-tab container to a grid**

Locate the Started `TabsContent` block. Replace:

```tsx
<div className="space-y-3">
    {startedQuests.map((quest) => (
        <QuestStatusCard
            key={quest.id}
            quest={quest}
            showProgress
        />
    ))}
</div>
```

with:

```tsx
<div
    data-testid="quest-grid"
    className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3"
>
    {startedQuests.map((quest) => (
        <QuestStatusWidget key={quest.id} quest={quest} />
    ))}
</div>
```

- [ ] **Step 3: Swap the Completed-tab container to a grid**

In the Completed `TabsContent` block, replace:

```tsx
<div className="space-y-3">
    {completedQuests.map((quest) => (
        <QuestStatusCard
            key={quest.id}
            quest={quest}
            showCompletionTime
        />
    ))}
</div>
```

with:

```tsx
<div
    data-testid="quest-grid"
    className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3"
>
    {completedQuests.map((quest) => (
        <QuestStatusWidget
            key={quest.id}
            quest={quest}
            showCompletionTime
        />
    ))}
</div>
```

- [ ] **Step 4: Replace the inner component**

Delete the entire `QuestStatusCardProps` interface and the `QuestStatusCard`
function (the block spanning roughly `interface QuestStatusCardProps { ... }`
through the closing `}` of `function QuestStatusCard(...)`).

Replace it with:

```tsx
interface QuestStatusWidgetProps {
    quest: CharacterQuestStatus
    showCompletionTime?: boolean
}

function QuestStatusWidget({ quest, showCompletionTime }: QuestStatusWidgetProps) {
    const attrs = quest.attributes

    return (
        <Link
            to={`/quests/${attrs.questId}`}
            className="block border rounded-lg p-3 overflow-hidden hover:bg-muted/50 transition-colors"
        >
            <div className="flex items-center justify-between gap-2 min-w-0">
                <QuestName
                    id={attrs.questId}
                    className="font-medium truncate"
                />
                {attrs.completedCount > 1 && (
                    <Badge variant="outline" className="text-xs shrink-0">
                        x{attrs.completedCount}
                    </Badge>
                )}
            </div>
            {showCompletionTime && attrs.completedAt && (
                <div
                    data-testid="completion-time"
                    className="mt-1 text-sm text-muted-foreground flex items-center gap-1"
                >
                    <Clock className="h-3 w-3" />
                    {formatDate(attrs.completedAt)}
                </div>
            )}
        </Link>
    )
}
```

Keep `formatDate`, `QuestStatusSkeleton`, the outer `Card`, `CardHeader`,
`CardDescription`, Tabs, and all other existing structures untouched.

- [ ] **Step 5: Run the full test file and confirm everything passes**

Run:
```bash
cd services/atlas-ui && npm run test -- --run src/components/features/quests/__tests__/QuestStatusTabs.test.tsx
```
Expected: PASS — both the baseline describe block (from Task 2) and the grid
+ widget describe block (from Task 3) green.

If `userEvent.click` on the Completed tab fails to flip panels in tests,
confirm the `TabsTrigger` value is still `"completed"` and that
`defaultValue="started"` remains on `<Tabs>`.

- [ ] **Step 6: Typecheck and lint**

Run:
```bash
cd services/atlas-ui && npm run build && npm run lint
```
Expected: Both succeed. In particular:
- `tsc -b` should flag no unused imports (if `ExternalLink` is still
  imported, remove it).
- ESLint should flag no unused vars (no leftover `showProgress` prop).

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/components/features/quests/QuestStatusTabs.tsx \
        services/atlas-ui/src/components/features/quests/__tests__/QuestStatusTabs.test.tsx
git commit -m "feat(atlas-ui): render quest status as responsive widget grid"
```

---

## Task 5: Manual smoke test

This task is not automated. It is required before declaring the feature done
because the unit tests assert class names, not visual layout.

**Files:** none.

- [ ] **Step 1: Start the dev server**

Run:
```bash
cd services/atlas-ui && npm run dev
```
The server listens on `http://localhost:5173` by default and proxies `/api`
to `http://localhost:8080`.

- [ ] **Step 2: Navigate to a Character Detail page**

Open a browser to `http://localhost:5173`, select a tenant with test data,
navigate to Characters, click into any character that has at least one
started and one completed quest. If no such character exists in the local
environment, skip to Step 6 and document the limitation.

- [ ] **Step 3: Verify the Started tab**

- The Quest Status section renders widgets in a grid.
- At window widths < `sm` (Tailwind's 640px), the grid has 2 columns.
- At `sm..lg` (640–1024px), the grid has 3 columns.
- At `>= lg` (1024px+), the grid has 4 columns.
- Each widget shows the quest name (or a short skeleton that resolves to a
  name within a second or two). No raw `#<infoNumber>: <progress>` text. No
  `Expires:` text.
- Hovering a widget highlights the whole card.
- Clicking anywhere on a widget (name text, empty padding, badge) navigates
  to `/quests/<id>` without a full page reload (the URL bar updates, nav
  stays on the SPA).

- [ ] **Step 4: Verify the Completed tab**

- Click the Completed tab. Widgets render in the same grid.
- Widgets with `completedCount > 1` show the `x<count>` badge.
- Each widget shows a second muted line with a clock icon and the formatted
  completion date.

- [ ] **Step 5: Verify error/empty/loading states**

- Stop the atlas-quest backend (or point the proxy at a failing URL) and
  refresh. The section renders the error Card with a Retry button.
- Start the backend, click Retry. Data loads again.
- Use a character with no started quests. The Started tab shows
  `No quests in progress`; the Completed tab shows `No completed quests`
  when empty.

- [ ] **Step 6: Record the result**

In the PR description, paste a one-line confirmation per step (or note the
deferred item) so a reviewer can trace it. If any step fails, stop and file
a bug — do NOT merge.

- [ ] **Step 7: No commit**

Nothing changed on disk in Task 5. Proceed to Task 6.

---

## Task 6: Final verification sweep

**Files:** none (verification only).

- [ ] **Step 1: Run the full atlas-ui test suite**

Run:
```bash
cd services/atlas-ui && npm run test -- --run
```
Expected: PASS. All pre-existing tests stay green; the two new files
(`QuestName.test.tsx` and `QuestStatusTabs.test.tsx`) are included and pass.

- [ ] **Step 2: Run lint**

Run:
```bash
cd services/atlas-ui && npm run lint
```
Expected: Zero errors or warnings introduced by this change. Pre-existing
warnings elsewhere in the repo are not this task's concern but should not
have grown in count.

- [ ] **Step 3: Run the production build**

Run:
```bash
cd services/atlas-ui && npm run build
```
Expected: `tsc -b` passes and `vite build` emits a `dist/` with no errors.
This verifies no type regressions leaked past the test runner's transform.

- [ ] **Step 4: Grep for dead references**

Run:
```bash
grep -n "QuestStatusCard" services/atlas-ui/src -r || echo "clean"
grep -n "ExternalLink" services/atlas-ui/src/components/features/quests -r || echo "clean"
grep -n "showProgress" services/atlas-ui/src/components/features/quests -r || echo "clean"
```
Expected: All three print `clean`. If any hit surfaces, remove the
stale reference and re-run Steps 1–3.

- [ ] **Step 5: Confirm git status is clean**

Run:
```bash
git status
```
Expected: `nothing to commit, working tree clean` (save for
`.claude/scheduled_tasks.lock` or other pre-existing untracked files
from the session start).

---

## Task 7: Wrap up

**Files:** none.

- [ ] **Step 1: Review the branch log**

Run:
```bash
git log --oneline main..HEAD
```
Expected three commits from this plan:
1. `feat(atlas-ui): add QuestName resolver component`
2. `test(atlas-ui): add QuestStatusTabs baseline integration tests`
3. `feat(atlas-ui): render quest status as responsive widget grid`

If the log differs, investigate before proceeding.

- [ ] **Step 2: Confirm acceptance criteria**

Walk the PRD § 10 checklist and confirm each item. Every item except the
`ExpirationTime`-serialization bug (explicitly out of scope) must be
satisfied by this plan's output. If one is not, open a follow-up task
rather than silently skipping it.

- [ ] **Step 3: Hand off**

Report: "task-024 implementation complete — 3 commits on the feature
branch, unit + integration tests added, manual smoke test passed (or
deferred with note). Ready for `/ultrareview` or PR."
