"use client"

import { Button } from "@/components/ui/button"
import {Form, FormControl, FormDescription, FormField as ShadcnFormField, FormItem, FormLabel } from "@/components/ui/form"
import {zodResolver} from "@hookform/resolvers/zod";
import {useForm} from "react-hook-form";
import { z } from "zod"
import {useParams} from "next/navigation";
import { Switch } from "@/components/ui/switch";
import {useEffect, useState} from "react";
import {fetchTemplates, updateTemplate} from "@/lib/templates";
import type {Template} from "@/types/models/template";
import {toast} from "sonner";
import { LoadingSpinner, ErrorDisplay, FormField } from "@/components/common";

const propertiesFormSchema = z.object({
    region: z
        .string()
        .min(3, {
            message: "Region must be 3 characters.",
        })
        .max(3, {
            message: "Region must be 3 characters.",
        }),
    major: z.number(),
    minor: z.number(),
    usesPin: z.boolean(),
})

type PropertiesFormValues = z.infer<typeof propertiesFormSchema>

export function PropertiesForm() {
    const { id } = useParams(); // Get templates ID from URL

    const [template, setTemplate] = useState<Template>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    const form = useForm<PropertiesFormValues>({
        resolver: zodResolver(propertiesFormSchema),
        defaultValues: {
            region: "",
            major: 0,
            minor: 0,
            usesPin: false,
        },
        mode: "onChange",
    });

    useEffect(() => {
        if (!id) return; // Ensure id is available

        setLoading(true); // Show loading while fetching

        fetchTemplates()
            .then((data) => {
                const template = data.find((t) => String(t.id) === String(id));
                setTemplate(template);

                form.reset({
                    region: template?.attributes.region || "",
                    major: template?.attributes.majorVersion || 0,
                    minor: template?.attributes.minorVersion || 0,
                    usesPin: template?.attributes.usesPin || false,
                });
            })
            .catch((err) => {
                setError(err.message);
            })
            .finally(() => setLoading(false));
    }, [id, form]);

    const onSubmit = async (data : PropertiesFormValues) => {
        updateTemplate(template, {
            region: data.region,
            majorVersion: data.major,
            minorVersion: data.minor,
            usesPin: data.usesPin,
        }).then(() => {
            toast.success("Successfully saved template.");
        });
        form.reset({
            region: template?.attributes.region || "",
            major: template?.attributes.majorVersion || 0,
            minor: template?.attributes.minorVersion || 0,
            usesPin: template?.attributes.usesPin || false,
        });
    }

    if (loading) return <LoadingSpinner />; // Show loading message while fetching data
    if (error) return <ErrorDisplay error={error} />; // Show error message if fetching failed

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="region"
                    label="Region"
                    type="text"
                    placeholder={template?.attributes.region || "Enter region"}
                    description="The MapleStory region."
                />
                <FormField
                    control={form.control}
                    name="major"
                    label="Major Version"
                    type="number"
                    placeholder={String(template?.attributes.majorVersion || 0)}
                    description="The MapleStory major version."
                />
                <FormField
                    control={form.control}
                    name="minor"
                    label="Minor Version"
                    type="number"
                    placeholder={String(template?.attributes.minorVersion || 0)}
                    description="The MapleStory minor version."
                />
                <ShadcnFormField
                    control={form.control}
                    name="usesPin"
                    render={({field}) => (
                        <FormItem
                            className="flex flex-row items-center justify-between rounded-lg border p-3 shadow-xs">
                            <div className="space-y-0.5">
                                <FormLabel>Uses PIN system</FormLabel>
                                <FormDescription>
                                    Receive emails about new products, features, and more.
                                </FormDescription>
                            </div>
                            <FormControl>
                                <Switch
                                    checked={field.value}
                                    onCheckedChange={field.onChange}
                                />
                            </FormControl>
                        </FormItem>
                    )}
                />
                <div className="flex flex-row gap-2 justify-end">
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
    );
}