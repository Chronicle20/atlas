import { api } from "@/lib/api/client";

const BASE_PATH = "/api/characters";

export type PendingChangeType = "NAME_CHANGE" | "WORLD_TRANSFER";
export type PendingChangeStatus =
  "PENDING" | "APPLIED" | "CANCELLED" | "REJECTED" | "EXPIRED";

export interface PendingChange {
  id: string;
  characterId: number;
  type: PendingChangeType;
  status: PendingChangeStatus;
  requestedName?: string;
  destinationWorldId: number;
  sourceWorldId: number;
  reason?: string;
  createdAt: string;
  expiresAt: string;
  resolvedAt?: string;
}

interface PendingChangeAttributes {
  characterId: number;
  type: PendingChangeType;
  status: PendingChangeStatus;
  requestedName?: string;
  destinationWorldId: number;
  sourceWorldId: number;
  reason?: string;
  createdAt: string;
  expiresAt: string;
  resolvedAt?: string;
}

interface PendingChangeResource {
  id: string;
  type: "pending-changes";
  attributes: PendingChangeAttributes;
}

function flatten(r: PendingChangeResource): PendingChange {
  return {
    id: r.id,
    ...r.attributes,
  };
}

export const pendingChangesService = {
  async getByCharacterId(characterId: string): Promise<PendingChange[]> {
    const resources = await api.getList<PendingChangeResource>(
      `${BASE_PATH}/${characterId}/pending-changes`,
    );
    return resources.map(flatten);
  },

  async cancel(characterId: string, id: string): Promise<void> {
    await api.delete<void>(`${BASE_PATH}/${characterId}/pending-changes/${id}`);
  },
};
