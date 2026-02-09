"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Pencil, Trash2, Save, X } from "lucide-react";
import { toast } from "sonner";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";

import {
  ServiceTypeBadge,
  DeleteServiceDialog,
  TaskConfigForm,
  LoginTenantConfig,
  ChannelTenantConfig,
} from "@/components/features/services";

import {
  useService,
  useUpdateService,
  useInvalidateServices,
} from "@/lib/hooks/api/useServices";
import type {
  Service,
  TaskConfig,
  LoginTenant,
  ChannelTenant,
  UpdateServiceInput,
} from "@/types/models/service";
import {
  isLoginService,
  isChannelService,
  isDropsService,
  getServiceTypeDisplayName,
} from "@/types/models/service";

export default function ServiceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { data: service, isLoading, error, refetch } = useService(id);
  const updateService = useUpdateService();
  const { invalidateAll } = useInvalidateServices();

  const [isEditing, setIsEditing] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  // Edit state
  const [editTasks, setEditTasks] = useState<TaskConfig[]>([]);
  const [editLoginTenants, setEditLoginTenants] = useState<LoginTenant[]>([]);
  const [editChannelTenants, setEditChannelTenants] = useState<ChannelTenant[]>([]);

  const startEditing = () => {
    if (!service) return;

    setEditTasks([...service.attributes.tasks]);

    if (isLoginService(service)) {
      setEditLoginTenants([...service.attributes.tenants]);
    } else if (isChannelService(service)) {
      setEditChannelTenants([...service.attributes.tenants]);
    }

    setIsEditing(true);
  };

  const cancelEditing = () => {
    setIsEditing(false);
    setEditTasks([]);
    setEditLoginTenants([]);
    setEditChannelTenants([]);
  };

  const handleSave = async () => {
    if (!service) return;

    const input: UpdateServiceInput = {
      type: service.attributes.type,
      tasks: editTasks,
    };

    if (isLoginService(service)) {
      input.tenants = editLoginTenants;
    } else if (isChannelService(service)) {
      input.tenants = editChannelTenants;
    }

    try {
      await updateService.mutateAsync({ id: service.id, input });
      toast.success("Service updated successfully");
      setIsEditing(false);
      refetch();
    } catch (err) {
      console.error("Failed to update service:", err);
      toast.error("Failed to update service. Please try again.");
    }
  };

  const handleDeleteSuccess = () => {
    invalidateAll();
    router.push("/services");
  };

  if (isLoading) {
    return <PageLoader />;
  }

  if (error || !service) {
    return (
      <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
        <ErrorDisplay
          error={error || new Error("Service not found")}
          retry={refetch}
        />
      </div>
    );
  }

  const typeName = getServiceTypeDisplayName(service.attributes.type);

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-auto">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Link href="/services">
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <ArrowLeft className="h-4 w-4" />
              </Button>
            </Link>
            <h2 className="text-2xl font-bold tracking-tight">{typeName}</h2>
            <ServiceTypeBadge type={service.attributes.type} />
          </div>
          <p className="text-sm text-muted-foreground font-mono pl-10">
            {service.id}
          </p>
        </div>
        <div className="flex gap-2">
          {isEditing ? (
            <>
              <Button
                variant="outline"
                onClick={cancelEditing}
                disabled={updateService.isPending}
              >
                <X className="mr-2 h-4 w-4" />
                Cancel
              </Button>
              <Button
                onClick={handleSave}
                disabled={updateService.isPending}
              >
                <Save className="mr-2 h-4 w-4" />
                {updateService.isPending ? "Saving..." : "Save Changes"}
              </Button>
            </>
          ) : (
            <>
              <Button variant="outline" onClick={startEditing}>
                <Pencil className="mr-2 h-4 w-4" />
                Edit
              </Button>
              <Button
                variant="destructive"
                onClick={() => setDeleteDialogOpen(true)}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </Button>
            </>
          )}
        </div>
      </div>

      <Separator />

      {/* Tasks Section */}
      <Card>
        <CardHeader>
          <CardTitle>Tasks</CardTitle>
        </CardHeader>
        <CardContent>
          {isEditing ? (
            <TaskConfigForm
              tasks={editTasks}
              serviceType={service.attributes.type}
              onChange={setEditTasks}
            />
          ) : (
            <div className="space-y-3">
              {service.attributes.tasks.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No tasks configured.
                </p>
              ) : (
                service.attributes.tasks.map((task, index) => (
                  <div
                    key={index}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div>
                      <p className="font-medium">{task.type}</p>
                      <p className="text-sm text-muted-foreground">
                        Interval: {task.interval}ms | Duration: {task.duration}ms
                      </p>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Tenants Section (Login/Channel only) */}
      {!isDropsService(service) && (
        <Card>
          <CardHeader>
            <CardTitle>Tenant Associations</CardTitle>
          </CardHeader>
          <CardContent>
            {isEditing ? (
              isLoginService(service) ? (
                <LoginTenantConfig
                  tenants={editLoginTenants}
                  onChange={setEditLoginTenants}
                />
              ) : isChannelService(service) ? (
                <ChannelTenantConfig
                  tenants={editChannelTenants}
                  onChange={setEditChannelTenants}
                />
              ) : null
            ) : (
              <div className="space-y-3">
                {isLoginService(service) && (
                  <>
                    {service.attributes.tenants.length === 0 ? (
                      <p className="text-sm text-muted-foreground">
                        No tenants associated.
                      </p>
                    ) : (
                      service.attributes.tenants.map((tenant, index) => (
                        <div
                          key={index}
                          className="flex items-center justify-between rounded-lg border p-3"
                        >
                          <div>
                            <p className="font-mono text-sm">{tenant.id}</p>
                            <p className="text-sm text-muted-foreground">
                              Port: {tenant.port}
                            </p>
                          </div>
                        </div>
                      ))
                    )}
                  </>
                )}

                {isChannelService(service) && (
                  <>
                    {service.attributes.tenants.length === 0 ? (
                      <p className="text-sm text-muted-foreground">
                        No tenants associated.
                      </p>
                    ) : (
                      service.attributes.tenants.map((tenant, tIndex) => (
                        <div key={tIndex} className="rounded-lg border p-4">
                          <div className="mb-3">
                            <p className="font-mono text-sm">{tenant.id}</p>
                            <p className="text-sm text-muted-foreground">
                              IP: {tenant.ipAddress}
                            </p>
                          </div>
                          {tenant.worlds.length === 0 ? (
                            <p className="text-sm text-muted-foreground pl-4">
                              No worlds configured.
                            </p>
                          ) : (
                            <div className="space-y-2 pl-4">
                              {tenant.worlds.map((world, wIndex) => (
                                <div key={wIndex} className="rounded border p-3 bg-muted/50">
                                  <p className="text-sm font-medium">
                                    World {world.id}
                                  </p>
                                  <div className="mt-2 flex flex-wrap gap-2">
                                    {world.channels.map((channel, cIndex) => (
                                      <span
                                        key={cIndex}
                                        className="text-xs bg-background rounded px-2 py-1"
                                      >
                                        Ch{channel.id}: {channel.port}
                                      </span>
                                    ))}
                                    {world.channels.length === 0 && (
                                      <span className="text-xs text-muted-foreground">
                                        No channels
                                      </span>
                                    )}
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      ))
                    )}
                  </>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      <DeleteServiceDialog
        service={service}
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        onSuccess={handleDeleteSuccess}
      />
    </div>
  );
}
