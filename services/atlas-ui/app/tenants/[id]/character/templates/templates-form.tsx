"use client"

import {useEffect} from "react";
import { useForm, useFieldArray } from "react-hook-form";
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";

export function TemplatesForm() {
    const { id } = useParams(); // Get tenants ID from URL
    const {tenants, updateTenant} = useTenant()
    const tenant = tenants.find((t) => t.id === id);

    const form = useForm({
        defaultValues: {
            templates: tenant?.attributes.characters.templates.map(template => ({
                jobIndex: template.jobIndex || 0,
                subJobIndex: template.subJobIndex || 0,
                gender: template.gender || 0,
                mapId: template.mapId || 0,
                faces: template.faces || [],
                hairs: template.hairs || [],
                hairColors: template.hairColors || [],
                skinColors: template.skinColors || [],
                tops: template.tops || [],
                bottoms: template.bottoms || [],
                shoes: template.shoes || [],
                weapons: template.weapons || [],
                items: template.items || [],
                skills: template.skills || [],
            }))
        }
    });

    const { fields, append, remove } = useFieldArray({
        control: form.control,
        name: "templates"
    });

    useEffect(() => {
        form.reset({
            templates: tenant?.attributes.characters.templates.map(template => ({
                jobIndex: template.jobIndex || 0,
                subJobIndex: template.subJobIndex || 0,
                gender: template.gender || 0,
                mapId: template.mapId || 0,
                faces: template.faces || [],
                hairs: template.hairs || [],
                hairColors: template.hairColors || [],
                skinColors: template.skinColors || [],
                tops: template.tops || [],
                bottoms: template.bottoms || [],
                shoes: template.shoes || [],
                weapons: template.weapons || [],
                items: template.items || [],
                skills: template.skills || [],
            }))
        });
    }, [tenant, form.reset, form]);

    const onSubmit = async (data) => {
        await updateTenant(tenant, {
            characters: {
                templates: data.templates,
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
                            name={`templates.${index}.jobIndex`}
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Job Index</FormLabel>
                                    <FormControl>
                                        <Input placeholder="0" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <FormField
                            control={form.control}
                            name={`templates.${index}.subJobIndex`}
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Sub Job Index</FormLabel>
                                    <FormControl>
                                        <Input placeholder="0" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <FormField
                            control={form.control}
                            name={`templates.${index}.gender`}
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Gender</FormLabel>
                                    <FormControl>
                                        <Input placeholder="0" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <FormField
                            control={form.control}
                            name={`templates.${index}.mapId`}
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Starting Map</FormLabel>
                                    <FormControl>
                                        <Input placeholder="0" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <NumbersField templateIndex={index} templateProperty={"faces"} form={form} />
                        <NumbersField templateIndex={index} templateProperty={"hairs"} form={form} />
                        <NumbersField templateIndex={index} templateProperty={"hairColors"} form={form} />
                        <NumbersField templateIndex={index} templateProperty={"skinColors"} form={form} />
                        <NumbersField templateIndex={index} templateProperty={"tops"} form={form} />
                        <NumbersField templateIndex={index} templateProperty={"bottoms"} form={form} />
                        <NumbersField templateIndex={index} templateProperty={"shoes"} form={form} />
                        <NumbersField templateIndex={index} templateProperty={"weapons"} form={form} />
                        <NumbersField templateIndex={index} templateProperty={"items"} form={form} />
                        <NumbersField templateIndex={index} templateProperty={"skills"} form={form} />
                        <Button type="button" variant="destructive" onClick={() => remove(index)}>Remove</Button>
                    </div>
                ))}
                <Button type="submit">Submit</Button>
            </form>
        </Form>
    );
}

// Component to handle numbers for a specific world
function NumbersField({ templateIndex, templateProperty, form }) {
    const { fields: fields, append: add, remove: remove } = useFieldArray({
        control: form.control,
        name: `templates.${templateIndex}.${templateProperty}`
    });

    return (
        <div className="border p-2 rounded-md">
            <FormLabel>{templateProperty ? templateProperty.charAt(0).toUpperCase() + templateProperty.slice(1) : ""}</FormLabel>
            {fields.map((field, index) => (
                <div key={field.id} className="flex items-center gap-2">
                    <FormField
                        control={form.control}
                        name={`templates.${templateIndex}.${templateProperty}.${index}`}
                        render={({ field }) => (
                            <FormItem>
                                <FormControl>
                                    <Input {...field} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <Button type="button" variant="destructive" size="sm" onClick={() => remove(index)}>
                        Remove
                    </Button>
                </div>
            ))}
            <Button type="button" onClick={() => add(0)} size="sm">
                Add
            </Button>
        </div>
    );
}