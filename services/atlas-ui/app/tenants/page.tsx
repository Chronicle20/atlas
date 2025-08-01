"use client"

import { useState } from "react";
import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { getColumns } from "@/app/tenants/columns";
import { tenantsService } from "@/services/api";
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
    const { tenants, refreshTenants } = useTenant();
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
    const [tenantToDelete, setTenantToDelete] = useState<string | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);

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

    const columns = getColumns({ onDelete: openDeleteDialog });

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
