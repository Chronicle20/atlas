import { api } from "@/lib/api/client";
import {
  buildQueryString,
  type ServiceOptions,
  type QueryOptions,
} from "@/lib/api/query-params";
import {
  fetchAll,
  fetchPaged,
  type PagedResult,
} from "@/services/api/pagination";
import { tenantHeaders } from "@/lib/headers";
import type { Account, AccountAttributes } from "@/types/models/account";
import type { Tenant } from "@/types/models/tenant";

const BASE_PATH = "/api/accounts";

interface AccountQueryOptions extends QueryOptions {
  name?: string;
  loggedIn?: boolean;
  language?: string;
  country?: string;
}

function transformAccount(data: Account): Account {
  return {
    ...data,
    attributes: {
      ...data.attributes,
      loggedIn: Number(data.attributes.loggedIn),
      lastLogin: Number(data.attributes.lastLogin),
      gender: Number(data.attributes.gender),
      pinAttempts: Number(data.attributes.pinAttempts),
      picAttempts: Number(data.attributes.picAttempts),
      birthDate: Number(data.attributes.birthDate ?? 0),
      tos: Boolean(data.attributes.tos),
    },
  };
}

function sortAccounts(accounts: Account[]): Account[] {
  return accounts.sort((a, b) =>
    a.attributes.name
      .toLowerCase()
      .localeCompare(b.attributes.name.toLowerCase()),
  );
}

function buildAccountQuery(options?: AccountQueryOptions): QueryOptions {
  const queryOptions: QueryOptions = { ...options };
  if (!options) return queryOptions;

  const filters: Record<string, unknown> = { ...queryOptions.filters };
  if (options.name) filters.name = options.name;
  if (options.loggedIn !== undefined) filters.loggedIn = options.loggedIn;
  if (options.language) filters.language = options.language;
  if (options.country) filters.country = options.country;
  if (Object.keys(filters).length > 0) queryOptions.filters = filters;

  return queryOptions;
}

