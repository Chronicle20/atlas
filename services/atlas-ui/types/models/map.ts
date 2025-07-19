// Map domain model types
// Re-exported from lib/maps.tsx to centralize type definitions

export interface Map {
    id: string;
    attributes: MapAttributes;
}

export interface MapAttributes {
    name: string;
    streetName: string;
}