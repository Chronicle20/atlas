"use client"

import {useEffect, useState} from "react";
import {useForm, useFieldArray} from "react-hook-form";
import {Form, FormField, FormItem, FormLabel, FormControl, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";
import {X, Plus} from "lucide-react"
import {Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog";

export function TemplatesForm() {
    const {id} = useParams(); // Get tenants ID from URL
    const {tenants, updateTenant} = useTenant()
    const tenant = tenants.find((t) => t.id === id);

    interface FormValues {
        templates: {
            jobIndex: number;
            subJobIndex: number;
            gender: number;
            mapId: number;
            faces: number[];
            hairs: number[];
            hairColors: number[];
            skinColors: number[];
            tops: number[];
            bottoms: number[];
            shoes: number[];
            weapons: number[];
            items: number[];
            skills: number[];
        }[];
    }

    const form = useForm<FormValues>({
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

    const {fields, remove} = useFieldArray({
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

    const onSubmit = async (data: FormValues) => {
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
                    <div key={field.id}
                         className="border p-4 rounded-md gap-2 relative flex flex-col justify-stretch">
                        <div className="flex flex-row justify-stretch">
                            <FormField
                                control={form.control}
                                name={`templates.${index}.jobIndex`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Job Index</FormLabel>
                                        <FormControl>
                                            <Input placeholder="0" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`templates.${index}.subJobIndex`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Sub Job Index</FormLabel>
                                        <FormControl>
                                            <Input placeholder="0" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                        </div>
                        <FormField
                            control={form.control}
                            name={`templates.${index}.gender`}
                            render={({field}) => (
                                <FormItem>
                                    <FormLabel>Gender</FormLabel>
                                    <FormControl>
                                        <Input placeholder="0" {...field} />
                                    </FormControl>
                                    <FormMessage/>
                                </FormItem>
                            )}
                        />
                        <FormField
                            control={form.control}
                            name={`templates.${index}.mapId`}
                            render={({field}) => (
                                <FormItem>
                                    <FormLabel>Starting Map</FormLabel>
                                    <FormControl>
                                        <Input placeholder="0" {...field} />
                                    </FormControl>
                                    <FormMessage/>
                                </FormItem>
                            )}
                        />
                        <NumbersField templateIndex={index} templateProperty={"faces"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"hairs"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"hairColors"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"skinColors"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"tops"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"bottoms"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"shoes"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"weapons"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"items"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"skills"} form={form}/>
                        <Button type="button" className="absolute top-0 right-0" variant="ghost" size="icon"
                                onClick={() => remove(index)}>
                            <X/>
                        </Button>
                    </div>
                ))}
                <div className="flex flex-row gap-2 justify-end">
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
    );
}

interface NumbersFieldProps {
    templateIndex: number
    templateProperty: string
    form: any
}

// Component to handle numbers for a specific world
function NumbersField({templateIndex, templateProperty, form}: NumbersFieldProps) {
    const {fields, append, remove} = useFieldArray({
        control: form.control,
        name: `templates.${templateIndex}.${templateProperty}`
    });

    const values = form.watch(`templates.${templateIndex}.${templateProperty}`);

    const [isDialogOpen, setDialogOpen] = useState(false);
    const [newValue, setNewValue] = useState("");

    const handleAdd = () => {
        if (newValue.trim() !== "") {
            append(newValue);
            setNewValue("");
            setDialogOpen(false);
        }
    };

    return (
        <div className="border p-2 rounded-md">
            <FormLabel>{templateProperty ? templateProperty.charAt(0).toUpperCase() + templateProperty.slice(1) : ""}</FormLabel>
            <div className="flex flex-row p-2 justify-start gap-2">
                {fields.map((field, index) => (
                    <div key={field.id} className="flex items-center">
                        <Button type="button" variant="outline" onClick={() => remove(index)}>
                            {values[index]}
                            <X/>
                        </Button>
                    </div>
                ))}
                <Button type="button" variant="outline" onClick={() => setDialogOpen(true)} size="icon">
                    <Plus/>
                </Button>
            </div>

            <Dialog open={isDialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Add a New Value</DialogTitle>
                    </DialogHeader>
                    <Input
                        value={newValue}
                        onChange={(e) => setNewValue(e.target.value)}
                        placeholder="Enter value..."
                    />
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDialogOpen(false)}>Cancel</Button>
                        <Button onClick={handleAdd}>Add</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}