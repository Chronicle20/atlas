import { render, screen } from "@testing-library/react";
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
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: fakeTenant }),
}));

const getCollectionMock = vi.fn();
const listCardsMock = vi.fn();
vi.mock("@/services/api/monster-book.service", () => ({
  monsterBookService: {
    getCollection: (...a: unknown[]) => getCollectionMock(...a),
    listCards: (...a: unknown[]) => listCardsMock(...a),
  },
}));

// CardRow / CoverImage call api.getOne to chase the consumable→monster
// relation. The widget should still render its skeleton/empty/header
// states without these resolving, so we stub them to never settle.
vi.mock("@/lib/api/client", () => ({
  api: {
    getOne: vi.fn(() => new Promise(() => {})),
  },
}));

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
  });

  it("renders the loading skeleton while the collection query is pending", () => {
    getCollectionMock.mockImplementation(() => new Promise(() => {}));
    listCardsMock.mockImplementation(() => new Promise(() => {}));

    renderWidget();

    expect(screen.getByTestId("monster-book-loading")).toBeInTheDocument();
  });

  it("renders the book level + stats once the collection resolves", async () => {
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
    getCollectionMock.mockResolvedValue(collection);
    listCardsMock.mockResolvedValue(cards);

    renderWidget();

    expect(await screen.findByText("Lv. 4")).toBeInTheDocument();
    expect(screen.getByText("EXP Bonus")).toBeInTheDocument();
    expect(screen.getByText("5%")).toBeInTheDocument();
    expect(screen.getByText("20")).toBeInTheDocument();
  });

  it("renders the empty-state message when the collection has zero cards", async () => {
    const collection: MonsterBookCollection = {
      characterId: 12345,
      bookLevel: 0,
      normalCount: 0,
      specialCount: 0,
      totalUniqueCards: 0,
      coverCardId: 0,
      expBonusPercent: 0,
    };
    getCollectionMock.mockResolvedValue(collection);
    listCardsMock.mockResolvedValue([]);

    renderWidget();

    expect(await screen.findByText(/no cards collected yet/i)).toBeInTheDocument();
  });
});
