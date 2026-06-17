import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useParams } from "react-router-dom";
import { useMtsConfig, useUpdateMtsConfig } from "@/lib/hooks/api/useMtsConfig";
import { mtsConfigSchema, type MtsConfigFormData } from "@/lib/schemas/mts-config.schema";
import { toast } from "sonner";

const FIELDS: { name: keyof MtsConfigFormData; label: string; description: string; step?: string }[] = [
  { name: "listingFee", label: "Listing Fee", description: "Flat NX fee charged when a listing is created." },
  { name: "commissionRate", label: "Commission Rate", description: "Fractional cut taken on a sale (0 – 1).", step: "0.01" },
  { name: "maxActiveListings", label: "Max Active Listings", description: "Per-character cap on simultaneously active listings." },
  { name: "minLevel", label: "Minimum Level", description: "Minimum character level required to use the marketplace." },
  { name: "auctionMinHours", label: "Auction Minimum Hours", description: "Shortest allowed auction duration, in hours." },
  { name: "auctionMaxHours", label: "Auction Maximum Hours", description: "Longest allowed auction duration, in hours." },
  { name: "priceFloor", label: "Price Floor", description: "Minimum NX list value allowed for any listing." },
  { name: "pageSize", label: "Page Size", description: "Number of listings returned per browse page." },
  { name: "minBidIncrement", label: "Minimum Bid Increment", description: "Smallest allowed increase over the current bid." },
];

const EMPTY_DEFAULTS: MtsConfigFormData = {
  listingFee: 0,
  commissionRate: 0,
  maxActiveListings: 1,
  minLevel: 0,
  auctionMinHours: 1,
  auctionMaxHours: 1,
  priceFloor: 0,
  pageSize: 1,
  minBidIncrement: 1,
};

export function MtsConfigForm() {
  const { id } = useParams();
  const tenantId = id ?? "";
  const configQuery = useMtsConfig(tenantId);
  const updateConfig = useUpdateMtsConfig();

  const config = configQuery.data ?? null;
  const loading = configQuery.isLoading;

  const form = useForm<MtsConfigFormData>({
    resolver: zodResolver(mtsConfigSchema),
    defaultValues: EMPTY_DEFAULTS,
  });

  useEffect(() => {
    if (config) {
      form.reset(config.attributes);
    }
  }, [config, form]);

  const onSubmit = (data: MtsConfigFormData) => {
    if (!config) return;
    updateConfig.mutate(
      { tenantId, config, updates: data },
      {
        onSuccess: () => toast.success("Successfully saved MTS configuration."),
        onError: () => toast.error("Failed to update MTS configuration"),
      },
    );
  };

  if (loading) {
    return <div className="flex justify-center items-center p-8">Loading MTS configuration...</div>;
  }

  if (!config) {
    return (
      <div className="flex justify-center items-center p-8">
        No MTS configuration found for this tenant.
      </div>
    );
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {FIELDS.map((f) => (
            <FormField
              key={f.name}
              control={form.control}
              name={f.name}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{f.label}</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step={f.step ?? "1"}
                      {...field}
                      onChange={(e) =>
                        field.onChange(e.target.value === "" ? "" : e.target.valueAsNumber)
                      }
                    />
                  </FormControl>
                  <FormDescription>{f.description}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          ))}
        </div>
        <div className="flex flex-row justify-end">
          <Button type="submit" disabled={updateConfig.isPending}>
            Save
          </Button>
        </div>
      </form>
    </Form>
  );
}
