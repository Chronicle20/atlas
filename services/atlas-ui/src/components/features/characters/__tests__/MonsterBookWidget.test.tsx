import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MonsterBookWidget } from "../MonsterBookWidget";
import type { MonsterBookCard, MonsterBookCollection } from "@/types/monster-book";

// Tenant context provides the four-header tenant; the widget pulls its
// region/version off `activeTenant` to build asset URLs. Stub it with a
// stable fake so we never touch the real provider chain.
const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
};
const useTenantMock = vi.fn(() => ({ activeTenant: fakeTenant }));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => useTenantMock(),
}));

const useMonsterBookCollectionMock = vi.fn();
const useMonsterBookCardsMock = vi.fn();
const useMonsterBookCardNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useMonsterBook", () => ({
  useMonsterBookCollection: (...a: unknown[]) =>
    useMonsterBookCollectionMock(...a),
  useMonsterBookCards: (...a: unknown[]) => useMonsterBookCardsMock(...a),
  useMonsterBookCardName: (...a: unknown[]) =>
    useMonsterBookCardNameMock(...a),
}));

const toastErrorMock = vi.fn();
vi.mock("sonner", () => ({
  toast: {
    error: (...a: unknown[]) => toastErrorMock(...a),
  },
}));

function makeCollectionResult(overrides: Partial<{
  data: MonsterBookCollection | undefined;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  refetch: () => void;
}> = {}) {
  return {
    data: undefined,
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
    ...overrides,
  };
}

function makeCardsResult(overrides: Partial<{
  data: { pages: MonsterBookCard[][] } | undefined;
  isFetchingNextPage: boolean;
  hasNextPage: boolean;
  fetchNextPage: () => void;
}> = {}) {
  return {
    data: undefined,
    isFetchingNextPage: false,
    hasNextPage: false,
    fetchNextPage: vi.fn(),
    ...overrides,
  };
}

function renderWidget(characterId = 12345) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MonsterBookWidget characterId={characterId} />
    </QueryClientProvider>,
  );
}

describe("MonsterBookWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTenantMock.mockReturnValue({ activeTenant: fakeTenant });
    // Cards + card-name hooks default to "loading, empty" so each test
    // only has to override the bits it cares about.
    useMonsterBookCardsMock.mockReturnValue(makeCardsResult());
    useMonsterBookCardNameMock.mockReturnValue({
      monsterId: null,
      name: null,
      isLoading: false,
      isError: false,
    });
  });

  it("renders the loading skeleton while the collection query is pending", () => {
    useMonsterBookCollectionMock.mockReturnValue(
      makeCollectionResult({ isLoading: true }),
    );

    renderWidget();

    expect(screen.getByTestId("monster-book-loading")).toBeInTheDocument();
  });

  it("renders the book level + stats once the collection resolves", () => {
    const collection: MonsterBookCollection = {
      characterId: 12345,
      bookLevel: 4,
      normalCount: 18,
      specialCount: 2,
      totalUniqueCards: 20,
      coverCardId: 2380000,
      expBonusPercent: 5,
    };
    const cards: MonsterBookCard[] = [
      { cardId: 2380000, level: 5, isSpecial: false, firstAcquiredAt: "2024-01-01T00:00:00Z" },
    ];
    useMonsterBookCollectionMock.mockReturnValue(
      makeCollectionResult({ data: collection }),
    );
    useMonsterBookCardsMock.mockReturnValue(
      makeCardsResult({ data: { pages: [cards] } }),
    );

    renderWidget();

    expect(screen.getByText("Lv. 4")).toBeInTheDocument();
    expect(screen.getByText("EXP Bonus")).toBeInTheDocument();
    expect(screen.getByText("5%")).toBeInTheDocument();
    expect(screen.getByText("20")).toBeInTheDocument();
  });

  it("renders the empty-state message when the collection has zero cards", () => {
    const collection: MonsterBookCollection = {
      characterId: 12345,
      bookLevel: 0,
      normalCount: 0,
      specialCount: 0,
      totalUniqueCards: 0,
      coverCardId: 0,
      expBonusPercent: 0,
    };
    useMonsterBookCollectionMock.mockReturnValue(
      makeCollectionResult({ data: collection }),
    );
    useMonsterBookCardsMock.mockReturnValue(
      makeCardsResult({ data: { pages: [[]] } }),
    );

    renderWidget();

    expect(screen.getByText(/no cards collected yet/i)).toBeInTheDocument();
  });

  it("shows the inline error chip + Retry and surfaces a toast on failure", () => {
    const refetch = vi.fn();
    useMonsterBookCollectionMock.mockReturnValue(
      makeCollectionResult({
        isError: true,
        error: new Error("boom"),
        refetch,
      }),
    );

    renderWidget();

    expect(screen.getByText(/failed to load monster book/i)).toBeInTheDocument();
    const retry = screen.getByRole("button", { name: /retry/i });
    fireEvent.click(retry);
    expect(refetch).toHaveBeenCalled();
    expect(toastErrorMock).toHaveBeenCalled();
  });

  it("keeps queries disabled when no tenant is selected and shows loading", () => {
    useTenantMock.mockReturnValue({ activeTenant: null as never });
    // The hooks would, in production, return isLoading=false because
    // their `enabled` gate is off. The widget must still render its
    // skeleton rather than the "Failed" chip.
    useMonsterBookCollectionMock.mockReturnValue(
      makeCollectionResult({ data: undefined, isLoading: false }),
    );

    renderWidget();

    expect(screen.getByTestId("monster-book-loading")).toBeInTheDocument();
  });

  it("loads more pages when the user clicks Load more", () => {
    const collection: MonsterBookCollection = {
      characterId: 12345,
      bookLevel: 1,
      normalCount: 1,
      specialCount: 0,
      totalUniqueCards: 1,
      coverCardId: 0,
      expBonusPercent: 0,
    };
    const cards: MonsterBookCard[] = [
      { cardId: 2380001, level: 1, isSpecial: false, firstAcquiredAt: "2024-01-01T00:00:00Z" },
    ];
    const fetchNextPage = vi.fn();
    useMonsterBookCollectionMock.mockReturnValue(
      makeCollectionResult({ data: collection }),
    );
    useMonsterBookCardsMock.mockReturnValue(
      makeCardsResult({
        data: { pages: [cards] },
        hasNextPage: true,
        fetchNextPage,
      }),
    );

    renderWidget();

    const loadMore = screen.getByRole("button", { name: /load more/i });
    fireEvent.click(loadMore);
    expect(fetchNextPage).toHaveBeenCalled();
  });
});
