
import { useEffect } from "react";
import { useFieldArray, useForm, type SubmitHandler } from "react-hook-form";
import { Form } from "@/components/ui/form";
import { Button } from "@/components/ui/button";
import { useParams } from "react-router-dom";
import { X } from "lucide-react";
import { useTemplate, useUpdateTemplate } from "@/lib/hooks/api/useTemplates";
import { toast } from "sonner";
import { LoadingSpinner, ErrorDisplay, FormField } from "@/components/common";

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
    const { id } = useParams();
    const templateQuery = useTemplate(String(id ?? ""));
    const updateTemplate = useUpdateTemplate();

    const template = templateQuery.data ?? null;
    const loading = templateQuery.isLoading;
    const error = templateQuery.error?.message ?? null;

    const form = useForm<FormValues>({ defaultValues: { worlds: [] } });

    useEffect(() => {
        if (template) {
            form.reset({
                worlds: template.attributes.worlds.map(world => ({
                    name: world.name,
                    flag: world.flag,
                    eventMessage: world.eventMessage,
                    serverMessage: world.serverMessage,
                    whyAmIRecommended: world.whyAmIRecommended,
                })),
            });
        }
    }, [template, form]);

    const { fields, append, remove } = useFieldArray({ control: form.control, name: "worlds" });

    const onSubmit: SubmitHandler<FormValues> = (data) => {
        if (!template) return;
        updateTemplate.mutate(
            { id: template.id, updates: { worlds: data.worlds } },
            { onSuccess: () => toast.success("Successfully saved template.") },
        );
    };

    if (loading) return <LoadingSpinner />;
    if (error) return <ErrorDisplay error={error} />;

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                {fields.map((field, index) => (
                    <div key={field.id} className="border p-4 rounded-md gap-2 relative flex flex-col justify-stretch">
                        <div className="flex flex-row justify-stretch">
                            <FormField
                                control={form.control}
                                name={`worlds.${index}.name`}
                                label="Name"
                                type="text"
                                placeholder="World Name"
                            />
                            <FormField
                                control={form.control}
                                name={`worlds.${index}.flag`}
                                label="Flag"
                                type="text"
                                placeholder="World Message"
                            />
                        </div>
                        <FormField
                            control={form.control}
                            name={`worlds.${index}.serverMessage`}
                            label="Server Message"
                            type="text"
                            placeholder="Server Message"
                        />
                        <div className="flex flex-row justify-stretch">
                            <FormField
                                control={form.control}
                                name={`worlds.${index}.eventMessage`}
                                label="Event Message"
                                type="text"
                                placeholder="Event Message"
                            />
                            <FormField
                                control={form.control}
                                name={`worlds.${index}.whyAmIRecommended`}
                                label="Recommendation Justification"
                                type="text"
                                placeholder="Recommendation Justification"
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
