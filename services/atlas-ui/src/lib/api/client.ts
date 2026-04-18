/**
 * Thin HTTP client for atlas-ui.
 *
 * Responsibilities kept intentionally narrow:
 * - Inject the four tenant headers via lib/headers.
 * - Apply a small retry budget on 5xx / 429.
 * - Parse JSON:API error payloads into ApiError via lib/api/errors.
 * - Expose get / getList / getOne / post / put / patch / delete + upload / download.
 *
 * Caching, request deduplication, stream downloads, progress tracking,
 * and stale-while-revalidate lived here in the Next.js era. They were
 * removed after the React Router + React Query migration — React Query
 * owns all of that now (see CLAUDE.md and docs/TODO.md). The CacheOptions
 * / ProgressCallback / staleWhileRevalidate / skipDeduplication / onProgress
 * fields on ApiRequestOptions are kept as a compatibility shim (several
 * services still pass them through) but the client ignores them.
 */

import type { Tenant } from "@/types/models/tenant";
import type { ApiListResponse, ApiSingleResponse } from "@/types/api/responses";
import { tenantHeaders } from "@/lib/headers";
import { createApiErrorFromResponse } from "@/types/api/errors";
import { sanitizeErrorData } from "@/lib/api/errors";

const DEFAULT_TIMEOUT_MS = 30_000;
const DEFAULT_MAX_RETRIES = 3;
const RETRY_BASE_DELAY_MS = 1_000;
const RETRY_MAX_DELAY_MS = 10_000;

/** Progress info (kept for back-compat; upload/download never emit it now). */
export interface ProgressInfo {
  total: number | undefined;
  loaded: number;
  percentage: number | undefined;
  rate: number | undefined;
  timeRemaining: number | undefined;
  done: boolean;
}

export type ProgressCallback = (progress: ProgressInfo) => void;

/** Cache options (kept as no-op shim for base.service). */
export interface CacheOptions {
  ttl?: number;
  keyPrefix?: string;
  staleWhileRevalidate?: boolean;
  maxStaleTime?: number;
}

export interface ApiRequestOptions extends Omit<RequestInit, "method" | "body" | "cache"> {
  /** Override request timeout (default: 30s). */
  timeout?: number;
  /** Skip tenant header injection — used by unauthenticated endpoints if any. */
  skipTenantHeaders?: boolean;
  /** Additional headers merged with the defaults. */
  headers?: HeadersInit;
  /** Retry budget for idempotent requests (default: 3). */
  maxRetries?: number;
  /** External AbortSignal for caller-controlled cancellation. */
  signal?: AbortSignal;

  /** @deprecated React Query owns caching; ignored. */
  cacheConfig?: CacheOptions | false;
  /** @deprecated React Query owns caching; ignored. */
  skipDeduplication?: boolean;
  /** @deprecated Progress tracking is not implemented; ignored. */
  onProgress?: ProgressCallback;
}

class ApiClient {
  private baseUrl: string;
  private tenant: Tenant | null = null;

  constructor() {
    this.baseUrl =
      import.meta.env.VITE_ROOT_API_URL ||
      (typeof window !== "undefined" ? window.location.origin : "");
  }

  setTenant(tenant: Tenant | null): void {
    this.tenant = tenant;
  }

  getTenant(): Tenant | null {
    return this.tenant;
  }

  private createHeaders(contentType: string | null, options?: ApiRequestOptions): Headers {
    const headers = new Headers();

    if (this.tenant && !options?.skipTenantHeaders) {
      tenantHeaders(this.tenant).forEach((value, key) => headers.set(key, value));
    }

    if (contentType) {
      headers.set("Content-Type", contentType);
    }

    if (options?.headers) {
      new Headers(options.headers).forEach((value, key) => headers.set(key, value));
    }

    return headers;
  }

  private createTimeoutSignal(timeoutMs: number, external?: AbortSignal): AbortSignal {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

    const abort = () => controller.abort();
    if (external) {
      if (external.aborted) controller.abort();
      else external.addEventListener("abort", abort, { once: true });
    }

    controller.signal.addEventListener(
      "abort",
      () => {
        clearTimeout(timeoutId);
        if (external) external.removeEventListener("abort", abort);
      },
      { once: true }
    );

    return controller.signal;
  }

