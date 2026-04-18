
import {useEffect} from "react";
import {useFieldArray, useForm, SubmitHandler} from "react-hook-form";
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import { useParams } from "react-router-dom";
import {X} from "lucide-react";
import { useTemplate, useUpdateTemplate } from "@/lib/hooks/api/useTemplates";
import {OptionsField} from "@/components/unknown-options";
import {toast} from "sonner";
import { LoadingSpinner, ErrorDisplay } from "@/components/common";

interface FormValues {
    writers: {
        opCode: string;
        writer: string;
        options: unknown;
    }[];
}

export function WritersForm() {
    const { id } = useParams();
    const templateQuery = useTemplate(String(id ?? ""));
    const updateTemplate = useUpdateTemplate();

    const template = templateQuery.data ?? null;
    const loading = templateQuery.isLoading;
    const error = templateQuery.error?.message ?? null;

    const form = useForm<FormValues>({ defaultValues: { writers: [] } });

    useEffect(() => {
        if (template) {
            form.reset({
                writers: template.attributes.socket.writers.map(writer => ({
                    opCode: writer.opCode,
                    writer: writer.writer,
                    options: writer.options,
                })),
            });
        }
    }, [template, form]);

    const { fields, append, remove } = useFieldArray({ control: form.control, name: "writers" });

    const onSubmit: SubmitHandler<FormValues> = (data) => {
        if (!template) return;
        updateTemplate.mutate(
            {
                id: template.id,
                updates: {
                    socket: {
                        handlers: template.attributes.socket.handlers || [],
                        writers: data.writers,
                    },
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
                {fields.map((field, index) => (
                    <div key={field.id} className="border p-4 rounded-md gap-2 relative flex flex-col justify-stretch">
                        <div className="gap-2 flex flex-row justify-stretch">
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
                        </div>
                        <OptionsField form={form} path={`writers.${index}.options`}/>
                        <Button type="button" className="absolute top-0 right-0" variant="ghost" size="icon"
                                onClick={() => remove(index)}>
                            <X/>
                        </Button>
                    </div>
                ))}
                <div className="flex flex-row gap-2 justify-between">
                    <Button type="button" onClick={() => append({opCode: "", writer: "", options: null})}>
                        Add
                    </Button>
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
    );
}
