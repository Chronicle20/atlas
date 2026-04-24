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
