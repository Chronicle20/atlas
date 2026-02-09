"use client";

import { useState } from "react";
import { Plus, Trash2, ChevronDown, ChevronRight } from "lucide-react";

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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

import { useTenantConfigurations } from "@/lib/hooks/api/useTenants";
import type { ChannelTenant, ChannelWorld, ChannelChannel } from "@/types/models/service";

interface ChannelTenantConfigProps {
  tenants: ChannelTenant[];
  onChange: (tenants: ChannelTenant[]) => void;
  disabled?: boolean;
}

/**
 * Form component for configuring channel service tenant associations.
 *
 * Provides hierarchical configuration:
 * - Tenant selection with IP address
 * - World configuration (ID 0-255)
 * - Channel configuration per world (ID 0-255, port)
 */
export function ChannelTenantConfig({
  tenants,
  onChange,
  disabled = false,
}: ChannelTenantConfigProps) {
  const { data: availableTenants, isLoading, error } = useTenantConfigurations();
  const [expandedTenants, setExpandedTenants] = useState<Set<number>>(new Set([0]));
  const [expandedWorlds, setExpandedWorlds] = useState<Set<string>>(new Set());

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

  const toggleTenantExpanded = (index: number) => {
    const newExpanded = new Set(expandedTenants);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedTenants(newExpanded);
  };

  const toggleWorldExpanded = (tenantIndex: number, worldIndex: number) => {
    const key = `${tenantIndex}-${worldIndex}`;
    const newExpanded = new Set(expandedWorlds);
    if (newExpanded.has(key)) {
      newExpanded.delete(key);
    } else {
      newExpanded.add(key);
    }
    setExpandedWorlds(newExpanded);
  };

  // Tenant operations
  const handleAddTenant = () => {
    const available = getAvailableTenantOptions();
    if (available.length === 0) return;

    onChange([
      ...tenants,
      {
        id: "",
        ipAddress: "127.0.0.1",
        worlds: [],
      },
    ]);
    setExpandedTenants(new Set([...expandedTenants, tenants.length]));
  };

  const handleRemoveTenant = (index: number) => {
    onChange(tenants.filter((_, i) => i !== index));
  };

  const handleTenantFieldChange = (
    index: number,
    field: "id" | "ipAddress",
    value: string
  ) => {
    const tenant = tenants[index];
    if (!tenant) return;

    const updated = tenants.map((t, i) =>
      i === index ? { ...t, [field]: value } : t
    );
    onChange(updated);
  };

  // World operations
  const handleAddWorld = (tenantIndex: number) => {
    const tenant = tenants[tenantIndex];
    if (!tenant) return;

    const existingWorldIds = new Set(tenant.worlds.map((w) => w.id));

    // Find next available world ID
    let nextId = 0;
    while (existingWorldIds.has(nextId) && nextId < 256) {
      nextId++;
    }
    if (nextId >= 256) return; // No more world IDs available

    // Calculate default port for the first channel (base port + world * 100 + channel)
    const basePort = 8585;
    const defaultChannelPort = basePort + nextId * 100;

    // Create world with one default channel (a world without channels is not valid)
    const newWorld: ChannelWorld = {
      id: nextId,
      channels: [{ id: 0, port: defaultChannelPort }]
    };
    const updated = tenants.map((t, i) =>
      i === tenantIndex
        ? { ...t, worlds: [...t.worlds, newWorld] }
        : t
    );
    onChange(updated);

    // Expand the new world
    const worldKey = `${tenantIndex}-${tenant.worlds.length}`;
    setExpandedWorlds(new Set([...expandedWorlds, worldKey]));
  };

  const handleRemoveWorld = (tenantIndex: number, worldIndex: number) => {
    const updated = tenants.map((t, i) =>
      i === tenantIndex
        ? { ...t, worlds: t.worlds.filter((_, wi) => wi !== worldIndex) }
        : t
    );
    onChange(updated);
  };

  const handleWorldIdChange = (
    tenantIndex: number,
    worldIndex: number,
    value: string
  ) => {
    const newId = Math.min(255, Math.max(0, parseInt(value, 10) || 0));
    const updated = tenants.map((t, ti) =>
      ti === tenantIndex
        ? {
            ...t,
            worlds: t.worlds.map((w, wi) =>
              wi === worldIndex ? { ...w, id: newId } : w
            ),
          }
        : t
    );
    onChange(updated);
  };

  // Channel operations
  const handleAddChannel = (tenantIndex: number, worldIndex: number) => {
    const tenant = tenants[tenantIndex];
    if (!tenant) return;

    const world = tenant.worlds[worldIndex];
    if (!world) return;

    const existingChannelIds = new Set(world.channels.map((c) => c.id));

    // Find next available channel ID
    let nextId = 0;
    while (existingChannelIds.has(nextId) && nextId < 256) {
      nextId++;
    }
    if (nextId >= 256) return;

    // Calculate next port (base port + world * 100 + channel)
    const basePort = 8585;
    const suggestedPort = basePort + world.id * 100 + nextId;
    const newChannel: ChannelChannel = { id: nextId, port: suggestedPort };

    const updated = tenants.map((t, ti) =>
      ti === tenantIndex
        ? {
            ...t,
            worlds: t.worlds.map((w, wi) =>
              wi === worldIndex
                ? { ...w, channels: [...w.channels, newChannel] }
                : w
            ),
          }
        : t
    );
    onChange(updated);
  };

  const handleRemoveChannel = (
    tenantIndex: number,
    worldIndex: number,
    channelIndex: number
  ) => {
    const updated = tenants.map((t, ti) =>
      ti === tenantIndex
        ? {
            ...t,
            worlds: t.worlds.map((w, wi) =>
              wi === worldIndex
                ? { ...w, channels: w.channels.filter((_, ci) => ci !== channelIndex) }
                : w
            ),
          }
        : t
    );
    onChange(updated);
  };

  const handleChannelChange = (
    tenantIndex: number,
    worldIndex: number,
    channelIndex: number,
    field: keyof ChannelChannel,
    value: string
  ) => {
    const numValue = parseInt(value, 10) || 0;
    const safeValue = field === "id" ? Math.min(255, Math.max(0, numValue)) : numValue;

    const updated = tenants.map((t, ti) =>
      ti === tenantIndex
        ? {
            ...t,
            worlds: t.worlds.map((w, wi) =>
              wi === worldIndex
                ? {
                    ...w,
                    channels: w.channels.map((c, ci) =>
                      ci === channelIndex ? { ...c, [field]: safeValue } : c
                    ),
                  }
                : w
            ),
          }
        : t
    );
    onChange(updated);
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <Label className="text-base font-medium">Tenant Associations</Label>
          <Skeleton className="h-9 w-24" />
        </div>
        <Skeleton className="h-32 w-full" />
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

      <div className="space-y-4">
        {tenants.map((tenant, tIndex) => {
          const tenantInfo = tenant.id ? getTenantInfo(tenant.id) : null;
          const isExpanded = expandedTenants.has(tIndex);

          return (
            <Card key={tIndex}>
              <Collapsible open={isExpanded} onOpenChange={() => toggleTenantExpanded(tIndex)}>
                <CardHeader className="py-3">
                  <div className="flex items-center justify-between">
                    <CollapsibleTrigger asChild>
                      <Button variant="ghost" size="sm" className="p-0 h-auto">
                        {isExpanded ? (
                          <ChevronDown className="h-4 w-4 mr-2" />
                        ) : (
                          <ChevronRight className="h-4 w-4 mr-2" />
                        )}
                        <CardTitle className="text-sm font-medium">
                          {tenantInfo
                            ? `${tenantInfo.region} v${tenantInfo.version}`
                            : "Select Tenant"}
                        </CardTitle>
                      </Button>
                    </CollapsibleTrigger>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-muted-foreground">
                        {tenant.worlds.length} world(s)
                      </span>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        onClick={() => handleRemoveTenant(tIndex)}
                        disabled={disabled}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </div>
                </CardHeader>

                <CollapsibleContent>
                  <CardContent className="pt-0 space-y-4">
                    {/* Tenant Selection */}
                    <div className="grid grid-cols-2 gap-4">
                      <div className="grid gap-2">
                        <Label>Tenant</Label>
                        <Select
                          value={tenant.id}
                          onValueChange={(value) =>
                            handleTenantFieldChange(tIndex, "id", value)
                          }
                          disabled={disabled}
                        >
                          <SelectTrigger>
                            <SelectValue placeholder="Select tenant" />
                          </SelectTrigger>
                          <SelectContent>
                            {getAvailableTenantOptions(tenant.id).map((t) => (
                              <SelectItem key={t.id} value={t.id}>
                                {t.attributes.region} v{t.attributes.majorVersion}.
                                {t.attributes.minorVersion}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="grid gap-2">
                        <Label>IP Address</Label>
                        <Input
                          value={tenant.ipAddress}
                          onChange={(e) =>
                            handleTenantFieldChange(tIndex, "ipAddress", e.target.value)
                          }
                          disabled={disabled}
                          placeholder="127.0.0.1"
                        />
                      </div>
                    </div>

                    {/* Worlds */}
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <Label className="text-sm">Worlds</Label>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => handleAddWorld(tIndex)}
                          disabled={disabled || tenant.worlds.length >= 256}
                        >
                          <Plus className="mr-1 h-3 w-3" />
                          Add World
                        </Button>
                      </div>

                      {tenant.worlds.length === 0 && (
                        <p className="text-xs text-muted-foreground pl-2">
                          No worlds configured.
                        </p>
                      )}

                      <div className="space-y-2 pl-2">
                        {tenant.worlds.map((world, wIndex) => {
                          const worldKey = `${tIndex}-${wIndex}`;
                          const isWorldExpanded = expandedWorlds.has(worldKey);

                          return (
                            <Card key={wIndex} className="bg-muted/50">
                              <Collapsible
                                open={isWorldExpanded}
                                onOpenChange={() => toggleWorldExpanded(tIndex, wIndex)}
                              >
                                <CardHeader className="py-2 px-3">
                                  <div className="flex items-center justify-between">
                                    <CollapsibleTrigger asChild>
                                      <Button variant="ghost" size="sm" className="p-0 h-auto">
                                        {isWorldExpanded ? (
                                          <ChevronDown className="h-3 w-3 mr-1" />
                                        ) : (
                                          <ChevronRight className="h-3 w-3 mr-1" />
                                        )}
                                        <span className="text-xs font-medium">
                                          World {world.id}
                                        </span>
                                      </Button>
                                    </CollapsibleTrigger>
                                    <div className="flex items-center gap-2">
                                      <span className="text-xs text-muted-foreground">
                                        {world.channels.length} ch
                                      </span>
                                      <Button
                                        type="button"
                                        variant="ghost"
                                        size="icon"
                                        className="h-6 w-6"
                                        onClick={() => handleRemoveWorld(tIndex, wIndex)}
                                        disabled={disabled}
                                      >
                                        <Trash2 className="h-3 w-3 text-destructive" />
                                      </Button>
                                    </div>
                                  </div>
                                </CardHeader>

                                <CollapsibleContent>
                                  <CardContent className="py-2 px-3 space-y-3">
                                    <div className="grid gap-2">
                                      <Label className="text-xs">World ID (0-255)</Label>
                                      <Input
                                        type="number"
                                        min={0}
                                        max={255}
                                        value={world.id}
                                        onChange={(e) =>
                                          handleWorldIdChange(tIndex, wIndex, e.target.value)
                                        }
                                        disabled={disabled}
                                        className="h-8 text-sm"
                                      />
                                    </div>

                                    {/* Channels */}
                                    <div className="space-y-2">
                                      <div className="flex items-center justify-between">
                                        <Label className="text-xs">Channels</Label>
                                        <Button
                                          type="button"
                                          variant="outline"
                                          size="sm"
                                          className="h-6 text-xs"
                                          onClick={() => handleAddChannel(tIndex, wIndex)}
                                          disabled={disabled || world.channels.length >= 256}
                                        >
                                          <Plus className="mr-1 h-2 w-2" />
                                          Add
                                        </Button>
                                      </div>

                                      {world.channels.length === 0 && (
                                        <p className="text-xs text-muted-foreground">
                                          No channels configured.
                                        </p>
                                      )}

                                      <div className="space-y-1">
                                        {world.channels.map((channel, cIndex) => (
                                          <div
                                            key={cIndex}
                                            className="flex items-center gap-2 bg-background rounded p-2"
                                          >
                                            <div className="grid gap-1 flex-1">
                                              <Label className="text-xs">Ch ID</Label>
                                              <Input
                                                type="number"
                                                min={0}
                                                max={255}
                                                value={channel.id}
                                                onChange={(e) =>
                                                  handleChannelChange(
                                                    tIndex,
                                                    wIndex,
                                                    cIndex,
                                                    "id",
                                                    e.target.value
                                                  )
                                                }
                                                disabled={disabled}
                                                className="h-7 text-xs"
                                              />
                                            </div>
                                            <div className="grid gap-1 flex-1">
                                              <Label className="text-xs">Port</Label>
                                              <Input
                                                type="number"
                                                min={1}
                                                max={65535}
                                                value={channel.port}
                                                onChange={(e) =>
                                                  handleChannelChange(
                                                    tIndex,
                                                    wIndex,
                                                    cIndex,
                                                    "port",
                                                    e.target.value
                                                  )
                                                }
                                                disabled={disabled}
                                                className="h-7 text-xs"
                                              />
                                            </div>
                                            <Button
                                              type="button"
                                              variant="ghost"
                                              size="icon"
                                              className="h-6 w-6 mt-5"
                                              onClick={() =>
                                                handleRemoveChannel(tIndex, wIndex, cIndex)
                                              }
                                              disabled={disabled}
                                            >
                                              <Trash2 className="h-3 w-3 text-destructive" />
                                            </Button>
                                          </div>
                                        ))}
                                      </div>
                                    </div>
                                  </CardContent>
                                </CollapsibleContent>
                              </Collapsible>
                            </Card>
                          );
                        })}
                      </div>
                    </div>
                  </CardContent>
                </CollapsibleContent>
              </Collapsible>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
