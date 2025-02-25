"use client"

import {useEffect, useState} from "react";
import {useFieldArray, useForm} from "react-hook-form";
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import {useParams} from "next/navigation";
import {X} from "lucide-react";
import {fetchTemplates, Template, updateTemplate} from "@/lib/templates";

export function WritersForm() {
    const {id} = useParams(); // Get templates ID from URL

    const [template, setTemplate] = useState<Template>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        if (!id) return; // Ensure id is available

        setLoading(true); // Show loading while fetching

        fetchTemplates()
            .then((data) => {
                const template = data.find((t) => String(t.id) === String(id));
                setTemplate(template);

                form.reset({
                    writers: template?.attributes.socket.writers.map(writer => ({
                        opCode: writer.opCode || "",
                        writer: writer.writer || "",
                    })),
                });
            })
            .catch((err) => {
                setError(err.message);
            })
            .finally(() => setLoading(false));
    }, [id]);

    interface FormValues {
        writers: {
            opCode: string;
            writer: string;
        }[];
    }

    const form = useForm<FormValues>({
        defaultValues: {
            writers: [],
        }
    });

    const {fields, append, remove} = useFieldArray({
        control: form.control,
        name: "writers"
    });

    const onSubmit = async (data: FormValues) => {
        await updateTemplate(template, {
            socket: {
                handlers: template?.attributes.socket.handlers,
                writers: data.writers,
            },
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
