import type {Tenant} from "@/types/models/tenant";
import {tenantHeaders} from "@/lib/headers";

export interface Inventory {
  type: string;
  id: string;
  attributes: {
    characterId: number;
  };
  relationships: {
    compartments: {
      links: {
        related: string;
        self: string;
      };
      data: Array<{
        type: string;
        id: string;
      }>;
    };
  };
}

export interface Compartment {
  type: string;
  id: string;
  attributes: {
    type: number;
    capacity: number;
  };
  relationships: {
    assets: {
      links: {
        related: string;
        self: string;
      };
      data: Array<{
        type: string;
        id: string;
      }>;
    };
  };
}

export interface Asset {
  type: string;
  id: string;
  attributes: {
    slot: number;
    templateId: number;
    expiration: string;
    referenceId: number;
    referenceType: string;
    referenceData: unknown;
  };
}

export interface InventoryResponse {
  data: Inventory;
  included: Array<Compartment | Asset>;
}

// Helper function to get compartment type name
export function getCompartmentTypeName(type: number): string {
  switch (type) {
    case 1:
      return "Equipables";
    case 2:
      return "Consumables";
    case 3:
      return "Setup";
    case 4:
      return "Etc";
    case 5:
      return "Cash";
    default:
      return `Type ${type}`;
  }
}

// Function to fetch inventory data
export async function fetchInventory(tenant: Tenant, characterId: string): Promise<InventoryResponse> {
  const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
  const response = await fetch(`${rootUrl}/api/characters/${characterId}/inventory`, {
    method: "GET",
    headers: tenantHeaders(tenant),
  });

  if (!response.ok) {
    throw new Error("Failed to fetch inventory.");
  }

  return await response.json();
}

// Helper function to get assets for a compartment
export function getAssetsForCompartment(compartment: Compartment, included: Array<Compartment | Asset>): Asset[] {
  return compartment.relationships.assets.data
      .map(assetRef => included.find(item => item.type === assetRef.type && item.id === assetRef.id))
      .filter((asset): asset is Asset => {
        return (
            asset !== undefined &&
            asset.type === 'assets' &&
            (asset as Asset).attributes.slot >= 0
        );
      })
      .sort((a, b) => a.attributes.slot - b.attributes.slot);
}

// Function to delete an asset
export async function deleteAsset(tenant: Tenant, characterId: string, compartmentId: string, assetId: string): Promise<void> {
  const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
  const response = await fetch(`${rootUrl}/api/characters/${characterId}/inventory/compartments/${compartmentId}/assets/${assetId}`, {
    method: "DELETE",
    headers: tenantHeaders(tenant),
  });

  if (!response.ok) {
    throw new Error("Failed to delete asset.");
  }
}
