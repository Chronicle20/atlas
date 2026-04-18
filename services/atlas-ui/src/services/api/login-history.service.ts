import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type { LoginHistoryEntry } from "@/types/models/ban";
import type { Tenant } from "@/types/models/tenant";

const BASE_PATH = "/api/history";

function transformEntry(data: LoginHistoryEntry): LoginHistoryEntry {
  return {
    ...data,
    attributes: {
      ...data.attributes,
      accountId: Number(data.attributes.accountId),
      success: Boolean(data.attributes.success),
    },
  };
}

export const loginHistoryService = {
  async getByIp(_tenant: Tenant, ip: string, options?: ServiceOptions): Promise<LoginHistoryEntry[]> {
    const entries = await api.getList<LoginHistoryEntry>(
      `${BASE_PATH}?ip=${encodeURIComponent(ip)}`,
      options,
    );
    return entries.map(transformEntry);
  },

  async getByHwid(_tenant: Tenant, hwid: string, options?: ServiceOptions): Promise<LoginHistoryEntry[]> {
    const entries = await api.getList<LoginHistoryEntry>(
      `${BASE_PATH}?hwid=${encodeURIComponent(hwid)}`,
      options,
    );
    return entries.map(transformEntry);
  },

  async getByAccountId(_tenant: Tenant, accountId: number, options?: ServiceOptions): Promise<LoginHistoryEntry[]> {
    const entries = await api.getList<LoginHistoryEntry>(
      `${BASE_PATH}/accounts/${accountId}`,
      options,
    );
    return entries.map(transformEntry);
  },

  async search(
    tenant: Tenant,
    criteria: { ip?: string; hwid?: string; accountId?: number },
    options?: ServiceOptions,
  ): Promise<LoginHistoryEntry[]> {
    // Prioritise by specificity: accountId > hwid > ip
    if (criteria.accountId) return loginHistoryService.getByAccountId(tenant, criteria.accountId, options);
    if (criteria.hwid) return loginHistoryService.getByHwid(tenant, criteria.hwid, options);
    if (criteria.ip) return loginHistoryService.getByIp(tenant, criteria.ip, options);
    return [];
  },
};

export type { LoginHistoryEntry };
