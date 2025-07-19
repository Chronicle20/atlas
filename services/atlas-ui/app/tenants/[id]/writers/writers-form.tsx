"use client"

import {useEffect, useState} from "react";
import {useFieldArray, useForm} from "react-hook-form";
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";
import {X} from "lucide-react";
import {OptionsField} from "@/components/unknown-options";
import {updateTenantConfiguration} from "@/lib/tenants";
import {TenantConfig} from "@/types/models/tenant";
import {toast} from "sonner";

export function WritersForm() {
    const {id} = useParams(); // Get tenants ID from URL
    const {fetchTenantConfiguration} = useTenant();
    const [tenant, setTenant] = useState<TenantConfig | null>(null);
    const [loading, setLoading] = useState(true);

    interface FormValues {
        writers: {
            opCode: string;
            writer: string;
            options: unknown;
        }[];
    }

    const form = useForm<FormValues>({
        defaultValues: {
            writers: []
        }
    });

    const {fields, append, remove} = useFieldArray({
        control: form.control,
        name: "writers"
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
                        writers: tenantConfig.attributes.socket.writers.map(writer => ({
                            opCode: writer.opCode || "",
                            writer: writer.writer || "",
                            options: writer.options,
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
            const updatedTenant = await updateTenantConfiguration(tenant, {
                socket: {
                    handlers: tenant.attributes.socket.handlers || [],
                    writers: data.writers,
                },
            });

            if (updatedTenant) {
                setTenant(updatedTenant);
                toast.success("Successfully saved tenant configuration.");

                form.reset({
                    writers: updatedTenant.attributes.socket.writers.map(writer => ({
                        opCode: writer.opCode || "",
                        writer: writer.writer || "",
                        options: writer.options,
                    }))
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
                                name={`writers.${index}.opCode`}
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
                                name={`writers.${index}.writer`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Writer</FormLabel>
                                        <FormControl>
                                            <Input {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                        </div>
                        <OptionsField form={form} path={`writers.${index}.options`}/>
                        <Button type="button" className="absolute top-0 right-0" variant="ghost" size="icon"
                                onClick={() => remove(index)}>
                            <X/>
                        </Button>
                    </div>
                ))}
                <div className="flex flex-row gap-2 justify-between">
                    <Button type="button" onClick={() => append({opCode: "", writer: "", options: null})}>
                        Add
                    </Button>
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
    );
}
