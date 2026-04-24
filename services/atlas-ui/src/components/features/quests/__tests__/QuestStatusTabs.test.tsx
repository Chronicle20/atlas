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
