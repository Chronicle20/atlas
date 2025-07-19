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
import {toast} from "sonner";

interface FormValues {
    worlds: {
        name: string;
        flag: string;
        eventMessage: string;
        serverMessage: string;
        whyAmIRecommended: string;
    }[];
}

export function WorldsForm() {
    const {id} = useParams(); // Get templates ID from URL

    const [template, setTemplate] = useState<Template>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    const form = useForm<FormValues>({
        defaultValues: {
            worlds: []
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
                        worlds: template.attributes.worlds.map(world => ({
                            name: world.name,
                            flag: world.flag,
                            eventMessage: world.eventMessage,
                            serverMessage: world.serverMessage,
                            whyAmIRecommended: world.whyAmIRecommended,
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
        name: "worlds"
    });

    const onSubmit: SubmitHandler<FormValues> = async (data: FormValues) => {
        updateTemplate(template, {
            worlds: data.worlds,
        }).then(() => {
            toast.success("Successfully saved template.");
        });
    }

    if (loading) return <div>Loading...</div>; // Show loading message while fetching data
    if (error) return <div>Error: {error}</div>; // Show error message if fetching failed

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
