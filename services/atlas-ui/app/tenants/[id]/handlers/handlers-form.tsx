"use client"

import {useEffect, useState} from "react";
import {useFieldArray, useForm} from "react-hook-form";
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";
import {X} from "lucide-react";
import {tenantsService} from "@/services/api";
import {TenantConfig} from "@/types/models/tenant";
import {OptionsField} from "@/components/unknown-options";
import {toast} from "sonner";

export function HandlersForm() {
    const {id} = useParams(); // Get tenants ID from URL
    const {fetchTenantConfiguration} = useTenant();
    const [tenant, setTenant] = useState<TenantConfig | null>(null);
    const [loading, setLoading] = useState(true);

    interface FormValues {
        handlers: {
            opCode: string;
            validator: string;
            handler: string;
            options: unknown;
        }[];
    }

    const form = useForm<FormValues>({
        defaultValues: {
            handlers: []
        }
    });

    const {fields, append, remove} = useFieldArray({
        control: form.control,
        name: "handlers"
    });

    // Fetch the full tenant configuration
    useEffect(() => {
        const fetchTenant = async () => {
            try {
                setLoading(true);
                if (id) {
                    const tenantConfig = await fetchTenantConfiguration(id as string);
                    setTenant(tenantConfig);

                    // Update form with tenant data
                    form.reset({
                        handlers: tenantConfig.attributes.socket.handlers.map(handler => ({
                            opCode: handler.opCode || "",
                            validator: handler.validator || "",
                            handler: handler.handler || "",
                            options: handler.options,
                        }))
                    });
                }
            } catch (error) {
                console.error("Error fetching tenant configuration:", error);
                toast.error("Failed to load tenant configuration");
            } finally {
                setLoading(false);
            }
        };

        fetchTenant();
    }, [id, fetchTenantConfiguration, form]);

    const onSubmit = async (data: FormValues) => {
        if (!tenant) return;

        try {
            const updatedTenant = await tenantsService.updateConfiguration(tenant, {
                socket: {
                    handlers: data.handlers,
                    writers: tenant.attributes.socket.writers || [],
                },
            });

            if (updatedTenant) {
                setTenant(updatedTenant);
                toast.success("Successfully saved tenant configuration.");

                form.reset({
                    handlers: updatedTenant.attributes.socket.handlers,
                });
            }
        } catch (error) {
            console.error("Error updating tenant configuration:", error);
            toast.error("Failed to update tenant configuration");
        }
    }

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
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Operation Code</FormLabel>
                                        <FormControl>
                                            <Input placeholder="0x00" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`handlers.${index}.validator`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Validator</FormLabel>
                                        <FormControl>
                                            <Input {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`handlers.${index}.handler`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Handler</FormLabel>
                                        <FormControl>
                                            <Input {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                        </div>
                        <OptionsField form={form} path={`handlers.${index}.options`}/>
                        <Button type="button" className="absolute top-0 right-0" variant="ghost" size="icon"
                                onClick={() => remove(index)}>
                            <X/>
                        </Button>
                    </div>
                ))}
                <div className="flex flex-row gap-2 justify-between">
                    <Button type="button"
                            onClick={() => append({opCode: "", validator: "", handler: "", options: null})}>
                        Add
                    </Button>
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
    );
}
