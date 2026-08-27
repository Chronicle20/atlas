import { describe, expect, it, vi, beforeEach } from "vitest";

const getMock = vi.fn();
const patchMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    get: (...args: unknown[]) => getMock(...args),
    getList: vi.fn(),
    getOne: vi.fn(),
    patch: (...args: unknown[]) => patchMock(...args),
    delete: vi.fn(),
  },
}));

import { accountsService } from "@/services/api/accounts.service";
import type { Account } from "@/types/models/account";

function makeAccount(id: string, name: string): Account {
  return {
    id,
    attributes: {
      name,
      pin: "",
      pic: "",
      pinAttempts: 0,
      picAttempts: 0,
      loggedIn: 0,
      lastLogin: 0,
      gender: 0,
      tos: true,
      language: "en",
      country: "US",
      birthDate: 0,
    },
  };
}

describe("accountsService.getAccountsPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("appends page params, filters, and returns data + meta", async () => {
    getMock.mockResolvedValue({
      data: [makeAccount("2", "bravo"), makeAccount("1", "alpha")],
      meta: { total: 5, page: { number: 2, size: 2, last: 3 } },
    });

    const result = await accountsService.getAccountsPage(
      { number: 2, size: 2 },
      { loggedIn: true },
    );

    expect(result.meta).toEqual({
      total: 5,
      page: { number: 2, size: 2, last: 3 },
    });
    // Sorted by name within the page, as the pre-pagination behavior did for the whole collection.
    expect(result.data.map((a) => a.attributes.name)).toEqual([
      "alpha",
      "bravo",
    ]);

    const [calledUrl] = getMock.mock.calls[0] as [string];
    const params = new URL(calledUrl, "http://example.test").searchParams;
    expect(params.get("page[number]")).toBe("2");
    expect(params.get("page[size]")).toBe("2");
    expect(params.get("filter[loggedIn]")).toBe("true");
  });
});

describe("accountsService.getAllAccounts", () => {
  beforeEach(() => vi.clearAllMocks());

  it("drains every page into a single sorted array", async () => {
    getMock
      .mockResolvedValueOnce({
        data: [makeAccount("1", "zeta")],
        meta: { total: 2, page: { number: 1, size: 1, last: 2 } },
      })
      .mockResolvedValueOnce({
        data: [makeAccount("2", "alpha")],
        meta: { total: 2, page: { number: 2, size: 1, last: 2 } },
      });

    const result = await accountsService.getAllAccounts();

    expect(result.map((a) => a.attributes.name)).toEqual(["alpha", "zeta"]);
    expect(getMock).toHaveBeenCalledTimes(2);
  });
});

// Regression test for bug-cash-shop-live-testing.md Defect C:
// api.patch returns the raw JSON:API envelope ({data:{...}}), not the
// unwrapped resource -- unlike api.getOne. updateAccountBirthDate must
// unwrap `.data` before handing the result to transformAccount, or
// transformAccount's first field access throws
// "cannot read properties of undefined (reading 'loggedIn')" even though
// the PATCH itself succeeded.
describe("accountsService.updateAccountBirthDate", () => {
  beforeEach(() => vi.clearAllMocks());

  it("resolves against a mocked JSON:API envelope response", async () => {
    const account = makeAccount("1", "alpha");
    patchMock.mockResolvedValue({
      data: {
        ...makeAccount("1", "alpha"),
        attributes: { ...account.attributes, birthDate: 19900101 },
      },
    });

    const result = await accountsService.updateAccountBirthDate(
      account,
      19900101,
    );

    expect(result.attributes.birthDate).toBe(19900101);
    expect(result.attributes.loggedIn).toBe(0);
  });
});
