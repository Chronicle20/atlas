import { useState } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { OptionsMatrixTable } from "@/components/features/socket/OptionsMatrix";
import type { SocketTarget } from "@/lib/hooks/api/useSocketObjects";
import { useSocketMutation } from "@/lib/hooks/api/useSocketObjects";
import { missingFromTenant } from "@/lib/socket/ancestry";
import {
  copyMissingFromAncestor,
  type AncestorAddition,
  type BindingInput,
} from "@/lib/socket/mutate";
import {
  entriesOf,
  stateOf,
  type Binding,
  type DefinitionKind,
  type DefinitionState,
  type SocketObject,
} from "@/lib/socket/model";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface CopyFromAncestorFlowProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The current Tenant (or Template-as-target) object being copied INTO. */
  tenant: SocketObject;
  /** The ancestor Template `missingFromTenant` scans for candidates. */
  ancestor: SocketObject;
  kind: DefinitionKind;
  target: SocketTarget;
}

type Step = "candidates" | "review";

/**
 * Keys a per-binding target-opcode override. `opCodeValue` (not stored index)
 * because `entriesOf` returns bindings re-sorted by opcode on read - the same
 * key discipline `mutate.ts#resolveOne` uses.
 */
function targetOpcodeKey(name: string, opCodeValue: number | null): string {
  return `${name}|${opCodeValue}`;
}

function stateLabel(state: DefinitionState): string {
  switch (state) {
    case "defined":
      return "Defined";
    case "unsupported":
      return "Unsupported";
    case "undefined":
      return "Undefined";
  }
}

/**
 * Top-level option key names (e.g. "types", "operations"), shown as a
 * caption alongside the full `OptionsMatrixTable` - the table itself only
 * labels a list-shaped option by row INDEX ("0", "1", ...), never by the
 * wrapping key name, so FR-9.3's "option differences" needs this to actually
 * name what's being compared.
 */
function optionKeyList(options: unknown): string | null {
  if (
    options === null ||
    options === undefined ||
    typeof options !== "object"
  ) {
    return null;
  }
  const keys = Object.keys(options as Record<string, unknown>);
  return keys.length > 0 ? keys.join(", ") : null;
}

/**
 * FR-9.1-9.6. Two steps in one dialog:
 *   1. Candidates - every name `missingFromTenant` reports (defined in the
 *      ancestor, undefined in the tenant - FR-9.5's Unsupported exclusion is
 *      already baked into that function, not re-implemented here).
 *   2. Review - one block per selected name showing all seven FR-9.3 fields,
 *      with the target opcode adjustable per binding.
 *
 * Apply builds the whole selection into one `AncestorAddition[]` and submits
 * it as exactly one `useSocketMutation` call (FR-9.6). The write itself is
 * `copyMissingFromAncestor`, which re-checks "already defined" against the
 * FRESH document `useSocketMutation` re-fetches - never against this stale
 * scan - so a name that gained a definition between the scan and the apply
 * is skipped, not clobbered (FR-9.4).
 */
