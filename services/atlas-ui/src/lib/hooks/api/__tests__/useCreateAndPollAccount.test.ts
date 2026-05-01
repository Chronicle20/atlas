// services/atlas-ui/src/lib/hooks/api/__tests__/useCreateAndPollAccount.test.ts
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useCreateAndPollAccount } from "../useCreateAndPollAccount";
import { accountsService } from "@/services/api/accounts.service";

const tenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as never;

const accountFixture = (id: string, name: string) => ({
  id,
  type: "accounts",
  attributes: {
    name,
    gender: 0,
    loggedIn: 0,
    lastLogin: 0,
    characterSlots: 3,
    pinAttempts: 0,
    picAttempts: 0,
    tos: false,
  },
}) as never;

describe("useCreateAndPollAccount", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("starts in idle status", () => {
    const { result } = renderHook(() => useCreateAndPollAccount(tenant));
    expect(result.current.status).toBe("idle");
    expect(result.current.accountId).toBeNull();
    expect(result.current.errorKind).toBeNull();
  });

  it("transitions submitting → polling → success when account materialises", async () => {
    const create = vi.spyOn(accountsService, "createAccount").mockResolvedValue();
    const list = vi
      .spyOn(accountsService, "getAllAccounts")
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([accountFixture("99", "alice")]);

    const { result } = renderHook(() => useCreateAndPollAccount(tenant));

    void act(() => {
      void result.current.submit({ name: "alice", password: "secret1" });
    });

    await waitFor(() => expect(create).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(result.current.status).toBe("polling"));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    await waitFor(() => expect(result.current.status).toBe("success"));
    expect(result.current.accountId).toBe(99);
    expect(list).toHaveBeenCalled();
  });

  it("sets errorKind=duplicate-name on 409", async () => {
    const err = Object.assign(new Error("conflict"), { status: 409 });
    vi.spyOn(accountsService, "createAccount").mockRejectedValue(err);

    const { result } = renderHook(() => useCreateAndPollAccount(tenant));
    await act(async () => {
      await result.current.submit({ name: "alice", password: "secret1" });
    });

    expect(result.current.status).toBe("error");
    expect(result.current.errorKind).toBe("duplicate-name");
  });

  it("sets errorKind=generic on non-409 errors", async () => {
    const err = Object.assign(new Error("boom"), { status: 500 });
    vi.spyOn(accountsService, "createAccount").mockRejectedValue(err);

    const { result } = renderHook(() => useCreateAndPollAccount(tenant));
    await act(async () => {
      await result.current.submit({ name: "alice", password: "secret1" });
    });

    expect(result.current.status).toBe("error");
    expect(result.current.errorKind).toBe("generic");
    expect(result.current.errorMessage).toBe("boom");
  });

  it("transitions to timeout when polling exceeds 30s", async () => {
    vi.spyOn(accountsService, "createAccount").mockResolvedValue();
    vi.spyOn(accountsService, "getAllAccounts").mockResolvedValue([]);

    const { result } = renderHook(() => useCreateAndPollAccount(tenant));
    void act(() => {
      void result.current.submit({ name: "alice", password: "secret1" });
    });

    await waitFor(() => expect(result.current.status).toBe("polling"));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(31_000);
    });

    await waitFor(() => expect(result.current.status).toBe("timeout"));
  });

  it("retry() from timeout re-polls without re-creating the account", async () => {
    const create = vi.spyOn(accountsService, "createAccount").mockResolvedValue();
    const list = vi.spyOn(accountsService, "getAllAccounts").mockResolvedValue([]);

    const { result } = renderHook(() => useCreateAndPollAccount(tenant));
    void act(() => {
      void result.current.submit({ name: "alice", password: "secret1" });
    });
    await waitFor(() => expect(result.current.status).toBe("polling"));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(31_000);
    });
    await waitFor(() => expect(result.current.status).toBe("timeout"));

    const callsBeforeRetry = list.mock.calls.length;
    list.mockResolvedValueOnce([accountFixture("42", "alice")]);

    void act(() => {
      void result.current.retry();
    });
    await waitFor(() => expect(result.current.status).toBe("polling"));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    await waitFor(() => expect(result.current.status).toBe("success"));
    expect(result.current.accountId).toBe(42);
    expect(create).toHaveBeenCalledTimes(1); // not re-submitted
    expect(list.mock.calls.length).toBeGreaterThan(callsBeforeRetry);
  });

  it("reset() aborts polling and returns to idle", async () => {
    vi.spyOn(accountsService, "createAccount").mockResolvedValue();
    vi.spyOn(accountsService, "getAllAccounts").mockResolvedValue([]);

    const { result } = renderHook(() => useCreateAndPollAccount(tenant));
    void act(() => {
      void result.current.submit({ name: "alice", password: "secret1" });
    });
    await waitFor(() => expect(result.current.status).toBe("polling"));

    act(() => {
      result.current.reset();
    });

    await waitFor(() => expect(result.current.status).toBe("idle"));
    expect(result.current.accountId).toBeNull();
  });

  it("bails out if tenant.id changes mid-poll", async () => {
    vi.spyOn(accountsService, "createAccount").mockResolvedValue();
    const list = vi.spyOn(accountsService, "getAllAccounts").mockResolvedValue([]);

    const initialTenant = tenant;
    const { result, rerender } = renderHook(
      ({ t }: { t: typeof tenant }) => useCreateAndPollAccount(t),
      { initialProps: { t: initialTenant } }
    );

    void act(() => {
      void result.current.submit({ name: "alice", password: "secret1" });
    });
    await waitFor(() => expect(result.current.status).toBe("polling"));

    // tenant swap mid-flight
    rerender({
      t: { ...(initialTenant as object), id: "t2" } as never,
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    // tenant change should abort the loop; status falls back to idle.
    await waitFor(() => expect(result.current.status).toBe("idle"));
    // Confirm no further polling fires after the tenant change.
    const callsAfterAbort = list.mock.calls.length;
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(list.mock.calls.length).toBe(callsAfterAbort);
  });
});
