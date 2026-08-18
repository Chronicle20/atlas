import { api } from "@/lib/api/client";
import {
  tenantHeaders,
  canonicalHeaders,
  type CanonicalSelection,
} from "@/lib/headers";
import type { Tenant } from "@/types/models/tenant";

export interface SeedResult {
  deletedCount?: number;
  createdCount?: number;
  failedCount?: number;
  errors?: string[];
}

export interface WzInputStatus {
  fileCount: number;
  totalBytes: number;
  updatedAt: string | null;
}

export interface DataStatus {
  documentCount: number;
  updatedAt: string | null;
  baselineRestoredAt: string | null;
  baselineSha256: string | null;
}

export interface DropsSeedStatus {
  monsterDropCount: number;
  continentDropCount: number;
  reactorDropCount: number;
  updatedAt: string | null;
}

export interface GachaponsSeedStatus {
  gachaponCount: number;
  itemCount: number;
  globalItemCount: number;
  updatedAt: string | null;
}

export interface NpcConversationsSeedStatus {
  conversationCount: number;
  updatedAt: string | null;
}

export interface QuestConversationsSeedStatus {
  conversationCount: number;
  updatedAt: string | null;
}

export interface ItemConversationsSeedStatus {
  conversationCount: number;
  updatedAt: string | null;
}

export interface NpcShopsSeedStatus {
  shopCount: number;
  commodityCount: number;
  updatedAt: string | null;
}

export interface PortalScriptsSeedStatus {
  scriptCount: number;
  updatedAt: string | null;
}

export interface ReactorScriptsSeedStatus {
  scriptCount: number;
  updatedAt: string | null;
}

export interface MapActionScriptsSeedStatus {
  scriptCount: number;
  updatedAt: string | null;
}

export interface TransportRoutesSeedStatus {
  routeCount: number;
  updatedAt: string | null;
}

export interface TransportVesselsSeedStatus {
  vesselCount: number;
  updatedAt: string | null;
}

export interface InstanceRoutesSeedStatus {
  routeCount: number;
  updatedAt: string | null;
}

export interface EventDefinitionsSeedStatus {
  definitionCount: number;
  updatedAt: string | null;
}

export type IngestPhase =
  "none" | "running" | "succeeded" | "failed" | "stuck" | "unknown";

export type IngestWorkerState =
  "pending" | "running" | "succeeded" | "failed" | "skipped";

export interface IngestRunWorker {
  name: string;
  state: IngestWorkerState;
  startedAt: string | null;
  finishedAt: string | null;
  error: string | null;
}

export interface IngestRun {
  runId: string;
  jobName: string;
  scope: string;
  region: string;
  version: string;
  tenant?: string;
  phase: IngestPhase;
  startedAt: string | null;
  finishedAt: string | null;
  reason: string | null;
  workersTotal: number;
  workersComplete: number;
  workers: IngestRunWorker[];
}

// Shape returned by libs/atlas-seeder's GET /<prefix>/seed/status handler.
// The handler emits a plain JSON object (not a JSON:API envelope), so we
// read it directly. Per-service status objects (DropsSeedStatus, etc.)
// are projections of this generic shape into the field names the UI
// already renders.
interface SeederSubdomainStatus {
  count: number;
  updatedAt: string | null;
}

interface SeedStatus {
  groupName: string;
  subdomains: Record<string, SeederSubdomainStatus>;
  updatedAt: string | null;
  catalogRevision: string;
  tenantSeededRevision: string | null;
  tenantSeededAt: string | null;
}

interface JsonApiEnvelope<A> {
  data: {
    type: string;
    id: string;
    attributes: A;
  };
}

async function fetchJsonApi<A>(url: string, headers: Headers): Promise<A> {
  headers.set("Accept", "application/vnd.api+json");
  const response = await fetch(url, { method: "GET", headers });
  if (!response.ok) {
    throw new Error(
      `GET ${url} failed: ${response.status} ${response.statusText}`,
    );
  }
  const body = (await response.json()) as JsonApiEnvelope<A>;
  return body.data.attributes;
}

async function patchWzZip(
  url: string,
  headers: Headers,
  file: File,
): Promise<void> {
  const formData = new FormData();
  formData.append("zip_file", file);

  const response = await fetch(url, {
    method: "PATCH",
    headers,
    body: formData,
  });

  if (!response.ok) {
    let message = `Upload failed: ${response.status} ${response.statusText}`;
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) {
        message = body.error;
      }
    } catch {
      // non-JSON error body; fall back to status text
    }
    const err = new Error(message) as Error & { status?: number };
    err.status = response.status;
    throw err;
  }
}

