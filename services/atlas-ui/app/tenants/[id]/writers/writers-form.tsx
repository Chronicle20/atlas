"use client"

import {useEffect} from "react";
import { useForm, useFieldArray } from "react-hook-form";
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";
import {X} from "lucide-react";

export function WritersForm() {
    const { id } = useParams(); // Get tenants ID from URL
    const {tenants, updateTenant} = useTenant()
    let tenant = tenants.find((t) => t.id === id);

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

    const onSubmit = async (data) => {
        tenant = await updateTenant(tenant, {
            socket: {
                handlers: tenant?.attributes.socket.handlers,
                writers: data.writers,
            },
        });
        form.reset({
            writers: tenant?.attributes.socket.writers.map(writer => ({
                opCode: writer.opCode || "",
                writer: writer.writer || "",
            }))
        });
    }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                {fields.map((field, index) => (
                    <div key={field.id} className="border p-4 rounded-md gap-2 relative flex flex-row justify-stretch">
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

                        <Button type="button" className="absolute top-0 right-0" variant="ghost" size="icon"
                                onClick={() => remove(index)}>
                            <X/>
                        </Button>
                    </div>
                ))}
                <div className="flex flex-row gap-2 justify-between">
                    <Button type="button" onClick={() => append({opCode: "", writer: "",})}>
                        Add
                    </Button>
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
);
}
