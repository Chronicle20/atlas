/**
 * Mirrors the atlas-events REST surface (event/definition/rest.go,
 * event/occurrence/rest.go, event/transition/rest.go — task-231 Tasks 13/16).
 */

/** Mirrors event/definition.RestModel (event/definition/rest.go). */
export interface EventDefinitionAttributes {
  type: string;
  name: string;
  enabled: boolean;
  /** Opaque per-event-type JSON — NOT `config`. */
  configuration: unknown;
  singleOccurrence: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface EventDefinition {
  id: string;
  type: string;
  attributes: EventDefinitionAttributes;
}

/**
 * GET /events/definitions query narrowings. `enabled` alone is a backend 400
 * (errFilterEnabledRequiresType) — it is only valid paired with `type`, so
 * eventsService only emits `filter[enabled]` when `type` is also set.
 */
export interface EventDefinitionFilters {
  type?: string;
  enabled?: boolean;
}

/** Mirrors event/transition.RestModel (event/transition/rest.go), flattened onto EventOccurrence.transitions. */
export interface EventOccurrenceTransition {
  id: string;
  occurrenceId: string;
  fromStage: string;
  toStage: string;
  occurredAt: string;
  triggerType: string;
  triggerReference?: string;
}

/** Mirrors event/occurrence.RestModel (event/occurrence/rest.go). */
export interface EventOccurrenceAttributes {
  type: string;
  state: string;
  stage: string;
  /** Opaque per-occurrence JSON. */
  context: unknown;
  startedAt: string;
  nextTransitionAt?: string;
  completedAt?: string;
  completionReason?: string;
}

/**
 * `transitions` is the `event-occurrence-transitions` `included` relationship,
 * flattened onto the occurrence. It is populated only by
 * eventsService.getOccurrence (the single-resource GET) — getOccurrences
 * (list) results carry an empty array since the backend only side-loads
 * transitions on the detail route.
 */
export interface EventOccurrence {
  id: string;
  type: string;
  attributes: EventOccurrenceAttributes;
  transitions: EventOccurrenceTransition[];
}

/** GET /events/occurrences query narrowings (event/occurrence/resource.go#parseListFilters). */
export interface EventOccurrenceFilters {
  definitionId?: string;
  type?: string;
  state?: string;
  worldId?: number;
  channelId?: number;
  mapId?: number;
  voyageId?: string;
  startedAt?: { from?: string; to?: string };
}
