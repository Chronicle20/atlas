import { api } from "@/lib/api/client";
import { buildQueryString, type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
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
      characterSlots: Number(data.attributes.characterSlots),
      pinAttempts: Number(data.attributes.pinAttempts),
      picAttempts: Number(data.attributes.picAttempts),
      tos: Boolean(data.attributes.tos),
    },
  };
}

function sortAccounts(accounts: Account[]): Account[] {
  return accounts.sort((a, b) =>
    a.attributes.name.toLowerCase().localeCompare(b.attributes.name.toLowerCase()),
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
  async getAllAccounts(_tenant: Tenant, options?: AccountQueryOptions): Promise<Account[]> {
    const queryOptions = buildAccountQuery(options);
    const accounts = await api.getList<Account>(
      `${BASE_PATH}${buildQueryString(queryOptions)}`,
      queryOptions,
    );
    return sortAccounts(accounts.map(transformAccount));
  },

  async getAccountById(_tenant: Tenant, id: string, options?: ServiceOptions): Promise<Account> {
    const account = await api.getOne<Account>(`${BASE_PATH}/${id}`, options);
    return transformAccount(account);
  },

  async accountExists(tenant: Tenant, id: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await accountsService.getAccountById(tenant, id, options);
      return true;
    } catch (error) {
      if (error && typeof error === "object" && "status" in error && (error as { status: number }).status === 404) {
        return false;
      }
      throw error;
    }
  },

  async searchAccountsByName(tenant: Tenant, namePattern: string, options?: ServiceOptions): Promise<Account[]> {
    return accountsService.getAllAccounts(tenant, {
      ...options,
      search: namePattern,
      name: namePattern,
    });
  },

  async getLoggedInAccounts(tenant: Tenant, options?: ServiceOptions): Promise<Account[]> {
    return accountsService.getAllAccounts(tenant, { ...options, loggedIn: true });
  },

  async terminateAccountSession(_tenant: Tenant, accountId: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASE_PATH}/${accountId}/session`, options);
  },

  async deleteAccount(_tenant: Tenant, accountId: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASE_PATH}/${accountId}`, options);
  },

  async getAccountStats(tenant: Tenant, options?: ServiceOptions): Promise<{
    total: number;
    loggedIn: number;
    totalCharacterSlots: number;
    averageCharacterSlots: number;
  }> {
    const accounts = await accountsService.getAllAccounts(tenant, options);
    const total = accounts.length;
    const loggedIn = accounts.filter(acc => acc.attributes.loggedIn > 0).length;
    const totalCharacterSlots = accounts.reduce((sum, acc) => sum + acc.attributes.characterSlots, 0);
    return {
      total,
      loggedIn,
      totalCharacterSlots,
      averageCharacterSlots: total > 0 ? totalCharacterSlots / total : 0,
    };
  },

  async terminateMultipleSessions(
    tenant: Tenant,
    accountIds: string[],
    options?: ServiceOptions,
  ): Promise<{ successful: string[]; failed: Array<{ id: string; error: string }> }> {
    const successful: string[] = [];
    const failed: Array<{ id: string; error: string }> = [];
    const concurrency = 3;

    for (let i = 0; i < accountIds.length; i += concurrency) {
      const batch = accountIds.slice(i, i + concurrency);
      const results = await Promise.all(
        batch.map(async (accountId) => {
          try {
            await accountsService.terminateAccountSession(tenant, accountId, options);
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

export type { Account, AccountAttributes, AccountQueryOptions };
