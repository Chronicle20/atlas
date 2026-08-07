import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import { fetchAll } from "@/services/api/pagination";
import type {
  InstanceRoute,
  InstanceStatus,
  ScheduledRoute,
  ScheduledRouteDetail,
  TripSchedule,
  TripScheduleAttributes,
  Vessel,
} from "@/types/models/transport";

const SCHEDULED_PATH = "/api/transports/routes";
const INSTANCE_PATH = "/api/transports/instance-routes";

/** JSON:API compound document for a single route plus its included trips. */
interface ScheduledRouteDocument {
  data: ScheduledRoute;
  included?: Array<{
    id: string;
    type: string;
    attributes: TripScheduleAttributes;
  }>;
}

/**
 * Read-only adapters for the transports surface.
 *
 * The list read deliberately omits `include=schedule`: the backend attaches a
 * full day of trip rows (~96 per route) only on request, and the board renders
 * entirely from route attributes.
 */
export const transportsService = {
  async getScheduledRoutes(
    options?: ServiceOptions,
  ): Promise<ScheduledRoute[]> {
    return fetchAll<ScheduledRoute>(SCHEDULED_PATH, undefined, options);
  },

  /**
   * One route plus its trip rows. Uses the raw document because the trips
   * arrive in `included`, which `api.getOne`'s data-only projection drops.
   */
  async getScheduledRoute(
    routeId: string,
    options?: ServiceOptions,
  ): Promise<ScheduledRouteDetail> {
    const doc = await api.get<ScheduledRouteDocument>(
      `${SCHEDULED_PATH}/${routeId}?include=schedule`,
      options,
    );
    const schedule: TripSchedule[] = (doc.included ?? [])
      .filter((resource) => resource.type === "trip-schedule")
      .map((resource) => ({
        id: resource.id,
        attributes: resource.attributes,
      }));
    return { route: doc.data, schedule };
  },

  /**
   * One route's attributes without its trip rows. The breadcrumb only needs
   * the name, and `include=schedule` would drag ~96 trip resources along
   * with it.
   */
  async getScheduledRouteById(
    routeId: string,
    options?: ServiceOptions,
  ): Promise<ScheduledRoute> {
    return api.getOne<ScheduledRoute>(`${SCHEDULED_PATH}/${routeId}`, options);
  },

  async getInstanceRoutes(options?: ServiceOptions): Promise<InstanceRoute[]> {
    return fetchAll<InstanceRoute>(INSTANCE_PATH, undefined, options);
  },

  async getInstanceStatuses(
    routeId: string,
    options?: ServiceOptions,
  ): Promise<InstanceStatus[]> {
    return fetchAll<InstanceStatus>(
      `${INSTANCE_PATH}/${routeId}/status`,
      undefined,
      options,
    );
  },

  /**
   * Vessels are tenant configuration, not runtime state — atlas-transports
   * hands them to the scheduler and never stores them, so there is no vessel
   * registry to serve from. Same pattern the UI already uses for handlers,
   * writers and MTS config.
   */
  async getVessels(
    tenantId: string,
    options?: ServiceOptions,
  ): Promise<Vessel[]> {
    return fetchAll<Vessel>(
      `/api/tenants/${tenantId}/configurations/vessels`,
      undefined,
      options,
    );
  },
};
