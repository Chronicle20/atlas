import { useReducer, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DialogFooter } from "@/components/ui/dialog";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useNameValidity } from "@/lib/hooks/api/useNameValidity";
import { factoryService } from "@/services/api/factory.service";
import type { Tenant } from "@/types/models/tenant";
import type { NameValidityResponse } from "@/services/api/factory.service";
import type { Row } from "@/components/features/accounts/AdminBootstrapWizard.types";
import { createErrorFromUnknown } from "@/types/api/errors";

// ---------------------------------------------------------------------------
// Preset shape from tenant config
// ---------------------------------------------------------------------------
interface PresetItem {
  id: string;
  attributes: {
    name: string;
    tags?: string[];
  };
}

// ---------------------------------------------------------------------------
// Step A — World + tag filter + preset selection
// ---------------------------------------------------------------------------
interface StepAProps {
  worldId: number;
  tagFilter: string[];
  selectedPresetIds: Set<string>;
  allPresets: PresetItem[];
  onWorldChange: (worldId: number) => void;
  onTagFilterChange: (tags: string[]) => void;
  onTogglePreset: (presetId: string, presetName: string) => void;
  onClose?: (() => void) | undefined;
  onNext: () => void;
}

function StepA({
  worldId,
  tagFilter,
  selectedPresetIds,
  allPresets,
  onWorldChange,
  onTagFilterChange,
  onTogglePreset,
  onClose,
  onNext,
}: StepAProps) {
  const allTags = Array.from(
    new Set(allPresets.flatMap((p) => p.attributes.tags ?? [])),
  ).sort();

  const filteredPresets =
    tagFilter.length === 0
      ? allPresets
      : allPresets.filter((p) =>
          tagFilter.every((t) => (p.attributes.tags ?? []).includes(t)),
        );

  const toggleTag = (tag: string) => {
    if (tagFilter.includes(tag)) {
      onTagFilterChange(tagFilter.filter((t) => t !== tag));
    } else {
      onTagFilterChange([...tagFilter, tag]);
    }
  };

  const canNext = selectedPresetIds.size > 0;

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="flow-world">World ID</Label>
        <Input
          id="flow-world"
          type="number"
          min={0}
          max={255}
          value={worldId}
          onChange={(e) => onWorldChange(Number(e.target.value))}
          className="w-32"
        />
      </div>

      {allTags.length > 0 && (
        <div className="space-y-2">
          <Label>Filter by tag</Label>
          <div className="flex flex-wrap gap-2">
            {allTags.map((tag) => (
              <button
                key={tag}
                type="button"
                onClick={() => toggleTag(tag)}
                className={`px-2 py-1 text-xs rounded-full border transition-colors ${
                  tagFilter.includes(tag)
                    ? "bg-primary text-primary-foreground border-primary"
                    : "bg-background border-border hover:bg-accent"
                }`}
              >
                {tag}
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="space-y-2">
        <Label>
          Presets{" "}
          <span className="text-muted-foreground text-xs">
            ({filteredPresets.length} shown, {selectedPresetIds.size} selected)
          </span>
        </Label>
        <div className="border rounded-md max-h-56 overflow-y-auto divide-y">
          {filteredPresets.length === 0 && (
            <p className="p-4 text-sm text-muted-foreground">
              No presets match the selected tags.
            </p>
          )}
          {filteredPresets.map((p) => {
            const selected = selectedPresetIds.has(p.id);
            return (
              <label
                key={p.id}
                className={`flex items-center gap-3 px-3 py-2 cursor-pointer hover:bg-accent ${
                  selected ? "bg-accent/50" : ""
                }`}
              >
                <input
                  type="checkbox"
                  checked={selected}
                  onChange={() => onTogglePreset(p.id, p.attributes.name)}
                  className="h-4 w-4"
                />
                <span className="text-sm">{p.attributes.name}</span>
                {(p.attributes.tags ?? []).length > 0 && (
                  <span className="text-xs text-muted-foreground ml-auto">
                    {(p.attributes.tags ?? []).join(", ")}
                  </span>
                )}
              </label>
            );
          })}
        </div>
      </div>

      <DialogFooter>
        {onClose && (
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
        )}
        <Button onClick={onNext} disabled={!canNext}>
          Next
        </Button>
      </DialogFooter>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step B — Name overrides (one row per selected preset)
// ---------------------------------------------------------------------------

interface NameRowProps {
  tenant: Tenant;
  row: Row;
  isDuplicate: boolean;
  onChangeName: (name: string) => void;
  onValidityUpdate: (validity: NameValidityResponse) => void;
}

function NameRow({
  tenant,
  row,
  isDuplicate,
  onChangeName,
  onValidityUpdate,
}: NameRowProps) {
  const validityQuery = useNameValidity(tenant, row.name, 0, {
    enabled: row.name.length >= 3 && !isDuplicate,
  });

  useEffect(() => {
    if (validityQuery.data) {
      onValidityUpdate(validityQuery.data);
    }
  }, [validityQuery.data, onValidityUpdate]);

  const effectiveValidity: NameValidityResponse | null = isDuplicate
    ? { valid: false, reason: "duplicate", detail: "Duplicate within selection" }
    : row.validity;

  return (
    <tr className="border-b">
      <td className="py-2 px-3 text-sm text-muted-foreground">{row.presetName}</td>
      <td className="py-2 px-3">
        <Input
          value={row.name}
          onChange={(e) => onChangeName(e.target.value)}
          placeholder="3–12 characters"
          className="h-8 text-sm"
        />
      </td>
      <td className="py-2 px-3 text-xs min-w-[120px]">
        {row.name.length < 3 ? (
          <span className="text-muted-foreground">—</span>
        ) : validityQuery.isLoading && !isDuplicate ? (
          <span className="text-muted-foreground">Checking…</span>
        ) : effectiveValidity ? (
          effectiveValidity.valid ? (
            <span className="text-primary">Available</span>
          ) : (
            <span className="text-destructive">
              {effectiveValidity.detail ?? effectiveValidity.reason ?? "Invalid"}
            </span>
          )
        ) : null}
      </td>
    </tr>
  );
}

interface StepBProps {
  tenant: Tenant;
  rows: Record<string, Row>;
  onChangeName: (presetId: string, name: string) => void;
  onValidityUpdate: (presetId: string, validity: NameValidityResponse) => void;
  onBack: () => void;
  onNext: () => void;
}

function StepB({
  tenant,
  rows,
  onChangeName,
  onValidityUpdate,
  onBack,
  onNext,
}: StepBProps) {
  const rowList = Object.values(rows);

  const nameCounts: Record<string, number> = {};
  for (const row of rowList) {
    if (row.name.length >= 3) {
      nameCounts[row.name] = (nameCounts[row.name] ?? 0) + 1;
    }
  }
  const duplicateNames = new Set(
    Object.entries(nameCounts)
      .filter(([, count]) => count > 1)
      .map(([name]) => name),
  );

  const allValid = rowList.every((row) => {
    if (duplicateNames.has(row.name)) return false;
    if (row.name.length < 3) return false;
    return row.validity?.valid === true;
  });

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Enter a character name for each selected preset.
      </p>
      <div className="overflow-auto max-h-72">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th className="py-2 px-3 font-medium">Preset</th>
              <th className="py-2 px-3 font-medium">Character name</th>
              <th className="py-2 px-3 font-medium">Validity</th>
            </tr>
          </thead>
          <tbody>
            {rowList.map((row) => (
              <NameRow
                key={row.presetId}
                tenant={tenant}
                row={row}
                isDuplicate={duplicateNames.has(row.name)}
                onChangeName={(name) => onChangeName(row.presetId, name)}
                onValidityUpdate={(validity) => onValidityUpdate(row.presetId, validity)}
              />
            ))}
          </tbody>
        </table>
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={onBack}>
          Back
        </Button>
        <Button onClick={onNext} disabled={!allValid}>
          Apply
        </Button>
      </DialogFooter>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step C — Apply (sequential mutations + per-row status table)
// ---------------------------------------------------------------------------

const STATUS_LABEL: Record<string, string> = {
  pending: "Pending",
  applying: "Applying…",
  success: "Success",
  failed: "Failed",
};

const STATUS_CLASS: Record<string, string> = {
  pending: "text-muted-foreground",
  applying: "text-muted-foreground",
  success: "text-primary",
  failed: "text-destructive",
};

interface StepCProps {
  tenant: Tenant;
  rows: Record<string, Row>;
  accountId: number;
  error: string | undefined;
  onRetry: (presetId: string) => void;
  onDone: () => void;
}

function StepC({ rows, accountId, error, onRetry, onDone }: StepCProps) {
  const rowList = Object.values(rows);
  const allTerminal = rowList.every(
    (r) => r.applyStatus === "success" || r.applyStatus === "failed",
  );

  return (
    <div className="space-y-4">
      {accountId && !error && (
        <p className="text-sm text-muted-foreground">
          Applying presets to account <strong>{accountId}</strong>…
        </p>
      )}
      {error && <p className="text-sm text-destructive">{error}</p>}

      <div className="overflow-auto max-h-72">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th className="py-2 px-3 font-medium">Preset</th>
              <th className="py-2 px-3 font-medium">Character</th>
              <th className="py-2 px-3 font-medium">Status</th>
              <th className="py-2 px-3" />
            </tr>
          </thead>
          <tbody>
            {rowList.map((row) => (
              <tr key={row.presetId} className="border-b">
                <td className="py-2 px-3 text-muted-foreground">{row.presetName}</td>
                <td className="py-2 px-3">{row.name}</td>
                <td className={`py-2 px-3 ${STATUS_CLASS[row.applyStatus] ?? ""}`}>
                  {STATUS_LABEL[row.applyStatus] ?? row.applyStatus}
                  {row.applyStatus === "failed" && row.error && (
                    <span className="block text-xs">{row.error}</span>
                  )}
                </td>
                <td className="py-2 px-3">
                  {row.applyStatus === "failed" && (
                    <Button size="sm" variant="outline" onClick={() => onRetry(row.presetId)}>
                      Retry
                    </Button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {allTerminal && (
        <DialogFooter>
          <Button onClick={onDone}>Done</Button>
        </DialogFooter>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// PresetApplicationFlow — public API
// ---------------------------------------------------------------------------

export interface PresetApplicationFlowProps {
  tenant: Tenant;
  accountId: number;
  onComplete?: () => void;
  onClose?: () => void;
}

// Internal step numbering: 1=filter/select, 2=names, 3=apply
type FlowStep = 1 | 2 | 3;

interface FlowState {
  step: FlowStep;
  worldId: number;
  tagFilter: string[];
  rows: Record<string, Row>;
  error?: string;
}

type FlowAction =
  | { type: "SET_WORLD"; worldId: number }
  | { type: "SET_TAG_FILTER"; tags: string[] }
  | { type: "TOGGLE_PRESET"; presetId: string; presetName: string }
  | { type: "SET_NAME"; presetId: string; name: string }
  | { type: "SET_VALIDITY"; presetId: string; validity: NameValidityResponse }
  | { type: "SET_ROW_STATUS"; presetId: string; status: Row["applyStatus"]; error?: string }
  | { type: "GOTO"; step: FlowStep }
  | { type: "SET_ERROR"; error: string }
  | { type: "RESET" };

const flowInitialState: FlowState = {
  step: 1,
  worldId: 0,
  tagFilter: [],
  rows: {},
};

function flowReducer(state: FlowState, action: FlowAction): FlowState {
  // Delegate shared actions to the wizard reducer by mapping to its shape, then extract
  switch (action.type) {
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
    case "GOTO":
      return { ...state, step: action.step };
    case "SET_ERROR":
      return { ...state, error: action.error };
    case "RESET":
      return flowInitialState;
    default:
      return state;
  }
}

export function PresetApplicationFlow({
  tenant,
  accountId,
  onComplete,
  onClose,
}: PresetApplicationFlowProps) {
  const [state, dispatch] = useReducer(flowReducer, flowInitialState);

  const tenantConfigQuery = useTenantConfiguration(tenant.id);
  const allPresets: PresetItem[] = (
    tenantConfigQuery.data?.attributes?.characters?.presets ?? []
  )
    .filter((p): p is typeof p & { id: string } => !!p.id)
    .map((p) => ({
      id: p.id,
      attributes: { name: p.attributes.name, tags: p.attributes.tags },
    }));

  // ---------------------------------------------------------------------------
  // Step 3 side effect: apply presets sequentially
  // ---------------------------------------------------------------------------
  const applyInProgress = useRef(false);

  useEffect(() => {
    if (state.step !== 3 || applyInProgress.current) return;
    applyInProgress.current = true;

    const rowList = Object.values(state.rows);

    async function run() {
      for (const row of rowList) {
        dispatch({ type: "SET_ROW_STATUS", presetId: row.presetId, status: "applying" });
        try {
          await factoryService.createFromPreset(tenant, {
            presetId: row.presetId,
            accountId,
            worldId: state.worldId,
            name: row.name,
          });
          dispatch({ type: "SET_ROW_STATUS", presetId: row.presetId, status: "success" });
        } catch (err) {
          const msg = createErrorFromUnknown(err).message;
          dispatch({
            type: "SET_ROW_STATUS",
            presetId: row.presetId,
            status: "failed",
            error: msg,
          });
        }
      }
      applyInProgress.current = false;
    }

    run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.step]);

  const handleRetry = async (presetId: string) => {
    const row = state.rows[presetId];
    if (!row) return;

    dispatch({ type: "SET_ROW_STATUS", presetId, status: "applying" });
    try {
      await factoryService.createFromPreset(tenant, {
        presetId,
        accountId,
        worldId: state.worldId,
        name: row.name,
      });
      dispatch({ type: "SET_ROW_STATUS", presetId, status: "success" });
    } catch (err) {
      const msg = createErrorFromUnknown(err).message;
      dispatch({ type: "SET_ROW_STATUS", presetId, status: "failed", error: msg });
    }
  };

  const stepLabels: Record<FlowStep, string> = {
    1: "Step 1 of 3 — World & presets",
    2: "Step 2 of 3 — Character names",
    3: "Step 3 of 3 — Applying",
  };

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">{stepLabels[state.step]}</p>

      {state.step === 1 && (
        <StepA
          worldId={state.worldId}
          tagFilter={state.tagFilter}
          selectedPresetIds={new Set(Object.keys(state.rows))}
          allPresets={allPresets}
          onWorldChange={(worldId) => dispatch({ type: "SET_WORLD", worldId })}
          onTagFilterChange={(tags) => dispatch({ type: "SET_TAG_FILTER", tags })}
          onTogglePreset={(presetId, presetName) =>
            dispatch({ type: "TOGGLE_PRESET", presetId, presetName })
          }
          onClose={onClose}
          onNext={() => dispatch({ type: "GOTO", step: 2 })}
        />
      )}

      {state.step === 2 && (
        <StepB
          tenant={tenant}
          rows={state.rows}
          onChangeName={(presetId, name) => dispatch({ type: "SET_NAME", presetId, name })}
          onValidityUpdate={(presetId, validity) =>
            dispatch({ type: "SET_VALIDITY", presetId, validity })
          }
          onBack={() => dispatch({ type: "GOTO", step: 1 })}
          onNext={() => {
            applyInProgress.current = false;
            dispatch({ type: "GOTO", step: 3 });
          }}
        />
      )}

      {state.step === 3 && (
        <StepC
          tenant={tenant}
          rows={state.rows}
          accountId={accountId}
          error={state.error}
          onRetry={handleRetry}
          onDone={() => {
            onComplete?.();
            onClose?.();
          }}
        />
      )}
    </div>
  );
}
