import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type { LoginHistoryEntry } from "@/types/models/ban";

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
  async getByIp(ip: string, options?: ServiceOptions): Promise<LoginHistoryEntry[]> {
    const entries = await api.getList<LoginHistoryEntry>(
      `${BASE_PATH}?ip=${encodeURIComponent(ip)}`,
      options,
    );
    return entries.map(transformEntry);
  },

  async getByHwid(hwid: string, options?: ServiceOptions): Promise<LoginHistoryEntry[]> {
    const entries = await api.getList<LoginHistoryEntry>(
      `${BASE_PATH}?hwid=${encodeURIComponent(hwid)}`,
      options,
    );
    return entries.map(transformEntry);
  },

  async getByAccountId(accountId: number, options?: ServiceOptions): Promise<LoginHistoryEntry[]> {
    const entries = await api.getList<LoginHistoryEntry>(
      `${BASE_PATH}/accounts/${accountId}`,
      options,
    );
    return entries.map(transformEntry);
  },

  async search(
    criteria: { ip?: string; hwid?: string; accountId?: number },
    options?: ServiceOptions,
  ): Promise<LoginHistoryEntry[]> {
    // Prioritise by specificity: accountId > hwid > ip
    if (criteria.accountId) return loginHistoryService.getByAccountId( criteria.accountId, options);
    if (criteria.hwid) return loginHistoryService.getByHwid( criteria.hwid, options);
    if (criteria.ip) return loginHistoryService.getByIp( criteria.ip, options);
    return [];
  },
};

export type { LoginHistoryEntry };
