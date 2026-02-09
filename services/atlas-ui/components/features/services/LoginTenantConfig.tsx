"use client";

import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

import { useTenantConfigurations } from "@/lib/hooks/api/useTenants";
import type { LoginTenant } from "@/types/models/service";

interface LoginTenantConfigProps {
  tenants: LoginTenant[];
  onChange: (tenants: LoginTenant[]) => void;
  disabled?: boolean;
}

/**
 * Form component for configuring login service tenant associations.
 *
 * Allows selecting from existing tenants and configuring the port for each.
 */
export function LoginTenantConfig({
  tenants,
  onChange,
  disabled = false,
}: LoginTenantConfigProps) {
  const { data: availableTenants, isLoading, error } = useTenantConfigurations();

  // Get tenant info by ID
  const getTenantInfo = (tenantId: string) => {
    const tenant = availableTenants?.find((t) => t.id === tenantId);
    if (!tenant) return null;
    return {
      region: tenant.attributes.region,
      version: `${tenant.attributes.majorVersion}.${tenant.attributes.minorVersion}`,
    };
  };

  // Get tenants not yet selected
  const getAvailableTenantOptions = (excludeId?: string) => {
    if (!availableTenants) return [];
    const selectedIds = new Set(tenants.map((t) => t.id).filter((id) => id !== excludeId));
    return availableTenants.filter((t) => !selectedIds.has(t.id));
  };

  const handleAddTenant = () => {
    const available = getAvailableTenantOptions();
    if (available.length === 0) return;

    onChange([
      ...tenants,
      {
        id: "", // Will be selected by user
        port: 8484, // Default login port
      },
    ]);
  };

  const handleRemoveTenant = (index: number) => {
    onChange(tenants.filter((_, i) => i !== index));
  };

  const handleTenantChange = (
    index: number,
    field: keyof LoginTenant,
    value: string | number
  ) => {
    const updated = tenants.map((tenant, i) => {
      if (i !== index) return tenant;
      if (field === "id") {
        return { ...tenant, id: value as string };
      } else {
        return {
          ...tenant,
          port: typeof value === "string" ? parseInt(value, 10) || 0 : value,
        };
      }
    });
    onChange(updated);
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <Label className="text-base font-medium">Tenant Associations</Label>
          <Skeleton className="h-9 w-24" />
        </div>
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-md bg-destructive/10 p-4 text-sm text-destructive">
        Failed to load available tenants. Please try again.
      </div>
    );
  }

  const availableOptions = getAvailableTenantOptions();

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Label className="text-base font-medium">Tenant Associations</Label>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleAddTenant}
          disabled={disabled || availableOptions.length === 0}
        >
          <Plus className="mr-2 h-4 w-4" />
          Add Tenant
        </Button>
      </div>

      {!availableTenants || availableTenants.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No tenants available. Please create a tenant first.
        </p>
      ) : tenants.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No tenants associated. Click &quot;Add Tenant&quot; to associate one.
        </p>
      ) : null}

      <div className="space-y-3">
        {tenants.map((tenant, index) => {
          const tenantInfo = tenant.id ? getTenantInfo(tenant.id) : null;

          return (
            <Card key={index}>
              <CardContent className="pt-4">
                <div className="grid gap-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 grid gap-2">
                      <Label htmlFor={`tenant-id-${index}`}>Tenant</Label>
                      <Select
                        value={tenant.id}
                        onValueChange={(value) =>
                          handleTenantChange(index, "id", value)
                        }
                        disabled={disabled}
                      >
                        <SelectTrigger id={`tenant-id-${index}`}>
                          <SelectValue placeholder="Select tenant" />
                        </SelectTrigger>
                        <SelectContent>
                          {/* Show all available options including currently selected */}
                          {getAvailableTenantOptions(tenant.id).map((t) => (
                            <SelectItem key={t.id} value={t.id}>
                              {t.attributes.region} v{t.attributes.majorVersion}.
                              {t.attributes.minorVersion}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      {tenantInfo && (
                        <p className="text-xs text-muted-foreground">
                          ID: {tenant.id}
                        </p>
                      )}
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => handleRemoveTenant(index)}
                      disabled={disabled}
                      className="mt-6"
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>

                  <div className="grid gap-2">
                    <Label htmlFor={`tenant-port-${index}`}>Port</Label>
                    <Input
                      id={`tenant-port-${index}`}
                      type="number"
                      min={1}
                      max={65535}
                      value={tenant.port}
                      onChange={(e) =>
                        handleTenantChange(index, "port", e.target.value)
                      }
                      disabled={disabled}
                      placeholder="8484"
                    />
                    {(tenant.port < 1 || tenant.port > 65535) && (
                      <p className="text-xs text-destructive">
                        Port must be between 1 and 65535
                      </p>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
