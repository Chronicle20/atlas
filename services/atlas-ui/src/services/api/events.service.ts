/**
 * atlas-events admin surface: definitions (config CRUD-lite — enable/disable
 * only) and occurrences (read-only, including their transition history).
 *
 * Backed by atlas-events (task-231 Tasks 13/16, event/definition/resource.go,
 * event/occurrence/resource.go):
 *   GET   /api/events/definitions
 *   GET   /api/events/definitions/{definitionId}
 *   PATCH /api/events/definitions/{definitionId}
 *   GET   /api/events/occurrences
 *   GET   /api/events/occurrences/{occurrenceId}
 *
 * PATCH definition bodies must carry exactly one attribute, `enabled` — the
 * handler 400s on anything else (definition.PatchRestModel).
 */

import { api } from "@/lib/api/client";
import { fetchPaged, type PagedResult } from "@/services/api/pagination";
import type { ApiSingleResponse, JsonApiResource } from "@/types/api/responses";
import type {
  EventDefinition,
  EventDefinitionFilters,
  EventOccurrence,
  EventOccurrenceFilters,
  EventOccurrenceTransition,
} from "@/types/models/events";

const DEFINITIONS_PATH = "/api/events/definitions";
const OCCURRENCES_PATH = "/api/events/occurrences";

export const EVENT_DEFINITION_TYPE = "event-definitions";
export const EVENT_OCCURRENCE_TYPE = "event-occurrences";
export const EVENT_OCCURRENCE_TRANSITION_TYPE = "event-occurrence-transitions";

/** The raw list-endpoint shape — `transitions` is only ever side-loaded on the detail route. */
type EventOccurrenceResource = Omit<EventOccurrence, "transitions">;

function definitionQueryString(filters?: EventDefinitionFilters): string {
  const params = new URLSearchParams();
  if (filters?.type) params.append("filter[type]", filters.type);
  // filter[enabled] alone is a 400 (errFilterEnabledRequiresType) — only send it paired with type.
  if (filters?.type && filters.enabled !== undefined) {
    params.append("filter[enabled]", String(filters.enabled));
  }
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

function occurrenceQueryString(filters?: EventOccurrenceFilters): string {
  const params = new URLSearchParams();
  if (filters?.definitionId)
    params.append("filter[definitionId]", filters.definitionId);
  if (filters?.type) params.append("filter[type]", filters.type);
  if (filters?.state) params.append("filter[state]", filters.state);
  if (filters?.worldId !== undefined)
    params.append("filter[worldId]", String(filters.worldId));
  if (filters?.channelId !== undefined)
    params.append("filter[channelId]", String(filters.channelId));
  if (filters?.mapId !== undefined)
    params.append("filter[mapId]", String(filters.mapId));
  if (filters?.voyageId) params.append("filter[voyageId]", filters.voyageId);
  if (filters?.startedAt?.from)
    params.append("filter[startedAt][from]", filters.startedAt.from);
  if (filters?.startedAt?.to)
    params.append("filter[startedAt][to]", filters.startedAt.to);
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

function mapTransition(resource: JsonApiResource): EventOccurrenceTransition {
  const attrs = (resource.attributes ?? {}) as Record<string, unknown>;
  const triggerReference = attrs.triggerReference as string | undefined;
  return {
    id: resource.id,
    occurrenceId: attrs.occurrenceId as string,
    fromStage: attrs.fromStage as string,
    toStage: attrs.toStage as string,
    occurredAt: attrs.occurredAt as string,
    triggerType: attrs.triggerType as string,
    ...(triggerReference !== undefined ? { triggerReference } : {}),
  };
}

export const eventsService = {
  /** GET /events/definitions — one page (or the whole list when `page` is omitted). */
  async getDefinitions(
    filters?: EventDefinitionFilters,
    page?: { number: number; size: number },
  ): Promise<PagedResult<EventDefinition>> {
    const url = `${DEFINITIONS_PATH}${definitionQueryString(filters)}`;
    if (page) return fetchPaged<EventDefinition>(url, page);
    return { data: await api.getList<EventDefinition>(url), meta: null };
  },

  /** GET /events/definitions/{id} */
  async getDefinition(id: string): Promise<EventDefinition> {
    return api.getOne<EventDefinition>(`${DEFINITIONS_PATH}/${id}`);
  },

  /** PATCH /events/definitions/{id} — body carries exactly `{ enabled }`. */
  async setDefinitionEnabled(
    id: string,
    enabled: boolean,
  ): Promise<EventDefinition> {
    const response = await api.patch<ApiSingleResponse<EventDefinition>>(
      `${DEFINITIONS_PATH}/${id}`,
      { data: { type: EVENT_DEFINITION_TYPE, id, attributes: { enabled } } },
    );
    return response.data;
  },

  /** GET /events/occurrences — one page (or the whole list when `page` is omitted). */
  async getOccurrences(
    filters?: EventOccurrenceFilters,
    page?: { number: number; size: number },
  ): Promise<PagedResult<EventOccurrence>> {
    const url = `${OCCURRENCES_PATH}${occurrenceQueryString(filters)}`;
    const paged = page
      ? await fetchPaged<EventOccurrenceResource>(url, page)
      : { data: await api.getList<EventOccurrenceResource>(url), meta: null };
    return {
      data: paged.data.map((resource) => ({ ...resource, transitions: [] })),
      meta: paged.meta,
    };
  },

  /**
   * GET /events/occurrences/{id}. Uses `api.get` over the whole document
   * (not `api.getOne`, which discards `included`) so the side-loaded
   * `event-occurrence-transitions` can be flattened onto the result.
   */
  async getOccurrence(id: string): Promise<EventOccurrence> {
    const doc = await api.get<{
      data: EventOccurrenceResource;
      included?: JsonApiResource[];
    }>(`${OCCURRENCES_PATH}/${id}`);
    const transitions = (doc.included ?? [])
      .filter((resource) => resource.type === EVENT_OCCURRENCE_TRANSITION_TYPE)
      .map(mapTransition);
    return { ...doc.data, transitions };
  },
};
