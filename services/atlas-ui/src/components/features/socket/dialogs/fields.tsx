/**
 * Component pieces shared by the six definition dialogs. Split out of
 * dialog-base.ts (non-component exports) so this file only exports
 * components, per react-refresh/only-export-components.
 */
import type { Control, FieldPath, FieldValues } from "react-hook-form";
import {
  FormControl,
  FormField as ShadcnFormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { KNOWN_SERVICES } from "@/lib/schemas/socket-definition";

interface ServicesFieldProps<
  TFieldValues extends FieldValues,
  TName extends FieldPath<TFieldValues>,
> {
  control: Control<TFieldValues>;
  name: TName;
  disabled?: boolean;
}

/**
 * KNOWN_SERVICES is the closed two-value set from libs/atlas-opcodes/config.go.
 * Leaving both boxes unchecked is a legal, common state (25 corpus entries
 * carry no `services` key at all) - this field never requires a selection.
 */
export function ServicesField<
  TFieldValues extends FieldValues,
  TName extends FieldPath<TFieldValues>,
>({ control, name, disabled }: ServicesFieldProps<TFieldValues, TName>) {
  return (
    <ShadcnFormField
      control={control}
      name={name}
      render={({ field }) => {
        const value = (field.value as string[] | undefined) ?? [];
        const toggle = (service: string, checked: boolean) => {
          field.onChange(
            checked ? [...value, service] : value.filter((s) => s !== service),
          );
        };
        return (
          <FormItem>
            <FormLabel>Services</FormLabel>
            <FormControl>
              <div className="flex gap-4">
                {KNOWN_SERVICES.map((service) => (
                  <label
                    key={service}
                    className="flex items-center gap-2 text-sm font-normal"
                  >
                    <input
                      type="checkbox"
                      checked={value.includes(service)}
                      disabled={disabled}
                      onChange={(e) => toggle(service, e.target.checked)}
                    />
                    {service}
                  </label>
                ))}
              </div>
            </FormControl>
            <FormMessage />
          </FormItem>
        );
      }}
    />
  );
}