export const accountsService = {
  /**
   * Get every account for a tenant (matching `options`), draining all pages
   * (task-117). Used by consumers that genuinely need the whole collection
   * (search, logged-in roster, stats, the Characters-page account join).
   */
  async getAllAccounts(options?: AccountQueryOptions): Promise<Account[]> {
    const queryOptions = buildAccountQuery(options);
    const accounts = await fetchAll<Account>(
      `${BASE_PATH}${buildQueryString(queryOptions)}`,
      undefined,
      queryOptions,
    );
    return sortAccounts(accounts.map(transformAccount));
  },

  /**
   * Get a single page of accounts (matching `options`). Used by the
   * Accounts list view (task-117), which pages server-side.
   */
  async getAccountsPage(
    page: { number: number; size: number },
    options?: AccountQueryOptions,
  ): Promise<PagedResult<Account>> {
    const queryOptions = buildAccountQuery(options);
    const result = await fetchPaged<Account>(
      `${BASE_PATH}${buildQueryString(queryOptions)}`,
      page,
      queryOptions,
    );
    return {
      data: sortAccounts(result.data.map(transformAccount)),
      meta: result.meta,
    };
  },

  async getAccountById(id: string, options?: ServiceOptions): Promise<Account> {
    const account = await api.getOne<Account>(`${BASE_PATH}/${id}`, options);
    return transformAccount(account);
  },

  /**
   * Sets an account's birth date (yyyymmdd as an integer, 0 = unset).
   *
   * The whole current attribute set is resent, not just birthDate. atlas-account's
   * PATCH handler rebuilds a full account Model from the request body and diffs
   * it against the stored row (account/processor.go Update): pinAttempts,
   * picAttempts and gender are written on ANY difference, including a difference
   * from an absent field defaulting to 0. Sending birthDate alone would therefore
   * silently reset those three. `account` must be the freshly-fetched account,
   * for the same reason.
   */
  async updateAccountBirthDate(
    account: Account,
    birthDate: number,
    options?: ServiceOptions,
  ): Promise<Account> {
    const body = {
      data: {
        id: account.id,
        type: "accounts",
        attributes: { ...account.attributes, birthDate },
      },
    };
    const updated = await api.patch<Account>(
      `${BASE_PATH}/${account.id}`,
      body,
      options,
    );
    return transformAccount(updated);
  },

  async accountExists(id: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await accountsService.getAccountById(id, options);
      return true;
    } catch (error) {
      if (
        error &&
        typeof error === "object" &&
        "status" in error &&
        (error as { status: number }).status === 404
      ) {
        return false;
      }
      throw error;
    }
  },

  async searchAccountsByName(
    namePattern: string,
    options?: ServiceOptions,
  ): Promise<Account[]> {
    return accountsService.getAllAccounts({
      ...options,
      search: namePattern,
      name: namePattern,
    });
  },

  async getLoggedInAccounts(options?: ServiceOptions): Promise<Account[]> {
    return accountsService.getAllAccounts({ ...options, loggedIn: true });
  },

  async terminateAccountSession(
    accountId: string,
    options?: ServiceOptions,
  ): Promise<void> {
    return api.delete(`${BASE_PATH}/${accountId}/session`, options);
  },

  async deleteAccount(
    accountId: string,
    options?: ServiceOptions,
  ): Promise<void> {
    return api.delete(`${BASE_PATH}/${accountId}`, options);
  },

  /**
   * Account-count stats for a tenant (total / logged-in). Character-slot
   * totals used to be included here for free, because `characterSlots` was
   * a flat attribute already present on every fetched Account. Slots are
   * now a world-scoped sub-resource (`accounts/{id}/worlds/{worldId}/character-slots`,
   * task-246), so a tenant-wide slot total would mean issuing
   * accounts.length * worlds.length additional requests from a function
   * that already drains every account for the tenant — and the resulting
   * number would sum caps across unrelated worlds, which isn't a
   * meaningful single statistic. No page renders a slot total today,
   * so it is dropped rather than computed wrong or left NaN; a per-world
   * breakdown belongs on a per-world view (see CharactersPanel/
   * WorldCharactersSection), not this account-count summary.
   */
  async getAccountStats(options?: ServiceOptions): Promise<{
    total: number;
    loggedIn: number;
  }> {
    const accounts = await accountsService.getAllAccounts(options);
    const total = accounts.length;
    const loggedIn = accounts.filter(
      (acc) => acc.attributes.loggedIn > 0,
    ).length;
    return { total, loggedIn };
  },

  async createAccount(
    tenant: Tenant,
    payload: { name: string; password: string },
  ): Promise<void> {
    const headers = tenantHeaders(tenant);
    headers.set("Content-Type", "application/json");

    const body = {
      data: {
        type: "accounts",
        attributes: {
          name: payload.name,
          password: payload.password,
        },
      },
    };

    const response = await fetch(`${BASE_PATH}/`, {
      method: "POST",
      headers,
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      let message = `createAccount failed with status ${response.status}`;
      try {
        const errBody = (await response.json()) as {
          error?: string;
          message?: string;
        };
        if (errBody.error) message = errBody.error;
        else if (errBody.message) message = errBody.message;
      } catch {
        // non-JSON error body; keep the default message
      }
      const err = new Error(message) as Error & { status?: number };
      err.status = response.status;
      throw err;
    }
  },

  async terminateMultipleSessions(
    accountIds: string[],
    options?: ServiceOptions,
  ): Promise<{
    successful: string[];
    failed: Array<{ id: string; error: string }>;
  }> {
    const successful: string[] = [];
    const failed: Array<{ id: string; error: string }> = [];
    const concurrency = 3;

    for (let i = 0; i < accountIds.length; i += concurrency) {
      const batch = accountIds.slice(i, i + concurrency);
      const results = await Promise.all(
        batch.map(async (accountId) => {
          try {
            await accountsService.terminateAccountSession(accountId, options);
            return { success: true as const, accountId };
          } catch (error) {
            return {
              success: false as const,
              accountId,
              error: error instanceof Error ? error.message : "Unknown error",
            };
          }
        }),
      );

      for (const result of results) {
        if (result.success) successful.push(result.accountId);
        else failed.push({ id: result.accountId, error: result.error });
      }
    }

    return { successful, failed };
  },
};

export type { Account, AccountAttributes, AccountQueryOptions, Tenant };