async function postProcess(url: string, headers: Headers): Promise<void> {
  const response = await fetch(url, { method: "POST", headers });
  if (!response.ok) {
    throw new Error(
      `Data processing failed: ${response.status} ${response.statusText}`,
    );
  }
}

async function fetchSeedStatus(
  url: string,
  tenant: Tenant,
): Promise<SeedStatus> {
  const headers = tenantHeaders(tenant);
  const response = await fetch(url, { method: "GET", headers });
  if (!response.ok) {
    throw new Error(
      `GET ${url} failed: ${response.status} ${response.statusText}`,
    );
  }
  return (await response.json()) as SeedStatus;
}

function subdomainCount(s: SeedStatus, key: string): number {
  // Optional chain on `subdomains` as well as the entry — guards
  // against a malformed response where the map itself is absent.
  return s.subdomains?.[key]?.count ?? 0;
}

class SeedService {
  async seedDrops(): Promise<void> {
    await api.post("/api/drops/seed", {});
  }

  async seedGachapons(): Promise<void> {
    await api.post("/api/gachapons/seed", {});
  }

  async seedNpcConversations(): Promise<SeedResult> {
    return api.post<SeedResult>("/api/npcs/conversations/seed", {});
  }

  async seedQuestConversations(): Promise<SeedResult> {
    return api.post<SeedResult>("/api/quests/conversations/seed", {});
  }

  async seedItemConversations(): Promise<SeedResult> {
    return api.post<SeedResult>("/api/items/conversations/seed", {});
  }

  async seedNpcShops(): Promise<SeedResult> {
    return api.post<SeedResult>("/api/shops/seed", {});
  }

  async seedPortalScripts(): Promise<SeedResult> {
    return api.post<SeedResult>("/api/portals/scripts/seed", {});
  }

  async seedReactorScripts(): Promise<SeedResult> {
    return api.post<SeedResult>("/api/reactors/actions/seed", {});
  }

  async seedMapActionScripts(): Promise<SeedResult> {
    return api.post<SeedResult>("/api/maps/actions/seed", {});
  }

  // The three transport-configuration seeds return 202 with no body and
  // seed in the background (libs/atlas-seeder's postSeed contract), so
  // these resolve to void and the Setup page's 5s status poll is what
  // surfaces the result.
  async seedRoutes(): Promise<void> {
    await api.post("/api/tenants/configurations/routes/seed", {});
  }

  async seedVessels(): Promise<void> {
    await api.post("/api/tenants/configurations/vessels/seed", {});
  }

  async seedInstanceRoutes(): Promise<void> {
    await api.post("/api/tenants/configurations/instance-routes/seed", {});
  }

  async seedEventDefinitions(): Promise<void> {
    await api.post("/api/events/definitions/seed", {});
  }

  async uploadWzFiles(tenant: Tenant, file: File): Promise<void> {
    return patchWzZip("/api/data/wz?scope=tenant", tenantHeaders(tenant), file);
  }

  async runDataProcessing(tenant: Tenant): Promise<void> {
    return postProcess("/api/data/process?scope=tenant", tenantHeaders(tenant));
  }

  async getWzInputStatus(tenant: Tenant): Promise<WzInputStatus> {
    return fetchJsonApi<WzInputStatus>(
      "/api/data/wz?scope=tenant",
      tenantHeaders(tenant),
    );
  }

  async getDataStatus(tenant: Tenant): Promise<DataStatus> {
    return fetchJsonApi<DataStatus>(
      "/api/data/status?scope=tenant",
      tenantHeaders(tenant),
    );
  }

  async getIngestRun(tenant: Tenant): Promise<IngestRun> {
    return fetchJsonApi<IngestRun>(
      "/api/data/process?scope=tenant",
      tenantHeaders(tenant),
    );
  }

  // Canonical (deployment-wide) variants: no Tenant anywhere — headers are
  // synthesized from the explicit region/version selection. This is what lets
  // an operator publish canonical data for a version with no live tenant.
  async uploadCanonicalWzFiles(
    sel: CanonicalSelection,
    file: File,
  ): Promise<void> {
    return patchWzZip("/api/data/wz?scope=shared", canonicalHeaders(sel), file);
  }

  async runCanonicalDataProcessing(sel: CanonicalSelection): Promise<void> {
    return postProcess("/api/data/process?scope=shared", canonicalHeaders(sel));
  }

  async getCanonicalWzInputStatus(
    sel: CanonicalSelection,
  ): Promise<WzInputStatus> {
    return fetchJsonApi<WzInputStatus>(
      "/api/data/wz?scope=shared",
      canonicalHeaders(sel),
    );
  }

