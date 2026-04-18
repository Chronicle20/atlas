import { useEffect } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useParams } from "react-router-dom";
import { X } from "lucide-react";
import { useTenantConfiguration, useUpdateTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { OptionsField } from "@/components/unknown-options";
import { toast } from "sonner";

interface FormValues {
    handlers: {
        opCode: string;
        validator: string;
        handler: string;
        options: unknown;
    }[];
}

export function HandlersForm() {
    const { id } = useParams();
    const tenantQuery = useTenantConfiguration(id ?? "");
    const updateTenantConfig = useUpdateTenantConfiguration();

    const tenant = tenantQuery.data ?? null;
    const loading = tenantQuery.isLoading;

    const form = useForm<FormValues>({ defaultValues: { handlers: [] } });

    const { fields, append, remove } = useFieldArray({ control: form.control, name: "handlers" });

    useEffect(() => {
        if (tenant) {
            form.reset({
                handlers: tenant.attributes.socket.handlers.map(handler => ({
                    opCode: handler.opCode || "",
                    validator: handler.validator || "",
                    handler: handler.handler || "",
                    options: handler.options,
                })),
            });
        }
    }, [tenant, form]);

    const onSubmit = (data: FormValues) => {
        if (!tenant) return;
        updateTenantConfig.mutate(
            {
                tenant,
                updates: {
                    socket: {
                        handlers: data.handlers,
                        writers: tenant.attributes.socket.writers || [],
                    },
                },
            },
            {
                onSuccess: () => toast.success("Successfully saved tenant configuration."),
                onError: () => toast.error("Failed to update tenant configuration"),
            },
        );
    };

    if (loading) {
        return <div className="flex justify-center items-center p-8">Loading tenant configuration...</div>;
    }

    if (!tenant) {
        return <div className="flex justify-center items-center p-8">Tenant configuration not found</div>;
    }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                {fields.map((field, index) => (
                    <div key={field.id} className="border p-4 rounded-md gap-2 relative flex flex-col justify-stretch">
                        <div className="gap-2 flex flex-row justify-stretch">
                            <FormField
                                control={form.control}
                                name={`handlers.${index}.opCode`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Operation Code</FormLabel>
                                        <FormControl><Input placeholder="0x00" {...field} /></FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`handlers.${index}.validator`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Validator</FormLabel>
                                        <FormControl><Input {...field} /></FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`handlers.${index}.handler`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Handler</FormLabel>
                                        <FormControl><Input {...field} /></FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                        </div>
                        <OptionsField form={form} path={`handlers.${index}.options`} />
                        <Button type="button" className="absolute top-0 right-0" variant="ghost" size="icon" onClick={() => remove(index)}>
                            <X />
                        </Button>
                    </div>
                ))}
                <div className="flex flex-row gap-2 justify-between">
                    <Button type="button" onClick={() => append({ opCode: "", validator: "", handler: "", options: null })}>
                        Add
                    </Button>
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
    );
}
