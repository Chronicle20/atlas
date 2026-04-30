import { tenantHeaders } from "@/lib/headers";
import type { Tenant } from "@/types/models/tenant";

const BASE_PATH = "/api/factory";

export interface CreateFromPresetPayload {
  presetId: string;
  accountId: number;
  worldId: number;
  name: string;
}

export interface CreateFromPresetResponse {
  transactionId: string;
}

export interface NameValidityResponse {
  valid: boolean;
  reason?: "regex" | "length" | "blocked" | "duplicate";
  detail?: string;
}

interface CreateFromPresetEnvelope {
  data: {
    type: string;
    id: string;
    attributes: {
      transactionId: string;
    };
  };
}

export const factoryService = {
  /**
   * POST /api/factory/characters/from-preset
   *
   * Sends a JSON:API-encoded body and receives a 202 Accepted with a JSON:API
   * envelope. The backend uses RegisterInputHandler[PresetCreateRestModel] which
   * requires the request body to be wrapped in the standard JSON:API format.
   *
   * Tenant headers are injected via a direct fetch call so they are applied even
   * when the caller passes a tenant object that differs from the singleton's current
   * tenant (matching the pattern used by seed.service.ts for tenant-aware POSTs).
   */
  async createFromPreset(
    tenant: Tenant,
    payload: CreateFromPresetPayload,
  ): Promise<CreateFromPresetResponse> {
    const headers = tenantHeaders(tenant);
    headers.set("Content-Type", "application/vnd.api+json");

    const requestBody = {
      data: {
        type: "preset-create",
        attributes: {
          presetId: payload.presetId,
          accountId: payload.accountId,
          worldId: payload.worldId,
          name: payload.name,
        },
      },
    };

    const response = await fetch(`${BASE_PATH}/characters/from-preset`, {
      method: "POST",
      headers,
      body: JSON.stringify(requestBody),
    });

    if (!response.ok) {
      let message = `createFromPreset failed with status ${response.status}`;
      try {
        const errBody = (await response.json()) as { error?: string; message?: string };
        if (errBody.error) message = errBody.error;
        else if (errBody.message) message = errBody.message;
      } catch {
        // non-JSON error body; keep the default message
      }
      const err = new Error(message) as Error & { status?: number };
      err.status = response.status;
      throw err;
    }

    const responseBody = (await response.json()) as CreateFromPresetEnvelope;
    return { transactionId: responseBody.data.attributes.transactionId };
  },
};
