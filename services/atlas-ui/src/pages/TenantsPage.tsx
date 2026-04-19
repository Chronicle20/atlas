
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { getColumns } from "@/pages/tenants-columns";
import { tenantsService } from "@/services/api";
import type { Tenant } from "@/types/models/tenant";
import { TenantPageSkeleton } from "@/components/common/skeletons/TenantPageSkeleton";
import { tenantNameSchema, type TenantNameFormData } from "@/lib/schemas/tenant.schema";
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
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

export function TenantsPage() {
    const { tenants, loading, refreshTenants } = useTenant();
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
    const [tenantToDelete, setTenantToDelete] = useState<string | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);

    const [renameDialogOpen, setRenameDialogOpen] = useState(false);
    const [tenantToRename, setTenantToRename] = useState<Tenant | null>(null);
    const [isRenaming, setIsRenaming] = useState(false);

    const renameForm = useForm<TenantNameFormData>({
        resolver: zodResolver(tenantNameSchema),
        defaultValues: { name: "" },
        mode: "onChange",
    });

    const watchedName = useWatch({ control: renameForm.control, name: "name" });
    const trimmedWatched = (watchedName ?? "").trim();
    const currentName = tenantToRename?.attributes.name ?? "";
    const isUnchanged = trimmedWatched === currentName.trim();

    // Function to open delete confirmation dialog
    const openDeleteDialog = (id: string) => {
        setTenantToDelete(id);
        setDeleteDialogOpen(true);
    };

    // Function to handle tenant deletion
    const handleDeleteTenant = async () => {
        if (!tenantToDelete) return;

        try {
            setIsDeleting(true);
            await tenantsService.deleteTenant(tenantToDelete);

            // Refresh tenant data using the context function
            await refreshTenants();
        } catch (err: unknown) {
            console.error("Failed to delete tenant:", err);
        } finally {
            setIsDeleting(false);
            setDeleteDialogOpen(false);
            setTenantToDelete(null);
        }
    };

    const openRenameDialog = (id: string) => {
        const tenant = tenants.find((t) => t.id === id);
        if (!tenant) return;
        setTenantToRename(tenant);
        renameForm.reset({ name: tenant.attributes.name });
        setRenameDialogOpen(true);
    };

    const handleRenameDialogOpenChange = (open: boolean) => {
        setRenameDialogOpen(open);
        if (!open) {
            setTenantToRename(null);
            renameForm.reset({ name: "" });
        }
    };

    const handleRenameSubmit = async (data: TenantNameFormData) => {
        if (!tenantToRename) return;

        try {
            setIsRenaming(true);
            await tenantsService.updateTenant(tenantToRename, { name: data.name });
            await refreshTenants();
            handleRenameDialogOpenChange(false);
            toast.success("Tenant renamed");
        } catch (err: unknown) {
            console.error("Failed to rename tenant:", err);
            toast.error("Failed to rename tenant");
        } finally {
            setIsRenaming(false);
        }
    };

    const columns = getColumns({ onDelete: openDeleteDialog, onRename: openRenameDialog });

    if (loading) {
        return <TenantPageSkeleton />;
    }

    const submitDisabled =
        !renameForm.formState.isValid || isRenaming || isUnchanged;

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Tenants</h2>
                </div>
            </div>
            <div className="mt-4">
                <DataTableWrapper
                    columns={columns}
                    data={tenants}
                    emptyState={{
                        title: "No tenants found",
                        description: "There are no tenants to display at this time."
                    }}
                />
            </div>

            {/* Rename Dialog */}
            <Dialog open={renameDialogOpen} onOpenChange={handleRenameDialogOpenChange}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Rename Tenant</DialogTitle>
                        <DialogDescription>
                            Change this tenant's display name. Other tenant settings are
                            unaffected.
                        </DialogDescription>
                    </DialogHeader>
                    <Form {...renameForm}>
                        <form
                            onSubmit={renameForm.handleSubmit(handleRenameSubmit)}
                            className="space-y-4"
                        >
                            <FormField
                                control={renameForm.control}
                                name="name"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Name</FormLabel>
                                        <FormControl>
                                            <Input placeholder="Tenant name" {...field} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <DialogFooter>
                                <Button
                                    type="button"
                                    variant="outline"
                                    onClick={() => handleRenameDialogOpenChange(false)}
                                >
                                    Cancel
                                </Button>
                                <Button type="submit" disabled={submitDisabled}>
                                    {isRenaming ? "Saving\u2026" : "Save"}
                                </Button>
                            </DialogFooter>
                        </form>
                    </Form>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Are you sure?</AlertDialogTitle>
                        <AlertDialogDescription>
                            This action cannot be undone. This will permanently delete the tenant.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={handleDeleteTenant}
                            disabled={isDeleting}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                            {isDeleting ? "Deleting..." : "Delete"}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
