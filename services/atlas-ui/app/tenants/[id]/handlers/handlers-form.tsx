"use client"

import {useEffect} from "react";
import { useForm, useFieldArray } from "react-hook-form";
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";

export function HandlersForm() {
    const { id } = useParams(); // Get tenants ID from URL
    const {tenants, updateTenant} = useTenant()
    const tenant = tenants.find((t) => t.id === id);

    const form = useForm({
        defaultValues: {
            handlers: tenant?.attributes.socket.handlers.map(handler => ({
                opCode: handler.opCode || "",
                validator: handler.validator || "",
                handler: handler.handler || "",
            }))
        }
    });

    const { fields, append, remove } = useFieldArray({
        control: form.control,
        name: "handlers"
    });

    // Reset form values when `handlers` data changes
    useEffect(() => {
        form.reset({
            handlers: tenant?.attributes.socket.handlers.map(handler => ({
                opCode: handler.opCode || "",
                validator: handler.validator || "",
                handler: handler.handler || "",
            }))
        });
    }, [tenant, form.reset, form]);

    const onSubmit = async (data) => {
        await updateTenant(tenant, {
            socket: {
                handlers: data.handlers,
                writers: tenant?.attributes.socket.writers,
            },
        });
    }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                {fields.map((field, index) => (
                    <div key={field.id} className="border p-4 rounded-md space-y-2">
                        <FormField
                            control={form.control}
                            name={`handlers.${index}.opCode`}
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Operation Code</FormLabel>
                                    <FormControl>
                                        <Input placeholder="0x00" {...field} />
                                    </FormControl>
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
                                    <FormControl>
                                        <Input {...field} />
                                    </FormControl>
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
                                    <FormControl>
                                        <Input {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <Button type="button" variant="destructive" onClick={() => remove(index)}>Remove</Button>
                    </div>
                ))}
                <Button type="button" onClick={() => append({ opCode: "", validator: "", handler: "", })}>
                    Add Handler
                </Button>
                <Button type="submit">Submit</Button>
            </form>
        </Form>
    );
}
