# Forms & Validation Patterns

## Overview

Forms use **react-hook-form** with **Zod** validation via `@hookform/resolvers/zod`. Schemas live in `lib/schemas/`, and form UI uses shadcn/ui `Form` components for consistent field rendering.

## Zod Schema Pattern

### Basic Schema

```typescript
// lib/schemas/tenant.schema.ts
import { z } from 'zod';

export const createTenantSchema = z.object({
  name: z
    .string()
    .min(1, 'Tenant name is required')
    .max(100, 'Tenant name must be 100 characters or less'),
  region: z
    .string()
    .min(1, 'Region is required'),
  majorVersion: z
    .number()
    .int('Major version must be an integer')
    .nonnegative('Major version must be non-negative'),
  minorVersion: z
    .number()
    .int('Minor version must be an integer')
    .nonnegative('Minor version must be non-negative'),
});

// Infer TypeScript type from schema
export type CreateTenantFormData = z.infer<typeof createTenantSchema>;

// Default values for form reset
export const createTenantDefaults: CreateTenantFormData = {
  name: '',
  region: '',
  majorVersion: 0,
  minorVersion: 0,
};
```

### Schema with Cross-Field Validation

```typescript
// Inline in component file for form-specific logic
const formSchema = z.object({
  banType: z.nativeEnum(BanType),
  value: z.string().min(1, "Value is required"),
  permanent: z.boolean(),
  expiresAt: z.string().optional(),
}).refine((data) => {
  // IP validation for IP type bans
  if (data.banType === BanType.IP && !ipRegex.test(data.value)) {
    return false;
  }
  return true;
}, {
  message: "Invalid IP address or CIDR format",
  path: ["value"],    // ← Attach error to specific field
}).refine((data) => {
  // Expiration required for non-permanent bans
  if (!data.permanent && !data.expiresAt) return false;
  return true;
}, {
  message: "Expiration date is required for non-permanent bans",
  path: ["expiresAt"],
});
```

### Discriminated Union Schema

```typescript
// lib/schemas/service.schema.ts
export const loginServiceSchema = z.object({
  type: z.literal('login-service'),
  tasks: z.array(taskConfigSchema),
  tenants: z.array(loginTenantSchema),
});

export const channelServiceSchema = z.object({
  type: z.literal('channel-service'),
  tasks: z.array(taskConfigSchema),
  tenants: z.array(channelTenantSchema),
});

// Combined with discriminated union
export const serviceSchema = z.discriminatedUnion('type', [
  loginServiceSchema,
  channelServiceSchema,
  dropsServiceSchema,
]);
```

### Reusable Validation Primitives

```typescript
export const portSchema = z
  .number()
  .int('Port must be an integer')
  .min(1, 'Port must be at least 1')
  .max(65535, 'Port must be at most 65535');

export const byteIdSchema = z
  .number()
  .int('ID must be an integer')
  .min(0, 'ID must be at least 0')
  .max(255, 'ID must be at most 255');
```

## Form Component Pattern

### Using shadcn/ui Form Components (Preferred)

```tsx
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";

export function CreateBanDialog({ open, onOpenChange, tenant, onSuccess }: Props) {
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      banType: BanType.IP,
      value: "",
      permanent: false,
    },
  });

  const isPermanent = form.watch("permanent");  // ← Watch for conditional rendering

  const onSubmit = async (values: FormValues) => {
    try {
      await bansService.createBan(tenant!, mapToRequest(values));
      toast.success("Ban created successfully");
      form.reset();
      onOpenChange(false);
      onSuccess?.();
    } catch (error) {
      toast.error("Failed to create ban: " + (error instanceof Error ? error.message : "Unknown error"));
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">

            {/* Select field */}
            <FormField
              control={form.control}
              name="banType"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Ban Type</FormLabel>
                  <Select
                    onValueChange={(value) => field.onChange(Number(value))}
                    defaultValue={field.value.toString()}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select ban type" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {Object.entries(BanTypeLabels).map(([value, label]) => (
                        <SelectItem key={value} value={value}>{label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Text input */}
            <FormField
              control={form.control}
              name="value"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Value</FormLabel>
                  <FormControl>
                    <Input placeholder="Enter value" {...field} />
                  </FormControl>
                  <FormDescription>The target to ban</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Switch (boolean) */}
            <FormField
              control={form.control}
              name="permanent"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between rounded-lg border p-3">
                  <div className="space-y-0.5">
                    <FormLabel>Permanent Ban</FormLabel>
                    <FormDescription>This ban will never expire</FormDescription>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />

            {/* Conditional field */}
            {!isPermanent && (
              <FormField
                control={form.control}
                name="expiresAt"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Expiration Date</FormLabel>
                    <FormControl>
                      <Input type="datetime-local" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={form.formState.isSubmitting}>
                {form.formState.isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Create Ban
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
```

### Using register() (Simpler forms)

```tsx
const { register, handleSubmit, setValue, watch, reset, formState: { errors, isSubmitting } } = useForm<CreateTenantFormData>({
  resolver: zodResolver(createTenantSchema),
  defaultValues: createTenantDefaults,
});

// Text input
<Input id="name" placeholder="Enter name" {...register("name")} disabled={isSubmitting} />
{errors.name && <p className="text-sm text-destructive">{errors.name.message}</p>}

// Select (manual setValue)
<Select value={selectedRegion} onValueChange={(v) => setValue("region", v)}>
  <SelectTrigger><SelectValue placeholder="Select region" /></SelectTrigger>
  <SelectContent>
    {regions.map(r => <SelectItem key={r} value={r}>{r}</SelectItem>)}
  </SelectContent>
</Select>
```

## Cascading Dropdown Pattern

For dependent selects (e.g., Region → Major Version → Minor Version):

```tsx
const selectedRegion = watch("region");
const selectedMajorVersion = watch("majorVersion");

// Filter options based on selected parent
const availableMajorVersions = useMemo(() => {
  if (!selectedRegion) return [];
  return [...new Set(
    templates.filter(t => t.attributes.region === selectedRegion)
      .map(t => t.attributes.majorVersion)
  )].sort((a, b) => a - b);
}, [templates, selectedRegion]);

// Reset dependent fields when parent changes
const handleRegionChange = (region: string) => {
  setValue("region", region);
  setValue("majorVersion", 0);   // ← Reset child
  setValue("minorVersion", 0);   // ← Reset grandchild
};
```

## Error Display Pattern

```tsx
// Field-level errors (via FormMessage or manual)
{errors.name && <p className="text-sm text-destructive">{errors.name.message}</p>}

// General form errors (via toast on submit)
toast.error("Failed to create: " + (error instanceof Error ? error.message : "Unknown error"));

// Typed error handling
if (error instanceof TemplateNotFoundError) {
  toast.error("Template no longer exists.");
} else if (error instanceof ConfigurationCreationError) {
  toast.error(`Created but configuration failed. ID: ${error.tenantId}`);
} else {
  toast.error("An unexpected error occurred.");
}
```

## Dialog Close Behavior

- Prevent close during submission: `if (!isSubmitting) { onOpenChange(newOpen); }`
- Reset form on close: `if (!newOpen) { reset(defaults); }`
- Navigate after success: `window.location.replace('/resource/' + id)` or `onSuccess?.()`
