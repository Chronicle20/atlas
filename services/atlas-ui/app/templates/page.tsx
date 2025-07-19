"use client"

import {DataTable} from "@/components/data-table";
import {getColumns} from "@/app/templates/columns";
import {useEffect, useState} from "react";
import {fetchTemplates, deleteTemplate, cloneTemplate, createTemplate} from "@/lib/templates";
import type {Template} from "@/types/models/template";
import {createTenantConfiguration, createTenantFromTemplate} from "@/lib/tenants";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import {createErrorFromUnknown} from "@/types/api/errors";

// Form schema for clone template
const cloneTemplateFormSchema = z.object({
    region: z
        .string()
        .min(3, {
            message: "Region must be 3 characters.",
        })
        .max(3, {
            message: "Region must be 3 characters.",
        }),
    majorVersion: z.number(),
    minorVersion: z.number(),
});

type CloneTemplateFormValues = z.infer<typeof cloneTemplateFormSchema>;


import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export default function Page() {
    const [templates, setTemplates] = useState<Template[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
    const [templateToDelete, setTemplateToDelete] = useState<string | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);

    // Clone template state
    const [cloneDialogOpen, setCloneDialogOpen] = useState(false);
    const [templateToClone, setTemplateToClone] = useState<Template | null>(null);
    const [isCloning, setIsCloning] = useState(false);

    // Create tenant from template state
    const [createTenantDialogOpen, setCreateTenantDialogOpen] = useState(false);
    const [templateForTenant, setTemplateForTenant] = useState<Template | null>(null);
    const [isCreatingTenant, setIsCreatingTenant] = useState(false);

    // Clone template form
    const form = useForm<CloneTemplateFormValues>({
        resolver: zodResolver(cloneTemplateFormSchema),
        defaultValues: {
            region: "",
            majorVersion: 0,
            minorVersion: 0,
        },
        mode: "onChange",
    });


    const fetchDataAgain = () => {
        setLoading(true)
        fetchTemplates()
            .then((data) => setTemplates(data))
            .catch((err: unknown) => {
                const errorInfo = createErrorFromUnknown(err, "Failed to fetch templates");
                setError(errorInfo.message);
            })
            .finally(() => setLoading(false))
    }

    useEffect(() => {
        fetchDataAgain()
    }, [])

    // Function to open delete confirmation dialog
    const openDeleteDialog = (id: string) => {
        setTemplateToDelete(id);
        setDeleteDialogOpen(true);
    };

    // Function to handle template deletion
    const handleDeleteTemplate = async () => {
        if (!templateToDelete) return;

        try {
            setIsDeleting(true);
            await deleteTemplate(templateToDelete);

            // Refresh template data
            fetchDataAgain();
        } catch (err: unknown) {
            console.error("Failed to delete template:", err);
        } finally {
            setIsDeleting(false);
            setDeleteDialogOpen(false);
            setTemplateToDelete(null);
        }
    };

    // Function to open clone dialog
    const openCloneDialog = (id: string) => {
        const template = templates.find(t => t.id === id);
        if (template) {
            setTemplateToClone(template);
            setCloneDialogOpen(true);

            // Reset form with empty values for region, majorVersion, and minorVersion
            form.reset({
                region: "",
                majorVersion: 0,
                minorVersion: 0,
            });
        }
    };

    // Function to handle template cloning
    const handleCloneTemplate = async (data: CloneTemplateFormValues) => {
        if (!templateToClone) return;

        try {
            setIsCloning(true);

            // Clone the template and update with form values
            const clonedAttributes = cloneTemplate(templateToClone);
            clonedAttributes.region = data.region;
            clonedAttributes.majorVersion = data.majorVersion;
            clonedAttributes.minorVersion = data.minorVersion;

            // Create the new template
            const newTemplate = await createTemplate(clonedAttributes);

            // Show success message
            toast.success("Template cloned successfully");

            // Close the dialog
            setCloneDialogOpen(false);
            setTemplateToClone(null);

            // Navigate to the new template
            console.log(`Navigating to: /templates/${newTemplate.id}/properties`);

            // Use window.location.replace for a more forceful navigation
            window.location.replace(`/templates/${newTemplate.id}/properties`);

            // The code below this point may not execute due to the page navigation
        } catch (err: unknown) {
            console.error("Failed to clone template:", err);
            toast.error("Failed to clone template");
        } finally {
            setIsCloning(false);
        }
    };

    // Function to open create tenant dialog
    const openCreateTenantDialog = (id: string) => {
        const template = templates.find(t => t.id === id);
        if (template) {
            setTemplateForTenant(template);
            setCreateTenantDialogOpen(true);
        }
    };

    // Function to handle tenant creation from template
    const handleCreateTenantFromTemplate = async () => {
        if (!templateForTenant) return;

        try {
            setIsCreatingTenant(true);

            // Create tenant attributes from template
            const tenantAttributes = createTenantFromTemplate(templateForTenant);

            // Create the new tenant configuration
            const newTenant = await createTenantConfiguration(tenantAttributes);

            // Show success message
            toast.success("Tenant created successfully");

            // Close the dialog
            setCreateTenantDialogOpen(false);
            setTemplateForTenant(null);

            // Navigate to the new tenant
            console.log(`Navigating to: /tenants/${newTenant.id}/properties`);

            // Use window.location.replace for a more forceful navigation
            window.location.replace(`/tenants/${newTenant.id}/properties`);

            // The code below this point may not execute due to the page navigation
        } catch (err: unknown) {
            console.error("Failed to create tenant:", err);
            toast.error("Failed to create tenant");
        } finally {
            setIsCreatingTenant(false);
        }
    };

    const columns = getColumns({ 
        onDelete: openDeleteDialog,
        onClone: openCloneDialog,
        onCreateTenant: openCreateTenantDialog
    });

    if (loading) return <div>Loading...</div>; // Show loading message while fetching data
    if (error) return <div>Error: {error}</div>; // Show error message if fetching failed

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Templates</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTable columns={columns} data={templates} onRefresh={fetchDataAgain}/>
            </div>

            {/* Delete Confirmation Dialog */}
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Are you sure?</AlertDialogTitle>
                        <AlertDialogDescription>
                            This action cannot be undone. This will permanently delete the template.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction 
                            onClick={handleDeleteTemplate}
                            disabled={isDeleting}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                            {isDeleting ? "Deleting..." : "Delete"}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>

            {/* Clone Template Dialog */}
            <Dialog open={cloneDialogOpen} onOpenChange={setCloneDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Clone Template</DialogTitle>
                        <DialogDescription>
                            Create a new template based on the selected template. Please provide the required information.
                        </DialogDescription>
                    </DialogHeader>

                    <Form {...form}>
                        <form onSubmit={form.handleSubmit(handleCloneTemplate)} className="space-y-4">
                            <FormField
                                control={form.control}
                                name="region"
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Region</FormLabel>
                                        <FormControl>
                                            <Input placeholder="Enter region (3 characters)" {...field} />
                                        </FormControl>
                                        <FormDescription>
                                            The MapleStory region (3 characters).
                                        </FormDescription>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="majorVersion"
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Major Version</FormLabel>
                                        <FormControl>
                                            <Input 
                                                type="number" 
                                                placeholder="Enter major version" 
                                                {...field}
                                                onChange={(e) => field.onChange(parseInt(e.target.value) || 0)}
                                            />
                                        </FormControl>
                                        <FormDescription>
                                            The MapleStory major version.
                                        </FormDescription>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="minorVersion"
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Minor Version</FormLabel>
                                        <FormControl>
                                            <Input 
                                                type="number" 
                                                placeholder="Enter minor version" 
                                                {...field}
                                                onChange={(e) => field.onChange(parseInt(e.target.value) || 0)}
                                            />
                                        </FormControl>
                                        <FormDescription>
                                            The MapleStory minor version.
                                        </FormDescription>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <DialogFooter>
                                <Button 
                                    type="button" 
                                    variant="outline" 
                                    onClick={() => setCloneDialogOpen(false)}
                                >
                                    Cancel
                                </Button>
                                <Button 
                                    type="submit" 
                                    disabled={isCloning}
                                >
                                    {isCloning ? "Cloning..." : "Clone Template"}
                                </Button>
                            </DialogFooter>
                        </form>
                    </Form>
                </DialogContent>
            </Dialog>

            {/* Create Tenant from Template Dialog */}
            <Dialog open={createTenantDialogOpen} onOpenChange={setCreateTenantDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Create Tenant from Template</DialogTitle>
                        <DialogDescription>
                            Create a new tenant based on the selected template. All information from the template will be used.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button 
                            type="button" 
                            variant="outline" 
                            onClick={() => setCreateTenantDialogOpen(false)}
                        >
                            Cancel
                        </Button>
                        <Button 
                            onClick={handleCreateTenantFromTemplate} 
                            disabled={isCreatingTenant}
                        >
                            {isCreatingTenant ? "Creating..." : "Create Tenant"}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
