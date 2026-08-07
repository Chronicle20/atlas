/**
 * JSON:API shapes for the atlas-transports surface.
 *
 * Durations are always the server's unit-explicit `…Seconds` integers. The
 * legacy nanosecond-valued fields (`cycleInterval` on routes, `boardingWindow`
 * and `travelDuration` on instance routes) are deliberately NOT declared here,
 * so nothing can read them by accident.
 */

export type RouteState =
  | "out_of_service"
  | "in_transit"
  | "locked_entry"
  | "open_entry"
  | "awaiting_return";

export type InstanceState = "boarding" | "in_transit";

export interface ScheduledRouteAttributes {
  name: string;
  startMapId: number;
  stagingMapId: number;
  enRouteMapIds: number[];
  destinationMapId: number;
  observationMapId: number;
  state: RouteState;
  boardingWindowSeconds: number;
  preDepartureSeconds: number;
  travelDurationSeconds: number;
  cycleIntervalSeconds: number;
  /** Absolute RFC3339 instant of the next state change; "" when out of service. */
  nextTransitionAt: string;
  /** The state reached at nextTransitionAt; "" when out of service. */
  nextState: RouteState | "";
}

export interface ScheduledRoute {
  id: string;
  attributes: ScheduledRouteAttributes;
}

/**
 * Trip boundaries carry the date of the day the schedule was computed; only
 * their time-of-day component is meaningful. Render them through
 * `formatTimeOfDay` and never as a date.
 */
export interface TripScheduleAttributes {
  boardingOpen: string;
  boardingClosed: string;
  departure: string;
  arrival: string;
}

export interface TripSchedule {
  id: string;
  attributes: TripScheduleAttributes;
}

export interface ScheduledRouteDetail {
  route: ScheduledRoute;
  schedule: TripSchedule[];
}

export interface InstanceRouteAttributes {
  name: string;
  startMapId: number;
  transitMapIds: number[];
  destinationMapId: number;
  capacity: number;
  boardingWindowSeconds: number;
  travelDurationSeconds: number;
}

export interface InstanceRoute {
  id: string;
  attributes: InstanceRouteAttributes;
}

export interface InstanceStatusAttributes {
  routeId: string;
  state: InstanceState;
  characters: number;
  boardingUntil: string;
  arrivalAt: string;
  createdAt: string;
}

export interface InstanceStatus {
  id: string;
  attributes: InstanceStatusAttributes;
}

/**
 * Vessels are pure tenant configuration served by atlas-tenants. The resource
 * id is the config slug; `routeAID`/`routeBID` are route **names**, which is
 * what the backend scheduler matches on.
 */
export interface VesselAttributes {
  uuid: string;
  name: string;
  routeAID: string;
  routeBID: string;
  /** Turnaround delay in seconds. */
  turnaroundDelay: number;
}

export interface Vessel {
  id: string;
  attributes: VesselAttributes;
}
