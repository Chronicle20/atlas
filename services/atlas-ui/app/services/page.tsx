"use client";

import { useState } from "react";
import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { PageLoader } from "@/components/common/PageLoader";
import {
  CreateServiceDialog,
  DeleteServiceDialog,
} from "@/components/features/services";

import { useServices, useInvalidateServices } from "@/lib/hooks/api/useServices";
import { getColumns } from "./columns";
import type { Service } from "@/types/models/service";

export default function ServicesPage() {
  const { data: services, isLoading, error, refetch } = useServices();
  const { invalidateAll } = useInvalidateServices();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [serviceToDelete, setServiceToDelete] = useState<Service | null>(null);

  const handleDeleteClick = (service: Service) => {
    setServiceToDelete(service);
    setDeleteDialogOpen(true);
  };

  const handleCreateSuccess = () => {
    invalidateAll();
  };

  const handleDeleteSuccess = () => {
    invalidateAll();
  };

  const columns = getColumns({ onDelete: handleDeleteClick });

  if (isLoading) {
    return <PageLoader />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Services</h2>
          <p className="text-muted-foreground">
            Manage service configurations for login, channel, and drops services.
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Service
        </Button>
      </div>

      <DataTableWrapper
        columns={columns}
        data={services || []}
        error={error}
        onRefresh={refetch}
        emptyState={{
          title: "No services configured",
          description:
            "Get started by creating a service configuration for your deployment.",
          action: {
            label: "Create Service",
            onClick: () => setCreateDialogOpen(true),
          },
        }}
      />

      <CreateServiceDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSuccess={handleCreateSuccess}
      />

      <DeleteServiceDialog
        service={serviceToDelete}
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        onSuccess={handleDeleteSuccess}
      />
    </div>
  );
}
