export interface PortalAttributes {
  name: string;
  target: string;
  type: number;
  targetMapId: number;
  scriptName: string;
  x: number;
  y: number;
}

export interface PortalData {
  id: string;
  type: string;
  attributes: PortalAttributes;
}
