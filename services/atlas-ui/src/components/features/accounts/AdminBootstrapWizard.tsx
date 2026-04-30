import { useReducer, useEffect, useRef } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useNameValidity } from "@/lib/hooks/api/useNameValidity";
import { factoryService } from "@/services/api/factory.service";
import { accountsService } from "@/services/api/accounts.service";
import type { Tenant } from "@/types/models/tenant";
import type { NameValidityResponse } from "@/services/api/factory.service";
import {
  wizardReducer,
  initialState,
  type Row,
} from "./AdminBootstrapWizard.types";

// ---------------------------------------------------------------------------
// Preset shape from tenant config (same casting pattern as ApplyPresetDialog)
// ---------------------------------------------------------------------------
interface PresetItem {
  id: string;
  attributes: {
    name: string;
    tags?: string[];
  };
}

// ---------------------------------------------------------------------------
// Step 1 — Account credentials
// ---------------------------------------------------------------------------
interface Step1Props {
  accountName: string;
  password: string;
  onChange: (name: string, password: string) => void;
  onNext: () => void;
}

function Step1({ accountName, password, onChange, onNext }: Step1Props) {
  const nameError = accountName.length > 0 && accountName.length < 4 ? "At least 4 characters" : undefined;
  const pwError = password.length > 0 && password.length < 6 ? "At least 6 characters" : undefined;
  const canNext = accountName.length >= 4 && password.length >= 6;

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Enter the credentials for the new admin account. An account with this name will be created.
      </p>
      <div className="space-y-2">
        <Label htmlFor="bootstrap-name">Account name</Label>
        <Input
          id="bootstrap-name"
          value={accountName}
          onChange={(e) => onChange(e.target.value, password)}
          placeholder="admin"
          autoComplete="off"
        />
        {nameError && <p className="text-xs text-destructive">{nameError}</p>}
      </div>
      <div className="space-y-2">
        <Label htmlFor="bootstrap-password">Password</Label>
        <Input
          id="bootstrap-password"
          type="password"
          value={password}
          onChange={(e) => onChange(accountName, e.target.value)}
          autoComplete="new-password"
        />
        {pwError && <p className="text-xs text-destructive">{pwError}</p>}
      </div>
      <DialogFooter>
        <Button onClick={onNext} disabled={!canNext}>
          Next
        </Button>
      </DialogFooter>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step 2 — World + tag filter + preset selection
// ---------------------------------------------------------------------------
interface Step2Props {
  worldId: number;
  tagFilter: string[];
  selectedPresetIds: Set<string>;
  allPresets: PresetItem[];
  onWorldChange: (worldId: number) => void;
  onTagFilterChange: (tags: string[]) => void;
  onTogglePreset: (presetId: string, presetName: string) => void;
  onBack: () => void;
  onNext: () => void;
}

function Step2({
  worldId,
  tagFilter,
  selectedPresetIds,
  allPresets,
  onWorldChange,
  onTagFilterChange,
  onTogglePreset,
  onBack,
  onNext,
}: Step2Props) {
  // Collect all unique tags across presets
  const allTags = Array.from(
    new Set(allPresets.flatMap((p) => p.attributes.tags ?? [])),
  ).sort();

  // Filter: presets where every selected tag is present in the preset's tags
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
        <Label htmlFor="bootstrap-world">World ID</Label>
        <Input
          id="bootstrap-world"
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
            <p className="p-4 text-sm text-muted-foreground">No presets match the selected tags.</p>
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
        <Button variant="outline" onClick={onBack}>
          Back
        </Button>
        <Button onClick={onNext} disabled={!canNext}>
          Next
        </Button>
      </DialogFooter>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step 3 — Name overrides (one row per selected preset)
// ---------------------------------------------------------------------------

/** Inner component that handles a single row's name validity query */
interface NameRowProps {
  tenant: Tenant;
  row: Row;
  isDuplicate: boolean;
  onChangeName: (name: string) => void;
  onValidityUpdate: (validity: NameValidityResponse) => void;
}

function NameRow({ tenant, row, isDuplicate, onChangeName, onValidityUpdate }: NameRowProps) {
  const validityQuery = useNameValidity(tenant, row.name, 0, {
    enabled: row.name.length >= 3 && !isDuplicate,
  });

  useEffect(() => {
    if (validityQuery.data) {
      onValidityUpdate(validityQuery.data);
    }
  }, [validityQuery.data, onValidityUpdate]);

  // Derive effective validity: wizard-internal duplicate overrides server result
  const effectiveValidity: NameValidityResponse | null = isDuplicate
    ? { valid: false, reason: "duplicate", detail: "Duplicate within wizard" }
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
            <span className="text-green-600">Available</span>
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

interface Step3Props {
  tenant: Tenant;
  rows: Record<string, Row>;
  onChangeName: (presetId: string, name: string) => void;
  onValidityUpdate: (presetId: string, validity: NameValidityResponse) => void;
  onBack: () => void;
  onNext: () => void;
}

function Step3({ tenant, rows, onChangeName, onValidityUpdate, onBack, onNext }: Step3Props) {
  const rowList = Object.values(rows);

  // Wizard-internal duplicate detection: find names that appear more than once
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
// Step 4 — Apply
// ---------------------------------------------------------------------------

const STATUS_LABEL: Record<string, string> = {
  pending: "Pending",
  applying: "Applying…",
  success: "Success",
  failed: "Failed",
};

const STATUS_CLASS: Record<string, string> = {
  pending: "text-muted-foreground",
  applying: "text-blue-600",
  success: "text-green-600",
  failed: "text-destructive",
};

interface Step4Props {
  tenant: Tenant;
  rows: Record<string, Row>;
  accountId: number | undefined;
  accountName: string;
  error: string | undefined;
  onRetry: (presetId: string) => void;
  onDone: () => void;
}

function Step4({ rows, accountId, accountName, error, onRetry, onDone }: Step4Props) {
  const rowList = Object.values(rows);
  const allTerminal = rowList.every(
    (r) => r.applyStatus === "success" || r.applyStatus === "failed",
  );

  return (
    <div className="space-y-4">
      {!accountId && !error && (
        <p className="text-sm text-muted-foreground">
          Creating account <strong>{accountName}</strong>…
        </p>
      )}
      {accountId && (
        <p className="text-sm text-green-600">
          Account <strong>{accountName}</strong> created (ID {accountId}).
        </p>
      )}
      {error && (
        <p className="text-sm text-destructive">{error}</p>
      )}

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
// Main wizard component
// ---------------------------------------------------------------------------

export interface AdminBootstrapWizardProps {
  tenant: Tenant;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AdminBootstrapWizard({ tenant, open, onOpenChange }: AdminBootstrapWizardProps) {
  const [state, dispatch] = useReducer(wizardReducer, initialState);

  // Fetch presets from tenant config
  const tenantConfigQuery = useTenantConfiguration(tenant.id);
  const allPresets = (
    ((tenantConfigQuery.data?.attributes as any)?.characters as any)?.presets ?? []
  ) as PresetItem[];

  // Reset wizard when dialog opens
  useEffect(() => {
    if (open) {
      dispatch({ type: "RESET" });
    }
  }, [open]);

  // ---------------------------------------------------------------------------
  // Step 4 side effects: create account, poll for account ID, apply presets
  // ---------------------------------------------------------------------------
  const applyInProgress = useRef(false);

  useEffect(() => {
    if (state.step !== 4 || applyInProgress.current) return;
    applyInProgress.current = true;

    const rowList = Object.values(state.rows);

    async function run() {
      // 1. Create the account
      try {
        await accountsService.createAccount(tenant, state.account);
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Account creation failed";
        dispatch({ type: "SET_ERROR", error: msg });
        applyInProgress.current = false;
        return;
      }

      // 2. Poll until account materializes (up to 30s, 1s intervals)
      let accountId: number | undefined;
      const POLL_INTERVAL = 1000;
      const POLL_TIMEOUT = 30000;
      const deadline = Date.now() + POLL_TIMEOUT;

      while (Date.now() < deadline) {
        await new Promise<void>((res) => setTimeout(res, POLL_INTERVAL));
        try {
          const accounts = await accountsService.getAllAccounts({ name: state.account.name });
          const found = accounts.find((a) => a.attributes.name === state.account.name);
          if (found) {
            accountId = Number(found.id);
            dispatch({ type: "ACCOUNT_CREATED", accountId });
            break;
          }
        } catch {
          // transient failure; keep polling
        }
      }

      if (!accountId) {
        dispatch({ type: "SET_ERROR", error: "Timed out waiting for account to appear." });
        applyInProgress.current = false;
        return;
      }

      // 3. Apply each preset sequentially
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
          const msg = err instanceof Error ? err.message : "Failed";
          dispatch({ type: "SET_ROW_STATUS", presetId: row.presetId, status: "failed", error: msg });
        }
      }

      applyInProgress.current = false;
    }

    run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.step]);

  // Retry a single failed row
  const handleRetry = async (presetId: string) => {
    if (!state.accountId) return;
    const row = state.rows[presetId];
    if (!row) return;

    dispatch({ type: "SET_ROW_STATUS", presetId, status: "applying" });
    try {
      await factoryService.createFromPreset(tenant, {
        presetId,
        accountId: state.accountId,
        worldId: state.worldId,
        name: row.name,
      });
      dispatch({ type: "SET_ROW_STATUS", presetId, status: "success" });
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed";
      dispatch({ type: "SET_ROW_STATUS", presetId, status: "failed", error: msg });
    }
  };

  const handleClose = () => {
    onOpenChange(false);
  };

  const stepTitles: Record<number, string> = {
    1: "Step 1 of 4 — Account credentials",
    2: "Step 2 of 4 — World & presets",
    3: "Step 3 of 4 — Character names",
    4: "Step 4 of 4 — Applying",
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Bootstrap Admin Account</DialogTitle>
          <p className="text-xs text-muted-foreground">{stepTitles[state.step]}</p>
        </DialogHeader>

        {state.step === 1 && (
          <Step1
            accountName={state.account.name}
            password={state.account.password}
            onChange={(name, password) =>
              dispatch({ type: "SET_ACCOUNT", account: { name, password } })
            }
            onNext={() => {
              dispatch({ type: "SET_ACCOUNT", account: state.account });
              dispatch({ type: "GOTO", step: 2 });
            }}
          />
        )}

        {state.step === 2 && (
          <Step2
            worldId={state.worldId}
            tagFilter={state.tagFilter}
            selectedPresetIds={new Set(Object.keys(state.rows))}
            allPresets={allPresets}
            onWorldChange={(worldId) => dispatch({ type: "SET_WORLD", worldId })}
            onTagFilterChange={(tags) => dispatch({ type: "SET_TAG_FILTER", tags })}
            onTogglePreset={(presetId, presetName) =>
              dispatch({ type: "TOGGLE_PRESET", presetId, presetName })
            }
            onBack={() => dispatch({ type: "GOTO", step: 1 })}
            onNext={() => dispatch({ type: "GOTO", step: 3 })}
          />
        )}

        {state.step === 3 && (
          <Step3
            tenant={tenant}
            rows={state.rows}
            onChangeName={(presetId, name) => dispatch({ type: "SET_NAME", presetId, name })}
            onValidityUpdate={(presetId, validity) =>
              dispatch({ type: "SET_VALIDITY", presetId, validity })
            }
            onBack={() => dispatch({ type: "GOTO", step: 2 })}
            onNext={() => {
              applyInProgress.current = false;
              dispatch({ type: "GOTO", step: 4 });
            }}
          />
        )}

        {state.step === 4 && (
          <Step4
            tenant={tenant}
            rows={state.rows}
            accountId={state.accountId}
            accountName={state.account.name}
            error={state.error}
            onRetry={handleRetry}
            onDone={handleClose}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}
