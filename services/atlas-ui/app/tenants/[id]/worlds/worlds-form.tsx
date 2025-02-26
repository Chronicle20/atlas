"use client"

import {useEffect} from "react";
import {useFieldArray, useForm} from "react-hook-form";
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";
import {X} from "lucide-react";
import {updateTenant} from "@/lib/tenants";

export function WorldsForm() {
    const {id} = useParams(); // Get tenants ID from URL
    const {tenants} = useTenant()
    const tenant = tenants.find((t) => t.id === id);

    interface FormValues {
        worlds: {
            name: string;
            flag: string;
            eventMessage: string;
            serverMessage: string;
            whyAmIRecommended: string;
        }[];
    }

    const form = useForm<FormValues>({
        defaultValues: {
            worlds: tenant?.attributes.worlds.map(world => ({
                name: world.name || "",
                flag: world.flag || "",
                eventMessage: world.eventMessage || "",
                serverMessage: world.serverMessage || "",
                whyAmIRecommended: world.whyAmIRecommended || "",
            }))
        }
    });

    const {fields, append, remove} = useFieldArray({
        control: form.control,
        name: "worlds"
    });

    // Reset form values when `worlds` data changes
    useEffect(() => {
        form.reset({
            worlds: tenant?.attributes.worlds.map(world => ({
                name: world.name || "",
                flag: world.flag || "",
                eventMessage: world.eventMessage || "",
                serverMessage: world.serverMessage || "",
                whyAmIRecommended: world.whyAmIRecommended || "",
            }))
        });
    }, [tenant, form.reset, form]);

    const onSubmit = async (data: FormValues) => {
        await updateTenant(tenant, {
            worlds: data.worlds,
        });
    }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                {fields.map((field, index) => (
                    <div key={field.id} className="border p-4 rounded-md gap-2 relative flex flex-col justify-stretch">
                        <div className="flex flex-row justify-stretch">
                            <FormField
                                control={form.control}
                                name={`worlds.${index}.name`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Name</FormLabel>
                                        <FormControl>
                                            <Input placeholder="World Name" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`worlds.${index}.flag`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Flag</FormLabel>
                                        <FormControl>
                                            <Input placeholder="World Message" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                        </div>
                        <FormField
                            control={form.control}
                            name={`worlds.${index}.serverMessage`}
                            render={({field}) => (
                                <FormItem>
                                    <FormLabel>Server Message</FormLabel>
                                    <FormControl>
                                        <Input placeholder="Server Message" {...field} />
                                    </FormControl>
                                    <FormMessage/>
                                </FormItem>
                            )}
                        />
                        <div className="flex flex-row justify-stretch">
                            <FormField
                                control={form.control}
                                name={`worlds.${index}.eventMessage`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Event Message</FormLabel>
                                        <FormControl>
                                            <Input placeholder="Event Message" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`worlds.${index}.whyAmIRecommended`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Recommendation Justification</FormLabel>
                                        <FormControl>
                                            <Input placeholder="Recommendation Justification" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                        </div>
                        <Button type="button" className="absolute top-0 right-0" variant="ghost" size="icon"
                                onClick={() => remove(index)}>
                            <X/>
                        </Button>
                    </div>
                ))}
                <div className="flex flex-row gap-2 justify-between">
                    <Button type="button" onClick={() => append({
                        name: "",
                        flag: "",
                        eventMessage: "",
                        serverMessage: "",
                        whyAmIRecommended: "",
                    })}>
                        Add
                    </Button>
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
    );
}
