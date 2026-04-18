
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormDescription, FormField as ShadcnFormField, FormItem, FormLabel } from "@/components/ui/form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useParams } from "react-router-dom";
import { Switch } from "@/components/ui/switch";
import { useEffect } from "react";
import { useTemplate, useUpdateTemplate } from "@/lib/hooks/api/useTemplates";
import { toast } from "sonner";
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
    const { id } = useParams();
    const templateQuery = useTemplate(String(id ?? ""));
    const updateTemplate = useUpdateTemplate();

    const template = templateQuery.data ?? null;
    const loading = templateQuery.isLoading;
    const error = templateQuery.error?.message ?? null;

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
        if (template) {
            form.reset({
                region: template.attributes.region || "",
                major: template.attributes.majorVersion || 0,
                minor: template.attributes.minorVersion || 0,
                usesPin: template.attributes.usesPin || false,
            });
        }
    }, [template, form]);

    const onSubmit = (data: PropertiesFormValues) => {
        if (!template) return;
        updateTemplate.mutate(
            {
                id: template.id,
                updates: {
                    region: data.region,
                    majorVersion: data.major,
                    minorVersion: data.minor,
                    usesPin: data.usesPin,
                },
            },
            { onSuccess: () => toast.success("Successfully saved template.") },
        );
    };

    if (loading) return <LoadingSpinner />;
    if (error) return <ErrorDisplay error={error} />;

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