export function CopyFromAncestorFlow({
  open,
  onOpenChange,
  tenant,
  ancestor,
  kind,
  target,
}: CopyFromAncestorFlowProps) {
  const mutation = useSocketMutation();
  const [step, setStep] = useState<Step>("candidates");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [targetOpcodes, setTargetOpcodes] = useState<Record<string, string>>(
    {},
  );

  const candidates = missingFromTenant(tenant, ancestor, kind);
  const selectedNames = candidates.filter((name) => selected.has(name));

  const close = () => {
    setStep("candidates");
    setSelected(new Set());
    setTargetOpcodes({});
    onOpenChange(false);
  };

  const toggle = (name: string, checked: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) next.add(name);
      else next.delete(name);
      return next;
    });
  };

  const toggleAll = (checked: boolean) => {
    setSelected(checked ? new Set(candidates) : new Set());
  };

  const allSelected =
    candidates.length > 0 && selected.size === candidates.length;

  const onApply = async () => {
    const additions: AncestorAddition[] = selectedNames.map((name) => {
      const bindings = entriesOf(ancestor, kind).get(name) ?? [];
      const inputs: BindingInput[] = bindings.map((b) => ({
        opCode: targetOpcodes[targetOpcodeKey(name, b.opCodeValue)] ?? b.opCode,
        services: b.services,
        ...(b.validator !== undefined ? { validator: b.validator } : {}),
        ...(b.options !== undefined ? { options: b.options } : {}),
        // fname is informational and is copied along with the rest; it never
        // participates in comparison or validation (FR-10.4).
        ...(b.fname !== undefined ? { fname: b.fname } : {}),
      }));
      return { name, bindings: inputs };
    });

    try {
      await mutation.mutateAsync({
        target,
        apply: (cfg) => copyMissingFromAncestor(cfg, kind, additions),
      });
      toast.success(
        `Copied ${additions.length} definition${additions.length === 1 ? "" : "s"} from ${ancestor.label} to ${tenant.label}`,
      );
      close();
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) close();
        else onOpenChange(next);
      }}
    >
      <DialogContent className="max-h-[85vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            Copy missing definitions from {ancestor.label} to {tenant.label}
          </DialogTitle>
        </DialogHeader>

        {step === "candidates" ? (
          <>
            <fieldset
              role="group"
              aria-label="Candidates"
              className="space-y-2"
            >
              <label className="flex items-center gap-2 text-sm font-medium">
                <input
                  type="checkbox"
                  aria-label="Select all"
                  checked={allSelected}
                  disabled={candidates.length === 0}
                  onChange={(e) => toggleAll(e.target.checked)}
                />
                Select all
              </label>
              {candidates.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                  {`Every definition ${ancestor.label} defines is already present or explicitly Unsupported in ${tenant.label}.`}
                </p>
              ) : (
                candidates.map((name) => {
                  const bindings = entriesOf(ancestor, kind).get(name) ?? [];
                  const opcodes = bindings.map((b) => b.opCode).join(", ");
                  return (
                    <label
                      key={name}
                      className="flex items-center gap-2 text-sm"
                    >
                      <input
                        type="checkbox"
                        checked={selected.has(name)}
                        onChange={(e) => toggle(name, e.target.checked)}
                      />
                      {`${name} (${opcodes})`}
                    </label>
                  );
                })
              )}
            </fieldset>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={close}>
                Cancel
              </Button>
              <Button
                type="button"
                disabled={selectedNames.length === 0}
                onClick={() => setStep("review")}
              >
                Review
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <section role="region" aria-label="Review" className="space-y-6">
              {selectedNames.map((name) => {
                const bindings = entriesOf(ancestor, kind).get(name) ?? [];
                const state = stateOf(tenant, kind, name);
                return (
                  <ReviewBlock
                    key={name}
                    name={name}
                    bindings={bindings}
                    kind={kind}
                    state={state}
                    ancestor={ancestor}
                    tenant={tenant}
                    targetOpcodes={targetOpcodes}
                    onTargetOpcodeChange={(key, value) =>
                      setTargetOpcodes((prev) => ({ ...prev, [key]: value }))
                    }
                  />
                );
              })}
            </section>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setStep("candidates")}
              >
                Back
              </Button>
              <Button
                type="button"
                disabled={mutation.isPending}
                onClick={onApply}
              >
                Apply
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

interface ReviewBlockProps {
  name: string;
  bindings: Binding[];
  kind: DefinitionKind;
  state: DefinitionState;
  ancestor: SocketObject;
  tenant: SocketObject;
  targetOpcodes: Record<string, string>;
  onTargetOpcodeChange: (key: string, value: string) => void;
}

/** One FR-9.3 review row: name, source opcode, target opcode, validator, services, option differences, current target state. */
function ReviewBlock({
  name,
  bindings,
  kind,
  state,
  ancestor,
  tenant,
  targetOpcodes,
  onTargetOpcodeChange,
}: ReviewBlockProps) {
  return (
    <div className="space-y-3 rounded-md border p-3">
      <h4 className="text-sm font-semibold">{name}</h4>
      <p className="text-muted-foreground text-xs">
        {`Current target state: ${stateLabel(state)}`}
      </p>
      {bindings.map((b, i) => {
        const label =
          bindings.length > 1
            ? `Target opcode for ${name} (${b.opCode})`
            : `Target opcode for ${name}`;
        const inputId = `copy-ancestor-target-opcode-${name}-${i}`;
        const key = targetOpcodeKey(name, b.opCodeValue);
        const keys = optionKeyList(b.options);
        return (
          <div key={key} className="grid grid-cols-2 gap-3 text-sm">
            <div>
              <p className="text-muted-foreground text-xs">Source opcode</p>
              <p className="font-mono">{b.opCode}</p>
            </div>
            <div>
              <Label htmlFor={inputId} className="text-xs font-normal">
                {label}
              </Label>
              <Input
                id={inputId}
                defaultValue={targetOpcodes[key] ?? b.opCode}
                onChange={(e) => onTargetOpcodeChange(key, e.target.value)}
              />
            </div>
            {kind === "handler" && (
              <div>
                <p className="text-muted-foreground text-xs">Validator</p>
                <p>{b.validator || "—"}</p>
              </div>
            )}
            <div>
              <p className="text-muted-foreground text-xs">Services</p>
              <p>{b.services.length > 0 ? b.services.join(", ") : "—"}</p>
            </div>
            {keys && (
              <p className="text-muted-foreground col-span-2 text-xs">
                {`Option keys: ${keys}`}
              </p>
            )}
            <div className="col-span-2">
              <OptionsMatrixTable
                objects={[ancestor, tenant]}
                kind={kind}
                name={name}
                baselineKey={ancestor.key}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}
