/**
 * Non-component pieces shared by the six definition dialogs. Kept out of
 * fields.tsx (component-only, so Fast Refresh's
 * react-refresh/only-export-components rule stays satisfied) and out of
 * mutate.ts (pure data layer, no UI/form concerns).
 */
import {
  KNOWN_SERVICES,
  type KnownService,
} from "@/lib/schemas/socket-definition";
import type { SocketTarget } from "@/lib/hooks/api/useSocketObjects";
import type { DefinitionKind } from "@/lib/socket/model";

/** Shared by all six dialogs (Required exported surface, Task 17 brief). */
export interface DialogBaseProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  target: SocketTarget;
  /** The scoped object's label, so every dialog title names it. */
  targetLabel: string;
  kind: DefinitionKind;
}

/**
 * Narrows a stored `services` list (plain `string[]` - `BindingInput` and
 * `Binding` do not constrain it) down to the closed set the form schema
 * accepts. Any other value is dropped rather than rejected outright: the
 * corpus never carries one, and a dialog pre-filling from live data should
 * not crash on an unexpected one either.
 */
export function toKnownServices(services: readonly string[]): KnownService[] {
  const known: readonly string[] = KNOWN_SERVICES;
  return services.filter((s): s is KnownService => known.includes(s));
}
