"use client"

import {DataTable} from "@/components/data-table";
import {getColumns} from "@/app/templates/columns";
import {useEffect, useState} from "react";
import {fetchTemplates, Template, deleteTemplate} from "@/lib/templates";
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

    const fetchDataAgain = () => {
        setLoading(true)
        fetchTemplates()
            .then((data) => setTemplates(data))
            .catch((err) => setError(err.message))
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
        } catch (err) {
            console.error("Failed to delete template:", err);
        } finally {
            setIsDeleting(false);
            setDeleteDialogOpen(false);
            setTemplateToDelete(null);
        }
    };

    const columns = getColumns({ onDelete: openDeleteDialog });

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
        </div>
    );
}
