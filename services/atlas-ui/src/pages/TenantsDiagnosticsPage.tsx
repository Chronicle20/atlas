import { useEffect } from "react";
import { useParams } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import {
  useTenantConfiguration,
  useUpdateTenantConfiguration,
} from "@/lib/hooks/api/useTenants";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
} from "@/components/ui/form";
import { Switch } from "@/components/ui/switch";

const diagnosticsFormSchema = z.object({
  tracePackets: z.boolean(),
});

type DiagnosticsFormValues = z.infer<typeof diagnosticsFormSchema>;

export function TenantsDiagnosticsPage() {
  const { id } = useParams();
  const tenantQuery = useTenantConfiguration(id ?? "");
  const updateTenantConfig = useUpdateTenantConfiguration();
  const tenant = tenantQuery.data;

  const form = useForm<DiagnosticsFormValues>({
    resolver: zodResolver(diagnosticsFormSchema),
    defaultValues: { tracePackets: false },
  });

  useEffect(() => {
    if (tenant) {
      form.reset({
        tracePackets: tenant.attributes.diagnostics?.tracePackets ?? false,
      });
    }
  }, [tenant, form]);

  const onSubmit = (values: DiagnosticsFormValues) => {
    if (!tenant) return;
    updateTenantConfig.mutate(
      {
        tenant,
        updates: { diagnostics: { tracePackets: values.tracePackets } },
      },
      {
        onSuccess: () =>
          toast.success("Successfully saved diagnostics configuration."),
        onError: (error) =>
          toast.error(
            `Failed to update diagnostics configuration: ${error.message}`,
          ),
      },
    );
  };

  return (
    <TenantDetailLayout>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="space-y-4 max-w-2xl"
        >
          <Alert variant="destructive">
            <AlertTitle>Packet tracing is credential-bearing</AlertTitle>
            <AlertDescription>
              <p>
                Turning this on writes every inbound and outbound packet for
                this tenant to its serving pods&apos; logs, which generates very
                large volumes of log output. Use it only for short reproduction
                windows.
              </p>
              <p>
                The dump is unredacted: login packets carry account passwords,
                PICs/PINs and HWIDs in plaintext, so any log captured while this
                is on must be treated as credential-bearing material.
              </p>
              <p>
                Nothing is emitted unless the serving pod also runs at
                LOG_LEVEL=Debug. Turning this on alone produces no output.
              </p>
            </AlertDescription>
          </Alert>
          <FormField
            control={form.control}
            name="tracePackets"
            render={({ field }) => (
              <FormItem className="flex flex-row items-center justify-between rounded-lg border p-3 shadow-xs">
                <div className="space-y-0.5">
                  <FormLabel>Trace packets</FormLabel>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />
          <div className="flex flex-row gap-2 justify-end">
            <Button type="submit" disabled={updateTenantConfig.isPending}>
              Save
            </Button>
          </div>
        </form>
      </Form>
    </TenantDetailLayout>
  );
}
