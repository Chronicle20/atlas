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
