import type { NameValidityResponse } from "@/services/api/factory.service";

export type RowStatus = "pending" | "applying" | "success" | "failed";

export interface Row {
  presetId: string;
  presetName: string;
  name: string;
  validity: NameValidityResponse | null;
  applyStatus: RowStatus;
  error?: string;
}

export interface WizardState {
  step: 1 | 2 | 3 | 4;
  account: { name: string; password: string };
  worldId: number;
  tagFilter: string[];
  rows: Record<string, Row>;
  accountId?: number;
  error?: string;
}

export type WizardAction =
  | { type: "SET_ACCOUNT"; account: { name: string; password: string } }
  | { type: "SET_WORLD"; worldId: number }
  | { type: "SET_TAG_FILTER"; tags: string[] }
  | { type: "TOGGLE_PRESET"; presetId: string; presetName: string }
  | { type: "SET_NAME"; presetId: string; name: string }
  | { type: "SET_VALIDITY"; presetId: string; validity: NameValidityResponse }
  | { type: "SET_ROW_STATUS"; presetId: string; status: RowStatus; error?: string }
  | { type: "ACCOUNT_CREATED"; accountId: number }
  | { type: "GOTO"; step: 1 | 2 | 3 | 4 }
  | { type: "SET_ERROR"; error: string }
  | { type: "RESET" };

export const initialState: WizardState = {
  step: 1,
  account: { name: "", password: "" },
  worldId: 0,
  tagFilter: [],
  rows: {},
};

export function wizardReducer(state: WizardState, action: WizardAction): WizardState {
  switch (action.type) {
    case "SET_ACCOUNT":
      return { ...state, account: action.account };
    case "SET_WORLD":
      return { ...state, worldId: action.worldId };
    case "SET_TAG_FILTER":
      return { ...state, tagFilter: action.tags };
    case "TOGGLE_PRESET": {
      if (state.rows[action.presetId]) {
        const next = { ...state.rows };
        delete next[action.presetId];
        return { ...state, rows: next };
      }
      return {
        ...state,
        rows: {
          ...state.rows,
          [action.presetId]: {
            presetId: action.presetId,
            presetName: action.presetName,
            name: "",
            validity: null,
            applyStatus: "pending",
          },
        },
      };
    }
    case "SET_NAME": {
      const row = state.rows[action.presetId];
      if (!row) return state;
      return {
        ...state,
        rows: {
          ...state.rows,
          [action.presetId]: { ...row, name: action.name, validity: null },
        },
      };
    }
    case "SET_VALIDITY": {
      const row = state.rows[action.presetId];
      if (!row) return state;
      return {
        ...state,
        rows: {
          ...state.rows,
          [action.presetId]: { ...row, validity: action.validity },
        },
      };
    }
    case "SET_ROW_STATUS": {
      const row = state.rows[action.presetId];
      if (!row) return state;
      return {
        ...state,
        rows: {
          ...state.rows,
          [action.presetId]: {
            ...row,
            applyStatus: action.status,
            ...(action.error !== undefined ? { error: action.error } : {}),
          },
        },
      };
    }
    case "ACCOUNT_CREATED":
      return { ...state, accountId: action.accountId };
    case "GOTO":
      return { ...state, step: action.step };
    case "SET_ERROR":
      return { ...state, error: action.error };
    case "RESET":
      return initialState;
    default:
      return state;
  }
}
