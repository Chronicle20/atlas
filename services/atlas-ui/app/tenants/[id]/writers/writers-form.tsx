"use client"

import { useEffect } from "react";
import { useForm, useFieldArray } from "react-hook-form";
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";

export function WritersForm() {
    const { id } = useParams(); // Get tenants ID from URL
    const {tenants} = useTenant()
    const tenant = tenants.find((t) => t.id === id);

    const form = useForm({
        defaultValues: {
            writers: tenant?.attributes.socket.writers.map(writer => ({
                opCode: writer.opCode || "",
                writer: writer.writer || "",
            }))
        }
    });

    const { fields, append, remove } = useFieldArray({
        control: form.control,
        name: "writers"
    });

    // Reset form values when `writers` data changes
    useEffect(() => {
        form.reset({
            writers: tenant?.attributes.socket.writers.map(writer => ({
                opCode: writer.opCode || "",
                writer: writer.writer || "",
            }))
        });
    }, [tenant, form.reset, form]);

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit((data) => console.log(data))} className="space-y-4">
                {fields.map((field, index) => (
                    <div key={field.id} className="border p-4 rounded-md space-y-2">
                        <FormField
                            control={form.control}
                            name={`writers.${index}.opCode`}
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
                            name={`writers.${index}.writer`}
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Writer</FormLabel>
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
                <Button type="button" onClick={() => append({ opCode: "", writer: "", })}>
                    Add Writer
                </Button>
                <Button type="submit">Submit</Button>
            </form>
        </Form>
    );
}