  async getCanonicalDataStatus(sel: CanonicalSelection): Promise<DataStatus> {
    return fetchJsonApi<DataStatus>(
      "/api/data/status?scope=shared",
      canonicalHeaders(sel),
    );
  }

  // canonicalHeaders already bakes in X-Atlas-Operator: 1, which the shared
  // scope requires — one construction path, no drift.
  async getCanonicalIngestRun(sel: CanonicalSelection): Promise<IngestRun> {
    return fetchJsonApi<IngestRun>(
      "/api/data/process?scope=shared",
      canonicalHeaders(sel),
    );
  }

  async getDropsSeedStatus(tenant: Tenant): Promise<DropsSeedStatus> {
    const s = await fetchSeedStatus("/api/drops/seed/status", tenant);
    return {
      monsterDropCount: subdomainCount(s, "monster-drop"),
      continentDropCount: subdomainCount(s, "continent-drop"),
      reactorDropCount: subdomainCount(s, "reactor-drop"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getGachaponsSeedStatus(tenant: Tenant): Promise<GachaponsSeedStatus> {
    const s = await fetchSeedStatus("/api/gachapons/seed/status", tenant);
    return {
      gachaponCount: subdomainCount(s, "gachapons"),
      itemCount: subdomainCount(s, "items"),
      globalItemCount: subdomainCount(s, "globalItems"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getNpcConversationsSeedStatus(
    tenant: Tenant,
  ): Promise<NpcConversationsSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/npcs/conversations/seed/status",
      tenant,
    );
    return {
      conversationCount: subdomainCount(s, "npc.conversation"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getQuestConversationsSeedStatus(
    tenant: Tenant,
  ): Promise<QuestConversationsSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/quests/conversations/seed/status",
      tenant,
    );
    return {
      conversationCount: subdomainCount(s, "quest.conversation"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getItemConversationsSeedStatus(
    tenant: Tenant,
  ): Promise<ItemConversationsSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/items/conversations/seed/status",
      tenant,
    );
    return {
      conversationCount: subdomainCount(s, "item.conversation"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getNpcShopsSeedStatus(tenant: Tenant): Promise<NpcShopsSeedStatus> {
    const s = await fetchSeedStatus("/api/shops/seed/status", tenant);
    // commodities arrives via SubdomainAuxiliary on ShopSubdomain — it
    // shares the response shape but is not its own primary subdomain.
    return {
      shopCount: subdomainCount(s, "npc-shops"),
      commodityCount: subdomainCount(s, "commodities"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getPortalScriptsSeedStatus(
    tenant: Tenant,
  ): Promise<PortalScriptsSeedStatus> {
    const s = await fetchSeedStatus("/api/portals/scripts/seed/status", tenant);
    return {
      scriptCount: subdomainCount(s, "portal-actions"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getReactorScriptsSeedStatus(
    tenant: Tenant,
  ): Promise<ReactorScriptsSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/reactors/actions/seed/status",
      tenant,
    );
    return {
      scriptCount: subdomainCount(s, "reactor-actions"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getMapActionScriptsSeedStatus(
    tenant: Tenant,
  ): Promise<MapActionScriptsSeedStatus> {
    const s = await fetchSeedStatus("/api/maps/actions/seed/status", tenant);
    // map-actions exposes two subdomains in the seeder lib
    // (onUserEnter + onFirstUserEnter); sum them for the single
    // scriptCount the UI displays.
    return {
      scriptCount:
        subdomainCount(s, "onUserEnter") +
        subdomainCount(s, "onFirstUserEnter"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getTransportRoutesSeedStatus(
    tenant: Tenant,
  ): Promise<TransportRoutesSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/tenants/configurations/routes/seed/status",
      tenant,
    );
    return {
      routeCount: subdomainCount(s, "routes"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getTransportVesselsSeedStatus(
    tenant: Tenant,
  ): Promise<TransportVesselsSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/tenants/configurations/vessels/seed/status",
      tenant,
    );
    return {
      vesselCount: subdomainCount(s, "vessels"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getInstanceRoutesSeedStatus(
    tenant: Tenant,
  ): Promise<InstanceRoutesSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/tenants/configurations/instance-routes/seed/status",
      tenant,
    );
    return {
      routeCount: subdomainCount(s, "instance-routes"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getEventDefinitionsSeedStatus(
    tenant: Tenant,
  ): Promise<EventDefinitionsSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/events/definitions/seed/status",
      tenant,
    );
    return {
      definitionCount: subdomainCount(s, "definition.event"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }
}

export const seedService = new SeedService();
