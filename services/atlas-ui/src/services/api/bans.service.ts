import { api } from "@/lib/api/client";
import { type ServiceOptions, type QueryOptions, type ValidationError } from "@/lib/api/query-params";
import { fetchAll, fetchPaged, type PagedResult } from "@/services/api/pagination";
import type { Ban, BanAttributes, CreateBanRequest, CheckBanResult, BanType } from "@/types/models/ban";

const BASE_PATH = "/api/bans";

interface BanQueryOptions extends QueryOptions {
  type?: BanType;
}

interface CheckBanParams {
  ip?: string;
  hwid?: string;
  accountId?: number;
}

function transformBan(data: Ban): Ban {
  return {
    ...data,
    attributes: {
      ...data.attributes,
      banType: Number(data.attributes.banType) as typeof data.attributes.banType,
      reasonCode: Number(data.attributes.reasonCode) as typeof data.attributes.reasonCode,
      permanent: Boolean(data.attributes.permanent),
      expiresAt: String(data.attributes.expiresAt),
    },
  };
}

function sortBans(bans: Ban[]): Ban[] {
  return bans.sort((a, b) => Number(b.id) - Number(a.id));
}

function validateCreateBan(data: CreateBanRequest): ValidationError[] {
  const errors: ValidationError[] = [];

  if (!data.value || data.value.trim().length === 0) {
    errors.push({ field: "value", message: "Ban value is required" });
  }
  if (data.banType === 0 && data.value) {
    const ipRegex = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/;
    if (!ipRegex.test(data.value)) {
      errors.push({ field: "value", message: "Invalid IP address or CIDR format" });
    }
  }
  if (!data.permanent && (!data.expiresAt || new Date(data.expiresAt) <= new Date())) {
    errors.push({ field: "expiresAt", message: "Expiration date must be in the future for non-permanent bans" });
  }

  return errors;
}

function banListUrl(options?: BanQueryOptions): string {
  return options?.type !== undefined ? `${BASE_PATH}?type=${options.type}` : BASE_PATH;
}

export const bansService = {
  /**
   * Get every ban (matching `options`), draining all pages (task-117). Used
   * by consumers that genuinely need the whole collection (e.g. looking up
   * the active ban for one account by type).
   */
  async getAllBans(options?: BanQueryOptions): Promise<Ban[]> {
    const bans = await fetchAll<Ban>(banListUrl(options), undefined, options);
    return sortBans(bans.map(transformBan));
  },

  /**
   * Get a single page of bans (matching `options`). Used by the Bans list
   * view (task-117), which pages server-side.
   */
  async getBansPage(
    page: { number: number; size: number },
    options?: BanQueryOptions,
  ): Promise<PagedResult<Ban>> {
    const result = await fetchPaged<Ban>(banListUrl(options), page, options);
    return { data: sortBans(result.data.map(transformBan)), meta: result.meta };
  },

  async getBanById(id: string, options?: ServiceOptions): Promise<Ban> {
    const ban = await api.getOne<Ban>(`${BASE_PATH}/${id}`, options);
    return transformBan(ban);
  },

  async banExists(id: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await bansService.getBanById( id, options);
      return true;
    } catch (error) {
      if (error && typeof error === "object" && "status" in error && (error as { status: number }).status === 404) {
        return false;
      }
      throw error;
    }
  },

  async createBan(data: CreateBanRequest, options?: ServiceOptions): Promise<Ban> {
    const validationErrors = validateCreateBan(data);
    if (validationErrors.length > 0) {
      throw new Error(`Validation failed: ${validationErrors.map(e => e.message).join(", ")}`);
    }
    const response = await api.post<{ data: Ban }>(BASE_PATH, data, options);
    return transformBan(response.data);
  },

  async deleteBan(id: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASE_PATH}/${id}`, options);
  },

  async expireBan(id: string, options?: ServiceOptions): Promise<void> {
    await api.post(`${BASE_PATH}/${id}/expire`, {}, options);
  },

  async checkBan(params: CheckBanParams, options?: ServiceOptions): Promise<CheckBanResult> {
    const queryParams = new URLSearchParams();
    if (params.ip) queryParams.append("ip", params.ip);
    if (params.hwid) queryParams.append("hwid", params.hwid);
    if (params.accountId) queryParams.append("accountId", params.accountId.toString());

    const url = `${BASE_PATH}/check?${queryParams.toString()}`;
    const response = await api.get<{ data: CheckBanResult }>(url, options);
    return response.data;
  },

  async getBansByType(type: BanType, options?: ServiceOptions): Promise<Ban[]> {
    return bansService.getAllBans({ ...options, type });
  },
};

export type { Ban, BanAttributes, CreateBanRequest, CheckBanResult, BanQueryOptions, CheckBanParams };