  private async fetchWithRetry(
    url: string,
    init: RequestInit,
    options?: ApiRequestOptions
  ): Promise<Response> {
    const maxRetries = options?.maxRetries ?? DEFAULT_MAX_RETRIES;
    const timeoutMs = options?.timeout ?? DEFAULT_TIMEOUT_MS;

    let lastError: unknown;
    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      const signal = this.createTimeoutSignal(timeoutMs, options?.signal);

      try {
        const response = await fetch(url, { ...init, signal });
        if (!response.ok && attempt < maxRetries && (response.status >= 500 || response.status === 429)) {
          await this.sleep(this.retryDelay(attempt));
          continue;
        }
        return response;
      } catch (error) {
        lastError = error;
        if (options?.signal?.aborted) throw error;
        if (attempt >= maxRetries) break;
        await this.sleep(this.retryDelay(attempt));
      }
    }
    throw lastError ?? new Error("Request failed");
  }

  private retryDelay(attempt: number): number {
    return Math.min(RETRY_BASE_DELAY_MS * 2 ** attempt, RETRY_MAX_DELAY_MS);
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  private async processResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      let message = `Request failed with status ${response.status}`;
      try {
        const body = await response.json();
        const sanitized = sanitizeErrorData(body) as Record<string, unknown>;
        const error = sanitized.error as Record<string, unknown> | undefined;
        if (error && typeof error.detail === "string") message = error.detail;
        else if (typeof sanitized.message === "string") message = sanitized.message;
        else if (Array.isArray((sanitized as { errors?: unknown }).errors)) {
          const first = (sanitized as { errors: Array<Record<string, unknown>> }).errors[0];
          if (first && typeof first.detail === "string") message = first.detail;
          else if (first && typeof first.title === "string") message = first.title;
        }
      } catch {
        // Use status-based default.
      }
      throw createApiErrorFromResponse(response.status, message);
    }

    if (response.status === 204) return undefined as T;
    const contentLength = response.headers.get("content-length");
    if (contentLength === "0") return undefined as T;

    const text = await response.text();
    if (!text.trim()) return undefined as T;
    try {
      return JSON.parse(text) as T;
    } catch {
      const contentType = response.headers.get("content-type") ?? "";
      if (!contentType.includes("application/json")) return undefined as T;
      throw createApiErrorFromResponse(500, "Invalid JSON response from server");
    }
  }

  async get<T>(url: string, options?: ApiRequestOptions): Promise<T> {
    const response = await this.fetchWithRetry(
      `${this.baseUrl}${url}`,
      { method: "GET", headers: this.createHeaders(null, options) },
      options
    );
    return this.processResponse<T>(response);
  }

  async post<T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> {
    const hasBody = data !== undefined && data !== null;
    const response = await this.fetchWithRetry(
      `${this.baseUrl}${url}`,
      {
        method: "POST",
        headers: this.createHeaders(hasBody ? "application/json" : null, options),
        ...(hasBody ? { body: JSON.stringify(data) } : {}),
      },
      { ...options, maxRetries: options?.maxRetries ?? 0 }
    );
    return this.processResponse<T>(response);
  }

  async put<T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> {
    const response = await this.fetchWithRetry(
      `${this.baseUrl}${url}`,
      {
        method: "PUT",
        headers: this.createHeaders("application/json", options),
        body: data !== undefined ? JSON.stringify(data) : null,
      },
      { ...options, maxRetries: options?.maxRetries ?? 0 }
    );
    return this.processResponse<T>(response);
  }

  async patch<T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> {
    const response = await this.fetchWithRetry(
      `${this.baseUrl}${url}`,
      {
        method: "PATCH",
        headers: this.createHeaders("application/json", options),
        body: data !== undefined ? JSON.stringify(data) : null,
      },
      { ...options, maxRetries: options?.maxRetries ?? 0 }
    );
    return this.processResponse<T>(response);
  }

  async delete<T>(url: string, options?: ApiRequestOptions): Promise<T> {
    const response = await this.fetchWithRetry(
      `${this.baseUrl}${url}`,
      { method: "DELETE", headers: this.createHeaders(null, options) },
      { ...options, maxRetries: options?.maxRetries ?? 0 }
    );
    return this.processResponse<T>(response);
  }

  async upload<T>(url: string, file: File | FormData, options?: ApiRequestOptions): Promise<T> {
    // Browser sets the multipart boundary automatically when Content-Type is omitted.
    const headers = this.createHeaders(null, options);
    headers.delete("Content-Type");

    const body = file instanceof FormData ? file : (() => {
      const fd = new FormData();
      fd.append("file", file);
      return fd;
    })();

    const response = await this.fetchWithRetry(
      `${this.baseUrl}${url}`,
      { method: "POST", headers, body },
      { ...options, maxRetries: options?.maxRetries ?? 0 }
    );
    return this.processResponse<T>(response);
  }

  async download(url: string, options?: ApiRequestOptions): Promise<Blob> {
    const headers = this.createHeaders(null, options);
    headers.set("Accept", "*/*");

    const response = await this.fetchWithRetry(
      `${this.baseUrl}${url}`,
      { method: "GET", headers },
      options
    );
    if (!response.ok) throw createApiErrorFromResponse(response.status, `Download failed: ${response.status}`);
    return response.blob();
  }
}

/** Singleton instance. Exported for direct use by tests or advanced callers. */
export const apiClient = new ApiClient();

/**
 * Convenience wrapper. Most services import `api` and call `api.getList` /
 * `api.getOne` / `api.post` etc.
 */
export const api = {
  get: <T>(url: string, options?: ApiRequestOptions): Promise<T> => apiClient.get<T>(url, options),

  getList: <T>(url: string, options?: ApiRequestOptions): Promise<T[]> =>
    apiClient.get<ApiListResponse<T>>(url, options).then(r => r.data),

  getOne: <T>(url: string, options?: ApiRequestOptions): Promise<T> =>
    apiClient.get<ApiSingleResponse<T>>(url, options).then(r => r.data),

  post: <T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> =>
    apiClient.post<T>(url, data, options),

  put: <T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> =>
    apiClient.put<T>(url, data, options),

  patch: <T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> =>
    apiClient.patch<T>(url, data, options),

  delete: <T = void>(url: string, options?: ApiRequestOptions): Promise<T> =>
    apiClient.delete<T>(url, options),

  upload: <T>(url: string, file: File | FormData, options?: ApiRequestOptions): Promise<T> =>
    apiClient.upload<T>(url, file, options),

  download: (url: string, options?: ApiRequestOptions): Promise<Blob> =>
    apiClient.download(url, options),

  setTenant: (tenant: Tenant | null): void => apiClient.setTenant(tenant),
  getTenant: (): Tenant | null => apiClient.getTenant(),

  // Back-compat no-ops. Callers (notably base.service.ts#clearServiceCache
  // and the Next.js-era examples) can be simplified once those are removed.
  clearCache: (): void => {},
  clearCacheByPattern: (_pattern: string): void => {},
  getCacheStats: () => ({ size: 0, entries: [] as Array<{ key: string }> }),
};
