export interface ReactorAttributes {
  name: string;
  stateInfo: Record<string, ReactorState[]>;
  timeoutInfo: Record<string, number>;
  tl: { x: number; y: number };
  br: { x: number; y: number };
}

export interface ReactorState {
  type: number;
  reactorItem?: { itemId: number; quantity: number } | null;
  activeSkills: number[];
  nextState: number;
}

export interface ReactorData {
  id: string;
  type: string;
  attributes: ReactorAttributes;
}
