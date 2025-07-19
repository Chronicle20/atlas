"use client"

import {useEffect, useState} from "react";
import {useFieldArray, useForm, SubmitHandler} from "react-hook-form";
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import {useParams} from "next/navigation";
import {X} from "lucide-react";
import {fetchTemplates, updateTemplate} from "@/lib/templates";
import type {Template} from "@/types/models/template";
import {OptionsField} from "@/components/unknown-options";
import {toast} from "sonner";
import { LoadingSpinner, ErrorDisplay } from "@/components/common";

interface FormValues {
    handlers: {
        opCode: string;
        validator: string;
        handler: string;
        options: unknown;
    }[];
}

export function HandlersForm() {
    const {id} = useParams(); // Get templates ID from URL

    const [template, setTemplate] = useState<Template>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    const form = useForm<FormValues>({
        defaultValues: {
            handlers: [],
        }
    });

    useEffect(() => {
        if (!id) return; // Ensure id is available

        setLoading(true); // Show loading while fetching

        fetchTemplates()
            .then((data) => {
                const template = data.find((t) => String(t.id) === String(id));
                setTemplate(template);

                if (template) {
                    const formValues: FormValues = {
                        handlers: template.attributes.socket.handlers.map(handler => ({
                            opCode: handler.opCode,
                            validator: handler.validator,
                            handler: handler.handler,
                            options: handler.options,
                        }))
                    };
                    form.reset(formValues);
                }
            })
            .catch((err) => {
                setError(err.message);
            })
            .finally(() => setLoading(false));
    }, [id, form]);

    const {fields, append, remove} = useFieldArray({
        control: form.control,
        name: "handlers"
    });

    const onSubmit: SubmitHandler<FormValues> = async (data: FormValues) => {
        updateTemplate(template, {
            socket: {
                handlers: data.handlers,
                writers: template?.attributes.socket.writers || [],
            },
        }).then(() => {
            toast.success("Successfully saved template.");
        });
    }

    if (loading) return <LoadingSpinner />; // Show loading message while fetching data
    if (error) return <ErrorDisplay error={error} />; // Show error message if fetching failed

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
